package article

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net"
	"net/textproto"
	"reflect"
	"strconv"
	"sync"
	"time"

	"han/internal/audit"
	"han/internal/mailer"
	"han/internal/platform"

	"github.com/jackc/pgx/v5"
)

// NoticeDispatcherConfig contains explicit notification operating budgets.
type NoticeDispatcherConfig struct {
	PollInterval        time.Duration
	BatchLimit          int
	LeaseDuration       time.Duration
	DBOperationBudget   time.Duration
	BackoffInitial      time.Duration
	BackoffMax          time.Duration
	SMTPOperationBudget time.Duration
}

// Validate checks notification configuration without opening resources.
func (c NoticeDispatcherConfig) Validate() error {
	for _, duration := range []time.Duration{c.PollInterval, c.LeaseDuration,
		c.DBOperationBudget, c.BackoffInitial, c.BackoffMax, c.SMTPOperationBudget} {
		if duration <= 0 || duration%time.Microsecond != 0 {
			return errors.New("notice configuration: duration")
		}
	}
	if c.BatchLimit <= 0 || int64(c.BatchLimit) > math.MaxInt32 || c.BackoffInitial > c.BackoffMax {
		return errors.New("notice configuration: range")
	}
	if c.DBOperationBudget > (time.Duration(math.MaxInt64)-c.SMTPOperationBudget)/4 ||
		c.LeaseDuration <= c.SMTPOperationBudget+4*c.DBOperationBudget {
		return errors.New("notice configuration: cross_constraint")
	}
	return nil
}

// NoticeTransactions begins explicitly isolated caller-owned transactions.
type NoticeTransactions interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// NoticeProcessingGate evaluates deployment evidence in the caller's transaction.
type NoticeProcessingGate interface {
	RequireOpen(context.Context) error
	RequireOpenInTx(context.Context, pgx.Tx) error
}

// NoticeAuditStore is the shared append/read audit authority.
type NoticeAuditStore interface {
	Append(context.Context, pgx.Tx, audit.Event) error
	Get(context.Context, audit.QueryRower, string) (audit.Event, error)
}

// NoticeDispatcherLogger accepts only bounded, non-sensitive observations.
type NoticeDispatcherLogger interface {
	Log(context.Context, slog.Level, string, ...any)
}

// NoticeDispatcherDependencies are explicit production collaborators.
type NoticeDispatcherDependencies struct {
	// AcquireWork retains admission through fenced result persistence and readback.
	AcquireWork  func(context.Context) (context.Context, func(), error)
	Transactions NoticeTransactions
	Repository   Repository
	Gate         NoticeProcessingGate
	Mailer       mailer.Mailer
	Audit        NoticeAuditStore
	IDs          ArticleIDGenerator
	NewTraceID   func() (string, error)
	Logger       NoticeDispatcherLogger
	WorkerID     string
}

// TransferNoticeDispatcher consumes committed recipient snapshots.
type TransferNoticeDispatcher struct {
	deps             NoticeDispatcherDependencies
	config           NoticeDispatcherConfig
	mu               sync.Mutex
	done             chan struct{}
	loopStarted      chan struct{}
	closed           bool
	stopping         bool
	started          bool
	running          bool
	active           bool
	cancel           context.CancelFunc
	loopCancel       context.CancelFunc
	shutdown         context.Context
	shutdownBound    bool
	settlementCancel context.CancelFunc
	waitExpired      bool
	err              error
}

// NewTransferNoticeDispatcher validates dependencies without starting work.
func NewTransferNoticeDispatcher(deps NoticeDispatcherDependencies, config NoticeDispatcherConfig) (*TransferNoticeDispatcher, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	for _, dependency := range []any{deps.Transactions, deps.Gate, deps.Mailer, deps.Audit, deps.IDs, deps.Logger} {
		if noticeNil(dependency) {
			return nil, errors.New("notice dependencies: missing")
		}
	}
	if deps.NewTraceID == nil || !validUUID(deps.WorkerID) {
		return nil, errors.New("notice dependencies: identity")
	}
	return &TransferNoticeDispatcher{deps: deps, config: config, done: make(chan struct{}), loopStarted: make(chan struct{})}, nil
}

func noticeNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	}
	return false
}

// Reconcile processes one bounded notification round.
func (d *TransferNoticeDispatcher) Reconcile(ctx context.Context) error {
	if d == nil || ctx == nil {
		return noticeFailure("context_required", nil, true)
	}
	work, cancel, ok := d.beginRound(ctx)
	if !ok {
		return nil
	}
	defer d.endRound(cancel)
	traceID, err := d.deps.NewTraceID()
	if err != nil {
		return d.roundFailure(ctx, noticeFailure("trace_unavailable", err, false))
	}
	work, err = platform.ContextWithTraceID(work, traceID)
	if err != nil {
		return d.roundFailure(ctx, noticeFailure("trace_invalid", err, true))
	}
	err = d.reconcile(work)
	return d.roundFailure(work, err)
}

