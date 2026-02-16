# Agent Hierarchy Design: 2-Layer Communication Model

**Status:** DRAFT v0.1
**Date:** 2026-02-16
**Based on:** "Running Agents — A Runbook for Smart Leaders" (Dr. Mirko Kämpf)
**Context:** GoMikroBot multi-agent architecture

---

## 1. Motivation

The "Running Agents" research defines three personal agent types that together form
a balanced communication ecosystem around a human individual:

- **Type A — World Interpreter**: Bridges the external world inward (translation,
  summarization, expert access, information retrieval)
- **Type B — World Exposer (Proxy)**: Represents me outward (avatar, proxy,
  answering on my behalf, 24/7 availability)
- **Type C — Cognitive Enhancer**: Works internally for me (reflection, journaling,
  decision support, bridging A→B)

The key insight from the research: **these three types must stay in balance**. If A
floods me with input, or B exposes me to too much demand, C becomes the
bottleneck. The agent triad is a *communication pattern*, not just a toolbox.

This design extends that pattern from individuals to teams and organizations across
two isolated layers, while keeping the interaction model human-like so that people
can accept and adopt it naturally.

---

## 2. Design Principles

1. **Human-mirrored interaction** — Agents communicate the way humans do:
   through conversation, delegation, handover, and feedback. No alien protocols.
2. **Encapsulation** — The inner workings of Layer 1 (the A/B/C triad) are not
   visible to Layer 2. A team sees a member agent as a single entity, not its
   internal subagents.
3. **Balance preservation** — Each layer maintains the A/B/C equilibrium
   independently. Overload at one layer does not cascade into the other.
4. **Communication over process** — Following the research: we optimize
   information flow and handover quality, not rigid process enforcement.
5. **Progressive adoption** — Start with individual agents, then form teams, then
   connect teams. Each level works standalone before composition.

---

## 3. The 2-Layer Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                    LAYER 2: TEAM / ORGANIZATION                     │
│                                                                     │
│  ┌──────────────┐    handover     ┌──────────────┐                  │
│  │   Team Alpha  │◄══════════════►│   Team Beta   │                 │
│  │              │   (B↔A bridge)  │               │                 │
│  │  Coordinator │                 │  Coordinator  │                 │
│  │  Shared Mem  │                 │  Shared Mem   │                 │
│  │  Team Policy │                 │  Team Policy  │                 │
│  └──┬───┬───┬──┘                  └──┬───┬───┬───┘                  │
│     │   │   │                        │   │   │                      │
│ ════╪═══╪═══╪════════════════════════╪═══╪═══╪═══ encapsulation ═══ │
│     │   │   │                        │   │   │         boundary     │
│  ┌──┴─┐┌┴──┐┌┴──┐                ┌──┴─┐┌┴──┐┌┴──┐                  │
│  │ M1 ││ M2││ M3│                │ M4 ││ M5││ M6│                  │
│  └────┘└───┘└───┘                └────┘└───┘└───┘                  │
│                                                                     │
│               LAYER 1: INDIVIDUAL AGENT TRIAD                       │
│                                                                     │
│  Each member (M1..M6) internally:                                   │
│                                                                     │
│  ┌────────────────────────────────────────┐                          │
│  │           Individual Agent              │                         │
│  │                                        │                         │
│  │   ┌─────────┐  ┌─────────┐  ┌───────┐ │                         │
│  │   │  A      │  │  C      │  │  B    │ │                         │
│  │   │ World   │─►│Cognitive│─►│ World │ │                         │
│  │   │Interpret│  │Enhancer │  │Exposer│ │                         │
│  │   │  (IN)   │◄─│ (CORE)  │◄─│ (OUT) │ │                         │
│  │   └─────────┘  └─────────┘  └───────┘ │                         │
│  │        ▲                        │      │                         │
│  │        │     human / agent      │      │                         │
│  │        └── external  input ─────┘      │                         │
│  │            external output             │                         │
│  └────────────────────────────────────────┘                          │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 4. Layer 1: The Individual Agent Triad

An individual agent — whether representing a human or an autonomous software
agent — is internally composed of three cooperating subagents.

### 4.1 Subagent A — World Interpreter (Inbound)

**Purpose:** Receive, filter, translate, and contextualize information from the
external world.

