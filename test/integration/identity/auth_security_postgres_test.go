//go:build integration

package identity

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	identitypkg "han/internal/identity"
	"han/internal/platform"
	"han/internal/session"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Only these two parent tops belong in the S3.3 PG execution selector.
func TestS33LoginSessionPostgres(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  func(*s33Fixture)
	}{
		{"SuccessRotationDigestAndProtectedRequest", s33Rotation},
		{"PasswordFailureKeepsOldSession", s33PasswordFailure},
		{"AuditFailureRollsBackRotation", s33AuditFailure},
	} {
		t.Run(tc.name, func(t *testing.T) { tc.run(s33New(t)) })
		if t.Failed() {
			return
		}
	}
}

func TestS33LoginAbusePostgres(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  func(*s33Fixture)
	}{
		{"SubjectAcrossSources", s33Subject},
		{"SourceAcrossSubjects", s33Source},
		{"SubjectSourceOnly", s33Pair},
		{"ConcurrentMissingPair", func(f *s33Fixture) { s33Concurrent(f, "missing") }},
		{"ConcurrentExistingPair", func(f *s33Fixture) { s33Concurrent(f, "existing") }},
		{"ConcurrentSubjectAcrossSources", func(f *s33Fixture) { s33Concurrent(f, "subject") }},
		{"ConcurrentSourceAcrossSubjects", func(f *s33Fixture) { s33Concurrent(f, "source") }},
		{"ThirdBucketFailureEmptyRollback", func(f *s33Fixture) { s33BucketRollback(f, false) }},
		{"ThirdBucketFailureExistingRollback", func(f *s33Fixture) { s33BucketRollback(f, true) }},
		{"FreshProcessRetainsBackoff", s33FreshProcess},
	} {
		t.Run(tc.name, func(t *testing.T) { tc.run(s33New(t)) })
		if t.Failed() {
			return
		}
	}
}

func s33AssertSession(f *s33Fixture, input session.CreateInput, revoked bool) {
	var token, csrf []byte
	var user, kind string
	var credential, lineageOp, lineageCredential, rotated, reason *string
	var generation int64
	var issued, seen, absolute, idle time.Time
	var revokedAt *time.Time
	err := f.observer.QueryRow(f.ctx, `SELECT token_digest,csrf_token_digest,user_id::text,
		authenticated_credential_kind,authenticated_credential_id::text,auth_generation,
		lineage_operation_id::text,lineage_credential_id::text,issued_at,last_seen_at,
		absolute_expires_at,idle_expires_at,rotated_from_id::text,revoked_at,revoke_reason
		FROM sessions WHERE id=$1`, input.ID).Scan(&token, &csrf, &user, &kind, &credential, &generation,
		&lineageOp, &lineageCredential, &issued, &seen, &absolute, &idle, &rotated, &revokedAt, &reason)
	s33Must(f.t, err)
	tokenHash, csrfHash := sha256.Sum256([]byte(input.Token)), sha256.Sum256([]byte(input.CSRFToken))
	s33Require(f.t, len(token) == 32 && len(csrf) == 32 && bytes.Equal(token, tokenHash[:]) && bytes.Equal(csrf, csrfHash[:]), "stored credential digests differ")
	s33Require(f.t, user == input.UserID && kind == string(input.AuthenticatedCredentialKind) && credential == nil && generation == input.AuthGeneration, "stored credential binding differs")
	s33Require(f.t, lineageOp == nil && lineageCredential == nil && rotated == nil, "unexpected permanent-session lineage")
	s33Require(f.t, issued.Equal(input.IssuedAt) && seen.Equal(input.IssuedAt) && absolute.Equal(input.AbsoluteExpiresAt) && idle.Equal(input.IdleExpiresAt), "stored session lifetimes differ")
	if revoked {
		s33Require(f.t, revokedAt != nil && revokedAt.Equal(f.clock.at) && reason != nil && *reason == "login_rotation", "old session was not revoked by login")
	} else {
		s33Require(f.t, revokedAt == nil && reason == nil, "unexpected session revocation")
	}
}

