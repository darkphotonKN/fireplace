package notecomment

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/common/errcode"
	commongrpc "github.com/darkphotonKN/fireplace/common/grpcerror"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeRepo plays the table: it answers err when set, else records what it is
// asked to store.
type fakeRepo struct {
	err      error
	comments []*Comment

	created *CreateInput
	updated *string
	deleted *CommentRef
}

func (f *fakeRepo) Get(_ context.Context, ref CommentRef) (*Comment, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &Comment{ID: ref.CommentID, ItemID: ref.ItemID}, nil
}

func (f *fakeRepo) Update(_ context.Context, ref CommentRef, body string) (*Comment, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.updated = &body
	return &Comment{ID: ref.CommentID, ItemID: ref.ItemID, AuthorID: ref.UserID, Body: body}, nil
}

func (f *fakeRepo) Delete(_ context.Context, ref CommentRef) error {
	if f.err != nil {
		return f.err
	}
	f.deleted = &ref
	return nil
}

func (f *fakeRepo) Create(_ context.Context, in CreateInput) (*Comment, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.created = &in
	return &Comment{ID: uuid.New(), ItemID: in.ItemID, AuthorID: in.AuthorID, Body: in.Body}, nil
}

func (f *fakeRepo) List(context.Context, uuid.UUID, uuid.UUID) ([]*Comment, error) {
	return f.comments, f.err
}

func createIn(body string) CreateInput {
	return CreateInput{PlanID: uuid.New(), ItemID: uuid.New(), AuthorID: uuid.New(), Body: body}
}

// R6, R8: the body is trimmed, inner line breaks kept, and the author is the
// caller the request names.
func TestCreate_StoresTheTrimmedBodyAsTheCaller(t *testing.T) {
	repo := &fakeRepo{}
	in := createIn("  first line\nsecond line \n")

	got, err := NewService(repo).Create(context.Background(), in)

	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if repo.created == nil || repo.created.Body != "first line\nsecond line" || repo.created.AuthorID != in.AuthorID {
		t.Fatalf("stored %+v, want the trimmed body by the caller", repo.created)
	}
	if got.AuthorID != in.AuthorID {
		t.Fatalf("author = %s, want %s", got.AuthorID, in.AuthorID)
	}
}

// R9: 1 to 2000 code points after trimming; anything else is InvalidArgument
// and nothing is stored.
func TestCreate_BodyLength(t *testing.T) {
	cases := []struct {
		name string
		body string
		ok   bool
	}{
		{"empty", "", false},
		{"whitespace", " \n\t ", false},
		{"2000 multi-byte", strings.Repeat("é", 1999) + "😀", true},
		{"2001 after trim", "  " + strings.Repeat("a", 2001) + "  ", false},
		{"2000 after trim", "  " + strings.Repeat("a", 2000) + "  ", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepo{}
			_, err := NewService(repo).Create(context.Background(), createIn(tc.body))
			if tc.ok {
				if err != nil || repo.created == nil {
					t.Fatalf("err = %v, stored %v; want stored", err, repo.created != nil)
				}
				return
			}
			if !errors.Is(err, commonconstants.ErrInvalidInput) {
				t.Fatalf("err = %v, want ErrInvalidInput", err)
			}
			if repo.created != nil {
				t.Fatal("an invalid body was stored")
			}
		})
	}
}

// R3: a task is FailedPrecondition with ITEM_NOT_A_NOTE, on create and on list.
func TestNotANote_IsFailedPreconditionWithItsCode(t *testing.T) {
	notANote := fmt.Errorf("item x: %w", ErrNotANote)
	calls := map[string]func(s *service) error{
		"create": func(s *service) error { _, err := s.Create(context.Background(), createIn("hi")); return err },
		"list":   func(s *service) error { _, err := s.List(context.Background(), uuid.New(), uuid.New()); return err },
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			err := call(NewService(&fakeRepo{err: notANote}))
			if status.Code(err) != codes.FailedPrecondition {
				t.Fatalf("class = %v, want FailedPrecondition (%v)", status.Code(err), err)
			}
			if code, ok := commongrpc.DomainCode(err); !ok || code != errcode.ItemNotANote {
				t.Fatalf("code = %q (ok %v), want ITEM_NOT_A_NOTE", code, ok)
			}
		})
	}
}

// R2: an item not in the plan stays a NotFound.
func TestNotFound_PassesThrough(t *testing.T) {
	_, err := NewService(&fakeRepo{err: commonconstants.ErrNotFound}).List(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, commonconstants.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

// R11: no comments is an empty list, never nil.
func TestList_NoneIsEmpty(t *testing.T) {
	got, err := NewService(&fakeRepo{}).List(context.Background(), uuid.New(), uuid.New())
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("got %v, err %v; want an empty list", got, err)
	}
}

// R10: an edit follows the create's body rules: trimmed, 1-2000 code points,
// and nothing is written when it fails.
func TestUpdate_BodyRules(t *testing.T) {
	ref := CommentRef{PlanID: uuid.New(), ItemID: uuid.New(), CommentID: uuid.New(), UserID: uuid.New()}
	cases := []struct {
		name, body, stored string
		ok                 bool
	}{
		{"trimmed", "  new text \n", "new text", true},
		{"empty", "   ", "", false},
		{"over 2000", strings.Repeat("😀", 2001), "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepo{}
			_, err := NewService(repo).Update(context.Background(), ref, tc.body)
			if !tc.ok {
				if !errors.Is(err, commonconstants.ErrInvalidInput) || repo.updated != nil {
					t.Fatalf("err = %v, written %v; want ErrInvalidInput and nothing written", err, repo.updated != nil)
				}
				return
			}
			if err != nil || repo.updated == nil || *repo.updated != tc.stored {
				t.Fatalf("err = %v, written %v; want %q", err, repo.updated, tc.stored)
			}
		})
	}
}

// R19: the backstop's refusal keeps its sentinel, so it leaves as
// PermissionDenied.
func TestUpdateAndDelete_ForbiddenPassesThrough(t *testing.T) {
	repo := &fakeRepo{err: fmt.Errorf("not theirs: %w", commonconstants.ErrForbidden)}
	ref := CommentRef{}
	if _, err := NewService(repo).Update(context.Background(), ref, "x"); !errors.Is(err, commonconstants.ErrForbidden) {
		t.Fatalf("update err = %v, want ErrForbidden", err)
	}
	if err := NewService(repo).Delete(context.Background(), ref); !errors.Is(err, commonconstants.ErrForbidden) {
		t.Fatalf("delete err = %v, want ErrForbidden", err)
	}
}
