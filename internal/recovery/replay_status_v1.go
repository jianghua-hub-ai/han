package recovery

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

const (
	// ReplayEvidenceVersionV1 is the only replay evidence version accepted by this consumer.
	ReplayEvidenceVersionV1 = "article-deletion-replay/v1"
	// ArticleDeletionNamespaceV1 is the article-owned Journal namespace represented by v1 evidence.
	ArticleDeletionNamespaceV1 = "article-deletion/v1"
	// readyInventoryScopeV1 is the only inventory scope accepted by v1 ready evidence.
	readyInventoryScopeV1 = "recoverable-backups-before-cutoff"
	// ReadyFixtureDigestV1 is the accepted digest of the shared v1 ready fixture.
	ReadyFixtureDigestV1 = "sha256:169e0f30407867083598ab146c950968e1b5f508ed4fd5d5198bb61e788926a7"
)

// ErrInvalidReplayStatus indicates a malformed, unsupported, or inconsistent replay status.
var ErrInvalidReplayStatus = errors.New("invalid article deletion replay status")

var (
	unsignedIntegerPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)$`)
	lowerHexDigestPattern  = regexp.MustCompile(`^sha256:[0-9a-f]+$`)
	sha256DigestPattern    = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

// ReadyReplayStatusV1 is the exact 20-field ready variant.
type ReadyReplayStatusV1 struct {
	EvidenceVersion                   string `json:"evidence_version"`
	RunID                             string `json:"run_id"`
	BackupRef                         string `json:"backup_ref"`
	Namespace                         string `json:"namespace"`
	JournalHeadSequence               uint64 `json:"journal_head_sequence"`
	JournalHeadHash                   string `json:"journal_head_hash"`
	InventoryScope                    string `json:"inventory_scope"`
	BackupCutoff                      string `json:"backup_cutoff"`
	InventoryDigest                   string `json:"inventory_digest"`
	DeletionOperationsDigest          string `json:"deletion_operations_digest"`
	AuditEventsDigest                 string `json:"audit_events_digest"`
	TombstonesTotal                   uint64 `json:"tombstones_total"`
	TombstonesReplayed                uint64 `json:"tombstones_replayed"`
	AlreadyAbsent                     uint64 `json:"already_absent"`
	DeletionOperationsReconciledCount uint64 `json:"deletion_operations_reconciled_count"`
	AuditEventsVerifiedCount          uint64 `json:"audit_events_verified_count"`
	Status                            string `json:"status"`
	EvidenceDigest                    string `json:"evidence_digest"`
	EvidenceRef                       string `json:"evidence_ref"`
	CompletedAt                       string `json:"completed_at"`
}

// UnavailableReplayStatusV1 is the exact six-field unavailable variant.
type UnavailableReplayStatusV1 struct {
	EvidenceVersion string `json:"evidence_version"`
	RunID           string `json:"run_id"`
	BackupRef       string `json:"backup_ref"`
	Namespace       string `json:"namespace"`
	Status          string `json:"status"`
	ReasonCode      string `json:"reason_code"`
}

// ReplayStatusV1 contains exactly one decoded tagged-union variant.
type ReplayStatusV1 struct {
	Ready       *ReadyReplayStatusV1
	Unavailable *UnavailableReplayStatusV1
}

// ReplayStatusV1Decoder strictly decodes v1 replay status objects.
type ReplayStatusV1Decoder struct {
	digest func([]byte) string
}

// NewReplayStatusV1Decoder creates a strict decoder. A nil digest function uses SHA-256.
func NewReplayStatusV1Decoder(digest func([]byte) string) ReplayStatusV1Decoder {
	if digest == nil {
		digest = SHA256Digest
	}
	return ReplayStatusV1Decoder{digest: digest}
}

// DecodeReplayStatusV1 decodes with the standard SHA-256 implementation.
func DecodeReplayStatusV1(data []byte) (ReplayStatusV1, error) {
	return NewReplayStatusV1Decoder(nil).Decode(data)
}

// Decode validates the exact tagged union and recomputes a digest only for ready.
func (decoder ReplayStatusV1Decoder) Decode(data []byte) (ReplayStatusV1, error) {
	object, err := decodeJSONObject(data)
	if err != nil {
		return ReplayStatusV1{}, replayStatusError(err)
	}
	status, err := decodeRequiredString(object.fields, "status")
	if err != nil {
		return ReplayStatusV1{}, replayStatusError(err)
	}
	if status == "ready" {
		return decoder.decodeReady(object)
	}
	if status == "pending" || status == "unknown" || status == "failed" {
		return decodeUnavailable(object.fields)
	}
	return ReplayStatusV1{}, replayStatusError(errors.New("unsupported status"))
}

func (decoder ReplayStatusV1Decoder) decodeReady(object decodedJSONObject) (ReplayStatusV1, error) {
	if err := requireExactFieldOrder(object, readyFieldNames); err != nil {
		return ReplayStatusV1{}, replayStatusError(err)
	}
	ready, err := decodeReadyFields(object.fields)
	if err != nil {
		return ReplayStatusV1{}, replayStatusError(err)
	}
	if err := validateReady(ready); err != nil {
		return ReplayStatusV1{}, replayStatusError(err)
	}
	canonical, err := ready.CanonicalJSON()
	if err != nil {
		return ReplayStatusV1{}, replayStatusError(err)
	}
	if decoder.digest == nil || decoder.digest(canonical) != ready.EvidenceDigest {
		return ReplayStatusV1{}, replayStatusError(errors.New("evidence digest mismatch"))
	}
	return ReplayStatusV1{Ready: &ready}, nil
}

func decodeUnavailable(fields map[string]json.RawMessage) (ReplayStatusV1, error) {
	if err := requireExactFields(fields, unavailableFieldNames); err != nil {
		return ReplayStatusV1{}, replayStatusError(err)
	}
	value := UnavailableReplayStatusV1{}
	var err error
	if value.EvidenceVersion, err = decodeRequiredString(fields, "evidence_version"); err != nil {
		return ReplayStatusV1{}, replayStatusError(err)
	}
	if value.RunID, err = decodeRequiredString(fields, "run_id"); err != nil {
		return ReplayStatusV1{}, replayStatusError(err)
	}
	if value.BackupRef, err = decodeRequiredString(fields, "backup_ref"); err != nil {
		return ReplayStatusV1{}, replayStatusError(err)
	}
	if value.Namespace, err = decodeRequiredString(fields, "namespace"); err != nil {
		return ReplayStatusV1{}, replayStatusError(err)
	}
	if value.Status, err = decodeRequiredString(fields, "status"); err != nil {
		return ReplayStatusV1{}, replayStatusError(err)
	}
	if value.ReasonCode, err = decodeRequiredString(fields, "reason_code"); err != nil {
		return ReplayStatusV1{}, replayStatusError(err)
	}
	if err := validateUnavailable(value); err != nil {
		return ReplayStatusV1{}, replayStatusError(err)
	}
	return ReplayStatusV1{Unavailable: &value}, nil
}

func decodeReadyFields(fields map[string]json.RawMessage) (ReadyReplayStatusV1, error) {
	value := ReadyReplayStatusV1{}
	stringTargets := []struct {
		name   string
		target *string
	}{
		{"evidence_version", &value.EvidenceVersion},
		{"run_id", &value.RunID},
		{"backup_ref", &value.BackupRef},
		{"namespace", &value.Namespace},
		{"journal_head_hash", &value.JournalHeadHash},
		{"inventory_scope", &value.InventoryScope},
		{"backup_cutoff", &value.BackupCutoff},
		{"inventory_digest", &value.InventoryDigest},
		{"deletion_operations_digest", &value.DeletionOperationsDigest},
		{"audit_events_digest", &value.AuditEventsDigest},
		{"status", &value.Status},
		{"evidence_digest", &value.EvidenceDigest},
		{"evidence_ref", &value.EvidenceRef},
		{"completed_at", &value.CompletedAt},
	}
	for _, item := range stringTargets {
		decoded, err := decodeRequiredString(fields, item.name)
		if err != nil {
			return ReadyReplayStatusV1{}, err
		}
		*item.target = decoded
	}
	if err := decodeReadyIntegers(fields, &value); err != nil {
		return ReadyReplayStatusV1{}, err
	}
	return value, nil
}

func decodeReadyIntegers(fields map[string]json.RawMessage, value *ReadyReplayStatusV1) error {
	integerTargets := []struct {
		name   string
		target *uint64
	}{
		{"journal_head_sequence", &value.JournalHeadSequence},
		{"tombstones_total", &value.TombstonesTotal},
		{"tombstones_replayed", &value.TombstonesReplayed},
		{"already_absent", &value.AlreadyAbsent},
		{"deletion_operations_reconciled_count", &value.DeletionOperationsReconciledCount},
		{"audit_events_verified_count", &value.AuditEventsVerifiedCount},
	}
	for _, item := range integerTargets {
		decoded, err := decodeRequiredUint64(fields, item.name)
		if err != nil {
			return err
		}
		*item.target = decoded
	}
	return nil
}

func validateReady(value ReadyReplayStatusV1) error {
	if value.Status != "ready" {
		return errors.New("ready status mismatch")
	}
	if err := validateReadyCanonical(value); err != nil {
		return err
	}
	if !sha256DigestPattern.MatchString(value.EvidenceDigest) {
		return errors.New("invalid evidence digest")
	}
	if value.EvidenceRef == "" || !isUTCRFC3339(value.CompletedAt) {
		return errors.New("invalid ready metadata")
	}
	return nil
}

func validateReadyCanonical(value ReadyReplayStatusV1) error {
	if value.EvidenceVersion != ReplayEvidenceVersionV1 || value.Namespace != ArticleDeletionNamespaceV1 {
		return errors.New("unsupported evidence binding")
	}
	if value.RunID == "" || value.BackupRef == "" {
		return errors.New("missing canonical binding")
	}
	if value.InventoryScope != readyInventoryScopeV1 {
		return errors.New("unsupported inventory scope")
	}
	if !lowerHexDigestPattern.MatchString(value.JournalHeadHash) ||
		!lowerHexDigestPattern.MatchString(value.InventoryDigest) ||
		!lowerHexDigestPattern.MatchString(value.DeletionOperationsDigest) ||
		!lowerHexDigestPattern.MatchString(value.AuditEventsDigest) {
		return errors.New("invalid canonical digest")
	}
	if !isUTCRFC3339(value.BackupCutoff) {
		return errors.New("invalid backup cutoff")
	}
	if value.TombstonesReplayed > value.TombstonesTotal ||
		value.TombstonesTotal-value.TombstonesReplayed != value.AlreadyAbsent {
		return errors.New("inconsistent tombstone counts")
	}
	return nil
}

func validateUnavailable(value UnavailableReplayStatusV1) error {
	if value.EvidenceVersion != ReplayEvidenceVersionV1 || value.Namespace != ArticleDeletionNamespaceV1 {
		return errors.New("unsupported evidence binding")
	}
	if value.RunID == "" || value.BackupRef == "" || value.ReasonCode == "" {
		return errors.New("missing unavailable binding")
	}
	if value.Status != "pending" && value.Status != "unknown" && value.Status != "failed" {
		return errors.New("invalid unavailable status")
	}
	return nil
}

// CanonicalJSON returns the fixed-order UTF-8 canonical bytes for the first 16 fields.
func (value ReadyReplayStatusV1) CanonicalJSON() ([]byte, error) {
	if err := validateReadyCanonical(value); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidReplayStatus, err)
	}
	canonical := readyCanonicalV1{
		EvidenceVersion:                   value.EvidenceVersion,
		RunID:                             value.RunID,
		BackupRef:                         value.BackupRef,
		Namespace:                         value.Namespace,
		JournalHeadSequence:               value.JournalHeadSequence,
		JournalHeadHash:                   value.JournalHeadHash,
		InventoryScope:                    value.InventoryScope,
		BackupCutoff:                      value.BackupCutoff,
		InventoryDigest:                   value.InventoryDigest,
		DeletionOperationsDigest:          value.DeletionOperationsDigest,
		AuditEventsDigest:                 value.AuditEventsDigest,
		TombstonesTotal:                   value.TombstonesTotal,
		TombstonesReplayed:                value.TombstonesReplayed,
		AlreadyAbsent:                     value.AlreadyAbsent,
		DeletionOperationsReconciledCount: value.DeletionOperationsReconciledCount,
		AuditEventsVerifiedCount:          value.AuditEventsVerifiedCount,
	}
	return json.Marshal(canonical)
}

// CanonicalDigest returns the SHA-256 digest of the fixed v1 canonical JSON.
func (value ReadyReplayStatusV1) CanonicalDigest() (string, error) {
	canonical, err := value.CanonicalJSON()
	if err != nil {
		return "", err
	}
	return SHA256Digest(canonical), nil
}

// SHA256Digest returns a lowercase prefixed SHA-256 digest.
func SHA256Digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

type readyCanonicalV1 struct {
	EvidenceVersion                   string `json:"evidence_version"`
	RunID                             string `json:"run_id"`
	BackupRef                         string `json:"backup_ref"`
	Namespace                         string `json:"namespace"`
	JournalHeadSequence               uint64 `json:"journal_head_sequence"`
	JournalHeadHash                   string `json:"journal_head_hash"`
	InventoryScope                    string `json:"inventory_scope"`
	BackupCutoff                      string `json:"backup_cutoff"`
	InventoryDigest                   string `json:"inventory_digest"`
	DeletionOperationsDigest          string `json:"deletion_operations_digest"`
	AuditEventsDigest                 string `json:"audit_events_digest"`
	TombstonesTotal                   uint64 `json:"tombstones_total"`
	TombstonesReplayed                uint64 `json:"tombstones_replayed"`
	AlreadyAbsent                     uint64 `json:"already_absent"`
	DeletionOperationsReconciledCount uint64 `json:"deletion_operations_reconciled_count"`
	AuditEventsVerifiedCount          uint64 `json:"audit_events_verified_count"`
}

var readyFieldNames = []string{
	"evidence_version", "run_id", "backup_ref", "namespace",
	"journal_head_sequence", "journal_head_hash", "inventory_scope", "backup_cutoff",
	"inventory_digest", "deletion_operations_digest", "audit_events_digest",
	"tombstones_total", "tombstones_replayed", "already_absent",
	"deletion_operations_reconciled_count", "audit_events_verified_count",
	"status", "evidence_digest", "evidence_ref", "completed_at",
}

var unavailableFieldNames = []string{
	"evidence_version", "run_id", "backup_ref", "namespace", "status", "reason_code",
}

type decodedJSONObject struct {
	fields     map[string]json.RawMessage
	fieldNames []string
}

func decodeJSONObject(data []byte) (decodedJSONObject, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil {
		return decodedJSONObject{}, err
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '{' {
		return decodedJSONObject{}, errors.New("expected object")
	}
	fields := make(map[string]json.RawMessage)
	fieldNames := make([]string, 0)
	for decoder.More() {
		nameToken, err := decoder.Token()
		if err != nil {
			return decodedJSONObject{}, err
		}
		name, ok := nameToken.(string)
		if !ok {
			return decodedJSONObject{}, errors.New("invalid object key")
		}
		if _, duplicate := fields[name]; duplicate {
			return decodedJSONObject{}, errors.New("duplicate object key")
		}
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return decodedJSONObject{}, err
		}
		fields[name] = raw
		fieldNames = append(fieldNames, name)
	}
	if _, err := decoder.Token(); err != nil {
		return decodedJSONObject{}, err
	}
	if token, err := decoder.Token(); err != io.EOF || token != nil {
		return decodedJSONObject{}, errors.New("trailing JSON value")
	}
	return decodedJSONObject{fields: fields, fieldNames: fieldNames}, nil
}

func requireExactFieldOrder(object decodedJSONObject, expected []string) error {
	if err := requireExactFields(object.fields, expected); err != nil {
		return err
	}
	for index, name := range expected {
		if object.fieldNames[index] != name {
			return errors.New("field order mismatch")
		}
	}
	return nil
}

func requireExactFields(fields map[string]json.RawMessage, expected []string) error {
	if len(fields) != len(expected) {
		return errors.New("field set mismatch")
	}
	for _, name := range expected {
		if _, ok := fields[name]; !ok {
			return errors.New("field set mismatch")
		}
	}
	return nil
}

func decodeRequiredString(fields map[string]json.RawMessage, name string) (string, error) {
	raw, ok := fields[name]
	if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return "", errors.New("missing string field")
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil || value == "" {
		return "", errors.New("invalid string field")
	}
	return value, nil
}

func decodeRequiredUint64(fields map[string]json.RawMessage, name string) (uint64, error) {
	raw, ok := fields[name]
	if !ok || !unsignedIntegerPattern.Match(bytes.TrimSpace(raw)) {
		return 0, errors.New("invalid integer field")
	}
	var value uint64
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, errors.New("invalid integer field")
	}
	return value, nil
}

func isUTCRFC3339(value string) bool {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return false
	}
	_, offset := parsed.Zone()
	return offset == 0 && (strings.HasSuffix(value, "Z") || strings.HasSuffix(value, "+00:00"))
}

func replayStatusError(cause error) error {
	return fmt.Errorf("%w: %v", ErrInvalidReplayStatus, cause)
}
