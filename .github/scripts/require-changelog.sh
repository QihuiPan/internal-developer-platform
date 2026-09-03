#!/usr/bin/env bash
set -euo pipefail

base_ref="${GITHUB_BASE_REF:?GITHUB_BASE_REF is required}"
git fetch --no-tags --depth=1 origin "${base_ref}"
base_commit="$(git merge-base HEAD "origin/${base_ref}")"

if ! git diff --name-only "${base_commit}" HEAD | grep -qx 'CHANGELOG.md'; then
  echo 'Every pull request must update CHANGELOG.md with an English entry.' >&2
  exit 1
fi
