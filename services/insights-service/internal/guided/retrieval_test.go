package guided

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeRetriever struct {
	snippets []Snippet
	err      error
	got      *RetrievalQuery
}

func (f *fakeRetriever) Retrieve(ctx context.Context, q RetrievalQuery) ([]Snippet, error) {
	f.got = &q
	return f.snippets, f.err
}

type fakeMemory struct {
	has   bool
	err   error
	asked []uuid.UUID
}

func (f *fakeMemory) HasPersonalRows(ctx context.Context, userID uuid.UUID) (bool, error) {
	f.asked = append(f.asked, userID)
	return f.has, f.err
}

func snippetKey() DayKey {
	return DayKey{PlanID: uuid.New(), UserID: uuid.New(), Date: time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)}
}

func TestSnippetSource_AsksForTheUserNeverThePlanAndReturnsWhatIsFound(t *testing.T) {
	r := &fakeRetriever{snippets: []Snippet{{Origin: OriginPersonal, Text: "Set up CI for the blog"}}}
	key := snippetKey()

	got := NewSnippetSource(r, &fakeMemory{has: true}).Find(context.Background(), key, []string{"Ship the API", "Learn Go"})

	if r.got == nil || r.got.UserID != key.UserID || r.got.ExcludePlanID != key.PlanID || r.got.Text != "Ship the API\nLearn Go" {
		t.Fatalf("retrieval query = %+v", r.got)
	}
	if len(got) != 1 || got[0].Text != "Set up CI for the blog" {
		t.Fatalf("snippets = %+v", got)
	}
}

// Retrieval improves output but never gates it (S14).
func TestSnippetSource_RetrievalFailureFallsBackToNoSnippets(t *testing.T) {
	got := NewSnippetSource(&fakeRetriever{err: errors.New("embedding API down")}, &fakeMemory{}).
		Find(context.Background(), snippetKey(), []string{"x"})

	if got != nil {
		t.Fatalf("snippets = %+v, want none", got)
	}
}

func TestSnippetSource_ScopeFollowsPersonalMemory(t *testing.T) {
	tests := []struct {
		name string
		has  bool
		err  error
		want RetrievalScope
	}{
		{"no personal rows: shared only", false, nil, ScopeShared},
		{"personal rows: shared and personal", true, nil, ScopeSharedAndPersonal},
		// The check only saves a personal search; failing it must not narrow
		// what a user with memory gets.
		{"memory check fails: shared and personal", false, errors.New("db down"), ScopeSharedAndPersonal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, memory, key := &fakeRetriever{}, &fakeMemory{has: tt.has, err: tt.err}, snippetKey()

			NewSnippetSource(r, memory).Find(context.Background(), key, []string{"x"})

			if r.got == nil || r.got.Scope != tt.want {
				t.Fatalf("retrieval query = %+v, want scope %q", r.got, tt.want)
			}
			if len(memory.asked) != 1 || memory.asked[0] != key.UserID {
				t.Fatalf("personal memory asked for %v, want [%s]", memory.asked, key.UserID)
			}
		})
	}
}
