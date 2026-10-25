package ai

import (
	"context"
	"errors"
	"time"

	"github.com/sashabaranov/go-openai"
)

// Generator is the OpenAI-backed implementation of insights.ContentGenerator.
// It pairs a fixed system prompt with a chat client; callers supply only the
// user message, so a generator IS its system prompt (one per generation kind).
//
// Ported from the api-gateway's internal/ai during the strangler move of the
// insights domain. Its requests now go through the shared chatCaller, so each
// is checked against the global token budget and recorded in the usage ledger
// under the caller's Call (FS-02JBZ R31–R33).
type Generator struct {
	caller       chatCaller
	systemPrompt string
}

// NewGenerator builds a Generator for one system prompt, metered by meter.
//
// NOTE:
// Secret Management: In production, use a secrets manager like HashiCorp Vault, AWS Secrets Manager, or GCP Secret Manager
func NewGenerator(systemPrompt string, clientKey string, meter *Meter) *Generator {
	return newGenerator(systemPrompt, openai.NewClient(clientKey), meter, time.Second)
}

func newGenerator(systemPrompt string, client *openai.Client, meter *Meter, retryDelay time.Duration) *Generator {
	return &Generator{caller: newChatCaller(client, meter, retryDelay), systemPrompt: systemPrompt}
}

// Generate returns the model's answer to message, on behalf of call.
func (g *Generator) Generate(ctx context.Context, call Call, message string) (string, error) {
	resp, err := g.caller.complete(ctx, call, openai.ChatCompletionRequest{
		Model:           modelName(),
		ReasoningEffort: "minimal",
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: g.systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: message},
		},
		MaxCompletionTokens: 200,
	}, func(resp openai.ChatCompletionResponse) error {
		if len(resp.Choices) == 0 {
			return errors.New("ai: chat completion returned no choices")
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}
