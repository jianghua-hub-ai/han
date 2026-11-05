//go:build integration

package main

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"han/internal/article"
	"han/internal/identity"
	"han/internal/recovery"

	"github.com/jackc/pgx/v5/pgxpool"
)

// This file proves only the closed port list, the injectable assembly seam and
// the recovery secret candidate seam on the one root. Injecting every port
// below proves the assembly path is executable; it never proves that the
// production Enter/Install action is reachable, complete or ready.

var errS421Sentinel = errors.New("s421: the injected enter/install port was reached")

const (
	s421SecretText = "s421-candidate-secret"
	s421RunID      = "s421-run"
	s421BackupRef  = "s421-backup"
)

// s421RequiredPorts is the fixed order the precheck must report.
var s421RequiredPorts = []string{
	"journal",
	"articleSnapshots",
	"restore",
	"consumer",
	"installation",
	"articleAccount",
	"commitments",
	"sources",
	"anchor",
	"exclusive",
	"secretCandidate",
}

type s421Clock struct{ now time.Time }

func (c s421Clock) Now() time.Time { return c.now }

type s421IDs struct{}

func (s421IDs) NewID() (string, error) { return "00000000-0000-4000-8000-000000000421", nil }

type s421Hasher struct{}

func (s421Hasher) Hash(string) (string, error) { return "sha256:s421", nil }

func (s421Hasher) Verify(string, string) (bool, bool, error) { return false, false, nil }

// Construction-only fixtures embed a nil interface on purpose: the assembly
// stores them but never calls them while wiring the runtime. A method call
// would fail loudly instead of fabricating a success.
type s421Journal struct {
	recovery.ContextRecoveryJournal
}

type s421Snapshots struct{ recovery.VerifiedJournalReader }

type s421Restore struct{ recovery.RestoreDriver }

type s421Consumer struct{ recovery.RecoveryConsumer }

type s421Installation struct{ recovery.InstallationExecutor }

type s421ArticleAccount struct{ identity.ArticleAccountPort }

type s421Commitments struct {
	recovery.InstallationCommitmentStore
}

type s421Anchor struct{ recovery.AnchorReader }

// s421Sources is the first dynamic Install seam: Install reaches it before any
// lifecycle, commitment or executor work.
type s421Sources struct{ err error }

func (s *s421Sources) VerifyInstallationSource(context.Context, recovery.InstallationManifest) error {
	return s.err
}

// s421Exclusive is the first dynamic Enter seam after the root lifecycle
// actually stops and joins its configured surface.
type s421Exclusive struct{ err error }

func (e *s421Exclusive) Acquire(context.Context) (recovery.ExclusiveLease, error) {
	if e.err != nil {
		return nil, e.err
	}
	return nil, nil
}

// s421LifecycleServer is the root-owned maintenance surface fake. Serve blocks
// until Shutdown so the real join owner has to complete a real stop.
type s421LifecycleServer struct {
	started   chan struct{}
	stop      chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
}

func newS421LifecycleServer() *s421LifecycleServer {
	return &s421LifecycleServer{started: make(chan struct{}), stop: make(chan struct{})}
}

func (s *s421LifecycleServer) Serve(net.Listener) error {
	s.startOnce.Do(func() { close(s.started) })
	<-s.stop
	return nil
}

func (s *s421LifecycleServer) Shutdown(context.Context) error {
	s.stopOnce.Do(func() { close(s.stop) })
	return nil
}

func (s *s421LifecycleServer) Close() error { return s.Shutdown(context.Background()) }

type s421Listener struct{}

func (s421Listener) Accept() (net.Conn, error) { return nil, net.ErrClosed }

func (s421Listener) Close() error { return nil }

func (s421Listener) Addr() net.Addr { return s421Addr{} }

type s421Addr struct{}

func (s421Addr) Network() string { return "s421" }

func (s421Addr) String() string { return "s421" }

