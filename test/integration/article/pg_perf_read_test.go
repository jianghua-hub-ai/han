//go:build integration

package articleintegration

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"han/internal/article"
)

// This file owns the PGPERF-C scenes (R01-R07, X01-X03). Every scene builds its
// own retained schema through newPgPerfHarness; expectations always come from
// the independent SQL oracle in pg_perf_harness_test.go and never from the
// cursor implementation under test. Nothing is dropped, truncated, or reused
// across scene points.

const (
	pgPerfReadDefaultLimit = 20
	pgPerfReadMaxLimit     = 100
	pgPerfReadMaxPages     = 100000
	pgPerfReadDetailRows   = 3
	pgPerfReadOracleAll    = 0
)

func pgPerfLimit(limit int) *int { return &limit }

// pgPerfReadPage performs one member list request and enforces the page size.
func pgPerfReadPage(ctx context.Context, service *article.ArticleQueryService, limit int, cursor string) ([]string, *string, error) {
	page, err := service.ListArticles(ctx, article.ListArticlesRequest{Cursor: cursor, Limit: pgPerfLimit(limit)})
	if err != nil {
		return nil, nil, err
	}
	ids := pgPerfPageIDs(page)
	if len(ids) > limit {
		return nil, nil, fmt.Errorf("list page returned %d items for limit %d", len(ids), limit)
	}
	return ids, page.NextCursor, nil
}

// pgPerfReadCompareWalk walks every page and compares the observed sequence to
// the ordered SQL oracle page by page. A duplicate or a missing row breaks the
// positional equality, so no separate duplicate map is needed.
func pgPerfReadCompareWalk(ctx context.Context, service *article.ArticleQueryService, limit int, expected []string) (int, error) {
	return pgPerfReadCompareFromCursor(ctx, service, limit, "", expected)
}

func pgPerfReadCompareFromCursor(ctx context.Context, service *article.ArticleQueryService, limit int, cursor string, expected []string) (int, error) {
	index := 0
	pages := 0
	for {
		ids, next, err := pgPerfReadPage(ctx, service, limit, cursor)
		if err != nil {
			return pages, err
		}
		pages++
		if index+len(ids) > len(expected) {
			return pages, fmt.Errorf("walk returned more items than oracle: page=%d observed=%d expected=%d", pages, index+len(ids), len(expected))
		}
		for _, id := range ids {
			if expected[index] != id {
				return pages, fmt.Errorf("walk mismatch at position %d on page %d", index, pages)
			}
			index++
		}
		if next == nil {
			break
		}
		if *next == "" || *next == cursor {
			return pages, errors.New("walk cursor did not advance")
		}
		cursor = *next
		if pages > pgPerfReadMaxPages {
			return pages, fmt.Errorf("walk exceeded %d pages", pgPerfReadMaxPages)
		}
	}
	if index != len(expected) {
		return pages, fmt.Errorf("walk stopped after %d of %d oracle rows", index, len(expected))
	}
	return pages, nil
}

// pgPerfReadLastPage returns the cursor that opens the terminal page and the
// ids on that page. The cursor is replayed by the measured operation.
func pgPerfReadLastPage(ctx context.Context, service *article.ArticleQueryService, limit int) (string, []string, int, error) {
	cursor := ""
	pages := 0
	for {
		ids, next, err := pgPerfReadPage(ctx, service, limit, cursor)
		if err != nil {
			return "", nil, pages, err
		}
		pages++
		if next == nil {
			return cursor, ids, pages, nil
		}
		if *next == "" || *next == cursor {
			return "", nil, pages, errors.New("last-page cursor did not advance")
		}
		cursor = *next
		if pages > pgPerfReadMaxPages {
			return "", nil, pages, fmt.Errorf("last-page walk exceeded %d pages", pgPerfReadMaxPages)
		}
	}
}

func pgPerfReadAssertPoint(t *testing.T, point string, summary pgPerfPointSummary) {
	t.Helper()
	if summary.Errors != 0 {
		t.Fatalf("%s observed %d execution errors", point, summary.Errors)
	}
	if summary.Successes == 0 {
		t.Fatalf("%s observed no successful iteration", point)
	}
}

