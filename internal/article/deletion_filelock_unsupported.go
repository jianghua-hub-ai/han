//go:build !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd

package article

import (
	"context"
	"io/fs"
	"os"
)

func lockDeletionAnchor(context.Context, string, bool) (deletionNamespaceLease, error) {
	return nil, ErrDeletionJournalUnsupportedPlatform
}
func openDeletionAnchorFile(context.Context, string, string, bool, fs.FileMode) (*os.File, error) {
	return nil, ErrDeletionJournalUnsupportedPlatform
}
func linkDeletionAnchor(context.Context, string, string, string) error {
	return ErrDeletionJournalUnsupportedPlatform
}
func unlinkDeletionAnchorAlias(context.Context, string, string, string, []byte) error {
	return ErrDeletionJournalUnsupportedPlatform
}

func lockArticleDeletionFile(
	string,
	fs.FileMode,
) (articleDeletionFileLock, error) {
	return nil, ErrDeletionJournalUnsupportedPlatform
}
