package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"han/internal/recovery"
)

const (
	restoreReceiptLimit = 65536
	restoreVersion      = "han-db-restore/v1"
	restoreTimeLayout   = "2006-01-02T15:04:05.000000Z"
)

func restoreContext(ctx context.Context) error {
	if ctx == nil {
		return recovery.ErrAdmissionBinding
	}
	return ctx.Err()
}

// Wire values are private: the public recovery DTO has no JSON field tags.
type restorePlanWire struct {
	Version              string `json:"version"`
	RunID                string `json:"run_id"`
	BackupRef            string `json:"backup_ref"`
	DeploymentID         string `json:"deployment_id"`
	InstallationID       string `json:"installation_id"`
	TargetDigest         string `json:"target_digest"`
	ArtifactDigest       string `json:"artifact_digest"`
	ArtifactBytes        int64  `json:"artifact_bytes"`
	SourceApprovalRef    string `json:"source_approval_ref"`
	SourceApprovalDigest string `json:"source_approval_digest"`
	SourceInstallationID string `json:"source_installation_id"`
	SourceSchemaDigest   string `json:"source_schema_digest"`
	ArtifactPath         string `json:"artifact_path"`
}

type restoreReceiptWire struct {
	Version              string `json:"version"`
	RunID                string `json:"run_id"`
	BackupRef            string `json:"backup_ref"`
	DeploymentID         string `json:"deployment_id"`
	InstallationID       string `json:"installation_id"`
	TargetDigest         string `json:"target_digest"`
	ArtifactDigest       string `json:"artifact_digest"`
	ArtifactBytes        int64  `json:"artifact_bytes"`
	SourceApprovalDigest string `json:"source_approval_digest"`
	RestoredSchemaDigest string `json:"restored_schema_digest"`
	RestoredAt           string `json:"restored_at"`
	ReceiptDigest        string `json:"receipt_digest,omitempty"`
}

func planWire(p recovery.RestorePlan) restorePlanWire {
	return restorePlanWire{p.Version, p.RunID, p.BackupRef, p.DeploymentID, p.InstallationID, p.TargetDigest, p.ArtifactDigest, p.ArtifactBytes, p.SourceApprovalRef, p.SourceApprovalDigest, p.SourceInstallationID, p.SourceSchemaDigest, p.ArtifactPath}
}

func privateRestoreInput(p recovery.RestorePlan, dsn string) ([]byte, error) {
	input := struct {
		Plan   restorePlanWire `json:"plan"`
		Target struct {
			DSN string `json:"dsn"`
		} `json:"target_connection"`
	}{Plan: planWire(p)}
	input.Target.DSN = dsn
	return json.Marshal(input)
}

// strictRestoreObject rejects duplicate decoded keys, nulls, and trailing values.
func strictRestoreObject(raw []byte, keys []string) error {
	if len(raw) == 0 || len(raw) > restoreReceiptLimit || !utf8.Valid(raw) {
		return recovery.ErrAdmissionBinding
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return recovery.ErrAdmissionBinding
	}
	seen := make(map[string]bool, len(keys))
	for d.More() {
		token, err = d.Token()
		key, ok := token.(string)
		if err != nil || !ok || seen[key] {
			return recovery.ErrAdmissionBinding
		}
		found := false
		for _, allowed := range keys {
			if key == allowed {
				found = true
				break
			}
		}
		if !found {
			return recovery.ErrAdmissionBinding
		}
		seen[key] = true
		var value json.RawMessage
		if d.Decode(&value) != nil || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return recovery.ErrAdmissionBinding
		}
	}
	if _, err = d.Token(); err != nil || len(seen) != len(keys) {
		return recovery.ErrAdmissionBinding
	}
	if _, err = d.Token(); err != io.EOF {
		return recovery.ErrAdmissionBinding
	}
	return nil
}

func receiptKeys() []string {
	return []string{"version", "run_id", "backup_ref", "deployment_id", "installation_id", "target_digest", "artifact_digest", "artifact_bytes", "source_approval_digest", "restored_schema_digest", "restored_at", "receipt_digest"}
}