func s33NoRawSecrets(f *s33Fixture, image s33Image, values ...string) {
	for _, raw := range image {
		for _, value := range values {
			s33Require(f.t, value != "" && !strings.Contains(raw, value), "raw credential reached a persisted row")
		}
	}
}

func s33Rotation(f *s33Fixture) {
	u := f.seedUser(true)
	before := f.snapshot("before-login")
	s33AssertSession(f, u.old, false)
	r := f.login(u.name, s33Password, "192.0.2.1:41000", u.old.Token)
	s33Envelope(f.t, r, 200, "")
	s33Require(f.t, r.token != "" && r.token != u.old.Token && r.csrf != "" && len(r.cookies) == 1, "missing fresh boundary credentials")
	c := r.cookies[0]
	s33Require(f.t, c.Name == platform.SessionCookieName && c.Secure && c.HttpOnly && c.SameSite == http.SameSiteLaxMode && c.Path == "/" && c.Domain == "" && c.MaxAge == 0 && c.Expires.IsZero(), "cookie contract differs")
	s33Require(f.t, len(f.issuer.inputs) == 2, "unexpected issuer calls")
	newSession := f.issuer.inputs[1]
	s33Require(f.t, r.token == newSession.Token && r.csrf == newSession.CSRFToken, "HTTP credentials do not match real issuance")
	after := f.snapshot("rotation-committed")
	s33SameTables(f.t, before, after, "sessions", "audit_events")
	s33AssertSession(f, u.old, true)
	s33AssertSession(f, newSession, false)
	var sessions, active int
	s33Must(f.t, f.observer.QueryRow(f.ctx, "SELECT count(*),count(*) FILTER(WHERE revoked_at IS NULL) FROM sessions").Scan(&sessions, &active))
	s33Require(f.t, sessions == 2 && active == 1, "rotation committed unexpected session set")
	f.assertAudit(u.id, "succeeded", "")
	s33NoRawSecrets(f, after, u.old.Token, u.old.CSRFToken, r.token, r.csrf, s33Password, s33FakeProbe)
	s33Require(f.t, f.observation.last.TransactionOutcome == identitypkg.LoginTransactionCommitted, "login did not report real commit")
	old, hits := f.probe(u.old.Token, u.old.CSRFToken)
	s33Envelope(f.t, old, 401, "auth.session_required")
	s33Require(f.t, hits == 0, "old token reached protected terminal")
	current, hits := f.probe(r.token, r.csrf)
	s33Require(f.t, current.status == 204 && hits == 1, "new token was not authorized")
	for _, csrf := range []string{"", "synthetic-forged-csrf"} {
		bad, hits := f.probe(r.token, csrf)
		s33Envelope(f.t, bad, 403, "auth.csrf_failed")
		s33Require(f.t, hits == 0, "invalid CSRF reached protected terminal")
	}
	s33SameTables(f.t, after, f.snapshot("after-protected-requests"))
}

func s33PasswordFailure(f *s33Fixture) {
	u := f.seedUser(true)
	before := f.snapshot("before-login")
	r := f.login(u.name, "synthetic-wrong-password", "192.0.2.1:41000", u.old.Token)
	s33Envelope(f.t, r, 401, "auth.login_failed")
	s33Require(f.t, len(f.issuer.inputs) == 1, "failed password issued new session")
	after := f.snapshot("failure-evidence-committed")
	s33SameTables(f.t, before, after, "login_attempt_buckets", "audit_events")
	s33AssertSession(f, u.old, false)
	f.bucketCounts("failure-buckets", s33ThreeCounts(1))
	f.verifyBuckets(f.observer, []identitypkg.LoginAttempt{f.attempt(u.name, "192.0.2.1:41000")})
	f.assertAudit(u.id, "failed", "auth.login_failed")
	allowed, hits := f.probe(u.old.Token, u.old.CSRFToken)
	s33Require(f.t, allowed.status == 204 && hits == 1, "old session lost after wrong password")
	s33SameTables(f.t, after, f.snapshot("old-session-still-allowed"))
}

