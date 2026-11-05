//go:build integration

package articleintegration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"han/internal/article"
	"han/internal/audit"
	"han/internal/identity"
	"han/internal/mailer"
	"han/internal/platform"
	"han/internal/privacy"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/puddle/v2"
)

// This file owns the independent PGPERF-A harness. It never redefines or wraps
// noticeHarness, noticePoolConfig, newNoticeHarness, newAdminPG, newAdminPGPool
// or adminPGMigrate. Every schema is retained; nothing is DROPped or truncated.

const (
	pgPerfDSNEnv         = "HAN_TEST_POSTGRES_DSN"
	pgPerfEvidenceEnv    = "HAN_PG_PERF_EVIDENCE_DIR"
	pgPerfRunIDEnv       = "HAN_PG_PERF_RUN_ID"
	pgPerfDatabaseEnv    = "HAN_PG_PERF_DATABASE"
	pgPerfWindowStartEnv = "HAN_PG_PERF_WINDOW_START"
	pgPerfBaseBytesEnv   = "HAN_PG_PERF_BASE_BYTES"
	pgPerfBaselineDirEnv = "HAN_PG_PERF_BASELINE_DIR"
	pgPerfDefaultDBName  = "han_perf_s24"
	pgPerfSchemaPrefix   = "han_pgperf_r04_"
	pgPerfAppNamePrefix  = "han_pgperf_r04_"
	pgPerfSchemaMaxBytes = 63

	pgPerfMeasureStatementTimeout = 30 * time.Second
	pgPerfNegativeLockTimeout     = 5 * time.Second
	pgPerfSeedStatementTimeout    = 0 * time.Second

	pgPerfMaxArticleBodyRunes  = 20000
	pgPerfMaxArticleTitleRunes = 200

	pgPerfModeTimed3Rep   = "TIMED-3REP"
	pgPerfModeSingleRun   = "SINGLE-RUN"
	pgPerfModeCorrectness = "CORRECTNESS-ONLY"
)

var pgPerfMigrationFiles = []string{
	"0001_shared.sql",
	"0002_identity.sql",
	"0003_article.sql",
	"0004_idempotency.sql",
	"0005_recovery_runs.sql",
	"0006_user_governance_indexes.sql",
	"0007_idempotency_response_snapshot_v1.sql",
	"0008_password_reset_durable.sql",
	"0009_article_replay_attempt.sql",
}

// pgPerfTier is the fixed r04 data/context/pool matrix.
type pgPerfTier struct {
	name          string
	code          string
	contextBudget time.Duration
	seedBudget    time.Duration
	maxConns      int32
	articles      int
	users         int
	deletions     int
	notices       int
	recipients    int
	audits        int
	bodyMin       int
	bodyMax       int
	nearLimitRate float64
	diskLimit     int64
}

var (
	pgPerfTierSmall = pgPerfTier{
		name: "small", code: "s", contextBudget: 300 * time.Second, seedBudget: 120 * time.Second,
		maxConns: 4, articles: 1000, users: 100, deletions: 100, notices: 500, recipients: 1000,
		audits: 100, bodyMin: 200, bodyMax: 2048, nearLimitRate: 0.01, diskLimit: 1024 * 1024 * 1024,
	}
	pgPerfTierMedium = pgPerfTier{
		name: "medium", code: "m", contextBudget: 780 * time.Second, seedBudget: 600 * time.Second,
		maxConns: 8, articles: 10000, users: 1000, deletions: 1000, notices: 5000, recipients: 10000,
		audits: 1000, bodyMin: 512, bodyMax: 10240, nearLimitRate: 0.005, diskLimit: 5905580032,
	}
	pgPerfTierLarge = pgPerfTier{
		name: "large", code: "l", contextBudget: 1980 * time.Second, seedBudget: 1800 * time.Second,
		maxConns: 16, articles: 100000, users: 10000, deletions: 10000, notices: 50000, recipients: 100000,
		audits: 10000, bodyMin: 512, bodyMax: 4096, nearLimitRate: 0.001, diskLimit: 9126805504,
	}
)

type pgPerfSchemaRecord struct {
	Schema    string `json:"schema"`
	Tier      string `json:"tier"`
	Case      string `json:"case"`
	CreatedAt string `json:"created_at"`
	Retained  bool   `json:"retained"`
	Length    int    `json:"length_bytes"`
}

type pgPerfMigrationRecord struct {
	Order  int    `json:"order"`
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
}

type pgPerfSeedManifest struct {
	Schema      string `json:"schema"`
	Tier        string `json:"tier"`
	Case        string `json:"case"`
	Users       int    `json:"users"`
	Articles    int    `json:"articles"`
	Hidden      int    `json:"hidden_articles"`
	NearLimit   int    `json:"near_limit_articles"`
	Notices     int    `json:"notices"`
	Recipients  int    `json:"recipients"`
	Audits      int    `json:"audits"`
	SeedMS      int64  `json:"seed_ms"`
	AnalyzeMS   int64  `json:"analyze_ms"`
	SeedTimeout string `json:"seed_statement_timeout"`
	Note        string `json:"note,omitempty"`
}

type pgPerfPointSummary struct {
	Point            string         `json:"point"`
	Scene            string         `json:"scene"`
	Tier             string         `json:"tier"`
	Schema           string         `json:"schema"`
	Mode             string         `json:"mode"`
	Concurrency      int            `json:"concurrency"`
	Repeats          int            `json:"repeats"`
	WarmupMS         int64          `json:"warmup_ms"`
	WindowMS         int64          `json:"window_ms"`
	ObservedMS       int64          `json:"observed_ms"`
	Attempts         int            `json:"attempts"`
	Successes        int            `json:"successes"`
	Errors           int            `json:"errors"`
	Rejected         int            `json:"rejected"`
	N                int            `json:"n"`
	Exploratory      bool           `json:"exploratory"`
	MaxConns         int32          `json:"max_conns"`
	AcquireCount     int64          `json:"acquire_count"`
	AcquireMS        float64        `json:"acquire_duration_ms"`
	EmptyAcquire     int64          `json:"empty_acquire_count"`
	CancelAcquire    int64          `json:"canceled_acquire_count"`
	P50MS            float64        `json:"p50_ms"`
	P95MS            float64        `json:"p95_ms"`
	P99MS            float64        `json:"p99_ms"`
	MinMS            float64        `json:"min_ms"`
	MaxMS            float64        `json:"max_ms"`
	MeanMS           float64        `json:"mean_ms"`
	QPS              float64        `json:"qps"`
	SeedMS           int64          `json:"seed_ms"`
	DBBytesDelta     int64          `json:"db_bytes_delta"`
	Truncated        bool           `json:"window_truncated"`
	CompletedRepeats int            `json:"completed_repeats"`
	ActiveMS         int64          `json:"actual_active_ms"`
	TailMS           int64          `json:"cancellation_tail_ms"`
	PointWallNS      int64          `json:"actual_point_wall_ns"`
	TailNS           int64          `json:"actual_point_tail_ns"`
	BoundaryCanceled int            `json:"boundary_canceled"`
	Late             int            `json:"late"`
	Unknown          int            `json:"unknown"`
	Buckets          []pgPerfBucket `json:"buckets,omitempty"`
	Note             string         `json:"note,omitempty"`
}

// pgPerfBucket keeps W02's typical and near-limit body classes separate inside
// one measurement point, as r04 §3 requires.
type pgPerfBucket struct {
	Name             string  `json:"name"`
	N                int     `json:"n"`
	Successes        int     `json:"successes"`
	Errors           int     `json:"errors"`
	Rejected         int     `json:"rejected"`
	BoundaryCanceled int     `json:"boundary_canceled"`
	Late             int     `json:"late"`
	P50MS            float64 `json:"p50_ms"`
	P95MS            float64 `json:"p95_ms"`
	P99MS            float64 `json:"p99_ms"`
	MinMS            float64 `json:"min_ms"`
	MaxMS            float64 `json:"max_ms"`
	MeanMS           float64 `json:"mean_ms"`
}

type pgPerfSample struct {
	Bucket     string `json:"bucket,omitempty"`
	Point      string `json:"point"`
	Repeat     int    `json:"repeat"`
	Worker     int    `json:"worker"`
	Seq        int64  `json:"seq"`
	StartUS    int64  `json:"start_us"`
	DurationUS int64  `json:"duration_us"`
	OK         bool   `json:"ok"`
	Error      string `json:"error,omitempty"`
}

type pgPerfEvidence struct {
	root            string
	tier            string
	runID           string
	started         time.Time
	once            sync.Once
	migOnce         sync.Once
	mu              sync.Mutex
	schemas         []pgPerfSchemaRecord
	seeds           []pgPerfSeedManifest
	points          []pgPerfPointSummary
	blocked         []string
	blockedSeen     map[string]bool
	fingerprintBase map[string]any
	fingerprints    []map[string]any
	finalErr        error
	baseDBBytes     int64
	tierBaseDBBytes int64
	stopReason      string
}

// pgPerfUUID derives a stable RFC-4122-shaped identifier from non-secret parts.
func pgPerfUUID(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	b := sum[:16]
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func pgPerfSeedInt(seed string) int64 {
	sum := sha256.Sum256([]byte(seed))
	var value int64
	for i := 0; i < 8; i++ {
		value = value<<8 | int64(sum[i])
	}
	return value
}

func pgPerfRunID(t testing.TB) string {
	t.Helper()
	if value := strings.TrimSpace(os.Getenv(pgPerfRunIDEnv)); value != "" {
		return value
	}
	return "r04-" + time.Now().UTC().Format("20060102T150405")
}

func pgPerfDatabaseName(t testing.TB) string {
	t.Helper()
	if value := strings.TrimSpace(os.Getenv(pgPerfDatabaseEnv)); value != "" {
		return value
	}
	return pgPerfDefaultDBName
}

func pgPerfValidDatabaseName(name string) bool {
	if len(name) == 0 || len(name) > 63 {
		return false
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return false
	}
	return true
}

var (
	pgPerfDatabaseOnce    sync.Once
	pgPerfDatabaseErr     error
	pgPerfDatabaseCreated bool
)

// pgPerfEnsureDatabase validates the existing authorized database without DDL.
func pgPerfEnsureDatabase(t testing.TB, ctx context.Context, appName string) {
	t.Helper()
	name := pgPerfDatabaseName(t)
	if !pgPerfValidDatabaseName(name) {
		t.Fatalf("pgperf database name must match [a-z0-9_]{1,63}, got %q", name)
	}
	pgPerfDatabaseOnce.Do(func() {
		pgPerfDatabaseCreated, pgPerfDatabaseErr = pgPerfCreateDatabase(ctx, name, appName)
	})
	if pgPerfDatabaseErr != nil {
		t.Fatalf("ensure pgperf database %s: %v", name, pgPerfDatabaseErr)
	}
}

func pgPerfCreateDatabase(ctx context.Context, name, appName string) (bool, error) {
	dsn := strings.TrimSpace(os.Getenv(pgPerfDSNEnv))
	if dsn == "" {
		return false, errors.New("HAN_TEST_POSTGRES_DSN missing; no skip")
	}
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return false, fmt.Errorf("parse bootstrap configuration: %T (value withheld)", err)
	}
	ip := net.ParseIP(config.Host)
	if ip == nil || !ip.IsLoopback() || config.Port != 5433 || name != pgPerfDefaultDBName {
		return false, errors.New("pgperf requires the existing authorized han_perf_s24 database on loopback:5433")
	}
	config.Database = name
	if config.RuntimeParams == nil {
		config.RuntimeParams = map[string]string{}
	}
	config.RuntimeParams["application_name"] = appName + "_bootstrap"
	config.RuntimeParams["statement_timeout"] = "60000"
	bootstrapCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	conn, err := pgx.ConnectConfig(bootstrapCtx, config)
	if err != nil {
		return false, fmt.Errorf("connect bootstrap database: %T (value withheld)", err)
	}
	defer conn.Close(context.Background())
	var host string
	var port, version int
	if err := conn.QueryRow(bootstrapCtx, "SELECT host(inet_server_addr()),inet_server_port(),current_setting('server_version_num')::int").Scan(&host, &port, &version); err != nil {
		return false, fmt.Errorf("probe existing database: %T", err)
	}
	if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() || port != 5433 || version != 180006 {
		return false, errors.New("pgperf requires PostgreSQL 18.6 on the authorized endpoint")
	}
	return false, nil
}

func pgPerfEvidenceRoot(t testing.TB) string {
	t.Helper()
	if value := strings.TrimSpace(os.Getenv(pgPerfEvidenceEnv)); value != "" {
		return value
	}
	t.Fatal("HAN_PG_PERF_EVIDENCE_DIR must name a fresh continuation author run directory")
	return ""
}

func pgPerfSchemaName(t testing.TB, tierCode, caseCode, runID string) string {
	t.Helper()
	if len(caseCode) == 0 || len(caseCode) > 6 {
		t.Fatalf("pgperf case code must be 1..6 ascii bytes, got %d", len(caseCode))
	}
	digest := sha256.Sum256([]byte(runID))
	name := pgPerfSchemaPrefix + tierCode + caseCode + "_" + hex.EncodeToString(digest[:8])
	if len(name) > pgPerfSchemaMaxBytes {
		t.Fatalf("pgperf schema exceeds %d bytes: %d", pgPerfSchemaMaxBytes, len(name))
	}
	return name
}

func pgPerfApplicationName(runID string) string {
	return pgPerfAppNamePrefix + strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, runID)
}

var (
	pgPerfEvidenceOnce sync.Once
	pgPerfEvidenceInst *pgPerfEvidence
)

func pgPerfGlobalEvidence(t testing.TB) *pgPerfEvidence {
	t.Helper()
	pgPerfEvidenceOnce.Do(func() {
		root := pgPerfEvidenceRoot(t)
		runID := pgPerfRunID(t)
		allowed, err := filepath.Abs(filepath.Join("..", "..", "..", ".spec", "wip", "article-management", "evidence-pg-perf-r04", "continuation-20261104", "author"))
		if err != nil {
			t.Fatal("resolve authorized evidence directory")
		}
		absolute, err := filepath.Abs(root)
		if err != nil {
			t.Fatal("resolve evidence directory")
		}
		relative, err := filepath.Rel(allowed, absolute)
		if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			t.Fatal("evidence root must be a new directory below the continuation author directory")
		}
		if err := os.MkdirAll(filepath.Dir(root), 0o755); err != nil {
			t.Fatal("create evidence parent directory")
		}
		if err := os.Mkdir(root, 0o755); err != nil {
			t.Fatalf("create evidence root: %v", err)
		}
		pgPerfEvidenceInst = &pgPerfEvidence{root: root, tier: "all", runID: runID, started: time.Now().UTC()}
		t.Cleanup(func() {
			if err := os.MkdirAll(root, 0o755); err != nil {
				t.Errorf("recreate evidence root: %v", err)
			}
		})
	})
	return pgPerfEvidenceInst
}

func (e *pgPerfEvidence) path(rel string) string {
	return filepath.Join(e.root, rel)
}

func (e *pgPerfEvidence) write(rel string, data []byte) {
	if e == nil {
		return
	}
	full := e.path(rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		e.noteErr(fmt.Errorf("mkdir %s: %w", rel, err))
		return
	}
	if err := os.WriteFile(full, data, 0o644); err != nil {
		e.noteErr(fmt.Errorf("write %s: %w", rel, err))
	}
}

func (e *pgPerfEvidence) writeJSON(rel string, value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		e.noteErr(fmt.Errorf("encode %s: %w", rel, err))
		return
	}
	e.write(rel, append(data, '\n'))
}

func (e *pgPerfEvidence) writeText(rel, text string) {
	e.write(rel, []byte(text))
}

func (e *pgPerfEvidence) noteErr(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.finalErr = errors.Join(e.finalErr, err)
}

func (e *pgPerfEvidence) addSchema(record pgPerfSchemaRecord) {
	e.mu.Lock()
	e.schemas = append(e.schemas, record)
	e.mu.Unlock()
	e.writeJSONExclusive(filepath.Join("events", record.Schema+".created.json"), record)
	e.checkpointInventory()
}

func (e *pgPerfEvidence) checkpointInventory() {
	e.mu.Lock()
	schemas := append([]pgPerfSchemaRecord(nil), e.schemas...)
	e.mu.Unlock()
	e.writeJSON("environment/schema-inventory.json", map[string]any{"run_id": e.runID, "run_ids": []string{e.runID}, "generated_at": time.Now().UTC().Format(time.RFC3339Nano), "planned_logical_scenarios": 41, "actual_schemas": len(schemas), "schemas": schemas, "calibration_invocations": pgPerfCalibrationRepeats.Load(), "checkpoint": true})
}

func (e *pgPerfEvidence) addSeed(record pgPerfSeedManifest) {
	e.mu.Lock()
	e.seeds = append(e.seeds, record)
	seeds := append([]pgPerfSeedManifest(nil), e.seeds...)
	e.mu.Unlock()
	e.writeJSON("seed-manifest.json", seeds)
}

func (e *pgPerfEvidence) addPoint(record pgPerfPointSummary) {
	e.mu.Lock()
	e.points = append(e.points, record)
	points := append([]pgPerfPointSummary(nil), e.points...)
	e.mu.Unlock()
	e.writeJSON(filepath.Join("raw", record.Point+"."+e.runID+".summary.json"), record)
	_ = points
}

func (e *pgPerfEvidence) addBlocked(reason string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.blockedSeen == nil {
		e.blockedSeen = map[string]bool{}
	}
	if e.blockedSeen[reason] {
		return
	}
	e.blockedSeen[reason] = true
	e.blocked = append(e.blocked, reason)
}

func (e *pgPerfEvidence) addMigrations(schema string, records []pgPerfMigrationRecord) {
	type migrationDoc struct {
		Schema     string                  `json:"schema"`
		RecordedAt string                  `json:"recorded_at"`
		Migrations []pgPerfMigrationRecord `json:"migrations"`
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	doc := migrationDoc{Schema: schema, RecordedAt: time.Now().UTC().Format(time.RFC3339Nano), Migrations: records}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		e.finalErr = errors.Join(e.finalErr, err)
		return
	}
	e.writeText(filepath.Join("migrations", schema+".json"), string(data)+"\n")
	e.migOnce.Do(func() {
		e.writeJSON(filepath.Join("migrations", "sha256.json"), doc)
	})
}

func (e *pgPerfEvidence) writeRaw(point string, samples []pgPerfSample) {
	if len(samples) == 0 {
		return
	}
	var builder strings.Builder
	for _, sample := range samples {
		data, err := json.Marshal(sample)
		if err != nil {
			e.noteErr(err)
			continue
		}
		builder.Write(data)
		builder.WriteByte('\n')
	}
	e.writeText(filepath.Join("raw", point+"."+e.runID+".jsonl"), builder.String())
}

func (e *pgPerfEvidence) writePlan(name string, plan any) {
	e.writeJSON(filepath.Join("plans", name+"."+e.runID+".json"), plan)
}

func (e *pgPerfEvidence) readJSON(rel string, target any) bool {
	data, err := os.ReadFile(filepath.Join(e.root, filepath.FromSlash(rel)))
	if err != nil {
		return false
	}
	return json.Unmarshal(data, target) == nil
}

func (e *pgPerfEvidence) readText(rel string) (string, bool) {
	data, err := os.ReadFile(filepath.Join(e.root, filepath.FromSlash(rel)))
	if err != nil {
		return "", false
	}
	return string(data), true
}

func appendUniqueStrings(values []string, value string) []string {
	out := append([]string(nil), values...)
	for _, existing := range out {
		if existing == value {
			return out
		}
	}
	return append(out, value)
}

// Finalize writes the aggregate evidence tree and checksums. Independent tier
// processes append to the same tree; earlier evidence is never overwritten or
// dropped, per r04 section 9.5.
func (e *pgPerfEvidence) Finalize(t testing.TB) {
	t.Helper()
	if e == nil {
		return
	}
	e.once.Do(func() {
		e.mu.Lock()
		schemas := append([]pgPerfSchemaRecord(nil), e.schemas...)
		seeds := append([]pgPerfSeedManifest(nil), e.seeds...)
		points := append([]pgPerfPointSummary(nil), e.points...)
		blocked := append([]string(nil), e.blocked...)
		e.mu.Unlock()

		var priorInventory struct {
			RunIDs  []string             `json:"run_ids"`
			Schemas []pgPerfSchemaRecord `json:"schemas"`
		}
		_ = e.readJSON("environment/schema-inventory.json", &priorInventory)
		mergedSchemas := append([]pgPerfSchemaRecord(nil), schemas...)
		knownSchema := map[string]bool{}
		for _, record := range schemas {
			knownSchema[record.Schema] = true
		}
		for _, record := range priorInventory.Schemas {
			if !knownSchema[record.Schema] {
				mergedSchemas = append(mergedSchemas, record)
			}
		}
		sort.Slice(mergedSchemas, func(i, j int) bool { return mergedSchemas[i].Schema < mergedSchemas[j].Schema })
		e.writeJSON("environment/schema-inventory.json", map[string]any{
			"run_id":                    e.runID,
			"run_ids":                   appendUniqueStrings(priorInventory.RunIDs, e.runID),
			"generated_at":              time.Now().UTC().Format(time.RFC3339Nano),
			"planned_logical_scenarios": 41,
			"actual_schemas":            len(mergedSchemas),
			"calibration_invocations":   pgPerfCalibrationRepeats.Load(),
			"schemas":                   mergedSchemas,
		})

		var priorSeeds []pgPerfSeedManifest
		_ = e.readJSON("seed-manifest.json", &priorSeeds)
		mergedSeeds := append([]pgPerfSeedManifest(nil), seeds...)
		knownSeed := map[string]bool{}
		for _, manifest := range seeds {
			knownSeed[manifest.Schema] = true
		}
		for _, manifest := range priorSeeds {
			if !knownSeed[manifest.Schema] {
				mergedSeeds = append(mergedSeeds, manifest)
			}
		}
		sort.Slice(mergedSeeds, func(i, j int) bool { return mergedSeeds[i].Schema < mergedSeeds[j].Schema })
		e.writeJSON("seed-manifest.json", mergedSeeds)

		var summary strings.Builder
		summary.WriteString("# PGPERF r04 evidence summary\n\n")
		summary.WriteString("Run: `" + e.runID + "`  \n")
		summary.WriteString("Started: " + e.started.Format(time.RFC3339) + "  \n")
		summary.WriteString("Generated: " + time.Now().UTC().Format(time.RFC3339) + "\n\n")
		summary.WriteString("All figures are observational on a shared development machine. ")
		summary.WriteString("They are not target-environment SLO evidence and do not imply capacity approval.\n\n")
		summary.WriteString("> Capacity limits: small <= 1.0 GB / medium <= 5.5 GB / large <= 8.5 GB / total <= 15 GB. ")
		summary.WriteString("2026-11-04 本轮用户允许采用建议值，Lead START 明确采用；原 r04 small=0.5/total=14 未修改，新值待本轮独立 Review。单位为 GiB（2^30 bytes）。\n\n")
		summary.WriteString("| Point | Tier | Mode | Concurrency | N | P50 ms | P95 ms | QPS | Errors | Exploratory |\n")
		summary.WriteString("|---|---|---|---:|---:|---:|---:|---:|---:|---|\n")
		for _, point := range points {
			fmt.Fprintf(&summary, "| %s | %s | %s | %d | %d | %.3f | %.3f | %.3f | %d | %t |\n",
				point.Point, point.Tier, point.Mode, point.Concurrency, point.N,
				point.P50MS, point.P95MS, point.QPS, point.Errors, point.Exploratory)
		}
		summary.WriteString("\n## Seeds\n\n")
		summary.WriteString("| Schema | Tier | Case | Articles | Hidden | Near-limit | Seed ms | ANALYZE ms |\n")
		summary.WriteString("|---|---|---|---:|---:|---:|---:|---:|\n")
		for _, seed := range seeds {
			fmt.Fprintf(&summary, "| %s | %s | %s | %d | %d | %d | %d | %d |\n",
				seed.Schema, seed.Tier, seed.Case, seed.Articles, seed.Hidden, seed.NearLimit, seed.SeedMS, seed.AnalyzeMS)
		}
		summary.WriteString("\n## Schemas\n\n")
		for _, schema := range schemas {
			fmt.Fprintf(&summary, "- `%s` tier=%s case=%s retained=%t\n", schema.Schema, schema.Tier, schema.Case, schema.Retained)
		}
		if e.finalErr != nil {
			summary.WriteString("\n## Evidence errors\n\n```\n" + e.finalErr.Error() + "\n```\n")
		}
		summaryText := summary.String()
		if prior, ok := e.readText("summary.md"); ok && strings.TrimSpace(prior) != "" {
			summaryText = prior + "\n---\n\n" + summaryText
		}
		e.writeText("summary.md", summaryText)

		lines := make([]string, 0, len(blocked)+4)
		knownBlocked := map[string]bool{}
		for _, reason := range blocked {
			knownBlocked[reason] = true
			lines = append(lines, "- "+reason)
		}
		if prior, ok := e.readText("blocked.md"); ok {
			for _, line := range strings.Split(prior, "\n") {
				line = strings.TrimSpace(line)
				if !strings.HasPrefix(line, "- ") || knownBlocked[strings.TrimPrefix(line, "- ")] {
					continue
				}
				lines = append(lines, line)
			}
		}
		var blockedDoc strings.Builder
		blockedDoc.WriteString("# Blocked / unavailable\n\n")
		if len(lines) == 0 {
			blockedDoc.WriteString("No blocked item was recorded by this run.\n")
		}
		for _, line := range lines {
			blockedDoc.WriteString(line + "\n")
		}
		e.writeText("blocked.md", blockedDoc.String())

		e.writeChecksums(t)
	})
	e.mu.Lock()
	err := e.finalErr
	e.mu.Unlock()
	if err != nil {
		t.Errorf("evidence persistence failed: %v", err)
	}
}

func (e *pgPerfEvidence) writeJSONExclusive(rel string, value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		e.noteErr(err)
		return
	}
	path := e.path(rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		e.noteErr(err)
		return
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		e.noteErr(fmt.Errorf("immutable evidence %s: %w", rel, err))
		return
	}
	_, writeErr := file.Write(append(data, '\n'))
	if err := errors.Join(writeErr, file.Close()); err != nil {
		e.noteErr(err)
	}
}
func (e *pgPerfEvidence) writeChecksums(t testing.TB) {
	t.Helper()
	var lines []string
	err := filepath.WalkDir(e.root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Base(path) == "checksums.txt" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		rel, err := filepath.Rel(e.root, path)
		if err != nil {
			return err
		}
		lines = append(lines, hex.EncodeToString(sum[:])+"  "+filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Errorf("checksum evidence tree: %v", err)
		return
	}
	sort.Strings(lines)
	e.writeText("checksums.txt", strings.Join(lines, "\n")+"\n")
}

func pgPerfDiskFree(path string) (int64, string) {
	if runtime.GOOS != "windows" {
		return 0, "unavailable"
	}
	proc := syscall.NewLazyDLL("kernel32.dll").NewProc("GetDiskFreeSpaceExW")
	if err := proc.Find(); err != nil {
		return 0, "unavailable"
	}
	pointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, "unavailable"
	}
	var available, total, free uint64
	ok, _, _ := proc.Call(uintptr(unsafe.Pointer(pointer)), uintptr(unsafe.Pointer(&available)), uintptr(unsafe.Pointer(&total)), uintptr(unsafe.Pointer(&free)))
	if ok == 0 {
		return 0, "unavailable"
	}
	return int64(available), "ok"
}

func pgPerfRepoHead() string {
	head, err := os.ReadFile(filepath.Join("..", "..", "..", ".git", "HEAD"))
	if err != nil {
		return "unknown"
	}
	value := strings.TrimSpace(string(head))
	if strings.HasPrefix(value, "ref: ") {
		ref := strings.TrimSpace(strings.TrimPrefix(value, "ref: "))
		if data, err := os.ReadFile(filepath.Join("..", "..", "..", ".git", ref)); err == nil {
			return strings.TrimSpace(string(data))
		}
		return "unknown"
	}
	return value
}

func (e *pgPerfEvidence) writeFingerprint(h *pgPerfHarness, settings map[string]string, extra map[string]any) {
	entry := map[string]any{"application": h.appName}
	for key, value := range extra {
		entry[key] = value
	}
	e.mu.Lock()
	e.fingerprints = append(e.fingerprints, entry)
	fingerprints := append([]map[string]any(nil), e.fingerprints...)
	if e.fingerprintBase == nil {
		e.fingerprintBase = map[string]any{
			"run_id":                    e.runID,
			"database_name":             pgPerfDatabaseName(h.t),
			"database_created":          pgPerfDatabaseCreated,
			"generated_at":              time.Now().UTC().Format(time.RFC3339Nano),
			"head":                      pgPerfRepoHead(),
			"go_version":                runtime.Version(),
			"goos":                      runtime.GOOS,
			"goarch":                    runtime.GOARCH,
			"gomaxprocs":                runtime.GOMAXPROCS(0),
			"pg_settings":               settings,
			"tiers":                     pgPerfTierFingerprint(),
			"context_budget":            map[string]string{"small": "300s", "medium": "780s", "large": "1980s"},
			"seed_budget":               map[string]string{"small": "120s", "medium": "600s", "large": "1800s"},
			"pool_max_conns":            map[string]int{"small": 4, "medium": 8, "large": 16},
			"application":               h.appName,
			"planned_logical_scenarios": 41,
			"timeouts": map[string]string{
				"measure_statement_timeout": "30s",
				"seed_statement_timeout":    "0",
				"negative_lock_timeout":     "5s",
				"default_lock_timeout":      "0",
			},
			"top_level_tests": []string{"TestPgPerfSmall", "TestPgPerfMedium", "TestPgPerfLarge", "BenchmarkPgPerfHarness"},
			"disk_limits": map[string]string{
				"small": "1.0 GiB", "medium": "5.5 GiB", "large": "8.5 GiB", "total": "15 GiB",
			},
			"disk_limits_lead_override": pgPerfLeadDiskOverride(),
		}
	}
	doc := make(map[string]any, len(e.fingerprintBase)+1)
	for key, value := range e.fingerprintBase {
		doc[key] = value
	}
	e.mu.Unlock()
	var priorFingerprint struct {
		RunIDs    []string         `json:"run_ids"`
		Harnesses []map[string]any `json:"harnesses"`
	}
	_ = e.readJSON("environment/fingerprint.json", &priorFingerprint)
	mergedHarnesses := make([]map[string]any, 0, len(priorFingerprint.Harnesses)+len(fingerprints))
	knownHarness := map[string]bool{}
	for _, entry := range append(append([]map[string]any(nil), priorFingerprint.Harnesses...), fingerprints...) {
		key := fmt.Sprint(entry["application"], "|", entry["case"])
		if knownHarness[key] {
			continue
		}
		knownHarness[key] = true
		mergedHarnesses = append(mergedHarnesses, entry)
	}
	doc["harnesses"] = mergedHarnesses
	doc["run_ids"] = appendUniqueStrings(priorFingerprint.RunIDs, e.runID)
	e.writeJSON("environment/fingerprint.json", doc)
}

func pgPerfTierFingerprint() []map[string]any {
	out := make([]map[string]any, 0, 3)
	for _, tier := range []pgPerfTier{pgPerfTierSmall, pgPerfTierMedium, pgPerfTierLarge} {
		out = append(out, map[string]any{
			"name": tier.name, "context_budget": tier.contextBudget.String(),
			"seed_budget": tier.seedBudget.String(), "max_conns": tier.maxConns,
			"articles": tier.articles, "users": tier.users,
			"disk_limit_bytes": tier.diskLimit,
		})
	}
	return out
}

func pgPerfLeadDiskOverride() map[string]string {
	return map[string]string{
		"small": "1.0 GB", "medium": "5.5 GB", "large": "8.5 GB", "total": "15 GB",
		"authorized_by": "Lead", "authorized_at": "2026-11-04",
		"reason": "本轮用户允许采用建议值，Lead SOURCE/LOCAL/PG START 明确采用；原r04为small0.5/total14，新值待本轮独立Review",
	}
}

func pgPerfPoolConfig(t testing.TB, schema, appName string, maxConns int32, statementTimeout, lockTimeout time.Duration) *pgxpool.Config {
	t.Helper()
	dsn := os.Getenv(pgPerfDSNEnv)
	if strings.TrimSpace(dsn) == "" {
		t.Fatal("HAN_TEST_POSTGRES_DSN missing; no skip")
	}
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse pgperf pool configuration: %T (value withheld)", err)
	}
	config.ConnConfig.Database = pgPerfDatabaseName(t)
	ip := net.ParseIP(config.ConnConfig.Host)
	if ip == nil || !ip.IsLoopback() || config.ConnConfig.Port != 5433 {
		t.Fatal("test DSN must target the authorized loopback:5433")
	}
	config.MaxConns = maxConns
	if config.ConnConfig.RuntimeParams == nil {
		config.ConnConfig.RuntimeParams = map[string]string{}
	}
	config.ConnConfig.RuntimeParams["application_name"] = appName
	config.ConnConfig.RuntimeParams["search_path"] = pgx.Identifier{schema}.Sanitize()
	config.ConnConfig.RuntimeParams["statement_timeout"] = strconv.FormatInt(statementTimeout.Milliseconds(), 10)
	config.ConnConfig.RuntimeParams["lock_timeout"] = strconv.FormatInt(lockTimeout.Milliseconds(), 10)
	return config
}

func pgPerfMigrate(t testing.TB, ctx context.Context, pool *pgxpool.Pool, schema string, evidence *pgPerfEvidence, tier, caseCode string, failAt int) []pgPerfMigrationRecord {
	t.Helper()
	if err := pgPerfOperationAdmit(ctx); err != nil {
		t.Fatalf("migration admission closed before schema: %v", err)
	}
	if _, err := pool.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		t.Fatalf("create pgperf schema failed: %T", err)
	}
	evidence.addSchema(pgPerfSchemaRecord{Schema: schema, Tier: tier, Case: caseCode,
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), Retained: true, Length: len(schema)})
	t.Logf("created_schema=%s retained=true migration_pending=true", schema)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin pgperf migration: %T", err)
	}
	committed := false
	failedAt := 0
	defer func() {
		evidence.writeJSON(filepath.Join("migrations", schema+".status.json"), map[string]any{
			"schema": schema, "created": true, "committed": committed, "failed_at": failedAt,
		})
		if !committed {
			_ = tx.Rollback(context.Background())
		}
	}()
	if _, err := tx.Exec(ctx, "SET LOCAL statement_timeout=0"); err != nil {
		t.Fatalf("disable seed statement timeout: %T", err)
	}
	records := make([]pgPerfMigrationRecord, 0, len(pgPerfMigrationFiles))
	for index, name := range pgPerfMigrationFiles {
		if err := pgPerfOperationAdmit(ctx); err != nil {
			t.Fatalf("migration batch admission closed: %v", err)
		}
		failedAt = index + 1
		if failAt == index+1 {
			t.Fatalf("synthetic migration failure at %d; retained schema recorded", failAt)
		}
		body, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", name))
		if err != nil {
			t.Fatalf("read migration %s: %v", name, err)
		}
		sum := sha256.Sum256(body)
		records = append(records, pgPerfMigrationRecord{Order: index + 1, File: name, SHA256: hex.EncodeToString(sum[:])})
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			t.Fatalf("apply migration %s failed: %T", name, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit pgperf migrations: %T", err)
	}
	committed = true
	failedAt = 0
	evidence.addMigrations(schema, records)
	return records
}

