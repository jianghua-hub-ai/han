//go:build integration

package articleintegration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"han/internal/article"
	"han/internal/platform"
	"han/internal/session"
)

// This file owns the PGPERF-B scenes (W01-W04, E01-E03). Every scene builds its
// own retained schema through newPgPerfHarness; nothing here redefines the
// shared notice/admin helpers and nothing is dropped or truncated.

// pgPerfMemberPrincipal returns one active member principal from the fixture.
func pgPerfMemberPrincipal(h *pgPerfHarness) session.Principal {
	return pgPerfPrincipal(h.activeUserID(), "member")
}

// pgPerfAdminPrincipal returns a synthetic administrator identity. audit_events
// carries no user foreign key, so the administrator need not exist in users.
func pgPerfAdminPrincipal(h *pgPerfHarness) session.Principal {
	return pgPerfPrincipal(pgPerfUUID(h.schema, "pgperf-admin"), "admin")
}

func pgPerfTraceCtx(parent context.Context) (context.Context, error) {
	return platform.ContextWithTraceID(parent, noticeTrace)
}

func pgPerfTimedSpec(point, scene string, concurrency int, bucket func(int) string, note string) pgPerfPointSpec {
	return pgPerfPointSpec{Point: point, Scene: scene, Mode: pgPerfModeTimed3Rep,
		Concurrency: concurrency, Window: pgPerfMinWindow, Note: note, BucketFn: bucket}
}

func pgPerfSingleSpec(point, scene string, concurrency int, note string) pgPerfPointSpec {
	return pgPerfPointSpec{Point: point, Scene: scene, Mode: pgPerfModeSingleRun,
		Concurrency: concurrency, Window: pgPerfMinWindow, Note: note}
}

// pgPerfPlanOnly records EXPLAIN (FORMAT JSON) without executing the statement.
func pgPerfPlanOnly(h *pgPerfHarness, name, query string, args ...any) {
	h.t.Helper()
	rows, err := h.pool.Query(h.ctx, "EXPLAIN (FORMAT JSON) "+query, args...)
	if err != nil {
		h.t.Logf("pgperf plan-only unavailable for %s: %T", name, err)
		h.evidence.addBlocked("EXPLAIN plan-only unavailable for " + name)
		return
	}
	defer rows.Close()
	if rows.Next() {
		var plan any
		if err := rows.Scan(&plan); err == nil {
			h.evidence.writePlan(name, plan)
		}
	}
}

func pgPerfPageIDs(page article.ListArticlesPage) []string {
	ids := make([]string, 0, len(page.Items))
	for _, item := range page.Items {
		ids = append(ids, item.ID)
	}
	return ids
}

