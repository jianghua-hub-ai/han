//go:build integration

package identity

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"han/internal/session"
	"han/internal/testenv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	sqlStateUniqueViolation         = "23505"
	sqlStateForeignKeyViolation     = "23503"
	sqlStateCheckViolation          = "23514"
	sqlStateNotNullViolation        = "23502"
	sqlStateObjectStatePrerequisite = "55000"
)

func TestIdentityRepository_SchemaConstraints_EnforceIdentityInvariants(t *testing.T) {
	sharedMigration := readMigration(t, "../../migrations/0001_shared.sql")
	identityMigration := readMigration(t, "../../migrations/0002_identity.sql")
	ctx, tx, schema := newIdentityTestTransaction(t, sharedMigration, identityMigration)

	now := time.Date(2026, time.October, 21, 8, 0, 0, 0, time.UTC)
	memberID := randomUUID(t)
	otherMemberID := randomUUID(t)
	insertUser(t, ctx, tx, memberID, "member-one", "member", false, "active", "permanent", now)
	insertUser(t, ctx, tx, otherMemberID, "member-two", "member", false, "active", "permanent", now)

	t.Run("identifier uniqueness", func(t *testing.T) {
		testIdentifierUniqueness(t, ctx, tx, memberID, otherMemberID, now)
	})
	t.Run("single initial administrator", func(t *testing.T) {
		testSingleInitialAdministrator(t, ctx, tx, now)
	})
	t.Run("deleted user credential erasure", func(t *testing.T) {
		testDeletedUserCredentialErasure(t, ctx, tx, now)
	})
	t.Run("session token uniqueness and user revocation index", func(t *testing.T) {
		testSessionConstraints(t, ctx, tx, schema, memberID, otherMemberID, now)
	})
	t.Run("session temporary credential binding", func(t *testing.T) {
		testSessionTemporaryCredentialBinding(t, ctx, tx, memberID, otherMemberID, now)
	})
	t.Run("temporary credential current row uniqueness", func(t *testing.T) {
		testTemporaryCredentialUniqueness(t, ctx, tx, memberID, now)
	})
	t.Run("privacy evidence uniqueness", func(t *testing.T) {
		testPrivacyEvidenceUniqueness(t, ctx, tx, memberID, now)
	})
	t.Run("declaration and recovery evidence", func(t *testing.T) {
		testDeclarationAndRecoveryEvidence(t, ctx, tx, memberID, now)
	})
	t.Run("recovery decision evidence consistency", func(t *testing.T) {
		testRecoveryDecisionEvidenceConsistency(t, ctx, tx, memberID, now)
	})
	t.Run("login bucket keys and state", func(t *testing.T) {
		testLoginBucketConstraints(t, ctx, tx, schema, now)
	})
	t.Run("illegal enumerations", func(t *testing.T) {
		testIllegalEnumerations(t, ctx, tx, memberID, otherMemberID, now)
	})
	t.Run("identity foreign keys", func(t *testing.T) {
		testIdentityForeignKeys(t, ctx, tx, memberID, now)
	})
}

func TestIdentityRepository_UserGovernancePaginationIndexes(t *testing.T) {
	sharedMigration := readMigration(t, "../../migrations/0001_shared.sql")
	identityMigration := readMigration(t, "../../migrations/0002_identity.sql")
	governanceIndexMigration := readMigration(
		t,
		"../../migrations/0006_user_governance_indexes.sql",
	)
	ctx, tx, schema := newIdentityTestTransaction(t, sharedMigration, identityMigration)

	if _, err := tx.Exec(ctx, governanceIndexMigration); err != nil {
		t.Fatalf("apply user governance index migration: %v", err)
	}
	var postgresVersion string
	if err := tx.QueryRow(ctx, "SELECT current_setting('server_version')").Scan(
		&postgresVersion,
	); err != nil {
		t.Fatalf("read PostgreSQL version evidence: %v", err)
	}
	t.Logf("PostgreSQL version: %s", postgresVersion)

	requireIndex(
		t,
		ctx,
		tx,
		schema,
		"users",
		"users_role_status_created_at_id_idx",
		"(role, status, created_at DESC, id DESC)",
	)
	requireIndex(
		t,
		ctx,
		tx,
		schema,
		"users",
		"users_created_at_id_idx",
		"(created_at DESC, id DESC)",
	)

	testUserGovernanceKeysetScope(t, ctx, tx)
	testUserGovernanceIndexPlans(t, ctx, tx)
}

