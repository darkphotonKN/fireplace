package errorutils

import (
	"strings"
)

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