func pgPerfEqualIDs(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func pgPerfSeedOverride(tier pgPerfTier, mode string, tweak func(*pgPerfSeedConfig)) *pgPerfSeedConfig {
	config := pgPerfDefaultSeedConfig(tier, mode)
	if tweak != nil {
		tweak(&config)
	}
	return &config
}

// pgPerfScenarioBody is the ~1 KiB typical body shared by W01/W02/W03/E01-E03.
func pgPerfScenarioBody() string {
	return strings.Repeat("文", 340)
}

func runPgPerfSmallWrite(t *testing.T) {
	t.Run("W01-typical", func(t *testing.T) {
		pgPerfSceneW01(t, pgPerfTierSmall, "w01a", "W01-typical", false)
	})
	t.Run("W01-near-limit", func(t *testing.T) {
		pgPerfSceneW01(t, pgPerfTierSmall, "w01b", "W01-near-limit", true)
	})
}

func runPgPerfMediumWrite(t *testing.T) {
	t.Run("W02-c1", func(t *testing.T) { pgPerfSceneW02(t, 1, "w02a", "W02-c1") })
	t.Run("W02-c16", func(t *testing.T) { pgPerfSceneW02(t, 16, "w02b", "W02-c16") })
	t.Run("W03", func(t *testing.T) { pgPerfSceneReplace(t, pgPerfTierMedium, "w03", "W03", false) })
	t.Run("W04", func(t *testing.T) { pgPerfSceneReplace(t, pgPerfTierMedium, "w04", "W04", true) })
	t.Run("E01", func(t *testing.T) { pgPerfSceneE01(t) })
	t.Run("E02", func(t *testing.T) { pgPerfSceneE02(t) })
	t.Run("E03", func(t *testing.T) { pgPerfSceneE03(t) })
}

func runPgPerfLargeWrite(t *testing.T) {
	source := pgPerfLargeGate(t)
	pgPerfLargeClone(t, source, "l01", "L01-W01-near-limit", "L01", 1,
		func(h *pgPerfHarness) func(context.Context, int) (bool, error) {
			member := pgPerfMemberPrincipal(h)
			service := h.commandService()
			body := pgPerfNearLimitBody()
			return func(ctx context.Context, worker int) (bool, error) {
				if _, err := service.Create(ctx, member, article.CreateArticleInput{Title: "pgperf L01", Body: body}); err != nil {
					return false, err
				}
				return true, nil
			}
		})
	pgPerfLargeClone(t, source, "l02", "L02-W02-c16", "L02", 16,
		func(h *pgPerfHarness) func(context.Context, int) (bool, error) {
			member := pgPerfMemberPrincipal(h)
			service := h.commandService()
			bodies := []string{pgPerfScenarioBody(), pgPerfNearLimitBody()}
			return func(ctx context.Context, worker int) (bool, error) {
				if _, err := service.Create(ctx, member, article.CreateArticleInput{Title: "pgperf L02", Body: bodies[worker%2]}); err != nil {
					return false, err
				}
				return true, nil
			}
		})
}

// pgPerfLargeGate runs the L00 minimal harness gate once and returns the
// retained canonical 100k source schema used by the L01-L07 clones.
var pgPerfLargeGateOnce sync.Once

func pgPerfLargeGate(t *testing.T) string {
	t.Helper()
	pgPerfLargeGateOnce.Do(func() {
		t.Run("L00-harness-gate", func(t *testing.T) {
			h := newPgPerfHarness(t, pgPerfTierLarge, "l00", func(options *pgPerfOptions) {
				options.seedMode = "minimal"
			})
			h.pgPerfRunPoint(pgPerfSingleSpec("L00-harness-gate", "L00", 1, "large harness gate"), func(ctx context.Context, worker int) (bool, error) {
				var one int
				if err := h.pool.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
					return false, err
				}
				return true, nil
			})
		})
	})
	return pgPerfEnsureLargeSource(t)
}

func pgPerfLargeClone(t *testing.T, source, caseCode, point, scene string, concurrency int,
	build func(*pgPerfHarness) func(context.Context, int) (bool, error),
) {
	t.Helper()
	t.Run(point, func(t *testing.T) {
		h := newPgPerfHarness(t, pgPerfTierLarge, caseCode, func(options *pgPerfOptions) {
			options.skipSeed = true
		})
		h.pgPerfCloneFrom(source)
		h.pgPerfRunPoint(pgPerfSingleSpec(point, scene, concurrency, "cloned 100k canonical source"), build(h))
	})
}

func pgPerfSceneW01(t *testing.T, tier pgPerfTier, caseCode, point string, nearLimit bool) {
	h := newPgPerfHarness(t, tier, caseCode, func(options *pgPerfOptions) { options.seedMode = "full" })
	member := pgPerfMemberPrincipal(h)
	service := h.commandService()
	title := "pgperf " + point
	body := pgPerfScenarioBody()
	if nearLimit {
		body = pgPerfNearLimitBody()
	}
	expectedRunes := int64(len([]rune(body)))
	before := h.countRows(h.ctx, "articles")
	h.pgPerfRunPoint(pgPerfTimedSpec(point, "W01", 1, nil, "create article"), func(ctx context.Context, worker int) (bool, error) {
		if _, err := service.Create(ctx, member, article.CreateArticleInput{Title: title, Body: body}); err != nil {
			return false, err
		}
		return true, nil
	})
	created := h.countRows(h.ctx, "articles") - before
	if created <= 0 {
		t.Fatalf("%s created no article row", point)
	}
	var wrongVersion, wrongBody int64
	if err := h.pool.QueryRow(h.ctx, `SELECT count(*) FILTER (WHERE version<>1), count(*) FILTER (WHERE char_length(body)<>$2)
		FROM articles WHERE title=$1`, title, expectedRunes).Scan(&wrongVersion, &wrongBody); err != nil {
		t.Fatalf("%s verification query failed: %T", point, err)
	}
	if wrongVersion != 0 || wrongBody != 0 {
		t.Fatalf("%s correctness mismatch: wrong_version=%d wrong_body=%d created=%d", point, wrongVersion, wrongBody, created)
	}
	t.Logf("%s created=%d body_runes=%d version_ok=true", point, created, expectedRunes)
}

func pgPerfBodyClass(worker int, iteration int64) int {
	return (worker + int(iteration)) % 2
}

