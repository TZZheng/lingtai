# Slash commands

Slash commands drive everything in `lingtai-tui` beyond plain chat. They open
views, manage the agent lifecycle, and configure the runtime.

## Using the command palette

- Type `/` in the message box to open the **command palette**.
- Keep typing to filter. Matching is fuzzy: `/skl` finds `skills`, `/ref` finds
  `refresh`.
- `↑`/`↓` move the selection, `Enter` runs the highlighted command.
- Some commands take an argument typed after the name, e.g. `/sleep all` or
  `/refresh mimo`.

## Browsing this help

The sidebar on the left lists every command. Use `↑`/`↓` to move, `Enter` on a
group header to collapse or expand it, and `Tab` to move focus to the content
panel so `PgUp`/`PgDn` scroll the page. `Ctrl+E` exports the current page to
`~/Downloads`. Press `Esc` or `q` to return to the mail view.

## Commands at a glance

### Talking to the agent
- `/btw` — ask a side question without interrupting the agent's work.
- `/insights` — request 2–3 concrete observations about the current task now.

### Agent lifecycle
- `/sleep` — put the agent to sleep (resumable with `/cpr`).
- `/suspend` — freeze the agent process entirely (resumable with `/cpr`).
- `/cpr` — revive a suspended or dead agent.
- `/refresh` — hard restart: reload config from disk, optionally switch presets.
- `/clear` — clear the conversation window, keep identity/pad/codex.
- `/molt` — force the agent to molt now (save context, reset the window).
- `/nirvana` — wipe everything and start fresh. Irreversible.

### Inspecting the agent
- `/kanban` — network dashboard of every agent's config and context usage.
- `/skills` — browse the agent's skill catalog.
- `/knowledge` — browse the agent's private knowledge (aliases: `/library`,
  `/codex`).
- `/system` — browse the agent's system files (system.md, covenant, …).
- `/daemons` — inspect per-agent daemon runs and their records.
- `/presets` — view the presets this agent can switch to with `/refresh`.

### Network & sharing
- `/mailbox` — browse all mailbox messages and attachments.
- `/projects` — browse registered projects and their networks.
- `/agora` — browse exported networks and recipes from the marketplace.
- `/export` — export a reusable recipe for sharing.
- `/viz` — open the network visualization in the browser.

### Configuration & diagnostics
- `/setup` — agent setup wizard (provider, model, capabilities).
- `/settings` — TUI preferences (theme, page size, language).
- `/mcp` — MCP control panel for external-service bridges.
- `/doctor` — diagnose connection issues.
- `/login` — check and manage saved credentials.

### This view & exit
- `/help` — open this help browser.
- `/quit` — quit `lingtai-tui` (agents keep running in the background).
