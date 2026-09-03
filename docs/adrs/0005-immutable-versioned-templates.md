# ADR 0005: Render from Immutable Versioned Templates

- Status: Accepted
- Date: 2026-09-04

## Context

Mutable templates make generated output irreproducible and can break existing services without review.

## Decision

Descriptors reference semantic template versions. Production rendering resolves that version to an immutable Git commit and records it with the service. Upgrades arrive as tested pull requests rather than implicit regeneration.

## Consequences

Every service can be reproduced and template regressions can be canaried. Template maintainers must publish versions and compatibility evidence.
