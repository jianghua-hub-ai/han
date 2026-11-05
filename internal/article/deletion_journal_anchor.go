package article

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"

	"han/internal/recovery"
)

// AnchoredDeletionJournal exposes committed deletion facts from one installed journal.
type AnchoredDeletionJournal interface {
	ArticleDeletionJournal
	VerifiedDeletionJournal
	ReadVerifiedSnapshot(context.Context, string) (recovery.VerifiedJournalSnapshot, error)
	CoordinateAnchor(context.Context, recovery.PhasePermit) error
	Close() error
}

// ArticleAppendRuntime is the shared runtime's scoped append boundary.
type ArticleAppendRuntime interface {
	CheckJournalUse(context.Context, string, string) error
	CheckArticleCoordination(context.Context, recovery.PhasePermit, string, string, uint64) (recovery.AnchorSnapshot, string, error)
	PauseArticleAppend(context.Context, string) (recovery.AppendBarrier, error)
	CompleteArticleAppend(context.Context, recovery.AppendBarrier, recovery.VerifiedHeadAdvance) error
	FaultArticleAppend(context.Context, recovery.AppendBarrier, error) error
}

// AnchoredJournalOptions binds installed article media to its independent anchor.
type AnchoredJournalOptions struct {
	Directory                    string
	FileMode                     fs.FileMode
	InstallationID               string
	JournalID                    string
	Epoch                        uint64
	Anchor                       recovery.JournalHeadAnchor
	Runtime                      ArticleAppendRuntime
	ReadMaxBytes, ReadMaxRecords int64
}

// OpenAnchoredDeletionJournalForCoordinate opens only an installed namespace.
func OpenAnchoredDeletionJournalForCoordinate(ctx context.Context, permit recovery.PhasePermit, options AnchoredJournalOptions, keys DeletionJournalKeyring) (AnchoredDeletionJournal, error) {
	if _, _, err := deletionCoordinationOrigin(ctx, permit, options); err != nil {
		return nil, err
	}
	if !articleDeletionJournalPlatformSupported() {
		return nil, ErrDeletionJournalUnsupportedPlatform
	}
	media, err := newDeletionAnchorMedia(ctx, options)
	if err != nil {
		return nil, err
	}
	return newAnchoredDeletionJournalForCoordinate(ctx, permit, options, keys, media)
}

func deletionCoordinationOrigin(ctx context.Context, permit recovery.PhasePermit, options AnchoredJournalOptions) (recovery.AnchorSnapshot, string, error) {
	if err := deletionAnchorContext(ctx); err != nil {
		return recovery.AnchorSnapshot{}, "", err
	}
	if missingDeletionAnchorPort(options.Runtime) {
		return recovery.AnchorSnapshot{}, "", recovery.ErrAdmissionBinding
	}
	return options.Runtime.CheckArticleCoordination(ctx, permit, options.InstallationID, options.JournalID, options.Epoch)
}

func newAnchoredDeletionJournalForCoordinate(ctx context.Context, permit recovery.PhasePermit, o AnchoredJournalOptions, keys DeletionJournalKeyring, media deletionAnchorMedia) (journal *anchoredArticleJournal, result error) {
	origin, revision, err := deletionCoordinationOrigin(ctx, permit, o)
	if err != nil {
		return nil, err
	}
	if err := validateAnchoredOptions(ctx, o, media); err != nil {
		return nil, err
	}
	if err := validateDeletionCoordinationOrigin(o, origin, revision); err != nil {
		return nil, err
	}
	retained, err := deletionJournalKeys(keys)
	if err != nil {
		return nil, err
	}
	j := &anchoredArticleJournal{options: o, media: media, gate: make(chan struct{}, 1), records: &FileArticleDeletionJournal{active: cloneDeletionJournalKey(keys.Active), keys: retained}}
	j.control = &deletionAnchorControl{ctx: ctx, permit: permit, origin: copyDeletionAnchor(origin), revision: revision, current: copyDeletionAnchor(origin), token: revision}
	opened := false
	defer func() {
		value := recover()
		if !opened {
			closeErr, closePanic := closeDeletionControl(j)
			result = errors.Join(result, closeErr, deletionNamespacePanicCause(closePanic))
			if value == nil {
				value = closePanic
			}
		}
		if value != nil {
			panic(value)
		}
	}()
	lease, err := media.lock(ctx, true)
	j.control.owner.lease = lease
	if err != nil || missingDeletionAnchorPort(lease) {
		return nil, errors.Join(recovery.ErrAdmissionClosed, err)
	}
	if err := j.checkControl(ctx); err != nil {
		return nil, err
	}
	opened = true
	return j, nil
}

