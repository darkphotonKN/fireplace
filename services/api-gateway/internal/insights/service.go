package insights

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/concepts"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/discovery"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/interfaces"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/models"
)

type service struct {
	repo               Repository
	contentGen         interfaces.ContentGenerator
	checklistService   ChecklistInsightsService
	planService        PlanInsightsService
	basePrompt         string
	youtubeVideoFinder InsightsYoutubeVideoFinder
}

type ChecklistInsightsService interface {
	GetAllByPlanId(ctx context.Context, planId uuid.UUID, scope *string, itemType *string, upcoming *string) ([]*models.ChecklistItem, error)
}

// PlanInsightsService is the narrow plan-side surface insights depends on.
// Local interface (no import of internal/plans) so this package is agnostic
// to whether plans are served in-process or via gRPC.
type PlanInsightsService interface {
	GetById(ctx context.Context, id uuid.UUID) (*models.Plan, error)
}

type InsightsYoutubeVideoFinder interface {
	FindResources(ctx context.Context, concepts []concepts.Concept) ([]discovery.Resource, error)
}

type Repository interface {
}

func NewService(repo Repository, contentGen interfaces.ContentGenerator, checklistService ChecklistInsightsService, planService PlanInsightsService, youtubeVideoFinder InsightsYoutubeVideoFinder) Service {
	return &service{
		repo:               repo,
		contentGen:         contentGen,
		checklistService:   checklistService,
		planService:        planService,
		basePrompt:         "",
		youtubeVideoFinder: youtubeVideoFinder,
	}
}

/**
* Generates the correct checklist item suggestion with some the context of user's focus and current checklist items.
**/
func (s *service) GenerateSuggestions(ctx context.Context, planId uuid.UUID) (string, error) {
	prompt, err := s.generatePromptWithChecklist(ctx, planId, "")
	if err != nil {
		return "", fmt.Errorf("insights: generate suggestions: %w", err)
	}

	res, err := s.contentGen.Generate(prompt)
	if err != nil {
		return "", fmt.Errorf("insights: generate suggestions: %w", err)
	}

	return res, nil
}

/**
* Generates 3 daily suggestions based on longterm checklist items and focus.
**/
func (s *service) GenerateDailySuggestions(ctx context.Context, planId uuid.UUID) ([]string, error) {
	// TODO: add default rules for daily suggestion to additional prompt argument.
	prompt, err := s.generatePromptWithChecklist(ctx, planId, "focus on tasks that are marked as \"longterm\" and breaking them down when you make your suggestions.")
	if err != nil {
		return nil, fmt.Errorf("insights: generate daily suggestions: %w", err)
	}

	suggestions := make([]string, 3)

	for i := 0; i < 3; i++ {
		// TODO:
		// - add each new entry in to prevent collision
		// - add longterm context
		//
		if i > 0 {
			prompt = fmt.Sprintf("%sAlso, don't choose one closely related to this specific action item as this has already been added to the list too:%s", prompt, suggestions[i-1])
		}
		res, err := s.contentGen.Generate(prompt)
		if err != nil {
			return nil, fmt.Errorf("insights: generate daily suggestions: %w", err)
		}
		suggestions[i] = res
	}

	return suggestions, nil
}

/**
* Generates the correct checklist item suggestion with some context of user's focus, current checklist items, and
* half finished user input for a checklist item.
**/
func (s *service) AutocompleteChecklistSuggestion(focus string) (string, error) {
	return "", nil
}

/**
* Takes a string prompt and converts it into a protocol for plan and checklist service to create a custom tailored plan.create a custom tailored plan.
**/
func (s *service) GenerateDailySummary() {
}

/**
* Sets up all the default checklist-based settings to for appropriate prompt string based on the checklists under a specific planId and any additional prompt information provided.
**/
func (s *service) generatePromptWithChecklist(ctx context.Context, planId uuid.UUID, additionalPrompt string) (string, error) {
	// sets primary prompt defaults
	// setup base prompt
	s.basePrompt = `
    Please suggest ONE specific, actionable task that would be the most valuable next step to add to my checklist.

    Your suggestion should:
    - Be a single, concrete task (not multiple tasks)
    - Start with a verb
    - Be specific enough to complete in a single sitting
    - Be directly relevant to the project focus
    - Use technical terminology accurately if applicable
    - Be 4-20 words in length

    Format your response as a single task item with no additional commentary, explanation or punctuation at the end.
		`

	// gather relevant data for constructing prompt
	focus, checklistPrompt, err := s.AcquireGenRelevantData(ctx, planId)

	if err != nil {
		return "", err
	}

	// focus - the primary topic input by the user for their plan.
	prompt := fmt.Sprintf(`Based on this project focus: "%s"
		%s
		So far the checklist already has these items, so either add one to follow the current progress or don't suggest one that's already present.
		This is the current existing checklist:
		%s
		%s
		`, focus, s.basePrompt, checklistPrompt, additionalPrompt)

	return prompt, nil
}

/**
* grabs relevant plan, checklist, focus data for LLM searches.
**/
func (s *service) AcquireGenRelevantData(ctx context.Context, planId uuid.UUID) (focus string, checklistItemPrompt string, error error) {

	// gets relavant planID and checklistItems
	plan, err := s.planService.GetById(ctx, planId)
	if err != nil {
		return "", "", fmt.Errorf("insights: acquire data: get plan %s: %w", planId, err)
	}

	// get entire checklist as context
	checklistItems, err := s.checklistService.GetAllByPlanId(ctx, planId, nil, nil, nil)
	if err != nil {
		return "", "", fmt.Errorf("insights: acquire data: get checklist items for plan %s: %w", planId, err)
	}

	// gets relavant focus from plan
	f := plan.Focus

	// construct the prompt context with checklist items
	c := ""

	for _, item := range checklistItems {
		c += fmt.Sprintf("A %s task: %s\n", item.Scope, item.Description)
	}

	return f, c, nil
}

/**
* Finds the focus and recent checklist items to find relevant search terms.
**/
func (s *service) GenerateSuggestedVideoLinks(ctx context.Context, planId uuid.UUID) ([]discovery.Resource, error) {
	// gather relevant data for constructing prompt
	focus, checklistPrompt, err := s.AcquireGenRelevantData(ctx, planId)
	if err != nil {
		return nil, fmt.Errorf("insights: suggested videos: %w", err)
	}

	// construct search term prompts

	message := fmt.Sprintf(`
	The user's focus for this task: %s
	Current checklist items for this task: 
	%s

	Please use this information to now provide exactly 3 relevant search terms.
	`, focus, checklistPrompt)

	searchTermsStr, err := s.contentGen.Generate(message)
	if err != nil {
		return nil, fmt.Errorf("insights: suggested videos: generate search terms: %w", err)
	}

	// format the string into individual search terms
	searchTerms := strings.Split(searchTermsStr, "\n")

	// crawl and find at least 5 suggested videos
	concepts := make([]concepts.Concept, len(searchTerms))

	for index, searchTerm := range searchTerms {
		concepts[index].Description = searchTerm
	}

	resources, err := s.youtubeVideoFinder.FindResources(ctx, concepts)
	if err != nil {
		return nil, fmt.Errorf("insights: suggested videos: find resources: %w", err)
	}

	return resources, nil
}
