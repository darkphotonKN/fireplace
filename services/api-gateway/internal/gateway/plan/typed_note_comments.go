package plangw

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/apierr"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/google/uuid"
)

// SERIALIZED note comment operations (FS-65B3N §API surface).
//
// plan-service stores comments and holds their rules: notes only, the body's
// trim and length, the author taken from the caller. The gateway decides who
// may do what (ADR-0015): every operation asks the access seam first, and one
// policy, commentRights, both sets the flags a list answers and decides edit
// and delete, so the two can never disagree (R18). Author names come from the
// gateway's users table.

// NoteComment is a comment as plan-service knows it. EditedAt is nil until
// the first edit.
type NoteComment struct {
	ID        uuid.UUID
	ItemID    uuid.UUID
	AuthorID  uuid.UUID
	Body      string
	CreatedAt time.Time
	EditedAt  *time.Time
}

// NoteCommentsClient is plan-service's comment half. *Client satisfies it.
// userID is the caller: plan-service makes it the author of a new comment
// (R6).
type NoteCommentsClient interface {
	ListNoteComments(ctx context.Context, planID, itemID, userID uuid.UUID) ([]NoteComment, error)
	CreateNoteComment(ctx context.Context, planID, itemID, userID uuid.UUID, body string) (*NoteComment, error)
	// GetNoteComment reads one comment, for its author before an edit or
	// delete is decided (R18).
	GetNoteComment(ctx context.Context, planID, itemID, commentID, userID uuid.UUID) (*NoteComment, error)
	UpdateNoteComment(ctx context.Context, planID, itemID, commentID, userID uuid.UUID, body string) (*NoteComment, error)
	DeleteNoteComment(ctx context.Context, planID, itemID, commentID, userID uuid.UUID) error
}

// NoteCommentResp is one comment as the caller sees it (R12).
type NoteCommentResp struct {
	ID         uuid.UUID  `json:"id" format:"uuid" doc:"Comment id"`
	AuthorID   uuid.UUID  `json:"authorId" format:"uuid" doc:"The user who wrote it"`
	AuthorName *string    `json:"authorName" nullable:"true" doc:"The author's display name if set, otherwise account name; null when the account no longer exists"`
	Body       string     `json:"body" doc:"Plain text, trimmed, line breaks kept"`
	CreatedAt  time.Time  `json:"createdAt" doc:"When it was written"`
	EditedAt   *time.Time `json:"editedAt" nullable:"true" doc:"When it was last edited; null until the first edit"`
	CanEdit    bool       `json:"canEdit" doc:"Whether the caller may edit it: only its author may"`
	CanDelete  bool       `json:"canDelete" doc:"Whether the caller may delete it: its author, or the plan's owner"`
}

// NoteCommentReq is the body to add (and, in slice 2, edit) a comment. Strict:
// an unknown member such as an author is a 422 (R6, ADR-0005). The body's
// 1-2000 characters after trimming is plan-service's rule, so the schema
// carries no length bounds (R9).
type NoteCommentReq struct {
	Body string `json:"body" doc:"Plain text. Trimmed by the server; must then be 1-2000 characters, else 400 VALIDATION_FAILED"`
}

type NoteCommentOutput struct{ Body NoteCommentResp }
type NoteCommentListOutput struct{ Body []NoteCommentResp }

type CreateNoteCommentInput struct {
	PlanID      uuid.UUID `path:"id" doc:"Plan id"`
	ChecklistID uuid.UUID `path:"checklist_id" doc:"The note's item id"`
	Body        NoteCommentReq
}

// NoteCommentInput addresses one comment under its note and plan.
type NoteCommentInput struct {
	PlanID      uuid.UUID `path:"id" doc:"Plan id"`
	ChecklistID uuid.UUID `path:"checklist_id" doc:"The note's item id"`
	CommentID   uuid.UUID `path:"comment_id" doc:"Comment id"`
}

type UpdateNoteCommentInput struct {
	PlanID      uuid.UUID `path:"id" doc:"Plan id"`
	ChecklistID uuid.UUID `path:"checklist_id" doc:"The note's item id"`
	CommentID   uuid.UUID `path:"comment_id" doc:"Comment id"`
	Body        NoteCommentReq
}

// rights is what a caller may do with one comment.
type rights struct {
	Edit, Delete bool
}

// commentRights is R16-R18's one policy: the author may edit; the author or
// the plan's owner may delete; nobody without a relation may do either.
func commentRights(rel Relation, caller, author uuid.UUID) rights {
	if rel != RelationOwner && rel != RelationMember {
		return rights{}
	}
	own := caller == author
	return rights{Edit: own, Delete: own || rel == RelationOwner}
}

