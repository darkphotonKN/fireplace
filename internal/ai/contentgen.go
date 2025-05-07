package ai

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/darkphotonKN/fireplace/internal/interfaces"
	"github.com/sashabaranov/go-openai"
)

type ContentGen struct {
	client       *openai.Client
	systemPrompt string
}

var (
	ErrAuthentication     = errors.New("authentication error")
	ErrRateLimit          = errors.New("rate limit exceeded")
	ErrContextLength      = errors.New("context length exceeded")
	ErrServiceUnavailable = errors.New("service unavailable")
)

const (
	maxRetries   int           = 3
	retryDelay   time.Duration = time.Second
	systemPrompt string        = `You are an AI assistant for the Fireplace productivity platform. Your purpose is to help users maintain focus, organize their tasks, and make progress on their learning and work projects.

Always provide concise, practical, and actionable responses. Your suggestions should be specific and tailored to the user's stated focus. When generating checklist items, each item should be concrete and implementable.

For plan summaries, identify the core objectives and key components. For checklist suggestions, recommend the next logical step to move the project forward.

Keep responses under 5 sentences unless detailed instructions are specifically requested.`
)

func NewContentGen() interfaces.ContentGenerator {
	// NOTE:
	// Secret Management: In production, use a secrets manager like HashiCorp Vault, AWS Secrets Manager, or GCP Secret Manager
	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

	return &ContentGen{
		client:       client,
		systemPrompt: systemPrompt,
	}
}

func (g *ContentGen) ChatCompletion(message string) (string, error) {
	var resp openai.ChatCompletionResponse
	var err error

	// retry on error
	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err = g.client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model: openai.GPT4o,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleSystem,
						Content: systemPrompt,
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
			fmt.Printf("Error occured while attempting chat completion: %v\n", err)
			time.Sleep(retryDelay)
			continue
		}

		break
	}

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return "", err
	}

	fmt.Printf("\nresult generated content: %+v\n\n", resp.Choices[0].Message.Content)
	return resp.Choices[0].Message.Content, nil
}