func testUserGovernanceKeysetScope(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
) {
	t.Helper()

	const (
		initialAdminID  = "00000000-0000-4000-8000-000000000001"
		administratorID = "00000000-0000-4000-8000-000000000010"
		otherAdminID    = "00000000-0000-4000-8000-000000000011"
	)
	sameCreatedAt := time.Date(2026, time.October, 23, 8, 0, 0, 0, time.UTC)
	olderCreatedAt := sameCreatedAt.Add(-time.Second)

	insertUser(
		t, ctx, tx, initialAdminID, "governance-initial-admin",
		"admin", true, "active", "bootstrap", sameCreatedAt,
	)
	insertUser(
		t, ctx, tx, administratorID, "governance-administrator",
		"admin", false, "active", "permanent", sameCreatedAt,
	)
	insertUser(
		t, ctx, tx, otherAdminID, "governance-other-admin",
		"admin", false, "suspended", "permanent", sameCreatedAt,
	)

	memberIDs := []string{
		"00000000-0000-4000-8000-000000000101",
		"00000000-0000-4000-8000-000000000102",
		"00000000-0000-4000-8000-000000000103",
		"00000000-0000-4000-8000-000000000104",
	}
	for index, id := range memberIDs {
		status := "active"
		if index%2 == 1 {
			status = "suspended"
		}
		insertUser(
			t,
			ctx,
			tx,
			id,
			fmt.Sprintf("governance-member-%d", index+1),
			"member",
			false,
			status,
			"permanent",
			sameCreatedAt,
		)
	}
	olderMemberID := "00000000-0000-4000-8000-000000000105"
	insertUser(
		t, ctx, tx, olderMemberID, "governance-member-older",
		"member", false, "active", "permanent", olderCreatedAt,
	)
	deletedAt := sameCreatedAt.Add(time.Second)
	deletedMemberID := "00000000-0000-4000-8000-000000000999"
	if err := (Repository{}).InsertUser(ctx, tx, User{
		ID: deletedMemberID, Username: "governance-member-deleted",
		Role: RoleMember, Status: UserStatusDeleted, CredentialMode: CredentialModePermanent,
		CreatedAt: sameCreatedAt, UpdatedAt: deletedAt, DeletedAt: &deletedAt,
	}); err != nil {
		t.Fatalf("insert deleted governance member: %v", err)
	}

	codec, err := NewIdentityCursorCodec(
		[]byte("integration-governance-cursor-key"),
		time.Hour,
		func() time.Time { return sameCreatedAt.Add(time.Minute) },
	)
	if err != nil {
		t.Fatalf("create governance cursor codec: %v", err)
	}
	query, err := NewUserQuery(tx, codec, UserQueryConfig{DefaultLimit: 2, MaxLimit: 10})
	if err != nil {
		t.Fatalf("create governance user query: %v", err)
	}
	ordinaryAdministrator := session.Principal{
		UserID: administratorID, Role: string(RoleAdmin), Status: string(UserStatusActive),
	}
	gotMemberIDs := collectUserGovernancePages(
		t,
		ctx,
		query,
		ordinaryAdministrator,
		"/api/v1/admin/users",
		UserQueryCategoryUsers,
		2,
	)
	wantMemberIDs := []string{
		memberIDs[3],
		memberIDs[2],
		memberIDs[1],
		memberIDs[0],
		olderMemberID,
	}
	requireStringSequence(t, "ordinary administrator member pages", gotMemberIDs, wantMemberIDs)

	initialAdministrator := session.Principal{
		UserID: initialAdminID, Role: string(RoleAdmin), IsInitialAdmin: true,
		Status: string(UserStatusActive),
	}
	gotAdministratorIDs := collectUserGovernancePages(
		t,
		ctx,
		query,
		initialAdministrator,
		"/api/v1/admin/administrators",
		UserQueryCategoryAdministrators,
		1,
	)
	requireStringSequence(
		t,
		"initial administrator pages",
		gotAdministratorIDs,
		[]string{otherAdminID, administratorID},
	)
}

func collectUserGovernancePages(
	t *testing.T,
	ctx context.Context,
	query *UserQuery,
	actor session.Principal,
	endpoint string,
	category UserQueryCategory,
	limit int,
) []string {
	t.Helper()

	var cursor string
	ids := make([]string, 0)
	seen := make(map[string]struct{})
	for pageNumber := 1; pageNumber <= 20; pageNumber++ {
		page, err := query.List(ctx, UserQueryListRequest{
			Actor: actor, Endpoint: endpoint, Category: category,
			Cursor: cursor, Limit: &limit, FilterDigest: identityUserQueryFilterDigest,
		})
		if err != nil {
			t.Fatalf("list governance page %d: %v", pageNumber, err)
		}
		for _, item := range page.Items {
			if _, duplicate := seen[item.ID]; duplicate {
				t.Fatalf("governance pagination returned duplicate id %s", item.ID)
			}
			seen[item.ID] = struct{}{}
			ids = append(ids, item.ID)
		}
		if !page.HasMore {
			if page.NextCursor != nil {
				t.Fatalf("last governance page next cursor = %q, want nil", *page.NextCursor)
			}
			return ids
		}
		if page.NextCursor == nil {
			t.Fatalf("governance page %d has_more=true with nil next cursor", pageNumber)
		}
		cursor = *page.NextCursor
	}
	t.Fatal("governance pagination exceeded 20 pages")
	return nil
}

func requireStringSequence(t *testing.T, label string, got []string, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("%s ids = %v, want %v", label, got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("%s ids = %v, want %v", label, got, want)
		}
	}
}

func testUserGovernanceIndexPlans(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
) {
	t.Helper()

	const planRows = 20000
	if _, err := tx.Exec(ctx, `
		INSERT INTO users (
			id,
			username,
			email,
			phone,
			role,
			is_initial_admin,
			status,
			credential_mode,
			recovery_required,
			created_at,
			updated_at
		)
		SELECT
			(
				substr(md5('governance-plan-' || value::text), 1, 8) || '-' ||
				substr(md5('governance-plan-' || value::text), 9, 4) || '-4' ||
				substr(md5('governance-plan-' || value::text), 14, 3) || '-8' ||
				substr(md5('governance-plan-' || value::text), 18, 3) || '-' ||
				substr(md5('governance-plan-' || value::text), 21, 12)
			)::uuid,
			'governance-plan-' || value::text,
			'governance-plan-' || value::text || '@example.test',
			'+1555' || lpad(value::text, 8, '0'),
			CASE WHEN value % 100 = 0 THEN 'admin' ELSE 'member' END,
			false,
			CASE WHEN value % 100 = 0 THEN 'suspended' ELSE 'active' END,
			'permanent',
			false,
			timestamptz '2026-10-22 00:00:00+00' - value * interval '1 second',
			timestamptz '2026-10-22 00:00:00+00' - value * interval '1 second'
		FROM generate_series(1, $1) AS generated(value)
	`, planRows); err != nil {
		t.Fatalf("insert governance plan fixture: %v", err)
	}
	if _, err := tx.Exec(ctx, "ANALYZE users"); err != nil {
		t.Fatalf("analyze governance plan fixture: %v", err)
	}

	roleStatusPlan := explainPlan(
		t,
		ctx,
		tx,
		`SELECT id
		 FROM users
		 WHERE role = $1 AND status = $2
		 ORDER BY created_at DESC, id DESC
		 LIMIT 25`,
		"admin",
		"suspended",
	)
	requirePlanIndex(
		t,
		"role/status governance path",
		roleStatusPlan,
		"users_role_status_created_at_id_idx",
	)

	unfilteredPlan := explainPlan(
		t,
		ctx,
		tx,
		`SELECT id
		 FROM users
		 ORDER BY created_at DESC, id DESC
		 LIMIT 25`,
	)
	requirePlanIndex(
		t,
		"unfiltered governance ordering path",
		unfilteredPlan,
		"users_created_at_id_idx",
	)

	finalQueryPlan := explainPlan(
		t,
		ctx,
		tx,
		listUserGovernanceSQL,
		RoleMember,
		"00000000-0000-4000-8000-000000000010",
		nil,
		nil,
		25,
	)
	if !strings.Contains(finalQueryPlan, "users_role_status_created_at_id_idx") &&
		!strings.Contains(finalQueryPlan, "users_created_at_id_idx") {
		t.Fatalf("final S7.1 list plan does not use a governance index:\n%s", finalQueryPlan)
	}
	t.Logf("role/status plan:\n%s", roleStatusPlan)
	t.Logf("unfiltered ordering plan:\n%s", unfilteredPlan)
	t.Logf("final S7.1 list plan:\n%s", finalQueryPlan)
}

