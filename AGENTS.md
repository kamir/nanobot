
# Agents in this Project – Python to Go Rewrite

## 1. Purpose of the Agent System

This project aims to rewrite an existing Python codebase into Go while:

- preserving functional behavior
- improving performance and reliability
- documenting architecture and decisions
- tracking risks and migration issues

The agents collaborate to transform raw ideas and problems into
a structured Go implementation.

---

## 2. Source of Truth

All reasoning originates from:

    PH/  – Prompt History (immutable)

Derived artifacts are created in:

- /requirements  – WHAT must be preserved or changed (new FR-xyz files, running number)
- /arch          – HOW the Go system will be structured
- /DEVTASKS      – Plans and solution designs
- /bug-tracker   – Investigations and migration problems
- /docs          – Feature docs for the bot
- /morphing/docs - bot-structure-and-dynamics.md

Direction of truth is one-way:

    PH → Requirements → Architecture → Tasks  
                  ↘ Problem Investigations ↗

No agent may modify PH files.

Documentation rule:
- Any change to CLI commands, flags, or arguments must be documented in `/docs` immediately.

Release hygiene checklist:
- Docs updated for any CLI changes.
- Specs updated for any behavior changes.
- Tests added/updated for new or changed flows.
- `docs/USER_MANUAL.md` reviewed for user-facing changes.
- `docs/OPERATIONS_GUIDE.md` reviewed for API/port/DB changes.
- `docs/ADMIN_GUIDE.md` reviewed for config/security/policy changes.

---

## 3. Agent Roles

### Agent 1 – Prompt History Tracker

**Role:** Memory & audit

- Records all prompts verbatim into PH/
- Hourly rotation
- Append-only

**Why for migration:**  
Preserves original intent of the Python system and all design discussions.

---

### Agent 2 – Requirements Tracker

**Role:** Product & behavior guardian

- Extracts functional behavior of the Python system
- Identifies constraints that must exist in Go
- Separates:
  - behavior to keep
  - behavior to improve
  - behavior to remove

**Key questions**
- What must the Go version do exactly like Python?
- Which side effects are accidental?
- Which contracts are implicit?

**Output**
- /requirements/FR-xyz-*.md

---

### Agent 3 – Architecture & Implementation Agent

**Role:** Systems architect

- Designs Go structure based on requirements
- Maps Python concepts to Go equivalents:

| Python Concept | Go Target |
|----------------|-----------|
| dynamic types  | static types |
| classes        | structs + interfaces |
| exceptions     | errors |
| threads/async  | goroutines |
| monkey patch   | DI patterns |

**Creates**

- /arch/component-map.md  
- /arch/python-to-go-mapping.md  
- /arch/data-models.md  
- /arch/interfaces.md

**Responsibilities**

- Define module boundaries  
- Define concurrency model  
- Define error strategy  
- Define test approach

---

### Agent 4 – Task Planner

**Role:** Migration executor

- Breaks rewrite into safe increments:

1. Identify Python module  
2. Define Go interface  
3. Create tests  
4. Implement Go version  
5. Parity validation

**Rules**

- No task > 2 days  
- Each task has:
  - acceptance criteria
  - parity test
  - rollback path

