//go:build integration

package identity

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"han/internal/audit"
	identitypkg "han/internal/identity"
	"han/internal/migration"
	"han/internal/platform"
	"han/internal/session"
	"han/internal/testenv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// These are synthetic inputs, not production password or abuse profiles.
const (
	s33Origin     = "https://s33.example.invalid"
	s33Trace      = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	s33Password   = "synthetic-s33-password"
	s33Digest     = "synthetic-s33-credential-digest"
	s33FakeProbe  = "synthetic-s33-fake-probe"
	s33FakeDigest = "synthetic-s33-fake-digest"
)

func s33Require(t *testing.T, ok bool, label string) {
	t.Helper()
	if !ok {
		t.Fatal(label)
	}
}

func s33State(err error) string {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		return pg.Code
	}
	return ""
}

func s33Must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("operation failed type=%T sqlstate=%s", err, s33State(err))
	}
}

type s33IDs struct{}

func (s33IDs) NewID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6], b[8] = b[6]&15|64, b[8]&63|128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

func s33UUID(t *testing.T) string {
	t.Helper()
	id, err := (s33IDs{}).NewID()
	s33Must(t, err)
	return id
}

type s33Clock struct{ at time.Time }

func (c *s33Clock) Now() time.Time { return c.at }

type s33Hasher struct{}

func (s33Hasher) Verify(password, digest string) (bool, bool, error) {
	switch digest {
	case s33Digest:
		return password == s33Password, false, nil
	case s33FakeDigest:
		return password == s33FakeProbe, false, nil
	default:
		return false, false, errors.New("unexpected synthetic credential")
	}
}

func (s33Hasher) Hash(string) (string, error) {
	return "", errors.New("unexpected synthetic rehash")
}

type s33Observer struct {
	last identitypkg.LoginObservation
}

func (o *s33Observer) ObserveFinal(_ context.Context, value identitypkg.LoginObservation) error {
	o.last = value
	return nil
}

func (*s33Observer) ObserveRejected(context.Context, platform.OriginRejectionObservation) error {
	return nil
}

type s33Issuer struct {
	inner  identitypkg.LoginSessionIssuer
	inputs []session.CreateInput
}

func (i *s33Issuer) Issue(u identitypkg.User, kind session.CredentialKind, credential *string) (session.CreateInput, error) {
	input, err := i.inner.Issue(u, kind, credential)
	if err == nil {
		i.inputs = append(i.inputs, input)
	}
	return input, err
}

// These adapters delegate every SQL operation to PG. They only record ownership.
type s33Tx struct {
	pgx.Tx
	pid     uint32
	mu      sync.Mutex
	ended   bool
	outcome string
}

func (tx *s33Tx) Commit(ctx context.Context) error {
	err := tx.Tx.Commit(ctx)
	tx.mu.Lock()
	defer tx.mu.Unlock()
	tx.ended = true
	tx.outcome = "committed"
	if err != nil {
		tx.outcome = "commit_error"
	}
	return err
}

func (tx *s33Tx) Rollback(ctx context.Context) error {
	err := tx.Tx.Rollback(ctx)
	tx.mu.Lock()
	defer tx.mu.Unlock()
	if !tx.ended {
		tx.ended = err == nil || errors.Is(err, pgx.ErrTxClosed)
		tx.outcome = "rolled_back"
		if !tx.ended {
			tx.outcome = "rollback_error"
		}
	}
	return err
}

type s33DB struct {
	pool *pgxpool.Pool
	mu   sync.Mutex
	seen []*s33Tx
}

func (d *s33DB) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return d.pool.QueryRow(ctx, query, args...)
}

func (d *s33DB) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	wrapped := &s33Tx{Tx: tx, pid: tx.Conn().PgConn().PID()}
	d.mu.Lock()
	d.seen = append(d.seen, wrapped)
	d.mu.Unlock()
	return wrapped, nil
}

type s33Image map[string]string

type s33Query interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type s33Audit struct {
	f       *s33Fixture
	state   string
	fault   bool
	eventID string
	staged  s33Image
}

func (a *s33Audit) Append(ctx context.Context, tx pgx.Tx, event audit.Event) error {
	a.eventID = event.ID
	image, err := s33ReadImage(ctx, tx, a.f.schema)
	if err != nil {
		return err
	}
	a.staged = image
	if err := a.f.saveImage("business-pre-audit", image); err != nil {
		return err
	}
	err = (audit.Store{}).Append(ctx, tx, event)
	a.state = s33State(err)
	a.fault = s33InjectedFault(err, a.f.schema, "audit_events")
	return err
}

