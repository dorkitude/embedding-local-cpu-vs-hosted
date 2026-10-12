---
title: Embedding GPU vs Local
type: research-study
status: active
created: 2026-10-12
updated: 2026-10-12
related: ["[[STUDY_PLAN]]", "[[CORPUS_CARD]]", "[[RESULTS]]"]
---
Private research study owned by Kyle Wild. Actual comparison: local CPU versus
hosted embedding retrieval; GPU hardware is not controlled or measured.

Execution issue: https://github.com/dorkitude/embedding-gpu-vs-local/issues/1

The study evaluates six embedding models on 100,000 MIRACL English passages and
all 799 native dev queries. A nested 20,000-passage corpus supports the chunk
sweep. The 96-cell design combines 128/256/512/native dimensions with primary
256-word chunks and secondary 128/256/512-word chunks. Turbopuffer stores and
searches every primary configuration; local exact cosine is a separate audit.

Read the [results and limitations](RESULTS.md), [frozen study plan](STUDY_PLAN.md),
[corpus license and attribution review](CORPUS_CARD.md),
[model access record](MODEL_DISCOVERY.md), and [execution ledger](LEDGER.md).
The results document reports actual completion status; initial fresh-ingest
observations are separate from the settled primary evaluation.

Models: local EmbeddingGemma 2, OpenAI text-embedding-3-large and 3-small,
Fireworks Qwen3-Embedding-8B, and OpenRouter Gemini Embedding 2 and BGE-M3.
The VM has eight CPU cores and no GPU. Hosted API observations do not identify
or control provider GPU hardware. Scores describe the sampled corpus and
end-to-end retrieval configurations, not official full-corpus MIRACL rankings.

Corpus text permits redistribution under Wikipedia CC BY-SA 3.0 with its
attribution/share-alike requirements, alongside Apache-2.0 MIRACL contributions.
The repository includes all selected passage IDs, attribution, license receipts,
checksums, and the deterministic acquisition recipe. Raw text, weights, caches,
and vectors stay outside Git. See [NOTICE](NOTICE).

Reproduction and recovery commands are in [RUNBOOK.md](RUNBOOK.md). The primary
harness uses Go/Cobra; Python via uv is confined to local inference/benchmarking.
Tests: `go test ./...` and `go vet ./...`. Offline metric verification from the
committed records: `go run ./cmd/verify`. This independent verifier does not
import the harness evaluator or require provider credentials.