func decodeRestoreReceipt(raw []byte, p recovery.RestorePlan) (recovery.RestoreReceipt, error) {
	var w restoreReceiptWire
	if err := strictRestoreObject(raw, receiptKeys()); err != nil {
		return recovery.RestoreReceipt{}, err
	}
	if json.Unmarshal(raw, &w) != nil {
		return recovery.RestoreReceipt{}, recovery.ErrAdmissionBinding
	}
	at, err := time.Parse(restoreTimeLayout, w.RestoredAt)
	if err != nil || at.IsZero() || at.Format(restoreTimeLayout) != w.RestoredAt {
		return recovery.RestoreReceipt{}, recovery.ErrAdmissionBinding
	}
	x := recovery.RestoreReceipt{Version: w.Version, RunID: w.RunID, BackupRef: w.BackupRef, DeploymentID: w.DeploymentID, InstallationID: w.InstallationID, TargetDigest: w.TargetDigest, ArtifactDigest: w.ArtifactDigest, ArtifactBytes: w.ArtifactBytes, SourceApprovalDigest: w.SourceApprovalDigest, RestoredSchemaDigest: w.RestoredSchemaDigest, RestoredAt: at, ReceiptDigest: w.ReceiptDigest}
	if !receiptMatchesPlan(x, p) || w.ReceiptDigest != restoreReceiptDigest(w) {
		return recovery.RestoreReceipt{}, recovery.ErrAdmissionBinding
	}
	return x, nil
}

func restoreReceiptDigest(w restoreReceiptWire) string {
	w.ReceiptDigest = ""
	raw, _ := json.Marshal(w)
	return recovery.SHA256Digest(raw)
}

func receiptMatchesPlan(x recovery.RestoreReceipt, p recovery.RestorePlan) bool {
	return x.Version == restoreVersion && x.Version == p.Version && x.RunID == p.RunID && x.BackupRef == p.BackupRef && x.DeploymentID == p.DeploymentID && x.InstallationID == p.InstallationID && x.TargetDigest == p.TargetDigest && x.ArtifactDigest == p.ArtifactDigest && x.ArtifactBytes == p.ArtifactBytes && x.ArtifactBytes > 0 && x.SourceApprovalDigest == p.SourceApprovalDigest && x.RestoredSchemaDigest == p.SourceSchemaDigest
}

// Error presentation never includes provider text; errors.Is retains the cause.
type restorePrivateFailure struct{ cause error }

func (e restorePrivateFailure) Error() string { return "recovery restore dependency failure" }
func (e restorePrivateFailure) Unwrap() error { return e.cause }
func privateRestoreError(err error) error {
	if err == nil {
		return nil
	}
	return restorePrivateFailure{err}
}

type restorePhaseAuthority interface {
	CheckPhase(context.Context, recovery.PhasePermit, string) error
	ReadPermitBinding(context.Context) (recovery.PermitBinding, error)
}

// These ports require actual approved facilities. Test implementations stay in tests.
type restoreSource interface {
	OpenApproved(context.Context, recovery.RunBinding) (recovery.RestorePlan, *os.File, error)
	CheckApproved(context.Context, recovery.RestorePlan, *os.File) error
}

type restoreTarget interface {
	ReadRestored(context.Context, recovery.RestorePlan) (recovery.RestoreReceipt, error)
}

type restoreTool interface {
	Preflight(context.Context) error
	Run(context.Context, string, recovery.RestorePlan, *os.File, []byte, restoreStartCheck) ([]byte, error)
}

type controlledRestoreDriver struct {
	mu        sync.Mutex
	authority restorePhaseAuthority
	ownership interface{ Check(context.Context) error }
	source    restoreSource
	target    restoreTarget
	tool      restoreTool
	dsn       string
	plan      recovery.RestorePlan
	file      *os.File
	attempted bool
	closed    bool
}

func newControlledRestoreDriver(source restoreSource, target restoreTarget, tool restoreTool, dsn string) (*controlledRestoreDriver, error) {
	if source == nil || target == nil || tool == nil || dsn == "" {
		return nil, recovery.ErrAdmissionUnsupported
	}
	return &controlledRestoreDriver{source: source, target: target, tool: tool, dsn: dsn}, nil
}

