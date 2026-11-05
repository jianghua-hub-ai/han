//go:build integration

package recovery_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"han/internal/recovery"
)

const (
	articleInventoryFile   = "article-backup-inventory.json"
	articleFixturesFile    = "article-deletion-fixtures.json"
	articleRunbookFile     = "article-deletion-recovery.md"
	articleInventoryDigest = "sha256:bbbb"
	articleSharedCaseID    = "article-recovery-ready-canonical"
	articleSharedBackupRef = "backup-2026-10-20T00:00:00Z"
	articleSharedEvidence  = "evidence-01"
	articleSharedCompleted = "2026-10-20T01:00:00Z"
)

const articleReadyCanonicalV1 = `{"evidence_version":"article-deletion-replay/v1","run_id":"run-01","backup_ref":"backup-2026-10-20T00:00:00Z","namespace":"article-deletion/v1","journal_head_sequence":7,"journal_head_hash":"sha256:aaaa","inventory_scope":"recoverable-backups-before-cutoff","backup_cutoff":"2026-10-20T00:00:00Z","inventory_digest":"sha256:bbbb","deletion_operations_digest":"sha256:cccc","audit_events_digest":"sha256:dddd","tombstones_total":3,"tombstones_replayed":2,"already_absent":1,"deletion_operations_reconciled_count":3,"audit_events_verified_count":3}`

func TestArticleRecoveryFixtures(t *testing.T) {
	artifacts := readArticleArtifacts(t)

	t.Run("positive fixtures", func(t *testing.T) {
		if err := validateArticleArtifactPair(artifacts.inventory, artifacts.fixtures); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("table driven mutations", func(t *testing.T) {
		testArticleFixtureMutations(t, artifacts.inventory, artifacts.fixtures)
	})

	t.Run("trailing data", func(t *testing.T) {
		testArticleFixtureTrailingData(t, artifacts.inventory, artifacts.fixtures)
	})

	t.Run("invalid UTF-8", func(t *testing.T) {
		testArticleFixtureInvalidUTF8(t, artifacts.inventory, artifacts.fixtures)
	})

	t.Run("sensitive canaries", func(t *testing.T) {
		testArticleFixtureSensitiveCanaries(t, artifacts.inventory, artifacts.fixtures)
	})

	t.Run("shared canonical fixture", func(t *testing.T) {
		testArticleSharedCanonicalFixture(t, artifacts.inventory, artifacts.fixtures)
	})

	t.Run("root discovery", func(t *testing.T) {
		testArticleRootDiscovery(t)
	})

	t.Run("runbook", func(t *testing.T) {
		if err := validateArticleRunbook(artifacts.runbook); err != nil {
			t.Fatal(err)
		}
	})
}

type articleArtifacts struct {
	inventory []byte
	fixtures  []byte
	runbook   []byte
}

func readArticleArtifacts(t *testing.T) articleArtifacts {
	t.Helper()
	root := articleModuleRoot(t)
	files := []struct {
		name string
		path string
		dest *[]byte
	}{
		{name: articleInventoryFile, path: filepath.Join(root, "testdata", "recovery", articleInventoryFile)},
		{name: articleFixturesFile, path: filepath.Join(root, "testdata", "recovery", articleFixturesFile)},
		{name: articleRunbookFile, path: filepath.Join(root, "docs", "runbooks", articleRunbookFile)},
	}
	var artifacts articleArtifacts
	files[0].dest = &artifacts.inventory
	files[1].dest = &artifacts.fixtures
	files[2].dest = &artifacts.runbook
	var failures []string
	for _, file := range files {
		raw, err := os.ReadFile(filepath.Clean(file.path))
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", file.name, err))
			continue
		}
		*file.dest = raw
	}
	if len(failures) != 0 {
		t.Fatalf("read S4.5 artifacts:\n%s", strings.Join(failures, "\n"))
	}
	return artifacts
}

func articleModuleRoot(t *testing.T) string {
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
		if root, ok := findArticleModuleRoot(start); ok {
			return root
		}
	}
	t.Fatalf("locate Article fixture module root from %v", starts)
	return ""
}

func findArticleModuleRoot(start string) (string, bool) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", false
	}
	for {
		if isArticleModuleRoot(current) {
			return current, true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		current = parent
	}
}

func isArticleModuleRoot(root string) bool {
	required := []string{
		"go.mod",
		filepath.Join("docs", "runbooks", articleRunbookFile),
		filepath.Join("testdata", "recovery", articleInventoryFile),
		filepath.Join("testdata", "recovery", articleFixturesFile),
	}
	for _, name := range required {
		info, err := os.Stat(filepath.Join(root, name))
		if err != nil || !info.Mode().IsRegular() {
			return false
		}
	}
	return true
}

type articleFixtureError struct {
	code   string
	detail string
}

func (err *articleFixtureError) Error() string {
	return fmt.Sprintf("%s: %s", err.code, err.detail)
}

func articleError(code string, format string, args ...any) error {
	return &articleFixtureError{code: code, detail: fmt.Sprintf(format, args...)}
}

func articleErrorCode(err error) string {
	var fixtureErr *articleFixtureError
	if errors.As(err, &fixtureErr) {
		return fixtureErr.code
	}
	return ""
}

type articleJSONKind uint8

const (
	articleJSONObject articleJSONKind = iota + 1
	articleJSONArray
	articleJSONString
	articleJSONNumber
	articleJSONBoolean
	articleJSONNull
)

type articleJSONField struct {
	name  string
	value *articleJSONNode
}

type articleJSONNode struct {
	kind   articleJSONKind
	object []articleJSONField
	array  []*articleJSONNode
	value  any
}

func parseArticleJSONDocument(raw []byte) (*articleJSONNode, error) {
	if !utf8.Valid(raw) {
		return nil, articleError("invalid_json", "document is not valid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	root, err := parseArticleJSONValue(decoder, "$")
	if err != nil {
		return nil, err
	}
	if token, err := decoder.Token(); err != io.EOF || token != nil {
		return nil, articleError("invalid_json", "trailing JSON data")
	}
	return root, nil
}

func parseArticleJSONValue(decoder *json.Decoder, path string) (*articleJSONNode, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, articleError("invalid_json", "%s: %v", path, err)
	}
	delimiter, isDelimiter := token.(json.Delim)
	if isDelimiter {
		switch delimiter {
		case '{':
			return parseArticleJSONObject(decoder, path)
		case '[':
			return parseArticleJSONArray(decoder, path)
		default:
			return nil, articleError("invalid_json", "%s: unexpected delimiter %q", path, delimiter)
		}
	}
	switch value := token.(type) {
	case string:
		return &articleJSONNode{kind: articleJSONString, value: value}, nil
	case json.Number:
		return &articleJSONNode{kind: articleJSONNumber, value: value}, nil
	case bool:
		return &articleJSONNode{kind: articleJSONBoolean, value: value}, nil
	case nil:
		return &articleJSONNode{kind: articleJSONNull}, nil
	default:
		return nil, articleError("invalid_json", "%s: unsupported token", path)
	}
}

func parseArticleJSONObject(decoder *json.Decoder, path string) (*articleJSONNode, error) {
	node := &articleJSONNode{kind: articleJSONObject}
	seen := make(map[string]struct{})
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, articleError("invalid_json", "%s: %v", path, err)
		}
		name, ok := token.(string)
		if !ok {
			return nil, articleError("invalid_json", "%s: non-string object key", path)
		}
		if _, duplicate := seen[name]; duplicate {
			return nil, articleError("duplicate_key", "%s: duplicate key %q", path, name)
		}
		seen[name] = struct{}{}
		value, err := parseArticleJSONValue(decoder, path+"/"+escapeArticlePointer(name))
		if err != nil {
			return nil, err
		}
		node.object = append(node.object, articleJSONField{name: name, value: value})
	}
	if _, err := decoder.Token(); err != nil {
		return nil, articleError("invalid_json", "%s: %v", path, err)
	}
	return node, nil
}

func parseArticleJSONArray(decoder *json.Decoder, path string) (*articleJSONNode, error) {
	node := &articleJSONNode{kind: articleJSONArray}
	for index := 0; decoder.More(); index++ {
		value, err := parseArticleJSONValue(decoder, fmt.Sprintf("%s/%d", path, index))
		if err != nil {
			return nil, err
		}
		node.array = append(node.array, value)
	}
	if _, err := decoder.Token(); err != nil {
		return nil, articleError("invalid_json", "%s: %v", path, err)
	}
	return node, nil
}

