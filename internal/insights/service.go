package insights

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/darkphotonKN/fireplace/internal/checklistitems"
	"github.com/darkphotonKN/fireplace/internal/interfaces"
	"github.com/darkphotonKN/fireplace/internal/plans"
)

type service struct {
	repo             Repository
	contentGen       interfaces.ContentGenerator
	checklistService checklistitems.Service
	planService      plans.Service
}

type Repository interface {
}

func NewService(repo Repository, contentGen interfaces.ContentGenerator, checklistService checklistitems.Service, planService plans.Service) Service {
	return &service{
		repo:             repo,
		contentGen:       contentGen,
		checklistService: checklistService,
		planService:      planService,
	}
}

/**
* Generates the correct checklist item suggestion with some the context of user's focus and current checklist items.
**/
func (s *service) GenerateChecklistSuggestion(ctx context.Context) (string, error) {
	// TODO: For now, using a static plan ID for development - this should be passed in or retrieved from context
	planID, err := uuid.Parse("22222222-2222-2222-2222-222222222222") // Test plan ID
	if err != nil {
		return "", err
	}
	
	plan, err := s.planService.GetById(ctx, planID)
	if err != nil {
		fmt.Println("Error when retrieving plan for generating checklist suggestion:", err)
		return "", err
	}
	
	// Use the plan's focus
	planFocus := plan.Focus

	// get entire checklist as context
	checklistItems, err := s.checklistService.GetAll(ctx)

	if err != nil {
		fmt.Println("Error when retrieving all checklist item for generating checklist suggestion.")
		return "", err
	}

	checklistPrompt := ""

	// construct the prompt context
	for _, item := range checklistItems {
		checklistPrompt += fmt.Sprintf("%s\n", item.Description)
	}

	fmt.Printf("constructed checklist item prompt: %s\n", checklistPrompt)

	// focus - the primary topic input by the user for their plan.
	prompt := fmt.Sprintf(`Based on this project focus: "%s"

    Please suggest ONE specific, actionable task that would be the most valuable next step to add to my checklist.

    Your suggestion should:
    - Be a single, concrete task (not multiple tasks)
    - Start with a verb
    - Be specific enough to complete in a single sitting
    - Be directly relevant to the project focus
    - Use technical terminology accurately if applicable
    - Be 4-20 words in length
    
    Format your response as a single task item with no additional commentary, explanation or punctuation at the end.

		So far the checklist already has these items, so either add one to follow the current progress or don't suggest one that's already present:

		%s
		`, planFocus, checklistPrompt)

	fmt.Printf("\nprompt was: \n%s\n\n", prompt)

	res, err := s.contentGen.ChatCompletion(prompt)

	if err != nil {
		return "", err
	}

	// TODO: checklist - the list of current checklist items for context
	return res, nil
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
