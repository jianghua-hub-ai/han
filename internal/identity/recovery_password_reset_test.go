package identity

import (
	"bytes"
	"context"
	"errors"
	"han/internal/idempotency"
	"han/internal/recovery"
	"math"
	"testing"
	"time"
)

func s412ApprovalFixture() (RecoveryReviewApproval, recovery.PermitBinding, RecoveryRestoreRequest) {
	p := recovery.PermitBinding{RunID: "synthetic-run", InstallationID: "synthetic-install", RuntimeID: "synthetic-runtime", Generation: 1}
	q := RecoveryRestoreRequest{TargetID: "22222222-2222-4222-8222-222222222222", Role: RoleAdmin, Status: UserStatusSuspended, EvidenceRef: "synthetic-approval", IdempotencyKey: "synthetic-restore-key"}
	a := RecoveryReviewApproval{Version: "han-recovery-review/v1", InstallationID: p.InstallationID, RunID: p.RunID, TargetID: q.TargetID, ActorID: "11111111-1111-4111-8111-111111111111", Role: q.Role, Status: q.Status, EvidenceRef: q.EvidenceRef, EvidenceDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	return a, p, q
}

func TestRecoveryRestoredPasswordReset(t *testing.T) {
	t.Run("TrustedApprovalAndDailyRejection", s412ApprovalCases)
	t.Run("CandidateAtomicAndCrossTargetCAS", s412F001GenerationGuards)
	t.Run("OldAuthorityUnknownBeforeBoundaryAfter", s412TargetCases)
	t.Run("DurableSingleSendAndAckLoss", s412ClaimIdentityCases)
	t.Run("ControlledDueLeaseDriverNoMailer", s412LeaseCases)
}

func s412ApprovalCases(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*RecoveryReviewApproval)
	}{
		{"exact-approved-source", func(*RecoveryReviewApproval) {}},
		{"version", func(a *RecoveryReviewApproval) { a.Version = "other" }},
		{"installation", func(a *RecoveryReviewApproval) { a.InstallationID = "other" }},
		{"run", func(a *RecoveryReviewApproval) { a.RunID = "other" }},
		{"target", func(a *RecoveryReviewApproval) { a.TargetID = "33333333-3333-4333-8333-333333333333" }},
		{"actor", func(a *RecoveryReviewApproval) { a.ActorID = "33333333-3333-4333-8333-333333333333" }},
		{"role", func(a *RecoveryReviewApproval) { a.Role = RoleMember }},
		{"status", func(a *RecoveryReviewApproval) { a.Status = UserStatusActive }},
		{"ref", func(a *RecoveryReviewApproval) { a.EvidenceRef = "other" }},
		{"digest-missing", func(a *RecoveryReviewApproval) { a.EvidenceDigest = "" }},
		{"digest-noncanonical", func(a *RecoveryReviewApproval) { a.EvidenceDigest = "sha256:ABC" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, p, q := s412ApprovalFixture()
			actor := a.ActorID
			tc.mutate(&a)
			if got := recoveryRestoreApprovalValid(a, p, q, actor); got != (tc.name == "exact-approved-source") {
				t.Fatalf("trusted source tuple accepted=%t", got)
			}
		})
	}
	t.Run("ordinary-recovery-target-rejected", func(t *testing.T) {
		u, _ := s412TargetFixture()
		if passwordResetTargetEligible(u) {
			t.Fatal("ordinary reset admitted recovery target")
		}
	})
}

func s412TargetFixture() (User, recoveryExecutionReview) {
	email := "synthetic@example.invalid"
	u := User{ID: "22222222-2222-4222-8222-222222222222", Email: &email, Role: RoleMember, Status: UserStatusSuspended, RecoveryRequired: true, CredentialMode: CredentialModeRecoveryLocked, AuthGeneration: 1}
	return u, recoveryExecutionReview{UserID: u.ID, BackupRole: "member", BackupStatus: "active", Decision: "pending"}
}

func s412TargetCases(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*User, *recoveryExecutionReview)
	}{
		{"clean-ND0", func(*User, *recoveryExecutionReview) {}},
		{"D0", func(_ *User, r *recoveryExecutionReview) { r.BackupStatus = "deleted" }},
		{"current-deleted", func(u *User, _ *recoveryExecutionReview) { u.Status = UserStatusDeleted }},
		{"hold", func(u *User, _ *recoveryExecutionReview) { u.AuthIntegrityHold = true }},
		{"initial", func(u *User, _ *recoveryExecutionReview) { u.IsInitialAdmin = true }},
		{"not-locked", func(u *User, _ *recoveryExecutionReview) { u.CredentialMode = CredentialModePermanent }},
		{"not-required", func(u *User, _ *recoveryExecutionReview) { u.RecoveryRequired = false }},
		{"terminal-not-pending", func(_ *User, r *recoveryExecutionReview) { r.Decision = "restored" }},
		{"auth-overflow", func(u *User, _ *recoveryExecutionReview) { u.AuthGeneration = math.MaxInt64 }},
		{"counter-overflow", func(u *User, _ *recoveryExecutionReview) { u.PasswordResetExecutionGeneration = math.MaxInt64 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u, r := s412TargetFixture()
			tc.mutate(&u, &r)
			if recoveryRestoreTargetEligible(u, r) != (tc.name == "clean-ND0") {
				t.Fatal("candidate eligibility does not preserve immutable source/hold/overflow")
			}
		})
	}
}

