//go:build integration

package identity

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"han/internal/audit"
	identitypkg "han/internal/identity"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maintenanceTestBinaryEnv   = "HAN_S105_TEST_BINARY"
	maintenanceTestEvidenceEnv = "HAN_S105_TEST_EVIDENCE_DIR"
	maintenanceChildWatchdog   = 10 * time.Second
	maintenanceEvidenceMode    = 0o600
	maintenanceWireTime        = "2006-01-02T15:04:05.000000Z"
	maintenanceWireDomain      = "Han/identity-auth/S10.5/requeue-approval/v1\x00"
	maintenanceWireAction      = "identity.password_reset.requeue"
	maintenancePinVersion      = 180006
)

type maintenanceCLISuite struct {
	binary, evidence, repo, subject string
	private                         ed25519.PrivateKey
	trust                           []byte
	mu                              sync.Mutex
	sequence                        int
	redactions                      []string // Sensitive values stay in memory and never enter artifacts.
}

type maintenanceWirePayload struct {
	Version         string                                   `json:"version"`
	DeploymentID    string                                   `json:"deployment_id"`
	Epoch           string                                   `json:"epoch"`
	RequestID       string                                   `json:"request_id"`
	Action          string                                   `json:"action"`
	UserID          string                                   `json:"user_id"`
	OperationID     string                                   `json:"operation_id"`
	ApproverKeyID   string                                   `json:"approver_key_id"`
	ExecutorSubject string                                   `json:"executor_subject"`
	IssuedAt        int64                                    `json:"issued_at"`
	ExpiresAt       int64                                    `json:"expires_at"`
	Reason          string                                   `json:"reason"`
	EvidenceRef     string                                   `json:"evidence_ref"`
	Before          identitypkg.PasswordResetRequeueSnapshot `json:"before"`
}

type maintenanceCLIObservation struct {
	Argv                []string `json:"argv"`
	ExitCode            int      `json:"exit_code"`
	Stdout              string   `json:"stdout"`
	Stderr              string   `json:"stderr"`
	ElapsedMilliseconds int64    `json:"elapsed_milliseconds"`
	WatchdogKilled      bool     `json:"watchdog_killed"`
	WaitJoined          bool     `json:"wait_joined"`
	WriterOpenUntilExit bool     `json:"writer_open_until_exit"`
	WriterJoined        bool     `json:"writer_joined"`
	DeliveredBytes      int      `json:"delivered_bytes"`
	InputEOF            bool     `json:"input_eof"`
}

// TestPasswordResetMaintenancePostgresCLI builds a retained real cmd binary by
// default. Optional harness overrides do not add product configuration keys.
func TestPasswordResetMaintenancePostgresCLI(t *testing.T) {
	suite := newMaintenanceCLISuite(t)
	t.Logf("retained evidence=%s", suite.evidence)
	t.Run("I-S105-01_23_argv_and_server_control", suite.testArgv)
	t.Run("I-S105-02_05_pre_dsn_config", suite.testConfig)
	t.Run("I-S105-03_04_07_08_09_10_pre_dsn_authority", suite.testAuthority)
	t.Run("I-S105-25_new_invocation_revoke", suite.testRevocation)
	t.Run("I-S105-06_half_no_eof", func(t *testing.T) { suite.testNoEOF(t, "half", false) })
	t.Run("I-S105-06_complete_no_eof", func(t *testing.T) { suite.testNoEOF(t, "complete", false) })
	t.Run("I-S105-06_blocked_producer", func(t *testing.T) { suite.testNoEOF(t, "blocked", false) })
	t.Run("I-S105-06_positive_bytes_no_eof", func(t *testing.T) { suite.testNoEOF(t, "positive", false) })
	fixture := newMaintenancePGFixture(t, suite)
	fixture.runCases(t)
}

func newMaintenanceCLISuite(t *testing.T) *maintenanceCLISuite {
	t.Helper()
	repo, err := filepath.Abs(filepath.Join("..", "..", ".."))
	maintenanceMust(t, err, "repository path")
	evidence, err := os.MkdirTemp(os.Getenv(maintenanceTestEvidenceEnv), "han-s105-cli-")
	maintenanceMust(t, err, "retained evidence directory")
	suite := &maintenanceCLISuite{repo: repo, evidence: evidence}
	suite.recordHarness(t)
	suite.binary = os.Getenv(maintenanceTestBinaryEnv)
	if suite.binary == "" {
		suite.binary = filepath.Join(evidence, "han-server.exe")
		suite.build(t)
	} else if !filepath.IsAbs(suite.binary) {
		t.Fatal("test binary override must be absolute")
	}
	raw, err := os.ReadFile(suite.binary)
	maintenanceMust(t, err, "built binary read")
	suite.writeJSON(t, "binary.json", map[string]any{"path": suite.binary, "sha256": maintenanceDigest(raw), "default_build": os.Getenv(maintenanceTestBinaryEnv) == "", "child_race_instrumentation": "see actual build argv; harness race does not imply child race"})
	if runtime.GOOS != "windows" {
		t.Fatal("actual current-SID CLI/PG carrier requires Windows; no OS seam substitutes for B")
	}
	current, err := user.Current()
	maintenanceMust(t, err, "actual process owner")
	suite.subject = "windows-sid:" + current.Uid
	public, private, err := ed25519.GenerateKey(rand.Reader)
	maintenanceMust(t, err, "in-memory synthetic signing key")
	suite.private = private
	now := time.Now().Unix()
	approver := "windows-sid:S-1-5-21-424242-424242-424242-424242"
	if approver == suite.subject {
		approver = "windows-sid:S-1-0-0"
	}
	suite.trust = maintenanceJSON(t, map[string]any{"version": "han-password-reset-maintenance-trust/v1", "deployment_id": "test-deployment", "epoch": "test-epoch", "approver_keys": []any{map[string]any{"key_id": "test-approver", "public_key": base64.RawURLEncoding.EncodeToString(public), "subject": approver, "action": maintenanceWireAction, "not_before": now - 60, "expires_at": now + 7200}}, "executors": []any{map[string]any{"subject": suite.subject, "action": maintenanceWireAction, "not_before": now - 60, "expires_at": now + 7200}}})
	return suite
}

func (suite *maintenanceCLISuite) build(t *testing.T) {
	t.Helper()
	before := maintenanceSourceHashes(t, suite.repo)
	suite.writeJSON(t, "source.before-build.json", before)
	args := []string{"build", "-mod=readonly", "-buildvcs=false", "-trimpath", "-o", suite.binary, "./cmd/han-server"}
	command := exec.Command("go", args...)
	command.Dir = suite.repo
	command.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "GOTELEMETRY=off")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	command = exec.CommandContext(ctx, command.Path, args...)
	command.Dir = suite.repo
	command.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "GOTELEMETRY=off")
	output, err := command.CombinedOutput()
	suite.writeBytes(t, "build.stdout-stderr.txt", output)
	suite.writeJSON(t, "build-command.json", map[string]any{"executable": command.Path, "argv": args, "cwd": suite.repo, "exit_ok": err == nil, "watchdog": ctx.Err() != nil})
	after := maintenanceSourceHashes(t, suite.repo)
	suite.writeJSON(t, "source.after-build.json", after)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("source changed during product build; source-to-binary proof unavailable")
	}
	maintenanceMust(t, err, "real product build; not a behavioral failure")
}

func (suite *maintenanceCLISuite) recordHarness(t *testing.T) {
	t.Helper()
	executable, err := os.Executable()
	maintenanceMust(t, err, "parent harness executable path")
	raw, err := os.ReadFile(executable)
	maintenanceMust(t, err, "parent harness binary digest")
	raceSetting := "not-present; use retained exact go test argv"
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "-race" {
				raceSetting = setting.Value
			}
		}
	}
	suite.writeJSON(t, "harness.json", map[string]any{"executable": executable, "sha256": maintenanceDigest(raw), "go_version": runtime.Version(), "GOOS": runtime.GOOS, "GOARCH": runtime.GOARCH, "argv": os.Args[1:], "race_build_setting": raceSetting})
}

func maintenanceSourceHashes(t *testing.T, repo string) map[string]string {
	t.Helper()
	result := map[string]string{}
	for _, directory := range []string{"cmd", "internal", "migrations", "test"} {
		err := filepath.WalkDir(filepath.Join(repo, directory), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, ".sql") {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			relative, err := filepath.Rel(repo, path)
			if err != nil {
				return err
			}
			result[filepath.ToSlash(relative)] = maintenanceDigest(raw)
			return nil
		})
		maintenanceMust(t, err, "source digest walk")
	}
	for _, name := range []string{"go.mod", "go.sum"} {
		raw, err := os.ReadFile(filepath.Join(repo, name))
		maintenanceMust(t, err, "module digest")
		result[name] = maintenanceDigest(raw)
	}
	return result
}

func maintenanceDigest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func maintenanceJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	maintenanceMust(t, err, "encode bounded evidence")
	return raw
}
func maintenanceMust(t *testing.T, err error, stage string) {
	t.Helper()
	if err != nil {
		var pg *pgconn.PgError
		state := ""
		if errors.As(err, &pg) {
			state = pg.Code
		}
		t.Fatalf("%s failed (type=%T sqlstate=%s); cause redacted", stage, err, state)
	}
}
func (suite *maintenanceCLISuite) writeJSON(t *testing.T, name string, value any) {
	t.Helper()
	suite.writeBytes(t, name, append(maintenanceJSON(t, value), '\n'))
}
func (suite *maintenanceCLISuite) writeBytes(t *testing.T, name string, raw []byte) {
	t.Helper()
	maintenanceMust(t, os.WriteFile(filepath.Join(suite.evidence, name), raw, maintenanceEvidenceMode), "retained evidence write")
}

func (suite *maintenanceCLISuite) config() map[string]string {
	return map[string]string{"HAN_PASSWORD_RESET_MAINTENANCE_TRUST": string(suite.trust), "HAN_PASSWORD_RESET_MAINTENANCE_MAX_TTL": "15m", "HAN_PASSWORD_RESET_MAINTENANCE_OPERATION_BUDGET": "1s", "HAN_PASSWORD_RESET_WORKER_CONNECTION_BUDGET": "1", "HAN_SHUTDOWN_BUDGET": "1s", "HAN_RECOVERY_MODE": "normal"}
}

func maintenanceChildEnvironment(config map[string]string) []string {
	var result []string
	for _, name := range []string{"SystemRoot", "WINDIR", "PATH", "TEMP", "TMP"} {
		if value, present := os.LookupEnv(name); present {
			result = append(result, name+"="+value)
		}
	}
	for name, value := range config {
		result = append(result, name+"="+value)
	}
	return result
}

func (suite *maintenanceCLISuite) observe(t *testing.T, config map[string]string, args []string, input []byte, eof bool) maintenanceCLIObservation {
	t.Helper()
	reader, writer, err := os.Pipe()
	maintenanceMust(t, err, "owned stdin pipe")
	cmd := exec.Command(suite.binary, args...)
	cmd.Env = maintenanceChildEnvironment(config)
	cmd.Stdin = reader
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	start := time.Now()
	err = cmd.Start()
	if err != nil {
		_ = reader.Close()
		_ = writer.Close()
		maintenanceMust(t, err, "real binary start")
	}
	maintenanceMust(t, reader.Close(), "parent read duplicate close")
	written := make(chan int, 1)
	go func() {
		n, _ := writer.Write(input)
		if eof {
			_ = writer.Close()
		}
		written <- n
	}()
	waitErr, killed := maintenanceWaitChild(t, cmd)
	if !eof {
		maintenanceMust(t, writer.Close(), "owned writer close after exit")
	}
	var delivered int
	select {
	case delivered = <-written:
	case <-time.After(time.Second):
		t.Fatal("owned writer goroutine did not join")
	}
	got := maintenanceCLIObservation{Argv: args, ExitCode: maintenanceChildExit(t, waitErr), Stdout: stdout.String(), Stderr: stderr.String(), ElapsedMilliseconds: time.Since(start).Milliseconds(), WatchdogKilled: killed, WaitJoined: true, WriterOpenUntilExit: !eof, WriterJoined: true, DeliveredBytes: delivered, InputEOF: eof}
	suite.retainObservation(t, got)
	if killed {
		t.Fatal("watchdog killed child; never accepted as Case success")
	}
	return got
}

func (suite *maintenanceCLISuite) retainObservation(t *testing.T, got maintenanceCLIObservation) {
	t.Helper()
	leaked := false
	for _, secret := range suite.redactions {
		if secret != "" {
			if strings.Contains(got.Stdout, secret) || strings.Contains(got.Stderr, secret) {
				leaked = true
			}
			got.Stdout = strings.ReplaceAll(got.Stdout, secret, "[REDACTED]")
			got.Stderr = strings.ReplaceAll(got.Stderr, secret, "[REDACTED]")
		}
	}
	suite.mu.Lock()
	suite.sequence++
	sequence := suite.sequence
	suite.mu.Unlock()
	stem := fmt.Sprintf("%04d", sequence)
	suite.writeBytes(t, stem+".stdout.txt", []byte(got.Stdout))
	suite.writeBytes(t, stem+".stderr.txt", []byte(got.Stderr))
	suite.writeJSON(t, stem+".observation.json", got)
	if leaked {
		t.Fatal("child output leaked sensitive configuration; retained outputs redacted")
	}
}

