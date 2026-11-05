//go:build integration

package articleintegration

import (
	"errors"
	"net/http"
	"testing"

	"han/internal/article"
)

// TestArticleETag verifies real multi-connection version ownership.
func TestArticleETag(t *testing.T) {
	t.Run("SameVersionTwoWriters", s22SameVersion)
	t.Run("EditVersusModeration", s22EditModeration)
	t.Run("MarkRestoreCompetition", s22MarkRestore)
	t.Run("RefreshAndRetry", s22Refresh)
	t.Run("HeaderVersionConsistency", s22HeaderVersion)
}

func s22SameVersion(t *testing.T) {
	s := s22New(t)
	r := s.base(1)
	s.insert(r)
	s22Concurrent(s, r, s22Operation{"edit", "winner"}, s22Operation{"edit", "loser"})
}
func s22EditModeration(t *testing.T) {
	for _, governance := range []string{"mark", "restore"} {
		for _, markFirst := range []bool{false, true} {
			t.Run(governance+map[bool]string{false: "_edit_first", true: "_governance_first"}[markFirst], func(t *testing.T) {
				s := s22New(t)
				r := s22StateRecord(s, s22StateCase{owner: "active", violating: governance == "restore"})
				s.insert(r)
				a, b := s22Operation{"edit", "winner"}, s22Operation{governance, ""}
				if markFirst {
					a, b = b, a
				}
				s22Concurrent(s, r, a, b)
			})
		}
	}
}
func s22MarkRestore(t *testing.T) {
	for _, restore := range []bool{false, true} {
		t.Run(map[bool]string{false: "mark_mark", true: "restore_restore"}[restore], func(t *testing.T) {
			s := s22New(t)
			r := s22StateRecord(s, s22StateCase{owner: "active", violating: restore})
			s.insert(r)
			kind := "mark"
			if restore {
				kind = "restore"
			}
			s22Concurrent(s, r, s22Operation{kind, ""}, s22Operation{kind, ""})
		})
	}
	t.Run("DeletionVersionStatePrecedence", s22Precedence)
}
func s22Precedence(t *testing.T) {
	s := s22New(t)
	clear := s.base(1)
	s.insert(clear)
	cmd := s.command()
	_, _, err := cmd.Moderate(s.ctx, s.admin, clear.ID, 1, nil)
	if !errors.Is(err, article.ErrArticleInvalidState) {
		t.Fatal("current clear restore must be invalid_state")
	}
	category := article.ViolationViolence
	_, _, err = cmd.Moderate(s.ctx, s.admin, clear.ID, 1, &category)
	noticeMust(t, err)
	_, _, err = cmd.Moderate(s.ctx, s.admin, clear.ID, 1, &category)
	if !errors.Is(err, article.ErrPreconditionFailed) {
		t.Fatal("stale version must precede repeated mark state")
	}
	pending := s22StateRecord(s, s22StateCase{owner: "active", pending: true})
	pending.ID = s.base(2).ID
	s.insert(pending)
	_, _, err = cmd.Moderate(s.ctx, s.admin, pending.ID, 99, &category)
	if !errors.Is(err, article.ErrArticleUnavailable) {
		t.Fatal("deletion must precede version mismatch")
	}
}
func s22Refresh(t *testing.T) {
	s := s22New(t)
	r := s.base(1)
	s.insert(r)
	s22Concurrent(s, r, s22Operation{"edit", "first"}, s22Operation{"edit", "stale"})
	current := s.record(r.ID)
	q := s.query()
	_, tag, err := q.GetAuthorArticle(s.ctx, s.author, r.ID)
	noticeMust(t, err)
	h := s.handler(s.command())
	w := s.request(h.UpdateArticle, s.author, http.MethodPut, "/api/v1/articles/"+r.ID, s22Body(article.CreateArticleInput{Title: "retried", Body: "fresh version"}), tag)
	s22AssertStatus(t, w, 200)
	after := s.record(r.ID)
	if after.Version != current.Version+1 || after.Title != "retried" || w.Header().Get("ETag") != s22Tag(t, after) {
		t.Fatal("fresh ETag retry/version failed")
	}
}
func s22HeaderVersion(t *testing.T) {
	s := s22New(t)
	r := s.base(1)
	s.insert(r)
	q := s.query()
	h := s.handler(s.command())
	before := s22Tag(t, r)
	w := s.request(h.UpdateArticle, s.author, http.MethodPut, "/api/v1/articles/"+r.ID, s22Body(article.CreateArticleInput{Title: "updated", Body: "content"}), before)
	s22AssertStatus(t, w, 200)
	stored := s.record(r.ID)
	member, memberTag, err := q.GetArticle(s.ctx, r.ID)
	noticeMust(t, err)
	author, authorTag, err := q.GetAuthorArticle(s.ctx, s.author, r.ID)
	noticeMust(t, err)
	expected := s22Tag(t, stored)
	mh, err := article.NewQueryHandler(q)
	noticeMust(t, err)
	mw := s.request(mh.GetArticle, s.other, http.MethodGet, "/api/v1/articles/"+r.ID, nil, "")
	s22AssertStatus(t, mw, 200)
	ah, err := article.NewAuthorDetailHandler(q)
	noticeMust(t, err)
	aw := s.request(ah.GetAuthorArticle, s.author, http.MethodGet, "/api/v1/users/me/articles/"+r.ID, nil, "")
	s22AssertStatus(t, aw, 200)
	if mw.Header().Get("ETag") != expected || aw.Header().Get("ETag") != expected {
		t.Fatal("actual read Header does not match committed SQL version")
	}
	if w.Header().Get("ETag") != expected || memberTag != expected || authorTag != expected || expected == before || member.ID != stored.ID || author.Title != stored.Title {
		t.Fatal("SQL/DTO/response ETag diverged")
	}
	if err := article.RequireMatchingETag(expected, stored); err != nil {
		t.Fatal("fresh strong ETag mismatch")
	}
	s22AssertSame(t, s22Keys(t, w.Body.Bytes()), []string{"body", "editable", "moderation_status", "title"}, "author response leaked internal version")
}