type s33Fixture struct {
	t                  *testing.T
	ctx                context.Context
	cancel             context.CancelFunc
	schema, dir        string
	pool, observerPool *pgxpool.Pool
	observer           *pgxpool.Conn
	db                 *s33DB
	clock              *s33Clock
	sessions           session.Store
	issuer             *s33Issuer
	source             *identitypkg.TrustedSourceEvidenceProvider
	digester           *identitypkg.HMACLoginAbuseDigester
	protector          *identitypkg.PersistentLoginAbuseProtector
	abuseKey           []byte
	handler            *identitypkg.Handler
	guard              platform.OriginGuard
	observation        *s33Observer
	auditor            *s33Audit
	workers            sync.WaitGroup
	evidenceMu         sync.Mutex
	sequence           int
	created            bool
	poolClosed         bool
}

func s33Policy() identitypkg.LoginAbuseConfig {
	return identitypkg.LoginAbuseConfig{Window: 10 * time.Minute, FailureThreshold: 2,
		BlockDuration: 5 * time.Minute, SuccessAction: identitypkg.LoginAbuseSuccessDecay, SuccessDecay: 1}
}

func s33Endpoint(host string, port uint16) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback() && port == 5433
}

func s33Config(t *testing.T, schema string) *pgxpool.Config {
	t.Helper()
	dsn := testenv.RequirePostgresDSN(t)
	forbidden := regexp.MustCompile(`(?i)(^|[?&\s])(service|servicefile|passfile|options|role)\s*=`)
	s33Require(t, !forbidden.MatchString(dsn), "uncontrolled connection option")
	c, err := pgxpool.ParseConfig(dsn)
	s33Must(t, err)
	s33Require(t, s33Endpoint(c.ConnConfig.Host, c.ConnConfig.Port), "requires literal loopback:5433")
	for _, fallback := range c.ConnConfig.Fallbacks {
		s33Require(t, s33Endpoint(fallback.Host, fallback.Port), "unsafe fallback endpoint")
	}
	s33Require(t, c.ConnConfig.User != "" && c.ConnConfig.Database != "", "explicit identity/database required")
	for key := range c.ConnConfig.RuntimeParams {
		s33Require(t, key == "application_name", "uncontrolled runtime parameter")
	}
	c.MaxConns, c.MinConns = 6, 0
	c.ConnConfig.RuntimeParams = map[string]string{"search_path": pgx.Identifier{schema}.Sanitize(), "application_name": "han_s33_test"}
	c.ConnConfig.Tracer = nil
	c.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := s33Preflight(ctx, conn, c, schema, false)
		return err
	}
	return c
}

func s33Preflight(ctx context.Context, q s33Query, c *pgxpool.Config, schema string, exists bool) (time.Time, error) {
	var host, user, database, path, owner string
	var currentSchema *string
	var port, version int
	var now time.Time
	err := q.QueryRow(ctx, `SELECT host(inet_server_addr()),inet_server_port(),
		current_setting('server_version_num')::int,current_user,current_database(),
		current_setting('search_path'),COALESCE((SELECT pg_get_userbyid(nspowner)
		FROM pg_namespace WHERE nspname=$1),''),current_schema(),clock_timestamp()`, schema).
		Scan(&host, &port, &version, &user, &database, &path, &owner, &currentSchema, &now)
	if err != nil {
		return time.Time{}, err
	}
	if !s33Endpoint(host, uint16(port)) || version != 180006 || user != c.ConnConfig.User ||
		database != c.ConnConfig.Database || path != (pgx.Identifier{schema}).Sanitize() {
		return time.Time{}, errors.New("endpoint/version/identity/database/search_path mismatch")
	}
	if (owner != "" && owner != user) || (exists && owner != user) {
		return time.Time{}, errors.New("test schema ownership mismatch")
	}
	if exists && (currentSchema == nil || *currentSchema != schema) {
		return time.Time{}, errors.New("current schema namespace mismatch")
	}
	return now.UTC().Truncate(time.Microsecond), nil
}

func s33New(t *testing.T) *s33Fixture {
	t.Helper()
	f := &s33Fixture{t: t, schema: "han_s33_" + strings.ReplaceAll(s33UUID(t), "-", "")}
	f.ctx, f.cancel = context.WithTimeout(context.Background(), 90*time.Second)
	t.Cleanup(f.close)
	var err error
	f.dir, err = os.MkdirTemp("", f.schema+"_") // Retained; never t.TempDir/RemoveAll.
	s33Must(t, err)
	t.Logf("schema=%s retained_dir=%s", f.schema, f.dir)
	c := s33Config(t, f.schema)
	f.pool, err = pgxpool.NewWithConfig(f.ctx, c)
	s33Must(t, err)
	now, err := s33Preflight(f.ctx, f.pool, c, f.schema, false)
	s33Must(t, err)
	f.clock = &s33Clock{at: now}
	f.createAndMigrate(c)
	f.observerPool, err = pgxpool.NewWithConfig(f.ctx, c.Copy())
	s33Must(t, err)
	f.observer, err = f.observerPool.Acquire(f.ctx)
	s33Must(t, err)
	_, err = s33Preflight(f.ctx, f.observer, c, f.schema, true)
	s33Must(t, err)
	f.ctx, err = platform.ContextWithTraceID(f.ctx, s33Trace)
	s33Must(t, err)
	f.db = &s33DB{pool: f.pool}
	f.compose()
	f.snapshot("migrated-empty")
	return f
}

