# ScalBotics Architecture (Draft v0.1)

Status: Draft  
Last Updated: 2026-02-09  
Owner: ScalBotics Core Team

## 1. Purpose

This document defines the initial architecture for ScalBotics so we can iterate quickly while keeping security, enterprise readiness, and performance as first-class constraints.

This is a living draft. Each section includes open decisions so we can evolve it without losing direction.

## 2. Product Direction

ScalBotics is a lightweight, fast, enterprise-oriented AI agent framework with:

- Task-oriented single-agent and multi-agent execution.
- Go-based runtime.
- gRPC for control plane and inter-agent orchestration.
- Kafka topic integration for cross-team asynchronous workflows.
- Zero-trust security model for humans and agents.
- OpenAI-compatible model provider abstraction.

## 3. Design Principles

- Secure by default: deny-by-default authz, short-lived credentials, full audit trails.
- Lightweight core: minimal dependencies, stdlib-first where practical.
- Fast path first: local tool execution and deterministic workflows before expensive model calls.
- Declarative operations: Git-backed desired state for teams/agents/policies.
- Clear boundaries: core runtime independent from optional infra adapters.

## 4. Goals

- Provide production-ready enterprise primitives from day one.
- Human authentication and authorization.
- Agent/workload identity and mTLS.
- Policy-based tool execution control.
- Tenant/team isolation.
- Auditable operations and data access.
- Support local VM install and enterprise deployment without architectural rewrite.
- Simple onboarding UX via CLI (`scalbotics initialize`, `scalbotics agent add`).

## 5. Non-Goals (v1)

- Building a generic chat app first.
- Heavy framework coupling that blocks single-VM operation.
- Storing secrets/private keys in Git.
- Hard dependency on Kubernetes for first usable release.

## 6. High-Level Architecture

### 6.1 Core Components

- `scalboticsd` (single Go binary in v1)
- gRPC API (control plane + orchestration)
- Scheduler and task runtime
- Tool execution manager
- Policy engine integration point
- Audit/event subsystem
- Model router (OpenAI-compatible providers + fallback chain)
- Memory service (structured JSON memory + vector index adapter)

### 6.2 Optional Adapters

- Git providers (local git, GitHub, GitLab)
- Kafka transport (cross-team event bus)
- Secret backends (Vault/KMS/cloud secret managers)
- External policy engines (OPA/Cedar)
- External identity providers (OIDC)

### 6.3 Communication Model

- Synchronous control and orchestration: gRPC (unary + streaming).
- Asynchronous inter-team events: Kafka topics (optional in local mode).
- Internal deterministic queue in local mode when Kafka is not configured.

## 7. Agent Runtime Model

### 7.1 Execution Modes

- Single agent mode: direct task execution, no team coordinator.
- Team mode: lead agent coordinates specialized workers.

### 7.2 Coordination Rules

- If one agent exists in scope, execute directly.
- If multiple agents exist, lead agent handles planning/delegation and worker aggregation.
- Idempotent task envelopes and replay-safe execution are mandatory.

### 7.3 Bootstrap Context

Each agent workspace includes:

- `SOUL.md`
- `AGENT.md`
- `TOOLS.md`
- `MEMORY/` (structured JSON memory artifacts)

These are read by default and versioned with workspace state.

## 8. Memory Architecture

### 8.1 Memory Format

- Memory source-of-truth is mixed by actor:
- Humans write Markdown (`*.md`) for maintainability and Git reviewability.
- Agents/services write JSON records for deterministic processing and auditability.
- All indexed memory is normalized into a versioned internal JSON chunk schema.
- Schema version in every stored record/chunk.
- Source metadata and classification tags are required.

### 8.2 Memory Pipeline

- Write path:
- Ingest Markdown or JSON input.
- Normalize to internal chunk schema.
- Validate schema and policy constraints.
- Store canonical record.
- Embed and index for retrieval.
- Read path: policy-filtered retrieval -> hybrid ranking (vector + metadata/keyword where configured).

### 8.3 Memory Backend Strategy (v1 concrete)

- Default backend (all profiles): SQLite + `sqlite-vec`.
- Rationale:
- Lightweight and fast for local and single-VM enterprise installs.
- Minimal ops complexity and dependency surface.
- Good fit for single-binary-first architecture.

- Backend abstraction:
- `MemoryStore` and `VectorIndex` interfaces are mandatory.
- Backends must be swappable without changing agent/runtime APIs.

