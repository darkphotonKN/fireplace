package checklistitem

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	commonbroker "github.com/darkphotonKN/fireplace/common/broker"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/common/errcode"
	commongrpc "github.com/darkphotonKN/fireplace/common/grpcerror"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
)

type Repository interface {
	Create(ctx context.Context, in CreateItemInput, sequenceNo int) (*Item, error)
	Update(ctx context.Context, in UpdateItemInput) error
	UpdateDates(ctx context.Context, planID, id uuid.UUID, startDate, dueDate *time.Time) error
	Delete(ctx context.Context, planID, id uuid.UUID) error
	ListByPlanID(ctx context.Context, in ListItemsInput) ([]*Item, error)
	ListSiblings(ctx context.Context, in SiblingSetInput) ([]*Item, error)
	Reorder(ctx context.Context, ids []uuid.UUID) ([]*Item, error)
	Duplicate(ctx context.Context, planID, id uuid.UUID) ([]*Item, error)
	ListArchivedByPlanID(ctx context.Context, planID uuid.UUID, scope *string) ([]*Item, error)
	ListInDateWindow(ctx context.Context, planID uuid.UUID, windowStart, windowEnd time.Time) ([]*Item, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Item, error)
	GetInPlan(ctx context.Context, planID, id uuid.UUID) (*Item, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*Item, error)
	HasChildren(ctx context.Context, id uuid.UUID) (bool, error)
	CountItems(ctx context.Context) (int, error)
}

type service struct {
	repo      Repository
	publishCh commonbroker.Publisher
}

func NewService(repo Repository, publishCh commonbroker.Publisher) *service {
	return &service{repo: repo, publishCh: publishCh}
}

// GetInPlan returns an item only when it belongs to planID. An item in another
// plan is ErrNotFound, indistinguishable from a missing one (FS-12ZMF R11).
func (s *service) GetInPlan(ctx context.Context, planID, id uuid.UUID) (*Item, error) {
	item, err := s.repo.GetInPlan(ctx, planID, id)
	if err != nil {
		return nil, fmt.Errorf("checklistitem: get: %w", err)
	}
	return item, nil
}

func (s *service) ListByPlanID(ctx context.Context, in ListItemsInput) ([]*Item, error) {
	if in.Scope != nil && *in.Scope != ScopeDaily && *in.Scope != ScopeLongterm {
		return nil, fmt.Errorf("%w: scope must be 'daily' or 'longterm'", commonconstants.ErrInvalidInput)
	}
	if in.Type != nil && *in.Type != TypeTask && *in.Type != TypeNote {
		return nil, fmt.Errorf("%w: type must be 'task' or 'note'", commonconstants.ErrInvalidInput)
	}
	if in.Upcoming != nil && *in.Upcoming != UpcomingWeek && *in.Upcoming != UpcomingMonth {
		return nil, fmt.Errorf("%w: upcoming must be 'week' or 'month'", commonconstants.ErrInvalidInput)
	}
	items, err := s.repo.ListByPlanID(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("checklistitem: list by plan: %w", err)
	}
	return items, nil
}

func (s *service) ListArchivedByPlanID(ctx context.Context, planID uuid.UUID, scope *string) ([]*Item, error) {
	if scope != nil && *scope != ScopeDaily && *scope != ScopeLongterm {
		return nil, fmt.Errorf("%w: scope must be 'daily' or 'longterm'", commonconstants.ErrInvalidInput)
	}
	items, err := s.repo.ListArchivedByPlanID(ctx, planID, scope)
	if err != nil {
		return nil, fmt.Errorf("checklistitem: list archived by plan: %w", err)
	}
	return items, nil
}

func (s *service) ListUpcoming(ctx context.Context, planID uuid.UUID) ([]*Item, error) {
	week := UpcomingWeek
	return s.ListByPlanID(ctx, ListItemsInput{PlanID: planID, Upcoming: &week})
}

