//go:build integration

package recovery_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"han/internal/article"
	"han/internal/audit"
	"han/internal/platform"
	"han/internal/recovery"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestArticleReplayEvidencePostgres covers three bounded SQL risks.
// Inventories and foreign evidence are synthetic fixtures; the Store, Source,
// Finalizer, facts, public Provider and strict decoder are real implementations.
func TestArticleReplayEvidencePostgres(t *testing.T) {
	t.Run("DifferentBackupFirstBinding", s44EvidenceBinding)
	t.Run("PublishRollback", s44EvidenceRollback)
	t.Run("CurrentUpdateRollback", s44CurrentUpdateRollback)
}

func s44EvidenceID(t *testing.T) string {
	t.Helper()
	value, err := platform.NewTraceID()
	s44Must(t, err)
	return value
}

type s44EvidenceInsert struct {
	PID               uint32
	Rows              int64
	TransactionStatus byte
	Err               error
}

type s44EvidenceQueryKey struct{}

// s44EvidenceTracer only observes real SQL and pauses its first successful
// run INSERT before Store.Begin can advance the owner or commit. No SQL/error,
// request context trace, execution outcome or transaction result is rewritten.
type s44EvidenceTracer struct {
	first       sync.Once
	releaseOnce sync.Once
	started     chan s44EvidenceInsert
	inserted    chan s44EvidenceInsert
	release     chan struct{}
	hold        bool
}

func s44EvidenceNewTracer(hold bool) *s44EvidenceTracer {
	return &s44EvidenceTracer{started: make(chan s44EvidenceInsert, 1),
		inserted: make(chan s44EvidenceInsert, 1), release: make(chan struct{}), hold: hold}
}

func (tracer *s44EvidenceTracer) open() {
	tracer.releaseOnce.Do(func() { close(tracer.release) })
}

func (tracer *s44EvidenceTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	if !strings.HasPrefix(strings.TrimSpace(data.SQL), "INSERT INTO deletion_replay_runs") {
		return ctx
	}
	first := false
	tracer.first.Do(func() { first = true })
	if !first {
		return ctx
	}
	observation := s44EvidenceInsert{PID: conn.PgConn().PID(), TransactionStatus: conn.PgConn().TxStatus()}
	select {
	case tracer.started <- observation:
	case <-ctx.Done():
		return ctx
	}
	return context.WithValue(ctx, s44EvidenceQueryKey{}, true)
}

func (tracer *s44EvidenceTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	if first, _ := ctx.Value(s44EvidenceQueryKey{}).(bool); !first {
		return
	}
	observation := s44EvidenceInsert{PID: conn.PgConn().PID(), Rows: data.CommandTag.RowsAffected(),
		TransactionStatus: conn.PgConn().TxStatus(), Err: data.Err}
	select {
	case tracer.inserted <- observation:
	case <-ctx.Done():
		return
	}
	if tracer.hold {
		select {
		case <-tracer.release:
		case <-ctx.Done():
		}
	}
}

type s44EvidencePeer struct {
	store  *article.PostgresReplayEvidenceStore
	pool   *pgxpool.Pool
	tracer *s44EvidenceTracer
	pid    uint32
}

func s44EvidenceNewPeer(f *s44Fixture, hold bool) *s44EvidencePeer {
	f.t.Helper()
	peer := &s44EvidencePeer{tracer: s44EvidenceNewTracer(hold)}
	config := f.pool.Config()
	config.MaxConns, config.MinConns = 1, 0
	config.ConnConfig.Tracer = peer.tracer
	var err error
	peer.pool, err = pgxpool.NewWithConfig(f.ctx, config)
	s44Must(f.t, err)
	f.t.Cleanup(func() {
		peer.tracer.open()
		if n := peer.pool.Stat().AcquiredConns(); n != 0 {
			f.t.Errorf("evidence peer acquired=%d; want0", n)
		}
		peer.pool.Close()
		f.t.Logf("evidence_peer=%d resources=pool_closed", peer.pid)
	})
	s44EvidencePeerPreflight(f, peer, config.ConnConfig.User)
	peer.store, err = article.NewPostgresReplayEvidenceStore(peer.pool)
	s44Must(f.t, err)
	return peer
}

