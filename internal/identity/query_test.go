package identity

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"han/internal/platform"
	"han/internal/session"
)

var _ UserQueryDatabase = (*pgxpool.Pool)(nil)

func TestUserQueryListUsesStableScopeAndCursor(t *testing.T) {
	now := time.Date(2026, 10, 23, 10, 0, 0, 0, time.UTC)
	database := &userQueryDatabaseStub{
		listRecords: []userQueryRecord{
			queryRecord("user-03", "charlie", RoleMember, now, CredentialModePermanent, true),
			queryRecord("user-02", "bob", RoleMember, now, CredentialModeTemporary, false),
			queryRecord("user-01", "alice", RoleMember, now.Add(-time.Minute), CredentialModePermanent, true),
		},
	}
	query, codec := newUserQueryForTest(t, database, now, UserQueryConfig{DefaultLimit: 2, MaxLimit: 5})
	actor := administratorPrincipal("admin-01", false)

	page, err := query.List(context.Background(), UserQueryListRequest{
		Actor:        actor,
		Endpoint:     "/api/v1/admin/users",
		Category:     UserQueryCategoryUsers,
		FilterDigest: identityUserQueryFilterDigest,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(page.Items) != 2 || !page.HasMore || page.NextCursor == nil {
		t.Fatalf("List() page = %#v, want two items and next cursor", page)
	}
	if page.Items[0].ID != "user-03" || page.Items[1].ID != "user-02" {
		t.Fatalf("List() ids = %q,%q, want user-03,user-02", page.Items[0].ID, page.Items[1].ID)
	}
	if !strings.Contains(database.lastQuery, "status <> 'deleted'") ||
		!strings.Contains(database.lastQuery, "role = $1") ||
		!strings.Contains(database.lastQuery, "id <> $2") ||
		!strings.Contains(database.lastQuery, "ORDER BY u.created_at DESC, u.id DESC") {
		t.Fatalf("List() query does not enforce governance scope:\n%s", database.lastQuery)
	}
	if got, want := database.lastArgs[0], RoleMember; got != want {
		t.Fatalf("List() role arg = %v, want %v", got, want)
	}
	if got, want := database.lastArgs[1], actor.UserID; got != want {
		t.Fatalf("List() actor exclusion = %v, want %v", got, want)
	}
	if got, want := database.lastArgs[len(database.lastArgs)-1], 3; got != want {
		t.Fatalf("List() SQL limit = %v, want limit+1 %v", got, want)
	}

	position, err := codec.Decode(*page.NextCursor, IdentityCursorBinding{
		Endpoint:     "/api/v1/admin/users",
		Category:     UserQueryCategoryUsers,
		Visibility:   UserQueryVisibilityAdministrator,
		Limit:        2,
		FilterDigest: identityUserQueryFilterDigest,
	})
	if err != nil {
		t.Fatalf("Decode(next_cursor) error = %v", err)
	}
	if position.ID != "user-02" || !position.CreatedAt.Equal(now) {
		t.Fatalf("next cursor position = %#v, want user-02 at %v", position, now)
	}
}

func TestUserQueryLimitContract(t *testing.T) {
	now := time.Date(2026, 10, 23, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		limit      *int
		wantCode   platform.Code
		wantDBCall bool
		wantLimit  int
	}{
		{name: "omitted uses injected default", wantDBCall: true, wantLimit: 3},
		{name: "nonpositive rejected", limit: intPointer(0), wantCode: platform.CodeIdentityValidationFailed},
		{name: "negative rejected", limit: intPointer(-1), wantCode: platform.CodeIdentityValidationFailed},
		{name: "over maximum clipped", limit: intPointer(99), wantDBCall: true, wantLimit: 5},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database := &userQueryDatabaseStub{}
			query, _ := newUserQueryForTest(t, database, now, UserQueryConfig{DefaultLimit: 3, MaxLimit: 5})
			page, err := query.List(context.Background(), UserQueryListRequest{
				Actor: administratorPrincipal("admin-01", false), Endpoint: "/api/v1/admin/users",
				Category: UserQueryCategoryUsers, Limit: test.limit,
				FilterDigest: identityUserQueryFilterDigest,
			})
			if test.wantCode == "" {
				if err != nil {
					t.Fatalf("List() error = %v (page=%#v)", err, page)
				}
			} else if code := platform.PublicCode(err); code != test.wantCode {
				t.Fatalf("List() code = %q, want %q (page=%#v err=%v)", code, test.wantCode, page, err)
			}
			if got := database.queryCalls > 0; got != test.wantDBCall {
				t.Fatalf("database called = %t, want %t", got, test.wantDBCall)
			}
			if test.wantDBCall {
				if got := database.lastArgs[len(database.lastArgs)-1]; got != test.wantLimit+1 {
					t.Fatalf("SQL limit = %v, want %d", got, test.wantLimit+1)
				}
			}
		})
	}
}