func TestPgPerfR02_W02SingleWorker_MixedBodies(t *testing.T) {
	seen := map[int]bool{}
	for iteration := int64(0); iteration < 8; iteration++ {
		seen[pgPerfBodyClass(0, iteration)] = true
	}
	if !seen[0] || !seen[1] {
		t.Fatalf("single worker failed mixed-body contract: classes=%v", seen)
	}
}

func TestPgPerfR02_W02Bodies_DirectedPG(t *testing.T) {
	e := pgPerfGlobalEvidence(t)
	t.Cleanup(func() { e.Finalize(t) })
	h := newPgPerfHarness(t, pgPerfTierSmall, "r2body", func(o *pgPerfOptions) { o.seedMode = "minimal" })
	bodies := []string{pgPerfScenarioBody(), pgPerfNearLimitBody()}
	service, member := h.commandService(), pgPerfMemberPrincipal(h)
	for _, workers := range []int{1, 16} {
		for repeat := 0; repeat < 3; repeat++ {
			start := make(chan struct{})
			done := make(chan error, workers)
			var classes [2]atomic.Int64
			for worker := 0; worker < workers; worker++ {
				h.group.Add(1)
				go func(worker int) {
					defer h.group.Done()
					<-start
					for iteration := int64(0); iteration < 2; iteration++ {
						class := pgPerfBodyClass(worker, iteration)
						body := bodies[class]
						created, err := service.Create(h.ctx, member, article.CreateArticleInput{Title: "directed W02 bodies", Body: body})
						if err != nil {
							done <- err
							return
						}
						var runes, bytes int
						if err := h.pool.QueryRow(h.ctx, "SELECT char_length(body),octet_length(body) FROM articles WHERE id=$1::uuid", created.ID).Scan(&runes, &bytes); err != nil {
							done <- err
							return
						}
						if runes != utf8.RuneCountInString(body) || bytes != len(body) {
							done <- errors.New("stored W02 rune/byte count mismatch")
							return
						}
						classes[class].Add(1)
					}
					done <- nil
				}(worker)
			}
			close(start)
			var failure error
			for range workers {
				failure = errors.Join(failure, <-done)
			}
			if failure != nil {
				t.Fatal(failure)
			}
			if classes[0].Load() != int64(workers) || classes[1].Load() != int64(workers) {
				t.Fatal("W02 repeat missing a body class")
			}
			h.evidence.writeJSONExclusive(fmt.Sprintf("regressions/W02.c%d.repeat%d.json", workers, repeat), map[string]any{"workers": workers, "repeat": repeat, "typical": classes[0].Load(), "near_limit": classes[1].Load(), "scope": "fixed-operation correctness regression; formal timing NOT RUN"})
		}
	}
	var seen sync.Map
	ctx := context.WithValue(h.ctx, pgPerfReadOnlyKey{}, true)
	phase := h.pgPerfPhaseContext(ctx, 100*time.Millisecond, 4, false, 0, func(ctx context.Context, worker int) (bool, error) {
		seen.Store(worker, true)
		var value int
		err := h.pool.QueryRow(ctx, "SELECT 1").Scan(&value)
		return err == nil, err
	})
	count := 0
	seen.Range(func(any, any) bool { count++; return true })
	if phase.errors != 0 || !phase.completed || count != 4 {
		t.Fatalf("four-worker diagnostic observation: seen=%d phase=%+v", count, phase)
	}
	h.evidence.writeJSONExclusive("regressions/W02.c4-diagnostic.json", map[string]any{"actual_workers": count, "observed_ns": phase.observed.Nanoseconds(), "scope": "executor observation only; formal three 10s warmups NOT RUN"})
}

func pgPerfSceneW02(t *testing.T, concurrency int, caseCode, point string) {
	h := newPgPerfHarness(t, pgPerfTierMedium, caseCode, func(options *pgPerfOptions) { options.seedMode = "full" })
	member := pgPerfMemberPrincipal(h)
	service := h.commandService()
	bodies := []string{pgPerfScenarioBody(), pgPerfNearLimitBody()}
	bucket := func(worker int) string {
		if worker%2 == 0 {
			return "typical"
		}
		return "near-limit"
	}
	before := h.countRows(h.ctx, "articles")
	spec := pgPerfTimedSpec(point, "W02", concurrency, bucket, "mixed typical/near-limit create; per-operation body buckets")
	if concurrency == 16 {
		spec.WarmupConcurrency = 4
	}
	summary := h.pgPerfRunPoint(spec, func(ctx context.Context, worker int) (bool, error) {
		if _, err := service.Create(ctx, member, article.CreateArticleInput{Title: "pgperf " + point, Body: bodies[worker%2]}); err != nil {
			return false, err
		}
		return true, nil
	})
	if len(summary.Buckets) != 2 || summary.Buckets[0].N == 0 || summary.Buckets[1].N == 0 {
		t.Fatalf("%s did not measure both actual body classes", point)
	}
	if created := h.countRows(h.ctx, "articles") - before; created <= 0 {
		t.Fatalf("%s created no article row", point)
	}
}

