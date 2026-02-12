## 1. Identity
*   **Name**: GoMikroBot
*   **Repo**: `github.com/kamir/gomikrobot` (proposed)
*   **Binary**: `gomikrobot`
*   **Config Dir**: `~/.gomikrobot/`
*   **Env Prefix**: `MIKROBOT_` (e.g., `MIKROBOT_TOKEN`)

## 2. Mission
To provide a statically compiled, single-binary AI agent framework that is:
*   **Micro**: Tiny footprint (vs Python's venv + deps).
*   **Fast**: Instant startup.
*   **Secure**: Compiled, type-safe, sandboxed.

## 3. Visual Identity (NanoBanana Requests)
We need a new logo and visual assets. 
*   **Theme**: Minimalist, futuristic, "Go" cyan + Robot metallic.
*   **Mascot**: A tiny, efficient robot holding a Go gopher or structured like a Gopher.

## 4. Community Structure (`./community`)
*   **Start**: `community/README.md`
*   **Channels**: 
    *   GitHub Discussions (Development)
    *   Discord (Users)
*   **Content**:
    *   Showcase of skills.
    *   "Micro-plugins" repository.

## 5. Migration Tasks
1.  **Rename Binary**: `cmd/nanobot` -> `cmd/gomicrobot`.
2.  **Update CLI**: Strings, specific help text.
3.  **Update Config**: `Loader` to check `~/.gomicrobot`.
4.  **Update EnvVars**: `nanobot/gonanobot/internal/config/loader.go` to use `MICROBOT_` prefix.
5.  **WhatsApp Bridge**: Replace Node.js bridge with pure Go `whatsmeow` library (User Objective: "rewrite... in Go").
6.  **Skills**: Convert Python skills (`nanobot/skills/`) to Go tools.

## 6. Success Criteria
*   [ ] Binary runs as `gomicrobot`.
*   [ ] Config loads from `~/.gomicrobot/config.json`.
*   [ ] Env vars `MICROBOT_...` work.
*   [ ] No Node.js dependency (WhatsApp via `whatsmeow`).
*   [ ] "Soul" (System prompts) migrated to Go templates/embedded FS.
