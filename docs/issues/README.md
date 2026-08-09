# Issues (local tracker mode)

Task issues for this repo, one file per issue: `I-NNNN-short-slug.md` — zero-padded and
sequential, the same numbering style as `docs/adr/` and `docs/specs/`.

Written by `spec-to-issues`; read by `develop` and `code-review`. Schema authority:
`docs/agents/README.md`. Backend selection: `docs/agents/tracker.md` (`mode: local`).

## Rules

- **Status changes edit frontmatter in place.** Never move or rename an issue file — every
  reference would break. Done issues stay here; `status: done` is the archive.
- **`implements:` is the durable anchor.** `I-NNNN` is backend-local and may change in a
  migration; `FS-NNNN` (or `ADR-NNNN`, for infrastructure tasks with no feature spec) does not.
- **`blocked_by:` lists issue ids**, and the same dependency is restated in the body's
  `## Blocked By` section for human readers.

## Migration note

These were migrated from GitHub issues (`darkphotonKN/fireplace#46–#51`) when the repo moved
to `mode: local`. Each file records its origin in `migrated_from:`. Cross-references in issue
bodies were rewritten from `#N` to `I-NNNN`; `Implements FS-NNNN` anchors needed no rewriting,
which is the point of anchoring to the spec rather than to the tracker.

Migrating back to a real tracker is the reverse and equally mechanical: iterate `I-*.md` with
`status != done`, publish each via the target backend, write the new ref into frontmatter, and
flip `mode` in `docs/agents/tracker.md`.
