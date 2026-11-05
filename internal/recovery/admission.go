package recovery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	admissionVersion        = "han-recovery-admission/v1"
	admissionReferenceLimit = 1024
	admissionProofAttempts  = 2
)

// InstallationVersion is the explicit installation commitment format.
const InstallationVersion = "han-recovery-install/v1"

var journalHMACDigestPattern = regexp.MustCompile(`^hmac-sha256:[0-9a-f]{64}$`)

var (
	ErrAdmissionClosed      = errors.New("recovery admission closed")
	ErrAdmissionIncomplete  = errors.New("recovery admission incomplete")
	ErrAdmissionDrift       = errors.New("recovery admission drift")
	ErrAdmissionBinding     = errors.New("recovery admission binding mismatch")
	ErrAdmissionUnsupported = errors.New("recovery admission unsupported")
	ErrAdmissionBusy        = errors.New("recovery admission busy")
	ErrRecoveryWriteUnknown = errors.New("recovery write result unknown")
)

// InstallationBinding is the immutable database installation projection.
type InstallationBinding struct {
	Version, DeploymentID, InstallationID         string
	TargetDigest, CommitmentRef, CommitmentDigest string
	RecoveryInitial, ArticleInitial               AnchorSnapshot
	InstalledAt                                   time.Time
}

// StoreRevision detects changes to the entire installation and run set.
type StoreRevision struct {
	InstallationID string
	Counter        int64
	Digest         string
}

// RestoreReceipt preserves the independently verified database restore source.
type RestoreReceipt struct {
	Version, RunID, BackupRef, DeploymentID, InstallationID, TargetDigest     string
	ArtifactDigest, SourceApprovalDigest, RestoredSchemaDigest, ReceiptDigest string
	ArtifactBytes                                                             int64
	RestoredAt                                                                time.Time
}

// AdmissionRun preserves every run field, including SQL NULL distinctions.
type AdmissionRun struct {
	RunID, BackupRef, JournalKeyID, JournalKeyVersion, JournalSecretFingerprint         string
	JournalStage                                                                        Stage
	StartedAt                                                                           time.Time
	JournalConsumedAt, ArticleEvidenceCompletedAt, ArticleReplayVerifiedAt, CompletedAt *time.Time
	ArticleEvidenceVersion, ArticleEvidenceDigest, ArticleEvidenceRef, ArticleNamespace *string
	ArticlePublicationRef, ArticlePublicationDigest, AuditEventID                       *string
	Outcome                                                                             RecoveryOutcome
	AccountCount, PermanentCredentialCount, SessionCount, TemporaryCredentialCount      int64
	IdempotencyCount, PendingReviewCount                                                int64
	RestoreBinding                                                                      *RestoreReceipt
	InitialAdminUserID                                                                  *string
	InitialAdminBootstrapGeneration, InitialAdminChangedGeneration                      *int64
	InitialAdminChangeAuditID                                                           *string
	InitialAdminChangedAt                                                               *time.Time
}

// AdmissionSnapshot contains the complete deployment run set, never a latest row.
type AdmissionSnapshot struct {
	Installation InstallationBinding
	Revision     StoreRevision
	Runs         []AdmissionRun
}

// AdmissionStore performs complete short, read-only snapshot transactions.
type AdmissionStore interface {
	ReadAdmissionSnapshot(context.Context) (AdmissionSnapshot, error)
	ReadAdmissionRevision(context.Context) (StoreRevision, error)
}

// GenesisProof binds an approved installation to both initial journal heads.
type GenesisProof struct {
	Binding                              InstallationBinding
	RecoveryInitial, ArticleInitial      AnchorSnapshot
	CommitmentRevision, CommitmentDigest string
}

// InstallationReader verifies an independently committed genesis, without writes.
type InstallationReader interface {
	VerifyGenesis(context.Context, InstallationBinding) (GenesisProof, error)
}

// CanonicalAdmissionSnapshot encodes the fixed complete Store projection.
func CanonicalAdmissionSnapshot(s AdmissionSnapshot) ([]byte, error) {
	if err := validateAdmissionSnapshot(s); err != nil {
		return nil, err
	}
	runs := append([]AdmissionRun{}, s.Runs...)
	sort.Slice(runs, func(i, j int) bool { return runs[i].RunID < runs[j].RunID })
	encoded := make([]json.RawMessage, 0, len(runs))
	for _, r := range runs {
		encoded = append(encoded, encodeAdmissionRun(r))
	}
	return encodeAdmissionObject([]admissionField{
		{"version", admissionVersion}, {"installation", encodeInstallation(s.Installation, s.Revision.Counter)}, {"runs", encoded},
	}), nil
}

type admissionField struct {
	name  string
	value any
}

