package plangw

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeMembers plays plan-service's member RPCs and records what reached it.
type fakeMembers struct {
	calls   []string
	members PlanMembers
	added   map[uuid.UUID]time.Time // re-adds return the original since
	addErr  error
	removed []uuid.UUID
	missing bool // RemovePlanMember answers NotFound
}

func (f *fakeMembers) ListPlanMembers(ctx context.Context, planID, userID uuid.UUID) (*PlanMembers, error) {
	f.calls = append(f.calls, "list")
	m := f.members
	return &m, nil
}

func (f *fakeMembers) AddPlanMember(ctx context.Context, planID, userID, memberID uuid.UUID) (*Membership, error) {
	f.calls = append(f.calls, "add")
	if f.addErr != nil {
		return nil, f.addErr
	}
	if f.added == nil {
		f.added = map[uuid.UUID]time.Time{}
	}
	since, ok := f.added[memberID]
	if !ok {
		since = time.Date(2026, 10, 1, 9, len(f.added), 0, 0, time.UTC)
		f.added[memberID] = since
	}
	return &Membership{UserID: memberID, Since: since}, nil
}

func (f *fakeMembers) RemovePlanMember(ctx context.Context, planID, userID, memberID uuid.UUID) error {
	f.calls = append(f.calls, "remove")
	if f.missing {
		return status.Error(codes.NotFound, "plan: remove member: resource not found")
	}
	f.removed = append(f.removed, memberID)
	return nil
}

// fakePeople plays the gateway's users table.
type fakePeople struct {
	people  []Person
	byEmail []string // every email it was asked for
	err     error
}

func (f *fakePeople) PersonByEmail(ctx context.Context, email string) (Person, error) {
	f.byEmail = append(f.byEmail, email)
	if f.err != nil {
		return Person{}, f.err
	}
	for _, p := range f.people {
		if p.Email == email {
			return p, nil
		}
	}
	return Person{}, commonconstants.ErrNotFound
}

func (f *fakePeople) PeopleByIDs(ctx context.Context, ids []uuid.UUID) ([]Person, error) {
	out := []Person{}
	for _, id := range ids {
		for _, p := range f.people {
			if p.ID == id {
				out = append(out, p)
			}
		}
	}
	return out, nil
}

func newMembersAPI(t *testing.T, members MembersClient, people People, rel Relation, caller uuid.UUID) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t, huma.DefaultConfig("test", "1"))
	RegisterMemberOperations(api, members, people, NewAccess(&fakeRelations{rel: rel}),
		func(ctx huma.Context, next func(huma.Context)) {
			next(huma.WithContext(ctx, auth.WithUserID(ctx.Context(), caller)))
		}, nil)
	return api.(humatest.TestAPI)
}

var (
	ownerP  = Person{ID: uuid.New(), Name: "Kranti", Email: "k@x.com"}
	memberP = Person{ID: uuid.New(), Name: "Nick", Email: "m@x.com"}
	otherP  = Person{ID: uuid.New(), Name: "Ana", Email: "a@x.com"}
)

func membersURL() string { return "/api/plans/" + uuid.NewString() + "/members" }

func memberOf(t *testing.T, body []byte) PlanMemberResp {
	t.Helper()
	var m PlanMemberResp
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("body is not a member: %v: %s", err, body)
	}
	return m
}

// R17: an email with no account behind it is a domain refusal, 400
// NO_ACCOUNT_FOR_EMAIL, and plan-service is never asked to add anyone.
func TestAddPlanMember_UnknownEmail_Is400NoAccountForEmail(t *testing.T) {
	members := &fakeMembers{}
	api := newMembersAPI(t, members, &fakePeople{people: []Person{ownerP}}, RelationOwner, ownerP.ID)

	resp := api.Post(membersURL(), map[string]any{"email": "nobody@x.com"})

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", resp.Code, resp.Body.String())
	}
	if got := problemCode(t, resp); got != "NO_ACCOUNT_FOR_EMAIL" {
		t.Errorf("code = %q, want NO_ACCOUNT_FOR_EMAIL", got)
	}
	if ct := resp.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}
	if len(members.calls) != 0 {
		t.Errorf("plan-service was called: %v", members.calls)
	}
}

// R17: the owner adds an existing account and gets the new entry.
func TestAddPlanMember_ExistingAccount_Is201WithTheEntry(t *testing.T) {
	members := &fakeMembers{}
	api := newMembersAPI(t, members, &fakePeople{people: []Person{ownerP, memberP}}, RelationOwner, ownerP.ID)

	resp := api.Post(membersURL(), map[string]any{"email": "m@x.com"})

	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", resp.Code, resp.Body.String())
	}
	got := memberOf(t, resp.Body.Bytes())
	want := PlanMemberResp{ID: memberP.ID, Name: "Nick", Email: "m@x.com", Relation: RelationMember, Since: members.added[memberP.ID]}
	if got.ID != want.ID || got.Name != want.Name || got.Email != want.Email || got.Relation != want.Relation || !got.Since.Equal(want.Since) {
		t.Errorf("entry = %+v, want %+v", got, want)
	}
}

