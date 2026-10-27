package rag

import (
	"context"
	"errors"
	"testing"

	"github.com/sashabaranov/go-openai"
)

// fakeEmbeddingsClient stands in for the OpenAI client: tests never call the
// live embeddings API.
type fakeEmbeddingsClient struct {
	got  []openai.EmbeddingRequest
	resp openai.EmbeddingResponse
	err  error
}

func (f *fakeEmbeddingsClient) CreateEmbeddings(_ context.Context, conv openai.EmbeddingRequestConverter) (openai.EmbeddingResponse, error) {
	f.got = append(f.got, conv.Convert())
	return f.resp, f.err
}

func responseWith(vectors ...[]float32) openai.EmbeddingResponse {
	resp := openai.EmbeddingResponse{Model: openai.SmallEmbedding3}
	for i, v := range vectors {
		resp.Data = append(resp.Data, openai.Embedding{Embedding: v, Index: i})
	}
	return resp
}

func TestEmbed_RequestsSmall3At512AndStampsTheResult(t *testing.T) {
	vec := make([]float32, 512)
	vec[7] = 0.5
	client := &fakeEmbeddingsClient{resp: responseWith(vec)}
	e := newEmbedder(client)

	got, err := e.Embed(context.Background(), "ship the landing page")
	if err != nil {
		t.Fatalf("embed: %v", err)
	}

	if len(client.got) != 1 {
		t.Fatalf("made %d embedding calls, want 1", len(client.got))
	}
	req := client.got[0]
	if req.Model != openai.SmallEmbedding3 || req.Dimensions != 512 {
		t.Errorf("requested %s at %d dims, want text-embedding-3-small at 512", req.Model, req.Dimensions)
	}
	if in, ok := req.Input.([]string); !ok || len(in) != 1 || in[0] != "ship the landing page" {
		t.Errorf("request input = %#v, want the one text", req.Input)
	}

	if got.Model != "text-embedding-3-small" || got.Dims != 512 {
		t.Errorf("stamped %s/%d, want text-embedding-3-small/512", got.Model, got.Dims)
	}
	if len(got.Vector) != 512 || got.Vector[7] != 0.5 {
		t.Errorf("vector not carried through (len %d)", len(got.Vector))
	}
}

func TestEmbed_RefusesAnythingItCannotStampTruthfully(t *testing.T) {
	apiDown := errors.New("connection refused")
	tests := []struct {
		name string
		resp openai.EmbeddingResponse
		err  error
	}{
		{"api error", openai.EmbeddingResponse{}, apiDown},
		{"no data", responseWith(), nil},
		{"wrong dimension count", responseWith(make([]float32, 1536)), nil},
		{"more than one vector", responseWith(make([]float32, 512), make([]float32, 512)), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEmbedder(&fakeEmbeddingsClient{resp: tt.resp, err: tt.err})
			got, err := e.Embed(context.Background(), "text")
			if err == nil {
				t.Fatalf("want an error, got embedding %s/%d", got.Model, got.Dims)
			}
			if tt.err != nil && !errors.Is(err, tt.err) {
				t.Errorf("error %v does not wrap the client error", err)
			}
		})
	}
}

func TestEmbed_RefusesEmptyTextWithoutCallingTheAPI(t *testing.T) {
	client := &fakeEmbeddingsClient{resp: responseWith(make([]float32, 512))}
	e := newEmbedder(client)

	if _, err := e.Embed(context.Background(), "   "); err == nil {
		t.Fatal("want an error for blank text")
	}
	if len(client.got) != 0 {
		t.Errorf("made %d embedding calls for blank text, want 0", len(client.got))
	}
}
