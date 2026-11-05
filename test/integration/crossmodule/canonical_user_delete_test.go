//go:build integration

package crossmodule_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"han/internal/article"
	"han/internal/audit"
	"han/internal/identity"
	"han/internal/migration"
	"han/internal/platform"
	"han/internal/recovery"
	"han/internal/session"
	"han/internal/testenv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// This is the S9.2 SQL slice, not HTTP/Register/SMTP, a production recovery
// Gate, or a durable deletion-journal test. Every selected leaf requires PG;
// no init/TestMain reads configuration, and every created schema is retained.
const cdTrace = "01ARZ3NDEKTSV4RRFFQ69G5FAV"

func TestCanonicalUserDeletePostgres(t *testing.T) {
	t.Run("Daily", func(t *testing.T) {
		for _, tc := range []cdCase{
			{name: "Zero", credential: "pending_delivery"},
			{name: "Multiple", kinds: []string{"owned", "owned"}, credential: "active", newly: 2, pending: 2},
			{name: "Violating", kinds: []string{"owned", "violating"}, credential: "expired", newly: 2, pending: 2},
			{name: "MixedPending", kinds: []string{"owned", "held", "pending_owned", "pending_held"}, credential: "active", newly: 1, already: 1, pending: 4},
		} {
			t.Run(tc.name, func(t *testing.T) { cdDaily(t, tc) })
		}
	})
	t.Run("Recovery", func(t *testing.T) {
		for _, tc := range []cdCase{
			{name: "PathAZero", recovery: true},
			{name: "PathAMixed", recovery: true, kinds: []string{"owned", "violating", "held", "pending_owned"}, newly: 2, already: 1, pending: 4},
			{name: "PathBPartial", recovery: true, deleted: true, kinds: []string{"held", "held_violating", "pending_held"}, already: 2, pending: 3},
		} {
			t.Run(tc.name, func(t *testing.T) { cdDaily(t, tc) })
		}
		t.Run("ReplayAfterDisposition", cdReplayDisposition)
		t.Run("ReplayAfterEmailReleased", cdReplayReleased)
	})
	t.Run("Retention", func(t *testing.T) {
		t.Run("LastPublish", func(t *testing.T) { cdRetention(t, false, false) })
		t.Run("LastTransferViolating", func(t *testing.T) { cdRetention(t, true, false) })
		t.Run("PendingStillRetains", func(t *testing.T) { cdRetention(t, false, true) })
		t.Run("IdentifierReuse", cdIdentifierReuse)
	})
}

type cdCase struct {
	name, credential        string
	kinds                   []string
	recovery, deleted       bool
	newly, already, pending int
}

type cdClock struct{ at time.Time }

func (c cdClock) Now() time.Time { return c.at }

type cdIDs struct{}

func (cdIDs) NewID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6], b[8] = b[6]&15|64, b[8]&63|128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

func cdUUID(t *testing.T) string {
	t.Helper()
	id, err := (cdIDs{}).NewID()
	cdMust(t, err)
	return id
}

func cdMust(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		t.Fatalf("SQL failure code=%s table=%s constraint=%s", pg.Code, pg.TableName, pg.ConstraintName)
	}
	// Do not print driver errors containing DSN, statement arguments or contacts.
	t.Fatalf("operation failed type=%T", err)
}

func cdRequire(t *testing.T, ok bool, label string) {
	t.Helper()
	if !ok {
		t.Fatal(label)
	}
}

func cdPtr(s string) *string { return &s }

func cdRollback(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = tx.Rollback(ctx)
}

type cdFixture struct {
	t               *testing.T
	ctx             context.Context
	cancel          context.CancelFunc
	pool            *pgxpool.Pool
	observer        *pgxpool.Conn
	schema          string
	dir             string
	evidenceMu      sync.Mutex
	observations    int
	clock           cdClock
	actor           session.Principal
	source, other   string
	email, phone    string
	rows            []article.Article
	request         identity.CanonicalDeleteRequest
	store           *identity.RecoveryRunStore
	gate            *cdGate
	transactions    *cdTransactions
	auditor         *cdAuditor
	lifecycle       *cdLifecycle
	articleCalls    int
	proofCalls      int
	created         bool
	mu              sync.Mutex
	before          cdImage
	fault           string
	faultHit        int
	articleWrites   int
	recipientWrites int
	hook            func(context.Context, pgx.Tx, string) error
}

func cdConfig(t *testing.T, schema string) *pgxpool.Config {
	t.Helper()
	c, err := pgxpool.ParseConfig(testenv.RequirePostgresDSN(t))
	if err != nil {
		t.Fatal("invalid PostgreSQL configuration")
	}
	cdEndpoint(t, c.ConnConfig.Host, c.ConnConfig.Port)
	for _, fallback := range c.ConnConfig.Fallbacks {
		cdEndpoint(t, fallback.Host, fallback.Port)
	}
	cdRequire(t, c.ConnConfig.User != "" && c.ConnConfig.Database != "", "explicit SQL identity/database required")
	c.MaxConns = 6
	c.ConnConfig.RuntimeParams["search_path"] = pgx.Identifier{schema}.Sanitize()
	return c
}

func cdEndpoint(t *testing.T, host string, port uint16) {
	t.Helper()
	ip := net.ParseIP(host)
	cdRequire(t, ip != nil && ip.IsLoopback() && port == 5433, "requires literal loopback:5433 for primary and all fallbacks")
}

func cdNew(t *testing.T, tc cdCase) *cdFixture {
	t.Helper()
	f := &cdFixture{t: t, schema: "han_s92_" + strings.ReplaceAll(cdUUID(t), "-", "")}
	f.ctx, f.cancel = context.WithTimeout(context.Background(), 90*time.Second)
	t.Cleanup(f.close)
	cdEvidenceDirectory(f)
	config := cdConfig(t, f.schema)
	var err error
	f.pool, err = pgxpool.NewWithConfig(f.ctx, config)
	cdMust(t, err)
	cdPreflight(f, config)
	_, err = f.pool.Exec(f.ctx, "CREATE SCHEMA "+pgx.Identifier{f.schema}.Sanitize())
	cdMust(t, err)
	f.created = true
	t.Logf("retained_schema=%s stage=create_committed", f.schema)
	cdMigrate(f)
	f.observer, err = f.pool.Acquire(f.ctx)
	cdMust(t, err)
	f.ctx, err = platform.ContextWithTraceID(f.ctx, cdTrace)
	cdMust(t, err)
	f.transactions = &cdTransactions{f: f}
	f.auditor = &cdAuditor{f: f}
	f.lifecycle, err = cdNewLifecycle(f)
	cdMust(t, err)
	cdSeed(f, tc)
	return f
}

func cdPreflight(f *cdFixture, config *pgxpool.Config) {
	var host, user, database string
	var port, version int
	var now time.Time
	err := f.pool.QueryRow(f.ctx, `SELECT host(inet_server_addr()), inet_server_port(),
		current_setting('server_version_num')::int, current_user, current_database(), clock_timestamp()`).
		Scan(&host, &port, &version, &user, &database, &now)
	cdMust(f.t, err)
	cdEndpoint(f.t, host, uint16(port))
	cdRequire(f.t, version == 180006 && user == config.ConnConfig.User && database == config.ConnConfig.Database,
		"PostgreSQL endpoint/version/identity/database mismatch; no schema created")
	f.clock = cdClock{now.UTC().Truncate(time.Microsecond)}
	f.t.Log("preflight=passed literal_loopback5433=true version180006=true identity_match=true database_match=true")
}