func (d *controlledRestoreDriver) Prepare(ctx context.Context, b recovery.RunBinding) (recovery.RestorePlan, error) {
	if err := restoreContext(ctx); err != nil {
		return recovery.RestorePlan{}, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed || d.source == nil || d.target == nil || d.tool == nil || d.ownership == nil || d.dsn == "" {
		return recovery.RestorePlan{}, recovery.ErrAdmissionUnsupported
	}
	if err := d.tool.Preflight(ctx); err != nil {
		return recovery.RestorePlan{}, privateRestoreError(err)
	}
	if d.file != nil {
		if b.RunID != d.plan.RunID || b.BackupRef != d.plan.BackupRef {
			return recovery.RestorePlan{}, recovery.ErrAdmissionBinding
		}
		return d.prepared(ctx)
	}
	p, f, err := d.source.OpenApproved(ctx, b)
	if err != nil {
		if f != nil {
			err = errors.Join(err, f.Close())
		}
		return recovery.RestorePlan{}, privateRestoreError(err)
	}
	if f == nil {
		return recovery.RestorePlan{}, recovery.ErrAdmissionIncomplete
	}
	d.plan, d.file = p, f
	p, err = d.prepared(ctx)
	if err != nil {
		err = errors.Join(err, d.file.Close())
		d.file = nil
	}
	return p, err
}

func (d *controlledRestoreDriver) prepared(ctx context.Context) (recovery.RestorePlan, error) {
	if err := d.source.CheckApproved(ctx, d.plan, d.file); err != nil {
		return recovery.RestorePlan{}, privateRestoreError(err)
	}
	if err := checkRestoreArtifact(ctx, d.plan, d.file); err != nil {
		return recovery.RestorePlan{}, err
	}
	return d.plan, nil
}

func checkRestoreArtifact(ctx context.Context, p recovery.RestorePlan, f *os.File) error {
	if err := restoreContext(ctx); err != nil {
		return err
	}
	before, err := f.Stat()
	if err != nil || !before.Mode().IsRegular() || before.Size() != p.ArtifactBytes || p.ArtifactBytes <= 0 {
		return recovery.ErrAdmissionBinding
	}
	current, err := os.Lstat(p.ArtifactPath)
	if err != nil || current.Mode()&os.ModeSymlink != 0 || !os.SameFile(before, current) {
		return recovery.ErrAdmissionDrift
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	hash := sha256.New()
	n, err := io.Copy(hash, &restoreContextReader{ctx: ctx, r: io.LimitReader(f, p.ArtifactBytes+1)})
	if err != nil {
		return err
	}
	after, err := f.Stat()
	if err != nil || n != p.ArtifactBytes || !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return recovery.ErrAdmissionDrift
	}
	current, err = os.Lstat(p.ArtifactPath)
	if err != nil || !os.SameFile(after, current) || p.ArtifactDigest != "sha256:"+hex.EncodeToString(hash.Sum(nil)) {
		return recovery.ErrAdmissionDrift
	}
	_, err = f.Seek(0, io.SeekStart)
	return errors.Join(err, ctx.Err())
}

type restoreContextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r *restoreContextReader) Read(b []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(b)
}

func (d *controlledRestoreDriver) checkPermit(ctx context.Context, p recovery.PhasePermit, plan recovery.RestorePlan) error {
	if d.authority == nil {
		return recovery.ErrAdmissionClosed
	}
	if err := d.authority.CheckPhase(ctx, p, recovery.PhaseRecoveryRestore); err != nil {
		return err
	}
	b, err := d.authority.ReadPermitBinding(ctx)
	if err != nil {
		return err
	}
	if b.Purpose != recovery.PhaseRecoveryRestore || b.RunID != plan.RunID || b.BackupRef != plan.BackupRef || b.InstallationID != plan.InstallationID {
		return recovery.ErrAdmissionBinding
	}
	return nil
}

func (d *controlledRestoreDriver) Replace(ctx context.Context, p recovery.PhasePermit, plan recovery.RestorePlan) (recovery.RestoreReceipt, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.checkPermit(ctx, p, plan); err != nil {
		return recovery.RestoreReceipt{}, err
	}
	if d.closed || d.attempted || d.file == nil || plan != d.plan {
		return recovery.RestoreReceipt{}, recovery.ErrAdmissionBinding
	}
	if _, err := d.prepared(ctx); err != nil {
		return recovery.RestoreReceipt{}, err
	}
	d.attempted = true
	x, err := d.execute(ctx, "restore-v1", func(start context.Context) error { return d.checkPermit(start, p, plan) })
	if err != nil {
		return recovery.RestoreReceipt{}, errors.Join(recovery.ErrRecoveryWriteUnknown, err)
	}
	if err = d.checkPermit(ctx, p, plan); err != nil {
		return recovery.RestoreReceipt{}, errors.Join(recovery.ErrRecoveryWriteUnknown, err)
	}
	return x, nil
}