func TestUserQueryAuthorizationMatrix(t *testing.T) {
	now := time.Date(2026, 10, 23, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		actor    session.Principal
		category UserQueryCategory
		wantCode platform.Code
		wantRole Role
	}{
		{
			name:  "ordinary administrator cannot enumerate administrators",
			actor: administratorPrincipal("admin-01", false), category: UserQueryCategoryAdministrators,
			wantCode: platform.CodeAuthForbidden,
		},
		{
			name:  "initial administrator can enumerate other administrators",
			actor: administratorPrincipal("initial-01", true), category: UserQueryCategoryAdministrators,
			wantRole: RoleAdmin,
		},
		{
			name:  "ordinary administrator can enumerate members",
			actor: administratorPrincipal("admin-01", false), category: UserQueryCategoryUsers,
			wantRole: RoleMember,
		},
		{
			name:     "member cannot enumerate users",
			actor:    session.Principal{UserID: "member-01", Role: string(RoleMember), Status: string(UserStatusActive)},
			category: UserQueryCategoryUsers, wantCode: platform.CodeAuthForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database := &userQueryDatabaseStub{}
			query, _ := newUserQueryForTest(t, database, now, UserQueryConfig{DefaultLimit: 2, MaxLimit: 5})
			_, err := query.List(context.Background(), UserQueryListRequest{
				Actor: test.actor, Endpoint: endpointForCategory(test.category),
				Category: test.category, FilterDigest: identityUserQueryFilterDigest,
			})
			if test.wantCode == "" {
				if err != nil {
					t.Fatalf("List() error = %v", err)
				}
			} else if code := platform.PublicCode(err); code != test.wantCode {
				t.Fatalf("List() code = %q, want %q (err=%v)", code, test.wantCode, err)
			}
			if test.wantCode != "" {
				if database.queryCalls != 0 {
					t.Fatalf("database calls = %d, want 0", database.queryCalls)
				}
				return
			}
			if got := database.lastArgs[0]; got != test.wantRole {
				t.Fatalf("role arg = %v, want %v", got, test.wantRole)
			}
			if got := database.lastArgs[1]; got != test.actor.UserID {
				t.Fatalf("actor exclusion = %v, want %v", got, test.actor.UserID)
			}
		})
	}
}

func TestUserQueryCursorRejectsMismatchedContinuationBeforeDatabase(t *testing.T) {
	now := time.Date(2026, 10, 23, 10, 0, 0, 0, time.UTC)
	database := &userQueryDatabaseStub{}
	query, codec := newUserQueryForTest(t, database, now, UserQueryConfig{DefaultLimit: 2, MaxLimit: 5})
	cursor, err := codec.Encode(
		IdentityCursorPosition{CreatedAt: now, ID: "user-02"},
		IdentityCursorBinding{
			Endpoint: "/api/v1/admin/users", Category: UserQueryCategoryUsers,
			Visibility: UserQueryVisibilityAdministrator, Limit: 2,
			FilterDigest: identityUserQueryFilterDigest,
		},
	)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	limit := 3
	_, err = query.List(context.Background(), UserQueryListRequest{
		Actor: administratorPrincipal("admin-01", false), Endpoint: "/api/v1/admin/users",
		Category: UserQueryCategoryUsers, Cursor: cursor, Limit: &limit,
		FilterDigest: identityUserQueryFilterDigest,
	})
	if code := platform.PublicCode(err); code != platform.CodeRequestInvalidCursor {
		t.Fatalf("List() code = %q, want %q", code, platform.CodeRequestInvalidCursor)
	}
	if database.queryCalls != 0 {
		t.Fatalf("database calls = %d, want 0", database.queryCalls)
	}
}

