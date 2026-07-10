---
name: preset-skill-deepseek
description: >
  Nested lingtai-preset-skill reference for the `deepseek` built-in preset.
  Read when verifying shipped identity, provider sources, and compatibility
  facts for the `deepseek` template preset.
version: 1.0.0
last_changed_at: "2026-07-10T10:00:00Z"
maintenance: "If you find stale or incorrect information here, use the lingtai-issue-report skill to assemble evidence and obtain per-issue human consent before filing an issue. Never include secrets, credentials, tokens, or private paths."
---

# `deepseek` — built-in preset manual

> Progressive-disclosure manual. Its job is to tell a future maintainer where
> to verify current truth, not to duplicate volatile catalogs.

## When to read

- Editing or health-checking the `deepseek` built-in template preset.
- Confirming the shipped model string or base URL.
- Needing authoritative DeepSeek model/pricing/API/auth links.

## Shipped identity (verify in TUI source)

Built through the `openAICompatTextPreset` helper (`preset.go:904-932`).

| Field | Value | Source |
|---|---|---|
| Preset name | `deepseek` | `preset.go:1015` |
| `llm.provider` | `deepseek` | `preset.go:921` |
| `llm.model` | `deepseek-v4-pro` | `preset.go:1018` |
| `llm.api_compat` | `openai` | `preset.go:924` |
| `llm.api_key_env` | `DEEPSEEK_API_KEY` | `preset.go:1018` |
| `llm.base_url` | `https://api.deepseek.com` | `preset.go:1018` |
| Capabilities | `web_search` (DuckDuckGo), `skills` default | `preset.go:926-929` |
| Text only | No vision/media capability | `preset.go:1011-1014` |

Single global endpoint — no regional URLs (not in `ProviderRegionURLs`).

## Official provider sources

| Resource | URL |
|---|---|
| API documentation | <https://api-docs.deepseek.com/> |
| Platform / API keys | <https://platform.deepseek.com/api_keys> |
| Tool calls guide | <https://api-docs.deepseek.com/guides/tool_calls/> |
| Thinking mode | <https://api-docs.deepseek.com/guides/thinking_mode/> |
| Coding agent integrations | <https://api-docs.deepseek.com/guides/coding_agents/> |

## Use when / avoid when

- **Use** when you want DeepSeek V4 Pro as the default LLM — verify current context
  window and capabilities via the DeepSeek API docs; tool calls, thinking mode,
  OpenAI-compatible API.
- **Avoid** when you need vision/media capabilities — this preset is text-only.
  For media creation, register a provider MCP server.

## Verification index

When maintaining this preset, verify in this order:

1. **Model string:** `preset.go:1018` — matches current DeepSeek docs.
2. **Base URL:** `preset.go:1018` — matches `https://api.deepseek.com`.
3. **Builtin list:** `preset.go:489-504` — `deepseek` is present at position 4.
4. **Test assertion:** `preset_test.go:492` — `deepseekPreset()` model is `deepseek-v4-pro`.
5. **Anatomy:** `ANATOMY.md:50` — `BuiltinPresets()` row enumerates deepseek.

## Maintenance checklist

- **Provider-owned (refresh):** current models, pricing, rate limits, auth flows.
- **LingTai-owned (verify in source):** manifest shape, `api_compat`, `base_url`,
  `DEEPSEEK_API_KEY`, text-only design choice.