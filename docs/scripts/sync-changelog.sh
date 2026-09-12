#!/usr/bin/env bash
# Regenerate docs/content/changelog.md from GitHub Releases.
#
# GitHub Releases are the single source of truth for the changelog; the
# committed docs/content/changelog.md is a generated artifact refreshed on
# every docs deploy.
#
# Usage: sync-changelog.sh [repo-root]   (default: .)
set -euo pipefail

root="${1:-.}"
out="${root}/docs/content/changelog.md"
repo="${GITHUB_REPOSITORY:?GITHUB_REPOSITORY must be set}"

{
  echo "---"
  echo "title: Changelog"
  echo "description: Release history and updates for Aegis."
  echo "layout: default"
  echo "navigation:"
  echo "  icon: i-lucide-history"
  echo "---"
  echo ""
  echo "# Changelog"
  echo ""
  echo "The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),"
  echo "and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html)."
  echo ""
  curl -s "https://api.github.com/repos/${repo}/releases" \
    | jq -r '.[] | "## [\(.tag_name)] - \(.published_at | split("T")[0])\n\n\(.body)\n\n---\n"'
} > "${out}"

if [ "$(wc -l < "${out}")" -lt 15 ]; then
  {
    echo "## [Unreleased]"
    echo ""
    echo "No releases yet. Check the [GitHub repository](https://github.com/${repo}) for updates."
  } >> "${out}"
fi
