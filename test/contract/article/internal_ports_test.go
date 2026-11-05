//go:build contract

package articlecontract

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"han/internal/article"
	"han/internal/audit"
	"han/internal/identity"
	"han/internal/mailer"
	"han/internal/platform"
	"han/internal/privacy"
	"han/internal/recovery"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestArticleInternalPortsContract(t *testing.T) {
	t.Run("PrincipalAndRoleBoundary", portPrincipals)
	t.Run("AuthorReaderContract", portAuthorReader)
	t.Run("CanonicalAdapter", portCanonical)
	t.Run("CleanupAndSnapshotOrder", portCleanup)
	t.Run("ProcessingGateAndPurpose", portProcessing)
	t.Run("ErrorRegistryAndRetry", portErrors)
	t.Run("NoKeyedIdempotency", portNoKey)
	t.Run("RecoveryDTOBoundary", portRecovery)
}

// scriptTx/rows return fixed SQL facts and record calls. They do not model a DB
// or reproduce any transition, proof validation, Gate, lease, or retry algorithm.
type sqlStep struct {
	kind, contains, name string
	values               []any
	rows                 [][]any
	err                  error
	check                func(context.Context, []any)
}
type scriptTx struct {
	pgx.Tx
	t                  *testing.T
	steps              []sqlStep
	events             *[]string
	commits, rollbacks int
	rows               []*scriptRows
}

func (tx *scriptTx) step(ctx context.Context, kind, sql string, args []any) sqlStep {
	tx.t.Helper()
	if len(tx.steps) == 0 {
		tx.t.Fatalf("unexpected %s SQL call", kind)
	}
	s := tx.steps[0]
	tx.steps = tx.steps[1:]
	if s.kind != kind || !strings.Contains(strings.Join(strings.Fields(sql), " "), s.contains) {
		tx.t.Fatalf("SQL script expected %s/%s, got %s", s.kind, s.name, kind)
	}
	if tx.events != nil {
		*tx.events = append(*tx.events, s.name)
	}
	if s.check != nil {
		s.check(ctx, args)
	}
	return s
}
func (tx *scriptTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	s := tx.step(ctx, "exec", sql, args)
	return pgconn.NewCommandTag("UPDATE 1"), s.err
}
func (tx *scriptTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	s := tx.step(ctx, "row", sql, args)
	return fixedRow{tx.t, s.values, s.err}
}
func (tx *scriptTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	s := tx.step(ctx, "query", sql, args)
	r := &scriptRows{t: tx.t, values: s.rows}
	tx.rows = append(tx.rows, r)
	return r, s.err
}
func (tx *scriptTx) Commit(context.Context) error {
	tx.commits++
	if tx.events != nil {
		*tx.events = append(*tx.events, "commit")
	}
	return nil
}
func (tx *scriptTx) Rollback(context.Context) error { tx.rollbacks++; return nil }
func (tx *scriptTx) assertConsumed() {
	tx.t.Helper()
	if len(tx.steps) != 0 {
		tx.t.Fatalf("unconsumed SQL script steps=%d", len(tx.steps))
	}
	for _, r := range tx.rows {
		if !r.closed || r.errCalls == 0 {
			tx.t.Fatal("rows not closed/Err not checked")
		}
	}
}

type fixedRow struct {
	t      *testing.T
	values []any
	err    error
}

func (r fixedRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	assignValues(r.t, dest, r.values)
	return nil
}
func assignValues(t *testing.T, dest, values []any) {
	t.Helper()
	if len(dest) != len(values) {
		t.Fatalf("Scan columns=%d expected=%d", len(dest), len(values))
	}
	for i, d := range dest {
		out := reflect.ValueOf(d).Elem()
		if values[i] == nil {
			out.Set(reflect.Zero(out.Type()))
			continue
		}
		v := reflect.ValueOf(values[i])
		if v.Type().AssignableTo(out.Type()) {
			out.Set(v)
		} else if v.Type().ConvertibleTo(out.Type()) {
			out.Set(v.Convert(out.Type()))
		} else {
			t.Fatalf("Scan column %d cannot assign %s to %s", i, v.Type(), out.Type())
		}
	}
}

type scriptRows struct {
	pgx.Rows
	t                *testing.T
	values           [][]any
	index            int
	closed           bool
	scanErr, rowsErr error
	errCalls         int
}

func (r *scriptRows) Next() bool {
	if r.index >= len(r.values) {
		return false
	}
	r.index++
	return true
}
func (r *scriptRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	assignValues(r.t, dest, r.values[r.index-1])
	return nil
}
func (r *scriptRows) Err() error { r.errCalls++; return r.rowsErr }
func (r *scriptRows) Close()     { r.closed = true }

