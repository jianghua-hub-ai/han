//go:build integration

package recovery_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRecoveryFixtures(t *testing.T) {
	keyring := readFixture[keyringDocument](t, "keyring-fixtures.json")
	manifest := readFixture[backupManifest](t, "backup-manifest.json")

	t.Run("synthetic guards", func(t *testing.T) {
		validateSyntheticGuards(t, keyring, manifest)
	})

	fixture, keysByVersion, vectorsByVersion := validateKeyring(t, keyring)

	t.Run("manifest bindings and time", func(t *testing.T) {
		validateManifestBindings(t, manifest, fixture, keysByVersion, vectorsByVersion)
		assertUTCMicrosecondText(t, "backup.captured_at", manifest.Backup.CapturedAt)
		assertUTCMicrosecondText(t, "run.started_at", manifest.Run.StartedAt)
	})

	t.Run("operational gates", func(t *testing.T) {
		validateOperationalUnknowns(t, manifest.OperationalUnknowns)
		validateEntryGate(t, manifest.EntryGate)
		validateRecoverySteps(t, manifest.RecoverySteps)
		validateAnchorPreflight(t, manifest.AnchorPreflight)
	})

	t.Run("strict JSON rejects invalid object streams", testStrictJSONRejections)

	t.Run("cross-version input drift is rejected", func(t *testing.T) {
		testCrossVersionInputDrift(t, fixture.TestVectors, keysByVersion)
	})
}

type keyringDocument struct {
	SchemaVersion    string           `json:"schema_version"`
	SyntheticOnly    bool             `json:"synthetic_only"`
	ProductionUse    string           `json:"production_use"`
	MaterialEncoding string           `json:"material_encoding"`
	Fixtures         []keyringFixture `json:"fixtures"`
}

type keyringFixture struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Keys        []fixtureKey `json:"keys"`
	TestVectors []testVector `json:"test_vectors"`
}

type fixtureKey struct {
	ID       string `json:"id"`
	Version  string `json:"version"`
	State    string `json:"state"`
	Material string `json:"material"`
}

type testVector struct {
	Name                   string `json:"name"`
	SyntheticRecoveryInput string `json:"synthetic_recovery_input"`
	ExpectedKeyID          string `json:"expected_key_id"`
	ExpectedKeyVersion     string `json:"expected_key_version"`
	ExpectedFingerprint    string `json:"expected_fingerprint"`
}

type backupManifest struct {
	SchemaVersion       string            `json:"schema_version"`
	SyntheticOnly       bool              `json:"synthetic_only"`
	ProductionUse       string            `json:"production_use"`
	Backup              backupMetadata    `json:"backup"`
	KeyringFixture      keyringReference  `json:"keyring_fixture"`
	Run                 recoveryRun       `json:"run"`
	ArticleEvidence     articleEvidence   `json:"article_evidence"`
	OperationalUnknowns map[string]string `json:"operational_unknowns"`
	EntryGate           entryGate         `json:"entry_gate"`
	AnchorPreflight     anchorPreflight   `json:"anchor_preflight"`
	Cleanup             cleanupContract   `json:"cleanup"`
	RecoverySteps       []string          `json:"recovery_steps"`
}

type backupMetadata struct {
	BackupRef   string         `json:"backup_ref"`
	CapturedAt  string         `json:"captured_at"`
	Artifact    backupArtifact `json:"artifact"`
	DataClasses []string       `json:"data_classes"`
}

type backupArtifact struct {
	Status   string `json:"status"`
	Location string `json:"location"`
	SHA256   string `json:"sha256"`
}

type keyringReference struct {
	File                string   `json:"file"`
	Name                string   `json:"name"`
	ActiveKeyID         string   `json:"active_key_id"`
	ActiveKeyVersion    string   `json:"active_key_version"`
	TestVector          string   `json:"test_vector"`
	RetainedKeyVersions []string `json:"retained_key_versions"`
}

