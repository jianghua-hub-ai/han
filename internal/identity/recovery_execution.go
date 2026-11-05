package identity

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"han/internal/audit"
	"han/internal/idempotency"
	"han/internal/platform"
	"han/internal/recovery"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
)

const (
	recoveryBindAction     = "identity.recovery.bind"
	recoveryFinalizeAction = "identity.recovery.finalize"
	recoveryTrustAction    = "identity.recovery.trust.prepare"
	recoveryReviewAction   = "identity.recovery.review"
	recoveryExitAction     = "recovery_maintenance_exit"
)

// RecoveryIdentityExecutionDependencies is construction-time DI for the later
// cmd root. Runtime is the actual instance; Store is the sole existing Store.
// FailedEvidence may be absent only for actions which do not consume that port.
type RecoveryIdentityExecutionDependencies struct {
	Store          *RecoveryRunStore
	Runtime        recovery.Runtime
	Journal        recovery.ContextRecoveryJournal
	Articles       ArticleAccountPort
	Audit          CanonicalDeleteAuditAppender
	Passwords      RegistrationPasswordHasher
	IDs            RegistrationIDGenerator
	Clock          RegistrationClock
	FailedEvidence RecoveryFailedEvidenceReader
}

// RecoveryIdentityMaintenance delegates the two already-designed identity
// consumer actions. HTTP/session/CSRF decoding remains the root's existing
// responsibility. Both calls recheck a REAL current raw phase permit internally;
// this interface creates no route, mint, retry, business authority or ready flag.
type RecoveryIdentityMaintenance interface {
	ResolveRestrictedAccountReview(context.Context, recovery.PhasePermit, recovery.RunBinding, string, string, string) error
	ResolveRecoveryIdempotency(context.Context, recovery.PhasePermit, recovery.RunBinding, []byte, []byte, int64, string, string) error
}

// NewRecoveryIdentityExecution returns the SAME implementation through its
// original five-method Runtime interface and its narrow maintenance consumer.
// Dependencies are copied privately once; there is no setter or I/O here.
func NewRecoveryIdentityExecution(d RecoveryIdentityExecutionDependencies) (recovery.IdentityRecoveryExecutor, RecoveryIdentityMaintenance, error) {
	e, err := newRecoveryIdentityExecution(recoveryExecutionDependencies{store: d.Store, runtime: d.Runtime, journal: d.Journal, articles: d.Articles, audit: d.Audit, passwords: d.Passwords, ids: d.IDs, clock: d.Clock, failedEvidence: d.FailedEvidence})
	if err != nil {
		return nil, nil, err
	}
	return e, e, nil
}

// RecoveryIdentityForwarder solves only the construction cycle: create this,
// construct the real Runtime with it, construct the executor with that Runtime,
// then Seal exactly once BEFORE Enter/Resume. Unsealed calls fail; replacement,
// nil and self-delegation fail. It implements only the original five methods.
type RecoveryIdentityForwarder = recoveryExecutionForwarder

func NewRecoveryIdentityForwarder() *RecoveryIdentityForwarder { return &recoveryExecutionForwarder{} }
func (f *recoveryExecutionForwarder) Seal(e recovery.IdentityRecoveryExecutor) error {
	if f == nil {
		return recovery.ErrAdmissionBinding
	}
	return f.seal(e)
}
func (e *recoveryIdentityExecution) ResolveRecoveryIdempotency(c context.Context, p recovery.PhasePermit, b recovery.RunBinding, s, k []byte, g int64, d, r string) error {
	return e.resolveIdempotency(c, p, b, s, k, g, d, r)
}
func (e *recoveryIdentityExecution) ResolveRestrictedAccountReview(c context.Context, p recovery.PhasePermit, b recovery.RunBinding, u, d, r string) error {
	return e.resolveRestricted(c, p, b, u, d, r)
}

// This implementation shares the original Store; no additional pool or durable
// recovery authority is created. Constructors do no I/O.
type recoveryExecutionDependencies struct {
	store          *RecoveryRunStore
	runtime        recovery.Runtime
	journal        recovery.ContextRecoveryJournal
	articles       ArticleAccountPort
	audit          CanonicalDeleteAuditAppender
	passwords      RegistrationPasswordHasher
	ids            RegistrationIDGenerator
	clock          RegistrationClock
	failedEvidence RecoveryFailedEvidenceReader // optional only for unrelated actions; sealed with dependencies
}

type recoveryIdentityExecution struct {
	d         recoveryExecutionDependencies
	canonical *CanonicalDeleteCoordinator
	// A private transaction-boundary seam for actual Commit/ACK fault tests.
	// It is initialized to pgx.Tx.Commit and is not an alternate write path.
	commit func(context.Context, pgx.Tx) error
}

var _ recovery.IdentityRecoveryExecutor = (*recoveryIdentityExecution)(nil)

func newRecoveryIdentityExecution(d recoveryExecutionDependencies) (*recoveryIdentityExecution, error) {
	for _, v := range []any{d.store, d.runtime, d.journal, d.articles, d.audit, d.passwords, d.ids, d.clock} {
		if nilIdentityDependency(v) {
			return nil, recovery.ErrAdmissionBinding
		}
	}
	if d.store.pool == nil {
		return nil, recovery.ErrAdmissionBinding
	}
	e := &recoveryIdentityExecution{d: d, commit: func(ctx context.Context, tx pgx.Tx) error { return tx.Commit(ctx) }}
	c, err := NewCanonicalDeleteService(CanonicalDeleteDependencies{
		Transactions: d.store.pool, Repository: PostgresCanonicalDeleteRepository{},
		Articles: d.articles, Audit: d.audit, IDs: d.ids, Clock: d.clock,
		RecoveryRuns: d.store, RecoveryGate: recoveryExecutionGate{e: e},
	})
	if err != nil {
		return nil, err
	}
	e.canonical = c
	return e, nil
}

// The forwarder is construction-only. All methods reject before sealing; the
// original interface alone is forwarded and the delegate cannot be replaced.
type recoveryExecutionForwarder struct {
	mu     sync.RWMutex
	target recovery.IdentityRecoveryExecutor
}

func (f *recoveryExecutionForwarder) seal(t recovery.IdentityRecoveryExecutor) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.target != nil || nilIdentityDependency(t) || t == f {
		return recovery.ErrAdmissionBinding
	}
	f.target = t
	return nil
}
func (f *recoveryExecutionForwarder) get() (recovery.IdentityRecoveryExecutor, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.target == nil {
		return nil, recovery.ErrAdmissionBinding
	}
	return f.target, nil
}
func (f *recoveryExecutionForwarder) BindRestoredRun(c context.Context, p recovery.PhasePermit, b recovery.RunBinding, x recovery.RestoreReceipt) (recovery.RecoveryRun, error) {
	t, e := f.get()
	if e != nil {
		return recovery.RecoveryRun{}, e
	}
	return t.BindRestoredRun(c, p, b, x)
}
func (f *recoveryExecutionForwarder) FinalizeRestoredIdentity(c context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
	t, e := f.get()
	if e != nil {
		return e
	}
	return t.FinalizeRestoredIdentity(c, p, b)
}
func (f *recoveryExecutionForwarder) PrepareRecoveryTrust(c context.Context, p recovery.PhasePermit, b recovery.RunBinding, s []byte) error {
	t, e := f.get()
	if e != nil {
		return e
	}
	return t.PrepareRecoveryTrust(c, p, b, s)
}
func (f *recoveryExecutionForwarder) CompleteIdentityReview(c context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
	t, e := f.get()
	if e != nil {
		return e
	}
	return t.CompleteIdentityReview(c, p, b)
}
func (f *recoveryExecutionForwarder) RevokeRecoverySessions(c context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
	t, e := f.get()
	if e != nil {
		return e
	}
	return t.RevokeRecoverySessions(c, p, b)
}

type recoveryExecutionGate struct{ e *recoveryIdentityExecution }

func (g recoveryExecutionGate) CheckRecoveryOpenGate(ctx context.Context, tx pgx.Tx) error {
	if tx == nil {
		return recovery.ErrAdmissionBinding
	}
	b, err := g.e.d.runtime.ReadPermitBinding(ctx)
	if err != nil {
		return err
	}
	if b.Purpose != recovery.PhaseRecoveryIdentity {
		return recovery.ErrAdmissionBinding
	}
	_, _, err = g.e.lockedRun(ctx, tx, recovery.RunBinding{RunID: b.RunID, BackupRef: b.BackupRef}, b, recovery.RecoveryOutcomeRunning)
	return err
}

func (e *recoveryIdentityExecution) phase(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding, purpose string) (context.Context, recovery.PermitBinding, error) {
	if e == nil || ctx == nil || b.RunID == "" || b.BackupRef == "" {
		return nil, recovery.PermitBinding{}, recovery.ErrAdmissionBinding
	}
	if _, ok := ctx.Deadline(); !ok {
		return nil, recovery.PermitBinding{}, recovery.ErrAdmissionBinding
	}
	if err := e.d.runtime.CheckPhase(ctx, p, purpose); err != nil {
		return nil, recovery.PermitBinding{}, err
	}
	pc, err := e.d.runtime.PhaseContext(ctx, p)
	if err != nil {
		return nil, recovery.PermitBinding{}, err
	}
	x, err := e.d.runtime.ReadPermitBinding(pc)
	if err != nil {
		return nil, x, err
	}
	if x.Purpose != purpose || x.RunID != b.RunID || x.BackupRef != b.BackupRef || x.InstallationID == "" || x.RuntimeID == "" || x.Generation <= 0 {
		return nil, x, recovery.ErrAdmissionBinding
	}
	return pc, x, nil
}

func (e *recoveryIdentityExecution) guard(ctx context.Context, p recovery.PhasePermit, purpose string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return e.d.runtime.CheckPhase(ctx, p, purpose)
}

// write owns both physical connections. Unknown Commit is never retried here.
// The independent verifier checks the complete expected row image, including
// all related audit content/residual rows and the original Store binding/proof.
func (e *recoveryIdentityExecution) write(ctx context.Context, p recovery.PhasePermit, purpose string, scope recoveryExecutionScope, work func(pgx.Tx) error) error {
	return e.writeVerified(ctx, p, purpose, scope, work, nil)
}
func (e *recoveryIdentityExecution) writeVerified(ctx context.Context, p recovery.PhasePermit, purpose string, scope recoveryExecutionScope, work func(pgx.Tx) error, verifyOutside func() error) (result error) {
	if err := e.guard(ctx, p, purpose); err != nil {
		return err
	}
	conn, err := e.d.store.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, finishAdmissionRead(ctx, tx)) }()
	if err = work(tx); err != nil {
		return err
	}
	expected, err := recoveryExecutionImage(ctx, tx, scope)
	if err != nil {
		return err
	}
	if err = e.guard(ctx, p, purpose); err != nil {
		return err
	}
	ack := e.commit(ctx, tx)
	// Commit returning an error does not prove the transaction ended. Join the
	// writer transaction before any source or independent database observation.
	ended := finishAdmissionRead(ctx, tx)
	if ended != nil {
		return errors.Join(recovery.ErrRecoveryWriteUnknown, ack, ended)
	}
	if verifyOutside != nil {
		if err = verifyOutside(); err != nil {
			return errors.Join(recovery.ErrRecoveryWriteUnknown, ack, err)
		}
	}
	if ack == nil && verifyOutside == nil {
		return e.guard(ctx, p, purpose)
	}
	if err = e.guard(ctx, p, purpose); err != nil {
		return errors.Join(recovery.ErrRecoveryWriteUnknown, ack, err)
	}
	verify, err := e.d.store.pool.Acquire(ctx)
	if err != nil {
		return errors.Join(recovery.ErrRecoveryWriteUnknown, ack, err)
	}
	defer verify.Release()
	if verify.Conn().PgConn().PID() == conn.Conn().PgConn().PID() {
		return errors.Join(recovery.ErrRecoveryWriteUnknown, ack, recovery.ErrAdmissionBinding)
	}
	vt, err := verify.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return errors.Join(recovery.ErrRecoveryWriteUnknown, ack, err)
	}
	defer func() {
		if closeErr := finishAdmissionRead(ctx, vt); closeErr != nil {
			result = errors.Join(result, recovery.ErrRecoveryWriteUnknown, ack, closeErr)
		}
	}()
	actual, err := recoveryExecutionImage(ctx, vt, scope)
	if err == nil && !bytes.Equal(actual, expected) {
		err = recovery.ErrAdmissionBinding
	}
	if err == nil {
		err = vt.Commit(ctx)
	}
	if err == nil {
		err = e.guard(ctx, p, purpose)
	}
	if err != nil {
		return errors.Join(recovery.ErrRecoveryWriteUnknown, ack, err)
	}
	return nil
}

// Scoped observation implementation follows below.

func (e *recoveryIdentityExecution) begun(ctx context.Context, b recovery.RunBinding) (recovery.JournalEntry, error) {
	s, err := e.d.journal.ReadVerifiedSnapshot(ctx, recovery.RecoveryNamespaceV1)
	if err != nil {
		return recovery.JournalEntry{}, err
	}
	if s.Pending != nil || s.Namespace != recovery.RecoveryNamespaceV1 || s.Revision == "" {
		return recovery.JournalEntry{}, recovery.ErrAdmissionBinding
	}
	var found recovery.JournalEntry
	for _, x := range s.Entries {
		if x.RunID != b.RunID {
			continue
		}
		if x.BackupRef != b.BackupRef {
			return found, recovery.ErrAdmissionBinding
		}
		if x.Stage == recovery.StageBegun {
			if found.RunID != "" {
				return found, recovery.ErrAdmissionBinding
			}
			found = x
		}
	}
	if found.RunID == "" {
		return found, recovery.ErrAdmissionIncomplete
	}
	return found, nil
}

func recoveryExecutionMatchBegin(r recovery.AdmissionRun, j recovery.JournalEntry) error {
	if r.RunID != j.RunID || r.BackupRef != j.BackupRef || r.JournalKeyID != j.KeyID || r.JournalKeyVersion != j.KeyVersion || r.JournalSecretFingerprint != j.SecretFingerprint || !r.StartedAt.Equal(j.RecordedAt) {
		return recovery.ErrAdmissionBinding
	}
	return nil
}

func (e *recoveryIdentityExecution) lockedRun(ctx context.Context, tx pgx.Tx, b recovery.RunBinding, p recovery.PermitBinding, out recovery.RecoveryOutcome) (recovery.AdmissionRun, recovery.AdmissionSnapshot, error) {
	var id string
	if err := tx.QueryRow(ctx, `SELECT run_id FROM recovery_runs WHERE run_id=$1 FOR UPDATE`, b.RunID).Scan(&id); err != nil {
		return recovery.AdmissionRun{}, recovery.AdmissionSnapshot{}, err
	}
	s, err := readAdmissionTransaction(ctx, tx)
	if err != nil {
		return recovery.AdmissionRun{}, s, err
	}
	if s.Installation.InstallationID != p.InstallationID {
		return recovery.AdmissionRun{}, s, recovery.ErrAdmissionBinding
	}
	for _, r := range s.Runs {
		if r.RunID == b.RunID {
			if r.BackupRef != b.BackupRef || r.Outcome != out || r.RestoreBinding == nil {
				return r, s, recovery.ErrAdmissionBinding
			}
			return r, s, nil
		}
	}
	return recovery.AdmissionRun{}, s, recovery.ErrRunNotFound
}

func (e *recoveryIdentityExecution) event(ctx context.Context, tx pgx.Tx, action, target string, b recovery.RunBinding, install string, metadata map[string]any) error {
	observed, err := e.d.runtime.ReadPermitBinding(ctx)
	if err != nil {
		return err
	}
	purpose := recovery.PhaseRecoveryIdentity
	if action == recoveryBindAction {
		purpose = recovery.PhaseRecoveryBind
	}
	if action == recoveryExitAction {
		purpose = recovery.PhaseRecoveryReplay
	}
	if observed.Purpose != purpose || observed.RunID != b.RunID || observed.BackupRef != b.BackupRef || observed.InstallationID != install {
		return recovery.ErrAdmissionBinding
	}
	id, err := e.d.ids.NewID()
	if err != nil {
		return err
	}
	trace, ok := platform.TraceIDFromContext(ctx)
	if !ok {
		return recovery.ErrAdmissionBinding
	}
	system := "recovery_runtime"
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["run_id"] = b.RunID
	metadata["backup_ref"] = b.BackupRef
	metadata["installation_id"] = install
	event := audit.Event{ID: id, OccurredAt: e.d.clock.Now().UTC().Truncate(time.Microsecond), ActorKind: audit.ActorSystem, ActorSystem: &system, Action: action, TargetType: "recovery_run", TargetID: target, Outcome: "succeeded", TraceID: trace, Metadata: metadata}
	if action == recoveryReviewAction {
		actor, ok := metadata["actor_id"].(string)
		if !ok || actor == "" {
			return recovery.ErrAdmissionBinding
		}
		role := "admin"
		event.ActorKind = audit.ActorUser
		event.ActorSystem = nil
		event.ActorUserID = &actor
		event.ActorRole = &role
		event.TargetType = "user"
	}
	return e.d.audit.Append(ctx, tx, event)
}

