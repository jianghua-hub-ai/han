//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package article

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

type flockArticleDeletionFile struct {
	file *os.File
}

type anchoredDeletionFileLock struct {
	file     *os.File
	path     string
	attempt  sync.Once
	terminal chan struct{}
	result   error
}

func lockDeletionAnchor(ctx context.Context, path string, write bool) (lease deletionNamespaceLease, result error) {
	if err := deletionAnchorContext(ctx); err != nil {
		return nil, err
	}
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	lock := &anchoredDeletionFileLock{file: file, path: path, terminal: make(chan struct{})}
	defer joinDeletionNamespaceAcquisitionPanic(ctx, lock)
	if err := lock.Check(ctx); err != nil {
		return lock, err
	}
	if err := waitDeletionAnchorLock(ctx, file, write); err != nil {
		return lock, err
	}
	if err := lock.Check(ctx); err != nil {
		return lock, err
	}
	return lock, nil
}

func joinDeletionNamespaceAcquisitionPanic(ctx context.Context, lease deletionNamespaceLease) {
	if value := recover(); value != nil {
		owner := deletionNamespaceOwner{lease: lease}
		owner.attempt()
		owner.join(ctx)
		panic(value)
	}
}

func waitDeletionAnchorLock(ctx context.Context, file *os.File, write bool) error {
	mode := unix.LOCK_SH | unix.LOCK_NB
	if write {
		mode = unix.LOCK_EX | unix.LOCK_NB
	}
	for {
		if err := deletionAnchorContext(ctx); err != nil {
			return err
		}
		err := unix.Flock(int(file.Fd()), mode)
		if err == nil {
			return nil
		}
		if !errors.Is(err, unix.EWOULDBLOCK) && !errors.Is(err, unix.EAGAIN) {
			return err
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (l *anchoredDeletionFileLock) Check(ctx context.Context) error {
	if err := deletionAnchorContext(ctx); err != nil {
		return err
	}
	return checkDeletionAnchorFile(l.file, l.path)
}

func (l *anchoredDeletionFileLock) Close() error {
	l.attempt.Do(func() {
		unlockErr := unix.Flock(int(l.file.Fd()), unix.LOCK_UN)
		closeErr := l.file.Close()
		l.result = errors.Join(unlockErr, closeErr)
		if closeErr == nil {
			close(l.terminal)
		}
		// Ambiguous raw Close retains this original owner and pending terminal.
		// Never retry a numeric FD or infer release from File.Fd/Stat/pointers.
	})
	return l.result
}

func (l *anchoredDeletionFileLock) WaitReleased(ctx context.Context) error {
	<-l.terminal
	return errors.Join(l.result, deletionAnchorContext(ctx))
}

func checkDeletionAnchorFile(file *os.File, path string) error {
	if file == nil {
		return ErrDeletionJournalCorrupt
	}
	var opened, current unix.Stat_t
	if err := unix.Fstat(int(file.Fd()), &opened); err != nil {
		return err
	}
	if err := unix.Lstat(path, &current); err != nil {
		return err
	}
	if opened.Mode&unix.S_IFMT != unix.S_IFREG || current.Mode&unix.S_IFMT != unix.S_IFREG ||
		opened.Uid != uint32(unix.Geteuid()) || current.Uid != uint32(unix.Geteuid()) ||
		opened.Mode&0o077 != 0 || current.Mode&0o077 != 0 || opened.Dev != current.Dev || opened.Ino != current.Ino {
		return ErrDeletionJournalCorrupt
	}
	return nil
}

func deletionAnchorDirectory(ctx context.Context, directory string) (int, error) {
	if err := deletionAnchorContext(ctx); err != nil {
		return -1, err
	}
	if err := validateArticleDeletionPathComponents(directory, true, os.Lstat); err != nil {
		return -1, err
	}
	fd, err := unix.Open(directory, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return -1, err
	}
	var stat unix.Stat_t
	err = unix.Fstat(fd, &stat)
	if err != nil || stat.Uid != uint32(unix.Geteuid()) || stat.Mode&0o077 != 0 {
		return -1, errors.Join(ErrDeletionJournalCorrupt, err, unix.Close(fd))
	}
	return fd, nil
}

func openDeletionAnchorFile(ctx context.Context, directory, name string, create bool, mode fs.FileMode) (file *os.File, result error) {
	if filepath.Base(name) != name || name == "." || name == ".." {
		return nil, ErrDeletionJournalCorrupt
	}
	parent, err := deletionAnchorDirectory(ctx, directory)
	if err != nil {
		return nil, err
	}
	defer func() { result = errors.Join(result, unix.Close(parent)) }()
	flags := unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK
	if create {
		flags = unix.O_WRONLY | unix.O_CREAT | unix.O_EXCL | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK
	}
	fd, err := unix.Openat(parent, name, flags, uint32(mode.Perm()))
	if err != nil {
		return nil, err
	}
	file = os.NewFile(uintptr(fd), filepath.Join(directory, name))
	if err := checkDeletionAnchorFile(file, file.Name()); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	return file, nil
}

func linkDeletionAnchor(ctx context.Context, directory, from, to string) (result error) {
	parent, err := deletionAnchorDirectory(ctx, directory)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, unix.Close(parent)) }()
	err = unix.Linkat(parent, from, parent, to, 0)
	if !errors.Is(err, unix.EEXIST) {
		return err
	}
	return verifyDeletionAnchorAlias(ctx, directory, from, to, nil)
}

func verifyDeletionAnchorAlias(ctx context.Context, directory, from, to string, expected []byte) (result error) {
	a, err := openDeletionAnchorFile(ctx, directory, from, false, 0)
	if a != nil {
		defer func() { result = errors.Join(result, a.Close()) }()
	}
	if err != nil {
		return err
	}
	b, err := openDeletionAnchorFile(ctx, directory, to, false, 0)
	if b != nil {
		defer func() { result = errors.Join(result, b.Close()) }()
	}
	if err != nil {
		return err
	}
	x, err := a.Stat()
	if err != nil {
		return err
	}
	y, err := b.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(x, y) {
		return ErrDeletionJournalCorrupt
	}
	if expected != nil {
		raw, err := io.ReadAll(io.LimitReader(a, int64(len(expected))+1))
		if err != nil || !bytes.Equal(raw, expected) {
			return errors.Join(ErrDeletionJournalCorrupt, err)
		}
	}
	return deletionAnchorContext(ctx)
}

func unlinkDeletionAnchorAlias(ctx context.Context, directory, from, to string, expected []byte) (result error) {
	if err := verifyDeletionAnchorAlias(ctx, directory, from, to, expected); err != nil {
		return err
	}
	parent, err := deletionAnchorDirectory(ctx, directory)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, unix.Close(parent)) }()
	return unix.Unlinkat(parent, from, 0)
}

