package plangw

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/darkphotonKN/fireplace/common/errcode"
	commongrpc "github.com/darkphotonKN/fireplace/common/grpcerror"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeComments plays plan-service's comment RPCs and records what reached it.
type fakeComments struct {
	comments []NoteComment
	err      error

	calls                     int
	gotPlan, gotItem, gotUser uuid.UUID
	gotBody                   string

	// stored is what GetNoteComment answers; writes records the write RPCs
	// that reached plan-service.
	stored *NoteComment
	getErr error
	writes []string
}

func (f *fakeComments) GetNoteComment(_ context.Context, planID, itemID, commentID, userID uuid.UUID) (*NoteComment, error) {
	f.calls++
	if f.getErr != nil {
		return nil, f.getErr
	}
	c := *f.stored
	return &c, nil
}

func (f *fakeComments) UpdateNoteComment(_ context.Context, planID, itemID, commentID, userID uuid.UUID, body string) (*NoteComment, error) {
	f.calls++
	f.writes = append(f.writes, "update")
	f.gotUser, f.gotBody = userID, body
	if f.err != nil {
		return nil, f.err
	}
	c := *f.stored
	c.Body = body
	now := time.Now().UTC()
	c.EditedAt = &now
	return &c, nil
}

func (f *fakeComments) DeleteNoteComment(_ context.Context, planID, itemID, commentID, userID uuid.UUID) error {
	f.calls++
	f.writes = append(f.writes, "delete")
	f.gotUser = userID
	return f.err
}

func (f *fakeComments) ListNoteComments(_ context.Context, planID, itemID, userID uuid.UUID) ([]NoteComment, error) {
	f.calls++
	f.gotPlan, f.gotItem, f.gotUser = planID, itemID, userID
	return f.comments, f.err
}

func (f *fakeComments) CreateNoteComment(_ context.Context, planID, itemID, userID uuid.UUID, body string) (*NoteComment, error) {
	f.calls++
	f.gotPlan, f.gotItem, f.gotUser, f.gotBody = planID, itemID, userID, body
	if f.err != nil {
		return nil, f.err
	}
	return &NoteComment{ID: uuid.New(), ItemID: itemID, AuthorID: userID, Body: body, CreatedAt: time.Now().UTC()}, nil
}

// countingPeople counts batch lookups over fakePeople.
type countingPeople struct {
	fakePeople
	batches int
	err     error
}

func (c *countingPeople) PeopleByIDs(ctx context.Context, ids []uuid.UUID) ([]Person, error) {
	c.batches++
	if c.err != nil {
		return nil, c.err
	}
	return c.fakePeople.PeopleByIDs(ctx, ids)
}

func newCommentsAPI(t *testing.T, c NoteCommentsClient, people People, rel Relation, caller uuid.UUID) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t, huma.DefaultConfig("test", "1"))
	RegisterNoteCommentOperations(api, c, people, NewAccess(&fakeRelations{rel: rel}),
		func(ctx huma.Context, next func(huma.Context)) {
			next(huma.WithContext(ctx, auth.WithUserID(ctx.Context(), caller)))
		}, nil)
	return api.(humatest.TestAPI)
}

func commentsURL(planID, itemID uuid.UUID) string {
	return "/api/plans/" + planID.String() + "/checklists/" + itemID.String() + "/comments"
}

// R6, R12, R13, R15: a member adds a comment; it comes back 201 as theirs,
// under their name, unedited, theirs to edit and delete.
func TestCreateNoteComment_MemberAdds_201AsTheCaller(t *testing.T) {
	c := &fakeComments{}
	planID, itemID := uuid.New(), uuid.New()
	api := newCommentsAPI(t, c, &countingPeople{fakePeople: fakePeople{people: []Person{memberP}}}, RelationMember, memberP.ID)

	resp := api.Post(commentsURL(planID, itemID), map[string]any{"body": "Looks good"})

	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", resp.Code, resp.Body.String())
	}
	if c.gotUser != memberP.ID || c.gotPlan != planID || c.gotItem != itemID || c.gotBody != "Looks good" {
		t.Fatalf("plan-service got user %s plan %s item %s body %q", c.gotUser, c.gotPlan, c.gotItem, c.gotBody)
	}
	var raw map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	if raw["authorId"] != memberP.ID.String() || raw["authorName"] != "Nick" || raw["editedAt"] != nil ||
		raw["canEdit"] != true || raw["canDelete"] != true {
		t.Fatalf("comment = %s", resp.Body.String())
	}
	if _, ok := raw["editedAt"]; !ok {
		t.Fatalf("editedAt is absent, want null: %s", resp.Body.String())
	}
}

