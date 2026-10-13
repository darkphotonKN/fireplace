package config

import (
	"context"

	authgw "github.com/darkphotonKN/fireplace/services/api-gateway/internal/gateway/auth"
	plangw "github.com/darkphotonKN/fireplace/services/api-gateway/internal/gateway/plan"
	"github.com/google/uuid"
)

// userLookups is the slice of the users repository the plan members list needs.
type userLookups interface {
	GetByEmail(ctx context.Context, email string) (*authgw.User, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*authgw.User, error)
}

// planPeople answers plangw.People from the gateway's users table (FS-12ZMF
// R15–R17). It lives here, at the composition root, so neither domain imports
// the other: plangw sees a Person, never a users row with its password hash.
type planPeople struct {
	users userLookups
}

// PersonByEmail is the exact match sign-in uses. No account stays ErrNotFound,
// which the add-member operation turns into NO_ACCOUNT_FOR_EMAIL.
func (p planPeople) PersonByEmail(ctx context.Context, email string) (plangw.Person, error) {
	u, err := p.users.GetByEmail(ctx, email)
	if err != nil {
		return plangw.Person{}, err
	}
	return personOf(u), nil
}

func (p planPeople) PeopleByIDs(ctx context.Context, ids []uuid.UUID) ([]plangw.Person, error) {
	users, err := p.users.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]plangw.Person, 0, len(users))
	for _, u := range users {
		out = append(out, personOf(u))
	}
	return out, nil
}

func personOf(u *authgw.User) plangw.Person {
	return plangw.Person{ID: u.ID, Name: u.ShownName(), Email: u.Email}
}
