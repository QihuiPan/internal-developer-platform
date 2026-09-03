# Service Level Objectives

## Platform API availability

- SLI: successful non-5xx HTTP requests divided by valid HTTP requests at the load balancer.
- Objective: 99.9% over a rolling 30-day window.
- Exclusions: planned maintenance announced at least 48 hours in advance.
- Alert: page when the one-hour burn rate exceeds 14.4 or the six-hour burn rate exceeds 6.

## Service-creation success

- SLI: operations reaching `SUCCEEDED` divided by accepted create operations, excluding descriptor-validation failures.
- Objective: at least 99% over 30 days.
- Alert: ticket on a 24-hour rate below 99%; page when a one-hour window falls below 95% with at least 20 operations.

## Time to first deployment

- SLI: duration from accepted create request to verified development readiness.
- Objective: p50 below five minutes and p95 below ten minutes.
- Metric labels: template major version and environment only; service names and operation IDs belong in traces and logs.

## Queue and drift

- Provisioning queue wait: p95 below 60 seconds.
- Drift detection: p95 below five minutes.
- Recovery point: no accepted operation may be lost after an API process crash.

## Measurement gaps

The MVP exposes process health and request counters. Histograms, operation result counters, distributed traces, and a scheduled drift loop are required before claiming these production objectives are met.