func TestUserQueryEmptyAndTerminalPagesUseExplicitNullCursor(t *testing.T) {
	now := time.Date(2026, 10, 23, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		records []userQueryRecord
	}{
		{name: "empty"},
		{name: "terminal", records: []userQueryRecord{
			queryRecord("user-01", "alice", RoleMember, now, CredentialModePermanent, true),
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database := &userQueryDatabaseStub{listRecords: test.records}
			query, _ := newUserQueryForTest(t, database, now, UserQueryConfig{DefaultLimit: 2, MaxLimit: 5})
			page, err := query.List(context.Background(), UserQueryListRequest{
				Actor: administratorPrincipal("admin-01", false), Endpoint: "/api/v1/admin/users",
				Category: UserQueryCategoryUsers, FilterDigest: identityUserQueryFilterDigest,
			})
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}
			if page.NextCursor != nil || page.HasMore {
				t.Fatalf("List() page = %#v, want null cursor and has_more=false", page)
			}
			data, err := json.Marshal(page)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			if !strings.Contains(string(data), `"next_cursor":null`) ||
				!strings.Contains(string(data), `"has_more":false`) ||
				!strings.Contains(string(data), `"items":[`) {
				t.Fatalf("page JSON = %s, want explicit items/cursor/has_more", data)
			}
		})
	}
}

func TestUserQueryDetailAndSummaryFieldAllowlists(t *testing.T) {
	now := time.Date(2026, 10, 23, 10, 0, 0, 0, time.UTC)
	email := "alice@example.com"
	record := queryRecord(
		"00000000-0000-0000-0000-000000000101",
		"alice",
		RoleMember,
		now,
		CredentialModePermanent,
		true,
	)
	record.Email = &email
	record.Phone = nil
	record.UpdatedAt = now.Add(time.Minute)
	database := &userQueryDatabaseStub{detailRecord: &record}
	query, _ := newUserQueryForTest(t, database, now, UserQueryConfig{DefaultLimit: 2, MaxLimit: 5})

	detail, err := query.Detail(context.Background(), UserQueryDetailRequest{
		Actor:    administratorPrincipal("admin-01", false),
		Category: UserQueryCategoryUsers,
		ID:       record.ID,
	})
	if err != nil {
		t.Fatalf("Detail() error = %v", err)
	}
	assertExactJSONKeys(t, detail, []string{
		"id", "username", "role", "status", "is_initial_admin",
		"must_change_password", "must_acknowledge_privacy", "created_at",
		"email", "phone", "updated_at",
	})
	data, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("json.Marshal(detail) error = %v", err)
	}
	if !strings.Contains(string(data), `"phone":null`) {
		t.Fatalf("detail JSON = %s, want explicit null phone", data)
	}
	for _, denied := range []string{
		"password_digest", "permanent_password", "temporary_credential",
		"session", "cookie", "csrf", "retained_email", "audit", "metadata",
	} {
		if strings.Contains(strings.ToLower(string(data)), denied) {
			t.Fatalf("detail JSON contains denied token %q: %s", denied, data)
		}
	}

	assertExactJSONKeys(t, detail.UserGovernanceSummary, []string{
		"id", "username", "role", "status", "is_initial_admin",
		"must_change_password", "must_acknowledge_privacy", "created_at",
	})
}

