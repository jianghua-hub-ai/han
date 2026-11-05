//go:build integration

package recovery_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"han/internal/article"
	"han/internal/identity"
	"han/internal/platform"
	"han/internal/recovery"
	"han/internal/session"
)

// s417LogPorts records the real and controlled-adapter port inventory for one
// leaf. The real list is produced by this leaf; the adapter list is the
// controlled software media set from runtime_fixtures_test.go.
func s417LogPorts(t *testing.T, real string) {
	t.Helper()
	t.Logf("real=%s; adapter=softwareJournal,softwareExclusive,softwareLifecycle,softwareCommitments,softwareSourceVerifier,softwareDeletionJournal (controlled software media only; no POSIX journal/CAS/ACL/approved-backup/E2E claim)", real)
}

func s417LeafGenesisDeletionHistoricalRestart(t *testing.T) {
	t.Helper()
	s417LogPorts(t, "recovery.NewRuntime,recovery.NewInstallationExecutor,identity.NewRecoveryRunStore,article.NewDeletionFinalizer,article.Repository,audit.Store,identity.NewArticleIdentityLifecycle,session.NewStore,platform.NewHTTPAuthorizer(session.AccessGeneral),PostgreSQL schema")
	world := s417NewFullWorld(t)

	// 1. Real bound installation through the production installation algorithm.
	world.s417Install(t)
	t.Logf("install accepted: manifest=%s/%s commitment_phase=bound", world.manifest.DeploymentID, world.manifest.InstallationID)

	// 2. Daily deletion through the real finalizer against real PostgreSQL.
	userID, articleID, operationID := world.s417SeedPendingDeletion(t)
	traceID, err := platform.NewTraceID()
	if err != nil {
		t.Fatalf("generate platform trace ID failed: %v", err)
	}
	if err := world.finalizer.Finalize(world.f.ctx, operationID, traceID); err != nil {
		t.Fatalf("real deletion finalizer Finalize failed: %v", err)
	}
	var remainingArticles, auditEvents int
	var operationStatus string
	if err := world.f.pool.QueryRow(world.f.ctx, "SELECT count(*) FROM articles WHERE id=$1", articleID).Scan(&remainingArticles); err != nil {
		t.Fatalf("read deleted article row failed: %v", err)
	}
	if err := world.f.pool.QueryRow(world.f.ctx, "SELECT status FROM deletion_operations WHERE id=$1", operationID).Scan(&operationStatus); err != nil {
		t.Fatalf("read deletion operation status failed: %v", err)
	}
	if err := world.f.pool.QueryRow(world.f.ctx, "SELECT count(*) FROM audit_events").Scan(&auditEvents); err != nil {
		t.Fatalf("read audit events failed: %v", err)
	}
	tombstone, err := world.deletionJournal.FindByOperation(world.f.ctx, operationID)
	if err != nil {
		t.Fatalf("real deletion journal lost tombstone: %v", err)
	}
	if remainingArticles != 0 || operationStatus != "completed" || auditEvents == 0 {
		t.Fatalf("daily deletion not durable: articles=%d status=%q audit_events=%d", remainingArticles, operationStatus, auditEvents)
	}
	if tombstone.DeletionOperationID != operationID || tombstone.ArticleID != articleID || tombstone.Sequence <= 0 {
		t.Fatalf("tombstone binding mismatch: %#v", tombstone)
	}
	t.Logf("daily deletion durable: user=%s article=%s op=%s remaining_articles=%d op_status=%s audit_events=%d tombstone_seq=%d",
		userID, articleID, operationID, remainingArticles, operationStatus, auditEvents, tombstone.Sequence)

	// 3. Real session store and AccessGeneral authorization for the same user.
	// The unprotected probe rejects with a session-required envelope; the same
	// real session then resolves to the seeded principal.
	sessions, err := session.NewStore(session.StoreDependencies{
		Transactions:      world.f.pool,
		PasswordResetGate: identity.PasswordResetStore{},
	})
	if err != nil {
		t.Fatalf("real session store failed: %v", err)
	}
	sessionToken, csrfToken := "s417-session-"+s417NewUUID(), "s417-csrf-"+s417NewUUID()
	if _, err := sessions.Create(world.f.ctx, world.f.pool, session.CreateInput{
		ID:                          s417NewUUID(),
		UserID:                      userID,
		Token:                       sessionToken,
		CSRFToken:                   csrfToken,
		AuthenticatedCredentialKind: session.CredentialPermanent,
		AuthGeneration:              0,
		IssuedAt:                    world.f.now,
		AbsoluteExpiresAt:           world.f.now.Add(24 * time.Hour),
		IdleExpiresAt:               world.f.now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("real session Create failed: %v", err)
	}
	authorizer, err := platform.NewHTTPAuthorizer(sessions, func() time.Time { return world.f.now })
	if err != nil {
		t.Fatalf("real HTTP authorizer failed: %v", err)
	}
	principalProbe := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		principal, ok := platform.PrincipalFromContext(request.Context())
		if !ok || principal.UserID != userID {
			t.Fatalf("AccessGeneral principal missing or foreign: %#v", principal)
		}
		response.WriteHeader(http.StatusNoContent)
	})
	protected := authorizer.Protect(session.AccessGeneral, principalProbe)
	anonymous := httptest.NewRecorder()
	protected.ServeHTTP(anonymous, httptest.NewRequest(http.MethodGet, "/s417/session-probe", nil))
	if anonymous.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous AccessGeneral status=%d; want 401 body=%s", anonymous.Code, anonymous.Body.String())
	}
	if envelope := s417Envelope(t, anonymous); envelope.Code != platform.CodeAuthSessionRequired {
		t.Fatalf("anonymous AccessGeneral code=%q; want %q", envelope.Code, platform.CodeAuthSessionRequired)
	}
	authenticatedRequest := httptest.NewRequest(http.MethodGet, "/s417/session-probe", nil)
	authenticatedRequest.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: sessionToken})
	authenticated := httptest.NewRecorder()
	protected.ServeHTTP(authenticated, authenticatedRequest)
	if authenticated.Code != http.StatusNoContent {
		t.Fatalf("authenticated AccessGeneral status=%d; want 204 body=%s", authenticated.Code, authenticated.Body.String())
	}
	t.Logf("real session/authorizer: anonymous=401/%s authenticated=204 user=%s access=AccessGeneral",
		platform.CodeAuthSessionRequired, userID)

	// 3. Historical restart over the same schema and the SAME authoritative
	// article journal instance the finalizer wrote: no manual head alignment.
	anchor, token, err := world.deletionJournal.Load(world.f.ctx, recovery.ArticleDeletionNamespaceV1)
	if err != nil {
		t.Fatalf("single journal authority Load failed: %v", err)
	}
	if anchor.Committed.Sequence != uint64(tombstone.Sequence) || token == "" {
		t.Fatalf("finalizer journal head does not match tombstone: %#v token=%q tombstone_seq=%d", anchor, token, tombstone.Sequence)
	}
	world.s417WireRuntime(t)
	if err := world.runtime.RunNormal(world.f.ctx); err != nil {
		t.Fatalf("historical restart RunNormal failed: %v", err)
	}
	observation, err := world.runtime.Verify(world.f.ctx)
	if err != nil {
		t.Fatalf("historical restart Verify failed: %v", err)
	}
	if observation.State != "public_ready" || observation.Reason != "verified" {
		t.Fatalf("historical restart not accepted: %#v", observation)
	}
	t.Logf("historical restart accepted: head_seq=%d head_hash_prefix=%s revision=%s observation=%#v",
		anchor.Committed.Sequence, anchor.Committed.Hash[:12], token, observation)
	if err := world.runtime.Stop(world.f.ctx); err != nil {
		t.Fatalf("historical restart Stop failed: %v", err)
	}
	if holders := world.exclusive.holders(); holders != 0 {
		t.Fatalf("exclusive holders after restart Stop=%d; want 0", holders)
	}

	// 4. A new instance cannot claim recovered progress without the real
	// recovery ports: the rejection is explicit, never a fabricated Ready.
	world.s417WireRuntime(t)
	resumeErr := world.runtime.Resume(world.f.ctx, recovery.RunBinding{RunID: s417NewUUID(), BackupRef: "s417-backup"})
	if !errors.Is(resumeErr, recovery.ErrAdmissionBinding) {
		t.Fatalf("Resume without recovery ports err=%v; want ErrAdmissionBinding", resumeErr)
	}
	t.Logf("historical restart Resume rejected explicitly: err=%v", resumeErr)
}

