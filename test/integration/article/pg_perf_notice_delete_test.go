//go:build integration

package articleintegration

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"han/internal/article"
	"han/internal/audit"
	"han/internal/mailer"

	"github.com/jackc/pgx/v5"
)

// This file owns the PGPERF-D scenes (D01-D04, N01-N04, X04-X09). X09 is the
// blocked inventory recorded by the harness and is never executed. Every scene
// builds its own retained schema; nothing here drops, truncates, or reuses a
// schema across scene points and no notice helper from the existing tests is
// redefined.

const (
	pgPerfClaimLease = 6 * time.Second

	pgPerfResetClaimSQL = `UPDATE transfer_notice_recipients
	SET status='pending',attempt_count=0,claimed_by=NULL,locked_until=NULL,next_attempt_at=NULL,
		last_attempt_at=NULL,failure_bucket=NULL,accepted_at=NULL,updated_at=clock_timestamp()
	WHERE notice_id=$1::uuid AND recipient_kind=$2 AND claimed_by=$3 AND status='processing'`

	pgPerfArticleSnapshotSQL = `SELECT jsonb_build_object(
		'version', a.version, 'title', a.title, 'body', a.body, 'updated_at', a.updated_at::text,
		'publication_state', a.publication_state, 'violation_state', a.violation_state,
		'deletion_state', a.deletion_state,
		'audit', (SELECT count(*) FROM audit_events e WHERE e.target_id::text = a.id::text))::text
		FROM articles a WHERE a.id=$1::uuid`
)

func runPgPerfSmallNoticeDelete(t *testing.T) {
	t.Run("D04-100", func(t *testing.T) {
		override := pgPerfSeedOverride(pgPerfTierSmall, "minimal", func(config *pgPerfSeedConfig) {
			config.Articles = 200
			config.Deletions = 0
			config.PendingOverride = 0
			config.HiddenDensity = 0
			config.Notices = 0
			config.Audits = 0
		})
		h := newPgPerfHarness(t, pgPerfTierSmall, "d04s", func(options *pgPerfOptions) {
			options.seedMode = "minimal"
			options.seedConfig = override
		})
		pgPerfRunD04(t, h, "D04-100", 100, 1)
	})
}

func runPgPerfMediumNoticeDelete(t *testing.T) {
	t.Run("D01", pgPerfSceneD01)
	t.Run("N01", pgPerfSceneN01)
	t.Run("N02", pgPerfSceneN02)
	t.Run("D02", pgPerfSceneD02)
	t.Run("D04-1000", func(t *testing.T) {
		override := pgPerfSeedOverride(pgPerfTierMedium, "minimal", func(config *pgPerfSeedConfig) {
			config.Articles = 1100
			config.Deletions = 0
			config.PendingOverride = 0
			config.HiddenDensity = 0
			config.Notices = 0
			config.Audits = 0
		})
		h := newPgPerfHarness(t, pgPerfTierMedium, "d04m", func(options *pgPerfOptions) {
			options.seedMode = "minimal"
			options.seedConfig = override
		})
		pgPerfRunD04(t, h, "D04-1000", 1000, 4)
	})
	t.Run("D04-10000", func(t *testing.T) {
		override := pgPerfSeedOverride(pgPerfTierMedium, "full", func(config *pgPerfSeedConfig) {
			config.Deletions = 0
			config.PendingOverride = 0
			config.HiddenDensity = 0
			config.Notices = 0
			config.Audits = 0
		})
		h := newPgPerfHarness(t, pgPerfTierMedium, "d04l", func(options *pgPerfOptions) {
			options.seedMode = "full"
			options.seedConfig = override
		})
		pgPerfRunD04(t, h, "D04-10000", 10000, 8)
	})
	t.Run("N03-1k", func(t *testing.T) {
		override := pgPerfSeedOverride(pgPerfTierMedium, "minimal", func(config *pgPerfSeedConfig) {
			config.Deletions = 0
			config.PendingOverride = 0
			config.HiddenDensity = 0
			config.Notices = 750
			config.Audits = 0
		})
		pgPerfSceneN03(t, pgPerfTierMedium, "n03a", "N03-1k", override)
	})
	t.Run("N03-10k", func(t *testing.T) {
		override := pgPerfSeedOverride(pgPerfTierMedium, "full", func(config *pgPerfSeedConfig) {
			config.Deletions = 0
			config.PendingOverride = 0
			config.HiddenDensity = 0
			config.Notices = 7500
			config.Audits = 0
		})
		pgPerfSceneN03(t, pgPerfTierMedium, "n03b", "N03-10k", override)
	})
	t.Run("N04", pgPerfSceneN04)
	t.Run("X04", pgPerfSceneX04)
	t.Run("X05", pgPerfSceneX05)
	t.Run("X06", pgPerfSceneX06)
	t.Run("X07", pgPerfSceneX07)
	t.Run("X08", pgPerfSceneX08)
	t.Run("D03", pgPerfSceneD03)
}

func runPgPerfLargeNoticeDelete(t *testing.T) {
	source := pgPerfLargeGate(t)
	t.Run("L07-D04-10000", func(t *testing.T) {
		h := newPgPerfHarness(t, pgPerfTierLarge, "l07", func(options *pgPerfOptions) {
			options.skipSeed = true
		})
		h.pgPerfCloneFrom(source)
		pgPerfRunD04(t, h, "L07-D04-10000", 10000, 8)
	})
}

func pgPerfArticleSnapshot(t *testing.T, h *pgPerfHarness, id string) string {
	t.Helper()
	var value string
	if err := h.pool.QueryRow(h.ctx, pgPerfArticleSnapshotSQL, id).Scan(&value); err != nil {
		t.Fatalf("snapshot article %s failed: %T", id, err)
	}
	return value
}

func pgPerfRunD04(t *testing.T, h *pgPerfHarness, point string, targets, concurrency int) {
	t.Helper()
	actor := h.activeUserID()
	summary := h.pgPerfRunPoint(pgPerfSingleSpec(point, "D04", 1,
		fmt.Sprintf("fresh committed T1 batches of %d articles, %d batch workers; latency/QPS unit is one batch including COPY preparation; no idempotent tail", targets, concurrency)),
		func(ctx context.Context, worker int) (bool, error) {
			err := pgPerfT1Batch(ctx, h, actor, targets, concurrency)
			return err == nil, err
		})
	pgPerfReadAssertPoint(t, point, summary)
	t.Logf("%s batch_size=%d batch_workers=%d measured_batches=%d actual_pending=%d", point, targets, concurrency,
		summary.Successes, h.scalar("SELECT count(*) FROM articles WHERE deletion_state='deletion_pending'"))
}