func recoveryExecutionAudit(ctx context.Context, tx pgx.Tx, action, target string, b recovery.RunBinding, install string) (json.RawMessage, bool, error) {
	rows, err := tx.Query(ctx, `SELECT metadata::text FROM audit_events WHERE action=$1 AND target_id=$2 AND outcome='succeeded' ORDER BY id`, action, target)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	var found json.RawMessage
	for rows.Next() {
		var raw []byte
		if err = rows.Scan(&raw); err != nil {
			return nil, false, err
		}
		var m struct {
			RunID     string `json:"run_id"`
			BackupRef string `json:"backup_ref"`
			Install   string `json:"installation_id"`
		}
		if err = json.Unmarshal(raw, &m); err != nil {
			return nil, false, err
		}
		if m.RunID == b.RunID {
			if m.BackupRef != b.BackupRef || m.Install != install || found != nil {
				return nil, false, recovery.ErrAdmissionBinding
			}
			found = append(json.RawMessage{}, raw...)
		}
	}
	return found, found != nil, rows.Err()
}

func (e *recoveryIdentityExecution) BindRestoredRun(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding, x recovery.RestoreReceipt) (recovery.RecoveryRun, error) {
	pc, v, err := e.phase(ctx, p, b, recovery.PhaseRecoveryBind)
	if err != nil {
		return recovery.RecoveryRun{}, err
	}
	j, err := e.begun(pc, b)
	if err != nil {
		return recovery.RecoveryRun{}, err
	}
	if x.RunID != b.RunID || x.BackupRef != b.BackupRef || x.InstallationID != v.InstallationID || x.Version != "han-db-restore/v1" {
		return recovery.RecoveryRun{}, recovery.ErrAdmissionBinding
	}
	err = e.write(pc, p, recovery.PhaseRecoveryBind, recoveryExecutionScope{action: recoveryBindAction, binding: b, installation: v.InstallationID}, func(tx pgx.Tx) error {
		s, err := readAdmissionTransaction(pc, tx)
		if err != nil {
			return err
		}
		if x.DeploymentID != s.Installation.DeploymentID || x.TargetDigest != s.Installation.TargetDigest {
			return recovery.ErrAdmissionBinding
		}
		for _, r := range s.Runs {
			if r.RunID == b.RunID {
				if r.RestoreBinding == nil || *r.RestoreBinding != x || r.Outcome != recovery.RecoveryOutcomeRunning {
					return recovery.ErrAdmissionBinding
				}
				if err := recoveryExecutionMatchBegin(r, j); err != nil {
					return err
				}
				_, ok, err := recoveryExecutionAudit(pc, tx, recoveryBindAction, b.RunID, b, v.InstallationID)
				if err != nil {
					return err
				}
				if !ok {
					return recovery.ErrAdmissionIncomplete
				}
				return nil
			}
		}
		if err := e.guard(pc, p, recovery.PhaseRecoveryBind); err != nil {
			return err
		}
		_, err = tx.Exec(pc, insertRecoveryRunSQL, b.RunID, b.BackupRef, j.KeyID, j.KeyVersion, j.SecretFingerprint, recovery.StageBegun, j.RecordedAt, recovery.RecoveryOutcomeRunning)
		if err != nil {
			return err
		}
		if err = e.guard(pc, p, recovery.PhaseRecoveryBind); err != nil {
			return err
		}
		_, err = tx.Exec(pc, `INSERT INTO recovery_restore_bindings(run_id,backup_ref,installation_id,deployment_id,artifact_digest,target_digest,receipt_digest,source_approval_digest,restored_schema_digest,artifact_bytes,restored_at,receipt_version) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, b.RunID, b.BackupRef, x.InstallationID, x.DeploymentID, x.ArtifactDigest, x.TargetDigest, x.ReceiptDigest, x.SourceApprovalDigest, x.RestoredSchemaDigest, x.ArtifactBytes, x.RestoredAt, x.Version)
		if err != nil {
			return err
		}
		return e.event(pc, tx, recoveryBindAction, b.RunID, b, v.InstallationID, map[string]any{"receipt": x})
	})
	if err != nil {
		return recovery.RecoveryRun{}, err
	}
	return e.d.store.Load(pc, b.RunID)
}

type recoveryExecutionUser struct {
	ID                  string     `json:"id"`
	Role                string     `json:"role"`
	Status              string     `json:"status"`
	Initial             bool       `json:"is_initial_admin"`
	Required            bool       `json:"recovery_required"`
	Mode                string     `json:"credential_mode"`
	Auth                int64      `json:"auth_generation"`
	Digest              *string    `json:"permanent_password_digest"`
	PermanentGeneration *int64     `json:"permanent_password_generation"`
	SourceOperation     *string    `json:"permanent_password_source_operation_id"`
	SourceCredential    *string    `json:"permanent_password_source_credential_id"`
	Email               *string    `json:"email"`
	Phone               *string    `json:"phone"`
	Retained            *string    `json:"retained_email"`
	DeletedAt           *time.Time `json:"deleted_at"`
}
type recoveryExecutionSource struct {
	ID        string
	Role      string
	Status    string
	DeletedAt *time.Time
	Initial   bool
}
type recoveryExecutionFinalization struct {
	RunID       string                      `json:"run_id"`
	BackupRef   string                      `json:"backup_ref"`
	Install     string                      `json:"installation_id"`
	Sources     []recoveryExecutionSource   `json:"sources"`
	Cohort      []recoveryIdempotencyMember `json:"idempotency_cohort"`
	CohortCount string                      `json:"idempotency_cohort_count"`
	Accounts    int64                       `json:"accounts"`
	Permanent   int64                       `json:"permanent"`
	Sessions    int64                       `json:"sessions"`
	Temporary   int64                       `json:"temporary"`
	Idempotency int64                       `json:"idempotency"`
	Reviews     int64                       `json:"reviews"`
}

func recoveryExecutionUsers(ctx context.Context, tx pgx.Tx, ids []string, lock bool) ([]recoveryExecutionUser, error) {
	sql := `SELECT to_jsonb(u)::text FROM users u`
	args := []any{}
	if ids != nil {
		sql += ` WHERE id=ANY($1::uuid[])`
		args = append(args, ids)
	}
	sql += ` ORDER BY id`
	if lock {
		sql += ` FOR UPDATE`
	}
	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []recoveryExecutionUser{}
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var raw []byte
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		var u recoveryExecutionUser
		if err = json.Unmarshal(raw, &u); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func recoveryExecutionOne(tag int64, err error) error {
	if err != nil {
		return err
	}
	if tag != 1 {
		return recovery.ErrRecoveryRunConflict
	}
	return nil
}

func (e *recoveryIdentityExecution) FinalizeRestoredIdentity(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
	pc, v, err := e.phase(ctx, p, b, recovery.PhaseRecoveryIdentity)
	if err != nil {
		return err
	}
	j, err := e.begun(pc, b)
	if err != nil {
		return err
	}
	return e.write(pc, p, recovery.PhaseRecoveryIdentity, recoveryExecutionScope{action: recoveryFinalizeAction, binding: b, installation: v.InstallationID}, func(tx pgx.Tx) error {
		users, err := recoveryExecutionUsers(pc, tx, nil, true)
		if err != nil {
			return err
		}
		r, _, err := e.lockedRun(pc, tx, b, v, recovery.RecoveryOutcomeRunning)
		if err != nil {
			return err
		}
		if err = recoveryExecutionMatchBegin(r, j); err != nil {
			return err
		}
		if r.AuditEventID != nil {
			_, err = e.finalization(pc, tx, r, b, v.InstallationID)
			return err
		}
		return e.finalizeFirst(pc, tx, p, b, v, users)
	})
}

func (e *recoveryIdentityExecution) finalizeFirst(ctx context.Context, tx pgx.Tx, p recovery.PhasePermit, b recovery.RunBinding, v recovery.PermitBinding, users []recoveryExecutionUser) error {
	f := recoveryExecutionFinalization{Sources: []recoveryExecutionSource{}}
	var initial int
	ids := make([]string, 0, len(users))
	for _, u := range users {
		if u.Status != "active" && u.Status != "suspended" && u.Status != "deleted" {
			return recovery.ErrAdmissionBinding
		}
		ids = append(ids, u.ID)
		f.Accounts++
		if u.Digest != nil {
			f.Permanent++
		}
		f.Sources = append(f.Sources, recoveryExecutionSource{ID: u.ID, Role: u.Role, Status: u.Status, DeletedAt: u.DeletedAt, Initial: u.Initial})
		if u.Initial {
			initial++
			continue
		}
		f.Reviews++
		if (u.Status == "deleted") != (u.DeletedAt != nil) {
			return recovery.ErrAdmissionBinding
		}
		if err := e.guard(ctx, p, recovery.PhaseRecoveryIdentity); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, insertRecoveryAccountReviewSQL, b.RunID, u.ID, u.Role, u.Status, u.DeletedAt, RecoveryDecisionPending, nil, nil, nil, nil)
		if err != nil {
			return err
		}
	}
	if initial != 1 {
		return recovery.ErrAdmissionIncomplete
	}
	if err := e.guard(ctx, p, recovery.PhaseRecoveryIdentity); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE users SET status=CASE WHEN is_initial_admin THEN status WHEN status='deleted' THEN status ELSE 'suspended' END,recovery_required=true,credential_mode='recovery_locked',permanent_password_digest=NULL,permanent_password_generation=NULL,updated_at=$2 WHERE id=ANY($1::uuid[])`, ids, e.d.clock.Now().UTC().Truncate(time.Microsecond))
	if err != nil {
		return err
	}
	if tag.RowsAffected() != int64(len(ids)) {
		return recovery.ErrRecoveryRunConflict
	}
	if err := e.guard(ctx, p, recovery.PhaseRecoveryIdentity); err != nil {
		return err
	}
	tag, err = tx.Exec(ctx, `DELETE FROM sessions WHERE user_id=ANY($1::uuid[])`, ids)
	if err != nil {
		return err
	}
	f.Sessions = tag.RowsAffected()
	if err := e.guard(ctx, p, recovery.PhaseRecoveryIdentity); err != nil {
		return err
	}
	tag, err = tx.Exec(ctx, `UPDATE temporary_credentials SET state='recovery_invalidated',invalidated_at=$2 WHERE user_id=ANY($1::uuid[]) AND state IN('pending_delivery','active','expired')`, ids, e.d.clock.Now().UTC().Truncate(time.Microsecond))
	if err != nil {
		return err
	}
	f.Temporary = tag.RowsAffected()
	f.Cohort, err = e.invalidateIdempotency(ctx, tx, p)
	if err != nil {
		return err
	}
	f.Idempotency = int64(len(f.Cohort))
	f.CohortCount = strconv.FormatInt(f.Idempotency, 10)
	raw, err := json.Marshal(f)
	if err != nil {
		return err
	}
	var metadata map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err = decoder.Decode(&metadata); err != nil {
		return err
	}
	if err = e.event(ctx, tx, recoveryFinalizeAction, b.RunID, b, v.InstallationID, metadata); err != nil {
		return err
	}
	var auditID string
	if err = tx.QueryRow(ctx, `SELECT id::text FROM audit_events WHERE action=$1 AND target_id=$2 AND metadata->>'run_id'=$2 AND metadata->>'installation_id'=$3`, recoveryFinalizeAction, b.RunID, v.InstallationID).Scan(&auditID); err != nil {
		return err
	}
	if err = e.guard(ctx, p, recovery.PhaseRecoveryIdentity); err != nil {
		return err
	}
	tag, err = tx.Exec(ctx, `UPDATE recovery_runs SET account_count=$2,permanent_credential_count=$3,session_count=$4,temporary_credential_count=$5,idempotency_count=$6,pending_review_count=$7,audit_event_id=$8 WHERE run_id=$1 AND outcome='running' AND audit_event_id IS NULL`, b.RunID, f.Accounts, f.Permanent, f.Sessions, f.Temporary, f.Idempotency, f.Reviews, auditID)
	return recoveryExecutionOne(tag.RowsAffected(), err)
}

// Capture the original full identity set before the transition, in the very
// same caller transaction as RequireRecoveryReview and the finalization Audit.
func (e *recoveryIdentityExecution) invalidateIdempotency(ctx context.Context, tx pgx.Tx, p recovery.PhasePermit) ([]recoveryIdempotencyMember, error) {
	rows, err := tx.Query(ctx, `SELECT scope_hash,key_locator,key_digest,request_digest,digest_key_version,execution_generation,created_at,expires_at FROM idempotency_records WHERE status IN('processing','uncertain') ORDER BY scope_hash,key_locator FOR UPDATE`)
	if err != nil {
		return nil, err
	}
	items := []recoveryIdempotencyMember{}
	for rows.Next() {
		var x idempotency.Transition
		var c, re time.Time
		if err = rows.Scan(&x.ScopeHash, &x.KeyLocator, &x.KeyDigest, &x.RequestDigest, &x.DigestKeyVersion, &x.ExecutionGeneration, &c, &re); err != nil {
			rows.Close()
			return nil, err
		}
		items = append(items, recoveryMember(x, c, re))
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if err = recoveryCohortValid(items, int64(len(items))); err != nil {
		return nil, err
	}
	now := e.d.clock.Now().UTC().Truncate(time.Microsecond)
	for _, m := range items {
		if err = e.guard(ctx, p, recovery.PhaseRecoveryIdentity); err != nil {
			return nil, err
		}
		x := m.transition()
		x.Now = now
		if err = (idempotency.Store{}).RequireRecoveryReview(ctx, tx, x); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (e *recoveryIdentityExecution) finalization(ctx context.Context, tx pgx.Tx, r recovery.AdmissionRun, b recovery.RunBinding, install string) (recoveryExecutionFinalization, error) {
	raw, ok, err := recoveryExecutionAudit(ctx, tx, recoveryFinalizeAction, b.RunID, b, install)
	if err != nil {
		return recoveryExecutionFinalization{}, err
	}
	if !ok || r.AuditEventID == nil {
		return recoveryExecutionFinalization{}, recovery.ErrAdmissionIncomplete
	}
	var f recoveryExecutionFinalization
	if err = json.Unmarshal(raw, &f); err != nil {
		return f, err
	}
	if f.Accounts != r.AccountCount || f.Permanent != r.PermanentCredentialCount || f.Sessions != r.SessionCount || f.Temporary != r.TemporaryCredentialCount || f.Idempotency != r.IdempotencyCount || int64(len(f.Sources)) != f.Accounts {
		return f, recovery.ErrAdmissionBinding
	}
	if f.CohortCount != strconv.FormatInt(f.Idempotency, 10) {
		return f, recovery.ErrAdmissionIncomplete
	}
	if err = recoveryCohortValid(f.Cohort, f.Idempotency); err != nil {
		return f, err
	}
	var auditMatches bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM audit_events WHERE id=$1 AND action=$2 AND target_id=$3 AND metadata::jsonb=$4::jsonb)`, *r.AuditEventID, recoveryFinalizeAction, b.RunID, string(raw)).Scan(&auditMatches); err != nil {
		return f, err
	}
	if !auditMatches {
		return f, recovery.ErrAdmissionBinding
	}
	users, err := recoveryExecutionUsers(ctx, tx, nil, false)
	if err != nil {
		return f, err
	}
	if len(users) != len(f.Sources) {
		return f, recovery.ErrAdmissionBinding
	}
	for i, u := range users {
		if u.ID != f.Sources[i].ID || u.Initial != f.Sources[i].Initial {
			return f, recovery.ErrAdmissionBinding
		}
	}
	reviews, err := recoveryExecutionReviews(ctx, tx, b.RunID)
	if err != nil {
		return f, err
	}
	if int64(len(reviews)) != f.Reviews {
		return f, recovery.ErrAdmissionIncomplete
	}
	for _, s := range f.Sources {
		if s.Initial {
			continue
		}
		x, ok := reviews[s.ID]
		sameDeletedAt := x.BackupDeletedAt == nil && s.DeletedAt == nil ||
			x.BackupDeletedAt != nil && s.DeletedAt != nil && x.BackupDeletedAt.Equal(*s.DeletedAt)
		if !ok || x.BackupRole != s.Role || x.BackupStatus != s.Status || !sameDeletedAt {
			return f, recovery.ErrAdmissionBinding
		}
	}
	return f, nil
}

type recoveryExecutionReview struct {
	UserID          string     `json:"user_id"`
	BackupRole      string     `json:"backup_role"`
	BackupStatus    string     `json:"backup_status"`
	BackupDeletedAt *time.Time `json:"backup_deleted_at"`
	Decision        string     `json:"decision"`
	EvidenceRef     *string    `json:"evidence_ref"`
	ReviewedBy      *string    `json:"reviewed_by"`
	ReviewedAt      *time.Time `json:"reviewed_at"`
	ResetAt         *time.Time `json:"credential_reset_at"`
	OperationID     *string    `json:"restoration_operation_id"`
}

func recoveryExecutionReviews(ctx context.Context, tx pgx.Tx, run string) (map[string]recoveryExecutionReview, error) {
	rows, err := tx.Query(ctx, `SELECT to_jsonb(r)::text FROM recovery_account_reviews r WHERE recovery_run_id=$1 ORDER BY user_id`, run)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]recoveryExecutionReview{}
	for rows.Next() {
		var raw []byte
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		var v recoveryExecutionReview
		if err = json.Unmarshal(raw, &v); err != nil {
			return nil, err
		}
		if _, ok := out[v.UserID]; ok {
			return nil, recovery.ErrAdmissionBinding
		}
		out[v.UserID] = v
	}
	return out, rows.Err()
}

func recoveryExecutionInitial(users []recoveryExecutionUser) (recoveryExecutionUser, error) {
	var u recoveryExecutionUser
	for _, x := range users {
		if x.Initial {
			if u.ID != "" {
				return u, recovery.ErrAdmissionBinding
			}
			u = x
		}
	}
	if u.ID == "" {
		return u, recovery.ErrAdmissionIncomplete
	}
	return u, nil
}

func (e *recoveryIdentityExecution) trustRead(ctx context.Context, tx pgx.Tx, b recovery.RunBinding, v recovery.PermitBinding, j recovery.JournalEntry) (recovery.AdmissionRun, recoveryExecutionUser, bool, error) {
	s, err := readAdmissionTransaction(ctx, tx)
	if err != nil {
		return recovery.AdmissionRun{}, recoveryExecutionUser{}, false, err
	}
	var r recovery.AdmissionRun
	for _, x := range s.Runs {
		if x.RunID == b.RunID {
			r = x
		}
	}
	if r.RunID == "" || r.Outcome != recovery.RecoveryOutcomeRunning || r.BackupRef != b.BackupRef || s.Installation.InstallationID != v.InstallationID {
		return r, recoveryExecutionUser{}, false, recovery.ErrAdmissionBinding
	}
	if err = recoveryExecutionMatchBegin(r, j); err != nil {
		return r, recoveryExecutionUser{}, false, err
	}
	if _, err = e.finalization(ctx, tx, r, b, v.InstallationID); err != nil {
		return r, recoveryExecutionUser{}, false, err
	}
	users, err := recoveryExecutionUsers(ctx, tx, nil, false)
	if err != nil {
		return r, recoveryExecutionUser{}, false, err
	}
	u, err := recoveryExecutionInitial(users)
	if err != nil {
		return r, u, false, err
	}
	if r.InitialAdminBootstrapGeneration != nil {
		if r.InitialAdminUserID == nil || *r.InitialAdminUserID != u.ID || u.Digest == nil || u.PermanentGeneration == nil {
			return r, u, false, recovery.ErrAdmissionBinding
		}
		if r.InitialAdminChangedGeneration != nil {
			err = e.changedInitial(ctx, tx, r, u)
		} else if u.Mode != "recovery_bootstrap" || !u.Required || u.Auth != *r.InitialAdminBootstrapGeneration || *u.PermanentGeneration != u.Auth {
			err = recovery.ErrAdmissionBinding
		}
		return r, u, true, err
	}
	if u.Mode != "recovery_locked" || !u.Required || u.Auth > math.MaxInt64-2 {
		return r, u, false, recovery.ErrAdmissionBinding
	}
	err = (PasswordResetStore{}).CheckLockedUser(ctx, tx, u.ID)
	return r, u, false, err
}

func (e *recoveryIdentityExecution) PrepareRecoveryTrust(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding, secret []byte) (result error) {
	pc, v, err := e.phase(ctx, p, b, recovery.PhaseRecoveryIdentity)
	if err != nil {
		return err
	}
	j, err := e.begun(pc, b)
	if err != nil {
		return err
	}
	tx, err := e.d.store.pool.BeginTx(pc, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return err
	}
	var before recoveryExecutionUser
	var prepared bool
	err = func() (inner error) {
		defer func() { inner = errors.Join(inner, finishAdmissionRead(pc, tx)) }()
		_, before, prepared, inner = e.trustRead(pc, tx, b, v, j)
		if inner == nil {
			inner = tx.Commit(pc)
		}
		return inner
	}()
	if err != nil {
		return err
	}
	if prepared {
		return e.guard(pc, p, recovery.PhaseRecoveryIdentity)
	}
	if len(secret) == 0 {
		return recovery.ErrAdmissionBinding
	}
	if err = e.guard(pc, p, recovery.PhaseRecoveryIdentity); err != nil {
		return err
	}
	digest, err := e.d.passwords.Hash(string(secret))
	if err != nil {
		return err
	}
	if err = e.guard(pc, p, recovery.PhaseRecoveryIdentity); err != nil {
		return err
	}
	return e.write(pc, p, recovery.PhaseRecoveryIdentity, recoveryExecutionScope{action: recoveryTrustAction, binding: b, installation: v.InstallationID, actor: before.ID}, func(tx pgx.Tx) error {
		users, err := recoveryExecutionUsers(pc, tx, []string{before.ID}, true)
		if err != nil {
			return err
		}
		if len(users) != 1 {
			return recovery.ErrAdmissionBinding
		}
		if _, _, err = e.lockedRun(pc, tx, b, v, recovery.RecoveryOutcomeRunning); err != nil {
			return err
		}
		_, u, done, err := e.trustRead(pc, tx, b, v, j)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		if u.Auth != before.Auth || u.ID != before.ID {
			return recovery.ErrAdmissionBinding
		}
		if err = e.guard(pc, p, recovery.PhaseRecoveryIdentity); err != nil {
			return err
		}
		g := u.Auth + 1
		tag, err := tx.Exec(pc, `UPDATE users SET role='admin',status='active',recovery_required=true,credential_mode='recovery_bootstrap',auth_generation=$2,permanent_password_generation=$2,permanent_password_digest=$3,permanent_password_source_operation_id=NULL,permanent_password_source_credential_id=NULL,updated_at=$4 WHERE id=$1 AND is_initial_admin AND credential_mode='recovery_locked' AND auth_generation=$5`, u.ID, g, digest, e.d.clock.Now().UTC().Truncate(time.Microsecond), u.Auth)
		if err = recoveryExecutionOne(tag.RowsAffected(), err); err != nil {
			return err
		}
		if err = e.guard(pc, p, recovery.PhaseRecoveryIdentity); err != nil {
			return err
		}
		tag, err = tx.Exec(pc, `UPDATE recovery_runs SET initial_admin_user_id=$2,initial_admin_bootstrap_generation=$3 WHERE run_id=$1 AND outcome='running' AND initial_admin_bootstrap_generation IS NULL`, b.RunID, u.ID, g)
		if err = recoveryExecutionOne(tag.RowsAffected(), err); err != nil {
			return err
		}
		if err = e.guard(pc, p, recovery.PhaseRecoveryIdentity); err != nil {
			return err
		}
		if _, err = tx.Exec(pc, `UPDATE sessions SET revoked_at=$2,revoke_reason='recovery_trust' WHERE user_id=$1 AND revoked_at IS NULL`, u.ID, e.d.clock.Now().UTC().Truncate(time.Microsecond)); err != nil {
			return err
		}
		return e.event(pc, tx, recoveryTrustAction, b.RunID, b, v.InstallationID, map[string]any{"initial_admin_user_id": u.ID, "bootstrap_generation": g})
	})
}

func (e *recoveryIdentityExecution) changedInitial(ctx context.Context, tx pgx.Tx, r recovery.AdmissionRun, u recoveryExecutionUser) error {
	if r.InitialAdminUserID == nil || r.InitialAdminBootstrapGeneration == nil || r.InitialAdminChangedGeneration == nil || r.InitialAdminChangeAuditID == nil || r.InitialAdminChangedAt == nil {
		return recovery.ErrAdmissionIncomplete
	}
	if u.ID != *r.InitialAdminUserID || !u.Initial || u.Role != "admin" || u.Status != "active" || u.Required || u.Mode != "permanent" || u.Digest == nil || u.PermanentGeneration == nil || u.Auth != *r.InitialAdminChangedGeneration || *u.PermanentGeneration != u.Auth || u.SourceOperation != nil || u.SourceCredential != nil || u.Auth != *r.InitialAdminBootstrapGeneration+1 {
		return recovery.ErrAdmissionBinding
	}
	if err := (PasswordResetStore{}).CheckLockedUser(ctx, tx, u.ID); err != nil {
		return err
	}
	var good bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM audit_events WHERE id=$1 AND action=$2 AND actor_kind='user' AND actor_user_id=$3::uuid AND target_type='user' AND target_id=$3::text AND outcome='succeeded' AND occurred_at=$4 AND occurred_at >= $5)`, *r.InitialAdminChangeAuditID, passwordChangeAuditAction, u.ID, *r.InitialAdminChangedAt, r.StartedAt).Scan(&good)
	if err != nil {
		return err
	}
	if !good {
		return recovery.ErrAdmissionIncomplete
	}
	return nil
}

// resolveRestricted is private domain orchestration, not an additional route or
// authority. The raw current phase and independently changed initial are checked.
func (e *recoveryIdentityExecution) resolveRestricted(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding, target, decision, evidence string) error {
	pc, v, err := e.phase(ctx, p, b, recovery.PhaseRecoveryIdentity)
	if err != nil {
		return err
	}
	if strings.TrimSpace(evidence) == "" || (decision != "denied" && decision != "kept_deleted") {
		return recovery.ErrAdmissionBinding
	}
	s, err := e.d.store.ReadAdmissionSnapshot(pc)
	if err != nil {
		return err
	}
	var initial string
	for _, x := range s.Runs {
		if x.RunID == b.RunID && x.InitialAdminUserID != nil {
			initial = *x.InitialAdminUserID
		}
	}
	if initial == "" || target == initial {
		return recovery.ErrAdmissionBinding
	}
	return e.write(pc, p, recovery.PhaseRecoveryIdentity, recoveryExecutionScope{action: recoveryReviewAction, binding: b, installation: v.InstallationID, actor: initial, target: target, canonical: decision == "kept_deleted"}, func(tx pgx.Tx) error {
		// Match canonical actor->target ordering; never hold Store before Users.
		a, err := recoveryExecutionUsers(pc, tx, []string{initial}, true)
		if err != nil {
			return err
		}
		targets, err := recoveryExecutionUsers(pc, tx, []string{target}, true)
		if err != nil {
			return err
		}
		if len(a) != 1 || len(targets) != 1 {
			return recovery.ErrAdmissionIncomplete
		}
		if decision == "kept_deleted" {
			return e.kept(pc, tx, p, b, v, a[0], targets[0], evidence)
		}
		r, _, err := e.lockedRun(pc, tx, b, v, recovery.RecoveryOutcomeRunning)
		if err != nil {
			return err
		}
		if err = e.changedInitial(pc, tx, r, a[0]); err != nil {
			return err
		}
		if _, err = (PostgresCanonicalDeleteRepository{}).LockRecoveryAccountReview(pc, tx, b.RunID, target); err != nil {
			return err
		}
		reviews, err := recoveryExecutionReviews(pc, tx, b.RunID)
		if err != nil {
			return err
		}
		x, ok := reviews[target]
		if !ok {
			return recovery.ErrAdmissionIncomplete
		}
		if x.Decision != "pending" {
			if x.Decision != "denied" || x.EvidenceRef == nil || *x.EvidenceRef != evidence {
				return recovery.ErrAdmissionBinding
			}
			return e.reviewProof(pc, tx, b, v.InstallationID, x, targets[0])
		}
		if err = e.restricted(pc, tx, b, v.InstallationID, x, targets[0], false); err != nil {
			return err
		}
		if err = e.guard(pc, p, recovery.PhaseRecoveryIdentity); err != nil {
			return err
		}
		now := e.d.clock.Now().UTC().Truncate(time.Microsecond)
		tag, err := tx.Exec(pc, `UPDATE recovery_account_reviews SET decision='denied',evidence_ref=$3,reviewed_by=$4,reviewed_at=$5 WHERE recovery_run_id=$1 AND user_id=$2 AND decision='pending'`, b.RunID, target, evidence, initial, now)
		if err = recoveryExecutionOne(tag.RowsAffected(), err); err != nil {
			return err
		}
		return e.event(pc, tx, recoveryReviewAction, target, b, v.InstallationID, map[string]any{"decision": "denied", "evidence_ref": evidence, "actor_id": initial})
	})
}

func (e *recoveryIdentityExecution) kept(ctx context.Context, tx pgx.Tx, p recovery.PhasePermit, b recovery.RunBinding, v recovery.PermitBinding, a, u recoveryExecutionUser, evidence string) error {
	// canonical takes the Store lock only AFTER reentering its already-held User
	// locks; no distinct User is acquired while Store is held.
	reviews, err := recoveryExecutionReviews(ctx, tx, b.RunID)
	if err != nil {
		return err
	}
	x, ok := reviews[u.ID]
	if !ok {
		return recovery.ErrAdmissionIncomplete
	}
	if x.Decision != "pending" {
		if x.Decision != "kept_deleted" || x.EvidenceRef == nil || *x.EvidenceRef != evidence {
			return recovery.ErrAdmissionBinding
		}
		return e.reviewProof(ctx, tx, b, v.InstallationID, x, u)
	}
	var r recovery.AdmissionRun
	s, err := readAdmissionTransaction(ctx, tx)
	if err != nil {
		return err
	}
	for _, q := range s.Runs {
		if q.RunID == b.RunID {
			r = q
		}
	}
	if err = e.changedInitial(ctx, tx, r, a); err != nil {
		return err
	}
	if err = e.guard(ctx, p, recovery.PhaseRecoveryIdentity); err != nil {
		return err
	}
	cap, err := NewRecoveryDeleteCapability(b.RunID, b.BackupRef)
	if err != nil {
		return err
	}
	trace, _ := platform.TraceIDFromContext(ctx)
	request := CanonicalDeleteRequest{Actor: session.Principal{UserID: a.ID, Role: a.Role, Status: "active", IsInitialAdmin: true}, TargetID: u.ID, TargetRole: Role(u.Role), Mode: CanonicalDeleteRecoveryKept, Reason: AccountDeletionGovernance, TraceID: trace, Recovery: &RecoveryKeptDeletedRequest{Capability: cap, EvidenceRef: &evidence}}
	if err = validateCanonicalDeleteRequest(request); err != nil {
		return err
	}
	result, err := e.canonical.canonicalDeleteInTransaction(ctx, tx, request)
	if err != nil {
		return err
	}
	var id string
	err = tx.QueryRow(ctx, `SELECT id::text FROM audit_events WHERE action=$1 AND target_id=$2 AND trace_id=$3 ORDER BY occurred_at DESC,id DESC LIMIT 1`, canonicalDeleteAuditAction, u.ID, trace).Scan(&id)
	if err != nil {
		return err
	}
	return e.event(ctx, tx, recoveryReviewAction, u.ID, b, v.InstallationID, map[string]any{"decision": "kept_deleted", "evidence_ref": evidence, "actor_id": a.ID, "canonical_audit_id": id, "pending_orphan_count": result.PendingOrphanCount})
}

func (e *recoveryIdentityExecution) restricted(ctx context.Context, tx pgx.Tx, b recovery.RunBinding, install string, x recoveryExecutionReview, u recoveryExecutionUser, canonical bool) error {
	if !u.Required || u.Mode != "recovery_locked" || u.Digest != nil || x.ResetAt != nil {
		return recovery.ErrAdmissionBinding
	}
	if x.BackupStatus == "deleted" || canonical {
		if u.Status != "deleted" || u.DeletedAt == nil || u.Email != nil || u.Phone != nil {
			return recovery.ErrAdmissionBinding
		}
		if x.BackupStatus == "deleted" && (x.BackupDeletedAt == nil || !u.DeletedAt.Equal(*x.BackupDeletedAt)) {
			return recovery.ErrAdmissionBinding
		}
	} else if u.Status != "suspended" || u.DeletedAt != nil {
		return recovery.ErrAdmissionBinding
	}
	var active int64
	if err := tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM temporary_credentials WHERE user_id=$1 AND state IN('pending_delivery','active','expired'))+(SELECT count(*) FROM sessions WHERE user_id=$1 AND revoked_at IS NULL)`, u.ID).Scan(&active); err != nil {
		return err
	}
	if active != 0 {
		return recovery.ErrAdmissionIncomplete
	}
	if u.Status == "deleted" {
		if err := recoveryExecutionTombstone(ctx, tx, u); err != nil {
			return err
		}
	}
	return e.pointer(ctx, tx, b, install, x)
}

func recoveryReadResetOperation(ctx context.Context, tx pgx.Tx, id string) (PasswordResetOperation, error) {
	var o PasswordResetOperation
	err := tx.QueryRow(ctx, `SELECT operation_id::text,user_id::text,temporary_credential_id::text,scope_hash,key_locator,key_digest,request_digest,digest_key_version,execution_generation,request_kind,predecessor_operation_id::text,replaced_credential_id::text,base_auth_generation,target_auth_generation,settlement_state,settled_auth_generation,successor_auth_generation,settled_at,delivery_state,delivery_attempt_id::text,delivery_attempted_at,delivery_result_recorded_at,state,terminal_at,created_at,updated_at FROM password_reset_operations WHERE operation_id=$1`, id).Scan(&o.ID, &o.UserID, &o.TemporaryCredentialID, &o.ScopeHash, &o.KeyLocator, &o.KeyDigest, &o.RequestDigest, &o.DigestKeyVersion, &o.ExecutionGeneration, &o.RequestKind, &o.PredecessorOperationID, &o.ReplacedCredentialID, &o.BaseAuthGeneration, &o.TargetAuthGeneration, &o.SettlementState, &o.SettledAuthGeneration, &o.SuccessorAuthGeneration, &o.SettledAt, &o.DeliveryState, &o.DeliveryAttemptID, &o.DeliveryAttemptedAt, &o.DeliveryResultRecordedAt, &o.State, &o.TerminalAt, &o.CreatedAt, &o.UpdatedAt)
	return o, err
}

// Original worker invalidation and synchronous rejected delivery have distinct
// writers. Terminal job retries do not produce a second settlement Audit.
func recoveryFailedResetSettlement(ctx context.Context, tx pgx.Tx, o PasswordResetOperation) error {
	if o.SettledAt == nil || o.TerminalAt == nil || !passwordResetOperationAuthorityValid(o) || !passwordResetDeliveryEvidenceValid(o) || o.SettlementState != PasswordResetSettlementInvalidated {
		return recovery.ErrAdmissionIncomplete
	}
	var good bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM password_reset_recovery_jobs j WHERE j.operation_id=$1::uuid AND j.status='succeeded' AND j.completed_at IS NOT NULL AND j.completed_at >= $2 AND j.lease_owner IS NULL AND j.lease_expires_at IS NULL)`, o.ID, *o.SettledAt).Scan(&good)
	if err != nil {
		return err
	}
	if !good {
		return recovery.ErrAdmissionIncomplete
	}
	switch o.State {
	case PasswordResetOperationInvalidated:
		// The original settlement may precede a later terminal job completion.
		// Match its own immutable attempt, not an invented Audit for the retry.
		err = tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM audit_events a WHERE a.action='identity.password_reset.recovered' AND a.target_type='password_reset_operation' AND a.target_id=$1::text)=1 AND EXISTS(SELECT 1 FROM audit_events a JOIN password_reset_recovery_jobs j ON j.operation_id=$1::uuid WHERE a.action='identity.password_reset.recovered' AND a.target_type='password_reset_operation' AND a.target_id=$1::text AND a.outcome='succeeded' AND a.actor_kind='system' AND a.actor_system='password-reset-worker' AND a.occurred_at=$2 AND jsonb_typeof(a.metadata->'attempt_count')='number' AND CASE WHEN a.metadata->>'attempt_count' ~ '^[1-9][0-9]*$' THEN (a.metadata->>'attempt_count')::numeric BETWEEN 1 AND j.attempt_count ELSE false END)`, o.ID, *o.SettledAt).Scan(&good)
		if err != nil {
			return err
		}
		if !good {
			return recovery.ErrAdmissionIncomplete
		}
	case PasswordResetOperationFailed:
		// RecordRejectedDelivery proves non-acceptance and closes only this
		// operation/credential/job. It never appends the worker's recovered Audit.
		if o.DeliveryState != PasswordResetDeliveryRejected {
			return recovery.ErrAdmissionIncomplete
		}
	default:
		return recovery.ErrAdmissionIncomplete
	}
	_, replay := classifyPasswordResetReplay(passwordResetPrepared{operation: o, target: User{ID: o.UserID}})
	if platform.PublicCode(replay) != platform.CodeIdentityEmailDeliveryFailed {
		return recovery.ErrAdmissionIncomplete
	}
	return nil
}

// This is a retained original-domain history, never a new reset/Retry producer.
func (e *recoveryIdentityExecution) pointer(ctx context.Context, tx pgx.Tx, b recovery.RunBinding, install string, x recoveryExecutionReview) error {
	if x.OperationID == nil {
		return nil
	}
	if x.BackupStatus == "deleted" {
		return recovery.ErrAdmissionBinding
	}
	o, err := recoveryReadResetOperation(ctx, tx, *x.OperationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return recovery.ErrAdmissionIncomplete
	}
	if err != nil {
		return err
	}
	if o.UserID != x.UserID || o.RequestKind != PasswordResetInitialReset || o.PredecessorOperationID != nil || o.ReplacedCredentialID != nil || !passwordResetOperationAuthorityValid(o) || !passwordResetDeliveryEvidenceValid(o) || (o.State != PasswordResetOperationFailed && o.State != PasswordResetOperationInvalidated) || o.SettlementState != PasswordResetSettlementInvalidated {
		return recovery.ErrAdmissionIncomplete
	}
	var bound bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM password_reset_operations o JOIN recovery_runs r ON r.run_id=o.recovery_run_id WHERE o.operation_id=$1 AND o.user_id=$2 AND o.recovery_run_id=$3 AND o.recovery_installation_id=$4 AND o.recovery_actor_id=r.initial_admin_user_id::uuid AND o.recovery_evidence_ref IS NOT NULL AND btrim(o.recovery_evidence_ref)<>'' AND o.recovery_evidence_digest ~ '^sha256:[0-9a-f]{64}$')`, o.ID, x.UserID, b.RunID, install).Scan(&bound)
	if err != nil {
		return err
	}
	if !bound {
		return recovery.ErrAdmissionBinding
	}
	var c TemporaryCredential
	err = tx.QueryRow(ctx, `SELECT id::text,user_id::text,purpose,auth_generation,source_operation_id::text,state FROM temporary_credentials WHERE id=$1`, o.TemporaryCredentialID).Scan(&c.ID, &c.UserID, &c.Purpose, &c.AuthGeneration, &c.SourceOperationID, &c.State)
	if err != nil {
		return err
	}
	if !passwordResetResendCredentialExact(o, c) || c.State != TemporaryCredentialRecoveryInvalidated {
		return recovery.ErrAdmissionIncomplete
	}
	if err = recoveryFailedResetSettlement(ctx, tx, o); err != nil {
		return err
	}
	var status string
	var code, snapshot *string
	var created, expires time.Time
	err = tx.QueryRow(ctx, `SELECT i.status,i.result_code,i.response_snapshot::text,i.created_at,i.expires_at FROM idempotency_records i WHERE i.password_reset_operation_id=$1 AND i.password_reset_user_id=$2 AND i.scope_hash=$3 AND i.key_locator=$4 AND i.key_digest=$5 AND i.request_digest=$6 AND i.digest_key_version=$7 AND i.execution_generation=$8`, o.ID, o.UserID, o.ScopeHash, o.KeyLocator, o.KeyDigest, o.RequestDigest, o.DigestKeyVersion, o.ExecutionGeneration).Scan(&status, &code, &snapshot, &created, &expires)
	if errors.Is(err, pgx.ErrNoRows) {
		return recovery.ErrAdmissionIncomplete
	}
	if err != nil {
		return err
	}
	if !recoveryFailedResetResponse(status, code, snapshot, o.ID) {
		return recovery.ErrAdmissionBinding
	}
	if o.State == PasswordResetOperationFailed {
		// Direct rejection does not write a worker-settlement Audit or a failed
		// generic claim. A subsequently resolved claim needs the REAL original
		// r56 cohort/resolution, not an invented direct-failure variant. Complete
		// independently re-reads that cohort's source outside this transaction.
		member := recoveryMember(idempotency.Transition{ScopeHash: o.ScopeHash, KeyLocator: o.KeyLocator, KeyDigest: o.KeyDigest, RequestDigest: o.RequestDigest, DigestKeyVersion: o.DigestKeyVersion, ExecutionGeneration: o.ExecutionGeneration}, created, expires)
		raw, exists, er := recoveryExecutionAudit(ctx, tx, recoveryFinalizeAction, b.RunID, b, install)
		if er != nil {
			return er
		}
		var final recoveryExecutionFinalization
		if !exists || json.Unmarshal(raw, &final) != nil {
			return recovery.ErrAdmissionIncomplete
		}
		found := false
		for _, m := range final.Cohort {
			if reflect.DeepEqual(member, m) {
				found = true
			}
		}
		if !found {
			return recovery.ErrAdmissionIncomplete
		}
		a, er := recoveryReadResolution(ctx, tx, b, install, member)
		if er != nil {
			return er
		}
		record, er := recoveryReadIdempotency(ctx, tx, member, false)
		if er != nil {
			return er
		}
		if record.ReviewedBy == nil || *record.ReviewedBy != a.Actor || record.ReviewedAt == nil || !record.ReviewedAt.Equal(a.Now) || !record.UpdatedAt.Equal(a.Now) || record.LockedUntil != nil || a.ResultCode != *code || a.Purpose != passwordResetOperation || a.Now.Before(a.PriorUpdatedAt) {
			return recovery.ErrAdmissionBinding
		}
	}
	return (PasswordResetStore{}).CheckLockedUser(ctx, tx, o.UserID)
}

