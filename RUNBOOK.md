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
HF_ENDPOINT for sidecar. Dataset downloader currently uses the injected HF route.

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