func s33AuditFailure(f *s33Fixture) {
	u := f.seedUser(true)
	f.trigger("audit_events", "NEW.action='identity.login' AND NEW.outcome='succeeded'")
	before := f.snapshot("before-fault")
	r := f.login(u.name, s33Password, "192.0.2.1:41000", u.old.Token)
	s33Envelope(f.t, r, 503, "platform.database_unavailable")
	s33Require(f.t, f.auditor.state == "23514" && f.auditor.fault, "schema-local audit fault identity/SQLSTATE not observed")
	f.t.Log("fault_verified=true table=audit_events sqlstate=23514")
	s33Require(f.t, len(f.issuer.inputs) == 2, "rotation was not attempted before audit failure")
	s33Require(f.t, f.auditor.staged["sessions"] != before["sessions"], "rotation was not staged in the failing real transaction")
	after := f.snapshot("rotation-rolled-back")
	s33SameTables(f.t, before, after, "audit_events")
	s33AssertSession(f, u.old, false)
	f.assertAudit(u.id, "failed", "platform.database_unavailable")
	var count int
	s33Must(f.t, f.observer.QueryRow(f.ctx, "SELECT count(*) FROM sessions WHERE id=$1", f.issuer.inputs[1].ID).Scan(&count))
	s33Require(f.t, count == 0, "rolled-back replacement session committed")
	o := f.observation.last
	s33Require(f.t, o.TransactionOutcome == identitypkg.LoginTransactionRolledBack && o.RollbackOutcome == identitypkg.LoginRollbackSucceeded && o.AuditPersisted == identitypkg.LoginAuditPersistedYes, "rollback/failure-audit ownership not confirmed")
	allowed, hits := f.probe(u.old.Token, u.old.CSRFToken)
	s33Require(f.t, allowed.status == 204 && hits == 1, "old session lost after audit rollback")
	s33SameTables(f.t, after, f.snapshot("old-session-after-rollback"))
}

func s33ThreeCounts(n int64) map[string][]int64 {
	return map[string][]int64{"subject": {n}, "source": {n}, "subject_source": {n}}
}

func s33SharedCounts(kind string) map[string][]int64 {
	want := map[string][]int64{"subject": {1, 1}, "source": {1, 1}, "subject_source": {1, 1}}
	want[kind] = []int64{2}
	return want
}

func s33Subject(f *s33Fixture) {
	u := f.seedUser(false)
	before := f.snapshot("before-attempts")
	first := f.login(u.name, "synthetic-wrong-password", "192.0.2.1:41000", "")
	s33Envelope(f.t, first, 401, "auth.login_failed")
	s33Unified(f.t, first, f.login(u.name, "synthetic-wrong-password", "192.0.2.2:41000", ""))
	f.bucketCounts("subject-blocked", s33SharedCounts("subject"))
	f.verifyBuckets(f.observer, []identitypkg.LoginAttempt{f.attempt(u.name, "192.0.2.1:41000"), f.attempt(u.name, "192.0.2.2:41000")})
	f.clock.at = f.clock.at.Add(time.Minute)
	s33Decision(f, f.attempt(u.name, "192.0.2.3:41000"), identitypkg.LoginAbuseSubjectBlocked)
	s33Unified(f.t, first, f.login(u.name, s33Password, "192.0.2.3:41000", ""))
	s33SameTables(f.t, before, f.snapshot("account-block-response"), "login_attempt_buckets", "audit_events")
}

func s33Source(f *s33Fixture) {
	u, v := f.seedUser(false), f.seedUser(false)
	before := f.snapshot("before-attempts")
	s33Envelope(f.t, f.login(u.name, "synthetic-wrong-password", "192.0.2.1:41000", ""), 401, "auth.login_failed")
	s33Envelope(f.t, f.login(v.name, "synthetic-wrong-password", "192.0.2.1:41000", ""), 401, "auth.login_failed")
	f.bucketCounts("source-blocked", s33SharedCounts("source"))
	f.verifyBuckets(f.observer, []identitypkg.LoginAttempt{f.attempt(u.name, "192.0.2.1:41000"), f.attempt(v.name, "192.0.2.1:41000")})
	f.clock.at = f.clock.at.Add(time.Minute)
	s33Decision(f, f.attempt(u.name, "192.0.2.1:41000"), identitypkg.LoginAbuseSourceBlocked)
	for _, name := range []string{u.name, "synthetic-absent-user"} {
		r := f.login(name, s33Password, "192.0.2.1:41000", "")
		s33Envelope(f.t, r, 429, "request.rate_limited")
		s33Require(f.t, r.retry == "240", "source Retry-After did not derive from persisted deadline")
	}
	s33SameTables(f.t, before, f.snapshot("source-block-responses"), "login_attempt_buckets", "audit_events")
}

