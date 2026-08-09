// Command openapi emits the serialized contract to stdout.
//
// This is the ONLY thing that turns typed Go handler signatures into
// openapi.yaml. It constructs NO infrastructure — no database, no service
// registry, no cron jobs — which is exactly why config.RegisterAPI exists
// separately from config.SetupRouter. Without that split this command could not
// run in CI, and the regenerate-and-diff gate would be unenforceable.
//
//	make openapi   ->   go run ./cmd/openapi > openapi.yaml
package main

import (
	"fmt"
	"os"

	"github.com/darkphotonKN/fireplace/services/api-gateway/config"
	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	api := engine.Group("/api")

	// nil deps: registration reads type signatures, it never invokes handlers.
	spec := config.RegisterAPI(engine, api.Group(""), api.Group(""), config.APIDeps{})

	out, err := spec.OpenAPI().YAML()
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate spec: %v\n", err)
		os.Exit(1)
	}
	os.Stdout.Write(out)
}