func s44EvidencePeerPreflight(f *s44Fixture, peer *s44EvidencePeer, expectedUser string) {
	conn, err := peer.pool.Acquire(f.ctx)
	s44Must(f.t, err)
	defer conn.Release()
	var host, user string
	var port, version int
	err = conn.QueryRow(f.ctx, `SELECT pg_backend_pid(),host(inet_server_addr()),inet_server_port(),
		current_setting('server_version_num')::int,current_user`).Scan(&peer.pid, &host, &port, &version, &user)
	s44Must(f.t, err)
	s44Address(f.t, host, uint16(port))
	if version != 180006 || user != expectedUser {
		f.t.Fatal("evidence peer endpoint/version/identity mismatch")
	}
	if peer.pid == uint32(f.observerPID) {
		f.t.Fatal("evidence peer reused the independent observer backend")
	}
}

type s44EvidenceCall struct {
	result article.BeginResult
	err    error
	done   chan struct{}
}

func s44EvidenceStartBegin(ctx context.Context, peer *s44EvidencePeer, backup, candidate string,
	expected article.AttemptSnapshot, facts article.DeletionReplayFacts) *s44EvidenceCall {
	call := &s44EvidenceCall{done: make(chan struct{})}
	go func() {
		defer close(call.done)
		call.result, call.err = peer.store.Begin(ctx, s44Run, backup, expected, facts, candidate)
	}()
	return call
}

func s44EvidenceAwaitInsert(t *testing.T, ctx context.Context, channel <-chan s44EvidenceInsert) s44EvidenceInsert {
	t.Helper()
	select {
	case observation := <-channel:
		return observation
	case <-ctx.Done():
		t.Fatal("real INSERT barrier was not reached")
	}
	return s44EvidenceInsert{}
}

func s44EvidenceJoin(t *testing.T, call *s44EvidenceCall) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	select {
	case <-call.done:
	case <-ctx.Done():
		t.Fatal("evidence Begin worker did not join")
	}
}

func s44EvidenceBinding(t *testing.T) {
	f := s44Prepared(t, 3)
	const backupB = "synthetic-evidence-backup-B"
	sourceB, probeB := s44EvidenceSourceB(f, backupB)
	factsA, err := f.source.Snapshot(f.ctx, s44Backup)
	s44Must(t, err)
	factsB, err := sourceB.Snapshot(f.ctx, backupB)
	s44Must(t, err)
	first, second := s44EvidenceNewPeer(f, true), s44EvidenceNewPeer(f, false)
	// Real Store.Begin requires the same sealed publication ports as the shared
	// fixture; each peer's synthetic runtime binding names its own backup.
	first.store = s44ConfigureReplayPublication(f.t, f, f.source, first.store, &s44PublicationRuntime{})
	second.store = s44ConfigureReplayPublication(f.t, f, sourceB, second.store,
		&s44PublicationRuntime{backupRef: backupB})
	if first.pid == second.pid {
		t.Fatal("different-backup Begin requires distinct actual backends")
	}
	expectedA, err := first.store.LoadAttempt(f.ctx, s44Run, s44Backup)
	s44Must(t, err)
	expectedB, err := second.store.LoadAttempt(f.ctx, s44Run, backupB)
	s44Must(t, err)
	if expectedA.Exists || expectedB.Exists {
		t.Fatal("first-binding fixture already has a run")
	}
	owner := s44EvidenceRaceBegin(f, first, second, backupB, expectedA, expectedB, factsA, factsB)
	if owner.Disposition != article.AttemptCreated || owner.Attempt.Generation != 1 {
		t.Fatal("first committed Begin was not created/gen1")
	}
	providerA := s44EvidenceProvider(f, first.store, f.source)
	providerB := s44EvidenceProvider(f, second.store, sourceB)
	raw, err := providerA.ReplayAndVerify(f.ctx, s44Run, s44Backup)
	s44Must(t, err)
	ready := s44CheckReady(f, raw, 0, 3)
	s44EvidenceAssertBinding(f, owner.Attempt, ready, backupB)
	before := s44Image(f)
	s44EvidenceWrongBackup(f, providerB, backupB)
	if s44Image(f) != before || probeB.calls != 0 {
		t.Fatal("losing backup changed committed facts or executed ReplayAll")
	}
	f.save("different-backup-final-sql", s44Image(f))
}

