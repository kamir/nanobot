# Memory Architecture Research Insights

> Captured 2026-02-16. Sources: ER1 Memory Service API (v1.1.0), Mastra AI Framework (mastra-ai/mastra).

---

## 1. ER1 Memory Service

### Overview

ER1 is a capture-first personal memory API (OpenAPI 3.0.3, v1.1.0) running at `localhost:8080`. It captures multimodal memories (audio + image + transcript) scoped by user context, with structured Q&A interactions.

### API Surface

| Domain | Key Endpoints | Purpose |
|--------|--------------|---------|
| **User** | `POST /user/access` | Login/provision, returns `ctx_id` + tier + role presets |
| **Memory** | `GET/POST/PATCH/DELETE /memory/{ctx_id}` | CRUD for multimodal memories |
| **Memory Media** | `GET /memory/{ctx_id}/{id}/audio\|image` | Binary media retrieval |
| **Tags** | `POST /memory/{ctx_id}/add_tags` | Bulk tagging across memories |
| **QnA** | `GET /qna/topics`, `GET/POST /qna/interactions` | Threaded Q&A with topic grouping |

Authentication: API Key via `X-API-KEY` header.

### Data Model

```
Memory {
  id, type ("record"), audio (bool), image (bool),
  description, transcript, transcript_status ("processed"|"pending"),
  review_status ("n"|"y"), location_lat/lon, created_at, tags[]
}

Context (ctx_id) — namespace for all memories, bound to user identity + tier
UserRole — label_color, label_icon_key, categories[], tags[] (presets per role)
QnAInteraction — topicID, messages[{messageType, content, timestamp, location}], summary
```

### Strengths vs. GoMikroBot

| ER1 Has | GoMikroBot Has |
|---------|---------------|
| Multimodal capture (audio/image binaries) | Semantic vector search (cosine similarity) |
| Geolocation on every memory | Distributed multi-agent memory (Kafka) |
| Automatic transcription pipeline | TTL-based lifecycle management |
| Tier-based access control + role presets | Expertise tracking with trends |
| Structured QnA threads with topics | Soul file identity system |
| Temporal filtering (startDate/endDate) | Source-prefix filtering |

### Integration Insight

ER1 is a **capture layer** — it excels at recording raw personal experiences. GoMikroBot is a **reasoning layer** — it excels at semantic retrieval and multi-agent collaboration. The natural integration: ER1 captures, GoMikroBot indexes transcripts for semantic search via a new `er1:` source prefix.

---

## 2. Mastra AI Framework

### Overview

Mastra is a TypeScript AI agent framework (YC-backed, from the Gatsby team). Their memory system is one of the most thoroughly designed in the open-source ecosystem. Key innovation: **Observational Memory** which achieved 94.87% on LongMemEval with gpt-5-mini.

### Four-Tier Memory Architecture

#### Tier 1: Message History (Short-Term)
- Recent N messages from current conversation
- Configurable via `lastMessages` parameter
- SQL-backed (PostgreSQL, MongoDB, libSQL)
- **GoMikroBot equivalent:** Session JSONL history

#### Tier 2: Working Memory (Persistent Scratchpad)
- Continuously updated key user/task information
- **Two scoping modes:**
  - **Resource-scoped** (default): Persists across ALL threads for same user
  - **Thread-scoped**: Isolated per conversation
- **Two format modes:**
  - **Markdown template**: Replace semantics (agent rewrites full content)
  - **Zod schema (JSON)**: Merge semantics (agent updates only changed fields)
- Updated via `updateWorkingMemory` tool the agent calls
- Can be pre-seeded on thread creation
- Supports read-only mode for sub-agents
- **GoMikroBot equivalent:** `MEMORY.md` (global only, no merge, no scoping)

#### Tier 3: Semantic Recall (Associative/Long-Term)
- RAG-based retrieval via vector embeddings
- Embeds all messages after each LLM response
- Configurable `topK`, `messageRange` (context window around matches)
- **17 vector store providers** (Pinecone, Qdrant, PgVector, LanceDB, etc.)
- **GoMikroBot equivalent:** `MemoryService.Search` + `AutoIndexer` (2 stores: SQLite-vec, Qdrant)

