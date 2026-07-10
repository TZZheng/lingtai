---
name: lingtai-preset-skill
description: >
  Routing table for the 12 TUI-shipped built-in preset templates. Reach for
  this skill when you need to select, configure, compare, or troubleshoot a
  built-in preset template — not an arbitrary saved preset. Routes to one
  nested manual per built-in preset under reference/<preset>/SKILL.md.
version: 1.0.0
last_changed_at: "2026-07-10T10:00:00Z"
related_files:
  - tui/internal/preset/skills/lingtai-preset-skill/SKILL.md
  - tui/internal/preset/preset.go
  - tui/internal/preset/ANATOMY.md
  - tui/internal/preset/skill_metadata_test.go
  - tui/internal/preset/preset_skill_router_test.go
maintenance: "If you find stale or incorrect information here, use the lingtai-issue-report skill to assemble evidence and obtain per-issue human consent before filing an issue. Never include secrets, credentials, tokens, or private paths."
---

# lingtai-preset-skill — built-in preset routing table

This skill is the **sole agent-facing routing table** for the 12 TUI-shipped
built-in preset templates. It routes you to one nested manual per preset.

**Boundary:** this router covers **template** presets (the `BuiltinPresets()`
list at `tui/internal/preset/preset.go:489`). A user-saved preset clone
(living under `~/.lingtai-tui/presets/saved/`) may diverge from its template —
inspect the saved preset's actual `manifest` JSON for current truth.

## Nested reference catalog

`lingtai-preset-skill` owns these nested references. They are parent-owned
drill-down files, not standalone top-level skills.

```yaml
- name: preset-skill-minimax
  location: reference/minimax/SKILL.md
  description: |
    Shipped built-in preset manual for `minimax` — verify preset identity, provider sources, and compatibility facts.
- name: preset-skill-zhipu
  location: reference/zhipu/SKILL.md
  description: |
    Shipped built-in preset manual for `zhipu` — verify preset identity, provider sources, and compatibility facts.
- name: preset-skill-mimo
  location: reference/mimo/SKILL.md
  description: |
    Shipped built-in preset manual for `mimo` — verify preset identity, provider sources, and compatibility facts.
- name: preset-skill-deepseek
  location: reference/deepseek/SKILL.md
  description: |
    Shipped built-in preset manual for `deepseek` — verify preset identity, provider sources, and compatibility facts.
- name: preset-skill-gemini
  location: reference/gemini/SKILL.md
  description: |
    Shipped built-in preset manual for `gemini` — verify preset identity, provider sources, and compatibility facts.
- name: preset-skill-kimi
  location: reference/kimi/SKILL.md
  description: |
    Shipped built-in preset manual for `kimi` — verify preset identity, provider sources, and compatibility facts.
- name: preset-skill-nvidia
  location: reference/nvidia/SKILL.md
  description: |
    Shipped built-in preset manual for `nvidia` — verify preset identity, provider sources, and compatibility facts.
- name: preset-skill-openrouter
  location: reference/openrouter/SKILL.md
  description: |
    Shipped built-in preset manual for `openrouter` — verify preset identity, provider sources, and compatibility facts.
- name: preset-skill-codex
  location: reference/codex/SKILL.md
  description: |
    Shipped built-in preset manual for `codex` — verify preset identity, provider sources, and compatibility facts.
- name: preset-skill-codex-pool
  location: reference/codex-pool/SKILL.md
  description: |
    Shipped built-in preset manual for `codex-pool` — verify preset identity, provider sources, and compatibility facts.
- name: preset-skill-claude-agent-sdk
  location: reference/claude-agent-sdk/SKILL.md
  description: |
    Shipped built-in preset manual for `claude-agent-sdk` — verify preset identity, provider sources, and compatibility facts.
- name: preset-skill-custom
  location: reference/custom/SKILL.md
  description: |
    Shipped built-in preset manual for `custom` — verify preset identity, provider sources, and compatibility facts.
```

## Routing table

| Preset | Manual | Trigger |
|--------|--------|---------|
| `minimax` | `reference/minimax/SKILL.md` | MiniMax-M3 — Anthropic-compatible, CN/INTL regions |
| `zhipu` | `reference/zhipu/SKILL.md` | Zhipu/Z.AI GLM Coding Plan — OpenAI-compatible |
| `mimo` | `reference/mimo/SKILL.md` | Xiaomi MiMo — OpenAI-compatible, multimodal |
| `deepseek` | `reference/deepseek/SKILL.md` | DeepSeek V4 Pro — OpenAI-compatible, text-only |
| `gemini` | `reference/gemini/SKILL.md` | Google Gemini — native adapter, multimodal |
| `kimi` | `reference/kimi/SKILL.md` | Kimi Code (Moonshot) — OpenAI-compatible, text-only |
| `nvidia` | `reference/nvidia/SKILL.md` | NVIDIA NIM — OpenAI-compatible, text-only |
| `openrouter` | `reference/openrouter/SKILL.md` | OpenRouter — unified gateway, text-only |
| `codex` | `reference/codex/SKILL.md` | Codex — ChatGPT OAuth, single account |
| `codex-pool` | `reference/codex-pool/SKILL.md` | Codex Pool — ChatGPT OAuth, multi-account load balancing |
| `claude-agent-sdk` | `reference/claude-agent-sdk/SKILL.md` | Claude Agent SDK — local CLI auth, no API key |
| `custom` | `reference/custom/SKILL.md` | Custom — user-supplied OpenAI/Anthropic-compatible endpoint |

## Maintenance rule

Provider-owned facts (models, pricing, endpoints, auth) change at the
provider's cadence. LingTai-owned facts (preset wiring, env var names,
capability flags, compatibility protocol) change only when the TUI source at
`tui/internal/preset/preset.go` changes. Each child manual documents which
facts belong to which owner.
