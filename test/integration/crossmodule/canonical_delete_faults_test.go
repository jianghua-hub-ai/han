//go:build integration

package crossmodule_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"han/internal/audit"
	"han/internal/identity"
	"han/internal/platform"
	"han/internal/recovery"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestCanonicalUserDeleteFaultsPostgres(t *testing.T) {
	t.Run("Rejection", func(t *testing.T) {
		t.Run("ActorTarget", cdRejectActorTarget)
		t.Run("Identifier", cdRejectIdentifier)
		t.Run("Trace", cdRejectTrace)
		t.Run("Recovery", cdRejectRecovery)
	})
	t.Run("Rollback", func(t *testing.T) {
		for _, stage := range []string{"User", "Identifier", "Session", "Credential", "Article",
			"ArticleAudit", "CanonicalAudit", "RetainedClear", "EmailDelete", "ReleaseAudit"} {
			t.Run("Daily"+stage, func(t *testing.T) { cdCanonicalFault(t, stage, false, false) })
		}
		t.Run("RecoveryReviewComplete", func(t *testing.T) { cdCanonicalFault(t, "ReviewComplete", true, false) })
		t.Run("RecoveryCanonicalAudit", func(t *testing.T) { cdCanonicalFault(t, "CanonicalAudit", true, false) })
		t.Run("RecoveryPathBLateAudit", func(t *testing.T) { cdCanonicalFault(t, "CanonicalAudit", true, true) })
		for _, stage := range []string{"RecipientInsert", "DispositionAudit", "RetainedClear", "EmailDelete", "ReleaseAudit"} {
			t.Run("LastTransfer"+stage, func(t *testing.T) { cdDispositionFault(t, stage) })
		}
		t.Run("CancelAfterArticle", cdCancelAfterArticle)
	})
	t.Run("Concurrency", func(t *testing.T) {
		t.Run("CanonicalDisposition", cdCanonicalDispositionRace)
		t.Run("TwoLastDispositions", cdTwoLastDispositions)
	})
}

func cdCollision(f *cdFixture, action string, nth int) {
	id := cdUUID(f.t)
	tx, err := f.pool.Begin(f.ctx)
	cdMust(f.t, err)
	defer cdRollback(tx)
	actor := "synthetic-collision-fixture"
	cdMust(f.t, (audit.Store{}).Append(f.ctx, tx, audit.Event{ID: id, OccurredAt: f.clock.at,
		ActorKind: audit.ActorSystem, ActorSystem: &actor, Action: "synthetic.seed",
		TargetType: "synthetic", TargetID: "collision", Outcome: "succeeded", TraceID: cdTrace,
		Metadata: map[string]any{"fixture": true}}))
	cdMust(f.t, tx.Commit(f.ctx))
	f.auditor.action, f.auditor.collision, f.auditor.nth = action, id, nth
}

func cdSetFault(f *cdFixture, stage string) {
	switch stage {
	case "ArticleAudit":
		cdCollision(f, "article.orphan.canonical", 2)
	case "CanonicalAudit":
		cdCollision(f, "identity.user.delete.canonical", 1)
	case "ReleaseAudit":
		cdCollision(f, "identity.identifier.email.release", 1)
	case "DispositionAudit":
		cdCollision(f, "article.orphan.transfer", 1)
	default:
		f.fault = stage
	}
}

func cdDisableFault(f *cdFixture) {
	f.fault, f.auditor.action = "", ""
	f.before, f.hook = nil, nil
}

func cdCanonicalFault(t *testing.T, stage string, kept, deleted bool) {
	kinds := []string{"owned", "violating"}
	if deleted {
		kinds = []string{"held", "held_violating", "pending_held"}
	}
	if stage == "RetainedClear" || stage == "EmailDelete" || stage == "ReleaseAudit" {
		kinds = nil
	}
	f := cdNew(t, cdCase{recovery: kept, deleted: deleted, kinds: kinds})
	cdSetFault(f, stage)
	before := cdObserve(f, "before_fault")
	f.before = before
	_, err := cdService(f).CanonicalDelete(f.ctx, f.request)
	cdRequire(t, err != nil && platform.PublicCode(err) == platform.CodePlatformDatabaseUnavailable,
		"canonical fault did not return persistence error")
	cdAssertFault(f, stage, before)
	if kept {
		cdBool(f, `SELECT decision='pending' AND reviewed_at IS NULL AND reviewed_by IS NULL
			FROM recovery_account_reviews WHERE recovery_run_id=$1 AND user_id=$2`, f.request.Recovery.Capability.RunID(), f.source)
	}
	cdDisableFault(f)
	r := cdDelete(f)
	cdRequire(t, r.PendingOrphanCount == len(kinds), "legal retry failed to converge")
	cdAssertDeleted(f, len(kinds) == 0)
	cdAssertArticles(f)
	if kept {
		cdAssertRecovery(f)
	}
	cdObserve(f, "after_legal_retry")
}