// commentAction is a write on a comment: the right it needs out of
// commentRights, and why it is refused without it.
type commentAction struct {
	allowed func(rights) bool
	refusal string
}

var (
	editComment = commentAction{
		allowed: func(r rights) bool { return r.Edit },
		refusal: "only a comment's author may edit it",
	}
	deleteComment = commentAction{
		allowed: func(r rights) bool { return r.Delete },
		refusal: "only a comment's author or the plan's owner may delete it",
	}
)

func RegisterNoteCommentOperations(api huma.API, c NoteCommentsClient, people People, access *Access,
	protect func(huma.Context, func(huma.Context)), secured []map[string][]string,
) {
	mw := huma.Middlewares{protect}

	// authorize resolves the caller and their relation: owner or member may
	// list and add (R14, R15), and the relation feeds commentRights.
	authorize := func(ctx context.Context, op string, planID uuid.UUID) (uuid.UUID, Relation, error) {
		userID, ok := auth.UserIDFromCtx(ctx)
		if !ok {
			return uuid.Nil, RelationNone, apierr.ProblemFor(op, apierr.ErrNoIdentity())
		}
		rel, err := access.Require(ctx, planID, userID, NeedMember)
		if err != nil {
			return uuid.Nil, RelationNone, apierr.ProblemFor(op, err)
		}
		return userID, rel, nil
	}

	huma.Register(api, huma.Operation{
		OperationID: "listNoteComments", Method: http.MethodGet,
		Path:        "/api/plans/{id}/checklists/{checklist_id}/comments",
		Middlewares: mw, Security: secured,
		Summary: "List a note's comments",
		Description: "Every comment on the note, oldest first, with no pagination; [] when it has none. " +
			"Open to the plan's owner and members. `canEdit` and `canDelete` say what the caller may do " +
			"with each. `authorName` is null for an author whose account no longer exists. " +
			"400 `ITEM_NOT_A_NOTE` when the item is a task: only notes have comments.",
		Errors: []int{
			http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound,
			http.StatusUnprocessableEntity, http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, in *ChecklistItemInput) (*NoteCommentListOutput, error) {
		const op = "list note comments"
		userID, rel, err := authorize(ctx, op, in.PlanID)
		if err != nil {
			return nil, err
		}
		comments, err := c.ListNoteComments(ctx, in.PlanID, in.ChecklistID, userID)
		if err != nil {
			return nil, apierr.ProblemFor(op, err)
		}
		names, err := authorNames(ctx, people, comments)
		if err != nil {
			return nil, apierr.ProblemFor(op, err)
		}
		out := make([]NoteCommentResp, 0, len(comments))
		for _, cm := range comments {
			out = append(out, commentResp(cm, names, rel, userID))
		}
		return &NoteCommentListOutput{Body: out}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "createNoteComment", Method: http.MethodPost,
		Path:          "/api/plans/{id}/checklists/{checklist_id}/comments",
		DefaultStatus: http.StatusCreated,
		Middlewares:   mw, Security: secured,
		Summary: "Add a comment to a note",
		Description: "Adds a comment by the caller, who is always its author: the body carries no author. " +
			"Open to the plan's owner and members. The body is trimmed and must then be 1-2000 characters, " +
			"else 400 `VALIDATION_FAILED`. 400 `ITEM_NOT_A_NOTE` when the item is a task. Adding a comment " +
			"never changes the note itself.",
		Errors: []int{
			http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound,
			http.StatusUnprocessableEntity, http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, in *CreateNoteCommentInput) (*NoteCommentOutput, error) {
		const op = "create note comment"
		userID, rel, err := authorize(ctx, op, in.PlanID)
		if err != nil {
			return nil, err
		}
		cm, err := c.CreateNoteComment(ctx, in.PlanID, in.ChecklistID, userID, in.Body.Body)
		if err != nil {
			return nil, apierr.ProblemFor(op, err)
		}
		names, err := authorNames(ctx, people, []NoteComment{*cm})
		if err != nil {
			return nil, apierr.ProblemFor(op, err)
		}
		return &NoteCommentOutput{Body: commentResp(*cm, names, rel, userID)}, nil
	})

	// authorizeWrite is the edit and delete gate (R16-R18): the access seam,
	// then the comment's author from plan-service, then commentRights, the
	// same function that sets the flags a list answers. A refusal is 403 and
	// nothing is written; a comment out of scope is plan-service's 404.
	authorizeWrite := func(ctx context.Context, op string, in NoteCommentInput, action commentAction) (uuid.UUID, Relation, error) {
		userID, rel, err := authorize(ctx, op, in.PlanID)
		if err != nil {
			return uuid.Nil, RelationNone, err
		}
		cm, err := c.GetNoteComment(ctx, in.PlanID, in.ChecklistID, in.CommentID, userID)
		if err != nil {
			return uuid.Nil, RelationNone, apierr.ProblemFor(op, err)
		}
		if !action.allowed(commentRights(rel, userID, cm.AuthorID)) {
			return uuid.Nil, RelationNone, apierr.ProblemFor(op,
				fmt.Errorf("%w: %s", commonconstants.ErrForbidden, action.refusal))
		}
		return userID, rel, nil
	}

	huma.Register(api, huma.Operation{
		OperationID: "updateNoteComment", Method: http.MethodPatch,
		Path:        "/api/plans/{id}/checklists/{checklist_id}/comments/{comment_id}",
		Middlewares: mw, Security: secured,
		Summary: "Edit a comment on a note",
		Description: "Replaces the comment's body and marks it edited; no earlier version is kept. Only the " +
			"comment's author may edit it, the plan's owner included: anyone else with access gets 403 " +
			"`FORBIDDEN`. The body is trimmed and must then be 1-2000 characters, else 400 " +
			"`VALIDATION_FAILED`. The same text as stored changes nothing and leaves `editedAt` as it was. " +
			"404 when the comment is not on this note.",
		Errors: []int{
			http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound,
			http.StatusUnprocessableEntity, http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, in *UpdateNoteCommentInput) (*NoteCommentOutput, error) {
		const op = "update note comment"
		ref := NoteCommentInput{PlanID: in.PlanID, ChecklistID: in.ChecklistID, CommentID: in.CommentID}
		userID, rel, err := authorizeWrite(ctx, op, ref, editComment)
		if err != nil {
			return nil, err
		}
		cm, err := c.UpdateNoteComment(ctx, in.PlanID, in.ChecklistID, in.CommentID, userID, in.Body.Body)
		if err != nil {
			return nil, apierr.ProblemFor(op, err)
		}
		names, err := authorNames(ctx, people, []NoteComment{*cm})
		if err != nil {
			return nil, apierr.ProblemFor(op, err)
		}
		return &NoteCommentOutput{Body: commentResp(*cm, names, rel, userID)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "deleteNoteComment", Method: http.MethodDelete,
		Path:          "/api/plans/{id}/checklists/{checklist_id}/comments/{comment_id}",
		DefaultStatus: http.StatusNoContent,
		Middlewares:   mw, Security: secured,
		Summary: "Delete a comment on a note",
		Description: "Deletes the comment for good and answers 204 with no body. Its author may delete it, " +
			"and so may the plan's owner, whoever wrote it; anyone else with access gets 403 `FORBIDDEN`. " +
			"404 when the comment is not on this note, including one already deleted.",
		Errors: []int{
			http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound,
			http.StatusUnprocessableEntity, http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, in *NoteCommentInput) (*DeleteOutput, error) {
		const op = "delete note comment"
		userID, _, err := authorizeWrite(ctx, op, *in, deleteComment)
		if err != nil {
			return nil, err
		}
		if err := c.DeleteNoteComment(ctx, in.PlanID, in.ChecklistID, in.CommentID, userID); err != nil {
			return nil, apierr.ProblemFor(op, err)
		}
		return &DeleteOutput{}, nil
	})
}

// authorNames resolves the comments' authors in one batch (R13). An author
// with no account is absent from the map.
func authorNames(ctx context.Context, people People, comments []NoteComment) (map[uuid.UUID]string, error) {
	names := map[uuid.UUID]string{}
	if len(comments) == 0 {
		return names, nil
	}
	seen := map[uuid.UUID]bool{}
	ids := make([]uuid.UUID, 0, len(comments))
	for _, cm := range comments {
		if !seen[cm.AuthorID] {
			seen[cm.AuthorID] = true
			ids = append(ids, cm.AuthorID)
		}
	}
	found, err := people.PeopleByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, p := range found {
		names[p.ID] = p.Name
	}
	return names, nil
}

func commentResp(cm NoteComment, names map[uuid.UUID]string, rel Relation, caller uuid.UUID) NoteCommentResp {
	may := commentRights(rel, caller, cm.AuthorID)
	out := NoteCommentResp{
		ID: cm.ID, AuthorID: cm.AuthorID, Body: cm.Body,
		CreatedAt: cm.CreatedAt, EditedAt: cm.EditedAt,
		CanEdit: may.Edit, CanDelete: may.Delete,
	}
	if name, ok := names[cm.AuthorID]; ok {
		out.AuthorName = &name
	}
	return out
}
