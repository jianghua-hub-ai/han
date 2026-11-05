// Package articleperf measures in-process Article service baselines.
//
// The benchmarks use deterministic in-memory implementations of the existing
// Article persistence and identity interfaces. They exercise the production
// Article service, policy, worker scheduling, finalizer, and replay coordinator
// code. They do not measure PostgreSQL, HTTP decoding, SMTP, or filesystem
// journal durability.
package articleperf

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"han/internal/article"
	"han/internal/audit"
	"han/internal/mailer"
	"han/internal/session"
)

const (
	benchmarkArticleSize    = 1000
	benchmarkHiddenBatch    = 100
	benchmarkWorkerOps      = 100
	benchmarkReplayCount    = 1000
	benchmarkDefaultLimit   = 20
	benchmarkMaxLimit       = 50
	benchmarkCandidateBatch = 100
)

var benchmarkNow = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// benchmarkTx implements only the transaction operations exercised by the
// measured paths. Any other pgx.Tx call would panic through the nil embedded
// interface rather than silently pretending to run SQL.
type benchmarkTx struct {
	pgx.Tx
}

func (benchmarkTx) Commit(context.Context) error   { return nil }
func (benchmarkTx) Rollback(context.Context) error { return nil }

type benchmarkTransactor struct{}

func (benchmarkTransactor) Begin(context.Context) (pgx.Tx, error) {
	return benchmarkTx{}, nil
}

type fixedClock struct {
	now time.Time
}

func (clock fixedClock) Now() time.Time { return clock.now }

type sliceIDGenerator struct {
	ids  []string
	next atomic.Int64
}

func (generator *sliceIDGenerator) NewID() (string, error) {
	index := int(generator.next.Add(1) - 1)
	if index < 0 || index >= len(generator.ids) {
		return "", errors.New("benchmark article ID sequence exhausted")
	}
	return generator.ids[index], nil
}

type countingIDGenerator struct {
	next atomic.Int64
}

func (generator *countingIDGenerator) NewID() (string, error) {
	return benchmarkUUID(int(generator.next.Add(1))), nil
}

func benchmarkUUID(value int) string {
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", value)
}

func benchmarkPrincipal() session.Principal {
	return session.Principal{
		UserID: benchmarkUUID(999999),
		Role:   "member",
		Status: "active",
	}
}

// memoryArticleRepository is a minimal implementation of both command-side
// persistence interfaces. It stores one Article and never grows with b.N.
type memoryArticleRepository struct {
	article article.Article
	stored  bool
}

func (repository *memoryArticleRepository) InsertArticle(
	_ context.Context,
	_ pgx.Tx,
	value article.Article,
) error {
	repository.article = value
	repository.stored = true
	return nil
}

func (repository *memoryArticleRepository) LockArticleForUpdate(
	_ context.Context,
	_ pgx.Tx,
	articleID string,
) (article.Article, error) {
	if !repository.stored || repository.article.ID != articleID {
		return article.Article{}, pgx.ErrNoRows
	}
	return repository.article, nil
}

func (repository *memoryArticleRepository) UpdateArticleContent(
	_ context.Context,
	_ pgx.Tx,
	value article.Article,
) error {
	repository.article = value
	return nil
}

func BenchmarkCreateArticleMaxContent(b *testing.B) {
	b.Run("ascii", func(b *testing.B) {
		benchmarkCreateArticle(
			b,
			strings.Repeat("a", 200),
			strings.Repeat("b", 20000),
		)
	})
	b.Run("unicode-3-byte", func(b *testing.B) {
		benchmarkCreateArticle(
			b,
			strings.Repeat("\u754c", 200),
			strings.Repeat("\u6587", 20000),
		)
	})
}

