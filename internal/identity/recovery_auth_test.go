package identity

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"han/internal/recovery"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
)

func TestRecoveryMaintenanceAuthentication(t *testing.T) {
	t.Run("ScopeAndPrivateBinding", func(t *testing.T) {
		t.Run("LoginCookieSelection", s411LoginCookieSelection)
		t.Run("RealRuntimePrivateContextAndStaleGrant", s411PrivateBinding)
		t.Run("MissingRecoveryAuthorityNeverStartsChange", func(t *testing.T) {
			for _, ctx := range []context.Context{nil, context.Background()} {
				result, err := (&Service{}).ChangeRecoveryPassword(ctx, recovery.PhasePermit{}, PasswordChangeRequest{})
				if err == nil || result.SessionToken != "" {
					t.Fatal("untrusted principal authorized recovery")
				}
			}
		})
		t.Run("MissingStoreRejectedBeforeCredentialProbe", func(t *testing.T) {
			hasher := &loginHasherStub{matches: map[string]string{"fake-digest": testFakeCredentialProbe}}
			d := loginConstructorDependencies(hasher, "fake-digest", testFakeCredentialProbe)
			d.RecoveryMode = &recoveryIdentityExecution{}
			if _, err := NewLoginService(d); err == nil || len(hasher.verifiedDigests) != 0 {
				t.Fatal("invalid maintenance constructor accessed credentials")
			}
		})
	})
	t.Run("BootstrapOnlyMePasswordAndDailyReject", func(t *testing.T) {
		t.Run("BootstrapAndPermanentProofMutationMatrix", s411ProofMatrix)
		t.Run("StaticBoolCannotIssueRecoveryCredentials", func(t *testing.T) {
			f := newLoginFixture(t)
			u := loginTestUserWithMode(CredentialModeRecoveryBootstrap)
			u.IsInitialAdmin = true
			u.RecoveryRequired = true
			f.repository.user = u
			f.service.loginDependencies.RecoveryMode = loginRecoveryModeStub{allowed: true}
			got, err := f.service.Login(f.context, f.request(LoginRequest{Username: u.Username, Password: "Password123", SourceFingerprint: "synthetic-source"}))
			if err == nil || got.SessionToken != "" || f.sessions.calls != 0 {
				t.Fatal("static mode created recovery session")
			}
		})
	})
}

func s411PrivateBinding(t *testing.T) {
	s, j := s410MemoryInput()
	f := &recoveryExecutionForwarder{}
	rt, ctx, done := s410Runtime(t, s, s, j, f)
	defer done()
	e := &recoveryIdentityExecution{d: recoveryExecutionDependencies{runtime: rt}}
	var escaped context.Context
	called := 0
	err := f.seal(&s410Callbacks{finalize: func(c context.Context, p recovery.PhasePermit, _ recovery.RunBinding) error {
		called++
		escaped = c
		binding, err := e.maintenanceBinding(c)
		if err != nil || binding.RunID != s410Binding.RunID || binding.Generation <= 0 {
			t.Fatalf("real private binding missing: %v", err)
		}
		if ok, err := e.RecoveryBootstrapAllowed(c); !ok || err != nil {
			t.Fatalf("real phase rejected: %v", err)
		}
		if _, err = e.maintenanceBinding(ctx); err == nil {
			t.Fatal("ordinary context self-authorized")
		}
		bad := recoverySessionBinding(binding)
		bad.RuntimeID += "-other"
		if err = e.CheckLockedMaintenanceUser(c, newLoginTxStub(), "11111111-1111-4111-8111-111111111111", bad); err == nil {
			t.Fatal("caller binding accepted")
		}
		canceled, cancel := context.WithCancel(c)
		cancel()
		if _, err = e.maintenanceBinding(canceled); err == nil {
			t.Fatal("canceled phase accepted")
		}
		return s410Stop
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err = rt.Resume(ctx, s410Binding); !errors.Is(err, s410Stop) || called != 1 {
		t.Fatalf("public producer callback: %v / %d", err, called)
	}
	if _, err = e.maintenanceBinding(escaped); err == nil {
		t.Fatal("escaped runtime context survived phase end")
	}
}

func s411ProofFixture(changed bool) (recovery.AdmissionRun, recoveryExecutionUser) {
	id, digest := "11111111-1111-4111-8111-111111111111", "synthetic-digest"
	g := int64(2)
	r := recovery.AdmissionRun{InitialAdminUserID: &id, InitialAdminBootstrapGeneration: &g}
	u := recoveryExecutionUser{ID: id, Initial: true, Role: "admin", Status: "active", Required: true, Mode: "recovery_bootstrap", Auth: g, Digest: &digest, PermanentGeneration: &g}
	if changed {
		now, auditID := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC), "22222222-2222-4222-8222-222222222222"
		next := g + 1
		r.InitialAdminChangedGeneration = &next
		r.InitialAdminChangeAuditID = &auditID
		r.InitialAdminChangedAt = &now
		u.Required = false
		u.Mode = "permanent"
		u.Auth = next
		u.PermanentGeneration = &next
	}
	return r, u
}

func s411ProofMatrix(t *testing.T) {
	for _, changed := range []bool{false, true} {
		name := "bootstrap"
		if changed {
			name = "permanent"
		}
		t.Run(name, func(t *testing.T) {
			r, u := s411ProofFixture(changed)
			if err := recoveryMaintenanceUser(r, u); err != nil {
				t.Fatalf("valid G/G+1 proof rejected: %v", err)
			}
			cases := []struct {
				name   string
				mutate func(*recovery.AdmissionRun, *recoveryExecutionUser)
			}{
				{"noninitial", func(_ *recovery.AdmissionRun, u *recoveryExecutionUser) { u.Initial = false }},
				{"member", func(_ *recovery.AdmissionRun, u *recoveryExecutionUser) { u.Role = "member" }},
				{"inactive", func(_ *recovery.AdmissionRun, u *recoveryExecutionUser) { u.Status = "suspended" }},
				{"wrong-user", func(_ *recovery.AdmissionRun, u *recoveryExecutionUser) { u.ID = "other" }},
				{"false-proof", func(_ *recovery.AdmissionRun, u *recoveryExecutionUser) { u.Required = !u.Required }},
				{"wrong-generation", func(_ *recovery.AdmissionRun, u *recoveryExecutionUser) { u.Auth++ }},
				{"missing-digest", func(_ *recovery.AdmissionRun, u *recoveryExecutionUser) { u.Digest = nil }},
				{"missing-generation", func(_ *recovery.AdmissionRun, u *recoveryExecutionUser) { u.PermanentGeneration = nil }},
				{"source-operation", func(_ *recovery.AdmissionRun, u *recoveryExecutionUser) { v := "unrelated"; u.SourceOperation = &v }},
				{"source-credential", func(_ *recovery.AdmissionRun, u *recoveryExecutionUser) { v := "unrelated"; u.SourceCredential = &v }},
				{"missing-initial-proof", func(r *recovery.AdmissionRun, _ *recoveryExecutionUser) { r.InitialAdminUserID = nil }},
				{"overflow", func(r *recovery.AdmissionRun, _ *recoveryExecutionUser) {
					v := int64(math.MaxInt64)
					r.InitialAdminBootstrapGeneration = &v
				}},
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					r, u := s411ProofFixture(changed)
					tc.mutate(&r, &u)
					if recoveryMaintenanceUser(r, u) == nil {
						t.Fatal("mutated trust proof accepted")
					}
				})
			}
		})
	}
}

