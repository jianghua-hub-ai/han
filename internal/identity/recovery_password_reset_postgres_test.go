//go:build integration

package identity

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"han/internal/idempotency"
	"han/internal/mailer"
	"han/internal/platform"
	"han/internal/recovery"
	"han/internal/security"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type s412Secret struct{ calls int }

func (s *s412Secret) Generate() (string, error) { s.calls++; return "SyntheticRestored123", nil }

type s412Mail struct {
	calls int
	err   error
}

func (s *s412Mail) Send(ctx context.Context, _ mailer.Message) error {
	s.calls++
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.err
}

type s412IDs struct{ calls atomic.Int64 }

func (s *s412IDs) NewID() (string, error) { s.calls.Add(1); return (s410IDs{}).NewID() }

type s412Reader struct {
	approved RecoveryReviewApproval
	calls    atomic.Int64
	err      error
}

func (s *s412Reader) ReadApprovedReview(ctx context.Context, install, run, target, ref string) (RecoveryReviewApproval, error) {
	s.calls.Add(1)
	if err := ctx.Err(); err != nil {
		return RecoveryReviewApproval{}, err
	}
	if s.err != nil {
		return RecoveryReviewApproval{}, s.err
	}
	return s.approved, nil
}

type s412PG struct {
	h         *s410PG
	ctx       context.Context
	p         recovery.PhasePermit
	e         *recoveryIdentityExecution
	owner     *recoveryRestoreReset
	reader    *s412Reader
	secrets   *s412Secret
	mail      *s412Mail
	ids       *s412IDs
	passwords *s411Passwords
	actor     LoginResult
	request   RecoveryRestoreRequest
}

func s412WithPG(t *testing.T, run func(*s412PG)) {
	h := s411PGHarness(t)
	raw, err := os.ReadFile(filepath.Join("..", "..", "migrations", "0011_article_replay_publication.sql"))
	if err != nil {
		t.Fatal(err)
	}
	h.exec(string(raw))
	h.running()
	s411Phase(h, func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		passwords := s411Trusted(t, h, c, p, e)
		old, err := s411Login(t, h, c, e, passwords, "synthetic-secret")
		if err != nil {
			t.Fatal(err)
		}
		changed, err := s411Change(t, h, c, p, e, passwords, old)
		if err != nil {
			t.Fatal(err)
		}
		f := s412NewPG(t, h, c, p, e, passwords)
		f.actor = LoginResult{SessionToken: changed.SessionToken, CSRFToken: changed.CSRFToken}
		run(f)
	})
}

func s412NewPG(t *testing.T, h *s410PG, c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution, passwords *s411Passwords) *s412PG {
	t.Helper()
	b, err := e.maintenanceBinding(c)
	if err != nil {
		t.Fatal(err)
	}
	f := &s412PG{h: h, ctx: c, p: p, e: e, passwords: passwords, secrets: &s412Secret{}, mail: &s412Mail{}, ids: &s412IDs{}}
	f.request = RecoveryRestoreRequest{TargetID: s410ND, Role: RoleAdmin, Status: UserStatusActive, EvidenceRef: "synthetic-independent-reviewed-source", IdempotencyKey: "synthetic-restore-key-1"}
	f.reader = &s412Reader{approved: RecoveryReviewApproval{Version: recoveryReviewVersion, InstallationID: b.InstallationID, RunID: b.RunID,
		TargetID: s410ND, ActorID: s410Initial, Role: f.request.Role, Status: f.request.Status, EvidenceRef: f.request.EvidenceRef,
		EvidenceDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}
	store := &PasswordResetStore{}
	reset, err := NewPasswordResetService(PasswordResetDependencies{Transactions: h.pool, Repository: PostgresPasswordResetRepository{}, Store: store,
		PasswordPolicy: security.PasswordPolicy{}, PasswordHasher: passwords, Secrets: f.secrets, Mailer: f.mail,
		IdempotencyKeys: idempotency.DigestKeyring{ActiveVersion: "synthetic-s412", Keys: map[string][]byte{"synthetic-s412": bytes.Repeat([]byte{0x43}, 32)}},
		IdempotencyLock: 10 * time.Second, IdempotencyTTL: time.Hour, SMTPOperationBudget: time.Second, Audit: h.audit, IDs: f.ids, Clock: h.clock})
	if err != nil {
		t.Fatal(err)
	}
	permit, err := NewPasswordResetConnectionPermit(1)
	if err != nil {
		t.Fatal(err)
	}
	port, err := NewRecoveryRestoreReset(RecoveryRestoreResetDependencies{Execution: e, Resets: reset, Evidence: f.reader, Permit: permit, WorkerID: "synthetic-s412-driver",
		WorkerConfig: PasswordResetWorkerConfig{LeaseDuration: time.Minute, RetryBackoff: time.Second, MaxRetries: 3, ClaimBatchLimit: 1, ConnectionBudget: 1}})
	if err != nil {
		t.Fatal(err)
	}
	f.owner = port.(*recoveryRestoreReset)
	e.d.ids = f.ids
	t.Cleanup(func() {
		stats := permit.Stats()
		if stats.Current != 0 || stats.Acquired != stats.Released {
			t.Errorf("permit not joined: %+v", stats)
		}
	})
	return f
}

func (f *s412PG) withActor(fn func(context.Context) error) error {
	t := f.h.t
	t.Helper()
	store, err := f.e.maintenanceSessions(&PasswordResetStore{})
	if err != nil {
		t.Fatal(err)
	}
	auth, err := platform.NewHTTPAuthorizer(store, platform.Clock(f.h.clock.Now))
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "https://synthetic.invalid/reviews/resolve", nil).WithContext(f.ctx)
	request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: f.actor.SessionToken})
	request.Header.Set(platform.CSRFHeaderName, f.actor.CSRFToken)
	defer request.Body.Close()
	response := httptest.NewRecorder()
	called := false
	var result error
	auth.Protect(session.AccessGeneral, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		result = fn(r.Context())
		if result == nil {
			w.WriteHeader(http.StatusNoContent)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	})).ServeHTTP(response, request)
	defer response.Result().Body.Close()
	if !called {
		t.Fatalf("real maintenance authentication refused: %s", response.Body.String())
	}
	return result
}

func (f *s412PG) resolve() error {
	return f.withActor(func(c context.Context) error { return f.owner.ResolveRestored(c, f.p, f.request) })
}