func cdMigrate(f *cdFixture) {
	items, err := cdFrozenMigrations(os.DirFS(filepath.Join("..", "..", "..")))
	cdMust(f.t, err)
	cdRequire(f.t, len(items) == 11, "expected frozen migrations 0001 through 0011")
	tx, err := f.pool.Begin(f.ctx)
	cdMust(f.t, err)
	defer cdRollback(tx)
	for i, item := range items {
		cdRequire(f.t, item.Version == i+1, "migration order mismatch")
		_, err = tx.Exec(f.ctx, item.SQL)
		cdMust(f.t, err)
		f.t.Logf("schema=%s migration=%s stage=applied_in_tx", f.schema, item.Name)
	}
	cdMust(f.t, tx.Commit(f.ctx))
	f.t.Logf("schema=%s stage=migrations_committed constraints=enabled", f.schema)
	// 0010 requires a bound installation before any recovery run is inserted.
	_, err = f.pool.Exec(f.ctx, `INSERT INTO recovery_installation(singleton,metadata_version,deployment_id,
		installation_id,target_digest,commitment_ref,commitment_digest,recovery_journal_id,article_journal_id,
		recovery_epoch,article_epoch,recovery_initial_sequence,article_initial_sequence,recovery_initial_hash,
		article_initial_hash,revision,installed_at)
		VALUES(true,'han-recovery-install/v1','s92-synthetic-deployment','s92-synthetic-installation',
		'sha256:0000000000000000000000000000000000000000000000000000000000000000',
		'recovery/v1/installation/s92-synthetic-deployment',
		'sha256:1111111111111111111111111111111111111111111111111111111111111111',
		's92-recovery-journal','s92-article-journal',1,1,0,0,'','',1,'2026-01-01T00:00:00Z')`)
	cdMust(f.t, err)
	f.t.Logf("schema=%s stage=recovery_installation_bound", f.schema)
}

func (f *cdFixture) close() {
	if f.transactions != nil {
		for _, tx := range f.transactions.seen {
			if !tx.ended {
				cdRollback(tx)
				f.t.Error("business transaction reached cleanup without explicit end")
			}
		}
	}
	if f.lifecycle != nil && f.lifecycle.verifier.count() != 0 {
		f.t.Error("native proof registration outlived its transaction")
	}
	if f.observer != nil {
		cdFinalEvidence(f)
		f.observer.Release()
	}
	if f.cancel != nil {
		f.cancel()
	}
	if f.pool != nil {
		if f.pool.Stat().AcquiredConns() != 0 {
			f.t.Error("SQL connection leak")
		}
		f.pool.Close()
	}
	f.t.Logf("schema=%s retained=%t resources=rollback_observer_release_cancel_pool_close workers=none", f.schema, f.created)
}

func cdSeed(f *cdFixture, tc cdCase) {
	tx, err := f.pool.Begin(f.ctx)
	cdMust(f.t, err)
	defer cdRollback(tx)
	f.actor = session.Principal{UserID: cdUser(f, tx, "admin", tc.recovery), Role: "admin", Status: "active", IsInitialAdmin: tc.recovery}
	f.source, f.other = cdUser(f, tx, "member", false), cdUser(f, tx, "member", false)
	f.email, f.phone = "cd_"+f.source+"@example.invalid", "synthetic-phone-"+f.source
	cdCredential(f, tx, tc.credential)
	for i := 0; i < 2; i++ {
		cdSession(f, tx, f.source)
	}
	cdSession(f, tx, f.actor.UserID)
	cdSession(f, tx, f.other)
	for _, kind := range tc.kinds {
		f.rows = append(f.rows, cdArticle(f, tx, kind, f.source))
	}
	cdArticle(f, tx, "owned", f.other)
	if tc.deleted {
		cdSeedDeleted(f, tx)
	}
	cdMust(f.t, tx.Commit(f.ctx))
	f.request = identity.CanonicalDeleteRequest{Actor: f.actor, TargetID: f.source, TargetRole: identity.RoleMember,
		Mode: identity.CanonicalDeleteDailyGovernance, Reason: identity.AccountDeletionGovernance, TraceID: cdTrace}
	if tc.recovery {
		cdRecovery(f)
	}
	f.clock.at = f.clock.at.Add(time.Second)
}

func cdUser(f *cdFixture, tx pgx.Tx, role string, initial bool) string {
	id := cdUUID(f.t)
	email, phone, username := "cd_"+id+"@example.invalid", "synthetic-phone-"+id, "cd_"+strings.ReplaceAll(id, "-", "")
	_, err := tx.Exec(f.ctx, `INSERT INTO users (id,username,email,phone,permanent_password_digest,
		permanent_password_generation,role,is_initial_admin,status,credential_mode,onboarding_completed_at,created_at,updated_at)
		VALUES ($1,$2,$3,$4,'synthetic-not-an-authenticating-digest',0,$5,$6,'active','permanent',$7,$7,$7)`,
		id, username, email, phone, role, initial, f.clock.at)
	cdMust(f.t, err)
	for kind, value := range map[string]string{"username": username, "email": email, "phone": phone} {
		_, err = tx.Exec(f.ctx, `INSERT INTO identity_identifiers (user_id,kind,normalized_value,lifecycle,created_at)
			VALUES ($1,$2,$3,'active',$4)`, id, kind, value, f.clock.at)
		cdMust(f.t, err)
	}
	return id
}

func cdCredential(f *cdFixture, tx pgx.Tx, current string) {
	if current == "" {
		current = "active"
	}
	for i, state := range []string{current, "consumed", "superseded"} {
		issued, expires := f.clock.at, f.clock.at.Add(time.Hour)
		if state == "expired" {
			issued, expires = f.clock.at.Add(-2*time.Hour), f.clock.at.Add(-time.Hour)
		}
		_, err := tx.Exec(f.ctx, `INSERT INTO temporary_credentials
			(id,user_id,purpose,generation,state,secret_digest,delivered_to_email,issued_at,expires_at,consumed_at,superseded_at)
			VALUES ($1,$2,'new_administrator',$3,$4,'synthetic-nonsecret-digest',$5,$6,$7,
			CASE WHEN $4='consumed' THEN $6::timestamptz ELSE NULL END,
			CASE WHEN $4='superseded' THEN $6::timestamptz ELSE NULL END)`,
			cdUUID(f.t), f.source, i+1, state, f.email, issued, expires)
		cdMust(f.t, err)
	}
	// Purpose/gen=0/source=NULL are legal legacy fixtures after 0008. This does
	// not claim password-reset durable lineage or a real delivered credential.
}

func cdSession(f *cdFixture, tx pgx.Tx, userID string) {
	id := cdUUID(f.t)
	_, err := tx.Exec(f.ctx, `INSERT INTO sessions (id,user_id,token_digest,csrf_token_digest,
		authenticated_credential_kind,issued_at,last_seen_at,absolute_expires_at,idle_expires_at)
		VALUES ($1,$2,$3,$4,'permanent',$5,$5,$6,$6)`, id, userID,
		[]byte("synthetic-token-"+id), []byte("synthetic-csrf-"+id), f.clock.at, f.clock.at.Add(time.Hour))
	cdMust(f.t, err)
}