type recoveryRun struct {
	RunID                    string `json:"run_id"`
	BackupRef                string `json:"backup_ref"`
	JournalKeyID             string `json:"journal_key_id"`
	JournalKeyVersion        string `json:"journal_key_version"`
	JournalSecretFingerprint string `json:"journal_secret_fingerprint"`
	JournalStage             string `json:"journal_stage"`
	StartedAt                string `json:"started_at"`
	Outcome                  string `json:"outcome"`
}

type articleEvidence struct {
	Status          string `json:"status"`
	RequiredBinding string `json:"required_binding"`
	FutureTask      string `json:"future_task"`
}

type entryGate struct {
	RealExerciseAllowed bool   `json:"real_exercise_allowed"`
	Reason              string `json:"reason"`
	UnknownPolicy       string `json:"unknown_policy"`
	NotApplicablePolicy string `json:"not_applicable_policy"`
}

type anchorPreflight struct {
	Mode                   string `json:"mode"`
	Source                 string `json:"source"`
	Restore                string `json:"restore"`
	Reset                  string `json:"reset"`
	Bootstrap              string `json:"bootstrap"`
	Rollback               string `json:"rollback"`
	MissingOrIndeterminate string `json:"missing_or_indeterminate"`
}

type cleanupContract struct {
	ResponsibleOwner string `json:"responsible_owner"`
	Status           string `json:"status"`
	AllowedScope     string `json:"allowed_scope"`
	ForbiddenScope   string `json:"forbidden_scope"`
}

type invalidJSONCase struct {
	name string
	raw  string
}

var strictJSONRejectionCases = []invalidJSONCase{
	{
		name: "duplicate top-level field",
		raw:  `{"schema_version":"one","schema_version":"two"}`,
	},
	{
		name: "duplicate nested field",
		raw: `{
			"schema_version":"test",
			"synthetic_only":true,
			"production_use":"forbidden",
			"material_encoding":"utf8",
			"fixtures":[{
				"name":"one",
				"name":"two",
				"description":"",
				"keys":[],
				"test_vectors":[]
			}]
		}`,
	},
	{
		name: "unknown field",
		raw:  `{"unexpected":true}`,
	},
	{
		name: "trailing data",
		raw:  `{} {}`,
	},
}

func validateSyntheticGuards(
	t *testing.T,
	keyring keyringDocument,
	manifest backupManifest,
) {
	t.Helper()
	assertSyntheticDocument(t, keyring.SchemaVersion, keyring.SyntheticOnly, keyring.ProductionUse)
	assertSyntheticDocument(t, manifest.SchemaVersion, manifest.SyntheticOnly, manifest.ProductionUse)
	if manifest.Backup.Artifact.Status != "not_included" ||
		manifest.Backup.Artifact.Location != "unknown" ||
		manifest.Backup.Artifact.SHA256 != "unknown" {
		t.Fatalf("backup artifact = %#v, want synthetic metadata without an artifact", manifest.Backup.Artifact)
	}
}

func testStrictJSONRejections(t *testing.T) {
	for _, test := range strictJSONRejectionCases {
		t.Run(test.name, func(t *testing.T) {
			var document keyringDocument
			if err := decodeStrictJSON([]byte(test.raw), &document); err == nil {
				t.Fatal("decodeStrictJSON() error = nil, want rejection")
			}
		})
	}
}

func testCrossVersionInputDrift(
	t *testing.T,
	vectors []testVector,
	keysByVersion map[string]fixtureKey,
) {
	t.Helper()
	drifted := append([]testVector(nil), vectors...)
	drifted[1].SyntheticRecoveryInput += "-DRIFT"
	key := keysByVersion[drifted[1].ExpectedKeyVersion]
	drifted[1].ExpectedFingerprint = syntheticFingerprint(key.Material, drifted[1].SyntheticRecoveryInput)
	if err := validateSharedSyntheticInput(drifted); err == nil {
		t.Fatal("validateSharedSyntheticInput() error = nil, want input drift rejection")
	}
}

func readFixture[T any](t *testing.T, name string) T {
	t.Helper()
	path := filepath.Join(moduleRoot(t), "testdata", "recovery", name)
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	var document T
	if err := decodeStrictJSON(raw, &document); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	return document
}