func benchmarkCreateArticle(b *testing.B, title, body string) {
	b.Helper()

	ids := make([]string, b.N)
	for index := range ids {
		ids[index] = benchmarkUUID(index + 1)
	}
	repository := &memoryArticleRepository{}
	service, err := article.NewCommandService(article.CommandDependencies{
		Transactions: benchmarkTransactor{},
		Repository:   repository,
		IDs:          &sliceIDGenerator{ids: ids},
		Clock:        fixedClock{now: benchmarkNow},
	})
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	principal := benchmarkPrincipal()
	input := article.CreateArticleInput{Title: title, Body: body}

	b.ReportAllocs()
	b.SetBytes(int64(len(title) + len(body)))
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		if _, err := service.Create(ctx, principal, input); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()

	if !repository.stored ||
		repository.article.Version != 1 ||
		repository.article.Title != title ||
		repository.article.Body != body {
		b.Fatal("create benchmark did not persist the expected article")
	}
}

func BenchmarkReplaceContentMaxContent(b *testing.B) {
	b.Run("ascii", func(b *testing.B) {
		benchmarkReplaceContent(
			b,
			strings.Repeat("c", 200),
			strings.Repeat("d", 20000),
		)
	})
	b.Run("unicode-3-byte", func(b *testing.B) {
		benchmarkReplaceContent(
			b,
			strings.Repeat("\u754c", 200),
			strings.Repeat("\u6587", 20000),
		)
	})
}

func benchmarkReplaceContent(b *testing.B, title, body string) {
	b.Helper()

	principal := benchmarkPrincipal()
	authorID := principal.UserID
	articleID := benchmarkUUID(424242)
	repository := &memoryArticleRepository{
		article: article.Article{
			ID:               articleID,
			AuthorID:         &authorID,
			Title:            "seed title",
			Body:             "seed body",
			PublicationState: article.PublicationPublished,
			ViolationState:   article.ViolationClear,
			DeletionState:    article.DeletionActive,
			Version:          1,
			CreatedAt:        benchmarkNow,
			UpdatedAt:        benchmarkNow,
		},
		stored: true,
	}
	service, err := article.NewCommandService(article.CommandDependencies{
		Transactions: benchmarkTransactor{},
		Repository:   repository,
		IDs:          &sliceIDGenerator{ids: []string{benchmarkUUID(1)}},
		Clock:        fixedClock{now: benchmarkNow},
	})
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	input := article.CreateArticleInput{Title: title, Body: body}
	expectedVersion := repository.article.Version

	b.ReportAllocs()
	b.SetBytes(int64(len(title) + len(body)))
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		updated, err := service.ReplaceContent(
			ctx,
			principal,
			articleID,
			expectedVersion,
			input,
		)
		if err != nil {
			b.Fatal(err)
		}
		expectedVersion = updated.Version
	}
	b.StopTimer()

	if repository.article.Title != title || repository.article.Body != body {
		b.Fatal("replace benchmark did not persist the expected content")
	}
}

type staticQueryRepository struct {
	articles []article.Article
	scanned  atomic.Int64
	batches  atomic.Int64
}

func (repository *staticQueryRepository) ListCandidateBatch(
	_ context.Context,
	after *article.ArticleCursor,
	limit int,
) ([]article.Article, error) {
	if limit <= 0 {
		return nil, errors.New("benchmark candidate limit must be positive")
	}
	start := 0
	if after != nil {
		start = sort.Search(len(repository.articles), func(index int) bool {
			candidate := repository.articles[index]
			return candidate.CreatedAt.Before(after.CreatedAt) ||
				(candidate.CreatedAt.Equal(after.CreatedAt) && candidate.ID < after.ID)
		})
	}
	end := start + limit
	if end > len(repository.articles) {
		end = len(repository.articles)
	}
	batch := repository.articles[start:end]
	repository.scanned.Add(int64(len(batch)))
	repository.batches.Add(1)
	return batch, nil
}

func (repository *staticQueryRepository) GetArticle(
	_ context.Context,
	articleID string,
) (article.Article, error) {
	for _, candidate := range repository.articles {
		if candidate.ID == articleID {
			return candidate, nil
		}
	}
	return article.Article{}, pgx.ErrNoRows
}

