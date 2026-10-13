package plangw

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/common/errcode"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/apierr"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SERIALIZED plan member operations (FS-12ZMF §API surface, R15–R21).
//
// plan-service holds who is on a plan as user ids; the gateway holds the users
// table, so names and emails are resolved here. Every operation asks the access
// seam first: listing needs owner-or-member, adding and removing need the owner.

// Membership is one share as plan-service knows it: who, and since when.
type Membership struct {
	UserID uuid.UUID
	Since  time.Time
}

// PlanMembers is a plan's people as plan-service knows them. Members are in add
// order and never include the owner.
type PlanMembers struct {
	OwnerID       uuid.UUID
	PlanCreatedAt time.Time
	Members       []Membership
}

// MembersClient is plan-service's member half. *Client satisfies it.
type MembersClient interface {
	ListPlanMembers(ctx context.Context, planID, userID uuid.UUID) (*PlanMembers, error)
	AddPlanMember(ctx context.Context, planID, userID, memberID uuid.UUID) (*Membership, error)
	RemovePlanMember(ctx context.Context, planID, userID, memberID uuid.UUID) error
}

// Person is an account as the members list shows it. Name is already the
// display name when one is set, otherwise the account name (R15).
type Person struct {
	ID    uuid.UUID
	Name  string
	Email string
}

// People looks accounts up in the gateway's users table.
type People interface {
	// PersonByEmail matches the stored email exactly, as sign-in does (R17).
	// No account is commonconstants.ErrNotFound.
	PersonByEmail(ctx context.Context, email string) (Person, error)
	// PeopleByIDs answers the accounts that exist among ids, in any order. An id
	// with no account is simply absent (R16).
	PeopleByIDs(ctx context.Context, ids []uuid.UUID) ([]Person, error)
}

// PlanMemberResp is one person on a plan (R15).
type PlanMemberResp struct {
	ID       uuid.UUID `json:"id" format:"uuid" doc:"The person's user id"`
	Name     string    `json:"name" doc:"Display name if set, otherwise account name"`
	Email    string    `json:"email" doc:"The account's email"`
	Relation Relation  `json:"relation" enum:"owner,member" doc:"owner for the plan's owner, member for everyone it is shared with"`
	Since    time.Time `json:"since" doc:"Owner: when the plan was created. Member: when they were added."`
}

// AddPlanMemberReq is the body for POST /api/plans/{id}/members. Strict: an
// unknown member is a 422 (ADR-0005), so there is no role to send (D5).
type AddPlanMemberReq struct {
	Email string `json:"email" format:"email" minLength:"1" maxLength:"254" doc:"Email of an existing Fireplace account. Surrounding whitespace is trimmed, then matched exactly, as sign-in does." example:"friend@example.com"`
}

type PlanMembersOutput struct{ Body []PlanMemberResp }
type PlanMemberOutput struct{ Body PlanMemberResp }

type AddPlanMemberInput struct {
	ID   uuid.UUID `path:"id" doc:"Plan id"`
	Body AddPlanMemberReq
}

type RemovePlanMemberInput struct {
	ID     uuid.UUID `path:"id" doc:"Plan id"`
	UserID uuid.UUID `path:"userId" format:"uuid" doc:"The member's user id"`
}

