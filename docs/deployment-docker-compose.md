# Docker Compose Deployment (Local Volumes, Local Images)

## Build Image
From `gomikrobot/`:
```
make docker-build
```

## Run
Set host repo paths and start (system repo defaults to current folder; work repo defaults to `~/GoMikroBot-Workspace`):
```
make docker-up
```

## Stop
```
make docker-down
```

## Logs
```
make docker-logs
```

## Notes
- Uses `gomikrobot:local` image only (no pulls).
- Mounts:
  - `SYSTEM_REPO_PATH` → `/opt/system-repo`
  - `WORK_REPO_PATH` → `/opt/work-repo`
  - `~/.gomikrobot` → `/root/.gomikrobot`
- Ports: `18790` (gateway), `18791` (dashboard)