func (f *s412PG) image() string {
	f.h.t.Helper()
	var raw string
	err := f.h.pool.QueryRow(f.h.ctx, `SELECT jsonb_build_object(
 'users',(SELECT jsonb_agg(to_jsonb(x) ORDER BY id) FROM users x),
 'runs',(SELECT jsonb_agg(to_jsonb(x) ORDER BY run_id) FROM recovery_runs x),
 'reviews',(SELECT jsonb_agg(to_jsonb(x) ORDER BY user_id) FROM recovery_account_reviews x),
 'operations',(SELECT jsonb_agg(to_jsonb(x) ORDER BY operation_id) FROM password_reset_operations x),
 'credentials',(SELECT jsonb_agg(to_jsonb(x) ORDER BY id) FROM temporary_credentials x),
 'jobs',(SELECT jsonb_agg(to_jsonb(x) ORDER BY operation_id) FROM password_reset_recovery_jobs x),
 'claims',(SELECT jsonb_agg(to_jsonb(x) ORDER BY scope_hash,key_locator) FROM idempotency_records x),
 'sessions',(SELECT jsonb_agg(to_jsonb(x) ORDER BY id) FROM sessions x),
 'audits',(SELECT jsonb_agg(to_jsonb(x) ORDER BY id) FROM audit_events x))::text`).Scan(&raw)
	if err != nil {
		f.h.t.Fatal(err)
	}
	f.h.t.Logf("independent_s412_image=%s", raw)
	return raw
}

func TestRecoveryRestoredPasswordResetPostgres(t *testing.T) {
	t.Run("TrustedApprovalAndDailyRejection", func(t *testing.T) {
		t.Run("RealND0ApprovedRoleChange", func(t *testing.T) {
			s412WithPG(t, func(f *s412PG) {
				if err := f.resolve(); err != nil {
					t.Fatalf("trusted ND0 restored: %v", err)
				}
				if f.secrets.calls != 1 || f.mail.calls != 1 {
					t.Fatal("new durable owner did not send exactly once")
				}
				f.assertRestored()
			})
		})
		s412ApprovalPGCases(t)
	})
	t.Run("OldAuthorityUnknownBeforeBoundaryAfter", func(t *testing.T) { s412UnknownPGCases(t); s412LegacyPGCases(t) })
	t.Run("CandidateAtomicAndCrossTargetCAS", func(t *testing.T) { s412CandidatePGCases(t); s412CounterPGCases(t); s412F001HistoryCases(t) })
	t.Run("DurableSingleSendAndAckLoss", func(t *testing.T) { s412DispatchPGCases(t); s412CandidateACKPGCase(t); s412FailedPointerPGCase(t) })
	t.Run("ControlledDueLeaseDriverNoMailer", func(t *testing.T) { s412DriverPGCases(t); s412DriverRefusalPGCases(t); s412ExpiredLeasePGCase(t) })
	t.Run("AcceptedAtomicRestoredAndSuspended", s412AcceptedPGCases)
	t.Run("TerminalWholeReadbackAndComplete", func(t *testing.T) { s412TerminalPGCases(t); s412TerminalIntentPGCases(t) })
}

func (f *s412PG) assertRestored() {
	f.h.t.Helper()
	var valid bool
	err := f.h.pool.QueryRow(f.h.ctx, `SELECT u.auth_generation=o.target_auth_generation AND u.credential_mode='temporary'
 AND NOT u.recovery_required AND u.role=o.recovery_target_role AND u.status=o.recovery_target_status
 AND u.password_reset_current_operation_id=o.operation_id AND c.state='active' AND c.source_operation_id=o.operation_id
 AND c.auth_generation=o.target_auth_generation AND v.decision='restored' AND v.credential_reset_at=o.settled_at
 AND v.reviewed_at=v.credential_reset_at AND v.reviewed_by=o.recovery_actor_id AND v.evidence_ref=o.recovery_evidence_ref
 AND j.status='succeeded' AND j.completed_at IS NOT NULL AND j.lease_owner IS NULL AND j.lease_expires_at IS NULL
 AND i.status='succeeded' AND i.result_code='password_reset_succeeded'
 AND i.response_snapshot=jsonb_build_object('outcome','succeeded','resource_id',o.operation_id::text)
 FROM users u JOIN password_reset_operations o ON o.operation_id=u.password_reset_current_operation_id
 JOIN temporary_credentials c ON c.id=o.temporary_credential_id JOIN recovery_account_reviews v ON v.restoration_operation_id=o.operation_id
 JOIN password_reset_recovery_jobs j ON j.operation_id=o.operation_id JOIN idempotency_records i ON i.password_reset_operation_id=o.operation_id
 WHERE u.id=$1`, f.request.TargetID).Scan(&valid)
	if err != nil || !valid {
		f.h.t.Fatalf("whole terminal SQL group: %v", err)
	}
	_ = f.image()
}

func s412ApprovalPGCases(t *testing.T) {
	for _, name := range []string{"missing-source", "version", "installation", "run", "actor", "target", "role", "status", "ref", "digest", "no-private-principal", "D0", "current-deleted"} {
		t.Run(name, func(t *testing.T) {
			s412WithPG(t, func(f *s412PG) {
				s412MutateApproval(f, name)
				before := f.image()
				var err error
				if name == "no-private-principal" {
					err = f.owner.ResolveRestored(f.ctx, f.p, f.request)
				} else {
					err = f.resolve()
				}
				if err == nil || f.secrets.calls != 0 || f.mail.calls != 0 || f.ids.calls.Load() != 0 || f.image() != before {
					t.Fatalf("invalid approval produced side effects: %s / %v", name, err)
				}
			})
		})
	}
	t.Run("DailyResetAndModeStillReject", func(t *testing.T) {
		s412WithPG(t, func(f *s412PG) {
			before := f.image()
			err := f.withActor(func(c context.Context) error {
				actor, _ := platform.PrincipalFromContext(c)
				_, err := f.owner.resets.ResetPassword(c, PasswordResetRequest{Actor: actor, TargetID: s410ND, TargetRole: RoleMember, IdempotencyKey: "ordinary-recovery-refusal", TraceID: "00000000000000000000000001"})
				return err
			})
			if err == nil || f.image() != before || f.secrets.calls != 0 || f.mail.calls != 0 {
				t.Fatal("ordinary reset weakened recovery eligibility")
			}
			tx, err := f.h.pool.Begin(f.ctx)
			if err != nil {
				t.Fatal(err)
			}
			err = (PostgresPasswordResetRepository{}).SetCredentialMode(f.ctx, tx, s410ND, CredentialModeTemporary, f.h.clock.Now())
			closeErr := finishAdmissionRead(f.ctx, tx)
			if err == nil || closeErr != nil || f.image() != before {
				t.Fatal("ordinary mode setter bypassed false predicate")
			}
		})
	})
}