func TestUserQueryDetailDoesNotDiscloseIneligibleTargets(t *testing.T) {
	now := time.Date(2026, 10, 23, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		actor    session.Principal
		category UserQueryCategory
		id       string
		wantCode platform.Code
		wantCall bool
	}{
		{
			name:     "ordinary administrator administrator detail forbidden",
			actor:    administratorPrincipal("admin-01", false),
			category: UserQueryCategoryAdministrators,
			id:       "00000000-0000-0000-0000-000000000202",
			wantCode: platform.CodeAuthForbidden,
		},
		{
			name:     "initial administrator self excluded",
			actor:    administratorPrincipal("00000000-0000-0000-0000-000000000001", true),
			category: UserQueryCategoryAdministrators,
			id:       "00000000-0000-0000-0000-000000000001",
			wantCode: platform.CodeIdentityUserNotFound, wantCall: true,
		},
		{
			name:     "deleted or absent target is not found",
			actor:    administratorPrincipal("admin-01", false),
			category: UserQueryCategoryUsers,
			id:       "00000000-0000-0000-0000-000000000303",
			wantCode: platform.CodeIdentityUserNotFound, wantCall: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database := &userQueryDatabaseStub{detailErr: pgx.ErrNoRows}
			query, _ := newUserQueryForTest(t, database, now, UserQueryConfig{DefaultLimit: 2, MaxLimit: 5})
			_, err := query.Detail(context.Background(), UserQueryDetailRequest{
				Actor: test.actor, Category: test.category, ID: test.id,
			})
			if code := platform.PublicCode(err); code != test.wantCode {
				t.Fatalf("Detail() code = %q, want %q (err=%v)", code, test.wantCode, err)
			}
			if got := database.rowCalls > 0; got != test.wantCall {
				t.Fatalf("database called = %t, want %t", got, test.wantCall)
			}
			if test.wantCall {
				if !strings.Contains(database.lastQuery, "status <> 'deleted'") ||
					!strings.Contains(database.lastQuery, "id <> $3") {
					t.Fatalf("Detail() query does not exclude tombstone/self:\n%s", database.lastQuery)
				}
			}
		})
	}
}

func TestUserQueryDetailRejectsMalformedIDBeforeDatabase(t *testing.T) {
	now := time.Date(2026, 10, 23, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		actor    session.Principal
		category UserQueryCategory
	}{
		{
			name:     "users",
			actor:    administratorPrincipal("admin-01", false),
			category: UserQueryCategoryUsers,
		},
		{
			name:     "administrators",
			actor:    administratorPrincipal("initial-01", true),
			category: UserQueryCategoryAdministrators,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database := &userQueryDatabaseStub{}
			query, _ := newUserQueryForTest(
				t,
				database,
				now,
				UserQueryConfig{DefaultLimit: 2, MaxLimit: 5},
			)
			_, err := query.Detail(context.Background(), UserQueryDetailRequest{
				Actor: test.actor, Category: test.category, ID: "not-a-uuid",
			})
			if code := platform.PublicCode(err); code != platform.CodeIdentityUserNotFound {
				t.Fatalf(
					"Detail() code = %q, want %q (err=%v)",
					code,
					platform.CodeIdentityUserNotFound,
					err,
				)
			}
			if database.rowCalls != 0 {
				t.Fatalf("database calls = %d, want 0", database.rowCalls)
			}
		})
	}
}

func TestUserQueryHandlerParsesLimitAndWritesAllowlistedEnvelope(t *testing.T) {
	now := time.Date(2026, 10, 23, 10, 0, 0, 0, time.UTC)
	next := "next-token"
	service := &userQueryServiceStub{
		page: UserQueryPage{
			Items: []UserGovernanceSummary{{
				ID: "user-01", Username: "alice", Role: RoleMember, Status: UserStatusActive,
				IsInitialAdmin: false, MustChangePassword: false,
				MustAcknowledgePrivacy: false, CreatedAt: now.Format(time.RFC3339),
			}},
			NextCursor: &next,
			HasMore:    true,
		},
	}
	handler, err := NewUserQueryHandler(service)
	if err != nil {
		t.Fatalf("NewUserQueryHandler() error = %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users?limit=2&cursor=old", nil)
	response := httptest.NewRecorder()
	serveWithPrincipal(t, administratorPrincipal("admin-01", false), http.HandlerFunc(handler.ListUsers)).
		ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body)
	}
	if service.listRequest.Limit == nil || *service.listRequest.Limit != 2 ||
		service.listRequest.Cursor != "old" ||
		service.listRequest.Endpoint != "/api/v1/admin/users" ||
		service.listRequest.Category != UserQueryCategoryUsers {
		t.Fatalf("service request = %#v", service.listRequest)
	}
	assertExactJSONKeysFromBytes(t, response.Body.Bytes(), []string{"items", "next_cursor", "has_more"})
}

