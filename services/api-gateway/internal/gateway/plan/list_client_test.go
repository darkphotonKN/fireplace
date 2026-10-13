package plangw

import (
	"context"
	"net"
	"testing"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// listRecorder is a plan-service that remembers the ListPlans request.
type listRecorder struct {
	pb.UnimplementedPlanServiceServer
	got *pb.ListPlansRequest
}

func (r *listRecorder) ListPlans(ctx context.Context, req *pb.ListPlansRequest) (*pb.ListPlansResponse, error) {
	r.got = req
	return &pb.ListPlansResponse{}, nil
}

// clientOver serves srv in-process and returns a Client already holding the
// connection, so no discovery is involved.
func clientOver(t *testing.T, srv pb.PlanServiceServer) *Client {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	s := grpc.NewServer()
	pb.RegisterPlanServiceServer(s, srv)
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

// R22: GET /api/plans is owned ∪ shared, so the gateway asks plan-service for
// the shared plans too.
func TestClientListPlans_AsksForSharedPlans(t *testing.T) {
	srv := &listRecorder{}
	c := clientOver(t, srv)
	caller := uuid.New()

	if _, err := c.ListPlans(context.Background(), caller); err != nil {
		t.Fatalf("ListPlans: %v", err)
	}
	if srv.got == nil || srv.got.UserId != caller.String() || !srv.got.IncludeShared {
		t.Errorf("request = %+v, want user_id %s with include_shared", srv.got, caller)
	}
}
