# Bot Work Repo (Exclusive Write Target)

> Extracted by Requirements Tracker Agent
> Source: User request (2026-02-12). PH entry not found in repo; link pending.

---

## REQ-WORK-001 – Dedicated Bot Work Repo
**Type:** functional + safety constraint  
**Status:** to-add  

The system must provide a **dedicated local Git repository** ("bot-work-repo") that is the **exclusive write target** for the bot. This repo is used for bot‑generated artifacts (requirements, tasks, docs, drafts) to avoid writing into other folders by default.

**Key properties**
- Local, dedicated folder (configurable path).
- Git-initialized so all bot outputs are versioned.
- Write operations from the bot are constrained to this repo by default.
- Explicit override required to write elsewhere.

---

## REQ-WORK-002 – Safe Defaults & Isolation
**Type:** safety constraint  
**Status:** to-add  

- Default path should be a safe, dedicated folder (e.g., `~/.gomikrobot/work-repo`).
- If the repo does not exist, the system should initialize it (create folder + `git init`).
- If Git is unavailable, the system should still create the folder but warn that history tracking is disabled.

---

## REQ-WORK-003 – Artifact Routing
**Type:** functional  
**Status:** to-add  

When the bot creates artifacts, it must use this structure inside the work repo:
- `/requirements`
- `/tasks`
- `/docs`

These paths are the **default targets** unless the user explicitly requests a different location.

---

## Acceptance Criteria
- [ ] Bot outputs are written only to the work repo by default.
- [ ] Work repo auto‑initializes on first use.
- [ ] Git history records bot outputs when available.
- [ ] User can configure the path.
- [ ] Explicit override required for writing outside the work repo.

---

## Traceability
- Source: User request 2026-02-12 (pending PH reference).
