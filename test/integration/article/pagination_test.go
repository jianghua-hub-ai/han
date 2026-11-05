//go:build integration

package articleintegration

import (
	"errors"
	"testing"
	"time"

	"han/internal/article"
)

// TestArticlePagination exercises the real keyset/identity query path.
func TestArticlePagination(t *testing.T) {
	t.Run("HiddenAcrossBatches", s22HiddenBatches)
	t.Run("SameTimestampNoMissingDuplicates", s22SameTimestamp)
	t.Run("LookaheadCursorBoundary", s22Lookahead)
	t.Run("EmptyAndLastPage", s22EmptyPage)
	t.Run("InsertEditAndAuthorStateBetweenPages", s22BetweenPages)
}

func s22HiddenBatches(t *testing.T) {
	s := s22New(t)
	s22Dense(s)
	db := &s22QueryDB{real: s.h.pool}
	q := s22QueryWith(s, db, s.reader)
	expected := s22Oracle(s, nil)
	actual := s22Traverse(s, q, "")
	s22AssertSame(t, actual, expected, "hidden batches skipped/duplicated SQL-visible rows")
	if db.calls.Load() < 5 {
		t.Fatal("fixture did not exercise multiple hidden SQL batches")
	}
	t.Run("FailureNeverReturnsHalfPage", s22NoHalfPage)
}
func s22NoHalfPage(t *testing.T) {
	for _, failure := range []string{"candidate", "identity"} {
		t.Run(failure, func(t *testing.T) {
			s := s22New(t)
			for i := 1; i <= 6; i++ {
				r := s.base(i)
				if i != 6 && i != 3 {
					category := article.ViolationSexual
					r.ViolationState = article.ViolationViolating
					r.ViolationCategory = &category
				}
				s.insert(r)
			}
			db := &s22QueryDB{real: s.h.pool}
			if failure == "candidate" {
				db.failAt = 2
			}
			reader := &s22ReaderFault{real: s.reader, mode: "fault", failAt: 2}
			if failure == "candidate" {
				reader.failAt = 100
			}
			q := s22QueryWith(s, db, reader)
			page, err := q.ListArticles(s.ctx, article.ListArticlesRequest{})
			if err == nil || len(page.Items) != 0 || page.NextCursor != nil {
				t.Fatal("batch dependency failure returned accumulated half page")
			}
		})
	}
}
func s22SameTimestamp(t *testing.T) {
	s := s22New(t)
	for i := 1; i <= 9; i++ {
		r := s.base(i)
		if i%3 == 0 {
			r.AuthorID = &s.suspended.UserID
		}
		s.insert(r)
	}
	expected := s22Oracle(s, nil)
	s22AssertSame(t, s22Traverse(s, s.query(), ""), expected, "same-timestamp UUID order leaked/missed/duplicated rows")
}
func s22Lookahead(t *testing.T) {
	s := s22New(t)
	for i := 1; i <= 5; i++ {
		s.insert(s.base(i))
	}
	q := s.query()
	first, err := q.ListArticles(s.ctx, article.ListArticlesRequest{})
	noticeMust(t, err)
	if first.NextCursor == nil || len(first.Items) != 2 {
		t.Fatal("first page did not produce lookahead cursor")
	}
	cursor, err := (article.CursorCodec{}).Decode(*first.NextCursor, article.CursorViewMember)
	noticeMust(t, err)
	if cursor.ID != first.Items[1].ID {
		t.Fatal("public cursor consumed lookahead")
	}
	actual := append(s22PageIDs(first), s22Traverse(s, q, *first.NextCursor)...)
	s22AssertSame(t, actual, s22Oracle(s, nil), "lookahead row was consumed/lost")
	t.Run("CrossTimestampContinuation", s22CrossTimestamp)
}

func s22CrossTimestamp(t *testing.T) {
	s := s22New(t)
	// Each two-row page has a different timestamp; larger IDs are older.
	for _, seed := range []struct {
		id  int
		age time.Duration
	}{{2, 0}, {1, 0}, {99, time.Second}, {98, time.Second}, {199, 2 * time.Second}, {198, 2 * time.Second}} {
		r := s.base(seed.id)
		r.CreatedAt = s.now.Add(-seed.age)
		r.UpdatedAt = r.CreatedAt
		s.insert(r)
	}
	expected := []string{s.base(2).ID, s.base(1).ID, s.base(99).ID, s.base(98).ID, s.base(199).ID, s.base(198).ID}
	s22AssertSame(t, s22Oracle(s, nil), expected, "cross-time fixture independent SQL order mismatch")
	s22AssertSame(t, s22Traverse(s, s.query(), ""), expected, "cross-timestamp continuation lost or reordered visible rows")
}
func s22EmptyPage(t *testing.T) {
	for _, mode := range []string{"empty", "all_hidden", "terminal"} {
		t.Run(mode, func(t *testing.T) {
			s := s22New(t)
			if mode != "empty" {
				for i := 1; i <= 3; i++ {
					r := s.base(i)
					if mode == "all_hidden" {
						r.AuthorID = &s.suspended.UserID
					}
					s.insert(r)
				}
			}
			q := s.query()
			expected := s22Oracle(s, nil)
			actual := s22Traverse(s, q, "")
			s22AssertSame(t, actual, expected, "empty/terminal page incorrect")
			if len(expected) == 0 {
				page, err := q.ListArticles(s.ctx, article.ListArticlesRequest{})
				noticeMust(t, err)
				if page.Items == nil || len(page.Items) != 0 || page.NextCursor != nil {
					t.Fatal("empty envelope was not []/null")
				}
			}
		})
	}
}
func s22BetweenPages(t *testing.T) {
	s := s22New(t)
	for i := 1; i <= 6; i++ {
		r := s.base(i)
		if i == 3 {
			r.AuthorID = &s.other.UserID
		}
		s.insert(r)
	}
	q := s.query()
	first, err := q.ListArticles(s.ctx, article.ListArticlesRequest{})
	noticeMust(t, err)
	if first.NextCursor == nil {
		t.Fatal("fixture needs continuation")
	}
	cursor, err := (article.CursorCodec{}).Decode(*first.NextCursor, article.CursorViewMember)
	noticeMust(t, err)
	newest := s.base(99)
	newest.CreatedAt = s.now.Add(time.Second)
	newest.UpdatedAt = newest.CreatedAt
	s.insert(newest)
	edited := s.base(4)
	_, err = s.command().ReplaceContent(s.ctx, s.author, edited.ID, 1, article.CreateArticleInput{Title: "edited-after-first-page", Body: "new body"})
	noticeMust(t, err)
	s.h.exec("UPDATE users SET status='suspended' WHERE id=$1", s.other.UserID)
	tail := s22Traverse(s, q, *first.NextCursor)
	s22AssertSame(t, tail, s22Oracle(s, &cursor), "cross-request changes did not follow current facts/keyset boundary")
	for _, id := range tail {
		if id == newest.ID {
			t.Fatal("newer insertion entered old traversal")
		}
	}
	updated, _, err := q.GetArticle(s.ctx, edited.ID)
	noticeMust(t, err)
	if updated.Title != "edited-after-first-page" {
		t.Fatal("query cached pre-edit content")
	}
	_, _, err = q.GetArticle(s.ctx, s.base(3).ID)
	if !errors.Is(err, article.ErrArticleUnavailable) {
		t.Fatal("query ignored changed author status")
	}
	reopened, err := q.ListArticles(s.ctx, article.ListArticlesRequest{})
	noticeMust(t, err)
	if reopened.Items[0].ID != newest.ID {
		t.Fatal("new insertion did not appear on fresh first page")
	}
}