func encodeAdmissionObject(fields []admissionField) json.RawMessage {
	var out bytes.Buffer
	out.WriteByte('{')
	for i, f := range fields {
		if i > 0 {
			out.WriteByte(',')
		}
		out.Write(encodeAdmissionValue(f.name))
		out.WriteByte(':')
		out.Write(encodeAdmissionValue(f.value))
	}
	out.WriteByte('}')
	return out.Bytes()
}

func encodeAdmissionValue(value any) []byte {
	switch v := value.(type) {
	case time.Time:
		value = v.Format("2006-01-02T15:04:05.000000Z")
	case *time.Time:
		if v != nil {
			value = v.Format("2006-01-02T15:04:05.000000Z")
		}
	}
	var b bytes.Buffer
	e := json.NewEncoder(&b)
	e.SetEscapeHTML(false)
	// All callers supply validated primitives, typed slices or internally encoded JSON.
	if err := e.Encode(value); err != nil {
		panic("invalid internal admission encoding")
	}
	return bytes.TrimSuffix(b.Bytes(), []byte("\n"))
}

func encodeInstallation(i InstallationBinding, counter int64) json.RawMessage {
	r, a := i.RecoveryInitial, i.ArticleInitial
	return encodeAdmissionObject([]admissionField{
		{"metadata_version", i.Version}, {"deployment_id", i.DeploymentID}, {"installation_id", i.InstallationID},
		{"target_digest", i.TargetDigest}, {"commitment_ref", i.CommitmentRef}, {"commitment_digest", i.CommitmentDigest},
		{"recovery_journal_id", r.JournalID}, {"recovery_epoch", r.Epoch}, {"recovery_initial_sequence", r.Committed.Sequence}, {"recovery_initial_hash", r.Committed.Hash},
		{"article_journal_id", a.JournalID}, {"article_epoch", a.Epoch}, {"article_initial_sequence", a.Committed.Sequence}, {"article_initial_hash", a.Committed.Hash},
		{"revision", counter}, {"installed_at", i.InstalledAt},
	})
}

func encodeAdmissionRun(r AdmissionRun) json.RawMessage {
	return encodeAdmissionObject([]admissionField{
		{"run_id", r.RunID}, {"backup_ref", r.BackupRef}, {"journal_key_id", r.JournalKeyID}, {"journal_key_version", r.JournalKeyVersion},
		{"journal_secret_fingerprint", r.JournalSecretFingerprint}, {"journal_stage", r.JournalStage}, {"started_at", r.StartedAt}, {"journal_consumed_at", r.JournalConsumedAt},
		{"article_evidence_version", r.ArticleEvidenceVersion}, {"article_evidence_digest", r.ArticleEvidenceDigest}, {"article_evidence_ref", r.ArticleEvidenceRef},
		{"article_evidence_completed_at", r.ArticleEvidenceCompletedAt}, {"article_namespace", r.ArticleNamespace}, {"article_replay_verified_at", r.ArticleReplayVerifiedAt},
		{"article_publication_ref", r.ArticlePublicationRef}, {"article_publication_digest", r.ArticlePublicationDigest}, {"completed_at", r.CompletedAt}, {"outcome", r.Outcome},
		{"account_count", r.AccountCount}, {"permanent_credential_count", r.PermanentCredentialCount}, {"session_count", r.SessionCount}, {"temporary_credential_count", r.TemporaryCredentialCount},
		{"idempotency_count", r.IdempotencyCount}, {"pending_review_count", r.PendingReviewCount}, {"audit_event_id", r.AuditEventID}, {"restore_binding", encodeRestoreReceipt(r.RestoreBinding)},
		{"initial_admin_user_id", r.InitialAdminUserID}, {"initial_admin_bootstrap_generation", r.InitialAdminBootstrapGeneration},
		{"initial_admin_changed_generation", r.InitialAdminChangedGeneration}, {"initial_admin_change_audit_id", r.InitialAdminChangeAuditID}, {"initial_admin_changed_at", r.InitialAdminChangedAt},
	})
}

func encodeRestoreReceipt(r *RestoreReceipt) json.RawMessage {
	if r == nil {
		return json.RawMessage("null")
	}
	return encodeAdmissionObject([]admissionField{
		{"receipt_version", r.Version}, {"deployment_id", r.DeploymentID}, {"installation_id", r.InstallationID}, {"backup_ref", r.BackupRef},
		{"artifact_digest", r.ArtifactDigest}, {"artifact_bytes", r.ArtifactBytes}, {"target_digest", r.TargetDigest}, {"source_approval_digest", r.SourceApprovalDigest},
		{"restored_schema_digest", r.RestoredSchemaDigest}, {"receipt_digest", r.ReceiptDigest}, {"restored_at", r.RestoredAt},
	})
}

