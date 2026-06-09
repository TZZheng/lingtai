# /presets

Open the preset library for the current agent.

## Usage

```
/presets
```

## What it does

Opens the preset library scoped to the current agent. It shows only the presets
in this agent's `manifest.preset.allowed` list — exactly the ones you can switch
to with `/refresh <name>`. The currently-active preset is marked with ●.

You can view each preset's LLM and capabilities, and tag it with a 1–5 star
cost/quality tier (higher is better). Tags propagate to agents and are used for
daemon/avatar selection.

This view is read-only inspection plus tag editing — full preset creation
happens in `/setup`.

## When to use it

When you want to see which presets an agent is allowed to switch to, compare
their configs, or set quality tiers that guide automatic selection.
