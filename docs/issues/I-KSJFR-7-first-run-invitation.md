---
id: I-KSJFR-7
status: in-progress
implements: FS-KSJFR
blocked_by: [I-KSJFR-2, I-KSJFR-6]
labels: [feature]
title: "FS-KSJFR slice 7: the first-run invitation"
---
Implements FS-KSJFR §Requirements (R43–R44, R47–R51), §Edge States

## What to Build

What a brand-new account sees. Without this slice the worst screen in the product is the first
one after signing up: the gate fires (nothing has been touched), the user skips, and three
hollow blocks appear.

**Zero plans renders one invitation panel instead of the three blocks (R43):** what a plan is,
and a single action that reopens the focus prompt **in place**. Not three empty states, not a
skeleton of a page the user hasn't earned yet.

**It does not navigate and does not duplicate the logged-out tour (R44).** The tour is FS-0003's
and belongs to unauthenticated visitors; this is a signed-in user who simply has nothing yet.
The action returns them to the prompt on the same route, consistent with D1's one-route-two-
states shape.

The three blocks reappear on their own once a plan exists — this is a zero-plan branch, not a
mode.

**This slice is HITL for its copy.** The FS fixes the behavior and deliberately does not write
the words; "what a plan is" is a product-voice decision. `client/docs/design-guideline.md`
carries the voice ("Start your plan now. Sit down by the fire.") — take the register from there
and get the copy confirmed before shipping.

Visual conformance per R47–R51.

## Acceptance Criteria

- [ ] With zero plans, the dashboard renders the invitation and none of the three blocks.
- [ ] The invitation's action reopens the focus prompt without navigating.
- [ ] The invitation does not reuse the logged-out tour's content.
- [ ] With one plan, the invitation is gone and the normal blocks render.
- [ ] The zero-plan path issues one request, not four.
- [ ] Copy is confirmed with the user before the slice closes.
- [ ] Tokens only; both themes correct.
- [ ] Tests pass.

## Blocked By

I-KSJFR-2 — this replaces the blocks the shell renders.
I-KSJFR-6 — "reopens the prompt" needs the gate to own which state `/` is in.

## Spec Reference

FS-KSJFR §Requirements (R43–R44), §Edge States (zero plans), §Decisions and why (D13).

## TDD Approach

- RED: a test mounting home with an empty plan list, asserting the invitation renders and
  Today / Continue / Your other plans do not.
- GREEN: the zero-plan branch.
- Then: the action returns to the prompt without a route change; one plan restores the blocks.
