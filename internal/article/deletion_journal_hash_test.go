package article

import (
	"testing"
	"time"
)

// TestDeletionJournalRecordHashPinsDiskProjection freezes the production
// on-disk record projection. The expected digest was computed independently
// from the documented deletionDiskRecord field order; a mismatch means the
// hash projection drifted and the synthetic S4.4 recovery harnesses in
// test/integration/recovery must be updated together with production.
func TestDeletionJournalRecordHashPinsDiskProjection(t *testing.T) {
	record := ArticleDeletionTombstone{
		Namespace:           ArticleDeletionNamespace,
		Sequence:            7,
		ArticleID:           "10000000-0000-4000-8000-000000000007",
		ArticleVersion:      3,
		DeletionOperationID: "20000000-0000-4000-8000-000000000007",
		DeletedAt:           time.Date(2026, 10, 31, 12, 34, 56, 123456789, time.UTC),
		KeyID:               "key-7",
		KeyVersion:          "1",
		Stage:               deletionTombstoneStage,
		PreviousRecordHash:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RecordHMAC:          "hmac-sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
	const want = "sha256:4ce06ce79ad8639d2468f0774d6d2d2097909bab6203a2dd5f63701d2f4bfacf"
	if got := deletionRecordHash(record); got != want {
		t.Fatalf("deletionRecordHash() = %q, want %q", got, want)
	}
}