func (d *controlledRestoreDriver) execute(ctx context.Context, verb string, check restoreStartCheck) (recovery.RestoreReceipt, error) {
	input, err := privateRestoreInput(d.plan, d.dsn)
	if err != nil {
		return recovery.RestoreReceipt{}, recovery.ErrAdmissionBinding
	}
	defer clear(input)
	raw, err := d.tool.Run(ctx, verb, d.plan, d.file, input, check)
	if err != nil {
		return recovery.RestoreReceipt{}, privateRestoreError(err)
	}
	defer clear(raw)
	if _, err = d.prepared(ctx); err != nil {
		return recovery.RestoreReceipt{}, err
	}
	x, err := decodeRestoreReceipt(raw, d.plan)
	if err != nil {
		return recovery.RestoreReceipt{}, err
	}
	independent, err := d.target.ReadRestored(ctx, d.plan)
	if err != nil || independent != x {
		return recovery.RestoreReceipt{}, errors.Join(recovery.ErrRecoveryWriteUnknown, privateRestoreError(err))
	}
	return x, ctx.Err()
}

func (d *controlledRestoreDriver) VerifyRestored(ctx context.Context, p recovery.RestorePlan) (recovery.RestoreReceipt, error) {
	if err := restoreContext(ctx); err != nil {
		return recovery.RestoreReceipt{}, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed || d.file == nil || p != d.plan {
		return recovery.RestoreReceipt{}, recovery.ErrAdmissionBinding
	}
	if _, err := d.prepared(ctx); err != nil {
		return recovery.RestoreReceipt{}, err
	}
	if d.ownership == nil {
		return recovery.RestoreReceipt{}, recovery.ErrAdmissionClosed
	}
	return d.execute(ctx, "verify-v1", d.ownership.Check)
}

// Close joins current calls through mu. The future root must also close idle owners.
func (d *controlledRestoreDriver) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil
	}
	d.closed = true
	d.dsn = ""
	if d.file != nil {
		return d.file.Close()
	}
	return nil
}

// restoreProcessTool is an unregistered runner, requiring explicit trusted image
// and same-handle handoff checks. Standard os/exec alone proves neither facility.
type restoreProcessTool struct {
	cleanupMu          sync.Mutex
	cleanupFailure     error
	executable, digest string
	environment        []string
	approveImage       func(context.Context, string, *os.File) error
	acquireImage       func(context.Context, string, string, *os.File) (restoreImageLease, error)
	useSource          func(context.Context, string, *os.File) error
	observed           func(restoreProcessObservation)
	inputClosed        func()
	outputSeen         func()
}

type restoreProcessObservation struct {
	PID                 int
	Waited, PipesJoined bool
}

// restoreStartCheck borrows the existing phase/lease authority; it mints nothing.
type restoreStartCheck func(context.Context) error

// restoreImageLease binds approval to the executable used by Start, retaining
// protection through that call. Start must call check at its concrete launch
// boundary. On error it returns any started cmd for the caller's actual Wait.
// CloseAndJoin returns only after its image/IO ownership is terminal, even on
// error; an unknown release must remain joined, not be reported as completed.
// No native implementation or replaceable-path fallback is registered here.
type restoreImageLease interface {
	Start(context.Context, string, []string, *restoreChildPipes, restoreStartCheck) (*exec.Cmd, error)
	CloseAndJoin() error
}

type restoreCheckedImage struct {
	file  *os.File
	lease restoreImageLease
}

func (i *restoreCheckedImage) closeAndJoin() (result error) {
	if i == nil {
		return nil
	}
	// The lease borrows this file until its join; its panic must still unwind
	// our descriptor, while its original panic value continues to propagate.
	defer func() { result = errors.Join(result, i.file.Close()) }()
	if i.lease != nil {
		return i.lease.CloseAndJoin()
	}
	return nil
}

