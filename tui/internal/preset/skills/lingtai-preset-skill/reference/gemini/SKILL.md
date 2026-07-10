---
name: preset-skill-gemini
description: >
  Nested lingtai-preset-skill reference for the `gemini` built-in preset.
  Read when verifying shipped identity, provider sources, and compatibility
  facts for the `gemini` template preset.
version: 1.0.0
last_changed_at: "2026-07-10T10:00:00Z"
maintenance: "If you find stale or incorrect information here, use the lingtai-issue-report skill to assemble evidence and obtain per-issue human consent before filing an issue. Never include secrets, credentials, tokens, or private paths."
---

# `gemini` — built-in preset manual

Google Gemini — native adapter, multimodal, tool calling, vision.

## When to read

- Someone asks about the `gemini` built-in preset, its model, auth, or protocol.
- Verifying Gemini model/pricing information.

## Shipped identity (verify in TUI source)

| Field | Value | Source |
|---|---|---|
| Preset name | `gemini` | `preset.go:1026` |
| `llm.provider` | `gemini` | `preset.go:1030` |
| `llm.model` | `gemini-3-flash-preview` | `preset.go:1030` |
| `llm.api_key_env` | `GEMINI_API_KEY` | `preset.go:1031` |
| Protocol | **Native Gemini** — NOT OpenAI-compat | `preset.go:1023-1024` |
| Tier | `3` | `preset.go:1027` |
| Capabilities | `vision` (native), `web_search` (DuckDuckGo), `skills` default | `preset.go:1033-1038` |

**No OAuth, no base_url, no api_compat.** The kernel's `GeminiAdapter` wraps
`google.genai.Client` directly (`llm/gemini/adapter.py:702`).

## Kernel adapter architecture

Two session backends:
1. **Interactions API** (primary, `adapter.py:739`) — server-side conversation state.
2. **Chat API** (fallback for `json_schema` mode, `adapter.py:762-798`).

The adapter auto-selects thinking config for Gemini 3+ models
(`_supports_thinking()`, `adapter.py:115-126`).

## Official provider sources

| Topic | URL |
|---|---|
| Models and capabilities | <https://ai.google.dev/gemini-api/docs/models> |
| Pricing and billing | <https://ai.google.dev/gemini-api/docs/pricing> |
| API key setup | <https://ai.google.dev/gemini-api/docs/api-key> |
| API reference | <https://ai.google.dev/api> |
| Python SDK | <https://pypi.org/project/google-genai/> |

Do not embed current model lists, pricing, or quotas — they change frequently.

## Maintenance checklist

- **Provider-owned (refresh):** models, pricing, rate limits, available model IDs.
- **LingTai-owned (verify in source):** provider string, adapter class, env var
  name, capability flags, kernel registration.