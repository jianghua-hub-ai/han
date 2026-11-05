//go:build !(aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris)

package recovery

import (
	"context"
	"io/fs"
	"os"
)

type journalFileLock struct {
	check  func(context.Context) error
	unlock func() error
	close  func() error
}

func fileJournalSupported() bool { return false }

func acquireJournalFileLock(context.Context, string, fs.FileMode) (*journalFileLock, error) {
	return nil, ErrUnsupportedPlatform
}

func acquireJournalReadLock(context.Context, string, fs.FileMode) (*journalFileLock, error) {
	return nil, ErrUnsupportedPlatform
}
func openJournalReadFile(context.Context, string) (*os.File, error) {
	return nil, ErrUnsupportedPlatform
}