func (f *s33Fixture) createAndMigrate(c *pgxpool.Config) {
	tx, err := f.pool.Begin(f.ctx)
	s33Must(f.t, err)
	defer s33Rollback(f.t, tx)
	_, err = tx.Exec(f.ctx, "CREATE SCHEMA "+pgx.Identifier{f.schema}.Sanitize()+" AUTHORIZATION CURRENT_USER")
	s33Must(f.t, err)
	s33Must(f.t, tx.Commit(f.ctx))
	f.created = true
	_, err = s33Preflight(f.ctx, f.pool, c, f.schema, true)
	s33Must(f.t, err)
	items, err := s33FrozenIdentityMigrations(os.DirFS(filepath.Join("..", "..", "..")))
	s33Must(f.t, err)
	s33Require(f.t, len(items) == 11, "expected frozen eleven migrations")
	tx, err = f.pool.Begin(f.ctx)
	s33Must(f.t, err)
	defer s33Rollback(f.t, tx)
	for i, item := range items {
		s33Require(f.t, item.Version == i+1, "migration sequence mismatch")
		_, err = tx.Exec(f.ctx, item.SQL)
		s33Must(f.t, err)
		f.t.Logf("schema=%s migration=%s stage=applied", f.schema, item.Name)
	}
	s33Must(f.t, tx.Commit(f.ctx))
	f.t.Logf("schema=%s create_committed=true migrations_committed=11", f.schema)
}

func (f *s33Fixture) compose() {
	var err error
	f.abuseKey = make([]byte, 32)
	_, err = rand.Read(f.abuseKey)
	s33Must(f.t, err)
	f.digester, err = identitypkg.NewHMACLoginAbuseDigester("s33_abuse_v1", f.abuseKey)
	s33Must(f.t, err)
	f.protector, err = identitypkg.NewPersistentLoginAbuseProtector(identitypkg.PostgresLoginAbuseBucketStore{}, f.digester, s33Policy())
	s33Must(f.t, err)
	key := make([]byte, 32)
	_, err = rand.Read(key)
	s33Must(f.t, err)
	f.source, err = identitypkg.NewTrustedSourceEvidenceProvider(key, "s33_source_v1", nil)
	clear(key)
	s33Must(f.t, err)
	gate := &identitypkg.PasswordResetStore{}
	f.sessions, err = session.NewStore(session.StoreDependencies{Transactions: f.db, PasswordResetGate: gate})
	s33Must(f.t, err)
	issuer, err := identitypkg.NewRandomLoginSessionIssuer(rand.Reader, s33IDs{}, f.clock, 32, 32, 8*time.Hour, time.Hour)
	s33Must(f.t, err)
	f.issuer = &s33Issuer{inner: issuer}
	f.observation = &s33Observer{}
	f.auditor = &s33Audit{f: f}
	f.composeHandler(gate)
}

func (f *s33Fixture) composeHandler(gate *identitypkg.PasswordResetStore) {
	mode, err := identitypkg.NewConfiguredLoginRecoveryMode("normal")
	s33Must(f.t, err)
	service, err := identitypkg.NewLoginService(identitypkg.LoginDependencies{
		Transactions: f.db, Repository: identitypkg.PostgresLoginRepository{}, PasswordResetGate: gate,
		Passwords: s33Hasher{}, Sessions: f.sessions, SessionIssuer: f.issuer, Abuse: f.protector,
		RecoveryMode: mode, Audit: f.auditor, IDs: s33IDs{}, Clock: f.clock,
		FakeCredentialDigest: s33FakeDigest, FakeCredentialProbe: s33FakeProbe})
	s33Must(f.t, err)
	failure, err := identitypkg.NewPostgresLoginFailureAuditor(f.db, audit.Store{})
	s33Must(f.t, err)
	coordinator, err := identitypkg.NewLoginAttemptCoordinator(s33IDs{}, f.clock, failure)
	s33Must(f.t, err)
	f.handler, err = identitypkg.NewLoginHandler(service, f.source, 4096, identitypkg.LoginHandlerConfig{
		Coordinator: coordinator, Observer: f.observation, ObservationBudget: time.Second})
	s33Must(f.t, err)
	f.guard, err = platform.NewOriginGuard(s33Origin, f.observation, time.Second)
	s33Must(f.t, err)
}

func s33Rollback(t *testing.T, tx pgx.Tx) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := tx.Rollback(ctx)
	if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		t.Errorf("rollback failed type=%T sqlstate=%s", err, s33State(err))
	}
}

