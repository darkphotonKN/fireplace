package authgw

import (
	"context"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/auth"
	"github.com/darkphotonKN/fireplace/common/discovery"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/models"
	"github.com/google/uuid"
)

const targetService = "auth"

type Client struct {
	registry discovery.Registry
}

func NewClient(registry discovery.Registry) *Client {
	return &Client{registry: registry}
}

// dial picks an instance of auth-service from the registry and returns a
// gRPC client plus a close function. The connection is opened per call so the
// gateway doesn't pin to a stale instance; for hot paths, swap in a pool.
func (c *Client) dial(ctx context.Context) (pb.AuthServiceClient, func() error, error) {
	conn, err := discovery.ServiceConnection(ctx, targetService, c.registry)
	if err != nil {
		return nil, nil, err
	}
	return pb.NewAuthServiceClient(conn), conn.Close, nil
}

func (c *Client) SignUp(ctx context.Context, req *SignUpRequest) (*LoginResponse, error) {
	client, closer, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer closer()

	resp, err := client.SignUp(ctx, &pb.SignUpRequest{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		return nil, err
	}
	return authRespToHTTP(resp), nil
}

func (c *Client) SignIn(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	client, closer, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer closer()

	resp, err := client.SignIn(ctx, &pb.SignInRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return nil, err
	}
	return authRespToHTTP(resp), nil
}

func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	client, closer, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer closer()

	resp, err := client.RefreshToken(ctx, &pb.RefreshTokenRequest{RefreshToken: refreshToken})
	if err != nil {
		return nil, err
	}
	return authRespToHTTP(resp), nil
}

func (c *Client) GetById(ctx context.Context, id uuid.UUID) (*models.User, error) {
	client, closer, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer closer()
	u, err := client.GetUser(ctx, &pb.GetUserRequest{Id: id.String()})
	if err != nil {
		return nil, err
	}
	return userFromProto(u), nil
}

func (c *Client) GetProfile(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return c.GetById(ctx, id)
}

func (c *Client) UpdateProfile(ctx context.Context, id uuid.UUID, req UpdateProfileRequest) (*models.User, error) {
	client, closer, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer closer()
	u, err := client.UpdateProfile(ctx, &pb.UpdateProfileRequest{
		Id:          id.String(),
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Bio:         req.Bio,
	})
	if err != nil {
		return nil, err
	}
	return userFromProto(u), nil
}

func (c *Client) ListUsers(ctx context.Context) ([]*models.User, error) {
	client, closer, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer closer()
	resp, err := client.ListUsers(ctx, &pb.ListUsersRequest{})
	if err != nil {
		return nil, err
	}
	users := make([]*models.User, 0, len(resp.Users))
	for _, u := range resp.Users {
		users = append(users, userFromProto(u))
	}
	return users, nil
}

func userFromProto(u *pb.User) *models.User {
	id, _ := uuid.Parse(u.Id)
	return &models.User{
		BaseDBDateModel: models.BaseDBDateModel{
			ID:        id,
			CreatedAt: u.CreatedAt.AsTime(),
			UpdatedAt: u.UpdatedAt.AsTime(),
		},
		Email:       u.Email,
		Name:        u.Name,
		DisplayName: u.DisplayName,
		Bio:         u.Bio,
	}
}

func authRespToHTTP(r *pb.AuthResponse) *LoginResponse {
	return &LoginResponse{
		AccessToken:      r.AccessToken,
		RefreshToken:     r.RefreshToken,
		AccessExpiresIn:  r.AccessExpiresIn,
		RefreshExpiresIn: r.RefreshExpiresIn,
		UserInfo:         userFromProto(r.User),
	}
}
