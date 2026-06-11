#!/usr/bin/env bash
#
# Release helper: computes the next semantic version, generates a changelog
# section from the commits since the last tag, bumps wails.json, then commits
# and creates an annotated git tag.
#
# Usage:
#   scripts/release.sh changelog                 # preview unreleased changes
#   scripts/release.sh release patch|minor|major # bump from the latest tag
#   scripts/release.sh release 1.2.3             # set an explicit version
#
set -euo pipefail

cd "$(dirname "$0")/.."

CHANGELOG="CHANGELOG.md"
MARKER="<!-- new-release -->"

latest_tag() {
  git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"
}

bump_version() {
  local ver="${1#v}" level="$2"
  IFS=. read -r major minor patch <<<"$ver"
  case "$level" in
    major) major=$((major + 1)); minor=0; patch=0 ;;
    minor) minor=$((minor + 1)); patch=0 ;;
    patch) patch=$((patch + 1)) ;;
    *) echo "error: unknown bump level '$level' (use patch|minor|major)" >&2; exit 1 ;;
  esac
  echo "v${major}.${minor}.${patch}"
}

# notes <range> -> bulleted list of commit subjects (excludes merges & releases)
notes() {
  local range="$1"
  if [ -n "$range" ]; then
    git log --no-merges --invert-grep --grep='^Release v' --pretty='- %s' "$range"
  else
    git log --no-merges --invert-grep --grep='^Release v' --pretty='- %s'
  fi
}

range_since() {
  local last="$1"
  if [ "$last" = "v0.0.0" ] && ! git rev-parse "v0.0.0" >/dev/null 2>&1; then
    echo "" # no tags yet -> all history
  else
    echo "${last}..HEAD"
  fi
}

cmd="${1:-}"
case "$cmd" in
  changelog)
    last="$(latest_tag)"
    body="$(notes "$(range_since "$last")")"
    echo "## Unreleased (since ${last})"
    echo
    [ -n "$body" ] && echo "$body" || echo "- (no changes)"
    ;;

  release)
    arg="${2:-patch}"
    [ -z "$arg" ] && arg="patch" # empty (e.g. `make release` with no V) -> patch
    last="$(latest_tag)"
    if [[ "$arg" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
      next="v${arg#v}"
    else
      next="$(bump_version "$last" "$arg")"
    fi

    if [ -n "$(git status --porcelain)" ]; then
      echo "error: working tree is not clean; commit or stash first." >&2
      exit 1
    fi
    if git rev-parse "$next" >/dev/null 2>&1; then
      echo "error: tag $next already exists." >&2
      exit 1
    fi

    body="$(notes "$(range_since "$last")")"
    if [ -z "$body" ]; then
      echo "error: no commits since ${last}; nothing to release." >&2
      exit 1
    fi

    date="$(date +%Y-%m-%d)"
    ver="${next#v}"
    echo "Releasing ${next} (previous: ${last})"

    # Ensure the changelog exists with a header + insertion marker.
    if [ ! -f "$CHANGELOG" ]; then
      {
        echo "# Changelog"
        echo
        echo "All notable changes to this project are documented here."
        echo "The format is based on Keep a Changelog, and this project adheres to"
        echo "Semantic Versioning (https://semver.org)."
        echo
        echo "$MARKER"
      } >"$CHANGELOG"
    fi

    # Write the new section and insert it right after the marker.
    entry="$(mktemp)"
    {
      echo "## [${ver}] - ${date}"
      echo
      echo "$body"
    } >"$entry"

    tmp="$(mktemp)"
    awk -v marker="$MARKER" -v ef="$entry" '
      { print }
      $0 == marker {
        print ""
        while ((getline line < ef) > 0) print line
        close(ef)
      }
    ' "$CHANGELOG" >"$tmp"
    mv "$tmp" "$CHANGELOG"
    rm -f "$entry"

    # Keep the bundle version (wails.json) in sync.
    tmp="$(mktemp)"
    sed -E 's/("productVersion"[[:space:]]*:[[:space:]]*")[^"]*(")/\1'"$ver"'\2/' wails.json >"$tmp"
    mv "$tmp" wails.json

    git add "$CHANGELOG" wails.json
    git commit -q -m "Release ${next}"
    git tag -a "${next}" -m "Release ${next}"

    echo "Tagged ${next}."
    echo "Push with:  git push && git push origin ${next}"
    ;;

  *)
    echo "usage: $0 {changelog | release [patch|minor|major|X.Y.Z]}" >&2
    exit 1
    ;;
esac
