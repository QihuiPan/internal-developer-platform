# Runbook: Failed Service Operation

## Trigger

Use this runbook when an operation remains in `FAILED`, queue latency breaches its SLO, or a user reports that generated state and external state disagree.

## Triage

1. Record the operation ID, service name, actor, current status, failed step, attempt count, request ID, and first failure time.
2. Query `GET /v1/operations/{id}` as a platform administrator. Do not ask the user to inspect worker logs.
3. Confirm whether the error is validation, local storage, planning, rendering, verification, or a future remote-adapter failure.
4. Check API health, disk availability, state-file permissions, queue depth, and recent structured error logs.
5. Compare the recorded descriptor and plan artefact with the generated repository. Never infer success only from the presence of a directory.

## Recovery

- Validation failure: correct the descriptor and submit a new idempotency key.
- Transient worker failure: remove the cause and call `POST /v1/operations/{id}/retry` with a precise reason.
- Crash after a remote mutation: query the remote provider by operation tag or desired-state hash before retrying. Adopt the existing resource when it matches; otherwise stop for manual review.
- Corrupt local state: stop the API, preserve the state file and logs, restore the last verified backup, and reconcile every non-terminal operation.
- GitOps verification failure: revert or repair the environment commit, wait for Argo health, and retry only the verification step.

## Verification

Recovery is complete only when the operation is `SUCCEEDED`, every step is `SUCCEEDED`, the service record is `READY`, generated files pass their tests, and declared state matches observed state. Attach request IDs, before/after state, and the retry audit event to the incident.

## Escalation

Escalate immediately for suspected credential exposure, audit-log tampering, cross-team data access, policy bypass, or an unbounded duplicate-resource event. Freeze automated retries until containment is complete.

## Rollback

Stop the worker, revert the GitOps commit, restore the previous immutable image digest, and leave the failed operation record intact. Resource deletion requires a separately authorized operation because deletion is recoverable and auditable.
