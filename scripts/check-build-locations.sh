#!/usr/bin/env bash
#
# check-build-locations.sh
#
# Verifies that every service in services/ builds to its expected binary path
# AND that no build artefact ends up in a tracked or non-gitignored location.
#
# What it checks (in order):
#   1. Pre-clean: removes bin/ and tmp/ in every service so we test from scratch.
#   2. Snapshots `git status --porcelain` before the build.
#   3. Runs `make build` in each service.
#   4. Confirms the expected binary path exists.
#   5. Confirms each produced binary is matched by .gitignore (git check-ignore).
#   6. Confirms `git status --porcelain` is unchanged from the pre-build snapshot
#      — i.e. no untracked artefact escaped into a non-gitignored path.
#
# Usage: ./scripts/check-build-locations.sh
#        or: make check-builds (from repo root)
#
# Exits non-zero on any failure with a clear message.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${ROOT}"

# (service-dir-relative-to-root, expected-binary-relative-to-service-dir)
SERVICES=(
  "services/api-gateway:bin/app"
  "services/calendar-service:bin/calendar-service"
  "services/insights-service:bin/insights-service"
  "services/plan-service:bin/plan-service"
)

red()    { printf '\033[0;31m%s\033[0m\n' "$1"; }
green()  { printf '\033[0;32m%s\033[0m\n' "$1"; }
yellow() { printf '\033[0;33m%s\033[0m\n' "$1"; }

# 1. pre-clean
echo "→ cleaning bin/ and tmp/ across services..."
for entry in "${SERVICES[@]}"; do
  svc="${entry%%:*}"
  rm -rf "${ROOT}/${svc}/bin" "${ROOT}/${svc}/tmp"
done

# 2. snapshot git state pre-build
before="$(git status --porcelain || true)"

# 3. build each service
echo "→ building each service..."
for entry in "${SERVICES[@]}"; do
  svc="${entry%%:*}"
  bin="${entry##*:}"
  printf '  • %-30s → %s\n' "${svc}" "${bin}"
  if ! (cd "${ROOT}/${svc}" && make build >/dev/null 2>&1); then
    red "✗ build failed for ${svc}"
    exit 1
  fi
done

# 4. confirm expected binary exists per service
echo "→ verifying binaries landed at the expected paths..."
fail=0
for entry in "${SERVICES[@]}"; do
  svc="${entry%%:*}"
  bin="${entry##*:}"
  full="${svc}/${bin}"
  if [[ ! -x "${ROOT}/${full}" ]]; then
    red "✗ ${full} is missing or not executable"
    fail=1
  fi
done
if [[ ${fail} -ne 0 ]]; then
  exit 1
fi

# 5. each produced binary must be gitignored
echo "→ verifying each binary is gitignored..."
fail=0
for entry in "${SERVICES[@]}"; do
  svc="${entry%%:*}"
  bin="${entry##*:}"
  full="${svc}/${bin}"
  if ! git check-ignore -q "${full}" 2>/dev/null; then
    red "✗ ${full} is NOT covered by .gitignore — add a pattern that matches it"
    fail=1
  fi
done
if [[ ${fail} -ne 0 ]]; then
  exit 1
fi

# 6. confirm no escape: git status must be unchanged
after="$(git status --porcelain || true)"
if [[ "${before}" != "${after}" ]]; then
  red "✗ Build produced files that escaped .gitignore. New entries in git status:"
  diff <(echo "${before}") <(echo "${after}") | grep -E '^[<>]' | sed 's/^/    /'
  echo ""
  yellow "Hint: any path showing up above means a build artefact landed somewhere"
  yellow "      .gitignore doesn't cover. Either fix the Makefile / .air.toml to"
  yellow "      write into bin/ or tmp/, or add a .gitignore pattern for it."
  exit 1
fi

green "✓ All 5 services build to their expected bin/ path."
green "✓ All produced binaries are gitignored."
green "✓ Nothing escaped into git status."
