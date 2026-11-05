//go:build integration

package identity

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"han/internal/article"
	"han/internal/audit"
	"han/internal/migration"
	"han/internal/platform"
	"han/internal/recovery"
	"han/internal/security"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// No legacy B13D PG harness or ambient/business DSN is used. Main provides this
// exact new-PG signal file only after SOURCE/LOCAL/PG windows have been granted.
type s411PGSignal struct {
	Purpose              string
	DSN                  string
	Host, Database, User string
	Port                 uint16
	ServerVersion        int
}

func s411PGConfig(t *testing.T) (*pgxpool.Config, int) {
	t.Helper()
	name := os.Getenv("HAN_B13E_S411_PG_CONFIG")
	if !filepath.IsAbs(name) || !strings.HasPrefix(strings.ToLower(filepath.ToSlash(name)), "d:/temp/") {
		t.Fatal("explicit own D Temp new-PG signal required")
	}
	raw, err := os.ReadFile(name)
	if err != nil {
		t.Fatal("read explicit new-PG signal")
	}
	var signal s411PGSignal
	if json.Unmarshal(raw, &signal) != nil || signal.Purpose != "B13E-S411" || signal.Host != "127.0.0.1" || signal.Port == 0 || signal.Database != "han_b13e_identity" || signal.User != "han_b13e" || signal.ServerVersion < 180000 {
		t.Fatal("new B13E signal binding required")
	}
	u, err := url.Parse(signal.DSN)
	if err != nil || u.Scheme != "postgres" || u.Hostname() != signal.Host || u.Port() == "" || u.User == nil || u.User.Username() != signal.User || u.Path != "/"+signal.Database || u.Query().Get("sslmode") != "disable" {
		t.Fatal("fully explicit synthetic local DSN required")
	}
	cfg, err := pgxpool.ParseConfig(signal.DSN)
	if err != nil || cfg.ConnConfig.Port != signal.Port {
		t.Fatal("new PG configuration mismatch")
	}
	cfg.MaxConns = 4
	return cfg, signal.ServerVersion
}

func s411PGHarness(t *testing.T) *s410PG {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	ctx, err := platform.ContextWithTraceID(ctx, "00000000000000000000000001")
	if err != nil {
		t.Fatal(err)
	}
	cfg, version := s411PGConfig(t)
	var random [10]byte
	if _, err := rand.Read(random[:]); err != nil {
		t.Fatal(err)
	}
	schema := "han_b13e_s411_" + hex.EncodeToString(random[:])
	cfg.ConnConfig.RuntimeParams["search_path"] = pgx.Identifier{schema}.Sanitize()
	cfg.ConnConfig.RuntimeParams["application_name"] = "b13e_s411_" + schema
	trace := &s410PGTrace{}
	cfg.ConnConfig.Tracer = trace
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal("open explicit B13E pool")
	}
	t.Cleanup(func() {
		pool.Close()
		t.Logf("actual_pool_close_join=true schema=%s acquired=%d", schema, pool.Stat().AcquiredConns())
	})
	s411Server(t, ctx, pool, cfg, version)
	h := &s410PG{t: t, ctx: ctx, pool: pool, schema: schema, clock: &s410Clock{now: s410Time.Add(time.Minute)}, passwords: &s410Passwords{}, audit: &s410Audit{marker: errors.New("synthetic audit failure")}, j: &s410Journal{}, trace: trace}
	h.exec(`CREATE SCHEMA ` + pgx.Identifier{schema}.Sanitize())
	t.Logf("retained_schema=%s", schema)
	t.Cleanup(func() { s411Retained(t, pool, schema) })
	s411Migrate(t, ctx, pool)
	h.install()
	h.users()
	h.store, err = NewRecoveryRunStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	commands, err := article.NewCommandService(article.CommandDependencies{Transactions: pool, Repository: article.Repository{}, IDs: s410IDs{}, Clock: h.clock, Orphaning: &article.OrphanArticlesCanonicalDependencies{Repository: article.CanonicalOrphanRepository{}, Auditor: h.audit, AuditIDs: s410IDs{}, Clock: h.clock}})
	if err != nil {
		t.Fatal(err)
	}
	h.articles = ArticleAccountPortFunc(func(c context.Context, tx pgx.Tx, id string) (CanonicalDeleteArticleResult, error) {
		r, e := commands.OrphanArticlesCanonical(c, tx, id)
		return CanonicalDeleteArticleResult{NewlyOrphanedCount: r.NewlyOrphanedCount, AlreadyOrphanedCount: r.AlreadyOrphanedCount, PendingOrphanCount: r.PendingOrphanCount}, e
	})
	return h
}

func s411Server(t *testing.T, ctx context.Context, pool *pgxpool.Pool, cfg *pgxpool.Config, expectedVersion int) {
	t.Helper()
	var addr, db, user string
	var port, version int
	err := pool.QueryRow(ctx, `SELECT host(inet_server_addr()),inet_server_port(),current_database(),current_user,current_setting('server_version_num')::int`).Scan(&addr, &port, &db, &user, &version)
	if err != nil || addr != cfg.ConnConfig.Host || port != int(cfg.ConnConfig.Port) || db != cfg.ConnConfig.Database || user != cfg.ConnConfig.User || version != expectedVersion {
		t.Fatal("new PG identity mismatch before DDL")
	}
}

func s411Migrate(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	names := []string{"0001_shared.sql", "0002_identity.sql", "0003_article.sql", "0004_idempotency.sql", "0005_recovery_runs.sql", "0006_user_governance_indexes.sql", "0007_idempotency_response_snapshot_v1.sql", "0008_password_reset_durable.sql", "0009_article_replay_attempt.sql", "0010_recovery_runtime_metadata.sql"}
	fs := fstest.MapFS{}
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join("..", "..", "migrations", name))
		if err != nil {
			t.Fatal(err)
		}
		fs["migrations/"+name] = &fstest.MapFile{Data: raw}
	}
	ms, err := migration.Load(fs, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range ms {
		func() {
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := finishAdmissionRead(ctx, tx); err != nil {
					t.Error(err)
				}
			}()
			if _, err = tx.Exec(ctx, m.SQL); err == nil {
				err = tx.Commit(ctx)
			}
			if err != nil {
				t.Fatalf("migration %d: %v", m.Version, err)
			}
		}()
	}
}