func s417LeafCrossNamespaceABAAndExactPublication(t *testing.T) {
	t.Helper()
	s417LogPorts(t, "recovery.NewRuntime,recovery.NewInstallationExecutor,identity.NewRecoveryRunStore,article.PostgresReplayEvidenceStore,article.NewReplayPublicationPorts,article.NewDeletionReplayProvider,article.NewProductionDeletionReplaySource,article.NewPostgresReplayFactReader,recovery.NewCoordinator,article.DeletionFinalizer,PostgreSQL(article_replay_publications,deletion_replay_evidence,deletion_replay_runs); adapter=softwareDeletionJournal(media),softwareJournal(media),softwareCommitments,softwareExclusive,softwareLifecycle,softwareSourceVerifier,s417IDAdapter,s417ClockAdapter,file_inventory_manifest")
	world := s417NewFullWorld(t)

	// 1. The real runtime binds the exact installation/article coordination
	// triple. A recovery-namespace journal identity used as the article journal
	// is a real cross-namespace misuse and must be rejected by the binding check.
	var coordination []string
	world.coordinateProbe = func(ctx context.Context, permit recovery.PhasePermit) error {
		manifest := world.manifest
		if _, _, err := world.runtime.CheckArticleCoordination(ctx, permit, manifest.InstallationID, manifest.Article.JournalID, manifest.Article.Epoch); err != nil {
			t.Fatalf("exact coordination triple rejected: %v", err)
		}
		cases := []struct {
			name, installation, journal string
			epoch                       uint64
		}{
			{"installation", manifest.InstallationID + "-other", manifest.Article.JournalID, manifest.Article.Epoch},
			{"journal", manifest.InstallationID, manifest.Article.JournalID + "-other", manifest.Article.Epoch},
			{"epoch", manifest.InstallationID, manifest.Article.JournalID, manifest.Article.Epoch + 1},
			{"cross_namespace", manifest.InstallationID, manifest.Recovery.JournalID, manifest.Article.Epoch},
		}
		for _, c := range cases {
			if _, _, err := world.runtime.CheckArticleCoordination(ctx, permit, c.installation, c.journal, c.epoch); !errors.Is(err, recovery.ErrAdmissionBinding) {
				t.Fatalf("coordination with wrong %s err=%v; want ErrAdmissionBinding", c.name, err)
			}
			coordination = append(coordination, c.name+"=ErrAdmissionBinding")
		}
		return nil
	}
	world.s417Install(t)
	t.Logf("coordination exact=accepted mismatches=%v", coordination)

	// 2. Real namespace identities: the recovery journal and the article
	// deletion journal keep independent journal id, epoch and head.
	recoveryBefore, err := world.journal.ReadVerifiedSnapshot(world.f.ctx, recovery.RecoveryNamespaceV1)
	if err != nil {
		t.Fatalf("recovery journal snapshot failed: %v", err)
	}
	articleBefore, err := world.deletionJournal.ReadVerifiedSnapshot(world.f.ctx, recovery.ArticleDeletionNamespaceV1)
	if err != nil {
		t.Fatalf("article journal snapshot failed: %v", err)
	}
	if recoveryBefore.JournalID == articleBefore.JournalID || recoveryBefore.Namespace == articleBefore.Namespace {
		t.Fatalf("namespaces not independent: recovery=%s/%s article=%s/%s",
			recoveryBefore.JournalID, recoveryBefore.Namespace, articleBefore.JournalID, articleBefore.Namespace)
	}
	if len(recoveryBefore.Entries) != 0 {
		t.Fatalf("recovery journal unexpectedly holds article entries: %d", len(recoveryBefore.Entries))
	}
	t.Logf("independent namespaces: recovery=%s/%s/%d head=%d article=%s/%s/%d head=%d",
		recoveryBefore.JournalID, recoveryBefore.Namespace, recoveryBefore.Epoch, recoveryBefore.Head.Sequence,
		articleBefore.JournalID, articleBefore.Namespace, articleBefore.Epoch, articleBefore.Head.Sequence)

	// 3. ABA on the single authoritative article journal. The head moves
	// A(seq1) -> B(seq2); the REAL publication store decides freshness against
	// the REAL verified snapshot. The media CAS checks in step 5 are
	// adapter-scoped conformance only and are never used as ABA proof.
	t.Logf("adapter_scoped=anchor_cas; not real_cas/journal; reason=windows_unsupported")
	if _, err := world.deletionJournal.Append(world.f.ctx, article.DeletionTombstoneRequest{
		ArticleID: s417NewUUID(), ArticleVersion: 1, DeletionOperationID: s417NewUUID(), DeletedAt: world.f.now,
	}); err != nil {
		t.Fatalf("ABA append A failed: %v", err)
	}
	snapshotA, err := world.deletionJournal.ReadVerifiedSnapshot(world.f.ctx, recovery.ArticleDeletionNamespaceV1)
	if err != nil {
		t.Fatalf("snapshot after A failed: %v", err)
	}
	headA := snapshotA.Head
	if headA.Sequence == 0 {
		t.Fatalf("article journal head after A is genesis: %#v", headA)
	}
	_, tokenAfterA, err := world.deletionJournal.Load(world.f.ctx, recovery.ArticleDeletionNamespaceV1)
	if err != nil {
		t.Fatalf("load revision after A failed: %v", err)
	}
	seedA := s417SeedPublishedClosure(t, world, snapshotA.JournalID, snapshotA.Epoch, headA)

	if _, err := world.deletionJournal.Append(world.f.ctx, article.DeletionTombstoneRequest{
		ArticleID: s417NewUUID(), ArticleVersion: 1, DeletionOperationID: s417NewUUID(), DeletedAt: world.f.now,
	}); err != nil {
		t.Fatalf("ABA append B failed: %v", err)
	}
	snapshotB, err := world.deletionJournal.ReadVerifiedSnapshot(world.f.ctx, recovery.ArticleDeletionNamespaceV1)
	if err != nil {
		t.Fatalf("snapshot after B failed: %v", err)
	}
	headB := snapshotB.Head
	if headB.Sequence != headA.Sequence+1 {
		t.Fatalf("ABA head did not advance: A=%#v B=%#v", headA, headB)
	}
	_, tokenAfterB, err := world.deletionJournal.Load(world.f.ctx, recovery.ArticleDeletionNamespaceV1)
	if err != nil {
		t.Fatalf("load revision after B failed: %v", err)
	}
	seedB := s417SeedPublishedClosure(t, world, snapshotB.JournalID, snapshotB.Epoch, headB)

	// 4. Real freshness decisions: an older published head that is still a
	// verified prefix is accepted as historical while the current head is newer;
	// an exact current head is accepted as fresh; a fabricated head, a foreign
	// journal identity, a foreign epoch and a tampered digest are rejected.
	closureA, err := world.chain.store.ReadPublishedClosure(world.f.ctx, seedA.request())
	if err != nil {
		t.Fatalf("real publication store rejected the historical fresh prefix A: %v", err)
	}
	if closureA.JournalID != snapshotA.JournalID || closureA.Epoch != snapshotA.Epoch ||
		closureA.HistoricalHead != headA || closureA.CurrentHead != headB ||
		closureA.Ready.EvidenceDigest != seedA.evidenceDigest {
		t.Fatalf("historical closure A mismatch: %#v want journal=%s epoch=%d historical=%#v current=%#v digest=%s",
			closureA, snapshotA.JournalID, snapshotA.Epoch, headA, headB, seedA.evidenceDigest)
	}
	t.Logf("publication history fresh prefix accepted: historical=%#v current=%#v journal=%s epoch=%d digest=%s",
		closureA.HistoricalHead, closureA.CurrentHead, closureA.JournalID, closureA.Epoch, closureA.Ready.EvidenceDigest)

	closureB, err := world.chain.store.ReadPublishedClosure(world.f.ctx, seedB.request())
	if err != nil {
		t.Fatalf("real publication store rejected the exact fresh head B: %v", err)
	}
	if closureB.HistoricalHead != headB || closureB.CurrentHead != headB {
		t.Fatalf("exact fresh closure B mismatch: historical=%#v current=%#v want %#v", closureB.HistoricalHead, closureB.CurrentHead, headB)
	}
	t.Logf("exact fresh head accepted: head=%#v", closureB.HistoricalHead)

	fabricated := recovery.JournalHead{Sequence: headB.Sequence, Hash: s417Digest("s417-aba-fabricated-head")}
	staleSeed := s417SeedPublishedClosure(t, world, snapshotB.JournalID, snapshotB.Epoch, fabricated)
	if _, err := world.chain.store.ReadPublishedClosure(world.f.ctx, staleSeed.request()); !errors.Is(err, recovery.ErrAnchorRollback) {
		t.Fatalf("stale publication head err=%v; want ErrAnchorRollback", err)
	}
	crossJournal := s417SeedPublishedClosure(t, world, world.journal.journalID, snapshotB.Epoch, headB)
	if _, err := world.chain.store.ReadPublishedClosure(world.f.ctx, crossJournal.request()); !errors.Is(err, recovery.ErrAdmissionBinding) {
		t.Fatalf("cross-namespace publication err=%v; want ErrAdmissionBinding", err)
	}
	epochRollback := s417SeedPublishedClosure(t, world, snapshotB.JournalID, snapshotB.Epoch+1, headB)
	if _, err := world.chain.store.ReadPublishedClosure(world.f.ctx, epochRollback.request()); !errors.Is(err, recovery.ErrAdmissionBinding) {
		t.Fatalf("epoch rollback publication err=%v; want ErrAdmissionBinding", err)
	}
	tampered := seedA.request()
	tampered.PublicationDigest = s417Digest("s417-tampered-publication-digest")
	if _, err := world.chain.store.ReadPublishedClosure(world.f.ctx, tampered); err == nil {
		t.Fatalf("tampered publication digest was accepted")
	} else if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("tampered publication digest err=%v; want pgx.ErrNoRows", err)
	}
	t.Logf("publication rejections: stale=ErrAnchorRollback cross_namespace=ErrAdmissionBinding epoch_rollback=ErrAdmissionBinding tampered=pgx.ErrNoRows")

	// 5. Real PostgreSQL rows carry exactly the seeded journal identity/head,
	// and the recovery namespace stayed untouched by the article ABA.
	var rowJournal, rowHash string
	var rowEpoch, rowSequence int64
	if err := world.f.pool.QueryRow(world.f.ctx, `SELECT journal_id, journal_epoch, journal_head_sequence, journal_head_hash
		FROM article_replay_publications WHERE publication_ref=$1`, seedA.publicationRef).
		Scan(&rowJournal, &rowEpoch, &rowSequence, &rowHash); err != nil {
		t.Fatalf("read real publication row failed: %v", err)
	}
	if rowJournal != snapshotA.JournalID || uint64(rowEpoch) != snapshotA.Epoch || uint64(rowSequence) != headA.Sequence || rowHash != headA.Hash {
		t.Fatalf("real publication row mismatch: %s/%d/%d/%s want %s/%d/%d/%s",
			rowJournal, rowEpoch, rowSequence, rowHash, snapshotA.JournalID, snapshotA.Epoch, headA.Sequence, headA.Hash)
	}
	recoveryAfter, err := world.journal.ReadVerifiedSnapshot(world.f.ctx, recovery.RecoveryNamespaceV1)
	if err != nil {
		t.Fatalf("recovery journal snapshot after ABA failed: %v", err)
	}
	if recoveryAfter.Head != recoveryBefore.Head || len(recoveryAfter.Entries) != 0 {
		t.Fatalf("article ABA/publication moved the recovery namespace: head %#v -> %#v entries=%d",
			recoveryBefore.Head, recoveryAfter.Head, len(recoveryAfter.Entries))
	}

	// 6. Adapter-scoped media conformance only (NOT ABA proof, NOT real CAS).
	rollback := recovery.AnchorSnapshot{JournalID: snapshotB.JournalID, Namespace: recovery.ArticleDeletionNamespaceV1,
		Epoch: snapshotB.Epoch, Committed: headA}
	if _, _, err := world.deletionJournal.CompareAndSwap(world.f.ctx, recovery.ArticleDeletionNamespaceV1, tokenAfterB, rollback); !errors.Is(err, recovery.ErrAnchorRollback) {
		t.Fatalf("adapter-scoped rollback err=%v; want ErrAnchorRollback", err)
	}
	forward := recovery.AnchorSnapshot{JournalID: snapshotB.JournalID, Namespace: recovery.ArticleDeletionNamespaceV1, Epoch: snapshotB.Epoch,
		Committed: recovery.JournalHead{Sequence: headB.Sequence + 1, Hash: s417Digest("s417-aba-forward")}}
	if _, _, err := world.deletionJournal.CompareAndSwap(world.f.ctx, recovery.ArticleDeletionNamespaceV1, tokenAfterA, forward); !errors.Is(err, recovery.ErrAnchorStaleRevision) {
		t.Fatalf("adapter-scoped stale revision err=%v; want ErrAnchorStaleRevision", err)
	}

	t.Logf("leaf2 summary: coordination=%v publication_row=%s/%d/%d/%s recovery_head=%d adapter_cas=rollback:ErrAnchorRollback,stale:ErrAnchorStaleRevision",
		coordination, rowJournal, rowEpoch, rowSequence, rowHash, recoveryAfter.Head.Sequence)
}

