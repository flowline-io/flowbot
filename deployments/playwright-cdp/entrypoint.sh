#!/usr/bin/env bash
# Chromium binds remote debugging to loopback only; proxy 0.0.0.0:9222 for Docker networks.
set -euo pipefail

CHROME="$(find /ms-playwright -type f -path '*/chrome-linux/chrome' | head -n1 || true)"
if [[ -z "${CHROME}" || ! -x "${CHROME}" ]]; then
	echo "chromium binary not found under /ms-playwright" >&2
	exit 1
fi

socat TCP-LISTEN:9222,fork,reuseaddr,bind=0.0.0.0 TCP:127.0.0.1:9223 &
exec "${CHROME}" \
	--headless=new \
	--no-sandbox \
	--disable-gpu \
	--disable-dev-shm-usage \
	--remote-debugging-port=9223 \
	about:blank