type txSupplier struct {
	t        *testing.T
	txs      []*scriptTx
	calls    int
	contexts []context.Context
}

func (s *txSupplier) Begin(ctx context.Context) (pgx.Tx, error) {
	s.t.Helper()
	if s.calls >= len(s.txs) {
		s.t.Fatal("unexpected transaction")
	}
	tx := s.txs[s.calls]
	s.calls++
	s.contexts = append(s.contexts, ctx)
	return tx, nil
}
func (s *txSupplier) BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	if opts.IsoLevel != pgx.ReadCommitted {
		s.t.Fatal("dispatcher isolation changed")
	}
	return s.Begin(ctx)
}

type neverInsert struct{ t *testing.T }

func (n neverInsert) InsertArticle(context.Context, pgx.Tx, article.Article) error {
	n.t.Fatal("unexpected InsertArticle")
	return nil
}
func command(t *testing.T, txs *txSupplier, orphan *article.OrphanArticlesCanonicalDependencies, disposition *article.OrphanDispositionDependencies, ids article.ArticleIDGenerator) *article.CommandService {
	t.Helper()
	s, err := article.NewCommandService(article.CommandDependencies{Transactions: txs, Repository: neverInsert{t}, IDs: ids, Clock: fixedClock{}, Orphaning: orphan, Disposition: disposition})
	must(t, err)
	return s
}

func portPrincipals(t *testing.T) {
	t.Run("actual_content_authorization", portContentAuthorization)
	for _, name := range []string{"MarkViolation", "AdminList", "AdminDetail", "PublishOrphan", "TransferOrphan", "DeleteOrphan"} {
		t.Run(name, func(t *testing.T) {
			f := newTransport(t)
			e := endpointFor(t, f, name)
			p := principal("member")
			w := serve(t, e.fn, &p, request(e.method, e.path, e.body))
			assertError(t, w, 403, platform.CodeAuthForbidden)
			if f.port.calls != 0 {
				t.Fatal("actual admin Handler did not reject before port")
			}
		})
	}
	t.Run("actual_command_non_author", func(t *testing.T) {
		s := command(t, &txSupplier{t: t}, nil, nil, fixedIDs{})
		p := principal("member")
		_, _, err := s.Moderate(context.Background(), p, articleID, 2, nil)
		if !errors.Is(err, article.ErrArticleForbidden) {
			t.Fatal("actual service authorized member governance")
		}
		_, err = s.PublishOrphan(context.Background(), p, articleID, 2)
		if !errors.Is(err, article.ErrArticleForbidden) {
			t.Fatal("actual service authorized member disposition")
		}
	})
}

type authorDB struct {
	t     *testing.T
	rows  *scriptRows
	err   error
	calls int
	ctx   context.Context
	ids   []pgtype.UUID
}

