package inbox

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository is the persistence seam for the processed-events inbox (the dedup
// ledger). The consumer owns the abstraction (DIP); the concrete *repository is
// injected at SetupServices.
type Repository interface {
	Create(ctx context.Context, eventID uuid.UUID) error
	CreateTx(ctx context.Context, tx *sqlx.Tx, eventID uuid.UUID) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// MarkProcessed records an event as handled in the dedup ledger. Stub for now.
func (s *Service) MarkProcessed(ctx context.Context, eventID uuid.UUID) error {
	return s.repo.Create(ctx, eventID)
}

// MarkProcessedTx is the transactional sibling of MarkProcessed, meant to be
// called inside a caller-owned tx (e.g. alongside the insight write so the
// dedup mark commits atomically with it). Stub for now.
func (s *Service) MarkProcessedTx(ctx context.Context, tx *sqlx.Tx, eventID uuid.UUID) error {
	return s.repo.CreateTx(ctx, tx, eventID)
}
