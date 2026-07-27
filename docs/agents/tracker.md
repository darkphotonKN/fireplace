# Tracker config (read by spec-to-issues, code-review; written once at setup)
# One tracker per repo. Changing modes later is a migration (see docs/agents/README.md), not an edit-and-hope.
# Skills READ this file; only the setup/detection step WRITES it.

mode: github
remote: origin         # git@github.com:darkphotonKN/fireplace.git
# Real repo labels usable for task issues. The repo has NO dedicated agent-workflow
# labels yet (no `ready-for-agent`, no `blocked`) — skills must not invent them.
# Create those labels in GitHub if you want spec-to-issues to apply them, then add
# them to this line (labels are a closed set; only what's listed here is used).
labels: enhancement, feature, bug