// R14: a stranger is 404 on both routes and never reaches plan-service.
func TestNoteComments_Stranger_Is404(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			c := &fakeComments{}
			api := newCommentsAPI(t, c, &fakePeople{}, RelationNone, uuid.New())
			url := commentsURL(uuid.New(), uuid.New())

			resp := api.Get(url)
			if method == http.MethodPost {
				resp = api.Post(url, map[string]any{"body": "hi"})
			}

			if resp.Code != http.StatusNotFound || c.calls != 0 {
				t.Fatalf("status = %d, calls = %d; want 404 and none", resp.Code, c.calls)
			}
		})
	}
}

// R6, ADR-0005: the body is strict; an author field, a missing body or a
// non-string one is 422 at the boundary.
func TestCreateNoteComment_ShapeFailuresAre422(t *testing.T) {
	for name, body := range map[string]map[string]any{
		"authorId sent":   {"body": "hi", "authorId": uuid.NewString()},
		"missing body":    {},
		"non-string body": {"body": 42},
		"unknown member":  {"body": "hi", "pinned": true},
	} {
		t.Run(name, func(t *testing.T) {
			c := &fakeComments{}
			resp := newCommentsAPI(t, c, &fakePeople{}, RelationOwner, uuid.New()).Post(commentsURL(uuid.New(), uuid.New()), body)
			if resp.Code != http.StatusUnprocessableEntity || c.calls != 0 {
				t.Fatalf("status = %d, calls = %d; want 422 and none: %s", resp.Code, c.calls, resp.Body.String())
			}
		})
	}
}

// R3, R9, R2: plan-service's refusals leave with their status and code.
func TestCreateNoteComment_DomainRefusals(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"empty or over-length body", status.Error(codes.InvalidArgument, "invalid input: comment body must not be empty"), http.StatusBadRequest, string(errcode.ValidationFailed)},
		{"a task", commongrpc.WithCode(codes.FailedPrecondition, errcode.ItemNotANote, "task"), http.StatusBadRequest, string(errcode.ItemNotANote)},
		{"item not in plan", status.Error(codes.NotFound, "no item"), http.StatusNotFound, string(errcode.NotFound)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := newCommentsAPI(t, &fakeComments{err: tc.err}, &fakePeople{}, RelationOwner, uuid.New()).
				Post(commentsURL(uuid.New(), uuid.New()), map[string]any{"body": "x"})
			if resp.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d: %s", resp.Code, tc.wantStatus, resp.Body.String())
			}
			if got := problemCode(t, resp); got != tc.wantCode {
				t.Fatalf("code = %q, want %q", got, tc.wantCode)
			}
			if ct := resp.Header().Get("Content-Type"); ct != "application/problem+json" {
				t.Fatalf("Content-Type = %q, want application/problem+json", ct)
			}
		})
	}
}

// R3: listing a task's comments is 400 ITEM_NOT_A_NOTE.
func TestListNoteComments_ATask_Is400ItemNotANote(t *testing.T) {
	c := &fakeComments{err: commongrpc.WithCode(codes.FailedPrecondition, errcode.ItemNotANote, "task")}
	resp := newCommentsAPI(t, c, &fakePeople{}, RelationMember, uuid.New()).Get(commentsURL(uuid.New(), uuid.New()))
	if resp.Code != http.StatusBadRequest || problemCode(t, resp) != string(errcode.ItemNotANote) {
		t.Fatalf("status = %d: %s", resp.Code, resp.Body.String())
	}
}