type staticAuthorStates struct {
	states map[string]string
}

func (reader staticAuthorStates) GetAuthorStates(
	_ context.Context,
	authorIDs []string,
) (map[string]string, error) {
	states := make(map[string]string, len(authorIDs))
	for _, authorID := range authorIDs {
		if status, exists := reader.states[authorID]; exists {
			states[authorID] = status
		}
	}
	return states, nil
}

func memberListDataset(size, hiddenPercent int) ([]article.Article, map[string]string) {
	if size <= 0 || hiddenPercent < 0 || hiddenPercent > 100 {
		panic("invalid member list benchmark dataset parameters")
	}
	records := make([]article.Article, 0, size)
	states := make(map[string]string, size)
	for index := 0; index < size; index++ {
		authorID := benchmarkUUID(1000000 + index)
		status := "active"
		if index%benchmarkHiddenBatch < hiddenPercent {
			status = "suspended"
		}
		states[authorID] = status

		createdAt := benchmarkNow.Add(-time.Duration(index) * time.Second)
		records = append(records, article.Article{
			ID:               benchmarkUUID(index + 1),
			AuthorID:         &authorID,
			Title:            "benchmark article",
			Body:             "benchmark body",
			PublicationState: article.PublicationPublished,
			ViolationState:   article.ViolationClear,
			DeletionState:    article.DeletionActive,
			Version:          1,
			CreatedAt:        createdAt,
			UpdatedAt:        createdAt,
		})
	}
	return records, states
}

func BenchmarkMemberListHiddenDensity(b *testing.B) {
	for _, requestedLimit := range []int{benchmarkDefaultLimit, benchmarkMaxLimit} {
		requestedLimit := requestedLimit
		for _, hiddenPercent := range []int{0, 50, 90, 100} {
			hiddenPercent := hiddenPercent
			name := fmt.Sprintf("limit-%02d/hidden-%03d", requestedLimit, hiddenPercent)
			b.Run(name, func(b *testing.B) {
				records, states := memberListDataset(benchmarkArticleSize, hiddenPercent)
				repository := &staticQueryRepository{articles: records}
				service, err := article.NewArticleQueryService(
					repository,
					staticAuthorStates{states: states},
					article.QueryConfig{
						DefaultLimit:       benchmarkDefaultLimit,
						MaxLimit:           benchmarkMaxLimit,
						CandidateBatchSize: benchmarkCandidateBatch,
					},
				)
				if err != nil {
					b.Fatal(err)
				}

				limit := requestedLimit
				request := article.ListArticlesRequest{Limit: &limit}
				ctx := context.Background()
				page, err := service.ListArticles(ctx, request)
				if err != nil {
					b.Fatal(err)
				}
				expectedVisible := benchmarkArticleSize * (100 - hiddenPercent) / 100
				if expectedVisible > requestedLimit {
					expectedVisible = requestedLimit
				}
				if len(page.Items) != expectedVisible {
					b.Fatalf(
						"setup page items = %d, want %d",
						len(page.Items),
						expectedVisible,
					)
				}
				repository.scanned.Store(0)
				repository.batches.Store(0)

				b.ReportAllocs()
				b.ResetTimer()
				for iteration := 0; iteration < b.N; iteration++ {
					if _, err := service.ListArticles(ctx, request); err != nil {
						b.Fatal(err)
					}
				}
				b.StopTimer()

				b.ReportMetric(
					float64(repository.scanned.Load())/float64(b.N),
					"candidates/op",
				)
				b.ReportMetric(
					float64(repository.batches.Load())/float64(b.N),
					"batches/op",
				)
				b.ReportMetric(float64(expectedVisible), "visible-items/op")
			})
		}
	}
}

type staticDiscoverer struct {
	operationIDs []string
}

