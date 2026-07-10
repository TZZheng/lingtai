---
name: preset-skill-zhipu
description: >
  Nested lingtai-preset-skill reference for the `zhipu` built-in preset.
  Read when verifying shipped identity, provider sources, and compatibility
  facts for the `zhipu` template preset.
version: 1.0.0
last_changed_at: "2026-07-10T10:00:00Z"
maintenance: "If you find stale or incorrect information here, use the lingtai-issue-report skill to assemble evidence and obtain per-issue human consent before filing an issue. Never include secrets, credentials, tokens, or private paths."
---

# `zhipu` — built-in preset manual

> Thin pointer. Code is truth for the wiring; provider docs are truth for
> volatile catalogs (models, prices, quotas).

## When to read

- Editing or health-checking the `zhipu` built-in template preset.
- Confirming region-bound endpoint behavior or env-var slot naming.
- Needing authoritative Zhipu/GLM model/pricing/API/auth links.

## Shipped identity (verify in TUI source)

From `zhipuPreset()` (`tui/internal/preset/preset.go:961`).

| Field | Value | Source |
|---|---|---|
| Preset name | `zhipu` | `preset.go:961` |
| `llm.provider` | `zhipu` | `preset.go:961` |
| `llm.model` | `GLM-5.2` | `preset.go:961` |
| `llm.api_compat` | `openai` | OpenAI Chat Completions protocol |
| `llm.api_key_env` | `ZHIPU_API_KEY` | `preset.go:961` |
| `llm.base_url` | `ProviderRegionURLs["zhipu"][0].URL` (CN default) | `preset.go:478` |
| Capabilities | `web_search` (provider `zhipu`), `vision` (provider `zhipu`), `skills` default | `preset.go:961` |

## Regional endpoints

`ProviderRegionURLs["zhipu"]` (`preset.go:478`). Keys are region-bound — not
interchangeable.

| Region | Host | base_url |
|---|---|---|
| CN | `open.bigmodel.cn` | `https://open.bigmodel.cn/api/coding/paas/v4` |
| INTL | `api.z.ai` | `https://api.z.ai/api/coding/paas/v4` |

Region detection (`regionSuffix`, `preset.go:1465`): `base_url` containing
`api.z.ai` -> INTL, otherwise -> CN. Env-var slots: `ZHIPU_CN_<N>_API_KEY` /
`ZHIPU_INTL_<N>_API_KEY` via `AutoEnvVarName` (`preset.go:1421`).

## Official provider sources

| Topic | International (Z.AI) | Mainland (BigModel) |
|---|---|---|
| Overview / models | <https://docs.z.ai/devpack/overview> | <https://docs.bigmodel.cn/cn/coding-plan/overview> |
| Quick start / API key | <https://docs.z.ai/devpack/quick-start> | <https://docs.bigmodel.cn/cn/coding-plan/quick-start> |
| Pricing | <https://z.ai/subscribe> | <https://docs.bigmodel.cn/cn/coding-plan/faq.md> |
| API reference | <https://docs.z.ai/api-reference> | <https://docs.bigmodel.cn/api-reference> |

This preset uses the OpenAI path (`api_compat: openai`) at `/api/coding/paas/v4`.

## Related capabilities

The same Zhipu coding-plan key unlocks MCP servers (vision, web search, web
reader, zread). That is a separate concern — see `swiss-knife/reference/zhipu-coding-plan/SKILL.md`.

## Maintenance checklist

- **Provider-owned (refresh):** model names, pricing, plan tiers, quotas, doc paths.
- **LingTai-owned (verify in source):** preset wiring, two regional `base_url`s,
  `api_compat`, `ZHIPU_*_<N>_API_KEY` env-var rules.