func (s *service) Create(ctx context.Context, in CreateItemInput) (*Item, error) {
	if in.Scope != nil && *in.Scope != ScopeDaily && *in.Scope != ScopeLongterm {
		return nil, fmt.Errorf("%w: scope must be 'daily' or 'longterm'", commonconstants.ErrInvalidInput)
	}
	if err := refuseDailyScope(in.Scope); err != nil {
		return nil, err
	}
	if in.Type != nil && *in.Type != TypeTask && *in.Type != TypeNote {
		return nil, fmt.Errorf("%w: type must be 'task' or 'note'", commonconstants.ErrInvalidInput)
	}
	if in.ParentID != nil {
		if err := s.validateParent(ctx, in.PlanID, *in.ParentID); err != nil {
			return nil, err
		}
	}

	count, err := s.repo.CountItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("checklistitem: create: count items: %w", err)
	}
	item, err := s.repo.Create(ctx, in, count+1)
	if err != nil {
		return nil, fmt.Errorf("checklistitem: create: %w", err)
	}
	return item, nil
}

// validateParent enforces the two-tier rule: parent must exist, belong to the
// same plan, and itself have no parent. A missing parent is a client error
// (invalid input); any other repo error is propagated unchanged so a transient
// DB failure is not misreported as bad input.
func (s *service) validateParent(ctx context.Context, planID, parentID uuid.UUID) error {
	parent, err := s.repo.GetByID(ctx, parentID)
	if err != nil {
		if errors.Is(err, commonconstants.ErrNotFound) {
			return fmt.Errorf("%w: parent not found", commonconstants.ErrInvalidInput)
		}
		return fmt.Errorf("checklistitem: validate parent: %w", err)
	}
	if parent.PlanID != planID {
		return fmt.Errorf("%w: parent belongs to a different plan", commonconstants.ErrInvalidInput)
	}
	if parent.ParentID != nil {
		return fmt.Errorf("%w: cannot nest under a child item (two-tier max)", commonconstants.ErrInvalidInput)
	}
	return nil
}

func (s *service) Update(ctx context.Context, in UpdateItemInput) (*Item, error) {
	if err := refuseDailyScope(in.Scope); err != nil {
		return nil, err
	}
	if in.Type != nil && *in.Type != TypeTask && *in.Type != TypeNote {
		return nil, fmt.Errorf("%w: type must be 'task' or 'note'", commonconstants.ErrInvalidInput)
	}

	// The item must be in the addressed plan before any rule below is checked:
	// a rule answered for another plan's row would tell the caller it exists.
	current, err := s.repo.GetInPlan(ctx, in.PlanID, in.ID)
	if err != nil {
		return nil, fmt.Errorf("checklistitem: update: %w", err)
	}

	if in.SetIntention {
		in.Intention, err = normalizeIntention(in.Intention)
		if err != nil {
			return nil, err
		}
	}

	// task→note transition: reject if it has children (notes can't be parents).
	if in.Type != nil && *in.Type == TypeNote {
		hasKids, err := s.repo.HasChildren(ctx, in.ID)
		if err != nil {
			return nil, fmt.Errorf("checklistitem: update: check children: %w", err)
		}
		if hasKids {
			return nil, fmt.Errorf("%w: cannot convert a parent with children into a note", commonconstants.ErrInvalidInput)
		}
		// force done=false to satisfy chk_note_not_done.
		if in.Done == nil && current.Done {
			f := false
			in.Done = &f
		}
	}

	// Re-parent (indent): validate target is in same plan, is top-level, and
	// this row has no children of its own (would push past tier 2).
	if in.SetParent && in.ParentID != nil {
		if err := s.validateParent(ctx, current.PlanID, *in.ParentID); err != nil {
			return nil, err
		}
		hasKids, err := s.repo.HasChildren(ctx, in.ID)
		if err != nil {
			return nil, fmt.Errorf("checklistitem: update: check children: %w", err)
		}
		if hasKids {
			return nil, fmt.Errorf("%w: cannot re-parent a row with children", commonconstants.ErrInvalidInput)
		}
	}

	if err := s.repo.Update(ctx, in); err != nil {
		return nil, fmt.Errorf("checklistitem: update: %w", err)
	}

	updated, err := s.repo.GetInPlan(ctx, in.PlanID, in.ID)
	if err != nil {
		return nil, fmt.Errorf("checklistitem: update: reload: %w", err)
	}

	// Emit completion event when an Update sets done=true.
	if in.Done != nil && *in.Done {
		s.PublishItemCompleted(ctx, updated)
	}
	if in.Done != nil && !*in.Done {
		s.PublishItemUncompleted(ctx, updated)
	}
	return updated, nil
}