// s421PlanValues is a complete, valid recovery data set whose file paths live
// under dir. It never writes; callers use dir to prove parsing side effects.
func s421PlanValues(t *testing.T, dir string) map[string]string {
	t.Helper()
	values := recoveryConfigFixture(t)
	values["HAN_RECOVERY_RUNTIME_LOCK_PATH"] = filepath.Join(dir, "deployment.lock")
	values["HAN_RECOVERY_JOURNAL_PATH"] = filepath.Join(dir, "journal")
	values["HAN_RECOVERY_KEYRING_PATH"] = filepath.Join(dir, "keyring")
	values["HAN_RECOVERY_ANCHOR_CONFIG_PATH"] = filepath.Join(dir, "anchor")
	values["HAN_RECOVERY_RESTORE_EXECUTABLE"] = filepath.Join(dir, "restore")
	values["HAN_RECOVERY_RESTORE_MANIFEST_PATH"] = filepath.Join(dir, "restore-manifest.json")
	values["HAN_RECOVERY_RESTORE_EXECUTABLE_SHA256"] = strings.Repeat("a", 64)
	values["HAN_RECOVERY_CONTROL_HTTP_ADDR"] = "127.0.0.1:18080"
	values["HAN_RECOVERY_CONTROL_ORIGIN"] = "https://control.example.test"
	values["HAN_HTTP_ADDR"] = "127.0.0.1:18081"
	values["HAN_METRICS_ADDR"] = "127.0.0.1:18082"
	return values
}

func s421Plan(t *testing.T, action string, binding recovery.RunBinding) recoveryStartupPlan {
	t.Helper()
	values := s421PlanValues(t, t.TempDir())
	values["HAN_RECOVERY_MODE"] = "maintenance"
	if action == "normal" || action == "verify" {
		values["HAN_RECOVERY_MODE"] = "normal"
	}
	command := &recoveryCommand{action: action, binding: binding}
	if action == "install" {
		command.manifest = filepath.Join(t.TempDir(), "installation-manifest.json")
	}
	plan, err := planRecoveryStartup(func(key string) string { return values[key] }, command)
	if err != nil {
		t.Fatalf("plan %s startup: %v", action, err)
	}
	if !plan.present || plan.action != action {
		t.Fatalf("startup plan present=%v action=%q, want present action %q", plan.present, plan.action, action)
	}
	wantMaintenance := action == "enter" || action == "install" || action == "resume"
	if plan.verify != (action == "verify") || plan.maintenance != wantMaintenance {
		t.Fatalf("startup plan verify=%v maintenance=%v for %s", plan.verify, plan.maintenance, action)
	}
	return plan
}

func s421Prepared() *preparedLoginDependencies {
	return &preparedLoginDependencies{
		clock:     s421Clock{now: time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC)},
		ids:       s421IDs{},
		passwords: s421Hasher{},
	}
}

func s421ValidManifest() recovery.InstallationManifest {
	digest := "sha256:" + strings.Repeat("a", 64)
	return recovery.InstallationManifest{
		Version: recovery.InstallationVersion, Kind: "genesis",
		DeploymentID: "s421-deployment", InstallationID: "s421-installation",
		TargetDigest: digest, SourceApprovalRef: "s421-approved", SourceApprovalDigest: digest,
		Recovery: recovery.AnchorSnapshot{JournalID: "s421-recovery", Namespace: recovery.RecoveryNamespaceV1, Epoch: 1},
		Article:  recovery.AnchorSnapshot{JournalID: "s421-article", Namespace: recovery.ArticleDeletionNamespaceV1, Epoch: 1},
		IssuedAt: time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC),
	}
}

// s421FullInputs injects every enter/install port through the seam. It proves
// the assembly path, never a production or complete recovery outcome.
func s421FullInputs(t *testing.T, action string) recoveryRootInputs {
	t.Helper()
	pool := &pgxpool.Pool{}
	runs, err := identity.NewRecoveryRunStore(pool)
	if err != nil {
		t.Fatalf("new recovery run store: %v", err)
	}
	closures, err := article.NewPostgresReplayEvidenceStore(pool)
	if err != nil {
		t.Fatalf("new replay evidence store: %v", err)
	}
	return recoveryRootInputs{
		pool:     pool,
		plan:     s421Plan(t, action, recovery.RunBinding{RunID: s421RunID, BackupRef: s421BackupRef}),
		prepared: s421Prepared(),
		runs:     runs, closures: closures,
		journal:          s421Journal{},
		articleSnapshots: s421Snapshots{},
		restore:          s421Restore{},
		consumer:         s421Consumer{},
		installation:     s421Installation{},
		articleAccount:   s421ArticleAccount{},
		commitments:      s421Commitments{},
		sources:          &s421Sources{err: errS421Sentinel},
		anchor:           s421Anchor{},
		exclusive:        &s421Exclusive{err: errS421Sentinel},
		secretCandidate: func(context.Context, recovery.RunBinding) ([]byte, error) {
			return []byte(s421SecretText), nil
		},
	}
}

