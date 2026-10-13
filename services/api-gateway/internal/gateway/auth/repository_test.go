package authgw

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// The member lookups are SQL, so they are proven against the local gateway
// database (docker compose up postgres). Skipped under -short or when the
// database is unreachable.

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func openTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("needs the gateway database; run without -short")
	}
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		env("DB_HOST", "localhost"), env("DB_PORT", "5500"),
		env("DB_USER", "fireplace_gateway_app"), env("DB_PASSWORD", "app"),
		env("DB_NAME", "fireplace_gateway"),
	)
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("gateway database unreachable (%v)", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func seedUser(t *testing.T, db *sqlx.DB, name, email string, displayName *string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := db.Get(&id,
		`INSERT INTO users (name, email, password, display_name) VALUES ($1, $2, 'x', $3) RETURNING id`,
		name, email, displayName)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec(`DELETE FROM users WHERE id = $1`, id); err != nil {
			t.Errorf("cleanup user %s: %v", id, err)
		}
	})
	return id
}

// FS-12ZMF R15, R16: one batch read answers the users that exist, without
// their password hash; an id with no user is absent.
func TestIntegrationGetByIDs_AnswersOnlyUsersThatExist(t *testing.T) {
	db := openTestDB(t)
	suffix := uuid.NewString()[:8]
	shown := "Nicky"
	a := seedUser(t, db, "Nick", "nick-"+suffix+"@x.test", &shown)
	b := seedUser(t, db, "Ana", "ana-"+suffix+"@x.test", nil)

	got, err := NewRepository(db).GetByIDs(context.Background(), []uuid.UUID{a, uuid.New(), b})
	if err != nil {
		t.Fatalf("GetByIDs: %v", err)
	}
	byID := map[uuid.UUID]*User{}
	for _, u := range got {
		byID[u.ID] = u
		if u.HashedPassword != "" {
			t.Errorf("user %s carries a password hash", u.ID)
		}
	}
	if len(got) != 2 || byID[a] == nil || byID[b] == nil {
		t.Fatalf("got %d users, want exactly %s and %s", len(got), a, b)
	}
	if byID[a].ShownName() != "Nicky" || byID[b].ShownName() != "Ana" {
		t.Errorf("names = %q, %q; want Nicky, Ana", byID[a].ShownName(), byID[b].ShownName())
	}
}

func TestIntegrationGetByIDs_NoIDs_IsEmpty(t *testing.T) {
	got, err := NewRepository(openTestDB(t)).GetByIDs(context.Background(), nil)
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v; want an empty result", got, err)
	}
}

// FS-12ZMF R17: the email lookup is the exact match sign-in uses, so a
// different case finds nobody.
func TestIntegrationGetByEmail_IsAnExactMatch(t *testing.T) {
	db := openTestDB(t)
	email := "case-" + uuid.NewString()[:8] + "@x.test"
	id := seedUser(t, db, "Case", email, nil)
	repo := NewRepository(db)

	u, err := repo.GetByEmail(context.Background(), email)
	if err != nil || u.ID != id {
		t.Fatalf("exact: got %v, %v; want %s", u, err, id)
	}
	if _, err := repo.GetByEmail(context.Background(), "CASE-"+email[5:]); !errors.Is(err, commonconstants.ErrNotFound) {
		t.Errorf("different case: err = %v, want ErrNotFound", err)
	}
}
