---
name: preset-skill-codex-pool
description: >
  Nested lingtai-preset-skill reference for the `codex-pool` built-in preset.
  Read when verifying shipped identity, provider sources, and compatibility
  facts for the `codex-pool` template preset.
version: 1.0.0
last_changed_at: "2026-07-10T10:00:00Z"
maintenance: "If you find stale or incorrect information here, use the lingtai-issue-report skill to assemble evidence and obtain per-issue human consent before filing an issue. Never include secrets, credentials, tokens, or private paths."
---

# `codex-pool` — built-in preset manual

Identical to `codex` in model, endpoint, and capabilities; only the provider
changes. `codex-pool` routes each agent session to one of several ChatGPT
OAuth token files selected by weighted, sticky load balancing from a
non-secret pool file.

## When to read

- Load-balancing a Codex agent across several ChatGPT accounts.
- Editing the non-secret pool file.
- Understanding v1 flat vs v2 exact-model classification.

## Shipped defaults (verify in TUI source)

Source: `tui/internal/preset/preset.go:1137-1160`.

| Field | Value |
|---|---|
| `name` | `codex-pool` |
| `llm.provider` | `codex-pool` |
| `llm.model` | `gpt-5.6-sol` |
| `llm.base_url` | `https://chatgpt.com/backend-api/codex` |
| `llm.thinking` | `xhigh` |
| `llm.api_key_env` | `""` (OAuth only) |

## Pool file schema

Two stable shapes, both handled by TUI and kernel:

**v1 — flat (`accounts`)**

```json
{"version": 1, "accounts": [{"path": "codex-auth.json", "weight": 1}]}
```

**v2 — exact-model classification (`models`)** (kernel #841 / TUI #612)

```json
{"version": 2, "models": {"gpt-5.6-sol": [{"path": "codex-auth.json", "weight": 1}]}}
```

### Exact-model classification rules (kernel #841 / TUI #612)

- The kernel keys off **structure**, not the `version` field.
- A top-level `models` dict makes the file classified; `models: null` does not.
- When classified, `models` is the **sole source of truth** — sibling flat
  `accounts` list is ignored.
- Matching is **exact, case-sensitive** — no prefix, family, wildcard, or default
  fallback within the classified file.
- The TUI credentials UI refuses flat weight edits on a classified pool so it
  never silently destroys a hand-written classification
  (`codex_pool_store.go:64-70, 292-306`).

Kernel source: merge `b05be6d4` in `lingtai-kernel`, `src/lingtai/auth/codex_pool.py`.

## Operational caveats

- Base URL ends in `/codex`; omitting it produces HTML/Cloudflare errors.
- Token files are secret (`0600`); pool files are non-secret (`0644`).
- Weight `0` disables an account without removing it.
- Selection is **sticky per agent session** and does not rotate on molt.
- A missing/empty/invalid pool file causes fallback to legacy `codex-auth.json`.

## Official sources

<https://developers.openai.com/codex/auth> | <https://developers.openai.com/codex/models>
| <https://github.com/openai/codex>

## Maintenance checklist

- **Provider-owned (refresh):** model names, availability, auth URLs from OpenAI.
- **LingTai-owned (verify in source):** provider string, endpoint, pool-file
  schema, exact-model classification semantics, sticky selection, OAuth layout,
  capability wiring. Reference both TUI and kernel source.