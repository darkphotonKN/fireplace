package plangw

import (
	"context"
	"time"

	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/models"
	"github.com/google/uuid"
)

// Adapter wraps Client and satisfies the in-process interfaces that the
// monolith's cross-domain consumers (insights, notes, calendar, useranalytics,
// jobs) used to get from internal/plans + internal/checklistitems. The
// monolith packages are deleted in 4c; each consumer now talks to plan-service
// via this adapter.
//
// user_id is required by GetPlanRequest but plan-service does not enforce
// ownership on direct reads — the gateway is a trusted caller. Where the
// calling interface doesn't surface a user_id (e.g. notes.PlanService.GetById,
// insights' planService.GetById), pass uuid.Nil. When ownership matters,
// callers should use AssertPlanOwnership explicitly.
type Adapter struct {
	client *Client
}

func NewAdapter(client *Client) *Adapter {
	return &Adapter{client: client}
}

// GetById satisfies notes.PlanService.GetById and insights' planService.GetById.
func (a *Adapter) GetById(ctx context.Context, id uuid.UUID) (*models.Plan, error) {
	p, err := a.client.GetPlan(ctx, id, uuid.Nil)
	if err != nil {
		return nil, err
	}
	return planRespToModel(p), nil
}

// AssertPlanOwnership satisfies calendar.PlanOwnership.
func (a *Adapter) AssertPlanOwnership(ctx context.Context, planID, userID uuid.UUID) error {
	return a.client.AssertPlanOwnership(ctx, planID, userID)
}

// GetAllByPlanId satisfies insights.ChecklistInsightsService and
// notes.ChecklistService. Routes to ListUpcomingItems when "upcoming" is set,
// to ListItems otherwise.
func (a *Adapter) GetAllByPlanId(ctx context.Context, planID uuid.UUID, scope *string, itemType *string, upcoming *string) ([]*models.ChecklistItem, error) {
	if upcoming != nil {
		items, err := a.client.ListUpcomingChecklists(ctx, planID, uuid.Nil)
		if err != nil {
			return nil, err
		}
		return checklistRespToModelSlice(items), nil
	}
	items, err := a.client.ListChecklists(ctx, planID, uuid.Nil, scope, itemType)
	if err != nil {
		return nil, err
	}
	return checklistRespToModelSlice(items), nil
}

// GetByUserID satisfies useranalytics' checklist dependency.
func (a *Adapter) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.ChecklistItem, error) {
	items, err := a.client.ListChecklistsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return checklistRespToModelSlice(items), nil
}

// ResetDailyItems satisfies jobs.ChecklistDailyResetService.
func (a *Adapter) ResetDailyItems(ctx context.Context) error {
	_, err := a.client.DailyReset(ctx)
	return err
}

// --- DTO conversions: gateway's PlanResp/ChecklistResp → monolith's
// models.Plan/models.ChecklistItem (still used by the cross-domain consumers'
// internal signatures). ---

func planRespToModel(p *PlanResp) *models.Plan {
	return &models.Plan{
		BaseDBDateModel: models.BaseDBDateModel{
			ID:        p.ID,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		},
		UserID:      p.UserID,
		Name:        p.Name,
		Focus:       p.Focus,
		Description: p.Description,
		PlanType:    p.PlanType,
		DailyReset:  p.DailyReset,
	}
}

func checklistRespToModel(c *ChecklistResp) *models.ChecklistItem {
	out := &models.ChecklistItem{
		BaseDBDateModel: models.BaseDBDateModel{
			ID:        c.ID,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		},
		PlanID:      c.PlanID,
		Description: c.Description,
		Done:        c.Done,
		Sequence:    c.Sequence,
		Type:        c.Type,
		Scope:       c.Scope,
		Archived:    c.Archived,
	}
	if c.ParentID != nil {
		out.ParentID = c.ParentID
	}
	if c.StartDate != nil {
		out.StartDate = c.StartDate
	}
	if c.DueDate != nil {
		out.DueDate = c.DueDate
	}
	_ = time.Time{}
	return out
}

func checklistRespToModelSlice(items []*ChecklistResp) []*models.ChecklistItem {
	out := make([]*models.ChecklistItem, 0, len(items))
	for _, c := range items {
		out = append(out, checklistRespToModel(c))
	}
	return out
}
