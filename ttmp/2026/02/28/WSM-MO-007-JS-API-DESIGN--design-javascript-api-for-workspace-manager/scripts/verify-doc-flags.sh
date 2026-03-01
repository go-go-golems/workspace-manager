#!/usr/bin/env bash
# verify-doc-flags.sh — Verify that documented flags match actual CLI help output.
# Run from the workspace-manager root after building wsm.
set -euo pipefail

WSM="${WSM:-./wsm}"

echo "=== Registry Commands ==="
echo "--- discover ---"
$WSM discover --help 2>&1 || true
echo
echo "--- list repos ---"
$WSM list repos --help 2>&1 || true
echo
echo "--- list workspaces ---"
$WSM list workspaces --help 2>&1 || true

echo
echo "=== Workspace Commands ==="
for cmd in create info status add remove fork merge delete; do
  echo "--- $cmd ---"
  $WSM $cmd --help 2>&1 || true
  echo
done

echo "=== Git Commands ==="
for cmd in commit diff log; do
  echo "--- $cmd ---"
  $WSM $cmd --help 2>&1 || true
  echo
done

echo "--- branch ---"
$WSM branch --help 2>&1 || true
for sub in create switch list; do
  echo "--- branch $sub ---"
  $WSM branch $sub --help 2>&1 || true
  echo
done

echo "--- rebase ---"
$WSM rebase --help 2>&1 || true
for sub in status continue abort; do
  echo "--- rebase $sub ---"
  $WSM rebase $sub --help 2>&1 || true
  echo
done

echo "=== JS Commands ==="
echo "--- runner ---"
$WSM runner --help 2>&1 || true

echo
echo "Done. Compare above output against pkg/docs/*.md documentation."
