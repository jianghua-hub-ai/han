package article

import (
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"testing"
	"time"

	"han/internal/session"
)

func TestAdminCursor(t *testing.T) {
	t.Run("FiveViewIsolation", func(t *testing.T) {
		views := []CursorView{CursorViewMember, CursorViewAdminAll, CursorViewAdminViolating, CursorView("admin_hidden"), CursorViewAdminOrphan}
		for _, view := range views {
			t.Run(string(view), func(t *testing.T) {
				cursor := ArticleCursor{CreatedAt: time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC), ID: "10000000-0000-4000-8000-000000000001", View: view}
				encoded, err := (CursorCodec{}).Encode(cursor)
				if err != nil {
					t.Fatalf("valid view %s rejected: %v", view, err)
				}
				decoded, err := (CursorCodec{}).Decode(encoded, view)
				if err != nil || decoded != cursor {
					t.Fatalf("round trip = %+v/%v, want %+v", decoded, err, cursor)
				}
				for _, other := range views {
					if other == view {
						continue
					}
					t.Run("reject_"+string(other), func(t *testing.T) {
						if _, err := (CursorCodec{}).Decode(encoded, other); !errors.Is(err, ErrInvalidCursor) {
							t.Fatalf("cross view %s -> %s accepted: %v", view, other, err)
						}
					})
				}
			})
		}
	})
}

func TestVisibility_StateMatrix_FailsClosed(t *testing.T) {
	tests := []struct {
		name         string
		mutate       func(*Article)
		authorStatus string
		wantValid    bool
		wantVisible  bool
	}{
		{name: "published clear active author", authorStatus: authorStatusActive, wantValid: true, wantVisible: true},
		{name: "published clear suspended author", authorStatus: authorStatusSuspended, wantValid: true},
		{name: "published clear missing author status", wantValid: true},
		{name: "published clear anonymous", mutate: func(article *Article) {
			article.AuthorID = nil
		}, wantValid: true, wantVisible: true},
		{name: "published violating", mutate: func(article *Article) {
			article.ViolationState = ViolationViolating
			article.ViolationCategory = violationCategoryPointer(ViolationViolence)
		}, authorStatus: authorStatusActive, wantValid: true},
		{name: "held orphan", mutate: func(article *Article) {
			article.AuthorID = nil
			article.OrphanedFromUserID = stringPointer("10000000-0000-4000-8000-000000000003")
			article.PublicationState = PublicationHeld
		}, wantValid: true},
		{name: "deletion pending", mutate: func(article *Article) {
			article.DeletionState = DeletionPending
			article.DeletionOperationID = stringPointer("10000000-0000-4000-8000-000000000004")
		}, authorStatus: authorStatusActive, wantValid: true},
		{name: "held with author", mutate: func(article *Article) {
			article.PublicationState = PublicationHeld
			article.OrphanedFromUserID = stringPointer("10000000-0000-4000-8000-000000000003")
		}, authorStatus: authorStatusActive},
		{name: "held without source", mutate: func(article *Article) {
			article.AuthorID = nil
			article.PublicationState = PublicationHeld
		}},
		{name: "published with orphan source", mutate: func(article *Article) {
			article.AuthorID = nil
			article.OrphanedFromUserID = stringPointer("10000000-0000-4000-8000-000000000003")
		}},
		{name: "violating without category", mutate: func(article *Article) {
			article.ViolationState = ViolationViolating
		}, authorStatus: authorStatusActive},
		{name: "clear with category", mutate: func(article *Article) {
			article.ViolationCategory = violationCategoryPointer(ViolationSexual)
		}, authorStatus: authorStatusActive},
		{name: "pending without operation", mutate: func(article *Article) {
			article.DeletionState = DeletionPending
		}, authorStatus: authorStatusActive},
		{name: "active with operation", mutate: func(article *Article) {
			article.DeletionOperationID = stringPointer("10000000-0000-4000-8000-000000000004")
		}, authorStatus: authorStatusActive},
		{name: "zero version", mutate: func(article *Article) {
			article.Version = 0
		}, authorStatus: authorStatusActive},
	}

	policy := VisibilityPolicy{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			article := validTestArticle()
			if test.mutate != nil {
				test.mutate(&article)
			}

			err := ValidateArticleState(article)
			if test.wantValid && err != nil {
				t.Fatalf("ValidateArticleState() error = %v", err)
			}
			if !test.wantValid && !errors.Is(err, ErrArticleInvalidState) {
				t.Fatalf("ValidateArticleState() error = %v, want ErrArticleInvalidState", err)
			}
			if got := policy.MemberVisible(article, test.authorStatus); got != test.wantVisible {
				t.Fatalf("MemberVisible() = %v, want %v", got, test.wantVisible)
			}
		})
	}
}