func TestUserQueryHandlerRejectsInvalidLimitWithoutServiceCall(t *testing.T) {
	tests := []string{"", "abc", "1.5"}
	for index, value := range tests {
		t.Run(strconv.Itoa(index), func(t *testing.T) {
			service := &userQueryServiceStub{}
			handler, err := NewUserQueryHandler(service)
			if err != nil {
				t.Fatalf("NewUserQueryHandler() error = %v", err)
			}
			request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users?limit="+value, nil)
			response := httptest.NewRecorder()
			serveWithPrincipal(t, administratorPrincipal("admin-01", false), http.HandlerFunc(handler.ListUsers)).
				ServeHTTP(response, request)
			if service.listCalls != 0 {
				t.Fatalf("service calls = %d, want 0", service.listCalls)
			}
			var envelope platform.Envelope
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("json.Unmarshal() error = %v; body=%s", err, response.Body)
			}
			if envelope.Code != platform.CodeIdentityValidationFailed {
				t.Fatalf("code = %q, want %q", envelope.Code, platform.CodeIdentityValidationFailed)
			}
		})
	}
}

func TestUserQueryHandlerCursorPresenceContract(t *testing.T) {
	targets := []struct {
		name     string
		endpoint string
		handler  func(*Handler) http.HandlerFunc
	}{
		{
			name:     "users",
			endpoint: "/api/v1/admin/users",
			handler:  func(handler *Handler) http.HandlerFunc { return handler.ListUsers },
		},
		{
			name:     "administrators",
			endpoint: "/api/v1/admin/administrators",
			handler:  func(handler *Handler) http.HandlerFunc { return handler.ListAdministrators },
		},
	}
	cases := []struct {
		name       string
		query      string
		wantStatus int
		wantCode   platform.Code
		wantCalls  int
		wantCursor string
	}{
		{
			name: "missing", wantStatus: http.StatusOK,
			wantCalls: 1, wantCursor: "",
		},
		{
			name: "empty", query: "?cursor=",
			wantStatus: http.StatusBadRequest,
			wantCode:   platform.CodeRequestInvalidCursor,
		},
		{
			name: "repeated", query: "?cursor=first&cursor=second",
			wantStatus: http.StatusBadRequest,
			wantCode:   platform.CodeRequestInvalidCursor,
		},
		{
			name: "valid", query: "?cursor=opaque",
			wantStatus: http.StatusOK,
			wantCalls:  1, wantCursor: "opaque",
		},
	}
	for _, target := range targets {
		for _, test := range cases {
			t.Run(target.name+"/"+test.name, func(t *testing.T) {
				service := &userQueryServiceStub{page: UserQueryPage{
					Items: make([]UserGovernanceSummary, 0),
				}}
				handler, err := NewUserQueryHandler(service)
				if err != nil {
					t.Fatalf("NewUserQueryHandler() error = %v", err)
				}
				request := httptest.NewRequest(
					http.MethodGet,
					target.endpoint+test.query,
					nil,
				)
				response := httptest.NewRecorder()
				serveWithPrincipal(
					t,
					administratorPrincipal("admin-01", target.name == "administrators"),
					target.handler(handler),
				).ServeHTTP(response, request)
				if response.Code != test.wantStatus {
					t.Fatalf(
						"status = %d, want %d; body=%s",
						response.Code,
						test.wantStatus,
						response.Body,
					)
				}
				if service.listCalls != test.wantCalls {
					t.Fatalf("service calls = %d, want %d", service.listCalls, test.wantCalls)
				}
				if test.wantCalls == 1 && service.listRequest.Cursor != test.wantCursor {
					t.Fatalf(
						"cursor = %q, want %q",
						service.listRequest.Cursor,
						test.wantCursor,
					)
				}
				if test.wantCode != "" {
					var envelope platform.Envelope
					if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
						t.Fatalf("json.Unmarshal() error = %v; body=%s", err, response.Body)
					}
					if envelope.Code != test.wantCode {
						t.Fatalf("code = %q, want %q", envelope.Code, test.wantCode)
					}
				}
			})
		}
	}
}

