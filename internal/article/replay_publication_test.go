package article

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"han/internal/recovery"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestReplayPublicationClosure(t *testing.T) {
	t.Run("PublicationCurrentEvidenceAtomic", testPublicationConstructionSeal)
	t.Run("CompositeFKImmutableCanonical", testPublicationCanonical)
	t.Run("ExactHistoricalClosureAllowsRequiredReads", testPublicationReadProjection)
	t.Run("HeadAdvanceAndReadDrift", testPublicationPrefixDrift)
	t.Run("LegacyReassertionVersusTrustedImport", testPublicationMissingVerifier)
}

// This constructor-only probe cannot grant a phase or make an operational call succeed.
type publicationConstructorProbe struct {
	recovery.Runtime
	calls atomic.Int64
}

func (p *publicationConstructorProbe) ReadPermitBinding(context.Context) (recovery.PermitBinding, error) {
	p.calls.Add(1)
	return recovery.PermitBinding{}, errors.New("constructor probe is not authority")
}

type publicationConstructorJournal struct{ calls atomic.Int64 }

func (p *publicationConstructorJournal) ReadAllVerified(context.Context) ([]ArticleDeletionTombstone, error) {
	p.calls.Add(1)
	return nil, errors.New("constructor must not read")
}
func (p *publicationConstructorJournal) ReadVerifiedSnapshot(context.Context, string) (recovery.VerifiedJournalSnapshot, error) {
	p.calls.Add(1)
	return recovery.VerifiedJournalSnapshot{}, errors.New("constructor must not read")
}

type publicationIDs struct{ calls atomic.Int64 }

func (p *publicationIDs) NewID() (string, error) {
	return fmt.Sprintf("77777777-7777-4777-8777-%012d", p.calls.Add(1)), nil
}

type publicationClock struct{ calls atomic.Int64 }

func (p *publicationClock) Now() time.Time {
	p.calls.Add(1)
	return time.Date(2026, 10, 29, 10, 0, 0, 123456000, time.UTC)
}

func publicationConstructorFixture() (*PostgresReplayEvidenceStore, ReplayPublicationDependencies) {
	j := &publicationConstructorJournal{}
	source := publicationSourceReferences(j)
	return &PostgresReplayEvidenceStore{pool: &pgxpool.Pool{}}, ReplayPublicationDependencies{
		Source: source, Runtime: &publicationConstructorProbe{}, InstallationID: "synthetic-installation",
		PublicationIDs: &publicationIDs{}, Clock: &publicationClock{},
	}
}

func publicationSourceReferences(j VerifiedDeletionJournal) *ProductionDeletionReplaySource {
	return &ProductionDeletionReplaySource{journal: j, inventory: &FileBackupInventoryReader{},
		replayer: &replayRecorderSourceStub{}, operations: &staticReplayOperationReader{}, audits: &staticReplayAuditReader{}, now: time.Now}
}

func testPublicationConstructionSeal(t *testing.T) {
	t.Run("SamePointerPoolZeroPortIOAndNoRebind", func(t *testing.T) {
		store, d := publicationConstructorFixture()
		pool := store.pool
		result, err := NewReplayPublicationPorts(store, d)
		if err != nil || result != store || result.pool != pool {
			t.Fatalf("sameStore construction: %v", err)
		}
		if d.Runtime.(*publicationConstructorProbe).calls.Load() != 0 || d.Source.journal.(*publicationConstructorJournal).calls.Load() != 0 ||
			d.PublicationIDs.(*publicationIDs).calls.Load() != 0 || d.Clock.(*publicationClock).calls.Load() != 0 {
			t.Fatal("constructor invoked a port")
		}
		if _, err := NewReplayPublicationPorts(store, d); !errors.Is(err, errReplayPublicationSealed) {
			t.Fatal("second bind accepted")
		}
	})
	t.Run("EveryFirstFailureSeals", testPublicationAllFirstEntries)
	t.Run("ConstructionAndFirstReadRace", func(t *testing.T) {
		for range 32 {
			store, d := publicationConstructorFixture()
			var workers sync.WaitGroup
			workers.Add(2)
			go func() { defer workers.Done(); _, _ = NewReplayPublicationPorts(store, d) }()
			go func() { defer workers.Done(); _, _, _ = store.ReadCurrent(nil, "", "", nil) }()
			workers.Wait()
			if _, err := NewReplayPublicationPorts(store, d); !errors.Is(err, errReplayPublicationSealed) {
				t.Fatal("race allowed rebinding")
			}
		}
	})
}