// s421RuntimePortPointer reads the single runtime's private dependency. It is
// used only to prove the injected Store/Closures were not constructed again.
func s421RuntimePortPointer(t *testing.T, value recovery.Runtime, field string) uintptr {
	t.Helper()
	pointer := reflect.ValueOf(value)
	if pointer.Kind() != reflect.Pointer || pointer.IsNil() {
		t.Fatalf("runtime %T is not a non-nil pointer", value)
	}
	dependencies := pointer.Elem().FieldByName("deps")
	if !dependencies.IsValid() {
		t.Fatalf("runtime %T has no deps field; the single-instance assertion must be updated", value)
	}
	port := dependencies.FieldByName(field)
	if !port.IsValid() || port.Kind() != reflect.Interface || port.IsNil() {
		t.Fatalf("runtime dependency %s is not a non-nil interface", field)
	}
	return port.Elem().Pointer()
}

// TestRecoveryEnterInstallAssemblyWiring proves the explicit closed list, the
// injectable assembly seam and the secret candidate seam on the one root. The
// injected run only reaches a sentinel failure; it never proves completion,
// production reachability or readiness.
func TestRecoveryEnterInstallAssemblyWiring(t *testing.T) {
	binding := recovery.RunBinding{RunID: s421RunID, BackupRef: s421BackupRef}
	expectedClosed := recovery.ErrAdmissionClosed.Error() + ": missing recovery enter/install ports: " + strings.Join(s421RequiredPorts, ", ")

	t.Run("production shape stays closed", func(t *testing.T) {
		for _, action := range []string{"enter", "install"} {
			t.Run(action, func(t *testing.T) {
				assembly, err := newRecoveryRootAssembly(context.Background(), recoveryRootInputs{
					pool: &pgxpool.Pool{}, plan: s421Plan(t, action, binding), prepared: s421Prepared(),
				})
				if assembly != nil {
					t.Fatalf("production %s shape built an assembly without ports", action)
				}
				if !errors.Is(err, recovery.ErrAdmissionClosed) {
					t.Fatalf("production %s error = %v, want ErrAdmissionClosed", action, err)
				}
				if err.Error() != expectedClosed {
					t.Fatalf("production %s error text = %q, want %q", action, err.Error(), expectedClosed)
				}
				for _, port := range s421RequiredPorts {
					if !strings.Contains(err.Error(), port) {
						t.Fatalf("production %s error text is missing port %q: %v", action, port, err)
					}
				}
			})
		}
	})

	t.Run("injected ports reach the install seam", func(t *testing.T) {
		assembly, err := newRecoveryRootAssembly(context.Background(), s421FullInputs(t, "install"))
		if err != nil {
			t.Fatalf("full install assembly refused injected ports: %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := assembly.runtime.Install(ctx, s421ValidManifest()); !errors.Is(err, errS421Sentinel) {
			t.Fatalf("install error = %v, want the injected source sentinel", err)
		}
	})

	t.Run("injected ports reach the enter seam", func(t *testing.T) {
		assembly, err := newRecoveryRootAssembly(context.Background(), s421FullInputs(t, "enter"))
		if err != nil {
			t.Fatalf("full enter assembly refused injected ports: %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		secret, err := assembly.composition.ports.Secret.ReadInitialRecoverySecret(ctx, binding)
		if err != nil || string(secret) != s421SecretText {
			t.Fatalf("secret candidate = %q, %v, want the injected candidate", secret, err)
		}
		server := newS421LifecycleServer()
		joined, err := assembly.bindControl(server, s421Listener{})
		if err != nil {
			t.Fatalf("bind the maintenance surface: %v", err)
		}
		go func() { _ = joined.Serve(s421Listener{}) }()
		select {
		case <-server.started:
		case <-ctx.Done():
			t.Fatalf("maintenance surface never started: %v", ctx.Err())
		}
		err = assembly.runtime.Enter(ctx, recovery.RecoveryInput{RunBinding: binding, Secret: []byte(s421SecretText)})
		if !errors.Is(err, errS421Sentinel) {
			t.Fatalf("enter error = %v, want the injected exclusive sentinel", err)
		}
	})

	t.Run("one runtime and one store", func(t *testing.T) {
		inputs := s421FullInputs(t, "install")
		assembly, err := newRecoveryRootAssembly(context.Background(), inputs)
		if err != nil {
			t.Fatalf("full install assembly refused injected ports: %v", err)
		}
		if assembly.runtime == nil {
			t.Fatal("assembly.runtime is nil")
		}
		if got := assembly.composition.runtime(); got != assembly.runtime {
			t.Fatalf("composition runtime = %v, assembly runtime = %v (must be one value)", got, assembly.runtime)
		}
		if assembly.composition.ports.Runtime != assembly.runtime {
			t.Fatal("composition ports hold a different runtime")
		}
		if got := assembly.bridge.boundRuntime(); got != assembly.runtime {
			t.Fatalf("bridge runtime = %v, assembly runtime = %v (must be one value)", got, assembly.runtime)
		}
		if got, want := s421RuntimePortPointer(t, assembly.runtime, "Store"), reflect.ValueOf(inputs.runs).Pointer(); got != want {
			t.Fatal("the runtime holds a second recovery run store")
		}
		if got, want := s421RuntimePortPointer(t, assembly.runtime, "Closures"), reflect.ValueOf(inputs.closures).Pointer(); got != want {
			t.Fatal("the runtime holds a second replay evidence store")
		}
	})

	t.Run("default secret source stays closed", func(t *testing.T) {
		assembly, err := newRecoveryRootAssembly(context.Background(), recoveryRootInputs{
			pool: &pgxpool.Pool{}, plan: s421Plan(t, "verify", binding), prepared: s421Prepared(), journal: s421Journal{},
		})
		if err != nil {
			t.Fatalf("verify assembly refused a delivered journal: %v", err)
		}
		if _, err := assembly.composition.ports.Secret.ReadInitialRecoverySecret(context.Background(), binding); !errors.Is(err, recovery.ErrAdmissionClosed) {
			t.Fatalf("default secret source error = %v, want ErrAdmissionClosed", err)
		}
	})
}

// TestRecoveryEnterInstallConfigCommandMatrix proves only that the startup and
// command parsers accept the documented enter/install shapes and reject the
// documented invalid ones. Parsing passing does not mean Enter or Install is
// executable: the assembly still refuses a production shape with
// ErrAdmissionClosed unless every port is injected, and no port is delivered.
func TestRecoveryEnterInstallConfigCommandMatrix(t *testing.T) {
	binding := recovery.RunBinding{RunID: s421RunID, BackupRef: s421BackupRef}
	manifest := filepath.Join(t.TempDir(), "installation-manifest.json")

	t.Run("configuration accepts the documented shapes", func(t *testing.T) {
		cases := []struct {
			name    string
			action  string
			command *recoveryCommand
		}{
			{"normal", "normal", nil},
			{"install", "install", &recoveryCommand{action: "install", manifest: manifest}},
			{"enter", "enter", &recoveryCommand{action: "enter", binding: binding}},
			{"resume", "resume", &recoveryCommand{action: "resume", binding: binding}},
			{"verify", "verify", &recoveryCommand{action: "verify"}},
		}
		for _, test := range cases {
			t.Run(test.name, func(t *testing.T) {
				dir := t.TempDir()
				values := s421PlanValues(t, dir)
				if test.action == "enter" || test.action == "install" || test.action == "resume" {
					values["HAN_RECOVERY_MODE"] = "maintenance"
				}
				plan, err := planRecoveryStartup(func(key string) string { return values[key] }, test.command)
				if err != nil {
					t.Fatalf("%s configuration rejected: %v", test.name, err)
				}
				if !plan.present || plan.action != test.action {
					t.Fatalf("plan present=%v action=%q, want present action %q", plan.present, plan.action, test.action)
				}
				if plan.maintenance != (test.action == "enter" || test.action == "install" || test.action == "resume") {
					t.Fatalf("%s plan maintenance=%v", test.name, plan.maintenance)
				}
				if plan.verify != (test.action == "verify") {
					t.Fatalf("%s plan verify=%v", test.name, plan.verify)
				}
				s421AssertEmptyDir(t, dir)
			})
		}
	})

	t.Run("configuration rejects invalid values without writing", func(t *testing.T) {
		cases := []struct {
			name   string
			action string
			mutate func(values map[string]string)
		}{
			{"illegal mode", "enter", func(values map[string]string) { values["HAN_RECOVERY_MODE"] = "Normal" }},
			{"uppercase target digest", "enter", func(values map[string]string) {
				values["HAN_RECOVERY_TARGET_DIGEST"] = "sha256:" + strings.Repeat("A", 64)
			}},
			{"journal mode 0644", "enter", func(values map[string]string) { values["HAN_RECOVERY_JOURNAL_FILE_MODE"] = "0644" }},
			{"journal mode 0000", "enter", func(values map[string]string) { values["HAN_RECOVERY_JOURNAL_FILE_MODE"] = "0000" }},
			{"relative journal path", "enter", func(values map[string]string) { values["HAN_RECOVERY_JOURNAL_PATH"] = "journal" }},
			{"control port conflict", "enter", func(values map[string]string) { values["HAN_HTTP_ADDR"] = values["HAN_RECOVERY_CONTROL_HTTP_ADDR"] }},
			{"relative restore executable", "enter", func(values map[string]string) { values["HAN_RECOVERY_RESTORE_EXECUTABLE"] = "restore" }},
			{"invalid restore sha", "enter", func(values map[string]string) { values["HAN_RECOVERY_RESTORE_EXECUTABLE_SHA256"] = "abc" }},
			{"install under normal mode", "install", func(values map[string]string) { values["HAN_RECOVERY_MODE"] = "normal" }},
		}
		for _, test := range cases {
			t.Run(test.name, func(t *testing.T) {
				dir := t.TempDir()
				s421AssertEmptyDir(t, dir)
				values := s421PlanValues(t, dir)
				values["HAN_RECOVERY_MODE"] = "maintenance"
				test.mutate(values)
				command := &recoveryCommand{action: test.action, binding: binding}
				if test.action == "install" {
					command = &recoveryCommand{action: "install", manifest: manifest}
				}
				if _, err := planRecoveryStartup(func(key string) string { return values[key] }, command); err == nil {
					t.Fatalf("%s was accepted", test.name)
				}
				s421AssertEmptyDir(t, dir)
			})
		}
	})

	t.Run("command parser accepts the documented shapes", func(t *testing.T) {
		cases := []struct {
			name     string
			args     []string
			selected bool
			action   string
			manifest string
			binding  recovery.RunBinding
		}{
			{"ordinary start", nil, false, "normal", "", recovery.RunBinding{}},
			{"install", []string{"recovery", "enter", "--installation-manifest", manifest}, true, "install", manifest, recovery.RunBinding{}},
			{"enter", []string{"recovery", "enter", "--run-id", s421RunID, "--backup-ref", s421BackupRef}, true, "enter", "", binding},
			{"resume", []string{"recovery", "resume", "--run-id", s421RunID, "--backup-ref", s421BackupRef}, true, "resume", "", binding},
			{"verify", []string{"recovery", "verify"}, true, "verify", "", recovery.RunBinding{}},
		}
		for _, test := range cases {
			t.Run(test.name, func(t *testing.T) {
				got, selected, err := selectRecoveryCommand(test.args)
				if err != nil || selected != test.selected || got.action != test.action || got.manifest != test.manifest || got.binding != test.binding {
					t.Fatalf("select(%v) = %+v, %v, %v", test.args, got, selected, err)
				}
				if !test.selected {
					return
				}
				parsed, err := parseRecoveryCommand(test.args[1:])
				if err != nil || parsed != got {
					t.Fatalf("parse(%v) = %+v, %v, want %+v", test.args[1:], parsed, err, got)
				}
			})
		}
	})

	t.Run("command parser rejects invalid shapes", func(t *testing.T) {
		cases := [][]string{
			{"recovery"},
			{"recovery", "enter"},
			{"recovery", "enter", "--installation-manifest"},
			{"recovery", "enter", "--run-id", s421RunID, "--run-id", "run-2"},
			{"recovery", "enter", "--unknown", "value"},
			{"recovery", "enter", "--installation-manifest", "relative"},
			{"recovery", "enter", "--run-id", s421RunID, "--backup-ref", s421BackupRef, "extra"},
			{"recovery", "enter", "--run-id", " " + s421RunID, "--backup-ref", s421BackupRef},
			{"recovery", "enter", "--run-id", s421RunID, "--backup-ref", s421BackupRef, "--secret", "forbidden"},
			{"recovery", "enter", "--installation-manifest", manifest, "--run-id", s421RunID, "--backup-ref", s421BackupRef},
			{"recovery", "resume", "--installation-manifest", manifest},
		}
		for i, args := range cases {
			if _, selected, err := selectRecoveryCommand(args); !selected || err == nil {
				t.Errorf("invalid command case %d %v was accepted", i, args)
			}
		}
	})
}

func s421AssertEmptyDir(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read temp dir %s: %v", dir, err)
	}
	if len(entries) != 0 {
		t.Fatalf("parsing wrote %d entries into %s", len(entries), dir)
	}
}