type pgPerfIDs struct {
	prefix string
	next   atomic.Int64
}

func (ids *pgPerfIDs) NewID() (string, error) {
	return pgPerfUUID(ids.prefix, strconv.FormatInt(ids.next.Add(1), 10)), nil
}

type pgPerfClock struct{}

func (pgPerfClock) Now() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }

type pgPerfTransactor struct {
	pool  *pgxpool.Pool
	owner *pgPerfHarness
}

func (transactor pgPerfTransactor) Begin(ctx context.Context) (pgx.Tx, error) {
	if transactor.owner != nil {
		return transactor.owner.beginOwned(ctx)
	}
	return transactor.pool.Begin(ctx)
}

func (transactor pgPerfTransactor) BeginTx(ctx context.Context, options pgx.TxOptions) (pgx.Tx, error) {
	if transactor.owner != nil {
		return transactor.owner.beginOwnedOptions(ctx, options)
	}
	return transactor.pool.BeginTx(ctx, options)
}

type pgPerfOperationKey struct{}
type pgPerfRacePIDKey struct{}
type pgPerfOperationState struct {
	mu                           sync.Mutex
	begun, rolledBack, committed int
	unknown                      bool
}
type pgPerfOwnedTx struct {
	pgx.Tx
	owner *pgPerfHarness
	state *pgPerfOperationState
	ended atomic.Bool
}

func (tx *pgPerfOwnedTx) Commit(ctx context.Context) error {
	if !tx.ended.CompareAndSwap(false, true) {
		return pgx.ErrTxClosed
	}
	err := tx.Tx.Commit(ctx)
	if tx.state != nil {
		tx.state.mu.Lock()
		switch {
		case err == nil:
			tx.state.committed++
		case errors.Is(err, pgx.ErrTxCommitRollback):
			tx.state.rolledBack++
		default:
			tx.state.unknown = true
		}
		tx.state.mu.Unlock()
	}
	tx.owner.ownedMu.Lock()
	delete(tx.owner.ownedTx, tx)
	tx.owner.ownedMu.Unlock()
	return err
}

func (tx *pgPerfOwnedTx) Rollback(ctx context.Context) error {
	if !tx.ended.CompareAndSwap(false, true) {
		return pgx.ErrTxClosed
	}
	err := tx.Tx.Rollback(ctx)
	if tx.state != nil {
		tx.state.mu.Lock()
		if err == nil {
			tx.state.rolledBack++
		} else {
			tx.state.unknown = true
		}
		tx.state.mu.Unlock()
	}
	tx.owner.ownedMu.Lock()
	delete(tx.owner.ownedTx, tx)
	tx.owner.ownedMu.Unlock()
	return err
}

func (h *pgPerfHarness) beginOwned(ctx context.Context) (pgx.Tx, error) {
	return h.beginOwnedOptions(ctx, pgx.TxOptions{})
}

func (h *pgPerfHarness) beginOwnedOptions(ctx context.Context, options pgx.TxOptions) (pgx.Tx, error) {
	if err := pgPerfOperationAdmit(ctx); err != nil {
		return nil, err
	}
	tx, err := h.pool.BeginTx(ctx, options)
	if err != nil {
		return nil, err
	}
	owned := &pgPerfOwnedTx{Tx: tx, owner: h}
	if state, ok := ctx.Value(pgPerfOperationKey{}).(*pgPerfOperationState); ok {
		owned.state = state
		state.mu.Lock()
		state.begun++
		state.mu.Unlock()
	}
	h.ownedMu.Lock()
	if h.ownedTx == nil {
		h.ownedTx = map[*pgPerfOwnedTx]bool{}
	}
	h.ownedTx[owned] = true
	h.ownedMu.Unlock()
	if pids, ok := ctx.Value(pgPerfRacePIDKey{}).(chan uint32); ok {
		select {
		case pids <- tx.Conn().PgConn().PID():
		default:
		}
	}
	return owned, nil
}

func (e *pgPerfEvidence) hasSchema(schema string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, record := range e.schemas {
		if record.Schema == schema {
			return true
		}
	}
	return false
}

func (h *pgPerfHarness) closeResources(goroutineBase int) {
	h.cancel()
	if h.diskCancel != nil {
		h.diskCancel()
	}
	done := make(chan struct{})
	go func() { h.group.Wait(); close(done) }()
	closureStarted := time.Now()
	closureEnd := pgPerfClosureEnd(h.policy, closureStarted)
	timer := time.NewTimer(time.Until(closureEnd))
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
		select {
		case <-done:
		default:
			h.resourceFailStop("owned workers did not join within existing closure deadline")
		}
	}
	if h.resource != nil {
		h.resource.mu.Lock()
		failure, failed, last := h.resource.terminal, h.resource.failedOwn, h.resource.last
		h.resource.mu.Unlock()
		h.evidence.writeJSON(filepath.Join("resources", h.schema+".resource-close.json"), map[string]any{"terminal": fmt.Sprint(failure), "resource_failure": failed, "last_complete_success": last, "root_canceled": h.ctx.Err() != nil, "workers_joined": true, "actual_utc": time.Now().UTC()})
		if failed {
			h.t.Errorf("harness resource failure (terminal): %v", failure)
		}
	}
	h.reclaimTransactions(closureEnd)
	if h.pool != nil {
		settleCtx, settleCancel := context.WithDeadline(context.Background(), pgPerfMinTime(time.Now().Add(3*time.Second), closureEnd))
		settleStarted := time.Now()
		settleErr := h.waitReturned(settleCtx)
		settleCancel()
		acquired := h.pool.Stat().AcquiredConns()
		h.evidence.writeJSON(filepath.Join("resources", h.schema+".pre-close.json"), map[string]any{"acquired": acquired, "failed": h.t.Failed(), "return_wait_ns": time.Since(settleStarted).Nanoseconds(), "return_error": pgPerfErrorClass(settleErr)})
		if acquired != 0 {
			h.t.Errorf("unreturned owned connection after transaction reclamation: %d; synchronous pool.Close not entered", acquired)
			h.ownedMu.Lock()
			for conn := range h.ownedConnections {
				_ = conn.PgConn().Conn().Close()
			}
			h.ownedMu.Unlock()
		} else {
			h.closeResourcePool("business", closureEnd, func() int32 { return h.pool.Stat().AcquiredConns() }, h.pool.Close)
		}
	}
	if h.diskPool != nil {
		h.closeResourcePool("disk", closureEnd, func() int32 { return h.diskPool.Stat().AcquiredConns() }, h.diskPool.Close)
	}
	residual := h.closeProbe(closureEnd)
	ctx, cancel := context.WithDeadline(context.Background(), closureEnd)
	defer cancel()
	if err := h.waitFinalClose(ctx, closureEnd, func(context.Context) (bool, error) { return runtime.NumGoroutine() <= goroutineBase, nil }); err != nil {
		h.t.Errorf("goroutines did not return to baseline: before=%d after=%d", goroutineBase, runtime.NumGoroutine())
	}
	h.evidence.writeJSON(filepath.Join("resources", h.schema+".json"), map[string]any{
		"schema": h.schema, "residual_backends": residual, "goroutines_before": goroutineBase,
		"goroutines_after": runtime.NumGoroutine(), "failed": h.t.Failed(), "closure_started_utc": closureStarted.UTC(), "closure_deadline_utc": closureEnd.UTC(), "closure_elapsed_ns": time.Since(closureStarted).Nanoseconds(),
	})
	if h.t.Failed() {
		h.evidence.stopReason = h.t.Name()
	}
}

func (h *pgPerfHarness) waitFinalClose(ctx context.Context, closureEnd time.Time, check func(context.Context) (bool, error)) error {
	expired := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !time.Now().Before(closureEnd) {
			return context.DeadlineExceeded
		}
		return nil
	}
	err := h.pollUntil(ctx, 4*time.Second, func(ctx context.Context) (bool, error) {
		if err := expired(); err != nil {
			return false, err
		}
		ok, err := check(ctx)
		if err != nil {
			return false, err
		}
		if err := expired(); err != nil {
			return false, err
		}
		return ok, nil
	})
	if err != nil {
		return err
	}
	return expired()
}

func (h *pgPerfHarness) resourceFailStop(reason string) {
	if h.evidence != nil {
		h.evidence.checkpointInventory()
		h.evidence.writeJSONExclusive(filepath.Join("failures", fmt.Sprintf("resource.%d.fail-stop.json", time.Now().UnixNano())), map[string]any{"pid": os.Getpid(), "schema": h.schema, "reason": reason, "unknown": true, "exit_code": 86, "actual_utc": time.Now().UTC(), "root_canceled": h.ctx.Err() != nil})
	}
	fmt.Fprintln(os.Stderr, "PGPERF resource owned test child fail-stop exit=86:", reason)
	os.Exit(86)
}

func (h *pgPerfHarness) waitReturned(ctx context.Context) error {
	return h.pollUntil(ctx, 3*time.Second, func(context.Context) (bool, error) {
		h.ownedMu.Lock()
		remaining := len(h.ownedTx)
		h.ownedMu.Unlock()
		return remaining == 0 && h.pool.Stat().AcquiredConns() == 0, nil
	})
}

func (h *pgPerfHarness) reclaimTransactions(closureEnd time.Time) {
	h.ownedMu.Lock()
	txs := make([]*pgPerfOwnedTx, 0, len(h.ownedTx))
	for tx := range h.ownedTx {
		txs = append(txs, tx)
	}
	h.ownedMu.Unlock()
	if len(txs) != 0 {
		h.t.Errorf("owned transactions leaked before cleanup: %d", len(txs))
	}
	ctx, cancel := context.WithDeadline(context.Background(), pgPerfMinTime(time.Now().Add(3*time.Second), closureEnd))
	defer cancel()
	for _, tx := range txs {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			h.t.Errorf("reclaim leaked transaction: %T", err)
		}
	}
}

func (h *pgPerfHarness) awaitResourcePool(label string, deadline time.Time, acquired func() int32) {
	started := time.Now()
	before := acquired()
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	err := h.pollUntil(ctx, time.Until(deadline), func(context.Context) (bool, error) {
		if !time.Now().Before(deadline) {
			return false, context.DeadlineExceeded
		}
		return acquired() == 0, nil
	})
	after := acquired()
	if h.evidence != nil {
		h.evidence.writeJSON(filepath.Join("resources", h.schema+"."+label+"-return.json"), map[string]any{"acquired_before": before, "acquired_after": after, "started_utc": started.UTC(), "finished_utc": time.Now().UTC(), "deadline_utc": deadline.UTC(), "elapsed_ns": time.Since(started).Nanoseconds(), "error_class": pgPerfErrorClass(err), "returned": err == nil})
	}
	if err != nil {
		h.resourceFailStop(label + " did not return within original closure deadline")
	}
}

func pgPerfMinTime(first, second time.Time) time.Time {
	if first.Before(second) {
		return first
	}
	return second
}

func pgPerfClosureEnd(policy pgPerfExecutionPolicy, started time.Time) time.Time {
	end := started.Add(5 * time.Second)
	if policy.HardStopSeconds > 0 {
		end = pgPerfMinTime(end, policy.ExecutionStart.Add(time.Duration(policy.HardStopSeconds)*time.Second))
	}
	return end
}

func (h *pgPerfHarness) closeResourcePool(label string, deadline time.Time, acquired func() int32, closePool func()) {
	h.awaitResourcePool(label, deadline, acquired)
	started := time.Now()
	if !started.Before(deadline) {
		h.resourceFailStop(label + " Close cannot start past original closure deadline")
	}
	done := make(chan struct{})
	h.group.Add(1)
	var completedAt time.Time
	go func() { defer func() { h.group.Done(); close(done) }(); closePool(); completedAt = time.Now() }()
	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()
	var finished time.Time
	select {
	case <-done:
		finished = completedAt
	case <-timer.C:
		select {
		case <-done:
			finished = completedAt
		default:
		}
	}
	valid := !finished.IsZero() && finished.Before(deadline)
	if h.evidence != nil {
		h.evidence.writeJSON(filepath.Join("resources", h.schema+"."+label+"-close.json"), map[string]any{"started_utc": started.UTC(), "finished_utc": finished.UTC(), "observed_utc": time.Now().UTC(), "deadline_utc": deadline.UTC(), "elapsed_ns": time.Since(started).Nanoseconds(), "close_completed": !finished.IsZero(), "completed_within_deadline": valid, "acquired_after": acquired()})
	}
	if !valid {
		h.resourceFailStop(label + " Close did not complete within original closure deadline")
	}
}

func (h *pgPerfHarness) closeProbe(closureEnd time.Time) int {
	if h.probe == nil {
		return -1
	}
	h.awaitResourcePool("probe", closureEnd, func() int32 { return h.probe.Stat().AcquiredConns() })
	defer h.closeResourcePool("probe-close", closureEnd, func() int32 { return h.probe.Stat().AcquiredConns() }, h.probe.Close)
	ctx, cancel := context.WithDeadline(context.Background(), closureEnd)
	defer cancel()
	residual := -1
	err := h.pollUntil(ctx, 4*time.Second, func(ctx context.Context) (bool, error) {
		err := h.probe.QueryRow(ctx, "SELECT count(*) FROM pg_stat_activity WHERE application_name=$1 AND pid<>pg_backend_pid()", h.appName).Scan(&residual)
		return residual == 0, err
	})
	if err != nil {
		h.t.Errorf("residual backend check failed: residual=%d err=%T", residual, err)
	}
	return residual
}

func (h *pgPerfHarness) ownedPoolConfig(maxConns int32, statementTimeout, lockTimeout time.Duration) *pgxpool.Config {
	config := pgPerfPoolConfig(h.t, h.schema, h.appName, maxConns, statementTimeout, lockTimeout)
	config.BeforeAcquire = func(ctx context.Context, _ *pgx.Conn) bool {
		if pgPerfOperationAdmit(ctx) != nil {
			return false
		}
		return h.resource == nil || h.resource.admit(ctx) == nil
	}
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		h.ownedMu.Lock()
		defer h.ownedMu.Unlock()
		if h.ownedConnections == nil {
			h.ownedConnections = map[*pgx.Conn]bool{}
		}
		h.ownedConnections[conn] = true
		return nil
	}
	return config
}

func pgPerfOperationAdmit(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if deadline, ok := ctx.Deadline(); ok && !time.Now().Before(deadline) {
		return context.DeadlineExceeded
	}
	if resource, ok := ctx.Value(pgPerfResourceKey{}).(*pgPerfResourceState); ok {
		return resource.admit(ctx)
	}
	return nil
}

type pgPerfGate struct{ pool *pgxpool.Pool }

func pgPerfGateConfig() platform.ProcessingGateConfig {
	return platform.ProcessingGateConfig{
		RegionRef:        "synthetic-region-open",
		ControllerRef:    "synthetic-controller",
		LegalBasisRef:    "synthetic-basis",
		SMTPProcessorRef: "synthetic-processor",
	}
}

func (gate pgPerfGate) RequireOpen(ctx context.Context) error {
	store, err := privacy.NewStore(gate.pool)
	if err != nil {
		return err
	}
	policy, err := privacy.NewProcessingGate(pgPerfGateConfig(), store)
	if err != nil {
		return err
	}
	return policy.RequireOpen(ctx)
}

func (gate pgPerfGate) RequireOpenInTx(ctx context.Context, tx pgx.Tx) error {
	store, err := privacy.NewStore(tx)
	if err != nil {
		return err
	}
	policy, err := privacy.NewProcessingGate(pgPerfGateConfig(), store)
	if err != nil {
		return err
	}
	return policy.RequireOpen(ctx)
}

type pgPerfOptions struct {
	directedCase       string
	testDiskLimitDelta int64
	diskProbe          func(string) (int64, string)
	failMigration      int
	beforeSeed         func(*pgPerfHarness)
	maxConns           int32
	statementTimeout   time.Duration
	lockTimeout        time.Duration
	seedMode           string
	seedConfig         *pgPerfSeedConfig
	skipSeed           bool
	probe              bool
}

type pgPerfHarness struct {
	directedCase      string
	caseBudget        pgPerfCaseBudget
	raceEvidence      bool
	seedEvents        atomic.Uint64
	seedCommitted     bool
	policy            pgPerfExecutionPolicy
	diskProbe         func(string) (int64, string)
	diskPool          *pgxpool.Pool
	diskCtx           context.Context
	diskCancel        context.CancelFunc
	resource          *pgPerfResourceState
	ownedMu           sync.Mutex
	ownedTx           map[*pgPerfOwnedTx]bool
	ownedConnections  map[*pgx.Conn]bool
	t                 testing.TB
	ctx               context.Context
	cancel            context.CancelFunc
	tier              pgPerfTier
	caseCode          string
	schema            string
	appName           string
	pool              *pgxpool.Pool
	probe             *pgxpool.Pool
	evidence          *pgPerfEvidence
	statementTimeout  time.Duration
	lockTimeout       time.Duration
	seed              pgPerfSeedManifest
	group             sync.WaitGroup
	baselineDBBytes   int64
	resourceBaseBytes int64
	resourceTierBytes int64
	actualBaseline    pgPerfLoadedBaseline
	ids               *pgPerfIDs
	pgdataPath        string
}

func pgPerfWindowDeadline(t testing.TB) time.Time {
	t.Helper()
	policy := pgPerfPolicy(t)
	return policy.ExecutionStart.Add(time.Duration(policy.StopNewSeconds) * time.Second)
}

type pgPerfExecutionPolicy struct {
	Revision          string                           `json:"revision"`
	AttemptID         string                           `json:"attempt_id"`
	ParentOldBatch    string                           `json:"parent_old_batch"`
	AdoptionSource    string                           `json:"adoption_source"`
	AdoptionSHA256    string                           `json:"adoption_sha256"`
	WindowStart       string                           `json:"window_start"`
	StopNewSeconds    int64                            `json:"stop_new_seconds"`
	HardStopSeconds   int64                            `json:"hard_stop_seconds"`
	TimedWallSeconds  int64                            `json:"timed_wall_seconds"`
	SingleWallSeconds int64                            `json:"single_wall_seconds"`
	PointTailSeconds  int64                            `json:"point_tail_seconds"`
	ResourceWindow    json.RawMessage                  `json:"resource_window_start,omitempty"`
	ResourceBaseBytes int64                            `json:"resource_base_bytes,omitempty"`
	ResourceDirectory string                           `json:"resource_baseline_dir,omitempty"`
	ResourceBaselines map[string]pgPerfBaselineBinding `json:"resource_baselines,omitempty"`
	ExecutionStart    time.Time                        `json:"-"`
	Resource          pgPerfResourceBinding            `json:"-"`
	ProbeTimeout      json.RawMessage                  `json:"resource_probe_timeout_ms,omitempty"`
	MonitorTick       json.RawMessage                  `json:"resource_monitor_tick_ms,omitempty"`
	FreshnessLimit    json.RawMessage                  `json:"resource_freshness_limit_ms,omitempty"`
	DetectionEnvelope json.RawMessage                  `json:"resource_detection_envelope_ms,omitempty"`
	ProbeAdoption     string                           `json:"resource_probe_adoption_source,omitempty"`
	ProbeAdoptionSHA  string                           `json:"resource_probe_adoption_sha256,omitempty"`
	ProbeBudget       pgPerfProbeBudget                `json:"-"`
}

const pgPerfResourceOrigin = "2026-11-04T03:29:04.586115+00:00"
const pgPerfResourceBase int64 = 1406932671
const pgPerfLegacyRevision = "execution-policy-r02-adopted-20261104"

type pgPerfBaselineBinding struct {
	File          string `json:"file"`
	State         string `json:"state"`
	Window        string `json:"window"`
	Tier          string `json:"tier"`
	Bytes         int64  `json:"bytes"`
	FileBytes     int64  `json:"file_bytes"`
	SHA256        string `json:"sha256"`
	AllowFirst    bool   `json:"allow_first_registration"`
	ReceiptSHA256 string `json:"receipt_sha256,omitempty"`
}

type pgPerfResourceBinding struct {
	Window          string
	BaseBytes       int64
	Directory       string
	Baselines       map[string]pgPerfBaselineBinding
	ManifestSHA256  string
	AttemptID       string
	ExecutionWindow string
	Legacy          bool
}

type pgPerfPolicyInputs struct {
	Now                  time.Time
	ExecutionWindow      string
	BaseBytes            string
	BaselineDirectory    string
	BaseBytesSet         bool
	BaselineDirectorySet bool
	ExpectedDirectory    string
	GlobalBaseFile       string
	AdoptionFile         string
	ProbeAdoptionFile    string
}

func pgPerfPolicyPaths() (string, string, string) {
	_, source, _, ok := runtime.Caller(0)
	if !ok || !filepath.IsAbs(source) {
		return "", "", ""
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..", ".."))
	author := filepath.Join(root, ".spec", "wip", "article-management", "evidence-pg-perf-r04", "continuation-20261104", "author")
	return filepath.Join(author, "r02", "baselines"), filepath.Join(author, "pg-base-r01.json"), filepath.Join(author, "..", "lead", "followup-clock-contract-adoption-r04.md")
}

func pgPerfParsePolicy(data []byte, input pgPerfPolicyInputs) (pgPerfExecutionPolicy, error) {
	var policy pgPerfExecutionPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return policy, err
	}
	start, err := time.Parse(time.RFC3339Nano, policy.WindowStart)
	execution, envErr := time.Parse(time.RFC3339Nano, input.ExecutionWindow)
	if err != nil || envErr != nil || !start.Equal(execution) || start.After(input.Now) || !input.Now.Before(start.Add(8700*time.Second)) {
		return policy, errors.New("invalid/future/expired execution window")
	}
	if policy.StopNewSeconds != 8700 || policy.HardStopSeconds != 9000 || policy.TimedWallSeconds != 121 || policy.SingleWallSeconds != 60 || policy.PointTailSeconds != 1 {
		return policy, errors.New("execution limits differ from adopted contract")
	}
	policy.ExecutionStart = start
	legacy := len(policy.ResourceWindow) == 0 && policy.Revision == pgPerfLegacyRevision && policy.AttemptID == "" && policy.ParentOldBatch == "" && policy.AdoptionSHA256 == "" && !strings.Contains(policy.AdoptionSource, "followup")
	if legacy {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(data, &fields); err != nil {
			return policy, err
		}
		for _, key := range []string{"resource_base_bytes", "resource_baseline_dir", "resource_baselines"} {
			if _, exists := fields[key]; exists {
				return policy, errors.New("partial resource fields cannot claim legacy")
			}
		}
		origin, _ := time.Parse(time.RFC3339Nano, pgPerfResourceOrigin)
		if !start.Equal(origin) || policy.ResourceBaseBytes != 0 || policy.ResourceDirectory != "" || policy.ResourceBaselines != nil {
			return policy, errors.New("new/partial resource policy cannot claim legacy")
		}
		policy.Resource, err = pgPerfLegacyResource(input)
	} else {
		policy.Resource, err = pgPerfExplicitResource(policy, input)
	}
	if err != nil {
		return policy, err
	}
	if err = pgPerfValidateResource(policy.Resource, input.ExpectedDirectory); err != nil {
		return policy, err
	}
	if err = pgPerfValidateResourceEnv(policy.Resource, input); err != nil {
		return policy, err
	}
	policy.ProbeBudget, err = pgPerfParseProbeBudget(policy, input.ProbeAdoptionFile)
	if err != nil {
		return policy, err
	}
	return policy, nil
}

type pgPerfProbeBudget struct{ Timeout, Tick, Freshness time.Duration }

func pgPerfParseProbeBudget(policy pgPerfExecutionPolicy, adoption string) (pgPerfProbeBudget, error) {
	legacy := pgPerfProbeBudget{Timeout: 5 * time.Second, Tick: 5 * time.Second}
	if len(policy.DetectionEnvelope) != 0 {
		return legacy, errors.New("retired resource detection-envelope claim is not supported")
	}
	if len(policy.ProbeTimeout) == 0 && len(policy.MonitorTick) == 0 && len(policy.FreshnessLimit) == 0 && policy.ProbeAdoption == "" && policy.ProbeAdoptionSHA == "" {
		return legacy, nil
	}
	for _, pair := range []struct {
		raw  json.RawMessage
		want int64
	}{{policy.ProbeTimeout, 15000}, {policy.MonitorTick, 5000}, {policy.FreshnessLimit, 20000}} {
		var value int64
		if len(pair.raw) == 0 || string(pair.raw) == "null" || json.Unmarshal(pair.raw, &value) != nil || value != pair.want {
			return legacy, errors.New("complete adopted resource budget 15000/5000/20000 required")
		}
	}
	if err := pgPerfVerifyAdoption(pgPerfExecutionPolicy{AdoptionSource: policy.ProbeAdoption, AdoptionSHA256: policy.ProbeAdoptionSHA}, adoption); err != nil {
		return legacy, err
	}
	return pgPerfProbeBudget{Timeout: 15 * time.Second, Tick: 5 * time.Second, Freshness: 20 * time.Second}, nil
}

func pgPerfExplicitResource(policy pgPerfExecutionPolicy, input pgPerfPolicyInputs) (pgPerfResourceBinding, error) {
	var window string
	if len(policy.ResourceWindow) == 0 || string(policy.ResourceWindow) == "null" || json.Unmarshal(policy.ResourceWindow, &window) != nil || window != pgPerfResourceOrigin {
		return pgPerfResourceBinding{}, errors.New("explicit original resource window required")
	}
	if policy.AttemptID == "" || policy.ParentOldBatch != pgPerfLegacyRevision || policy.Revision == "" || policy.ResourceBaseBytes != pgPerfResourceBase {
		return pgPerfResourceBinding{}, errors.New("new attempt/resource identity incomplete")
	}
	if err := pgPerfVerifyAdoption(policy, input.AdoptionFile); err != nil {
		return pgPerfResourceBinding{}, err
	}
	manifest, err := json.Marshal(policy.ResourceBaselines)
	if err != nil {
		return pgPerfResourceBinding{}, err
	}
	sum := sha256.Sum256(manifest)
	return pgPerfResourceBinding{Window: window, BaseBytes: policy.ResourceBaseBytes, Directory: policy.ResourceDirectory, Baselines: policy.ResourceBaselines, ManifestSHA256: hex.EncodeToString(sum[:]), AttemptID: policy.AttemptID, ExecutionWindow: policy.WindowStart}, nil
}

func pgPerfVerifyAdoption(policy pgPerfExecutionPolicy, expected string) error {
	if policy.AdoptionSource == "" || policy.AdoptionSHA256 == "" {
		return errors.New("new policy has no adoption binding")
	}
	if err := pgPerfSamePath(policy.AdoptionSource, expected, false); err != nil {
		return err
	}
	data, err := os.ReadFile(expected)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	if !strings.EqualFold(policy.AdoptionSHA256, hex.EncodeToString(sum[:])) {
		return errors.New("adoption hash mismatch")
	}
	return nil
}

func pgPerfLegacyResource(input pgPerfPolicyInputs) (pgPerfResourceBinding, error) {
	resource := pgPerfResourceBinding{Window: input.ExecutionWindow, BaseBytes: pgPerfResourceBase, Directory: input.ExpectedDirectory, Baselines: map[string]pgPerfBaselineBinding{}, Legacy: true, AttemptID: pgPerfLegacyRevision, ExecutionWindow: input.ExecutionWindow}
	for _, tier := range []string{"small", "medium", "large"} {
		binding := pgPerfBaselineBinding{File: tier + ".json", State: "unregistered", Tier: tier, Window: resource.Window, AllowFirst: tier == "medium"}
		data, err := os.ReadFile(filepath.Join(resource.Directory, binding.File))
		if err == nil {
			var prior pgPerfBaselineRecord
			if err = json.Unmarshal(data, &prior); err != nil {
				return resource, err
			}
			sum := sha256.Sum256(data)
			binding.State = "existing"
			binding.AllowFirst = false
			binding.Bytes = prior.Bytes
			binding.FileBytes = int64(len(data))
			binding.SHA256 = hex.EncodeToString(sum[:])
			receipt, readErr := os.ReadFile(filepath.Join(resource.Directory, tier+".registration.json"))
			if readErr == nil {
				sum = sha256.Sum256(receipt)
				binding.ReceiptSHA256 = hex.EncodeToString(sum[:])
			} else if !errors.Is(readErr, os.ErrNotExist) {
				return resource, readErr
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return resource, err
		}
		resource.Baselines[tier] = binding
	}
	manifest, _ := json.Marshal(resource.Baselines)
	sum := sha256.Sum256(manifest)
	resource.ManifestSHA256 = hex.EncodeToString(sum[:])
	return resource, nil
}

func pgPerfValidateResourceEnv(resource pgPerfResourceBinding, input pgPerfPolicyInputs) error {
	execution, err := time.Parse(time.RFC3339Nano, input.ExecutionWindow)
	bound, boundErr := time.Parse(time.RFC3339Nano, resource.ExecutionWindow)
	if err != nil || boundErr != nil || !execution.Equal(bound) {
		return errors.New("child/execution environment identity mismatch")
	}
	if (input.BaseBytesSet || input.BaseBytes != "") && input.BaseBytes != strconv.FormatInt(resource.BaseBytes, 10) {
		return errors.New("resource base environment cannot override policy")
	}
	if input.BaselineDirectorySet || input.BaselineDirectory != "" {
		if err := pgPerfSamePath(input.BaselineDirectory, resource.Directory, true); err != nil {
			return err
		}
	}
	if err := pgPerfSamePath(input.GlobalBaseFile, input.GlobalBaseFile, false); err != nil {
		return err
	}
	data, err := os.ReadFile(input.GlobalBaseFile)
	if err != nil {
		return err
	}
	var global struct {
		Bytes int64 `json:"database_size_base"`
	}
	if err = json.Unmarshal(data, &global); err != nil {
		return err
	}
	if global.Bytes != resource.BaseBytes {
		return errors.New("global resource source mismatch")
	}
	return nil
}

func pgPerfSamePath(path, expected string, directory bool) error {
	if !filepath.IsAbs(path) || !filepath.IsAbs(expected) {
		return errors.New("resource/adoption path must be absolute")
	}
	for current := filepath.Clean(path); ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
			return errors.New("resource path contains reparse/symlink")
		}
		if attrs, ok := info.Sys().(*syscall.Win32FileAttributeData); ok && attrs.FileAttributes&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
			return errors.New("resource path contains reparse point")
		}
		if filepath.Dir(current) == current {
			break
		}
	}
	actual, err := os.Stat(path)
	if err != nil {
		return err
	}
	want, err := os.Stat(expected)
	if err != nil {
		return err
	}
	if actual.IsDir() != directory || !os.SameFile(actual, want) {
		return errors.New("resource/adoption path identity mismatch")
	}
	return nil
}

func pgPerfPolicy(t testing.TB) pgPerfExecutionPolicy {
	t.Helper()
	data, err := os.ReadFile(os.Getenv("HAN_PG_PERF_POLICY_FILE"))
	if err != nil {
		t.Fatal("explicit adopted execution policy required")
	}
	directory, global, adoption := pgPerfPolicyPaths()
	base, baseSet := os.LookupEnv(pgPerfBaseBytesEnv)
	baselineDir, dirSet := os.LookupEnv(pgPerfBaselineDirEnv)
	policy, err := pgPerfParsePolicy(data, pgPerfPolicyInputs{Now: time.Now(), ExecutionWindow: os.Getenv(pgPerfWindowStartEnv), BaseBytes: base, BaselineDirectory: baselineDir, BaseBytesSet: baseSet, BaselineDirectorySet: dirSet, ExpectedDirectory: directory, GlobalBaseFile: global, AdoptionFile: adoption, ProbeAdoptionFile: filepath.Join(filepath.Dir(adoption), "probe-budget-contract-adoption-r02.md")})
	if err != nil {
		t.Fatalf("invalid adopted execution/resource policy: %v", err)
	}
	return policy
}

type pgPerfResourceObservation struct {
	Requested, Started, Finished       time.Time
	ReadStarted, Deadline              time.Time
	Bytes, Free                        int64
	AcquireNS, SQLNS, WaitNS, VolumeNS int64
	ReadErrorClass                     string
	Reused                             bool
	Returned, SourceRequested          time.Time
	SourceSQLNS                        int64
	FlightID                           uint64
}

type pgPerfResourceState struct {
	mu         sync.Mutex
	flight     chan struct{}
	changed    chan struct{}
	owner      context.Context
	cancel     context.CancelFunc
	budget     pgPerfProbeBudget
	last       pgPerfResourceObservation
	terminal   error
	failedOwn  bool
	now        func() time.Time
	successful uint64
	receipts   atomic.Uint64
}

type pgPerfResourceKey struct{}

func newPgPerfResourceState(owner context.Context, cancel context.CancelFunc, budget pgPerfProbeBudget) *pgPerfResourceState {
	return &pgPerfResourceState{flight: make(chan struct{}, 1), changed: make(chan struct{}, 1), owner: owner, cancel: cancel, budget: budget, now: time.Now}
}

func (s *pgPerfResourceState) stopLocked(err error) {
	if s.terminal == nil {
		s.failedOwn = s.owner.Err() == nil || !pgPerfContextOnly(err)
		s.terminal = err
		s.cancel()
		select {
		case s.changed <- struct{}{}:
		default:
		}
	}
}

func (s *pgPerfResourceState) freshnessLocked() time.Time {
	if s.last.Started.IsZero() || s.budget.Freshness == 0 {
		return time.Time{}
	}
	return s.last.Started.Add(s.budget.Freshness)
}

func (s *pgPerfResourceState) watchRequest(ctx context.Context) func() {
	done := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { s.mu.Lock(); s.stopLocked(ctx.Err()); s.mu.Unlock(); close(done) })
	return func() {
		if !stop() {
			<-done
		}
	}
}

