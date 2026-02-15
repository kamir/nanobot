# WhatsApp Setup (Default Deny)

## Overview
WhatsApp access is **default‑deny**. Only explicitly whitelisted senders can reach the bot.

## CLI Setup
Command:
```
./gomikrobot whatsapp-setup
```

Prompts:
- Enable WhatsApp
- Pairing token (share out‑of‑band)
- Initial allowlist
- Initial denylist

Notes:
- Settings are stored in `~/.gomikrobot/timeline.db`.
- After changes, restart `./gomikrobot gateway` to apply.

## CLI Approve/Deny (No Web UI)
Commands:
```
./gomikrobot whatsapp-auth --approve <jid>
./gomikrobot whatsapp-auth --deny <jid>
./gomikrobot whatsapp-auth --list
```
Use this to manage pending approvals directly from the terminal.

## Web UI Setup
Open Config Manager in the Web UI:
- Set pairing token
- Edit allowlist / denylist
- Review pending (token submitted)
- Approve or blacklist pending

## Token Verification
- Unknown senders who submit the token are added to **Pending**.
- They are still unauthorized until you whitelist them.

## Storage Keys
- `whatsapp_pair_token`
- `whatsapp_allowlist`
- `whatsapp_denylist`
- `whatsapp_pending`

## Security Rules
- If allowlist is empty, **nobody is authorized**.
- Denylist always blocks.
- No automatic responses to unauthorized senders.