func s412LeaseCases(t *testing.T) {
	now := time.Date(2026, 10, 30, 3, 0, 0, 0, time.UTC)
	for _, delta := range []time.Duration{-time.Nanosecond, 0, time.Nanosecond} {
		t.Run(delta.String(), func(t *testing.T) {
			due := now.Add(delta)
			if recoveryRestoreLeaseDue(PasswordResetRecoveryJobReady, 0, due, nil, now, 3) != (delta <= 0) {
				t.Fatal("due before/boundary/after")
			}
		})
	}
	t.Run("lease-expiry-boundary", func(t *testing.T) {
		if !recoveryRestoreLeaseDue(PasswordResetRecoveryJobLeased, 1, now, &now, now, 3) {
			t.Fatal("expired lease not reclaimable at boundary")
		}
	})
	t.Run("live-lease", func(t *testing.T) {
		end := now.Add(time.Second)
		if recoveryRestoreLeaseDue(PasswordResetRecoveryJobLeased, 1, now, &end, now, 3) {
			t.Fatal("live lease stolen")
		}
	})
	t.Run("max-retries", func(t *testing.T) {
		if recoveryRestoreLeaseDue(PasswordResetRecoveryJobRetryWait, 3, now, nil, now, 3) {
			t.Fatal("retry limit bypass")
		}
	})
	t.Run("dead-letter", func(t *testing.T) {
		if recoveryRestoreLeaseDue(PasswordResetRecoveryJobDeadLetter, 1, now, nil, now, 3) {
			t.Fatal("dead letter reopened")
		}
	})
}

func s412ClaimIdentityCases(t *testing.T) {
	for _, field := range []string{"same-replay", "run", "install", "actor", "target", "role", "status", "evidence"} {
		t.Run(field, func(t *testing.T) {
			a, p, q := s412ApprovalFixture()
			d := PasswordResetDependencies{Clock: passwordResetClockStub{}, IdempotencyKeys: passwordResetStoreTestCandidate(t).IdempotencyRequest.DigestKeys, IdempotencyLock: time.Minute, IdempotencyTTL: time.Hour}
			x := recoveryRestoreAttempt{owner: &recoveryRestoreReset{resets: &PasswordResetCoordinator{dependencies: d}}, binding: p, approval: a, request: q}
			original := s412Claim(t, x)
			switch field {
			case "run":
				x.binding.RunID = "other"
			case "install":
				x.binding.InstallationID = "other"
			case "actor":
				x.approval.ActorID = q.TargetID
			case "target":
				x.request.TargetID = a.ActorID
			case "role":
				x.request.Role = RoleMember
			case "status":
				x.request.Status = UserStatusActive
			case "evidence":
				x.request.EvidenceRef = "other"
			}
			changed := s412Claim(t, x)
			scopeSame := bytes.Equal(original.ScopeHash, changed.ScopeHash)
			digestSame := bytes.Equal(original.RequestDigest, changed.RequestDigest)
			if field == "same-replay" {
				if !scopeSame || !digestSame || !bytes.Equal(original.KeyLocator, changed.KeyLocator) {
					t.Fatal("same intent lost durable identity")
				}
				return
			}
			if field == "role" || field == "status" || field == "evidence" {
				if !scopeSame || digestSame {
					t.Fatal("changed body bypassed exact same-scope digest")
				}
				return
			}
			if scopeSame {
				t.Fatal("cross actor/target/run/install reused authority scope")
			}
		})
	}
}

func s412Claim(t *testing.T, x recoveryRestoreAttempt) idempotency.Record {
	t.Helper()
	request, err := x.beginRequest()
	if err != nil {
		t.Fatal(err)
	}
	claim, err := idempotency.NewPasswordResetClaim(request, 1)
	if err != nil {
		t.Fatal(err)
	}
	return claim
}

// Guard regressions complement the real account-history PostgreSQL allocation
// probe; this scripted read does not claim to prove cross-purpose SQL semantics.
func s412F001GenerationGuards(t *testing.T) {
	for _, name := range []string{"max-int64", "negative-max", "query-error", "context-error"} {
		t.Run(name, func(t *testing.T) {
			value := int64(math.MaxInt64)
			want := ErrTemporaryCredentialGeneration
			step := queryStep("SELECT COALESCE(MAX(generation),0)", []any{value})
			switch name {
			case "negative-max":
				step.row = []any{int64(-1)}
			case "query-error":
				want = errors.New("synthetic generation read error")
				step.err = want
			case "context-error":
				want = context.Canceled
				step.err = want
			}
			effects := &s412F001PreparationEffects{}
			owner := &recoveryRestoreReset{resets: &PasswordResetCoordinator{dependencies: PasswordResetDependencies{Secrets: effects, IDs: effects}}}
			attempt := recoveryRestoreAttempt{owner: owner, ctx: context.Background()}
			tx := &passwordResetScriptTx{steps: []passwordResetSQLStep{step}}
			u, _ := s412TargetFixture()
			err := attempt.create(tx, u, idempotency.BeginRequest{})
			if !errors.Is(err, want) || effects.calls != 0 || tx.executes != 0 {
				t.Fatalf("generation read failure allocated authority: err=%v effects=%d writes=%d", err, effects.calls, tx.executes)
			}
			tx.assertComplete(t)
		})
	}
}

type s412F001PreparationEffects struct{ calls int }

func (p *s412F001PreparationEffects) Generate() (string, error) {
	p.calls++
	return "", errors.New("unexpected secret/ID before generation guard")
}
func (p *s412F001PreparationEffects) NewID() (string, error) { return p.Generate() }
