# /sleep

Put the agent to sleep.

## Usage

```
/sleep         # the current agent
/sleep all     # every agent in the project
```

## What it does

Sleeping pauses the agent while preserving its full state. A sleeping agent can
be woken later with `/cpr`.

## When to use it

When you want to stop an agent from consuming resources but intend to resume it
exactly where it left off. For a harder freeze of the OS process, use
`/suspend`.
