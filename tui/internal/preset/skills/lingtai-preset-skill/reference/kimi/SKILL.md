---
name: preset-skill-kimi
description: >
  Nested lingtai-preset-skill reference for the `kimi` built-in preset.
  Read when verifying shipped identity, provider sources, and compatibility
  facts for the `kimi` template preset.
version: 1.0.0
last_changed_at: "2026-07-10T10:00:00Z"
maintenance: "If you find stale or incorrect information here, use the lingtai-issue-report skill to assemble evidence and obtain per-issue human consent before filing an issue. Never include secrets, credentials, tokens, or private paths."
---

# `kimi` — built-in preset manual

> Progressive-disclosure manual for the `kimi` built-in preset.

## When to read

- Editing or health-checking the `kimi` built-in template preset.
- Confirming the shipped model string, base URL, or API key env.
- Needing authoritative Kimi/Moonshot model/pricing/API/auth links.

## Shipped identity (verify in TUI source)

Built through `openAICompatTextPreset` (`preset.go:912-932`).

| Field | Value | Source |
|---|---|---|
| Preset name | `kimi` | `preset.go:1053` |
| `llm.provider` | `kimi` | `preset.go:922` |
| `llm.model` | `kimi-for-coding` | `preset.go:1055` |
| `llm.api_compat` | `openai` | `preset.go:924` |
| `llm.api_key_env` | `KIMI_CODE_API_KEY` | `preset.go:1055` |
| `llm.base_url` | `https://api.kimi.com/coding/v1` | `preset.go:1055` |
| Tier | `3` | `preset.go:1055` |
| Capabilities | `web_search` (DuckDuckGo), `skills` default | `preset.go:926-929` |
| Text only | No vision/media capability | `preset.go:1046-1051` |

**Kernel note:** the kernel sets `User-Agent: LingTai-Agent/1.0` for every Kimi
request to comply with Kimi's Terms of Service (`lingtai-kernel/llm/service.py:272-281`).

## Official provider sources

| Resource | URL |
|---|---|
| Platform intro | <https://platform.kimi.com/docs/intro> |
| API overview | <https://platform.kimi.com/docs/api> |
| Chat completions reference | <https://platform.kimi.com/docs/api/chat> |
| Pricing / billing | <https://platform.kimi.com/docs/pricing> |
| Legacy Moonshot platform | <https://platform.moonshot.cn/docs/intro> |

**Base URL note:** shipped `base_url` is `https://api.kimi.com/coding/v1`.
Official docs may illustrate `https://api.moonshot.cn/v1` or another endpoint.

## Maintenance checklist

- **Provider-owned (refresh):** current coding model names, pricing, rate limits.
- **LingTai-owned (verify in source):** manifest shape, `api_compat`, `base_url`,
  `KIMI_CODE_API_KEY`, mandatory `User-Agent` header, text-only design choice.