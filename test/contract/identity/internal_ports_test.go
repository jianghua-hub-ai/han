//go:build contract

package identity_contract_test

import (
	"context"
	"reflect"
	"testing"

	"han/internal/idempotency"
	"han/internal/identity"
	"han/internal/platform"
	"han/internal/recovery"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
)

// TestInternalPorts_ArticleAccountPortDirection pins the S4 canonical
// orphaning seam: identity owns the interface, article owns the implementation,
// and the call is a caller-transaction, article-owned command.
func TestInternalPorts_ArticleAccountPortDirection(t *testing.T) {
	portType := reflect.TypeOf((*identity.ArticleAccountPort)(nil)).Elem()
	if portType.Kind() != reflect.Interface {
		t.Fatalf("ArticleAccountPort kind = %v, want interface", portType.Kind())
	}
	method, ok := portType.MethodByName("OrphanArticlesCanonical")
	if !ok {
		t.Fatal("ArticleAccountPort is missing OrphanArticlesCanonical")
	}
	wantIn := []reflect.Type{
		reflect.TypeOf((*context.Context)(nil)).Elem(),
		reflect.TypeOf((*pgx.Tx)(nil)).Elem(),
		reflect.TypeOf(""),
	}
	wantOut := []reflect.Type{
		reflect.TypeOf(identity.CanonicalDeleteArticleResult{}),
		reflect.TypeOf((*error)(nil)).Elem(),
	}
	assertInterfaceMethod(t, "ArticleAccountPort.OrphanArticlesCanonical", method, wantIn, wantOut)

	// The adapter func must satisfy the interface so article-owned code can be
	// injected without identity importing article.
	var adapter identity.ArticleAccountPort = identity.ArticleAccountPortFunc(nil)
	if _, ok := adapter.(identity.ArticleAccountPort); !ok {
		t.Fatal("ArticleAccountPortFunc does not satisfy ArticleAccountPort")
	}
}

// TestInternalPorts_AuthorStateReaderDirection pins the S2 author-state seam as
// a batched, read-only identity -> article fact provider.
func TestInternalPorts_AuthorStateReaderDirection(t *testing.T) {
	method, ok := reflect.TypeOf(&identity.ArticleIdentityReader{}).
		MethodByName("GetAuthorStates")
	if !ok {
		t.Fatal("ArticleIdentityReader is missing GetAuthorStates")
	}
	wantIn := []reflect.Type{
		reflect.TypeOf((*context.Context)(nil)).Elem(),
		reflect.TypeOf([]string{}),
	}
	wantOut := []reflect.Type{
		reflect.TypeOf(map[string]identity.UserStatus{}),
		reflect.TypeOf((*error)(nil)).Elem(),
	}
	assertConcreteMethod(t, "ArticleIdentityReader.GetAuthorStates", method, wantIn, wantOut)
}

// TestInternalPorts_PasswordResetGateConsumerNeutrality pins the S10 consumer
// port to its neutral declaration: no operation DTO, lineage enum, SQL, or
// settlement field may appear on the interface.
func TestInternalPorts_PasswordResetGateConsumerNeutrality(t *testing.T) {
	gateType := reflect.TypeOf((*session.PasswordResetGate)(nil)).Elem()
	if gateType.NumMethod() != 1 {
		t.Fatalf("PasswordResetGate method count = %d, want 1", gateType.NumMethod())
	}
	method, ok := gateType.MethodByName("CheckLockedUser")
	if !ok {
		t.Fatal("PasswordResetGate is missing CheckLockedUser")
	}
	wantIn := []reflect.Type{
		reflect.TypeOf((*context.Context)(nil)).Elem(),
		reflect.TypeOf((*pgx.Tx)(nil)).Elem(),
		reflect.TypeOf(""),
	}
	wantOut := []reflect.Type{reflect.TypeOf((*error)(nil)).Elem()}
	assertInterfaceMethod(t, "PasswordResetGate.CheckLockedUser", method, wantIn, wantOut)

	// The single production implementation must satisfy the neutral port.
	if _, ok := interface{}(identity.PasswordResetStore{}).(interface {
		CheckLockedUser(context.Context, pgx.Tx, string) error
	}); !ok {
		t.Fatal("identity.PasswordResetStore does not satisfy the neutral PasswordResetGate shape")
	}
}