func admissionText(s string) bool {
	return s != "" && len(s) <= admissionReferenceLimit && utf8.ValidString(s) && strings.TrimSpace(s) == s && !strings.ContainsFunc(s, unicode.IsControl)
}

func admissionTime(t time.Time) bool { return !t.IsZero() && isCanonicalTime(t) }
func admissionHead(h JournalHead) bool {
	if h.Sequence == 0 {
		return h.Hash == ""
	}
	return h.Sequence <= math.MaxInt64 && sha256DigestPattern.MatchString(h.Hash)
}

func validateInitial(s AnchorSnapshot, namespace string) bool {
	return admissionText(s.JournalID) && s.Namespace == namespace && s.Epoch > 0 && s.Epoch <= math.MaxInt64 && s.Pending == nil && admissionHead(s.Committed)
}

func validateAdmissionSnapshot(s AdmissionSnapshot) error {
	i := s.Installation
	if i.Version != InstallationVersion || !admissionText(i.DeploymentID) || !admissionText(i.InstallationID) ||
		!admissionText(i.CommitmentRef) || !sha256DigestPattern.MatchString(i.TargetDigest) || !sha256DigestPattern.MatchString(i.CommitmentDigest) ||
		!admissionTime(i.InstalledAt) || !validateInitial(i.RecoveryInitial, RecoveryNamespaceV1) || !validateInitial(i.ArticleInitial, ArticleDeletionNamespaceV1) {
		return ErrAdmissionBinding
	}
	if s.Revision.InstallationID != i.InstallationID || s.Revision.Counter <= 0 || s.Runs == nil {
		return ErrAdmissionIncomplete
	}
	seen := make(map[string]bool, len(s.Runs))
	for _, r := range s.Runs {
		if seen[r.RunID] {
			return ErrAdmissionIncomplete
		}
		seen[r.RunID] = true
		if err := validateAdmissionRun(r, i); err != nil {
			return err
		}
	}
	return nil
}

func validateAdmissionRun(r AdmissionRun, i InstallationBinding) error {
	for _, s := range []string{r.RunID, r.BackupRef, r.JournalKeyID, r.JournalKeyVersion} {
		if !admissionText(s) {
			return ErrAdmissionBinding
		}
	}
	if !journalHMACDigestPattern.MatchString(r.JournalSecretFingerprint) || !admissionTime(r.StartedAt) {
		return ErrAdmissionBinding
	}
	if r.JournalStage != StageBegun && r.JournalStage != StageConsumed {
		return ErrAdmissionIncomplete
	}
	switch r.Outcome {
	case RecoveryOutcomeRunning, RecoveryOutcomeFailed, RecoveryOutcomeIdentityReady, RecoveryOutcomeArticleDeletionReady, RecoveryOutcomeReadyToConsume, RecoveryOutcomeJournalConsumed, RecoveryOutcomeSucceeded:
	default:
		return ErrAdmissionIncomplete
	}
	for _, n := range []int64{r.AccountCount, r.PermanentCredentialCount, r.SessionCount, r.TemporaryCredentialCount, r.IdempotencyCount, r.PendingReviewCount} {
		if n < 0 {
			return ErrAdmissionIncomplete
		}
	}
	for _, t := range []*time.Time{r.JournalConsumedAt, r.ArticleEvidenceCompletedAt, r.ArticleReplayVerifiedAt, r.CompletedAt, r.InitialAdminChangedAt} {
		if t != nil && !admissionTime(*t) {
			return ErrAdmissionIncomplete
		}
	}
	for _, s := range []*string{r.ArticleEvidenceVersion, r.ArticleEvidenceRef, r.ArticleNamespace, r.ArticlePublicationRef} {
		if s != nil && !admissionText(*s) {
			return ErrAdmissionBinding
		}
	}
	for _, s := range []*string{r.ArticleEvidenceDigest, r.ArticlePublicationDigest} {
		if s != nil && !sha256DigestPattern.MatchString(*s) {
			return ErrAdmissionBinding
		}
	}
	for _, s := range []*string{r.AuditEventID, r.InitialAdminUserID, r.InitialAdminChangeAuditID} {
		if s != nil && !admissionUUID(*s) {
			return ErrAdmissionBinding
		}
	}
	if err := validateInitialAdminProof(r); err != nil {
		return err
	}
	if r.RestoreBinding != nil {
		return validateRestoreBinding(*r.RestoreBinding, r, i)
	}
	return nil
}

func admissionUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for n, c := range s {
		if n == 8 || n == 13 || n == 18 || n == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return s != "00000000-0000-0000-0000-000000000000"
}