func closeDeletionControl(j *anchoredArticleJournal) (result error, panicValue any) {
	defer func() { panicValue = recover() }()
	return j.Close(), nil
}

func validateDeletionCoordinationOrigin(o AnchoredJournalOptions, a recovery.AnchorSnapshot, token string) error {
	if a.Namespace != ArticleDeletionNamespace || a.JournalID != o.JournalID || a.Epoch != o.Epoch || token == "" || strings.TrimSpace(token) != token || !validDeletionAnchorHead(a.Committed) {
		return recovery.ErrAdmissionBinding
	}
	if p := a.Pending; p != nil {
		parts := strings.Split(p.OperationID, ":")
		if p.OldHead != a.Committed || p.OldHead.Sequence >= math.MaxInt64 || p.NewHead.Sequence != p.OldHead.Sequence+1 || !validDeletionAnchorHead(p.NewHead) || len(parts) != 3 || parts[0] != "article-append/v1" || !validUUID(parts[1]) || !validDeletionAnchorHead(recovery.JournalHead{Sequence: 1, Hash: "sha256:" + parts[2]}) {
			return recovery.ErrAdmissionBinding
		}
	}
	return nil
}

type deletionAnchorControl struct {
	ctx             context.Context
	permit          recovery.PhasePermit
	origin, current recovery.AnchorSnapshot
	revision, token string
	owner           deletionNamespaceOwner
	failed          bool
}

type deletionAnchorMedia struct {
	lock     func(context.Context, bool) (deletionNamespaceLease, error)
	check    func(context.Context) error
	names    func(context.Context) ([]string, error)
	open     func(context.Context, string) (*os.File, error)
	create   func(context.Context, string, fs.FileMode) (*os.File, error)
	write    func(context.Context, *os.File, []byte) error
	syncFile func(context.Context, *os.File) error
	syncDir  func(context.Context) error
	link     func(context.Context, string, string) error
	unlink   func(context.Context, string, string, []byte) error
}

// Every WaitReleased return proves the same resource's I/O and ownership ended.
// Cancellation records failure; it never permits an early resource-join return.
type deletionNamespaceLease interface {
	recovery.ExclusiveLease
	WaitReleased(context.Context) error
}

type anchoredArticleJournal struct {
	options  AnchoredJournalOptions
	records  *FileArticleDeletionJournal
	media    deletionAnchorMedia
	gate     chan struct{}
	closed   bool
	closing  atomic.Bool
	closeErr error
	control  *deletionAnchorControl
}

func newAnchoredDeletionJournal(ctx context.Context, options AnchoredJournalOptions, keys DeletionJournalKeyring, media deletionAnchorMedia) (*anchoredArticleJournal, error) {
	if err := validateAnchoredOptions(ctx, options, media); err != nil {
		return nil, err
	}
	retained, err := deletionJournalKeys(keys)
	if err != nil {
		return nil, err
	}
	j := &anchoredArticleJournal{options: options, media: media, gate: make(chan struct{}, 1),
		records: &FileArticleDeletionJournal{active: cloneDeletionJournalKey(keys.Active), keys: retained}}
	opened := false
	defer func() {
		if !opened {
			_ = j.records.Close()
		}
	}()
	if _, err := j.ReadVerifiedSnapshot(ctx, ArticleDeletionNamespace); err != nil {
		return nil, err
	}
	opened = true
	return j, nil
}