func listedComments(t *testing.T, body []byte) []NoteCommentResp {
	t.Helper()
	var out []NoteCommentResp
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("body is not a comment list: %v: %s", err, body)
	}
	return out
}

// R11-R13, R18: the list keeps plan-service's order, names authors from one
// batch, answers null for a missing account, and sets the flags per caller.
func TestListNoteComments_NamesAndFlags(t *testing.T) {
	gone := uuid.New()
	at := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	stored := []NoteComment{
		{ID: uuid.New(), AuthorID: ownerP.ID, Body: "owner's", CreatedAt: at},
		{ID: uuid.New(), AuthorID: memberP.ID, Body: "member's", CreatedAt: at.Add(time.Minute)},
		{ID: uuid.New(), AuthorID: gone, Body: "gone's", CreatedAt: at.Add(2 * time.Minute)},
		{ID: uuid.New(), AuthorID: memberP.ID, Body: "member's again", CreatedAt: at.Add(3 * time.Minute)},
	}
	cases := []struct {
		name      string
		rel       Relation
		caller    uuid.UUID
		canEdit   []bool
		canDelete []bool
	}{
		{"as the owner", RelationOwner, ownerP.ID, []bool{true, false, false, false}, []bool{true, true, true, true}},
		{"as a member", RelationMember, memberP.ID, []bool{false, true, false, true}, []bool{false, true, false, true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			people := &countingPeople{fakePeople: fakePeople{people: []Person{ownerP, memberP}}}
			resp := newCommentsAPI(t, &fakeComments{comments: stored}, people, tc.rel, tc.caller).Get(commentsURL(uuid.New(), uuid.New()))

			if resp.Code != http.StatusOK {
				t.Fatalf("status = %d: %s", resp.Code, resp.Body.String())
			}
			got := listedComments(t, resp.Body.Bytes())
			if len(got) != 4 || people.batches != 1 {
				t.Fatalf("got %d comments from %d lookups, want 4 from 1", len(got), people.batches)
			}
			wantNames := []*string{&ownerP.Name, &memberP.Name, nil, &memberP.Name}
			for i, c := range got {
				if c.ID != stored[i].ID || c.Body != stored[i].Body {
					t.Fatalf("comment %d = %+v, want plan-service's order", i, c)
				}
				if (c.AuthorName == nil) != (wantNames[i] == nil) || (c.AuthorName != nil && *c.AuthorName != *wantNames[i]) {
					t.Errorf("comment %d authorName = %v, want %v", i, c.AuthorName, wantNames[i])
				}
				if c.CanEdit != tc.canEdit[i] || c.CanDelete != tc.canDelete[i] {
					t.Errorf("comment %d canEdit/canDelete = %v/%v, want %v/%v", i, c.CanEdit, c.CanDelete, tc.canEdit[i], tc.canDelete[i])
				}
			}
		})
	}
}

// R11: a note with none answers [], never null.
func TestListNoteComments_NoneIsAnEmptyArray(t *testing.T) {
	resp := newCommentsAPI(t, &fakeComments{}, &countingPeople{}, RelationMember, uuid.New()).Get(commentsURL(uuid.New(), uuid.New()))
	if resp.Code != http.StatusOK || resp.Body.String() != "[]\n" && resp.Body.String() != "[]" {
		t.Fatalf("status = %d, body %q; want 200 []", resp.Code, resp.Body.String())
	}
}

// Edge states: a users lookup that fails fails the list with its status; no
// partial names.
func TestListNoteComments_LookupFailureFailsTheList(t *testing.T) {
	people := &countingPeople{err: status.Error(codes.Unavailable, "users down")}
	c := &fakeComments{comments: []NoteComment{{ID: uuid.New(), AuthorID: uuid.New(), Body: "b"}}}
	resp := newCommentsAPI(t, c, people, RelationOwner, uuid.New()).Get(commentsURL(uuid.New(), uuid.New()))
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503: %s", resp.Code, resp.Body.String())
	}
}

