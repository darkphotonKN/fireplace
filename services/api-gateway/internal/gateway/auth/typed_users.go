package authgw

import (
	"context"
	"net/http"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/auth"
	"github.com/danielgtaylor/huma/v2"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/apierr"
	"github.com/google/uuid"
)

// SERIALIZED users operations (FS-0004 §API surface, users).
//
// This is the first group carrying BOTH public and protected operations, which
// is why it went first: signup and signin run no middleware and declare no
// Security, while get-user and list-users declare both. Everything before this
// slice was uniformly protected, so the split was never actually exercised.

// UsersClient is the narrow seam these operations depend on. *Client satisfies
// it; a test double satisfies it without a gRPC connection.
type UsersClient interface {
	SignUpProto(ctx context.Context, req SignupRequest) (*pb.AuthResponse, error)
	SignInProto(ctx context.Context, req SigninRequest) (*pb.AuthResponse, error)
	GetUserProto(ctx context.Context, id uuid.UUID) (*pb.User, error)
	ListUsersProto(ctx context.Context) ([]*pb.User, error)
}

func RegisterUsersOperations(api huma.API, c UsersClient,
	protect func(huma.Context, func(huma.Context)), secured []map[string][]string,
) {
	mw := huma.Middlewares{protect}

	// --- PUBLIC ---------------------------------------------------------
	//
	// No Middlewares, no Security. Note that signin still documents 401: it
	// answers 401 for bad credentials while requiring no token to call. A
	// documented 401 and a required token are different claims, and any gate
	// that infers one from the other is wrong.

	huma.Register(api, huma.Operation{
		OperationID:   "signup",
		Method:        http.MethodPost,
		Path:          "/api/users/signup",
		DefaultStatus: http.StatusCreated,
		Summary:       "Create an account",
		Description: "Registers a new user. Returns 201 with an EMPTY body: " +
			"auth-service issues a token pair on signup, but this endpoint has " +
			"always discarded it and the retrofit preserves that. Sign in " +
			"afterwards to obtain tokens.",
		Errors: []int{
			http.StatusBadRequest, http.StatusConflict,
			http.StatusUnprocessableEntity, http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, in *SignupInput) (*SignupOutput, error) {
		if _, err := c.SignUpProto(ctx, in.Body); err != nil {
			return nil, apierr.ProblemFor("create user", err)
		}
		return &SignupOutput{}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "signin",
		Method:      http.MethodPost,
		Path:        "/api/users/signin",
		Summary:     "Sign in",
		Description: "Exchanges email and password for an access/refresh token pair. " +
			"Expiry fields are in NANOSECONDS, inherited from the monolith. " +
			"A 401 here means bad credentials — it does not mean a token was missing.",
		Errors: []int{
			http.StatusUnauthorized, http.StatusUnprocessableEntity,
			http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, in *SigninInput) (*SigninOutput, error) {
		resp, err := c.SignInProto(ctx, in.Body)
		if err != nil {
			return nil, apierr.ProblemFor("login", err)
		}
		return &SigninOutput{Body: authResponseFromProto(resp)}, nil
	})

	// --- PROTECTED ------------------------------------------------------

	huma.Register(api, huma.Operation{
		OperationID: "getUser",
		Method:      http.MethodGet,
		Path:        "/api/users/{id}",
		Middlewares: mw,
		Security:    secured,
		Summary:     "Get a user by id",
		Description: "Returns the public profile of any user. Identity is required " +
			"but not checked against the id — this is not the caller's own profile, " +
			"which is GET /api/users/profile.",
		Errors: []int{
			http.StatusUnauthorized, http.StatusNotFound,
			http.StatusUnprocessableEntity, http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, in *GetUserInput) (*GetUserOutput, error) {
		u, err := c.GetUserProto(ctx, in.ID)
		if err != nil {
			return nil, apierr.ProblemFor("get user by id", err)
		}
		return &GetUserOutput{Body: userResponseFromProto(u)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listUsers",
		Method:      http.MethodGet,
		Path:        "/api/users",
		Middlewares: mw,
		Security:    secured,
		Summary:     "List users",
		Description: "Returns every user. UNBOUNDED — there is no pagination today, " +
			"and this is transcribed as-is (ADR-0006 §2). Adding pagination later " +
			"is a breaking change to a shape that arguably should not have been " +
			"published unbounded.",
		Errors: []int{http.StatusUnauthorized, http.StatusServiceUnavailable},
	}, func(ctx context.Context, _ *struct{}) (*ListUsersOutput, error) {
		users, err := c.ListUsersProto(ctx)
		if err != nil {
			return nil, apierr.ProblemFor("list users", err)
		}
		// Explicitly empty, not nil — a nil slice marshals to null.
		out := make([]UserResponse, 0, len(users))
		for _, u := range users {
			out = append(out, userResponseFromProto(u))
		}
		return &ListUsersOutput{Body: out}, nil
	})
}