func maintenanceWaitChild(t *testing.T, cmd *exec.Cmd) (error, bool) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	timer := time.NewTimer(maintenanceChildWatchdog)
	defer timer.Stop()
	select {
	case err := <-done:
		return err, false
	case <-timer.C:
		if err := cmd.Process.Kill(); err != nil {
			t.Errorf("watchdog could not stop owned child")
		}
	}
	join := time.NewTimer(maintenanceChildWatchdog)
	defer join.Stop()
	select {
	case err := <-done:
		return err, true
	case <-join.C:
		t.Fatal("child Wait did not join after failed watchdog")
		return nil, true
	}
}
func maintenanceChildExit(t *testing.T, err error) int {
	t.Helper()
	if err == nil {
		return 0
	}
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatal("child Wait infrastructure failure")
	}
	return exit.ExitCode()
}
func maintenanceExpect(t *testing.T, got maintenanceCLIObservation, exit int, code string) {
	t.Helper()
	if got.ExitCode != exit || got.Stdout != "" || got.Stderr != code+"\n" {
		t.Fatalf("CLI tuple: got exit=%d stdout=%q stderr=%q; want exit=%d code=%s", got.ExitCode, got.Stdout, got.Stderr, exit, code)
	}
}

func maintenanceUUID(t *testing.T) string {
	t.Helper()
	var value [16]byte
	_, err := rand.Read(value[:])
	maintenanceMust(t, err, "synthetic UUID")
	value[6] = value[6]&15 | 64
	value[8] = value[8]&63 | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", value[:4], value[4:6], value[6:8], value[8:10], value[10:])
}
func (suite *maintenanceCLISuite) payload(t *testing.T, before identitypkg.PasswordResetRequeueSnapshot, userID, operationID string) maintenanceWirePayload {
	now := time.Now().Unix()
	return maintenanceWirePayload{Version: "han-password-reset-requeue/v1", DeploymentID: "test-deployment", Epoch: "test-epoch", RequestID: maintenanceUUID(t), Action: maintenanceWireAction, UserID: userID, OperationID: operationID, ApproverKeyID: "test-approver", ExecutorSubject: suite.subject, IssuedAt: now - 1, ExpiresAt: now + 600, Reason: "synthetic maintenance integration approval", EvidenceRef: "test:retained-local-evidence", Before: before}
}
func (suite *maintenanceCLISuite) envelope(t *testing.T, payload []byte, domain string) []byte {
	signature := ed25519.Sign(suite.private, append([]byte(domain), payload...))
	return maintenanceJSON(t, struct {
		Payload   string `json:"payload"`
		Signature string `json:"approval_signature"`
	}{base64.RawURLEncoding.EncodeToString(payload), base64.RawURLEncoding.EncodeToString(signature)})
}
func (suite *maintenanceCLISuite) syntheticPayload(t *testing.T) maintenanceWirePayload {
	operation := maintenanceUUID(t)
	timestamp := time.Now().UTC().Add(-time.Minute).Truncate(time.Microsecond).Format(maintenanceWireTime)
	before := identitypkg.PasswordResetRequeueSnapshot{UserStatus: "active", UserAuthGeneration: 0, UserExecutionGeneration: 1, UserCurrentOperationID: &operation, CredentialMode: "permanent", UserUpdatedAt: timestamp, OperationState: "prepared", DeliveryState: "not_started", SettlementState: "pending", OperationExecutionGeneration: 1, TemporaryCredentialID: maintenanceUUID(t), OperationUpdatedAt: timestamp, JobStatus: "dead_letter", AttemptCount: 2, NextAttemptAt: timestamp, JobCreatedAt: timestamp, JobUpdatedAt: timestamp}
	return suite.payload(t, before, maintenanceUUID(t), operation)
}

func (suite *maintenanceCLISuite) testArgv(t *testing.T) {
	maintenanceExpect(t, suite.observe(t, nil, nil, nil, false), 1, "han-server startup failed")
	for _, args := range [][]string{{"maintenance"}, {"maintenance", "password-reset", "settle"}, {"maintenance", "password-reset", "retry", "--unexpected"}, {"unknown"}} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			maintenanceExpect(t, suite.observe(t, nil, args, nil, false), 2, "maintenance.invalid_request")
		})
	}
}
func (suite *maintenanceCLISuite) testConfig(t *testing.T) {
	for _, name := range []string{"HAN_PASSWORD_RESET_MAINTENANCE_TRUST", "HAN_PASSWORD_RESET_MAINTENANCE_MAX_TTL", "HAN_PASSWORD_RESET_MAINTENANCE_OPERATION_BUDGET", "HAN_PASSWORD_RESET_WORKER_CONNECTION_BUDGET", "HAN_SHUTDOWN_BUDGET", "HAN_RECOVERY_MODE"} {
		for _, variant := range []string{"missing", "blank", "invalid"} {
			t.Run(name+"_"+variant, func(t *testing.T) {
				config := suite.config()
				if variant == "missing" {
					delete(config, name)
				} else if variant == "blank" {
					config[name] = " "
				} else {
					config[name] = "invalid"
				}
				code := "maintenance.invalid_config"
				if name == "HAN_RECOVERY_MODE" {
					code = "maintenance.mode_conflict"
				}
				maintenanceExpect(t, suite.observe(t, config, []string{"maintenance", "password-reset", "retry"}, nil, false), 2, code)
			})
		}
	}
	for _, value := range []string{"0", "-1", "01", "2147483648"} {
		t.Run("connection_"+value, func(t *testing.T) {
			config := suite.config()
			config["HAN_PASSWORD_RESET_WORKER_CONNECTION_BUDGET"] = value
			maintenanceExpect(t, suite.observe(t, config, []string{"maintenance", "password-reset", "retry"}, nil, false), 2, "maintenance.invalid_config")
		})
	}
}
func (suite *maintenanceCLISuite) testAuthority(t *testing.T) {
	for _, kind := range []string{"valid_before_missing_dsn", "wrong_subject", "wrong_domain", "expired", "future", "duplicate_payload", "padding_envelope"} {
		t.Run(kind, func(t *testing.T) {
			payload := suite.syntheticPayload(t)
			config := suite.config()
			domain := maintenanceWireDomain
			code := "maintenance.forbidden"
			exit := 3
			switch kind {
			case "valid_before_missing_dsn":
				code = "maintenance.invalid_config"
				exit = 2
			case "wrong_subject":
				payload.ExecutorSubject = "windows-sid:S-1-0-0"
				config["USERNAME"] = "irrelevant-user"
			case "wrong_domain":
				domain = "wrong\x00"
			case "expired":
				payload.IssuedAt = time.Now().Unix() - 20
				payload.ExpiresAt = time.Now().Unix() - 1
			case "future":
				payload.IssuedAt = time.Now().Unix() + 60
				payload.ExpiresAt = payload.IssuedAt + 60
			}
			raw := maintenanceJSON(t, payload)
			if kind == "duplicate_payload" {
				raw = bytes.Replace(raw, []byte(`"version":`), []byte(`"version":"han-password-reset-requeue/v1","version":`), 1)
				code = "maintenance.invalid_request"
				exit = 2
			}
			input := suite.envelope(t, raw, domain)
			if kind == "padding_envelope" {
				input = append(input, '\n')
				code = "maintenance.invalid_request"
				exit = 2
			}
			maintenanceExpect(t, suite.observe(t, config, []string{"maintenance", "password-reset", "retry"}, input, true), exit, code)
		})
	}
}
func (suite *maintenanceCLISuite) testNoEOF(t *testing.T, kind string, eof bool) {
	input := suite.envelope(t, maintenanceJSON(t, suite.syntheticPayload(t)), maintenanceWireDomain)
	switch kind {
	case "half":
		input = input[:len(input)/2]
	case "blocked":
		input = bytes.Repeat([]byte{'x'}, 128*1024)
	case "positive":
		input = []byte{'{'}
	}
	got := suite.observe(t, suite.config(), []string{"maintenance", "password-reset", "retry"}, input, eof)
	if kind == "blocked" {
		if got.DeliveredBytes >= len(input) {
			t.Fatal("producer did not exercise a partial blocked write")
		}
		maintenanceExpect(t, got, 2, "maintenance.invalid_request")
	} else {
		maintenanceExpect(t, got, 6, "maintenance.failed")
	}
	if !got.WriterOpenUntilExit || !got.WriterJoined {
		t.Fatal("parent EOF or unjoined writer invalidated no-EOF evidence")
	}
	// Positive-byte B observes bounded cancellation. Exact Peek->Read ordering is
	// the Root U barrier's evidence; black-box timing does not establish that edge.
}

type maintenancePGFixture struct {
	suite                    *maintenanceCLISuite
	ctx                      context.Context
	pool, observer           *pgxpool.Pool
	schema, dsn, application string
}

type maintenancePGCase struct {
	userID, operationID, credentialID string
	before                            identitypkg.PasswordResetRequeueSnapshot
}

// Input lexical validation happens before official parsing, to exclude implicit
// certificate/passfile/service reads. These are the retained preflight boundaries.
func maintenanceDSNKeys(raw string) (map[string]string, bool) {
	values := map[string]string{}
	put := func(key, value string) bool {
		if !maintenanceDSNKeyAllowed(key) {
			return false
		}
		if key == "dbname" {
			key = "database"
		}
		if _, duplicate := values[key]; duplicate {
			return false
		}
		values[key] = value
		return true
	}
	if strings.HasPrefix(raw, "postgres://") || strings.HasPrefix(raw, "postgresql://") {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Fragment != "" || parsed.Opaque != "" {
			return nil, false
		}
		if parsed.User != nil {
			values["user"] = parsed.User.Username()
		}
		if parsed.Path != "" {
			values["database"] = strings.TrimPrefix(parsed.Path, "/")
		}
		query, err := url.ParseQuery(parsed.RawQuery)
		if err != nil {
			return nil, false
		}
		for key, list := range query {
			if len(list) != 1 || !put(key, list[0]) {
				return nil, false
			}
		}
		return values, true
	}
	return maintenanceDSNKeywordKeys(raw, values, put)
}

func maintenanceDSNKeywordKeys(raw string, values map[string]string, put func(string, string) bool) (map[string]string, bool) {
	rest := strings.TrimSpace(raw)
	for rest != "" {
		equal := strings.IndexByte(rest, '=')
		if equal < 0 {
			return nil, false
		}
		key := strings.TrimSpace(rest[:equal])
		if key == "" || strings.ContainsAny(key, " \t\r\n\v\f") {
			return nil, false
		}
		value, remaining, ok := maintenanceDSNValue(strings.TrimLeft(rest[equal+1:], " \t\r\n\v\f"))
		if !ok || !put(key, value) {
			return nil, false
		}
		rest = strings.TrimLeft(remaining, " \t\r\n\v\f")
	}
	return values, true
}
func maintenanceDSNValue(rest string) (string, string, bool) {
	quoted := strings.HasPrefix(rest, "'")
	if quoted {
		rest = rest[1:]
	}
	var value strings.Builder
	end := 0
	closed := !quoted
	for end < len(rest) {
		ch := rest[end]
		if ch == '\\' {
			end++
			if end == len(rest) {
				return "", "", false
			}
			value.WriteByte(rest[end])
			end++
			continue
		}
		if quoted && ch == '\'' {
			end++
			closed = true
			break
		}
		if !quoted && strings.ContainsRune(" \t\r\n\v\f", rune(ch)) {
			break
		}
		value.WriteByte(ch)
		end++
	}
	return value.String(), rest[end:], closed
}
func maintenanceDSNKeyAllowed(key string) bool {
	switch key {
	case "host", "port", "user", "password", "database", "dbname", "sslmode", "connect_timeout", "application_name", "sslsni", "sslnegotiation", "channel_binding", "min_protocol_version", "max_protocol_version":
		return true
	default:
		return false
	}
}
func maintenanceEndpoint(host string, port uint16) bool {
	ip, err := netip.ParseAddr(host)
	return err == nil && ip.Zone() == "" && ip.IsLoopback() && port == 5433
}

func maintenanceSafePGConfig(t *testing.T, suite *maintenanceCLISuite) (*pgxpool.Config, string) {
	t.Helper()
	for _, entry := range os.Environ() {
		key, value, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(strings.ToUpper(key), "PG") && value != "" {
			t.Fatal("unsafe ambient PG configuration; no connection attempted")
		}
	}
	raw := os.Getenv("HAN_TEST_POSTGRES_DSN")
	if strings.TrimSpace(raw) == "" || strings.IndexByte(raw, 0) >= 0 {
		t.Fatal("missing or unsafe process test DSN; no alternate source used")
	}
	values, ok := maintenanceDSNKeys(raw)
	if !ok || strings.TrimSpace(values["user"]) == "" || strings.TrimSpace(values["database"]) == "" {
		t.Fatal("unsafe DSN keys or missing explicit user/database; no connection attempted")
	}
	if values["sslmode"] == "verify-ca" || values["sslmode"] == "verify-full" {
		t.Fatal("unapproved certificate source; no connection attempted")
	}
	isolated := raw + " passfile='' servicefile='' sslcert='' sslkey='' sslrootcert=''"
	if strings.HasPrefix(raw, "postgres://") || strings.HasPrefix(raw, "postgresql://") {
		separator := "?"
		if strings.Contains(raw, "?") {
			separator = "&"
		}
		isolated = raw + separator + "passfile=&servicefile=&sslcert=&sslkey=&sslrootcert="
	}
	config, err := pgxpool.ParseConfig(isolated)
	maintenanceMust(t, err, "official pgx parse (redacted)")
	if config.ConnConfig.User != values["user"] || config.ConnConfig.Database != values["database"] || !maintenanceEndpoint(config.ConnConfig.Host, config.ConnConfig.Port) {
		t.Fatal("unsafe parsed identity/primary endpoint; no connection attempted")
	}
	for _, fallback := range config.ConnConfig.Fallbacks {
		if fallback == nil || !maintenanceEndpoint(fallback.Host, fallback.Port) {
			t.Fatal("unsafe fallback endpoint; no connection attempted")
		}
	}
	for key := range config.ConnConfig.RuntimeParams {
		if key != "application_name" {
			t.Fatal("uncontrolled runtime parameter; no connection attempted")
		}
	}
	if config.ConnConfig.ValidateConnect != nil || config.ConnConfig.AfterConnect != nil {
		t.Fatal("unexpected connect hook")
	}
	suite.redactions = append(suite.redactions, raw, config.ConnConfig.Password, config.ConnConfig.User, config.ConnConfig.Database)
	return maintenancePinPGNetwork(config), raw
}