func newUserQueryForTest(
	t *testing.T,
	database UserQueryDatabase,
	now time.Time,
	config UserQueryConfig,
) (*UserQuery, *IdentityCursorCodec) {
	t.Helper()
	codec, err := NewIdentityCursorCodec(
		[]byte("0123456789abcdef0123456789abcdef"),
		time.Hour,
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatalf("NewIdentityCursorCodec() error = %v", err)
	}
	query, err := NewUserQuery(database, codec, config)
	if err != nil {
		t.Fatalf("NewUserQuery() error = %v", err)
	}
	return query, codec
}

func queryRecord(
	id string,
	username string,
	role Role,
	createdAt time.Time,
	mode CredentialMode,
	privacyComplete bool,
) userQueryRecord {
	var completedAt *time.Time
	if privacyComplete {
		value := createdAt
		completedAt = &value
	}
	return userQueryRecord{
		ID: id, Username: username, Role: role, Status: UserStatusActive,
		IsInitialAdmin: false,
		CreatedAt:      createdAt, UpdatedAt: createdAt,
		MustChangePassword:     mode != CredentialModePermanent,
		MustAcknowledgePrivacy: role == RoleAdmin && completedAt == nil,
	}
}

func administratorPrincipal(id string, initial bool) session.Principal {
	return session.Principal{
		UserID: id, Role: string(RoleAdmin), IsInitialAdmin: initial,
		Status: string(UserStatusActive),
	}
}

func endpointForCategory(category UserQueryCategory) string {
	if category == UserQueryCategoryAdministrators {
		return "/api/v1/admin/administrators"
	}
	return "/api/v1/admin/users"
}

func intPointer(value int) *int {
	return &value
}

func assertExactJSONKeys(t *testing.T, value any, want []string) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	assertExactJSONKeysFromBytes(t, data, want)
}

func assertExactJSONKeysFromBytes(t *testing.T, data []byte, want []string) {
	t.Helper()
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; data=%s", err, data)
	}
	got := make([]string, 0, len(object))
	for key := range object {
		got = append(got, key)
	}
	if !sameStringSet(got, want) {
		t.Fatalf("JSON keys = %v, want exactly %v; data=%s", got, want, data)
	}
}

func sameStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	counts := make(map[string]int, len(left))
	for _, value := range left {
		counts[value]++
	}
	for _, value := range right {
		counts[value]--
	}
	for _, count := range counts {
		if count != 0 {
			return false
		}
	}
	return true
}

type userQueryDatabaseStub struct {
	listRecords  []userQueryRecord
	listErr      error
	detailRecord *userQueryRecord
	detailErr    error
	queryCalls   int
	rowCalls     int
	lastQuery    string
	lastArgs     []any
}

func (s *userQueryDatabaseStub) Query(
	_ context.Context,
	query string,
	args ...any,
) (pgx.Rows, error) {
	s.queryCalls++
	s.lastQuery = query
	s.lastArgs = append([]any(nil), args...)
	if s.listErr != nil {
		return nil, s.listErr
	}
	return &userQueryRowsStub{records: append([]userQueryRecord(nil), s.listRecords...)}, nil
}

func (s *userQueryDatabaseStub) QueryRow(
	_ context.Context,
	query string,
	args ...any,
) pgx.Row {
	s.rowCalls++
	s.lastQuery = query
	s.lastArgs = append([]any(nil), args...)
	return userQueryRowStub{record: s.detailRecord, err: s.detailErr}
}