func runPgPerfSmallRead(t *testing.T) {
	t.Run("R01-limit20", func(t *testing.T) {
		h := newPgPerfHarness(t, pgPerfTierSmall, "r01a", func(options *pgPerfOptions) {
			options.seedMode = "full"
			options.seedConfig = pgPerfSeedOverride(pgPerfTierSmall, "full", func(config *pgPerfSeedConfig) {
				config.Deletions = 0
				config.PendingOverride = 0
				config.HiddenDensity = 0
			})
		})
		oracle := h.memberVisibleIDs(h.ctx, pgPerfReadDefaultLimit)
		if len(oracle) != pgPerfReadDefaultLimit {
			t.Fatalf("R01-limit20 oracle expected %d rows, got %d", pgPerfReadDefaultLimit, len(oracle))
		}
		service := h.queryService()
		summary := h.pgPerfRunPoint(pgPerfTimedSpec("R01-limit20", "R01", 1, nil, "member list first page limit 20, hidden density 0"), func(ctx context.Context, worker int) (bool, error) {
			ids, next, err := pgPerfReadPage(ctx, service, pgPerfReadDefaultLimit, "")
			if err != nil {
				return false, err
			}
			if next == nil {
				return false, errors.New("R01-limit20 expected a non-nil next cursor")
			}
			if !pgPerfEqualIDs(ids, oracle) {
				return false, errors.New("R01-limit20 first page differs from the SQL oracle")
			}
			return true, nil
		})
		pgPerfReadAssertPoint(t, "R01-limit20", summary)
	})

	t.Run("R04-10rows", func(t *testing.T) {
		h := newPgPerfHarness(t, pgPerfTierSmall, "r04a", func(options *pgPerfOptions) {
			options.seedMode = "minimal"
			options.seedConfig = pgPerfSeedOverride(pgPerfTierSmall, "minimal", func(config *pgPerfSeedConfig) {
				config.Articles = 10
				config.Deletions = 0
				config.PendingOverride = 0
				config.HiddenDensity = 0
				config.Notices = 0
				config.Audits = 0
			})
		})
		oracle := h.memberVisibleIDs(h.ctx, pgPerfReadOracleAll)
		if len(oracle) != 10 {
			t.Fatalf("R04-10rows oracle expected 10 rows, got %d", len(oracle))
		}
		service := h.queryService()
		summary := h.pgPerfRunPoint(pgPerfTimedSpec("R04-10rows", "R04", 1, nil, "ten-row dataset terminal page"), func(ctx context.Context, worker int) (bool, error) {
			ids, next, err := pgPerfReadPage(ctx, service, pgPerfReadDefaultLimit, "")
			if err != nil {
				return false, err
			}
			if next != nil {
				return false, errors.New("R04-10rows expected a terminal page")
			}
			if !pgPerfEqualIDs(ids, oracle) {
				return false, errors.New("R04-10rows page differs from the SQL oracle")
			}
			return true, nil
		})
		pgPerfReadAssertPoint(t, "R04-10rows", summary)
	})

	t.Run("R04-empty", func(t *testing.T) {
		h := newPgPerfHarness(t, pgPerfTierSmall, "r04b", func(options *pgPerfOptions) {
			options.seedMode = "empty"
		})
		service := h.queryService()
		summary := h.pgPerfRunPoint(pgPerfSingleSpec("R04-empty", "R04", 1, "empty dataset list"), func(ctx context.Context, worker int) (bool, error) {
			ids, next, err := pgPerfReadPage(ctx, service, pgPerfReadDefaultLimit, "")
			if err != nil {
				return false, err
			}
			if next != nil {
				return false, errors.New("R04-empty expected a nil cursor")
			}
			if len(ids) != 0 {
				return false, fmt.Errorf("R04-empty expected an empty page, got %d items", len(ids))
			}
			return true, nil
		})
		pgPerfReadAssertPoint(t, "R04-empty", summary)
	})

	t.Run("R04-all-hidden", func(t *testing.T) {
		h := newPgPerfHarness(t, pgPerfTierSmall, "r04c", func(options *pgPerfOptions) {
			options.seedMode = "full"
			options.seedConfig = pgPerfSeedOverride(pgPerfTierSmall, "full", func(config *pgPerfSeedConfig) {
				config.Deletions = 0
				config.PendingOverride = 0
				config.HiddenDensity = 1
			})
		})
		if oracle := h.memberVisibleIDs(h.ctx, pgPerfReadOracleAll); len(oracle) != 0 {
			t.Fatalf("R04-all-hidden oracle expected no visible row, got %d", len(oracle))
		}
		service := h.queryService()
		summary := h.pgPerfRunPoint(pgPerfSingleSpec("R04-all-hidden", "R04", 1, "fully hidden dataset list"), func(ctx context.Context, worker int) (bool, error) {
			ids, next, err := pgPerfReadPage(ctx, service, pgPerfReadDefaultLimit, "")
			if err != nil {
				return false, err
			}
			if next != nil || len(ids) != 0 {
				return false, fmt.Errorf("R04-all-hidden expected empty items and nil cursor, got items=%d cursor=%t", len(ids), next != nil)
			}
			return true, nil
		})
		pgPerfReadAssertPoint(t, "R04-all-hidden", summary)
	})

	t.Run("R04-last-page", func(t *testing.T) {
		h := newPgPerfHarness(t, pgPerfTierSmall, "r04d", func(options *pgPerfOptions) {
			options.seedMode = "full"
			options.seedConfig = pgPerfSeedOverride(pgPerfTierSmall, "full", func(config *pgPerfSeedConfig) {
				config.Deletions = 0
				config.PendingOverride = 0
				config.HiddenDensity = 0
			})
		})
		service := h.queryService()
		startCursor, lastIDs, pages, err := pgPerfReadLastPage(h.ctx, service, pgPerfReadDefaultLimit)
		if err != nil {
			t.Fatalf("R04-last-page walk failed: %v", err)
		}
		if pages < 2 || len(lastIDs) == 0 {
			t.Fatalf("R04-last-page expected a multi-page set, pages=%d last=%d", pages, len(lastIDs))
		}
		summary := h.pgPerfRunPoint(pgPerfTimedSpec("R04-last-page", "R04", 1, nil, "replay of the terminal keyset page"), func(ctx context.Context, worker int) (bool, error) {
			ids, next, err := pgPerfReadPage(ctx, service, pgPerfReadDefaultLimit, startCursor)
			if err != nil {
				return false, err
			}
			if next != nil {
				return false, errors.New("R04-last-page expected a nil next cursor")
			}
			if !pgPerfEqualIDs(ids, lastIDs) {
				return false, errors.New("R04-last-page replay differs from the terminal page")
			}
			return true, nil
		})
		pgPerfReadAssertPoint(t, "R04-last-page", summary)
	})

	pgPerfSceneX01(t)
}

