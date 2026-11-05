//go:build contract

package articlecontract

import (
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"han/internal/article"
	"han/internal/session"
)

const (
	articleID     = "10000000-0000-4000-8000-000000000001"
	authorID      = "20000000-0000-4000-8000-000000000002"
	adminID       = "30000000-0000-4000-8000-000000000003"
	targetID      = "40000000-0000-4000-8000-000000000004"
	noticeID      = "50000000-0000-4000-8000-000000000005"
	auditID       = "60000000-0000-4000-8000-000000000006"
	traceID       = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	privateCanary = "private-contact-lease-generation-canary"
)

var fixtureTime = time.Date(2026, 10, 28, 0, 0, 0, 123456000, time.UTC)

func baseArticle() article.Article {
	owner := authorID
	return article.Article{ID: articleID, AuthorID: &owner, Title: " <b>标题</b> ", Body: " **正文🙂** ",
		PublicationState: article.PublicationPublished, ViolationState: article.ViolationClear,
		DeletionState: article.DeletionActive, Version: 2, CreatedAt: fixtureTime, UpdatedAt: fixtureTime}
}

func principal(role string) session.Principal {
	id := authorID
	if role == "admin" {
		id = adminID
	}
	return session.Principal{SessionID: "synthetic-session", UserID: id, Role: role, Status: "active"}
}

func TestArticleProjectionContract(t *testing.T) {
	t.Run("MemberViews", projectionMembers)
	t.Run("AuthorView", projectionAuthor)
	t.Run("AdminViews", projectionAdmin)
	t.Run("AnonymousAndRejectedState", projectionRejected)
	t.Run("PrivacyCanaries", projectionPrivacy)
}

func jsonObject(t *testing.T, raw []byte, keys ...string) map[string]json.RawMessage {
	t.Helper()
	var got map[string]json.RawMessage
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for name := range got {
		names = append(names, name)
	}
	want := append([]string{}, keys...)
	sort.Strings(names)
	sort.Strings(want)
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("JSON keys=%v expected=%v", names, want)
	}
	return got
}