func s412MutateApproval(f *s412PG, name string) {
	switch name {
	case "missing-source":
		f.reader.err = errors.New("synthetic approved source unavailable")
	case "version":
		f.reader.approved.Version = "other"
	case "installation":
		f.reader.approved.InstallationID = "other"
	case "run":
		f.reader.approved.RunID = "other"
	case "actor":
		f.reader.approved.ActorID = s410ND
	case "target":
		f.reader.approved.TargetID = s410D
	case "role":
		f.reader.approved.Role = RoleMember
	case "status":
		f.reader.approved.Status = UserStatusSuspended
	case "ref":
		f.reader.approved.EvidenceRef = "other"
	case "digest":
		f.reader.approved.EvidenceDigest = "not-a-trusted-sha256"
	case "D0":
		f.request.TargetID = s410D
		f.reader.approved.TargetID = s410D
	case "current-deleted":
		f.h.exec(`UPDATE users SET status='deleted',deleted_at=$2,email=NULL,phone=NULL WHERE id=$1`, s410ND, f.h.clock.Now())
	}
}

func s412UnknownPGCases(t *testing.T) {
	for _, delta := range []time.Duration{-time.Microsecond, 0, time.Microsecond} {
		t.Run(delta.String(), func(t *testing.T) {
			s412WithPG(t, func(f *s412PG) {
				f.mail.err = errors.New("synthetic transport acceptance unknown")
				if err := f.resolve(); err == nil {
					t.Fatal("unknown delivery returned restored success")
				}
				var due time.Time
				if err := f.h.pool.QueryRow(f.ctx, `SELECT next_attempt_at FROM password_reset_recovery_jobs`).Scan(&due); err != nil {
					t.Fatal(err)
				}
				f.h.clock.mu.Lock()
				f.h.clock.now = due.Add(delta)
				f.h.clock.mu.Unlock()
				err := f.owner.SettleRestored(f.ctx, f.p, s410Binding.RunID, s410ND)
				if err == nil {
					t.Fatal("unaccepted candidate settlement falsely returned restored 204")
				}
				var decision, state string
				var attempts int64
				err = f.h.pool.QueryRow(f.ctx, `SELECT v.decision,o.state,j.attempt_count FROM recovery_account_reviews v JOIN password_reset_operations o ON o.operation_id=v.restoration_operation_id JOIN password_reset_recovery_jobs j ON j.operation_id=o.operation_id WHERE v.user_id=$1`, s410ND).Scan(&decision, &state, &attempts)
				if err != nil || decision != "pending" || f.mail.calls != 1 || f.secrets.calls != 1 {
					t.Fatal("unknown window or once-send evidence changed")
				}
				if delta < 0 && (state != "commit_unknown" || attempts != 0) {
					t.Fatal("unknown settled before protection boundary")
				}
				if delta >= 0 && state != "invalidated" {
					t.Fatal("due nonacceptance did not legally settle original candidate")
				}
				_ = f.image()
			})
		})
	}
	t.Run("HoldNoNewCandidate", func(t *testing.T) {
		s412WithPG(t, func(f *s412PG) {
			f.h.exec(`UPDATE users SET auth_integrity_hold=true,auth_integrity_hold_reason='synthetic-prior-hold',auth_integrity_hold_at=$2 WHERE id=$1`, s410ND, f.h.clock.Now())
			before := f.image()
			if err := f.resolve(); err == nil || f.image() != before || f.secrets.calls != 0 || f.mail.calls != 0 {
				t.Fatal("prior hold cleared or new candidate created")
			}
		})
	})
}

func s412CandidatePGCases(t *testing.T) {
	t.Run("CandidateCommitRollbackWholeImage", func(t *testing.T) {
		s412WithPG(t, func(f *s412PG) {
			marker := errors.New("synthetic candidate rollback ACK")
			f.e.commit = func(c context.Context, tx pgx.Tx) error {
				if err := tx.Rollback(c); err != nil {
					return err
				}
				return marker
			}
			before := f.image()
			if err := f.resolve(); err == nil || f.image() != before || f.mail.calls != 0 {
				t.Fatal("candidate counter/claim/operation/credential/job/pointer half committed")
			}
		})
	})
	t.Run("CrossTargetCompositeFK", func(t *testing.T) {
		s412WithPG(t, func(f *s412PG) {
			f.mail.err = errors.New("synthetic unknown")
			if err := f.resolve(); err == nil {
				t.Fatal("unknown expected")
			}
			before := f.image()
			tx, err := f.h.pool.Begin(f.ctx)
			if err != nil {
				t.Fatal(err)
			}
			_, err = tx.Exec(f.ctx, `UPDATE recovery_account_reviews SET restoration_operation_id=(SELECT restoration_operation_id FROM recovery_account_reviews WHERE user_id=$1) WHERE user_id=$2`, s410ND, s410D)
			if err == nil {
				err = tx.Commit(f.ctx)
			}
			closeErr := finishAdmissionRead(f.ctx, tx)
			if err == nil || closeErr != nil || f.image() != before {
				t.Fatal("cross-target restoration pointer accepted")
			}
			t.Log("actual composite FK rejected; transaction ended and full beforeimage retained")
		})
	})
	s412ConcurrentPGCase(t)
}