func recoveryFailedResetResponse(status string, code, snapshot *string, operation string) bool {
	if !passwordResetIdempotencyTerminal(passwordResetSettlementAuthority{idempotencyStatus: status, resultCode: code, responseSnapshot: snapshot}, operation, false) {
		return false
	}
	var fields map[string]json.RawMessage
	return json.Unmarshal([]byte(*snapshot), &fields) == nil && len(fields) == 2 && fields["outcome"] != nil && fields["resource_id"] != nil
}

func (e *recoveryIdentityExecution) reviewProof(ctx context.Context, tx pgx.Tx, b recovery.RunBinding, install string, x recoveryExecutionReview, u recoveryExecutionUser) error {
	if x.ReviewedBy == nil || x.ReviewedAt == nil || x.EvidenceRef == nil || strings.TrimSpace(*x.EvidenceRef) == "" {
		return recovery.ErrAdmissionIncomplete
	}
	raw, ok, err := recoveryExecutionAudit(ctx, tx, recoveryReviewAction, u.ID, b, install)
	if err != nil {
		return err
	}
	if !ok {
		return recovery.ErrAdmissionIncomplete
	}
	var m struct {
		Decision  string `json:"decision"`
		Evidence  string `json:"evidence_ref"`
		Actor     string `json:"actor_id"`
		Canonical string `json:"canonical_audit_id"`
	}
	if err = json.Unmarshal(raw, &m); err != nil {
		return err
	}
	if m.Decision != x.Decision || m.Evidence != *x.EvidenceRef || m.Actor != *x.ReviewedBy {
		return recovery.ErrAdmissionBinding
	}
	var auditCount int64
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action=$1 AND target_id=$2::text AND target_type='user' AND outcome='succeeded' AND actor_kind='user' AND actor_user_id=$3::uuid AND actor_role='admin' AND occurred_at >= $4 AND metadata=$5::jsonb`, recoveryReviewAction, u.ID, *x.ReviewedBy, *x.ReviewedAt, string(raw)).Scan(&auditCount); err != nil {
		return err
	}
	if auditCount != 1 {
		return recovery.ErrAdmissionBinding
	}
	switch x.Decision {
	case "denied":
		return e.restricted(ctx, tx, b, install, x, u, false)
	case "kept_deleted":
		if err = e.restricted(ctx, tx, b, install, x, u, true); err != nil {
			return err
		}
		return e.canonicalProof(ctx, tx, x, u, m.Canonical)
	case "restored":
		return e.restoredProof(ctx, tx, b, install, x, u)
	default:
		return recovery.ErrAdmissionIncomplete
	}
}

func (e *recoveryIdentityExecution) canonicalProof(ctx context.Context, tx pgx.Tx, x recoveryExecutionReview, u recoveryExecutionUser, auditID string) error {
	var good bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM audit_events WHERE id=$1 AND action=$2 AND target_id=$3::text AND outcome='succeeded' AND actor_user_id=$4::uuid AND metadata->>'mode'='recovery_kept_deleted') AND NOT EXISTS(SELECT 1 FROM articles WHERE author_id=$3::uuid AND deletion_state='active') AND NOT EXISTS(SELECT 1 FROM articles a WHERE a.orphaned_from_user_id=$3::uuid AND a.deletion_state='active' AND (a.author_id IS NOT NULL OR NOT EXISTS(SELECT 1 FROM audit_events ae WHERE ae.action='article.orphan.canonical' AND ae.target_id=a.id::text AND ae.metadata->>'deleted_user_id'=$3::text AND (ae.metadata->>'article_version')::bigint<=a.version)))`, auditID, canonicalDeleteAuditAction, u.ID, *x.ReviewedBy).Scan(&good)
	if err != nil {
		return err
	}
	if !good {
		return recovery.ErrAdmissionIncomplete
	}
	return recoveryExecutionTombstone(ctx, tx, u)
}

