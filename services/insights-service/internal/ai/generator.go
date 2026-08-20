package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/sashabaranov/go-openai"
)

// Generator is the OpenAI-backed implementation of insights.ContentGenerator.
// It pairs a fixed system prompt with a chat client; callers supply only the
// user message, so a generator IS its system prompt (one per generation kind).
//
// Ported verbatim from the api-gateway's internal/ai during the strangler move
// of the insights domain — behaviour is deliberately unchanged.
type Generator struct {
	client       *openai.Client
	systemPrompt string

	maxRetries int
	retryDelay time.Duration
}

// NewGenerator builds a Generator for one system prompt.
//
// NOTE:
// Secret Management: In production, use a secrets manager like HashiCorp Vault, AWS Secrets Manager, or GCP Secret Manager
func NewGenerator(systemPrompt string, clientKey string) *Generator {
	client := openai.NewClient(clientKey)

	return &Generator{
		client:       client,
		systemPrompt: systemPrompt,

		maxRetries: 3,
		retryDelay: time.Second,
	}
}

func (g *Generator) Generate(message string) (string, error) {
	var resp openai.ChatCompletionResponse
	var err error

	// retry on error
	for attempt := 0; attempt < g.maxRetries; attempt++ {
		resp, err = g.client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model:           modelName(),
				ReasoningEffort: "minimal",
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleSystem,
						Content: g.systemPrompt,
					},
					{
						Role:    openai.ChatMessageRoleUser,
						Content: message,
					},
				},
				MaxCompletionTokens: 200,
			},
		)

		// retry chat completion gen after a short delay
		if err != nil {
			time.Sleep(g.retryDelay)
			continue
		}

		break
	}

	if err != nil {
		return "", fmt.Errorf("ai: chat completion failed after %d attempts: %w", g.maxRetries, err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("ai: chat completion returned no choices")
	}

	return resp.Choices[0].Message.Content, nil
}
