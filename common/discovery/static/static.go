// Package static resolves services from configuration instead of a registry.
//
// Fireplace runs three services with fixed names on one Docker network
// (ADR-0009), where Docker's own DNS already resolves a container name to its
// address. Consul was solving a problem the platform no longer has: it cost a
// container, a dependency and registration wiring in every main(), and bought
// nothing that `plan-service:7103` does not.
//
// This implements the SAME discovery.Registry interface, so no call site
// changed when Consul was removed (ADR-0012 §4) — including the cached
// ClientConn behaviour in the gateway's gRPC clients. If the platform ever runs
// on more than one host, swapping the implementation back is the whole change.
package static

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// Registry answers Discover from the environment. Registration and health are
// no-ops: there is nothing to register with, and a service that is not running
// fails at dial time instead — which is where a static topology reports it.
type Registry struct{}

func NewRegistry() *Registry { return &Registry{} }

// EnvKey is the variable consulted for a service. "plan" → PLAN_SERVICE_ADDR.
// Exported so callers can name the variable in an error or a startup log
// without re-deriving the convention.
func EnvKey(serviceName string) string {
	return strings.ToUpper(strings.ReplaceAll(serviceName, "-", "_")) + "_SERVICE_ADDR"
}

// Register is a no-op. Addresses are configuration, not runtime state.
func (r *Registry) Register(ctx context.Context, instanceID, serviceName, hostPort string) error {
	return nil
}

// Deregister is a no-op, for the same reason as Register.
func (r *Registry) Deregister(ctx context.Context, instanceID, serviceName string) error {
	return nil
}

// HealthCheck is a no-op that must return nil: callers run it on a ticker and
// treat any error as fatal. There is no registry to report liveness to, and
// reporting an error here would kill a perfectly healthy process.
func (r *Registry) HealthCheck(instanceID, serviceName string) error {
	return nil
}

// Discover returns the single configured address for serviceName.
//
// A missing variable is an error rather than a guessed default: defaulting to
// localhost would turn a misconfigured container into a connection refused at
// first RPC, minutes later and far from the cause.
func (r *Registry) Discover(ctx context.Context, serviceName string) ([]string, error) {
	key := EnvKey(serviceName)
	addr := os.Getenv(key)
	if addr == "" {
		return nil, fmt.Errorf("static discovery: %s is not set, cannot resolve service %q", key, serviceName)
	}
	return []string{addr}, nil
}
