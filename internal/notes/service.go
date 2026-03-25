package notes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/darkphotonKN/fireplace/internal/models"
	"github.com/google/uuid"
)

type AIGenerator interface {
	GenerateContent(context string) (string, error)
}

// ChecklistService interface for checklist operations
type ChecklistService interface {
	GetAllByPlanId(ctx context.Context, planId uuid.UUID, scope *string, upcoming *string) ([]*models.ChecklistItem, error)
}

// PlanService interface for plan operations
type PlanService interface {
	GetById(ctx context.Context, id uuid.UUID) (*models.Plan, error)
}

type Service struct {
	repo               *Repository
	aiGenerator        AIGenerator
	checklistService   ChecklistService
	planService        PlanService
}

// NewService creates a new notes service
func NewService(
	repo *Repository,
	aiGenerator AIGenerator,
	checklistService ChecklistService,
	planService PlanService,
) *Service {
	return &Service{
		repo:             repo,
		aiGenerator:      aiGenerator,
		checklistService: checklistService,
		planService:      planService,
	}
}

// CreateNote creates a new note
func (s *Service) CreateNote(planID uuid.UUID, req *CreateNoteReq) (*Note, error) {
	// Validate plan exists
	ctx := context.Background()
	_, err := s.planService.GetById(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("plan not found: %w", err)
	}

	// Set defaults
	if req.Type == "" {
		req.Type = TypeUser
	}
	if req.Priority == "" {
		req.Priority = PriorityMedium
	}

	// Generate tags from content if not provided
	if len(req.Tags) == 0 {
		req.Tags = s.generateTagsFromContent(req.Content)
	}

	note := &Note{
		PlanID:         planID,
		Content:        req.Content,
		Type:           req.Type,
		Priority:       req.Priority,
		Tags:           req.Tags,
		RelatedTaskIDs: req.RelatedTaskIDs,
		AIMetadata:     req.AIMetadata,
	}

	return s.repo.Create(note)
}

// GetNoteByID retrieves a note by its ID
func (s *Service) GetNoteByID(id uuid.UUID) (*Note, error) {
	return s.repo.GetByID(id)
}

// GetNotesByPlanID retrieves all notes for a plan with optional filters
func (s *Service) GetNotesByPlanID(planID uuid.UUID, filters *FilterOptions) ([]Note, error) {
	return s.repo.GetByPlanID(planID, filters)
}

// UpdateNote updates an existing note
func (s *Service) UpdateNote(id uuid.UUID, updates *UpdateNoteReq) (*Note, error) {
	// Verify note exists
	_, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	return s.repo.Update(id, updates)
}

// DeleteNote removes a note
func (s *Service) DeleteNote(id uuid.UUID) error {
	return s.repo.Delete(id)
}

// GenerateAINotes generates AI-powered notes based on plan context
func (s *Service) GenerateAINotes(planID uuid.UUID, requestType string) ([]Note, error) {
	// Get plan details
	ctx := context.Background()
	plan, err := s.planService.GetById(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}

	// Get checklist items for context
	scope := "daily"
	checklistItems, err := s.checklistService.GetAllByPlanId(ctx, planID, &scope, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get checklist items: %w", err)
	}

	var notes []Note

	switch requestType {
	case "warning":
		note, err := s.generateWarningNote(plan, checklistItems)
		if err == nil && note != nil {
			notes = append(notes, *note)
		}

	case "insight":
		note, err := s.generateInsightNote(plan, checklistItems)
		if err == nil && note != nil {
			notes = append(notes, *note)
		}

	case "suggestion":
		note, err := s.generateSuggestionNote(plan, checklistItems)
		if err == nil && note != nil {
			notes = append(notes, *note)
		}

	default:
		// Generate all types
		if warning, err := s.generateWarningNote(plan, checklistItems); err == nil && warning != nil {
			notes = append(notes, *warning)
		}
		if insight, err := s.generateInsightNote(plan, checklistItems); err == nil && insight != nil {
			notes = append(notes, *insight)
		}
		if suggestion, err := s.generateSuggestionNote(plan, checklistItems); err == nil && suggestion != nil {
			notes = append(notes, *suggestion)
		}
	}

	return notes, nil
}

