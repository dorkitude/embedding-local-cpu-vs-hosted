---
title: Reproduction and recovery
type: runbook
status: active
created: 2026-10-12
updated: 2026-10-12
related: ["[[STUDY_PLAN]]", "[[CORPUS_CARD]]", "[[RESULTS]]"]
---
Owner: Kyle Wild. Issue: https://github.com/dorkitude/embedding-gpu-vs-local/issues/1

Read CORPUS_CARD.md before redistributing source text. No raw corpus or vectors
are committed. Attribution manifest and checksums are in results/manifests.

1. Install Go (version in go.mod), uv, and Python 3.12. Use an actual system
   interpreter: uv venv --python /usr/bin/python3.12 sidecar/.venv
2. uv sync --frozen --project sidecar
3. go build -o .local/study ./cmd/study
4. .local/study acquire && .local/study prepare
5. Start sidecar using configs/embedding-study-sidecar.service (adjust absolute
   paths to checkout), systemctl --user daemon-reload, then start service.
6. .local/study preflight; .local/study integration; .local/study estimate
7. .local/study matrix --model MODEL --run UNIQUE-ID --pilot
8. Inspect/freeze manifests and budget. Launch full matrix from persistent
   systemd service, model gemma2 separately from hosted to overlap network work.
9. .local/study audit; go test ./...; go vet ./...
10. LIVE_INTEGRATION=1 go test ./internal/study -run TestIntegrationLive

Available integration defaults: OpenAI, Fireworks, OpenRouter, Hugging Face,
turbopuffer. Never read local secret files. Portable overrides:
OPENAI_BASE_URL / OPENAI_API_KEY; FIREWORKS_BASE_URL / FIREWORKS_API_KEY;
OPENROUTER_BASE_URL / OPENROUTER_API_KEY; TURBOPUFFER_BASE_URL / TURBOPUFFER_API_KEY;
HF_ENDPOINT for sidecar and corpus downloads (default: injected HF route).

Resume uses same run ID, corpus checksum and binary cache. Cache keys include
model, revision, formatting version, role, requested dimensions and full input.
Partial trailing cache records are truncated and retried. Writes are idempotent;
ingest checkpoints persist successful batch boundaries. Passed cells skip.
Failed embedding calls stop a model process rather than repeating all cells.
Retries max3 with 180s network timeout; total byte-token reservation cap $120.
Choose a NEW run ID when changing methodology. Never reuse another study's
namespace. Primary database is always turbopuffer; exact cosine is an audit only.

Smoke runs have only 1,000 documents/20 queries and are excluded from conclusions.
Full run has 96 expected cells: 6 models x 4 dimensions x 4 corpus/chunk settings.
Secondary 20k/256 is necessary for within-subset 128/256/512 comparisons.

Discovery: .local/study discover snapshots actual provider catalogs; preflight
performs real embedding calls. Catalog listing alone is never treated as access.
Generate tables/CIs/costs with .local/study report, then run
go run ./cmd/verify for the independent audit. Query logs may be compressed using
gzip -n after completion; both audit implementations transparently read .gz.
Primary full run local precision: bfloat16, batch8, threads8; benchmark scripts
must not be run concurrently with the final local run if isolated timing matters.

Primary evaluation amendment: run
.local/study requery-settled --model MODEL after that model's16 initial cells
finish. This is read-only with respect to vectors and turbopuffer data. It
archives the fresh-ingest observations under results/fresh-ingest and replaces
primary metrics only after recording index metadata/readiness probes. The
provider may retain exhaustive-search fallback even with up-to-date metadata.
Per-query counters expose that behavior; do not describe all ANN-operator
requests as pure approximate search. configs/embedding-study-settled@.service
waits up to4hours for the initial model lane and runs this second pass once.

The complete audit also runs go run ./cmd/verify-sample (rescans all66 source
shards), .local/study diagnostics, .local/study refresh-metadata, and
.local/study sentinel. The sentinel makes one bounded real embedding request per
model, comparing two fixed query vectors with the cached run. Its cosine/hash
receipts do not substitute for immutable hosted model revisions.

For a separate repetition, export STUDY_RUN_ID to a new unique identifier and
pass the same value to matrix --run. Report, requery-settled, refresh-metadata
and the independent verifier honor STUDY_RUN_ID (default r20261012a).
The committed systemd scripts are the exact commands for the original run;
copy/update their run ID for a new study repetition rather than overwriting the
original namespace. Corpus preparation remains fixed to this study's frozen
100k/20k design.