func (d *TransferNoticeDispatcher) beginRound(ctx context.Context) (context.Context, context.CancelFunc, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.stopping || d.active {
		return nil, nil, false
	}
	work, cancel := context.WithCancel(ctx)
	d.active = true
	d.cancel = cancel
	return work, cancel, true
}

func (d *TransferNoticeDispatcher) endRound(cancel context.CancelFunc) {
	cancel()
	d.mu.Lock()
	defer d.mu.Unlock()
	d.active = false
	d.cancel = nil
	d.closeDoneLocked()
}

func (d *TransferNoticeDispatcher) roundFailure(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	var failure *noticeError
	if errors.As(err, &failure) {
		d.observe(ctx, failure.bucket)
		if !failure.terminal {
			d.mu.Lock()
			stopping := d.stopping
			d.mu.Unlock()
			if !stopping {
				return nil
			}
		} else {
			d.Stop()
		}
		d.mu.Lock()
		d.err = errors.Join(d.err, err)
		d.mu.Unlock()
		return err
	}
	return err
}

// Start starts at most one serial polling loop.
func (d *TransferNoticeDispatcher) Start(ctx context.Context) {
	if d == nil {
		return
	}
	d.mu.Lock()
	if d.started || d.stopping {
		d.mu.Unlock()
		return
	}
	if ctx == nil {
		d.err = noticeFailure("context_required", nil, true)
		d.stopping = true
		d.closeDoneLocked()
		d.mu.Unlock()
		return
	}
	work, cancel := context.WithCancel(ctx)
	d.started = true
	d.running = true
	d.loopCancel = cancel
	d.mu.Unlock()
	go d.run(work)
}

func (d *TransferNoticeDispatcher) run(ctx context.Context) {
	defer func() { d.mu.Lock(); d.running = false; d.stopping = true; d.closeDoneLocked(); d.mu.Unlock() }()
	ticker := time.NewTicker(d.config.PollInterval)
	defer ticker.Stop()
	close(d.loopStarted)
	for {
		select {
		case <-ctx.Done():
			d.Stop()
			return
		case <-ticker.C:
			if ctx.Err() != nil {
				d.Stop()
				return
			}
			if err := d.Reconcile(ctx); err != nil {
				d.Stop()
				return
			}
		}
	}
}

// BeginShutdown binds the coordinator's existing shared deadline before canceling work.
func (d *TransferNoticeDispatcher) BeginShutdown(shared context.Context) {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.shutdownBound {
		d.shutdownBound = true
		if shared != nil {
			if _, ok := shared.Deadline(); ok {
				d.shutdown = shared
			}
		}
	}
	d.stopLocked()
}

// Stop prevents new work and cancels current operations.
func (d *TransferNoticeDispatcher) Stop() {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.stopLocked()
}

func (d *TransferNoticeDispatcher) stopLocked() {
	d.stopping = true
	if d.cancel != nil {
		d.cancel()
	}
	if d.loopCancel != nil {
		d.loopCancel()
	}
	d.closeDoneLocked()
}

func (d *TransferNoticeDispatcher) closeDoneLocked() {
	if d.stopping && !d.active && !d.running && !d.closed {
		close(d.done)
		d.closed = true
	}
}

// Wait joins actual operations, including initial reconciliation.
func (d *TransferNoticeDispatcher) Wait(ctx context.Context) error {
	if d == nil || ctx == nil {
		return noticeFailure("context_required", nil, true)
	}
	select {
	case <-d.done:
		return d.Err()
	default:
	}
	select {
	case <-d.done:
		return d.Err()
	case <-ctx.Done():
		d.Stop()
		d.mu.Lock()
		d.waitExpired = true
		if d.settlementCancel != nil {
			d.settlementCancel()
		}
		d.err = errors.Join(d.err, noticeFailure("shutdown_timeout", ctx.Err(), false))
		d.mu.Unlock()
		<-d.done
		return d.Err()
	}
}

// Done closes only after stopped operations have actually exited.
func (d *TransferNoticeDispatcher) Done() <-chan struct{} { return d.done }

// Err returns the terminal or shutdown failure.
func (d *TransferNoticeDispatcher) Err() error {
	if d == nil {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.err
}

type noticeError struct {
	bucket   string
	cause    error
	terminal bool
}

func (e *noticeError) Error() string { return "article notice: " + e.bucket }
func (e *noticeError) Unwrap() error { return e.cause }
func noticeFailure(bucket string, cause error, terminal bool) error {
	return &noticeError{bucket: bucket, cause: cause, terminal: terminal}
}

func (d *TransferNoticeDispatcher) observe(ctx context.Context, bucket string) {
	traceID, _ := platform.TraceIDFromContext(ctx)
	d.deps.Logger.Log(ctx, slog.LevelWarn, "article transfer notice", "reason_bucket", bucket, "trace_id", traceID)
}

type noticeClaim struct {
	NoticeID      string
	Kind          RecipientKind
	ClaimedBy     string
	AttemptCount  int64
	LockedUntil   time.Time
	ArticleRef    string
	EmailSnapshot string
	MaxAttempts   int64
	LastAttemptAt time.Time
	// issued preserves this exact internally committed handle, not a send right.
	issued *noticeClaim
}

type noticeDisposition string

const (
	noticeAccepted noticeDisposition = "accepted"
	noticeRetry    noticeDisposition = "retry"
	noticeTerminal noticeDisposition = "terminal"
	noticeDeferred noticeDisposition = "deferred"
)

type noticeResult struct {
	disposition noticeDisposition
	bucket      string
	risk        bool
	stopRound   bool
	eventID     string
	traceID     string
	retryDelay  time.Duration
	attempted   bool
}

func (d *TransferNoticeDispatcher) reconcile(ctx context.Context) error {
	for i := 0; i < d.config.BatchLimit; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		found, err := d.maintainWork(ctx)
		if err != nil {
			return err
		}
		if !found {
			break
		}
	}
	for i := 0; i < d.config.BatchLimit; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		stop, err := d.dispatchWork(ctx)
		if err != nil {
			return err
		}
		if stop {
			return nil
		}
	}
	return nil
}

