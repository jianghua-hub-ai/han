// Package article owns article, transfer notification, and permanent deletion models.
package article

import (
	"time"

	"han/internal/recovery"
)

const (
	// ArticleDeletionNamespace separates article deletion evidence from recovery records.
	// It is a compile-time alias of the unique recovery authority (F004).
	ArticleDeletionNamespace = recovery.ArticleDeletionNamespaceV1
	// DeletionReplayEvidenceVersion is the canonical replay evidence schema version.
	// It is a compile-time alias of the unique recovery authority (F004).
	DeletionReplayEvidenceVersion = recovery.ReplayEvidenceVersionV1
	// TransferNoticePurpose is the only permitted purpose for transfer notifications.
	TransferNoticePurpose = "orphan_article_transfer_notice"
)

// PublicationState is the article-owned publication intent.
type PublicationState string

const (
	// PublicationPublished permits visibility when every other independent constraint allows it.
	PublicationPublished PublicationState = "published"
	// PublicationHeld keeps an orphan article unavailable until explicit disposition.
	PublicationHeld PublicationState = "held"
)

// ViolationState is the independent content-governance fact.
type ViolationState string

const (
	// ViolationClear means no active violation decision exists.
	ViolationClear ViolationState = "clear"
	// ViolationViolating means an active violation decision exists.
	ViolationViolating ViolationState = "violating"
)

// ViolationCategory identifies the accepted violation classification.
type ViolationCategory string

const (
	// ViolationViolence classifies violent content.
	ViolationViolence ViolationCategory = "violence"
	// ViolationSexual classifies sexual content.
	ViolationSexual ViolationCategory = "sexual"
)

// DeletionState is the internal permanent-deletion protocol state.
type DeletionState string

const (
	// DeletionActive permits normal article state handling.
	DeletionActive DeletionState = "active"
	// DeletionPending hides content while irreversible deletion converges.
	DeletionPending DeletionState = "deletion_pending"
)

// RecipientKind identifies one of the two transfer notification recipients.
type RecipientKind string

const (
	// RecipientNewAuthor notifies the new article author.
	RecipientNewAuthor RecipientKind = "new_author"
	// RecipientFormerDeletedAuthor notifies the deleted former author.
	RecipientFormerDeletedAuthor RecipientKind = "former_deleted_author"
)

// RecipientStatus is the independent delivery state for one recipient.
type RecipientStatus string

const (
	// RecipientPending has not been claimed for delivery.
	RecipientPending RecipientStatus = "pending"
	// RecipientRetryWait waits until its next permitted claim.
	RecipientRetryWait RecipientStatus = "retry_wait"
	// RecipientProcessing is held by one lease owner.
	RecipientProcessing RecipientStatus = "processing"
	// RecipientAccepted records SMTP acceptance.
	RecipientAccepted RecipientStatus = "accepted"
	// RecipientFailedTerminal will not be retried automatically.
	RecipientFailedTerminal RecipientStatus = "failed_terminal"
)

// DeleteKind identifies the authorized permanent-deletion entry point.
type DeleteKind string

const (
	// DeleteOrdinary deletes an ordinary article.
	DeleteOrdinary DeleteKind = "ordinary"
	// DeleteViolating deletes an article with an active violation decision.
	DeleteViolating DeleteKind = "violating"
	// DeleteOrphan deletes an orphan article.
	DeleteOrphan DeleteKind = "orphan"
)

// DeletionOperationStatus is the irreversible deletion protocol state.
type DeletionOperationStatus string

const (
	// DeletionOperationPendingJournal awaits a durable external tombstone.
	DeletionOperationPendingJournal DeletionOperationStatus = "pending_journal"
	// DeletionOperationJournaled has a durable external tombstone.
	DeletionOperationJournaled DeletionOperationStatus = "journaled"
	// DeletionOperationFinalizing is held by one finalizer lease.
	DeletionOperationFinalizing DeletionOperationStatus = "finalizing"
	// DeletionOperationCompleted has finished physical deletion and audit.
	DeletionOperationCompleted DeletionOperationStatus = "completed"
)

// ReplayRunStatus is the article deletion replay coordination state.
type ReplayRunStatus string