func s411Retained(t *testing.T, pool *pgxpool.Pool, schema string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname=$1)`, schema).Scan(&exists); err != nil || !exists {
		t.Error("exact owned schema not retained")
		return
	}
	for _, table := range []string{"users", "sessions", "recovery_runs", "audit_events"} {
		var raw []byte
		if err := pool.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(to_jsonb(x) ORDER BY to_jsonb(x)::text),'[]')::text FROM `+pgx.Identifier{schema, table}.Sanitize()+` x`).Scan(&raw); err != nil {
			t.Error(err)
			return
		}
		t.Logf("retained_rows schema=%s table=%s rows=%s", schema, table, raw)
	}
	if pool.Stat().AcquiredConns() != 0 {
		t.Error("owned connection not returned")
	}
}

type s411Passwords struct {
	hasher *security.GatedArgon2idHasher
	hashes atomic.Int32
}

func (p *s411Passwords) Hash(s string) (string, error)          { p.hashes.Add(1); return p.hasher.Hash(s) }
func (p *s411Passwords) Verify(s, d string) (bool, bool, error) { return p.hasher.Verify(s, d) }
func s411Hasher(t *testing.T) *s411Passwords {
	t.Helper()
	h, err := security.NewArgon2idHasher(security.Argon2idConfig{MemoryKiB: 8, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	g, err := security.NewArgon2idProcessGate(1)
	if err != nil {
		t.Fatal(err)
	}
	p, err := security.NewGatedArgon2idHasher(h, g)
	if err != nil {
		t.Fatal(err)
	}
	return &s411Passwords{hasher: p}
}

func s411Issuer(t *testing.T, h *s410PG) LoginSessionIssuer {
	t.Helper()
	i, err := NewRandomLoginSessionIssuer(rand.Reader, s410IDs{}, h.clock, 32, 32, time.Hour, time.Minute*30)
	if err != nil {
		t.Fatal(err)
	}
	return i
}

func s411Login(t *testing.T, h *s410PG, c context.Context, e *recoveryIdentityExecution, passwords *s411Passwords, secret string) (LoginResult, error) {
	t.Helper()
	fake, err := passwords.Hash("SyntheticFake123")
	if err != nil {
		t.Fatal(err)
	}
	gate := &PasswordResetStore{}
	ordinary, err := session.NewStore(session.StoreDependencies{Transactions: h.pool, PasswordResetGate: gate})
	if err != nil {
		t.Fatal(err)
	}
	digester, err := NewHMACLoginAbuseDigester("synthetic-s411-key", bytes.Repeat([]byte{0x4b}, 32))
	if err != nil {
		t.Fatal(err)
	}
	abuse, err := NewPersistentLoginAbuseProtector(PostgresLoginAbuseBucketStore{}, digester, LoginAbuseConfig{Window: time.Minute, FailureThreshold: 100, BlockDuration: time.Second, SuccessAction: LoginAbuseSuccessClear})
	if err != nil {
		t.Fatal(err)
	}
	var mode LoginRecoveryMode = e
	if e == nil {
		m, err := NewConfiguredLoginRecoveryMode("normal")
		if err != nil {
			t.Fatal(err)
		}
		mode = m
	}
	svc, err := NewLoginService(LoginDependencies{Transactions: h.pool, Repository: PostgresLoginRepository{}, PasswordResetGate: gate, Passwords: passwords, Sessions: ordinary, SessionIssuer: s411Issuer(t, h), Abuse: abuse, RecoveryMode: mode, Audit: h.audit, IDs: s410IDs{}, Clock: h.clock, FakeCredentialDigest: fake, FakeCredentialProbe: "SyntheticFake123"})
	if err != nil {
		t.Fatal(err)
	}
	auditor, err := NewPostgresLoginFailureAuditor(h.pool, audit.Store{})
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := NewLoginAttemptCoordinator(s410IDs{}, h.clock, auditor)
	if err != nil {
		t.Fatal(err)
	}
	attempt, err := coordinator.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err = attempt.enterService(); err != nil {
		t.Fatal(err)
	}
	return svc.Login(c, LoginRequest{Username: "initial", Password: secret, SourceFingerprint: "synthetic-s411-source", Attempt: attempt})
}

func s411Resolve(t *testing.T, h *s410PG, c context.Context, e *recoveryIdentityExecution, r LoginResult) (session.Principal, error) {
	t.Helper()
	s, err := e.maintenanceSessions(&PasswordResetStore{})
	if err != nil {
		t.Fatal(err)
	}
	return s.ResolvePrincipal(c, session.ResolveInput{Token: r.SessionToken, CSRFToken: r.CSRFToken, RequireCSRF: true, Now: h.clock.Now()})
}

func s411Change(t *testing.T, h *s410PG, c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution, passwords *s411Passwords, login LoginResult) (PasswordChangeResult, error) {
	t.Helper()
	principal, err := s411Resolve(t, h, c, e, login)
	if err != nil {
		return PasswordChangeResult{}, err
	}
	gate := &PasswordResetStore{}
	ordinary, err := session.NewStore(session.StoreDependencies{Transactions: h.pool, PasswordResetGate: gate})
	if err != nil {
		t.Fatal(err)
	}
	svc, err := NewPasswordChangeService(PasswordChangeDependencies{RecoveryAuthority: e, Transactions: h.pool, Repository: PostgresPasswordChangeRepository{}, PasswordResetGate: gate, Passwords: passwords, Policy: security.PasswordPolicy{}, Sessions: PostgresPasswordChangeSessions{Store: ordinary}, SessionIssuer: s411Issuer(t, h), Audit: e.d.audit, IDs: s410IDs{}, Clock: h.clock})
	if err != nil {
		t.Fatal(err)
	}
	return svc.ChangeRecoveryPassword(c, p, PasswordChangeRequest{Principal: principal, CurrentSessionToken: login.SessionToken, CurrentPassword: "synthetic-secret", NewPassword: "SyntheticReplacement123", TraceID: "00000000000000000000000001"})
}

func s411Image(t *testing.T, h *s410PG) string {
	t.Helper()
	var raw []byte
	// Independent explicit SQL oracle, not the production ACK collector.
	err := h.pool.QueryRow(h.ctx, `SELECT jsonb_build_object(
 'users',(SELECT jsonb_agg(to_jsonb(u) ORDER BY id) FROM users u),
 'runs',(SELECT jsonb_agg(to_jsonb(r) ORDER BY run_id) FROM recovery_runs r),
 'sessions',(SELECT jsonb_agg(to_jsonb(s) ORDER BY id) FROM sessions s),
 'audits',(SELECT jsonb_agg(to_jsonb(a) ORDER BY id) FROM audit_events a),
 'installation',(SELECT jsonb_agg(to_jsonb(i)) FROM recovery_installation i))::text`).Scan(&raw)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("independent_full_image=%s", raw)
	return string(raw)
}

func s411Trusted(t *testing.T, h *s410PG, c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) *s411Passwords {
	t.Helper()
	passwords := s411Hasher(t)
	e.d.passwords = passwords
	h.trusted(c, p, e)
	return passwords
}

func TestRecoveryMaintenanceAuthenticationPostgres(t *testing.T) {
	t.Run("ScopeAndPrivateBinding", s411PGScope)
	t.Run("BootstrapOnlyMePasswordAndDailyReject", func(t *testing.T) {
		s411PGResolver(t)
		s411AuditProofCases(t)
	})
	t.Run("PrepareAndChangeAtomicGPlusOne", func(t *testing.T) {
		t.Run("RealLoginStrongChangeAndIndependentRows", s411PGChange)
		for _, fault := range []string{"user", "run", "session", "password_audit", "recovery_audit"} {
			t.Run("Rollback/"+fault, func(t *testing.T) { s411PGChangeRollback(t, fault) })
		}
	})
	t.Run("PrepareChangeAckLossAndResume", func(t *testing.T) {
		t.Run("BrowserCookieRelogin", s411CookieRecoveryCases)
		for _, fault := range []string{"prepare_committed", "change_committed", "rolled_back", "partial_readback"} {
			t.Run(fault, func(t *testing.T) { s411PGACK(t, fault) })
		}
	})
	t.Run("RevokeMaintenanceBeforeOrdinaryLogin", func(t *testing.T) {
		for _, fault := range []string{"none", "audit", "committed_ack", "rollback_ack"} {
			t.Run(fault, func(t *testing.T) { s411PGExit(t, fault) })
		}
	})
}

func s411PGScope(t *testing.T) {
	h := s411PGHarness(t)
	h.running()
	s411Phase(h, func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		passwords := s411Trusted(t, h, c, p, e)
		r, err := s411Login(t, h, c, e, passwords, "synthetic-secret")
		if err != nil {
			t.Fatal(err)
		}
		principal, err := s411Resolve(t, h, c, e, r)
		if err != nil || !principal.MustChangePassword {
			t.Fatalf("maintenance control: %v", err)
		}
		s411OrdinaryReject(t, h, r)
		tx, err := h.pool.Begin(c)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := finishAdmissionRead(c, tx); err != nil {
				t.Error(err)
			}
		}()
		u, err := (PostgresPasswordChangeRepository{}).FindUserForPasswordChange(c, tx, s410Initial)
		if err != nil {
			t.Fatal(err)
		}
		in, err := s411Issuer(t, h).Issue(u, session.CredentialRecoveryBootstrap, nil)
		if err != nil {
			t.Fatal(err)
		}
		store, err := e.maintenanceSessions(&PasswordResetStore{})
		if err != nil {
			t.Fatal(err)
		}
		b, err := e.maintenanceBinding(c)
		if err != nil {
			t.Fatal(err)
		}
		wrong := recoverySessionBinding(b)
		wrong.RuntimeID += "-forged"
		if got, err := store.CreateMaintenance(c, tx, in, wrong); err == nil || got.Token != "" {
			t.Fatal("caller-created runtime binding authorized")
		}
		if got, err := store.CreateMaintenance(h.ctx, tx, in, recoverySessionBinding(b)); err == nil || got.Token != "" {
			t.Fatal("unbound context authorized")
		}
	})
	s411ScopeChecks(t, h)
}