func s412DispatchPGCases(t *testing.T) {
	t.Run("ReplayNoSecretIDOrSecondSend", func(t *testing.T) {
		s412WithPG(t, func(f *s412PG) {
			if err := f.resolve(); err != nil {
				t.Fatal(err)
			}
			before, ids := f.image(), f.ids.calls.Load()
			if err := f.resolve(); err != nil || f.image() != before || f.ids.calls.Load() != ids || f.secrets.calls != 1 || f.mail.calls != 1 {
				t.Fatalf("terminal retry was not read-only: %v", err)
			}
		})
	})
	t.Run("ActualFinalCommitLostACKWholeReadback", func(t *testing.T) {
		s412WithPG(t, func(f *s412PG) {
			s412FinalCommitFault(f, "ack")
			if err := f.resolve(); err != nil {
				t.Fatalf("proved final commit ACK loss: %v", err)
			}
			if f.mail.calls != 1 || f.secrets.calls != 1 {
				t.Fatal("ACK loss repeated dispatch")
			}
			f.assertRestored()
		})
	})
	t.Run("ExplicitRejectedKeepsLockedReview", func(t *testing.T) {
		s412WithPG(t, func(f *s412PG) {
			f.mail.err = mailer.ErrMessageRejected
			if err := f.resolve(); err == nil {
				t.Fatal("rejected SMTP restored user")
			}
			var valid bool
			err := f.h.pool.QueryRow(f.ctx, `SELECT u.recovery_required AND u.credential_mode='recovery_locked' AND v.decision='pending' AND o.state='failed' AND c.state='recovery_invalidated' AND j.status='succeeded' FROM users u JOIN recovery_account_reviews v ON v.user_id=u.id JOIN password_reset_operations o ON o.operation_id=v.restoration_operation_id JOIN temporary_credentials c ON c.id=o.temporary_credential_id JOIN password_reset_recovery_jobs j ON j.operation_id=o.operation_id WHERE u.id=$1`, s410ND).Scan(&valid)
			if err != nil || !valid || f.mail.calls != 1 {
				t.Fatal("rejected evidence invalidated wrong authority")
			}
			_ = f.image()
		})
	})
}

func s412FinalCommitFault(f *s412PG, mode string) {
	f.e.commit = func(c context.Context, tx pgx.Tx) error {
		var terminal bool
		if err := tx.QueryRow(c, `SELECT EXISTS(SELECT 1 FROM recovery_account_reviews WHERE user_id=$1 AND decision='restored')`, s410ND).Scan(&terminal); err != nil {
			return err
		}
		if !terminal {
			return tx.Commit(c)
		}
		if mode == "rollback" {
			if err := tx.Rollback(c); err != nil {
				return err
			}
			return errors.New("synthetic final rollback ACK")
		}
		if err := tx.Commit(c); err != nil {
			return err
		}
		if mode == "partial" {
			f.h.exec(`UPDATE users SET role='member' WHERE id=$1`, s410ND)
		}
		return errors.New("synthetic actual final COMMIT ACK loss")
	}
}

func s412DriverPGCases(t *testing.T) {
	t.Run("DurableActorNoSessionNoMailerDueDriver", func(t *testing.T) {
		s412WithPG(t, func(f *s412PG) {
			s412DeferAcceptedJob(f)
			if err := f.resolve(); err == nil {
				t.Fatal("not-due accepted candidate returned early 204")
			}
			f.owner.resets = nil
			f.h.exec(`UPDATE sessions SET revoked_at=$2,revoke_reason='synthetic-actor-session-gone' WHERE user_id=$1 AND revoked_at IS NULL`, s410Initial, f.h.clock.Now())
			var due time.Time
			if err := f.h.pool.QueryRow(f.ctx, `SELECT next_attempt_at FROM password_reset_recovery_jobs`).Scan(&due); err != nil {
				t.Fatal(err)
			}
			f.h.clock.mu.Lock()
			f.h.clock.now = due
			f.h.clock.mu.Unlock()
			if err := f.owner.SettleRestored(f.ctx, f.p, s410Binding.RunID, s410ND); err != nil {
				t.Fatalf("no-mailer controlled driver: %v", err)
			}
			if f.mail.calls != 1 || f.secrets.calls != 1 {
				t.Fatal("settlement required request sender or new secret")
			}
			f.assertRestored()
		})
	})
	s412OrdinaryAcceptedPGCase(t)
}

func s412DeferAcceptedJob(f *s412PG) {
	armed := true
	f.e.commit = func(c context.Context, tx pgx.Tx) error {
		var accepted bool
		if err := tx.QueryRow(c, `SELECT EXISTS(SELECT 1 FROM password_reset_operations WHERE delivery_state='accepted')`).Scan(&accepted); err != nil {
			return err
		}
		if armed && accepted {
			armed = false
			if _, err := tx.Exec(c, `UPDATE password_reset_recovery_jobs SET next_attempt_at=$1`, f.h.clock.Now().Add(time.Second)); err != nil {
				return err
			}
		}
		return tx.Commit(c)
	}
}

func s412AcceptedPGCases(t *testing.T) {
	for _, status := range []UserStatus{UserStatusActive, UserStatusSuspended} {
		t.Run(string(status), func(t *testing.T) {
			s412WithPG(t, func(f *s412PG) {
				f.request.Status = status
				f.reader.approved.Status = status
				if err := f.resolve(); err != nil {
					t.Fatal(err)
				}
				f.assertRestored()
			})
		})
	}
	t.Run("RecoveryReviewAuditRollback", func(t *testing.T) {
		s412WithPG(t, func(f *s412PG) {
			f.h.audit.set(recoveryReviewAction)
			if err := f.resolve(); err == nil {
				t.Fatal("failed review Audit returned 204")
			}
			var valid bool
			err := f.h.pool.QueryRow(f.ctx, `SELECT u.recovery_required AND u.credential_mode='recovery_locked' AND v.decision='pending' AND o.state='recovery_required' AND c.state='pending_delivery' AND i.status='processing' FROM users u JOIN recovery_account_reviews v ON v.user_id=u.id JOIN password_reset_operations o ON o.operation_id=v.restoration_operation_id JOIN temporary_credentials c ON c.id=o.temporary_credential_id JOIN idempotency_records i ON i.password_reset_operation_id=o.operation_id WHERE u.id=$1`, s410ND).Scan(&valid)
			if err != nil || !valid {
				t.Fatal("review Audit failure partially activated restored user")
			}
			_ = f.image()
		})
	})
	t.Run("DefiniteFinalRollbackNoPartialRestoration", func(t *testing.T) {
		s412WithPG(t, func(f *s412PG) {
			s412FinalCommitFault(f, "rollback")
			if err := f.resolve(); err == nil {
				t.Fatal("rolled-back settlement returned 204")
			}
			var valid bool
			err := f.h.pool.QueryRow(f.ctx, `SELECT u.recovery_required AND u.credential_mode='recovery_locked' AND v.decision='pending' AND c.state='pending_delivery' FROM users u JOIN recovery_account_reviews v ON v.user_id=u.id JOIN password_reset_operations o ON o.operation_id=v.restoration_operation_id JOIN temporary_credentials c ON c.id=o.temporary_credential_id WHERE u.id=$1`, s410ND).Scan(&valid)
			if err != nil || !valid {
				t.Fatal("final rollback left partial committed capability")
			}
			_ = f.image()
		})
	})
}

