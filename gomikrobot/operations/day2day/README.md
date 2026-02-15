# Day2Day Task Tracker

The Day2Day system manages daily task files at `operations/day2day/tasks/YYYY-MM-DD.md`.

## Commands (3-letter prefixes)

Users send these as the first word of a message. The bot intercepts them before they reach the LLM.

| Command | Name | Description |
|---------|------|-------------|
| `dtu <text>` | Update | Add new tasks and log an UPDATE entry. Multi-line: send `dtu` alone to start capture, then content, then `dtc` to close. |
| `dtp <text>` | Progress | Log a PROGRESS entry without adding tasks. Multi-line: send `dtp` alone to start capture, then content, then `dtc` to close. |
| `dts` | Consolidate | Deduplicate tasks, update consolidated state (open/done counts), and suggest next step. |
| `dtn` | Next | Show the next suggested task (first open checkbox). |
| `dta` | All | Show all open tasks for today. |
| `dtc` | Close capture | End a multi-line `dtu`/`dtp` capture session. |

## Status query

Any message containing **"status"** and one of **"task"**, **"aufgabe"**, or **"day2day"** triggers a status report for today (or a specific date if included as `YYYY-MM-DD`, or "yesterday"/"tomorrow"/"gestern"/"morgen").

Example: `status tasks` or `day2day task status 2026-02-14`

## Task file format

```markdown
# Day2Day — 2026-02-14 (Friday)

## Tasks
- [ ] Open task
- [x] Completed task

## Progress Log
- 14:30: UPDATE — Added deployment steps
- 15:00: PROGRESS — Reviewed PR #42

## Notes / Context

## Consolidated State
- Open: 3
- Done: 2
- Last Consolidation: 16:00

## Next Step
- Deploy staging environment
```

## Important

- If a user asks about day2day commands in natural language, explain the commands listed above.
- The commands are **case-insensitive** and must be the **first word** of the message.
- Task files are stored in the bot system repo under `operations/day2day/tasks/`.