func s44EvidenceRaceBegin(f *s44Fixture, first, second *s44EvidencePeer, backupB string,
	expectedA, expectedB article.AttemptSnapshot, factsA, factsB article.DeletionReplayFacts) article.BeginResult {
	ctx, cancel := context.WithTimeout(f.ctx, 20*time.Second)
	candidateA, candidateB := s44EvidenceID(f.t), s44EvidenceID(f.t)
	var calls []*s44EvidenceCall
	f.t.Cleanup(func() {
		first.tracer.open()
		second.tracer.open()
		cancel()
		for _, call := range calls {
			s44EvidenceJoin(f.t, call)
		}
	})
	winner := s44EvidenceStartBegin(ctx, first, s44Backup, candidateA, expectedA, factsA)
	calls = append(calls, winner)
	inserted := s44EvidenceAwaitInsert(f.t, ctx, first.tracer.inserted)
	s44Must(f.t, inserted.Err)
	if inserted.PID != first.pid || inserted.Rows != 1 || inserted.TransactionStatus != 'T' {
		f.t.Fatal("first INSERT was not real/uncommitted")
	}
	loser := s44EvidenceStartBegin(ctx, second, backupB, candidateB, expectedB, factsB)
	calls = append(calls, loser)
	started := s44EvidenceAwaitInsert(f.t, ctx, second.tracer.started)
	if started.PID != second.pid || started.TransactionStatus != 'T' {
		f.t.Fatal("second INSERT is not in its own transaction")
	}
	s44EvidenceWaitBlocked(f, ctx, first.pid, second.pid)
	first.tracer.open()
	s44EvidenceJoin(f.t, winner)
	s44EvidenceJoin(f.t, loser)
	s44Must(f.t, winner.err)
	if !errors.Is(loser.err, article.ErrDeletionReplayBackupMismatch) || loser.result != (article.BeginResult{}) {
		f.t.Fatal("competing backup did not fail binding without an authoritative handle")
	}
	f.save("different-backup-begin", map[string]any{"winner_pid": first.pid, "loser_pid": second.pid,
		"winner_generation": winner.result.Attempt.Generation, "loser_backup_mismatch": true, "first_insert_uncommitted_rows": inserted.Rows})
	return winner.result
}

func s44EvidenceWaitBlocked(f *s44Fixture, ctx context.Context, first, second uint32) {
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		var blocked bool
		var winnerXID, loserXID *string
		err := f.observer.QueryRow(ctx, `SELECT $1::int=ANY(pg_blocking_pids($2::int)),
			(SELECT backend_xid::text FROM pg_stat_activity WHERE pid=$1::int),
			(SELECT backend_xid::text FROM pg_stat_activity WHERE pid=$2::int)`, first, second).Scan(&blocked, &winnerXID, &loserXID)
		s44Must(f.t, err)
		if blocked {
			// A waiter need not allocate an XID before its conflicting INSERT.
			// Distinct backends, both real 'T' states and this lock edge prove
			// the two transactions; record optional XIDs without forcing writes.
			if winnerXID != nil && loserXID != nil && *winnerXID == *loserXID {
				f.t.Fatal("assigned competing transaction IDs are not distinct")
			}
			f.save("different-backup-lock-observed", map[string]any{"blocker_pid": first, "waiter_pid": second,
				"winner_xid": winnerXID, "loser_xid": loserXID, "pg_blocking_pids_confirmed": true})
			return
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			f.t.Fatal("actual PostgreSQL first-binding contention was not observed")
		}
	}
}

func s44EvidenceSourceB(f *s44Fixture, backup string) (*article.ProductionDeletionReplaySource, *s44ReplayProbe) {
	raw := strings.ReplaceAll(s44Inventory, s44Backup, backup)
	at := strings.LastIndex(raw, `,"inventory_digest":`)
	if at < 0 {
		f.t.Fatal("fixed inventory fixture has no canonical digest field")
	}
	raw = raw[:at] + `,"inventory_digest":"` + recovery.SHA256Digest([]byte(raw[:at]+"}")) + `"}`
	name := "synthetic-backup-B.inventory.json"
	f.saveRaw(name, []byte(raw))
	reader, err := article.NewFileBackupInventoryReader(filepath.Join(f.dir, name))
	s44Must(f.t, err)
	manifest, err := reader.Read(f.ctx)
	s44Must(f.t, err)
	if manifest.BackupRef != backup {
		f.t.Fatal("second real inventory did not bind its own backup")
	}
	finalizer, err := article.NewDeletionFinalizer(article.DeletionFinalizerDependencies{
		Transactions: &s44Transactions{f: f}, Repository: article.Repository{}, Journal: f.journal, Auditor: audit.Store{},
		AuditIDs: &s44IDs{}, Clock: f.clock, WorkerID: s44EvidenceID(f.t), LeaseDuration: time.Minute, Identity: &s44Identity{f: f}})
	s44Must(f.t, err)
	probe := &s44ReplayProbe{real: finalizer}
	facts, err := article.NewPostgresReplayFactReader(f.pool)
	s44Must(f.t, err)
	// The publication path needs the verified-snapshot projection; it delegates
	// every pre-existing journal read unchanged.
	source, err := article.NewProductionDeletionReplaySource(&s44PublicationJournal{s44Journal: f.journal},
		reader, probe, facts, facts, f.clock.Now)
	s44Must(f.t, err)
	return source, probe
}

