package article

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	deletionTombstoneStage = "tombstoned"
	deletionHMACDomain     = "han/article-deletion/v1/tombstone\x00"
)

var (
	// ErrDeletionJournalCorrupt marks an unreadable or unverifiable deletion journal.
	ErrDeletionJournalCorrupt = errors.New("article deletion journal corrupt")
	// ErrDeletionJournalBusy marks a concurrent journal writer.
	ErrDeletionJournalBusy = errors.New("article deletion journal busy")
	// ErrDeletionTombstoneNotFound marks an absent operation tombstone.
	ErrDeletionTombstoneNotFound = errors.New("article deletion tombstone not found")
	// ErrDeletionJournalUnsupportedPlatform rejects production durability claims
	// where the standard library cannot sync directory metadata.
	ErrDeletionJournalUnsupportedPlatform = errors.New(
		"article deletion journal unsupported platform",
	)
)

var errDeletionJournalSequenceConflict = errors.New(
	"article deletion journal sequence conflict",
)

var errDeletionJournalFilesystem = errors.New(
	"article deletion journal filesystem failure",
)

// VerifiedDeletionJournal is the verified-read view of the production journal
// that the replay source consumes; it is satisfied by the concrete journal.
type VerifiedDeletionJournal interface {
	ReadAllVerified(context.Context) ([]ArticleDeletionTombstone, error)
}

// DeletionTombstoneRequest contains only the facts permitted in the external journal.
type DeletionTombstoneRequest struct {
	ArticleID           string
	ArticleVersion      int64
	DeletionOperationID string
	DeletedAt           time.Time
}

// ArticleDeletionJournal appends and resolves verified deletion tombstones.
type ArticleDeletionJournal interface {
	Append(context.Context, DeletionTombstoneRequest) (ArticleDeletionTombstone, error)
	FindByOperation(context.Context, string) (ArticleDeletionTombstone, error)
}

// DeletionJournalKey is one retained HMAC key.
type DeletionJournalKey struct {
	ID       string
	Version  string
	Material []byte
}

// DeletionJournalKeyring identifies the active writer key and retained verifier keys.
type DeletionJournalKeyring struct {
	Active   DeletionJournalKey
	Retained []DeletionJournalKey
}

// ArticleDeletionJournalOptions configures one dedicated deletion-journal directory.
type ArticleDeletionJournalOptions struct {
	Directory string
	FileMode  fs.FileMode
}

// FileArticleDeletionJournal stores one atomically renamed record per sequence.
type FileArticleDeletionJournal struct {
	directory     string
	fileMode      fs.FileMode
	active        DeletionJournalKey
	keys          map[string]DeletionJournalKey
	writerLock    articleDeletionFileLock
	closed        bool
	mu            sync.Mutex
	syncFile      func(*os.File) error
	syncDirectory func(string) error
}

type articleDeletionFileLock interface {
	Close() error
}

// NewFileArticleDeletionJournal validates a dedicated durable journal directory.
func NewFileArticleDeletionJournal(
	options ArticleDeletionJournalOptions,
	keyring DeletionJournalKeyring,
) (*FileArticleDeletionJournal, error) {
	if !articleDeletionJournalPlatformSupported() {
		return nil, ErrDeletionJournalUnsupportedPlatform
	}
	return newFileArticleDeletionJournal(
		options,
		keyring,
		syncDeletionJournalDirectory,
	)
}

func newFileArticleDeletionJournal(
	options ArticleDeletionJournalOptions,
	keyring DeletionJournalKeyring,
	syncDirectory func(string) error,
) (*FileArticleDeletionJournal, error) {
	if options.Directory == "" || strings.TrimSpace(options.Directory) != options.Directory ||
		options.FileMode.Perm() == 0 || options.FileMode.Perm()&0o077 != 0 {
		return nil, fmt.Errorf("%w: invalid journal options", ErrDeletionJournalCorrupt)
	}
	if syncDirectory == nil {
		return nil, fmt.Errorf("%w: directory sync is required", ErrDeletionJournalCorrupt)
	}
	if articleDeletionJournalPlatformSupported() {
		if err := validateArticleDeletionPathComponents(
			options.Directory,
			true,
			os.Lstat,
		); err != nil {
			return nil, err
		}
	} else {
		info, err := os.Lstat(options.Directory)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf(
				"%w: journal directory unavailable",
				ErrDeletionJournalCorrupt,
			)
		}
	}
	keys, err := deletionJournalKeys(keyring)
	if err != nil {
		return nil, err
	}
	journal := &FileArticleDeletionJournal{
		directory: options.Directory, fileMode: options.FileMode,
		active: cloneDeletionJournalKey(keyring.Active), keys: keys,
		syncFile:      func(file *os.File) error { return file.Sync() },
		syncDirectory: syncDirectory,
	}
	if _, err := journal.readVerified(); err != nil {
		_ = journal.Close()
		return nil, err
	}
	return journal, nil
}

