package ai

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/darkphotonKN/fireplace/internal/insights"
	"github.com/sashabaranov/go-openai"
)

type ContentGen struct {
	client *openai.Client
}

var (
	ErrAuthentication     = errors.New("authentication error")
	ErrRateLimit          = errors.New("rate limit exceeded")
	ErrContextLength      = errors.New("context length exceeded")
	ErrServiceUnavailable = errors.New("service unavailable")
)

const (
	maxRetries int           = 3
	retryDelay time.Duration = time.Second
)

func NewContentGen() insights.ContentGenAI {
	// NOTE:
	// Secret Management: In production, use a secrets manager like HashiCorp Vault, AWS Secrets Manager, or GCP Secret Manager
	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	return &ContentGen{
		client: client,
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
						Content: "You are a helpful assistant.",
					},
					{
						Role:    openai.ChatMessageRoleUser,
						Content: "Hello, can you tell me about integrating LLMs with Go?",
					},
				},
			},
		)

		// retry chat completion gen
		if err != nil {
			fmt.Printf("Error occured while attempting chat completion: %v\n", err)
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
