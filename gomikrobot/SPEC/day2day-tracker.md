# Day2Day Task Tracker — Specification

## Purpose
Create a best‑in‑class Day2Day task tracker for GoMikroBot, with strict separation between:
- Bot Identity/System Repo (personal brain and skills)
- Project/Collaboration Repos (work context)

Day2Day data must only live inside the bot’s system repo.

## System Repo (Identity Repo)
- Root path (current default): `/Users/kamir/GITHUB.kamir/nanobot/gomikrobot`
- Must be cloned per machine, main branch.
- Contains bot skills, Day2Day data, identity metadata.

## Data Location (Non‑Negotiable)
- Day2Day data lives only here:
  - `operations/day2day/tasks/YYYY-MM-DD.md`
- No Day2Day data in memory files.
- No Day2Day data in work/project repos.

## Command Grammar (dt*)
Exact command tokens only. Commands are 3‑letter `dt*`.

### Commands
- `dtu` — Update task list (add tasks, log update).
- `dtp` — Progress entry only (log progress, no task mutation).
- `dts` — Consolidate state (dedupe tasks, recompute counts).
- `dtn` — Propose next step (single best open task).
- `dta` — Propose all open steps.
- `dtc` — Close capture (finalize multiline input).

### Multiline Capture
If a message is only `dtu` or `dtp`, the bot enters capture mode:
- Following messages are appended as content.
- Capture ends when `dtc` is received.
- Content is then applied as `dtu` or `dtp`.

## Status Requests
“Show today’s task status” should:
- Read from `operations/day2day/tasks/`.
- Support `today`, `yesterday`, `tomorrow`, or explicit `YYYY-MM-DD`.

## Daily File Format
```
# Day2Day — 2026-02-13 (Friday)

## Tasks
- [ ] Task

## Progress Log
- 17:15: PROGRESS — ...

## Notes / Context
- ...

## Consolidated State
- Open: N
- Done: M
- Last Consolidation: HH:MM

## Next Step
- ...
```

## UI Requirements
1. Dedicated Identity Repo panel:
   - Purple GitHub icon with silver glow.
   - Shows current system repo path.
2. Repo selector explicitly labeled as “Project/Collab Repo”.
3. Clear text that project repos are external context only.

## Dev Task Tracking
Development tasks live in `DEVTASK/` and are tied to SPEC/REQUIREMENTS.
They are not Day2Day tasks.

## Implementation Steps
1. Route all Day2Day read/write to `operations/day2day/tasks` under system repo.
2. Implement dt* parser with capture mode and dtc closing.
3. Add status handler for today/yesterday/tomorrow/explicit dates.
4. Add consolidation and planning commands.
5. Update UI with identity‑repo panel and repo separation copy.
6. Add tests to ensure Day2Day never writes to work repo.

## Acceptance Criteria
- Day2Day reads/writes never touch work repos.
- Status command uses system repo only.
- UI clearly distinguishes identity repo from project repos.
- dt* grammar is explicit and deterministic.