func (f *s33Fixture) begin() *s33Tx {
	tx, err := f.db.Begin(f.ctx)
	s33Must(f.t, err)
	return tx.(*s33Tx)
}

func (f *s33Fixture) closeTransactions() {
	if f.db == nil {
		return
	}
	f.db.mu.Lock()
	txs := append([]*s33Tx(nil), f.db.seen...)
	f.db.mu.Unlock()
	for _, tx := range txs {
		tx.mu.Lock()
		ended := tx.ended
		tx.mu.Unlock()
		if !ended {
			s33Rollback(f.t, tx)
		}
		tx.mu.Lock()
		f.t.Logf("schema=%s tx_pid=%d ended=%t outcome=%s", f.schema, tx.pid, tx.ended, tx.outcome)
		if !tx.ended {
			f.t.Error("transaction was not closed")
		}
		tx.mu.Unlock()
	}
}

func (f *s33Fixture) close() {
	f.cancel()
	f.closeTransactions()
	f.workers.Wait()
	if f.observer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		image, err := s33ReadImage(ctx, f.observer, f.schema)
		if err == nil {
			err = f.saveImage("final-retained", image)
		}
		if err != nil {
			f.t.Errorf("final evidence failure type=%T sqlstate=%s", err, s33State(err))
		}
		cancel()
		f.observer.Release()
	}
	f.closePool(f.observerPool)
	if !f.poolClosed {
		f.closePool(f.pool)
	}
	clear(f.abuseKey)
	f.t.Logf("schema=%s created_here=%t retain_policy=always workers_joined=true pools_closed=true", f.schema, f.created)
}

func (f *s33Fixture) closePool(pool *pgxpool.Pool) {
	if pool == nil {
		return
	}
	if pool.Stat().AcquiredConns() != 0 {
		f.t.Error("pool has acquired connection at close")
	}
	pool.Close()
}

func s33Tables() []string {
	return []string{"processing_gate_records", "audit_events", "users", "identity_identifiers", "sessions",
		"temporary_credentials", "privacy_notices", "privacy_acknowledgements", "administrator_information_declarations",
		"login_attempt_buckets", "recovery_account_reviews", "deletion_operations", "articles", "transfer_notices",
		"transfer_notice_recipients", "deletion_replay_runs", "deletion_replay_evidence", "idempotency_records",
		"recovery_runs", "password_reset_operations", "password_reset_recovery_jobs"}
}

func s33ImageSQL(schema, table string) string {
	return `SELECT COALESCE(jsonb_agg(v ORDER BY v::text),'[]'::jsonb)::text FROM
		(SELECT jsonb_build_object('row',to_jsonb(r),'xmin',r.xmin::text,'ctid',r.ctid::text) v FROM ` +
		pgx.Identifier{schema, table}.Sanitize() + ` r) images`
}

func s33ReadImage(ctx context.Context, q s33Query, schema string) (s33Image, error) {
	image := s33Image{}
	for _, table := range s33Tables() {
		var value string
		if err := q.QueryRow(ctx, s33ImageSQL(schema, table)).Scan(&value); err != nil {
			return nil, err
		}
		image[table] = value
	}
	return image, nil
}

func (f *s33Fixture) snapshot(stage string) s33Image {
	tx, err := f.observer.BeginTx(f.ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	s33Must(f.t, err)
	defer s33Rollback(f.t, tx)
	image, err := s33ReadImage(f.ctx, tx, f.schema)
	s33Must(f.t, err)
	s33Must(f.t, tx.Commit(f.ctx))
	s33Must(f.t, f.saveImage(stage, image))
	return image
}

func s33Redact(value any) any {
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			if item != nil && (strings.Contains(key, "digest") || strings.Contains(key, "fingerprint")) {
				encoded, _ := json.Marshal(item)
				v[key] = map[string]string{"sha256_json_value": fmt.Sprintf("%x", sha256.Sum256(encoded))}
			} else {
				v[key] = s33Redact(item)
			}
		}
	case []any:
		for i, item := range v {
			v[i] = s33Redact(item)
		}
	}
	return value
}

func (f *s33Fixture) saveImage(stage string, image s33Image) error {
	f.evidenceMu.Lock()
	defer f.evidenceMu.Unlock()
	tables := map[string]any{}
	for _, table := range s33Tables() {
		var rows any
		decoder := json.NewDecoder(strings.NewReader(image[table]))
		decoder.UseNumber()
		if err := decoder.Decode(&rows); err != nil {
			return err
		}
		redacted, err := json.Marshal(s33Redact(rows))
		if err != nil {
			return err
		}
		tables[table] = map[string]any{"sql_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(s33ImageSQL(f.schema, table)))),
			"raw_rows_sha256":      fmt.Sprintf("%x", sha256.Sum256([]byte(image[table]))),
			"redacted_rows_sha256": fmt.Sprintf("%x", sha256.Sum256(redacted)), "redacted_rows": json.RawMessage(redacted)}
	}
	f.sequence++
	name := fmt.Sprintf("%03d-%s.json", f.sequence, stage)
	return s33WriteJSON(filepath.Join(f.dir, name), map[string]any{"leaf": f.t.Name(), "schema": f.schema, "stage": stage, "tables": tables})
}