func (d *TransferNoticeDispatcher) transaction(ctx context.Context) (context.Context, context.CancelFunc, pgx.Tx, error) {
	work, cancel := context.WithTimeout(ctx, d.config.DBOperationBudget)
	tx, err := d.deps.Transactions.BeginTx(work, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		cancel()
		return nil, nil, nil, noticeDBFailure("database_unavailable", err)
	}
	if noticeNil(tx) {
		cancel()
		return nil, nil, nil, noticeFailure("transaction_missing", nil, true)
	}
	return work, cancel, tx, nil
}

// finishNoticeTx releases rollback resources before canceling the operation budget.
func finishNoticeTx(ctx context.Context, cancel context.CancelFunc, tx pgx.Tx) {
	_ = tx.Rollback(ctx) // Commit/rollback may already have closed the transaction.
	cancel()
}

func noticeDBFailure(bucket string, err error) error {
	var pgerr interface{ SQLState() string }
	terminal := errors.As(err, &pgerr) && (pgerr.SQLState() == "42P01" || pgerr.SQLState() == "42703" || pgerr.SQLState() == "23514")
	return noticeFailure(bucket, err, terminal)
}

func noticeDBTime(ctx context.Context, tx pgx.Tx) (time.Time, time.Time, error) {
	// Sample before the query so its round trip is conservatively charged to lease time.
	sampled := time.Now()
	var now time.Time
	if err := tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); err != nil {
		return time.Time{}, sampled, noticeDBFailure("database_clock", err)
	}
	now = now.UTC().Truncate(time.Microsecond)
	if now.IsZero() || now.Year() < 1 || now.Year() > 9999 {
		return time.Time{}, sampled, noticeFailure("database_clock_invalid", nil, true)
	}
	return now, sampled, nil
}

func (d *TransferNoticeDispatcher) claimDue(ctx context.Context) (noticeClaim, bool, error) {
	work, cancel, tx, err := d.transaction(ctx)
	if err != nil {
		return noticeClaim{}, false, err
	}
	defer finishNoticeTx(work, cancel, tx)
	if err := d.deps.Gate.RequireOpenInTx(work, tx); err != nil {
		d.observe(ctx, "processing_gate_closed")
		return noticeClaim{}, false, nil
	}
	notice, before, found, err := d.deps.Repository.LockDueNoticeRecipient(work, tx)
	if err != nil || !found {
		return noticeClaim{}, false, err
	}
	now, _, err := noticeDBTime(work, tx)
	if err != nil {
		return noticeClaim{}, false, err
	}
	if err := validNoticeRecipient(notice, before); err != nil {
		return noticeClaim{}, false, err
	}
	if !noticeDue(before, now) || before.AttemptCount >= before.MaxAttempts {
		return noticeClaim{}, false, nil
	}
	after, changed, err := d.deps.Repository.ClaimNoticeRecipient(work, tx, before, d.deps.WorkerID, d.config.LeaseDuration, now)
	if err != nil || !changed {
		return noticeClaim{}, false, err
	}
	claim := newNoticeClaim(notice, after)
	if before.Status == RecipientProcessing {
		result := noticeResult{bucket: "smtp_outcome_unknown", risk: true}
		if _, err = d.appendNoticeAudit(work, tx, claim, result, now, &before, true); err != nil {
			return noticeClaim{}, false, err
		}
	}
	if err := work.Err(); err != nil {
		return noticeClaim{}, false, err
	}
	if err := tx.Commit(work); err != nil {
		return noticeClaim{}, false, noticeDBFailure("claim_commit_unknown", err)
	}
	issued := claim
	claim.issued = &issued
	return claim, true, nil
}

func validNoticeRecipient(n TransferNotice, r TransferNoticeRecipient) error {
	if !validUUID(n.ID) || !validUUID(n.ArticleRef) || n.ID != r.NoticeID ||
		(r.Kind != RecipientNewAuthor && r.Kind != RecipientFormerDeletedAuthor) ||
		r.AttemptCount < 0 || r.MaxAttempts < 1 || r.AttemptCount > r.MaxAttempts {
		return noticeFailure("recipient_invariant", nil, true)
	}
	switch r.Status {
	case RecipientPending, RecipientAccepted, RecipientFailedTerminal:
		return nil
	case RecipientProcessing:
		if r.ClaimedBy != nil && r.LockedUntil != nil && r.LastAttemptAt != nil && r.AttemptCount > 0 {
			return nil
		}
	case RecipientRetryWait:
		if r.NextAttemptAt != nil {
			return nil
		}
	}
	return noticeFailure("recipient_invariant", nil, true)
}

