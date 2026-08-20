package ai

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// NotesGenerator handles AI-powered note generation
type NotesGenerator struct {
	client *openai.Client
}

// NewNotesGenerator creates a new AI notes generator
func NewNotesGenerator() *NotesGenerator {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Println("Warning: OPENAI_API_KEY not set. AI note generation will be limited.")
		return &NotesGenerator{}
	}

	client := openai.NewClient(apiKey)
	return &NotesGenerator{client: client}
}

// GenerateContent generates AI content for notes based on context
func (ng *NotesGenerator) GenerateContent(contextStr string) (string, error) {
	if ng.client == nil {
		return "", fmt.Errorf("OpenAI client not initialized")
	}

	prompt := fmt.Sprintf(`You are an intelligent productivity assistant helping users manage their tasks and projects.
Based on the following context, generate a helpful, actionable note:

%s

Provide a concise, practical suggestion or insight that helps the user improve their productivity.
Use emojis sparingly for visual clarity. Keep the response under 150 words.`, contextStr)

	resp, err := ng.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:           modelName(),
			ReasoningEffort: "minimal",
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are a helpful productivity assistant that provides actionable insights and suggestions.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			MaxCompletionTokens: 200,
		},
	)

	if err != nil {
		return "", fmt.Errorf("failed to generate AI content: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// GenerateWarning generates a warning note based on task analysis
func (ng *NotesGenerator) GenerateWarning(tasksContext string) (string, error) {
	if ng.client == nil {
		return "", fmt.Errorf("OpenAI client not initialized")
	}

	prompt := fmt.Sprintf(`Analyze the following task information and generate a warning or alert if needed:

%s

If there are any concerns (overdue tasks, too many pending items, scheduling conflicts, etc.),
generate a warning message. If everything looks good, provide encouragement.
Start with an appropriate emoji. Keep it under 100 words.`, tasksContext)

	resp, err := ng.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:           modelName(),
			ReasoningEffort: "minimal",
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are a task management assistant focused on identifying potential issues and providing warnings.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			MaxCompletionTokens: 150,
		},
	)

	if err != nil {
		return "", fmt.Errorf("failed to generate warning: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// GenerateInsight generates an insight note based on progress analysis
func (ng *NotesGenerator) GenerateInsight(progressContext string) (string, error) {
	if ng.client == nil {
		return "", fmt.Errorf("OpenAI client not initialized")
	}

	prompt := fmt.Sprintf(`Analyze the following progress information and generate an insightful observation:

%s

Provide a data-driven insight about the user's progress, patterns, or opportunities for improvement.
Be encouraging but honest. Start with an appropriate emoji. Keep it under 100 words.`, progressContext)

	resp, err := ng.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:           modelName(),
			ReasoningEffort: "minimal",
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are an analytical assistant that provides insights based on task completion patterns and progress.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			MaxCompletionTokens: 150,
		},
	)

	if err != nil {
		return "", fmt.Errorf("failed to generate insight: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// GenerateSuggestion generates a suggestion note for next actions
func (ng *NotesGenerator) GenerateSuggestion(planContext string) (string, error) {
	if ng.client == nil {
		return "", fmt.Errorf("OpenAI client not initialized")
	}

	prompt := fmt.Sprintf(`Based on the following plan and task information, suggest the next best actions:

%s

Provide 1-2 specific, actionable suggestions for what the user should focus on next.
Be practical and specific. Start with an appropriate emoji. Keep it under 120 words.`, planContext)

	resp, err := ng.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:           modelName(),
			ReasoningEffort: "minimal",
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are a strategic planning assistant that suggests next actions based on current progress and goals.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			MaxCompletionTokens: 150,
		},
	)

	if err != nil {
		return "", fmt.Errorf("failed to generate suggestion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}
