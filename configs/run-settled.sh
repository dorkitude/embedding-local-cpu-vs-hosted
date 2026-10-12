#!/usr/bin/env bash
set -euo pipefail
model="$1"
for attempt in $(seq 1 480); do
 shopt -s nullglob
 files=(results/cells/r20261012a-"$model"-*.json)
 if [ "${#files[@]}" -eq 16 ] && jq -e -s 'all(.[]; .status == "passed")' "${files[@]}" >/dev/null; then
  exec .local/study requery-settled --model "$model"
 fi
 sleep 30
done
echo "Timed out waiting for initial model lane: $model" >&2
exit 1
