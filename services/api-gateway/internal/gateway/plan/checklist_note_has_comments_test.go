package plangw

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/darkphotonKN/fireplace/common/errcode"
	commongrpc "github.com/darkphotonKN/fireplace/common/grpcerror"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
)

// FS-65B3N R4: plan-service refuses `type: task` on a note with comments with
// NOTE_HAS_COMMENTS as an ErrorInfo detail on FailedPrecondition. The gateway
// adds no mapping: the detail alone surfaces as 400 · NOTE_HAS_COMMENTS.

type commentedNoteItems struct {
	ChecklistsClient
}

func (commentedNoteItems) UpdateChecklist(ctx context.Context, planID, id, userID uuid.UUID, req UpdateChecklistReq) (*ChecklistResp, error) {
	return nil, commongrpc.WithCode(codes.FailedPrecondition, errcode.NoteHasComments,
		"this note has comments; delete them to make it a task")
}

func TestUpdateChecklist_NoteWithComments_ToTask_Is400NoteHasComments(t *testing.T) {
	resp := newDailyAPI(t, commentedNoteItems{}).Patch(itemURL(), map[string]any{"type": "task"})

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", resp.Code, resp.Body.String())
	}
	if ct := resp.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/problem+json") {
		t.Errorf("content type = %q, want application/problem+json", ct)
	}
	var problem struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if problem.Code != string(errcode.NoteHasComments) {
		t.Errorf("code = %q, want %q", problem.Code, errcode.NoteHasComments)
	}
}