func (j *anchoredArticleJournal) Append(ctx context.Context, request DeletionTombstoneRequest) (record ArticleDeletionTombstone, result error) {
	if err := validateTombstoneRequest(request); err != nil {
		return record, err
	}
	completion := &deletionAppendCompletion{operation: request.DeletionOperationID}
	result = j.withLock(ctx, true, func() error {
		snapshot, records, err := j.snapshot(ctx)
		if err != nil {
			return err
		}
		for _, existing := range records {
			if existing.DeletionOperationID != request.DeletionOperationID {
				continue
			}
			if existing.ArticleID != request.ArticleID || existing.ArticleVersion != request.ArticleVersion || !existing.DeletedAt.Equal(request.DeletedAt.UTC()) {
				return ErrDeletionJournalCorrupt
			}
			if err := j.options.Runtime.CheckJournalUse(ctx, request.DeletionOperationID, recovery.JournalUseCommittedRead); err != nil {
				return err
			}
			record = existing
			return nil
		}
		if err := j.options.Runtime.CheckJournalUse(ctx, request.DeletionOperationID, recovery.JournalUseNewAppend); err != nil {
			return err
		}
		record, err = j.appendNew(ctx, request, snapshot, records, completion)
		return err
	}, completion)
	if result != nil {
		return ArticleDeletionTombstone{}, result
	}
	return record, nil
}

// CoordinateAnchor completes only the exact pending operation sealed by Install.
func (j *anchoredArticleJournal) CoordinateAnchor(ctx context.Context, permit recovery.PhasePermit) (result error) {
	if j == nil || j.control == nil {
		return recovery.ErrAdmissionClosed
	}
	if j.closing.Load() {
		return recovery.ErrAdmissionClosed
	}
	if err := deletionAnchorContext(ctx); err != nil {
		return err
	}
	select {
	case j.gate <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-j.gate }()
	defer j.failControl(&result)
	origin, token, err := deletionCoordinationOrigin(ctx, permit, j.options)
	if err != nil {
		return err
	}
	if !sameDeletionAnchor(origin, j.control.origin) || token != j.control.revision {
		return recovery.ErrAdmissionDrift
	}
	if err := j.checkControl(ctx); err != nil {
		return err
	}
	if j.control.current.Pending != nil {
		if err := j.coordinatePending(ctx); err != nil {
			return err
		}
	}
	_, _, err = j.snapshot(ctx)
	if err != nil {
		return err
	}
	return j.checkControl(ctx)
}

// Close waits for this instance's actual I/O to finish before erasing its keys.
func (j *anchoredArticleJournal) Close() error {
	if j == nil {
		return nil
	}
	j.closing.Store(true)
	j.gate <- struct{}{}
	defer func() { <-j.gate }()
	if j.closed {
		return j.closeErr
	}
	j.closed = true
	if j.control != nil {
		value := j.control.owner.attempt()
		j.closeErr = errors.Join(j.control.owner.closeErr, j.control.owner.join(context.Background()), j.records.Close())
		if value != nil {
			panic(value)
		}
		return j.closeErr
	}
	j.closeErr = j.records.Close()
	return j.closeErr
}

func (j *anchoredArticleJournal) failControl(result *error) {
	if value := recover(); value != nil {
		j.control.failed = true
		panic(value)
	}
	if *result != nil {
		j.control.failed = true
	}
}

func (j *anchoredArticleJournal) checkControl(ctx context.Context) error {
	c := j.control
	if j.closed || j.closing.Load() || c.failed {
		return recovery.ErrAdmissionClosed
	}
	if err := deletionAnchorContext(ctx); err != nil {
		return err
	}
	origin, token, err := deletionCoordinationOrigin(c.ctx, c.permit, j.options)
	if err != nil {
		return err
	}
	if !sameDeletionAnchor(origin, c.origin) || token != c.revision {
		return recovery.ErrAdmissionDrift
	}
	if err := c.owner.lease.Check(ctx); err != nil {
		return err
	}
	if err := j.media.check(ctx); err != nil {
		return err
	}
	a, revision, err := j.options.Anchor.Load(ctx, ArticleDeletionNamespace)
	if err != nil {
		return err
	}
	if !sameDeletionAnchor(a, c.current) || revision != c.token {
		return recovery.ErrAdmissionDrift
	}
	return deletionAnchorContext(ctx)
}

func (j *anchoredArticleJournal) controlRead(ctx context.Context, write bool, fn func() error) (result error) {
	if write {
		return recovery.ErrAdmissionClosed
	}
	defer j.failControl(&result)
	if err := j.checkControl(ctx); err != nil {
		return err
	}
	if err := fn(); err != nil {
		return err
	}
	return j.checkControl(ctx)
}