func (s *pgPerfResourceState) admit(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.owner.Err(); err != nil {
		s.stopLocked(err)
	}
	if end := s.freshnessLocked(); !end.IsZero() && !s.now().Before(end) {
		s.stopLocked(errors.New("resource freshness expired"))
	}
	if s.terminal != nil {
		return s.terminal
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if end, ok := ctx.Deadline(); ok && !s.now().Before(end) {
		return context.DeadlineExceeded
	}
	if s.last.Started.IsZero() {
		return errors.New("resource observation unknown; admission closed")
	}
	return nil
}

func (s *pgPerfResourceState) collect(parent context.Context, read func(context.Context) (pgPerfResourceObservation, error)) (pgPerfResourceObservation, error) {
	entered := s.now()
	s.mu.Lock()
	generation := s.successful
	end := entered.Add(s.budget.Timeout)
	if old := s.freshnessLocked(); !old.IsZero() && old.Before(end) {
		end = old
	}
	if bound, ok := s.owner.Deadline(); ok && bound.Before(end) {
		end = bound
	}
	if bound, ok := parent.Deadline(); ok && bound.Before(end) {
		end = bound
	}
	if s.terminal != nil {
		err := s.terminal
		s.mu.Unlock()
		return pgPerfResourceObservation{}, err
	}
	s.mu.Unlock()
	ctx, cancel := context.WithDeadline(parent, end)
	defer cancel()
	defer s.watchRequest(ctx)()
	ownerDone := make(chan struct{})
	stop := context.AfterFunc(s.owner, func() { cancel(); close(ownerDone) })
	defer func() {
		if !stop() {
			<-ownerDone
		}
	}()
	select {
	case s.flight <- struct{}{}:
		defer func() { <-s.flight }()
	case <-ctx.Done():
		s.mu.Lock()
		s.stopLocked(ctx.Err())
		err := s.terminal
		s.mu.Unlock()
		return pgPerfResourceObservation{Requested: entered, Started: entered, Finished: s.now(), Deadline: end}, err
	}
	s.mu.Lock()
	err := s.terminal
	if old := s.freshnessLocked(); !old.IsZero() && old.Before(end) {
		end = old
	}
	if err == nil {
		err = ctx.Err()
	}
	if err == nil {
		err = s.owner.Err()
	}
	returned := s.now()
	if err == nil && !returned.Before(end) {
		err = context.DeadlineExceeded
	}
	if err == nil && s.successful != generation && s.last.Finished.Before(end) {
		shared := s.last
		shared.Reused = true
		shared.SourceRequested = shared.Requested
		shared.SourceSQLNS = shared.SQLNS
		shared.Requested, shared.Returned, shared.Deadline = entered, returned, end
		shared.WaitNS = returned.Sub(entered).Nanoseconds()
		shared.AcquireNS, shared.SQLNS, shared.VolumeNS = 0, 0, 0
		s.mu.Unlock()
		return shared, nil
	}
	s.mu.Unlock()
	bounded, boundedCancel := context.WithDeadline(ctx, end)
	defer boundedCancel()
	ctx = bounded
	defer s.watchRequest(ctx)()
	var observation pgPerfResourceObservation
	var readStarted time.Time
	if err == nil {
		err = ctx.Err()
	}
	if err == nil && !s.now().Before(end) {
		err = context.DeadlineExceeded
	}
	if err == nil {
		readStarted = s.now()
		observation, err = read(ctx)
	}
	observation.Requested = entered
	observation.Started = entered
	observation.Finished = s.now()
	observation.Returned = observation.Finished
	observation.ReadStarted = readStarted
	observation.ReadErrorClass = pgPerfErrorClass(err)
	observation.Deadline = end
	if !readStarted.IsZero() {
		observation.WaitNS = readStarted.Sub(entered).Nanoseconds()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.terminal != nil {
		if err != nil && !pgPerfContextOnly(err) {
			s.failedOwn = true
			return observation, errors.Join(s.terminal, err)
		}
		return observation, s.terminal
	}
	if err == nil {
		err = ctx.Err()
	}
	if err == nil {
		err = s.owner.Err()
	}
	if err == nil && (!observation.Finished.Before(end) || !s.now().Before(end) || observation.Bytes <= 0 || observation.Free <= 0) {
		err = errors.New("late/incomplete resource observation")
	}
	if err != nil {
		s.stopLocked(err)
		return observation, err
	}
	s.successful++
	observation.FlightID = s.successful
	s.last = observation
	select {
	case s.changed <- struct{}{}:
	default:
	}
	return observation, nil
}

func (h *pgPerfHarness) checkDiskBudget() {
	if err := h.checkDiskBudgetContext(h.ctx); err != nil {
		h.t.Errorf("resource guard failed closed: %v", err)
	}
}

func (h *pgPerfHarness) checkDiskBudgetContext(parent context.Context) error {
	return h.collectDiskBudgetContext(parent, h.readDiskObservation)
}

func (h *pgPerfHarness) collectDiskBudgetContext(parent context.Context, read func(context.Context) (pgPerfResourceObservation, error)) error {
	if h.resource == nil {
		return errors.New("resource guard not initialized")
	}
	observation, err := h.resource.collect(parent, read)
	if h.evidence != nil {
		h.evidence.writeJSONExclusive(filepath.Join("diagnosis", fmt.Sprintf("disk.%s.%d.%d.json", h.schema, time.Now().UnixNano(), h.resource.receipts.Add(1))), map[string]any{"requested_utc": observation.Requested.UTC(), "started_utc": observation.Started.UTC(), "read_started_utc": observation.ReadStarted.UTC(), "finished_utc": observation.Finished.UTC(), "deadline_utc": observation.Deadline.UTC(), "returned_utc": observation.Returned.UTC(), "reused": observation.Reused, "flight_id": observation.FlightID, "source_requested_utc": observation.SourceRequested.UTC(), "source_sql_ns": observation.SourceSQLNS, "wait_ns": observation.WaitNS, "interval_ns": observation.Finished.Sub(observation.Started).Nanoseconds(), "acquire_ns": observation.AcquireNS, "sql_ns": observation.SQLNS, "database_bytes": observation.Bytes, "volume_free": observation.Free, "error": fmt.Sprint(err), "read_error_class": observation.ReadErrorClass, "accepted": err == nil, "owner": "harness root/diskCtx", "tier_base": h.resourceTierBytes, "global_base": h.resourceBaseBytes})
	}
	return err
}

func (h *pgPerfHarness) readDiskObservation(ctx context.Context) (pgPerfResourceObservation, error) {
	var result pgPerfResourceObservation
	pool := h.diskPool
	if pool == nil {
		pool = h.probe
	}
	started := time.Now()
	conn, err := pool.Acquire(ctx)
	result.AcquireNS = time.Since(started).Nanoseconds()
	if err != nil {
		return result, pgPerfResourceReadError(ctx, "acquire", err)
	}
	defer conn.Release()
	if err = ctx.Err(); err != nil {
		return result, err
	}
	started = time.Now()
	err = conn.QueryRow(ctx, "SELECT pg_database_size(current_database())").Scan(&result.Bytes)
	result.SQLNS = time.Since(started).Nanoseconds()
	if err != nil {
		return result, pgPerfResourceReadError(ctx, "SQL", err)
	}
	if h.pgdataPath == "" {
		return result, errors.New("PGDATA unknown before resource observation")
	}
	var status string
	started = time.Now()
	result.Free, status = h.diskProbe(h.pgdataPath)
	result.VolumeNS = time.Since(started).Nanoseconds()
	return result, h.validateResourceObservation(result, status)
}

type pgPerfSafeResourceError struct{ Stage, OriginalType, SQLState, ContextClass string }

func (err *pgPerfSafeResourceError) Error() string {
	return fmt.Sprintf("resource %s: class=non_context type=%s sqlstate=%s context=%s", err.Stage, err.OriginalType, err.SQLState, err.ContextClass)
}

func pgPerfResourceReadError(ctx context.Context, stage string, err error) error {
	if err == nil {
		return ctx.Err()
	}
	if pgPerfContextOnly(err) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return context.DeadlineExceeded
		}
		return context.Canceled
	}
	safe := &pgPerfSafeResourceError{Stage: stage, OriginalType: fmt.Sprintf("%T", err), ContextClass: pgPerfErrorClass(ctx.Err())}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		safe.SQLState = pgErr.Code
		if len(safe.SQLState) != 5 {
			safe.SQLState = "invalid"
		}
		for _, value := range safe.SQLState {
			if !(value >= 'A' && value <= 'Z' || value >= '0' && value <= '9') {
				safe.SQLState = "invalid"
				break
			}
		}
	}
	return safe
}

type pgPerfResourceCapError struct {
	Observed, TierBase, TierCap, GlobalBase, TotalCap int64
}

func (err *pgPerfResourceCapError) Error() string {
	return fmt.Sprintf("resource cap reached/exceeded: tier_delta=%d tier_cap=%d total_delta=%d total_cap=%d", err.Observed-err.TierBase, err.TierCap, err.Observed-err.GlobalBase, err.TotalCap)
}

func (h *pgPerfHarness) validateResourceObservation(result pgPerfResourceObservation, status string) error {
	if result.Bytes <= 0 {
		return errors.New("invalid database size observation")
	}
	if status != "ok" || result.Free <= 0 || result.Free < result.Bytes+h.tier.diskLimit {
		return fmt.Errorf("PGDATA reserve unavailable: bytes=%d free=%d status=%s", result.Bytes, result.Free, status)
	}
	if h.resourceBaseBytes > 0 {
		tierDelta := int64(0)
		if h.resourceTierBytes > 0 {
			tierDelta = result.Bytes - h.resourceTierBytes
		}
		totalDelta := result.Bytes - h.resourceBaseBytes
		if (h.resourceTierBytes > 0 && tierDelta >= h.tier.diskLimit) || totalDelta >= 15*1024*1024*1024 {
			return &pgPerfResourceCapError{Observed: result.Bytes, TierBase: h.resourceTierBytes, TierCap: h.tier.diskLimit, GlobalBase: h.resourceBaseBytes, TotalCap: 15 * 1024 * 1024 * 1024}
		}
	}
	return nil
}

type pgPerfCaseBudget struct {
	LogicalCase                                          string `json:"logical_case"`
	Profile                                              string `json:"profile"`
	DefaultSeedSeconds                                   int64  `json:"default_seed_seconds"`
	EffectiveSeedSeconds                                 int64  `json:"effective_seed_seconds"`
	RootSeconds                                          int64  `json:"root_seconds"`
	AdoptionRef                                          string `json:"adoption_ref,omitempty"`
	AdoptionSHA                                          string `json:"adoption_sha256,omitempty"`
	RootStarted, RootDeadline, SeedStarted, SeedDeadline time.Time
}

func pgPerfDirectedTier(tier pgPerfTier, code, directed string) (pgPerfTier, pgPerfCaseBudget, error) {
	budget := pgPerfCaseBudget{LogicalCase: directed, Profile: "tier-default", DefaultSeedSeconds: int64(tier.seedBudget / time.Second), EffectiveSeedSeconds: int64(tier.seedBudget / time.Second), RootSeconds: int64(tier.contextBudget / time.Second)}
	if directed == "initial-drain-10000" {
		if code != "r2n3b" || tier.seedBudget != 120*time.Second || tier.contextBudget != 300*time.Second || tier.name != pgPerfTierSmall.name {
			return tier, budget, errors.New("initial-drain-10000 requires the directed small profile")
		}
		tier.seedBudget = 240 * time.Second
		budget.EffectiveSeedSeconds = 240
		budget.Profile = "directed-normal-initial-drain-10000-seed240-root300"
		budget.AdoptionRef = "lead/semantics-seed-budget-adoption-r01.json"
		budget.AdoptionSHA = "43B7F52CC1200D584E1DF87FA072AC19863CEC0B26F9AB1917D1504EEFA5C8EB"
	}
	return tier, budget, nil
}

func TestPgPerfR10_CaseBudgetScope(t *testing.T) {
	got, profile, err := pgPerfDirectedTier(pgPerfTierSmall, "r2n3b", "initial-drain-10000")
	if err != nil || got.seedBudget != 240*time.Second || got.contextBudget != 300*time.Second || profile.EffectiveSeedSeconds != 240 || profile.DefaultSeedSeconds != 120 || profile.AdoptionSHA == "" {
		t.Fatalf("normal10k profile absent: seed=%s root=%s profile=%+v err=%v", got.seedBudget, got.contextBudget, profile, err)
	}
	for _, tier := range []pgPerfTier{pgPerfTierSmall, pgPerfTierMedium, pgPerfTierLarge} {
		got, _, err := pgPerfDirectedTier(tier, "formal", "")
		if err != nil || got.seedBudget != tier.seedBudget || got.contextBudget != tier.contextBudget {
			t.Fatal("formal tier changed")
		}
	}
	if _, _, err := pgPerfDirectedTier(pgPerfTierMedium, "r2n3b", "initial-drain-10000"); err == nil {
		t.Fatal("invalid directed tier accepted")
	}
}

func (h *pgPerfHarness) seedProgress(phase string, detail any) {
	if h.directedCase == "" {
		return
	}
	h.evidence.writeJSONExclusive(filepath.Join("seed-progress", fmt.Sprintf("%s.%06d.json", h.schema, h.seedEvents.Add(1))), map[string]any{"phase": phase, "detail": detail, "observed_at": time.Now(), "budget": h.caseBudget, "transaction_committed": h.seedCommitted, "run_id": h.evidence.runID, "schema": h.schema})
}

func pgPerfRaceSnapshot(ctx context.Context, id string, read func(context.Context, string) (json.RawMessage, error)) (json.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	data, err := read(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var snapshot struct {
		Article struct {
			ID string `json:"id"`
		} `json:"article"`
	}
	if err := json.Unmarshal(data, &snapshot); err != nil || snapshot.Article.ID != id {
		return nil, errors.New("race snapshot missing or mismatched article")
	}
	return data, nil
}

func TestPgPerfR10_RaceSnapshotBoundaries(t *testing.T) {
	good := json.RawMessage(`{"article":{"id":"same"},"operations":[],"audit":[]}`)
	fault := errors.New("synthetic snapshot read failure")
	for _, mode := range []string{"success", "read-error", "expired", "late", "wrong-id", "invalid-json"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "expired" {
				cancel()
			}
			calls := 0
			_, err := pgPerfRaceSnapshot(ctx, "same", func(context.Context, string) (json.RawMessage, error) {
				calls++
				switch mode {
				case "read-error":
					return nil, fault
				case "late":
					cancel()
				case "wrong-id":
					return json.RawMessage(`{"article":{"id":"other"}}`), nil
				case "invalid-json":
					return json.RawMessage(`{`), nil
				}
				return good, nil
			})
			if (err == nil) != (mode == "success") {
				t.Fatalf("mode=%s error=%v", mode, err)
			}
			if mode == "expired" && calls != 0 {
				t.Fatal("expired context performed read")
			}
			if mode == "read-error" && !errors.Is(err, fault) {
				t.Fatal("read failure lost")
			}
		})
	}
	h := &pgPerfHarness{}
	if err := h.recordRaceOutcome(context.Background(), "unused", [2]error{}, "", nil); err != nil {
		t.Fatal(err)
	}
}

func (h *pgPerfHarness) recordRaceOutcome(ctx context.Context, id string, results [2]error, graphPath string, pids []int32) error {
	if !h.raceEvidence {
		return nil
	}
	started := time.Now()
	snapshot, err := pgPerfRaceSnapshot(ctx, id, func(ctx context.Context, id string) (json.RawMessage, error) {
		var data []byte
		err := h.pool.QueryRow(ctx, `SELECT json_build_object('article',json_build_object('id',a.id,'version',a.version,'publication_state',a.publication_state,'violation_state',a.violation_state,'violation_category',a.violation_category,'deletion_state',a.deletion_state,'deletion_operation_id',a.deletion_operation_id),'operations',COALESCE((SELECT json_agg(d) FROM deletion_operations d WHERE d.article_id=a.id),'[]'::json),'audit',COALESCE((SELECT json_agg(e) FROM audit_events e WHERE e.target_id=a.id::text),'[]'::json)) FROM articles a WHERE a.id=$1::uuid`, id).Scan(&data)
		return json.RawMessage(data), err
	})
	if err != nil {
		return err
	}
	actions := [2]string{"ReplaceContent", "Moderate"}
	if h.directedCase == "delete-race" {
		actions = [2]string{"beginT1", "ReplaceContent"}
	}
	outcomes := make([]map[string]any, 2)
	for i, result := range results {
		outcomes[i] = map[string]any{"participant": i, "action": actions[i], "expected_version": 1, "success": result == nil, "error_type": fmt.Sprintf("%T", result), "error_class": pgPerfErrorClass(result), "precondition": errors.Is(result, article.ErrPreconditionFailed), "unavailable": errors.Is(result, article.ErrArticleUnavailable)}
	}
	deadline, _ := ctx.Deadline()
	h.evidence.writeJSONExclusive(filepath.Join("races", h.schema+"."+id+".outcome.json"), map[string]any{"run_id": h.evidence.runID, "schema": h.schema, "logical_case": h.directedCase, "article_id": id, "outcomes": outcomes, "snapshot": snapshot, "lock_graph_path": graphPath, "pids": pids, "started": started, "completed": time.Now(), "deadline": deadline})
	h.evidence.mu.Lock()
	writeErr := h.evidence.finalErr
	h.evidence.mu.Unlock()
	return errors.Join(writeErr, ctx.Err())
}

func newPgPerfHarness(t testing.TB, tier pgPerfTier, caseCode string, options ...func(*pgPerfOptions)) *pgPerfHarness {
	t.Helper()
	settings := pgPerfOptions{
		diskProbe:        pgPerfDiskFree,
		maxConns:         tier.maxConns,
		statementTimeout: pgPerfMeasureStatementTimeout,
		lockTimeout:      0,
		seedMode:         "full",
	}
	for _, option := range options {
		option(&settings)
	}
	var caseBudget pgPerfCaseBudget
	var profileErr error
	tier, caseBudget, profileErr = pgPerfDirectedTier(tier, caseCode, settings.directedCase)
	if profileErr != nil {
		t.Fatal(profileErr)
	}
	evidence := pgPerfGlobalEvidence(t)
	if evidence.stopReason != "" {
		t.Fatalf("NOT RUN after prior failure: %s", evidence.stopReason)
	}
	policy := pgPerfPolicy(t)
	deadline := policy.ExecutionStart.Add(time.Duration(policy.StopNewSeconds) * time.Second)
	if time.Until(deadline) <= tier.contextBudget {
		t.Fatal("NOT RUN: insufficient remaining small/medium window for another scene")
	}
	caseBudget.RootStarted = time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), tier.contextBudget)
	caseBudget.RootDeadline, _ = ctx.Deadline()
	windowCancel := time.AfterFunc(time.Until(deadline), cancel)
	goroutineBase := runtime.NumGoroutine()
	h := &pgPerfHarness{
		directedCase:     settings.directedCase,
		caseBudget:       caseBudget,
		raceEvidence:     settings.directedCase == "moderation-race" || settings.directedCase == "delete-race",
		policy:           policy,
		diskProbe:        settings.diskProbe,
		t:                t,
		ctx:              ctx,
		cancel:           cancel,
		tier:             tier,
		caseCode:         caseCode,
		schema:           pgPerfSchemaName(t, tier.code, caseCode, evidence.runID),
		appName:          pgPerfApplicationName(evidence.runID),
		evidence:         evidence,
		statementTimeout: settings.statementTimeout,
		lockTimeout:      settings.lockTimeout,
		ids:              &pgPerfIDs{prefix: caseCode + "|" + evidence.runID},
	}
	h.diskCtx, h.diskCancel = context.WithCancel(ctx)
	h.resource = newPgPerfResourceState(h.diskCtx, cancel, policy.ProbeBudget)
	ctx = context.WithValue(ctx, pgPerfResourceKey{}, h.resource)
	h.ctx = ctx
	t.Cleanup(func() {
		windowCancel.Stop()
		h.closeResources(goroutineBase)
		t.Logf("pgperf_schema=%s created=%t retained=true", h.schema, evidence.hasSchema(h.schema))
	})

	pgPerfEnsureDatabase(t, ctx, h.appName)
	probe, err := pgxpool.NewWithConfig(ctx, pgPerfPoolConfig(t, h.schema, h.appName, 1, 0, 0))
	if err != nil {
		t.Fatalf("open pgperf probe pool: %T", err)
	}
	h.probe = probe
	var host string
	var port, version int
	if err := probe.QueryRow(ctx, "SELECT host(inet_server_addr()),inet_server_port(),current_setting('server_version_num')::int").Scan(&host, &port, &version); err != nil {
		t.Fatalf("pgperf endpoint probe failed: %T", err)
	}
	if net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() || port != 5433 || version != 180006 {
		t.Fatal("pgperf requires the authorized loopback PostgreSQL 18.6 endpoint")
	}
	if err := probe.QueryRow(ctx, "SHOW data_directory").Scan(&h.pgdataPath); err != nil {
		t.Fatalf("read PGDATA before resource observation: %T", err)
	}
	h.resourceBaseBytes = policy.Resource.BaseBytes
	h.resourceTierBytes = policy.Resource.Baselines[tier.name].Bytes
	h.startResourceExpiry()
	if err := h.checkDiskBudgetContext(ctx); err != nil {
		t.Fatalf("initial accurate resource observation failed; no DDL: %v", err)
	}
	h.resource.mu.Lock()
	h.baselineDBBytes = h.resource.last.Bytes
	h.resource.mu.Unlock()
	actual, err := pgPerfLoadTierBaseline(policy.Resource, tier.name, h.baselineDBBytes)
	if err != nil {
		t.Fatalf("read bound persistent tier baseline: %v", err)
	}
	h.actualBaseline = actual
	h.resourceTierBytes = actual.Bytes
	evidence.mu.Lock()
	evidence.tierBaseDBBytes = actual.Bytes
	evidence.baseDBBytes = policy.Resource.BaseBytes
	evidence.mu.Unlock()
	if free, status := h.diskProbe(h.pgdataPath); status != "ok" || free <= 0 {
		t.Fatal("PGDATA volume free space unavailable; no DDL authorized")
	}
	h.checkDiskBudget()
	if t.Failed() {
		t.Fatal("NOT RUN: disk budget check failed before schema creation")
	}
	var activityBaseline int
	if err := probe.QueryRow(ctx, "SELECT count(*) FROM pg_stat_activity").Scan(&activityBaseline); err != nil {
		t.Fatalf("read pg_stat_activity baseline: %v", err)
	}
	settingsMap := map[string]string{}
	rows, err := probe.Query(ctx, `SELECT name, setting FROM pg_settings WHERE name IN (
		'server_version','max_connections','shared_buffers','work_mem','maintenance_work_mem',
		'effective_cache_size','fsync','synchronous_commit','wal_level','max_wal_size',
		'checkpoint_timeout','log_lock_waits','default_statistics_target','random_page_cost',
		'max_worker_processes','max_parallel_workers','max_parallel_workers_per_gather')`)
	if err != nil {
		t.Fatalf("read pg_settings: %T", err)
	}
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			rows.Close()
			t.Fatalf("scan pg_settings: %T", err)
		}
		settingsMap[name] = value
	}
	rows.Close()
	diskFree, diskProbe := pgPerfDiskFree(pgPerfEvidenceRoot(t))
	dataDirectory := h.pgdataPath
	pgdataFree, pgdataProbe := pgPerfDiskFree(dataDirectory)
	hardStopFree := pgdataFree
	if pgdataProbe != "ok" || hardStopFree <= 0 {
		t.Fatal("PGDATA free-space probe failed before DDL")
	}
	var pgStatStatements bool
	_ = probe.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname='pg_stat_statements')").Scan(&pgStatStatements)
	evidence.addBlocked("X09: Journal fsync/physical delete/204, real SMTP/TLS, HTTP E2E, retention/RPO/RTO, backup/restore are unavailable or unauthorized in this window")
	if !pgStatStatements {
		evidence.addBlocked("pg_stat_statements extension is not installed; optional statement statistics recorded as unavailable")
	}
	if hardStopFree > 0 && h.baselineDBBytes+tier.diskLimit > hardStopFree {
		t.Fatalf("pgperf disk budget exceeds free space: db=%d tier_limit=%d pgdata_free=%d evidence_free=%d",
			h.baselineDBBytes, tier.diskLimit, pgdataFree, diskFree)
	}
	fingerprint := map[string]any{
		"execution_window_start":         policy.WindowStart,
		"attempt_id":                     policy.AttemptID,
		"resource_binding":               policy.Resource,
		"resource_actual_baseline":       h.actualBaseline,
		"adoption_source":                policy.AdoptionSource,
		"adoption_sha256":                policy.AdoptionSHA256,
		"resource_probe_budget":          policy.ProbeBudget,
		"resource_probe_adoption":        policy.ProbeAdoption,
		"resource_probe_adoption_sha256": policy.ProbeAdoptionSHA,
		"pg_version_num":                 version,
		"pg_stat_activity":               activityBaseline,
		"database_size_base":             h.baselineDBBytes,
		"disk_free_bytes":                diskFree,
		"disk_probe":                     diskProbe,
		"pg_data_directory":              dataDirectory,
		"disk_free_pgdata_bytes":         pgdataFree,
		"disk_free_pgdata_probe":         pgdataProbe,
		"pg_stat_statements":             pgStatStatements,
		"tier":                           tier.name,
		"case":                           caseCode,
		"schema":                         h.schema,
		"max_conns":                      settings.maxConns,
		"seed_mode":                      settings.seedMode,
	}
	if h.directedCase == "" {
		evidence.writeFingerprint(h, settingsMap, fingerprint)
	}

	diskPool, err := pgxpool.NewWithConfig(ctx, pgPerfPoolConfig(t, h.schema, h.appName, 1, 0, 0))
	if err != nil {
		t.Fatalf("open dedicated disk monitor: %T", err)
	}
	h.diskPool = diskPool
	if settings.testDiskLimitDelta > 0 {
		h.tier.diskLimit = min(h.tier.diskLimit, h.baselineDBBytes-h.resourceTierBytes+settings.testDiskLimitDelta)
	}
	h.startDiskMonitor(ctx)
	pool, err := pgxpool.NewWithConfig(ctx, h.ownedPoolConfig(settings.maxConns, settings.statementTimeout, settings.lockTimeout))
	if err != nil {
		t.Fatalf("open pgperf pool: %T", err)
	}
	h.pool = pool
	h.caseBudget.SeedStarted = time.Now()
	seedCtx, seedCancel := context.WithTimeout(ctx, tier.seedBudget)
	h.caseBudget.SeedDeadline, _ = seedCtx.Deadline()
	if h.directedCase != "" {
		fingerprint["case_budget"] = h.caseBudget
		evidence.writeFingerprint(h, settingsMap, fingerprint)
	}
	defer seedCancel()
	h.ctx = seedCtx
	if err := h.checkDiskBudgetContext(seedCtx); err != nil {
		t.Fatalf("synchronous pre-DDL resource guard: %v", err)
	}
	pgPerfMigrate(t, seedCtx, pool, h.schema, evidence, tier.name, caseCode, settings.failMigration)
	t.Logf("retained_schema=%s", h.schema)
	if settings.beforeSeed != nil {
		settings.beforeSeed(h)
	}
	if !settings.skipSeed {
		config := pgPerfDefaultSeedConfig(tier, settings.seedMode)
		if settings.seedConfig != nil {
			config = *settings.seedConfig
		}
		h.seed = pgPerfSeed(h, config)
	}
	h.ctx = ctx
	h.checkDiskBudget()
	if t.Failed() {
		t.Fatal("NOT RUN: disk budget check failed after seed")
	}
	if strings.HasPrefix(caseCode, "r0") || strings.HasPrefix(caseCode, "x02") || strings.HasPrefix(caseCode, "x03") || caseCode == "d50" || caseCode == "d90" || caseCode == "d100" {
		h.captureReadPlans()
		if t.Failed() {
			t.Fatal("read plan collection failed before measurement")
		}
	}
	return h
}

func (h *pgPerfHarness) startDiskMonitor(ctx context.Context) {
	h.group.Add(1)
	go func() {
		defer h.group.Done()
		ticker := time.NewTicker(h.resource.budget.Tick)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-h.diskCtx.Done():
				return
			case <-ticker.C:
				if err := h.checkDiskBudgetContext(h.diskCtx); err != nil {
					return
				}
				select {
				case <-ticker.C:
				default:
				}
			}
		}
	}()
}

func (h *pgPerfHarness) startResourceExpiry() {
	h.group.Add(1)
	go func() {
		defer h.group.Done()
		for {
			h.resource.mu.Lock()
			end := h.resource.freshnessLocked()
			h.resource.mu.Unlock()
			var expired <-chan time.Time
			var timer *time.Timer
			if !end.IsZero() {
				timer = time.NewTimer(end.Sub(h.resource.now()))
				expired = timer.C
			}
			select {
			case <-h.diskCtx.Done():
				if timer != nil {
					timer.Stop()
				}
				return
			case <-h.resource.changed:
				if timer != nil {
					timer.Stop()
				}
			case <-expired:
				if h.diskCtx.Err() != nil {
					return
				}
				if err := h.resource.admit(h.diskCtx); err != nil {
					if h.evidence != nil {
						h.evidence.writeJSON(filepath.Join("failures", fmt.Sprintf("resource-expiry.%d.json", time.Now().UnixNano())), map[string]any{"actual_utc": time.Now().UTC(), "deadline_utc": end.UTC(), "overrun_ns": h.resource.now().Sub(end).Nanoseconds(), "error": err.Error(), "root_canceled": h.diskCtx.Err() != nil})
					}
					return
				}
			}
		}
	}()
}

func (h *pgPerfHarness) exec(sql string, args ...any) {
	h.t.Helper()
	if _, err := h.pool.Exec(h.ctx, sql, args...); err != nil {
		h.t.Fatalf("pgperf exec failed: %T", err)
	}
}

func (h *pgPerfHarness) scalar(sql string, args ...any) int64 {
	h.t.Helper()
	var value int64
	if err := h.pool.QueryRow(h.ctx, sql, args...).Scan(&value); err != nil {
		h.t.Fatalf("pgperf scalar failed: %T", err)
	}
	return value
}

func (h *pgPerfHarness) now() time.Time {
	h.t.Helper()
	var value time.Time
	if err := h.pool.QueryRow(h.ctx, "SELECT clock_timestamp()").Scan(&value); err != nil {
		h.t.Fatalf("pgperf clock failed: %T", err)
	}
	return value.UTC().Truncate(time.Microsecond)
}

func (h *pgPerfHarness) begin(ctx context.Context) pgx.Tx {
	h.t.Helper()
	tx, err := h.beginOwned(ctx)
	if err != nil {
		h.t.Fatalf("pgperf begin failed: %T", err)
	}
	return tx
}

func (h *pgPerfHarness) queryService() *article.ArticleQueryService {
	h.t.Helper()
	repo, err := article.NewPostgresArticleQueryRepository(h.pool)
	if err != nil {
		h.t.Fatalf("create pgperf query repository: %v", err)
	}
	reader, err := identity.NewArticleIdentityReader(h.pool)
	if err != nil {
		h.t.Fatalf("create pgperf identity reader: %v", err)
	}
	service, err := article.NewArticleQueryService(repo, reader, article.QueryConfig{DefaultLimit: 20, MaxLimit: 100, CandidateBatchSize: 200})
	if err != nil {
		h.t.Fatalf("create pgperf query service: %v", err)
	}
	return service
}

func (h *pgPerfHarness) commandService() *article.CommandService {
	h.t.Helper()
	service, err := article.NewCommandService(article.CommandDependencies{
		Transactions: pgPerfTransactor{pool: h.pool, owner: h},
		Repository:   article.Repository{},
		IDs:          h.ids,
		Clock:        pgPerfClock{},
		Moderation: &article.ModerationDependencies{
			Auditor:      audit.Store{},
			AuthorStates: s22AuthorStates,
		},
	})
	if err != nil {
		h.t.Fatalf("create pgperf command service: %v", err)
	}
	return service
}

func pgPerfPrincipal(id, role string) session.Principal {
	return session.Principal{UserID: id, Role: role, Status: "active"}
}

func (h *pgPerfHarness) memberPrincipal(ctx context.Context) (session.Principal, error) {
	if err := pgPerfOperationAdmit(ctx); err != nil {
		return session.Principal{}, err
	}
	var id string
	err := h.pool.QueryRow(ctx, "SELECT id::text FROM users WHERE status='active' ORDER BY id LIMIT 1").Scan(&id)
	return pgPerfPrincipal(id, "member"), err
}

