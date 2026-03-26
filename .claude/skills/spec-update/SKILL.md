---
name: spec-update
description: Update SPECIFICATION.md with a newly scoped feature from a Claude.ai session. Use when user says "update spec", "add feature to spec", "sync spec", "spec update", pastes scoping output, or says they finished scoping a feature in Claude.ai. Also trigger when user mentions bridging a scope-it or challenge-me session to the codebase. This skill ONLY updates the spec — it never writes PRDs, creates issues, or implements code.
---

# Spec Update

Update SPECIFICATION.md with a newly scoped feature. This is a handoff step between Claude.ai scoping sessions and the write-a-prd → prd-to-issues pipeline.

## CRITICAL RULES

- **ONLY edit SPECIFICATION.md** (and optionally docs/schema/ files)
- **NEVER create GitHub issues**
- **NEVER write PRDs**
- **NEVER write implementation code**
- **NEVER run write-a-prd or prd-to-issues**
- After updating, remind the user to run `/write-a-prd` as the next step

## Process

### 1. Get the scoped feature

The user will either:

- Paste scoping output directly (from a Claude.ai scope-it or challenge-me session)
- Describe the feature verbally and expect you to slot it in
- Reference a chat or document

If the input is vague, ask ONE clarifying question max. Assume the user has already thought this through.

### 2. Read current SPECIFICATION.md

```bash
cat SPECIFICATION.md
```

Understand the existing structure, sections, and conventions. Match the style exactly.

### 3. Determine what to update

A scoped feature can touch up to 6 sections of SPECIFICATION.md:

| Section                 | When to update                                                             |
| ----------------------- | -------------------------------------------------------------------------- |
| Domain Terms            | New terminology introduced                                                 |
| Features (Current)      | Always — add the feature checklist                                         |
| Data Model              | New tables or columns                                                      |
| API Surface             | New endpoints                                                              |
| Feature Detailed Design | If the feature has algorithm/logic worth documenting (like Smart Calendar) |
| Business Rules          | New rules or constraints                                                   |
| Edge Cases              | New edge cases identified                                                  |

Do NOT touch:

- Features (Implemented) — only the user moves things here when done
- Features (Future) — only remove items from here if they're now "Current"
- Non-Functional Requirements — rarely changes
- Package Structure — only add new packages if specified

### 4. Make surgical edits

Edit SPECIFICATION.md section by section. For each edit:

- Match existing formatting conventions exactly
- Keep entries concise — one line per checklist item, minimal prose
- For data models, use the same tree notation as existing tables
- If a feature was previously in "Future", move it to "Current"

### 5. Update schema doc (if new tables)

If the feature adds new database tables, create or update `docs/schema/<feature-name>.md` with the full table definition. Keep it lean — just the table, indexes, and constraints.

### 6. Confirm and suggest next step

After editing, output:

- Summary of what sections were updated
- **"Next step: run `/write-a-prd [feature name]` to create the PRD"**

Do NOT offer to do anything else. The pipeline is:

```
spec-update (you are here) → write-a-prd → prd-to-issues → tdd
```
