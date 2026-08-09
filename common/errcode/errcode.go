// Package errcode is the platform-wide domain error vocabulary (ADR-0004).
//
// HTTP status is the COARSE routing signal; a Code is the precise one. Two
// failures that are both "the request was invalid" share a 400 and differ by
// Code. Frontend code switches on Code, never on the human-readable detail
// string — that string is prose and is explicitly allowed to change.
//
// STABILITY: a Code is contract. Removing or repurposing one is a breaking
// change, reviewed like removing a response field. ADDING one is non-breaking,
// so handlers may become more specific over time without a coordinated release.
package errcode

type Code string

const (
	// Generic — map 1:1 to the gRPC status classes apierr already understands.
	Unauthenticated  Code = "UNAUTHENTICATED"
	ValidationFailed Code = "VALIDATION_FAILED"
	NotFound         Code = "NOT_FOUND"
	AlreadyExists    Code = "ALREADY_EXISTS"
	Forbidden        Code = "FORBIDDEN"
	Internal         Code = "INTERNAL_ERROR"

	// Domain-specific. This one exists because errors[] cannot be populated
	// from a downstream gRPC failure (the wire carries only a string message),
	// so the field-level precision has to live in the code itself.
	// See FS-0002 §Requirements 16.
	ProfileNameEmpty Code = "PROFILE_NAME_EMPTY"
)