func explainPlan(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	query string,
	args ...any,
) string {
	t.Helper()

	rows, err := tx.Query(ctx, "EXPLAIN (ANALYZE, BUFFERS) "+query, args...)
	if err != nil {
		t.Fatalf("explain governance query: %v", err)
	}
	defer rows.Close()

	var plan strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatalf("scan governance query plan: %v", err)
		}
		plan.WriteString(line)
		plan.WriteByte('\n')
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read governance query plan: %v", err)
	}
	return plan.String()
}

func requirePlanIndex(t *testing.T, label string, plan string, index string) {
	t.Helper()

	if !strings.Contains(plan, index) {
		t.Fatalf("%s plan does not use %s:\n%s", label, index, plan)
	}
}

func testIdentifierUniqueness(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	memberID string,
	otherMemberID string,
	now time.Time,
) {
	insertIdentifier(t, ctx, tx, memberID, "username", "member-one", "active", now)
	requireSQLState(
		t, ctx, tx, sqlStateUniqueViolation,
		`INSERT INTO identity_identifiers
			(user_id, kind, normalized_value, lifecycle, created_at)
		 VALUES ($1, 'username', 'member-one', 'active', $2)`,
		otherMemberID, now,
	)
}

func testSingleInitialAdministrator(t *testing.T, ctx context.Context, tx pgx.Tx, now time.Time) {
	insertUser(t, ctx, tx, randomUUID(t), "initial-admin-one", "admin", true, "active", "bootstrap", now)
	requireSQLState(
		t, ctx, tx, sqlStateUniqueViolation, userInsertSQL,
		randomUUID(t), "initial-admin-two", "initial-admin-two@example.test", "+15550000002",
		"admin", true, "active", "bootstrap", now, now,
	)
}

func testDeletedUserCredentialErasure(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	now time.Time,
) {
	deletedAt := now.Add(time.Minute)
	user := User{
		ID: randomUUID(t), Username: "deleted-user-valid",
		Role: RoleMember, Status: UserStatusDeleted, CredentialMode: CredentialModePermanent,
		CreatedAt: now, UpdatedAt: deletedAt, DeletedAt: &deletedAt,
	}
	if err := (Repository{}).InsertUser(ctx, tx, user); err != nil {
		t.Fatalf("insert valid deleted user tombstone: %v", err)
	}

	invalidValues := [][]any{
		{"deleted-email@example.test", nil, nil},
		{nil, "+15550000009", nil},
		{nil, nil, "$argon2id$synthetic-deleted"},
	}
	for index, values := range invalidValues {
		requireSQLState(
			t, ctx, tx, sqlStateCheckViolation, deletedUserInsertSQL,
			randomUUID(t), fmt.Sprintf("deleted-user-invalid-%d", index),
			values[0], values[1], values[2], now, deletedAt,
		)
	}
}

func testSessionConstraints(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	schema string,
	memberID string,
	otherMemberID string,
	now time.Time,
) {
	insertSession(t, ctx, tx, randomUUID(t), memberID, []byte("session-token-one"), now)
	requireSQLState(
		t, ctx, tx, sqlStateUniqueViolation, sessionInsertSQL,
		randomUUID(t), memberID, []byte("session-token-one"), []byte("csrf-token-two"),
		"permanent", nil, now, now, now.Add(time.Hour), now.Add(30*time.Minute),
	)
	insertSession(t, ctx, tx, randomUUID(t), otherMemberID, []byte("session-token-other"), now)
	requireIndex(t, ctx, tx, schema, "sessions", "sessions_user_id_idx", "(user_id)")

	count, err := (Repository{}).RevokeSessionsByUserID(ctx, tx, memberID, now.Add(time.Minute), "governance")
	if err != nil {
		t.Fatalf("revoke sessions by user: %v", err)
	}
	if count != 1 {
		t.Fatalf("revoked session count = %d, want 1", count)
	}
	assertSessionRevocation(t, ctx, tx, memberID, otherMemberID)
}