func noticeDue(r TransferNoticeRecipient, now time.Time) bool {
	switch r.Status {
	case RecipientPending:
		return true
	case RecipientRetryWait:
		return r.NextAttemptAt != nil && !r.NextAttemptAt.After(now)
	case RecipientProcessing:
		return r.LockedUntil != nil && !r.LockedUntil.After(now)
	}
	return false
}

func newNoticeClaim(n TransferNotice, r TransferNoticeRecipient) noticeClaim {
	c := noticeClaim{NoticeID: n.ID, ArticleRef: n.ArticleRef, Kind: r.Kind, EmailSnapshot: r.EmailSnapshot,
		AttemptCount: r.AttemptCount, MaxAttempts: r.MaxAttempts}
	if r.ClaimedBy != nil {
		c.ClaimedBy = *r.ClaimedBy
	}
	if r.LockedUntil != nil {
		c.LockedUntil = *r.LockedUntil
	}
	if r.LastAttemptAt != nil {
		c.LastAttemptAt = *r.LastAttemptAt
	}
	return c
}

func noticeOwns(c noticeClaim, r TransferNoticeRecipient, now time.Time) bool {
	return c.NoticeID == r.NoticeID && c.Kind == r.Kind && r.Status == RecipientProcessing &&
		r.ClaimedBy != nil && *r.ClaimedBy == c.ClaimedBy && r.AttemptCount == c.AttemptCount &&
		r.LockedUntil != nil && r.LockedUntil.Equal(c.LockedUntil) && r.LockedUntil.After(now)
}

func (d *TransferNoticeDispatcher) maintain(ctx context.Context) (bool, error) {
	work, cancel, tx, err := d.transaction(ctx)
	if err != nil {
		return false, err
	}
	defer finishNoticeTx(work, cancel, tx)
	notice, before, found, err := d.deps.Repository.LockExhaustedNoticeRecipient(work, tx)
	if err != nil || !found {
		return false, err
	}
	now, _, err := noticeDBTime(work, tx)
	if err != nil {
		return false, err
	}
	if err := validNoticeRecipient(notice, before); err != nil {
		return false, err
	}
	if !noticeDue(before, now) || before.AttemptCount != before.MaxAttempts {
		return false, nil
	}
	after, changed, err := d.deps.Repository.FinishExhaustedNoticeRecipient(work, tx, before, now)
	if err != nil || !changed {
		return false, err
	}
	r := noticeResult{disposition: noticeTerminal, bucket: "attempts_exhausted", risk: noticeExhaustedRisk(before)}
	if before.Status == RecipientProcessing {
		r.bucket = "attempts_exhausted_unknown"
	}
	claim := newNoticeClaim(notice, before)
	if _, err = d.appendNoticeAudit(work, tx, claim, r, now, &before, false); err != nil {
		return false, err
	}
	if err := tx.Commit(work); err != nil {
		return false, noticeDBFailure("maintenance_commit_unknown", err)
	}
	_ = after // RETURNING is consumed before commit; no result grants sending rights.
	return true, nil
}

func noticeExhaustedRisk(r TransferNoticeRecipient) bool {
	if r.Status == RecipientProcessing {
		return true
	}
	if r.Status != RecipientRetryWait {
		return false
	}
	if r.FailureBucket == nil {
		return true
	}
	switch *r.FailureBucket {
	case "processing_gate_closed", "lease_insufficient", "invalid_message", "unsupported_purpose",
		"smtp_sender_or_policy", "smtp_transient_rejection", "smtp_permanent_rejection", "smtp_security_or_config":
		return false
	}
	return true
}

func noticeBackoff(initial, maximum time.Duration, attempt int64) time.Duration {
	if attempt <= 1 {
		return initial
	}
	// Duration has at most 63 magnitude bits; giant bigint attempts never drive a loop.
	if attempt > 63 {
		return maximum
	}
	value := initial
	for shift := int64(1); shift < attempt; shift++ {
		if value >= maximum || value > maximum/2 {
			return maximum
		}
		value *= 2
	}
	return value
}

func noticeAdd(now time.Time, delay time.Duration) (time.Time, error) {
	if delay <= 0 || delay%time.Microsecond != 0 || now.IsZero() || now.Year() < 1 || now.Year() > 9999 {
		return time.Time{}, noticeFailure("time_range", nil, true)
	}
	after := now.Add(delay)
	if !after.After(now) || after.Year() > 9999 {
		return time.Time{}, noticeFailure("time_range", nil, true)
	}
	return after.UTC().Truncate(time.Microsecond), nil
}

