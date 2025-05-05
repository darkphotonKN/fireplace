package insights

import (
	"fmt"

	"github.com/darkphotonKN/fireplace/internal/interfaces"
)

type service struct {
	repo       Repository
	contentGen interfaces.ContentGenerator
}

type Repository interface {
}

func NewService(repo Repository, contentGen interfaces.ContentGenerator) Service {
	return &service{
		repo:       repo,
		contentGen: contentGen,
	}
}

/**
* Generates the correct checklist item suggestion with some the context of user's focus and current checklist items.
**/
func (s *service) GenerateChecklistSuggestion() (string, error) {
	// TODO: update to retrieve from plans table
	focus := "Making a react project for a web app where people can share ideas and stay productive."

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
    
    Format your response as a single task item with no additional commentary, explanation or punctuation at the end.`, focus)

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