func pgPerfT1Batch(ctx context.Context, h *pgPerfHarness, actor string, targets, concurrency int) error {
	prepared := time.Now()
	ids := make([]string, targets)
	rows := make([][]any, targets)
	now := time.Now().UTC()
	for index := range ids {
		id, err := h.ids.NewID()
		if err != nil {
			return err
		}
		ids[index] = id
		rows[index] = []any{id, actor, "pgperf fresh T1", pgPerfScenarioBody(), "published", "clear", "active", int64(1), now, now}
	}
	columns := []string{"id", "author_id", "title", "body", "publication_state", "violation_state", "deletion_state", "version", "created_at", "updated_at"}
	if _, err := h.pool.CopyFrom(ctx, pgx.Identifier{"articles"}, columns, pgx.CopyFromRows(rows)); err != nil {
		return err
	}
	h.evidence.writeJSON(filepath.Join("workloads", ids[0]+".json"), map[string]any{
		"scene": "D04", "articles_prepared": targets, "first_article_id": ids[0],
		"preparation_ms": time.Since(prepared).Milliseconds(), "included_in_point_wallclock": true,
	})
	if err := pgPerfApplyT1Batch(ctx, h, ids, actor, concurrency); err != nil {
		return err
	}
	var valid int
	if err := h.pool.QueryRow(ctx, `SELECT count(*) FROM articles a JOIN deletion_operations d
		ON d.id=a.deletion_operation_id AND d.article_id=a.id
		WHERE a.id::text=ANY($1) AND a.deletion_state='deletion_pending' AND a.version=2
		AND d.status='pending_journal'`, ids).Scan(&valid); err != nil {
		return err
	}
	if valid != targets {
		return fmt.Errorf("T1 batch committed %d valid pending rows, want %d", valid, targets)
	}
	if _, _, err := h.queryService().GetArticle(ctx, ids[0]); !errors.Is(err, article.ErrArticleUnavailable) {
		return fmt.Errorf("T1 pending detail: %v, want ErrArticleUnavailable", err)
	}
	return nil
}

func pgPerfApplyT1Batch(ctx context.Context, h *pgPerfHarness, ids []string, actor string, workers int) error {
	var index atomic.Int64
	var group sync.WaitGroup
	failures := make(chan error, workers)
	for worker := 0; worker < workers; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for {
				next := int(index.Add(1)) - 1
				if next >= len(ids) {
					return
				}
				if _, err := h.beginT1(ctx, ids[next], actor, "member"); err != nil {
					failures <- err
					return
				}
			}
		}()
	}
	group.Wait()
	close(failures)
	var errs []error
	for err := range failures {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func pgPerfSceneD01(t *testing.T) {
	h := newPgPerfHarness(t, pgPerfTierMedium, "d01", func(options *pgPerfOptions) { options.seedMode = "full" })
	summary := h.pgPerfRunPoint(pgPerfTimedSpec("D01", "D01", 1, nil,
		"fresh Create plus real first T1 transition per operation; preparation included; no idempotent reuse"),
		func(ctx context.Context, worker int) (bool, error) {
			err := pgPerfFreshT1(ctx, h)
			return err == nil, err
		})
	pgPerfReadAssertPoint(t, "D01", summary)
}

func pgPerfFreshT1(ctx context.Context, h *pgPerfHarness) error {
	member, err := h.memberPrincipal(ctx)
	if err != nil {
		return err
	}
	created, err := h.commandService().Create(ctx, member, article.CreateArticleInput{Title: "pgperf D01 fresh", Body: pgPerfScenarioBody()})
	if err != nil {
		return err
	}
	operation, err := h.beginT1(ctx, created.ID, member.UserID, "member", 1)
	if err != nil {
		return err
	}
	if operation.ArticleID != created.ID || operation.Status != article.DeletionOperationPendingJournal {
		return errors.New("D01 first T1 operation mismatch")
	}
	var valid bool
	if err := h.pool.QueryRow(ctx, `SELECT deletion_state='deletion_pending' AND version=2 AND deletion_operation_id=$2::uuid
		FROM articles WHERE id=$1::uuid`, created.ID, operation.ID).Scan(&valid); err != nil {
		return err
	}
	if !valid {
		return errors.New("D01 fresh T1 did not commit pending/version/operation")
	}
	if _, _, err := h.queryService().GetArticle(ctx, created.ID); !errors.Is(err, article.ErrArticleUnavailable) {
		return errors.New("D01 pending detail remained visible")
	}
	return nil
}

func pgPerfSceneD02(t *testing.T) {
	h := newPgPerfHarness(t, pgPerfTierMedium, "d02", func(options *pgPerfOptions) { options.seedMode = "minimal" })
	h.pgPerfRunPoint(pgPerfSingleSpec("D02", "D02", 1, "two real contenders: versioned T1 and content update; both wait behind a witnessed row lock"),
		func(ctx context.Context, _ int) (bool, error) {
			err := pgPerfD02Iteration(ctx, h)
			return err == nil, err
		})
}

func pgPerfD02Iteration(ctx context.Context, h *pgPerfHarness) error {
	member, err := h.memberPrincipal(ctx)
	if err != nil {
		return err
	}
	service := h.commandService()
	created, err := service.Create(ctx, member, article.CreateArticleInput{Title: "pgperf D02", Body: pgPerfScenarioBody()})
	if err != nil {
		return err
	}
	results, err := h.raceRow(ctx, created.ID, [2]func(context.Context) error{
		func(ctx context.Context) error {
			_, err := h.beginT1(ctx, created.ID, member.UserID, "member", 1)
			return err
		},
		func(ctx context.Context) error {
			_, err := service.ReplaceContent(ctx, member, created.ID, 1, article.CreateArticleInput{Title: "pgperf D02 update", Body: "updated"})
			return err
		},
	})
	if err != nil {
		return err
	}
	if (results[0] == nil) == (results[1] == nil) {
		return fmt.Errorf("D02 expected one winner: %v", results)
	}
	loser := results[0]
	if results[0] == nil {
		loser = results[1]
	}
	if !errors.Is(loser, article.ErrPreconditionFailed) && !errors.Is(loser, article.ErrArticleUnavailable) {
		return fmt.Errorf("D02 unexpected loser: %w", loser)
	}
	var version int64
	var state string
	var operation *string
	if err := h.pool.QueryRow(ctx, "SELECT version,deletion_state,deletion_operation_id::text FROM articles WHERE id=$1::uuid", created.ID).Scan(&version, &state, &operation); err != nil {
		return err
	}
	if version != 2 {
		return fmt.Errorf("D02 final version=%d", version)
	}
	if results[0] == nil && (state != "deletion_pending" || operation == nil) {
		return errors.New("D02 T1 winner missing pending operation")
	}
	if results[1] == nil && (state != "active" || operation != nil) {
		return errors.New("D02 update winner left a deletion operation")
	}
	return nil
}

// pgPerfTxNow reads one transaction-visible wall clock and normalizes it the
// same way the notice repository normalizes lease timestamps.
func pgPerfTxNow(ctx context.Context, tx pgx.Tx) (time.Time, error) {
	var now time.Time
	if err := tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); err != nil {
		return time.Time{}, err
	}
	return now.UTC().Truncate(time.Microsecond), nil
}

// pgPerfNoticeID returns one retained notice id without assuming seed order.
func pgPerfNoticeID(t *testing.T, h *pgPerfHarness) string {
	t.Helper()
	var id string
	if err := h.pool.QueryRow(h.ctx, "SELECT id::text FROM transfer_notices ORDER BY created_at DESC, id LIMIT 1").Scan(&id); err != nil {
		t.Fatalf("notice id lookup failed: %T", err)
	}
	return id
}

