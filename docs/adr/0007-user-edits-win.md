# ADR-0007 — User edits win: AI regeneration may add or propose, never overwrite

Status: accepted
Date: 2026-08-19
Scope: root — governs every feature in which generated content lands in a table users can edit
Realized by: FS-0006 (first feature to encode it); pre-dates it in intent

## Context

Fireplace generates content into tables users own and edit: checklist items derived from
insights, daily suggestions accepted into a plan, notes, and the plan draft itself. Every one
of these has the same shape — a generator writes rows, then a human reshapes them, then the
generator runs again.

The question of what the second run may do to the first run's output has now surfaced
independently in **three** features, and been re-argued each time from scratch. That is the
signal that it is a constraint rather than a per-feature design choice: it holds across the
platform, so deciding it once is strictly cheaper than deciding it three more times.

The failure it guards against is specific and unrecoverable. A user spends real effort
rewording a generated item, adding dates, nesting it under a parent, marking it done — and a
regeneration silently replaces or deletes it. Unlike a bad generation, which the user can
simply ignore, destroyed edits cannot be recovered by the user, and the loss is invisible until
they go looking for work they know they did.

The counter-pressure is real and was weighed: a "never overwrite" rule means regenerating a
plan the user has half-edited produces a *mixture* — new proposals sitting alongside old
touched rows — rather than a clean result. Some users will want the clean sweep. That case is
served by an explicit destructive action the user chooses, not by regeneration's default.

Recorded without adversarial review — the decision was locked collaboratively during the
FS-0006 scoping session and not run through `challenge-me`.

## Decision

**Once a user modifies AI-generated content, that content is theirs.**

A regeneration pass may:

- **add** new rows
- **propose** changes for the user to accept or reject

A regeneration pass may **not**:

- **overwrite** a row the user has touched
- **delete** a row the user has touched

"Touched" must be recorded, not inferred. A feature that generates into a user-editable table
carries the state needed to distinguish an untouched generated row from an edited one; how that
state is represented is the feature's design, but its existence is not optional.

Untouched generated rows are not protected by this ADR. A regeneration may freely replace
content the user has never engaged with — that is what makes regeneration useful at all.

Destructive bulk operations (clear all, start over) remain available as **explicit user
actions**. This ADR governs what regeneration does by default, not what the user may ask for.

## Consequences

**Good**

- The worst outcome — silent, unrecoverable loss of human work — is structurally impossible
  rather than dependent on each feature getting it right.
- Users can safely regenerate. Without this rule, regeneration is a gamble, so users avoid it,
  and the feature that cost the most to build gets used the least.
- Removes a recurring design argument. Three features have paid for it; the fourth does not.

**Bad / accepted**

- **Regeneration produces mixtures.** A partly-edited plan regenerated yields protected rows
  interleaved with fresh proposals. That is harder to present well than a clean replacement,
  and the presentation problem is pushed onto every consuming feature.
- **Every generating feature carries provenance state.** A column, a flag, or a comparison
  against the generated original — some cost is unavoidable, and it is paid per feature.
- **"Touched" needs a definition per surface**, and the definitions will not be identical.
  Editing text is clearly touching; checking a checkbox, dragging a date, or reordering are
  judgment calls the feature must make and state.
- **Accept/reject flows are more work than overwrite.** "Propose" implies a review surface;
  features that would have been a one-line regenerate now need a way to show a diff.