// NewProductionFileArticleDeletionJournal owns the dedicated directory and writer lock.
func NewProductionFileArticleDeletionJournal(
	root string,
	fileMode fs.FileMode,
	keyring DeletionJournalKeyring,
) (*FileArticleDeletionJournal, error) {
	if err := ValidateArticleDeletionJournalRoot(root); err != nil {
		return nil, err
	}
	namespaceDirectory, err := ensureArticleDeletionDirectory(root, "article-deletion")
	if err != nil {
		return nil, err
	}
	versionDirectory, err := ensureArticleDeletionDirectory(namespaceDirectory, "v1")
	if err != nil {
		return nil, err
	}
	recordsDirectory, err := ensureArticleDeletionDirectory(versionDirectory, "records")
	if err != nil {
		return nil, err
	}
	writerLock, err := lockArticleDeletionFile(
		filepath.Join(versionDirectory, ".writer.lock"),
		fileMode,
	)
	if err != nil {
		return nil, err
	}
	journal, err := NewFileArticleDeletionJournal(
		ArticleDeletionJournalOptions{
			Directory: recordsDirectory,
			FileMode:  fileMode,
		},
		keyring,
	)
	if err != nil {
		return nil, errors.Join(err, sanitizeDeletionJournalFilesystemError(
			"close writer lock",
			writerLock.Close(),
		))
	}
	journal.writerLock = writerLock
	return journal, nil
}

// ValidateArticleDeletionJournalRoot performs the no-side-effect production preflight.
func ValidateArticleDeletionJournalRoot(root string) error {
	if !articleDeletionJournalPlatformSupported() {
		return ErrDeletionJournalUnsupportedPlatform
	}
	if root == "" || root != filepath.Clean(root) ||
		(!filepath.IsAbs(root) &&
			!strings.HasPrefix(root, string(filepath.Separator))) {
		return fmt.Errorf("%w: invalid journal root", ErrDeletionJournalCorrupt)
	}
	return validateArticleDeletionPathComponents(root, true, os.Lstat)
}

func ensureArticleDeletionDirectory(parent, name string) (string, error) {
	if err := validateArticleDeletionPathComponents(parent, true, os.Lstat); err != nil {
		return "", err
	}
	directory, err := articleDeletionChildPath(parent, name)
	if err != nil {
		return "", err
	}
	created := false
	if err := os.Mkdir(directory, 0o700); err == nil {
		created = true
	} else if !errors.Is(err, fs.ErrExist) {
		return "", sanitizeDeletionJournalFilesystemError("create directory", err)
	}
	if err := validateArticleDeletionPathComponents(directory, true, os.Lstat); err != nil {
		return "", err
	}
	if created {
		if err := syncDeletionJournalDirectory(parent); err != nil {
			return "", sanitizeDeletionJournalFilesystemError("sync parent directory", err)
		}
	}
	return directory, nil
}

func articleDeletionChildPath(parent, name string) (string, error) {
	if name == "" || name == "." || name == ".." ||
		filepath.Base(name) != name ||
		strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("%w: invalid journal directory name", ErrDeletionJournalCorrupt)
	}
	directory := filepath.Join(parent, name)
	relative, err := filepath.Rel(parent, directory)
	if err != nil || relative != name {
		return "", fmt.Errorf("%w: invalid journal directory", ErrDeletionJournalCorrupt)
	}
	return directory, nil
}

