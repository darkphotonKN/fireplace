package auth

import (
	"context"
	"errors"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/auth"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service is the narrow interface the handler depends on (consumer owns).
// Only GetUser is wired in Phase 1; SignUp/SignIn/RefreshToken/DeleteUser
// inherit Unimplemented from pb.UnimplementedAuthServiceServer until Phase 3
// migrates the monolith's user logic here.
type Service interface {
	GetUser(ctx context.Context, id uuid.UUID) (*User, error)
}

type Handler struct {
	pb.UnimplementedAuthServiceServer
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}

	u, err := h.service.GetUser(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}

	return toProto(u), nil
}

func toProto(u *User) *pb.User {
	return &pb.User{
		Id:        u.ID.String(),
		Email:     u.Email,
		Name:      u.Name,
		CreatedAt: timestamppb.New(u.CreatedAt),
		UpdatedAt: timestamppb.New(u.UpdatedAt),
	}
}

func mapError(err error) error {
	switch {
	case errors.Is(err, commonconstants.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, commonconstants.ErrDuplicateResource):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, commonconstants.ErrInvalidInput),
		errors.Is(err, commonconstants.ErrConstraintViolation):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, commonconstants.ErrUnauthorized):
		return status.Error(codes.Unauthenticated, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