// pgPerfSceneX01 compares the empty and ten-row datasets as correctness-only
// baselines; it produces no performance sample.
func pgPerfSceneX01(t *testing.T) {
	t.Run("X01", func(t *testing.T) {
		t.Run("empty", func(t *testing.T) {
			h := newPgPerfHarness(t, pgPerfTierSmall, "x01a", func(options *pgPerfOptions) {
				options.seedMode = "empty"
			})
			service := h.queryService()
			ids, next, err := pgPerfReadPage(h.ctx, service, pgPerfReadDefaultLimit, "")
			if err != nil {
				t.Fatalf("X01 empty list failed: %v", err)
			}
			if next != nil || len(ids) != 0 {
				t.Fatalf("X01 empty expected empty items and nil cursor, got items=%d cursor=%t", len(ids), next != nil)
			}
		})
		t.Run("ten", func(t *testing.T) {
			h := newPgPerfHarness(t, pgPerfTierSmall, "x01b", func(options *pgPerfOptions) {
				options.seedMode = "ten"
			})
			oracle := h.memberVisibleIDs(h.ctx, pgPerfReadOracleAll)
			if len(oracle) != 10 {
				t.Fatalf("X01 ten oracle expected 10 rows, got %d", len(oracle))
			}
			service := h.queryService()
			ids, next, err := pgPerfReadPage(h.ctx, service, pgPerfReadDefaultLimit, "")
			if err != nil {
				t.Fatalf("X01 ten list failed: %v", err)
			}
			if next != nil {
				t.Fatalf("X01 ten expected a terminal page")
			}
			if !pgPerfEqualIDs(ids, oracle) {
				t.Fatalf("X01 ten page differs from the SQL oracle")
			}
		})
	})
}

// pgPerfReadVisibleHarness builds a full fixture with every article visible at
// the requested hidden density and no pending deletions.
func pgPerfReadVisibleHarness(t *testing.T, tier pgPerfTier, caseCode string, density float64) *pgPerfHarness {
	t.Helper()
	return newPgPerfHarness(t, tier, caseCode, func(options *pgPerfOptions) {
		options.seedMode = "full"
		options.seedConfig = pgPerfSeedOverride(tier, "full", func(config *pgPerfSeedConfig) {
			config.Deletions = 0
			config.PendingOverride = 0
			config.HiddenDensity = density
		})
	})
}

