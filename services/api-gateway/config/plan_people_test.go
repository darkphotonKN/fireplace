package config

import (
	"context"
	"errors"
	"testing"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	authgw "github.com/darkphotonKN/fireplace/services/api-gateway/internal/gateway/auth"
	"github.com/google/uuid"
)

type fakeUsers struct{ users []*authgw.User }

func (f fakeUsers) GetByEmail(ctx context.Context, email string) (*authgw.User, error) {
	for _, u := range f.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, commonconstants.ErrNotFound
}

func (f fakeUsers) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*authgw.User, error) {
	return f.users, nil
}

// FS-12ZMF R15, R17: a person carries the shown name, and no account stays
// ErrNotFound so the operation can answer NO_ACCOUNT_FOR_EMAIL.
func TestPlanPeople(t *testing.T) {
	shown := "Nicky"
	nick := &authgw.User{ID: uuid.New(), Name: "Nick", Email: "m@x.com", DisplayName: &shown}
	ana := &authgw.User{ID: uuid.New(), Name: "Ana", Email: "a@x.com"}
	people := planPeople{users: fakeUsers{users: []*authgw.User{nick, ana}}}

	p, err := people.PersonByEmail(context.Background(), "m@x.com")
	if err != nil || p.ID != nick.ID || p.Name != "Nicky" || p.Email != "m@x.com" {
		t.Errorf("PersonByEmail = %+v, %v", p, err)
	}
	if _, err := people.PersonByEmail(context.Background(), "nobody@x.com"); !errors.Is(err, commonconstants.ErrNotFound) {
		t.Errorf("unknown email: err = %v, want ErrNotFound", err)
	}
	all, err := people.PeopleByIDs(context.Background(), []uuid.UUID{nick.ID, ana.ID})
	if err != nil || len(all) != 2 || all[1].Name != "Ana" {
		t.Errorf("PeopleByIDs = %+v, %v", all, err)
	}
}
