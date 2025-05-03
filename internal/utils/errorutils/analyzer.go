package errorutils

import (
	"database/sql"
	"errors"
)

/**
* Analyzes which type of custom error an error is and returns the
* appropriate error type. If the error is a new type then return it directly.
**/
func AnalyzeDBErr(err error) error {
	if err == nil {
		return nil
	}
	// match custom error types
	if IsDuplicateError(err) {
		return ErrDuplicateResource
	}
	if IsConstraintViolation(err) {
		return ErrConstraintViolation
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}

	// unexpected errors
	return err
}
