# ADR 0004: Use OIDC and Operation-Scoped Identity

- Status: Accepted; production implementation pending
- Date: 2026-09-04

## Context

Long-lived cloud keys and portal-held administrator credentials create broad compromise paths.

## Decision

Use verified user OIDC claims at the API and short-lived workload identity for CI and workers. Each Terraform environment receives a separate role; production applies require protected-environment approval.

## Consequences

Credentials are attributable and narrowly scoped. Local development uses a clearly labeled header adapter and cannot be exposed to untrusted clients.