func validateInitialAdminProof(r AdmissionRun) error {
	if (r.InitialAdminUserID == nil) != (r.InitialAdminBootstrapGeneration == nil) {
		return ErrAdmissionIncomplete
	}
	if r.InitialAdminBootstrapGeneration != nil && (*r.InitialAdminBootstrapGeneration <= 0 || *r.InitialAdminBootstrapGeneration == math.MaxInt64) {
		return ErrAdmissionIncomplete
	}
	changed := r.InitialAdminChangedGeneration != nil
	if changed != (r.InitialAdminChangeAuditID != nil) || changed != (r.InitialAdminChangedAt != nil) {
		return ErrAdmissionIncomplete
	}
	if changed && (r.InitialAdminBootstrapGeneration == nil || *r.InitialAdminChangedGeneration != *r.InitialAdminBootstrapGeneration+1) {
		return ErrAdmissionIncomplete
	}
	return nil
}

func validateRestoreBinding(x RestoreReceipt, r AdmissionRun, i InstallationBinding) error {
	if x.Version != "han-db-restore/v1" || x.RunID != r.RunID || x.BackupRef != r.BackupRef || x.DeploymentID != i.DeploymentID ||
		x.InstallationID != i.InstallationID || x.TargetDigest != i.TargetDigest || x.ArtifactBytes <= 0 || !admissionTime(x.RestoredAt) {
		return ErrAdmissionBinding
	}
	for _, s := range []string{x.ArtifactDigest, x.SourceApprovalDigest, x.RestoredSchemaDigest, x.ReceiptDigest} {
		if !sha256DigestPattern.MatchString(s) {
			return ErrAdmissionBinding
		}
	}
	return nil
}

type admissionProof struct {
	installation      InstallationBinding
	store             StoreRevision
	recovery, article VerifiedJournalSnapshot
	last              RunBinding
}

func checkedContext(ctx context.Context) error {
	if ctx == nil {
		return ErrAdmissionBinding
	}
	return ctx.Err()
}

func readProof(ctx context.Context, d RuntimeDependencies, lease ExclusiveLease) (admissionProof, error) {
	var last error
	for attempt := 0; attempt < admissionProofAttempts; attempt++ {
		p, err := readProofAttempt(ctx, d, lease)
		if err == nil {
			return p, nil
		}
		last = err
		if !errors.Is(err, ErrAdmissionDrift) {
			break
		}
	}
	return admissionProof{}, last
}

func readProofAttempt(ctx context.Context, d RuntimeDependencies, lease ExclusiveLease) (admissionProof, error) {
	if err := checkExclusive(ctx, lease); err != nil {
		return admissionProof{}, err
	}
	r, err := d.Journal.ReadVerifiedSnapshot(ctx, RecoveryNamespaceV1)
	if err != nil {
		return admissionProof{}, admissionReadError(err)
	}
	a, err := d.ArticleSnapshots.ReadVerifiedSnapshot(ctx, ArticleDeletionNamespaceV1)
	if err != nil {
		return admissionProof{}, admissionReadError(err)
	}
	s, err := d.Store.ReadAdmissionSnapshot(ctx)
	if err != nil {
		return admissionProof{}, admissionReadError(err)
	}
	encoded, err := CanonicalAdmissionSnapshot(s)
	if err != nil {
		return admissionProof{}, err
	}
	if !sha256DigestPattern.MatchString(s.Revision.Digest) || s.Revision.Digest != SHA256Digest(encoded) {
		return admissionProof{}, ErrAdmissionDrift
	}
	p := admissionProof{installation: s.Installation, store: s.Revision, recovery: r, article: a}
	if err := validateProofJournals(p); err != nil {
		return admissionProof{}, err
	}
	if err := proveHistory(ctx, d, s, &p); err != nil {
		return admissionProof{}, err
	}
	if err := recheckProof(ctx, d, lease, p); err != nil {
		return admissionProof{}, err
	}
	return p, nil
}

func admissionReadError(err error) error { return fmt.Errorf("%w: %w", ErrAdmissionIncomplete, err) }
func checkExclusive(ctx context.Context, l ExclusiveLease) error {
	if err := checkedContext(ctx); err != nil {
		return err
	}
	if missingRuntimePort(l) {
		return ErrAdmissionBinding
	}
	if err := l.Check(ctx); err != nil {
		return fmt.Errorf("%w: %w", ErrAdmissionClosed, err)
	}
	return checkedContext(ctx)
}

func validateProofJournals(p admissionProof) error {
	if err := validateVerifiedJournal(p.recovery, p.installation.RecoveryInitial); err != nil {
		return err
	}
	return validateVerifiedJournal(p.article, p.installation.ArticleInitial)
}