| Responsibility               | Example                                     |
|------------------------------|---------------------------------------------|
| Language translation         | Email in English → summary in German         |
| Information retrieval        | Search documents, query databases            |
| Expert consultation          | Access specialized knowledge via tools/RAG   |
| Format conversion            | Audio → text, image → description            |
| Relevance filtering          | Noise reduction, priority classification     |

**GoMikroBot mapping:** The existing `agent/context.go` context builder +
RAG injection + tool reads (`read_file`, `list_dir`, `recall`) + provider
transcription (Whisper).

### 4.2 Subagent C — Cognitive Enhancer (Core)

**Purpose:** Process, reflect, decide, and plan. The bridge between inbound
information (A) and outbound representation (B).

| Responsibility               | Example                                     |
|------------------------------|---------------------------------------------|
| Reflection & journaling      | Weekly review of recorded interactions       |
| Decision support             | Weighing options, risk assessment            |
| Task planning                | Breaking work into steps, prioritizing       |
| Pattern recognition          | Identifying recurring bottlenecks            |
| Learning path creation       | Skill gap analysis, improvement suggestions  |
| Balance monitoring           | Detecting A/B overload, adjusting flow       |

**GoMikroBot mapping:** The agent loop itself (`agent/loop.go`), cognitive
mode injection, the day2day task tracker, session history, and memory
service (`memory/`).

### 4.3 Subagent B — World Exposer (Outbound)

**Purpose:** Represent the individual to the outside world. Act as proxy,
avatar, and delivery interface.

| Responsibility               | Example                                     |
|------------------------------|---------------------------------------------|
| Proxy/Avatar responses       | Answering on behalf when unavailable         |
| Deliverable handover         | Submitting work products to team members     |
| Status broadcasting          | Availability, capability, progress updates   |
| Validation before delivery   | Pre-submission quality checks                |
| External communication       | Sending messages via WhatsApp, email, etc.   |

**GoMikroBot mapping:** The message bus outbound path (`bus/bus.go`),
channel implementations (`channels/`), group announcements and heartbeats
(`group/manager.go`), and the `AgentIdentity` envelope.

### 4.4 Internal Flow

```
External World                    Individual Agent
     │                    ┌─────────────────────────────┐
     │   incoming msg     │                             │
     ├───────────────────►│  [A] interpret, filter,     │
     │                    │      contextualize          │
     │                    │         │                   │
     │                    │         ▼                   │
     │                    │  [C] think, decide, plan    │
     │                    │      reflect, learn         │
     │                    │         │                   │
     │                    │         ▼                   │
     │   outgoing msg     │  [B] format, validate,     │
     │◄───────────────────│      deliver, represent    │
     │                    │                             │
     │                    └─────────────────────────────┘
```

The A→C→B flow is the primary path. But C also feeds back to A (requesting
more information) and receives feedback from B (delivery failures, rejection
signals). This creates internal feedback loops that enable self-correction
without exposing internal complexity to the outside.

**Critical balance rule:** If A is overwhelmed (too much input), C must
throttle or prioritize. If B is overwhelmed (too much demand from others),
C must queue or delegate. C itself must never become a silent bottleneck —
its overload should trigger escalation to the human or to Layer 2.

---

## 5. Layer 2: Team and Cross-Team Interaction

### 5.1 Team Formation

A team is a group of individual agents that share:

1. **Shared Memory** — A common knowledge space accessible to all members
   (maps to `group.{name}.memory.shared` Kafka topic)
2. **Coordinator Role** — One agent acts as team coordinator, responsible for
   task routing, conflict resolution, and cross-team communication
3. **Team Policy** — Agreed-upon rules for handover quality, response times,
   and escalation paths
4. **Roster** — Known members with their capabilities, availability, and roles

