package identity

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"han/internal/recovery"

	"github.com/jackc/pgx/v5"
)

func TestRecoveryAdmissionStoreEncoding(t *testing.T) {
	t.Run("CanonicalNullFieldsAndInitialProof", func(t *testing.T) {
		t.Run("PublicationTransitionPreservesSource", func(t *testing.T) {
			run := recovery.RecoveryRun{RunID: "run", BackupRef: "backup"}
			publication := recovery.PublicationBinding{Ref: "publication", Digest: "sha256:source"}
			applyRecoveryChange(&run, recovery.RecoveryOutcomeArticleDeletionReady,
				recovery.RecoveryRunChange{ArticlePublication: &publication})
			if run.ArticlePublicationRef != publication.Ref || run.ArticlePublicationDigest != publication.Digest {
				t.Fatal("Store transition discarded the publication supplied by the real Coordinator contract")
			}
		})
		t.Run("FixedCanonicalGolden", testS49Golden)
		t.Run("EverySourceFieldAffectsDigest", testS49DigestFields)
		t.Run("InvalidFieldsClosed", testS49InvalidFields)
		t.Run("NullableScannerPreservesEmpty", testS49NullableScan)
		t.Run("RowsErrorAndPanicClose", testS49RowsClosure)
		t.Run("PublicationExactBinding", testS49PublicationBinding)
		t.Run("OwnedReadTransactionFinalization", testS49OwnedRead)
		t.Run("LegacyScannerTimeGuards", testS49LegacyScanner)
	})
}

const s49Hash = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const s49HMAC = "hmac-sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
const s49User = "11111111-1111-4111-8111-111111111111"
const s49Audit = "22222222-2222-4222-8222-222222222222"

var s49Time = time.Date(2026, 10, 29, 1, 2, 3, 123456000, time.UTC)

func s49Ptr[T any](v T) *T { return &v }

func s49Snapshot() recovery.AdmissionSnapshot {
	i := recovery.InstallationBinding{Version: recovery.InstallationVersion, DeploymentID: "deployment", InstallationID: "installation",
		TargetDigest: s49Hash, CommitmentRef: "commitment<&>", CommitmentDigest: s49Hash, InstalledAt: s49Time,
		RecoveryInitial: recovery.AnchorSnapshot{JournalID: "recovery", Namespace: recovery.RecoveryNamespaceV1, Epoch: 1},
		ArticleInitial:  recovery.AnchorSnapshot{JournalID: "article", Namespace: recovery.ArticleDeletionNamespaceV1, Epoch: 2}}
	r := recovery.AdmissionRun{RunID: "run", BackupRef: "backup", JournalKeyID: "key", JournalKeyVersion: "v1",
		JournalSecretFingerprint: s49HMAC, JournalStage: recovery.StageBegun, StartedAt: s49Time, Outcome: recovery.RecoveryOutcomeRunning,
		AccountCount: 1, PermanentCredentialCount: 2, SessionCount: 3, TemporaryCredentialCount: 4, IdempotencyCount: 5, PendingReviewCount: 6}
	return recovery.AdmissionSnapshot{Installation: i, Revision: recovery.StoreRevision{InstallationID: i.InstallationID, Counter: 7}, Runs: []recovery.AdmissionRun{r}}
}