func hasPrefix(s VerifiedJournalSnapshot, h JournalHead) bool {
	for _, p := range s.Prefixes {
		if p == h {
			return true
		}
	}
	return false
}

func validateVerifiedJournal(s VerifiedJournalSnapshot, initial AnchorSnapshot) error {
	if s.JournalID != initial.JournalID || s.Namespace != initial.Namespace || s.Epoch != initial.Epoch {
		return ErrAdmissionBinding
	}
	if s.Pending != nil || !admissionText(s.Revision) || !admissionHead(s.Head) || !hasPrefix(s, initial.Committed) || !hasPrefix(s, s.Head) {
		return ErrAdmissionIncomplete
	}
	for n, h := range s.Prefixes {
		if !admissionHead(h) || n > 0 && h.Sequence != s.Prefixes[n-1].Sequence+1 {
			return ErrAdmissionIncomplete
		}
	}
	if s.Namespace == RecoveryNamespaceV1 {
		return validateRecoveryEntries(s)
	}
	return nil
}

func validateRecoveryEntries(s VerifiedJournalSnapshot) error {
	if uint64(len(s.Entries)) != s.Head.Sequence {
		return ErrAdmissionIncomplete
	}
	for n, e := range s.Entries {
		if e.Namespace != RecoveryNamespaceV1 || e.Sequence != uint64(n+1) || !admissionTime(e.RecordedAt) ||
			!admissionText(e.RunID) || !admissionText(e.BackupRef) || !admissionText(e.KeyID) || !admissionText(e.KeyVersion) ||
			!journalHMACDigestPattern.MatchString(e.SecretFingerprint) || !journalHMACDigestPattern.MatchString(e.RecordHMAC) {
			return ErrAdmissionIncomplete
		}
		if n == 0 && e.PreviousRecordHash != "" {
			return ErrAdmissionBinding
		}
		if n > 0 && !hasPrefix(s, JournalHead{Sequence: uint64(n), Hash: e.PreviousRecordHash}) {
			return ErrAdmissionBinding
		}
	}
	return nil
}

func proveHistory(ctx context.Context, d RuntimeDependencies, s AdmissionSnapshot, p *admissionProof) error {
	if len(s.Runs) == 0 {
		if len(p.recovery.Entries) != 0 {
			return ErrAdmissionIncomplete
		}
		return proveGenesis(ctx, d.Genesis, *p)
	}
	runs := make(map[string]AdmissionRun, len(s.Runs))
	for _, r := range s.Runs {
		if !completeAdmissionRun(r) {
			return ErrAdmissionIncomplete
		}
		runs[r.RunID] = r
	}
	begun := make(map[string]JournalEntry)
	consumed := make(map[string]bool)
	for _, e := range p.recovery.Entries {
		r, ok := runs[e.RunID]
		if !ok {
			return ErrAdmissionIncomplete
		}
		if err := matchHistoryEntry(e, r, begun, consumed); err != nil {
			return err
		}
		if e.Stage == StageBegun {
			p.last = RunBinding{RunID: e.RunID, BackupRef: e.BackupRef}
		}
	}
	if len(consumed) != len(runs) {
		return ErrAdmissionIncomplete
	}
	for _, r := range s.Runs {
		if err := proveClosure(ctx, d.Closures, r, p.article); err != nil {
			return err
		}
	}
	return checkedContext(ctx)
}

func completeAdmissionRun(r AdmissionRun) bool {
	return r.Outcome == RecoveryOutcomeSucceeded && r.JournalStage == StageConsumed && r.PendingReviewCount == 0 &&
		r.JournalConsumedAt != nil && r.ArticleEvidenceVersion != nil && *r.ArticleEvidenceVersion == ReplayEvidenceVersionV1 &&
		r.ArticleEvidenceDigest != nil && r.ArticleEvidenceRef != nil && r.ArticleEvidenceCompletedAt != nil &&
		r.ArticleNamespace != nil && *r.ArticleNamespace == ArticleDeletionNamespaceV1 && r.ArticleReplayVerifiedAt != nil &&
		r.ArticlePublicationRef != nil && r.ArticlePublicationDigest != nil && r.CompletedAt != nil && r.AuditEventID != nil &&
		r.RestoreBinding != nil && r.InitialAdminUserID != nil && r.InitialAdminBootstrapGeneration != nil &&
		r.InitialAdminChangedGeneration != nil && r.InitialAdminChangeAuditID != nil && r.InitialAdminChangedAt != nil
}

