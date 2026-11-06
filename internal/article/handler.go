package article

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"han/internal/platform"
	"han/internal/session"
)

const (
	// MaxArticleWriteBodyBytes is the actual wire-body limit for Article writes.
	MaxArticleWriteBodyBytes                = 262144
	articleLogModule                        = "article"
	createArticleEndpoint                   = "POST /api/v1/articles"
	createArticleResponseWriteFailureEvent  = "article.create.response_write_failed"
	createArticleResponseWriteFailureReason = "response_write"
	// CodeRequestInvalidPrecondition identifies malformed If-Match input.
	CodeRequestInvalidPrecondition platform.Code = "request.invalid_precondition"
	// CodeRequestPreconditionRequired identifies a missing If-Match header.
	CodeRequestPreconditionRequired platform.Code = "request.precondition_required"
	// CodeArticleUnavailable intentionally combines absent and invisible articles.
	CodeArticleUnavailable platform.Code = "article.unavailable"
	// CodeArticleInvalidState identifies a forbidden article state transition.
	CodeArticleInvalidState platform.Code = "article.invalid_state"
	// CodeArticlePreconditionFailed identifies a stale article version.
	CodeArticlePreconditionFailed platform.Code = "article.precondition_failed"
	// CodeArticleValidationFailed identifies invalid title or body input.
	CodeArticleValidationFailed platform.Code = "article.validation_failed"
	// CodeArticleTransferTargetInvalid identifies an ineligible new author.
	CodeArticleTransferTargetInvalid platform.Code = "article.transfer_target_invalid"
	// CodeArticleDeletionInProgress identifies an irreversible deletion in progress.
	CodeArticleDeletionInProgress platform.Code = "article.deletion_in_progress"
)

var articleErrorRegistry = mustArticleErrorRegistry(
	platform.Definition{
		Code: CodeRequestInvalidPrecondition, HTTPStatus: http.StatusBadRequest,
		Message: "资源版本条件格式无效", Retry: platform.RetryAfterCorrection,
		Responsibility: "Conditional Request Parser", LogLevel: platform.LogNone,
	},
	platform.Definition{
		Code: CodeRequestPreconditionRequired, HTTPStatus: http.StatusPreconditionRequired,
		Message: "必须提供当前资源版本", Retry: platform.RetryAllowed,
		Responsibility: "Conditional Request Middleware", LogLevel: platform.LogNone,
	},
	platform.Definition{
		Code: CodeArticleUnavailable, HTTPStatus: http.StatusNotFound,
		Message: "文章不存在或不可用", Retry: platform.RetryNever,
		Responsibility: "Article Query/Command Service", LogLevel: platform.LogNone,
	},
	platform.Definition{
		Code: CodeArticleInvalidState, HTTPStatus: http.StatusConflict,
		Message: "当前文章状态不允许该操作", Retry: platform.RetryAfterStateChange,
		Responsibility: "Article State Policy", LogLevel: platform.LogNone,
	},
	platform.Definition{
		Code: CodeArticlePreconditionFailed, HTTPStatus: http.StatusPreconditionFailed,
		Message: "文章已变化，请刷新后重试", Retry: platform.RetryAllowed,
		Responsibility: "Article Version Compare", LogLevel: platform.LogNone,
	},
	platform.Definition{
		Code: CodeArticleValidationFailed, HTTPStatus: http.StatusUnprocessableEntity,
		Message: "标题或正文不符合要求", Retry: platform.RetryAfterCorrection,
		Responsibility: "Article Validator", LogLevel: platform.LogNone,
	},
	platform.Definition{
		Code: CodeArticleTransferTargetInvalid, HTTPStatus: http.StatusUnprocessableEntity,
		Message: "目标用户不能成为新作者", Retry: platform.RetryAfterCorrection,
		Responsibility: "Article Identity Target Qualification", LogLevel: platform.LogNone,
	},
	platform.Definition{
		Code: CodeArticleDeletionInProgress, HTTPStatus: http.StatusServiceUnavailable,
		Message: "删除正在完成，请稍后核对", Retry: platform.RetryAllowed,
		Responsibility: "Permanent Deletion Coordinator", LogLevel: platform.LogNone,
	},
)

// ArticleCreator creates one Article for a trusted principal.
type ArticleCreator interface {
	Create(context.Context, session.Principal, CreateArticleInput) (Article, error)
}

// ArticleContentReplacer fully replaces author-owned Article content.
type ArticleContentReplacer interface {
	ReplaceContent(
		context.Context,
		session.Principal,
		string,
		int64,
		CreateArticleInput,
	) (Article, error)
}

// AuthorArticleReader serves the strict author-controlled detail projection.
type AuthorArticleReader interface {
	GetAuthorArticle(
		context.Context,
		session.Principal,
		string,
	) (AuthorArticleView, string, error)
}