func assertSessionRevocation(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	memberID string,
	otherMemberID string,
) {
	t.Helper()
	var memberRevoked, otherRevoked bool
	if err := tx.QueryRow(ctx, `
		SELECT
			bool_and(revoked_at IS NOT NULL) FILTER (WHERE user_id = $1),
			bool_and(revoked_at IS NOT NULL) FILTER (WHERE user_id = $2)
		FROM sessions
	`, memberID, otherMemberID).Scan(&memberRevoked, &otherRevoked); err != nil {
		t.Fatalf("query session revocation state: %v", err)
	}
	if !memberRevoked || otherRevoked {
		t.Fatalf("session revocation state = member:%t other:%t, want true/false", memberRevoked, otherRevoked)
	}
}

func testSessionTemporaryCredentialBinding(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	memberID string,
	otherMemberID string,
	now time.Time,
) {
	credentialID := randomUUID(t)
	insertTemporaryCredential(
		t, ctx, tx, credentialID, otherMemberID, "password_reset", 1, "active", now,
	)
	insertTemporarySession(t, ctx, tx, randomUUID(t), otherMemberID, credentialID, now)

	requireSQLState(
		t, ctx, tx, sqlStateCheckViolation, sessionInsertSQL,
		randomUUID(t), memberID, []byte("temporary-without-id"), []byte("csrf-temporary-without-id"),
		"temporary", nil, now, now, now.Add(time.Hour), now.Add(30*time.Minute),
	)
	requireSQLState(
		t, ctx, tx, sqlStateCheckViolation, sessionInsertSQL,
		randomUUID(t), otherMemberID, []byte("permanent-with-id"), []byte("csrf-permanent-with-id"),
		"permanent", credentialID, now, now, now.Add(time.Hour), now.Add(30*time.Minute),
	)
	requireSQLState(
		t, ctx, tx, sqlStateForeignKeyViolation, sessionInsertSQL,
		randomUUID(t), memberID, []byte("temporary-dangling-id"), []byte("csrf-temporary-dangling-id"),
		"temporary", randomUUID(t), now, now, now.Add(time.Hour), now.Add(30*time.Minute),
	)
	requireSQLState(
		t, ctx, tx, sqlStateForeignKeyViolation, sessionInsertSQL,
		randomUUID(t), memberID, []byte("temporary-cross-user"), []byte("csrf-temporary-cross-user"),
		"temporary", credentialID, now, now, now.Add(time.Hour), now.Add(30*time.Minute),
	)
}

func testTemporaryCredentialUniqueness(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	memberID string,
	now time.Time,
) {
	insertTemporaryCredential(t, ctx, tx, randomUUID(t), memberID, "password_reset", 1, "active", now)
	requireSQLState(
		t, ctx, tx, sqlStateUniqueViolation, temporaryCredentialInsertSQL,
		randomUUID(t), memberID, "password_reset", 2, "expired",
		"$argon2id$synthetic-second", "member-one@example.test", now, now.Add(time.Hour),
	)
	insertTemporaryCredential(t, ctx, tx, randomUUID(t), memberID, "password_reset", 2, "superseded", now)
}

func testPrivacyEvidenceUniqueness(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	memberID string,
	now time.Time,
) {
	repository := Repository{}
	notice := PrivacyNotice{
		Version: "notice-v1", Content: "synthetic privacy notice",
		ContentHash: []byte("notice-hash-v1"), EffectiveAt: now, Status: PrivacyNoticeCurrent,
	}
	if err := repository.InsertPrivacyNotice(ctx, tx, notice); err != nil {
		t.Fatalf("insert privacy notice: %v", err)
	}
	requireSQLState(
		t, ctx, tx, sqlStateUniqueViolation,
		`INSERT INTO privacy_notices (version, content, content_hash, effective_at, status)
		 VALUES ('notice-v2', 'new synthetic notice', $1, $2, 'current')`,
		[]byte("notice-hash-v2"), now.Add(time.Hour),
	)
	acknowledgement := PrivacyAcknowledgement{
		ID: randomUUID(t), UserID: memberID, NoticeVersion: "notice-v1",
		AcknowledgedAt: now, SourceKind: "registration", SourceFingerprint: "source-fingerprint-one",
	}
	if err := repository.InsertPrivacyAcknowledgement(ctx, tx, acknowledgement); err != nil {
		t.Fatalf("insert privacy acknowledgement: %v", err)
	}
	requireSQLState(
		t, ctx, tx, sqlStateUniqueViolation,
		`INSERT INTO privacy_acknowledgements (
			id, user_id, notice_version, acknowledged_at, source_kind, source_fingerprint
		) VALUES ($1, $2, 'notice-v1', $3, 'registration', 'source-fingerprint-two')`,
		randomUUID(t), memberID, now.Add(time.Minute),
	)
	requireSQLState(
		t, ctx, tx, sqlStateObjectStatePrerequisite,
		"UPDATE privacy_notices SET content = 'mutated' WHERE version = 'notice-v1'",
	)
	requireSQLState(
		t, ctx, tx, sqlStateObjectStatePrerequisite,
		"UPDATE privacy_acknowledgements SET source_kind = 'mutated'",
	)
}

func testDeclarationAndRecoveryEvidence(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	now time.Time,
) {
	repository := Repository{}
	declaration := AdministratorInformationDeclaration{
		ID: randomUUID(t), ActorUserID: userID,
		TargetLocatorDigest: []byte("synthetic-target-digest"),
		StatementVersion:    "statement-v1", DeclaredAt: now, Purpose: "account_creation",
	}
	if err := repository.InsertAdministratorInformationDeclaration(ctx, tx, declaration); err != nil {
		t.Fatalf("insert administrator information declaration: %v", err)
	}
	requireSQLState(
		t, ctx, tx, sqlStateObjectStatePrerequisite,
		"DELETE FROM administrator_information_declarations WHERE id = $1",
		declaration.ID,
	)
	review := RecoveryAccountReview{
		RecoveryRunID: "recovery-run-one", UserID: userID,
		BackupRole: RoleMember, BackupStatus: UserStatusActive, Decision: RecoveryDecisionPending,
	}
	if err := repository.InsertRecoveryAccountReview(ctx, tx, review); err != nil {
		t.Fatalf("insert recovery account review: %v", err)
	}
	requireSQLState(
		t, ctx, tx, sqlStateUniqueViolation,
		`INSERT INTO recovery_account_reviews (
			recovery_run_id, user_id, backup_role, backup_status, decision
		) VALUES ('recovery-run-one', $1, 'member', 'active', 'pending')`,
		userID,
	)
}