func (d *authorDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	d.calls++
	d.ctx = ctx
	if !strings.Contains(sql, "FROM users") || len(args) != 1 {
		d.t.Fatal("reader query contract changed")
	}
	var ok bool
	d.ids, ok = args[0].([]pgtype.UUID)
	if !ok {
		d.t.Fatal("reader did not bind typed UUID array")
	}
	d.rows.index = 0
	d.rows.closed = false
	return d.rows, d.err
}
func portAuthorReader(t *testing.T) {
	for _, mode := range []string{"facts", "query_error", "scan_error", "rows_error", "empty"} {
		t.Run(mode, func(t *testing.T) { authorReaderCase(t, mode) })
	}
	t.Run("consumer_semantics", portReaderConsumers)
}
func authorReaderCase(t *testing.T, mode string) {
	const mixedCaseID = "abcdef01-2345-4678-9abc-def012345678"
	cause := errors.New(privateCanary)
	rows := &scriptRows{t: t, values: [][]any{{mixedCaseID, identity.UserStatusActive}, {targetID, identity.UserStatusSuspended}, {adminID, identity.UserStatusDeleted}}}
	db := &authorDB{t: t, rows: rows}
	switch mode {
	case "query_error":
		db.err = cause
	case "scan_error":
		rows.scanErr = cause
	case "rows_error":
		rows.rowsErr = cause
	}
	reader, err := identity.NewArticleIdentityReader(db)
	must(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ids := []string{strings.ToUpper(mixedCaseID), mixedCaseID, targetID, adminID, noticeID}
	if mode == "empty" {
		ids = nil
	}
	states, err := reader.GetAuthorStates(ctx, ids)
	if mode == "empty" {
		must(t, err)
		if db.calls != 0 || len(states) != 0 {
			t.Fatal("empty reader input queried DB")
		}
		return
	}
	if mode != "facts" {
		if !errors.Is(err, cause) {
			t.Fatal("reader lost underlying failure")
		}
	} else {
		must(t, err)
		if states[mixedCaseID] != identity.UserStatusActive || states[targetID] != identity.UserStatusSuspended || states[adminID] != identity.UserStatusDeleted || len(states) != 3 {
			t.Fatal("reader filled sparse/deleted states")
		}
	}
	assertAuthorReaderBinding(t, db, ctx, mode, mixedCaseID)
}

func assertAuthorReaderBinding(t *testing.T, db *authorDB, ctx context.Context, mode, mixedCaseID string) {
	t.Helper()
	if db.ctx != ctx || len(db.ids) != 4 {
		t.Fatal("reader context/dedup changed")
	}
	for i, want := range []string{mixedCaseID, targetID, adminID, noticeID} {
		if !db.ids[i].Valid || db.ids[i].String() != want {
			t.Fatal("reader UUID array lost canonical value/order")
		}
	}
	if mode != "query_error" && !db.rows.closed {
		t.Fatal("reader leaked rows")
	}
	if mode != "query_error" && mode != "scan_error" && db.rows.errCalls != 1 {
		t.Fatal("reader failed to check rows Err")
	}
}
func portReaderConsumers(t *testing.T) {
	for _, state := range []string{"missing", "active", "suspended", "deleted"} {
		t.Run(state, func(t *testing.T) {
			rows := &scriptRows{t: t}
			if state != "missing" {
				rows.values = [][]any{{authorID, identity.UserStatus(state)}}
			}
			db := &authorDB{t: t, rows: rows}
			reader, err := identity.NewArticleIdentityReader(db)
			must(t, err)
			q := &queryFacts{rows: []article.Article{baseArticle()}}
			s, err := article.NewArticleQueryService(q, reader, article.QueryConfig{DefaultLimit: 2, MaxLimit: 2, CandidateBatchSize: 3})
			must(t, err)
			page, err := s.ListArticles(context.Background(), article.ListArticlesRequest{})
			must(t, err)
			want := 0
			if state == "active" {
				want = 1
			}
			if len(page.Items) != want || q.rows[0].ViolationState != article.ViolationClear {
				t.Fatal("current reader fact changed member policy/article")
			}
			_, _, err = s.GetArticle(context.Background(), articleID)
			if state != "active" && !errors.Is(err, article.ErrArticleUnavailable) {
				t.Fatal("member missing/hidden fact did not404")
			}
			if state == "active" {
				must(t, err)
			}
			_, err = s.ListAdminArticles(context.Background(), principal("admin"), article.ListAdminArticlesRequest{})
			if state == "missing" || state == "deleted" {
				if platform.PublicCode(err) != platform.CodePlatformDependencyUnavailable {
					t.Fatal("Admin incomplete reader fact did not503")
				}
			} else {
				must(t, err)
			}
			if !rows.closed {
				t.Fatal("consumer left real reader rows open")
			}
		})
	}
}

type contentRepo struct {
	neverInsert
	row     article.Article
	updates int
}

func (r *contentRepo) LockArticleForUpdate(context.Context, pgx.Tx, string) (article.Article, error) {
	return r.row, nil
}
func (r *contentRepo) UpdateArticleContent(_ context.Context, _ pgx.Tx, row article.Article) error {
	r.updates++
	r.row = row
	return nil
}
func portContentAuthorization(t *testing.T) {
	for _, mode := range []string{"admin_non_author", "member_non_author", "stale", "owner_restricted"} {
		t.Run(mode, func(t *testing.T) {
			tx := &scriptTx{t: t}
			repo := &contentRepo{neverInsert: neverInsert{t}, row: baseArticle()}
			c := article.ViolationViolence
			repo.row.ViolationState = article.ViolationViolating
			repo.row.ViolationCategory = &c
			supply := &txSupplier{t: t, txs: []*scriptTx{tx}}
			s, err := article.NewCommandService(article.CommandDependencies{Transactions: supply, Repository: repo, IDs: fixedIDs{}, Clock: fixedClock{}})
			must(t, err)
			p := principal("member")
			version := int64(2)
			want := article.ErrArticleForbidden
			switch mode {
			case "admin_non_author":
				p = principal("admin")
			case "member_non_author":
				p.UserID = targetID
				want = article.ErrArticleUnavailable
			case "stale":
				version = 1
				want = article.ErrPreconditionFailed
			}
			row, err := s.ReplaceContent(context.Background(), p, articleID, version, article.CreateArticleInput{Title: " literal ", Body: " new🙂 "})
			if mode == "owner_restricted" {
				must(t, err)
				if row.Version != 3 || row.Title != " literal " || row.Body != " new🙂 " || row.ViolationState != article.ViolationViolating || row.ViolationCategory == nil || *row.ViolationCategory != c || repo.updates != 1 || tx.commits != 1 {
					t.Fatal("author replacement changed ownership/governance")
				}
			} else if !errors.Is(err, want) || repo.updates != 0 || tx.commits != 0 {
				t.Fatal("actual content command did not reject before update")
			}
			if tx.commits+tx.rollbacks != 1 {
				t.Fatal("content command did not release transaction")
			}
		})
	}
}

type canonicalRepo struct {
	t     *testing.T
	ctx   context.Context
	tx    pgx.Tx
	calls int
	err   error
}

func (r *canonicalRepo) LockCanonicalOrphanCandidates(ctx context.Context, tx pgx.Tx, id string) ([]article.Article, error) {
	if id != authorID {
		r.t.Fatal("canonical user changed")
	}
	r.ctx = ctx
	r.tx = tx
	return []article.Article{baseArticle()}, r.err
}
func (r *canonicalRepo) SetCanonicalOrphan(ctx context.Context, tx pgx.Tx, in article.CanonicalOrphanUpdate) error {
	r.calls++
	if ctx != r.ctx || tx != r.tx || in.ArticleID != articleID || in.DeletedUserID != authorID || in.ExpectedVersion != 2 || !in.UpdatedAt.Equal(fixtureTime) {
		r.t.Fatal("canonical update binding changed")
	}
	return nil
}

type auditSpy struct {
	t      *testing.T
	tx     pgx.Tx
	ctx    context.Context
	events *[]string
	err    error
	calls  int
}

func (a *auditSpy) Append(ctx context.Context, tx pgx.Tx, event audit.Event) error {
	a.calls++
	if a.events != nil {
		*a.events = append(*a.events, "audit")
	}
	if a.tx != nil && (a.tx != tx || a.ctx != ctx) {
		a.t.Fatal("audit not in caller transaction")
	}
	if event.Action == "article.orphan.transfer" {
		if event.ID != auditID || event.ActorKind != audit.ActorUser || event.ActorUserID == nil || *event.ActorUserID != adminID || event.ActorRole == nil || *event.ActorRole != "admin" || event.TargetType != "article" || event.TargetID != articleID || event.TraceID != traceID || event.Outcome != "succeeded" || event.Metadata["article_version"] != int64(3) {
			a.t.Fatal("actual transfer audit actor/target/trace/action/version changed")
		}
	}
	return a.err
}
func (a *auditSpy) Get(context.Context, audit.QueryRower, string) (audit.Event, error) {
	a.t.Fatal("unexpected audit Get collaborator")
	return audit.Event{}, nil
}
func portCanonical(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "repository_failure"}[fail], func(t *testing.T) {
			ctx, err := platform.ContextWithTraceID(context.Background(), traceID)
			must(t, err)
			tx := &scriptTx{t: t}
			repo := &canonicalRepo{t: t}
			cause := errors.New(privateCanary)
			if fail {
				repo.err = cause
			}
			auditor := &auditSpy{t: t, ctx: ctx, tx: tx}
			supplier := &txSupplier{t: t}
			s := command(t, supplier, &article.OrphanArticlesCanonicalDependencies{Repository: repo, Auditor: auditor, AuditIDs: fixedIDs{}, Clock: fixedClock{}}, nil, fixedIDs{})
			port := identity.ArticleAccountPortFunc(func(got context.Context, gotTx pgx.Tx, user string) (identity.CanonicalDeleteArticleResult, error) {
				r, e := s.OrphanArticlesCanonical(got, gotTx, user)
				return identity.CanonicalDeleteArticleResult{NewlyOrphanedCount: r.NewlyOrphanedCount, AlreadyOrphanedCount: r.AlreadyOrphanedCount, PendingOrphanCount: r.PendingOrphanCount}, e
			})
			got, err := port.OrphanArticlesCanonical(ctx, tx, authorID)
			if fail {
				if !errors.Is(err, cause) || got != (identity.CanonicalDeleteArticleResult{}) {
					t.Fatal("canonical failure lost or partial result")
				}
			} else {
				must(t, err)
				if got != (identity.CanonicalDeleteArticleResult{NewlyOrphanedCount: 1, PendingOrphanCount: 1}) || repo.calls != 1 || auditor.calls != 1 {
					t.Fatal("canonical result/count changed")
				}
			}
			if repo.ctx != ctx || repo.tx != tx || supplier.calls != 0 || tx.commits != 0 || tx.rollbacks != 0 {
				t.Fatal("adapter took ownership of caller transaction")
			}
		})
	}
}

