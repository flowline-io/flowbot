#!/usr/bin/env bash
set -euo pipefail

# Downloads frontend CDN dependencies into public/vendor/ for local serving.
# Run from repository root. No node_modules / npm project required.
#
# Pinned versions (update these to bump dependencies):
HTMX_VERSION=2.0.4
ALPINE_VERSION=3.15.12
JS_YAML_VERSION=4.1.0
DAISYUI_VERSION=5.0.9

VENDOR_DIR=public/vendor
mkdir -p "$VENDOR_DIR"

echo "Downloading htmx ${HTMX_VERSION}..."
curl -sL "https://unpkg.com/htmx.org@${HTMX_VERSION}/dist/htmx.min.js" -o "${VENDOR_DIR}/htmx.min.js"

echo "Downloading Alpine.js CSP ${ALPINE_VERSION}..."
curl -sL "https://cdn.jsdelivr.net/npm/@alpinejs/csp@${ALPINE_VERSION}/dist/cdn.min.js" -o "${VENDOR_DIR}/alpine.csp.min.js"

echo "Downloading Alpine.js (non-CSP, optional) ${ALPINE_VERSION}..."
curl -sL "https://cdn.jsdelivr.net/npm/alpinejs@${ALPINE_VERSION}/dist/cdn.min.js" -o "${VENDOR_DIR}/alpine.min.js"

echo "Downloading js-yaml ${JS_YAML_VERSION}..."
curl -sL "https://cdn.jsdelivr.net/npm/js-yaml@${JS_YAML_VERSION}/dist/js-yaml.min.js" -o "${VENDOR_DIR}/js-yaml.min.js"

echo "Downloading DaisyUI ${DAISYUI_VERSION} CSS (reference; layouts use public/css/app.css)..."
curl -sL "https://cdn.jsdelivr.net/npm/daisyui@${DAISYUI_VERSION}/daisyui.css" -o "${VENDOR_DIR}/daisyui.css"

echo "Downloading DaisyUI ${DAISYUI_VERSION} themes CSS..."
curl -sL "https://cdn.jsdelivr.net/npm/daisyui@${DAISYUI_VERSION}/themes.css" -o "${VENDOR_DIR}/themes.css"

echo ""
echo "Vendor files downloaded to ${VENDOR_DIR}/"
echo "Note: public/css/app.css is a committed static bundle (Tailwind utilities + DaisyUI)."
echo "It is not produced by this script; update it separately if styles change."
echo "Files:"
ls -lh "${VENDOR_DIR}"/
echo ""
echo "Done. Vendor files are tracked in git (under public/). Re-run to update versions."