func s44EvidenceProvider(f *s44Fixture, store article.ReplayEvidenceStore, source article.DeletionReplaySource) *article.DeletionReplayProvider {
	provider, err := article.NewDeletionReplayProvider(article.DeletionReplayDependencies{Store: store, Source: source,
		EvidenceIDs: &s44IDs{remaining: []string{s44EvidenceID(f.t)}},
		Tokens:      &s44IDs{remaining: []string{s44EvidenceID(f.t)}}, Clock: f.clock})
	s44Must(f.t, err)
	return provider
}

func s44EvidenceAssertBinding(f *s44Fixture, owner article.AttemptSnapshot, ready recovery.ReadyReplayStatusV1, backupB string) {
	actual, err := f.store.LoadAttempt(f.ctx, s44Run, s44Backup)
	s44Must(f.t, err)
	if actual.BackupRef != s44Backup || actual.Generation != owner.Generation || actual.Token != owner.Token {
		f.t.Fatal("loser changed first binding/owner")
	}
	var runs, evidence, losingEvidence int
	err = f.observer.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM deletion_replay_runs),
		(SELECT count(*) FROM deletion_replay_evidence),
		(SELECT count(*) FROM deletion_replay_evidence WHERE backup_ref=$1)`, backupB).Scan(&runs, &evidence, &losingEvidence)
	s44Must(f.t, err)
	if runs != 1 || evidence != 1 || losingEvidence != 0 || actual.CurrentEvidenceRef == nil || *actual.CurrentEvidenceRef != ready.EvidenceRef {
		f.t.Fatal("competing backups produced duplicate or wrongly bound current/evidence")
	}
}

func s44EvidenceWrongBackup(f *s44Fixture, provider *article.DeletionReplayProvider, backup string) {
	raw, err := provider.ReadStatus(f.ctx, s44Run, backup)
	s44Must(f.t, err)
	s44EvidenceUnavailable(f.t, raw, backup, "failed", "backup_binding_mismatch")
	f.saveRaw("different-backup-loser-read.raw.json", raw)
	raw, err = provider.ReplayAndVerify(f.ctx, s44Run, backup)
	s44Must(f.t, err)
	s44EvidenceUnavailable(f.t, raw, backup, "failed", "backup_binding_mismatch")
	f.saveRaw("different-backup-loser-replay.raw.json", raw)
}

func s44EvidenceUnavailable(t *testing.T, raw []byte, backup, status, reason string) {
	t.Helper()
	s44Unavailable(t, raw, status, reason)
	value, err := recovery.DecodeReplayStatusV1(raw)
	s44Must(t, err)
	if value.Unavailable.RunID != s44Run || value.Unavailable.BackupRef != backup {
		t.Fatal("Unavailable has wrong run/backup binding")
	}
}

// PublishRollback is the real evidence INSERT 23505 subset only. It does not
// simulate or claim rollback after a successful INSERT/current UPDATE failure.
func s44EvidenceRollback(t *testing.T) {
	f := s44Prepared(t, 3)
	reserved, next := s44EvidenceID(t), s44EvidenceID(t)
	s44EvidenceReserveForeign(f, reserved)
	owner := s44Begin(f)
	input := s44ReplayInput(f, 0, 3)
	ids := &s44IDs{remaining: []string{reserved, next}}
	request := s44Request(f, owner, input)
	request.EvidenceIDs = ids
	before, history := s44Image(f), s44EvidenceImage(f)
	ownerBefore, err := f.store.LoadAttempt(f.ctx, s44Run, s44Backup)
	s44Must(t, err)
	f.save("publish-insert-before-sql", before)
	evidence, err := f.store.Publish(f.ctx, request)
	var conflict *pgconn.PgError
	if !errors.As(err, &conflict) || conflict.Code != "23505" || conflict.ConstraintName != "deletion_replay_evidence_pkey" || evidence != (article.DeletionReplayEvidence{}) {
		t.Fatal("Publish did not return the actual foreign-ref PostgreSQL primary-key violation")
	}
	f.save("publish-insert-conflict", map[string]any{"sqlstate": conflict.Code, "constraint": conflict.ConstraintName,
		"boundary": "evidence INSERT itself; not post-INSERT current UPDATE"})
	s44EvidenceAfterConflict(f, before, ownerBefore)
	evidence, err = f.store.Publish(f.ctx, request)
	s44Must(t, err)
	if evidence.EvidenceRef != next || evidence.EvidenceRef == reserved || ids.calls != 2 {
		t.Fatal("same-owner retry did not use exactly the next new ref")
	}
	s44EvidenceAfterRetry(f, owner, evidence, history)
}

func s44EvidenceReserveForeign(f *s44Fixture, reserved string) {
	facts, err := f.source.Snapshot(f.ctx, s44Backup)
	s44Must(f.t, err)
	// Only the foreign preload is synthetic. It never becomes a target Publish input.
	fixture := article.DeletionReplayInput{DeletionReplayFacts: facts, AlreadyAbsent: 3}
	evidence := s44SyntheticEvidence(f, fixture, reserved)
	evidence.RunID = s44EvidenceID(f.t)
	evidence.EvidenceDigest, err = s44ReadyOf(evidence).CanonicalDigest()
	s44Must(f.t, err)
	tx, err := f.pool.Begin(f.ctx)
	s44Must(f.t, err)
	defer s44Rollback(f.t, tx)
	repository := article.Repository{}
	s44Must(f.t, repository.InsertDeletionReplayRun(f.ctx, tx, article.DeletionReplayRun{
		RunID: evidence.RunID, Namespace: recovery.ArticleDeletionNamespaceV1, BackupRef: s44Backup,
		Status: article.ReplayRunRunning, StartedAt: f.clock.at, UpdatedAt: f.clock.at}))
	s44Must(f.t, repository.InsertDeletionReplayEvidence(f.ctx, tx, evidence))
	s44Must(f.t, tx.Commit(f.ctx))
	f.save("foreign-ref-provenance", map[string]any{"synthetic_foreign_run": evidence.RunID, "reserved_ref": reserved,
		"canonical_digest": evidence.EvidenceDigest, "no_target_current": true})
}

func s44EvidenceAfterConflict(f *s44Fixture, before string, owner article.AttemptSnapshot) {
	actual, err := f.store.LoadAttempt(f.ctx, s44Run, s44Backup)
	s44Must(f.t, err)
	if !reflect.DeepEqual(actual, owner) || s44Image(f) != before {
		f.t.Fatal("failed INSERT changed complete row image or authoritative owner")
	}
	f.save("publish-insert-rolled-back-sql", s44Image(f))
	raw, err := f.provider.ReadStatus(f.ctx, s44Run, s44Backup)
	s44Must(f.t, err)
	s44EvidenceUnavailable(f.t, raw, s44Backup, "pending", "replay_in_progress")
	f.saveRaw("publish-insert-pending.raw.json", raw)
	if s44Image(f) != before {
		f.t.Fatal("pending ReadStatus changed failed-Publish state")
	}
	s44EvidenceUnlocked(f)
}

func s44EvidenceUnlocked(f *s44Fixture) {
	tx, err := f.observer.Begin(f.ctx)
	s44Must(f.t, err)
	defer s44Rollback(f.t, tx)
	var run string
	err = tx.QueryRow(f.ctx, `SELECT run_id FROM deletion_replay_runs WHERE run_id=$1 AND namespace=$2 FOR UPDATE NOWAIT`,
		s44Run, recovery.ArticleDeletionNamespaceV1).Scan(&run)
	s44Must(f.t, err)
	if run != s44Run {
		f.t.Fatal("independent NOWAIT observed the wrong run")
	}
	s44Must(f.t, tx.Rollback(f.ctx))
	if f.pool.Stat().AcquiredConns() != 1 {
		f.t.Fatal("failed Publish retained a transaction/connection beyond the observer")
	}
	f.save("publish-insert-resource-observation", map[string]any{"observer_nowait_unlocked": true, "pool_acquired": 1, "observer_reservation": 1})
}

func s44EvidenceAfterRetry(f *s44Fixture, owner article.AttemptSnapshot, evidence article.DeletionReplayEvidence, history map[string]json.RawMessage) {
	raw, err := f.provider.ReadStatus(f.ctx, s44Run, s44Backup)
	s44Must(f.t, err)
	ready := s44CheckReady(f, raw, 0, 3)
	if ready != s44ReadyOf(evidence) || s44EvidenceCount(f) != 2 {
		f.t.Fatal("retry current did not select exactly one target evidence plus unchanged foreign evidence")
	}
	actual, err := f.store.LoadAttempt(f.ctx, s44Run, s44Backup)
	s44Must(f.t, err)
	if actual.Generation != owner.Generation || actual.Token != owner.Token || actual.BackupRef != owner.BackupRef || f.probe.calls != 1 || f.tokens.calls != 1 {
		f.t.Fatal("retry re-Began, replayed, advanced generation or changed original binding/owner")
	}
	s44RequireHistoryPreserved(f, history)
	before := s44Image(f)
	again, err := f.provider.ReadStatus(f.ctx, s44Run, s44Backup)
	s44Must(f.t, err)
	if s44CheckReady(f, again, 0, 3) != ready || s44Image(f) != before {
		f.t.Fatal("retried public Ready is not stable/read-only")
	}
	f.save("publish-insert-retry-final-sql", s44Image(f))
}

var _ pgx.QueryTracer = (*s44EvidenceTracer)(nil)

func s44CurrentUpdateRollback(t *testing.T) {
	f := s44Prepared(t, 3)
	owner := s44Begin(f)
	input := s44ReplayInput(f, 0, 3)
	failed, next := s44EvidenceID(t), s44EvidenceID(t)
	if failed == next {
		t.Fatal("current UPDATE fault and retry require distinct refs")
	}
	ids := &s44IDs{remaining: []string{failed, next}}
	request := s44Request(f, owner, input)
	request.EvidenceIDs = ids
	s44CurrentUpdateInstall(f, failed)
	s44NoEvidence(f)
	history, before := s44EvidenceImage(f), s44Image(f)
	if len(history) != 0 || len(f.storeProbe.history()) != 0 || f.evidenceClock.calls.Load() != 0 {
		t.Fatal("current UPDATE fixture requires zero history and no prior Publish")
	}
	s44CurrentUpdateOwner(f, request, owner, input)
	f.save("publish-update-before-sql", before)
	evidence, err := f.storeProbe.Publish(f.ctx, request)
	if evidence != (article.DeletionReplayEvidence{}) || ids.calls != 1 {
		t.Fatal("failed current UPDATE returned evidence or consumed extra IDs")
	}
	s44CurrentUpdateError(f, err, failed, input)
	s44CurrentUpdateRolledBack(f, before)
	s44CurrentUpdateOwner(f, request, owner, input)
	evidence, err = f.storeProbe.Publish(f.ctx, request)
	s44Must(t, err)
	if evidence.EvidenceRef != next || evidence.EvidenceRef == failed || ids.calls != 2 {
		t.Fatal("same-owner current UPDATE retry did not use exactly the new ref")
	}
	s44CurrentUpdateOwner(f, request, owner, input)
	s44CurrentUpdateReady(f, before, history, evidence, failed)
	s44CurrentUpdateOwner(f, request, owner, input)
	want := []string{"Publish", "ReadCurrent", "Publish", "ReadCurrent", "ReadCurrent"}
	if !reflect.DeepEqual(f.storeProbe.history(), want) || f.storeProbe.overflow || f.evidenceClock.calls.Load() != 2 {
		t.Fatal("current UPDATE retry/reads added Begin, Fail, Publish, Replay or clock work")
	}
	f.save("publish-update-call-history", map[string]any{"events": f.storeProbe.history(),
		"replay_calls": f.probe.calls, "token_calls": f.tokens.calls, "evidence_id_calls": ids.calls,
		"evidence_clock_calls": f.evidenceClock.calls.Load()})
}

func s44CurrentUpdateInstall(f *s44Fixture, failed string) {
	function := pgx.Identifier{f.schema, "s44_current_update_fault"}.Sanitize()
	table := pgx.Identifier{f.schema, "deletion_replay_runs"}.Sanitize()
	evidence := pgx.Identifier{f.schema, "deletion_replay_evidence"}.Sanitize()
	_, err := f.pool.Exec(f.ctx, fmt.Sprintf(s44CurrentUpdateFaultSQL, function, evidence))
	s44Must(f.t, err)
	var ddl string
	err = f.pool.QueryRow(f.ctx, `SELECT format('CREATE TRIGGER %I BEFORE UPDATE OF current_evidence_ref ON %s
		FOR EACH ROW EXECUTE FUNCTION %s(%L)', $1::text, $2::text, $3::text, $4::text)`,
		"s44_current_update_fault", table, function, failed).Scan(&ddl)
	s44Must(f.t, err)
	_, err = f.pool.Exec(f.ctx, ddl)
	s44Must(f.t, err)
}