```
┌─────────────────────────── Team Alpha ──────────────────────────┐
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                  Shared Memory (Context)                  │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐                     │
│  │ Agent 1  │    │ Agent 2  │    │ Agent 3  │                    │
│  │(A₁,C₁,B₁)│   │(A₂,C₂,B₂)│   │(A₃,C₃,B₃)│                   │
│  │          │    │          │    │          │                    │
│  │ ★ Coord  │    │ Worker   │    │ Worker   │                    │
│  └─────┬────┘    └────┬─────┘    └────┬─────┘                    │
│        │              │               │                          │
│  ══════╪══════════════╪═══════════════╪═══ intra-team bus ═════  │
│        │              │               │                          │
│        ▼              ▼               ▼                          │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │              Team Communication Bus                       │   │
│  │         (delegation, handover, status, traces)            │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### 5.2 Intra-Team Communication

Within a team, agents interact through their B and A subagents:

- **Agent 1's B** hands over a deliverable → **Agent 2's A** receives and interprets it
- **Agent 2's C** processes, creates output → **Agent 2's B** delivers result back
- The coordinator's C monitors flow, detects bottlenecks, rebalances work

This mirrors how humans work in teams: you *hand things over* to colleagues,
they *interpret and process* them, and *deliver back*. The handover protocol
is the critical interface.

**Handover Protocol (Intra-Team):**

```
Sender.B                         Receiver.A
   │                                 │
   │  1. validate_before_send()      │
   │  2. ──── deliverable ──────────►│
   │                                 │  3. validate_on_receive()
   │                                 │  4. acknowledge or reject
   │  5. ◄── ack/reject ────────────│
   │                                 │
   │  (if rejected)                  │
   │  6. C analyzes rejection        │
   │  7. adjust & retry              │
   │                                 │
```

### 5.3 Cross-Team Communication (The B↔A Bridge)

Teams interact with each other exclusively through their **boundary agents** —
members whose B-subagent is exposed to the other team's A-subagent. This is
the encapsulation boundary.

```
    Team Alpha                              Team Beta
┌───────────────┐                      ┌───────────────┐
│               │                      │               │
│  Agent 1 (C)  │                      │  Agent 4 (C)  │
│       │       │                      │       │       │
│       ▼       │                      │       ▼       │
│  Agent 2 (B) ─┼───── handover ──────►┼─ Agent 5 (A)  │
│  (boundary)   │   cross-team msg     │  (boundary)   │
│               │◄──── feedback ───────┼─ Agent 5 (B)  │
│  Agent 3 (A) ─┼                      │               │
│               │                      │               │
└───────────────┘                      └───────────────┘
```

**Key properties of cross-team communication:**

1. **Isolation** — Team Alpha's internal discussions, shared memory, and
   decision processes are invisible to Team Beta. Only the boundary agent's
   B-output is visible.
2. **Handover quality** — Cross-team handovers require stricter validation
   than intra-team ones (the research emphasizes this as a major source of
   time waste and friction).
3. **Proxy representation** — The boundary agent acts as a proxy for the
   entire team, just as a Type B agent acts as a proxy for an individual.
4. **Coordinator mediation** — The team coordinator decides which internal
   work is ready for cross-team handover and routes incoming cross-team
   requests to the right internal member.

### 5.4 The Fractal Pattern: Team as Super-Agent

A team itself exhibits the A/B/C pattern at a higher level:

| Team Role         | Maps to  | Function                                    |
|-------------------|----------|---------------------------------------------|
| Boundary intake   | Team-A   | Receives cross-team requests, interprets     |
| Coordinator       | Team-C   | Routes, plans, reflects on team performance  |
| Boundary output   | Team-B   | Delivers cross-team results, represents team |

This fractal structure means the same communication model works at every
scale: individual → team → organization. An organization is simply a
"team of teams" where each team appears as a single agent at the
organizational layer.

```
Organization (Layer 2+)
├── Team Alpha (appears as single agent)
│   ├── Agent 1 (A₁, C₁, B₁)    ← individual Layer 1
│   ├── Agent 2 (A₂, C₂, B₂)
│   └── Agent 3 (A₃, C₃, B₃)
├── Team Beta (appears as single agent)
│   ├── Agent 4 (A₄, C₄, B₄)
│   └── Agent 5 (A₅, C₅, B₅)
└── Team Gamma (appears as single agent)
    ├── Agent 6 (A₆, C₆, B₆)
    └── Agent 7 (A₇, C₇, B₇)
