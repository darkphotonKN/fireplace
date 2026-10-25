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

	// NotImplemented is 501 and is a DELIVERY statement, not a failure: the
	// operation is published in the contract with its success shape declared,
	// but its data path has not landed yet. A client can distinguish "this will
	// work later" from "this broke" — which a bare 500 cannot express.
	NotImplemented Code = "NOT_IMPLEMENTED"

	// Domain-specific. This one exists because errors[] cannot be populated
	// from a downstream gRPC failure (the wire carries only a string message),
	// so the field-level precision has to live in the code itself.
	// See FS-0002 §Requirements 16.
	ProfileNameEmpty Code = "PROFILE_NAME_EMPTY"

	// Plan members (FS-12ZMF R17). Both are 400: a well-formed request the
	// domain refused, not a malformed one (ADR-0005; malformed is 422).
	//
	// NoAccountForEmail: no Fireplace account uses the email the owner named.
	// Decided by the gateway, which holds the users table.
	NoAccountForEmail Code = "NO_ACCOUNT_FOR_EMAIL"
	// MemberIsOwner: the email names the plan's own owner. plan-service owns
	// the rule and answers InvalidArgument; the add-member operation attaches
	// this code itself rather than leaving it to message sniffing.
	MemberIsOwner Code = "MEMBER_IS_OWNER"

	// Guided suggestions (FS-DF4ZB R53), kept by the day plan that replaced
	// the daily brief (FS-02JBZ R22, R15). Decided by insights-service and sent
	// as an ErrorInfo detail (commongrpc.WithCode), so the gateway needs no
	// mapping of its own. The BRIEF_ prefix is the wire value clients already
	// switch on, so it stays.
	//
	// BriefDateOutOfRange (400): the day plan's date is more than one day from
	// the server's UTC date. Bounds cap evasion by date-hopping.
	BriefDateOutOfRange Code = "BRIEF_DATE_OUT_OF_RANGE"
	// BriefRefreshLimitReached (429): the 3 counted draws for this plan and
	// date are spent. The stored day plan is untouched; the next date resets.
	BriefRefreshLimitReached Code = "BRIEF_REFRESH_LIMIT_REACHED"

	// DailyScopeRetired (400): an item was created as, or updated to, scope
	// `daily` (FS-02JBZ R51). The enum value stays in the proto and the HTTP
	// contract so the request is well-formed; the refusal is a domain decision
	// (ADR-0005). Decided by plan-service and sent as an ErrorInfo detail.
	DailyScopeRetired Code = "DAILY_SCOPE_RETIRED"

	// LLMBudgetExhausted (429): today's (server UTC) global token budget,
	// LLM_DAILY_TOKEN_BUDGET, is spent, so no model call was made (FS-02JBZ
	// R33, ADR-0019). Retrying after the UTC day rolls over is correct;
	// retrying now is not, which is why it is not 503. Decided by
	// insights-service and sent as an ErrorInfo detail on ResourceExhausted.
	LLMBudgetExhausted Code = "LLM_BUDGET_EXHAUSTED"

	// DayPlanEntryNotTickable (400): the day plan entry cannot carry a day
	// tick (FS-02JBZ R26, D3): an unbound new step without the id of a
	// non-archived task created under its parent in the same plan, an entry
	// whose item is now archived or deleted, or an item id that contradicts
	// the entry's own. Decided by insights-service and sent as an ErrorInfo
	// detail on FailedPrecondition.
	DayPlanEntryNotTickable Code = "DAY_PLAN_ENTRY_NOT_TICKABLE"
)