func RegisterMemberOperations(api huma.API, c MembersClient, people People, access *Access,
	protect func(huma.Context, func(huma.Context)), secured []map[string][]string,
) {
	mw := huma.Middlewares{protect}

	authorize := func(ctx context.Context, op string, planID uuid.UUID, need Need) (uuid.UUID, error) {
		userID, ok := auth.UserIDFromCtx(ctx)
		if !ok {
			return uuid.Nil, apierr.ProblemFor(op, apierr.ErrNoIdentity())
		}
		if _, err := access.Require(ctx, planID, userID, need); err != nil {
			return uuid.Nil, apierr.ProblemFor(op, err)
		}
		return userID, nil
	}

	huma.Register(api, huma.Operation{
		OperationID: "listPlanMembers", Method: http.MethodGet, Path: "/api/plans/{id}/members",
		Middlewares: mw, Security: secured,
		Summary: "List a plan's people",
		Description: "The plan's owner first, then its members in the order they were added. " +
			"Open to the owner and to members. Answers 404 when the caller has no relation " +
			"to the plan, exactly as when no such plan exists.",
		Errors: []int{
			http.StatusUnauthorized, http.StatusNotFound,
			http.StatusUnprocessableEntity, http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, in *PlanIDInput) (*PlanMembersOutput, error) {
		const op = "list plan members"
		userID, err := authorize(ctx, op, in.ID, NeedMember)
		if err != nil {
			return nil, err
		}
		members, err := c.ListPlanMembers(ctx, in.ID, userID)
		if err != nil {
			return nil, apierr.ProblemFor(op, err)
		}
		ids := make([]uuid.UUID, 0, len(members.Members)+1)
		ids = append(ids, members.OwnerID)
		for _, m := range members.Members {
			ids = append(ids, m.UserID)
		}
		found, err := people.PeopleByIDs(ctx, ids)
		if err != nil {
			return nil, apierr.ProblemFor(op, err)
		}
		return &PlanMembersOutput{Body: peopleList(members, found)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "addPlanMember", Method: http.MethodPost, Path: "/api/plans/{id}/members",
		DefaultStatus: http.StatusCreated,
		Middlewares:   mw, Security: secured,
		Summary: "Add a member to a plan by email",
		Description: "Shares the plan with the account that uses this email. Owner only: a " +
			"member gets 403, and anyone else 404. Adding someone already on the plan answers " +
			"201 with their existing entry and original `since`. 400 `NO_ACCOUNT_FOR_EMAIL` " +
			"when no account uses the email; 400 `MEMBER_IS_OWNER` when it is the owner's own.",
		Errors: []int{
			http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden,
			http.StatusNotFound, http.StatusUnprocessableEntity, http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, in *AddPlanMemberInput) (*PlanMemberOutput, error) {
		const op = "add plan member"
		userID, err := authorize(ctx, op, in.ID, NeedOwner)
		if err != nil {
			return nil, err
		}
		person, err := people.PersonByEmail(ctx, strings.TrimSpace(in.Body.Email))
		if errors.Is(err, commonconstants.ErrNotFound) {
			return nil, apierr.Refusal(op, errcode.NoAccountForEmail, "no Fireplace account uses that email")
		}
		if err != nil {
			return nil, apierr.ProblemFor(op, err)
		}
		m, err := c.AddPlanMember(ctx, in.ID, userID, person.ID)
		// The ids are parsed uuids, so the InvalidArgument plan-service means
		// here is its owner-is-not-a-member rule. It is not the only one it
		// could send: a share insert that broke a column constraint (say, its
		// plan_id foreign key, if the plan is deleted between plan-service's
		// owner check and the insert) maps to ErrConstraintViolation, which is
		// InvalidArgument too, and would be mislabelled as this refusal. That
		// cannot fire today: commonhelpers' analyzeDBErr recognises only
		// *pgconn.PgError and plan-service drives lib/pq, so such an insert
		// surfaces as Internal. If plan-service moves to pgx, give self-share
		// its own code. The code is attached at this operation, never derived
		// from the message (FS-12ZMF §API surface).
		if status.Code(err) == codes.InvalidArgument {
			return nil, apierr.Refusal(op, errcode.MemberIsOwner, "that email is the plan owner's own")
		}
		if err != nil {
			return nil, apierr.ProblemFor(op, err)
		}
		return &PlanMemberOutput{Body: memberEntry(person, RelationMember, m.Since)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "removePlanMember", Method: http.MethodDelete,
		Path:          "/api/plans/{id}/members/{userId}",
		DefaultStatus: http.StatusNoContent,
		Middlewares:   mw, Security: secured,
		Summary: "Remove a member from a plan",
		Description: "Ends the member's access from their next request. Answers 204 with no " +
			"body. Owner only: a member gets 403, and anyone else 404. 404 too when the user " +
			"is not a member, including the owner's own id.",
		Errors: []int{
			http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound,
			http.StatusUnprocessableEntity, http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, in *RemovePlanMemberInput) (*DeleteOutput, error) {
		const op = "remove plan member"
		userID, err := authorize(ctx, op, in.ID, NeedOwner)
		if err != nil {
			return nil, err
		}
		if err := c.RemovePlanMember(ctx, in.ID, userID, in.UserID); err != nil {
			return nil, apierr.ProblemFor(op, err)
		}
		return &DeleteOutput{}, nil
	})
}

// peopleList is the owner first, then the members in plan-service's order. An
// id with no account behind it is left out rather than shown blank (R16).
func peopleList(members *PlanMembers, found []Person) []PlanMemberResp {
	byID := make(map[uuid.UUID]Person, len(found))
	for _, p := range found {
		byID[p.ID] = p
	}
	out := make([]PlanMemberResp, 0, len(members.Members)+1)
	if p, ok := byID[members.OwnerID]; ok {
		out = append(out, memberEntry(p, RelationOwner, members.PlanCreatedAt))
	}
	for _, m := range members.Members {
		if p, ok := byID[m.UserID]; ok {
			out = append(out, memberEntry(p, RelationMember, m.Since))
		}
	}
	return out
}

func memberEntry(p Person, rel Relation, since time.Time) PlanMemberResp {
	return PlanMemberResp{ID: p.ID, Name: p.Name, Email: p.Email, Relation: rel, Since: since}
}
