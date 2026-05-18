---
name: swiss-knife
description: >
  Umbrella skill for small, focused CLI tools and integrations. Sub-skills:
  (1) claude-code — multi-turn Claude Code CLI with persistent sessions via
  --session-id/--resume. Supports parallel sessions, model switching
  (haiku/sonnet/opus), budget control, and tool permission management.
  Use for delegating coding tasks, code review, iterative development;
  (2) minimax-cli — MiniMax CLI for text-to-image, text-to-video, music
  generation, TTS, and vision. Read when the human asks for any media
  creation or vision task;
  (3) openai-codex — OpenAI Codex CLI for local coding agent with remote
  control, Vim editing, plugins, hooks, and Chrome browser integration.
  Read when the human asks to use OpenAI Codex or compare with Claude Code;
  (4) token-usage — token usage tracking and cost reporting;
  (5) html-report — checklist + template for producing standalone HTML
  research memos, dashboards, and audit reports (with MathJax math
  rendering, anchor navigation, print styles). Read when the human asks
  for an HTML deliverable, especially one containing equations.
  Each sub-skill is independent — read only the one you need.
  Note: if a sub-skill listed below is missing from your on-disk
  utilities (e.g. you pulled a TUI update), ask the human to run
  `lingtai-tui bootstrap` in a shell — that re-extracts shipped skills
  to ~/.lingtai-tui/utilities/ without restarting the TUI — then call
  `system(action="refresh")` to pick them up.
version: 1.5.0
tags: [utilities, umbrella, toolkit]
---

# Swiss Knife — Utility Toolkit

A collection of small, useful skills. Each sub-skill lives in its own folder under `swiss-knife/` and is fully self-contained — scripts, assets, and a SKILL.md with complete instructions.

## Sub-Skills

| Sub-Skill | Description | When to Use |
|-----------|-------------|-------------|
| [token-usage](token-usage/) | Network-wide token cost calculator using litellm + OpenRouter pricing | Human asks about costs, budget, token usage, or spending |
| [claude-code](claude-code/) | Delegate code implementation, patch writing, docs, and refactoring to Claude Code CLI | Human asks to write code, generate patches, refactor, or delegate implementation work |
| [minimax-cli](minimax-cli/) | MiniMax CLI for text-to-image, text-to-video, music generation, TTS, and vision | Human asks for image/video/music generation, TTS narration, or vision tasks |
| [openai-codex](openai-codex/) | OpenAI Codex CLI — local coding agent with remote control, Vim editing, plugins, hooks, and Chrome extension | Human asks to use OpenAI Codex CLI, compare with Claude Code, or needs browser integration |
| [html-report](html-report/) | Checklist + standalone HTML template (MathJax, nav, print styles) for research memos, dashboards, audit reports | Human asks for an HTML deliverable — especially one with equations, where `<pre>`/`<code>` won't render LaTeX |

## How to Use

1. **Identify the sub-skill** — match the human's request to the one-liner in the table above.
2. **Read the sub-skill's SKILL.md** — `swiss-knife/<name>/SKILL.md` has full instructions, script paths, and examples.
3. **Run the script** — each sub-skill bundles its own executable scripts. Follow the sub-skill's README for the exact command.

## Adding New Sub-Skills

To add a new utility to the swiss-knife:

1. Create a folder: `swiss-knife/<name>/`
2. Add a `SKILL.md` with frontmatter (`name`, `description`, `version`) and full usage instructions
3. Add any scripts/assets in a `scripts/` subfolder
4. Update the table above with a one-liner
5. Re-extract the embedded skill bundle to disk: human runs `lingtai-tui bootstrap` (no TUI restart needed — `~/.lingtai-tui/utilities/` is rewritten from the TUI binary's embed.FS). Then refresh the catalog: `skills(action='info')` or `system(action='refresh')`. The catalog rescan alone is NOT enough — it only reads what's on disk; bootstrap is what gets new files there.

## Design Philosophy

Each sub-skill follows these principles:
- **Self-contained** — all code and assets live in the sub-skill folder
- **Single-purpose** — one sub-skill does one thing well
- **Documented** — SKILL.md has enough context to use without reading source code
- **Small** — if it's bigger than ~200 lines of code, it probably deserves its own top-level skill
