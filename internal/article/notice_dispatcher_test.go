package article

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/big"
	"net"
	"net/textproto"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/quick"
	"time"

	"han/internal/audit"
	"han/internal/mailer"
	"han/internal/platform"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func noticeTestConfig() NoticeDispatcherConfig {
	return NoticeDispatcherConfig{PollInterval: time.Second, BatchLimit: 2,
		LeaseDuration: 10 * time.Second, DBOperationBudget: time.Second,
		BackoffInitial: time.Second, BackoffMax: 8 * time.Second,
		SMTPOperationBudget: 2 * time.Second}
}

func TestNoticeDispatcherConstructor_TypedNilAndIdentity_Rejected(t *testing.T) {
	m, _ := newNoticeMemory(t)
	var missing *noticeMemory
	tests := []struct {
		name   string
		change func(*NoticeDispatcherDependencies)
	}{
		{"transactions", func(d *NoticeDispatcherDependencies) { d.Transactions = missing }},
		{"gate", func(d *NoticeDispatcherDependencies) { d.Gate = missing }},
		{"mailer", func(d *NoticeDispatcherDependencies) { d.Mailer = missing }},
		{"audit", func(d *NoticeDispatcherDependencies) { d.Audit = missing }},
		{"ids", func(d *NoticeDispatcherDependencies) { d.IDs = missing }},
		{"logger", func(d *NoticeDispatcherDependencies) { d.Logger = missing }},
		{"trace", func(d *NoticeDispatcherDependencies) { d.NewTraceID = nil }},
		{"worker", func(d *NoticeDispatcherDependencies) { d.WorkerID = "invalid" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := m.dependencies()
			tt.change(&deps)
			if _, err := NewTransferNoticeDispatcher(deps, noticeTestConfig()); err == nil {
				t.Fatal("invalid dependency accepted")
			}
			if m.begins != 0 || m.sends != 0 {
				t.Fatal("constructor performed I/O")
			}
		})
	}
}

func TestNoticeDispatcherMaintenance_ExhaustionStates_GateIndependent(t *testing.T) {
	tests := []struct {
		name   string
		status RecipientStatus
		future bool
		want   RecipientStatus
		risk   bool
	}{
		{"pending", RecipientPending, false, RecipientFailedTerminal, false},
		{"processing_expired", RecipientProcessing, false, RecipientFailedTerminal, true},
		{"processing_live", RecipientProcessing, true, RecipientProcessing, false},
		{"retry_due", RecipientRetryWait, false, RecipientFailedTerminal, true},
		{"retry_future", RecipientRetryWait, true, RecipientRetryWait, false},
		{"accepted", RecipientAccepted, false, RecipientAccepted, false},
		{"terminal", RecipientFailedTerminal, false, RecipientFailedTerminal, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, d := newNoticeMemory(t)
			r := &m.rows[0]
			r.Status = tt.status
			r.AttemptCount = 3
			at := m.now
			if tt.future {
				at = at.Add(time.Hour)
			}
			fixtureNoticeDates(r, at, m.now)
			m.gate = func(context.Context, *noticeMemoryTx) error { return errors.New("closed") }
			if err := d.Reconcile(context.Background()); err != nil {
				t.Fatal("maintenance round failed")
			}
			if m.rows[0].Status != tt.want || m.sends != 0 || m.rows[0].AttemptCount != 3 {
				t.Fatal("maintenance state contract failed")
			}
			wantEvents := 0
			if tt.want != tt.status {
				wantEvents = 1
				if tt.risk {
					wantEvents++
				}
			}
			if len(m.events) != wantEvents {
				t.Fatal("exhaustion audit count failed")
			}
		})
	}
}

func fixtureNoticeDates(r *TransferNoticeRecipient, at, now time.Time) {
	owner := noticeTestWorker
	r.ClaimedBy = &owner
	r.LockedUntil = &at
	r.LastAttemptAt = &now
	r.NextAttemptAt = &at
	if r.Status == RecipientAccepted {
		r.AcceptedAt = &now
	}
}

func TestTransferNoticeBudget_ElapsedAndCallerDeadline_BoundsSend(t *testing.T) {
	now := time.Now()
	if noticeSendBudget(context.Background(), time.Second, time.Second, now.Add(-2*time.Second)) > 0 {
		t.Fatal("query elapsed time ignored")
	}
	ctx, cancel := context.WithDeadline(context.Background(), now.Add(100*time.Millisecond))
	defer cancel()
	if budget := noticeSendBudget(ctx, time.Second, 2*time.Second, now); budget <= 0 || budget > 100*time.Millisecond {
		t.Fatal("caller budget ignored")
	}
	cancel()
	if noticeSendBudget(ctx, time.Second, time.Second, now) > 0 {
		t.Fatal("canceled caller granted budget")
	}
}

func TestNoticeDispatcherShutdown_ExpiredSharedDeadline_ErrorRetained(t *testing.T) {
	m, d := newNoticeMemory(t)
	entered, release := make(chan struct{}), make(chan struct{})
	m.mail = func(ctx context.Context, _ mailer.Message) error { close(entered); <-ctx.Done(); <-release; return nil }
	round := make(chan error, 1)
	go func() { round <- d.Reconcile(context.Background()) }()
	<-entered
	shared, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	d.BeginShutdown(shared)
	close(release)
	if err := <-round; err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("expired shared deadline reported success")
	}
	if m.rows[0].Status != RecipientProcessing || m.sends != 1 || len(m.events) != 0 {
		t.Fatal("expired deadline started settlement")
	}
}

func TestNoticeDispatcherWait_TimeoutDuringSharedSave_CancelsSettlement(t *testing.T) {
	m, d := newNoticeMemory(t)
	mailEntered, saveEntered := make(chan struct{}), make(chan struct{})
	m.mail = func(ctx context.Context, _ mailer.Message) error { close(mailEntered); <-ctx.Done(); return nil }
	var saveCanceled error
	m.onSave = func(ctx context.Context) { close(saveEntered); <-ctx.Done(); saveCanceled = ctx.Err() }
	round := make(chan error, 1)
	go func() { round <- d.Reconcile(context.Background()) }()
	<-mailEntered
	shared, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	d.BeginShutdown(shared)
	<-saveEntered
	waitCtx, cancelWait := context.WithCancel(context.Background())
	cancelWait()
	if err := d.Wait(waitCtx); err == nil {
		t.Fatal("Wait timeout lost")
	}
	if err := <-round; err == nil {
		t.Fatal("canceled shared save reported success")
	}
	if !errors.Is(saveCanceled, context.Canceled) || shared.Err() != nil {
		t.Fatal("Wait failed to cancel settlement within shared deadline")
	}
	if m.rows[0].Status != RecipientProcessing || m.sends != 1 || len(m.events) != 0 {
		t.Fatal("canceled settlement restarted or committed")
	}
}