func pgPerfSceneN01(t *testing.T) {
	h := newPgPerfHarness(t, pgPerfTierMedium, "n01", func(options *pgPerfOptions) {
		options.seedMode = "full"
		options.seedConfig = pgPerfSeedOverride(pgPerfTierMedium, "full", func(config *pgPerfSeedConfig) {
			config.Notices = 1
			config.Deletions = 0
			config.PendingOverride = 0
			config.HiddenDensity = 0
		})
	})
	repo := article.Repository{}
	if _, err := h.pool.Exec(h.ctx, `UPDATE transfer_notice_recipients
		SET status='accepted',attempt_count=1,last_attempt_at=created_at,accepted_at=created_at,
			claimed_by=NULL,locked_until=NULL,next_attempt_at=NULL,failure_bucket=NULL,updated_at=clock_timestamp()
		WHERE recipient_kind=$1`, article.RecipientFormerDeletedAuthor); err != nil {
		t.Fatalf("N01 normalize former recipient: %T", err)
	}
	due := h.scalar(`SELECT count(*) FROM transfer_notice_recipients
		WHERE attempt_count<max_attempts AND (
			status='pending' OR (status='retry_wait' AND next_attempt_at<=clock_timestamp())
			OR (status='processing' AND locked_until<=clock_timestamp()))`)
	if due != 1 {
		t.Fatalf("N01 expected exactly one due recipient, got %d", due)
	}

	var claimSeq atomic.Int64
	point := "N01"
	summary := h.pgPerfRunPoint(pgPerfTimedSpec(point, "N01", 1, nil,
		"single due recipient: FOR UPDATE SKIP LOCKED then CAS claim; fenced reset restores pending after each iteration"), func(ctx context.Context, worker int) (bool, error) {
		tx, err := h.beginOwned(ctx)
		if err != nil {
			return false, err
		}
		committed := false
		defer func() {
			if !committed {
				_ = tx.Rollback(context.Background())
			}
		}()
		_, before, found, err := repo.LockDueNoticeRecipient(ctx, tx)
		if err != nil {
			return false, err
		}
		if !found {
			return false, nil
		}
		now, err := pgPerfTxNow(ctx, tx)
		if err != nil {
			return false, err
		}
		claimWorker := pgPerfUUID(h.schema, "n01w", fmt.Sprintf("%d", claimSeq.Add(1)))
		after, changed, err := repo.ClaimNoticeRecipient(ctx, tx, before, claimWorker, pgPerfClaimLease, now)
		if err != nil {
			return false, err
		}
		if !changed {
			return false, nil
		}
		if err := tx.Commit(ctx); err != nil {
			return false, err
		}
		committed = true
		tag, err := h.pool.Exec(ctx, pgPerfResetClaimSQL, after.NoticeID, after.Kind, claimWorker)
		if err != nil {
			return false, err
		}
		if tag.RowsAffected() != 1 {
			return false, fmt.Errorf("N01 fenced reset affected %d rows", tag.RowsAffected())
		}
		return true, nil
	})
	pgPerfReadAssertPoint(t, point, summary)

	noticeID := pgPerfNoticeID(t, h)
	worker := pgPerfUUID(h.schema, "n01verify")
	tx := h.begin(h.ctx)
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(context.Background())
		}
	}()
	_, before, found, err := repo.LockNoticeRecipient(h.ctx, tx, noticeID, article.RecipientNewAuthor)
	if err != nil || !found {
		t.Fatalf("N01 verification lock failed: found=%t err=%T", found, err)
	}
	now, err := pgPerfTxNow(h.ctx, tx)
	if err != nil {
		t.Fatalf("N01 verification clock: %T", err)
	}
	after, changed, err := repo.ClaimNoticeRecipient(h.ctx, tx, before, worker, pgPerfClaimLease, now)
	if err != nil || !changed {
		t.Fatalf("N01 verification claim changed=%t err=%T", changed, err)
	}
	if err := tx.Commit(h.ctx); err != nil {
		t.Fatalf("N01 verification commit: %T", err)
	}
	committed = true
	if after.Status != article.RecipientProcessing || after.AttemptCount != 1 ||
		after.ClaimedBy == nil || *after.ClaimedBy != worker || after.NextAttemptAt != nil {
		t.Fatalf("N01 claim state=%s attempt=%d claimed_by=%v next=%v", after.Status, after.AttemptCount, after.ClaimedBy, after.NextAttemptAt)
	}
	if after.LockedUntil == nil {
		t.Fatal("N01 claim did not set locked_until")
	}
	if lease := after.LockedUntil.Sub(now); lease < pgPerfClaimLease-time.Second || lease > pgPerfClaimLease+time.Second {
		t.Fatalf("N01 lease %s outside %s tolerance", lease, pgPerfClaimLease)
	}
	tag, err := h.pool.Exec(h.ctx, pgPerfResetClaimSQL, noticeID, article.RecipientNewAuthor, worker)
	if err != nil {
		t.Fatalf("N01 verification reset: %T", err)
	}
	if tag.RowsAffected() != 1 {
		t.Fatalf("N01 verification reset affected %d rows", tag.RowsAffected())
	}
	var status string
	var attempt int64
	var claimed *string
	var locked *time.Time
	if err := h.pool.QueryRow(h.ctx, `SELECT status::text,attempt_count,claimed_by,locked_until
		FROM transfer_notice_recipients WHERE notice_id=$1::uuid AND recipient_kind=$2`,
		noticeID, article.RecipientNewAuthor).Scan(&status, &attempt, &claimed, &locked); err != nil {
		t.Fatalf("N01 verification readback: %T", err)
	}
	if status != string(article.RecipientPending) || attempt != 0 || claimed != nil || locked != nil {
		t.Fatalf("N01 reset state=%s attempt=%d claimed_by=%v locked_until=%v", status, attempt, claimed, locked)
	}
}