// This schema-local fault observes a real bound INSERT before rejecting UPDATE.
// Its ref predicate lets the new-ref retry succeed with the trigger still enabled.
const s44CurrentUpdateFaultSQL = `CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $s44$
DECLARE inserted_digest text;
BEGIN
	IF NEW.current_evidence_ref IS DISTINCT FROM TG_ARGV[0] THEN
		RETURN NEW;
	END IF;
	IF NEW.status IS DISTINCT FROM 'ready' OR OLD.status IS DISTINCT FROM 'running'
		OR OLD.current_evidence_ref IS NOT NULL OR OLD.input_digest IS NOT NULL
		OR NEW.run_id IS DISTINCT FROM OLD.run_id OR NEW.namespace IS DISTINCT FROM OLD.namespace
		OR NEW.backup_ref IS DISTINCT FROM OLD.backup_ref
		OR NEW.attempt_generation IS DISTINCT FROM OLD.attempt_generation
		OR NEW.attempt_token IS DISTINCT FROM OLD.attempt_token THEN
		RAISE EXCEPTION USING ERRCODE='P0001', MESSAGE='current UPDATE witness precondition missing';
	END IF;
	SELECT e.evidence_digest INTO inserted_digest FROM %s e
	WHERE e.evidence_ref=NEW.current_evidence_ref AND e.run_id=NEW.run_id
		AND e.namespace=NEW.namespace AND e.backup_ref=NEW.backup_ref
		AND e.evidence_version='article-deletion-replay/v1' AND e.evidence_digest=NEW.input_digest
		AND e.tombstones_total=3 AND e.tombstones_replayed=0 AND e.already_absent=3;
	IF NOT FOUND THEN
		RAISE EXCEPTION USING ERRCODE='P0001', MESSAGE='current UPDATE matching inserted evidence missing';
	END IF;
	RAISE EXCEPTION USING ERRCODE='23514', MESSAGE='controlled current UPDATE failure after evidence INSERT',
		SCHEMA=TG_TABLE_SCHEMA, TABLE=TG_TABLE_NAME, CONSTRAINT='s44_current_update_after_insert',
		DETAIL=json_build_object('witness_version','s44-current-update/v1',
			'stage','current_update_after_evidence_insert','evidence_seen',true,'binding_ok',true,
			'pid',pg_backend_pid(),'evidence_ref',NEW.current_evidence_ref,'evidence_digest',inserted_digest)::text;
END;
$s44$;`

