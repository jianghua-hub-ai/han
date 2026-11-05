package session

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type maintenanceAuthorityStub struct {
	err   error
	calls int
	check func(pgx.Tx, string, RecoverySessionBinding)
}

func (a *maintenanceAuthorityStub) CheckLockedMaintenanceUser(_ context.Context, tx pgx.Tx, u string, b RecoverySessionBinding) error {
	a.calls++
	if a.check != nil {
		a.check(tx, u, b)
	}
	return a.err
}

func maintenanceFixture(t *testing.T) (MaintenanceStore, *transactionStub, *maintenanceAuthorityStub, *passwordResetGateStub, context.Context, CreateInput, RecoverySessionBinding) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)
	in := validCreateInput(time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC))
	in.AuthGeneration = 2
	in.AuthenticatedCredentialKind = CredentialRecoveryBootstrap
	b := RecoverySessionBinding{RunID: "unit-run", InstallationID: "unit-install", RuntimeID: "unit-runtime", Generation: 1}
	tx := newTransactionStub()
	a := &maintenanceAuthorityStub{}
	g := &passwordResetGateStub{}
	s, err := NewMaintenanceStore(MaintenanceStoreDependencies{Transactions: &storeTransactorStub{tx: tx}, PasswordResetGate: g, Authority: a})
	if err != nil {
		t.Fatal(err)
	}
	return s, tx, a, g, ctx, in, b
}

func TestRecoveryMaintenanceStore(t *testing.T) {
	t.Run("ScopeAndPrivateBinding", func(t *testing.T) {
		t.Run("MissingAndTypedNilDependencies", maintenanceMissingDependencies)
		t.Run("CreateOwnsOnlyDigestsAndCallerTransaction", maintenanceCreateContract)
		t.Run("RejectedBindingNeverWrites", maintenanceRejectBindings)
		t.Run("CredentialLineageCannotBecomeRecoveryTrust", maintenanceRejectLineage)
	})
	t.Run("BootstrapOnlyMePasswordAndDailyReject", func(t *testing.T) {
		t.Run("RealResolverCommitAndCSRFBoundary", maintenanceResolveMatrix)
		t.Run("RestrictedPrincipalOnlyMeAndPassword", func(t *testing.T) {
			p := Principal{UserID: "unit", Status: "active", IsInitialAdmin: true, MustChangePassword: true, maintenance: true}
			for _, access := range []Access{AccessGeneral, AccessCurrentPrivacyNotice, AccessAcknowledgePrivacy, Access("review"), Access("complete")} {
				if p.Allows(access) {
					t.Fatalf("bootstrap allowed %s", access)
				}
			}
			if !p.Allows(AccessCurrentIdentity) || !p.Allows(AccessChangePassword) {
				t.Fatal("bootstrap cannot establish trust")
			}
		})
	})
}

func maintenanceResolveMatrix(t *testing.T) {
	for _, name := range []string{"bootstrap", "permanent", "csrf", "expired", "gate", "authority", "commit"} {
		t.Run(name, func(t *testing.T) {
			s, tx, a, g, ctx, in, b := maintenanceFixture(t)
			row := validPrincipalRow(in.CSRFToken)
			row.values[2] = "admin"
			row.values[3] = true
			row.values[5] = "recovery_bootstrap"
			row.values[10] = "recovery_bootstrap"
			tx.queryRows = []pgx.Row{rowStub{values: []any{in.UserID}}, row}
			boundary := s.dependencies.Transactions.(*storeTransactorStub)
			boundary.row = rowStub{values: []any{in.UserID, b.RunID, b.InstallationID, b.RuntimeID, b.Generation}}
			a.check = func(txArg pgx.Tx, u string, got RecoverySessionBinding) {
				if txArg != tx || u != in.UserID || got != b || len(tx.queryCalls) != 1 {
					t.Fatal("User/run/Session order or binding lost")
				}
			}
			input := ResolveInput{Token: in.Token, CSRFToken: in.CSRFToken, RequireCSRF: true, Now: in.IssuedAt}
			switch name {
			case "permanent":
				row.values[5] = "permanent"
				row.values[10] = "permanent"
			case "csrf":
				input.CSRFToken = "other"
			case "expired":
				row.values[12] = false
			case "gate":
				g.err = ErrPasswordResetGateClosed
			case "authority":
				a.err = ErrSessionRequired
			case "commit":
				tx.commitErr = errors.New("synthetic commit unknown")
			}
			p, err := s.ResolvePrincipal(ctx, input)
			good := name == "bootstrap" || name == "permanent"
			if good {
				if err != nil || !tx.committed || !p.IsInitialAdmin || !p.maintenance || p.MustAcknowledgePrivacy {
					t.Fatalf("missing trusted principal: %v", err)
				}
				if p.MustChangePassword != (name == "bootstrap") {
					t.Fatal("wrong credential restriction")
				}
			} else if err == nil || p.UserID != "" || !tx.rolledBack {
				t.Fatalf("failed resolve escaped: %v", err)
			}
		})
	}
}