func (d *TransferNoticeDispatcher) preflight(ctx context.Context, c noticeClaim) (time.Duration, time.Time, bool, error) {
	work, cancel, tx, err := d.transaction(ctx)
	if err != nil {
		return 0, time.Time{}, false, err
	}
	defer finishNoticeTx(work, cancel, tx)
	_, current, found, err := d.deps.Repository.LockNoticeRecipient(work, tx, c.NoticeID, c.Kind)
	if err != nil || !found {
		return 0, time.Time{}, false, err
	}
	now, sampled, err := noticeDBTime(work, tx)
	if err != nil {
		return 0, sampled, false, err
	}
	if !noticeOwns(c, current, now) {
		d.observe(ctx, "lease_competition")
		return 0, sampled, false, nil
	}
	gateErr := d.deps.Gate.RequireOpenInTx(work, tx)
	bucket := ""
	if gateErr != nil {
		bucket = "processing_gate_closed"
	} else if c.LockedUntil.Sub(now) <= d.config.SMTPOperationBudget+d.config.DBOperationBudget {
		bucket = "lease_insufficient"
	}
	if bucket != "" {
		err := d.deferNotice(ctx, work, tx, c, bucket, gateErr, now)
		return 0, sampled, false, err
	}
	if err := work.Err(); err != nil {
		return 0, sampled, false, err
	}
	if err := tx.Commit(work); err != nil {
		return 0, sampled, false, noticeDBFailure("preflight_commit_unknown", err)
	}
	return c.LockedUntil.Sub(now) - d.config.DBOperationBudget, sampled, true, nil
}

func (d *TransferNoticeDispatcher) deliver(ctx context.Context, c noticeClaim) (bool, error) {
	leaseBudget, sampled, send, err := d.preflight(ctx, c)
	if err != nil || !send {
		return true, err
	}
	if err := ctx.Err(); err != nil {
		return true, err
	}
	message, err := noticeMessage(c)
	attempted := false
	if err == nil {
		budget := noticeSendBudget(ctx, d.config.SMTPOperationBudget, leaseBudget, sampled)
		if budget <= 0 {
			_, err := d.saveResult(ctx, c, noticeResult{disposition: noticeDeferred, bucket: "lease_insufficient"})
			return true, err
		}
		mailCtx, cancel := context.WithTimeout(ctx, budget)
		if mailCtx.Err() != nil {
			cancel()
			return true, mailCtx.Err()
		}
		attempted = true
		err = d.deps.Mailer.Send(mailCtx, message)
		cancel()
	}
	result := classifyNoticeSMTP(err)
	result.attempted = attempted
	result.traceID, _ = platform.TraceIDFromContext(ctx)
	if result.stopRound {
		d.observe(ctx, result.bucket)
	}
	parent, cancelResult, available := d.resultContext(ctx)
	if !available {
		return true, d.shutdownResultError()
	}
	defer cancelResult()
	_, saveErr := d.saveResult(parent, c, result)
	if saveErr != nil {
		return true, saveErr
	}
	return result.stopRound || ctx.Err() != nil, nil
}

func noticeSendBudget(ctx context.Context, smtp, lease time.Duration, sampled time.Time) time.Duration {
	if ctx.Err() != nil {
		return 0
	}
	budget := min(smtp, lease-time.Since(sampled))
	if deadline, ok := ctx.Deadline(); ok {
		budget = min(budget, time.Until(deadline))
	}
	return budget
}

func (d *TransferNoticeDispatcher) resultContext(work context.Context) (context.Context, context.CancelFunc, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	parent := work
	if d.stopping || work.Err() != nil {
		if d.shutdown == nil || d.shutdown.Err() != nil {
			return nil, nil, false
		}
		parent = d.shutdown
	}
	ctx, cancel := context.WithCancel(parent)
	d.settlementCancel = cancel
	if d.waitExpired {
		cancel()
	}
	return ctx, func() { cancel(); d.mu.Lock(); d.settlementCancel = nil; d.mu.Unlock() }, true
}

func (d *TransferNoticeDispatcher) shutdownResultError() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.shutdown != nil && d.shutdown.Err() != nil {
		return noticeFailure("shutdown_timeout", d.shutdown.Err(), false)
	}
	return nil
}

func noticeMessage(c noticeClaim) (mailer.Message, error) {
	if !validUUID(c.NoticeID) || !validUUID(c.ArticleRef) {
		return mailer.Message{}, mailer.ErrInvalidMessage
	}
	line := ""
	switch c.Kind {
	case RecipientNewAuthor:
		line = "一篇文章已转移至您的名下。"
	case RecipientFormerDeletedAuthor:
		line = "您原有文章的归属已完成转移。"
	default:
		return mailer.Message{}, mailer.ErrInvalidMessage
	}
	message := mailer.Message{Purpose: mailer.PurposeOrphanArticleTransferNotice, To: c.EmailSnapshot,
		Subject: "Han article transfer notice", Body: line + "\n文章引用：" + c.ArticleRef + "\n通知引用：" + c.NoticeID + "\n此邮件仅告知已完成的归属变化，不授予访问权限。"}
	return message, mailer.ValidateMessage(message)
}