func (h *pgPerfHarness) activeUserIDs(limit int) []string {
	h.t.Helper()
	rows, err := h.pool.Query(h.ctx, "SELECT id::text FROM users WHERE status='active' ORDER BY id LIMIT $1", limit)
	if err != nil {
		h.t.Fatalf("pgperf active users: %T", err)
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			h.t.Fatalf("scan active user: %T", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		h.t.Fatalf("iterate active users: %T", err)
	}
	if len(ids) == 0 {
		h.t.Fatal("pgperf fixture has no active user")
	}
	return ids
}

func (h *pgPerfHarness) activeUserID() string {
	h.t.Helper()
	return h.activeUserIDs(1)[0]
}

func (h *pgPerfHarness) articleByID(ctx context.Context, id string) article.Article {
	h.t.Helper()
	var record article.Article
	err := h.pool.QueryRow(ctx, `SELECT id, author_id, orphaned_from_user_id, title, body, publication_state,
		violation_state, violation_category, deletion_state, deletion_operation_id, version, created_at, updated_at
		FROM articles WHERE id=$1::uuid`, id).Scan(
		&record.ID, &record.AuthorID, &record.OrphanedFromUserID, &record.Title, &record.Body,
		&record.PublicationState, &record.ViolationState, &record.ViolationCategory,
		&record.DeletionState, &record.DeletionOperationID, &record.Version,
		&record.CreatedAt, &record.UpdatedAt)
	if err != nil {
		h.t.Fatalf("pgperf read article: %T", err)
	}
	return record
}

func (h *pgPerfHarness) activeArticleFor(ctx context.Context, authorID string) article.Article {
	h.t.Helper()
	var id string
	err := h.pool.QueryRow(ctx, `SELECT id::text FROM articles
		WHERE deletion_state='active' AND publication_state='published' AND violation_state='clear'
			AND author_id=$1::uuid ORDER BY id LIMIT 1`, authorID).Scan(&id)
	if err != nil {
		h.t.Fatalf("pgperf pick active article: %T", err)
	}
	return h.articleByID(ctx, id)
}

// memberVisibleIDs is the independent SQL oracle; it never calls the query
// service or its cursor codec to derive expectations.
func (h *pgPerfHarness) memberVisibleIDs(ctx context.Context, limit int) []string {
	h.t.Helper()
	query := `SELECT a.id::text FROM articles a LEFT JOIN users u ON u.id=a.author_id
		WHERE a.deletion_state='active' AND a.publication_state='published'
			AND a.violation_state='clear' AND (a.author_id IS NULL OR u.status='active')
		ORDER BY a.created_at DESC,a.id DESC`
	args := []any{}
	if limit > 0 {
		query += " LIMIT $1"
		args = append(args, limit)
	}
	rows, err := h.pool.Query(ctx, query, args...)
	if err != nil {
		h.t.Fatalf("pgperf oracle query: %T", err)
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			h.t.Fatalf("scan oracle row: %T", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		h.t.Fatalf("iterate oracle rows: %T", err)
	}
	return ids
}

func (h *pgPerfHarness) countRows(ctx context.Context, table string) int64 {
	h.t.Helper()
	var value int64
	if err := h.pool.QueryRow(ctx, "SELECT count(*) FROM "+pgx.Identifier{table}.Sanitize()).Scan(&value); err != nil {
		h.t.Fatalf("pgperf count %s: %T", table, err)
	}
	return value
}

func (h *pgPerfHarness) currentPID(ctx context.Context) uint32 {
	h.t.Helper()
	var pid uint32
	if err := h.pool.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
		h.t.Fatalf("pgperf backend pid: %T", err)
	}
	return pid
}

func (h *pgPerfHarness) pollUntil(ctx context.Context, timeout time.Duration, check func(context.Context) (bool, error)) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		ok, err := check(ctx)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New("bounded poll timed out")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// waitBlocking waits until at least want backends report pid as a blocker.
func (h *pgPerfHarness) waitBlocking(ctx context.Context, pid uint32, want int) error {
	return h.pollUntil(ctx, 5*time.Second, func(ctx context.Context) (bool, error) {
		var blocked int
		if err := h.pool.QueryRow(ctx,
			"SELECT count(*) FROM pg_stat_activity WHERE $1::int = ANY(pg_blocking_pids(pid))", pid).Scan(&blocked); err != nil {
			return false, err
		}
		return blocked >= want, nil
	})
}

func (h *pgPerfHarness) raceRow(ctx context.Context, id string, actions [2]func(context.Context) error) (results [2]error, failure error) {
	raceCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	holder, err := h.beginOwned(raceCtx)
	if err != nil {
		return results, err
	}
	type outcome struct {
		index int
		err   error
	}
	done := make(chan outcome, 2)
	joined := make(chan struct{}, 2)
	remaining := 0
	defer func() {
		cancel()
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer closeCancel()
		if err := holder.Rollback(closeCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			failure = errors.Join(failure, err)
		}
		for remaining > 0 {
			select {
			case result := <-done:
				results[result.index] = result.err
				remaining--
			case <-closeCtx.Done():
				failure = errors.Join(failure, errors.New("race workers failed to join after cancellation"))
				return
			}
		}
	}()
	var pid uint32
	if err := holder.QueryRow(raceCtx, "SELECT pg_backend_pid() FROM articles WHERE id=$1::uuid FOR UPDATE", id).Scan(&pid); err != nil {
		return results, err
	}
	pids := [2]chan uint32{make(chan uint32, 1), make(chan uint32, 1)}
	start := make(chan struct{})
	for index, action := range actions {
		h.group.Add(1)
		remaining++
		go func(index int, action func(context.Context) error) {
			defer func() { h.group.Done(); joined <- struct{}{} }()
			<-start
			done <- outcome{index, action(context.WithValue(raceCtx, pgPerfRacePIDKey{}, pids[index]))}
		}(index, action)
	}
	close(start)
	waiters := make([]int32, 2)
	for i, ch := range pids {
		select {
		case p := <-ch:
			waiters[i] = int32(p)
		case result := <-done:
			remaining--
			results[result.index] = result.err
			return results, fmt.Errorf("race contender returned before lock observation: %w", result.err)
		case <-raceCtx.Done():
			return results, raceCtx.Err()
		}
	}
	if waiters[0] == waiters[1] || uint32(waiters[0]) == pid || uint32(waiters[1]) == pid {
		return results, errors.New("race requires three distinct backends")
	}
	var graph any
	err = h.pollUntil(raceCtx, 4*time.Second, func(ctx context.Context) (bool, error) {
		var both bool
		err := h.probe.QueryRow(ctx, `WITH RECURSIVE dependencies(root,pid,path) AS (
			SELECT p,p,ARRAY[p] FROM unnest($1::int[]) p
			UNION ALL SELECT d.root,b,d.path||b FROM dependencies d,
			LATERAL unnest(pg_blocking_pids(d.pid)) b WHERE NOT b=ANY(d.path)
		) SELECT count(DISTINCT root)=2 FROM dependencies WHERE pid=$2::int`, waiters, pid).Scan(&both)
		return both, err
	})
	probeCtx, probeCancel := context.WithTimeout(context.Background(), time.Second)
	graphErr := h.probe.QueryRow(probeCtx, `SELECT json_agg(json_build_object('pid',pid,'blockers',pg_blocking_pids(pid),'wait_event_type',wait_event_type,'wait_event',wait_event)) FROM pg_stat_activity WHERE pid=ANY($1::int[])`, append(append([]int32{}, waiters...), int32(pid))).Scan(&graph)
	probeCancel()
	graphPath := filepath.Join("locks", fmt.Sprintf("%s.%d.race.json", h.schema, time.Now().UnixNano()))
	h.evidence.writeJSONExclusive(graphPath, map[string]any{"holder_pid": pid, "contender_pids": waiters, "graph": graph, "both_depend_on_holder": err == nil, "probe_error": pgPerfErrorClass(graphErr)})
	if err != nil || graphErr != nil {
		return results, errors.Join(err, graphErr)
	}
	if err := holder.Rollback(raceCtx); err != nil {
		return results, err
	}
	for remaining > 0 {
		select {
		case result := <-done:
			remaining--
			results[result.index] = result.err
		case <-raceCtx.Done():
			return results, raceCtx.Err()
		}
	}
	if h.raceEvidence {
		for i := 0; i < 2; i++ {
			select {
			case <-joined:
			case <-raceCtx.Done():
				return results, raceCtx.Err()
			}
		}
	}
	return results, h.recordRaceOutcome(raceCtx, id, results, graphPath, append(waiters, int32(pid)))
}

func (h *pgPerfHarness) insertGateRecordTx(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `INSERT INTO processing_gate_records(
		region_ref,controller_ref,legal_basis_ref,smtp_processor_ref,effective_at,recorded_by)
		VALUES('synthetic-region-open','synthetic-controller','synthetic-basis','synthetic-processor',
		clock_timestamp()-interval '1 second','pgperf-fixture')`)
	return err
}

