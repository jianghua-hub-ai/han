package identity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"han/internal/platform"
	"han/internal/session"
)

// UserQueryCategory is one governance target class.
type UserQueryCategory string

const (
	// UserQueryCategoryUsers selects ordinary members.
	UserQueryCategoryUsers UserQueryCategory = "users"
	// UserQueryCategoryAdministrators selects non-initial administrators.
	UserQueryCategoryAdministrators UserQueryCategory = "administrators"
)

// UserQueryVisibility binds cursors to the actor's governance scope.
type UserQueryVisibility string

const (
	// UserQueryVisibilityAdministrator is the ordinary administrator scope.
	UserQueryVisibilityAdministrator UserQueryVisibility = "administrator"
	// UserQueryVisibilityInitialAdministrator is the initial administrator scope.
	UserQueryVisibilityInitialAdministrator UserQueryVisibility = "initial_administrator"
)

var identityUserQueryFilterDigest = identityEmptyFilterDigest()

// UserQueryConfig supplies deployment-owned pagination bounds.
type UserQueryConfig struct {
	DefaultLimit int
	MaxLimit     int
}

// UserQueryListRequest contains one authorized list request.
type UserQueryListRequest struct {
	Actor        session.Principal
	Endpoint     string
	Category     UserQueryCategory
	Cursor       string
	Limit        *int
	FilterDigest string
}

// UserQueryDetailRequest contains one authorized detail request.
type UserQueryDetailRequest struct {
	Actor    session.Principal
	Category UserQueryCategory
	ID       string
}

// UserGovernanceSummary is the exact minimum list view.
type UserGovernanceSummary struct {
	ID                     string     `json:"id"`
	Username               string     `json:"username"`
	Role                   Role       `json:"role"`
	Status                 UserStatus `json:"status"`
	IsInitialAdmin         bool       `json:"is_initial_admin"`
	MustChangePassword     bool       `json:"must_change_password"`
	MustAcknowledgePrivacy bool       `json:"must_acknowledge_privacy"`
	CreatedAt              string     `json:"created_at"`
}

// UserGovernanceDetail adds only current contact fields and update time.
type UserGovernanceDetail struct {
	UserGovernanceSummary
	Email     *string `json:"email"`
	Phone     *string `json:"phone"`
	UpdatedAt string  `json:"updated_at"`
}

// UserQueryPage is the fixed governance list envelope.
type UserQueryPage struct {
	Items      []UserGovernanceSummary `json:"items"`
	NextCursor *string                 `json:"next_cursor"`
	HasMore    bool                    `json:"has_more"`
}

// UserQueryDatabase is the read-only PostgreSQL capability consumed by UserQuery.
type UserQueryDatabase interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

var _ UserQueryDatabase = (*pgxpool.Pool)(nil)

// UserQuery serves minimum-disclosure governance reads.
type UserQuery struct {
	database UserQueryDatabase
	cursor   *IdentityCursorCodec
	config   UserQueryConfig
}

type userQueryRecord struct {
	ID                     string
	Username               string
	Role                   Role
	Status                 UserStatus
	IsInitialAdmin         bool
	MustChangePassword     bool
	MustAcknowledgePrivacy bool
	CreatedAt              time.Time
	Email                  *string
	Phone                  *string
	UpdatedAt              time.Time
}

// NewUserQuery creates a query service with explicit pagination policy.
func NewUserQuery(
	database UserQueryDatabase,
	cursor *IdentityCursorCodec,
	config UserQueryConfig,
) (*UserQuery, error) {
	if database == nil {
		return nil, errors.New("identity user query database is required")
	}
	if cursor == nil {
		return nil, errors.New("identity cursor codec is required")
	}
	if config.DefaultLimit <= 0 || config.MaxLimit <= 0 ||
		config.DefaultLimit > config.MaxLimit {
		return nil, errors.New("identity user query limits are invalid")
	}
	return &UserQuery{database: database, cursor: cursor, config: config}, nil
}

