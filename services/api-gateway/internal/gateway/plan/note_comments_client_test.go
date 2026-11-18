package plangw

import (
	"context"
	"net"
	"testing"
	"time"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// commentRecorder is a plan-service that remembers the comment requests.
type commentRecorder struct {
	pb.UnimplementedChecklistServiceServer
	list   *pb.ListNoteCommentsRequest
	create *pb.CreateNoteCommentRequest
	reply  []*pb.NoteComment

	get    *pb.GetNoteCommentRequest
	update *pb.UpdateNoteCommentRequest
	del    *pb.DeleteNoteCommentRequest
}

func (r *commentRecorder) GetNoteComment(_ context.Context, req *pb.GetNoteCommentRequest) (*pb.NoteComment, error) {
	r.get = req
	return r.reply[0], nil
}

func (r *commentRecorder) UpdateNoteComment(_ context.Context, req *pb.UpdateNoteCommentRequest) (*pb.NoteComment, error) {
	r.update = req
	return r.reply[0], nil
}

func (r *commentRecorder) DeleteNoteComment(_ context.Context, req *pb.DeleteNoteCommentRequest) (*emptypb.Empty, error) {
	r.del = req
	return &emptypb.Empty{}, nil
}

func (r *commentRecorder) ListNoteComments(_ context.Context, req *pb.ListNoteCommentsRequest) (*pb.ListNoteCommentsResponse, error) {
	r.list = req
	return &pb.ListNoteCommentsResponse{Comments: r.reply}, nil
}

func (r *commentRecorder) CreateNoteComment(_ context.Context, req *pb.CreateNoteCommentRequest) (*pb.NoteComment, error) {
	r.create = req
	return r.reply[0], nil
}

func checklistClientOver(t *testing.T, srv pb.ChecklistServiceServer) *Client {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	s := grpc.NewServer()
	pb.RegisterChecklistServiceServer(s, srv)
	go s.Serve(lis)
	t.Cleanup(s.Stop)
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return &Client{conn: conn}
}

func TestClientNoteComments_SendTheCallerAndMapTheReply(t *testing.T) {
	id, item, author := uuid.New(), uuid.New(), uuid.New()
	created, edited := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC), time.Date(2026, 5, 2, 9, 0, 0, 0, time.UTC)
	srv := &commentRecorder{reply: []*pb.NoteComment{
		{Id: id.String(), ItemId: item.String(), AuthorId: author.String(), Body: "b", CreatedAt: timestamppb.New(created), EditedAt: timestamppb.New(edited)},
	}}
	c := checklistClientOver(t, srv)
	plan, caller := uuid.New(), uuid.New()
	ctx := context.Background()

	listed, err := c.ListNoteComments(ctx, plan, item, caller)
	if err != nil {
		t.Fatalf("ListNoteComments: %v", err)
	}
	if srv.list.GetPlanId() != plan.String() || srv.list.GetItemId() != item.String() || srv.list.GetUserId() != caller.String() {
		t.Fatalf("list request = %v", srv.list)
	}
	if len(listed) != 1 || listed[0].ID != id || listed[0].AuthorID != author || listed[0].Body != "b" ||
		!listed[0].CreatedAt.Equal(created) || listed[0].EditedAt == nil || !listed[0].EditedAt.Equal(edited) {
		t.Fatalf("listed = %+v", listed)
	}

	srv.reply[0].EditedAt = nil
	made, err := c.CreateNoteComment(ctx, plan, item, caller, "hello")
	if err != nil {
		t.Fatalf("CreateNoteComment: %v", err)
	}
	if srv.create.GetUserId() != caller.String() || srv.create.GetBody() != "hello" || srv.create.GetPlanId() != plan.String() {
		t.Fatalf("create request = %v", srv.create)
	}
	if made.EditedAt != nil {
		t.Fatalf("editedAt = %v, want nil", made.EditedAt)
	}
}

// The comment-by-id calls send the whole reference with the caller.
func TestClientNoteCommentByID_SendTheReference(t *testing.T) {
	comment, item := uuid.New(), uuid.New()
	srv := &commentRecorder{reply: []*pb.NoteComment{
		{Id: comment.String(), ItemId: item.String(), AuthorId: uuid.NewString(), Body: "b", CreatedAt: timestamppb.Now()},
	}}
	c := checklistClientOver(t, srv)
	plan, caller := uuid.New(), uuid.New()
	ctx := context.Background()

	if _, err := c.GetNoteComment(ctx, plan, item, comment, caller); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if _, err := c.UpdateNoteComment(ctx, plan, item, comment, caller, "new"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if err := c.DeleteNoteComment(ctx, plan, item, comment, caller); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	for name, got := range map[string][4]string{
		"get":    {srv.get.GetPlanId(), srv.get.GetItemId(), srv.get.GetCommentId(), srv.get.GetUserId()},
		"update": {srv.update.GetPlanId(), srv.update.GetItemId(), srv.update.GetCommentId(), srv.update.GetUserId()},
		"delete": {srv.del.GetPlanId(), srv.del.GetItemId(), srv.del.GetCommentId(), srv.del.GetUserId()},
	} {
		if got != [4]string{plan.String(), item.String(), comment.String(), caller.String()} {
			t.Errorf("%s sent %v", name, got)
		}
	}
	if srv.update.GetBody() != "new" {
		t.Errorf("update body = %q", srv.update.GetBody())
	}
}
