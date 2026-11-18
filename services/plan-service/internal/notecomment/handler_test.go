package notecomment

import (
	"context"
	"fmt"
	"testing"
	"time"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeService struct {
	comments []*Comment
	err      error

	created *CreateInput
	ref     *CommentRef
	body    string
	calls   int
}

func (f *fakeService) Get(_ context.Context, ref CommentRef) (*Comment, error) {
	f.calls++
	f.ref = &ref
	if f.err != nil {
		return nil, f.err
	}
	return &Comment{ID: ref.CommentID, ItemID: ref.ItemID, AuthorID: uuid.New(), Body: "b", CreatedAt: time.Now()}, nil
}

func (f *fakeService) Update(_ context.Context, ref CommentRef, body string) (*Comment, error) {
	f.calls++
	f.ref, f.body = &ref, body
	if f.err != nil {
		return nil, f.err
	}
	now := time.Now()
	return &Comment{ID: ref.CommentID, ItemID: ref.ItemID, AuthorID: ref.UserID, Body: body, CreatedAt: now, EditedAt: &now}, nil
}

func (f *fakeService) Delete(_ context.Context, ref CommentRef) error {
	f.calls++
	f.ref = &ref
	return f.err
}

func (f *fakeService) Create(_ context.Context, in CreateInput) (*Comment, error) {
	f.calls++
	f.created = &in
	if f.err != nil {
		return nil, f.err
	}
	return &Comment{ID: uuid.New(), ItemID: in.ItemID, AuthorID: in.AuthorID, Body: in.Body, CreatedAt: time.Now()}, nil
}

func (f *fakeService) List(context.Context, uuid.UUID, uuid.UUID) ([]*Comment, error) {
	f.calls++
	return f.comments, f.err
}

// R6: the author is the request's user_id, the caller as the gateway fills it.
func TestCreateNoteComment_AuthorIsTheCaller(t *testing.T) {
	svc := &fakeService{}
	plan, item, user := uuid.New(), uuid.New(), uuid.New()

	got, err := NewHandler(svc).CreateNoteComment(context.Background(), &pb.CreateNoteCommentRequest{
		PlanId: plan.String(), ItemId: item.String(), UserId: user.String(), Body: "hi",
	})

	if err != nil {
		t.Fatalf("CreateNoteComment: %v", err)
	}
	if svc.created.PlanID != plan || svc.created.ItemID != item || svc.created.AuthorID != user || svc.created.Body != "hi" {
		t.Fatalf("service got %+v", svc.created)
	}
	if got.GetAuthorId() != user.String() || got.GetCreatedAt() == nil || got.GetEditedAt() != nil {
		t.Fatalf("comment = %v", got)
	}
}

func TestListNoteComments_MapsEveryField(t *testing.T) {
	created, edited := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC), time.Date(2026, 5, 2, 9, 0, 0, 0, time.UTC)
	c := &Comment{ID: uuid.New(), ItemID: uuid.New(), AuthorID: uuid.New(), Body: "b", CreatedAt: created, EditedAt: &edited}

	resp, err := NewHandler(&fakeService{comments: []*Comment{c}}).ListNoteComments(context.Background(), &pb.ListNoteCommentsRequest{
		PlanId: uuid.NewString(), ItemId: c.ItemID.String(), UserId: uuid.NewString(),
	})

	if err != nil || len(resp.GetComments()) != 1 {
		t.Fatalf("resp = %v, err = %v", resp, err)
	}
	got := resp.GetComments()[0]
	if got.GetId() != c.ID.String() || got.GetItemId() != c.ItemID.String() || got.GetAuthorId() != c.AuthorID.String() ||
		got.GetBody() != "b" || !got.GetCreatedAt().AsTime().Equal(created) || !got.GetEditedAt().AsTime().Equal(edited) {
		t.Fatalf("comment = %v", got)
	}
}

// Malformed ids are InvalidArgument and never reach the service.
func TestNoteCommentRPCs_MalformedIds(t *testing.T) {
	ok := uuid.NewString()
	cases := map[string]func(h *Handler) error{
		"list plan": func(h *Handler) error {
			_, err := h.ListNoteComments(context.Background(), &pb.ListNoteCommentsRequest{PlanId: "x", ItemId: ok, UserId: ok})
			return err
		},
		"list item": func(h *Handler) error {
			_, err := h.ListNoteComments(context.Background(), &pb.ListNoteCommentsRequest{PlanId: ok, ItemId: "x", UserId: ok})
			return err
		},
		"create user": func(h *Handler) error {
			_, err := h.CreateNoteComment(context.Background(), &pb.CreateNoteCommentRequest{PlanId: ok, ItemId: ok, UserId: "", Body: "b"})
			return err
		},
	}
	for name, call := range cases {
		t.Run(name, func(t *testing.T) {
			svc := &fakeService{}
			if err := call(NewHandler(svc)); status.Code(err) != codes.InvalidArgument || svc.calls != 0 {
				t.Fatalf("class = %v, calls = %d; want InvalidArgument and no call", status.Code(err), svc.calls)
			}
		})
	}
}

// The three comment-by-id RPCs carry the whole (plan, item, comment, caller)
// reference to the service.
func TestCommentByIDRPCs_CarryTheReference(t *testing.T) {
	plan, item, comment, user := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	want := CommentRef{PlanID: plan, ItemID: item, CommentID: comment, UserID: user}
	ctx := context.Background()
	calls := map[string]func(h *Handler) error{
		"get": func(h *Handler) error {
			_, err := h.GetNoteComment(ctx, &pb.GetNoteCommentRequest{PlanId: plan.String(), ItemId: item.String(), CommentId: comment.String(), UserId: user.String()})
			return err
		},
		"update": func(h *Handler) error {
			got, err := h.UpdateNoteComment(ctx, &pb.UpdateNoteCommentRequest{PlanId: plan.String(), ItemId: item.String(), CommentId: comment.String(), UserId: user.String(), Body: "new"})
			if err == nil && (got.GetBody() != "new" || got.GetEditedAt() == nil) {
				t.Errorf("update answered %v", got)
			}
			return err
		},
		"delete": func(h *Handler) error {
			_, err := h.DeleteNoteComment(ctx, &pb.DeleteNoteCommentRequest{PlanId: plan.String(), ItemId: item.String(), CommentId: comment.String(), UserId: user.String()})
			return err
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			svc := &fakeService{}
			if err := call(NewHandler(svc)); err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if svc.ref == nil || *svc.ref != want {
				t.Fatalf("service got %v, want %v", svc.ref, want)
			}
		})
	}
}

// R19: the backstop's refusal leaves as PermissionDenied; a malformed
// comment id as InvalidArgument without reaching the service.
func TestCommentByIDRPCs_Refusals(t *testing.T) {
	ok := uuid.NewString()
	svc := &fakeService{err: fmt.Errorf("x: %w", commonconstants.ErrForbidden)}
	_, err := NewHandler(svc).DeleteNoteComment(context.Background(), &pb.DeleteNoteCommentRequest{PlanId: ok, ItemId: ok, CommentId: ok, UserId: ok})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("class = %v, want PermissionDenied", status.Code(err))
	}

	svc = &fakeService{}
	_, err = NewHandler(svc).UpdateNoteComment(context.Background(), &pb.UpdateNoteCommentRequest{PlanId: ok, ItemId: ok, CommentId: "x", UserId: ok, Body: "b"})
	if status.Code(err) != codes.InvalidArgument || svc.calls != 0 {
		t.Fatalf("class = %v, calls = %d; want InvalidArgument and none", status.Code(err), svc.calls)
	}
}
