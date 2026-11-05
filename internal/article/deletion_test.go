package article

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"han/internal/audit"
	"han/internal/platform"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	deletionArticleID   = "10000000-0000-4000-8000-000000000101"
	deletionAuthorID    = "20000000-0000-4000-8000-000000000101"
	deletionAdminID     = "20000000-0000-4000-8000-000000000102"
	deletionOperationID = "30000000-0000-4000-8000-000000000101"
	deletionAuditID     = "40000000-0000-4000-8000-000000000101"
	deletionTraceID     = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
)

var deletionNow = time.Date(2026, 10, 23, 10, 30, 0, 0, time.UTC)

func TestPermanentDeletion_AuthorDualCompletion_ReturnsNoContent(t *testing.T) {
	fixture := newPermanentDeletionFixture(t)
	handler := fixture.handler(t)
	request := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/articles/"+deletionArticleID,
		nil,
	)
	request.Header.Set("If-Match", `"article-`+deletionArticleID+`-v3"`)
	response := httptest.NewRecorder()

	fixture.serve(t, handler.DeleteOrdinary, deletionAuthorPrincipal(), response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusNoContent, response.Body)
	}
	if fixture.repository.article != nil {
		t.Fatal("article remains after successful permanent deletion")
	}
	if fixture.repository.operation == nil ||
		fixture.repository.operation.Status != DeletionOperationCompleted {
		t.Fatalf("operation = %#v, want completed", fixture.repository.operation)
	}
	if len(fixture.journal.records) != 1 {
		t.Fatalf("tombstones = %d, want 1", len(fixture.journal.records))
	}
	if len(fixture.auditor.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(fixture.auditor.events))
	}
}

func TestPermanentDeletion_FailuresRemainPendingAndRetryConverges(t *testing.T) {
	t.Run("journal failure", func(t *testing.T) {
		fixture := newPermanentDeletionFixture(t)
		fixture.journal.appendErr = errors.New("synthetic fsync failure")

		err := fixture.coordinator.DeletePermanent(
			context.Background(),
			deletionAuthorPrincipal(),
			deletionRequest(DeleteOrdinary),
		)

		if !errors.Is(err, ErrDeletionInProgress) {
			t.Fatalf("DeletePermanent() error = %v, want ErrDeletionInProgress", err)
		}
		requirePendingDeletion(t, fixture.repository)
		if fixture.finalizer.calls.Load() != 0 {
			t.Fatalf("finalizer calls = %d, want 0", fixture.finalizer.calls.Load())
		}
	})

	t.Run("journal binding database failure then original retry", func(t *testing.T) {
		fixture := newPermanentDeletionFixture(t)
		fixture.repository.markJournaledErr = errors.New("synthetic database failure")

		firstErr := fixture.coordinator.DeletePermanent(
			context.Background(),
			deletionAuthorPrincipal(),
			deletionRequest(DeleteOrdinary),
		)
		if !errors.Is(firstErr, ErrDeletionInProgress) {
			t.Fatalf("first DeletePermanent() error = %v, want ErrDeletionInProgress", firstErr)
		}
		requirePendingDeletion(t, fixture.repository)
		if len(fixture.journal.records) != 1 {
			t.Fatalf("first tombstones = %d, want 1", len(fixture.journal.records))
		}

		fixture.repository.markJournaledErr = nil
		if err := fixture.coordinator.DeletePermanent(
			context.Background(),
			deletionAuthorPrincipal(),
			deletionRequest(DeleteOrdinary),
		); err != nil {
			t.Fatalf("retry DeletePermanent() error = %v", err)
		}
		if fixture.repository.article != nil {
			t.Fatal("article remains after retry convergence")
		}
		if len(fixture.journal.records) != 1 {
			t.Fatalf("retry tombstones = %d, want 1", len(fixture.journal.records))
		}
	})

	t.Run("finalizer failure returns non success then original retry", func(t *testing.T) {
		fixture := newPermanentDeletionFixture(t)
		fixture.finalizer.err = errors.New("synthetic finalizer failure")
		handler := fixture.handler(t)

		firstRequest := httptest.NewRequest(
			http.MethodDelete,
			"/api/v1/articles/"+deletionArticleID,
			nil,
		)
		firstRequest.Header.Set("If-Match", `"article-`+deletionArticleID+`-v3"`)
		firstResponse := httptest.NewRecorder()
		fixture.serve(
			t,
			handler.DeleteOrdinary,
			deletionAuthorPrincipal(),
			firstResponse,
			firstRequest,
		)

		if firstResponse.Code != http.StatusServiceUnavailable {
			t.Fatalf("first status = %d, want %d; body=%s",
				firstResponse.Code, http.StatusServiceUnavailable, firstResponse.Body)
		}
		if fixture.repository.article == nil ||
			fixture.repository.article.DeletionState != DeletionPending {
			t.Fatalf("article = %#v, want deletion_pending", fixture.repository.article)
		}
		if fixture.repository.operation == nil ||
			fixture.repository.operation.Status != DeletionOperationJournaled {
			t.Fatalf("operation = %#v, want journaled", fixture.repository.operation)
		}
		if len(fixture.journal.records) != 1 || len(fixture.auditor.events) != 0 {
			t.Fatalf("first attempt tombstones=%d audits=%d, want 1/0",
				len(fixture.journal.records), len(fixture.auditor.events))
		}

		fixture.finalizer.err = nil
		retryRequest := httptest.NewRequest(
			http.MethodDelete,
			"/api/v1/articles/"+deletionArticleID,
			nil,
		)
		retryRequest.Header.Set("If-Match", `"article-`+deletionArticleID+`-v3"`)
		retryResponse := httptest.NewRecorder()
		fixture.serve(
			t,
			handler.DeleteOrdinary,
			deletionAuthorPrincipal(),
			retryResponse,
			retryRequest,
		)

		if retryResponse.Code != http.StatusNoContent {
			t.Fatalf("retry status = %d, want %d; body=%s",
				retryResponse.Code, http.StatusNoContent, retryResponse.Body)
		}
		if fixture.repository.article != nil ||
			fixture.repository.operation == nil ||
			fixture.repository.operation.Status != DeletionOperationCompleted {
			t.Fatalf("retry did not converge: article=%#v operation=%#v",
				fixture.repository.article, fixture.repository.operation)
		}
		if len(fixture.journal.records) != 1 || len(fixture.auditor.events) != 1 {
			t.Fatalf("retry tombstones=%d audits=%d, want 1/1",
				len(fixture.journal.records), len(fixture.auditor.events))
		}
	})
}

func TestPermanentDeletion_AuthorizationAndIfMatch_PrecedeT1(t *testing.T) {
	tests := []struct {
		name      string
		principal session.Principal
		request   PermanentDeletionRequest
		wantError error
	}{
		{
			name:      "other member",
			principal: activeDeletionPrincipal("20000000-0000-4000-8000-000000000199", "member"),
			request:   deletionRequest(DeleteOrdinary),
			wantError: ErrArticleUnavailable,
		},
		{
			name:      "stale etag",
			principal: deletionAuthorPrincipal(),
			request: PermanentDeletionRequest{
				ArticleID:  deletionArticleID,
				IfMatch:    `"article-` + deletionArticleID + `-v2"`,
				DeleteKind: DeleteOrdinary,
				TraceID:    deletionTraceID,
			},
			wantError: ErrPreconditionFailed,
		},
		{
			name:      "wrong deletion entry",
			principal: deletionAdminPrincipal(),
			request:   deletionRequest(DeleteViolating),
			wantError: ErrArticleInvalidState,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newPermanentDeletionFixture(t)

			err := fixture.coordinator.DeletePermanent(
				context.Background(),
				test.principal,
				test.request,
			)

			if !errors.Is(err, test.wantError) {
				t.Fatalf("DeletePermanent() error = %v, want %v", err, test.wantError)
			}
			if fixture.repository.operation != nil ||
				fixture.repository.article == nil ||
				fixture.repository.article.DeletionState != DeletionActive {
				t.Fatalf("T1 changed state: article=%#v operation=%#v",
					fixture.repository.article, fixture.repository.operation)
			}
			if len(fixture.journal.records) != 0 || fixture.finalizer.calls.Load() != 0 {
				t.Fatal("authorization/precondition failure reached irreversible deletion")
			}
		})
	}
}