```

---

## 6. Communication Patterns

### 6.1 Human-Like Interaction Catalog

To ensure humans can accept and adopt the system, all agent communication
maps to recognizable human interaction patterns:

| Human Pattern             | Agent Pattern                  | Layer   |
|---------------------------|--------------------------------|---------|
| Asking a colleague        | A→A delegation request         | L1↔L1   |
| Handing over work         | B→A deliverable transfer       | L1↔L1   |
| Team standup              | Coordinator polls all C states | L2      |
| Cross-team sync meeting   | Boundary B↔A exchange          | L2↔L2   |
| Escalation to manager     | C→Coordinator.C escalation     | L1→L2   |
| Personal reflection       | C self-review loop             | L1      |
| Team retrospective        | Coordinator.C reviews traces   | L2      |
| Onboarding new member     | Roster announce + shared mem   | L2      |
| Availability check        | B heartbeat / status broadcast | L1+L2   |

### 6.2 Message Types

Every message in the system carries a `communication_type` that maps to
human intent:

```
request          — "Can you do X?"           (delegation)
deliverable      — "Here is X."             (handover)
acknowledgment   — "Got it, working on it."  (confirmation)
rejection        — "Can't accept X because." (feedback)
status           — "I'm at 60% on X."       (progress)
escalation       — "I need help with X."     (escalation)
reflection       — "Last week, X went well." (retrospective)
announcement     — "I just joined / I'm here." (presence)
```

### 6.3 Handover Quality Gates

Following the research insight that handover friction is the primary source
of wasted time, every cross-boundary message passes through quality gates:

**Sender-side (B validates before sending):**
- Is the deliverable complete per acceptance criteria?
- Are all prerequisites documented?
- Is the recipient identified and available?

**Receiver-side (A validates on receipt):**
- Does the deliverable match expected format/content?
- Are prerequisites met?
- Can I actually process this now?

**On rejection:**
- C analyzes the rejection reason
- C adjusts the deliverable or escalates
- Retry with improved deliverable (not just repeat)

---

## 7. Mapping to GoMikroBot Architecture

### 7.1 Layer 1 → Existing Components

| Subagent | Existing Component                  | Package                |
|----------|--------------------------------------|------------------------|
| A        | Context builder, RAG injection       | `agent/context.go`     |
| A        | Whisper transcription, tool reads    | `provider/`, `tools/`  |
| C        | Agent loop, cognitive modes          | `agent/loop.go`        |
| C        | Day2day tracker, session history     | `agent/`, `session/`   |
| C        | Memory observer, working memory      | `memory/`              |
| B        | Message bus outbound, channels       | `bus/`, `channels/`    |
| B        | Group announcements, identity        | `group/manager.go`     |

### 7.2 Layer 2 → Existing Components

| Team Concept        | Existing Component                  | Package              |
|---------------------|--------------------------------------|----------------------|
| Shared memory       | Shared memory Kafka topic            | `group/manager.go`   |
| Coordinator         | First-join coordinator role          | `group/manager.go`   |
| Roster              | GroupMember + heartbeat              | `group/manager.go`   |
| Task delegation     | SubmitDelegatedTask (depth-limited)  | `group/manager.go`   |
| Cross-team bridge   | Orchestrator parent/child hierarchy  | `orchestrator/`      |
| Team isolation      | Zones (public/shared/private)        | `orchestrator/zone.go`|
| Handover validation | Policy engine                        | `policy/engine.go`   |
| Observability       | Trace sharing, audit events          | `timeline/`          |

### 7.3 What Needs to Be Built

The existing architecture already has strong foundations. The following
extensions would realize the full 2-layer model:

| Gap                              | Description                                          | Priority |
|----------------------------------|------------------------------------------------------|----------|
| **Subagent decomposition**       | Explicit A/B/C subagent roles within the agent loop  | High     |
| **Handover protocol**            | Structured deliverable format with validation gates  | High     |
| **Balance monitor**              | C-subagent detects A/B overload, throttles or queues | Medium   |
| **Boundary agent designation**   | Mark specific agents as cross-team boundary nodes    | Medium   |
| **Team-level A/B/C mapping**     | Coordinator acts as Team-C, routes to Team-A/B       | Medium   |
| **Rejection feedback loop**      | Structured rejection reasons + retry protocol        | Medium   |
| **Communication type headers**   | Message metadata indicating human-intent category    | Low      |
| **Cross-team quality gates**     | Stricter validation for inter-team handovers         | Low      |

---

## 8. Interaction Scenarios

### 8.1 Scenario: Individual Agent Handles a Request

```
Human writes message on WhatsApp
    │
    ▼
[A] WhatsApp channel receives → bus delivers inbound message
    │
    ▼
[A] Context builder loads soul files, injects RAG context
    │
    ▼
[C] Agent loop: LLM reasoning, cognitive mode selection
    │  ├── needs more info? → [A] tool call (read_file, recall, web)
    │  ├── needs reflection? → [C] check session history, memory
    │  └── ready to respond? → continue
    │
    ▼
[B] Format response, policy check, publish to bus → WhatsApp channel sends
```

### 8.2 Scenario: Intra-Team Task Delegation

```
Agent 1 (Coordinator) receives a complex task
    │
    ▼
