package notes

import (
	"context"
	"errors"
	"testing"

	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/models"
	"github.com/google/uuid"
)

// FS-02JBZ R56 / S13: with the daily scope retired, AI notes builds its context
// from the plan's long-term TASK items. Reading daily items would silently hand
// every generator an empty list.

type contextRecorder struct {
	gotScope, gotType *string
}

var errStopAfterRead = errors.New("stop after the context read")

// GetAllByPlanId records the filters and then fails, so GenerateAINotes stops
// before any generator reaches the notes repository (a database).
func (r *contextRecorder) GetAllByPlanId(ctx context.Context, planID uuid.UUID, scope, itemType, upcoming *string) ([]*models.ChecklistItem, error) {
	r.gotScope, r.gotType = scope, itemType
	return nil, errStopAfterRead
}

type onePlan struct{}

func (onePlan) GetById(ctx context.Context, id uuid.UUID) (*models.Plan, error) {
	return &models.Plan{Name: "fixture"}, nil
}

func TestGenerateAINotes_ContextIsTheLongtermTaskItems(t *testing.T) {
	rec := &contextRecorder{}
	svc := NewService(nil, nil, rec, onePlan{})

	_, err := svc.GenerateAINotes(uuid.New(), "warning")

	if !errors.Is(err, errStopAfterRead) {
		t.Fatalf("err = %v, want the recorder's stop error", err)
	}
	if rec.gotScope == nil || *rec.gotScope != "longterm" {
		t.Errorf("scope filter = %v, want \"longterm\"", deref(rec.gotScope))
	}
	if rec.gotType == nil || *rec.gotType != "task" {
		t.Errorf("type filter = %v, want \"task\"", deref(rec.gotType))
	}
}

func deref(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