func TestPermanentDeletion_AllDeleteKindsShareProtocol(t *testing.T) {
	tests := []struct {
		name        string
		kind        DeleteKind
		principal   session.Principal
		prepare     func(*Article)
		auditAction string
	}{
		{
			name:        "administrator ordinary",
			kind:        DeleteOrdinary,
			principal:   deletionAdminPrincipal(),
			prepare:     func(*Article) {},
			auditAction: "article.delete",
		},
		{
			name:      "administrator violating",
			kind:      DeleteViolating,
			principal: deletionAdminPrincipal(),
			prepare: func(article *Article) {
				category := ViolationViolence
				article.ViolationState = ViolationViolating
				article.ViolationCategory = &category
			},
			auditAction: "article.violation.delete",
		},
		{
			name:      "administrator orphan",
			kind:      DeleteOrphan,
			principal: deletionAdminPrincipal(),
			prepare: func(article *Article) {
				source := deletionAuthorID
				article.AuthorID = nil
				article.OrphanedFromUserID = &source
				article.PublicationState = PublicationHeld
			},
			auditAction: "article.orphan.delete",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newPermanentDeletionFixture(t)
			test.prepare(fixture.repository.article)

			if err := fixture.coordinator.DeletePermanent(
				context.Background(),
				test.principal,
				deletionRequest(test.kind),
			); err != nil {
				t.Fatalf("DeletePermanent() error = %v", err)
			}

			if fixture.repository.article != nil {
				t.Fatal("article remains after permanent deletion")
			}
			if fixture.repository.operation == nil ||
				fixture.repository.operation.Status != DeletionOperationCompleted ||
				fixture.repository.operation.DeleteKind != test.kind {
				t.Fatalf("operation = %#v, want completed %s",
					fixture.repository.operation, test.kind)
			}
			if len(fixture.journal.records) != 1 {
				t.Fatalf("tombstones = %d, want 1", len(fixture.journal.records))
			}
			if len(fixture.auditor.events) != 1 ||
				fixture.auditor.events[0].Action != test.auditAction {
				t.Fatalf("audit events = %#v, want action %q",
					fixture.auditor.events, test.auditAction)
			}
		})
	}
}

func TestPermanentDeletion_AWIRERTR05ThreeRouteRoleStatusAuthorizationMatrix(
	t *testing.T,
) {
	ordinary := deletionTestArticle()
	violating := deletionTestArticle()
	category := ViolationViolence
	violating.ViolationState = ViolationViolating
	violating.ViolationCategory = &category
	orphan := deletionTestArticle()
	orphan.AuthorID = nil
	orphan.OrphanedFromUserID = stringPointer(deletionAuthorID)
	orphan.PublicationState = PublicationHeld

	tests := []struct {
		name      string
		principal session.Principal
		kind      DeleteKind
		article   Article
		wantError error
	}{
		{
			name: "ordinary active author", principal: deletionAuthorPrincipal(),
			kind: DeleteOrdinary, article: ordinary,
		},
		{
			name: "ordinary active administrator", principal: deletionAdminPrincipal(),
			kind: DeleteOrdinary, article: ordinary,
		},
		{
			name: "ordinary active other member",
			principal: activeDeletionPrincipal(
				"20000000-0000-4000-8000-000000000099",
				"member",
			),
			kind: DeleteOrdinary, article: ordinary, wantError: ErrArticleUnavailable,
		},
		{
			name: "ordinary disabled author",
			principal: session.Principal{
				SessionID: "session", UserID: deletionAuthorID,
				Role: "member", Status: "disabled",
			},
			kind: DeleteOrdinary, article: ordinary, wantError: ErrArticleForbidden,
		},
		{
			name: "violating active administrator", principal: deletionAdminPrincipal(),
			kind: DeleteViolating, article: violating,
		},
		{
			name: "violating active member", principal: deletionAuthorPrincipal(),
			kind: DeleteViolating, article: violating, wantError: ErrArticleForbidden,
		},
		{
			name: "violating restricted administrator",
			principal: session.Principal{
				SessionID: "session", UserID: deletionAdminID,
				Role: "admin", Status: "active", MustChangePassword: true,
			},
			kind: DeleteViolating, article: violating, wantError: ErrArticleForbidden,
		},
		{
			name: "orphan active administrator", principal: deletionAdminPrincipal(),
			kind: DeleteOrphan, article: orphan,
		},
		{
			name: "orphan active member", principal: deletionAuthorPrincipal(),
			kind: DeleteOrphan, article: orphan, wantError: ErrArticleForbidden,
		},
		{
			name: "orphan disabled administrator",
			principal: session.Principal{
				SessionID: "session", UserID: deletionAdminID,
				Role: "admin", Status: "disabled",
			},
			kind: DeleteOrphan, article: orphan, wantError: ErrArticleForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := authorizeActiveDeletion(test.principal, test.kind, test.article)
			if test.wantError == nil {
				if err != nil {
					t.Fatalf("authorizeActiveDeletion() error = %v", err)
				}
				return
			}
			if !errors.Is(err, test.wantError) {
				t.Fatalf("authorizeActiveDeletion() error = %v, want %v", err, test.wantError)
			}
		})
	}
}

func TestDeletionJournal_AWIREJRN03DurableIdempotentTombstoneContainsNoPII(t *testing.T) {
	directory := newJournalTestDir(t)
	key := DeletionJournalKey{
		ID: "journal-key", Version: "v1", Material: []byte("synthetic-test-key"),
	}
	journal, err := newTestFileArticleDeletionJournal(
		ArticleDeletionJournalOptions{
			Directory: directory,
			FileMode:  0o600,
		},
		DeletionJournalKeyring{Active: key, Retained: []DeletionJournalKey{key}},
	)
	if err != nil {
		t.Fatalf("NewFileArticleDeletionJournal() error = %v", err)
	}
	request := DeletionTombstoneRequest{
		ArticleID: deletionArticleID, ArticleVersion: 3,
		DeletionOperationID: deletionOperationID, DeletedAt: deletionNow,
	}

	first, err := journal.Append(context.Background(), request)
	if err != nil {
		t.Fatalf("Append(first) error = %v", err)
	}
	second, err := journal.Append(context.Background(), request)
	if err != nil {
		t.Fatalf("Append(second) error = %v", err)
	}
	if first != second || first.Sequence != 1 ||
		first.Namespace != ArticleDeletionNamespace ||
		first.Stage != deletionTombstoneStage {
		t.Fatalf("idempotent tombstone mismatch: first=%#v second=%#v", first, second)
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	var recordBytes []byte
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		recordBytes, err = os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}
	}
	if len(recordBytes) == 0 {
		t.Fatal("durable journal record was not written")
	}
	for _, forbidden := range []string{
		"title", "body", "content_digest", "content_hash", "email", "author",
	} {
		if strings.Contains(strings.ToLower(string(recordBytes)), forbidden) {
			t.Fatalf("journal record contains forbidden fragment %q: %s", forbidden, recordBytes)
		}
	}
	var decoded map[string]any
	if err := json.Unmarshal(recordBytes, &decoded); err != nil {
		t.Fatalf("decode journal record: %v", err)
	}
	if len(decoded) != 11 {
		t.Fatalf("journal field count = %d, want 11: %#v", len(decoded), decoded)
	}
}

func newTestFileArticleDeletionJournal(
	options ArticleDeletionJournalOptions,
	keyring DeletionJournalKeyring,
) (*FileArticleDeletionJournal, error) {
	return newFileArticleDeletionJournal(
		options,
		keyring,
		func(string) error { return nil },
	)
}

func newJournalTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if articleDeletionJournalPlatformSupported() {
		if err := os.Chmod(dir, 0o700); err != nil {
			t.Fatalf("chmod journal dir: %v", err)
		}
	}
	return dir
}

func TestDeletionJournal_AWIREJRN05JRN06HMACDomainAndKeyLifecycle(t *testing.T) {
	directory := newJournalTestDir(t)
	material := []byte("synthetic-test-key-material-000001")
	journal, err := newTestFileArticleDeletionJournal(
		ArticleDeletionJournalOptions{Directory: directory, FileMode: 0o600},
		DeletionJournalKeyring{Active: DeletionJournalKey{
			ID: "article-deletion-hmac", Version: "v1", Material: material,
		}},
	)
	if err != nil {
		t.Fatalf("newTestFileArticleDeletionJournal() error = %v", err)
	}
	record, err := journal.Append(context.Background(), DeletionTombstoneRequest{
		ArticleID: deletionArticleID, ArticleVersion: 3,
		DeletionOperationID: deletionOperationID, DeletedAt: deletionNow,
	})
	if err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	raw, err := json.Marshal(deletionMACPayloadFromModel(record))
	if err != nil {
		t.Fatalf("marshal HMAC payload: %v", err)
	}
	legacy := hmac.New(sha256.New, material)
	_, _ = legacy.Write(raw)
	if record.RecordHMAC == "hmac-sha256:"+hex.EncodeToString(legacy.Sum(nil)) {
		t.Fatal("Article tombstone accepted the legacy unscoped HMAC domain")
	}

	retained := journal.active.Material
	if err := journal.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	for _, value := range retained {
		if value != 0 {
			t.Fatal("Close() did not clear active key material")
		}
	}
}