func cdAssertFault(f *cdFixture, stage string, before cdImage) {
	cdRequire(f.t, f.faultHit == 1 || f.auditor.hits == 1, "requested fault stage never reached")
	cdSame(f.t, before, cdObserve(f, "after_rollback"))
	cdRequire(f.t, len(f.transactions.seen) == 1, "unexpected transaction count during fault")
	tx := f.transactions.seen[0]
	cdRequire(f.t, tx.ended && !tx.committed && tx.rollbacks == 1 && tx.rollbackErr == nil && tx.writes > 0, "failure lacked real writes and confirmed rollback")
	if stage == "Article" || stage == "ArticleAudit" {
		cdRequire(f.t, f.articleWrites == 2 && tx.audits >= 1, "late article failure did not follow first article SQL and audit")
	}
	if stage == "ReviewComplete" {
		cdRequire(f.t, tx.audits >= 3, "review completed before article/canonical audits")
	}
	if strings.HasSuffix(stage, "Audit") {
		cdRequire(f.t, f.auditor.hits == 1, "audit failure was not a real INSERT 23505")
	}
}

func cdDispositionFault(t *testing.T, stage string) {
	f := cdNew(t, cdCase{kinds: []string{"owned", "violating"}})
	cdDelete(f)
	cdDispose(f, 0, false)
	cdRetained(f, true)
	cdSetFault(f, stage)
	before := cdObserve(f, "before_last_transfer_fault")
	f.before = before
	row := cdReadArticle(f, f.rows[1].ID)
	_, err := cdCommand(f).TransferOrphan(f.ctx, f.actor, row.ID, row.Version, f.other)
	cdRequire(t, err != nil, "last transfer fault unexpectedly succeeded")
	cdSame(t, before, cdObserve(f, "after_last_transfer_rollback"))
	cdRequire(t, f.recipientWrites == 2 && (f.faultHit == 1 || f.auditor.hits == 1), "transfer fault/real recipient writes not reached")
	last := f.transactions.seen[len(f.transactions.seen)-1]
	cdRequire(t, last.ended && !last.committed && last.rollbacks == 1 && last.rollbackErr == nil, "transfer did not roll back")
	if stage == "RetainedClear" || stage == "EmailDelete" || stage == "ReleaseAudit" {
		cdRequire(t, f.proofCalls >= 2, "identity fault bypassed real completion verifier")
	}
	cdDisableFault(f)
	cdDispose(f, 1, true)
	cdRetained(f, false)
	cdReleaseCount(f, 1)
	cdObserve(f, "after_transfer_legal_retry")
}

func cdRejected(f *cdFixture, ctx context.Context, code platform.Code) {
	before := cdObserve(f, "before_rejection")
	_, err := cdService(f).CanonicalDelete(ctx, f.request)
	cdRequire(f.t, err != nil && platform.PublicCode(err) == code, "rejection public code mismatch")
	cdSame(f.t, before, cdObserve(f, "after_rejection"))
	cdRequire(f.t, f.articleCalls == 0 && f.auditor.hits == 0, "rejection called article or collision appender")
	for _, tx := range f.transactions.seen {
		cdRequire(f.t, tx.writes == 0 && tx.ended && !tx.committed && tx.rollbackErr == nil, "rejection was not zero-write")
	}
	f.t.Logf("schema=%s rejection_code=%s zero_business_writes=true article_calls=0", f.schema, code)
}

func cdFixtureSQL(f *cdFixture, sql string, args ...any) {
	_, err := f.pool.Exec(f.ctx, sql, args...)
	cdMust(f.t, err)
}