func matchHistoryEntry(e JournalEntry, r AdmissionRun, begun map[string]JournalEntry, consumed map[string]bool) error {
	if e.BackupRef != r.BackupRef || e.KeyID != r.JournalKeyID || e.KeyVersion != r.JournalKeyVersion || e.SecretFingerprint != r.JournalSecretFingerprint {
		return ErrAdmissionBinding
	}
	switch e.Stage {
	case StageBegun:
		if _, ok := begun[e.RunID]; ok || consumed[e.RunID] || !e.RecordedAt.Equal(r.StartedAt) {
			return ErrAdmissionBinding
		}
		begun[e.RunID] = e
	case StageConsumed:
		b, ok := begun[e.RunID]
		if !ok || consumed[e.RunID] || !e.RecordedAt.Equal(*r.JournalConsumedAt) || e.RecordedAt.Before(b.RecordedAt) {
			return ErrAdmissionBinding
		}
		consumed[e.RunID] = true
	default:
		return ErrAdmissionIncomplete
	}
	return nil
}

func proveGenesis(ctx context.Context, reader InstallationReader, p admissionProof) error {
	g, err := reader.VerifyGenesis(ctx, p.installation)
	if err != nil {
		return admissionReadError(err)
	}
	if g.Binding != p.installation || g.RecoveryInitial != p.installation.RecoveryInitial || g.ArticleInitial != p.installation.ArticleInitial ||
		!admissionText(g.CommitmentRevision) || g.CommitmentDigest != p.installation.CommitmentDigest {
		return ErrAdmissionBinding
	}
	return checkedContext(ctx)
}

func proveClosure(ctx context.Context, reader PublishedClosureReader, r AdmissionRun, a VerifiedJournalSnapshot) error {
	q := ClosureRequest{RunID: r.RunID, BackupRef: r.BackupRef, EvidenceRef: *r.ArticleEvidenceRef, EvidenceDigest: *r.ArticleEvidenceDigest, PublicationRef: *r.ArticlePublicationRef, PublicationDigest: *r.ArticlePublicationDigest}
	c, err := reader.ReadPublishedClosure(ctx, q)
	if err != nil {
		return admissionReadError(err)
	}
	if c.VerifiedArticleRevision != a.Revision || c.CurrentHead != a.Head {
		return ErrAdmissionDrift
	}
	if c.PublicationRef != q.PublicationRef || c.PublicationDigest != q.PublicationDigest || c.JournalID != a.JournalID || c.Epoch != a.Epoch ||
		!hasPrefix(a, c.HistoricalHead) || !admissionTime(c.PublishedAt) {
		return ErrAdmissionBinding
	}
	v := c.Ready
	if v.RunID != q.RunID || v.BackupRef != q.BackupRef || v.EvidenceRef != q.EvidenceRef || v.EvidenceDigest != q.EvidenceDigest ||
		v.JournalHeadSequence != c.HistoricalHead.Sequence || v.JournalHeadHash != c.HistoricalHead.Hash {
		return ErrAdmissionBinding
	}
	if err := validateReady(v); err != nil {
		return errors.Join(ErrAdmissionBinding, err)
	}
	digest, err := v.CanonicalDigest()
	if err != nil || digest != v.EvidenceDigest {
		return errors.Join(ErrAdmissionBinding, err)
	}
	completed, err := time.Parse(time.RFC3339Nano, v.CompletedAt)
	if err != nil || !admissionTime(completed) || !completed.Equal(*r.ArticleEvidenceCompletedAt) || c.PublishedAt.Before(completed) {
		return ErrAdmissionBinding
	}
	return checkedContext(ctx)
}

func sameJournalObservation(a, b VerifiedJournalSnapshot) bool {
	return a.JournalID == b.JournalID && a.Namespace == b.Namespace && a.Epoch == b.Epoch && a.Head == b.Head && a.Revision == b.Revision && a.Pending == nil && b.Pending == nil
}

func recheckProof(ctx context.Context, d RuntimeDependencies, lease ExclusiveLease, p admissionProof) error {
	r, err := d.Store.ReadAdmissionRevision(ctx)
	if err != nil {
		return admissionReadError(err)
	}
	if r != p.store {
		return ErrAdmissionDrift
	}
	recovery, err := d.Journal.ReadVerifiedSnapshot(ctx, RecoveryNamespaceV1)
	if err != nil {
		return admissionReadError(err)
	}
	article, err := d.ArticleSnapshots.ReadVerifiedSnapshot(ctx, ArticleDeletionNamespaceV1)
	if err != nil {
		return admissionReadError(err)
	}
	if !sameJournalObservation(recovery, p.recovery) || !sameJournalObservation(article, p.article) {
		return ErrAdmissionDrift
	}
	return checkExclusive(ctx, lease)
}

type controlledView struct {
	snapshot      AdmissionSnapshot
	proof         admissionProof
	run           *AdmissionRun
	begin, latest JournalEntry
}