func decodeStrictJSON[T any](raw []byte, document *T) error {
	if err := rejectDuplicateObjectKeys(raw); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(document); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("trailing JSON data: %v", err)
	}
	return nil
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	var starts []string
	if cwd, err := os.Getwd(); err == nil {
		starts = append(starts, cwd)
	}
	if _, source, _, ok := runtime.Caller(0); ok && filepath.IsAbs(source) {
		starts = append(starts, filepath.Dir(source))
	}
	if executable, err := os.Executable(); err == nil {
		starts = append(starts, filepath.Dir(executable))
	}
	for _, start := range starts {
		if root, ok := findModuleRoot(start); ok {
			return root
		}
	}
	t.Fatalf("locate module root from %v", starts)
	return ""
}

func findModuleRoot(start string) (string, bool) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", false
	}
	for {
		if fixtureModuleRoot(current) {
			return current, true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		current = parent
	}
}

func fixtureModuleRoot(root string) bool {
	required := []string{
		"go.mod",
		filepath.Join("testdata", "recovery", "keyring-fixtures.json"),
		filepath.Join("testdata", "recovery", "backup-manifest.json"),
	}
	for _, name := range required {
		info, err := os.Stat(filepath.Join(root, name))
		if err != nil || !info.Mode().IsRegular() {
			return false
		}
	}
	return true
}

func rejectDuplicateObjectKeys(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := scanJSONValue(decoder, "$"); err != nil {
		return err
	}
	if token, err := decoder.Token(); err != io.EOF {
		return fmt.Errorf("trailing JSON data after token %v: %v", token, err)
	}
	return nil
}

func scanJSONValue(decoder *json.Decoder, path string) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		return scanJSONObject(decoder, path)
	case '[':
		return scanJSONArray(decoder, path)
	default:
		return fmt.Errorf("unexpected JSON delimiter %q at %s", delimiter, path)
	}
}

func scanJSONObject(decoder *json.Decoder, path string) error {
	seen := make(map[string]struct{})
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		key, ok := token.(string)
		if !ok {
			return fmt.Errorf("non-string JSON object key at %s", path)
		}
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("duplicate JSON object key %q at %s", key, path)
		}
		seen[key] = struct{}{}
		if err := scanJSONValue(decoder, path+"."+key); err != nil {
			return err
		}
	}
	return expectJSONDelimiter(decoder, '}', path)
}

func scanJSONArray(decoder *json.Decoder, path string) error {
	for index := 0; decoder.More(); index++ {
		if err := scanJSONValue(decoder, fmt.Sprintf("%s[%d]", path, index)); err != nil {
			return err
		}
	}
	return expectJSONDelimiter(decoder, ']', path)
}

func expectJSONDelimiter(decoder *json.Decoder, want json.Delim, path string) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if token != want {
		return fmt.Errorf("JSON delimiter at %s = %v, want %q", path, token, want)
	}
	return nil
}

func assertSyntheticDocument(t *testing.T, schema string, syntheticOnly bool, productionUse string) {
	t.Helper()
	if schema == "" || !syntheticOnly || productionUse != "forbidden" {
		t.Fatalf(
			"document guards = schema:%q synthetic_only:%t production_use:%q",
			schema,
			syntheticOnly,
			productionUse,
		)
	}
}

func validateKeyring(
	t *testing.T,
	document keyringDocument,
) (keyringFixture, map[string]fixtureKey, map[string]testVector) {
	t.Helper()
	if document.MaterialEncoding != "utf8" || len(document.Fixtures) != 1 {
		t.Fatalf("keyring metadata = encoding:%q fixtures:%d", document.MaterialEncoding, len(document.Fixtures))
	}
	fixture := document.Fixtures[0]
	if fixture.Name != "rotated_complete" || len(fixture.Keys) != 3 {
		t.Fatalf("keyring fixture = name:%q keys:%d", fixture.Name, len(fixture.Keys))
	}
	keysByVersion := make(map[string]fixtureKey, len(fixture.Keys))
	activeCount := 0
	for _, key := range fixture.Keys {
		if key.ID != "recovery-fixture" || !strings.HasPrefix(key.Material, "SYNTHETIC-TEST-ONLY-") {
			t.Fatalf("unsafe or unexpected key metadata: %#v", key)
		}
		if _, duplicate := keysByVersion[key.Version]; duplicate {
			t.Fatalf("duplicate key version %q", key.Version)
		}
		keysByVersion[key.Version] = key
		if key.State == "active" {
			activeCount++
		}
	}
	if activeCount != 1 || keysByVersion["v3"].State != "active" ||
		keysByVersion["v2"].State != "verify_only" || keysByVersion["v1"].State != "verify_only" {
		t.Fatalf("key states = %#v", keysByVersion)
	}
	vectorsByVersion := validateHMACVectors(t, fixture.TestVectors, keysByVersion)
	return fixture, keysByVersion, vectorsByVersion
}

