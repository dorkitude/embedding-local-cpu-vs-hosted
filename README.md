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

## Summary results

Primary comparison: 100,000 MIRACL English passages, all 799 native dev
queries, 256-word chunks, native dimensions, turbopuffer settled-state search.

| Model | Native dims | nDCG@10 (95% bootstrap CI) | MRR@10 | Recall@100 |
|---|---:|---:|---:|---:|
| Qwen3-Embedding-8B (Fireworks) | 4096 | **0.8474** [0.8332, 0.8606] | 0.8622 | 0.9911 |
| Gemini Embedding 2 (OpenRouter) | 3072 | 0.8424 [0.8270, 0.8578] | 0.8680 | 0.9728 |
| OpenAI text-embedding-3-large | 3072 | 0.8059 [0.7905, 0.8209] | 0.8270 | 0.9908 |
| Local EmbeddingGemma 2 (8 vCPU, no GPU) | 768 | 0.8002 [0.7836, 0.8160] | 0.8320 | 0.9567 |
| BGE-M3 (OpenRouter) | 1024 | 0.7975 [0.7816, 0.8137] | 0.8318 | 0.9642 |
| OpenAI text-embedding-3-small | 1536 | 0.7719 [0.7555, 0.7879] | 0.7886 | 0.9860 |

Headline findings:

- **Local CPU was competitive.** Local Gemma 2 scored within 0.006 nDCG of
  OpenAI-large (paired difference -0.0056 [-0.0207, 0.0088]; not proven
  equivalent) at $0 API token cost, averaging 13.02 distinct inputs/s.
- **Qwen at 512d is the leading follow-up configuration:** 0.8362 nDCG at one
  eighth of its native vector payload. Gemini at 512d scored 0.8363 but is
  below Google's recommended operating dimensions.
- **MRL truncation does not generalize across families.** BGE-M3's unsupported
  128d prefix collapsed to 0.6359 (from 0.7975 native).
- **ANN approximation losses are uneven.** On the 64-query exact-cosine audit,
  native Gemma lost 0.029 nDCG to ANN while OpenAI-large lost 0.002; Qwen's ANN
  scored slightly above exact. Scores include turbopuffer search effects.
- **Chunk size mattered little** on these short passages (median 53 words):
  128/256/512-word chunks stayed within roughly 0.01 nDCG on the 20k corpus.
- **Cost:** estimated $10.31 embedding usage plus $13.52 turbopuffer metering;
  retained primary namespaces approximately $4.25/month.

These scores describe the sampled corpus and end-to-end retrieval
configurations, not official full-corpus MIRACL rankings. No GPU hardware was
controlled or measured.

**Deeper reading:** [practical interpretation](DISCUSSION.md) ·
[full results, sweeps, audits and limitations](RESULTS.md) ·
[machine-readable completion receipt](COMPLETION.json) ·
[per-query records and diagnostics](results/) ·
[reproduction runbook](RUNBOOK.md)

## Design

The study evaluates six embedding models on 100,000 MIRACL English passages and
all 799 native dev queries. A nested 20,000-passage corpus supports the chunk
sweep. The 96-cell design combines 128/256/512/native dimensions with primary
256-word chunks and secondary 128/256/512-word chunks. Turbopuffer stores and
searches every primary configuration; local exact cosine is a separate audit.

Read the [frozen study plan](STUDY_PLAN.md),
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
