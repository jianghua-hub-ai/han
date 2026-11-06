package article

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"han/internal/session"
)

const (
	cursorVersion          = 1
	maxEncodedCursorLength = 1024
	articleETagPrefix      = "article-"
	articleETagVersionMark = "-v"
	authorKindUser         = "user"
	authorKindAnonymous    = "anonymous"
	anonymousAuthorLabel   = "匿名"
	authorStatusActive     = "active"
	authorStatusSuspended  = "suspended"
	principalRoleMember    = "member"
	principalRoleAdmin     = "admin"
	principalStatusActive  = "active"
)

var (
	// ErrArticleUnavailable intentionally combines absent and invisible resources.
	ErrArticleUnavailable = errors.New("article unavailable")
	// ErrArticleInvalidState marks an impossible authoritative state combination.
	ErrArticleInvalidState = errors.New("article state is invalid")
	// ErrArticleForbidden marks a trusted principal without the required role.
	ErrArticleForbidden = errors.New("article operation forbidden")
	// ErrInvalidCursor marks an invalid or view-mismatched list cursor.
	ErrInvalidCursor = errors.New("article cursor is invalid")
	// ErrInvalidPrecondition marks malformed or unsupported If-Match syntax.
	ErrInvalidPrecondition = errors.New("article precondition is invalid")
	// ErrPreconditionRequired marks a missing If-Match header.
	ErrPreconditionRequired = errors.New("article precondition is required")
	// ErrPreconditionFailed marks a valid but stale strong ETag.
	ErrPreconditionFailed = errors.New("article precondition failed")
	// ErrArticleValidationFailed marks invalid public article fields.
	ErrArticleValidationFailed = errors.New("article validation failed")
	// ErrTransferTargetInvalid marks an ineligible transfer target.
	ErrTransferTargetInvalid = errors.New("article transfer target is invalid")
	// ErrDeletionInProgress marks an irreversible deletion still converging.
	ErrDeletionInProgress = errors.New("article deletion is in progress")
)

// VisibilityPolicy derives consumer visibility without persisting it.
type VisibilityPolicy struct{}

func moderationAllowed(article Article, expectedVersion int64, category *ViolationCategory) error {
	if article.DeletionState != DeletionActive {
		return ErrArticleUnavailable
	}
	if article.Version != expectedVersion {
		return ErrPreconditionFailed
	}
	if err := ValidateArticleState(article); err != nil {
		return err
	}
	if (category == nil) != (article.ViolationState == ViolationViolating) {
		return ErrArticleInvalidState
	}
	return nil
}

// ValidateArticleState validates authoritative orthogonal state.
func ValidateArticleState(article Article) error {
	if article.PublicationState != PublicationPublished &&
		article.PublicationState != PublicationHeld {
		return ErrArticleInvalidState
	}
	orphanHeld := article.AuthorID == nil && article.OrphanedFromUserID != nil
	if (article.PublicationState == PublicationHeld) != orphanHeld {
		return ErrArticleInvalidState
	}
	if article.AuthorID != nil && article.OrphanedFromUserID != nil {
		return ErrArticleInvalidState
	}

	if article.ViolationState != ViolationClear &&
		article.ViolationState != ViolationViolating {
		return ErrArticleInvalidState
	}
	hasViolationCategory := article.ViolationCategory != nil
	if (article.ViolationState == ViolationViolating) != hasViolationCategory {
		return ErrArticleInvalidState
	}
	if hasViolationCategory &&
		*article.ViolationCategory != ViolationViolence &&
		*article.ViolationCategory != ViolationSexual {
		return ErrArticleInvalidState
	}

	if article.DeletionState != DeletionActive &&
		article.DeletionState != DeletionPending {
		return ErrArticleInvalidState
	}
	hasDeletionOperation := article.DeletionOperationID != nil
	if (article.DeletionState == DeletionPending) != hasDeletionOperation {
		return ErrArticleInvalidState
	}
	if article.Version < 1 {
		return ErrArticleInvalidState
	}
	return nil
}