func TestNoticeDispatcherRetry_TwoKindsSameAddress_IndependentAndImmutable(t *testing.T) {
	m, d := newNoticeMemory(t)
	other := m.rows[0]
	other.Kind = RecipientFormerDeletedAuthor
	m.rows = append(m.rows, other)
	m.mail = func(context.Context, mailer.Message) error { return errors.New("unknown") }
	if err := d.Reconcile(context.Background()); err != nil {
		t.Fatal("retry round failed")
	}
	if m.sends != 2 || m.rows[0].Status != RecipientRetryWait || m.rows[1].Status != RecipientRetryWait {
		t.Fatal("same address recipients merged")
	}
	for _, r := range m.rows {
		if r.MaxAttempts != 3 || r.EmailSnapshot != other.EmailSnapshot || r.NoticeID != noticeTestID || r.AttemptCount != 1 || r.LockedUntil != nil || r.AcceptedAt != nil {
			t.Fatal("immutable snapshot or retry fields changed")
		}
	}
	m.now = *m.rows[0].NextAttemptAt
	m.mail = nil
	if err := d.Reconcile(context.Background()); err != nil {
		t.Fatal("retry recovery failed")
	}
	if m.sends != 4 || m.rows[0].Status != RecipientAccepted || m.rows[1].Status != RecipientAccepted {
		t.Fatal("due retries not recovered")
	}
}

func TestNoticeDispatcherPreflight_InsufficientOrExpired_NeverSends(t *testing.T) {
	for _, advance := range []time.Duration{7 * time.Second, 10 * time.Second} {
		t.Run(advance.String(), func(t *testing.T) {
			m, d := newNoticeMemory(t)
			clocks := 0
			m.onClock = func(context.Context) {
				clocks++
				if clocks == 2 {
					m.now = m.now.Add(advance)
				}
			}
			if err := d.Reconcile(context.Background()); err != nil {
				t.Fatal("preflight failed")
			}
			want := RecipientRetryWait
			if advance == 10*time.Second {
				want = RecipientProcessing
			}
			if m.sends != 0 || m.rows[0].Status != want {
				t.Fatal("unsafe preflight sent")
			}
		})
	}
}

func TestNoticeDispatcherPreflight_CommitUnknown_NeverSends(t *testing.T) {
	m, d := newNoticeMemory(t)
	m.commitFault[3] = true
	if err := d.Reconcile(context.Background()); err != nil {
		t.Fatal("recoverable preflight commit unknown")
	}
	if m.sends != 0 || m.rows[0].Status != RecipientProcessing {
		t.Fatal("unknown preflight granted send rights")
	}
}

func TestNoticeDispatcherFencing_AnyOwnershipMismatch_Lost(t *testing.T) {
	m, d := newNoticeMemory(t)
	ctx := noticeTestContext(t)
	c, found, err := d.claimDue(ctx)
	if err != nil || !found {
		t.Fatal("fixture claim failed")
	}
	tests := []struct {
		name   string
		change func(*noticeClaim)
	}{
		{"pk", func(c *noticeClaim) { c.NoticeID = noticeTestArticle }},
		{"kind", func(c *noticeClaim) { c.Kind = RecipientFormerDeletedAuthor }},
		{"owner", func(c *noticeClaim) { c.ClaimedBy = noticeTestArticle }},
		{"attempt_ABA", func(c *noticeClaim) { c.AttemptCount++ }},
		{"lease_microsecond", func(c *noticeClaim) { c.LockedUntil = c.LockedUntil.Add(time.Microsecond) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrong := c
			tt.change(&wrong)
			applied, err := d.saveResult(ctx, wrong, noticeResult{disposition: noticeAccepted})
			if err != nil || applied || m.rows[0].Status != RecipientProcessing || len(m.events) != 0 {
				t.Fatal("stale claim overwrote owner")
			}
		})
	}
	m.now = c.LockedUntil
	if applied, err := d.saveResult(ctx, c, noticeResult{disposition: noticeAccepted}); err != nil || applied {
		t.Fatal("expired unclaimed lease saved")
	}
}

func TestNoticeDispatcherReclaim_AuditFailure_RollsBackNewAttempt(t *testing.T) {
	m, d := newNoticeMemory(t)
	ctx := noticeTestContext(t)
	c, ok, err := d.claimDue(ctx)
	if err != nil || !ok {
		t.Fatal("fixture claim failed")
	}
	m.now = c.LockedUntil
	m.auditErr = errors.New("sensitive-canary")
	if _, ok, err := d.claimDue(ctx); err == nil || ok {
		t.Fatal("reclaim granted rights without risk audit")
	}
	if m.rows[0].AttemptCount != 1 || m.sends != 0 || len(m.events) != 0 {
		t.Fatal("reclaim audit rollback failed")
	}
	m.auditErr = nil
	if _, ok, err := d.claimDue(ctx); err != nil || !ok || m.rows[0].AttemptCount != 2 || len(m.events) != 1 {
		t.Fatal("reclaim recovery failed")
	}
}

func TestNoticeDispatcherBounds_HugeBatchEmptyQueue_NoPreallocation(t *testing.T) {
	m, d := newNoticeMemory(t)
	m.rows = nil
	d.config.BatchLimit = math.MaxInt32
	if err := d.Reconcile(context.Background()); err != nil {
		t.Fatal("empty queue failed")
	}
	if m.begins != 2 || m.active != 0 {
		t.Fatal("empty queue did not stop promptly")
	}
}

func TestTransferNoticeClock_SamplePrecedesQueryRoundTrip_Conservative(t *testing.T) {
	m, _ := newNoticeMemory(t)
	var queryStarted time.Time
	m.onClock = func(context.Context) { queryStarted = time.Now() }
	tx, err := m.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		t.Fatal("fixture begin failed")
	}
	defer tx.Rollback(context.Background())
	now, sampled, err := noticeDBTime(context.Background(), tx)
	if err != nil || sampled.After(queryStarted) || now.Nanosecond()%1000 != 0 {
		t.Fatal("clock sample excluded query round trip")
	}
}

func TestTransferNoticeBackoff_Properties_BoundedAndMonotonic(t *testing.T) {
	property := func(a uint32, b uint16, n uint64) bool {
		initial := time.Duration(a%100000+1) * time.Microsecond
		maximum := initial + time.Duration(b)*time.Microsecond
		attempt := int64(n%math.MaxInt64) + 1
		value := noticeBackoff(initial, maximum, attempt)
		return value >= initial && value <= maximum && value%time.Microsecond == 0 &&
			(attempt == math.MaxInt64 || value <= noticeBackoff(initial, maximum, attempt+1))
	}
	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Fatal("backoff property violated")
	}
}

func TestNoticeDispatcherShutdown_InitialActive_AcceptedSavedOnceBeforeDone(t *testing.T) {
	m, d := newNoticeMemory(t)
	other := m.rows[0]
	other.Kind = RecipientFormerDeletedAuthor
	m.rows = append(m.rows, other)
	entered, release := make(chan struct{}), make(chan struct{})
	m.mail = func(ctx context.Context, _ mailer.Message) error { close(entered); <-ctx.Done(); <-release; return nil }
	result := make(chan error, 1)
	go func() { result <- d.Reconcile(context.Background()) }()
	<-entered
	shared, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	want, _ := shared.Deadline()
	saves := 0
	m.onSave = func(ctx context.Context) {
		saves++
		got, _ := ctx.Deadline()
		if got.After(want) {
			t.Error("save exceeded shared deadline")
		}
	}
	d.BeginShutdown(shared)
	d.Stop()
	d.BeginShutdown(context.Background())
	select {
	case <-d.Done():
		t.Fatal("Done closed before initial call returned")
	default:
	}
	close(release)
	if err := <-result; err != nil {
		t.Fatal("accepted shutdown save failed")
	}
	if err := d.Wait(shared); err != nil {
		t.Fatal("shutdown join failed")
	}
	if saves != 1 || m.sends != 1 || m.rows[0].Status != RecipientAccepted || m.rows[1].Status != RecipientPending {
		t.Fatal("shutdown saved twice or started successor")
	}
	if m.events[0].TraceID != noticeTestTrace {
		t.Fatal("shared shutdown detached original trace")
	}
}

