# ADR 0003: Deliver Workloads through GitOps

- Status: Accepted
- Date: 2026-09-04

## Context

Direct cluster mutation from the platform API creates an unreviewed path and weakens recovery and auditability.

## Decision

The platform generates or updates environment manifests. Argo CD is the only production mechanism that converges application workloads. The API never shells out to `kubectl`.

## Consequences

Git history becomes deployment evidence and rollback is explicit. The operation must wait for Argo sync, health, smoke tests, and telemetry registration before success.
