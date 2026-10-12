---
title: Local CPU versus hosted embedding retrieval
type: research-report
status: partial
created: 2026-10-12
updated: 2026-10-12
related: ["[[STUDY_PLAN]]", "[[CORPUS_CARD]]", "[[MODEL_CARD]]"]
---

Owner: Kyle Wild. [Repository](https://github.com/dorkitude/embedding-gpu-vs-local), [execution issue](https://github.com/dorkitude/embedding-gpu-vs-local/issues/1).

8 of 96 full-matrix cells passed. Run ID: r20261012a. All reported cells use 799 native MIRACL English dev queries and turbopuffer ANN retrieval. Primary corpus: 100,000 passages; chunk sweep: a nested 20,000-passage corpus. No GPU hardware was measured.

## Primary results: 100k passages, 256-word chunks, native dimensions

| Model | Native dimensions | nDCG@10 (95% bootstrap CI) | MRR@10 | Recall@5 | Recall@10 | Recall@100 |
|---|---:|---:|---:|---:|---:|---:|
| google/embeddinggemma-2 | 768 | pending | | | | |
| text-embedding-3-large | 3072 | pending | | | | |
| accounts/fireworks/models/qwen3-embedding-8b | 4096 | pending | | | | |
| text-embedding-3-small | 1536 | pending | | | | |
| google/gemini-embedding-2 | 3072 | pending | | | | |
| baai/bge-m3 | 1024 | pending | | | | |

## Primary dimension sweep

| Model | 128d nDCG@10 | 256d | 512d | Native |
|---|---:|---:|---:|---:|
| gemma2 | pending | pending | pending | pending |
| openai-large | 0.7387 | 0.7852 | pending | pending |
| qwen8b | pending | pending | pending | pending |
| openai-small | pending | pending | pending | pending |
| gemini2 | pending | pending | pending | pending |
| bge | pending | pending | pending | pending |

BGE reduced dimensions are exploratory prefixes, not validated MRL. Gemini's 128/256/512 treatments are below Google's recommended 768/1536/3072 operating points, although its MRL representation permits truncation. Google Gemma/Qwen use supported MRL; OpenAI dimensions are separately requested from its API. Native vectors are cached for MRL/prefix views and L2-normalized after truncation.

## Secondary chunk sweep: 20k passages only, native dimensions

| Model | 128 words | 256 words | 512 words |
|---|---:|---:|---:|
| gemma2 | pending | pending | pending |
| openai-large | pending | pending | pending |
| qwen8b | pending | pending | pending |
| openai-small | pending | pending | pending |
| gemini2 | pending | pending | pending |
| bge | pending | pending | pending |

Compare chunk sizes only within this 20k table. Primary-to-secondary changes also change distractors and cannot identify a chunk-size effect. The unit is whitespace-delimited words, not provider tokenizer tokens; natural MIRACL passages are often shorter than a target, so many treatment inputs coincide. The 12.5% overlap and a conservative common 6000-byte title/text ceiling are identical across models.

## Timing, spend and storage

Successful embedding usage totals an estimated $3.5145. Turbopuffer metered writes/queries/returns add $0.5429. Retaining the study namespaces is approximately $0.0914/month (conservative logical-byte estimate). These are list-price estimates, not an invoice; see [machine-readable costs](results/costs.json), request receipts and the pre-run budget.

| Model | Embedding requests | Prompt tokens | Observed request p50 seconds | p95 seconds | Successful usage USD |
|---|---:|---:|---:|---:|---:|
| gemma2 | 734 | 921603 | 0.799 | 2.165 | 0.0000 |
| openai-large | 4386 | 25296676 | 0.393 | 0.618 | 3.2886 |
| qwen8b | 725 | 852627 | 0.588 | 3.265 | 0.0853 |
| openai-small | 2 | 36 | 0.199 | 0.199 | 0.0000 |
| gemini2 | 615 | 703227 | 1.147 | 1.770 | 0.1406 |
| bge | 2 | 44 | 0.694 | 0.694 | 0.0000 |

Observed API latency includes network, batching, queues and serialization, and differs in batch sizes; it is not controlled GPU execution or a direct latency ranking. Cache hits are excluded from request timings. Local CPU measurements include exact tokenization and serialization at the sidecar; standalone batch/precision/thread experiments are in results/cpu-benchmark.json and results/cpu-thread-scaling.json. Cold model initialization uses locally cached files, not cold physical disk. Parallel hosted work shares the VM, so CPU throughput is contextual.

## Method, uncertainty and audit

The deterministic SHA-256 sample scans all 32,893,221 English passages, retaining all 7,921 unique dev-judged documents (positive and negative), then hash-selected distractors. This deliberately enriches judged/hard documents and removes most original distractors. It is not the official full-corpus MIRACL benchmark. Unjudged documents count as nonrelevant; incomplete judgments can penalize useful results. All 799 dev queries are included; no evaluation-score-based tuning was performed. The smoke pilot lacks complete qrels coverage and is excluded.

Chunks map to original passage IDs. ANN retrieves at least 500 chunks, increases overfetch until 100 unique passages are obtained, and ranks each passage by its best chunk. Metrics measure passage retrieval, not answer-bearing chunk relevance. Every query retains its top 100 doc IDs. Recall is macro-averaged; micro Recall@100 is also retained. Binary MIRACL nDCG normalizes by the ideal top min(10, number relevant).

All confidence intervals use 2,000 paired bootstrap samples, fixed seed 20261012 and the same sorted query order. results/bootstrap.json includes reduced-minus-native, model-native-minus-OpenAI-large-native, and chunk-minus-256 comparisons within the same corpus. Intervals are exploratory and unadjusted for multiple comparisons. They cover query sampling uncertainty, not provider drift or corpus-sampling variation.

Eight fixed queries per cell have exhaustive local float64 dot-product cosine audits over all normalized chunk vectors, followed by identical doc deduplication. Exact and ANN rankings/metrics are preserved under results/exact. This diagnoses ANN effects but does not exhaustively guarantee ANN accuracy for all 799 queries. Namespace row counts are exact Count aggregations; provider metadata counts/bytes are approximate and may lag ingestion. Vector dimensions are checked against returned vectors and namespace schema.

Reproduction: [RUNBOOK](RUNBOOK.md). Frozen corpus/chunk manifests, full attribution and license receipts accompany the recipe. Raw corpus text, weights, caches and vectors stay uncommitted. Hosted providers do not expose immutable weight revisions; model names and returned model IDs/timestamps are recorded, so perfect future hosted reproducibility is unavailable. Original EmbeddingGemma 300M remains HTTP 403 gated and is not measured. Local Qwen0.6B was deferred to bound CPU time.

Final audit artifacts: results/audit.json, results/independent-audit.json, results/bootstrap.json and COMPLETION.json. The independent verifier separately recomputes headline metrics from ranked document IDs and qrels; it does not import the evaluator. Future GPU work requires a controlled GPU host, pinned runtime/weights and matched batching.
