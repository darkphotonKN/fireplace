package ai

import (
	"context"
	"time"

	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/interfaces"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/logger"
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
				Model: openai.GPT4o,
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
				Temperature: 0.7,
				MaxTokens:   200,
			},
		)

		// retry chat completion gen after a short delay
		if err != nil {
			logger.Error("Error attempting chat completion", "error", err, "attempt", attempt+1)
			time.Sleep(g.retryDelay)
			continue
		}

		break
	}

	if err != nil {
		logger.Error("ChatCompletion failed after retries", "error", err)
		return "", err
	}

	logger.Debug("Generated AI content", "content", resp.Choices[0].Message.Content)

	return resp.Choices[0].Message.Content, nil
}