func cdArticle(f *cdFixture, tx pgx.Tx, kind, userID string) article.Article {
	a := article.Article{ID: cdUUID(f.t), AuthorID: cdPtr(userID), Title: "合成标题", Body: "synthetic <b>literal</b> body",
		PublicationState: article.PublicationPublished, ViolationState: article.ViolationClear,
		DeletionState: article.DeletionActive, Version: 7, CreatedAt: f.clock.at, UpdatedAt: f.clock.at}
	if strings.Contains(kind, "held") {
		a.AuthorID, a.OrphanedFromUserID, a.PublicationState = nil, cdPtr(userID), article.PublicationHeld
	}
	if strings.Contains(kind, "violating") {
		category := article.ViolationCategory("violence")
		a.ViolationState, a.ViolationCategory = article.ViolationViolating, &category
	}
	if strings.HasPrefix(kind, "pending_") {
		cdPending(f, tx, &a)
	}
	_, err := tx.Exec(f.ctx, `INSERT INTO articles (id,author_id,orphaned_from_user_id,title,body,
		publication_state,violation_state,violation_category,deletion_state,deletion_operation_id,version,created_at,updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, a.ID, a.AuthorID, a.OrphanedFromUserID,
		a.Title, a.Body, a.PublicationState, a.ViolationState, a.ViolationCategory, a.DeletionState,
		a.DeletionOperationID, a.Version, a.CreatedAt, a.UpdatedAt)
	cdMust(f.t, err)
	return a
}

func cdPending(f *cdFixture, tx pgx.Tx, a *article.Article) {
	op := cdUUID(f.t)
	kind := "ordinary"
	if a.OrphanedFromUserID != nil {
		kind = "orphan"
	}
	_, err := tx.Exec(f.ctx, `INSERT INTO deletion_operations
		(id,article_id,article_version,delete_kind,actor_user_id,actor_role,orphaned_from_user_id,status,started_at)
		VALUES ($1,$2,$3,$4,$5,'admin',$6,'pending_journal',$7)`,
		op, a.ID, a.Version-1, kind, f.actor.UserID, a.OrphanedFromUserID, f.clock.at)
	cdMust(f.t, err)
	a.DeletionState, a.DeletionOperationID = article.DeletionPending, &op
	// Legal existing pending DB state, not a forged tombstone or finalization.
}

func cdSeedDeleted(f *cdFixture, tx pgx.Tx) {
	_, err := tx.Exec(f.ctx, `UPDATE users SET status='deleted',retained_email=email,email=NULL,phone=NULL,
		permanent_password_digest=NULL,permanent_password_generation=NULL,deleted_at=$2 WHERE id=$1`, f.source, f.clock.at)
	cdMust(f.t, err)
	_, err = tx.Exec(f.ctx, `UPDATE identity_identifiers SET lifecycle='reserved' WHERE user_id=$1`, f.source)
	cdMust(f.t, err)
}

func cdRecovery(f *cdFixture) {
	var err error
	f.store, err = identity.NewRecoveryRunStore(f.pool)
	cdMust(f.t, err)
	runID, backup := "synthetic-run-"+cdUUID(f.t), "synthetic-immutable-backup"
	cdMust(f.t, f.store.Create(f.ctx, recovery.RecoveryRun{RunID: runID, BackupRef: backup,
		JournalKeyID: "synthetic-key", JournalKeyVersion: "fixture-v1", JournalSecretFingerprint: "synthetic-nonsecret-fingerprint",
		JournalStage: recovery.StageBegun, StartedAt: f.clock.at, Outcome: recovery.RecoveryOutcomeRunning}))
	_, err = f.pool.Exec(f.ctx, `INSERT INTO recovery_account_reviews
		(recovery_run_id,user_id,backup_role,backup_status,backup_deleted_at,decision)
		SELECT $1,id,role,status,deleted_at,'pending' FROM users WHERE id=$2`, runID, f.source)
	cdMust(f.t, err)
	capability, err := identity.NewRecoveryDeleteCapability(runID, backup)
	cdMust(f.t, err)
	f.request.Mode = identity.CanonicalDeleteRecoveryKept
	f.request.Recovery = &identity.RecoveryKeptDeletedRequest{Capability: capability, EvidenceRef: cdPtr("synthetic-review-evidence")}
	f.gate = &cdGate{f: f, maintenance: true, readiness: false}
	f.t.Log("recovery_fixture=Store.Create+native_capability controlled_gate=true production_gate_or_journal_evidence=false")
}

type cdGate struct {
	f                      *cdFixture
	maintenance, readiness bool
	err                    error
	calls                  int
}

func (g *cdGate) CheckRecoveryOpenGate(ctx context.Context, tx pgx.Tx) error {
	g.calls++
	if _, ok := tx.(*cdTx); !ok || ctx != g.f.ctx {
		return errors.New("controlled Gate lost caller transaction/context")
	}
	if g.err != nil {
		return g.err
	}
	if !g.maintenance || g.readiness {
		return errors.New("controlled maintenance/readiness input denies deletion")
	}
	return nil
}

func cdCommand(f *cdFixture) *article.CommandService {
	s, err := article.NewCommandService(article.CommandDependencies{Transactions: f.transactions,
		Repository: article.Repository{}, IDs: cdIDs{}, Clock: f.clock,
		Orphaning: &article.OrphanArticlesCanonicalDependencies{Repository: article.CanonicalOrphanRepository{},
			Auditor: f.auditor, AuditIDs: cdIDs{}, Clock: f.clock},
		Disposition: &article.OrphanDispositionDependencies{Identity: f.lifecycle, Auditor: f.auditor, RecipientMaxAttempts: 2}})
	cdMust(f.t, err)
	return s
}

func cdService(f *cdFixture) *identity.CanonicalDeleteCoordinator {
	command := cdCommand(f)
	port := identity.ArticleAccountPortFunc(func(ctx context.Context, tx pgx.Tx, userID string) (identity.CanonicalDeleteArticleResult, error) {
		f.mu.Lock()
		f.articleCalls++
		f.mu.Unlock()
		if _, ok := tx.(*cdTx); !ok || ctx != f.ctx {
			return identity.CanonicalDeleteArticleResult{}, errors.New("article bridge lost caller tx/context")
		}
		tx.(*cdTx).canonical = true
		r, err := command.OrphanArticlesCanonical(ctx, tx, userID)
		return identity.CanonicalDeleteArticleResult{NewlyOrphanedCount: r.NewlyOrphanedCount,
			AlreadyOrphanedCount: r.AlreadyOrphanedCount, PendingOrphanCount: r.PendingOrphanCount}, err
	})
	dependencies := identity.CanonicalDeleteDependencies{Transactions: f.transactions,
		Repository: identity.PostgresCanonicalDeleteRepository{}, Articles: port, Audit: f.auditor, IDs: cdIDs{}, Clock: f.clock}
	if f.store != nil {
		dependencies.RecoveryRuns = f.store
	}
	if f.gate != nil {
		dependencies.RecoveryGate = f.gate
	}
	s, err := identity.NewCanonicalDeleteService(dependencies)
	cdMust(f.t, err)
	return s
}

// All 11 tables have full stable JSON plus xmin/ctid pre/post images in memory.
// Retained stage files hash credential columns; logs never contain row contents.
var cdTables = []string{"users", "identity_identifiers", "sessions", "temporary_credentials", "articles",
	"deletion_operations", "transfer_notices", "transfer_notice_recipients", "audit_events", "recovery_account_reviews", "recovery_runs"}

type cdImage map[string]string

type cdQuery interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func cdSnapshot(ctx context.Context, q cdQuery, schema string) (cdImage, error) {
	image := cdImage{}
	for _, table := range cdTables {
		var value string
		sql := `SELECT COALESCE(jsonb_agg(v ORDER BY v::text),'[]'::jsonb)::text FROM
			(SELECT jsonb_build_object('row',to_jsonb(r),'xmin',r.xmin::text,'ctid',r.ctid::text) v FROM ` +
			pgx.Identifier{schema, table}.Sanitize() + ` r) images`
		if err := q.QueryRow(ctx, sql).Scan(&value); err != nil {
			return nil, err
		}
		image[table] = value
	}
	return image, nil
}

func cdObserve(f *cdFixture, stage string) cdImage {
	image, err := cdSnapshot(f.ctx, f.observer, f.schema)
	cdMust(f.t, err)
	cdMust(f.t, cdSaveImage(f, stage, image))
	for _, table := range cdTables {
		f.t.Logf("schema=%s stage=%s table=%s image_sha256=%x includes=xmin,ctid,all_columns",
			f.schema, stage, table, sha256.Sum256([]byte(image[table])))
	}
	return image
}

func cdEvidenceDirectory(f *cdFixture) {
	var err error
	f.dir, err = os.MkdirTemp("", f.schema+"_") // Intentionally retained; no t.TempDir/RemoveAll.
	cdMust(f.t, err)
	f.t.Logf("schema=%s leaf=%s retained_dir=%s", f.schema, f.t.Name(), f.dir)
	data, err := json.MarshalIndent(map[string]any{"leaf": f.t.Name(), "schema": f.schema,
		"retained_dir": f.dir, "tables": cdTables,
		"redaction":          "credential values: SHA256(JSON value), null unchanged; synthetic relationships/state retained",
		"raw_image_hash":     "SHA256(exact PostgreSQL JSON text)",
		"redacted_rows_hash": "SHA256(compact encoding/json rows with sorted object keys); JSON numbers preserved"}, "", "  ")
	cdMust(f.t, err)
	cdMust(f.t, cdWriteEvidence(filepath.Join(f.dir, "manifest.json"), data))
}

type cdTableEvidence struct {
	RawSHA256      string          `json:"raw_image_sha256"`
	RedactedSHA256 string          `json:"redacted_rows_sha256"`
	Rows           json.RawMessage `json:"rows"`
}

func cdSaveImage(f *cdFixture, stage string, image cdImage) error {
	f.evidenceMu.Lock()
	defer f.evidenceMu.Unlock()
	tables := make(map[string]cdTableEvidence, len(cdTables))
	for _, table := range cdTables {
		rows, err := cdRedactedRows(image[table])
		if err != nil {
			return fmt.Errorf("redact evidence table %s: %w", table, err)
		}
		tables[table] = cdTableEvidence{RawSHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(image[table]))),
			RedactedSHA256: fmt.Sprintf("%x", sha256.Sum256(rows)), Rows: rows}
	}
	f.observations++
	data, err := json.MarshalIndent(map[string]any{"leaf": f.t.Name(), "schema": f.schema,
		"stage": stage, "sequence": f.observations, "tables": tables}, "", "  ")
	if err != nil {
		return err
	}
	name := fmt.Sprintf("%03d-%s.json", f.observations, stage)
	if err := cdWriteEvidence(filepath.Join(f.dir, name), data); err != nil {
		return err
	}
	f.t.Logf("schema=%s stage=%s retained_image=%s image_file_sha256=%x tables=11 credentials=hashed",
		f.schema, stage, filepath.Join(f.dir, name), sha256.Sum256(data))
	return nil
}

func cdRedactedRows(raw string) (json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.UseNumber()
	var rows []any
	if err := decoder.Decode(&rows); err != nil {
		return nil, err
	}
	redacted, err := cdRedact(rows)
	if err != nil {
		return nil, err
	}
	return json.Marshal(redacted)
}

func cdRedact(value any) (any, error) {
	switch current := value.(type) {
	case map[string]any:
		for key, field := range current {
			if cdCredentialColumn(key) && field != nil {
				encoded, err := json.Marshal(field)
				if err != nil {
					return nil, err
				}
				current[key] = map[string]any{"present": true, "sha256": fmt.Sprintf("%x", sha256.Sum256(encoded))}
			} else {
				redacted, err := cdRedact(field)
				if err != nil {
					return nil, err
				}
				current[key] = redacted
			}
		}
	case []any:
		for i, item := range current {
			redacted, err := cdRedact(item)
			if err != nil {
				return nil, err
			}
			current[i] = redacted
		}
	}
	return value, nil
}

func cdCredentialColumn(name string) bool {
	name = strings.ToLower(name)
	return strings.Contains(name, "digest") || strings.Contains(name, "secret") ||
		strings.Contains(name, "token") || strings.Contains(name, "fingerprint") ||
		name == "password" || name == "dsn" || name == "connection_string"
}

func cdWriteEvidence(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(data)
	return errors.Join(writeErr, file.Sync(), file.Close())
}

func cdFinalEvidence(f *cdFixture) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	image, err := cdSnapshot(ctx, f.observer, f.schema)
	if err == nil {
		err = cdSaveImage(f, "resource_close_final", image)
	}
	if err != nil {
		f.t.Errorf("final retained SQL evidence unavailable type=%T", err)
	}
}

func cdSame(t *testing.T, before, after cdImage) {
	t.Helper()
	for _, table := range cdTables {
		cdRequire(t, before[table] == after[table], "independent SQL image changed: "+table)
	}
}

func cdBool(f *cdFixture, sql string, args ...any) {
	f.t.Helper()
	var ok bool
	cdMust(f.t, f.observer.QueryRow(f.ctx, sql, args...).Scan(&ok))
	cdRequire(f.t, ok, "independent SQL expectation failed")
}

type cdTransactions struct {
	f    *cdFixture
	seen []*cdTx
	mu   sync.Mutex
}

func (s *cdTransactions) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := s.f.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	pid, observer := tx.Conn().PgConn().PID(), s.f.observer.Conn().PgConn().PID()
	if pid == observer {
		cdRollback(tx)
		return nil, errors.New("observer is not independent")
	}
	w := &cdTx{Tx: tx, f: s.f}
	s.mu.Lock()
	s.seen = append(s.seen, w)
	s.mu.Unlock()
	s.f.t.Logf("schema=%s begin business_pid=%d observer_pid=%d independent=true", s.f.schema, pid, observer)
	return w, nil
}

type cdTx struct {
	pgx.Tx
	f                         *cdFixture
	ended, committed          bool
	writes, audits, rollbacks int
	runLocks                  int
	canonical                 bool
	rollbackErr               error
}

func (tx *cdTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if strings.Contains(sql, "FROM recovery_runs") && strings.Contains(sql, "FOR UPDATE") {
		tx.runLocks++
		tx.f.t.Logf("schema=%s recovery_authority=real_Store_lock_read caller_tx=true", tx.f.schema)
	}
	return tx.Tx.QueryRow(ctx, sql, args...)
}

func (tx *cdTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	tag, err := tx.Tx.Exec(ctx, sql, args...)
	if err != nil {
		return tag, err
	}
	trim := strings.ToUpper(strings.TrimSpace(sql))
	if strings.HasPrefix(trim, "INSERT") || strings.HasPrefix(trim, "UPDATE") || strings.HasPrefix(trim, "DELETE") {
		tx.writes++
	}
	if strings.Contains(sql, "INSERT INTO audit_events") {
		tx.audits++
	}
	if err = tx.f.afterSQL(ctx, tx, sql); err != nil {
		return tag, err
	}
	return tag, nil
}

func (tx *cdTx) Commit(ctx context.Context) error {
	if tx.f.before != nil {
		outside, err := cdSnapshot(ctx, tx.f.observer, tx.f.schema)
		if err != nil || !reflect.DeepEqual(tx.f.before, outside) {
			return errors.New("precommit observer saw business writes or could not read")
		}
		inside, err := cdSnapshot(ctx, tx, tx.f.schema)
		if err != nil || reflect.DeepEqual(inside, outside) {
			return errors.New("precommit caller tx lacks SQL effects")
		}
		if err := cdSaveImage(tx.f, "precommit_observer", outside); err != nil {
			return err
		}
		if err := cdSaveImage(tx.f, "precommit_caller", inside); err != nil {
			return err
		}
		tx.f.t.Logf("schema=%s stage=precommit caller_sees_new=true observer_sees_old=true", tx.f.schema)
	}
	err := tx.Tx.Commit(ctx)
	tx.ended, tx.committed = true, err == nil
	tx.f.lifecycle.verifier.forget(tx)
	return err
}

func (tx *cdTx) Rollback(context.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := tx.Tx.Rollback(ctx)
	if !tx.ended {
		tx.rollbacks++
		tx.rollbackErr = err
	}
	tx.ended = true
	tx.f.lifecycle.verifier.forget(tx)
	return err
}

// A fault is returned only AFTER the real SQL succeeds. This tests caller
// rollback ownership; it is not an engine/IO failure simulation. Audit failures
// instead use real audit.Store INSERT primary-key violations (23505).
func (f *cdFixture) afterSQL(ctx context.Context, tx pgx.Tx, sql string) error {
	f.mu.Lock()
	stage := cdSQLStage(sql)
	if stage == "Article" {
		f.articleWrites++
		if f.articleWrites != 2 {
			stage = "ArticleFirst"
		}
	}
	if stage == "RecipientInsert" {
		f.recipientWrites++
		if f.recipientWrites != 2 {
			stage = "RecipientFirst"
		}
	}
	hit := stage != "" && stage == f.fault
	if hit {
		f.faultHit++
	}
	hook := f.hook
	f.mu.Unlock()
	if hit {
		f.t.Logf("schema=%s fault=%s boundary=after_real_SQL_success", f.schema, stage)
		return cdInjected
	}
	if hook != nil {
		return hook(ctx, tx, stage)
	}
	return nil
}

func cdSQLStage(sql string) string {
	for _, pair := range [][2]string{
		{"status = 'deleted',", "User"}, {"SET lifecycle = 'reserved'", "Identifier"},
		{"DELETE FROM sessions", "Session"}, {"SET state = 'recovery_invalidated'", "Credential"},
		{"author_id = NULL", "Article"}, {"SET retained_email = NULL", "RetainedClear"},
		{"DELETE FROM identity_identifiers", "EmailDelete"}, {"decision = 'kept_deleted'", "ReviewComplete"},
		{"INSERT INTO transfer_notice_recipients", "RecipientInsert"},
		{"SET author_id = $4::uuid", "Disposition"},
	} {
		if strings.Contains(sql, pair[0]) {
			return pair[1]
		}
	}
	return ""
}

var cdInjected = errors.New("controlled failure after real SQL")

type cdAuditor struct {
	f                   *cdFixture
	action, collision   string
	nth, matching, hits int
}

func (a *cdAuditor) Append(ctx context.Context, tx pgx.Tx, event audit.Event) error {
	if _, ok := tx.(*cdTx); !ok {
		return errors.New("audit lost caller transaction")
	}
	if event.Action == a.action {
		a.matching++
		if a.matching == a.nth {
			event.ID = a.collision // Controlled invalid INSERT input; no fake appender result.
			err := (audit.Store{}).Append(ctx, tx, event)
			var pg *pgconn.PgError
			if !errors.As(err, &pg) || pg.Code != "23505" || pg.ConstraintName != "audit_events_pkey" {
				return errors.New("expected real audit_events_pkey INSERT 23505")
			}
			a.hits++
			a.f.t.Logf("schema=%s audit_action=%s sqlstate=%s constraint=%s previous_successful_audits=%d",
				a.f.schema, event.Action, pg.Code, pg.ConstraintName, tx.(*cdTx).audits)
			return err
		}
	}
	return (audit.Store{}).Append(ctx, tx, event)
}

// The fixture fixes this native lifecycle and its trusted provider once.
// Each command can register a native opaque proof, never replace the provider.
type cdLifecycle struct {
	native   *identity.ArticleIdentityLifecycle
	verifier *cdVerifier
}

func cdNewLifecycle(f *cdFixture) (*cdLifecycle, error) {
	verifier := &cdVerifier{f: f, proofs: make(map[cdProofKey]cdBoundProof)}
	native, err := identity.NewArticleIdentityLifecycle(verifier, f.auditor, cdIDs{}, &f.clock)
	if err != nil {
		return nil, err
	}
	f.t.Logf("schema=%s lifecycle=fixed_native_instance verifier=fixed_provider", f.schema)
	return &cdLifecycle{native: native, verifier: verifier}, nil
}

func (a *cdLifecycle) LockTransferParties(ctx context.Context, tx pgx.Tx, target *string, source string) (article.DispositionParties, error) {
	p, err := a.native.LockTransferParties(ctx, tx, target, source)
	return article.DispositionParties{SourceID: p.SourceID, SourceEmail: p.SourceEmail, TargetID: p.TargetID, TargetEmail: p.TargetEmail}, err
}

func (a *cdLifecycle) LockDeletionCandidate(ctx context.Context, tx pgx.Tx, id string) (article.DispositionDeletionCandidate, error) {
	p, err := a.native.LockDeletionCandidate(ctx, tx, id)
	return article.DispositionDeletionCandidate{UserID: p.UserID, Deleted: p.Deleted}, err
}

func (a *cdLifecycle) ClearRetainedEmailIfEligible(ctx context.Context, tx pgx.Tx, source string, proof *article.DispositionCompletionProof) (bool, error) {
	reference, err := a.verifier.register(ctx, tx, source, proof)
	if err != nil {
		return false, err
	}
	return a.native.ClearRetainedEmailIfEligible(ctx, tx, source, reference)
}

type cdVerifier struct {
	f      *cdFixture
	mu     sync.Mutex
	proofs map[cdProofKey]cdBoundProof
}

type cdProofKey struct {
	tx      *cdTx
	auditID string
}

type cdBoundProof struct {
	proof     *article.DispositionCompletionProof
	reference identity.ArticleCompletionReference
	traceID   string
}

func (v *cdVerifier) register(ctx context.Context, tx pgx.Tx, source string, proof *article.DispositionCompletionProof) (identity.ArticleCompletionReference, error) {
	caller, ok := tx.(*cdTx)
	traceID, present := platform.TraceIDFromContext(ctx)
	if !ok || caller.ended || !present || proof == nil {
		return identity.ArticleCompletionReference{}, errors.New("active caller transaction, trace and native proof required")
	}
	reference := cdReference(proof.Reference())
	if reference.Transaction != tx || reference.SourceID != source || reference.AuditEventID == "" {
		return identity.ArticleCompletionReference{}, errors.New("native proof transaction/source/audit mismatch")
	}
	key := cdProofKey{tx: caller, auditID: reference.AuditEventID}
	v.mu.Lock()
	defer v.mu.Unlock()
	if registered, found := v.proofs[key]; found {
		if registered.proof != proof || registered.traceID != traceID || !reflect.DeepEqual(registered.reference, reference) {
			return identity.ArticleCompletionReference{}, errors.New("native proof registration cannot be replaced")
		}
		return reference, nil
	}
	v.proofs[key] = cdBoundProof{proof: proof, reference: cdReference(proof.Reference()), traceID: traceID}
	return reference, nil
}

func (v *cdVerifier) VerifyCompletion(ctx context.Context, tx pgx.Tx, reference identity.ArticleCompletionReference) (identity.VerifiedArticleCompletion, error) {
	caller, ok := tx.(*cdTx)
	if !ok || caller.ended {
		return identity.VerifiedArticleCompletion{}, errors.New("active caller transaction required")
	}
	v.mu.Lock()
	registered, found := v.proofs[cdProofKey{tx: caller, auditID: reference.AuditEventID}]
	v.mu.Unlock()
	traceID, present := platform.TraceIDFromContext(ctx)
	if !found || !present || traceID != registered.traceID || reference.Transaction != tx ||
		!reflect.DeepEqual(reference, registered.reference) ||
		!reflect.DeepEqual(registered.reference, cdReference(registered.proof.Reference())) {
		return identity.VerifiedArticleCompletion{}, errors.New("unregistered or changed native completion binding")
	}
	facts, err := registered.proof.VerifyCompletion(ctx, tx)
	if err != nil {
		return identity.VerifiedArticleCompletion{}, err
	}
	v.f.mu.Lock()
	v.f.proofCalls++
	v.f.mu.Unlock()
	v.f.t.Logf("schema=%s completion=native_proof_verify exact_tx=true", v.f.schema)
	return cdFacts(facts), nil
}

func (v *cdVerifier) forget(tx *cdTx) {
	v.mu.Lock()
	defer v.mu.Unlock()
	for key := range v.proofs {
		if key.tx == tx {
			delete(v.proofs, key)
		}
	}
}

func (v *cdVerifier) count() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return len(v.proofs)
}

func cdState(s article.DispositionArticleState) identity.ArticleDispositionState {
	var category *string
	if s.Category != nil {
		category = cdPtr(string(*s.Category))
	}
	return identity.ArticleDispositionState{Exists: s.Exists, ID: s.ID, Version: s.Version, AuthorID: s.AuthorID,
		SourceID: s.SourceID, Publication: string(s.Publication), Violation: string(s.Violation), Category: category,
		Deletion: string(s.Deletion), OperationID: s.OperationID}
}

func cdReference(r article.DispositionCompletionReference) identity.ArticleCompletionReference {
	return identity.ArticleCompletionReference{Transaction: r.Transaction, ExpectedArticleID: r.ExpectedArticleID,
		SourceID: r.SourceID, BeforeVersion: r.BeforeVersion, Action: identity.ArticleDispositionAction(r.Action),
		NoticeID: r.NoticeID, NewAuthorID: r.NewAuthorID, OperationID: r.OperationID, AuditEventID: r.AuditEventID,
		BeforeArticle: cdState(r.BeforeArticle)}
}

func cdFacts(f article.DispositionCompletionFacts) identity.VerifiedArticleCompletion {
	r := identity.VerifiedArticleCompletion{AfterArticle: cdState(f.AfterArticle), RemainingActiveCount: f.RemainingActiveCount,
		RemainingPendingCount: f.RemainingPendingCount, Audit: identity.ArticleDispositionAudit{ID: f.Audit.ID,
			ArticleID: f.Audit.TargetID, Action: f.Audit.Action, Outcome: f.Audit.Outcome, TraceID: f.Audit.TraceID}}
	if f.Notice != nil {
		r.Notice = &identity.ArticleDispositionNotice{ID: f.Notice.ID, ArticleID: f.Notice.ArticleRef,
			NewAuthorID: f.Notice.NewAuthorID, FormerDeletedUserID: f.Notice.FormerDeletedUserID}
	}
	for _, p := range f.Recipients {
		r.Recipients = append(r.Recipients, identity.ArticleDispositionRecipient{Kind: string(p.Kind), EmailSnapshot: p.EmailSnapshot, Status: string(p.Status)})
	}
	if p := f.CompletedOperation; p != nil {
		r.CompletedOperation = &identity.ArticleDispositionOperation{ID: p.ID, ArticleID: p.ArticleID,
			ArticleVersion: p.ArticleVersion, DeleteKind: string(p.DeleteKind), SourceID: p.OrphanedFromUserID, Status: string(p.Status)}
		if p.AuditEventID != nil {
			r.CompletedOperation.AuditEventID = *p.AuditEventID
		}
	}
	return r
}

func cdDelete(f *cdFixture) identity.CanonicalDeleteResult {
	f.before = cdObserve(f, "before_canonical")
	r, err := cdService(f).CanonicalDelete(f.ctx, f.request)
	cdMust(f.t, err)
	cdObserve(f, "after_canonical_commit")
	cdRequire(f.t, len(f.transactions.seen) > 0 && f.transactions.seen[len(f.transactions.seen)-1].committed, "canonical did not commit")
	return r
}

func cdDaily(t *testing.T, tc cdCase) {
	f := cdNew(t, tc)
	before := cdObserve(f, "seed")
	r := cdDelete(f)
	cdRequire(t, r.TargetID == f.source && r.NewlyOrphanedCount == tc.newly &&
		r.AlreadyOrphanedCount == tc.already && r.PendingOrphanCount == tc.pending &&
		r.EmailIdentifierFreed == (tc.pending == 0), "canonical result differs from seed-derived expectation")
	cdAssertDeleted(f, tc.pending == 0)
	cdAssertArticles(f)
	cdAssertAudit(f, tc.newly, tc.already, tc.pending, tc.pending == 0)
	cdUnrelated(f, before)
	if tc.deleted {
		cdBool(f, `SELECT deleted_at=$2 FROM users WHERE id=$1`, f.source, f.clock.at.Add(-time.Second))
	}
	if tc.recovery {
		cdAssertRecovery(f)
		cdReplay(f, tc.pending, tc.newly+tc.already)
	}
	cdObserve(f, "committed")
}

func cdAssertDeleted(f *cdFixture, released bool) {
	cdBool(f, `SELECT status='deleted' AND email IS NULL AND phone IS NULL AND permanent_password_digest IS NULL
		AND permanent_password_generation IS NULL AND deleted_at IS NOT NULL
		AND (($2 AND retained_email IS NULL) OR (NOT $2 AND retained_email=$3)) FROM users WHERE id=$1`, f.source, released, f.email)
	cdBool(f, `SELECT count(*)=2 AND bool_and(lifecycle='reserved' AND released_at IS NULL)
		FROM identity_identifiers WHERE user_id=$1 AND kind IN ('username','phone')`, f.source)
	cdBool(f, `SELECT count(*)=CASE WHEN $2 THEN 0 ELSE 1 END AND
		COALESCE(bool_and(normalized_value=$3 AND lifecycle='reserved' AND released_at IS NULL),true)
		FROM identity_identifiers WHERE user_id=$1 AND kind='email'`, f.source, released, f.email)
	cdBool(f, `SELECT NOT EXISTS (SELECT 1 FROM sessions WHERE user_id=$1)`, f.source)
	cdBool(f, `SELECT count(*)=3 AND count(*) FILTER(WHERE state='recovery_invalidated' AND invalidated_at=$2)=1
		AND count(*) FILTER(WHERE state='consumed' AND invalidated_at IS NULL)=1
		AND count(*) FILTER(WHERE state='superseded' AND invalidated_at IS NULL)=1
		FROM temporary_credentials WHERE user_id=$1`, f.source, f.clock.at)
}

func cdReadArticle(f *cdFixture, id string) article.Article {
	var a article.Article
	cdMust(f.t, f.observer.QueryRow(f.ctx, `SELECT id,author_id,orphaned_from_user_id,title,body,publication_state,
		violation_state,violation_category,deletion_state,deletion_operation_id,version,created_at,updated_at FROM articles WHERE id=$1`, id).
		Scan(&a.ID, &a.AuthorID, &a.OrphanedFromUserID, &a.Title, &a.Body, &a.PublicationState, &a.ViolationState,
			&a.ViolationCategory, &a.DeletionState, &a.DeletionOperationID, &a.Version, &a.CreatedAt, &a.UpdatedAt))
	a.CreatedAt, a.UpdatedAt = a.CreatedAt.UTC(), a.UpdatedAt.UTC()
	return a
}

func cdAssertArticles(f *cdFixture) {
	for _, before := range f.rows {
		want := before
		if before.DeletionState == article.DeletionActive && before.AuthorID != nil {
			want.AuthorID, want.OrphanedFromUserID, want.PublicationState = nil, cdPtr(f.source), article.PublicationHeld
			want.Version, want.UpdatedAt = before.Version+1, f.clock.at
		}
		cdRequire(f.t, reflect.DeepEqual(want, cdReadArticle(f, before.ID)), "article independent full preimage/readback differs")
	}
	cdBool(f, `SELECT count(*) FILTER(WHERE deletion_state='active')+count(*) FILTER(WHERE deletion_state='deletion_pending')=$2
		FROM articles WHERE author_id=$1 OR orphaned_from_user_id=$1`, f.source, len(f.rows))
}

func cdAssertAudit(f *cdFixture, newly, already, pending int, released bool) {
	cdBool(f, `SELECT count(*)=$2 AND COALESCE(bool_and(trace_id=$3 AND actor_kind='system'
		AND actor_system='identity-canonical-delete' AND target_type='article' AND outcome='succeeded'
		AND metadata->>'deleted_user_id'=$1::text),true) FROM audit_events WHERE action='article.orphan.canonical'`, f.source, newly, cdTrace)
	cdBool(f, `SELECT count(*)=1 AND bool_and(trace_id=$2 AND actor_user_id=$3 AND actor_role='admin'
		AND actor_kind='user' AND target_type='user' AND target_id=$1::text AND outcome='succeeded'
		AND metadata->>'mode'=$4 AND metadata->>'reason'='governance'
		AND (metadata->>'newly_orphaned_count')::int=$5 AND (metadata->>'already_orphaned_count')::int=$6
		AND (metadata->>'pending_orphan_count')::int=$7 AND (metadata->>'email_identifier_released')::boolean=$8)
		FROM audit_events WHERE action='identity.user.delete.canonical'`,
		f.source, cdTrace, f.actor.UserID, f.request.Mode, newly, already, pending, released)
	cdReleaseCount(f, map[bool]int{true: 1, false: 0}[released])
	for _, row := range f.rows {
		if row.DeletionState == article.DeletionActive && row.AuthorID != nil {
			cdBool(f, `SELECT count(*)=1 AND bool_and((metadata->>'article_version')::bigint=$2)
				FROM audit_events WHERE action='article.orphan.canonical' AND target_id=$1`, row.ID, row.Version+1)
		}
	}
	cdBool(f, `SELECT COALESCE(bool_and(actor_kind='user' AND actor_user_id=$1 AND actor_role='admin'
		AND metadata->>'mode'=$2),true) FROM audit_events WHERE action='identity.identifier.email.release'`, f.actor.UserID, f.request.Mode)
}

func cdReleaseCount(f *cdFixture, expected int) {
	cdBool(f, `SELECT count(*)=$2 AND COALESCE(bool_and(trace_id=$3 AND target_id=$1::text
		AND target_type='identity_identifier' AND outcome='succeeded' AND metadata->>'kind'='email'),true)
		FROM audit_events WHERE action='identity.identifier.email.release'`, f.source, expected, cdTrace)
}

func cdUnrelated(f *cdFixture, before cdImage) {
	after := cdObserve(f, "after")
	cdRequire(f.t, before["recovery_runs"] == after["recovery_runs"], "canonical changed recovery run authority")
	cdRequire(f.t, before["deletion_operations"] == after["deletion_operations"], "canonical changed pending operation authority")
	for _, table := range []string{"users", "identity_identifiers", "sessions", "articles"} {
		var oldRows, newRows []map[string]json.RawMessage
		cdMust(f.t, json.Unmarshal([]byte(before[table]), &oldRows))
		cdMust(f.t, json.Unmarshal([]byte(after[table]), &newRows))
		for _, row := range oldRows {
			var fields map[string]json.RawMessage
			cdMust(f.t, json.Unmarshal(row["row"], &fields))
			owner := string(fields["user_id"])
			if table == "users" {
				owner = string(fields["id"])
			}
			if table == "articles" {
				owner = string(fields["author_id"])
			}
			if owner == `"`+f.other+`"` || owner == `"`+f.actor.UserID+`"` {
				cdRequire(f.t, cdContainsRow(newRows, row), "actor/bystander SQL row/version changed: "+table)
			}
		}
	}
}

func cdContainsRow(rows []map[string]json.RawMessage, expected map[string]json.RawMessage) bool {
	for _, row := range rows {
		if reflect.DeepEqual(row, expected) {
			return true
		}
	}
	return false
}

func cdAssertRecovery(f *cdFixture) {
	run := f.request.Recovery.Capability.RunID()
	cdBool(f, `SELECT decision='kept_deleted' AND reviewed_by=$3 AND reviewed_at=$4
		AND credential_reset_at IS NULL AND evidence_ref='synthetic-review-evidence'
		FROM recovery_account_reviews WHERE recovery_run_id=$1 AND user_id=$2`, run, f.source, f.actor.UserID, f.clock.at)
	cdBool(f, `SELECT outcome='running' AND backup_ref=$2 AND journal_stage='begun' AND completed_at IS NULL
		FROM recovery_runs WHERE run_id=$1`, run, f.request.Recovery.Capability.BackupRef())
	cdRequire(f.t, f.gate.calls > 0 && f.gate.maintenance && !f.gate.readiness, "controlled Gate changed or not consumed")
	for _, tx := range f.transactions.seen {
		if tx.canonical {
			cdRequire(f.t, tx.runLocks == 1, "canonical recovery did not lock the real run in its caller transaction")
		}
	}
}

func cdReplay(f *cdFixture, pending, already int) {
	before := cdObserve(f, "before_replay")
	var deleted time.Time
	cdMust(f.t, f.observer.QueryRow(f.ctx, `SELECT deleted_at FROM users WHERE id=$1`, f.source).Scan(&deleted))
	rows := make([]article.Article, len(f.rows))
	for i, row := range f.rows {
		rows[i] = cdReadArticle(f, row.ID)
	}
	r := cdDelete(f)
	cdRequire(f.t, r.NewlyOrphanedCount == 0 && r.AlreadyOrphanedCount == already && r.PendingOrphanCount == pending &&
		!r.EmailIdentifierFreed, "replay counted completed transitions twice")
	cdBool(f, `SELECT deleted_at=$2 FROM users WHERE id=$1`, f.source, deleted)
	for _, row := range rows {
		cdRequire(f.t, reflect.DeepEqual(row, cdReadArticle(f, row.ID)), "replay changed article/version")
	}
	cdReleaseCount(f, map[bool]int{true: 1, false: 0}[pending == 0])
	after := cdObserve(f, "after_replay")
	cdRequire(f.t, before["recovery_account_reviews"] == after["recovery_account_reviews"] &&
		before["recovery_runs"] == after["recovery_runs"], "replay rewrote completed review/run facts")
	cdAssertRecovery(f)
}

func cdDispose(f *cdFixture, index int, transfer bool) {
	f.before = cdObserve(f, "before_disposition")
	a := cdReadArticle(f, f.rows[index].ID)
	var r article.OrphanDispositionResult
	var err error
	if transfer {
		r, err = cdCommand(f).TransferOrphan(f.ctx, f.actor, a.ID, a.Version, f.other)
	} else {
		r, err = cdCommand(f).PublishOrphan(f.ctx, f.actor, a.ID, a.Version)
	}
	cdMust(f.t, err)
	cdObserve(f, "after_disposition_commit")
	cdRequire(f.t, r.Article.ID == a.ID && r.Article.Version == a.Version+1 && f.proofCalls > 0, "native disposition/proof missing")
	cdAssertDisposition(f, a, r, transfer)
}

func cdAssertDisposition(f *cdFixture, before article.Article, r article.OrphanDispositionResult, transfer bool) {
	after := cdReadArticle(f, before.ID)
	cdRequire(f.t, after.Title == before.Title && after.Body == before.Body && after.Version == before.Version+1 &&
		after.PublicationState == article.PublicationPublished && after.OrphanedFromUserID == nil &&
		after.ViolationState == before.ViolationState && reflect.DeepEqual(after.ViolationCategory, before.ViolationCategory), "disposition content/state drift")
	if !transfer {
		cdRequire(f.t, after.AuthorID == nil && r.NoticeID == nil, "anonymous publish gained owner/notice")
		return
	}
	cdRequire(f.t, after.AuthorID != nil && *after.AuthorID == f.other && r.NoticeID != nil, "transfer target/notice missing")
	cdBool(f, `SELECT article_ref=$2 AND new_author_id=$3 AND former_deleted_user_id=$4
		FROM transfer_notices WHERE id=$1`, *r.NoticeID, before.ID, f.other, f.source)
	cdBool(f, `SELECT count(*)=2 AND bool_and(status='pending' AND attempt_count=0)
		AND count(*) FILTER(WHERE recipient_kind='former_deleted_author' AND email_snapshot=$2)=1
		AND count(*) FILTER(WHERE recipient_kind='new_author' AND email_snapshot=$3)=1
		FROM transfer_notice_recipients WHERE notice_id=$1`, *r.NoticeID, f.email, "cd_"+f.other+"@example.invalid")
}

func cdRetained(f *cdFixture, expected bool) {
	cdBool(f, `SELECT (retained_email IS NOT NULL)=$2 AND ((NOT $2) OR retained_email=$3)
		FROM users WHERE id=$1`, f.source, expected, f.email)
	cdBool(f, `SELECT count(*)=CASE WHEN $2 THEN 1 ELSE 0 END
		FROM identity_identifiers WHERE user_id=$1 AND kind='email'`, f.source, expected)
	cdBool(f, `SELECT ((active_count+pending_count)>0)=$2 FROM (
		SELECT count(*) FILTER(WHERE deletion_state='active') active_count,
		count(*) FILTER(WHERE deletion_state='deletion_pending') pending_count
		FROM articles WHERE author_id=$1 OR orphaned_from_user_id=$1) counts`, f.source, expected)
}

func cdRetention(t *testing.T, transfer, pending bool) {
	kinds := []string{"owned", "owned"}
	if transfer {
		kinds[1] = "violating"
	}
	if pending {
		kinds = append(kinds, "pending_owned")
	}
	f := cdNew(t, cdCase{kinds: kinds})
	cdDelete(f)
	cdDispose(f, 0, false)
	cdRetained(f, true)
	cdReleaseCount(f, 0)
	cdDispose(f, 1, transfer)
	cdRetained(f, pending)
	cdReleaseCount(f, map[bool]int{true: 0, false: 1}[pending])
	cdBool(f, `SELECT count(*) FILTER(WHERE deletion_state='active')=0
		AND count(*) FILTER(WHERE deletion_state='deletion_pending')=$2
		FROM articles WHERE author_id=$1 OR orphaned_from_user_id=$1`, f.source, map[bool]int{true: 1, false: 0}[pending])
}

func cdReplayDisposition(t *testing.T) {
	f := cdNew(t, cdCase{recovery: true, kinds: []string{"owned", "violating", "held"}})
	cdDelete(f)
	cdDispose(f, 0, false)
	cdDispose(f, 1, true)
	cdRetained(f, true)
	cdReplay(f, 1, 1)
	cdDispose(f, 2, false)
	cdRetained(f, false)
	cdReplay(f, 0, 0)
}

func cdReplayReleased(t *testing.T) {
	f := cdNew(t, cdCase{recovery: true})
	cdDelete(f)
	cdRetained(f, false)
	cdReplay(f, 0, 0)
}

func cdIdentifierReuse(t *testing.T) {
	f := cdNew(t, cdCase{kinds: []string{"owned"}})
	cdDelete(f)
	cdReuse(f, false)
	cdDispose(f, 0, false)
	cdReuse(f, true)
	f.t.Log("reuse_evidence=SQL_UNIQUE_storage_only Register_HTTP_response=false phone_residual=reserved_nonreleased")
}

func cdReuse(f *cdFixture, emailReleased bool) {
	before := cdObserve(f, "before_unique_probes")
	var username string
	cdMust(f.t, f.observer.QueryRow(f.ctx, `SELECT username FROM users WHERE id=$1`, f.source).Scan(&username))
	for kind, value := range map[string]string{"username": username, "email": f.email, "phone": f.phone} {
		cdReuseOne(f, kind, value, kind == "email" && emailReleased)
	}
	cdSame(f.t, before, cdObserve(f, "after_unique_probes"))
}

func cdReuseOne(f *cdFixture, kind, value string, available bool) {
	tx, err := f.pool.Begin(f.ctx)
	cdMust(f.t, err)
	defer cdRollback(tx)
	// Avoid the probe owner's (user_id,kind) PK hiding the value UNIQUE.
	_, err = tx.Exec(f.ctx, `DELETE FROM identity_identifiers WHERE user_id=$1 AND kind=$2`, f.other, kind)
	cdMust(f.t, err)
	_, err = tx.Exec(f.ctx, `INSERT INTO identity_identifiers(user_id,kind,normalized_value,lifecycle,created_at)
		VALUES ($1,$2,$3,'active',$4)`, f.other, kind, value, f.clock.at)
	if available {
		cdMust(f.t, err)
		return
	}
	var pg *pgconn.PgError
	cdRequire(f.t, errors.As(err, &pg) && pg.Code == "23505" &&
		pg.ConstraintName == "identity_identifiers_kind_normalized_value_key", "reserved identifier value UNIQUE not enforced")
}

// cdFrozenMigrations selects the migration input used by the historical PG fixture.
func cdFrozenMigrations(input fs.FS) ([]migration.Migration, error) {
	names := []string{
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
	selected := fstest.MapFS{}
	for _, name := range names {
		path := "migrations/" + name
		sql, err := fs.ReadFile(input, path)
		if err != nil {
			return nil, fmt.Errorf("read historical migration %s: %w", name, err)
		}
		selected[path] = &fstest.MapFile{Data: sql}
	}
	return migration.Load(selected, "migrations")
}

// TestB13CFrozenCanonicalDeleteMigrations exercises only fixture loading; it never opens PostgreSQL.
func TestB13CFrozenCanonicalDeleteMigrations(t *testing.T) {

	for _, tc := range cdMigrationCases() {
		t.Run(tc.name, func(t *testing.T) {
			input := cdMigrationInputs()
			if tc.change != nil {
				tc.change(input)
			}
			got, err := cdFrozenMigrations(input)
			if tc.bad {
				if err == nil {
					t.Fatal("invalid historical chain accepted")
				}
				return
			}
			if err != nil {
				t.Fatalf("load historical chain: %v", err)
			}
			cdCheckHistoricalMigrations(t, got)
		})
	}
	t.Run("NoDirectoryEnumeration", func(t *testing.T) {
		got, err := cdFrozenMigrations(cdHistoricalFilesOnly{cdMigrationInputs()})
		if err != nil {
			t.Fatal(err)
		}
		cdCheckHistoricalMigrations(t, got)
	})
	t.Run("RepositoryHistoricalChain", func(t *testing.T) {
		got, err := cdFrozenMigrations(os.DirFS(filepath.Join("..", "..", "..")))
		if err != nil || len(got) != 11 {
			t.Fatalf("repository historical chain count=%d error=%v", len(got), err)
		}
	})
}

func cdMigrationInputs() fstest.MapFS {
	return fstest.MapFS{
		"migrations/0001_shared.sql":                           &fstest.MapFile{Data: []byte("SELECT 1;")},
		"migrations/0002_identity.sql":                         &fstest.MapFile{Data: []byte("SELECT 2;")},
		"migrations/0003_article.sql":                          &fstest.MapFile{Data: []byte("SELECT 3;")},
		"migrations/0004_idempotency.sql":                      &fstest.MapFile{Data: []byte("SELECT 4;")},
		"migrations/0005_recovery_runs.sql":                    &fstest.MapFile{Data: []byte("SELECT 5;")},
		"migrations/0006_user_governance_indexes.sql":          &fstest.MapFile{Data: []byte("SELECT 6;")},
		"migrations/0007_idempotency_response_snapshot_v1.sql": &fstest.MapFile{Data: []byte("SELECT 7;")},
		"migrations/0008_password_reset_durable.sql":           &fstest.MapFile{Data: []byte("SELECT 8;")},
		"migrations/0009_article_replay_attempt.sql":           &fstest.MapFile{Data: []byte("SELECT 9;")},
		"migrations/0010_recovery_runtime_metadata.sql":        &fstest.MapFile{Data: []byte("SELECT 10;")},
		"migrations/0011_article_replay_publication.sql":       &fstest.MapFile{Data: []byte("SELECT 11;")},
	}
}

func cdCheckHistoricalMigrations(t *testing.T, got []migration.Migration) {
	t.Helper()
	if len(got) != 11 {
		t.Fatalf("historical count=%d, want 11", len(got))
	}
	for i, item := range got {
		if item.Version != i+1 || item.SQL != fmt.Sprintf("SELECT %d;", i+1) {
			t.Fatalf("historical position %d: version=%d SQL=%q", i, item.Version, item.SQL)
		}
	}
}

type cdHistoricalFilesOnly struct{ fs.FS }

func (f cdHistoricalFilesOnly) Open(name string) (fs.File, error) {
	if name == "migrations" {
		return nil, fmt.Errorf("historical fixture must not enumerate the live directory")
	}
	return f.FS.Open(name)
}

type cdMigrationCase struct {
	name   string
	change func(fstest.MapFS)
	bad    bool
}

func cdMigrationCases() []cdMigrationCase {
	return []cdMigrationCase{
		{name: "FrozenChain"},
		{name: "Future0012Ignored", change: func(f fstest.MapFS) { f["migrations/0012_future.sql"] = &fstest.MapFile{Data: []byte("SELECT 10;")} }},
		{name: "FutureBadSQLIgnored", change: func(f fstest.MapFS) { f["migrations/0012_future.sql"] = &fstest.MapFile{Data: []byte("COMMIT;")} }},
		{name: "FutureBadNameIgnored", change: func(f fstest.MapFS) { f["migrations/future.sql"] = &fstest.MapFile{Data: []byte("COMMIT;")} }},
		{name: "MissingHistoricalRejected", change: func(f fstest.MapFS) { delete(f, "migrations/0005_recovery_runs.sql") }, bad: true},
		{name: "WrongHistoricalNameRejected", change: func(f fstest.MapFS) {
			f["migrations/0005_wrong.sql"] = f["migrations/0005_recovery_runs.sql"]
			delete(f, "migrations/0005_recovery_runs.sql")
		}, bad: true},
		{name: "InvalidHistoricalSQLRejected", change: func(f fstest.MapFS) {
			f["migrations/0005_recovery_runs.sql"] = &fstest.MapFile{Data: []byte("COMMIT;")}
		}, bad: true},
	}
}