// ArticleQuerier serves member-visible Article reads.
type ArticleQuerier interface {
	ListArticles(context.Context, ListArticlesRequest) (ListArticlesPage, error)
	GetArticle(context.Context, string) (MemberArticleDetailView, string, error)
}

// ArticleResponseWriteLogger records the terminal failure to write a committed response.
type ArticleResponseWriteLogger interface {
	LogCreateResponseWriteFailure(context.Context, string, error)
}

// SlogArticleResponseWriteLogger emits the bounded Article response-write event.
type SlogArticleResponseWriteLogger struct {
	logger  *slog.Logger
	service string
}

// NewSlogArticleResponseWriteLogger creates the structured Article response-write logger.
func NewSlogArticleResponseWriteLogger(
	logger *slog.Logger,
	service string,
) (*SlogArticleResponseWriteLogger, error) {
	service = strings.TrimSpace(service)
	if logger == nil || service == "" {
		return nil, errors.New("article response write logger dependencies are incomplete")
	}
	return &SlogArticleResponseWriteLogger{logger: logger, service: service}, nil
}

// LogCreateResponseWriteFailure records one low-sensitivity terminal write failure.
func (l *SlogArticleResponseWriteLogger) LogCreateResponseWriteFailure(
	ctx context.Context,
	traceID string,
	writeErr error,
) {
	if l == nil || l.logger == nil || strings.TrimSpace(l.service) == "" {
		return
	}
	if writeErr == nil {
		writeErr = errors.New("article response write failed without a cause")
	}
	logger, traceErr := platform.LoggerWithTrace(
		l.logger,
		l.service,
		articleLogModule,
		traceID,
	)
	if traceErr != nil {
		logger = l.logger.With(
			slog.String("service", l.service),
			slog.String("module", articleLogModule),
			slog.String("trace_id", traceID),
			slog.String("trace_context_error", traceErr.Error()),
		)
	}
	logger.ErrorContext(
		ctx,
		createArticleResponseWriteFailureEvent,
		slog.String("event", createArticleResponseWriteFailureEvent),
		slog.String("endpoint", createArticleEndpoint),
		slog.Int("status", http.StatusCreated),
		slog.String("reason_bucket", createArticleResponseWriteFailureReason),
		slog.String("error_code", string(platform.CodePlatformInternalError)),
		slog.Any("error", writeErr),
	)
}

// Handler owns Article HTTP transport behavior.
type Handler struct {
	adminQuery          AdminArticleQuerier
	orphanQuery         OrphanArticleQuerier
	orphanCommands      OrphanArticleCommander
	orphanDeliveryGate  OrphanDeliveryGate
	creator             ArticleCreator
	replacer            ArticleContentReplacer
	authorReader        AuthorArticleReader
	authorLister        AuthorArticleLister
	query               ArticleQuerier
	deleter             PermanentArticleDeleter
	responseWriteLogger ArticleResponseWriteLogger
	moderator           articleModerator
}

type articleModerator interface {
	Moderate(context.Context, session.Principal, string, int64, *ViolationCategory) (AdminArticleView, string, error)
}

// NewHandler creates an Article HTTP handler.
func NewHandler(
	creator ArticleCreator,
	responseWriteLogger ArticleResponseWriteLogger,
	deleters ...PermanentArticleDeleter,
) (*Handler, error) {
	if creator == nil {
		return nil, errors.New("article creator is required")
	}
	if responseWriteLogger == nil {
		return nil, errors.New("article response write logger is required")
	}
	if len(deleters) > 1 || len(deleters) == 1 && deleters[0] == nil {
		return nil, errors.New("article permanent deleter is invalid")
	}
	var deleter PermanentArticleDeleter
	if len(deleters) == 1 {
		deleter = deleters[0]
	}
	replacer, _ := creator.(ArticleContentReplacer)
	moderator, _ := creator.(articleModerator)
	return &Handler{
		creator:             creator,
		replacer:            replacer,
		deleter:             deleter,
		responseWriteLogger: responseWriteLogger,
		moderator:           moderator,
	}, nil
}

func (handler *Handler) MarkViolation(response http.ResponseWriter, request *http.Request) {
	handler.moderate(response, request, true)
}

func (handler *Handler) RestoreArticle(response http.ResponseWriter, request *http.Request) {
	handler.moderate(response, request, false)
}

func (handler *Handler) moderate(response http.ResponseWriter, request *http.Request, mark bool) {
	principal, available := platform.PrincipalFromContext(request.Context())
	if !available {
		_ = WriteArticleError(response, request, platform.NewPublicError(platform.CodeAuthSessionRequired,
			errors.New("article principal is required")))
		return
	}
	if !principal.Allows(session.AccessGeneral) || principal.Role != principalRoleAdmin {
		_ = WriteArticleError(response, request, ErrArticleForbidden)
		return
	}
	if handler == nil || handler.moderator == nil {
		_ = WriteArticleError(response, request, platform.NewPublicError(platform.CodePlatformInternalError,
			errors.New("article moderation handler is not configured")))
		return
	}
	articleID, version, category, err := moderationRequest(request, mark)
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	payload, etag, err := handler.moderator.Moderate(request.Context(), principal, articleID, version, category)
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	_ = WriteArticleJSON(response, http.StatusOK, payload, etag)
}

