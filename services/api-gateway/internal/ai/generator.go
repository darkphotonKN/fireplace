package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/interfaces"
	"github.com/sashabaranov/go-openai"
)

type Generator struct {
	client       *openai.Client
	systemPrompt string

	maxRetries int
	retryDelay time.Duration
}

func NewGenerator(systemPrompt string, clientKey string) interfaces.ContentGenerator {
	// NOTE:
	// Secret Management: In production, use a secrets manager like HashiCorp Vault, AWS Secrets Manager, or GCP Secret Manager
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

	return resp.Choices[0].Message.Content, nil
}