func pgPerfSceneN02(t *testing.T) {
	h := newPgPerfHarness(t, pgPerfTierMedium, "n02", func(options *pgPerfOptions) { options.seedMode = "full" })
	repo := article.Repository{}
	rows, err := h.pool.Query(h.ctx, "SELECT id::text FROM transfer_notices ORDER BY created_at DESC, id LIMIT 4")
	if err != nil {
		t.Fatalf("N02 notice ids: %T", err)
	}
	ids := make([]string, 0, 4)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			t.Fatalf("N02 notice id scan: %T", err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("N02 notice id rows: %T", err)
	}
	if len(ids) != 4 {
		t.Fatalf("N02 expected four notices, got %d", len(ids))
	}
	if _, err := h.pool.Exec(h.ctx, `UPDATE transfer_notice_recipients
		SET status='pending',attempt_count=0,claimed_by=NULL,locked_until=NULL,next_attempt_at=NULL,
			last_attempt_at=NULL,failure_bucket=NULL,accepted_at=NULL,updated_at=clock_timestamp()
		WHERE notice_id::text = ANY($1) AND recipient_kind=$2`, ids, article.RecipientNewAuthor); err != nil {
		t.Fatalf("N02 normalize recipients: %T", err)
	}

	type c4Result struct {
		worker string
		before article.TransferNoticeRecipient
		err    error
	}
	results := make([]c4Result, len(ids))
	start := make(chan struct{})
	var group sync.WaitGroup
	for index, id := range ids {
		group.Add(1)
		go func(index int, id string) {
			defer group.Done()
			worker := pgPerfUUID(h.schema, "n02c4", fmt.Sprintf("%d", index))
			tx, err := h.beginOwned(h.ctx)
			if err != nil {
				results[index].err = err
				return
			}
			committed := false
			defer func() {
				if !committed {
					_ = tx.Rollback(context.Background())
				}
			}()
			<-start
			_, before, found, err := repo.LockNoticeRecipient(h.ctx, tx, id, article.RecipientNewAuthor)
			if err != nil {
				results[index].err = err
				return
			}
			if !found {
				results[index].err = fmt.Errorf("N02 c4 notice %s not found", id)
				return
			}
			now, err := pgPerfTxNow(h.ctx, tx)
			if err != nil {
				results[index].err = err
				return
			}
			_, changed, err := repo.ClaimNoticeRecipient(h.ctx, tx, before, worker, pgPerfClaimLease, now)
			if err != nil {
				results[index].err = err
				return
			}
			if !changed {
				results[index].err = fmt.Errorf("N02 c4 claim for %s was not applied", id)
				return
			}
			if err := tx.Commit(h.ctx); err != nil {
				results[index].err = err
				return
			}
			committed = true
			results[index].worker = worker
			results[index].before = before
		}(index, id)
	}
	close(start)
	group.Wait()
	seen := map[string]bool{}
	for index, result := range results {
		if result.err != nil {
			t.Fatalf("N02 c4 worker %d: %T", index, result.err)
		}
		if result.worker == "" {
			t.Fatalf("N02 c4 worker %d produced no claim", index)
		}
		seen[result.worker] = true
	}
	if len(seen) != len(ids) {
		t.Fatalf("N02 c4 expected %d distinct claim owners, got %d", len(ids), len(seen))
	}
	var processing int64
	if err := h.pool.QueryRow(h.ctx, `SELECT count(*) FROM transfer_notice_recipients
		WHERE notice_id::text = ANY($1) AND recipient_kind=$2 AND status='processing' AND attempt_count=1`,
		ids, article.RecipientNewAuthor).Scan(&processing); err != nil {
		t.Fatalf("N02 c4 processing count: %T", err)
	}
	if processing != int64(len(ids)) {
		t.Fatalf("N02 c4 expected %d processing rows, got %d", len(ids), processing)
	}
	var distinct int64
	if err := h.pool.QueryRow(h.ctx, `SELECT count(DISTINCT claimed_by) FROM transfer_notice_recipients
		WHERE notice_id::text = ANY($1) AND recipient_kind=$2`, ids, article.RecipientNewAuthor).Scan(&distinct); err != nil {
		t.Fatalf("N02 c4 distinct owners: %T", err)
	}
	if distinct != int64(len(ids)) {
		t.Fatalf("N02 c4 expected %d distinct owners, got %d", len(ids), distinct)
	}

	stale := results[0].before
	staleTx := h.begin(h.ctx)
	_, changedStale, err := repo.ClaimNoticeRecipient(h.ctx, staleTx, stale,
		pgPerfUUID(h.schema, "n02stale"), pgPerfClaimLease, time.Now().UTC().Truncate(time.Microsecond))
	if err != nil {
		_ = staleTx.Rollback(context.Background())
		t.Fatalf("N02 stale claim: %T", err)
	}
	if changedStale {
		_ = staleTx.Rollback(context.Background())
		t.Fatal("N02 stale preimage claim was applied")
	}
	if err := staleTx.Rollback(h.ctx); err != nil {
		t.Fatalf("N02 stale rollback: %T", err)
	}
	for index, id := range ids {
		tag, err := h.pool.Exec(h.ctx, pgPerfResetClaimSQL, id, article.RecipientNewAuthor, results[index].worker)
		if err != nil {
			t.Fatalf("N02 c4 reset %d: %T", index, err)
		}
		if tag.RowsAffected() != 1 {
			t.Fatalf("N02 c4 reset %d affected %d rows", index, tag.RowsAffected())
		}
	}
	if pending := h.scalar(`SELECT count(*) FROM transfer_notice_recipients
		WHERE notice_id::text = ANY($1) AND recipient_kind=$2 AND status='pending'`, ids, article.RecipientNewAuthor); pending != int64(len(ids)) {
		t.Fatalf("N02 c4 reset left %d pending rows", pending)
	}

	var claimSeq atomic.Int64
	point := "N02"
	summary := h.pgPerfRunPoint(pgPerfTimedSpec(point, "N02", 2, nil,
		"two workers claim due rows with SKIP LOCKED and fenced reset; empty-scan outcomes are rejected samples"), func(ctx context.Context, worker int) (bool, error) {
		tx, err := h.beginOwned(ctx)
		if err != nil {
			return false, err
		}
		committed := false
		defer func() {
			if !committed {
				_ = tx.Rollback(context.Background())
			}
		}()
		_, before, found, err := repo.LockDueNoticeRecipient(ctx, tx)
		if err != nil {
			return false, err
		}
		if !found {
			return false, nil
		}
		now, err := pgPerfTxNow(ctx, tx)
		if err != nil {
			return false, err
		}
		claimWorker := pgPerfUUID(h.schema, "n02w", fmt.Sprintf("%d", claimSeq.Add(1)))
		after, changed, err := repo.ClaimNoticeRecipient(ctx, tx, before, claimWorker, pgPerfClaimLease, now)
		if err != nil {
			return false, err
		}
		if !changed {
			return false, nil
		}
		if err := tx.Commit(ctx); err != nil {
			return false, err
		}
		committed = true
		tag, err := h.pool.Exec(ctx, pgPerfResetClaimSQL, after.NoticeID, after.Kind, claimWorker)
		if err != nil {
			return false, err
		}
		if tag.RowsAffected() != 1 {
			return false, fmt.Errorf("N02 fenced reset affected %d rows", tag.RowsAffected())
		}
		return true, nil
	})
	if summary.Errors != 0 {
		t.Fatalf("N02 saw %d execution errors", summary.Errors)
	}
	if summary.Successes == 0 {
		t.Fatal("N02 saw no successful claim")
	}
	t.Logf("N02 claimed=%d rejected=%d attempts=%d", summary.Successes, summary.Rejected, summary.Attempts)
}

