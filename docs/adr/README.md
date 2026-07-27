# Architecture Decision Records (ADR)

Immutable records of **constraints and non-obvious decisions** — the reasoning behind a
tradeoff, a technology choice, or a boundary, captured so it isn't relitigated later.

## Format (Nygard)

Each ADR is `NNNN-short-slug.md` with: **Title, Status** (proposed / accepted /
superseded), **Context, Decision, Consequences**, and a date.

## Rules

- **Numbering** — zero-padded, sequential, allocated the same way as `docs/specs/`
  (`FS-NNNN`). ADR and FS numbers are independent sequences.
- **Immutable** — never retro-edit an accepted ADR. To change a decision, write a new
  ADR that supersedes it, noting the superseding/superseded numbers in both.
- **Check before architectural changes** — an architectural change should be consistent
  with the accepted ADRs, or explicitly supersede the one it contradicts.

Written by the `record-decision` skill (or by hand in the same format).
