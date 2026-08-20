package ai

import "os"

// defaultModel is the chat model every generator uses unless OPENAI_MODEL overrides it.
//
// gpt-5-mini sits in OpenAI's mini/nano complimentary-token bucket, so routine
// generation draws on the free daily allowance rather than billed credit. Overrides
// are expected to stay in the gpt-5 family: these models reject `max_tokens` and any
// `temperature` other than the default, and the requests below are built to match.
const defaultModel = "gpt-5-mini"

// modelName resolves the chat model, letting a deployment retarget it without a rebuild.
func modelName() string {
	if model := os.Getenv("OPENAI_MODEL"); model != "" {
		return model
	}
	return defaultModel
}