// R17: a re-add answers 201 with the original since.
func TestAddPlanMember_Again_KeepsTheOriginalSince(t *testing.T) {
	members := &fakeMembers{}
	api := newMembersAPI(t, members, &fakePeople{people: []Person{ownerP, memberP, otherP}}, RelationOwner, ownerP.ID)
	url := membersURL()

	first := memberOf(t, api.Post(url, map[string]any{"email": "m@x.com"}).Body.Bytes())
	api.Post(url, map[string]any{"email": "a@x.com"})
	resp := api.Post(url, map[string]any{"email": "m@x.com"})

	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", resp.Code, resp.Body.String())
	}
	if again := memberOf(t, resp.Body.Bytes()); !again.Since.Equal(first.Since) {
		t.Errorf("since = %s, want the original %s", again.Since, first.Since)
	}
}

// R17: surrounding whitespace is trimmed before the exact-match lookup.
func TestAddPlanMember_SurroundingWhitespace_IsTrimmed(t *testing.T) {
	people := &fakePeople{people: []Person{ownerP, memberP}}
	api := newMembersAPI(t, &fakeMembers{}, people, RelationOwner, ownerP.ID)

	resp := api.Post(membersURL(), map[string]any{"email": "  m@x.com  "})

	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", resp.Code, resp.Body.String())
	}
	if len(people.byEmail) != 1 || people.byEmail[0] != "m@x.com" {
		t.Errorf("looked up %q, want exactly \"m@x.com\"", people.byEmail)
	}
}

// R17: plan-service refuses the owner as a member with InvalidArgument, and
// this operation attaches MEMBER_IS_OWNER itself. The refusal message contains
// "owner", never parsed: even a message the shared mapper would misread as
// PROFILE_NAME_EMPTY comes out right.
func TestAddPlanMember_OwnersOwnEmail_Is400MemberIsOwner(t *testing.T) {
	for _, msg := range []string{
		"plan: add member: invalid input: the plan's owner cannot be its member",
		"display name of the owner", // would read as PROFILE_NAME_EMPTY through CodeFor
	} {
		t.Run(msg, func(t *testing.T) {
			members := &fakeMembers{addErr: status.Error(codes.InvalidArgument, msg)}
			api := newMembersAPI(t, members, &fakePeople{people: []Person{ownerP}}, RelationOwner, ownerP.ID)

			resp := api.Post(membersURL(), map[string]any{"email": "k@x.com"})

			if resp.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400: %s", resp.Code, resp.Body.String())
			}
			if got := problemCode(t, resp); got != "MEMBER_IS_OWNER" {
				t.Errorf("code = %q, want MEMBER_IS_OWNER", got)
			}
		})
	}
}

// R18: malformed bodies are the boundary's 422, before any lookup.
func TestAddPlanMember_MalformedBody_Is422(t *testing.T) {
	bodies := map[string]any{
		"not an email":  map[string]any{"email": "nope"},
		"empty email":   map[string]any{"email": ""},
		"no email":      map[string]any{},
		"unknown field": map[string]any{"email": "m@x.com", "role": "view"},
		"display form":  map[string]any{"email": "Nick <m@x.com>"},
		"too long":      map[string]any{"email": strings.Repeat("a", 250) + "@x.com"},
	}
	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			people := &fakePeople{people: []Person{ownerP, memberP}}
			members := &fakeMembers{}
			api := newMembersAPI(t, members, people, RelationOwner, ownerP.ID)

			resp := api.Post(membersURL(), body)

			if resp.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422: %s", resp.Code, resp.Body.String())
			}
			if len(people.byEmail) != 0 || len(members.calls) != 0 {
				t.Errorf("a malformed body reached lookup %v or plan-service %v", people.byEmail, members.calls)
			}
		})
	}
}

// R6: a member may list but not add or remove; R5: a stranger gets 404 on all
// three, and nothing past the seam runs.
func TestMemberRoutes_AccessByRelation(t *testing.T) {
	routes := map[string]func(api humatest.TestAPI) *httptest.ResponseRecorder{
		"GET": func(api humatest.TestAPI) *httptest.ResponseRecorder { return api.Get(membersURL()) },
		"POST": func(api humatest.TestAPI) *httptest.ResponseRecorder {
			return api.Post(membersURL(), map[string]any{"email": "a@x.com"})
		},
		"DELETE": func(api humatest.TestAPI) *httptest.ResponseRecorder {
			return api.Delete(membersURL() + "/" + otherP.ID.String())
		},
	}
	want := map[Relation]map[string]int{
		RelationMember: {"POST": http.StatusForbidden, "DELETE": http.StatusForbidden},
		RelationNone:   {"GET": http.StatusNotFound, "POST": http.StatusNotFound, "DELETE": http.StatusNotFound},
	}
	for rel, byRoute := range want {
		for route, code := range byRoute {
			t.Run(string(rel)+" "+route, func(t *testing.T) {
				members := &fakeMembers{}
				people := &fakePeople{people: []Person{ownerP, memberP, otherP}}
				api := newMembersAPI(t, members, people, rel, memberP.ID)

				resp := routes[route](api)

				if resp.Code != code {
					t.Fatalf("status = %d, want %d: %s", resp.Code, code, resp.Body.String())
				}
				if len(members.calls) != 0 || len(people.byEmail) != 0 {
					t.Errorf("refused request ran: plan-service %v, lookup %v", members.calls, people.byEmail)
				}
			})
		}
	}
}