func validateArticleDeletionPathComponents(
	path string,
	requireOwnerOnly bool,
	lstat func(string) (fs.FileInfo, error),
) error {
	if lstat == nil || path == "" || filepath.Clean(path) != path {
		return fmt.Errorf("%w: journal path unavailable", ErrDeletionJournalCorrupt)
	}
	volume := filepath.VolumeName(path)
	remainder := strings.TrimPrefix(path, volume)
	rooted := strings.HasPrefix(remainder, string(filepath.Separator))
	current := volume
	if rooted {
		current += string(filepath.Separator)
		remainder = strings.TrimLeftFunc(
			remainder,
			func(value rune) bool { return os.IsPathSeparator(uint8(value)) },
		)
	}
	components := strings.FieldsFunc(
		remainder,
		func(value rune) bool { return os.IsPathSeparator(uint8(value)) },
	)
	paths := make([]string, 0, len(components)+1)
	if current != "" {
		paths = append(paths, current)
	}
	for _, component := range components {
		if current == "" {
			current = component
		} else {
			current = filepath.Join(current, component)
		}
		paths = append(paths, current)
	}
	if len(paths) == 0 {
		return fmt.Errorf("%w: journal path unavailable", ErrDeletionJournalCorrupt)
	}
	for index, componentPath := range paths {
		info, err := lstat(componentPath)
		if err != nil || info == nil ||
			info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("%w: journal path unavailable", ErrDeletionJournalCorrupt)
		}
		if requireOwnerOnly && index == len(paths)-1 &&
			info.Mode().Perm()&0o077 != 0 {
			return fmt.Errorf("%w: journal path permissions", ErrDeletionJournalCorrupt)
		}
	}
	return nil
}

// Close releases the writer lock and clears all retained key material.
func (journal *FileArticleDeletionJournal) Close() error {
	if journal == nil {
		return nil
	}
	journal.mu.Lock()
	defer journal.mu.Unlock()
	if journal.closed {
		return nil
	}
	journal.closed = true
	clear(journal.active.Material)
	journal.active = DeletionJournalKey{}
	for reference, key := range journal.keys {
		clear(key.Material)
		delete(journal.keys, reference)
	}
	var err error
	if journal.writerLock != nil {
		err = sanitizeDeletionJournalFilesystemError(
			"close writer lock",
			journal.writerLock.Close(),
		)
		journal.writerLock = nil
	}
	return err
}

// Append durably appends one idempotent operation tombstone.
func (journal *FileArticleDeletionJournal) Append(
	ctx context.Context,
	request DeletionTombstoneRequest,
) (ArticleDeletionTombstone, error) {
	if journal == nil {
		return ArticleDeletionTombstone{}, errors.New("article deletion journal is required")
	}
	if err := validateTombstoneRequest(request); err != nil {
		return ArticleDeletionTombstone{}, err
	}
	journal.mu.Lock()
	defer journal.mu.Unlock()
	if journal.closed {
		return ArticleDeletionTombstone{}, errors.New("article deletion journal is closed")
	}
	for {
		if err := ctx.Err(); err != nil {
			return ArticleDeletionTombstone{}, err
		}
		records, err := journal.readVerified()
		if err != nil {
			return ArticleDeletionTombstone{}, err
		}
		for _, record := range records {
			if record.DeletionOperationID != request.DeletionOperationID {
				continue
			}
			if record.ArticleID != request.ArticleID ||
				record.ArticleVersion != request.ArticleVersion ||
				!record.DeletedAt.Equal(request.DeletedAt.UTC()) {
				return ArticleDeletionTombstone{}, ErrDeletionJournalCorrupt
			}
			if err := journal.syncDirectory(journal.directory); err != nil {
				return ArticleDeletionTombstone{}, sanitizeDeletionJournalFilesystemError(
					"sync existing record",
					err,
				)
			}
			return record, nil
		}
		record := journal.newTombstone(request, records)
		if err := journal.persist(record); err != nil {
			if err == errDeletionJournalSequenceConflict {
				continue
			}
			return ArticleDeletionTombstone{}, err
		}
		return record, nil
	}
}

