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

	// ServiceUnavailable is 503, NOT 500, and that distinction is why it is named
	// separately. 500 says "your request broke us" — a client must not retry,
	// because the same request will break us again. 503 says "we are temporarily
	// down": the request was fine, retry is correct, and the response can carry
	// Retry-After. Collapsing a downstream outage into 500 tells every client to
	// give up on a request that would have succeeded a second later.
	//
	// DEFINED BUT NOT YET WIRED: apierr.StatusFor has no codes.Unavailable case,
	// so an unreachable downstream still falls through to 500 here. Wiring it is a
	// published-contract change (this repo has a live openapi.yaml and gates), so
	// it needs its own slice rather than riding along with a vocabulary update.
	ServiceUnavailable Code = "SERVICE_UNAVAILABLE"

	// Domain-specific. This one exists because errors[] cannot be populated
	// from a downstream gRPC failure (the wire carries only a string message),
	// so the field-level precision has to live in the code itself.
	// See FS-0002 §Requirements 16.
	ProfileNameEmpty Code = "PROFILE_NAME_EMPTY"
)
