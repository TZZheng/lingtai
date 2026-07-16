# TUI release migration

This is the one repo-owned migration document at the stable path
`migration/migration.md`. At each release, future releases REPLACE this file
with that release's migration instructions. Git commits and release tags
preserve the prior document versions. Do not create a second migration table.

## m040 — `bash` capability to `shell`

- **Product:** `lingtai-tui`
- **Migration version:** `40` (the release's migration registry entry)
- **Release tag:** record the exact repository release tag here at release time,
  using the repository's actual `vX.Y.Z` tag convention; this branch does not
  prove a next tag and therefore invents none.
- **Scope:** existing per-agent `init.json` capability maps and TUI preset
  setup/editor read/write paths.
- **Read compatibility:** a legacy `bash` entry is accepted and its
  configuration object is moved unchanged to `shell`.
- **Conflict behavior:** if both keys exist with different configuration
  objects, migration and writes fail closed without merging or rewriting the
  conflicting file. Malformed JSON or present non-object `manifest` or
  `manifest.capabilities` values also fail closed before the version advances.
- **Canonical writes:** new preset, setup, editor, and rehydration writes emit
  `shell` only. Identical `bash` and `shell` values choose the existing
  canonical `shell` value deterministically.