func (h *pgPerfHarness) insertGateRecord(ctx context.Context) error {
	tx, err := h.beginOwned(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if err := h.insertGateRecordTx(ctx, tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// beginT1 performs the real T1 visibility transition without the journal or
// finalizer, which r04 keeps out of scope.
func (h *pgPerfHarness) beginT1(ctx context.Context, articleID, actorID, role string, expected ...int64) (article.DeletionOperation, error) {
	tx, err := h.beginOwned(ctx)
	if err != nil {
		return article.DeletionOperation{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	record, existing, err := (article.Repository{}).LockArticleForDeletion(ctx, tx, articleID)
	if err != nil {
		return article.DeletionOperation{}, err
	}
	if len(expected) != 0 && record.Version != expected[0] {
		return article.DeletionOperation{}, article.ErrPreconditionFailed
	}
	if existing != nil {
		return *existing, tx.Commit(ctx)
	}
	var now time.Time
	if err := tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); err != nil {
		return article.DeletionOperation{}, err
	}
	now = now.UTC().Truncate(time.Microsecond)
	operationID := pgPerfUUID(h.schema, "t1", articleID)
	actor, actorRole := actorID, role
	operation := article.DeletionOperation{
		ID: operationID, ArticleID: articleID, ArticleVersion: record.Version,
		DeleteKind: article.DeleteOrdinary, ActorUserID: &actor, ActorRole: &actorRole,
		Status: article.DeletionOperationPendingJournal, StartedAt: now,
	}
	if err := (article.Repository{}).InsertDeletionOperation(ctx, tx, operation); err != nil {
		return article.DeletionOperation{}, err
	}
	if err := (article.Repository{}).SetArticleDeletionPending(ctx, tx, articleID, record.Version, operationID, now); err != nil {
		return article.DeletionOperation{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return article.DeletionOperation{}, err
	}
	return operation, nil
}

// pgPerfCloneFrom copies the canonical large fixture into this migrated schema
// in FK-safe order. The source schema is retained and never dropped.
func (h *pgPerfHarness) pgPerfCloneFrom(sourceSchema string) pgPerfSeedManifest {
	h.t.Helper()
	started := time.Now()
	manifest := pgPerfSeedManifest{Schema: h.schema, Tier: h.tier.name, Case: h.caseCode, SeedTimeout: "0",
		Note: "cloned from " + sourceSchema}
	tx, err := h.beginOwned(h.ctx)
	if err != nil {
		h.t.Fatalf("begin pgperf clone: %T", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(context.Background())
		}
	}()
	if _, err := tx.Exec(h.ctx, "SET LOCAL statement_timeout=0"); err != nil {
		h.t.Fatalf("disable clone statement timeout: %T", err)
	}
	for _, table := range []string{"users", "deletion_operations", "articles", "transfer_notices", "transfer_notice_recipients", "audit_events"} {
		if err := h.checkDiskBudgetContext(h.ctx); err != nil {
			h.t.Fatalf("clone batch resource guard: %v", err)
		}
		if err := pgPerfOperationAdmit(h.ctx); err != nil {
			h.t.Fatalf("clone batch admission: %v", err)
		}
		statement := "INSERT INTO " + pgx.Identifier{table}.Sanitize() + " SELECT * FROM " + pgx.Identifier{sourceSchema, table}.Sanitize()
		if _, err := tx.Exec(h.ctx, statement); err != nil {
			h.t.Fatalf("clone %s: %T", table, err)
		}
	}
	if err := h.insertGateRecordTx(h.ctx, tx); err != nil {
		h.t.Fatalf("clone gate record: %T", err)
	}
	if err := tx.Commit(h.ctx); err != nil {
		h.t.Fatalf("commit pgperf clone: %T", err)
	}
	committed = true
	manifest.Users = int(h.countRows(h.ctx, "users"))
	manifest.Articles = int(h.countRows(h.ctx, "articles"))
	manifest.Notices = int(h.countRows(h.ctx, "transfer_notices"))
	manifest.Recipients = int(h.countRows(h.ctx, "transfer_notice_recipients"))
	manifest.Audits = int(h.countRows(h.ctx, "audit_events"))
	manifest.Hidden = int(h.scalar("SELECT count(*) FROM articles WHERE NOT (deletion_state='active' AND publication_state='published' AND violation_state='clear')"))
	manifest.SeedMS = time.Since(started).Milliseconds()
	analyzeStarted := time.Now()
	analyzeTx, err := h.beginOwned(h.ctx)
	if err != nil {
		h.t.Fatalf("begin clone analyze: %T", err)
	}
	if _, err := analyzeTx.Exec(h.ctx, "SET LOCAL statement_timeout=0"); err != nil {
		_ = analyzeTx.Rollback(context.Background())
		h.t.Fatalf("disable analyze timeout: %T", err)
	}
	if _, err := analyzeTx.Exec(h.ctx, "ANALYZE users, articles, deletion_operations, transfer_notices, transfer_notice_recipients, audit_events"); err != nil {
		_ = analyzeTx.Rollback(context.Background())
		h.t.Fatalf("analyze clone: %T", err)
	}
	if err := analyzeTx.Commit(h.ctx); err != nil {
		h.t.Fatalf("commit clone analyze: %T", err)
	}
	manifest.AnalyzeMS = time.Since(analyzeStarted).Milliseconds()
	h.evidence.addSeed(manifest)
	return manifest
}

var (
	pgPerfLargeSourceOnce   sync.Once
	pgPerfLargeSourceSchema string
)

// pgPerfEnsureLargeSource builds the canonical 100k source exactly once per
// large window and returns its retained schema name.
func pgPerfEnsureLargeSource(t *testing.T) string {
	t.Helper()
	pgPerfLargeSourceOnce.Do(func() {
		t.Run("L00-canonical-source", func(t *testing.T) {
			h := newPgPerfHarness(t, pgPerfTierLarge, "canon", func(options *pgPerfOptions) {
				options.seedMode = "full"
			})
			pgPerfLargeSourceSchema = h.schema
			t.Logf("pgperf_large_source=%s retained=true", h.schema)
		})
	})
	if pgPerfLargeSourceSchema == "" {
		t.Fatal("pgperf large canonical source is unavailable")
	}
	return pgPerfLargeSourceSchema
}

func (h *pgPerfHarness) writePlan(name string, query string, args ...any) {
	h.t.Helper()
	rows, err := h.pool.Query(h.ctx, "EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) "+query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42501" {
			h.evidence.addBlocked("EXPLAIN permission unavailable for " + name)
		} else {
			h.t.Errorf("EXPLAIN failed for %s: %T", name, err)
		}
		return
	}
	defer rows.Close()
	if rows.Next() {
		var plan any
		if err := rows.Scan(&plan); err != nil {
			h.t.Errorf("scan EXPLAIN plan: %T", err)
			return
		}
		h.evidence.writePlan(name, map[string]any{"schema": h.schema, "query": query, "args": args, "plan": plan})
	} else {
		h.t.Error("EXPLAIN returned no plan")
	}
	if err := rows.Err(); err != nil {
		h.t.Errorf("read EXPLAIN rows: %T", err)
	}
}

func (h *pgPerfHarness) captureReadPlans() {
	path := filepath.Join("..", "..", "..", "internal", "article", "query_service.go")
	source, err := os.ReadFile(path)
	if err != nil {
		h.t.Fatal(err)
	}
	query := func(name string) string {
		marker := "const " + name + " = `"
		start := strings.Index(string(source), marker)
		if start < 0 {
			h.t.Fatalf("production query %s is no longer a raw SQL constant", name)
		}
		rest := string(source)[start+len(marker):]
		end := strings.Index(rest, "`")
		if end < 0 {
			h.t.Fatal("unterminated production query source")
		}
		return rest[:end]
	}
	h.writePlan(h.caseCode+"-candidates-first", query("listArticleCandidatesSQL"), nil, nil, 200)
	var created time.Time
	var id string
	err = h.pool.QueryRow(h.ctx, "SELECT created_at,id::text FROM articles ORDER BY created_at,id LIMIT 1").Scan(&created, &id)
	if errors.Is(err, pgx.ErrNoRows) {
		return
	}
	if err != nil {
		h.t.Fatalf("read plan keyset fixture: %T", err)
	}
	h.writePlan(h.caseCode+"-candidates-tail-edge", query("listArticleCandidatesSQL"), created, id, 200)
	h.writePlan(h.caseCode+"-detail", query("getArticleQuerySQL"), id)
}

type pgPerfSeedConfig struct {
	Label           string
	Users           int
	Articles        int
	Deletions       int
	PendingOverride int
	Notices         int
	Audits          int
	HiddenDensity   float64
	NearLimitShare  float64
	BodyMin         int
	BodyMax         int
}

func pgPerfDefaultSeedConfig(tier pgPerfTier, mode string) pgPerfSeedConfig {
	switch mode {
	case "empty":
		return pgPerfSeedConfig{Label: "empty"}
	case "ten":
		return pgPerfSeedConfig{Label: "ten", Users: 3, Articles: 10, Audits: 2, BodyMin: 200, BodyMax: 400}
	case "minimal":
		return pgPerfSeedConfig{
			Label: "minimal", Users: 8, Articles: 200, Deletions: 20, Notices: 20, Audits: 10,
			HiddenDensity: 0.2, NearLimitShare: 0.05, BodyMin: tier.bodyMin, BodyMax: tier.bodyMax,
		}
	default:
		return pgPerfSeedConfig{
			Label: "full", Users: tier.users, Articles: tier.articles, Deletions: tier.deletions,
			Notices: tier.notices, Audits: tier.audits, HiddenDensity: 0.25,
			NearLimitShare: tier.nearLimitRate, BodyMin: tier.bodyMin, BodyMax: tier.bodyMax,
		}
	}
}

func pgPerfNearLimitBody() string {
	return strings.Repeat("中", 19990) + "end"
}

func pgPerfTypicalBody(r *rand.Rand, minBytes, maxBytes int) string {
	if maxBytes <= minBytes {
		maxBytes = minBytes + 1
	}
	target := minBytes + r.Intn(maxBytes-minBytes+1)
	if target < 16 {
		target = 16
	}
	return strings.Repeat("文", target/3+1)
}

func pgPerfSeed(h *pgPerfHarness, config pgPerfSeedConfig) pgPerfSeedManifest {
	h.t.Helper()
	started := time.Now()
	h.seedProgress("seed-start", config)
	defer func() {
		h.seedProgress("seed-exit", map[string]any{"elapsed_ns": time.Since(started).Nanoseconds(), "context_error": pgPerfErrorClass(h.ctx.Err())})
	}()
	manifest := pgPerfSeedManifest{Schema: h.schema, Tier: h.tier.name, Case: h.caseCode, SeedTimeout: "0"}
	if config.Articles <= 0 {
		manifest.SeedMS = time.Since(started).Milliseconds()
		h.evidence.addSeed(manifest)
		return manifest
	}
	rng := rand.New(rand.NewSource(pgPerfSeedInt(h.schema + "|" + config.Label)))
	base := time.Now().UTC().Truncate(time.Microsecond)

	activeUsers := config.Users - 2
	if activeUsers < 1 {
		activeUsers = 1
	}
	if activeUsers > config.Articles {
		activeUsers = config.Articles
	}
	userIDs := make([]string, 0, activeUsers+2)
	for i := 0; i < activeUsers; i++ {
		userIDs = append(userIDs, pgPerfUUID(h.schema, "user", strconv.Itoa(i)))
	}
	suspendedID := pgPerfUUID(h.schema, "user", "suspended")
	deletedID := pgPerfUUID(h.schema, "user", "deleted")

	pendingTarget := config.PendingOverride
	if pendingTarget <= 0 {
		pendingTarget = config.Deletions
		if limit := config.Articles / 5; pendingTarget > limit {
			pendingTarget = limit
		}
	}
	if pendingTarget > config.Articles {
		pendingTarget = config.Articles
	}
	hiddenByDensity := int(math.Round(config.HiddenDensity * float64(config.Articles-pendingTarget)))
	nearLimit := int(math.Round(config.NearLimitShare * float64(config.Articles)))
	if nearLimit < 1 {
		nearLimit = 1
	}
	if nearLimit > config.Articles {
		nearLimit = config.Articles
	}

	tx, err := h.beginOwned(h.ctx)
	if err != nil {
		h.t.Fatalf("begin pgperf seed: %T", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(context.Background())
		}
	}()
	if _, err := tx.Exec(h.ctx, "SET LOCAL statement_timeout=0"); err != nil {
		h.t.Fatalf("disable seed statement timeout: %T", err)
	}

	userColumns := []string{
		"id", "username", "email", "phone", "permanent_password_digest", "permanent_password_generation", "role",
		"is_initial_admin", "status", "credential_mode", "recovery_required",
		"onboarding_privacy_acknowledged_version", "onboarding_completed_at",
		"retained_email", "created_at", "updated_at", "deleted_at",
	}
	userRows := make([][]any, 0, len(userIDs)+2)
	pushUser := func(id, username, status string) {
		email := username + "@example.test"
		phone := "+1555" + strconv.Itoa(len(username)) + strconv.Itoa(len(userRows))
		digest := "synthetic-digest-" + username
		var emailValue, phoneValue, digestValue, generationValue, retained, deletedAt any
		switch status {
		case "deleted":
			retained = email
			deletedAt = base.Add(-24 * time.Hour)
		default:
			emailValue, phoneValue, digestValue = email, phone, digest
			generationValue = int64(0)
		}
		userRows = append(userRows, []any{
			id, username, emailValue, phoneValue, digestValue, generationValue, "member", false, status, "permanent", false,
			nil, base.Add(-48 * time.Hour), retained, base.Add(-48 * time.Hour), base.Add(-48 * time.Hour), deletedAt,
		})
	}
	for index, id := range userIDs {
		pushUser(id, fmt.Sprintf("%s-active-%03d", h.caseCode, index), "active")
	}
	pushUser(suspendedID, fmt.Sprintf("%s-suspended", h.caseCode), "suspended")
	pushUser(deletedID, fmt.Sprintf("%s-deleted", h.caseCode), "deleted")
	if err := h.copySeedRows(h.ctx, tx, "users", userColumns, userRows); err != nil {
		h.t.Fatalf("seed users: %T", err)
	}
	manifest.Users = len(userRows)

	articleColumns := []string{
		"id", "author_id", "orphaned_from_user_id", "title", "body", "publication_state",
		"violation_state", "violation_category", "deletion_state", "deletion_operation_id",
		"version", "created_at", "updated_at",
	}
	articleRows := make([][]any, 0, config.Articles)
	operationColumns := []string{
		"id", "article_id", "article_version", "delete_kind", "actor_user_id", "actor_role",
		"orphaned_from_user_id", "status", "tombstone_sequence", "journal_key_id",
		"journal_key_version", "claimed_by", "locked_until", "audit_event_id", "started_at", "completed_at",
	}
	operationRows := make([][]any, 0, pendingTarget)
	for i := 0; i < config.Articles; i++ {
		if i%1000 == 0 {
			if err := pgPerfOperationAdmit(h.ctx); err != nil {
				h.t.Fatalf("seed generation canceled: %v", err)
			}
		}
		articleID := pgPerfUUID(h.schema, "article", strconv.Itoa(i))
		created := base.Add(-time.Duration(i/3) * time.Minute)
		body := pgPerfTypicalBody(rng, config.BodyMin, config.BodyMax)
		if i < nearLimit {
			body = pgPerfNearLimitBody()
			manifest.NearLimit++
		}
		authorID := any(userIDs[i%len(userIDs)])
		orphanID := any(nil)
		publication := article.PublicationPublished
		violation := article.ViolationClear
		var category any
		deletionState := article.DeletionActive
		var operationID any
		version := int64(1)
		if i < pendingTarget {
			opID := pgPerfUUID(h.schema, "op", strconv.Itoa(i))
			operationID = opID
			deletionState = article.DeletionPending
			version = 2
			operationRows = append(operationRows, []any{
				opID, articleID, int64(1), article.DeleteOrdinary, authorID, "member", nil,
				article.DeletionOperationPendingJournal, nil, nil, nil, nil, nil, nil, created, nil,
			})
			manifest.Hidden++
		} else if i < pendingTarget+hiddenByDensity {
			switch i % 4 {
			case 0:
				authorID = nil
				orphanID = deletedID
				publication = article.PublicationHeld
			case 1:
				violation = article.ViolationViolating
				category = article.ViolationViolence
			case 2:
				authorID = suspendedID
			default:
				authorID = nil
				orphanID = deletedID
				publication = article.PublicationHeld
			}
			manifest.Hidden++
		}
		articleRows = append(articleRows, []any{
			articleID, authorID, orphanID, fmt.Sprintf("pgperf %s article %d", h.caseCode, i), body,
			publication, violation, category, deletionState, operationID, version, created, created,
		})
	}
	if len(operationRows) > 0 {
		if err := h.copySeedRows(h.ctx, tx, "deletion_operations", operationColumns, operationRows); err != nil {
			h.t.Fatalf("seed deletion operations: %T", err)
		}
	}
	if err := h.copySeedRows(h.ctx, tx, "articles", articleColumns, articleRows); err != nil {
		h.t.Fatalf("seed articles: %T", err)
	}
	manifest.Articles = len(articleRows)

	if config.Notices > 0 {
		noticeColumns := []string{"id", "article_ref", "new_author_id", "former_deleted_user_id", "created_at", "updated_at"}
		recipientColumns := []string{
			"notice_id", "recipient_kind", "email_snapshot", "status", "claimed_by", "locked_until",
			"next_attempt_at", "attempt_count", "max_attempts", "last_attempt_at", "accepted_at",
			"failure_bucket", "created_at", "updated_at",
		}
		noticeRows := make([][]any, 0, config.Notices)
		recipientRows := make([][]any, 0, config.Notices*2)
		for i := 0; i < config.Notices; i++ {
			noticeID := pgPerfUUID(h.schema, "notice", strconv.Itoa(i))
			created := base.Add(-time.Duration(i) * time.Second)
			noticeRows = append(noticeRows, []any{
				noticeID, pgPerfUUID(h.schema, "notice-article", strconv.Itoa(i)),
				userIDs[i%len(userIDs)], deletedID, created, created,
			})
			recipientRows = append(recipientRows, []any{
				noticeID, article.RecipientNewAuthor, fmt.Sprintf("%s-new-%d@example.test", h.caseCode, i),
				article.RecipientPending, nil, nil, nil, int64(0), int64(3), nil, nil, nil, created, created,
			})
			if i%3 == 0 {
				recipientRows = append(recipientRows, []any{
					noticeID, article.RecipientFormerDeletedAuthor, fmt.Sprintf("%s-former-%d@example.test", h.caseCode, i),
					article.RecipientPending, nil, nil, nil, int64(0), int64(3), nil, nil, nil, created, created,
				})
			} else {
				recipientRows = append(recipientRows, []any{
					noticeID, article.RecipientFormerDeletedAuthor, fmt.Sprintf("%s-former-%d@example.test", h.caseCode, i),
					article.RecipientAccepted, nil, nil, nil, int64(1), int64(3), created, created, nil, created, created,
				})
			}
		}
		if err := h.copySeedRows(h.ctx, tx, "transfer_notices", noticeColumns, noticeRows); err != nil {
			h.t.Fatalf("seed transfer notices: %T", err)
		}
		if err := h.copySeedRows(h.ctx, tx, "transfer_notice_recipients", recipientColumns, recipientRows); err != nil {
			h.t.Fatalf("seed transfer notice recipients: %T", err)
		}
		manifest.Notices = len(noticeRows)
		manifest.Recipients = len(recipientRows)
	}

	if config.Audits > 0 {
		auditColumns := []string{
			"id", "occurred_at", "actor_kind", "actor_user_id", "actor_system", "actor_role",
			"action", "target_type", "target_id", "outcome", "error_code", "trace_id",
			"source_fingerprint", "metadata",
		}
		auditRows := make([][]any, 0, config.Audits)
		for i := 0; i < config.Audits; i++ {
			auditRows = append(auditRows, []any{
				pgPerfUUID(h.schema, "audit", strconv.Itoa(i)), base.Add(-time.Duration(i) * time.Second),
				audit.ActorSystem, nil, "pgperf", "system", "article.pgperf.seed", "article",
				pgPerfUUID(h.schema, "article", strconv.Itoa(i%len(articleRows))), "success", nil,
				noticeTrace, nil, map[string]any{},
			})
		}
		if err := h.copySeedRows(h.ctx, tx, "audit_events", auditColumns, auditRows); err != nil {
			h.t.Fatalf("seed audit events: %T", err)
		}
		manifest.Audits = len(auditRows)
	}

	if _, err := tx.Exec(h.ctx, `INSERT INTO processing_gate_records(
		region_ref,controller_ref,legal_basis_ref,smtp_processor_ref,effective_at,recorded_by)
		VALUES('synthetic-region-open','synthetic-controller','synthetic-basis','synthetic-processor',
		clock_timestamp()-interval '1 second','pgperf-fixture')`); err != nil {
		h.t.Fatalf("seed processing gate: %T", err)
	}
	if err := tx.Commit(h.ctx); err != nil {
		h.t.Fatalf("commit pgperf seed: %T", err)
	}
	committed = true
	h.seedCommitted = true
	h.seedProgress("commit-success", nil)
	manifest.SeedMS = time.Since(started).Milliseconds()
	if manifest.SeedMS > h.tier.seedBudget.Milliseconds() {
		h.t.Fatalf("seed exceeded tier budget: %dms > %dms", manifest.SeedMS, h.tier.seedBudget.Milliseconds())
	}

	analyzeStarted := time.Now()
	h.seedProgress("analyze-start", nil)
	analyzeTx, err := h.beginOwned(h.ctx)
	if err != nil {
		h.t.Fatalf("begin pgperf analyze: %T", err)
	}
	if _, err := analyzeTx.Exec(h.ctx, "SET LOCAL statement_timeout=0"); err != nil {
		_ = analyzeTx.Rollback(context.Background())
		h.t.Fatalf("disable analyze statement timeout: %T", err)
	}
	if _, err := analyzeTx.Exec(h.ctx, "ANALYZE users, articles, deletion_operations, transfer_notices, transfer_notice_recipients, audit_events"); err != nil {
		_ = analyzeTx.Rollback(context.Background())
		h.t.Fatalf("analyze pgperf seed: %T", err)
	}
	if err := analyzeTx.Commit(h.ctx); err != nil {
		h.t.Fatalf("commit pgperf analyze: %T", err)
	}
	manifest.AnalyzeMS = time.Since(analyzeStarted).Milliseconds()
	h.seedProgress("analyze-commit-success", manifest)
	h.evidence.addSeed(manifest)
	return manifest
}

// pgPerfPointSpec fixes one r04 measurement point. TIMED-3REP runs three
// warm-up + fixed-window repeats; SINGLE-RUN performs one exploratory repeat.
type pgPerfPointSpec struct {
	WarmupConcurrency int
	Point             string
	Scene             string
	Mode              string
	Concurrency       int
	Window            time.Duration
	Repeats           int
	Warmup            time.Duration
	Note              string
	BucketFn          func(worker int) string
}

const (
	pgPerfTimedRepeats     = 3
	pgPerfSingleRepeats    = 1
	pgPerfMinWindow        = 30 * time.Second
	pgPerfMinWarmup        = 10 * time.Second
	pgPerfMaxWarmup        = 30 * time.Second
	pgPerfTimedPointCap    = 120 * time.Second
	pgPerfSinglePointCap   = 60 * time.Second
	pgPerfSampleSoftTarget = 500
)

type pgPerfPhaseResult struct {
	startedNS         int64
	deadlineNS        int64
	contextObservedNS int64
	lastCompletionNS  int64
	workersJoinedNS   int64
	samples           []pgPerfSample
	attempts          int
	successes         int
	errors            int
	rejected          int
	observed          time.Duration
	completed         bool
	canceled          bool
	cutoffs           int
	tail              time.Duration
	late              int
	unknown           int
}

func pgPerfWarmupFor(window time.Duration) time.Duration {
	warmup := window / 5
	if warmup > pgPerfMaxWarmup {
		warmup = pgPerfMaxWarmup
	}
	if warmup < pgPerfMinWarmup {
		warmup = pgPerfMinWarmup
	}
	return warmup
}

func pgPerfErrorClass(err error) string {
	if err == nil {
		return ""
	}
	var resource *pgPerfSafeResourceError
	if errors.As(err, &resource) {
		return resource.Error()
	}
	switch {
	case errors.Is(err, pgPerfErrUnknownWriteOutcome):
		return "unknown_write_outcome"
	case errors.Is(err, context.Canceled):
		return "context_canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "context_deadline"
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return "pg:" + pgErr.Code
	}
	return fmt.Sprintf("%T", err)
}

func pgPerfPercentile(sorted []float64, percent float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	rank := int(math.Ceil(percent / 100 * float64(len(sorted))))
	if rank < 1 {
		rank = 1
	}
	if rank > len(sorted) {
		rank = len(sorted)
	}
	return sorted[rank-1]
}

func pgPerfBucketSummaries(samples []pgPerfSample, bucketOf func(int) string) []pgPerfBucket {
	order := []string{}
	index := map[string]*pgPerfBucket{}
	for _, sample := range samples {
		name := pgPerfSampleBucket(sample, bucketOf)
		bucket, ok := index[name]
		if !ok {
			bucket = &pgPerfBucket{Name: name}
			index[name] = bucket
			order = append(order, name)
		}
		switch {
		case sample.Error == "phase_deadline_cutoff":
			bucket.BoundaryCanceled++
		case sample.Error == "late_completed":
			bucket.Late++
		case sample.Error != "":
			bucket.Errors++
		case sample.OK:
			bucket.Successes++
		default:
			bucket.Rejected++
		}
	}
	result := make([]pgPerfBucket, 0, len(order))
	for _, name := range order {
		bucket := index[name]
		durations := make([]float64, 0, bucket.Successes)
		for _, sample := range samples {
			if pgPerfSampleBucket(sample, bucketOf) == name && sample.OK {
				durations = append(durations, float64(sample.DurationUS)/1000)
			}
		}
		bucket.N = len(durations)
		if bucket.N > 0 {
			sort.Float64s(durations)
			bucket.P50MS = pgPerfPercentile(durations, 50)
			bucket.P95MS = pgPerfPercentile(durations, 95)
			bucket.P99MS = pgPerfPercentile(durations, 99)
			bucket.MinMS = durations[0]
			bucket.MaxMS = durations[len(durations)-1]
			for _, duration := range durations {
				bucket.MeanMS += duration
			}
			bucket.MeanMS /= float64(len(durations))
		}
		result = append(result, *bucket)
	}
	return result
}

func pgPerfSampleBucket(sample pgPerfSample, bucketOf func(int) string) string {
	if sample.Bucket != "" {
		return sample.Bucket
	}
	return bucketOf(sample.Worker)
}

func (h *pgPerfHarness) copySeedRows(ctx context.Context, tx pgx.Tx, table string, columns []string, rows [][]any) error {
	for first := 0; first < len(rows); first += 1000 {
		h.seedProgress("guard-start", map[string]any{"table": table, "first": first})
		if err := h.checkDiskBudgetContext(ctx); err != nil {
			h.seedProgress("guard-failure", map[string]any{"table": table, "first": first, "error_class": pgPerfErrorClass(err)})
			return err
		}
		h.seedProgress("guard-success", map[string]any{"table": table, "first": first})
		if err := pgPerfOperationAdmit(ctx); err != nil {
			return err
		}
		n, err := tx.CopyFrom(ctx, pgx.Identifier{table}, columns, pgx.CopyFromRows(rows[first:min(first+1000, len(rows))]))
		h.seedProgress("copy-return", map[string]any{"table": table, "first": first, "requested": min(1000, len(rows)-first), "copied_not_committed": n, "error_class": pgPerfErrorClass(err)})
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *pgPerfHarness) pgPerfDatabaseSize() int64 {
	if h.resource == nil {
		return -1
	}
	requested := time.Now()
	h.resource.mu.Lock()
	observation := h.resource.last
	terminal := h.resource.terminal
	h.resource.mu.Unlock()
	if h.evidence != nil {
		h.evidence.writeJSON(filepath.Join("statistics", fmt.Sprintf("resource.%d.json", requested.UnixNano())), map[string]any{"role": "post/pre-point metric only, not a resource guard or business sample", "started_utc": observation.Started.UTC(), "finished_utc": observation.Finished.UTC(), "database_bytes": observation.Bytes, "terminal": fmt.Sprint(terminal), "state_read_wait_ns": time.Since(requested).Nanoseconds()})
	}
	if observation.Started.IsZero() {
		return -1
	}
	return observation.Bytes
}

// pgPerfPhase executes one barrier-started fixed-duration closed loop. It never
// uses sleep to infer ordering; the timer only closes the admission window.
func (h *pgPerfHarness) pgPerfPhase(duration time.Duration, workers int, collect bool, repeat int, op func(context.Context, int) (bool, error), buckets ...func(int) string) pgPerfPhaseResult {
	return h.pgPerfPhaseContext(h.ctx, duration, workers, collect, repeat, op, buckets...)
}

func (h *pgPerfHarness) pgPerfPhaseContext(parent context.Context, duration time.Duration, workers int, collect bool, repeat int, op func(context.Context, int) (bool, error), buckets ...func(int) string) pgPerfPhaseResult {
	result := pgPerfPhaseResult{}
	if workers < 1 {
		workers = 1
	}
	started := time.Now()
	deadline := started.Add(duration)
	result.startedNS = started.UnixNano()
	result.deadlineNS = deadline.UnixNano()
	ctx, cancel := context.WithDeadline(parent, deadline)
	defer cancel()
	var canceledAt atomic.Int64
	observerDone := make(chan struct{})
	h.group.Add(1)
	stopObserver := context.AfterFunc(ctx, func() { defer h.group.Done(); canceledAt.Store(time.Since(started).Nanoseconds()); close(observerDone) })
	var observerOnce sync.Once
	finishObserver := func() {
		observerOnce.Do(func() {
			if stopObserver() {
				h.group.Done()
				close(observerDone)
			} else {
				<-observerDone
			}
		})
	}
	defer finishObserver()
	var group sync.WaitGroup
	var mu sync.Mutex
	for worker := 0; worker < workers; worker++ {
		group.Add(1)
		h.group.Add(1)
		go func(worker int) {
			defer group.Done()
			defer h.group.Done()
			for iteration := int64(0); pgPerfOperationAdmit(ctx) == nil; iteration++ {
				logical := worker
				if len(buckets) != 0 && buckets[0] != nil {
					logical = pgPerfBodyClass(worker, iteration)
				}
				before := time.Now()
				state := &pgPerfOperationState{}
				operationCtx := context.WithValue(ctx, pgPerfOperationKey{}, state)
				ok, err := op(operationCtx, logical)
				finished := time.Now()
				sample := pgPerfSample{Repeat: repeat, Worker: worker, Seq: iteration,
					StartUS: before.Sub(started).Microseconds(), DurationUS: time.Since(before).Microseconds(), OK: ok && err == nil}
				if len(buckets) != 0 && buckets[0] != nil {
					sample.Bucket = buckets[0](logical)
				}
				boundary := !finished.Before(deadline) && parent.Err() == nil
				contextError := pgPerfContextOnly(err)
				state.mu.Lock()
				knownRollback := state.begun > 0 && state.committed == 0 && state.rolledBack == state.begun && !state.unknown
				state.mu.Unlock()
				readOnly, _ := parent.Value(pgPerfReadOnlyKey{}).(bool)
				cutoff := boundary && contextError && (readOnly || knownRollback || h.pool == nil)
				mu.Lock()
				if completed := finished.Sub(started).Nanoseconds(); completed > result.lastCompletionNS {
					result.lastCompletionNS = completed
				}
				switch {
				case boundary && err == nil:
					result.late++
					sample.OK = false
					sample.Error = "late_completed"
					if collect {
						result.samples = append(result.samples, sample)
					}
					result.attempts++
				case boundary && contextError && !cutoff:
					result.unknown++
					pgPerfRecordPhase(&result, sample, pgPerfErrUnknownWriteOutcome, false, collect)
				default:
					pgPerfRecordPhase(&result, sample, err, cutoff, collect)
				}
				mu.Unlock()
				if err != nil {
					return
				}
			}
		}(worker)
	}
	done := make(chan struct{})
	h.group.Add(1)
	go func() {
		defer h.group.Done()
		group.Wait()
		mu.Lock()
		result.workersJoinedNS = time.Since(started).Nanoseconds()
		mu.Unlock()
		close(done)
	}()
	joined := true
	select {
	case <-done:
	case <-ctx.Done():
		tailEnd := deadline.Add(time.Second)
		if bound, ok := parent.Value(pgPerfTailDeadlineKey{}).(time.Time); ok && bound.Before(tailEnd) {
			tailEnd = bound
		}
		if bound, ok := parent.Deadline(); ok && bound.Before(tailEnd) {
			tailEnd = bound
		}
		tailDeadline := time.NewTimer(time.Until(tailEnd))
		select {
		case <-done:
		case <-tailDeadline.C:
			select {
			case <-done:
			default:
				joined = false
			}
		}
		tailDeadline.Stop()
	}
	finishObserver()
	mu.Lock()
	defer mu.Unlock()
	result.observed = time.Since(started)
	result.canceled = parent.Err() != nil
	result.contextObservedNS = canceledAt.Load()
	result.completed = joined && !result.canceled && result.observed >= duration
	if !joined {
		result.errors++
		result.unknown++
		if h.evidence != nil {
			h.evidence.checkpointInventory()
			h.evidence.writeJSONExclusive(filepath.Join("failures", fmt.Sprintf("phase.%d.fail-stop.json", started.UnixNano())), map[string]any{"pid": os.Getpid(), "run_id": h.evidence.runID, "schema": h.schema, "phase_started_unix_ns": result.startedNS, "phase_deadline_unix_ns": result.deadlineNS, "context_done_observed_offset_ns": result.contextObservedNS, "last_completion_offset_ns": result.lastCompletionNS, "workers_joined": false, "observed_ns": result.observed.Nanoseconds(), "samples": result.samples, "unknown": true, "exit_code": 86, "reason": "phase worker exceeded adopted remaining cancellation tail; owned test child fail-stop"})
		}
		fmt.Fprintln(os.Stderr, "PGPERF owned test child fail-stop exit=86; worker join deadline exceeded; no success; inspect persisted evidence")
		os.Exit(86)
	}
	result.tail = result.observed - duration
	return result
}

func pgPerfContextOnly(err error) bool {
	if err == nil {
		return false
	}
	if err == context.Canceled || err == context.DeadlineExceeded {
		return true
	}
	if many, ok := err.(interface{ Unwrap() []error }); ok {
		children := many.Unwrap()
		if len(children) == 0 {
			return false
		}
		for _, child := range children {
			if !pgPerfContextOnly(child) {
				return false
			}
		}
		return true
	}
	if one, ok := err.(interface{ Unwrap() error }); ok {
		return pgPerfContextOnly(one.Unwrap())
	}
	return false
}

var pgPerfErrUnknownWriteOutcome = errors.New("unknown_write_outcome")

func pgPerfRecordPhase(result *pgPerfPhaseResult, sample pgPerfSample, err error, cutoff, collect bool) {
	result.attempts++
	switch {
	case cutoff:
		result.cutoffs++
		sample.OK = false
		sample.Error = "phase_deadline_cutoff"
	case err != nil:
		result.errors++
		sample.Error = pgPerfErrorClass(err)
	case sample.OK:
		result.successes++
	default:
		result.rejected++
	}
	if collect {
		result.samples = append(result.samples, sample)
	}
}

type pgPerfReadOnlyKey struct{}
type pgPerfTailDeadlineKey struct{}

// pgPerfRunPoint records one fixed-duration measurement point. Correctness-only
// scenarios must not call it because they never produce P50/P95/QPS samples.
func (h *pgPerfHarness) pgPerfRunPoint(spec pgPerfPointSpec, op func(context.Context, int) (bool, error)) pgPerfPointSummary {
	h.t.Helper()
	if err := pgPerfOperationAdmit(h.ctx); err != nil {
		h.t.Fatalf("point resource admission closed: %v", err)
	}
	if strings.TrimSpace(spec.Point) == "" {
		h.t.Fatal("pgperf point name is required")
	}
	mode := spec.Mode
	if mode == "" {
		mode = pgPerfModeTimed3Rep
	}
	window := spec.Window
	if window == 0 {
		window = pgPerfMinWindow
	}
	if window < pgPerfMinWindow {
		h.t.Fatalf("pgperf point %s window %s is below %s", spec.Point, window, pgPerfMinWindow)
	}
	warmup := spec.Warmup
	if warmup == 0 {
		warmup = pgPerfWarmupFor(window)
	}
	if warmup < pgPerfMinWarmup {
		h.t.Fatalf("pgperf point %s warm-up %s is below %s", spec.Point, warmup, pgPerfMinWarmup)
	}
	repeats := spec.Repeats
	if repeats == 0 {
		if mode == pgPerfModeSingleRun {
			repeats = pgPerfSingleRepeats
		} else {
			repeats = pgPerfTimedRepeats
		}
	}
	cap := pgPerfTimedPointCap
	if mode == pgPerfModeSingleRun {
		cap = pgPerfSinglePointCap
	}
	if total := pgPerfPointDuration(warmup, window, repeats); total > cap {
		h.t.Fatalf("pgperf point %s budget %s exceeds %s", spec.Point, total, cap)
	}
	policy := h.policy
	wallCap := time.Duration(policy.TimedWallSeconds) * time.Second
	if mode == pgPerfModeSingleRun {
		wallCap = time.Duration(policy.SingleWallSeconds) * time.Second
	}
	if mode == pgPerfModeSingleRun && pgPerfPointDuration(warmup, window, repeats) > wallCap-time.Duration(policy.PointTailSeconds)*time.Second {
		h.t.Fatal("SINGLE nominal activity must reserve the adopted bounded tail inside 60s")
	}
	summary := pgPerfPointSummary{
		Point: spec.Point, Scene: spec.Scene, Tier: h.tier.name, Schema: h.schema,
		Mode: mode, Concurrency: spec.Concurrency, Repeats: repeats,
		WarmupMS: warmup.Milliseconds(), WindowMS: window.Milliseconds(),
		MaxConns: h.pool.Stat().MaxConns(), Note: spec.Note,
	}
	if summary.Concurrency < 1 {
		summary.Concurrency = 1
	}
	startPool := h.pool.Stat()
	startDB := h.pgPerfDatabaseSize()
	var samples []pgPerfSample
	observed := time.Duration(0)
	active := time.Duration(0)
	pointStarted := time.Now()
	pointCtx, pointCancel := context.WithTimeout(h.ctx, wallCap)
	defer pointCancel()
	pointCtx = context.WithValue(pointCtx, pgPerfReadOnlyKey{}, (strings.HasPrefix(spec.Scene, "R") && spec.Scene != "R05") || spec.Scene == "X06" || spec.Scene == "X07")
	for repeat := 0; repeat < repeats; repeat++ {
		if err := h.ctx.Err(); err != nil {
			summary.Truncated = true
			summary.Note = strings.TrimSpace(summary.Note + " NOT RUN: tier context exhausted")
			h.evidence.addBlocked(spec.Point + ": NOT RUN, tier context exhausted before repeat " + strconv.Itoa(repeat+1))
			break
		}
		warmupWorkers := summary.Concurrency
		if spec.WarmupConcurrency > 0 {
			warmupWorkers = spec.WarmupConcurrency
		}
		warmupCtx := context.WithValue(pointCtx, pgPerfTailDeadlineKey{}, pointStarted.Add(time.Duration(repeat)*(warmup+window)+warmup+time.Second))
		warmupPhase := h.pgPerfPhaseContext(warmupCtx, warmup, warmupWorkers, false, repeat, op, spec.BucketFn)
		active += warmupPhase.observed
		summary.TailMS += warmupPhase.tail.Milliseconds()
		h.evidence.writeJSON(filepath.Join("repetitions", fmt.Sprintf("%s.%s.%d.warmup.json", spec.Point, h.evidence.runID, repeat)), map[string]any{
			"repeat": repeat, "role": "warmup", "observed_ms": warmupPhase.observed.Milliseconds(), "errors": warmupPhase.errors,
			"started_unix_ns": warmupPhase.startedNS, "admission_deadline_unix_ns": warmupPhase.deadlineNS, "context_done_observed_offset_ns": warmupPhase.contextObservedNS, "last_completion_offset_ns": warmupPhase.lastCompletionNS, "all_workers_joined_offset_ns": warmupPhase.workersJoinedNS,
			"nominal_ns": warmup.Nanoseconds(), "observed_ns": warmupPhase.observed.Nanoseconds(), "tail_ns": warmupPhase.tail.Nanoseconds(),
			"successes": warmupPhase.successes, "rejected": warmupPhase.rejected, "late": warmupPhase.late, "unknown": warmupPhase.unknown,
			"completed": warmupPhase.completed, "canceled": warmupPhase.canceled, "concurrency": warmupWorkers,
			"attempts": warmupPhase.attempts, "cutoffs": warmupPhase.cutoffs, "tail_ms": warmupPhase.tail.Milliseconds(),
		})
		if phase := warmupPhase; phase.errors != 0 {
			summary.Errors += phase.errors
			summary.Truncated = true
			break
		}
		if err := pointCtx.Err(); err != nil || !warmupPhase.completed {
			summary.Truncated = true
			summary.Note = strings.TrimSpace(summary.Note + " NOT RUN: tier context exhausted after warm-up")
			h.evidence.addBlocked(spec.Point + ": NOT RUN, tier context exhausted after warm-up in repeat " + strconv.Itoa(repeat+1))
			break
		}
		measureCtx := context.WithValue(pointCtx, pgPerfTailDeadlineKey{}, pointStarted.Add(time.Duration(repeat+1)*(warmup+window)+time.Second))
		phase := h.pgPerfPhaseContext(measureCtx, window, summary.Concurrency, true, repeat, op, spec.BucketFn)
		active += phase.observed
		summary.TailMS += phase.tail.Milliseconds()
		h.evidence.writeJSON(filepath.Join("repetitions", fmt.Sprintf("%s.%s.%d.measurement.json", spec.Point, h.evidence.runID, repeat)), map[string]any{
			"repeat": repeat, "role": "measurement", "observed_ms": phase.observed.Milliseconds(),
			"started_unix_ns": phase.startedNS, "admission_deadline_unix_ns": phase.deadlineNS, "context_done_observed_offset_ns": phase.contextObservedNS, "last_completion_offset_ns": phase.lastCompletionNS, "all_workers_joined_offset_ns": phase.workersJoinedNS,
			"nominal_ns": window.Nanoseconds(), "observed_ns": phase.observed.Nanoseconds(), "tail_ns": phase.tail.Nanoseconds(), "late": phase.late, "unknown": phase.unknown,
			"attempts": phase.attempts, "successes": phase.successes, "errors": phase.errors, "rejected": phase.rejected,
			"completed": phase.completed, "canceled": phase.canceled, "cutoffs": phase.cutoffs, "tail_ms": phase.tail.Milliseconds(),
		})
		for index := range phase.samples {
			phase.samples[index].Point = spec.Point
		}
		samples = append(samples, phase.samples...)
		summary.Attempts += phase.attempts
		summary.Successes += phase.successes
		summary.Errors += phase.errors
		summary.Rejected += phase.rejected
		summary.BoundaryCanceled += phase.cutoffs
		summary.Late += phase.late
		summary.Unknown += phase.unknown
		observed += phase.observed
		if phase.completed && phase.errors == 0 {
			summary.CompletedRepeats++
		} else {
			summary.Truncated = true
			break
		}
	}
	pointWall := time.Since(pointStarted)
	totalTail := pointWall - pgPerfPointDuration(warmup, window, repeats)
	endPool := h.pool.Stat()
	endDB := h.pgPerfDatabaseSize()
	summary.ObservedMS = observed.Milliseconds()
	summary.ActiveMS = active.Milliseconds()
	summary.PointWallNS = pointWall.Nanoseconds()
	summary.TailNS = totalTail.Nanoseconds()
	summary.N = len(samples)
	summary.P50MS, summary.P95MS, summary.P99MS = 0, 0, 0
	summary.MinMS, summary.MaxMS, summary.MeanMS = 0, 0, 0
	if summary.N > 0 {
		durations := make([]float64, 0, summary.N)
		for _, sample := range samples {
			if sample.OK {
				durations = append(durations, float64(sample.DurationUS)/1000)
			}
		}
		if len(durations) > 0 {
			sort.Float64s(durations)
			summary.N = len(durations)
			summary.P50MS = pgPerfPercentile(durations, 50)
			summary.P95MS = pgPerfPercentile(durations, 95)
			summary.P99MS = pgPerfPercentile(durations, 99)
			summary.MinMS = durations[0]
			summary.MaxMS = durations[len(durations)-1]
			for _, duration := range durations {
				summary.MeanMS += duration
			}
			summary.MeanMS /= float64(len(durations))
		} else {
			summary.N = 0
		}
	}
	if spec.BucketFn != nil {
		summary.Buckets = pgPerfBucketSummaries(samples, spec.BucketFn)
	}
	if observed > 0 {
		summary.QPS = float64(summary.Successes) / observed.Seconds()
	}
	summary.AcquireCount = endPool.AcquireCount() - startPool.AcquireCount()
	summary.AcquireMS = float64(endPool.AcquireDuration()-startPool.AcquireDuration()) / float64(time.Millisecond)
	summary.EmptyAcquire = endPool.EmptyAcquireCount() - startPool.EmptyAcquireCount()
	summary.CancelAcquire = endPool.CanceledAcquireCount() - startPool.CanceledAcquireCount()
	summary.DBBytesDelta = endDB - startDB
	summary.Exploratory = mode == pgPerfModeSingleRun || summary.N < pgPerfSampleSoftTarget
	if summary.BoundaryCanceled+summary.Late+summary.Unknown > 0 {
		summary.Exploratory = true
		summary.Note += " conditional quantiles of on-time completed samples; boundary censoring present; exploratory"
	}
	if summary.Exploratory && !strings.Contains(summary.Note, "exploratory") {
		if mode == pgPerfModeSingleRun {
			summary.Note = strings.TrimSpace(summary.Note + " exploratory (SINGLE-RUN)")
		}
		if summary.N < pgPerfSampleSoftTarget {
			summary.Note = strings.TrimSpace(summary.Note + " exploratory (N<500)")
		}
	}
	if len(samples) > 0 {
		h.evidence.writeRaw(spec.Point, samples)
	}
	h.evidence.addPoint(summary)
	if err := pgPerfPointOutcome(summary, repeats, pointWall, wallCap); err != nil {
		h.evidence.stopReason = spec.Point
		h.t.Fatalf("NONCONFORMING point %s: %v", spec.Point, err)
	}
	h.t.Logf("pgperf_point point=%s tier=%s mode=%s c=%d repeats=%d n=%d p50=%.3fms p95=%.3fms p99=%.3fms qps=%.3f errors=%d rejected=%d observed=%dms",
		summary.Point, summary.Tier, summary.Mode, summary.Concurrency, summary.Repeats, summary.N,
		summary.P50MS, summary.P95MS, summary.P99MS, summary.QPS, summary.Errors, summary.Rejected, summary.ObservedMS)
	return summary
}

func pgPerfPointOutcome(summary pgPerfPointSummary, expected int, active, cap time.Duration) error {
	if summary.Errors != 0 {
		return fmt.Errorf("execution/correctness errors=%d", summary.Errors)
	}
	if summary.Truncated || summary.CompletedRepeats != expected {
		return fmt.Errorf("incomplete repeats=%d expected=%d truncated=%t", summary.CompletedRepeats, expected, summary.Truncated)
	}
	if summary.Successes <= 0 || summary.N <= 0 {
		return errors.New("no valid on-time successful samples; no performance conclusion")
	}
	if active > cap {
		return fmt.Errorf("actual active duration %s exceeds %s", active, cap)
	}
	if summary.TailNS > int64(time.Second) {
		return fmt.Errorf("cumulative point tail %s exceeds adopted 1s", time.Duration(summary.TailNS))
	}
	return nil
}

func TestPgPerfSmall(t *testing.T) {
	evidence := pgPerfGlobalEvidence(t)
	t.Cleanup(func() { evidence.Finalize(t) })
	runPgPerfSmallWrite(t)
	runPgPerfSmallRead(t)
	runPgPerfSmallNoticeDelete(t)
	pgPerfGlobalEvidence(t).Finalize(t)
}

func pgPerfPointDuration(warmup, window time.Duration, repeats int) time.Duration {
	return time.Duration(repeats) * (warmup + window)
}

type pgPerfBaselineRecord struct {
	Window string
	Tier   string
	Bytes  int64
}
type pgPerfRegistrationReceipt struct {
	Path                     string
	Window                   string
	Tier                     string
	Bytes                    int64
	FileBytes                int64
	SHA256                   string
	AttemptID                string
	ExecutionWindow          string
	RegisteredAt             string
	FirstRegistrationAllowed bool
	ParentManifestSHA256     string
}

func pgPerfValidateResource(resource pgPerfResourceBinding, expected string) error {
	if resource.BaseBytes != pgPerfResourceBase || resource.Window != pgPerfResourceOrigin || len(resource.Baselines) != 3 || resource.AttemptID == "" || len(resource.ManifestSHA256) != 64 {
		return errors.New("resource binding incomplete")
	}
	if err := pgPerfSamePath(resource.Directory, expected, true); err != nil {
		return err
	}
	entries, err := os.ReadDir(resource.Directory)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || (entry.Name() != "small.json" && entry.Name() != "medium.json" && entry.Name() != "large.json" && entry.Name() != "medium.registration.json") {
			return errors.New("unknown resource ledger entry")
		}
	}
	for _, tier := range []string{"small", "medium", "large"} {
		binding, ok := resource.Baselines[tier]
		if !ok || binding.File != tier+".json" || binding.Window != resource.Window || binding.Tier != tier {
			return errors.New("resource manifest tier/origin/path mismatch")
		}
		if binding.State == "existing" {
			if binding.AllowFirst {
				return errors.New("existing ledger cannot permit first registration")
			}
			if !resource.Legacy && tier == "medium" && binding.ReceiptSHA256 == "" {
				return errors.New("existing medium registration receipt must be bound")
			}
			if _, err := pgPerfReadBaseline(resource, tier, binding); err != nil {
				return err
			}
		} else if binding.State == "unregistered" && tier != "small" {
			if binding.Bytes != 0 || binding.FileBytes != 0 || binding.SHA256 != "" || binding.ReceiptSHA256 != "" || (binding.AllowFirst && tier != "medium") {
				return errors.New("invalid first-registration manifest")
			}
			if _, err := os.Lstat(filepath.Join(resource.Directory, binding.File)); err == nil {
				if _, err := pgPerfReadBaseline(resource, tier, binding); err != nil {
					return err
				}
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			} else if _, err := os.Lstat(filepath.Join(resource.Directory, tier+".registration.json")); !errors.Is(err, os.ErrNotExist) {
				return errors.New("orphan/partial registration receipt")
			}
		} else {
			return errors.New("unknown/invalid resource manifest state")
		}
	}
	return nil
}

type pgPerfLoadedBaseline struct {
	Path, Window, Tier                                  string
	Bytes, FileBytes                                    int64
	SHA256                                              string
	ReceiptPath, ReceiptSHA256                          string
	ReceiptFileBytes                                    int64
	AuthorizationState, LoadState, SourceManifestSHA256 string
}

func pgPerfLoadTierBaseline(resource pgPerfResourceBinding, tier string, current int64) (pgPerfLoadedBaseline, error) {
	var actual pgPerfLoadedBaseline
	_, beforeErr := os.Lstat(filepath.Join(resource.Directory, tier+".json"))
	baseline, err := pgPerfTierBaseline(resource, tier, current)
	if err != nil {
		return actual, err
	}
	actual, err = pgPerfReadBaselineActual(resource, tier, resource.Baselines[tier])
	if err != nil {
		return actual, err
	}
	if actual.Bytes != baseline {
		return actual, errors.New("loaded baseline changed between validation and fingerprint")
	}
	actual.LoadState = "existing_file_reuse"
	if errors.Is(beforeErr, os.ErrNotExist) {
		actual.LoadState = "first_registration"
	}
	return actual, nil
}

func pgPerfReadBaseline(resource pgPerfResourceBinding, tier string, binding pgPerfBaselineBinding) (int64, error) {
	loaded, err := pgPerfReadBaselineActual(resource, tier, binding)
	return loaded.Bytes, err
}

func pgPerfReadBaselineActual(resource pgPerfResourceBinding, tier string, binding pgPerfBaselineBinding) (loaded pgPerfLoadedBaseline, err error) {
	path := filepath.Join(resource.Directory, binding.File)
	if err := pgPerfSamePath(path, path, false); err != nil {
		return loaded, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return loaded, err
	}
	var prior pgPerfBaselineRecord
	if err = json.Unmarshal(data, &prior); err != nil {
		return loaded, err
	}
	if prior.Window != resource.Window || prior.Tier != tier || prior.Bytes <= 0 {
		return loaded, errors.New("persistent resource origin/tier/bytes mismatch")
	}
	sum := sha256.Sum256(data)
	actualHash := hex.EncodeToString(sum[:])
	if binding.State == "existing" && (binding.Bytes != prior.Bytes || binding.FileBytes != int64(len(data)) || !strings.EqualFold(binding.SHA256, actualHash)) {
		return loaded, errors.New("bound ledger length/hash/bytes mismatch")
	}
	if tier == "small" && (prior.Bytes != pgPerfResourceBase || len(data) != 80 || !strings.EqualFold(actualHash, "2B368282C7667FC2B3563F82FC293FBBEEE59A350A52CB41D08C57BDB87FC4D2")) {
		return loaded, errors.New("original small ledger changed")
	}
	receiptPath := filepath.Join(resource.Directory, tier+".registration.json")
	_, receiptErr := os.Lstat(receiptPath)
	if binding.State == "unregistered" || binding.ReceiptSHA256 != "" || receiptErr == nil {
		receiptHash, receiptBytes, verifyErr := pgPerfVerifyReceipt(resource, tier, binding, data, prior, receiptPath)
		if verifyErr != nil {
			return loaded, verifyErr
		}
		loaded.ReceiptPath = receiptPath
		loaded.ReceiptSHA256 = receiptHash
		loaded.ReceiptFileBytes = receiptBytes
	} else if !errors.Is(receiptErr, os.ErrNotExist) {
		return loaded, receiptErr
	}
	loaded.Path = path
	loaded.Window = prior.Window
	loaded.Tier = tier
	loaded.Bytes = prior.Bytes
	loaded.FileBytes = int64(len(data))
	loaded.SHA256 = actualHash
	loaded.AuthorizationState = binding.State
	loaded.SourceManifestSHA256 = resource.ManifestSHA256
	return loaded, nil
}

func pgPerfVerifyReceipt(resource pgPerfResourceBinding, tier string, binding pgPerfBaselineBinding, data []byte, prior pgPerfBaselineRecord, path string) (string, int64, error) {
	if err := pgPerfSamePath(path, path, false); err != nil {
		return "", 0, err
	}
	encoded, err := os.ReadFile(path)
	if err != nil {
		return "", 0, err
	}
	var receipt pgPerfRegistrationReceipt
	if err = json.Unmarshal(encoded, &receipt); err != nil {
		return "", 0, err
	}
	sum := sha256.Sum256(data)
	if receipt.Path != filepath.Join(resource.Directory, binding.File) || receipt.Window != resource.Window || receipt.Tier != tier || receipt.Bytes != prior.Bytes || receipt.FileBytes != int64(len(data)) || !strings.EqualFold(receipt.SHA256, hex.EncodeToString(sum[:])) || !receipt.FirstRegistrationAllowed || receipt.AttemptID == "" || receipt.ParentManifestSHA256 == "" {
		return "", 0, errors.New("registration receipt identity/content mismatch")
	}
	if _, err = time.Parse(time.RFC3339Nano, receipt.ExecutionWindow); err != nil {
		return "", 0, err
	}
	if _, err = time.Parse(time.RFC3339Nano, receipt.RegisteredAt); err != nil {
		return "", 0, err
	}
	if binding.State == "unregistered" {
		if !binding.AllowFirst || receipt.ParentManifestSHA256 != resource.ManifestSHA256 || receipt.AttemptID != resource.AttemptID || receipt.ExecutionWindow != resource.ExecutionWindow {
			return "", 0, errors.New("first-registration receipt permission/parent/attempt mismatch")
		}
	} else {
		sum = sha256.Sum256(encoded)
		if binding.ReceiptSHA256 == "" || !strings.EqualFold(binding.ReceiptSHA256, hex.EncodeToString(sum[:])) {
			return "", 0, errors.New("existing registration receipt is not bound")
		}
	}
	sum = sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), int64(len(encoded)), nil
}

func pgPerfTierBaseline(resource pgPerfResourceBinding, tier string, current int64) (int64, error) {
	if current <= 0 {
		return 0, errors.New("invalid current database bytes")
	}
	if err := pgPerfValidateResource(resource, resource.Directory); err != nil {
		return 0, err
	}
	binding, ok := resource.Baselines[tier]
	if !ok {
		return 0, errors.New("unbound tier")
	}
	path := filepath.Join(resource.Directory, binding.File)
	if _, err := os.Lstat(path); err == nil {
		return pgPerfReadBaseline(resource, tier, binding)
	} else if !errors.Is(err, os.ErrNotExist) {
		return 0, err
	}
	if tier != "medium" || binding.State != "unregistered" || !binding.AllowFirst {
		return 0, errors.New("first registration not permitted")
	}
	data, err := json.Marshal(pgPerfBaselineRecord{resource.Window, tier, current})
	if err != nil {
		return 0, err
	}
	data = append(data, '\n')
	if err = pgPerfWriteExclusive(path, data); err != nil {
		return 0, err
	}
	sum := sha256.Sum256(data)
	receipt := pgPerfRegistrationReceipt{Path: path, Window: resource.Window, Tier: tier, Bytes: current, FileBytes: int64(len(data)), SHA256: hex.EncodeToString(sum[:]), AttemptID: resource.AttemptID, ExecutionWindow: resource.ExecutionWindow, RegisteredAt: time.Now().UTC().Format(time.RFC3339Nano), FirstRegistrationAllowed: true, ParentManifestSHA256: resource.ManifestSHA256}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		return 0, err
	}
	if err = pgPerfWriteExclusive(filepath.Join(resource.Directory, "medium.registration.json"), append(encoded, '\n')); err != nil {
		return 0, err
	}
	return pgPerfReadBaseline(resource, tier, binding)
}

func pgPerfWriteExclusive(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	n, writeErr := file.Write(data)
	if n != len(data) && writeErr == nil {
		writeErr = io.ErrShortWrite
	}
	return errors.Join(writeErr, file.Close())
}

func TestPgPerfR02_MediumRetry_KeepsTierBaseline(t *testing.T) {
	if encoded := os.Getenv("HAN_PG_PERF_OFFLINE_CHILD_INPUT"); encoded != "" {
		var input struct {
			Policy pgPerfExecutionPolicy
			Inputs pgPerfPolicyInputs
		}
		if err := json.Unmarshal([]byte(encoded), &input); err != nil {
			t.Fatal(err)
		}
		input.Inputs.ExecutionWindow = os.Getenv(pgPerfWindowStartEnv)
		input.Inputs.BaseBytes = os.Getenv(pgPerfBaseBytesEnv)
		input.Inputs.BaselineDirectory = os.Getenv(pgPerfBaselineDirEnv)
		data, err := json.Marshal(input.Policy)
		if err != nil {
			t.Fatal(err)
		}
		policy, err := pgPerfParsePolicy(data, input.Inputs)
		if err != nil {
			t.Fatal(err)
		}
		baseline, err := pgPerfTierBaseline(policy.Resource, "medium", pgPerfResourceBase+300)
		if err != nil || baseline != pgPerfResourceBase+100 || pgPerfResourceBase+300-baseline <= 150 {
			t.Fatalf("child retry reset medium cap: baseline=%d err=%v", baseline, err)
		}
		t.Logf("independent_child_pid=%d persisted_medium_baseline=%d tier_delta=200 tier_cap=150 total_delta=280 total_cap=1000", os.Getpid(), baseline)
		return
	}
	policy, input := pgPerfR04Fixture(t)
	parsed := pgPerfR04Parse(t, policy, input)
	if baseline, err := pgPerfTierBaseline(parsed.Resource, "medium", pgPerfResourceBase+100); err != nil || baseline != pgPerfResourceBase+100 {
		t.Fatalf("initial baseline=%d err=%v", baseline, err)
	}
	baseline, err := pgPerfTierBaseline(parsed.Resource, "medium", pgPerfResourceBase+300)
	if err != nil || baseline != pgPerfResourceBase+100 {
		t.Fatalf("retry reset tier baseline=%d err=%v", baseline, err)
	}
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, binary, "-test.run=^TestPgPerfR02_MediumRetry_KeepsTierBaseline$", "-test.v")
	encoded, err := json.Marshal(struct {
		Policy pgPerfExecutionPolicy
		Inputs pgPerfPolicyInputs
	}{policy, input})
	if err != nil {
		t.Fatal(err)
	}
	child.Env = pgPerfChildEnvironment(os.Environ(), map[string]string{
		"HAN_PG_PERF_OFFLINE_CHILD_INPUT": string(encoded),
		pgPerfWindowStartEnv:              policy.WindowStart, pgPerfBaseBytesEnv: strconv.FormatInt(policy.ResourceBaseBytes, 10), pgPerfBaselineDirEnv: policy.ResourceDirectory,
	})
	output, err := child.CombinedOutput()
	if err != nil {
		t.Fatalf("baseline retry child failed: %v %s", err, output)
	}
	t.Logf("%s", output)
	for key, marker := range map[string]string{pgPerfBaseBytesEnv: "resource base environment cannot override policy", pgPerfWindowStartEnv: "invalid/future/expired execution window"} {
		invalid := exec.CommandContext(ctx, binary, "-test.run=^TestPgPerfR02_MediumRetry_KeepsTierBaseline$", "-test.v")
		invalid.Env = pgPerfChildEnvironment(child.Env, map[string]string{key: "1"})
		output, err := invalid.CombinedOutput()
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 1 || !strings.Contains(string(output), marker) {
			t.Fatalf("child mismatch was not rejected at policy gate: key=%s err=%v output=%s", key, err, output)
		}
		t.Logf("expected child rejection key=%s; actual wait exit=1\n%s", key, output)
	}
}

func TestPgPerfR04_NewExecution_LoadsOldResource(t *testing.T) {
	policy, input := pgPerfR04Fixture(t)
	input.ExecutionWindow = "2026-11-04T06:00:00+00:00"
	parsed := pgPerfR04Parse(t, policy, input)
	got, err := pgPerfTierBaseline(parsed.Resource, "small", 1992988351)
	if err != nil || got != 1406932671 {
		t.Fatalf("new execution must retain old resource origin: bytes=%d err=%v", got, err)
	}
	if parsed.ExecutionStart.Equal(time.Date(2026, 11, 4, 3, 29, 4, 586115000, time.UTC)) || parsed.Resource.Window != pgPerfResourceOrigin {
		t.Fatal("execution/resource clocks were conflated")
	}
}

func pgPerfChildEnvironment(base []string, overrides map[string]string) []string {
	var result []string
	seen := map[string]bool{}
	for _, value := range base {
		key, _, ok := strings.Cut(value, "=")
		if !ok {
			continue
		}
		normalized := strings.ToUpper(key)
		replaced := false
		for override := range overrides {
			if strings.EqualFold(key, override) {
				replaced = true
				break
			}
		}
		if !replaced && !seen[normalized] {
			result = append(result, value)
			seen[normalized] = true
		}
	}
	keys := make([]string, 0, len(overrides))
	for key := range overrides {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		result = append(result, key+"="+overrides[key])
	}
	return result
}

func pgPerfR04Fixture(t *testing.T) (pgPerfExecutionPolicy, pgPerfPolicyInputs) {
	t.Helper()
	parent, err := filepath.Abs(filepath.Join("..", "..", "..", ".spec", "wip", "article-management", "evidence-pg-perf-r04", "continuation-20261104", "author", "semantics-seed-budget-r10", "fixtures"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	root, err := os.MkdirTemp(parent, "offline-")
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, "ledgers")
	if err = os.Mkdir(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	old := []byte("{\"Window\":\"2026-11-04T03:29:04.586115+00:00\",\"Tier\":\"small\",\"Bytes\":1406932671}\n")
	for name, data := range map[string][]byte{"ledgers/small.json": old, "global.json": []byte(`{"database_size_base":1406932671}`), "adoption.md": []byte("offline adoption fixture; not a real execution authorization\n")} {
		if err = pgPerfWriteExclusive(filepath.Join(root, name), data); err != nil {
			t.Fatal(err)
		}
	}
	adoption, err := os.ReadFile(filepath.Join(root, "adoption.md"))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(adoption)
	policy := pgPerfExecutionPolicy{Revision: "followup-r04-offline-fixture", AttemptID: "offline-not-a-real-attempt", ParentOldBatch: pgPerfLegacyRevision, AdoptionSource: filepath.Join(root, "adoption.md"), AdoptionSHA256: hex.EncodeToString(sum[:]), WindowStart: "2026-11-04T06:00:00Z", StopNewSeconds: 8700, HardStopSeconds: 9000, TimedWallSeconds: 121, SingleWallSeconds: 60, PointTailSeconds: 1, ResourceWindow: json.RawMessage(`"` + pgPerfResourceOrigin + `"`), ResourceBaseBytes: pgPerfResourceBase, ResourceDirectory: directory, ResourceBaselines: map[string]pgPerfBaselineBinding{}}
	for _, tier := range []string{"small", "medium", "large"} {
		policy.ResourceBaselines[tier] = pgPerfBaselineBinding{File: tier + ".json", State: "unregistered", Window: pgPerfResourceOrigin, Tier: tier, AllowFirst: tier == "medium"}
	}
	sum = sha256.Sum256(old)
	small := policy.ResourceBaselines["small"]
	small.State = "existing"
	small.Bytes = pgPerfResourceBase
	small.FileBytes = int64(len(old))
	small.SHA256 = hex.EncodeToString(sum[:])
	policy.ResourceBaselines["small"] = small
	start, _ := time.Parse(time.RFC3339Nano, policy.WindowStart)
	return policy, pgPerfPolicyInputs{Now: start.Add(time.Minute), ExecutionWindow: policy.WindowStart, ExpectedDirectory: directory, GlobalBaseFile: filepath.Join(root, "global.json"), AdoptionFile: policy.AdoptionSource}
}

func pgPerfR04Parse(t *testing.T, policy pgPerfExecutionPolicy, input pgPerfPolicyInputs) pgPerfExecutionPolicy {
	t.Helper()
	data, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := pgPerfParsePolicy(data, input)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func TestPgPerfR04_PolicyRejectsInvalidBindings(t *testing.T) {
	type mutation func(*pgPerfExecutionPolicy, *pgPerfPolicyInputs)
	changeSmall := func(p *pgPerfExecutionPolicy, change func(*pgPerfBaselineBinding)) {
		binding := p.ResourceBaselines["small"]
		change(&binding)
		p.ResourceBaselines["small"] = binding
	}
	cases := map[string]mutation{
		"origin-null":  func(p *pgPerfExecutionPolicy, _ *pgPerfPolicyInputs) { p.ResourceWindow = json.RawMessage(`null`) },
		"origin-empty": func(p *pgPerfExecutionPolicy, _ *pgPerfPolicyInputs) { p.ResourceWindow = json.RawMessage(`""`) },
		"origin-equivalent-Z": func(p *pgPerfExecutionPolicy, _ *pgPerfPolicyInputs) {
			p.ResourceWindow = json.RawMessage(`"2026-11-04T03:29:04.586115Z"`)
		},
		"new-attempt-missing-origin": func(p *pgPerfExecutionPolicy, _ *pgPerfPolicyInputs) { p.ResourceWindow = nil },
		"attempt-missing":            func(p *pgPerfExecutionPolicy, _ *pgPerfPolicyInputs) { p.AttemptID = "" },
		"parent-missing":             func(p *pgPerfExecutionPolicy, _ *pgPerfPolicyInputs) { p.ParentOldBatch = "" },
		"adoption-missing":           func(p *pgPerfExecutionPolicy, _ *pgPerfPolicyInputs) { p.AdoptionSource = "" },
		"adoption-hash":              func(p *pgPerfExecutionPolicy, _ *pgPerfPolicyInputs) { p.AdoptionSHA256 = strings.Repeat("0", 64) },
		"adoption-copy":              func(p *pgPerfExecutionPolicy, i *pgPerfPolicyInputs) { p.AdoptionSource = i.GlobalBaseFile },
		"execution-missing":          func(_ *pgPerfExecutionPolicy, i *pgPerfPolicyInputs) { i.ExecutionWindow = "" },
		"execution-env-conflict":     func(_ *pgPerfExecutionPolicy, i *pgPerfPolicyInputs) { i.ExecutionWindow = pgPerfResourceOrigin },
		"execution-future":           func(_ *pgPerfExecutionPolicy, i *pgPerfPolicyInputs) { i.Now = i.Now.Add(-2 * time.Minute) },
		"execution-expired":          func(_ *pgPerfExecutionPolicy, i *pgPerfPolicyInputs) { i.Now = i.Now.Add(8700 * time.Second) },
		"limit-override":             func(p *pgPerfExecutionPolicy, _ *pgPerfPolicyInputs) { p.StopNewSeconds = 9000 },
		"base-policy":                func(p *pgPerfExecutionPolicy, _ *pgPerfPolicyInputs) { p.ResourceBaseBytes++ },
		"base-env":                   func(_ *pgPerfExecutionPolicy, i *pgPerfPolicyInputs) { i.BaseBytes = "1" },
		"base-env-empty":             func(_ *pgPerfExecutionPolicy, i *pgPerfPolicyInputs) { i.BaseBytesSet = true },
		"directory-env-empty":        func(_ *pgPerfExecutionPolicy, i *pgPerfPolicyInputs) { i.BaselineDirectorySet = true },
		"directory-relative":         func(p *pgPerfExecutionPolicy, _ *pgPerfPolicyInputs) { p.ResourceDirectory = "." },
		"directory-copy": func(p *pgPerfExecutionPolicy, i *pgPerfPolicyInputs) {
			p.ResourceDirectory = filepath.Dir(i.ExpectedDirectory)
		},
		"directory-env-override": func(_ *pgPerfExecutionPolicy, i *pgPerfPolicyInputs) {
			i.BaselineDirectory = filepath.Dir(i.ExpectedDirectory)
		},
		"tier-omitted": func(p *pgPerfExecutionPolicy, _ *pgPerfPolicyInputs) { delete(p.ResourceBaselines, "large") },
		"tier-extra": func(p *pgPerfExecutionPolicy, _ *pgPerfPolicyInputs) {
			p.ResourceBaselines["other"] = p.ResourceBaselines["small"]
		},
		"small-sha": func(p *pgPerfExecutionPolicy, _ *pgPerfPolicyInputs) {
			changeSmall(p, func(b *pgPerfBaselineBinding) { b.SHA256 = strings.Repeat("0", 64) })
		},
		"small-length": func(p *pgPerfExecutionPolicy, _ *pgPerfPolicyInputs) {
			changeSmall(p, func(b *pgPerfBaselineBinding) { b.FileBytes++ })
		},
		"small-bytes": func(p *pgPerfExecutionPolicy, _ *pgPerfPolicyInputs) {
			changeSmall(p, func(b *pgPerfBaselineBinding) { b.Bytes++ })
		},
		"small-tier": func(p *pgPerfExecutionPolicy, _ *pgPerfPolicyInputs) {
			changeSmall(p, func(b *pgPerfBaselineBinding) { b.Tier = "medium" })
		},
		"small-window": func(p *pgPerfExecutionPolicy, _ *pgPerfPolicyInputs) {
			changeSmall(p, func(b *pgPerfBaselineBinding) { b.Window = p.WindowStart })
		},
		"small-missing": func(p *pgPerfExecutionPolicy, _ *pgPerfPolicyInputs) {
			changeSmall(p, func(b *pgPerfBaselineBinding) { b.State = "unregistered" })
		},
		"path-escape": func(p *pgPerfExecutionPolicy, _ *pgPerfPolicyInputs) {
			changeSmall(p, func(b *pgPerfBaselineBinding) { b.File = "../small.json" })
		},
		"global-source-changed": func(_ *pgPerfExecutionPolicy, i *pgPerfPolicyInputs) { i.GlobalBaseFile = i.AdoptionFile },
		"large-first-registration": func(p *pgPerfExecutionPolicy, _ *pgPerfPolicyInputs) {
			b := p.ResourceBaselines["large"]
			b.AllowFirst = true
			p.ResourceBaselines["large"] = b
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			p, i := pgPerfR04Fixture(t)
			mutate(&p, &i)
			data, err := json.Marshal(p)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = pgPerfParsePolicy(data, i); err == nil {
				t.Fatal("invalid binding admitted")
			}
			t.Logf("rejected: %v", err)
		})
	}
}

func TestPgPerfR04_LegacyMissingOnly_ExpiredStillRejected(t *testing.T) {
	p, i := pgPerfR04Fixture(t)
	p.Revision = pgPerfLegacyRevision
	p.AttemptID = ""
	p.ParentOldBatch = ""
	p.AdoptionSource = "../../lead/execution-policy-r02-adoption.md"
	p.AdoptionSHA256 = ""
	p.WindowStart = pgPerfResourceOrigin
	p.ResourceWindow = nil
	p.ResourceBaseBytes = 0
	p.ResourceDirectory = ""
	p.ResourceBaselines = nil
	i.ExecutionWindow = p.WindowStart
	start, _ := time.Parse(time.RFC3339Nano, p.WindowStart)
	i.Now = start.Add(time.Minute)
	parsed := pgPerfR04Parse(t, p, i)
	if !parsed.Resource.Legacy || parsed.Resource.Window != pgPerfResourceOrigin {
		t.Fatal("legacy original literal not preserved")
	}
	if baseline, err := pgPerfTierBaseline(parsed.Resource, "small", 1992988351); err != nil || baseline != pgPerfResourceBase {
		t.Fatalf("legacy load: %d %v", baseline, err)
	}
	encoded, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"resource_base_bytes", "resource_baseline_dir", "resource_baselines"} {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(encoded, &fields); err != nil {
			t.Fatal(err)
		}
		fields[key] = json.RawMessage(`null`)
		data, err := json.Marshal(fields)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pgPerfParsePolicy(data, i); err == nil {
			t.Fatalf("legacy admitted explicit null %s", key)
		}
	}
	for _, name := range []string{"expired", "null", "empty", "pseudo-legacy"} {
		t.Run(name, func(t *testing.T) {
			candidate := p
			input := i
			switch name {
			case "expired":
				input.Now = start.Add(8700 * time.Second)
			case "null":
				candidate.ResourceWindow = json.RawMessage(`null`)
			case "empty":
				candidate.ResourceWindow = json.RawMessage(`""`)
			case "pseudo-legacy":
				candidate.AttemptID = "new-attempt"
			}
			data, err := json.Marshal(candidate)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = pgPerfParsePolicy(data, input); err == nil {
				t.Fatal("invalid legacy fallback admitted")
			}
		})
	}
}

func TestPgPerfR04_MediumReceiptAndPartialRegistration(t *testing.T) {
	for _, fault := range []string{"none", "missing-receipt", "partial-baseline", "orphan-receipt", "receipt-mismatch", "existing-not-bound"} {
		t.Run(fault, func(t *testing.T) {
			p, i := pgPerfR04Fixture(t)
			parsed := pgPerfR04Parse(t, p, i)
			r := parsed.Resource
			if fault == "missing-receipt" {
				data, err := json.Marshal(pgPerfBaselineRecord{r.Window, "medium", pgPerfResourceBase + 100})
				if err != nil {
					t.Fatal(err)
				}
				if err = pgPerfWriteExclusive(filepath.Join(r.Directory, "medium.json"), data); err != nil {
					t.Fatal(err)
				}
			} else if fault == "partial-baseline" {
				if err := pgPerfWriteExclusive(filepath.Join(r.Directory, "medium.json"), []byte(`{"Window":`)); err != nil {
					t.Fatal(err)
				}
			} else if fault == "orphan-receipt" {
				if err := pgPerfWriteExclusive(filepath.Join(r.Directory, "medium.registration.json"), []byte(`{}`)); err != nil {
					t.Fatal(err)
				}
			} else {
				if got, err := pgPerfTierBaseline(r, "medium", pgPerfResourceBase+100); err != nil || got != pgPerfResourceBase+100 {
					t.Fatalf("registration=%d %v", got, err)
				}
				if fault == "receipt-mismatch" {
					r.AttemptID = "different-attempt"
				}
			}
			if fault == "missing-receipt" || fault == "partial-baseline" || fault == "orphan-receipt" || fault == "receipt-mismatch" {
				if _, err := pgPerfTierBaseline(r, "medium", pgPerfResourceBase+300); err == nil {
					t.Fatal("partial/conflicting registration admitted")
				}
				return
			}
			data, err := os.ReadFile(filepath.Join(r.Directory, "medium.json"))
			if err != nil {
				t.Fatal(err)
			}
			receipt, err := os.ReadFile(filepath.Join(r.Directory, "medium.registration.json"))
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(data)
			receiptSum := sha256.Sum256(receipt)
			b := p.ResourceBaselines["medium"]
			b.State = "existing"
			b.AllowFirst = false
			b.Bytes = pgPerfResourceBase + 100
			b.FileBytes = int64(len(data))
			b.SHA256 = hex.EncodeToString(sum[:])
			b.ReceiptSHA256 = hex.EncodeToString(receiptSum[:])
			if fault == "existing-not-bound" {
				b.ReceiptSHA256 = ""
			}
			if fault == "missing-receipt" {
				b.ReceiptSHA256 = strings.Repeat("0", 64)
			}
			p.ResourceBaselines["medium"] = b
			p.AttemptID = "next-offline-attempt"
			encoded, err := json.Marshal(p)
			if err != nil {
				t.Fatal(err)
			}
			next, err := pgPerfParsePolicy(encoded, i)
			if fault != "none" {
				if err == nil {
					t.Fatal("unbound receipt admitted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got, err := pgPerfTierBaseline(next.Resource, "medium", pgPerfResourceBase+300); err != nil || got != pgPerfResourceBase+100 {
				t.Fatalf("later attempt reset baseline=%d %v", got, err)
			}
			var saved pgPerfBaselineRecord
			if err = json.Unmarshal(data, &saved); err != nil || saved.Window != pgPerfResourceOrigin {
				t.Fatal("registration changed origin")
			}
		})
	}
}

func TestPgPerfR04_ExclusiveWriteAndChildEnvironment(t *testing.T) {
	p, i := pgPerfR04Fixture(t)
	path := filepath.Join(filepath.Dir(i.ExpectedDirectory), "exclusive.json")
	var successes atomic.Int32
	start := make(chan struct{})
	var joined sync.WaitGroup
	for index := 0; index < 2; index++ {
		joined.Add(1)
		go func() {
			defer joined.Done()
			<-start
			if pgPerfWriteExclusive(path, []byte("complete\n")) == nil {
				successes.Add(1)
			}
		}()
	}
	close(start)
	joined.Wait()
	if successes.Load() != 1 {
		t.Fatalf("exclusive creators=%d", successes.Load())
	}
	base := []string{pgPerfWindowStartEnv + "=stale", strings.ToLower(pgPerfWindowStartEnv) + "=duplicate", pgPerfBaseBytesEnv + "=1", "KEPT=value", "KEPT=other"}
	env := pgPerfChildEnvironment(base, map[string]string{pgPerfWindowStartEnv: p.WindowStart, pgPerfBaseBytesEnv: strconv.FormatInt(pgPerfResourceBase, 10), pgPerfBaselineDirEnv: p.ResourceDirectory})
	seen := map[string]string{}
	for _, entry := range env {
		key, value, _ := strings.Cut(entry, "=")
		key = strings.ToUpper(key)
		if _, ok := seen[key]; ok {
			t.Fatal("duplicate child environment key")
		}
		seen[key] = value
	}
	if seen[pgPerfWindowStartEnv] != p.WindowStart || seen[pgPerfBaseBytesEnv] != strconv.FormatInt(pgPerfResourceBase, 10) {
		t.Fatal("stale environment won")
	}
}

func TestPgPerfR04_ActualLedgerFaultsAndReparseRejected(t *testing.T) {
	for _, fault := range []string{"small-missing", "unknown-file", "unbound-medium", "small-rebound-bytes", "small-rebound-length", "junction"} {
		t.Run(fault, func(t *testing.T) {
			p, i := pgPerfR04Fixture(t)
			if fault == "junction" {
				link := filepath.Join(filepath.Dir(i.ExpectedDirectory), "ledger-junction")
				output, err := exec.Command("cmd", "/c", "mklink", "/J", link, p.ResourceDirectory).CombinedOutput()
				if err != nil {
					t.Fatalf("junction fixture unavailable: %v %s", err, output)
				}
				p.ResourceDirectory = link
			} else {
				directory := filepath.Join(filepath.Dir(i.ExpectedDirectory), "fault-ledgers")
				if err := os.Mkdir(directory, 0o755); err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(filepath.Join(p.ResourceDirectory, "small.json"))
				if err != nil {
					t.Fatal(err)
				}
				if fault == "small-rebound-bytes" {
					data = []byte(strings.Replace(string(data), "1406932671", "1406932672", 1))
				}
				if fault == "small-rebound-length" {
					data = append(data, ' ')
				}
				if fault != "small-missing" {
					if err = pgPerfWriteExclusive(filepath.Join(directory, "small.json"), data); err != nil {
						t.Fatal(err)
					}
				}
				if strings.HasPrefix(fault, "small-rebound") {
					b := p.ResourceBaselines["small"]
					sum := sha256.Sum256(data)
					b.SHA256 = hex.EncodeToString(sum[:])
					b.FileBytes = int64(len(data))
					if fault == "small-rebound-bytes" {
						b.Bytes++
					}
					p.ResourceBaselines["small"] = b
				}
				if fault == "unknown-file" {
					if err = pgPerfWriteExclusive(filepath.Join(directory, "other.json"), []byte(`{}`)); err != nil {
						t.Fatal(err)
					}
				}
				if fault == "unbound-medium" {
					if err = pgPerfWriteExclusive(filepath.Join(directory, "medium.json"), []byte(`{"Window":"2026-11-04T03:29:04.586115+00:00","Tier":"medium","Bytes":1500000000}`)); err != nil {
						t.Fatal(err)
					}
				}
				p.ResourceDirectory = directory
				i.ExpectedDirectory = directory
			}
			data, err := json.Marshal(p)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = pgPerfParsePolicy(data, i); err == nil {
				t.Fatal("invalid actual ledger accepted")
			}
			t.Logf("actual ledger rejected: %v", err)
		})
	}
}

func TestPgPerfR04_OriginalSources_ReadOnly(t *testing.T) {
	p, i := pgPerfR04Fixture(t)
	directory, global, adoption := pgPerfPolicyPaths()
	data, err := os.ReadFile(filepath.Join(directory, "small.json"))
	if err != nil {
		t.Fatal(err)
	}
	before := sha256.Sum256(data)
	p.ResourceDirectory = directory
	p.AdoptionSource = adoption
	adopted, err := os.ReadFile(adoption)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(adopted)
	p.AdoptionSHA256 = hex.EncodeToString(sum[:])
	i.ExpectedDirectory = directory
	i.GlobalBaseFile = global
	i.AdoptionFile = adoption
	parsed := pgPerfR04Parse(t, p, i)
	if got, err := pgPerfTierBaseline(parsed.Resource, "small", 1992988351); err != nil || got != pgPerfResourceBase {
		t.Fatalf("original baseline=%d err=%v", got, err)
	}
	after, err := os.ReadFile(filepath.Join(directory, "small.json"))
	if err != nil || sha256.Sum256(after) != before {
		t.Fatal("original 80-byte baseline drift")
	}
	t.Logf("read-only original small length=%d sha256=%X; synthetic clock input only, no actual attempt", len(after), before)
}

func TestPgPerfR05_PartialBudgetRejected(t *testing.T) {
	p, input := pgPerfR04Fixture(t)
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	fields["resource_probe_timeout_ms"] = json.RawMessage(`15000`)
	data, err = json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pgPerfParsePolicy(data, input); err == nil {
		t.Fatal("partial resource budget silently accepted; requires adopted 15/5/20 or absent legacy 5s")
	}
}

func pgPerfR05State(t *testing.T) (*pgPerfResourceState, *atomic.Int64, time.Time) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	s := newPgPerfResourceState(ctx, cancel, pgPerfProbeBudget{15 * time.Second, 5 * time.Second, 20 * time.Second})
	base := time.Now()
	clock := &atomic.Int64{}
	s.now = func() time.Time { return base.Add(time.Duration(clock.Load())) }
	return s, clock, base
}

func TestPgPerfR05_StateTimeline_ExpiryAndLate(t *testing.T) {
	for _, timely := range []bool{false, true} {
		t.Run(strconv.FormatBool(timely), func(t *testing.T) {
			s, clock, base := pgPerfR05State(t)
			if s.admit(s.owner) == nil {
				t.Fatal("first UNKNOWN admitted")
			}
			_, err := s.collect(s.owner, func(ctx context.Context) (pgPerfResourceObservation, error) {
				end, _ := ctx.Deadline()
				if !end.Equal(base.Add(15 * time.Second)) {
					t.Fatal("first total deadline differs")
				}
				clock.Store(int64(14900 * time.Millisecond))
				return pgPerfResourceObservation{Bytes: 100, Free: 1000}, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			s.mu.Lock()
			end := s.freshnessLocked()
			s.mu.Unlock()
			if !end.Equal(base.Add(20 * time.Second)) {
				t.Fatal("finish incorrectly extended freshness")
			}
			for _, tick := range []time.Duration{5 * time.Second, 10 * time.Second} {
				select {
				case s.changed <- struct{}{}:
				default:
				}
				s.mu.Lock()
				same := s.freshnessLocked().Equal(end)
				s.mu.Unlock()
				if !same {
					t.Fatalf("coalesced tick %s refreshed success", tick)
				}
			}
			clock.Store(int64(15 * time.Second))
			if err := s.admit(s.owner); err != nil {
				t.Fatal("valid old observation denied while next scan starts", err)
			}
			_, err = s.collect(s.owner, func(ctx context.Context) (pgPerfResourceObservation, error) {
				bound, _ := ctx.Deadline()
				if !bound.Equal(base.Add(20 * time.Second)) {
					t.Fatal("second scan escaped old freshness")
				}
				if timely {
					clock.Store(int64(19 * time.Second))
				} else {
					clock.Store(int64(20 * time.Second))
					if s.admit(s.owner) == nil {
						t.Fatal("t20 admitted despite blocked IO")
					}
					if s.owner.Err() == nil {
						t.Fatal("expiry did not cancel root independently")
					}
					clock.Store(int64(29900 * time.Millisecond))
				}
				return pgPerfResourceObservation{Bytes: 200, Free: 1000}, nil
			})
			s.mu.Lock()
			latest, deadline := s.last, s.freshnessLocked()
			s.mu.Unlock()
			if timely {
				if err != nil || latest.Bytes != 200 || !deadline.Equal(base.Add(35*time.Second)) {
					t.Fatalf("timely success not published: %v", err)
				}
			} else if err == nil || latest.Bytes != 100 || !deadline.Equal(base.Add(20*time.Second)) {
				t.Fatal("late result revived terminal root")
			}
			t.Logf("controlled monotonic timeline timely=%t deadline_offset=%s last_bytes=%d root_error=%v", timely, deadline.Sub(base), latest.Bytes, s.owner.Err())
		})
	}
}

func TestPgPerfR05_RequestWaitParentAndFailure(t *testing.T) {
	for _, fault := range []string{"queue", "SQL-error", "parent-cancel", "late-first"} {
		t.Run(fault, func(t *testing.T) {
			s, clock, _ := pgPerfR05State(t)
			if fault != "late-first" {
				if _, err := s.collect(s.owner, func(context.Context) (pgPerfResourceObservation, error) {
					return pgPerfResourceObservation{Bytes: 100, Free: 1000}, nil
				}); err != nil {
					t.Fatal(err)
				}
			}
			var called atomic.Bool
			ctx, cancel := context.WithTimeout(s.owner, 10*time.Millisecond)
			defer cancel()
			if fault == "queue" {
				s.flight <- struct{}{}
				defer func() { <-s.flight }()
			}
			_, err := s.collect(ctx, func(ctx context.Context) (pgPerfResourceObservation, error) {
				called.Store(true)
				switch fault {
				case "SQL-error":
					return pgPerfResourceObservation{}, errors.New("controlled SQL failure")
				case "parent-cancel":
					cancel()
				case "late-first":
					clock.Store(int64(15 * time.Second))
				}
				return pgPerfResourceObservation{Bytes: 200, Free: 1000}, nil
			})
			if err == nil || s.admit(s.owner) == nil {
				t.Fatal("failed/late/canceled request admitted")
			}
			if fault == "queue" && called.Load() {
				t.Fatal("lock wait received a new SQL budget")
			}
			s.mu.Lock()
			latest := s.last
			s.mu.Unlock()
			if latest.Bytes == 200 {
				t.Fatal("failed request refreshed success")
			}
		})
	}
}

func TestPgPerfR05_IndependentExpiryAndMetricCache(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := newPgPerfResourceState(ctx, cancel, pgPerfProbeBudget{15 * time.Second, 5 * time.Second, 20 * time.Millisecond})
	if _, err := s.collect(ctx, func(context.Context) (pgPerfResourceObservation, error) {
		return pgPerfResourceObservation{Bytes: 100, Free: 1000}, nil
	}); err != nil {
		t.Fatal(err)
	}
	h := &pgPerfHarness{resource: s, diskCtx: ctx, cancel: cancel, ctx: ctx}
	h.startResourceExpiry()
	s.flight <- struct{}{}
	bound := time.NewTimer(time.Second)
	defer bound.Stop()
	select {
	case <-ctx.Done():
	case <-bound.C:
		t.Fatal("expiry blocked behind single-flight IO")
	}
	<-s.flight
	h.group.Wait()
	s.mu.Lock()
	anchor := s.last.Started
	s.mu.Unlock()
	if got := h.pgPerfDatabaseSize(); got != 100 {
		t.Fatal("metric lost its recorded value")
	}
	s.mu.Lock()
	same := s.last.Started.Equal(anchor)
	s.mu.Unlock()
	if !same || s.admit(ctx) == nil {
		t.Fatal("post-point metric renewed expired guard")
	}
	t.Log("scaled 20ms logical expiry fixture only; production policy remains 20000ms; independent expiry canceled while flight token held")
}

func TestPgPerfR05_ObservedCapsAndReserve(t *testing.T) {
	h := &pgPerfHarness{tier: pgPerfTierSmall, resourceBaseBytes: pgPerfResourceBase, resourceTierBytes: pgPerfResourceBase}
	for _, delta := range []int64{1024, h.tier.diskLimit - 1, h.tier.diskLimit, h.tier.diskLimit + 1} {
		value := pgPerfResourceObservation{Bytes: pgPerfResourceBase + delta, Free: 100 * 1024 * 1024 * 1024}
		err := h.validateResourceObservation(value, "ok")
		if (err != nil) != (delta >= h.tier.diskLimit) {
			t.Fatalf("observed delta=%d err=%v", delta, err)
		}
	}
	if h.validateResourceObservation(pgPerfResourceObservation{Bytes: pgPerfResourceBase, Free: 1}, "ok") == nil {
		t.Fatal("insufficient target reserve admitted")
	}
	if h.validateResourceObservation(pgPerfResourceObservation{Bytes: pgPerfResourceBase, Free: 100 * 1024 * 1024 * 1024}, "unknown") == nil {
		t.Fatal("unknown volume admitted")
	}
	h.resourceTierBytes = 0
	if h.validateResourceObservation(pgPerfResourceObservation{Bytes: pgPerfResourceBase + 15*1024*1024*1024, Free: 100 * 1024 * 1024 * 1024}, "ok") == nil {
		t.Fatal("initial medium unknown tier bypassed global cap")
	}
}

func TestPgPerfR05_BudgetPolicyAndLegacy(t *testing.T) {
	p, input := pgPerfR04Fixture(t)
	if got := pgPerfR04Parse(t, p, input).ProbeBudget; got.Timeout != 5*time.Second || got.Freshness != 0 {
		t.Fatal("missing fields changed legacy 5s semantics")
	}
	p.ProbeTimeout = json.RawMessage(`15000`)
	p.MonitorTick = json.RawMessage(`5000`)
	p.FreshnessLimit = json.RawMessage(`20000`)
	p.ProbeAdoption = p.AdoptionSource
	p.ProbeAdoptionSHA = p.AdoptionSHA256
	input.ProbeAdoptionFile = input.AdoptionFile
	if got := pgPerfR04Parse(t, p, input).ProbeBudget; got.Timeout != 15*time.Second || got.Tick != 5*time.Second || got.Freshness != 20*time.Second {
		t.Fatal("explicit budget not adopted")
	}
	for _, fault := range []string{"null", "empty", "wrong", "missing", "adoption", "old-detection-field"} {
		t.Run(fault, func(t *testing.T) {
			candidate := p
			switch fault {
			case "null":
				candidate.ProbeTimeout = json.RawMessage(`null`)
			case "empty":
				candidate.ProbeTimeout = json.RawMessage(`""`)
			case "wrong":
				candidate.FreshnessLimit = json.RawMessage(`21000`)
			case "missing":
				candidate.MonitorTick = nil
			case "adoption":
				candidate.ProbeAdoptionSHA = strings.Repeat("0", 64)
			case "old-detection-field":
				candidate.DetectionEnvelope = json.RawMessage(`20000`)
			}
			data, err := json.Marshal(candidate)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = pgPerfParsePolicy(data, input); err == nil {
				t.Fatal("invalid probe budget admitted")
			}
		})
	}
}

func TestPgPerfR05_PhaseOwnerAndNoncooperativeDeadline(t *testing.T) {
	t.Run("phase-owned", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		s := newPgPerfResourceState(ctx, cancel, pgPerfProbeBudget{15 * time.Second, 5 * time.Second, 20 * time.Second})
		h := &pgPerfHarness{ctx: ctx, cancel: cancel, resource: s}
		var seen atomic.Bool
		result := h.pgPerfPhase(5*time.Millisecond, 1, true, 0, func(phaseCtx context.Context, _ int) (bool, error) {
			_, err := s.collect(phaseCtx, func(queryCtx context.Context) (pgPerfResourceObservation, error) {
				phaseEnd, _ := phaseCtx.Deadline()
				queryEnd, _ := queryCtx.Deadline()
				seen.Store(!queryEnd.After(phaseEnd))
				<-queryCtx.Done()
				return pgPerfResourceObservation{}, queryCtx.Err()
			})
			return false, err
		})
		if !seen.Load() || ctx.Err() == nil || result.completed {
			t.Fatal("synchronous phase probe escaped phase deadline/cancellation")
		}
	})
	t.Run("root-observer-survives-normal-phase", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		s := newPgPerfResourceState(ctx, cancel, pgPerfProbeBudget{15 * time.Second, 5 * time.Second, 20 * time.Second})
		if _, err := s.collect(ctx, func(context.Context) (pgPerfResourceObservation, error) {
			return pgPerfResourceObservation{Bytes: 100, Free: 1000}, nil
		}); err != nil {
			t.Fatal(err)
		}
		ready, release := make(chan struct{}), make(chan struct{})
		done := make(chan error, 1)
		h := &pgPerfHarness{ctx: ctx, cancel: cancel, resource: s}
		h.group.Add(1)
		go func() {
			defer h.group.Done()
			_, err := s.collect(ctx, func(context.Context) (pgPerfResourceObservation, error) {
				close(ready)
				<-release
				return pgPerfResourceObservation{Bytes: 200, Free: 1000}, nil
			})
			done <- err
		}()
		<-ready
		h.pgPerfPhase(2*time.Millisecond, 1, false, 0, func(context.Context, int) (bool, error) { return true, nil })
		alive := ctx.Err() == nil
		close(release)
		err := <-done
		h.group.Wait()
		if !alive || err != nil {
			t.Fatal("normal phase end incorrectly canceled root observer", err)
		}
	})
	t.Run("request-cancel-independent-of-reader", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		s := newPgPerfResourceState(ctx, cancel, pgPerfProbeBudget{10 * time.Millisecond, 5 * time.Second, 20 * time.Second})
		ready, release := make(chan struct{}), make(chan struct{})
		done := make(chan error, 1)
		go func() {
			_, err := s.collect(ctx, func(context.Context) (pgPerfResourceObservation, error) {
				close(ready)
				<-release
				return pgPerfResourceObservation{Bytes: 100, Free: 1000}, nil
			})
			done <- err
		}()
		<-ready
		deadline := time.NewTimer(time.Second)
		defer deadline.Stop()
		select {
		case <-ctx.Done():
		case <-deadline.C:
			close(release)
			<-done
			t.Fatal("request deadline waited for uncooperative read")
		}
		close(release)
		if err := <-done; err == nil {
			t.Fatal("late uncooperative reader was accepted")
		}
	})
}

func TestPgPerfR05_RootClose_CooperativeAndFailStop(t *testing.T) {
	_, input := pgPerfR04Fixture(t)
	directory := filepath.Join(filepath.Dir(input.ExpectedDirectory), "closure-evidence")
	childMode := os.Getenv("HAN_PG_PERF_RESOURCE_CLOSE_CHILD") == "uncooperative"
	if childMode {
		if inherited := os.Getenv("HAN_PG_PERF_RESOURCE_CLOSE_EVIDENCE"); inherited != "" {
			directory = inherited
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e := &pgPerfEvidence{root: directory, runID: "r05-offline-closure", started: time.Now()}
	h := &pgPerfHarness{ctx: ctx, diskCtx: ctx, cancel: cancel, diskCancel: cancel, t: t, evidence: e, schema: "offline-no-schema"}
	h.resource = newPgPerfResourceState(ctx, cancel, pgPerfProbeBudget{15 * time.Second, 5 * time.Second, 20 * time.Second})
	baseline := runtime.NumGoroutine()
	ready := make(chan struct{})
	h.group.Add(1)
	go func() {
		defer h.group.Done()
		_, _ = h.resource.collect(ctx, func(ctx context.Context) (pgPerfResourceObservation, error) {
			close(ready)
			if childMode {
				select {}
			}
			<-ctx.Done()
			return pgPerfResourceObservation{}, ctx.Err()
		})
	}()
	<-ready
	started := time.Now()
	h.closeResources(baseline)
	if time.Since(started) > 2*time.Second {
		t.Fatal("cooperative close consumed resource 15s opportunity")
	}
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	childCtx, childCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer childCancel()
	child := exec.CommandContext(childCtx, binary, "-test.run=^TestPgPerfR05_RootClose_CooperativeAndFailStop$", "-test.v")
	childEvidence := filepath.Join(directory, "noncooperative-child")
	child.Env = pgPerfChildEnvironment(os.Environ(), map[string]string{"HAN_PG_PERF_RESOURCE_CLOSE_CHILD": "uncooperative", "HAN_PG_PERF_RESOURCE_CLOSE_EVIDENCE": childEvidence})
	output, err := child.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 86 || !strings.Contains(string(output), "resource owned test child fail-stop exit=86") {
		t.Fatalf("noncooperative close not fail-stopped by owned child boundary: %v %s", err, output)
	}
	paths, err := filepath.Glob(filepath.Join(childEvidence, "failures", "resource.*.fail-stop.json"))
	if err != nil || len(paths) != 1 {
		t.Fatal("missing unique persisted resource fail-stop evidence")
	}
	data, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	var proof struct {
		PID      int  `json:"pid"`
		Exit     int  `json:"exit_code"`
		Unknown  bool `json:"unknown"`
		Canceled bool `json:"root_canceled"`
	}
	if err = json.Unmarshal(data, &proof); err != nil || proof.PID != child.Process.Pid || proof.Exit != 86 || !proof.Unknown || !proof.Canceled {
		t.Fatal("fail-stop record is not tied to waited owned child")
	}
	if _, err = os.Stat(filepath.Join(childEvidence, "environment", "schema-inventory.json")); err != nil {
		t.Fatal("fail-stop did not persist inventory before owned exit")
	}
	t.Logf("actual_fail_stop_evidence=%s owned_pid=%d UNKNOWN retained; child wait complete", paths[0], proof.PID)
	t.Logf("owned child pid=%d actual wait exit=86 (expected failure, no resource success); evidence retained\n%s", child.Process.Pid, output)
}

func TestPgPerfR05_ActualBaseline_FirstReuseAndBoundReceipt(t *testing.T) {
	p, input := pgPerfR04Fixture(t)
	parsed := pgPerfR04Parse(t, p, input)
	before, _ := json.Marshal(parsed.Resource.Baselines)
	first, err := pgPerfLoadTierBaseline(parsed.Resource, "medium", pgPerfResourceBase+100)
	if err != nil {
		t.Fatal(err)
	}
	if first.LoadState != "first_registration" || first.AuthorizationState != "unregistered" || first.Bytes != pgPerfResourceBase+100 || first.SHA256 == "" || first.ReceiptSHA256 == "" || first.ReceiptFileBytes <= 0 {
		t.Fatalf("missing first-registration actual identity: %+v", first)
	}
	second, err := pgPerfLoadTierBaseline(parsed.Resource, "medium", pgPerfResourceBase+300)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(parsed.Resource.Baselines)
	if string(before) != string(after) {
		t.Fatal("authorization map was mutated")
	}
	if second.LoadState != "existing_file_reuse" || second.SHA256 != first.SHA256 || second.ReceiptSHA256 != first.ReceiptSHA256 || second.Bytes != first.Bytes {
		t.Fatal("same-policy reuse lost effective identity")
	}
	b := p.ResourceBaselines["medium"]
	b.State = "existing"
	b.AllowFirst = false
	b.Bytes = first.Bytes
	b.FileBytes = first.FileBytes
	b.SHA256 = first.SHA256
	b.ReceiptSHA256 = first.ReceiptSHA256
	p.ResourceBaselines["medium"] = b
	p.AttemptID = "next-offline-attempt"
	bound := pgPerfR04Parse(t, p, input)
	third, err := pgPerfLoadTierBaseline(bound.Resource, "medium", pgPerfResourceBase+500)
	if err != nil {
		t.Fatal(err)
	}
	if third.AuthorizationState != "existing" || third.LoadState != "existing_file_reuse" || third.Bytes != first.Bytes || third.SHA256 != first.SHA256 || third.ReceiptSHA256 != first.ReceiptSHA256 {
		t.Fatal("bound later policy did not load original file and receipt")
	}
	data, err := os.ReadFile(first.Path)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := os.ReadFile(first.ReceiptPath)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	receiptSum := sha256.Sum256(receipt)
	if !strings.EqualFold(first.SHA256, hex.EncodeToString(sum[:])) || !strings.EqualFold(first.ReceiptSHA256, hex.EncodeToString(receiptSum[:])) || first.FileBytes != int64(len(data)) || first.ReceiptFileBytes != int64(len(receipt)) {
		t.Fatal("actual fingerprint not derived from validated physical bytes")
	}
	e := &pgPerfEvidence{root: filepath.Join(filepath.Dir(input.ExpectedDirectory), "actual-fingerprint"), runID: "offline-F001-no-PG"}
	for index, actual := range []pgPerfLoadedBaseline{first, second, third} {
		authorization := parsed.Resource
		if index == 2 {
			authorization = bound.Resource
		}
		h := &pgPerfHarness{t: t, appName: fmt.Sprintf("offline-harness-%d", index), actualBaseline: actual}
		e.writeFingerprint(h, nil, map[string]any{"case": index, "resource_binding": authorization, "resource_actual_baseline": h.actualBaseline})
	}
	var document struct {
		Harnesses []struct {
			Authorization pgPerfResourceBinding `json:"resource_binding"`
			Actual        pgPerfLoadedBaseline  `json:"resource_actual_baseline"`
		} `json:"harnesses"`
	}
	if !e.readJSON("environment/fingerprint.json", &document) || len(document.Harnesses) != 3 {
		t.Fatal("actual fingerprint consumer omitted a harness")
	}
	for index, harness := range document.Harnesses {
		if harness.Actual.Bytes != first.Bytes || harness.Actual.ReceiptSHA256 != first.ReceiptSHA256 {
			t.Fatal("persisted actual identity differs from validated file")
		}
		if index < 2 && harness.Authorization.Baselines["medium"].Bytes != 0 {
			t.Fatal("authorization overwritten with actual observation")
		}
	}
	t.Logf("offline persisted fingerprint=%s", e.path("environment/fingerprint.json"))
	t.Logf("authorization=unregistered; actual first=%+v; reuse=%+v; later-bound=%+v", first, second, third)
}

func TestPgPerfR05_ClosingKeepsUnexpectedReadFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := newPgPerfResourceState(ctx, cancel, pgPerfProbeBudget{15 * time.Second, 5 * time.Second, 20 * time.Second})
	ready, release := make(chan struct{}), make(chan struct{})
	done := make(chan pgPerfResourceObservation, 1)
	go func() {
		observation, _ := s.collect(ctx, func(context.Context) (pgPerfResourceObservation, error) {
			close(ready)
			<-release
			return pgPerfResourceObservation{}, errors.New("unexpected reader failure concurrent with root close")
		})
		done <- observation
	}()
	<-ready
	cancel()
	close(release)
	observation := <-done
	s.mu.Lock()
	failed := s.failedOwn
	s.mu.Unlock()
	if !failed || observation.ReadErrorClass == "" {
		t.Fatal("closing root masked unexpected observer failure")
	}
}

func TestPgPerfR06_ReaderErrorDuringClosing(t *testing.T) {
	for _, stage := range []string{"acquire", "SQL"} {
		for _, kind := range []string{"pg", "wrapped-pg", "plain", "mixed"} {
			t.Run(stage+"/"+kind, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				s := newPgPerfResourceState(ctx, cancel, pgPerfProbeBudget{15 * time.Second, 5 * time.Second, 20 * time.Second})
				var original error = &pgconn.PgError{Code: "42501", Message: "synthetic-sensitive-detail-must-not-escape"}
				switch kind {
				case "wrapped-pg":
					original = fmt.Errorf("synthetic-sensitive-wrapper: %w", original)
				case "plain":
					original = errors.New("synthetic-sensitive-detail-must-not-escape")
				case "mixed":
					original = errors.Join(context.Canceled, original)
				}
				ready, release := make(chan struct{}), make(chan struct{})
				type outcome struct {
					observation pgPerfResourceObservation
					err         error
				}
				done := make(chan outcome, 1)
				joined := make(chan struct{})
				go func() {
					defer close(joined)
					observation, readErr := s.collect(ctx, func(operationCtx context.Context) (pgPerfResourceObservation, error) {
						close(ready)
						<-release
						return pgPerfResourceObservation{}, pgPerfResourceReadError(operationCtx, stage, original)
					})
					done <- outcome{observation, readErr}
				}()
				<-ready
				cancel()
				s.mu.Lock()
				s.stopLocked(context.Canceled)
				s.mu.Unlock()
				close(release)
				result := <-done
				<-joined
				observation := result.observation
				if pgPerfContextOnly(result.err) || strings.Contains(result.err.Error(), "synthetic-sensitive") {
					t.Fatal("collector lost safe non-context failure or exposed original text")
				}
				s.mu.Lock()
				failed := s.failedOwn
				s.mu.Unlock()
				if !failed {
					t.Fatalf("actual reader helper masked non-context failure: resource_failure=%t class=%s", failed, observation.ReadErrorClass)
				}
				if strings.Contains(observation.ReadErrorClass, "synthetic-sensitive") {
					t.Fatal("sensitive error text escaped classification")
				}
				if kind != "plain" && !strings.Contains(observation.ReadErrorClass, "42501") {
					t.Fatalf("SQLSTATE lost: class=%s", observation.ReadErrorClass)
				}
				t.Logf("stage=%s kind=%s resource_failure=%t safe_class=%s; reader joined", stage, kind, failed, observation.ReadErrorClass)
			})
		}
	}
}

func TestPgPerfR06_PureCancellationAndSuccessfulLateResult(t *testing.T) {
	for _, original := range []error{context.Canceled, context.DeadlineExceeded, fmt.Errorf("synthetic-sensitive-wrapper: %w", context.Canceled), errors.Join(context.Canceled, context.DeadlineExceeded), nil} {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		for _, stage := range []string{"acquire", "SQL"} {
			normalized := pgPerfResourceReadError(ctx, stage, original)
			if !pgPerfContextOnly(normalized) || strings.Contains(normalized.Error(), "synthetic-sensitive") {
				t.Fatal("pure cancellation misclassified or leaked error text")
			}
		}
		s := newPgPerfResourceState(ctx, cancel, pgPerfProbeBudget{15 * time.Second, 5 * time.Second, 20 * time.Second})
		s.mu.Lock()
		s.stopLocked(context.Canceled)
		failed := s.failedOwn
		s.mu.Unlock()
		if failed {
			t.Fatal("normal close incorrectly marked resource failure")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := newPgPerfResourceState(ctx, cancel, pgPerfProbeBudget{15 * time.Second, 5 * time.Second, 20 * time.Second})
	_, err := s.collect(ctx, func(context.Context) (pgPerfResourceObservation, error) {
		cancel()
		return pgPerfResourceObservation{Bytes: 100, Free: 1000}, nil
	})
	if err == nil || !s.last.Started.IsZero() {
		t.Fatal("successful but canceled result was accepted")
	}
	invalid := pgPerfResourceReadError(context.Background(), "SQL", &pgconn.PgError{Code: "synthetic-sensitive-detail", Message: "synthetic-sensitive-detail"})
	if strings.Contains(invalid.Error(), "synthetic-sensitive") {
		t.Fatal("invalid SQLSTATE leaked untrusted text")
	}
}

func TestPgPerfR07_AsyncDestroy_TransientReturn(t *testing.T) {
	if directory := os.Getenv("HAN_PG_PERF_R07_CLOSE_CHILD"); directory != "" {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		entered, release, returned := make(chan struct{}), make(chan struct{}), make(chan struct{})
		pool, err := puddle.NewPool(&puddle.Config[int]{MaxSize: 1, Constructor: func(context.Context) (int, error) { return 1, nil }, Destructor: func(int) { close(entered); <-release; close(returned) }})
		if err != nil {
			t.Fatal(err)
		}
		resource, err := pool.Acquire(ctx)
		if err != nil {
			t.Fatal(err)
		}
		resource.Destroy()
		<-entered
		observed := make(chan struct{})
		var once sync.Once
		joined := make(chan struct{})
		go func() { defer close(joined); <-observed; close(release) }()
		h := &pgPerfHarness{t: t, ctx: ctx, cancel: cancel, evidence: &pgPerfEvidence{root: directory, runID: "offline-r07-real-puddle"}, schema: "offline-no-schema"}
		h.closeResourcePool("probe", time.Now().Add(time.Second), func() int32 {
			count := pool.Stat().AcquiredResources()
			once.Do(func() { close(observed) })
			return count
		}, pool.Close)
		<-joined
		<-returned
		h.group.Wait()
		if pool.Stat().AcquiredResources() != 0 {
			t.Fatal("real asynchronous destruction did not settle")
		}
		return
	}
	_, input := pgPerfR04Fixture(t)
	directory := filepath.Join(filepath.Dir(input.ExpectedDirectory), "async-child")
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, binary, "-test.run=^TestPgPerfR07_AsyncDestroy_TransientReturn$", "-test.v")
	child.Env = pgPerfChildEnvironment(os.Environ(), map[string]string{"HAN_PG_PERF_R07_CLOSE_CHILD": directory})
	output, err := child.CombinedOutput()
	t.Logf("owned_pid=%d actual_wait=%v\n%s", child.Process.Pid, err, output)
	if err != nil {
		t.Fatalf("transient real puddle Destroy was rejected before bounded settlement: %v", err)
	}
}

func TestPgPerfR07_CloseDeadline_FailStopAndIdleDestroy(t *testing.T) {
	if mode := os.Getenv("HAN_PG_PERF_R07_CLOSE_MODE"); mode != "" {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		entered, release := make(chan struct{}), make(chan struct{})
		pool, err := puddle.NewPool(&puddle.Config[int]{MaxSize: 1, Constructor: func(context.Context) (int, error) { return 1, nil }, Destructor: func(int) { close(entered); <-release }})
		if err != nil {
			t.Fatal(err)
		}
		resource, err := pool.Acquire(ctx)
		if err != nil {
			t.Fatal(err)
		}
		h := &pgPerfHarness{t: t, ctx: ctx, cancel: cancel, evidence: &pgPerfEvidence{root: os.Getenv("HAN_PG_PERF_R07_CLOSE_ROOT"), runID: "offline-r07-deadline"}, schema: "offline-no-schema"}
		started := time.Now()
		policy := pgPerfExecutionPolicy{}
		if mode == "early-hard" || mode == "close-blocked" {
			policy.ExecutionStart = started.Add(-9000*time.Second + 150*time.Millisecond)
			policy.HardStopSeconds = 9000
		}
		end := pgPerfClosureEnd(policy, started)
		if mode == "idle-destroy" || mode == "close-blocked" {
			resource.Release()
		}
		var joined chan struct{}
		if mode == "idle-destroy" {
			joined = make(chan struct{})
			go func() { defer close(joined); <-entered; close(release) }()
		}
		h.closeResourcePool("probe", end, func() int32 { return pool.Stat().AcquiredResources() }, pool.Close)
		if joined != nil {
			<-joined
			h.group.Wait()
		}
		if mode != "idle-destroy" {
			t.Fatal("persistent resource or Close escaped deadline fail-stop")
		}
		return
	}
	for _, mode := range []string{"idle-destroy", "persistent-acquired", "early-hard", "close-blocked"} {
		t.Run(mode, func(t *testing.T) {
			_, input := pgPerfR04Fixture(t)
			directory := filepath.Join(filepath.Dir(input.ExpectedDirectory), "deadline-child")
			binary, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			child := exec.CommandContext(ctx, binary, "-test.run=^TestPgPerfR07_CloseDeadline_FailStopAndIdleDestroy$", "-test.v")
			child.Env = pgPerfChildEnvironment(os.Environ(), map[string]string{"HAN_PG_PERF_R07_CLOSE_MODE": mode, "HAN_PG_PERF_R07_CLOSE_ROOT": directory})
			started := time.Now()
			output, err := child.CombinedOutput()
			elapsed := time.Since(started)
			if mode == "idle-destroy" {
				if err != nil {
					t.Fatalf("idle destructor did not finish bounded Close: %v %s", err, output)
				}
			} else {
				var exit *exec.ExitError
				if !errors.As(err, &exit) || exit.ExitCode() != 86 || ctx.Err() != nil {
					t.Fatalf("owned child did not fail-stop at original deadline: %v %s", err, output)
				}
				if mode == "persistent-acquired" && elapsed < 5*time.Second {
					t.Fatal("normal original 5s closure budget was shortened")
				}
				if (mode == "early-hard" || mode == "close-blocked") && elapsed > 2*time.Second {
					t.Fatal("earlier hard deadline was ignored")
				}
				paths, _ := filepath.Glob(filepath.Join(directory, "failures", "*.fail-stop.json"))
				if len(paths) != 1 {
					t.Fatal("missing unique immutable fail-stop record")
				}
				data, readErr := os.ReadFile(paths[0])
				if readErr != nil {
					t.Fatal(readErr)
				}
				var failure struct {
					PID  int `json:"pid"`
					Exit int `json:"exit_code"`
				}
				if json.Unmarshal(data, &failure) != nil || failure.PID != child.Process.Pid || failure.Exit != 86 {
					t.Fatal("fail-stop evidence not tied to actual waited child")
				}
			}
			t.Logf("mode=%s owned_pid=%d real_wait_error=%v elapsed_ns=%d evidence=%s\n%s", mode, child.Process.Pid, err, elapsed.Nanoseconds(), directory, output)
		})
	}
}

type pgPerfR09QueuedContext struct {
	context.Context
	bound   time.Time
	entered chan struct{}
	once    sync.Once
}

func (ctx *pgPerfR09QueuedContext) Deadline() (time.Time, bool) {
	ctx.once.Do(func() { close(ctx.entered) })
	return ctx.bound, true
}

func TestPgPerfR09_FlightSharesInProgressSuccess(t *testing.T) {
	owner, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	s := newPgPerfResourceState(owner, cancel, pgPerfProbeBudget{15 * time.Second, 5 * time.Second, 20 * time.Second})
	oldEnd := time.Now().Add(2 * time.Second)
	s.last = pgPerfResourceObservation{Started: oldEnd.Add(-20 * time.Second), Finished: time.Now().Add(-time.Second), Bytes: 100, Free: 1000}
	started, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	type result struct {
		observation pgPerfResourceObservation
		err         error
	}
	a, b := make(chan result, 1), make(chan result, 1)
	var reads atomic.Int64
	go func() {
		v, err := s.collect(owner, func(context.Context) (pgPerfResourceObservation, error) {
			reads.Add(1)
			close(started)
			<-release
			return pgPerfResourceObservation{Bytes: 200, Free: 1000, SQLNS: 123}, nil
		})
		a <- result{v, err}
	}()
	<-started
	queued := &pgPerfR09QueuedContext{Context: owner, bound: oldEnd, entered: make(chan struct{})}
	go func() {
		v, err := s.collect(queued, func(ctx context.Context) (pgPerfResourceObservation, error) {
			reads.Add(1)
			<-ctx.Done()
			return pgPerfResourceObservation{}, ctx.Err()
		})
		b <- result{v, err}
	}()
	<-queued.entered
	releaseOnce.Do(func() { close(release) })
	first, waiter := <-a, <-b
	if first.err != nil || waiter.err != nil || reads.Load() != 1 {
		t.Fatalf("expected one shared reader before original waiter deadline: reads=%d A=%v B=%v original_end=%s", reads.Load(), first.err, waiter.err, oldEnd.Format(time.RFC3339Nano))
	}
	if !waiter.observation.Started.Equal(first.observation.Started) || waiter.observation.Requested.Before(first.observation.Requested) || !waiter.observation.Deadline.Equal(oldEnd) || waiter.observation.Bytes != 200 {
		t.Fatal("waiter changed source START, request identity or original deadline")
	}
	if !waiter.observation.Reused || waiter.observation.SQLNS != 0 || waiter.observation.SourceSQLNS != 123 || !waiter.observation.SourceRequested.Equal(first.observation.Requested) || waiter.observation.WaitNS != waiter.observation.Returned.Sub(waiter.observation.Requested).Nanoseconds() {
		t.Fatal("shared accounting did not distinguish waiter and original SQL")
	}
	s.mu.Lock()
	last := s.last
	s.mu.Unlock()
	if !last.Requested.Equal(first.observation.Requested) || !last.Started.Equal(first.observation.Started) {
		t.Fatal("waiter republished resource freshness")
	}
	if first.observation.FlightID == 0 || waiter.observation.FlightID != first.observation.FlightID {
		t.Fatal("consumers cannot identify their single actual SQL flight")
	}
	next, err := s.collect(owner, func(context.Context) (pgPerfResourceObservation, error) {
		reads.Add(1)
		return pgPerfResourceObservation{Bytes: 300, Free: 1000}, nil
	})
	if err != nil || next.Reused || next.FlightID == first.observation.FlightID || reads.Load() != 2 || next.Bytes != 300 {
		t.Fatal("new request after completion reused an old cache instead of reading")
	}
}

func TestPgPerfR09_FlightRejectsExpiredCanceledAndErrors(t *testing.T) {
	for _, mode := range []string{"expired-waiter", "canceled-waiter", "SQL-error", "mixed-error"} {
		t.Run(mode, func(t *testing.T) {
			owner, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			s := newPgPerfResourceState(owner, cancel, pgPerfProbeBudget{15 * time.Second, 5 * time.Second, 20 * time.Second})
			s.last = pgPerfResourceObservation{Started: time.Now(), Bytes: 100, Free: 1000}
			started, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			defer once.Do(func() { close(release) })
			a, b := make(chan error, 1), make(chan error, 1)
			original := &pgPerfSafeResourceError{Stage: "SQL", OriginalType: "*pgconn.PgError", SQLState: "42501"}
			var reads atomic.Int64
			go func() {
				_, err := s.collect(owner, func(context.Context) (pgPerfResourceObservation, error) {
					reads.Add(1)
					close(started)
					<-release
					if mode == "SQL-error" {
						return pgPerfResourceObservation{}, original
					}
					if mode == "mixed-error" {
						return pgPerfResourceObservation{}, errors.Join(context.Canceled, original)
					}
					return pgPerfResourceObservation{Bytes: 200, Free: 1000}, nil
				})
				a <- err
			}()
			<-started
			parent, parentCancel := context.WithCancel(owner)
			defer parentCancel()
			bound := time.Now().Add(time.Second)
			if mode == "expired-waiter" {
				bound = time.Now().Add(-time.Second)
			}
			queued := &pgPerfR09QueuedContext{Context: parent, bound: bound, entered: make(chan struct{})}
			go func() {
				_, err := s.collect(queued, func(context.Context) (pgPerfResourceObservation, error) {
					reads.Add(1)
					return pgPerfResourceObservation{Bytes: 300, Free: 1000}, nil
				})
				b <- err
			}()
			<-queued.entered
			var waiterErr error
			if mode == "canceled-waiter" {
				parentCancel()
			}
			if mode == "expired-waiter" || mode == "canceled-waiter" {
				waiterErr = <-b
			}
			once.Do(func() { close(release) })
			firstErr := <-a
			if mode == "SQL-error" || mode == "mixed-error" {
				waiterErr = <-b
			}
			if firstErr == nil || waiterErr == nil || reads.Load() != 1 {
				t.Fatalf("failed/late flight accepted or reread: reads=%d A=%v B=%v", reads.Load(), firstErr, waiterErr)
			}
			s.mu.Lock()
			last, terminal, failedOwn := s.last, s.terminal, s.failedOwn
			s.mu.Unlock()
			if last.Bytes != 100 || terminal == nil {
				t.Fatal("late result republished after terminal")
			}
			if mode == "SQL-error" || mode == "mixed-error" {
				if !errors.Is(firstErr, original) || !errors.Is(waiterErr, original) || !errors.Is(terminal, original) || !failedOwn {
					t.Fatal("real non-context SQL error lost across canceled waiters")
				}
			}
		})
	}
}

func TestPgPerfR09_ConsumerSharedEvidence(t *testing.T) {
	owner, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s := newPgPerfResourceState(owner, cancel, pgPerfProbeBudget{15 * time.Second, 5 * time.Second, 20 * time.Second})
	base := time.Now()
	var clock atomic.Int64
	s.now = func() time.Time { return base.Add(time.Duration(clock.Add(1)) * time.Microsecond) }
	s.last = pgPerfResourceObservation{Started: base, Bytes: 100, Free: 1000}
	_, input := pgPerfR04Fixture(t)
	e := &pgPerfEvidence{root: filepath.Join(filepath.Dir(input.ExpectedDirectory), "consumer"), runID: "offline-r09-no-PG"}
	h := &pgPerfHarness{resource: s, evidence: e, schema: "offline-r09", resourceBaseBytes: 100, resourceTierBytes: 100}
	started, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	a, b := make(chan error, 1), make(chan error, 1)
	go func() {
		a <- h.collectDiskBudgetContext(owner, func(context.Context) (pgPerfResourceObservation, error) {
			close(started)
			<-release
			return pgPerfResourceObservation{Bytes: 200, Free: 1000, SQLNS: 321}, nil
		})
	}()
	<-started
	queued := &pgPerfR09QueuedContext{Context: owner, bound: time.Now().Add(time.Second), entered: make(chan struct{})}
	go func() {
		b <- h.collectDiskBudgetContext(queued, func(context.Context) (pgPerfResourceObservation, error) {
			return pgPerfResourceObservation{}, errors.New("duplicate consumer read")
		})
	}()
	<-queued.entered
	once.Do(func() { close(release) })
	if first, waiter := <-a, <-b; first != nil || waiter != nil {
		t.Fatalf("consumer shared read: A=%v B=%v", first, waiter)
	}
	files, err := filepath.Glob(filepath.Join(e.root, "diagnosis", "*.json"))
	if err != nil || len(files) != 2 {
		t.Fatalf("consumer observations=%d err=%v", len(files), err)
	}
	shared := 0
	for _, path := range files {
		var record struct {
			Reused, Accepted bool
			SourceSQLNS      int64  `json:"source_sql_ns"`
			SQLNS            int64  `json:"sql_ns"`
			Started          string `json:"started_utc"`
			Requested        string `json:"requested_utc"`
			Returned         string `json:"returned_utc"`
		}
		data, err := os.ReadFile(path)
		if err != nil || json.Unmarshal(data, &record) != nil {
			t.Fatal("consumer evidence invalid")
		}
		if record.Reused {
			shared++
			if !record.Accepted || record.SourceSQLNS != 321 || record.SQLNS != 0 || record.Started == record.Requested || record.Returned == "" {
				t.Fatal("consumer confused SQL source and waiter")
			}
		}
	}
	if shared != 1 {
		t.Fatal("consumer omitted shared observation")
	}
}

func pgPerfSeedGuardReached(rootCanceled, failedOwn bool, terminal error) bool {
	var cap *pgPerfResourceCapError
	return rootCanceled && failedOwn && errors.As(terminal, &cap) && ((cap.TierBase > 0 && cap.TierCap > 0 && cap.Observed-cap.TierBase >= cap.TierCap) || (cap.GlobalBase > 0 && cap.TotalCap > 0 && cap.Observed-cap.GlobalBase >= cap.TotalCap))
}

func TestPgPerfR09_SeedGuardClassifiesTarget(t *testing.T) {
	if pgPerfSeedGuardReached(true, true, context.DeadlineExceeded) {
		t.Error("deadline was accepted as the cap target")
	}
	if pgPerfSeedGuardReached(true, true, errors.New("resource freshness expired")) {
		t.Error("freshness expiration was accepted as the cap target")
	}
	cap := &pgPerfResourceCapError{Observed: 201, TierBase: 100, TierCap: 100, GlobalBase: 100, TotalCap: 1000}
	if !pgPerfSeedGuardReached(true, true, cap) || pgPerfSeedGuardReached(false, true, cap) || pgPerfSeedGuardReached(true, false, cap) {
		t.Fatal("real owned cap cancellation classification differs")
	}
	cap.Observed = 199
	if pgPerfSeedGuardReached(true, true, cap) {
		t.Fatal("typed error without actual cap observation accepted")
	}
	owner, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := newPgPerfResourceState(owner, cancel, pgPerfProbeBudget{15 * time.Second, 5 * time.Second, 20 * time.Second})
	h := &pgPerfHarness{resource: s, resourceBaseBytes: 100, resourceTierBytes: 100, tier: pgPerfTier{diskLimit: 100}}
	err := h.validateResourceObservation(pgPerfResourceObservation{Bytes: 201, Free: 10000}, "ok")
	if err == nil {
		t.Fatal("actual guard did not classify observed cap")
	}
	s.mu.Lock()
	s.stopLocked(err)
	terminal, failedOwn := s.terminal, s.failedOwn
	s.mu.Unlock()
	if !pgPerfSeedGuardReached(owner.Err() != nil, failedOwn, terminal) {
		t.Fatal("fault predicate is disconnected from actual resource guard/terminal")
	}
}

func TestPgPerfR08_FinalClose_ExpiredCompliant(t *testing.T) {
	end := time.Now().Add(-time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), end)
	defer cancel()
	err := (&pgPerfHarness{}).waitFinalClose(ctx, end, func(context.Context) (bool, error) { return true, nil })
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expired closure accepted compliant predicate: error=%v", err)
	}
}

func TestPgPerfR08_FinalClose_DeadlineAndErrors(t *testing.T) {
	t.Run("shared-deadline-with-live-context", func(t *testing.T) {
		called := false
		err := (&pgPerfHarness{}).waitFinalClose(context.Background(), time.Now().Add(-time.Second), func(context.Context) (bool, error) { called = true; return true, nil })
		if !errors.Is(err, context.DeadlineExceeded) || called {
			t.Fatal("expired shared deadline accepted or invoked predicate")
		}
	})
	t.Run("late-return", func(t *testing.T) {
		end := time.Now().Add(20 * time.Millisecond)
		ctx, cancel := context.WithDeadline(context.Background(), end)
		defer cancel()
		settled := false
		err := (&pgPerfHarness{}).waitFinalClose(ctx, end, func(ctx context.Context) (bool, error) {
			<-ctx.Done()
			settled = true
			return true, nil
		})
		if !settled || !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("late compliant observation accepted: settled=%v error=%v", settled, err)
		}
	})
	t.Run("in-time", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := (&pgPerfHarness{}).waitFinalClose(ctx, time.Now().Add(time.Second), func(context.Context) (bool, error) { return true, nil }); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("cancel-during-observation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		err := (&pgPerfHarness{}).waitFinalClose(ctx, time.Now().Add(time.Second), func(context.Context) (bool, error) { cancel(); return true, nil })
		if !errors.Is(err, context.Canceled) {
			t.Fatal("canceled observation accepted")
		}
	})
	t.Run("real-error-survives-cancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		original := &pgconn.PgError{Code: "42501"}
		err := (&pgPerfHarness{}).waitFinalClose(ctx, time.Now().Add(time.Second), func(context.Context) (bool, error) { cancel(); return false, original })
		if err != original {
			t.Fatal("non-context error replaced by cancellation")
		}
	})
}