func s412TerminalPGCases(t *testing.T) {
	t.Run("WholeUserMismatchNotOperationOnlySuccess", func(t *testing.T) {
		s412WithPG(t, func(f *s412PG) {
			s412FinalCommitFault(f, "partial")
			if err := f.resolve(); err == nil {
				t.Fatal("operation succeeded hid inconsistent User/approval")
			}
			before, ids := f.image(), f.ids.calls.Load()
			if err := f.resolve(); err == nil || f.image() != before || f.ids.calls.Load() != ids || f.mail.calls != 1 || f.secrets.calls != 1 {
				t.Fatal("partial terminal blindly rewrote User or repeated send")
			}
		})
	})
	t.Run("TerminalWrongIntentAndCompleteWholeCollection", func(t *testing.T) {
		s412WithPG(t, func(f *s412PG) {
			if err := f.resolve(); err != nil {
				t.Fatal(err)
			}
			before, ids := f.image(), f.ids.calls.Load()
			if err := f.resolve(); err != nil || f.image() != before || f.ids.calls.Load() != ids {
				t.Fatal("legal terminal sameintent not read-only")
			}
			f.request.IdempotencyKey = "synthetic-wrong-terminal-key"
			if err := f.resolve(); err == nil || f.image() != before {
				t.Fatal("terminal wrongkey accepted")
			}
			if err := f.e.CompleteIdentityReview(f.ctx, f.p, s410Binding); err == nil {
				t.Fatal("one restored user hid other pending D0 review")
			}
			if err := f.e.ResolveRestrictedAccountReview(f.ctx, f.p, s410Binding, s410D, "denied", "synthetic-D0-denied-evidence"); err != nil {
				t.Fatal(err)
			}
			if err := f.e.CompleteIdentityReview(f.ctx, f.p, s410Binding); err != nil {
				t.Fatalf("complete original C/eachP collection: %v", err)
			}
			if f.h.j.consumes != 0 {
				t.Fatal("component Complete consumed/opened global runtime")
			}
			_ = f.image()
		})
	})
}

func s412LegacyPGCases(t *testing.T) {
	for _, delta := range []time.Duration{-time.Microsecond, 0, time.Microsecond} {
		t.Run("legacy-"+delta.String(), func(t *testing.T) {
			s412WithPG(t, func(f *s412PG) {
				f.h.linkedResetFixture("unknown")
				due := f.h.clock.Now().Add(time.Second)
				f.h.exec(`UPDATE password_reset_recovery_jobs SET next_attempt_at=$1`, due)
				f.h.clock.mu.Lock()
				f.h.clock.now = due.Add(delta)
				f.h.clock.mu.Unlock()
				before := f.image()
				if err := f.resolve(); err == nil || f.image() != before || f.secrets.calls != 0 || f.mail.calls != 0 || f.ids.calls.Load() != 0 {
					t.Fatal("old unresolved authority admitted new candidate")
				}
				if err := f.owner.SettleRestored(f.ctx, f.p, s410Binding.RunID, s410ND); err == nil {
					t.Fatal("old nonaccepted settlement falsely reported restored 204")
				}
				var state, decision string
				err := f.h.pool.QueryRow(f.ctx, `SELECT o.state,v.decision FROM password_reset_operations o JOIN recovery_account_reviews v ON v.user_id=o.user_id`).Scan(&state, &decision)
				if err != nil || decision != "pending" {
					t.Fatal("old settlement changed review")
				}
				if delta < 0 && (state != "commit_unknown" || f.image() != before) {
					t.Fatal("old unknown protection shortened")
				}
				if delta >= 0 && state != "invalidated" {
					t.Fatal("old due evidence did not legally terminate")
				}
				if f.secrets.calls != 0 || f.mail.calls != 0 {
					t.Fatal("legacy driver dispatched")
				}
				if delta >= 0 {
					s412OriginalFailureProof(f)
				}
				_ = f.image()
			})
		})
	}
}

func s412CounterPGCases(t *testing.T) {
	for _, column := range []string{"auth_generation", "credential_generation", "password_reset_execution_generation"} {
		t.Run("overflow-"+column, func(t *testing.T) {
			s412WithPG(t, func(f *s412PG) {
				if column == "password_reset_execution_generation" {
					s412RejectCounterGap(f)
					return
				}
				if column == "auth_generation" {
					f.h.exec(`UPDATE users SET auth_generation=9223372036854775807 WHERE id=$1`, s410ND)
				} else {
					f.h.linkedResetFixture("invalidated")
					f.h.exec(`UPDATE temporary_credentials SET generation=9223372036854775807 WHERE user_id=$1`, s410ND)
				}
				before := f.image()
				if err := f.resolve(); err == nil || f.image() != before || f.secrets.calls != 0 || f.ids.calls.Load() != 0 || f.mail.calls != 0 {
					t.Fatal("generation overflow allocated candidate or secret")
				}
			})
		})
	}
}

func s412DriverRefusalPGCases(t *testing.T) {
	for _, kind := range []string{"live-lease", "max-retries", "dead-letter", "source-changed", "current-deleted"} {
		t.Run(kind, func(t *testing.T) {
			s412WithPG(t, func(f *s412PG) {
				s412DeferAcceptedJob(f)
				if err := f.resolve(); err == nil {
					t.Fatal("expected deferred candidate")
				}
				f.e.commit = func(c context.Context, tx pgx.Tx) error { return tx.Commit(c) }
				var due time.Time
				if err := f.h.pool.QueryRow(f.ctx, `SELECT next_attempt_at FROM password_reset_recovery_jobs`).Scan(&due); err != nil {
					t.Fatal(err)
				}
				f.h.clock.mu.Lock()
				f.h.clock.now = due
				f.h.clock.mu.Unlock()
				switch kind {
				case "live-lease":
					f.h.exec(`UPDATE password_reset_recovery_jobs SET status='leased',lease_owner='synthetic-other-owner',lease_expires_at=$1`, due.Add(time.Minute))
				case "max-retries":
					f.h.exec(`UPDATE password_reset_recovery_jobs SET attempt_count=3`)
				case "dead-letter":
					f.h.exec(`UPDATE password_reset_recovery_jobs SET status='dead_letter'`)
				case "source-changed":
					f.reader.approved.EvidenceDigest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
				case "current-deleted":
					f.h.exec(`UPDATE users SET status='deleted',deleted_at=$2,email=NULL,phone=NULL WHERE id=$1`, s410ND, due)
				}
				before, ids := f.image(), f.ids.calls.Load()
				f.owner.resets = nil
				if err := f.owner.SettleRestored(f.ctx, f.p, s410Binding.RunID, s410ND); err == nil || f.image() != before || f.ids.calls.Load() != ids || f.mail.calls != 1 || f.secrets.calls != 1 {
					t.Fatal("driver bypassed lease/source/eligibility or created authority")
				}
			})
		})
	}
}