func TestDeletionJournal_AWIREJRN01JRN03JRN04ProductionProvider(
	t *testing.T,
) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatalf("Chmod(root) error = %v", err)
	}
	keyring := DeletionJournalKeyring{Active: DeletionJournalKey{
		ID: "article-deletion-hmac", Version: "v1",
		Material: []byte("synthetic-test-key-material-000001"),
	}}
	first, err := NewProductionFileArticleDeletionJournal(root, 0o600, keyring)
	if !articleDeletionJournalPlatformSupported() {
		if !errors.Is(err, ErrDeletionJournalUnsupportedPlatform) || first != nil {
			t.Fatalf("unsupported provider result = %#v/%v", first, err)
		}
		return
	}
	if err != nil {
		t.Fatalf("NewProductionFileArticleDeletionJournal() error = %v", err)
	}
	active := first.active.Material
	second, secondErr := NewProductionFileArticleDeletionJournal(root, 0o600, keyring)
	if second != nil || !errors.Is(secondErr, ErrDeletionJournalBusy) {
		t.Fatalf("second provider result = %#v/%v, want busy", second, secondErr)
	}
	for _, path := range []string{
		filepath.Join(root, "article-deletion"),
		filepath.Join(root, "article-deletion", "v1"),
		filepath.Join(root, "article-deletion", "v1", "records"),
	} {
		info, statErr := os.Lstat(path)
		if statErr != nil || !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
			t.Fatalf("private directory %q = %#v/%v", filepath.Base(path), info, statErr)
		}
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	for _, value := range active {
		if value != 0 {
			t.Fatal("provider Close() retained active key material")
		}
	}
	reopened, err := NewProductionFileArticleDeletionJournal(root, 0o600, keyring)
	if err != nil {
		t.Fatalf("provider reopen error = %v", err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatalf("reopened Close() error = %v", err)
	}
}

func TestDeletionJournal_AWIREJRN02RejectsUnsafePathComponents(t *testing.T) {
	base := t.TempDir()
	ancestor := filepath.Join(base, "ancestor")
	root := filepath.Join(ancestor, "root")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.Chmod(ancestor, 0o700); err != nil {
		t.Fatalf("Chmod(ancestor) error = %v", err)
	}
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatalf("Chmod(root) error = %v", err)
	}

	tests := []struct {
		name        string
		replace     string
		replacement fs.FileMode
	}{
		{name: "ancestor symlink", replace: ancestor, replacement: os.ModeDir | os.ModeSymlink | 0o700},
		{name: "direct symlink", replace: root, replacement: os.ModeDir | os.ModeSymlink | 0o700},
		{name: "wrong object", replace: root, replacement: 0o600},
		{name: "group permission", replace: root, replacement: os.ModeDir | 0o750},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateArticleDeletionPathComponents(
				root,
				true,
				func(path string) (fs.FileInfo, error) {
					info, statErr := os.Lstat(path)
					if statErr != nil || path != test.replace {
						return info, statErr
					}
					return deletionFileInfoWithMode{FileInfo: info, mode: test.replacement}, nil
				},
			)
			if !errors.Is(err, ErrDeletionJournalCorrupt) {
				t.Fatalf("validation error = %v, want ErrDeletionJournalCorrupt", err)
			}
			if strings.Contains(err.Error(), root) {
				t.Fatalf("validation error leaked root: %v", err)
			}
		})
	}
	if err := validateArticleDeletionPathComponents(
		root,
		true,
		func(path string) (fs.FileInfo, error) {
			info, statErr := os.Lstat(path)
			if statErr != nil || path != root {
				return info, statErr
			}
			return deletionFileInfoWithMode{
				FileInfo: info,
				mode:     os.ModeDir | 0o700,
			}, nil
		},
	); err != nil {
		t.Fatalf("valid path rejected: %v", err)
	}
	for _, name := range []string{"..", "../escape", `..\escape`, "nested/escape"} {
		if _, err := articleDeletionChildPath(root, name); !errors.Is(
			err,
			ErrDeletionJournalCorrupt,
		) {
			t.Fatalf("child name %q error = %v, want corruption", name, err)
		}
	}
}

func TestDeletionFinalizer_AWIREWKR07ClassifiesActiveLeaseCompetition(t *testing.T) {
	fixture := newPermanentDeletionFixture(t)
	operation, _ := fixture.prepareJournaledDeletion(t)
	otherWorker := "other-finalizer"
	lockedUntil := deletionNow.Add(time.Minute)
	operation.Status = DeletionOperationFinalizing
	operation.ClaimedBy = &otherWorker
	operation.LockedUntil = &lockedUntil
	fixture.repository.operation = &operation

	err := fixture.realFinalizer.Finalize(
		context.Background(),
		deletionOperationID,
		deletionTraceID,
	)

	if !errors.Is(err, ErrDeletionInProgress) ||
		!errors.Is(err, errDeletionLeaseCompetition) {
		t.Fatalf("Finalize() error = %v, want active lease competition", err)
	}
	if fixture.repository.deleteCalls.Load() != 0 {
		t.Fatal("active lease competition reached physical deletion")
	}
}

type deletionFileInfoWithMode struct {
	fs.FileInfo
	mode fs.FileMode
}

func (info deletionFileInfoWithMode) Mode() fs.FileMode {
	return info.mode
}

func (info deletionFileInfoWithMode) IsDir() bool {
	return info.mode.IsDir()
}

func TestDeletionJournal_AWIREJRN07FsyncFailureDoesNotPublishRecord(t *testing.T) {
	directory := newJournalTestDir(t)
	key := DeletionJournalKey{
		ID: "journal-key", Version: "v1", Material: []byte("synthetic-test-key"),
	}
	journal, err := newTestFileArticleDeletionJournal(
		ArticleDeletionJournalOptions{Directory: directory, FileMode: 0o600},
		DeletionJournalKeyring{Active: key, Retained: []DeletionJournalKey{key}},
	)
	if err != nil {
		t.Fatalf("NewFileArticleDeletionJournal() error = %v", err)
	}
	journal.syncFile = func(*os.File) error {
		return errors.New("synthetic fsync failure")
	}

	_, err = journal.Append(context.Background(), DeletionTombstoneRequest{
		ArticleID: deletionArticleID, ArticleVersion: 3,
		DeletionOperationID: deletionOperationID, DeletedAt: deletionNow,
	})
	if err == nil {
		t.Fatal("Append() error = nil, want fsync failure")
	}
	entries, readErr := os.ReadDir(directory)
	if readErr != nil {
		t.Fatalf("ReadDir() error = %v", readErr)
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".json" {
			t.Fatalf("fsync failure published journal record %q", entry.Name())
		}
	}
}

func TestDeletionJournal_AWIREJRN07SanitizesOperationAndCloseFailures(t *testing.T) {
	const (
		pathCanary = `C:\sensitive\article-journal`
		keyCanary  = "key-material-canary"
		bodyCanary = "private-record-body-canary"
	)
	key := DeletionJournalKey{
		ID: "journal-key", Version: "v1", Material: []byte("synthetic-test-key"),
	}

	t.Run("operation", func(t *testing.T) {
		journal, err := newTestFileArticleDeletionJournal(
			ArticleDeletionJournalOptions{Directory: newJournalTestDir(t), FileMode: 0o600},
			DeletionJournalKeyring{Active: key},
		)
		if err != nil {
			t.Fatalf("newTestFileArticleDeletionJournal() error = %v", err)
		}
		journal.syncFile = func(*os.File) error {
			return &os.PathError{
				Op:   "sync",
				Path: pathCanary,
				Err:  errors.New(keyCanary + bodyCanary),
			}
		}

		_, err = journal.Append(context.Background(), DeletionTombstoneRequest{
			ArticleID: deletionArticleID, ArticleVersion: 3,
			DeletionOperationID: deletionOperationID, DeletedAt: deletionNow,
		})
		requireSanitizedDeletionJournalError(
			t,
			err,
			pathCanary,
			keyCanary,
			bodyCanary,
		)
	})

	t.Run("close", func(t *testing.T) {
		journal, err := newTestFileArticleDeletionJournal(
			ArticleDeletionJournalOptions{Directory: newJournalTestDir(t), FileMode: 0o600},
			DeletionJournalKeyring{Active: key},
		)
		if err != nil {
			t.Fatalf("newTestFileArticleDeletionJournal() error = %v", err)
		}
		journal.writerLock = deletionFileLockStub{err: &os.PathError{
			Op:   "close",
			Path: pathCanary,
			Err:  errors.New(keyCanary + bodyCanary),
		}}

		requireSanitizedDeletionJournalError(
			t,
			journal.Close(),
			pathCanary,
			keyCanary,
			bodyCanary,
		)
	})
}

func requireSanitizedDeletionJournalError(
	t *testing.T,
	err error,
	canaries ...string,
) {
	t.Helper()
	if !errors.Is(err, errDeletionJournalFilesystem) {
		t.Fatalf("error = %v, want filesystem category", err)
	}
	for _, canary := range canaries {
		if strings.Contains(err.Error(), canary) {
			t.Fatalf("error leaked canary %q: %v", canary, err)
		}
	}
}

type deletionFileLockStub struct {
	err error
}

func (stub deletionFileLockStub) Close() error {
	return stub.err
}

