---
title: Embedding Local CPU vs Hosted
---
Research study comparing local CPU versus hosted embedding retrieval; GPU hardware is not controlled or measured.

Execution issue: https://github.com/dorkitude/embedding-local-cpu-vs-hosted/issues/1

**Completed and audited: 96/96 configurations, six models, 76,704 primary
query evaluations.** Independent code recomputed all reported metrics and
296 confidence intervals. Unit tests, vet, live integration, the expanded
exact-search audit, and the isolated CPU confirmation passed.

## Summary results

I did this on a variety of chunk sizes and dimension sizes. The study swept
**96 configurations**: six models × four dimension settings (128/256/512/native)
× three chunk sizes (128/256/512 words), evaluated on all 799 native MIRACL
English dev queries over a 100,000-passage primary corpus and a nested
20,000-passage chunk-sweep corpus.

The first table below is a summary / rollup that highlights the primary
head-to-head slice; the dimension and chunk sweeps and the per-model strengths
and weaknesses after it synthesize the rest of the matrix.

### Primary rollup: 100k passages, 256-word chunks, native dimensions

Every model at its best-supported configuration, turbopuffer settled-state
search. Sorted by nDCG@10.

| Model | Native dims | nDCG@10 (95% bootstrap CI) | MRR@10 | Recall@10 | Recall@100 | Request p50 (s) | p95 (s) | Embedding usage |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Qwen3-Embedding-8B (Fireworks) | 4096 | **0.8474** [0.8332, 0.8606] | 0.8622 | 0.9598 | 0.9911 | 0.586 | 3.412 | $1.18 |
| Gemini Embedding 2 (OpenRouter) | 3072 | 0.8424 [0.8270, 0.8578] | 0.8680 | 0.9367 | 0.9728 | 1.153 | 1.420 | $2.32 |
| OpenAI text-embedding-3-large | 3072 | 0.8059 [0.7905, 0.8209] | 0.8270 | 0.9389 | 0.9908 | 0.455 | 1.132 | $5.79 |
| Local EmbeddingGemma 2 (8 vCPU, no GPU) | 768 | 0.8002 [0.7836, 0.8160] | 0.8320 | 0.9134 | 0.9567 | 2.214 | 5.229 | $0 API* |
| BGE-M3 (OpenRouter) | 1024 | 0.7975 [0.7816, 0.8137] | 0.8318 | 0.9124 | 0.9642 | 0.971 | 1.754 | $0.12 |
| OpenAI text-embedding-3-small | 1536 | 0.7719 [0.7555, 0.7879] | 0.7886 | 0.9210 | 0.9860 | 0.367 | 0.730 | $0.89 |

\*Local EmbeddingGemma 2 paid no API token fees but consumed VM CPU time
(unpriced): 13.02 distinct inputs/s end-to-end, 1,389 model tokens/s warm
inference, 157.4 minutes for its full 16-treatment matrix. Request latencies
are observed end-to-end values including network, batching, queues and
differing batch sizes — not a controlled latency ranking. Usage figures are
study totals at list prices; OpenAI models ran four paid dimension passes.
Dimension, chunk, exact-search and cost detail: [RESULTS.md](RESULTS.md).

### Dimension robustness: nDCG@10 across the dimension sweep

Same 100k corpus and 256-word chunks as the rollup; only the output dimension
changes. "Retained @128d" is 128d nDCG as a share of native nDCG — the
compression-fragility signal.

| Model | 128d | 256d | 512d | Native | Retained @128d |
|---|---:|---:|---:|---:|---:|
| Qwen3-Embedding-8B | 0.8015 | 0.8254 | 0.8362 | 0.8474 | **94.6%** |
| Gemini Embedding 2 | 0.7445 | 0.8005 | 0.8363 | 0.8424 | 88.4% |
| OpenAI text-embedding-3-large | 0.7272 | 0.7725 | 0.7945 | 0.8059 | 90.2% |
| Local EmbeddingGemma 2 | 0.7389 | 0.7925 | 0.7963 | 0.8002 | 92.3% |
| BGE-M3 | 0.6359 | 0.7389 | 0.7845 | 0.7975 | **79.7%** |
| OpenAI text-embedding-3-small | 0.6932 | 0.7557 | 0.7687 | 0.7719 | 89.8% |