func (discoverer staticDiscoverer) DiscoverDeletionOperations(
	_ context.Context,
	_ time.Time,
	limit int,
) ([]string, error) {
	if limit <= 0 {
		return nil, errors.New("benchmark discovery limit must be positive")
	}
	end := limit
	if end > len(discoverer.operationIDs) {
		end = len(discoverer.operationIDs)
	}
	return discoverer.operationIDs[:end], nil
}

type countingConverger struct {
	calls atomic.Int64
}

func (converger *countingConverger) ConvergeDeletionOperation(
	_ context.Context,
	_ string,
	_ string,
) error {
	converger.calls.Add(1)
	return nil
}

func BenchmarkDeletionWorkerReconcile100(b *testing.B) {
	operationIDs := make([]string, benchmarkWorkerOps)
	for index := range operationIDs {
		operationIDs[index] = benchmarkUUID(index + 1)
	}
	converger := &countingConverger{}
	worker, err := article.NewDeletionWorker(
		article.DeletionWorkerDependencies{
			Discoverer: staticDiscoverer{operationIDs: operationIDs},
			Converger:  converger,
			Now:        func() time.Time { return benchmarkNow },
			NewTraceID: func() (string, error) { return "benchmark-trace", nil },
		},
		article.DeletionWorkerConfig{
			PollInterval:      time.Millisecond,
			BatchSize:         benchmarkWorkerOps,
			OperationBudget:   time.Second,
			BackoffInitial:    time.Millisecond,
			BackoffMax:        10 * time.Millisecond,
			MaxBatchesPerWake: 1,
			ShutdownBudget:    time.Second,
		},
	)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	if err := worker.Reconcile(ctx); err != nil {
		b.Fatal(err)
	}
	if calls := converger.calls.Load(); calls != benchmarkWorkerOps {
		b.Fatalf("setup convergence calls = %d, want %d", calls, benchmarkWorkerOps)
	}
	converger.calls.Store(0)

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		if err := worker.Reconcile(ctx); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()

	if calls := converger.calls.Load(); calls != int64(b.N*benchmarkWorkerOps) {
		b.Fatalf(
			"convergence calls = %d, want %d",
			calls,
			b.N*benchmarkWorkerOps,
		)
	}
	b.ReportMetric(benchmarkWorkerOps, "operations/op")
}

type completedOperationRepository struct {
	operation article.DeletionOperation
}

func (repository *completedOperationRepository) LoadDeletionOperationForUpdate(
	_ context.Context,
	_ pgx.Tx,
	operationID string,
) (article.DeletionOperation, error) {
	operation := repository.operation
	operation.ID = operationID
	return operation, nil
}

func (repository *completedOperationRepository) LockArticleForDisposition(
	context.Context,
	pgx.Tx,
	string,
) (article.Article, error) {
	return article.Article{}, errors.New("benchmark completed path must not lock an article")
}

func (repository *completedOperationRepository) MarkDeletionOperationJournaled(
	context.Context,
	pgx.Tx,
	string,
	article.ArticleDeletionTombstone,
) error {
	return errors.New("benchmark completed path must not journal an operation")
}

func (repository *completedOperationRepository) ClaimDeletionOperation(
	context.Context,
	pgx.Tx,
	string,
	string,
	time.Time,
	time.Time,
) (article.DeletionOperation, bool, error) {
	return article.DeletionOperation{}, false,
		errors.New("benchmark completed path must not claim an operation")
}

func (repository *completedOperationRepository) DeletePendingArticle(
	context.Context,
	pgx.Tx,
	string,
	int64,
	string,
) (article.PendingArticleDeletionResult, error) {
	return "", errors.New("benchmark completed path must not delete an article")
}

func (repository *completedOperationRepository) CompleteDeletionOperation(
	context.Context,
	pgx.Tx,
	string,
	string,
	string,
	time.Time,
) error {
	return errors.New("benchmark completed path must not complete an operation")
}

func (repository *completedOperationRepository) InsertReplayedDeletionOperation(
	context.Context,
	pgx.Tx,
	article.DeletionOperation,
) error {
	return errors.New("benchmark completed path must not insert an operation")
}

