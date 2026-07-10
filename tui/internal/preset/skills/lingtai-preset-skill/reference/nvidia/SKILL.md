---
name: preset-skill-nvidia
description: >
  Nested lingtai-preset-skill reference for the `nvidia` built-in preset.
  Read when verifying shipped identity, provider sources, and compatibility
  facts for the `nvidia` template preset.
version: 1.0.0
last_changed_at: "2026-07-10T10:00:00Z"
maintenance: "If you find stale or incorrect information here, use the lingtai-issue-report skill to assemble evidence and obtain per-issue human consent before filing an issue. Never include secrets, credentials, tokens, or private paths."
---

# `nvidia` — NVIDIA NIM preset manual

> `nvidia` — a shipped built-in template in `BuiltinPresets()` (`preset.go:489`).

## When to read

- Verifying the `nvidia` preset's stable identity or current behavior.
- Confirming the correct NVIDIA API Catalog URLs.
- Understanding the `prompt_cache_key` disable.

## Shipped identity (verify in TUI source)

Built through `openAICompatTextPreset` (`preset.go:912`).

| Field | Value | Source |
|---|---|---|
| `name` | `nvidia` | `preset.go:1074` |
| `llm.provider` | `nvidia` | `preset.go:922` |
| `llm.model` | `meta/llama-3.3-70b-instruct` | `preset.go:1076` |
| `llm.api_key_env` | `NVIDIA_API_KEY` | `preset.go:1076` |
| `llm.base_url` | `https://integrate.api.nvidia.com/v1` | `preset.go:1076` |
| `llm.api_compat` | `openai` | `preset.go:924` |
| Capabilities | `web_search` (DuckDuckGo), `skills` default | `preset.go:927-928` |

**Kernel note (`preset.go:1070-1072`):** the kernel registers `"nvidia"` with
`prompt_cache_key` disabled — NVIDIA NIM rejects that field with HTTP 400.

## Model switching

Default model is `meta/llama-3.3-70b-instruct`. Users clone this preset to
switch to any NVIDIA API Catalog model ID. The full catalog is at the provider
listing below; LingTai does not duplicate it.

## Official provider sources

| Resource | URL |
|---|---|
| NVIDIA API Catalog | <https://build.nvidia.com> |
| Available models | <https://build.nvidia.com/models> |
| API reference (LLM APIs) | <https://docs.api.nvidia.com/nim/reference/llm-apis> |
| API key generation | <https://build.nvidia.com> -> API key |

## Maintenance checklist

- **Provider-owned (refresh):** model catalog, pricing, free-tier quotas, API URLs.
- **LingTai-owned (verify in source):** provider string `"nvidia"`, `api_compat`
  routing, `prompt_cache_key` disable, `NVIDIA_API_KEY` env name, text-only
  restriction, DuckDuckGo default.