// R16-R18: one policy decides the flags and enforcement.
func TestCommentRights(t *testing.T) {
	me, someone := uuid.New(), uuid.New()
	cases := []struct {
		name               string
		rel                Relation
		author             uuid.UUID
		canEdit, canDelete bool
	}{
		{"owner, own", RelationOwner, me, true, true},
		{"owner, another's", RelationOwner, someone, false, true},
		{"member, own", RelationMember, me, true, true},
		{"member, another's", RelationMember, someone, false, false},
		{"no relation, own", RelationNone, me, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := commentRights(tc.rel, me, tc.author)
			if got.Edit != tc.canEdit || got.Delete != tc.canDelete {
				t.Fatalf("commentRights = %v/%v, want %v/%v", got.Edit, got.Delete, tc.canEdit, tc.canDelete)
			}
		})
	}
}

func commentURL(commentID uuid.UUID) string {
	return commentsURL(uuid.New(), uuid.New()) + "/" + commentID.String()
}

// R16-R18: (relation, whose comment) x (edit, delete) through the one policy.
// Refused writes answer 403 FORBIDDEN and never reach plan-service.
func TestNoteCommentWrites_RightsTable(t *testing.T) {
	gone := uuid.New()
	cases := []struct {
		name              string
		rel               Relation
		caller, author    uuid.UUID
		editCode, delCode int
	}{
		{"owner, own", RelationOwner, ownerP.ID, ownerP.ID, http.StatusOK, http.StatusNoContent},
		{"owner, a member's", RelationOwner, ownerP.ID, memberP.ID, http.StatusForbidden, http.StatusNoContent},
		{"owner, a deleted account's", RelationOwner, ownerP.ID, gone, http.StatusForbidden, http.StatusNoContent},
		{"member, own", RelationMember, memberP.ID, memberP.ID, http.StatusOK, http.StatusNoContent},
		{"member, another member's", RelationMember, memberP.ID, otherP.ID, http.StatusForbidden, http.StatusForbidden},
		{"member, the owner's", RelationMember, memberP.ID, ownerP.ID, http.StatusForbidden, http.StatusForbidden},
		{"member, a deleted account's", RelationMember, memberP.ID, gone, http.StatusForbidden, http.StatusForbidden},
	}
	for _, tc := range cases {
		people := &countingPeople{fakePeople: fakePeople{people: []Person{ownerP, memberP, otherP}}}
		stored := &NoteComment{ID: uuid.New(), AuthorID: tc.author, Body: "before", CreatedAt: time.Now().UTC()}

		t.Run(tc.name+"/edit", func(t *testing.T) {
			c := &fakeComments{stored: stored}
			resp := newCommentsAPI(t, c, people, tc.rel, tc.caller).Patch(commentURL(stored.ID), map[string]any{"body": "after"})
			if resp.Code != tc.editCode {
				t.Fatalf("status = %d, want %d: %s", resp.Code, tc.editCode, resp.Body.String())
			}
			if wrote := len(c.writes) > 0; wrote != (tc.editCode == http.StatusOK) {
				t.Fatalf("writes = %v", c.writes)
			}
			if tc.editCode == http.StatusForbidden {
				if got := problemCode(t, resp); got != string(errcode.Forbidden) {
					t.Fatalf("code = %q, want FORBIDDEN", got)
				}
				if ct := resp.Header().Get("Content-Type"); ct != "application/problem+json" {
					t.Fatalf("Content-Type = %q, want application/problem+json", ct)
				}
			}
		})
		t.Run(tc.name+"/delete", func(t *testing.T) {
			c := &fakeComments{stored: stored}
			resp := newCommentsAPI(t, c, people, tc.rel, tc.caller).Delete(commentURL(stored.ID))
			if resp.Code != tc.delCode {
				t.Fatalf("status = %d, want %d: %s", resp.Code, tc.delCode, resp.Body.String())
			}
			if wrote := len(c.writes) > 0; wrote != (tc.delCode == http.StatusNoContent) {
				t.Fatalf("writes = %v", c.writes)
			}
		})
	}
}