func (node *articleJSONNode) marshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	if err := node.writeJSON(&buffer); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (node *articleJSONNode) writeJSON(buffer *bytes.Buffer) error {
	switch node.kind {
	case articleJSONObject:
		return node.writeJSONObject(buffer)
	case articleJSONArray:
		return node.writeJSONArray(buffer)
	case articleJSONString, articleJSONNumber, articleJSONBoolean, articleJSONNull:
		return node.writeJSONScalar(buffer)
	default:
		return articleError("invalid_json", "unknown JSON node kind")
	}
}

func (node *articleJSONNode) writeJSONObject(buffer *bytes.Buffer) error {
	buffer.WriteByte('{')
	for index, field := range node.object {
		if index > 0 {
			buffer.WriteByte(',')
		}
		name, _ := json.Marshal(field.name)
		buffer.Write(name)
		buffer.WriteByte(':')
		if err := field.value.writeJSON(buffer); err != nil {
			return err
		}
	}
	buffer.WriteByte('}')
	return nil
}

func (node *articleJSONNode) writeJSONArray(buffer *bytes.Buffer) error {
	buffer.WriteByte('[')
	for index, value := range node.array {
		if index > 0 {
			buffer.WriteByte(',')
		}
		if err := value.writeJSON(buffer); err != nil {
			return err
		}
	}
	buffer.WriteByte(']')
	return nil
}

func (node *articleJSONNode) writeJSONScalar(buffer *bytes.Buffer) error {
	switch node.kind {
	case articleJSONString, articleJSONBoolean:
		raw, _ := json.Marshal(node.value)
		buffer.Write(raw)
	case articleJSONNumber:
		buffer.WriteString(node.value.(json.Number).String())
	case articleJSONNull:
		buffer.WriteString("null")
	default:
		return articleError("invalid_json", "non-scalar JSON node")
	}
	return nil
}

func (node *articleJSONNode) clone() *articleJSONNode {
	copyNode := &articleJSONNode{kind: node.kind, value: node.value}
	for _, field := range node.object {
		copyNode.object = append(copyNode.object, articleJSONField{
			name: field.name, value: field.value.clone(),
		})
	}
	for _, value := range node.array {
		copyNode.array = append(copyNode.array, value.clone())
	}
	return copyNode
}

func (node *articleJSONNode) field(name string) (*articleJSONNode, bool) {
	for _, field := range node.object {
		if field.name == name {
			return field.value, true
		}
	}
	return nil, false
}

func (node *articleJSONNode) hasField(name string) bool {
	_, ok := node.field(name)
	return ok
}

func validateArticleArtifactPair(inventoryRaw []byte, fixturesRaw []byte) error {
	inventoryNode, err := parseArticleJSONDocument(inventoryRaw)
	if err != nil {
		return err
	}
	fixturesNode, err := parseArticleJSONDocument(fixturesRaw)
	if err != nil {
		return err
	}
	if err := scanArticleCanaries(inventoryNode, "$inventory"); err != nil {
		return err
	}
	if err := scanArticleFixtureCanaries(fixturesNode); err != nil {
		return err
	}
	if err := validateArticleInventorySchema(inventoryNode); err != nil {
		return err
	}
	if err := validateArticleFixturesSchema(fixturesNode); err != nil {
		return err
	}
	var inventory articleInventoryDocument
	if err := json.Unmarshal(inventoryRaw, &inventory); err != nil {
		return articleError("wrong_type", "decode inventory: %v", err)
	}
	var fixtures articleDeletionDocument
	if err := json.Unmarshal(fixturesRaw, &fixtures); err != nil {
		return articleError("wrong_type", "decode deletion fixtures: %v", err)
	}
	if err := validateArticleInventoryValues(inventory); err != nil {
		return err
	}
	return validateArticleDeletionValues(fixtures, fixturesNode, inventory)
}

var articleInventoryRootFields = []string{
	"schema_version", "synthetic_only", "production_use",
	"inventory_scope", "backups", "inventory_digest",
}

var articleInventoryBackupFields = []string{
	"backup_id", "backup_ref", "created_at", "status", "retired_at",
	"retirement_owner_ref", "retirement_evidence_ref",
}

func validateArticleInventorySchema(root *articleJSONNode) error {
	if err := requireArticleObjectOrder(root, articleInventoryRootFields); err != nil {
		return err
	}
	for _, name := range []string{
		"schema_version", "production_use", "inventory_scope", "inventory_digest",
	} {
		if err := requireArticleFieldKind(root, name, articleJSONString); err != nil {
			return err
		}
	}
	if err := requireArticleFieldKind(root, "synthetic_only", articleJSONBoolean); err != nil {
		return err
	}
	backups, err := requireArticleField(root, "backups", articleJSONArray)
	if err != nil {
		return err
	}
	if len(backups.array) == 0 {
		return articleError("missing_field", "backups must not be empty")
	}
	for _, backup := range backups.array {
		if err := validateArticleBackupSchema(backup); err != nil {
			return err
		}
	}
	return nil
}

func validateArticleBackupSchema(backup *articleJSONNode) error {
	if err := requireArticleObjectOrder(backup, articleInventoryBackupFields); err != nil {
		return err
	}
	for _, name := range []string{"backup_id", "backup_ref", "created_at", "status"} {
		if err := requireArticleFieldKind(backup, name, articleJSONString); err != nil {
			return err
		}
	}
	for _, name := range []string{
		"retired_at", "retirement_owner_ref", "retirement_evidence_ref",
	} {
		field, ok := backup.field(name)
		if !ok {
			return articleError("missing_field", "missing field %q", name)
		}
		if field.kind != articleJSONString && field.kind != articleJSONNull {
			return articleError("wrong_type", "%s must be string or null", name)
		}
	}
	return nil
}

var articleFixturesRootFields = []string{
	"schema_version", "synthetic_only", "production_use", "inventory_file",
	"keys", "operational_unknowns", "entry_gate", "positive_cases", "negative_cases",
}

var articleKeyFields = []string{"key_id", "key_version", "state", "synthetic_material"}

var articleUnknownFields = []string{
	"tombstone_retention", "key_retention", "inventory_retention",
	"evidence_retention", "rpo", "rto",
}

var articleEntryGateFields = []string{
	"real_recovery_allowed", "retirement_authorized", "cleanup_allowed",
	"reason", "unknown_policy",
}

var articlePositiveCaseFields = []string{
	"case_id", "description", "references", "tombstone", "replay_status", "expected_gate",
}

var articleReferenceFields = []string{
	"inventory_backup_id", "backup_ref", "key_id", "key_version",
	"deletion_operation_id", "evidence_ref",
}

var articleTombstoneFields = []string{
	"namespace", "sequence", "article_id", "article_version", "deletion_operation_id",
	"deleted_at", "key_id", "key_version", "stage", "previous_record_hash", "record_hmac",
}

var articleExpectedGateFields = []string{
	"replay_variant", "retirement_preconditions_met",
	"real_recovery_allowed", "cleanup_allowed", "reason_code",
}

var articleNegativeCaseFields = []string{
	"case_id", "base_case_id", "target_file", "target_path",
	"mutation", "operand_json", "expected_error",
}

func validateArticleFixturesSchema(root *articleJSONNode) error {
	if err := requireArticleObjectOrder(root, articleFixturesRootFields); err != nil {
		return err
	}
	for _, name := range []string{"schema_version", "production_use", "inventory_file"} {
		if err := requireArticleFieldKind(root, name, articleJSONString); err != nil {
			return err
		}
	}
	if err := requireArticleFieldKind(root, "synthetic_only", articleJSONBoolean); err != nil {
		return err
	}
	if err := validateArticleKeysSchema(root); err != nil {
		return err
	}
	if err := validateArticleGateSchema(root); err != nil {
		return err
	}
	if err := validateArticlePositiveCasesSchema(root); err != nil {
		return err
	}
	return validateArticleNegativeCasesSchema(root)
}