func (t *restoreProcessTool) imageConfiguration(ctx context.Context) error {
	if err := restoreContext(ctx); err != nil {
		return err
	}
	if t == nil || t.approveImage == nil || t.acquireImage == nil || t.useSource == nil {
		return recovery.ErrAdmissionUnsupported
	}
	if !filepath.IsAbs(t.executable) || filepath.Clean(t.executable) != t.executable {
		return recovery.ErrAdmissionBinding
	}
	for _, entry := range t.environment {
		key, _, ok := strings.Cut(entry, "=")
		if !ok || (key != "SystemRoot" && key != "WINDIR") {
			return recovery.ErrAdmissionBinding
		}
	}
	return nil
}

func (t *restoreProcessTool) openImage(ctx context.Context) (image *restoreCheckedImage, result error) {
	if err := t.imageConfiguration(ctx); err != nil {
		return nil, err
	}
	f, err := os.Open(t.executable)
	if err != nil {
		return nil, recovery.ErrAdmissionIncomplete
	}
	owner := &restoreCheckedImage{file: f}
	// Named return transfers owner (including owner+error) exactly once. Until
	// then this acquisition owns cleanup, before either external callback runs.
	// A panicking acquire has not returned a lease; its unreturned resources
	// remain the provider's responsibility, while this checked file is ours.
	defer func() {
		if image == nil {
			result = errors.Join(result, owner.closeAndJoin())
		}
	}()
	hash := sha256.New()
	if _, err = io.Copy(hash, &restoreContextReader{ctx: ctx, r: f}); err != nil {
		return owner, err
	}
	if "sha256:"+hex.EncodeToString(hash.Sum(nil)) != t.digest {
		return owner, recovery.ErrAdmissionBinding
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return owner, err
	}
	if err = t.approveImage(ctx, t.executable, f); err != nil {
		return owner, err
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return owner, err
	}
	owner.lease, err = t.acquireImage(ctx, t.executable, t.digest, f)
	if err == nil && owner.lease == nil {
		err = recovery.ErrAdmissionUnsupported
	}
	return owner, err
}

func (t *restoreProcessTool) Preflight(ctx context.Context) (result error) {
	image, err := t.openImage(ctx)
	defer func() { result = errors.Join(result, image.closeAndJoin()) }()
	return err
}

func (t *restoreProcessTool) Run(ctx context.Context, verb string, p recovery.RestorePlan, f *os.File, input []byte, check restoreStartCheck) (raw []byte, result error) {
	if (verb != "restore-v1" && verb != "verify-v1") || check == nil {
		return nil, recovery.ErrAdmissionBinding
	}
	image, err := t.openImage(ctx)
	defer func() { t.finishImage(ctx, image, &raw, &result, recover()) }()
	if err != nil {
		return nil, err
	}
	if err = checkRestoreArtifact(ctx, p, f); err != nil {
		return nil, err
	}
	if err = t.useSource(ctx, verb, f); err != nil {
		return nil, err
	}
	if err = check(ctx); err != nil {
		return nil, err
	}
	return t.runChild(ctx, verb, input, image.lease, check)
}

// Panic values remain opaque: their text is never used for diagnostics.
type restoreCleanupPanic struct{ value any }

func (p restoreCleanupPanic) Error() string { return "recovery restore cleanup panic" }
func (p restoreCleanupPanic) Unwrap() error {
	cause, _ := p.value.(error)
	return cause
}

// Secondary failures are private to this tool and drained by its owner.
func (t *restoreProcessTool) retainCleanupFailure(cause error) {
	if cause == nil {
		return
	}
	t.cleanupMu.Lock()
	defer t.cleanupMu.Unlock()
	t.cleanupFailure = errors.Join(t.cleanupFailure, privateRestoreError(cause))
}

func (t *restoreProcessTool) takeCleanupFailure() error {
	t.cleanupMu.Lock()
	defer t.cleanupMu.Unlock()
	cause := t.cleanupFailure
	t.cleanupFailure = nil
	return cause
}

func captureRestoreImageCleanup(image *restoreCheckedImage) (cause error, raised any) {
	defer func() { raised = recover() }()
	cause = image.closeAndJoin()
	return cause, nil
}