func TestPgPerfR07_Cleanup_CanceledAndFailedRoot(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(strconv.FormatBool(failed), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			s := newPgPerfResourceState(ctx, cancel, pgPerfProbeBudget{15 * time.Second, 5 * time.Second, 20 * time.Second})
			if failed {
				s.mu.Lock()
				s.stopLocked(pgPerfResourceReadError(ctx, "SQL", &pgconn.PgError{Code: "42501"}))
				s.mu.Unlock()
			} else {
				cancel()
			}
			_, input := pgPerfR04Fixture(t)
			h := &pgPerfHarness{ctx: ctx, cancel: cancel, resource: s, evidence: &pgPerfEvidence{root: filepath.Join(filepath.Dir(input.ExpectedDirectory), "cleanup"), runID: "offline-root-state"}, schema: "offline-no-schema"}
			var closed atomic.Bool
			h.closeResourcePool("probe", time.Now().Add(time.Second), func() int32 { return 0 }, func() { closed.Store(true) })
			h.group.Wait()
			s.mu.Lock()
			stillFailed := s.failedOwn
			s.mu.Unlock()
			if !closed.Load() || stillFailed != failed {
				t.Fatal("cleanup skipped canceled root or erased genuine resource failure")
			}
		})
	}
}

func TestPgPerfR05_ReadOnlySQL(t *testing.T) {
	for _, mode := range []string{"natural-three", "controlled-six", "total-fifteen-failure", "earlier-freshness-failure"} {
		t.Run(mode, func(t *testing.T) {
			root := os.Getenv(pgPerfEvidenceEnv)
			if !filepath.IsAbs(root) {
				t.Fatal("explicit absolute evidence directory required for read-only SQL diagnostic")
			}
			if err := os.MkdirAll(root, 0o755); err != nil {
				t.Fatal(err)
			}
			directory := filepath.Join(root, fmt.Sprintf("readonly-%s-%d-%d", mode, os.Getpid(), time.Now().UnixNano()))
			if err := os.Mkdir(directory, 0o755); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()
			app := pgPerfApplicationName(fmt.Sprintf("r05-readonly-%d-%d", os.Getpid(), time.Now().UnixNano()))
			config := pgPerfPoolConfig(t, "public", app, 1, 0, 0)
			if config.ConnConfig.Database != pgPerfDefaultDBName {
				t.Fatal("read-only diagnostic requires the existing authorized database before connect")
			}
			config.ConnConfig.RuntimeParams["default_transaction_read_only"] = "on"
			pool, err := pgxpool.NewWithConfig(ctx, config)
			if err != nil {
				t.Fatalf("readonly pool configuration: %T", err)
			}
			identityCtx, identityCancel := context.WithTimeout(ctx, 5*time.Second)
			var host, database, readOnly, path string
			var port, version int
			err = pool.QueryRow(identityCtx, "SELECT host(inet_server_addr()),inet_server_port(),current_setting('server_version_num')::int,current_database(),current_setting('default_transaction_read_only'),current_setting('data_directory')").Scan(&host, &port, &version, &database, &readOnly, &path)
			identityCancel()
			if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() || port != 5433 || version != 180006 || database != pgPerfDefaultDBName || readOnly != "on" {
				pool.Close()
				t.Fatalf("readonly target identity rejected: error_class=%s", pgPerfErrorClass(err))
			}
			e := &pgPerfEvidence{root: directory, runID: app, started: time.Now()}
			h := &pgPerfHarness{t: t, ctx: ctx, cancel: cancel, diskCtx: ctx, diskCancel: cancel, probe: pool, appName: app, pgdataPath: path, diskProbe: pgPerfDiskFree, tier: pgPerfTierSmall, resourceBaseBytes: pgPerfResourceBase, resourceTierBytes: pgPerfResourceBase, evidence: e, schema: "readonly-no-schema"}
			h.resource = newPgPerfResourceState(ctx, cancel, pgPerfProbeBudget{15 * time.Second, 5 * time.Second, 20 * time.Second})
			baseline := runtime.NumGoroutine()
			defer h.closeResources(baseline)
			h.startResourceExpiry()
			repetitions := 1
			if mode == "natural-three" {
				repetitions = 3
			}
			if mode == "earlier-freshness-failure" {
				if err = h.checkDiskBudgetContext(ctx); err != nil {
					t.Fatalf("freshness prerequisite SQL failed: %v", err)
				}
				waitCtx, waitCancel := context.WithTimeout(ctx, 8*time.Second)
				_, err = pool.Exec(waitCtx, "SELECT pg_sleep(7)")
				waitCancel()
				if err != nil {
					t.Fatalf("controlled pre-request age stage failed: %s", pgPerfErrorClass(err))
				}
			}
			for sample := 0; sample < repetitions; sample++ {
				delay := 0
				if mode == "controlled-six" {
					delay = 6
				}
				if mode == "total-fifteen-failure" || mode == "earlier-freshness-failure" {
					delay = 16
				}
				enteredDelay := false
				observation, collectErr := h.resource.collect(ctx, func(operationCtx context.Context) (pgPerfResourceObservation, error) {
					if delay > 0 {
						enteredDelay = true
						if _, err := pool.Exec(operationCtx, "SELECT pg_sleep($1)", delay); err != nil {
							return pgPerfResourceObservation{}, fmt.Errorf("controlled readonly delay: %s", pgPerfErrorClass(err))
						}
					}
					return h.readDiskObservation(operationCtx)
				})
				e.writeJSONExclusive(fmt.Sprintf("sample-%d.json", sample), map[string]any{"mode": mode, "sample": sample, "observation": observation, "injected_sql_delay_seconds": delay, "injection_reached": enteredDelay, "accepted": collectErr == nil, "error_class": pgPerfErrorClass(collectErr), "acquired_after_release": pool.Stat().AcquiredConns(), "identity_read_only": readOnly, "resource_origin": pgPerfResourceOrigin, "formal_performance": "NOT RUN", "new_execution_window": false})
				if collectErr != nil {
					t.Fatalf("READONLY_RESOURCE_FAILURE accepted=false injection_reached=%t: %v (retained failure; never a resource-success sample)", enteredDelay, collectErr)
				}
				if delay == 16 {
					t.Fatal("controlled deadline failure unexpectedly accepted")
				}
				if mode == "controlled-six" && (observation.Finished.Sub(observation.Started) <= 5*time.Second || observation.Finished.Sub(observation.Started) >= 15*time.Second) {
					t.Fatal("controlled opportunity was not >5s and <15s")
				}
			}
		})
	}
}

