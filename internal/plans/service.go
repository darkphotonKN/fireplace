package plans

import (
	"context"

	"github.com/darkphotonKN/fireplace/internal/constants"
	"github.com/darkphotonKN/fireplace/internal/logger"
	"github.com/darkphotonKN/fireplace/internal/models"
	"github.com/google/uuid"
)

type service struct {
	repo        Repository
	userService UserService
}

type Repository interface {
	GetById(ctx context.Context, id uuid.UUID) (*models.Plan, error)
	Create(ctx context.Context, plan models.Plan) (*models.Plan, error)
	Update(ctx context.Context, id uuid.UUID, req UpdatePlanReq, userID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	GetAll(ctx context.Context, userID uuid.UUID) ([]*models.Plan, error)
	GetAllShared(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Plan, error)
	CreateSharedPlans(ctx context.Context, planID uuid.UUID, userID uuid.UUID) error
	UpdateSharedPlans(ctx context.Context, planID uuid.UUID, userID uuid.UUID) error
	SearchPlan(ctx context.Context, userID uuid.UUID, params SearchParam) ([]*SearchPlanRes, error)
}

type UserService interface {
	GetById(ctx context.Context, id uuid.UUID) (*models.User, error)
}

func NewService(repo Repository) *service {
	return &service{
		repo: repo,
	}
}

func (s *service) GetById(ctx context.Context, id uuid.UUID) (*models.Plan, error) {
	return s.repo.GetById(ctx, id)
}

func (s *service) Create(ctx context.Context, req CreatePlanReq, userID uuid.UUID) (*models.Plan, error) {

	// default to true if its learning based, but false if its project based
	dailyReset := true

	logger.Debug("Comparing plan types", "requestType", constants.PlanType(req.PlanType), "projectType", constants.TypeProject)
	if constants.PlanType(req.PlanType) == constants.TypeProject {
		dailyReset = false
	}

	// Create a plan model from the request with user ID from auth (static for now)
	plan := models.Plan{
		UserID:      userID,
		Name:        req.Name,
		Focus:       req.Focus,
		Description: req.Description,
		PlanType:    req.PlanType,
		DailyReset:  dailyReset,
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

// GetAllShared returns all plans owned by or shared with the user
func (s *service) GetAllShared(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Plan, error) {
	return s.repo.GetAllShared(ctx, userID, limit, offset)
}

// Share a plan
func (s *service) SharePlan(ctx context.Context, planID uuid.UUID, userID uuid.UUID) error {
	// validate the user to share to
	_, err := s.userService.GetById(ctx, userID)

	if err != nil {
		return err
	}

	return s.repo.UpdateSharedPlans(ctx, planID, userID)
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

func (s *service) SearchPlan(ctx context.Context, userID uuid.UUID, params SearchParam) ([]*SearchPlanRes, error) {
	return s.repo.SearchPlan(ctx, userID, params)
}
