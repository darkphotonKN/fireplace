package checklistitems

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/constants"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/models"
	"github.com/google/uuid"
)

// mockRepository captures calls and returns canned values.
type mockRepository struct {
	// GetByID can return different items per id via getByIDByID; falls back to getByIDItem.
	getByIDItem *models.ChecklistItem
	getByIDErr  error
	getByIDByID map[uuid.UUID]*models.ChecklistItem

	lastUpdateDatesID    uuid.UUID
	lastUpdateDatesStart *time.Time
	lastUpdateDatesDue   *time.Time
	updateDatesErr       error
	updateDatesCalled    bool

	// Create / Update capture
	createCalled  bool
	lastCreateReq CreateReq
	createReturn  *models.ChecklistItem

	updateCalled  bool
	lastUpdateID  uuid.UUID
	lastUpdateReq UpdateReq

	// HasChildren: keyed by id; default false
	hasChildrenForID map[uuid.UUID]bool

	// GetAllByPlanId capture
	lastGetAllPlanID uuid.UUID
	lastGetAllScope  *string
	lastGetAllType   *string
	lastGetAllUpc    *string
}

func (m *mockRepository) Create(ctx context.Context, req CreateReq, planID uuid.UUID, sequenceNo int) (*models.ChecklistItem, error) {
	m.createCalled = true
	m.lastCreateReq = req
	if m.createReturn != nil {
		return m.createReturn, nil
	}
	return &models.ChecklistItem{BaseDBDateModel: models.BaseDBDateModel{ID: uuid.New()}, PlanID: planID}, nil
}
func (m *mockRepository) Update(ctx context.Context, id uuid.UUID, req UpdateReq) error {
	m.updateCalled = true
	m.lastUpdateID = id
	m.lastUpdateReq = req
	return nil
}
func (m *mockRepository) HasChildren(ctx context.Context, id uuid.UUID) (bool, error) {
	if m.hasChildrenForID == nil {
		return false, nil
	}
	return m.hasChildrenForID[id], nil
}
func (m *mockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (m *mockRepository) GetAll(ctx context.Context, scope *string) ([]*models.ChecklistItem, error) {
	return nil, nil
}
func (m *mockRepository) GetAllByPlanId(ctx context.Context, planId uuid.UUID, scope *string, itemType *string, upcoming *string) ([]*models.ChecklistItem, error) {
	m.lastGetAllPlanID = planId
	m.lastGetAllScope = scope
	m.lastGetAllType = itemType
	m.lastGetAllUpc = upcoming
	return nil, nil
}
func (m *mockRepository) GetAllArchivedByPlanId(ctx context.Context, planId uuid.UUID, scope *string) ([]*models.ChecklistItem, error) {
	return nil, nil
}
func (m *mockRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ChecklistItem, error) {
	if m.getByIDByID != nil {
		if item, ok := m.getByIDByID[id]; ok {
			return item, nil
		}
	}
	return m.getByIDItem, m.getByIDErr
}
func (m *mockRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.ChecklistItem, error) {
	return nil, nil
}
func (m *mockRepository) CountItems(ctx context.Context) (int, error) { return 0, nil }
func (m *mockRepository) BulkResetDailyItems(ctx context.Context) error {
	return nil
}
func (m *mockRepository) UpdateDates(ctx context.Context, id uuid.UUID, startDate, dueDate *time.Time) error {
	m.updateDatesCalled = true
	m.lastUpdateDatesID = id
	m.lastUpdateDatesStart = startDate
	m.lastUpdateDatesDue = dueDate
	return m.updateDatesErr
}

func dateUTC(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestUpdateDates_BothFieldsPresent_PersistsBoth(t *testing.T) {
	id := uuid.New()
	repo := &mockRepository{getByIDItem: &models.ChecklistItem{}}
	svc := NewService(repo)

	start := optDate{Present: true, Valid: true, Value: dateUTC(2026, 3, 10)}
	due := optDate{Present: true, Valid: true, Value: dateUTC(2026, 3, 14)}

	if _, err := svc.UpdateDates(context.Background(), id, UpdateDatesReq{StartDate: start, DueDate: due}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.updateDatesCalled {
		t.Fatal("expected repo.UpdateDates to be called")
	}
	if repo.lastUpdateDatesStart == nil || !repo.lastUpdateDatesStart.Equal(dateUTC(2026, 3, 10)) {
		t.Errorf("expected start_date 2026-03-10, got %v", repo.lastUpdateDatesStart)
	}
	if repo.lastUpdateDatesDue == nil || !repo.lastUpdateDatesDue.Equal(dateUTC(2026, 3, 14)) {
		t.Errorf("expected due_date 2026-03-14, got %v", repo.lastUpdateDatesDue)
	}
}

func TestUpdateDates_StartAfterDue_ReturnsError_NoPersist(t *testing.T) {
	repo := &mockRepository{getByIDItem: &models.ChecklistItem{}}
	svc := NewService(repo)

	start := optDate{Present: true, Valid: true, Value: dateUTC(2026, 3, 20)}
	due := optDate{Present: true, Valid: true, Value: dateUTC(2026, 3, 15)}

	_, err := svc.UpdateDates(context.Background(), uuid.New(), UpdateDatesReq{StartDate: start, DueDate: due})
	if err == nil {
		t.Fatal("expected validation error for start > due")
	}
	if !errors.Is(err, constants.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
	if repo.updateDatesCalled {
		t.Error("expected repo.UpdateDates NOT to be called on validation error")
	}
}

func TestUpdateDates_PartialBody_MergesWithExisting(t *testing.T) {
	id := uuid.New()
	existingDue := dateUTC(2026, 3, 14)
	repo := &mockRepository{
		getByIDItem: &models.ChecklistItem{DueDate: &existingDue},
	}
	svc := NewService(repo)

	// Only startDate present in request; dueDate absent.
	start := optDate{Present: true, Valid: true, Value: dateUTC(2026, 3, 10)}

	if _, err := svc.UpdateDates(context.Background(), id, UpdateDatesReq{StartDate: start}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.lastUpdateDatesStart == nil || !repo.lastUpdateDatesStart.Equal(dateUTC(2026, 3, 10)) {
		t.Errorf("expected start_date 2026-03-10, got %v", repo.lastUpdateDatesStart)
	}
	if repo.lastUpdateDatesDue == nil || !repo.lastUpdateDatesDue.Equal(existingDue) {
		t.Errorf("expected due_date preserved as 2026-03-14, got %v", repo.lastUpdateDatesDue)
	}
}

func TestUpdateDates_PartialBody_ValidatesAgainstExisting(t *testing.T) {
	existingDue := dateUTC(2026, 3, 14)
	repo := &mockRepository{
		getByIDItem: &models.ChecklistItem{DueDate: &existingDue},
	}
	svc := NewService(repo)

	// startDate after the existing dueDate — must be rejected.
	start := optDate{Present: true, Valid: true, Value: dateUTC(2026, 3, 20)}

	_, err := svc.UpdateDates(context.Background(), uuid.New(), UpdateDatesReq{StartDate: start})
	if err == nil {
		t.Fatal("expected validation error against existing due_date")
	}
	if repo.updateDatesCalled {
		t.Error("expected repo.UpdateDates NOT to be called")
	}
}

func TestUpdateDates_ExplicitNull_ClearsField(t *testing.T) {
	id := uuid.New()
	existingStart := dateUTC(2026, 3, 10)
	existingDue := dateUTC(2026, 3, 14)
	repo := &mockRepository{
		getByIDItem: &models.ChecklistItem{StartDate: &existingStart, DueDate: &existingDue},
	}
	svc := NewService(repo)

	// dueDate explicitly null; startDate absent.
	due := optDate{Present: true, Valid: false}

	if _, err := svc.UpdateDates(context.Background(), id, UpdateDatesReq{DueDate: due}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.lastUpdateDatesStart == nil || !repo.lastUpdateDatesStart.Equal(existingStart) {
		t.Errorf("expected start_date preserved, got %v", repo.lastUpdateDatesStart)
	}
	if repo.lastUpdateDatesDue != nil {
		t.Errorf("expected due_date cleared (nil), got %v", repo.lastUpdateDatesDue)
	}
}

func TestUpdateDates_AbsentBody_NoChange(t *testing.T) {
	repo := &mockRepository{getByIDItem: &models.ChecklistItem{}}
	svc := NewService(repo)

	// Both fields absent — request is a no-op.
	_, err := svc.UpdateDates(context.Background(), uuid.New(), UpdateDatesReq{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.updateDatesCalled {
		t.Error("expected repo NOT to be called when no fields present")
	}
}

func TestUpdateDates_ItemNotFound_ReturnsError(t *testing.T) {
	repo := &mockRepository{getByIDErr: constants.ErrNotFound}
	svc := NewService(repo)

	start := optDate{Present: true, Valid: true, Value: dateUTC(2026, 3, 10)}
	_, err := svc.UpdateDates(context.Background(), uuid.New(), UpdateDatesReq{StartDate: start})
	if err == nil {
		t.Fatal("expected error when item not found")
	}
}

// ---------- #42: Create / Update / parent_id / type ----------

func TestCreate_AllowsScopeDaily(t *testing.T) {
	// The PRD's "no manual daily POST" rule was dropped — the AI accept flow
	// also POSTs here and we cannot distinguish it. UX gating now lives on
	// the FE (dailyAIOnly prop). See service.Create comment.
	repo := &mockRepository{}
	svc := NewService(repo)
	daily := "daily"

	if _, err := svc.Create(context.Background(), CreateReq{Description: "x", Scope: &daily}, uuid.New()); err != nil {
		t.Fatalf("expected no error for scope=daily, got %v", err)
	}
	if !repo.createCalled {
		t.Error("expected repo.Create to be called")
	}
}

func TestCreate_RejectsBogusScope(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo)
	bogus := "weekly"

	_, err := svc.Create(context.Background(), CreateReq{Description: "x", Scope: &bogus}, uuid.New())
	if err == nil || !errors.Is(err, constants.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	if repo.createCalled {
		t.Error("expected repo.Create NOT to be called for bogus scope")
	}
}

func TestCreate_LongtermStillWorks(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo)
	longterm := "longterm"

	if _, err := svc.Create(context.Background(), CreateReq{Description: "x", Scope: &longterm}, uuid.New()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.createCalled {
		t.Error("expected repo.Create to be called")
	}
}

func TestCreate_WithParentID_ValidParent_Succeeds(t *testing.T) {
	planID := uuid.New()
	parentID := uuid.New()
	repo := &mockRepository{
		getByIDByID: map[uuid.UUID]*models.ChecklistItem{
			parentID: {BaseDBDateModel: models.BaseDBDateModel{ID: parentID}, PlanID: planID, ParentID: nil},
		},
	}
	svc := NewService(repo)

	if _, err := svc.Create(context.Background(), CreateReq{Description: "child", ParentID: &parentID}, planID); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.createCalled {
		t.Error("expected repo.Create called")
	}
	if repo.lastCreateReq.ParentID == nil || *repo.lastCreateReq.ParentID != parentID {
		t.Errorf("expected parent_id forwarded to repo, got %v", repo.lastCreateReq.ParentID)
	}
}

func TestCreate_WithParentID_ParentIsChild_Returns400(t *testing.T) {
	planID := uuid.New()
	grandparent := uuid.New()
	parentID := uuid.New() // this row IS a child
	repo := &mockRepository{
		getByIDByID: map[uuid.UUID]*models.ChecklistItem{
			parentID: {BaseDBDateModel: models.BaseDBDateModel{ID: parentID}, PlanID: planID, ParentID: &grandparent},
		},
	}
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), CreateReq{Description: "x", ParentID: &parentID}, planID)
	if err == nil || !errors.Is(err, constants.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	if repo.createCalled {
		t.Error("expected repo.Create NOT to be called")
	}
}

func TestCreate_WithParentID_DifferentPlan_Returns400(t *testing.T) {
	planID := uuid.New()
	otherPlan := uuid.New()
	parentID := uuid.New()
	repo := &mockRepository{
		getByIDByID: map[uuid.UUID]*models.ChecklistItem{
			parentID: {BaseDBDateModel: models.BaseDBDateModel{ID: parentID}, PlanID: otherPlan, ParentID: nil},
		},
	}
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), CreateReq{Description: "x", ParentID: &parentID}, planID)
	if err == nil || !errors.Is(err, constants.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdate_TypeNoteOnDoneTask_ClearsDone(t *testing.T) {
	id := uuid.New()
	tru := true
	repo := &mockRepository{
		getByIDByID: map[uuid.UUID]*models.ChecklistItem{
			id: {BaseDBDateModel: models.BaseDBDateModel{ID: id}, Done: true, Type: "task"},
		},
	}
	svc := NewService(repo)

	noteType := "note"
	if err := svc.Update(context.Background(), id, UpdateReq{Type: &noteType}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !repo.updateCalled {
		t.Fatal("expected repo.Update called")
	}
	if repo.lastUpdateReq.Type == nil || *repo.lastUpdateReq.Type != "note" {
		t.Errorf("expected type='note' forwarded, got %v", repo.lastUpdateReq.Type)
	}
	if repo.lastUpdateReq.Done == nil || *repo.lastUpdateReq.Done != false {
		t.Errorf("expected done=false forced on type=note transition, got %v", repo.lastUpdateReq.Done)
	}
	_ = tru
}

func TestUpdate_TypeNoteOnAlreadyNotDone_DoesNotForceDone(t *testing.T) {
	id := uuid.New()
	repo := &mockRepository{
		getByIDByID: map[uuid.UUID]*models.ChecklistItem{
			id: {BaseDBDateModel: models.BaseDBDateModel{ID: id}, Done: false, Type: "task"},
		},
	}
	svc := NewService(repo)

	noteType := "note"
	if err := svc.Update(context.Background(), id, UpdateReq{Type: &noteType}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// done is already false; service shouldn't bother setting it explicitly.
	if repo.lastUpdateReq.Done != nil {
		t.Errorf("expected service NOT to touch done when already false, got %v", repo.lastUpdateReq.Done)
	}
}

func TestUpdate_ParentID_ValidIndent_Succeeds(t *testing.T) {
	planID := uuid.New()
	id := uuid.New()
	parentID := uuid.New()
	repo := &mockRepository{
		getByIDByID: map[uuid.UUID]*models.ChecklistItem{
			id:       {BaseDBDateModel: models.BaseDBDateModel{ID: id}, PlanID: planID, ParentID: nil},
			parentID: {BaseDBDateModel: models.BaseDBDateModel{ID: parentID}, PlanID: planID, ParentID: nil},
		},
	}
	svc := NewService(repo)

	pid := optUUID{Present: true, Valid: true, Value: parentID}
	if err := svc.Update(context.Background(), id, UpdateReq{ParentID: pid}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.lastUpdateReq.ParentID.Present || !repo.lastUpdateReq.ParentID.Valid {
		t.Error("expected parent_id forwarded as set")
	}
	if repo.lastUpdateReq.ParentID.Value != parentID {
		t.Errorf("expected parent_id=%s, got %s", parentID, repo.lastUpdateReq.ParentID.Value)
	}
}

func TestUpdate_ParentID_Outdent_Succeeds(t *testing.T) {
	id := uuid.New()
	parentID := uuid.New()
	planID := uuid.New()
	repo := &mockRepository{
		getByIDByID: map[uuid.UUID]*models.ChecklistItem{
			id: {BaseDBDateModel: models.BaseDBDateModel{ID: id}, PlanID: planID, ParentID: &parentID},
		},
	}
	svc := NewService(repo)

	pid := optUUID{Present: true, Valid: false}
	if err := svc.Update(context.Background(), id, UpdateReq{ParentID: pid}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.lastUpdateReq.ParentID.Present || repo.lastUpdateReq.ParentID.Valid {
		t.Error("expected parent_id forwarded as cleared (Present, !Valid)")
	}
}

func TestUpdate_ParentID_DifferentPlan_Returns400(t *testing.T) {
	planA := uuid.New()
	planB := uuid.New()
	id := uuid.New()
	parentID := uuid.New()
	repo := &mockRepository{
		getByIDByID: map[uuid.UUID]*models.ChecklistItem{
			id:       {BaseDBDateModel: models.BaseDBDateModel{ID: id}, PlanID: planA, ParentID: nil},
			parentID: {BaseDBDateModel: models.BaseDBDateModel{ID: parentID}, PlanID: planB, ParentID: nil},
		},
	}
	svc := NewService(repo)

	pid := optUUID{Present: true, Valid: true, Value: parentID}
	err := svc.Update(context.Background(), id, UpdateReq{ParentID: pid})
	if err == nil || !errors.Is(err, constants.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	if repo.updateCalled {
		t.Error("expected repo NOT called on validation error")
	}
}

func TestUpdate_ParentID_TargetIsChild_Returns400(t *testing.T) {
	planID := uuid.New()
	id := uuid.New()
	grandparent := uuid.New()
	parentID := uuid.New() // is a child
	repo := &mockRepository{
		getByIDByID: map[uuid.UUID]*models.ChecklistItem{
			id:       {BaseDBDateModel: models.BaseDBDateModel{ID: id}, PlanID: planID, ParentID: nil},
			parentID: {BaseDBDateModel: models.BaseDBDateModel{ID: parentID}, PlanID: planID, ParentID: &grandparent},
		},
	}
	svc := NewService(repo)

	pid := optUUID{Present: true, Valid: true, Value: parentID}
	err := svc.Update(context.Background(), id, UpdateReq{ParentID: pid})
	if err == nil || !errors.Is(err, constants.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdate_ParentID_RowHasChildren_Returns400(t *testing.T) {
	planID := uuid.New()
	id := uuid.New()
	parentID := uuid.New()
	repo := &mockRepository{
		getByIDByID: map[uuid.UUID]*models.ChecklistItem{
			id:       {BaseDBDateModel: models.BaseDBDateModel{ID: id}, PlanID: planID, ParentID: nil},
			parentID: {BaseDBDateModel: models.BaseDBDateModel{ID: parentID}, PlanID: planID, ParentID: nil},
		},
		hasChildrenForID: map[uuid.UUID]bool{id: true},
	}
	svc := NewService(repo)

	pid := optUUID{Present: true, Valid: true, Value: parentID}
	err := svc.Update(context.Background(), id, UpdateReq{ParentID: pid})
	if err == nil || !errors.Is(err, constants.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput when re-parenting a row with children, got %v", err)
	}
	if repo.updateCalled {
		t.Error("expected repo NOT called")
	}
}

func TestGetAllByPlanId_TypeFilter_ForwardedToRepo(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo)
	noteType := "note"

	if _, err := svc.GetAllByPlanId(context.Background(), uuid.New(), nil, &noteType, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.lastGetAllType == nil || *repo.lastGetAllType != "note" {
		t.Errorf("expected type='note' forwarded, got %v", repo.lastGetAllType)
	}
}

func TestGetAllByPlanId_InvalidType_Returns400(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo)
	bogus := "bogus"

	_, err := svc.GetAllByPlanId(context.Background(), uuid.New(), nil, &bogus, nil)
	if err == nil || !errors.Is(err, constants.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdate_TypeNoteOnParentWithChildren_Returns400(t *testing.T) {
	id := uuid.New()
	repo := &mockRepository{
		getByIDByID: map[uuid.UUID]*models.ChecklistItem{
			id: {BaseDBDateModel: models.BaseDBDateModel{ID: id}, Type: "task", Done: false},
		},
		hasChildrenForID: map[uuid.UUID]bool{id: true},
	}
	svc := NewService(repo)

	noteType := "note"
	err := svc.Update(context.Background(), id, UpdateReq{Type: &noteType})
	if err == nil || !errors.Is(err, constants.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput when flipping a parent to note, got %v", err)
	}
	if repo.updateCalled {
		t.Error("expected repo.Update NOT to be called")
	}
}
