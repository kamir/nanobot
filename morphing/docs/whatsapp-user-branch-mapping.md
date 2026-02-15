# WhatsApp User ↔ Git Branch Mapping (Proposed Behavior)

## Goal
Allow each selected WhatsApp user (via Web-UI) to be associated with a specific Git branch so repo actions can operate in the correct context.

## Scope (Documentation Only)
This document describes intended behavior. No implementation is planned today.

## Data Model
- Extend the web user record with:
  - `git_branch` (string)
  - Optional future: `repo_path` (string) for multi-repo support

## UI Behavior
- When a web user is selected, show a **Branch** dropdown beside WhatsApp link controls.
- Branch list is fetched from `/api/v1/repo/branches`.
- Selecting a branch persists it to the web user record.
- If no branch is set, show a subtle warning: “No branch assigned (defaulting to main)”.

## Runtime Behavior
- On inbound message from a linked WhatsApp user:
  - Resolve the user’s `git_branch`.
  - Auto-checkout that branch before any repo actions.
  - If the branch is missing or invalid:
    - Fall back to `main`.
    - Log a warning in timeline/console.

## Matching Strategy (Optional Suggestions)
We should not auto-bind without confirmation.

If we add a suggestion hint:
1. Exact match on username ↔ branch name.
2. Partial match on first token (lowercased).
3. Show a “Suggested branch” badge, but require explicit user confirmation.

## Safety Rules
- Only allow branches from the repo’s actual branch list.
- If there are uncommitted changes:
  - Default: block auto-checkout and warn.
  - Optional: allow auto-stash if user toggles a “Safe Checkout” option.

## Open Questions
- Should auto-checkout happen on every inbound message or only on explicit repo actions?
- Should per-user branch be mandatory before enabling repo actions?
- Should repo branch selection live in user onboarding or the main chat panel?

