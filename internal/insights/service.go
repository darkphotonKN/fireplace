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
	// TODO: update to retrieve from database from plan
	focus := "Making a react project for a web app where people can share ideas and stay productive."

	// focus - the primary topic input by the user for their plan.
	prompt := fmt.Sprintf("The primary focus of this todo list is: %s.\nPlease provide a single suggestion for what I could put on my todo that would work towards this focus.", focus)

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