func cdRejectActorTarget(t *testing.T) {
	for _, name := range []string{"Missing", "Self", "ProtectedInitial", "AdminTarget", "DatabaseActorRevoked"} {
		t.Run(name, func(t *testing.T) {
			f := cdNew(t, cdCase{kinds: []string{"owned"}})
			code := platform.CodeAuthForbidden
			switch name {
			case "Missing":
				f.request.TargetID, code = cdUUID(t), platform.CodeIdentityUserNotFound
			case "Self":
				f.request.TargetID = f.actor.UserID
			case "ProtectedInitial":
				tx, err := f.pool.Begin(f.ctx)
				cdMust(t, err)
				defer cdRollback(tx)
				f.request.TargetID, f.request.TargetRole = cdUser(f, tx, "admin", true), identity.RoleAdmin
				cdMust(t, tx.Commit(f.ctx))
			case "AdminTarget":
				cdFixtureSQL(f, `UPDATE users SET role='admin' WHERE id=$1`, f.source)
				f.request.TargetRole = identity.RoleAdmin
			case "DatabaseActorRevoked":
				cdFixtureSQL(f, `UPDATE users SET status='suspended' WHERE id=$1`, f.actor.UserID)
			}
			cdRejected(f, f.ctx, code)
		})
	}
}

func cdRejectIdentifier(t *testing.T) {
	for _, name := range []string{"ZeroRows", "MissingUsername", "MissingEmail", "MissingPhone", "UsernameValue",
		"EmailValue", "PhoneValue", "ReleasedPhone", "PathBEmailBinding", "UnknownKindConstraint"} {
		t.Run(name, func(t *testing.T) {
			b := name == "PathBEmailBinding"
			f := cdNew(t, cdCase{recovery: b, deleted: b, kinds: []string{"held"}})
			if name == "UnknownKindConstraint" {
				cdIdentifierConstraint(f)
				return
			}
			cdMutateIdentifier(f, name)
			cdRejected(f, f.ctx, platform.CodeIdentityInvalidState)
		})
	}
}

func cdMutateIdentifier(f *cdFixture, name string) {
	kind := strings.ToLower(strings.TrimPrefix(name, "Missing"))
	if name == "ZeroRows" {
		cdFixtureSQL(f, `DELETE FROM identity_identifiers WHERE user_id=$1`, f.source)
	} else if strings.HasPrefix(name, "Missing") {
		cdFixtureSQL(f, `DELETE FROM identity_identifiers WHERE user_id=$1 AND kind=$2`, f.source, kind)
	} else if strings.HasSuffix(name, "Value") {
		kind = strings.ToLower(strings.TrimSuffix(name, "Value"))
		cdFixtureSQL(f, `UPDATE identity_identifiers SET normalized_value=$3 WHERE user_id=$1 AND kind=$2`, f.source, kind, "synthetic-drift-"+cdUUID(f.t))
	} else if name == "ReleasedPhone" {
		cdFixtureSQL(f, `UPDATE identity_identifiers SET released_at=$2 WHERE user_id=$1 AND kind='phone'`, f.source, f.clock.at)
	} else {
		cdFixtureSQL(f, `UPDATE identity_identifiers SET normalized_value=$2 WHERE user_id=$1 AND kind='email'`, f.source, "different@example.invalid")
	}
}

func cdIdentifierConstraint(f *cdFixture) {
	before := cdObserve(f, "before_constraint_probe")
	tx, err := f.pool.Begin(f.ctx)
	cdMust(f.t, err)
	defer cdRollback(tx)
	_, err = tx.Exec(f.ctx, `UPDATE identity_identifiers SET kind='unknown_kind' WHERE user_id=$1 AND kind='phone'`, f.source)
	var pg *pgconn.PgError
	cdRequire(f.t, errors.As(err, &pg) && pg.Code == "23514", "unknown kind escaped actual CHECK")
	cdRollback(tx)
	cdSame(f.t, before, cdObserve(f, "after_constraint_probe"))
	f.t.Log("negative_control=unknown_kind sqlstate=23514 constraints_enabled=true fixture_not_committed=true")
}

func cdRejectTrace(t *testing.T) {
	for _, name := range []string{"MissingContext", "Mismatch", "MissingAssertion"} {
		t.Run(name, func(t *testing.T) {
			f := cdNew(t, cdCase{kinds: []string{"owned"}})
			ctx, code := f.ctx, platform.CodePlatformInternalError
			switch name {
			case "MissingContext":
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()
			case "Mismatch":
				f.request.TraceID, code = "01ARZ3NDEKTSV4RRFFQ69G5FAW", platform.CodeIdentityInvalidState
			case "MissingAssertion":
				f.request.TraceID = ""
			}
			cdRejected(f, ctx, code)
		})
	}
}