func pgPerfSceneReplace(t *testing.T, tier pgPerfTier, caseCode, point string, nearLimit bool) {
	h := newPgPerfHarness(t, tier, caseCode)
	author := h.activeUserIDs(1)[0]
	record := h.activeArticleFor(h.ctx, author)
	member := pgPerfPrincipal(author, "member")
	service := h.commandService()
	body := pgPerfScenarioBody()
	if nearLimit {
		body = pgPerfNearLimitBody()
	}
	var version atomic.Int64
	version.Store(record.Version)
	h.pgPerfRunPoint(pgPerfTimedSpec(point, "W03/W04", 1, nil, "replace article content"), func(ctx context.Context, worker int) (bool, error) {
		updated, err := service.ReplaceContent(ctx, member, record.ID, version.Load(), article.CreateArticleInput{Title: "pgperf " + point, Body: body})
		if err != nil {
			return false, err
		}
		version.Store(updated.Version)
		return true, nil
	})
	final := h.articleByID(h.ctx, record.ID)
	if final.Version < version.Load() {
		t.Fatalf("%s version regressed: %d < %d", point, final.Version, version.Load())
	}
	if final.PublicationState != record.PublicationState || final.ViolationState != record.ViolationState ||
		final.DeletionState != record.DeletionState || final.AuthorID == nil || *final.AuthorID != author {
		t.Fatalf("%s changed governance/ownership fields", point)
	}
	pgPerfPlanOnly(h, point+"-content-update", `UPDATE articles SET title=$2,body=$3,version=version+1,updated_at=clock_timestamp() WHERE id=$1::uuid`, record.ID, "pgperf "+point, body)
	t.Logf("%s final_version=%d governance_unchanged=true", point, final.Version)
}

func pgPerfSceneE01(t *testing.T) {
	h := newPgPerfHarness(t, pgPerfTierMedium, "e01", func(options *pgPerfOptions) { options.seedMode = "minimal" })
	member := pgPerfMemberPrincipal(h)
	service := h.commandService()
	h.pgPerfRunPoint(pgPerfSingleSpec("E01", "E01", 1, "same-version concurrent replace"), func(ctx context.Context, worker int) (bool, error) {
		created, err := service.Create(ctx, member, article.CreateArticleInput{Title: "pgperf e01", Body: "initial"})
		if err != nil {
			return false, err
		}
		start := make(chan struct{})
		results := make([]error, 2)
		var group sync.WaitGroup
		for index := 0; index < 2; index++ {
			group.Add(1)
			go func(index int) {
				defer group.Done()
				<-start
				_, err := service.ReplaceContent(ctx, member, created.ID, 1, article.CreateArticleInput{Title: "pgperf e01", Body: fmt.Sprintf("update-%d", index)})
				results[index] = err
			}(index)
		}
		close(start)
		group.Wait()
		var successes, conflicts int
		for _, result := range results {
			switch {
			case result == nil:
				successes++
			case errors.Is(result, article.ErrPreconditionFailed):
				conflicts++
			default:
				return false, result
			}
		}
		if successes != 1 || conflicts != 1 {
			return false, fmt.Errorf("e01 expected one success and one conflict, got %d/%d", successes, conflicts)
		}
		var version int64
		if err := h.pool.QueryRow(ctx, "SELECT version FROM articles WHERE id=$1::uuid", created.ID).Scan(&version); err != nil {
			return false, err
		}
		if version != 2 {
			return false, fmt.Errorf("e01 expected version 2, got %d", version)
		}
		return true, nil
	})
}

func pgPerfSceneE02(t *testing.T) {
	h := newPgPerfHarness(t, pgPerfTierMedium, "e02", func(options *pgPerfOptions) { options.seedMode = "minimal" })
	h.pgPerfRunPoint(pgPerfSingleSpec("E02", "E02", 1, "real concurrent content update versus moderation with row-lock barrier"),
		func(ctx context.Context, _ int) (bool, error) {
			err := pgPerfE02Iteration(ctx, h)
			return err == nil, err
		})
}

