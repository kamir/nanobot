# Migration Tasks – 2026-02-12 – Web UI Chat to WhatsApp Linking

> Task Planner Agent
> Scope: Go native gateway + timeline UI

---

## TASK-001 – Data Model for Web Users and Links
**Goal:** Persist Web-User ↔ WhatsApp mapping.

**Steps:**
1. Add tables to timeline DB (or a small dedicated DB):
   - `web_users(id, name, created_at)`
   - `web_links(web_user_id, whatsapp_jid, created_at, updated_at)`
2. Add CRUD methods in `TimelineService` or a new `LinkService`.

**Acceptance Criteria:**
- [ ] Link persisted and queryable by web_user_id.
- [ ] Link can be updated/removed.

**Parity Test:**
- [ ] Unit test: create link, fetch link, update link.

**Rollback:**
- [ ] Revert schema change and service methods.

---

## TASK-002 – Web UI Chat Inbound as External Message
**Goal:** Web UI chat input enters agent loop as a normal inbound message.

**Steps:**
1. Add API endpoint: `POST /api/v1/webchat/send` with `{ web_user_id, message }`.
2. Convert request into `bus.InboundMessage` with channel `webui`.
3. Ensure it flows through `agent.Loop` and emits outbound response.

**Acceptance Criteria:**
- [ ] Web chat messages reach the agent loop.
- [ ] Agent response published as outbound message with channel `webui`.

**Parity Test:**
- [ ] Integration test: POST -> response event stored and visible.

**Rollback:**
- [ ] Remove endpoint and routing.

---

## TASK-003 – Route Web UI Replies to Linked WhatsApp
**Goal:** Send the agent reply to the linked WhatsApp user and mirror in UI.

**Steps:**
1. Extend outbound dispatcher to detect `webui` channel replies.
2. Resolve `web_user_id` -> `whatsapp_jid` via link service.
3. Send outbound to WhatsApp channel with that JID.
4. Log timeline event for Web UI so it appears in the chat UI.

**Acceptance Criteria:**
- [ ] Reply sent to linked WhatsApp JID.
- [ ] Reply appears in Web UI timeline.
- [ ] If no link exists, error is returned to Web UI.

**Parity Test:**
- [ ] End-to-end test: web chat -> WhatsApp outbound + UI update.

**Rollback:**
- [ ] Remove routing override and timeline log.

---

## TASK-004 – Simple Onboarding UI
**Goal:** Create a minimal user list and link assignment UI (no OAuth).

**Steps:**
1. Add endpoints: list web users, create web user, link/unlink WhatsApp JID.
2. Add UI controls in `gomikrobot/web/timeline.html`:
   - Create/select web user
   - Assign WhatsApp JID
   - Show current link

**Acceptance Criteria:**
- [ ] User can create/select a Web-User.
- [ ] User can link/unlink a WhatsApp JID.
- [ ] Link persists across restarts.

**Parity Test:**
- [ ] Refresh UI and confirm link persists.

**Rollback:**
- [ ] Revert UI and endpoints.
