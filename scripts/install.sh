#!/bin/sh
# pev installer — one-line bootstrap.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/samcofer/pev/main/scripts/install.sh | sh
#
# Environment overrides:
#   PEV_VERSION     — release tag to pin (default: latest, e.g. v0.0.2)
#   PEV_INSTALL_DIR — destination directory (default: ~/.local/bin for non-root,
#                     /usr/local/bin for root)
#
# Behavior:
#   - Detects Linux amd64 / arm64; refuses other platforms.
#   - Downloads the matching pev_linux_<arch> binary and the published
#     pev_<version>_checksums.txt; verifies SHA-256 before install.
#   - Always overwrites: re-running upgrades in place.
#
# Verifying this script before running it is recommended for regulated
# environments. Save it locally, inspect, then `sh install.sh`.

set -eu

REPO="samcofer/pev"
RELEASES_API="https://api.github.com/repos/${REPO}/releases/latest"
RELEASES_DOWNLOAD="https://github.com/${REPO}/releases/download"

err() { printf 'pev-install: %s\n' "$*" >&2; exit 1; }
log() { printf 'pev-install: %s\n' "$*"; }

need() { command -v "$1" >/dev/null 2>&1 || err "missing required tool: $1"; }
need curl
need sha256sum
need install
need mktemp

# -- Platform ------------------------------------------------------------------
os=$(uname -s)
[ "$os" = "Linux" ] || err "pev only supports Linux (detected: $os)"

case "$(uname -m)" in
  x86_64)  arch="amd64" ;;
  aarch64) arch="arm64" ;;
  *) err "pev only supports amd64 and arm64 (detected: $(uname -m))" ;;
esac

# -- Version -------------------------------------------------------------------
if [ -n "${PEV_VERSION:-}" ]; then
  version="$PEV_VERSION"
else
  # The "latest" endpoint returns a JSON blob; grep+sed is enough for the
  # tag_name field and avoids requiring jq.
  version=$(curl -fsSL "$RELEASES_API" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1)
  [ -n "$version" ] || err "could not resolve latest version from $RELEASES_API (set PEV_VERSION to override)"
fi
case "$version" in
  v*) ;;
  *) err "PEV_VERSION must start with 'v' (got: $version)" ;;
esac
version_no_v="${version#v}"

# -- Destination ---------------------------------------------------------------
if [ -n "${PEV_INSTALL_DIR:-}" ]; then
  dest="$PEV_INSTALL_DIR"
elif [ "$(id -u)" = "0" ]; then
  dest="/usr/local/bin"
else
  dest="${HOME}/.local/bin"
fi

# -- Download + verify + install ----------------------------------------------
tmp=$(mktemp -d) || err "could not create temp dir"
trap 'rm -rf "$tmp"' EXIT INT TERM

binary_name="pev_linux_${arch}"
checksums_name="pev_${version_no_v}_checksums.txt"

log "downloading $binary_name ($version)"
curl -fsSL "${RELEASES_DOWNLOAD}/${version}/${binary_name}"   -o "${tmp}/${binary_name}"
log "downloading $checksums_name"
curl -fsSL "${RELEASES_DOWNLOAD}/${version}/${checksums_name}" -o "${tmp}/${checksums_name}"

# sha256sum -c needs the checksum line in its CWD-relative form. We grep the
# row for our binary, then run -c against just that row.
expected_line=$(grep "  ${binary_name}\$" "${tmp}/${checksums_name}" || true)
[ -n "$expected_line" ] || err "no checksum entry for ${binary_name} in ${checksums_name}"
( cd "$tmp" && printf '%s\n' "$expected_line" | sha256sum -c - >/dev/null ) \
  || err "checksum mismatch for ${binary_name} — refusing to install"

mkdir -p "$dest"
install -m 0755 "${tmp}/${binary_name}" "${dest}/pev"

log "installed pev ${version} to ${dest}/pev"

# Hint about PATH if a non-root user installed somewhere not on PATH.
case ":${PATH}:" in
  *:"${dest}":*) ;;
  *) log "note: ${dest} is not on PATH; add it (e.g. export PATH=\"${dest}:\$PATH\")" ;;
esac
