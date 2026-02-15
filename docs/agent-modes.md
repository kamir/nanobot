# GoMikroBot CLI Modes

This document describes the runtime modes exposed by the `gomikrobot` CLI.

## Agent Mode
Command:
```
./gomikrobot agent -m "Hello"
```

Behavior:
- Loads config via `internal/config.Load()`.
- Ensures the work repo path exists (`EnsureWorkRepo`).
- Initializes the LLM provider (OpenAI, optionally Local Whisper wrapper).
- Creates an agent loop with:
  - Workspace (`Agents.Defaults.Workspace`)
  - Work repo (`Agents.Defaults.WorkRepoPath`)
  - System repo (`Agents.Defaults.SystemRepoPath`)
  - Model and iteration limits
- Sends the message to `Loop.ProcessDirect(...)` and prints the response.

Flags:
- `-m, --message` (required): message content
- `-s, --session` (optional): session ID (default `cli:default`)

Primary use:
- Single interactive message testing via CLI.

Source:
- `gomikrobot/cmd/gomikrobot/cmd/agent.go`

## Gateway Mode
Command:
```
./gomikrobot gateway
```

Behavior:
- Loads config and sets up timeline DB at `~/.gomikrobot/timeline.db`.
- Seeds settings (bot repo path, default work repo, repo search path, etc.).
- Resolves work repo and system repo path (settings override config).
- Starts:
  - Message bus
  - LLM provider
  - Agent loop
  - WhatsApp channel
  - Web UI routing
  - Local HTTP server
- Routes outbound from Web UI to WhatsApp (with silent‑mode rules).
- Logs timeline events for Web UI visibility.

Primary use:
- Run the full multi‑channel gateway with WhatsApp + Web UI.

Source:
- `gomikrobot/cmd/gomikrobot/cmd/gateway.go`

## Other Modes

### WhatsApp Setup
Command:
```
./gomikrobot whatsapp-setup
```
Behavior:
- Interactive CLI prompts.
- Updates config to enable WhatsApp.
- Stores token + allow/deny lists in timeline settings.

Source:
- `gomikrobot/cmd/gomikrobot/cmd/whatsapp_setup.go`

### WhatsApp Auth
Command:
```
./gomikrobot whatsapp-auth --approve <jid>
./gomikrobot whatsapp-auth --deny <jid>
./gomikrobot whatsapp-auth --list
```
Behavior:
- Approve/deny a pending JID without the Web UI.
- List allow/deny/pending.

Source:
- `gomikrobot/cmd/gomikrobot/cmd/whatsapp_auth.go`

### Onboard
Command:
```
./gomikrobot onboard
```
Behavior:
- Creates default config at `~/.gomikrobot/config.json`.
- Prints next‑steps guidance.

Source:
- `gomikrobot/cmd/gomikrobot/cmd/onboard.go`

### Status
Command:
```
./gomikrobot status
```
Behavior:
- Prints version and checks if config exists.

Source:
- `gomikrobot/cmd/gomikrobot/cmd/status.go`

### Version
Command:
```
./gomikrobot version
```
Behavior:
- Prints version string.

Source:
- `gomikrobot/cmd/gomikrobot/cmd/status.go`

### Install
Command:
```
./gomikrobot install
```
Behavior:
- Installs the current binary to `/usr/local/bin/gomikrobot`.
- If permissions fail, run with `sudo`.

Source:
- `gomikrobot/cmd/gomikrobot/cmd/install.go`
