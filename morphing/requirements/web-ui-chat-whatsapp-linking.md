# Web UI Chat -> WhatsApp Routing Requirement

> Extracted by Requirements Tracker Agent
> Source: User request (2026-02-12). PH entry not found in repo; link pending.

---

## REQ-WEB-001 – Web UI Chat as External Inbound
**Type:** functional  
**Status:** to-add  

When the operator sends a message via the Web-UI Chat Box, the system must:
- Treat it **as if it were an external inbound message** (same agent path as other channels).
- Route the response **back to a linked personal WhatsApp channel**.
- Display the response in the Web-UI Chat Box timeline as a visible chat event.

---

## REQ-WEB-002 – Web User ↔ WhatsApp Link (Onboarding)
**Type:** functional  
**Status:** to-add  

The system must provide a simple onboarding flow that links:
- a **Web-User** (Web UI identity), to
- a **WhatsApp User ID** (WhatsApp JID or phone),
so outbound replies from the Web-UI Chat Box are delivered to the linked WhatsApp user and shown in the Web UI.

**Constraints:**
- No OAuth (simple user list only).
- A minimal user list for selecting/assigning the link.
- Should be explicit and reversible.

---

## REQ-WEB-003 – Web UI Visibility
**Type:** functional  
**Status:** to-add  

The Web UI must:
- Show inbound and outbound messages for the Web-User ↔ WhatsApp link.
- Clearly indicate delivery target and channel (WhatsApp).

---

## Acceptance Criteria
- [ ] Web UI chat input triggers the same agent loop used by external channels.
- [ ] Response is sent to the linked WhatsApp user.
- [ ] Response also appears in Web UI chat timeline.
- [ ] Onboarding links a Web-User to a WhatsApp ID without OAuth.
- [ ] Link is persisted and can be updated or removed.

---

## Traceability
- Source: User request 2026-02-12 (pending PH reference).
