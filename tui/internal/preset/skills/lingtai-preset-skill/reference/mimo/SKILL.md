---
name: preset-skill-mimo
description: >
  Nested lingtai-preset-skill reference for the `mimo` built-in preset.
  Read when verifying shipped identity, provider sources, and compatibility
  facts for the `mimo` template preset.
version: 1.0.0
last_changed_at: "2026-07-10T10:00:00Z"
maintenance: "If you find stale or incorrect information here, use the lingtai-issue-report skill to assemble evidence and obtain per-issue human consent before filing an issue. Never include secrets, credentials, tokens, or private paths."
---

# `mimo` — built-in preset manual

> Built-in name: `mimo` — index 3 in `BuiltinPresets()` (`preset.go:493`).

## When to read

- Choosing or configuring Xiaomi MiMo as the default LLM.
- Switching among TUI-exposed MiMo model variants.
- Debugging API key, endpoint, or vision-capability mismatches.

## Shipped identity (verify in TUI source)

| Field | Value | Source |
|---|---|---|
| `llm.provider` | `mimo` | `preset.go:989` |
| `llm.model` | `mimo-v2.5` | `preset.go:990` |
| `llm.api_key_env` | `XIAOMI_API_KEY` | `preset.go:998` |
| `llm.base_url` | `https://api.xiaomimimo.com/v1` | `preset.go:999` |
| `llm.api_compat` | `openai` | `preset.go:999` |
| `web_search` | `duckduckgo` | `preset.go:1002` |
| `vision` | provider `mimo`, model `mimo-v2.5` | `preset.go:1003` |

## TUI model picker

The preset editor exposes three MiMo models (`preset_editor.go:132`):

| Model | Vision |
|---|---|
| `mimo-v2.5` | yes (default) |
| `mimo-v2.5-pro` | no (text-only; 400 on image input) |
| `mimo-v2-flash` | no (text-only; 400 on image input) |

Vision is gated per-model (`preset_editor.go:198-201`). The kernel dispatches
MiMo vision through `MiMoVisionService` (kernel `services/vision/mimo.py`).

## Auth / setup

1. Obtain an API key from the Xiaomi MiMo platform.
2. Store in `~/.lingtai-tui/.env` under `XIAOMI_API_KEY`.
3. Default `base_url` is pay-as-you-go. For Token Plan (`tp-` prefixed keys),
   switch to the regional cluster assigned to your subscription.

**Key format:** `sk-*` keys -> `api.xiaomimimo.com`; `tp-*` keys -> Token Plan
cluster. Wrong combination returns `401 invalid_key`.

## Official provider sources

| Purpose | URL |
|---|---|
| Developer docs (LLM-friendly) | <https://platform.xiaomimimo.com/llms.txt> |
| OpenAI-compat API reference | <https://platform.xiaomimimo.com/docs/zh-CN/api/chat/openai-api> |
| API reference landing | <https://platform.xiaomimimo.com/api-reference> |

## Maintenance checklist

- **Provider-owned (refresh):** model IDs, pricing, API schemas, endpoint URLs.
- **LingTai-owned (verify in source):** preset fields, capability wiring,
  env-var names, vision service dispatch.
- **Operational caveat:** changing `llm.model` to a text-only variant without
  removing `capabilities.vision` causes 400 on vision tool calls.