Qwen at 128d (0.8015) nearly matches OpenAI-large at its full 3072 dimensions
(0.8059). BGE's reduced dimensions are exploratory prefixes, not validated MRL;
Gemini's 128/256/512 sit below Google's recommended operating points. Storage
does not shrink by exactly the vector ratio, and API token charges generally do
not decrease with output dimension.

### Chunk sensitivity: nDCG@10 across the chunk sweep (20k corpus, native dims)

| Model | 128 words | 256 words | 512 words | Max spread |
|---|---:|---:|---:|---:|
| Qwen3-Embedding-8B | 0.8580 | 0.8567 | 0.8562 | 0.002 |
| Gemini Embedding 2 | 0.8631 | 0.8632 | 0.8621 | 0.001 |
| OpenAI text-embedding-3-large | 0.8155 | 0.8176 | 0.8164 | 0.002 |
| Local EmbeddingGemma 2 | 0.8186 | 0.8280 | 0.8299 | 0.011 |
| BGE-M3 | 0.8160 | 0.8180 | 0.8174 | 0.002 |
| OpenAI text-embedding-3-small | 0.7894 | 0.7860 | 0.7845 | 0.005 |

Chunk size is a weak lever on these short passages (median 53 words): every
model stays within ~0.01 nDCG across the sweep, and Gemma — the most sensitive
— *gains* slightly with bigger chunks. Compare chunk scores only within this
20k table; the 100k primary corpus is a different, harder distractor set.

### Strengths and weaknesses across all 96 cells

- **Qwen3-Embedding-8B** — strength: highest nDCG@10 at *every* dimension
  tested, and the most compression-robust (94.6% retained at 128d); 512d is
  the standout efficiency configuration (0.8362 at one eighth of the native
  vector payload). Weakness: the largest native vectors (4096d storage) and
  the slowest hosted tail latency (p95 3.412s).
- **Gemini Embedding 2** — strength: second-highest native quality, best
  MRR@10 (0.8680), essentially zero chunk sensitivity. Weakness: quality drops
  sharply below Google's recommended operating points (88.4% retained @128d).
- **OpenAI text-embedding-3-large** — strength: the most ANN-friendly (only
  0.002 nDCG lost to approximate search on the exact-cosine audit), top-tier
  Recall@100 (0.9908), fast (p50 0.455s). Weakness: middling quality per
  dollar — the most expensive model in the study ($5.79 usage, four paid
  dimension passes).
- **Local EmbeddingGemma 2** — strength: within 0.006 nDCG of OpenAI-large
  (paired difference -0.0056 [-0.0207, 0.0088]; not proven equivalent) at zero
  API token cost, with graceful degradation to 128d (92.3%). Weakness: slowest
  lane (p50 2.214s; 13.02 distinct inputs/s), weakest Recall@100 (0.9567), and
  the largest ANN approximation loss on the exact audit (0.029 nDCG).
- **BGE-M3** — strength: cheapest hosted option ($0.12 usage) with respectable
  native quality. Weakness: its unsupported 128d prefix collapses (79.7%
  retained) — do not generalize MRL compression to arbitrary families.
- **OpenAI text-embedding-3-small** — strength: fastest (p50 0.367s) and
  cheap. Weakness: the lowest quality at every dimension tested.

Cross-cutting: ANN approximation losses are uneven (Qwen's ANN even scored
slightly *above* exact cosine on the audit subset — exact ranking is not an
upper bound on judged relevance), so scores describe end-to-end retrieval
configurations, not bare models. Total observed cost: ~$10.31 embedding usage
plus ~$13.52 turbopuffer metering; retained primary namespaces run about
$4.25/month.

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

## License

Original code, configurations, and documentation: [CC0 1.0 Universal](LICENSE)
(public domain dedication; no rights reserved). Incorporated third-party
materials keep their original licenses: MIRACL contributions under Apache-2.0
and Wikipedia-derived corpus selections under CC BY-SA 3.0. See [NOTICE](NOTICE)
and [receipts/](receipts/) for attribution and license receipts.

Reproduction and recovery commands are in [RUNBOOK.md](RUNBOOK.md). The primary
harness uses Go/Cobra; Python via uv is confined to local inference/benchmarking.
Tests: `go test ./...` and `go vet ./...`. Offline metric verification from the
committed records: `go run ./cmd/verify`. This independent verifier does not
import the harness evaluator or require provider credentials.