func TestNoticeDispatcherShutdown_NoSharedContext_LeavesProcessing(t *testing.T) {
	m, d := newNoticeMemory(t)
	entered := make(chan struct{})
	m.mail = func(ctx context.Context, _ mailer.Message) error { close(entered); <-ctx.Done(); return ctx.Err() }
	result := make(chan error, 1)
	go func() { result <- d.Reconcile(context.Background()) }()
	<-entered
	d.Stop()
	<-result
	if err := d.Wait(context.Background()); err != nil {
		t.Fatal("manual stop failed")
	}
	if m.rows[0].Status != RecipientProcessing || len(m.events) != 0 || m.sends != 1 {
		t.Fatal("invented shutdown save budget")
	}
}

func TestNoticeDispatcherWait_Timeout_CancelsAndJoinsActualExit(t *testing.T) {
	m, d := newNoticeMemory(t)
	entered, release := make(chan struct{}), make(chan struct{})
	m.mail = func(ctx context.Context, _ mailer.Message) error {
		close(entered)
		<-ctx.Done()
		<-release
		return ctx.Err()
	}
	round := make(chan error, 1)
	go func() { round <- d.Reconcile(context.Background()) }()
	<-entered
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	wait := make(chan error, 1)
	go func() { wait <- d.Wait(ctx) }()
	select {
	case <-wait:
		t.Fatal("Wait abandoned actual operation")
	case <-d.Done():
		t.Fatal("Done closed while Send active")
	default:
	}
	close(release)
	<-round
	if err := <-wait; err == nil || !errors.Is(err, context.Canceled) {
		t.Fatal("timeout not retained")
	}
}

func TestNoticeDispatcherShutdown_SaveAlreadyStarted_NoRestart(t *testing.T) {
	m, d := newNoticeMemory(t)
	entered := make(chan struct{})
	saves := 0
	m.onSave = func(ctx context.Context) { saves++; close(entered); <-ctx.Done() }
	round := make(chan error, 1)
	go func() { round <- d.Reconcile(context.Background()) }()
	<-entered
	shared, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	d.BeginShutdown(shared)
	if err := <-round; err == nil {
		t.Fatal("canceled save reported success")
	}
	if saves != 1 || m.sends != 1 || m.rows[0].Status != RecipientProcessing || len(m.events) != 0 {
		t.Fatal("save restarted after cancellation")
	}
	if err := d.Wait(shared); err == nil {
		t.Fatal("persistence failure lost at Wait")
	}
}

func TestNoticeDispatcherConcurrency_RepeatedStartAndReconcile_OneActiveSend(t *testing.T) {
	m, d := newNoticeMemory(t)
	entered, release := make(chan struct{}), make(chan struct{})
	m.mail = func(ctx context.Context, _ mailer.Message) error { close(entered); <-release; return nil }
	round := make(chan error, 1)
	go func() { round <- d.Reconcile(context.Background()) }()
	<-entered
	d.Start(context.Background())
	d.Start(context.Background())
	if err := d.Reconcile(context.Background()); err != nil {
		t.Fatal("concurrent round should coalesce")
	}
	shared, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	d.BeginShutdown(shared)
	close(release)
	<-round
	if err := d.Wait(shared); err != nil {
		t.Fatal("concurrent shutdown failed")
	}
	if m.sends != 1 {
		t.Fatal("overlapping sends")
	}
}

func noticeTestTLS(t *testing.T) (*tls.Config, *tls.Config) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal("fixture key generation failed")
	}
	now := time.Now()
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "synthetic.invalid"},
		DNSNames: []string{"synthetic.invalid"}, NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, IsCA: true, BasicConstraintsValid: true}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal("fixture certificate failed")
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal("fixture certificate parse failed")
	}
	roots := x509.NewCertPool()
	roots.AddCert(cert)
	return &tls.Config{MinVersion: tls.VersionTLS12, ServerName: "synthetic.invalid", RootCAs: roots},
		&tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}}
}

type noticePipeDialer struct{ connection net.Conn }

func (d noticePipeDialer) DialContext(context.Context, string, string) (net.Conn, error) {
	return d.connection, nil
}

func noticeAdapterRejection(t *testing.T, stage mailer.SMTPMessageStage, code int) error {
	t.Helper()
	clientTLS, serverTLS := noticeTestTLS(t)
	client, server := net.Pipe()
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		defer server.Close()
		_ = server.SetDeadline(time.Now().Add(2 * time.Second))
		noticeSMTPConversation(tls.Server(server, serverTLS), stage, code)
	}()
	adapter, err := mailer.NewSMTPMailer(mailer.SMTPConfig{Address: "synthetic.invalid:1", Host: "synthetic.invalid",
		From: "sender@example.invalid", Mode: mailer.TLSModeImplicit, TLSConfig: clientTLS}, noticePipeDialer{client})
	if err != nil {
		t.Fatal("fixture adapter failed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err = adapter.Send(ctx, mailer.Message{Purpose: mailer.PurposeOrphanArticleTransferNotice, To: "recipient@example.invalid", Subject: "synthetic", Body: "synthetic"})
	<-finished
	var rejection *mailer.SMTPMessageRejection
	if !errors.As(err, &rejection) {
		t.Fatal("fixture failed to obtain actual typed rejection")
	}
	return err
}

func noticeSMTPConversation(connection net.Conn, stage mailer.SMTPMessageStage, code int) {
	protocol := textproto.NewConn(connection)
	if err := protocol.PrintfLine("220 synthetic"); err != nil {
		return
	}
	if _, err := protocol.ReadLine(); err != nil {
		return
	}
	if err := protocol.PrintfLine("250 synthetic"); err != nil {
		return
	}
	for _, current := range []mailer.SMTPMessageStage{mailer.SMTPMessageStageMAILFrom, mailer.SMTPMessageStageRCPTTo, mailer.SMTPMessageStageDATACommand} {
		if _, err := protocol.ReadLine(); err != nil {
			return
		}
		if current == stage {
			_ = protocol.PrintfLine("%d synthetic-canary", code)
			return
		}
		reply := 250
		if current == mailer.SMTPMessageStageDATACommand {
			reply = 354
		}
		if err := protocol.PrintfLine("%d synthetic", reply); err != nil {
			return
		}
	}
	if _, err := protocol.ReadDotBytes(); err != nil {
		return
	}
	_ = protocol.PrintfLine("%d synthetic-canary", code)
}

func TestTransferNoticeClassifier_ActualAdapterStages_ConsistentPolicy(t *testing.T) {
	for _, stage := range []mailer.SMTPMessageStage{mailer.SMTPMessageStageMAILFrom, mailer.SMTPMessageStageRCPTTo, mailer.SMTPMessageStageDATACommand, mailer.SMTPMessageStageDATAFinal} {
		for _, code := range []int{421, 450, 550} {
			t.Run(fmt.Sprintf("%s_%d", stage, code), func(t *testing.T) {
				err := noticeAdapterRejection(t, stage, code)
				r := classifyNoticeSMTP(err)
				bucket := "smtp_transient_rejection"
				disposition := noticeRetry
				if stage == mailer.SMTPMessageStageMAILFrom {
					bucket = "smtp_sender_or_policy"
				} else if code >= 500 {
					bucket = "smtp_permanent_rejection"
					disposition = noticeTerminal
				}
				if r.bucket != bucket || r.disposition != disposition || r.risk || r.stopRound != (stage == mailer.SMTPMessageStageMAILFrom) {
					t.Fatal("typed stage policy mismatch")
				}
				var typed *mailer.SMTPMessageRejection
				_ = errors.As(err, &typed)
				typed.Code++
				if bad := classifyNoticeSMTP(err); bad.bucket != "smtp_outcome_unknown" || !bad.risk {
					t.Fatal("inconsistent code trusted")
				}
			})
		}
	}
}

const noticeTestID = "10000000-0000-0000-0000-000000000001"
const noticeTestArticle = "20000000-0000-0000-0000-000000000001"
const noticeTestWorker = "30000000-0000-0000-0000-000000000001"
const noticeTestTrace = "01ARZ3NDEKTSV4RRFFQ69G5FAV"

// This transaction fixture exercises the real Repository/dispatcher calls. It
// models commit/rollback faults, not PostgreSQL isolation or locking guarantees.
type noticeMemory struct {
	mu          sync.Mutex
	rows        []TransferNoticeRecipient
	events      []audit.Event
	now         time.Time
	active      int
	begins      int
	gateCalls   int
	sends       int
	ids         int
	logs        []string
	queries     []string
	commitFault map[int]bool // presence injects lost ACK; value chooses durable commit.
	auditErr    error
	beginErr    error
	readErr     error
	updateErr   error
	gate        func(context.Context, *noticeMemoryTx) error
	mail        func(context.Context, mailer.Message) error
	onClock     func(context.Context)
	onSave      func(context.Context)
	onAuditRead func(*audit.Event)
	onAppend    func(context.Context, *noticeMemoryTx, audit.Event) error
	onBegin     func(context.Context, int) error
}

type noticeMemoryTx struct {
	*createArticleTxStub
	db      *noticeMemory
	number  int
	rows    []TransferNoticeRecipient
	events  []audit.Event
	closed  bool
	aborted bool
}

type noticeMemoryRow struct {
	values []any
	err    error
}

func (r noticeMemoryRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return errors.New("fixture column count")
	}
	for i, value := range r.values {
		out := reflect.ValueOf(dest[i]).Elem()
		if value == nil {
			out.SetZero()
		} else {
			out.Set(reflect.ValueOf(value).Convert(out.Type()))
		}
	}
	return nil
}