// generateWarningNote creates a warning based on task analysis
func (s *Service) generateWarningNote(plan *models.Plan, items []*models.ChecklistItem) (*Note, error) {
	var overdueTasks []*models.ChecklistItem
	var incompleteTasks []*models.ChecklistItem

	for _, item := range items {
		if !item.Done {
			incompleteTasks = append(incompleteTasks, item)
			if item.ScheduledTime != nil && item.ScheduledTime.Before(time.Now()) {
				overdueTasks = append(overdueTasks, item)
			}
		}
	}

	var content string
	var priority string
	var relatedTaskIDs []string

	if len(overdueTasks) > 0 {
		content = fmt.Sprintf("⚠️ You have %d overdue task(s) in %s. Consider reprioritizing or breaking them down into smaller steps.",
			len(overdueTasks), plan.Name)
		priority = PriorityHigh
		for _, task := range overdueTasks {
			relatedTaskIDs = append(relatedTaskIDs, task.ID.String())
		}
	} else if len(incompleteTasks) > 10 {
		content = fmt.Sprintf("📝 You have %d pending tasks in %s. Consider archiving completed items and focusing on your top priorities.",
			len(incompleteTasks), plan.Name)
		priority = PriorityMedium
	} else {
		content = fmt.Sprintf("✅ Your task list for %s is well-managed. Keep up the great work!", plan.Name)
		priority = PriorityLow
	}

	note := &Note{
		PlanID:         plan.ID,
		Content:        content,
		Type:           TypeWarning,
		Priority:       priority,
		Tags:           []string{"task-management", "productivity"},
		RelatedTaskIDs: relatedTaskIDs,
		AIMetadata: &AIMetadata{
			GeneratedFrom: "task_analysis",
			Confidence:    0.85,
			SourceContext: fmt.Sprintf("Analysis of %d tasks in %s", len(items), plan.Name),
			GeneratedAt:   time.Now().Format(time.RFC3339),
		},
	}

	return s.repo.Create(note)
}

// generateInsightNote creates an insight based on progress analysis
func (s *Service) generateInsightNote(plan *models.Plan, items []*models.ChecklistItem) (*Note, error) {
	completedCount := 0
	for _, item := range items {
		if item.Done {
			completedCount++
		}
	}

	completionRate := float64(0)
	if len(items) > 0 {
		completionRate = float64(completedCount) / float64(len(items)) * 100
	}

	var content string
	var priority string

	if completionRate > 70 {
		content = fmt.Sprintf("🎯 Excellent progress! You've completed %.0f%% of your tasks in %s. Consider adding new challenges to maintain momentum.",
			completionRate, plan.Name)
		priority = PriorityLow
	} else if completionRate > 40 {
		content = fmt.Sprintf("📊 You're making steady progress on %s with %.0f%% completion. Focus on completing 2-3 more tasks to build momentum.",
			plan.Name, completionRate)
		priority = PriorityMedium
	} else {
		content = fmt.Sprintf("💡 Consider breaking down complex tasks in %s into smaller, actionable steps to improve your %.0f%% completion rate.",
			plan.Name, completionRate)
		priority = PriorityMedium
	}

	note := &Note{
		PlanID:   plan.ID,
		Content:  content,
		Type:     TypeInsight,
		Priority: priority,
		Tags:     []string{"progress", "analytics", "motivation"},
		AIMetadata: &AIMetadata{
			GeneratedFrom: "progress_review",
			Confidence:    0.90,
			SourceContext: fmt.Sprintf("Progress analysis for %s", plan.Name),
			GeneratedAt:   time.Now().Format(time.RFC3339),
		},
	}

	return s.repo.Create(note)
}