func s411OrdinaryReject(t *testing.T, h *s410PG, r LoginResult) {
	t.Helper()
	s, err := session.NewStore(session.StoreDependencies{Transactions: h.pool, PasswordResetGate: &PasswordResetStore{}})
	if err != nil {
		t.Fatal(err)
	}
	before := s411Image(t, h)
	p, err := s.ResolvePrincipal(h.ctx, session.ResolveInput{Token: r.SessionToken, Now: h.clock.Now()})
	if err == nil || p.UserID != "" {
		t.Fatal("maintenance cookie became ordinary authority")
	}
	if s411Image(t, h) != before {
		t.Fatal("ordinary denial revoked or changed maintenance state")
	}
}

func s411PGResolver(t *testing.T) {
	for _, name := range []string{"valid", "csrf", "missing_csrf", "expired", "idle_expired", "revoked", "generation", "runtime", "kind", "noninitial", "hold"} {
		t.Run(name, func(t *testing.T) {
			h := s411PGHarness(t)
			h.running()
			s411Phase(h, func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
				passwords := s411Trusted(t, h, c, p, e)
				r, err := s411Login(t, h, c, e, passwords, "synthetic-secret")
				if err != nil {
					t.Fatal(err)
				}
				h.clock.advance()
				switch name {
				case "missing_csrf":
					r.CSRFToken = ""
				case "idle_expired":
					h.exec(`UPDATE sessions SET idle_expires_at=$1`, h.clock.Now())
				case "csrf":
					r.CSRFToken = "different"
				case "expired":
					h.exec(`UPDATE sessions SET absolute_expires_at=$1`, h.clock.Now())
				case "revoked":
					h.exec(`UPDATE sessions SET revoked_at=$1,revoke_reason='synthetic'`, h.clock.Now())
				case "generation":
					h.exec(`UPDATE sessions SET recovery_generation=recovery_generation+1`)
				case "runtime":
					h.exec(`UPDATE sessions SET recovery_runtime_id='other-runtime'`)
				case "kind":
					h.exec(`UPDATE sessions SET authenticated_credential_kind='permanent'`)
				case "noninitial":
					h.exec(`UPDATE sessions SET user_id=$1`, s410ND)
				case "hold":
					h.exec(`UPDATE users SET auth_integrity_hold=true,auth_integrity_hold_reason='synthetic',auth_integrity_hold_at=$2 WHERE id=$1`, s410Initial, h.clock.Now())
				}
				got, err := s411Resolve(t, h, c, e, r)
				if name == "valid" {
					if err != nil || !got.IsInitialAdmin || !got.MustChangePassword || !got.Allows(session.AccessCurrentIdentity) || !got.Allows(session.AccessChangePassword) || got.Allows(session.AccessGeneral) || got.Allows(session.AccessAcknowledgePrivacy) {
						t.Fatalf("bootstrap permissions: %v", err)
					}
				} else if err == nil || got.UserID != "" {
					t.Fatal("invalid maintenance evidence authenticated")
				}
			})
		})
	}
}

