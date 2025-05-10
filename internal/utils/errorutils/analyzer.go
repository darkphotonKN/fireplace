package errorutils

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/darkphotonKN/fireplace/internal/constants"
)

/**
* Analyzes which type of custom error an error is and returns the
* appropriate error type. If the error is a new type then return it directly.
*
* Also helps check if there are no rows affected and returns an error in those cases.
**/
func AnalyzeDBErr(err error) error {
	if err == nil {
		return nil
	}
	// match custom error types
	if IsDuplicateError(err) {
		return constants.ErrDuplicateResource
	}
	if IsConstraintViolation(err) {
		return constants.ErrConstraintViolation
	}
	if errors.Is(err, sql.ErrNoRows) {
		return constants.ErrNotFound
	}

	// unexpected errors
	return err
}

/**
* Analyzes both the error and the SQL result to provide more detailed error information.
* This is useful for operations where the query might succeed but affect no rows.
**/
func AnalyzeDBResults(err error, result sql.Result) error {
	// check for standard errors
	if err != nil {
		return AnalyzeDBErr(err)
	}

	if result == nil {
		return nil
	}

	// check for errors based on sql results
	rowsAffected, err := result.RowsAffected()

	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		fmt.Printf("There were no errors but no rows affected by the sql query.\n")
		return constants.ErrNoRowsAffected
	}

	return nil
}
