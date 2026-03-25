---
name: write-a-prd
description: Create a PRD through codebase exploration and submit as a GitHub issue. Use when user wants to write a PRD, plan a new feature, or says /write-a-prd.
---

# Write a PRD

Creates a Product Requirements Document for a feature, then submits it as a GitHub issue.

## Process

1. **Check for existing spec** - Read SPECIFICATION.md and docs/schema/*.md for the feature.
   - If detailed spec exists: Use it as source of truth. Only ask about gaps or ambiguities.
   - If no spec exists: Interview the user thoroughly.

2. **Detect repo structure** - Monorepo or single repo? Note which services/modules are affected.

3. **Locate context** - Read SPECIFICATION.md, CLAUDE.md, and docs/schema/*.md if they exist.

4. **Interview the user** (if needed) - Ask for detailed problem description. Explore repo to verify assertions. Ask about alternatives considered. Be thorough about scope.

5. **Hammer out scope** - What we build vs what we DON'T build.

6. **Write the PRD** - Use the template below. Be exhaustive on user stories.

7. **Create GitHub issue** - Run gh issue create --title "PRD: [Feature Name]" --body "[content]". Just create it.

## PRD Template

## Problem Statement
[What problem does this feature solve? User's perspective.]

## Solution
[High-level description. 2-3 sentences max.]

## User Stories
[EXHAUSTIVE numbered list. Format: "As a [actor], I want [capability], so that [benefit]"]

1. As a ...
2. As a ...
(typically 15-30 stories)

## Implementation Decisions
- Affected services/packages
- Database tables
- API endpoints
- State transitions
- External services

## Testing Approach
- Key behaviors to test
- Edge cases

## Out of Scope
[What this PRD does NOT cover]

## Open Questions
[Anything needing clarification]

## After Creation

Output the GitHub issue URL and suggest: "Run /prd-to-issues #[number] to break this into task issues"