func newNoticeMemory(t *testing.T) (*noticeMemory, *TransferNoticeDispatcher) {
	t.Helper()
	now := time.Date(2026, 10, 27, 0, 0, 0, 123456000, time.UTC)
	m := &noticeMemory{now: now, commitFault: map[int]bool{}, rows: []TransferNoticeRecipient{
		{NoticeID: noticeTestID, Kind: RecipientNewAuthor, EmailSnapshot: "synthetic@example.invalid",
			Status: RecipientPending, MaxAttempts: 3, CreatedAt: now, UpdatedAt: now},
	}}
	d, err := NewTransferNoticeDispatcher(m.dependencies(), noticeTestConfig())
	if err != nil {
		t.Fatal("fixture constructor failed")
	}
	t.Cleanup(func() {
		d.Stop()
		if err := d.Wait(context.Background()); err != nil { /* faults are asserted by their tests */
		}
	})
	return m, d
}

func (m *noticeMemory) dependencies() NoticeDispatcherDependencies {
	return NoticeDispatcherDependencies{Transactions: m, Repository: Repository{}, Gate: m, Mailer: m, Audit: m, IDs: m,
		NewTraceID: func() (string, error) { return noticeTestTrace, nil }, Logger: m, WorkerID: noticeTestWorker}
}

func cloneNoticeRows(rows []TransferNoticeRecipient) []TransferNoticeRecipient {
	result := make([]TransferNoticeRecipient, len(rows))
	copy(result, rows)
	return result
}