func testS49Golden(t *testing.T) {
	s := s49Snapshot()
	raw, err := recovery.CanonicalAdmissionSnapshot(s)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"version":"han-recovery-admission/v1","installation":{"metadata_version":"han-recovery-install/v1","deployment_id":"deployment","installation_id":"installation","target_digest":"` + s49Hash + `","commitment_ref":"commitment<&>","commitment_digest":"` + s49Hash + `","recovery_journal_id":"recovery","recovery_epoch":1,"recovery_initial_sequence":0,"recovery_initial_hash":"","article_journal_id":"article","article_epoch":2,"article_initial_sequence":0,"article_initial_hash":"","revision":7,"installed_at":"2026-10-29T01:02:03.123456Z"},"runs":[{"run_id":"run","backup_ref":"backup","journal_key_id":"key","journal_key_version":"v1","journal_secret_fingerprint":"` + s49HMAC + `","journal_stage":"begun","started_at":"2026-10-29T01:02:03.123456Z","journal_consumed_at":null,"article_evidence_version":null,"article_evidence_digest":null,"article_evidence_ref":null,"article_evidence_completed_at":null,"article_namespace":null,"article_replay_verified_at":null,"article_publication_ref":null,"article_publication_digest":null,"completed_at":null,"outcome":"running","account_count":1,"permanent_credential_count":2,"session_count":3,"temporary_credential_count":4,"idempotency_count":5,"pending_review_count":6,"audit_event_id":null,"restore_binding":null,"initial_admin_user_id":null,"initial_admin_bootstrap_generation":null,"initial_admin_changed_generation":null,"initial_admin_change_audit_id":null,"initial_admin_changed_at":null}]}`
	if string(raw) != want {
		t.Fatalf("canonical bytes differ\ngot %s\nwant %s", raw, want)
	}
	s.Runs = []recovery.AdmissionRun{}
	raw, err = recovery.CanonicalAdmissionSnapshot(s)
	if err != nil || !bytes.HasSuffix(raw, []byte(`"runs":[]}`)) {
		t.Fatalf("empty encoding: %s %v", raw, err)
	}
	s.Runs = s49Snapshot().Runs
	x := s.Runs[0]
	x.RunID = "a"
	s.Runs = append(s.Runs, x)
	a, _ := recovery.CanonicalAdmissionSnapshot(s)
	s.Runs[0], s.Runs[1] = s.Runs[1], s.Runs[0]
	b, _ := recovery.CanonicalAdmissionSnapshot(s)
	if !bytes.Equal(a, b) || bytes.Index(a, []byte(`"run_id":"a"`)) > bytes.Index(a, []byte(`"run_id":"run"`)) {
		t.Fatal("run byte order changed")
	}
}

func testS49DigestFields(t *testing.T) {
	cases := map[string]func(*recovery.AdmissionSnapshot){
		"InstallationRevision": func(s *recovery.AdmissionSnapshot) { s.Revision.Counter++ },
		"InstalledAt":          func(s *recovery.AdmissionSnapshot) { s.Installation.InstalledAt = s49Time.Add(time.Microsecond) },
		"FailedRow":            func(s *recovery.AdmissionSnapshot) { s.Runs[0].Outcome = recovery.RecoveryOutcomeFailed },
		"NewRow":               func(s *recovery.AdmissionSnapshot) { r := s.Runs[0]; r.RunID = "other"; s.Runs = append(s.Runs, r) },
		"Audit":                func(s *recovery.AdmissionSnapshot) { s.Runs[0].AuditEventID = s49Ptr(s49Audit) },
		"Publication": func(s *recovery.AdmissionSnapshot) {
			s.Runs[0].ArticlePublicationRef = s49Ptr("p")
			s.Runs[0].ArticlePublicationDigest = s49Ptr(s49Hash)
		},
		"Restore":         func(s *recovery.AdmissionSnapshot) { s.Runs[0].RestoreBinding = s49Receipt() },
		"InitialPrepared": func(s *recovery.AdmissionSnapshot) { s49PrepareProof(&s.Runs[0]) },
		"InitialChanged": func(s *recovery.AdmissionSnapshot) {
			s49PrepareProof(&s.Runs[0])
			s.Runs[0].InitialAdminChangedGeneration = s49Ptr(int64(3))
			s.Runs[0].InitialAdminChangeAuditID = s49Ptr(s49Audit)
			s.Runs[0].InitialAdminChangedAt = s49Ptr(s49Time)
		},
	}
	for _, field := range []string{"AccountCount", "PermanentCredentialCount", "SessionCount", "TemporaryCredentialCount", "IdempotencyCount", "PendingReviewCount"} {
		cases[field] = func(s *recovery.AdmissionSnapshot) {
			v := reflect.ValueOf(&s.Runs[0]).Elem().FieldByName(field)
			v.SetInt(v.Int() + 1)
		}
	}
	base, _ := recovery.CanonicalAdmissionSnapshot(s49Snapshot())
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			s := s49Snapshot()
			change(&s)
			raw, e := recovery.CanonicalAdmissionSnapshot(s)
			if e != nil || recovery.SHA256Digest(raw) == recovery.SHA256Digest(base) {
				t.Fatalf("mutation missing: %v", e)
			}
		})
	}
}

func s49PrepareProof(r *recovery.AdmissionRun) {
	r.InitialAdminUserID, r.InitialAdminBootstrapGeneration = s49Ptr(s49User), s49Ptr(int64(2))
}

func s49Receipt() *recovery.RestoreReceipt {
	return &recovery.RestoreReceipt{Version: "han-db-restore/v1", RunID: "run", BackupRef: "backup", DeploymentID: "deployment", InstallationID: "installation", TargetDigest: s49Hash, ArtifactDigest: s49Hash, SourceApprovalDigest: s49Hash, RestoredSchemaDigest: s49Hash, ReceiptDigest: s49Hash, ArtifactBytes: 123, RestoredAt: s49Time}
}

func testS49InvalidFields(t *testing.T) {
	cases := map[string]func(*recovery.AdmissionSnapshot){
		"NullVsEmpty":       func(s *recovery.AdmissionSnapshot) { s.Runs[0].ArticleEvidenceRef = s49Ptr("") },
		"InvalidUTF8":       func(s *recovery.AdmissionSnapshot) { s.Runs[0].BackupRef = string([]byte{255}) },
		"Whitespace":        func(s *recovery.AdmissionSnapshot) { s.Runs[0].RunID = " run" },
		"PlainSHAIsNotHMAC": func(s *recovery.AdmissionSnapshot) { s.Runs[0].JournalSecretFingerprint = s49Hash },
		"NonUTC":            func(s *recovery.AdmissionSnapshot) { s.Runs[0].StartedAt = s49Time.In(time.FixedZone("other", 3600)) },
		"SubMicrosecond":    func(s *recovery.AdmissionSnapshot) { s.Runs[0].StartedAt = s49Time.Add(time.Nanosecond) },
		"NegativeCounter":   func(s *recovery.AdmissionSnapshot) { s.Runs[0].SessionCount = -1 },
		"BootstrapOverflow": func(s *recovery.AdmissionSnapshot) {
			s49PrepareProof(&s.Runs[0])
			s.Runs[0].InitialAdminBootstrapGeneration = s49Ptr(int64(math.MaxInt64))
		},
		"PartialChanged": func(s *recovery.AdmissionSnapshot) { s.Runs[0].InitialAdminChangedGeneration = s49Ptr(int64(3)) },
		"WrongRestore": func(s *recovery.AdmissionSnapshot) {
			s.Runs[0].RestoreBinding = s49Receipt()
			s.Runs[0].RestoreBinding.BackupRef = "wrong"
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			s := s49Snapshot()
			change(&s)
			if _, e := recovery.CanonicalAdmissionSnapshot(s); e == nil {
				t.Fatal("invalid projection accepted")
			}
		})
	}
}

type s49Scanner func(...any) error

func (s s49Scanner) Scan(dest ...any) error { return s(dest...) }

func s49Assign(dest []any, values []any) error {
	if len(dest) != len(values) {
		return fmt.Errorf("scan columns %d != %d", len(dest), len(values))
	}
	for i, p := range dest {
		v := reflect.ValueOf(p).Elem()
		if values[i] == nil {
			v.SetZero()
		} else {
			v.Set(reflect.ValueOf(values[i]))
		}
	}
	return nil
}

func s49RunValues() []any {
	r := s49Snapshot().Runs[0]
	return []any{r.RunID, r.BackupRef, r.JournalKeyID, r.JournalKeyVersion, r.JournalSecretFingerprint, r.JournalStage, r.StartedAt,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, r.Outcome, r.AccountCount, r.PermanentCredentialCount, r.SessionCount, r.TemporaryCredentialCount, r.IdempotencyCount, r.PendingReviewCount, nil,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil}
}

func testS49NullableScan(t *testing.T) {
	for _, empty := range []bool{false, true} {
		t.Run(fmt.Sprint(empty), func(t *testing.T) {
			values := s49RunValues()
			if empty {
				values[10] = s49Ptr("")
			}
			r, e := scanAdmissionRun(s49Scanner(func(dest ...any) error { return s49Assign(dest, values) }))
			if e != nil || (r.ArticleEvidenceRef == nil) == empty {
				t.Fatalf("NULL collapsed: %#v %v", r.ArticleEvidenceRef, e)
			}
		})
	}
}

type s49Rows struct {
	pgx.Rows
	values     [][]any
	at         int
	err        error
	scanErr    error
	closed     bool
	panicValue any
}

func (r *s49Rows) Next() bool {
	if r.at < len(r.values) {
		r.at++
		return true
	}
	return false
}
func (r *s49Rows) Scan(dest ...any) error {
	if r.panicValue != nil {
		panic(r.panicValue)
	}
	if r.scanErr != nil {
		return r.scanErr
	}
	return s49Assign(dest, r.values[r.at-1])
}
func (r *s49Rows) Err() error { return r.err }
func (r *s49Rows) Close()     { r.closed = true }

func testS49RowsClosure(t *testing.T) {
	cause := errors.New("late row stream error")
	r := &s49Rows{values: [][]any{s49RunValues()}, err: cause}
	runs, err := scanAdmissionRows(context.Background(), r)
	if !r.closed || runs != nil || !errors.Is(err, cause) {
		t.Fatalf("partial stream escaped: %v %v", runs, err)
	}
	marker := &struct{ label string }{"original scan panic"}
	r = &s49Rows{values: [][]any{s49RunValues()}, panicValue: marker}
	func() {
		defer func() {
			if recover() != marker || !r.closed {
				t.Error("scan panic lost or rows leaked")
			}
		}()
		_, _ = scanAdmissionRows(context.Background(), r)
	}()
}

func testS49PublicationBinding(t *testing.T) {
	r := recovery.RecoveryRun{RunID: "run", BackupRef: "backup"}
	v := recovery.ReadyReplayStatusV1{RunID: r.RunID, BackupRef: r.BackupRef, EvidenceVersion: recovery.ReplayEvidenceVersionV1, Namespace: recovery.ArticleDeletionNamespaceV1, EvidenceRef: "e", EvidenceDigest: s49Hash, JournalHeadSequence: 1, JournalHeadHash: s49Hash, CompletedAt: s49Time.Format("2006-01-02T15:04:05.000000Z")}
	p := recovery.PublicationBinding{Version: "article-replay-publication/v1", Ref: "p", Digest: s49Hash, RunID: r.RunID, BackupRef: r.BackupRef, EvidenceRef: v.EvidenceRef, EvidenceDigest: v.EvidenceDigest, JournalID: "article", Epoch: 2, Head: recovery.JournalHead{Sequence: 1, Hash: s49Hash}, EvidenceCompletedAt: s49Time, PublishedAt: s49Time}
	c := recovery.RecoveryRunChange{ArticleEvidence: &v, ArticlePublication: &p, ArticleEvidenceCompletedAt: s49Time}
	if !validRecoveryEvidence(r, c) {
		t.Fatal("valid Coordinator publication rejected")
	}
	for _, mutate := range []func(*recovery.PublicationBinding){func(x *recovery.PublicationBinding) { x.RunID = "other" }, func(x *recovery.PublicationBinding) { x.BackupRef = "other" }, func(x *recovery.PublicationBinding) { x.EvidenceRef = "other" }, func(x *recovery.PublicationBinding) { x.Digest = strings.ToUpper(s49Hash) }, func(x *recovery.PublicationBinding) { x.Head.Sequence++ }} {
		x := p
		mutate(&x)
		c.ArticlePublication = &x
		if validRecoveryEvidence(r, c) {
			t.Fatal("wrong publication accepted")
		}
	}
}

type s49ReadTx struct {
	pgx.Tx
	rows                   *s49Rows
	commitErr, rollbackErr error
	commits, rollbacks     int
	done                   bool
}

func (x *s49ReadTx) QueryRow(context.Context, string, ...any) pgx.Row {
	i := s49Snapshot().Installation
	return s49Scanner(func(dest ...any) error {
		return s49Assign(dest, []any{i.Version, i.DeploymentID, i.InstallationID, i.TargetDigest, i.CommitmentRef, i.CommitmentDigest,
			i.RecoveryInitial.JournalID, i.RecoveryInitial.Epoch, i.RecoveryInitial.Committed.Sequence, i.RecoveryInitial.Committed.Hash,
			i.ArticleInitial.JournalID, i.ArticleInitial.Epoch, i.ArticleInitial.Committed.Sequence, i.ArticleInitial.Committed.Hash, int64(7), i.InstalledAt})
	})
}
func (x *s49ReadTx) Query(context.Context, string, ...any) (pgx.Rows, error) { return x.rows, nil }
func (x *s49ReadTx) Commit(context.Context) error {
	x.commits++
	if !x.rows.closed {
		return errors.New("commit before rows closed")
	}
	if x.commitErr != nil {
		return x.commitErr
	}
	x.done = true
	return nil
}
func (x *s49ReadTx) Rollback(context.Context) error {
	x.rollbacks++
	if !x.rows.closed {
		return errors.New("rollback before rows closed")
	}
	if x.done {
		return pgx.ErrTxClosed
	}
	return x.rollbackErr
}

func testS49OwnedRead(t *testing.T) {
	cause, cleanup := errors.New("owned read failure"), errors.New("rollback failure")
	for _, name := range []string{"Success", "RowsError", "ScanError", "CommitError", "RollbackCauseJoin", "Panic"} {
		t.Run(name, func(t *testing.T) {
			x := &s49ReadTx{rows: &s49Rows{values: [][]any{s49RunValues()}}}
			switch name {
			case "RowsError":
				x.rows.err = cause
			case "ScanError":
				x.rows.scanErr = cause
			case "CommitError":
				x.commitErr = cause
			case "RollbackCauseJoin":
				x.rows.err = cause
				x.rollbackErr = cleanup
			case "Panic":
				x.rows.panicValue = cause
			}
			if name == "Panic" {
				func() {
					defer func() {
						if recover() != cause || !x.rows.closed || x.rollbacks != 1 {
							t.Error("owned panic/finalization changed")
						}
					}()
					_, _ = readOwnedAdmissionTransaction(context.Background(), x)
				}()
				return
			}
			s, e := readOwnedAdmissionTransaction(context.Background(), x)
			if !x.rows.closed || x.rollbacks != 1 {
				t.Fatal("owned resource leaked")
			}
			if name == "Success" {
				if e != nil || x.commits != 1 || len(s.Runs) != 1 || s.Revision.Digest == "" {
					t.Fatalf("positive read failed: %v", e)
				}
				return
			}
			if !errors.Is(e, cause) || !errors.Is(e, recovery.ErrAdmissionIncomplete) || s.Runs != nil {
				t.Fatalf("failure/partial snapshot escaped: %v", e)
			}
			if name == "RollbackCauseJoin" && !errors.Is(e, cleanup) {
				t.Fatal("cleanup cause discarded")
			}
		})
	}
}

func testS49LegacyScanner(t *testing.T) {
	for _, bad := range []time.Time{s49Time.Add(time.Nanosecond), s49Time.In(time.FixedZone("old-nonUTC", 3600))} {
		_, e := scanRecoveryRun(s49Scanner(func(dest ...any) error {
			return s49Assign(dest, []any{"run", "backup", "key", "v1", s49HMAC, recovery.StageBegun, nil, nil, nil, nil, nil, nil, nil, bad, nil, recovery.RecoveryOutcomeRunning})
		}))
		if !errors.Is(e, recovery.ErrInvalidRecoveryRun) {
			t.Fatal("legacy scanner time guard regressed", e)
		}
	}
}