func testPublicationAllFirstEntries(t *testing.T) {
	cases := map[string]func(*PostgresReplayEvidenceStore){
		"LoadAttempt": func(s *PostgresReplayEvidenceStore) { _, _ = s.LoadAttempt(nil, "", "") },
		"Begin": func(s *PostgresReplayEvidenceStore) {
			_, _ = s.Begin(nil, "", "", AttemptSnapshot{}, DeletionReplayFacts{}, "")
		},
		"Publish":                func(s *PostgresReplayEvidenceStore) { _, _ = s.Publish(nil, ReplayPublishRequest{}) },
		"Fail":                   func(s *PostgresReplayEvidenceStore) { _ = s.Fail(nil, "", "", AttemptSnapshot{}, "") },
		"ReadCurrent":            func(s *PostgresReplayEvidenceStore) { _, _, _ = s.ReadCurrent(nil, "", "", nil) },
		"ReadCurrentPublication": func(s *PostgresReplayEvidenceStore) { _, _ = s.ReadCurrentPublication(nil, "", "", "", "") },
		"ReadPublishedClosure":   func(s *PostgresReplayEvidenceStore) { _, _ = s.ReadPublishedClosure(nil, recovery.ClosureRequest{}) },
		"EnsureCurrentPublication": func(s *PostgresReplayEvidenceStore) {
			_, _ = s.EnsureCurrentPublication(nil, recovery.PhasePermit{}, "", "", "", "")
		},
		"ImportLegacyPublication": func(s *PostgresReplayEvidenceStore) {
			_, _ = s.ImportLegacyPublication(nil, recovery.PhasePermit{}, LegacyPublicationProof{})
		},
	}
	for name, first := range cases {
		t.Run(name, func(t *testing.T) {
			store, d := publicationConstructorFixture()
			first(store)
			if _, err := NewReplayPublicationPorts(store, d); !errors.Is(err, errReplayPublicationSealed) {
				t.Fatal("failure did not seal")
			}
		})
	}
}

func testPublicationCanonical(t *testing.T) {
	t.Run("Independent17FieldGolden", testPublicationGolden)
	t.Run("TypedNilRequiredDependencies", func(t *testing.T) {
		for _, name := range []string{"Source", "Runtime", "IDs", "Clock", "Journal", "Installation"} {
			t.Run(name, func(t *testing.T) {
				store, d := publicationConstructorFixture()
				switch name {
				case "Source":
					d.Source = nil
				case "Runtime":
					var p *publicationConstructorProbe
					d.Runtime = p
				case "IDs":
					var p *publicationIDs
					d.PublicationIDs = p
				case "Clock":
					var p *publicationClock
					d.Clock = p
				case "Journal":
					var p *publicationConstructorJournal
					d.Source.journal = p
				case "Installation":
					d.InstallationID = ""
				}
				if _, err := NewReplayPublicationPorts(store, d); err == nil {
					t.Fatal("missing required dependency accepted")
				}
			})
		}
	})
	t.Run("BigintAndGenerationTokenBounds", func(t *testing.T) {
		for _, mutate := range []func(*replayPublicationRecord){
			func(p *replayPublicationRecord) { p.Epoch = math.MaxUint64 },
			func(p *replayPublicationRecord) { p.Head.Sequence = math.MaxUint64 },
			func(p *replayPublicationRecord) { p.Generation = -1 },
			func(p *replayPublicationRecord) { p.Token = nil },
			func(p *replayPublicationRecord) { p.Generation = 0 },
			func(p *replayPublicationRecord) { p.Version = "unknown" },
			func(p *replayPublicationRecord) { p.SourceKind = "caller-approved" },
		} {
			p := publicationGoldenRecords(t)[0].record
			mutate(&p)
			if _, err := canonicalReplayPublication(p); err == nil {
				t.Fatal("invalid publication accepted")
			}
		}
	})
}