func validateHMACVectors(
	t *testing.T,
	vectors []testVector,
	keysByVersion map[string]fixtureKey,
) map[string]testVector {
	t.Helper()
	if len(vectors) != 3 {
		t.Fatalf("test vectors = %d, want 3", len(vectors))
	}
	if err := validateSharedSyntheticInput(vectors); err != nil {
		t.Fatal(err)
	}
	vectorsByVersion := make(map[string]testVector, len(vectors))
	for _, vector := range vectors {
		key, ok := keysByVersion[vector.ExpectedKeyVersion]
		if !ok || vector.ExpectedKeyID != key.ID ||
			!strings.HasPrefix(vector.SyntheticRecoveryInput, "SYNTHETIC-TEST-ONLY-") {
			t.Fatalf("invalid synthetic vector binding: %#v", vector)
		}
		actual := syntheticFingerprint(key.Material, vector.SyntheticRecoveryInput)
		if !hmac.Equal([]byte(actual), []byte(vector.ExpectedFingerprint)) {
			t.Fatalf("%s fingerprint = %q, want %q", vector.Name, actual, vector.ExpectedFingerprint)
		}
		vectorsByVersion[key.Version] = vector
	}
	for _, version := range []string{"v1", "v2", "v3"} {
		if _, ok := vectorsByVersion[version]; !ok {
			t.Fatalf("missing HMAC vector for %s", version)
		}
	}
	return vectorsByVersion
}

func validateSharedSyntheticInput(vectors []testVector) error {
	if len(vectors) == 0 {
		return fmt.Errorf("HMAC vectors are required")
	}
	input := vectors[0].SyntheticRecoveryInput
	if !strings.HasPrefix(input, "SYNTHETIC-TEST-ONLY-") {
		return fmt.Errorf("HMAC vector input is not synthetic")
	}
	for _, vector := range vectors[1:] {
		if vector.SyntheticRecoveryInput != input {
			return fmt.Errorf("HMAC vectors do not share one synthetic recovery input")
		}
	}
	return nil
}

func syntheticFingerprint(material string, input string) string {
	mac := hmac.New(sha256.New, []byte(material))
	_, _ = mac.Write([]byte(input))
	return "hmac-sha256:" + hex.EncodeToString(mac.Sum(nil))
}

func validateManifestBindings(
	t *testing.T,
	manifest backupManifest,
	fixture keyringFixture,
	keysByVersion map[string]fixtureKey,
	vectorsByVersion map[string]testVector,
) {
	t.Helper()
	if manifest.Backup.BackupRef == "" || !strings.HasPrefix(manifest.Backup.BackupRef, "synthetic-backup:") ||
		manifest.Run.RunID == "" || !strings.HasPrefix(manifest.Run.RunID, "synthetic-") {
		t.Fatalf("non-synthetic run binding: backup:%q run:%q", manifest.Backup.BackupRef, manifest.Run.RunID)
	}
	if manifest.Run.BackupRef != manifest.Backup.BackupRef ||
		manifest.KeyringFixture.Name != fixture.Name ||
		manifest.KeyringFixture.File != "keyring-fixtures.json" {
		t.Fatalf("manifest fixture binding mismatch")
	}
	active := keysByVersion["v3"]
	vector := vectorsByVersion["v3"]
	if manifest.KeyringFixture.ActiveKeyID != active.ID ||
		manifest.KeyringFixture.ActiveKeyVersion != active.Version ||
		manifest.KeyringFixture.TestVector != vector.Name ||
		manifest.Run.JournalKeyID != active.ID ||
		manifest.Run.JournalKeyVersion != active.Version ||
		manifest.Run.JournalSecretFingerprint != vector.ExpectedFingerprint {
		t.Fatalf("manifest active key or fingerprint binding mismatch")
	}
	if !reflect.DeepEqual(manifest.KeyringFixture.RetainedKeyVersions, []string{"v2", "v1"}) ||
		manifest.Run.JournalStage != "begun" || manifest.Run.Outcome != "running" {
		t.Fatalf("manifest retained keys or initial run state mismatch")
	}
}