// ReadAllVerified returns every chain-verified tombstone in sequence order.
//
// It is the source-side read used to derive the verified Journal head; it
// performs the same durable directory sync as FindByOperation before returning.
func (journal *FileArticleDeletionJournal) ReadAllVerified(
	ctx context.Context,
) ([]ArticleDeletionTombstone, error) {
	if journal == nil {
		return nil, errors.New("article deletion journal is required")
	}
	journal.mu.Lock()
	defer journal.mu.Unlock()
	if journal.closed {
		return nil, errors.New("article deletion journal is closed")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	records, err := journal.readVerified()
	if err != nil {
		return nil, err
	}
	if err := journal.syncDirectory(journal.directory); err != nil {
		return nil, sanitizeDeletionJournalFilesystemError("sync verified read", err)
	}
	return records, nil
}

// FindByOperation returns one verified tombstone.
func (journal *FileArticleDeletionJournal) FindByOperation(
	ctx context.Context,
	operationID string,
) (ArticleDeletionTombstone, error) {
	if journal == nil || operationID == "" {
		return ArticleDeletionTombstone{}, ErrDeletionTombstoneNotFound
	}
	journal.mu.Lock()
	defer journal.mu.Unlock()
	if journal.closed {
		return ArticleDeletionTombstone{}, errors.New("article deletion journal is closed")
	}
	if err := ctx.Err(); err != nil {
		return ArticleDeletionTombstone{}, err
	}
	records, err := journal.readVerified()
	if err != nil {
		return ArticleDeletionTombstone{}, err
	}
	// Sync after verification so every record in this read snapshot has a
	// durable directory entry before it can authorize physical deletion.
	if err := journal.syncDirectory(journal.directory); err != nil {
		return ArticleDeletionTombstone{}, sanitizeDeletionJournalFilesystemError(
			"sync verified read",
			err,
		)
	}
	var found *ArticleDeletionTombstone
	for index := range records {
		if records[index].DeletionOperationID != operationID {
			continue
		}
		if found != nil {
			return ArticleDeletionTombstone{}, ErrDeletionJournalCorrupt
		}
		copy := records[index]
		found = &copy
	}
	if found == nil {
		return ArticleDeletionTombstone{}, ErrDeletionTombstoneNotFound
	}
	return *found, nil
}

func deletionJournalKeys(keyring DeletionJournalKeyring) (map[string]DeletionJournalKey, error) {
	if invalidDeletionJournalKey(keyring.Active) {
		return nil, fmt.Errorf("%w: invalid active key", ErrDeletionJournalCorrupt)
	}
	keys := make(map[string]DeletionJournalKey, len(keyring.Retained)+1)
	for _, key := range append(keyring.Retained, keyring.Active) {
		if invalidDeletionJournalKey(key) {
			clearDeletionJournalKeys(keys)
			return nil, fmt.Errorf("%w: invalid retained key", ErrDeletionJournalCorrupt)
		}
		reference := key.ID + "\x00" + key.Version
		if existing, ok := keys[reference]; ok {
			if !hmac.Equal(existing.Material, key.Material) {
				clearDeletionJournalKeys(keys)
				return nil, fmt.Errorf(
					"%w: conflicting key reference",
					ErrDeletionJournalCorrupt,
				)
			}
			clear(existing.Material)
		}
		keys[reference] = cloneDeletionJournalKey(key)
	}
	return keys, nil
}

func clearDeletionJournalKeys(keys map[string]DeletionJournalKey) {
	for reference, key := range keys {
		clear(key.Material)
		delete(keys, reference)
	}
}

func cloneDeletionJournalKey(key DeletionJournalKey) DeletionJournalKey {
	return DeletionJournalKey{
		ID:       key.ID,
		Version:  key.Version,
		Material: append([]byte(nil), key.Material...),
	}
}

func invalidDeletionJournalKey(key DeletionJournalKey) bool {
	return key.ID == "" || key.Version == "" || len(key.Material) == 0 ||
		strings.TrimSpace(key.ID) != key.ID ||
		strings.TrimSpace(key.Version) != key.Version
}

func validateTombstoneRequest(request DeletionTombstoneRequest) error {
	if !validUUID(request.ArticleID) || !validUUID(request.DeletionOperationID) ||
		request.ArticleVersion < 1 || request.DeletedAt.IsZero() {
		return fmt.Errorf("%w: invalid tombstone request", ErrDeletionJournalCorrupt)
	}
	return nil
}

func (journal *FileArticleDeletionJournal) newTombstone(
	request DeletionTombstoneRequest,
	records []ArticleDeletionTombstone,
) ArticleDeletionTombstone {
	record := ArticleDeletionTombstone{
		Namespace: ArticleDeletionNamespace, Sequence: int64(len(records) + 1),
		ArticleID: request.ArticleID, ArticleVersion: request.ArticleVersion,
		DeletionOperationID: request.DeletionOperationID,
		DeletedAt:           request.DeletedAt.UTC(), KeyID: journal.active.ID,
		KeyVersion: journal.active.Version, Stage: deletionTombstoneStage,
	}
	if len(records) > 0 {
		record.PreviousRecordHash = deletionRecordHash(records[len(records)-1])
	}
	record.RecordHMAC = deletionRecordHMAC(record, journal.active.Material)
	return record
}

func (journal *FileArticleDeletionJournal) persist(
	record ArticleDeletionTombstone,
) (resultErr error) {
	raw, err := json.Marshal(deletionDiskRecordFromModel(record))
	if err != nil {
		return fmt.Errorf("%w: encode record", ErrDeletionJournalCorrupt)
	}
	temp, err := os.CreateTemp(journal.directory, ".article-deletion.tmp-*")
	if err != nil {
		return sanitizeDeletionJournalFilesystemError("create record", err)
	}
	tempPath := temp.Name()
	defer func() {
		if tempPath == "" {
			return
		}
		if err := os.Remove(tempPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			resultErr = errors.Join(
				resultErr,
				sanitizeDeletionJournalFilesystemError("remove temporary record", err),
			)
		}
	}()
	if err := temp.Chmod(journal.fileMode); err != nil {
		_ = temp.Close()
		return sanitizeDeletionJournalFilesystemError("set record permissions", err)
	}
	if _, err := temp.Write(raw); err != nil {
		_ = temp.Close()
		return sanitizeDeletionJournalFilesystemError("write record", err)
	}
	if err := journal.syncFile(temp); err != nil {
		_ = temp.Close()
		return sanitizeDeletionJournalFilesystemError("sync record", err)
	}
	if err := temp.Close(); err != nil {
		return sanitizeDeletionJournalFilesystemError("close record", err)
	}
	target := filepath.Join(journal.directory, deletionRecordName(record.Sequence))
	if err := os.Link(tempPath, target); errors.Is(err, fs.ErrExist) {
		if removeErr := os.Remove(tempPath); removeErr != nil {
			return errors.Join(
				errDeletionJournalSequenceConflict,
				sanitizeDeletionJournalFilesystemError(
					"remove conflicted temporary record",
					removeErr,
				),
			)
		}
		tempPath = ""
		return errDeletionJournalSequenceConflict
	} else if err != nil {
		return sanitizeDeletionJournalFilesystemError("publish record", err)
	}
	if err := os.Remove(tempPath); err != nil {
		return sanitizeDeletionJournalFilesystemError("remove linked temporary record", err)
	}
	tempPath = ""
	if err := journal.syncDirectory(journal.directory); err != nil {
		return sanitizeDeletionJournalFilesystemError("sync record directory", err)
	}
	return nil
}

func (journal *FileArticleDeletionJournal) readVerified() ([]ArticleDeletionTombstone, error) {
	entries, err := os.ReadDir(journal.directory)
	if err != nil {
		return nil, sanitizeDeletionJournalFilesystemError("read record directory", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, sanitizeDeletionJournalFilesystemError("inspect record", err)
		}
		if !info.Mode().IsRegular() ||
			info.Mode()&os.ModeSymlink != 0 ||
			(articleDeletionJournalPlatformSupported() &&
				info.Mode().Perm()&0o077 != 0) {
			return nil, fmt.Errorf("%w: invalid record file", ErrDeletionJournalCorrupt)
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	records := make([]ArticleDeletionTombstone, 0, len(names))
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(journal.directory, name))
		if err != nil {
			return nil, sanitizeDeletionJournalFilesystemError("read record", err)
		}
		var disk deletionDiskRecord
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&disk); err != nil {
			return nil, fmt.Errorf("%w: decode record", ErrDeletionJournalCorrupt)
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("%w: trailing record data", ErrDeletionJournalCorrupt)
		}
		record, err := disk.toModel()
		if err != nil || record.Sequence != int64(len(records)+1) ||
			name != deletionRecordName(record.Sequence) {
			return nil, ErrDeletionJournalCorrupt
		}
		if err := journal.verifyRecord(record, records); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func (journal *FileArticleDeletionJournal) verifyRecord(
	record ArticleDeletionTombstone,
	previous []ArticleDeletionTombstone,
) error {
	if record.Namespace != ArticleDeletionNamespace ||
		record.Stage != deletionTombstoneStage ||
		validateTombstoneRequest(DeletionTombstoneRequest{
			ArticleID: record.ArticleID, ArticleVersion: record.ArticleVersion,
			DeletionOperationID: record.DeletionOperationID, DeletedAt: record.DeletedAt,
		}) != nil {
		return ErrDeletionJournalCorrupt
	}
	expectedPrevious := ""
	if len(previous) > 0 {
		expectedPrevious = deletionRecordHash(previous[len(previous)-1])
	}
	if record.PreviousRecordHash != expectedPrevious {
		return ErrDeletionJournalCorrupt
	}
	key, ok := journal.keys[record.KeyID+"\x00"+record.KeyVersion]
	if !ok || !hmac.Equal(
		[]byte(record.RecordHMAC),
		[]byte(deletionRecordHMAC(record, key.Material)),
	) {
		return ErrDeletionJournalCorrupt
	}
	return nil
}

func deletionRecordName(sequence int64) string {
	return fmt.Sprintf("%020d.json", sequence)
}

func deletionRecordHash(record ArticleDeletionTombstone) string {
	raw, _ := json.Marshal(deletionDiskRecordFromModel(record))
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func deletionRecordHMAC(record ArticleDeletionTombstone, key []byte) string {
	raw, _ := json.Marshal(deletionMACPayloadFromModel(record))
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(deletionHMACDomain))
	_, _ = mac.Write(raw)
	return "hmac-sha256:" + hex.EncodeToString(mac.Sum(nil))
}

func syncDeletionJournalDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	syncErr := directory.Sync()
	closeErr := directory.Close()
	return errors.Join(syncErr, closeErr)
}

func sanitizeDeletionJournalFilesystemError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s", errDeletionJournalFilesystem, operation)
}