func classifyNoticeSMTP(err error) noticeResult {
	if err == nil {
		return noticeResult{disposition: noticeAccepted}
	}
	for _, local := range []struct {
		err    error
		bucket string
	}{{mailer.ErrInvalidMessage, "invalid_message"}, {mailer.ErrUnsupportedPurpose, "unsupported_purpose"}} {
		if errors.Is(err, local.err) {
			return noticeResult{disposition: noticeTerminal, bucket: local.bucket, stopRound: true}
		}
	}
	return classifyNoticeTransport(err)
}

func classifyNoticeTransport(err error) noticeResult {
	unknown := noticeResult{disposition: noticeRetry, bucket: "smtp_outcome_unknown", risk: true}
	var rejection *mailer.SMTPMessageRejection
	var protocol *textproto.Error
	if errors.As(err, &rejection) {
		if rejection == nil || !errors.Is(err, mailer.ErrMessageRejected) || !errors.As(rejection, &protocol) ||
			protocol == nil || protocol.Code != rejection.Code || protocol.Code < 400 || protocol.Code > 599 {
			return unknown
		}
		switch rejection.Stage {
		case mailer.SMTPMessageStageMAILFrom:
			return noticeResult{disposition: noticeRetry, bucket: "smtp_sender_or_policy", stopRound: true}
		case mailer.SMTPMessageStageRCPTTo, mailer.SMTPMessageStageDATACommand, mailer.SMTPMessageStageDATAFinal:
			if protocol.Code >= 500 {
				return noticeResult{disposition: noticeTerminal, bucket: "smtp_permanent_rejection"}
			}
			return noticeResult{disposition: noticeRetry, bucket: "smtp_transient_rejection"}
		default:
			return unknown
		}
	}
	if errors.Is(err, mailer.ErrMessageRejected) {
		return unknown
	}
	if errors.Is(err, mailer.ErrTLSRequired) || errors.Is(err, mailer.ErrInvalidSMTPConfig) || errors.Is(err, mailer.ErrDeadlineRequired) ||
		(errors.As(err, &protocol) && protocol != nil && protocol.Code >= 400 && protocol.Code <= 599) {
		return noticeResult{disposition: noticeRetry, bucket: "smtp_security_or_config", stopRound: true}
	}
	return unknown
}

func (d *TransferNoticeDispatcher) saveResult(ctx context.Context, c noticeClaim, r noticeResult) (bool, error) {
	work, cancel, tx, err := d.transaction(ctx)
	if err != nil {
		return false, noticeFailure("result_persistence_failed", err, false)
	}
	defer finishNoticeTx(work, cancel, tx)
	_, current, found, err := d.deps.Repository.LockNoticeRecipient(work, tx, c.NoticeID, c.Kind)
	if err != nil {
		return false, noticeDBFailure("result_persistence_failed", err)
	}
	if !found {
		return false, d.lostNoticeResult(ctx, work, cancel, tx, c, r)
	}
	now, _, err := noticeDBTime(work, tx)
	if err != nil {
		return false, err
	}
	if !noticeOwns(c, current, now) {
		return false, d.lostNoticeResult(ctx, work, cancel, tx, c, r)
	}
	return d.persistNoticeResult(ctx, work, tx, c, r, now)
}

func (d *TransferNoticeDispatcher) lostNoticeResult(ctx, work context.Context, cancel context.CancelFunc, tx pgx.Tx, c noticeClaim, r noticeResult) error {
	d.observe(ctx, "lease_competition")
	if err := tx.Rollback(work); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		return noticeFailure("late_result_risk_unpersisted", err, false)
	}
	cancel()
	if !r.attempted || c.issued == nil {
		return nil
	}
	issued := c
	issued.issued = nil
	if issued != *c.issued {
		return nil
	}
	return d.auditLateNoticeResult(ctx, c, r)
}

func (d *TransferNoticeDispatcher) auditLateNoticeResult(ctx context.Context, c noticeClaim, r noticeResult) error {
	work, cancel, tx, err := d.transaction(ctx)
	if err != nil {
		return noticeFailure("late_result_risk_unpersisted", err, false)
	}
	defer finishNoticeTx(work, cancel, tx)
	now, _, err := noticeDBTime(work, tx)
	if err != nil {
		return noticeFailure("late_result_risk_unpersisted", err, false)
	}
	if _, err = d.appendNoticeAudit(work, tx, c, r, now, nil, true); err != nil {
		return noticeFailure("late_result_risk_unpersisted", err, false)
	}
	if err := tx.Commit(work); err != nil {
		return noticeFailure("late_result_risk_unpersisted", err, false)
	}
	return nil
}