type sequenceIDs struct {
	t      *testing.T
	values []string
	index  int
}

func (g *sequenceIDs) NewID() (string, error) {
	if g.index >= len(g.values) {
		g.t.Fatal("unexpected ID allocation")
	}
	id := g.values[g.index]
	g.index++
	return id, nil
}

type dispositionIdentity struct {
	t      *testing.T
	tx     pgx.Tx
	ctx    context.Context
	events *[]string
	err    error
	clears int
}

func (d *dispositionIdentity) LockTransferParties(ctx context.Context, tx pgx.Tx, target *string, source string) (article.DispositionParties, error) {
	*d.events = append(*d.events, "parties")
	if tx != d.tx || ctx != d.ctx || target == nil || *target != targetID || source != authorID {
		d.t.Fatal("parties binding changed")
	}
	a, b := "old@example.invalid", "new@example.invalid"
	id := targetID
	return article.DispositionParties{SourceID: source, SourceEmail: &a, TargetID: &id, TargetEmail: &b}, nil
}
func (d *dispositionIdentity) LockDeletionCandidate(context.Context, pgx.Tx, string) (article.DispositionDeletionCandidate, error) {
	d.t.Fatal("unexpected deletion candidate")
	return article.DispositionDeletionCandidate{}, nil
}
func (d *dispositionIdentity) ClearRetainedEmailIfEligible(ctx context.Context, tx pgx.Tx, source string, proof *article.DispositionCompletionProof) (bool, error) {
	d.clears++
	*d.events = append(*d.events, "cleanup")
	if proof == nil || ctx != d.ctx || tx != d.tx || source != authorID {
		d.t.Fatal("cleanup binding/proof missing")
	}
	r := proof.Reference()
	if r.Transaction != tx || r.Action != article.DispositionTransfer || r.ExpectedArticleID != articleID || r.BeforeVersion != 2 || r.NoticeID == nil || *r.NoticeID != noticeID || r.AuditEventID != auditID {
		d.t.Fatal("real minted proof reference differs")
	}
	if _, err := proof.VerifyCompletion(ctx, &scriptTx{t: d.t}); err == nil {
		d.t.Fatal("opaque proof accepted another transaction")
	}
	*r.NoticeID = "tamper"
	if *proof.Reference().NoticeID != noticeID {
		d.t.Fatal("proof reference exposed mutable authority")
	}
	return true, d.err
}