type s44CurrentUpdateWitness struct {
	Version        string `json:"witness_version"`
	Stage          string `json:"stage"`
	EvidenceSeen   bool   `json:"evidence_seen"`
	BindingOK      bool   `json:"binding_ok"`
	PID            uint32 `json:"pid"`
	EvidenceRef    string `json:"evidence_ref"`
	EvidenceDigest string `json:"evidence_digest"`
}

func s44CurrentUpdateError(f *s44Fixture, cause error, failed string, input article.DeletionReplayInput) {
	var failure *pgconn.PgError
	if !errors.As(cause, &failure) || failure.Code != "23514" || failure.SchemaName != f.schema ||
		failure.TableName != "deletion_replay_runs" || failure.ConstraintName != "s44_current_update_after_insert" {
		f.t.Fatal("Publish did not return the real current UPDATE trigger violation")
	}
	var witness s44CurrentUpdateWitness
	decoder := json.NewDecoder(strings.NewReader(failure.Detail))
	decoder.DisallowUnknownFields()
	s44Must(f.t, decoder.Decode(&witness))
	var extra any
	if !errors.Is(decoder.Decode(&extra), io.EOF) {
		f.t.Fatal("current UPDATE witness has trailing data")
	}
	// Only an expected canonical digest is constructed; no synthetic evidence is inserted.
	expected := s44SyntheticEvidence(f, input, failed)
	if witness.Version != "s44-current-update/v1" || witness.Stage != "current_update_after_evidence_insert" ||
		!witness.EvidenceSeen || !witness.BindingOK || witness.PID == 0 || witness.PID == uint32(f.observerPID) ||
		witness.EvidenceRef != failed || witness.EvidenceDigest != expected.EvidenceDigest {
		f.t.Fatal("real current UPDATE witness does not bind the inserted candidate")
	}
	f.save("publish-update-failure-witness", map[string]any{"sqlstate": failure.Code,
		"constraint": failure.ConstraintName, "witness": witness, "boundary": "BEFORE current UPDATE; after evidence INSERT"})
}