- Planned adapters:
- `pgvector` for PostgreSQL-centric enterprise environments.
- `Qdrant` for dedicated vector-service deployments.

### 8.4 Embedding Model Profile (v1 concrete)

- Default local embedding model: `sentence-transformers/all-MiniLM-L6-v2` (384 dimensions).
- Rationale:
- Very small and CPU-friendly.
- Fast indexing/search latency on VM-class hardware.
- Mature ecosystem support (PyTorch/ONNX).

- Optional local alternatives:
- `BAAI/bge-small-en-v1.5` for English-heavy corpora.
- `intfloat/multilingual-e5-small` for multilingual corpora.

- Runtime guidance:
- Prefer ONNX Runtime with int8 quantization for CPU deployments.
- Batch embeddings conservatively (default batch size 32, configurable).
- Chunk size target: 400-800 tokens with 10-15% overlap.
- Enforce per-index memory caps and backpressure during bulk ingestion.

- Provider abstraction:
- Local embedding model is default.
- Remote embedding providers remain optional and policy-gated.

### 8.5 Guardrails

- PII/secret redaction pre-write.
- Tenant/team namespace isolation.
- Retention and legal hold support hooks.

## 9. Security and Zero Trust

### 9.1 Identity

- Humans: OIDC-based authentication with short-lived tokens.
- Agents/services: workload identity with mTLS certificates.
- No shared static service keys between components.

### 9.2 Authorization

- RBAC + ABAC policy model.
- Delegation rule: agent privileges cannot exceed delegated human scope.
- Tool risk tiers:
- `Tier 0`: read-only internal tools.
- `Tier 1`: controlled write/internal effects.
- `Tier 2`: external or high-impact actions (approval gate).

### 9.3 Key and Secret Management

- Central key authority abstraction (backed by Vault/KMS/internal CA).
- Git stores references and policy, never private keys.
- Key rotation and revocation workflows are first-class.

### 9.4 Audit

- Immutable append-only audit events.
- Every decision logs actor, delegated principal, action, target, policy result, and trace ID.

### 9.5 Auth and Identity (v1 concrete)

- Humans:
- OIDC login against enterprise IdP (e.g., Entra/Okta/GitLab/GitHub Enterprise OIDC).
- Access token is JWT, short-lived (default 10 minutes).
- Refresh token never sent to `scalboticsd`; refresh handled by CLI/web login broker.
- Required claims: `sub`, `iss`, `aud`, `exp`, `iat`, `tenant`, `groups` (or mapped roles).

- Agents/services:
- mTLS for service-to-service authentication.
- Each agent/service gets its own identity (`spiffe://scalbotics/<tenant>/<service>`-style URI).
- Certificates are short-lived (default 24h, configurable).
- No shared service credentials across agents.

- Authorization enforcement:
- Every gRPC call must have authenticated principal context.
- Policy check happens before handler execution.
- Delegation chain is explicit: `human -> lead agent -> worker agent`.

## 10. GitOps and State Model

Git is used as declarative desired state (“team operating system”).

Example structure:

- `teams/<team-id>/team.yaml`
- `agents/<agent-id>/agent.yaml`
- `workspaces/<agent-id>/{SOUL.md,AGENT.md,TOOLS.md}`
- `policies/*.rego` (or equivalent)
- `env/*.yaml` for deployment overlays

Reconciler behavior:

- Desired state from Git -> validated -> applied to runtime.
- Drift detection with explicit reconciliation events.

### 10.1 Identity and Secret Store Model

- Git stores configuration and secret references only.
- Identity and secrets live in a secure backend:
- Local mode: encrypted local keystore + internal CA.
- Enterprise mode: Vault/KMS/HSM-backed key authority.

Stored objects:

- Human principals (metadata, role bindings, tenant mappings).
- Agent/workload identities.
- Certificate authority metadata and issuance policies.
- Secret references and rotation metadata.

Explicitly not stored in Git:

- Private keys.
- OIDC client secrets.
- Provider API secrets.

## 11. CLI UX

### 11.1 Initialization

`scalbotics initialize` (interactive)

- Prompt: local git or remote git.
- Local git path bootstrapping.
- Remote git provider selection and auth token flow.
- Bootstrap first admin identity.
- Bootstrap first agent and workspace.
- Bootstrap key authority configuration.

### 11.2 Lifecycle Commands