func pgPerfReadDefaultFullHarness(t *testing.T, tier pgPerfTier, caseCode string) *pgPerfHarness {
	t.Helper()
	return newPgPerfHarness(t, tier, caseCode, func(options *pgPerfOptions) {
		options.seedMode = "full"
	})
}

type pgPerfReadDetailExpectation struct {
	id    string
	etag  string
	title string
	body  string
}

// pgPerfReadDetailExpectations derives expected detail fields and the strong
// ETag directly from the retained rows, never from the query service.
func pgPerfReadDetailExpectations(t testing.TB, h *pgPerfHarness, ids []string) []pgPerfReadDetailExpectation {
	t.Helper()
	expected := make([]pgPerfReadDetailExpectation, 0, len(ids))
	for _, id := range ids {
		record := h.articleByID(h.ctx, id)
		etag, err := article.StrongETag(record)
		if err != nil {
			t.Fatalf("strong etag for %s failed: %v", id, err)
		}
		expected = append(expected, pgPerfReadDetailExpectation{id: id, etag: etag, title: record.Title, body: record.Body})
	}
	return expected
}

func pgPerfReadContainsID(ids []string, want string) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

func pgPerfReadLimit100(t *testing.T) {
	h := pgPerfReadVisibleHarness(t, pgPerfTierMedium, "r01b", 0)
	oracle := h.memberVisibleIDs(h.ctx, pgPerfReadMaxLimit)
	if len(oracle) != pgPerfReadMaxLimit {
		t.Fatalf("R01-limit100 oracle expected %d rows, got %d", pgPerfReadMaxLimit, len(oracle))
	}
	service := h.queryService()
	summary := h.pgPerfRunPoint(pgPerfTimedSpec("R01-limit100", "R01", 1, nil, "member list first page limit 100, hidden density 0"), func(ctx context.Context, worker int) (bool, error) {
		ids, next, err := pgPerfReadPage(ctx, service, pgPerfReadMaxLimit, "")
		if err != nil {
			return false, err
		}
		if next == nil {
			return false, errors.New("R01-limit100 expected a non-nil next cursor")
		}
		if !pgPerfEqualIDs(ids, oracle) {
			return false, errors.New("R01-limit100 first page differs from the SQL oracle")
		}
		return true, nil
	})
	pgPerfReadAssertPoint(t, "R01-limit100", summary)
}

func pgPerfReadContinuous(t *testing.T) {
	h := pgPerfReadVisibleHarness(t, pgPerfTierMedium, "r02a", 0)
	oracle := h.memberVisibleIDs(h.ctx, pgPerfReadOracleAll)
	if len(oracle) < 200 {
		t.Fatalf("R02-continuous oracle expected at least 200 rows, got %d", len(oracle))
	}
	service := h.queryService()
	summary := h.pgPerfRunPoint(pgPerfTimedSpec("R02-continuous", "R02", 1, nil, "continuous keyset walk to the tail, limit 100"), func(ctx context.Context, worker int) (bool, error) {
		pages, err := pgPerfReadCompareWalk(ctx, service, pgPerfReadMaxLimit, oracle)
		if err != nil {
			return false, err
		}
		if pages < 2 {
			return false, fmt.Errorf("R02-continuous expected a multi-page walk, got %d page", pages)
		}
		return true, nil
	})
	pgPerfReadAssertPoint(t, "R02-continuous", summary)
}

func pgPerfSceneR03(t *testing.T, density float64, caseCode, point string) {
	h := pgPerfReadVisibleHarness(t, pgPerfTierMedium, caseCode, density)
	oracle := h.memberVisibleIDs(h.ctx, pgPerfReadOracleAll)
	if density >= 1 {
		if len(oracle) != 0 {
			t.Fatalf("%s oracle expected no visible row, got %d", point, len(oracle))
		}
		service := h.queryService()
		summary := h.pgPerfRunPoint(pgPerfTimedSpec(point, "R03", 1, nil, "fully hidden dataset list"), func(ctx context.Context, worker int) (bool, error) {
			ids, next, err := pgPerfReadPage(ctx, service, pgPerfReadDefaultLimit, "")
			if err != nil {
				return false, err
			}
			if next != nil || len(ids) != 0 {
				return false, fmt.Errorf("%s expected empty items and nil cursor, got items=%d cursor=%t", point, len(ids), next != nil)
			}
			return true, nil
		})
		pgPerfReadAssertPoint(t, point, summary)
		return
	}
	if len(oracle) == 0 {
		t.Fatalf("%s oracle expected visible rows", point)
	}
	service := h.queryService()
	summary := h.pgPerfRunPoint(pgPerfTimedSpec(point, "R03", 1, nil, "dense hidden mix keyset walk, limit 100"), func(ctx context.Context, worker int) (bool, error) {
		pages, err := pgPerfReadCompareWalk(ctx, service, pgPerfReadMaxLimit, oracle)
		if err != nil {
			return false, err
		}
		if pages < 2 {
			return false, fmt.Errorf("%s expected a multi-page walk, got %d page", point, pages)
		}
		return true, nil
	})
	pgPerfReadAssertPoint(t, point, summary)
}