func moderationRequest(request *http.Request, mark bool) (string, int64, *ViolationCategory, error) {
	ifMatch, err := requestIfMatch(request)
	if err != nil {
		return "", 0, nil, err
	}
	action := "/restore"
	if mark {
		action = "/violation"
	}
	articleID := articleIDFromPath(strings.TrimSuffix(request.URL.Path, action))
	etagID, version, err := parseStrongETag(ifMatch)
	if err != nil || etagID != articleID {
		return "", 0, nil, ErrPreconditionFailed
	}
	if !mark {
		if request.Body != nil {
			body, err := io.ReadAll(io.LimitReader(request.Body, 1))
			if err != nil || len(body) != 0 {
				return "", 0, nil, articleInvalidJSONError(errors.Join(err, errors.New("restore does not accept a body")))
			}
		}
		return articleID, version, nil, nil
	}
	var input struct {
		Category ViolationCategory `json:"category"`
	}
	if err := decodeArticleWriteJSON(request, &input); err != nil {
		return "", 0, nil, err
	}
	if input.Category != ViolationViolence && input.Category != ViolationSexual {
		return "", 0, nil, ErrArticleValidationFailed
	}
	return articleID, version, &input.Category, nil
}

// NewQueryHandler creates an Article member-query HTTP handler.
func NewQueryHandler(query ArticleQuerier) (*Handler, error) {
	if query == nil {
		return nil, errors.New("article query service is required")
	}
	return &Handler{query: query}, nil
}

// NewAuthorDetailHandler creates the author-controlled detail HTTP handler.
func NewAuthorDetailHandler(reader AuthorArticleReader) (*Handler, error) {
	if reader == nil {
		return nil, errors.New("article author detail reader is required")
	}
	return &Handler{authorReader: reader}, nil
}

// NewAuthorArticleListHandler creates the "my articles" list HTTP handler.
func NewAuthorArticleListHandler(lister AuthorArticleLister) (*Handler, error) {
	if lister == nil {
		return nil, errors.New("article author list service is required")
	}
	return &Handler{authorLister: lister}, nil
}

// ListArticles handles GET /api/v1/articles.
func (h *Handler) ListArticles(response http.ResponseWriter, request *http.Request) {
	if h == nil || h.query == nil {
		_ = WriteArticleError(response, request, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("article query handler is not configured"),
		))
		return
	}
	if _, ok := platform.PrincipalFromContext(request.Context()); !ok {
		_ = WriteArticleError(response, request, platform.NewPublicError(
			platform.CodeAuthSessionRequired,
			errors.New("article principal is required"),
		))
		return
	}
	cursor, cursorPresent, err := singleArticleQueryParameter(request, "cursor")
	if err != nil || cursorPresent && cursor == "" {
		_ = WriteArticleError(response, request, ErrInvalidCursor)
		return
	}
	limit, err := articleQueryLimit(request)
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	page, err := h.query.ListArticles(request.Context(), ListArticlesRequest{
		Cursor: cursor,
		Limit:  limit,
	})
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	_ = WriteArticleJSON(response, http.StatusOK, page, "")
}

// GetArticle handles GET /api/v1/articles/{id}.
func (h *Handler) GetArticle(response http.ResponseWriter, request *http.Request) {
	if h == nil || h.query == nil {
		_ = WriteArticleError(response, request, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("article query handler is not configured"),
		))
		return
	}
	if _, ok := platform.PrincipalFromContext(request.Context()); !ok {
		_ = WriteArticleError(response, request, platform.NewPublicError(
			platform.CodeAuthSessionRequired,
			errors.New("article principal is required"),
		))
		return
	}
	detail, etag, err := h.query.GetArticle(
		request.Context(),
		articleIDFromPath(request.URL.Path),
	)
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	_ = WriteArticleJSON(response, http.StatusOK, detail, etag)
}