func TestDeletionJournal_AWIREJRN07DirectoryFsyncFailureRequiresRetry(t *testing.T) {
	directory := newJournalTestDir(t)
	key := DeletionJournalKey{
		ID: "journal-key", Version: "v1", Material: []byte("synthetic-test-key"),
	}
	var syncCalls atomic.Int64
	journal, err := newFileArticleDeletionJournal(
		ArticleDeletionJournalOptions{Directory: directory, FileMode: 0o600},
		DeletionJournalKeyring{Active: key, Retained: []DeletionJournalKey{key}},
		func(string) error {
			if syncCalls.Add(1) == 1 {
				return errors.New("synthetic directory fsync failure")
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("newFileArticleDeletionJournal() error = %v", err)
	}
	request := DeletionTombstoneRequest{
		ArticleID: deletionArticleID, ArticleVersion: 3,
		DeletionOperationID: deletionOperationID, DeletedAt: deletionNow,
	}

	if _, err := journal.Append(context.Background(), request); err == nil {
		t.Fatal("Append(first) error = nil, want directory fsync failure")
	}
	record, err := journal.Append(context.Background(), request)
	if err != nil {
		t.Fatalf("Append(retry) error = %v", err)
	}
	if record.Sequence != 1 || syncCalls.Load() != 2 {
		t.Fatalf("retry record=%#v sync calls=%d, want sequence 1 and two sync attempts",
			record, syncCalls.Load())
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	var records int
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".json" {
			records++
		}
	}
	if records != 1 {
		t.Fatalf("journal records = %d, want 1", records)
	}
}

func TestDeletionJournal_AWIREJRN07DirectoryFsyncBlocksIndependentFinalizer(
	t *testing.T,
) {
	directory := newJournalTestDir(t)
	key := DeletionJournalKey{
		ID: "key", Version: "v1", Material: []byte("synthetic-test-key"),
	}
	writer, err := newFileArticleDeletionJournal(
		ArticleDeletionJournalOptions{Directory: directory, FileMode: 0o600},
		DeletionJournalKeyring{Active: key, Retained: []DeletionJournalKey{key}},
		func(string) error {
			return errors.New("synthetic writer directory fsync failure")
		},
	)
	if err != nil {
		t.Fatalf("new writer journal: %v", err)
	}
	_, err = writer.Append(context.Background(), DeletionTombstoneRequest{
		ArticleID: deletionArticleID, ArticleVersion: 3,
		DeletionOperationID: deletionOperationID, DeletedAt: deletionNow,
	})
	if err == nil {
		t.Fatal("Append() error = nil, want writer directory fsync failure")
	}
	if _, err := os.Stat(filepath.Join(directory, deletionRecordName(1))); err != nil {
		t.Fatalf("failed Append did not expose the regression record: %v", err)
	}

	readerSyncErr := errors.New("synthetic reader directory fsync failure")
	var readerCanProve atomic.Bool
	reader, err := newFileArticleDeletionJournal(
		ArticleDeletionJournalOptions{Directory: directory, FileMode: 0o600},
		DeletionJournalKeyring{Active: key, Retained: []DeletionJournalKey{key}},
		func(string) error {
			if !readerCanProve.Load() {
				return readerSyncErr
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("new independent reader journal: %v", err)
	}
	fixture := newPermanentDeletionFixture(t)
	operation := DeletionOperation{
		ID: deletionOperationID, ArticleID: deletionArticleID, ArticleVersion: 3,
		DeleteKind: DeleteOrdinary, ActorUserID: stringPointer(deletionAuthorID),
		ActorRole: stringPointer("member"), Status: DeletionOperationPendingJournal,
		StartedAt: deletionNow,
	}
	fixture.repository.operation = &operation
	fixture.repository.article.DeletionState = DeletionPending
	fixture.repository.article.DeletionOperationID = stringPointer(operation.ID)
	fixture.repository.article.Version = operation.ArticleVersion + 1
	fixture.realFinalizer.dependencies.Journal = reader

	err = fixture.realFinalizer.Finalize(
		context.Background(),
		operation.ID,
		deletionTraceID,
	)

	if !errors.Is(err, errDeletionJournalFilesystem) ||
		strings.Contains(err.Error(), readerSyncErr.Error()) {
		t.Fatalf("Finalize() error = %v, want reader durability proof failure", err)
	}
	if fixture.repository.article == nil ||
		fixture.repository.operation == nil ||
		fixture.repository.operation.Status != DeletionOperationPendingJournal ||
		fixture.repository.claimCalls.Load() != 0 ||
		fixture.repository.deleteCalls.Load() != 0 ||
		fixture.repository.completeCalls.Load() != 0 ||
		len(fixture.auditor.events) != 0 {
		t.Fatalf("unproved tombstone advanced deletion: article=%#v operation=%#v claim=%d delete=%d complete=%d audits=%d",
			fixture.repository.article, fixture.repository.operation,
			fixture.repository.claimCalls.Load(),
			fixture.repository.deleteCalls.Load(),
			fixture.repository.completeCalls.Load(),
			len(fixture.auditor.events))
	}

	readerCanProve.Store(true)
	if err := fixture.realFinalizer.Finalize(
		context.Background(),
		operation.ID,
		deletionTraceID,
	); err != nil {
		t.Fatalf("Finalize(retry) error = %v", err)
	}
	if fixture.repository.article != nil ||
		fixture.repository.operation == nil ||
		fixture.repository.operation.Status != DeletionOperationCompleted ||
		len(fixture.auditor.events) != 1 {
		t.Fatalf("durability retry did not converge: article=%#v operation=%#v audits=%d",
			fixture.repository.article, fixture.repository.operation,
			len(fixture.auditor.events))
	}
}

func TestDeletionJournal_AWIREJRN07RejectsTrailingRecordData(t *testing.T) {
	directory := newJournalTestDir(t)
	key := DeletionJournalKey{
		ID: "journal-key", Version: "v1", Material: []byte("synthetic-test-key"),
	}
	journal, err := newTestFileArticleDeletionJournal(
		ArticleDeletionJournalOptions{Directory: directory, FileMode: 0o600},
		DeletionJournalKeyring{Active: key, Retained: []DeletionJournalKey{key}},
	)
	if err != nil {
		t.Fatalf("NewFileArticleDeletionJournal() error = %v", err)
	}
	if _, err := journal.Append(context.Background(), DeletionTombstoneRequest{
		ArticleID: deletionArticleID, ArticleVersion: 3,
		DeletionOperationID: deletionOperationID, DeletedAt: deletionNow,
	}); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	path := filepath.Join(directory, deletionRecordName(1))
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	if _, err := file.WriteString(`{}`); err != nil {
		_ = file.Close()
		t.Fatalf("append trailing data: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close corrupted record: %v", err)
	}

	_, err = newTestFileArticleDeletionJournal(
		ArticleDeletionJournalOptions{Directory: directory, FileMode: 0o600},
		DeletionJournalKeyring{Active: key, Retained: []DeletionJournalKey{key}},
	)
	if !errors.Is(err, ErrDeletionJournalCorrupt) {
		t.Fatalf("NewFileArticleDeletionJournal() error = %v, want corrupt", err)
	}
}

func TestDeletionJournal_AWIREJRN07ProductionWindowsFailsClosed(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows production fail-closed contract")
	}
	directory := t.TempDir()
	key := DeletionJournalKey{
		ID: "journal-key", Version: "v1", Material: []byte("synthetic-test-key"),
	}

	_, err := NewFileArticleDeletionJournal(
		ArticleDeletionJournalOptions{Directory: directory, FileMode: 0o600},
		DeletionJournalKeyring{Active: key, Retained: []DeletionJournalKey{key}},
	)

	if !errors.Is(err, ErrDeletionJournalUnsupportedPlatform) {
		t.Fatalf("NewFileArticleDeletionJournal() error = %v, want unsupported platform", err)
	}
	entries, readErr := os.ReadDir(directory)
	if readErr != nil {
		t.Fatalf("ReadDir() error = %v", readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("unsupported production constructor created files: %v", entries)
	}
}

func TestDeletionJournal_StaleExistenceLockDoesNotBlockAppend(t *testing.T) {
	directory := newJournalTestDir(t)
	key := DeletionJournalKey{
		ID: "journal-key", Version: "v1", Material: []byte("synthetic-test-key"),
	}
	if err := os.WriteFile(
		filepath.Join(directory, ".article-deletion.lock"),
		[]byte("crashed writer"),
		0o600,
	); err != nil {
		t.Fatalf("WriteFile(stale lock) error = %v", err)
	}
	journal, err := newTestFileArticleDeletionJournal(
		ArticleDeletionJournalOptions{Directory: directory, FileMode: 0o600},
		DeletionJournalKeyring{Active: key, Retained: []DeletionJournalKey{key}},
	)
	if err != nil {
		t.Fatalf("NewFileArticleDeletionJournal() error = %v", err)
	}

	_, err = journal.Append(context.Background(), DeletionTombstoneRequest{
		ArticleID: deletionArticleID, ArticleVersion: 3,
		DeletionOperationID: deletionOperationID, DeletedAt: deletionNow,
	})
	if err != nil {
		t.Fatalf("Append() error = %v, stale existence lock must not block", err)
	}
}

func TestDeletionJournal_ConcurrentIndependentWritersConverge(t *testing.T) {
	directory := newJournalTestDir(t)
	key := DeletionJournalKey{
		ID: "journal-key", Version: "v1", Material: []byte("synthetic-test-key"),
	}
	if err := os.WriteFile(
		filepath.Join(directory, ".article-deletion.tmp-crashed"),
		[]byte("partial writer state"),
		0o600,
	); err != nil {
		t.Fatalf("WriteFile(stale temp) error = %v", err)
	}
	first, err := newTestFileArticleDeletionJournal(
		ArticleDeletionJournalOptions{Directory: directory, FileMode: 0o600},
		DeletionJournalKeyring{Active: key, Retained: []DeletionJournalKey{key}},
	)
	if err != nil {
		t.Fatalf("new first journal: %v", err)
	}
	second, err := newTestFileArticleDeletionJournal(
		ArticleDeletionJournalOptions{Directory: directory, FileMode: 0o600},
		DeletionJournalKeyring{Active: key, Retained: []DeletionJournalKey{key}},
	)
	if err != nil {
		t.Fatalf("new second journal: %v", err)
	}
	requests := []DeletionTombstoneRequest{
		{
			ArticleID: deletionArticleID, ArticleVersion: 3,
			DeletionOperationID: deletionOperationID, DeletedAt: deletionNow,
		},
		{
			ArticleID: "10000000-0000-4000-8000-000000000102", ArticleVersion: 7,
			DeletionOperationID: "30000000-0000-4000-8000-000000000102",
			DeletedAt:           deletionNow.Add(time.Second),
		},
	}
	journals := []*FileArticleDeletionJournal{first, second}
	results := make(chan ArticleDeletionTombstone, len(journals))
	errs := make(chan error, len(journals))
	var wait sync.WaitGroup
	for index := range journals {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			record, appendErr := journals[index].Append(context.Background(), requests[index])
			results <- record
			errs <- appendErr
		}(index)
	}
	wait.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent Append() error = %v", err)
		}
	}
	sequences := map[int64]bool{}
	for record := range results {
		sequences[record.Sequence] = true
	}
	if len(sequences) != 2 || !sequences[1] || !sequences[2] {
		t.Fatalf("sequences = %v, want unique contiguous 1 and 2", sequences)
	}
	reopened, err := newTestFileArticleDeletionJournal(
		ArticleDeletionJournalOptions{Directory: directory, FileMode: 0o600},
		DeletionJournalKeyring{Active: key, Retained: []DeletionJournalKey{key}},
	)
	if err != nil {
		t.Fatalf("reopen verified journal: %v", err)
	}
	for _, request := range requests {
		if _, err := reopened.FindByOperation(
			context.Background(),
			request.DeletionOperationID,
		); err != nil {
			t.Fatalf("FindByOperation(%s) error = %v",
				request.DeletionOperationID, err)
		}
	}
}

func TestDeletionFinalizer_AWIREWKR02ReconcilesPendingJournalWithoutHTTPRetry(t *testing.T) {
	fixture := newPermanentDeletionFixture(t)
	operation := DeletionOperation{
		ID: deletionOperationID, ArticleID: deletionArticleID, ArticleVersion: 3,
		DeleteKind: DeleteOrdinary, ActorUserID: stringPointer(deletionAuthorID),
		ActorRole: stringPointer("member"), Status: DeletionOperationPendingJournal,
		StartedAt: deletionNow,
	}
	_, err := fixture.journal.Append(
		context.Background(),
		DeletionTombstoneRequest{
			ArticleID: operation.ArticleID, ArticleVersion: operation.ArticleVersion,
			DeletionOperationID: operation.ID, DeletedAt: operation.StartedAt,
		},
	)
	if err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	fixture.repository.operation = &operation
	fixture.repository.article.DeletionState = DeletionPending
	fixture.repository.article.DeletionOperationID = stringPointer(operation.ID)
	fixture.repository.article.Version = 4

	err = fixture.realFinalizer.Finalize(
		context.Background(),
		operation.ID,
		deletionTraceID,
	)

	if err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}
	if fixture.repository.article != nil ||
		fixture.repository.operation == nil ||
		fixture.repository.operation.Status != DeletionOperationCompleted {
		t.Fatalf("pending reconciliation did not converge: article=%#v operation=%#v",
			fixture.repository.article, fixture.repository.operation)
	}
}

