package config

import (
	"context"
	"testing"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	"github.com/darkphotonKN/fireplace/services/plan-service/internal/checklistitem"
	"github.com/darkphotonKN/fireplace/services/plan-service/internal/notecomment"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FS-65B3N: the comment RPCs reach notecomment's handler, not the
// Unimplemented stub checklistitem's handler embeds. A malformed id is
// refused by notecomment itself, before any dependency is used.
func TestChecklistServer_CommentRPCsAreServed(t *testing.T) {
	var srv pb.ChecklistServiceServer = checklistServer{
		checklistitem.NewHandler(nil), notecomment.NewHandler(nil),
	}
	ctx := context.Background()

	_, listErr := srv.ListNoteComments(ctx, &pb.ListNoteCommentsRequest{PlanId: "x"})
	_, createErr := srv.CreateNoteComment(ctx, &pb.CreateNoteCommentRequest{PlanId: "x"})
	_, getErr := srv.GetNoteComment(ctx, &pb.GetNoteCommentRequest{PlanId: "x"})
	_, updateErr := srv.UpdateNoteComment(ctx, &pb.UpdateNoteCommentRequest{PlanId: "x"})
	_, deleteErr := srv.DeleteNoteComment(ctx, &pb.DeleteNoteCommentRequest{PlanId: "x"})

	for name, err := range map[string]error{
		"list": listErr, "create": createErr, "get": getErr, "update": updateErr, "delete": deleteErr,
	} {
		if status.Code(err) != codes.InvalidArgument {
			t.Errorf("%s: class = %v, want InvalidArgument from notecomment (%v)", name, status.Code(err), err)
		}
	}
}