func cdRejectRecovery(t *testing.T) {
	for _, name := range []string{"MissingCapability", "MissingStore", "MissingRun", "WrongBackup", "FailedRun",
		"IdentityReadyRun", "NilGate", "ClosedGate", "ErrorGate", "MissingReview", "DeniedReview", "ReviewEvidence"} {
		t.Run(name, func(t *testing.T) {
			f := cdNew(t, cdCase{recovery: true, kinds: []string{"owned", "violating"}})
			code := platform.CodeIdentityInvalidState
			if name == "MissingCapability" {
				code = platform.CodeAuthForbidden
			}
			cdMutateRecovery(f, name)
			cdRejected(f, f.ctx, code)
		})
	}
}

func cdMutateRecovery(f *cdFixture, name string) {
	run := f.request.Recovery.Capability.RunID()
	switch name {
	case "MissingCapability":
		f.request.Recovery = nil
	case "MissingStore":
		f.store = nil
	case "MissingRun", "WrongBackup":
		backup := f.request.Recovery.Capability.BackupRef()
		if name == "MissingRun" {
			run = "synthetic-absent-run"
		} else {
			backup = "synthetic-wrong-backup"
		}
		capability, err := identity.NewRecoveryDeleteCapability(run, backup)
		cdMust(f.t, err)
		f.request.Recovery.Capability = capability
	case "FailedRun", "IdentityReadyRun":
		next := recovery.RecoveryOutcomeFailed
		if name == "IdentityReadyRun" {
			next = recovery.RecoveryOutcomeIdentityReady
			// 0010 requires an initial-admin change proof and no pending review.
			auditID := cdUUID(f.t)
			cdFixtureSQL(f, `INSERT INTO audit_events(id,occurred_at,actor_kind,actor_system,action,target_type,target_id,outcome,trace_id)
				VALUES($1,$2,'system','synthetic-fixture','identity.initial_admin.change','user',$3,'succeeded',$4)`,
				auditID, f.clock.at, f.actor.UserID, f.request.TraceID)
			cdFixtureSQL(f, `UPDATE recovery_runs SET initial_admin_user_id=$2,initial_admin_bootstrap_generation=1,
				initial_admin_changed_generation=2,initial_admin_change_audit_id=$3,initial_admin_changed_at=$4 WHERE run_id=$1`,
				run, f.actor.UserID, auditID, f.clock.at)
			cdFixtureSQL(f, `UPDATE recovery_account_reviews SET decision='kept_deleted',reviewed_by=$2,reviewed_at=$3,
				evidence_ref='synthetic-review-evidence' WHERE recovery_run_id=$1`, run, f.actor.UserID, f.clock.at)
		}
		_, err := f.store.Transition(f.ctx, run, recovery.RecoveryOutcomeRunning, next, recovery.RecoveryRunChange{ChangedAt: f.clock.at})
		cdMust(f.t, err)
	case "NilGate":
		f.gate = nil
	case "ClosedGate":
		f.gate.readiness = true
	case "ErrorGate":
		f.gate.err = cdInjected
	case "MissingReview":
		cdFixtureSQL(f, `DELETE FROM recovery_account_reviews WHERE recovery_run_id=$1`, run)
	case "DeniedReview", "ReviewEvidence":
		decision := "denied"
		if name == "ReviewEvidence" {
			decision = "kept_deleted"
		}
		cdFixtureSQL(f, `UPDATE recovery_account_reviews SET decision=$2,reviewed_by=$3,reviewed_at=$4,evidence_ref='different-evidence'
			WHERE recovery_run_id=$1`, run, decision, f.actor.UserID, f.clock.at)
	}
}

func cdCancelAfterArticle(t *testing.T) {
	f := cdNew(t, cdCase{kinds: []string{"owned", "violating"}})
	before := cdObserve(f, "before_cancel")
	parent := f.ctx
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	f.ctx = ctx
	f.hook = func(_ context.Context, _ pgx.Tx, stage string) error {
		if stage == "Article" {
			cancel()
			return context.Canceled
		}
		return nil
	}
	_, err := cdService(f).CanonicalDelete(ctx, f.request)
	cdRequire(t, errors.Is(err, context.Canceled), "cancelled canonical did not retain cancellation cause")
	f.ctx = parent
	cdSame(t, before, cdObserve(f, "after_cancel_rollback"))
	last := f.transactions.seen[len(f.transactions.seen)-1]
	cdRequire(t, last.ended && !last.committed && last.rollbacks == 1 && last.rollbackErr == nil && last.audits >= 1, "cancel did not roll back real prior SQL/audit")
	cdDisableFault(f)
	cdDelete(f)
	cdAssertDeleted(f, false)
	cdObserve(f, "after_cancel_legal_retry")
}