// Create handles POST /api/v1/articles.
func (h *Handler) Create(response http.ResponseWriter, request *http.Request) {
	if h == nil || h.creator == nil || h.responseWriteLogger == nil {
		_ = WriteArticleError(
			response,
			request,
			platform.NewPublicError(
				platform.CodePlatformInternalError,
				errors.New("article handler is not configured"),
			),
		)
		return
	}
	principal, ok := platform.PrincipalFromContext(request.Context())
	if !ok {
		_ = WriteArticleError(
			response,
			request,
			platform.NewPublicError(
				platform.CodeAuthSessionRequired,
				errors.New("article principal is required"),
			),
		)
		return
	}

	var input CreateArticleInput
	if err := decodeArticleWriteJSON(request, &input); err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	created, err := h.creator.Create(request.Context(), principal, input)
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	payload, err := (VisibilityPolicy{}).MemberDetailProjection(created, authorStatusActive)
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	etag, err := StrongETag(created)
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	response.Header().Set("Location", "/api/v1/articles/"+created.ID)
	if err := WriteArticleJSON(response, http.StatusCreated, payload, etag); err != nil {
		traceID, _ := articleTraceID(request)
		h.responseWriteLogger.LogCreateResponseWriteFailure(
			request.Context(),
			traceID,
			err,
		)
	}
}

// ListMyArticles handles GET /api/v1/users/me/articles.
func (h *Handler) ListMyArticles(
	response http.ResponseWriter,
	request *http.Request,
) {
	if h == nil || h.authorLister == nil {
		_ = WriteArticleError(response, request, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("article author list handler is not configured"),
		))
		return
	}
	principal, ok := platform.PrincipalFromContext(request.Context())
	if !ok {
		_ = WriteArticleError(response, request, platform.NewPublicError(
			platform.CodeAuthSessionRequired,
			errors.New("article principal is required"),
		))
		return
	}
	cursor, cursorPresent, err := singleArticleQueryParameter(request, "cursor")
	if err != nil || cursorPresent && cursor == "" {
		_ = WriteArticleError(response, request, ErrInvalidCursor)
		return
	}
	limit, err := articleQueryLimit(request)
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	page, err := h.authorLister.ListAuthorArticles(
		request.Context(),
		principal,
		ListArticlesRequest{Cursor: cursor, Limit: limit},
	)
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	_ = WriteArticleJSON(response, http.StatusOK, page, "")
}

// GetAuthorArticle handles GET /api/v1/users/me/articles/{id}.
//
// It returns the strict author-controlled projection for the current author and
// never falls back to member visibility: an active author can read their own
// violating Article, but only as the minimal restricted projection.
func (h *Handler) GetAuthorArticle(response http.ResponseWriter, request *http.Request) {
	if h == nil || h.authorReader == nil {
		_ = WriteArticleError(response, request, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("article author detail handler is not configured"),
		))
		return
	}
	principal, ok := platform.PrincipalFromContext(request.Context())
	if !ok {
		_ = WriteArticleError(response, request, platform.NewPublicError(
			platform.CodeAuthSessionRequired,
			errors.New("article principal is required"),
		))
		return
	}
	payload, etag, err := h.authorReader.GetAuthorArticle(
		request.Context(),
		principal,
		articleIDFromPath(request.URL.Path),
	)
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	_ = WriteArticleJSON(response, http.StatusOK, payload, etag)
}

// UpdateArticle handles PUT /api/v1/articles/{id}.
//
// It requires one strong If-Match value before any repository call, decodes the
// shared 262144-byte author write body, and returns the same minimal author
// projection and new ETag on success.
func (h *Handler) UpdateArticle(response http.ResponseWriter, request *http.Request) {
	if h == nil || h.replacer == nil {
		_ = WriteArticleError(response, request, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("article content replacement is not configured"),
		))
		return
	}
	principal, ok := platform.PrincipalFromContext(request.Context())
	if !ok {
		_ = WriteArticleError(response, request, platform.NewPublicError(
			platform.CodeAuthSessionRequired,
			errors.New("article principal is required"),
		))
		return
	}
	ifMatch, err := requestIfMatch(request)
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	articleID := articleIDFromPath(request.URL.Path)
	etagArticleID, expectedVersion, err := parseStrongETag(ifMatch)
	if err != nil || etagArticleID != articleID {
		_ = WriteArticleError(response, request, ErrInvalidPrecondition)
		return
	}
	var input CreateArticleInput
	if err := decodeArticleWriteJSON(request, &input); err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	updated, err := h.replacer.ReplaceContent(
		request.Context(),
		principal,
		articleID,
		expectedVersion,
		input,
	)
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	payload, err := (VisibilityPolicy{}).AuthorProjection(updated, principal)
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	etag, err := StrongETag(updated)
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	_ = WriteArticleJSON(response, http.StatusOK, payload, etag)
}

// DeleteOrdinary handles the shared author/administrator ordinary DELETE path.
func (h *Handler) DeleteOrdinary(response http.ResponseWriter, request *http.Request) {
	h.deletePermanent(response, request, DeleteOrdinary)
}

// DeleteViolating handles the administrator violating-article DELETE path.
func (h *Handler) DeleteViolating(response http.ResponseWriter, request *http.Request) {
	h.deletePermanent(response, request, DeleteViolating)
}

