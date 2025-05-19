package plans

import (
	"context"

	"github.com/darkphotonKN/fireplace/internal/models"
	"github.com/google/uuid"
)

type service struct {
	repo Repository
}

type Repository interface {
	GetById(ctx context.Context, id uuid.UUID) (*models.Plan, error)
	Create(ctx context.Context, plan models.Plan) (*models.Plan, error)
	Update(ctx context.Context, id uuid.UUID, req UpdatePlanReq, userID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	GetAll(ctx context.Context, userID uuid.UUID) ([]*models.Plan, error)
}

func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) GetById(ctx context.Context, id uuid.UUID) (*models.Plan, error) {
	return s.repo.GetById(ctx, id)
}

func (s *service) Create(ctx context.Context, req CreatePlanReq, userID uuid.UUID) (*models.Plan, error) {
	// Create a plan model from the request with user ID from auth (static for now)
	plan := models.Plan{
		UserID:      userID,
		Name:        req.Name,
		Focus:       req.Focus,
		Description: req.Description,
		PlanType:    req.PlanType,
	}

	// Call repository to create the plan
	return s.repo.Create(ctx, plan)
}

func (s *service) Update(ctx context.Context, id uuid.UUID, req UpdatePlanReq, userID uuid.UUID) error {
	return s.repo.Update(ctx, id, req, userID)
}

// GetAll returns all plans for a specific user
func (s *service) GetAll(ctx context.Context, userID uuid.UUID) ([]*models.Plan, error) {
	return s.repo.GetAll(ctx, userID)
}

// Delete removes a plan by ID if it belongs to the specified user
func (s *service) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.repo.Delete(ctx, id, userID)
}

func (s *service) ToggleDailyReset(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	// get corresponding plan, check the daily reset and flip it with an update

	plan, err := s.GetById(ctx, id)

	if err != nil {
		return err
	}

	// update the daily reset setting to opposite
	flippedResetState := !plan.DailyReset

	return s.repo.Update(ctx, id, UpdatePlanReq{
		DailyReset: &flippedResetState,
	}, userID)
}