func s411PGChange(t *testing.T) {
	h := s411PGHarness(t)
	h.running()
	s411Phase(h, func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		passwords := s411Trusted(t, h, c, p, e)
		old, err := s411Login(t, h, c, e, passwords, "synthetic-secret")
		if err != nil {
			t.Fatal(err)
		}
		other, err := s411Login(t, h, c, e, passwords, "synthetic-secret")
		if err != nil {
			t.Fatal(err)
		}
		changed, err := s411Change(t, h, c, p, e, passwords, old)
		if err != nil {
			t.Fatal(err)
		}
		s411ChangedRows(t, h, old, other, changed)
		fresh := LoginResult{SessionToken: changed.SessionToken, CSRFToken: changed.CSRFToken}
		principal, err := s411Resolve(t, h, c, e, fresh)
		if err != nil || principal.MustChangePassword || !principal.IsInitialAdmin {
			t.Fatalf("permanent maintenance: %v", err)
		}
		s411OrdinaryReject(t, h, fresh)
		if _, err = s411Resolve(t, h, c, e, old); err == nil {
			t.Fatal("old cookie revived")
		}
		if _, err = s411Resolve(t, h, c, e, other); err == nil {
			t.Fatal("other session not revoked")
		}
	})
}

func s411ChangedRows(t *testing.T, h *s410PG, old, other LoginResult, changed PasswordChangeResult) {
	t.Helper()
	var mode, scope, oldReason, otherReason, actor, action, runUser string
	var required bool
	var auth, permanent, bootstrap, proof int64
	var sourceOperation, sourceCredential *string
	err := h.pool.QueryRow(h.ctx, `SELECT u.credential_mode,u.recovery_required,u.auth_generation,u.permanent_password_generation,
 u.permanent_password_source_operation_id::text,u.permanent_password_source_credential_id::text,
 r.initial_admin_user_id::text,r.initial_admin_bootstrap_generation,r.initial_admin_changed_generation,a.actor_user_id::text,a.action,
 n.access_scope,o.revoke_reason,z.revoke_reason
 FROM users u JOIN recovery_runs r ON r.initial_admin_user_id=u.id JOIN audit_events a ON a.id=r.initial_admin_change_audit_id
 JOIN sessions n ON n.token_digest=$1 JOIN sessions o ON o.token_digest=$2 JOIN sessions z ON z.token_digest=$3
 WHERE r.run_id=$4 AND n.user_id=u.id AND n.auth_generation=u.auth_generation AND n.authenticated_credential_kind='permanent'
 AND n.recovery_run_id=r.run_id AND n.recovery_installation_id=o.recovery_installation_id
 AND n.recovery_runtime_id=o.recovery_runtime_id AND n.recovery_generation=o.recovery_generation
 AND n.rotated_from_id=o.id AND n.revoked_at IS NULL AND a.occurred_at=r.initial_admin_changed_at AND a.outcome='succeeded'`, recoveryAuthDigest(changed.SessionToken), recoveryAuthDigest(old.SessionToken), recoveryAuthDigest(other.SessionToken), s410Binding.RunID).Scan(&mode, &required, &auth, &permanent, &sourceOperation, &sourceCredential, &runUser, &bootstrap, &proof, &actor, &action, &scope, &oldReason, &otherReason)
	if err != nil || mode != "permanent" || required || auth != bootstrap+1 || permanent != auth || proof != auth || actor != s410Initial || runUser != s410Initial || sourceOperation != nil || sourceCredential != nil || scope != "recovery_maintenance" || oldReason != "password_changed" || otherReason != "password_changed" || action != passwordChangeAuditAction {
		t.Fatalf("independent User/run/two Session/Audit tuple: %v", err)
	}
	s411Image(t, h)
}

func s411PGChangeRollback(t *testing.T, fault string) {
	h := s411PGHarness(t)
	h.running()
	s411Phase(h, func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		passwords := s411Trusted(t, h, c, p, e)
		login, err := s411Login(t, h, c, e, passwords, "synthetic-secret")
		if err != nil {
			t.Fatal(err)
		}
		switch fault {
		case "password_audit":
			h.audit.set(passwordChangeAuditAction)
		case "recovery_audit":
			h.audit.set(recoveryTrustChangeAction)
		default:
			table, condition, operation := "users", "NOT NEW.recovery_required", "UPDATE"
			if fault == "run" {
				table = "recovery_runs"
				condition = "NEW.initial_admin_changed_generation IS NOT NULL"
			}
			if fault == "session" {
				table = "sessions"
				condition = "true"
				operation = "INSERT"
			}
			h.exec(`CREATE FUNCTION s411_fail() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF ` + condition + ` THEN RAISE EXCEPTION 'synthetic mutation failure'; END IF; RETURN NEW; END $$`)
			h.exec(`CREATE TRIGGER s411_fail BEFORE ` + operation + ` ON ` + table + ` FOR EACH ROW EXECUTE FUNCTION s411_fail()`)
		}
		before := s411Image(t, h)
		result, err := s411Change(t, h, c, p, e, passwords, login)
		if err == nil || result.SessionToken != "" {
			t.Fatal("partial strong change returned success")
		}
		if s411Image(t, h) != before {
			t.Fatal("User/run/Session/Audit was not atomic")
		}
	})
}