func recoveryExecutionTombstone(ctx context.Context, tx pgx.Tx, u recoveryExecutionUser) error {
	var username string
	var err error
	if err = tx.QueryRow(ctx, `SELECT username FROM users WHERE id=$1`, u.ID).Scan(&username); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT user_id::text,kind,normalized_value,lifecycle,released_at FROM identity_identifiers WHERE user_id=$1 ORDER BY kind`, u.ID)
	if err != nil {
		return err
	}
	ids := []IdentityIdentifier{}
	for rows.Next() {
		var id IdentityIdentifier
		if err = rows.Scan(&id.UserID, &id.Kind, &id.NormalizedValue, &id.Lifecycle, &id.ReleasedAt); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	// Legal canonical email release removes its identifier; the original B
	// validator still checks every retained applicable row and username.
	return validateCanonicalDeleteIdentifierTombstones(User{ID: u.ID, Username: username, Status: UserStatusDeleted, RetainedEmail: u.Retained}, ids)
}

func (e *recoveryIdentityExecution) restoredProof(ctx context.Context, tx pgx.Tx, b recovery.RunBinding, install string, x recoveryExecutionReview, u recoveryExecutionUser) error {
	if x.BackupStatus == "deleted" || u.Status == "deleted" || x.OperationID == nil || x.ResetAt == nil || x.ReviewedAt == nil || !x.ReviewedAt.Equal(*x.ResetAt) || u.Required {
		return recovery.ErrAdmissionBinding
	}
	var good bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM password_reset_operations o JOIN temporary_credentials c ON c.id=o.temporary_credential_id AND c.user_id=o.user_id JOIN password_reset_recovery_jobs j ON j.operation_id=o.operation_id JOIN idempotency_records i ON i.password_reset_operation_id=o.operation_id AND i.password_reset_user_id=o.user_id AND i.scope_hash=o.scope_hash AND i.key_locator=o.key_locator AND i.key_digest=o.key_digest AND i.request_digest=o.request_digest AND i.digest_key_version=o.digest_key_version AND i.execution_generation=o.execution_generation WHERE o.operation_id=$1 AND o.user_id=$2 AND o.recovery_run_id=$3 AND o.recovery_installation_id=$4 AND o.recovery_actor_id=$5::uuid AND o.recovery_evidence_ref=$6 AND o.recovery_evidence_digest ~ '^sha256:[0-9a-f]{64}$' AND o.recovery_target_role=$7 AND o.recovery_target_status=$8 AND o.state='succeeded' AND o.delivery_state='accepted' AND o.settlement_state IN('temporary_active','permanent_successor') AND o.settled_at=$9 AND c.source_operation_id=o.operation_id AND c.auth_generation=o.target_auth_generation AND j.status='succeeded' AND j.lease_owner IS NULL AND i.status='succeeded' AND i.result_code IS NOT NULL AND i.response_snapshot IS NOT NULL AND (SELECT count(*) FROM audit_events ae WHERE ae.action='identity.password_reset.recovered' AND ae.target_id=o.operation_id::text AND ae.target_type='password_reset_operation' AND ae.outcome='succeeded' AND ae.actor_kind='system' AND ae.actor_system IS NOT NULL AND ae.occurred_at=o.settled_at AND (ae.metadata->>'attempt_count')::bigint=j.attempt_count)=1)`, *x.OperationID, u.ID, b.RunID, install, *x.ReviewedBy, *x.EvidenceRef, u.Role, u.Status, *x.ResetAt).Scan(&good)
	if err != nil {
		return err
	}
	if !good {
		return recovery.ErrAdmissionIncomplete
	}
	// Reuse the original durable domain terminal classifier. Non-NULL alone
	// admits generic legacy results that the real reset owner never emits.
	var resultCode, snapshot string
	if err = tx.QueryRow(ctx, `SELECT i.result_code,i.response_snapshot::text FROM password_reset_operations o JOIN idempotency_records i ON i.password_reset_operation_id=o.operation_id AND i.password_reset_user_id=o.user_id AND i.scope_hash=o.scope_hash AND i.key_locator=o.key_locator AND i.key_digest=o.key_digest AND i.request_digest=o.request_digest AND i.digest_key_version=o.digest_key_version AND i.execution_generation=o.execution_generation WHERE o.operation_id=$1::uuid AND o.user_id=$2::uuid AND o.recovery_run_id=$3 AND o.recovery_installation_id=$4 AND i.status='succeeded'`, *x.OperationID, u.ID, b.RunID, install).Scan(&resultCode, &snapshot); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal([]byte(snapshot), &fields); err != nil {
		return err
	}
	if len(fields) != 2 || fields["outcome"] == nil || fields["resource_id"] == nil {
		return recovery.ErrAdmissionBinding
	}
	authority := passwordResetSettlementAuthority{idempotencyStatus: "succeeded", resultCode: &resultCode, responseSnapshot: &snapshot}
	if !passwordResetIdempotencyTerminal(authority, *x.OperationID, true) {
		return recovery.ErrAdmissionBinding
	}
	return (PasswordResetStore{}).CheckLockedUser(ctx, tx, u.ID)
}