var _ session.MaintenanceSessionAuthority = (*recoveryIdentityExecution)(nil)

func s411LoginCookieSelection(t *testing.T) {
	marker := errors.New("synthetic cookie selection storage failure")
	for _, tc := range []struct {
		name, token, id string
		err             error
		wantCalls       int
	}{
		{name: "NoCookieNeedsNoLookup"},
		{name: "EligibleTokenSelected", token: "synthetic-old-token", id: "old-session", wantCalls: 1},
		{name: "NoEligibleRowPermitsFreshIssue", token: "synthetic-old-token", err: pgx.ErrNoRows, wantCalls: 1},
		{name: "UnknownStorageErrorMustNotBecomeFreshIssue", token: "synthetic-old-token", err: marker, wantCalls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := &s411CookieSelectionTx{loginTxStub: newLoginTxStub(), id: tc.id, err: tc.err}
			in := session.CreateInput{UserID: "initial", AuthGeneration: 3, AuthenticatedCredentialKind: session.CredentialPermanent,
				IssuedAt: time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)}
			b := session.RecoverySessionBinding{RunID: "run", InstallationID: "installation", RuntimeID: "runtime", Generation: 1}
			previous, err := maintenanceLoginPrevious(context.Background(), tx, tc.token, in, b)
			if tc.err == marker {
				if !errors.Is(err, marker) || previous != nil {
					t.Fatal("storage failure hidden as missing optional cookie")
				}
			} else if err != nil || (previous != nil) != (tc.id != "") || (previous != nil && *previous != tc.id) {
				t.Fatalf("optional selection: id=%v err=%v", previous, err)
			}
			if tx.calls != tc.wantCalls || tx.commitCalls != 0 || tx.rollbackCalls != 0 {
				t.Fatal("cookie selection changed caller transaction ownership")
			}
			if tx.calls > 0 && (len(tx.args) != 9 || tx.args[1] != in.UserID || tx.args[4] != b.RuntimeID || tx.args[5] != b.Generation) {
				t.Fatal("selection lost authenticated user or current runtime binding")
			}
		})
	}
}

type s411CookieSelectionTx struct {
	*loginTxStub
	id    string
	err   error
	calls int
	args  []any
}

func (tx *s411CookieSelectionTx) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	tx.calls++
	tx.args = args
	return s411CookieSelectionRow{id: tx.id, err: tx.err}
}

type s411CookieSelectionRow struct {
	id  string
	err error
}

func (r s411CookieSelectionRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*dest[0].(*string) = r.id
	return nil
}
