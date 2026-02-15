# Workspace Policy

## Fixed Workspace Path
GoMikroBot uses a **fixed workspace** for state:
```
~/GoMikroBot-Workspace
```

## What Lives There
- Bot state (sessions, media, runtime artifacts)
- Persistent local data needed by the bot

## What Does NOT Live There
- Project/work repos (these are selectable and changeable)
- Bot system repo (identity repo)

## Rules
1. Workspace is **always** `~/GoMikroBot-Workspace`.
2. Switching work repo does not change workspace.
3. All state remains in the workspace regardless of work repo selection.

