package authgw

import (
	"context"
	"sync"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/auth"
	"github.com/darkphotonKN/fireplace/common/discovery"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/models"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

const targetService = "auth"

// Client wraps a long-lived gRPC connection to auth-service. The connection is
// established lazily on first use and reused across all calls — gRPC's HTTP/2
// layer multiplexes streams over a single connection, so concurrent RPCs are
// cheap. Opening a new conn per RPC (the previous shape) serialized badly
// under load.
type Client struct {
	registry discovery.Registry
	mu       sync.Mutex
	conn     *grpc.ClientConn
}

func NewClient(registry discovery.Registry) *Client {
	return &Client{registry: registry}
}

// connClient returns a (re)usable gRPC client. If the cached conn is missing
// or in Shutdown state, it's recreated.
func (c *Client) connClient(ctx context.Context) (pb.AuthServiceClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil && c.conn.GetState() != connectivity.Shutdown {
		return pb.NewAuthServiceClient(c.conn), nil
	}

	conn, err := discovery.ServiceConnection(ctx, targetService, c.registry)
	if err != nil {
		return nil, err
	}
	c.conn = conn
	return pb.NewAuthServiceClient(conn), nil
}

// Close releases the underlying gRPC connection. Wire from gateway shutdown.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}

func (c *Client) SignUp(ctx context.Context, req *SignUpRequest) (*LoginResponse, error) {
	client, err := c.connClient(ctx)
	if err != nil {
		return nil, err
	}
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
	client, err := c.connClient(ctx)
	if err != nil {
		return nil, err
	}
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
	client, err := c.connClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.RefreshToken(ctx, &pb.RefreshTokenRequest{RefreshToken: refreshToken})
	if err != nil {
		return nil, err
	}
	return authRespToHTTP(resp), nil
}

func (c *Client) GetById(ctx context.Context, id uuid.UUID) (*models.User, error) {
	client, err := c.connClient(ctx)
	if err != nil {
		return nil, err
	}
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
	client, err := c.connClient(ctx)
	if err != nil {
		return nil, err
	}
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
	client, err := c.connClient(ctx)
	if err != nil {
		return nil, err
	}
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