func s44CurrentUpdateOwner(f *s44Fixture, request article.ReplayPublishRequest, owner article.AttemptSnapshot, input article.DeletionReplayInput) {
	actual, err := f.store.LoadAttempt(f.ctx, s44Run, s44Backup)
	s44Must(f.t, err)
	// Ready may change only the publication fields, never the authoritative handle.
	if actual.Status == article.ReplayRunReady {
		actual.Status, actual.InputDigest, actual.CurrentEvidenceRef = owner.Status, owner.InputDigest, owner.CurrentEvidenceRef
	}
	if !reflect.DeepEqual(actual, owner) || !reflect.DeepEqual(request.Attempt, owner) || request.Input != input ||
		f.probe.calls != 1 || f.tokens.calls != 1 || f.evidenceIDs.calls != 0 || f.storeProbe.ackObserver != nil {
		f.t.Fatal("current UPDATE path changed complete owner/Input or repeated Begin/Replay")
	}
}

func s44CurrentUpdateRolledBack(f *s44Fixture, before string) {
	if s44Image(f) != before {
		f.t.Fatal("failed current UPDATE changed seven-table rows/xmin/ctid")
	}
	s44NoEvidence(f)
	f.save("publish-update-rolled-back-sql", s44Image(f))
	raw, err := f.provider.ReadStatus(f.ctx, s44Run, s44Backup)
	s44Must(f.t, err)
	s44EvidenceUnavailable(f.t, raw, s44Backup, "pending", "replay_in_progress")
	f.saveRaw("publish-update-pending.raw.json", raw)
	if s44Image(f) != before {
		f.t.Fatal("pending ReadStatus changed current UPDATE rollback state")
	}
	tx, err := f.observer.Begin(f.ctx)
	s44Must(f.t, err)
	defer s44Rollback(f.t, tx)
	var run string
	err = tx.QueryRow(f.ctx, `SELECT run_id FROM deletion_replay_runs WHERE run_id=$1 AND namespace=$2 FOR UPDATE NOWAIT`,
		s44Run, recovery.ArticleDeletionNamespaceV1).Scan(&run)
	s44Must(f.t, err)
	s44Must(f.t, tx.Rollback(f.ctx))
	if run != s44Run || f.pool.Stat().AcquiredConns() != 1 || f.evidenceClock.calls.Load() != 1 {
		f.t.Fatal("current UPDATE rollback retained a lock/connection or pending read used the clock")
	}
	f.save("publish-update-resource-observation", map[string]any{"observer_nowait_unlocked": true,
		"pool_acquired": 1, "observer_reservation": 1})
}