func testRecoveryDecisionEvidenceConsistency(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	now time.Time,
) {
	testLegalRecoveryDecisionEvidence(t, ctx, tx, userID, now)
	testIllegalRecoveryDecisionEvidence(t, ctx, tx, userID, now)
}

func testLegalRecoveryDecisionEvidence(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	now time.Time,
) {
	reviewer := userID
	reviewedAt := now.Add(time.Minute)
	resetAt := now.Add(2 * time.Minute)
	cases := []RecoveryAccountReview{
		{RecoveryRunID: "recovery-legal-pending", UserID: userID, BackupRole: RoleMember,
			BackupStatus: UserStatusActive, Decision: RecoveryDecisionPending},
		{RecoveryRunID: "recovery-legal-restored", UserID: userID, BackupRole: RoleMember,
			BackupStatus: UserStatusActive, Decision: RecoveryDecisionRestored,
			ReviewedBy: &reviewer, ReviewedAt: &reviewedAt, CredentialResetAt: &resetAt},
		{RecoveryRunID: "recovery-legal-kept-deleted", UserID: userID, BackupRole: RoleMember,
			BackupStatus: UserStatusDeleted, BackupDeletedAt: &now, Decision: RecoveryDecisionKeptDeleted,
			ReviewedBy: &reviewer, ReviewedAt: &reviewedAt},
		{RecoveryRunID: "recovery-legal-denied", UserID: userID, BackupRole: RoleMember,
			BackupStatus: UserStatusActive, Decision: RecoveryDecisionDenied,
			ReviewedBy: &reviewer, ReviewedAt: &reviewedAt},
	}
	for _, review := range cases {
		if err := (Repository{}).InsertRecoveryAccountReview(ctx, tx, review); err != nil {
			t.Fatalf("insert legal recovery decision %s: %v", review.Decision, err)
		}
	}
}

func testIllegalRecoveryDecisionEvidence(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	now time.Time,
) {
	cases := illegalRecoveryDecisionCases(userID, now)
	for index, testCase := range cases {
		requireSQLState(
			t, ctx, tx, sqlStateCheckViolation, recoveryReviewInsertSQL,
			fmt.Sprintf("recovery-illegal-%d", index), userID, testCase.decision,
			testCase.reviewedBy, testCase.reviewedAt, testCase.credentialResetAt,
		)
	}
}

type recoveryDecisionCase struct {
	decision          string
	reviewedBy        any
	reviewedAt        any
	credentialResetAt any
}

func illegalRecoveryDecisionCases(userID string, now time.Time) []recoveryDecisionCase {
	return []recoveryDecisionCase{
		{decision: "pending", reviewedBy: userID},
		{decision: "pending", reviewedAt: now},
		{decision: "pending", credentialResetAt: now},
		{decision: "restored", reviewedAt: now, credentialResetAt: now},
		{decision: "restored", reviewedBy: userID, credentialResetAt: now},
		{decision: "restored", reviewedBy: userID, reviewedAt: now},
		{decision: "kept_deleted", reviewedAt: now},
		{decision: "kept_deleted", reviewedBy: userID},
		{decision: "kept_deleted", reviewedBy: userID, reviewedAt: now, credentialResetAt: now},
		{decision: "denied", reviewedAt: now},
		{decision: "denied", reviewedBy: userID},
		{decision: "denied", reviewedBy: userID, reviewedAt: now, credentialResetAt: now},
	}
}

func testLoginBucketConstraints(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	schema string,
	now time.Time,
) {
	insertLoginBucket(t, ctx, tx, "subject", []byte("subject-digest"), "v1", now)
	requireSQLState(
		t, ctx, tx, sqlStateUniqueViolation, loginBucketInsertSQL,
		"subject", []byte("subject-digest"), "v1", now, 1, now, now,
	)
	requireSQLState(
		t, ctx, tx, sqlStateNotNullViolation, loginBucketInsertSQL,
		"source", nil, "v1", now, 0, now, now,
	)
	requireSQLState(
		t, ctx, tx, sqlStateCheckViolation, loginBucketInsertSQL,
		"source", []byte("source-digest"), "v1", now, -1, now, now,
	)
	requireIndex(
		t, ctx, tx, schema, "login_attempt_buckets",
		"login_attempt_buckets_blocked_until_idx", "(blocked_until)",
	)
}

func testIllegalEnumerations(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	memberID string,
	otherMemberID string,
	now time.Time,
) {
	testIllegalUserEnumerations(t, ctx, tx, now)
	testIllegalIdentifierEnumerations(t, ctx, tx, memberID, now)
	testIllegalCredentialEnumerations(t, ctx, tx, otherMemberID, now)
	testIllegalBucketAndRecoveryEnumerations(t, ctx, tx, memberID, now)
}

func testIllegalUserEnumerations(t *testing.T, ctx context.Context, tx pgx.Tx, now time.Time) {
	cases := [][]any{
		{randomUUID(t), "invalid-role", "invalid-role@example.test", "+15550000003",
			"owner", false, "active", "permanent", now, now},
		{randomUUID(t), "invalid-status", "invalid-status@example.test", "+15550000004",
			"member", false, "pending", "permanent", now, now},
		{randomUUID(t), "invalid-mode", "invalid-mode@example.test", "+15550000005",
			"member", false, "active", "fallback", now, now},
	}
	for _, args := range cases {
		requireSQLState(t, ctx, tx, sqlStateCheckViolation, userInsertSQL, args...)
	}
}