// Barriers hold real SQL transactions. Contenders run synchronously while the
// owner goroutine is parked; all exit paths unblock and join before pool close.
type cdBarrier struct {
	entered chan struct{}
	release chan struct{}
	done    chan error
}

func cdStartBarrier(t *testing.T, f *cdFixture, stage string, run func() error) *cdBarrier {
	b := &cdBarrier{entered: make(chan struct{}), release: make(chan struct{}), done: make(chan error, 1)}
	var entered bool
	f.hook = func(ctx context.Context, _ pgx.Tx, current string) error {
		if current != stage || entered {
			return nil
		}
		entered = true
		close(b.entered)
		select {
		case <-b.release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	go func() { b.done <- run() }()
	t.Cleanup(func() { b.finish(t, f) })
	select {
	case <-b.entered:
	case err := <-b.done:
		b.done <- err
		t.Fatal("owner exited before reaching SQL barrier")
	case <-time.After(20 * time.Second):
		t.Fatal("owner did not reach SQL barrier")
	}
	return b
}

func (b *cdBarrier) finish(t *testing.T, f *cdFixture) {
	if b.done == nil {
		return
	}
	close(b.release)
	select {
	case err := <-b.done:
		b.done = nil
		cdMust(t, err)
	case <-time.After(15 * time.Second):
		f.cancel()
		select {
		case <-b.done:
			b.done = nil
		case <-time.After(5 * time.Second):
			t.Fatal("owner could not join after cancellation")
		}
		t.Fatal("owner did not finish within join budget")
	}
	f.hook = nil
	f.t.Logf("schema=%s barrier=released goroutine=joined", f.schema)
	cdObserve(f, "after_concurrent_commit")
}

func cdNOWAIT(t *testing.T, err error) {
	t.Helper()
	var pg *pgconn.PgError
	cdRequire(t, errors.As(err, &pg) && pg.Code == "55P03", "contender did not retreat on real NOWAIT lock")
}

func cdCanonicalDispositionRace(t *testing.T) {
	f := cdNew(t, cdCase{recovery: true, kinds: []string{"held", "owned"}})
	before := cdObserve(f, "before_canonical_barrier")
	f.before = before
	service := cdService(f)
	b := cdStartBarrier(t, f, "ArticleFirst", func() error {
		_, err := service.CanonicalDelete(f.ctx, f.request)
		return err
	})
	cdSame(t, before, cdObserve(f, "canonical_uncommitted_observer"))
	_, err := cdCommand(f).PublishOrphan(f.ctx, f.actor, f.rows[0].ID, f.rows[0].Version)
	cdNOWAIT(t, err)
	cdSame(t, before, cdObserve(f, "after_contender_retreat"))
	b.finish(t, f)
	cdRetained(f, true)
	cdDispose(f, 0, false)
	cdRetained(f, true)
	cdDispose(f, 1, false)
	cdRetained(f, false)
	cdReleaseCount(f, 1)
}

func cdTwoLastDispositions(t *testing.T) {
	f := cdNew(t, cdCase{kinds: []string{"owned", "owned"}})
	cdDelete(f)
	first, second := cdReadArticle(f, f.rows[0].ID), cdReadArticle(f, f.rows[1].ID)
	before := cdObserve(f, "before_two_dispositions")
	f.before = before
	command := cdCommand(f)
	b := cdStartBarrier(t, f, "Disposition", func() error {
		_, err := command.PublishOrphan(f.ctx, f.actor, first.ID, first.Version)
		return err
	})
	_, err := cdCommand(f).PublishOrphan(f.ctx, f.actor, second.ID, second.Version)
	cdNOWAIT(t, err)
	cdSame(t, before, cdObserve(f, "second_disposition_retreat"))
	b.finish(t, f)
	cdRetained(f, true)
	cdReleaseCount(f, 0)
	cdDispose(f, 1, false)
	cdRetained(f, false)
	cdReleaseCount(f, 1)
	cdRequire(t, f.proofCalls == 2, "two successful dispositions did not use exactly two identity verifications")
}