func s44CurrentUpdateReady(f *s44Fixture, before string, history map[string]json.RawMessage, evidence article.DeletionReplayEvidence, failed string) {
	var missing bool
	s44Must(f.t, f.observer.QueryRow(f.ctx, `SELECT NOT EXISTS(SELECT 1 FROM deletion_replay_evidence WHERE evidence_ref=$1)`, failed).Scan(&missing))
	if !missing || s44EvidenceCount(f) != 1 || evidence.TombstonesReplayed != 0 || evidence.AlreadyAbsent != 3 {
		f.t.Fatal("current UPDATE retry retained a failed/duplicate evidence or wrong partitions")
	}
	var enabled bool
	s44Must(f.t, f.observer.QueryRow(f.ctx, `SELECT tgenabled='O' FROM pg_trigger
		WHERE tgrelid=$1::regclass AND tgname='s44_current_update_fault'`,
		pgx.Identifier{f.schema, "deletion_replay_runs"}.Sanitize()).Scan(&enabled))
	if !enabled {
		f.t.Fatal("new-ref retry disabled the current UPDATE fault")
	}
	s44RequireHistoryPreserved(f, history)
	var oldImage, newImage map[string]json.RawMessage
	s44Must(f.t, json.Unmarshal([]byte(before), &oldImage))
	after := s44Image(f)
	s44Must(f.t, json.Unmarshal([]byte(after), &newImage))
	for _, table := range []string{"articles", "deletion_operations", "audit_events", "users", "identity_identifiers"} {
		if !reflect.DeepEqual(oldImage[table], newImage[table]) {
			f.t.Fatal("current UPDATE retry changed completed business rows/xmin/ctid")
		}
	}
	clock := f.evidenceClock.calls.Load()
	for range 2 {
		raw, err := f.provider.ReadStatus(f.ctx, s44Run, s44Backup)
		s44Must(f.t, err)
		if s44CheckReady(f, raw, 0, 3) != s44ReadyOf(evidence) || s44Image(f) != after || f.evidenceClock.calls.Load() != clock {
			f.t.Fatal("current UPDATE retry Ready/SQL/Facts is not stable/read-only")
		}
	}
	f.save("publish-update-retry-final-sql", after)
}
