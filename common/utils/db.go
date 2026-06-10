package commonhelpers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"
)

/**
* Database Utilities - Helper Functions
**/

/**
* Accepts a function that expects a transaction argument and helps wrap the function call
* with the initiation, passing, and error checking of that transaction.
**/
func ExecTx(ctx context.Context, db *sqlx.DB, fn func(tx *sqlx.Tx) error) (err error) {
	tx, txBeginErr := db.BeginTxx(ctx, nil)

	if txBeginErr != nil {
		slog.ErrorContext(ctx, "failed to begin transaction", "err", txBeginErr)
		return fmt.Errorf("Error when attempting to start transaction: %v", txBeginErr)
	}

	// handles panics and errors and rollback if they occur
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()

			slog.ErrorContext(ctx, "transaction rolled back due to panic", "panic", p)

			// re-throw panic after rollback
			panic(p)
		}

		if err != nil {
			slog.ErrorContext(ctx, "transaction failed, rolling back", "err", err)
			tx.Rollback()
		}
	}()

	// call the function passed in and provide the transaction to it
	err = fn(tx)

	if err != nil {
		return err
	}

	// no error, safe to commit
	if commitErr := tx.Commit(); commitErr != nil {
		slog.ErrorContext(ctx, "failed to commit transaction, rolling back", "err", commitErr)
		tx.Rollback() // rollback if commit fails

		return commitErr
	}

	return nil
}