func s411PGACK(t *testing.T, fault string) {
	h := s411PGHarness(t)
	h.running()
	var saved LoginResult
	var hasher *s411Passwords
	s411Phase(h, func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		hasher = s411Hasher(t)
		e.d.passwords = hasher
		h.finalized(c, p, e)
		if fault == "prepare_committed" {
			s411Fault(h, e, fault)
		}
		if err := e.PrepareRecoveryTrust(c, p, s410Binding, []byte("synthetic-secret")); err != nil {
			t.Fatal(err)
		}
		before, hashes := s411Image(t, h), hasher.hashes.Load()
		if err := e.PrepareRecoveryTrust(c, p, s410Binding, nil); err != nil || hasher.hashes.Load() != hashes || s411Image(t, h) != before {
			t.Fatal("prepare retry rehashed or wrote")
		}
		login, err := s411Login(t, h, c, e, hasher, "synthetic-secret")
		if err != nil {
			t.Fatal(err)
		}
		saved = login
		if fault != "prepare_committed" {
			s411Fault(h, e, fault)
		}
		h.trace.start()
		_, err = s411Change(t, h, c, p, e, hasher, login)
		pids := h.trace.stop()
		if fault == "rolled_back" || fault == "partial_readback" {
			if err == nil || platform.PublicCode(err) != platform.CodePlatformRecoveryInProgress {
				t.Fatal("unknown proof reported success")
			}
			return
		}
		if err != nil || len(pids) < 2 {
			t.Fatalf("commit ACK not independently confirmed: %v pids=%v", err, pids)
		}
	})
	if fault == "rolled_back" {
		return
	}
	s411ResumePassword(t, h, hasher, saved)
}

func s411PGExit(t *testing.T, fault string) {
	h := s411PGHarness(t)
	h.running()
	var hasher *s411Passwords
	var maintained LoginResult
	s411Phase(h, func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		hasher = s411Trusted(t, h, c, p, e)
		old, err := s411Login(t, h, c, e, hasher, "synthetic-secret")
		if err != nil {
			t.Fatal(err)
		}
		changed, err := s411Change(t, h, c, p, e, hasher, old)
		if err != nil {
			t.Fatal(err)
		}
		maintained = LoginResult{SessionToken: changed.SessionToken, CSRFToken: changed.CSRFToken}
		h.denied(c, p, e)
		if err = e.CompleteIdentityReview(c, p, s410Binding); err != nil {
			t.Fatal(err)
		}
	})
	s411Succeeded(h)
	h.session("maintenance", "old-runtime", true)
	s411ExitPhase(h, func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		s411ExitChecked(t, h, c, p, e, fault)
	})
	if fault == "audit" || fault == "rollback_ack" {
		return
	}
	s411OrdinaryRelogin(t, h, hasher, maintained)
}

func s411Succeeded(h *s410PG) {
	r := s410Ready()
	h.exec(`UPDATE recovery_runs SET outcome='article_deletion_ready',article_publication_ref='s410-publication',article_publication_digest=$2,article_evidence_version=$3,article_evidence_digest=$4,article_evidence_ref=$5,article_evidence_completed_at=$6,article_namespace=$7,article_replay_verified_at=$6 WHERE run_id=$1`, s410Binding.RunID, s410Hash, r.EvidenceVersion, r.EvidenceDigest, r.EvidenceRef, s410Time.Add(90*time.Second), r.Namespace)
	h.exec(`UPDATE recovery_runs SET outcome='ready_to_consume'`)
	h.j.append(recovery.StageConsumed)
	h.exec(`UPDATE recovery_runs SET outcome='journal_consumed',journal_stage='consumed',journal_consumed_at=$1`, s410Time.Add(120*time.Second))
	h.exec(`UPDATE recovery_runs SET outcome='succeeded',completed_at=$1`, s410Time.Add(121*time.Second))
	h.clock.mu.Lock()
	h.clock.now = s410Time.Add(123 * time.Second)
	h.clock.mu.Unlock()
}

func s411ScopeChecks(t *testing.T, h *s410PG) {
	for _, sql := range []string{
		`UPDATE sessions SET access_scope='ordinary'`,
		`UPDATE sessions SET recovery_runtime_id=NULL`,
		`UPDATE sessions SET recovery_generation=0`,
		`UPDATE sessions SET recovery_installation_id=NULL`,
		`UPDATE sessions SET recovery_run_id=NULL`,
	} {
		before := s411Image(t, h)
		if _, err := h.pool.Exec(h.ctx, sql); err == nil {
			t.Fatal("invalid scope accepted")
		} else {
			var pg *pgconn.PgError
			if !errors.As(err, &pg) || pg.Code != "23514" {
				t.Fatalf("expected CHECK failure: %v", err)
			}
		}
		if s411Image(t, h) != before {
			t.Fatal("failed scope mutation changed durable rows")
		}
	}
}

func s411ResumePassword(t *testing.T, h *s410PG, hasher *s411Passwords, saved LoginResult) {
	s411Phase(h, func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		e.d.passwords = hasher
		before, hashes := s411Image(t, h), hasher.hashes.Load()
		if err := e.PrepareRecoveryTrust(c, p, s410Binding, nil); err != nil || hasher.hashes.Load() != hashes || s411Image(t, h) != before {
			t.Fatal("Resume reset committed trust")
		}
		if _, err := s411Resolve(t, h, c, e, saved); err == nil {
			t.Fatal("previous runtime cookie accepted")
		}
		if got, err := s411Login(t, h, c, e, hasher, "synthetic-secret"); err == nil || got.SessionToken != "" {
			t.Fatal("old password revived")
		}
		fresh, err := s411Login(t, h, c, e, hasher, "SyntheticReplacement123")
		if err != nil {
			t.Fatal(err)
		}
		if principal, err := s411Resolve(t, h, c, e, fresh); err != nil || principal.MustChangePassword {
			t.Fatalf("new-password maintenance relogin: %v", err)
		}
	})
}

func s411OrdinaryRelogin(t *testing.T, h *s410PG, hasher *s411Passwords, maintained LoginResult) {
	s411OrdinaryReject(t, h, maintained)
	fresh, err := s411Login(t, h, h.ctx, nil, hasher, "SyntheticReplacement123")
	if err != nil {
		t.Fatal(err)
	}
	store, err := session.NewStore(session.StoreDependencies{Transactions: h.pool, PasswordResetGate: &PasswordResetStore{}})
	if err != nil {
		t.Fatal(err)
	}
	principal, err := store.ResolvePrincipal(h.ctx, session.ResolveInput{Token: fresh.SessionToken, Now: h.clock.Now()})
	if err != nil || principal.MustChangePassword || !principal.IsInitialAdmin {
		t.Fatalf("ordinary new-password relogin: %v", err)
	}
	var ordinary bool
	if err = h.pool.QueryRow(h.ctx, `SELECT access_scope='ordinary' AND recovery_run_id IS NULL AND recovery_installation_id IS NULL AND recovery_runtime_id IS NULL AND recovery_generation IS NULL FROM sessions WHERE token_digest=$1`, recoveryAuthDigest(fresh.SessionToken)).Scan(&ordinary); err != nil || !ordinary {
		t.Fatal("ordinary login retained recovery binding")
	}
}