func maintenancePinPGNetwork(config *pgxpool.Config) *pgxpool.Config {
	config.MaxConns = 1
	config.MinConns = 0
	config.MinIdleConns = 0
	config.HealthCheckPeriod = time.Hour
	config.ConnConfig.Tracer = nil
	config.ConnConfig.ConnectTimeout = 3 * time.Second
	config.ConnConfig.LookupFunc = func(_ context.Context, host string) ([]string, error) {
		if !maintenanceEndpoint(host, 5433) {
			return nil, errors.New("endpoint rejected")
		}
		return []string{host}, nil
	}
	config.ConnConfig.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil || network != "tcp" || port != "5433" || !maintenanceEndpoint(host, 5433) {
			return nil, errors.New("endpoint rejected")
		}
		return (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, network, address)
	}
	return config
}

func newMaintenancePGFixture(t *testing.T, suite *maintenanceCLISuite) *maintenancePGFixture {
	t.Helper()
	config, raw := maintenanceSafePGConfig(t, suite)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	t.Cleanup(cancel)
	f := &maintenancePGFixture{suite: suite, ctx: ctx, schema: "han_s105_" + strings.ReplaceAll(maintenanceUUID(t), "-", "")}
	f.application = f.schema + "_cli"
	preflight := config.Copy()
	preflight.ConnConfig.RuntimeParams = map[string]string{"application_name": "han_s105_readonly_preflight", "default_transaction_read_only": "on"}
	pool, err := pgxpool.NewWithConfig(ctx, preflight)
	maintenanceMust(t, err, "read-only preflight pool")
	t.Cleanup(pool.Close)
	f.verifyEndpoint(t, pool, preflight, true)
	pool.Close()
	config.ConnConfig.RuntimeParams = map[string]string{"application_name": f.schema + "_fixture", "search_path": pgx.Identifier{f.schema}.Sanitize()}
	config.BeforeClose = func(conn *pgx.Conn) {
		cleanup, closeCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer closeCancel()
		_ = conn.Close(cleanup)
	}
	f.pool, err = pgxpool.NewWithConfig(ctx, config)
	maintenanceMust(t, err, "retained fixture pool")
	t.Cleanup(func() {
		if f.observer != nil {
			f.observer.Close()
		}
		f.pool.Close()
	})
	f.verifyEndpoint(t, f.pool, config, false)
	var exists bool
	maintenanceMust(t, f.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname=$1)", f.schema).Scan(&exists), "new schema absence")
	if exists {
		t.Fatal("generated schema already exists; refusing reuse")
	}
	f.exec(t, "CREATE SCHEMA "+pgx.Identifier{f.schema}.Sanitize()+" AUTHORIZATION CURRENT_USER")
	f.applyMigrations(t)
	f.observer, err = pgxpool.NewWithConfig(ctx, config.Copy())
	maintenanceMust(t, err, "independent observer pool")
	f.verifyEndpoint(t, f.observer, config, false)
	f.dsn = maintenanceScopedDSN(t, raw, f.schema, f.application)
	suite.writeJSON(t, f.schema+".fixture.json", map[string]any{"schema": f.schema, "retained": true, "migration_count": 9, "pin_version": maintenancePinVersion, "main_and_all_fallbacks": "literal_loopback:5433", "explicit_identity_match": true, "DSN_recorded": false, "independent_observer": true})
	t.Logf("retained schema=%s", f.schema)
	return f
}

func (f *maintenancePGFixture) verifyEndpoint(t *testing.T, pool *pgxpool.Pool, config *pgxpool.Config, readonly bool) {
	t.Helper()
	var host, userName, database string
	var port uint16
	var version int
	var readOnly bool
	ctx, cancel := context.WithTimeout(f.ctx, 10*time.Second)
	defer cancel()
	maintenanceMust(t, pool.QueryRow(ctx, `SELECT host(inet_server_addr()),inet_server_port(),current_setting('server_version_num')::int,current_user,current_database(),current_setting('transaction_read_only')='on'`).Scan(&host, &port, &version, &userName, &database, &readOnly), "metadata-only endpoint preflight")
	if !maintenanceEndpoint(host, port) || version != maintenancePinVersion || userName != config.ConnConfig.User || database != config.ConnConfig.Database || readonly && !readOnly {
		t.Fatal("PG endpoint/version/identity/read-only mismatch; version drift is not adopted")
	}
	matched := host == config.ConnConfig.Host && port == config.ConnConfig.Port
	for _, fallback := range config.ConnConfig.Fallbacks {
		matched = matched || host == fallback.Host && port == fallback.Port
	}
	if !matched {
		t.Fatal("actual PG endpoint differs from approved endpoints")
	}
}

func maintenanceScopedDSN(t *testing.T, raw, schema, application string) string {
	t.Helper()
	if strings.HasPrefix(raw, "postgres://") || strings.HasPrefix(raw, "postgresql://") {
		parsed, err := url.Parse(raw)
		maintenanceMust(t, err, "scoped test config")
		query := parsed.Query()
		query.Set("search_path", pgx.Identifier{schema}.Sanitize())
		query.Set("application_name", application)
		query.Set("pool_max_conns", "1")
		parsed.RawQuery = query.Encode()
		return parsed.String()
	}
	return raw + " search_path='" + pgx.Identifier{schema}.Sanitize() + "' application_name='" + application + "' pool_max_conns=1"
}
func (f *maintenancePGFixture) exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	_, err := f.pool.Exec(f.ctx, sql, args...)
	maintenanceMust(t, err, "isolated fixture SQL")
}
func maintenanceRollback(t *testing.T, tx pgx.Tx) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := tx.Rollback(ctx)
	if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		maintenanceMust(t, err, "fixture transaction rollback")
	}
}
func (f *maintenancePGFixture) applyMigrations(t *testing.T) {
	items, err := s33FrozenIdentityMigrations(os.DirFS(f.suite.repo))
	maintenanceMust(t, err, "full migration load")
	if len(items) != 11 {
		t.Fatal("expected all eleven forward migrations")
	}
	tx, err := f.pool.Begin(f.ctx)
	maintenanceMust(t, err, "migration transaction")
	defer maintenanceRollback(t, tx)
	for i, item := range items {
		if item.Version != i+1 {
			t.Fatal("migration chain not contiguous")
		}
		_, err := tx.Exec(f.ctx, item.SQL)
		maintenanceMust(t, err, "retained migration apply")
	}
	maintenanceMust(t, tx.Commit(f.ctx), "migration commit")
}

func (f *maintenancePGFixture) newCase(t *testing.T, state, delivery string, attempt int64, nextOffset time.Duration) *maintenancePGCase {
	t.Helper()
	item := &maintenancePGCase{userID: maintenanceUUID(t), operationID: maintenanceUUID(t), credentialID: maintenanceUUID(t)}
	now := time.Now().UTC().Truncate(time.Microsecond)
	created := now.Add(-5 * time.Minute)
	tx, err := f.pool.Begin(f.ctx)
	maintenanceMust(t, err, "legal five-object fixture begin")
	defer maintenanceRollback(t, tx)
	_, err = tx.Exec(f.ctx, `INSERT INTO users(id,username,email,phone,role,status,credential_mode,auth_generation,password_reset_execution_generation,password_reset_current_operation_id,created_at,updated_at) VALUES($1,$2,$3,$4,'member','active','permanent',9007199254740993,1,$5,$6,$6)`, item.userID, "s105-"+item.userID, item.userID+"@example.invalid", "+1555"+strconv.FormatUint(uint64(now.UnixNano()%10000000000), 10), item.operationID, created)
	maintenanceMust(t, err, "legal fixture user")
	var material [128]byte
	_, err = rand.Read(material[:])
	maintenanceMust(t, err, "synthetic claim material")
	_, err = tx.Exec(f.ctx, `INSERT INTO identity_identifiers(user_id,kind,normalized_value,lifecycle,created_at)
 SELECT id,'username',username,'active',created_at FROM users WHERE id=$1
 UNION ALL SELECT id,'email',email,'active',created_at FROM users WHERE id=$1
 UNION ALL SELECT id,'phone',phone,'active',created_at FROM users WHERE id=$1`, item.userID)
	maintenanceMust(t, err, "legal fixture identifiers")
	f.insertFiveObjects(t, tx, item, material[:], created, now, attempt, nextOffset)
	f.advanceState(t, tx, item, state, delivery, now)
	maintenanceMust(t, tx.Commit(f.ctx), "legal five-object fixture commit")
	item.before = f.snapshot(t, item)
	return item
}

func (f *maintenancePGFixture) insertFiveObjects(t *testing.T, tx pgx.Tx, item *maintenancePGCase, material []byte, created, now time.Time, attempt int64, nextOffset time.Duration) {
	t.Helper()
	_, err := tx.Exec(f.ctx, `INSERT INTO password_reset_operations(operation_id,user_id,temporary_credential_id,scope_hash,key_locator,key_digest,digest_key_version,request_digest,execution_generation,request_kind,base_auth_generation,target_auth_generation,settlement_state,delivery_state,state,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,'test-v1',$7,1,'initial_reset',9007199254740993,9007199254740994,'pending','not_started','prepared',$8,$8)`, item.operationID, item.userID, item.credentialID, material[:32], material[32:64], material[64:96], material[96:], created)
	maintenanceMust(t, err, "legal operation fixture")
	_, err = tx.Exec(f.ctx, `INSERT INTO temporary_credentials(id,user_id,purpose,generation,auth_generation,source_operation_id,state,secret_digest,delivered_to_email,issued_at,expires_at) VALUES($1,$2,'password_reset',1,9007199254740994,$3,'pending_delivery','synthetic-maintenance-digest',$4,$5,$6)`, item.credentialID, item.userID, item.operationID, item.userID+"@example.invalid", created, now.Add(time.Hour))
	maintenanceMust(t, err, "legal credential fixture")
	_, err = tx.Exec(f.ctx, `INSERT INTO password_reset_recovery_jobs(operation_id,status,attempt_count,next_attempt_at,last_failure_code,created_at,updated_at) VALUES($1,'dead_letter',$2,$3,'synthetic_failure',$4,$5)`, item.operationID, attempt, now.Add(nextOffset), created, now.Add(-time.Minute))
	maintenanceMust(t, err, "legal dead-letter fixture")
	_, err = tx.Exec(f.ctx, `INSERT INTO idempotency_records(scope_hash,key_locator,key_digest,request_digest,digest_key_version,status,execution_generation,password_reset_operation_id,password_reset_user_id,locked_until,expires_at,created_at,updated_at) VALUES($1,$2,$3,$4,'test-v1','processing',1,$5,$6,$7,$8,$9,$9)`, material[:32], material[32:64], material[64:96], material[96:], item.operationID, item.userID, now.Add(-time.Minute), now.Add(24*time.Hour), created)
	maintenanceMust(t, err, "legal claim fixture")
}

func (f *maintenancePGFixture) advanceState(t *testing.T, tx pgx.Tx, item *maintenancePGCase, state, delivery string, now time.Time) {
	t.Helper()
	if state == "prepared" {
		return
	}
	_, err := tx.Exec(f.ctx, `UPDATE password_reset_operations SET state='dispatching',delivery_state='dispatching',delivery_attempt_id=$2,delivery_attempted_at=$3,updated_at=$3 WHERE operation_id=$1`, item.operationID, maintenanceUUID(t), now.Add(-3*time.Second))
	maintenanceMust(t, err, "legal dispatch transition")
	if state == "dispatching" {
		return
	}
	if state == "commit_unknown" {
		if delivery == "dispatching" {
			_, err = tx.Exec(f.ctx, `UPDATE password_reset_operations SET state='commit_unknown',updated_at=$2 WHERE operation_id=$1`, item.operationID, now.Add(-2*time.Second))
		} else {
			_, err = tx.Exec(f.ctx, `UPDATE password_reset_operations SET state='commit_unknown',delivery_state='unknown',delivery_result_recorded_at=$2,updated_at=$2 WHERE operation_id=$1`, item.operationID, now.Add(-2*time.Second))
		}
		maintenanceMust(t, err, "legal unknown transition")
		return
	}
	_, err = tx.Exec(f.ctx, `UPDATE password_reset_operations SET state='recovery_required',delivery_state='accepted',delivery_result_recorded_at=$2,updated_at=$2 WHERE operation_id=$1`, item.operationID, now.Add(-2*time.Second))
	maintenanceMust(t, err, "legal accepted transition")
	if state == "converging" {
		_, err = tx.Exec(f.ctx, `UPDATE password_reset_operations SET state='converging',updated_at=$2 WHERE operation_id=$1`, item.operationID, now.Add(-time.Second))
		maintenanceMust(t, err, "legal converging transition")
	}
}

