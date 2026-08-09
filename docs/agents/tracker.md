# Tracker config (read by spec-to-issues, code-review; written once at setup)
# One tracker per repo. Changing modes later is a migration (see docs/agents/README.md), not an edit-and-hope.
# Skills READ this file; only the setup/detection step WRITES it.

mode: local            # github | gitlab | local
# --- local mode only ---
issues_dir: docs/issues
labels: enhancement, feature, bug

# --- migration record ---
# Was `mode: github` (remote: origin, git@github.com:darkphotonKN/fireplace.git).
# Issues #46-#51 were migrated to docs/issues/I-0001..I-0006 and the GitHub issues
# deleted. See docs/issues/README.md. The repo has NO dedicated agent-workflow labels
# (no `ready-for-agent`, no `blocked`) — skills must not invent them; labels are a
# closed set and only what is listed above is used.
