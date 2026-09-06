#!/usr/bin/env bash
set -euo pipefail

version="${VERSION:-dev}"
commit="$(git rev-parse --short HEAD)"
build_date="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
ldflags="-s -w -X main.version=${version} -X main.commit=${commit} -X main.buildDate=${build_date}"
rm -rf dist
mkdir -p dist

for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do
  goos="${target%/*}"
  goarch="${target#*/}"
  name="internal-developer-platform_${version}_${goos}_${goarch}"
  staging="dist/${name}"
  extension=""
  if [[ "${goos}" == "windows" ]]; then
    extension=".exe"
  fi
  mkdir -p "${staging}"
  GOOS="${goos}" GOARCH="${goarch}" CGO_ENABLED=0 go build -trimpath -ldflags="${ldflags}" -o "${staging}/platform-api${extension}" ./cmd/platform-api
  GOOS="${goos}" GOARCH="${goarch}" CGO_ENABLED=0 go build -trimpath -ldflags="${ldflags}" -o "${staging}/platformctl${extension}" ./cmd/platformctl
  cp README.md LICENSE "${staging}/"
  mkdir -p "${staging}/api" "${staging}/docs" "${staging}/examples"
  cp api/openapi.yaml "${staging}/api/"
  cp docs/getting-started.md docs/configuration.md docs/troubleshooting.md "${staging}/docs/"
  cp examples/payments-notifier.json "${staging}/examples/"
  if [[ "${goos}" == "windows" ]]; then
    (cd dist && zip -qr "${name}.zip" "${name}")
  else
    tar -C dist -czf "dist/${name}.tar.gz" "${name}"
  fi
  rm -rf "${staging}"
done

(cd dist && sha256sum ./*.tar.gz ./*.zip > checksums.txt)