func testIllegalIdentifierEnumerations(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	memberID string,
	now time.Time,
) {
	requireSQLState(
		t, ctx, tx, sqlStateCheckViolation,
		`INSERT INTO identity_identifiers
			(user_id, kind, normalized_value, lifecycle, created_at)
		 VALUES ($1, 'external_id', 'synthetic', 'active', $2)`,
		memberID, now,
	)
	requireSQLState(
		t, ctx, tx, sqlStateCheckViolation,
		`INSERT INTO identity_identifiers
			(user_id, kind, normalized_value, lifecycle, created_at)
		 VALUES ($1, 'email', 'invalid-lifecycle@example.test', 'released', $2)`,
		memberID, now,
	)
}

func testIllegalCredentialEnumerations(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	now time.Time,
) {
	requireSQLState(
		t, ctx, tx, sqlStateCheckViolation, sessionInsertSQL,
		randomUUID(t), userID, []byte("session-token-invalid-kind"), []byte("csrf-token-invalid-kind"),
		"recovery_locked", nil, now, now, now.Add(time.Hour), now.Add(30*time.Minute),
	)
	requireSQLState(
		t, ctx, tx, sqlStateCheckViolation, temporaryCredentialInsertSQL,
		randomUUID(t), userID, "login", 1, "active",
		"$argon2id$synthetic-invalid-purpose", "member-two@example.test", now, now.Add(time.Hour),
	)
	requireSQLState(
		t, ctx, tx, sqlStateCheckViolation, temporaryCredentialInsertSQL,
		randomUUID(t), userID, "password_reset", 1, "revoked",
		"$argon2id$synthetic-invalid-state", "member-two@example.test", now, now.Add(time.Hour),
	)
}

func testIllegalBucketAndRecoveryEnumerations(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	now time.Time,
) {
	requireSQLState(
		t, ctx, tx, sqlStateCheckViolation, loginBucketInsertSQL,
		"global", []byte("global-digest"), "v1", now, 0, now, now,
	)
	requireSQLState(
		t, ctx, tx, sqlStateCheckViolation,
		`INSERT INTO recovery_account_reviews (
			recovery_run_id, user_id, backup_role, backup_status, decision
		) VALUES ('recovery-run-one', $1, 'member', 'active', 'approved')`,
		userID,
	)
}

func testIdentityForeignKeys(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	now time.Time,
) {
	testUnknownUserForeignKeys(t, ctx, tx, now)
	testNoticeRotationAndReviewerForeignKeys(t, ctx, tx, userID, now)
}

func testUnknownUserForeignKeys(t *testing.T, ctx context.Context, tx pgx.Tx, now time.Time) {
	unknownUserID := randomUUID(t)
	requireSQLState(
		t, ctx, tx, sqlStateForeignKeyViolation,
		`INSERT INTO identity_identifiers
			(user_id, kind, normalized_value, lifecycle, created_at)
		 VALUES ($1, 'username', 'unknown-user', 'active', $2)`,
		unknownUserID, now,
	)
	requireSQLState(
		t, ctx, tx, sqlStateForeignKeyViolation, sessionInsertSQL,
		randomUUID(t), unknownUserID, []byte("unknown-user-session"), []byte("unknown-user-csrf"),
		"permanent", nil, now, now, now.Add(time.Hour), now.Add(30*time.Minute),
	)
	requireSQLState(
		t, ctx, tx, sqlStateForeignKeyViolation, temporaryCredentialInsertSQL,
		randomUUID(t), unknownUserID, "password_reset", 1, "active",
		"$argon2id$synthetic-unknown-user", "unknown-user@example.test", now, now.Add(time.Hour),
	)
	requirePrivacyAcknowledgementUserForeignKey(t, ctx, tx, unknownUserID, now)
	requireAdministratorDeclarationUserForeignKey(t, ctx, tx, unknownUserID, now)
	requireSQLState(
		t, ctx, tx, sqlStateForeignKeyViolation,
		`INSERT INTO recovery_account_reviews (
			recovery_run_id, user_id, backup_role, backup_status, decision
		) VALUES ('unknown-review-user', $1, 'member', 'active', 'pending')`,
		unknownUserID,
	)
}

func requirePrivacyAcknowledgementUserForeignKey(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	unknownUserID string,
	now time.Time,
) {
	t.Helper()
	requireSQLState(
		t, ctx, tx, sqlStateForeignKeyViolation,
		`INSERT INTO privacy_acknowledgements (
			id, user_id, notice_version, acknowledged_at, source_kind, source_fingerprint
		) VALUES ($1, $2, 'notice-v1', $3, 'registration', 'unknown-user-source')`,
		randomUUID(t), unknownUserID, now,
	)
}

func requireAdministratorDeclarationUserForeignKey(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	unknownUserID string,
	now time.Time,
) {
	t.Helper()
	requireSQLState(
		t, ctx, tx, sqlStateForeignKeyViolation,
		`INSERT INTO administrator_information_declarations (
			id, actor_user_id, target_locator_digest, statement_version, declared_at, purpose
		) VALUES ($1, $2, $3, 'statement-v1', $4, 'account_creation')`,
		randomUUID(t), unknownUserID, []byte("unknown-user-target"), now,
	)
}

