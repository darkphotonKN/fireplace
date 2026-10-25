package rag

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// rag_chunks is SQL over the vector type, so the repository is proven against
// the test tier (postgres-test, ADR-0010 §5) on the pgvector image. The test
// connects as the APP role, so it also proves the default privileges reach a
// table the owner's migration created.
//
// TEST_DB_* rather than DB_*: the Makefile exports .env, whose DB_* point at
// the dev database. Skipped under -short, when the database is unreachable, or
// when rag_chunks has not been migrated there yet (it needs the vector
// extension first: deploy/README.md, "The vector extension on an existing
// volume").

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func openTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("needs the insights test database; run without -short")
	}
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		env("TEST_DB_HOST", "localhost"), env("TEST_DB_PORT", "6500"),
		env("TEST_DB_USER", "fireplace_insights_app"), env("TEST_DB_PASSWORD", "app"),
		env("TEST_DB_NAME", "fireplace_insights_test"),
	)
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("insights test database unreachable (%v)", err)
	}
	t.Cleanup(func() { db.Close() })

	var exists bool
	if err := db.Get(&exists, `SELECT to_regclass('public.rag_chunks') IS NOT NULL`); err != nil || !exists {
		t.Skip("rag_chunks not migrated in the test database (needs the vector extension first)")
	}
	return db
}

// unitVector is a stamped 512-dim embedding pointing along one axis, so two
// different axes are orthogonal (cosine distance 1) and the same axis is 0.
func unitVector(axis int) Embedding {
	v := make(Vector, EmbeddingDims)
	v[axis] = 1
	return Embedding{Vector: v, Model: EmbeddingModel, Dims: EmbeddingDims}
}

func cleanupSource(t *testing.T, db *sqlx.DB, source Source, sourceID string) {
	t.Helper()
	t.Cleanup(func() {
		if _, err := db.Exec(`DELETE FROM rag_chunks WHERE source = $1 AND source_id = $2`, source, sourceID); err != nil {
			t.Errorf("cleanup %s/%s: %v", source, sourceID, err)
		}
	})
}

func personalChunk(userID, planID uuid.UUID, sourceID, content string, emb Embedding) Chunk {
	return Chunk{
		UserID:      &userID,
		PlanID:      &planID,
		Source:      SourceCompletion,
		SourceID:    sourceID,
		ChunkNo:     0,
		Content:     content,
		ContentHash: ContentHash(content),
		Embedding:   emb,
	}
}

func TestUpsert_SameKeyTwice_ReplacesTheRow(t *testing.T) {
	db := openTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	sourceID := "playbook-" + uuid.NewString()
	cleanupSource(t, db, SourcePlaybook, sourceID)

	first := Chunk{Source: SourcePlaybook, SourceID: sourceID, ChunkNo: 2,
		Content: "first write", ContentHash: ContentHash("first write"), Embedding: unitVector(0)}
	second := first
	second.Content = "second write"
	second.ContentHash = ContentHash("second write")
	second.Embedding = unitVector(1)

	if err := repo.Upsert(ctx, first); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	if err := repo.Upsert(ctx, second); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	var rows []struct {
		Content     string  `db:"content"`
		ContentHash string  `db:"content_hash"`
		Model       string  `db:"embedding_model"`
		Dims        int     `db:"embedding_dims"`
		DistToAxis1 float64 `db:"dist"`
	}
	err := db.Select(&rows, `
		SELECT content, content_hash, embedding_model, embedding_dims,
		       embedding <=> $3::vector AS dist
		FROM rag_chunks WHERE source = $1 AND source_id = $2 AND chunk_no = 2`,
		SourcePlaybook, sourceID, unitVector(1).Vector)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows for one key, want 1", len(rows))
	}
	got := rows[0]
	if got.Content != "second write" || got.ContentHash != ContentHash("second write") {
		t.Errorf("row holds %q / %q, want the second write", got.Content, got.ContentHash)
	}
	if got.DistToAxis1 != 0 {
		t.Errorf("embedding not replaced: cosine distance to the second vector = %v, want 0", got.DistToAxis1)
	}
	if got.Model != EmbeddingModel || got.Dims != EmbeddingDims {
		t.Errorf("row records %s/%d, want %s/%d", got.Model, got.Dims, EmbeddingModel, EmbeddingDims)
	}
}

