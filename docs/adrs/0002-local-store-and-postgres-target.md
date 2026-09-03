# ADR 0002: Use a Local Durable Store with a PostgreSQL Target Schema

- Status: Accepted for MVP
- Date: 2026-09-04

## Context

The repository must run without external services while preserving state across process restarts. Production needs transactions, leases, an outbox, and horizontal scale.

## Decision

Use an atomically replaced JSON state file for the single-process demonstrator. Define the production PostgreSQL model in a migration and isolate persistence in `internal/store`.

## Consequences

The quick start is deterministic and dependency-free. Only one API replica may write the local store. Production readiness requires the PostgreSQL adapter and lease-based worker claiming.