// DeleteOrphan handles the administrator orphan-article DELETE path.
func (h *Handler) DeleteOrphan(response http.ResponseWriter, request *http.Request) {
	if _, err := orphanRequestPrincipal(request); err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	h.deletePermanent(response, request, DeleteOrphan)
}

func (h *Handler) deletePermanent(
	response http.ResponseWriter,
	request *http.Request,
	kind DeleteKind,
) {
	if h == nil || h.deleter == nil {
		_ = WriteArticleError(response, request, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("article permanent deletion is not configured"),
		))
		return
	}
	principal, ok := platform.PrincipalFromContext(request.Context())
	if !ok {
		_ = WriteArticleError(response, request, platform.NewPublicError(
			platform.CodeAuthSessionRequired,
			errors.New("article principal is required"),
		))
		return
	}
	ifMatch, err := requestIfMatch(request)
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	articleID := articleIDFromPath(request.URL.Path)
	traceID, ok := articleTraceID(request)
	if !ok {
		traceID, err = platform.NewTraceID()
		if err != nil {
			_ = WriteArticleError(response, request, err)
			return
		}
	}
	err = h.deleter.DeletePermanent(request.Context(), principal, PermanentDeletionRequest{
		ArticleID: articleID, IfMatch: ifMatch, DeleteKind: kind, TraceID: traceID,
	})
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	response.Header().Set("X-Content-Type-Options", "nosniff")
	response.WriteHeader(http.StatusNoContent)
}

// MapArticleError maps an internal article error to the public REST contract.
func MapArticleError(err error) error {
	if err == nil {
		err = errors.New("article error is required")
	}
	var code platform.Code
	switch {
	case errors.Is(err, ErrInvalidCursor):
		code = platform.CodeRequestInvalidCursor
	case errors.Is(err, ErrInvalidPrecondition):
		code = CodeRequestInvalidPrecondition
	case errors.Is(err, ErrPreconditionRequired):
		code = CodeRequestPreconditionRequired
	case errors.Is(err, ErrArticleForbidden):
		code = platform.CodeAuthForbidden
	case errors.Is(err, ErrArticleUnavailable):
		code = CodeArticleUnavailable
	case errors.Is(err, ErrArticleInvalidState):
		code = CodeArticleInvalidState
	case errors.Is(err, ErrPreconditionFailed):
		code = CodeArticlePreconditionFailed
	case errors.Is(err, ErrArticleValidationFailed):
		code = CodeArticleValidationFailed
	case errors.Is(err, ErrTransferTargetInvalid):
		code = CodeArticleTransferTargetInvalid
	case errors.Is(err, ErrDeletionInProgress):
		code = CodeArticleDeletionInProgress
	default:
		existingCode := platform.PublicCode(err)
		if allowedArticleBoundaryCode(existingCode) {
			return err
		}
		code = platform.CodePlatformInternalError
	}
	return platform.NewPublicError(code, err)
}

// ArticleErrorDefinition resolves article and shared public error definitions.
func ArticleErrorDefinition(code platform.Code) (platform.Definition, bool) {
	if definition, ok := articleErrorRegistry.Definition(code); ok {
		return definition, true
	}
	return platform.DefaultRegistry.Definition(code)
}

// RequireRequestMatchingETag validates the request If-Match precondition.
func RequireRequestMatchingETag(request *http.Request, article Article) error {
	if request == nil {
		return ErrInvalidPrecondition
	}
	values := request.Header.Values("If-Match")
	if len(values) == 0 {
		return ErrPreconditionRequired
	}
	if len(values) != 1 {
		return ErrInvalidPrecondition
	}
	return RequireMatchingETag(values[0], article)
}

func requestIfMatch(request *http.Request) (string, error) {
	if request == nil {
		return "", ErrInvalidPrecondition
	}
	values := request.Header.Values("If-Match")
	if len(values) == 0 {
		return "", ErrPreconditionRequired
	}
	if len(values) != 1 {
		return "", ErrInvalidPrecondition
	}
	if _, _, err := parseStrongETag(values[0]); err != nil {
		return "", err
	}
	return values[0], nil
}

func articleIDFromPath(path string) string {
	path = strings.TrimSuffix(path, "/")
	if index := strings.LastIndexByte(path, '/'); index >= 0 {
		return path[index+1:]
	}
	return ""
}

func singleArticleQueryParameter(
	request *http.Request,
	name string,
) (string, bool, error) {
	if request == nil || request.URL == nil {
		return "", false, errors.New("article query request is invalid")
	}
	values, exists := request.URL.Query()[name]
	if !exists {
		return "", false, nil
	}
	if len(values) != 1 {
		return "", true, errors.New("article query parameter must appear once")
	}
	return values[0], true, nil
}