func s33SeedPair(f *s33Fixture, a identitypkg.LoginAttempt) {
	digest, version, err := f.digester.Digest(identitypkg.LoginBucketSubjectSource, a.Subject, a.SourceFingerprint)
	s33Must(f.t, err)
	tx := f.begin()
	defer s33Rollback(f.t, tx)
	s33Must(f.t, (identitypkg.PostgresLoginAbuseBucketStore{}).Save(f.ctx, tx, identitypkg.LoginAttemptBucket{
		Kind: identitypkg.LoginBucketSubjectSource, KeyDigest: digest, DigestKeyVersion: version,
		WindowStartedAt: a.Now, FailureCount: 2, BlockedUntil: a.Now.Add(5 * time.Minute), UpdatedAt: a.Now}))
	s33Must(f.t, tx.Commit(f.ctx))
}

func s33Pair(f *s33Fixture) {
	u, control := f.seedUser(false), f.seedUser(false)
	baseline := f.login(control.name, "synthetic-wrong-password", "192.0.2.9:41000", "")
	a := f.attempt(u.name, "192.0.2.1:41000")
	s33SeedPair(f, a)
	var n int
	s33Must(f.t, f.observer.QueryRow(f.ctx, `SELECT count(*) FROM login_attempt_buckets WHERE bucket_kind<>'subject_source' AND blocked_until>$1`, f.clock.at).Scan(&n))
	s33Require(f.t, n == 0, "pair input also blocked subject/source")
	before := f.snapshot("pair-only-blocked")
	f.clock.at = f.clock.at.Add(time.Minute)
	a.Now = f.clock.at
	s33Decision(f, a, identitypkg.LoginAbuseSubjectBlocked)
	s33Unified(f.t, baseline, f.login(u.name, s33Password, "192.0.2.1:41000", ""))
	s33SameTables(f.t, before, f.snapshot("pair-block-response"), "audit_events")
}

func s33Decision(f *s33Fixture, a identitypkg.LoginAttempt, kind identitypkg.LoginAbuseDecisionKind) {
	tx := f.begin()
	defer s33Rollback(f.t, tx)
	d, err := f.protector.Check(f.ctx, tx, a)
	s33Must(f.t, err)
	s33Require(f.t, d.Kind == kind && d.RetryAfter == 4*time.Minute, "persisted decision or remaining delay differs")
	s33Must(f.t, tx.Rollback(f.ctx))
}

func s33ConcurrentInputs(f *s33Fixture, mode string) (identitypkg.LoginAttempt, identitypkg.LoginAttempt, []identitypkg.LoginAttempt, map[string][]int64) {
	a := f.attempt("synthetic-concurrent-subject", "192.0.2.1:41000")
	b := a
	want := s33ThreeCounts(2)
	prior := []identitypkg.LoginAttempt{}
	if mode == "subject" {
		b = f.attempt(a.Subject, "192.0.2.2:41000")
		want = s33SharedCounts("subject")
	}
	if mode == "source" {
		b.Subject = "synthetic-other-subject"
		want = s33SharedCounts("source")
	}
	if mode == "existing" {
		f.record(a)
		want = s33ThreeCounts(3)
		prior = append(prior, a)
	}
	return a, b, prior, want
}