func (f *maintenancePGFixture) snapshot(t *testing.T, item *maintenancePGCase) identitypkg.PasswordResetRequeueSnapshot {
	t.Helper()
	var s identitypkg.PasswordResetRequeueSnapshot
	var userAt, operationAt, nextAt, createdAt, updatedAt time.Time
	var holdAt, leaseAt, completedAt *time.Time
	err := f.observer.QueryRow(f.ctx, `SELECT u.status,u.auth_generation,u.password_reset_execution_generation,u.password_reset_current_operation_id::text,u.credential_mode,u.auth_integrity_hold,u.auth_integrity_hold_operation_id::text,u.auth_integrity_hold_reason,u.auth_integrity_hold_at,u.updated_at,o.state,o.delivery_state,o.settlement_state,o.execution_generation,o.temporary_credential_id::text,o.updated_at,j.status,j.attempt_count,j.next_attempt_at,j.lease_owner,j.lease_expires_at,j.last_failure_code,j.created_at,j.updated_at,j.completed_at FROM users u JOIN password_reset_operations o ON o.user_id=u.id JOIN password_reset_recovery_jobs j ON j.operation_id=o.operation_id WHERE u.id=$1 AND o.operation_id=$2`, item.userID, item.operationID).Scan(&s.UserStatus, &s.UserAuthGeneration, &s.UserExecutionGeneration, &s.UserCurrentOperationID, &s.CredentialMode, &s.IntegrityHold, &s.IntegrityHoldOperationID, &s.IntegrityHoldReason, &holdAt, &userAt, &s.OperationState, &s.DeliveryState, &s.SettlementState, &s.OperationExecutionGeneration, &s.TemporaryCredentialID, &operationAt, &s.JobStatus, &s.AttemptCount, &nextAt, &s.LeaseOwner, &leaseAt, &s.LastFailureCode, &createdAt, &updatedAt, &completedAt)
	maintenanceMust(t, err, "independent complete snapshot")
	s.IntegrityHoldAt = maintenanceOptionalTime(holdAt)
	s.LeaseExpiresAt = maintenanceOptionalTime(leaseAt)
	s.CompletedAt = maintenanceOptionalTime(completedAt)
	s.UserUpdatedAt = userAt.UTC().Format(maintenanceWireTime)
	s.OperationUpdatedAt = operationAt.UTC().Format(maintenanceWireTime)
	s.NextAttemptAt = nextAt.UTC().Format(maintenanceWireTime)
	s.JobCreatedAt = createdAt.UTC().Format(maintenanceWireTime)
	s.JobUpdatedAt = updatedAt.UTC().Format(maintenanceWireTime)
	return s
}
func maintenanceOptionalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(maintenanceWireTime)
	return &formatted
}
func (f *maintenancePGFixture) image(t *testing.T) map[string]string {
	t.Helper()
	image := map[string]string{}
	for _, table := range []string{"users", "identity_identifiers", "temporary_credentials", "sessions", "idempotency_records", "password_reset_operations", "password_reset_recovery_jobs", "audit_events"} {
		var data []byte
		maintenanceMust(t, f.observer.QueryRow(f.ctx, "SELECT COALESCE(jsonb_agg(to_jsonb(row) ORDER BY to_jsonb(row)::text),'[]'::jsonb)::text FROM "+pgx.Identifier{table}.Sanitize()+" row").Scan(&data), "independent table image")
		image[table] = maintenanceDigest(data)
	}
	return image
}
func (f *maintenancePGFixture) config() map[string]string {
	config := f.suite.config()
	config["HAN_DATABASE_DSN"] = f.dsn
	config["HAN_PASSWORD_RESET_MAINTENANCE_OPERATION_BUDGET"] = "3s"
	return config
}
func (f *maintenancePGFixture) invoke(t *testing.T, payload maintenanceWirePayload) maintenanceCLIObservation {
	return f.suite.observe(t, f.config(), []string{"maintenance", "password-reset", "retry"}, f.suite.envelope(t, maintenanceJSON(t, payload), maintenanceWireDomain), true)
}
func (f *maintenancePGFixture) noChildConnections(t *testing.T) {
	var count int
	maintenanceMust(t, f.observer.QueryRow(f.ctx, "SELECT count(*) FROM pg_stat_activity WHERE application_name=$1", f.application).Scan(&count), "independent child connection count")
	if count != 0 {
		t.Fatal("maintenance process left database connections")
	}
}
func (f *maintenancePGFixture) runCases(t *testing.T) {
	t.Run("I-S105-12_15_22_apply_replay_minimal_write", f.testApplyReplay)
	t.Run("I-S105-13_14_reject_stale_and_state", f.testRejects)
	t.Run("I-S105-13_real_worker_terminal", f.testTerminalStates)
	t.Run("I-S105-14_all_25_signed_before_fields", f.testAllBeforeFields)
	t.Run("I-S105-17_guard_Get_immutable", f.testImmutableGuard)
	t.Run("I-S105-17_different_requests_same_before", f.testDifferentRequests)
	t.Run("I-S105-16_oversize_uuid_guard", f.testOversizeAudit)
	t.Run("I-S105-17_absent_guard_uuid_race", f.testAbsentUUIDRace)
	t.Run("I-S105-17_same_user_and_uuid_concurrency", f.testConcurrency)
	t.Run("I-S105-22_shutdown_1s_socket_cleanup", f.testLockCancellation)
	t.Run("I-S105-22_applied_idle_sdk_close", f.testAppliedIdleClose)
	t.Run("I-S105-22_replay_idle_sdk_close", f.testReplayIdleClose)
	t.Run("I-S105-02_05_numeric_endpoints", f.testNumericEndpoints)
	t.Run("I-S105-19_one_enqueue_max_retries_2_to_5", func(t *testing.T) { isolated := newMaintenancePGFixture(t, f.suite); isolated.testWorkerRetries(t) })
	t.Run("I-S105-15_actual_commit_ack_loss", func(t *testing.T) { isolated := newMaintenancePGFixture(t, f.suite); isolated.testCommitAckLoss(t) })
	t.Run("I-S105-18_real_worker_maintenance_overlap", func(t *testing.T) {
		for _, state := range []struct{ operation, delivery string }{{"prepared", "not_started"}, {"recovery_required", "accepted"}} {
			t.Run(state.operation, func(t *testing.T) {
				isolated := newMaintenancePGFixture(t, f.suite)
				isolated.testWorkerMaintenanceOverlap(t, state.operation, state.delivery)
			})
		}
	})
	f.suite.writeJSON(t, f.schema+".coverage-limits.json", map[string]any{
		"actual_commit_ack_loss":                                             "separate I-S105-15_actual_commit_ack_loss leaf; only its actual result and observer proof establish coverage",
		"exact_native_positive_available_cancel_before_read":                 "Root U barrier only; black-box input bytes do not prove internal scheduling",
		"production_launcher_acl_key_issuance_second_sid_revoke_restore_e2e": "NOT RUN; S3.4 responsibility",
	})
}

type maintenanceCLIReceipt struct {
	RequestID   string `json:"request_id"`
	OperationID string `json:"operation_id"`
	Outcome     string `json:"outcome"`
	AuditID     string `json:"audit_id"`
}

func maintenanceSuccess(t *testing.T, got maintenanceCLIObservation, payload maintenanceWirePayload, outcome string) {
	t.Helper()
	want := maintenanceCLIReceipt{payload.RequestID, payload.OperationID, outcome, payload.RequestID}
	if got.ExitCode != 0 || got.Stderr != "" || got.Stdout != string(append(maintenanceJSON(t, want), '\n')) {
		t.Fatalf("success tuple mismatch: exit=%d stdout=%q stderr=%q", got.ExitCode, got.Stdout, got.Stderr)
	}
}
func (f *maintenancePGFixture) auditCount(t *testing.T, id string) int {
	t.Helper()
	var count int
	maintenanceMust(t, f.observer.QueryRow(f.ctx, "SELECT count(*) FROM audit_events WHERE id=$1", id).Scan(&count), "independent audit count")
	return count
}
func (f *maintenancePGFixture) checkAudit(t *testing.T, payload maintenanceWirePayload, after identitypkg.PasswordResetRequeueSnapshot) {
	t.Helper()
	var occurred time.Time
	var actor, role, action, target, trace string
	var metadataRaw []byte
	maintenanceMust(t, f.observer.QueryRow(f.ctx, `SELECT occurred_at,actor_system,actor_role,action,target_id,trace_id,metadata::text FROM audit_events WHERE id=$1`, payload.RequestID).Scan(&occurred, &actor, &role, &action, &target, &trace, &metadataRaw), "independent exact audit")
	var metadata map[string]string
	maintenanceMust(t, json.Unmarshal(metadataRaw, &metadata), "all-string audit metadata")
	if len(metadata) != 13 || len(metadataRaw) > 65536 || actor != f.suite.subject || role != "password_reset_requeue_executor" || action != "identity.password_reset.requeued" || target != payload.OperationID || trace != payload.RequestID {
		t.Fatal("audit binding/length mismatch")
	}
	p, err := base64.RawURLEncoding.Strict().DecodeString(metadata["payload_b64"])
	maintenanceMust(t, err, "independent payload carrier")
	signature, err := base64.RawURLEncoding.Strict().DecodeString(metadata["approval_signature"])
	maintenanceMust(t, err, "independent signature carrier")
	a, err := base64.RawURLEncoding.Strict().DecodeString(metadata["after_b64"])
	maintenanceMust(t, err, "independent after carrier")
	r, err := base64.RawURLEncoding.Strict().DecodeString(metadata["result_b64"])
	maintenanceMust(t, err, "independent receipt carrier")
	if !bytes.Equal(p, maintenanceJSON(t, payload)) || !bytes.Equal(a, maintenanceJSON(t, after)) || metadata["payload_sha256"] != maintenanceDigest(p) || metadata["after_sha256"] != maintenanceDigest(a) || metadata["result_sha256"] != maintenanceDigest(r) || !ed25519.Verify(f.suite.private.Public().(ed25519.PublicKey), append([]byte(maintenanceWireDomain), p...), signature) {
		t.Fatal("exact signed bytes/after/receipt digest mismatch")
	}
	expected := identitypkg.PasswordResetRequeueReceipt{Version: "han-password-reset-requeue-result/v1", RequestID: payload.RequestID, Outcome: "applied", OperationID: payload.OperationID, JobStatus: "ready", AuditID: payload.RequestID}
	if !bytes.Equal(r, maintenanceJSON(t, expected)) {
		t.Fatal("original canonical receipt mismatch")
	}
	next, _ := time.Parse(maintenanceWireTime, payload.Before.NextAttemptAt)
	updated, _ := time.Parse(maintenanceWireTime, payload.Before.JobUpdatedAt)
	want := payload.Before
	want.JobStatus = "ready"
	if occurred.After(next) {
		want.NextAttemptAt = occurred.UTC().Format(maintenanceWireTime)
	}
	if occurred.After(updated) {
		want.JobUpdatedAt = occurred.UTC().Format(maintenanceWireTime)
	}
	if !reflect.DeepEqual(want, after) {
		t.Fatal("after has forbidden changes or does not share Audit OccurredAt")
	}
	f.suite.writeJSON(t, payload.RequestID+".audit-proof.json", map[string]any{"metadata_fields": len(metadata), "metadata_UTF8_bytes": len(metadataRaw), "all_strings": true, "original_P_signature_after_receipt_exact": true, "generation_over_2pow53_exact": after.UserAuthGeneration == 9007199254740993})
}

type maintenanceAckProxy struct {
	listener                  net.Listener
	upstream                  string
	mu                        sync.Mutex
	connections               []net.Conn
	armed, stopped            bool
	accepted, joined, dropped int
	acceptDone                chan struct{}
	pumps                     sync.WaitGroup
}

func newMaintenanceAckProxy(t *testing.T, upstream string) *maintenanceAckProxy {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	maintenanceMust(t, err, "owned loopback fault proxy listen")
	proxy := &maintenanceAckProxy{listener: listener, upstream: upstream, acceptDone: make(chan struct{})}
	go proxy.accept()
	t.Cleanup(proxy.stop)
	return proxy
}

func (proxy *maintenanceAckProxy) accept() {
	defer close(proxy.acceptDone)
	for {
		client, err := proxy.listener.Accept()
		if err != nil {
			return
		}
		server, err := (&net.Dialer{Timeout: 3 * time.Second}).Dial("tcp4", proxy.upstream)
		if err != nil {
			_ = client.Close()
			continue
		}
		proxy.mu.Lock()
		if proxy.stopped {
			proxy.mu.Unlock()
			_ = client.Close()
			_ = server.Close()
			return
		}
		proxy.connections = append(proxy.connections, client, server)
		proxy.accepted++
		proxy.pumps.Add(2)
		proxy.mu.Unlock()
		go proxy.pump(client, server, false)
		go proxy.pump(server, client, true)
	}
}