#### Tier 4: Observational Memory (Novel — Agentic Compression)
- **Two background LLM agents compress conversation history:**
  - **Observer**: When unobserved messages exceed ~30K tokens, compresses into prioritized, date-anchored observation notes
  - **Reflector**: When observations exceed ~40K tokens, consolidates (GC for memory)
- **Three-date temporal anchoring**: observation date, referenced date, relative offset
- **Compression ratios**: 3-6x for text, 5-40x for tool-heavy workloads
- **Async buffering**: Pre-computes at 20% intervals to avoid conversation pauses
- **Prompt cacheability**: Append-only observation log enables 4-10x cost reduction
- **SOTA results**: gpt-4o 84.23%, Gemini-3-Pro 93.27%, gpt-5-mini 94.87% on LongMemEval
- **GoMikroBot equivalent:** Nothing — this is a major gap

### Key Design Patterns

#### 1. Resource/Thread Model
- `resourceId`: Stable user identifier (never changes)
- `threadId`: Unique conversation/session ID
- One resource → many threads
- Working memory scoped at either level
- Multiple agents can share threads

#### 2. Memory Processor Pipeline
- **Input phase**: Memory processors run first (load history, search vectors, load working memory), then guardrails
- **Output phase**: Guardrails run first, then memory processors persist. If guardrail aborts, no memory saved.
- Clean separation of concerns with abort semantics

#### 3. Append-Only Context for Cacheability
- Observations block (top, stable) + raw messages block (bottom, growing)
- Provider-side prompt caching works because the prefix is stable
- 4-10x cost reduction in practice

### Lessons for GoMikroBot

**High-value adoptions:**
1. **Observational Memory** — Background goroutine using LLM to compress old conversation chunks into dated observation logs. Solves long-conversation degradation.
2. **Scoped Working Memory** — Per-user (`resourceId`) and per-thread (`threadId`) working memory instead of global `MEMORY.md`. Critical for multi-user WhatsApp.
3. **Prompt cacheability** — Structure context window as append-only/stable prefix for provider caching.

**Medium-value adoptions:**
4. Schema-based working memory with merge semantics (update fields, not rewrite)
5. Memory processor pipeline with abort semantics
6. `messageRange` around semantic recall matches (context windowing)

**GoMikroBot advantages Mastra lacks:**
- TTL-based lifecycle with per-source retention policies (operationally explicit)
- Expertise tracking with trend analysis (unique differentiator)
- Distributed group memory via Kafka/S3 (true multi-agent collaboration)
- Soul files for rich, file-based identity (richer than Mastra's `instructions` string)

---

## 3. Combined Insight Map

```
                    ER1 Capture Layer
                    (audio/image/transcript/QnA)
                           │
                    ┌──────▼──────┐
                    │  er1: source │  ← sync transcripts into vector store
                    └──────┬──────┘
                           │
    ┌──────────────────────┼──────────────────────────┐
    │              GoMikroBot Memory v2                │
    │                                                  │
    │  Layer 1: soul:*           (permanent, identity) │
    │  Layer 2: conversation:*   (30d, auto-indexed)   │
    │  Layer 3: tool:*           (14d, auto-indexed)   │
    │  Layer 4: group:*          (60d, Kafka/S3)       │
    │  Layer 5: er1:*            (permanent, ER1 sync) │  ← NEW
    │  Layer 6: observation:*    (permanent, compressed)│  ← NEW (from Mastra)
    │                                                  │
    │  Cross-cutting:                                  │
    │  - Scoped Working Memory (per-user/per-thread)   │  ← NEW (from Mastra)
    │  - Expertise Tracker (skill proficiency)         │
    │  - Lifecycle Manager (TTL pruning)               │
    └──────────────────────────────────────────────────┘
```

---

## References

- ER1 API Spec: `/Users/kamir/ER1/integrations/ER1/memory-service.yml`
- Mastra GitHub: https://github.com/mastra-ai/mastra
- Mastra Memory Docs: https://mastra.ai/docs/memory/overview
- Mastra Observational Memory: https://mastra.ai/research/observational-memory
- LongMemEval benchmark results: gpt-5-mini 94.87% with observational memory
