# Architecture: Timeline & Memory (QMD/Qdrant)

## Objective
Create a unified, auditable timeline of all agent interactions (Text, Audio, Media) stored in SQLite, backed by a Vector Database (QMD/Qdrant) for semantic memory, and visualized via a Single Page Application (SPA).

## 1. Terminology & Tools
*   **Timeline DB (SQLite)**: The source of truth for the chronological history.
*   **Vector DB (QMD/Qdrant)**: "Queen/Quantum Memory Data" strategy using **Qdrant** for semantic indexing of all content.
*   **SPA**: A web interface to browse this history.

## 2. Data Architecture

### 2.1 SQLite Schema (`timeline` table)
We will extend the existing SQLite setup to include a dedicated, query-optimized timeline table.

```sql
CREATE TABLE timeline (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id TEXT UNIQUE,          -- UUID or WhatsApp MessageID
    timestamp DATETIME,            -- UTC
    sender_id TEXT,                -- Phone number (e.g., 4917...)
    sender_name TEXT,              -- Display name
    event_type TEXT,               -- "TEXT", "AUDIO", "IMAGE", "SYSTEM"
    content_text TEXT,             -- Raw text or Transcript
    media_path TEXT,               -- Local path to .ogg, .jpg
    vector_id TEXT,                -- Reference to Qdrant point
    classification TEXT            -- ABM1 Category ( EMERGENCY, ETC )
);
```

### 2.2 Vector Database (Qdrant)
We will run a local Qdrant instance (Docker or embedded if/when available for Go, but Docker is standard).
*   **Collection**: `agent_memory`
*   **Vectors**: Generated via OpenAI Embeddings (text-embedding-3-small) or local generic model.
*   **Payload**: 
    ```json
    {
      "text": "...",
      "sender": "...",
      "timestamp": 1234567890,
      "type": "audio_transcript"
    }
    ```

## 3. SPA Design (Timeline UI)

### 3.1 Stack
*   **Backend**: GoMikroBot Gateway (serving static files + REST API).
*   **Frontend**: Single HTML file with **Vue.js** (CDN) or **Alpine.js** for simplicity and "zero build" deployment. TailwindCSS for styling.

### 3.2 Features
*   **Visual Timeline**: A vertical stream of cards.
    *   **Left**: Agent responses.
    *   **Right**: User messages.
*   **Media Support**:
    *   Audio: Native HTML5 `<audio>` player for OGG files (Whatsapp voice notes).
    *   Images: Click-to-expand lightbox.
*   **Filtering (Focus Mode)**:
    *   Dropdown: "Select User" (e.g., Me).
    *   **Logic**: When a user is selected, their messages remain fully opaque (100%). Messages from *other* users (or system logs relevant to others) become semi-transparent (opacity 0.3) but remain visible to provide context of the agent's overall load.

## 4. Workflows

### 4.1 Ingestion (GoMikroBot)
1.  **Receive Message** (WhatsApp/System).
2.  **Process**: Transcribe audio, save media.
3.  **Log**: Insert into SQLite `timeline`.
4.  **Embed**: Generate vector -> Upsert to Qdrant.

### 4.2 Retrieval (SPA)
1.  **Frontend** calls `GET /api/timeline?limit=100`.
2.  **Backend** queries SQLite.
3.  **Frontend** renders the list.

## 5. Implementation Tasks

### Phase 1: Storage & API
*   [ ] **TASK-QMD-001**: Design and apply SQLite `timeline` migration.
*   [ ] **TASK-QMD-002**: Implement `TimelineService` in Go to handle insertions.
*   [ ] **TASK-QMD-003**: Integrate Qdrant client (Go) and implement `VectorService` for embeddings.
*   [ ] **TASK-QMD-004**: Create REST API endpoints (`GET /api/v1/timeline`, `GET /api/v1/users`).

### Phase 2: Frontend (SPA)
*   [ ] **TASK-SPA-001**: Create `timeline.html` skeleton with Tailwind + Vue3.
*   [ ] **TASK-SPA-002**: Implement message fetching and rendering loop.
*   [ ] **TASK-SPA-003**: Add "Focus Mode" filter (opacity logic).
*   [ ] **TASK-SPA-004**: Add Media Player support (Audio/Image).

### Phase 3: Integration
*   [ ] **TASK-INT-001**: Wire up WhatsApp incoming/outgoing events to `TimelineService`.