func (r *admissionRuntime) readControlled(ctx context.Context, b RunBinding) (controlledView, error) {
	if err := r.checkOwnership(ctx); err != nil {
		return controlledView{}, err
	}
	j, e := r.deps.Journal.ReadVerifiedSnapshot(ctx, RecoveryNamespaceV1)
	if e != nil {
		return controlledView{}, admissionReadError(e)
	}
	j = copyControlledJournal(j)
	a, e := r.deps.ArticleSnapshots.ReadVerifiedSnapshot(ctx, ArticleDeletionNamespaceV1)
	if e != nil {
		return controlledView{}, admissionReadError(e)
	}
	a = copyControlledJournal(a)
	s, e := r.deps.Store.ReadAdmissionSnapshot(ctx)
	if e != nil {
		return controlledView{}, admissionReadError(e)
	}
	raw, e := CanonicalAdmissionSnapshot(s)
	if e != nil {
		return controlledView{}, e
	}
	if SHA256Digest(raw) != s.Revision.Digest {
		return controlledView{}, ErrAdmissionDrift
	}
	v := controlledView{snapshot: s, proof: admissionProof{installation: s.Installation, store: s.Revision, recovery: j, article: a, last: b}}
	if e := validateProofJournals(v.proof); e != nil {
		return controlledView{}, e
	}
	if e := r.proveControlledHistory(ctx, b, &v); e != nil {
		return controlledView{}, e
	}
	if e := recheckProof(ctx, r.deps, r.exclusive, v.proof); e != nil {
		return controlledView{}, e
	}
	return v, nil
}

func (r *admissionRuntime) proveControlledHistory(ctx context.Context, b RunBinding, v *controlledView) error {
	others := make(map[string]AdmissionRun)
	begun := make(map[string]JournalEntry)
	consumed := make(map[string]bool)
	for i, x := range v.snapshot.Runs {
		if x.RunID == b.RunID {
			if x.BackupRef != b.BackupRef {
				return ErrAdmissionBinding
			}
			v.run = &v.snapshot.Runs[i]
			continue
		}
		if !completeAdmissionRun(x) {
			return ErrAdmissionIncomplete
		}
		others[x.RunID] = x
	}
	for _, e := range v.proof.recovery.Entries {
		if e.RunID == b.RunID {
			if err := v.acceptCurrentEntry(e, b); err != nil {
				return err
			}
			continue
		}
		x, ok := others[e.RunID]
		if !ok {
			return ErrAdmissionIncomplete
		}
		if err := matchHistoryEntry(e, x, begun, consumed); err != nil {
			return err
		}
	}
	if len(others) != len(consumed) || v.begin.RunID == "" {
		return ErrAdmissionIncomplete
	}
	for _, x := range others {
		if err := proveClosure(ctx, r.deps.Closures, x, v.proof.article); err != nil {
			return err
		}
	}
	if v.run != nil {
		return r.validateCurrentRun(ctx, *v)
	}
	if v.latest.Stage != StageBegun {
		return ErrAdmissionIncomplete
	}
	return nil
}

func (v *controlledView) acceptCurrentEntry(e JournalEntry, b RunBinding) error {
	if e.BackupRef != b.BackupRef {
		return ErrAdmissionBinding
	}
	if e.Stage == StageBegun {
		if v.begin.RunID != "" {
			return ErrAdmissionBinding
		}
		v.begin = e
	} else if e.Stage == StageConsumed {
		if v.begin.RunID == "" || v.latest.Stage == StageConsumed || e.KeyID != v.begin.KeyID || e.KeyVersion != v.begin.KeyVersion || e.SecretFingerprint != v.begin.SecretFingerprint {
			return ErrAdmissionBinding
		}
	} else {
		return ErrAdmissionIncomplete
	}
	v.latest = e
	return nil
}

