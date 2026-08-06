package config

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/apierr"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	authgw "github.com/darkphotonKN/fireplace/services/api-gateway/internal/gateway/auth"
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
		return p
	}
}

// APIDeps is the (currently tiny) set of collaborators serialized operations
// need. Nil is legal — spec generation constructs none of them.
type APIDeps struct {
	Profile authgw.ProfileClient
}

// RegisterAPI mounts a huma API on the protected group and registers every
// serialized operation. The doc surface mounts on the PUBLIC group so the
// contract stays browsable without a token, mirroring /swagger today
// (FS-0002 §Requirements 21).
//
// NOTE: huma's configured paths are relative to the group they mount on, which
// is why docs and operations take different groups.
func RegisterAPI(engine *gin.Engine, protected, public *gin.RouterGroup, deps APIDeps) huma.API {
	cfg := huma.DefaultConfig("Fireplace API", "1.0.0")
	cfg.Info.Description = "The serialized HTTP surface of the Fireplace api-gateway. " +
		"Endpoints not listed here are still described by the legacy OpenAPI 2.0 " +
		"document at /swagger and are migrated on touch (ADR-0002)."
	cfg.DocsPath = ""    // mounted separately, on the public group
	cfg.OpenAPIPath = "" // ditto

	// The operations mount on a group based at /api, but huma.Register records
	// the path it was GIVEN. Without this the spec advertises /users/profile
	// while the real URL is /api/users/profile — a spec that lies to clients.
	cfg.Servers = []*huma.Server{{URL: "/api", Description: "Gateway base path"}}

	// huma's default CreateHooks install a SchemaLinkTransformer that injects a
	// `$schema` member into every response body. That would publish an 8th field
	// on ProfileResponse, where FS-0002 §Requirements 6 specifies exactly seven.
	// The contract is a designed artifact; the framework does not get to add to it.
	cfg.CreateHooks = nil

	api := humagin.NewWithGroup(engine, protected, cfg)
	authgw.RegisterOperations(api, deps.Profile)

	// Public doc surface. Registered directly on the engine because huma's own
	// config paths would inherit the protected group's middleware.
	public.GET("/openapi.yaml", func(c *gin.Context) {
		b, err := api.OpenAPI().YAML()
		if err != nil {
			c.String(500, "spec error: %v", err)
			return
		}
		c.Data(200, "application/yaml", b)
	})
	public.GET("/docs", func(c *gin.Context) {
		c.Data(200, "text/html", []byte(docsHTML))
	})

	return api
}

// MountSerialized is the one line SetupRouter needs: it creates the serialized
// group (legacy auth middleware forked for problem+json) and registers on it.
func MountSerialized(engine *gin.Engine, api *gin.RouterGroup, public *gin.RouterGroup, deps APIDeps) huma.API {
	serialized := api.Group("")
	serialized.Use(auth.ProblemMiddleware())
	serialized.Use(auth.BridgeIdentity())
	return RegisterAPI(engine, serialized, public, deps)
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

const docsHTML = `<!doctype html><html><head><meta charset="utf-8">
<title>Fireplace API</title><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body><script id="api-reference" data-url="/api/openapi.yaml"></script>
<script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script></body></html>`
