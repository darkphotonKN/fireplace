// Package rag is the storage floor for retrieval (FS-DF4ZB R31–R34, ADR-0018):
// the RAG chunk model, the one embedder, and the rag_chunks repository.
//
// It deliberately holds no similarity query. Ranking, the tenant filter and the
// similarity floor are the Retriever's (R26–R27); this package only writes,
// deletes and answers "does this user have personal rows".
package rag

import (
	"crypto/sha256"
	"database/sql/driver"
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// Source says where a RAG chunk came from. A playbook chunk is a shared row
// (no user, no plan); a completion chunk is personal memory, owned by one user
// and tied to the plan the completed item lives in.
type Source string

const (
	SourcePlaybook   Source = "playbook"
	SourceCompletion Source = "completion"
)

// Vector is a pgvector value. It writes in pgvector's text form ("[1,2,3]"),
// which the server casts to vector(512) on insert, so no driver extension is
// needed.
type Vector []float32

// Value implements driver.Valuer.
func (v Vector) Value() (driver.Value, error) {
	var b strings.Builder
	b.Grow(len(v) * 10)
	b.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(f), 'g', -1, 32))
	}
	b.WriteByte(']')
	return b.String(), nil
}

// Embedding is a vector together with the model and dimension count that
// produced it. A vector is meaningless without its model (ADR-0018 §4), so the
// two never travel apart. Outside tests, only the Embedder makes one.
type Embedding struct {
	Vector Vector
	Model  string
	Dims   int
}

// Chunk is what is written to one row of rag_chunks. UserID and PlanID are nil
// for a shared (playbook) row and set for a personal (completion) row. id and
// the timestamps are the database's.
type Chunk struct {
	UserID      *uuid.UUID
	PlanID      *uuid.UUID
	Source      Source
	SourceID    string
	ChunkNo     int
	Content     string
	ContentHash string
	Embedding   Embedding
}

// ContentHash is the digest stored beside a chunk's content. The repository only
// stores it; comparing it to skip an unchanged chunk belongs to the ingest path.
func ContentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}
