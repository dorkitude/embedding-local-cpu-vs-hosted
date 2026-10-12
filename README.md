---
title: Embedding GPU vs Local
type: research-study
status: complete
created: 2026-10-12
updated: 2026-10-12
related: ["[[STUDY_PLAN]]", "[[CORPUS_CARD]]", "[[RESULTS]]"]
---
Private research study owned by Kyle Wild. Actual comparison: local CPU versus
hosted embedding retrieval; GPU hardware is not controlled or measured.

Execution issue: https://github.com/dorkitude/embedding-gpu-vs-local/issues/1

**Completed and audited: 96/96 configurations, six models, 76,704 primary
query evaluations.** Independent code recomputed all reported metrics and
296 confidence intervals. Unit tests, vet, live integration, the expanded
exact-search audit, and the isolated CPU confirmation passed.

Read [practical findings](DISCUSSION.md), [full results](RESULTS.md), and the
machine-readable [completion receipt](COMPLETION.json). Native primary nDCG@10:
Qwen 0.8474, Gemini 0.8424, OpenAI-large 0.8059, local Gemma 2 0.8002,
BGE-M3 0.7975, OpenAI-small 0.7719. These scores include turbopuffer search
effects; the exact audits show material differences in approximation loss.
Local inference averaged 13.02 distinct inputs/s. Estimated API/database usage
was $23.84, excluding storage and account minimums; retained primary namespaces
cost approximately $4.19/month at listed storage rates.

The study evaluates six embedding models on 100,000 MIRACL English passages and
all 799 native dev queries. A nested 20,000-passage corpus supports the chunk
sweep. The 96-cell design combines 128/256/512/native dimensions with primary
256-word chunks and secondary 128/256/512-word chunks. Turbopuffer stores and
searches every primary configuration; local exact cosine is a separate audit.

Read the [results and limitations](RESULTS.md), [frozen study plan](STUDY_PLAN.md),
[corpus license and attribution review](CORPUS_CARD.md),
[model access record](MODEL_DISCOVERY.md), and [execution ledger](LEDGER.md).
Initial fresh-ingest observations are preserved separately from the settled
primary evaluation. Run ID: r20261012a; primary phase: settled-v1.

Models: local EmbeddingGemma 2, OpenAI text-embedding-3-large and 3-small,
Fireworks Qwen3-Embedding-8B, and OpenRouter Gemini Embedding 2 and BGE-M3.
The VM has eight vCPUs and no GPU. Hosted API observations do not identify
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