func s411ExitChecked(t *testing.T, h *s410PG, c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution, fault string) {

	if fault == "audit" {
		h.audit.set(recoveryExitAction)
	}
	if fault == "committed_ack" || fault == "rollback_ack" {
		s411Fault(h, e, fault)
	}
	before := s411Image(t, h)
	err := e.RevokeRecoverySessions(c, p, s410Binding)
	if fault == "audit" || fault == "rollback_ack" {
		if err == nil || s411Image(t, h) != before {
			t.Fatal("failed exit changed aggregate")
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	var complete bool
	err = h.pool.QueryRow(h.ctx, `SELECT NOT EXISTS(SELECT 1 FROM sessions WHERE recovery_run_id=$1 AND access_scope='recovery_maintenance' AND (revoked_at IS NULL OR (revoke_reason<>'password_changed' AND revoke_reason<>'recovery_maintenance_exit')))
 AND EXISTS(SELECT 1 FROM audit_events a JOIN recovery_runs r ON a.target_id=r.run_id WHERE a.action='recovery_maintenance_exit' AND a.metadata->>'run_id'=r.run_id AND a.metadata->>'installation_id'='s410-install' AND r.initial_admin_changed_generation IS NOT NULL AND a.metadata ? 'revoked_ids')`, s410Binding.RunID).Scan(&complete)
	if err != nil || !complete {
		t.Fatalf("full exit rows/proof/Audit: %v", err)
	}
	before = s411Image(t, h)
	if err = e.RevokeRecoverySessions(c, p, s410Binding); err != nil || s411Image(t, h) != before {
		t.Fatal("exit exact retry not read-only")
	}

}

func s411Fault(h *s410PG, e *recoveryIdentityExecution, fault string) {
	e.commit = func(ctx context.Context, tx pgx.Tx) error {
		if fault == "rolled_back" || fault == "rollback_ack" {
			if err := tx.Rollback(ctx); err != nil {
				return err
			}
		} else if err := tx.Commit(ctx); err != nil {
			return err
		}
		if fault == "partial_readback" {
			h.exec(`UPDATE sessions SET revoked_at=$1,revoke_reason='synthetic-drift' WHERE revoked_at IS NULL`, h.clock.Now())
		}
		return errors.New("synthetic actual Commit/Rollback ACK loss")
	}
}

func s411Phase(h *s410PG, body func(context.Context, recovery.PhasePermit, *recoveryIdentityExecution)) {
	var rt recovery.Runtime
	defer s411RuntimeJoined(h.t, &rt)
	h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		rt = e.d.runtime
		body(c, p, e)
	})
}
func s411ExitPhase(h *s410PG, body func(context.Context, recovery.PhasePermit, *recoveryIdentityExecution)) {
	var rt recovery.Runtime
	defer s411RuntimeJoined(h.t, &rt)
	h.exit(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		rt = e.d.runtime
		body(c, p, e)
	})
}
func s411RuntimeJoined(t *testing.T, rt *recovery.Runtime) {
	if *rt == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	err := (*rt).Stop(ctx)
	t.Logf("actual_runtime_stop_join=%t callback_unwound=true error=%v", err == nil, err)
	if err != nil {
		t.Error(err)
	}
}

// Fault injection only: an appender returning nil is not persisted audit proof.
type s411AuditFault struct{ fault string }

func (f s411AuditFault) Append(ctx context.Context, tx pgx.Tx, event audit.Event) error {
	if event.Action == recoveryTrustAction {
		if f.fault == "missing_prepare" {
			return nil
		}
		if f.fault == "wrong_prepare" {
			event.Metadata["bootstrap_generation"] = int64(9999)
		}
	}
	if event.Action == recoveryTrustChangeAction {
		if f.fault == "missing_changed" {
			return nil
		}
		if f.fault == "wrong_changed" {
			event.Metadata["changed_generation"] = int64(9999)
		}
	}
	if event.Action == passwordChangeAuditAction && f.fault == "missing_password" {
		return nil
	}
	return (audit.Store{}).Append(ctx, tx, event)
}

func s411AuditProofCases(t *testing.T) {
	for _, fault := range []string{"wrong_prepare", "missing_prepare", "wrong_changed", "missing_changed", "missing_password"} {
		t.Run("AuditProof/"+fault, func(t *testing.T) { s411AuditProof(t, fault) })
	}
}

func s411AuditProof(t *testing.T, fault string) {
	h := s411PGHarness(t)
	h.running()
	s411Phase(h, func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		prepared := strings.HasSuffix(fault, "prepare")
		if prepared {
			e.d.audit = s411AuditFault{fault: fault}
		}
		passwords := s411Trusted(t, h, c, p, e)
		login, err := s411Login(t, h, c, e, passwords, "synthetic-secret")
		if prepared {
			if err == nil || login.SessionToken != "" {
				t.Fatal("incorrect bootstrap audit authenticated maintenance session")
			}
			s411Image(t, h)
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		e.d.audit = s411AuditFault{fault: fault}
		before := s411Image(t, h)
		result, err := s411Change(t, h, c, p, e, passwords, login)
		if err == nil || result.SessionToken != "" {
			t.Fatal("incomplete or incorrect change audit issued maintenance cookie")
		}
		if s411Image(t, h) != before {
			t.Fatal("audit proof failure partially committed trust")
		}
	})
}

// S411-ICR-R01-F001: exercise the existing HTTP cookie consumer, rather than
// dropping ExistingSessionToken in a service-only re-login helper.
func s411CookieRecoveryCases(t *testing.T) {
	for _, name := range []string{"no_cookie_positive", "old_runtime_cookie", "lost_change_response_old_cookie",
		"valid_current_cookie", "revoked_cookie", "absolute_expired_cookie", "idle_expired_cookie",
		"unknown_cookie", "old_password_rejected", "gate_blocks", "login_audit_rollback"} {
		t.Run(name, func(t *testing.T) { s411CookieRecoveryCase(t, name) })
	}
}

