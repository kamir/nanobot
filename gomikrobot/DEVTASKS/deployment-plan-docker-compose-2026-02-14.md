# Deployment Plan — Docker Compose (Local Volumes, Local Images)

## Goal
Provide a safe deployment plan using `docker-compose` with local folders mounted via `-v`, and ensure `make` uses locally built images (no remote pulls).

## Assumptions
- Repos live on the host and must be mounted into the container.
- Images are built locally.
- No external registry required for release.

## Plan
1. **Dockerfile Review**
   - Ensure the Dockerfile builds the `gomikrobot` binary.
   - Ensure runtime image includes only needed assets.

2. **Compose File**
   - Create `docker-compose.yml` with:
     - Service `gomikrobot`
     - `build: .` and `image: gomikrobot:local`
     - `pull_policy: never`
     - Volume mounts:
       - System repo: `- /path/to/system-repo:/opt/system-repo`
       - Work repo: `- /path/to/work-repo:/opt/work-repo`
       - Config: `- ~/.gomikrobot:/root/.gomikrobot`
   - Expose ports for gateway API/UI.

3. **Make Targets**
   - `make docker-build`: build `gomikrobot:local`.
   - `make docker-up`: `docker compose up` using local image only.
   - `make docker-down`: `docker compose down`.
   - `make docker-logs`: `docker compose logs -f`.

4. **Environment Configuration**
   - Pass API keys via `.env` or mounted config file.
   - Document required env vars and mounted paths.

5. **Safety Checks**
   - Verify mounted paths are read/write as required.
   - Confirm `pull_policy: never` prevents external pulls.
   - Confirm bot system repo is mounted read/write.

## Acceptance
- `make docker-build` builds image locally.
- `make docker-up` runs without pulling remote images.
- System repo and work repo are mounted from host and usable inside container.
- Gateway UI and API are reachable on configured ports.