// refuseDailyScope refuses an item write that names the retired daily scope
// (FS-02JBZ R51). The value stays a legal enum member in the proto and the
// schema, so an old client's request is well-formed; the refusal is a domain
// decision and carries DAILY_SCOPE_RETIRED as an ErrorInfo detail. It is
// returned unwrapped so the status, and its detail, reach the handler intact.
// Reads and reorders by daily are not refused (S15): they find nothing.
func refuseDailyScope(scope *string) error {
	if scope == nil || *scope != ScopeDaily {
		return nil
	}
	return commongrpc.WithCode(codes.InvalidArgument, errcode.DailyScopeRetired,
		"the daily scope is retired: items are long-term")
}

// normalizeIntention trims an intention and turns an empty result into nil, so
// whitespace clears the column rather than storing blanks. A trimmed value over
// MaxIntentionChars characters is refused (FS-043WP R38).
func normalizeIntention(v *string) (*string, error) {
	if v == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*v)
	if trimmed == "" {
		return nil, nil
	}
	if n := utf8.RuneCountInString(trimmed); n > MaxIntentionChars {
		return nil, fmt.Errorf("%w: intention must be at most %d characters (got %d)",
			commonconstants.ErrInvalidInput, MaxIntentionChars, n)
	}
	return &trimmed, nil
}

func (s *service) UpdateDates(ctx context.Context, in UpdateDatesInput) (*Item, error) {
	current, err := s.repo.GetInPlan(ctx, in.PlanID, in.ID)
	if err != nil {
		return nil, fmt.Errorf("checklistitem: update dates: %w", err)
	}
	if !in.SetStart && !in.SetDue {
		return current, nil
	}

	finalStart := current.StartDate
	finalDue := current.DueDate
	if in.SetStart {
		finalStart = in.StartDate
	}
	if in.SetDue {
		finalDue = in.DueDate
	}
	if finalStart != nil && finalDue != nil && finalStart.After(*finalDue) {
		return nil, fmt.Errorf("%w: start_date must be on or before due_date", commonconstants.ErrInvalidInput)
	}

	if err := s.repo.UpdateDates(ctx, in.PlanID, in.ID, finalStart, finalDue); err != nil {
		return nil, fmt.Errorf("checklistitem: update dates: %w", err)
	}
	current.StartDate = finalStart
	current.DueDate = finalDue
	return current, nil
}

// Reorder writes the order of ONE sibling set (FS-0009 R2). The request carries
// the set's complete final order; the sequences are written densely as 1..N in a
// single transaction and the reordered siblings are returned.
func (s *service) Reorder(ctx context.Context, in ReorderInput) ([]*Item, error) {
	if in.Scope != ScopeDaily && in.Scope != ScopeLongterm {
		return nil, fmt.Errorf("%w: scope must be 'daily' or 'longterm'", commonconstants.ErrInvalidInput)
	}
	if len(in.IDs) == 0 {
		return nil, fmt.Errorf("%w: ids must not be empty", commonconstants.ErrInvalidInput)
	}

	siblings, err := s.repo.ListSiblings(ctx, in.siblingSet())
	if err != nil {
		return nil, fmt.Errorf("checklistitem: reorder: load sibling set: %w", err)
	}
	if len(siblings) == 0 {
		// Nothing answers to this (plan, scope, parent): an unknown plan, an
		// unknown parent, or a set whose members are all gone.
		return nil, fmt.Errorf("checklistitem: reorder: no sibling set for plan %s: %w",
			in.PlanID, commonconstants.ErrNotFound)
	}

	if err := s.assertPermutation(ctx, siblings, in.IDs); err != nil {
		return nil, err
	}

	items, err := s.repo.Reorder(ctx, in.IDs)
	if err != nil {
		return nil, fmt.Errorf("checklistitem: reorder: %w", err)
	}
	return items, nil
}