func pgPerfReadR05(t *testing.T) {
	h := pgPerfReadVisibleHarness(t, pgPerfTierMedium, "r05a", 0)
	if h.scalar("SELECT count(*) FROM (SELECT created_at FROM articles GROUP BY created_at HAVING count(*)>1) ties") == 0 {
		t.Fatal("R05 requires timestamp ties")
	}
	summary := h.pgPerfRunPoint(pgPerfTimedSpec("R05", "R05", 1, nil, "page1 then real insert/edit/author-state change then keyset continuation"),
		func(ctx context.Context, _ int) (bool, error) {
			err := pgPerfCrossPageIteration(ctx, h)
			return err == nil, err
		})
	pgPerfReadAssertPoint(t, "R05", summary)
}

func pgPerfCrossPageIteration(ctx context.Context, h *pgPerfHarness) (resultErr error) {
	service := h.queryService()
	first, next, err := pgPerfReadPage(ctx, service, pgPerfReadMaxLimit, "")
	if err != nil {
		return err
	}
	if next == nil || len(first) == 0 {
		return errors.New("R05 first page has no continuation")
	}
	boundary := first[len(first)-1]
	member, err := h.memberPrincipal(ctx)
	if err != nil {
		return err
	}
	command := h.commandService()
	created, err := command.Create(ctx, member, article.CreateArticleInput{Title: "pgperf R05 between pages", Body: pgPerfScenarioBody()})
	if err != nil {
		return err
	}
	var editID, suspended string
	var version int64
	if err := h.pool.QueryRow(ctx, `SELECT id::text,version FROM articles WHERE author_id=$1::uuid AND deletion_state='active'
		AND (created_at,id)<(SELECT created_at,id FROM articles WHERE id=$2::uuid) ORDER BY created_at DESC,id DESC LIMIT 1`, member.UserID, boundary).Scan(&editID, &version); err != nil {
		return err
	}
	if _, err := command.ReplaceContent(ctx, member, editID, version, article.CreateArticleInput{Title: "pgperf R05 edited between pages", Body: pgPerfScenarioBody()}); err != nil {
		return err
	}
	if err := h.pool.QueryRow(ctx, `SELECT u.id::text FROM users u JOIN articles a ON a.author_id=u.id
		WHERE u.status='active' AND u.id<>$1::uuid AND a.deletion_state='active'
		AND (a.created_at,a.id)<(SELECT created_at,id FROM articles WHERE id=$2::uuid) LIMIT 1`, member.UserID, boundary).Scan(&suspended); err != nil {
		return err
	}
	if _, err := h.pool.Exec(ctx, "UPDATE users SET status='suspended',updated_at=clock_timestamp() WHERE id=$1::uuid", suspended); err != nil {
		return err
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, err := h.pool.Exec(closeCtx, "UPDATE users SET status='active',updated_at=clock_timestamp() WHERE id=$1::uuid", suspended)
		resultErr = errors.Join(resultErr, err)
	}()
	expected, err := pgPerfOracleAfter(ctx, h, boundary)
	if err != nil {
		return err
	}
	if pgPerfReadContainsID(expected, created.ID) {
		return errors.New("R05 new head item incorrectly entered continuation oracle")
	}
	pages, err := pgPerfReadCompareFromCursor(ctx, service, pgPerfReadMaxLimit, *next, expected)
	if err != nil {
		return err
	}
	h.evidence.writeJSONExclusive(filepath.Join("interleavings", fmt.Sprintf("%s.%d.json", h.schema, time.Now().UnixNano())), map[string]any{
		"order": "page1 -> insert/edit/suspend -> remaining pages", "boundary_id": boundary, "inserted_id": created.ID,
		"edited_id": editID, "suspended_user": suspended, "expected_remaining": len(expected), "remaining_pages": pages,
	})
	return nil
}

