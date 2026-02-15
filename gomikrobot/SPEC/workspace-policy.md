# Workspace Policy (GoMikroBot)

## Summary
The **workspace** is the bot’s state home and is **always** fixed to:
```
~/GoMikroBot-Workspace
```

This is independent of which work repo is currently selected.

## Rules
1. **Workspace is fixed** and does not change per project.
2. **State lives in workspace** (sessions, media, bot state).
3. **Work repo is switchable** for interactions and outputs only.
4. **System repo is the start point** (the cloned bot repo), but state does not live there.

## Implications
- Changing work repo will not affect any bot state.
- The workspace location is not configurable.