func s412TerminalIntentPGCases(t *testing.T) {
	for _, kind := range []string{"role", "status", "ref", "digest", "D0-request"} {
		t.Run("terminal-"+kind, func(t *testing.T) {
			s412WithPG(t, func(f *s412PG) {
				if err := f.resolve(); err != nil {
					t.Fatal(err)
				}
				switch kind {
				case "role":
					f.request.Role = RoleMember
					f.reader.approved.Role = RoleMember
				case "status":
					f.request.Status = UserStatusSuspended
					f.reader.approved.Status = UserStatusSuspended
				case "ref":
					f.request.EvidenceRef = "synthetic-other-ref"
					f.reader.approved.EvidenceRef = f.request.EvidenceRef
				case "digest":
					f.reader.approved.EvidenceDigest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
				case "D0-request":
					f.request.TargetID = s410D
					f.reader.approved.TargetID = s410D
				}
				before, ids := f.image(), f.ids.calls.Load()
				if err := f.resolve(); err == nil || f.image() != before || f.ids.calls.Load() != ids || f.mail.calls != 1 || f.secrets.calls != 1 {
					t.Fatal("terminal changed intent/source was accepted or rewritten")
				}
			})
		})
	}
}

func s412RejectCounterGap(f *s412PG) {
	t := f.h.t
	t.Helper()
	f.h.linkedResetFixture("invalidated")
	before := f.image()
	tx, err := f.h.pool.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(f.ctx, `UPDATE users SET password_reset_execution_generation=9223372036854775807 WHERE id=$1`, s410ND)
	if err == nil {
		err = tx.Commit(f.ctx)
	}
	closeErr := finishAdmissionRead(f.ctx, tx)
	var pgerr *pgconn.PgError
	if !errors.As(err, &pgerr) || (pgerr.Code != "23503" && pgerr.Code != "55000") || closeErr != nil || f.image() != before {
		t.Fatal("counter gap bypassed durable authority constraint")
	}
	if f.secrets.calls != 0 || f.ids.calls.Load() != 0 || f.mail.calls != 0 {
		t.Fatal("counter rejection created authority")
	}
	t.Logf("actual counter-max fixture rejected by original durable constraint SQLSTATE=%s; application max predicate separately covered by U", pgerr.Code)
}

func s412ConcurrentPGCase(t *testing.T) {
	t.Run("SameKeyConcurrentNewOwnerOnlyOnce", func(t *testing.T) {
		s412WithPG(t, func(f *s412PG) {
			_ = f.withActor(func(c context.Context) error {
				joined := make(chan error, 2)
				var workers sync.WaitGroup
				workers.Add(2)
				for i := 0; i < 2; i++ {
					go func() { defer workers.Done(); joined <- f.owner.ResolveRestored(c, f.p, f.request) }()
				}
				for i := 0; i < 2; i++ {
					_ = <-joined
				}
				workers.Wait()
				close(joined)
				return nil
			})
			if err := f.resolve(); err != nil {
				t.Fatalf("stable same-key retry: %v", err)
			}
			if f.secrets.calls != 1 || f.mail.calls != 1 {
				t.Fatal("concurrent same-key candidates generated or sent twice")
			}
			f.assertRestored()
		})
	})
}

func s412OrdinaryAcceptedPGCase(t *testing.T) {
	t.Run("OrdinaryAcceptedSetterStillRejectsRecoveryLocked", func(t *testing.T) {
		s412WithPG(t, func(f *s412PG) {
			s412DeferAcceptedJob(f)
			if err := f.resolve(); err == nil {
				t.Fatal("expected deferred candidate")
			}
			var id string
			var due time.Time
			if err := f.h.pool.QueryRow(f.ctx, `SELECT operation_id::text,next_attempt_at FROM password_reset_recovery_jobs`).Scan(&id, &due); err != nil {
				t.Fatal(err)
			}
			f.h.clock.mu.Lock()
			f.h.clock.now = due
			f.h.clock.mu.Unlock()
			backend := passwordResetPostgresBackend{pool: f.h.pool, store: f.owner.settlement.store, clock: f.h.clock}
			lease, claimed, err := backend.Claim(f.ctx, PasswordResetRecoveryCandidate{OperationID: id, UserID: s410ND}, "synthetic-ordinary-owner", due, time.Minute)
			if err != nil || !claimed {
				t.Fatal("ordinary lease fixture")
			}
			before := f.image()
			err = f.owner.settlement.store.settlePasswordResetRecovery(f.ctx, f.h.pool, f.h.audit, f.ids, f.h.clock, lease)
			if err == nil || f.image() != before {
				t.Fatal("ordinary accepted settlement bypassed locked mode")
			}
		})
	})
}

func s412CandidateACKPGCase(t *testing.T) {
	t.Run("ActualCandidateCommitLostACKResumeNeverResends", func(t *testing.T) {
		s412WithPG(t, func(f *s412PG) {
			armed := true
			f.e.commit = func(c context.Context, tx pgx.Tx) error {
				var pending bool
				if err := tx.QueryRow(c, `SELECT EXISTS(SELECT 1 FROM password_reset_operations WHERE state='prepared')`).Scan(&pending); err != nil {
					return err
				}
				if err := tx.Commit(c); err != nil {
					return err
				}
				if armed && pending {
					armed = false
					return errors.New("synthetic actual candidate COMMIT ACK loss")
				}
				return nil
			}
			if err := f.resolve(); err != nil {
				t.Fatal(err)
			}
			before, ids := f.image(), f.ids.calls.Load()
			if err := f.resolve(); err != nil || f.image() != before || f.ids.calls.Load() != ids || f.mail.calls != 1 || f.secrets.calls != 1 {
				t.Fatal("candidate ACK/resume repeated ownership or dispatch")
			}
			f.assertRestored()
		})
	})
}

