package checklistitems

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/darkphotonKN/fireplace/internal/constants"
	"github.com/darkphotonKN/fireplace/internal/models"
	"github.com/google/uuid"
)

// mockRepository captures calls and returns canned values.
type mockRepository struct {
	getByIDItem *models.ChecklistItem
	getByIDErr  error

	lastUpdateDatesID    uuid.UUID
	lastUpdateDatesStart *time.Time
	lastUpdateDatesDue   *time.Time
	updateDatesErr       error
	updateDatesCalled    bool
}

func (m *mockRepository) Create(ctx context.Context, req CreateReq, planID uuid.UUID, sequenceNo int) (*models.ChecklistItem, error) {
	return nil, nil
}
func (m *mockRepository) Update(ctx context.Context, id uuid.UUID, req UpdateReq) error {
	return nil
}
func (m *mockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (m *mockRepository) GetAll(ctx context.Context, scope *string) ([]*models.ChecklistItem, error) {
	return nil, nil
}
func (m *mockRepository) GetAllByPlanId(ctx context.Context, planId uuid.UUID, scope *string, upcoming *string) ([]*models.ChecklistItem, error) {
	return nil, nil
}
func (m *mockRepository) GetAllArchivedByPlanId(ctx context.Context, planId uuid.UUID, scope *string) ([]*models.ChecklistItem, error) {
	return nil, nil
}
func (m *mockRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ChecklistItem, error) {
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