func s33WriteJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(raw)
	return errors.Join(writeErr, file.Close())
}

func s33SameTables(t *testing.T, a, b s33Image, except ...string) {
	t.Helper()
	for _, table := range s33Tables() {
		skip := false
		for _, name := range except {
			if name == table {
				skip = true
			}
		}
		if !skip {
			s33Require(t, a[table] == b[table], "unexpected committed table image change: "+table)
		}
	}
}

type s33User struct {
	id, name string
	old      session.CreateInput
}

func (f *s33Fixture) seedUser(withSession bool) s33User {
	u := s33User{id: s33UUID(f.t)}
	u.name = "s33-" + u.id
	tx := f.begin()
	defer s33Rollback(f.t, tx)
	_, err := tx.Exec(f.ctx, `INSERT INTO users(id,username,email,phone,permanent_password_digest,
		role,status,credential_mode,created_at,updated_at,permanent_password_generation)
		VALUES($1,$2,$3,$4,$5,'member','active','permanent',$6,$6,0)`, u.id, u.name, u.name+"@example.invalid", "synthetic-phone-"+u.id, s33Digest, f.clock.at)
	s33Must(f.t, err)
	for _, v := range [][2]string{{"username", u.name}, {"email", u.name + "@example.invalid"}, {"phone", "synthetic-phone-" + u.id}} {
		_, err = tx.Exec(f.ctx, `INSERT INTO identity_identifiers(user_id,kind,normalized_value,lifecycle,created_at) VALUES($1,$2,$3,'active',$4)`, u.id, v[0], v[1], f.clock.at)
		s33Must(f.t, err)
	}
	if withSession {
		u.old, err = f.issuer.Issue(identitypkg.User{ID: u.id}, session.CredentialPermanent, nil)
		s33Must(f.t, err)
		_, err = f.sessions.Create(f.ctx, tx, u.old)
		s33Must(f.t, err)
	}
	s33Must(f.t, tx.Commit(f.ctx))
	return u
}

type s33HTTP struct {
	status             int
	code, message      string
	fields             any
	token, csrf, retry string
	cookies            []*http.Cookie
}

func s33Decode(t *testing.T, r *httptest.ResponseRecorder) s33HTTP {
	t.Helper()
	result := s33HTTP{status: r.Code, cookies: r.Result().Cookies(), retry: r.Header().Get("Retry-After")}
	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Fields  any    `json:"field_errors"`
		CSRF    string `json:"csrf_token"`
	}
	if r.Body.Len() != 0 {
		s33Must(t, json.Unmarshal(r.Body.Bytes(), &body))
	}
	result.code, result.message, result.fields, result.csrf = body.Code, body.Message, body.Fields, body.CSRF
	for _, c := range result.cookies {
		if c.Name == platform.SessionCookieName {
			result.token = c.Value
		}
	}
	return result
}

func (f *s33Fixture) request(method, path, peer string, body io.Reader) *http.Request {
	r := httptest.NewRequest(method, s33Origin+path, body).WithContext(f.ctx)
	r.RemoteAddr = peer
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", s33Origin)
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	return r
}

func (f *s33Fixture) login(name, password, peer, old string) s33HTTP {
	data, err := json.Marshal(map[string]string{"username": name, "password": password})
	s33Must(f.t, err)
	r := f.request(http.MethodPost, "/api/v1/sessions", peer, bytes.NewReader(data))
	if old != "" {
		r.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: old})
	}
	w := httptest.NewRecorder()
	f.guard.Protect(http.HandlerFunc(f.handler.Login)).ServeHTTP(w, r)
	return s33Decode(f.t, w)
}

func (f *s33Fixture) probe(token, csrf string) (s33HTTP, int) {
	a, err := platform.NewHTTPAuthorizer(f.sessions, f.clock.Now)
	s33Must(f.t, err)
	hits := 0
	terminal := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := platform.PrincipalFromContext(r.Context())
		s33Require(f.t, ok && p.Status == "active", "missing real authorized principal")
		hits++
		w.WriteHeader(http.StatusNoContent)
	})
	r := f.request(http.MethodPost, "/s33/protected", "192.0.2.1:41000", nil)
	r.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: token})
	if csrf != "" {
		r.Header.Set(platform.CSRFHeaderName, csrf)
	}
	w := httptest.NewRecorder()
	f.guard.Protect(a.Protect(session.AccessGeneral, terminal)).ServeHTTP(w, r)
	return s33Decode(f.t, w), hits
}