func articleDeletionJournalPlatformSupported() bool {
	switch runtime.GOOS {
	case "darwin", "dragonfly", "freebsd", "linux", "netbsd", "openbsd":
		return true
	default:
		return false
	}
}

type deletionDiskRecord struct {
	Namespace           string `json:"namespace"`
	Sequence            int64  `json:"sequence"`
	ArticleID           string `json:"article_id"`
	ArticleVersion      int64  `json:"article_version"`
	DeletionOperationID string `json:"deletion_operation_id"`
	DeletedAt           string `json:"deleted_at"`
	KeyID               string `json:"key_id"`
	KeyVersion          string `json:"key_version"`
	Stage               string `json:"stage"`
	PreviousRecordHash  string `json:"previous_record_hash"`
	RecordHMAC          string `json:"record_hmac"`
}

type deletionMACPayload struct {
	Namespace           string `json:"namespace"`
	Sequence            int64  `json:"sequence"`
	ArticleID           string `json:"article_id"`
	ArticleVersion      int64  `json:"article_version"`
	DeletionOperationID string `json:"deletion_operation_id"`
	DeletedAt           string `json:"deleted_at"`
	KeyID               string `json:"key_id"`
	KeyVersion          string `json:"key_version"`
	Stage               string `json:"stage"`
	PreviousRecordHash  string `json:"previous_record_hash"`
}