func testPublicationReadProjection(t *testing.T) {
	j, w, a, _ := actualAnchorCore(t, nil)
	lease, err := w.runtime.Acquire(context.Background(), recovery.WorkRequest)
	if err != nil {
		t.Fatal(err)
	}
	_, err = j.Append(lease.Context(), anchorRequest(1))
	lease.Release()
	if err != nil {
		t.Fatal(err)
	}
	source := publicationSourceReferences(j)
	reader, err := source.publicationJournal()
	if err != nil {
		t.Fatal(err)
	}
	before := anchorC1CASCount(a)
	observed, err := reader.ReadVerifiedSnapshot(context.Background(), ArticleDeletionNamespace)
	if err != nil || observed.Head.Sequence != 1 || len(observed.Prefixes) != 2 {
		t.Fatalf("ordinary ctx actual source projection: %v", err)
	}
	if before != anchorC1CASCount(a) {
		t.Fatal("reader repaired anchor")
	}
}

func testPublicationPrefixDrift(t *testing.T) {
	s := recovery.VerifiedJournalSnapshot{Namespace: ArticleDeletionNamespace, JournalID: "journal", Epoch: 1, Revision: "r1",
		Head: recovery.JournalHead{Sequence: 1, Hash: recovery.SHA256Digest([]byte("one"))}}
	s.Prefixes = []recovery.JournalHead{{}, s.Head}
	p := replayPublicationRecord{JournalID: s.JournalID, Epoch: s.Epoch, Head: s.Head}
	if err := publicationMatchesSnapshot(p, s, false); err != nil {
		t.Fatal(err)
	}
	s.Head = recovery.JournalHead{Sequence: 2, Hash: recovery.SHA256Digest([]byte("two"))}
	s.Prefixes = append(s.Prefixes, s.Head)
	s.Revision = "r2"
	if err := publicationMatchesSnapshot(p, s, false); err != nil {
		t.Fatal("legal historical prefix rejected")
	}
	if err := publicationMatchesSnapshot(p, s, true); err == nil {
		t.Fatal("historical prefix became exact current")
	}
	old := s
	old.Revision = "r1"
	if samePublicationSnapshot(old, s) {
		t.Fatal("revision drift ignored")
	}
}

type publicationVerifier struct{}

func (*publicationVerifier) VerifyLegacyPublication(context.Context, LegacyPublicationProof) error {
	return errors.New("constructor-only probe cannot approve a checkpoint")
}

func testPublicationMissingVerifier(t *testing.T) {
	for _, typed := range []bool{false, true} {
		t.Run(fmt.Sprint("TypedNil_", typed), func(t *testing.T) {
			store, d := publicationConstructorFixture()
			if typed {
				var p *publicationVerifier
				d.LegacySources = p
			}
			if _, err := NewReplayPublicationPorts(store, d); err != nil {
				t.Fatal("missing import-only dependency blocked constructor")
			}
			_, err := store.ImportLegacyPublication(context.Background(), recovery.PhasePermit{}, LegacyPublicationProof{RunID: "r", BackupRef: "b"})
			if !errors.Is(err, errReplayPublicationUnconfigured) {
				t.Fatal("missing verifier did not disable Import before ports")
			}
			if d.Runtime.(*publicationConstructorProbe).calls.Load() != 0 {
				t.Fatal("disabled Import invoked runtime")
			}
		})
	}
}

// Actual Runtime.Resume is held in its public replay consumer until explicit join.
type publicationReplayConsumer struct {
	*anchorReplayWorld
	calls   atomic.Int64
	entered chan context.Context
	release chan struct{}
	once    sync.Once
}

func (w *publicationReplayConsumer) Consume(ctx context.Context, run string) (recovery.ConsumptionResult, error) {
	w.calls.Add(1)
	if run != w.row.RunID {
		return recovery.ConsumptionResult{}, recovery.ErrAdmissionBinding
	}
	select {
	case w.entered <- ctx:
	case <-ctx.Done():
		return recovery.ConsumptionResult{}, ctx.Err()
	}
	select {
	case <-w.release:
	case <-ctx.Done():
	}
	return recovery.ConsumptionResult{}, w.stop
}

