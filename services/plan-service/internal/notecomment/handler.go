package notecomment

import (
	"context"
	"fmt"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	commongrpc "github.com/darkphotonKN/fireplace/common/grpcerror"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service is the comment rules the handler serves.
type Service interface {
	Create(ctx context.Context, in CreateInput) (*Comment, error)
	List(ctx context.Context, planID, itemID uuid.UUID) ([]*Comment, error)
	Get(ctx context.Context, ref CommentRef) (*Comment, error)
	Update(ctx context.Context, ref CommentRef, body string) (*Comment, error)
	Delete(ctx context.Context, ref CommentRef) error
}

// Handler serves ChecklistService's comment RPCs. It deliberately does not
// embed pb.UnimplementedChecklistServiceServer: the composition root joins it
// with checklistitem's handler, which does, into one ChecklistService server.
type Handler struct {
	service Service
}

func NewHandler(s Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) ListNoteComments(ctx context.Context, req *pb.ListNoteCommentsRequest) (*pb.ListNoteCommentsResponse, error) {
	const op = "notecomment: list"
	planID, itemID, _, err := parseRef(req.GetPlanId(), req.GetItemId(), req.GetUserId())
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, err)
	}
	comments, err := h.service.List(ctx, planID, itemID)
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, err)
	}
	out := make([]*pb.NoteComment, 0, len(comments))
	for _, c := range comments {
		out = append(out, toProto(c))
	}
	return &pb.ListNoteCommentsResponse{Comments: out}, nil
}

func (h *Handler) CreateNoteComment(ctx context.Context, req *pb.CreateNoteCommentRequest) (*pb.NoteComment, error) {
	const op = "notecomment: create"
	planID, itemID, userID, err := parseRef(req.GetPlanId(), req.GetItemId(), req.GetUserId())
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, err)
	}
	c, err := h.service.Create(ctx, CreateInput{PlanID: planID, ItemID: itemID, AuthorID: userID, Body: req.GetBody()})
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, err)
	}
	return toProto(c), nil
}

func (h *Handler) GetNoteComment(ctx context.Context, req *pb.GetNoteCommentRequest) (*pb.NoteComment, error) {
	const op = "notecomment: get"
	ref, err := parseCommentRef(req.GetPlanId(), req.GetItemId(), req.GetCommentId(), req.GetUserId())
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, err)
	}
	c, err := h.service.Get(ctx, ref)
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, err)
	}
	return toProto(c), nil
}

func (h *Handler) UpdateNoteComment(ctx context.Context, req *pb.UpdateNoteCommentRequest) (*pb.NoteComment, error) {
	const op = "notecomment: update"
	ref, err := parseCommentRef(req.GetPlanId(), req.GetItemId(), req.GetCommentId(), req.GetUserId())
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, err)
	}
	c, err := h.service.Update(ctx, ref, req.GetBody())
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, err)
	}
	return toProto(c), nil
}

func (h *Handler) DeleteNoteComment(ctx context.Context, req *pb.DeleteNoteCommentRequest) (*emptypb.Empty, error) {
	const op = "notecomment: delete"
	ref, err := parseCommentRef(req.GetPlanId(), req.GetItemId(), req.GetCommentId(), req.GetUserId())
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, err)
	}
	if err := h.service.Delete(ctx, ref); err != nil {
		return nil, commongrpc.Fail(ctx, op, err)
	}
	return &emptypb.Empty{}, nil
}

// parseCommentRef parses the (plan, item, comment, caller) of a comment-by-id
// RPC.
func parseCommentRef(rawPlan, rawItem, rawComment, rawUser string) (CommentRef, error) {
	planID, itemID, userID, err := parseRef(rawPlan, rawItem, rawUser)
	if err != nil {
		return CommentRef{}, err
	}
	commentID, err := uuid.Parse(rawComment)
	if err != nil {
		return CommentRef{}, badUUID("comment_id", rawComment)
	}
	return CommentRef{PlanID: planID, ItemID: itemID, CommentID: commentID, UserID: userID}, nil
}

// parseRef parses the (plan, item, caller) every comment RPC carries.
func parseRef(rawPlan, rawItem, rawUser string) (planID, itemID, userID uuid.UUID, err error) {
	if planID, err = uuid.Parse(rawPlan); err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, badUUID("plan_id", rawPlan)
	}
	if itemID, err = uuid.Parse(rawItem); err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, badUUID("item_id", rawItem)
	}
	if userID, err = uuid.Parse(rawUser); err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, badUUID("user_id", rawUser)
	}
	return planID, itemID, userID, nil
}

func badUUID(field, value string) error {
	return fmt.Errorf("%w: %s %q", commonconstants.ErrUUIDCouldNotBeParsed, field, value)
}

func toProto(c *Comment) *pb.NoteComment {
	out := &pb.NoteComment{
		Id:        c.ID.String(),
		ItemId:    c.ItemID.String(),
		AuthorId:  c.AuthorID.String(),
		Body:      c.Body,
		CreatedAt: timestamppb.New(c.CreatedAt),
	}
	if c.EditedAt != nil {
		out.EditedAt = timestamppb.New(*c.EditedAt)
	}
	return out
}