func s411CookieRecoveryCase(t *testing.T, name string) {
	h := s411PGHarness(t)
	h.running()
	var passwords *s411Passwords
	var saved LoginResult
	s411Phase(h, func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		passwords = s411Trusted(t, h, c, p, e)
		old, err := s411Login(t, h, c, e, passwords, "synthetic-secret")
		if err != nil {
			t.Fatal(err)
		}
		changed, err := s411Change(t, h, c, p, e, passwords, old)
		if err != nil {
			t.Fatal(err)
		}
		saved = LoginResult{SessionToken: changed.SessionToken, CSRFToken: changed.CSRFToken}
		if name == "lost_change_response_old_cookie" {
			s411CookieHTTPCheck(t, h, c, e, passwords, old, name)
		} else if name != "old_runtime_cookie" && name != "no_cookie_positive" {
			s411CookieMutate(h, saved, name)
			s411CookieHTTPCheck(t, h, c, e, passwords, saved, name)
		}
	})
	if name != "old_runtime_cookie" && name != "no_cookie_positive" {
		return
	}
	s411Phase(h, func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		e.d.passwords = passwords
		before, hashes := s411Image(t, h), passwords.hashes.Load()
		if err := e.PrepareRecoveryTrust(c, p, s410Binding, nil); err != nil || s411Image(t, h) != before || hashes != passwords.hashes.Load() {
			t.Fatal("Resume changed committed trust or rehashed password")
		}
		if _, err := s411Resolve(t, h, c, e, saved); err == nil {
			t.Fatal("old runtime cookie authorized protected maintenance")
		}
		if name == "no_cookie_positive" {
			saved = LoginResult{}
		}
		s411CookieHTTPCheck(t, h, c, e, passwords, saved, name)
	})
}

func s411CookieMutate(h *s410PG, old LoginResult, name string) {
	switch name {
	case "revoked_cookie":
		h.exec(`UPDATE sessions SET revoked_at=$2,revoke_reason='synthetic-cookie-revoked' WHERE token_digest=$1`, recoveryAuthDigest(old.SessionToken), h.clock.Now())
	case "absolute_expired_cookie", "idle_expired_cookie":
		h.clock.advance()
		column := "absolute_expires_at"
		if name == "idle_expired_cookie" {
			column = "idle_expires_at"
		}
		h.exec(`UPDATE sessions SET `+column+`=$2 WHERE token_digest=$1`, recoveryAuthDigest(old.SessionToken), h.clock.Now())
	case "gate_blocks":
		h.exec(`UPDATE users SET auth_integrity_hold=true,auth_integrity_hold_at=$2,auth_integrity_hold_reason='synthetic-cookie-gate' WHERE id=$1`, s410Initial, h.clock.Now())
	case "login_audit_rollback":
		h.audit.set(loginAuditAction)
	}
}

func s411CookieHTTPService(t *testing.T, h *s410PG, e *recoveryIdentityExecution, passwords *s411Passwords) (*Service, *LoginAttemptCoordinator) {
	t.Helper()
	fake, err := passwords.Hash("SyntheticFake123")
	if err != nil {
		t.Fatal(err)
	}
	gate := &PasswordResetStore{}
	ordinary, err := session.NewStore(session.StoreDependencies{Transactions: h.pool, PasswordResetGate: gate})
	if err != nil {
		t.Fatal(err)
	}
	digester, err := NewHMACLoginAbuseDigester("synthetic-s411-cookie-key", bytes.Repeat([]byte{0x4b}, 32))
	if err != nil {
		t.Fatal(err)
	}
	abuse, err := NewPersistentLoginAbuseProtector(PostgresLoginAbuseBucketStore{}, digester, LoginAbuseConfig{Window: time.Minute, FailureThreshold: 100, BlockDuration: time.Second, SuccessAction: LoginAbuseSuccessClear})
	if err != nil {
		t.Fatal(err)
	}
	svc, err := NewLoginService(LoginDependencies{Transactions: h.pool, Repository: PostgresLoginRepository{}, PasswordResetGate: gate,
		Passwords: passwords, Sessions: ordinary, SessionIssuer: s411Issuer(t, h), Abuse: abuse, RecoveryMode: e,
		Audit: h.audit, IDs: s410IDs{}, Clock: h.clock, FakeCredentialDigest: fake, FakeCredentialProbe: "SyntheticFake123"})
	if err != nil {
		t.Fatal(err)
	}
	auditor, err := NewPostgresLoginFailureAuditor(h.pool, audit.Store{})
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := NewLoginAttemptCoordinator(s410IDs{}, h.clock, auditor)
	if err != nil {
		t.Fatal(err)
	}
	return svc, coordinator
}

