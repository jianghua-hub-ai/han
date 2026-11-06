//go:build integration

package identity

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"han/internal/audit"
	"han/internal/platform"
	"han/internal/privacy"
	"han/internal/security"

	"github.com/jackc/pgx/v5/pgxpool"
)

// registrationPostgresNoticeVersion mirrors the version the web client sends
// from web/src/pages/register.tsx.
const registrationPostgresNoticeVersion = "v1.0.0"

const registrationPostgresPassword = "ir02RegistrationPassw0rd"

// registrationPostgresSource is a deterministic source provider so the
// integration fixture does not depend on proxy headers or HMAC keys.
type registrationPostgresSource struct{}

func (registrationPostgresSource) Evidence(*http.Request) (RegistrationSourceEvidence, error) {
	return RegistrationSourceEvidence{
		Kind:        "network_source_hmac_sha256_v1",
		Fingerprint: "ir02-registration-source-fingerprint",
	}, nil
}

// registrationPostgresFixture wires the production registration collaborators
// (real repository, notice reader, processing gate, password policy, gated
// Argon2id hasher, audit store, UUID generator and clock) against one isolated
// PostgreSQL schema that is dropped on cleanup.
type registrationPostgresFixture struct {
	ctx        context.Context
	pool       *pgxpool.Pool
	gateConfig platform.ProcessingGateConfig
	hasher     *security.GatedArgon2idHasher
	handler    *Handler
}

func newRegistrationPostgresFixture(t *testing.T) *registrationPostgresFixture {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	t.Cleanup(cancel)

	adminPool := newIdentityTestPool(t, ctx)
	schema := createCommittedIdentitySchema(t, ctx, adminPool)
	pool := newLoginSchemaPool(t, ctx, schema)

	rawHasher, err := security.NewArgon2idHasher(security.Argon2idConfig{
		MemoryKiB:   8192,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}, rand.Reader)
	if err != nil {
		t.Fatalf("create registration password hasher: %v", err)
	}
	processGate, err := security.NewArgon2idProcessGate(1)
	if err != nil {
		t.Fatalf("create registration password process gate: %v", err)
	}
	hasher, err := security.NewGatedArgon2idHasher(rawHasher, processGate)
	if err != nil {
		t.Fatalf("create gated registration password hasher: %v", err)
	}

	gateConfig := platform.ProcessingGateConfig{
		RegionRef:        "cn-ir02-region",
		ControllerRef:    "cn-ir02-controller",
		LegalBasisRef:    "cn-ir02-legal-basis",
		SMTPProcessorRef: "cn-ir02-smtp-processor",
	}
	store, err := privacy.NewStore(pool)
	if err != nil {
		t.Fatalf("create privacy processing store: %v", err)
	}
	gate, err := privacy.NewProcessingGate(gateConfig, store)
	if err != nil {
		t.Fatalf("create privacy processing gate: %v", err)
	}

	service, err := NewService(RegistrationDependencies{
		Transactions:   pool,
		Repository:     Repository{},
		Notices:        CurrentNoticeReader{},
		Gate:           gate,
		PasswordPolicy: security.PasswordPolicy{},
		PasswordHasher: hasher,
		Audit:          audit.Store{},
		IDs:            RandomUUIDGenerator{},
		Clock:          SystemClock{},
	})
	if err != nil {
		t.Fatalf("create registration service: %v", err)
	}
	handler, err := NewHandler(service, registrationPostgresSource{}, 262144)
	if err != nil {
		t.Fatalf("create registration handler: %v", err)
	}

	return &registrationPostgresFixture{
		ctx:        ctx,
		pool:       pool,
		gateConfig: gateConfig,
		hasher:     hasher,
		handler:    handler,
	}
}

func (f *registrationPostgresFixture) seedCurrentNotice(t *testing.T, version string) {
	t.Helper()
	if _, err := f.pool.Exec(f.ctx, `
		INSERT INTO privacy_notices (version, content, content_hash, effective_at, status)
		VALUES ($1, $2, $3, $4, 'current')
	`,
		version,
		"IR-02 registration privacy notice fixture",
		[]byte("ir02-registration-notice-hash"),
		time.Now().UTC().Add(-time.Hour),
	); err != nil {
		t.Fatalf("seed current privacy notice: %v", err)
	}
}