func TestVisibility_Projections_IsolateFields(t *testing.T) {
	policy := VisibilityPolicy{}
	memberArticle := validTestArticle()

	memberSummary, err := policy.MemberListProjection(memberArticle, authorStatusActive)
	if err != nil {
		t.Fatalf("MemberListProjection() error = %v", err)
	}
	assertJSONKeys(t, memberSummary, []string{
		"author", "created_at", "id", "title", "updated_at",
	})

	memberDetail, err := policy.MemberDetailProjection(memberArticle, authorStatusActive)
	if err != nil {
		t.Fatalf("MemberDetailProjection() error = %v", err)
	}
	assertJSONKeys(t, memberDetail, []string{
		"author", "body", "created_at", "id", "publication_state",
		"title", "updated_at", "violation_state",
	})

	restrictedArticle := validTestArticle()
	restrictedArticle.ViolationState = ViolationViolating
	restrictedArticle.ViolationCategory = violationCategoryPointer(ViolationSexual)
	author, err := policy.AuthorProjection(restrictedArticle, session.Principal{
		UserID: *restrictedArticle.AuthorID,
		Role:   principalRoleMember,
		Status: principalStatusActive,
	})
	if err != nil {
		t.Fatalf("AuthorProjection() error = %v", err)
	}
	if author.ModerationStatus != ModerationRestricted || !author.Editable {
		t.Fatalf("AuthorProjection() = %#v", author)
	}
	assertJSONKeys(t, author, []string{"body", "editable", "moderation_status", "title"})
	if _, err := policy.MemberDetailProjection(restrictedArticle, authorStatusActive); !errors.Is(err, ErrArticleUnavailable) {
		t.Fatalf("MemberDetailProjection(restricted) error = %v, want ErrArticleUnavailable", err)
	}
	restrictedAdmin, err := policy.AdminProjection(restrictedArticle, session.Principal{
		UserID: "10000000-0000-4000-8000-000000000009",
		Role:   principalRoleAdmin,
		Status: principalStatusActive,
	}, authorStatusActive)
	if err != nil {
		t.Fatalf("AdminProjection(restricted) error = %v", err)
	}
	if restrictedAdmin.ViolationCategory == nil ||
		*restrictedAdmin.ViolationCategory != ViolationSexual ||
		!reflect.DeepEqual(restrictedAdmin.HiddenReasons, []HiddenReason{HiddenViolation}) {
		t.Fatalf("AdminProjection(restricted) = %#v", restrictedAdmin)
	}

	orphan := validTestArticle()
	orphan.AuthorID = nil
	orphan.OrphanedFromUserID = stringPointer("10000000-0000-4000-8000-000000000003")
	orphan.PublicationState = PublicationHeld
	admin, err := policy.AdminProjection(orphan, session.Principal{
		UserID: "10000000-0000-4000-8000-000000000009",
		Role:   principalRoleAdmin,
		Status: principalStatusActive,
	}, "")
	if err != nil {
		t.Fatalf("AdminProjection() error = %v", err)
	}
	if !reflect.DeepEqual(admin.HiddenReasons, []HiddenReason{HiddenOrphanHold}) {
		t.Fatalf("AdminProjection().HiddenReasons = %#v", admin.HiddenReasons)
	}
	assertJSONKeys(t, admin, []string{
		"author", "body", "created_at", "hidden_reasons", "id",
		"orphaned_from_user_id", "publication_state", "title", "updated_at",
		"version", "violation_category", "violation_state",
	})

	encodedAuthor, err := json.Marshal(author)
	if err != nil {
		t.Fatalf("json.Marshal(author) error = %v", err)
	}
	for _, forbidden := range []string{
		"id", "version", "violation_state", "violation_category", "hidden_reasons",
		"publication_state", "orphaned_from_user_id", "deletion_state", "author_status",
	} {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(encodedAuthor, &fields); err != nil {
			t.Fatalf("json.Unmarshal(author) error = %v", err)
		}
		if _, exists := fields[forbidden]; exists {
			t.Fatalf("author projection leaked %q: %s", forbidden, encodedAuthor)
		}
	}
}

