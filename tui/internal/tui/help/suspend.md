# /suspend

Suspend the agent.

## Usage

```
/suspend       # the current agent
/suspend all   # every agent in the project
```

## What it does

Freezes the agent process entirely. A suspended agent must be revived with
`/cpr` before it can resume.

## When to use it

When you want a hard stop of the underlying process — heavier than `/sleep`.
Reach for `/sleep` if you just want a resumable pause.
