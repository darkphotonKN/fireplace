package checklistitem

import (
	"context"
	"errors"
	"testing"

	commonbroker "github.com/darkphotonKN/fireplace/common/broker"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
)

// duplicateRepo answers GetInPlan from the rows it holds, scoped by plan like
// the real WHERE id AND plan_id, and records whether Duplicate was reached.
type duplicateRepo struct {
	Repository

	rows       map[uuid.UUID]*Item
	duplicated []uuid.UUID
	out        []*Item
	err        error
}

func (r *duplicateRepo) GetInPlan(ctx context.Context, planID, id uuid.UUID) (*Item, error) {
	it, ok := r.rows[id]
	if !ok || it.PlanID != planID {
		return nil, commonconstants.ErrNotFound
	}
	copied := *it
	return &copied, nil
}

func (r *duplicateRepo) Duplicate(ctx context.Context, planID, id uuid.UUID) ([]*Item, error) {
	r.duplicated = append(r.duplicated, id)
	return r.out, r.err
}

// recordingPublisher fails the test's assertion if anything is published.
type recordingPublisher struct{ published []string }

func (p *recordingPublisher) PublishWithContext(_ context.Context, exchange, key string, _ commonbroker.Message) error {
	p.published = append(p.published, exchange+"/"+key)
	return nil
}

func TestDuplicate_LiveSourceReturnsWhatTheRepositoryCreatedAndPublishesNothing(t *testing.T) {
	planID := uuid.New()
	source := &Item{ID: uuid.New(), PlanID: planID, Done: true}
	copyItem, child := &Item{ID: uuid.New(), PlanID: planID}, &Item{ID: uuid.New(), PlanID: planID}
	repo := &duplicateRepo{
		rows: map[uuid.UUID]*Item{source.ID: source},
		out:  []*Item{copyItem, child},
	}
	pub := &recordingPublisher{}

	got, err := NewService(repo, pub).Duplicate(context.Background(), planID, source.ID)

	if err != nil {
		t.Fatalf("Duplicate: %v", err)
	}
	if len(got) != 2 || got[0] != copyItem || got[1] != child {
		t.Errorf("got %v, want the repository's [copy child]", got)
	}
	if len(repo.duplicated) != 1 || repo.duplicated[0] != source.ID {
		t.Errorf("repository duplicated %v, want [%s]", repo.duplicated, source.ID)
	}
	if len(pub.published) != 0 {
		t.Errorf("published %v, want nothing (R9)", pub.published)
	}
}

func TestDuplicate_Refusals_WriteNothing(t *testing.T) {
	planID, otherPlan := uuid.New(), uuid.New()
	archived := &Item{ID: uuid.New(), PlanID: planID, Archived: true}
	foreign := &Item{ID: uuid.New(), PlanID: otherPlan}

	cases := []struct {
		name    string
		id      uuid.UUID
		wantErr error
	}{
		{"archived source is invalid input", archived.ID, commonconstants.ErrInvalidInput},
		{"source in another plan is not found", foreign.ID, commonconstants.ErrNotFound},
		{"missing source is not found", uuid.New(), commonconstants.ErrNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &duplicateRepo{rows: map[uuid.UUID]*Item{archived.ID: archived, foreign.ID: foreign}}

			_, err := NewService(repo, nil).Duplicate(context.Background(), planID, tc.id)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if len(repo.duplicated) != 0 {
				t.Error("a refused duplicate reached the write")
			}
		})
	}
}

func TestDuplicate_RepositoryFailureIsPropagated(t *testing.T) {
	planID := uuid.New()
	source := &Item{ID: uuid.New(), PlanID: planID}
	boom := errors.New("tx aborted")
	repo := &duplicateRepo{rows: map[uuid.UUID]*Item{source.ID: source}, err: boom}

	_, err := NewService(repo, nil).Duplicate(context.Background(), planID, source.ID)

	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want it to wrap %v", err, boom)
	}
}