func articleQueryLimit(request *http.Request) (*int, error) {
	value, present, err := singleArticleQueryParameter(request, "limit")
	if err != nil {
		return nil, articleQueryLimitError(err)
	}
	if !present {
		return nil, nil
	}
	if value == "" {
		return nil, articleQueryLimitError(errors.New("article query limit is empty"))
	}
	limit, err := strconv.Atoi(value)
	if err != nil {
		return nil, articleQueryLimitError(err)
	}
	return &limit, nil
}

func articleQueryLimitError(cause error) error {
	return platform.NewPublicError(
		CodeArticleValidationFailed,
		cause,
		platform.FieldError{Field: "limit", Code: "invalid"},
	)
}

// WriteArticleError writes the shared public error envelope.
func WriteArticleError(
	response http.ResponseWriter,
	request *http.Request,
	err error,
) error {
	if err == nil {
		err = errors.New("article error is required")
	}
	mapped := MapArticleError(err)
	code := platform.PublicCode(mapped)
	if _, articleOwned := articleErrorRegistry.Definition(code); !articleOwned {
		if response != nil {
			response.Header().Set("X-Content-Type-Options", "nosniff")
		}
		return platform.WriteError(response, request, mapped)
	}
	if response == nil {
		return errors.New("HTTP response writer is required")
	}

	traceID, ok := articleTraceID(request)
	if !ok {
		var traceErr error
		traceID, traceErr = platform.NewTraceID()
		if traceErr != nil {
			return traceErr
		}
	}
	definition, _ := articleErrorRegistry.Definition(code)
	response.Header().Set(platform.TraceHeader, traceID)
	response.Header().Set("X-Content-Type-Options", "nosniff")
	return platform.WriteJSON(response, definition.HTTPStatus, platform.Envelope{
		Code:    code,
		Message: definition.Message,
		TraceID: traceID,
	})
}

// WriteArticleJSON writes an allowlisted article JSON response.
func WriteArticleJSON(
	response http.ResponseWriter,
	status int,
	payload any,
	etag string,
) error {
	if response == nil {
		return errors.New("HTTP response writer is required")
	}
	if etag != "" {
		if _, _, err := parseStrongETag(etag); err != nil {
			return err
		}
		response.Header().Set("ETag", etag)
	}
	response.Header().Set("X-Content-Type-Options", "nosniff")
	return platform.WriteJSON(response, status, payload)
}

func mustArticleErrorRegistry(definitions ...platform.Definition) platform.Registry {
	registry, err := platform.NewRegistry(definitions...)
	if err != nil {
		panic(err)
	}
	return registry
}

func articleTraceID(request *http.Request) (string, bool) {
	if request == nil {
		return "", false
	}
	return platform.TraceIDFromContext(request.Context())
}

func allowedArticleBoundaryCode(code platform.Code) bool {
	if _, articleOwned := articleErrorRegistry.Definition(code); articleOwned {
		return true
	}
	switch code {
	case platform.CodeRequestInvalidJSON,
		platform.CodeRequestInvalidCursor,
		platform.CodeAuthSessionRequired,
		platform.CodeAuthCSRFFailed,
		platform.CodeAuthForbidden,
		platform.CodeRequestRateLimited,
		platform.CodePlatformProcessingGateClosed,
		platform.CodePlatformDatabaseUnavailable,
		platform.CodePlatformRecoveryInProgress,
		platform.CodePlatformDependencyUnavailable,
		platform.CodePlatformInternalError:
		return true
	default:
		return false
	}
}

func decodeArticleWriteJSON(request *http.Request, destination any) error {
	if request == nil || request.Body == nil {
		return articleInvalidJSONError(errors.New("request body is required"))
	}
	body, err := io.ReadAll(io.LimitReader(request.Body, MaxArticleWriteBodyBytes+1))
	if err != nil {
		return articleInvalidJSONError(fmt.Errorf("read article JSON body: %w", err))
	}
	if len(body) > MaxArticleWriteBodyBytes {
		return articleInvalidJSONError(errors.New("article JSON body exceeds size limit"))
	}
	if !utf8.Valid(body) {
		return articleInvalidJSONError(errors.New("article JSON body is not valid UTF-8"))
	}
	request.Body = io.NopCloser(bytes.NewReader(body))
	return platform.DecodeJSON(request, MaxArticleWriteBodyBytes, destination)
}

func articleInvalidJSONError(cause error) error {
	return platform.NewPublicError(platform.CodeRequestInvalidJSON, cause)
}

// AdminArticleQuerier is the administrator-only read capability.
type AdminArticleQuerier interface {
	ListAdminArticles(context.Context, session.Principal, ListAdminArticlesRequest) (AdminArticlesPage, error)
	GetAdminArticle(context.Context, session.Principal, string) (AdminArticleView, string, error)
}

// NewAdminQueryHandler binds the shared query service without command dependencies.
func NewAdminQueryHandler(query AdminArticleQuerier) (*Handler, error) {
	if query == nil {
		return nil, errors.New("admin article query service is required")
	}
	return &Handler{adminQuery: query}, nil
}