func TestPgPerfPointDuration_RepeatedWarmup_IncludesEveryRepeat(t *testing.T) {
	if got := pgPerfPointDuration(10*time.Second, 35*time.Second, 3); got != 135*time.Second {
		t.Fatalf("three warm-up/measurement repeats cost %s, want 135s", got)
	}
}

func TestPgPerfPhase_CanceledContext_NoOperations(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	h := &pgPerfHarness{ctx: ctx}
	var calls atomic.Int64
	h.pgPerfPhase(time.Millisecond, 1, true, 0, func(context.Context, int) (bool, error) {
		calls.Add(1)
		return false, ctx.Err()
	})
	if calls.Load() != 0 {
		t.Fatalf("canceled phase admitted %d operations", calls.Load())
	}
}

func TestPgPerfPhase_WarmupFailure_RecordsError(t *testing.T) {
	h := &pgPerfHarness{ctx: context.Background()}
	result := h.pgPerfPhase(time.Millisecond, 1, false, 0, func(context.Context, int) (bool, error) {
		return false, errors.New("synthetic warm-up failure")
	})
	if result.errors == 0 {
		t.Fatal("warm-up failure was silently discarded")
	}
}

func TestPgPerfR02_PhaseDeadline_CancelsInflight(t *testing.T) {
	h := &pgPerfHarness{ctx: context.Background()}
	started := time.Now()
	result := h.pgPerfPhase(5*time.Millisecond, 1, true, 0, func(ctx context.Context, _ int) (bool, error) {
		timer := time.NewTimer(80 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-timer.C:
			return true, nil
		}
	})
	if elapsed := time.Since(started); elapsed > 40*time.Millisecond {
		t.Fatalf("phase deadline failed to cancel in-flight operation: elapsed=%s", elapsed)
	}
	if !result.completed || result.cutoffs != 1 {
		t.Fatalf("deadline result: completed=%t cutoffs=%d", result.completed, result.cutoffs)
	}
}