func (m *noticeMemory) BeginTx(ctx context.Context, options pgx.TxOptions) (pgx.Tx, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if options.IsoLevel != pgx.ReadCommitted {
		return nil, errors.New("fixture expected explicit read committed")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.beginErr != nil {
		return nil, m.beginErr
	}
	if m.onBegin != nil {
		if err := m.onBegin(ctx, m.begins+1); err != nil {
			return nil, err
		}
	}
	m.begins++
	m.active++
	return &noticeMemoryTx{createArticleTxStub: &createArticleTxStub{}, db: m, number: m.begins,
		rows: cloneNoticeRows(m.rows), events: append([]audit.Event(nil), m.events...)}, nil
}

func (tx *noticeMemoryTx) Commit(ctx context.Context) error {
	if tx.closed {
		return pgx.ErrTxClosed
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	m := tx.db
	m.mu.Lock()
	defer m.mu.Unlock()
	if tx.aborted {
		return errors.New("fixture aborted transaction")
	}
	durable, fault := m.commitFault[tx.number]
	if !fault || durable {
		m.rows = cloneNoticeRows(tx.rows)
		m.events = append([]audit.Event(nil), tx.events...)
	}
	tx.closed = true
	m.active--
	if fault {
		return errors.New("synthetic lost commit acknowledgement")
	}
	return nil
}

func (tx *noticeMemoryTx) Rollback(context.Context) error {
	tx.db.mu.Lock()
	defer tx.db.mu.Unlock()
	if tx.closed {
		return pgx.ErrTxClosed
	}
	tx.closed = true
	tx.db.active--
	return nil
}

func (tx *noticeMemoryTx) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	if err := ctx.Err(); err != nil {
		return noticeMemoryRow{err: err}
	}
	if tx.closed {
		return noticeMemoryRow{err: pgx.ErrTxClosed}
	}
	if tx.aborted {
		return noticeMemoryRow{err: &pgconn.PgError{Code: "25P02"}}
	}
	m := tx.db
	m.mu.Lock()
	m.queries = append(m.queries, query)
	m.mu.Unlock()
	if query == "SELECT clock_timestamp()" {
		if m.onClock != nil {
			m.onClock(ctx)
		}
		m.mu.Lock()
		now := m.now
		m.mu.Unlock()
		return noticeMemoryRow{values: []any{now}}
	}
	if query == claimNoticeRecipientSQL || query == saveNoticeResultSQL || query == finishExhaustedNoticeRecipientSQL {
		return tx.update(ctx, query, args)
	}
	return tx.read(query, args)
}

func noticeRowValues(r TransferNoticeRecipient) []any {
	return []any{r.NoticeID, r.Kind, r.EmailSnapshot, r.Status, r.ClaimedBy, r.LockedUntil, r.NextAttemptAt,
		r.AttemptCount, r.MaxAttempts, r.LastAttemptAt, r.AcceptedAt, r.FailureBucket, r.CreatedAt, r.UpdatedAt}
}

func (tx *noticeMemoryTx) read(query string, args []any) pgx.Row {
	if tx.db.readErr != nil {
		return noticeMemoryRow{err: tx.db.readErr}
	}
	for _, r := range tx.rows {
		match := false
		switch query {
		case lockDueNoticeRecipientSQL:
			match = r.AttemptCount < r.MaxAttempts && noticeDue(r, tx.db.now)
		case lockExhaustedNoticeRecipientSQL:
			match = r.AttemptCount == r.MaxAttempts && noticeDue(r, tx.db.now)
		case readNoticeRecipientSQL, readNoticeRecipientSQL + " FOR UPDATE OF recipient":
			match = r.NoticeID == args[0] && r.Kind == args[1]
		default:
			return noticeMemoryRow{err: errors.New("unexpected fixture read")}
		}
		if match {
			return noticeMemoryRow{values: append([]any{noticeTestArticle}, noticeRowValues(r)...)}
		}
	}
	return noticeMemoryRow{err: pgx.ErrNoRows}
}

func (tx *noticeMemoryTx) update(ctx context.Context, query string, args []any) pgx.Row {
	if tx.db.updateErr != nil {
		return noticeMemoryRow{err: tx.db.updateErr}
	}
	if query == saveNoticeResultSQL && tx.db.onSave != nil {
		tx.db.onSave(ctx)
	}
	if err := ctx.Err(); err != nil {
		return noticeMemoryRow{err: err}
	}
	for i, r := range tx.rows {
		if r.NoticeID != args[0] || r.Kind != args[1] {
			continue
		}
		var changed bool
		if query == saveNoticeResultSQL {
			r, changed = fixtureNoticeSave(r, args)
		} else {
			r, changed = fixtureNoticePreimage(r, query, args)
		}
		if !changed {
			return noticeMemoryRow{err: pgx.ErrNoRows}
		}
		tx.rows[i] = r
		return noticeMemoryRow{values: noticeRowValues(r)}
	}
	return noticeMemoryRow{err: pgx.ErrNoRows}
}

func fixtureNoticePreimage(r TransferNoticeRecipient, query string, a []any) (TransferNoticeRecipient, bool) {
	if r.Status != a[2] || r.AttemptCount != a[3] || !reflect.DeepEqual(r.ClaimedBy, a[4]) ||
		!reflect.DeepEqual(r.LockedUntil, a[5]) || !reflect.DeepEqual(r.NextAttemptAt, a[6]) {
		return r, false
	}
	if query == claimNoticeRecipientSQL {
		now := a[8].(time.Time)
		if !noticeDue(r, now) || r.AttemptCount >= r.MaxAttempts {
			return r, false
		}
		owner, until := a[7].(string), a[9].(time.Time)
		r.Status = RecipientProcessing
		r.AttemptCount++
		r.ClaimedBy = &owner
		r.LockedUntil = &until
		r.LastAttemptAt = &now
		r.UpdatedAt = now
		r.NextAttemptAt = nil
		r.FailureBucket = nil
		r.AcceptedAt = nil
	} else {
		now := a[7].(time.Time)
		if !noticeDue(r, now) || r.AttemptCount != r.MaxAttempts {
			return r, false
		}
		bucket := a[8].(string)
		r.Status = RecipientFailedTerminal
		r.FailureBucket = &bucket
		r.UpdatedAt = now
		r.ClaimedBy = nil
		r.LockedUntil = nil
		r.NextAttemptAt = nil
		r.AcceptedAt = nil
	}
	return r, true
}

func fixtureNoticeSave(r TransferNoticeRecipient, a []any) (TransferNoticeRecipient, bool) {
	now := a[5].(time.Time)
	c := noticeClaim{NoticeID: a[0].(string), Kind: a[1].(RecipientKind), ClaimedBy: a[2].(string),
		AttemptCount: a[3].(int64), LockedUntil: a[4].(time.Time)}
	if !noticeOwns(c, r, now) {
		return r, false
	}
	r.Status = RecipientStatus(a[6].(string))
	r.NextAttemptAt = a[7].(*time.Time)
	r.AcceptedAt = a[8].(*time.Time)
	r.FailureBucket = a[9].(*string)
	r.ClaimedBy = nil
	r.LockedUntil = nil
	r.UpdatedAt = now
	return r, true
}

func (m *noticeMemory) RequireOpen(context.Context) error {
	return errors.New("pool Gate must not be used")
}
func (m *noticeMemory) RequireOpenInTx(ctx context.Context, tx pgx.Tx) error {
	transaction, ok := tx.(*noticeMemoryTx)
	if !ok || transaction.db != m || transaction.closed {
		return errors.New("wrong caller transaction")
	}
	m.mu.Lock()
	m.gateCalls++
	m.mu.Unlock()
	if m.gate != nil {
		return m.gate(ctx, transaction)
	}
	return nil
}

func (m *noticeMemory) Send(ctx context.Context, message mailer.Message) error {
	m.mu.Lock()
	if m.active != 0 {
		m.mu.Unlock()
		return errors.New("fixture observed transaction during Send")
	}
	m.sends++
	m.mu.Unlock()
	if _, ok := ctx.Deadline(); !ok {
		return mailer.ErrDeadlineRequired
	}
	if m.mail != nil {
		return m.mail(ctx, message)
	}
	return nil
}

func (m *noticeMemory) NewID() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ids++
	return fmt.Sprintf("40000000-0000-0000-0000-%012d", m.ids), nil
}

func (m *noticeMemory) Append(ctx context.Context, tx pgx.Tx, event audit.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if m.auditErr != nil {
		return m.auditErr
	}
	transaction := tx.(*noticeMemoryTx)
	if m.onAppend != nil {
		if err := m.onAppend(ctx, transaction, event); err != nil {
			return err
		}
	}
	// Match audit.Store's JSON storage/read boundary rather than returning int64 maps.
	var err error
	event.Metadata, err = noticeFixtureJSON(event.Metadata)
	if err != nil {
		return err
	}
	transaction.events = append(transaction.events, event)
	return nil
}

func (m *noticeMemory) Get(ctx context.Context, q audit.QueryRower, id string) (audit.Event, error) {
	if err := ctx.Err(); err != nil {
		return audit.Event{}, err
	}
	if m.readErr != nil {
		return audit.Event{}, m.readErr
	}
	for _, event := range q.(*noticeMemoryTx).events {
		if event.ID == id {
			var err error
			event.Metadata, err = noticeFixtureJSON(event.Metadata)
			if err != nil {
				return audit.Event{}, err
			}
			if m.onAuditRead != nil {
				m.onAuditRead(&event)
			}
			return event, nil
		}
	}
	return audit.Event{}, pgx.ErrNoRows
}

func noticeFixtureJSON(metadata map[string]any) (map[string]any, error) {
	data, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	var decoded map[string]any
	err = json.Unmarshal(data, &decoded)
	return decoded, err
}

func noticeLateResultStory(t *testing.T) (*noticeMemory, *TransferNoticeDispatcher, chan struct{}, <-chan error) {
	t.Helper()
	m, old := newNoticeMemory(t)
	old.config.BatchLimit = 1
	entered, release := make(chan struct{}), make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})
	m.mail = func(context.Context, mailer.Message) error {
		m.mu.Lock()
		first := m.sends == 1
		m.mu.Unlock()
		if first {
			close(entered)
			<-release
		}
		return nil
	}
	returned := make(chan error, 1)
	go func() { returned <- old.Reconcile(context.Background()) }()
	<-entered
	m.mu.Lock()
	m.now = m.now.Add(old.config.LeaseDuration)
	m.mu.Unlock()
	deps := m.dependencies()
	deps.WorkerID = noticeTestArticle
	current, err := NewTransferNoticeDispatcher(deps, noticeTestConfig())
	if err != nil {
		t.Fatal("successor construction failed")
	}
	t.Cleanup(func() { current.Stop(); _ = current.Wait(context.Background()) })
	if err := current.Reconcile(context.Background()); err != nil || m.rows[0].Status != RecipientAccepted {
		t.Fatal("successor did not finish")
	}
	return m, old, release, returned
}