type unusedDeletionJournal struct{}

func (unusedDeletionJournal) Append(
	context.Context,
	article.DeletionTombstoneRequest,
) (article.ArticleDeletionTombstone, error) {
	return article.ArticleDeletionTombstone{},
		errors.New("benchmark completed path must not append a tombstone")
}

func (unusedDeletionJournal) FindByOperation(
	context.Context,
	string,
) (article.ArticleDeletionTombstone, error) {
	return article.ArticleDeletionTombstone{},
		errors.New("benchmark completed path must not read a tombstone")
}

type discardAuditor struct{}

func (discardAuditor) Append(context.Context, pgx.Tx, audit.Event) error {
	return nil
}

func newCompletedFinalizer(b *testing.B) *article.DeletionFinalizer {
	b.Helper()

	finalizer, err := article.NewDeletionFinalizer(article.DeletionFinalizerDependencies{
		Transactions: benchmarkTransactor{},
		Repository: &completedOperationRepository{
			operation: article.DeletionOperation{
				ID:                benchmarkUUID(1),
				ArticleID:         benchmarkUUID(500000),
				ArticleVersion:    1,
				DeleteKind:        article.DeleteOrdinary,
				Status:            article.DeletionOperationCompleted,
				StartedAt:         benchmarkNow,
				CompletedAt:       &benchmarkNow,
				TombstoneSequence: pointerToInt64(1),
			},
		},
		Journal:       unusedDeletionJournal{},
		Auditor:       discardAuditor{},
		AuditIDs:      &countingIDGenerator{},
		Clock:         fixedClock{now: benchmarkNow},
		WorkerID:      benchmarkUUID(900000),
		LeaseDuration: time.Minute,
	})
	if err != nil {
		b.Fatal(err)
	}
	return finalizer
}

func pointerToInt64(value int64) *int64 {
	return &value
}

func BenchmarkDeletionFinalizerCompletedNoop(b *testing.B) {
	finalizer := newCompletedFinalizer(b)
	ctx := context.Background()
	operationID := benchmarkUUID(1)

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		if err := finalizer.ConvergeDeletionOperation(
			ctx,
			operationID,
			"benchmark-trace",
		); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReplayAllCompletedTombstones1000(b *testing.B) {
	finalizer := newCompletedFinalizer(b)
	tombstones := make([]article.ArticleDeletionTombstone, benchmarkReplayCount)
	for index := range tombstones {
		tombstones[index] = article.ArticleDeletionTombstone{
			Namespace:           "benchmark",
			Sequence:            int64(index + 1),
			ArticleID:           benchmarkUUID(500000 + index),
			ArticleVersion:      1,
			DeletionOperationID: benchmarkUUID(index + 1),
			DeletedAt:           benchmarkNow,
			KeyID:               "benchmark",
			KeyVersion:          "1",
			Stage:               "tombstoned",
		}
	}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		result, err := finalizer.ReplayAllTombstones(
			ctx,
			tombstones,
			"benchmark-trace",
		)
		if err != nil {
			b.Fatal(err)
		}
		if result.Replayed != 0 ||
			result.AlreadyAbsent != benchmarkReplayCount {
			b.Fatalf(
				"replay result = %+v, want %d already absent",
				result,
				benchmarkReplayCount,
			)
		}
	}
	b.StopTimer()
	b.ReportMetric(benchmarkReplayCount, "tombstones/op")
}

// benchmarkNoticeTransactionsPerRound counts the production transactions opened
// by one empty TransferNoticeDispatcher.Reconcile round. reconcile calls
// maintainWork (one maintain lookup) and then dispatchWork (one due-claim
// lookup); both stop at the first empty result, so BatchLimit adds no more.
const benchmarkNoticeTransactionsPerRound = 2

// benchmarkNoticeTraceID is the verified 26-character Crockford base32 trace
// identifier expected by platform.ContextWithTraceID.
const benchmarkNoticeTraceID = "01ARZ3NDEKTSV4RRFFQ69G5FAV"

