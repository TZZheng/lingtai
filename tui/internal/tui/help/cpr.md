# /cpr

Revive a suspended or dead agent.

## Usage

```
/cpr           # the current agent
/cpr all       # every agent in the project
```

## What it does

Brings a suspended or dead agent back to life.

## When to use it

To resume an agent paused by `/sleep` or `/suspend`, or to recover one that has
died. For routine restarts prefer `/refresh` — `/cpr` is specifically for agents
that are down.
