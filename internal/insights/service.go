package insights

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/darkphotonKN/fireplace/internal/interfaces"
	"github.com/darkphotonKN/fireplace/internal/models"
	"github.com/darkphotonKN/fireplace/internal/plans"
)

type service struct {
	repo             Repository
	contentGen       interfaces.ContentGenerator
	checklistService ChecklistInsightsService
	planService      plans.Service
	basePrompt       string
}

type ChecklistInsightsService interface {
	GetAllByPlanId(ctx context.Context, planId uuid.UUID, scope *string) ([]*models.ChecklistItem, error)
}

type Repository interface {
}

func NewService(repo Repository, contentGen interfaces.ContentGenerator, checklistService ChecklistInsightsService, planService plans.Service) Service {
	// setup base prompt
	basePrompt := `
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

	return &service{
		repo:             repo,
		contentGen:       contentGen,
		checklistService: checklistService,
		planService:      planService,
		basePrompt:       basePrompt,
	}
}

/**
* Generates the correct checklist item suggestion with some the context of user's focus and current checklist items.
**/
func (s *service) GenerateSuggestions(ctx context.Context, planId uuid.UUID) (string, error) {
	prompt, err := s.generatePromptWithChecklist(ctx, planId, "")
	if err != nil {
		return "", err
	}

	res, err := s.contentGen.Generate(prompt)

	if err != nil {
		return "", err
	}

	return res, nil
}

func (s *service) GenerateDailySuggestions(ctx context.Context, planId uuid.UUID) ([]string, error) {
	// TODO: add default rules for daily suggestion to additional prompt argument.
	prompt, err := s.generatePromptWithChecklist(ctx, planId, "focus on tasks that are marked as \"longterm\" and breaking them down when you make your suggestions.")
	if err != nil {
		return nil, err
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
		fmt.Println("updated prompt:", prompt)
		res, err := s.contentGen.Generate(prompt)
		if err != nil {
			return nil, err
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

func (s *service) GenerateDailyReview() {
}

func (s *service) GenerateDailySummary() {
}

/**
* Helpers
**/

/**
* Generates a prompt string based on the checklists under a specific planId and any additional prompt information provided.
**/
func (s *service) generatePromptWithChecklist(ctx context.Context, planId uuid.UUID, additionalPrompt string) (string, error) {
	plan, err := s.planService.GetById(ctx, planId)
	if err != nil {
		fmt.Println("Error when retrieving plan for generating checklist suggestion:", err)
		return "", err
	}

	// get entire checklist as context
	checklistItems, err := s.checklistService.GetAllByPlanId(ctx, planId, nil)

	if err != nil {
		fmt.Println("Error when retrieving all checklist item for generating checklist suggestion.")
		return "", err
	}

	focus := plan.Focus
	checklistPrompt := ""

	// construct the prompt context
	for _, item := range checklistItems {
		checklistPrompt += fmt.Sprintf("A %s task: %s\n", item.Scope, item.Description)
	}

	// focus - the primary topic input by the user for their plan.
	prompt := fmt.Sprintf(`Based on this project focus: "%s"
		%s
		So far the checklist already has these items, so either add one to follow the current progress or don't suggest one that's already present.
		This is the current existing checklist:
		%s
		%s
		`, focus, s.basePrompt, checklistPrompt, additionalPrompt)

	fmt.Printf("\nfinal prompt was: \n%s\n\n", prompt)

	return prompt, nil
}