func validateArticleKeysSchema(root *articleJSONNode) error {
	keys, err := requireArticleField(root, "keys", articleJSONArray)
	if err != nil {
		return err
	}
	if len(keys.array) == 0 {
		return articleError("missing_field", "keys must not be empty")
	}
	for _, key := range keys.array {
		if err := requireArticleObjectOrder(key, articleKeyFields); err != nil {
			return err
		}
		for _, name := range articleKeyFields {
			if err := requireArticleFieldKind(key, name, articleJSONString); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateArticleGateSchema(root *articleJSONNode) error {
	unknowns, err := requireArticleField(root, "operational_unknowns", articleJSONObject)
	if err != nil {
		return err
	}
	if err := requireArticleObjectOrder(unknowns, articleUnknownFields); err != nil {
		return err
	}
	for _, name := range articleUnknownFields {
		if err := requireArticleFieldKind(unknowns, name, articleJSONString); err != nil {
			return err
		}
	}
	gate, err := requireArticleField(root, "entry_gate", articleJSONObject)
	if err != nil {
		return err
	}
	if err := requireArticleObjectOrder(gate, articleEntryGateFields); err != nil {
		return err
	}
	for _, name := range articleEntryGateFields[:3] {
		if err := requireArticleFieldKind(gate, name, articleJSONBoolean); err != nil {
			return err
		}
	}
	for _, name := range articleEntryGateFields[3:] {
		if err := requireArticleFieldKind(gate, name, articleJSONString); err != nil {
			return err
		}
	}
	return nil
}

func validateArticlePositiveCasesSchema(root *articleJSONNode) error {
	cases, err := requireArticleField(root, "positive_cases", articleJSONArray)
	if err != nil {
		return err
	}
	if len(cases.array) == 0 {
		return articleError("missing_field", "positive_cases must not be empty")
	}
	for _, recoveryCase := range cases.array {
		if err := validateArticlePositiveCaseSchema(recoveryCase); err != nil {
			return err
		}
	}
	return nil
}

func validateArticlePositiveCaseSchema(recoveryCase *articleJSONNode) error {
	if err := requireArticleObjectOrder(recoveryCase, articlePositiveCaseFields); err != nil {
		return err
	}
	for _, name := range []string{"case_id", "description"} {
		if err := requireArticleFieldKind(recoveryCase, name, articleJSONString); err != nil {
			return err
		}
	}
	if err := validateArticleReferencesSchema(recoveryCase); err != nil {
		return err
	}
	if err := validateArticleTombstoneSchema(recoveryCase); err != nil {
		return err
	}
	if err := validateArticleReplayStatusSchema(recoveryCase); err != nil {
		return err
	}
	return validateArticleExpectedGateSchema(recoveryCase)
}

func validateArticleReferencesSchema(recoveryCase *articleJSONNode) error {
	references, err := requireArticleField(recoveryCase, "references", articleJSONObject)
	if err != nil {
		return err
	}
	if err := requireArticleObjectOrder(references, articleReferenceFields); err != nil {
		return err
	}
	for _, name := range articleReferenceFields[:5] {
		if err := requireArticleFieldKind(references, name, articleJSONString); err != nil {
			return err
		}
	}
	evidenceRef, ok := references.field("evidence_ref")
	if !ok {
		return articleError("missing_field", "missing evidence_ref")
	}
	if evidenceRef.kind != articleJSONString && evidenceRef.kind != articleJSONNull {
		return articleError("wrong_type", "evidence_ref must be string or null")
	}
	return nil
}

func validateArticleTombstoneSchema(recoveryCase *articleJSONNode) error {
	tombstone, err := requireArticleField(recoveryCase, "tombstone", articleJSONObject)
	if err != nil {
		return err
	}
	if err := requireArticleObjectOrder(tombstone, articleTombstoneFields); err != nil {
		return err
	}
	for _, name := range []string{
		"namespace", "article_id", "deletion_operation_id", "deleted_at",
		"key_id", "key_version", "stage", "previous_record_hash", "record_hmac",
	} {
		if err := requireArticleFieldKind(tombstone, name, articleJSONString); err != nil {
			return err
		}
	}
	for _, name := range []string{"sequence", "article_version"} {
		field, err := requireArticleField(tombstone, name, articleJSONNumber)
		if err != nil {
			return err
		}
		if !articlePositiveIntegerPattern.MatchString(field.value.(json.Number).String()) {
			return articleError("wrong_type", "%s must be a positive integer", name)
		}
	}
	return nil
}

func validateArticleReplayStatusSchema(recoveryCase *articleJSONNode) error {
	status, err := requireArticleField(recoveryCase, "replay_status", articleJSONObject)
	if err != nil {
		return err
	}
	statusField, ok := status.field("status")
	if !ok || statusField.kind != articleJSONString {
		return articleError("missing_field", "replay status is required")
	}
	statusName := statusField.value.(string)
	if statusName == "ready" {
		if status.hasField("reason_code") {
			return articleError("mixed_variant", "ready status contains unavailable fields")
		}
		return requireArticleObjectOrder(status, articleReadyFields)
	}
	if statusName == "pending" || statusName == "unknown" || statusName == "failed" {
		for _, name := range articleReadyOnlyFields {
			if status.hasField(name) {
				return articleError("mixed_variant", "unavailable status contains ready field %q", name)
			}
		}
		return requireArticleObjectOrder(status, articleUnavailableFields)
	}
	return articleError("invalid_state", "unsupported replay status %q", statusName)
}

var articleReadyFields = []string{
	"evidence_version", "run_id", "backup_ref", "namespace",
	"journal_head_sequence", "journal_head_hash", "inventory_scope", "backup_cutoff",
	"inventory_digest", "deletion_operations_digest", "audit_events_digest",
	"tombstones_total", "tombstones_replayed", "already_absent",
	"deletion_operations_reconciled_count", "audit_events_verified_count",
	"status", "evidence_digest", "evidence_ref", "completed_at",
}

var articleUnavailableFields = []string{
	"evidence_version", "run_id", "backup_ref", "namespace", "status", "reason_code",
}

var articleReadyOnlyFields = []string{
	"journal_head_sequence", "journal_head_hash", "inventory_scope", "backup_cutoff",
	"inventory_digest", "deletion_operations_digest", "audit_events_digest",
	"tombstones_total", "tombstones_replayed", "already_absent",
	"deletion_operations_reconciled_count", "audit_events_verified_count",
	"evidence_digest", "evidence_ref", "completed_at",
}

func validateArticleExpectedGateSchema(recoveryCase *articleJSONNode) error {
	gate, err := requireArticleField(recoveryCase, "expected_gate", articleJSONObject)
	if err != nil {
		return err
	}
	if err := requireArticleObjectOrder(gate, articleExpectedGateFields); err != nil {
		return err
	}
	for _, name := range []string{"replay_variant", "reason_code"} {
		if err := requireArticleFieldKind(gate, name, articleJSONString); err != nil {
			return err
		}
	}
	for _, name := range []string{
		"retirement_preconditions_met", "real_recovery_allowed", "cleanup_allowed",
	} {
		if err := requireArticleFieldKind(gate, name, articleJSONBoolean); err != nil {
			return err
		}
	}
	return nil
}

func validateArticleNegativeCasesSchema(root *articleJSONNode) error {
	cases, err := requireArticleField(root, "negative_cases", articleJSONArray)
	if err != nil {
		return err
	}
	if len(cases.array) == 0 {
		return articleError("missing_field", "negative_cases must not be empty")
	}
	for _, negative := range cases.array {
		if err := requireArticleObjectOrder(negative, articleNegativeCaseFields); err != nil {
			return err
		}
		for _, name := range []string{
			"case_id", "base_case_id", "target_file", "mutation", "expected_error",
		} {
			if err := requireArticleFieldKind(negative, name, articleJSONString); err != nil {
				return err
			}
		}
		for _, name := range []string{"target_path", "operand_json"} {
			if _, err := requireArticleField(negative, name, articleJSONString); err != nil {
				return err
			}
		}
	}
	return nil
}

func requireArticleObjectOrder(node *articleJSONNode, expected []string) error {
	if node.kind != articleJSONObject {
		return articleError("wrong_type", "expected JSON object")
	}
	expectedSet := make(map[string]struct{}, len(expected))
	for _, name := range expected {
		expectedSet[name] = struct{}{}
		if !node.hasField(name) {
			return articleError("missing_field", "missing field %q", name)
		}
	}
	for _, field := range node.object {
		if _, ok := expectedSet[field.name]; !ok {
			return articleError("extra_field", "unexpected field %q", field.name)
		}
	}
	for index, name := range expected {
		if node.object[index].name != name {
			return articleError("invalid_order", "field %d = %q, want %q", index, node.object[index].name, name)
		}
	}
	return nil
}

func requireArticleField(
	node *articleJSONNode,
	name string,
	kind articleJSONKind,
) (*articleJSONNode, error) {
	field, ok := node.field(name)
	if !ok {
		return nil, articleError("missing_field", "missing field %q", name)
	}
	if field.kind != kind {
		return nil, articleError("wrong_type", "%s has the wrong JSON type", name)
	}
	return field, nil
}

func requireArticleFieldKind(node *articleJSONNode, name string, kind articleJSONKind) error {
	field, err := requireArticleField(node, name, kind)
	if err != nil {
		return err
	}
	if kind == articleJSONString && field.value.(string) == "" {
		return articleError("missing_field", "%s must not be empty", name)
	}
	return nil
}

type articleInventoryDocument struct {
	SchemaVersion   string                   `json:"schema_version"`
	SyntheticOnly   bool                     `json:"synthetic_only"`
	ProductionUse   string                   `json:"production_use"`
	InventoryScope  string                   `json:"inventory_scope"`
	Backups         []articleInventoryBackup `json:"backups"`
	InventoryDigest string                   `json:"inventory_digest"`
}

type articleInventoryBackup struct {
	BackupID              string  `json:"backup_id"`
	BackupRef             string  `json:"backup_ref"`
	CreatedAt             string  `json:"created_at"`
	Status                string  `json:"status"`
	RetiredAt             *string `json:"retired_at"`
	RetirementOwnerRef    *string `json:"retirement_owner_ref"`
	RetirementEvidenceRef *string `json:"retirement_evidence_ref"`
}

type articleDeletionDocument struct {
	SchemaVersion       string                     `json:"schema_version"`
	SyntheticOnly       bool                       `json:"synthetic_only"`
	ProductionUse       string                     `json:"production_use"`
	InventoryFile       string                     `json:"inventory_file"`
	Keys                []articleDeletionKey       `json:"keys"`
	OperationalUnknowns articleOperationalUnknowns `json:"operational_unknowns"`
	EntryGate           articleEntryGate           `json:"entry_gate"`
	PositiveCases       []articleRecoveryCase      `json:"positive_cases"`
	NegativeCases       []articleNegativeMutation  `json:"negative_cases"`
}

type articleDeletionKey struct {
	KeyID             string `json:"key_id"`
	KeyVersion        string `json:"key_version"`
	State             string `json:"state"`
	SyntheticMaterial string `json:"synthetic_material"`
}

type articleOperationalUnknowns struct {
	TombstoneRetention string `json:"tombstone_retention"`
	KeyRetention       string `json:"key_retention"`
	InventoryRetention string `json:"inventory_retention"`
	EvidenceRetention  string `json:"evidence_retention"`
	RPO                string `json:"rpo"`
	RTO                string `json:"rto"`
}

type articleEntryGate struct {
	RealRecoveryAllowed  bool   `json:"real_recovery_allowed"`
	RetirementAuthorized bool   `json:"retirement_authorized"`
	CleanupAllowed       bool   `json:"cleanup_allowed"`
	Reason               string `json:"reason"`
	UnknownPolicy        string `json:"unknown_policy"`
}

type articleRecoveryCase struct {
	CaseID       string                `json:"case_id"`
	Description  string                `json:"description"`
	References   articleCaseReferences `json:"references"`
	Tombstone    articleTombstone      `json:"tombstone"`
	ReplayStatus json.RawMessage       `json:"replay_status"`
	ExpectedGate articleExpectedGate   `json:"expected_gate"`
}

type articleCaseReferences struct {
	InventoryBackupID   string  `json:"inventory_backup_id"`
	BackupRef           string  `json:"backup_ref"`
	KeyID               string  `json:"key_id"`
	KeyVersion          string  `json:"key_version"`
	DeletionOperationID string  `json:"deletion_operation_id"`
	EvidenceRef         *string `json:"evidence_ref"`
}

type articleTombstone struct {
	Namespace           string `json:"namespace"`
	Sequence            uint64 `json:"sequence"`
	ArticleID           string `json:"article_id"`
	ArticleVersion      uint64 `json:"article_version"`
	DeletionOperationID string `json:"deletion_operation_id"`
	DeletedAt           string `json:"deleted_at"`
	KeyID               string `json:"key_id"`
	KeyVersion          string `json:"key_version"`
	Stage               string `json:"stage"`
	PreviousRecordHash  string `json:"previous_record_hash"`
	RecordHMAC          string `json:"record_hmac"`
}

type articleExpectedGate struct {
	ReplayVariant              string `json:"replay_variant"`
	RetirementPreconditionsMet bool   `json:"retirement_preconditions_met"`
	RealRecoveryAllowed        bool   `json:"real_recovery_allowed"`
	CleanupAllowed             bool   `json:"cleanup_allowed"`
	ReasonCode                 string `json:"reason_code"`
}

type articleNegativeMutation struct {
	CaseID        string `json:"case_id"`
	BaseCaseID    string `json:"base_case_id"`
	TargetFile    string `json:"target_file"`
	TargetPath    string `json:"target_path"`
	Mutation      string `json:"mutation"`
	OperandJSON   string `json:"operand_json"`
	ExpectedError string `json:"expected_error"`
}

var articlePositiveIntegerPattern = regexp.MustCompile(`^[1-9][0-9]*$`)
var articleSHA256Pattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
var articleHMACPattern = regexp.MustCompile(`^hmac-sha256:[0-9a-f]{64}$`)

func validateArticleInventoryValues(inventory articleInventoryDocument) error {
	if inventory.SchemaVersion != "article-backup-inventory-fixture/v1" {
		return articleError("unsupported_version", "inventory schema %q", inventory.SchemaVersion)
	}
	if !inventory.SyntheticOnly || inventory.ProductionUse != "forbidden" {
		return articleError("invalid_state", "inventory synthetic guards are invalid")
	}
	if inventory.InventoryScope != "recoverable-backups-before-cutoff" {
		return articleError("invalid_state", "inventory scope is invalid")
	}
	if inventory.InventoryDigest != articleInventoryDigest {
		return articleError("digest_mismatch", "inventory digest must equal fixed synthetic reference")
	}
	return validateArticleBackups(inventory.Backups)
}

func validateArticleBackups(backups []articleInventoryBackup) error {
	seenIDs := make(map[string]struct{})
	seenRefs := make(map[string]struct{})
	var previous articleInventoryBackup
	var previousTime time.Time
	for index, backup := range backups {
		createdAt, err := parseArticleMicrosecondTime(backup.CreatedAt)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(backup.BackupID, "synthetic-backup-") || backup.BackupRef == "" {
			return articleError("invalid_state", "backup %d has invalid synthetic identity", index)
		}
		if _, duplicate := seenIDs[backup.BackupID]; duplicate {
			return articleError("invalid_state", "duplicate backup_id %q", backup.BackupID)
		}
		if _, duplicate := seenRefs[backup.BackupRef]; duplicate {
			return articleError("invalid_state", "duplicate backup_ref %q", backup.BackupRef)
		}
		seenIDs[backup.BackupID] = struct{}{}
		seenRefs[backup.BackupRef] = struct{}{}
		if err := validateArticleBackupState(backup, createdAt); err != nil {
			return err
		}
		if index > 0 && (createdAt.Before(previousTime) ||
			(createdAt.Equal(previousTime) && backup.BackupID <= previous.BackupID)) {
			return articleError("invalid_order", "backups are not in canonical order")
		}
		previous, previousTime = backup, createdAt
	}
	return nil
}

func validateArticleBackupState(backup articleInventoryBackup, createdAt time.Time) error {
	switch backup.Status {
	case "recoverable":
		if backup.RetiredAt != nil ||
			backup.RetirementOwnerRef != nil ||
			backup.RetirementEvidenceRef != nil {
			return articleError("invalid_state", "recoverable backup has retirement metadata")
		}
	case "retired":
		if backup.RetiredAt == nil ||
			backup.RetirementOwnerRef == nil ||
			backup.RetirementEvidenceRef == nil ||
			*backup.RetirementOwnerRef == "" ||
			*backup.RetirementEvidenceRef == "" {
			return articleError("invalid_state", "retired backup lacks retirement metadata")
		}
		retiredAt, err := parseArticleMicrosecondTime(*backup.RetiredAt)
		if err != nil || !retiredAt.After(createdAt) {
			return articleError("invalid_time", "retired_at must be later than created_at")
		}
	default:
		return articleError("invalid_state", "unsupported backup status %q", backup.Status)
	}
	return nil
}

func parseArticleMicrosecondTime(value string) (time.Time, error) {
	if !utcMicrosecondPattern.MatchString(value) {
		return time.Time{}, articleError("invalid_time", "%q is not UTC microsecond text", value)
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || parsed.Location() != time.UTC || parsed.Nanosecond()%1000 != 0 {
		return time.Time{}, articleError("invalid_time", "%q is not valid UTC microsecond time", value)
	}
	return parsed, nil
}

func validateArticleDeletionValues(
	fixtures articleDeletionDocument,
	fixturesNode *articleJSONNode,
	inventory articleInventoryDocument,
) error {
	if fixtures.SchemaVersion != "article-deletion-fixtures/v1" {
		return articleError("unsupported_version", "fixture schema %q", fixtures.SchemaVersion)
	}
	if !fixtures.SyntheticOnly || fixtures.ProductionUse != "forbidden" {
		return articleError("invalid_state", "fixture synthetic guards are invalid")
	}
	if fixtures.InventoryFile != articleInventoryFile ||
		filepath.Base(fixtures.InventoryFile) != fixtures.InventoryFile {
		return articleError("reference_mismatch", "inventory file reference is invalid")
	}
	keys, err := validateArticleKeys(fixtures.Keys)
	if err != nil {
		return err
	}
	if err := validateArticleOperationalGate(fixtures.OperationalUnknowns, fixtures.EntryGate); err != nil {
		return err
	}
	if err := validateArticleMutationCatalog(fixtures); err != nil {
		return err
	}
	return validateArticlePositiveCases(fixtures, fixturesNode, inventory, keys)
}

func validateArticleKeys(keys []articleDeletionKey) (map[string]articleDeletionKey, error) {
	byBinding := make(map[string]articleDeletionKey)
	active := 0
	for _, key := range keys {
		binding := key.KeyID + "\x00" + key.KeyVersion
		if key.KeyID == "" || key.KeyVersion == "" ||
			!strings.HasPrefix(key.SyntheticMaterial, "SYNTHETIC-TEST-ONLY-") {
			return nil, articleError("invalid_state", "invalid synthetic Article key")
		}
		if _, duplicate := byBinding[binding]; duplicate {
			return nil, articleError("invalid_state", "duplicate Article key binding")
		}
		if key.State == "active" {
			active++
		} else if key.State != "verify_only" {
			return nil, articleError("invalid_state", "invalid Article key state")
		}
		byBinding[binding] = key
	}
	if active != 1 {
		return nil, articleError("invalid_state", "Article keys require exactly one active key")
	}
	return byBinding, nil
}

func validateArticleOperationalGate(
	unknowns articleOperationalUnknowns,
	gate articleEntryGate,
) error {
	values := []string{
		unknowns.TombstoneRetention, unknowns.KeyRetention, unknowns.InventoryRetention,
		unknowns.EvidenceRetention, unknowns.RPO, unknowns.RTO,
	}
	for _, value := range values {
		if value != "unknown" {
			return articleError("unknown_gate_open", "operational unknown was replaced")
		}
	}
	if gate.RealRecoveryAllowed || gate.RetirementAuthorized || gate.CleanupAllowed ||
		gate.Reason != "operational_unknowns_present" || gate.UnknownPolicy != "block" {
		return articleError("unknown_gate_open", "entry gate is not fail closed")
	}
	return nil
}

var articleAllowedMutationNames = map[string]struct{}{
	"remove_field": {}, "add_field": {}, "replace_value": {},
	"duplicate_key": {}, "reorder_fields": {},
}

var articleAllowedExpectedErrors = map[string]struct{}{
	"invalid_json": {}, "missing_field": {}, "extra_field": {}, "wrong_type": {},
	"duplicate_key": {}, "invalid_order": {}, "invalid_time": {}, "invalid_state": {},
	"pii_forbidden": {}, "reference_mismatch": {}, "unretired_backup": {},
	"digest_mismatch": {}, "mixed_variant": {}, "unsupported_version": {},
	"unknown_gate_open": {},
}

func validateArticleMutationCatalog(fixtures articleDeletionDocument) error {
	positive := make(map[string]struct{}, len(fixtures.PositiveCases))
	for _, recoveryCase := range fixtures.PositiveCases {
		positive[recoveryCase.CaseID] = struct{}{}
	}
	seen := make(map[string]struct{})
	for _, mutation := range fixtures.NegativeCases {
		if _, duplicate := seen[mutation.CaseID]; duplicate || mutation.CaseID == "" {
			return articleError("invalid_state", "duplicate or empty mutation case_id")
		}
		seen[mutation.CaseID] = struct{}{}
		if _, ok := positive[mutation.BaseCaseID]; !ok {
			return articleError("reference_mismatch", "unknown mutation base case")
		}
		if mutation.TargetFile != articleInventoryFile &&
			mutation.TargetFile != articleFixturesFile {
			return articleError("reference_mismatch", "unsupported mutation target file")
		}
		if mutation.TargetPath != "" && !strings.HasPrefix(mutation.TargetPath, "/") {
			return articleError("invalid_state", "mutation target is not an RFC 6901 pointer")
		}
		if _, ok := articleAllowedMutationNames[mutation.Mutation]; !ok {
			return articleError("invalid_state", "unsupported mutation")
		}
		if _, ok := articleAllowedExpectedErrors[mutation.ExpectedError]; !ok {
			return articleError("invalid_state", "unsupported expected_error")
		}
		if mutation.Mutation == "remove_field" && mutation.OperandJSON != "" {
			return articleError("invalid_state", "remove_field operand must be empty")
		}
	}
	return nil
}

func validateArticlePositiveCases(
	fixtures articleDeletionDocument,
	fixturesNode *articleJSONNode,
	inventory articleInventoryDocument,
	keys map[string]articleDeletionKey,
) error {
	inventoryByID := make(map[string]articleInventoryBackup, len(inventory.Backups))
	for _, backup := range inventory.Backups {
		inventoryByID[backup.BackupID] = backup
	}
	casesNode, _ := fixturesNode.field("positive_cases")
	state := articleCoverageState{unavailable: make(map[string]bool)}
	seenCases := make(map[string]struct{})
	seenOperations := make(map[string]struct{})
	for index, recoveryCase := range fixtures.PositiveCases {
		if err := validateArticleRecoveryCase(
			recoveryCase, casesNode.array[index], inventory, inventoryByID,
			keys, seenCases, seenOperations, &state,
		); err != nil {
			return err
		}
	}
	if !state.ready || !state.unavailable["pending"] ||
		!state.unavailable["unknown"] || !state.unavailable["failed"] ||
		!state.retirementTrue || !state.retirementFalse ||
		state.sharedCanonical != 1 {
		return articleError("invalid_state", "positive cases do not cover all required branches")
	}
	return nil
}

type articleCoverageState struct {
	ready           bool
	unavailable     map[string]bool
	retirementTrue  bool
	retirementFalse bool
	sharedCanonical int
}

func validateArticleRecoveryCase(
	recoveryCase articleRecoveryCase,
	caseNode *articleJSONNode,
	inventory articleInventoryDocument,
	inventoryByID map[string]articleInventoryBackup,
	keys map[string]articleDeletionKey,
	seenCases map[string]struct{},
	seenOperations map[string]struct{},
	state *articleCoverageState,
) error {
	if !strings.HasPrefix(recoveryCase.CaseID, "article-recovery-") {
		return articleError("invalid_state", "invalid recovery case ID")
	}
	if _, duplicate := seenCases[recoveryCase.CaseID]; duplicate {
		return articleError("invalid_state", "duplicate recovery case ID")
	}
	seenCases[recoveryCase.CaseID] = struct{}{}
	if recoveryCase.CaseID == articleSharedCaseID {
		state.sharedCanonical++
	}
	if _, duplicate := seenOperations[recoveryCase.References.DeletionOperationID]; duplicate {
		return articleError("invalid_state", "duplicate deletion operation binding")
	}
	seenOperations[recoveryCase.References.DeletionOperationID] = struct{}{}
	if err := validateArticleTombstone(recoveryCase.Tombstone); err != nil {
		return err
	}
	if err := validateArticleCaseReferences(recoveryCase, inventoryByID, keys); err != nil {
		return err
	}
	if err := validateArticleReplayStatus(recoveryCase, caseNode, inventory, state); err != nil {
		return err
	}
	return validateArticleRetirementGate(recoveryCase, inventory, state)
}

func validateArticleTombstone(tombstone articleTombstone) error {
	if tombstone.Namespace != recovery.ArticleDeletionNamespaceV1 ||
		tombstone.Stage != "tombstoned" ||
		tombstone.Sequence == 0 || tombstone.ArticleVersion == 0 ||
		tombstone.ArticleID == "" || tombstone.DeletionOperationID == "" {
		return articleError("invalid_state", "invalid tombstone state")
	}
	if _, err := parseArticleMicrosecondTime(tombstone.DeletedAt); err != nil {
		return err
	}
	if !articleSHA256Pattern.MatchString(tombstone.PreviousRecordHash) ||
		!articleHMACPattern.MatchString(tombstone.RecordHMAC) {
		return articleError("invalid_state", "invalid tombstone hash or HMAC fixture")
	}
	return nil
}

func validateArticleCaseReferences(
	recoveryCase articleRecoveryCase,
	inventoryByID map[string]articleInventoryBackup,
	keys map[string]articleDeletionKey,
) error {
	backup, ok := inventoryByID[recoveryCase.References.InventoryBackupID]
	if !ok || backup.BackupRef != recoveryCase.References.BackupRef {
		return articleError("reference_mismatch", "inventory backup reference mismatch")
	}
	keyBinding := recoveryCase.References.KeyID + "\x00" + recoveryCase.References.KeyVersion
	if _, ok := keys[keyBinding]; !ok ||
		recoveryCase.References.KeyID != recoveryCase.Tombstone.KeyID ||
		recoveryCase.References.KeyVersion != recoveryCase.Tombstone.KeyVersion {
		return articleError("reference_mismatch", "Article key reference mismatch")
	}
	if recoveryCase.References.DeletionOperationID !=
		recoveryCase.Tombstone.DeletionOperationID {
		return articleError("reference_mismatch", "deletion operation reference mismatch")
	}
	return nil
}

func validateArticleReplayStatus(
	recoveryCase articleRecoveryCase,
	caseNode *articleJSONNode,
	inventory articleInventoryDocument,
	state *articleCoverageState,
) error {
	decoded, err := recovery.DecodeReplayStatusV1(recoveryCase.ReplayStatus)
	if err != nil {
		return articleError("invalid_state", "strict replay status rejection: %v", err)
	}
	replayNode, _ := caseNode.field("replay_status")
	statusNode, _ := replayNode.field("status")
	statusName := statusNode.value.(string)
	if statusName == "ready" {
		state.ready = true
		if err := validateArticleReadyCase(recoveryCase, decoded, inventory); err != nil {
			return err
		}
		if recoveryCase.CaseID == articleSharedCaseID {
			return validateArticleSharedReadyCase(decoded)
		}
		return nil
	}
	if recoveryCase.CaseID == articleSharedCaseID {
		return articleError("invalid_state", "shared Ready20 fixture is not ready")
	}
	state.unavailable[statusName] = true
	return validateArticleUnavailableCase(recoveryCase, decoded)
}

func validateArticleReadyCase(
	recoveryCase articleRecoveryCase,
	decoded recovery.ReplayStatusV1,
	inventory articleInventoryDocument,
) error {
	if decoded.Ready == nil || decoded.Unavailable != nil ||
		recoveryCase.ExpectedGate.ReplayVariant != "ready" ||
		recoveryCase.References.EvidenceRef == nil ||
		*recoveryCase.References.EvidenceRef != decoded.Ready.EvidenceRef ||
		recoveryCase.References.BackupRef != decoded.Ready.BackupRef {
		return articleError("reference_mismatch", "ready variant binding mismatch")
	}
	if decoded.Ready.InventoryDigest != inventory.InventoryDigest {
		return articleError("digest_mismatch", "Ready inventory reference mismatch")
	}
	digest, err := decoded.Ready.CanonicalDigest()
	if err != nil || digest != decoded.Ready.EvidenceDigest {
		return articleError("invalid_state", "Ready canonical/evidence digest mismatch")
	}
	return nil
}

func validateArticleSharedReadyCase(decoded recovery.ReplayStatusV1) error {
	if decoded.Ready == nil || decoded.Unavailable != nil {
		return articleError("invalid_state", "shared Ready20 fixture is not ready")
	}
	canonical, err := decoded.Ready.CanonicalJSON()
	if err != nil {
		return articleError("invalid_state", "shared Ready20 canonical JSON: %v", err)
	}
	digest, err := decoded.Ready.CanonicalDigest()
	if err != nil ||
		string(canonical) != articleReadyCanonicalV1 ||
		digest != recovery.ReadyFixtureDigestV1 ||
		decoded.Ready.EvidenceDigest != recovery.ReadyFixtureDigestV1 ||
		decoded.Ready.Status != "ready" ||
		decoded.Ready.BackupRef != articleSharedBackupRef ||
		decoded.Ready.EvidenceRef != articleSharedEvidence ||
		decoded.Ready.CompletedAt != articleSharedCompleted {
		return articleError("invalid_state", "shared Ready20 fixture drift")
	}
	return nil
}

func validateArticleUnavailableCase(
	recoveryCase articleRecoveryCase,
	decoded recovery.ReplayStatusV1,
) error {
	if decoded.Unavailable == nil || decoded.Ready != nil ||
		recoveryCase.ExpectedGate.ReplayVariant != "unavailable" ||
		recoveryCase.References.EvidenceRef != nil ||
		recoveryCase.References.BackupRef != decoded.Unavailable.BackupRef {
		return articleError("reference_mismatch", "unavailable variant binding mismatch")
	}
	return nil
}

func validateArticleRetirementGate(
	recoveryCase articleRecoveryCase,
	inventory articleInventoryDocument,
	state *articleCoverageState,
) error {
	deletedAt, err := parseArticleMicrosecondTime(recoveryCase.Tombstone.DeletedAt)
	if err != nil {
		return err
	}
	preconditionsMet := true
	for _, backup := range inventory.Backups {
		createdAt, _ := parseArticleMicrosecondTime(backup.CreatedAt)
		if createdAt.Before(deletedAt) && backup.Status == "recoverable" {
			preconditionsMet = false
		}
	}
	if preconditionsMet != recoveryCase.ExpectedGate.RetirementPreconditionsMet {
		return articleError("unretired_backup", "retirement precondition result mismatch")
	}
	if recoveryCase.ExpectedGate.RealRecoveryAllowed ||
		recoveryCase.ExpectedGate.CleanupAllowed ||
		recoveryCase.ExpectedGate.ReasonCode == "" {
		return articleError("unknown_gate_open", "case gate is not fail closed")
	}
	state.retirementTrue = state.retirementTrue || preconditionsMet
	state.retirementFalse = state.retirementFalse || !preconditionsMet
	return nil
}

var articleForbiddenTombstoneFields = map[string]struct{}{
	"title": {}, "body": {}, "content": {}, "content_digest": {}, "author_id": {},
	"author": {}, "email": {}, "email_address": {}, "retained_email": {},
	"user_name": {}, "display_name": {},
}

var articleEmailPattern = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)

func scanArticleFixtureCanaries(root *articleJSONNode) error {
	if root.kind != articleJSONObject {
		return nil
	}
	for _, field := range root.object {
		if field.name != "negative_cases" {
			if err := scanArticleCanaries(field.value, "$fixtures/"+field.name); err != nil {
				return err
			}
			continue
		}
		if field.value.kind != articleJSONArray {
			continue
		}
		for index, negative := range field.value.array {
			if negative.kind != articleJSONObject {
				continue
			}
			for _, negativeField := range negative.object {
				if negativeField.name == "operand_json" {
					continue
				}
				path := fmt.Sprintf(
					"$fixtures/negative_cases/%d/%s",
					index,
					negativeField.name,
				)
				if err := scanArticleCanaries(negativeField.value, path); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func scanArticleCanaries(node *articleJSONNode, path string) error {
	if node.kind == articleJSONObject {
		for _, field := range node.object {
			if _, forbidden := articleForbiddenTombstoneFields[field.name]; forbidden {
				return articleError("pii_forbidden", "%s contains forbidden field %q", path, field.name)
			}
			if err := scanArticleCanaries(field.value, path+"/"+field.name); err != nil {
				return err
			}
		}
	}
	if node.kind == articleJSONArray {
		for index, value := range node.array {
			if err := scanArticleCanaries(value, fmt.Sprintf("%s/%d", path, index)); err != nil {
				return err
			}
		}
	}
	if node.kind == articleJSONString {
		value := strings.ToLower(node.value.(string))
		if articleEmailPattern.MatchString(value) ||
			strings.Contains(value, "postgres://") ||
			strings.Contains(value, "postgresql://") ||
			strings.Contains(value, "password=") ||
			strings.Contains(value, "credential=") ||
			strings.Contains(value, "begin private key") {
			return articleError("pii_forbidden", "%s contains a sensitive canary", path)
		}
	}
	return nil
}

func testArticleFixtureMutations(t *testing.T, inventoryRaw []byte, fixturesRaw []byte) {
	t.Helper()
	inventoryNode := mustParseArticleJSON(t, inventoryRaw)
	fixturesNode := mustParseArticleJSON(t, fixturesRaw)
	var fixtures articleDeletionDocument
	if err := json.Unmarshal(fixturesRaw, &fixtures); err != nil {
		t.Fatalf("decode mutation catalog: %v", err)
	}
	for _, mutation := range fixtures.NegativeCases {
		mutation := mutation
		t.Run(mutation.CaseID, func(t *testing.T) {
			mutatedInventory := inventoryNode.clone()
			mutatedFixtures := fixturesNode.clone()
			target := mutatedInventory
			if mutation.TargetFile == articleFixturesFile {
				target = mutatedFixtures
			}
			if err := applyArticleMutation(target, mutation); err != nil {
				t.Fatalf("apply mutation: %v", err)
			}
			inventoryBytes, _ := mutatedInventory.marshalJSON()
			fixturesBytes, _ := mutatedFixtures.marshalJSON()
			err := validateArticleArtifactPair(inventoryBytes, fixturesBytes)
			if err == nil || articleErrorCode(err) != mutation.ExpectedError {
				t.Fatalf("validation error = %v, want code %q", err, mutation.ExpectedError)
			}
		})
	}
}

func mustParseArticleJSON(t *testing.T, raw []byte) *articleJSONNode {
	t.Helper()
	node, err := parseArticleJSONDocument(raw)
	if err != nil {
		t.Fatalf("parse positive JSON: %v", err)
	}
	return node
}

func applyArticleMutation(root *articleJSONNode, mutation articleNegativeMutation) error {
	switch mutation.Mutation {
	case "remove_field":
		return removeArticleJSONValue(root, mutation.TargetPath)
	case "replace_value":
		value, err := parseArticleJSONDocument([]byte(mutation.OperandJSON))
		if err != nil {
			return err
		}
		return replaceArticleJSONValue(root, mutation.TargetPath, value)
	case "add_field":
		return addArticleJSONField(root, mutation.TargetPath, mutation.OperandJSON)
	case "duplicate_key":
		return duplicateArticleJSONField(root, mutation.TargetPath, mutation.OperandJSON)
	case "reorder_fields":
		return reorderArticleJSONFields(root, mutation.TargetPath, mutation.OperandJSON)
	default:
		return articleError("invalid_state", "unsupported mutation %q", mutation.Mutation)
	}
}

func removeArticleJSONValue(root *articleJSONNode, pointer string) error {
	parent, token, err := locateArticleJSONParent(root, pointer)
	if err != nil {
		return err
	}
	if parent.kind == articleJSONObject {
		for index, field := range parent.object {
			if field.name == token {
				parent.object = append(parent.object[:index], parent.object[index+1:]...)
				return nil
			}
		}
	}
	return articleError("reference_mismatch", "remove target %q not found", pointer)
}

func replaceArticleJSONValue(root *articleJSONNode, pointer string, value *articleJSONNode) error {
	parent, token, err := locateArticleJSONParent(root, pointer)
	if err != nil {
		return err
	}
	if parent.kind == articleJSONObject {
		for index, field := range parent.object {
			if field.name == token {
				parent.object[index].value = value
				return nil
			}
		}
	}
	if parent.kind == articleJSONArray {
		index, err := strconv.Atoi(token)
		if err == nil && index >= 0 && index < len(parent.array) {
			parent.array[index] = value
			return nil
		}
	}
	return articleError("reference_mismatch", "replace target %q not found", pointer)
}

func addArticleJSONField(root *articleJSONNode, pointer string, operand string) error {
	target, err := locateArticleJSONValue(root, pointer)
	if err != nil {
		return err
	}
	addition, err := parseArticleJSONDocument([]byte(operand))
	if err != nil {
		return err
	}
	if target.kind != articleJSONObject ||
		addition.kind != articleJSONObject ||
		len(addition.object) != 1 {
		return articleError("invalid_state", "add_field requires one object member")
	}
	target.object = append(target.object, addition.object[0])
	return nil
}

func duplicateArticleJSONField(root *articleJSONNode, pointer string, operand string) error {
	parent, token, err := locateArticleJSONParent(root, pointer)
	if err != nil {
		return err
	}
	if parent.kind != articleJSONObject {
		return articleError("invalid_state", "duplicate_key target is not an object field")
	}
	value, err := parseArticleJSONDocument([]byte(operand))
	if err != nil {
		return err
	}
	if !parent.hasField(token) {
		return articleError("reference_mismatch", "duplicate target %q not found", pointer)
	}
	parent.object = append(parent.object, articleJSONField{name: token, value: value})
	return nil
}

func reorderArticleJSONFields(root *articleJSONNode, pointer string, operand string) error {
	target, err := locateArticleJSONValue(root, pointer)
	if err != nil {
		return err
	}
	if target.kind != articleJSONObject {
		return articleError("invalid_state", "reorder target is not an object")
	}
	var order []string
	if err := json.Unmarshal([]byte(operand), &order); err != nil {
		return articleError("invalid_json", "decode reorder operand: %v", err)
	}
	fields := make(map[string]articleJSONField, len(target.object))
	for _, field := range target.object {
		fields[field.name] = field
	}
	if len(order) != len(fields) {
		return articleError("invalid_state", "reorder operand has the wrong field count")
	}
	reordered := make([]articleJSONField, 0, len(order))
	for _, name := range order {
		field, ok := fields[name]
		if !ok {
			return articleError("reference_mismatch", "reorder field %q not found", name)
		}
		reordered = append(reordered, field)
		delete(fields, name)
	}
	target.object = reordered
	return nil
}

func locateArticleJSONValue(root *articleJSONNode, pointer string) (*articleJSONNode, error) {
	if pointer == "" {
		return root, nil
	}
	tokens, err := parseArticleJSONPointer(pointer)
	if err != nil {
		return nil, err
	}
	current := root
	for _, token := range tokens {
		if current.kind == articleJSONObject {
			next, ok := current.field(token)
			if !ok {
				return nil, articleError("reference_mismatch", "pointer %q not found", pointer)
			}
			current = next
			continue
		}
		if current.kind == articleJSONArray {
			index, err := strconv.Atoi(token)
			if err != nil || index < 0 || index >= len(current.array) {
				return nil, articleError("reference_mismatch", "pointer %q not found", pointer)
			}
			current = current.array[index]
			continue
		}
		return nil, articleError("reference_mismatch", "pointer %q crosses a scalar", pointer)
	}
	return current, nil
}

func locateArticleJSONParent(
	root *articleJSONNode,
	pointer string,
) (*articleJSONNode, string, error) {
	tokens, err := parseArticleJSONPointer(pointer)
	if err != nil || len(tokens) == 0 {
		return nil, "", articleError("reference_mismatch", "pointer %q has no parent", pointer)
	}
	parentPointer := ""
	if len(tokens) > 1 {
		escaped := make([]string, len(tokens)-1)
		for index, token := range tokens[:len(tokens)-1] {
			escaped[index] = escapeArticlePointer(token)
		}
		parentPointer = "/" + strings.Join(escaped, "/")
	}
	parent, err := locateArticleJSONValue(root, parentPointer)
	return parent, tokens[len(tokens)-1], err
}

func parseArticleJSONPointer(pointer string) ([]string, error) {
	if !strings.HasPrefix(pointer, "/") {
		return nil, articleError("reference_mismatch", "invalid RFC 6901 pointer %q", pointer)
	}
	rawTokens := strings.Split(pointer[1:], "/")
	tokens := make([]string, len(rawTokens))
	for index, token := range rawTokens {
		token = strings.ReplaceAll(token, "~1", "/")
		token = strings.ReplaceAll(token, "~0", "~")
		if strings.Contains(token, "~") {
			return nil, articleError("reference_mismatch", "invalid pointer escape")
		}
		tokens[index] = token
	}
	return tokens, nil
}

func escapeArticlePointer(value string) string {
	value = strings.ReplaceAll(value, "~", "~0")
	return strings.ReplaceAll(value, "/", "~1")
}

func testArticleFixtureTrailingData(t *testing.T, inventory []byte, fixtures []byte) {
	t.Helper()
	tests := []struct {
		name      string
		inventory []byte
		fixtures  []byte
	}{
		{
			name:      "inventory",
			inventory: append(append([]byte(nil), inventory...), []byte(` {}`)...),
			fixtures:  fixtures,
		},
		{
			name:      "fixtures",
			inventory: inventory,
			fixtures:  append(append([]byte(nil), fixtures...), []byte(` {}`)...),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateArticleArtifactPair(test.inventory, test.fixtures)
			if articleErrorCode(err) != "invalid_json" {
				t.Fatalf("validation error = %v, want invalid_json", err)
			}
		})
	}
}

func testArticleFixtureInvalidUTF8(t *testing.T, inventory []byte, fixtures []byte) {
	t.Helper()
	tests := []struct {
		name      string
		inventory []byte
		fixtures  []byte
	}{
		{
			name: "inventory quoted string",
			inventory: bytes.Replace(
				inventory,
				[]byte("synthetic-owner-recovery-review"),
				[]byte("synthetic-owner-\xff-review"),
				1,
			),
			fixtures: fixtures,
		},
		{
			name:      "fixtures quoted string",
			inventory: inventory,
			fixtures: bytes.Replace(
				fixtures,
				[]byte("Shared synthetic Ready20 canonical fixture."),
				[]byte("Shared synthetic Ready20 \xff canonical fixture."),
				1,
			),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateArticleArtifactPair(test.inventory, test.fixtures)
			if articleErrorCode(err) != "invalid_json" {
				t.Fatalf("validation error = %v, want invalid_json", err)
			}
		})
	}
}

func testArticleFixtureSensitiveCanaries(t *testing.T, inventory []byte, fixtures []byte) {
	t.Helper()
	tests := []struct {
		name      string
		inventory []byte
		fixtures  []byte
	}{
		{
			name: "inventory retirement reference DSN",
			inventory: mutateArticleJSONForTest(
				t,
				inventory,
				"/backups/0/retirement_owner_ref",
				`"postgres://synthetic.invalid/recovery"`,
			),
			fixtures: fixtures,
		},
		{
			name:      "key material credential",
			inventory: inventory,
			fixtures: mutateArticleJSONForTest(
				t,
				fixtures,
				"/keys/0/synthetic_material",
				`"SYNTHETIC-TEST-ONLY-credential=synthetic"`,
			),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateArticleArtifactPair(test.inventory, test.fixtures)
			if articleErrorCode(err) != "pii_forbidden" {
				t.Fatalf("validation error = %v, want pii_forbidden", err)
			}
		})
	}
}

func testArticleSharedCanonicalFixture(t *testing.T, inventory []byte, fixtures []byte) {
	t.Helper()
	t.Run("renamed is missing", func(t *testing.T) {
		root := mustParseArticleJSON(t, fixtures)
		positiveCases, _ := root.field("positive_cases")
		caseID, _ := positiveCases.array[0].field("case_id")
		caseID.value = "article-recovery-ready-renamed"
		negativeCases, _ := root.field("negative_cases")
		for _, negative := range negativeCases.array {
			baseCaseID, _ := negative.field("base_case_id")
			if baseCaseID.value == "article-recovery-ready-canonical" {
				baseCaseID.value = "article-recovery-ready-renamed"
			}
		}
		mutated, err := root.marshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		assertArticleFixtureErrorCode(t, inventory, mutated, "invalid_state")
	})

	t.Run("duplicate", func(t *testing.T) {
		root := mustParseArticleJSON(t, fixtures)
		positiveCases, _ := root.field("positive_cases")
		positiveCases.array = append(positiveCases.array, positiveCases.array[0].clone())
		mutated, err := root.marshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		assertArticleFixtureErrorCode(t, inventory, mutated, "invalid_state")
	})

	t.Run("self consistent canonical drift", func(t *testing.T) {
		root := mustParseArticleJSON(t, fixtures)
		positiveCases, _ := root.field("positive_cases")
		replayStatus, _ := positiveCases.array[0].field("replay_status")
		replayRaw, err := replayStatus.marshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := recovery.DecodeReplayStatusV1(replayRaw)
		if err != nil || decoded.Ready == nil {
			t.Fatalf("decode shared Ready20: %v", err)
		}
		drifted := *decoded.Ready
		drifted.RunID = "run-drift"
		drifted.EvidenceDigest, err = drifted.CanonicalDigest()
		if err != nil {
			t.Fatal(err)
		}
		runID, _ := replayStatus.field("run_id")
		runID.value = drifted.RunID
		evidenceDigest, _ := replayStatus.field("evidence_digest")
		evidenceDigest.value = drifted.EvidenceDigest
		mutated, err := root.marshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		assertArticleFixtureErrorCode(t, inventory, mutated, "invalid_state")
	})
}

func mutateArticleJSONForTest(t *testing.T, raw []byte, pointer string, operand string) []byte {
	t.Helper()
	root := mustParseArticleJSON(t, raw)
	value, err := parseArticleJSONDocument([]byte(operand))
	if err != nil {
		t.Fatal(err)
	}
	if err := replaceArticleJSONValue(root, pointer, value); err != nil {
		t.Fatal(err)
	}
	mutated, err := root.marshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	return mutated
}

func assertArticleFixtureErrorCode(
	t *testing.T,
	inventory []byte,
	fixtures []byte,
	expected string,
) {
	t.Helper()
	err := validateArticleArtifactPair(inventory, fixtures)
	if articleErrorCode(err) != expected {
		t.Fatalf("validation error = %v, want %s", err, expected)
	}
}

func testArticleRootDiscovery(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	required := []string{
		"go.mod",
		filepath.Join("docs", "runbooks", articleRunbookFile),
		filepath.Join("testdata", "recovery", articleInventoryFile),
		filepath.Join("testdata", "recovery", articleFixturesFile),
	}
	for _, name := range required {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("synthetic\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	packageDir := filepath.Join(root, "test", "integration", "recovery")
	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, start := range []string{root, packageDir} {
		found, ok := findArticleModuleRoot(start)
		if !ok || found != root {
			t.Fatalf("findArticleModuleRoot(%q) = %q, %v; want %q, true", start, found, ok, root)
		}
	}
}

var articleRunbookHeadings = []string{
	"## 适用范围与非声明",
	"## Synthetic-only 数据规则",
	"## Entry Gate",
	"## Inventory、tombstone、key 与 evidence 对账",
	"## Retirement evidence 与人工授权",
	"## Fail-closed 条件",
	"## Cleanup 责任",
	"## Unknown：retention / RPO / RTO",
	"## 自动检查与人工核项",
	"## 关闭边界",
}

var articleRunbookRequiredText = []string{
	"只使用 synthetic-only 输入",
	"真实凭据、PII 和生产数据禁止",
	"unknown retention / RPO / RTO 阻止真实恢复和灾备演练",
	"缺少 inventory、key、tombstone 或 evidence",
	"引用不一致",
	"旧可恢复备份未退役",
	"版本未知",
	"fail closed",
	"cleanup owner、范围和授权未确认时不得清理",
	"不构成灾备演练",
	"不构成真实恢复证明",
	"不构成 retirement 授权",
	"不构成 production readiness",
	"真实恢复 entry gate",
	"retirement evidence 的真实性与批准",
	"cleanup owner、允许范围和回滚边界",
	"retention / RPO / RTO 决定",
	"真实运行窗口",
	"最终 readiness",
}

func validateArticleRunbook(raw []byte) error {
	if bytes.HasPrefix(raw, []byte{0xef, 0xbb, 0xbf}) {
		return articleError("invalid_state", "runbook must not contain a BOM")
	}
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	if len(lines) == 0 || !strings.HasPrefix(lines[0], "# ") ||
		strings.HasPrefix(lines[0], "## ") {
		return articleError("invalid_order", "runbook must start with one H1")
	}
	var headings []string
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			headings = append(headings, line)
		}
	}
	if len(headings) != len(articleRunbookHeadings) {
		return articleError("missing_field", "runbook H2 count = %d", len(headings))
	}
	for index, heading := range articleRunbookHeadings {
		if headings[index] != heading {
			return articleError("invalid_order", "runbook H2 %d = %q", index, headings[index])
		}
	}
	text := string(raw)
	for _, required := range articleRunbookRequiredText {
		if strings.Count(text, required) == 0 {
			return articleError("missing_field", "runbook missing declaration %q", required)
		}
	}
	return nil
}