func TestVisibility_PrincipalAndUnavailablePaths_FailClosed(t *testing.T) {
	policy := VisibilityPolicy{}
	article := validTestArticle()

	tests := []struct {
		name      string
		project   func() error
		wantError error
	}{
		{
			name: "member cannot read suspended author",
			project: func() error {
				_, err := policy.MemberDetailProjection(article, authorStatusSuspended)
				return err
			},
			wantError: ErrArticleUnavailable,
		},
		{
			name: "different author cannot read controlled projection",
			project: func() error {
				_, err := policy.AuthorProjection(article, session.Principal{
					UserID: "10000000-0000-4000-8000-000000000008",
					Role:   principalRoleMember,
					Status: principalStatusActive,
				})
				return err
			},
			wantError: ErrArticleUnavailable,
		},
		{
			name: "restricted author cannot enter article service",
			project: func() error {
				_, err := policy.AuthorProjection(article, session.Principal{
					UserID:                 *article.AuthorID,
					Role:                   principalRoleMember,
					Status:                 principalStatusActive,
					MustAcknowledgePrivacy: true,
				})
				return err
			},
			wantError: ErrArticleUnavailable,
		},
		{
			name: "member cannot use admin projection",
			project: func() error {
				_, err := policy.AdminProjection(article, session.Principal{
					UserID: "10000000-0000-4000-8000-000000000009",
					Role:   principalRoleMember,
					Status: principalStatusActive,
				}, authorStatusActive)
				return err
			},
			wantError: ErrArticleForbidden,
		},
		{
			name: "deletion pending is unavailable to author",
			project: func() error {
				pending := article
				pending.DeletionState = DeletionPending
				pending.DeletionOperationID = stringPointer(
					"10000000-0000-4000-8000-000000000004",
				)
				_, err := policy.AuthorProjection(pending, session.Principal{
					UserID: *pending.AuthorID,
					Role:   principalRoleMember,
					Status: principalStatusActive,
				})
				return err
			},
			wantError: ErrArticleUnavailable,
		},
		{
			name: "deletion pending is unavailable to admin",
			project: func() error {
				pending := article
				pending.DeletionState = DeletionPending
				pending.DeletionOperationID = stringPointer(
					"10000000-0000-4000-8000-000000000004",
				)
				_, err := policy.AdminProjection(pending, session.Principal{
					UserID: "10000000-0000-4000-8000-000000000009",
					Role:   principalRoleAdmin,
					Status: principalStatusActive,
				}, authorStatusActive)
				return err
			},
			wantError: ErrArticleUnavailable,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.project(); !errors.Is(err, test.wantError) {
				t.Fatalf("projection error = %v, want %v", err, test.wantError)
			}
		})
	}
}

func validTestArticle() Article {
	now := time.Date(2026, time.October, 22, 1, 2, 3, 456000000, time.UTC)
	return Article{
		ID:               "10000000-0000-4000-8000-000000000001",
		AuthorID:         stringPointer("10000000-0000-4000-8000-000000000002"),
		Title:            "title",
		Body:             "body",
		PublicationState: PublicationPublished,
		ViolationState:   ViolationClear,
		DeletionState:    DeletionActive,
		Version:          7,
		CreatedAt:        now,
		UpdatedAt:        now.Add(time.Minute),
	}
}

func assertJSONKeys(t *testing.T, value any, want []string) {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	got := make([]string, 0, len(fields))
	for field := range fields {
		got = append(got, field)
	}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("JSON keys = %v, want %v; body=%s", got, want, encoded)
	}
}

func stringPointer(value string) *string {
	return &value
}

func violationCategoryPointer(value ViolationCategory) *ViolationCategory {
	return &value
}