func (e *recoveryIdentityExecution) completeFacts(ctx context.Context, tx pgx.Tx, b recovery.RunBinding, v recovery.PermitBinding, r recovery.AdmissionRun, proof *recoveryCohortProof) error {
	f, err := e.finalization(ctx, tx, r, b, v.InstallationID)
	if err != nil {
		return err
	}
	if proof == nil {
		return recovery.ErrAdmissionIncomplete
	}
	if err = e.checkCohortProof(ctx, tx, b, v, r, f, proof); err != nil {
		return err
	}
	users, err := recoveryExecutionUsers(ctx, tx, nil, false)
	if err != nil {
		return err
	}
	initial, err := recoveryExecutionInitial(users)
	if err != nil {
		return err
	}
	if err = e.changedInitial(ctx, tx, r, initial); err != nil {
		return err
	}
	reviews, err := recoveryExecutionReviews(ctx, tx, b.RunID)
	if err != nil {
		return err
	}
	if int64(len(reviews)) != f.Reviews {
		return recovery.ErrAdmissionIncomplete
	}
	for _, u := range users {
		if u.Initial {
			continue
		}
		x, ok := reviews[u.ID]
		if !ok || x.ReviewedBy == nil || *x.ReviewedBy != initial.ID {
			return recovery.ErrAdmissionIncomplete
		}
		if err = e.reviewProof(ctx, tx, b, v.InstallationID, x, u); err != nil {
			return err
		}
	}
	var unresolved int64
	err = tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM idempotency_records WHERE status IN('processing','uncertain','recovery_review_required'))+(SELECT count(*) FROM password_reset_operations o WHERE o.state NOT IN('succeeded','failed','invalidated'))`).Scan(&unresolved)
	if err != nil {
		return err
	}
	if unresolved != 0 {
		return recovery.ErrAdmissionIncomplete
	}
	return nil
}

func (e *recoveryIdentityExecution) CompleteIdentityReview(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
	pc, v, err := e.phase(ctx, p, b, recovery.PhaseRecoveryIdentity)
	if err != nil {
		return err
	}
	j, err := e.begun(pc, b)
	if err != nil {
		return err
	}
	proof, err := e.prepareCohortProof(pc, p, b, v)
	if err != nil {
		return err
	}
	return e.write(pc, p, recovery.PhaseRecoveryIdentity, recoveryExecutionScope{action: recoveryCompleteObservation, binding: b, installation: v.InstallationID}, func(tx pgx.Tx) error {
		var outcome recovery.RecoveryOutcome
		if err := tx.QueryRow(pc, `SELECT outcome FROM recovery_runs WHERE run_id=$1 FOR UPDATE`, b.RunID).Scan(&outcome); err != nil {
			return err
		}
		if outcome == recovery.RecoveryOutcomeFailed {
			return recovery.ErrAdmissionBinding
		}
		r, _, err := e.lockedRun(pc, tx, b, v, outcome)
		if err != nil {
			return err
		}
		if err = recoveryExecutionMatchBegin(r, j); err != nil {
			return err
		}
		if err = e.completeFacts(pc, tx, b, v, r, proof); err != nil {
			return err
		}
		if err = e.guard(pc, p, recovery.PhaseRecoveryIdentity); err != nil {
			return err
		}
		if outcome != recovery.RecoveryOutcomeRunning {
			return nil
		} // exact accepted readonly retry, never Consume
		tag, err := tx.Exec(pc, `UPDATE recovery_runs SET outcome='identity_ready',pending_review_count=0 WHERE run_id=$1 AND outcome='running'`, b.RunID)
		return recoveryExecutionOne(tag.RowsAffected(), err)
	})
}

func recoveryExecutionExitIDs(ctx context.Context, tx pgx.Tx, run string) ([]string, error) {
	rows, err := tx.Query(ctx, `SELECT DISTINCT user_id::text FROM sessions WHERE access_scope='recovery_maintenance' AND recovery_run_id=$1 ORDER BY user_id::text`, run)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (e *recoveryIdentityExecution) RevokeRecoverySessions(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
	pc, v, err := e.phase(ctx, p, b, recovery.PhaseRecoveryReplay)
	if err != nil {
		return err
	}
	j, err := e.begun(pc, b)
	if err != nil {
		return err
	}
	prior, err := e.d.store.Load(pc, b.RunID)
	if err != nil {
		return err
	}
	consumed, err := e.d.journal.VerifyAgainst(pc, prior)
	if err != nil {
		return err
	}
	if consumed.Stage != recovery.StageConsumed || prior.JournalConsumedAt == nil || !consumed.RecordedAt.Equal(*prior.JournalConsumedAt) {
		return recovery.ErrAdmissionBinding
	}
	return e.write(pc, p, recovery.PhaseRecoveryReplay, recoveryExecutionScope{action: recoveryExitAction, binding: b, installation: v.InstallationID}, func(tx pgx.Tx) error {
		ids, err := recoveryExecutionExitIDs(pc, tx, b.RunID)
		if err != nil {
			return err
		}
		s, err := readAdmissionTransaction(pc, tx)
		if err != nil {
			return err
		}
		for _, r := range s.Runs {
			if r.RunID == b.RunID && r.InitialAdminUserID != nil {
				ids = append(ids, *r.InitialAdminUserID)
			}
		}
		sort.Strings(ids)
		unique := []string{}
		for _, id := range ids {
			if len(unique) == 0 || unique[len(unique)-1] != id {
				unique = append(unique, id)
			}
		}
		users, err := recoveryExecutionUsers(pc, tx, unique, true)
		if err != nil {
			return err
		}
		if len(users) != len(unique) {
			return recovery.ErrAdmissionIncomplete
		}
		r, _, err := e.lockedRun(pc, tx, b, v, recovery.RecoveryOutcomeSucceeded)
		if err != nil {
			return err
		}
		if err = recoveryExecutionMatchBegin(r, j); err != nil {
			return err
		}
		if r.JournalStage != recovery.StageConsumed || r.JournalConsumedAt == nil || !r.JournalConsumedAt.Equal(consumed.RecordedAt) || r.ArticlePublicationRef == nil || r.ArticlePublicationDigest == nil || r.ArticleEvidenceRef == nil || r.ArticleEvidenceDigest == nil || r.ArticleEvidenceCompletedAt == nil || r.CompletedAt == nil || r.PendingReviewCount != 0 {
			return recovery.ErrAdmissionIncomplete
		}
		u, err := recoveryExecutionInitial(users)
		if err != nil {
			return err
		}
		if err = e.changedInitial(pc, tx, r, u); err != nil {
			return err
		}
		after, err := recoveryExecutionExitIDs(pc, tx, b.RunID)
		if err != nil {
			return err
		}
		for _, id := range after {
			n := sort.SearchStrings(unique, id)
			if n == len(unique) || unique[n] != id {
				return recovery.ErrAdmissionBinding
			}
		}
		return e.exitLocked(pc, tx, p, b, v)
	})
}

func recoveryExitReceiptIDs(raw json.RawMessage) ([]string, error) {
	var m struct {
		IDs   *[]string `json:"revoked_ids"`
		Count *int64    `json:"revoked_count"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, recovery.ErrAdmissionBinding
	}
	if m.IDs == nil || m.Count == nil || *m.Count < 0 || int64(len(*m.IDs)) != *m.Count {
		return nil, recovery.ErrAdmissionBinding
	}
	seen := map[string]bool{}
	for _, id := range *m.IDs {
		if !recoveryUUID.MatchString(id) || seen[id] {
			return nil, recovery.ErrAdmissionBinding
		}
		seen[id] = true
	}
	return append([]string{}, (*m.IDs)...), nil
}

