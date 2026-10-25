package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/sashabaranov/go-openai"
)

// chatCaller is the one path from this service to the chat completions API.
// Both generators make their requests through it, so the budget check and the
// usage record (Meter) cannot be skipped by a generator, present or future.
type chatCaller struct {
	client     *openai.Client
	meter      *Meter
	maxRetries int
	retryDelay time.Duration

	// afterAttempt, when set, runs after each attempt is recorded. Test-only:
	// it lets a test change the ledger between attempts.
	afterAttempt func()
}

func newChatCaller(client *openai.Client, meter *Meter, retryDelay time.Duration) chatCaller {
	return chatCaller{client: client, meter: meter, maxRetries: 3, retryDelay: retryDelay}
}

// complete makes req on behalf of call. A transport failure, a throttle (429)
// or a provider error (5xx) is retried, up to maxRetries attempts in all; any
// other error status (a bad request, a bad key, an unknown model) fails at
// once, since no retry can change the answer. Before every attempt the meter is asked, so an
// over-budget call, or retry, is never sent (ErrBudgetExhausted). After every
// attempt its usage is recorded, whatever the outcome (FS-02JBZ R31: each
// transport retry records what the provider reported).
//
// accept judges a response that arrived: its error fails the call with no
// retry, since the provider answered, and the attempt is recorded as failed.
func (c chatCaller) complete(ctx context.Context, call Call, req openai.ChatCompletionRequest, accept func(openai.ChatCompletionResponse) error) (openai.ChatCompletionResponse, error) {
	var resp openai.ChatCompletionResponse
	var err error
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return resp, fmt.Errorf("ai: chat completion failed after %d attempts: %w", attempt, err)
			case <-time.After(c.retryDelay):
			}
		}
		if allowErr := c.meter.Allow(ctx); allowErr != nil {
			return resp, allowErr
		}

		resp, err = c.client.CreateChatCompletion(ctx, req)
		if err != nil {
			c.record(ctx, Attempt{Call: call, Model: req.Model, Succeeded: false})
			if ctx.Err() != nil || !retryable(err) {
				return resp, fmt.Errorf("ai: chat completion failed: %w", err)
			}
			continue
		}

		acceptErr := accept(resp)
		model := req.Model
		if resp.Model != "" {
			model = resp.Model
		}
		c.record(ctx, Attempt{Call: call, Model: model, Usage: usageOf(resp), Succeeded: acceptErr == nil})
		return resp, acceptErr
	}
	return resp, fmt.Errorf("ai: chat completion failed after %d attempts: %w", c.maxRetries, err)
}

// retryable reports whether a failed request may succeed if sent again: a
// transport failure (no HTTP status), a timeout or throttle (408, 429) or a
// provider error (5xx). Every other status is the provider's final answer.
func retryable(err error) bool {
	status := 0
	var apiErr *openai.APIError
	var reqErr *openai.RequestError
	switch {
	case errors.As(err, &apiErr):
		status = apiErr.HTTPStatusCode
	case errors.As(err, &reqErr):
		status = reqErr.HTTPStatusCode
	}
	switch {
	case status == 0:
		return true
	case status == http.StatusRequestTimeout, status == http.StatusTooManyRequests:
		return true
	default:
		return status >= http.StatusInternalServerError
	}
}

func (c chatCaller) record(ctx context.Context, a Attempt) {
	c.meter.Record(ctx, a)
	if c.afterAttempt != nil {
		c.afterAttempt()
	}
}

// usageOf is the token usage resp reports (FS-DF4ZB R9).
func usageOf(resp openai.ChatCompletionResponse) Usage {
	u := Usage{InputTokens: resp.Usage.PromptTokens, OutputTokens: resp.Usage.CompletionTokens}
	if d := resp.Usage.PromptTokensDetails; d != nil {
		u.CachedInputTokens = d.CachedTokens
	}
	return u
}
