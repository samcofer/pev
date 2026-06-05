#!/usr/bin/env bash
# Local Posit-image runner: builds pev, mounts it into the pinned official
# Workbench and Connect images, runs `pev assess`, and asserts that the
# baseline scenario (no inputs supplied; SSL/egress checks skipped) finds
# zero blocking failures.
#
# The same digests must be set in .github/workflows/posit-images.yml.
# Refresh both with scripts/refresh-posit-images.sh.
set -euo pipefail

cd "$(dirname "$0")/../.."

# Posit's official, fully-installed product images. Pinned by repository
# digest (sha256) so an upstream rebuild cannot silently change pev's test
# results. Refresh with scripts/refresh-posit-images.sh.
WORKBENCH_IMAGE="docker.io/posit/workbench@sha256:3d76d4b38651287d158c59fe82fc7e7d5bb8de1a1178a922bd50a90892d0d87a"
CONNECT_IMAGE="docker.io/posit/connect@sha256:8de3d13ba5fdd46576ecdcbdff43420305407192d2c0de462545eba1ba082de9"

CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o pev ./

run_one() {
  local label="$1" image="$2" product="$3"
  local outdir
  outdir=$(mktemp -d -t "pev-posit-${label}.XXXX")
  echo "=========================================="
  echo "  ${label}: ${image}"
  echo "=========================================="
  docker run --rm \
    --entrypoint="" \
    -v "$PWD/pev":/usr/local/bin/pev:ro \
    -v "$outdir":/out \
    "$image" \
    pev assess --non-interactive --products "$product" \
      --out-dir /out --skip-tags egress,ssl || true

  local report
  report=$(ls "$outdir"/pev-report-*.json | head -1)
  echo "report: $report"
  jq '.summary' "$report"

  local blocking
  blocking=$(jq -r '.summary.blocking_failures' "$report")
  if [[ "$blocking" != "0" ]]; then
    echo "FAIL: ${label} produced ${blocking} blocking failure(s)"
    jq -r '.results[] | select(.severity=="blocking" and .status=="fail") | "  - \(.id): \(.reason // "")"' "$report"
    return 1
  fi
  echo "OK: ${label} blocking_failures=0"
}

run_one workbench "$WORKBENCH_IMAGE" workbench
run_one connect   "$CONNECT_IMAGE"   connect