func TestNoticeDispatcherR02F001_LateRealResult_IndependentRiskAudit(t *testing.T) {
	m, _, release, returned := noticeLateResultStory(t)
	before, _ := json.Marshal(m.rows)
	priorEvents := len(m.events)
	priorBegins := m.begins
	m.onAppend = func(_ context.Context, tx *noticeMemoryTx, event audit.Event) error {
		if tx.number != priorBegins+2 || m.active != 1 {
			t.Error("risk audit retained the result transaction")
		}
		if event.Action != "article.transfer_notice.duplicate_risk" || event.Outcome != "risk" {
			t.Error("late result wrote non-risk event")
		}
		return nil
	}
	close(release)
	if err := <-returned; err != nil {
		t.Fatal("late result should converge without recipient changes")
	}
	after, _ := json.Marshal(m.rows)
	if string(before) != string(after) || m.sends != 2 || m.active != 0 {
		t.Fatal("late result changed recipient or resent")
	}
	if len(m.events) != priorEvents+1 {
		t.Fatal("real late result has no independent durable risk audit")
	}
}

func TestNoticeDispatcherR02F002_JSONRoundTripBigints_AlreadyApplied(t *testing.T) {
	tests := []struct{ attempt, maximum int64 }{{1, 3}, {1, 9007199254740993}, {9007199254740993, math.MaxInt64}, {math.MaxInt64, math.MaxInt64}}
	for i, tt := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			m, d := newNoticeMemory(t)
			m.rows[0].AttemptCount = tt.attempt - 1
			m.rows[0].MaxAttempts = tt.maximum
			m.commitFault[4] = true
			if err := d.Reconcile(context.Background()); err != nil {
				t.Fatal("durable result failed")
			}
			if m.rows[0].Status != RecipientAccepted || m.sends != 1 || len(m.events) != 1 {
				t.Fatal("unknown commit resent or duplicated audit")
			}
			if len(m.logs) != 0 {
				t.Fatal("JSON-roundtripped exact binding was misreported unknown")
			}
			if m.events[0].Metadata["attempt_count"] != strconv.FormatInt(tt.attempt, 10) || m.events[0].Metadata["max_attempts"] != strconv.FormatInt(tt.maximum, 10) {
				t.Fatal("audit bigint representation is not exact decimal text")
			}
		})
	}
}

func TestNoticeDispatcherR02F001_ForgedOrUnsentHandle_NoRiskAudit(t *testing.T) {
	m, d := newNoticeMemory(t)
	ctx := noticeTestContext(t)
	c, found, err := d.claimDue(ctx)
	if err != nil || !found {
		t.Fatal("fixture claim failed")
	}
	m.now = c.LockedUntil
	tests := []struct {
		name      string
		change    func(*noticeClaim)
		attempted bool
	}{
		{"wrong_pk", func(c *noticeClaim) { c.NoticeID = noticeTestArticle }, true},
		{"wrong_kind", func(c *noticeClaim) { c.Kind = RecipientFormerDeletedAuthor }, true},
		{"wrong_attempt", func(c *noticeClaim) { c.AttemptCount++ }, true},
		{"no_issuance", func(c *noticeClaim) { c.issued = nil }, true},
		{"unsent", func(*noticeClaim) {}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidate := c
			tt.change(&candidate)
			if applied, err := d.saveResult(ctx, candidate, noticeResult{disposition: noticeAccepted, attempted: tt.attempted}); err != nil || applied {
				t.Fatal("invalid result applied")
			}
			if len(m.events) != 0 || m.sends != 0 || m.rows[0].Status != RecipientProcessing {
				t.Fatal("fabricated result recorded as real risk")
			}
		})
	}
}

func TestNoticeDispatcherR02F001_IndependentAuditFault_LowSensitiveFallback(t *testing.T) {
	for _, fault := range []string{"begin", "append", "ack_rollback", "ack_durable"} {
		t.Run(fault, func(t *testing.T) {
			m, _, release, returned := noticeLateResultStory(t)
			before, _ := json.Marshal(m.rows)
			priorEvents := len(m.events)
			independent := m.begins + 2
			switch fault {
			case "begin":
				m.onBegin = func(_ context.Context, number int) error {
					if number == independent {
						if m.active != 0 {
							t.Error("result transaction not released")
						}
						return errors.New("sensitive-canary")
					}
					return nil
				}
			case "append":
				m.auditErr = errors.New("sensitive-canary")
			case "ack_rollback":
				m.commitFault[independent] = false
			case "ack_durable":
				m.commitFault[independent] = true
			}
			close(release)
			if err := <-returned; err != nil {
				t.Fatal("recoverable audit fault should end round")
			}
			after, _ := json.Marshal(m.rows)
			if string(before) != string(after) || m.sends != 2 || m.active != 0 {
				t.Fatal("audit fallback changed state, resent, or leaked transaction")
			}
			want := priorEvents
			if fault == "ack_durable" {
				want++
			}
			if len(m.events) != want {
				t.Fatal("audit fault durability incorrect")
			}
			logs := strings.Join(m.logs, "")
			if !strings.Contains(logs, "late_result_risk_unpersisted") || strings.Contains(logs, "sensitive-canary") {
				t.Fatal("audit failure was hidden or leaked cause")
			}
		})
	}
}

func TestNoticeDispatcherR02F001_SharedShutdownDeadline_BoundsIndependentAudit(t *testing.T) {
	m, old, release, returned := noticeLateResultStory(t)
	priorEvents := len(m.events)
	shared, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	deadline, _ := shared.Deadline()
	entered := make(chan struct{})
	m.onAppend = func(ctx context.Context, _ *noticeMemoryTx, _ audit.Event) error {
		got, ok := ctx.Deadline()
		if !ok || got.After(deadline) {
			t.Error("independent audit extended shared deadline")
		}
		close(entered)
		<-ctx.Done()
		return ctx.Err()
	}
	old.BeginShutdown(shared)
	close(release)
	<-entered
	cancel()
	if err := <-returned; err == nil || !errors.Is(err, context.Canceled) {
		t.Fatal("shared audit cancellation not retained")
	}
	if m.rows[0].Status != RecipientAccepted || len(m.events) != priorEvents || m.sends != 2 || m.active != 0 {
		t.Fatal("shutdown audit did not roll back or join")
	}
}

func TestNoticeDispatcherR02F002_RealBindingMismatch_StillUnknown(t *testing.T) {
	tests := []struct {
		name   string
		change func(*audit.Event)
	}{
		{"target", func(e *audit.Event) { e.TargetID = noticeTestArticle }},
		{"trace", func(e *audit.Event) { e.TraceID = "01BX5ZZKBKACTAV9WEVGEMMVRZ" }},
		{"owner", func(e *audit.Event) { e.Metadata["claimed_by"] = "wrong-hash" }},
		{"attempt", func(e *audit.Event) { e.Metadata["attempt_count"] = "2" }},
		{"max", func(e *audit.Event) { e.Metadata["max_attempts"] = "9007199254740992" }},
		{"lease", func(e *audit.Event) { e.Metadata["locked_until"] = "wrong-lease" }},
		{"kind", func(e *audit.Event) { e.Metadata["recipient_kind"] = string(RecipientFormerDeletedAuthor) }},
		{"numeric_type", func(e *audit.Event) { e.Metadata["attempt_count"] = float64(1) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, d := newNoticeMemory(t)
			m.rows[0].MaxAttempts = 9007199254740993
			m.commitFault[4] = true
			m.onAuditRead = tt.change
			if err := d.Reconcile(context.Background()); err != nil {
				t.Fatal("recoverable unknown did not converge")
			}
			if m.rows[0].Status != RecipientAccepted || m.sends != 1 || len(m.events) != 1 {
				t.Fatal("mismatch resent or duplicated audit")
			}
			if !strings.Contains(strings.Join(m.logs, ""), "result_commit_unknown") {
				t.Fatal("wrong audit binding accepted")
			}
		})
	}
}