func dispositionScript(t *testing.T, events *[]string) *scriptTx {
	t.Helper()
	source := authorID
	target := targetID
	row := baseArticle()
	row.AuthorID = nil
	row.OrphanedFromUserID = &source
	row.PublicationState = article.PublicationHeld
	return &scriptTx{t: t, events: events, steps: []sqlStep{
		{kind: "exec", contains: "SET TRANSACTION", name: "isolation"},
		{kind: "row", contains: "SELECT id, orphaned_from_user_id, version", name: "locate", values: []any{articleID, &source, int64(2)}},
		{kind: "row", contains: "FOR UPDATE NOWAIT", name: "locked_preimage", values: articleValues(row)},
		{kind: "exec", contains: "INSERT INTO transfer_notices", name: "notice"},
		{kind: "exec", contains: "INSERT INTO transfer_notice_recipients", name: "snapshot_new", check: func(_ context.Context, a []any) {
			if a[1] != article.RecipientNewAuthor || a[2] != "new@example.invalid" {
				t.Fatal("new-author snapshot changed")
			}
		}},
		{kind: "exec", contains: "INSERT INTO transfer_notice_recipients", name: "snapshot_former", check: func(_ context.Context, a []any) {
			if a[1] != article.RecipientFormerDeletedAuthor || a[2] != "old@example.invalid" {
				t.Fatal("former-author snapshot changed")
			}
		}},
		{kind: "exec", contains: "UPDATE articles", name: "update"},
		{kind: "row", contains: "SELECT id, version, author_id", name: "proof_article", values: []any{articleID, int64(3), &target, nil, article.PublicationPublished, article.ViolationClear, nil, article.DeletionActive, nil}},
		{kind: "row", contains: "FROM audit_events", name: "proof_audit", values: []any{auditID, fixtureTime, audit.ActorUser, &[]string{adminID}[0], nil, &[]string{"admin"}[0], "article.orphan.transfer", "article", articleID, "succeeded", nil, traceID, nil, []byte(`{"article_version":3}`)}},
		{kind: "row", contains: "FROM transfer_notices", name: "proof_notice", values: []any{noticeID, articleID, targetID, authorID}},
		{kind: "query", contains: "FROM transfer_notice_recipients", name: "proof_recipients", rows: [][]any{{noticeID, article.RecipientNewAuthor, "new@example.invalid", article.RecipientPending}, {noticeID, article.RecipientFormerDeletedAuthor, "old@example.invalid", article.RecipientPending}}},
		{kind: "row", contains: "count(*) FILTER", name: "proof_counts", values: []any{int64(0), int64(0)}},
	}}
}
func articleValues(r article.Article) []any {
	return []any{r.ID, r.AuthorID, r.OrphanedFromUserID, r.Title, r.Body, r.PublicationState, r.ViolationState, r.ViolationCategory, r.DeletionState, r.DeletionOperationID, r.Version, r.CreatedAt, r.UpdatedAt}
}