**Output**
- /DEVTASKS/*.md

---

### Agent 5 – Problem Analysis & Insight Tracker (JSON)

**Role:** Investigator during migration

Used when:

- Go behavior differs from Python  
- performance regressions  
- semantic mismatches

**Produces**

    ./bug-tracker/BUG-<id>.json
    /morphing/docs/bot-structure-and-dynamics.md - Update the   chart of structure and the flow

including:

- hypotheses  
- comparison Python vs Go  
- experiments  
- test proposals

**Central question**

> Why does Go behave differently from Python?

---

## 4. Migration Workflow

### Step A – Capture Truth
Prompt Tracker records all context about the Python system.

### Step B – Extract Requirements
Requirements Agent defines:

- external contracts  
- data formats  
- timing expectations  
- error behavior

### Step C – Design Go Architecture
Architecture Agent:

- creates type system  
- defines packages  
- selects libraries  
- defines concurrency

### Step D – Plan Rewrite
Task Planner creates incremental path:

Python module → Go interface → Tests → Implementation

### Step E – Investigate Divergence
Problem Agent compares:

- Python output  
- Go output  
- edge cases

---

## 5. Quality Gates

The Go rewrite is accepted when:

1. All requirements have Go equivalents  
2. Parity tests pass  
3. Performance ≥ Python baseline  
4. Error semantics documented  
5. No open critical bugs

---

## 6. Folder Contracts

### /requirements
- behavior contracts
- invariants
- compatibility rules

### /arch
- python-to-go mapping
- package design
- concurrency model

### /tasks
- migration steps
- test plans

### /bug-tracker
- investigations
- experiments
- comparisons

### /docs
- onboarding for Go devs
- rationale
- `USER_MANUAL.md` — End-user guide (CLI, WhatsApp, Web UI, memory, Day2Day)
- `OPERATIONS_GUIDE.md` — DevOps guide (build, deploy, DB, API, observability)
- `ADMIN_GUIDE.md` — Admin guide (config, security, providers, quotas, extending)

---

## 7. Governance Rules

- No agent edits another agent’s files  
- PH is immutable  
- Every decision links to PH  
- Behavior parity first, improvements second

---

## 8. Schedule

- Tracker: hourly  
- Requirements: on new PH  
- Architecture: on new requirements  
- Tasks: after architecture  
- Problem: on demand

---

## 9. Success Definition

The agents succeed when:

- The Go system can replace Python in production  
- All behaviors are explained  
- Differences are intentional  
- Knowledge is documented

---

## 10. Version

AGENTS spec: 2.0 – Go Migration Edition


---

## 11. Documentation Synchronization Protocol

The three published guides in `/docs` must stay in sync with the codebase.
Every code change that affects user-visible behavior triggers a doc update.

### Trigger Matrix

| Change Type | USER_MANUAL | OPERATIONS_GUIDE | ADMIN_GUIDE |
|---|---|---|---|
| New/changed CLI command or flag | Yes | — | — |
| New/changed API endpoint | — | Yes | — |
| New/changed config key or env var | — | — | Yes |
| New/changed tool | Yes (usage) | — | Yes (extending) |
| Database schema change | — | Yes | — |
| Security/policy change | — | — | Yes |
| Build/release/CI change | — | Yes | — |
| WhatsApp flow change | Yes | — | Yes |
| Memory/RAG change | Yes (usage) | — | Yes (admin) |
| Port/network change | — | Yes | — |
| Day2Day command change | Yes | — | — |
| Soul file/workspace change | Yes | — | Yes |
| Web dashboard feature | Yes | Yes (API) | — |
| Token quota change | — | — | Yes |

### Rules

1. **Same-PR rule** — Doc updates ship in the same PR as the code change.
   No code PR is complete if it leaves a guide stale.
2. **Cross-check on release** — Before tagging a release, verify all three
   guides against the current source. Use the release hygiene checklist
   in Section 2.
3. **Single source of truth** — The Go source code is authoritative.
   If a guide contradicts the code, the guide is wrong. Fix the guide.
4. **No speculative docs** — Only document what is implemented and merged.
   Do not document planned features.
5. **Audience separation** — Keep user-facing details in USER_MANUAL,
   operator details in OPERATIONS_GUIDE, and admin/security details
   in ADMIN_GUIDE. Avoid duplicating content across guides; cross-reference
   instead.

---

How This Helps You Day-to-Day

This file gives you:

1) A migration compass
	•	No random rewriting
	•	Behavior-first
	•	Parity before optimization

2) Clear thinking tools
	•	Python → Go concept map
	•	structured investigations
	•	experiment logs

3) Safety net
	•	every divergence documented
	•	reversible decisions
	•	knowledge preserved

⸻
