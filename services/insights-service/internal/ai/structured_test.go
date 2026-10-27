package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/sashabaranov/go-openai"
)

// These tests run against a local fake of the chat completions endpoint. They
// never reach OpenAI.

func fakeOpenAI(t *testing.T, handler func(w http.ResponseWriter, body map[string]any)) *openai.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Errorf("request body is not JSON: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		handler(w, body)
	}))
	t.Cleanup(srv.Close)
	cfg := openai.DefaultConfig("test-key")
	cfg.BaseURL = srv.URL + "/v1"
	return openai.NewClientWithConfig(cfg)
}

const okResponse = `{
  "id": "x", "object": "chat.completion", "model": "gpt-5-mini-2025-08-07",
  "choices": [{"index": 0, "finish_reason": "stop", "message": {"role": "assistant", "content": "{\"moves\":[]}"}}],
  "usage": {"prompt_tokens": 1200, "completion_tokens": 90, "total_tokens": 1290, "prompt_tokens_details": {"cached_tokens": 1024}}
}`

func TestStructuredGenerator_SendsOneStrictSchemaCallWithItsConfig(t *testing.T) {
	var calls atomic.Int32
	var got map[string]any
	client := fakeOpenAI(t, func(w http.ResponseWriter, body map[string]any) {
		calls.Add(1)
		got = body
		io.WriteString(w, okResponse)
	})
	cfg := GeneratorConfig{Model: "gpt-5-mini", MaxOutputTokens: 321, ReasoningEffort: "minimal"}
	gen := newStructuredGenerator(cfg, client, openMeter(), 0)

	out, err := gen.Generate(context.Background(), StructuredRequest{
		System: "rules", User: "tree", SchemaName: "daily_brief",
		Schema: json.RawMessage(`{"type":"object"}`),
	})

	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}
	if got["model"] != "gpt-5-mini" || got["reasoning_effort"] != "minimal" || got["max_completion_tokens"] != float64(321) {
		t.Fatalf("request config = model %v effort %v max %v", got["model"], got["reasoning_effort"], got["max_completion_tokens"])
	}
	rf, _ := got["response_format"].(map[string]any)
	schema, _ := rf["json_schema"].(map[string]any)
	if rf["type"] != "json_schema" || schema["strict"] != true || schema["name"] != "daily_brief" {
		t.Fatalf("response_format = %v, want strict json_schema", rf)
	}
	msgs, _ := got["messages"].([]any)
	if len(msgs) != 2 || msgs[0].(map[string]any)["role"] != "system" || msgs[1].(map[string]any)["content"] != "tree" {
		t.Fatalf("messages = %v, want system then user", msgs)
	}
	if out.Content != `{"moves":[]}` || out.Model != "gpt-5-mini-2025-08-07" {
		t.Fatalf("completion = %+v", out)
	}
	if out.Usage != (Usage{InputTokens: 1200, CachedInputTokens: 1024, OutputTokens: 90}) {
		t.Fatalf("usage = %+v", out.Usage)
	}
}

func TestStructuredGenerator_TruncatedOutputFailsWithUsage(t *testing.T) {
	client := fakeOpenAI(t, func(w http.ResponseWriter, _ map[string]any) {
		io.WriteString(w, `{"model":"gpt-5-mini","choices":[{"finish_reason":"length","message":{"role":"assistant","content":"{\"mo"}}],
		  "usage":{"prompt_tokens":10,"completion_tokens":800}}`)
	})
	gen := newStructuredGenerator(GeneratorConfig{Model: "gpt-5-mini"}, client, openMeter(), 0)

	out, err := gen.Generate(context.Background(), StructuredRequest{Schema: json.RawMessage(`{}`)})

	if err == nil {
		t.Fatal("want an error for truncated output")
	}
	if out.Usage.OutputTokens != 800 || out.Model != "gpt-5-mini" {
		t.Fatalf("completion = %+v, want usage reported on failure", out)
	}
}

func TestStructuredGenerator_TransportFailureRetriesThenFails(t *testing.T) {
	var calls atomic.Int32
	client := fakeOpenAI(t, func(w http.ResponseWriter, _ map[string]any) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
		io.WriteString(w, `{"error":{"message":"down"}}`)
	})
	gen := newStructuredGenerator(GeneratorConfig{Model: "m"}, client, openMeter(), 0)

	out, err := gen.Generate(context.Background(), StructuredRequest{Schema: json.RawMessage(`{}`)})

	if err == nil {
		t.Fatal("want an error")
	}
	if calls.Load() != 3 {
		t.Fatalf("calls = %d, want 3 transport attempts", calls.Load())
	}
	if out.Model != "m" || out.Usage != (Usage{}) {
		t.Fatalf("completion = %+v, want configured model and no usage", out)
	}
}

func TestDayPlanConfig_DefaultsAndOwnOverride(t *testing.T) {
	t.Setenv("DAY_PLAN_MODEL", "")
	t.Setenv("DAY_PLAN_MAX_OUTPUT_TOKENS", "")
	t.Setenv("DAY_PLAN_REASONING_EFFORT", "")
	t.Setenv("OPENAI_MODEL", "some-other-model")

	if got := DayPlanConfig(); got.Model != "gpt-5-mini" || got.ReasoningEffort != "minimal" || got.MaxOutputTokens <= 0 {
		t.Fatalf("default config = %+v, want gpt-5-mini, minimal, a ceiling", got)
	}

	t.Setenv("DAY_PLAN_MODEL", "gpt-5")
	t.Setenv("DAY_PLAN_MAX_OUTPUT_TOKENS", "2000")
	if got := DayPlanConfig(); got.Model != "gpt-5" || got.MaxOutputTokens != 2000 {
		t.Fatalf("overridden config = %+v", got)
	}
}
