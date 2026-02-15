# WhatsApp Setup Plan — 2026-02-14

## Goal
Provide a secure WhatsApp setup flow with default‑deny access, token verification, and explicit whitelist approval.

## Steps
1. **CLI Setup Command**
   - Add `gomikrobot whatsapp-setup` with prompts for enable, token, and initial allow/deny lists.
   - Persist config and auth settings.
2. **Auth Storage**
   - Use timeline settings: `whatsapp_pair_token`, `whatsapp_allowlist`, `whatsapp_denylist`, `whatsapp_pending`.
3. **Channel Enforcement**
   - Default deny if allowlist empty.
   - Token submission adds sender to pending list only.
   - Only allowlisted senders are forwarded to the agent.
4. **Web UI**
   - Add WhatsApp auth panel for token + allow/deny/pending lists.
   - Approve pending -> allowlist.
   - Deny pending -> denylist.
5. **Docs**
   - Document CLI and web flows, security rules, and settings.

## Acceptance
- Unauthorized users never receive responses.
- Token submission adds to pending only.
- Only allowlisted senders are processed.
- CLI and web flows can manage lists and token.

