package orchestrator

import (
	"context"
	"os"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/orchestrator"
)

// Service is the narrow interface the handler depends on (injected by config).
type Service interface {
	Ping(ctx context.Context, msg string) string
}

type Handler struct {
	pb.UnimplementedOrchestratorServiceServer
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Ping is implemented end-to-end so the service can be smoke-tested via grpcurl
// once the stack is up:
//
//	grpcurl -plaintext -d '{"message":"hi"}' localhost:7105 orchestrator.OrchestratorService/Ping
func (h *Handler) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	hostname, _ := os.Hostname()
	reply := h.service.Ping(ctx, req.Message)
	return &pb.PingResponse{
		Reply:    reply,
		ServedBy: hostname,
	}, nil
}