func deletionDiskRecordFromModel(record ArticleDeletionTombstone) deletionDiskRecord {
	payload := deletionMACPayloadFromModel(record)
	return deletionDiskRecord{
		Namespace: payload.Namespace, Sequence: payload.Sequence,
		ArticleID: payload.ArticleID, ArticleVersion: payload.ArticleVersion,
		DeletionOperationID: payload.DeletionOperationID, DeletedAt: payload.DeletedAt,
		KeyID: payload.KeyID, KeyVersion: payload.KeyVersion, Stage: payload.Stage,
		PreviousRecordHash: payload.PreviousRecordHash, RecordHMAC: record.RecordHMAC,
	}
}

func deletionMACPayloadFromModel(record ArticleDeletionTombstone) deletionMACPayload {
	return deletionMACPayload{
		Namespace: record.Namespace, Sequence: record.Sequence,
		ArticleID: record.ArticleID, ArticleVersion: record.ArticleVersion,
		DeletionOperationID: record.DeletionOperationID,
		DeletedAt:           record.DeletedAt.UTC().Format(time.RFC3339Nano),
		KeyID:               record.KeyID, KeyVersion: record.KeyVersion, Stage: record.Stage,
		PreviousRecordHash: record.PreviousRecordHash,
	}
}

func (record deletionDiskRecord) toModel() (ArticleDeletionTombstone, error) {
	deletedAt, err := time.Parse(time.RFC3339Nano, record.DeletedAt)
	if err != nil || deletedAt.Location() != time.UTC ||
		deletedAt.Format(time.RFC3339Nano) != record.DeletedAt {
		return ArticleDeletionTombstone{}, ErrDeletionJournalCorrupt
	}
	return ArticleDeletionTombstone{
		Namespace: record.Namespace, Sequence: record.Sequence,
		ArticleID: record.ArticleID, ArticleVersion: record.ArticleVersion,
		DeletionOperationID: record.DeletionOperationID, DeletedAt: deletedAt,
		KeyID: record.KeyID, KeyVersion: record.KeyVersion, Stage: record.Stage,
		PreviousRecordHash: record.PreviousRecordHash, RecordHMAC: record.RecordHMAC,
	}, nil
}