func s33Envelope(t *testing.T, r s33HTTP, status int, code string) {
	t.Helper()
	s33Require(t, r.status == status && r.code == code, "HTTP status/public code mismatch")
	if status != http.StatusOK {
		s33Require(t, len(r.cookies) == 0, "unexpected replacement cookie")
	}
}

func s33Unified(t *testing.T, a, b s33HTTP) {
	t.Helper()
	s33Envelope(t, a, 401, "auth.login_failed")
	s33Envelope(t, b, 401, "auth.login_failed")
	s33Require(t, a.message == b.message && reflect.DeepEqual(a.fields, b.fields), "account failure envelope differs")
}

func (f *s33Fixture) fingerprint(peer string) string {
	e, err := f.source.Evidence(f.request(http.MethodPost, "/api/v1/sessions", peer, nil))
	s33Must(f.t, err)
	return e.Fingerprint
}

func (f *s33Fixture) attempt(subject, peer string) identitypkg.LoginAttempt {
	return identitypkg.LoginAttempt{Subject: subject, SourceFingerprint: f.fingerprint(peer), Now: f.clock.at}
}

func (f *s33Fixture) record(a identitypkg.LoginAttempt) {
	tx := f.begin()
	defer s33Rollback(f.t, tx)
	s33Must(f.t, f.protector.RecordFailure(f.ctx, tx, a))
	s33Must(f.t, tx.Commit(f.ctx))
}

func (f *s33Fixture) trigger(table, condition string) {
	q := pgx.Identifier{f.schema}.Sanitize()
	prior := ""
	if table == "login_attempt_buckets" {
		prior = `IF (SELECT count(*) FROM ` + q + `.login_attempt_buckets WHERE bucket_kind IN ('subject','source')
			AND failure_count=NEW.failure_count AND updated_at=NEW.updated_at)<>2 THEN
			RAISE EXCEPTION 'prior staged bucket writes missing' USING ERRCODE='P0001'; END IF;`
	}
	_, err := f.pool.Exec(f.ctx, `CREATE FUNCTION `+q+`.s33_fault() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN `+prior+` RAISE EXCEPTION 'synthetic fault' USING ERRCODE='23514',
		CONSTRAINT='s33_security_fault',SCHEMA=TG_TABLE_SCHEMA,TABLE=TG_TABLE_NAME;
		RETURN NEW; END $$`)
	s33Must(f.t, err)
	_, err = f.pool.Exec(f.ctx, `CREATE TRIGGER s33_fault BEFORE INSERT OR UPDATE ON `+
		pgx.Identifier{f.schema, table}.Sanitize()+` FOR EACH ROW WHEN (`+condition+`) EXECUTE FUNCTION `+q+`.s33_fault()`)
	s33Must(f.t, err)
}

func s33InjectedFault(err error, schema, table string) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23514" && pg.ConstraintName == "s33_security_fault" &&
		pg.SchemaName == schema && pg.TableName == table
}

func (f *s33Fixture) bucketCounts(stage string, want map[string][]int64) {
	f.snapshot(stage)
	rows, err := f.observer.Query(f.ctx, `SELECT bucket_kind,failure_count FROM `+
		pgx.Identifier{f.schema, "login_attempt_buckets"}.Sanitize()+` ORDER BY bucket_kind,failure_count`)
	s33Must(f.t, err)
	defer rows.Close()
	got := map[string][]int64{}
	for rows.Next() {
		var kind string
		var n int64
		s33Must(f.t, rows.Scan(&kind, &n))
		got[kind] = append(got[kind], n)
	}
	s33Must(f.t, rows.Err())
	s33Require(f.t, reflect.DeepEqual(got, want), "exact persisted bucket counts mismatch")
}

type s33BucketExpected struct {
	kind    identitypkg.LoginBucketKind
	digest  []byte
	version string
	count   int64
	at      time.Time
}

func (f *s33Fixture) expectedBuckets(attempts []identitypkg.LoginAttempt) map[string]*s33BucketExpected {
	want := map[string]*s33BucketExpected{}
	for _, a := range attempts {
		for _, kind := range []identitypkg.LoginBucketKind{identitypkg.LoginBucketSubject, identitypkg.LoginBucketSource, identitypkg.LoginBucketSubjectSource} {
			digest, version, err := f.digester.Digest(kind, a.Subject, a.SourceFingerprint)
			s33Must(f.t, err)
			key := string(kind) + ":" + string(digest)
			if want[key] == nil {
				want[key] = &s33BucketExpected{kind: kind, digest: digest, version: version, at: a.Now}
			}
			want[key].count++
		}
	}
	return want
}