// Duplicate copies one live item, with a top-level item's live children, and
// places the copy directly after it (FS-F6AEE). The source must be in the
// addressed plan (else NotFound, like a missing item) and not archived (else
// InvalidInput: duplicating something archived would really be a restore).
// Nothing is published: the copy is never done, so no completion is owed (R9).
func (s *service) Duplicate(ctx context.Context, planID, id uuid.UUID) ([]*Item, error) {
	source, err := s.repo.GetInPlan(ctx, planID, id)
	if err != nil {
		return nil, fmt.Errorf("checklistitem: duplicate: %w", err)
	}
	if source.Archived {
		return nil, fmt.Errorf("%w: an archived item cannot be duplicated", commonconstants.ErrInvalidInput)
	}
	items, err := s.repo.Duplicate(ctx, planID, id)
	if err != nil {
		return nil, fmt.Errorf("checklistitem: duplicate: %w", err)
	}
	return items, nil
}

// assertPermutation refuses anything that is not exactly the sibling set in a
// new order: a stranger, or a missing member (FS-0009 §Edge States).
func (s *service) assertPermutation(ctx context.Context, siblings []*Item, ids []uuid.UUID) error {
	set := make(map[uuid.UUID]struct{}, len(siblings))
	for _, sib := range siblings {
		set[sib.ID] = struct{}{}
	}
	seen := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			return fmt.Errorf("%w: item %s appears twice in ids", commonconstants.ErrInvalidInput, id)
		}
		seen[id] = struct{}{}
		if _, member := set[id]; !member {
			// An id that exists but sits elsewhere (another parent, scope or
			// plan) is a stranger to this set — bad input. An id that exists
			// nowhere is a 404, so the two are told apart by a lookup rather
			// than collapsed into one status.
			if _, err := s.repo.GetByID(ctx, id); err != nil {
				return fmt.Errorf("checklistitem: reorder: unknown item %s: %w", id, err)
			}
			return fmt.Errorf("%w: item %s is not a member of this sibling set", commonconstants.ErrInvalidInput, id)
		}
	}
	if len(ids) != len(siblings) {
		return fmt.Errorf("%w: ids must be the complete sibling set (%d given, %d in the set)",
			commonconstants.ErrInvalidInput, len(ids), len(siblings))
	}
	return nil
}

func (s *service) Archive(ctx context.Context, planID, id uuid.UUID, archived bool) (*Item, error) {
	if err := s.repo.Update(ctx, UpdateItemInput{ID: id, PlanID: planID, Archived: &archived}); err != nil {
		return nil, fmt.Errorf("checklistitem: archive: %w", err)
	}
	item, err := s.repo.GetInPlan(ctx, planID, id)
	if err != nil {
		return nil, fmt.Errorf("checklistitem: archive: reload: %w", err)
	}
	return item, nil
}

func (s *service) Delete(ctx context.Context, planID, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, planID, id); err != nil {
		return fmt.Errorf("checklistitem: delete: %w", err)
	}
	return nil
}

// GetByUserID returns non-archived items across all the user's plans.
// Used by useranalytics in 4c via gRPC.
func (s *service) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*Item, error) {
	items, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("checklistitem: get by user: %w", err)
	}
	return items, nil
}

// ListInDateWindow returns items whose date range overlaps the given window.
// Used by the gateway's calendar read model over gRPC.
func (s *service) ListInDateWindow(ctx context.Context, planID uuid.UUID, windowStart, windowEnd time.Time) ([]*Item, error) {
	items, err := s.repo.ListInDateWindow(ctx, planID, windowStart, windowEnd)
	if err != nil {
		return nil, fmt.Errorf("checklistitem: list in date window: %w", err)
	}
	return items, nil
}