func (proxy *maintenanceAckProxy) pump(source, destination net.Conn, serverToClient bool) {
	defer proxy.pumps.Done()
	defer func() {
		_ = source.Close()
		_ = destination.Close()
		proxy.mu.Lock()
		proxy.joined++
		proxy.mu.Unlock()
	}()
	if !serverToClient {
		_, _ = io.Copy(destination, source)
		return
	}
	buffer := make([]byte, 32*1024)
	for {
		count, readErr := source.Read(buffer)
		if count > 0 && !proxy.forward(destination, buffer[:count]) {
			return
		}
		if readErr != nil {
			return
		}
	}
}

func (proxy *maintenanceAckProxy) forward(destination net.Conn, buffer []byte) bool {
	proxy.mu.Lock()
	defer proxy.mu.Unlock()
	if proxy.armed {
		proxy.dropped += len(buffer)
		return true
	}
	if err := destination.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
		return false
	}
	for len(buffer) > 0 {
		count, err := destination.Write(buffer)
		if err != nil || count == 0 {
			return false
		}
		buffer = buffer[count:]
	}
	return true
}

func (proxy *maintenanceAckProxy) arm() {
	proxy.mu.Lock()
	defer proxy.mu.Unlock()
	proxy.armed = true
}

func (proxy *maintenanceAckProxy) counts() (int, int, int) {
	proxy.mu.Lock()
	defer proxy.mu.Unlock()
	return proxy.accepted, proxy.joined, proxy.dropped
}

func (proxy *maintenanceAckProxy) stop() {
	_ = proxy.listener.Close()
	proxy.mu.Lock()
	proxy.stopped = true
	connections := append([]net.Conn(nil), proxy.connections...)
	proxy.mu.Unlock()
	for _, conn := range connections {
		_ = conn.Close()
	}
	<-proxy.acceptDone
	proxy.pumps.Wait()
}

func maintenanceAckProxyDSN(t *testing.T, raw string, port int, schema, application string) string {
	t.Helper()
	if strings.HasPrefix(raw, "postgres://") || strings.HasPrefix(raw, "postgresql://") {
		parsed, err := url.Parse(raw)
		maintenanceMust(t, err, "approved proxy URL carrier")
		query := parsed.Query()
		query.Set("host", "127.0.0.1")
		query.Set("port", strconv.Itoa(port))
		parsed.RawQuery = query.Encode()
		return maintenanceScopedDSN(t, parsed.String(), schema, application)
	}
	values, ok := maintenanceDSNKeys(raw)
	if !ok {
		t.Fatal("fault-proxy approved DSN lexical validation")
	}
	for _, host := range strings.Split(values["host"], ",") {
		if host != "127.0.0.1" {
			t.Fatal("ACK-loss NOTRUN: approved endpoints cannot preserve host identity through authorized IPv4 proxy")
		}
	}
	values["port"] = strconv.Itoa(port)
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var encoded strings.Builder
	escape := strings.NewReplacer("\\", "\\\\", "'", "\\'")
	for _, key := range keys {
		fmt.Fprintf(&encoded, "%s='%s' ", key, escape.Replace(values[key]))
	}
	return maintenanceScopedDSN(t, encoded.String(), schema, application)
}

type maintenanceAckLock struct {
	conn *pgxpool.Conn
	key  int64
	once sync.Once
}

func (f *maintenancePGFixture) installAckFault(t *testing.T, payload maintenanceWirePayload) *maintenanceAckLock {
	t.Helper()
	var random [8]byte
	_, err := rand.Read(random[:])
	maintenanceMust(t, err, "random private advisory key")
	var key int64
	for _, octet := range random {
		key = (key << 8) | int64(octet)
	}
	key &= 0x7fffffffffffffff
	if key == 0 {
		key = 1
	}
	function := pgx.Identifier{f.schema, "s105_commit_ack_fault"}.Sanitize()
	f.exec(t, fmt.Sprintf("CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_advisory_xact_lock(%d::bigint); RETURN NEW; END; $$", function, key))
	f.exec(t, fmt.Sprintf("CREATE CONSTRAINT TRIGGER s105_commit_ack_fault AFTER INSERT ON %s DEFERRABLE INITIALLY DEFERRED FOR EACH ROW WHEN (NEW.id='%s'::uuid) EXECUTE FUNCTION %s()", pgx.Identifier{f.schema, "audit_events"}.Sanitize(), payload.RequestID, function))
	conn, err := f.pool.Acquire(f.ctx)
	maintenanceMust(t, err, "private advisory-lock session acquire")
	lock := &maintenanceAckLock{conn: conn, key: key}
	t.Cleanup(func() { lock.release(t) })
	_, err = conn.Exec(f.ctx, "SELECT pg_advisory_lock($1::bigint)", key)
	maintenanceMust(t, err, "private advisory-lock barrier hold")
	return lock
}

func (lock *maintenanceAckLock) release(t *testing.T) {
	t.Helper()
	lock.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		var unlocked bool
		err := lock.conn.QueryRow(ctx, "SELECT pg_advisory_unlock($1::bigint)", lock.key).Scan(&unlocked)
		lock.conn.Release()
		if err != nil || !unlocked {
			t.Error("fixture advisory barrier release failed; cause redacted")
		}
	})
}

func maintenanceAckPoll(t *testing.T, childDone <-chan struct{}, predicate func() bool, stage string) {
	t.Helper()
	timer := time.NewTimer(4 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		if predicate() {
			return
		}
		select {
		case <-childDone:
			t.Fatalf("ACK-loss fixture child ended before %s", stage)
		case <-timer.C:
			t.Fatalf("ACK-loss fixture barrier unproved at %s; not a product PASS", stage)
		case <-ticker.C:
		}
	}
}

func (f *maintenancePGFixture) waitCommitBarrier(t *testing.T, lock *maintenanceAckLock, done <-chan struct{}) int {
	t.Helper()
	var pid int
	maintenanceAckPoll(t, done, func() bool {
		err := f.observer.QueryRow(f.ctx, "SELECT a.pid FROM pg_stat_activity a JOIN pg_locks l ON l.pid=a.pid WHERE a.application_name=$1 AND a.state='active' AND lower(btrim(a.query))='commit' AND a.wait_event_type='Lock' AND a.wait_event='advisory' AND l.locktype='advisory' AND NOT l.granted AND l.classid::bigint=$2 AND l.objid::bigint=$3 AND l.objsubid=1", f.application, lock.key>>32, lock.key&0xffffffff).Scan(&pid)
		if errors.Is(err, pgx.ErrNoRows) {
			return false
		}
		maintenanceMust(t, err, "observer exact COMMIT advisory waiter")
		return true
	}, "COMMIT advisory wait")
	return pid
}

func (f *maintenancePGFixture) waitCommittedAck(t *testing.T, payload maintenanceWirePayload, done <-chan struct{}) {
	t.Helper()
	maintenanceAckPoll(t, done, func() bool {
		var committed bool
		err := f.observer.QueryRow(f.ctx, "SELECT EXISTS(SELECT 1 FROM audit_events a JOIN password_reset_recovery_jobs j ON j.operation_id=$2 WHERE a.id=$1 AND j.status='ready')", payload.RequestID, payload.OperationID).Scan(&committed)
		maintenanceMust(t, err, "independent committed job plus Audit visibility")
		return committed
	}, "committed job and Audit")
}

func (f *maintenancePGFixture) testCommitAckLoss(t *testing.T) {
	config, raw := maintenanceSafePGConfig(t, f.suite)
	if config.ConnConfig.Host != "127.0.0.1" {
		t.Fatal("ACK-loss NOTRUN: IPv4 proxy would change approved host identity")
	}
	for _, fallback := range config.ConnConfig.Fallbacks {
		if fallback.Host != "127.0.0.1" {
			t.Fatal("ACK-loss NOTRUN: fallback would change approved host identity")
		}
	}
	item := f.newCase(t, "prepared", "not_started", 2, -time.Minute)
	payload := f.suite.payload(t, item.before, item.userID, item.operationID)
	lock := f.installAckFault(t, payload)
	proxy := newMaintenanceAckProxy(t, net.JoinHostPort(config.ConnConfig.Host, "5433"))
	port := proxy.listener.Addr().(*net.TCPAddr).Port
	childConfig := f.config()
	childConfig["HAN_DATABASE_DSN"] = maintenanceAckProxyDSN(t, raw, port, f.schema, f.application)
	childConfig["HAN_PASSWORD_RESET_MAINTENANCE_OPERATION_BUDGET"] = "8s"
	input := f.suite.envelope(t, maintenanceJSON(t, payload), maintenanceWireDomain)
	done := make(chan struct{})
	var got maintenanceCLIObservation
	go func() {
		defer close(done)
		got = f.suite.observe(t, childConfig, []string{"maintenance", "password-reset", "retry"}, input, true)
	}()
	defer func() { lock.release(t); proxy.stop(); <-done }()
	pid := f.waitCommitBarrier(t, lock, done)
	var negotiatedTLS bool
	maintenanceMust(t, f.observer.QueryRow(f.ctx, "SELECT ssl FROM pg_stat_ssl WHERE pid=$1", pid).Scan(&negotiatedTLS), "observer negotiated TLS fact")
	f.suite.writeJSON(t, payload.RequestID+".ack-barrier.json", map[string]any{"schema": f.schema, "request_id": payload.RequestID, "backend_pid": pid, "commit_advisory_wait": true, "negotiated_TLS": negotiatedTLS, "opaque_proxy": true})
	proxy.arm()
	lock.release(t)
	f.waitCommittedAck(t, payload, done)
	maintenanceAckPoll(t, done, func() bool { _, _, dropped := proxy.counts(); return dropped > 0 }, "opaque server bytes discarded after COMMIT")
	proxy.stop()
	<-done
	maintenanceExpect(t, got, 5, "maintenance.outcome_unknown")
	f.verifyAckReplay(t, item, payload, pid, proxy)
}

func (f *maintenancePGFixture) verifyAckReplay(t *testing.T, item *maintenancePGCase, payload maintenanceWirePayload, pid int, proxy *maintenanceAckProxy) {
	t.Helper()
	after := f.snapshot(t, item)
	f.checkAudit(t, payload, after)
	committed := f.image(t)
	maintenanceSuccess(t, f.invoke(t, payload), payload, "already_applied")
	if !reflect.DeepEqual(committed, f.image(t)) || f.auditCount(t, payload.RequestID) != 1 {
		t.Fatal("ACK-loss fresh invocation replay wrote or duplicated Audit")
	}
	f.noChildConnections(t)
	accepted, joined, dropped := proxy.counts()
	if accepted < 1 || joined != 2*accepted || dropped == 0 {
		t.Fatal("ACK-loss proxy pumps did not actually join or discard server bytes")
	}
	f.suite.writeJSON(t, payload.RequestID+".ack-loss-proof.json", map[string]any{"schema": f.schema, "request_id": payload.RequestID, "backend_pid": pid, "target_constraint_deferred": true, "exact_commit_advisory_wait_observed": true, "arm_before_lock_release": true, "job_and_audit_committed_before_disconnect": true, "opaque_server_bytes_discarded": dropped, "proxy_connections": accepted, "proxy_pumps_joined": joined, "listener_joined": true, "child_wait_joined": true, "child_exit": 5, "stdout_empty": true, "fresh_replay_exit": 0, "replay_no_additional_writes": true, "audit_count": 1, "TLS_configuration_preserved": true, "upstream": "literal_loopback:5433"})
}

func (f *maintenancePGFixture) testApplyReplay(t *testing.T) {
	for _, state := range []struct{ state, delivery string }{{"prepared", "not_started"}, {"dispatching", "dispatching"}, {"commit_unknown", "dispatching"}, {"commit_unknown", "unknown"}, {"recovery_required", "accepted"}, {"converging", "accepted"}} {
		for _, offset := range []time.Duration{-time.Minute, 0, time.Minute} {
			t.Run(state.state+"_"+state.delivery+"_"+offset.String(), func(t *testing.T) {
				item := f.newCase(t, state.state, state.delivery, 2, offset)
				payload := f.suite.payload(t, item.before, item.userID, item.operationID)
				before := f.image(t)
				maintenanceSuccess(t, f.invoke(t, payload), payload, "applied")
				after := f.snapshot(t, item)
				changed := f.image(t)
				for table, digest := range before {
					if table != "password_reset_recovery_jobs" && table != "audit_events" && changed[table] != digest {
						t.Fatalf("forbidden table change: %s", table)
					}
				}
				f.checkAudit(t, payload, after)
				maintenanceSuccess(t, f.invoke(t, payload), payload, "already_applied")
				if !reflect.DeepEqual(changed, f.image(t)) || f.auditCount(t, payload.RequestID) != 1 {
					t.Fatal("replay wrote or duplicate audit")
				}
				f.noChildConnections(t)
				f.suite.writeJSON(t, payload.RequestID+".images.json", map[string]any{"before": before, "after": changed, "snapshot_before": item.before, "snapshot_after": after, "replay_unchanged": true})
			})
		}
	}
}

