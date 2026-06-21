package commonhelpers

/*
Commonly shared error helpers utilities.
*/

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"database/sql/driver"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/jackc/pgx/v5/pgconn"
)

/**
* Analyzes which type of custom error an error is and returns the
* appropriate error type. If the error is a new type then return nil and let wrapper
* handle it directly.
**/
func analyzeDBErr(err error) error {
	if err == nil {
		return nil
	}

	// match custom error types
	if IsDuplicateError(err) {
		return commonconstants.ErrDuplicateResource
	}
	if IsConstraintViolation(err) {
		return commonconstants.ErrConstraintViolation
	}
	if IsTransientError(err) {
		return commonconstants.ErrTransient
	}
	if errors.Is(err, sql.ErrNoRows) {
		return commonconstants.ErrNotFound
	}

	// unexpected errors
	return nil
}

/**
* Helper function to determine if an error is a "duplicate item" error.
**/
func IsDuplicateError(err error) bool {
	if err == nil {
		return false
	}

	var pgErrors *pgconn.PgError
	errors.As(err, &pgErrors)

	if pgErrors != nil && pgErrors.Code == "23505" {
		return true
	}

	return false
}

/**
* Helper function to determine if an error is from an attempt to insert without
* following column constraints.
**/
func IsConstraintViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "violates check constraint")
}

/**
* Helper that detemrines if an error is considered a transient error that means we could retry consuming the event message and running the subsequent processes.
**/
func IsTransientError(err error) bool {
	// context errors
	contextErrors := errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)

	// sql and sql driver errors
	sqlErrors := errors.Is(err, sql.ErrConnDone) || errors.Is(err, driver.ErrBadConn)

	// postgres specific errors
	var pgErrors *pgconn.PgError
	isPgTransientErr := false

	// sets error to the matching error if any error inside "err" matches a case
	hasPgErr := errors.As(err, &pgErrors)

	if hasPgErr {
		switch pgErrors.Code {
		// pg transient errors
		case "40001", "40P01", "57P03", "08000", "08003", "08006", "08001", "08004":
			isPgTransientErr = true
		}
	}

	if contextErrors || sqlErrors || isPgTransientErr {
		return true
	}

	return false
}

// wrapDBErr is the repo boundary translation point: it converts infrastructure
// errors (sql.ErrNoRows, duplicate keys, constraint violations, transient
// failures) into domain sentinels via AnalyzeDBErr, and wraps anything else
// with the repo name + operation for context. It never logs and never decides
// transport status.
// repoName - where the error occured, e.g. users repo
// op - the operation that was attempted to run
// err - the error
func WrapDBErr(repoName string, op string, err error) error {
	if err == nil {
		return err
	}

	// matches a sentinel error
	if analyzedDBEr := analyzeDBErr(err); analyzedDBEr != nil {
		// return only sentinel
		return analyzedDBEr
	}

	// return with context wrapped
	return fmt.Errorf("error occured in %s during %s: %w", repoName, op, err)
}
