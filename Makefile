# Repo-root Makefile. Per-service work lives in services/<svc>/Makefile;
# this one is for cross-service tasks: building everything, verifying build
# outputs, cleaning all artefacts, proto codegen.

SERVICES = api-gateway auth-service calendar-service insights-service plan-service

.PHONY: build-all clean-builds check-builds proto-gen list-services

# Build every service into its own bin/. Stops on first failure.
build-all:
	@for svc in $(SERVICES); do \
		echo "→ building services/$$svc"; \
		$(MAKE) -C services/$$svc build || exit 1; \
	done
	@echo "✓ all services built"

# Wipe every service's bin/ and tmp/ — handy before running check-builds.
clean-builds:
	@for svc in $(SERVICES); do \
		rm -rf services/$$svc/bin services/$$svc/tmp; \
	done
	@echo "✓ cleaned bin/ + tmp/ in all services"

# Run the audit: builds → verifies expected paths → verifies .gitignore covers
# everything → fails if any artefact escaped into a tracked path.
check-builds:
	@./scripts/check-build-locations.sh

# Regenerate all proto-derived .pb.go files (lives in common/).
proto-gen:
	@$(MAKE) -C common gen

list-services:
	@echo "$(SERVICES)"
