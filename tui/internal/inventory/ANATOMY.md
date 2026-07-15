---
related_files:
  - ANATOMY.md
  - tui/ANATOMY.md
  - tui/internal/tui/ANATOMY.md
  - tui/internal/processscan/ANATOMY.md
  - tui/internal/fs/ANATOMY.md
  - tui/internal/inventory/inventory.go
  - tui/internal/inventory/inventory_test.go
  - tui/list_common.go
  - tui/list_common_test.go
  - tui/list_unix.go
  - tui/list_windows.go
  - tui/purge_common.go
  - tui/internal/tui/projects.go
  - tui/internal/tui/projects_test.go
maintenance: |
  Keep related_files as repo-relative paths to real files. Include neighboring
  ANATOMY.md files so the anatomy graph stays connected rather than isolated;
  anatomy links must be bidirectional. If you create a new ANATOMY.md, copy this
  maintenance field. If you notice drift between this anatomy and the code,
  report it. See lingtai-dev-guide for details.
---

# inventory

> **Maintenance:** see `lingtai-tui-anatomy` (at `~/.lingtai-tui/utilities/lingtai-tui-anatomy/SKILL.md`). Update this file in the same commit as code changes.

`inventory` is the typed running-agent inventory shared by the `lingtai-tui list` CLI and the interactive `/projects` switcher. It sits below both callers: process visibility comes from `internal/processscan`, while filesystem enrichment comes from `internal/fs`. It writes nothing.

## Components

| Component | File | Purpose |
|---|---|---|
| `Options`, `Snapshot`, `Group`, `Record`, `EnterabilityReason`, `AgentIdentity` | `tui/internal/inventory/inventory.go:31-100` | Public typed API for process-visible agents. `Record.ManifestAddressVerified` distinguishes a nonempty address read from `.agent.json` from the display-only basename fallback stored in `Address` (`tui/internal/inventory/inventory.go:70-87`). |
| `Scan` / `FromProcesses` | `tui/internal/inventory/inventory.go:105-167` | Scan the process table or convert supplied process rows into enriched, duplicate-collapsed, sorted inventory. `FromProcesses` is the deterministic test seam. |
| `NormalizePath`, `IdentityFor`, `AgentDirInFilter`, `ProjectFromAgentDir` | `tui/internal/inventory/inventory.go:169-212` | Shared lexical path normalization, stable identity, filtering, and process-to-project derivation. Agent directories with spaces are normal paths. |
| `enrichRecord`, `RoleFor`, `enterability` | `tui/internal/inventory/inventory.go:224-299` | Seed display fallbacks from the agent-directory basename, enrich from manifest/status/filesystem metadata, and derive global project-visit enterability. A readable manifest with an empty address keeps its fallback and no `ReadError`, but does not acquire verified-address provenance (`tui/internal/inventory/inventory.go:224-240`). |
| `collapseByAgentDir` | `tui/internal/inventory/inventory.go:301-324` | Collapse duplicate visible processes for the same agent dir, preferring the PID advertised by `.status.json` when available. |
| `sortRecords` / `groupRecords` | `tui/internal/inventory/inventory.go:350-401` | Deterministic project, role, display-name, path, PID sorting and project grouping. |
| Display-only address provenance contract | `tui/internal/inventory/inventory_test.go:161-186` | Proves that a successfully read empty manifest address remains visible through basename/name/nickname fallbacks while `ManifestAddressVerified` stays false. |
| `SummarizeIMIdentities`, `SummarizeAdmin`, `HumanUptimeFromEtime`, `HeartbeatLabel` | `tui/internal/inventory/inventory.go:459-606` | Rendering helpers shared by CLI and TUI callers without importing either package. |

## Connections

- **Called from:** `lingtai-tui list` after platform process scanning (`tui/list_unix.go:19-27`, `tui/list_windows.go:19-24`), `/projects` via `ProjectsModel.loadDataMsg` (`tui/internal/tui/projects.go:247-318`), and purge filtering through `AgentDirInFilter`.
- **Home Agent rail:** accepted snapshots become immutable display/target rows only through the ordinary rail predicate (`tui/internal/tui/rail.go:210-269`). Activation and substantive target work re-run that home-rail contract (`tui/internal/tui/project_mail_store.go:202-233`) instead of borrowing global `Record.Enterable`.
- **Cross-project visit:** the temporary visit adapter deliberately retains global orchestrator/admin `Enterable` semantics, while also requiring verified address/PID/project/fingerprint identity (`tui/internal/tui/project_mail_store.go:179-200`).
- **Calls out:** `internal/processscan` for visible process rows when using `Scan`, and `internal/fs` for `.agent.json`, `.agent.heartbeat`, `.status.json`, `.agent.lock`, and MCP identity metadata.
- **Does not call:** package `main`, `internal/tui`, or any root Bubble Tea model. Keep this package below both CLI and TUI callers to avoid import cycles.

## Composition

- **Parent:** `tui/` (`tui/ANATOMY.md`)
- **Siblings:** `tui/internal/processscan/` (process table), `tui/internal/fs/` (agent filesystem), `tui/internal/tui/` (interactive consumers).
- **Consumers:** `tui/list_common.go` owns CLI rendering; `tui/internal/tui/projects.go` owns the grouped switcher; `tui/internal/tui/rail.go` owns the same-project Agent rail.

## State

- **Reads:** process table via `processscan`; per-agent `.agent.json`, `.agent.heartbeat`, `.status.json`, `.agent.lock`, and `system/mcp_identities/*.json`.
- **Writes:** none.
- **Errors/provenance:** scan failures are returned so empty and failed inventories differ. Per-record read failures stay visible on `Record.ReadError`. A successfully read but empty manifest address is not a read error; it is explicitly displayable but not an actionable rail identity.

## Invariants

1. Process-table visibility is the inclusion boundary. Human pseudo-agents are excluded by default, but stale-heartbeat agents remain visible.
2. Phantom projects and unreadable manifests remain visible with explicit non-enterable errors.
3. Display fallback and actionable identity are different facts: basename fallback never sets `ManifestAddressVerified`.
4. Global project-visit `Enterable` and ordinary same-project rail admission are separate contracts; neither silently broadens the other.
5. Duplicate rows collapse deterministically, preferring the runtime PID; sorting/grouping is stable for CLI and TUI tests.
6. Role detection is owned by `fs.IsOrchestratorManifest`, not duplicated in CLI or TUI packages.
