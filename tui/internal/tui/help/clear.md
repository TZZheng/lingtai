# /clear

Clear the agent's context window and restart.

## Usage

```
/clear
```

## What it does

Clears the agent's entire context window and restarts it with a blank
conversation. Identity, pad, and codex are preserved — only the live
conversation is wiped.

## When to use it

When the conversation has accumulated noise or gone off track and you want a
clean slate without losing the agent's durable memory. To preserve context by
saving it first, use `/molt` instead.