func s33Concurrent(f *s33Fixture, mode string) {
	a, b, prior, want := s33ConcurrentInputs(f, mode)
	before := f.snapshot("before-concurrency")
	first, second := f.begin(), f.begin()
	s33Require(f.t, first.pid != second.pid && first.pid != f.observer.Conn().PgConn().PID() && second.pid != f.observer.Conn().PgConn().PID(), "transactions/observer do not have independent PIDs")
	ctx, cancel := context.WithCancel(f.ctx)
	defer func() { cancel(); s33Rollback(f.t, first); f.workers.Wait(); s33Rollback(f.t, second) }()
	s33Must(f.t, f.protector.RecordFailure(ctx, first, a))
	f.verifyBuckets(first, append(append([]identitypkg.LoginAttempt{}, prior...), a))
	staged, err := s33ReadImage(ctx, first, f.schema)
	s33Must(f.t, err)
	s33Must(f.t, f.saveImage("a-uncommitted", staged))
	start, result := make(chan struct{}), make(chan error, 1)
	f.workers.Add(1)
	go func() { defer f.workers.Done(); <-start; result <- f.protector.RecordFailure(ctx, second, b) }()
	close(start)
	f.waitBlocked(first, second)
	s33SameTables(f.t, before, f.snapshot("blocked-observer-baseline"))
	s33Must(f.t, first.Commit(ctx))
	select {
	case err = <-result:
		s33Must(f.t, err)
	case <-ctx.Done():
		s33Must(f.t, ctx.Err())
	}
	f.workers.Wait()
	aCommitted := f.snapshot("a-committed-b-uncommitted")
	s33Require(f.t, aCommitted["login_attempt_buckets"] != before["login_attempt_buckets"], "first transaction did not commit")
	f.verifyBuckets(f.observer, append(append([]identitypkg.LoginAttempt{}, prior...), a))
	staged, err = s33ReadImage(ctx, second, f.schema)
	s33Must(f.t, err)
	s33Must(f.t, f.saveImage("b-uncommitted", staged))
	s33Must(f.t, second.Commit(ctx))
	f.bucketCounts("both-committed", want)
	f.verifyBuckets(f.observer, append(prior, a, b))
	s33SameTables(f.t, before, f.snapshot("after-concurrency"), "login_attempt_buckets")
}

func s33BucketRollback(f *s33Fixture, existing bool) {
	a := f.attempt("synthetic-rollback-subject", "192.0.2.1:41000")
	if existing {
		f.record(a)
	}
	f.trigger("login_attempt_buckets", "NEW.bucket_kind='subject_source'")
	before := f.snapshot("before-partial-write-fault")
	tx := f.begin()
	defer s33Rollback(f.t, tx)
	err := f.protector.RecordFailure(f.ctx, tx, a)
	s33Require(f.t, s33InjectedFault(err, f.schema, "login_attempt_buckets"), "schema-local third-bucket fault identity/SQLSTATE not observed")
	f.t.Log("fault_verified=true table=login_attempt_buckets sqlstate=23514 prior_two_bucket_writes_verified=true")
	s33Must(f.t, tx.Rollback(f.ctx))
	s33SameTables(f.t, before, f.snapshot("all-bucket-writes-rolled-back"))
	if existing {
		f.bucketCounts("original-counts", s33ThreeCounts(1))
	} else {
		f.bucketCounts("still-empty", map[string][]int64{})
	}
}