func (m *noticeMemory) Log(_ context.Context, _ slog.Level, message string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, fmt.Sprint(append([]any{message}, args...)...))
}

func noticeTestContext(t *testing.T) context.Context {
	t.Helper()
	ctx, err := platform.ContextWithTraceID(context.Background(), noticeTestTrace)
	if err != nil {
		t.Fatal("invalid test trace")
	}
	return ctx
}

func TestNoticeDispatcherFlow_Accepted_AtomicAuditAndReleasedTx(t *testing.T) {
	m, d := newNoticeMemory(t)
	m.mail = func(ctx context.Context, message mailer.Message) error {
		if m.rows[0].Status != RecipientProcessing || m.rows[0].AttemptCount != 1 {
			t.Error("Send preceded durable claim")
		}
		if m.gateCalls != 2 {
			t.Error("Send preceded both Gate checks")
		}
		if message.Purpose != mailer.PurposeOrphanArticleTransferNotice {
			t.Error("wrong purpose")
		}
		return nil
	}
	if err := d.Reconcile(context.Background()); err != nil {
		t.Fatal("reconcile failed")
	}
	r := m.rows[0]
	if r.Status != RecipientAccepted || r.AttemptCount != 1 || r.AcceptedAt == nil || r.ClaimedBy != nil || r.LockedUntil != nil || r.FailureBucket != nil || m.sends != 1 || m.active != 0 {
		t.Fatal("accepted state or resource contract failed")
	}
	if len(m.events) != 1 || m.events[0].Action != "article.transfer_notice.accepted" || !m.events[0].OccurredAt.Equal(*r.AcceptedAt) {
		t.Fatal("accepted audit not atomic or bound to DB time")
	}
}

func TestNoticeDispatcherGate_LastAttemptDeferred_ExhaustsWhileClosed(t *testing.T) {
	m, d := newNoticeMemory(t)
	m.rows[0].MaxAttempts = 1
	m.gate = func(_ context.Context, _ *noticeMemoryTx) error {
		if m.gateCalls >= 2 {
			return errors.New("synthetic gate closed")
		}
		return nil
	}
	if err := d.Reconcile(context.Background()); err != nil {
		t.Fatal("deferred round failed")
	}
	if m.rows[0].Status != RecipientRetryWait || m.rows[0].AttemptCount != 1 || m.sends != 0 || len(m.events) != 1 || m.events[0].Action != "article.transfer_notice.retry_scheduled" {
		t.Fatal("last attempt was not deferred")
	}
	if m.begins != 3 {
		t.Fatal("non-aborted Gate deferral did not share preflight transaction")
	}
	m.now = *m.rows[0].NextAttemptAt
	if err := d.Reconcile(context.Background()); err != nil {
		t.Fatal("maintenance failed")
	}
	if m.rows[0].Status != RecipientFailedTerminal || m.sends != 0 || len(m.events) != 2 || m.events[1].Metadata["prior_reason_bucket"] != "processing_gate_closed" {
		t.Fatal("closed Gate exhausted maintenance failed")
	}
}

func TestNoticeDispatcherGate_QueryAbort_RollsBackBeforeDeferredSave(t *testing.T) {
	m, d := newNoticeMemory(t)
	m.gate = func(_ context.Context, tx *noticeMemoryTx) error {
		if m.gateCalls == 2 {
			tx.aborted = true
			return &pgconn.PgError{Code: "XX000", Message: "sensitive-canary"}
		}
		return nil
	}
	if err := d.Reconcile(context.Background()); err != nil {
		t.Fatal("aborted Gate failed to converge")
	}
	if m.rows[0].Status != RecipientRetryWait || m.sends != 0 || m.active != 0 {
		t.Fatal("aborted tx used or not released")
	}
	if m.begins != 4 {
		t.Fatal("deferred save did not open a new transaction")
	}
}

func TestNoticeDispatcherFailures_LastAttempt_PreservesBucket(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		bucket string
		risk   bool
	}{
		{"accepted", nil, "", false}, {"invalid", mailer.ErrInvalidMessage, "invalid_message", false},
		{"purpose", mailer.ErrUnsupportedPurpose, "unsupported_purpose", false},
		{"tls", mailer.ErrTLSRequired, "smtp_security_or_config", false},
		{"unknown", errors.New("sensitive-canary"), "smtp_outcome_unknown", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, d := newNoticeMemory(t)
			m.rows[0].MaxAttempts = 1
			m.mail = func(context.Context, mailer.Message) error { return tt.err }
			if err := d.Reconcile(context.Background()); err != nil {
				t.Fatal("round failed")
			}
			r := m.rows[0]
			want := RecipientFailedTerminal
			if tt.err == nil {
				want = RecipientAccepted
			}
			if r.Status != want || r.AttemptCount != 1 || r.NextAttemptAt != nil {
				t.Fatal("last attempt state failed")
			}
			if tt.bucket != "" && (r.FailureBucket == nil || *r.FailureBucket != tt.bucket) {
				t.Fatal("failure bucket changed")
			}
			if tt.risk && (len(m.events) != 2 || m.events[1].Action != "article.transfer_notice.duplicate_risk") {
				t.Fatal("unknown missing durable risk")
			}
		})
	}
}

func TestNoticeDispatcherClaim_CommitUnknown_NeverSends(t *testing.T) {
	for _, durable := range []bool{false, true} {
		t.Run(fmt.Sprint(durable), func(t *testing.T) {
			m, d := newNoticeMemory(t)
			m.commitFault[2] = durable
			if err := d.Reconcile(context.Background()); err != nil {
				t.Fatal("recoverable unknown should end round")
			}
			want := RecipientPending
			if durable {
				want = RecipientProcessing
			}
			if m.rows[0].Status != want || m.sends != 0 || m.gateCalls != 1 || m.active != 0 {
				t.Fatal("unknown claim granted send rights")
			}
		})
	}
}

func TestNoticeDispatcherResult_CommitUnknown_VerifiesBindingWithoutResend(t *testing.T) {
	for _, durable := range []bool{false, true} {
		t.Run(fmt.Sprint(durable), func(t *testing.T) {
			m, d := newNoticeMemory(t)
			m.commitFault[4] = durable
			if err := d.Reconcile(context.Background()); err != nil {
				t.Fatal("recoverable result unknown should end round")
			}
			want := RecipientProcessing
			if durable {
				want = RecipientAccepted
			}
			if m.rows[0].Status != want || m.sends != 1 || m.active != 0 {
				t.Fatal("result unknown resent or corrupted state")
			}
			if durable && len(m.events) != 1 {
				t.Fatal("unknown appended duplicate audit")
			}
			if !durable && len(m.logs) == 0 {
				t.Fatal("unconfirmed result reported success")
			}
		})
	}
}