func portCleanup(t *testing.T) {
	for _, mode := range []string{"success", "snapshot_failure", "audit_failure", "proof_mismatch", "cleanup_failure"} {
		t.Run(mode, func(t *testing.T) { cleanupCase(t, mode) })
	}
}
func cleanupCase(t *testing.T, mode string) {
	events := []string{}
	tx := dispositionScript(t, &events)
	ctx, err := platform.ContextWithTraceID(context.Background(), traceID)
	must(t, err)
	cause := errors.New(privateCanary)
	if mode == "snapshot_failure" {
		tx.steps[5].err = cause
		tx.steps = tx.steps[:6]
	}
	if mode == "audit_failure" {
		tx.steps = tx.steps[:7]
	}
	if mode == "proof_mismatch" {
		tx.steps[7].values[1] = int64(9)
	}
	identityPort := &dispositionIdentity{t: t, tx: tx, ctx: ctx, events: &events}
	if mode == "cleanup_failure" {
		identityPort.err = cause
	}
	auditor := &auditSpy{t: t, tx: tx, ctx: ctx, events: &events}
	if mode == "audit_failure" {
		auditor.err = cause
	}
	supplier := &txSupplier{t: t, txs: []*scriptTx{tx}}
	s := command(t, supplier, nil, &article.OrphanDispositionDependencies{Identity: identityPort, Auditor: auditor, RecipientMaxAttempts: 2}, &sequenceIDs{t: t, values: []string{auditID, noticeID}})
	result, err := s.TransferOrphan(ctx, principal("admin"), articleID, 2, targetID)
	assertCleanupResult(t, mode, tx, identityPort, result, err, events)
}
func assertCleanupResult(t *testing.T, mode string, tx *scriptTx, identityPort *dispositionIdentity, result article.OrphanDispositionResult, err error, events []string) {
	t.Helper()
	if mode == "success" {
		must(t, err)
		if result.NoticeID == nil || *result.NoticeID != noticeID || result.Article.AuthorID == nil || *result.Article.AuthorID != targetID || result.ETag != `"article-`+articleID+`-v3"` {
			t.Fatal("actual disposition result changed")
		}
		if tx.commits != 1 {
			t.Fatal("success not committed")
		}
	} else {
		if err == nil || result.NoticeID != nil || tx.commits != 0 {
			t.Fatal("failure returned partial success")
		}
	}
	if tx.rollbacks != 1 {
		t.Fatal("disposition transaction not released")
	}
	tx.assertConsumed()
	if mode == "success" || mode == "cleanup_failure" {
		if identityPort.clears != 1 {
			t.Fatal("cleanup missing")
		}
		assertBefore(t, events, "snapshot_new", "cleanup")
		assertBefore(t, events, "snapshot_former", "cleanup")
		assertBefore(t, events, "audit", "cleanup")
		assertBefore(t, events, "proof_counts", "cleanup")
	} else if identityPort.clears != 0 {
		t.Fatal("cleanup occurred before verified completion")
	}
}
func assertBefore(t *testing.T, events []string, a, b string) {
	t.Helper()
	left, right := -1, -1
	for i, e := range events {
		if e == a {
			left = i
		}
		if e == b {
			right = i
		}
	}
	if left < 0 || right < 0 || left >= right {
		t.Fatalf("order %s/%s changed: %v", a, b, events)
	}
}

type noticeGate struct {
	err   error
	calls int
	txs   []pgx.Tx
}

