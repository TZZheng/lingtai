---
name: preset-skill-minimax
description: >
  Nested lingtai-preset-skill reference for the `minimax` built-in preset.
  Read when verifying shipped identity, provider sources, and compatibility
  facts for the `minimax` template preset.
version: 1.0.0
last_changed_at: "2026-07-10T10:00:00Z"
maintenance: "If you find stale or incorrect information here, use the lingtai-issue-report skill to assemble evidence and obtain per-issue human consent before filing an issue. Never include secrets, credentials, tokens, or private paths."
---

# `minimax` — built-in preset manual

> **Manual, not a tool.** The TUI source is authority for shipped preset
> identity; official MiniMax docs are authority for models, pricing, endpoints,
> and auth. Do not copy volatile values into this file — link them.

## When to read

- Editing, debugging, or health-checking the `minimax` built-in template preset.
- Rotating a MiniMax API key, adding a second account, or resolving a
  CN-vs-INTL region mismatch.
- Needing the authoritative MiniMax model/pricing/API/auth link.

## Shipped identity (verify in TUI source)

Produced by `minimaxPreset()` (`tui/internal/preset/preset.go`).

| Field | Value | Source |
|---|---|---|
| Preset name | `minimax` | `preset.go:940` |
| `llm.provider` | `minimax` | `preset.go:936,944` |
| `llm.model` | `MiniMax-M3` | `preset.go:944` |
| `llm.api_key_env` | `MINIMAX_API_KEY` | `preset.go:937,945` |
| `llm.base_url` | `ProviderRegionURLs["minimax"][0].URL` (CN default) | `preset.go:946` |
| `llm.api_compat` | not set (Anthropic-compatible path) | `preset.go:943-946` |
| Capabilities | `web_search`, `vision` (provider `minimax`); `skills` default | `preset.go:952-956` |

**Notes:**
- Default preset: `DefaultPreset()` returns `minimaxPreset()` (`preset.go:1400-1402`).
- API shape: no `api_compat` override; base URL ends in `/anthropic` — MiniMax
  is reached through its Anthropic-compatible Messages API.

## Region endpoints

`ProviderRegionURLs` (`preset.go:467-486`); first entry (CN) is default.

| Label | base_url |
|---|---|
| CN | `https://api.minimaxi.com/anthropic` |
| INTL | `https://api.minimax.io/anthropic` |

Region detection: `base_url` containing `minimaxi.com` -> CN; otherwise -> INTL
(`regionSuffix`, `preset.go:1465-1479`).

**API key slot naming:** default `MINIMAX_API_KEY`; additional keys use
`MINIMAX_<REGION>_<N>_API_KEY` via `AutoEnvVarName` (`preset.go:1405-1459`).

## Official provider sources

These are provider-owned and refreshable. International docs at
`platform.minimax.io`, mainland-China at `platform.minimaxi.com`.

| Need | Source |
|---|---|
| Anthropic Messages API (intl) | <https://platform.minimax.io/docs/api-reference/text-chat-anthropic> |
| Anthropic Messages API (CN) | <https://platform.minimaxi.com/docs/api-reference/text-chat-anthropic> |
| Models list | <https://platform.minimax.io/docs/api-reference/models/anthropic/list-models> |
| Auth and setup | <https://platform.minimax.io/docs/guides/quickstart-preparation> |
| Pricing | <https://platform.minimax.io/docs/pricing/overview> |
| Claude Code base-URL setup | <https://platform.minimax.io/docs/token-plan/claude-code> |

Auth: **Bearer API key** — `Authorization: Bearer <API_KEY>`.

## Maintenance checklist

- **Provider-owned (refresh):** model lineup, pricing, endpoints, region hosts,
  request/response schema. Re-verify against official links before relying.
- **LingTai-owned (verify in source):** preset name, summary, default model
  string, `MINIMAX_API_KEY` slot, `ProviderRegionURLs` entries, region detection,
  capabilities wiring, "default preset" status.