func s417LeafF001F002AndCanonicalIdempotencyReview(t *testing.T) {
	t.Helper()
	s417LogPorts(t, "identity.NewCanonicalDeleteService,identity.PostgresCanonicalDeleteRepository,identity.NewRecoveryIdentityExecution(recoveryExecutionGate),recovery.NewCoordinator,recovery.NewRuntime,article.NewCommandService,article.CanonicalOrphanRepository,audit.Store,PostgreSQL users/identity_identifiers/articles/audit_events/recovery_runs; adapter=s417PasswordHasher(boundary,unused)")
	world := s417NewFullWorld(t)
	wiring := world.s417WireCanonical(t)
	traceID, err := platform.NewTraceID()
	if err != nil {
		t.Fatalf("generate platform trace ID failed: %v", err)
	}
	tracedCtx, err := platform.ContextWithTraceID(world.f.ctx, traceID)
	if err != nil {
		t.Fatalf("attach trace ID to canonical delete context failed: %v", err)
	}
	world.f.ctx = tracedCtx
	ctx := world.f.ctx
	request := identity.CanonicalDeleteRequest{
		Actor: wiring.adminPrincipal(), TargetID: wiring.memberID, TargetRole: identity.RoleMember,
		Mode: identity.CanonicalDeleteDailyGovernance, Reason: identity.AccountDeletionGovernance, TraceID: traceID,
	}

	// s417CanonicalState reads the four durable tables the F002 zero-write
	// contract binds plus the identifier rows, so every rejection below can be
	// proven to have changed nothing.
	type s417CanonicalState struct {
		users, articles, audit, runs, identifiers int64
		userStatus                                string
		articleVersion                            string
	}
	read := func() s417CanonicalState {
		t.Helper()
		var state s417CanonicalState
		for _, probe := range []struct {
			name string
			sql  string
			dst  *int64
		}{
			{"users", "SELECT count(*) FROM users", &state.users},
			{"articles", "SELECT count(*) FROM articles", &state.articles},
			{"audit_events", "SELECT count(*) FROM audit_events", &state.audit},
			{"recovery_runs", "SELECT count(*) FROM recovery_runs", &state.runs},
			{"identity_identifiers", "SELECT count(*) FROM identity_identifiers", &state.identifiers},
		} {
			if err := world.f.pool.QueryRow(ctx, probe.sql).Scan(probe.dst); err != nil {
				t.Fatalf("read %s count failed: %v", probe.name, err)
			}
		}
		if err := world.f.pool.QueryRow(ctx, "SELECT status FROM users WHERE id=$1", wiring.memberID).Scan(&state.userStatus); err != nil {
			t.Fatalf("read user status failed: %v", err)
		}
		if err := world.f.pool.QueryRow(ctx, "SELECT version::text FROM articles WHERE id=$1", wiring.articleID).Scan(&state.articleVersion); err != nil {
			t.Fatalf("read article version failed: %v", err)
		}
		if acquired := world.f.pool.Stat().AcquiredConns(); acquired != 0 {
			t.Fatalf("pool acquired connections after canonical read = %d; want 0 (transaction leaked)", acquired)
		}
		return state
	}
	assertUnchanged := func(label string, before, after s417CanonicalState) {
		t.Helper()
		if before != after {
			t.Fatalf("%s wrote state: before=%#v after=%#v", label, before, after)
		}
		t.Logf("%s zero-write confirmed: users=%d articles=%d audit=%d recovery_runs=%d identifiers=%d status=%s version=%s",
			label, after.users, after.articles, after.audit, after.runs, after.identifiers, after.userStatus, after.articleVersion)
	}

	// 1. F001 pre-write guard: a non-canonical identifier row (username value no
	// longer matching the user row) is rejected before any write.
	canonicalUsername := "s417-member-" + wiring.memberID[:8]
	guardedBefore := read()
	if _, err := world.f.pool.Exec(ctx,
		"UPDATE identity_identifiers SET normalized_value=$2 WHERE user_id=$1 AND kind='username'",
		wiring.memberID, "s417-tampered-"+s417NewUUID()[:8]); err != nil {
		t.Fatalf("tamper identifier value failed: %v", err)
	}
	tamperedAfter := read()
	if _, err := wiring.coordinator.CanonicalDelete(ctx, request); platform.PublicCode(err) != platform.CodeIdentityInvalidState {
		t.Fatalf("non-canonical identifier err=%v code=%v; want CodeIdentityInvalidState", err, platform.PublicCode(err))
	}
	assertUnchanged("non-canonical identifier rejection", tamperedAfter, read())
	if _, err := world.f.pool.Exec(ctx,
		"UPDATE identity_identifiers SET normalized_value=$2 WHERE user_id=$1 AND kind='username'",
		wiring.memberID, canonicalUsername); err != nil {
		t.Fatalf("restore identifier value failed: %v", err)
	}
	restored := read()
	if restored != guardedBefore {
		t.Fatalf("identifier restore drifted state: before=%#v after=%#v", guardedBefore, restored)
	}
	if restored.userStatus != "active" || restored.articleVersion != "1" {
		t.Fatalf("rejected non-canonical delete wrote state: status=%s version=%s", restored.userStatus, restored.articleVersion)
	}
	t.Logf("non-canonical identifier value rejected before write: status=%s version=%s audit=%d", restored.userStatus, restored.articleVersion, restored.audit)

	// 2. Real canonical delete: identity tombstones, status and article orphaning
	// through the real identity coordinator over the real PostgreSQL rows.
	first, err := wiring.coordinator.CanonicalDelete(ctx, request)
	if err != nil {
		for cause := err; cause != nil; cause = errors.Unwrap(cause) {
			t.Logf("canonical delete cause chain: %T: %v", cause, cause)
		}
		t.Fatalf("real canonical delete failed: %v", err)
	}
	if first.NewlyOrphanedCount != 1 || first.AlreadyOrphanedCount != 0 ||
		first.PendingOrphanCount != 1 || first.EmailIdentifierFreed {
		t.Fatalf("canonical delete result mismatch: %#v", first)
	}
	var status string
	var activeIdentifiers int
	var authorID *string
	var orphanedFrom *string
	var publicationState string
	if err := world.f.pool.QueryRow(ctx, "SELECT status FROM users WHERE id=$1", wiring.memberID).Scan(&status); err != nil {
		t.Fatalf("read deleted user status failed: %v", err)
	}
	if err := world.f.pool.QueryRow(ctx,
		"SELECT count(*) FROM identity_identifiers WHERE user_id=$1 AND lifecycle='active'", wiring.memberID).Scan(&activeIdentifiers); err != nil {
		t.Fatalf("read identifier tombstones failed: %v", err)
	}
	if err := world.f.pool.QueryRow(ctx,
		"SELECT author_id, orphaned_from_user_id, publication_state FROM articles WHERE id=$1",
		wiring.articleID).Scan(&authorID, &orphanedFrom, &publicationState); err != nil {
		t.Fatalf("read orphaned article failed: %v", err)
	}
	// F001: no identifier may stay active and the article must be an owned orphan.
	if status != "deleted" || activeIdentifiers != 0 || authorID != nil ||
		orphanedFrom == nil || *orphanedFrom != wiring.memberID || publicationState != "held" {
		t.Fatalf("F001 invariants violated: status=%q active_identifiers=%d author_id=%v orphaned_from=%v publication=%q",
			status, activeIdentifiers, authorID, orphanedFrom, publicationState)
	}
	deleted := read()
	if deleted.audit <= restored.audit {
		t.Fatalf("canonical delete appended no audit row: %d -> %d", restored.audit, deleted.audit)
	}
	t.Logf("canonical delete durable: status=%s active_identifiers=%d author_id=%v orphaned_from=%v publication=%s audit=%d->%d result=%#v",
		status, activeIdentifiers, authorID, orphanedFrom, publicationState, restored.audit, deleted.audit, first)

	// 3. Canonical idempotency boundary: a repeated daily-governance call on the
	// already deleted target fails closed before any write, so a duplicate never
	// produces a second durable effect.
	replayBefore := read()
	if _, err := wiring.coordinator.CanonicalDelete(ctx, request); platform.PublicCode(err) != platform.CodeIdentityInvalidState {
		t.Fatalf("daily replay on deleted target err=%v code=%v; want CodeIdentityInvalidState", err, platform.PublicCode(err))
	}
	assertUnchanged("daily canonical replay", replayBefore, read())
	t.Logf("daily canonical replay failed closed: code=identity_invalid_state audit=%d version=%s", replayBefore.audit, replayBefore.articleVersion)

	// 4. F002 outside maintenance: without the real recovery run authority the
	// replay fails closed before any write.
	missingCapability, err := identity.NewRecoveryDeleteCapability("s417-missing-run", "s417-synthetic-backup")
	if err != nil {
		t.Fatalf("mint recovery capability failed: %v", err)
	}
	closedRequest := request
	closedRequest.Mode = identity.CanonicalDeleteRecoveryKept
	closedRequest.Recovery = &identity.RecoveryKeptDeletedRequest{Capability: missingCapability}
	closedBefore := read()
	if _, err := wiring.coordinator.CanonicalDelete(ctx, closedRequest); platform.PublicCode(err) != platform.CodeIdentityInvalidState {
		t.Fatalf("F002 without recovery authority err=%v code=%v; want CodeIdentityInvalidState", err, platform.PublicCode(err))
	}
	assertUnchanged("F002 closed-gate rejection", closedBefore, read())
	t.Logf("F002 outside maintenance failed closed: code=identity_invalid_state audit=%d recovery_runs=%d", closedBefore.audit, closedBefore.runs)

	// 5. F001/F002 maintenance scope decided by the REAL identity execution gate
	// (identity.NewRecoveryIdentityExecution -> recoveryExecutionGate). Every
	// call without a live runtime PhaseRecoveryIdentity permit is rejected by the
	// real gate, never by a test capability switch, and writes nothing.
	permitBefore := read()
	zeroPermit := recovery.PhasePermit{}
	zeroBinding := recovery.RunBinding{}
	if err := wiring.maintenance.ResolveRestrictedAccountReview(ctx, zeroPermit, zeroBinding, wiring.memberID, "kept_deleted", "s417-review"); err == nil {
		t.Fatalf("real maintenance gate accepted a restricted review without a live permit")
	} else {
		t.Logf("real maintenance gate rejected restricted review: code=%v err=%v", platform.PublicCode(err), err)
	}
	if err := wiring.maintenance.ResolveRecoveryIdempotency(
		ctx, zeroPermit, zeroBinding,
		[]byte("s417-non-canonical-source"), []byte(""), 0, "s417-detail", "s417-ref",
	); err == nil {
		t.Fatalf("real maintenance gate accepted a non-canonical idempotency input without a live permit")
	} else {
		t.Logf("real maintenance gate rejected non-canonical idempotency input: code=%v err=%v", platform.PublicCode(err), err)
	}
	if err := wiring.exec.CompleteIdentityReview(ctx, zeroPermit, zeroBinding); err == nil {
		t.Fatalf("real identity recovery execution accepted CompleteIdentityReview without a live permit")
	} else {
		t.Logf("real identity recovery execution rejected CompleteIdentityReview: code=%v err=%v", platform.PublicCode(err), err)
	}
	assertUnchanged("real maintenance-gate rejection", permitBefore, read())

	// 5b. H04 REAL maintenance-open: the SAME real
	// identity.NewRecoveryIdentityExecution maintenance consumer is driven with
	// the live PhaseRecoveryIdentity permit context that runtime.Enter hands its
	// identity executor, so recoveryExecutionGate.CheckRecoveryOpenGate
	// (internal/identity/recovery_execution.go:201) really opens and the real
	// canonical delete service injected at recovery_execution.go:127 commits the
	// recovery_kept_deleted path through real PostgreSQL
	// (internal/identity/deletion.go:792).
	live := s417NewRecoveryWorld(t, true, nil, func(w *s417RecoveryWorld) func(context.Context, recovery.PhasePermit, recovery.RunBinding) error {
		return func(liveCtx context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
			return w.s417BootstrapIdentity(liveCtx, p, b, true)
		}
	})
	liveRunID := "s417-run-maintenance-" + s417NewUUID()[:8]
	liveUser, liveArticle, liveOperation := live.s417SeedFinalizedDeletion(t)
	t.Logf("live-permit deletion seeded: user=%s article=%s operation=%s", liveUser, liveArticle, liveOperation)
	liveTrace, err := platform.NewTraceID()
	if err != nil {
		t.Fatalf("generate live-permit trace ID failed: %v", err)
	}
	tracedLiveCtx, err := platform.ContextWithTraceID(live.f.ctx, liveTrace)
	if err != nil {
		t.Fatalf("attach live-permit trace ID failed: %v", err)
	}
	liveCtx, liveCancel := context.WithTimeout(tracedLiveCtx, 120*time.Second)
	defer liveCancel()
	liveErr := live.runtime.Enter(liveCtx, recovery.RecoveryInput{
		RunBinding: recovery.RunBinding{RunID: liveRunID, BackupRef: live.backupRef},
		Secret:     append([]byte(nil), live.secret...),
	})
	if liveErr != nil {
		for cause := liveErr; cause != nil; cause = errors.Unwrap(cause) {
			t.Logf("live-permit Enter cause chain: %T: %v", cause, cause)
		}
		t.Fatalf("real runtime.Enter with live maintenance review failed: %v", liveErr)
	}
	liveRun, err := live.store.Load(liveCtx, liveRunID)
	if err != nil {
		t.Fatalf("live-permit run Load failed: %v", err)
	}
	var liveStatus, liveDecision, liveEvidence string
	if err := live.f.pool.QueryRow(liveCtx, "SELECT status FROM users WHERE id=$1", live.memberID).Scan(&liveStatus); err != nil {
		t.Fatalf("read live review target status failed: %v", err)
	}
	if err := live.f.pool.QueryRow(liveCtx,
		"SELECT decision, COALESCE(evidence_ref,'') FROM recovery_account_reviews WHERE recovery_run_id=$1 AND user_id=$2",
		liveRunID, live.memberID).Scan(&liveDecision, &liveEvidence); err != nil {
		t.Fatalf("read real recovery_account_reviews row failed: %v", err)
	}
	var canonicalDeletes int64
	if err := live.f.pool.QueryRow(liveCtx,
		"SELECT count(*) FROM audit_events WHERE action='identity.user.delete.canonical' AND target_id=$1 AND metadata->>'mode'='recovery_kept_deleted'",
		live.memberID).Scan(&canonicalDeletes); err != nil {
		t.Fatalf("read canonical delete audit failed: %v", err)
	}
	if liveRun.Outcome != recovery.RecoveryOutcomeSucceeded {
		t.Fatalf("live-permit run outcome=%s; want succeeded", liveRun.Outcome)
	}
	if liveStatus != "deleted" || liveDecision != "kept_deleted" || liveEvidence != "s417-recovery-review-evidence" || canonicalDeletes != 1 {
		t.Fatalf("H04 live maintenance review mismatch: status=%q decision=%q evidence=%q canonical_audit=%d",
			liveStatus, liveDecision, liveEvidence, canonicalDeletes)
	}
	t.Logf("REAL maintenance-open proven: run=%s outcome=%s member_status=%s review_decision=%s evidence=%s canonical_delete_audit=%d gate=recoveryExecutionGate(deletion.go:792) live_permit=PhaseRecoveryIdentity",
		liveRunID, liveRun.Outcome, liveStatus, liveDecision, liveEvidence, canonicalDeletes)
	t.Logf("unproven_port=canonical_recovery_replay_same_result; reason=recovery_kept_deleted_replay_requires_that_same_live_permit_plus_RecoveryRuns/RecoveryGate_wiring_inside_identity_execution; proven=non_canonical_and_unauthorized_replay_fail_closed_with_zero_write")

	// 6. Tampered tombstone state is rejected, never repaired.
	tamperBefore := read()
	if _, err := world.f.pool.Exec(ctx,
		"UPDATE identity_identifiers SET lifecycle='active' WHERE user_id=$1 AND kind='username'", wiring.memberID); err != nil {
		t.Fatalf("tamper identifier tombstone failed: %v", err)
	}
	tampered := read()
	if _, err := wiring.coordinator.CanonicalDelete(ctx, request); platform.PublicCode(err) != platform.CodeIdentityInvalidState {
		t.Fatalf("tampered canonical replay err=%v code=%v; want CodeIdentityInvalidState", err, platform.PublicCode(err))
	}
	assertUnchanged("tampered tombstone rejection", tampered, read())
	afterTamper := read()
	if afterTamper.users != tampered.users || afterTamper.articles != tampered.articles || afterTamper.audit != tampered.audit || afterTamper.runs != tampered.runs || afterTamper.identifiers != tampered.identifiers {
		t.Fatalf("tampered rejection committed a partial effect: before=%#v after=%#v", tampered, afterTamper)
	}
	t.Logf("tampered tombstone ack=withheld atomic_rollback=true users=%d articles=%d audit=%d recovery_runs=%d identifiers=%d (no partial commit)",
		afterTamper.users, afterTamper.articles, afterTamper.audit, afterTamper.runs, afterTamper.identifiers)
	if _, err := world.f.pool.Exec(ctx,
		"UPDATE identity_identifiers SET lifecycle='reserved' WHERE user_id=$1 AND kind='username'", wiring.memberID); err != nil {
		t.Fatalf("restore identifier tombstone failed: %v", err)
	}
	if restored := read(); restored != tamperBefore {
		t.Fatalf("tombstone restore drifted state: before=%#v after=%#v", tamperBefore, restored)
	}
	t.Logf("tampered tombstone replay rejected and never repaired: code=identity_invalid_state users=%d articles=%d audit=%d recovery_runs=%d",
		tamperBefore.users, tamperBefore.articles, tamperBefore.audit, tamperBefore.runs)
}