func (t *restoreProcessTool) finishImage(ctx context.Context, image *restoreCheckedImage, raw *[]byte, result *error, primary any) {
	cause, raised := captureRestoreImageCleanup(image)
	if primary != nil || cause != nil || raised != nil {
		clear(*raw)
		*raw = nil
	}
	if primary != nil {
		if raised != nil {
			cause = errors.Join(cause, restoreCleanupPanic{raised})
		}
		t.retainCleanupFailure(cause)
		panic(primary)
	}
	if raised != nil {
		panic(raised)
	}
	if cause != nil {
		*result = errors.Join(*result, cause, ctx.Err())
	}
}

type restoreChildPipes struct{ inR, inW, outR, outW, errR, errW *os.File }

func newRestoreChildPipes() (*restoreChildPipes, error) {
	p := &restoreChildPipes{}
	var err error
	p.inR, p.inW, err = os.Pipe()
	if err == nil {
		p.outR, p.outW, err = os.Pipe()
	}
	if err == nil {
		p.errR, p.errW, err = os.Pipe()
	}
	if err != nil {
		p.close()
		return nil, err
	}
	return p, nil
}
func (p *restoreChildPipes) close() {
	for _, f := range []*os.File{p.inR, p.inW, p.outR, p.outW, p.errR, p.errW} {
		if f != nil {
			f.Close()
		}
	}
}

type restoreOutput struct {
	raw []byte
	err error
}

func (t *restoreProcessTool) readOutput(r *os.File, kill func()) restoreOutput {
	defer r.Close()
	raw := make([]byte, 0, restoreReceiptLimit+1)
	buffer := make([]byte, 4096)
	for len(raw) <= restoreReceiptLimit {
		n, err := r.Read(buffer[:min(len(buffer), restoreReceiptLimit+1-len(raw))])
		if n > 0 {
			raw = append(raw, buffer[:n]...)
			if t.outputSeen != nil {
				t.outputSeen()
			}
		}
		if len(raw) > restoreReceiptLimit {
			kill()
			return restoreOutput{err: recovery.ErrAdmissionBinding}
		}
		if err == io.EOF {
			return restoreOutput{raw: raw}
		}
		if err != nil {
			return restoreOutput{err: recovery.ErrRecoveryWriteUnknown}
		}
	}
	return restoreOutput{err: recovery.ErrAdmissionBinding}
}

func (t *restoreProcessTool) runChild(ctx context.Context, verb string, input []byte, image restoreImageLease, check restoreStartCheck) ([]byte, error) {
	pipes, err := newRestoreChildPipes()
	if err != nil {
		return nil, recovery.ErrAdmissionIncomplete
	}
	defer pipes.close()
	cmd, startErr := image.Start(ctx, verb, append([]string{}, t.environment...), pipes, check)
	if cmd == nil || cmd.Process == nil {
		return nil, errors.Join(recovery.ErrRecoveryWriteUnknown, privateRestoreError(startErr), ctx.Err())
	}
	pipes.childStarted()
	var once sync.Once
	kill := func() { once.Do(func() { cmd.Process.Kill() }) }
	if startErr != nil {
		kill()
	}
	killDone := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { defer close(killDone); kill() })
	defer func() {
		if !stop() {
			<-killDone
		}
	}()
	writeDone := make(chan error, 1)
	outputDone := make(chan restoreOutput, 1)
	stderrDone := make(chan error, 1)
	go func() {
		_, e := pipes.inW.Write(input)
		if t.inputClosed != nil {
			t.inputClosed()
		}
		pipes.inW.Close()
		writeDone <- e
	}()
	go func() { outputDone <- t.readOutput(pipes.outR, kill) }()
	go func() { io.Copy(io.Discard, pipes.errR); pipes.errR.Close(); close(stderrDone) }()
	waitErr := cmd.Wait()
	output := <-outputDone
	writeErr := <-writeDone
	stderrErr := <-stderrDone
	if t.observed != nil {
		t.observed(restoreProcessObservation{cmd.Process.Pid, true, true})
	}
	if startErr != nil || waitErr != nil || writeErr != nil || stderrErr != nil || output.err != nil || ctx.Err() != nil {
		clear(output.raw)
		return nil, errors.Join(recovery.ErrRecoveryWriteUnknown, privateRestoreError(startErr), output.err, ctx.Err())
	}
	return output.raw, nil
}

func (p *restoreChildPipes) childStarted() {
	p.inR.Close()
	p.outW.Close()
	p.errW.Close()
}
