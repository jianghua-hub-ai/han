package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log/slog"
	"math/big"
	mathrand "math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/quick"
	"time"

	"han/internal/article"
	"han/internal/audit"
	"han/internal/idempotency"
	"han/internal/identity"
	"han/internal/mailer"
	"han/internal/platform"
	"han/internal/recovery"
	"han/internal/security"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/jackc/pgx/v5/pgxpool"
)

const mainHelperEnvironment = "HAN_MAIN_HELPER_PROCESS"

func maintenanceRootModeTable(t *testing.T) {
	for _, test := range []struct {
		name                 string
		args                 []string
		maintenance, invalid bool
	}{
		{"server", nil, false, false}, {"exact", []string{"maintenance", "password-reset", "retry"}, true, false},
		{"short", []string{"maintenance"}, true, true}, {"unknown", []string{"server"}, true, true},
		{"extra", []string{"maintenance", "password-reset", "retry", "--x"}, true, true},
		{"wrong_action", []string{"maintenance", "password-reset", "confirm"}, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			maintenance, err := selectMaintenanceMode(test.args)
			if maintenance != test.maintenance || (err != nil) != test.invalid {
				t.Fatalf("mode=%v err=%v", maintenance, err)
			}
		})
	}
}

func maintenanceRootValues() map[string]string {
	return map[string]string{
		maintenanceTrustEnvironment: "{}", maintenanceTTLEnvironment: "15m",
		maintenanceOperationEnvironment: "30s", passwordResetWorkerConnectionBudgetEnvironment: "1",
		shutdownBudgetEnvironment: "1s", recoveryModeEnvironment: "normal",
	}
}

func maintenanceRootConfigTable(t *testing.T) {
	for _, key := range []string{maintenanceTrustEnvironment, maintenanceTTLEnvironment,
		maintenanceOperationEnvironment, passwordResetWorkerConnectionBudgetEnvironment,
		shutdownBudgetEnvironment, recoveryModeEnvironment} {
		for _, value := range []string{"", " ", "invalid", "+1", "0", "-1", "2147483648"} {
			t.Run(key+"/"+strconv.Quote(value), func(t *testing.T) {
				values, reads := maintenanceRootValues(), map[string]int{}
				values[key] = value
				_, err := loadMaintenanceConfig(func(k string) string { reads[k]++; return values[k] })
				// Trust schema is owned and rejected by the real Identity constructor.
				if key != maintenanceTrustEnvironment || len(value) == 0 || len(value) > maintenanceEnvelopeLimit {
					if err == nil {
						t.Fatal("invalid configuration accepted")
					}
				}
				if reads[databaseDSNEnvironment] != 0 || len(reads) > 6 {
					t.Fatalf("unexpected reads=%v", reads)
				}
			})
		}
	}
	for _, budget := range []string{"1s", "60s", "999ms", "60001ms", " 1s", "1s "} {
		t.Run("shutdown/"+budget, func(t *testing.T) {
			values := maintenanceRootValues()
			values[shutdownBudgetEnvironment] = budget
			config, err := loadMaintenanceConfig(func(k string) string { return values[k] })
			valid := budget == "1s" || budget == "60s"
			if (err == nil) != valid || valid && config.shutdownBudget <= 0 {
				t.Fatalf("config=%+v err=%v", config, err)
			}
		})
	}
	for _, budget := range []string{"1", "2147483647", "01", "+1", "2147483648"} {
		values := maintenanceRootValues()
		values[passwordResetWorkerConnectionBudgetEnvironment] = budget
		_, err := loadMaintenanceConfig(func(k string) string { return values[k] })
		if (err == nil) != (budget == "1" || budget == "2147483647") {
			t.Fatalf("budget=%q err=%v", budget, err)
		}
	}
}

type maintenanceInputFake struct {
	data     []byte
	chunk    int
	reads    []int
	closes   int
	pending  int
	peek     func() (uint32, bool, error)
	read     func([]byte) (int, error)
	closeErr error
}

func (input *maintenanceInputFake) Peek() (uint32, bool, error) {
	if input.peek != nil {
		return input.peek()
	}
	return uint32(len(input.data)), len(input.data) == 0, nil
}

func (input *maintenanceInputFake) ReadAvailable(buffer []byte) (int, error) {
	input.reads = append(input.reads, len(buffer))
	input.pending++
	defer func() { input.pending-- }()
	if input.read != nil {
		return input.read(buffer)
	}
	size := len(buffer)
	if input.chunk > 0 {
		size = min(size, input.chunk)
	}
	n := copy(buffer[:size], input.data)
	input.data = input.data[n:]
	return n, nil
}

func (input *maintenanceInputFake) Close() error {
	input.closes++
	if input.pending != 0 {
		panic("close with native I/O pending")
	}
	return input.closeErr
}

func maintenanceRootInputTable(t *testing.T) {
	for _, test := range []struct {
		name string
		size int
		code string
	}{
		{"empty", 0, "maintenance.invalid_request"}, {"single", 1, ""},
		{"limit", maintenanceEnvelopeLimit, ""}, {"plus_one", maintenanceEnvelopeLimit + 1, "maintenance.invalid_request"},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := &maintenanceInputFake{data: bytes.Repeat([]byte("x"), test.size), chunk: 17}
			got, err := readMaintenanceInput(context.Background(), input)
			if test.code == "" && (err != nil || len(got) != test.size) {
				t.Fatalf("len=%d err=%v", len(got), err)
			}
			if test.code != "" {
				code, _ := maintenanceErrorCode(err)
				if code != test.code {
					t.Fatalf("code=%s", code)
				}
			}
			if input.closes != 1 || input.pending != 0 {
				t.Fatalf("closes=%d pending=%d", input.closes, input.pending)
			}
		})
	}
	for _, point := range []string{"before_peek", "after_empty_peek", "after_positive_peek", "after_read", "at_eof", "waiting"} {
		t.Run("cancel/"+point, func(t *testing.T) { maintenanceRootInputCancellation(t, point) })
	}
	for _, kind := range []string{"zero_progress", "native_error", "close_error", "panic"} {
		t.Run(kind, func(t *testing.T) { maintenanceRootInputFailure(t, kind) })
	}
}

func maintenanceRootInputCancellation(t *testing.T, point string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	input := &maintenanceInputFake{data: []byte("complete")}
	input.peek = func() (uint32, bool, error) {
		if point == "waiting" {
			cancel()
			return 0, false, nil
		}
		if point == "after_empty_peek" {
			cancel()
			return 0, false, nil
		}
		if point == "after_positive_peek" {
			cancel()
			return 8, false, nil
		}
		if point == "at_eof" {
			cancel()
			return 0, true, nil
		}
		return uint32(len(input.data)), len(input.data) == 0, nil
	}
	if point == "before_peek" {
		cancel()
	}
	if point == "after_read" {
		input.read = func(buffer []byte) (int, error) { cancel(); return copy(buffer, "complete"), nil }
	}
	got, err := readMaintenanceInput(ctx, input)
	if got != nil || !errors.Is(err, context.Canceled) || input.closes != 1 {
		t.Fatalf("got=%q err=%v closes=%d", got, err, input.closes)
	}
	if point != "after_read" && len(input.reads) != 0 {
		t.Fatal("read after cancellation")
	}
}

func maintenanceRootInputFailure(t *testing.T, kind string) {
	input := &maintenanceInputFake{data: []byte("x")}
	if kind == "zero_progress" {
		input.read = func([]byte) (int, error) { return 0, nil }
	}
	if kind == "native_error" {
		input.read = func([]byte) (int, error) { return 0, errors.New("private cause") }
	}
	if kind == "close_error" {
		input.closeErr = errors.New("private close")
	}
	if kind == "panic" {
		input.read = func([]byte) (int, error) { panic("private panic") }
		defer func() {
			if recover() == nil || input.closes != 1 || input.pending != 0 {
				t.Error("panic did not release input")
			}
		}()
	}
	_, err := readMaintenanceInput(context.Background(), input)
	code, exit := maintenanceErrorCode(err)
	if code != "maintenance.failed" || exit != 6 || input.closes != 1 {
		t.Fatalf("code=%s exit=%d closes=%d", code, exit, input.closes)
	}
}

func maintenanceRootInputProperties(t *testing.T) {
	property := func(data []byte, chunk uint16) bool {
		if len(data) == 0 {
			data = []byte{0}
		}
		original := bytes.Clone(data)
		input := &maintenanceInputFake{data: bytes.Clone(data), chunk: int(chunk%4096) + 1}
		got, err := readMaintenanceInput(context.Background(), input)
		for _, n := range input.reads {
			if n <= 0 || n > maintenanceReadChunk || n > len(original) {
				return false
			}
		}
		return err == nil && bytes.Equal(got, original) && input.closes == 1 && input.pending == 0
	}
	if err := quick.Check(property, &quick.Config{MaxCount: 100, Rand: mathrand.New(mathrand.NewSource(10506))}); err != nil {
		t.Fatal(err)
	}
}

func maintenanceRootDeadlineProperties(t *testing.T) {
	property := func(offset int16, duration uint16) bool {
		budget := time.Duration(duration%60+1) * time.Second
		deadline := time.Now().Add(time.Duration(offset) * time.Millisecond)
		operation, stop := context.WithDeadline(context.Background(), deadline)
		defer stop()
		ctx, cancel := maintenanceCleanupContext(operation, deadline, budget)
		defer cancel()
		got, ok := ctx.Deadline()
		return ok && !got.After(deadline.Add(budget)) && !got.After(time.Now().Add(budget))
	}
	if err := quick.Check(property, &quick.Config{MaxCount: 100, Rand: mathrand.New(mathrand.NewSource(10511))}); err != nil {
		t.Fatal(err)
	}
}

func newMaintenanceRootTestOwner(t *testing.T) *maintenanceConnectionOwner {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	owner := newMaintenanceConnectionOwner(ctx, time.Second)
	t.Cleanup(func() { owner.stopWatch(); cancel() })
	return owner
}

func maintenanceRootCloseCausality(t *testing.T) {
	for _, outcome := range []string{"applied", "already_applied", "async_closed"} {
		t.Run(outcome, func(t *testing.T) {
			owner := newMaintenanceRootTestOwner(t)
			done, events := make(chan struct{}), []string{}
			attempt := &maintenanceAttempt{handed: true}
			entry := &maintenanceCloseEntry{attempt: attempt, finished: make(chan struct{}), cleanupDone: done,
				isClosed: func() bool { return outcome == "async_closed" },
				closeSDK: func(context.Context) error { events = append(events, "sdk_close"); close(done); return nil }}
			if outcome == "async_closed" {
				close(done)
			}
			owner.connections[&pgx.Conn{}] = entry
			err := owner.cleanup(func() (int32, int32) { return 0, 0 }, func() {
				select {
				case <-done:
					events = append(events, "pool_close")
				default:
					t.Error("pool close before SDK done")
				}
			})
			want := []string{"sdk_close", "pool_close"}
			if outcome == "async_closed" {
				want = []string{"pool_close"}
			}
			if err != nil || !reflect.DeepEqual(events, want) {
				t.Fatalf("events=%v err=%v", events, err)
			}
		})
	}
	t.Run("root_before_close_only_one_winner", maintenanceRootCloseRace)
	t.Run("raw_close_not_sdk_done", maintenanceRootAsyncJoin)
}

func maintenanceRootCloseRace(t *testing.T) {
	owner := newMaintenanceRootTestOwner(t)
	done, entered, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	connection := &pgx.Conn{}
	entry := &maintenanceCloseEntry{attempt: &maintenanceAttempt{handed: true}, finished: make(chan struct{}), cleanupDone: done,
		isClosed: func() bool { return false }, closeSDK: func(context.Context) error {
			calls.Add(1)
			close(entered)
			<-release
			close(done)
			return nil
		}}
	owner.connections[connection] = entry
	config := &pgxpool.Config{}
	owner.configure(config)
	var workers sync.WaitGroup
	workers.Add(2)
	go func() { defer workers.Done(); _ = owner.closeMaintenanceConnection(context.Background(), connection) }()
	<-entered
	go func() { defer workers.Done(); config.BeforeClose(connection) }()
	close(release)
	workers.Wait()
	if calls.Load() != 1 {
		t.Fatalf("SDK close calls=%d", calls.Load())
	}
}

func maintenanceRootAsyncJoin(t *testing.T) {
	owner := newMaintenanceRootTestOwner(t)
	done, entered, completed := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	raw, peer := net.Pipe()
	defer peer.Close()
	attempt := &maintenanceAttempt{handed: true}
	transport := &maintenanceTransport{Conn: raw, owner: owner}
	owner.sockets[transport] = attempt
	entry := &maintenanceCloseEntry{attempt: attempt, finished: make(chan struct{}), cleanupDone: done,
		isClosed: func() bool { close(entered); return true }, closeSDK: func(context.Context) error { t.Error("asyncClose reentered"); return nil }}
	go func() { completed <- owner.closeEntry(context.Background(), entry) }()
	<-entered
	var b [1]byte
	_, _ = peer.Read(b[:]) // Physical close is observable while SDK done is still open.
	select {
	case <-completed:
		t.Fatal("raw close was mistaken for SDK completion")
	default:
	}
	close(done)
	if err := <-completed; err != nil {
		t.Fatal(err)
	}
}

func maintenanceRootConstructorStages(t *testing.T) {
	owner := newMaintenanceRootTestOwner(t)
	lookupEntered, lookupDone := make(chan struct{}), make(chan struct{})
	config := &pgx.ConnConfig{Config: pgconn.Config{
		LookupFunc: func(context.Context, string) ([]string, error) { t.Error("inherited resolver called"); return nil, nil },
		DialFunc: func(context.Context, string, string) (net.Conn, error) {
			t.Error("Dial before Lookup completed")
			return nil, errors.New("unexpected dial")
		},
	}}
	if err := owner.beforeConnect(context.Background(), config); err != nil {
		t.Fatal(err)
	}
	tracer := config.Tracer.(pgx.ConnectTracer)
	stage := tracer.TraceConnectStart(context.Background(), pgx.TraceConnectStartData{ConnConfig: config})
	deadline, ok := stage.Deadline()
	if !ok || deadline.After(owner.deadline) {
		t.Fatal("constructor lost original D")
	}
	var active atomic.Int32
	active.Store(1)
	go func() {
		// Hold a local stage before Dial; no OS resolver or injectable production Lookup.
		local, finish, err := owner.stage(stage, config.Tracer.(*maintenanceConnectTracer).attempt)
		if err == nil {
			_, err = config.LookupFunc(local, "127.0.0.1")
			close(lookupEntered)
			<-local.Done()
			finish()
		}
		tracer.TraceConnectEnd(stage, pgx.TraceConnectEndData{Err: err})
		active.Store(0)
		close(lookupDone)
	}()
	<-lookupEntered
	closed := false
	err := owner.cleanup(func() (int32, int32) { return active.Load(), 0 }, func() { closed = true })
	<-lookupDone
	if err != nil || !closed || len(owner.stages) != 0 || len(owner.attempts) != 0 {
		t.Fatalf("err=%v closed=%v stages=%d", err, closed, len(owner.stages))
	}
	if _, _, err := owner.stage(context.Background(), &maintenanceAttempt{}); err == nil {
		t.Fatal("new stage after quiesce")
	}
	if got := config.Tracer.TraceQueryStart(context.Background(), nil, pgx.TraceQueryStartData{}); got != context.Background() {
		t.Fatal("query tracer changed context")
	}
}

func maintenanceRootLateConnection(t *testing.T) {
	owner := newMaintenanceRootTestOwner(t)
	owner.quiesce(false)
	done := make(chan struct{})
	connection := &pgx.Conn{}
	entry := &maintenanceCloseEntry{attempt: &maintenanceAttempt{}, finished: make(chan struct{}), cleanupDone: done,
		isClosed: func() bool { return false }, closeSDK: func(context.Context) error { close(done); return nil }}
	owner.connections[connection] = entry
	if err := owner.afterConnect(context.Background(), connection); err == nil {
		t.Fatal("late connection published")
	}
	select {
	case <-done:
	default:
		t.Fatal("late constructor did not finish SDK close")
	}
	if err := owner.beforeConnect(context.Background(), &pgx.ConnConfig{}); err == nil {
		t.Fatal("constructor started after cleanup")
	}
}

func maintenanceRootEarlyInputOwnership(t *testing.T) {
	for _, stage := range []string{"config", "constructor", "panic", "close_error"} {
		t.Run(stage, func(t *testing.T) {
			input, writer, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer writer.Close()
			values := maintenanceRootValues()
			if stage == "config" || stage == "close_error" {
				delete(values, shutdownBudgetEnvironment)
			}
			if stage == "close_error" {
				if err := input.Close(); err != nil {
					t.Fatal(err)
				}
			}
			dependencies := maintenanceRootDependencies{getenv: func(k string) string {
				if k == databaseDSNEnvironment {
					t.Error("early DSN read")
				}
				return values[k]
			}, prepare: func(maintenanceConfig) (maintenanceValidate, error) {
				if stage == "panic" {
					panic("private constructor material")
				}
				return nil, &identity.PasswordResetMaintenanceError{Code: identity.PasswordResetMaintenanceForbidden}
			}}
			_, err = runMaintenance(context.Background(), input, dependencies)
			if err == nil {
				t.Fatal("early path unexpectedly succeeded")
			}
			if stage == "close_error" {
				code, exit := maintenanceErrorCode(err)
				if code != "maintenance.invalid_config" || exit != 2 || !errors.Is(err, os.ErrClosed) {
					t.Fatal("stdin cleanup lost primary or private close cause")
				}
			}
			if input.Fd() != ^uintptr(0) {
				t.Fatal("stdin file still has an owned handle")
			}
		})
	}
}

func maintenanceRootDatabaseStages(t *testing.T) {
	for _, scenario := range []string{"missing_dsn", "parse", "budget", "connect", "ping", "applied", "already_applied", "unknown", "cleanup"} {
		t.Run(scenario, func(t *testing.T) { maintenanceRootDatabaseStage(t, scenario) })
	}
}

func maintenanceRootDatabaseStage(t *testing.T, scenario string) {
	values := maintenanceRootValues()
	values[databaseDSNEnvironment] = "synthetic-not-a-real-dsn"
	if scenario == "missing_dsn" {
		delete(values, databaseDSNEnvironment)
	}
	config, err := loadMaintenanceConfig(func(k string) string { return values[k] })
	if err != nil {
		t.Fatal(err)
	}
	events := []string{}
	dependencies := maintenanceRootDependencies{
		getenv: func(k string) string { events = append(events, "get:"+k); return values[k] },
		parse: func(string) (*pgxpool.Config, error) {
			events = append(events, "parse")
			if scenario == "parse" {
				return nil, errors.New("private parse")
			}
			max := int32(2)
			if scenario == "budget" {
				max = 0
			}
			return &pgxpool.Config{MaxConns: max, ConnConfig: &pgx.ConnConfig{Config: pgconn.Config{Host: "127.0.0.1"}}}, nil
		}, connect: func(context.Context, *pgxpool.Config) (*pgxpool.Pool, error) {
			events = append(events, "connect")
			if scenario == "connect" {
				return nil, errors.New("private connect")
			}
			return new(pgxpool.Pool), nil
		}, ping: func(context.Context, *pgxpool.Pool) error {
			events = append(events, "ping")
			if scenario == "ping" {
				return errors.New("private ping")
			}
			return nil
		}, state: func(*pgxpool.Pool) (int32, int32) { return 0, 0 },
		closePool: func(*pgxpool.Pool) {
			events = append(events, "pool_close")
			if scenario == "cleanup" {
				panic("private cleanup")
			}
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	execute := func(context.Context, *pgxpool.Pool, int) (maintenanceOutput, error) {
		events = append(events, "execute")
		if scenario == "unknown" {
			return maintenanceOutput{}, &identity.PasswordResetMaintenanceError{Code: identity.PasswordResetMaintenanceOutcomeUnknown}
		}
		return maintenanceOutput{RequestID: "request", OperationID: "operation", AuditID: "audit", Outcome: scenario}, nil
	}
	if scenario == "cleanup" {
		defer func() {
			if recover() == nil {
				t.Error("cleanup panic missing")
			}
		}()
	}
	result, err := executeMaintenanceDatabase(ctx, config, execute, dependencies)
	if scenario == "applied" || scenario == "already_applied" {
		if err != nil || result.Outcome != scenario || events[len(events)-1] != "pool_close" {
			t.Fatalf("events=%v result=%+v err=%v", events, result, err)
		}
	} else if err == nil {
		t.Fatalf("scenario=%s succeeded events=%v", scenario, events)
	}
	for _, forbidden := range map[string][]string{"missing_dsn": {"parse", "connect", "ping", "execute"}, "parse": {"connect", "ping", "execute"}, "budget": {"connect", "ping", "execute"}, "connect": {"ping", "execute"}, "ping": {"execute"}}[scenario] {
		for _, event := range events {
			if event == forbidden {
				t.Fatalf("forbidden event=%s sequence=%v", event, events)
			}
		}
	}
}

type maintenanceShortWriter struct{}

func (maintenanceShortWriter) Write(data []byte) (int, error) {
	return min(1, len(data)), io.ErrShortWrite
}

func maintenanceRootOutputTable(t *testing.T) {
	for _, test := range []struct {
		code string
		exit int
	}{
		{"maintenance.invalid_request", 2}, {"maintenance.invalid_config", 2}, {"maintenance.mode_conflict", 2},
		{"maintenance.unsupported_platform", 3}, {"maintenance.authority_unavailable", 3}, {"maintenance.forbidden", 3},
		{"maintenance.state_conflict", 4}, {"maintenance.outcome_unknown", 5}, {"maintenance.failed", 6}, {"private-unknown", 6},
	} {
		var stdout, stderr bytes.Buffer
		err := &identity.PasswordResetMaintenanceError{Code: identity.PasswordResetMaintenanceCode(test.code)}
		exit := finishMaintenanceCommand(&stdout, &stderr, maintenanceOutput{}, err)
		code := test.code
		if test.exit == 6 {
			code = "maintenance.failed"
		}
		if exit != test.exit || stdout.Len() != 0 || stderr.String() != code+"\n" {
			t.Fatalf("code=%s exit=%d stdout=%q stderr=%q", test.code, exit, stdout.String(), stderr.String())
		}
	}
	for _, outcome := range []string{"applied", "already_applied"} {
		var stdout, stderr bytes.Buffer
		exit := finishMaintenanceCommand(&stdout, &stderr, maintenanceOutput{"request", "operation", outcome, "audit"}, nil)
		want := `{"request_id":"request","operation_id":"operation","outcome":"` + outcome + `","audit_id":"audit"}` + "\n"
		if exit != 0 || stdout.String() != want || stderr.Len() != 0 {
			t.Fatalf("exit=%d stdout=%q stderr=%q", exit, stdout.String(), stderr.String())
		}
	}
	var stderr bytes.Buffer
	if exit := finishMaintenanceCommand(maintenanceShortWriter{}, &stderr, maintenanceOutput{Outcome: "applied"}, nil); exit != 6 || stderr.String() != "maintenance.failed\n" {
		t.Fatalf("short output exit=%d stderr=%q", exit, stderr.String())
	}
}

func TestPasswordResetMaintenanceRoot(t *testing.T) {
	if maintenanceRootHelper(t) {
		return
	}
	t.Run("I-S105-02_05_r49_numeric_endpoints", maintenanceRootR49NumericEndpoints)
	t.Run("I-S105-15_21_r49_cancel_after_success", maintenanceRootR49CanceledSuccess)
	t.Run("I-S105-01_extra_argv_rejected_before_server", func(t *testing.T) {
		command := exec.Command(os.Args[0], "-test.run=^TestPasswordResetMaintenanceRoot$", "--",
			"maintenance", "password-reset", "retry", "extra")
		command.Env = []string{mainHelperEnvironment + "=maintenance-root"}
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		err := command.Run()
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 2 || stdout.Len() != 0 ||
			stderr.String() != "maintenance.invalid_request\n" {
			t.Fatalf("exit=%v stdout=%q stderr=%q", err, stdout.String(), stderr.String())
		}
	})
	t.Run("I-S105-01_mode_table", maintenanceRootModeTable)
	t.Run("I-S105-02_config_pre_dsn", maintenanceRootConfigTable)
	t.Run("I-S105-06_input_boundaries", maintenanceRootInputTable)
	t.Run("I-S105-06_input_properties_100", maintenanceRootInputProperties)
	t.Run("I-S105-11_shared_deadline_properties_100", maintenanceRootDeadlineProperties)
	t.Run("I-S105-15_22_sdk_close_causality", maintenanceRootCloseCausality)
	t.Run("I-S105-22_constructor_lookup_barriers", maintenanceRootConstructorStages)
	t.Run("I-S105-22_late_after_connect", maintenanceRootLateConnection)
	t.Run("I-S105-02_early_returns_close_stdin", maintenanceRootEarlyInputOwnership)
	t.Run("I-S105-02_database_stage_counts", maintenanceRootDatabaseStages)
	t.Run("I-S105-21_output_error_priority", maintenanceRootOutputTable)
	// r48 hard-cutoff FAIL/raw/source snapshots remain in the retained evidence.
	// r49 requires an active-at-C observation, actual join, and a failed result.
	t.Run("I-S105-22_r49_supported_scram_join_overrun", maintenanceRootSCRAMChild)
	t.Run("I-S105-22_r49_local_lookup_join_overrun", func(t *testing.T) { maintenanceRootCutoffChild(t, "lookup") })
	t.Run("I-S105-02_05_r49_literal_lookup_numeric_dial", maintenanceRootR49Network)
	t.Run("I-S105-11_r49_stop_new_business", maintenanceRootR49BusinessStages)
	t.Run("I-S105-11_r49_cancel_owner_returns", maintenanceRootR49CancelOwnership)
	t.Run("I-S105-11_r49_authority_before_D", maintenanceRootR49AuthorityDeadline)
	t.Run("I-S105-11_22_r49_sticky_end", maintenanceRootR49StickyEnd)
	t.Run("I-S105-22_tls_close_backpressure_1s", func(t *testing.T) { maintenanceRootCutoffChild(t, "tls") })
	t.Run("I-S105-22_tls_healthy_close", func(t *testing.T) { maintenanceRootCutoffChild(t, "tls-healthy") })
	t.Run("I-S105-22_cleanup_deadline_limits", maintenanceRootDeadlineClamping)
	t.Run("I-S105-22_cleanup_callback_join", maintenanceRootCleanupCallbackJoin)
}

func maintenanceRootHelper(t *testing.T) bool {
	switch os.Getenv(mainHelperEnvironment) {
	case "maintenance-tls-cutoff":
		maintenanceRootTLSClose(t, true)
	case "maintenance-tls-healthy-cutoff":
		maintenanceRootTLSClose(t, false)
	case "maintenance-scram-cutoff":
		maintenanceRootConstructorDeadline(t, "scram")
	case "maintenance-lookup-cutoff":
		maintenanceRootConstructorDeadline(t, "lookup")
	case "maintenance-root":
		for i, arg := range os.Args {
			if arg == "--" {
				os.Args = append([]string{os.Args[0]}, os.Args[i+1:]...)
				main()
				return true
			}
		}
		t.Fatal("missing helper argument separator")
	default:
		return false
	}
	return true
}

func maintenanceRootDeadlineClamping(t *testing.T) {
	t.Run("property_100", maintenanceRootDeadlineClampProperty)
	for _, method := range []string{"both", "read", "write"} {
		for _, input := range []string{"clear", "extend", "earlier", "equal"} {
			t.Run(method+"/"+input, func(t *testing.T) {
				owner := newMaintenanceRootTestOwner(t)
				client, peer := net.Pipe()
				defer peer.Close()
				cutoff := time.Now().Add(time.Second)
				observed := &maintenanceDeadlineObservation{Conn: client, cutoff: cutoff, sdkStarted: true}
				transport := &maintenanceTransport{Conn: observed, owner: owner}
				defer transport.Close()
				if err := transport.limitCleanupDeadline(cutoff); err != nil {
					t.Fatal(err)
				}
				if err := transport.limitCleanupDeadline(cutoff.Add(time.Hour)); err != nil {
					t.Fatal(err)
				}
				deadline := cutoff
				switch input {
				case "clear":
					deadline = time.Time{}
				case "extend":
					deadline = cutoff.Add(time.Hour)
				case "earlier":
					deadline = cutoff.Add(-time.Millisecond)
				}
				setter := transport.SetDeadline
				if method == "read" {
					setter = transport.SetReadDeadline
				}
				if method == "write" {
					setter = transport.SetWriteDeadline
				}
				if err := setter(deadline); err != nil {
					t.Fatal(err)
				}
				if observed.extended || transport.cleanupCutoff != cutoff {
					t.Fatal("cleanup deadline cleared or relaxed")
				}
			})
		}
	}
}

func maintenanceRootDeadlineClampProperty(t *testing.T) {
	property := func(budget uint16, offset int32, clear bool) bool {
		cutoff := time.Unix(1700000000, 0).Add(time.Duration(budget%60+1) * time.Second)
		requested := cutoff.Add(time.Duration(offset) * time.Microsecond)
		if clear {
			requested = time.Time{}
		}
		transport := &maintenanceTransport{cleanupCutoff: cutoff}
		got := transport.clampDeadline(requested)
		return !got.IsZero() && !got.After(cutoff) &&
			(requested.IsZero() || !requested.Before(cutoff) || got.Equal(requested))
	}
	if err := quick.Check(property, &quick.Config{MaxCount: 100, Rand: mathrand.New(mathrand.NewSource(105002))}); err != nil {
		t.Fatal(err)
	}
}

func maintenanceRootCleanupCallbackJoin(t *testing.T) {
	for _, scenario := range []string{"stopped_before_cancel", "already_completed", "running_joined"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			entered, release, exited := make(chan struct{}), make(chan struct{}), make(chan struct{})
			stop := watchMaintenanceCleanup(ctx, func() { defer close(exited); close(entered); <-release })
			if scenario == "stopped_before_cancel" {
				stop()
				cancel()
				stop()
				select {
				case <-entered:
					t.Fatal("stopped callback executed")
				default:
				}
				return
			}
			if scenario == "already_completed" {
				close(release)
			}
			cancel()
			<-entered
			if scenario == "already_completed" {
				<-exited
				stop()
				stop()
				return
			}
			invoked, returned := make(chan struct{}), make(chan struct{})
			go func() { close(invoked); stop(); close(returned) }()
			<-invoked
			select {
			case <-returned:
				t.Fatal("callback was abandoned")
			default:
			}
			close(release)
			<-returned
			select {
			case <-exited:
			default:
				t.Fatal("stop returned before callback completion")
			}
			stop()
		})
	}
}

type maintenanceDeadlineObservation struct {
	net.Conn
	mu           sync.Mutex
	cutoff       time.Time
	extended     bool
	sdkStarted   bool
	rawBeforeSDK bool
	closes       int
}

func (connection *maintenanceDeadlineObservation) recordDeadline(deadline time.Time) {
	connection.mu.Lock()
	if !connection.cutoff.IsZero() && (deadline.IsZero() || deadline.After(connection.cutoff)) {
		connection.extended = true
	}
	connection.mu.Unlock()
}

func (connection *maintenanceDeadlineObservation) SetDeadline(deadline time.Time) error {
	connection.recordDeadline(deadline)
	return connection.Conn.SetDeadline(deadline)
}

func (connection *maintenanceDeadlineObservation) SetWriteDeadline(deadline time.Time) error {
	connection.recordDeadline(deadline)
	return connection.Conn.SetWriteDeadline(deadline)
}

func (connection *maintenanceDeadlineObservation) SetReadDeadline(deadline time.Time) error {
	connection.recordDeadline(deadline)
	return connection.Conn.SetReadDeadline(deadline)
}

func (connection *maintenanceDeadlineObservation) Close() error {
	connection.mu.Lock()
	connection.closes++
	connection.rawBeforeSDK = connection.rawBeforeSDK || !connection.sdkStarted
	connection.mu.Unlock()
	return connection.Conn.Close()
}

func maintenanceRootTLSCertificate(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), cryptorand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	template := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(cryptorand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

func maintenanceRootTLSPeer(ctx context.Context, raw net.Conn, certificate tls.Certificate, ready chan<- struct{}, release <-chan struct{}, backpressure bool) error {
	defer raw.Close()
	server := tls.Server(raw, &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS12, SessionTicketsDisabled: true})
	if err := server.HandshakeContext(ctx); err != nil {
		return err
	}
	backend := pgproto3.NewBackend(server, server)
	if _, err := backend.ReceiveStartupMessage(); err != nil {
		return err
	}
	backend.Send(&pgproto3.AuthenticationOk{})
	backend.Send(&pgproto3.ParameterStatus{Name: "client_encoding", Value: "UTF8"})
	backend.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
	if err := backend.Flush(); err != nil {
		return err
	}
	close(ready)
	if backpressure {
		<-release
		return nil
	}
	message, err := backend.Receive()
	if err != nil {
		return err
	}
	if _, ok := message.(*pgproto3.Terminate); !ok {
		return errors.New("expected Terminate, not SQL")
	}
	var buffer [1]byte
	_, err = server.Read(buffer[:])
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

type maintenanceTLSFixture struct {
	owner       *maintenanceConnectionOwner
	pool        *pgxpool.Pool
	raw         *maintenanceDeadlineObservation
	ready       chan struct{}
	release     chan struct{}
	peerDone    chan error
	peerJoined  chan struct{}
	releaseOnce sync.Once
	started     atomic.Bool
}

func maintenanceRootTLSFixture(t *testing.T, backpressure bool) *maintenanceTLSFixture {
	t.Helper()
	certificate := maintenanceRootTLSCertificate(t)
	ctx, cancel := context.WithTimeout(context.Background(), 9*time.Second)
	fixture := &maintenanceTLSFixture{ready: make(chan struct{}), release: make(chan struct{}), peerDone: make(chan error, 1), peerJoined: make(chan struct{})}
	fixture.owner = newMaintenanceConnectionOwner(ctx, time.Second)
	t.Cleanup(func() { fixture.finish(); cancel() })
	config, err := pgxpool.ParseConfig("host=127.0.0.1 user=synthetic dbname=synthetic sslmode=disable pool_max_conns=1")
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.TLSConfig = &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12, MaxVersion: tls.VersionTLS12} // In-memory synthetic peer only.
	config.ConnConfig.SSLNegotiation = "direct"
	config.ConnConfig.LookupFunc = func(context.Context, string) ([]string, error) { return []string{"127.0.0.1"}, nil }
	config.ConnConfig.DialFunc = func(context.Context, string, string) (net.Conn, error) {
		if !fixture.started.CompareAndSwap(false, true) {
			return nil, errors.New("extra synthetic dial refused")
		}
		client, peer := net.Pipe()
		fixture.raw = &maintenanceDeadlineObservation{Conn: client}
		go func() {
			defer close(fixture.peerJoined)
			fixture.peerDone <- maintenanceRootTLSPeer(ctx, peer, certificate, fixture.ready, fixture.release, backpressure)
		}()
		return fixture.raw, nil
	}
	fixture.owner.configure(config)
	fixture.pool, err = pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	borrowed, err := fixture.pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	borrowed.Release()
	<-fixture.ready
	return fixture
}

func (fixture *maintenanceTLSFixture) finish() {
	fixture.releaseOnce.Do(func() { close(fixture.release) })
	fixture.owner.stopWatch()
	fixture.owner.interrupt(nil)
	if fixture.pool != nil {
		fixture.pool.Close()
	}
	if fixture.started.Load() {
		<-fixture.peerJoined
	}
}

func maintenanceRootTLSClose(t *testing.T, backpressure bool) {
	fixture := maintenanceRootTLSFixture(t, backpressure)
	owner, pool := fixture.owner, fixture.pool
	owner.mu.Lock()
	var entry *maintenanceCloseEntry
	for _, current := range owner.connections {
		entry = current
	}
	owner.mu.Unlock()
	if entry == nil {
		t.Fatal("missing real SDK connection")
	}
	var sdkCalls atomic.Int32
	originalClose := entry.closeSDK
	entry.closeSDK = func(ctx context.Context) error {
		cutoff, _ := ctx.Deadline()
		fixture.raw.mu.Lock()
		fixture.raw.cutoff, fixture.raw.sdkStarted = cutoff, true
		fixture.raw.mu.Unlock()
		sdkCalls.Add(1)
		return originalClose(ctx)
	}
	start := time.Now()
	cleanupErr := owner.cleanup(func() (int32, int32) { state := pool.Stat(); return state.ConstructingConns(), state.AcquiredConns() }, pool.Close)
	elapsed := time.Since(start)
	fixture.raw.mu.Lock()
	extended, rawFirst, closes := fixture.raw.extended, fixture.raw.rawBeforeSDK, fixture.raw.closes
	fixture.raw.mu.Unlock()
	select {
	case <-entry.cleanupDone:
	default:
		t.Fatal("raw close did not finish SDK cleanup")
	}
	select {
	case <-entry.finished:
	default:
		t.Fatal("SDK close entry not joined")
	}
	owner.mu.Lock()
	stages, sockets := len(owner.stages), len(owner.sockets)
	owner.mu.Unlock()
	fixture.finish()
	if peerErr := <-fixture.peerDone; peerErr != nil {
		t.Error(peerErr)
	}
	t.Logf("tls=1.2 sdk=pgx/v5.11.0 backpressure=%t cleanup_budget_ms=1000 elapsed_ms=%d cleanup_error=%t deadline_extended=%t raw_before_sdk=%t sdk_close_calls=%d raw_close_calls=%d final_stages=%d final_sockets=%d sdk_done=true pool_close_returned=true peer_joined=true", backpressure, elapsed.Milliseconds(), cleanupErr != nil, extended, rawFirst, sdkCalls.Load(), closes, stages, sockets)
	// The tolerance measures scheduler/return overhead; every effective deadline must still be <= the original cutoff.
	if elapsed > time.Second+250*time.Millisecond || extended || rawFirst || sdkCalls.Load() != 1 || closes != 1 || stages != 0 || sockets != 0 || (cleanupErr != nil) != backpressure {
		t.Fatal("TLS SDK close violated original cleanup cutoff/ownership")
	}
}

func maintenanceRootSCRAMChild(t *testing.T) {
	maintenanceRootCutoffChild(t, "scram")
}

func maintenanceRootCutoffChild(t *testing.T, kind string) {
	// Race instrumentation makes the supported 10M-iteration CPU stage much
	// slower. This is a diagnostic ceiling, not a production D/S/C extension.
	command := exec.Command(os.Args[0], "-test.run=^TestPasswordResetMaintenanceRoot$", "-test.v", "-test.timeout=180s")
	// The isolated process has no deployment credentials or ambient PG configuration.
	command.Env = []string{mainHelperEnvironment + "=maintenance-" + kind + "-cutoff"}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("constructor cutoff diagnostic: %v\n%s", err, output)
	}
	t.Log(string(output))
}

func maintenanceRootSCRAMPeer(connection net.Conn, sent chan<- struct{}) error {
	defer connection.Close()
	backend := pgproto3.NewBackend(connection, connection)
	if _, err := backend.ReceiveStartupMessage(); err != nil {
		return err
	}
	backend.Send(&pgproto3.AuthenticationSASL{AuthMechanisms: []string{"SCRAM-SHA-256"}})
	if err := backend.Flush(); err != nil {
		return err
	}
	if err := backend.SetAuthType(pgproto3.AuthTypeSASL); err != nil {
		return err
	}
	message, err := backend.Receive()
	if err != nil {
		return err
	}
	initial, ok := message.(*pgproto3.SASLInitialResponse)
	if !ok {
		return errors.New("expected synthetic SASL initial response")
	}
	_, nonce, ok := strings.Cut(string(initial.Data), ",r=")
	if !ok {
		return errors.New("missing synthetic nonce")
	}
	backend.Send(&pgproto3.AuthenticationSASLContinue{Data: []byte("r=" + nonce + "peer,s=cHVibGljLXNhbHQ=,i=10000000")})
	if err := backend.Flush(); err != nil {
		return err
	}
	close(sent)
	_, _ = backend.Receive() // Socket close ends the peer; no SQL, listener, or database exists.
	return nil
}

type maintenanceCutoffObservation struct {
	constructing int32
	stages       int
	sockets      int
	at           time.Time
}

func maintenanceRootConstructorDeadline(t *testing.T, kind string) {
	config, err := pgxpool.ParseConfig("host=127.0.0.1 user=synthetic dbname=synthetic password=synthetic sslmode=disable pool_max_conns=1")
	if err != nil {
		t.Fatal(err)
	}
	sent, peerDone := make(chan struct{}), make(chan error, 1)
	peerJoined := make(chan struct{})
	lookupRelease := make(chan struct{})
	var dialed atomic.Bool
	config.ConnConfig.LookupFunc = func(context.Context, string) ([]string, error) { t.Error("inherited resolver called"); return nil, nil }
	config.ConnConfig.DialFunc = func(context.Context, string, string) (net.Conn, error) {
		if !dialed.CompareAndSwap(false, true) {
			return nil, errors.New("additional synthetic dial refused")
		}
		client, peer := net.Pipe()
		go func() { defer close(peerJoined); peerDone <- maintenanceRootSCRAMPeer(peer, sent) }()
		return client, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	owner := newMaintenanceConnectionOwner(ctx, time.Second)
	owner.configure(config)
	if kind == "lookup" {
		before := config.BeforeConnect
		config.BeforeConnect = func(ctx context.Context, config *pgx.ConnConfig) error {
			if err := before(ctx, config); err != nil {
				return err
			}
			literal := config.LookupFunc
			config.LookupFunc = func(ctx context.Context, host string) ([]string, error) {
				stage, finish, err := owner.stage(ctx, config.Tracer.(*maintenanceConnectTracer).attempt)
				if err != nil {
					return nil, err
				}
				defer finish()
				if _, err := literal(stage, host); err != nil {
					return nil, err
				}
				<-stage.Done()
				<-lookupRelease
				return nil, stage.Err()
			}
			return nil
		}
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		owner.stopWatch()
		t.Fatal(err)
	}
	observed := make(chan maintenanceCutoffObservation, 1)
	observerDone := make(chan struct{})
	cutoff := owner.deadline.Add(time.Second)
	timer := time.AfterFunc(time.Until(cutoff), func() {
		defer close(observerDone)
		stat := pool.Stat()
		owner.mu.Lock()
		snapshot := maintenanceCutoffObservation{stat.ConstructingConns(), len(owner.stages), len(owner.sockets), time.Now()}
		owner.mu.Unlock()
		observed <- snapshot
		if kind == "lookup" {
			close(lookupRelease)
		}
	})
	defer timer.Stop()
	connection, acquireErr := pool.Acquire(ctx)
	if connection != nil {
		connection.Release()
	}
	peerReceived := false
	var peerErr error
	if kind == "scram" {
		select {
		case <-sent:
		case peerErr = <-peerDone:
			peerReceived = true
			// Both may already be ready after Acquire cancellation. A completed
			// peer is only premature if it never sent the SCRAM continuation.
			select {
			case <-sent:
			default:
				t.Errorf("synthetic peer before continuation: %v; acquire=%v", peerErr, acquireErr)
			}
		}
	}
	cleanupErr := owner.cleanup(func() (int32, int32) { stat := pool.Stat(); return stat.ConstructingConns(), stat.AcquiredConns() }, pool.Close)
	snapshot := <-observed
	<-observerDone
	if kind == "scram" && !peerReceived {
		peerErr = <-peerDone
	}
	if peerErr != nil {
		t.Error(peerErr)
	}
	if kind == "scram" {
		<-peerJoined
	}
	owner.mu.Lock()
	finalStages, finalSockets := len(owner.stages), len(owner.sockets)
	sticky := errors.Is(owner.cleanupErr, context.DeadlineExceeded)
	owner.mu.Unlock()
	var stdout, stderr bytes.Buffer
	resultErr := maintenanceFinalError(nil, cleanupErr, ctx)
	exit := finishMaintenanceCommand(&stdout, &stderr, maintenanceOutput{Outcome: "applied"}, resultErr)
	t.Logf("kind=%s acquire_canceled=%t cleanup_deadline=%t constructing_at_cutoff=%d stages_at_cutoff=%d sockets_at_cutoff=%d observed_after_cutoff=%s joined_after_cutoff=%s final_constructing=%d final_stages=%d final_sockets=%d peer_joined=%t pool_close_returned=true",
		kind, errors.Is(acquireErr, context.DeadlineExceeded), errors.Is(cleanupErr, context.DeadlineExceeded), snapshot.constructing, snapshot.stages, snapshot.sockets, snapshot.at.Sub(cutoff), time.Since(cutoff), pool.Stat().ConstructingConns(), finalStages, finalSockets, kind == "scram")
	t.Logf("r49 exit=%d stdout_empty=%t sticky_overrun=%t final_acquired=%d dialed=%t", exit, stdout.Len() == 0, sticky, pool.Stat().AcquiredConns(), dialed.Load())
	if snapshot.constructing == 0 || snapshot.stages == 0 || finalStages != 0 || finalSockets != 0 ||
		pool.Stat().ConstructingConns() != 0 || pool.Stat().AcquiredConns() != 0 || !sticky ||
		!errors.Is(cleanupErr, context.DeadlineExceeded) || exit != 6 || stdout.Len() != 0 || stderr.String() != "maintenance.failed\n" || (kind == "lookup" && dialed.Load()) {
		t.Fatal("r49 requires an active-at-C observation, actual final join, and sticky failed/6")
	}
}

func TestProductionAdminArticleQuery(t *testing.T) {
	t.Run("PrincipalBoundary", adminRootPrincipalBoundary)
	t.Run("ProductionSharedChain", adminRootSharedChain)
	t.Run("ExactMethodsAndCoexistence", adminRootExactRoutes)
	t.Run("OldConsumerCompatibility", adminRootOldConsumers)
}

func adminRootExactRoutes(t *testing.T) {
	t.Run("ExactMethodsAndCoexistence", func(t *testing.T) {
		for _, route := range []struct{ name, path string }{
			{"list", "/api/v1/admin/articles"},
			{"detail", "/api/v1/admin/articles/10000000-0000-4000-8000-000000000001"},
		} {
			t.Run(route.name, func(t *testing.T) {
				handler := newArticleProductionHandlerWithTestSeam(t, nil)
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, route.path, nil))
				if response.Code != http.StatusUnauthorized {
					t.Fatalf("registered GET status=%d, want session boundary 401", response.Code)
				}
			})
		}
	})
	t.Run("method_path_matrix", adminRootRouteMatrix)
	t.Run("missing_handlers", adminRootMissingHandlers)
}

type adminRootRepository struct {
	events *[]string
	rows   []article.Article
	lists  int
	gets   int
	err    error
}

func (r *adminRootRepository) ListCandidateBatch(_ context.Context, after *article.ArticleCursor, limit int) ([]article.Article, error) {
	r.lists++
	*r.events = append(*r.events, "repository")
	if r.err != nil {
		return nil, r.err
	}
	rows := []article.Article{}
	for _, row := range r.rows {
		if after == nil || row.CreatedAt.Before(after.CreatedAt) || row.CreatedAt.Equal(after.CreatedAt) && row.ID < after.ID {
			rows = append(rows, row)
		}
		if len(rows) == limit {
			break
		}
	}
	return rows, nil
}

func (r *adminRootRepository) GetArticle(_ context.Context, id string) (article.Article, error) {
	r.gets++
	*r.events = append(*r.events, "repository")
	if r.err != nil {
		return article.Article{}, r.err
	}
	for _, row := range r.rows {
		if row.ID == id {
			return row, nil
		}
	}
	return article.Article{}, pgx.ErrNoRows
}

func (r *adminRootRepository) ListOrphanBatch(ctx context.Context, after *article.ArticleCursor, limit int) ([]article.Article, error) {
	rows, err := r.ListCandidateBatch(ctx, after, limit)
	orphans := []article.Article{}
	for _, row := range rows {
		if row.AuthorID == nil && row.PublicationState == article.PublicationHeld {
			orphans = append(orphans, row)
		}
	}
	return orphans, err
}

type adminRootReader struct {
	events *[]string
	calls  int
	states map[string]string
}

func (r *adminRootReader) GetAuthorStates(context.Context, []string) (map[string]string, error) {
	r.calls++
	*r.events = append(*r.events, "identity")
	return r.states, nil
}

type adminRootFixture struct {
	handler      http.Handler
	resolver     *articleAuthorRootResolver
	repository   *adminRootRepository
	reader       *adminRootReader
	events       []string
	traceID      string
	observations []articleSharedDependency
}

func newAdminRootFixture(t *testing.T, controlled bool) *adminRootFixture {
	t.Helper()
	f := &adminRootFixture{}
	f.resolver = &articleAuthorRootResolver{events: &f.events, principal: session.Principal{
		SessionID: "session-admin", UserID: "30000000-0000-4000-8000-000000000003", Role: "admin", Status: "active",
	}}
	seam := &articleProductionTestSeam{principalResolver: f.resolver,
		shared: func(value articleSharedDependency) { f.observations = append(f.observations, value) },
		trace: func(r *http.Request) {
			f.traceID, _ = platform.TraceIDFromContext(r.Context())
			f.events = append(f.events, "trace")
		},
	}
	if controlled {
		f.installReaders(t, seam)
	}
	f.handler = newArticleProductionHandlerWithTestSeam(t, seam)
	return f
}

func (f *adminRootFixture) installReaders(t *testing.T, seam *articleProductionTestSeam) {
	t.Helper()
	author := "20000000-0000-4000-8000-000000000002"
	now := time.Date(2026, 10, 28, 0, 0, 0, 123456000, time.UTC)
	clear := article.Article{ID: articleAuthorRootID, AuthorID: &author, Title: " 标题 <b>literal</b> ", Body: " **正文🙂** ",
		PublicationState: article.PublicationPublished, ViolationState: article.ViolationClear, DeletionState: article.DeletionActive,
		Version: 2, CreatedAt: now, UpdatedAt: now}
	held := clear
	held.ID, held.AuthorID, held.OrphanedFromUserID = "10000000-0000-4000-8000-000000000002", nil, &author
	held.PublicationState, held.ViolationState = article.PublicationHeld, article.ViolationViolating
	category := article.ViolationViolence
	held.ViolationCategory = &category
	f.repository = &adminRootRepository{events: &f.events, rows: []article.Article{held, clear}}
	f.reader = &adminRootReader{events: &f.events, states: map[string]string{author: "active"}}
	config := article.QueryConfig{DefaultLimit: 2, MaxLimit: 2, CandidateBatchSize: 3}
	service, err := article.NewArticleQueryService(f.repository, f.reader, config)
	if err != nil {
		t.Fatal(err)
	}
	f.installControlledHandlers(t, seam, service, config)
}

func (f *adminRootFixture) installControlledHandlers(t *testing.T, seam *articleProductionTestSeam, service *article.ArticleQueryService, config article.QueryConfig) {
	t.Helper()
	admin, err := article.NewAdminQueryHandler(service)
	if err != nil {
		t.Fatal(err)
	}
	member, err := article.NewQueryHandler(service)
	if err != nil {
		t.Fatal(err)
	}
	author, err := article.NewAuthorDetailHandler(service)
	if err != nil {
		t.Fatal(err)
	}
	orphans, err := article.NewOrphanQueryService(f.repository, config)
	if err != nil {
		t.Fatal(err)
	}
	orphan, err := article.NewOrphanQueryHandler(orphans)
	if err != nil {
		t.Fatal(err)
	}
	seam.adminQueryHandlers = [2]http.Handler{http.HandlerFunc(admin.ListAdminArticles), http.HandlerFunc(admin.GetAdminArticle)}
	seam.queryHandlers = [2]http.Handler{http.HandlerFunc(member.ListArticles), http.HandlerFunc(member.GetArticle)}
	seam.authorDetailHandler = http.HandlerFunc(author.GetAuthorArticle)
	seam.orphanHandlers[0], seam.orphanHandlers[1] = http.HandlerFunc(orphan.ListOrphanArticles), http.HandlerFunc(orphan.GetOrphanArticle)
}

func (f *adminRootFixture) request(path string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, path, nil)
	r.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: "admin-session-token"})
	return r
}

func (f *adminRootFixture) serve(t *testing.T, r *http.Request, status int) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	if w.Code != status {
		t.Fatalf("HTTP=%d expected=%d (body withheld)", w.Code, status)
	}
	if f.traceID == "" || w.Header().Get(platform.TraceHeader) != f.traceID {
		t.Fatal("production Trace was not outermost")
	}
	return w
}

func adminRootPrincipalBoundary(t *testing.T) {
	for _, endpoint := range []string{"", "/" + articleAuthorRootID} {
		t.Run(map[bool]string{true: "list", false: "detail"}[endpoint == ""], func(t *testing.T) {
			for _, name := range []string{"ordinary_admin", "no_cookie", "invalid_session", "resolver_database", "member", "suspended", "deleted", "password_onboarding", "privacy_onboarding"} {
				t.Run(name, func(t *testing.T) { adminRootPrincipalCase(t, endpoint, name) })
			}
		})
	}
}

func adminRootPrincipalCase(t *testing.T, endpoint, name string) {
	f := newAdminRootFixture(t, true)
	r := f.request("/api/v1/admin/articles" + endpoint + "?role=admin")
	status, code := 403, platform.CodeAuthForbidden
	switch name {
	case "ordinary_admin":
		status = 200
		r.Header.Set("Origin", "https://untrusted.example.invalid")
	case "no_cookie":
		r.Header.Del("Cookie")
		status, code = 401, platform.CodeAuthSessionRequired
	case "invalid_session":
		f.resolver.err = session.ErrSessionRequired
		status, code = 401, platform.CodeAuthSessionRequired
	case "resolver_database":
		f.resolver.err = session.ErrStoreUnavailable
		status, code = 503, platform.CodePlatformDatabaseUnavailable
	case "member":
		f.resolver.principal.Role = "member"
	case "suspended", "deleted":
		f.resolver.principal.Status = name
	case "password_onboarding", "privacy_onboarding":
		f.resolver.principal.MustChangePassword = name == "password_onboarding"
		f.resolver.principal.MustAcknowledgePrivacy = name == "privacy_onboarding"
		code = platform.CodeAuthMustCompleteOnboarding
	}
	w := f.serve(t, r, status)
	if status != 200 {
		adminRootAssertError(t, w, code)
		if f.repository.lists+f.repository.gets+f.reader.calls != 0 {
			t.Fatal("rejected Principal reached article downstream")
		}
	} else if f.resolver.principal.IsInitialAdmin || f.repository.lists+f.repository.gets != 1 {
		t.Fatal("ordinary administrator did not reach real query Handler")
	}
	for _, input := range f.resolver.inputs {
		if input.RequireCSRF || input.CSRFToken != "" || input.Token != "admin-session-token" || input.Now.IsZero() {
			t.Fatal("GET resolver input changed")
		}
	}
	adminRootPrincipalEvents(t, f, name)
}

func adminRootPrincipalEvents(t *testing.T, f *adminRootFixture, name string) {
	t.Helper()
	want := []string{"trace", "session", "principal"}
	switch name {
	case "no_cookie":
		want = want[:1]
	case "invalid_session", "resolver_database":
		want = want[:2]
	case "ordinary_admin":
		want = append(want, "repository", "identity")
	}
	if !reflect.DeepEqual(f.events, want) {
		t.Fatalf("Principal short circuit events=%v, expected=%v", f.events, want)
	}
}

func adminRootAssertError(t *testing.T, w *httptest.ResponseRecorder, code platform.Code) {
	t.Helper()
	var e platform.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &e); err != nil {
		t.Fatal(err)
	}
	if e.Code != code || e.TraceID == "" || e.TraceID != w.Header().Get(platform.TraceHeader) || w.Header().Get("ETag") != "" || strings.Contains(w.Body.String(), "items") {
		t.Fatal("error envelope leaked data or changed code/trace")
	}
}

func adminRootSharedChain(t *testing.T) {
	t.Run("actual_factory_references", adminRootActualFactory)
	t.Run("actual_handlers_reject_before_sql", adminRootActualRejections)
	t.Run("real_handler_success", adminRootHandlerSuccess)
	t.Run("query_errors_stop", adminRootQueryErrors)
}

func adminRootActualRejections(t *testing.T) {
	for _, path := range []string{"/api/v1/admin/articles", "/api/v1/admin/articles/" + articleAuthorRootID} {
		f := newAdminRootFixture(t, false)
		f.resolver.principal.Role = "member"
		adminRootAssertError(t, f.serve(t, f.request(path), 403), platform.CodeAuthForbidden)
		if !reflect.DeepEqual(f.events, []string{"trace", "session", "principal"}) {
			t.Fatal("actual admin Handler failed to reject before article SQL")
		}
	}
	f := newAdminRootFixture(t, false)
	adminRootAssertError(t, f.serve(t, f.request("/api/v1/admin/articles?other=%ZZ"), 422), article.CodeArticleValidationFailed)
	adminRootAssertError(t, f.serve(t, f.request("/api/v1/admin/articles/invalid"), 404), article.CodeArticleUnavailable)
}

func (f *adminRootFixture) lookup(t *testing.T, kind, consumer string) any {
	t.Helper()
	var found []any
	for _, item := range f.observations {
		if item.Kind == kind && item.Consumer == consumer {
			found = append(found, item.Instance)
		}
	}
	if len(found) != 1 || found[0] == nil {
		t.Fatalf("construction reference count for %s/%s=%d", kind, consumer, len(found))
	}
	return found[0]
}

func adminRootPrivateReference(value any, field string) uintptr {
	v := reflect.ValueOf(value).Elem().FieldByName(field)
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	return v.Pointer()
}

func adminRootActualFactory(t *testing.T) {
	f := newAdminRootFixture(t, false)
	service := f.lookup(t, "article_query_service", "article_query_handler").(*article.ArticleQueryService)
	if f.lookup(t, "article_query_service", "article_admin_query_handler") != service || f.lookup(t, "article_query_service", "article_author_detail_handler") != service {
		t.Fatal("admin/member/author constructors received different query services")
	}
	for _, entry := range []struct{ kind, field string }{{"article_admin_query_handler", "adminQuery"}, {"article_query_handler", "query"}, {"article_author_detail_handler", "authorReader"}} {
		handler := f.lookup(t, entry.kind, "router").(*article.Handler)
		if adminRootPrivateReference(handler, entry.field) != reflect.ValueOf(service).Pointer() {
			t.Fatal("actual Handler constructor did not retain observed service")
		}
	}
	repo := f.lookup(t, "article_query_repository", "article_query_service").(*article.PostgresArticleQueryRepository)
	reader := f.lookup(t, "article_author_reader", "article_query_service").(*identity.ArticleIdentityReader)
	pool := f.lookup(t, "pool", "article_query_repository").(*pgxpool.Pool)
	if pool != f.lookup(t, "pool", "article_author_reader") || pool != f.lookup(t, "pool", "login") || adminRootPrivateReference(repo, "database") != reflect.ValueOf(pool).Pointer() || adminRootPrivateReference(reader, "database") != reflect.ValueOf(pool).Pointer() || adminRootPrivateReference(service, "repository") != reflect.ValueOf(repo).Pointer() {
		t.Fatal("query/session constructors split shared pool/repository/reader")
	}
	config := reflect.ValueOf(service).Elem().FieldByName("config")
	for key, value := range map[string]int64{"DefaultLimit": 2, "MaxLimit": 2, "CandidateBatchSize": 3} {
		if config.FieldByName(key).Int() != value {
			t.Fatal("query factory changed shared fixture config")
		}
	}
	store := f.lookup(t, "session_store", "authorizer").(*session.Store)
	if store != f.lookup(t, "session_store", "login") || adminRootPrivateReference(store, "transactions") != reflect.ValueOf(pool).Pointer() {
		t.Fatal("production authorizer and login split actual SessionStore/pool")
	}
	if len(f.resolver.inputs) != 0 {
		t.Fatal("construction executed a session read")
	}
}

func adminRootHandlerSuccess(t *testing.T) {
	for _, path := range []string{"?view=all", "?view=violating", "?view=hidden", "/" + articleAuthorRootID} {
		t.Run(map[string]string{"?view=all": "all", "?view=violating": "violating", "?view=hidden": "hidden", "/" + articleAuthorRootID: "detail"}[path], func(t *testing.T) {
			f := newAdminRootFixture(t, true)
			w := f.serve(t, f.request("/api/v1/admin/articles"+path), 200)
			if !reflect.DeepEqual(f.events, []string{"trace", "session", "principal", "repository", "identity"}) {
				t.Fatalf("real Handler chain=%v", f.events)
			}
			if strings.HasPrefix(path, "/") {
				adminRootPayload(t, w.Body.Bytes(), f.repository.rows[1], []article.HiddenReason{})
				if w.Header().Get("ETag") != `"article-`+articleAuthorRootID+`-v2"` {
					t.Fatal("same-record detail ETag changed")
				}
				return
			}
			var page struct {
				Items []json.RawMessage `json:"items"`
				Next  *string           `json:"next_cursor"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
				t.Fatal(err)
			}
			count := 1
			if path == "?view=all" {
				count = 2
			}
			if len(page.Items) != count || page.Next != nil || w.Header().Get("ETag") != "" {
				t.Fatal("root list envelope/view changed")
			}
			adminRootPayload(t, page.Items[0], f.repository.rows[0], []article.HiddenReason{article.HiddenViolation, article.HiddenOrphanHold})
			if count == 2 {
				adminRootPayload(t, page.Items[1], f.repository.rows[1], []article.HiddenReason{})
			}
		})
	}
}

func adminRootPayload(t *testing.T, raw []byte, row article.Article, reasons []article.HiddenReason) {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	want := []string{"id", "title", "body", "author", "publication_state", "violation_state", "violation_category", "hidden_reasons", "orphaned_from_user_id", "version", "created_at", "updated_at"}
	if len(fields) != len(want) {
		t.Fatal("root Admin12 key count changed")
	}
	for _, key := range want {
		if _, ok := fields[key]; !ok {
			t.Fatal("root Admin12 key missing")
		}
	}
	var got article.AdminArticleView
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != row.ID || got.Title != row.Title || got.Body != row.Body || got.Version != row.Version || got.PublicationState != row.PublicationState || got.ViolationState != row.ViolationState || !reflect.DeepEqual(got.ViolationCategory, row.ViolationCategory) || !reflect.DeepEqual(got.OrphanedFromUserID, row.OrphanedFromUserID) || !reflect.DeepEqual(got.HiddenReasons, reasons) || !got.CreatedAt.Equal(row.CreatedAt) || !got.UpdatedAt.Equal(row.UpdatedAt) {
		t.Fatal("root Admin12 values differ from independent fixture")
	}
	if row.AuthorID == nil && (got.Author.Kind != "anonymous" || got.Author.Label != "匿名" || got.Author.ID != nil) || row.AuthorID != nil && (got.Author.Kind != "user" || got.Author.ID == nil || *got.Author.ID != *row.AuthorID || got.Author.Label != "") {
		t.Fatal("root nested author values changed")
	}
}

func adminRootQueryErrors(t *testing.T) {
	for _, name := range []string{"raw_query", "database", "identity", "invalid_state", "pending"} {
		t.Run(name, func(t *testing.T) {
			f := newAdminRootFixture(t, true)
			path, status, code := "/api/v1/admin/articles/"+articleAuthorRootID, 503, platform.CodePlatformDatabaseUnavailable
			switch name {
			case "raw_query":
				path, status, code = "/api/v1/admin/articles?other=%ZZ", 422, article.CodeArticleValidationFailed
			case "database":
				f.repository.err = errors.New("synthetic query failure")
			case "identity":
				f.reader.states = nil
				code = platform.CodePlatformDependencyUnavailable
			case "invalid_state":
				f.repository.rows[1].Version = 0
				status, code = 500, platform.CodePlatformInternalError
			case "pending":
				f.repository.rows[1].DeletionState = article.DeletionPending
				status, code = 404, article.CodeArticleUnavailable
			}
			w := f.serve(t, f.request(path), status)
			adminRootAssertError(t, w, code)
			if name == "raw_query" && f.repository.lists+f.repository.gets != 0 || name != "identity" && f.reader.calls != 0 {
				t.Fatal("error did not stop subsequent article calls")
			}
		})
	}
}

func adminRootRouteMatrix(t *testing.T) {
	for _, target := range []struct{ name, path, allow string }{
		{"list", "/api/v1/admin/articles", "GET"},
		{"detail", "/api/v1/admin/articles/" + articleAuthorRootID, "GET, DELETE"},
		{"violation", "/api/v1/admin/articles/" + articleAuthorRootID + "/violation", "POST"},
		{"restore", "/api/v1/admin/articles/" + articleAuthorRootID + "/restore", "POST"},
	} {
		t.Run(target.name, func(t *testing.T) {
			for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
				t.Run(method, func(t *testing.T) { adminRootMethodCase(t, method, target.path, target.allow) })
			}
		})
	}
	for index, path := range []string{"/api/v1/admin/articles/", "/api/v1/admin/articles//violation", "/api/v1/admin/articles/" + articleAuthorRootID + "/extra", "/api/v1/admin/articlesx", "/api/v1/admin/articles/" + articleAuthorRootID + "/restore/extra"} {
		t.Run(fmt.Sprintf("unknown_%d", index), func(t *testing.T) {
			f := newAdminRootFixture(t, true)
			w := f.serve(t, f.request(path), 404)
			adminRootAssertError(t, w, platform.CodeRequestRouteNotFound)
			if !reflect.DeepEqual(f.events, []string{"trace"}) || w.Header().Get("Allow") != "" || w.Header().Get("Location") != "" {
				t.Fatal("unknown path redirected/reached security")
			}
		})
	}
}

func adminRootMethodCase(t *testing.T, method, path, allow string) {
	f := newAdminRootFixture(t, true)
	r := f.request(path)
	r.Method = method
	allowed := strings.Contains(", "+allow+", ", ", "+method+", ")
	status := 405
	if allowed {
		status = 200
		if method != http.MethodGet {
			status = 403
		}
	}
	w := f.serve(t, r, status)
	if !allowed {
		adminRootAssertError(t, w, platform.CodeRequestMethodNotAllowed)
		if w.Header().Get("Allow") != allow || !reflect.DeepEqual(f.events, []string{"trace"}) {
			t.Fatal("method rejection did not precede Session/article calls")
		}
	} else if method != http.MethodGet {
		adminRootAssertError(t, w, platform.CodeAuthOriginRejected)
		if !reflect.DeepEqual(f.events, []string{"trace"}) {
			t.Fatal("write Origin rejection reached Session")
		}
	}
}

func adminRootMissingHandlers(t *testing.T) {
	router := newLoginBusinessRouter(nil)
	for _, path := range []string{"/api/v1/admin/articles", "/api/v1/admin/articles/" + articleAuthorRootID} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusNotFound {
			t.Fatal("absent Handler returned successful data")
		}
	}
}

func adminRootOldConsumers(t *testing.T) {
	t.Run("read_projections", adminRootOldReadProjections)
	t.Run("mutation_security", adminRootMutationSecurity)
}

func adminRootOldReadProjections(t *testing.T) {
	for _, target := range []struct {
		name, path string
		status     int
	}{
		{"member_list", "/api/v1/articles", 200},
		{"member_detail", "/api/v1/articles/" + articleAuthorRootID, 200},
		{"member_hidden", "/api/v1/articles/10000000-0000-4000-8000-000000000002", 404},
		{"author_detail", "/api/v1/users/me/articles/" + articleAuthorRootID, 200},
		{"orphan_list", "/api/v1/admin/orphan-articles", 200},
		{"orphan_detail", "/api/v1/admin/orphan-articles/10000000-0000-4000-8000-000000000002", 200},
	} {
		t.Run(target.name, func(t *testing.T) {
			f := newAdminRootFixture(t, true)
			if target.name == "author_detail" {
				f.resolver.principal.UserID = *f.repository.rows[1].AuthorID
			}
			w := f.serve(t, f.request(target.path), target.status)
			if target.name == "author_detail" {
				var fields map[string]json.RawMessage
				if err := json.Unmarshal(w.Body.Bytes(), &fields); err != nil {
					t.Fatal(err)
				}
				if len(fields) != 4 || fields["title"] == nil || fields["body"] == nil || fields["editable"] == nil || fields["moderation_status"] == nil {
					t.Fatal("old author four-field projection changed")
				}
			}
		})
	}
}

func adminRootMutationSecurity(t *testing.T) {
	for _, target := range []struct{ name, method, path string }{
		{"create", http.MethodPost, "/api/v1/articles"},
		{"update", http.MethodPut, "/api/v1/articles/" + articleAuthorRootID},
		{"delete", http.MethodDelete, "/api/v1/admin/articles/" + articleAuthorRootID},
		{"violation", http.MethodPost, "/api/v1/admin/articles/" + articleAuthorRootID + "/violation"},
		{"restore", http.MethodPost, "/api/v1/admin/articles/" + articleAuthorRootID + "/restore"},
	} {
		t.Run(target.name, func(t *testing.T) {
			f := newAdminRootFixture(t, true)
			r := f.request(target.path)
			r.Method = target.method
			r.Header.Set("Origin", "https://community.example.invalid")
			w := f.serve(t, r, 403)
			adminRootAssertError(t, w, platform.CodeAuthCSRFFailed)
			if !reflect.DeepEqual(f.events, []string{"trace", "session", "csrf"}) || f.repository.lists+f.repository.gets+f.reader.calls != 0 {
				t.Fatal("old write CSRF rejection did not stop")
			}
			if target.name != "create" {
				adminRootMissingIfMatch(t, f, r)
			}
		})
	}
}

func adminRootMissingIfMatch(t *testing.T, f *adminRootFixture, r *http.Request) {
	t.Helper()
	f.events = nil
	r.Header.Set(platform.CSRFHeaderName, "csrf-token")
	w := f.serve(t, r, 428)
	adminRootAssertError(t, w, article.CodeRequestPreconditionRequired)
	if f.repository.lists+f.repository.gets+f.reader.calls != 0 {
		t.Fatal("missing If-Match reached query ports")
	}
}

const articleAuthorRootID = "10000000-0000-4000-8000-000000000001"

// articleAuthorRootPort is a controlled downstream, not a database substitute.
type articleAuthorRootPort struct {
	events    *[]string
	record    article.Article
	calls     [3]int
	principal session.Principal
	input     article.CreateArticleInput
	id        string
	version   int64
	err       error
}

func (p *articleAuthorRootPort) called(index int, principal session.Principal) {
	p.calls[index]++
	p.principal = principal
	*p.events = append(*p.events, "handler")
}

func (p *articleAuthorRootPort) Create(_ context.Context, principal session.Principal, input article.CreateArticleInput) (article.Article, error) {
	p.called(0, principal)
	p.input = input
	created := p.record
	created.Title, created.Body, created.Version = input.Title, input.Body, 1
	return created, p.err
}

func (p *articleAuthorRootPort) ReplaceContent(_ context.Context, principal session.Principal, id string, version int64, input article.CreateArticleInput) (article.Article, error) {
	p.called(2, principal)
	p.input, p.id, p.version = input, id, version
	updated := p.record
	updated.Title, updated.Body, updated.Version = input.Title, input.Body, version+1
	return updated, p.err
}

func (p *articleAuthorRootPort) GetAuthorArticle(_ context.Context, principal session.Principal, id string) (article.AuthorArticleView, string, error) {
	p.called(1, principal)
	p.id = id
	if p.err != nil {
		return article.AuthorArticleView{}, "", p.err
	}
	view, err := (article.VisibilityPolicy{}).AuthorProjection(p.record, principal)
	etag, etagErr := article.StrongETag(p.record)
	return view, etag, errors.Join(err, etagErr)
}

type articleAuthorRootResolver struct {
	events    *[]string
	principal session.Principal
	inputs    []session.ResolveInput
	err       error
}

func (r *articleAuthorRootResolver) ResolvePrincipal(_ context.Context, input session.ResolveInput) (session.Principal, error) {
	r.inputs = append(r.inputs, input)
	*r.events = append(*r.events, "session")
	if r.err != nil {
		return session.Principal{}, r.err
	}
	if input.RequireCSRF {
		*r.events = append(*r.events, "csrf")
		if input.CSRFToken != "csrf-token" {
			return session.Principal{}, session.ErrCSRFFailed
		}
	}
	*r.events = append(*r.events, "principal")
	return r.principal, nil
}

type articleAuthorRootFixture struct {
	handler      http.Handler
	port         *articleAuthorRootPort
	resolver     *articleAuthorRootResolver
	events       []string
	traceID      string
	guard        *platform.OriginGuard
	clock        identity.LoginClock
	observations []articleSharedDependency
}

func newArticleAuthorRootFixture(t *testing.T, controlled bool) *articleAuthorRootFixture {
	t.Helper()
	f := &articleAuthorRootFixture{}
	f.resolver = &articleAuthorRootResolver{events: &f.events, principal: session.Principal{
		SessionID: "session-1", UserID: "20000000-0000-4000-8000-000000000002", Role: "member", Status: "active",
	}}
	f.port = &articleAuthorRootPort{events: &f.events, record: article.Article{
		ID: articleAuthorRootID, AuthorID: &f.resolver.principal.UserID, Title: "original", Body: "body",
		PublicationState: article.PublicationPublished, ViolationState: article.ViolationClear,
		DeletionState: article.DeletionActive, Version: 2,
		CreatedAt: time.Date(2026, 10, 27, 1, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 10, 27, 1, 0, 0, 0, time.UTC),
	}}
	seam := &articleProductionTestSeam{principalResolver: f.resolver, shared: f.observe,
		trace: func(request *http.Request) {
			f.traceID, _ = platform.TraceIDFromContext(request.Context())
			f.events = append(f.events, "trace")
		},
		authorized: func(string) { f.events = append(f.events, "authorization") },
		protectOrigin: func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				f.events = append(f.events, "origin")
				f.guard.Protect(next).ServeHTTP(response, request)
			})
		},
	}
	if controlled {
		f.installControlledHandlers(t, seam)
	}
	f.handler = newArticleProductionHandlerWithTestSeam(t, seam)
	return f
}

func (f *articleAuthorRootFixture) observe(observation articleSharedDependency) {
	f.observations = append(f.observations, observation)
	if observation.Kind == "origin_guard" {
		f.guard = observation.Instance.(*platform.OriginGuard)
	}
	if observation.Kind == "clock" && observation.Consumer == "authorizer" {
		f.clock = observation.Instance.(identity.LoginClock)
	}
}

func (f *articleAuthorRootFixture) installControlledHandlers(t *testing.T, seam *articleProductionTestSeam) {
	t.Helper()
	logger, err := article.NewSlogArticleResponseWriteLogger(slog.New(slog.NewJSONHandler(io.Discard, nil)), serviceName)
	if err != nil {
		t.Fatal(err)
	}
	commands, err := article.NewHandler(f.port, logger)
	if err != nil {
		t.Fatal(err)
	}
	author, err := article.NewAuthorDetailHandler(f.port)
	if err != nil {
		t.Fatal(err)
	}
	seam.createHandler = http.HandlerFunc(commands.Create)
	seam.updateHandler = http.HandlerFunc(commands.UpdateArticle)
	seam.authorDetailHandler = http.HandlerFunc(author.GetAuthorArticle)
}

func articleAuthorRootRequest(method string) *http.Request {
	path := "/api/v1/articles/" + articleAuthorRootID
	if method == http.MethodPost {
		path = "/api/v1/articles"
	} else if method == http.MethodGet {
		path = "/api/v1/users/me/articles/" + articleAuthorRootID
	}
	request := httptest.NewRequest(method, path, strings.NewReader(`{"title":"  <b>标题</b>  ","body":" **literal** "}`))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: "session-token"})
	if method != http.MethodGet {
		request.Header.Set("Origin", "https://community.example.invalid")
		request.Header.Set(platform.CSRFHeaderName, "csrf-token")
	}
	if method == http.MethodPut {
		request.Header.Set("If-Match", `"article-`+articleAuthorRootID+`-v2"`)
	}
	return request
}

func (f *articleAuthorRootFixture) serve(t *testing.T, request *http.Request, status int) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	f.handler.ServeHTTP(response, request)
	if response.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, status, response.Body)
	}
	if f.traceID == "" || response.Header().Get(platform.TraceHeader) != f.traceID {
		t.Fatalf("trace context/header = %q/%q", f.traceID, response.Header().Get(platform.TraceHeader))
	}
	if !strings.HasPrefix(response.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("response security headers = %v", response.Header())
	}
	return response
}

func TestProductionArticleAuthorRoutes_ControlledRealHandlers_SuccessResponses(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodGet, http.MethodPut} {
		t.Run(method, func(t *testing.T) {
			f := newArticleAuthorRootFixture(t, true)
			status, version, index := http.StatusOK, int64(2), 1
			if method == http.MethodPost {
				status, version, index = http.StatusCreated, 1, 0
			} else if method == http.MethodPut {
				version, index = 3, 2
			}
			response := f.serve(t, articleAuthorRootRequest(method), status)
			wantCalls := [3]int{}
			wantCalls[index] = 1
			if f.port.calls != wantCalls || f.port.principal != f.resolver.principal {
				t.Fatalf("downstream calls/principal = %v/%+v", f.port.calls, f.port.principal)
			}
			wantETag := fmt.Sprintf(`"article-%s-v%d"`, articleAuthorRootID, version)
			if response.Header().Get("ETag") != wantETag {
				t.Fatalf("ETag = %q, want %q", response.Header().Get("ETag"), wantETag)
			}
			if method == http.MethodPost {
				assertArticleAuthorRootCreate(t, f, response)
			} else {
				assertArticleAuthorRootView(t, response, article.ModerationNormal)
				var view article.AuthorArticleView
				if err := json.Unmarshal(response.Body.Bytes(), &view); err != nil {
					t.Fatal(err)
				}
				title, body := f.port.record.Title, f.port.record.Body
				if method == http.MethodPut {
					title, body = "  <b>标题</b>  ", " **literal** "
				}
				if view.Title != title || view.Body != body {
					t.Fatalf("author content = %q/%q, want %q/%q", view.Title, view.Body, title, body)
				}
			}
			if method == http.MethodPut && (f.port.id != articleAuthorRootID || f.port.version != 2) {
				t.Fatalf("replacement id/version = %q/%d", f.port.id, f.port.version)
			}
		})
	}
}

func assertArticleAuthorRootCreate(t *testing.T, f *articleAuthorRootFixture, response *httptest.ResponseRecorder) {
	t.Helper()
	if response.Header().Get("Location") != "/api/v1/articles/"+articleAuthorRootID {
		t.Fatalf("Location = %q", response.Header().Get("Location"))
	}
	var view article.MemberArticleDetailView
	if err := json.Unmarshal(response.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view.ID != articleAuthorRootID || view.Title != f.port.input.Title || view.Body != f.port.input.Body || view.Title != "  <b>标题</b>  " || view.Body != " **literal** " {
		t.Fatalf("create payload/input = %+v/%+v", view, f.port.input)
	}
}

func assertArticleAuthorRootView(t *testing.T, response *httptest.ResponseRecorder, moderation article.ModerationStatus) {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields) != 4 {
		t.Fatalf("author fields = %v", fields)
	}
	for _, name := range []string{"title", "body", "moderation_status", "editable"} {
		if _, ok := fields[name]; !ok {
			t.Fatalf("missing author field %q", name)
		}
	}
	var view article.AuthorArticleView
	if err := json.Unmarshal(response.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view.ModerationStatus != moderation || !view.Editable {
		t.Fatalf("author view = %+v", view)
	}
}

func TestProductionArticleAuthorRoutes_ProductionConstructors_ShareActualReferences(t *testing.T) {
	f := newArticleAuthorRootFixture(t, false)
	lookup := func(kind, consumer string) any {
		t.Helper()
		var matches []any
		for _, observation := range f.observations {
			if observation.Kind == kind && observation.Consumer == consumer {
				matches = append(matches, observation.Instance)
			}
		}
		if len(matches) != 1 || matches[0] == nil {
			t.Fatalf("%s/%s constructor references = %v", kind, consumer, matches)
		}
		return matches[0]
	}
	query := lookup("article_query_service", "article_query_handler").(*article.ArticleQueryService)
	if query != lookup("article_query_service", "article_author_detail_handler") {
		t.Fatal("author and member handlers received different query services")
	}
	creator := lookup("article_command_service", "article_creator").(*article.CommandService)
	if creator != lookup("article_command_service", "article_content_replacer") {
		t.Fatal("create and replacement received different command services")
	}
	commands := lookup("article_command_handler", "create").(*article.Handler)
	if commands != lookup("article_command_handler", "update") {
		t.Fatal("create and update received different handlers")
	}
	if lookup("article_author_detail_handler", "router").(*article.Handler) == lookup("article_query_handler", "router") {
		t.Fatal("author route reused the member handler")
	}
	_ = lookup("article_response_logger", "article_command_handler").(*article.SlogArticleResponseWriteLogger)
	if lookup("logger", "article_response_logger") != lookup("logger", "article") || lookup("clock", "article") != lookup("clock", "authorizer") {
		t.Fatal("response logger or clock was recreated")
	}
	if lookup("pool", "article") != lookup("pool", "article_query_repository") || lookup("pool", "article") != lookup("pool", "article_author_reader") {
		t.Fatal("article constructor pools differ")
	}
}

func TestProductionArticleAuthorRoutes_SecurityChain_StopsEveryRejectedLayer(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodGet} {
		t.Run(method, func(t *testing.T) {
			cases := []string{"accepted", "no_cookie", "session_rejected", "suspended", "recovery", "must_change", "must_acknowledge"}
			if method != http.MethodGet {
				cases = append(cases, "origin_rejected", "csrf_missing", "csrf_forged")
			} else {
				cases = append(cases, "no_extra_csrf")
			}
			for _, name := range cases {
				t.Run(name, func(t *testing.T) { checkArticleAuthorRootSecurity(t, method, name) })
			}
		})
	}
}

func checkArticleAuthorRootSecurity(t *testing.T, method, name string) {
	t.Helper()
	f := newArticleAuthorRootFixture(t, true)
	request := articleAuthorRootRequest(method)
	status, code, events := configureArticleAuthorRootSecurity(f, request, name)
	before := f.clock.Now()
	response := f.serve(t, request, status)
	after := f.clock.Now()
	if !reflect.DeepEqual(f.events, events) {
		t.Fatalf("security events = %v, want %v", f.events, events)
	}
	if status >= http.StatusBadRequest {
		assertArticleAuthorRootError(t, response, code)
		if f.port.calls != [3]int{} {
			t.Fatalf("rejected request reached business downstream: %v", f.port.calls)
		}
	}
	if name == "origin_rejected" || name == "no_cookie" {
		if len(f.resolver.inputs) != 0 {
			t.Fatalf("early rejection reached resolver: %+v", f.resolver.inputs)
		}
		return
	}
	if len(f.resolver.inputs) != 1 {
		t.Fatalf("resolver calls = %d", len(f.resolver.inputs))
	}
	input := f.resolver.inputs[0]
	if input.RequireCSRF != (method != http.MethodGet) || input.Token != "session-token" || input.CSRFToken != request.Header.Get(platform.CSRFHeaderName) {
		t.Fatalf("resolver input does not match method/evidence: %+v", input)
	}
	if input.Now.IsZero() || input.Now.Before(before) || input.Now.After(after) {
		t.Fatalf("resolver time = %v, outside shared clock interval %v..%v", input.Now, before, after)
	}
}

func configureArticleAuthorRootSecurity(f *articleAuthorRootFixture, request *http.Request, name string) (int, platform.Code, []string) {
	events, status := []string{"trace"}, http.StatusOK
	if request.Method != http.MethodGet {
		events = append(events, "origin")
	}
	if request.Method == http.MethodPost {
		status = http.StatusCreated
	}
	if name == "origin_rejected" {
		request.Header.Set("Origin", "https://attacker.example.invalid")
		return http.StatusForbidden, platform.CodeAuthOriginRejected, events
	}
	if name == "no_cookie" {
		request.Header.Del("Cookie")
		return http.StatusUnauthorized, platform.CodeAuthSessionRequired, events
	}
	events = append(events, "session")
	if name == "session_rejected" {
		f.resolver.err = session.ErrSessionRequired
		return http.StatusUnauthorized, platform.CodeAuthSessionRequired, events
	}
	if request.Method != http.MethodGet {
		events = append(events, "csrf")
	}
	if name == "csrf_missing" || name == "csrf_forged" {
		request.Header.Set(platform.CSRFHeaderName, "forged")
		if name == "csrf_missing" {
			request.Header.Del(platform.CSRFHeaderName)
		}
		return http.StatusForbidden, platform.CodeAuthCSRFFailed, events
	}
	return configureArticleAuthorRootPrincipal(f, request, name, status, append(events, "principal"))
}

func configureArticleAuthorRootPrincipal(f *articleAuthorRootFixture, request *http.Request, name string, status int, events []string) (int, platform.Code, []string) {
	switch name {
	case "suspended", "recovery":
		f.resolver.principal.Status = name
		return http.StatusForbidden, platform.CodeAuthForbidden, events
	case "must_change":
		f.resolver.principal.MustChangePassword = true
		return http.StatusForbidden, platform.CodeAuthMustCompleteOnboarding, events
	case "must_acknowledge":
		f.resolver.principal.MustAcknowledgePrivacy = true
		return http.StatusForbidden, platform.CodeAuthMustCompleteOnboarding, events
	case "no_extra_csrf":
		request.Header.Set(platform.CSRFHeaderName, "forged-but-unused-for-GET")
		request.Header.Set("Origin", "https://attacker.example.invalid")
	}
	if request.Method != http.MethodGet {
		events = append(events, "authorization")
	}
	return status, "", append(events, "handler")
}

func assertArticleAuthorRootError(t *testing.T, response *httptest.ResponseRecorder, code platform.Code) {
	t.Helper()
	var envelope platform.Envelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Code != code || envelope.TraceID == "" || envelope.TraceID != response.Header().Get(platform.TraceHeader) {
		t.Fatalf("error envelope = %+v, want %s with matching trace", envelope, code)
	}
}

func TestProductionArticleAuthorRoutes_ExactRouting_RejectsBeforeSecurity(t *testing.T) {
	routes := []struct{ name, path, allow string }{
		{"collection", "/api/v1/articles", "GET, POST"},
		{"detail", "/api/v1/articles/" + articleAuthorRootID, "GET, PUT, DELETE"},
		{"author", "/api/v1/users/me/articles/" + articleAuthorRootID, "GET"},
	}
	for _, route := range routes {
		for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodHead, http.MethodOptions} {
			if strings.Contains(route.allow, method) {
				continue
			}
			t.Run(route.name+"_"+method, func(t *testing.T) {
				f := newArticleAuthorRootFixture(t, true)
				request := httptest.NewRequest(method, route.path, nil)
				request.Header.Set("Origin", "https://attacker.example.invalid")
				response := f.serve(t, request, http.StatusMethodNotAllowed)
				assertArticleAuthorRootError(t, response, platform.CodeRequestMethodNotAllowed)
				if response.Header().Get("Allow") != route.allow {
					t.Fatalf("Allow = %q, want %q", response.Header().Get("Allow"), route.allow)
				}
				if response.Header().Get("X-Content-Type-Options") != "nosniff" {
					t.Fatal("405 missing nosniff")
				}
				assertArticleAuthorRootNoSecurity(t, f)
			})
		}
	}
}

func TestProductionArticleAuthorRoutes_ExactRouting_UnknownPathsDoNotRedirect(t *testing.T) {
	for _, path := range []string{
		"/api/v1/articles/", "/api/v1/articles/" + articleAuthorRootID + "/",
		"/api/v1/articles/" + articleAuthorRootID + "/extra", "/api/v1/users/me/articles",
		"/api/v1/users/me/articles/", "/api/v1/users/me/articles/" + articleAuthorRootID + "/",
		"/api/v1/users/me/articles/" + articleAuthorRootID + "/extra", "/unknown",
	} {
		t.Run(strings.ReplaceAll(path, "/", "_"), func(t *testing.T) {
			f := newArticleAuthorRootFixture(t, true)
			request := httptest.NewRequest(http.MethodPost, path, nil)
			request.Header.Set("Origin", "https://attacker.example.invalid")
			response := f.serve(t, request, http.StatusNotFound)
			assertArticleAuthorRootError(t, response, platform.CodeRequestRouteNotFound)
			if response.Header().Get("Location") != "" || response.Header().Get("Allow") != "" {
				t.Fatalf("unknown path redirected or advertised methods: %v", response.Header())
			}
			if response.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatal("404 missing nosniff")
			}
			assertArticleAuthorRootNoSecurity(t, f)
		})
	}
}

func assertArticleAuthorRootNoSecurity(t *testing.T, f *articleAuthorRootFixture) {
	t.Helper()
	if !reflect.DeepEqual(f.events, []string{"trace"}) || len(f.resolver.inputs) != 0 || f.port.calls != [3]int{} {
		t.Fatalf("fallback reached security/downstream: %v / %v / %v", f.events, f.resolver.inputs, f.port.calls)
	}
}

func TestProductionArticleAuthorRoutes_ControlledRealHandlers_RestrictedAuthorView(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		t.Run(method, func(t *testing.T) {
			f := newArticleAuthorRootFixture(t, true)
			category := article.ViolationViolence
			f.port.record.ViolationState, f.port.record.ViolationCategory = article.ViolationViolating, &category
			response := f.serve(t, articleAuthorRootRequest(method), http.StatusOK)
			assertArticleAuthorRootView(t, response, article.ModerationRestricted)
			if strings.Contains(response.Body.String(), "violence") || response.Header().Get("ETag") == "" {
				t.Fatalf("restricted response leaked category or lost ETag: %s", response.Body)
			}
		})
	}
}

func TestProductionArticleAuthorRoutes_ControlledRealHandlers_ExactWireLimitContinues(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		t.Run(method, func(t *testing.T) {
			f := newArticleAuthorRootFixture(t, true)
			request := articleAuthorRootRequest(method)
			body := `{"title":"title","body":"body"}`
			body += strings.Repeat(" ", 262144-len(body))
			request.Body, request.ContentLength = io.NopCloser(strings.NewReader(body)), -1
			status, index := http.StatusCreated, 0
			if method == http.MethodPut {
				status, index = http.StatusOK, 2
			}
			response := f.serve(t, request, status)
			want := [3]int{}
			want[index] = 1
			if f.port.calls != want || f.port.input.Title != "title" || f.port.input.Body != "body" || response.Header().Get("ETag") == "" {
				t.Fatalf("exact N downstream/input/headers = %v/%+v/%v", f.port.calls, f.port.input, response.Header())
			}
		})
	}
}

func TestProductionArticleAuthorRoutes_ControlledRealHandlers_DownstreamErrorsPropagate(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodGet, http.MethodPut} {
		t.Run(method, func(t *testing.T) {
			f := newArticleAuthorRootFixture(t, true)
			f.port.err = platform.NewPublicError(platform.CodePlatformDatabaseUnavailable, errors.New("controlled failure"))
			response := f.serve(t, articleAuthorRootRequest(method), http.StatusServiceUnavailable)
			assertArticleAuthorRootError(t, response, platform.CodePlatformDatabaseUnavailable)
			if f.port.calls[0]+f.port.calls[1]+f.port.calls[2] != 1 || response.Header().Get("ETag") != "" || response.Header().Get("Location") != "" {
				t.Fatalf("downstream failure returned success evidence: %v / %v", f.port.calls, response.Header())
			}
		})
	}
}

func TestProductionArticleAuthorRoutes_RealHandlers_RejectInvalidWritesBeforeDownstream(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		for _, name := range []string{"invalid_json", "unknown_field", "duplicate_field", "oversize", "invalid_utf8", "missing_if_match", "weak_if_match", "cross_article_etag"} {
			if method == http.MethodPost && strings.Contains(name, "match") || method == http.MethodPost && name == "cross_article_etag" {
				continue
			}
			t.Run(method+"/"+name, func(t *testing.T) {
				for _, controlled := range []bool{true, false} {
					t.Run(fmt.Sprintf("controlled_%t", controlled), func(t *testing.T) {
						f := newArticleAuthorRootFixture(t, controlled)
						request := articleAuthorRootRequest(method)
						status, code := invalidArticleAuthorRootWrite(request, name)
						response := f.serve(t, request, status)
						assertArticleAuthorRootError(t, response, code)
						if f.port.calls != [3]int{} {
							t.Fatalf("invalid request reached controlled downstream: %v", f.port.calls)
						}
					})
				}
			})
		}
	}
}

func invalidArticleAuthorRootWrite(request *http.Request, name string) (int, platform.Code) {
	body := "{"
	switch name {
	case "unknown_field":
		body = `{"title":"ok","body":"ok","author_id":"untrusted"}`
	case "duplicate_field":
		body = `{"title":"ok","title":"again","body":"ok"}`
	case "oversize":
		body = `{"title":"ok","body":"ok"}` + strings.Repeat(" ", 262145-len(`{"title":"ok","body":"ok"}`))
	case "invalid_utf8":
		body = "{\"title\":\"" + string([]byte{0xff}) + "\",\"body\":\"ok\"}"
	case "missing_if_match":
		request.Header.Del("If-Match")
		return http.StatusPreconditionRequired, article.CodeRequestPreconditionRequired
	case "weak_if_match":
		request.Header.Set("If-Match", "W/"+request.Header.Get("If-Match"))
		return http.StatusBadRequest, article.CodeRequestInvalidPrecondition
	case "cross_article_etag":
		request.Header.Set("If-Match", `"article-20000000-0000-4000-8000-000000000002-v2"`)
		return http.StatusBadRequest, article.CodeRequestInvalidPrecondition
	}
	request.Body = io.NopCloser(strings.NewReader(body))
	request.ContentLength = -1 // Exercise actual wire bytes, not Content-Length.
	return http.StatusBadRequest, platform.CodeRequestInvalidJSON
}

func TestProductionArticleAuthorRoutes_SecurityEntry_UsesProductionRoot(t *testing.T) {
	tests := []struct {
		name, method, path string
		status             int
		code               platform.Code
	}{
		{"create", http.MethodPost, "/api/v1/articles", http.StatusForbidden, platform.CodeAuthOriginRejected},
		{"author_detail", http.MethodGet, "/api/v1/users/me/articles/article-1", http.StatusUnauthorized, platform.CodeAuthSessionRequired},
		{"update", http.MethodPut, "/api/v1/articles/article-1", http.StatusForbidden, platform.CodeAuthOriginRejected},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := newArticleProductionHandlerWithTestSeam(t, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(test.method, test.path, nil))
			var envelope platform.Envelope
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response.Code != test.status || envelope.Code != test.code {
				t.Fatalf("status/code = %d/%s, want %d/%s", response.Code, envelope.Code, test.status, test.code)
			}
		})
	}
}

func noticeRootEnvironment() map[string]string {
	values := validCompositionEnvironment()
	for key, value := range map[string]string{
		"POLL_INTERVAL": "1s", "BATCH_LIMIT": "2", "LEASE_DURATION": "10s",
		"DB_OPERATION_BUDGET": "1s", "BACKOFF_INITIAL": "1s", "BACKOFF_MAX": "8s",
	} {
		values["HAN_ARTICLE_TRANSFER_NOTICE_"+key] = value
	}
	values[smtpOperationBudgetEnvironment] = "2s"
	values[shutdownBudgetEnvironment] = "5s"
	values[articleFinalizerShutdownBudgetEnvironment] = "1s"
	return values
}

func TestNoticeConfig_InvalidFields_RejectBeforeDatabase(t *testing.T) {
	for _, suffix := range []string{"POLL_INTERVAL", "BATCH_LIMIT", "LEASE_DURATION", "DB_OPERATION_BUDGET", "BACKOFF_INITIAL", "BACKOFF_MAX"} {
		for _, value := range []string{"", " ", "-1", "0", "invalid", "1ns", " 1s", "999999999999999999999999h"} {
			t.Run(suffix+"/"+value, func(t *testing.T) {
				values := noticeRootEnvironment()
				key := "HAN_ARTICLE_TRANSFER_NOTICE_" + suffix
				values[key] = value
				deps := validCompositionDependencies(values)
				calls := 0
				deps.parseDatabase = func(string) (*pgxpool.Config, error) { calls++; return &pgxpool.Config{MaxConns: 8}, nil }
				if err := run(context.Background(), deps); err == nil {
					t.Error("invalid notification configuration accepted")
				}
				if calls != 0 {
					t.Errorf("database parse calls = %d, want zero", calls)
				}
			})
		}
	}
}

func TestNoticeConfig_CrossBudgets_RejectBeforeDatabase(t *testing.T) {
	for _, tc := range []struct{ key, value string }{
		{"LEASE_DURATION", "6s"}, {"DB_OPERATION_BUDGET", "2562047h47m16s"},
		{"BACKOFF_INITIAL", "9s"}, {"BATCH_LIMIT", "2147483648"},
	} {
		t.Run(tc.key, func(t *testing.T) {
			values := noticeRootEnvironment()
			values["HAN_ARTICLE_TRANSFER_NOTICE_"+tc.key] = tc.value
			if _, err := loadStartupConfig(func(key string) string { return values[key] }, mapEnvironmentLookup(values)); err == nil {
				t.Fatal("cross constraint accepted")
			}
		})
	}
	t.Run("shutdown equality", func(t *testing.T) {
		values := noticeRootEnvironment()
		values[shutdownBudgetEnvironment] = "4s"
		if _, err := loadStartupConfig(func(key string) string { return values[key] }, mapEnvironmentLookup(values)); err == nil {
			t.Fatal("shutdown equality accepted")
		}
	})
}

func TestNoticeProductionWiring_ProcessingGate_IsShared(t *testing.T) {
	config := mustLoadStartupConfig(t, noticeRootEnvironment())
	prepared, err := prepareProductionLoginDependencies(config, bytes.NewReader(bytes.Repeat([]byte{0x51}, 128)))
	clearStartupSecrets(&config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(prepared.cleanup)
	composition := productionBusinessComposition(t, prepared)
	builder, err := newProductionBusinessBuilder(new(pgxpool.Pool), composition)
	if err != nil {
		t.Fatal(err)
	}
	first, err := builder.newProcessingGate()
	if err != nil {
		t.Fatal(err)
	}
	second, err := builder.newProcessingGate()
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("production ProcessingGate was reconstructed")
	}
}

func TestNoticeConfig_LegalTuple_PreservesValues(t *testing.T) {
	values := noticeRootEnvironment()
	config := mustLoadStartupConfig(t, values)
	want := article.NoticeDispatcherConfig{PollInterval: time.Second, BatchLimit: 2, LeaseDuration: 10 * time.Second,
		DBOperationBudget: time.Second, BackoffInitial: time.Second, BackoffMax: 8 * time.Second, SMTPOperationBudget: 2 * time.Second}
	if config.noticeDispatcher != want {
		t.Fatalf("notice config = %#v, want %#v", config.noticeDispatcher, want)
	}
	for _, input := range []string{"1", "2147483647"} {
		values[noticeBatchLimitEnvironment] = input
		if got := mustLoadStartupConfig(t, values).noticeDispatcher.BatchLimit; strconv.Itoa(got) != input {
			t.Fatalf("batch = %d", got)
		}
	}
}

func TestNoticeConfig_StrictDecimalAndMicroseconds_NoDisclosure(t *testing.T) {
	for _, input := range []string{"01", "+1", "1.0", "1e2", "１２", "1 ", "2147483648"} {
		t.Run(input, func(t *testing.T) {
			values := noticeRootEnvironment()
			values[noticeBatchLimitEnvironment] = input
			_, err := loadStartupConfig(func(key string) string { return values[key] }, mapEnvironmentLookup(values))
			if err == nil || !strings.Contains(err.Error(), noticeBatchLimitEnvironment) || strings.Contains(err.Error(), input) {
				t.Fatalf("batch validation error = %v", err)
			}
		})
	}
	values := noticeRootEnvironment()
	values[noticePollIntervalEnvironment] = "1us1ns"
	if _, err := loadStartupConfig(func(key string) string { return values[key] }, mapEnvironmentLookup(values)); err == nil {
		t.Fatal("fractional microsecond accepted")
	}
}

func TestNoticeProductionWiring_ActualFactory_PublishesSharedWorker(t *testing.T) {
	config := mustLoadStartupConfig(t, noticeRootEnvironment())
	prepared, err := prepareProductionLoginDependencies(config, bytes.NewReader(bytes.Repeat([]byte{0x51}, 128)))
	clearStartupSecrets(&config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(prepared.cleanup)
	composition := productionBusinessComposition(t, prepared)
	composition.noticeDispatcher = config.noticeDispatcher
	composition.runtime = &businessRuntime{}
	observed := map[string]any{}
	composition.articleTestSeam = &articleProductionTestSeam{shared: func(value articleSharedDependency) {
		if value.Consumer == "notice" {
			observed[value.Kind] = value.Instance
		}
	}}
	pool := new(pgxpool.Pool)
	handler, err := newProductionBusinessHandler(pool, composition)
	if err != nil || handler == nil {
		t.Fatalf("production factory: %v", err)
	}
	worker, ok := composition.runtime.noticeWorker.(*article.TransferNoticeDispatcher)
	if !ok || worker == nil {
		t.Fatal("runtime did not receive real dispatcher")
	}
	if observed["pool"] != pool || observed["mailer"] != prepared.administratorMailer || observed["id_provider"] != prepared.ids || observed["notice_worker"] != worker {
		t.Fatal("notice factory did not use shared dependencies")
	}
	if logger, ok := observed["logger"].(*slog.Logger); !ok || logger == composition.logger {
		t.Fatal("notice factory did not inject its derived logger")
	}
	if _, ok := observed["audit_store"].(*audit.Store); !ok {
		t.Fatal("notice audit is not production Store")
	}
	if _, ok := observed["repository"].(article.Repository); !ok {
		t.Fatal("notice repository is not production Repository")
	}
	select {
	case <-worker.Done():
		t.Fatal("factory started/stopped worker")
	default:
	}
	worker.Stop()
	if err := worker.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
}

type noticeLogCanaryKey int

func TestNoticeProductionWiring_JointR01F001_BoundLoggerUsesProductionSink(t *testing.T) {
	var sink bytes.Buffer
	composition, worker, injected := noticeFactoryLoggerForTest(t, &sink)
	defer func() {
		worker.Stop()
		if err := worker.Wait(context.Background()); err != nil {
			t.Error(err)
		}
	}()
	ctx, err := platform.ContextWithTraceID(context.Background(), mainOrphanTraceID)
	if err != nil {
		t.Fatal(err)
	}
	canaries := []string{"smtp-response-private-canary", "private-recipient@example.invalid", "smtp-credential-private-canary"}
	for i, canary := range canaries {
		ctx = context.WithValue(ctx, noticeLogCanaryKey(i), canary)
	}
	// Exercise the exact logger and low-sensitivity argument contract consumed by core.observe.
	injected.Log(ctx, slog.LevelWarn, "article transfer notice", "reason_bucket", "result_persistence_failed", "trace_id", mainOrphanTraceID)
	composition.logger.InfoContext(ctx, "root sink probe", "event", "root_test_probe")
	assertNoticeProductionLog(t, sink.Bytes(), canaries)
}

func noticeFactoryLoggerForTest(t *testing.T, sink *bytes.Buffer) (businessComposition, *article.TransferNoticeDispatcher, article.NoticeDispatcherLogger) {
	t.Helper()
	config := mustLoadStartupConfig(t, noticeRootEnvironment())
	prepared, err := prepareProductionLoginDependencies(config, bytes.NewReader(bytes.Repeat([]byte{0x51}, 128)))
	clearStartupSecrets(&config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(prepared.cleanup)
	composition := productionBusinessComposition(t, prepared)
	composition.noticeDispatcher = config.noticeDispatcher
	composition.runtime = &businessRuntime{}
	composition.logger = slog.New(slog.NewJSONHandler(sink, nil))
	var injected article.NoticeDispatcherLogger
	composition.articleTestSeam = &articleProductionTestSeam{shared: func(value articleSharedDependency) {
		if value.Consumer == "notice" && value.Kind == "logger" {
			var ok bool
			injected, ok = value.Instance.(article.NoticeDispatcherLogger)
			if !ok {
				t.Error("factory did not observe its actual injected logger")
			}
		}
	}}
	handler, err := newProductionBusinessHandler(new(pgxpool.Pool), composition)
	if err != nil || handler == nil || injected == nil {
		t.Fatalf("production factory failed: %v", err)
	}
	worker, ok := composition.runtime.noticeWorker.(*article.TransferNoticeDispatcher)
	if !ok || worker == nil {
		t.Fatal("production runtime does not contain actual notice worker")
	}
	return composition, worker, injected
}

func assertNoticeProductionLog(t *testing.T, records []byte, canaries []string) {
	t.Helper()
	for _, canary := range canaries {
		if strings.Contains(string(records), canary) {
			t.Error("notice log disclosed a sensitive context canary")
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(records))
	var notice, base map[string]any
	if err := decoder.Decode(&notice); err != nil {
		t.Fatal(err)
	}
	if err := decoder.Decode(&base); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"service": serviceName, "module": "article", "event": "article.transfer_notice.warning", "msg": "article transfer notice", "level": "WARN", "trace_id": mainOrphanTraceID, "reason_bucket": "result_persistence_failed"} {
		if notice[key] != want {
			t.Errorf("notice field %s = %v, want %s", key, notice[key], want)
		}
	}
	if notice["time"] == nil || base["msg"] != "root sink probe" || base["event"] != "root_test_probe" {
		t.Error("notice and root did not write valid JSON into the same sink")
	}
	if _, exists := base["module"]; exists {
		t.Error("derived notice logger mutated the base logger")
	}
	if err := decoder.Decode(new(map[string]any)); err != io.EOF {
		t.Errorf("unexpected additional log record: %v", err)
	}
}

func TestNoticeProductionWiring_MissingWorker_RejectsBeforeListen(t *testing.T) {
	for _, invalid := range []lifecycleWorker{nil, (*article.TransferNoticeDispatcher)(nil), newLifecycleWorkerFake()} {
		values := noticeRootEnvironment()
		deps := validCompositionDependencies(values)
		deps.requireNoticeWorker = true
		deps.composeBusiness = func(_ *pgxpool.Pool, composition businessComposition) (http.Handler, error) {
			composition.runtime.noticeWorker = invalid
			return http.NotFoundHandler(), nil
		}
		closes, listens := 0, 0
		deps.closePool = func(*pgxpool.Pool) { closes++ }
		deps.listen = func(string, string) (net.Listener, error) { listens++; return &recordingListener{}, nil }
		if err := run(context.Background(), deps); err == nil {
			t.Fatal("missing notice worker accepted")
		}
		if listens != 0 || closes != 1 {
			t.Fatalf("listen/pool close = %d/%d", listens, closes)
		}
	}
}

type noticeGateTransaction struct {
	pgx.Tx
	config     platform.ProcessingGateConfig
	queryError error
	queries    int
}

func (tx *noticeGateTransaction) QueryRow(context.Context, string, ...any) pgx.Row {
	tx.queries++
	return noticeGateRow{tx: tx}
}

type noticeGateRow struct{ tx *noticeGateTransaction }

func (row noticeGateRow) Scan(dest ...any) error {
	if row.tx.queryError != nil {
		return row.tx.queryError
	}
	values := []any{int64(1), row.tx.config.RegionRef, row.tx.config.ControllerRef, row.tx.config.LegalBasisRef, row.tx.config.SMTPProcessorRef, time.Now().UTC(), "synthetic-registrar"}
	for i, value := range values {
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(value))
	}
	return nil
}

func TestNoticeProductionWiring_Gate_UsesCallerTransactionAndSameConfig(t *testing.T) {
	config := platform.ProcessingGateConfig{RegionRef: "region", ControllerRef: "controller", LegalBasisRef: "basis", SMTPProcessorRef: "processor"}
	tx := &noticeGateTransaction{config: config}
	gate := noticeProcessingGate{config: config}
	if err := gate.RequireOpenInTx(context.Background(), tx); err != nil || tx.queries != 1 {
		t.Fatalf("caller tx gate: calls=%d err=%v", tx.queries, err)
	}
	tx.config.RegionRef = "different"
	if err := gate.RequireOpenInTx(context.Background(), tx); err == nil {
		t.Fatal("mismatched registration opened gate")
	}
	failure := errors.New("synthetic gate query failure")
	tx.queryError = failure
	if err := gate.RequireOpenInTx(context.Background(), tx); !errors.Is(err, failure) {
		t.Fatalf("query cause lost: %v", err)
	}
}

func TestNoticeProductionWiring_RunComposed_PassesActualWorkerToLifecycle(t *testing.T) {
	values := noticeRootEnvironment()
	deps := validCompositionDependencies(values)
	deps.requireNoticeWorker = true
	deps.prepareBusiness = func(config startupConfig) (*preparedLoginDependencies, error) {
		return prepareProductionLoginDependencies(config, bytes.NewReader(bytes.Repeat([]byte{0x41}, 128)))
	}
	var constructed *article.TransferNoticeDispatcher
	deps.composeBusiness = func(pool *pgxpool.Pool, composition businessComposition) (http.Handler, error) {
		handler, err := newProductionBusinessHandler(pool, composition)
		if err == nil {
			var ok bool
			constructed, ok = composition.runtime.noticeWorker.(*article.TransferNoticeDispatcher)
			if !ok {
				t.Error("factory did not supply actual dispatcher")
			}
		}
		return handler, err
	}
	coordinateCalls, poolCloses := 0, 0
	deps.closePool = func(*pgxpool.Pool) { poolCloses++ }
	deps.coordinate = func(_ context.Context, config lifecycleConfig) error {
		coordinateCalls++
		if config.noticeWorker != constructed || constructed == nil {
			t.Error("lifecycle did not receive factory worker")
		}
		config.noticeWorker.(interface{ BeginShutdown(context.Context) }).BeginShutdown(context.Background())
		config.noticeWorker.Stop()
		if err := config.noticeWorker.Wait(context.Background()); err != nil {
			t.Error(err)
		}
		return config.closePool()
	}
	if err := run(context.Background(), deps); err != nil {
		t.Fatal(err)
	}
	if coordinateCalls != 1 || poolCloses != 1 {
		t.Fatalf("coordinate/poolclose=%d/%d", coordinateCalls, poolCloses)
	}
}

type noticeRootIDs struct {
	calls         int
	invalidNotice bool
}

func (ids *noticeRootIDs) NewID() (string, error) {
	ids.calls++
	if ids.invalidNotice && ids.calls == 2 {
		return "invalid-worker-id", nil
	}
	return fmt.Sprintf("30000000-0000-4000-8000-%012d", ids.calls), nil
}

func TestNoticeProductionWiring_WorkerIdentity_OncePerInstanceAndFailClosed(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		t.Run(strconv.FormatBool(invalid), func(t *testing.T) {
			config := mustLoadStartupConfig(t, noticeRootEnvironment())
			prepared, err := prepareProductionLoginDependencies(config, bytes.NewReader(bytes.Repeat([]byte{0x51}, 128)))
			clearStartupSecrets(&config)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(prepared.cleanup)
			ids := &noticeRootIDs{invalidNotice: invalid}
			prepared.ids = ids
			composition := productionBusinessComposition(t, prepared)
			composition.runtime = &businessRuntime{}
			pool := new(pgxpool.Pool)
			builder, err := newProductionBusinessBuilder(pool, composition)
			if invalid {
				if err == nil || composition.runtime.noticeWorker != nil {
					t.Fatal("invalid instance identity accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if ids.calls != 2 {
				t.Fatalf("UUID calls=%d, want one per password-reset and notice instance", ids.calls)
			}
			next, err := newProductionBusinessBuilder(pool, composition)
			if err != nil || next.noticeWorker == builder.noticeWorker || ids.calls != 4 {
				t.Fatalf("new instance identity was reused: %v, calls=%d", err, ids.calls)
			}
			for _, worker := range []*article.TransferNoticeDispatcher{builder.noticeWorker, next.noticeWorker} {
				worker.Stop()
				if err := worker.Wait(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

const (
	mainOrphanSourceID  = "20000000-0000-4000-8000-000000000001"
	mainOrphanTargetID  = "20000000-0000-4000-8000-000000000002"
	mainOrphanArticleID = "10000000-0000-4000-8000-000000000001"
	mainOrphanTraceID   = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
)

type mainOrphanState struct {
	userError       error
	record          article.Article
	absent          bool
	sourceEmail     *string
	emailIdentifier bool
	notice          *article.TransferNotice
	recipients      []article.TransferNoticeRecipient
	audits          map[string]audit.Event
	operation       *article.DeletionOperation
}

func (state *mainOrphanState) copy() mainOrphanState {
	image := *state
	image.recipients = append([]article.TransferNoticeRecipient(nil), state.recipients...)
	image.audits = make(map[string]audit.Event)
	for eventID, event := range state.audits {
		image.audits[eventID] = event
	}
	if state.operation != nil {
		operation := *state.operation
		image.operation = &operation
	}
	return image
}

type mainOrphanTransactor struct {
	state        *mainOrphanState
	transactions []*mainOrphanTx
}

func (transactor *mainOrphanTransactor) Begin(context.Context) (pgx.Tx, error) {
	transaction := &mainOrphanTx{mainLoginTx: &mainLoginTx{}, state: transactor.state, before: transactor.state.copy()}
	transactor.transactions = append(transactor.transactions, transaction)
	return transaction, nil
}

type mainOrphanTx struct {
	*mainLoginTx
	state         *mainOrphanState
	before        mainOrphanState
	readCommitted bool
	identityReads int
}

func (transaction *mainOrphanTx) Rollback(ctx context.Context) error {
	if !transaction.closed {
		*transaction.state = transaction.before
	}
	return transaction.mainLoginTx.Rollback(ctx)
}

type mainOrphanRow struct {
	values []any
	err    error
}

func (row mainOrphanRow) Scan(destinations ...any) error {
	if row.err != nil {
		return row.err
	}
	if len(destinations) != len(row.values) {
		return errors.New("unexpected root SQL row shape")
	}
	for index, value := range row.values {
		if value == nil {
			reflect.ValueOf(destinations[index]).Elem().SetZero()
		} else {
			reflect.ValueOf(destinations[index]).Elem().Set(reflect.ValueOf(value))
		}
	}
	return nil
}

func (transaction *mainOrphanTx) QueryRow(ctx context.Context, query string, arguments ...any) pgx.Row {
	if transaction.closed {
		return mainOrphanRow{err: pgx.ErrTxClosed}
	}
	state := transaction.state
	if strings.Contains(query, "FROM users") {
		transaction.identityReads++
		if state.userError != nil {
			return mainOrphanRow{err: state.userError}
		}
		if !strings.Contains(query, "FOR UPDATE NOWAIT") {
			return mainOrphanRow{err: errors.New("identity lock was not NOWAIT")}
		}
		userID := arguments[0].(string)
		if userID == mainOrphanSourceID {
			return mainOrphanRow{values: []any{userID, "former", (*string)(nil), (*string)(nil),
				identity.UserStatusDeleted, state.sourceEmail, mainOrphanTimePointer(time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC))}}
		}
		if userID == mainOrphanTargetID {
			return mainOrphanRow{values: []any{userID, "new", mainOrphanStringPointer("New@Example.test"),
				(*string)(nil), identity.UserStatusActive, (*string)(nil), (*time.Time)(nil)}}
		}
		return mainOrphanRow{err: pgx.ErrNoRows}
	}
	if strings.Contains(query, "FROM audit_events") {
		event, exists := state.audits[arguments[0].(string)]
		if !exists {
			return mainOrphanRow{err: pgx.ErrNoRows}
		}
		metadata, _ := json.Marshal(event.Metadata)
		return mainOrphanRow{values: []any{event.ID, event.OccurredAt, event.ActorKind, event.ActorUserID, event.ActorSystem,
			event.ActorRole, event.Action, event.TargetType, event.TargetID, event.Outcome, event.ErrorCode, event.TraceID, event.SourceFingerprint, metadata}}
	}
	if strings.Contains(query, "FROM transfer_notices") {
		if state.notice == nil {
			return mainOrphanRow{err: pgx.ErrNoRows}
		}
		return mainOrphanRow{values: []any{state.notice.ID, state.notice.ArticleRef, state.notice.NewAuthorID, state.notice.FormerDeletedUserID}}
	}
	if strings.Contains(query, "deletion_operations") {
		return transaction.operationRow(query, arguments...)
	}
	return transaction.articleRow(query, arguments...)
}

func (transaction *mainOrphanTx) articleRow(query string, arguments ...any) pgx.Row {
	state, record := transaction.state, transaction.state.record
	if strings.Contains(query, "count(*) FILTER") {
		if !transaction.readCommitted || arguments[0] != mainOrphanSourceID {
			return mainOrphanRow{err: errors.New("root used stale isolation or wrong count source")}
		}
		return mainOrphanRow{values: []any{int64(0), int64(0)}}
	}
	if strings.Contains(query, "SELECT EXISTS") {
		return mainOrphanRow{values: []any{!state.absent}}
	}
	if state.absent {
		return mainOrphanRow{err: pgx.ErrNoRows}
	}
	if strings.Contains(query, "SELECT id, orphaned_from_user_id, version") {
		return mainOrphanRow{values: []any{record.ID, record.OrphanedFromUserID, record.Version}}
	}
	if strings.Contains(query, "SELECT id, version, author_id") {
		return mainOrphanRow{values: []any{record.ID, record.Version, record.AuthorID, record.OrphanedFromUserID,
			record.PublicationState, record.ViolationState, record.ViolationCategory, record.DeletionState, record.DeletionOperationID}}
	}
	if strings.Contains(query, "FROM articles") && strings.Contains(query, "FOR UPDATE NOWAIT") {
		return mainOrphanRow{values: []any{record.ID, record.AuthorID, record.OrphanedFromUserID, record.Title, record.Body,
			record.PublicationState, record.ViolationState, record.ViolationCategory, record.DeletionState,
			record.DeletionOperationID, record.Version, record.CreatedAt, record.UpdatedAt}}
	}
	return mainOrphanRow{err: errors.New("unexpected root article SQL")}
}

func (transaction *mainOrphanTx) operationRow(query string, arguments ...any) pgx.Row {
	operation := transaction.state.operation
	if operation == nil {
		return mainOrphanRow{err: pgx.ErrNoRows}
	}
	if strings.Contains(query, "UPDATE deletion_operations") {
		deadline := arguments[2].(time.Time).UTC().Truncate(time.Microsecond)
		operation.Status, operation.ClaimedBy, operation.LockedUntil = article.DeletionOperationFinalizing,
			mainOrphanStringPointer(arguments[1].(string)), &deadline
	}
	return mainOrphanRow{values: []any{operation.ID, operation.ArticleID, operation.ArticleVersion, operation.DeleteKind,
		operation.ActorUserID, operation.ActorRole, operation.OrphanedFromUserID, operation.Status, operation.TombstoneSequence,
		operation.JournalKeyID, operation.JournalKeyVersion, operation.ClaimedBy, operation.LockedUntil,
		operation.AuditEventID, operation.StartedAt, operation.CompletedAt}}
}

func (transaction *mainOrphanTx) Query(ctx context.Context, query string, arguments ...any) (pgx.Rows, error) {
	if transaction.closed {
		return nil, pgx.ErrTxClosed
	}
	var values [][]any
	if strings.Contains(query, "FROM identity_identifiers") {
		if !strings.Contains(query, "FOR UPDATE NOWAIT") {
			return nil, errors.New("identifier lock was not NOWAIT")
		}
		userID := arguments[0].(string)
		username, lifecycle, email := "former", identity.IdentifierReserved, "former@example.test"
		if userID == mainOrphanTargetID {
			username, lifecycle, email = "new", identity.IdentifierActive, "new@example.test"
		}
		values = append(values, []any{userID, identity.IdentifierUsername, username, lifecycle, (*time.Time)(nil)})
		if userID == mainOrphanTargetID || transaction.state.emailIdentifier {
			values = append(values, []any{userID, identity.IdentifierEmail, email, lifecycle, (*time.Time)(nil)})
		}
	} else if strings.Contains(query, "FROM transfer_notice_recipients") {
		for _, recipient := range transaction.state.recipients {
			values = append(values, []any{recipient.NoticeID, recipient.Kind, recipient.EmailSnapshot, recipient.Status})
		}
	} else {
		return nil, errors.New("unexpected root row query")
	}
	return &mainOrphanRows{values: values}, nil
}

type mainOrphanRows struct {
	pgx.Rows
	values [][]any
	index  int
}

func (rows *mainOrphanRows) Next() bool { rows.index++; return rows.index <= len(rows.values) }
func (rows *mainOrphanRows) Close()     {}
func (rows *mainOrphanRows) Err() error { return nil }
func (rows *mainOrphanRows) Scan(destinations ...any) error {
	return (mainOrphanRow{values: rows.values[rows.index-1]}).Scan(destinations...)
}

func (transaction *mainOrphanTx) Exec(ctx context.Context, query string, arguments ...any) (pgconn.CommandTag, error) {
	if transaction.closed {
		return pgconn.CommandTag{}, pgx.ErrTxClosed
	}
	state := transaction.state
	switch {
	case query == "SET TRANSACTION ISOLATION LEVEL READ COMMITTED":
		transaction.readCommitted = true
	case strings.Contains(query, "UPDATE articles"):
		state.record.AuthorID, state.record.OrphanedFromUserID = arguments[3].(*string), nil
		state.record.PublicationState, state.record.UpdatedAt = article.PublicationPublished, arguments[4].(time.Time)
		state.record.Version++
	case strings.Contains(query, "INSERT INTO transfer_notices"):
		state.notice = &article.TransferNotice{ID: arguments[0].(string), ArticleRef: arguments[1].(string),
			NewAuthorID: arguments[2].(string), FormerDeletedUserID: arguments[3].(string)}
	case strings.Contains(query, "INSERT INTO transfer_notice_recipients"):
		state.recipients = append(state.recipients, article.TransferNoticeRecipient{NoticeID: arguments[0].(string),
			Kind: arguments[1].(article.RecipientKind), EmailSnapshot: arguments[2].(string), Status: arguments[3].(article.RecipientStatus), MaxAttempts: arguments[8].(int64)})
	case strings.Contains(query, "INSERT INTO audit_events"):
		event := audit.Event{ID: arguments[0].(string), OccurredAt: arguments[1].(time.Time), ActorKind: arguments[2].(audit.ActorKind),
			ActorUserID: arguments[3].(*string), ActorSystem: arguments[4].(*string), ActorRole: arguments[5].(*string),
			Action: arguments[6].(string), TargetType: arguments[7].(string), TargetID: arguments[8].(string), Outcome: arguments[9].(string), TraceID: arguments[11].(string)}
		if err := json.Unmarshal(arguments[13].([]byte), &event.Metadata); err != nil {
			return pgconn.CommandTag{}, err
		}
		state.audits[event.ID] = event
	case strings.Contains(query, "UPDATE users"):
		if state.sourceEmail == nil || arguments[0] != mainOrphanSourceID || arguments[1] != *state.sourceEmail {
			return pgconn.NewCommandTag("UPDATE 0"), nil
		}
		state.sourceEmail = nil
	case strings.Contains(query, "DELETE FROM identity_identifiers"):
		state.emailIdentifier = false
	case strings.Contains(query, "DELETE FROM articles"):
		state.absent = true
	case strings.Contains(query, "UPDATE deletion_operations"):
		state.operation.Status, state.operation.ClaimedBy, state.operation.LockedUntil = article.DeletionOperationCompleted, nil, nil
		state.operation.AuditEventID, state.operation.CompletedAt = mainOrphanStringPointer(arguments[2].(string)), mainOrphanTimePointer(arguments[3].(time.Time))
	default:
		return pgconn.CommandTag{}, errors.New("unexpected root mutation SQL")
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func mainOrphanStringPointer(value string) *string     { return &value }
func mainOrphanTimePointer(value time.Time) *time.Time { return &value }

type mainOrphanClock struct{ now time.Time }

func (clock mainOrphanClock) Now() time.Time { return clock.now }

type mainOrphanIdentityCapture struct {
	delegate    *articleIdentityLifecycleAdapter
	proof       *article.DispositionCompletionProof
	transaction pgx.Tx
	beforeClear func(context.Context, pgx.Tx, *article.DispositionCompletionProof) error
}

func (capture *mainOrphanIdentityCapture) LockTransferParties(ctx context.Context, transaction pgx.Tx, targetID *string, sourceID string) (article.DispositionParties, error) {
	return capture.delegate.LockTransferParties(ctx, transaction, targetID, sourceID)
}
func (capture *mainOrphanIdentityCapture) LockDeletionCandidate(ctx context.Context, transaction pgx.Tx, userID string) (article.DispositionDeletionCandidate, error) {
	return capture.delegate.LockDeletionCandidate(ctx, transaction, userID)
}
func (capture *mainOrphanIdentityCapture) ClearRetainedEmailIfEligible(ctx context.Context, transaction pgx.Tx, sourceID string, proof *article.DispositionCompletionProof) (bool, error) {
	capture.proof, capture.transaction = proof, transaction
	if capture.beforeClear != nil {
		if err := capture.beforeClear(ctx, transaction, proof); err != nil {
			return false, err
		}
	}
	return capture.delegate.ClearRetainedEmailIfEligible(ctx, transaction, sourceID, proof)
}

type mainOrphanFixture struct {
	ctx        context.Context
	state      *mainOrphanState
	transactor *mainOrphanTransactor
	identity   *mainOrphanIdentityCapture
	service    *article.CommandService
	now        time.Time
}

func newMainOrphanFixture(t *testing.T) *mainOrphanFixture {
	t.Helper()
	now := time.Date(2026, 10, 27, 10, 0, 0, 0, time.UTC)
	state := &mainOrphanState{record: article.Article{ID: mainOrphanArticleID, OrphanedFromUserID: mainOrphanStringPointer(mainOrphanSourceID),
		Title: "synthetic", Body: "synthetic", PublicationState: article.PublicationHeld, ViolationState: article.ViolationClear,
		DeletionState: article.DeletionActive, Version: 3, CreatedAt: now, UpdatedAt: now},
		sourceEmail: mainOrphanStringPointer("Former@Example.test"), emailIdentifier: true, audits: make(map[string]audit.Event)}
	adapter, err := newArticleIdentityLifecycleAdapter(audit.Store{}, identity.RandomUUIDGenerator{}, mainOrphanClock{now: now})
	if err != nil {
		t.Fatal(err)
	}
	capture := &mainOrphanIdentityCapture{delegate: adapter}
	transactor := &mainOrphanTransactor{state: state}
	service, err := article.NewCommandService(article.CommandDependencies{Transactions: transactor, Repository: article.Repository{},
		IDs: identity.RandomUUIDGenerator{}, Clock: mainOrphanClock{now: now}, Disposition: &article.OrphanDispositionDependencies{Identity: capture, Auditor: audit.Store{}, RecipientMaxAttempts: 7}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := platform.ContextWithTraceID(context.Background(), mainOrphanTraceID)
	if err != nil {
		t.Fatal(err)
	}
	return &mainOrphanFixture{ctx: ctx, state: state, transactor: transactor, identity: capture, service: service, now: now}
}

func TestOrphanDispositionRoot_ProductionFactoryAndNativeProof_ReleaseInCallerTransaction(t *testing.T) {
	for _, transfer := range []bool{false, true} {
		fixture := newMainOrphanFixture(t)
		principal := session.Principal{UserID: mainOrphanTargetID, Role: "admin", Status: "active"}
		var err error
		if transfer {
			_, err = fixture.service.TransferOrphan(fixture.ctx, principal, mainOrphanArticleID, 3, mainOrphanTargetID)
		} else {
			_, err = fixture.service.PublishOrphan(fixture.ctx, principal, mainOrphanArticleID, 3)
		}
		if err != nil || fixture.state.sourceEmail != nil || fixture.state.emailIdentifier || fixture.identity.proof == nil ||
			fixture.transactor.transactions[0].commitCalls != 1 || fixture.transactor.transactions[0].identityReads < 2 {
			t.Fatalf("production factory did not complete real SQL methods: %v", err)
		}
		releases := 0
		for _, event := range fixture.state.audits {
			if event.Action == "identity.identifier.email.release" {
				releases++
			}
		}
		if releases != 1 {
			t.Fatal("release audit missing or duplicated")
		}
		for _, recipient := range fixture.state.recipients {
			if recipient.MaxAttempts != 7 {
				t.Fatal("max_attempts did not retain its independent creation snapshot")
			}
		}
		if _, err := fixture.identity.proof.VerifyCompletion(fixture.ctx, fixture.identity.transaction); !errors.Is(err, pgx.ErrTxClosed) {
			t.Fatal("proof survived caller commit")
		}
	}
}

func TestOrphanDispositionRoot_BoundVerifier_RejectsEveryReferenceField(t *testing.T) {
	for _, testCase := range mainOrphanReferenceMutations() {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newMainOrphanFixture(t)
			fixture.identity.beforeClear = func(ctx context.Context, transaction pgx.Tx, proof *article.DispositionCompletionProof) error {
				reference := articleIdentityReference(proof.Reference())
				verifier := &articleBoundCompletionVerifier{proof: proof, reference: reference, traceID: mainOrphanTraceID}
				testCase.mutate(&reference)
				if _, err := verifier.VerifyCompletion(ctx, transaction, reference); err == nil {
					return errors.New("tampered reference accepted")
				}
				if fixture.state.sourceEmail == nil {
					return errors.New("tampered reference released identity state")
				}
				return nil
			}
			_, err := fixture.service.TransferOrphan(fixture.ctx, session.Principal{UserID: mainOrphanTargetID, Role: "admin", Status: "active"}, mainOrphanArticleID, 3, mainOrphanTargetID)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func mainOrphanReferenceMutations() []struct {
	name   string
	mutate func(*identity.ArticleCompletionReference)
} {
	return []struct {
		name   string
		mutate func(*identity.ArticleCompletionReference)
	}{
		{"article id", func(reference *identity.ArticleCompletionReference) { reference.ExpectedArticleID = mainOrphanTargetID }},
		{"source id", func(reference *identity.ArticleCompletionReference) { reference.SourceID = mainOrphanTargetID }},
		{"before version", func(reference *identity.ArticleCompletionReference) { reference.BeforeVersion++ }},
		{"action", func(reference *identity.ArticleCompletionReference) {
			reference.Action = identity.ArticleDispositionDeleteComplete
		}},
		{"audit id", func(reference *identity.ArticleCompletionReference) { reference.AuditEventID = mainOrphanTargetID }},
		{"notice id", func(reference *identity.ArticleCompletionReference) { *reference.NoticeID = mainOrphanTargetID }},
		{"new author", func(reference *identity.ArticleCompletionReference) { *reference.NewAuthorID = mainOrphanSourceID }},
		{"operation id", func(reference *identity.ArticleCompletionReference) {
			reference.OperationID = mainOrphanStringPointer(mainOrphanTargetID)
		}},
		{"before existence", func(reference *identity.ArticleCompletionReference) { reference.BeforeArticle.Exists = false }},
		{"before id", func(reference *identity.ArticleCompletionReference) { reference.BeforeArticle.ID = mainOrphanTargetID }},
		{"before state version", func(reference *identity.ArticleCompletionReference) { reference.BeforeArticle.Version++ }},
		{"before author", func(reference *identity.ArticleCompletionReference) {
			reference.BeforeArticle.AuthorID = mainOrphanStringPointer(mainOrphanTargetID)
		}},
		{"before source", func(reference *identity.ArticleCompletionReference) {
			*reference.BeforeArticle.SourceID = mainOrphanTargetID
		}},
		{"before publication", func(reference *identity.ArticleCompletionReference) {
			reference.BeforeArticle.Publication = "published"
		}},
		{"before violation", func(reference *identity.ArticleCompletionReference) { reference.BeforeArticle.Violation = "violating" }},
		{"before category", func(reference *identity.ArticleCompletionReference) {
			reference.BeforeArticle.Category = mainOrphanStringPointer("violence")
		}},
		{"before deletion", func(reference *identity.ArticleCompletionReference) {
			reference.BeforeArticle.Deletion = "deletion_pending"
		}},
		{"before operation", func(reference *identity.ArticleCompletionReference) {
			reference.BeforeArticle.OperationID = mainOrphanStringPointer(mainOrphanTargetID)
		}},
		{"transaction", func(reference *identity.ArticleCompletionReference) { reference.Transaction = &mainLoginTx{} }},
	}
}

func TestOrphanDispositionRoot_BoundFactory_RejectsWrongTxTraceAndEndedTx(t *testing.T) {
	fixture := newMainOrphanFixture(t)
	fixture.identity.beforeClear = func(ctx context.Context, transaction pgx.Tx, proof *article.DispositionCompletionProof) error {
		reference := articleIdentityReference(proof.Reference())
		verifier := &articleBoundCompletionVerifier{proof: proof, reference: reference, traceID: mainOrphanTraceID}
		wrongTrace, err := platform.ContextWithTraceID(ctx, "01ARZ3NDEKTSV4RRFFQ69G5FAW")
		if err != nil {
			return err
		}
		for _, supplied := range []context.Context{context.Background(), wrongTrace} {
			if _, err := verifier.VerifyCompletion(supplied, transaction, reference); err == nil {
				return errors.New("wrong trace accepted")
			}
		}
		wrapped := &mainOrphanTxWrapper{Tx: transaction}
		if _, err := verifier.VerifyCompletion(ctx, wrapped, reference); err == nil {
			return errors.New("different tx wrapper accepted")
		}
		if _, _, err := fixture.identity.delegate.bindCompletion(ctx, wrapped, proof); err == nil {
			return errors.New("factory bound another tx")
		}
		return nil
	}
	_, err := fixture.service.PublishOrphan(fixture.ctx, session.Principal{UserID: mainOrphanTargetID, Role: "admin", Status: "active"}, mainOrphanArticleID, 3)
	if err != nil {
		t.Fatal(err)
	}
	reference := articleIdentityReference(fixture.identity.proof.Reference())
	verifier := &articleBoundCompletionVerifier{proof: fixture.identity.proof, reference: reference, traceID: mainOrphanTraceID}
	if _, err := verifier.VerifyCompletion(fixture.ctx, fixture.identity.transaction, reference); !errors.Is(err, pgx.ErrTxClosed) {
		t.Fatalf("ended tx accepted: %v", err)
	}
}

type mainOrphanTxWrapper struct{ pgx.Tx }

type mainOrphanJournal struct {
	record article.ArticleDeletionTombstone
}

func (journal mainOrphanJournal) Append(context.Context, article.DeletionTombstoneRequest) (article.ArticleDeletionTombstone, error) {
	return journal.record, nil
}
func (journal mainOrphanJournal) FindByOperation(ctx context.Context, operationID string) (article.ArticleDeletionTombstone, error) {
	if journal.record.DeletionOperationID != operationID {
		return article.ArticleDeletionTombstone{}, article.ErrDeletionTombstoneNotFound
	}
	return journal.record, nil
}

func TestOrphanDispositionRoot_Finalizer_ProductionFactoryMapsCompletion(t *testing.T) {
	for _, kind := range []article.DeleteKind{article.DeleteOrphan, article.DeleteOrdinary, article.DeleteViolating} {
		t.Run(string(kind), func(t *testing.T) {
			fixture := newMainOrphanFixture(t)
			operationID, err := (identity.RandomUUIDGenerator{}).NewID()
			if err != nil {
				t.Fatal(err)
			}
			record := article.ArticleDeletionTombstone{Namespace: article.ArticleDeletionNamespace, Sequence: 1, ArticleID: mainOrphanArticleID,
				ArticleVersion: 3, DeletionOperationID: operationID, DeletedAt: fixture.now, KeyID: "fixture-key", KeyVersion: "v1"}
			fixture.state.operation = &article.DeletionOperation{ID: operationID, ArticleID: mainOrphanArticleID, ArticleVersion: 3,
				DeleteKind: kind, ActorUserID: mainOrphanStringPointer(mainOrphanTargetID), ActorRole: mainOrphanStringPointer("admin"),
				Status: article.DeletionOperationJournaled, TombstoneSequence: &record.Sequence, JournalKeyID: &record.KeyID, JournalKeyVersion: &record.KeyVersion, StartedAt: fixture.now}
			if kind == article.DeleteOrphan {
				fixture.state.operation.OrphanedFromUserID = mainOrphanStringPointer(mainOrphanSourceID)
			} else {
				fixture.state.record.AuthorID, fixture.state.record.OrphanedFromUserID = mainOrphanStringPointer(mainOrphanSourceID), nil
				fixture.state.record.PublicationState = article.PublicationPublished
			}
			fixture.state.record.DeletionOperationID, fixture.state.record.Version, fixture.state.record.DeletionState = &operationID, 4, article.DeletionPending
			finalizer, err := article.NewDeletionFinalizer(article.DeletionFinalizerDependencies{Transactions: fixture.transactor, Repository: article.Repository{},
				Journal: mainOrphanJournal{record: record}, Auditor: audit.Store{}, AuditIDs: identity.RandomUUIDGenerator{},
				Clock: mainOrphanClock{now: fixture.now}, WorkerID: "root-finalizer", LeaseDuration: time.Minute, Identity: fixture.identity})
			if err != nil {
				t.Fatal(err)
			}
			if err := finalizer.Finalize(context.Background(), operationID, mainOrphanTraceID); err != nil {
				t.Fatal(err)
			}
			if fixture.state.sourceEmail != nil || !fixture.state.absent || fixture.state.operation.DeleteKind != kind ||
				fixture.identity.proof.Reference().Action != article.DispositionDeleteComplete {
				t.Fatal("finalizer factory lost its completion binding")
			}
			if err := finalizer.Finalize(context.Background(), operationID, mainOrphanTraceID); err != nil || len(fixture.state.audits) != 2 {
				t.Fatal("completed replay duplicated audit")
			}
		})
	}
}

func TestOrphanDispositionConfig_IndependentPositiveBigint_NoFallback(t *testing.T) {
	for _, input := range []string{"", "0", "-1", "+1", "01", " 7", "7 ", "1.5", "9223372036854775808"} {
		values := validCompositionEnvironment()
		values[articleTransferNoticeMaxAttemptsEnvironment] = input
		config, err := loadStartupConfig(func(key string) string { return values[key] }, mapEnvironmentLookup(values))
		clearStartupSecrets(&config)
		if err == nil {
			t.Fatalf("invalid max_attempts accepted: %q", input)
		}
	}
	for _, input := range []string{"1", "7", "9223372036854775807"} {
		values := validCompositionEnvironment()
		values[articleTransferNoticeMaxAttemptsEnvironment] = input
		config := mustLoadStartupConfig(t, values)
		if strconv.FormatInt(config.articleTransferNoticeMaxAttempts, 10) != input {
			t.Fatal("notification retry policy was replaced by another policy")
		}
		clearStartupSecrets(&config)
	}
}

type mainOrphanResolver struct {
	principal session.Principal
	calls     int
	csrf      []bool
}

func (resolver *mainOrphanResolver) ResolvePrincipal(ctx context.Context, input session.ResolveInput) (session.Principal, error) {
	resolver.calls++
	resolver.csrf = append(resolver.csrf, input.RequireCSRF)
	if input.RequireCSRF && input.CSRFToken != "synthetic-csrf" {
		return session.Principal{}, session.ErrCSRFFailed
	}
	return resolver.principal, nil
}

func TestOrphanDispositionProductionRoutes_RealHandlersEnforceSecurity(t *testing.T) {
	for _, action := range []string{"list", "detail", "publish", "transfer", "delete"} {
		for _, variant := range []string{"session", "member", "password restricted", "privacy restricted", "method", "origin", "csrf", "precondition"} {
			if (action == "list" || action == "detail") && (variant == "origin" || variant == "csrf" || variant == "precondition") {
				continue
			}
			t.Run(action+"/"+variant, func(t *testing.T) { requireOrphanProductionRejection(t, action, variant) })
		}
	}
}

func requireOrphanProductionRejection(t *testing.T, action, variant string) {
	t.Helper()
	resolver := &mainOrphanResolver{principal: session.Principal{UserID: mainOrphanTargetID, Role: "admin", Status: "active"}}
	if variant == "member" {
		resolver.principal.Role = "member"
	}
	if variant == "password restricted" {
		resolver.principal.MustChangePassword = true
	}
	if variant == "privacy restricted" {
		resolver.principal.MustAcknowledgePrivacy = true
	}
	var events []string
	handler := newArticleProductionHandlerWithTestSeam(t, &articleProductionTestSeam{
		principalResolver: resolver,
		trace:             func(*http.Request) { events = append(events, "trace") },
	})
	path, method := mainOrphanRoute(action)
	if variant == "method" {
		method = http.MethodPatch
	}
	request := httptest.NewRequest(method, path, strings.NewReader(`{"new_author_id":"`+mainOrphanTargetID+`"}`))
	request.Header.Set("Origin", "https://community.example.invalid")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(platform.CSRFHeaderName, "synthetic-csrf")
	request.Header.Set("If-Match", `"article-`+mainOrphanArticleID+`-v3"`)
	if variant != "session" {
		request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: "synthetic-session"})
	}
	if variant == "origin" {
		request.Header.Set("Origin", "https://other.example.invalid")
	}
	if variant == "csrf" {
		request.Header.Del(platform.CSRFHeaderName)
	}
	if variant == "precondition" || variant == "member" {
		request.Header.Del("If-Match")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	status := http.StatusForbidden
	switch variant {
	case "session":
		status = http.StatusUnauthorized
	case "method":
		status = http.StatusMethodNotAllowed
	case "precondition":
		status = http.StatusPreconditionRequired
	}
	if response.Code != status {
		t.Fatalf("status=%d, want=%d", response.Code, status)
	}
	if variant == "method" {
		if got, want := response.Header().Get("Allow"), mainOrphanAllow(t, action); got != want {
			t.Fatalf("Allow=%q, want=%q", got, want)
		}
		if !reflect.DeepEqual(events, []string{"trace"}) || resolver.calls != 0 {
			t.Fatalf("method rejection events=%v resolverCalls=%d, want trace only", events, resolver.calls)
		}
	}
}

func mainOrphanAllow(t *testing.T, action string) string {
	t.Helper()
	switch action {
	case "list":
		return http.MethodGet
	case "detail", "delete":
		return http.MethodGet + ", " + http.MethodDelete
	case "publish", "transfer":
		return http.MethodPost
	default:
		t.Fatalf("unexpected orphan action %q", action)
		return ""
	}
}

func mainOrphanRoute(action string) (string, string) {
	base := "/api/v1/admin/orphan-articles"
	switch action {
	case "list":
		return base, http.MethodGet
	case "detail":
		return base + "/" + mainOrphanArticleID, http.MethodGet
	case "delete":
		return base + "/" + mainOrphanArticleID, http.MethodDelete
	default:
		return base + "/" + mainOrphanArticleID + "/" + action, http.MethodPost
	}
}

func TestOrphanDispositionProductionRoutes_ExactPathsAndSharedAdapter(t *testing.T) {
	instances := make(map[string]any)
	resolver := &mainOrphanResolver{principal: session.Principal{UserID: mainOrphanTargetID, Role: "admin", Status: "active"}}
	seam := &articleProductionTestSeam{principalResolver: resolver, shared: func(dependency articleSharedDependency) {
		if dependency.Kind == "identity_lifecycle" {
			instances[dependency.Consumer] = dependency.Instance
		}
	}}
	for index := range seam.orphanHandlers {
		seam.orphanHandlers[index] = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) { response.WriteHeader(http.StatusNoContent) })
	}
	handler := newArticleProductionHandlerWithTestSeam(t, seam)
	if instances["article_command"] == nil || instances["article_command"] != instances["article_finalizer"] {
		t.Fatal("command and finalizer did not receive the same production adapter")
	}
	for _, action := range []string{"list", "detail", "publish", "transfer"} {
		path, method := mainOrphanRoute(action)
		request := httptest.NewRequest(method, path, nil)
		request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: "synthetic-session"})
		request.Header.Set("Origin", "https://community.example.invalid")
		request.Header.Set(platform.CSRFHeaderName, "synthetic-csrf")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("registered route %s status=%d", action, response.Code)
		}
		suffixes := []string{"/", "/extra"}
		if action == "list" {
			suffixes = []string{"/", "/extra/other"}
		}
		for _, suffix := range suffixes {
			response = httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(method, path+suffix, nil))
			if response.Code != http.StatusNotFound {
				t.Fatalf("malformed route status=%d", response.Code)
			}
		}
	}
}

func TestOrphanDispositionRoot_IdentityErrors_PreserveClassification(t *testing.T) {
	for _, cause := range []error{identity.ErrArticleTransferTargetInvalid, &pgconn.PgError{Code: "55P03"}, errors.New("identifier drift")} {
		mapped := articleIdentityBoundaryError(cause)
		if !errors.Is(mapped, cause) {
			t.Fatal("root lost the original cause")
		}
		code := platform.PublicCode(article.MapArticleError(mapped))
		expected := platform.CodePlatformInternalError
		if errors.Is(cause, identity.ErrArticleTransferTargetInvalid) {
			expected = article.CodeArticleTransferTargetInvalid
		}
		var postgresError *pgconn.PgError
		if errors.As(cause, &postgresError) {
			expected = platform.CodePlatformDatabaseUnavailable
		}
		if code != expected {
			t.Fatalf("error code=%s, want=%s", code, expected)
		}
	}
}

func TestOrphanDispositionRoot_HTTPNOWAIT_ReturnsDatabaseUnavailableWithoutLoop(t *testing.T) {
	fixture := newMainOrphanFixture(t)
	fixture.state.userError = &pgconn.PgError{Code: "55P03", Message: "row lock unavailable"}
	handler, err := article.NewOrphanCommandHandler(&article.Handler{}, fixture.service, nil)
	if err != nil {
		t.Fatal(err)
	}
	resolver := &mainOrphanResolver{principal: session.Principal{UserID: mainOrphanTargetID, Role: "admin", Status: "active"}}
	authorizer, err := platform.NewHTTPAuthorizer(resolver, func() time.Time { return fixture.now })
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/orphan-articles/"+mainOrphanArticleID+"/publish", nil).WithContext(fixture.ctx)
	request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: "synthetic-session"})
	request.Header.Set(platform.CSRFHeaderName, "synthetic-csrf")
	request.Header.Set("If-Match", `"article-`+mainOrphanArticleID+`-v3"`)
	response := httptest.NewRecorder()
	authorizer.Protect(session.AccessGeneral, http.HandlerFunc(handler.PublishOrphan)).ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), string(platform.CodePlatformDatabaseUnavailable)) ||
		len(fixture.transactor.transactions) != 1 || fixture.transactor.transactions[0].identityReads != 1 || fixture.transactor.transactions[0].commitCalls != 0 {
		t.Fatalf("NOWAIT response/attempts mismatch: status=%d", response.Code)
	}
}

func TestOrphanDispositionConfig_MissingSnapshotLimit_RejectsStartup(t *testing.T) {
	values := validCompositionEnvironment()
	delete(values, "HAN_ARTICLE_TRANSFER_NOTICE_MAX_ATTEMPTS")
	config, err := loadStartupConfig(func(key string) string { return values[key] }, mapEnvironmentLookup(values))
	clearStartupSecrets(&config)
	if err == nil {
		t.Fatal("missing transfer notification max_attempts was accepted")
	}
}

func TestOrphanDispositionRouter_ListRoute_IsRegistered(t *testing.T) {
	handlers := make([]http.Handler, 17)
	handlers[13] = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) { response.WriteHeader(http.StatusNoContent) })
	router := newLoginBusinessRouter(nil, handlers...)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/admin/orphan-articles", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("orphan list route status=%d", response.Code)
	}
}

func TestModeration_Router_RegistersExactPOSTPaths(t *testing.T) {
	for _, action := range []string{"violation", "restore"} {
		t.Run(action, func(t *testing.T) {
			handlers := make([]http.Handler, 13)
			index := 11
			if action == "restore" {
				index = 12
			}
			handlers[index] = http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				response.WriteHeader(http.StatusNoContent)
			})
			router := newLoginBusinessRouter(nil, handlers...)
			path := "/api/v1/admin/articles/10000000-0000-4000-8000-000000000001/" + action
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, nil))
			if response.Code != http.StatusNoContent {
				t.Fatalf("POST %s = %d %s, want registered handler", path, response.Code, response.Body)
			}
		})
	}
}

func TestModeration_Production_RealHandlerSecurityAndRouting(t *testing.T) {
	for _, action := range []string{"violation", "restore"} {
		t.Run(action, func(t *testing.T) {
			for _, variant := range []string{"origin", "session", "csrf", "member", "password restricted", "privacy restricted", "precondition", "invalid body", "method", "trailing slash", "extra segment", "missing id"} {
				t.Run(variant, func(t *testing.T) { requireModerationProductionRejection(t, action, variant) })
			}
		})
	}
}

func requireModerationProductionRejection(t *testing.T, action, variant string) {
	t.Helper()
	resolver := &mainModerationResolver{principal: session.Principal{UserID: "20000000-0000-4000-8000-000000000002", Role: "admin", Status: "active"}}
	if variant == "member" {
		resolver.principal.Role = "member"
	}
	if variant == "password restricted" {
		resolver.principal.MustChangePassword = true
	}
	if variant == "privacy restricted" {
		resolver.principal.MustAcknowledgePrivacy = true
	}
	var authorized []string
	handler := newArticleProductionHandlerWithTestSeam(t, &articleProductionTestSeam{
		principalResolver: resolver,
		authorized:        func(route string) { authorized = append(authorized, route) },
	})
	path := "/api/v1/admin/articles/10000000-0000-4000-8000-000000000001/" + action
	method, body := http.MethodPost, `{"category":"unknown"}`
	if action == "restore" {
		body = `{}`
	}
	status, code := http.StatusForbidden, platform.CodeAuthForbidden
	switch variant {
	case "origin":
		code = platform.CodeAuthOriginRejected
	case "session":
		status, code = http.StatusUnauthorized, platform.CodeAuthSessionRequired
	case "csrf":
		code = platform.CodeAuthCSRFFailed
	case "password restricted", "privacy restricted":
		code = platform.CodeAuthMustCompleteOnboarding
	case "precondition":
		status, code = http.StatusPreconditionRequired, article.CodeRequestPreconditionRequired
	case "invalid body":
		status, code = http.StatusUnprocessableEntity, article.CodeArticleValidationFailed
		if action == "restore" {
			status, code = http.StatusBadRequest, platform.CodeRequestInvalidJSON
		}
	case "method":
		method, status, code = http.MethodGet, http.StatusMethodNotAllowed, platform.CodeRequestMethodNotAllowed
	case "trailing slash":
		path, status, code = path+"/", http.StatusNotFound, platform.CodeRequestRouteNotFound
	case "extra segment":
		path, status, code = path+"/more", http.StatusNotFound, platform.CodeRequestRouteNotFound
	case "missing id":
		path, status, code = "/api/v1/admin/articles//"+action, http.StatusNotFound, platform.CodeRequestRouteNotFound
	}
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if variant != "origin" {
		request.Header.Set("Origin", "https://community.example.invalid")
	}
	if variant != "session" {
		request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: "synthetic-session"})
	}
	if variant != "csrf" {
		request.Header.Set(platform.CSRFHeaderName, "synthetic-csrf")
	}
	if variant != "precondition" {
		request.Header.Set("If-Match", `"article-10000000-0000-4000-8000-000000000001-v1"`)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	var envelope platform.Envelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if response.Code != status || envelope.Code != code || envelope.TraceID == "" || response.Header().Get(platform.TraceHeader) != envelope.TraceID {
		t.Fatalf("response = %d %#v, want %d/%s", response.Code, envelope, status, code)
	}
	if variant == "method" && response.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("Allow = %q", response.Header().Get("Allow"))
	}
	reachedResolver := variant != "origin" && variant != "session" && status != http.StatusNotFound && variant != "method"
	if (resolver.calls == 1) != reachedResolver {
		t.Fatalf("resolver calls = %d", resolver.calls)
	}
	reachedHandler := variant == "member" || variant == "precondition" || variant == "invalid body"
	if (len(authorized) == 1) != reachedHandler {
		t.Fatalf("authorized routes = %v", authorized)
	}
}

type mainModerationResolver struct {
	principal session.Principal
	calls     int
}

func (resolver *mainModerationResolver) ResolvePrincipal(_ context.Context, input session.ResolveInput) (session.Principal, error) {
	resolver.calls++
	if !input.RequireCSRF || input.CSRFToken != "synthetic-csrf" {
		return session.Principal{}, session.ErrCSRFFailed
	}
	return resolver.principal, nil
}

func TestModeration_Production_AuthorReaderUsesCurrentTransaction(t *testing.T) {
	for _, status := range []identity.UserStatus{identity.UserStatusActive, identity.UserStatusSuspended} {
		t.Run(string(status), func(t *testing.T) {
			rows := &mainModerationRows{authorID: "20000000-0000-4000-8000-000000000002", status: status}
			transaction := &mainModerationReadTransaction{rows: rows}
			states, err := moderationAuthorStates(context.Background(), transaction, []string{rows.authorID})
			if err != nil || !reflect.DeepEqual(states, map[string]string{rows.authorID: string(status)}) || transaction.queries != 1 || !rows.closed {
				t.Fatalf("reader = %v/%v, queries = %d, closed = %v", states, err, transaction.queries, rows.closed)
			}
		})
	}
}

type mainModerationReadTransaction struct {
	pgx.Tx
	rows    *mainModerationRows
	queries int
}

func (transaction *mainModerationReadTransaction) Query(context.Context, string, ...any) (pgx.Rows, error) {
	transaction.queries++
	return transaction.rows, nil
}

type mainModerationRows struct {
	pgx.Rows
	authorID string
	status   identity.UserStatus
	scanned  bool
	closed   bool
}

func (rows *mainModerationRows) Next() bool { return !rows.scanned }
func (rows *mainModerationRows) Close()     { rows.closed = true }
func (*mainModerationRows) Err() error      { return nil }
func (rows *mainModerationRows) Scan(destinations ...any) error {
	*destinations[0].(*string), *destinations[1].(*identity.UserStatus) = rows.authorID, rows.status
	rows.scanned = true
	return nil
}

func TestRun_StartupGate(t *testing.T) {
	t.Parallel()

	databaseConfigErr := errors.New("database configuration failed")
	connectionErr := errors.New("connection failed")
	migrationErr := errors.New("migration failed")
	listenErr := errors.New("listen failed")
	serveErr := errors.New("serve failed")

	tests := []struct {
		name              string
		dsn               string
		httpAddress       string
		databaseConfigErr error
		connectErr        error
		pingErr           error
		migrationErr      error
		listenErr         error
		serveErr          error
		missingConfigKey  string
		wantCause         error
		wantCalls         []string
	}{
		{
			name:        "missing DSN",
			httpAddress: "127.0.0.1:8080",
		},
		{
			name:              "invalid DSN",
			dsn:               "invalid",
			httpAddress:       "127.0.0.1:8080",
			databaseConfigErr: databaseConfigErr,
			wantCause:         databaseConfigErr,
			wantCalls:         []string{"validate-database"},
		},
		{
			name:      "missing HTTP address",
			dsn:       "host=example.invalid user=han dbname=han",
			wantCalls: []string{"validate-database"},
		},
		{
			name:        "invalid HTTP address",
			dsn:         "host=example.invalid user=han dbname=han",
			httpAddress: "localhost:8080",
			wantCalls:   []string{"validate-database"},
		},
		{
			name:             "missing deployment security configuration",
			dsn:              "host=example.invalid user=han dbname=han",
			httpAddress:      "127.0.0.1:8080",
			missingConfigKey: platform.EnvPublicOrigin,
		},
		{
			name:        "connection creation fails",
			dsn:         "host=example.invalid user=han dbname=han",
			httpAddress: "127.0.0.1:8080",
			connectErr:  connectionErr,
			wantCause:   connectionErr,
			wantCalls:   []string{"validate-database", "connect"},
		},
		{
			name:        "connection validation fails",
			dsn:         "host=example.invalid user=han dbname=han",
			httpAddress: "127.0.0.1:8080",
			pingErr:     connectionErr,
			wantCause:   connectionErr,
			wantCalls:   []string{"validate-database", "connect", "ping", "close-pool"},
		},
		{
			name:         "migration fails",
			dsn:          "host=example.invalid user=han dbname=han",
			httpAddress:  "127.0.0.1:8080",
			migrationErr: migrationErr,
			wantCause:    migrationErr,
			wantCalls:    []string{"validate-database", "connect", "ping", "migrate", "close-pool"},
		},
		{
			name:        "listener fails after migration",
			dsn:         "host=example.invalid user=han dbname=han",
			httpAddress: "127.0.0.1:8080",
			listenErr:   listenErr,
			wantCause:   listenErr,
			wantCalls:   []string{"validate-database", "connect", "ping", "migrate", "listen", "close-pool"},
		},
		{
			name:        "Serve failure closes listener then pool",
			dsn:         "host=example.invalid user=han dbname=han",
			httpAddress: "127.0.0.1:8080",
			serveErr:    serveErr,
			wantCause:   serveErr,
			wantCalls: []string{
				"validate-database",
				"connect",
				"ping",
				"migrate",
				"listen",
				"serve",
				"close-listener",
				"close-pool",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var calls []string
			pool := new(pgxpool.Pool)
			listener := &recordingListener{
				onClose: func() {
					calls = append(calls, "close-listener")
				},
			}
			deploymentValues := validStartupDeploymentValues()
			delete(deploymentValues, tt.missingConfigKey)
			deps := startupDependencies{
				getenv: func(key string) string {
					switch key {
					case databaseDSNEnvironment:
						return tt.dsn
					case httpAddressEnvironment:
						return tt.httpAddress
					default:
						t.Fatalf("unexpected environment lookup %q", key)
						return ""
					}
				},
				lookupEnvironment: func(key string) (string, bool) {
					value, exists := deploymentValues[key]
					return value, exists
				},
				parseDatabase: func(string) (*pgxpool.Config, error) {
					calls = append(calls, "validate-database")
					return new(pgxpool.Config), tt.databaseConfigErr
				},
				connect: func(context.Context, *pgxpool.Config) (*pgxpool.Pool, error) {
					calls = append(calls, "connect")
					return pool, tt.connectErr
				},
				ping: func(context.Context, *pgxpool.Pool) error {
					calls = append(calls, "ping")
					return tt.pingErr
				},
				migrate: func(context.Context, *pgxpool.Pool) error {
					calls = append(calls, "migrate")
					return tt.migrationErr
				},
				listen: func(network, address string) (net.Listener, error) {
					calls = append(calls, "listen")
					if network != "tcp" {
						t.Fatalf("listen network = %q, want tcp", network)
					}
					if address != tt.httpAddress {
						t.Fatalf("listen address = %q, want %q", address, tt.httpAddress)
					}
					if tt.listenErr != nil {
						return nil, tt.listenErr
					}
					return listener, nil
				},
				serve: func(got net.Listener) error {
					calls = append(calls, "serve")
					if got != listener {
						t.Fatal("serve listener did not match listen result")
					}
					return tt.serveErr
				},
				closePool: func(got *pgxpool.Pool) {
					calls = append(calls, "close-pool")
					if got != pool {
						t.Fatal("closed pool did not match connect result")
					}
				},
			}

			err := run(context.Background(), deps)
			if err == nil {
				t.Fatal("run() error = nil, want fail-closed startup error")
			}
			if tt.wantCause != nil && !errors.Is(err, tt.wantCause) {
				t.Fatalf("run() error = %v, want wrapped cause %v", err, tt.wantCause)
			}
			if tt.missingConfigKey != "" {
				if !strings.Contains(err.Error(), tt.missingConfigKey) {
					t.Fatalf("run() error = %q, want missing configuration key", err)
				}
				for _, value := range deploymentValues {
					if value != "" && strings.Contains(err.Error(), value) {
						t.Fatal("run() error leaked deployment security configuration")
					}
				}
			}
			if !reflect.DeepEqual(calls, tt.wantCalls) {
				t.Fatalf("calls = %v, want %v", calls, tt.wantCalls)
			}
		})
	}
}

func TestRun_HANTestBaseURLDoesNotConfigureProductionListener(t *testing.T) {
	t.Parallel()

	const testBaseURL = "http://192.0.2.10:9000"
	tests := []struct {
		name        string
		httpAddress string
		wantListen  bool
	}{
		{name: "missing production address does not fall back"},
		{name: "invalid production address is not overridden", httpAddress: "localhost:8080"},
		{name: "valid production address remains authoritative", httpAddress: "127.0.0.1:8080", wantListen: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var lookedUp []string
			listened := false
			deps := startupDependencies{
				getenv: func(key string) string {
					lookedUp = append(lookedUp, key)
					switch key {
					case databaseDSNEnvironment:
						return "host=example.invalid user=han dbname=han"
					case httpAddressEnvironment:
						return tt.httpAddress
					case "HAN_TEST_BASE_URL":
						return testBaseURL
					default:
						return ""
					}
				},
				lookupEnvironment: mapEnvironmentLookup(validStartupDeploymentValues()),
				parseDatabase: func(string) (*pgxpool.Config, error) {
					return new(pgxpool.Config), nil
				},
				connect: func(context.Context, *pgxpool.Config) (*pgxpool.Pool, error) {
					return new(pgxpool.Pool), nil
				},
				ping:    func(context.Context, *pgxpool.Pool) error { return nil },
				migrate: func(context.Context, *pgxpool.Pool) error { return nil },
				listen: func(_ string, address string) (net.Listener, error) {
					listened = true
					if address == strings.TrimPrefix(testBaseURL, "http://") {
						t.Fatal("listener used HAN_TEST_BASE_URL")
					}
					if address != tt.httpAddress {
						t.Fatalf("listen address = %q, want %q", address, tt.httpAddress)
					}
					return &recordingListener{}, nil
				},
				serve:     func(net.Listener) error { return errors.New("stop test server") },
				closePool: func(*pgxpool.Pool) {},
			}

			if err := run(context.Background(), deps); err == nil {
				t.Fatal("run() error = nil, want fail-closed return")
			}
			if listened != tt.wantListen {
				t.Fatalf("listen called = %v, want %v", listened, tt.wantListen)
			}
			if !reflect.DeepEqual(lookedUp, []string{databaseDSNEnvironment, httpAddressEnvironment}) {
				t.Fatalf("environment lookups = %v, want only production keys", lookedUp)
			}
		})
	}
}

func TestMain_ConnectionFailureIsNonZeroAndDoesNotLogConfiguration(t *testing.T) {
	const secret = "b0-review-secret"
	dsn := "postgres://han:" + secret + "@127.0.0.1:1/han?connect_timeout=1"
	httpAddress := "127.0.0.1:65535"

	cmd := exec.Command(os.Args[0], "-test.run=^TestMainHelperProcess$")
	cmd.Env = append(
		environmentWithout(append(
			[]string{databaseDSNEnvironment, httpAddressEnvironment, mainHelperEnvironment},
			deploymentEnvironmentKeys()...,
		)...),
		databaseDSNEnvironment+"="+dsn,
		httpAddressEnvironment+"="+httpAddress,
		mainHelperEnvironment+"=1",
	)
	cmd.Env = append(cmd.Env, deploymentEnvironmentEntries(validStartupDeploymentValues())...)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("main process exit error = nil, want non-zero exit")
	}
	if strings.Contains(string(output), secret) ||
		strings.Contains(string(output), dsn) ||
		strings.Contains(string(output), httpAddress) {
		t.Fatalf("main process output exposed configuration: %q", output)
	}
	for _, value := range validStartupDeploymentValues() {
		if value != "" && strings.Contains(string(output), value) {
			t.Fatalf("main process output exposed deployment security configuration")
		}
	}
	if !strings.Contains(string(output), startupFailureMessage) {
		t.Fatalf("main process output = %q, want generic startup failure", output)
	}
}

type recordingListener struct {
	onClose func()
}

func (*recordingListener) Accept() (net.Conn, error) {
	return nil, errors.New("Accept is not used in startup orchestration tests")
}

func (listener *recordingListener) Close() error {
	if listener.onClose != nil {
		listener.onClose()
	}
	return nil
}

func (*recordingListener) Addr() net.Addr {
	return &net.TCPAddr{}
}

func TestMainHelperProcess(t *testing.T) {
	if os.Getenv(mainHelperEnvironment) != "1" {
		return
	}
	os.Args = []string{os.Args[0]}
	main()
}

func environmentWithout(keys ...string) []string {
	excluded := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		excluded[strings.ToUpper(key)] = struct{}{}
	}

	env := make([]string, 0, len(os.Environ()))
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if _, found := excluded[strings.ToUpper(key)]; !found {
			env = append(env, entry)
		}
	}
	return env
}

func validStartupDeploymentValues() map[string]string {
	return map[string]string{
		platform.EnvProcessingRegionRef:     "synthetic-region-ref",
		platform.EnvProcessingControllerRef: "synthetic-controller-ref",
		platform.EnvProcessingLegalBasisRef: "synthetic-basis-ref",
		platform.EnvSMTPProcessorRef:        "synthetic-smtp-processor-ref",
		platform.EnvPublicOrigin:            "https://community.example.invalid",
		platform.EnvTrustedProxyCIDRs:       "192.0.2.0/24",
		platform.EnvSMTPTLSMode:             "starttls_required",
		platform.EnvSMTPHost:                "smtp.example.invalid",
		platform.EnvSMTPServerName:          "smtp.example.invalid",
		platform.EnvSMTPCAPath:              "",
	}
}

func mapEnvironmentLookup(values map[string]string) platform.EnvironmentLookup {
	return func(key string) (string, bool) {
		value, exists := values[key]
		return value, exists
	}
}

func deploymentEnvironmentKeys() []string {
	return []string{
		platform.EnvProcessingRegionRef,
		platform.EnvProcessingControllerRef,
		platform.EnvProcessingLegalBasisRef,
		platform.EnvSMTPProcessorRef,
		platform.EnvPublicOrigin,
		platform.EnvTrustedProxyCIDRs,
		platform.EnvSMTPTLSMode,
		platform.EnvSMTPHost,
		platform.EnvSMTPServerName,
		platform.EnvSMTPCAPath,
	}
}

func deploymentEnvironmentEntries(values map[string]string) []string {
	entries := make([]string, 0, len(values))
	for _, key := range deploymentEnvironmentKeys() {
		entries = append(entries, key+"="+values[key])
	}
	return entries
}

func TestLoadStartupConfig_ObservabilityMatrix(t *testing.T) {
	tests := []struct {
		name      string
		overrides map[string]string
		wantErr   bool
	}{
		{name: "deployment baseline"},
		{name: "legal non baseline", overrides: map[string]string{
			loginObservationBudgetEnvironment:  "1ms",
			originObservationBudgetEnvironment: "5s",
			failureAuditBudgetEnvironment:      "10ms",
			shutdownBudgetEnvironment:          "60s",
			smtpOperationBudgetEnvironment:     "17s",
			metricsAddressEnvironment:          "[::1]:65535",
			loginDurationBucketsEnvironment:    "0.001,0.2,60",
		}},
		{name: "login below range", overrides: map[string]string{
			loginObservationBudgetEnvironment: "999us",
		}, wantErr: true},
		{name: "origin above range", overrides: map[string]string{
			originObservationBudgetEnvironment: "5s1ns",
		}, wantErr: true},
		{name: "failure below range", overrides: map[string]string{
			failureAuditBudgetEnvironment: "9ms",
		}, wantErr: true},
		{name: "shutdown above range", overrides: map[string]string{
			shutdownBudgetEnvironment: "61s",
		}, wantErr: true},
		{name: "smtp lower endpoint", overrides: map[string]string{
			smtpOperationBudgetEnvironment: "1s",
		}},
		{name: "smtp upper endpoint", overrides: map[string]string{
			smtpOperationBudgetEnvironment: "60s",
		}, wantErr: true},
		{name: "smtp below range", overrides: map[string]string{
			smtpOperationBudgetEnvironment: "999ms",
		}, wantErr: true},
		{name: "smtp above range", overrides: map[string]string{
			smtpOperationBudgetEnvironment: "60s1ns",
		}, wantErr: true},
		{name: "smtp missing", overrides: map[string]string{
			smtpOperationBudgetEnvironment: "",
		}, wantErr: true},
		{name: "missing duration", overrides: map[string]string{
			loginObservationBudgetEnvironment: "",
		}, wantErr: true},
		{name: "invalid duration", overrides: map[string]string{
			failureAuditBudgetEnvironment: "invalid",
		}, wantErr: true},
		{name: "metrics hostname", overrides: map[string]string{
			metricsAddressEnvironment: "localhost:9090",
		}, wantErr: true},
		{name: "metrics URL", overrides: map[string]string{
			metricsAddressEnvironment: "http://127.0.0.1:9090",
		}, wantErr: true},
		{name: "invalid buckets", overrides: map[string]string{
			loginDurationBucketsEnvironment: "1,1",
		}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := validCompositionEnvironment()
			for key, value := range test.overrides {
				values[key] = value
			}
			config, err := loadStartupConfig(
				func(key string) string { return values[key] },
				mapEnvironmentLookup(values),
			)
			if test.wantErr {
				if err == nil {
					t.Fatal("loadStartupConfig() error = nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("loadStartupConfig() error = %v", err)
			}
			if config.loginObservationBudget <= 0 ||
				config.originObservationBudget <= 0 ||
				config.failureAuditBudget <= 0 ||
				config.shutdownBudget <= 0 ||
				config.smtpOperationBudget <= 0 ||
				len(config.loginDurationBuckets) == 0 {
				t.Fatalf("incomplete config = %#v", config)
			}
		})
	}
}

func TestRun_SMTPBudgetFailureHasNoStartupSideEffects(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "missing"},
		{name: "invalid", value: "not-a-duration"},
		{name: "below range", value: "999ms"},
		{name: "above range", value: "60s1ns"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := validCompositionEnvironment()
			values[smtpOperationBudgetEnvironment] = test.value
			dependencies := validCompositionDependencies(values)
			var calls []string
			dependencies.prepareBusiness = func(startupConfig) (*preparedLoginDependencies, error) {
				calls = append(calls, "prepare")
				return nil, errors.New("must not prepare")
			}
			dependencies.parseDatabase = func(string) (*pgxpool.Config, error) {
				calls = append(calls, "parse-database")
				return nil, errors.New("must not parse database")
			}
			dependencies.connect = func(
				context.Context,
				*pgxpool.Config,
			) (*pgxpool.Pool, error) {
				calls = append(calls, "connect")
				return nil, errors.New("must not connect")
			}
			dependencies.ping = func(context.Context, *pgxpool.Pool) error {
				calls = append(calls, "ping")
				return errors.New("must not ping")
			}
			dependencies.migrate = func(context.Context, *pgxpool.Pool) error {
				calls = append(calls, "migrate")
				return errors.New("must not migrate")
			}
			dependencies.bootstrapInitialAdmin = func(
				context.Context,
				*pgxpool.Pool,
				*preparedLoginDependencies,
			) error {
				calls = append(calls, "bootstrap")
				return errors.New("must not bootstrap")
			}
			dependencies.composeBusiness = func(
				*pgxpool.Pool,
				businessComposition,
			) (http.Handler, error) {
				calls = append(calls, "compose")
				return nil, errors.New("must not compose")
			}
			dependencies.listen = func(string, string) (net.Listener, error) {
				calls = append(calls, "listen")
				return nil, errors.New("must not listen")
			}

			runErr := run(context.Background(), dependencies)
			if runErr == nil {
				t.Fatal("run() error = nil")
			}
			if len(calls) != 0 {
				t.Fatalf("startup side effects = %v, want none", calls)
			}
		})
	}
}

func TestRun_AdministratorInformationNoticeEffectiveAtFailureHasNoStartupSideEffects(t *testing.T) {
	tests := []string{
		"2026-10-24T00:00:00.1Z",
		"2026-10-24T00:00:00.123Z",
		"2026-10-24T00:00:00.000Z",
		"2026-10-24T08:00:00+08:00",
		"2026-10-24T00:00:00+00:00",
		"0001-01-01T00:00:00Z",
		"not-a-time",
	}
	for _, value := range tests {
		t.Run(value, func(t *testing.T) {
			values := validCompositionEnvironment()
			values["HAN_ADMINISTRATOR_INFORMATION_NOTICE_EFFECTIVE_AT"] = value
			dependencies := validCompositionDependencies(values)
			var calls []string
			dependencies.prepareBusiness = func(startupConfig) (*preparedLoginDependencies, error) {
				calls = append(calls, "prepare")
				return nil, errors.New("must not prepare")
			}
			dependencies.parseDatabase = func(string) (*pgxpool.Config, error) {
				calls = append(calls, "parse-database")
				return nil, errors.New("must not parse database")
			}
			dependencies.connect = func(
				context.Context,
				*pgxpool.Config,
			) (*pgxpool.Pool, error) {
				calls = append(calls, "connect")
				return nil, errors.New("must not connect")
			}
			dependencies.composeBusiness = func(
				*pgxpool.Pool,
				businessComposition,
			) (http.Handler, error) {
				calls = append(calls, "compose")
				return nil, errors.New("must not compose")
			}
			dependencies.listen = func(string, string) (net.Listener, error) {
				calls = append(calls, "listen")
				return nil, errors.New("must not listen")
			}

			err := run(context.Background(), dependencies)
			if err == nil {
				t.Fatal("run() error = nil")
			}
			if len(calls) != 0 {
				t.Fatalf("startup side effects = %v, want none", calls)
			}
			if strings.Contains(err.Error(), value) {
				t.Fatal("startup error exposed rejected effective-at value")
			}
		})
	}
}

func TestLoadStartupConfig_AdministratorProductionConfiguration(t *testing.T) {
	values := validCompositionEnvironment()
	config := mustLoadStartupConfig(t, values)
	defer clearStartupSecrets(&config)

	if got := config.administratorNotice.Version; got != values[administratorNoticeVersionEnvironment] {
		t.Fatalf("notice version = %q", got)
	}
	if got := config.administratorNotice.EffectiveAt.Format(time.RFC3339); got !=
		values[administratorNoticeEffectiveAtEnvironment] {
		t.Fatalf("notice effective_at = %q", got)
	}
	if config.administratorNotice.Content != values[administratorNoticeContentEnvironment] ||
		config.administratorNotice.ContentHash != values[administratorNoticeSHA256Environment] {
		t.Fatal("administrator notice content or hash changed during loading")
	}
	if config.idempotencyKeys.ActiveVersion != "v2" ||
		len(config.idempotencyKeys.Keys) != 2 ||
		string(config.idempotencyKeys.Keys["v1"]) != values[idempotencyKeyPrefix+"v1"] ||
		string(config.idempotencyKeys.Keys["v2"]) != values[idempotencyKeyPrefix+"v2"] {
		t.Fatal("idempotency keyring was not loaded exactly")
	}
	if config.idempotencyLock != 2*time.Minute ||
		config.idempotencyTTL != 24*time.Hour {
		t.Fatalf(
			"idempotency lock/ttl = %s/%s",
			config.idempotencyLock,
			config.idempotencyTTL,
		)
	}
	if config.smtpPort != 587 || config.smtpFrom != "no-reply@example.invalid" ||
		config.smtpAuthMode != "none" ||
		config.smtpUsername != "" || config.smtpPassword != "" {
		t.Fatalf("SMTP startup config = %#v", config)
	}
}

func TestRun_AdministratorProductionConfigurationFailuresHaveNoSideEffects(t *testing.T) {
	required := []string{
		administratorNoticeVersionEnvironment,
		administratorNoticeEffectiveAtEnvironment,
		administratorNoticeContentEnvironment,
		administratorNoticeSHA256Environment,
		idempotencyCurrentKeyVersionEnvironment,
		idempotencyRetainedKeyVersionsEnvironment,
		idempotencyKeyPrefix + "v1",
		idempotencyKeyPrefix + "v2",
		idempotencyLockDurationEnvironment,
		idempotencyRecordTTLEnvironment,
		smtpPortEnvironment,
		smtpFromEnvironment,
		smtpAuthModeEnvironment,
		smtpUsernameEnvironment,
		smtpPasswordEnvironment,
	}
	tests := make(map[string]func(map[string]string), len(required)+24)
	for _, key := range required {
		key := key
		tests["missing "+key] = func(values map[string]string) {
			delete(values, key)
		}
	}
	tests["notice version path"] = func(values map[string]string) {
		values[administratorNoticeVersionEnvironment] = "notice/path"
	}
	tests["notice version invalid first character"] = func(values map[string]string) {
		values[administratorNoticeVersionEnvironment] = "-notice"
	}
	tests["notice version too long"] = func(values map[string]string) {
		values[administratorNoticeVersionEnvironment] = strings.Repeat("a", 129)
	}
	tests["notice content whitespace"] = func(values map[string]string) {
		setAdministratorNoticeContent(values, " \r\n\t")
	}
	tests["notice content too long"] = func(values map[string]string) {
		setAdministratorNoticeContent(values, strings.Repeat("a", 65537))
	}
	tests["notice content invalid UTF-8"] = func(values map[string]string) {
		setAdministratorNoticeContent(values, string([]byte{0xff}))
	}
	tests["notice hash uppercase"] = func(values map[string]string) {
		values[administratorNoticeSHA256Environment] =
			strings.ToUpper(values[administratorNoticeSHA256Environment])
	}
	tests["notice hash invalid hex"] = func(values map[string]string) {
		values[administratorNoticeSHA256Environment] = strings.Repeat("g", 64)
	}
	tests["notice hash wrong length"] = func(values map[string]string) {
		values[administratorNoticeSHA256Environment] = strings.Repeat("a", 63)
	}
	tests["notice hash mismatch"] = func(values map[string]string) {
		values[administratorNoticeContentEnvironment] += "x"
	}
	tests["current version reserved"] = func(values map[string]string) {
		values[idempotencyCurrentKeyVersionEnvironment] = "none"
	}
	tests["current version too long"] = func(values map[string]string) {
		values[idempotencyCurrentKeyVersionEnvironment] = strings.Repeat("a", 65)
	}
	tests["retained unsorted"] = func(values map[string]string) {
		values[idempotencyRetainedKeyVersionsEnvironment] = "v2,v1"
	}
	tests["retained contains current"] = func(values map[string]string) {
		values[idempotencyRetainedKeyVersionsEnvironment] = "v1,v2"
	}
	tests["retained empty item"] = func(values map[string]string) {
		values[idempotencyRetainedKeyVersionsEnvironment] = "v1,"
	}
	tests["retained exceeds item limit"] = func(values map[string]string) {
		versions := make([]string, 17)
		for index := range versions {
			versions[index] = fmt.Sprintf("k%02d", index)
		}
		values[idempotencyRetainedKeyVersionsEnvironment] = strings.Join(versions, ",")
	}
	tests["idempotency key too short"] = func(values map[string]string) {
		values[idempotencyKeyPrefix+"v2"] = strings.Repeat("k", 31)
	}
	tests["idempotency key too long"] = func(values map[string]string) {
		values[idempotencyKeyPrefix+"v2"] = strings.Repeat("k", 65)
	}
	tests["idempotency key reused"] = func(values map[string]string) {
		values[idempotencyKeyPrefix+"v2"] = values[idempotencyKeyPrefix+"v1"]
	}
	tests["idempotency key cross domain reused"] = func(values map[string]string) {
		values[idempotencyKeyPrefix+"v2"] = values[sourceKeyPrefix+"v1"]
	}
	tests["lock equal SMTP budget"] = func(values map[string]string) {
		values[idempotencyLockDurationEnvironment] = values[smtpOperationBudgetEnvironment]
	}
	tests["lock below range"] = func(values map[string]string) {
		values[idempotencyLockDurationEnvironment] = "999ms"
	}
	tests["lock above range"] = func(values map[string]string) {
		values[idempotencyLockDurationEnvironment] = "5m1ns"
	}
	tests["ttl equal lock"] = func(values map[string]string) {
		values[idempotencyRecordTTLEnvironment] = values[idempotencyLockDurationEnvironment]
	}
	tests["ttl below range"] = func(values map[string]string) {
		values[idempotencyRecordTTLEnvironment] = "59m"
	}
	tests["ttl above range"] = func(values map[string]string) {
		values[idempotencyRecordTTLEnvironment] = "720h1ns"
	}
	tests["SMTP host contains port"] = func(values map[string]string) {
		values[platform.EnvSMTPHost] = "smtp.example.invalid:587"
	}
	tests["SMTP host contains path"] = func(values map[string]string) {
		values[platform.EnvSMTPHost] = "smtp.example.invalid/path"
	}
	tests["SMTP host contains zone"] = func(values map[string]string) {
		values[platform.EnvSMTPHost] = "fe80::1%eth0"
	}
	tests["SMTP host contains whitespace"] = func(values map[string]string) {
		values[platform.EnvSMTPHost] = " smtp.example.invalid"
	}
	tests["SMTP server name bracketed"] = func(values map[string]string) {
		values[platform.EnvSMTPServerName] = "[192.0.2.10]"
	}
	tests["SMTP port zero"] = func(values map[string]string) {
		values[smtpPortEnvironment] = "0"
	}
	tests["SMTP port overflow"] = func(values map[string]string) {
		values[smtpPortEnvironment] = "65536"
	}
	tests["SMTP port non decimal"] = func(values map[string]string) {
		values[smtpPortEnvironment] = "0x24b"
	}
	tests["SMTP from display name"] = func(values map[string]string) {
		values[smtpFromEnvironment] = "Han <no-reply@example.invalid>"
	}
	tests["SMTP from whitespace"] = func(values map[string]string) {
		values[smtpFromEnvironment] = " no-reply@example.invalid"
	}
	tests["SMTP from header injection"] = func(values map[string]string) {
		values[smtpFromEnvironment] = "no-reply@example.invalid\r\nBcc:x@example.invalid"
	}
	tests["SMTP auth unsupported"] = func(values map[string]string) {
		values[smtpAuthModeEnvironment] = "PLAIN"
	}
	tests["SMTP plain username missing"] = func(values map[string]string) {
		values[smtpAuthModeEnvironment] = "plain"
		values[smtpUsernameEnvironment] = ""
		values[smtpPasswordEnvironment] = "smtp-password-canary"
	}
	tests["SMTP plain password missing"] = func(values map[string]string) {
		values[smtpAuthModeEnvironment] = "plain"
		values[smtpUsernameEnvironment] = "smtp-user-canary"
		values[smtpPasswordEnvironment] = ""
	}
	tests["SMTP plain username whitespace"] = func(values map[string]string) {
		values[smtpAuthModeEnvironment] = "plain"
		values[smtpUsernameEnvironment] = " smtp-user-canary"
		values[smtpPasswordEnvironment] = "smtp-password-canary"
	}
	tests["SMTP plain username control"] = func(values map[string]string) {
		values[smtpAuthModeEnvironment] = "plain"
		values[smtpUsernameEnvironment] = "smtp-user\ncanary"
		values[smtpPasswordEnvironment] = "smtp-password-canary"
	}
	tests["SMTP plain password too long"] = func(values map[string]string) {
		values[smtpAuthModeEnvironment] = "plain"
		values[smtpUsernameEnvironment] = "smtp-user-canary"
		values[smtpPasswordEnvironment] = strings.Repeat("p", 1025)
	}
	tests["SMTP none carries credentials"] = func(values map[string]string) {
		values[smtpAuthModeEnvironment] = "none"
		values[smtpUsernameEnvironment] = "smtp-user-canary"
		values[smtpPasswordEnvironment] = "smtp-password-canary"
	}
	tests["SMTP CA path relative"] = func(values map[string]string) {
		values[platform.EnvSMTPCAPath] = "relative/ca.pem"
	}
	tests["SMTP CA path whitespace"] = func(values map[string]string) {
		values[platform.EnvSMTPCAPath] = " " + filepath.Join(t.TempDir(), "ca.pem")
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			values := validCompositionEnvironment()
			mutate(values)
			dependencies := validCompositionDependencies(values)
			var calls []string
			dependencies.prepareBusiness = func(startupConfig) (*preparedLoginDependencies, error) {
				calls = append(calls, "prepare")
				return nil, errors.New("must not prepare")
			}
			dependencies.parseDatabase = func(string) (*pgxpool.Config, error) {
				calls = append(calls, "parse")
				return nil, errors.New("must not parse")
			}
			dependencies.connect = func(context.Context, *pgxpool.Config) (*pgxpool.Pool, error) {
				calls = append(calls, "connect")
				return nil, errors.New("must not connect")
			}
			dependencies.composeBusiness = func(
				*pgxpool.Pool,
				businessComposition,
			) (http.Handler, error) {
				calls = append(calls, "factory")
				return nil, errors.New("must not compose")
			}
			dependencies.listen = func(string, string) (net.Listener, error) {
				calls = append(calls, "listen")
				return nil, errors.New("must not listen")
			}

			err := run(context.Background(), dependencies)
			if err == nil {
				t.Fatal("run() error = nil")
			}
			if len(calls) != 0 {
				t.Fatalf("startup side effects = %v, want none", calls)
			}
			for _, denied := range []string{
				values[administratorNoticeContentEnvironment],
				values[idempotencyKeyPrefix+"v1"],
				values[idempotencyKeyPrefix+"v2"],
				values[smtpUsernameEnvironment],
				values[smtpPasswordEnvironment],
			} {
				if denied != "" && strings.Contains(err.Error(), denied) {
					t.Fatal("configuration error exposed controlled or secret input")
				}
			}
		})
	}
}

func TestLoadStartupConfig_AdministratorProductionBoundaries(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(map[string]string)
	}{
		{
			name: "minimum notice values and no retained key",
			mutate: func(values map[string]string) {
				values[administratorNoticeVersionEnvironment] = "a"
				setAdministratorNoticeContent(values, "x")
				values[idempotencyRetainedKeyVersionsEnvironment] = "none"
				values[smtpOperationBudgetEnvironment] = "1s"
				values[idempotencyLockDurationEnvironment] = "1s1ns"
				values[idempotencyRecordTTLEnvironment] = "1h"
				values[smtpPortEnvironment] = "1"
			},
		},
		{
			name: "maximum notice key lifecycle and port values",
			mutate: func(values map[string]string) {
				values[administratorNoticeVersionEnvironment] = strings.Repeat("a", 128)
				setAdministratorNoticeContent(values, strings.Repeat("x", 65536))
				values[idempotencyCurrentKeyVersionEnvironment] = strings.Repeat("z", 64)
				values[idempotencyRetainedKeyVersionsEnvironment] = "none"
				values[idempotencyKeyPrefix+strings.Repeat("z", 64)] = strings.Repeat("k", 64)
				values[smtpOperationBudgetEnvironment] = "60s"
				// The SMTP parser's 60s endpoint remains valid; a full tuple must also leave shutdown headroom.
				if _, err := parseRequiredDuration("60s", smtpOperationBudgetEnvironment, time.Second, time.Minute); err != nil {
					t.Fatal(err)
				}
				values[smtpOperationBudgetEnvironment] = "59s"
				values[shutdownBudgetEnvironment] = "60s"
				values[noticeDBOperationBudgetEnvironment] = "1us"
				values[noticeLeaseDurationEnvironment] = "60s"
				values[idempotencyLockDurationEnvironment] = "5m"
				values[idempotencyRecordTTLEnvironment] = "720h"
				values[smtpPortEnvironment] = "65535"
				values[platform.EnvSMTPHost] = "192.0.2.10"
				values[platform.EnvSMTPServerName] = "2001:db8::10"
			},
		},
		{
			name: "sixteen retained versions and plain auth",
			mutate: func(values map[string]string) {
				versions := make([]string, 16)
				for index := range versions {
					version := fmt.Sprintf("k%02d", index)
					versions[index] = version
					values[idempotencyKeyPrefix+version] = fmt.Sprintf("%032d", index+1)
				}
				values[idempotencyCurrentKeyVersionEnvironment] = "z_current"
				values[idempotencyRetainedKeyVersionsEnvironment] = strings.Join(versions, ",")
				values[idempotencyKeyPrefix+"z_current"] = strings.Repeat("z", 32)
				values[smtpAuthModeEnvironment] = "plain"
				values[smtpUsernameEnvironment] = "smtp-user"
				values[smtpPasswordEnvironment] = " smtp password remains exact "
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := validCompositionEnvironment()
			test.mutate(values)
			config, err := loadStartupConfig(
				func(key string) string { return values[key] },
				mapEnvironmentLookup(values),
			)
			if err != nil {
				t.Fatalf("loadStartupConfig() error = %v", err)
			}
			clearStartupSecrets(&config)
		})
	}
}

func TestRun_SMTPCAPreparationFailureHasNoDatabaseOrRouteSideEffects(t *testing.T) {
	badBundle := filepath.Join(t.TempDir(), "bad-ca.pem")
	if err := os.WriteFile(badBundle, []byte("not a certificate"), 0o600); err != nil {
		t.Fatalf("write bad CA bundle: %v", err)
	}
	tests := []struct {
		name string
		path string
	}{
		{name: "missing bundle", path: filepath.Join(t.TempDir(), "missing-ca.pem")},
		{name: "invalid bundle", path: badBundle},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := validCompositionEnvironment()
			values[platform.EnvSMTPCAPath] = test.path
			dependencies := validCompositionDependencies(values)
			var calls []string
			var startupKey []byte
			dependencies.prepareBusiness = func(config startupConfig) (*preparedLoginDependencies, error) {
				calls = append(calls, "prepare")
				startupKey = config.idempotencyKeys.Keys["v2"]
				return prepareProductionLoginDependencies(
					config,
					bytes.NewReader(bytes.Repeat([]byte{0x31}, 128)),
				)
			}
			dependencies.parseDatabase = func(string) (*pgxpool.Config, error) {
				calls = append(calls, "parse")
				return nil, errors.New("must not parse")
			}
			dependencies.connect = func(context.Context, *pgxpool.Config) (*pgxpool.Pool, error) {
				calls = append(calls, "connect")
				return nil, errors.New("must not connect")
			}
			dependencies.composeBusiness = func(
				*pgxpool.Pool,
				businessComposition,
			) (http.Handler, error) {
				calls = append(calls, "factory")
				return nil, errors.New("must not compose")
			}
			dependencies.listen = func(string, string) (net.Listener, error) {
				calls = append(calls, "listen")
				return nil, errors.New("must not listen")
			}

			err := run(context.Background(), dependencies)
			if err == nil {
				t.Fatal("run() error = nil")
			}
			if !reflect.DeepEqual(calls, []string{"prepare"}) {
				t.Fatalf("calls = %v, want prepare only", calls)
			}
			assertClearedSecret(t, startupKey, values[idempotencyKeyPrefix+"v2"])
			if strings.Contains(err.Error(), test.path) {
				t.Fatal("startup error exposed controlled CA path")
			}
		})
	}
}

func TestLoadStartupConfig_IdentityKeysAreRequiredWithoutDefaults(t *testing.T) {
	required := []string{
		loginMaxBodyBytesEnvironment,
		argon2MemoryEnvironment,
		argon2IterationsEnvironment,
		argon2ParallelismEnvironment,
		argon2SaltBytesEnvironment,
		argon2KeyBytesEnvironment,
		argon2ProfileIDEnvironment,
		argon2MaxConcurrencyEnvironment,
		loginBucketKeyVersionEnvironment,
		loginBucketKeyPrefix + "v1",
		loginAbuseWindowEnvironment,
		loginAbuseThresholdEnvironment,
		loginAbuseBlockEnvironment,
		loginAbuseActionEnvironment,
		loginAbuseDecayEnvironment,
		sessionTokenEntropyEnvironment,
		csrfTokenEntropyEnvironment,
		sessionAbsoluteTTLEnvironment,
		sessionIdleTTLEnvironment,
		sourceKeyVersionEnvironment,
		sourceKeyPrefix + "v1",
		platform.EnvTrustedProxyCIDRs,
		recoveryModeEnvironment,
	}
	if len(required) != 23 {
		t.Fatalf("required identity keys = %d, want 23", len(required))
	}
	for _, key := range required {
		t.Run(key, func(t *testing.T) {
			values := validCompositionEnvironment()
			delete(values, key)
			_, err := loadStartupConfig(
				func(name string) string { return values[name] },
				mapEnvironmentLookup(values),
			)
			if err == nil {
				t.Fatal("loadStartupConfig() error = nil")
			}
			for _, secret := range []string{
				"0123456789abcdef0123456789abcdef",
				"abcdef0123456789abcdef0123456789",
			} {
				if strings.Contains(err.Error(), secret) {
					t.Fatal("configuration error exposed secret material")
				}
			}
		})
	}
}

func TestLoadStartupConfig_IdentityCrossConstraintsFailClosed(t *testing.T) {
	tests := []struct {
		name      string
		overrides map[string]string
	}{
		{
			name: "argon memory concurrency product",
			overrides: map[string]string{
				argon2MemoryEnvironment:         "262144",
				argon2MaxConcurrencyEnvironment: "3",
			},
		},
		{
			name: "decay must be below threshold",
			overrides: map[string]string{
				loginAbuseThresholdEnvironment: "5",
				loginAbuseDecayEnvironment:     "5",
			},
		},
		{
			name: "idle cannot exceed absolute",
			overrides: map[string]string{
				sessionAbsoluteTTLEnvironment: "15m",
				sessionIdleTTLEnvironment:     "30m",
			},
		},
		{
			name: "source and bucket keys are independent",
			overrides: map[string]string{
				sourceKeyPrefix + "v1": "0123456789abcdef0123456789abcdef",
			},
		},
		{
			name: "trusted proxy cannot trust all",
			overrides: map[string]string{
				platform.EnvTrustedProxyCIDRs: "0.0.0.0/0",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := validCompositionEnvironment()
			for key, value := range test.overrides {
				values[key] = value
			}
			if _, err := loadStartupConfig(
				func(name string) string { return values[name] },
				mapEnvironmentLookup(values),
			); err == nil {
				t.Fatal("loadStartupConfig() error = nil")
			}
		})
	}
}

func TestLoadStartupConfig_NoTrustedProxyIsExplicitNone(t *testing.T) {
	values := validCompositionEnvironment()
	values[platform.EnvTrustedProxyCIDRs] = "none"
	config, err := loadStartupConfig(
		func(name string) string { return values[name] },
		mapEnvironmentLookup(values),
	)
	if err != nil {
		t.Fatalf("loadStartupConfig() error = %v", err)
	}
	if len(config.deployment.TrustedProxyCIDRs) != 0 {
		t.Fatalf("trusted proxies = %v, want none", config.deployment.TrustedProxyCIDRs)
	}
}

func TestLoadIdentityStartupConfig_ClearsMaterializedSecretsOnEveryLateFailure(t *testing.T) {
	const (
		bucketCanary = "bucket-canary-0123456789abcdef01"
		sourceCanary = "source-canary-0123456789abcdef01"
	)
	tests := []struct {
		name       string
		overrides  map[string]string
		wantSource bool
	}{
		{name: "abuse window", overrides: map[string]string{loginAbuseWindowEnvironment: "invalid"}},
		{name: "abuse threshold", overrides: map[string]string{loginAbuseThresholdEnvironment: "invalid"}},
		{name: "abuse block", overrides: map[string]string{loginAbuseBlockEnvironment: "invalid"}},
		{name: "abuse action", overrides: map[string]string{loginAbuseActionEnvironment: "invalid"}},
		{name: "abuse decay", overrides: map[string]string{loginAbuseDecayEnvironment: "invalid"}},
		{name: "session token entropy", overrides: map[string]string{sessionTokenEntropyEnvironment: "invalid"}},
		{name: "csrf token entropy", overrides: map[string]string{csrfTokenEntropyEnvironment: "invalid"}},
		{name: "absolute ttl", overrides: map[string]string{sessionAbsoluteTTLEnvironment: "invalid"}},
		{name: "idle ttl", overrides: map[string]string{sessionIdleTTLEnvironment: "invalid"}},
		{name: "source key version", overrides: map[string]string{sourceKeyVersionEnvironment: " invalid"}},
		{name: "source key", overrides: map[string]string{sourceKeyPrefix + "v1": "short"}},
		{
			name:       "key independence",
			overrides:  map[string]string{sourceKeyPrefix + "v1": bucketCanary},
			wantSource: true,
		},
		{
			name:       "trusted proxy",
			overrides:  map[string]string{platform.EnvTrustedProxyCIDRs: " 192.0.2.0/24"},
			wantSource: true,
		},
		{
			name:       "recovery mode",
			overrides:  map[string]string{recoveryModeEnvironment: "invalid"},
			wantSource: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := validCompositionEnvironment()
			values[loginBucketKeyPrefix+"v1"] = bucketCanary
			values[sourceKeyPrefix+"v1"] = sourceCanary
			for key, value := range test.overrides {
				values[key] = value
			}
			var config startupConfig
			err := loadIdentityStartupConfig(&config, func(key string) string {
				return values[key]
			})
			if err == nil {
				t.Fatal("loadIdentityStartupConfig() error = nil")
			}
			if strings.Contains(err.Error(), bucketCanary) ||
				strings.Contains(err.Error(), sourceCanary) {
				t.Fatal("configuration error exposed secret material")
			}
			assertClearedSecret(t, config.loginBucketKey, bucketCanary)
			if test.wantSource {
				assertClearedSecret(t, config.sourceKey, sourceCanary)
			}
		})
	}
}

func TestRun_ComposedClearsStartupSecretsAfterSuccessfulPreparation(t *testing.T) {
	values := validCompositionEnvironment()
	var retainedBucket, retainedSource, retainedIdempotency []byte
	dependencies := validCompositionDependencies(values)
	dependencies.prepareBusiness = func(config startupConfig) (*preparedLoginDependencies, error) {
		retainedBucket = config.loginBucketKey
		retainedSource = config.sourceKey
		retainedIdempotency = config.idempotencyKeys.Keys["v2"]
		return nil, errors.New("synthetic preparation stop")
	}

	if err := run(context.Background(), dependencies); err == nil {
		t.Fatal("run() error = nil")
	}
	assertClearedSecret(t, retainedBucket, values[loginBucketKeyPrefix+"v1"])
	assertClearedSecret(t, retainedSource, values[sourceKeyPrefix+"v1"])
	assertClearedSecret(t, retainedIdempotency, values[idempotencyKeyPrefix+"v2"])
}

func TestRun_ComposedClearsPreparedAdministratorSecretsAfterLifecycle(t *testing.T) {
	values := validCompositionEnvironment()
	dependencies := validCompositionDependencies(values)
	var prepared *preparedLoginDependencies
	var startupKey, preparedKey []byte
	dependencies.prepareBusiness = func(config startupConfig) (*preparedLoginDependencies, error) {
		startupKey = config.idempotencyKeys.Keys["v2"]
		var err error
		prepared, err = prepareProductionLoginDependencies(
			config,
			bytes.NewReader(bytes.Repeat([]byte{0x42}, 128)),
		)
		if prepared != nil {
			preparedKey = prepared.idempotencyKeys.Keys["v2"]
		}
		return prepared, err
	}
	dependencies.composeBusiness = func(
		*pgxpool.Pool,
		businessComposition,
	) (http.Handler, error) {
		return http.NotFoundHandler(), nil
	}
	stopErr := errors.New("synthetic lifecycle stop")
	dependencies.coordinate = func(context.Context, lifecycleConfig) error {
		return stopErr
	}

	err := run(context.Background(), dependencies)
	if !errors.Is(err, stopErr) {
		t.Fatalf("run() error = %v, want lifecycle stop", err)
	}
	assertClearedSecret(t, startupKey, values[idempotencyKeyPrefix+"v2"])
	assertClearedSecret(t, preparedKey, values[idempotencyKeyPrefix+"v2"])
	if prepared == nil || prepared.administratorMailer != nil ||
		prepared.administratorNotices != nil ||
		prepared.idempotencyKeys.Keys != nil {
		t.Fatal("prepared administrator dependencies were not released")
	}
}

func TestRun_AWIRELIFE04ComposedNilBusinessHandlerFailsBeforeListener(t *testing.T) {
	values := validCompositionEnvironment()
	dependencies := validCompositionDependencies(values)
	poolCloses, listenCalls := 0, 0
	dependencies.closePool = func(*pgxpool.Pool) { poolCloses++ }
	dependencies.composeBusiness = func(
		*pgxpool.Pool,
		businessComposition,
	) (http.Handler, error) {
		return nil, nil
	}
	dependencies.listen = func(string, string) (net.Listener, error) {
		listenCalls++
		return &recordingListener{}, nil
	}

	if err := run(context.Background(), dependencies); err == nil {
		t.Fatal("run() error = nil")
	}
	if poolCloses != 1 || listenCalls != 0 {
		t.Fatalf("pool closes/listen calls = %d/%d, want 1/0", poolCloses, listenCalls)
	}
}

func TestParseTrustedProxyConfiguration_StrictCanonicalBoundaries(t *testing.T) {
	sixtyFour := make([]string, 64)
	for index := range sixtyFour {
		sixtyFour[index] = netip.PrefixFrom(
			netip.AddrFrom4([4]byte{10, byte(index), 0, 0}),
			16,
		).String()
	}
	tests := []struct {
		name    string
		raw     string
		want    []netip.Prefix
		wantErr bool
	}{
		{name: "explicit none", raw: "none"},
		{name: "outer leading space", raw: " 192.0.2.0/24", wantErr: true},
		{name: "outer trailing space", raw: "192.0.2.0/24 ", wantErr: true},
		{name: "spaced none", raw: " none ", wantErr: true},
		{name: "item leading space", raw: "192.0.2.0/24, 198.51.100.0/24", wantErr: true},
		{name: "item trailing space", raw: "192.0.2.0/24 ,198.51.100.0/24", wantErr: true},
		{name: "ipv4 trust all", raw: "0.0.0.0/0", wantErr: true},
		{name: "ipv6 trust all", raw: "::/0", wantErr: true},
		{name: "mapped trust all", raw: "::ffff:0.0.0.0/96", wantErr: true},
		{
			name: "mapped canonical boundary",
			raw:  "::ffff:192.0.2.0/120",
			want: []netip.Prefix{netip.MustParsePrefix("192.0.2.0/24")},
		},
		{
			name:    "mapped duplicate equivalence",
			raw:     "192.0.2.0/24,::ffff:192.0.2.0/120",
			wantErr: true,
		},
		{name: "host bits", raw: "192.0.2.1/24", wantErr: true},
		{name: "one item", raw: "192.0.2.0/24", want: []netip.Prefix{netip.MustParsePrefix("192.0.2.0/24")}},
		{name: "sixty four items", raw: strings.Join(sixtyFour, ","), want: mustParsePrefixes(t, sixtyFour)},
		{name: "sixty five items", raw: strings.Join(append(sixtyFour, "192.0.2.0/24"), ","), wantErr: true},
		{name: "over 4096 bytes", raw: strings.Repeat("x", 4097), wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseTrustedProxyConfiguration(test.raw)
			if test.wantErr {
				if err == nil {
					t.Fatalf("parseTrustedProxyConfiguration(%q) error = nil", test.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseTrustedProxyConfiguration() error = %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("prefixes = %v, want %v", got, test.want)
			}
		})
	}
}

func TestRun_NonCanonicalTrustedProxyHasZeroSideEffects(t *testing.T) {
	values := validCompositionEnvironment()
	values[platform.EnvTrustedProxyCIDRs] = "192.0.2.0/24 "
	var calls []string
	dependencies := validCompositionDependencies(values)
	dependencies.prepareBusiness = func(startupConfig) (*preparedLoginDependencies, error) {
		calls = append(calls, "prepare")
		return nil, nil
	}
	dependencies.parseDatabase = func(string) (*pgxpool.Config, error) {
		calls = append(calls, "parse")
		return new(pgxpool.Config), nil
	}
	dependencies.connect = func(context.Context, *pgxpool.Config) (*pgxpool.Pool, error) {
		calls = append(calls, "connect")
		return new(pgxpool.Pool), nil
	}
	dependencies.ping = func(context.Context, *pgxpool.Pool) error {
		calls = append(calls, "ping")
		return nil
	}
	dependencies.migrate = func(context.Context, *pgxpool.Pool) error {
		calls = append(calls, "migrate")
		return nil
	}
	dependencies.composeBusiness = func(*pgxpool.Pool, businessComposition) (http.Handler, error) {
		calls = append(calls, "compose")
		return http.NotFoundHandler(), nil
	}
	dependencies.listen = func(string, string) (net.Listener, error) {
		calls = append(calls, "listen")
		return &recordingListener{}, nil
	}

	if err := run(context.Background(), dependencies); err == nil {
		t.Fatal("run() error = nil")
	}
	if len(calls) != 0 {
		t.Fatalf("startup side effects = %v, want none", calls)
	}
}

func TestLoginBusinessRouter_JSONFallbackPrecedesLogin(t *testing.T) {
	loginCalls := 0
	handler := newLoginBusinessRouter(http.HandlerFunc(func(
		response http.ResponseWriter,
		request *http.Request,
	) {
		loginCalls++
		response.WriteHeader(http.StatusNoContent)
	}))
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantCode   platform.Code
		wantAllow  string
		wantCalls  int
	}{
		{"unknown path", http.MethodPost, "/unknown", http.StatusNotFound, platform.CodeRequestRouteNotFound, "", 0},
		{"wrong method", http.MethodGet, "/api/v1/sessions", http.StatusMethodNotAllowed, platform.CodeRequestMethodNotAllowed, http.MethodPost, 0},
		{"exact post", http.MethodPost, "/api/v1/sessions", http.StatusNoContent, "", "", 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			loginCalls = 0
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(test.method, test.path, nil))
			if response.Code != test.wantStatus || loginCalls != test.wantCalls {
				t.Fatalf("status/calls = %d/%d, want %d/%d", response.Code, loginCalls, test.wantStatus, test.wantCalls)
			}
			if response.Header().Get("Allow") != test.wantAllow {
				t.Fatalf("Allow = %q, want %q", response.Header().Get("Allow"), test.wantAllow)
			}
			if test.wantCode == "" {
				return
			}
			var envelope platform.Envelope
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decode error envelope: %v", err)
			}
			if envelope.Code != test.wantCode || envelope.TraceID == "" ||
				response.Header().Get(platform.TraceHeader) != envelope.TraceID {
				t.Fatalf("error envelope = %#v", envelope)
			}
		})
	}
}

func TestInitialAdminBootstrapRunsAfterMigrationBeforeCompositionAndListen(t *testing.T) {
	values := validCompositionEnvironment()
	var calls []string
	dependencies := validCompositionDependencies(values)
	dependencies.migrate = func(context.Context, *pgxpool.Pool) error {
		calls = append(calls, "migrate")
		return nil
	}
	dependencies.bootstrapInitialAdmin = func(
		context.Context,
		*pgxpool.Pool,
		*preparedLoginDependencies,
	) error {
		calls = append(calls, "bootstrap")
		return nil
	}
	dependencies.composeBusiness = func(
		*pgxpool.Pool,
		businessComposition,
	) (http.Handler, error) {
		calls = append(calls, "compose")
		return http.NotFoundHandler(), nil
	}
	dependencies.listen = func(string, string) (net.Listener, error) {
		calls = append(calls, "listen")
		return nil, errors.New("stop after ordering evidence")
	}

	if err := run(context.Background(), dependencies); err == nil {
		t.Fatal("run() error = nil")
	}
	if !reflect.DeepEqual(calls, []string{"migrate", "bootstrap", "compose", "listen"}) {
		t.Fatalf("calls = %v", calls)
	}
}

func TestPasswordBusinessRouterUsesExactPutWithoutRegressingLogin(t *testing.T) {
	loginCalls, passwordCalls := 0, 0
	router := newLoginBusinessRouter(
		http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			loginCalls++
			response.WriteHeader(http.StatusNoContent)
		}),
		http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			passwordCalls++
			response.WriteHeader(http.StatusOK)
		}),
	)
	tests := []struct {
		method       string
		path         string
		wantStatus   int
		wantAllow    string
		wantLogin    int
		wantPassword int
	}{
		{http.MethodPost, "/api/v1/sessions", http.StatusNoContent, "", 1, 0},
		{http.MethodGet, "/api/v1/sessions", http.StatusMethodNotAllowed, http.MethodPost, 0, 0},
		{http.MethodPut, "/api/v1/users/me/password", http.StatusOK, "", 0, 1},
		{http.MethodPost, "/api/v1/users/me/password", http.StatusMethodNotAllowed, http.MethodPut, 0, 0},
		{http.MethodPut, "/api/v1/users/me/password/extra", http.StatusNotFound, "", 0, 0},
	}
	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			loginCalls, passwordCalls = 0, 0
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(test.method, test.path, nil))
			if response.Code != test.wantStatus ||
				response.Header().Get("Allow") != test.wantAllow ||
				loginCalls != test.wantLogin ||
				passwordCalls != test.wantPassword {
				t.Fatalf(
					"status/Allow/login/password = %d/%q/%d/%d",
					response.Code,
					response.Header().Get("Allow"),
					loginCalls,
					passwordCalls,
				)
			}
			if response.Header().Get(platform.TraceHeader) == "" {
				t.Fatal("trace header is empty")
			}
		})
	}
}

func TestAdministratorBusinessRouterUsesExactMethods(t *testing.T) {
	calls := map[string]int{}
	handler := func(name string) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			calls[name]++
			response.WriteHeader(http.StatusNoContent)
		})
	}
	router := newLoginBusinessRouter(
		handler("login"),
		handler("password"),
		handler("notice"),
		handler("create"),
		handler("privacy"),
	)
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantAllow  string
		wantCall   string
	}{
		{
			name:       "current notice",
			method:     http.MethodGet,
			path:       "/api/v1/admin/administrator-information-provision-notices/current",
			wantStatus: http.StatusNoContent,
			wantCall:   "notice",
		},
		{
			name:       "current notice wrong method",
			method:     http.MethodPost,
			path:       "/api/v1/admin/administrator-information-provision-notices/current",
			wantStatus: http.StatusMethodNotAllowed,
			wantAllow:  http.MethodGet,
		},
		{
			name:       "create administrator",
			method:     http.MethodPost,
			path:       "/api/v1/admin/administrators",
			wantStatus: http.StatusNoContent,
			wantCall:   "create",
		},
		{
			name:       "create administrator wrong method",
			method:     http.MethodGet,
			path:       "/api/v1/admin/administrators",
			wantStatus: http.StatusMethodNotAllowed,
			wantAllow:  http.MethodPost,
		},
		{
			name:       "acknowledge privacy",
			method:     http.MethodPost,
			path:       "/api/v1/users/me/privacy-acknowledgements",
			wantStatus: http.StatusNoContent,
			wantCall:   "privacy",
		},
		{
			name:       "acknowledge privacy wrong method",
			method:     http.MethodGet,
			path:       "/api/v1/users/me/privacy-acknowledgements",
			wantStatus: http.StatusMethodNotAllowed,
			wantAllow:  http.MethodPost,
		},
		{
			name:       "unknown path",
			method:     http.MethodPost,
			path:       "/api/v1/admin/administrators/extra",
			wantStatus: http.StatusNotFound,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clear(calls)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(test.method, test.path, nil))
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if response.Header().Get("Allow") != test.wantAllow {
				t.Fatalf("Allow = %q, want %q", response.Header().Get("Allow"), test.wantAllow)
			}
			if test.wantCall != "" && calls[test.wantCall] != 1 {
				t.Fatalf("%s calls = %d, want 1", test.wantCall, calls[test.wantCall])
			}
			for name, count := range calls {
				if name != test.wantCall && count != 0 {
					t.Fatalf("%s calls = %d, want 0", name, count)
				}
			}
			if response.Header().Get(platform.TraceHeader) == "" {
				t.Fatal("trace header is empty")
			}
		})
	}
}

func TestArticleBusinessRouter_AWIRERTR01RTR02RTR04UsesExactDeletePaths(t *testing.T) {
	calls := map[string]int{}
	handler := func(name string) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			calls[name]++
			response.WriteHeader(http.StatusNoContent)
		})
	}
	router := newLoginBusinessRouter(
		handler("login"),
		handler("password"),
		handler("notice"),
		handler("administrator"),
		handler("privacy"),
		handler("ordinary"),
		handler("violating"),
		handler("orphan"),
		nil,
		nil,
		handler("list"),
		handler("detail"),
		nil, nil, nil, nil, nil, nil,
		handler("create"), handler("author"), handler("update"),
	)
	type routeTest struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantAllow  string
		wantCall   string
	}
	tests := []routeTest{
		{"member list", http.MethodGet, "/api/v1/articles", http.StatusNoContent, "", "list"},
		{"member detail", http.MethodGet, "/api/v1/articles/article-1", http.StatusNoContent, "", "detail"},
		{"create", http.MethodPost, "/api/v1/articles", http.StatusNoContent, "", "create"},
		{"author detail", http.MethodGet, "/api/v1/users/me/articles/article-1", http.StatusNoContent, "", "author"},
		{"update", http.MethodPut, "/api/v1/articles/article-1", http.StatusNoContent, "", "update"},
		{"ordinary", http.MethodDelete, "/api/v1/articles/article-1", http.StatusNoContent, "", "ordinary"},
		{"violating", http.MethodDelete, "/api/v1/admin/articles/article-1", http.StatusNoContent, "", "violating"},
		{"orphan", http.MethodDelete, "/api/v1/admin/orphan-articles/article-1", http.StatusNoContent, "", "orphan"},
	}
	routes := []struct {
		name string
		path string
	}{
		{name: "ordinary", path: "/api/v1/articles/article-1"},
		{name: "violating", path: "/api/v1/admin/articles/article-1"},
		{name: "orphan", path: "/api/v1/admin/orphan-articles/article-1"},
	}
	for _, route := range routes {
		for _, method := range []string{
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
		} {
			if route.name == "ordinary" && method == http.MethodPut {
				continue // PUT is covered as a supported method above.
			}
			allow := http.MethodDelete
			if route.name == "ordinary" || route.name == "orphan" || route.name == "violating" {
				allow = http.MethodGet + ", " + http.MethodDelete
			}
			if route.name == "ordinary" {
				allow = http.MethodGet + ", " + http.MethodPut + ", " + http.MethodDelete
			}
			tests = append(tests, routeTest{
				name: route.name + " " + method, method: method, path: route.path,
				wantStatus: http.StatusMethodNotAllowed, wantAllow: allow,
			})
		}
	}
	tests = append(tests,
		routeTest{"list wrong method", http.MethodPatch, "/api/v1/articles", http.StatusMethodNotAllowed, http.MethodGet + ", " + http.MethodPost, ""},
		routeTest{"missing ordinary id", http.MethodDelete, "/api/v1/articles/", http.StatusNotFound, "", ""},
		routeTest{"extra ordinary segment", http.MethodDelete, "/api/v1/articles/article-1/extra", http.StatusNotFound, "", ""},
		routeTest{"missing violating id", http.MethodDelete, "/api/v1/admin/articles/", http.StatusNotFound, "", ""},
		routeTest{"extra violating segment", http.MethodDelete, "/api/v1/admin/articles/article-1/extra", http.StatusNotFound, "", ""},
		routeTest{"missing orphan id", http.MethodDelete, "/api/v1/admin/orphan-articles/", http.StatusNotFound, "", ""},
		routeTest{"extra orphan segment", http.MethodDelete, "/api/v1/admin/orphan-articles/article-1/extra", http.StatusNotFound, "", ""},
	)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clear(calls)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(test.method, test.path, nil))
			if response.Code != test.wantStatus ||
				response.Header().Get("Allow") != test.wantAllow {
				t.Fatalf(
					"status/Allow = %d/%q, want %d/%q",
					response.Code,
					response.Header().Get("Allow"),
					test.wantStatus,
					test.wantAllow,
				)
			}
			for name, count := range calls {
				want := 0
				if name == test.wantCall {
					want = 1
				}
				if count != want {
					t.Fatalf("%s calls = %d, want %d", name, count, want)
				}
			}
			if response.Header().Get(platform.TraceHeader) == "" {
				t.Fatal("trace header is empty")
			}
			if test.wantStatus != http.StatusNoContent {
				if got := response.Header().Get("Content-Type"); !strings.HasPrefix(
					got,
					"application/json",
				) {
					t.Fatalf("Content-Type = %q, want application/json", got)
				}
				if got := response.Header().Get("X-Content-Type-Options"); got != "nosniff" {
					t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
				}
				var envelope platform.Envelope
				if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
					t.Fatalf("decode error envelope: %v", err)
				}
				if envelope.TraceID == "" {
					t.Fatal("error envelope trace_id is empty")
				}
			}
		})
	}
}

func TestProductionBusinessHandler_AWIRERTR03RTR05ArticleSecurityChain(t *testing.T) {
	config := mustLoadStartupConfig(t, validCompositionEnvironment())
	prepared, err := prepareProductionLoginDependencies(
		config,
		bytes.NewReader(bytes.Repeat([]byte{0x5a}, 128)),
	)
	clearStartupSecrets(&config)
	if err != nil {
		t.Fatalf("prepareProductionLoginDependencies() error = %v", err)
	}
	t.Cleanup(prepared.cleanup)
	composition := productionBusinessComposition(t, prepared)
	composition.articleJournal = mainArticleJournalStub{}
	composition.articleDeletion = &articleDeletionStartupConfig{
		workerID: "article-worker-test",
		lease:    30 * time.Second,
		worker: article.DeletionWorkerConfig{
			PollInterval:      time.Second,
			BatchSize:         10,
			OperationBudget:   10 * time.Second,
			BackoffInitial:    250 * time.Millisecond,
			BackoffMax:        time.Second,
			MaxBatchesPerWake: 3,
			ShutdownBudget:    time.Second,
		},
	}
	composition.runtime = &businessRuntime{}
	handler, err := newProductionBusinessHandler(new(pgxpool.Pool), composition)
	if err != nil {
		t.Fatalf("newProductionBusinessHandler() error = %v", err)
	}
	if composition.runtime.articleWorker == nil {
		t.Fatal("Article deletion worker was not composed")
	}

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantAllow  string
		wantCode   platform.Code
	}{
		{
			name: "wrong method before security", method: http.MethodPost,
			path:       "/api/v1/articles/10000000-0000-4000-8000-000000000001",
			wantStatus: http.StatusMethodNotAllowed,
			wantAllow:  http.MethodGet + ", " + http.MethodPut + ", " + http.MethodDelete,
			wantCode:   platform.CodeRequestMethodNotAllowed,
		},
		{
			name: "correct method reaches origin first", method: http.MethodDelete,
			path:       "/api/v1/articles/10000000-0000-4000-8000-000000000001",
			wantStatus: http.StatusForbidden, wantCode: platform.CodeAuthOriginRejected,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(
				response,
				httptest.NewRequest(test.method, test.path, nil),
			)
			if response.Code != test.wantStatus ||
				response.Header().Get("Allow") != test.wantAllow {
				t.Fatalf(
					"status/Allow = %d/%q, want %d/%q",
					response.Code,
					response.Header().Get("Allow"),
					test.wantStatus,
					test.wantAllow,
				)
			}
			var envelope platform.Envelope
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decode error envelope: %v", err)
			}
			if envelope.Code != test.wantCode || envelope.TraceID == "" ||
				response.Header().Get(platform.TraceHeader) != envelope.TraceID {
				t.Fatalf("error envelope = %#v", envelope)
			}
		})
	}
}

func TestProductionBusinessHandler_AWIRERTR03RTR05OrdersAndStopsEverySecurityLayer(
	t *testing.T,
) {
	tests := []struct {
		name       string
		rejectAt   string
		principal  session.Principal
		wantStatus int
		wantEvents []string
	}{
		{
			name:     "origin rejection stops session",
			rejectAt: "origin", wantStatus: http.StatusForbidden,
			wantEvents: []string{"trace", "origin"},
		},
		{
			name:     "session rejection stops csrf",
			rejectAt: "session", wantStatus: http.StatusUnauthorized,
			wantEvents: []string{"trace", "origin", "session"},
		},
		{
			name:     "csrf rejection stops principal",
			rejectAt: "csrf", wantStatus: http.StatusForbidden,
			wantEvents: []string{"trace", "origin", "session", "csrf"},
		},
		{
			name: "principal authorization rejection stops handler",
			principal: session.Principal{
				SessionID: "session-1", UserID: "member-1",
				Role: "member", Status: "disabled",
			},
			wantStatus: http.StatusForbidden,
			wantEvents: []string{
				"trace", "origin", "session", "csrf", "principal",
			},
		},
		{
			name: "accepted request reaches handler in order",
			principal: session.Principal{
				SessionID: "session-1", UserID: "member-1",
				Role: "member", Status: "active",
			},
			wantStatus: http.StatusNoContent,
			wantEvents: []string{
				"trace", "origin", "session", "csrf", "principal",
				"authorization", "handler",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var events []string
			resolver := &articleSecurityResolverSpy{
				events: &events, rejectAt: test.rejectAt, principal: test.principal,
			}
			seam := &articleProductionTestSeam{
				principalResolver: resolver,
				trace:             func(*http.Request) { events = append(events, "trace") },
				protectOrigin: func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(
						response http.ResponseWriter,
						request *http.Request,
					) {
						events = append(events, "origin")
						if test.rejectAt == "origin" {
							_ = platform.WriteError(
								response,
								request,
								platform.NewPublicError(
									platform.CodeAuthOriginRejected,
									errors.New("synthetic origin rejection"),
								),
							)
							return
						}
						next.ServeHTTP(response, request)
					})
				},
				authorized: func(string) {
					events = append(events, "authorization")
				},
				handlers: [3]http.Handler{
					http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
						events = append(events, "handler")
						response.WriteHeader(http.StatusNoContent)
					}),
				},
			}
			handler := newArticleProductionHandlerWithTestSeam(t, seam)
			request := httptest.NewRequest(
				http.MethodDelete,
				"/api/v1/articles/10000000-0000-4000-8000-000000000001",
				nil,
			)
			request.AddCookie(&http.Cookie{
				Name: platform.SessionCookieName, Value: "session-token",
			})
			request.Header.Set(platform.CSRFHeaderName, "csrf-token")
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if !reflect.DeepEqual(events, test.wantEvents) {
				t.Fatalf("ordered calls = %v, want %v", events, test.wantEvents)
			}
		})
	}
}

func TestProductionBusinessHandler_AWIRERTR05ThreeDeleteRoutesCarryRoleAndStatus(
	t *testing.T,
) {
	tests := []struct {
		name       string
		path       string
		principal  session.Principal
		wantRoute  string
		wantStatus int
	}{
		{
			name: "ordinary active member", path: "/api/v1/articles/10000000-0000-4000-8000-000000000001",
			principal: session.Principal{
				SessionID: "session-1", UserID: "member-1", Role: "member", Status: "active",
			},
			wantRoute: "ordinary", wantStatus: http.StatusNoContent,
		},
		{
			name: "ordinary disabled member", path: "/api/v1/articles/10000000-0000-4000-8000-000000000001",
			principal: session.Principal{
				SessionID: "session-1", UserID: "member-1", Role: "member", Status: "disabled",
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "violating active administrator", path: "/api/v1/admin/articles/10000000-0000-4000-8000-000000000001",
			principal: session.Principal{
				SessionID: "session-1", UserID: "admin-1", Role: "admin", Status: "active",
			},
			wantRoute: "violating", wantStatus: http.StatusNoContent,
		},
		{
			name: "violating restricted administrator", path: "/api/v1/admin/articles/10000000-0000-4000-8000-000000000001",
			principal: session.Principal{
				SessionID: "session-1", UserID: "admin-1", Role: "admin", Status: "active",
				MustChangePassword: true,
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "orphan active administrator", path: "/api/v1/admin/orphan-articles/10000000-0000-4000-8000-000000000001",
			principal: session.Principal{
				SessionID: "session-1", UserID: "admin-1", Role: "admin", Status: "active",
			},
			wantRoute: "orphan", wantStatus: http.StatusNoContent,
		},
		{
			name: "orphan inactive administrator", path: "/api/v1/admin/orphan-articles/10000000-0000-4000-8000-000000000001",
			principal: session.Principal{
				SessionID: "session-1", UserID: "admin-1", Role: "admin", Status: "disabled",
			},
			wantStatus: http.StatusForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var reached string
			seam := &articleProductionTestSeam{
				principalResolver: &articleSecurityResolverSpy{
					principal: test.principal,
				},
				protectOrigin: func(next http.Handler) http.Handler { return next },
				handlers: [3]http.Handler{
					http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
						reached = "ordinary"
						response.WriteHeader(http.StatusNoContent)
					}),
					http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
						reached = "violating"
						response.WriteHeader(http.StatusNoContent)
					}),
					http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
						reached = "orphan"
						response.WriteHeader(http.StatusNoContent)
					}),
				},
			}
			handler := newArticleProductionHandlerWithTestSeam(t, seam)
			request := httptest.NewRequest(http.MethodDelete, test.path, nil)
			request.AddCookie(&http.Cookie{
				Name: platform.SessionCookieName, Value: "session-token",
			})
			request.Header.Set(platform.CSRFHeaderName, "csrf-token")
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != test.wantStatus || reached != test.wantRoute {
				t.Fatalf(
					"status/route = %d/%q, want %d/%q",
					response.Code,
					reached,
					test.wantStatus,
					test.wantRoute,
				)
			}
		})
	}
}

func TestLoadStartupConfig_ArticleDeletionConfiguration_AWIRECFG01To05(t *testing.T) {
	values := validCompositionEnvironment()
	config := mustLoadStartupConfig(t, values)
	defer clearStartupSecrets(&config)

	if config.articleDeletion.journalRoot != values[articleDeletionJournalRootEnvironment] ||
		config.articleDeletion.fileMode != 0o600 ||
		config.articleDeletion.workerID != values[articleFinalizerWorkerIDEnvironment] ||
		config.articleDeletion.worker.BatchSize != 50 ||
		config.articleDeletion.worker.PollInterval != time.Second ||
		config.articleDeletion.worker.OperationBudget != 10*time.Second ||
		len(config.articleDeletion.keyring.Retained) != 1 {
		t.Fatalf("article deletion config = %#v", config.articleDeletion)
	}
	if config.articleDeletion.keyring.Active.ID != "article-deletion-hmac" ||
		config.articleDeletion.keyring.Active.Version != "v2" {
		t.Fatalf("active Article deletion key = %#v", config.articleDeletion.keyring.Active)
	}

	tests := []struct {
		name   string
		mutate func(map[string]string)
	}{
		{"missing retained list", func(values map[string]string) {
			delete(values, articleDeletionRetainedKeyVersionsEnvironment)
		}},
		{"relative root", func(values map[string]string) {
			values[articleDeletionJournalRootEnvironment] = "relative"
		}},
		{"padded key", func(values map[string]string) {
			values[articleDeletionKeyPrefix+"v2"] += "="
		}},
		{"operation budget not below lease", func(values map[string]string) {
			values[articleFinalizerOperationBudgetEnvironment] = "30s"
		}},
		{"poll not below lease", func(values map[string]string) {
			values[articleFinalizerPollIntervalEnvironment] = "30s"
		}},
		{"worker shutdown exceeds global", func(values map[string]string) {
			values[articleFinalizerShutdownBudgetEnvironment] = "11s"
		}},
		{"current retained duplicate", func(values map[string]string) {
			values[articleDeletionRetainedKeyVersionsEnvironment] = "v1,v2"
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := validCompositionEnvironment()
			test.mutate(values)
			if _, err := loadStartupConfig(
				func(key string) string { return values[key] },
				mapEnvironmentLookup(values),
			); err == nil {
				t.Fatal("loadStartupConfig() error = nil")
			}
		})
	}
}

func TestLoadStartupConfig_ArticleDeletionMatrix(t *testing.T) {
	load := func(t *testing.T, values map[string]string) (startupConfig, error) {
		t.Helper()
		return loadStartupConfig(
			func(key string) string { return values[key] },
			mapEnvironmentLookup(values),
		)
	}
	requireRejected := func(
		t *testing.T,
		mutate func(map[string]string),
	) {
		t.Helper()
		values := validCompositionEnvironment()
		mutate(values)
		config, err := load(t, values)
		clearStartupSecrets(&config)
		if err == nil {
			t.Fatal("loadStartupConfig() error = nil")
		}
	}
	requireAccepted := func(
		t *testing.T,
		mutate func(map[string]string),
	) startupConfig {
		t.Helper()
		values := validCompositionEnvironment()
		mutate(values)
		config, err := load(t, values)
		if err != nil {
			t.Fatalf("loadStartupConfig() error = %v", err)
		}
		return config
	}

	t.Run("A-WIRE-CFG-01 required and retained presence", func(t *testing.T) {
		required := []string{
			articleDeletionJournalRootEnvironment,
			articleDeletionCurrentKeyVersionEnvironment,
			articleFinalizerWorkerIDEnvironment,
			articleFinalizerPollIntervalEnvironment,
			articleFinalizerBatchSizeEnvironment,
			articleFinalizerLeaseDurationEnvironment,
			articleFinalizerOperationBudgetEnvironment,
			articleFinalizerBackoffInitialEnvironment,
			articleFinalizerBackoffMaxEnvironment,
			articleFinalizerMaxAttemptsEnvironment,
			articleFinalizerShutdownBudgetEnvironment,
			articleDeletionKeyPrefix + "v2",
		}
		for _, key := range required {
			t.Run(key, func(t *testing.T) {
				requireRejected(t, func(values map[string]string) {
					delete(values, key)
				})
			})
		}
		requireRejected(t, func(values map[string]string) {
			delete(values, articleDeletionRetainedKeyVersionsEnvironment)
		})
		config := requireAccepted(t, func(values map[string]string) {
			values[articleDeletionRetainedKeyVersionsEnvironment] = ""
		})
		defer clearStartupSecrets(&config)
		if len(config.articleDeletion.keyring.Retained) != 0 {
			t.Fatalf(
				"retained keys = %d, want 0",
				len(config.articleDeletion.keyring.Retained),
			)
		}
	})

	t.Run("A-WIRE-CFG-02 empty blank and path inputs", func(t *testing.T) {
		for _, mutation := range []func(map[string]string){
			func(values map[string]string) {
				values[articleDeletionJournalRootEnvironment] = ""
			},
			func(values map[string]string) {
				values[articleDeletionJournalRootEnvironment] = "   "
			},
			func(values map[string]string) {
				values[articleDeletionJournalRootEnvironment] = "relative"
			},
			func(values map[string]string) {
				values[articleDeletionJournalRootEnvironment] =
					values[articleDeletionJournalRootEnvironment] +
						string(filepath.Separator) + ".." +
						string(filepath.Separator) + "journal"
			},
			func(values map[string]string) {
				values[articleFinalizerWorkerIDEnvironment] = " "
			},
		} {
			requireRejected(t, mutation)
		}
	})

	t.Run("A-WIRE-CFG-03 format validation", func(t *testing.T) {
		for _, mutation := range []func(map[string]string){
			func(values map[string]string) {
				values[articleDeletionJournalFileModeEnvironment] = "600"
			},
			func(values map[string]string) {
				values[articleDeletionCurrentKeyVersionEnvironment] = "v\u00e9"
			},
			func(values map[string]string) {
				values[articleDeletionCurrentKeyVersionEnvironment] = strings.Repeat("v", 65)
			},
			func(values map[string]string) {
				values[articleDeletionRetainedKeyVersionsEnvironment] = "v1,,v3"
			},
			func(values map[string]string) {
				values[articleDeletionRetainedKeyVersionsEnvironment] = "v1,v1"
			},
			func(values map[string]string) {
				values[articleDeletionKeyPrefix+"v2"] += "="
			},
			func(values map[string]string) {
				values[articleDeletionKeyPrefix+"v2"] = base64.RawURLEncoding.EncodeToString(
					bytes.Repeat([]byte{0x32}, 31),
				)
			},
			func(values map[string]string) {
				values[articleFinalizerPollIntervalEnvironment] = "invalid"
			},
			func(values map[string]string) {
				values[articleFinalizerBatchSizeEnvironment] = "1.5"
			},
		} {
			requireRejected(t, mutation)
		}
	})

	t.Run("A-WIRE-CFG-04 numeric boundaries", func(t *testing.T) {
		accepted := []func(map[string]string){
			func(values map[string]string) {
				values[articleFinalizerPollIntervalEnvironment] = "100ms"
			},
			func(values map[string]string) {
				values[articleFinalizerPollIntervalEnvironment] = "1m"
				values[articleFinalizerLeaseDurationEnvironment] = "5m"
			},
			func(values map[string]string) {
				values[articleFinalizerBatchSizeEnvironment] = "1"
			},
			func(values map[string]string) {
				values[articleFinalizerBatchSizeEnvironment] = "500"
			},
			func(values map[string]string) {
				values[articleFinalizerLeaseDurationEnvironment] = "5s"
				values[articleFinalizerOperationBudgetEnvironment] = "1s"
				values[articleFinalizerPollIntervalEnvironment] = "100ms"
			},
			func(values map[string]string) {
				values[articleFinalizerLeaseDurationEnvironment] = "5m"
			},
			func(values map[string]string) {
				values[articleFinalizerOperationBudgetEnvironment] = "1s"
			},
			func(values map[string]string) {
				values[articleFinalizerOperationBudgetEnvironment] = "1m"
				values[articleFinalizerLeaseDurationEnvironment] = "5m"
			},
			func(values map[string]string) {
				values[articleFinalizerBackoffInitialEnvironment] = "50ms"
			},
			func(values map[string]string) {
				values[articleFinalizerBackoffInitialEnvironment] = "10s"
			},
			func(values map[string]string) {
				values[articleFinalizerBackoffMaxEnvironment] = "1s"
			},
			func(values map[string]string) {
				values[articleFinalizerBackoffMaxEnvironment] = "5m"
			},
			func(values map[string]string) {
				values[articleFinalizerMaxAttemptsEnvironment] = "1"
			},
			func(values map[string]string) {
				values[articleFinalizerMaxAttemptsEnvironment] = "10"
			},
			func(values map[string]string) {
				values[articleFinalizerShutdownBudgetEnvironment] = "1s"
			},
			func(values map[string]string) {
				values[articleFinalizerShutdownBudgetEnvironment] = "1m"
				values[shutdownBudgetEnvironment] = "1m"
			},
		}
		for _, mutation := range accepted {
			config := requireAccepted(t, mutation)
			clearStartupSecrets(&config)
		}
		rejected := []struct {
			name   string
			mutate func(map[string]string)
		}{
			{"poll below minimum", func(values map[string]string) {
				values[articleFinalizerPollIntervalEnvironment] = "99ms"
			}},
			{"poll above maximum", func(values map[string]string) {
				values[articleFinalizerPollIntervalEnvironment] = "1m1ns"
				values[articleFinalizerLeaseDurationEnvironment] = "5m"
			}},
			{"batch below minimum", func(values map[string]string) {
				values[articleFinalizerBatchSizeEnvironment] = "0"
			}},
			{"batch above maximum", func(values map[string]string) {
				values[articleFinalizerBatchSizeEnvironment] = "501"
			}},
			{"lease below minimum", func(values map[string]string) {
				values[articleFinalizerLeaseDurationEnvironment] = "4999ms"
			}},
			{"lease above maximum", func(values map[string]string) {
				values[articleFinalizerLeaseDurationEnvironment] = "301s"
			}},
			{"operation budget below minimum", func(values map[string]string) {
				values[articleFinalizerOperationBudgetEnvironment] = "999ms"
			}},
			{"operation budget above maximum", func(values map[string]string) {
				values[articleFinalizerOperationBudgetEnvironment] = "1m1ns"
				values[articleFinalizerLeaseDurationEnvironment] = "5m"
			}},
			{"initial backoff below minimum", func(values map[string]string) {
				values[articleFinalizerBackoffInitialEnvironment] = "49ms"
			}},
			{"initial backoff above maximum", func(values map[string]string) {
				values[articleFinalizerBackoffInitialEnvironment] = "10s1ns"
				values[articleFinalizerBackoffMaxEnvironment] = "30s"
			}},
			{"maximum backoff below minimum", func(values map[string]string) {
				values[articleFinalizerBackoffMaxEnvironment] = "999ms"
			}},
			{"maximum backoff above maximum", func(values map[string]string) {
				values[articleFinalizerBackoffMaxEnvironment] = "301s"
			}},
			{"attempts below minimum", func(values map[string]string) {
				values[articleFinalizerMaxAttemptsEnvironment] = "0"
			}},
			{"attempts above maximum", func(values map[string]string) {
				values[articleFinalizerMaxAttemptsEnvironment] = "11"
			}},
			{"shutdown below minimum", func(values map[string]string) {
				values[articleFinalizerShutdownBudgetEnvironment] = "999ms"
			}},
			{"shutdown above maximum", func(values map[string]string) {
				values[articleFinalizerShutdownBudgetEnvironment] = "1m1ns"
				values[shutdownBudgetEnvironment] = "2m"
			}},
		}
		for _, test := range rejected {
			t.Run(test.name, func(t *testing.T) {
				requireRejected(t, test.mutate)
			})
		}
	})

	t.Run("A-WIRE-CFG-05 cross constraints", func(t *testing.T) {
		for _, mutation := range []func(map[string]string){
			func(values map[string]string) {
				values[articleFinalizerBackoffInitialEnvironment] = "2s"
				values[articleFinalizerBackoffMaxEnvironment] = "1s"
			},
			func(values map[string]string) {
				values[articleFinalizerOperationBudgetEnvironment] = "30s"
			},
			func(values map[string]string) {
				values[articleFinalizerPollIntervalEnvironment] = "30s"
			},
			func(values map[string]string) {
				values[articleFinalizerShutdownBudgetEnvironment] = "11s"
			},
			func(values map[string]string) {
				values[articleDeletionRetainedKeyVersionsEnvironment] = "v1,v2"
			},
			func(values map[string]string) {
				values[articleDeletionRetainedKeyVersionsEnvironment] = "v1,v3"
			},
		} {
			requireRejected(t, mutation)
		}
	})
}

func TestLoadStartupConfig_ArticleDeletion_AWIRECFG06HasZeroSideEffects(t *testing.T) {
	keyCanary := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x6b}, 32))
	canaries := []string{
		"database-dsn-canary.invalid",
		"article-root-canary",
		keyCanary,
		"key-material-canary",
		"private-body-canary",
		"pii-canary@example.invalid",
	}
	tests := []struct {
		name   string
		mutate func(map[string]string)
	}{
		{"root", func(values map[string]string) {
			values[articleDeletionJournalRootEnvironment] = "relative-article-root-canary"
		}},
		{"file mode", func(values map[string]string) {
			values[articleDeletionJournalFileModeEnvironment] = "0644"
		}},
		{"current key version", func(values map[string]string) {
			values[articleDeletionCurrentKeyVersionEnvironment] = " bad"
		}},
		{"retained keys missing", func(values map[string]string) {
			delete(values, articleDeletionRetainedKeyVersionsEnvironment)
		}},
		{"retained keys invalid", func(values map[string]string) {
			values[articleDeletionRetainedKeyVersionsEnvironment] = "v1,v1"
		}},
		{"current key", func(values map[string]string) {
			values[articleDeletionKeyPrefix+"v2"] = "key-material-canary"
		}},
		{"retained key", func(values map[string]string) {
			values[articleDeletionKeyPrefix+"v1"] = "key-material-canary"
		}},
		{"worker ID", func(values map[string]string) {
			values[articleFinalizerWorkerIDEnvironment] = " "
		}},
		{"poll interval", func(values map[string]string) {
			values[articleFinalizerPollIntervalEnvironment] = "99ms"
		}},
		{"batch size", func(values map[string]string) {
			values[articleFinalizerBatchSizeEnvironment] = "0"
		}},
		{"lease duration", func(values map[string]string) {
			values[articleFinalizerLeaseDurationEnvironment] = "4999ms"
		}},
		{"operation budget", func(values map[string]string) {
			values[articleFinalizerOperationBudgetEnvironment] = "1m1ns"
			values[articleFinalizerLeaseDurationEnvironment] = "5m"
		}},
		{"initial backoff", func(values map[string]string) {
			values[articleFinalizerBackoffInitialEnvironment] = "10s1ns"
		}},
		{"maximum backoff", func(values map[string]string) {
			values[articleFinalizerBackoffMaxEnvironment] = "999ms"
		}},
		{"maximum attempts", func(values map[string]string) {
			values[articleFinalizerMaxAttemptsEnvironment] = "0"
		}},
		{"shutdown budget", func(values map[string]string) {
			values[articleFinalizerShutdownBudgetEnvironment] = "1m1ns"
			values[shutdownBudgetEnvironment] = "2m"
		}},
		{"backoff order", func(values map[string]string) {
			values[articleFinalizerBackoffInitialEnvironment] = "2s"
			values[articleFinalizerBackoffMaxEnvironment] = "1s"
		}},
		{"operation versus lease", func(values map[string]string) {
			values[articleFinalizerOperationBudgetEnvironment] = "30s"
		}},
		{"poll versus lease", func(values map[string]string) {
			values[articleFinalizerPollIntervalEnvironment] = "30s"
		}},
		{"worker versus global shutdown", func(values map[string]string) {
			values[articleFinalizerShutdownBudgetEnvironment] = "11s"
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := validCompositionEnvironment()
			values[databaseDSNEnvironment] =
				"host=" + canaries[0] + " user=han dbname=han"
			values[articleDeletionJournalRootEnvironment] = filepath.Join(
				string(filepath.Separator),
				"sensitive",
				canaries[1],
			)
			values[articleDeletionKeyPrefix+"v2"] = keyCanary
			setAdministratorNoticeContent(values, canaries[4])
			values[smtpFromEnvironment] = canaries[5]
			test.mutate(values)
			var calls struct {
				parse       atomic.Int64
				connect     atomic.Int64
				ping        atomic.Int64
				migrate     atomic.Int64
				composition atomic.Int64
				listener    atomic.Int64
			}
			dependencies := validCompositionDependencies(values)
			dependencies.parseDatabase = func(string) (*pgxpool.Config, error) {
				calls.parse.Add(1)
				return new(pgxpool.Config), nil
			}
			dependencies.connect = func(
				context.Context,
				*pgxpool.Config,
			) (*pgxpool.Pool, error) {
				calls.connect.Add(1)
				return new(pgxpool.Pool), nil
			}
			dependencies.ping = func(context.Context, *pgxpool.Pool) error {
				calls.ping.Add(1)
				return nil
			}
			dependencies.migrate = func(context.Context, *pgxpool.Pool) error {
				calls.migrate.Add(1)
				return nil
			}
			dependencies.composeBusiness = func(
				*pgxpool.Pool,
				businessComposition,
			) (http.Handler, error) {
				calls.composition.Add(1)
				return http.NotFoundHandler(), nil
			}
			dependencies.listen = func(string, string) (net.Listener, error) {
				calls.listener.Add(1)
				return &recordingListener{}, nil
			}

			err := run(context.Background(), dependencies)

			if err == nil {
				t.Fatal("run() error = nil")
			}
			gotCalls := []int64{
				calls.parse.Load(),
				calls.connect.Load(),
				calls.ping.Load(),
				calls.migrate.Load(),
				calls.composition.Load(),
				calls.listener.Load(),
			}
			if !reflect.DeepEqual(gotCalls, []int64{0, 0, 0, 0, 0, 0}) {
				t.Fatalf("startup side-effect calls = %v, want all zero", gotCalls)
			}
			for _, canary := range canaries {
				if strings.Contains(err.Error(), canary) {
					t.Fatalf("startup error leaked canary %q: %v", canary, err)
				}
			}
		})
	}
}

func TestArticleJournal_AWIRECFG06SanitizesStartupFailure(t *testing.T) {
	const (
		pathCanary = `C:\sensitive\article-journal`
		keyCanary  = "article-key-canary"
		bodyCanary = "private-record-body-canary"
	)
	values := validCompositionEnvironment()
	dependencies := validCompositionDependencies(values)
	dependencies.prepareArticle = func(
		config articleDeletionStartupConfig,
	) (*preparedArticleDeletion, error) {
		return &preparedArticleDeletion{config: config}, nil
	}
	dependencies.openArticle = func(
		*preparedArticleDeletion,
	) (articleDeletionJournalOwner, error) {
		return nil, &os.PathError{
			Op:   "open",
			Path: pathCanary,
			Err:  errors.New(keyCanary + bodyCanary),
		}
	}

	err := run(context.Background(), dependencies)

	if !errors.Is(err, errArticleDeletionJournalUnavailable) {
		t.Fatalf("run() error = %v, want stable Article journal category", err)
	}
	for _, canary := range []string{pathCanary, keyCanary, bodyCanary} {
		if strings.Contains(err.Error(), canary) {
			t.Fatalf("startup error leaked canary %q: %v", canary, err)
		}
	}
}

func TestArticleJournal_AWIREJRN07SanitizesLifecycleCloseFailure(t *testing.T) {
	const (
		pathCanary = `C:\sensitive\article-journal`
		keyCanary  = "article-key-canary"
		bodyCanary = "private-record-body-canary"
	)
	values := validCompositionEnvironment()
	dependencies := validCompositionDependencies(values)
	dependencies.prepareArticle = func(
		config articleDeletionStartupConfig,
	) (*preparedArticleDeletion, error) {
		return &preparedArticleDeletion{config: config}, nil
	}
	dependencies.openArticle = func(
		*preparedArticleDeletion,
	) (articleDeletionJournalOwner, error) {
		return mainArticleJournalOwnerStub{closeErr: &os.PathError{
			Op:   "close",
			Path: pathCanary,
			Err:  errors.New(keyCanary + bodyCanary),
		}}, nil
	}
	dependencies.composeBusiness = func(
		_ *pgxpool.Pool,
		composition businessComposition,
	) (http.Handler, error) {
		composition.runtime.articleWorker = newLifecycleWorkerFake()
		return http.NotFoundHandler(), nil
	}
	dependencies.coordinate = func(_ context.Context, config lifecycleConfig) error {
		return errors.Join(config.closeArticle(), config.closePool())
	}

	err := run(context.Background(), dependencies)

	if !errors.Is(err, errArticleDeletionJournalUnavailable) {
		t.Fatalf("run() error = %v, want stable Article journal category", err)
	}
	for _, canary := range []string{pathCanary, keyCanary, bodyCanary} {
		if strings.Contains(err.Error(), canary) {
			t.Fatalf("lifecycle error leaked canary %q: %v", canary, err)
		}
	}
}

type mainArticleJournalStub struct{}

func (mainArticleJournalStub) Append(
	context.Context,
	article.DeletionTombstoneRequest,
) (article.ArticleDeletionTombstone, error) {
	return article.ArticleDeletionTombstone{}, errors.New("not called")
}

func (mainArticleJournalStub) FindByOperation(
	context.Context,
	string,
) (article.ArticleDeletionTombstone, error) {
	return article.ArticleDeletionTombstone{}, errors.New("not called")
}

type mainArticleJournalOwnerStub struct {
	mainArticleJournalStub
	closeErr error
}

func (stub mainArticleJournalOwnerStub) Close() error {
	return stub.closeErr
}

func TestAdministratorInformationNoticeEffectiveAtFlowsUnchangedToHTTP(t *testing.T) {
	const effectiveAt = "2026-10-24T00:00:00Z"
	values := validCompositionEnvironment()
	values[administratorNoticeEffectiveAtEnvironment] = effectiveAt
	config := mustLoadStartupConfig(t, values)
	defer clearStartupSecrets(&config)

	reader := identity.StaticAdministratorInformationNoticeReader{
		Notice: config.administratorNotice,
	}
	service := &mainAdministratorNoticeService{
		reader: reader,
		tx:     &mainLoginTx{},
	}
	privacyService := mainAdministratorPrivacyService{}
	handler, err := identity.NewAdministratorHandler(
		service,
		privacyService,
		service,
		routerSourceProvider{},
		4096,
		4096,
	)
	if err != nil {
		t.Fatalf("NewAdministratorHandler() error = %v", err)
	}
	authorizer, err := platform.NewHTTPAuthorizer(
		mainAdministratorResolver{},
		func() time.Time { return time.Date(2026, time.October, 23, 12, 0, 0, 0, time.UTC) },
	)
	if err != nil {
		t.Fatalf("NewHTTPAuthorizer() error = %v", err)
	}
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/admin/administrator-information-provision-notices/current",
		nil,
	)
	request.AddCookie(&http.Cookie{
		Name: platform.SessionCookieName, Value: "session-token",
	})
	response := httptest.NewRecorder()
	platform.TraceMiddleware(authorizer.Protect(
		session.AccessCurrentIdentity,
		http.HandlerFunc(handler.CurrentAdministratorInformationNotice),
	)).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var body struct {
		EffectiveAt string `json:"effective_at"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode notice response: %v", err)
	}
	if body.EffectiveAt != effectiveAt {
		t.Fatalf("effective_at = %q, want %q", body.EffectiveAt, effectiveAt)
	}
	if service.noticeCalls != 1 {
		t.Fatalf("notice service calls = %d, want 1", service.noticeCalls)
	}
}

func TestProductionBusinessHandler_AWIRERTR06IdentityRoutesRemainShared(t *testing.T) {
	config := mustLoadStartupConfig(t, validCompositionEnvironment())
	prepared, err := prepareProductionLoginDependencies(
		config,
		bytes.NewReader(bytes.Repeat([]byte{0x5a}, 128)),
	)
	clearStartupSecrets(&config)
	if err != nil {
		t.Fatalf("prepareProductionLoginDependencies() error = %v", err)
	}
	t.Cleanup(prepared.cleanup)
	handler, err := newProductionBusinessHandler(
		new(pgxpool.Pool),
		productionBusinessComposition(t, prepared),
	)
	if err != nil {
		t.Fatalf("newProductionBusinessHandler() error = %v", err)
	}
	tests := []struct {
		name       string
		method     string
		path       string
		origin     bool
		wantStatus int
		wantAllow  string
		wantCode   platform.Code
	}{
		{
			name:   "unknown path before origin",
			method: http.MethodPost, path: "/api/v1/admin/administrators/extra",
			wantStatus: http.StatusNotFound, wantCode: platform.CodeRequestRouteNotFound,
		},
		{
			name:       "notice wrong method before origin",
			method:     http.MethodPost,
			path:       "/api/v1/admin/administrator-information-provision-notices/current",
			wantStatus: http.StatusMethodNotAllowed, wantAllow: http.MethodGet,
			wantCode: platform.CodeRequestMethodNotAllowed,
		},
		{
			name:       "notice origin rejection",
			method:     http.MethodGet,
			path:       "/api/v1/admin/administrator-information-provision-notices/current",
			wantStatus: http.StatusForbidden, wantCode: platform.CodeAuthOriginRejected,
		},
		{
			name:   "notice session rejection",
			method: http.MethodGet,
			path:   "/api/v1/admin/administrator-information-provision-notices/current",
			origin: true, wantStatus: http.StatusUnauthorized,
			wantCode: platform.CodeAuthSessionRequired,
		},
		{
			name:   "create wrong method before origin",
			method: http.MethodGet, path: "/api/v1/admin/administrators",
			wantStatus: http.StatusMethodNotAllowed, wantAllow: http.MethodPost,
			wantCode: platform.CodeRequestMethodNotAllowed,
		},
		{
			name:   "create origin rejection",
			method: http.MethodPost, path: "/api/v1/admin/administrators",
			wantStatus: http.StatusForbidden, wantCode: platform.CodeAuthOriginRejected,
		},
		{
			name:   "create session rejection",
			method: http.MethodPost, path: "/api/v1/admin/administrators",
			origin: true, wantStatus: http.StatusUnauthorized,
			wantCode: platform.CodeAuthSessionRequired,
		},
		{
			name:   "privacy wrong method before origin",
			method: http.MethodGet, path: "/api/v1/users/me/privacy-acknowledgements",
			wantStatus: http.StatusMethodNotAllowed, wantAllow: http.MethodPost,
			wantCode: platform.CodeRequestMethodNotAllowed,
		},
		{
			name:   "privacy origin rejection",
			method: http.MethodPost, path: "/api/v1/users/me/privacy-acknowledgements",
			wantStatus: http.StatusForbidden, wantCode: platform.CodeAuthOriginRejected,
		},
		{
			name:   "privacy session rejection",
			method: http.MethodPost, path: "/api/v1/users/me/privacy-acknowledgements",
			origin: true, wantStatus: http.StatusUnauthorized,
			wantCode: platform.CodeAuthSessionRequired,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, nil)
			if test.origin {
				request.Header.Set("Origin", "https://community.example.invalid")
				request.Header.Set("Sec-Fetch-Site", "same-origin")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf(
					"status = %d, want %d; body = %s",
					response.Code,
					test.wantStatus,
					response.Body.String(),
				)
			}
			if response.Header().Get("Allow") != test.wantAllow {
				t.Fatalf("Allow = %q, want %q", response.Header().Get("Allow"), test.wantAllow)
			}
			var envelope platform.Envelope
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decode error envelope: %v", err)
			}
			if envelope.Code != test.wantCode || envelope.TraceID == "" ||
				response.Header().Get(platform.TraceHeader) != envelope.TraceID {
				t.Fatalf("error envelope = %#v", envelope)
			}
		})
	}
}

func TestProductionBusinessHandler_AWIRERTR06UsesOneSharedInstancePerDependency(
	t *testing.T,
) {
	var sink bytes.Buffer
	baseLogger := slog.New(slog.NewJSONHandler(&sink, nil))
	var noticeLogger *slog.Logger
	observations := map[string][]articleSharedDependency{}
	seam := &articleProductionTestSeam{
		shared: func(observation articleSharedDependency) {
			observations[observation.Kind] = append(
				observations[observation.Kind],
				observation,
			)
		},
	}

	_ = newArticleProductionHandlerWithTestSeam(t, seam, baseLogger)

	minimumConsumers := map[string]int{
		"pool":          4,
		"session_store": 4,
		"origin_guard":  4,
		"audit_store":   4,
		"logger":        2,
		"clock":         4,
		"id_provider":   4,
	}
	for kind, minimum := range minimumConsumers {
		entries := observations[kind]
		if len(entries) < minimum {
			t.Fatalf("%s observations = %#v, want at least %d consumers", kind, entries, minimum)
		}
		first := entries[0].Instance
		consumers := map[string]struct{}{}
		for _, entry := range entries {
			consumers[entry.Consumer] = struct{}{}
			if kind == "logger" && entry.Consumer == "notice" {
				var ok bool
				noticeLogger, ok = entry.Instance.(*slog.Logger)
				if !ok || noticeLogger == baseLogger {
					t.Fatal("notice logger is not derived from the shared root")
				}
				continue
			}
			if entry.Instance != first {
				t.Fatalf("%s used multiple instances: %#v", kind, entries)
			}
		}
		if len(consumers) < minimum {
			t.Fatalf("%s consumers = %v, want at least %d unique", kind, consumers, minimum)
		}
	}
	if noticeLogger == nil {
		t.Fatal("production notice logger was not observed")
	}
	noticeLogger.Log(context.Background(), slog.LevelWarn, "article transfer notice", "reason_bucket", "result_persistence_failed", "trace_id", mainOrphanTraceID)
	baseLogger.Info("root sink probe", "event", "root_test_probe")
	assertNoticeProductionLog(t, sink.Bytes(), nil)
}

func TestProductionPasswordResetWiring_UsesOneStoreForAllConsumers(t *testing.T) {
	var observations []articleSharedDependency
	seam := &articleProductionTestSeam{
		shared: func(observation articleSharedDependency) {
			if observation.Kind == "password_reset_gate" {
				observations = append(observations, observation)
			}
		},
	}

	_ = newArticleProductionHandlerWithTestSeam(t, seam)

	if len(observations) != 4 {
		t.Fatalf("password reset Gate observations = %#v, want four consumers", observations)
	}
	first, ok := observations[0].Instance.(*identity.PasswordResetStore)
	if !ok || first == nil {
		t.Fatalf("password reset Gate instance = %#v, want non-nil *identity.PasswordResetStore", observations[0].Instance)
	}
	wantConsumers := map[string]bool{
		"session": true, "login": true, "password": true, "password_reset": true,
	}
	for _, observation := range observations {
		if observation.Instance != first {
			t.Fatalf("password reset Gate instances = %#v, want one shared pointer", observations)
		}
		delete(wantConsumers, observation.Consumer)
	}
	if len(wantConsumers) != 0 {
		t.Fatalf("missing password reset Gate consumers = %v", wantConsumers)
	}
}

func TestProductionPasswordResetCompositionRoutes(t *testing.T) {
	expiresAt := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		path     string
		wantRole identity.Role
	}{
		{
			name: "member",
			path: "/api/v1/admin/users/" +
				"10000000-0000-4000-8000-000000000001/password-reset",
			wantRole: identity.RoleMember,
		},
		{
			name: "administrator",
			path: "/api/v1/admin/administrators/" +
				"10000000-0000-4000-8000-000000000002/password-reset",
			wantRole: identity.RoleAdmin,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &mainPasswordResetService{
				result: identity.PasswordResetResult{
					TemporaryCredentialExpiresAt: expiresAt,
					DeliveryStatus:               "accepted",
				},
			}
			handler := newArticleProductionHandlerWithTestSeam(
				t,
				&articleProductionTestSeam{
					principalResolver: &articleSecurityResolverSpy{
						principal: activeAdministratorPrincipal(),
					},
					passwordReset: service,
				},
			)
			request := passwordResetProductionRequest(
				http.MethodPost,
				test.path,
				true,
			)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != http.StatusOK || service.calls != 1 {
				t.Fatalf(
					"status/calls = %d/%d, want 200/1; body=%s",
					response.Code,
					service.calls,
					response.Body.String(),
				)
			}
			if service.request.TargetRole != test.wantRole ||
				service.request.TargetID == "" ||
				service.request.IdempotencyKey != "password-reset-key" ||
				service.request.Actor.UserID != "administrator-1" {
				t.Fatalf("password reset request = %#v", service.request)
			}
			var body map[string]json.RawMessage
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode password reset response: %v", err)
			}
			if len(body) != 3 || body["user_id"] == nil ||
				body["temporary_credential_expires_at"] == nil ||
				body["delivery_status"] == nil {
				t.Fatalf("password reset response fields = %v", body)
			}
			if strings.Contains(response.Body.String(), "temporary-password-canary") ||
				strings.Contains(response.Body.String(), "temporary_password") {
				t.Fatalf("password reset response leaked secret: %s", response.Body.String())
			}
		})
	}
}

func TestProductionPasswordResetOriginAuthorizationAndIdempotency(t *testing.T) {
	path := "/api/v1/admin/users/" +
		"10000000-0000-4000-8000-000000000001/password-reset"
	tests := []struct {
		name       string
		request    func() *http.Request
		resolver   *articleSecurityResolverSpy
		wantStatus int
		wantCode   platform.Code
	}{
		{
			name: "origin rejection",
			request: func() *http.Request {
				return passwordResetProductionRequest(http.MethodPost, path, false)
			},
			resolver: &articleSecurityResolverSpy{
				principal: activeAdministratorPrincipal(),
			},
			wantStatus: http.StatusForbidden,
			wantCode:   platform.CodeAuthOriginRejected,
		},
		{
			name: "session rejection",
			request: func() *http.Request {
				request := passwordResetProductionRequest(http.MethodPost, path, true)
				request.Header.Del("Cookie")
				return request
			},
			resolver: &articleSecurityResolverSpy{
				principal: activeAdministratorPrincipal(),
			},
			wantStatus: http.StatusUnauthorized,
			wantCode:   platform.CodeAuthSessionRequired,
		},
		{
			name: "csrf rejection",
			request: func() *http.Request {
				return passwordResetProductionRequest(http.MethodPost, path, true)
			},
			resolver: &articleSecurityResolverSpy{
				rejectAt: "csrf", principal: activeAdministratorPrincipal(),
			},
			wantStatus: http.StatusForbidden,
			wantCode:   platform.CodeAuthCSRFFailed,
		},
		{
			name: "restricted principal rejection",
			request: func() *http.Request {
				return passwordResetProductionRequest(http.MethodPost, path, true)
			},
			resolver: &articleSecurityResolverSpy{
				principal: session.Principal{
					SessionID: "session-1", UserID: "administrator-1",
					Role: "admin", Status: "active", MustChangePassword: true,
					AuthenticatedCredential: session.CredentialPermanent,
				},
			},
			wantStatus: http.StatusForbidden,
			wantCode:   platform.CodeAuthMustCompleteOnboarding,
		},
		{
			name: "missing idempotency key",
			request: func() *http.Request {
				request := passwordResetProductionRequest(http.MethodPost, path, true)
				request.Header.Del("Idempotency-Key")
				return request
			},
			resolver: &articleSecurityResolverSpy{
				principal: activeAdministratorPrincipal(),
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   platform.CodeIdentityIdempotencyKeyRequired,
		},
		{
			name: "invalid target input",
			request: func() *http.Request {
				return passwordResetProductionRequest(
					http.MethodPost,
					"/api/v1/admin/users/not-a-uuid/password-reset",
					true,
				)
			},
			resolver: &articleSecurityResolverSpy{
				principal: activeAdministratorPrincipal(),
			},
			wantStatus: http.StatusNotFound,
			wantCode:   platform.CodeIdentityUserNotFound,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &mainPasswordResetService{}
			handler := newArticleProductionHandlerWithTestSeam(
				t,
				&articleProductionTestSeam{
					principalResolver: test.resolver,
					passwordReset:     service,
				},
			)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, test.request())

			if response.Code != test.wantStatus || service.calls != 0 {
				t.Fatalf(
					"status/service calls = %d/%d, want %d/0; body=%s",
					response.Code,
					service.calls,
					test.wantStatus,
					response.Body.String(),
				)
			}
			var envelope platform.Envelope
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decode error envelope: %v", err)
			}
			if envelope.Code != test.wantCode || envelope.TraceID == "" {
				t.Fatalf("error envelope = %#v", envelope)
			}
		})
	}
}

func TestProductionPasswordResetCompositionFailsClosedWithoutDurableDependency(
	t *testing.T,
) {
	tests := []struct {
		name   string
		remove func(*productionBusinessBuilder)
	}{
		{"store", func(builder *productionBusinessBuilder) {
			builder.passwordResetGate = nil
		}},
		{"permit", func(builder *productionBusinessBuilder) {
			builder.passwordResetPermit = nil
		}},
		{"operator", func(builder *productionBusinessBuilder) {
			builder.passwordResetOperator = nil
		}},
		{"worker", func(builder *productionBusinessBuilder) {
			builder.passwordResetWorker = nil
		}},
		{"mailer", func(builder *productionBusinessBuilder) {
			builder.prepared.administratorMailer = nil
		}},
		{"hasher", func(builder *productionBusinessBuilder) {
			builder.prepared.passwords = nil
		}},
		{"id provider", func(builder *productionBusinessBuilder) {
			builder.prepared.ids = nil
		}},
		{"clock", func(builder *productionBusinessBuilder) {
			builder.prepared.clock = nil
		}},
		{"idempotency keyring", func(builder *productionBusinessBuilder) {
			builder.prepared.idempotencyKeys = idempotency.DigestKeyring{}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := mustLoadStartupConfig(t, validCompositionEnvironment())
			prepared, err := prepareProductionLoginDependencies(
				config,
				bytes.NewReader(bytes.Repeat([]byte{0x5a}, 128)),
			)
			clearStartupSecrets(&config)
			if err != nil {
				t.Fatalf("prepareProductionLoginDependencies() error = %v", err)
			}
			t.Cleanup(prepared.cleanup)
			builder, err := newProductionBusinessBuilder(
				new(pgxpool.Pool),
				productionBusinessComposition(t, prepared),
			)
			if err != nil {
				t.Fatalf("newProductionBusinessBuilder() error = %v", err)
			}
			test.remove(builder)

			if handler, err := builder.newPasswordResetHandler(); err == nil ||
				handler != nil {
				t.Fatalf(
					"newPasswordResetHandler() = %#v, %v; want fail closed",
					handler,
					err,
				)
			}
		})
	}
}

func TestProductionPasswordResetRouteMethodNotFound(t *testing.T) {
	resetCalls := 0
	reset := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		resetCalls++
		response.WriteHeader(http.StatusNoContent)
	})
	router := newLoginBusinessRouter(
		http.NotFoundHandler(),
		http.NotFoundHandler(),
		http.NotFoundHandler(),
		http.NotFoundHandler(),
		http.NotFoundHandler(),
		http.NotFoundHandler(),
		http.NotFoundHandler(),
		http.NotFoundHandler(),
		reset,
	)
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantAllow  string
		wantCalls  int
		wantCode   platform.Code
	}{
		{
			name: "member route", method: http.MethodPost,
			path:       "/api/v1/admin/users/10000000-0000-4000-8000-000000000001/password-reset",
			wantStatus: http.StatusNoContent, wantCalls: 1,
		},
		{
			name: "administrator route", method: http.MethodPost,
			path:       "/api/v1/admin/administrators/10000000-0000-4000-8000-000000000002/password-reset",
			wantStatus: http.StatusNoContent, wantCalls: 1,
		},
		{
			name: "known path wrong method", method: http.MethodGet,
			path:       "/api/v1/admin/users/10000000-0000-4000-8000-000000000001/password-reset",
			wantStatus: http.StatusMethodNotAllowed, wantAllow: http.MethodPost,
			wantCode: platform.CodeRequestMethodNotAllowed,
		},
		{
			name: "missing target id", method: http.MethodPost,
			path:       "/api/v1/admin/users/password-reset",
			wantStatus: http.StatusNotFound,
			wantCode:   platform.CodeRequestRouteNotFound,
		},
		{
			name: "extra path segment", method: http.MethodPost,
			path:       "/api/v1/admin/users/10000000-0000-4000-8000-000000000001/password-reset/extra",
			wantStatus: http.StatusNotFound,
			wantCode:   platform.CodeRequestRouteNotFound,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := resetCalls
			response := httptest.NewRecorder()
			router.ServeHTTP(
				response,
				httptest.NewRequest(test.method, test.path, nil),
			)
			if response.Code != test.wantStatus ||
				response.Header().Get("Allow") != test.wantAllow {
				t.Fatalf(
					"status/Allow = %d/%q, want %d/%q; body=%s",
					response.Code,
					response.Header().Get("Allow"),
					test.wantStatus,
					test.wantAllow,
					response.Body.String(),
				)
			}
			if resetCalls-before != test.wantCalls {
				t.Fatalf("reset calls = %d, want %d", resetCalls-before, test.wantCalls)
			}
			if test.wantCode != "" {
				var envelope platform.Envelope
				if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
					t.Fatalf("decode route error envelope: %v", err)
				}
				if envelope.Code != test.wantCode || envelope.TraceID == "" ||
					response.Header().Get(platform.TraceHeader) != envelope.TraceID {
					t.Fatalf("route error envelope = %#v", envelope)
				}
			}
		})
	}
}

func TestProductionPasswordResetWiring_CompositionFunctionFitsLineLimit(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() could not locate test file")
	}
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(
		fileSet,
		filepath.Join(filepath.Dir(testFile), "main.go"),
		nil,
		0,
	)
	if err != nil {
		t.Fatalf("parse main.go: %v", err)
	}
	targets := map[string]bool{
		"newProductionBusinessHandler": true, "newProductionBusinessBuilder": true,
		"validateProductionBusinessComposition": true, "observeIdentityCore": true,
		"buildIdentityHandlers": true, "newLoginService": true,
		"newPasswordChangeService": true, "observePasswordDependencies": true,
		"newLoginHandler": true, "newPasswordResetHandler": true,
		"newAuthorizer": true, "newAdministratorHandler": true,
		"newProcessingGate": true, "newAdministratorServices": true,
		"observeAdministratorDependencies": true, "protectIdentityHandlers": true,
		"buildArticleHandlers": true, "observeArticleDependencies": true,
		"newArticleComponents": true, "newArticleFinalizer": true,
		"newArticleWorker": true, "newArticleCoordinator": true,
		"newArticleHandler": true, "protectArticleHandlers": true,
	}
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil || !targets[function.Name.Name] {
			continue
		}
		lines := fileSet.Position(function.Body.End()).Line -
			fileSet.Position(function.Body.Pos()).Line + 1
		t.Logf("%s body lines = %d", function.Name.Name, lines)
		if lines > 50 {
			t.Errorf("%s body lines = %d, want <= 50", function.Name.Name, lines)
		}
		delete(targets, function.Name.Name)
	}
	for name := range targets {
		t.Errorf("%s declaration not found", name)
	}
}

func TestAdministratorRoutesOrderOriginSessionCSRFAccessThenHandler(t *testing.T) {
	tests := []struct {
		name             string
		method           string
		path             string
		origin           bool
		cookie           bool
		csrf             bool
		restricted       bool
		wantStatus       int
		wantAllow        string
		wantOriginReject int
		wantResolve      int
		wantHandler      int
	}{
		{
			name:   "unknown path",
			method: http.MethodPost, path: "/unknown",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "notice wrong method",
			method:     http.MethodPost,
			path:       "/api/v1/admin/administrator-information-provision-notices/current",
			wantStatus: http.StatusMethodNotAllowed, wantAllow: http.MethodGet,
		},
		{
			name:       "notice origin rejection",
			method:     http.MethodGet,
			path:       "/api/v1/admin/administrator-information-provision-notices/current",
			wantStatus: http.StatusForbidden, wantOriginReject: 1,
		},
		{
			name:   "notice requires session",
			method: http.MethodGet,
			path:   "/api/v1/admin/administrator-information-provision-notices/current",
			origin: true, wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "notice does not require CSRF and permits restricted principal to coordinator",
			method: http.MethodGet,
			path:   "/api/v1/admin/administrator-information-provision-notices/current",
			origin: true, cookie: true, restricted: true,
			wantStatus: http.StatusNoContent, wantResolve: 1, wantHandler: 1,
		},
		{
			name:   "create requires CSRF",
			method: http.MethodPost, path: "/api/v1/admin/administrators",
			origin: true, cookie: true,
			wantStatus: http.StatusForbidden, wantResolve: 1,
		},
		{
			name:   "create rejects restricted AccessGeneral",
			method: http.MethodPost, path: "/api/v1/admin/administrators",
			origin: true, cookie: true, csrf: true, restricted: true,
			wantStatus: http.StatusForbidden, wantResolve: 1,
		},
		{
			name:   "create reaches handler",
			method: http.MethodPost, path: "/api/v1/admin/administrators",
			origin: true, cookie: true, csrf: true,
			wantStatus: http.StatusNoContent, wantResolve: 1, wantHandler: 1,
		},
		{
			name:   "privacy permits restricted acknowledgement access",
			method: http.MethodPost, path: "/api/v1/users/me/privacy-acknowledgements",
			origin: true, cookie: true, csrf: true, restricted: true,
			wantStatus: http.StatusNoContent, wantResolve: 1, wantHandler: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			originRejects, handlerCalls := 0, 0
			resolver := &mainAdministratorRouteResolver{
				restricted: test.restricted,
			}
			guard, err := platform.NewObservedOriginGuard(
				"https://community.example.invalid",
				routerOriginObserver{calls: &originRejects},
				time.Second,
			)
			if err != nil {
				t.Fatalf("NewObservedOriginGuard() error = %v", err)
			}
			authorizer, err := platform.NewHTTPAuthorizer(
				resolver,
				func() time.Time {
					return time.Date(2026, time.October, 23, 12, 0, 0, 0, time.UTC)
				},
			)
			if err != nil {
				t.Fatalf("NewHTTPAuthorizer() error = %v", err)
			}
			reached := http.HandlerFunc(func(
				response http.ResponseWriter,
				_ *http.Request,
			) {
				handlerCalls++
				response.WriteHeader(http.StatusNoContent)
			})
			router := newLoginBusinessRouter(
				http.NotFoundHandler(),
				http.NotFoundHandler(),
				guard.Protect(authorizer.Protect(session.AccessCurrentIdentity, reached)),
				guard.Protect(authorizer.Protect(session.AccessGeneral, reached)),
				guard.Protect(authorizer.Protect(session.AccessAcknowledgePrivacy, reached)),
			)
			request := httptest.NewRequest(test.method, test.path, nil)
			if test.origin {
				request.Header.Set("Origin", "https://community.example.invalid")
				request.Header.Set("Sec-Fetch-Site", "same-origin")
			}
			if test.cookie {
				request.AddCookie(&http.Cookie{
					Name: platform.SessionCookieName, Value: "session-token",
				})
			}
			if test.csrf {
				request.Header.Set(platform.CSRFHeaderName, "csrf-token")
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != test.wantStatus ||
				response.Header().Get("Allow") != test.wantAllow ||
				originRejects != test.wantOriginReject ||
				resolver.calls != test.wantResolve ||
				handlerCalls != test.wantHandler {
				t.Fatalf(
					"status/Allow/origin/resolver/handler = %d/%q/%d/%d/%d",
					response.Code,
					response.Header().Get("Allow"),
					originRejects,
					resolver.calls,
					handlerCalls,
				)
			}
		})
	}
}

func TestOnboardingRouterOrdersExactOriginSessionCSRFThenHandler(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		path         string
		origin       bool
		cookie       bool
		csrf         bool
		wantStatus   int
		wantOrigin   int
		wantResolver int
		wantHandler  int
	}{
		{
			name: "unknown path", method: http.MethodPut, path: "/unknown",
			wantStatus: http.StatusNotFound,
		},
		{
			name: "wrong method", method: http.MethodPost,
			path:       "/api/v1/users/me/password",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name: "origin rejection", method: http.MethodPut,
			path:       "/api/v1/users/me/password",
			wantStatus: http.StatusForbidden, wantOrigin: 1,
		},
		{
			name: "session required", method: http.MethodPut,
			path: "/api/v1/users/me/password", origin: true,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "csrf rejected", method: http.MethodPut,
			path: "/api/v1/users/me/password", origin: true, cookie: true,
			wantStatus: http.StatusForbidden, wantResolver: 1,
		},
		{
			name: "handler reached", method: http.MethodPut,
			path: "/api/v1/users/me/password", origin: true, cookie: true, csrf: true,
			wantStatus: http.StatusNoContent, wantResolver: 1, wantHandler: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			originCalls, resolverCalls, handlerCalls := 0, 0, 0
			guard, err := platform.NewObservedOriginGuard(
				"https://community.example.invalid",
				routerOriginObserver{calls: &originCalls},
				time.Second,
			)
			if err != nil {
				t.Fatalf("NewObservedOriginGuard() error = %v", err)
			}
			resolver := routerPasswordResolver{calls: &resolverCalls}
			authorizer, err := platform.NewHTTPAuthorizer(
				resolver,
				func() time.Time { return routerLoginClock{}.Now() },
			)
			if err != nil {
				t.Fatalf("NewHTTPAuthorizer() error = %v", err)
			}
			password := guard.Protect(authorizer.Protect(
				session.AccessChangePassword,
				http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
					handlerCalls++
					response.WriteHeader(http.StatusNoContent)
				}),
			))
			router := newLoginBusinessRouter(http.NotFoundHandler(), password)
			request := httptest.NewRequest(test.method, test.path, nil)
			if test.origin {
				request.Header.Set("Origin", "https://community.example.invalid")
				request.Header.Set("Sec-Fetch-Site", "same-origin")
			}
			if test.cookie {
				request.AddCookie(&http.Cookie{
					Name: platform.SessionCookieName, Value: "session-token",
				})
			}
			if test.csrf {
				request.Header.Set(platform.CSRFHeaderName, "csrf-token")
			}
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code != test.wantStatus ||
				originCalls != test.wantOrigin ||
				resolverCalls != test.wantResolver ||
				handlerCalls != test.wantHandler {
				t.Fatalf(
					"status/origin/resolver/handler = %d/%d/%d/%d",
					response.Code,
					originCalls,
					resolverCalls,
					handlerCalls,
				)
			}
			if response.Header().Get(platform.TraceHeader) == "" {
				t.Fatal("trace header is empty")
			}
		})
	}
}

func TestLoginBusinessRouter_CrossLayerOrderingAndZeroCalls(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		path        string
		origin      string
		fetchSite   string
		wantStatus  int
		wantOrigin  int
		wantAttempt int
		wantService int
		wantMetrics int
	}{
		{
			name:   "unknown path bypasses origin and identity",
			method: http.MethodPost, path: "/unknown",
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "wrong method bypasses origin and identity",
			method: http.MethodGet, path: "/api/v1/sessions",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "exact post origin rejection precedes identity",
			method: http.MethodPost, path: "/api/v1/sessions",
			wantStatus: http.StatusForbidden, wantOrigin: 1,
		},
		{
			name:   "exact post accepted origin reaches identity",
			method: http.MethodPost, path: "/api/v1/sessions",
			origin: "https://community.example.invalid", fetchSite: "same-origin",
			wantStatus:  http.StatusUnauthorized,
			wantAttempt: 1, wantService: 1, wantMetrics: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var counters routerLayerCounters
			originGuard, err := platform.NewObservedOriginGuard(
				"https://community.example.invalid",
				routerOriginObserver{calls: &counters.origin},
				time.Second,
			)
			if err != nil {
				t.Fatalf("NewObservedOriginGuard() error = %v", err)
			}
			coordinator, err := identity.NewLoginAttemptCoordinator(
				&routerLoginIDGenerator{calls: &counters.attempt},
				routerLoginClock{},
				routerFailureAuditor{},
			)
			if err != nil {
				t.Fatalf("NewLoginAttemptCoordinator() error = %v", err)
			}
			handler, err := identity.NewLoginHandler(
				routerLoginService{calls: &counters.service},
				routerSourceProvider{},
				4096,
				identity.LoginHandlerConfig{
					Coordinator:       coordinator,
					Observer:          routerLoginObserver{calls: &counters.metrics},
					ObservationBudget: time.Second,
				},
			)
			if err != nil {
				t.Fatalf("NewLoginHandler() error = %v", err)
			}
			router := newLoginBusinessRouter(
				originGuard.Protect(http.HandlerFunc(handler.Login)),
			)
			request := httptest.NewRequest(
				test.method,
				test.path,
				strings.NewReader(`{"username":"Alice","password":"Wrong123"}`),
			)
			request.Header.Set("Content-Type", "application/json")
			if test.origin != "" {
				request.Header.Set("Origin", test.origin)
			}
			if test.fetchSite != "" {
				request.Header.Set("Sec-Fetch-Site", test.fetchSite)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if counters.origin != test.wantOrigin ||
				counters.attempt != test.wantAttempt ||
				counters.service != test.wantService ||
				counters.metrics != test.wantMetrics {
				t.Fatalf(
					"origin/attempt/service/metrics calls = %d/%d/%d/%d, want %d/%d/%d/%d",
					counters.origin,
					counters.attempt,
					counters.service,
					counters.metrics,
					test.wantOrigin,
					test.wantAttempt,
					test.wantService,
					test.wantMetrics,
				)
			}
		})
	}
}

func TestProductionDependencies_ExecuteRealLoginPreparationAndFactory(t *testing.T) {
	config := mustLoadStartupConfig(t, validCompositionEnvironment())
	prepared, err := prepareProductionLoginDependencies(
		config,
		bytes.NewReader(bytes.Repeat([]byte{0x5a}, 128)),
	)
	clear(config.loginBucketKey)
	clear(config.sourceKey)
	if err != nil {
		t.Fatalf("prepareProductionLoginDependencies() error = %v", err)
	}
	composition := productionBusinessComposition(t, prepared)
	handler, err := newProductionBusinessHandler(new(pgxpool.Pool), composition)
	if err != nil {
		t.Fatalf("newProductionBusinessHandler() error = %v", err)
	}
	if handler == nil {
		t.Fatal("newProductionBusinessHandler() handler = nil")
	}
	if prepared.fakeProbe != "" {
		t.Fatal("production factory retained the fake credential probe")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/not-a-route", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("production router status = %d, want 404", response.Code)
	}
}

func TestRun_ProductionArgon2StartupConsumersFailBeforeListener(t *testing.T) {
	t.Run("startup fake Hash failure", func(t *testing.T) {
		values := validCompositionEnvironment()
		dependencies := validCompositionDependencies(values)
		var calls []string
		dependencies.prepareBusiness = func(config startupConfig) (*preparedLoginDependencies, error) {
			calls = append(calls, "prepare")
			config.argon2 = security.Argon2idConfig{
				MemoryKiB: 8, Iterations: 1, Parallelism: 1, SaltLength: 8, KeyLength: 16,
			}
			return prepareProductionLoginDependencies(
				config,
				&mainNthReadErrorReader{
					failAt: 2,
					err:    errors.New("synthetic startup fake hash entropy failure"),
				},
			)
		}
		dependencies.parseDatabase = func(string) (*pgxpool.Config, error) {
			calls = append(calls, "parse")
			return new(pgxpool.Config), nil
		}
		dependencies.connect = func(context.Context, *pgxpool.Config) (*pgxpool.Pool, error) {
			calls = append(calls, "connect")
			return new(pgxpool.Pool), nil
		}
		dependencies.listen = func(string, string) (net.Listener, error) {
			calls = append(calls, "listen")
			return &recordingListener{}, nil
		}

		if err := run(context.Background(), dependencies); err == nil {
			t.Fatal("run() error = nil")
		}
		if !reflect.DeepEqual(calls, []string{"prepare"}) {
			t.Fatalf("calls = %v, want prepare only", calls)
		}
	})

	t.Run("startup fake Verify capacity", func(t *testing.T) {
		values := validCompositionEnvironment()
		config := mustLoadStartupConfig(t, values)
		config.argon2 = security.Argon2idConfig{
			MemoryKiB: 8, Iterations: 1, Parallelism: 1, SaltLength: 8, KeyLength: 16,
		}
		random := &mainGateRandom{}
		prepared, err := prepareProductionLoginDependencies(config, random)
		clear(config.loginBucketKey)
		clear(config.sourceKey)
		if err != nil {
			t.Fatalf("prepareProductionLoginDependencies() error = %v", err)
		}
		if random.ReadCalls() != 2 {
			t.Fatalf(
				"startup random reads = %d, want probe plus fake Hash salt",
				random.ReadCalls(),
			)
		}
		fakeProbe := prepared.fakeProbe
		fakeDigest := prepared.fakeDigest
		entered, release := random.Arm()
		holderDone := make(chan error, 1)
		go func() {
			_, hashErr := prepared.passwords.Hash("synthetic-production-permit-holder")
			holderDone <- hashErr
		}()
		<-entered
		defer func() {
			if release != nil {
				close(release)
			}
			if holderDone != nil {
				if err := <-holderDone; err != nil {
					t.Errorf("permit holder error = %v", err)
				}
			}
		}()

		_, hashCapacityErr := prepared.passwords.Hash(fakeProbe)
		var hashCapacity *security.Argon2idCapacityError
		if !errors.As(hashCapacityErr, &hashCapacity) ||
			hashCapacity.RetryAfter() != time.Second {
			t.Fatalf("startup fake Hash error = %v, want typed 1s capacity", hashCapacityErr)
		}
		_, _, capacityErr := prepared.passwords.Verify(fakeProbe, fakeDigest)
		var capacity *security.Argon2idCapacityError
		if !errors.As(capacityErr, &capacity) || capacity.RetryAfter() != time.Second {
			t.Fatalf("startup fake Verify error = %v, want typed 1s capacity", capacityErr)
		}

		dependencies := validCompositionDependencies(values)
		poolCloses, listenCalls, serverCalls, coordinateCalls := 0, 0, 0, 0
		dependencies.prepareBusiness = func(startupConfig) (*preparedLoginDependencies, error) {
			return prepared, nil
		}
		dependencies.composeBusiness = newProductionBusinessHandler
		dependencies.closePool = func(*pgxpool.Pool) { poolCloses++ }
		dependencies.listen = func(string, string) (net.Listener, error) {
			listenCalls++
			return &recordingListener{}, nil
		}
		dependencies.newServer = func(string, http.Handler) lifecycleServer {
			serverCalls++
			return nil
		}
		dependencies.coordinate = func(context.Context, lifecycleConfig) error {
			coordinateCalls++
			return nil
		}

		if err := run(context.Background(), dependencies); err == nil {
			t.Fatal("run() error = nil")
		}
		if poolCloses != 1 || listenCalls != 0 || serverCalls != 0 || coordinateCalls != 0 {
			t.Fatalf(
				"pool/listen/server/coordinate calls = %d/%d/%d/%d, want 1/0/0/0",
				poolCloses,
				listenCalls,
				serverCalls,
				coordinateCalls,
			)
		}

		close(release)
		release = nil
		if err := <-holderDone; err != nil {
			t.Fatalf("permit holder error = %v", err)
		}
		holderDone = nil
		if _, _, err := prepared.passwords.Verify(fakeProbe, fakeDigest); err != nil {
			t.Fatalf("startup fake Verify after release error = %v", err)
		}
	})
}

func TestProductionArgon2Gate_IsSharedByRuntimeLoginConsumers(t *testing.T) {
	config := mustLoadStartupConfig(t, validCompositionEnvironment())
	config.argon2 = security.Argon2idConfig{
		MemoryKiB: 16, Iterations: 2, Parallelism: 1, SaltLength: 8, KeyLength: 16,
	}
	random := &mainGateRandom{}
	prepared, err := prepareProductionLoginDependencies(config, random)
	clear(config.loginBucketKey)
	clear(config.sourceKey)
	if err != nil {
		t.Fatalf("prepareProductionLoginDependencies() error = %v", err)
	}
	if random.ReadCalls() != 2 {
		t.Fatalf(
			"startup random reads = %d, want probe plus fake Hash salt",
			random.ReadCalls(),
		)
	}
	oldConfig := config.argon2
	oldConfig.MemoryKiB = 8
	oldConfig.Iterations = 1
	oldHasher, err := security.NewArgon2idHasher(
		oldConfig,
		bytes.NewReader(bytes.Repeat([]byte{0x61}, 2*int(oldConfig.SaltLength))),
	)
	if err != nil {
		t.Fatalf("NewArgon2idHasher(old) error = %v", err)
	}
	permanentDigest, err := oldHasher.Hash("Password123")
	if err != nil {
		t.Fatalf("Hash(permanent) error = %v", err)
	}
	temporaryDigest, err := oldHasher.Hash("Temporary123")
	if err != nil {
		t.Fatalf("Hash(temporary) error = %v", err)
	}
	now := time.Date(2026, time.October, 23, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name          string
		password      string
		setup         func(*mainLoginRepository)
		rehashDigest  string
		startOccupied bool
	}{
		{
			name:     "unknown user fake Verify",
			password: "Password123",
			setup: func(repository *mainLoginRepository) {
				repository.userErr = pgx.ErrNoRows
			},
			startOccupied: true,
		},
		{
			name:     "temporary credential rehash",
			password: "Temporary123",
			setup: func(repository *mainLoginRepository) {
				repository.user = mainLoginUser(identity.CredentialModeTemporary, "")
				repository.temporary = &identity.TemporaryCredential{
					ID:               "00000000-0000-4000-8000-000000000010",
					UserID:           repository.user.ID,
					Purpose:          identity.TemporaryCredentialPasswordReset,
					Generation:       2,
					State:            identity.TemporaryCredentialActive,
					SecretDigest:     temporaryDigest,
					DeliveredToEmail: "synthetic@example.invalid",
					IssuedAt:         now.Add(-time.Hour),
					ExpiresAt:        now.Add((7 * 24 * time.Hour) - time.Hour),
				}
			},
			rehashDigest: temporaryDigest,
		},
		{
			name:     "permanent credential rehash",
			password: "Password123",
			setup: func(repository *mainLoginRepository) {
				repository.user = mainLoginUser(identity.CredentialModePermanent, permanentDigest)
			},
			rehashDigest: permanentDigest,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entered, release := random.Arm()
			holderDone := make(chan error, 1)
			holderStarted := false
			startHolder := func() {
				if holderStarted {
					t.Fatal("permit holder started more than once")
				}
				holderStarted = true
				go func() {
					_, hashErr := prepared.passwords.Hash("synthetic-production-permit-holder")
					holderDone <- hashErr
				}()
				<-entered
			}
			defer func() {
				if holderStarted {
					close(release)
					if err := <-holderDone; err != nil {
						t.Errorf("permit holder error = %v", err)
					}
				}
			}()

			var passwords identity.LoginPasswordHasher = prepared.passwords
			if test.rehashDigest != "" {
				passwords = &mainRehashBarrierHasher{
					delegate: prepared.passwords,
					digest:   test.rehashDigest,
					onRehash: startHolder,
				}
			}
			transaction := &mainLoginTx{}
			repository := &mainLoginRepository{}
			test.setup(repository)
			abuse := &mainLoginAbuse{}
			auditAppender := &mainLoginAudit{}
			sessions := &mainLoginSessions{}
			service, err := identity.NewLoginService(identity.LoginDependencies{
				Transactions:         mainLoginTransactor{transaction: transaction},
				Repository:           repository,
				PasswordResetGate:    mainPasswordResetGate{},
				Passwords:            passwords,
				Sessions:             sessions,
				SessionIssuer:        mainLoginSessionIssuer{},
				Abuse:                abuse,
				RecoveryMode:         mainLoginRecoveryMode{},
				Audit:                auditAppender,
				IDs:                  mainLoginIDs{},
				Clock:                mainLoginClock{now: now},
				FakeCredentialDigest: prepared.fakeDigest,
				FakeCredentialProbe:  prepared.fakeProbe,
			})
			if err != nil {
				t.Fatalf("NewLoginService() error = %v", err)
			}
			failureAuditor := &mainLoginFailureAuditor{}
			coordinator, err := identity.NewLoginAttemptCoordinator(
				mainLoginIDs{},
				mainLoginClock{now: now},
				failureAuditor,
			)
			if err != nil {
				t.Fatalf("NewLoginAttemptCoordinator() error = %v", err)
			}
			observer := &mainLoginObserver{}
			handler, err := identity.NewLoginHandler(
				service,
				routerSourceProvider{},
				4096,
				identity.LoginHandlerConfig{
					Coordinator:       coordinator,
					Observer:          observer,
					ObservationBudget: time.Second,
				},
			)
			if err != nil {
				t.Fatalf("NewLoginHandler() error = %v", err)
			}
			if test.startOccupied {
				startHolder()
			}

			request := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/sessions",
				strings.NewReader(
					`{"username":"Alice","password":"`+test.password+`"}`,
				),
			)
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			platform.TraceMiddleware(http.HandlerFunc(handler.Login)).ServeHTTP(response, request)

			if !holderStarted {
				t.Fatal("runtime consumer did not reach the shared production gate")
			}
			if response.Code != http.StatusTooManyRequests ||
				response.Header().Get("Retry-After") != "1" {
				t.Fatalf(
					"status/Retry-After = %d/%q, want 429/1",
					response.Code,
					response.Header().Get("Retry-After"),
				)
			}
			var envelope platform.Envelope
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if envelope.Code != platform.CodeRequestRateLimited {
				t.Fatalf("response code = %q, want %q", envelope.Code, platform.CodeRequestRateLimited)
			}
			if abuse.failureCalls != 0 || abuse.successCalls != 0 ||
				auditAppender.calls != 0 || failureAuditor.calls != 1 ||
				sessions.calls != 0 {
				t.Fatalf(
					"abuse failure/success, audit/failure-audit, session calls = %d/%d, %d/%d, %d",
					abuse.failureCalls,
					abuse.successCalls,
					auditAppender.calls,
					failureAuditor.calls,
					sessions.calls,
				)
			}
			if transaction.commitCalls != 0 || transaction.rollbackCalls != 1 ||
				repository.permanentRehashes != 0 || repository.temporaryRehashes != 0 {
				t.Fatalf(
					"commit/rollback/permanent-rehash/temporary-rehash calls = %d/%d/%d/%d",
					transaction.commitCalls,
					transaction.rollbackCalls,
					repository.permanentRehashes,
					repository.temporaryRehashes,
				)
			}
			if observer.calls != 1 ||
				observer.last.Outcome != identity.LoginOutcomeRateLimited ||
				observer.last.ReasonBucket != "resource_capacity" {
				t.Fatalf("observer calls/last = %d/%#v", observer.calls, observer.last)
			}
			_, _, capacityErr := prepared.passwords.Verify(
				prepared.fakeProbe,
				prepared.fakeDigest,
			)
			var capacity *security.Argon2idCapacityError
			if !errors.As(capacityErr, &capacity) || capacity.RetryAfter() != time.Second {
				t.Fatalf("shared gate error = %v, want typed 1s capacity", capacityErr)
			}
			for _, denied := range []string{
				"Alice",
				test.password,
				prepared.fakeProbe,
				prepared.fakeDigest,
				permanentDigest,
				temporaryDigest,
				"v1:synthetic",
			} {
				if strings.Contains(response.Body.String(), denied) {
					t.Fatal("capacity response exposed sensitive input")
				}
			}

			close(release)
			holderStarted = false
			if err := <-holderDone; err != nil {
				t.Fatalf("permit holder error = %v", err)
			}
			if _, _, err := prepared.passwords.Verify(
				prepared.fakeProbe,
				prepared.fakeDigest,
			); err != nil {
				t.Fatalf("Verify() after capacity release error = %v", err)
			}
		})
	}
}

func TestRun_AWIRELIFE04LIFE05ProductionCompositionUnwindsOnBindFailure(t *testing.T) {
	for _, test := range []struct {
		name        string
		failAddress string
		wantClosed  int
	}{
		{name: "first bind", failAddress: "127.0.0.1:8080"},
		{name: "second bind", failAddress: "127.0.0.1:9090", wantClosed: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			values := validCompositionEnvironment()
			dependencies := validCompositionDependencies(values)
			prepareCalls, factoryCalls, poolCloses, listenerCloses := 0, 0, 0, 0
			var retainedBucket, retainedSource []byte
			dependencies.prepareBusiness = func(config startupConfig) (*preparedLoginDependencies, error) {
				prepareCalls++
				retainedBucket = config.loginBucketKey
				retainedSource = config.sourceKey
				return prepareProductionLoginDependencies(
					config,
					bytes.NewReader(bytes.Repeat([]byte{0x42}, 128)),
				)
			}
			dependencies.composeBusiness = func(
				pool *pgxpool.Pool,
				composition businessComposition,
			) (http.Handler, error) {
				factoryCalls++
				return newProductionBusinessHandler(pool, composition)
			}
			dependencies.closePool = func(*pgxpool.Pool) { poolCloses++ }
			dependencies.listen = func(_ string, address string) (net.Listener, error) {
				if address == test.failAddress {
					return nil, errors.New("synthetic bind failure")
				}
				return &recordingListener{onClose: func() { listenerCloses++ }}, nil
			}
			dependencies.newServer = func(string, http.Handler) lifecycleServer {
				t.Fatal("server constructed before both listeners bound")
				return nil
			}

			runErr := run(context.Background(), dependencies)
			if runErr == nil {
				t.Fatal("run() error = nil")
			}
			if prepareCalls != 1 || factoryCalls != 1 {
				t.Fatalf("prepare/factory calls = %d/%d, want 1/1", prepareCalls, factoryCalls)
			}
			if poolCloses != 1 || listenerCloses != test.wantClosed {
				t.Fatalf("pool/listener closes = %d/%d, want 1/%d; error = %v", poolCloses, listenerCloses, test.wantClosed, runErr)
			}
			assertClearedSecret(t, retainedBucket, values[loginBucketKeyPrefix+"v1"])
			assertClearedSecret(t, retainedSource, values[sourceKeyPrefix+"v1"])
		})
	}
}

func TestRun_AWIRELIFE04ConstructorFactoryFailuresHaveNoRouteOrListener(t *testing.T) {
	t.Run("prepared constructor failure", func(t *testing.T) {
		values := validCompositionEnvironment()
		dependencies := validCompositionDependencies(values)
		var calls []string
		dependencies.prepareBusiness = func(config startupConfig) (*preparedLoginDependencies, error) {
			calls = append(calls, "prepare")
			return prepareProductionLoginDependencies(
				config,
				mainErrorReader{err: errors.New("synthetic entropy failure")},
			)
		}
		dependencies.parseDatabase = func(string) (*pgxpool.Config, error) {
			calls = append(calls, "parse")
			return new(pgxpool.Config), nil
		}
		dependencies.connect = func(context.Context, *pgxpool.Config) (*pgxpool.Pool, error) {
			calls = append(calls, "connect")
			return new(pgxpool.Pool), nil
		}
		dependencies.listen = func(string, string) (net.Listener, error) {
			calls = append(calls, "listen")
			return &recordingListener{}, nil
		}

		if err := run(context.Background(), dependencies); err == nil {
			t.Fatal("run() error = nil")
		}
		if !reflect.DeepEqual(calls, []string{"prepare"}) {
			t.Fatalf("calls = %v, want prepare only", calls)
		}
	})

	t.Run("real handler factory failure", func(t *testing.T) {
		values := validCompositionEnvironment()
		dependencies := validCompositionDependencies(values)
		poolCloses, listenCalls, serverCalls, coordinateCalls := 0, 0, 0, 0
		dependencies.prepareBusiness = func(config startupConfig) (*preparedLoginDependencies, error) {
			return prepareProductionLoginDependencies(
				config,
				bytes.NewReader(bytes.Repeat([]byte{0x24}, 128)),
			)
		}
		dependencies.composeBusiness = func(
			pool *pgxpool.Pool,
			composition businessComposition,
		) (http.Handler, error) {
			composition.failureAuditor = nil
			return newProductionBusinessHandler(pool, composition)
		}
		dependencies.closePool = func(*pgxpool.Pool) { poolCloses++ }
		dependencies.listen = func(string, string) (net.Listener, error) {
			listenCalls++
			return &recordingListener{}, nil
		}
		dependencies.newServer = func(string, http.Handler) lifecycleServer {
			serverCalls++
			return nil
		}
		dependencies.coordinate = func(context.Context, lifecycleConfig) error {
			coordinateCalls++
			return nil
		}

		if err := run(context.Background(), dependencies); err == nil {
			t.Fatal("run() error = nil")
		}
		if poolCloses != 1 || listenCalls != 0 || serverCalls != 0 || coordinateCalls != 0 {
			t.Fatalf(
				"pool/listen/server/coordinate calls = %d/%d/%d/%d, want 1/0/0/0",
				poolCloses,
				listenCalls,
				serverCalls,
				coordinateCalls,
			)
		}
	})
}

func TestRun_ComposedInvalidConfigHasNoSideEffects(t *testing.T) {
	values := validCompositionEnvironment()
	values[loginObservationBudgetEnvironment] = ""
	var calls []string
	dependencies := validCompositionDependencies(values)
	dependencies.parseDatabase = func(string) (*pgxpool.Config, error) {
		calls = append(calls, "parse-database")
		return new(pgxpool.Config), nil
	}
	dependencies.connect = func(context.Context, *pgxpool.Config) (*pgxpool.Pool, error) {
		calls = append(calls, "connect")
		return new(pgxpool.Pool), nil
	}
	dependencies.listen = func(string, string) (net.Listener, error) {
		calls = append(calls, "listen")
		return &recordingListener{}, nil
	}
	dependencies.composeBusiness = func(
		*pgxpool.Pool,
		businessComposition,
	) (http.Handler, error) {
		calls = append(calls, "compose-routes")
		return http.NotFoundHandler(), nil
	}

	if err := run(context.Background(), dependencies); err == nil {
		t.Fatal("run() error = nil")
	}
	if len(calls) != 0 {
		t.Fatalf("side effects = %v, want none", calls)
	}
}

func TestRun_ComposesJSONObserversRoutesAndDualBind(t *testing.T) {
	values := validCompositionEnvironment()
	var output bytes.Buffer
	var calls []string
	pool := new(pgxpool.Pool)
	businessListener := &recordingListener{}
	metricsListener := &recordingListener{}
	stopErr := errors.New("synthetic lifecycle stop")
	dependencies := validCompositionDependencies(values)
	dependencies.logOutput = &output
	dependencies.connect = func(context.Context, *pgxpool.Config) (*pgxpool.Pool, error) {
		calls = append(calls, "connect")
		return pool, nil
	}
	dependencies.ping = func(context.Context, *pgxpool.Pool) error {
		calls = append(calls, "ping")
		return nil
	}
	dependencies.migrate = func(context.Context, *pgxpool.Pool) error {
		calls = append(calls, "migrate")
		return nil
	}
	dependencies.composeBusiness = func(
		gotPool *pgxpool.Pool,
		composition businessComposition,
	) (http.Handler, error) {
		calls = append(calls, "compose")
		if gotPool != pool || composition.logger == nil ||
			composition.metrics == nil || composition.loginObserver == nil ||
			composition.originGuard == nil || composition.failureAuditor == nil {
			t.Fatal("business composition dependencies are incomplete")
		}
		composition.logger.Info("composition-test", slog.String("service", serviceName))
		return http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			response.WriteHeader(http.StatusNoContent)
		}), nil
	}
	dependencies.listen = func(_ string, address string) (net.Listener, error) {
		calls = append(calls, "listen:"+address)
		if address == values[httpAddressEnvironment] {
			return businessListener, nil
		}
		return metricsListener, nil
	}
	var businessHandler http.Handler
	var metricsHandler http.Handler
	dependencies.newServer = func(name string, handler http.Handler) lifecycleServer {
		calls = append(calls, "server:"+name)
		if name == "business" {
			businessHandler = handler
		} else {
			metricsHandler = handler
		}
		return newLifecycleServer(name)
	}
	dependencies.coordinate = func(_ context.Context, config lifecycleConfig) error {
		calls = append(calls, "coordinate")
		if config.businessListener != businessListener ||
			config.metricsListener != metricsListener {
			t.Fatal("lifecycle did not receive both bound listeners")
		}
		return stopErr
	}

	err := run(context.Background(), dependencies)
	if !errors.Is(err, stopErr) {
		t.Fatalf("run() error = %v", err)
	}
	want := []string{
		"connect", "ping", "migrate", "compose",
		"listen:" + values[httpAddressEnvironment],
		"listen:" + values[metricsAddressEnvironment],
		"server:business", "server:metrics", "coordinate",
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
	if businessHandler == nil || metricsHandler == nil {
		t.Fatal("server handlers were not injected")
	}
	response := httptest.NewRecorder()
	metricsHandler.ServeHTTP(
		response,
		httptest.NewRequest(http.MethodGet, "/metrics", nil),
	)
	if response.Code != http.StatusOK {
		t.Fatalf("metrics status = %d", response.Code)
	}
	var record map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &record); err != nil {
		t.Fatalf("stdout log is not JSON Lines: %q", output.Bytes())
	}
}

func TestRun_AWIRELIFE04LIFE05SecondBindFailureDoesNotStartServers(t *testing.T) {
	values := validCompositionEnvironment()
	bindErr := errors.New("synthetic metrics bind failure")
	closed := 0
	dependencies := validCompositionDependencies(values)
	dependencies.listen = func(_ string, address string) (net.Listener, error) {
		if address == values[httpAddressEnvironment] {
			return &recordingListener{onClose: func() { closed++ }}, nil
		}
		return nil, bindErr
	}
	dependencies.newServer = func(string, http.Handler) lifecycleServer {
		t.Fatal("server constructed before both listeners bound")
		return nil
	}

	err := run(context.Background(), dependencies)
	if !errors.Is(err, bindErr) {
		t.Fatalf("run() error = %v", err)
	}
	if closed != 1 {
		t.Fatalf("business listener close calls = %d, want 1", closed)
	}
}

func TestCompositionMetricsAdapters(t *testing.T) {
	registry, err := platform.NewMetricsRegistry([]float64{0.1, 1})
	if err != nil {
		t.Fatalf("NewMetricsRegistry() error = %v", err)
	}
	if err := (loginMetricsAdapter{registry: registry}).RecordLogin(
		context.Background(),
		identity.LoginMetricObservation{
			Endpoint:     "POST /api/v1/sessions",
			Duration:     250 * time.Millisecond,
			Outcome:      identity.LoginOutcomeInternalFailure,
			ReasonBucket: "persistence",
			ErrorCode:    string(platform.CodePlatformDatabaseUnavailable),
			FailureStage: identity.LoginFailureStageTransactionCommit,
		},
	); err != nil {
		t.Fatalf("RecordLogin() error = %v", err)
	}
	if err := (originMetricsAdapter{registry: registry}).RecordOriginRejection(
		context.Background(),
		platform.OriginRejectionObservation{
			Reason:        platform.OriginRejectionOriginMissing,
			ResponseWrite: platform.OriginResponseWriteOK,
		},
	); err != nil {
		t.Fatalf("RecordOriginRejection() error = %v", err)
	}
	response := httptest.NewRecorder()
	platform.NewMetricsHandler(registry).ServeHTTP(
		response,
		httptest.NewRequest(http.MethodGet, "/metrics", nil),
	)
	for _, name := range []string{
		"han_identity_login_attempts_total",
		"han_identity_login_internal_failures_total",
		"han_identity_login_duration_seconds",
		"han_platform_origin_rejections_total",
	} {
		if !strings.Contains(response.Body.String(), name) {
			t.Fatalf("metrics body missing %q", name)
		}
	}
}

func validCompositionEnvironment() map[string]string {
	values := validStartupDeploymentValues()
	values[articleTransferNoticeMaxAttemptsEnvironment] = "7"
	values[databaseDSNEnvironment] = "host=example.invalid user=han dbname=han"
	values[httpAddressEnvironment] = "127.0.0.1:8080"
	values[metricsAddressEnvironment] = "127.0.0.1:9090"
	values[loginObservationBudgetEnvironment] = "250ms"
	values[originObservationBudgetEnvironment] = "250ms"
	values[failureAuditBudgetEnvironment] = "2s"
	values[shutdownBudgetEnvironment] = "10s"
	values[smtpOperationBudgetEnvironment] = "2s"
	values[noticePollIntervalEnvironment] = "1s"
	values[noticeBatchLimitEnvironment] = "2"
	values[noticeLeaseDurationEnvironment] = "30s"
	values[noticeDBOperationBudgetEnvironment] = "1s"
	values[noticeBackoffInitialEnvironment] = "1s"
	values[noticeBackoffMaxEnvironment] = "8s"
	values[passwordResetWorkerLeaseDurationEnvironment] = "30s"
	values[passwordResetWorkerRetryBackoffEnvironment] = "1s"
	values[passwordResetWorkerMaxRetriesEnvironment] = "3"
	values[passwordResetWorkerClaimBatchLimitEnvironment] = "10"
	values[passwordResetWorkerConnectionBudgetEnvironment] = "2"
	values[articleQueryDefaultLimitEnvironment] = "2"
	values[articleQueryMaxLimitEnvironment] = "2"
	values[articleQueryCandidateBatchSizeEnvironment] = "3"
	values[loginDurationBucketsEnvironment] = "0.005,0.01,0.025,0.05,0.1,0.25,0.5,1,2.5,5"
	values[loginMaxBodyBytesEnvironment] = "16384"
	values[argon2MemoryEnvironment] = "19456"
	values[argon2IterationsEnvironment] = "2"
	values[argon2ParallelismEnvironment] = "1"
	values[argon2SaltBytesEnvironment] = "16"
	values[argon2KeyBytesEnvironment] = "32"
	values[argon2ProfileIDEnvironment] = "test-profile"
	values[argon2MaxConcurrencyEnvironment] = "1"
	values[loginBucketKeyVersionEnvironment] = "v1"
	values[loginBucketKeyPrefix+"v1"] = "0123456789abcdef0123456789abcdef"
	values[loginAbuseWindowEnvironment] = "15m"
	values[loginAbuseThresholdEnvironment] = "5"
	values[loginAbuseBlockEnvironment] = "15m"
	values[loginAbuseActionEnvironment] = "decay"
	values[loginAbuseDecayEnvironment] = "1"
	values[sessionTokenEntropyEnvironment] = "32"
	values[csrfTokenEntropyEnvironment] = "32"
	values[sessionAbsoluteTTLEnvironment] = "24h"
	values[sessionIdleTTLEnvironment] = "30m"
	values[sourceKeyVersionEnvironment] = "v1"
	values[sourceKeyPrefix+"v1"] = "abcdef0123456789abcdef0123456789"
	values[recoveryModeEnvironment] = "normal"
	noticeContent := "Approved administrator information notice."
	noticeDigest := sha256.Sum256([]byte(noticeContent))
	values["HAN_ADMINISTRATOR_INFORMATION_NOTICE_VERSION"] = "notice-2026-10-24"
	values["HAN_ADMINISTRATOR_INFORMATION_NOTICE_EFFECTIVE_AT"] = "2026-10-24T00:00:00Z"
	values["HAN_ADMINISTRATOR_INFORMATION_NOTICE_CONTENT"] = noticeContent
	values["HAN_ADMINISTRATOR_INFORMATION_NOTICE_SHA256"] = fmt.Sprintf("%x", noticeDigest)
	values["HAN_IDEMPOTENCY_CURRENT_KEY_VERSION"] = "v2"
	values["HAN_IDEMPOTENCY_RETAINED_KEY_VERSIONS"] = "v1"
	values["HAN_IDEMPOTENCY_HMAC_KEY_v1"] = "retained-idempotency-key-0000001"
	values["HAN_IDEMPOTENCY_HMAC_KEY_v2"] = "current-idempotency-key-00000002"
	values["HAN_IDEMPOTENCY_LOCK_DURATION"] = "2m"
	values["HAN_IDEMPOTENCY_RECORD_TTL"] = "24h"
	values["HAN_SMTP_PORT"] = "587"
	values["HAN_SMTP_FROM"] = "no-reply@example.invalid"
	values["HAN_SMTP_AUTH_MODE"] = "none"
	values["HAN_SMTP_USERNAME"] = ""
	values["HAN_SMTP_PASSWORD"] = ""
	values[articleDeletionJournalRootEnvironment] = filepath.Join(
		string(filepath.Separator),
		"synthetic",
		"article-journal",
	)
	values[articleBackupInventoryPathEnvironment] = filepath.Join(
		string(filepath.Separator),
		"synthetic",
		"article-backup-inventory.json",
	)
	values[articleDeletionJournalFileModeEnvironment] = "0600"
	values[articleDeletionCurrentKeyVersionEnvironment] = "v2"
	values[articleDeletionRetainedKeyVersionsEnvironment] = "v1"
	values[articleDeletionKeyPrefix+"v1"] = base64.RawURLEncoding.EncodeToString(
		bytes.Repeat([]byte{0x31}, 32),
	)
	values[articleDeletionKeyPrefix+"v2"] = base64.RawURLEncoding.EncodeToString(
		bytes.Repeat([]byte{0x32}, 32),
	)
	values[articleFinalizerWorkerIDEnvironment] = "han-server-test-1"
	values[articleFinalizerPollIntervalEnvironment] = "1s"
	values[articleFinalizerBatchSizeEnvironment] = "50"
	values[articleFinalizerLeaseDurationEnvironment] = "30s"
	values[articleFinalizerOperationBudgetEnvironment] = "10s"
	values[articleFinalizerBackoffInitialEnvironment] = "250ms"
	values[articleFinalizerBackoffMaxEnvironment] = "30s"
	values[articleFinalizerMaxAttemptsEnvironment] = "3"
	values[articleFinalizerShutdownBudgetEnvironment] = "10s"
	return values
}

func setAdministratorNoticeContent(values map[string]string, content string) {
	values[administratorNoticeContentEnvironment] = content
	digest := sha256.Sum256([]byte(content))
	values[administratorNoticeSHA256Environment] = fmt.Sprintf("%x", digest)
}

func validCompositionDependencies(values map[string]string) startupDependencies {
	return startupDependencies{
		getenv:            func(key string) string { return values[key] },
		lookupEnvironment: mapEnvironmentLookup(values),
		parseDatabase: func(string) (*pgxpool.Config, error) {
			config := new(pgxpool.Config)
			config.MaxConns = 8
			return config, nil
		},
		connect: func(context.Context, *pgxpool.Config) (*pgxpool.Pool, error) {
			return new(pgxpool.Pool), nil
		},
		ping:    func(context.Context, *pgxpool.Pool) error { return nil },
		migrate: func(context.Context, *pgxpool.Pool) error { return nil },
		bootstrapInitialAdmin: func(
			context.Context,
			*pgxpool.Pool,
			*preparedLoginDependencies,
		) error {
			return nil
		},
		listen: func(string, string) (net.Listener, error) {
			return &recordingListener{}, nil
		},
		closePool: func(*pgxpool.Pool) {},
		composeBusiness: func(
			*pgxpool.Pool,
			businessComposition,
		) (http.Handler, error) {
			return http.NotFoundHandler(), nil
		},
		newServer: func(name string, _ http.Handler) lifecycleServer {
			return newLifecycleServer(name)
		},
		signals:    make(chan os.Signal),
		coordinate: func(context.Context, lifecycleConfig) error { return nil },
		logOutput:  io.Discard,
	}
}

func assertClearedSecret(t *testing.T, retained []byte, canary string) {
	t.Helper()
	for _, value := range retained {
		if value != 0 {
			t.Fatalf("retained secret was not cleared; canary length = %d", len(canary))
		}
	}
}

func mustParsePrefixes(t *testing.T, values []string) []netip.Prefix {
	t.Helper()
	prefixes := make([]netip.Prefix, len(values))
	for index, value := range values {
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			t.Fatalf("ParsePrefix(%q) error = %v", value, err)
		}
		prefixes[index] = prefix
	}
	return prefixes
}

func mustLoadStartupConfig(t *testing.T, values map[string]string) startupConfig {
	t.Helper()
	config, err := loadStartupConfig(
		func(key string) string { return values[key] },
		mapEnvironmentLookup(values),
	)
	if err != nil {
		t.Fatalf("loadStartupConfig() error = %v", err)
	}
	return config
}

func productionBusinessComposition(
	t *testing.T,
	prepared *preparedLoginDependencies,
) businessComposition {
	t.Helper()
	registry, err := platform.NewMetricsRegistry([]float64{0.1, 1})
	if err != nil {
		t.Fatalf("NewMetricsRegistry() error = %v", err)
	}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	loginLogger, err := identity.NewSlogLoginEventLogger(logger, serviceName)
	if err != nil {
		t.Fatalf("NewSlogLoginEventLogger() error = %v", err)
	}
	loginObserver, err := identity.NewLoginOutcomeObserver(
		loginLogger,
		loginMetricsAdapter{registry: registry},
	)
	if err != nil {
		t.Fatalf("NewLoginOutcomeObserver() error = %v", err)
	}
	originLogger, err := platform.NewSlogOriginRejectionLogger(logger, serviceName)
	if err != nil {
		t.Fatalf("NewSlogOriginRejectionLogger() error = %v", err)
	}
	originObserver, err := platform.NewOriginRejectionObserver(
		originLogger,
		originMetricsAdapter{registry: registry},
	)
	if err != nil {
		t.Fatalf("NewOriginRejectionObserver() error = %v", err)
	}
	guard, err := platform.NewObservedOriginGuard(
		"https://community.example.invalid",
		originObserver,
		time.Second,
	)
	if err != nil {
		t.Fatalf("NewObservedOriginGuard() error = %v", err)
	}
	return businessComposition{
		noticeDispatcher: article.NoticeDispatcherConfig{PollInterval: time.Second, BatchLimit: 2,
			LeaseDuration: 10 * time.Second, DBOperationBudget: time.Second,
			BackoffInitial: time.Second, BackoffMax: 8 * time.Second, SMTPOperationBudget: time.Second},
		articleTransferNoticeMaxAttempts: 7,
		logger:                           logger,
		metrics:                          registry,
		loginObserver:                    loginObserver,
		originGuard:                      &guard,
		failureAuditor:                   productionFailureAuditorStub{},
		failureAuditFor:                  time.Second,
		loginObserveFor:                  time.Second,
		loginMaxBodyBytes:                4096,
		smtpOperationBudget:              time.Second,
		processingGate: platform.ProcessingGateConfig{
			RegionRef: "region-ref", ControllerRef: "controller-ref",
			LegalBasisRef: "legal-basis-ref", SMTPProcessorRef: "smtp-processor-ref",
		},
		prepared: prepared,
		passwordResetWorker: identity.PasswordResetWorkerConfig{
			LeaseDuration: time.Minute, RetryBackoff: time.Second,
			MaxRetries: 3, ClaimBatchLimit: 10, ConnectionBudget: 2,
		},
		articleQuery: article.QueryConfig{
			DefaultLimit: 2, MaxLimit: 2, CandidateBatchSize: 3,
		},
	}
}

func newArticleProductionHandlerWithTestSeam(
	t *testing.T,
	seam *articleProductionTestSeam,
	loggers ...*slog.Logger,
) http.Handler {
	t.Helper()
	config := mustLoadStartupConfig(t, validCompositionEnvironment())
	prepared, err := prepareProductionLoginDependencies(
		config,
		bytes.NewReader(bytes.Repeat([]byte{0x5a}, 128)),
	)
	clearStartupSecrets(&config)
	if err != nil {
		t.Fatalf("prepareProductionLoginDependencies() error = %v", err)
	}
	t.Cleanup(prepared.cleanup)
	composition := productionBusinessComposition(t, prepared)
	if len(loggers) > 0 {
		composition.logger = loggers[0]
	}
	composition.articleJournal = mainArticleJournalStub{}
	composition.articleDeletion = &articleDeletionStartupConfig{
		workerID: "article-worker-test",
		lease:    30 * time.Second,
		worker: article.DeletionWorkerConfig{
			PollInterval:      time.Second,
			BatchSize:         10,
			OperationBudget:   10 * time.Second,
			BackoffInitial:    250 * time.Millisecond,
			BackoffMax:        time.Second,
			MaxBatchesPerWake: 3,
			ShutdownBudget:    time.Second,
		},
	}
	composition.runtime = &businessRuntime{}
	composition.articleTestSeam = seam
	handler, err := newProductionBusinessHandler(new(pgxpool.Pool), composition)
	if err != nil {
		t.Fatalf("newProductionBusinessHandler() error = %v", err)
	}
	return handler
}

type articleSecurityResolverSpy struct {
	events    *[]string
	rejectAt  string
	principal session.Principal
}

type mainPasswordResetService struct {
	calls   int
	request identity.PasswordResetRequest
	result  identity.PasswordResetResult
	err     error
}

func (service *mainPasswordResetService) ResetPassword(
	_ context.Context,
	request identity.PasswordResetRequest,
) (identity.PasswordResetResult, error) {
	service.calls++
	service.request = request
	result := service.result
	if result.UserID == "" {
		result.UserID = request.TargetID
	}
	return result, service.err
}

type mainTemporaryCredentialResendBusinessProbe struct {
	principal        session.Principal
	beginCalls       int
	userLockCalls    int
	actorLookupCalls int
	resolveCalls     int
	secretCalls      int
	idCalls          int
	hashCalls        int
	mailCalls        int
	writeCalls       int
}

func newMainTemporaryCredentialResendBusinessService(
	t *testing.T,
	probe *mainTemporaryCredentialResendBusinessProbe,
) identity.TemporaryCredentialResendService {
	t.Helper()
	keyring := idempotency.DigestKeyring{
		ActiveVersion: "v1",
		Keys:          map[string][]byte{"v1": []byte("temporary-resend-test-key")},
	}
	clock := mainLoginClock{now: time.Date(2026, 10, 25, 10, 0, 0, 0, time.UTC)}
	passwordReset, err := identity.NewPasswordResetService(identity.PasswordResetDependencies{
		Transactions: probe, Repository: probe, Store: &identity.PasswordResetStore{},
		PasswordPolicy: security.PasswordPolicy{}, PasswordHasher: probe,
		Secrets: probe, Mailer: probe, IdempotencyKeys: keyring,
		IdempotencyLock: time.Minute, IdempotencyTTL: 24 * time.Hour,
		SMTPOperationBudget: time.Second, Audit: probe, IDs: probe, Clock: clock,
	})
	if err != nil {
		t.Fatalf("NewPasswordResetService() error = %v", err)
	}
	administratorCreate, err := identity.NewAdministratorCreateService(
		identity.AdministratorCreateDependencies{
			Transactions: probe, Repository: probe, InformationNotices: probe,
			Gate: probe, PasswordPolicy: security.PasswordPolicy{},
			PasswordHasher: probe, Mailer: probe,
			Idempotency: mainTemporaryCredentialResendIdempotencyProbe{
				probe: probe,
			},
			IdempotencyKeys: keyring, IdempotencyLock: time.Minute,
			IdempotencyTTL: 24 * time.Hour, SMTPOperationBudget: time.Second,
			Audit: probe, IDs: probe, Clock: clock,
		},
	)
	if err != nil {
		t.Fatalf("NewAdministratorCreateService() error = %v", err)
	}
	service, err := identity.NewTemporaryCredentialResendService(
		passwordReset,
		administratorCreate,
	)
	if err != nil {
		t.Fatalf("NewTemporaryCredentialResendService() error = %v", err)
	}
	return service
}

func (p *mainTemporaryCredentialResendBusinessProbe) Begin(
	context.Context,
) (pgx.Tx, error) {
	p.beginCalls++
	return &mainLoginTx{}, nil
}

func (p *mainTemporaryCredentialResendBusinessProbe) LockUser(
	_ context.Context,
	_ pgx.Tx,
	userID string,
) (identity.User, error) {
	p.userLockCalls++
	if userID == p.principal.UserID {
		completedAt := time.Date(2026, 10, 24, 10, 0, 0, 0, time.UTC)
		return identity.User{
			ID: userID, Role: identity.RoleAdmin,
			IsInitialAdmin:        p.principal.IsInitialAdmin,
			Status:                identity.UserStatusActive,
			CredentialMode:        identity.CredentialModePermanent,
			OnboardingCompletedAt: &completedAt,
		}, nil
	}
	email := "target@example.invalid"
	return identity.User{
		ID: userID, Email: &email, Role: identity.RoleMember,
		Status: identity.UserStatusActive, CredentialMode: identity.CredentialModePermanent,
	}, nil
}

func (*mainTemporaryCredentialResendBusinessProbe) LockLatestTemporaryCredential(
	context.Context,
	pgx.Tx,
	string,
) (identity.TemporaryCredential, error) {
	return identity.TemporaryCredential{}, pgx.ErrNoRows
}

func (p *mainTemporaryCredentialResendBusinessProbe) UpdateTemporaryCredential(
	context.Context,
	pgx.Tx,
	identity.TemporaryCredential,
) error {
	p.writeCalls++
	return nil
}

func (p *mainTemporaryCredentialResendBusinessProbe) InsertTemporaryCredential(
	context.Context,
	pgx.Tx,
	identity.TemporaryCredential,
) error {
	p.writeCalls++
	return nil
}

func (p *mainTemporaryCredentialResendBusinessProbe) SetCredentialMode(
	context.Context,
	pgx.Tx,
	string,
	identity.CredentialMode,
	time.Time,
) error {
	p.writeCalls++
	return nil
}

func (p *mainTemporaryCredentialResendBusinessProbe) RevokeSessions(
	context.Context,
	pgx.Tx,
	string,
	time.Time,
	string,
) error {
	p.writeCalls++
	return nil
}

func (p *mainTemporaryCredentialResendBusinessProbe) FindAdministratorCreationActor(
	_ context.Context,
	_ pgx.Tx,
	userID string,
) (identity.User, error) {
	p.actorLookupCalls++
	return identity.User{
		ID: userID, Role: identity.RoleAdmin,
		IsInitialAdmin: p.principal.IsInitialAdmin,
		Status:         identity.UserStatusActive,
		CredentialMode: identity.CredentialModePermanent,
	}, nil
}

func (p *mainTemporaryCredentialResendBusinessProbe) InsertUser(
	context.Context,
	pgx.Tx,
	identity.User,
) error {
	p.writeCalls++
	return nil
}

func (p *mainTemporaryCredentialResendBusinessProbe) InsertIdentifier(
	context.Context,
	pgx.Tx,
	identity.IdentityIdentifier,
) error {
	p.writeCalls++
	return nil
}

func (p *mainTemporaryCredentialResendBusinessProbe) InsertAdministratorInformationDeclaration(
	context.Context,
	pgx.Tx,
	identity.AdministratorInformationDeclaration,
) error {
	p.writeCalls++
	return nil
}

func (p *mainTemporaryCredentialResendBusinessProbe) ActivateTemporaryCredential(
	context.Context,
	pgx.Tx,
	string,
	time.Time,
) error {
	p.writeCalls++
	return nil
}

func (*mainTemporaryCredentialResendBusinessProbe) FindCreatedAdministrator(
	context.Context,
	pgx.Tx,
	string,
) (identity.CreatedAdministrator, error) {
	return identity.CreatedAdministrator{}, errors.New("unexpected created administrator read")
}

func (*mainTemporaryCredentialResendBusinessProbe) LockTemporaryCredentialResendTarget(
	context.Context,
	pgx.Tx,
	string,
) (identity.User, identity.TemporaryCredential, error) {
	return identity.User{}, identity.TemporaryCredential{}, pgx.ErrNoRows
}

func (*mainTemporaryCredentialResendBusinessProbe) CurrentAdministratorInformationNotice(
	context.Context,
	pgx.Tx,
) (identity.AdministratorInformationNotice, error) {
	return identity.AdministratorInformationNotice{}, nil
}

func (*mainTemporaryCredentialResendBusinessProbe) RequireOpen(context.Context) error {
	return nil
}

func (p *mainTemporaryCredentialResendBusinessProbe) Generate() (string, error) {
	p.secretCalls++
	return "Temporary123", nil
}

func (p *mainTemporaryCredentialResendBusinessProbe) Hash(string) (string, error) {
	p.hashCalls++
	return "digest", nil
}

func (*mainTemporaryCredentialResendBusinessProbe) Verify(
	string,
	string,
) (bool, bool, error) {
	return true, false, nil
}

func (p *mainTemporaryCredentialResendBusinessProbe) Send(
	context.Context,
	mailer.Message,
) error {
	p.mailCalls++
	return nil
}

func (p *mainTemporaryCredentialResendBusinessProbe) ResolveExisting(
	context.Context,
	idempotency.Transaction,
	idempotency.BeginRequest,
) (idempotency.BeginResult, bool, error) {
	p.resolveCalls++
	return idempotency.BeginResult{}, false, errors.New("authorization probe stop")
}

type mainTemporaryCredentialResendIdempotencyProbe struct {
	probe *mainTemporaryCredentialResendBusinessProbe
}

func (p mainTemporaryCredentialResendIdempotencyProbe) Begin(
	context.Context,
	idempotency.Transaction,
	idempotency.BeginRequest,
) (idempotency.BeginResult, error) {
	p.probe.writeCalls++
	return idempotency.BeginResult{}, errors.New("unexpected idempotency begin")
}

func (p mainTemporaryCredentialResendIdempotencyProbe) ResolveExisting(
	ctx context.Context,
	tx idempotency.Transaction,
	request idempotency.BeginRequest,
) (idempotency.BeginResult, bool, error) {
	return p.probe.ResolveExisting(ctx, tx, request)
}

func (p mainTemporaryCredentialResendIdempotencyProbe) Complete(
	context.Context,
	idempotency.Transaction,
	idempotency.Completion,
) error {
	p.probe.writeCalls++
	return nil
}

func (p mainTemporaryCredentialResendIdempotencyProbe) MarkUncertain(
	context.Context,
	idempotency.Transaction,
	idempotency.Transition,
) error {
	p.probe.writeCalls++
	return nil
}

func (p *mainTemporaryCredentialResendBusinessProbe) Append(
	context.Context,
	pgx.Tx,
	audit.Event,
) error {
	p.writeCalls++
	return nil
}

func (p *mainTemporaryCredentialResendBusinessProbe) NewID() (string, error) {
	p.idCalls++
	return "10000000-0000-4000-8000-000000000099", nil
}

type mainTemporaryCredentialResendService struct {
	calls   int
	request identity.TemporaryCredentialResendRequest
	result  identity.PasswordResetResult
	err     error
}

func (service *mainTemporaryCredentialResendService) ResendTemporaryCredential(
	_ context.Context,
	request identity.TemporaryCredentialResendRequest,
) (identity.PasswordResetResult, error) {
	service.calls++
	service.request = request
	result := service.result
	if result.UserID == "" {
		result.UserID = request.TargetID
	}
	return result, service.err
}

func activeAdministratorPrincipal() session.Principal {
	return session.Principal{
		SessionID: "session-1", UserID: "administrator-1",
		Role: "admin", Status: "active",
		AuthenticatedCredential: session.CredentialPermanent,
	}
}

func passwordResetProductionRequest(
	method string,
	path string,
	origin bool,
) *http.Request {
	request := httptest.NewRequest(method, path, nil)
	if origin {
		request.Header.Set("Origin", "https://community.example.invalid")
		request.Header.Set("Sec-Fetch-Site", "same-origin")
	}
	request.AddCookie(&http.Cookie{
		Name: platform.SessionCookieName, Value: "session-token",
	})
	request.Header.Set(platform.CSRFHeaderName, "csrf-token")
	request.Header.Set("Idempotency-Key", "password-reset-key")
	return request
}

func (resolver *articleSecurityResolverSpy) ResolvePrincipal(
	_ context.Context,
	_ session.ResolveInput,
) (session.Principal, error) {
	appendEvent := func(event string) {
		if resolver.events != nil {
			*resolver.events = append(*resolver.events, event)
		}
	}
	appendEvent("session")
	if resolver.rejectAt == "session" {
		return session.Principal{}, session.ErrSessionRequired
	}
	appendEvent("csrf")
	if resolver.rejectAt == "csrf" {
		return session.Principal{}, session.ErrCSRFFailed
	}
	appendEvent("principal")
	return resolver.principal, nil
}

type productionFailureAuditorStub struct{}

func (productionFailureAuditorStub) RecordOnce(
	context.Context,
	identity.LoginFailureAuditRequest,
) (identity.LoginAuditPersistence, error) {
	return identity.LoginAuditPersistedNo, nil
}

type mainAdministratorNoticeService struct {
	reader      identity.AdministratorInformationNoticeReader
	tx          pgx.Tx
	noticeCalls int
}

func (s *mainAdministratorNoticeService) CurrentAdministratorInformationNotice(
	ctx context.Context,
	_ session.Principal,
) (identity.AdministratorInformationNotice, error) {
	s.noticeCalls++
	return s.reader.CurrentAdministratorInformationNotice(ctx, s.tx)
}

func (*mainAdministratorNoticeService) CreateAdministrator(
	context.Context,
	identity.CreateAdministratorRequest,
) (identity.CreatedAdministrator, error) {
	return identity.CreatedAdministrator{}, errors.New("unexpected administrator creation")
}

type mainAdministratorPrivacyService struct{}

func (mainAdministratorPrivacyService) AcknowledgePrivacy(
	context.Context,
	identity.AdministratorPrivacyAcknowledgementRequest,
) (identity.AdministratorOnboardingResult, error) {
	return identity.AdministratorOnboardingResult{}, errors.New(
		"unexpected privacy acknowledgement",
	)
}

type mainAdministratorResolver struct{}

func (mainAdministratorResolver) ResolvePrincipal(
	context.Context,
	session.ResolveInput,
) (session.Principal, error) {
	return session.Principal{
		SessionID: "session-1", UserID: "initial-admin", Role: "admin",
		IsInitialAdmin: true, Status: "active",
		AuthenticatedCredential: session.CredentialPermanent,
	}, nil
}

type mainAdministratorRouteResolver struct {
	calls      int
	restricted bool
}

func (r *mainAdministratorRouteResolver) ResolvePrincipal(
	_ context.Context,
	input session.ResolveInput,
) (session.Principal, error) {
	r.calls++
	if input.RequireCSRF && input.CSRFToken != "csrf-token" {
		return session.Principal{}, session.ErrCSRFFailed
	}
	return session.Principal{
		SessionID: "session-1", UserID: "administrator-1", Role: "admin",
		IsInitialAdmin: true, Status: "active",
		MustChangePassword:      r.restricted,
		MustAcknowledgePrivacy:  r.restricted,
		AuthenticatedCredential: session.CredentialPermanent,
	}, nil
}

type mainErrorReader struct {
	err error
}

func (r mainErrorReader) Read([]byte) (int, error) {
	return 0, r.err
}

type mainNthReadErrorReader struct {
	readCalls int
	failAt    int
	err       error
	next      byte
}

func (r *mainNthReadErrorReader) Read(target []byte) (int, error) {
	r.readCalls++
	if r.readCalls == r.failAt {
		return 0, r.err
	}
	for index := range target {
		r.next++
		target[index] = r.next
	}
	return len(target), nil
}

type mainGateRandom struct {
	mu        sync.Mutex
	readCalls int
	next      byte
	block     *mainGateRandomBlock
}

type mainGateRandomBlock struct {
	entered chan struct{}
	release chan struct{}
}

func (r *mainGateRandom) Read(target []byte) (int, error) {
	r.mu.Lock()
	r.readCalls++
	block := r.block
	r.block = nil
	r.mu.Unlock()
	if block != nil {
		close(block.entered)
		<-block.release
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := range target {
		r.next++
		target[index] = r.next
	}
	return len(target), nil
}

func (r *mainGateRandom) Arm() (<-chan struct{}, chan<- struct{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.block != nil {
		panic("main gate random is already armed")
	}
	block := &mainGateRandomBlock{
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}
	r.block = block
	return block.entered, block.release
}

func (r *mainGateRandom) ReadCalls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.readCalls
}

type mainRehashBarrierHasher struct {
	delegate identity.LoginPasswordHasher
	digest   string
	onRehash func()
}

func (h *mainRehashBarrierHasher) Verify(
	password string,
	digest string,
) (bool, bool, error) {
	matched, needsRehash, err := h.delegate.Verify(password, digest)
	if err == nil && matched && needsRehash && digest == h.digest && h.onRehash != nil {
		h.onRehash()
	}
	return matched, needsRehash, err
}

func (h *mainRehashBarrierHasher) Hash(password string) (string, error) {
	return h.delegate.Hash(password)
}

func mainLoginUser(mode identity.CredentialMode, digest string) identity.User {
	var permanent *string
	if digest != "" {
		permanent = &digest
	}
	return identity.User{
		ID:                      "00000000-0000-4000-8000-000000000001",
		Username:                "Alice",
		PermanentPasswordDigest: permanent,
		Role:                    identity.RoleMember,
		Status:                  identity.UserStatusActive,
		CredentialMode:          mode,
	}
}

type mainLoginTransactor struct {
	transaction pgx.Tx
}

type mainPasswordResetGate struct{}

func (mainPasswordResetGate) CheckLockedUser(context.Context, pgx.Tx, string) error {
	return nil
}

func (t mainLoginTransactor) Begin(context.Context) (pgx.Tx, error) {
	return t.transaction, nil
}

type mainLoginTx struct {
	commitCalls   int
	rollbackCalls int
	closed        bool
}

func (tx *mainLoginTx) Begin(context.Context) (pgx.Tx, error) {
	return nil, errors.New("nested transaction is not supported")
}

func (tx *mainLoginTx) Commit(context.Context) error {
	tx.commitCalls++
	if tx.closed {
		return pgx.ErrTxClosed
	}
	tx.closed = true
	return nil
}

func (tx *mainLoginTx) Rollback(context.Context) error {
	tx.rollbackCalls++
	if tx.closed {
		return pgx.ErrTxClosed
	}
	tx.closed = true
	return nil
}

func (tx *mainLoginTx) CopyFrom(
	context.Context,
	pgx.Identifier,
	[]string,
	pgx.CopyFromSource,
) (int64, error) {
	return 0, errors.New("not implemented")
}

func (tx *mainLoginTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (tx *mainLoginTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (tx *mainLoginTx) Prepare(
	context.Context,
	string,
	string,
) (*pgconn.StatementDescription, error) {
	return nil, errors.New("not implemented")
}
func (tx *mainLoginTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("not implemented")
}
func (tx *mainLoginTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}
func (tx *mainLoginTx) QueryRow(context.Context, string, ...any) pgx.Row {
	return mainLoginRow{err: errors.New("not implemented")}
}
func (tx *mainLoginTx) Conn() *pgx.Conn { return nil }

type mainLoginRow struct {
	err error
}

func (r mainLoginRow) Scan(...any) error { return r.err }

type mainLoginRepository struct {
	user              identity.User
	userErr           error
	temporary         *identity.TemporaryCredential
	permanentRehashes int
	temporaryRehashes int
}

func (r *mainLoginRepository) FindUserByUsername(
	context.Context,
	pgx.Tx,
	string,
) (identity.User, error) {
	return r.user, r.userErr
}

func (r *mainLoginRepository) FindLatestTemporaryCredential(
	context.Context,
	pgx.Tx,
	string,
) (identity.TemporaryCredential, error) {
	if r.temporary == nil {
		return identity.TemporaryCredential{}, pgx.ErrNoRows
	}
	return *r.temporary, nil
}

func (r *mainLoginRepository) UpdatePermanentPasswordDigest(
	context.Context,
	pgx.Tx,
	string,
	string,
) error {
	r.permanentRehashes++
	return nil
}

func (r *mainLoginRepository) UpdateTemporaryCredentialDigest(
	context.Context,
	pgx.Tx,
	string,
	string,
) error {
	r.temporaryRehashes++
	return nil
}

type mainLoginSessions struct {
	calls int
}

func (s *mainLoginSessions) Create(
	context.Context,
	session.Executor,
	session.CreateInput,
) (session.Credentials, error) {
	s.calls++
	return session.Credentials{}, errors.New("unexpected session creation")
}

func (s *mainLoginSessions) Rotate(
	context.Context,
	pgx.Tx,
	string,
	time.Time,
	string,
	session.CreateInput,
) (session.Credentials, error) {
	s.calls++
	return session.Credentials{}, errors.New("unexpected session rotation")
}

type mainLoginSessionIssuer struct{}

func (mainLoginSessionIssuer) Issue(
	identity.User,
	session.CredentialKind,
	*string,
) (session.CreateInput, error) {
	return session.CreateInput{}, errors.New("unexpected session issue")
}

type mainLoginAbuse struct {
	failureCalls int
	successCalls int
}

func (a *mainLoginAbuse) Check(
	context.Context,
	pgx.Tx,
	identity.LoginAttempt,
) (identity.LoginAbuseDecision, error) {
	return identity.LoginAbuseDecision{
		SubjectDigest:    []byte("synthetic-subject-digest"),
		DigestKeyVersion: "synthetic-v1",
	}, nil
}

func (a *mainLoginAbuse) RecordFailure(
	context.Context,
	pgx.Tx,
	identity.LoginAttempt,
) error {
	a.failureCalls++
	return nil
}

func (a *mainLoginAbuse) RecordSuccess(
	context.Context,
	pgx.Tx,
	identity.LoginAttempt,
) error {
	a.successCalls++
	return nil
}

type mainLoginRecoveryMode struct{}

func (mainLoginRecoveryMode) RecoveryBootstrapAllowed(context.Context) (bool, error) {
	return false, nil
}

type mainLoginAudit struct {
	calls int
}

func (a *mainLoginAudit) Append(context.Context, pgx.Tx, audit.Event) error {
	a.calls++
	return nil
}

type mainLoginIDs struct{}

func (mainLoginIDs) NewID() (string, error) {
	return "00000000-0000-4000-8000-000000000099", nil
}

type mainLoginClock struct {
	now time.Time
}

func (c mainLoginClock) Now() time.Time { return c.now }

type mainLoginFailureAuditor struct {
	calls int
}

func (a *mainLoginFailureAuditor) RecordOnce(
	context.Context,
	identity.LoginFailureAuditRequest,
) (identity.LoginAuditPersistence, error) {
	a.calls++
	return identity.LoginAuditPersistedNo, nil
}

type mainLoginObserver struct {
	calls int
	last  identity.LoginObservation
}

func (o *mainLoginObserver) ObserveFinal(
	_ context.Context,
	observation identity.LoginObservation,
) error {
	o.calls++
	o.last = observation
	return nil
}

type routerLayerCounters struct {
	origin  int
	attempt int
	service int
	metrics int
}

type routerOriginObserver struct {
	calls *int
}

func (o routerOriginObserver) ObserveRejected(
	context.Context,
	platform.OriginRejectionObservation,
) error {
	*o.calls++
	return nil
}

type routerLoginIDGenerator struct {
	calls *int
}

func (g *routerLoginIDGenerator) NewID() (string, error) {
	*g.calls++
	return "00000000-0000-4000-8000-000000000077", nil
}

type routerLoginClock struct{}

func (routerLoginClock) Now() time.Time {
	return time.Date(2026, time.October, 23, 9, 0, 0, 0, time.UTC)
}

type routerFailureAuditor struct{}

func (routerFailureAuditor) RecordOnce(
	context.Context,
	identity.LoginFailureAuditRequest,
) (identity.LoginAuditPersistence, error) {
	return identity.LoginAuditPersistedNo, nil
}

type routerLoginService struct {
	calls *int
}

func (s routerLoginService) Login(
	context.Context,
	identity.LoginRequest,
) (identity.LoginResult, error) {
	*s.calls++
	return identity.LoginResult{}, platform.NewPublicError(
		platform.CodeAuthLoginFailed,
		errors.New("synthetic authentication failure"),
	)
}

type routerSourceProvider struct{}

func (routerSourceProvider) Evidence(
	*http.Request,
) (identity.RegistrationSourceEvidence, error) {
	return identity.RegistrationSourceEvidence{
		Kind: "network_source_hmac_sha256_v1", Fingerprint: "v1:synthetic",
	}, nil
}

type routerLoginObserver struct {
	calls *int
}

func (o routerLoginObserver) ObserveFinal(
	context.Context,
	identity.LoginObservation,
) error {
	*o.calls++
	return nil
}

type routerPasswordResolver struct {
	calls *int
}

func (r routerPasswordResolver) ResolvePrincipal(
	_ context.Context,
	input session.ResolveInput,
) (session.Principal, error) {
	*r.calls++
	if input.CSRFToken != "csrf-token" {
		return session.Principal{}, session.ErrCSRFFailed
	}
	return session.Principal{
		SessionID: "session-1", UserID: "user-1", Role: "admin",
		IsInitialAdmin: true, Status: "active",
		AuthenticatedCredential: session.CredentialPermanent,
	}, nil
}

func TestPasswordResetWorkerConfigFailsBeforeDatabaseParsing(t *testing.T) {
	keys := []string{
		passwordResetWorkerLeaseDurationEnvironment,
		passwordResetWorkerRetryBackoffEnvironment,
		passwordResetWorkerMaxRetriesEnvironment,
		passwordResetWorkerClaimBatchLimitEnvironment,
		passwordResetWorkerConnectionBudgetEnvironment,
	}
	invalid := []struct{ name, value string }{
		{"missing", ""}, {"blank", "   "}, {"parse", "not-a-number"},
		{"overflow", "999999999999999999999999999999999999999999999999h"},
		{"zero", "0"}, {"negative", "-1"}, {"noncanonical", "+1"},
	}
	for _, key := range keys {
		for _, test := range invalid {
			t.Run(key+"/"+test.name, func(t *testing.T) {
				values := validCompositionEnvironment()
				value := test.value
				if test.name == "missing" {
					delete(values, key)
				} else {
					if test.name == "overflow" &&
						key != passwordResetWorkerLeaseDurationEnvironment &&
						key != passwordResetWorkerRetryBackoffEnvironment {
						value = "999999999999999999999999999999999999999999999999"
					}
					values[key] = value
				}
				dependencies := validCompositionDependencies(values)
				parseCalls := 0
				dependencies.parseDatabase = func(string) (*pgxpool.Config, error) { parseCalls++; return &pgxpool.Config{}, nil }
				err := run(context.Background(), dependencies)
				if err == nil {
					t.Fatal("run() succeeded")
				}
				if !strings.Contains(err.Error(), key) {
					t.Fatalf("error = %v, want key %s", err, key)
				}
				if value != "" && strings.Contains(err.Error(), value) {
					t.Fatalf("error disclosed invalid value: %v", err)
				}
				if parseCalls != 0 {
					t.Fatalf("database parse calls = %d, want 0", parseCalls)
				}
			})
		}
	}
}

func TestPasswordResetWorkerConnectionBudgetConfigFailsAfterParseBeforeConnect(t *testing.T) {
	values := validCompositionEnvironment()
	values[passwordResetWorkerConnectionBudgetEnvironment] = "3"
	dependencies := validCompositionDependencies(values)
	parseCalls := 0
	connectCalls := 0
	dependencies.parseDatabase = func(string) (*pgxpool.Config, error) {
		parseCalls++
		config := &pgxpool.Config{}
		config.MaxConns = 2
		return config, nil
	}
	dependencies.connect = func(context.Context, *pgxpool.Config) (*pgxpool.Pool, error) {
		connectCalls++
		return new(pgxpool.Pool), nil
	}
	err := run(context.Background(), dependencies)
	if err == nil || !strings.Contains(err.Error(), passwordResetWorkerConnectionBudgetEnvironment) {
		t.Fatalf("run() error = %v", err)
	}
	if parseCalls != 1 || connectCalls != 0 {
		t.Fatalf("parse/connect calls = %d/%d, want 1/0", parseCalls, connectCalls)
	}
}

func TestPasswordResetProductionWiringReusesGateAndConstructsOneOwner(t *testing.T) {
	config := mustLoadStartupConfig(t, validCompositionEnvironment())
	prepared, err := prepareProductionLoginDependencies(config, bytes.NewReader(bytes.Repeat([]byte{0x71}, 128)))
	clearStartupSecrets(&config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(prepared.cleanup)
	composition := productionBusinessComposition(t, prepared)
	composition.passwordResetWorker = config.passwordResetWorker
	composition.runtime = &businessRuntime{}
	observed := map[string][]any{}
	composition.articleTestSeam = &articleProductionTestSeam{shared: func(value articleSharedDependency) {
		observed[value.Kind] = append(observed[value.Kind], value.Instance)
	}}
	builder, err := newProductionBusinessBuilder(new(pgxpool.Pool), composition)
	if err != nil {
		t.Fatalf("newProductionBusinessBuilder() error = %v", err)
	}
	if builder.passwordResetGate == nil || builder.passwordResetOperator == nil || builder.passwordResetWorker == nil || builder.passwordResetPermit == nil {
		t.Fatal("password-reset production dependencies are incomplete")
	}
	for _, kind := range []string{"password_reset_gate", "password_reset_operator_store", "password_reset_worker_store"} {
		instances := observed[kind]
		if len(instances) == 0 {
			t.Fatalf("missing observation %s", kind)
		}
		for _, instance := range instances {
			if instance != builder.passwordResetGate {
				t.Fatalf("%s instance = %p, gate = %p", kind, instance, builder.passwordResetGate)
			}
		}
	}
	if len(observed["password_reset_operator"]) != 1 || len(observed["password_reset_worker"]) != 1 || len(observed["password_reset_permit"]) != 1 {
		t.Fatalf("construction observations = operator:%d worker:%d permit:%d", len(observed["password_reset_operator"]), len(observed["password_reset_worker"]), len(observed["password_reset_permit"]))
	}
}

func TestProductionTemporaryCredentialResendCompositionRoutes(t *testing.T) {
	expiresAt := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		path        string
		wantPurpose identity.TemporaryCredentialResendPurpose
		wantRole    identity.Role
	}{
		{
			name: "password reset",
			path: "/api/v1/admin/users/" +
				"10000000-0000-4000-8000-000000000001/password-reset/resend",
			wantPurpose: identity.TemporaryCredentialResendPasswordReset,
			wantRole:    identity.RoleMember,
		},
		{
			name: "new administrator",
			path: "/api/v1/admin/administrators/" +
				"10000000-0000-4000-8000-000000000002/temporary-password/resend",
			wantPurpose: identity.TemporaryCredentialResendNewAdministrator,
			wantRole:    identity.RoleAdmin,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &mainTemporaryCredentialResendService{
				result: identity.PasswordResetResult{
					TemporaryCredentialExpiresAt: expiresAt,
					DeliveryStatus:               "accepted",
				},
			}
			handler := newArticleProductionHandlerWithTestSeam(
				t,
				&articleProductionTestSeam{
					principalResolver: &articleSecurityResolverSpy{
						principal: activeAdministratorPrincipal(),
					},
					temporaryCredentialResend: service,
				},
			)
			request := passwordResetProductionRequest(http.MethodPost, test.path, true)
			request.Header.Set("X-Temporary-Credential-Purpose", "attacker")
			request.URL.RawQuery = "purpose=attacker"
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != http.StatusOK || service.calls != 1 {
				t.Fatalf("status/calls = %d/%d; body=%s",
					response.Code, service.calls, response.Body.String())
			}
			if service.request.Purpose != test.wantPurpose ||
				service.request.TargetRole != test.wantRole ||
				service.request.IdempotencyKey != "password-reset-key" {
				t.Fatalf("resend request = %#v", service.request)
			}
			if strings.Contains(response.Body.String(), "temporary-password-canary") ||
				strings.Contains(response.Body.String(), "temporary_password") {
				t.Fatalf("resend response leaked secret: %s", response.Body.String())
			}
		})
	}
}

func TestProductionTemporaryCredentialResendBusinessAuthorizationAndPurposeIsolation(
	t *testing.T,
) {
	member := session.Principal{
		SessionID: "session-member",
		UserID:    "30000000-0000-4000-8000-000000000001",
		Role:      string(identity.RoleMember), Status: string(identity.UserStatusActive),
		AuthenticatedCredential: session.CredentialPermanent,
	}
	administrator := activeAdministratorPrincipal()
	administrator.UserID = "30000000-0000-4000-8000-000000000002"
	initialAdministrator := activeAdministratorPrincipal()
	initialAdministrator.UserID = "30000000-0000-4000-8000-000000000003"
	initialAdministrator.IsInitialAdmin = true
	tests := []struct {
		name             string
		path             string
		principal        session.Principal
		tamperedPurpose  string
		wantStatus       int
		wantCode         platform.Code
		wantBegins       int
		wantUserLocks    int
		wantActorLookups int
		wantResolves     int
	}{
		{
			name: "member cannot resend password reset",
			path: "/api/v1/admin/users/" +
				"10000000-0000-4000-8000-000000000001/password-reset/resend",
			principal: member, tamperedPurpose: "new_administrator",
			wantStatus: http.StatusForbidden, wantCode: platform.CodeAuthForbidden,
		},
		{
			name: "member cannot resend administrator credential",
			path: "/api/v1/admin/administrators/" +
				"10000000-0000-4000-8000-000000000002/temporary-password/resend",
			principal: member, tamperedPurpose: "password_reset",
			wantStatus: http.StatusForbidden, wantCode: platform.CodeAuthForbidden,
		},
		{
			name: "current administrator reaches password reset authority",
			path: "/api/v1/admin/users/" +
				"10000000-0000-4000-8000-000000000001/password-reset/resend",
			principal: administrator, tamperedPurpose: "new_administrator",
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   platform.CodePlatformDatabaseUnavailable,
			wantBegins: 1, wantUserLocks: 2,
		},
		{
			name: "ordinary administrator cannot select new administrator purpose",
			path: "/api/v1/admin/administrators/" +
				"10000000-0000-4000-8000-000000000002/temporary-password/resend",
			principal: administrator, tamperedPurpose: "password_reset",
			wantStatus: http.StatusForbidden, wantCode: platform.CodeAuthForbidden,
		},
		{
			name: "initial administrator reaches password reset authority",
			path: "/api/v1/admin/users/" +
				"10000000-0000-4000-8000-000000000001/password-reset/resend",
			principal: initialAdministrator, tamperedPurpose: "new_administrator",
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   platform.CodePlatformDatabaseUnavailable,
			wantBegins: 1, wantUserLocks: 2,
		},
		{
			name: "initial administrator reaches new administrator authority",
			path: "/api/v1/admin/administrators/" +
				"10000000-0000-4000-8000-000000000002/temporary-password/resend",
			principal: initialAdministrator, tamperedPurpose: "password_reset",
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   platform.CodePlatformDatabaseUnavailable,
			wantBegins: 1, wantActorLookups: 1, wantResolves: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			probe := &mainTemporaryCredentialResendBusinessProbe{
				principal: test.principal,
			}
			service := newMainTemporaryCredentialResendBusinessService(t, probe)
			handler := newArticleProductionHandlerWithTestSeam(
				t,
				&articleProductionTestSeam{
					principalResolver: &articleSecurityResolverSpy{
						principal: test.principal,
					},
					temporaryCredentialResend: service,
				},
			)
			request := passwordResetProductionRequest(http.MethodPost, test.path, true)
			request.Header.Set("X-Temporary-Credential-Purpose", test.tamperedPurpose)
			request.URL.RawQuery = "purpose=" + test.tamperedPurpose
			request.Body = io.NopCloser(strings.NewReader(
				`{"purpose":"` + test.tamperedPurpose + `"}`,
			))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			var envelope platform.Envelope
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			if response.Code != test.wantStatus || envelope.Code != test.wantCode {
				t.Fatalf("status/code = %d/%q, want %d/%q; body=%s",
					response.Code, envelope.Code, test.wantStatus, test.wantCode,
					response.Body.String())
			}
			if probe.beginCalls != test.wantBegins ||
				probe.userLockCalls != test.wantUserLocks ||
				probe.actorLookupCalls != test.wantActorLookups ||
				probe.resolveCalls != test.wantResolves {
				t.Fatalf(
					"authority calls begin/user-lock/actor/resolve = %d/%d/%d/%d, "+
						"want %d/%d/%d/%d",
					probe.beginCalls, probe.userLockCalls, probe.actorLookupCalls,
					probe.resolveCalls, test.wantBegins, test.wantUserLocks,
					test.wantActorLookups, test.wantResolves,
				)
			}
			if probe.secretCalls != 0 || probe.idCalls != 0 ||
				probe.hashCalls != 0 || probe.mailCalls != 0 ||
				probe.writeCalls != 0 {
				t.Fatalf(
					"business side effects secret/id/hash/mail/write = %d/%d/%d/%d/%d",
					probe.secretCalls, probe.idCalls, probe.hashCalls,
					probe.mailCalls, probe.writeCalls,
				)
			}
		})
	}
}

func TestProductionTemporaryCredentialResendOriginAuthorizationMethodNotFound(t *testing.T) {
	path := "/api/v1/admin/users/" +
		"10000000-0000-4000-8000-000000000001/password-reset/resend"
	tests := []struct {
		name       string
		request    func() *http.Request
		resolver   *articleSecurityResolverSpy
		wantStatus int
		wantCode   platform.Code
		wantAllow  string
	}{
		{
			name: "origin",
			request: func() *http.Request {
				return passwordResetProductionRequest(http.MethodPost, path, false)
			},
			resolver: &articleSecurityResolverSpy{
				principal: activeAdministratorPrincipal(),
			},
			wantStatus: http.StatusForbidden, wantCode: platform.CodeAuthOriginRejected,
		},
		{
			name: "authorization",
			request: func() *http.Request {
				return passwordResetProductionRequest(http.MethodPost, path, true)
			},
			resolver: &articleSecurityResolverSpy{
				principal: session.Principal{
					SessionID: "session-1", UserID: "administrator-1",
					Role: "admin", Status: "active", MustChangePassword: true,
					AuthenticatedCredential: session.CredentialPermanent,
				},
			},
			wantStatus: http.StatusForbidden,
			wantCode:   platform.CodeAuthMustCompleteOnboarding,
		},
		{
			name: "missing idempotency key",
			request: func() *http.Request {
				request := passwordResetProductionRequest(http.MethodPost, path, true)
				request.Header.Del("Idempotency-Key")
				return request
			},
			resolver: &articleSecurityResolverSpy{
				principal: activeAdministratorPrincipal(),
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   platform.CodeIdentityIdempotencyKeyRequired,
		},
		{
			name: "method",
			request: func() *http.Request {
				return passwordResetProductionRequest(http.MethodGet, path, true)
			},
			resolver: &articleSecurityResolverSpy{
				principal: activeAdministratorPrincipal(),
			},
			wantStatus: http.StatusMethodNotAllowed,
			wantCode:   platform.CodeRequestMethodNotAllowed,
			wantAllow:  http.MethodPost,
		},
		{
			name: "unknown path",
			request: func() *http.Request {
				return passwordResetProductionRequest(
					http.MethodPost,
					path+"/extra",
					true,
				)
			},
			resolver: &articleSecurityResolverSpy{
				principal: activeAdministratorPrincipal(),
			},
			wantStatus: http.StatusNotFound,
			wantCode:   platform.CodeRequestRouteNotFound,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &mainTemporaryCredentialResendService{}
			handler := newArticleProductionHandlerWithTestSeam(
				t,
				&articleProductionTestSeam{
					principalResolver:         test.resolver,
					temporaryCredentialResend: service,
				},
			)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, test.request())

			if response.Code != test.wantStatus || service.calls != 0 {
				t.Fatalf("status/calls = %d/%d, want %d/0; body=%s",
					response.Code, service.calls, test.wantStatus, response.Body.String())
			}
			if response.Header().Get("Allow") != test.wantAllow {
				t.Fatalf("Allow = %q, want %q",
					response.Header().Get("Allow"), test.wantAllow)
			}
			var envelope platform.Envelope
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			if envelope.Code != test.wantCode {
				t.Fatalf("code = %q, want %q", envelope.Code, test.wantCode)
			}
		})
	}
}

func TestProductionTemporaryCredentialResendWiringReusesCoordinators(t *testing.T) {
	var observations []articleSharedDependency
	_ = newArticleProductionHandlerWithTestSeam(t, &articleProductionTestSeam{
		shared: func(observation articleSharedDependency) {
			if observation.Kind == "password_reset_coordinator" ||
				observation.Kind == "administrator_create_coordinator" {
				observations = append(observations, observation)
			}
		},
	})
	byKind := map[string][]articleSharedDependency{}
	for _, observation := range observations {
		byKind[observation.Kind] = append(byKind[observation.Kind], observation)
	}
	for _, kind := range []string{
		"password_reset_coordinator",
		"administrator_create_coordinator",
	} {
		entries := byKind[kind]
		if len(entries) != 2 || entries[0].Instance != entries[1].Instance {
			t.Fatalf("%s observations = %#v, want two consumers sharing one instance",
				kind, entries)
		}
	}
}

func TestArticleQueryConfig_FieldParserUsesStrictSigned32BitContract(t *testing.T) {
	keys := []string{
		articleQueryDefaultLimitEnvironment,
		articleQueryMaxLimitEnvironment,
		articleQueryCandidateBatchSizeEnvironment,
	}
	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			for _, value := range []string{"1", "2147483647"} {
				got, err := parseArticleQueryConfigField(
					mapEnvironmentLookup(map[string]string{key: value}),
					key,
				)
				if err != nil {
					t.Fatalf("parseArticleQueryConfigField(%q) error = %v", value, err)
				}
				want, _ := strconv.Atoi(value)
				if got != want {
					t.Fatalf("parseArticleQueryConfigField(%q) = %d, want %d", value, got, want)
				}
			}

			tests := []struct {
				name     string
				values   map[string]string
				category string
			}{
				{name: "missing", values: map[string]string{}, category: "missing"},
				{name: "empty", values: map[string]string{key: ""}, category: "empty"},
				{name: "whitespace", values: map[string]string{key: " 1"}, category: "invalid_format"},
				{name: "trailing whitespace", values: map[string]string{key: "1 "}, category: "invalid_format"},
				{name: "plus", values: map[string]string{key: "+1"}, category: "invalid_format"},
				{name: "minus", values: map[string]string{key: "-1"}, category: "invalid_format"},
				{name: "decimal", values: map[string]string{key: "1.0"}, category: "invalid_format"},
				{name: "exponent", values: map[string]string{key: "1e2"}, category: "invalid_format"},
				{name: "separator", values: map[string]string{key: "1,000"}, category: "invalid_format"},
				{name: "unicode digit", values: map[string]string{key: "１"}, category: "invalid_format"},
				{name: "leading zero", values: map[string]string{key: "01"}, category: "invalid_format"},
				{name: "zero", values: map[string]string{key: "0"}, category: "invalid_format"},
				{name: "letters", values: map[string]string{key: "one"}, category: "invalid_format"},
				{name: "out of range", values: map[string]string{key: "2147483648"}, category: "out_of_range"},
			}
			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					_, err := parseArticleQueryConfigField(
						mapEnvironmentLookup(test.values),
						key,
					)
					requireArticleQueryConfigError(t, err, key, test.category)
				})
			}
		})
	}
}

func TestArticleQueryConfig_CompleteTupleAndCrossConstraints(t *testing.T) {
	tests := []struct {
		name      string
		values    [3]string
		want      article.QueryConfig
		wantError bool
	}{
		{
			name:   "lower valid",
			values: [3]string{"1", "1", "2"},
			want: article.QueryConfig{
				DefaultLimit: 1, MaxLimit: 1, CandidateBatchSize: 2,
			},
		},
		{
			name:   "upper valid",
			values: [3]string{"1", "2147483646", "2147483647"},
			want: article.QueryConfig{
				DefaultLimit:       1,
				MaxLimit:           2147483646,
				CandidateBatchSize: 2147483647,
			},
		},
		{
			name: "default above max",
			values: [3]string{
				"2", "1", "3",
			},
			wantError: true,
		},
		{
			name:      "candidate equals max",
			values:    [3]string{"1", "2", "2"},
			wantError: true,
		},
		{
			name: "default maximum parses before cross constraint",
			values: [3]string{
				"2147483647", "2147483647", "2147483647",
			},
			wantError: true,
		},
		{
			name: "max maximum parses before cross constraint",
			values: [3]string{
				"1", "2147483647", "2147483647",
			},
			wantError: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := map[string]string{
				articleQueryDefaultLimitEnvironment:       test.values[0],
				articleQueryMaxLimitEnvironment:           test.values[1],
				articleQueryCandidateBatchSizeEnvironment: test.values[2],
			}
			got, err := loadArticleQueryConfig(mapEnvironmentLookup(values))
			if test.wantError {
				requireArticleQueryConfigError(t, err, "", "cross_constraint")
				return
			}
			if err != nil {
				t.Fatalf("loadArticleQueryConfig() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("loadArticleQueryConfig() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestRun_ArticleQueryConfigFailsBeforeDatabaseSideEffects(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		category string
	}{
		{
			name: "invalid format",
			key:  articleQueryDefaultLimitEnvironment, value: "query-value-canary",
			category: "invalid_format",
		},
		{
			name: "out of range",
			key:  articleQueryMaxLimitEnvironment, value: "2147483648",
			category: "out_of_range",
		},
		{
			name: "cross constraint",
			key:  articleQueryCandidateBatchSizeEnvironment, value: "2",
			category: "cross_constraint",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := validCompositionEnvironment()
			values[test.key] = test.value
			var calls []string
			dependencies := validCompositionDependencies(values)
			dependencies.parseDatabase = func(string) (*pgxpool.Config, error) {
				calls = append(calls, "parse")
				return nil, errors.New("must not parse")
			}
			dependencies.connect = func(
				context.Context,
				*pgxpool.Config,
			) (*pgxpool.Pool, error) {
				calls = append(calls, "connect")
				return nil, errors.New("must not connect")
			}
			dependencies.ping = func(context.Context, *pgxpool.Pool) error {
				calls = append(calls, "ping")
				return errors.New("must not ping")
			}
			dependencies.migrate = func(context.Context, *pgxpool.Pool) error {
				calls = append(calls, "migrate")
				return errors.New("must not migrate")
			}
			dependencies.composeBusiness = func(
				*pgxpool.Pool,
				businessComposition,
			) (http.Handler, error) {
				calls = append(calls, "compose")
				return nil, errors.New("must not compose")
			}

			err := run(context.Background(), dependencies)

			errorKey := test.key
			if test.category == "cross_constraint" {
				errorKey = ""
			}
			requireArticleQueryConfigError(t, err, errorKey, test.category)
			if len(calls) != 0 {
				t.Fatalf("database/composition calls = %v, want none", calls)
			}
			if strings.Contains(err.Error(), test.value) {
				t.Fatalf("startup error leaked raw value: %v", err)
			}
		})
	}
}

func TestProductionArticleQueryCompositionUsesSharedPoolAndComponents(t *testing.T) {
	config := mustLoadStartupConfig(t, validCompositionEnvironment())
	prepared, err := prepareProductionLoginDependencies(
		config,
		bytes.NewReader(bytes.Repeat([]byte{0x5a}, 128)),
	)
	clearStartupSecrets(&config)
	if err != nil {
		t.Fatalf("prepareProductionLoginDependencies() error = %v", err)
	}
	t.Cleanup(prepared.cleanup)
	pool := new(pgxpool.Pool)
	var observations []articleSharedDependency
	composition := productionBusinessComposition(t, prepared)
	composition.articleTestSeam = &articleProductionTestSeam{
		shared: func(observation articleSharedDependency) {
			observations = append(observations, observation)
		},
	}

	handler, err := newProductionBusinessHandler(pool, composition)

	if err != nil {
		t.Fatalf("newProductionBusinessHandler() error = %v", err)
	}
	if handler == nil {
		t.Fatal("newProductionBusinessHandler() handler = nil")
	}
	required := map[string]bool{
		"article_query_repository": false,
		"article_author_reader":    false,
		"article_query_service":    false,
		"article_query_handler":    false,
	}
	poolConsumers := map[string]bool{
		"article_query_repository": false,
		"article_author_reader":    false,
	}
	for _, observation := range observations {
		if _, exists := required[observation.Kind]; exists {
			if observation.Instance == nil {
				t.Fatalf("%s instance = nil", observation.Kind)
			}
			required[observation.Kind] = true
		}
		if observation.Kind == "pool" {
			if _, exists := poolConsumers[observation.Consumer]; exists {
				if observation.Instance != pool {
					t.Fatalf("%s pool = %#v, want shared pool %#v",
						observation.Consumer, observation.Instance, pool)
				}
				poolConsumers[observation.Consumer] = true
			}
		}
	}
	for kind, found := range required {
		if !found {
			t.Fatalf("missing %s construction observation", kind)
		}
	}
	for consumer, found := range poolConsumers {
		if !found {
			t.Fatalf("missing shared pool consumer %s", consumer)
		}
	}
}

func TestProductionArticleQueryRoutesUseTraceSessionPrincipalAndCoexist(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantRoute  string
		wantStatus int
		wantAllow  string
	}{
		{
			name: "list", method: http.MethodGet, path: "/api/v1/articles",
			wantRoute: "list", wantStatus: http.StatusNoContent,
		},
		{
			name: "detail", method: http.MethodGet,
			path:      "/api/v1/articles/10000000-0000-4000-8000-000000000001",
			wantRoute: "detail", wantStatus: http.StatusNoContent,
		},
		{
			name: "delete coexists", method: http.MethodDelete,
			path:      "/api/v1/articles/10000000-0000-4000-8000-000000000001",
			wantRoute: "delete", wantStatus: http.StatusNoContent,
		},
		{
			name: "update coexists", method: http.MethodPut,
			path:      "/api/v1/articles/10000000-0000-4000-8000-000000000001",
			wantRoute: "update", wantStatus: http.StatusNoContent,
		},
		{
			name: "detail unsupported method", method: http.MethodPatch,
			path:       "/api/v1/articles/10000000-0000-4000-8000-000000000001",
			wantStatus: http.StatusMethodNotAllowed,
			wantAllow:  http.MethodGet + ", " + http.MethodPut + ", " + http.MethodDelete,
		},
		{
			name: "identity route coexists", method: http.MethodGet,
			path:       "/api/v1/admin/administrator-information-provision-notices/current",
			wantStatus: http.StatusForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var events []string
			var reached string
			seam := &articleProductionTestSeam{
				principalResolver: &articleQuerySecurityResolverSpy{
					events: &events,
					principal: session.Principal{
						SessionID: "session-1", UserID: "member-1",
						Role: "member", Status: "active",
					},
				},
				updateHandler: http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
					reached = "update"
					response.WriteHeader(http.StatusNoContent)
				}),
				trace: func(*http.Request) {
					events = append(events, "trace")
				},
				protectOrigin: func(next http.Handler) http.Handler {
					return next
				},
				handlers: [3]http.Handler{
					http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
						reached = "delete"
						response.WriteHeader(http.StatusNoContent)
					}),
				},
				queryHandlers: [2]http.Handler{
					http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
						reached = "list"
						events = append(events, "handler")
						response.WriteHeader(http.StatusNoContent)
					}),
					http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
						reached = "detail"
						events = append(events, "handler")
						response.WriteHeader(http.StatusNoContent)
					}),
				},
			}
			handler := newArticleProductionHandlerWithTestSeam(t, seam)
			request := httptest.NewRequest(test.method, test.path, nil)
			if test.name != "identity route coexists" {
				request.AddCookie(&http.Cookie{
					Name: platform.SessionCookieName, Value: "session-token",
				})
				request.Header.Set(platform.CSRFHeaderName, "csrf-token")
			}
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != test.wantStatus ||
				response.Header().Get("Allow") != test.wantAllow {
				t.Fatalf(
					"status/Allow = %d/%q, want %d/%q; body=%s",
					response.Code,
					response.Header().Get("Allow"),
					test.wantStatus,
					test.wantAllow,
					response.Body.String(),
				)
			}
			if reached != test.wantRoute {
				t.Fatalf("route = %q, want %q", reached, test.wantRoute)
			}
			if test.wantRoute == "list" || test.wantRoute == "detail" {
				wantEvents := []string{
					"trace", "session", "principal", "handler",
				}
				if !reflect.DeepEqual(events, wantEvents) {
					t.Fatalf("query security chain = %v, want %v", events, wantEvents)
				}
			}
			if response.Header().Get(platform.TraceHeader) == "" {
				t.Fatal("trace header is empty")
			}
		})
	}
}

type articleQuerySecurityResolverSpy struct {
	events    *[]string
	principal session.Principal
}

func (resolver *articleQuerySecurityResolverSpy) ResolvePrincipal(
	_ context.Context,
	input session.ResolveInput,
) (session.Principal, error) {
	*resolver.events = append(*resolver.events, "session")
	if input.RequireCSRF {
		*resolver.events = append(*resolver.events, "csrf")
	}
	*resolver.events = append(*resolver.events, "principal")
	return resolver.principal, nil
}

func requireArticleQueryConfigError(
	t *testing.T,
	err error,
	key string,
	category string,
) {
	t.Helper()
	if err == nil {
		t.Fatal("Article query configuration error = nil")
	}
	if !strings.Contains(err.Error(), "validate Article query configuration:") ||
		!strings.Contains(err.Error(), category) {
		t.Fatalf("Article query configuration error = %v, want category %q", err, category)
	}
	if key != "" && !strings.Contains(err.Error(), key) {
		t.Fatalf("Article query configuration error = %v, want key %q", err, key)
	}
}

// TestArticleReplayProductionWiring proves the unique composition root builds
// exactly one production source/provider and refuses to fall back to a stub.
func TestArticleReplayProductionWiring(t *testing.T) {
	t.Run("inventory path is mandatory", func(t *testing.T) {
		values := validCompositionEnvironment()
		delete(values, articleBackupInventoryPathEnvironment)
		if _, err := loadStartupConfig(
			func(key string) string { return values[key] },
			mapEnvironmentLookup(values),
		); err == nil {
			t.Fatal("missing backup inventory path accepted, want fail closed")
		}
	})

	t.Run("relative inventory path is rejected", func(t *testing.T) {
		values := validCompositionEnvironment()
		values[articleBackupInventoryPathEnvironment] = "relative-inventory.json"
		config, err := loadStartupConfig(
			func(key string) string { return values[key] },
			mapEnvironmentLookup(values),
		)
		clearStartupSecrets(&config)
		if err == nil {
			t.Fatal("relative backup inventory path accepted, want fail closed")
		}
	})

	t.Run("provider requires inventory and journal owner", func(t *testing.T) {
		config := mustLoadStartupConfig(t, validCompositionEnvironment())
		prepared, err := prepareProductionLoginDependencies(
			config,
			bytes.NewReader(bytes.Repeat([]byte{0x5a}, 128)),
		)
		clearStartupSecrets(&config)
		if err != nil {
			t.Fatalf("prepareProductionLoginDependencies() error = %v", err)
		}
		t.Cleanup(prepared.cleanup)
		composition := productionBusinessComposition(t, prepared)
		composition.articleBackupInv = ""
		builder := &productionBusinessBuilder{pool: new(pgxpool.Pool), composition: composition, prepared: prepared}
		if _, err := builder.newArticleReplayProvider(nil); err == nil {
			t.Fatal("provider built without inventory path, want fail closed")
		}
	})

	t.Run("provider builds from the unique composition", func(t *testing.T) {
		config := mustLoadStartupConfig(t, validCompositionEnvironment())
		prepared, err := prepareProductionLoginDependencies(config, bytes.NewReader(bytes.Repeat([]byte{0x5a}, 128)))
		clearStartupSecrets(&config)
		if err != nil {
			t.Fatalf("prepareProductionLoginDependencies() error = %v", err)
		}
		t.Cleanup(prepared.cleanup)
		composition := productionBusinessComposition(t, prepared)
		composition.articleBackupInv = filepath.Join(
			t.TempDir(), "inventory.json",
		)
		composition.articleJournalOwner = mainArticleJournalOwnerStub{}
		builder := &productionBusinessBuilder{
			pool: new(pgxpool.Pool), composition: composition, prepared: prepared,
		}
		provider, err := builder.newArticleReplayProvider(&article.DeletionFinalizer{})
		if err != nil {
			t.Fatalf("newArticleReplayProvider() error = %v", err)
		}
		if provider == nil {
			t.Fatal("newArticleReplayProvider() = nil")
		}
	})
}

var _ = context.Background

func (mainArticleJournalOwnerStub) ReadAllVerified(
	context.Context,
) ([]article.ArticleDeletionTombstone, error) {
	return nil, nil
}

// r49 root unit probes use synthetic configuration and never open a database.
// The pool is a sentinel; no SDK constructor, real environment, or I/O is used.
type maintenanceR49ProbeCounts struct {
	Get, Parse, Connect, Ping, Execute, State, Close, Lookup, Dial int
}

func maintenanceR49Probe(t *testing.T, host string, fallbacks []string, cancelOnReturn bool, primary string) (maintenanceOutput, error, maintenanceR49ProbeCounts, string, string, int, bool) {
	t.Helper()
	var counts maintenanceR49ProbeCounts
	configuration, err := loadMaintenanceConfig(func(key string) string { return maintenanceRootValues()[key] })
	if err != nil {
		t.Fatal(err)
	}
	// A real pgxpool.Config with actual primary and fallback fields, not a fake schema.
	parsed := &pgxpool.Config{MaxConns: 2, ConnConfig: &pgx.ConnConfig{Config: pgconn.Config{Host: host, Port: 5432}}}
	for _, fallback := range fallbacks {
		parsed.ConnConfig.Fallbacks = append(parsed.ConnConfig.Fallbacks, &pgconn.FallbackConfig{Host: fallback, Port: 5432})
	}
	parsed.ConnConfig.LookupFunc = func(context.Context, string) ([]string, error) {
		counts.Lookup++
		return nil, errors.New("unexpected synthetic lookup")
	}
	parsed.ConnConfig.DialFunc = func(context.Context, string, string) (net.Conn, error) {
		counts.Dial++
		return nil, errors.New("unexpected synthetic dial")
	}
	pool := new(pgxpool.Pool)
	dependencies := maintenanceRootDependencies{
		getenv: func(key string) string {
			counts.Get++
			if key != databaseDSNEnvironment {
				t.Errorf("unexpected getenv key=%q", key)
			}
			return "synthetic-not-a-real-dsn"
		},
		parse:   func(string) (*pgxpool.Config, error) { counts.Parse++; return parsed, nil },
		connect: func(context.Context, *pgxpool.Config) (*pgxpool.Pool, error) { counts.Connect++; return pool, nil },
		ping:    func(context.Context, *pgxpool.Pool) error { counts.Ping++; return nil },
		// This probe creates zero resources. Real construction liveness is not inferred from this sentinel.
		state:     func(*pgxpool.Pool) (int32, int32) { counts.State++; return 0, 0 },
		closePool: func(*pgxpool.Pool) { counts.Close++ },
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	execute := func(context.Context, *pgxpool.Pool, int) (maintenanceOutput, error) {
		counts.Execute++
		result := maintenanceOutput{RequestID: "synthetic-request", OperationID: "synthetic-operation", Outcome: "applied", AuditID: "synthetic-audit"}
		if primary == "already_applied" {
			result.Outcome = primary
		}
		if cancelOnReturn {
			cancel()
		}
		switch primary {
		case "outcome_unknown":
			return maintenanceOutput{}, &identity.PasswordResetMaintenanceError{Code: identity.PasswordResetMaintenanceOutcomeUnknown}
		case "state_conflict":
			return maintenanceOutput{}, &identity.PasswordResetMaintenanceError{Code: identity.PasswordResetMaintenanceStateConflict}
		case "failed":
			return maintenanceOutput{}, &identity.PasswordResetMaintenanceError{Code: identity.PasswordResetMaintenanceFailed}
		}
		return result, nil
	}
	result, err := executeMaintenanceDatabase(ctx, configuration, execute, dependencies)
	canceled := ctx.Err() != nil
	var stdout, stderr bytes.Buffer
	exit := finishMaintenanceCommand(&stdout, &stderr, result, err)
	return result, err, counts, stdout.String(), stderr.String(), exit, canceled
}

func maintenanceRootR49NumericEndpoints(t *testing.T) {
	for _, tc := range []struct {
		name, host string
		fallbacks  []string
		valid      bool
	}{
		{"primary_hostname", "primary.invalid", []string{"127.0.0.1"}, false},
		{"only_last_fallback_hostname", "127.0.0.1", []string{"::1", "192.0.2.1", "fallback.invalid"}, false},
		{"implicit_localhost", "localhost", nil, false},
		{"empty_endpoint", "", nil, false},
		{"all_literals_ipv4_primary", "127.0.0.1", []string{"::1", "192.0.2.1"}, true},
		{"all_literals_ipv6_primary", "2001:db8::1", []string{"127.0.0.1", "::1"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err, counts, stdout, stderr, exit, canceled := maintenanceR49Probe(t, tc.host, tc.fallbacks, false, "applied")
			t.Logf("counts=%+v actual_exit=%d err_nil=%t canceled=%t result_outcome=%q stdout=%q stderr=%q", counts, exit, err == nil, canceled, result.Outcome, stdout, stderr)
			if counts.Get != 1 || counts.Parse != 1 || counts.Lookup != 0 || counts.Dial != 0 {
				t.Errorf("expected one synthetic get/parse and zero lookup/dial: %+v", counts)
			}
			if !tc.valid {
				if exit != 2 || stderr != "maintenance.invalid_config\n" || stdout != "" || counts.Connect != 0 || counts.Ping != 0 || counts.Execute != 0 || counts.Close != 0 {
					t.Errorf("r49 requires invalid_config/2 before connect; got exit=%d counts=%+v", exit, counts)
				}
			} else if err != nil || exit != 0 || stdout == "" || stderr != "" || counts.Connect != 1 || counts.Ping != 1 || counts.Execute != 1 || counts.Close != 1 {
				t.Errorf("literal control must execute and close successfully: exit=%d counts=%+v", exit, counts)
			}
		})
	}
}

func maintenanceRootR49CanceledSuccess(t *testing.T) {
	for _, tc := range []struct {
		name, primary string
		cancel        bool
		exit          int
		code          string
	}{
		{"applied_then_cancel", "applied", true, 6, "maintenance.failed\n"},
		{"replay_then_cancel", "already_applied", true, 6, "maintenance.failed\n"},
		{"unknown_priority_control", "outcome_unknown", true, 5, "maintenance.outcome_unknown\n"},
		{"existing_primary_control", "state_conflict", true, 4, "maintenance.state_conflict\n"},
		{"operator_failed_control", "failed", false, 6, "maintenance.failed\n"},
		{"uncanceled_success_control", "applied", false, 0, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err, counts, stdout, stderr, exit, canceled := maintenanceR49Probe(t, "127.0.0.1", []string{"::1"}, tc.cancel, tc.primary)
			t.Logf("counts=%+v actual_exit=%d err_nil=%t canceled=%t result_outcome=%q stdout=%q stderr=%q", counts, exit, err == nil, canceled, result.Outcome, stdout, stderr)
			if counts.Get != 1 || counts.Parse != 1 || counts.Connect != 1 || counts.Ping != 1 || counts.Execute != 1 || counts.Close != 1 || counts.State == 0 || counts.Lookup != 0 || counts.Dial != 0 || canceled != tc.cancel {
				t.Errorf("unexpected synthetic pipeline/resource observations: %+v canceled=%t", counts, canceled)
			}
			if exit != tc.exit || stderr != tc.code || (tc.exit != 0 && stdout != "") || (tc.exit == 0 && stdout == "") {
				t.Errorf("r49 expected exit=%d stderr=%q stdout_empty=%t; got exit=%d stdout=%q stderr=%q", tc.exit, tc.code, tc.exit != 0, exit, stdout, stderr)
			}
		})
	}
}

func maintenanceRootR49Network(t *testing.T) {
	t.Run("literal_lookup_replaces_resolver", func(t *testing.T) {
		owner := newMaintenanceRootTestOwner(t)
		tlsConfig := &tls.Config{ServerName: "unchanged.invalid", MinVersion: tls.VersionTLS12}
		config := &pgx.ConnConfig{Config: pgconn.Config{TLSConfig: tlsConfig,
			LookupFunc: func(context.Context, string) ([]string, error) { t.Error("inherited resolver called"); return nil, nil }}}
		if err := owner.beforeConnect(context.Background(), config); err != nil {
			t.Fatal(err)
		}
		for _, host := range []string{"127.0.0.1", "::1", "2001:db8::1", "hostname.invalid", "localhost", ""} {
			addresses, err := config.LookupFunc(context.Background(), host)
			valid := net.ParseIP(host) != nil
			if (err == nil) != valid || (valid && !reflect.DeepEqual(addresses, []string{host})) {
				t.Fatalf("literal host=%q addresses=%v err=%v", host, addresses, err)
			}
		}
		if config.TLSConfig != tlsConfig || tlsConfig.ServerName != "unchanged.invalid" {
			t.Fatal("maintenance altered TLS verification/SNI")
		}
		if err := owner.cleanup(func() (int32, int32) { return 0, 0 }, func() {}); err != nil {
			t.Fatal(err)
		}
	})
	for _, tc := range []struct {
		name, network, address string
		valid                  bool
	}{
		{"ipv4", "tcp", "127.0.0.1:5432", true}, {"ipv6", "tcp6", "[::1]:5432", true},
		{"hostname", "tcp", "host.invalid:5432", false}, {"empty_host", "tcp", ":5432", false},
		{"bad_port", "tcp", "127.0.0.1:postgres", false}, {"port_overflow", "tcp", "127.0.0.1:65536", false},
		{"unix", "unix", "/synthetic/socket", false}, {"missing_port", "tcp", "127.0.0.1", false},
	} {
		t.Run("dial_"+tc.name, func(t *testing.T) {
			owner := newMaintenanceRootTestOwner(t)
			calls := 0
			config := &pgx.ConnConfig{Config: pgconn.Config{DialFunc: func(ctx context.Context, _, _ string) (net.Conn, error) {
				calls++
				deadline, ok := ctx.Deadline()
				if !ok || deadline.After(owner.deadline) {
					t.Error("numeric Dial lost original D")
				}
				client, peer := net.Pipe()
				t.Cleanup(func() { _ = peer.Close() })
				return client, nil
			}}}
			if err := owner.beforeConnect(context.Background(), config); err != nil {
				t.Fatal(err)
			}
			connection, err := config.DialFunc(context.Background(), tc.network, tc.address)
			if connection != nil {
				_ = connection.Close()
			}
			if (err == nil) != tc.valid || (calls == 1) != tc.valid {
				t.Fatalf("valid=%t calls=%d err=%v", tc.valid, calls, err)
			}
			if err := owner.cleanup(func() (int32, int32) { return 0, 0 }, func() {}); err != nil {
				t.Fatal(err)
			}
		})
	}
	t.Run("literal_and_all_fallback_properties_100", func(t *testing.T) {
		property := func(value uint32) bool {
			host := net.IPv4(byte(value>>24), byte(value>>16), byte(value>>8), byte(value)).String()
			addresses, err := maintenanceLiteralLookup(context.Background(), host)
			config := &pgxpool.Config{ConnConfig: &pgx.ConnConfig{Config: pgconn.Config{Host: host, Fallbacks: []*pgconn.FallbackConfig{{Host: "::1"}, {Host: host}, {Host: "bad.invalid"}}}}}
			return err == nil && len(addresses) == 1 && addresses[0] == host && validateMaintenanceEndpoints(config) != nil
		}
		if err := quick.Check(property, &quick.Config{MaxCount: 100, Rand: mathrand.New(mathrand.NewSource(1054902))}); err != nil {
			t.Fatal(err)
		}
	})
}

func maintenanceRootR49BusinessStages(t *testing.T) {
	for _, phase := range []string{"parse", "connect", "ping"} {
		t.Run(phase, func(t *testing.T) {
			values := maintenanceRootValues()
			config, err := loadMaintenanceConfig(func(key string) string { return values[key] })
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			var connect, ping, execute, closed int
			deps := maintenanceRootDependencies{
				getenv: func(string) string { return "synthetic" },
				parse: func(string) (*pgxpool.Config, error) {
					if phase == "parse" {
						cancel()
					}
					return &pgxpool.Config{MaxConns: 2, ConnConfig: &pgx.ConnConfig{Config: pgconn.Config{Host: "127.0.0.1"}}}, nil
				},
				connect: func(context.Context, *pgxpool.Config) (*pgxpool.Pool, error) {
					connect++
					if phase == "connect" {
						cancel()
					}
					return new(pgxpool.Pool), nil
				},
				ping:      func(context.Context, *pgxpool.Pool) error { ping++; cancel(); return nil },
				state:     func(*pgxpool.Pool) (int32, int32) { return 0, 0 },
				closePool: func(*pgxpool.Pool) { closed++ },
			}
			result, err := executeMaintenanceDatabase(ctx, config, func(context.Context, *pgxpool.Pool, int) (maintenanceOutput, error) {
				execute++
				return maintenanceOutput{Outcome: "applied"}, nil
			}, deps)
			var stdout, stderr bytes.Buffer
			exit := finishMaintenanceCommand(&stdout, &stderr, result, err)
			wantConnect, wantPing := 1, 0
			if phase == "parse" {
				wantConnect = 0
			}
			if phase == "ping" {
				wantPing = 1
			}
			if exit != 6 || stdout.Len() != 0 || connect != wantConnect || ping != wantPing || execute != 0 || closed != wantConnect {
				t.Fatalf("phase=%s exit=%d connect=%d ping=%d execute=%d close=%d", phase, exit, connect, ping, execute, closed)
			}
		})
	}
	t.Run("original_D_even_if_timer_late", func(t *testing.T) {
		ctx := maintenanceRootLateTimerContext{Context: context.Background(), deadline: time.Now().Add(-time.Nanosecond)}
		owner := newMaintenanceConnectionOwner(ctx, time.Second)
		defer owner.stopWatch()
		if err := owner.beforeConnect(context.Background(), &pgx.ConnConfig{}); err == nil {
			t.Fatal("constructor started after D with unscheduled timer")
		}
		if _, err := maintenanceLiteralLookup(ctx, "127.0.0.1"); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal("literal Lookup accepted expired D")
		}
		if _, _, err := owner.stage(context.Background(), &maintenanceAttempt{}); err == nil {
			t.Fatal("stage started after D")
		}
		done := make(chan struct{})
		connection := &pgx.Conn{}
		closed := 0
		owner.connections[connection] = &maintenanceCloseEntry{attempt: &maintenanceAttempt{}, finished: make(chan struct{}), cleanupDone: done,
			isClosed: func() bool { return false }, closeSDK: func(context.Context) error { closed++; close(done); return nil }}
		if err := owner.afterConnect(context.Background(), connection); err == nil || closed != 1 {
			t.Fatal("late constructor was handed off without joining SDK close")
		}
	})
	t.Run("canceled_numeric_dial_return_is_closed", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		owner := newMaintenanceConnectionOwner(ctx, time.Second)
		defer owner.stopWatch()
		client, peer := net.Pipe()
		defer peer.Close()
		config := &pgx.ConnConfig{Config: pgconn.Config{DialFunc: func(context.Context, string, string) (net.Conn, error) { cancel(); return client, nil }}}
		if err := owner.beforeConnect(ctx, config); err != nil {
			t.Fatal(err)
		}
		connection, err := config.DialFunc(ctx, "tcp", "127.0.0.1:5432")
		if err == nil || connection != nil {
			t.Fatal("late Dial handed off a usable transport")
		}
		var b [1]byte
		if _, err := peer.Read(b[:]); err != io.EOF {
			t.Fatalf("late transport not closed: %v", err)
		}
		owner.mu.Lock()
		sockets, stages := len(owner.sockets), len(owner.stages)
		owner.mu.Unlock()
		if sockets != 0 || stages != 0 {
			t.Fatal("late transport/stage escaped owner")
		}
	})
}

func maintenanceRootR49CancelOwnership(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("successful native maintenance input is Windows-only")
	}
	for _, outcome := range []string{"applied", "already_applied", "outcome_unknown"} {
		t.Run(outcome, func(t *testing.T) {
			input, writer, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer input.Close()
			if _, err := writer.Write([]byte("synthetic-envelope")); err != nil {
				_ = writer.Close()
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			values := maintenanceRootValues()
			values[databaseDSNEnvironment] = "synthetic-not-a-real-dsn"
			var operation context.Context
			closed := 0
			deps := maintenanceRootDependencies{
				getenv: func(key string) string { return values[key] },
				prepare: func(maintenanceConfig) (maintenanceValidate, error) {
					return func(ctx context.Context, envelope []byte) (maintenanceExecute, error) {
						operation = ctx
						if string(envelope) != "synthetic-envelope" || ctx.Err() != nil {
							t.Fatal("invalid synthetic input or premature cancel")
						}
						return func(ctx context.Context, _ *pgxpool.Pool, _ int) (maintenanceOutput, error) {
							if ctx.Err() != nil {
								t.Error("self-cancel before execution")
							}
							if outcome == "outcome_unknown" {
								return maintenanceOutput{}, &identity.PasswordResetMaintenanceError{Code: identity.PasswordResetMaintenanceOutcomeUnknown}
							}
							return maintenanceOutput{RequestID: "request", OperationID: "operation", Outcome: outcome, AuditID: "audit"}, nil
						}, nil
					}, nil
				},
				parse: func(string) (*pgxpool.Config, error) {
					return &pgxpool.Config{MaxConns: 2, ConnConfig: &pgx.ConnConfig{Config: pgconn.Config{Host: "127.0.0.1"}}}, nil
				},
				connect: func(context.Context, *pgxpool.Config) (*pgxpool.Pool, error) { return new(pgxpool.Pool), nil },
				ping:    func(context.Context, *pgxpool.Pool) error { return nil },
				state:   func(*pgxpool.Pool) (int32, int32) { return 0, 0 },
				closePool: func(*pgxpool.Pool) {
					closed++
					if operation.Err() != nil {
						t.Error("self-cancel before resource release/final checks")
					}
				},
			}
			result, err := runMaintenance(context.Background(), input, deps)
			if operation == nil || !errors.Is(operation.Err(), context.Canceled) || closed != 1 || input.Fd() != ^uintptr(0) {
				t.Fatal("operation cancel/input/pool ownership not completed")
			}
			var stdout, stderr bytes.Buffer
			exit := finishMaintenanceCommand(&stdout, &stderr, result, err)
			want := 0
			if outcome == "outcome_unknown" {
				want = 5
			}
			if exit != want || (want == 0 && stdout.Len() == 0) || (want != 0 && stdout.Len() != 0) {
				t.Fatalf("outcome=%s exit=%d err=%v stdout=%q stderr=%q", outcome, exit, err, stdout.String(), stderr.String())
			}
		})
	}
}

func maintenanceRootR49AuthorityDeadline(t *testing.T) {
	for _, scenario := range []string{"already_canceled", "elapsed_1ns_budget", "late_parent_timer", "timely_prepare"} {
		t.Run(scenario, func(t *testing.T) {
			values := maintenanceRootValues()
			parent, cancel := context.WithCancel(context.Background())
			defer cancel()
			var ctx context.Context = parent
			switch scenario {
			case "already_canceled":
				cancel()
			case "elapsed_1ns_budget":
				values[maintenanceOperationEnvironment] = "1ns"
			case "late_parent_timer":
				ctx = maintenanceRootLateTimerContext{Context: parent, deadline: time.Now().Add(-time.Nanosecond)}
			}
			input, writer, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer writer.Close()
			prepares, dsnReads := 0, 0
			var firstConfigRead time.Time
			deps := maintenanceRootDependencies{
				getenv: func(key string) string {
					if key == maintenanceTrustEnvironment {
						firstConfigRead = time.Now()
					}
					if scenario == "elapsed_1ns_budget" && key == recoveryModeEnvironment {
						for !time.Now().After(firstConfigRead) {
							runtime.Gosched()
						}
					}
					if key == databaseDSNEnvironment {
						dsnReads++
					}
					return values[key]
				},
				prepare: func(maintenanceConfig) (maintenanceValidate, error) {
					prepares++
					return nil, &identity.PasswordResetMaintenanceError{Code: identity.PasswordResetMaintenanceForbidden}
				},
			}
			result, err := runMaintenance(ctx, input, deps)
			var stdout, stderr bytes.Buffer
			exit := finishMaintenanceCommand(&stdout, &stderr, result, err)
			wantPrepare, wantExit, wantCode := 0, 6, "maintenance.failed\n"
			if scenario == "timely_prepare" {
				wantPrepare, wantExit, wantCode = 1, 3, "maintenance.forbidden\n"
			}
			t.Logf("scenario=%s prepare_calls=%d dsn_reads=%d exit=%d stdin_closed=%t", scenario, prepares, dsnReads, exit, input.Fd() == ^uintptr(0))
			if prepares != wantPrepare || dsnReads != 0 || exit != wantExit || stdout.Len() != 0 || stderr.String() != wantCode || input.Fd() != ^uintptr(0) {
				t.Fatalf("authority phase violated original D/cancel boundary: want_prepare=%d want_exit=%d err=%v stdout=%q stderr=%q", wantPrepare, wantExit, err, stdout.String(), stderr.String())
			}
		})
	}
}

// Models a delayed deadline timer: Deadline and monotonic time have passed, but
// Err/Done have not been dispatched. It does not replace a production clock.
type maintenanceRootLateTimerContext struct {
	context.Context
	deadline time.Time
}

func (ctx maintenanceRootLateTimerContext) Deadline() (time.Time, bool) { return ctx.deadline, true }

func maintenanceRootR49StickyEnd(t *testing.T) {
	for _, tc := range []struct {
		name           string
		canceled, late bool
	}{
		{"timely", false, false}, {"late_timer", false, true}, {"ctx_expired", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.canceled {
				cancel()
			}
			cutoff := time.Now().Add(time.Second)
			ctx := maintenanceRootLateTimerContext{Context: base, deadline: cutoff}
			end := cutoff.Add(-time.Nanosecond)
			if tc.late {
				end = cutoff.Add(time.Nanosecond)
			}
			owner := &maintenanceConnectionOwner{}
			err := owner.finishCleanup(ctx, nil, end)
			want := tc.canceled || tc.late
			if (err != nil) != want {
				t.Fatalf("want_failure=%t err=%v", want, err)
			}
			// A later, successful callback and zero counters must not erase failure.
			err = owner.finishCleanup(context.Background(), nil, end)
			if (err != nil) != want {
				t.Fatal("sticky cleanup failure erased")
			}
		})
	}
	t.Run("nested_common_cutoff", func(t *testing.T) {
		owner := newMaintenanceRootTestOwner(t)
		parent, cancel := context.WithDeadline(context.Background(), time.Now().Add(500*time.Millisecond))
		defer cancel()
		first, stopFirst := owner.connectionCloseContext(parent)
		defer stopFirst()
		second, stopSecond := owner.connectionCloseContext(context.Background())
		defer stopSecond()
		a, _ := first.Deadline()
		b, _ := second.Deadline()
		if !a.Equal(b) || a.After(owner.deadline.Add(owner.budget)) {
			t.Fatal("nested cleanup reopened its window")
		}
	})
	t.Run("close_entry_final_end_before_latch", func(t *testing.T) {
		owner := newMaintenanceRootTestOwner(t)
		// No timer callback is dispatched by this parent. Actual SDK completion
		// and final monotonic time must still record failure before publishing done.
		parent := maintenanceRootLateTimerContext{Context: context.Background(), deadline: time.Now().Add(-time.Nanosecond)}
		done := make(chan struct{})
		entry := &maintenanceCloseEntry{attempt: &maintenanceAttempt{}, finished: make(chan struct{}), cleanupDone: done,
			isClosed: func() bool { return false }, closeSDK: func(context.Context) error { close(done); return nil }}
		err := owner.closeEntry(parent, entry)
		if !errors.Is(err, context.DeadlineExceeded) || parent.Err() != nil {
			t.Fatalf("late timer final-end failure missing: %v", err)
		}
		select {
		case <-entry.finished:
		default:
			t.Fatal("entry not joined")
		}
		if !errors.Is(entry.err, context.DeadlineExceeded) {
			t.Fatal("entry latch published success after C")
		}
		closed := false
		err = owner.cleanup(func() (int32, int32) { return 0, 0 }, func() { closed = true })
		var stdout, stderr bytes.Buffer
		if !closed || finishMaintenanceCommand(&stdout, &stderr, maintenanceOutput{Outcome: "applied"}, maintenanceFinalError(nil, err, context.Background())) != 6 || stdout.Len() != 0 {
			t.Fatal("later empty cleanup erased sticky overrun")
		}
	})
	t.Run("private_cleanup_cause_preserves_primary", func(t *testing.T) {
		for _, code := range []identity.PasswordResetMaintenanceCode{identity.PasswordResetMaintenanceOutcomeUnknown, identity.PasswordResetMaintenanceStateConflict, identity.PasswordResetMaintenanceForbidden} {
			primary := &identity.PasswordResetMaintenanceError{Code: code}
			cause := errors.New("private cleanup diagnostic")
			err := maintenanceFinalError(primary, cause, context.Background())
			var typed *identity.PasswordResetMaintenanceError
			if !errors.As(err, &typed) || typed != primary || !errors.Is(err, cause) {
				t.Fatal("primary or private cause lost")
			}
			var stdout, stderr bytes.Buffer
			_, want := maintenanceErrorCode(primary)
			if finishMaintenanceCommand(&stdout, &stderr, maintenanceOutput{}, err) != want || stdout.Len() != 0 || strings.Contains(stderr.String(), "private") {
				t.Fatal("cleanup changed primary or exposed cause")
			}
		}
	})
	t.Run("sticky_end_properties_100", func(t *testing.T) {
		property := func(delta int16) bool {
			cutoff := time.Now().Add(time.Second)
			ctx := maintenanceRootLateTimerContext{Context: context.Background(), deadline: cutoff}
			owner := &maintenanceConnectionOwner{}
			err := owner.finishCleanup(ctx, nil, cutoff.Add(time.Duration(delta)))
			later := owner.finishCleanup(context.Background(), nil, cutoff)
			return (err != nil) == (delta > 0) && (later != nil) == (delta > 0)
		}
		if err := quick.Check(property, &quick.Config{MaxCount: 100, Rand: mathrand.New(mathrand.NewSource(1054922))}); err != nil {
			t.Fatal(err)
		}
	})
}

func TestRun_RecoveryMaintenanceIsolation(t *testing.T) {
	for _, scenario := range []struct {
		name, mode       string
		invalid, blocked bool
	}{
		{"maintenance_composes_single_pipeline", "maintenance", false, false},
		{"normal_pipeline_control", "normal", false, false},
		{"invalid_config_priority_control", "maintenance", true, true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			values := validCompositionEnvironment()
			values[recoveryModeEnvironment] = scenario.mode
			if scenario.invalid {
				values[loginObservationBudgetEnvironment] = ""
			}
			cfg, configErr := loadStartupConfig(func(environmentName string) string { return values[environmentName] }, mapEnvironmentLookup(values))
			if (configErr != nil) != scenario.invalid {
				t.Fatalf("fixture validity mismatch: %v", configErr)
			}
			if configErr == nil && cfg.recoveryMode != scenario.mode {
				t.Fatal("fixture mode not parsed")
			}
			clearStartupSecrets(&cfg)
			keys := []string{"prepare_business", "prepare_article", "parse_db", "connect", "ping", "migrate", "bootstrap", "open_article", "compose", "worker_factory", "listen", "new_server", "coordinate", "worker_handoff", "worker_reconcile", "worker_start", "serve", "close_pool", "close_listener", "close_article"}
			counts := map[string]int{}
			for _, key := range keys {
				counts[key] = 0
			}
			deps := validCompositionDependencies(values)
			deps.getenv = func(environmentName string) string {
				if environmentName == databaseDSNEnvironment {
					counts["config_dsn_read"]++
				}
				return values[environmentName]
			}
			var retained [][]byte
			deps.prepareBusiness = func(config startupConfig) (*preparedLoginDependencies, error) {
				counts["prepare_business"]++
				retained = append(retained, config.loginBucketKey, config.sourceKey)
				for _, key := range config.idempotencyKeys.Keys {
					retained = append(retained, key)
				}
				return &preparedLoginDependencies{}, nil
			}
			deps.prepareArticle = func(config articleDeletionStartupConfig) (*preparedArticleDeletion, error) {
				counts["prepare_article"]++
				retained = append(retained, config.keyring.Active.Material)
				for _, key := range config.keyring.Retained {
					retained = append(retained, key.Material)
				}
				return &preparedArticleDeletion{config: config}, nil
			}
			deps.parseDatabase = func(string) (*pgxpool.Config, error) { counts["parse_db"]++; return &pgxpool.Config{MaxConns: 8}, nil }
			deps.connect = func(context.Context, *pgxpool.Config) (*pgxpool.Pool, error) {
				counts["connect"]++
				return new(pgxpool.Pool), nil
			}
			deps.ping = func(context.Context, *pgxpool.Pool) error { counts["ping"]++; return nil }
			deps.migrate = func(context.Context, *pgxpool.Pool) error { counts["migrate"]++; return nil }
			deps.bootstrapInitialAdmin = func(context.Context, *pgxpool.Pool, *preparedLoginDependencies) error {
				counts["bootstrap"]++
				return nil
			}
			deps.openArticle = func(*preparedArticleDeletion) (articleDeletionJournalOwner, error) {
				counts["open_article"]++
				return mainArticleJournalOwnerStub{}, nil
			}
			var articleWorker, passwordWorker *lifecycleWorkerFake
			var noticeWorker *noticeLifecycleProbe
			deps.requirePasswordResetWorker, deps.requireNoticeWorker = true, true
			deps.composeBusiness = func(_ *pgxpool.Pool, composition businessComposition) (http.Handler, error) {
				counts["compose"]++
				articleWorker, passwordWorker, noticeWorker = newLifecycleWorkerFake(), newLifecycleWorkerFake(), newNoticeLifecycleProbe()
				counts["worker_factory"] += 3
				composition.runtime.articleWorker, composition.runtime.passwordResetWorker, composition.runtime.noticeWorker = articleWorker, passwordWorker, noticeWorker
				return http.NotFoundHandler(), nil
			}
			deps.listen = func(string, string) (net.Listener, error) {
				counts["listen"]++
				return &recordingListener{onClose: func() { counts["close_listener"]++ }}, nil
			}
			deps.newServer = func(name string, _ http.Handler) lifecycleServer {
				counts["new_server"]++
				return newLifecycleServer(name)
			}
			deps.closePool = func(*pgxpool.Pool) { counts["close_pool"]++ }
			stop := errors.New("synthetic lifecycle stop")
			deps.coordinate = func(ctx context.Context, config lifecycleConfig) error {
				counts["coordinate"]++
				if config.articleWorker != articleWorker || config.passwordResetWorker != passwordWorker || config.noticeWorker != noticeWorker {
					t.Fatal("worker handoff mismatch")
				}
				counts["worker_handoff"] += 3
				for _, worker := range []lifecycleWorker{articleWorker, passwordWorker, noticeWorker} {
					worker.Stop()
					if err := worker.Wait(ctx); err != nil {
						t.Fatal(err)
					}
				}
				if err := config.businessListener.Close(); err != nil {
					t.Fatal(err)
				}
				if err := config.metricsListener.Close(); err != nil {
					t.Fatal(err)
				}
				if config.closeArticle != nil {
					counts["close_article"]++
					if err := config.closeArticle(); err != nil {
						t.Fatal(err)
					}
				}
				if err := config.closePool(); err != nil {
					t.Fatal(err)
				}
				return stop
			}
			err := run(context.Background(), deps)
			cleared := len(retained) > 0
			for _, key := range retained {
				for _, valueByte := range key {
					if valueByte != 0 {
						cleared = false
					}
				}
			}
			if articleWorker != nil {
				counts["worker_reconcile"] = articleWorker.reconcileCalls + passwordWorker.reconcileCalls + noticeWorker.reconcileCalls
				counts["worker_start"] = articleWorker.startCalls + passwordWorker.startCalls + noticeWorker.startCalls
			}
			encoded, _ := json.Marshal(counts)
			t.Logf("mode=%s valid_config=%t counts=%s synthetic_secret_buffers_observed=%d cleared=%t stop_cause=%t", scenario.mode, !scenario.invalid, encoded, len(retained), cleared, errors.Is(err, stop))
			if scenario.blocked {
				if err == nil {
					t.Error("expected startup refusal")
				}
				for _, key := range keys {
					if counts[key] != 0 {
						t.Errorf("startup isolation boundary crossed: %s=%d, want0", key, counts[key])
					}
				}
				if scenario.invalid && (err == nil || !strings.Contains(err.Error(), loginObservationBudgetEnvironment)) {
					t.Error("configuration error priority lost")
				}
			} else {
				want := map[string]int{"prepare_business": 1, "prepare_article": 1, "parse_db": 1, "connect": 1, "ping": 1, "migrate": 1, "bootstrap": 1, "open_article": 1, "compose": 1, "worker_factory": 3, "listen": 2, "new_server": 2, "coordinate": 1, "worker_handoff": 3, "worker_reconcile": 0, "worker_start": 0, "serve": 0, "close_pool": 1, "close_listener": 2, "close_article": 1}
				if !errors.Is(err, stop) {
					t.Fatalf("normal did not reach synthetic lifecycle: %v", err)
				}
				for key, value := range want {
					if counts[key] != value {
						t.Errorf("normal %s=%d want%d", key, counts[key], value)
					}
				}
			}
			if len(retained) > 0 && !cleared {
				t.Fatal("existing synthetic startup secret buffers not cleared")
			}
		})
	}
	t.Run("existing_password_reset_argv_control", maintenanceRootModeTable)
}

func TestRecoveryRootStartup(t *testing.T) {
	if maintenanceRootHelper(t) {
		return
	}
	t.Run("recovery_selected_before_legacy_maintenance_parser", func(t *testing.T) {
		if _, selected, err := selectRecoveryCommand([]string{"recovery", "verify"}); !selected || err != nil {
			t.Fatalf("recovery parser did not claim its entry: %v", err)
		}
		if maintenance, err := selectMaintenanceMode([]string{"recovery", "verify"}); !maintenance || err == nil {
			t.Fatalf("legacy maintenance parser claim = %t, %v", maintenance, err)
		}
		stdout, stderr, code := runMainHelperProcess(t, "recovery", "verify")
		if code != 1 {
			t.Fatalf("recovery verify exit = %d", code)
		}
		if stdout != "" {
			t.Fatalf("legacy maintenance output on recovery entry: %q", stdout)
		}
		if !strings.Contains(stderr, startupFailureMessage) {
			t.Fatalf("recovery verify stderr = %q", stderr)
		}
	})

	t.Run("invalid_recovery_arguments_have_no_side_effects", func(t *testing.T) {
		for _, args := range [][]string{
			{"recovery", "enter", "--run-id"},
			{"recovery", "unknown"},
			{"recovery"},
		} {
			stdout, stderr, code := runMainHelperProcess(t, args...)
			if code != 2 {
				t.Fatalf("%v exit = %d", args, code)
			}
			if stdout != "" {
				t.Fatalf("%v produced output: %q", args, stdout)
			}
			if strings.TrimSpace(stderr) != startupFailureMessage {
				t.Fatalf("%v stderr = %q", args, stderr)
			}
		}
	})

	t.Run("verify_opens_no_listener_worker_or_repair", func(t *testing.T) {
		values := validCompositionEnvironment()
		for key, value := range recoveryConfigFixture(t) {
			values[key] = value
		}
		counts := map[string]int{}
		deps := validCompositionDependencies(values)
		deps.prepareBusiness = func(startupConfig) (*preparedLoginDependencies, error) {
			return &preparedLoginDependencies{clock: mainLoginClock{now: time.Date(2026, time.October, 30, 12, 0, 0, 0, time.UTC)}}, nil
		}
		deps.prepareArticle = func(config articleDeletionStartupConfig) (*preparedArticleDeletion, error) {
			return &preparedArticleDeletion{config: config}, nil
		}
		deps.openArticle = func(*preparedArticleDeletion) (articleDeletionJournalOwner, error) {
			return mainArticleJournalOwnerStub{}, nil
		}
		deps.listen = func(string, string) (net.Listener, error) {
			counts["listen"]++
			return &recordingListener{}, nil
		}
		deps.newServer = func(name string, _ http.Handler) lifecycleServer {
			counts["new_server"]++
			return newLifecycleServer(name)
		}
		deps.coordinate = func(context.Context, lifecycleConfig) error {
			counts["coordinate"]++
			return nil
		}
		deps.composeBusiness = func(_ *pgxpool.Pool, composition businessComposition) (http.Handler, error) {
			composition.runtime.articleWorker = newLifecycleWorkerFake()
			composition.runtime.passwordResetWorker = newLifecycleWorkerFake()
			composition.runtime.noticeWorker = newNoticeLifecycleProbe()
			return http.NotFoundHandler(), nil
		}
		deps.recoveryCommand = &recoveryCommand{action: "verify"}
		err := run(context.Background(), deps)
		if !errors.Is(err, recovery.ErrAdmissionClosed) {
			t.Fatalf("verify with a closed keyring loader = %v", err)
		}
		for _, key := range []string{"listen", "new_server", "coordinate"} {
			if counts[key] != 0 {
				t.Fatalf("verify crossed %s=%d", key, counts[key])
			}
		}
	})

	t.Run("control_allowlist_serves_only_exact_routes", func(t *testing.T) {
		handler := newRecoveryControlAllowlistFixture(t)
		for _, scenario := range []struct {
			method, path string
			status       int
			allow        string
		}{
			{http.MethodGet, "/api/v1/recovery", http.StatusNotFound, ""},
			{http.MethodGet, "/api/v1/recovery/unknown", http.StatusNotFound, ""},
			{http.MethodGet, "/api/v1/recovery/reviews/x", http.StatusNotFound, ""},
			{http.MethodGet, "/api/v1/recovery/complete", http.StatusMethodNotAllowed, "POST"},
			{http.MethodPut, "/api/v1/recovery/complete", http.StatusMethodNotAllowed, "POST"},
			{http.MethodPost, "/api/v1/recovery/reviews", http.StatusMethodNotAllowed, "GET"},
			{http.MethodGet, "/api/v1/recovery/reviews/x/resolve", http.StatusMethodNotAllowed, "POST"},
			{http.MethodGet, "/api/v1/recovery/idempotency/resolve", http.StatusMethodNotAllowed, "POST"},
		} {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(scenario.method, scenario.path, nil))
			if response.Code != scenario.status {
				t.Fatalf("%s %s status = %d, want %d", scenario.method, scenario.path, response.Code, scenario.status)
			}
			if scenario.allow != "" && response.Header().Get("Allow") != scenario.allow {
				t.Fatalf("%s %s Allow = %q, want %q", scenario.method, scenario.path, response.Header().Get("Allow"), scenario.allow)
			}
		}
	})
}

func runMainHelperProcess(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	command := exec.Command(os.Args[0], append([]string{"-test.run=^TestRecoveryRootStartup$", "--"}, args...)...)
	command.Env = []string{mainHelperEnvironment + "=maintenance-root"}
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	var exit *exec.ExitError
	if err != nil && !errors.As(err, &exit) {
		t.Fatalf("helper process failed: %v", err)
	}
	code := 0
	if exit != nil {
		code = exit.ExitCode()
	}
	return stdout.String(), stderr.String(), code
}

func newRecoveryControlAllowlistFixture(t *testing.T) http.Handler {
	t.Helper()
	authorizer, err := platform.NewHTTPAuthorizer(
		mainAdministratorResolver{},
		func() time.Time { return time.Date(2026, time.October, 30, 12, 0, 0, 0, time.UTC) },
	)
	if err != nil {
		t.Fatal(err)
	}
	handlers, err := identity.NewRecoveryControlHandlers(identity.RecoveryControlDependencies{
		Maintenance: recoveryControlAllowlistMaintenance{},
		Reviews:     recoveryControlAllowlistReviews{},
		Complete:    recoveryControlAllowlistComplete{},
		Authorizer:  authorizer,
		Phase: func(context.Context) (context.Context, recovery.PhasePermit, recovery.RunBinding, error) {
			return context.Background(), recovery.PhasePermit{}, recovery.RunBinding{}, nil
		},
		Session:      func(http.ResponseWriter, *http.Request) {},
		Liveness:     func(http.ResponseWriter, *http.Request) {},
		Readiness:    func(http.ResponseWriter, *http.Request) {},
		MaxBodyBytes: 65536,
	})
	if err != nil {
		t.Fatal(err)
	}
	return handlers.Handler()
}

type recoveryControlAllowlistMaintenance struct{}

func (recoveryControlAllowlistMaintenance) ResolveRestrictedAccountReview(
	context.Context, recovery.PhasePermit, recovery.RunBinding, string, string, string,
) error {
	return nil
}

func (recoveryControlAllowlistMaintenance) ResolveRecoveryIdempotency(
	context.Context, recovery.PhasePermit, recovery.RunBinding, []byte, []byte, int64, string, string,
) error {
	return nil
}

type recoveryControlAllowlistReviews struct{}

func (recoveryControlAllowlistReviews) ListPendingReviews(context.Context, string, int) ([]identity.RecoveryReviewItem, string, error) {
	return nil, "", nil
}

type recoveryControlAllowlistComplete struct{}

func (recoveryControlAllowlistComplete) RequestComplete() error { return nil }