// MemberVisible reports whether a regular member may receive the article.
func (VisibilityPolicy) MemberVisible(article Article, authorStatus string) bool {
	if ValidateArticleState(article) != nil ||
		article.DeletionState != DeletionActive ||
		article.PublicationState != PublicationPublished ||
		article.ViolationState != ViolationClear {
		return false
	}
	return article.AuthorID == nil || authorStatus == authorStatusActive
}

// MemberListProjection creates the body-free regular-member list projection.
func (policy VisibilityPolicy) MemberListProjection(
	article Article,
	authorStatus string,
) (MemberArticleSummaryView, error) {
	if !policy.MemberVisible(article, authorStatus) {
		return MemberArticleSummaryView{}, ErrArticleUnavailable
	}
	return MemberArticleSummaryView{
		ID:        article.ID,
		Title:     article.Title,
		Author:    publicAuthor(article.AuthorID),
		CreatedAt: article.CreatedAt,
		UpdatedAt: article.UpdatedAt,
	}, nil
}

// AuthorListProjection creates the body-free summary of one Article the
// current author owns. Unlike MemberListProjection it keeps Articles that are
// member-invisible (violating or held) so the author can still manage them;
// it never exposes another user's Article and never exposes a deleted one.
func (policy VisibilityPolicy) AuthorListProjection(
	article Article,
	principal session.Principal,
) (MemberArticleSummaryView, error) {
	if !principal.Allows(session.AccessGeneral) ||
		ValidateArticleState(article) != nil ||
		article.DeletionState != DeletionActive ||
		article.AuthorID == nil ||
		*article.AuthorID != principal.UserID {
		return MemberArticleSummaryView{}, ErrArticleUnavailable
	}
	return MemberArticleSummaryView{
		ID:        article.ID,
		Title:     article.Title,
		Author:    publicAuthor(article.AuthorID),
		CreatedAt: article.CreatedAt,
		UpdatedAt: article.UpdatedAt,
	}, nil
}

// MemberDetailProjection creates the regular-member detail projection.
func (policy VisibilityPolicy) MemberDetailProjection(
	article Article,
	authorStatus string,
) (MemberArticleDetailView, error) {
	if !policy.MemberVisible(article, authorStatus) {
		return MemberArticleDetailView{}, ErrArticleUnavailable
	}
	return MemberArticleDetailView{
		ID:               article.ID,
		Title:            article.Title,
		Body:             article.Body,
		Author:           publicAuthor(article.AuthorID),
		PublicationState: article.PublicationState,
		ViolationState:   article.ViolationState,
		CreatedAt:        article.CreatedAt,
		UpdatedAt:        article.UpdatedAt,
	}, nil
}

// AuthorProjection creates the controlled projection for the current author.
func (VisibilityPolicy) AuthorProjection(
	article Article,
	principal session.Principal,
) (AuthorArticleView, error) {
	if !principal.Allows(session.AccessGeneral) ||
		ValidateArticleState(article) != nil ||
		article.DeletionState != DeletionActive ||
		article.AuthorID == nil ||
		*article.AuthorID != principal.UserID {
		return AuthorArticleView{}, ErrArticleUnavailable
	}

	moderationStatus := ModerationNormal
	if article.ViolationState == ViolationViolating {
		moderationStatus = ModerationRestricted
	}
	return AuthorArticleView{
		Title:            article.Title,
		Body:             article.Body,
		ModerationStatus: moderationStatus,
		Editable:         true,
	}, nil
}