type userQueryRowsStub struct {
	records []userQueryRecord
	index   int
	closed  bool
	err     error
}

func (s *userQueryRowsStub) Close() {
	s.closed = true
}

func (s *userQueryRowsStub) Next() bool {
	if s.index >= len(s.records) {
		s.closed = true
		return false
	}
	s.index++
	return true
}

func (s *userQueryRowsStub) Scan(dest ...any) error {
	if s.index == 0 || s.index > len(s.records) {
		return errors.New("Scan called without current row")
	}
	return scanUserQueryRecordForTest(s.records[s.index-1], dest...)
}

func (s *userQueryRowsStub) Err() error {
	return s.err
}

func (s *userQueryRowsStub) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (s *userQueryRowsStub) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (s *userQueryRowsStub) Values() ([]any, error) {
	return nil, errors.New("Values is not supported by user query test rows")
}

func (s *userQueryRowsStub) RawValues() [][]byte {
	return nil
}

func (s *userQueryRowsStub) Conn() *pgx.Conn {
	return nil
}

func (s *userQueryRowsStub) TypeMap() *pgtype.Map {
	return pgtype.NewMap()
}

type userQueryRowStub struct {
	record *userQueryRecord
	err    error
}

func (s userQueryRowStub) Scan(dest ...any) error {
	if s.err != nil {
		return s.err
	}
	if s.record == nil {
		return pgx.ErrNoRows
	}
	return scanUserQueryRecordForTest(*s.record, dest...)
}

func scanUserQueryRecordForTest(record userQueryRecord, dest ...any) error {
	values := []any{
		record.ID, record.Username, record.Role, record.Status, record.IsInitialAdmin,
		record.MustChangePassword, record.MustAcknowledgePrivacy, record.CreatedAt,
		record.Email, record.Phone, record.UpdatedAt,
	}
	if len(dest) != len(values) {
		return errors.New("unexpected scan destination count")
	}
	for index := range dest {
		target := reflect.ValueOf(dest[index])
		if target.Kind() != reflect.Pointer || target.IsNil() {
			return errors.New("scan destination is not a pointer")
		}
		value := reflect.ValueOf(values[index])
		if values[index] == nil {
			target.Elem().Set(reflect.Zero(target.Elem().Type()))
			continue
		}
		if value.Type().AssignableTo(target.Elem().Type()) {
			target.Elem().Set(value)
			continue
		}
		return errors.New("scan destination type mismatch")
	}
	return nil
}

type userQueryServiceStub struct {
	page          UserQueryPage
	detail        UserGovernanceDetail
	err           error
	listCalls     int
	detailCalls   int
	listRequest   UserQueryListRequest
	detailRequest UserQueryDetailRequest
}

func (s *userQueryServiceStub) List(
	_ context.Context,
	request UserQueryListRequest,
) (UserQueryPage, error) {
	s.listCalls++
	s.listRequest = request
	return s.page, s.err
}

func (s *userQueryServiceStub) Detail(
	_ context.Context,
	request UserQueryDetailRequest,
) (UserGovernanceDetail, error) {
	s.detailCalls++
	s.detailRequest = request
	return s.detail, s.err
}

type principalResolverStub struct {
	principal session.Principal
}

func (s principalResolverStub) ResolvePrincipal(
	context.Context,
	session.ResolveInput,
) (session.Principal, error) {
	return s.principal, nil
}

func serveWithPrincipal(
	t *testing.T,
	principal session.Principal,
	next http.Handler,
) http.Handler {
	t.Helper()
	authorizer, err := platform.NewHTTPAuthorizer(
		principalResolverStub{principal: principal},
		func() time.Time { return time.Date(2026, 10, 23, 10, 0, 0, 0, time.UTC) },
	)
	if err != nil {
		t.Fatalf("NewHTTPAuthorizer() error = %v", err)
	}
	protected := authorizer.Protect(session.AccessGeneral, next)
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: "session-token"})
		protected.ServeHTTP(response, request)
	})
}
