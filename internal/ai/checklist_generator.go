package ai

import (
	"os"

	"github.com/darkphotonKN/fireplace/internal/interfaces"
)

type ChecklistGen struct {
	*Generator
}

const (
	checklistSystemPrompt string = `You are an AI assistant for the Fireplace productivity platform. Your purpose is to help users maintain focus, organize their tasks, and make progress on their learning and work projects.

Always provide concise, practical, and actionable responses. Your suggestions should be specific and tailored to the user's stated focus. When generating checklist items, each item should be concrete and implementable.

For plan summaries, identify the core objectives and key components. For checklist suggestions, recommend the next logical step to move the project forward.

Keep responses under 5 sentences unless detailed instructions are specifically requested.`
)

func NewChecklistGen() interfaces.ContentGenerator {
	// NOTE:
	// Secret Management: In production, use a secrets manager like HashiCorp Vault, AWS Secrets Manager, or GCP Secret Manager
	generator := NewGenerator(checklistSystemPrompt, os.Getenv("OPENAI_API_KEY"))

	return &ChecklistGen{
		Generator: generator.(*Generator),
	}
}
