#!/usr/bin/env bash
set -euo pipefail

smoke_root="$(mktemp -d)"
api_pid=""
cleanup() {
  if [[ -n "${api_pid}" ]]; then
    kill "${api_pid}" 2>/dev/null || true
    wait "${api_pid}" 2>/dev/null || true
  fi
  rm -rf "${smoke_root}"
}
trap cleanup EXIT

go build -o "${smoke_root}/platform-api" ./cmd/platform-api
go build -o "${smoke_root}/platformctl" ./cmd/platformctl
"${smoke_root}/platform-api" \
  --address 127.0.0.1:18080 \
  --data "${smoke_root}/state.json" \
  --generated-root "${smoke_root}/generated" >"${smoke_root}/api.log" 2>&1 &
api_pid="$!"

for _ in {1..40}; do
  if curl --fail --silent http://127.0.0.1:18080/healthz >/dev/null; then
    break
  fi
  sleep 0.25
done
curl --fail --silent http://127.0.0.1:18080/ | grep --quiet "Internal Developer Platform"
"${smoke_root}/platformctl" create \
  --address http://127.0.0.1:18080 \
  --file examples/payments-notifier.json \
  --timeout 10s >"${smoke_root}/operation.json"
grep --quiet '"status": "SUCCEEDED"' "${smoke_root}/operation.json"
"${smoke_root}/platformctl" list --address http://127.0.0.1:18080 >"${smoke_root}/services.json"
grep --quiet 'payments-notifier' "${smoke_root}/services.json"
"${smoke_root}/platformctl" download \
  --address http://127.0.0.1:18080 \
  --output "${smoke_root}/payments-notifier.zip" \
  payments-notifier
test -s "${smoke_root}/payments-notifier.zip"
test -f "${smoke_root}/generated/payments-notifier/Dockerfile"