func s33FreshProcess(f *s33Fixture) {
	base := f.clock.at
	subject := f.attempt("synthetic-child-subject", "192.0.2.1:41000")
	f.record(subject)
	f.record(f.attempt(subject.Subject, "192.0.2.2:41000"))
	source := f.attempt("synthetic-child-source-one", "192.0.2.3:41000")
	f.record(source)
	f.record(f.attempt("synthetic-child-source-two", "192.0.2.3:41000"))
	pair := f.attempt("synthetic-child-pair", "192.0.2.4:41000")
	s33SeedPair(f, pair)
	counts := map[string][]int64{"subject": {1, 1, 2}, "source": {1, 1, 2}, "subject_source": {1, 1, 1, 1, 2}}
	f.bucketCounts("parent-blocks-committed", counts)
	before := f.snapshot("before-process-probes")
	f.closePool(f.pool)
	f.poolClosed = true
	subject.SourceFingerprint = f.fingerprint("192.0.2.5:41000")
	source.Subject = "synthetic-child-source-three"
	for _, tc := range []struct {
		mode    string
		attempt identitypkg.LoginAttempt
		kind    identitypkg.LoginAbuseDecisionKind
	}{
		{"Subject", subject, identitypkg.LoginAbuseSubjectBlocked},
		{"Source", source, identitypkg.LoginAbuseSourceBlocked},
		{"SubjectSource", pair, identitypkg.LoginAbuseSubjectBlocked},
	} {
		tc.attempt.Now = base.Add(time.Minute)
		f.child(s33ProbeInput{Mode: tc.mode, Schema: f.schema, Now: base, Key: f.abuseKey, Attempt: tc.attempt, Kind: tc.kind, Counts: counts})
		s33SameTables(f.t, before, f.snapshot("after-child-"+tc.mode))
	}
}

// This top is NOT in the parent selector. Missing/invalid mode fails before DSN access.
func TestS33LoginAbuseChildProbe(t *testing.T) {
	input := s33ReadProbe(t)
	t.Run(input.Mode, func(t *testing.T) { s33RunProbe(t, input) })
}

func s33ReadProbe(t *testing.T) s33ProbeInput {
	t.Helper()
	var input s33ProbeInput
	raw, err := io.ReadAll(io.LimitReader(os.Stdin, 16385))
	if err != nil || len(raw) > 16384 {
		t.Fatal("helper mode required before DSN access")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&input); err != nil || (input.Mode != "Subject" && input.Mode != "Source" && input.Mode != "SubjectSource") {
		t.Fatal("helper mode required before DSN access")
	}
	var extra any
	s33Require(t, decoder.Decode(&extra) == io.EOF, "unexpected helper input suffix")
	s33Require(t, regexp.MustCompile(`^han_s33_[a-f0-9]{32}$`).MatchString(input.Schema), "invalid retained schema name")
	s33Require(t, len(input.Key) == 32 && !input.Now.IsZero() && input.Attempt.Now.Equal(input.Now.Add(time.Minute)), "invalid synthetic probe inputs")
	want := identitypkg.LoginAbuseSubjectBlocked
	if input.Mode == "Source" {
		want = identitypkg.LoginAbuseSourceBlocked
	}
	s33Require(t, input.Kind == want, "invalid probe expectation")
	return input
}

func s33RunProbe(t *testing.T, input s33ProbeInput) {
	f := &s33Fixture{t: t, schema: input.Schema, clock: &s33Clock{at: input.Now}, abuseKey: input.Key}
	f.ctx, f.cancel = context.WithTimeout(context.Background(), 8*time.Second)
	t.Cleanup(f.close)
	var err error
	f.dir, err = os.MkdirTemp("", f.schema+"_probe_") // Retained, including failures.
	s33Must(t, err)
	c := s33Config(t, f.schema)
	f.pool, err = pgxpool.NewWithConfig(f.ctx, c)
	s33Must(t, err)
	_, err = s33Preflight(f.ctx, f.pool, c, f.schema, true)
	s33Must(t, err)
	f.observer, err = f.pool.Acquire(f.ctx)
	s33Must(t, err)
	f.db = &s33DB{pool: f.pool}
	f.digester, err = identitypkg.NewHMACLoginAbuseDigester("s33_abuse_v1", input.Key)
	s33Must(t, err)
	f.protector, err = identitypkg.NewPersistentLoginAbuseProtector(identitypkg.PostgresLoginAbuseBucketStore{}, f.digester, s33Policy())
	s33Must(t, err)
	before := f.snapshot("probe-before")
	f.bucketCounts("probe-existing-counts", input.Counts)
	s33Decision(f, input.Attempt, input.Kind)
	s33SameTables(t, before, f.snapshot("probe-after"))
	t.Logf("S33_PROBE_OK mode=%s pid=%d schema=%s retained_dir=%s", input.Mode, os.Getpid(), input.Schema, f.dir)
}