func TestDeletionFinalizer_RejectsTamperedJournalRecordBeforeClaim(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*deletionDiskRecord)
	}{
		{name: "record hmac", mutate: func(record *deletionDiskRecord) {
			record.RecordHMAC = "hmac-sha256:forged"
		}},
		{name: "previous hash", mutate: func(record *deletionDiskRecord) {
			record.PreviousRecordHash = "sha256:forged"
		}},
		{name: "deleted at", mutate: func(record *deletionDiskRecord) {
			record.DeletedAt = deletionNow.Add(time.Second).Format(time.RFC3339Nano)
		}},
		{name: "sequence", mutate: func(record *deletionDiskRecord) {
			record.Sequence = 2
		}},
		{name: "key id", mutate: func(record *deletionDiskRecord) {
			record.KeyID = "unknown-key"
		}},
		{name: "key version", mutate: func(record *deletionDiskRecord) {
			record.KeyVersion = "unknown-version"
		}},
		{name: "namespace", mutate: func(record *deletionDiskRecord) {
			record.Namespace = "article-deletion/forged"
		}},
		{name: "stage", mutate: func(record *deletionDiskRecord) {
			record.Stage = "forged"
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newPermanentDeletionFixture(t)
			operation, _ := fixture.prepareJournaledDeletion(t)
			directory := newJournalTestDir(t)
			key := DeletionJournalKey{
				ID: "key", Version: "v1", Material: []byte("synthetic-test-key"),
			}
			journal, err := newTestFileArticleDeletionJournal(
				ArticleDeletionJournalOptions{Directory: directory, FileMode: 0o600},
				DeletionJournalKeyring{Active: key, Retained: []DeletionJournalKey{key}},
			)
			if err != nil {
				t.Fatalf("new journal: %v", err)
			}
			if _, err := journal.Append(context.Background(), DeletionTombstoneRequest{
				ArticleID: operation.ArticleID, ArticleVersion: operation.ArticleVersion,
				DeletionOperationID: operation.ID, DeletedAt: operation.StartedAt,
			}); err != nil {
				t.Fatalf("Append() error = %v", err)
			}
			path := filepath.Join(directory, deletionRecordName(1))
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}
			var record deletionDiskRecord
			if err := json.Unmarshal(raw, &record); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			test.mutate(&record)
			raw, err = json.Marshal(record)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if err := os.WriteFile(path, raw, 0o600); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}
			fixture.realFinalizer.dependencies.Journal = journal

			err = fixture.realFinalizer.Finalize(
				context.Background(),
				operation.ID,
				deletionTraceID,
			)

			if !errors.Is(err, ErrDeletionJournalCorrupt) {
				t.Fatalf("Finalize() error = %v, want journal corrupt", err)
			}
			if fixture.repository.article == nil ||
				fixture.repository.operation == nil ||
				fixture.repository.operation.Status != DeletionOperationJournaled ||
				fixture.repository.claimCalls.Load() != 0 ||
				fixture.repository.deleteCalls.Load() != 0 ||
				fixture.repository.completeCalls.Load() != 0 ||
				len(fixture.auditor.events) != 0 {
				t.Fatalf("tampered tombstone changed deletion state: article=%#v operation=%#v claim=%d delete=%d complete=%d audits=%d",
					fixture.repository.article, fixture.repository.operation,
					fixture.repository.claimCalls.Load(),
					fixture.repository.deleteCalls.Load(),
					fixture.repository.completeCalls.Load(),
					len(fixture.auditor.events))
			}
		})
	}
}