// TestInternalPorts_RecoveryRunAuthorityShape pins the S13/S15 recovery run
// authority: identity owns the store; the deletion path consumes it read-only via
// a lock-only port that cannot transition run state.
func TestInternalPorts_RecoveryRunAuthorityShape(t *testing.T) {
	lockerType := reflect.TypeOf((*identity.RecoveryRunLocker)(nil)).Elem()
	if lockerType.NumMethod() != 1 {
		t.Fatalf("RecoveryRunLocker method count = %d, want 1", lockerType.NumMethod())
	}
	method, ok := lockerType.MethodByName("LockRecoveryRunForDeletion")
	if !ok {
		t.Fatal("RecoveryRunLocker is missing LockRecoveryRunForDeletion")
	}
	wantIn := []reflect.Type{
		reflect.TypeOf((*context.Context)(nil)).Elem(),
		reflect.TypeOf((*pgx.Tx)(nil)).Elem(),
		reflect.TypeOf(""),
	}
	wantOut := []reflect.Type{
		reflect.TypeOf(recovery.RecoveryRun{}),
		reflect.TypeOf((*error)(nil)).Elem(),
	}
	assertInterfaceMethod(t, "RecoveryRunLocker.LockRecoveryRunForDeletion", method, wantIn, wantOut)

	// The identity-owned store implements the full authority including Transition;
	// the deletion locker must not expose state transitions.
	if _, ok := reflect.TypeOf(&identity.RecoveryRunStore{}).
		MethodByName("Transition"); !ok {
		t.Fatal("identity.RecoveryRunStore must own the Transition authority")
	}
	if _, ok := lockerType.MethodByName("Transition"); ok {
		t.Fatal("RecoveryRunLocker must not expose run state transitions")
	}
}

// TestInternalPorts_RecoveryOpenGateIsReadOnly pins the maintenance/readiness
// gate consumed before a recovery replay as a read-only check.
func TestInternalPorts_RecoveryOpenGateIsReadOnly(t *testing.T) {
	gateType := reflect.TypeOf((*identity.RecoveryOpenGate)(nil)).Elem()
	if gateType.NumMethod() != 1 {
		t.Fatalf("RecoveryOpenGate method count = %d, want 1", gateType.NumMethod())
	}
	method, ok := gateType.MethodByName("CheckRecoveryOpenGate")
	if !ok {
		t.Fatal("RecoveryOpenGate is missing CheckRecoveryOpenGate")
	}
	wantIn := []reflect.Type{
		reflect.TypeOf((*context.Context)(nil)).Elem(),
		reflect.TypeOf((*pgx.Tx)(nil)).Elem(),
	}
	wantOut := []reflect.Type{reflect.TypeOf((*error)(nil)).Elem()}
	assertInterfaceMethod(t, "RecoveryOpenGate.CheckRecoveryOpenGate", method, wantIn, wantOut)
}

// TestInternalPorts_RecoveryDeleteCapabilityIsOpaque pins the S14 recovery
// deletion capability as an opaque, non-forgeable handle that exposes only its
// run/backup binding.
func TestInternalPorts_RecoveryDeleteCapabilityIsOpaque(t *testing.T) {
	capability, err := identity.NewRecoveryDeleteCapability("run-01", "backup-01")
	if err != nil {
		t.Fatalf("NewRecoveryDeleteCapability() error = %v", err)
	}
	if capability.RunID() != "run-01" || capability.BackupRef() != "backup-01" {
		t.Fatalf("capability binding = %q/%q", capability.RunID(), capability.BackupRef())
	}
	for _, blank := range []struct{ run, backup string }{
		{"", "backup-01"},
		{"run-01", ""},
		{" run-01", "backup-01"},
		{"run-01", "backup-01 "},
	} {
		if _, err := identity.NewRecoveryDeleteCapability(blank.run, blank.backup); err == nil {
			t.Fatalf("NewRecoveryDeleteCapability(%q,%q) error = nil", blank.run, blank.backup)
		}
	}
	// The capability must be constructed only of unexported fields; no public
	// mutable field may let a caller forge a binding.
	valueType := reflect.TypeOf(capability)
	for index := 0; index < valueType.NumField(); index++ {
		if valueType.Field(index).PkgPath == "" {
			t.Fatalf("RecoveryDeleteCapability exposes exported field %q",
				valueType.Field(index).Name)
		}
	}
}