func testNoticeRotationAndReviewerForeignKeys(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	now time.Time,
) {
	requireSQLState(
		t, ctx, tx, sqlStateForeignKeyViolation,
		`INSERT INTO privacy_acknowledgements (
			id, user_id, notice_version, acknowledged_at, source_kind, source_fingerprint
		) VALUES ($1, $2, 'unknown-notice', $3, 'registration', 'unknown-notice-source')`,
		randomUUID(t), userID, now,
	)
	requireSQLState(
		t, ctx, tx, sqlStateForeignKeyViolation, rotatedSessionInsertSQL,
		randomUUID(t), userID, []byte("unknown-rotation-token"), []byte("unknown-rotation-csrf"),
		now, now.Add(time.Hour), now.Add(30*time.Minute), randomUUID(t),
	)
	requireSQLState(
		t, ctx, tx, sqlStateForeignKeyViolation, recoveryReviewInsertSQL,
		"unknown-reviewer", userID, "denied", randomUUID(t), now, nil,
	)
}

const userInsertSQL = `
	INSERT INTO users (
		id,
		username,
		email,
		phone,
		role,
		is_initial_admin,
		status,
		credential_mode,
		recovery_required,
		created_at,
		updated_at
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, false, $9, $10)
`

const deletedUserInsertSQL = `
	INSERT INTO users (
		id,
		username,
		email,
		phone,
		permanent_password_digest,
		role,
		is_initial_admin,
		status,
		credential_mode,
		recovery_required,
		created_at,
		updated_at,
		deleted_at
	)
	VALUES ($1, $2, $3, $4, $5, 'member', false, 'deleted', 'permanent', false, $6, $7, $7)
`

const sessionInsertSQL = `
	INSERT INTO sessions (
		id,
		user_id,
		token_digest,
		csrf_token_digest,
		authenticated_credential_kind,
		authenticated_credential_id,
		issued_at,
		last_seen_at,
		absolute_expires_at,
		idle_expires_at
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
`

const rotatedSessionInsertSQL = `
	INSERT INTO sessions (
		id,
		user_id,
		token_digest,
		csrf_token_digest,
		authenticated_credential_kind,
		issued_at,
		last_seen_at,
		absolute_expires_at,
		idle_expires_at,
		rotated_from_id
	)
	VALUES ($1, $2, $3, $4, 'permanent', $5, $5, $6, $7, $8)
`

const temporaryCredentialInsertSQL = `
	INSERT INTO temporary_credentials (
		id,
		user_id,
		purpose,
		generation,
		state,
		secret_digest,
		delivered_to_email,
		issued_at,
		expires_at
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
`

const recoveryReviewInsertSQL = `
	INSERT INTO recovery_account_reviews (
		recovery_run_id,
		user_id,
		backup_role,
		backup_status,
		decision,
		reviewed_by,
		reviewed_at,
		credential_reset_at
	)
	VALUES ($1, $2, 'member', 'active', $3, $4, $5, $6)
`

const loginBucketInsertSQL = `
	INSERT INTO login_attempt_buckets (
		bucket_kind,
		bucket_key_digest,
		digest_key_version,
		window_started_at,
		failure_count,
		blocked_until,
		updated_at
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
`

func readMigration(t *testing.T, migrationPath string) string {
	t.Helper()
	content, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read migration %s: %v", migrationPath, err)
	}
	return string(content)
}

func newIdentityTestTransaction(
	t *testing.T,
	sharedMigration string,
	identityMigration string,
) (context.Context, pgx.Tx, string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	pool := newIdentityTestPool(t, ctx)
	tx := beginIdentitySchemaTransaction(t, ctx, pool)
	schema := "identity_" + randomHex(t, 8)
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	registerIdentitySchemaRollback(t, pool, tx, schema)

	if _, err := tx.Exec(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		t.Fatalf("create transaction-scoped identity schema: %v", err)
	}
	if _, err := tx.Exec(ctx, "SET LOCAL search_path TO "+quotedSchema); err != nil {
		t.Fatalf("set transaction-scoped identity search path: %v", err)
	}
	if _, err := tx.Exec(ctx, sharedMigration); err != nil {
		t.Fatalf("apply shared migration in identity test transaction: %v", err)
	}
	if _, err := tx.Exec(ctx, identityMigration); err != nil {
		t.Fatalf("apply identity migration in identity test transaction: %v", err)
	}

	return ctx, tx, schema
}

func newIdentityTestPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()

	dsn := testenv.RequirePostgresDSN(t)
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("configure PostgreSQL identity test database")
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal("open PostgreSQL identity test database")
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatal("connect PostgreSQL identity test database")
	}
	requirePostgres18(t, ctx, pool)
	return pool
}

func beginIdentitySchemaTransaction(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
) pgx.Tx {
	t.Helper()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin identity schema test transaction: %v", err)
	}
	return tx
}

func registerIdentitySchemaRollback(
	t *testing.T,
	pool *pgxpool.Pool,
	tx pgx.Tx,
	schema string,
) {
	t.Helper()
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := tx.Rollback(cleanupCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("rollback identity schema test transaction: %v", err)
			return
		}
		var remaining *string
		if err := pool.QueryRow(
			cleanupCtx, "SELECT to_regnamespace($1)::text", schema,
		).Scan(&remaining); err != nil {
			t.Errorf("verify identity schema rollback: %v", err)
			return
		}
		if remaining != nil {
			t.Errorf("identity schema remains after rollback: %s", *remaining)
		}
	})
}

func requirePostgres18(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	var version int
	if err := pool.QueryRow(ctx, "SELECT current_setting('server_version_num')::integer").Scan(&version); err != nil {
		t.Fatalf("read PostgreSQL identity test server version: %v", err)
	}
	if major := version / 10000; major != 18 {
		t.Fatalf("PostgreSQL identity test major = %d, want 18", major)
	}
}

func insertUser(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	id string,
	username string,
	role string,
	initialAdmin bool,
	status string,
	credentialMode string,
	now time.Time,
) {
	t.Helper()
	email := username + "@example.test"
	phone := "+15550000001"
	user := User{
		ID: id, Username: username, Email: &email, Phone: &phone,
		Role: Role(role), IsInitialAdmin: initialAdmin, Status: UserStatus(status),
		CredentialMode: CredentialMode(credentialMode), CreatedAt: now, UpdatedAt: now,
	}
	if err := (Repository{}).InsertUser(ctx, tx, user); err != nil {
		t.Fatalf("insert synthetic user %s: %v", username, err)
	}
}