func (r *admissionRuntime) validateCurrentRun(ctx context.Context, v controlledView) error {
	x := *v.run
	b := v.begin
	if x.Outcome == RecoveryOutcomeFailed || x.JournalKeyID != b.KeyID || x.JournalKeyVersion != b.KeyVersion || x.JournalSecretFingerprint != b.SecretFingerprint || !x.StartedAt.Equal(b.RecordedAt) || x.RestoreBinding == nil {
		return ErrAdmissionBinding
	}
	if x.Outcome != RecoveryOutcomeRunning && !changedRecoveryTrust(x) {
		return ErrAdmissionIncomplete
	}
	if v.latest.Stage == StageConsumed {
		if x.Outcome != RecoveryOutcomeReadyToConsume && (x.JournalStage != StageConsumed || x.JournalConsumedAt == nil || !x.JournalConsumedAt.Equal(v.latest.RecordedAt)) {
			return ErrAdmissionBinding
		}
		if x.Outcome == RecoveryOutcomeRunning || x.Outcome == RecoveryOutcomeIdentityReady || x.Outcome == RecoveryOutcomeArticleDeletionReady {
			return ErrAdmissionBinding
		}
	} else if x.JournalStage != StageBegun || x.JournalConsumedAt != nil || x.Outcome == RecoveryOutcomeJournalConsumed || x.Outcome == RecoveryOutcomeSucceeded {
		return ErrAdmissionBinding
	}
	loaded, e := r.deps.Runs.Load(ctx, x.RunID)
	if e != nil {
		return admissionReadError(e)
	}
	if loaded.RunID != x.RunID || loaded.BackupRef != x.BackupRef || loaded.Outcome != x.Outcome || loaded.JournalStage != x.JournalStage || loaded.JournalKeyID != b.KeyID || loaded.JournalKeyVersion != b.KeyVersion || loaded.JournalSecretFingerprint != b.SecretFingerprint || !loaded.StartedAt.Equal(x.StartedAt) {
		return ErrAdmissionBinding
	}
	return checkedContext(ctx)
}

func changedRecoveryTrust(x AdmissionRun) bool {
	return x.AuditEventID != nil && x.InitialAdminUserID != nil && x.InitialAdminBootstrapGeneration != nil && x.InitialAdminChangedGeneration != nil && x.InitialAdminChangeAuditID != nil && x.InitialAdminChangedAt != nil && x.PendingReviewCount == 0
}

func validateRestorePlan(p RestorePlan, i InstallationBinding, b RunBinding) error {
	if p.Version != "han-db-restore/v1" || p.RunID != b.RunID || p.BackupRef != b.BackupRef || p.DeploymentID != i.DeploymentID || p.InstallationID != i.InstallationID || p.TargetDigest != i.TargetDigest || p.ArtifactBytes <= 0 {
		return ErrAdmissionBinding
	}
	for _, s := range []string{p.SourceApprovalRef, p.SourceInstallationID, p.ArtifactPath} {
		if !admissionText(s) {
			return ErrAdmissionBinding
		}
	}
	for _, s := range []string{p.ArtifactDigest, p.SourceApprovalDigest, p.SourceSchemaDigest} {
		if !sha256DigestPattern.MatchString(s) {
			return ErrAdmissionBinding
		}
	}
	return nil
}

func validatePlanReceipt(p RestorePlan, x RestoreReceipt) error {
	if x.Version != p.Version || x.RunID != p.RunID || x.BackupRef != p.BackupRef || x.DeploymentID != p.DeploymentID || x.InstallationID != p.InstallationID || x.TargetDigest != p.TargetDigest || x.ArtifactDigest != p.ArtifactDigest || x.ArtifactBytes != p.ArtifactBytes || x.SourceApprovalDigest != p.SourceApprovalDigest || x.RestoredSchemaDigest != p.SourceSchemaDigest || !sha256DigestPattern.MatchString(x.ReceiptDigest) || !admissionTime(x.RestoredAt) {
		return ErrAdmissionBinding
	}
	return nil
}

func sameBegin(a, b JournalEntry) bool { return a == b }

func copyControlledJournal(s VerifiedJournalSnapshot) VerifiedJournalSnapshot {
	s.Entries = slices.Clone(s.Entries)
	s.Prefixes = slices.Clone(s.Prefixes)
	return s
}

// A stable new read cannot replace a previously accepted namespace token.
func sameControlledJournal(a, b VerifiedJournalSnapshot) bool {
	return sameJournalObservation(a, b) && slices.Equal(a.Entries, b.Entries) && slices.Equal(a.Prefixes, b.Prefixes)
}

func exactJournalAppend(a, b VerifiedJournalSnapshot, entry JournalEntry) bool {
	if a.JournalID != b.JournalID || a.Namespace != b.Namespace || a.Epoch != b.Epoch || a.Pending != nil || b.Pending != nil ||
		a.Revision == b.Revision || !admissionText(b.Revision) || a.Head.Sequence == math.MaxUint64 ||
		entry.Sequence != a.Head.Sequence+1 || entry.PreviousRecordHash != a.Head.Hash || b.Head.Sequence != entry.Sequence ||
		len(b.Entries) != len(a.Entries)+1 || len(b.Prefixes) != len(a.Prefixes)+1 {
		return false
	}
	return slices.Equal(a.Entries, b.Entries[:len(a.Entries)]) && b.Entries[len(a.Entries)] == entry &&
		slices.Equal(a.Prefixes, b.Prefixes[:len(a.Prefixes)]) && b.Prefixes[len(a.Prefixes)] == b.Head
}