func (j *anchoredArticleJournal) coordinatePending(ctx context.Context) error {
	c := j.control
	inventory, err := j.coordinationInventory(ctx)
	if err != nil {
		return err
	}
	if err := j.checkControl(ctx); err != nil {
		return err
	}
	next := copyDeletionAnchor(c.current)
	if inventory.candidate != nil {
		if err := j.publishCoordinated(ctx, inventory); err != nil {
			return err
		}
		next.Committed = c.current.Pending.NewHead
	}
	next.Pending = nil
	if err := j.checkControl(ctx); err != nil {
		return err
	}
	revision, err := j.casAndLoad(ctx, c.token, next)
	if err != nil {
		return err
	}
	if revision == c.revision {
		return recovery.ErrAnchorIndeterminate
	}
	c.current, c.token = copyDeletionAnchor(next), revision
	return j.checkControl(ctx)
}

func (j *anchoredArticleJournal) publishCoordinated(ctx context.Context, v deletionCoordinationInventory) error {
	if err := j.controlIO(ctx, func() error { return j.syncCoordinationCandidate(ctx, v) }); err != nil {
		return err
	}
	if v.stage != "" && !v.published {
		if err := j.controlIO(ctx, func() error { return j.media.link(ctx, v.stage, v.target) }); err != nil {
			return err
		}
	}
	if err := j.controlIO(ctx, func() error { return j.media.syncDir(ctx) }); err != nil {
		return err
	}
	if v.stage != "" {
		if err := j.controlIO(ctx, func() error { return j.media.unlink(ctx, v.stage, v.target, v.candidate) }); err != nil {
			return err
		}
		if err := j.controlIO(ctx, func() error { return j.media.syncDir(ctx) }); err != nil {
			return err
		}
	}
	records, err := j.readRecords(ctx)
	if err != nil || deletionAnchorHead(records) != j.control.current.Pending.NewHead {
		return errors.Join(recovery.ErrAdmissionDrift, err)
	}
	return j.checkControl(ctx)
}

func (j *anchoredArticleJournal) syncCoordinationCandidate(ctx context.Context, v deletionCoordinationInventory) (result error) {
	name := v.target
	if v.stage != "" {
		name = v.stage
	}
	file, err := j.media.open(ctx, name)
	if file != nil {
		defer func() { result = errors.Join(result, file.Close()) }()
	}
	if err != nil || file == nil {
		return errors.Join(recovery.ErrAdmissionIncomplete, err)
	}
	if err := j.checkControl(ctx); err != nil {
		return err
	}
	return j.media.syncFile(ctx, file)
}

func (j *anchoredArticleJournal) controlIO(ctx context.Context, fn func() error) error {
	if err := j.checkControl(ctx); err != nil {
		return err
	}
	if err := fn(); err != nil {
		return err
	}
	return j.checkControl(ctx)
}

