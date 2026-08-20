package config

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/apierr"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	authgw "github.com/darkphotonKN/fireplace/services/api-gateway/internal/gateway/auth"
	calendargw "github.com/darkphotonKN/fireplace/services/api-gateway/internal/gateway/calendar"
	plangw "github.com/darkphotonKN/fireplace/services/api-gateway/internal/gateway/plan"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/insights"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/notes"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/useranalytics"
	"github.com/gin-gonic/gin"
)

// The serialized (typed) API surface — ADR-0002 plane 1.
//
// THE SPLIT THAT MAKES CODE-FIRST GENERATION POSSIBLE:
//
//	SetupRouter(db, registry)  = WIRING. Needs Postgres, Consul, an API key,
//	                             and starts cron jobs. Cannot run in CI.
//	RegisterAPI(engine, deps)  = REGISTRATION. Pure. Reads type signatures only.
//	                             cmd/openapi calls THIS to emit the spec.
//
// Without the split, generating openapi.yaml would mean booting the whole
// gateway — so the spec could never be regenerated in CI, and the
// regenerate-and-diff gate ADR-0002 mandates would be unenforceable.
// Handlers may be nil: registration never invokes them.

func init() {
	// Every huma-generated error becomes RFC 9457 problem+json with a domain
	// code (ADR-0004). This covers huma's own errors (validation, 404, 422) —
	// handler-returned errors already come back as *apierr.Problem.
	huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
		p := apierr.ProblemFor("request", nil)
		p.Status = status
		p.Title = statusTitle(status)
		p.Detail = msg
		p.Code = apierr.CodeForStatus(status)
		p.Errors = []*huma.ErrorDetail{}
		return p
	}
}

// APIDeps is the (currently tiny) set of collaborators serialized operations
// need. Nil is legal — spec generation constructs none of them.
type APIDeps struct {
	Profile    authgw.ProfileClient
	Users      authgw.UsersClient
	Plans      plangw.PlansClient
	Checklists plangw.ChecklistsClient

	// Notes is a gateway-LOCAL domain: unlike the fields above it is not a
	// downstream client but this service's own service object, paired with the
	// ownership checker its operations authorize against.
	Notes          notes.NotesService
	NotesOwnership notes.PlanOwnership

	// insights is served in-process today; these two seams are what the gRPC
	// repoint will swap out (ADR-0002 slice ⓪ established the contract first).
	// Two, not one: the router builds a separate service per generator.
	Suggestions      insights.SuggestionsService
	VideoSuggestions insights.VideoSuggestionsService

	// Analytics may be nil — its operation answers 501 before consulting it.
	Analytics useranalytics.AnalyticsService

	Calendar calendargw.CalendarClient
}

// Doc surface paths. Public by design: the contract is browsable without a
// token, and every operation appears whether or not it needs one.
const (
	// BearerAuth names the security scheme protected operations reference.
	BearerAuth = "bearerAuth"

	OpenAPIPath = "/api/openapi"
	DocsPath    = "/api/docs"
)

// Protected adapts the gateway's gin auth middleware into a per-operation Huma
// middleware, and bridges the identity it sets onto the typed handler context.
//
// Why per-operation rather than scoping the whole API to a gin group: huma
// registers on the ENGINE here, so one document can hold public AND protected
// operations. humagin.NewWithGroup — which this used to use — scopes every
// operation to a single group, which cannot express a surface with both, and
// forced the docs onto a separately hand-mounted route to escape the auth
// middleware.
//
// mw is invoked directly rather than through gin's chain: the middleware
// writes-and-aborts on failure and falls through on success, so there is no
// chain to re-enter.
func Protected(mws ...gin.HandlerFunc) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		gc := humagin.Unwrap(ctx)
		for _, mw := range mws {
			if mw == nil {
				return // fail closed
			}
			mw(gc)
			if gc.IsAborted() {
				return
			}
		}
		next(ctx)
	}
}

// Secured is the Security value every protected operation must carry. Kept next
// to Protected: an operation that runs the middleware but omits this publishes a
// contract saying it is public, then answers 401.
var Secured = []map[string][]string{{BearerAuth: {}}}

// RegisterAPI mounts a huma API on the ENGINE and registers every serialized
// operation. Handlers may be nil: registration never invokes them.
func RegisterAPI(engine *gin.Engine, deps APIDeps, protect func(huma.Context, func(huma.Context))) huma.API {
	cfg := huma.DefaultConfig("Fireplace API", "1.0.0")
	cfg.Info.Description = "The HTTP surface of the Fireplace api-gateway. " +
		"This document is generated from the typed handlers and is the only " +
		"description of the API — there is no second, hand-written one. " +
		"Endpoints not listed here are not yet serialized (FS-0004); they are " +
		"reachable but undocumented until their group lands."

	// huma's built-in doc surface, served on the engine and therefore public.
	// Replaces a hand-rolled HTML page: the renderer is now huma's default
	// (Stoplight Elements), which is also the house that makes Spectral — the
	// linter already in this repo's gates.
	cfg.OpenAPIPath = OpenAPIPath
	cfg.DocsPath = DocsPath

	// Operations declare their FULL path, so the document describes the real
	// URL without a Servers indirection.
	//
	// huma's default CreateHooks install a SchemaLinkTransformer that injects a
	// `$schema` member into every response body. That would publish an 8th field
	// on ProfileResponse, where FS-0002 §Requirements 6 specifies exactly seven.
	// The contract is a designed artifact; the framework does not get to add to it.
	cfg.CreateHooks = nil

	// Declare HOW to authenticate, in the document itself. Without this the
	// contract lists protected operations but never says they need a token — no
	// padlock in the docs, nowhere to paste credentials in try-it, and a
	// generated client unaware auth exists.
	cfg.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		BearerAuth: {
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
			Description:  "Access token sent as `Authorization: Bearer <token>`.",
		},
	}

	api := humagin.New(engine, cfg)
	authgw.RegisterOperations(api, deps.Profile, protect, Secured)
	authgw.RegisterUsersOperations(api, deps.Users, protect, Secured)
	plangw.RegisterPlanOperations(api, deps.Plans, protect, Secured)
	plangw.RegisterChecklistOperations(api, deps.Checklists, protect, Secured)
	notes.RegisterNotesOperations(api, deps.Notes, deps.NotesOwnership, protect, Secured)
	insights.RegisterInsightsOperations(api, deps.Suggestions, deps.VideoSuggestions, protect, Secured)
	useranalytics.RegisterAnalyticsOperations(api, deps.Analytics, protect, Secured)
	calendargw.RegisterCalendarOperations(api, deps.Calendar, protect, Secured)
	return api
}

// MountSerialized is the one line SetupRouter needs.
func MountSerialized(engine *gin.Engine, deps APIDeps) huma.API {
	return RegisterAPI(engine, deps, Protected(auth.ProblemMiddleware(), auth.BridgeIdentity()))
}

func statusTitle(status int) string {
	switch status {
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 409:
		return "Conflict"
	case 422:
		return "Unprocessable Entity"
	default:
		return "Internal Server Error"
	}
}