func TestDeletionFinalizer_PhysicalDeleteThreeStateContract(t *testing.T) {
	tests := []struct {
		name      string
		prepare   func(*permanentDeletionFixture)
		wantError error
	}{
		{name: "deleted", prepare: func(*permanentDeletionFixture) {}},
		{name: "already absent", prepare: func(fixture *permanentDeletionFixture) {
			fixture.repository.article = nil
		}},
		{name: "active high version", prepare: func(fixture *permanentDeletionFixture) {
			fixture.repository.article.DeletionState = DeletionActive
			fixture.repository.article.Version = 99
		}, wantError: ErrDeletionArticleMismatch},
		{name: "pending high version", prepare: func(fixture *permanentDeletionFixture) {
			fixture.repository.article.Version = 99
		}, wantError: ErrDeletionArticleMismatch},
		{name: "wrong operation binding", prepare: func(fixture *permanentDeletionFixture) {
			fixture.repository.article.DeletionOperationID = stringPointer(
				"30000000-0000-4000-8000-000000000199",
			)
		}, wantError: ErrDeletionArticleMismatch},
		{name: "missing operation binding", prepare: func(fixture *permanentDeletionFixture) {
			fixture.repository.article.DeletionOperationID = nil
		}, wantError: ErrDeletionArticleMismatch},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newPermanentDeletionFixture(t)
			operation, _ := fixture.prepareJournaledDeletion(t)
			test.prepare(&fixture)

			err := fixture.realFinalizer.Finalize(
				context.Background(),
				operation.ID,
				deletionTraceID,
			)

			if !errors.Is(err, test.wantError) {
				t.Fatalf("Finalize() error = %v, want %v", err, test.wantError)
			}
			if test.wantError == nil {
				if fixture.repository.article != nil ||
					fixture.repository.operation == nil ||
					fixture.repository.operation.Status != DeletionOperationCompleted ||
					len(fixture.auditor.events) != 1 {
					t.Fatalf("successful delete did not complete: article=%#v operation=%#v audits=%d",
						fixture.repository.article, fixture.repository.operation,
						len(fixture.auditor.events))
				}
				return
			}
			if fixture.repository.article == nil ||
				fixture.repository.operation == nil ||
				fixture.repository.operation.Status == DeletionOperationCompleted ||
				fixture.repository.completeCalls.Load() != 0 ||
				len(fixture.auditor.events) != 0 {
				t.Fatalf("present mismatch completed deletion: article=%#v operation=%#v complete=%d audits=%d",
					fixture.repository.article, fixture.repository.operation,
					fixture.repository.completeCalls.Load(), len(fixture.auditor.events))
			}
		})
	}
}

func TestDeletionFinalizer_AWIREWKR09DuplicateConcurrentCallsCompleteOnce(t *testing.T) {
	fixture := newPermanentDeletionFixture(t)
	operation, _ := fixture.prepareJournaledDeletion(t)

	var wait sync.WaitGroup
	errorsSeen := make(chan error, 2)
	for index := 0; index < 2; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			errorsSeen <- fixture.realFinalizer.Finalize(
				context.Background(),
				operation.ID,
				deletionTraceID,
			)
		}()
	}
	wait.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		if err != nil && !errors.Is(err, ErrDeletionInProgress) {
			t.Fatalf("Finalize() error = %v", err)
		}
	}
	if err := fixture.realFinalizer.Finalize(
		context.Background(),
		operation.ID,
		deletionTraceID,
	); err != nil {
		t.Fatalf("Finalize(retry) error = %v", err)
	}

	if fixture.repository.article != nil {
		t.Fatal("article remains after finalization")
	}
	if fixture.repository.operation == nil ||
		fixture.repository.operation.Status != DeletionOperationCompleted {
		t.Fatalf("operation = %#v, want completed", fixture.repository.operation)
	}
	if len(fixture.auditor.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(fixture.auditor.events))
	}
}

func TestDeletionFinalizer_AWIREWKR02WKR09ConcurrentPendingReconciliation(t *testing.T) {
	fixture := newPermanentDeletionFixture(t)
	fixture.transactor.firstTransactionLocked = make(chan struct{})
	fixture.transactor.secondTransactionAttempted = make(chan struct{})
	fixture.transactor.releaseFirstTransaction = make(chan struct{})
	operation := DeletionOperation{
		ID: deletionOperationID, ArticleID: deletionArticleID, ArticleVersion: 3,
		DeleteKind: DeleteOrdinary, ActorUserID: stringPointer(deletionAuthorID),
		ActorRole: stringPointer("member"), Status: DeletionOperationPendingJournal,
		StartedAt: deletionNow,
	}
	if _, err := fixture.journal.Append(
		context.Background(),
		DeletionTombstoneRequest{
			ArticleID: operation.ArticleID, ArticleVersion: operation.ArticleVersion,
			DeletionOperationID: operation.ID, DeletedAt: operation.StartedAt,
		},
	); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	fixture.repository.operation = &operation
	fixture.repository.article.DeletionState = DeletionPending
	fixture.repository.article.DeletionOperationID = stringPointer(operation.ID)
	fixture.repository.article.Version = operation.ArticleVersion + 1

	var wait sync.WaitGroup
	ready := make(chan struct{}, 2)
	start := make(chan struct{})
	errorsSeen := make(chan error, 2)
	for index := 0; index < 2; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			ready <- struct{}{}
			<-start
			errorsSeen <- fixture.realFinalizer.Finalize(
				context.Background(),
				operation.ID,
				deletionTraceID,
			)
		}()
	}
	<-ready
	<-ready
	close(start)
	var releaseOnce sync.Once
	releaseFirst := func() {
		releaseOnce.Do(func() {
			close(fixture.transactor.releaseFirstTransaction)
		})
	}
	defer releaseFirst()
	select {
	case <-fixture.transactor.firstTransactionLocked:
	case <-time.After(time.Second):
		releaseFirst()
		t.Fatal("first finalizer did not acquire the transaction lock")
	}
	select {
	case <-fixture.transactor.secondTransactionAttempted:
	case <-time.After(time.Second):
		releaseFirst()
		t.Fatal("second finalizer did not contend for the transaction lock")
	}
	releaseFirst()
	wait.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		if err != nil && !errors.Is(err, ErrDeletionInProgress) {
			t.Fatalf("Finalize() error = %v", err)
		}
	}
	if err := fixture.realFinalizer.Finalize(
		context.Background(),
		operation.ID,
		deletionTraceID,
	); err != nil {
		t.Fatalf("Finalize(retry) error = %v", err)
	}
	if fixture.repository.article != nil ||
		fixture.repository.operation == nil ||
		fixture.repository.operation.Status != DeletionOperationCompleted ||
		fixture.repository.markJournaledCalls.Load() != 1 ||
		fixture.repository.deleteCalls.Load() != 1 ||
		fixture.repository.completeCalls.Load() != 1 ||
		len(fixture.auditor.events) != 1 {
		t.Fatalf("pending reconciliation did not converge once: article=%#v operation=%#v mark=%d delete=%d complete=%d audits=%d",
			fixture.repository.article, fixture.repository.operation,
			fixture.repository.markJournaledCalls.Load(),
			fixture.repository.deleteCalls.Load(),
			fixture.repository.completeCalls.Load(),
			len(fixture.auditor.events))
	}
}

func TestDeletionFinalizer_AWIREWKR01ConvergeAppendsMissingTombstone(t *testing.T) {
	fixture := newPermanentDeletionFixture(t)
	operation := DeletionOperation{
		ID: deletionOperationID, ArticleID: deletionArticleID, ArticleVersion: 3,
		DeleteKind: DeleteOrdinary, ActorUserID: stringPointer(deletionAuthorID),
		ActorRole: stringPointer("member"), Status: DeletionOperationPendingJournal,
		StartedAt: deletionNow,
	}
	fixture.repository.operation = &operation
	fixture.repository.article.DeletionState = DeletionPending
	fixture.repository.article.DeletionOperationID = stringPointer(operation.ID)
	fixture.repository.article.Version = operation.ArticleVersion + 1

	if err := fixture.realFinalizer.ConvergeDeletionOperation(
		context.Background(),
		operation.ID,
		deletionTraceID,
	); err != nil {
		t.Fatalf("ConvergeDeletionOperation() error = %v", err)
	}
	if fixture.repository.article != nil ||
		fixture.repository.operation == nil ||
		fixture.repository.operation.Status != DeletionOperationCompleted ||
		fixture.repository.markJournaledCalls.Load() != 1 ||
		fixture.repository.deleteCalls.Load() != 1 ||
		fixture.repository.completeCalls.Load() != 1 {
		t.Fatalf(
			"convergence result: article=%#v operation=%#v mark=%d delete=%d complete=%d",
			fixture.repository.article,
			fixture.repository.operation,
			fixture.repository.markJournaledCalls.Load(),
			fixture.repository.deleteCalls.Load(),
			fixture.repository.completeCalls.Load(),
		)
	}
	if len(fixture.journal.records) != 1 {
		t.Fatalf("journal records = %d, want 1", len(fixture.journal.records))
	}
}

type permanentDeletionFixture struct {
	repository    *deletionRepositoryStub
	transactor    *deletionTransactorStub
	journal       *deletionJournalStub
	auditor       *deletionAuditorStub
	realFinalizer *DeletionFinalizer
	finalizer     *deletionFinalizerStub
	coordinator   *PermanentDeletionCoordinator
}

