# Test Evidence

## Local verification

Recorded on 2026-09-04 with Go 1.26.2 on Windows amd64.

| Check | Result |
| --- | --- |
| `go test ./...` | Passed across API, domain, operations, and store packages |
| `go vet ./...` | Passed with no findings |
| `go build ./cmd/platform-api ./cmd/platformctl` | Passed |
| `go test -count=20 ./internal/...` | Passed twenty consecutive runs |
| Statement coverage | 39.9% repository-wide; 47.0% API, 63.6% domain, 68.5% reconciler, 30.4% store |
| Go generated service build | Passed |
| Python generated service syntax | Passed with `python -m py_compile` |
| Node generated service syntax | Passed with `node --check` |
| Portal JavaScript syntax | Passed with `node --check` |
| Portal visual inspection | Passed at desktop width with no clipping or overlap |

The local host has no C compiler, so the Go race detector cannot run locally. The Linux CI job runs `go test -race` and is the release gate for data-race verification.

## End-to-end create and replay

The compiled API accepted the example descriptor with HTTP 202, reconciled all four steps to `SUCCEEDED`, generated the service repository, and returned its links. Replaying the same payload with the same idempotency key returned HTTP 200, `replayed: true`, and the original operation rather than duplicating side effects.

## Fault exercise

`TestFailureCanBeRetriedFromCheckpoint` injects a render-step failure. The operation reaches `FAILED`, the service becomes `FAILED`, and a retry reason is appended to the audit trail. After the failpoint is removed, the operation resumes without repeating completed validation and planning steps, then reaches `SUCCEEDED`.

## Scope

Local evidence covers the dependency-free control plane and generated artefacts. A disposable Kubernetes environment is still required for Helm installation, Kyverno admission, Argo synchronization, and Terraform provider integration evidence.
