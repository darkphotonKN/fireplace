package guided

import (
	"context"
	"log/slog"
	"strings"

	"github.com/google/uuid"
)

// SnippetOrigin says where a retrieved snippet came from, so the prompt can
// label it (R4).
type SnippetOrigin string

const (
	// OriginPlaybook is a shared worked example.
	OriginPlaybook SnippetOrigin = "playbook"
	// OriginPersonal is something this user finished on another plan.
	OriginPersonal SnippetOrigin = "personal"
)

// Snippet is one retrieved passage that survived the similarity floor.
type Snippet struct {
	Origin SnippetOrigin
	Text   string
}

// RetrievalScope says which rows retrieval may read.
type RetrievalScope string

const (
	// ScopeShared reads shared (playbook) rows only: the user has no personal
	// memory, so a personal search could only come back empty (R18).
	ScopeShared RetrievalScope = "shared"
	// ScopeSharedAndPersonal reads shared rows and the user's own (R26).
	ScopeSharedAndPersonal RetrievalScope = "shared_and_personal"
)

// RetrievalQuery is what a generation flow asks retrieval for: snippets for this
// user, within Scope, never from the plan being suggested for (R3, R26).
type RetrievalQuery struct {
	UserID        uuid.UUID
	ExcludePlanID uuid.UUID
	Scope         RetrievalScope
	Text          string
}

// PersonalMemory answers whether a user has any personal-memory rows, so the
// rule gate can skip personal retrieval for a user with none (R18).
type PersonalMemory interface {
	HasPersonalRows(ctx context.Context, userID uuid.UUID) (bool, error)
}

// Retriever is the retrieval seam (R29), owned by its consumer. The real
// implementation is the human-owned Retriever (I-DF4ZB-14); until it lands the
// flow runs on NoRetrieval. Zero snippets is a valid answer and means weak
// retrieval (R5). An implementation reads only the rows q.Scope allows.
type Retriever interface {
	Retrieve(ctx context.Context, q RetrievalQuery) ([]Snippet, error)
}

// NoRetrieval is the stub Retriever: it always returns zero snippets, so the
// weak-retrieval path is the only path.
type NoRetrieval struct{}

func (NoRetrieval) Retrieve(context.Context, RetrievalQuery) ([]Snippet, error) {
	return nil, nil
}

// SnippetSource is retrieval as a generation flow uses it: the personal-memory
// clause of the rule gate (R18) in front of the Retriever, and failure
// tolerated. Any flow that sends the plan tree to a model with snippets (the
// day plan) asks it the same way.
type SnippetSource struct {
	retriever Retriever
	memory    PersonalMemory
}

func NewSnippetSource(retriever Retriever, memory PersonalMemory) SnippetSource {
	return SnippetSource{retriever: retriever, memory: memory}
}

// Find asks retrieval for snippets about texts, for key's user, never from
// key's plan. Retrieval improves output but never gates it: on failure the
// draw proceeds with none (S14).
func (s SnippetSource) Find(ctx context.Context, key DayKey, texts []string) []Snippet {
	snippets, err := s.retriever.Retrieve(ctx, RetrievalQuery{
		UserID: key.UserID, ExcludePlanID: key.PlanID, Scope: s.scope(ctx, key.UserID),
		Text: strings.Join(texts, "\n"),
	})
	if err != nil {
		slog.WarnContext(ctx, "guided: retrieval failed, drawing without snippets", "err", err, "plan_id", key.PlanID)
		return nil
	}
	return snippets
}

// scope is the rule gate's third clause (R18): a user with no personal-memory
// rows gets shared rows only. The check only saves a search that would come
// back empty, so when it fails the full scope is kept.
func (s SnippetSource) scope(ctx context.Context, userID uuid.UUID) RetrievalScope {
	has, err := s.memory.HasPersonalRows(ctx, userID)
	if err != nil {
		slog.WarnContext(ctx, "guided: personal memory check failed, keeping personal retrieval", "err", err)
		return ScopeSharedAndPersonal
	}
	if !has {
		return ScopeShared
	}
	return ScopeSharedAndPersonal
}
