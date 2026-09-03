# ADR 0001: Prioritize the Control Plane over Portal Breadth

- Status: Accepted
- Date: 2026-09-04

## Context

A polished portal can demonstrate a happy path while hiding retries, partial failures, and drift. The portfolio signal depends on reliable orchestration rather than UI surface area.

## Decision

Build a small portal and CLI over an operation-oriented API. Invest first in durable desired state, idempotency, checkpoints, audit events, and actionable status.

## Consequences

The demo exposes real failure states and is testable end to end. Catalogue search, scorecards, and template upgrade UX remain future work.
