package authgw

import (
	"context"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/auth"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// The auth domain runs IN-PROCESS in the gateway (ADR-0009 §1). auth-service
// was folded back in, so there is no gRPC hop left to make.
//
// Why this package still returns protobuf types: the serialized surface in
// typed.go / typed_users.go maps straight from *pb.User to its transport type,
// deleting the models.User hop (ADR-0003). Those handlers and their tests are
// the gateway's HTTP contract and did not change when the process boundary
// went away, so LocalClient keeps producing the same protos the gRPC client
// used to return. The protos are now plain in-process DTOs — a wart worth
// revisiting, but not while a fold is in flight.

// Service is the slice of the auth domain LocalClient needs. Declared at the
// consumer, so the concrete service is free to grow methods nobody here calls.
type Service interface {
	SignUp(ctx context.Context, in *SignUpInput) (*AuthTokens, error)
	SignIn(ctx context.Context, in *SignInInput) (*AuthTokens, error)
	Refresh(ctx context.Context, refreshToken string) (*AuthTokens, error)
	GetUser(ctx context.Context, id uuid.UUID) (*User, error)
	ListUsers(ctx context.Context) ([]*User, error)
	UpdateProfile(ctx context.Context, in *UpdateProfileParams) (*User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
}

// LocalClient satisfies ProfileClient and UsersClient against the in-process
// auth service. It is the seam the gRPC Client used to occupy.
type LocalClient struct {
	svc Service
}

func NewLocalClient(svc Service) *LocalClient {
	return &LocalClient{svc: svc}
}

func (c *LocalClient) GetUserProto(ctx context.Context, id uuid.UUID) (*pb.User, error) {
	u, err := c.svc.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	return userToProto(u), nil
}

// GetProfileProto is "my profile" — the same read as GetUserProto with the id
// taken from the token rather than the path.
func (c *LocalClient) GetProfileProto(ctx context.Context, id uuid.UUID) (*pb.User, error) {
	return c.GetUserProto(ctx, id)
}

func (c *LocalClient) ListUsersProto(ctx context.Context) ([]*pb.User, error) {
	users, err := c.svc.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	// Non-nil even when empty: the serialized body must marshal to [] and never
	// null, or a client iterating the result breaks (FS-0004 §Edge States).
	out := make([]*pb.User, 0, len(users))
	for _, u := range users {
		out = append(out, userToProto(u))
	}
	return out, nil
}

func (c *LocalClient) SignUpProto(ctx context.Context, req SignupRequest) (*pb.AuthResponse, error) {
	tokens, err := c.svc.SignUp(ctx, &SignUpInput{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		return nil, err
	}
	return tokensToProto(tokens), nil
}

func (c *LocalClient) SignInProto(ctx context.Context, req SigninRequest) (*pb.AuthResponse, error) {
	tokens, err := c.svc.SignIn(ctx, &SignInInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return nil, err
	}
	return tokensToProto(tokens), nil
}

func (c *LocalClient) UpdateProfileProto(ctx context.Context, id uuid.UUID, req UpdateProfileRequest) (*pb.User, error) {
	u, err := c.svc.UpdateProfile(ctx, &UpdateProfileParams{
		ID:          id,
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Bio:         req.Bio,
	})
	if err != nil {
		return nil, err
	}
	return userToProto(u), nil
}

func userToProto(u *User) *pb.User {
	return &pb.User{
		Id:          u.ID.String(),
		Email:       u.Email,
		Name:        u.Name,
		DisplayName: u.DisplayName,
		Bio:         u.Bio,
		CreatedAt:   timestamppb.New(u.CreatedAt),
		UpdatedAt:   timestamppb.New(u.UpdatedAt),
	}
}

func tokensToProto(t *AuthTokens) *pb.AuthResponse {
	return &pb.AuthResponse{
		User:             userToProto(t.User),
		AccessToken:      t.AccessToken,
		RefreshToken:     t.RefreshToken,
		AccessExpiresIn:  int64(t.AccessExpiresIn),
		RefreshExpiresIn: int64(t.RefreshExpiresIn),
	}
}