func (f *maintenancePGFixture) testRejects(t *testing.T) {
	for _, kind := range []string{"stale_before", "ready", "retry_wait", "leased", "hold", "max_attempt", "foreign_user", "missing_operation"} {
		t.Run(kind, func(t *testing.T) {
			item := f.newCase(t, "prepared", "not_started", 2, -time.Minute)
			payload := f.suite.payload(t, item.before, item.userID, item.operationID)
			switch kind {
			case "stale_before":
				f.exec(t, "UPDATE users SET status='suspended' WHERE id=$1", item.userID)
			case "ready", "retry_wait":
				f.exec(t, "UPDATE password_reset_recovery_jobs SET status=$2 WHERE operation_id=$1", item.operationID, kind)
			case "leased":
				f.exec(t, "UPDATE password_reset_recovery_jobs SET status='leased',lease_owner='fixture-worker',lease_expires_at=$2 WHERE operation_id=$1", item.operationID, time.Now().Add(time.Hour))
			case "hold":
				f.exec(t, "UPDATE users SET auth_integrity_hold=true,auth_integrity_hold_operation_id=$2,auth_integrity_hold_reason='fixture_hold',auth_integrity_hold_at=$3 WHERE id=$1", item.userID, item.operationID, time.Now())
			case "max_attempt":
				f.exec(t, "UPDATE password_reset_recovery_jobs SET attempt_count=$2 WHERE operation_id=$1", item.operationID, int64(1<<63-1))
			case "foreign_user":
				payload.UserID = maintenanceUUID(t)
			case "missing_operation":
				payload.OperationID = maintenanceUUID(t)
			}
			if kind != "stale_before" && kind != "foreign_user" && kind != "missing_operation" {
				payload.Before = f.snapshot(t, item)
			}
			image := f.image(t)
			maintenanceExpect(t, f.invoke(t, payload), 4, "maintenance.state_conflict")
			if !reflect.DeepEqual(image, f.image(t)) || f.auditCount(t, payload.RequestID) != 0 {
				t.Fatal("rejected request changed data")
			}
			f.noChildConnections(t)
		})
	}
}

func (f *maintenancePGFixture) appendForeign(t *testing.T, id string, metadata map[string]any) {
	t.Helper()
	tx, err := f.observer.Begin(f.ctx)
	maintenanceMust(t, err, "foreign audit fixture begin")
	defer maintenanceRollback(t, tx)
	maintenanceMust(t, (audit.Store{}).Append(f.ctx, tx, audit.Event{ID: id, OccurredAt: time.Now().UTC(), ActorKind: audit.ActorAnonymous, Action: "synthetic.foreign", TargetType: "synthetic_fixture", TargetID: "foreign", Outcome: "succeeded", TraceID: id, Metadata: metadata}), "foreign append-only event fixture")
	maintenanceMust(t, tx.Commit(f.ctx), "foreign audit fixture commit")
}
func (f *maintenancePGFixture) testOversizeAudit(t *testing.T) {
	for _, size := range []int64{65536, 65537} {
		t.Run("actual_UTF8_JSONB_text_"+strconv.FormatInt(size, 10), func(t *testing.T) { f.testAuditTextBoundary(t, size) })
	}
	for _, kind := range []string{"giant_metadata", "giant_actor_text", "invalid_metadata"} {
		t.Run(kind, func(t *testing.T) {
			item := f.newCase(t, "prepared", "not_started", 2, -time.Minute)
			payload := f.suite.payload(t, item.before, item.userID, item.operationID)
			if kind == "giant_actor_text" {
				tx, err := f.observer.Begin(f.ctx)
				maintenanceMust(t, err, "giant text fixture begin")
				defer maintenanceRollback(t, tx)
				actor := strings.Repeat("foreign", 200000)
				maintenanceMust(t, (audit.Store{}).Append(f.ctx, tx, audit.Event{ID: payload.RequestID, OccurredAt: time.Now(), ActorKind: audit.ActorSystem, ActorSystem: &actor, Action: "synthetic.foreign", TargetType: "synthetic_fixture", TargetID: "foreign", Outcome: "succeeded", TraceID: payload.RequestID, Metadata: map[string]any{}}), "giant Event text fixture")
				maintenanceMust(t, tx.Commit(f.ctx), "giant text commit")
			} else {
				value := "invalid"
				if kind == "giant_metadata" {
					value = strings.Repeat("x", 1024*1024)
				}
				f.appendBoundFixture(t, payload, map[string]any{"foreign": value})
			}
			before := f.image(t)
			maintenanceExpect(t, f.invoke(t, payload), 4, "maintenance.state_conflict")
			if !reflect.DeepEqual(before, f.image(t)) {
				t.Fatal("UUID guard rejection wrote")
			}
			starter := &maintenanceRecordingStarter{pool: f.pool}
			_, domainErr := f.domainCall(t, payload, starter)
			var failure *identitypkg.PasswordResetMaintenanceError
			wantGet := 0
			if kind == "invalid_metadata" {
				wantGet = 1
			}
			if !errors.As(domainErr, &failure) || failure.Code != identitypkg.PasswordResetMaintenanceStateConflict || starter.guard != 1 || starter.get != wantGet || starter.update != 0 || starter.append != 0 {
				t.Fatal("real-PG guard/Get bound or refusal order failed")
			}
			f.suite.writeJSON(t, payload.RequestID+".guard-pg-counters.json", map[string]any{"layer": "real PG domain recording adapter, separate from B", "guard": starter.guard, "Get": starter.get, "update": starter.update, "append": starter.append})
			f.noChildConnections(t)
		})
	}
}

func (f *maintenancePGFixture) auditTextMeasures(t *testing.T, metadata map[string]any) (int64, int64, int64) {
	t.Helper()
	var rendered, characters, valueBytes int64
	err := f.observer.QueryRow(f.ctx, "SELECT octet_length(convert_to($1::jsonb::text,'UTF8'))::bigint,length($1::jsonb::text)::bigint,octet_length(convert_to($1::jsonb->>'foreign','UTF8'))::bigint", string(maintenanceJSON(t, metadata))).Scan(&rendered, &characters, &valueBytes)
	maintenanceMust(t, err, "PG UTF8 JSONB text rendering measurement")
	return rendered, characters, valueBytes
}

func (f *maintenancePGFixture) testAuditTextBoundary(t *testing.T, target int64) {
	item := f.newCase(t, "prepared", "not_started", 2, -time.Minute)
	payload := f.suite.payload(t, item.before, item.userID, item.operationID)
	value := strings.Repeat("界\"\\\n\t", 128)
	metadata := map[string]any{"foreign": value}
	base, _, _ := f.auditTextMeasures(t, metadata)
	if base >= target {
		t.Fatal("boundary seed unexpectedly exceeds target")
	}
	metadata["foreign"] = value + strings.Repeat("x", int(target-base))
	rendered, characters, valueBytes := f.auditTextMeasures(t, metadata)
	if rendered != target || rendered <= characters || rendered-valueBytes <= 15 {
		t.Fatal("fixture lacks exact PG byte boundary, multibyte or escape expansion")
	}
	f.appendBoundFixture(t, payload, metadata)
	var stored int64
	maintenanceMust(t, f.observer.QueryRow(f.ctx, "SELECT octet_length(convert_to(metadata::text,'UTF8'))::bigint FROM audit_events WHERE id=$1", payload.RequestID).Scan(&stored), "stored append-only Audit UTF8 text boundary")
	if stored != target {
		t.Fatal("stored PG JSONB text differs from exact boundary")
	}
	before := f.image(t)
	starter := &maintenanceRecordingStarter{pool: f.pool}
	_, domainErr := f.domainCall(t, payload, starter)
	var failure *identitypkg.PasswordResetMaintenanceError
	wantGet := 0
	if target == 65536 {
		wantGet = 1
	}
	if !errors.As(domainErr, &failure) || failure.Code != identitypkg.PasswordResetMaintenanceStateConflict || starter.guard != 1 || starter.get != wantGet || starter.update != 0 || starter.append != 0 || starter.isolation != pgx.ReadCommitted {
		t.Fatal("actual PG UTF8 cap guard/Get boundary or strict rejection failed")
	}
	if !reflect.DeepEqual(before, f.image(t)) || f.auditCount(t, payload.RequestID) != 1 {
		t.Fatal("boundary call changed job/Audit business data")
	}
	f.suite.writeJSON(t, payload.RequestID+".utf8-jsonb-boundary-proof.json", map[string]any{"layer": "real PG domain adapter", "schema": f.schema, "stored_metadata_UTF8_bytes": stored, "JSONB_text_characters": characters, "value_UTF8_bytes": valueBytes, "multibyte_and_escape_expansion": true, "guard": starter.guard, "Get": starter.get, "update": starter.update, "append": starter.append, "fixture_audit_count": 1, "business_images_unchanged": true, "isolation": starter.isolation})
}

type maintenanceRecordingStarter struct {
	pool                               *pgxpool.Pool
	onGuard                            func(error)
	mu                                 sync.Mutex
	guard, get, append, update, begins int
	isolation                          pgx.TxIsoLevel
}
type maintenanceRecordingTx struct {
	pgx.Tx
	owner *maintenanceRecordingStarter
}
type maintenanceGuardRow struct {
	pgx.Row
	callback func(error)
}

func (row maintenanceGuardRow) Scan(dest ...any) error {
	err := row.Row.Scan(dest...)
	row.callback(err)
	return err
}
func (s *maintenanceRecordingStarter) BeginTx(ctx context.Context, options pgx.TxOptions) (pgx.Tx, error) {
	tx, err := s.pool.BeginTx(ctx, options)
	if err != nil {
		return nil, err
	}
	s.begins++
	s.isolation = options.IsoLevel
	return &maintenanceRecordingTx{Tx: tx, owner: s}, nil
}
func (tx *maintenanceRecordingTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	row := tx.Tx.QueryRow(ctx, sql, args...)
	if strings.Contains(sql, "octet_length(convert_to(metadata::text") {
		tx.owner.guard++
		if tx.owner.onGuard != nil {
			return maintenanceGuardRow{Row: row, callback: tx.owner.onGuard}
		}
	} else if strings.Contains(sql, "FROM audit_events") {
		tx.owner.get++
	}
	return row
}
func (tx *maintenanceRecordingTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if strings.Contains(sql, "INSERT INTO audit_events") {
		tx.owner.append++
	}
	if strings.Contains(sql, "UPDATE password_reset_recovery_jobs") {
		tx.owner.update++
	}
	return tx.Tx.Exec(ctx, sql, args...)
}
func (f *maintenancePGFixture) domainCall(t *testing.T, payload maintenanceWirePayload, starter *maintenanceRecordingStarter) (identitypkg.PasswordResetRequeueResult, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(f.ctx, 3*time.Second)
	defer cancel()
	operator, err := identitypkg.NewPasswordResetMaintenanceOperator(identitypkg.PasswordResetMaintenanceConfig{TrustJSON: f.suite.trust, MaxTTL: 15 * time.Minute, OperationBudget: 3 * time.Second, ShutdownBudget: time.Second}, maintenanceRealClock{})
	maintenanceMust(t, err, "actual-SID production constructor")
	request, err := operator.ValidateRequeue(ctx, f.suite.envelope(t, maintenanceJSON(t, payload), maintenanceWireDomain))
	maintenanceMust(t, err, "actual production validation")
	permit, err := identitypkg.NewPasswordResetConnectionPermit(1)
	maintenanceMust(t, err, "domain permit")
	result, err := operator.Requeue(ctx, request, identitypkg.PasswordResetRequeueDependencies{Transactions: starter, Store: &identitypkg.PasswordResetStore{}, Audit: audit.Store{}, Permit: permit})
	stats := permit.Stats()
	if stats.Current != 0 || stats.Acquired != 1 || stats.Released != 1 {
		t.Fatal("permit not held/returned exactly once")
	}
	return result, err
}

type maintenanceRealClock struct{}

func (maintenanceRealClock) Now() time.Time { return time.Now() }
func (f *maintenancePGFixture) testAbsentUUIDRace(t *testing.T) {
	item := f.newCase(t, "prepared", "not_started", 2, -time.Minute)
	payload := f.suite.payload(t, item.before, item.userID, item.operationID)
	before := f.snapshot(t, item)
	starter := &maintenanceRecordingStarter{pool: f.pool}
	inserted := false
	starter.onGuard = func(err error) {
		if errors.Is(err, pgx.ErrNoRows) && !inserted {
			inserted = true
			f.appendForeign(t, payload.RequestID, map[string]any{"oversize": strings.Repeat("x", 1024*1024)})
		}
	}
	_, err := f.domainCall(t, payload, starter)
	var failure *identitypkg.PasswordResetMaintenanceError
	if !errors.As(err, &failure) || failure.Code != identitypkg.PasswordResetMaintenanceStateConflict || !inserted || starter.guard != 1 || starter.get != 0 || starter.append != 1 || starter.begins != 1 || starter.isolation != pgx.ReadCommitted {
		t.Fatal("absent-guard UUID race did not roll back without Get/retry")
	}
	if !reflect.DeepEqual(before, f.snapshot(t, item)) || f.auditCount(t, payload.RequestID) != 1 {
		t.Fatal("UUID collision failed atomic rollback")
	}
	f.suite.writeJSON(t, payload.RequestID+".uuid-race.json", map[string]any{"layer": "PG domain with real SQL recording adapter; not CLI scheduling proof", "global_uuid_concurrent_foreign_append": true, "guard": starter.guard, "get": starter.get, "append_attempt": starter.append, "begin": starter.begins, "explicit_read_committed": true, "job_rollback_exact": true})
}