func newPermanentDeletionFixture(t *testing.T) permanentDeletionFixture {
	t.Helper()
	article := deletionTestArticle()
	repository := &deletionRepositoryStub{article: &article}
	transactor := &deletionTransactorStub{}
	journal := &deletionJournalStub{}
	auditor := &deletionAuditorStub{}
	transactor.proofRepository, transactor.proofAuditor = repository, auditor
	realFinalizer, err := NewDeletionFinalizer(DeletionFinalizerDependencies{
		Transactions:  transactor,
		Repository:    repository,
		Journal:       journal,
		Auditor:       auditor,
		AuditIDs:      deletionIDStub{id: deletionAuditID},
		Clock:         deletionClockStub{now: deletionNow.Add(time.Second)},
		WorkerID:      "finalizer-test",
		LeaseDuration: time.Minute,
		Identity:      deletionIdentityStub{},
	})
	if err != nil {
		t.Fatalf("NewDeletionFinalizer() error = %v", err)
	}
	finalizer := &deletionFinalizerStub{delegate: realFinalizer}
	coordinator, err := NewPermanentDeletionCoordinator(PermanentDeletionDependencies{
		Transactions: transactor,
		Repository:   repository,
		Journal:      journal,
		Finalizer:    finalizer,
		OperationIDs: deletionIDStub{id: deletionOperationID},
		Clock:        deletionClockStub{now: deletionNow},
	})
	if err != nil {
		t.Fatalf("NewPermanentDeletionCoordinator() error = %v", err)
	}
	return permanentDeletionFixture{
		repository: repository, transactor: transactor, journal: journal,
		auditor: auditor, realFinalizer: realFinalizer, finalizer: finalizer,
		coordinator: coordinator,
	}
}

func (f permanentDeletionFixture) handler(t *testing.T) *Handler {
	t.Helper()
	service, err := NewCommandService(CommandDependencies{
		Transactions: f.transactor,
		Repository:   &createArticleRepositoryStub{},
		IDs:          createArticleIDStub{id: deletionArticleID},
		Clock:        deletionClockStub{now: deletionNow},
		Deletion:     f.coordinator,
	})
	if err != nil {
		t.Fatalf("NewCommandService() error = %v", err)
	}
	logger, err := NewSlogArticleResponseWriteLogger(
		slog.New(slog.NewJSONHandler(io.Discard, nil)),
		"han-server",
	)
	if err != nil {
		t.Fatalf("NewSlogArticleResponseWriteLogger() error = %v", err)
	}
	handler, err := NewHandler(service, logger, service)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	return handler
}

func (f permanentDeletionFixture) serve(
	t *testing.T,
	handler http.HandlerFunc,
	principal session.Principal,
	response http.ResponseWriter,
	request *http.Request,
) {
	t.Helper()
	authorizer, err := platform.NewHTTPAuthorizer(
		createArticlePrincipalResolver{principal: principal},
		func() time.Time { return deletionNow },
	)
	if err != nil {
		t.Fatalf("NewHTTPAuthorizer() error = %v", err)
	}
	request.Header.Set(platform.CSRFHeaderName, "csrf")
	request.AddCookie(&http.Cookie{
		Name: platform.SessionCookieName, Value: "synthetic-session",
	})
	protected := authorizer.Protect(session.AccessGeneral, handler)
	platform.TraceMiddleware(protected).ServeHTTP(response, request)
}

func (f permanentDeletionFixture) prepareJournaledDeletion(
	t *testing.T,
) (DeletionOperation, ArticleDeletionTombstone) {
	t.Helper()
	tombstone, err := f.journal.Append(
		context.Background(),
		DeletionTombstoneRequest{
			ArticleID: deletionArticleID, ArticleVersion: 3,
			DeletionOperationID: deletionOperationID, DeletedAt: deletionNow,
		},
	)
	if err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	operation := DeletionOperation{
		ID: deletionOperationID, ArticleID: deletionArticleID, ArticleVersion: 3,
		DeleteKind: DeleteOrdinary, ActorUserID: stringPointer(deletionAuthorID),
		ActorRole: stringPointer("member"), Status: DeletionOperationJournaled,
		TombstoneSequence: int64Pointer(tombstone.Sequence),
		JournalKeyID:      stringPointer(tombstone.KeyID),
		JournalKeyVersion: stringPointer(tombstone.KeyVersion), StartedAt: deletionNow,
	}
	f.repository.operation = &operation
	f.repository.article.DeletionState = DeletionPending
	f.repository.article.DeletionOperationID = stringPointer(operation.ID)
	f.repository.article.Version = 4
	return operation, tombstone
}

func deletionTestArticle() Article {
	return Article{
		ID: deletionArticleID, AuthorID: stringPointer(deletionAuthorID),
		Title: "private title", Body: "private body",
		PublicationState: PublicationPublished, ViolationState: ViolationClear,
		DeletionState: DeletionActive, Version: 3,
		CreatedAt: deletionNow.Add(-time.Hour), UpdatedAt: deletionNow.Add(-time.Hour),
	}
}

func deletionRequest(kind DeleteKind) PermanentDeletionRequest {
	return PermanentDeletionRequest{
		ArticleID:  deletionArticleID,
		IfMatch:    `"article-` + deletionArticleID + `-v3"`,
		DeleteKind: kind,
		TraceID:    deletionTraceID,
	}
}

func deletionAuthorPrincipal() session.Principal {
	return activeDeletionPrincipal(deletionAuthorID, "member")
}

func deletionAdminPrincipal() session.Principal {
	return activeDeletionPrincipal(deletionAdminID, "admin")
}

func activeDeletionPrincipal(userID string, role string) session.Principal {
	return session.Principal{
		SessionID: "session", UserID: userID, Role: role, Status: "active",
	}
}

func requirePendingDeletion(t *testing.T, repository *deletionRepositoryStub) {
	t.Helper()
	if repository.article == nil ||
		repository.article.DeletionState != DeletionPending ||
		repository.article.DeletionOperationID == nil ||
		repository.operation == nil ||
		repository.operation.Status != DeletionOperationPendingJournal {
		t.Fatalf("pending state mismatch: article=%#v operation=%#v",
			repository.article, repository.operation)
	}
}

type deletionTransactorStub struct {
	proofRepository            *deletionRepositoryStub
	proofAuditor               *deletionAuditorStub
	mu                         sync.Mutex
	beginAttempts              atomic.Int64
	firstLockOnce              sync.Once
	firstTransactionLocked     chan struct{}
	secondTransactionAttempted chan struct{}
	releaseFirstTransaction    chan struct{}
}

func (s *deletionTransactorStub) Begin(context.Context) (pgx.Tx, error) {
	attempt := s.beginAttempts.Add(1)
	if attempt == 2 && s.secondTransactionAttempted != nil {
		close(s.secondTransactionAttempted)
	}
	s.mu.Lock()
	s.firstLockOnce.Do(func() {
		if s.firstTransactionLocked == nil || s.releaseFirstTransaction == nil {
			return
		}
		close(s.firstTransactionLocked)
		<-s.releaseFirstTransaction
	})
	return &deletionTxStub{
		createArticleTxStub: &createArticleTxStub{},
		unlock:              s.mu.Unlock,
		proofRepository:     s.proofRepository,
		proofAuditor:        s.proofAuditor,
	}, nil
}

type deletionTxStub struct {
	*createArticleTxStub
	unlockOnce      sync.Once
	unlock          func()
	proofRepository *deletionRepositoryStub
	proofAuditor    *deletionAuditorStub
}

func (transaction *deletionTxStub) QueryRow(ctx context.Context, query string, arguments ...any) pgx.Row {
	if transaction.closed {
		return orphanSQLRow{err: pgx.ErrTxClosed}
	}
	return readDeletionFixtureRow(transaction.proofRepository, transaction.proofAuditor, query, arguments...)
}

func (transaction *deletionTxStub) Exec(ctx context.Context, query string, arguments ...any) (pgconn.CommandTag, error) {
	if transaction.closed {
		return pgconn.CommandTag{}, pgx.ErrTxClosed
	}
	if query == dispositionReadCommittedSQL {
		return pgconn.NewCommandTag("SET"), nil
	}
	return transaction.createArticleTxStub.Exec(ctx, query, arguments...)
}

func (tx *deletionTxStub) Commit(ctx context.Context) error {
	err := tx.createArticleTxStub.Commit(ctx)
	tx.unlockOnce.Do(tx.unlock)
	return err
}

func (tx *deletionTxStub) Rollback(ctx context.Context) error {
	err := tx.createArticleTxStub.Rollback(ctx)
	tx.unlockOnce.Do(tx.unlock)
	return err
}

type deletionRepositoryStub struct {
	onLoadOperation    func()
	mu                 sync.Mutex
	article            *Article
	operation          *DeletionOperation
	markJournaledErr   error
	markJournaledCalls atomic.Int64
	claimCalls         atomic.Int64
	deleteCalls        atomic.Int64
	completeCalls      atomic.Int64
}

func (repository *deletionRepositoryStub) LockArticleForDisposition(
	ctx context.Context, transaction pgx.Tx, articleID string,
) (Article, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if repository.article == nil || repository.article.ID != articleID {
		return Article{}, pgx.ErrNoRows
	}
	return *repository.article, nil
}