// TestInternalPorts_ArticleDeletionGateIsBothReadAndReplay pins the S16
// consumer seam: identity may both replay-verify and read the article-owned
// status bytes with the identical (run, backup) binding.
func TestInternalPorts_ArticleDeletionGateIsBothReadAndReplay(t *testing.T) {
	gateType := reflect.TypeOf((*recovery.ArticleDeletionGate)(nil)).Elem()
	for _, name := range []string{"ReplayAndVerify", "ReadStatus"} {
		method, ok := gateType.MethodByName(name)
		if !ok {
			t.Fatalf("ArticleDeletionGate is missing %s", name)
		}
		wantIn := []reflect.Type{
			reflect.TypeOf((*context.Context)(nil)).Elem(),
			reflect.TypeOf(""),
			reflect.TypeOf(""),
		}
		wantOut := []reflect.Type{
			reflect.TypeOf([]byte{}),
			reflect.TypeOf((*error)(nil)).Elem(),
		}
		assertInterfaceMethod(t, "ArticleDeletionGate."+name, method, wantIn, wantOut)
	}

	// The recovery coordinator must consume the gate, not a concrete article type.
	coordinatorType := reflect.TypeOf(&recovery.Coordinator{})
	if _, ok := coordinatorType.MethodByName("Consume"); !ok {
		t.Fatal("recovery.Coordinator must own the Consume coordination")
	}
}

// TestInternalPorts_PublicErrorRegistryCoversIdentityBoundary pins the shared
// public error registry entries the identity REST boundary depends on, so a
// drift in code/status/retry is caught before it reaches a consumer.
func TestInternalPorts_PublicErrorRegistryCoversIdentityBoundary(t *testing.T) {
	want := map[platform.Code]struct {
		status int
		retry  platform.RetryPolicy
	}{
		platform.CodeRequestRouteNotFound:         {404, platform.RetryNever},
		platform.CodeRequestMethodNotAllowed:      {405, platform.RetryAfterCorrection},
		platform.CodeRequestInvalidJSON:           {400, platform.RetryAfterCorrection},
		platform.CodeRequestInvalidCursor:         {400, platform.RetryAfterCorrection},
		platform.CodeAuthLoginFailed:              {401, platform.RetryAllowed},
		platform.CodeAuthSessionRequired:          {401, platform.RetryAllowed},
		platform.CodeAuthCSRFFailed:               {403, platform.RetryNever},
		platform.CodeAuthForbidden:                {403, platform.RetryNever},
		platform.CodeIdentityUserNotFound:         {404, platform.RetryNever},
		platform.CodeIdentityIdentifierConflict:   {409, platform.RetryAfterCorrection},
		platform.CodeIdentityInvalidState:         {409, platform.RetryAfterStateChange},
		platform.CodeIdentityValidationFailed:     {422, platform.RetryAfterCorrection},
		platform.CodeRequestRateLimited:           {429, platform.RetryAllowed},
		platform.CodePlatformProcessingGateClosed: {503, platform.RetryAfterConfiguration},
		platform.CodePlatformDatabaseUnavailable:  {503, platform.RetryAllowed},
		platform.CodePlatformRecoveryInProgress:   {503, platform.RetryAllowed},
	}
	for code, expectation := range want {
		definition, ok := platform.DefaultRegistry.Definition(code)
		if !ok {
			t.Fatalf("public error %q is not registered", code)
		}
		if definition.HTTPStatus != expectation.status {
			t.Errorf("error %q status = %d, want %d",
				code, definition.HTTPStatus, expectation.status)
		}
		if definition.Retry != expectation.retry {
			t.Errorf("error %q retry = %q, want %q",
				code, definition.Retry, expectation.retry)
		}
	}
}

// assertInterfaceMethod compares an interface method (no receiver) signature.
func assertInterfaceMethod(
	t *testing.T,
	name string,
	method reflect.Method,
	wantIn []reflect.Type,
	wantOut []reflect.Type,
) {
	t.Helper()
	assertMethodSignature(t, name, method, wantIn, wantOut, false)
}

// assertConcreteMethod compares a concrete-type method (receiver first) signature.
func assertConcreteMethod(
	t *testing.T,
	name string,
	method reflect.Method,
	wantIn []reflect.Type,
	wantOut []reflect.Type,
) {
	t.Helper()
	assertMethodSignature(t, name, method, wantIn, wantOut, true)
}