type publicationReplayPhase struct {
	ctx     context.Context
	runtime recovery.Runtime
	journal *anchoredArticleJournal
	world   *publicationReplayConsumer
	done    chan error
	workers sync.WaitGroup
}

func startPublicationReplay(t *testing.T, parent context.Context, run, backup string) *publicationReplayPhase {
	t.Helper()
	j, base, _, _ := actualAnchorCore(t, nil)
	lease, err := base.runtime.Acquire(parent, recovery.WorkRequest)
	if err != nil {
		t.Fatal(err)
	}
	for n := 1; n <= 3; n++ {
		if _, err := j.Append(lease.Context(), anchorRequest(n)); err != nil {
			lease.Release()
			t.Fatal(err)
		}
	}
	lease.Release()
	if err := base.runtime.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	w := &publicationReplayConsumer{anchorReplayWorld: newAnchorReplayWorld(base), entered: make(chan context.Context), release: make(chan struct{})}
	w.begin.RunID, w.begin.BackupRef = run, backup
	w.row.RunID, w.row.BackupRef = run, backup
	w.plan.RunID, w.plan.BackupRef = run, backup
	w.receipt.RunID, w.receipt.BackupRef = run, backup
	w.row.RestoreBinding = &w.receipt
	r, err := recovery.NewRuntime(parent, recovery.RuntimeDependencies{Store: w, Journal: w, ArticleSnapshots: w, Closures: w,
		Genesis: w, Exclusive: w, Lifecycle: w, Runs: w, Restore: w, Identity: w, Consumer: w, Exit: w, RecoverySecret: w})
	if err != nil {
		t.Fatal(err)
	}
	w.runtime = r
	j.options.Runtime = r
	p := &publicationReplayPhase{runtime: r, journal: j, world: w, done: make(chan error, 1)}
	p.workers.Add(1)
	go func() {
		defer p.workers.Done()
		p.done <- r.Resume(parent, recovery.RunBinding{RunID: run, BackupRef: backup})
	}()
	t.Cleanup(func() { p.close(t) })
	select {
	case p.ctx = <-w.entered:
	case err := <-p.done:
		p.workers.Wait()
		t.Fatalf("actual replay producer: %v", err)
	case <-parent.Done():
		p.close(t)
		t.Fatal(parent.Err())
	}
	return p
}

func (p *publicationReplayPhase) close(t *testing.T) {
	t.Helper()
	// This explicit component owner closes its idle journal before releasing the live action.
	if err := p.journal.Close(); err != nil {
		t.Error(err)
	}
	p.world.once.Do(func() { close(p.world.release) })
	p.workers.Wait()
	if err := p.runtime.Stop(context.Background()); err != nil {
		t.Error(err)
	}
}

type publicationGoldenCase struct {
	record            replayPublicationRecord
	canonical, digest string
}