func s417LeafCrashConsumedWindowAndDrain(t *testing.T) {
	t.Helper()
	s417LogPorts(t, "recovery.NewRuntime,recovery.NewCoordinator,recovery.NewReplayStatusV1Decoder,article.NewPostgresReplayEvidenceStore,article.NewProductionDeletionReplaySource,article.NewDeletionReplayProvider,article.NewPostgresReplayFactReader,identity.NewRecoveryRunStore,PostgreSQL recovery_runs; file journal bytes: softwareJournal adapter")
	world := s417NewFullWorld(t)
	// 0. The REAL coordinator is constructed over the real store, journal, real
	// publication ports and the real gating provider. Nil ports are rejected by
	// the real constructor, and Consume without a live replay phase permit is
	// fail-closed.
	clock := func() time.Time { return world.f.now }
	if _, err := recovery.NewCoordinator(nil, world.journal, world.chain.provider, world.chain.ports, recovery.NewReplayStatusV1Decoder(nil), clock); !errors.Is(err, recovery.ErrAdmissionBinding) {
		t.Fatalf("NewCoordinator(nil store) err=%v; want ErrAdmissionBinding", err)
	}
	if _, err := world.coordinator.Consume(world.f.ctx, "s417-run-without-phase"); !errors.Is(err, recovery.ErrAdmissionBinding) {
		t.Fatalf("real Coordinator.Consume without live permit err=%v; want ErrAdmissionBinding", err)
	}
	// 1. The REAL runtime.Enter drives the unique consumed window:
	// recovery.NewRuntime -> Begin -> Replace -> real identity execution ->
	// real recovery.Coordinator.Consume -> real journal.MarkConsumed.
	t.Logf("real=identity.NewRecoveryIdentityExecution,recovery.NewCoordinator,identity.NewRecoveryRunStore; adapter=journal_media,restore_tool,lifecycle")
	entered := s417NewRecoveryWorld(t, false, nil, func(w *s417RecoveryWorld) func(context.Context, recovery.PhasePermit, recovery.RunBinding) error {
		return func(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
			return w.s417BootstrapIdentity(ctx, p, b, false)
		}
	})
	runID := "s417-run-enter-" + s417NewUUID()[:8]
	deletedUser, deletedArticle, deletedOperation := entered.s417SeedFinalizedDeletion(t)
	t.Logf("real finalized deletion seeded: user=%s article=%s operation=%s", deletedUser, deletedArticle, deletedOperation)
	traceID, err := platform.NewTraceID()
	if err != nil {
		t.Fatalf("generate recovery trace ID failed: %v", err)
	}
	tracedCtx, err := platform.ContextWithTraceID(entered.f.ctx, traceID)
	if err != nil {
		t.Fatalf("attach recovery trace ID failed: %v", err)
	}
	enterCtx, cancel := context.WithTimeout(tracedCtx, 120*time.Second)
	defer cancel()
	enterErr := entered.runtime.Enter(enterCtx, recovery.RecoveryInput{
		RunBinding: recovery.RunBinding{RunID: runID, BackupRef: entered.backupRef},
		Secret:     append([]byte(nil), entered.secret...),
	})
	if enterErr != nil {
		for cause := enterErr; cause != nil; cause = errors.Unwrap(cause) {
			t.Logf("real Enter cause chain: %T: %v", cause, cause)
		}
		t.Fatalf("real runtime.Enter failed: %v", enterErr)
	}
	if marks := entered.journal.consumedMarkCount(); marks != 1 {
		t.Fatalf("real MarkConsumed calls=%d; want exactly 1", marks)
	}
	callers := entered.journal.consumedCallerSnapshot()
	if len(callers) != 1 || !strings.Contains(callers[0], "coordinator.go") {
		t.Fatalf("MarkConsumed caller=%v; want the real recovery.Coordinator", callers)
	}
	var consumedEntries []recovery.JournalEntry
	if snapshot, err := entered.journal.ReadVerifiedSnapshot(enterCtx, recovery.RecoveryNamespaceV1); err != nil {
		t.Fatalf("read entered journal snapshot failed: %v", err)
	} else {
		for _, entry := range snapshot.Entries {
			if entry.RunID == runID && entry.Stage == recovery.StageConsumed {
				consumedEntries = append(consumedEntries, entry)
			}
		}
	}
	if len(consumedEntries) != 1 {
		t.Fatalf("consumed journal entries for run=%s count=%d; want exactly 1", runID, len(consumedEntries))
	}
	run, err := entered.store.Load(enterCtx, runID)
	if err != nil {
		t.Fatalf("real store.Load(%s) failed: %v", runID, err)
	}
	if run.Outcome != recovery.RecoveryOutcomeSucceeded || run.JournalStage != recovery.StageConsumed ||
		run.JournalConsumedAt == nil || !run.JournalConsumedAt.Equal(consumedEntries[0].RecordedAt) {
		t.Fatalf("real run terminal state mismatch: outcome=%s stage=%s consumed_at=%v entry=%v",
			run.Outcome, run.JournalStage, run.JournalConsumedAt, consumedEntries[0].RecordedAt)
	}
	observation, err := entered.runtime.Verify(enterCtx)
	if err != nil || observation.State != "public_ready" {
		t.Fatalf("runtime after Enter Verify=%#v err=%v; want public_ready", observation, err)
	}
	prepared, replaced, verified := entered.restore.counts()
	t.Logf("real runtime.Enter consumed window: run=%s outcome=%s stage=%s consumed_marks=%d caller=%s entry_seq=%d restore=prepare:%d replace:%d verify:%d observation=%s",
		runID, run.Outcome, run.JournalStage, entered.journal.consumedMarkCount(), callers[0],
		consumedEntries[0].Sequence, prepared, replaced, verified, observation.State)

	// 1a. Real negative paths: a second consumption after the phase closed and a
	// duplicate MarkConsumed are rejected by the real coordinator/journal.
	journalEntriesBefore := len(consumedEntries)

	_, duplicateErr := entered.journal.MarkConsumed(enterCtx, runID)
	if !errors.Is(duplicateErr, recovery.ErrJournalMismatch) {
		t.Fatalf("duplicate journal.MarkConsumed err=%v; want ErrJournalMismatch", duplicateErr)
	}
	consumedAfterDuplicate := 0
	if snapshot, err := entered.journal.ReadVerifiedSnapshot(enterCtx, recovery.RecoveryNamespaceV1); err != nil {
		t.Fatalf("read journal after duplicate consume failed: %v", err)
	} else {
		for _, entry := range snapshot.Entries {
			if entry.RunID == runID && entry.Stage == recovery.StageConsumed {
				consumedAfterDuplicate++
			}
		}
	}
	if consumedAfterDuplicate != journalEntriesBefore {
		t.Fatalf("rejected duplicate MarkConsumed appended a consumed journal entry: before=%d after=%d",
			journalEntriesBefore, consumedAfterDuplicate)
	}
	secondConsume, secondErr := entered.coordinator.Consume(enterCtx, runID)
	if secondErr == nil {
		t.Fatalf("second real Consume after Enter unexpectedly succeeded: %#v", secondConsume)
	}
	unknownResult, unknownErr := entered.coordinator.Consume(enterCtx, "s417-unknown-run")
	if unknownErr == nil {
		t.Fatalf("unknown run Consume unexpectedly succeeded: %#v", unknownResult)
	}
	if _, err := entered.store.Load(enterCtx, "s417-unknown-run"); err == nil {
		t.Fatalf("real store.Load accepted an unknown run")
	}
	t.Logf("real negative paths: duplicate_journal_mark=%v second_consume=%v unknown_consume=%v consumed_entries=%d mark_attempts=%d",
		duplicateErr, secondErr, unknownErr, consumedAfterDuplicate, entered.journal.consumedMarkCount())

	// 2. The real runtime owns the journal window: unbound append use is
	// rejected, a bound work lease may append, and a controlled unknown write
	// faults the runtime so later ordinary work stays isolated.
	world.s417RunNormal(t)
	rejectedBefore := world.exclusive.rejections()
	if lease, err := world.exclusive.Acquire(world.f.ctx); err == nil {
		_ = lease.Close()
		t.Fatalf("second deployment exclusive Acquire succeeded; real mutex must reject")
	} else if !errors.Is(err, recovery.ErrAdmissionClosed) {
		t.Fatalf("second deployment exclusive Acquire err=%v; want ErrAdmissionClosed", err)
	}
	if got := world.exclusive.rejections(); got != rejectedBefore+1 {
		t.Fatalf("exclusive rejection counter=%d; want %d (second Acquire must be denied)", got, rejectedBefore+1)
	}
	t.Logf("deployment exclusive mutex: first Acquire=held second Acquire=ErrAdmissionClosed rejected=%d holders=%d", rejectedBefore+1, world.exclusive.holders())
	if err := world.runtime.CheckJournalUse(world.f.ctx, "s417-unbound-append", recovery.JournalUseNewAppend); !errors.Is(err, recovery.ErrAdmissionBinding) {
		t.Fatalf("unbound journal append use err=%v; want ErrAdmissionBinding", err)
	}
	lease, err := world.runtime.Acquire(world.f.ctx, recovery.WorkRequest)
	if err != nil {
		t.Fatalf("real runtime Acquire(WorkRequest) failed: %v", err)
	}
	if err := world.runtime.CheckJournalUse(lease.Context(), "s417-bound-append", recovery.JournalUseNewAppend); err != nil {
		t.Fatalf("bound journal append use failed: %v", err)
	}
	barrier, err := world.runtime.PauseArticleAppend(lease.Context(), "s417-unknown-append")
	if err != nil {
		t.Fatalf("PauseArticleAppend failed: %v", err)
	}
	unknown := errors.New("s417-controlled-unknown-article-write")
	if err := world.runtime.FaultArticleAppend(lease.Context(), barrier, unknown); !errors.Is(err, recovery.ErrRecoveryWriteUnknown) {
		t.Fatalf("FaultArticleAppend err=%v; want ErrRecoveryWriteUnknown", err)
	}
	lease.Release()
	if _, err := world.runtime.Acquire(world.f.ctx, recovery.WorkRequest); !errors.Is(err, recovery.ErrAdmissionClosed) {
		t.Fatalf("Acquire after unknown err=%v; want ErrAdmissionClosed", err)
	}
	observation, verifyErr := world.runtime.Verify(world.f.ctx)
	if verifyErr == nil || observation.State == "public_ready" {
		t.Fatalf("unknown work leaked readiness: observation=%#v err=%v", observation, verifyErr)
	}
	if err := world.runtime.Stop(world.f.ctx); err != nil {
		t.Fatalf("Stop after unknown failed: %v", err)
	}
	if holders := world.exclusive.holders(); holders != 0 {
		t.Fatalf("exclusive holders after unknown Stop=%d; want 0", holders)
	}
	t.Logf("unknown isolated: unbound=ErrAdmissionBinding bound=ok fault=ErrRecoveryWriteUnknown acquire_after=ErrAdmissionClosed verify_state=%s holders=0",
		observation.State)

	// 3. Three real workers and their child lifecycle join: Stop must wait for
	// every held WorkLease, then join the ordinary lifecycle exactly once.
	workers := s417NewWorld(t)
	workers.s417RunNormal(t)
	kinds := []recovery.WorkKind{recovery.WorkRequest, recovery.WorkArticleDeletion, recovery.WorkPasswordReset}
	leases := make([]recovery.WorkLease, 0, len(kinds))
	for _, kind := range kinds {
		held, err := workers.runtime.Acquire(workers.f.ctx, kind)
		if err != nil {
			t.Fatalf("worker Acquire(%s) failed: %v", kind, err)
		}
		leases = append(leases, held)
	}
	stopDone := make(chan error, 1)
	go func() { stopDone <- workers.runtime.Stop(workers.f.ctx) }()
	select {
	case err := <-stopDone:
		t.Fatalf("Stop returned before workers released: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	for _, held := range leases {
		if err := held.Context().Err(); !errors.Is(err, context.Canceled) {
			t.Fatalf("worker lease not canceled by Stop: %v", err)
		}
		held.Release()
	}
	select {
	case err := <-stopDone:
		if err != nil {
			t.Fatalf("Stop join failed: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Stop did not join three released workers")
	}
	_, stopCalls := workers.lifecycle.counts()
	if holders := workers.exclusive.holders(); holders != 0 || stopCalls != 1 {
		t.Fatalf("worker join incomplete: holders=%d lifecycle_stop=%d", holders, stopCalls)
	}
	if _, err := workers.runtime.Acquire(workers.f.ctx, recovery.WorkRequest); !errors.Is(err, recovery.ErrAdmissionClosed) {
		t.Fatalf("Acquire after worker join err=%v; want ErrAdmissionClosed", err)
	}
	t.Logf("three worker join complete: workers=%d holders=0 lifecycle_stop=%d acquire_after=ErrAdmissionClosed", len(kinds), stopCalls)
}
func s417LeafCase18SameRevisionAndNoProductionClaim(t *testing.T) {
	t.Helper()
	s417LogPorts(t, "recovery.NewRuntime,recovery.NewInstallationExecutor,identity.NewRecoveryRunStore,session.NewStore,platform.NewHTTPAuthorizer,platform.NewRecoveryGuard,platform.NewOriginGuard,article.NewQueryHandler,article.NewHandler,article.NewPermanentDeletionCoordinator,article.NewCommandService,PostgreSQL schema")
	world := s417NewCase18World(t)
	s417Case18ClosedStream(t, world)
	s417Case18OpenStream(t, world)
	s417Case18Routing(t, world)
	s417Case18Conditional(t, world)
}

// s417Envelope decodes one public REST error envelope.
func s417Envelope(t *testing.T, recorder *httptest.ResponseRecorder) platform.Envelope {
	t.Helper()
	var envelope platform.Envelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode public envelope failed: %v body=%s", err, recorder.Body.String())
	}
	return envelope
}

// request drives the production exact-route transport skeleton with explicit
// Origin, session cookie, CSRF and If-Match evidence.
func (w *s417Case18World) request(
	t *testing.T,
	method, path string,
	withOrigin, withSession, withCSRF bool,
	ifMatch string,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, nil)
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	if withOrigin {
		request.Header.Set("Origin", w.publicOrigin)
	}
	if withSession {
		request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: w.sessionToken})
	}
	if withCSRF {
		request.Header.Set(platform.CSRFHeaderName, w.csrfToken)
	}
	if ifMatch != "" {
		request.Header.Set("If-Match", ifMatch)
	}
	recorder := httptest.NewRecorder()
	w.router.ServeHTTP(recorder, request)
	return recorder
}