func (g *noticeGate) RequireOpen(context.Context) error { return g.err }
func (g *noticeGate) RequireOpenInTx(_ context.Context, tx pgx.Tx) error {
	g.calls++
	g.txs = append(g.txs, tx)
	return g.err
}

type recordingMailer struct {
	messages []mailer.Message
	cancel   context.CancelFunc
	contexts []context.Context
}

func (m *recordingMailer) Send(ctx context.Context, in mailer.Message) error {
	m.messages = append(m.messages, in)
	m.contexts = append(m.contexts, ctx)
	if m.cancel != nil {
		m.cancel()
	}
	return context.Canceled
}

type discardNoticeLog struct{}

func (discardNoticeLog) Log(context.Context, slog.Level, string, ...any) {}

type processingRecords struct {
	err   error
	calls int
}

func (r *processingRecords) CurrentProcessingRecord(context.Context) (privacy.ProcessingRecord, error) {
	r.calls++
	return privacy.ProcessingRecord{}, r.err
}
func noticeConfig() article.NoticeDispatcherConfig {
	return article.NoticeDispatcherConfig{PollInterval: time.Second, BatchLimit: 1, LeaseDuration: 10 * time.Second, DBOperationBudget: time.Second, SMTPOperationBudget: time.Second, BackoffInitial: time.Second, BackoffMax: 2 * time.Second}
}
func dispatcher(t *testing.T, supplier *txSupplier, gate article.NoticeProcessingGate, m *recordingMailer) (*article.TransferNoticeDispatcher, error) {
	t.Helper()
	return article.NewTransferNoticeDispatcher(article.NoticeDispatcherDependencies{Transactions: supplier, Repository: article.Repository{}, Gate: gate, Mailer: m, Audit: &auditSpy{t: t}, IDs: fixedIDs{}, NewTraceID: func() (string, error) { return traceID, nil }, Logger: discardNoticeLog{}, WorkerID: adminID}, noticeConfig())
}

func portProcessing(t *testing.T) {
	t.Run("nil_gate", func(t *testing.T) {
		m := &recordingMailer{}
		_, err := dispatcher(t, &txSupplier{t: t}, nil, m)
		if err == nil || len(m.messages) != 0 {
			t.Fatal("nil Gate accepted/sent")
		}
	})
	t.Run("closed_dispatcher", func(t *testing.T) {
		maintain := &scriptTx{t: t, steps: []sqlStep{{kind: "row", contains: "recipient.attempt_count=recipient.max_attempts", name: "maintain", err: pgx.ErrNoRows}}}
		claim := &scriptTx{t: t}
		g := &noticeGate{err: errors.New(privateCanary)}
		m := &recordingMailer{}
		supply := &txSupplier{t: t, txs: []*scriptTx{maintain, claim}}
		d, err := dispatcher(t, supply, g, m)
		must(t, err)
		must(t, d.Reconcile(context.Background()))
		d.Stop()
		if len(m.messages) != 0 || g.calls != 1 || g.txs[0] != claim || maintain.rollbacks != 1 || claim.rollbacks != 1 {
			t.Fatal("closed Gate did not stop/release before Send")
		}
		maintain.assertConsumed()
		claim.assertConsumed()
		select {
		case <-d.Done():
		default:
			t.Fatal("stopped dispatcher not done")
		}
	})
	t.Run("actual_processing_gate", func(t *testing.T) {
		r := &processingRecords{err: errors.New(privateCanary)}
		g, err := privacy.NewProcessingGate(platform.ProcessingGateConfig{RegionRef: "r", ControllerRef: "c", LegalBasisRef: "l", SMTPProcessorRef: "s"}, r)
		must(t, err)
		if platform.PublicCode(g.RequireOpen(context.Background())) != platform.CodePlatformProcessingGateClosed || r.calls != 1 {
			t.Fatal("actual ProcessingGate did not fail closed")
		}
	})
	for _, kind := range []article.RecipientKind{article.RecipientNewAuthor, article.RecipientFormerDeletedAuthor} {
		t.Run(string(kind), func(t *testing.T) { noticePurpose(t, kind) })
	}
	if !errors.Is(mailer.ValidateMessage(mailer.Message{Purpose: "unsupported", To: "synthetic@example.invalid", Subject: "s", Body: "b"}), mailer.ErrUnsupportedPurpose) {
		t.Fatal("unsupported purpose accepted")
	}
}