type deletionIdentityStub struct{}

func (deletionIdentityStub) LockTransferParties(
	ctx context.Context, transaction pgx.Tx, targetID *string, sourceID string,
) (DispositionParties, error) {
	return DispositionParties{SourceID: sourceID}, nil
}

func (deletionIdentityStub) LockDeletionCandidate(
	ctx context.Context, transaction pgx.Tx, userID string,
) (DispositionDeletionCandidate, error) {
	return DispositionDeletionCandidate{UserID: userID}, nil
}

func (deletionIdentityStub) ClearRetainedEmailIfEligible(
	ctx context.Context, transaction pgx.Tx, sourceID string, proof *DispositionCompletionProof,
) (bool, error) {
	_, err := proof.VerifyCompletion(ctx, transaction)
	return false, err
}

func (s *deletionRepositoryStub) LockArticleForDeletion(
	context.Context,
	pgx.Tx,
	string,
) (Article, *DeletionOperation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.article == nil {
		return Article{}, nil, pgx.ErrNoRows
	}
	article := *s.article
	if s.operation == nil {
		return article, nil, nil
	}
	operation := *s.operation
	return article, &operation, nil
}

func (s *deletionRepositoryStub) InsertDeletionOperation(
	_ context.Context,
	_ pgx.Tx,
	operation DeletionOperation,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.operation != nil {
		return errors.New("duplicate deletion operation")
	}
	copy := operation
	s.operation = &copy
	return nil
}

func (s *deletionRepositoryStub) SetArticleDeletionPending(
	_ context.Context,
	_ pgx.Tx,
	articleID string,
	expectedVersion int64,
	operationID string,
	updatedAt time.Time,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.article == nil || s.article.ID != articleID ||
		s.article.Version != expectedVersion ||
		s.article.DeletionState != DeletionActive {
		return errors.New("article pending transition rejected")
	}
	s.article.DeletionState = DeletionPending
	s.article.DeletionOperationID = stringPointer(operationID)
	s.article.Version++
	s.article.UpdatedAt = updatedAt
	return nil
}

func (s *deletionRepositoryStub) MarkDeletionOperationJournaled(
	_ context.Context,
	_ pgx.Tx,
	operationID string,
	tombstone ArticleDeletionTombstone,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.markJournaledErr != nil {
		return s.markJournaledErr
	}
	if s.operation == nil || s.operation.ID != operationID {
		return errors.New("operation not found")
	}
	if s.operation.Status != DeletionOperationPendingJournal {
		if s.operation.Status == DeletionOperationJournaled &&
			s.operation.TombstoneSequence != nil &&
			*s.operation.TombstoneSequence == tombstone.Sequence &&
			s.operation.JournalKeyID != nil &&
			*s.operation.JournalKeyID == tombstone.KeyID &&
			s.operation.JournalKeyVersion != nil &&
			*s.operation.JournalKeyVersion == tombstone.KeyVersion {
			return nil
		}
		return errors.New("operation is not pending journal")
	}
	s.markJournaledCalls.Add(1)
	s.operation.Status = DeletionOperationJournaled
	s.operation.TombstoneSequence = int64Pointer(tombstone.Sequence)
	s.operation.JournalKeyID = stringPointer(tombstone.KeyID)
	s.operation.JournalKeyVersion = stringPointer(tombstone.KeyVersion)
	return nil
}

func (s *deletionRepositoryStub) InsertReplayedDeletionOperation(
	_ context.Context,
	_ pgx.Tx,
	operation DeletionOperation,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	copy := operation
	s.operation = &copy
	return nil
}
func (s *deletionRepositoryStub) LoadDeletionOperationForUpdate(
	_ context.Context,
	_ pgx.Tx,
	operationID string,
) (DeletionOperation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.onLoadOperation != nil {
		s.onLoadOperation()
	}
	if s.operation == nil || s.operation.ID != operationID {
		return DeletionOperation{}, pgx.ErrNoRows
	}
	return *s.operation, nil
}

func (s *deletionRepositoryStub) ClaimDeletionOperation(
	_ context.Context,
	_ pgx.Tx,
	operationID string,
	workerID string,
	now time.Time,
	lockedUntil time.Time,
) (DeletionOperation, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.claimCalls.Add(1)
	if s.operation == nil || s.operation.ID != operationID {
		return DeletionOperation{}, false, pgx.ErrNoRows
	}
	if s.operation.Status == DeletionOperationCompleted {
		return *s.operation, false, nil
	}
	if s.operation.Status == DeletionOperationFinalizing &&
		s.operation.LockedUntil != nil && s.operation.LockedUntil.After(now) {
		return *s.operation, false, nil
	}
	if s.operation.Status != DeletionOperationJournaled &&
		s.operation.Status != DeletionOperationFinalizing {
		return DeletionOperation{}, false, errors.New("operation cannot be claimed")
	}
	s.operation.Status = DeletionOperationFinalizing
	s.operation.ClaimedBy = stringPointer(workerID)
	s.operation.LockedUntil = timePointer(lockedUntil)
	return *s.operation, true, nil
}

func (s *deletionRepositoryStub) DeletePendingArticle(
	_ context.Context,
	_ pgx.Tx,
	articleID string,
	pendingVersion int64,
	operationID string,
) (PendingArticleDeletionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteCalls.Add(1)
	if s.article == nil {
		return PendingArticleAlreadyAbsent, nil
	}
	if s.article.ID != articleID ||
		s.article.Version != pendingVersion ||
		s.article.DeletionState != DeletionPending ||
		s.article.DeletionOperationID == nil ||
		*s.article.DeletionOperationID != operationID {
		return PendingArticlePresentMismatch, nil
	}
	s.article = nil
	return PendingArticleDeleted, nil
}

func (s *deletionRepositoryStub) CompleteDeletionOperation(
	_ context.Context,
	_ pgx.Tx,
	operationID string,
	workerID string,
	auditEventID string,
	completedAt time.Time,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.completeCalls.Add(1)
	if s.operation == nil || s.operation.ID != operationID {
		return pgx.ErrNoRows
	}
	if s.operation.Status == DeletionOperationCompleted {
		return nil
	}
	if s.operation.Status != DeletionOperationFinalizing ||
		s.operation.ClaimedBy == nil || *s.operation.ClaimedBy != workerID {
		return errors.New("operation lease is not owned")
	}
	s.operation.Status = DeletionOperationCompleted
	s.operation.ClaimedBy = nil
	s.operation.LockedUntil = nil
	s.operation.AuditEventID = stringPointer(auditEventID)
	s.operation.CompletedAt = timePointer(completedAt)
	return nil
}

type deletionJournalStub struct {
	mu        sync.Mutex
	records   []ArticleDeletionTombstone
	appendErr error
}

func (s *deletionJournalStub) Append(
	_ context.Context,
	request DeletionTombstoneRequest,
) (ArticleDeletionTombstone, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.appendErr != nil {
		return ArticleDeletionTombstone{}, s.appendErr
	}
	for _, record := range s.records {
		if record.DeletionOperationID == request.DeletionOperationID {
			return record, nil
		}
	}
	record := ArticleDeletionTombstone{
		Namespace: ArticleDeletionNamespace, Sequence: int64(len(s.records) + 1),
		ArticleID: request.ArticleID, ArticleVersion: request.ArticleVersion,
		DeletionOperationID: request.DeletionOperationID, DeletedAt: request.DeletedAt,
		KeyID: "key", KeyVersion: "v1", Stage: deletionTombstoneStage,
		RecordHMAC: "hmac:synthetic",
	}
	s.records = append(s.records, record)
	return record, nil
}

func (s *deletionJournalStub) FindByOperation(
	_ context.Context,
	operationID string,
) (ArticleDeletionTombstone, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, record := range s.records {
		if record.DeletionOperationID == operationID {
			return record, nil
		}
	}
	return ArticleDeletionTombstone{}, ErrDeletionTombstoneNotFound
}

type deletionFinalizerStub struct {
	delegate *DeletionFinalizer
	calls    atomic.Int64
	err      error
}

func (s *deletionFinalizerStub) Finalize(
	ctx context.Context,
	operationID string,
	traceID string,
) error {
	s.calls.Add(1)
	if s.err != nil {
		return s.err
	}
	return s.delegate.Finalize(ctx, operationID, traceID)
}

type deletionAuditorStub struct {
	mu     sync.Mutex
	events []audit.Event
}

func (s *deletionAuditorStub) Append(
	_ context.Context,
	_ pgx.Tx,
	event audit.Event,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

type deletionIDStub struct {
	id  string
	err error
}

func (s deletionIDStub) NewID() (string, error) {
	return s.id, s.err
}

type deletionClockStub struct {
	now time.Time
}

func (s deletionClockStub) Now() time.Time {
	return s.now
}

func int64Pointer(value int64) *int64 {
	return &value
}

func timePointer(value time.Time) *time.Time {
	return &value
}