// AdminProjection creates the complete governance projection.
func (VisibilityPolicy) AdminProjection(
	article Article,
	principal session.Principal,
	authorStatus string,
) (AdminArticleView, error) {
	if !principal.Allows(session.AccessGeneral) || principal.Role != principalRoleAdmin {
		return AdminArticleView{}, ErrArticleForbidden
	}
	if ValidateArticleState(article) != nil || article.DeletionState != DeletionActive {
		return AdminArticleView{}, ErrArticleUnavailable
	}
	return AdminArticleView{
		ID:                 article.ID,
		Title:              article.Title,
		Body:               article.Body,
		Author:             publicAuthor(article.AuthorID),
		PublicationState:   article.PublicationState,
		ViolationState:     article.ViolationState,
		ViolationCategory:  article.ViolationCategory,
		HiddenReasons:      hiddenReasons(article, authorStatus),
		OrphanedFromUserID: article.OrphanedFromUserID,
		Version:            article.Version,
		CreatedAt:          article.CreatedAt,
		UpdatedAt:          article.UpdatedAt,
	}, nil
}

// CursorCodec encodes and decodes versioned, view-bound article cursors.
type CursorCodec struct{}

// Encode returns the stable Base64URL cursor representation.
func (CursorCodec) Encode(cursor ArticleCursor) (string, error) {
	if !validCursorView(cursor.View) ||
		cursor.CreatedAt.IsZero() ||
		cursor.CreatedAt.Location() != time.UTC ||
		!validUUID(cursor.ID) {
		return "", ErrInvalidCursor
	}
	payload, err := json.Marshal(cursorPayload{
		Version:   cursorVersion,
		CreatedAt: cursor.CreatedAt.Format(time.RFC3339Nano),
		ID:        cursor.ID,
		View:      cursor.View,
	})
	if err != nil {
		return "", fmt.Errorf("%w: encode payload: %v", ErrInvalidCursor, err)
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

// Decode validates a cursor against the requested list view.
func (CursorCodec) Decode(encoded string, expectedView CursorView) (ArticleCursor, error) {
	if encoded == "" ||
		len(encoded) > maxEncodedCursorLength ||
		strings.Contains(encoded, "=") ||
		!validCursorView(expectedView) {
		return ArticleCursor{}, ErrInvalidCursor
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || base64.RawURLEncoding.EncodeToString(raw) != encoded {
		return ArticleCursor{}, ErrInvalidCursor
	}
	payload, err := decodeCursorPayload(raw)
	if err != nil ||
		payload.Version != cursorVersion ||
		payload.View != expectedView ||
		!validCursorView(payload.View) ||
		!validUUID(payload.ID) {
		return ArticleCursor{}, ErrInvalidCursor
	}
	createdAt, err := time.Parse(time.RFC3339Nano, payload.CreatedAt)
	if err != nil ||
		createdAt.IsZero() ||
		createdAt.Location() != time.UTC ||
		createdAt.Format(time.RFC3339Nano) != payload.CreatedAt {
		return ArticleCursor{}, ErrInvalidCursor
	}
	return ArticleCursor{
		CreatedAt: createdAt,
		ID:        payload.ID,
		View:      payload.View,
	}, nil
}

// StrongETag returns the current strong article entity tag.
func StrongETag(article Article) (string, error) {
	if ValidateArticleState(article) != nil || !validUUID(article.ID) {
		return "", ErrArticleInvalidState
	}
	return fmt.Sprintf(
		`"%s%s%s%d"`,
		articleETagPrefix,
		article.ID,
		articleETagVersionMark,
		article.Version,
	), nil
}

// RequireMatchingETag validates one exact strong If-Match value.
func RequireMatchingETag(ifMatch string, article Article) error {
	if ifMatch == "" {
		return ErrPreconditionRequired
	}
	if _, _, err := parseStrongETag(ifMatch); err != nil {
		return err
	}
	current, err := StrongETag(article)
	if err != nil {
		return err
	}
	if ifMatch != current {
		return ErrPreconditionFailed
	}
	return nil
}

type cursorPayload struct {
	Version   int        `json:"v"`
	CreatedAt string     `json:"created_at"`
	ID        string     `json:"id"`
	View      CursorView `json:"view"`
}

func decodeCursorPayload(raw []byte) (cursorPayload, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return cursorPayload{}, ErrInvalidCursor
	}

	var payload cursorPayload
	seen := make(map[string]struct{}, 4)
	for decoder.More() {
		token, err = decoder.Token()
		if err != nil {
			return cursorPayload{}, ErrInvalidCursor
		}
		field, ok := token.(string)
		if !ok {
			return cursorPayload{}, ErrInvalidCursor
		}
		if _, duplicate := seen[field]; duplicate {
			return cursorPayload{}, ErrInvalidCursor
		}
		seen[field] = struct{}{}

		switch field {
		case "v":
			err = decoder.Decode(&payload.Version)
		case "created_at":
			err = decoder.Decode(&payload.CreatedAt)
		case "id":
			err = decoder.Decode(&payload.ID)
		case "view":
			err = decoder.Decode(&payload.View)
		default:
			return cursorPayload{}, ErrInvalidCursor
		}
		if err != nil {
			return cursorPayload{}, ErrInvalidCursor
		}
	}
	token, err = decoder.Token()
	if err != nil || token != json.Delim('}') || len(seen) != 4 {
		return cursorPayload{}, ErrInvalidCursor
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return cursorPayload{}, ErrInvalidCursor
	}
	return payload, nil
}

func publicAuthor(authorID *string) ArticleAuthorView {
	if authorID == nil {
		return ArticleAuthorView{
			Kind:  authorKindAnonymous,
			Label: anonymousAuthorLabel,
		}
	}
	return ArticleAuthorView{
		Kind: authorKindUser,
		ID:   authorID,
	}
}

func hiddenReasons(article Article, authorStatus string) []HiddenReason {
	reasons := make([]HiddenReason, 0, 3)
	if article.AuthorID != nil && authorStatus != authorStatusActive {
		reasons = append(reasons, HiddenAuthorSuspended)
	}
	if article.ViolationState == ViolationViolating {
		reasons = append(reasons, HiddenViolation)
	}
	if article.PublicationState == PublicationHeld {
		reasons = append(reasons, HiddenOrphanHold)
	}
	return reasons
}

func validCursorView(view CursorView) bool {
	switch view {
	case CursorViewMember, CursorViewAdminAll, CursorViewAdminViolating, CursorViewAdminHidden,
		CursorViewAdminOrphan:
		return true
	default:
		return false
	}
}

func parseStrongETag(value string) (string, int64, error) {
	if value == "" ||
		value != strings.TrimSpace(value) ||
		value == "*" ||
		strings.HasPrefix(value, "W/") ||
		strings.Contains(value, ",") ||
		len(value) < 2 ||
		value[0] != '"' ||
		value[len(value)-1] != '"' {
		return "", 0, ErrInvalidPrecondition
	}

	opaque := value[1 : len(value)-1]
	if !strings.HasPrefix(opaque, articleETagPrefix) {
		return "", 0, ErrInvalidPrecondition
	}
	opaque = strings.TrimPrefix(opaque, articleETagPrefix)
	if len(opaque) <= 36+len(articleETagVersionMark) {
		return "", 0, ErrInvalidPrecondition
	}
	id := opaque[:36]
	versionText := opaque[36:]
	if !validUUID(id) || !strings.HasPrefix(versionText, articleETagVersionMark) {
		return "", 0, ErrInvalidPrecondition
	}
	versionText = strings.TrimPrefix(versionText, articleETagVersionMark)
	version, err := strconv.ParseInt(versionText, 10, 64)
	if err != nil || version < 1 || strconv.FormatInt(version, 10) != versionText {
		return "", 0, ErrInvalidPrecondition
	}
	return id, version, nil
}

func validUUID(value string) bool {
	if len(value) != 36 ||
		value[8] != '-' ||
		value[13] != '-' ||
		value[18] != '-' ||
		value[23] != '-' {
		return false
	}
	for index, character := range value {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			continue
		}
		if !(character >= '0' && character <= '9') &&
			!(character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}