// generateSuggestionNote creates a suggestion for next steps
func (s *Service) generateSuggestionNote(plan *models.Plan, items []*models.ChecklistItem) (*Note, error) {
	// If AI generator is available, use it for intelligent suggestions
	if s.aiGenerator != nil {
		context := fmt.Sprintf("Plan: %s\nFocus: %s\nTasks: %d total, analyze and suggest improvements",
			plan.Name, plan.Focus, len(items))

		aiContent, err := s.aiGenerator.GenerateContent(context)
		if err == nil && aiContent != "" {
			note := &Note{
				PlanID:   plan.ID,
				Content:  aiContent,
				Type:     TypeSuggestion,
				Priority: PriorityMedium,
				Tags:     []string{"ai-suggestion", "planning"},
				AIMetadata: &AIMetadata{
					GeneratedFrom: "manual_request",
					Confidence:    0.75,
					SourceContext: fmt.Sprintf("AI analysis for %s", plan.Name),
					GeneratedAt:   time.Now().Format(time.RFC3339),
				},
			}
			return s.repo.Create(note)
		}
	}

	// Fallback to rule-based suggestions
	var unscheduledTasks []*models.ChecklistItem
	for _, item := range items {
		if !item.Done && item.ScheduledTime == nil {
			unscheduledTasks = append(unscheduledTasks, item)
		}
	}

	var content string
	var relatedTaskIDs []string

	if len(unscheduledTasks) > 0 {
		content = fmt.Sprintf("📅 You have %d unscheduled tasks in %s. Consider scheduling them to better manage your time and priorities.",
			len(unscheduledTasks), plan.Name)
		// Add first 3 unscheduled task IDs
		for i, task := range unscheduledTasks {
			if i >= 3 {
				break
			}
			relatedTaskIDs = append(relatedTaskIDs, task.ID.String())
		}
	} else if len(items) == 0 {
		content = fmt.Sprintf("🚀 Your %s plan is empty. Start by adding some actionable tasks aligned with your focus: %s",
			plan.Name, plan.Focus)
	} else {
		content = fmt.Sprintf("🎯 All tasks in %s are scheduled! Review your long-term goals and consider adding new objectives.",
			plan.Name)
	}

	note := &Note{
		PlanID:         plan.ID,
		Content:        content,
		Type:           TypeSuggestion,
		Priority:       PriorityMedium,
		Tags:           []string{"scheduling", "planning", "productivity"},
		RelatedTaskIDs: relatedTaskIDs,
		AIMetadata: &AIMetadata{
			GeneratedFrom: "task_analysis",
			Confidence:    0.70,
			SourceContext: fmt.Sprintf("Scheduling analysis for %s", plan.Name),
			GeneratedAt:   time.Now().Format(time.RFC3339),
		},
	}

	return s.repo.Create(note)
}

// generateTagsFromContent analyzes content and generates relevant tags
func (s *Service) generateTagsFromContent(content string) []string {
	tags := []string{}
	lowerContent := strings.ToLower(content)

	// Time-based tags
	timeKeywords := map[string][]string{
		"morning":  {"morning", "am"},
		"evening":  {"evening", "pm", "night"},
		"today":    {"today"},
		"tomorrow": {"tomorrow"},
		"deadline": {"deadline", "due"},
		"urgent":   {"urgent", "asap", "critical"},
	}

	for tag, keywords := range timeKeywords {
		for _, keyword := range keywords {
			if strings.Contains(lowerContent, keyword) {
				tags = append(tags, tag)
				break
			}
		}
	}

	// Action tags
	actionKeywords := map[string][]string{
		"review":    {"review", "check", "examine"},
		"testing":   {"test", "testing", "qa"},
		"bug-fix":   {"bug", "fix", "issue", "error"},
		"feature":   {"feature", "enhancement", "improve"},
		"refactor":  {"refactor", "cleanup", "optimize"},
		"research":  {"research", "investigate", "explore"},
		"planning":  {"plan", "planning", "strategy"},
		"meeting":   {"meeting", "discussion", "sync"},
	}

	for tag, keywords := range actionKeywords {
		for _, keyword := range keywords {
			if strings.Contains(lowerContent, keyword) {
				tags = append(tags, tag)
				break
			}
		}
	}

	// Remove duplicates
	tagMap := make(map[string]bool)
	uniqueTags := []string{}
	for _, tag := range tags {
		if !tagMap[tag] {
			tagMap[tag] = true
			uniqueTags = append(uniqueTags, tag)
		}
	}

	return uniqueTags
}