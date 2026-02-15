# Release Process

## Versioning
The CLI uses semantic versioning (`MAJOR.MINOR.PATCH`). The version is defined in:
`gomikrobot/cmd/gomikrobot/cmd/root.go`

It can be overridden at build time:
```
go build -ldflags "-X github.com/kamir/gomikrobot/cmd/gomikrobot/cmd.version=1.2.3" ./cmd/gomikrobot
```

## Make Targets
From `gomikrobot/`:
```
make release          # bumps MAJOR, builds
make release-major    # bumps MAJOR, builds
make release-minor    # bumps MINOR, builds
make release-patch    # bumps PATCH, builds
```

`make release*` will also:
- `git commit -m "Release vX.Y.Z"`
- `git tag vX.Y.Z`
- `git push`
- `git push --tags`

Note: release commits are created from the **repository root** (all changes in the repo are included).

## GitHub Actions
- Workflow: `.github/workflows/release-go.yml`
- Trigger: tag push `v*` or manual `workflow_dispatch`
- Builds artifacts for Linux, macOS, and Windows and attaches them to the GitHub release.

## Script
Release bump logic is in:
`gomikrobot/scripts/release.sh`
