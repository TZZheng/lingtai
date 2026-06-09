# /doctor

Diagnose connection issues.

## Usage

```
/doctor
```

## What it does

Runs diagnostics on the agent's connectivity — checks API keys, model
availability, and network configuration — and reports what it finds.

## When to use it

When an agent fails to start or stops responding and you suspect a credential,
model, or network problem. Pair with `/login` to re-authenticate if `/doctor`
flags a credential issue.
