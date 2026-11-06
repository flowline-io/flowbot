#!/usr/bin/env bash
# Warn (or, from 2026-11-16 UTC, fail) when a contract-sensitive diff lands
# without a new non-empty line under CHANGELOG.md ## Unreleased.
#
# Triggers (any):
#   - deleted immediate child dirs under pkg/providers, pkg/capability,
#     internal/modules, or cmd
#   - changed string/alias constants in pkg/auth/scope.go
#   - changed top-level keys in docs/reference/config.yaml
#   - any change to internal/server/mcp/testdata/tools.snapshot.json
#
# Env:
#   BASE_SHA   git commit to diff against (required unless resolvable below)
#   BASE_REF   fallback name/SHA when BASE_SHA is empty or all-zero
#   BLOCK_ON   YYYY-MM-DD (UTC); default 2026-11-16
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

BLOCK_ON="${BLOCK_ON:-2026-11-16}"
CHANGELOG="CHANGELOG.md"
SCOPE_FILE="pkg/auth/scope.go"
CONFIG_FILE="docs/reference/config.yaml"
SNAPSHOT_FILE="internal/server/mcp/testdata/tools.snapshot.json"
WATCH_DIR_PARENTS=(pkg/providers pkg/capability internal/modules cmd)

resolve_base() {
  local sha="${BASE_SHA:-}"
  if [[ -z "$sha" || "$sha" =~ ^0+$ ]]; then
    sha="${BASE_REF:-}"
  fi
  if [[ -z "$sha" || "$sha" =~ ^0+$ ]]; then
    if git rev-parse --verify origin/master >/dev/null 2>&1; then
      sha="origin/master"
    elif git rev-parse --verify master >/dev/null 2>&1; then
      sha="master"
    else
      echo "check-changelog-unreleased: no BASE_SHA/BASE_REF; skip" >&2
      exit 0
    fi
  fi
  if ! git cat-file -e "${sha}^{commit}" 2>/dev/null; then
    echo "check-changelog-unreleased: base ${sha} not in this clone; skip" >&2
    exit 0
  fi
  # Compare against the merge-base so PR tips only see their own delta.
  sha="$(git merge-base HEAD "$sha")"
  if [[ "$(git rev-parse HEAD)" == "$(git rev-parse "$sha")" ]]; then
    echo "check-changelog-unreleased: HEAD equals base; skip"
    exit 0
  fi
  printf '%s\n' "$sha"
}

# Non-empty lines under ## Unreleased until the next ## heading (sorted for comm).
unreleased_lines() {
  local rev="$1"
  git show "${rev}:${CHANGELOG}" 2>/dev/null | awk '
    BEGIN { in_u = 0 }
    /^## / {
      if ($0 == "## Unreleased") { in_u = 1; next }
      if (in_u) { exit }
    }
    in_u && NF { print }
  ' | sort
}

child_dirs_at() {
  local rev="$1" parent="$2"
  if ! git cat-file -e "${rev}:${parent}" 2>/dev/null; then
    return 0
  fi
  git ls-tree -d --name-only "${rev}:${parent}" 2>/dev/null | sed "s|^|${parent}/|" || true
}

deleted_watched_dirs() {
  local base="$1" parent base_dirs head_dirs d
  for parent in "${WATCH_DIR_PARENTS[@]}"; do
    mapfile -t base_dirs < <(child_dirs_at "$base" "$parent")
    mapfile -t head_dirs < <(child_dirs_at "HEAD" "$parent")
    for d in "${base_dirs[@]}"; do
      [[ -z "$d" ]] && continue
      local found=0 h
      for h in "${head_dirs[@]}"; do
        if [[ "$h" == "$d" ]]; then
          found=1
          break
        fi
      done
      if [[ "$found" -eq 0 ]]; then
        printf '%s\n' "$d"
      fi
    done
  done
}

