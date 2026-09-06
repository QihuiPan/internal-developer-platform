# Changelog

All notable changes to this project are documented in this file. Every pull request must add an English entry under `Unreleased`.

The format is based on Keep a Changelog, and this project follows Semantic Versioning.

## [Unreleased]

## [0.2.0] - 2026-09-07

### Added

- Added an embedded service catalogue portal so one API binary provides the complete local experience.
- Added service and operation collection endpoints with newest-first ordering.
- Added authenticated ZIP downloads for ready generated repositories through the portal, API, and CLI.
- Added a multi-command `platformctl` client for create, list, get, operation, retry, and audit workflows.
- Added opt-in Bearer token authentication with a server-assigned RBAC role for small shared deployments.
- Added cross-platform release packaging for Windows, macOS, and Linux on AMD64 and ARM64 with SHA-256 checksums, plus a multi-architecture GHCR image.
- Added binary smoke testing, container-build validation, Helm values validation, setup documentation, configuration guidance, troubleshooting, and a security policy.
- Added the initial Internal Developer Platform portfolio MVP.
- Added a Go control-plane API with descriptor validation, RBAC, idempotency request fingerprints, durable operations, audit events, and retry checkpoints.
- Added a reconciler that generates a secure service repository, resource plan, CI workflow, and GitOps manifest.
- Added a self-service portal, CLI, OpenAPI contract, PostgreSQL target schema, Terraform module, Helm chart, and Kyverno policy.
- Added unit, integration, end-to-end, resilience, and benchmark coverage.
- Added architecture records, threat model, SLOs, runbook, demo guide, and contribution controls.

### Changed

- Changed the default API listener to `127.0.0.1:8080`; container deployments explicitly bind to all interfaces.
- Simplified Docker Compose to one hardened container serving the API, worker, and embedded portal on port 8080.
- Ensured the non-root container owns its persistent data mount point on first volume initialization.
- Expanded generated Kubernetes assets with a Service, scoped NetworkPolicy, pod security context, named port, and clearer image replacement instructions.
- Changed generated repository links to absolute file URLs and removed the non-functional dashboard placeholder.
- Updated the API contract and Helm chart to version 0.2.0.
- Changed repository visibility from private to public on 2026-09-04 with the repository owner's explicit approval.

### Fixed

- Applied canonical Terraform formatting so the infrastructure CI gate passes.
- Updated the Terraform setup action to its Node 24-compatible major release.
- Updated the Helm setup action to its Node 24-compatible major release.

### Security

- Added browser security headers and prevented request headers from elevating the server-configured role in token mode.
- Added durable audit events for failed reconciliation operations.
- Added explicit local-only warnings for demo authentication and protected Helm Secret integration for token mode.
- Restricted default Compose access to loopback and Helm ingress to pods in the release namespace.

[Unreleased]: https://github.com/QihuiPan/internal-developer-platform/commits/main
[0.2.0]: https://github.com/QihuiPan/internal-developer-platform/releases/tag/v0.2.0
