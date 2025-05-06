package insights

import (
	"context"
	"fmt"

	"github.com/darkphotonKN/fireplace/internal/checklistitems"
	"github.com/darkphotonKN/fireplace/internal/interfaces"
)

type service struct {
	repo             Repository
	contentGen       interfaces.ContentGenerator
	checklistService checklistitems.Service
}

type Repository interface {
}

func NewService(repo Repository, contentGen interfaces.ContentGenerator, checklistService checklistitems.Service) Service {
	return &service{
		repo:             repo,
		contentGen:       contentGen,
		checklistService: checklistService,
	}
}

/**
* Generates the correct checklist item suggestion with some the context of user's focus and current checklist items.
**/
func (s *service) GenerateChecklistSuggestion(ctx context.Context) (string, error) {
	// TODO: update to retrieve from plans table
	focus := "Making a react project for a web app where people can share ideas and stay productive."

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

    Please suggest ONE specific, actionable task th20 would be the most valuable next step to add to my checklist.

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
		`, focus, checklistPrompt)

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