func pgPerfE02Iteration(ctx context.Context, h *pgPerfHarness) error {
	member, err := h.memberPrincipal(ctx)
	if err != nil {
		return err
	}
	admin := pgPerfAdminPrincipal(h)
	service := h.commandService()
	created, err := service.Create(ctx, member, article.CreateArticleInput{Title: "pgperf E02", Body: "initial"})
	if err != nil {
		return err
	}
	category := article.ViolationViolence
	results, err := h.raceRow(ctx, created.ID, [2]func(context.Context) error{
		func(ctx context.Context) error {
			_, err := service.ReplaceContent(ctx, member, created.ID, 1, article.CreateArticleInput{Title: "pgperf E02 update", Body: "updated"})
			return err
		},
		func(ctx context.Context) error {
			trace, err := pgPerfTraceCtx(ctx)
			if err != nil {
				return err
			}
			_, _, err = service.Moderate(trace, admin, created.ID, 1, &category)
			return err
		},
	})
	if err != nil {
		return err
	}
	if (results[0] == nil) == (results[1] == nil) {
		return fmt.Errorf("E02 expected one winner: %v", results)
	}
	loser := results[0]
	if loser == nil {
		loser = results[1]
	}
	if !errors.Is(loser, article.ErrPreconditionFailed) {
		return fmt.Errorf("E02 unexpected loser: %w", loser)
	}
	var version, marks, matching int64
	var state string
	if err := h.pool.QueryRow(ctx, "SELECT version,violation_state FROM articles WHERE id=$1::uuid", created.ID).Scan(&version, &state); err != nil {
		return err
	}
	if err := h.pool.QueryRow(ctx, `SELECT count(*),count(*) FILTER(WHERE action='article.violation.mark'
		AND metadata->>'category'=$2 AND (metadata->>'version')::bigint=2) FROM audit_events WHERE target_id=$1`,
		created.ID, string(category)).Scan(&marks, &matching); err != nil {
		return err
	}
	if version != 2 {
		return fmt.Errorf("E02 final version=%d", version)
	}
	if results[1] == nil && (marks != 1 || matching != 1 || state != "violating") {
		return errors.New("E02 moderation winner audit/category/version mismatch")
	}
	if results[0] == nil && (marks != 0 || state != "clear") {
		return errors.New("E02 update winner produced ghost moderation audit")
	}
	return nil
}

func pgPerfSceneE03(t *testing.T) {
	h := newPgPerfHarness(t, pgPerfTierMedium, "e03", func(options *pgPerfOptions) { options.seedMode = "minimal" })
	member := pgPerfMemberPrincipal(h)
	service := h.commandService()
	query := h.queryService()
	h.pgPerfRunPoint(pgPerfSingleSpec("E03", "E03", 1, "stale ETag retry"), func(ctx context.Context, worker int) (bool, error) {
		created, err := service.Create(ctx, member, article.CreateArticleInput{Title: "pgperf e03", Body: "initial"})
		if err != nil {
			return false, err
		}
		detail, staleETag, err := query.GetArticle(ctx, created.ID)
		if err != nil {
			return false, err
		}
		if staleETag == "" || detail.ID != created.ID {
			return false, errors.New("e03 missing member detail or ETag")
		}
		updated, err := service.ReplaceContent(ctx, member, created.ID, 1, article.CreateArticleInput{Title: "pgperf e03", Body: "first"})
		if err != nil {
			return false, err
		}
		freshETag, err := article.StrongETag(updated)
		if err != nil {
			return false, err
		}
		if freshETag == staleETag {
			return false, errors.New("e03 ETag did not change with version")
		}
		if _, err := service.ReplaceContent(ctx, member, created.ID, 1, article.CreateArticleInput{Title: "pgperf e03", Body: "stale"}); !errors.Is(err, article.ErrPreconditionFailed) {
			return false, fmt.Errorf("e03 stale retry: %v", err)
		}
		if _, err := service.ReplaceContent(ctx, member, created.ID, updated.Version, article.CreateArticleInput{Title: "pgperf e03", Body: "fresh"}); err != nil {
			return false, err
		}
		var version int64
		if err := h.pool.QueryRow(ctx, "SELECT version FROM articles WHERE id=$1::uuid", created.ID).Scan(&version); err != nil {
			return false, err
		}
		if version != 3 {
			return false, fmt.Errorf("e03 expected version 3, got %d", version)
		}
		return true, nil
	})
}