var utcMicrosecondPattern = regexp.MustCompile(
	`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$`,
)

func assertUTCMicrosecondText(t *testing.T, name string, value string) {
	t.Helper()
	if !utcMicrosecondPattern.MatchString(value) {
		t.Fatalf("%s = %q, want RFC3339 UTC with six fractional digits", name, value)
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || parsed.Location() != time.UTC || parsed.Nanosecond()%1000 != 0 {
		t.Fatalf("%s = %q, want valid UTC microsecond time: %v", name, value, err)
	}
}

func validateOperationalUnknowns(t *testing.T, unknowns map[string]string) {
	t.Helper()
	required := []string{
		"rpo", "rto", "recovery_batch_capacity", "recovery_runtime_owner",
		"account_review_owner", "recovery_record_retention", "key_retention",
		"tombstone_inventory_retention", "q10_cycle",
		"historical_noncanonical_time_migration_plan",
		"evidence_version_compatibility_window", "deletion_operation_capacity",
		"finalizer_capacity", "key_retirement_conditions", "production_journal_volume",
		"production_anchor_provider_config_wiring", "independent_failure_domain_validation",
		"target_posix_filesystem_evidence", "future_windows_allowed_sid_list",
	}
	if len(unknowns) != len(required) {
		t.Fatalf("operational unknown fields = %d, want %d", len(unknowns), len(required))
	}
	for _, name := range required {
		if unknowns[name] != "unknown" {
			t.Fatalf("operational_unknowns.%s = %q, want unknown", name, unknowns[name])
		}
	}
}

func validateEntryGate(t *testing.T, gate entryGate) {
	t.Helper()
	if gate.RealExerciseAllowed ||
		gate.Reason != "operational_unknowns_present" ||
		gate.UnknownPolicy != "block" ||
		gate.NotApplicablePolicy != "explicit_owner_confirmation_and_recorded_basis_required" {
		t.Fatalf("entry gate = %#v", gate)
	}
}

func validateRecoverySteps(t *testing.T, steps []string) {
	t.Helper()
	expected := []string{
		"entry_gate_blocks_required_unknowns",
		"enter_maintenance_before_any_state_change",
		"bind_immutable_backup_ref",
		"read_only_preflight_current_journal_complete_keyring_and_current_independent_anchor",
		"journal_begin",
		"restore_database",
		"create_identity_recovery_run",
		"finalize_identity_and_review_accounts",
		"replay_and_verify_article_deletions",
		"advance_to_ready_to_consume",
		"journal_mark_consumed",
		"final_reconciliation_and_readiness",
		"owner_confirmed_cleanup",
	}
	if !reflect.DeepEqual(steps, expected) {
		t.Fatalf("recovery steps = %#v, want %#v", steps, expected)
	}
}

func validateAnchorPreflight(t *testing.T, preflight anchorPreflight) {
	t.Helper()
	if preflight.Mode != "read_only" ||
		preflight.Source != "current_independent_anchor" ||
		preflight.Restore != "forbidden" ||
		preflight.Reset != "forbidden" ||
		preflight.Bootstrap != "forbidden" ||
		preflight.Rollback != "forbidden" ||
		preflight.MissingOrIndeterminate !=
			"stop_and_wait_for_identity-auth-S4.5_approved_process" {
		t.Fatalf("anchor preflight = %#v", preflight)
	}
}