func (f *registrationPostgresFixture) seedProcessingGateRecord(
	t *testing.T,
	config platform.ProcessingGateConfig,
) {
	t.Helper()
	if _, err := f.pool.Exec(f.ctx, `
		INSERT INTO processing_gate_records (
			region_ref, controller_ref, legal_basis_ref, smtp_processor_ref,
			effective_at, recorded_by
		) VALUES ($1, $2, $3, $4, $5, $6)
	`,
		config.RegionRef,
		config.ControllerRef,
		config.LegalBasisRef,
		config.SMTPProcessorRef,
		time.Now().UTC().Add(-time.Hour),
		"ir02-registration-fixture",
	); err != nil {
		t.Fatalf("seed processing gate record: %v", err)
	}
}

func (f *registrationPostgresFixture) countUsers(t *testing.T) int {
	t.Helper()
	var count int
	if err := f.pool.QueryRow(f.ctx, `SELECT count(*) FROM users`).Scan(&count); err != nil {
		t.Fatalf("count registered users: %v", err)
	}
	return count
}

func (f *registrationPostgresFixture) countAuditEvents(t *testing.T, outcome string) int {
	t.Helper()
	var count int
	if err := f.pool.QueryRow(
		f.ctx,
		`SELECT count(*) FROM audit_events WHERE action = $1 AND outcome = $2`,
		registrationAuditAction,
		outcome,
	).Scan(&count); err != nil {
		t.Fatalf("count registration audit events: %v", err)
	}
	return count
}

func registrationPostgresBody(username, email, phone string) map[string]any {
	return map[string]any{
		"username":               username,
		"email":                  email,
		"phone":                  phone,
		"password":               registrationPostgresPassword,
		"privacy_notice_version": registrationPostgresNoticeVersion,
		"privacy_acknowledged":   true,
	}
}

// serveRegistrationPostgres drives the public registration handler through the
// same trace middleware the production router applies.
func serveRegistrationPostgres(
	t *testing.T,
	handler *Handler,
	body map[string]any,
) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("encode registration request body: %v", err)
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users",
		strings.NewReader(string(encoded)),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	platform.TraceMiddleware(http.HandlerFunc(handler.Register)).ServeHTTP(response, request)
	return response
}

func requireRegistrationEnvelope(
	t *testing.T,
	response *httptest.ResponseRecorder,
	status int,
	code platform.Code,
) platform.Envelope {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, status, response.Body)
	}
	var envelope platform.Envelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode registration error envelope: %v; body = %s", err, response.Body)
	}
	if envelope.Code != code {
		t.Fatalf("error code = %q, want %q; body = %s", envelope.Code, code, response.Body)
	}
	return envelope
}

