#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

export GOPATH="${GOPATH:-$ROOT_DIR/.cache/gopath}"
export GOCACHE="${GOCACHE:-$ROOT_DIR/.cache/go-build}"
export GOMODCACHE="${GOMODCACHE:-$ROOT_DIR/.cache/gomod}"
export GOSUMDB="${GOSUMDB:-off}"

export VERSION="${VERSION:-0.1.0-dev}"
export COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo local)}"
export BUILD_DATE="${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
export REPO="${REPO:-}"
export PLATFORM="${PLATFORM:-darwin/arm64}"

node scripts/build-release.mjs