// List returns one stable keyset page within the actor's governance scope.
func (q *UserQuery) List(
	ctx context.Context,
	request UserQueryListRequest,
) (UserQueryPage, error) {
	role, visibility, err := authorizeUserQuery(request.Actor, request.Category)
	if err != nil {
		return UserQueryPage{}, err
	}
	if request.Endpoint != endpointForUserQueryCategory(request.Category) {
		return UserQueryPage{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("identity user query endpoint is invalid"),
		)
	}
	filterDigest := request.FilterDigest
	if filterDigest == "" {
		filterDigest = identityUserQueryFilterDigest
	}
	limit, err := q.resolveLimit(request.Limit)
	if err != nil {
		return UserQueryPage{}, err
	}
	binding := IdentityCursorBinding{
		Endpoint: request.Endpoint, Category: request.Category,
		Visibility: visibility, Limit: limit, FilterDigest: filterDigest,
	}
	var cursorCreatedAt any
	var cursorID any
	if request.Cursor != "" {
		position, err := q.cursor.Decode(request.Cursor, binding)
		if err != nil {
			return UserQueryPage{}, err
		}
		cursorCreatedAt = position.CreatedAt
		cursorID = position.ID
	}

	rows, err := q.database.Query(
		ctx,
		listUserGovernanceSQL,
		role,
		request.Actor.UserID,
		cursorCreatedAt,
		cursorID,
		limit+1,
	)
	if err != nil {
		return UserQueryPage{}, queryDatabaseError("list identity users", err)
	}
	defer rows.Close()

	records := make([]userQueryRecord, 0, limit+1)
	for rows.Next() {
		record, err := scanUserQueryRecord(rows)
		if err != nil {
			return UserQueryPage{}, queryDatabaseError("scan identity user list", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return UserQueryPage{}, queryDatabaseError("read identity user list", err)
	}

	hasMore := len(records) > limit
	if hasMore {
		records = records[:limit]
	}
	page := UserQueryPage{
		Items:   make([]UserGovernanceSummary, 0, len(records)),
		HasMore: hasMore,
	}
	for _, record := range records {
		page.Items = append(page.Items, record.summary())
	}
	if hasMore {
		last := records[len(records)-1]
		nextCursor, err := q.cursor.Encode(
			IdentityCursorPosition{CreatedAt: last.CreatedAt, ID: last.ID},
			binding,
		)
		if err != nil {
			return UserQueryPage{}, platform.NewPublicError(
				platform.CodePlatformInternalError,
				fmt.Errorf("encode identity user query cursor: %w", err),
			)
		}
		page.NextCursor = &nextCursor
	}
	return page, nil
}

// Detail returns one minimum-disclosure target within the actor's scope.
func (q *UserQuery) Detail(
	ctx context.Context,
	request UserQueryDetailRequest,
) (UserGovernanceDetail, error) {
	role, _, err := authorizeUserQuery(request.Actor, request.Category)
	if err != nil {
		return UserGovernanceDetail{}, err
	}
	if strings.TrimSpace(request.ID) == "" {
		return UserGovernanceDetail{}, userNotFoundError()
	}
	var targetID pgtype.UUID
	if err := targetID.Scan(request.ID); err != nil || !targetID.Valid {
		return UserGovernanceDetail{}, userNotFoundError()
	}
	record, err := scanUserQueryRecord(q.database.QueryRow(
		ctx,
		detailUserGovernanceSQL,
		targetID,
		role,
		request.Actor.UserID,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return UserGovernanceDetail{}, userNotFoundError()
	}
	if err != nil {
		return UserGovernanceDetail{}, queryDatabaseError("get identity user detail", err)
	}
	return UserGovernanceDetail{
		UserGovernanceSummary: record.summary(),
		Email:                 cloneOptionalString(record.Email),
		Phone:                 cloneOptionalString(record.Phone),
		UpdatedAt:             record.UpdatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (q *UserQuery) resolveLimit(requested *int) (int, error) {
	if requested == nil {
		return q.config.DefaultLimit, nil
	}
	if *requested <= 0 {
		return 0, platform.NewPublicError(
			platform.CodeIdentityValidationFailed,
			errors.New("identity user query limit must be positive"),
			platform.FieldError{Field: "limit", Code: "must_be_positive"},
		)
	}
	if *requested > q.config.MaxLimit {
		return q.config.MaxLimit, nil
	}
	return *requested, nil
}

func authorizeUserQuery(
	actor session.Principal,
	category UserQueryCategory,
) (Role, UserQueryVisibility, error) {
	if actor.UserID == "" ||
		actor.Role != string(RoleAdmin) ||
		actor.Status != string(UserStatusActive) ||
		actor.Restricted() ||
		!category.valid() {
		return "", "", forbiddenUserQueryError()
	}
	visibility := UserQueryVisibilityAdministrator
	if actor.IsInitialAdmin {
		visibility = UserQueryVisibilityInitialAdministrator
	}
	switch category {
	case UserQueryCategoryUsers:
		return RoleMember, visibility, nil
	case UserQueryCategoryAdministrators:
		if !actor.IsInitialAdmin {
			return "", "", forbiddenUserQueryError()
		}
		return RoleAdmin, visibility, nil
	default:
		return "", "", forbiddenUserQueryError()
	}
}

func scanUserQueryRecord(row interface{ Scan(...any) error }) (userQueryRecord, error) {
	var record userQueryRecord
	err := row.Scan(
		&record.ID,
		&record.Username,
		&record.Role,
		&record.Status,
		&record.IsInitialAdmin,
		&record.MustChangePassword,
		&record.MustAcknowledgePrivacy,
		&record.CreatedAt,
		&record.Email,
		&record.Phone,
		&record.UpdatedAt,
	)
	return record, err
}

func (record userQueryRecord) summary() UserGovernanceSummary {
	return UserGovernanceSummary{
		ID: record.ID, Username: record.Username, Role: record.Role, Status: record.Status,
		IsInitialAdmin:         record.IsInitialAdmin,
		MustChangePassword:     record.MustChangePassword,
		MustAcknowledgePrivacy: record.MustAcknowledgePrivacy,
		CreatedAt:              record.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (category UserQueryCategory) valid() bool {
	return category == UserQueryCategoryUsers ||
		category == UserQueryCategoryAdministrators
}

func (visibility UserQueryVisibility) valid() bool {
	return visibility == UserQueryVisibilityAdministrator ||
		visibility == UserQueryVisibilityInitialAdministrator
}

func endpointForUserQueryCategory(category UserQueryCategory) string {
	if category == UserQueryCategoryAdministrators {
		return "/api/v1/admin/administrators"
	}
	return "/api/v1/admin/users"
}

func identityEmptyFilterDigest() string {
	digest := sha256.Sum256([]byte("identity-user-query-filter/v1:none"))
	return "sha256:" + hex.EncodeToString(digest[:])
}

func queryDatabaseError(operation string, cause error) error {
	return platform.NewPublicError(
		platform.CodePlatformDatabaseUnavailable,
		fmt.Errorf("%s: %w", operation, cause),
	)
}

func forbiddenUserQueryError() error {
	return platform.NewPublicError(
		platform.CodeAuthForbidden,
		errors.New("identity user query is forbidden"),
	)
}

func userNotFoundError() error {
	return platform.NewPublicError(
		platform.CodeIdentityUserNotFound,
		errors.New("identity user target is unavailable"),
	)
}

func cloneOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

const userGovernanceColumnsSQL = `
	u.id::text,
	u.username,
	u.role,
	u.status,
	u.is_initial_admin,
	u.credential_mode <> 'permanent' AS must_change_password,
	u.role = 'admin'
		AND u.onboarding_completed_at IS NULL
		AND (
			current_notice.version IS NULL
			OR u.onboarding_privacy_acknowledged_version
				IS DISTINCT FROM current_notice.version
		) AS must_acknowledge_privacy,
	u.created_at,
	u.email,
	u.phone,
	u.updated_at
`

const listUserGovernanceSQL = `
	SELECT
` + userGovernanceColumnsSQL + `
	FROM users AS u
	LEFT JOIN LATERAL (
		SELECT version
		FROM privacy_notices
		WHERE status = 'current'
		LIMIT 1
	) AS current_notice ON true
	WHERE u.status <> 'deleted'
		AND u.role = $1
		AND u.id <> $2
		AND (
			$3::timestamptz IS NULL
			OR (u.created_at, u.id) < ($3::timestamptz, $4::uuid)
		)
	ORDER BY u.created_at DESC, u.id DESC
	LIMIT $5
`

const detailUserGovernanceSQL = `
	SELECT
` + userGovernanceColumnsSQL + `
	FROM users AS u
	LEFT JOIN LATERAL (
		SELECT version
		FROM privacy_notices
		WHERE status = 'current'
		LIMIT 1
	) AS current_notice ON true
	WHERE u.id = $1
		AND u.role = $2
		AND u.id <> $3
		AND u.status <> 'deleted'
`
