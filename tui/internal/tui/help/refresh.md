# /refresh

Hard restart the agent.

## Usage

```
/refresh           # restart, reverting to the default preset
/refresh <name>    # switch to preset <name>, then relaunch (e.g. /refresh mimo)
/refresh all       # refresh every agent in the project
```

## What it does

Reloads `init.json`, capabilities, and all configuration from disk, then
relaunches the agent. Passing a preset name switches the active preset before
relaunching; the preset must be in the agent's `manifest.preset.allowed` list
(see `/presets`).

## When to use it

The go-to command after editing configuration, or to switch the agent onto a
different LLM/capability preset. It is the preferred restart for most cases —
reach for `/cpr` only when an agent is actually down.
