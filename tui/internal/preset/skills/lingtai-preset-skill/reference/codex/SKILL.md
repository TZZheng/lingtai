---
name: preset-skill-codex
description: >
  Nested lingtai-preset-skill reference for the `codex` built-in preset.
  Read when verifying shipped identity, provider sources, and compatibility
  facts for the `codex` template preset.
version: 1.0.0
last_changed_at: "2026-07-10T10:00:00Z"
maintenance: "If you find stale or incorrect information here, use the lingtai-issue-report skill to assemble evidence and obtain per-issue human consent before filing an issue. Never include secrets, credentials, tokens, or private paths."
---

# `codex` — built-in preset manual

Routes the agent through OpenAI's Codex backend using a ChatGPT account
OAuth session, not a standard OpenAI API key.

## When to read

- Choosing or troubleshooting the single-account Codex preset.
- Picking a model, binding an account, or distinguishing from `codex-pool`.

## Shipped defaults (verify in TUI source)

Source: `tui/internal/preset/preset.go:1100-1128`.

| Field | Value |
|---|---|
| `name` | `codex` |
| `llm.provider` | `codex` |
| `llm.model` | `gpt-5.6-sol` |
| `llm.base_url` | `https://chatgpt.com/backend-api/codex` |
| `llm.thinking` | `xhigh` |
| `llm.api_key_env` | `""` (OAuth only) |

Model picker: `preset_editor.go:147-152`.

## Use when / avoid when

- **Use** when you want a single ChatGPT account to power a Codex agent.
- **Avoid** when you need load-balancing across multiple accounts — use `codex-pool`.
- **Avoid** when you want a standard OpenAI API key flow — this uses OAuth tokens.

## Authentication

Codex uses ChatGPT OAuth token files. The TUI writes these during first-run
or Credentials setup.

- Default token file: `~/.lingtai-tui/codex-auth.json`
- Additional accounts: `~/.lingtai-tui/codex-auth/<slug>.json`
- Bind specific account via `manifest.llm.codex_auth_path`.
- Credential validity: judged by non-empty `refresh_token` in the bound file.

Official docs: <https://developers.openai.com/codex/auth>.
Official model catalog: <https://developers.openai.com/codex/models>.
Reference implementation: <https://github.com/openai/codex>.

## Operational caveats

- Base URL ends in `/codex`; omitting it produces HTML/Cloudflare errors.
- Token files are secret (mode `0600`); paths may appear in presets but
  contents must never be logged or copied into reports.
- Model availability depends on ChatGPT account tier and OpenAI rollout.

## Maintenance checklist

- **Provider-owned (refresh):** model names, availability, auth URLs from OpenAI.
- **LingTai-owned (verify in source):** provider string, endpoint,
  `codex_auth_path` behavior, OAuth token-file layout, capability wiring.