func noticeValues(kind article.RecipientKind, processing bool) []any {
	status := article.RecipientPending
	var worker *string
	var until, last *time.Time
	attempt := int64(0)
	if processing {
		status = article.RecipientProcessing
		v := adminID
		worker = &v
		u := fixtureTime.Add(10 * time.Second)
		until = &u
		l := fixtureTime
		last = &l
		attempt = 1
	}
	return []any{articleID, noticeID, kind, "synthetic@example.invalid", status, worker, until, nil, attempt, int64(2), last, nil, nil, fixtureTime, fixtureTime}
}
func noticePurpose(t *testing.T, kind article.RecipientKind) {
	maintain := &scriptTx{t: t, steps: []sqlStep{{kind: "row", contains: "recipient.attempt_count=recipient.max_attempts", name: "maintain", err: pgx.ErrNoRows}}}
	claimed := noticeValues(kind, true)
	claim := &scriptTx{t: t, steps: []sqlStep{{kind: "row", contains: "recipient.attempt_count<recipient.max_attempts", name: "due", values: noticeValues(kind, false)}, {kind: "row", contains: "SELECT clock_timestamp()", name: "clock", values: []any{fixtureTime}}, {kind: "row", contains: "UPDATE transfer_notice_recipients", name: "claim", values: claimed[1:]}}}
	preflight := &scriptTx{t: t, steps: []sqlStep{{kind: "row", contains: "FOR UPDATE OF recipient", name: "preflight", values: claimed}, {kind: "row", contains: "SELECT clock_timestamp()", name: "clock", values: []any{fixtureTime}}}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := &recordingMailer{cancel: cancel}
	g := &noticeGate{}
	supply := &txSupplier{t: t, txs: []*scriptTx{maintain, claim, preflight}}
	d, err := dispatcher(t, supply, g, m)
	must(t, err)
	must(t, d.Reconcile(ctx))
	d.Stop()
	if len(m.messages) != 1 || m.messages[0].Purpose != mailer.PurposeOrphanArticleTransferNotice || m.messages[0].To != "synthetic@example.invalid" || strings.Contains(m.messages[0].Body, baseArticle().Body) {
		t.Fatal("actual dispatcher purpose/privacy changed")
	}
	if g.calls != 2 || claim.commits != 1 || preflight.commits != 1 || supply.calls != 3 {
		t.Fatal("send preflight/transaction order changed")
	}
	for _, tx := range supply.txs {
		tx.assertConsumed()
		if tx.rollbacks != 1 {
			t.Fatal("notice transaction leaked")
		}
	}
	for _, ctx := range supply.contexts {
		if ctx.Err() == nil {
			t.Fatal("notice DB-operation context not cancelled")
		}
	}
	for _, c := range m.contexts {
		if c.Err() == nil {
			t.Fatal("mail operation context not cancelled")
		}
	}
	select {
	case <-d.Done():
	default:
		t.Fatal("dispatcher did not stop")
	}
}

func portErrors(t *testing.T) {
	for _, tc := range publicErrorCases() {
		t.Run(string(tc.code), func(t *testing.T) {
			def, ok := article.ArticleErrorDefinition(tc.code)
			if !ok || def.HTTPStatus != tc.status || def.Retry != tc.retry || def.Message != tc.message {
				t.Fatal("public status/retry/message contract changed")
			}
			w := serve(t, func(w http.ResponseWriter, r *http.Request) {
				_ = article.WriteArticleError(w, r, platform.NewPublicError(tc.code, errors.New(privateCanary)))
			}, nil, request(http.MethodGet, "/component", nil))
			assertError(t, w, tc.status, tc.code)
		})
	}
}
func portNoKey(t *testing.T) {
	f := newTransport(t)
	e := endpointFor(t, f, "Create")
	for i := 0; i < 2; i++ {
		r := request(e.method, e.path, e.body)
		r.Header.Set("Idempotency-Key", "same-synthetic-key")
		w := serve(t, e.fn, &e.actor, r)
		if w.Code != 201 {
			t.Fatal("article gained keyed rejection/replay contract")
		}
	}
	if f.port.calls != 2 {
		t.Fatal("article transport unexpectedly cached keyed request")
	}
}
func portRecovery(t *testing.T) {
	e := fixtureEvidence(t)
	raw := callProducer(t, &e, article.DeletionReplayRun{}, nil)
	decoded, err := recovery.DecodeReplayStatusV1(raw)
	must(t, err)
	if decoded.Ready == nil || decoded.Ready.RunID != e.RunID || decoded.Ready.BackupRef != e.BackupRef {
		t.Fatal("DTO-only recovery binding changed")
	}
	for _, key := range []string{"generation", "token", "deletion_operation_id", "email"} {
		if strings.Contains(string(raw), `"`+key+`"`) {
			t.Fatal("recovery DTO leaked internal authority")
		}
	}
}