func insertIdentifier(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	kind string,
	normalizedValue string,
	lifecycle string,
	now time.Time,
) {
	t.Helper()
	identifier := IdentityIdentifier{
		UserID: userID, Kind: IdentifierKind(kind), NormalizedValue: normalizedValue,
		Lifecycle: IdentifierLifecycle(lifecycle), CreatedAt: now,
	}
	if err := (Repository{}).InsertIdentifier(ctx, tx, identifier); err != nil {
		t.Fatalf("insert synthetic identity identifier: %v", err)
	}
}

func insertSession(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	id string,
	userID string,
	tokenDigest []byte,
	now time.Time,
) {
	t.Helper()
	session := Session{
		ID: id, UserID: userID, TokenDigest: tokenDigest,
		CSRFTokenDigest:             []byte("csrf-token-one"),
		AuthenticatedCredentialKind: SessionCredentialPermanent,
		IssuedAt:                    now, LastSeenAt: now,
		AbsoluteExpiresAt: now.Add(time.Hour), IdleExpiresAt: now.Add(30 * time.Minute),
	}
	if err := (Repository{}).InsertSession(ctx, tx, session); err != nil {
		t.Fatalf("insert synthetic session: %v", err)
	}
}

func insertTemporarySession(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	id string,
	userID string,
	credentialID string,
	now time.Time,
) {
	t.Helper()
	session := Session{
		ID: id, UserID: userID,
		TokenDigest: []byte("temporary-session-token"), CSRFTokenDigest: []byte("temporary-session-csrf"),
		AuthenticatedCredentialKind: SessionCredentialTemporary,
		AuthenticatedCredentialID:   &credentialID,
		IssuedAt:                    now, LastSeenAt: now,
		AbsoluteExpiresAt: now.Add(time.Hour), IdleExpiresAt: now.Add(30 * time.Minute),
	}
	if err := (Repository{}).InsertSession(ctx, tx, session); err != nil {
		t.Fatalf("insert temporary credential session: %v", err)
	}
}

func insertTemporaryCredential(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	id string,
	userID string,
	purpose string,
	generation int,
	state string,
	now time.Time,
) {
	t.Helper()
	credential := TemporaryCredential{
		ID: id, UserID: userID, Purpose: TemporaryCredentialPurpose(purpose),
		Generation: int64(generation), State: TemporaryCredentialState(state),
		SecretDigest:     "$argon2id$synthetic-" + state,
		DeliveredToEmail: "synthetic-recipient@example.test",
		IssuedAt:         now, ExpiresAt: now.Add(time.Hour),
	}
	if err := (Repository{}).InsertTemporaryCredential(ctx, tx, credential); err != nil {
		t.Fatalf("insert synthetic temporary credential: %v", err)
	}
}

func insertLoginBucket(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	kind string,
	keyDigest []byte,
	keyVersion string,
	now time.Time,
) {
	t.Helper()
	bucket := LoginAttemptBucket{
		Kind: LoginBucketKind(kind), KeyDigest: keyDigest, DigestKeyVersion: keyVersion,
		WindowStartedAt: now, FailureCount: 0, BlockedUntil: now, UpdatedAt: now,
	}
	if err := (Repository{}).InsertLoginAttemptBucket(ctx, tx, bucket); err != nil {
		t.Fatalf("insert synthetic login bucket: %v", err)
	}
}

func requireSQLState(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	wantCode string,
	query string,
	args ...any,
) {
	t.Helper()

	const savepoint = "identity_constraint_probe"
	if _, err := tx.Exec(ctx, "SAVEPOINT "+savepoint); err != nil {
		t.Fatalf("create constraint probe savepoint: %v", err)
	}
	_, execErr := tx.Exec(ctx, query, args...)
	var postgresError *pgconn.PgError
	matched := errors.As(execErr, &postgresError) && postgresError.Code == wantCode
	if _, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT "+savepoint); err != nil {
		t.Fatalf("rollback constraint probe savepoint: %v", err)
	}
	if _, err := tx.Exec(ctx, "RELEASE SAVEPOINT "+savepoint); err != nil {
		t.Fatalf("release constraint probe savepoint: %v", err)
	}
	if execErr == nil {
		t.Fatalf("constraint probe succeeded, want SQLSTATE %s", wantCode)
	}
	if !matched {
		t.Fatalf("constraint probe SQLSTATE = %q, want %q", postgresErrorCode(execErr), wantCode)
	}
}

func requireIndex(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	schema string,
	table string,
	index string,
	wantFragment string,
) {
	t.Helper()

	var definition string
	if err := tx.QueryRow(ctx, `
		SELECT indexdef
		FROM pg_indexes
		WHERE schemaname = $1 AND tablename = $2 AND indexname = $3
	`, schema, table, index).Scan(&definition); err != nil {
		t.Fatalf("query index %s: %v", index, err)
	}
	if !strings.Contains(definition, wantFragment) {
		t.Fatalf("index %s definition = %q, want fragment %q", index, definition, wantFragment)
	}
}

func postgresErrorCode(err error) string {
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		return ""
	}
	return postgresError.Code
}

func randomUUID(t *testing.T) string {
	t.Helper()

	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		t.Fatalf("generate synthetic UUID: %v", err)
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(value[:])
	return strings.Join([]string{
		encoded[0:8],
		encoded[8:12],
		encoded[12:16],
		encoded[16:20],
		encoded[20:32],
	}, "-")
}

func randomHex(t *testing.T, size int) string {
	t.Helper()

	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		t.Fatalf("generate identity schema suffix: %v", err)
	}
	return hex.EncodeToString(value)
}