func pgPerfSceneN03(t *testing.T, tier pgPerfTier, caseCode, point string, override *pgPerfSeedConfig) {
	if override == nil {
		t.Fatal("N03 requires an explicit seed override")
	}
	seedConfig := override
	h := newPgPerfHarness(t, tier, caseCode, func(options *pgPerfOptions) {
		options.seedMode = "full"
		options.seedConfig = seedConfig
	})
	expectedDue := int64(seedConfig.Notices + (seedConfig.Notices+2)/3)
	due := h.scalar(`SELECT count(*) FROM transfer_notice_recipients
		WHERE attempt_count<max_attempts AND (
			status='pending' OR (status='retry_wait' AND next_attempt_at<=clock_timestamp())
			OR (status='processing' AND locked_until<=clock_timestamp()))`)
	if due != expectedDue {
		t.Fatalf("N03 expected %d due recipients, got %d", expectedDue, due)
	}
	var sent atomic.Int64
	var faultMode atomic.Int32
	cfg := noticeConfig()
	cfg.BatchLimit = 50
	dispatcher, err := article.NewTransferNoticeDispatcher(article.NoticeDispatcherDependencies{
		Transactions: pgPerfTransactor{pool: h.pool, owner: h},
		Repository:   article.Repository{},
		Gate:         pgPerfGate{pool: h.pool},
		Mailer: noticeSendFunc(func(context.Context, mailer.Message) error {
			switch faultMode.Load() {
			case 1:
				return errors.New("synthetic transport outcome unknown")
			case 2:
				return mailer.ErrInvalidMessage
			}
			sent.Add(1)
			return nil
		}),
		Audit:      audit.Store{},
		IDs:        h.ids,
		NewTraceID: func() (string, error) { return noticeTrace, nil },
		Logger:     &noticeLog{},
		WorkerID:   pgPerfUUID(h.schema, "n03worker"),
	}, cfg)
	if err != nil {
		t.Fatalf("N03 dispatcher: %T", err)
	}

	if _, err := h.pool.Exec(h.ctx, `CREATE TABLE pgperf_due AS SELECT notice_id,recipient_kind,0::bigint AS batch_no
		FROM transfer_notice_recipients WHERE status='pending'`); err != nil {
		t.Fatalf("N03 retain due fixture keys: %T", err)
	}
	var cycles int64
	summary := h.pgPerfRunPoint(pgPerfSingleSpec(point, "N03", 1,
		"fixed-window reconcile rounds with BatchLimit=50; each fully drained batch is followed by new unique notices/recipients; QPS unit is rounds including probes/COPY preparation"),
		func(ctx context.Context, worker int) (bool, error) {
			if err := dispatcher.Reconcile(ctx); err != nil {
				return false, err
			}
			var remaining, accepted int64
			if err := h.pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE r.status<>'accepted'),
				count(*) FILTER (WHERE r.status='accepted') FROM transfer_notice_recipients r
				JOIN pgperf_due d USING(notice_id,recipient_kind) WHERE d.batch_no=$1`, cycles).Scan(&remaining, &accepted); err != nil {
				return false, err
			}
			if remaining == 0 {
				if accepted != expectedDue || sent.Load() != (cycles+1)*expectedDue {
					return false, fmt.Errorf("N03 drain count mismatch: accepted=%d sent=%d cycles=%d", accepted, sent.Load(), cycles)
				}
				cycles++
				if err := pgPerfAppendNoticeBatch(ctx, h, seedConfig.Notices, cycles); err != nil {
					return false, err
				}
			}
			return true, nil
		})
	pgPerfReadAssertPoint(t, point, summary)
	acceptedAfter := h.scalar(`SELECT count(*) FROM transfer_notice_recipients r JOIN pgperf_due d USING(notice_id,recipient_kind) WHERE r.status='accepted'`)
	initialAccepted := h.scalar(`SELECT count(*) FROM transfer_notice_recipients r JOIN pgperf_due d USING(notice_id,recipient_kind)
		WHERE d.batch_no=0 AND r.status='accepted'`)
	failedInitial := h.scalar(`SELECT count(*) FROM transfer_notice_recipients r JOIN pgperf_due d USING(notice_id,recipient_kind)
		WHERE d.batch_no=0 AND r.status='failed_terminal'`)
	if err := pgPerfInitialDrainOutcome(cycles, initialAccepted, expectedDue, failedInitial); err != nil {
		t.Fatal(err)
	}
	if sent.Load() != acceptedAfter {
		t.Fatalf("N03 actual send count disagrees with completed and partial drains: sent=%d cycles=%d", sent.Load(), cycles)
	}
	t.Logf("N03 initial_due=%d completed_drains=%d actual_sends=%d", expectedDue, cycles, sent.Load())
	if err := pgPerfAppendNoticeBatch(h.ctx, h, seedConfig.Notices, cycles+1); err != nil {
		t.Fatalf("N03 fresh batch for failure correctness: %T", err)
	}
	faultMode.Store(1)
	if err := dispatcher.Reconcile(h.ctx); err != nil {
		t.Fatalf("N03 retry correctness: %T", err)
	}
	if h.scalar("SELECT count(*) FROM transfer_notice_recipients WHERE status='retry_wait' AND next_attempt_at>clock_timestamp()") == 0 {
		t.Fatal("N03 expected future retry_wait after synthetic transport failure")
	}
	faultMode.Store(2)
	if err := dispatcher.Reconcile(h.ctx); err != nil {
		t.Fatalf("N03 terminal correctness: %T", err)
	}
	if h.scalar("SELECT count(*) FROM transfer_notice_recipients WHERE status='failed_terminal'") == 0 {
		t.Fatal("N03 expected failed_terminal after synthetic invalid message")
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	dispatcher.BeginShutdown(shutdownCtx)
	dispatcher.Stop()
	if err := dispatcher.Wait(h.ctx); err != nil {
		t.Fatalf("N03 shutdown: %T", err)
	}
}

func pgPerfInitialDrainOutcome(cycles, accepted, expected, failed int64) error {
	if cycles < 1 || accepted != expected || failed != 0 {
		return fmt.Errorf("initial batch incomplete: cycles=%d accepted=%d expected=%d failed=%d", cycles, accepted, expected, failed)
	}
	return nil
}

func TestPgPerfR02_InitialDrain_RejectsPartialBatch(t *testing.T) {
	if err := pgPerfInitialDrainOutcome(0, 50, 10000, 0); err == nil {
		t.Fatal("50 matching accepted/sent rows incorrectly satisfy the full 10000-recipient batch")
	}
}

func pgPerfAppendNoticeBatch(ctx context.Context, h *pgPerfHarness, notices int, batch int64) error {
	prepared := time.Now()
	var articleID, authorID, formerID string
	if err := h.pool.QueryRow(ctx, "SELECT article_ref::text,new_author_id::text,former_deleted_user_id::text FROM transfer_notices LIMIT 1").Scan(&articleID, &authorID, &formerID); err != nil {
		return err
	}
	now := time.Now().UTC()
	noticeRows := make([][]any, 0, notices)
	recipientRows := make([][]any, 0, notices*2)
	keys := make([][]any, 0, notices*2)
	for index := 0; index < notices; index++ {
		id, err := h.ids.NewID()
		if err != nil {
			return err
		}
		noticeRows = append(noticeRows, []any{id, articleID, authorID, formerID, now, now})
		kinds := []article.RecipientKind{article.RecipientNewAuthor, article.RecipientFormerDeletedAuthor}
		for _, kind := range kinds {
			status, attempt := "pending", int64(0)
			var lastAttempt, acceptedAt any
			if kind == article.RecipientFormerDeletedAuthor && index%3 != 0 {
				status = "accepted"
				attempt = 1
				lastAttempt = now
				acceptedAt = now
			} else {
				keys = append(keys, []any{id, kind, batch})
			}
			recipientRows = append(recipientRows, []any{id, kind, "synthetic@example.test", status, attempt, int64(3), lastAttempt, acceptedAt, now, now})
		}
	}
	tx, err := h.beginOwned(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"transfer_notices"}, []string{"id", "article_ref", "new_author_id", "former_deleted_user_id", "created_at", "updated_at"}, pgx.CopyFromRows(noticeRows)); err != nil {
		return err
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"transfer_notice_recipients"}, []string{"notice_id", "recipient_kind", "email_snapshot", "status", "attempt_count", "max_attempts", "last_attempt_at", "accepted_at", "created_at", "updated_at"}, pgx.CopyFromRows(recipientRows)); err != nil {
		return err
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"pgperf_due"}, []string{"notice_id", "recipient_kind", "batch_no"}, pgx.CopyFromRows(keys)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	h.evidence.writeJSONExclusive(filepath.Join("workloads", h.schema, fmt.Sprintf("N03.%s.batch%d.json", h.evidence.runID, batch)), map[string]any{
		"schema": h.schema, "case": h.caseCode, "run_id": h.evidence.runID,
		"scene": "N03", "batch": batch, "notices_prepared": len(noticeRows), "recipients_prepared": len(recipientRows),
		"due_recipients_prepared": len(keys),
		"preparation_ms":          time.Since(prepared).Milliseconds(), "included_in_point_wallclock": true,
	})
	return nil
}

func pgPerfSceneN04(t *testing.T) {
	h := newPgPerfHarness(t, pgPerfTierMedium, "n04", func(options *pgPerfOptions) {
		options.seedMode = "minimal"
		options.seedConfig = pgPerfSeedOverride(pgPerfTierMedium, "minimal", func(config *pgPerfSeedConfig) {
			config.Notices = 1
			config.Articles = 20
			config.Deletions = 0
			config.PendingOverride = 0
			config.HiddenDensity = 0
			config.Audits = 0
		})
	})
	if _, err := h.pool.Exec(h.ctx, `UPDATE transfer_notice_recipients SET status='accepted',attempt_count=1,
		last_attempt_at=created_at,accepted_at=clock_timestamp(),claimed_by=NULL,locked_until=NULL,next_attempt_at=NULL,failure_bucket=NULL`); err != nil {
		t.Fatalf("N04 isolate initial recipient fixture: %T", err)
	}
	summary := h.pgPerfRunPoint(pgPerfSingleSpec("N04", "N04", 1,
		"each iteration performs real claim, SQL lease rewind, takeover, stale CAS rejection and fenced fixture reset"), func(ctx context.Context, worker int) (bool, error) {
		err := pgPerfLeaseCycle(ctx, t, h)
		return err == nil, err
	})
	pgPerfReadAssertPoint(t, "N04", summary)
}

func pgPerfLeaseCycle(ctx context.Context, t *testing.T, h *pgPerfHarness) error {
	for _, sameWorker := range []bool{false, true} {
		if err := pgPerfLateNoticeResult(ctx, h, sameWorker); err != nil {
			return err
		}
	}
	return nil
}

func pgPerfNoticeDispatcher(h *pgPerfHarness, worker string, sender mailer.Mailer, batches ...int) (*article.TransferNoticeDispatcher, error) {
	config := noticeConfig()
	config.BatchLimit = 1
	if len(batches) > 0 {
		config.BatchLimit = batches[0]
	}
	return article.NewTransferNoticeDispatcher(article.NoticeDispatcherDependencies{
		Transactions: pgPerfTransactor{pool: h.pool, owner: h}, Repository: article.Repository{},
		Gate: pgPerfGate{pool: h.pool}, Mailer: sender, Audit: audit.Store{}, IDs: h.ids,
		NewTraceID: func() (string, error) { return noticeTrace, nil }, Logger: &noticeLog{}, WorkerID: worker,
	}, config)
}

func pgPerfFreshNotice(ctx context.Context, h *pgPerfHarness) (string, error) {
	id, err := h.ids.NewID()
	if err != nil {
		return "", err
	}
	tx, err := h.beginOwned(ctx)
	if err != nil {
		return "", err
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = tx.Rollback(closeCtx)
	}()
	_, err = tx.Exec(ctx, `INSERT INTO transfer_notices(id,article_ref,new_author_id,former_deleted_user_id,created_at,updated_at)
		SELECT $1::uuid,article_ref,new_author_id,former_deleted_user_id,clock_timestamp(),clock_timestamp()
		FROM transfer_notices ORDER BY created_at LIMIT 1`, id)
	if err != nil {
		return "", err
	}
	_, err = tx.Exec(ctx, `INSERT INTO transfer_notice_recipients(notice_id,recipient_kind,email_snapshot,status,
		attempt_count,max_attempts,last_attempt_at,accepted_at,created_at,updated_at)
		VALUES($1::uuid,'new_author','synthetic@example.test','pending',0,3,NULL,NULL,clock_timestamp(),clock_timestamp()),
		($1::uuid,'former_deleted_author','synthetic-former@example.test','accepted',1,3,clock_timestamp(),clock_timestamp(),clock_timestamp(),clock_timestamp())`, id)
	if err != nil {
		return "", err
	}
	return id, tx.Commit(ctx)
}

func pgPerfLateNoticeResult(parent context.Context, h *pgPerfHarness, sameWorker bool) (failure error) {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	id, err := pgPerfFreshNotice(ctx, h)
	if err != nil {
		return err
	}
	workerA, workerB := pgPerfUUID(h.schema, "late-a"), pgPerfUUID(h.schema, "late-b")
	if sameWorker {
		workerB = workerA
	}
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	var once sync.Once
	launched, joined := false, false
	defer func() {
		once.Do(func() { close(release) })
		cancel()
		if launched && !joined {
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				failure = errors.Join(failure, errors.New("N04 owned dispatcher failed to join"))
			}
		}
	}()
	dispatcherA, err := pgPerfNoticeDispatcher(h, workerA, noticeSendFunc(func(ctx context.Context, _ mailer.Message) error {
		close(started)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}))
	if err != nil {
		return err
	}
	var firstLease, nextLease time.Time
	var nextOwner string
	var nextAttempt int64
	dispatcherB, err := pgPerfNoticeDispatcher(h, workerB, noticeSendFunc(func(ctx context.Context, _ mailer.Message) error {
		return h.pool.QueryRow(ctx, `SELECT claimed_by,locked_until,attempt_count FROM transfer_notice_recipients
			WHERE notice_id=$1::uuid AND recipient_kind='new_author'`, id).Scan(&nextOwner, &nextLease, &nextAttempt)
	}))
	if err != nil {
		return err
	}
	h.group.Add(1)
	launched = true
	go func() { defer h.group.Done(); done <- dispatcherA.Reconcile(ctx) }()
	select {
	case <-started:
	case err := <-done:
		joined = true
		return fmt.Errorf("N04 first claim did not reach mailer: %w", err)
	case <-ctx.Done():
		return ctx.Err()
	}
	var firstOwner string
	var firstAttempt int64
	if err := h.pool.QueryRow(ctx, `SELECT claimed_by,locked_until,attempt_count FROM transfer_notice_recipients
		WHERE notice_id=$1::uuid AND recipient_kind='new_author'`, id).Scan(&firstOwner, &firstLease, &firstAttempt); err != nil {
		return err
	}
	if firstOwner != workerA || firstAttempt != 1 {
		return errors.New("N04 first ownership/attempt mismatch")
	}
	if _, err := h.pool.Exec(ctx, `UPDATE transfer_notice_recipients SET locked_until=clock_timestamp()-interval '1 second',
		updated_at=clock_timestamp() WHERE notice_id=$1::uuid AND recipient_kind='new_author'`, id); err != nil {
		return err
	}
	if err := dispatcherB.Reconcile(ctx); err != nil {
		return err
	}
	if nextOwner != workerB || nextAttempt != 2 || !nextLease.After(firstLease) {
		return errors.New("N04 successor ownership/attempt/lease fence mismatch")
	}
	var accepted time.Time
	var attempt int64
	if err := h.pool.QueryRow(ctx, `SELECT accepted_at,attempt_count FROM transfer_notice_recipients
		WHERE notice_id=$1::uuid AND recipient_kind='new_author' AND status='accepted'`, id).Scan(&accepted, &attempt); err != nil {
		return err
	}
	if attempt != 2 {
		return fmt.Errorf("N04 takeover attempt=%d want2", attempt)
	}
	once.Do(func() { close(release) })
	select {
	case err := <-done:
		joined = true
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	}
	var stable bool
	if err := h.pool.QueryRow(ctx, `SELECT status='accepted' AND attempt_count=2 AND accepted_at=$2
		AND claimed_by IS NULL AND locked_until IS NULL FROM transfer_notice_recipients
		WHERE notice_id=$1::uuid AND recipient_kind='new_author'`, id, accepted).Scan(&stable); err != nil {
		return err
	}
	if !stable {
		return errors.New("N04 late result overwrote the accepted successor")
	}
	h.evidence.writeJSONExclusive(filepath.Join("leases", id+".json"), map[string]any{
		"notice_id": id, "same_worker_id": sameWorker, "attempt": attempt, "accepted_at": accepted, "late_result_rejected": stable,
		"first_owner": firstOwner, "successor_owner": nextOwner, "first_attempt": firstAttempt, "successor_attempt": nextAttempt, "first_lease_until": firstLease, "successor_lease_until": nextLease,
	})
	return nil
}

const pgPerfCancelAfter = 150 * time.Millisecond

func pgPerfSceneX04(t *testing.T) {
	h := newPgPerfHarness(t, pgPerfTierMedium, "x04", func(options *pgPerfOptions) { options.seedMode = "minimal" })
	member := pgPerfMemberPrincipal(h)
	service := h.commandService()
	created, err := service.Create(h.ctx, member, article.CreateArticleInput{Title: "pgperf x04", Body: "initial"})
	if err != nil {
		t.Fatalf("X04 create: %T", err)
	}
	start := make(chan struct{})
	results := make([]error, 2)
	var group sync.WaitGroup
	for index := 0; index < 2; index++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			<-start
			_, err := service.ReplaceContent(h.ctx, member, created.ID, 1,
				article.CreateArticleInput{Title: "pgperf x04", Body: fmt.Sprintf("update-%d", index)})
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
			t.Fatalf("X04 unexpected error: %T", result)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("X04 expected one success and one conflict, got %d/%d", successes, conflicts)
	}
	var version int64
	if err := h.pool.QueryRow(h.ctx, "SELECT version FROM articles WHERE id=$1::uuid", created.ID).Scan(&version); err != nil {
		t.Fatalf("X04 version: %T", err)
	}
	if version != 2 {
		t.Fatalf("X04 expected version 2, got %d", version)
	}
}

func pgPerfSceneX05(t *testing.T) {
	h := newPgPerfHarness(t, pgPerfTierMedium, "x05", func(options *pgPerfOptions) { options.seedMode = "minimal" })
	member := pgPerfMemberPrincipal(h)
	created, err := h.commandService().Create(h.ctx, member, article.CreateArticleInput{Title: "pgperf x05", Body: "initial"})
	if err != nil {
		t.Fatalf("X05 create: %T", err)
	}
	before := pgPerfArticleSnapshot(t, h, created.ID)
	service, err := article.NewCommandService(article.CommandDependencies{
		Transactions: pgPerfTransactor{pool: h.pool, owner: h},
		Repository:   s22ContentFault{},
		IDs:          h.ids,
		Clock:        pgPerfClock{},
	})
	if err != nil {
		t.Fatalf("X05 command service: %T", err)
	}
	if _, err := service.ReplaceContent(h.ctx, member, created.ID, 1,
		article.CreateArticleInput{Title: "pgperf x05", Body: "post-write fault"}); err == nil {
		t.Fatal("X05 post-write fault was swallowed")
	}
	after := pgPerfArticleSnapshot(t, h, created.ID)
	if after != before {
		t.Fatalf("X05 snapshot changed after fault: before=%s after=%s", before, after)
	}
}

func pgPerfSceneX06(t *testing.T) {
	h := newPgPerfHarness(t, pgPerfTierMedium, "x06", func(options *pgPerfOptions) {
		options.seedMode = "minimal"
		options.maxConns = 2
	})
	point := "X06"
	summary := h.pgPerfRunPoint(pgPerfSingleSpec(point, "X06", 8,
		"pool saturation: eight workers against MaxConns=2"), func(ctx context.Context, worker int) (bool, error) {
		var one int
		if err := h.pool.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
			return false, err
		}
		return true, nil
	})
	pgPerfReadAssertPoint(t, point, summary)
	if max := h.pool.Stat().MaxConns(); max != 2 {
		t.Fatalf("X06 MaxConns=%d, want 2", max)
	}
	if err := h.pollUntil(h.ctx, 10*time.Second, func(context.Context) (bool, error) {
		return h.pool.Stat().AcquiredConns() == 0, nil
	}); err != nil {
		t.Fatalf("X06 acquired connections did not drain: %v", err)
	}
}

func pgPerfSceneX07(t *testing.T) {
	h := newPgPerfHarness(t, pgPerfTierMedium, "x07", func(options *pgPerfOptions) { options.seedMode = "minimal" })
	point := "X07"
	summary := h.pgPerfRunPoint(pgPerfSingleSpec(point, "X07", 1,
		"statement cancellation at 150ms during pg_sleep(10)"), func(ctx context.Context, worker int) (bool, error) {
		cancelCtx, cancel := context.WithTimeout(ctx, pgPerfCancelAfter)
		defer cancel()
		var value int
		err := h.pool.QueryRow(cancelCtx, "SELECT 1 FROM pg_sleep(10)").Scan(&value)
		if err == nil {
			return false, errors.New("X07 pg_sleep completed without cancellation")
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			return false, fmt.Errorf("X07 cancellation error = %T", err)
		}
		return true, nil
	})
	pgPerfReadAssertPoint(t, point, summary)
	if err := h.pollUntil(h.ctx, 10*time.Second, func(context.Context) (bool, error) {
		return h.pool.Stat().AcquiredConns() == 0, nil
	}); err != nil {
		t.Fatalf("X07 acquired connections did not drain: %v", err)
	}
}

func pgPerfSceneX08(t *testing.T) {
	h := newPgPerfHarness(t, pgPerfTierMedium, "x08", func(options *pgPerfOptions) {
		options.seedMode = "minimal"
		options.lockTimeout = pgPerfNegativeLockTimeout
	})
	h.pgPerfRunPoint(pgPerfSingleSpec("X08", "X08", 1, "observed three-connection A<-B<-C lock chain; release and join each contender"),
		func(ctx context.Context, _ int) (bool, error) { err := pgPerfLockChain(ctx, h); return err == nil, err })
}

type pgPerfLockWorker struct {
	pid      uint32
	acquired <-chan struct{}
	done     <-chan error
}

func pgPerfStartLockWorker(ctx context.Context, h *pgPerfHarness, id string, release <-chan struct{}) (pgPerfLockWorker, error) {
	pids := make(chan uint32, 1)
	acquired := make(chan struct{})
	done := make(chan error, 1)
	h.group.Add(1)
	go func() {
		var result error
		defer func() { done <- result }()
		defer h.group.Done()
		tx, err := h.beginOwned(ctx)
		if err != nil {
			result = err
			return
		}
		defer func() {
			closeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = tx.Rollback(closeCtx)
		}()
		var pid uint32
		if err := tx.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
			result = err
			return
		}
		pids <- pid
		var locked string
		if err := tx.QueryRow(ctx, "SELECT id::text FROM articles WHERE id=$1::uuid FOR UPDATE", id).Scan(&locked); err != nil {
			result = err
			return
		}
		close(acquired)
		if release != nil {
			select {
			case <-release:
			case <-ctx.Done():
				result = ctx.Err()
				return
			}
		}
		result = tx.Rollback(ctx)
	}()
	select {
	case pid := <-pids:
		return pgPerfLockWorker{pid, acquired, done}, nil
	case err := <-done:
		return pgPerfLockWorker{}, err
	case <-ctx.Done():
		return pgPerfLockWorker{}, ctx.Err()
	}
}

func pgPerfLockChain(parent context.Context, h *pgPerfHarness) (failure error) {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	member, err := h.memberPrincipal(ctx)
	if err != nil {
		return err
	}
	created, err := h.commandService().Create(ctx, member, article.CreateArticleInput{Title: "pgperf X08", Body: "initial"})
	if err != nil {
		return err
	}
	holder, err := h.beginOwned(ctx)
	if err != nil {
		return err
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = holder.Rollback(closeCtx)
	}()
	var a uint32
	if err := holder.QueryRow(ctx, "SELECT pg_backend_pid() FROM articles WHERE id=$1::uuid FOR UPDATE", created.ID).Scan(&a); err != nil {
		return err
	}
	releaseB := make(chan struct{})
	var releaseOnce sync.Once
	var pending []pgPerfLockWorker
	joined := map[<-chan error]bool{}
	defer func() {
		cancel()
		releaseOnce.Do(func() { close(releaseB) })
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer closeCancel()
		if err := holder.Rollback(closeCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			failure = errors.Join(failure, err)
		}
		for _, worker := range pending {
			if joined[worker.done] {
				continue
			}
			select {
			case <-worker.done:
			case <-closeCtx.Done():
				failure = errors.Join(failure, errors.New("X08 owned worker failed to join"))
				return
			}
		}
	}()
	b, err := pgPerfStartLockWorker(ctx, h, created.ID, releaseB)
	if err != nil {
		return err
	}
	pending = append(pending, b)
	if err := h.waitBlocking(ctx, a, 1); err != nil {
		return err
	}
	c, err := pgPerfStartLockWorker(ctx, h, created.ID, nil)
	if err != nil {
		return err
	}
	pending = append(pending, c)
	if err := h.pollUntil(ctx, 4*time.Second, func(ctx context.Context) (bool, error) {
		var chain bool
		err := h.probe.QueryRow(ctx, "SELECT $1::int=ANY(pg_blocking_pids($2::int)) AND $2::int=ANY(pg_blocking_pids($3::int))", a, b.pid, c.pid).Scan(&chain)
		return chain, err
	}); err != nil {
		return err
	}
	var graph any
	if err := h.probe.QueryRow(ctx, `SELECT json_agg(json_build_object('pid',pid,'blockers',pg_blocking_pids(pid),
		'wait_event_type',wait_event_type,'wait_event',wait_event)) FROM pg_stat_activity WHERE pid=ANY($1::int[])`, []int32{int32(a), int32(b.pid), int32(c.pid)}).Scan(&graph); err != nil {
		return err
	}
	h.evidence.writeJSONExclusive(filepath.Join("locks", fmt.Sprintf("%s.%d.chain.json", h.schema, time.Now().UnixNano())), map[string]any{"a": a, "b": b.pid, "c": c.pid, "graph": graph, "chain_observed": true})
	if err := holder.Rollback(ctx); err != nil {
		return err
	}
	select {
	case <-b.acquired:
	case err := <-b.done:
		joined[b.done] = true
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
	releaseOnce.Do(func() { close(releaseB) })
	for _, worker := range []pgPerfLockWorker{b, c} {
		select {
		case err := <-worker.done:
			joined[worker.done] = true
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func pgPerfSceneD03(t *testing.T) {
	h := newPgPerfHarness(t, pgPerfTierMedium, "d03", func(options *pgPerfOptions) { options.seedMode = "minimal" })
	member := pgPerfMemberPrincipal(h)
	created, err := h.commandService().Create(h.ctx, member, article.CreateArticleInput{Title: "pgperf d03", Body: "initial"})
	if err != nil {
		t.Fatalf("D03 create: %T", err)
	}
	before := pgPerfArticleSnapshot(t, h, created.ID)
	service, err := article.NewCommandService(article.CommandDependencies{
		Transactions: pgPerfTransactor{pool: h.pool, owner: h},
		Repository:   article.Repository{},
		IDs:          h.ids,
		Clock:        pgPerfClock{},
		Moderation: &article.ModerationDependencies{
			Auditor:      s22AuditFault{real: audit.Store{}},
			AuthorStates: s22AuthorStates,
		},
	})
	if err != nil {
		t.Fatalf("D03 command service: %T", err)
	}
	traceCtx, err := pgPerfTraceCtx(h.ctx)
	if err != nil {
		t.Fatalf("D03 trace context: %T", err)
	}
	category := article.ViolationViolence
	if _, _, err := service.Moderate(traceCtx, pgPerfAdminPrincipal(h), created.ID, 1, &category); err == nil {
		t.Fatal("D03 post-audit fault was swallowed")
	}
	after := pgPerfArticleSnapshot(t, h, created.ID)
	if after != before {
		t.Fatalf("D03 snapshot changed after audit fault: before=%s after=%s", before, after)
	}
}