// s417Case18ClosedStream proves that the same revision requested twice while
// the real runtime has no public stream yields two independent 503 responses
// and never reaches the Article handler, service, repository or author reader.
func s417Case18ClosedStream(t *testing.T, world *s417Case18World) {
	t.Helper()
	traceIDs := make([]string, 0, 2)
	for attempt := 1; attempt <= 2; attempt++ {
		recorder := world.request(t, http.MethodGet, world.articlePath(), true, true, false, "")
		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("closed stream GET attempt %d status=%d; want 503 body=%s", attempt, recorder.Code, recorder.Body.String())
		}
		envelope := s417Envelope(t, recorder)
		if envelope.Code != platform.CodePlatformRecoveryInProgress {
			t.Fatalf("closed stream GET attempt %d code=%q; want %q", attempt, envelope.Code, platform.CodePlatformRecoveryInProgress)
		}
		if envelope.TraceID == "" || envelope.TraceID != recorder.Header().Get(platform.TraceHeader) {
			t.Fatalf("closed stream GET attempt %d trace=%q header=%q", attempt, envelope.TraceID, recorder.Header().Get(platform.TraceHeader))
		}
		traceIDs = append(traceIDs, envelope.TraceID)
	}
	if traceIDs[0] == traceIDs[1] {
		t.Fatalf("two closed-stream requests reused trace ID %q", traceIDs[0])
	}
	handler, trace, service, repository, authorRead := world.counts()
	if handler != 0 || trace != 0 || service != 0 || repository != 0 || authorRead != 0 {
		t.Fatalf("closed stream reached Article downstream: handler=%d trace=%d service=%d repository=%d author_read=%d",
			handler, trace, service, repository, authorRead)
	}
	t.Logf("closed stream: statuses=[503 503] trace_ids=%v article_handler=%d article_service=%d article_repository=%d article_author_reader=%d",
		traceIDs, handler, service, repository, authorRead)
}

