package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/sashabaranov/go-openai"
)

// GeneratorConfig is one structured generator's model settings, configurable per
// generator (FS-DF4ZB R8, FS-02JBZ R6): the day plan has its own, as step
// suggestions and the item tree will.
type GeneratorConfig struct {
	Model           string
	MaxOutputTokens int
	ReasoningEffort string
}

// DayPlanConfig is the day plan generator's config (FS-02JBZ R6): gpt-5-mini
// with minimal reasoning unless DAY_PLAN_MODEL, DAY_PLAN_MAX_OUTPUT_TOKENS or
// DAY_PLAN_REASONING_EFFORT say otherwise.
//
// The default ceiling sizes six entries (two ids, a step's text, a reason and
// minutes each) under up to three labels, plus JSON overhead and minimal
// reasoning. Reasoning tokens count against the same ceiling, so it is not set
// to the bare output size: a truncated answer is invalid JSON and therefore a
// failed generation.
func DayPlanConfig() GeneratorConfig {
	return configFromEnv("DAY_PLAN", GeneratorConfig{
		Model:           defaultModel,
		MaxOutputTokens: 1600,
		ReasoningEffort: "minimal",
	})
}

// configFromEnv overrides defaults with <PREFIX>_MODEL,
// <PREFIX>_MAX_OUTPUT_TOKENS and <PREFIX>_REASONING_EFFORT when set. An
// unparsable or non-positive token ceiling keeps the default.
func configFromEnv(prefix string, defaults GeneratorConfig) GeneratorConfig {
	cfg := defaults
	if v := os.Getenv(prefix + "_MODEL"); v != "" {
		cfg.Model = v
	}
	if v := os.Getenv(prefix + "_MAX_OUTPUT_TOKENS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxOutputTokens = n
		}
	}
	if v := os.Getenv(prefix + "_REASONING_EFFORT"); v != "" {
		cfg.ReasoningEffort = v
	}
	return cfg
}

// StructuredRequest is one structured-output call: what it is for (Call), the
// system message, the user message, and the strict JSON schema the answer must
// fit. Accept, when set, judges the content (typically by parsing it against
// the same schema): its error fails the generation and the attempt is recorded
// as failed, so the ledger's succeeded flag means "usable output".
type StructuredRequest struct {
	Call       Call
	System     string
	User       string
	SchemaName string
	Schema     json.RawMessage
	Accept     func(content string) error
}

// Usage is the token usage the API reported for one call (FS-DF4ZB R9).
type Usage struct {
	InputTokens       int
	CachedInputTokens int
	OutputTokens      int
}

// Completion is the result of one structured call. Model and Usage are filled
// even when Generate also returns an error, as far as the API reported them.
// The usage is already recorded in the ledger by the time Generate returns.
type Completion struct {
	Content string
	Model   string
	Usage   Usage
}

// StructuredGenerator makes one strict JSON-schema structured-output call per
// Generate (FS-DF4ZB R7). It never repairs the content; judging it is the
// caller's job, through StructuredRequest.Accept. Its requests go through the
// shared chatCaller, so each is checked against the global token budget and
// recorded in the usage ledger (FS-02JBZ R31–R33).
type StructuredGenerator struct {
	caller chatCaller
	cfg    GeneratorConfig
}

// NewStructuredGenerator builds a generator for one config, metered by meter.
func NewStructuredGenerator(cfg GeneratorConfig, apiKey string, meter *Meter) *StructuredGenerator {
	return newStructuredGenerator(cfg, openai.NewClient(apiKey), meter, time.Second)
}

func newStructuredGenerator(cfg GeneratorConfig, client *openai.Client, meter *Meter, retryDelay time.Duration) *StructuredGenerator {
	return &StructuredGenerator{caller: newChatCaller(client, meter, retryDelay), cfg: cfg}
}

// Generate makes the call. A transport failure is retried (transport retries,
// not extra generations, though each is recorded); content that comes back
// refused, truncated or rejected by Accept is an error with usage still
// reported. Over budget, no request is sent and the error is
// ErrBudgetExhausted.
func (g *StructuredGenerator) Generate(ctx context.Context, req StructuredRequest) (Completion, error) {
	out := Completion{Model: g.cfg.Model}
	chatReq := openai.ChatCompletionRequest{
		Model:               g.cfg.Model,
		ReasoningEffort:     g.cfg.ReasoningEffort,
		MaxCompletionTokens: g.cfg.MaxOutputTokens,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: req.System},
			{Role: openai.ChatMessageRoleUser, Content: req.User},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
				Name:   req.SchemaName,
				Schema: req.Schema,
				Strict: true,
			},
		},
	}

	resp, err := g.caller.complete(ctx, req.Call, chatReq, func(resp openai.ChatCompletionResponse) error {
		if resp.Model != "" {
			out.Model = resp.Model
		}
		out.Usage = usageOf(resp)
		if len(resp.Choices) == 0 {
			return errors.New("ai: structured completion returned no choices")
		}
		choice := resp.Choices[0]
		if choice.Message.Refusal != "" {
			return fmt.Errorf("ai: structured completion refused: %s", choice.Message.Refusal)
		}
		if choice.FinishReason == openai.FinishReasonLength {
			return errors.New("ai: structured completion hit the output-token ceiling")
		}
		if req.Accept != nil {
			return req.Accept(choice.Message.Content)
		}
		return nil
	})
	if err != nil {
		return out, err
	}
	out.Content = resp.Choices[0].Message.Content
	return out, nil
}