const (
	// ReplayRunRunning is computing or reconciling evidence.
	ReplayRunRunning ReplayRunStatus = "running"
	// ReplayRunFailed has stable low-sensitivity failure evidence.
	ReplayRunFailed ReplayRunStatus = "failed"
	// ReplayRunReady points at current immutable evidence.
	ReplayRunReady ReplayRunStatus = "ready"
)

// Article stores only authoritative article-owned state.
type Article struct {
	ID                  string
	AuthorID            *string
	OrphanedFromUserID  *string
	Title               string
	Body                string
	PublicationState    PublicationState
	ViolationState      ViolationState
	ViolationCategory   *ViolationCategory
	DeletionState       DeletionState
	DeletionOperationID *string
	Version             int64
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// TransferNotice records one committed orphan-article transfer notification.
type TransferNotice struct {
	ID                  string
	ArticleRef          string
	NewAuthorID         string
	FormerDeletedUserID string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// TransferNoticeRecipient stores one independently deliverable address snapshot.
type TransferNoticeRecipient struct {
	NoticeID      string
	Kind          RecipientKind
	EmailSnapshot string
	Status        RecipientStatus
	ClaimedBy     *string
	LockedUntil   *time.Time
	NextAttemptAt *time.Time
	AttemptCount  int64
	MaxAttempts   int64
	LastAttemptAt *time.Time
	AcceptedAt    *time.Time
	FailureBucket *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// DeletionOperation coordinates a durable tombstone and physical deletion.
type DeletionOperation struct {
	ID                 string
	ArticleID          string
	ArticleVersion     int64
	DeleteKind         DeleteKind
	ActorUserID        *string
	ActorRole          *string
	OrphanedFromUserID *string
	Status             DeletionOperationStatus
	TombstoneSequence  *int64
	JournalKeyID       *string
	JournalKeyVersion  *string
	ClaimedBy          *string
	LockedUntil        *time.Time
	AuditEventID       *string
	StartedAt          time.Time
	CompletedAt        *time.Time
}

// ArticleDeletionTombstone is the minimal external journal record shape.
type ArticleDeletionTombstone struct {
	Namespace           string
	Sequence            int64
	ArticleID           string
	ArticleVersion      int64
	DeletionOperationID string
	DeletedAt           time.Time
	KeyID               string
	KeyVersion          string
	Stage               string
	PreviousRecordHash  string
	RecordHMAC          string
}

// DeletionReplayRun owns one immutable run, namespace, and backup binding.
type DeletionReplayRun struct {
	RunID              string
	Namespace          string
	BackupRef          string
	InputDigest        *string
	CurrentEvidenceRef *string
	Status             ReplayRunStatus
	LastErrorBucket    *string
	StartedAt          time.Time
	UpdatedAt          time.Time
}

// DeletionReplayEvidence is one immutable canonical replay result.
type DeletionReplayEvidence struct {
	EvidenceVersion                   string
	RunID                             string
	BackupRef                         string
	Namespace                         string
	JournalHeadSequence               int64
	JournalHeadHash                   string
	InventoryScope                    string
	BackupCutoff                      time.Time
	InventoryDigest                   string
	DeletionOperationsDigest          string
	AuditEventsDigest                 string
	TombstonesTotal                   int64
	TombstonesReplayed                int64
	AlreadyAbsent                     int64
	DeletionOperationsReconciledCount int64
	AuditEventsVerifiedCount          int64
	EvidenceDigest                    string
	EvidenceRef                       string
	CompletedAt                       time.Time
}

// AttemptDisposition is the store-decided outcome of one Begin call.
type AttemptDisposition string

const (
	// AttemptCreated means this invocation created the first attempt row.
	AttemptCreated AttemptDisposition = "created"
	// AttemptResumedCandidate means the candidate token is already persisted.
	AttemptResumedCandidate AttemptDisposition = "resumed_candidate"
	// AttemptJoinedCurrent means another owner is already running this attempt.
	AttemptJoinedCurrent AttemptDisposition = "joined_current"
	// AttemptAdvancedFailed means a matching failed attempt advanced a generation.
	AttemptAdvancedFailed AttemptDisposition = "advanced_from_failed"
	// AttemptAdvancedStale means a matching stale Ready attempt advanced a generation.
	AttemptAdvancedStale AttemptDisposition = "advanced_from_stale"
	// AttemptAlreadyReady means the persisted input still matches and is reusable.
	AttemptAlreadyReady AttemptDisposition = "already_ready"
	// AttemptSuperseded means the expected snapshot no longer describes the row.
	AttemptSuperseded AttemptDisposition = "superseded"
)

// AttemptSnapshot is the article-internal authoritative attempt handle.
//
// It is derived from the run row and never crosses the article module boundary
// or the public ReplayStatus DTO. Generation and token are the only values that
// authorize Ready/Fail, so callers must use the exact handle returned by Begin.
type AttemptSnapshot struct {
	Exists             bool
	Status             ReplayRunStatus
	Generation         int64
	Token              string
	BackupRef          string
	InputDigest        *string
	CurrentEvidenceRef *string
	LastErrorBucket    *string
}

// BeginResult carries the authoritative attempt handle and the store decision.
type BeginResult struct {
	Attempt     AttemptSnapshot
	Disposition AttemptDisposition
}

// ArticleAuthorView is the public author reference used by article responses.
type ArticleAuthorView struct {
	Kind  string  `json:"kind"`
	ID    *string `json:"id"`
	Label string  `json:"label,omitempty"`
}

// MemberArticleSummaryView is the body-free member list projection.
type MemberArticleSummaryView struct {
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	Author    ArticleAuthorView `json:"author"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// MemberArticleDetailView is the allowlisted member detail projection.
type MemberArticleDetailView struct {
	ID               string            `json:"id"`
	Title            string            `json:"title"`
	Body             string            `json:"body"`
	Author           ArticleAuthorView `json:"author"`
	PublicationState PublicationState  `json:"publication_state"`
	ViolationState   ViolationState    `json:"violation_state"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

// ModerationStatus is the non-sensitive author-facing governance projection.
type ModerationStatus string

const (
	// ModerationNormal means the article has no active violation decision.
	ModerationNormal ModerationStatus = "normal"
	// ModerationRestricted means an active governance constraint exists.
	ModerationRestricted ModerationStatus = "restricted"
)

// AuthorArticleView is the strict author-controlled detail projection.
type AuthorArticleView struct {
	Title            string           `json:"title"`
	Body             string           `json:"body"`
	ModerationStatus ModerationStatus `json:"moderation_status"`
	Editable         bool             `json:"editable"`
}

// HiddenReason is a stable administrator-facing visibility explanation.
type HiddenReason string

const (
	// HiddenAuthorSuspended means the owning account is not active.
	HiddenAuthorSuspended HiddenReason = "author_suspended"
	// HiddenViolation means governance currently restricts the article.
	HiddenViolation HiddenReason = "violation"
	// HiddenOrphanHold means an orphan article awaits administrator disposition.
	HiddenOrphanHold HiddenReason = "orphan_hold"
)

// AdminArticleView is the allowlisted governance projection.
type AdminArticleView struct {
	ID                 string             `json:"id"`
	Title              string             `json:"title"`
	Body               string             `json:"body"`
	Author             ArticleAuthorView  `json:"author"`
	PublicationState   PublicationState   `json:"publication_state"`
	ViolationState     ViolationState     `json:"violation_state"`
	ViolationCategory  *ViolationCategory `json:"violation_category"`
	HiddenReasons      []HiddenReason     `json:"hidden_reasons"`
	OrphanedFromUserID *string            `json:"orphaned_from_user_id"`
	Version            int64              `json:"version"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}

// CursorView binds a cursor to one list projection.
type CursorView string

const (
	// CursorViewMember is the regular member article list.
	CursorViewMember CursorView = "member"
	// CursorViewAdminAll is the complete administrator list.
	CursorViewAdminAll CursorView = "admin_all"
	// CursorViewAdminViolating is the administrator violating list.
	CursorViewAdminViolating CursorView = "admin_violating"
	// CursorViewAdminHidden is the administrator hidden-article list.
	CursorViewAdminHidden CursorView = "admin_hidden"
	// CursorViewAdminOrphan is the administrator orphan list.
	CursorViewAdminOrphan CursorView = "admin_orphan"
)

// ArticleCursor is the stable keyset boundary for an article list.
type ArticleCursor struct {
	CreatedAt time.Time
	ID        string
	View      CursorView
}