// R12, R16: the author's edit answers the comment as a list would, by the
// caller, with the trimmed body plan-service stored.
func TestUpdateNoteComment_AnswersTheEditedComment(t *testing.T) {
	stored := &NoteComment{ID: uuid.New(), AuthorID: memberP.ID, Body: "before", CreatedAt: time.Now().UTC()}
	c := &fakeComments{stored: stored}
	people := &countingPeople{fakePeople: fakePeople{people: []Person{memberP}}}

	resp := newCommentsAPI(t, c, people, RelationMember, memberP.ID).Patch(commentURL(stored.ID), map[string]any{"body": "after"})

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", resp.Code, resp.Body.String())
	}
	var got NoteCommentResp
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Body != "after" || got.EditedAt == nil || got.AuthorName == nil || *got.AuthorName != "Nick" ||
		!got.CanEdit || !got.CanDelete || c.gotUser != memberP.ID || c.gotBody != "after" {
		t.Fatalf("comment = %+v (plan-service got user %s body %q)", got, c.gotUser, c.gotBody)
	}
}

// R2, R14, R19: a missing comment is 404 before any write, a stranger is 404
// before any read, and plan-service's own refusals keep their status.
func TestNoteCommentWrites_Refusals(t *testing.T) {
	stored := &NoteComment{ID: uuid.New(), AuthorID: memberP.ID, Body: "b"}
	cases := []struct {
		name       string
		rel        Relation
		c          *fakeComments
		wantStatus int
		wantCode   string
	}{
		{"comment missing", RelationMember, &fakeComments{getErr: status.Error(codes.NotFound, "no comment")}, http.StatusNotFound, string(errcode.NotFound)},
		{"stranger", RelationNone, &fakeComments{stored: stored}, http.StatusNotFound, string(errcode.NotFound)},
		{"backstop", RelationMember, &fakeComments{stored: stored, err: status.Error(codes.PermissionDenied, "not theirs")}, http.StatusForbidden, string(errcode.Forbidden)},
		{"empty body", RelationMember, &fakeComments{stored: stored, err: status.Error(codes.InvalidArgument, "invalid input: comment body must not be empty")}, http.StatusBadRequest, string(errcode.ValidationFailed)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := newCommentsAPI(t, tc.c, &fakePeople{people: []Person{memberP}}, tc.rel, memberP.ID).
				Patch(commentURL(stored.ID), map[string]any{"body": "x"})
			if resp.Code != tc.wantStatus || problemCode(t, resp) != tc.wantCode {
				t.Fatalf("status = %d, want %d %s: %s", resp.Code, tc.wantStatus, tc.wantCode, resp.Body.String())
			}
			if tc.wantStatus == http.StatusNotFound && len(tc.c.writes) != 0 {
				t.Fatalf("writes = %v, want none", tc.c.writes)
			}
		})
	}
}

// ADR-0005: the edit body is as strict as the create body.
func TestUpdateNoteComment_ShapeFailuresAre422(t *testing.T) {
	stored := &NoteComment{ID: uuid.New(), AuthorID: memberP.ID, Body: "b"}
	for name, body := range map[string]map[string]any{
		"authorId sent":   {"body": "hi", "authorId": uuid.NewString()},
		"missing body":    {},
		"non-string body": {"body": 7},
	} {
		t.Run(name, func(t *testing.T) {
			c := &fakeComments{stored: stored}
			resp := newCommentsAPI(t, c, &fakePeople{}, RelationMember, memberP.ID).Patch(commentURL(stored.ID), body)
			if resp.Code != http.StatusUnprocessableEntity || c.calls != 0 {
				t.Fatalf("status = %d, calls = %d; want 422 and none", resp.Code, c.calls)
			}
		})
	}
}