# Const assignments and aliases: Name = "..." or Name = OtherName
scope_consts() {
  local rev="$1"
  git show "${rev}:${SCOPE_FILE}" 2>/dev/null | awk '
    /^const \(/ { in_c = 1; next }
    in_c && /^\)/ { in_c = 0; next }
    in_c {
      line = $0
      sub(/\/\/.*/, "", line)
      if (match(line, /[A-Za-z_][A-Za-z0-9_]*[[:space:]]*=[[:space:]]*("([^"\\]|\\.)*"|[A-Za-z_][A-Za-z0-9_]*)/)) {
        s = substr(line, RSTART, RLENGTH)
        gsub(/[[:space:]]+/, " ", s)
        print s
      }
    }
  ' | sort -u
}

# Uncommented top-level YAML keys (indent 0, key:).
config_top_keys() {
  local rev="$1"
  git show "${rev}:${CONFIG_FILE}" 2>/dev/null | awk '
    /^[a-zA-Z0-9_]+:/ {
      key = $0
      sub(/:.*/, "", key)
      print key
    }
  ' | sort -u
}

file_changed() {
  local base="$1" path="$2"
  ! git diff --quiet "$base" HEAD -- "$path"
}

BASE="$(resolve_base)"
echo "check-changelog-unreleased: base=${BASE} head=$(git rev-parse --short HEAD)"

triggers=()

while IFS= read -r d; do
  [[ -z "$d" ]] && continue
  triggers+=("deleted directory ${d}")
done < <(deleted_watched_dirs "$BASE")

if file_changed "$BASE" "$SCOPE_FILE"; then
  tmp_base="$(mktemp)"
  tmp_head="$(mktemp)"
  scope_consts "$BASE" >"$tmp_base"
  scope_consts "HEAD" >"$tmp_head"
  if ! cmp -s "$tmp_base" "$tmp_head"; then
    triggers+=("pkg/auth/scope.go constants changed")
  fi
  rm -f "$tmp_base" "$tmp_head"
fi

if file_changed "$BASE" "$CONFIG_FILE"; then
  tmp_base="$(mktemp)"
  tmp_head="$(mktemp)"
  config_top_keys "$BASE" >"$tmp_base"
  config_top_keys "HEAD" >"$tmp_head"
  if ! cmp -s "$tmp_base" "$tmp_head"; then
    triggers+=("docs/reference/config.yaml top-level keys changed")
  fi
  rm -f "$tmp_base" "$tmp_head"
fi

if file_changed "$BASE" "$SNAPSHOT_FILE"; then
  triggers+=("${SNAPSHOT_FILE} changed")
fi

if [[ "${#triggers[@]}" -eq 0 ]]; then
  echo "check-changelog-unreleased: no contract-sensitive triggers"
  exit 0
fi

echo "check-changelog-unreleased: triggers:"
for t in "${triggers[@]}"; do
  echo "  - ${t}"
done

tmp_u_base="$(mktemp)"
tmp_u_head="$(mktemp)"
unreleased_lines "$BASE" >"$tmp_u_base"
unreleased_lines "HEAD" >"$tmp_u_head"

added="$(comm -13 "$tmp_u_base" "$tmp_u_head" || true)"
rm -f "$tmp_u_base" "$tmp_u_head"

if [[ -n "$added" ]]; then
  echo "check-changelog-unreleased: Unreleased has new line(s); ok"
  exit 0
fi

today="$(date -u +%Y-%m-%d)"
trigger_summary=$(printf '%s; ' "${triggers[@]}")
trigger_summary="${trigger_summary%; }"
msg="Contract-sensitive change without a new CHANGELOG.md Unreleased line. Triggers: ${trigger_summary}. Add a bullet under ## Unreleased (see CONTRIBUTING.md)."

if [[ "$today" < "$BLOCK_ON" ]]; then
  echo "::warning title=Changelog Unreleased::${msg}"
  echo "check-changelog-unreleased: WARNING (non-blocking until ${BLOCK_ON} UTC): ${msg}" >&2
  exit 0
fi

echo "::error title=Changelog Unreleased::${msg}"
echo "check-changelog-unreleased: FAIL (blocking since ${BLOCK_ON} UTC): ${msg}" >&2
exit 1
