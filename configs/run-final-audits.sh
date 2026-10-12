#!/usr/bin/env bash
set -euo pipefail
run="${STUDY_RUN_ID:-r20261012a}"
ready=false
for attempt in $(seq 1 360); do
  shopt -s nullglob
  files=(results/cells/"$run"-*.json)
  if [ "${#files[@]}" -eq 96 ] && jq -e -s 'all(.[]; .status == "passed" and .evaluation_phase == "settled-v1")' "${files[@]}" >/dev/null; then
    ready=true
    break
  fi
  sleep 30
done
if [ "$ready" != true ]; then
  echo 'Timed out waiting for complete settled matrix' >&2
  exit 1
fi
date -u
.local/study exact-expanded
.local/study sentinel
.local/study refresh-metadata
go test ./...
go vet ./...
LIVE_INTEGRATION=1 go test ./internal/study -run TestIntegrationLive -count=1
# Full inference has ended; benchmark runs alone with explicit original FP32 load.
uv run --frozen --project sidecar sidecar/benchmark.py
.local/study audit
.local/study report
go run ./cmd/verify
.local/study report
jq -n --arg utc "$(date -u +%FT%TZ)" --arg run "$run" \
  '{status:"passed",utc:$utc,run_id:$run,commands:["go test ./...","go vet ./...","LIVE_INTEGRATION=1 go test ./internal/study -run TestIntegrationLive -count=1","study exact-expanded","study sentinel","study refresh-metadata","uv run --frozen --project sidecar sidecar/benchmark.py","study audit","study report","go run ./cmd/verify"],sampling_audit:"results/independent-sampling-audit.json"}' > results/validation.json
date -u