func s412FailedPointerPGCase(t *testing.T) {
	t.Run("FailedPointerPreservedThenNewKeyReplacesAtomically", func(t *testing.T) {
		s412WithPG(t, func(f *s412PG) {
			f.mail.err = mailer.ErrMessageRejected
			if err := f.resolve(); err == nil {
				t.Fatal("expected delivery rejection")
			}
			before, ids := f.image(), f.ids.calls.Load()
			var old string
			if err := f.h.pool.QueryRow(f.ctx, `SELECT restoration_operation_id::text FROM recovery_account_reviews WHERE user_id=$1`, s410ND).Scan(&old); err != nil {
				t.Fatal(err)
			}
			history := s412History(f, old)
			if err := f.resolve(); err == nil || f.image() != before || f.ids.calls.Load() != ids || f.mail.calls != 1 {
				t.Fatal("failed same-key replay erased pointer or sent")
			}
			f.request.IdempotencyKey = "synthetic-new-key-after-proved-rejection"
			f.mail.err = nil
			if err := f.resolve(); err != nil {
				t.Fatal(err)
			}
			if s412History(f, old) != history || f.secrets.calls != 2 || f.mail.calls != 2 {
				t.Fatal("new root erased failed operation/credential history")
			}
			f.assertRestored()
		})
	})
}

func s412History(f *s412PG, id string) string {
	f.h.t.Helper()
	var raw string
	err := f.h.pool.QueryRow(f.ctx, `SELECT jsonb_build_array(to_jsonb(o),to_jsonb(c),to_jsonb(j),to_jsonb(i))::text FROM password_reset_operations o JOIN temporary_credentials c ON c.id=o.temporary_credential_id JOIN password_reset_recovery_jobs j ON j.operation_id=o.operation_id JOIN idempotency_records i ON i.password_reset_operation_id=o.operation_id WHERE o.operation_id=$1`, id).Scan(&raw)
	if err != nil {
		f.h.t.Fatal(err)
	}
	return raw
}

func s412ExpiredLeasePGCase(t *testing.T) {
	t.Run("ExpiredLeaseBoundaryReclaimsNextAttempt", func(t *testing.T) {
		s412WithPG(t, func(f *s412PG) {
			s412DeferAcceptedJob(f)
			if err := f.resolve(); err == nil {
				t.Fatal("expected deferred candidate")
			}
			f.e.commit = func(c context.Context, tx pgx.Tx) error { return tx.Commit(c) }
			due := f.h.clock.Now().Add(time.Second)
			f.h.exec(`UPDATE password_reset_recovery_jobs SET status='leased',attempt_count=1,lease_owner='synthetic-expired-owner',lease_expires_at=$1`, due)
			f.h.clock.mu.Lock()
			f.h.clock.now = due
			f.h.clock.mu.Unlock()
			f.owner.resets = nil
			if err := f.owner.SettleRestored(f.ctx, f.p, s410Binding.RunID, s410ND); err != nil {
				t.Fatal(err)
			}
			var attempts int64
			if err := f.h.pool.QueryRow(f.ctx, `SELECT attempt_count FROM password_reset_recovery_jobs`).Scan(&attempts); err != nil || attempts != 2 {
				t.Fatal("expired lease lost original attempt counter")
			}
			if f.secrets.calls != 1 || f.mail.calls != 1 {
				t.Fatal("lease retry dispatched")
			}
			f.assertRestored()
		})
	})
}

func s412OriginalFailureProof(f *s412PG) {
	t := f.h.t
	t.Helper()
	tx, err := f.h.pool.BeginTx(f.ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := finishAdmissionRead(f.ctx, tx); err != nil {
			t.Error(err)
		}
	}()
	var id string
	if err := tx.QueryRow(f.ctx, `SELECT password_reset_current_operation_id::text FROM users WHERE id=$1`, s410ND).Scan(&id); err != nil {
		t.Fatal(err)
	}
	op, err := recoveryReadResetOperation(f.ctx, tx, id)
	if err != nil {
		t.Fatal(err)
	}
	if err = recoveryFailedResetSettlement(f.ctx, tx, op); err != nil {
		t.Fatalf("original failed-reset consumer cannot prove legacy terminal: %v", err)
	}
	if err = tx.Commit(f.ctx); err != nil {
		t.Fatal(err)
	}
}

// These fixtures call the real aggregate/repository/credential lifecycle
// producers before Finalize. They do not claim creation HTTP/provider coverage.
func s412F001HistoryCases(t *testing.T) {
	for _, kind := range []string{"consumed-admin", "active-admin", "resend-active-admin"} {
		t.Run("AccountWideGeneration/"+kind, func(t *testing.T) {
			s412F001WithHistory(t, kind, func(f *s412PG, maximum int64) {
				oldRows := s412F001OldRows(f.h, f.request.TargetID)
				before := f.image()
				if err := f.resolve(); err != nil {
					var sqlError *pgconn.PgError
					if errors.As(err, &sqlError) {
						t.Logf("actual SQLSTATE=%s constraint=%s", sqlError.Code, sqlError.ConstraintName)
					}
					t.Logf("history resolve before=%s after=%s send=%d", before, f.image(), f.mail.calls)
					t.Fatalf("lawful administrator history must restore: %v", err)
				}
				var generation int64
				if err := f.h.pool.QueryRow(f.ctx, `SELECT c.generation FROM recovery_account_reviews v JOIN password_reset_operations o ON o.operation_id=v.restoration_operation_id JOIN temporary_credentials c ON c.id=o.temporary_credential_id WHERE v.user_id=$1`, f.request.TargetID).Scan(&generation); err != nil {
					t.Fatal(err)
				}
				if generation != maximum+1 || f.mail.calls != 1 || f.secrets.calls != 1 || s412F001OldRows(f.h, f.request.TargetID) != oldRows {
					t.Fatal("account-wide allocation rewrote old history or repeated dispatch")
				}
				f.assertRestored()
				terminal, ids := f.image(), f.ids.calls.Load()
				if err := f.resolve(); err != nil || f.image() != terminal || f.ids.calls.Load() != ids || f.mail.calls != 1 || f.secrets.calls != 1 {
					t.Fatal("history-bearing terminal replay allocated or dispatched")
				}
			})
		})
	}
}

