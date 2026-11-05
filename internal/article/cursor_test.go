package article

import (
	"encoding/base64"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestCursor_RoundTrip_IsStableAndViewBound(t *testing.T) {
	codec := CursorCodec{}
	createdAt := time.Date(2026, time.October, 22, 1, 2, 3, 456000000, time.UTC)
	views := []CursorView{
		CursorViewMember,
		CursorViewAdminAll,
		CursorViewAdminViolating,
		CursorViewAdminOrphan,
	}

	for _, view := range views {
		t.Run(string(view), func(t *testing.T) {
			input := ArticleCursor{
				CreatedAt: createdAt,
				ID:        "10000000-0000-4000-8000-000000000001",
				View:      view,
			}
			first, err := codec.Encode(input)
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}
			second, err := codec.Encode(input)
			if err != nil {
				t.Fatalf("Encode() second error = %v", err)
			}
			if first != second {
				t.Fatalf("Encode() unstable: %q != %q", first, second)
			}

			decoded, err := codec.Decode(first, view)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if decoded.ID != input.ID || decoded.View != input.View ||
				!decoded.CreatedAt.Equal(input.CreatedAt) {
				t.Fatalf("Decode() = %#v, want %#v", decoded, input)
			}

			wrongView := CursorViewAdminAll
			if view == wrongView {
				wrongView = CursorViewMember
			}
			if _, err := codec.Decode(first, wrongView); !errors.Is(err, ErrInvalidCursor) {
				t.Fatalf("Decode() wrong view error = %v, want ErrInvalidCursor", err)
			}
		})
	}
}

func TestCursor_InvalidPayloads_FailClosed(t *testing.T) {
	validTime := "2026-10-22T01:02:03.456Z"
	validID := "10000000-0000-4000-8000-000000000001"
	tests := []struct {
		name   string
		cursor string
	}{
		{name: "empty"},
		{name: "not base64url", cursor: "%%%"},
		{name: "padded base64url", cursor: base64.RawURLEncoding.EncodeToString([]byte(
			fmt.Sprintf(`{"v":1,"created_at":%q,"id":%q,"view":"member"}`, validTime, validID),
		)) + "="},
		{name: "missing field", cursor: encodeCursorJSON(t,
			fmt.Sprintf(`{"v":1,"created_at":%q,"view":"member"}`, validTime))},
		{name: "unknown version", cursor: encodeCursorJSON(t,
			fmt.Sprintf(`{"v":2,"created_at":%q,"id":%q,"view":"member"}`, validTime, validID))},
		{name: "wrong view value", cursor: encodeCursorJSON(t,
			fmt.Sprintf(`{"v":1,"created_at":%q,"id":%q,"view":"unknown"}`, validTime, validID))},
		{name: "invalid time", cursor: encodeCursorJSON(t,
			fmt.Sprintf(`{"v":1,"created_at":"yesterday","id":%q,"view":"member"}`, validID))},
		{name: "non utc time", cursor: encodeCursorJSON(t,
			fmt.Sprintf(`{"v":1,"created_at":"2026-10-22T09:02:03+08:00","id":%q,"view":"member"}`, validID))},
		{name: "invalid uuid", cursor: encodeCursorJSON(t,
			fmt.Sprintf(`{"v":1,"created_at":%q,"id":"not-a-uuid","view":"member"}`, validTime))},
		{name: "duplicate field", cursor: encodeCursorJSON(t,
			fmt.Sprintf(`{"v":1,"created_at":%q,"id":%q,"id":%q,"view":"member"}`, validTime, validID, validID))},
		{name: "unknown field", cursor: encodeCursorJSON(t,
			fmt.Sprintf(`{"v":1,"created_at":%q,"id":%q,"view":"member","extra":true}`, validTime, validID))},
		{name: "trailing json", cursor: encodeCursorJSON(t,
			fmt.Sprintf(`{"v":1,"created_at":%q,"id":%q,"view":"member"} {}`, validTime, validID))},
		{name: "array payload", cursor: encodeCursorJSON(t, `[]`)},
	}

	codec := CursorCodec{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := codec.Decode(test.cursor, CursorViewMember); !errors.Is(err, ErrInvalidCursor) {
				t.Fatalf("Decode() error = %v, want ErrInvalidCursor", err)
			}
		})
	}
}

func TestCursor_InvalidModel_FailsClosed(t *testing.T) {
	codec := CursorCodec{}
	valid := ArticleCursor{
		CreatedAt: time.Date(2026, time.October, 22, 1, 2, 3, 0, time.UTC),
		ID:        "10000000-0000-4000-8000-000000000001",
		View:      CursorViewMember,
	}
	tests := []struct {
		name   string
		mutate func(*ArticleCursor)
	}{
		{name: "zero time", mutate: func(cursor *ArticleCursor) { cursor.CreatedAt = time.Time{} }},
		{name: "non utc", mutate: func(cursor *ArticleCursor) {
			cursor.CreatedAt = cursor.CreatedAt.In(time.FixedZone("offset", 8*60*60))
		}},
		{name: "invalid id", mutate: func(cursor *ArticleCursor) { cursor.ID = "not-a-uuid" }},
		{name: "invalid view", mutate: func(cursor *ArticleCursor) { cursor.View = CursorView("unknown") }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cursor := valid
			test.mutate(&cursor)
			if _, err := codec.Encode(cursor); !errors.Is(err, ErrInvalidCursor) {
				t.Fatalf("Encode() error = %v, want ErrInvalidCursor", err)
			}
		})
	}
}

func encodeCursorJSON(t *testing.T, payload string) string {
	t.Helper()
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}