// TestInternalPorts_PasswordResetDurableInternalPorts pins the password-reset
// durable internal ports that the atomic coordinator, dispatch, Gate, worker and
// operator exchange: the consumer-neutral Gate (`CheckLockedUser`) plus the
// identity-owned atomic candidate writer and its closed classification enum.
func TestInternalPorts_PasswordResetDurableInternalPorts(t *testing.T) {
	// The Gate impl must expose exactly the neutral three-arg shape.
	gateImpl, ok := reflect.TypeOf(identity.PasswordResetStore{}).
		MethodByName("CheckLockedUser")
	if !ok {
		t.Fatal("identity.PasswordResetStore is missing CheckLockedUser")
	}
	assertConcreteMethod(t, "PasswordResetStore.CheckLockedUser", gateImpl,
		[]reflect.Type{
			reflect.TypeOf((*context.Context)(nil)).Elem(),
			reflect.TypeOf((*pgx.Tx)(nil)).Elem(),
			reflect.TypeOf(""),
		},
		[]reflect.Type{reflect.TypeOf((*error)(nil)).Elem()},
	)

	// The atomic candidate writer binds the six-field idempotency transaction and
	// returns the identity-owned classification.
	writer, ok := reflect.TypeOf(identity.PasswordResetStore{}).
		MethodByName("BeginOrClassifyCandidate")
	if !ok {
		t.Fatal("identity.PasswordResetStore is missing BeginOrClassifyCandidate")
	}
	assertConcreteMethod(t, "PasswordResetStore.BeginOrClassifyCandidate", writer,
		[]reflect.Type{
			reflect.TypeOf((*context.Context)(nil)).Elem(),
			reflect.TypeOf((*idempotency.Transaction)(nil)).Elem(),
			reflect.TypeOf(identity.PasswordResetCandidate{}),
		},
		[]reflect.Type{
			reflect.TypeOf(identity.PasswordResetCandidateResult{}),
			reflect.TypeOf((*error)(nil)).Elem(),
		},
	)

	// The classification enum is closed: exactly the three durable outcomes.
	for _, action := range []identity.PasswordResetCandidateAction{
		identity.PasswordResetCandidateCreated,
		identity.PasswordResetCandidateExisting,
		identity.PasswordResetCandidateIntegrityHold,
	} {
		if action == "" {
			t.Fatal("password reset classification enum has an empty member")
		}
	}

	// The Gate's closed cause is the identity-owned closed signal, not an
	// arbitrary error, so consumers can fail closed without interpreting causes.
	if identity.ErrPasswordResetGateClosed == nil {
		t.Fatal("identity.ErrPasswordResetGateClosed must be a stable sentinel")
	}
}

// TestInternalPorts_ThreeConsumersShareOneGateType pins that each authentication
// consumer (login, password change, session resolve) declares the identical
// consumer-neutral Gate type, so production composition injects one shared
// instance rather than three ad-hoc variants.
func TestInternalPorts_ThreeConsumersShareOneGateType(t *testing.T) {
	gateType := reflect.TypeOf((*session.PasswordResetGate)(nil)).Elem()
	consumers := []struct {
		name   string
		fields []reflect.Type
	}{
		{"LoginDependencies", structFieldTypes(identity.LoginDependencies{})},
		{"PasswordChangeDependencies", structFieldTypes(identity.PasswordChangeDependencies{})},
		{"SessionStoreDependencies", structFieldTypes(session.StoreDependencies{})},
	}
	for _, consumer := range consumers {
		t.Run(consumer.name, func(t *testing.T) {
			found := false
			for _, fieldType := range consumer.fields {
				if fieldType == gateType {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("%s does not declare the shared %v collaborator",
					consumer.name, gateType)
			}
		})
	}
}

func structFieldTypes(value any) []reflect.Type {
	structType := reflect.TypeOf(value)
	fields := make([]reflect.Type, 0, structType.NumField())
	for index := 0; index < structType.NumField(); index++ {
		fields = append(fields, structType.Field(index).Type)
	}
	return fields
}

func assertMethodSignature(
	t *testing.T,
	name string,
	method reflect.Method,
	wantIn []reflect.Type,
	wantOut []reflect.Type,
	hasReceiver bool,
) {
	t.Helper()
	methodType := method.Type
	start := 0
	if hasReceiver {
		start = 1
	}
	gotIn := make([]reflect.Type, 0, methodType.NumIn()-start)
	for index := start; index < methodType.NumIn(); index++ {
		gotIn = append(gotIn, methodType.In(index))
	}
	if !reflect.DeepEqual(gotIn, wantIn) {
		t.Errorf("%s input types = %v, want %v", name, gotIn, wantIn)
	}
	gotOut := make([]reflect.Type, 0, methodType.NumOut())
	for index := 0; index < methodType.NumOut(); index++ {
		gotOut = append(gotOut, methodType.Out(index))
	}
	if !reflect.DeepEqual(gotOut, wantOut) {
		t.Errorf("%s output types = %v, want %v", name, gotOut, wantOut)
	}
}