func TestNoticeDispatcherAuditFailure_AfterAccepted_RollsBackAndStopsRound(t *testing.T) {
	m, d := newNoticeMemory(t)
	m.mail = func(context.Context, mailer.Message) error { m.auditErr = errors.New("sensitive-canary"); return nil }
	if err := d.Reconcile(context.Background()); err != nil {
		t.Fatal("recoverable persistence failure should end round")
	}
	if m.rows[0].Status != RecipientProcessing || len(m.events) != 0 || m.sends != 1 || m.active != 0 {
		t.Fatal("audit failure not atomic")
	}
	if strings.Contains(strings.Join(m.logs, ""), "sensitive-canary") {
		t.Fatal("cause leaked into logger")
	}
}

func TestTransferNoticeTemplate_KindsAndInvalidReferences_MinimalDisclosure(t *testing.T) {
	for _, kind := range []RecipientKind{RecipientNewAuthor, RecipientFormerDeletedAuthor} {
		t.Run(string(kind), func(t *testing.T) {
			c := noticeClaim{NoticeID: noticeTestID, ArticleRef: noticeTestArticle, Kind: kind, EmailSnapshot: "synthetic@example.invalid"}
			message, err := noticeMessage(c)
			if err != nil {
				t.Fatal("valid snapshot rejected")
			}
			if strings.Count(message.Body, "\n") != 3 || message.Subject != "Han article transfer notice" || strings.Contains(message.Body, "@") || strings.Contains(message.Body, "http") || strings.Contains(message.Body, "<") {
				t.Fatal("template exceeds minimal disclosure")
			}
			c.ArticleRef = "sensitive-canary\n"
			if _, err := noticeMessage(c); !errors.Is(err, mailer.ErrInvalidMessage) {
				t.Fatal("unsafe reference accepted")
			}
		})
	}
}

func TestTransferNoticeBackoff_HugeAttempts_SaturatesWithoutOverflow(t *testing.T) {
	for _, attempt := range []int64{1, 2, 3, 4, 63, 64, math.MaxInt64} {
		got := noticeBackoff(time.Second, 8*time.Second, attempt)
		want := 8 * time.Second
		if attempt < 4 {
			want = time.Second << uint(attempt-1)
		}
		if got != want {
			t.Fatal("backoff did not saturate")
		}
	}
	if _, err := noticeAdd(time.Date(9999, 12, 31, 23, 59, 59, 999999000, time.UTC), time.Microsecond); err == nil {
		t.Fatal("time overflow accepted")
	}
}

func TestTransferNoticeClassifier_UnverifiedChains_ConservativeAndSafe(t *testing.T) {
	tests := []struct {
		err    error
		bucket string
		risk   bool
	}{
		{mailer.ErrMessageRejected, "smtp_outcome_unknown", true},
		{&mailer.SMTPMessageRejection{Stage: mailer.SMTPMessageStageRCPTTo, Code: 550}, "smtp_outcome_unknown", true},
		{&textproto.Error{Code: 535, Msg: "sensitive-canary"}, "smtp_security_or_config", false},
		{context.Canceled, "smtp_outcome_unknown", true},
		{fmt.Errorf("sensitive-canary: %w", mailer.ErrInvalidSMTPConfig), "smtp_security_or_config", false},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			r := classifyNoticeSMTP(tt.err)
			if r.bucket != tt.bucket || r.risk != tt.risk {
				t.Fatal("unsafe classification")
			}
			if strings.Contains(fmt.Sprint(r), "sensitive-canary") {
				t.Fatal("classification retained raw cause")
			}
		})
	}
}

func TestTransferNoticeAudit_BindingMismatch_NotAlreadyApplied(t *testing.T) {
	m, d := newNoticeMemory(t)
	m.commitFault[4] = true
	m.onAuditRead = func(event *audit.Event) { event.TargetID = noticeTestArticle }
	if err := d.Reconcile(context.Background()); err != nil {
		t.Fatal("unknown should end round")
	}
	if m.sends != 1 || len(m.logs) == 0 {
		t.Fatal("mismatched audit accepted or resent")
	}
	encoded, _ := json.Marshal(m.events)
	if strings.Contains(string(encoded), m.rows[0].EmailSnapshot) || strings.Contains(string(encoded), noticeTestWorker) {
		t.Fatal("audit leaked address or raw owner")
	}
}

func TestNoticeDispatcherConfig_InvalidBudgets_Rejected(t *testing.T) {
	tests := []struct {
		name   string
		change func(*NoticeDispatcherConfig)
	}{
		{"zero_poll", func(c *NoticeDispatcherConfig) { c.PollInterval = 0 }},
		{"submicrosecond", func(c *NoticeDispatcherConfig) { c.DBOperationBudget = time.Nanosecond }},
		{"zero_batch", func(c *NoticeDispatcherConfig) { c.BatchLimit = 0 }},
		{"batch_overflow", func(c *NoticeDispatcherConfig) { c.BatchLimit = math.MaxInt32 + 1 }},
		{"lease_equal", func(c *NoticeDispatcherConfig) { c.LeaseDuration = 6 * time.Second }},
		{"budget_overflow", func(c *NoticeDispatcherConfig) { c.DBOperationBudget = time.Duration(math.MaxInt64 / 1000 * 1000) }},
		{"backoff_order", func(c *NoticeDispatcherConfig) { c.BackoffInitial = c.BackoffMax + time.Second }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := noticeTestConfig()
			tt.change(&c)
			if c.Validate() == nil {
				t.Fatal("invalid configuration accepted")
			}
		})
	}
}

func TestNoticeDispatcherConstructor_MissingDependencies_Rejected(t *testing.T) {
	if _, err := NewTransferNoticeDispatcher(NoticeDispatcherDependencies{}, noticeTestConfig()); err == nil {
		t.Fatal("missing dependencies accepted")
	}
}

func TestNoticeDispatcherStop_BeforeStart_DoneClosed(t *testing.T) {
	d := &TransferNoticeDispatcher{done: make(chan struct{})}
	d.Stop()
	select {
	case <-d.Done():
	default:
		t.Fatal("idle Stop did not close Done")
	}
	if err := d.Wait(context.Background()); err != nil {
		t.Fatal("idle Wait failed")
	}
}

func TestNoticeDispatcher_SharedDeadlineNoticeFencedSave(t *testing.T) {
	m, d := newNoticeMemory(t)
	var heldMu sync.Mutex
	held, acquired, released := 0, 0, 0
	d.deps.AcquireWork = func(ctx context.Context) (context.Context, func(), error) {
		heldMu.Lock()
		held++
		acquired++
		heldMu.Unlock()
		return ctx, func() { heldMu.Lock(); held--; released++; heldMu.Unlock() }, nil
	}
	mailEntered, saveEntered, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	m.mail = func(ctx context.Context, _ mailer.Message) error { close(mailEntered); <-ctx.Done(); return nil }
	m.onSave = func(ctx context.Context) {
		heldMu.Lock()
		if held != 1 {
			t.Error("fenced save lost claim work lease")
		}
		heldMu.Unlock()
		close(saveEntered)
		<-release
	}
	result := make(chan error, 1)
	go func() { result <- d.Reconcile(context.Background()) }()
	<-mailEntered
	shared, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	d.BeginShutdown(shared)
	<-saveEntered
	select {
	case <-d.Done():
		t.Error("Done before actual fenced save return")
	default:
	}
	heldMu.Lock()
	if held != 1 || released != 1 {
		t.Error("lease released before save joined")
	}
	heldMu.Unlock()
	close(release)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	if err := d.Wait(shared); err != nil {
		t.Fatal(err)
	}
	if held != 0 || acquired != 2 || released != 2 || m.sends != 1 || m.rows[0].Status != RecipientAccepted {
		t.Fatal("fenced save/lease ledger changed")
	}
}
