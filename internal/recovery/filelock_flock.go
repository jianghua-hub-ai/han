//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package recovery

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"
)

type journalFileLock struct {
	check  func(context.Context) error
	unlock func() error
	close  func() error
}

func fileJournalSupported() bool { return true }

func acquireJournalFileLock(ctx context.Context, path string, mode fs.FileMode) (*journalFileLock, error) {
	return acquireJournalLock(ctx, path, mode, true)
}
func acquireJournalReadLock(ctx context.Context, path string, mode fs.FileMode) (*journalFileLock, error) {
	return acquireJournalLock(ctx, path, mode, false)
}
func acquireJournalLock(ctx context.Context, path string, mode fs.FileMode, create bool) (*journalFileLock, error) {
	if err := checkedContext(ctx); err != nil {
		return nil, err
	}
	lockPath := path + ".lock"
	flags := unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK | unix.O_RDONLY
	if create {
		flags = unix.O_CLOEXEC | unix.O_CREAT | unix.O_NOFOLLOW | unix.O_NONBLOCK | unix.O_RDWR
	}
	fd, err := unix.Open(lockPath, flags, uint32(mode.Perm()))
	if err != nil {
		return nil, fmt.Errorf("%w: open lock: %w", ErrJournalBusy, err)
	}
	file := os.NewFile(uintptr(fd), lockPath)
	if file == nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("%w: create lock handle", ErrJournalBusy)
	}
	if err := validateOpenedJournalLock(lockPath, file); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	if err := waitJournalLock(ctx, file, create); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	if err := validateOpenedJournalLock(lockPath, file); err != nil {
		unlockErr := unix.Flock(int(file.Fd()), unix.LOCK_UN)
		return nil, errors.Join(err, unlockErr, file.Close())
	}
	return &journalFileLock{
		check: func(ctx context.Context) error {
			if e := checkedContext(ctx); e != nil {
				return e
			}
			return validateOpenedJournalLock(lockPath, file)
		},
		unlock: func() error {
			return unix.Flock(int(file.Fd()), unix.LOCK_UN)
		},
		close: file.Close,
	}, nil
}

func validateOpenedJournalLock(path string, file *os.File) error {
	if file == nil {
		return fmt.Errorf("%w: nil lock handle", ErrJournalBusy)
	}
	var opened, current unix.Stat_t
	if err := unix.Fstat(int(file.Fd()), &opened); err != nil {
		return fmt.Errorf("%w: inspect lock handle: %w", ErrJournalBusy, err)
	}
	if err := unix.Lstat(path, &current); err != nil {
		return fmt.Errorf("%w: inspect lock path: %w", ErrJournalBusy, err)
	}
	if opened.Mode&unix.S_IFMT != unix.S_IFREG ||
		current.Mode&unix.S_IFMT != unix.S_IFREG ||
		opened.Uid != uint32(unix.Geteuid()) ||
		current.Uid != uint32(unix.Geteuid()) ||
		opened.Mode&0o077 != 0 ||
		current.Mode&0o077 != 0 ||
		opened.Dev != current.Dev ||
		opened.Ino != current.Ino {
		return fmt.Errorf(
			"%w: lock must be an owner-only regular file with a stable inode",
			ErrJournalBusy,
		)
	}
	return nil
}

func waitJournalLock(ctx context.Context, file *os.File, exclusive bool) error {
	mode := unix.LOCK_SH | unix.LOCK_NB
	if exclusive {
		mode = unix.LOCK_EX | unix.LOCK_NB
	}
	for {
		if e := checkedContext(ctx); e != nil {
			return e
		}
		e := unix.Flock(int(file.Fd()), mode)
		if e == nil {
			return nil
		}
		if !errors.Is(e, unix.EWOULDBLOCK) && !errors.Is(e, unix.EAGAIN) {
			return errors.Join(ErrJournalBusy, e)
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return errors.Join(ErrJournalBusy, ctx.Err())
		case <-timer.C:
		}
	}
}

func openJournalReadFile(ctx context.Context, path string) (*os.File, error) {
	if e := checkedContext(ctx); e != nil {
		return nil, e
	}
	parent, e := unix.Open(filepath.Dir(path), unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY|unix.O_RDONLY, 0)
	if e != nil {
		return nil, e
	}
	var st unix.Stat_t
	e = unix.Fstat(parent, &st)
	if e != nil || st.Uid != uint32(unix.Geteuid()) || st.Mode&0o077 != 0 {
		unix.Close(parent)
		return nil, errors.Join(ErrJournalCorrupt, e)
	}
	fd, e := unix.Openat(parent, filepath.Base(path), unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	closeErr := unix.Close(parent)
	if e != nil {
		return nil, errors.Join(e, closeErr)
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		unix.Close(fd)
		return nil, ErrJournalCorrupt
	}
	if e := validateOpenedJournalLock(path, file); e != nil {
		return nil, errors.Join(e, file.Close(), closeErr)
	}
	if closeErr != nil {
		return nil, errors.Join(closeErr, file.Close())
	}
	return file, nil
}