// s417Case18OpenStream completes the real genesis installation and opens the
// ordinary stream, then proves the same session and AccessGeneral principal can
// read the same revision with content loaded from real PostgreSQL.
func s417Case18OpenStream(t *testing.T, world *s417Case18World) {
	t.Helper()
	closed, err := world.full.runtime.Verify(world.f.ctx)
	if err != nil {
		t.Fatalf("real runtime closed Verify failed: %v", err)
	}
	if closed.State == "public_ready" {
		t.Fatalf("real runtime unexpectedly public_ready before RunNormal: %#v", closed)
	}
	if err := world.full.runtime.RunNormal(world.f.ctx); err != nil {
		t.Fatalf("real runtime RunNormal failed: %v", err)
	}
	observation, err := world.full.runtime.Verify(world.f.ctx)
	if err != nil || observation.State != "public_ready" {
		t.Fatalf("real runtime readiness state=%#v err=%v", observation, err)
	}
	recorder := world.request(t, http.MethodGet, world.articlePath(), true, true, false, "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("open stream GET status=%d; want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	world.etag = recorder.Header().Get("ETag")
	if world.etag == "" {
		t.Fatalf("open stream GET returned no ETag header")
	}
	var detail article.MemberArticleDetailView
	if err := json.Unmarshal(recorder.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode member article detail failed: %v body=%s", err, recorder.Body.String())
	}
	if detail.ID != world.articleID || detail.Title != "S4.17 Case18 title" || detail.Body != "S4.17 Case18 body" {
		t.Fatalf("detail mismatch: id=%q title=%q body=%q", detail.ID, detail.Title, detail.Body)
	}
	if detail.Author.ID == nil || *detail.Author.ID != world.userID {
		t.Fatalf("detail author=%v; want seeded author %q", detail.Author.ID, world.userID)
	}
	handler, trace, service, repository, authorRead := world.counts()
	if handler != 1 || trace != 1 || service != 1 || repository != 1 || authorRead != 1 {
		t.Fatalf("open stream downstream counts: handler=%d trace=%d service=%d repository=%d author_read=%d",
			handler, trace, service, repository, authorRead)
	}
	t.Logf("open stream: status=200 etag=%s detail_id=%s title=%q handler=%d trace=%d service=%d repository=%d author_read=%d",
		recorder.Header().Get("ETag"), detail.ID, detail.Title, handler, trace, service, repository, authorRead)
}

// s417Case18Routing verifies the exact-route contract around the protected
// Case18 handlers: unknown path 404, unavailable article 404 and unsupported
// method 405 with the production Allow header.
func s417Case18Routing(t *testing.T, world *s417Case18World) {
	t.Helper()
	missing := world.request(t, http.MethodGet, "/api/v1/articles/"+s417NewUUID(), true, true, false, "")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing article status=%d; want 404 body=%s", missing.Code, missing.Body.String())
	}
	if envelope := s417Envelope(t, missing); envelope.Code != article.CodeArticleUnavailable {
		t.Fatalf("missing article code=%q; want %q", envelope.Code, article.CodeArticleUnavailable)
	}
	unknown := world.request(t, http.MethodGet, "/api/v1/s417-unknown", true, true, false, "")
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("unknown route status=%d; want 404 body=%s", unknown.Code, unknown.Body.String())
	}
	if envelope := s417Envelope(t, unknown); envelope.Code != platform.CodeRequestRouteNotFound {
		t.Fatalf("unknown route code=%q; want %q", envelope.Code, platform.CodeRequestRouteNotFound)
	}
	unsupported := world.request(t, http.MethodPost, world.articlePath(), true, true, false, "")
	if unsupported.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unsupported method status=%d; want 405 body=%s", unsupported.Code, unsupported.Body.String())
	}
	if envelope := s417Envelope(t, unsupported); envelope.Code != platform.CodeRequestMethodNotAllowed {
		t.Fatalf("unsupported method code=%q; want %q", envelope.Code, platform.CodeRequestMethodNotAllowed)
	}
	if allow := unsupported.Header().Get("Allow"); allow != "DELETE, GET, HEAD, PUT" {
		t.Fatalf("unsupported method Allow=%q; want %q", allow, "DELETE, GET, HEAD, PUT")
	}
	t.Logf("routing: missing_article=404/%s unknown_route=404/%s unsupported_method=405/%s allow=%q",
		article.CodeArticleUnavailable, platform.CodeRequestRouteNotFound,
		platform.CodeRequestMethodNotAllowed, unsupported.Header().Get("Allow"))
}

