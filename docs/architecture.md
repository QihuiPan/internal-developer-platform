# Architecture

## Goal

The platform accepts small, versioned declarations of service intent and converges external systems toward that intent. The API records desired state and an operation before any side effect. Workers then execute retry-safe steps and expose progress to users.

## Control-plane boundaries

| Boundary | Owns | Does not own |
| --- | --- | --- |
| Portal / CLI | Input collection, progress display | Cloud credentials or orchestration state |
| Platform API | Authentication adapter, RBAC, validation, catalogue, operations, audit | Direct Kubernetes mutation |
| Store | Desired state, idempotency records, checkpoints, audit events | External-system state |
| Reconciler | Step ordering, retries, verification | User-facing request lifecycle |
| Repository adapter | Template rendering and repository settings | Infrastructure apply |
| Terraform adapter | Plan and apply lifecycle | Kubernetes convergence |
| GitOps adapter | Environment repository commits | Direct `kubectl` deployment |

## Consistency model

The create request writes the service record, operation, idempotency binding, and audit event under one store lock and one durable file replacement. No external side effect begins before that commit. In the PostgreSQL target, the same transaction also writes an outbox record. Workers checkpoint every step; after a crash, startup requeues pending or retrying operations and completed steps are skipped.

The current worker is single-process. The PostgreSQL target adds lease ownership and expiry so multiple workers can claim operations safely. Adapters must implement read-before-write reconciliation because remote calls can succeed even when the response is lost.

## Generated golden path

Each published HTTP renderer creates:

- a minimal service with health, readiness, and metrics endpoints;
- a two-stage non-root container image;
- a Kubernetes Deployment with probes, requests, limits, dropped capabilities, and a read-only filesystem;
- a default-deny NetworkPolicy;
- a CI workflow and CODEOWNERS file;
- the original service descriptor for provenance.

The renderer is deterministic for the same descriptor. Production repository creation should use a GitHub App installation token, record the immutable template commit, set branch protection, and return the remote URL.

## Production evolution

1. Replace the header identity adapter with JWT verification and team claims from OIDC.
2. Implement the PostgreSQL store and transactional outbox migration already modeled in `migrations/001_initial.sql`.
3. Move the reconciler to a separately scaled worker deployment with expiring leases.
4. Add GitHub App, Terraform runner, and Argo CD adapters with operation-scoped identities.
5. Propagate W3C trace context across the outbox and adapters.
6. Add External Secrets references and managed PostgreSQL/Redis provider modules.
