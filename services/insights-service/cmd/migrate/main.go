// Command migrate applies this service's schema migrations and exits.
//
// It runs as `<db>_owner`, the ONLY role holding DDL (ADR-0010 §2), and is a
// discrete step that completes before the application starts (ADR-0010 §4).
// Migrations used to run in-process at boot, which forced the runtime role to
// hold CREATE permanently — so an injected or compromised process could DROP
// TABLE, and two replicas would race on startup.
//
// Failure semantics are the reason this is a separate program: any error exits
// NON-ZERO and the caller — a Makefile target now, a compose init container
// after I-0040 — stops before the service is started against a half-migrated
// schema. A partially applied migration leaves golang-migrate's version dirty;
// `make migrate-status` reports it and `make migrate-fix version=N` clears it
// once the cause is understood. It is deliberately not cleared automatically.
//
// Usage: DB_OWNER_USER=... DB_OWNER_PASSWORD=... go run ./cmd/migrate [up|down]
package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	direction := "up"
	if len(os.Args) > 1 {
		direction = os.Args[1]
	}

	// The migrations path is resolved from the env so this works both from the
	// service directory (where `make` runs it) and from a container WORKDIR.
	source := "file://" + env("MIGRATIONS_PATH", "migrations")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("DB_OWNER_USER"),
		os.Getenv("DB_OWNER_PASSWORD"),
		env("DB_HOST", "localhost"),
		env("DB_PORT", "5500"),
		os.Getenv("DB_NAME"),
	)

	m, err := migrate.New(source, dsn)
	if err != nil {
		log.Fatalf("migrate: open %s: %v", source, err)
	}

	switch direction {
	case "up":
		err = m.Up()
	case "down":
		err = m.Down()
	default:
		log.Fatalf("migrate: unknown direction %q (want up or down)", direction)
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("migrate: %s: %v", direction, err)
	}

	version, dirty, verr := m.Version()
	if verr != nil && !errors.Is(verr, migrate.ErrNilVersion) {
		log.Fatalf("migrate: read version: %v", verr)
	}
	if dirty {
		log.Fatalf("migrate: schema is DIRTY at version %d — a migration failed partway; "+
			"inspect it, then clear with 'make migrate-fix version=%d'", version, version)
	}
	log.Printf("✓ migrations %s — %s at version %d", direction, os.Getenv("DB_NAME"), version)
}