func lockArticleDeletionFile(
	path string,
	mode fs.FileMode,
) (articleDeletionFileLock, error) {
	descriptor, err := unix.Open(
		path,
		unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW,
		uint32(mode.Perm()),
	)
	if err != nil {
		return nil, sanitizeDeletionJournalFilesystemError("open writer lock", err)
	}
	file := os.NewFile(uintptr(descriptor), path)
	info, err := file.Stat()
	if err != nil {
		return nil, errors.Join(
			sanitizeDeletionJournalFilesystemError("inspect writer lock", err),
			sanitizeDeletionJournalFilesystemError("close writer lock", file.Close()),
		)
	}
	if !info.Mode().IsRegular() {
		return nil, errors.Join(
			ErrDeletionJournalCorrupt,
			sanitizeDeletionJournalFilesystemError("close writer lock", file.Close()),
		)
	}
	if err := file.Chmod(mode); err != nil {
		return nil, errors.Join(
			sanitizeDeletionJournalFilesystemError("set writer lock permissions", err),
			sanitizeDeletionJournalFilesystemError("close writer lock", file.Close()),
		)
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
			return nil, errors.Join(
				ErrDeletionJournalBusy,
				sanitizeDeletionJournalFilesystemError("close writer lock", file.Close()),
			)
		}
		return nil, errors.Join(
			sanitizeDeletionJournalFilesystemError("lock writer", err),
			sanitizeDeletionJournalFilesystemError("close writer lock", file.Close()),
		)
	}
	return &flockArticleDeletionFile{file: file}, nil
}

func (lock *flockArticleDeletionFile) Close() error {
	if lock == nil || lock.file == nil {
		return nil
	}
	unlockErr := unix.Flock(int(lock.file.Fd()), unix.LOCK_UN)
	closeErr := lock.file.Close()
	lock.file = nil
	return errors.Join(
		sanitizeDeletionJournalFilesystemError("unlock writer", unlockErr),
		sanitizeDeletionJournalFilesystemError("close writer lock", closeErr),
	)
}