// s417Case18Conditional covers legacy Origin/CSRF/If-Match compatibility on the
// real protected DELETE chain and then performs one successful deletion against
// real PostgreSQL.
func s417Case18Conditional(t *testing.T, world *s417Case18World) {
	t.Helper()
	noOrigin := world.request(t, http.MethodDelete, world.articlePath(), false, true, true, world.etag)
	if noOrigin.Code != http.StatusForbidden {
		t.Fatalf("missing origin status=%d; want 403 body=%s", noOrigin.Code, noOrigin.Body.String())
	}
	if envelope := s417Envelope(t, noOrigin); envelope.Code != platform.CodeAuthOriginRejected {
		t.Fatalf("missing origin code=%q; want %q", envelope.Code, platform.CodeAuthOriginRejected)
	}
	if got := world.originObserver.rejections.Load(); got != 1 {
		t.Fatalf("origin observer rejections=%d; want 1", got)
	}
	if reason, _ := world.originObserver.lastReason.Load().(string); reason != string(platform.OriginRejectionOriginMissing) {
		t.Fatalf("origin rejection reason=%q; want %q", reason, platform.OriginRejectionOriginMissing)
	}

	noCSRF := world.request(t, http.MethodDelete, world.articlePath(), true, true, false, world.etag)
	if noCSRF.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF status=%d; want 403 body=%s", noCSRF.Code, noCSRF.Body.String())
	}
	if envelope := s417Envelope(t, noCSRF); envelope.Code != platform.CodeAuthCSRFFailed {
		t.Fatalf("missing CSRF code=%q; want %q", envelope.Code, platform.CodeAuthCSRFFailed)
	}

	missingPrecondition := world.request(t, http.MethodDelete, world.articlePath(), true, true, true, "")
	if missingPrecondition.Code != http.StatusPreconditionRequired {
		t.Fatalf("missing If-Match status=%d; want 428 body=%s", missingPrecondition.Code, missingPrecondition.Body.String())
	}
	if envelope := s417Envelope(t, missingPrecondition); envelope.Code != article.CodeRequestPreconditionRequired {
		t.Fatalf("missing If-Match code=%q; want %q", envelope.Code, article.CodeRequestPreconditionRequired)
	}

	staleEtag := strings.Replace(world.etag, "-v1", "-v2", 1)
	stale := world.request(t, http.MethodDelete, world.articlePath(), true, true, true, staleEtag)
	if stale.Code != http.StatusPreconditionFailed {
		t.Fatalf("stale If-Match status=%d; want 412 body=%s", stale.Code, stale.Body.String())
	}
	if envelope := s417Envelope(t, stale); envelope.Code != article.CodeArticlePreconditionFailed {
		t.Fatalf("stale If-Match code=%q; want %q", envelope.Code, article.CodeArticlePreconditionFailed)
	}
	if got := world.originObserver.rejections.Load(); got != 1 {
		t.Fatalf("origin observer rejections after accepted-origin probes=%d; want 1", got)
	}

	accepted := world.request(t, http.MethodDelete, world.articlePath(), true, true, true, world.etag)
	if accepted.Code != http.StatusNoContent {
		t.Fatalf("real DELETE status=%d; want 204 body=%s", accepted.Code, accepted.Body.String())
	}
	var remaining, operations, audits int64
	if err := world.f.pool.QueryRow(world.f.ctx, "SELECT count(*) FROM articles WHERE id=$1", world.articleID).Scan(&remaining); err != nil {
		t.Fatalf("count deleted article failed: %v", err)
	}
	if err := world.f.pool.QueryRow(world.f.ctx,
		"SELECT count(*) FROM deletion_operations WHERE article_id=$1 AND status='completed'", world.articleID).Scan(&operations); err != nil {
		t.Fatalf("count completed deletion operation failed: %v", err)
	}
	if err := world.f.pool.QueryRow(world.f.ctx, "SELECT count(*) FROM audit_events").Scan(&audits); err != nil {
		t.Fatalf("count audit events failed: %v", err)
	}
	if remaining != 0 || operations != 1 || audits <= 0 {
		t.Fatalf("real deletion not durable: remaining_articles=%d completed_operations=%d audit_events=%d",
			remaining, operations, audits)
	}
	t.Logf("conditional: origin_missing=403/%s csrf_missing=403/%s if_match_missing=428/%s stale_if_match=412/%s accepted=204 remaining_articles=%d completed_operations=%d audit_events=%d origin_rejections=%d",
		platform.CodeAuthOriginRejected, platform.CodeAuthCSRFFailed,
		article.CodeRequestPreconditionRequired, article.CodeArticlePreconditionFailed,
		remaining, operations, audits, world.originObserver.rejections.Load())
}