func (e *recoveryIdentityExecution) exitLocked(ctx context.Context, tx pgx.Tx, p recovery.PhasePermit, b recovery.RunBinding, v recovery.PermitBinding) error {
	rows, err := tx.Query(ctx, `SELECT id::text,recovery_installation_id,revoked_at,revoke_reason FROM sessions WHERE access_scope='recovery_maintenance' AND recovery_run_id=$1 ORDER BY user_id,id FOR UPDATE`, b.RunID)
	if err != nil {
		return err
	}
	ids := []string{}
	exitRevoked := map[string]bool{}
	for rows.Next() {
		var id, install string
		var revoked *time.Time
		var reason *string
		if err = rows.Scan(&id, &install, &revoked, &reason); err != nil {
			rows.Close()
			return err
		}
		if install != v.InstallationID {
			rows.Close()
			return recovery.ErrAdmissionBinding
		}
		if revoked == nil {
			ids = append(ids, id)
		} else if reason != nil && *reason == recoveryExitAction {
			exitRevoked[id] = true
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	raw, exists, err := recoveryExecutionAudit(ctx, tx, recoveryExitAction, b.RunID, b, v.InstallationID)
	if err != nil {
		return err
	}
	if exists {
		receipt, er := recoveryExitReceiptIDs(raw)
		if er != nil {
			return er
		}
		if len(ids) != 0 || len(receipt) != len(exitRevoked) {
			return recovery.ErrAdmissionBinding
		}
		for _, id := range receipt {
			if !exitRevoked[id] {
				return recovery.ErrAdmissionBinding
			}
		}
		var exact int64
		if er = tx.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action=$1 AND target_type='recovery_run' AND target_id=$2 AND outcome='succeeded' AND actor_kind='system' AND actor_system='recovery_runtime' AND metadata=$3::jsonb`, recoveryExitAction, b.RunID, string(raw)).Scan(&exact); er != nil {
			return er
		}
		if exact != 1 {
			return recovery.ErrAdmissionBinding
		}
		return nil
	}
	// An earlier exit write without its atomic receipt is partial evidence,
	// not a new empty first exit. Never synthesize an Audit to repair it.
	if len(exitRevoked) != 0 {
		return recovery.ErrAdmissionIncomplete
	}
	if err = e.guard(ctx, p, recovery.PhaseRecoveryReplay); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at=$3,revoke_reason='recovery_maintenance_exit' WHERE access_scope='recovery_maintenance' AND recovery_run_id=$1 AND recovery_installation_id=$2 AND revoked_at IS NULL`, b.RunID, v.InstallationID, e.d.clock.Now().UTC().Truncate(time.Microsecond))
	if err != nil {
		return err
	}
	if tag.RowsAffected() != int64(len(ids)) {
		return recovery.ErrRecoveryRunConflict
	}
	if err = e.guard(ctx, p, recovery.PhaseRecoveryReplay); err != nil {
		return err
	}
	return e.event(ctx, tx, recoveryExitAction, b.RunID, b, v.InstallationID, map[string]any{"revoked_ids": ids, "revoked_count": len(ids)})
}

// RecoveryFailedEvidence is an owned, read-only projection of one existing
// independently approved immutable original. It is not a grant or new wire DTO.
// Root supplies the independently approved read-only source adapter. Missing
// real provider/approval/ACL evidence keeps the affected branch closed.
type RecoveryFailedEvidence struct {
	Version, EvidenceRef, EvidenceDigest      string
	InstallationID, RunID, BackupRef, ActorID string
	Scope                                     []byte
	ApprovedAt                                time.Time
	Recovery                                  idempotency.Recovery
}
type RecoveryFailedEvidenceReader interface {
	ReadFailed(context.Context, recovery.PermitBinding, string, []byte, []byte, int64) (RecoveryFailedEvidence, error)
}

var recoveryUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

const recoveryFailedEvidenceVersion = "han-recovery-idempotency-failed/v1"
const recoveryIdempotencyAction = "identity.recovery.idempotency.resolve"

type recoveryIdempotencyMember struct {
	ScopeHash           string    `json:"scope_hash"`
	KeyLocator          string    `json:"key_locator"`
	KeyDigest           string    `json:"key_digest"`
	RequestDigest       string    `json:"request_digest"`
	DigestKeyVersion    string    `json:"digest_key_version"`
	ExecutionGeneration int64     `json:"-"`
	CreatedAt           time.Time `json:"created_at"`
	ExpiresAt           time.Time `json:"expires_at"`
}

func recoveryMember(x idempotency.Transition, c, re time.Time) recoveryIdempotencyMember {
	return recoveryIdempotencyMember{ScopeHash: hex.EncodeToString(x.ScopeHash), KeyLocator: hex.EncodeToString(x.KeyLocator), KeyDigest: hex.EncodeToString(x.KeyDigest), RequestDigest: hex.EncodeToString(x.RequestDigest), DigestKeyVersion: x.DigestKeyVersion, ExecutionGeneration: x.ExecutionGeneration, CreatedAt: c.UTC(), ExpiresAt: re.UTC()}
}
func recoveryHex(s string) bool {
	if len(s) != 64 || s != strings.ToLower(s) {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}
func (m recoveryIdempotencyMember) transition() idempotency.Transition {
	a, _ := hex.DecodeString(m.ScopeHash)
	b, _ := hex.DecodeString(m.KeyLocator)
	c, _ := hex.DecodeString(m.KeyDigest)
	d, _ := hex.DecodeString(m.RequestDigest)
	return idempotency.Transition{ScopeHash: a, KeyLocator: b, KeyDigest: c, RequestDigest: d, DigestKeyVersion: m.DigestKeyVersion, ExecutionGeneration: m.ExecutionGeneration}
}
func (m recoveryIdempotencyMember) target() string {
	return m.ScopeHash + "/" + m.KeyLocator + "/" + strconv.FormatInt(m.ExecutionGeneration, 10)
}
func recoveryCohortValid(items []recoveryIdempotencyMember, n int64) error {
	if items == nil || n < 0 || int64(len(items)) != n {
		return recovery.ErrAdmissionIncomplete
	}
	seen := map[string]bool{}
	previous := ""
	for _, m := range items {
		key := m.ScopeHash + "/" + m.KeyLocator
		if !recoveryHex(m.ScopeHash) || !recoveryHex(m.KeyLocator) || !recoveryHex(m.KeyDigest) || !recoveryHex(m.RequestDigest) || strings.TrimSpace(m.DigestKeyVersion) == "" || m.ExecutionGeneration <= 0 || m.CreatedAt.IsZero() || !m.ExpiresAt.After(m.CreatedAt) || seen[key] || (previous != "" && key <= previous) {
			return recovery.ErrAdmissionBinding
		}
		seen[key] = true
		previous = key
	}
	return nil
}
func recoveryCopyFailed(x RecoveryFailedEvidence) RecoveryFailedEvidence {
	x.Scope = append([]byte(nil), x.Scope...)
	r := &x.Recovery
	r.ScopeHash = append([]byte(nil), r.ScopeHash...)
	r.KeyLocator = append([]byte(nil), r.KeyLocator...)
	r.KeyDigest = append([]byte(nil), r.KeyDigest...)
	r.RequestDigest = append([]byte(nil), r.RequestDigest...)
	if r.ResponseSnapshot.Body != nil {
		b := *r.ResponseSnapshot.Body
		if b.FieldErrors != nil {
			b.FieldErrors = append([]idempotency.ResponseFieldError{}, b.FieldErrors...)
		}
		r.ResponseSnapshot.Body = &b
	}
	return x
}
func recoveryScopeHash(scope []byte) []byte {
	h := sha256.New()
	for _, part := range [][]byte{[]byte("han/idempotency/scope/v1"), scope} {
		var n [8]byte
		binary.BigEndian.PutUint64(n[:], uint64(len(part)))
		_, _ = h.Write(n[:])
		_, _ = h.Write(part)
	}
	return h.Sum(nil)
}

// Scope is returned by the verified source, never reconstructed from a hash or
// the current request. Only actual existing domain encodings are recognized.
func recoveryFailedPurpose(x RecoveryFailedEvidence) (purpose, target string, err error) {
	parts := strings.Split(string(x.Scope), "\x00")
	if len(parts) < 3 || !recoveryUUID.MatchString(parts[0]) {
		return "", "", recovery.ErrAdmissionBinding
	}
	s := x.Recovery.ResponseSnapshot
	switch parts[1] {
	case administratorCreateOperation:
		if len(parts) != 3 || !validUsername(parts[2]) || parts[2] != strings.ToLower(strings.TrimSpace(parts[2])) {
			return "", "", recovery.ErrAdmissionBinding
		}
		want, e := administratorFailureSnapshot(platform.Code(x.Recovery.ResultCode))
		if e != nil || !reflect.DeepEqual(s, want) {
			return "", "", recovery.ErrAdmissionBinding
		}
		return parts[1], "", nil
	case temporaryCredentialResendOperation:
		if len(parts) != 4 || !recoveryUUID.MatchString(parts[2]) || parts[3] != string(TemporaryCredentialResendNewAdministrator) || x.Recovery.ResultCode != string(platform.CodeIdentityEmailDeliveryFailed) || !reflect.DeepEqual(s, temporaryCredentialResendFailureSnapshot()) {
			return "", "", recovery.ErrAdmissionBinding
		}
		return parts[1], parts[2], nil
	case passwordResetOperation:
		want := idempotency.ResponseSnapshot{Outcome: "invalidated", ResourceID: s.ResourceID}
		if len(parts) != 3 || !recoveryUUID.MatchString(parts[2]) || !recoveryUUID.MatchString(s.ResourceID) || x.Recovery.ResultCode != "password_reset_invalidated" || !reflect.DeepEqual(s, want) {
			return "", "", recovery.ErrAdmissionBinding
		}
		return parts[1], parts[2], nil
	default:
		return "", "", recovery.ErrAdmissionIncomplete
	}
}
func recoveryFailedValid(x RecoveryFailedEvidence, m recoveryIdempotencyMember, b recovery.RunBinding, v recovery.PermitBinding, actor, ref string, now time.Time) error {
	r := x.Recovery
	t := m.transition()
	if x.Version != recoveryFailedEvidenceVersion || x.EvidenceRef != ref || strings.TrimSpace(ref) == "" || !recoveryHex(x.EvidenceDigest) || x.InstallationID != v.InstallationID || x.RunID != b.RunID || x.BackupRef != b.BackupRef || x.ActorID != actor || actor == "" || x.ApprovedAt.IsZero() || x.ApprovedAt.Location() != time.UTC || !x.ApprovedAt.Equal(x.ApprovedAt.Truncate(time.Microsecond)) || (!now.IsZero() && x.ApprovedAt.After(now)) {
		return recovery.ErrAdmissionBinding
	}
	if r.Action != idempotency.RecoveryFail || r.ReviewedBy != "" || !r.Now.IsZero() || !r.ExpiresAt.IsZero() || !r.LockedUntil.IsZero() || !bytes.Equal(r.ScopeHash, t.ScopeHash) || !bytes.Equal(r.KeyLocator, t.KeyLocator) || !bytes.Equal(r.KeyDigest, t.KeyDigest) || !bytes.Equal(r.RequestDigest, t.RequestDigest) || r.DigestKeyVersion != t.DigestKeyVersion || r.ExecutionGeneration != t.ExecutionGeneration || !bytes.Equal(recoveryScopeHash(x.Scope), t.ScopeHash) {
		return recovery.ErrAdmissionBinding
	}
	_, _, err := recoveryFailedPurpose(x)
	return err
}

// No generic snapshot codec: project only the two existing domain-produced
// safe failure envelopes after exact DTO comparison above. Store.Recover still
// performs its original encoder/DB constraints. This projection verifies bytes
// independently returned by PostgreSQL, including all allowed fields.
func recoveryFailureWire(x RecoveryFailedEvidence) ([]byte, error) {
	if _, _, err := recoveryFailedPurpose(x); err != nil {
		return nil, err
	}
	s := x.Recovery.ResponseSnapshot
	if s.SchemaVersion == "" {
		m := map[string]any{"outcome": s.Outcome}
		if s.ErrorCode != "" {
			m["error_code"] = s.ErrorCode
		}
		if s.ResourceID != "" {
			m["resource_id"] = s.ResourceID
		}
		return json.Marshal(m)
	}
	return json.Marshal(map[string]any{"schema_version": s.SchemaVersion, "http_status": s.HTTPStatus, "result_code": s.ResultCode, "body": map[string]any{"code": s.Body.Code, "message": s.Body.Message, "field_errors": s.Body.FieldErrors}})
}
func recoveryJSONSame(a, b []byte) bool {
	var x, y any
	return json.Unmarshal(a, &x) == nil && json.Unmarshal(b, &y) == nil && reflect.DeepEqual(x, y)
}

type recoveryResolution struct {
	Member         recoveryIdempotencyMember `json:"record"`
	RunID          string                    `json:"run_id"`
	BackupRef      string                    `json:"backup_ref"`
	InstallationID string                    `json:"installation_id"`
	Version        string                    `json:"source_version"`
	Ref            string                    `json:"evidence_ref"`
	Digest         string                    `json:"evidence_digest"`
	Purpose        string                    `json:"purpose"`
	ApprovedAt     time.Time                 `json:"approved_at"`
	Actor          string                    `json:"actor_id"`
	Now            time.Time                 `json:"reviewed_at"`
	PriorUpdatedAt time.Time                 `json:"prior_updated_at"`
	Decision       string                    `json:"decision"`
	ResultCode     string                    `json:"result_code"`
}
type recoveryIdempotencyRecord struct {
	Member              recoveryIdempotencyMember
	Status              idempotency.Status
	ResultCode          *string
	Snapshot            []byte
	LockedUntil         *time.Time
	ReviewedBy          *string
	ReviewedAt          *time.Time
	UpdatedAt           time.Time
	OperationID, UserID *string
}

func recoveryReadIdempotency(ctx context.Context, tx pgx.Tx, m recoveryIdempotencyMember, lock bool) (recoveryIdempotencyRecord, error) {
	x := recoveryIdempotencyRecord{}
	t := m.transition()
	var a, b, c, d []byte
	sql := `SELECT scope_hash,key_locator,key_digest,request_digest,digest_key_version,execution_generation,created_at,expires_at,status,result_code,response_snapshot::text,locked_until,reviewed_by,reviewed_at,updated_at,password_reset_operation_id::text,password_reset_user_id::text FROM idempotency_records WHERE scope_hash=$1 AND key_locator=$2`
	if lock {
		sql += ` FOR UPDATE`
	}
	err := tx.QueryRow(ctx, sql, t.ScopeHash, t.KeyLocator).Scan(&a, &b, &c, &d, &x.Member.DigestKeyVersion, &x.Member.ExecutionGeneration, &x.Member.CreatedAt, &x.Member.ExpiresAt, &x.Status, &x.ResultCode, &x.Snapshot, &x.LockedUntil, &x.ReviewedBy, &x.ReviewedAt, &x.UpdatedAt, &x.OperationID, &x.UserID)
	x.Member.ScopeHash = hex.EncodeToString(a)
	x.Member.KeyLocator = hex.EncodeToString(b)
	x.Member.KeyDigest = hex.EncodeToString(c)
	x.Member.RequestDigest = hex.EncodeToString(d)
	if err != nil {
		return x, err
	}
	x.Member.CreatedAt = x.Member.CreatedAt.UTC()
	x.Member.ExpiresAt = x.Member.ExpiresAt.UTC()
	x.UpdatedAt = x.UpdatedAt.UTC()
	if x.ReviewedAt != nil {
		q := x.ReviewedAt.UTC()
		x.ReviewedAt = &q
	}
	if !reflect.DeepEqual(x.Member, m) {
		return x, recovery.ErrAdmissionBinding
	}
	return x, nil
}
func (e *recoveryIdentityExecution) readFailure(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding, v recovery.PermitBinding, m recoveryIdempotencyMember, actor, ref string, now time.Time) (RecoveryFailedEvidence, error) {
	if nilIdentityDependency(e.d.failedEvidence) {
		return RecoveryFailedEvidence{}, recovery.ErrAdmissionIncomplete
	}
	if err := e.guard(ctx, p, recovery.PhaseRecoveryIdentity); err != nil {
		return RecoveryFailedEvidence{}, err
	}
	t := m.transition()
	x, err := e.d.failedEvidence.ReadFailed(ctx, v, ref, t.ScopeHash, t.KeyLocator, t.ExecutionGeneration)
	if err != nil {
		return RecoveryFailedEvidence{}, err
	}
	// Reader owns/joined all I/O before returning. Copy both inbound lookups and
	// returned data; no returned bytes can later alter an in-progress proof.
	x = recoveryCopyFailed(x)
	if err = e.guard(ctx, p, recovery.PhaseRecoveryIdentity); err != nil {
		return x, err
	}
	if err = recoveryFailedValid(x, m, b, v, actor, ref, now); err != nil {
		return x, err
	}
	return x, nil
}
func recoveryResolutionFor(x RecoveryFailedEvidence, m recoveryIdempotencyMember, now, prior time.Time) recoveryResolution {
	purpose, _, _ := recoveryFailedPurpose(x)
	return recoveryResolution{Member: m, RunID: x.RunID, BackupRef: x.BackupRef, InstallationID: x.InstallationID, Version: x.Version, Ref: x.EvidenceRef, Digest: x.EvidenceDigest, Purpose: purpose, ApprovedAt: x.ApprovedAt, Actor: x.ActorID, Now: now, PriorUpdatedAt: prior, Decision: "failed", ResultCode: x.Recovery.ResultCode}
}
func recoveryReadResolution(ctx context.Context, tx pgx.Tx, b recovery.RunBinding, install string, m recoveryIdempotencyMember) (recoveryResolution, error) {
	raw, ok, err := recoveryExecutionAudit(ctx, tx, recoveryIdempotencyAction, m.target(), b, install)
	if err != nil {
		return recoveryResolution{}, err
	}
	if !ok {
		return recoveryResolution{}, recovery.ErrAdmissionIncomplete
	}
	var a recoveryResolution
	if err = json.Unmarshal(raw, &a); err != nil {
		return a, err
	}
	if a.RunID != b.RunID || a.BackupRef != b.BackupRef || a.InstallationID != install || !reflect.DeepEqual(a.Member, m) || a.Decision != "failed" {
		return a, recovery.ErrAdmissionBinding
	}
	// Metadata alone is not the actual actor/time/target. Bind the immutable
	// Audit event itself and reject duplicate/mismatched events.
	var n int64
	err = tx.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action=$1 AND target_type='idempotency_record' AND target_id=$2 AND actor_kind='user' AND actor_user_id=$3::uuid AND actor_role='admin' AND occurred_at=$4 AND outcome='succeeded' AND metadata=$5::jsonb`, recoveryIdempotencyAction, m.target(), a.Actor, a.Now, string(raw)).Scan(&n)
	if err != nil {
		return a, err
	}
	if n != 1 {
		return a, recovery.ErrAdmissionBinding
	}
	return a, nil
}
func recoveryTerminal(x recoveryIdempotencyRecord, a recoveryResolution, src RecoveryFailedEvidence) error {
	if x.Status != idempotency.StatusFailed || x.ResultCode == nil || *x.ResultCode != src.Recovery.ResultCode || x.ReviewedBy == nil || *x.ReviewedBy != a.Actor || x.ReviewedAt == nil || !x.ReviewedAt.Equal(a.Now) || !x.UpdatedAt.Equal(a.Now) || x.LockedUntil != nil || !reflect.DeepEqual(a, recoveryResolutionFor(src, x.Member, a.Now, a.PriorUpdatedAt)) || a.Now.Before(a.PriorUpdatedAt) || a.Now.Before(x.Member.CreatedAt) || a.Now.Before(src.ApprovedAt) {
		return recovery.ErrAdmissionBinding
	}
	want, err := recoveryFailureWire(src)
	if err != nil {
		return err
	}
	if !recoveryJSONSame(x.Snapshot, want) {
		return recovery.ErrAdmissionBinding
	}
	return nil
}
func (e *recoveryIdentityExecution) appendResolution(ctx context.Context, tx pgx.Tx, a recoveryResolution) error {
	id, err := e.d.ids.NewID()
	if err != nil {
		return err
	}
	trace, ok := platform.TraceIDFromContext(ctx)
	if !ok {
		return recovery.ErrAdmissionBinding
	}
	raw, err := json.Marshal(a)
	if err != nil {
		return err
	}
	var metadata map[string]any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if err = d.Decode(&metadata); err != nil {
		return err
	}
	role := "admin"
	return e.d.audit.Append(ctx, tx, audit.Event{ID: id, OccurredAt: a.Now, ActorKind: audit.ActorUser, ActorUserID: &a.Actor, ActorRole: &role, Action: recoveryIdempotencyAction, TargetType: "idempotency_record", TargetID: a.Member.target(), Outcome: "succeeded", TraceID: trace, Metadata: metadata})
}

// Linked durable work is never settled here. Reuse the original domain's Gate,
// authority/delivery/credential and replay classifiers after reading its exact
// aggregate. Unknown/hold/dead-letter/nonterminal histories remain closed.
func (e *recoveryIdentityExecution) failedDomain(ctx context.Context, tx pgx.Tx, r recoveryIdempotencyRecord, src RecoveryFailedEvidence, lock bool) error {
	purpose, target, err := recoveryFailedPurpose(src)
	if err != nil {
		return err
	}
	t := r.Member.transition()
	if purpose != passwordResetOperation {
		var n int64
		err = tx.QueryRow(ctx, `SELECT count(*) FROM password_reset_operations WHERE scope_hash=$1 AND key_locator=$2`, t.ScopeHash, t.KeyLocator).Scan(&n)
		if err != nil {
			return err
		}
		if n != 0 || r.OperationID != nil || r.UserID != nil {
			return recovery.ErrAdmissionBinding
		}
		return nil
	}
	if r.OperationID == nil || r.UserID == nil || *r.UserID != target {
		return recovery.ErrAdmissionBinding
	}
	// The write path locks durable domain rows BEFORE the claim; Complete and
	// terminal retry only observe them and never acquire a reverse User lock.
	if lock {
		var id string
		if err = tx.QueryRow(ctx, `SELECT operation_id::text FROM password_reset_operations WHERE operation_id=$1 FOR UPDATE`, *r.OperationID).Scan(&id); err != nil {
			return err
		}
		if err = tx.QueryRow(ctx, `SELECT c.id::text FROM temporary_credentials c JOIN password_reset_operations o ON o.temporary_credential_id=c.id WHERE o.operation_id=$1 FOR UPDATE OF c`, *r.OperationID).Scan(&id); err != nil {
			return err
		}
		if err = tx.QueryRow(ctx, `SELECT operation_id::text FROM password_reset_recovery_jobs WHERE operation_id=$1 FOR UPDATE`, *r.OperationID).Scan(&id); err != nil {
			return err
		}
	}
	o, err := recoveryReadResetOperation(ctx, tx, *r.OperationID)
	if err != nil {
		return err
	}
	if o.UserID != target || o.RequestKind != PasswordResetInitialReset || o.PredecessorOperationID != nil || o.ReplacedCredentialID != nil || !bytes.Equal(o.ScopeHash, t.ScopeHash) || !bytes.Equal(o.KeyLocator, t.KeyLocator) || !bytes.Equal(o.KeyDigest, t.KeyDigest) || !bytes.Equal(o.RequestDigest, t.RequestDigest) || o.DigestKeyVersion != t.DigestKeyVersion || o.ExecutionGeneration != t.ExecutionGeneration || !passwordResetOperationAuthorityValid(o) || !passwordResetDeliveryEvidenceValid(o) || (o.State != PasswordResetOperationFailed && o.State != PasswordResetOperationInvalidated) || o.SettlementState != PasswordResetSettlementInvalidated {
		return recovery.ErrAdmissionIncomplete
	}
	if o.TerminalAt == nil || o.SettledAt == nil || src.ApprovedAt.Before(*o.TerminalAt) || src.ApprovedAt.Before(o.UpdatedAt) {
		return recovery.ErrAdmissionBinding
	}
	if err = (PasswordResetStore{}).CheckLockedUser(ctx, tx, target); err != nil {
		return err
	}
	var c TemporaryCredential
	err = tx.QueryRow(ctx, `SELECT id::text,user_id::text,purpose,auth_generation,source_operation_id::text,state FROM temporary_credentials WHERE id=$1`, o.TemporaryCredentialID).Scan(&c.ID, &c.UserID, &c.Purpose, &c.AuthGeneration, &c.SourceOperationID, &c.State)
	if err != nil {
		return err
	}
	if !passwordResetResendCredentialExact(o, c) || c.State != TemporaryCredentialRecoveryInvalidated {
		return recovery.ErrAdmissionIncomplete
	}
	if err = recoveryFailedResetSettlement(ctx, tx, o); err != nil {
		return err
	}
	raw, err := recoveryFailureWire(src)
	if err != nil {
		return err
	}
	body := string(raw)
	code := src.Recovery.ResultCode
	authority := passwordResetSettlementAuthority{idempotencyStatus: "failed", resultCode: &code, responseSnapshot: &body}
	if !passwordResetIdempotencyTerminal(authority, o.ID, false) {
		return recovery.ErrAdmissionBinding
	}
	// Actual original classifier owns the stable failed replay. A successful
	// or indeterminate operation cannot be converted into a generic failure.
	_, replay := classifyPasswordResetReplay(passwordResetPrepared{operation: o, target: User{ID: target}})
	if replay == nil {
		return recovery.ErrAdmissionBinding
	}
	if platform.PublicCode(replay) != platform.CodeIdentityEmailDeliveryFailed {
		return recovery.ErrAdmissionIncomplete
	}
	return nil
}

// This helper is private consumer orchestration for the unchanged failed-only
// body. Root owns maintenance Session/CSRF/HTTP and constructs sealed dependencies;
// it cannot supply actor/time/result or treat a binding as a capability.
func (e *recoveryIdentityExecution) resolveIdempotency(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding, scope, locator []byte, generation int64, decision, ref string) error {
	pc, v, err := e.phase(ctx, p, b, recovery.PhaseRecoveryIdentity)
	if err != nil {
		return err
	}
	if len(scope) != 32 || len(locator) != 32 || generation <= 0 || decision != "failed" || strings.TrimSpace(ref) == "" || nilIdentityDependency(e.d.failedEvidence) {
		return recovery.ErrAdmissionBinding
	}
	scope = append([]byte(nil), scope...)
	locator = append([]byte(nil), locator...)
	state, err := e.idempotencyPreflight(pc, p, b, v, scope, locator, generation)
	if err != nil {
		return err
	}
	src, err := e.readFailure(pc, p, b, v, state.record.Member, state.initial, ref, time.Time{})
	if err != nil {
		return err
	}
	// Source I/O is already joined. Take the consumer clock exactly once now;
	// no source-provided clock or old execution timestamp can replace it.
	now := e.d.clock.Now().UTC().Truncate(time.Microsecond)
	if err = recoveryFailedValid(src, state.record.Member, b, v, state.initial, ref, now); err != nil {
		return err
	}
	purpose, target, err := recoveryFailedPurpose(src)
	if err != nil {
		return err
	}
	_ = purpose
	if state.record.Status == idempotency.StatusFailed {
		return e.terminalRetry(pc, p, b, v, state.record.Member, src, now)
	}
	if state.record.Status != idempotency.StatusRecoveryReviewRequired || state.record.LockedUntil != nil || state.record.ResultCode != nil || len(state.record.Snapshot) != 0 {
		return recovery.ErrAdmissionIncomplete
	}
	// The external source has returned and joined before acquiring ANY new DB
	// transaction. Writer locks actor/target Users, Store, then original record.
	return e.writeVerified(pc, p, recovery.PhaseRecoveryIdentity, recoveryExecutionScope{action: recoveryIdempotencyAction, binding: b, installation: v.InstallationID, actor: state.initial, target: target, member: &state.record.Member}, func(tx pgx.Tx) error {
		ids := []string{state.initial}
		if target != "" && target != state.initial {
			ids = append(ids, target)
		}
		users, err := recoveryExecutionUsers(pc, tx, []string{state.initial}, true)
		if err != nil {
			return err
		}
		if len(users) != 1 {
			return recovery.ErrAdmissionIncomplete
		}
		if len(ids) > 1 {
			targetUsers, er := recoveryExecutionUsers(pc, tx, []string{target}, true)
			if er != nil {
				return er
			}
			if len(targetUsers) != 1 {
				return recovery.ErrAdmissionIncomplete
			}
			users = append(users, targetUsers...)
		}
		r, _, err := e.lockedRun(pc, tx, b, v, recovery.RecoveryOutcomeRunning)
		if err != nil {
			return err
		}
		initial, err := recoveryExecutionInitial(users)
		if err != nil || initial.ID != state.initial {
			return recovery.ErrAdmissionBinding
		}
		if err = e.changedInitial(pc, tx, r, initial); err != nil {
			return err
		}
		if err = e.failedDomain(pc, tx, state.record, src, true); err != nil {
			return err
		}
		x, err := recoveryReadIdempotency(pc, tx, state.record.Member, true)
		if err != nil {
			return err
		}
		if x.Status == idempotency.StatusFailed {
			a, er := recoveryReadResolution(pc, tx, b, v.InstallationID, x.Member)
			if er != nil {
				return er
			}
			return recoveryTerminal(x, a, src)
		}
		if !reflect.DeepEqual(x, state.record) {
			return recovery.ErrAdmissionBinding
		}
		if now.Before(x.UpdatedAt) || now.Before(x.Member.CreatedAt) || now.Before(r.StartedAt) || r.InitialAdminChangedAt == nil || now.Before(*r.InitialAdminChangedAt) {
			return recovery.ErrAdmissionBinding
		}
		if err = e.failedDomain(pc, tx, x, src, false); err != nil {
			return err
		}
		if err = e.guard(pc, p, recovery.PhaseRecoveryIdentity); err != nil {
			return err
		}
		write := recoveryCopyFailed(src).Recovery
		write.ReviewedBy = initial.ID
		write.Now = now
		write.Action = idempotency.RecoveryFail
		if err = (idempotency.Store{}).Recover(pc, tx, write); err != nil {
			return err
		}
		if err = e.guard(pc, p, recovery.PhaseRecoveryIdentity); err != nil {
			return err
		}
		return e.appendResolution(pc, tx, recoveryResolutionFor(src, x.Member, now, x.UpdatedAt))
	}, func() error {
		// Both normal ACK and lost ACK must re-read the immutable source after
		// the writer transaction ends, before the independent physical RR read.
		after, err := e.readFailure(pc, p, b, v, state.record.Member, state.initial, ref, now)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(after, src) {
			return recovery.ErrAdmissionBinding
		}
		return nil
	})
}

type recoveryIdempotencyPreflight struct {
	record  recoveryIdempotencyRecord
	initial string
}

func (e *recoveryIdentityExecution) idempotencyPreflight(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding, v recovery.PermitBinding, scope, locator []byte, gen int64) (out recoveryIdempotencyPreflight, result error) {
	begin, err := e.begun(ctx, b)
	if err != nil {
		return out, err
	}
	snapshot, err := e.d.store.ReadAdmissionSnapshot(ctx)
	if err != nil {
		return out, err
	}
	var initial string
	for _, r := range snapshot.Runs {
		if r.RunID == b.RunID && r.InitialAdminUserID != nil {
			initial = *r.InitialAdminUserID
		}
	}
	if initial == "" {
		return out, recovery.ErrAdmissionIncomplete
	}
	tx, err := e.d.store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return out, err
	}
	defer func() { result = errors.Join(result, finishAdmissionRead(ctx, tx)) }()
	users, err := recoveryExecutionUsers(ctx, tx, []string{initial}, true)
	if err != nil {
		return out, err
	}
	if len(users) != 1 {
		return out, recovery.ErrAdmissionIncomplete
	}
	r, _, err := e.lockedRun(ctx, tx, b, v, recovery.RecoveryOutcomeRunning)
	if err != nil {
		return out, err
	}
	if err = recoveryExecutionMatchBegin(r, begin); err != nil {
		return out, err
	}
	if err = e.changedInitial(ctx, tx, r, users[0]); err != nil {
		return out, err
	}
	f, err := e.finalization(ctx, tx, r, b, v.InstallationID)
	if err != nil {
		return out, err
	}
	var member *recoveryIdempotencyMember
	for i := range f.Cohort {
		m := &f.Cohort[i]
		if m.ScopeHash == hex.EncodeToString(scope) && m.KeyLocator == hex.EncodeToString(locator) && m.ExecutionGeneration == gen {
			member = m
		}
	}
	if member == nil {
		return out, recovery.ErrAdmissionBinding
	}
	out.record, err = recoveryReadIdempotency(ctx, tx, *member, false)
	out.initial = initial
	if err != nil {
		return out, err
	}
	if err = e.guard(ctx, p, recovery.PhaseRecoveryIdentity); err != nil {
		return out, err
	}
	return out, tx.Rollback(ctx)
}
func (e *recoveryIdentityExecution) terminalRetry(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding, v recovery.PermitBinding, m recoveryIdempotencyMember, src RecoveryFailedEvidence, now time.Time) (result error) {
	after, err := e.readFailure(ctx, p, b, v, m, src.ActorID, src.EvidenceRef, now)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(src, after) {
		return recovery.ErrAdmissionBinding
	}
	tx, err := e.d.store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, finishAdmissionRead(ctx, tx)) }()
	s, err := readAdmissionTransaction(ctx, tx)
	if err != nil {
		return err
	}
	var run recovery.AdmissionRun
	for _, r := range s.Runs {
		if r.RunID == b.RunID {
			run = r
		}
	}
	if run.Outcome != recovery.RecoveryOutcomeRunning || s.Installation.InstallationID != v.InstallationID {
		return recovery.ErrAdmissionBinding
	}
	users, err := recoveryExecutionUsers(ctx, tx, nil, false)
	if err != nil {
		return err
	}
	initial, err := recoveryExecutionInitial(users)
	if err != nil {
		return err
	}
	if initial.ID != src.ActorID {
		return recovery.ErrAdmissionBinding
	}
	if err = e.changedInitial(ctx, tx, run, initial); err != nil {
		return err
	}
	f, err := e.finalization(ctx, tx, run, b, v.InstallationID)
	if err != nil {
		return err
	}
	found := false
	for _, item := range f.Cohort {
		if reflect.DeepEqual(item, m) {
			found = true
		}
	}
	if !found {
		return recovery.ErrAdmissionBinding
	}
	x, err := recoveryReadIdempotency(ctx, tx, m, false)
	if err != nil {
		return err
	}
	a, err := recoveryReadResolution(ctx, tx, b, v.InstallationID, m)
	if err != nil {
		return err
	}
	if now.Before(x.UpdatedAt) || now.Before(run.StartedAt) || run.InitialAdminChangedAt == nil || now.Before(*run.InitialAdminChangedAt) {
		return recovery.ErrAdmissionBinding
	}
	if err = recoveryTerminal(x, a, src); err != nil {
		return err
	}
	if err = e.failedDomain(ctx, tx, x, src, false); err != nil {
		return err
	}
	if err = e.guard(ctx, p, recovery.PhaseRecoveryIdentity); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type recoveryCohortItemProof struct {
	record     recoveryIdempotencyRecord
	resolution recoveryResolution
	source     RecoveryFailedEvidence
}
type recoveryCohortProof struct {
	image     []byte
	items     []recoveryCohortItemProof
	notBefore time.Time
}

func (e *recoveryIdentityExecution) prepareCohortProof(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding, v recovery.PermitBinding) (out *recoveryCohortProof, result error) {
	out = &recoveryCohortProof{items: []recoveryCohortItemProof{}}
	tx, err := e.d.store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	// Explicitly finish this RR observation before calling the external reader.
	func() {
		defer func() { result = errors.Join(result, finishAdmissionRead(ctx, tx)) }()
		s, er := readAdmissionTransaction(ctx, tx)
		if er != nil {
			result = er
			return
		}
		var r recovery.AdmissionRun
		for _, run := range s.Runs {
			if run.RunID == b.RunID {
				r = run
			}
		}
		if r.RunID == "" || r.BackupRef != b.BackupRef || s.Installation.InstallationID != v.InstallationID || r.InitialAdminUserID == nil {
			result = recovery.ErrAdmissionBinding
			return
		}
		users, er := recoveryExecutionUsers(ctx, tx, nil, false)
		if er != nil {
			result = er
			return
		}
		initial, er := recoveryExecutionInitial(users)
		if er != nil {
			result = er
			return
		}
		if er = e.changedInitial(ctx, tx, r, initial); er != nil {
			result = er
			return
		}
		out.notBefore = r.StartedAt
		if r.InitialAdminChangedAt != nil && r.InitialAdminChangedAt.After(out.notBefore) {
			out.notBefore = *r.InitialAdminChangedAt
		}
		f, er := e.finalization(ctx, tx, r, b, v.InstallationID)
		if er != nil {
			result = er
			return
		}
		for _, m := range f.Cohort {
			x, er := recoveryReadIdempotency(ctx, tx, m, false)
			if er != nil {
				result = er
				return
			}
			a, er := recoveryReadResolution(ctx, tx, b, v.InstallationID, m)
			if er != nil {
				result = er
				return
			}
			if a.Actor != *r.InitialAdminUserID {
				result = recovery.ErrAdmissionBinding
				return
			}
			out.items = append(out.items, recoveryCohortItemProof{record: x, resolution: a})
		}
		out.image, result = recoveryExecutionImage(ctx, tx, recoveryExecutionScope{action: recoveryCompleteObservation, binding: b, installation: v.InstallationID})
		if result == nil {
			result = tx.Commit(ctx)
		}
	}()
	if result != nil {
		return nil, result
	}
	for i := range out.items {
		q := &out.items[i]
		q.source, result = e.readFailure(ctx, p, b, v, q.record.Member, q.resolution.Actor, q.resolution.Ref, time.Time{})
		if result != nil {
			return nil, result
		}
	}
	now := e.d.clock.Now().UTC().Truncate(time.Microsecond)
	if now.Before(out.notBefore) {
		return nil, recovery.ErrAdmissionBinding
	}
	for _, q := range out.items {
		if now.Before(q.record.UpdatedAt) {
			return nil, recovery.ErrAdmissionBinding
		}
		if result = recoveryFailedValid(q.source, q.record.Member, b, v, q.resolution.Actor, q.resolution.Ref, now); result != nil {
			return nil, result
		}
		if result = recoveryTerminal(q.record, q.resolution, q.source); result != nil {
			return nil, result
		}
	}
	if err = e.guard(ctx, p, recovery.PhaseRecoveryIdentity); err != nil {
		return nil, err
	}
	return out, nil
}
func (e *recoveryIdentityExecution) checkCohortProof(ctx context.Context, tx pgx.Tx, b recovery.RunBinding, v recovery.PermitBinding, r recovery.AdmissionRun, f recoveryExecutionFinalization, proof *recoveryCohortProof) error {
	if len(proof.items) != len(f.Cohort) {
		return recovery.ErrAdmissionBinding
	}
	image, err := recoveryExecutionImage(ctx, tx, recoveryExecutionScope{action: recoveryCompleteObservation, binding: b, installation: v.InstallationID})
	if err != nil {
		return err
	}
	if !bytes.Equal(image, proof.image) {
		return recovery.ErrAdmissionBinding
	}
	for i, m := range f.Cohort {
		q := proof.items[i]
		if !reflect.DeepEqual(q.record.Member, m) {
			return recovery.ErrAdmissionBinding
		}
		x, err := recoveryReadIdempotency(ctx, tx, m, false)
		if err != nil {
			return err
		}
		a, err := recoveryReadResolution(ctx, tx, b, v.InstallationID, m)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(x, q.record) || !reflect.DeepEqual(a, q.resolution) || r.InitialAdminUserID == nil || a.Actor != *r.InitialAdminUserID || a.Now.Before(r.StartedAt) || r.InitialAdminChangedAt == nil || a.Now.Before(*r.InitialAdminChangedAt) {
			return recovery.ErrAdmissionBinding
		}
		if err = recoveryTerminal(x, a, q.source); err != nil {
			return err
		}
		if err = e.failedDomain(ctx, tx, x, q.source, false); err != nil {
			return err
		}
	}
	return nil
}

// Only the NEW cohort/resolution metadata uses this decimal-string projection;
// existing audit count fields and the existing Store/JSON contracts are intact.
type recoveryMemberWire struct {
	ScopeHash        string    `json:"scope_hash"`
	KeyLocator       string    `json:"key_locator"`
	KeyDigest        string    `json:"key_digest"`
	RequestDigest    string    `json:"request_digest"`
	DigestKeyVersion string    `json:"digest_key_version"`
	Generation       string    `json:"execution_generation"`
	CreatedAt        time.Time `json:"created_at"`
	ExpiresAt        time.Time `json:"expires_at"`
}

func (m recoveryIdempotencyMember) MarshalJSON() ([]byte, error) {
	return json.Marshal(recoveryMemberWire{ScopeHash: m.ScopeHash, KeyLocator: m.KeyLocator, KeyDigest: m.KeyDigest, RequestDigest: m.RequestDigest, DigestKeyVersion: m.DigestKeyVersion, Generation: strconv.FormatInt(m.ExecutionGeneration, 10), CreatedAt: m.CreatedAt, ExpiresAt: m.ExpiresAt})
}
func (m *recoveryIdempotencyMember) UnmarshalJSON(raw []byte) error {
	if m == nil {
		return recovery.ErrAdmissionBinding
	}
	var w recoveryMemberWire
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&w); err != nil {
		return err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return recovery.ErrAdmissionBinding
	}
	n, err := strconv.ParseInt(w.Generation, 10, 64)
	if err != nil || n <= 0 || strconv.FormatInt(n, 10) != w.Generation {
		return recovery.ErrAdmissionBinding
	}
	*m = recoveryIdempotencyMember{ScopeHash: w.ScopeHash, KeyLocator: w.KeyLocator, KeyDigest: w.KeyDigest, RequestDigest: w.RequestDigest, DigestKeyVersion: w.DigestKeyVersion, ExecutionGeneration: n, CreatedAt: w.CreatedAt, ExpiresAt: w.ExpiresAt}
	return recoveryCohortValid([]recoveryIdempotencyMember{*m}, 1)
}

// The ACK image is an observation of this existing action's scope, never a new
// admission digest/authority. The original S49 reader still validates/recomputes
// the complete Store projection; unrelated runs/global revision changes do not
// become an assertion that this action's own bound facts changed. Runtime owns
// current raw permit/exclusive/phase checks and the later full opening proof.
const recoveryCompleteObservation = "identity.recovery.complete.observation"

type recoveryExecutionScope struct {
	action                      string
	binding                     recovery.RunBinding
	installation, actor, target string
	canonical                   bool
	member                      *recoveryIdempotencyMember
}

func recoveryAckRows(ctx context.Context, tx pgx.Tx, selectSQL string, args ...any) (json.RawMessage, error) {
	var raw []byte
	err := tx.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(x ORDER BY x::text),'[]'::jsonb)::text FROM (`+selectSQL+`) selected(x)`, args...).Scan(&raw)
	return json.RawMessage(raw), err
}
func recoveryExecutionImage(ctx context.Context, tx pgx.Tx, q recoveryExecutionScope) ([]byte, error) {
	switch q.action {
	case recoveryBindAction, recoveryFinalizeAction, recoveryTrustAction, recoveryReviewAction, recoveryCompleteObservation, recoveryExitAction, recoveryIdempotencyAction:
	default:
		return nil, recovery.ErrAdmissionBinding
	}
	s, err := readAdmissionTransaction(ctx, tx)
	if err != nil {
		return nil, err
	}
	var r recovery.AdmissionRun
	for _, x := range s.Runs {
		if x.RunID == q.binding.RunID {
			r = x
		}
	}
	if r.RunID == "" || r.BackupRef != q.binding.BackupRef || s.Installation.InstallationID != q.installation || r.RestoreBinding == nil {
		return nil, recovery.ErrAdmissionBinding
	}
	image := map[string]json.RawMessage{}
	base, err := json.Marshal(struct {
		Installation recovery.InstallationBinding
		Run          recovery.AdmissionRun
	}{s.Installation, r})
	if err != nil {
		return nil, err
	}
	image["bound_store"] = base
	// Aggregate-source actions retain the existing complete canonical revision
	// (counter AND digest) fence. Narrow maintenance ACKs prove their own exact
	// result and recheck current Runtime authority; they do not claim a global
	// revision is unchanged when another authorized target review has committed.
	if q.action != recoveryTrustAction && q.action != recoveryReviewAction && q.action != recoveryIdempotencyAction {
		revision, er := json.Marshal(s.Revision)
		if er != nil {
			return nil, er
		}
		image["admission_revision"] = revision
	}
	// No per-table whole-database collector. These fixed SQL selections retain
	// every row in a required set, including nil/empty, full six-field identity,
	// paired result/review timestamps, domain lineage and exact Audit events.
	add := func(name, sql string, args ...any) error {
		raw, err := recoveryAckRows(ctx, tx, sql, args...)
		if err == nil {
			image[name] = raw
		}
		return err
	}
	ids := []string{}
	cohort := []recoveryIdempotencyMember{}
	auditTargets := []string{}
	sourceIDs := []string{}
	var f recoveryExecutionFinalization
	if r.AuditEventID != nil {
		raw, ok, err := recoveryExecutionAudit(ctx, tx, recoveryFinalizeAction, r.RunID, q.binding, q.installation)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, recovery.ErrAdmissionIncomplete
		}
		if err = json.Unmarshal(raw, &f); err != nil {
			return nil, err
		}
		if err = recoveryCohortValid(f.Cohort, f.Idempotency); err != nil {
			return nil, err
		}
		if f.CohortCount != strconv.FormatInt(f.Idempotency, 10) {
			return nil, recovery.ErrAdmissionBinding
		}
		for _, u := range f.Sources {
			sourceIDs = append(sourceIDs, u.ID)
		}
		// Original membership/source is necessary even for narrow actions; it
		// contains IDs/backup facts, not unrelated credentials or result bodies.
		if err = add("original_user_membership", `SELECT jsonb_build_array(id::text,is_initial_admin) FROM users ORDER BY id`); err != nil {
			return nil, err
		}
		if err = add("original_review_sources", `SELECT jsonb_build_array(user_id::text,backup_role,backup_status,backup_deleted_at) FROM recovery_account_reviews WHERE recovery_run_id=$1 ORDER BY user_id`, r.RunID); err != nil {
			return nil, err
		}
	}
	full := q.action == recoveryFinalizeAction || q.action == recoveryCompleteObservation
	if full {
		ids = append(ids, sourceIDs...)
		cohort = append(cohort, f.Cohort...)
	} else {
		if q.actor != "" {
			ids = append(ids, q.actor)
		}
		if q.target != "" && q.target != q.actor {
			ids = append(ids, q.target)
		}
		if q.action == recoveryExitAction {
			ids, err = recoveryExecutionExitIDs(ctx, tx, r.RunID)
			if err != nil {
				return nil, err
			}
			if r.InitialAdminUserID != nil {
				ids = append(ids, *r.InitialAdminUserID)
			}
		}
		if q.member != nil {
			cohort = append(cohort, *q.member)
		}
	}
	sort.Strings(ids)
	unique := []string{}
	for _, id := range ids {
		if len(unique) == 0 || unique[len(unique)-1] != id {
			unique = append(unique, id)
		}
	}
	ids = unique
	for _, m := range cohort {
		auditTargets = append(auditTargets, m.target())
	}
	if len(ids) > 0 {
		if err = add("users", `SELECT to_jsonb(u) FROM users u WHERE id=ANY($1::uuid[])`, ids); err != nil {
			return nil, err
		}
		if full {
			if err = add("identifiers", `SELECT to_jsonb(i) FROM identity_identifiers i WHERE user_id=ANY($1::uuid[])`, ids); err != nil {
				return nil, err
			}
		}
		if q.action == recoveryReviewAction {
			if err = add("identifiers", `SELECT to_jsonb(i) FROM identity_identifiers i WHERE user_id=$1`, q.target); err != nil {
				return nil, err
			}
		}
		if full || q.action == recoveryTrustAction || q.action == recoveryExitAction {
			if err = add("sessions", `SELECT to_jsonb(x) FROM sessions x WHERE user_id=ANY($1::uuid[])`, ids); err != nil {
				return nil, err
			}
		}
		if q.action == recoveryReviewAction {
			if err = add("sessions", `SELECT to_jsonb(x) FROM sessions x WHERE user_id=$1`, q.target); err != nil {
				return nil, err
			}
		}
		if err = add("credentials", `SELECT to_jsonb(c) FROM temporary_credentials c WHERE user_id=ANY($1::uuid[])`, ids); err != nil {
			return nil, err
		}
		if err = add("operations", `SELECT to_jsonb(o) FROM password_reset_operations o WHERE user_id=ANY($1::uuid[])`, ids); err != nil {
			return nil, err
		}
		if err = add("jobs", `SELECT to_jsonb(j) FROM password_reset_recovery_jobs j JOIN password_reset_operations o ON o.operation_id=j.operation_id WHERE o.user_id=ANY($1::uuid[])`, ids); err != nil {
			return nil, err
		}
	}
	scopes := []string{}
	locators := []string{}
	for _, m := range cohort {
		scopes = append(scopes, m.ScopeHash)
		locators = append(locators, m.KeyLocator)
	}
	if len(cohort) > 0 || len(ids) > 0 {
		if err = add("claims", `SELECT to_jsonb(i) FROM idempotency_records i WHERE EXISTS(SELECT 1 FROM unnest($1::text[],$2::text[]) original(scope_hash,key_locator) WHERE i.scope_hash=decode(original.scope_hash,'hex') AND i.key_locator=decode(original.key_locator,'hex')) OR i.password_reset_operation_id IN(SELECT operation_id FROM password_reset_operations WHERE user_id=ANY($3::uuid[]))`, scopes, locators, ids); err != nil {
			return nil, err
		}
	}
	if full {
		if err = add("reviews", `SELECT to_jsonb(v) FROM recovery_account_reviews v WHERE recovery_run_id=$1`, r.RunID); err != nil {
			return nil, err
		}
	} else if q.action == recoveryReviewAction {
		if err = add("reviews", `SELECT to_jsonb(v) FROM recovery_account_reviews v WHERE recovery_run_id=$1 AND user_id=$2`, r.RunID, q.target); err != nil {
			return nil, err
		}
	}
	if q.action == recoveryExitAction {
		// All same-run maintenance generations, including revoked/expired rows
		// and wrong-install residuals, remain in the exact Exit collection.
		if err = add("maintenance_sessions", `SELECT to_jsonb(x) FROM sessions x WHERE access_scope='recovery_maintenance' AND recovery_run_id=$1`, r.RunID); err != nil {
			return nil, err
		}
	}
	if q.action == recoveryCompleteObservation {
		// Complete's original no-unresolved precondition remains global. Keep
		// actual full unresolved rows, not a count; unrelated terminal history
		// cannot substitute for missing original cohort members.
		if err = add("unresolved_claims", `SELECT to_jsonb(i) FROM idempotency_records i WHERE status IN('processing','uncertain','recovery_review_required')`); err != nil {
			return nil, err
		}
		if err = add("unresolved_operations", `SELECT to_jsonb(o) FROM password_reset_operations o WHERE state NOT IN('succeeded','failed','invalidated')`); err != nil {
			return nil, err
		}
	}
	canonicalIDs := []string{}
	if q.canonical {
		canonicalIDs = append(canonicalIDs, q.target)
	}
	if q.action == recoveryCompleteObservation {
		rows, er := tx.Query(ctx, `SELECT user_id::text FROM recovery_account_reviews WHERE recovery_run_id=$1 AND decision='kept_deleted' ORDER BY user_id`, r.RunID)
		if er != nil {
			return nil, er
		}
		for rows.Next() {
			var id string
			if er = rows.Scan(&id); er != nil {
				rows.Close()
				return nil, er
			}
			canonicalIDs = append(canonicalIDs, id)
		}
		er = rows.Err()
		rows.Close()
		if er != nil {
			return nil, er
		}
	}
	if len(canonicalIDs) > 0 {
		// Body/title are not an ACK proof of canonical orphan/deletion state.
		// Keep every related row and its original authority/version/link fields.
		if err = add("articles", `SELECT to_jsonb(a)-'body'-'title' FROM articles a WHERE author_id=ANY($1::uuid[]) OR orphaned_from_user_id=ANY($1::uuid[])`, canonicalIDs); err != nil {
			return nil, err
		}
		if err = add("deletions", `SELECT to_jsonb(d) FROM deletion_operations d WHERE orphaned_from_user_id=ANY($1::uuid[]) OR actor_user_id=ANY($1::uuid[]) OR article_id IN(SELECT id FROM articles WHERE author_id=ANY($1::uuid[]) OR orphaned_from_user_id=ANY($1::uuid[]))`, canonicalIDs); err != nil {
			return nil, err
		}
		if err = add("notices", `SELECT to_jsonb(n) FROM transfer_notices n WHERE former_deleted_user_id=ANY($1::uuid[]) OR new_author_id=ANY($1::uuid[])`, canonicalIDs); err != nil {
			return nil, err
		}
		if err = add("recipients", `SELECT to_jsonb(v) FROM transfer_notice_recipients v JOIN transfer_notices n ON n.id=v.notice_id WHERE n.former_deleted_user_id=ANY($1::uuid[]) OR n.new_author_id=ANY($1::uuid[])`, canonicalIDs); err != nil {
			return nil, err
		}
	}
	actionTarget := r.RunID
	if q.action == recoveryReviewAction {
		actionTarget = q.target
	}
	if q.member != nil {
		actionTarget = q.member.target()
	}
	// Select complete event rows (including mismatched/duplicate variants), not
	// all events emitted by the same actor or an unrelated target's audit.
	if err = add("audit", `SELECT to_jsonb(a) FROM audit_events a WHERE (a.action=$1 AND a.target_id=$2) OR a.id=$3::uuid OR a.id=$4::uuid OR (a.metadata->>'run_id'=$5 AND a.target_id=$5 AND a.action IN('identity.recovery.bind','identity.recovery.trust.prepare')) OR (a.action='identity.recovery.idempotency.resolve' AND a.target_id=ANY($6::text[])) OR ($7 AND a.metadata->>'run_id'=$5 AND a.action='identity.recovery.review') OR (a.target_id=ANY($8::text[]) AND a.action=$9) OR (a.action='article.orphan.canonical' AND a.metadata->>'deleted_user_id'=ANY($8::text[])) OR (a.target_id IN(SELECT operation_id::text FROM password_reset_operations WHERE user_id=ANY($10::uuid[])) AND a.action='identity.password_reset.recovered')`, q.action, actionTarget, r.AuditEventID, r.InitialAdminChangeAuditID, r.RunID, auditTargets, full, canonicalIDs, canonicalDeleteAuditAction, ids); err != nil {
		return nil, err
	}
	return json.Marshal(image)
}