func maintenanceMissingDependencies(t *testing.T) {
	var nilAuthority *maintenanceAuthorityStub
	for _, a := range []MaintenanceSessionAuthority{nil, nilAuthority} {
		if _, err := NewMaintenanceStore(MaintenanceStoreDependencies{Transactions: &storeTransactorStub{}, PasswordResetGate: &passwordResetGateStub{}, Authority: a}); err == nil {
			t.Fatal("missing authority accepted")
		}
	}
}

func maintenanceCreateContract(t *testing.T) {
	s, tx, a, g, ctx, in, b := maintenanceFixture(t)
	tx.queryRows = []pgx.Row{rowStub{values: []any{true}}}
	got, err := s.CreateMaintenance(ctx, tx, in, b)
	if err != nil || got.Token != in.Token || a.calls != 1 || g.calls != 1 {
		t.Fatalf("create/authority/Gate: %v", err)
	}
	if tx.committed || tx.rolledBack || len(tx.calls) != 1 {
		t.Fatal("store closed caller transaction")
	}
	call := tx.calls[0]
	if !bytes.Equal(call.args[2].([]byte), digest(in.Token)) || !bytes.Equal(call.args[3].([]byte), digest(in.CSRFToken)) {
		t.Fatal("raw credential persisted")
	}
	if !strings.Contains(call.query, "'recovery_maintenance'") || call.args[10] != b.RunID || call.args[13] != b.Generation {
		t.Fatal("binding omitted")
	}
}

func maintenanceRejectBindings(t *testing.T) {
	for _, name := range []string{"run", "installation", "runtime", "zero", "negative", "authority", "nilctx", "unbounded", "cancel"} {
		t.Run(name, func(t *testing.T) {
			s, tx, a, _, ctx, in, b := maintenanceFixture(t)
			switch name {
			case "run":
				b.RunID = ""
			case "installation":
				b.InstallationID = ""
			case "runtime":
				b.RuntimeID = " wrong "
			case "zero":
				b.Generation = 0
			case "negative":
				b.Generation = -1
			case "authority":
				a.err = ErrSessionRequired
			case "nilctx":
				ctx = nil
			case "unbounded":
				ctx = context.Background()
			case "cancel":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			got, err := s.CreateMaintenance(ctx, tx, in, b)
			if err == nil || got.Token != "" || len(tx.calls) != 0 {
				t.Fatal("invalid evidence issued session")
			}
		})
	}
}

func maintenanceRejectLineage(t *testing.T) {
	for _, name := range []string{"bootstrap", "temporary", "zero_generation", "operation", "credential", "authenticated_id"} {
		t.Run(name, func(t *testing.T) {
			s, tx, _, _, ctx, in, b := maintenanceFixture(t)
			id := "11111111-1111-4111-8111-111111111111"
			switch name {
			case "bootstrap":
				in.AuthenticatedCredentialKind = CredentialBootstrap
			case "temporary":
				in.AuthenticatedCredentialKind = CredentialTemporary
				in.AuthenticatedCredentialID = &id
			case "zero_generation":
				in.AuthGeneration = 0
			case "operation":
				in.LineageOperationID = &id
			case "credential":
				in.LineageCredentialID = &id
			case "authenticated_id":
				in.AuthenticatedCredentialID = &id
			}
			if got, err := s.CreateMaintenance(ctx, tx, in, b); err == nil || got.Token != "" || len(tx.calls) != 0 {
				t.Fatal("unrelated lineage issued recovery authority")
			}
		})
	}
}