func s411CookieHTTPInvoke(t *testing.T, c context.Context, svc *Service, coordinator *LoginAttemptCoordinator, cookie, name string) (*httptest.ResponseRecorder, LoginResult) {
	t.Helper()
	config := newLoginHandlerTestConfig(t)
	config.Coordinator = coordinator
	handler, err := NewLoginHandler(svc, registrationSourceStub{evidence: RegistrationSourceEvidence{Kind: "trusted-source", Fingerprint: "synthetic-cookie-source"}}, 4096, config)
	if err != nil {
		t.Fatal(err)
	}
	password := "SyntheticReplacement123"
	if name == "old_password_rejected" {
		password = "synthetic-secret"
	}
	raw, err := json.Marshal(map[string]string{"username": "initial", "password": password})
	if err != nil {
		t.Fatal(err)
	}
	request := newLoginHTTPRequest(string(raw)).WithContext(c)
	if cookie != "" {
		request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: cookie})
	}
	defer func() {
		if err := request.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	response := httptest.NewRecorder()
	handler.Login(response, request)
	result := response.Result()
	defer func() {
		if err := result.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	if response.Code != http.StatusOK {
		if len(result.Cookies()) != 0 {
			t.Fatal("failed login issued a cookie")
		}
		return response, LoginResult{}
	}
	cookies := result.Cookies()
	var body loginResponse
	if len(cookies) != 1 || cookies[0].Value == "" || cookies[0].Value == cookie || json.Unmarshal(response.Body.Bytes(), &body) != nil || body.CSRFToken == "" || body.User.ID != s410Initial {
		t.Fatal("real Handler failed to issue distinct current Session/CSRF")
	}
	return response, LoginResult{SessionToken: cookies[0].Value, CSRFToken: body.CSRFToken}
}

func s411CookieHTTPCheck(t *testing.T, h *s410PG, c context.Context, e *recoveryIdentityExecution, passwords *s411Passwords, old LoginResult, name string) {
	t.Helper()
	svc, coordinator := s411CookieHTTPService(t, h, e, passwords)
	cookie := old.SessionToken
	if name == "unknown_cookie" {
		cookie = "synthetic-unknown-cookie"
		old = LoginResult{SessionToken: cookie}
	}
	beforeUser, beforeRun := string(h.rawUser(s410Initial)), s411CookieRow(t, h, "run", "")
	beforeOld, beforeSessions := s411CookieRow(t, h, "session", cookie), s411CookieRow(t, h, "sessions", "")
	hashes := passwords.hashes.Load()
	response, fresh := s411CookieHTTPInvoke(t, c, svc, coordinator, cookie, name)
	if string(h.rawUser(s410Initial)) != beforeUser || s411CookieRow(t, h, "run", "") != beforeRun || passwords.hashes.Load() != hashes {
		t.Fatal("cookie re-login reset User/run proof or rehashed committed password")
	}
	if name == "old_password_rejected" || name == "gate_blocks" || name == "login_audit_rollback" {
		if response.Code == http.StatusOK || fresh.SessionToken != "" || s411CookieRow(t, h, "sessions", "") != beforeSessions {
			t.Fatal("failed authenticated issuance committed Session changes")
		}
		return
	}
	if response.Code != http.StatusOK {
		t.Fatalf("valid new password blocked by cookie: variant=%s status=%d body=%s", name, response.Code, response.Body.String())
	}
	s411CookieIssuedRows(t, h, c, e, old, fresh, name == "valid_current_cookie")
	if name != "valid_current_cookie" && s411CookieRow(t, h, "session", cookie) != beforeOld {
		t.Fatal("ineligible optional cookie row was changed or linked")
	}
}

func s411CookieRow(t *testing.T, h *s410PG, kind, token string) string {
	t.Helper()
	query := `SELECT COALESCE((SELECT to_jsonb(r)::text FROM recovery_runs r WHERE run_id=$1),'null')`
	var arg any = s410Binding.RunID
	if kind == "session" {
		query = `SELECT COALESCE((SELECT to_jsonb(s)::text FROM sessions s WHERE token_digest=$1),'null')`
		arg = recoveryAuthDigest(token)
	} else if kind == "sessions" {
		query = `SELECT COALESCE(jsonb_agg(to_jsonb(s) ORDER BY id),'[]')::text FROM sessions s WHERE user_id=$1`
		arg = s410Initial
	}
	var raw string
	if err := h.pool.QueryRow(h.ctx, query, arg).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	t.Logf("cookie_physical_row kind=%s row=%s", kind, raw)
	return raw
}

func s411CookieIssuedRows(t *testing.T, h *s410PG, c context.Context, e *recoveryIdentityExecution, old, fresh LoginResult, rotated bool) {
	t.Helper()
	principal, err := s411Resolve(t, h, c, e, fresh)
	if err != nil || principal.UserID != s410Initial || principal.MustChangePassword {
		t.Fatalf("new maintenance cookie did not resolve current permanent identity: %v", err)
	}
	p, err := e.maintenanceBinding(c)
	if err != nil {
		t.Fatal(err)
	}
	var exact bool
	err = h.pool.QueryRow(h.ctx, `SELECT id::text=$2 AND user_id=$3 AND access_scope='recovery_maintenance'
 AND recovery_run_id=$4 AND recovery_installation_id=$5 AND recovery_runtime_id=$6 AND recovery_generation=$7
 AND auth_generation=3 AND authenticated_credential_kind='permanent' AND authenticated_credential_id IS NULL
 AND lineage_operation_id IS NULL AND lineage_credential_id IS NULL AND revoked_at IS NULL
 AND absolute_expires_at>$8 AND idle_expires_at>$8 AND csrf_token_digest=$9
 AND CASE WHEN $10 THEN rotated_from_id=(SELECT id FROM sessions WHERE token_digest=$11)
 ELSE rotated_from_id IS NULL END FROM sessions WHERE token_digest=$1`, recoveryAuthDigest(fresh.SessionToken), principal.SessionID,
		s410Initial, p.RunID, p.InstallationID, p.RuntimeID, p.Generation, h.clock.Now(), recoveryAuthDigest(fresh.CSRFToken), rotated, recoveryAuthDigest(old.SessionToken)).Scan(&exact)
	if err != nil || !exact {
		t.Fatalf("new physical Session binding/scope/lineage differs: %v", err)
	}
	if old.SessionToken != "" {
		if _, err := s411Resolve(t, h, c, e, old); err == nil {
			t.Fatal("old cookie authorized protected maintenance after re-login")
		}
		s411OrdinaryReject(t, h, old)
	}
	s411OrdinaryReject(t, h, fresh)
	s411CookieWrongCSRF(t, h, c, e, fresh)
	if rotated {
		var revoked bool
		err := h.pool.QueryRow(h.ctx, `SELECT revoked_at=$2 AND revoke_reason=$3 AND csrf_token_digest<>$4 FROM sessions WHERE token_digest=$1`, recoveryAuthDigest(old.SessionToken), h.clock.Now(), loginSessionRotateReason, recoveryAuthDigest(fresh.CSRFToken)).Scan(&revoked)
		if err != nil || !revoked {
			t.Fatal("valid current cookie did not rotate/revoke atomically")
		}
	}
	_ = s411Image(t, h)
}

func s411CookieWrongCSRF(t *testing.T, h *s410PG, c context.Context, e *recoveryIdentityExecution, fresh LoginResult) {
	t.Helper()
	fresh.CSRFToken = "synthetic-wrong-csrf"
	if _, err := s411Resolve(t, h, c, e, fresh); !errors.Is(err, session.ErrCSRFFailed) {
		t.Fatal("fresh maintenance cookie bypassed CSRF")
	}
}
