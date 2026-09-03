# Threat Model

## Assets and trust boundaries

Protected assets include service desired state, repository administration, Terraform state and plans, deployment approvals, secrets references, audit evidence, and workload admission policy. Trust boundaries exist between the user and API, API and store, worker and external providers, GitOps repository and Argo CD, and Kubernetes admission and runtime.

## Threats and controls

| Threat | Example | Current control | Required production control |
| --- | --- | --- | --- |
| Spoofing | Caller supplies another team's role | Demo adapter requires explicit identity and fails closed | Verify signed OIDC JWTs and derive roles from trusted claims |
| Tampering | Request changes under a reused idempotency key | SHA-256 request fingerprint returns a conflict | Sign outbox events and protect database writes with least privilege |
| Repudiation | Administrator retries without explanation | Append-only audit event and required retry reason | Export audit events to immutable retention storage |
| Information disclosure | Secret value appears in Git or portal | No secret-value fields; generated assets use references only | External Secrets and automated secret scanning |
| Denial of service | Oversized JSON or operation flood | 1 MiB request cap, bounded queue, HTTP timeouts | Rate limits, quotas, backpressure, and workload isolation |
| Elevation of privilege | Privileged or hostPath workload | Kyverno deny policy, non-root images, dropped capabilities | Signed images, verified provenance, and protected policy changes |
| Supply-chain compromise | Mutable image or workflow action changes | Generated manifest requires a digest placeholder | Enforce digests, pin actions by commit, generate SBOMs, verify signatures |
| Cross-tenant access | Team reads another team's service or secret | Role-level RBAC only | Resource-level authorization using ownership claims and policy tests |

## Abuse cases

1. Replaying a create request must return the original operation without creating another repository.
2. Reusing an idempotency key for different intent must fail with a conflict.
3. A developer must not read audit events or bypass production policy.
4. A worker crash after an external success must reconcile before issuing another mutation.
5. A malicious template must not obtain platform administrator credentials.

## Accepted MVP risks

Header-based identity, file-backed state, and filesystem repository generation are local-demo mechanisms. They are not acceptable on an untrusted network. The quick start states this explicitly, and the Helm chart should be deployed only in an isolated development cluster until OIDC and the PostgreSQL adapter are complete.
