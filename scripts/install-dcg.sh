#!/bin/sh
# Download and verify Destructive Command Guard (dcg) into OUT_DIR (default /out).
# Shared by deployments/Dockerfile and deployments/agent-sandbox/Dockerfile.
set -eu

OUT_DIR="${1:-/out}"
DCG_VERSION="${DCG_VERSION:-v0.6.7}"
DCG_REPO="${DCG_REPO:-Dicklesworthstone/destructive_command_guard}"
DCG_ASSET="${DCG_ASSET:-dcg-x86_64-unknown-linux-musl.tar.xz}"

mkdir -p "${OUT_DIR}"
BASE="https://github.com/${DCG_REPO}/releases/download/${DCG_VERSION}"
curl -fsSL --retry 3 --retry-delay 2 -o "/tmp/${DCG_ASSET}" "${BASE}/${DCG_ASSET}"
curl -fsSL --retry 3 --retry-delay 2 -o "/tmp/${DCG_ASSET}.sha256" "${BASE}/${DCG_ASSET}.sha256"
EXPECTED="$(awk '{print $1}' "/tmp/${DCG_ASSET}.sha256")"
ACTUAL="$(sha256sum "/tmp/${DCG_ASSET}" | awk '{print $1}')"
test -n "${EXPECTED}"
test "${EXPECTED}" = "${ACTUAL}"
tar -xJf "/tmp/${DCG_ASSET}" -C /tmp
install -m 755 /tmp/dcg "${OUT_DIR}/dcg"
