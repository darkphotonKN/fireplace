---
name: prd-to-issues
description: Break a PRD into independently-grabbable GitHub issues using vertical slices. Use when user says /prd-to-issues or wants to convert a PRD to implementation tasks.
---

# PRD to Issues

Breaks a PRD GitHub issue into task issues. Each task is a vertical slice.

## Vertical Slice

A thin cut through ALL layers delivering ONE complete behavior:

CORRECT: migration → model → repo → service → handler → tests = ONE working endpoint
WRONG: "all migrations" then "all models" then "all handlers"

## Process

### 1. Fetch the PRD
- /prd-to-issues #42 → fetch gh issue view 42
- /prd-to-issues (no number) → find recent PRD or ask

### 2. Read context
Check SPECIFICATION.md, CLAUDE.md, docs/schema/*.md

### 3. Draft vertical slices
- Each slice = ONE complete behavior through all layers
- First slice = foundation (migrations, models)
- Prefer thin slices

### 4. Present to user

Proposed breakdown for PRD #[number]:

1. **[Title]**
   - Scope: [description]
   - Type: AFK / HITL
   - Blocked by: none / #issue
   - User stories: [numbers]

AFK = autonomous, HITL = needs human decisions

Ask: "Is this granularity right?" Wait for approval.

### 5. Create issues

Use gh issue create for each. Template:

## Parent PRD
#[number]

## What to Build
[End-to-end behavior]

## Acceptance Criteria
- [ ] [Criterion]
- [ ] Tests pass

## Blocked By
#[number] or "None"

## User Stories Addressed
From PRD #[number]: [list]

## TDD Approach
- RED: [first test]
- GREEN: [what passes it]

### 6. Output summary
List all created issues with dependency graph.
Suggest: "Run /tdd #[number] when working on each issue."