func marshal(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func fieldValue(t *testing.T, object map[string]json.RawMessage, key string, want any) {
	t.Helper()
	var got, expected any
	if err := json.Unmarshal(object[key], &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(marshal(t, want), &expected); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("JSON field %s differs from independent fixture", key)
	}
}

func assertMember(t *testing.T, raw []byte, row article.Article, detail bool) {
	t.Helper()
	keys := []string{"id", "title", "author", "created_at", "updated_at"}
	if detail {
		keys = append(keys, "body", "publication_state", "violation_state")
	}
	obj := jsonObject(t, raw, keys...)
	fieldValue(t, obj, "id", row.ID)
	fieldValue(t, obj, "title", row.Title)
	fieldValue(t, obj, "created_at", row.CreatedAt)
	fieldValue(t, obj, "updated_at", row.UpdatedAt)
	assertAuthorReference(t, obj["author"], row.AuthorID)
	if detail {
		fieldValue(t, obj, "body", row.Body)
		fieldValue(t, obj, "publication_state", row.PublicationState)
		fieldValue(t, obj, "violation_state", row.ViolationState)
	}
}

func assertAuthorReference(t *testing.T, raw []byte, owner *string) {
	t.Helper()
	if owner == nil {
		obj := jsonObject(t, raw, "kind", "id", "label")
		fieldValue(t, obj, "kind", "anonymous")
		fieldValue(t, obj, "id", nil)
		fieldValue(t, obj, "label", "匿名")
		return
	}
	obj := jsonObject(t, raw, "kind", "id")
	fieldValue(t, obj, "kind", "user")
	fieldValue(t, obj, "id", *owner)
}

func assertAuthor(t *testing.T, raw []byte, row article.Article, restricted bool) {
	t.Helper()
	obj := jsonObject(t, raw, "title", "body", "moderation_status", "editable")
	status := "normal"
	if restricted {
		status = "restricted"
	}
	fieldValue(t, obj, "title", row.Title)
	fieldValue(t, obj, "body", row.Body)
	fieldValue(t, obj, "moderation_status", status)
	fieldValue(t, obj, "editable", true)
}

func assertAdmin(t *testing.T, raw []byte, row article.Article, reasons []string) {
	t.Helper()
	obj := jsonObject(t, raw, "id", "title", "body", "author", "publication_state", "violation_state", "violation_category", "hidden_reasons", "orphaned_from_user_id", "version", "created_at", "updated_at")
	for key, value := range map[string]any{"id": row.ID, "title": row.Title, "body": row.Body, "publication_state": row.PublicationState, "violation_state": row.ViolationState, "violation_category": row.ViolationCategory, "hidden_reasons": reasons, "orphaned_from_user_id": row.OrphanedFromUserID, "version": row.Version, "created_at": row.CreatedAt, "updated_at": row.UpdatedAt} {
		fieldValue(t, obj, key, value)
	}
	assertAuthorReference(t, obj["author"], row.AuthorID)
}

func projectionMembers(t *testing.T) {
	row := baseArticle()
	policy := article.VisibilityPolicy{}
	list, err := policy.MemberListProjection(row, "active")
	must(t, err)
	assertMember(t, marshal(t, list), row, false)
	detail, err := policy.MemberDetailProjection(row, "active")
	must(t, err)
	assertMember(t, marshal(t, detail), row, true)
	if strings.Contains(string(marshal(t, list)), row.Body) {
		t.Fatal("member list leaked body")
	}
}

func projectionAuthor(t *testing.T) {
	for _, restricted := range []bool{false, true} {
		t.Run(map[bool]string{false: "normal", true: "restricted"}[restricted], func(t *testing.T) {
			row := baseArticle()
			if restricted {
				category := article.ViolationViolence
				row.ViolationState = article.ViolationViolating
				row.ViolationCategory = &category
			}
			view, err := (article.VisibilityPolicy{}).AuthorProjection(row, principal("member"))
			must(t, err)
			assertAuthor(t, marshal(t, view), row, restricted)
		})
	}
}

func projectionAdmin(t *testing.T) {
	for _, state := range []string{"clear", "suspended_violation", "held_violation"} {
		t.Run(state, func(t *testing.T) {
			row, status, reasons := baseArticle(), "active", []string{}
			if state != "clear" {
				c := article.ViolationSexual
				row.ViolationState = article.ViolationViolating
				row.ViolationCategory = &c
				reasons = []string{"violation"}
			}
			if state == "suspended_violation" {
				status = "suspended"
				reasons = []string{"author_suspended", "violation"}
			}
			if state == "held_violation" {
				source := authorID
				row.AuthorID = nil
				row.OrphanedFromUserID = &source
				row.PublicationState = article.PublicationHeld
				reasons = []string{"violation", "orphan_hold"}
			}
			view, err := (article.VisibilityPolicy{}).AdminProjection(row, principal("admin"), status)
			must(t, err)
			assertAdmin(t, marshal(t, view), row, reasons)
		})
	}
}

func projectionRejected(t *testing.T) {
	row := baseArticle()
	row.AuthorID = nil
	view, err := (article.VisibilityPolicy{}).MemberDetailProjection(row, "deleted")
	must(t, err)
	assertMember(t, marshal(t, view), row, true)
	for _, mode := range []string{"pending", "non_author", "non_admin", "invalid"} {
		t.Run(mode, func(t *testing.T) {
			r := baseArticle()
			p := principal("member")
			policy := article.VisibilityPolicy{}
			switch mode {
			case "pending":
				op := privateCanary
				r.DeletionState = article.DeletionPending
				r.DeletionOperationID = &op
			case "invalid":
				r.Version = 0
			case "non_author":
				p.UserID = targetID
			}
			if mode == "non_admin" {
				_, err = policy.AdminProjection(r, p, "active")
				if !errors.Is(err, article.ErrArticleForbidden) {
					t.Fatal("nonadmin accepted")
				}
				return
			}
			_, err := policy.AuthorProjection(r, p)
			if !errors.Is(err, article.ErrArticleUnavailable) {
				t.Fatal("rejected author state accepted")
			}
		})
	}
}

func projectionPrivacy(t *testing.T) {
	row := baseArticle()
	row.Body = privateCanary
	policy := article.VisibilityPolicy{}
	list, err := policy.MemberListProjection(row, "active")
	must(t, err)
	if strings.Contains(string(marshal(t, list)), privateCanary) {
		t.Fatal("list leaked body canary")
	}
	category := article.ViolationViolence
	row.ViolationState = article.ViolationViolating
	row.ViolationCategory = &category
	author, err := policy.AuthorProjection(row, principal("member"))
	must(t, err)
	assertAuthor(t, marshal(t, author), row, true)
	for _, key := range []string{"violation_category", "hidden_reasons", "deletion_operation_id", "generation", "token", "email", "status"} {
		var fields map[string]json.RawMessage
		must(t, json.Unmarshal(marshal(t, author), &fields))
		if fields[key] != nil {
			t.Fatalf("author leaked internal key %s", key)
		}
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("operation failed: %T", err)
	}
}