// R15, R16: owner first, then members in plan-service's add order, each with
// name, email, relation and since; an id with no account is left out.
func TestListPlanMembers_OwnerFirstThenMembersSkippingTheGone(t *testing.T) {
	created := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	gone := uuid.New()
	members := &fakeMembers{members: PlanMembers{
		OwnerID:       ownerP.ID,
		PlanCreatedAt: created,
		Members: []Membership{
			{UserID: otherP.ID, Since: created.Add(time.Hour)},
			{UserID: gone, Since: created.Add(2 * time.Hour)},
			{UserID: memberP.ID, Since: created.Add(3 * time.Hour)},
		},
	}}
	// The lookup answers in its own order; the list must not depend on it.
	people := &fakePeople{people: []Person{memberP, otherP, ownerP}}

	for _, rel := range []Relation{RelationOwner, RelationMember} {
		t.Run(string(rel), func(t *testing.T) {
			api := newMembersAPI(t, members, people, rel, memberP.ID)

			resp := api.Get(membersURL())

			if resp.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", resp.Code, resp.Body.String())
			}
			var got []PlanMemberResp
			if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
				t.Fatalf("body: %v", err)
			}
			want := []PlanMemberResp{
				{ID: ownerP.ID, Name: "Kranti", Email: "k@x.com", Relation: RelationOwner, Since: created},
				{ID: otherP.ID, Name: "Ana", Email: "a@x.com", Relation: RelationMember, Since: created.Add(time.Hour)},
				{ID: memberP.ID, Name: "Nick", Email: "m@x.com", Relation: RelationMember, Since: created.Add(3 * time.Hour)},
			}
			if len(got) != len(want) {
				t.Fatalf("got %d entries, want %d: %+v", len(got), len(want), got)
			}
			for i := range want {
				g, w := got[i], want[i]
				if g.ID != w.ID || g.Name != w.Name || g.Email != w.Email || g.Relation != w.Relation || !g.Since.Equal(w.Since) {
					t.Errorf("[%d] = %+v, want %+v", i, g, w)
				}
			}
		})
	}
}

// NO_ACCOUNT_FOR_EMAIL means exactly "the lookup found no account". A lookup
// that fails, or plan-service's NotFound for the plan, keeps its own mapping.
func TestAddPlanMember_OtherFailures_AreNotNoAccountForEmail(t *testing.T) {
	cases := map[string]struct {
		people  *fakePeople
		members *fakeMembers
		status  int
		code    string
	}{
		"lookup fails": {
			&fakePeople{err: commonconstants.ErrTransient}, &fakeMembers{},
			http.StatusInternalServerError, "INTERNAL_ERROR",
		},
		"plan-service says no plan": {
			&fakePeople{people: []Person{memberP}},
			&fakeMembers{addErr: status.Error(codes.NotFound, "plan: add member: resource not found")},
			http.StatusNotFound, "NOT_FOUND",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			api := newMembersAPI(t, tc.members, tc.people, RelationOwner, ownerP.ID)

			resp := api.Post(membersURL(), map[string]any{"email": "m@x.com"})

			if resp.Code != tc.status {
				t.Fatalf("status = %d, want %d: %s", resp.Code, tc.status, resp.Body.String())
			}
			if got := problemCode(t, resp); got != tc.code {
				t.Errorf("code = %q, want %q", got, tc.code)
			}
		})
	}
}

// R19: the owner removes a member → 204; plan-service's NotFound (not a
// member, already removed, the owner's own id) → 404.
func TestRemovePlanMember_ByOwner(t *testing.T) {
	members := &fakeMembers{}
	api := newMembersAPI(t, members, &fakePeople{}, RelationOwner, ownerP.ID)

	resp := api.Delete(membersURL() + "/" + memberP.ID.String())

	if resp.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: %s", resp.Code, resp.Body.String())
	}
	if len(members.removed) != 1 || members.removed[0] != memberP.ID {
		t.Errorf("removed %v, want [%s]", members.removed, memberP.ID)
	}

	members.missing = true
	resp = api.Delete(membersURL() + "/" + ownerP.ID.String())
	if resp.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404: %s", resp.Code, resp.Body.String())
	}
	if got := problemCode(t, resp); got != "NOT_FOUND" {
		t.Errorf("code = %q, want NOT_FOUND", got)
	}
}