func TestPgPerfR02_FinalPhaseCancellation_IsIncomplete(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	timer := time.AfterFunc(2*time.Millisecond, cancel)
	defer timer.Stop()
	h := &pgPerfHarness{ctx: ctx}
	result := h.pgPerfPhase(40*time.Millisecond, 1, true, 2, func(context.Context, int) (bool, error) {
		return true, nil
	})
	if result.completed || !result.canceled {
		t.Fatalf("final-repeat cancellation accepted: completed=%t canceled=%t", result.completed, result.canceled)
	}
}

func TestPgPerfR02_DiskProbe_StageTimings(t *testing.T) {
	evidence := pgPerfGlobalEvidence(t)
	t.Cleanup(func() { evidence.Finalize(t) })
	h := newPgPerfHarness(t, pgPerfTierSmall, "r2dsk", func(options *pgPerfOptions) { options.skipSeed = true })
	for sample := 0; sample < 3; sample++ {
		ctx, cancel := context.WithTimeout(h.ctx, 5*time.Second)
		started := time.Now()
		conn, err := h.probe.Acquire(ctx)
		acquire := time.Since(started)
		if err != nil {
			cancel()
			t.Fatalf("disk diagnostic acquire_ms=%d err=%T", acquire.Milliseconds(), err)
		}
		var one int
		queryStarted := time.Now()
		err = conn.QueryRow(ctx, "SELECT 1").Scan(&one)
		ping := time.Since(queryStarted)
		var size int64
		queryStarted = time.Now()
		if err == nil {
			err = conn.QueryRow(ctx, "SELECT pg_database_size(current_database())").Scan(&size)
		}
		query := time.Since(queryStarted)
		conn.Release()
		deadlineErr := ctx.Err()
		cancel()
		record := map[string]any{"sample": sample, "acquire_ms": acquire.Milliseconds(), "ping_ms": ping.Milliseconds(),
			"pg_database_size_ms": query.Milliseconds(), "database_bytes": size, "error_class": pgPerfErrorClass(err),
			"context_error": pgPerfErrorClass(deadlineErr), "acquired_after_release": h.probe.Stat().AcquiredConns()}
		evidence.writeJSON(filepath.Join("diagnosis", fmt.Sprintf("disk-%d.json", sample)), record)
		t.Logf("disk_diagnostic=%v", record)
		if err != nil {
			t.Fatalf("disk stage diagnostic failed: %T", err)
		}
	}
}

func TestPgPerfR02_FaultMigration_PreservesCreatedSchema(t *testing.T) {
	pgPerfRequireFaultMode(t)
	e := pgPerfGlobalEvidence(t)
	t.Cleanup(func() { e.Finalize(t) })
	newPgPerfHarness(t, pgPerfTierSmall, "r2mig", func(o *pgPerfOptions) { o.skipSeed = true; o.failMigration = 3 })
	t.Fatal("fault injection did not stop migration")
}

func TestPgPerfR02_FaultPGData_StopsBeforeDDL(t *testing.T) {
	pgPerfRequireFaultMode(t)
	e := pgPerfGlobalEvidence(t)
	t.Cleanup(func() { e.Finalize(t) })
	newPgPerfHarness(t, pgPerfTierSmall, "r2vol", func(o *pgPerfOptions) {
		o.skipSeed = true
		o.diskProbe = func(string) (int64, string) { return 0, "synthetic_target_volume_unavailable" }
	})
	t.Fatal("unavailable target volume did not stop DDL")
}

func TestPgPerfR02_FaultLeakedTransaction_BoundedCleanup(t *testing.T) {
	pgPerfRequireFaultMode(t)
	e := pgPerfGlobalEvidence(t)
	t.Cleanup(func() { e.Finalize(t) })
	h := newPgPerfHarness(t, pgPerfTierSmall, "r2leak", func(o *pgPerfOptions) { o.skipSeed = true })
	tx, err := h.beginOwned(h.ctx)
	if err != nil {
		t.Fatal(err)
	}
	var one int
	if err := tx.QueryRow(h.ctx, "SELECT 1").Scan(&one); err != nil {
		t.Fatal(err)
	}
	t.Log("synthetic owned transaction intentionally left for the closure guard")
}

func TestPgPerfR02_FaultSeedDisk_CancelsDuringSeed(t *testing.T) {
	pgPerfRequireFaultMode(t)
	e := pgPerfGlobalEvidence(t)
	t.Cleanup(func() { e.Finalize(t) })
	newPgPerfHarness(t, pgPerfTierSmall, "r2seed", func(o *pgPerfOptions) {
		o.skipSeed = true
		o.testDiskLimitDelta = 2 * 1024 * 1024
		o.beforeSeed = func(h *pgPerfHarness) {
			stageStarted := time.Now()
			seedFinished, loopStarted := time.Time{}, time.Time{}
			type batchRecord struct {
				Sequence          int
				Rows              int64
				Started, Finished time.Time
			}
			var batches []batchRecord
			var rows int64
			defer func() {
				finished := time.Now()
				h.resource.mu.Lock()
				terminal, failedOwn, last := h.resource.terminal, h.resource.failedOwn, h.resource.last
				h.resource.mu.Unlock()
				rootCanceled := h.ctx.Err() != nil
				reached := pgPerfSeedGuardReached(rootCanceled, failedOwn, terminal)
				kind := "other"
				var cap *pgPerfResourceCapError
				switch {
				case errors.As(terminal, &cap):
					kind = "cap"
				case errors.Is(terminal, context.DeadlineExceeded):
					kind = "deadline"
				case errors.Is(terminal, context.Canceled):
					kind = "canceled"
				case terminal != nil && terminal.Error() == "resource freshness expired":
					kind = "freshness"
				case terminal == nil:
					kind = "none"
				}
				observed := int64(0)
				if cap != nil {
					observed = cap.Observed
				}
				h.evidence.writeJSONExclusive("diagnosis/seed-guard-batches.json", map[string]any{"batches": len(batches), "rows": rows, "batch_boundaries": batches, "root_canceled": rootCanceled, "guard_failed": failedOwn, "target_reached": reached, "termination_kind": kind, "terminal_class": pgPerfErrorClass(terminal), "stage_started_utc": stageStarted.UTC(), "seed_finished_utc": seedFinished.UTC(), "loop_started_utc": loopStarted.UTC(), "finished_utc": finished.UTC(), "loop_limit_seconds": 60, "initial_database_bytes": h.baselineDBBytes, "target_database_bytes": h.resourceTierBytes + h.tier.diskLimit, "observed_cap_bytes": observed, "last_complete_success": last})
				if !reached {
					t.Errorf("seed guard cap target NOT REACHED: termination=%s batches=%d rows=%d", kind, len(batches), rows)
				} else {
					t.Error("expected seed guard cap target reached; retained negative exit")
				}
			}()
			pgPerfSeed(h, pgPerfSeedConfig{Label: "r02-seed-guard", Users: 100, Articles: 1000, BodyMin: 1024, BodyMax: 2048})
			seedFinished = time.Now()
			ctx, cancel := context.WithTimeout(h.ctx, 60*time.Second)
			defer cancel()
			loopStarted = time.Now()
			for ctx.Err() == nil {
				if err := h.resource.admit(ctx); err != nil {
					break
				}
				started := time.Now()
				tag, err := h.pool.Exec(ctx, `INSERT INTO articles SELECT gen_random_uuid(),author_id,orphaned_from_user_id,title,body,
					publication_state,violation_state,violation_category,deletion_state,deletion_operation_id,version,created_at,updated_at
					FROM articles WHERE deletion_state='active' LIMIT 10`)
				if err != nil {
					break
				}
				rows += tag.RowsAffected()
				batches = append(batches, batchRecord{Sequence: len(batches) + 1, Rows: tag.RowsAffected(), Started: started.UTC(), Finished: time.Now().UTC()})
			}
		}
	})
}

func TestPgPerfR02_RealSemantics_Directed(t *testing.T) {
	e := pgPerfGlobalEvidence(t)
	t.Cleanup(func() { e.Finalize(t) })
	cases := []struct {
		name, code string
		notices    int
		run        func(context.Context, *pgPerfHarness) error
	}{
		{"fresh-t1", "r2t1", 1, func(ctx context.Context, h *pgPerfHarness) error {
			for range 3 {
				if err := pgPerfFreshT1(ctx, h); err != nil {
					return err
				}
			}
			if h.scalar("SELECT count(*) FROM deletion_operations") != 3 {
				return errors.New("fresh T1 did not create three distinct operations")
			}
			return nil
		}},
		{"moderation-race", "r2e02", 1, pgPerfE02Iteration},
		{"delete-race", "r2d02", 1, pgPerfD02Iteration},
		{"cross-page", "r2r05", 1, pgPerfCrossPageIteration},
		{"late-notice-results", "r2n04", 1, func(ctx context.Context, h *pgPerfHarness) error {
			if _, err := h.pool.Exec(ctx, `UPDATE transfer_notice_recipients SET status='accepted',attempt_count=1,last_attempt_at=created_at,accepted_at=clock_timestamp()`); err != nil {
				return err
			}
			return pgPerfLeaseCycle(ctx, t, h)
		}},
		{"three-lock-chain", "r2x08", 1, pgPerfLockChain},
		{"production-read-plans", "r2plan", 1, func(ctx context.Context, h *pgPerfHarness) error {
			h.captureReadPlans()
			plans, _ := filepath.Glob(filepath.Join(h.evidence.root, "plans", h.caseCode+"-*.json"))
			if len(plans) != 3 {
				return fmt.Errorf("read plans=%d want3", len(plans))
			}
			return nil
		}},
		{"initial-drain-1000", "r2n3a", 750, func(ctx context.Context, h *pgPerfHarness) error { return pgPerfVerifyInitialDrain(ctx, h, 1000) }},
		{"initial-drain-10000", "r2n3b", 7500, func(ctx context.Context, h *pgPerfHarness) error { return pgPerfVerifyInitialDrain(ctx, h, 10000) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newPgPerfHarness(t, pgPerfTierSmall, tc.code, func(o *pgPerfOptions) {
				o.directedCase = tc.name
				o.seedMode = "minimal"
				o.seedConfig = pgPerfSeedOverride(pgPerfTierSmall, "minimal", func(c *pgPerfSeedConfig) {
					c.Users = 20
					c.Articles = 300
					c.Notices = tc.notices
					c.Deletions = 0
					c.PendingOverride = 0
					c.HiddenDensity = 0
					c.Audits = 0
				})
			})
			if err := tc.run(h.ctx, h); err != nil {
				t.Fatal(err)
			}
			t.Logf("directed_semantics=%s schema=%s", tc.name, h.schema)
		})
	}
}

func pgPerfVerifyInitialDrain(ctx context.Context, h *pgPerfHarness, expected int64) error {
	before := h.scalar("SELECT count(*) FROM transfer_notice_recipients WHERE status='accepted'")
	var sent atomic.Int64
	dispatcher, err := pgPerfNoticeDispatcher(h, pgPerfUUID(h.schema, "drain"), noticeSendFunc(func(context.Context, mailer.Message) error { sent.Add(1); return nil }), 50)
	if err != nil {
		return err
	}
	for round := int64(0); round < expected/50+5; round++ {
		if err := dispatcher.Reconcile(ctx); err != nil {
			return err
		}
		if h.scalar("SELECT count(*) FROM transfer_notice_recipients WHERE status NOT IN ('accepted','failed_terminal')") == 0 {
			break
		}
	}
	accepted := h.scalar("SELECT count(*) FROM transfer_notice_recipients WHERE status='accepted'") - before
	failed := h.scalar("SELECT count(*) FROM transfer_notice_recipients WHERE status='failed_terminal'")
	if err := pgPerfInitialDrainOutcome(1, accepted, expected, failed); err != nil {
		return err
	}
	if sent.Load() != expected {
		return fmt.Errorf("initial drain sent=%d expected=%d", sent.Load(), expected)
	}
	h.evidence.writeJSONExclusive(filepath.Join("regressions", h.schema+".initial-drain.json"), map[string]any{"expected": expected, "accepted": accepted, "sent": sent.Load(), "failed_terminal": failed})
	return nil
}

func pgPerfRequireFaultMode(t *testing.T) {
	t.Helper()
	if os.Getenv("HAN_PG_PERF_INJECT_FAULT") != "1" {
		t.Skip("explicit diagnostic fault mode required; this is an expected-failure test")
	}
}

func TestPgPerfR02_FaultMeasurement_FailsAfterWarmup(t *testing.T) {
	pgPerfRequireFaultMode(t)
	e := pgPerfGlobalEvidence(t)
	t.Cleanup(func() { e.Finalize(t) })
	t.Run("measurement-error", func(t *testing.T) {
		h := newPgPerfHarness(t, pgPerfTierSmall, "r2ferr", func(o *pgPerfOptions) { o.seedMode = "minimal" })
		var measured int
		h.pgPerfRunPoint(pgPerfTimedSpec("R02-fault-measurement", "R02", 1, nil, "diagnostic correctness error at unified runner; never a performance acceptance point"), func(ctx context.Context, _ int) (bool, error) {
			deadline, ok := ctx.Deadline()
			if !ok {
				return false, errors.New("missing controlled phase deadline")
			}
			if time.Until(deadline) > pgPerfMinWarmup {
				measured++
				if measured == 2 {
					return false, errors.New("injected E02 correctness assertion after measurement success")
				}
			}
			var value int
			err := h.pool.QueryRow(ctx, "SELECT 1").Scan(&value)
			return err == nil && value == 1, err
		})
	})
	t.Run("must-not-create-schema", func(t *testing.T) {
		newPgPerfHarness(t, pgPerfTierSmall, "r2aft", func(o *pgPerfOptions) { o.skipSeed = true })
	})
}

func TestPgPerfR02_Outcome_RejectsErrorsTruncationAndTail(t *testing.T) {
	base := pgPerfPointSummary{CompletedRepeats: 3, Successes: 2, N: 2}
	for _, tc := range []struct {
		name   string
		change func(*pgPerfPointSummary)
		wall   time.Duration
	}{
		{"partial-success-then-error", func(s *pgPerfPointSummary) { s.Errors = 1 }, 120 * time.Second},
		{"missing-final-repeat", func(s *pgPerfPointSummary) { s.CompletedRepeats = 2 }, 120 * time.Second},
		{"truncated", func(s *pgPerfPointSummary) { s.Truncated = true }, 120 * time.Second},
		{"excess-tail", func(s *pgPerfPointSummary) { s.TailNS = int64(time.Second) + 1 }, 121 * time.Second},
		{"excess-wall", func(s *pgPerfPointSummary) {}, 121*time.Second + 1},
		{"zero-valid-success", func(s *pgPerfPointSummary) { s.Successes = 0; s.N = 0 }, 120 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := base
			tc.change(&s)
			if pgPerfPointOutcome(s, 3, tc.wall, 121*time.Second) == nil {
				t.Fatal("nonconforming point accepted")
			}
		})
	}
	base.TailNS = int64(3 * time.Millisecond)
	if err := pgPerfPointOutcome(base, 3, 120*time.Second+3*time.Millisecond, 121*time.Second); err != nil {
		t.Fatalf("adopted finite scheduling tail rejected: %v", err)
	}
}

func TestPgPerfR02_PhaseBoundary_MixedBusinessErrorFails(t *testing.T) {
	h := &pgPerfHarness{ctx: context.Background()}
	result := h.pgPerfPhase(5*time.Millisecond, 1, true, 0, func(ctx context.Context, _ int) (bool, error) {
		<-ctx.Done()
		return false, errors.Join(ctx.Err(), errors.New("business assertion failed"))
	})
	if result.errors != 1 || result.cutoffs != 0 || result.successes != 0 {
		t.Fatalf("business error swallowed as cutoff: %+v", result)
	}
}

func TestPgPerfR02_PhaseBoundary_UnknownWriteFails(t *testing.T) {
	h := &pgPerfHarness{ctx: context.Background(), pool: &pgxpool.Pool{}}
	result := h.pgPerfPhase(5*time.Millisecond, 1, true, 0, func(ctx context.Context, _ int) (bool, error) { <-ctx.Done(); return false, ctx.Err() })
	if result.errors != 1 || result.unknown != 1 || result.cutoffs != 0 || result.successes != 0 || result.samples[0].Error != "unknown_write_outcome" {
		t.Fatalf("unknown write accepted: %+v", result)
	}
}

func TestPgPerfR02_PhaseBoundary_UncooperativeWorkerFailsAndJoins(t *testing.T) {
	if root := os.Getenv("HAN_PG_PERF_FAILSTOP_CHILD_DIR"); root != "" {
		h := &pgPerfHarness{ctx: context.Background(), t: t, evidence: &pgPerfEvidence{root: root, runID: "owned-unit-failstop"}}
		h.pgPerfPhase(5*time.Millisecond, 1, true, 0, func(context.Context, int) (bool, error) { select {} })
		t.Fatal("uncooperative worker escaped fail-stop")
	}
	parent := filepath.Join("..", "..", "..", ".spec", "wip", "article-management", "evidence-pg-perf-r04", "continuation-20261104", "author", "r02", "unit-fixtures")
	if err := os.MkdirAll(parent, 0755); err != nil {
		t.Fatal(err)
	}
	root, err := os.MkdirTemp(parent, "failstop-")
	if err != nil {
		t.Fatal(err)
	}
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, binary, "-test.run=^TestPgPerfR02_PhaseBoundary_UncooperativeWorkerFailsAndJoins$", "-test.v")
	child.Env = append(os.Environ(), "HAN_PG_PERF_FAILSTOP_CHILD_DIR="+root)
	started := time.Now()
	output, err := child.CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 86 || ctx.Err() != nil {
		t.Fatalf("owned fail-stop child did not exit86 within bound: %v %s", err, output)
	}
	files, _ := filepath.Glob(filepath.Join(root, "failures", "*.json"))
	if len(files) != 1 {
		t.Fatal("missing immutable fail-stop evidence")
	}
	t.Logf("owned_child_pid=%d actual_wait_exit=86 elapsed_ns=%d evidence=%s", child.Process.Pid, time.Since(started).Nanoseconds(), files[0])
}

func TestPgPerfR02_ImmutableWorkloads_KeepBothSchemas(t *testing.T) {
	parent := filepath.Join("..", "..", "..", ".spec", "wip", "article-management", "evidence-pg-perf-r04", "continuation-20261104", "author", "r02", "unit-fixtures")
	if err := os.MkdirAll(parent, 0755); err != nil {
		t.Fatal(err)
	}
	root, err := os.MkdirTemp(parent, "workload-")
	if err != nil {
		t.Fatal(err)
	}
	e := &pgPerfEvidence{root: root}
	a, b := filepath.Join("schema-a", "batch1.json"), filepath.Join("schema-b", "batch1.json")
	e.writeJSONExclusive(a, map[string]int{"rows": 1000})
	e.writeJSONExclusive(b, map[string]int{"rows": 10000})
	before, err := os.ReadFile(e.path(a))
	if err != nil {
		t.Fatal(err)
	}
	e.writeJSONExclusive(a, map[string]int{"rows": 1})
	after, err := os.ReadFile(e.path(a))
	if err != nil || string(before) != string(after) || e.finalErr == nil {
		t.Fatal("immutable workload was overwritten or duplicate was ignored")
	}
	if _, err := os.Stat(e.path(b)); err != nil {
		t.Fatal(err)
	}
}

func TestPgPerfR02_NoticeAppend_CompleteRecipients(t *testing.T) {
	e := pgPerfGlobalEvidence(t)
	t.Cleanup(func() { e.Finalize(t) })
	h := newPgPerfHarness(t, pgPerfTierSmall, "r2app", func(o *pgPerfOptions) { o.seedMode = "minimal" })
	if _, err := h.pool.Exec(h.ctx, `UPDATE transfer_notice_recipients SET status='accepted',attempt_count=1,last_attempt_at=created_at,accepted_at=clock_timestamp()`); err != nil {
		t.Fatal(err)
	}
	if _, err := h.pool.Exec(h.ctx, `CREATE TABLE pgperf_due(notice_id uuid,recipient_kind text,batch_no bigint)`); err != nil {
		t.Fatal(err)
	}
	if err := pgPerfAppendNoticeBatch(h.ctx, h, 750, 1); err != nil {
		t.Fatal(err)
	}
	if got := h.scalar("SELECT count(*) FROM pgperf_due"); got != 1000 {
		t.Fatalf("fresh due keys=%d want1000", got)
	}
	var mode, sent atomic.Int64
	dispatcher, err := pgPerfNoticeDispatcher(h, pgPerfUUID(h.schema, "append-worker"), noticeSendFunc(func(context.Context, mailer.Message) error {
		switch mode.Load() {
		case 1:
			return errors.New("synthetic unknown transport")
		case 2:
			return mailer.ErrInvalidMessage
		}
		sent.Add(1)
		return nil
	}), 50)
	if err != nil {
		t.Fatal(err)
	}
	for range 20 {
		if err := dispatcher.Reconcile(h.ctx); err != nil {
			t.Fatal(err)
		}
	}
	accepted := h.scalar(`SELECT count(*) FROM transfer_notice_recipients r JOIN pgperf_due d USING(notice_id,recipient_kind) WHERE d.batch_no=1 AND r.status='accepted'`)
	if accepted != 1000 || sent.Load() != 1000 {
		t.Fatalf("appended batch incomplete: accepted=%d sent=%d", accepted, sent.Load())
	}
	for _, batch := range []int64{2, 3} {
		if err := pgPerfAppendNoticeBatch(h.ctx, h, 3, batch); err != nil {
			t.Fatal(err)
		}
		mode.Store(batch - 1)
		if err := dispatcher.Reconcile(h.ctx); err != nil {
			t.Fatal(err)
		}
		if batch == 2 {
			if got := h.scalar("SELECT count(*) FROM transfer_notice_recipients WHERE status='retry_wait' AND next_attempt_at>clock_timestamp()"); got != 4 {
				t.Fatalf("retry rows=%d want4", got)
			}
			h.evidence.writeJSONExclusive("regressions/retry-checkpoint.json", map[string]any{"schema": h.schema, "batch": batch, "future_retry_wait": 4, "observed_at": time.Now().UTC().Format(time.RFC3339Nano)})
		}
	}
	terminalQuery := `SELECT count(*) FROM transfer_notice_recipients r JOIN pgperf_due d USING(notice_id,recipient_kind) WHERE d.batch_no=3 AND r.status='failed_terminal'`
	for round := 1; round < 8 && h.scalar(terminalQuery) < 4; round++ {
		if err := dispatcher.Reconcile(h.ctx); err != nil {
			t.Fatal(err)
		}
	}
	if got := h.scalar(terminalQuery); got != 4 {
		t.Fatalf("terminal rows=%d want4", got)
	}
	h.evidence.writeJSONExclusive("regressions/appended-batch.json", map[string]any{"schema": h.schema, "new_batch_accepted": accepted, "sent": sent.Load(), "future_retry_wait_at_checkpoint": 4, "batch3_failed_terminal": 4, "formal_performance": "NOT RUN"})
}

func TestPgPerfR02_ExpiredCompositeHelper_NoAcquire(t *testing.T) {
	e := pgPerfGlobalEvidence(t)
	t.Cleanup(func() { e.Finalize(t) })
	h := newPgPerfHarness(t, pgPerfTierSmall, "r2ctx", func(o *pgPerfOptions) { o.seedMode = "minimal" })
	before := h.pool.Stat().AcquireCount()
	ctx, cancel := context.WithDeadline(h.ctx, time.Now().Add(-time.Second))
	defer cancel()
	for name, op := range map[string]func(context.Context, *pgPerfHarness) error{"T1": pgPerfFreshT1, "E02": pgPerfE02Iteration, "D02": pgPerfD02Iteration, "R05": pgPerfCrossPageIteration, "X08": pgPerfLockChain} {
		if err := op(ctx, h); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("%s did not preserve operation deadline: %v", name, err)
		}
	}
	if after := h.pool.Stat().AcquireCount(); after != before {
		t.Fatalf("expired composite helper acquired a connection: before=%d after=%d", before, after)
	}
}

func TestPgPerfR02_FaultPhaseJoin_FailStop(t *testing.T) {
	pgPerfRequireFaultMode(t)
	h := newPgPerfHarness(t, pgPerfTierSmall, "r2stop", func(o *pgPerfOptions) { o.skipSeed = true })
	h.pgPerfPhase(5*time.Millisecond, 1, true, 0, func(context.Context, int) (bool, error) { select {} })
	t.Fatal("uncooperative owned test child returned without fail-stop")
}

func TestPgPerfR02_CanceledWorkers_ReleaseAndJoin(t *testing.T) {
	e := pgPerfGlobalEvidence(t)
	t.Cleanup(func() { e.Finalize(t) })
	h := newPgPerfHarness(t, pgPerfTierSmall, "r2join", func(o *pgPerfOptions) { o.seedMode = "minimal" })
	created, err := h.commandService().Create(h.ctx, pgPerfMemberPrincipal(h), article.CreateArticleInput{Title: "directed cancellation", Body: "body"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(h.ctx)
	ready := make(chan struct{}, 2)
	cancelerDone := make(chan struct{})
	go func() {
		defer close(cancelerDone)
		for range 2 {
			select {
			case <-ready:
			case <-ctx.Done():
				return
			}
		}
		cancel()
	}()
	action := func(ctx context.Context) error {
		tx, err := h.beginOwned(ctx)
		if err != nil {
			return err
		}
		defer func() {
			closeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = tx.Rollback(closeCtx)
		}()
		ready <- struct{}{}
		_, err = tx.Exec(ctx, "SELECT id FROM articles WHERE id=$1::uuid FOR UPDATE", created.ID)
		return err
	}
	_, raceErr := h.raceRow(ctx, created.ID, [2]func(context.Context) error{action, action})
	cancel()
	<-cancelerDone
	if raceErr == nil {
		t.Fatal("canceled race was accepted")
	}
	ctx, cancel = context.WithCancel(h.ctx)
	release := make(chan struct{})
	worker, err := pgPerfStartLockWorker(ctx, h, created.ID, release)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	select {
	case <-worker.acquired:
	case err := <-worker.done:
		cancel()
		t.Fatalf("lock worker exited early: %v", err)
	case <-ctx.Done():
		cancel()
		t.Fatal(ctx.Err())
	}
	cancel()
	select {
	case err := <-worker.done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancel release wait: %v", err)
		}
	case <-time.After(4 * time.Second):
		t.Fatal("owned release worker failed to join")
	}
	returnCtx, returnCancel := context.WithTimeout(h.ctx, 3*time.Second)
	defer returnCancel()
	if err := h.waitReturned(returnCtx); err != nil {
		t.Fatalf("owned workers did not return connections: acquired=%d err=%v", h.pool.Stat().AcquiredConns(), err)
	}
	h.evidence.writeJSONExclusive("regressions/canceled-workers.json", map[string]any{"schema": h.schema, "race_failed": raceErr != nil, "release_wait_canceled": true, "acquired": 0})
}

func TestPgPerfMedium(t *testing.T) {
	evidence := pgPerfGlobalEvidence(t)
	t.Cleanup(func() { evidence.Finalize(t) })
	runPgPerfMediumWrite(t)
	runPgPerfMediumRead(t)
	runPgPerfMediumNoticeDelete(t)
	pgPerfGlobalEvidence(t).Finalize(t)
}

func TestPgPerfLarge(t *testing.T) {
	t.Fatal("NOT RUN: Lead 2026-11-04 prioritizes small/medium within the 90 minute window")
	runPgPerfLargeWrite(t)
	runPgPerfLargeRead(t)
	runPgPerfLargeNoticeDelete(t)
	pgPerfGlobalEvidence(t).Finalize(t)
}

// BenchmarkPgPerfHarness calibrates only the timer loop; it never writes
// P50/P95/QPS scenario samples.
var pgPerfCalibrationRepeats atomic.Int64

func BenchmarkPgPerfHarness(b *testing.B) {
	repeat := pgPerfCalibrationRepeats.Add(1)
	evidence := pgPerfGlobalEvidence(b)
	b.Cleanup(func() {
		evidence.checkpointInventory()
		if b.Failed() || repeat == 3 {
			evidence.Finalize(b)
		}
	})
	h := newPgPerfHarness(b, pgPerfTierSmall, fmt.Sprintf("bc%03d", repeat), func(options *pgPerfOptions) {
		options.skipSeed = true
		options.seedMode = "empty"
		options.maxConns = 2
		if os.Getenv("HAN_PG_PERF_INJECT_FAULT") == "1" && os.Getenv("HAN_PG_PERF_CALIBRATION_FAULT") == "1" {
			options.failMigration = 3
		}
	})
	b.ResetTimer()
	for b.Loop() {
		var one int
		if err := h.pool.QueryRow(h.ctx, "SELECT 1").Scan(&one); err != nil {
			b.Fatalf("pgperf calibration query failed: %T", err)
		}
	}
}