func (f *maintenancePGFixture) testConcurrency(t *testing.T) {
	item := f.newCase(t, "prepared", "not_started", 2, -time.Minute)
	payload := f.suite.payload(t, item.before, item.userID, item.operationID)
	done := make(chan maintenanceCLIObservation, 2)
	for range 2 {
		go func() { var got maintenanceCLIObservation; defer func() { done <- got }(); got = f.invoke(t, payload) }()
	}
	outcomes := map[string]int{}
	for range 2 {
		got := <-done
		if got.ExitCode != 0 || got.Stderr != "" {
			t.Fatal("concurrent same request failed")
		}
		var receipt maintenanceCLIReceipt
		maintenanceMust(t, json.Unmarshal([]byte(got.Stdout), &receipt), "concurrent CLI receipt")
		outcomes[receipt.Outcome]++
	}
	if outcomes["applied"] != 1 || outcomes["already_applied"] != 1 || f.auditCount(t, payload.RequestID) != 1 {
		t.Fatal("same request did not commit one enqueue")
	}
	f.noChildConnections(t)
}

func (f *maintenancePGFixture) testLockCancellation(t *testing.T) {
	item := f.newCase(t, "prepared", "not_started", 2, -time.Minute)
	payload := f.suite.payload(t, item.before, item.userID, item.operationID)
	before := f.image(t)
	lock, err := f.observer.Begin(f.ctx)
	maintenanceMust(t, err, "independent user-lock transaction")
	defer maintenanceRollback(t, lock)
	var id string
	maintenanceMust(t, lock.QueryRow(f.ctx, "SELECT id FROM users WHERE id=$1 FOR UPDATE", item.userID).Scan(&id), "independent real user lock")
	config := f.config()
	config["HAN_PASSWORD_RESET_MAINTENANCE_OPERATION_BUDGET"] = "250ms"
	got := f.suite.observe(t, config, []string{"maintenance", "password-reset", "retry"}, f.suite.envelope(t, maintenanceJSON(t, payload), maintenanceWireDomain), true)
	if !((got.ExitCode == 5 && got.Stderr == "maintenance.outcome_unknown\n") || (got.ExitCode == 6 && got.Stderr == "maintenance.failed\n")) || got.Stdout != "" {
		t.Fatal("lock cancellation did not fail closed with rollback certainty preserved")
	}
	// r49 explicitly replaces the old r48 1.5s completion assertion. Raw
	// elapsed time remains recorded; a watchdog Kill is failed fixture evidence,
	// never proof of actual SDK join or a product process-lifetime guarantee.
	maintenanceMust(t, lock.Rollback(f.ctx), "release owned user lock")
	if !reflect.DeepEqual(before, f.image(t)) {
		t.Fatal("lock-canceled maintenance changed data")
	}
	f.noChildConnections(t)
}

func (f *maintenancePGFixture) testAppliedIdleClose(t *testing.T) {
	item := f.newCase(t, "prepared", "not_started", 2, -time.Minute)
	payload := f.suite.payload(t, item.before, item.userID, item.operationID)
	maintenanceSuccess(t, f.invoke(t, payload), payload, "applied")
	f.noChildConnections(t)
}
func (f *maintenancePGFixture) testReplayIdleClose(t *testing.T) {
	item := f.newCase(t, "prepared", "not_started", 2, -time.Minute)
	payload := f.suite.payload(t, item.before, item.userID, item.operationID)
	maintenanceSuccess(t, f.invoke(t, payload), payload, "applied")
	before := f.image(t)
	maintenanceSuccess(t, f.invoke(t, payload), payload, "already_applied")
	if !reflect.DeepEqual(before, f.image(t)) {
		t.Fatal("replay changed data")
	}
	f.noChildConnections(t)
}

// Endpoint refusals are B output observations; Root U separately counts zero
// OS lookup/connect. The negative DSNs are synthetic rejection inputs, never
// alternate credentials or a TLS configuration for a successful connection.
func (f *maintenancePGFixture) testNumericEndpoints(t *testing.T) {
	item := f.newCase(t, "prepared", "not_started", 2, -time.Minute)
	for _, test := range []struct{ name, endpoint string }{
		{"primary_hostname", "host=numeric-refusal.example.invalid port=5433"},
		{"fallback_hostname", "host=127.0.0.1,numeric-refusal.example.invalid port=5433"},
		{"implicit_localhost", "port=5433"},
	} {
		t.Run(test.name, func(t *testing.T) {
			payload := f.suite.payload(t, item.before, item.userID, item.operationID)
			config := f.config()
			config["HAN_DATABASE_DSN"] = test.endpoint + " user=synthetic_numeric_refusal dbname=synthetic_numeric_refusal sslmode=disable"
			before := f.image(t)
			got := f.suite.observe(t, config, []string{"maintenance", "password-reset", "retry"}, f.suite.envelope(t, maintenanceJSON(t, payload), maintenanceWireDomain), true)
			maintenanceExpect(t, got, 2, "maintenance.invalid_config")
			if !reflect.DeepEqual(before, f.image(t)) {
				t.Fatal("post-parse endpoint refusal changed data")
			}
		})
	}
	t.Run("approved_literal_primary_and_all_fallbacks", func(t *testing.T) {
		payload := f.suite.payload(t, item.before, item.userID, item.operationID)
		maintenanceSuccess(t, f.invoke(t, payload), payload, "applied")
		f.checkAudit(t, payload, f.snapshot(t, item))
		f.noChildConnections(t)
	})
}

type maintenanceWorkerAuditFailure struct{}

type maintenanceWorkerBarrierState struct {
	PID     uint32
	Status  string
	Attempt int64
	Owner   string
	Expires time.Time
}

type maintenanceWorkerBarrierAudit struct {
	entered chan maintenanceWorkerBarrierState
	release chan struct{}
}

func (barrier *maintenanceWorkerBarrierAudit) Append(ctx context.Context, tx pgx.Tx, event audit.Event) error {
	state := maintenanceWorkerBarrierState{PID: tx.Conn().PgConn().PID()}
	err := tx.QueryRow(ctx, "SELECT status,attempt_count,lease_owner,lease_expires_at FROM password_reset_recovery_jobs WHERE operation_id=$1", event.TargetID).Scan(&state.Status, &state.Attempt, &state.Owner, &state.Expires)
	if err != nil {
		return err
	}
	if err := (audit.Store{}).Append(ctx, tx, event); err != nil {
		return err
	}
	select {
	case barrier.entered <- state:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case <-barrier.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (f *maintenancePGFixture) overlapWorker(t *testing.T, barrier *maintenanceWorkerBarrierAudit) (*identitypkg.PasswordResetRecoveryWorker, *identitypkg.PasswordResetConnectionPermit) {
	t.Helper()
	permit, err := identitypkg.NewPasswordResetConnectionPermit(1)
	maintenanceMust(t, err, "overlap worker permit")
	store := &identitypkg.PasswordResetStore{}
	clock := maintenanceRealClock{}
	operator, err := identitypkg.NewPasswordResetOperator(identitypkg.PasswordResetOperatorDependencies{Transactions: f.pool, Store: store, Audit: barrier, IDs: maintenanceWorkerIDs{}, Clock: clock, Permit: permit})
	maintenanceMust(t, err, "overlap real settlement operator")
	worker, err := identitypkg.NewPasswordResetRecoveryWorker(identitypkg.PasswordResetWorkerDependencies{Pool: f.pool, Store: store, Operator: operator, Permit: permit, WorkerID: "s105-overlap-worker", Clock: clock}, identitypkg.PasswordResetWorkerConfig{LeaseDuration: time.Minute, RetryBackoff: time.Second, MaxRetries: 5, ClaimBatchLimit: 1, ConnectionBudget: 1})
	maintenanceMust(t, err, "overlap real recovery worker")
	return worker, permit
}

func (f *maintenancePGFixture) waitWorkerBlockedCLI(t *testing.T, workerPID uint32, done <-chan struct{}) int {
	t.Helper()
	var childPID int
	maintenanceAckPoll(t, done, func() bool {
		err := f.observer.QueryRow(f.ctx, "SELECT pid FROM pg_stat_activity WHERE application_name=$1 AND wait_event_type='Lock' AND $2::int=ANY(pg_blocking_pids(pid)) AND lower(query) LIKE '%from users%for update%'", f.application, int64(workerPID)).Scan(&childPID)
		if errors.Is(err, pgx.ErrNoRows) {
			return false
		}
		maintenanceMust(t, err, "observer maintenance User lock blocked by real worker")
		return true
	}, "maintenance User-first wait behind real worker")
	return childPID
}

func (f *maintenancePGFixture) testWorkerMaintenanceOverlap(t *testing.T, state, delivery string) {
	item := f.newCase(t, state, delivery, 2, -time.Minute)
	approved := f.suite.payload(t, item.before, item.userID, item.operationID)
	stale := f.suite.payload(t, item.before, item.userID, item.operationID)
	maintenanceSuccess(t, f.invoke(t, approved), approved, "applied")
	f.checkAudit(t, approved, f.snapshot(t, item))
	barrier := &maintenanceWorkerBarrierAudit{entered: make(chan maintenanceWorkerBarrierState, 1), release: make(chan struct{})}
	worker, permit := f.overlapWorker(t, barrier)
	ctx, cancel := context.WithTimeout(f.ctx, 8*time.Second)
	defer cancel()
	done := make(chan struct{})
	var workerErr error
	go func() { defer close(done); workerErr = worker.Reconcile(ctx) }()
	var release sync.Once
	defer func() { release.Do(func() { close(barrier.release) }); cancel(); <-done }()
	var lease maintenanceWorkerBarrierState
	select {
	case lease = <-barrier.entered:
	case <-done:
		maintenanceMust(t, workerErr, "real worker before overlap barrier")
		t.Fatal("real worker did not enter settlement barrier")
	case <-ctx.Done():
		t.Fatal("real worker did not establish overlap before watchdog")
	}
	if lease.Status != "leased" || lease.Attempt != 3 || lease.Owner != "s105-overlap-worker" || !lease.Expires.After(time.Now()) {
		t.Fatal("real Claim/Renew did not establish a valid third-attempt lease")
	}
	childDone := make(chan struct{})
	var got maintenanceCLIObservation
	go func() { defer close(childDone); got = f.invoke(t, stale) }()
	defer func() { release.Do(func() { close(barrier.release) }); <-childDone }()
	childPID := f.waitWorkerBlockedCLI(t, lease.PID, childDone)
	release.Do(func() { close(barrier.release) })
	<-done
	maintenanceMust(t, workerErr, "real concurrent worker Claim/Renew/settlement")
	<-childDone
	maintenanceExpect(t, got, 4, "maintenance.state_conflict")
	f.verifyWorkerOverlap(t, item, approved, stale, lease, childPID, permit)
}

func (f *maintenancePGFixture) verifyWorkerOverlap(t *testing.T, item *maintenancePGCase, approved, stale maintenanceWirePayload, lease maintenanceWorkerBarrierState, childPID int, permit *identitypkg.PasswordResetConnectionPermit) {
	t.Helper()
	after := f.snapshot(t, item)
	if after.JobStatus != "succeeded" || after.AttemptCount != 3 || (after.OperationState != "succeeded" && after.OperationState != "invalidated") {
		t.Fatal("concurrent worker did not reach its legal terminal state")
	}
	var requeues, recovered, jobs int
	maintenanceMust(t, f.observer.QueryRow(f.ctx, "SELECT count(*) FILTER(WHERE action='identity.password_reset.requeued'),count(*) FILTER(WHERE action='identity.password_reset.recovered') FROM audit_events WHERE target_id=$1", item.operationID).Scan(&requeues, &recovered), "observer concurrent Audit counts")
	maintenanceMust(t, f.observer.QueryRow(f.ctx, "SELECT count(*) FROM password_reset_recovery_jobs WHERE operation_id=$1", item.operationID).Scan(&jobs), "observer concurrent job count")
	if requeues != 1 || recovered != 1 || jobs != 1 || f.auditCount(t, stale.RequestID) != 0 {
		t.Fatal("overlap produced extra enqueue/Audit or consumed stale approval")
	}
	beforeReplay := f.image(t)
	maintenanceSuccess(t, f.invoke(t, approved), approved, "already_applied")
	if !reflect.DeepEqual(beforeReplay, f.image(t)) {
		t.Fatal("overlap replay changed data")
	}
	stats := permit.Stats()
	if stats.Current != 0 || stats.Acquired != stats.Released || stats.Acquired < 4 {
		t.Fatal("real worker permits did not return after actual join")
	}
	f.noChildConnections(t)
	f.suite.writeJSON(t, approved.RequestID+".worker-overlap-proof.json", map[string]any{"schema": f.schema, "worker_lease": lease, "maintenance_backend_pid": childPID, "observer_blocking_PID_and_User_FOR_UPDATE": true, "real_worker_Reconcile": true, "worker_and_child_joined": true, "stale_request_exit": 4, "stale_audit_count": 0, "maintenance_audits": requeues, "worker_audits": recovered, "job_count": jobs, "permit": stats, "after": after, "replay_unchanged": true})
}

func (maintenanceWorkerAuditFailure) Append(context.Context, pgx.Tx, audit.Event) error {
	return errors.New("synthetic retryable settlement audit failure")
}

type maintenanceWorkerIDs struct{}

func (maintenanceWorkerIDs) NewID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", value[:4], value[4:6], value[6:8], value[8:10], value[10:]), nil
}

type maintenanceWorkerClock struct {
	mu sync.Mutex
	at time.Time
}

func (clock *maintenanceWorkerClock) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	return clock.at
}
func (clock *maintenanceWorkerClock) advance() {
	clock.mu.Lock()
	clock.at = clock.at.Add(2 * time.Second)
	clock.mu.Unlock()
}