func (d *TransferNoticeDispatcher) prepareNoticeResult(ctx context.Context, c noticeClaim, r noticeResult) (noticeResult, error) {
	r.retryDelay = noticeBackoff(d.config.BackoffInitial, d.config.BackoffMax, c.AttemptCount)
	var err error
	r.eventID, err = d.deps.IDs.NewID()
	if err != nil || !validUUID(r.eventID) {
		return r, noticeFailure("audit_identity", err, true)
	}
	if r.traceID == "" {
		r.traceID, _ = platform.TraceIDFromContext(ctx)
	}
	// Shared shutdown context need not contain the working trace; carry it explicitly.
	if r.traceID == "" {
		r.traceID, err = d.deps.NewTraceID()
		if err != nil {
			return r, noticeFailure("trace_unavailable", err, false)
		}
	}
	if _, err := platform.ContextWithTraceID(ctx, r.traceID); err != nil {
		return r, noticeFailure("trace_invalid", err, true)
	}
	return r, nil
}

func (d *TransferNoticeDispatcher) persistNoticeResult(ctx, work context.Context, tx pgx.Tx, c noticeClaim, r noticeResult, now time.Time) (bool, error) {
	r, err := d.prepareNoticeResult(ctx, c, r)
	if err != nil {
		return false, err
	}
	after, changed, err := d.deps.Repository.SaveNoticeResult(work, tx, c, r, now)
	if err != nil {
		return false, noticeDBFailure("result_persistence_failed", err)
	}
	if !changed {
		return false, nil
	}
	event, err := d.appendNoticeAudit(work, tx, c, r, now, nil, false)
	if err != nil {
		return false, err
	}
	if err := tx.Commit(work); err != nil {
		// Release the uncertain transaction before acquiring a new bounded read transaction.
		_ = tx.Rollback(work)
		return d.readResult(ctx, c, after, event)
	}
	return true, nil
}

func (d *TransferNoticeDispatcher) deferNotice(ctx, work context.Context, tx pgx.Tx, c noticeClaim, bucket string, gateErr error, now time.Time) error {
	r := noticeResult{disposition: noticeDeferred, bucket: bucket}
	if noticeGateTransactionBroken(gateErr) {
		if err := tx.Rollback(work); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			return noticeDBFailure("preflight_rollback", err)
		}
		_, err := d.saveResult(ctx, c, r)
		return err
	}
	_, err := d.persistNoticeResult(ctx, work, tx, c, r, now)
	return err
}

func noticeGateTransactionBroken(err error) bool {
	if err == nil {
		return false
	}
	var pgerr interface{ SQLState() string }
	var neterr net.Error
	return errors.As(err, &pgerr) || errors.As(err, &neterr) || errors.Is(err, io.EOF) ||
		errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, pgx.ErrTxClosed)
}

func safeNoticeBucket(bucket string) string {
	switch bucket {
	case "processing_gate_closed", "lease_insufficient", "invalid_message", "unsupported_purpose",
		"smtp_sender_or_policy", "smtp_transient_rejection", "smtp_permanent_rejection", "smtp_security_or_config",
		"smtp_outcome_unknown", "attempts_exhausted", "attempts_exhausted_unknown":
		return bucket
	}
	return "smtp_outcome_unknown"
}

func (d *TransferNoticeDispatcher) appendNoticeAudit(ctx context.Context, tx pgx.Tx, c noticeClaim, r noticeResult, now time.Time, prior *TransferNoticeRecipient, riskOnly bool) (audit.Event, error) {
	event, err := d.noticeAudit(ctx, c, r, now, prior, riskOnly)
	if err != nil {
		return audit.Event{}, err
	}
	if err := d.deps.Audit.Append(ctx, tx, event); err != nil {
		return audit.Event{}, noticeDBFailure("result_persistence_failed", err)
	}
	if r.risk && !riskOnly {
		risk, err := d.noticeAudit(ctx, c, noticeResult{bucket: r.bucket, traceID: r.traceID}, now, prior, true)
		if err != nil {
			return audit.Event{}, err
		}
		if err := d.deps.Audit.Append(ctx, tx, risk); err != nil {
			return audit.Event{}, noticeDBFailure("result_persistence_failed", err)
		}
	}
	return event, nil
}

func (d *TransferNoticeDispatcher) noticeAudit(ctx context.Context, c noticeClaim, r noticeResult, now time.Time, prior *TransferNoticeRecipient, riskOnly bool) (audit.Event, error) {
	id := r.eventID
	var err error
	if id == "" {
		id, err = d.deps.IDs.NewID()
	}
	if err != nil || !validUUID(id) {
		return audit.Event{}, noticeFailure("audit_identity", err, true)
	}
	traceID := r.traceID
	if traceID == "" {
		traceID, _ = platform.TraceIDFromContext(ctx)
	}
	if _, err := platform.ContextWithTraceID(ctx, traceID); err != nil {
		return audit.Event{}, noticeFailure("trace_invalid", err, true)
	}
	outcome := noticeResultState(c, r)
	action := outcome
	if outcome == string(RecipientRetryWait) {
		action = "retry_scheduled"
	}
	if riskOnly {
		outcome = "risk"
		action = "duplicate_risk"
	}
	actor := "article-notice-dispatcher"
	metadata := map[string]any{"recipient_kind": string(c.Kind), "attempt_count": strconv.FormatInt(c.AttemptCount, 10), "max_attempts": strconv.FormatInt(c.MaxAttempts, 10),
		"claimed_by": fmt.Sprintf("%x", sha256.Sum256([]byte(c.ClaimedBy))), "locked_until": c.LockedUntil.UTC().Format(time.RFC3339Nano), "reason_bucket": r.bucket}
	if prior != nil {
		if riskOnly && prior.Status == RecipientProcessing {
			metadata["prior_attempt"] = strconv.FormatInt(prior.AttemptCount, 10)
			if prior.ClaimedBy != nil {
				metadata["prior_owner_hash"] = fmt.Sprintf("%x", sha256.Sum256([]byte(*prior.ClaimedBy)))
			}
		}
		if prior.FailureBucket != nil {
			metadata["prior_reason_bucket"] = safeNoticeBucket(*prior.FailureBucket)
		}
	}
	return audit.Event{ID: id, OccurredAt: now, ActorKind: audit.ActorSystem, ActorSystem: &actor,
		Action: "article.transfer_notice." + action, TargetType: "article_transfer_notice", TargetID: c.NoticeID,
		Outcome: outcome, TraceID: traceID, Metadata: metadata}, nil
}

