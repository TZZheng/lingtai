---
name: preset-skill-openrouter
description: >
  Nested lingtai-preset-skill reference for the `openrouter` built-in preset.
  Read when verifying shipped identity, provider sources, and compatibility
  facts for the `openrouter` template preset.
version: 1.0.0
last_changed_at: "2026-07-10T10:00:00Z"
maintenance: "If you find stale or incorrect information here, use the lingtai-issue-report skill to assemble evidence and obtain per-issue human consent before filing an issue. Never include secrets, credentials, tokens, or private paths."
---

# `openrouter` — built-in preset manual

> Progressive-disclosure manual for the `openrouter` built-in preset.

## When to read

- Editing or health-checking the `openrouter` built-in template preset.
- Confirming the shipped model slug, base URL resolution, or API key env.
- Needing authoritative OpenRouter model/pricing/API/auth links.

## Shipped identity (verify in TUI source)

| Field | Value | Source |
|---|---|---|
| Preset name | `openrouter` | `preset.go:1081` |
| `llm.provider` | `openrouter` | `preset.go:1085` |
| `llm.model` | `z-ai/glm-5.1` | `preset.go:1085` |
| `llm.api_key_env` | `OPENROUTER_API_KEY` | `preset.go:1086` |
| `llm.base_url` | `null` (TUI-resolved to `https://openrouter.ai/api/v1`) | `preset.go:1087`; `init.jsonc:35-36` |
| Capabilities | `web_search` (DuckDuckGo), `skills` default | `preset.go:1092-1095` |
| Text only | No vision/media | `preset.go:1089-1091` |

OpenRouter does not use the generic `api_compat` field. The manifest sets
`provider: "openrouter"` and leaves `base_url` as `null`; the TUI/kernel
resolve this to `https://openrouter.ai/api/v1` (`init.jsonc:35-36`).

## Official provider sources

| Resource | URL |
|---|---|
| Models and pricing | <https://openrouter.ai/models> |
| Models API | `GET https://openrouter.ai/api/v1/models` |
| API reference | <https://openrouter.ai/docs/api> |
| Quickstart | <https://openrouter.ai/docs/quickstart> |
| API keys | <https://openrouter.ai/keys> |

## Provider compatibility contract

The OpenRouter preset is first-class in LingTai: it does not use the generic
`api_compat` field. The manifest sets `provider: "openrouter"` and leaves
`base_url` as `null`; the TUI/kernel resolve this to the OpenRouter endpoint
`https://openrouter.ai/api/v1` (`init.jsonc:35-36`).

OpenRouter's own API is OpenAI-compatible:
- Primary endpoint: `POST https://openrouter.ai/api/v1/chat/completions`.
- Authentication: `Authorization: Bearer <OPENROUTER_API_KEY>`.
- Model values are OpenRouter slugs such as `provider/model`; the shipped
  default `z-ai/glm-5.1` is one such slug.
- Optional attribution headers (`HTTP-Referer`, `X-OpenRouter-Title`) are
  documented in the API reference; the preset does not set them by default.

## Use when / avoid when

- **Use** when you want access to many providers through a single API key and
  unified billing.
- **Avoid** when you need vision/media capabilities — this preset is text-only.
  For media, use a provider-specific preset or register MCP servers.

## Verification index

1. **Model string:** `preset.go:1085` — matches a current slug on <https://openrouter.ai/models>.
2. **Base URL:** `preset.go:1087` (null) + `init.jsonc:35-36` — resolves to `https://openrouter.ai/api/v1`.
3. **Builtin list:** `preset.go:489-504` — `openrouter` is present at position 7.
4. **Test assertion:** `preset_test.go:227` — the 12 built-in preset names include `openrouter`.
5. **Anatomy:** `ANATOMY.md:50` — `BuiltinPresets()` row enumerates openrouter.

## Maintenance checklist

- **Provider-owned (refresh):** current models, slugs, pricing, rate limits.
- **LingTai-owned (verify in source):** manifest shape, TUI-fixed base URL
  behavior, `api_compat` absence, text-only design choice.