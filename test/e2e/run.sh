#!/usr/bin/env bash
# Local e2e runner: builds pev, then runs it inside Ubuntu 22/24 + Alma 9/10
# containers, exercising both root and non-root paths. Mirrors the CI workflow
# in .github/workflows/e2e.yml so behavior matches between local and CI.
set -euo pipefail

cd "$(dirname "$0")/../.."

CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o pev ./

IMAGES=(ubuntu:22.04 ubuntu:24.04 almalinux:9 almalinux:10)

for img in "${IMAGES[@]}"; do
  echo "=========================================="
  echo "  $img"
  echo "=========================================="
  docker run --rm \
    -v "$PWD":/work -w /work \
    "$img" \
    bash -c '
      set -e
      if command -v apt-get >/dev/null; then
        apt-get update -qq
        apt-get install -yqq --no-install-recommends ca-certificates curl iproute2 procps openssl >/dev/null
      elif command -v dnf >/dev/null; then
        dnf -y install ca-certificates curl iproute procps-ng openssl which >/dev/null
      fi

      ./pev version
      ./pev discover --format json | head -40

      mkdir -p /tmp/pev-root
      ./pev assess --non-interactive --out-dir /tmp/pev-root --skip-tags egress || true
      ls /tmp/pev-root/

      mkdir -p /tmp/pev-nobody && chown -R 65534:65534 /tmp/pev-nobody
      if command -v runuser >/dev/null; then
        runuser -u nobody -- ./pev assess --non-interactive --out-dir /tmp/pev-nobody --skip-tags egress || true
      else
        su -s /bin/sh nobody -c "./pev assess --non-interactive --out-dir /tmp/pev-nobody --skip-tags egress" || true
      fi
      ls /tmp/pev-nobody/
    '
done
