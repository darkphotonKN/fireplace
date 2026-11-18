package checklistitem

import (
	"context"
	"testing"

	"github.com/darkphotonKN/fireplace/common/errcode"
	commongrpc "github.com/darkphotonKN/fireplace/common/grpcerror"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FS-65B3N R4: a note with comments cannot become a task. The repository
// decides it inside the locked write (R5) and answers ErrNoteHasComments; the
// service gives that its wire form, FailedPrecondition with NOTE_HAS_COMMENTS.

// commentedRepo is a scopedRepo whose notes carry comments: converting one to
// a task is refused the way the real repository refuses it, writing nothing.
type commentedRepo struct {
	*scopedRepo
}

func (r *commentedRepo) Update(ctx context.Context, in UpdateItemInput) error {
	if it, err := r.inPlan(in.PlanID, in.ID); err == nil &&
		it.Type == TypeNote && in.Type != nil && *in.Type == TypeTask {
		return ErrNoteHasComments
	}
	return r.scopedRepo.Update(ctx, in)
}

func TestUpdate_NoteWithComments_ToTask_IsNoteHasCommentsAndUnchanged(t *testing.T) {
	note, scoped := noteInPlanB(strPtr("gist"))
	repo := &commentedRepo{scoped}

	_, err := newTestService(repo).Update(context.Background(), UpdateItemInput{
		ID: note.ID, PlanID: note.PlanID, Type: strPtr(TypeTask),
	})

	if got := status.Code(err); got != codes.FailedPrecondition {
		t.Fatalf("code = %v (err %v), want FailedPrecondition", got, err)
	}
	if code, ok := commongrpc.DomainCode(err); !ok || code != errcode.NoteHasComments {
		t.Errorf("domain code = %q (ok %v), want %q", code, ok, errcode.NoteHasComments)
	}
	if it := scoped.rows[note.ID]; it.Type != TypeNote || it.Summary == nil || *it.Summary != "gist" {
		t.Errorf("item = type %q summary %v, want the note unchanged", it.Type, it.Summary)
	}
}