func pgPerfOracleAfter(ctx context.Context, h *pgPerfHarness, boundary string) ([]string, error) {
	rows, err := h.pool.Query(ctx, `SELECT a.id::text FROM articles a LEFT JOIN users u ON u.id=a.author_id
		WHERE a.deletion_state='active' AND a.publication_state='published' AND a.violation_state='clear'
		AND (a.author_id IS NULL OR u.status='active')
		AND (a.created_at,a.id)<(SELECT created_at,id FROM articles WHERE id=$1::uuid)
		ORDER BY a.created_at DESC,a.id DESC`, boundary)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func pgPerfReadR06(t *testing.T) {
	h := pgPerfReadDefaultFullHarness(t, pgPerfTierMedium, "r06a")
	var author string
	if err := h.pool.QueryRow(h.ctx, `SELECT a.author_id::text FROM articles a JOIN users u ON u.id=a.author_id
		WHERE a.deletion_state='active' AND a.publication_state='published' AND a.violation_state='clear'
			AND u.status='active' GROUP BY a.author_id HAVING count(*)>=3 ORDER BY a.author_id LIMIT 1`).Scan(&author); err != nil {
		t.Fatalf("R06 pick author failed: %v", err)
	}
	rows, err := h.pool.Query(h.ctx, `SELECT id::text FROM articles
		WHERE author_id=$1::uuid AND deletion_state='active' AND publication_state='published'
			AND violation_state='clear' ORDER BY id LIMIT 3`, author)
	if err != nil {
		t.Fatalf("R06 pick articles failed: %v", err)
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			t.Fatalf("R06 scan article: %v", err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	if len(ids) != pgPerfReadDetailRows {
		t.Fatalf("R06 expected %d author articles, got %d", pgPerfReadDetailRows, len(ids))
	}
	expected := pgPerfReadDetailExpectations(t, h, ids)
	principal := pgPerfPrincipal(author, "member")
	service := h.queryService()
	summary := h.pgPerfRunPoint(pgPerfTimedSpec("R06", "R06", 1, nil, "author-controlled detail repeated reads"), func(ctx context.Context, worker int) (bool, error) {
		for _, want := range expected {
			view, etag, err := service.GetAuthorArticle(ctx, principal, want.id)
			if err != nil {
				return false, err
			}
			if view.Title != want.title || view.Body != want.body || view.ModerationStatus != article.ModerationNormal || !view.Editable || etag != want.etag {
				return false, fmt.Errorf("R06 author projection mismatch for %s", want.id)
			}
			again, etagAgain, err := service.GetAuthorArticle(ctx, principal, want.id)
			if err != nil {
				return false, err
			}
			if etagAgain != etag || again.Title != view.Title || again.Body != view.Body || again.ModerationStatus != view.ModerationStatus || again.Editable != view.Editable {
				return false, fmt.Errorf("R06 repeated author read drifted for %s", want.id)
			}
		}
		return true, nil
	})
	pgPerfReadAssertPoint(t, "R06", summary)
	var pendingID, pendingAuthor string
	if err := h.pool.QueryRow(h.ctx, `SELECT id::text, author_id::text FROM articles
		WHERE deletion_state='deletion_pending' AND author_id IS NOT NULL ORDER BY id LIMIT 1`).Scan(&pendingID, &pendingAuthor); err != nil {
		t.Fatalf("R06 pick pending article failed: %v", err)
	}
	if _, _, err := service.GetAuthorArticle(h.ctx, pgPerfPrincipal(pendingAuthor, "member"), pendingID); !errors.Is(err, article.ErrArticleUnavailable) {
		t.Fatalf("R06 pending author detail = %v, want ErrArticleUnavailable", err)
	}
	if _, _, err := service.GetAuthorArticle(h.ctx, principal, pgPerfUUID(h.schema, "r06-missing")); !errors.Is(err, article.ErrArticleUnavailable) {
		t.Fatalf("R06 missing author detail = %v, want ErrArticleUnavailable", err)
	}
}

func pgPerfReadR07(t *testing.T) {
	h := pgPerfReadDefaultFullHarness(t, pgPerfTierMedium, "r07a")
	ids := h.memberVisibleIDs(h.ctx, pgPerfReadDetailRows)
	if len(ids) != pgPerfReadDetailRows {
		t.Fatalf("R07 expected %d visible rows, got %d", pgPerfReadDetailRows, len(ids))
	}
	expected := pgPerfReadDetailExpectations(t, h, ids)
	service := h.queryService()
	summary := h.pgPerfRunPoint(pgPerfTimedSpec("R07", "R07", 1, nil, "member detail repeated reads with strong ETag"), func(ctx context.Context, worker int) (bool, error) {
		for _, want := range expected {
			view, etag, err := service.GetArticle(ctx, want.id)
			if err != nil {
				return false, err
			}
			if view.ID != want.id || view.Title != want.title || view.Body != want.body || etag != want.etag {
				return false, fmt.Errorf("R07 member projection mismatch for %s", want.id)
			}
			again, etagAgain, err := service.GetArticle(ctx, want.id)
			if err != nil {
				return false, err
			}
			if etagAgain != etag || again.ID != view.ID || again.Title != view.Title || again.Body != view.Body {
				return false, fmt.Errorf("R07 repeated member read drifted for %s", want.id)
			}
		}
		return true, nil
	})
	pgPerfReadAssertPoint(t, "R07", summary)
	var hiddenID string
	if err := h.pool.QueryRow(h.ctx, "SELECT id::text FROM articles WHERE deletion_state='deletion_pending' ORDER BY id LIMIT 1").Scan(&hiddenID); err != nil {
		t.Fatalf("R07 pick hidden article failed: %v", err)
	}
	if _, _, err := service.GetArticle(h.ctx, hiddenID); !errors.Is(err, article.ErrArticleUnavailable) {
		t.Fatalf("R07 hidden detail = %v, want ErrArticleUnavailable", err)
	}
	if _, _, err := service.GetArticle(h.ctx, pgPerfUUID(h.schema, "r07-missing")); !errors.Is(err, article.ErrArticleUnavailable) {
		t.Fatalf("R07 missing detail = %v, want ErrArticleUnavailable", err)
	}
}

// pgPerfSceneX02 walks the whole minimal fixture as a correctness-only check.
func pgPerfSceneX02(t *testing.T) {
	t.Run("X02", func(t *testing.T) {
		h := newPgPerfHarness(t, pgPerfTierMedium, "x02a", func(options *pgPerfOptions) {
			options.seedMode = "minimal"
		})
		oracle := h.memberVisibleIDs(h.ctx, pgPerfReadOracleAll)
		if len(oracle) < 40 {
			t.Fatalf("X02 expected a multi-page oracle, got %d rows", len(oracle))
		}
		service := h.queryService()
		pages, err := pgPerfReadCompareWalk(h.ctx, service, pgPerfReadDefaultLimit, oracle)
		if err != nil {
			t.Fatalf("X02 walk failed: %v", err)
		}
		if pages < 2 {
			t.Fatalf("X02 expected a multi-page walk, got %d page", pages)
		}
	})
}

// pgPerfSceneX03 verifies a fully hidden minimal fixture as correctness-only.
func pgPerfSceneX03(t *testing.T) {
	t.Run("X03", func(t *testing.T) {
		h := newPgPerfHarness(t, pgPerfTierMedium, "x03a", func(options *pgPerfOptions) {
			options.seedMode = "minimal"
			options.seedConfig = pgPerfSeedOverride(pgPerfTierMedium, "minimal", func(config *pgPerfSeedConfig) {
				config.HiddenDensity = 1
			})
		})
		if oracle := h.memberVisibleIDs(h.ctx, pgPerfReadOracleAll); len(oracle) != 0 {
			t.Fatalf("X03 oracle expected no visible row, got %d", len(oracle))
		}
		service := h.queryService()
		ids, next, err := pgPerfReadPage(h.ctx, service, pgPerfReadDefaultLimit, "")
		if err != nil {
			t.Fatalf("X03 list failed: %v", err)
		}
		if next != nil || len(ids) != 0 {
			t.Fatalf("X03 expected empty items and nil cursor, got items=%d cursor=%t", len(ids), next != nil)
		}
	})
}

func runPgPerfMediumRead(t *testing.T) {
	t.Run("R01-limit100", pgPerfReadLimit100)
	t.Run("R02-continuous", pgPerfReadContinuous)
	t.Run("R03-density50", func(t *testing.T) { pgPerfSceneR03(t, 0.5, "d50", "R03-density50") })
	t.Run("R03-density90", func(t *testing.T) { pgPerfSceneR03(t, 0.9, "d90", "R03-density90") })
	t.Run("R03-density100", func(t *testing.T) { pgPerfSceneR03(t, 1, "d100", "R03-density100") })
	t.Run("R05", pgPerfReadR05)
	t.Run("R06", pgPerfReadR06)
	t.Run("R07", pgPerfReadR07)
	pgPerfSceneX02(t)
	pgPerfSceneX03(t)
}

func runPgPerfLargeRead(t *testing.T) {
	source := pgPerfLargeGate(t)
	pgPerfLargeClone(t, source, "l03", "L03-R01-limit100", "L03", 1,
		func(h *pgPerfHarness) func(context.Context, int) (bool, error) {
			oracle := h.memberVisibleIDs(h.ctx, pgPerfReadMaxLimit)
			if len(oracle) != pgPerfReadMaxLimit {
				h.t.Fatalf("L03 oracle expected %d rows, got %d", pgPerfReadMaxLimit, len(oracle))
			}
			service := h.queryService()
			return func(ctx context.Context, worker int) (bool, error) {
				ids, next, err := pgPerfReadPage(ctx, service, pgPerfReadMaxLimit, "")
				if err != nil {
					return false, err
				}
				if next == nil || !pgPerfEqualIDs(ids, oracle) {
					return false, errors.New("L03 first page differs from the SQL oracle")
				}
				return true, nil
			}
		})
	pgPerfLargeClone(t, source, "l04", "L04-R02-tail", "L04", 1,
		func(h *pgPerfHarness) func(context.Context, int) (bool, error) {
			service := h.queryService()
			startCursor, lastIDs, pages, err := pgPerfReadLastPage(h.ctx, service, pgPerfReadMaxLimit)
			if err != nil {
				h.t.Fatalf("L04 tail walk failed: %v", err)
			}
			if pages < 2 || len(lastIDs) == 0 {
				h.t.Fatalf("L04 expected a multi-page tail, pages=%d last=%d", pages, len(lastIDs))
			}
			return func(ctx context.Context, worker int) (bool, error) {
				ids, next, err := pgPerfReadPage(ctx, service, pgPerfReadMaxLimit, startCursor)
				if err != nil {
					return false, err
				}
				if next != nil || !pgPerfEqualIDs(ids, lastIDs) {
					return false, errors.New("L04 terminal page replay differs")
				}
				return true, nil
			}
		})
	pgPerfLargeClone(t, source, "l05", "L05-R03-density100", "L05", 1,
		func(h *pgPerfHarness) func(context.Context, int) (bool, error) {
			tx := h.begin(h.ctx)
			if _, err := tx.Exec(h.ctx, "SET LOCAL statement_timeout=0"); err != nil {
				_ = tx.Rollback(context.Background())
				h.t.Fatalf("L05 disable timeout failed: %v", err)
			}
			if _, err := tx.Exec(h.ctx, "UPDATE users SET status='suspended' WHERE status='active'"); err != nil {
				_ = tx.Rollback(context.Background())
				h.t.Fatalf("L05 suspend users failed: %v", err)
			}
			if err := tx.Commit(h.ctx); err != nil {
				h.t.Fatalf("L05 commit suspension failed: %v", err)
			}
			h.exec("ANALYZE users")
			if oracle := h.memberVisibleIDs(h.ctx, pgPerfReadOracleAll); len(oracle) != 0 {
				h.t.Fatalf("L05 oracle expected no visible row, got %d", len(oracle))
			}
			service := h.queryService()
			return func(ctx context.Context, worker int) (bool, error) {
				ids, next, err := pgPerfReadPage(ctx, service, pgPerfReadDefaultLimit, "")
				if err != nil {
					return false, err
				}
				if next != nil || len(ids) != 0 {
					return false, fmt.Errorf("L05 expected empty items and nil cursor, got items=%d cursor=%t", len(ids), next != nil)
				}
				return true, nil
			}
		})
	pgPerfLargeClone(t, source, "l06", "L06-R07-member-detail", "L06", 1,
		func(h *pgPerfHarness) func(context.Context, int) (bool, error) {
			ids := h.memberVisibleIDs(h.ctx, 1)
			if len(ids) != 1 {
				h.t.Fatalf("L06 expected one visible row, got %d", len(ids))
			}
			want := pgPerfReadDetailExpectations(h.t, h, ids)[0]
			service := h.queryService()
			return func(ctx context.Context, worker int) (bool, error) {
				view, etag, err := service.GetArticle(ctx, want.id)
				if err != nil {
					return false, err
				}
				if view.ID != want.id || view.Title != want.title || etag != want.etag {
					return false, errors.New("L06 member detail differs from the retained row")
				}
				return true, nil
			}
		})
}
