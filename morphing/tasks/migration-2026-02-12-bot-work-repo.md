# Migration Tasks – 2026-02-12 – Bot Work Repo

> Task Planner Agent
> Scope: Go native

---

## TASK-001 – Config: Work Repo Path
**Goal:** Add config option for a dedicated work repo path.

**Steps:**
1. Add `agents.workRepoPath` (or similar) to config schema + env override.
2. Default to `~/.gomikrobot/work-repo`.

**Acceptance Criteria:**
- [ ] Path is configurable via JSON + env.
- [ ] Default path is applied when not set.

**Rollback:**
- [ ] Remove config option.

---

## TASK-002 – Repo Initialization
**Goal:** Auto‑initialize the work repo on first use.

**Steps:**
1. Ensure directory exists (mkdir -p).
2. If `.git` missing, run `git init` (best effort).
3. Warn if Git is missing and proceed without it.

**Acceptance Criteria:**
- [ ] Repo directory exists after first run.
- [ ] Git repo initialized when git is present.
- [ ] Warning logged if Git is absent.

**Rollback:**
- [ ] Remove initialization logic.

---

## TASK-003 – Enforce Exclusive Write Target
**Goal:** Bot writes only inside the work repo by default.

**Steps:**
1. Adjust tool layer (write/edit) to enforce path under work repo unless user overrides.
2. Add a safe override mechanism (explicit user instruction).

**Acceptance Criteria:**
- [ ] Default writes only to work repo.
- [ ] Explicit override required to write elsewhere.

**Rollback:**
- [ ] Revert write‑path enforcement.

---

## TASK-004 – Artifact Routing
**Goal:** Route artifacts into `/requirements`, `/tasks`, `/docs` under the work repo.

**Steps:**
1. Add helper for resolving default artifact path by type.
2. Update Action Policy to mention work repo paths.

**Acceptance Criteria:**
- [ ] Artifacts land in correct subfolders.

**Rollback:**
- [ ] Revert routing helper.
