---
name: preset-skill-custom
description: >
  Nested lingtai-preset-skill reference for the `custom` built-in preset.
  Read when verifying shipped identity, provider sources, and compatibility
  facts for the `custom` template preset.
version: 1.0.0
last_changed_at: "2026-07-10T10:00:00Z"
maintenance: "If you find stale or incorrect information here, use the lingtai-issue-report skill to assemble evidence and obtain per-issue human consent before filing an issue. Never include secrets, credentials, tokens, or private paths."
---

# `custom` — built-in preset manual

> The generic escape hatch for any endpoint that speaks one of the wire
> protocols LingTai's custom adapter implements.

## When to read

- Verifying the `custom` preset's manifest shape or supported protocols.
- Understanding capability inheritance (vision `inherit`).
- Routing through OpenAI-compatible, Anthropic, or Gemini endpoints.

## Shipped identity (verify in TUI source)

| Field | Value | Source |
|---|---|---|
| Preset name | `custom` | `preset.go:1196` |
| `llm.provider` | `custom` | `preset.go:1200` |
| `llm.model` | `""` (user must set) | `preset.go:1200` |
| `llm.api_key_env` | `LLM_API_KEY` | `preset.go:1201` |
| `llm.base_url` | `nil` (user must set) | `preset.go:1201` |
| `llm.api_compat` | unset; defaults to `openai` | `custom/defaults.py:2`; `_register.py:60` |
| TUI editor compat options | `""`, `"openai"`, `"anthropic"` | `preset_editor.go:1114-1116` |
| Capabilities | `web_search` (empty -> default), `vision` (`inherit`), `skills` default | `preset.go:1203-1212` |

## Kernel custom adapter

`src/lingtai/llm/custom/adapter.py:20-51` implements `create_custom_adapter()`:

| api_compat | Backend | Requirements |
|---|---|---|
| `"openai"` (default) | `OpenAIAdapter` | `base_url` required |
| `"anthropic"` | `AnthropicAdapter` | `base_url` required |
| `"gemini"` | `GeminiAdapter` | `api_key` required; `base_url` ignored |

**Note:** the TUI preset editor exposes only `openai` and `anthropic` for
the `custom` preset. Gemini is supported by the kernel adapter but not
exposed in the TUI editor for `custom`.

## Capability inheritance

- `vision` is configured with `provider: "inherit"` (`preset.go:1211`).
  The kernel's `expand_inherit` (`presets.py:570-594`) copies the main LLM's
  provider, api_key, api_key_env, base_url, and api_compat into the capability
  config. Vision then falls back to `OpenAIVisionService` or
  `AnthropicVisionService` based on the inherited `api_compat`.
- `web_search` is shipped as an empty map (`preset.go:1204`). The kernel
  defaults to `duckduckgo` when no provider is supplied.

## Official protocol sources

| Protocol | Documentation |
|---|---|
| OpenAI Chat Completions | <https://platform.openai.com/docs/api-reference/chat> |
| Anthropic Messages | <https://docs.anthropic.com/en/api/messages> |
| Google Gemini | <https://ai.google.dev/gemini-api/docs> |

Do not embed model tables, pricing, or marketing claims — those live with
the endpoint operator.

## Maintenance checklist

- **Provider-owned (refresh):** model IDs, pricing, auth flows of the
  configured endpoint.
- **LingTai-owned (verify in source):** manifest shape, TUI editor options,
  kernel adapter factory, adapter registration, `api_compat` pass-through,
  capability inheritance wiring, vision fallback routing.