func TestDeleteBySource_RemovesEveryChunkOfThatSourceOnly(t *testing.T) {
	db := openTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	doomed := "playbook-" + uuid.NewString()
	kept := "playbook-" + uuid.NewString()
	cleanupSource(t, db, SourcePlaybook, doomed)
	cleanupSource(t, db, SourcePlaybook, kept)

	for _, c := range []Chunk{
		{Source: SourcePlaybook, SourceID: doomed, ChunkNo: 0, Content: "a", ContentHash: ContentHash("a"), Embedding: unitVector(0)},
		{Source: SourcePlaybook, SourceID: doomed, ChunkNo: 1, Content: "b", ContentHash: ContentHash("b"), Embedding: unitVector(1)},
		{Source: SourcePlaybook, SourceID: kept, ChunkNo: 0, Content: "c", ContentHash: ContentHash("c"), Embedding: unitVector(2)},
	} {
		if err := repo.Upsert(ctx, c); err != nil {
			t.Fatalf("seed %s/%d: %v", c.SourceID, c.ChunkNo, err)
		}
	}

	deleted, err := repo.DeleteBySource(ctx, SourcePlaybook, doomed)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if deleted != 2 {
		t.Errorf("deleted %d rows, want 2", deleted)
	}

	left := map[string]int{}
	for _, id := range []string{doomed, kept} {
		var n int
		if err := db.Get(&n, `SELECT COUNT(*) FROM rag_chunks WHERE source = $1 AND source_id = $2`, SourcePlaybook, id); err != nil {
			t.Fatalf("count: %v", err)
		}
		left[id] = n
	}
	if left[doomed] != 0 || left[kept] != 1 {
		t.Errorf("after delete: doomed=%d kept=%d, want 0 and 1", left[doomed], left[kept])
	}

	again, err := repo.DeleteBySource(ctx, SourcePlaybook, doomed)
	if err != nil || again != 0 {
		t.Errorf("deleting an absent source = (%d, %v), want (0, nil)", again, err)
	}
}

func TestHasPersonalRows_TrueOnlyForTheOwningUser(t *testing.T) {
	db := openTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner, stranger := uuid.New(), uuid.New()
	itemID := uuid.NewString()
	cleanupSource(t, db, SourceCompletion, itemID)

	// A shared row exists for everyone; it is not anyone's personal row.
	sharedID := "playbook-" + uuid.NewString()
	cleanupSource(t, db, SourcePlaybook, sharedID)
	if err := repo.Upsert(ctx, Chunk{Source: SourcePlaybook, SourceID: sharedID,
		Content: "shared", ContentHash: ContentHash("shared"), Embedding: unitVector(3)}); err != nil {
		t.Fatalf("seed shared: %v", err)
	}

	before, err := repo.HasPersonalRows(ctx, owner)
	if err != nil || before {
		t.Fatalf("before any completion: (%v, %v), want (false, nil)", before, err)
	}

	if err := repo.Upsert(ctx, personalChunk(owner, uuid.New(), itemID, "finished the outline", unitVector(4))); err != nil {
		t.Fatalf("seed personal: %v", err)
	}

	tests := []struct {
		name string
		user uuid.UUID
		want bool
	}{
		{"owner of a personal row", owner, true},
		{"another user", stranger, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.HasPersonalRows(ctx, tt.user)
			if err != nil {
				t.Fatalf("has personal rows: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpsert_RowFromTheEmbedder_RecordsSmall3And512(t *testing.T) {
	db := openTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	vec := make([]float32, EmbeddingDims)
	vec[0] = 1
	e := newEmbedder(&fakeEmbeddingsClient{resp: responseWith(vec)})
	emb, err := e.Embed(ctx, "wrote the first chapter")
	if err != nil {
		t.Fatalf("embed: %v", err)
	}

	itemID := uuid.NewString()
	cleanupSource(t, db, SourceCompletion, itemID)
	if err := repo.Upsert(ctx, personalChunk(uuid.New(), uuid.New(), itemID, "wrote the first chapter", emb)); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	var row struct {
		Model string `db:"embedding_model"`
		Dims  int    `db:"embedding_dims"`
		Real  int    `db:"real_dims"`
	}
	err = db.Get(&row, `SELECT embedding_model, embedding_dims, vector_dims(embedding) AS real_dims
		FROM rag_chunks WHERE source = $1 AND source_id = $2`, SourceCompletion, itemID)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if row.Model != "text-embedding-3-small" || row.Dims != 512 || row.Real != 512 {
		t.Errorf("row records %s/%d (vector %d), want text-embedding-3-small/512", row.Model, row.Dims, row.Real)
	}
}

func TestUpsert_RejectsAPersonalRowWithoutItsPlan(t *testing.T) {
	db := openTestDB(t)
	repo := NewRepository(db)

	itemID := uuid.NewString()
	cleanupSource(t, db, SourceCompletion, itemID)
	c := personalChunk(uuid.New(), uuid.New(), itemID, "x", unitVector(0))
	c.PlanID = nil

	if err := repo.Upsert(context.Background(), c); err == nil {
		t.Fatal("a completion row with no plan was stored; retrieval could not exclude its plan (R26)")
	}
}
