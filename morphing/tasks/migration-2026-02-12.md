# Migration Tasks – 2026-02-12

> Task Planner Agent
> Scope: Go native WhatsApp strong silent-inbound mode

---

## TASK-001 – Enforce Silent Mode in Go WhatsApp Channel
**Goal:** Hard block outbound WhatsApp sends when silent mode is enabled.

**Steps:**
1. Ensure silent-mode guard is the final send gate in `gomikrobot/internal/channels/whatsapp.go`.
2. Log suppressed outbound attempts with recipient and reason.

**Acceptance Criteria:**
- [ ] No outbound send when `silent_mode` is true.
- [ ] Suppressed outbound attempts are logged with recipient and reason.

**Parity Test:**
- [ ] Unit test: silent mode true -> no send call.

**Rollback:**
- [ ] Revert guard/log changes.

---

## TASK-002 – Preserve Safe Default & Persistence
**Goal:** Silent mode defaults to true when unset, and persists when toggled.

**Steps:**
1. Confirm `TimelineService.IsSilentMode()` returns true on missing setting.
2. Ensure settings API/UI toggle persists `silent_mode` across restarts.

**Acceptance Criteria:**
- [ ] Default silent mode is true when no setting exists.
- [ ] Toggle persists across restart.

**Parity Test:**
- [ ] Restart and verify `silent_mode` remains enabled unless explicitly disabled.

**Rollback:**
- [ ] Revert settings changes (if any).

---

## TASK-003 – Documentation
**Goal:** Document strong silent-inbound mode as a Go-native requirement.

**Steps:**
1. Update requirement doc to Go-native scope only.
2. Link to code locations providing proof (guard + default).

**Acceptance Criteria:**
- [ ] Requirement doc reflects Go-native scope and strong guarantee.

**Rollback:**
- [ ] Revert requirement doc edits.
