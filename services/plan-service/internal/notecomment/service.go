package notecomment

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/common/errcode"
	commongrpc "github.com/darkphotonKN/fireplace/common/grpcerror"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
)

// ErrNotANote: the item is a task, which has no comments (R3). The repository
// answers it; the service gives it its wire form.
var ErrNotANote = errors.New("notecomment: item is not a note")

// Repository is the comment table plus the item lookup it is checked against.
type Repository interface {
	// Create inserts the comment after checking, under a lock on the item row,
	// that ItemID is a note in PlanID (R5). A missing item, or one in another
	// plan, is commonconstants.ErrNotFound; a task is ErrNotANote.
	Create(ctx context.Context, in CreateInput) (*Comment, error)
	// List answers the comments on itemID in planID oldest first (R11), with
	// List's refusals those of Create.
	List(ctx context.Context, planID, itemID uuid.UUID) ([]*Comment, error)
	// Get answers ref's comment; out of scope is commonconstants.ErrNotFound
	// (R2).
	Get(ctx context.Context, ref CommentRef) (*Comment, error)
	// Update sets the body as ref.UserID, the author only, else
	// commonconstants.ErrForbidden (R19). The same body is a no-op (R16).
	Update(ctx context.Context, ref CommentRef, body string) (*Comment, error)
	// Delete removes the comment as ref.UserID, its author or the plan's
	// owner, else commonconstants.ErrForbidden (R19).
	Delete(ctx context.Context, ref CommentRef) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) *service {
	return &service{repo: repo}
}

// Create stores a comment by in.AuthorID with the trimmed body (R6, R8, R9).
func (s *service) Create(ctx context.Context, in CreateInput) (*Comment, error) {
	body, err := normalizeBody(in.Body)
	if err != nil {
		return nil, err
	}
	in.Body = body
	c, err := s.repo.Create(ctx, in)
	if err != nil {
		return nil, domainError("notecomment: create", err)
	}
	return c, nil
}

// List answers a note's comments, oldest first; none is an empty list (R11).
func (s *service) List(ctx context.Context, planID, itemID uuid.UUID) ([]*Comment, error) {
	comments, err := s.repo.List(ctx, planID, itemID)
	if err != nil {
		return nil, domainError("notecomment: list", err)
	}
	if comments == nil {
		comments = []*Comment{}
	}
	return comments, nil
}

// Get answers one comment, for the gateway to decide edit and delete (R18).
func (s *service) Get(ctx context.Context, ref CommentRef) (*Comment, error) {
	c, err := s.repo.Get(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("notecomment: get: %w", err)
	}
	return c, nil
}

// Update replaces the body under the create's rules (R10, R16).
func (s *service) Update(ctx context.Context, ref CommentRef, body string) (*Comment, error) {
	trimmed, err := normalizeBody(body)
	if err != nil {
		return nil, err
	}
	c, err := s.repo.Update(ctx, ref, trimmed)
	if err != nil {
		return nil, fmt.Errorf("notecomment: update: %w", err)
	}
	return c, nil
}

// Delete removes the comment for good (R17).
func (s *service) Delete(ctx context.Context, ref CommentRef) error {
	if err := s.repo.Delete(ctx, ref); err != nil {
		return fmt.Errorf("notecomment: delete: %w", err)
	}
	return nil
}

// normalizeBody trims body and holds it to 1-MaxBodyLen code points (R8, R9).
func normalizeBody(body string) (string, error) {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return "", fmt.Errorf("%w: comment body must not be empty", commonconstants.ErrInvalidInput)
	}
	if n := utf8.RuneCountInString(trimmed); n > MaxBodyLen {
		return "", fmt.Errorf("%w: comment body must be at most %d characters (got %d)",
			commonconstants.ErrInvalidInput, MaxBodyLen, n)
	}
	return trimmed, nil
}

// domainError gives ErrNotANote its wire form, FailedPrecondition with
// ITEM_NOT_A_NOTE (R3); anything else keeps its sentinel for the shared mapper.
func domainError(op string, err error) error {
	if errors.Is(err, ErrNotANote) {
		return commongrpc.WithCode(codes.FailedPrecondition, errcode.ItemNotANote, op+": "+err.Error())
	}
	return fmt.Errorf("%s: %w", op, err)
}