- `scalbotics agent add`
- `scalbotics agent rotate-keys`
- `scalbotics agent suspend`
- `scalbotics agent offboard`
- `scalbotics team add`
- `scalbotics policy validate`

### 11.3 Credential Distribution

- `scalbotics initialize`:
- Creates root tenant context.
- Configures OIDC trust (issuer, audience, JWKS).
- Initializes local CA or connects to external key authority.

- `scalbotics agent add`:
- Registers agent identity.
- Issues bootstrap credential with one-time token (TTL default 10 minutes).
- Generates workspace and policy bindings.
- Agent exchanges bootstrap token for short-lived cert/token on first start.

- Rotation and revocation:
- `scalbotics agent rotate-keys` forces cert/key rollover.
- `scalbotics agent suspend` immediately blocks token/cert issuance and policy access.
- Revocation events are published to runtime cache invalidators.

## 12. Deployment Profiles

### 12.1 Local/VM Profile (v1 default)

- Single host deployment.
- `scalboticsd` + local storage.
- Optional local git repo.
- Optional Kafka disabled (internal queue active).

### 12.2 Enterprise Profile

- External OIDC provider.
- External key authority (Vault/KMS).
- Kafka enabled for cross-team workflows.
- Centralized observability and SIEM export.

### 12.3 Kubernetes Profile (post-v1)

- Agent runtime as pods/workloads.
- Persistent workspaces via volumes.
- Init/sync process for skills/addons artifacts.
- Same control plane contracts as VM profile.

## 13. Dependency Budget

Core runtime should stay small and auditable.

- Required dependencies:
- gRPC/protobuf
- SQL driver
- OIDC/JWT validation
- Structured logging/tracing
- Optional dependencies:
- Kafka client
- Vault/KMS SDKs
- OPA/Cedar adapters

Dependency acceptance criteria:

- Active maintenance.
- License compatibility.
- Security patch cadence.
- Clear operational value for enterprise use.

### 13.1 Observability Stack (v1 concrete)

- Metrics:
- Prometheus scrape endpoint (`/metrics`) from `scalboticsd`.
- OpenMetrics format.
- Core metrics:
- auth success/failure counts (human/agent).
- cert issuance/rotation/revocation counters.
- gRPC request count/latency/error by method and tenant.
- policy decision allow/deny counters.
- model call token usage, latency, fallback counts.
- tool execution latency and failure rates.

- Tracing:
- OpenTelemetry SDK with OTLP export.
- Trace context propagated through gRPC metadata and Kafka headers.
- Required span attributes: `tenant_id`, `team_id`, `agent_id`, `principal_type`, `policy_decision`.

- Logging:
- Structured JSON logs only.
- Correlate with `trace_id`, `span_id`, `request_id`, and delegation chain ids.
- Redaction pipeline for secrets and sensitive payload fields.

- Dashboards and alerting:
- Prometheus + Alertmanager + Grafana as reference stack.
- SLO alerts:
- auth failure spikes.
- cert issuance failures.
- policy deny anomaly spikes.
- p95/p99 gRPC latency breaches.
- model fallback-rate anomalies.

## 14. Open Questions

- v1 storage choice: PostgreSQL only, or PostgreSQL + SQLite local mode?
- Built-in policy engine scope vs immediate OPA integration?
- Built-in CA vs SPIRE integration phase?
- Kafka in v1 or v1.1?
- Priority and rollout timing for `pgvector` and `Qdrant` adapters after SQLite v1.
- Approval workflow UX for Tier 2 actions (CLI only vs CLI + web UI)?

## 15. Iteration Plan

### Milestone A: Foundation

- Finalize core contracts (gRPC + task envelope schema).
- Implement authn/authz skeleton.
- Implement local mode runtime and audit trail.

### Milestone B: Agent Lifecycle

- Implement workspace bootstrap and agent onboarding lifecycle.
- Implement key issuance/rotation/revocation flows.
- Implement policy-gated tool execution tiers.

### Milestone C: Enterprise Extensions

- Add Kafka adapter and topic taxonomy.
- Add external key provider and OIDC production integrations.
- Add Git reconciler and drift handling.

### Milestone D: Hardening

- Threat model validation and security tests.
- Performance benchmarks and token-cost controls.
- Operational runbooks and disaster recovery flows.

## 16. Change Log

- v0.1: Initial architecture draft from discovery and brainstorming sessions.
