# Benchmark Evidence

## Method

Run benchmarks on an otherwise idle machine with:

```bash
go test -run '^$' -bench . -benchmem ./internal/...
```

Record the Go version, operating system, architecture, CPU, command, and raw output. The local store benchmark measures a full service acceptance transaction, including JSON serialization and durable file replacement. It does not include GitHub, Terraform, Argo CD, or network latency.

## Results

Measured on 2026-09-04 with Go 1.26.2 on Windows amd64 and an AMD Ryzen 9 9950X:

```text
BenchmarkCreateServiceDurableTransaction-32    2763    460213 ns/op    10724 B/op    53 allocs/op
```

This microbenchmark reports approximately 0.46 ms of local control-plane persistence per accepted service on this machine. Do not compare local filesystem numbers directly with the PostgreSQL production target.

## Interpretation

The blueprint's user outcome is time to first deployment, not raw handler throughput. The benchmark is a regression signal for control-plane overhead. End-to-end load evidence must separately measure queue wait, remote adapter duration, operation success, and first-ready time at representative concurrency.