func (f *maintenancePGFixture) worker(t *testing.T, maxRetries int64, clock *maintenanceWorkerClock) *identitypkg.PasswordResetRecoveryWorker {
	return f.workerWithAudit(t, maxRetries, clock, maintenanceWorkerAuditFailure{})
}

func (f *maintenancePGFixture) workerWithAudit(t *testing.T, maxRetries int64, clock *maintenanceWorkerClock, auditor identitypkg.LoginAuditAppender) *identitypkg.PasswordResetRecoveryWorker {
	t.Helper()
	permit, err := identitypkg.NewPasswordResetConnectionPermit(1)
	maintenanceMust(t, err, "worker permit")
	store := &identitypkg.PasswordResetStore{}
	operator, err := identitypkg.NewPasswordResetOperator(identitypkg.PasswordResetOperatorDependencies{Transactions: f.pool, Store: store, Audit: auditor, IDs: maintenanceWorkerIDs{}, Clock: clock, Permit: permit})
	maintenanceMust(t, err, "real lease operator with rollback-only audit fault")
	worker, err := identitypkg.NewPasswordResetRecoveryWorker(identitypkg.PasswordResetWorkerDependencies{Pool: f.pool, Store: store, Operator: operator, Permit: permit, WorkerID: "s105-actual-worker", Clock: clock}, identitypkg.PasswordResetWorkerConfig{LeaseDuration: time.Minute, RetryBackoff: time.Second, MaxRetries: maxRetries, ClaimBatchLimit: 1, ConnectionBudget: 1})
	maintenanceMust(t, err, "real recovery worker")
	return worker
}
func (f *maintenancePGFixture) testWorkerRetries(t *testing.T) {
	item := f.newCase(t, "prepared", "not_started", 0, -time.Minute)
	f.exec(t, "UPDATE password_reset_recovery_jobs SET status='ready' WHERE operation_id=$1", item.operationID)
	clock := &maintenanceWorkerClock{at: time.Now().UTC().Truncate(time.Microsecond)}
	oldWorker := f.worker(t, 2, clock)
	for _, attempt := range []int64{1, 2} {
		maintenanceMust(t, oldWorker.Reconcile(f.ctx), "real worker Claim/Renew/failing Execute")
		s := f.snapshot(t, item)
		if s.AttemptCount != attempt {
			t.Fatal("original worker attempt mismatch")
		}
		clock.advance()
	}
	item.before = f.snapshot(t, item)
	if item.before.JobStatus != "dead_letter" || item.before.AttemptCount != 2 {
		t.Fatal("MaxRetries2 did not produce real dead letter")
	}
	payload := f.suite.payload(t, item.before, item.userID, item.operationID)
	maintenanceSuccess(t, f.invoke(t, payload), payload, "applied")
	newWorker := f.worker(t, 5, clock)
	var history []identitypkg.PasswordResetRequeueSnapshot
	for _, attempt := range []int64{3, 4, 5} {
		maintenanceMust(t, newWorker.Reconcile(f.ctx), "real worker automatic subsequent attempt")
		snapshot := f.snapshot(t, item)
		history = append(history, snapshot)
		if snapshot.AttemptCount != attempt {
			t.Fatal("MaxRetries5 did not continue real attempts")
		}
		want := "retry_wait"
		if attempt == 5 {
			want = "dead_letter"
		}
		if snapshot.JobStatus != want {
			t.Fatal("worker retry budget/status mismatch")
		}
		maintenanceSuccess(t, f.invoke(t, payload), payload, "already_applied")
		clock.advance()
	}
	if f.auditCount(t, payload.RequestID) != 1 {
		t.Fatal("automatic attempts duplicated consumption audit")
	}
	fresh := f.suite.payload(t, f.snapshot(t, item), item.userID, item.operationID)
	maintenanceSuccess(t, f.invoke(t, fresh), fresh, "applied")
	maintenanceSuccess(t, f.invoke(t, payload), payload, "already_applied")
	f.suite.writeJSON(t, payload.RequestID+".max-retries-2-to-5.json", map[string]any{"real_worker_claim_and_renew": true, "settlement_failure": "real SQL followed by synthetic Audit error and rollback", "automatic_attempts": history, "original_consumption_audit_count": 1, "fresh_approval_after_dead_letter": true})
}

func (f *maintenancePGFixture) testAllBeforeFields(t *testing.T) {
	item := f.newCase(t, "prepared", "not_started", 2, -time.Minute)
	for i := 0; i < reflect.TypeOf(item.before).NumField(); i++ {
		t.Run(reflect.TypeOf(item.before).Field(i).Name, func(t *testing.T) {
			before := item.before
			value := reflect.ValueOf(&before).Elem().Field(i)
			name := reflect.TypeOf(before).Field(i).Name
			maintenanceMutateBefore(t, value, name, item.before)
			payload := f.suite.payload(t, before, item.userID, item.operationID)
			image := f.image(t)
			maintenanceExpect(t, f.invoke(t, payload), 4, "maintenance.state_conflict")
			if !reflect.DeepEqual(image, f.image(t)) || f.auditCount(t, payload.RequestID) != 0 {
				t.Fatal("stale signed field changed data")
			}
		})
	}
}
func maintenanceMutateBefore(t *testing.T, value reflect.Value, name string, before identitypkg.PasswordResetRequeueSnapshot) {
	t.Helper()
	switch value.Kind() {
	case reflect.Int64:
		value.SetInt(value.Int() + 1)
	case reflect.Bool:
		value.SetBool(!value.Bool())
	case reflect.Pointer:
		text := "different"
		switch name {
		case "UserCurrentOperationID", "IntegrityHoldOperationID":
			text = maintenanceUUID(t)
		case "IntegrityHoldAt", "LeaseExpiresAt", "CompletedAt":
			text = before.JobUpdatedAt
		}
		value.Set(reflect.ValueOf(&text))
	case reflect.String:
		replacements := map[string]string{"UserStatus": "suspended", "CredentialMode": "temporary", "OperationState": "dispatching", "DeliveryState": "dispatching", "SettlementState": "invalidated", "JobStatus": "ready", "TemporaryCredentialID": maintenanceUUID(t)}
		if replacement, ok := replacements[name]; ok {
			value.SetString(replacement)
		} else {
			at, err := time.Parse(maintenanceWireTime, value.String())
			maintenanceMust(t, err, "typed stale timestamp")
			value.SetString(at.Add(time.Microsecond).Format(maintenanceWireTime))
		}
	default:
		t.Fatal("unhandled before field kind")
	}
}
func (f *maintenancePGFixture) testImmutableGuard(t *testing.T) {
	item := f.newCase(t, "prepared", "not_started", 2, -time.Minute)
	payload := f.suite.payload(t, item.before, item.userID, item.operationID)
	maintenanceSuccess(t, f.invoke(t, payload), payload, "applied")
	before := f.image(t)
	rejected := false
	starter := &maintenanceRecordingStarter{pool: f.pool, onGuard: func(err error) {
		if err != nil {
			return
		}
		_, changeErr := f.observer.Exec(f.ctx, "UPDATE audit_events SET metadata='{}'::jsonb WHERE id=$1", payload.RequestID)
		var pg *pgconn.PgError
		rejected = errors.As(changeErr, &pg) && pg.Code == "55000"
	}}
	result, err := f.domainCall(t, payload, starter)
	maintenanceMust(t, err, "guarded real-PG replay")
	if !rejected || starter.guard != 1 || starter.get != 1 || starter.append != 0 || result.Disposition != identitypkg.PasswordResetRequeueAlreadyApplied || !reflect.DeepEqual(before, f.image(t)) {
		t.Fatal("committed guard/Get append-only stability failed")
	}
	f.suite.writeJSON(t, payload.RequestID+".immutable-guard.json", map[string]any{"layer": "PG domain recording adapter", "append_only_mutation_rejected_55000": true, "guard": starter.guard, "get": starter.get, "unchanged": true})
}
func (f *maintenancePGFixture) testDifferentRequests(t *testing.T) {
	item := f.newCase(t, "prepared", "not_started", 2, -time.Minute)
	first := f.suite.payload(t, item.before, item.userID, item.operationID)
	second := first
	second.RequestID = maintenanceUUID(t)
	done := make(chan maintenanceCLIObservation, 2)
	for _, payload := range []maintenanceWirePayload{first, second} {
		go func() { var got maintenanceCLIObservation; defer func() { done <- got }(); got = f.invoke(t, payload) }()
	}
	applied, conflict := 0, 0
	for range 2 {
		got := <-done
		if got.ExitCode == 0 && got.Stderr == "" {
			applied++
		} else if got.ExitCode == 4 && got.Stdout == "" && got.Stderr == "maintenance.state_conflict\n" {
			conflict++
		} else {
			t.Fatal("different IDs/same before unexpected tuple")
		}
	}
	if applied != 1 || conflict != 1 || f.auditCount(t, first.RequestID)+f.auditCount(t, second.RequestID) != 1 {
		t.Fatal("different requests committed duplicate enqueue")
	}
	f.noChildConnections(t)
}
func (suite *maintenanceCLISuite) testRevocation(t *testing.T) {
	for _, kind := range []string{"epoch", "approver_key", "executor", "username_ignored"} {
		t.Run(kind, func(t *testing.T) {
			payload := suite.syntheticPayload(t)
			config := suite.config()
			var trust map[string]any
			maintenanceMust(t, json.Unmarshal(suite.trust, &trust), "test trust copy")
			switch kind {
			case "epoch":
				trust["epoch"] = "new-test-epoch"
			case "approver_key":
				trust["approver_keys"].([]any)[0].(map[string]any)["key_id"] = "new-test-key"
			case "executor":
				trust["executors"].([]any)[0].(map[string]any)["subject"] = "windows-sid:S-1-0-0"
			case "username_ignored":
				config["USER"] = "synthetic-other-user"
				config["USERNAME"] = "synthetic-other-user"
			}
			config["HAN_PASSWORD_RESET_MAINTENANCE_TRUST"] = string(maintenanceJSON(t, trust))
			got := suite.observe(t, config, []string{"maintenance", "password-reset", "retry"}, suite.envelope(t, maintenanceJSON(t, payload), maintenanceWireDomain), true)
			if kind == "username_ignored" {
				maintenanceExpect(t, got, 2, "maintenance.invalid_config")
			} else {
				maintenanceExpect(t, got, 3, "maintenance.forbidden")
			}
		})
	}
}

func (f *maintenancePGFixture) testTerminalStates(t *testing.T) {
	for _, state := range []struct{ operation, delivery string }{{"prepared", "not_started"}, {"recovery_required", "accepted"}} {
		t.Run(state.operation, func(t *testing.T) {
			isolated := newMaintenancePGFixture(t, f.suite)
			item := isolated.newCase(t, state.operation, state.delivery, 2, -time.Minute)
			isolated.exec(t, "UPDATE password_reset_recovery_jobs SET status='ready' WHERE operation_id=$1", item.operationID)
			clock := &maintenanceWorkerClock{at: time.Now().UTC().Truncate(time.Microsecond)}
			worker := isolated.workerWithAudit(t, 5, clock, audit.Store{})
			maintenanceMust(t, worker.Reconcile(isolated.ctx), "real worker terminal transition")
			item.before = isolated.snapshot(t, item)
			if item.before.JobStatus != "succeeded" || (item.before.OperationState != "succeeded" && item.before.OperationState != "invalidated") {
				t.Fatal("real worker did not establish legal terminal fixture")
			}
			payload := isolated.suite.payload(t, item.before, item.userID, item.operationID)
			before := isolated.image(t)
			maintenanceExpect(t, isolated.invoke(t, payload), 4, "maintenance.state_conflict")
			if !reflect.DeepEqual(before, isolated.image(t)) {
				t.Fatal("terminal maintenance rejection changed data")
			}
		})
	}
}

func (f *maintenancePGFixture) appendBoundFixture(t *testing.T, payload maintenanceWirePayload, metadata map[string]any) {
	t.Helper()
	tx, err := f.observer.Begin(f.ctx)
	maintenanceMust(t, err, "bounded audit fixture begin")
	defer maintenanceRollback(t, tx)
	actor, role := f.suite.subject, "password_reset_requeue_executor"
	event := audit.Event{ID: payload.RequestID, OccurredAt: time.Now().UTC(), ActorKind: audit.ActorSystem, ActorSystem: &actor, ActorRole: &role, Action: "identity.password_reset.requeued", TargetType: "password_reset_operation", TargetID: payload.OperationID, Outcome: "succeeded", TraceID: payload.RequestID, Metadata: metadata}
	maintenanceMust(t, (audit.Store{}).Append(f.ctx, tx, event), "correct Event binding with oversized/corrupt metadata fixture")
	maintenanceMust(t, tx.Commit(f.ctx), "bound audit fixture commit")
}