func (f *s33Fixture) verifyBuckets(q s33Query, attempts []identitypkg.LoginAttempt) {
	want := f.expectedBuckets(attempts)
	var count int
	s33Must(f.t, q.QueryRow(f.ctx, "SELECT count(*) FROM login_attempt_buckets").Scan(&count))
	s33Require(f.t, count == len(want), "unexpected bucket identity cardinality")
	for _, w := range want {
		var n int64
		var version string
		var window, blocked, updated time.Time
		err := q.QueryRow(f.ctx, `SELECT failure_count,digest_key_version,window_started_at,blocked_until,updated_at
			FROM login_attempt_buckets WHERE bucket_kind=$1 AND bucket_key_digest=$2`, w.kind, w.digest).
			Scan(&n, &version, &window, &blocked, &updated)
		s33Must(f.t, err)
		until := w.at
		if w.count >= 2 {
			until = w.at.Add(5 * time.Minute)
		}
		s33Require(f.t, n == w.count && version == w.version && window.Equal(w.at) && updated.Equal(w.at) && blocked.Equal(until), "exact keyed bucket state differs")
	}
}

func (f *s33Fixture) assertAudit(userID, outcome, code string) {
	var count int
	s33Must(f.t, f.observer.QueryRow(f.ctx, "SELECT count(*) FROM audit_events").Scan(&count))
	s33Require(f.t, count == 1, "expected exactly one committed audit event")
	e, err := (audit.Store{}).Get(f.ctx, f.observer, f.auditor.eventID)
	s33Must(f.t, err)
	s33Require(f.t, e.ActorKind == audit.ActorAnonymous && e.Action == "identity.login" && e.TargetType == "user" && e.TargetID == userID && e.Outcome == outcome && e.TraceID == s33Trace, "committed audit identity/outcome differs")
	s33Require(f.t, e.OccurredAt.Equal(f.clock.at) && e.SourceFingerprint != nil, "committed audit time/source missing")
	if code == "" {
		s33Require(f.t, e.ErrorCode == nil, "unexpected audit error code")
	} else {
		s33Require(f.t, e.ErrorCode != nil && *e.ErrorCode == code, "committed audit public error differs")
	}
}