// TestRegistrationPostgresPersistsAggregate registers through the real HTTP
// boundary and verifies the committed user, identifiers, acknowledgement and
// audit evidence in PostgreSQL.
func TestRegistrationPostgresPersistsAggregate(t *testing.T) {
	fixture := newRegistrationPostgresFixture(t)
	fixture.seedCurrentNotice(t, registrationPostgresNoticeVersion)
	fixture.seedProcessingGateRecord(t, fixture.gateConfig)

	username := "ir02-user-" + randomHex(t, 4)
	email := username + "@example.test"
	phone := "+861380000" + randomHex(t, 2)

	response := serveRegistrationPostgres(
		t,
		fixture.handler,
		registrationPostgresBody(username, email, phone),
	)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", response.Code, response.Body)
	}
	var created registrationResponse
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode registration response: %v", err)
	}
	if created.Username != username ||
		created.Role != RoleMember ||
		created.Status != UserStatusActive ||
		created.MustChangePassword {
		t.Fatalf("registration response = %#v", created)
	}

	var (
		storedUsername string
		storedRole     Role
		storedStatus   UserStatus
		credentialMode CredentialMode
		noticeVersion  *string
		passwordDigest *string
		generation     *int64
	)
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT username, role, status, credential_mode,
			onboarding_privacy_acknowledged_version, permanent_password_digest,
			permanent_password_generation
		FROM users
		WHERE id = $1
	`, created.ID).Scan(
		&storedUsername,
		&storedRole,
		&storedStatus,
		&credentialMode,
		&noticeVersion,
		&passwordDigest,
		&generation,
	); err != nil {
		t.Fatalf("read registered user: %v", err)
	}
	if storedUsername != username ||
		storedRole != RoleMember ||
		storedStatus != UserStatusActive ||
		credentialMode != CredentialModePermanent {
		t.Fatalf(
			"stored user = %q/%q/%q/%q",
			storedUsername,
			storedRole,
			storedStatus,
			credentialMode,
		)
	}
	if noticeVersion == nil || *noticeVersion != registrationPostgresNoticeVersion {
		t.Fatalf("onboarding notice version = %v, want %q", noticeVersion, registrationPostgresNoticeVersion)
	}
	if passwordDigest == nil {
		t.Fatal("registered user has no permanent password digest")
	}
	if generation == nil || *generation != 0 {
		t.Fatalf("permanent password generation = %v, want 0", generation)
	}
	matched, needsRehash, err := fixture.hasher.Verify(registrationPostgresPassword, *passwordDigest)
	if err != nil || !matched || needsRehash {
		t.Fatalf("verify stored credential = matched:%v needsRehash:%v err:%v", matched, needsRehash, err)
	}

	identifiers := map[string]string{}
	rows, err := fixture.pool.Query(fixture.ctx, `
		SELECT kind, normalized_value
		FROM identity_identifiers
		WHERE user_id = $1
	`, created.ID)
	if err != nil {
		t.Fatalf("read registration identifiers: %v", err)
	}
	for rows.Next() {
		var kind, value string
		if err := rows.Scan(&kind, &value); err != nil {
			rows.Close()
			t.Fatalf("scan registration identifier: %v", err)
		}
		identifiers[kind] = value
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate registration identifiers: %v", err)
	}
	wantIdentifiers := map[string]string{
		string(IdentifierUsername): strings.ToLower(username),
		string(IdentifierEmail):    strings.ToLower(email),
		string(IdentifierPhone):    phone,
	}
	if len(identifiers) != len(wantIdentifiers) {
		t.Fatalf("identifiers = %v, want %v", identifiers, wantIdentifiers)
	}
	for kind, want := range wantIdentifiers {
		if identifiers[kind] != want {
			t.Fatalf("identifier %q = %q, want %q", kind, identifiers[kind], want)
		}
	}

	var acknowledgementVersion string
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT notice_version
		FROM privacy_acknowledgements
		WHERE user_id = $1
	`, created.ID).Scan(&acknowledgementVersion); err != nil {
		t.Fatalf("read registration acknowledgement: %v", err)
	}
	if acknowledgementVersion != registrationPostgresNoticeVersion {
		t.Fatalf(
			"acknowledged notice version = %q, want %q",
			acknowledgementVersion,
			registrationPostgresNoticeVersion,
		)
	}

	var (
		auditOutcome   string
		auditActorKind audit.ActorKind
		auditTraceID   string
	)
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT outcome, actor_kind, trace_id
		FROM audit_events
		WHERE action = $1 AND target_id = $2
	`, registrationAuditAction, created.ID).Scan(
		&auditOutcome,
		&auditActorKind,
		&auditTraceID,
	); err != nil {
		t.Fatalf("read registration audit event: %v", err)
	}
	if auditOutcome != registrationAuditOutcomeSucceeded || auditActorKind != audit.ActorAnonymous {
		t.Fatalf("registration audit event = %q/%q", auditOutcome, auditActorKind)
	}
	if strings.TrimSpace(auditTraceID) == "" {
		t.Fatal("registration audit event has no trace ID")
	}
}

// TestRegistrationPostgresRejectsOutdatedNotice proves the client notice
// version must match the current effective notice in the database.
func TestRegistrationPostgresRejectsOutdatedNotice(t *testing.T) {
	fixture := newRegistrationPostgresFixture(t)
	fixture.seedCurrentNotice(t, "v1.1.0")
	fixture.seedProcessingGateRecord(t, fixture.gateConfig)

	username := "ir02-outdated-" + randomHex(t, 4)
	response := serveRegistrationPostgres(
		t,
		fixture.handler,
		registrationPostgresBody(username, username+"@example.test", "+861380000"+randomHex(t, 2)),
	)
	requireRegistrationEnvelope(t, response, http.StatusConflict, platform.CodePrivacyNoticeOutdated)
	if count := fixture.countUsers(t); count != 0 {
		t.Fatalf("users after rejected registration = %d, want 0", count)
	}
}

// TestRegistrationPostgresRejectsUnacknowledgedNotice proves the request must
// carry an explicit acknowledgement before any write happens.
func TestRegistrationPostgresRejectsUnacknowledgedNotice(t *testing.T) {
	fixture := newRegistrationPostgresFixture(t)
	fixture.seedCurrentNotice(t, registrationPostgresNoticeVersion)
	fixture.seedProcessingGateRecord(t, fixture.gateConfig)

	username := "ir02-unacknowledged-" + randomHex(t, 4)
	body := registrationPostgresBody(
		username,
		username+"@example.test",
		"+861380000"+randomHex(t, 2),
	)
	body["privacy_acknowledged"] = false
	response := serveRegistrationPostgres(t, fixture.handler, body)

	requireRegistrationEnvelope(
		t,
		response,
		http.StatusForbidden,
		platform.CodePrivacyAcknowledgementRequired,
	)
	if count := fixture.countUsers(t); count != 0 {
		t.Fatalf("users after unacknowledged registration = %d, want 0", count)
	}
}

// TestRegistrationPostgresRejectsDuplicateIdentifier proves a duplicate
// username rolls the whole aggregate back and records a failure audit event.
func TestRegistrationPostgresRejectsDuplicateIdentifier(t *testing.T) {
	fixture := newRegistrationPostgresFixture(t)
	fixture.seedCurrentNotice(t, registrationPostgresNoticeVersion)
	fixture.seedProcessingGateRecord(t, fixture.gateConfig)

	username := "ir02-duplicate-" + randomHex(t, 4)
	first := serveRegistrationPostgres(
		t,
		fixture.handler,
		registrationPostgresBody(username, username+"@example.test", "+861380000"+randomHex(t, 2)),
	)
	if first.Code != http.StatusCreated {
		t.Fatalf("first registration status = %d; body = %s", first.Code, first.Body)
	}

	second := serveRegistrationPostgres(
		t,
		fixture.handler,
		registrationPostgresBody(strings.ToUpper(username), "other-"+username+"@example.test", "+861390000"+randomHex(t, 2)),
	)
	requireRegistrationEnvelope(t, second, http.StatusConflict, platform.CodeIdentityIdentifierConflict)

	if count := fixture.countUsers(t); count != 1 {
		t.Fatalf("users after duplicate registration = %d, want 1", count)
	}
	if count := fixture.countAuditEvents(t, registrationAuditOutcomeFailed); count != 1 {
		t.Fatalf("failed registration audit events = %d, want 1", count)
	}
}

// TestRegistrationPostgresRequiresProcessingGate proves registration stays
// closed without matching deployment processing evidence.
func TestRegistrationPostgresRequiresProcessingGate(t *testing.T) {
	t.Run("missing record", func(t *testing.T) {
		fixture := newRegistrationPostgresFixture(t)
		fixture.seedCurrentNotice(t, registrationPostgresNoticeVersion)

		username := "ir02-gate-missing-" + randomHex(t, 4)
		response := serveRegistrationPostgres(
			t,
			fixture.handler,
			registrationPostgresBody(username, username+"@example.test", "+861380000"+randomHex(t, 2)),
		)
		requireRegistrationEnvelope(
			t,
			response,
			http.StatusServiceUnavailable,
			platform.CodePlatformProcessingGateClosed,
		)
		if count := fixture.countUsers(t); count != 0 {
			t.Fatalf("users after closed gate = %d, want 0", count)
		}
	})

	t.Run("mismatched record", func(t *testing.T) {
		fixture := newRegistrationPostgresFixture(t)
		fixture.seedCurrentNotice(t, registrationPostgresNoticeVersion)
		mismatched := fixture.gateConfig
		mismatched.RegionRef = "cn-ir02-other-region"
		fixture.seedProcessingGateRecord(t, mismatched)

		username := "ir02-gate-mismatch-" + randomHex(t, 4)
		response := serveRegistrationPostgres(
			t,
			fixture.handler,
			registrationPostgresBody(username, username+"@example.test", "+861380000"+randomHex(t, 2)),
		)
		requireRegistrationEnvelope(
			t,
			response,
			http.StatusServiceUnavailable,
			platform.CodePlatformProcessingGateClosed,
		)
		if count := fixture.countUsers(t); count != 0 {
			t.Fatalf("users after mismatched gate = %d, want 0", count)
		}
	})
}
