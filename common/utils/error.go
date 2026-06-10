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

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
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
	return strings.Contains(err.Error(), "duplicate key value")
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
	if err == nil {
		return false
	}

	errStr := err.Error()

	if errors.Is(err, sql.ErrConnDone) || errors.Is(err, context.DeadlineExceeded) ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "too many connections") {
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