// ListAdminArticles handles the complete administrator collection GET.
func (h *Handler) ListAdminArticles(response http.ResponseWriter, request *http.Request) {
	principal, err := orphanRequestPrincipal(request)
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	if h == nil || h.adminQuery == nil {
		_ = WriteArticleError(response, request, errors.New("admin query is not configured"))
		return
	}
	values, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		_ = WriteArticleError(response, request, ErrArticleValidationFailed)
		return
	}
	input, err := adminArticleQueryRequest(values)
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	page, err := h.adminQuery.ListAdminArticles(request.Context(), principal, input)
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	_ = WriteArticleJSON(response, http.StatusOK, page, "")
}

func adminArticleQueryRequest(values url.Values) (ListAdminArticlesRequest, error) {
	input := ListAdminArticlesRequest{View: AdminListViewAll}
	if entries, present := values["view"]; present {
		if len(entries) != 1 || entries[0] == "" {
			return input, ErrArticleValidationFailed
		}
		input.View = AdminArticleListView(entries[0])
		if _, err := adminCursorView(input.View); err != nil {
			return input, err
		}
	}
	if entries, present := values["cursor"]; present {
		if len(entries) != 1 || entries[0] == "" {
			return input, ErrInvalidCursor
		}
		input.Cursor = entries[0]
	}
	if entries, present := values["limit"]; present {
		if len(entries) != 1 || entries[0] == "" {
			return input, articleQueryLimitError(errors.New("admin limit must appear once"))
		}
		limit, err := strconv.Atoi(entries[0])
		if err != nil {
			return input, articleQueryLimitError(errors.New("admin limit is not an integer"))
		}
		input.Limit = &limit
	}
	return input, nil
}

// GetAdminArticle handles one administrator Article detail GET.
func (h *Handler) GetAdminArticle(response http.ResponseWriter, request *http.Request) {
	principal, err := orphanRequestPrincipal(request)
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	if h == nil || h.adminQuery == nil {
		_ = WriteArticleError(response, request, errors.New("admin query is not configured"))
		return
	}
	view, etag, err := h.adminQuery.GetAdminArticle(request.Context(), principal, articleIDFromPath(request.URL.Path))
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	_ = WriteArticleJSON(response, http.StatusOK, view, etag)
}

// OrphanArticleQuerier serves the administrator-only disposition set.
type OrphanArticleQuerier interface {
	ListOrphanArticles(context.Context, session.Principal, ListArticlesRequest) (OrphanArticlesPage, error)
	GetOrphanArticle(context.Context, session.Principal, string) (AdminArticleView, string, error)
}

// OrphanArticleCommander is the already-authorized publish/transfer capability.
type OrphanArticleCommander interface {
	PublishOrphan(context.Context, session.Principal, string, int64) (OrphanDispositionResult, error)
	TransferOrphan(context.Context, session.Principal, string, int64, string) (OrphanDispositionResult, error)
}

// OrphanDeliveryGate reports permission, never SMTP delivery success.
type OrphanDeliveryGate interface{ RequireOpen(context.Context) error }

// NewOrphanQueryHandler adds no capabilities to the member query contract.
func NewOrphanQueryHandler(query OrphanArticleQuerier) (*Handler, error) {
	if query == nil {
		return nil, errors.New("orphan query is required")
	}
	return &Handler{orphanQuery: query}, nil
}

// NewOrphanCommandHandler adds disposition to a privately owned handler copy.
func NewOrphanCommandHandler(base *Handler, commands OrphanArticleCommander, gate OrphanDeliveryGate) (*Handler, error) {
	if base == nil || commands == nil {
		return nil, errors.New("orphan command dependencies are required")
	}
	handler := *base
	handler.orphanCommands, handler.orphanDeliveryGate = commands, gate
	return &handler, nil
}

func orphanRequestPrincipal(request *http.Request) (session.Principal, error) {
	principal, present := platform.PrincipalFromContext(request.Context())
	if !present {
		return principal, platform.NewPublicError(platform.CodeAuthSessionRequired, errors.New("orphan principal is required"))
	}
	return principal, requireOrphanAdministrator(principal)
}

// ListOrphanArticles handles the administrator orphan collection GET.
func (handler *Handler) ListOrphanArticles(response http.ResponseWriter, request *http.Request) {
	principal, err := orphanRequestPrincipal(request)
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	if handler == nil || handler.orphanQuery == nil {
		_ = WriteArticleError(response, request, errors.New("orphan query is not configured"))
		return
	}
	cursor, present, err := singleArticleQueryParameter(request, "cursor")
	if err != nil || present && cursor == "" {
		_ = WriteArticleError(response, request, ErrInvalidCursor)
		return
	}
	limit, err := articleQueryLimit(request)
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	page, err := handler.orphanQuery.ListOrphanArticles(request.Context(), principal, ListArticlesRequest{Cursor: cursor, Limit: limit})
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	_ = WriteArticleJSON(response, http.StatusOK, page, "")
}

