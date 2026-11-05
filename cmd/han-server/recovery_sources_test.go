package main

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"han/internal/recovery"
)

func TestRecoveryIdentityBridgePermitBinding(t *testing.T) {
	binding := recoveryTestPermitBinding()
	run := recovery.RunBinding{RunID: binding.RunID, BackupRef: binding.BackupRef}

	t.Run("captures_matching_identity_permit", func(t *testing.T) {
		bridge, runtime, _ := newRecoveryTestBridge(t)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := bridge.FinalizeRestoredIdentity(ctx, recovery.PhasePermit{}, run); err != nil {
			t.Fatal(err)
		}
		if err := bridge.waitIdentityReady(ctx); err != nil {
			t.Fatal("identity surface was not announced")
		}
		phaseCtx, _, authorized, err := bridge.identityAuthorization(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if phaseCtx == nil || authorized != run {
			t.Fatalf("authorization = %+v", authorized)
		}
		runtime.mu.Lock()
		checks, reads := runtime.checks, runtime.reads
		runtime.mu.Unlock()
		if checks == 0 || reads == 0 {
			t.Fatalf("permit was not observed: checks=%d reads=%d", checks, reads)
		}
	})

	t.Run("rejects_failed_phase_check", func(t *testing.T) {
		bridge, runtime, _ := newRecoveryTestBridge(t)
		runtime.checkErr = recovery.ErrAdmissionBinding
		if _, err := bridge.capture(context.Background(), recovery.PhasePermit{}, run, recovery.PhaseRecoveryIdentity); !errors.Is(err, recovery.ErrAdmissionBinding) {
			t.Fatalf("failed phase check accepted: %v", err)
		}
	})

	t.Run("rejects_purpose_mismatch", func(t *testing.T) {
		bridge, runtime, _ := newRecoveryTestBridge(t)
		mismatched := binding
		mismatched.Purpose = recovery.PhaseRecoveryReplay
		runtime.binding = mismatched
		if _, err := bridge.capture(context.Background(), recovery.PhasePermit{}, run, recovery.PhaseRecoveryIdentity); !errors.Is(err, recovery.ErrAdmissionBinding) {
			t.Fatalf("wrong purpose accepted: %v", err)
		}
	})

	t.Run("rejects_run_mismatch", func(t *testing.T) {
		bridge, runtime, _ := newRecoveryTestBridge(t)
		mismatched := binding
		mismatched.RunID = "other-run"
		runtime.binding = mismatched
		if _, err := bridge.capture(context.Background(), recovery.PhasePermit{}, run, recovery.PhaseRecoveryIdentity); !errors.Is(err, recovery.ErrAdmissionBinding) {
			t.Fatalf("wrong run accepted: %v", err)
		}
	})

	t.Run("rejects_empty_installation", func(t *testing.T) {
		bridge, runtime, _ := newRecoveryTestBridge(t)
		mismatched := binding
		mismatched.InstallationID = ""
		runtime.binding = mismatched
		if _, err := bridge.capture(context.Background(), recovery.PhasePermit{}, run, recovery.PhaseRecoveryIdentity); !errors.Is(err, recovery.ErrAdmissionBinding) {
			t.Fatalf("empty installation accepted: %v", err)
		}
	})

	t.Run("rejects_drifted_generation_after_capture", func(t *testing.T) {
		bridge, runtime, _ := newRecoveryTestBridge(t)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := bridge.capture(ctx, recovery.PhasePermit{}, run, recovery.PhaseRecoveryIdentity); err != nil {
			t.Fatal(err)
		}
		runtime.mu.Lock()
		runtime.binding.Generation++
		runtime.mu.Unlock()
		if _, _, _, err := bridge.identityAuthorization(ctx); !errors.Is(err, recovery.ErrAdmissionBinding) {
			t.Fatalf("generation drift accepted: %v", err)
		}
	})

	t.Run("rejects_replay_binding_drift", func(t *testing.T) {
		bridge, runtime, _ := newRecoveryTestBridge(t)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		runtime.binding.Purpose = recovery.PhaseRecoveryReplay
		if _, err := bridge.capture(ctx, recovery.PhasePermit{}, run, recovery.PhaseRecoveryReplay); err != nil {
			t.Fatal(err)
		}
		runtime.mu.Lock()
		runtime.binding.InstallationID = "other-installation"
		runtime.mu.Unlock()
		if _, _, _, err := bridge.replayAuthorization(ctx); !errors.Is(err, recovery.ErrAdmissionBinding) {
			t.Fatalf("installation drift accepted: %v", err)
		}
	})
}

func TestRecoveryIdentityBridgeCompleteDefersToSignal(t *testing.T) {
	bridge, _, executor := newRecoveryTestBridge(t)
	run := recovery.RunBinding{RunID: "run-1", BackupRef: "backup-1"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- bridge.CompleteIdentityReview(ctx, recovery.PhasePermit{}, run)
	}()
	if err := bridge.waitIdentityReady(ctx); err != nil {
		t.Fatal(err)
	}
	if err := bridge.RequestComplete(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		t.Fatalf("completion returned before the root delegated: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if executor.count("complete") != 0 {
		t.Fatal("the runtime call stack reached the executor before root delegation")
	}
	if err := bridge.completeIdentity(ctx); err != nil {
		t.Fatal(err)
	}
	if executor.count("complete") != 1 {
		t.Fatalf("executor complete count = %d", executor.count("complete"))
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("completion did not resume after root delegation")
	}
	if err := bridge.completeIdentity(ctx); !errors.Is(err, recovery.ErrAdmissionBusy) {
		t.Fatalf("second delegation accepted: %v", err)
	}
	if executor.count("complete") != 1 {
		t.Fatalf("duplicate delegation reached the executor: %d", executor.count("complete"))
	}
}

func TestRecoveryIdentityBridgeCompletePropagatesRootFailure(t *testing.T) {
	bridge, _, executor := newRecoveryTestBridge(t)
	executor.completeErr = errors.New("complete failed")
	run := recovery.RunBinding{RunID: "run-1", BackupRef: "backup-1"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- bridge.CompleteIdentityReview(ctx, recovery.PhasePermit{}, run)
	}()
	if err := bridge.waitIdentityReady(ctx); err != nil {
		t.Fatal(err)
	}
	if err := bridge.RequestComplete(); err != nil {
		t.Fatal(err)
	}
	if err := bridge.completeIdentity(ctx); err == nil {
		t.Fatal("root delegation failure was swallowed")
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("runtime flow did not observe the root delegation failure")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("completion did not resume after root delegation")
	}
}

func TestRecoveryIdentityBridgeRequestCompleteRequiresPermit(t *testing.T) {
	bridge, _, executor := newRecoveryTestBridge(t)
	if err := bridge.RequestComplete(); !errors.Is(err, recovery.ErrAdmissionBinding) {
		t.Fatalf("signal without a permit accepted: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := bridge.completeIdentity(ctx); err == nil {
		t.Fatal("delegation without a permit accepted")
	}
	if executor.count("complete") != 0 {
		t.Fatal("signal without a permit reached the executor")
	}
}

func TestRecoveryIdentityBridgeRevokeOnlyCaptures(t *testing.T) {
	bridge, _, executor := newRecoveryTestBridge(t)
	run := recovery.RunBinding{RunID: "run-1", BackupRef: "backup-1"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := bridge.RevokeRecoverySessions(ctx, recovery.PhasePermit{}, run); err != nil {
		t.Fatal(err)
	}
	if executor.count("revoke") != 0 {
		t.Fatal("revocation ran before the joins finished")
	}
	if err := bridge.revokeSessions(ctx); err != nil {
		t.Fatal(err)
	}
	if executor.count("revoke") != 1 {
		t.Fatalf("revoke count = %d", executor.count("revoke"))
	}
}

func TestRecoveryIdentityBridgeFinalizeAndTrust(t *testing.T) {
	bridge, _, executor := newRecoveryTestBridge(t)
	run := recovery.RunBinding{RunID: "run-1", BackupRef: "backup-1"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := bridge.FinalizeRestoredIdentity(ctx, recovery.PhasePermit{}, run); err != nil {
		t.Fatal(err)
	}
	if err := bridge.PrepareRecoveryTrust(ctx, recovery.PhasePermit{}, run, []byte("candidate")); err != nil {
		t.Fatal(err)
	}
	if executor.count("finalize") != 1 || executor.count("prepare") != 1 {
		t.Fatalf("delegated calls = %v", executor.calls)
	}
}

func TestRecoverySecretSourceIsLazyCandidate(t *testing.T) {
	if _, err := newRecoverySecretSource(nil); !errors.Is(err, recovery.ErrAdmissionBinding) {
		t.Fatalf("nil reader accepted: %v", err)
	}
	var observed recovery.RunBinding
	source, err := newRecoverySecretSource(func(_ context.Context, binding recovery.RunBinding) ([]byte, error) {
		observed = binding
		return []byte("candidate"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	run := recovery.RunBinding{RunID: "run-1", BackupRef: "backup-1"}
	material, err := source.ReadInitialRecoverySecret(context.Background(), run)
	if err != nil || string(material) != "candidate" || observed != run {
		t.Fatalf("lazy read = %q %+v %v", material, observed, err)
	}
	if _, err := source.ReadInitialRecoverySecret(context.Background(), recovery.RunBinding{}); !errors.Is(err, recovery.ErrAdmissionBinding) {
		t.Fatalf("empty binding accepted: %v", err)
	}
}

func TestRecoveryAnchorAdapterRegistry(t *testing.T) {
	t.Run("empty_registry_is_legal_and_closes", func(t *testing.T) {
		registry, err := newRecoveryAnchorRegistry(nil)
		if err != nil {
			t.Fatalf("empty registry rejected: %v", err)
		}
		ports := registry.resolve("product-cas", filepath.Join(t.TempDir(), "anchor.json"))
		requireClosedRecoveryAnchorPorts(t, ports)
	})

	t.Run("registered_adapter_resolves_exactly", func(t *testing.T) {
		adapter := newRecoveryAnchorAdapterFixture()
		registry, err := newRecoveryAnchorRegistry(map[string]recoveryAnchorAdapter{"product-cas": adapter})
		if err != nil {
			t.Fatal(err)
		}
		ports := registry.resolve("product-cas", filepath.Join(t.TempDir(), "anchor.json"))
		if ports.AdapterID != "product-cas" || ports.Commitments != adapter.Commitments ||
			ports.Sources != adapter.Sources || ports.Anchor != adapter.Anchor || ports.Exclusive != adapter.Exclusive {
			t.Fatalf("resolved ports = %+v", ports)
		}
	})

	t.Run("unregistered_id_closes", func(t *testing.T) {
		registry, err := newRecoveryAnchorRegistry(map[string]recoveryAnchorAdapter{"product-cas": newRecoveryAnchorAdapterFixture()})
		if err != nil {
			t.Fatal(err)
		}
		requireClosedRecoveryAnchorPorts(t, registry.resolve("other-product", filepath.Join(t.TempDir(), "anchor.json")))
		requireClosedRecoveryAnchorPorts(t, registry.resolve("", filepath.Join(t.TempDir(), "anchor.json")))
	})

	t.Run("unavailable_configuration_closes", func(t *testing.T) {
		registry, err := newRecoveryAnchorRegistry(map[string]recoveryAnchorAdapter{"product-cas": newRecoveryAnchorAdapterFixture()})
		if err != nil {
			t.Fatal(err)
		}
		requireClosedRecoveryAnchorPorts(t, registry.resolve("product-cas", "relative/anchor.json"))
		requireClosedRecoveryAnchorPorts(t, registry.resolve("product-cas", ""))
	})

	t.Run("incomplete_registration_is_rejected", func(t *testing.T) {
		adapter := newRecoveryAnchorAdapterFixture()
		adapter.Exclusive = nil
		if _, err := newRecoveryAnchorRegistry(map[string]recoveryAnchorAdapter{"product-cas": adapter}); !errors.Is(err, recovery.ErrAdmissionBinding) {
			t.Fatalf("incomplete registration accepted: %v", err)
		}
	})
}

func newRecoveryAnchorAdapterFixture() recoveryAnchorAdapter {
	closed := recoveryClosedPort{reason: "fixture adapter is not a product"}
	return recoveryAnchorAdapter{Commitments: closed, Sources: closed, Anchor: closed, Exclusive: closed}
}

func requireClosedRecoveryAnchorPorts(t *testing.T, ports recoveryAnchorPorts) {
	t.Helper()
	ctx := context.Background()
	if ports.AdapterID != "" {
		t.Fatalf("closed ports reported adapter %q", ports.AdapterID)
	}
	_, _, err := ports.Commitments.LoadInstallation(ctx, "installation")
	if !errors.Is(err, recovery.ErrAdmissionClosed) {
		t.Fatalf("LoadInstallation = %v", err)
	}
	_, _, err = ports.Commitments.CompareAndSwapInstallation(ctx, "installation", "revision", recovery.InstallationCommitment{})
	if !errors.Is(err, recovery.ErrAdmissionClosed) {
		t.Fatalf("CompareAndSwapInstallation = %v", err)
	}
	if err := ports.Sources.VerifyInstallationSource(ctx, recovery.InstallationManifest{}); !errors.Is(err, recovery.ErrAdmissionClosed) {
		t.Fatalf("VerifyInstallationSource = %v", err)
	}
	_, _, err = ports.Anchor.Load(ctx, "namespace")
	if !errors.Is(err, recovery.ErrAdmissionClosed) {
		t.Fatalf("Anchor Load = %v", err)
	}
	lease, err := ports.Exclusive.Acquire(ctx)
	if !errors.Is(err, recovery.ErrAdmissionClosed) || lease != nil {
		t.Fatalf("Acquire = %v, %v", lease, err)
	}
}

// F008: a complete signal that cannot be honoured must be observable by the
// root controller instead of leaving it blocked in waitCompleteRequest until
// the whole budget expires.
func TestRecoveryIdentityBridgeFailedCompleteSignalWakesRoot(t *testing.T) {
	bridge, _, _ := newRecoveryTestBridge(t)
	signalErr := bridge.RequestComplete()
	if !errors.Is(signalErr, recovery.ErrAdmissionBinding) {
		t.Fatalf("signal without a permit accepted: %v", signalErr)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	began := time.Now()
	err := bridge.waitCompleteRequest(ctx)
	if !errors.Is(err, recovery.ErrAdmissionBinding) {
		t.Fatalf("root did not observe the failed complete signal: %v", err)
	}
	if elapsed := time.Since(began); elapsed > 250*time.Millisecond {
		t.Fatalf("root waited %v for a failed signal", elapsed)
	}
}

// TestRecoveryBridgePhaseBudgetIsConfigured proves the phase contexts the
// bridge derives follow the configured recovery budget instead of a hardcoded
// constant, and that a non-positive budget refuses at construction.
func TestRecoveryBridgePhaseBudgetIsConfigured(t *testing.T) {
	runtime := &recoveryTestRuntime{binding: recoveryTestPermitBinding()}
	executor := &recoveryTestExecutor{}
	bridge, err := newRecoveryIdentityBridge(runtime, executor, 250*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	binding := recoveryTestPermitBinding()
	run := recovery.RunBinding{RunID: binding.RunID, BackupRef: binding.BackupRef}
	if _, err := bridge.capture(context.Background(), recovery.PhasePermit{}, run, recovery.PhaseRecoveryIdentity); err != nil {
		t.Fatal(err)
	}
	phaseCtx, _, _, err := bridge.identityAuthorization(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	deadline, ok := phaseCtx.Deadline()
	if !ok {
		t.Fatal("phase context has no deadline")
	}
	if remaining := time.Until(deadline); remaining <= 0 || remaining > 250*time.Millisecond {
		t.Fatalf("phase budget = %v, want the configured 250ms", remaining)
	}
	if _, err := newRecoveryIdentityBridge(runtime, executor, 0); !errors.Is(err, recovery.ErrAdmissionBinding) {
		t.Fatalf("non-positive budget accepted: %v", err)
	}
}
