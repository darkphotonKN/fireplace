package guided

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/darkphotonKN/fireplace/services/insights-service/internal/ai"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // registers the postgres driver
)

// The repository is proven against the insights test database (postgres-test,
// ADR-0010 §5), connecting as the APP role so the default privileges are proven
// too. TEST_DB_* rather than DB_*: the Makefile exports .env, whose DB_* point
// at the dev database. Skipped under -short, when the database is unreachable,
// or when generations has not been migrated there.

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
	if err := db.Get(&exists, `SELECT to_regclass('public.generations') IS NOT NULL`); err != nil || !exists {
		t.Skip("generations not migrated in the test database")
	}
	return db
}

func cleanupPlan(t *testing.T, db *sqlx.DB, planID uuid.UUID) {
	t.Helper()
	t.Cleanup(func() {
		if _, err := db.Exec(`DELETE FROM generations WHERE plan_id = $1`, planID); err != nil {
			t.Errorf("cleanup generations: %v", err)
		}
	})
}

func TestRepositoryIntegration_RecordAttemptStoresUsage(t *testing.T) {
	db := openTestDB(t)
	repo := NewRepository(db)
	planID := uuid.New()
	cleanupPlan(t, db, planID)
	// "daily_suggestions" and "daily_brief" have no constant since their
	// generators were removed (FS-DF4ZB R55, FS-02JBZ R92), but rows written
	// before then must stay valid.
	kinds := []ai.Kind{ai.Kind("daily_brief"), ai.KindChecklistSuggestion, ai.KindVideoTerms, ai.Kind("daily_suggestions"), ai.KindDayPlan}

	for _, kind := range kinds {
		a := ai.Attempt{
			Call:  ai.Call{Kind: kind, PlanID: planID, UserID: uuid.New()},
			Model: "gpt-5-mini", Usage: ai.Usage{InputTokens: 1200, CachedInputTokens: 1024, OutputTokens: 90},
		}
		if err := repo.RecordAttempt(context.Background(), a); err != nil {
			t.Fatalf("RecordAttempt(%s): %v", kind, err)
		}
	}

	var rows []struct {
		Kind      string    `db:"kind"`
		Model     string    `db:"model"`
		In        int       `db:"input_tokens"`
		Cached    int       `db:"cached_input_tokens"`
		Out       int       `db:"output_tokens"`
		Succeeded bool      `db:"succeeded"`
		CreatedAt time.Time `db:"created_at"`
	}
	if err := db.Select(&rows, `SELECT kind, model, input_tokens, cached_input_tokens, output_tokens, succeeded, created_at
		FROM generations WHERE plan_id = $1 ORDER BY created_at, kind`, planID); err != nil {
		t.Fatal(err)
	}
	if len(rows) != len(kinds) {
		t.Fatalf("rows = %d, want one per kind (%d)", len(rows), len(kinds))
	}
	for _, row := range rows {
		if row.Model != "gpt-5-mini" || row.In != 1200 || row.Cached != 1024 || row.Out != 90 || row.Succeeded || row.CreatedAt.IsZero() {
			t.Fatalf("row = %+v", row)
		}
	}
}

// The budget sums input plus output tokens (cached input is already inside
// input) over every row from the window start on, whatever its kind, plan or
// user; rows before the start do not count (FS-02JBZ R32).
func TestRepositoryIntegration_TokensSinceSumsInputAndOutputFromTheWindowStart(t *testing.T) {
	db := openTestDB(t)
	repo := NewRepository(db)
	planA, planB := uuid.New(), uuid.New()
	cleanupPlan(t, db, planA)
	cleanupPlan(t, db, planB)
	// Far in the future, so rows other tests write now fall before the window.
	dayStart := time.Date(2099, 1, 2, 0, 0, 0, 0, time.UTC)
	insert := func(planID uuid.UUID, kind string, in, cached, out int, at time.Time) {
		t.Helper()
		if _, err := db.Exec(`
			INSERT INTO generations (kind, plan_id, user_id, model, input_tokens, cached_input_tokens, output_tokens, succeeded, created_at)
			VALUES ($1, $2, $3, 'm', $4, $5, $6, true, $7)`, kind, planID, uuid.New(), in, cached, out, at); err != nil {
			t.Fatal(err)
		}
	}
	insert(planA, "daily_brief", 1000, 800, 100, dayStart.Add(-time.Second)) // yesterday (UTC)
	insert(planA, "checklist_suggestion", 300, 128, 20, dayStart)            // at the start counts
	insert(planB, "video_terms", 200, 0, 30, dayStart.Add(23*time.Hour))

	got, err := repo.TokensSince(context.Background(), dayStart)
	if err != nil {
		t.Fatalf("TokensSince: %v", err)
	}
	if got != 550 {
		t.Fatalf("sum = %d, want 550 (320 + 230; yesterday and cached input excluded)", got)
	}
}