[C₁] Breaks task into subtasks, checks roster capabilities
    │
    ▼
[B₁] Sends delegation request to Agent 2 (type: "request")
    │
    ▼
[A₂] Receives, validates prerequisites
    │
    ▼
[C₂] Plans execution, works on subtask
    │
    ▼
[B₂] Delivers result back to Agent 1 (type: "deliverable")
    │
    ▼
[A₁] Receives, validates quality
    │
    ▼
[C₁] Integrates subtask result into overall response
    │
    ▼
[B₁] Delivers final result to requester
```

### 8.3 Scenario: Cross-Team Handover

```
Team Alpha's Coordinator identifies need for Team Beta's expertise
    │
    ▼
[C_coord_α] Prepares handover package, selects boundary agent
    │
    ▼
[B_boundary_α] Validates deliverable → sends cross-team request
    │                                    to group.beta.requests
    │
    ▼
[A_boundary_β] Receives, validates format and prerequisites
    │              │
    │              ├── reject? → feedback to B_boundary_α
    │              │              → C_coord_α adjusts and retries
    │              │
    │              └── accept? → continue
    │
    ▼
[C_coord_β] Routes to appropriate internal worker
    │
    ▼
[internal work within Team Beta — invisible to Alpha]
    │
    ▼
[B_boundary_β] Delivers result back (type: "deliverable")
    │
    ▼
[A_boundary_α] Receives, Coordinator integrates result
```

---

## 9. Future: Optimizing Communication Structures

Starting with human-like interaction patterns makes adoption easy. Over time,
the system can learn and evolve:

### 9.1 Observable Metrics (from Agent Traces)

- **Handover rejection rate** — Which boundaries have the most friction?
- **Delegation depth** — Are tasks being bounced too many times?
- **Response latency per communication type** — Where are the bottlenecks?
- **A/B/C balance ratio** — Is any subagent consistently overloaded?
- **Cross-team vs intra-team message ratio** — Is team composition optimal?

### 9.2 Learning from Experience

Once agents have accumulated trace data, we can:

1. **Detect suboptimal team composition** — If Agent 3 always delegates to
   Team Beta, maybe Agent 3 belongs in Team Beta
2. **Identify missing handover criteria** — High rejection rates signal unclear
   acceptance criteria
3. **Discover implicit communication paths** — Actual message flow may differ
   from the org chart, revealing the "real" team structure
4. **Suggest boundary agent rotation** — Preventing bottleneck at a single
   cross-team bridge
5. **Evolve from human-like to optimized** — Gradually introduce batch
   processing, parallel delegation, or predictive routing where humans
   would not naturally do so — but only after the human-like baseline is
   established and trusted

### 9.3 Principle: Human-First, Optimize Later

```
Phase 1: Mirror human communication patterns exactly
         (people understand and trust the system)
              │
              ▼
Phase 2: Surface metrics and insights from traces
         (people see where friction exists)
              │
              ▼
Phase 3: Suggest optimizations based on data
         (the system proposes, humans approve)
              │
              ▼
Phase 4: Autonomous optimization within guardrails
         (agents restructure communication, humans audit)
```

---

## 10. Summary

| Concept               | Layer 1 (Individual)           | Layer 2 (Team)                    |
|------------------------|--------------------------------|-----------------------------------|
| **Unit**               | Single agent (human or AI)     | Group of agents                   |
| **Internal structure** | A + C + B subagents            | Members + Coordinator + Memory    |
| **Inbound**            | A (World Interpreter)          | Boundary agent A-subagent         |
| **Processing**         | C (Cognitive Enhancer)         | Coordinator + internal routing    |
| **Outbound**           | B (World Exposer)              | Boundary agent B-subagent         |
| **Communication**      | Human-like conversation        | Handover protocol with validation |
| **Isolation**          | A/B/C internals hidden         | Team internals hidden from others |
| **Balance**            | C monitors A/B load            | Coordinator monitors member load  |
| **Scaling**            | —                              | Team acts as single agent at L2+  |

The 2-layer hierarchy provides:

1. **Individuals as agents** — Each person/agent has a balanced A/B/C triad
2. **Groups as teams** — Agents share memory, coordinate through a leader
3. **Organizations as teams-of-teams** — The fractal A/B/C pattern repeats
4. **Human-like interaction** — Every agent message maps to a recognizable
   human communication pattern
5. **Future optimization** — Trace data enables learning and restructuring
   communication paths based on real experience
