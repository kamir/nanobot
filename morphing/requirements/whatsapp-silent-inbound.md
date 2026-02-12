# WhatsApp Silent-Inbound Mode Requirement

> Extracted by Requirements Tracker Agent
> Source: User request (2026-02-12). PH entry not found in repo; link pending.

---

## REQ-WA-001 – Silent-Inbound Mode (Strong, Default-On, Go Native)
**Type:** functional + safety constraint  
**Status:** to-add  

When Silent-Inbound Mode is enabled, the system must:
- Receive and log inbound WhatsApp messages as usual.
- **Never** send any WhatsApp outbound message (including automatic replies, tool-driven sends, or scheduled jobs).
- Suppress outbound sends even if a response is generated internally.
- Emit a clear local audit log when an outbound send is suppressed.

The mode must be **default-on** after startup and reconnects unless explicitly disabled by the operator.

**Scope:**
- Applies to the native Go WhatsApp channel (`gomikrobot`).

**Rationale:**
User requires a strong guarantee that turning the Go WhatsApp gateway back on results in receive-only behavior with zero outbound messaging unless explicitly re-enabled.

**Acceptance Criteria:**
- [ ] On startup, silent mode is enabled by default (no explicit config needed to be safe).
- [ ] Any outbound attempt while silent is blocked and logged (with recipient and reason).
- [ ] Inbound messages are still processed, logged, and can be viewed in the UI.
- [ ] Manual toggle is explicit and persisted (silent on/off survives restart).
- [ ] Unit/integration test proves no outbound send call is made while silent is enabled.

**Traceability:**
- Source: User request 2026-02-12 (pending PH reference).