func testPublicationGolden(t *testing.T) {
	for _, tc := range publicationGoldenRecords(t) {
		raw, err := canonicalReplayPublication(tc.record)
		if err != nil || string(raw) != tc.canonical || recovery.SHA256Digest(raw) != tc.digest {
			t.Fatalf("independent17 golden mismatch: %v", err)
		}
		if tc.record.Token != nil {
			token := "changed-synthetic-token"
			tc.record.Token = &token
			other, err := canonicalReplayPublication(tc.record)
			if err != nil || string(other) != tc.canonical {
				t.Fatal("token leaked into canonical")
			}
		}
	}
}
func publicationGoldenRecords(t *testing.T) []publicationGoldenCase {
	return []publicationGoldenCase{
		publicationGoldenAtomicUnicodeAndEscapes(t), publicationGoldenAtomicBigintBoundary(t), publicationGoldenLegacyImportCodecOnly(t),
	}
}
func publicationParseTime(t *testing.T, raw string) time.Time {
	t.Helper()
	v, e := time.Parse(replayPublicationTime, raw)
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func publicationGoldenAtomicUnicodeAndEscapes(t *testing.T) publicationGoldenCase {
	t.Helper()
	token := "synthetic-private-token"
	return publicationGoldenCase{record: replayPublicationRecord{
		Ref:            "11111111-1111-4111-8111-111111111111",
		Version:        "article-replay-publication/v1",
		SourceKind:     "atomic_current_v1",
		RunID:          "run-中文/\"quote\"\\<>&  /tail",
		Namespace:      "article-deletion/v1",
		BackupRef:      "backup-001",
		EvidenceRef:    "22222222-2222-4222-8222-222222222222",
		EvidenceDigest: "sha256:751f4f28d65cc87633a04261eabc56a424b0ced7e726dbd839e7a61a48a9ba8c",
		JournalID:      "article-journal-中文",
		SourceRef:      "11111111-1111-4111-8111-111111111111",
		SourceDigest:   "sha256:d27e9964ec869836970763495bf921cab83e7578d6d46548a009e5d3f4826413",
		Epoch:          1, Generation: 2, Token: &token,
		Head:                recovery.JournalHead{Sequence: 3, Hash: "sha256:245b7a19665ee2507a3faf403ea7121ed4a4a1b4b05f4b89af4e0865b450194c"},
		EvidenceCompletedAt: publicationParseTime(t, "2026-10-29T04:05:06.123456Z"), PublishedAt: publicationParseTime(t, "2026-10-29T04:05:07.654321Z"),
	}, canonical: "{\"publication_version\":\"article-replay-publication/v1\",\"publication_ref\":\"11111111-1111-4111-8111-111111111111\",\"source_kind\":\"atomic_current_v1\",\"run_id\":\"run-中文/\\\"quote\\\"\\\\<>&\\u2028\\u2029/tail\",\"backup_ref\":\"backup-001\",\"namespace\":\"article-deletion/v1\",\"evidence_ref\":\"22222222-2222-4222-8222-222222222222\",\"evidence_digest\":\"sha256:751f4f28d65cc87633a04261eabc56a424b0ced7e726dbd839e7a61a48a9ba8c\",\"evidence_completed_at\":\"2026-10-29T04:05:06.123456Z\",\"journal_id\":\"article-journal-中文\",\"journal_epoch\":1,\"journal_head_sequence\":3,\"journal_head_hash\":\"sha256:245b7a19665ee2507a3faf403ea7121ed4a4a1b4b05f4b89af4e0865b450194c\",\"attempt_generation\":2,\"source_ref\":\"11111111-1111-4111-8111-111111111111\",\"source_digest\":\"sha256:d27e9964ec869836970763495bf921cab83e7578d6d46548a009e5d3f4826413\",\"published_at\":\"2026-10-29T04:05:07.654321Z\"}", digest: "sha256:b1b044bab36db24b9f0befb24a792cb8cb6b3067a5e64d90cf3dbccae53b9323"}
}
func publicationGoldenAtomicBigintBoundary(t *testing.T) publicationGoldenCase {
	t.Helper()
	token := "synthetic-private-token"
	return publicationGoldenCase{record: replayPublicationRecord{
		Ref:            "33333333-3333-4333-8333-333333333333",
		Version:        "article-replay-publication/v1",
		SourceKind:     "atomic_current_v1",
		RunID:          "run-boundary",
		Namespace:      "article-deletion/v1",
		BackupRef:      "backup-001",
		EvidenceRef:    "22222222-2222-4222-8222-222222222222",
		EvidenceDigest: "sha256:751f4f28d65cc87633a04261eabc56a424b0ced7e726dbd839e7a61a48a9ba8c",
		JournalID:      "journal-boundary",
		SourceRef:      "33333333-3333-4333-8333-333333333333",
		SourceDigest:   "sha256:ef9a21ab7f17f08cc86cf39d119daefc47d747abd818c5beb4b99a41fa9e13e0",
		Epoch:          9223372036854775807, Generation: 9223372036854775807, Token: &token,
		Head:                recovery.JournalHead{Sequence: 9223372036854775807, Hash: "sha256:245b7a19665ee2507a3faf403ea7121ed4a4a1b4b05f4b89af4e0865b450194c"},
		EvidenceCompletedAt: publicationParseTime(t, "2026-10-29T04:05:06.123456Z"), PublishedAt: publicationParseTime(t, "2026-10-29T04:05:07.654321Z"),
	}, canonical: "{\"publication_version\":\"article-replay-publication/v1\",\"publication_ref\":\"33333333-3333-4333-8333-333333333333\",\"source_kind\":\"atomic_current_v1\",\"run_id\":\"run-boundary\",\"backup_ref\":\"backup-001\",\"namespace\":\"article-deletion/v1\",\"evidence_ref\":\"22222222-2222-4222-8222-222222222222\",\"evidence_digest\":\"sha256:751f4f28d65cc87633a04261eabc56a424b0ced7e726dbd839e7a61a48a9ba8c\",\"evidence_completed_at\":\"2026-10-29T04:05:06.123456Z\",\"journal_id\":\"journal-boundary\",\"journal_epoch\":9223372036854775807,\"journal_head_sequence\":9223372036854775807,\"journal_head_hash\":\"sha256:245b7a19665ee2507a3faf403ea7121ed4a4a1b4b05f4b89af4e0865b450194c\",\"attempt_generation\":9223372036854775807,\"source_ref\":\"33333333-3333-4333-8333-333333333333\",\"source_digest\":\"sha256:ef9a21ab7f17f08cc86cf39d119daefc47d747abd818c5beb4b99a41fa9e13e0\",\"published_at\":\"2026-10-29T04:05:07.654321Z\"}", digest: "sha256:cb861bca22dbc8602ea8368f4b242253ad8fe8f5e124f7b9f407a4dd772acb49"}
}
func publicationGoldenLegacyImportCodecOnly(t *testing.T) publicationGoldenCase {
	t.Helper()
	return publicationGoldenCase{record: replayPublicationRecord{
		Ref:            "44444444-4444-4444-8444-444444444444",
		Version:        "article-replay-publication/v1",
		SourceKind:     "verified_legacy_import_v1",
		RunID:          "legacy-run",
		Namespace:      "article-deletion/v1",
		BackupRef:      "backup-001",
		EvidenceRef:    "22222222-2222-4222-8222-222222222222",
		EvidenceDigest: "sha256:751f4f28d65cc87633a04261eabc56a424b0ced7e726dbd839e7a61a48a9ba8c",
		JournalID:      "article-journal-中文",
		SourceRef:      "approved-checkpoint/中文/\"old\"\\<>&",
		SourceDigest:   "sha256:b45d7b4757e43d06b6fa17ac99b9df54bf9b5c872bffe40de4870d60b93806fc",
		Epoch:          1, Generation: 0, Token: nil,
		Head:                recovery.JournalHead{Sequence: 3, Hash: "sha256:245b7a19665ee2507a3faf403ea7121ed4a4a1b4b05f4b89af4e0865b450194c"},
		EvidenceCompletedAt: publicationParseTime(t, "2026-10-29T04:05:06.123456Z"), PublishedAt: publicationParseTime(t, "2026-10-29T04:05:07.654321Z"),
	}, canonical: "{\"publication_version\":\"article-replay-publication/v1\",\"publication_ref\":\"44444444-4444-4444-8444-444444444444\",\"source_kind\":\"verified_legacy_import_v1\",\"run_id\":\"legacy-run\",\"backup_ref\":\"backup-001\",\"namespace\":\"article-deletion/v1\",\"evidence_ref\":\"22222222-2222-4222-8222-222222222222\",\"evidence_digest\":\"sha256:751f4f28d65cc87633a04261eabc56a424b0ced7e726dbd839e7a61a48a9ba8c\",\"evidence_completed_at\":\"2026-10-29T04:05:06.123456Z\",\"journal_id\":\"article-journal-中文\",\"journal_epoch\":1,\"journal_head_sequence\":3,\"journal_head_hash\":\"sha256:245b7a19665ee2507a3faf403ea7121ed4a4a1b4b05f4b89af4e0865b450194c\",\"attempt_generation\":0,\"source_ref\":\"approved-checkpoint/中文/\\\"old\\\"\\\\<>&\",\"source_digest\":\"sha256:b45d7b4757e43d06b6fa17ac99b9df54bf9b5c872bffe40de4870d60b93806fc\",\"published_at\":\"2026-10-29T04:05:07.654321Z\"}", digest: "sha256:199869c7eee9f415371927968e8a89dfcb52a7cfe4c8d12c62881b7602d1e0e0"}
}
