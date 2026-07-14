# Feature Specs (FS-NNNN)

Tier-2 detailed specs for our **two-tier spec system**.

## The two tiers

- **Tier 1 — `SPECIFICATION.md`** (thin, living, always-loaded): the current truth of what the system does. One line per capability, grouped by bounded context, each with a status checkbox and a pointer to its feature spec, e.g.
  `- [x] Drag-reorder in tree view → FS-0042`
  Hard budget ~200–300 lines. When a section bloats past budget, that signals a bounded context wants its own file — the spec does not grow. (Specs are per-service under `services/<name>/SPECIFICATION.md`; feature specs here are the shared, globally-numbered work orders.)
- **Tier 2 — feature specs (this folder)** (deep, write-once, loaded only when working that feature).

## Numbering & references

- Files live here as `NNNN-short-slug.md` — zero-padded, sequential, allocated the same way as `docs/adr/`.
- Reference them everywhere as `FS-NNNN` (in `SPECIFICATION.md` lines, issue bodies, PRs, commits).
- A feature spec contains: **Summary, Requirements, User Stories, Acceptance Criteria, Edge States, Out of Scope**, plus a header linking back to its `SPECIFICATION.md` entry and any related ADRs.
- The thin line points; the FS elaborates. Never copy content between them.

## Write-once rule

Feature specs are **work orders, not living documents**. Once the feature ships, the durable truth is the one line in `SPECIFICATION.md`; the FS file becomes a historical record, exactly like an ADR. **Never retro-edit a shipped FS** — write a new one that supersedes it, noting the superseding/superseded FS numbers in both.

## Division of checking labor

- `code-review` (per-PR, deep) reads only the FS file(s) a change references.
- `spec-audit` (periodic, thin) reads only `SPECIFICATION.md`.