// OpenAnchoredDeletionJournal verifies installed media without initialization.
func OpenAnchoredDeletionJournal(ctx context.Context, options AnchoredJournalOptions, keys DeletionJournalKeyring) (AnchoredDeletionJournal, error) {
	if ctx == nil {
		return nil, recovery.ErrAdmissionBinding
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !articleDeletionJournalPlatformSupported() {
		return nil, ErrDeletionJournalUnsupportedPlatform
	}
	media, err := newDeletionAnchorMedia(ctx, options)
	if err != nil {
		return nil, err
	}
	return newAnchoredDeletionJournal(ctx, options, keys, media)
}

func newDeletionAnchorMedia(ctx context.Context, o AnchoredJournalOptions) (deletionAnchorMedia, error) {
	check, err := bindDeletionAnchorDirectory(ctx, o.Directory)
	if err != nil {
		return deletionAnchorMedia{}, err
	}
	m := deletionAnchorMedia{check: check,
		lock: func(ctx context.Context, write bool) (deletionNamespaceLease, error) {
			return lockDeletionAnchor(ctx, filepath.Join(filepath.Dir(o.Directory), ".writer.lock"), write)
		},
		open: func(ctx context.Context, name string) (*os.File, error) {
			return openDeletionAnchorFile(ctx, o.Directory, name, false, o.FileMode)
		},
		create: func(ctx context.Context, name string, mode fs.FileMode) (*os.File, error) {
			return openDeletionAnchorFile(ctx, o.Directory, name, true, mode)
		},
		write:    writeDeletionAnchorBytes,
		syncFile: func(ctx context.Context, file *os.File) error { return file.Sync() },
		syncDir: func(ctx context.Context) error {
			if err := check(ctx); err != nil {
				return err
			}
			return syncDeletionJournalDirectory(o.Directory)
		},
		link: func(ctx context.Context, from, to string) error {
			return linkDeletionAnchor(ctx, o.Directory, from, to)
		},
		unlink: func(ctx context.Context, from, to string, raw []byte) error {
			return unlinkDeletionAnchorAlias(ctx, o.Directory, from, to, raw)
		},
	}
	m.names = func(ctx context.Context) ([]string, error) {
		// A controlled crash state may include one exact alias in addition to the
		// bounded record set. Normal parsing still rejects every extra entry.
		return deletionAnchorNames(ctx, o.Directory, o.ReadMaxRecords+1)
	}
	return m, nil
}

func bindDeletionAnchorDirectory(ctx context.Context, directory string) (func(context.Context) error, error) {
	if err := deletionAnchorContext(ctx); err != nil {
		return nil, err
	}
	if err := validateArticleDeletionPathComponents(directory, true, os.Lstat); err != nil {
		return nil, err
	}
	identity, err := os.Lstat(directory)
	if err != nil {
		return nil, err
	}
	check := func(ctx context.Context) error {
		if err := deletionAnchorContext(ctx); err != nil {
			return err
		}
		current, err := os.Lstat(directory)
		if err != nil || !current.IsDir() || current.Mode()&os.ModeSymlink != 0 || !os.SameFile(identity, current) || current.Mode().Perm()&0o077 != 0 {
			return errors.Join(recovery.ErrAdmissionDrift, err)
		}
		return nil
	}
	return check, nil
}

func deletionAnchorNames(ctx context.Context, directory string, limit int64) (names []string, result error) {
	if err := deletionAnchorContext(ctx); err != nil {
		return nil, err
	}
	file, err := os.Open(directory)
	if file != nil {
		defer func() { result = errors.Join(result, file.Close()) }()
	}
	if err != nil {
		return nil, err
	}
	for {
		if err := deletionAnchorContext(ctx); err != nil {
			return nil, err
		}
		entries, err := file.ReadDir(1)
		if errors.Is(err, io.EOF) {
			return names, nil
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.Type()&os.ModeSymlink != 0 || entry.IsDir() {
				return nil, ErrDeletionJournalCorrupt
			}
			names = append(names, entry.Name())
			if int64(len(names)) > limit {
				return nil, recovery.ErrAdmissionIncomplete
			}
		}
	}
}

func deletionAnchorContext(ctx context.Context) error {
	if ctx == nil {
		return recovery.ErrAdmissionBinding
	}
	return ctx.Err()
}

func missingDeletionAnchorPort(port any) bool {
	if port == nil {
		return true
	}
	v := reflect.ValueOf(port)
	switch v.Kind() {
	case reflect.Interface, reflect.Pointer, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return v.IsNil()
	}
	return false
}

func validateAnchoredOptions(ctx context.Context, o AnchoredJournalOptions, m deletionAnchorMedia) error {
	if err := deletionAnchorContext(ctx); err != nil {
		return err
	}
	if !filepath.IsAbs(o.Directory) || filepath.Clean(o.Directory) != o.Directory || strings.TrimSpace(o.Directory) != o.Directory ||
		strings.TrimSpace(o.JournalID) != o.JournalID || o.JournalID == "" || strings.TrimSpace(o.InstallationID) != o.InstallationID || o.InstallationID == "" || o.Epoch == 0 || o.Epoch > math.MaxInt64 ||
		o.FileMode == 0 || o.FileMode != o.FileMode.Perm() || o.FileMode.Perm()&0o077 != 0 ||
		o.ReadMaxBytes <= 0 || o.ReadMaxBytes >= int64(math.MaxInt) || o.ReadMaxRecords <= 0 || o.ReadMaxRecords >= int64(math.MaxInt) {
		return recovery.ErrAdmissionBinding
	}
	for _, port := range []any{o.Anchor, o.Runtime, m.lock, m.check, m.names, m.open, m.create, m.write, m.syncFile, m.syncDir, m.link, m.unlink} {
		if missingDeletionAnchorPort(port) {
			return recovery.ErrAdmissionBinding
		}
	}
	return nil
}

func (j *anchoredArticleJournal) withLock(ctx context.Context, write bool, fn func() error, completions ...*deletionAppendCompletion) (result error) {
	if err := deletionAnchorContext(ctx); err != nil {
		return err
	}
	if j == nil {
		return recovery.ErrAdmissionBinding
	}
	if j.closing.Load() {
		return recovery.ErrAdmissionClosed
	}
	select {
	case j.gate <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-j.gate }()
	if j.closed || j.closing.Load() {
		return recovery.ErrAdmissionClosed
	}
	if j.control != nil {
		return j.controlRead(ctx, write, fn)
	}
	var completion *deletionAppendCompletion
	if len(completions) > 0 {
		completion = completions[0]
	}
	lease, err := j.media.lock(ctx, write)
	owner := deletionNamespaceOwner{lease: lease}
	defer j.finalizeNamespace(ctx, &owner, completion, &result)
	if err != nil || missingDeletionAnchorPort(lease) {
		return errors.Join(recovery.ErrAdmissionClosed, err)
	}
	if err := lease.Check(ctx); err != nil {
		return err
	}
	if err := j.media.check(ctx); err != nil {
		return err
	}
	if err := fn(); err != nil {
		return err
	}
	return errors.Join(deletionAnchorContext(ctx), j.media.check(ctx), lease.Check(ctx))
}

type deletionNamespaceOwner struct {
	lease                       deletionNamespaceLease
	attempted, terminal, closed bool
	closeErr, joinErr           error
}

func (o *deletionNamespaceOwner) attempt() (panicValue any) {
	if o.attempted || missingDeletionAnchorPort(o.lease) {
		return nil
	}
	o.attempted = true
	defer func() {
		panicValue = recover()
		o.closeErr = errors.Join(o.closeErr, deletionNamespacePanicCause(panicValue))
	}()
	o.closeErr = o.lease.Close()
	return nil
}

func (o *deletionNamespaceOwner) join(ctx context.Context) error {
	if o.closed {
		return o.joinErr
	}
	if !missingDeletionAnchorPort(o.lease) {
		o.joinErr = o.lease.WaitReleased(ctx)
	}
	// A returned join, even with an error, is the private port's terminal proof.
	o.terminal = true
	o.closed = true
	return o.joinErr
}

func deletionNamespacePanicCause(value any) error {
	if value == nil {
		return nil
	}
	if err, ok := value.(error); ok {
		return err
	}
	return recovery.ErrRecoveryWriteUnknown
}

type deletionAppendCompletion struct {
	operation                string
	barrier                  recovery.AppendBarrier
	advance                  recovery.VerifiedHeadAdvance
	armed, accepted, faulted bool
}

func (c *deletionAppendCompletion) fault(j *anchoredArticleJournal, ctx context.Context, result *error) (panicValue any) {
	if c == nil || !c.armed || c.accepted || c.faulted || *result == nil {
		return nil
	}
	c.faulted = true
	defer func() {
		panicValue = recover()
		*result = errors.Join(*result, deletionNamespacePanicCause(panicValue))
	}()
	*result = errors.Join(recovery.ErrRecoveryWriteUnknown, *result,
		j.options.Runtime.FaultArticleAppend(ctx, c.barrier, errors.Join(*result, recovery.ErrRecoveryWriteUnknown)))
	return nil
}

func (j *anchoredArticleJournal) finalizeNamespace(ctx context.Context, owner *deletionNamespaceOwner, c *deletionAppendCompletion, result *error) {
	panicValue := recover()
	closePanic := owner.attempt()
	if panicValue == nil {
		panicValue = closePanic
	}
	*result = errors.Join(*result, owner.closeErr, deletionNamespacePanicCause(panicValue), deletionAnchorContext(ctx))
	faultPanic := c.fault(j, ctx, result)
	if panicValue == nil {
		panicValue = faultPanic
	}
	// Keep the original call/action and journal gate until actual terminal release.
	*result = errors.Join(*result, owner.join(ctx), deletionAnchorContext(ctx))
	faultPanic = c.fault(j, ctx, result)
	if panicValue == nil {
		panicValue = faultPanic
	}
	if panicValue != nil {
		panic(panicValue)
	}
	if *result == nil {
		*result = j.completeNamespaceAppend(ctx, c)
		if *result != nil {
			if value := c.fault(j, ctx, result); value != nil {
				panic(value)
			}
		}
	}
}

func (j *anchoredArticleJournal) completeNamespaceAppend(ctx context.Context, c *deletionAppendCompletion) error {
	if c == nil || !c.armed {
		return nil
	}
	defer c.faultCompletionPanic(j, ctx)
	// Check actual deployment ownership before the final synchronous acceptance.
	if err := j.options.Runtime.CheckJournalUse(ctx, c.operation, recovery.JournalUseNewAppend); err != nil {
		return err
	}
	if err := deletionAnchorContext(ctx); err != nil {
		return err
	}
	if err := j.options.Runtime.CompleteArticleAppend(ctx, c.barrier, c.advance); err != nil {
		return err
	}
	// No fallible port, resource release or cancellation check follows acceptance.
	c.accepted = true
	return nil
}

func (c *deletionAppendCompletion) faultCompletionPanic(j *anchoredArticleJournal, ctx context.Context) {
	if value := recover(); value != nil {
		cause := errors.Join(recovery.ErrRecoveryWriteUnknown, deletionNamespacePanicCause(value))
		c.fault(j, ctx, &cause)
		panic(value)
	}
}

type deletionAnchorAppend struct {
	before           recovery.VerifiedJournalSnapshot
	record           ArticleDeletionTombstone
	raw              []byte
	stage, operation string
	pending          recovery.AnchorSnapshot
	pendingRevision  string
}

func (j *anchoredArticleJournal) appendNew(ctx context.Context, request DeletionTombstoneRequest, before recovery.VerifiedJournalSnapshot, records []ArticleDeletionTombstone, c *deletionAppendCompletion) (record ArticleDeletionTombstone, result error) {
	a, err := j.prepareAppend(request, before, records)
	if err != nil {
		return record, err
	}
	c.barrier, err = j.options.Runtime.PauseArticleAppend(ctx, request.DeletionOperationID)
	c.armed = true
	if err != nil {
		return record, err
	}
	if err := j.publishAppend(ctx, a); err != nil {
		return record, err
	}
	c.advance, err = j.commitAppend(ctx, a)
	if err != nil {
		return record, err
	}
	return a.record, nil
}

func (j *anchoredArticleJournal) prepareAppend(request DeletionTombstoneRequest, before recovery.VerifiedJournalSnapshot, records []ArticleDeletionTombstone) (*deletionAnchorAppend, error) {
	if before.Head.Sequence >= math.MaxInt64 || int64(len(records)) >= j.options.ReadMaxRecords {
		return nil, recovery.ErrAdmissionIncomplete
	}
	record := j.records.newTombstone(request, records)
	raw, err := json.Marshal(deletionDiskRecordFromModel(record))
	if err != nil || int64(len(raw)) > j.options.ReadMaxBytes {
		return nil, errors.Join(recovery.ErrAdmissionIncomplete, err)
	}
	total := int64(len(raw))
	for _, previous := range records {
		encoded, err := json.Marshal(deletionDiskRecordFromModel(previous))
		if err != nil || int64(len(encoded)) > j.options.ReadMaxBytes-total {
			return nil, errors.Join(recovery.ErrAdmissionIncomplete, err)
		}
		total += int64(len(encoded))
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return nil, err
	}
	id[6], id[8] = id[6]&0x0f|0x40, id[8]&0x3f|0x80
	uuid := fmt.Sprintf("%x-%x-%x-%x-%x", id[:4], id[4:6], id[6:8], id[8:10], id[10:])
	sum := sha256.Sum256(raw)
	operation := "article-append/v1:" + uuid + ":" + hex.EncodeToString(sum[:])
	next := recovery.JournalHead{Sequence: uint64(record.Sequence), Hash: deletionRecordHash(record)}
	pending := recovery.AnchorSnapshot{JournalID: j.options.JournalID, Namespace: ArticleDeletionNamespace, Epoch: j.options.Epoch, Committed: before.Head,
		Pending: &recovery.AnchorPending{OldHead: before.Head, NewHead: next, OperationID: operation}}
	return &deletionAnchorAppend{before: before, record: record, raw: raw, stage: ".article-deletion.stage-" + uuid, operation: operation, pending: pending}, nil
}

func (j *anchoredArticleJournal) writeStage(ctx context.Context, a *deletionAnchorAppend) (result error) {
	if err := deletionAnchorContext(ctx); err != nil {
		return err
	}
	file, err := j.media.create(ctx, a.stage, j.options.FileMode)
	if file != nil {
		defer func() { result = errors.Join(result, file.Close()) }()
	}
	if err != nil || file == nil {
		return errors.Join(recovery.ErrAdmissionIncomplete, err)
	}
	if err := j.media.write(ctx, file, a.raw); err != nil {
		return err
	}
	if err := deletionAnchorContext(ctx); err != nil {
		return err
	}
	return j.media.syncFile(ctx, file)
}

func writeDeletionAnchorBytes(ctx context.Context, file *os.File, raw []byte) error {
	if err := deletionAnchorContext(ctx); err != nil {
		return err
	}
	n, err := file.Write(raw)
	if err != nil || n != len(raw) {
		return errors.Join(io.ErrShortWrite, err)
	}
	return deletionAnchorContext(ctx)
}

func (j *anchoredArticleJournal) publishAppend(ctx context.Context, a *deletionAnchorAppend) error {
	if err := j.writeStage(ctx, a); err != nil {
		return err
	}
	if err := j.media.syncDir(ctx); err != nil {
		return err
	}
	revision, err := j.casAndLoad(ctx, a.before.Revision, a.pending)
	if err != nil {
		return err
	}
	a.pendingRevision = revision
	target := deletionRecordName(a.record.Sequence)
	if err := j.media.link(ctx, a.stage, target); err != nil {
		return err
	}
	if err := j.media.syncDir(ctx); err != nil {
		return err
	}
	if err := j.media.unlink(ctx, a.stage, target, a.raw); err != nil {
		return err
	}
	if err := j.media.syncDir(ctx); err != nil {
		return err
	}
	records, err := j.readRecords(ctx)
	if err != nil || deletionAnchorHead(records) != a.pending.Pending.NewHead {
		return errors.Join(recovery.ErrAdmissionDrift, err)
	}
	return nil
}

func (j *anchoredArticleJournal) casAndLoad(ctx context.Context, previous string, next recovery.AnchorSnapshot) (string, error) {
	if err := deletionAnchorContext(ctx); err != nil {
		return "", err
	}
	expected := copyDeletionAnchor(next)
	returned, token, writeErr := j.options.Anchor.CompareAndSwap(ctx, ArticleDeletionNamespace, previous, copyDeletionAnchor(expected))
	// Every attempted CAS is followed by an independent read, including ACK loss.
	observed, revision, readErr := j.options.Anchor.Load(ctx, ArticleDeletionNamespace)
	if writeErr != nil || readErr != nil {
		return "", errors.Join(recovery.ErrAnchorIndeterminate, writeErr, readErr)
	}
	if !sameDeletionAnchor(returned, expected) || !sameDeletionAnchor(observed, expected) || token != revision || revision == previous || revision == "" || strings.TrimSpace(revision) != revision {
		return "", recovery.ErrAnchorIndeterminate
	}
	if err := deletionAnchorContext(ctx); err != nil {
		return "", err
	}
	return revision, nil
}

func copyDeletionAnchor(a recovery.AnchorSnapshot) recovery.AnchorSnapshot {
	if a.Pending != nil {
		p := *a.Pending
		a.Pending = &p
	}
	return a
}

func (j *anchoredArticleJournal) commitAppend(ctx context.Context, a *deletionAnchorAppend) (recovery.VerifiedHeadAdvance, error) {
	next := a.pending
	next.Committed, next.Pending = a.pending.Pending.NewHead, nil
	revision, err := j.casAndLoad(ctx, a.pendingRevision, next)
	if err != nil || revision == a.before.Revision {
		return recovery.VerifiedHeadAdvance{}, errors.Join(recovery.ErrAnchorIndeterminate, err)
	}
	return recovery.VerifiedHeadAdvance{JournalID: next.JournalID, Namespace: next.Namespace, Epoch: next.Epoch,
		OldHead: a.before.Head, NewHead: next.Committed, OldRevision: a.before.Revision, PendingRevision: a.pendingRevision,
		CommittedRevision: revision, OperationID: a.record.DeletionOperationID}, nil
}