func noticeResultState(c noticeClaim, r noticeResult) string {
	if r.disposition == noticeAccepted {
		return string(RecipientAccepted)
	}
	if r.disposition == noticeTerminal || (c.AttemptCount >= c.MaxAttempts && r.bucket != "processing_gate_closed") {
		return string(RecipientFailedTerminal)
	}
	return string(RecipientRetryWait)
}

func (d *TransferNoticeDispatcher) readResult(ctx context.Context, c noticeClaim, expected TransferNoticeRecipient, event audit.Event) (bool, error) {
	work, cancel, tx, err := d.transaction(ctx)
	if err != nil {
		return false, noticeFailure("result_commit_unknown", err, false)
	}
	defer finishNoticeTx(work, cancel, tx)
	_, actual, found, err := d.deps.Repository.ReadNoticeRecipient(work, tx, c.NoticeID, c.Kind)
	if err != nil || !found {
		return false, noticeFailure("result_commit_unknown", err, false)
	}
	stored, err := d.deps.Audit.Get(work, tx, event.ID)
	if err != nil || !noticeSameResult(actual, expected) || !noticeAuditBound(stored, event) {
		return false, noticeFailure("result_commit_unknown", err, false)
	}
	if err := tx.Commit(work); err != nil {
		return false, noticeDBFailure("result_commit_unknown", err)
	}
	return true, nil
}

func noticeSameResult(a, b TransferNoticeRecipient) bool {
	return a.NoticeID == b.NoticeID && a.Kind == b.Kind && a.Status == b.Status && a.AttemptCount == b.AttemptCount &&
		a.MaxAttempts == b.MaxAttempts && reflect.DeepEqual(a.ClaimedBy, b.ClaimedBy) && reflect.DeepEqual(a.LockedUntil, b.LockedUntil) &&
		reflect.DeepEqual(a.NextAttemptAt, b.NextAttemptAt) && reflect.DeepEqual(a.AcceptedAt, b.AcceptedAt) &&
		reflect.DeepEqual(a.FailureBucket, b.FailureBucket) && a.UpdatedAt.Equal(b.UpdatedAt)
}

func noticeAuditBound(a, b audit.Event) bool {
	if a.ID != b.ID || a.ActorKind != audit.ActorSystem || a.ActorSystem == nil || b.ActorSystem == nil || *a.ActorSystem != *b.ActorSystem ||
		a.TargetType != b.TargetType || a.TargetID != b.TargetID || a.Action != b.Action || a.Outcome != b.Outcome ||
		a.TraceID != b.TraceID || !a.OccurredAt.Equal(b.OccurredAt) {
		return false
	}
	for _, key := range []string{"recipient_kind", "attempt_count", "max_attempts", "claimed_by", "locked_until", "reason_bucket"} {
		actual, actualString := a.Metadata[key].(string)
		expected, expectedString := b.Metadata[key].(string)
		if !actualString || !expectedString || actual != expected {
			return false
		}
	}
	return true
}

// Started closes when the actual serial polling goroutine has entered its loop.
func (d *TransferNoticeDispatcher) Started() <-chan struct{} { return d.loopStarted }

func (d *TransferNoticeDispatcher) acquireWork(ctx context.Context) (context.Context, func(), error) {
	if d.deps.AcquireWork == nil {
		return ctx, func() {}, nil
	}
	return d.deps.AcquireWork(ctx)
}
func (d *TransferNoticeDispatcher) maintainWork(ctx context.Context) (bool, error) {
	work, release, err := d.acquireWork(ctx)
	if err != nil {
		return false, err
	}
	defer release()
	return d.maintain(work)
}
func (d *TransferNoticeDispatcher) dispatchWork(ctx context.Context) (bool, error) {
	work, release, err := d.acquireWork(ctx)
	if err != nil {
		return true, err
	}
	defer release()
	claim, found, err := d.claimDue(work)
	if err != nil || !found {
		return true, err
	}
	if err := work.Err(); err != nil {
		return true, err
	}
	return d.deliver(work, claim)
}