// GetOrphanArticle handles the administrator orphan detail GET.
func (handler *Handler) GetOrphanArticle(response http.ResponseWriter, request *http.Request) {
	principal, err := orphanRequestPrincipal(request)
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	if handler == nil || handler.orphanQuery == nil {
		_ = WriteArticleError(response, request, errors.New("orphan query is not configured"))
		return
	}
	view, etag, err := handler.orphanQuery.GetOrphanArticle(request.Context(), principal, articleIDFromPath(request.URL.Path))
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	_ = WriteArticleJSON(response, http.StatusOK, view, etag)
}

// PublishOrphan handles the body-free administrator publish POST.
func (handler *Handler) PublishOrphan(response http.ResponseWriter, request *http.Request) {
	handler.disposeOrphanRequest(response, request, false)
}

// TransferOrphan snapshots contacts and never sends SMTP from the HTTP path.
func (handler *Handler) TransferOrphan(response http.ResponseWriter, request *http.Request) {
	handler.disposeOrphanRequest(response, request, true)
}

func (handler *Handler) disposeOrphanRequest(response http.ResponseWriter, request *http.Request, transfer bool) {
	principal, err := orphanRequestPrincipal(request)
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	if handler == nil || handler.orphanCommands == nil {
		_ = WriteArticleError(response, request, errors.New("orphan command is not configured"))
		return
	}
	articleID, version, targetID, err := orphanWriteRequest(request, transfer)
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	var result OrphanDispositionResult
	if transfer {
		result, err = handler.orphanCommands.TransferOrphan(request.Context(), principal, articleID, version, targetID)
	} else {
		result, err = handler.orphanCommands.PublishOrphan(request.Context(), principal, articleID, version)
	}
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	view, err := (VisibilityPolicy{}).AdminProjection(result.Article, principal, authorStatusActive)
	if err != nil {
		_ = WriteArticleError(response, request, err)
		return
	}
	if !transfer {
		_ = WriteArticleJSON(response, http.StatusOK, view, result.ETag)
		return
	}
	capability := orphanDeliveryClosed
	if handler.orphanDeliveryGate != nil && handler.orphanDeliveryGate.RequireOpen(request.Context()) == nil {
		capability = orphanDeliveryOpen
	}
	if result.NoticeID == nil || !validUUID(*result.NoticeID) {
		_ = WriteArticleError(response, request, errors.New("committed transfer notice is required"))
		return
	}
	payload := orphanTransferResponse{Article: view, Notification: orphanNotificationView{ID: *result.NoticeID,
		DeliveryCapability: capability, Recipients: [2]orphanRecipientView{
			{Kind: RecipientNewAuthor, Status: RecipientPending}, {Kind: RecipientFormerDeletedAuthor, Status: RecipientPending}}}}
	_ = WriteArticleJSON(response, http.StatusOK, payload, result.ETag)
}

const (
	orphanDeliveryClosed = "closed"
	orphanDeliveryOpen   = "open"
)

type orphanRecipientView struct {
	Kind   RecipientKind   `json:"kind"`
	Status RecipientStatus `json:"status"`
}
type orphanNotificationView struct {
	ID                 string                 `json:"id"`
	DeliveryCapability string                 `json:"delivery_capability"`
	Recipients         [2]orphanRecipientView `json:"recipients"`
}
type orphanTransferResponse struct {
	Article      AdminArticleView       `json:"article"`
	Notification orphanNotificationView `json:"notification"`
}

func orphanWriteRequest(request *http.Request, transfer bool) (string, int64, string, error) {
	ifMatch, err := requestIfMatch(request)
	if err != nil {
		return "", 0, "", err
	}
	action := "/publish"
	if transfer {
		action = "/transfer"
	}
	articleID := articleIDFromPath(strings.TrimSuffix(request.URL.Path, action))
	etagID, version, err := parseStrongETag(ifMatch)
	if err != nil || etagID != articleID {
		return "", 0, "", ErrPreconditionFailed
	}
	if !transfer {
		if request.Body != nil {
			body, err := io.ReadAll(io.LimitReader(request.Body, 1))
			if err != nil || len(body) != 0 {
				return "", 0, "", articleInvalidJSONError(errors.Join(err, errors.New("publish does not accept a body")))
			}
		}
		return articleID, version, "", nil
	}
	var input struct {
		NewAuthorID string `json:"new_author_id"`
	}
	if err := decodeArticleWriteJSON(request, &input); err != nil {
		return "", 0, "", err
	}
	if !validUUID(input.NewAuthorID) {
		return "", 0, "", ErrTransferTargetInvalid
	}
	return articleID, version, input.NewAuthorID, nil
}
