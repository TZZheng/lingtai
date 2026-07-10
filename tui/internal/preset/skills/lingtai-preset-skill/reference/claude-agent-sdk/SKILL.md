---
name: preset-skill-claude-agent-sdk
description: >
  Nested lingtai-preset-skill reference for the `claude-agent-sdk` built-in preset.
  Read when verifying shipped identity, provider sources, and compatibility
  facts for the `claude-agent-sdk` template preset.
version: 1.0.0
last_changed_at: "2026-07-10T10:00:00Z"
maintenance: "If you find stale or incorrect information here, use the lingtai-issue-report skill to assemble evidence and obtain per-issue human consent before filing an issue. Never include secrets, credentials, tokens, or private paths."
---

# `claude-agent-sdk` — built-in preset manual

> Completion-only provider that rides on Anthropic's local Claude Code CLI.
> Not OpenAI-compatible; does not accept a per-request API key.

## When to read

- Verifying the `claude-agent-sdk` preset's shipped identity.
- Understanding the CLI-login dependency and absence of API-key auth.
- Distinguishing LingTai integration from Anthropic's SDK/CLI.

## Shipped identity (verify in TUI source)

| Field | Value | Source |
|---|---|---|
| Preset name | `claude-agent-sdk` | `preset.go:1164` |
| `llm.provider` | `claude-agent-sdk` | `preset.go:1178` |
| `llm.model` | `opus` (CLI alias) | `preset.go:1178` |
| `llm.api_key_env` | `""` (none) | `preset.go:1179` |
| `llm.base_url` | none (CLI-owned) | `preset.go:1178-1179` |
| Auth mode | Local `claude` CLI login | `preset.go:1169-1174`; `claude_auth.go:17-27` |
| Capabilities | `skills` default only | `preset.go:1187-1189` |
| Absent | `web_search`, `vision` | `preset.go:1181-1186` |
| Alias | `claude_agent_sdk` (compatibility) | `preset.go:524` |

## Auth detection

The TUI detects CLI login via `claude auth status --json` (timeout 4s,
`claude_auth.go:15`). Success signal: JSON with `loggedIn: true`.

Uncertain outcomes (CLI missing, nonzero exit, timeout) all resolve to
"not configured" (`claude_auth.go:24-25`).

The credential guard (`ResolveRefsWithAuth`) treats the preset as "HasKey"
only when `AuthState.ClaudeCodeAuthConfigured` is true (`preset.go:776-780`).

## Official Anthropic sources

| Resource | URL |
|---|---|
| Claude Code overview | <https://docs.anthropic.com/en/docs/agents-and-tools/claude-code/overview> |
| Agent SDK overview | <https://code.claude.com/docs/en/agent-sdk/overview> |
| Agent SDK Python reference | <https://code.claude.com/docs/en/agent-sdk/python> |
| CLI reference | <https://code.claude.com/docs/en/cli-reference> |
| Model configuration | <https://code.claude.com/docs/en/model-config> |

The `opus` alias resolves to the latest Opus model available to the active
account. To pin a version, use a full model ID or `ANTHROPIC_DEFAULT_OPUS_MODEL`.

## Maintenance checklist

- **Provider-owned (refresh):** CLI installation, SDK versions, model aliases,
  pricing, auth methods.
- **LingTai-owned (verify in source):** manifest shape, CLI auth probe,
  credential guard mapping, deliberate absence of `web_search`/`vision`.