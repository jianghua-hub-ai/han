//go:build integration

package article

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"han/internal/session"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var authorListMigrationFiles = []string{
	"0001_shared.sql",
	"0002_identity.sql",
	"0003_article.sql",
	"0004_idempotency.sql",
	"0005_recovery_runs.sql",
	"0006_user_governance_indexes.sql",
	"0007_idempotency_response_snapshot_v1.sql",
	"0008_password_reset_durable.sql",
	"0009_article_replay_attempt.sql",
	"0010_recovery_runtime_metadata.sql",
	"0011_article_replay_publication.sql",
}

// newAuthorListFixture owns one isolated schema that is dropped on cleanup.
func newAuthorListFixture(t *testing.T) (*pgxpool.Pool, context.Context) {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("HAN_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Fatal("HAN_TEST_POSTGRES_DSN is required for the author list integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	t.Cleanup(cancel)

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("invalid HAN_TEST_POSTGRES_DSN value")
	}
	if config.ConnConfig.RuntimeParams == nil {
		config.ConnConfig.RuntimeParams = map[string]string{}
	}
	schema := "article_author_list_" + authorListRandomHex(t)
	quoted := pgx.Identifier{schema}.Sanitize()
	config.ConnConfig.RuntimeParams["search_path"] = quoted
	config.MaxConns = 4
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal("open PostgreSQL pool for the author list integration test")
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatalf("create isolated schema: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cleanupCancel()
		if _, err := pool.Exec(cleanupCtx, "DROP SCHEMA IF EXISTS "+quoted+" CASCADE"); err != nil {
			t.Errorf("drop isolated schema %s: %v", schema, err)
		}
	})

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin migration transaction: %v", err)
	}
	for _, name := range authorListMigrationFiles {
		raw, readErr := os.ReadFile(filepath.Join("..", "..", "migrations", name))
		if readErr != nil {
			_ = tx.Rollback(ctx)
			t.Fatalf("read migration %s: %v", name, readErr)
		}
		if _, err := tx.Exec(ctx, string(raw)); err != nil {
			_ = tx.Rollback(ctx)
			t.Fatalf("apply migration %s: %v", name, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit migrations: %v", err)
	}
	return pool, ctx
}

// TestAuthorArticleListKeysetAgainstPostgres proves the new "my articles" seam
// against a real PostgreSQL 18 schema: only the caller's own Articles, newest
// first, keyset-paged without repeats or gaps, and restricted Articles still
// visible to their author.
func TestAuthorArticleListKeysetAgainstPostgres(t *testing.T) {
	pool, ctx := newAuthorListFixture(t)
	authorID := authorListSeedUser(t, pool, ctx, "author")
	otherID := authorListSeedUser(t, pool, ctx, "other")

	base := time.Date(2026, time.November, 6, 8, 0, 0, 0, time.UTC)
	authorListSeedArticle(t, pool, ctx, authorID, "第一篇", base.Add(2*time.Hour), false)
	authorListSeedArticle(t, pool, ctx, authorID, "第二篇", base.Add(1*time.Hour), true)
	authorListSeedArticle(t, pool, ctx, authorID, "第三篇", base, false)
	authorListSeedArticle(t, pool, ctx, otherID, "别人的文章", base.Add(3*time.Hour), false)

	repository, err := NewPostgresArticleQueryRepository(pool)
	if err != nil {
		t.Fatalf("query repository: %v", err)
	}
	service, err := NewArticleQueryService(
		repository,
		&articleAuthorStateReaderStub{states: map[string]queryAuthorStatus{}},
		QueryConfig{DefaultLimit: 20, MaxLimit: 50, CandidateBatchSize: 51},
	)
	if err != nil {
		t.Fatalf("query service: %v", err)
	}
	principal := session.Principal{UserID: authorID, Role: "member", Status: "active"}
	limit := 2

	first, err := service.ListAuthorArticles(ctx, principal, ListArticlesRequest{Limit: &limit})
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if len(first.Items) != 2 {
		t.Fatalf("first page items = %d, want 2", len(first.Items))
	}
	if first.Items[0].Title != "第一篇" || first.Items[1].Title != "第二篇" {
		t.Fatalf("first page order = %q, %q", first.Items[0].Title, first.Items[1].Title)
	}
	if first.NextCursor == nil || *first.NextCursor == "" {
		t.Fatal("first page next_cursor is empty, want a continuation")
	}

	second, err := service.ListAuthorArticles(ctx, principal, ListArticlesRequest{
		Cursor: *first.NextCursor,
		Limit:  &limit,
	})
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if len(second.Items) != 1 || second.Items[0].Title != "第三篇" {
		t.Fatalf("second page = %#v, want only 第三篇", second.Items)
	}
	if second.NextCursor != nil {
		t.Fatalf("second page next_cursor = %q, want nil", *second.NextCursor)
	}

	seen := map[string]bool{}
	for _, item := range append(first.Items, second.Items...) {
		if seen[item.ID] {
			t.Fatalf("article %s appeared twice across pages", item.ID)
		}
		seen[item.ID] = true
		if item.Title == "别人的文章" {
			t.Fatal("another author's article leaked into the list")
		}
	}
	if len(seen) != 3 {
		t.Fatalf("distinct articles = %d, want 3", len(seen))
	}

	// A restricted (violating) Article stays manageable by its own author.
	if first.Items[1].Title != "第二篇" {
		t.Fatalf("restricted article missing from the author's page")
	}
}

func authorListSeedUser(
	t *testing.T,
	pool *pgxpool.Pool,
	ctx context.Context,
	name string,
) string {
	t.Helper()
	id := authorListUUID(t)
	username := "author-list-" + name + "-" + authorListRandomHex(t)
	if _, err := pool.Exec(ctx, `
		INSERT INTO users (
			id, username, email, phone, role, status, credential_mode,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, 'member', 'active', 'permanent', $5, $5)
	`,
		id,
		username,
		username+"@example.test",
		"+1555"+strings.ReplaceAll(id, "-", "")[:8],
		time.Now().UTC(),
	); err != nil {
		t.Fatalf("insert user %s: %v", name, err)
	}
	return id
}

func authorListSeedArticle(
	t *testing.T,
	pool *pgxpool.Pool,
	ctx context.Context,
	authorID string,
	title string,
	createdAt time.Time,
	violating bool,
) string {
	t.Helper()
	id := authorListUUID(t)
	violationState := "clear"
	var category *string
	if violating {
		violationState = "violating"
		value := "violence"
		category = &value
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO articles (
			id, author_id, title, body, publication_state,
			violation_state, violation_category, deletion_state,
			version, created_at, updated_at
		)
		VALUES ($1, $2, $3, '正文', 'published', $4, $5, 'active', 1, $6, $6)
	`,
		id, authorID, title, violationState, category, createdAt,
	); err != nil {
		t.Fatalf("insert article %s: %v", title, err)
	}
	return id
}

func authorListRandomHex(t *testing.T) string {
	t.Helper()
	raw := make([]byte, 6)
	if _, err := rand.Read(raw); err != nil {
		t.Fatalf("read random bytes: %v", err)
	}
	return hex.EncodeToString(raw)
}

func authorListUUID(t *testing.T) string {
	t.Helper()
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		t.Fatalf("read random UUID bytes: %v", err)
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	return strings.Join([]string{
		hex.EncodeToString(raw[0:4]),
		hex.EncodeToString(raw[4:6]),
		hex.EncodeToString(raw[6:8]),
		hex.EncodeToString(raw[8:10]),
		hex.EncodeToString(raw[10:16]),
	}, "-")
}