func s412F001WithHistory(t *testing.T, kind string, run func(*s412PG, int64)) {
	h := s411PGHarness(t)
	raw, err := os.ReadFile(filepath.Join("..", "..", "migrations", "0011_article_replay_publication.sql"))
	if err != nil {
		t.Fatal(err)
	}
	h.exec(string(raw))
	target, maximum := s412F001ProduceHistory(h, kind)
	t.Logf("history_before_finalize=%s", s412F001OldRows(h, target))
	h.running()
	s411Phase(h, func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		passwords := s411Trusted(t, h, c, p, e)
		initial, err := s411Login(t, h, c, e, passwords, "synthetic-secret")
		if err != nil {
			t.Fatal(err)
		}
		changed, err := s411Change(t, h, c, p, e, passwords, initial)
		if err != nil {
			t.Fatal(err)
		}
		f := s412NewPG(t, h, c, p, e, passwords)
		f.actor = LoginResult{SessionToken: changed.SessionToken, CSRFToken: changed.CSRFToken}
		f.request.TargetID = target
		f.reader.approved.TargetID = target
		t.Logf("history_after_finalize=%s", s412F001OldRows(h, target))
		run(f, maximum)
	})
}

func s412F001ProduceHistory(h *s410PG, kind string) (string, int64) {
	t := h.t
	t.Helper()
	passwords := s411Hasher(t)
	creator := &AdministratorCreateCoordinator{dependencies: AdministratorCreateDependencies{Repository: PostgresAdministratorRepository{}, PasswordPolicy: security.PasswordPolicy{}, PasswordHasher: passwords, IDs: s410IDs{}, Clock: h.clock, IdempotencyKeys: idempotency.DigestKeyring{ActiveVersion: "synthetic-f001", Keys: map[string][]byte{"synthetic-f001": bytes.Repeat([]byte{0x45}, 32)}}}}
	actor := session.Principal{UserID: s410Initial, Role: "admin", IsInitialAdmin: true}
	q := CreateAdministratorRequest{Principal: actor, TemporaryPassword: "SyntheticCreator123", TraceID: "00000000000000000000000001"}
	normalized := normalizedCreateAdministrator{username: "historyadmin", normalizedUser: "historyadmin", email: "historyadmin@example.invalid", normalizedEmail: "historyadmin@example.invalid", phone: "+12025550199", noticeVersion: "synthetic-f001-notice"}
	aggregate, err := creator.newAdministratorAggregate(q, normalized)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := h.pool.Begin(h.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := finishAdmissionRead(h.ctx, tx); err != nil {
			t.Error(err)
		}
	}()
	if err = creator.persistPendingAdministrator(h.ctx, tx, aggregate); err != nil {
		t.Fatal(err)
	}
	repo := PostgresAdministratorRepository{}
	if err = repo.ActivateTemporaryCredential(h.ctx, tx, aggregate.credential.ID, h.clock.Now()); err != nil {
		t.Fatal(err)
	}
	current, err := TransitionTemporaryCredential(aggregate.credential, TemporaryCredentialActive, h.clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err = h.audit.Append(h.ctx, tx, aggregate.auditEvent); err != nil {
		t.Fatal(err)
	}
	if kind == "resend-active-admin" {
		current = s412F001Resend(h, tx, creator, actor, aggregate.user, current)
	}
	if kind == "consumed-admin" {
		s412F001Consume(h, tx, passwords, aggregate.user.ID, current.ID)
	}
	if err = tx.Commit(h.ctx); err != nil {
		t.Fatal(err)
	}
	return aggregate.user.ID, current.Generation
}

func s412F001Resend(h *s410PG, tx pgx.Tx, c *AdministratorCreateCoordinator, actor session.Principal, user User, current TemporaryCredential) TemporaryCredential {
	t := h.t
	t.Helper()
	prepared, err := c.prepareAdministratorCredentialResend(TemporaryCredentialResendRequest{Actor: actor, TraceID: "00000000000000000000000001"}, user, current, idempotency.BeginRequest{}, idempotency.Record{CreatedAt: h.clock.Now()}, &s412Secret{})
	if err != nil {
		t.Fatal(err)
	}
	old, active, err := ActivateTemporaryCredentialReplacement(current, prepared.pending, h.clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	repo := PostgresAdministratorRepository{}
	if err = repo.UpdateTemporaryCredential(h.ctx, tx, old); err != nil {
		t.Fatal(err)
	}
	if err = repo.InsertTemporaryCredential(h.ctx, tx, prepared.pending); err != nil {
		t.Fatal(err)
	}
	if err = repo.ActivateTemporaryCredential(h.ctx, tx, active.ID, h.clock.Now()); err != nil {
		t.Fatal(err)
	}
	if err = h.audit.Append(h.ctx, tx, prepared.auditEvent); err != nil {
		t.Fatal(err)
	}
	return active
}

func s412F001Consume(h *s410PG, tx pgx.Tx, passwords *s411Passwords, user, credential string) {
	t := h.t
	t.Helper()
	repo := PostgresPasswordChangeRepository{}
	locked, err := repo.FindUserForPasswordChange(h.ctx, tx, user)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := passwords.Hash("SyntheticOnboarded123")
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.SetPermanentPassword(h.ctx, tx, user, digest, CredentialModePermanent, locked.AuthGeneration, locked.AuthGeneration+1, nil, nil, h.clock.Now()); err != nil {
		t.Fatal(err)
	}
	if err = repo.ConsumeTemporaryCredential(h.ctx, tx, credential, h.clock.Now()); err != nil {
		t.Fatal(err)
	}
}

func s412F001OldRows(h *s410PG, target string) string {
	h.t.Helper()
	var rows string
	err := h.pool.QueryRow(h.ctx, `SELECT COALESCE(jsonb_agg(to_jsonb(c) ORDER BY generation),'[]')::text FROM temporary_credentials c WHERE user_id=$1 AND purpose='new_administrator'`, target).Scan(&rows)
	if err != nil {
		h.t.Fatal(err)
	}
	return rows
}
