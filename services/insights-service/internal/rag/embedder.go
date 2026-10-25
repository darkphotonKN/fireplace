package rag

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
)

// The one model and dimension count (R34, ADR-0018 §4). Constants, not
// configuration: rag_chunks.embedding is typed vector(512), and a model change
// is a re-embed migration, never an env edit.
const (
	EmbeddingModel = string(openai.SmallEmbedding3)
	EmbeddingDims  = 512
)

// ErrEmptyText is returned for blank input, which is never sent to the API.
var ErrEmptyText = errors.New("rag: nothing to embed")

// embeddingsClient is the slice of the OpenAI client the embedder uses, so tests
// substitute a fake and never call the live API.
type embeddingsClient interface {
	CreateEmbeddings(ctx context.Context, conv openai.EmbeddingRequestConverter) (openai.EmbeddingResponse, error)
}

// Embedder turns text into a stamped Embedding. Ingest and query both go
// through it, so the two cannot disagree about model or dimensions: every
// Embedding it returns records text-embedding-3-small and 512, and it refuses a
// response it could not stamp truthfully rather than mislabel it.
//
// Embedding calls are not generation: they are not metered, recorded in the
// usage ledger or counted against the daily token budget (FS-02JBZ S8).
type Embedder struct {
	client embeddingsClient
}

// NewEmbedder builds the OpenAI-backed embedder.
func NewEmbedder(apiKey string) *Embedder {
	return newEmbedder(openai.NewClient(apiKey))
}

func newEmbedder(client embeddingsClient) *Embedder {
	return &Embedder{client: client}
}

// Embed returns the embedding of one text. It makes exactly one API call and
// does not retry: callers decide (ingest requeues; a request falls back to weak
// retrieval).
func (e *Embedder) Embed(ctx context.Context, text string) (Embedding, error) {
	if strings.TrimSpace(text) == "" {
		return Embedding{}, ErrEmptyText
	}

	resp, err := e.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input:      []string{text},
		Model:      openai.SmallEmbedding3,
		Dimensions: EmbeddingDims,
	})
	if err != nil {
		return Embedding{}, fmt.Errorf("rag: embed: %w", err)
	}
	if len(resp.Data) != 1 {
		return Embedding{}, fmt.Errorf("rag: embed: got %d vectors for one text", len(resp.Data))
	}
	vec := resp.Data[0].Embedding
	if len(vec) != EmbeddingDims {
		return Embedding{}, fmt.Errorf("rag: embed: got %d dimensions, want %d", len(vec), EmbeddingDims)
	}

	return Embedding{Vector: vec, Model: EmbeddingModel, Dims: EmbeddingDims}, nil
}
