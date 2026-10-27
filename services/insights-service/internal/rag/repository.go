package rag

import (
	"context"

	commonhelpers "github.com/darkphotonKN/fireplace/common/utils"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository writes rag_chunks. It holds no similarity query: that shape is the
// Retriever's (R26–R27).
type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// Upsert writes a chunk keyed by (source, source_id, chunk_no). A second write
// with the same key replaces the row. It stores content_hash but never compares
// it and never skips: the compare-and-skip on re-ingest belongs to the ingest
// path (H4), which decides before it embeds.
func (r *Repository) Upsert(ctx context.Context, c Chunk) error {
	query := `
	INSERT INTO rag_chunks (
		user_id, plan_id, source, source_id, chunk_no,
		content, content_hash, embedding, embedding_model, embedding_dims
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8::vector, $9, $10)
	ON CONFLICT (source, source_id, chunk_no) DO UPDATE SET
		user_id         = EXCLUDED.user_id,
		plan_id         = EXCLUDED.plan_id,
		content         = EXCLUDED.content,
		content_hash    = EXCLUDED.content_hash,
		embedding       = EXCLUDED.embedding,
		embedding_model = EXCLUDED.embedding_model,
		embedding_dims  = EXCLUDED.embedding_dims
	`
	_, err := r.db.ExecContext(ctx, query,
		c.UserID, c.PlanID, c.Source, c.SourceID, c.ChunkNo,
		c.Content, c.ContentHash, c.Embedding.Vector, c.Embedding.Model, c.Embedding.Dims,
	)
	if err != nil {
		return commonhelpers.WrapDBErr("rag_chunks", "upsert", err)
	}
	return nil
}

// DeleteBySource removes every chunk of one source document or item, returning
// how many rows went. Deleting an absent source is not an error. Callers: the
// playbook sweep for a removed document (H4) and un-completion (H3).
func (r *Repository) DeleteBySource(ctx context.Context, source Source, sourceID string) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM rag_chunks WHERE source = $1 AND source_id = $2`, source, sourceID)
	if err != nil {
		return 0, commonhelpers.WrapDBErr("rag_chunks", "delete by source", err)
	}
	return res.RowsAffected()
}

// HasPersonalRows reports whether a user owns any personal row. Shared rows do
// not count. The rule gate (I-DF4ZB-5) uses it to skip personal retrieval for a
// user with no memory yet.
func (r *Repository) HasPersonalRows(ctx context.Context, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists,
		`SELECT EXISTS (SELECT 1 FROM rag_chunks WHERE user_id = $1)`, userID)
	if err != nil {
		return false, commonhelpers.WrapDBErr("rag_chunks", "has personal rows", err)
	}
	return exists, nil
}
