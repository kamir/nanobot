# Standalone Desktop Mode

## Summary
When GoMikroBot runs in Standalone Desktop mode, all group collaboration
features are disabled. The bot operates as a single-agent personal assistant.

## Mode Determination
Computed at gateway startup from config:
- `standalone` — default: Group.Enabled=false, Orchestrator.Enabled=false, Host!="0.0.0.0"
- `group` — Group.Enabled=true
- `full` — Group.Enabled=true AND Orchestrator.Enabled=true
- `headless` — Gateway.Host="0.0.0.0"

## Standalone Rules
1. No auto-rejoin: DB-persisted group_active settings are ignored at startup
2. API blocked: All mutating /api/v1/group/* endpoints return 403 Forbidden
3. UI hidden: index.html hides GROUP MGMT/APPROVALS cards; /group redirects to /timeline
4. Audit: SYSTEM event with classification=MODE_CHANGE logged at startup
5. Read-only group endpoints remain accessible (return empty data)

## Audit Event Schema
- EventType: "SYSTEM"
- Classification: "MODE_CHANGE"
- Metadata: {"mode":"standalone","event":"ENTER_STANDALONE"}
- Visible in unified audit log as source "mode_change"

## Switching Modes
Mode is a startup-time decision from config. To switch:
- Edit ~/.gomikrobot/config.json (set group.enabled, orchestrator.enabled)
- Restart the gateway
Previously joined groups remain in DB for re-joining when mode changes.