func (f *s33Fixture) waitBlocked(a, b *s33Tx) {
	ctx, cancel := context.WithTimeout(f.ctx, 3*time.Second)
	defer cancel()
	for {
		var blocked bool
		err := f.observer.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity p
			WHERE p.pid=$1 AND p.wait_event_type='Lock' AND p.wait_event='advisory'
			AND $2=ANY(pg_blocking_pids(p.pid)) AND EXISTS(SELECT 1 FROM pg_locks l
			WHERE l.pid=p.pid AND l.locktype='advisory' AND NOT l.granted))`, b.pid, a.pid).Scan(&blocked)
		s33Must(f.t, err)
		if blocked {
			f.t.Logf("schema=%s observer_pid=%d blocker_pid=%d blocked_pid=%d lock=advisory", f.schema, f.observer.Conn().PgConn().PID(), a.pid, b.pid)
			s33Must(f.t, s33WriteJSON(filepath.Join(f.dir, "blockers.json"), map[string]any{
				"observer_pid": f.observer.Conn().PgConn().PID(), "blocker_pid": a.pid, "blocked_pid": b.pid, "advisory_ungranted": true}))
			return
		}
	}
}

type s33ProbeInput struct {
	Mode    string
	Schema  string
	Now     time.Time
	Key     []byte
	Attempt identitypkg.LoginAttempt
	Kind    identitypkg.LoginAbuseDecisionKind
	Counts  map[string][]int64
}

func s33ChildEnvironment(t *testing.T) []string {
	t.Helper()
	env := []string{}
	for _, key := range []string{"SystemRoot", "WINDIR", "PATH", "TEMP", "TMP"} {
		if value, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+value)
		}
	}
	return append(env, testenv.EnvPostgresDSN+"="+testenv.RequirePostgresDSN(t))
}

func (f *s33Fixture) child(input s33ProbeInput) {
	data, err := json.Marshal(input)
	s33Must(f.t, err)
	binary, err := os.Executable()
	s33Must(f.t, err)
	ctx, cancel := context.WithTimeout(f.ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "-test.run=^TestS33LoginAbuseChildProbe$", "-test.v=true", "-test.count=1", "-test.timeout=10s")
	cmd.Env = s33ChildEnvironment(f.t)
	cmd.Stdin = bytes.NewReader(data)
	stdout, err := os.OpenFile(filepath.Join(f.dir, input.Mode+"-stdout.raw"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	s33Must(f.t, err)
	defer func() { s33Must(f.t, stdout.Close()) }()
	stderr, err := os.OpenFile(filepath.Join(f.dir, input.Mode+"-stderr.raw"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	s33Must(f.t, err)
	defer func() { s33Must(f.t, stderr.Close()) }()
	cmd.Stdout, cmd.Stderr = stdout, stderr
	started := time.Now().UTC()
	s33Must(f.t, s33WriteJSON(filepath.Join(f.dir, input.Mode+"-launch.json"), map[string]any{
		"argv": cmd.Args, "cwd": s33WorkingDirectory(f.t), "parent_pid": os.Getpid(), "started": started,
		"schema": f.schema, "mode": input.Mode, "environment": "minimal allowlist; DSN omitted from evidence"}))
	err = cmd.Start()
	s33Must(f.t, err)
	pid := cmd.Process.Pid
	err = cmd.Wait() // Always reap; CommandContext kills on deadline, Wait confirms completion.
	finished := time.Now().UTC()
	rawOut, readOutErr := os.ReadFile(filepath.Join(f.dir, input.Mode+"-stdout.raw"))
	rawErr, readErrErr := os.ReadFile(filepath.Join(f.dir, input.Mode+"-stderr.raw"))
	s33Must(f.t, s33WriteJSON(filepath.Join(f.dir, input.Mode+"-process.json"), map[string]any{
		"argv": cmd.Args, "cwd": s33WorkingDirectory(f.t), "parent_pid": os.Getpid(), "child_pid": pid,
		"started": started, "finished": finished, "wait_completed": true, "exit_code": cmd.ProcessState.ExitCode(),
		"timed_out": ctx.Err() != nil, "schema": f.schema, "mode": input.Mode,
		"stdout_sha256": fmt.Sprintf("%x", sha256.Sum256(rawOut)), "stderr_sha256": fmt.Sprintf("%x", sha256.Sum256(rawErr))}))
	s33Require(f.t, pid != os.Getpid(), "child PID was not independent")
	s33Must(f.t, err)
	s33Must(f.t, errors.Join(readOutErr, readErrErr))
	s33Require(f.t, cmd.ProcessState.ExitCode() == 0, "child failed")
	s33Require(f.t, len(rawErr) == 0, "unexpected child stderr")
	s33Require(f.t, strings.Contains(string(rawOut), "S33_PROBE_OK mode="+input.Mode+" pid="+fmt.Sprint(pid)), "child probe/PID receipt missing")
	s33Require(f.t, strings.Count(string(rawOut), "=== RUN") == 2 && strings.Contains(string(rawOut), "--- PASS: TestS33LoginAbuseChildProbe/"+input.Mode), "child top/leaf run-pass receipt differs")
}

func s33WorkingDirectory(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	s33Must(t, err)
	return wd
}

// s33FrozenIdentityMigrations selects the migration input used by the historical PG fixture.
func s33FrozenIdentityMigrations(input fs.FS) ([]migration.Migration, error) {
	selected := fstest.MapFS{}
	for _, name := range migrationFiles {
		path := "migrations/" + name
		sql, err := fs.ReadFile(input, path)
		if err != nil {
			return nil, fmt.Errorf("read historical migration %s: %w", name, err)
		}
		selected[path] = &fstest.MapFile{Data: sql}
	}
	return migration.Load(selected, "migrations")
}

// TestB13CFrozenIdentityMigrations exercises only fixture loading; it never opens PostgreSQL.
func TestB13CFrozenIdentityMigrations(t *testing.T) {

	for _, tc := range s33MigrationCases() {
		t.Run(tc.name, func(t *testing.T) {
			input := s33MigrationInputs()
			if tc.change != nil {
				tc.change(input)
			}
			got, err := s33FrozenIdentityMigrations(input)
			if tc.bad {
				if err == nil {
					t.Fatal("invalid historical chain accepted")
				}
				return
			}
			if err != nil {
				t.Fatalf("load historical chain: %v", err)
			}
			s33CheckHistoricalMigrations(t, got)
		})
	}
	t.Run("NoDirectoryEnumeration", func(t *testing.T) {
		got, err := s33FrozenIdentityMigrations(s33HistoricalFilesOnly{s33MigrationInputs()})
		if err != nil {
			t.Fatal(err)
		}
		s33CheckHistoricalMigrations(t, got)
	})
	t.Run("RepositoryHistoricalChain", func(t *testing.T) {
		got, err := s33FrozenIdentityMigrations(os.DirFS(filepath.Join("..", "..", "..")))
		if err != nil || len(got) != 11 {
			t.Fatalf("repository historical chain count=%d error=%v", len(got), err)
		}
	})
}

func s33MigrationInputs() fstest.MapFS {
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

func s33CheckHistoricalMigrations(t *testing.T, got []migration.Migration) {
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

type s33HistoricalFilesOnly struct{ fs.FS }

func (f s33HistoricalFilesOnly) Open(name string) (fs.File, error) {
	if name == "migrations" {
		return nil, fmt.Errorf("historical fixture must not enumerate the live directory")
	}
	return f.FS.Open(name)
}

type s33MigrationCase struct {
	name   string
	change func(fstest.MapFS)
	bad    bool
}

func s33MigrationCases() []s33MigrationCase {
	return []s33MigrationCase{
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