// noticeEmptyRow makes the production readNoticeRow path translate the scan
// into "no recipient found" without any SQL execution.
type noticeEmptyRow struct{}

func (noticeEmptyRow) Scan(...any) error { return pgx.ErrNoRows }

// noticeEmptyTx implements only the pgx.Tx surface used by an empty dispatcher
// round. Any other pgx.Tx call would panic through the nil embedded interface.
type noticeEmptyTx struct {
	pgx.Tx
}

func (noticeEmptyTx) QueryRow(context.Context, string, ...any) pgx.Row {
	return noticeEmptyRow{}
}

func (noticeEmptyTx) Commit(context.Context) error   { return nil }
func (noticeEmptyTx) Rollback(context.Context) error { return nil }

type noticeTransactionCounter struct {
	begun atomic.Int64
}

func (counter *noticeTransactionCounter) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	counter.begun.Add(1)
	return noticeEmptyTx{}, nil
}

type openNoticeGate struct{}

func (openNoticeGate) RequireOpen(context.Context) error             { return nil }
func (openNoticeGate) RequireOpenInTx(context.Context, pgx.Tx) error { return nil }

type unusedNoticeMailer struct{}

func (unusedNoticeMailer) Send(context.Context, mailer.Message) error { return nil }

type unusedNoticeAudit struct{}

func (unusedNoticeAudit) Append(context.Context, pgx.Tx, audit.Event) error { return nil }

func (unusedNoticeAudit) Get(context.Context, audit.QueryRower, string) (audit.Event, error) {
	return audit.Event{}, nil
}

type quietNoticeLogger struct{}

func (quietNoticeLogger) Log(context.Context, slog.Level, string, ...any) {}

func BenchmarkNoticeDispatcherEmptyRound(b *testing.B) {
	transactions := &noticeTransactionCounter{}
	dispatcher, err := article.NewTransferNoticeDispatcher(
		article.NoticeDispatcherDependencies{
			AcquireWork: func(ctx context.Context) (context.Context, func(), error) {
				return ctx, func() {}, nil
			},
			Transactions: transactions,
			Repository:   article.Repository{},
			Gate:         openNoticeGate{},
			Mailer:       unusedNoticeMailer{},
			Audit:        unusedNoticeAudit{},
			IDs:          &countingIDGenerator{},
			NewTraceID:   func() (string, error) { return benchmarkNoticeTraceID, nil },
			Logger:       quietNoticeLogger{},
			WorkerID:     benchmarkUUID(900001),
		},
		article.NoticeDispatcherConfig{
			PollInterval:        time.Second,
			BatchLimit:          2,
			LeaseDuration:       10 * time.Second,
			DBOperationBudget:   time.Second,
			BackoffInitial:      time.Second,
			BackoffMax:          8 * time.Second,
			SMTPOperationBudget: 2 * time.Second,
		},
	)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	if err := dispatcher.Reconcile(ctx); err != nil {
		b.Fatalf("setup reconcile: %v", err)
	}
	if begun := transactions.begun.Load(); begun != benchmarkNoticeTransactionsPerRound {
		b.Fatalf(
			"setup transactions = %d, want %d",
			begun,
			benchmarkNoticeTransactionsPerRound,
		)
	}
	transactions.begun.Store(0)

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		if err := dispatcher.Reconcile(ctx); err != nil {
			b.Fatalf("reconcile: %v", err)
		}
	}
	b.StopTimer()

	if err := dispatcher.Err(); err != nil {
		b.Fatalf("dispatcher err = %v", err)
	}
	expected := int64(b.N) * benchmarkNoticeTransactionsPerRound
	if begun := transactions.begun.Load(); begun != expected {
		b.Fatalf("transactions = %d, want %d", begun, expected)
	}
	b.ReportMetric(float64(benchmarkNoticeTransactionsPerRound), "transactions/op")
	b.ReportMetric(1, "rounds/op")
}
