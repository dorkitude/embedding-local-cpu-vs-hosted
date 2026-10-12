---
title: Local CPU versus hosted embedding retrieval
type: research-report
status: complete
created: 2026-10-12
updated: 2026-10-12
related: ["[[STUDY_PLAN]]", "[[CORPUS_CARD]]", "[[MODEL_CARD]]"]
---

Owner: Kyle Wild. [Repository](https://github.com/dorkitude/embedding-gpu-vs-local), [execution issue](https://github.com/dorkitude/embedding-gpu-vs-local/issues/1).

96 of 96 settled-state primary evaluations passed. Run ID: r20261012a. Original fresh-ingest observations are retained separately under results/fresh-ingest. All reported cells use 799 native MIRACL English dev queries and the turbopuffer ANN operator with service-reported exhaustive fallback. Primary corpus: 100,000 passages; chunk sweep: a nested 20,000-passage corpus. No GPU hardware was measured.

## Primary results: 100k passages, 256-word chunks, native dimensions

| Model | Native dimensions | nDCG@10 (95% bootstrap CI) | MRR@10 | Recall@5 | Recall@10 | Recall@100 |
|---|---:|---:|---:|---:|---:|---:|
| google/embeddinggemma-2 | 768 | 0.8002 [0.7836, 0.8160] | 0.8320 | 0.7666 | 0.9134 | 0.9567 |
| text-embedding-3-large | 3072 | 0.8059 [0.7905, 0.8209] | 0.8270 | 0.7663 | 0.9389 | 0.9908 |
| accounts/fireworks/models/qwen3-embedding-8b | 4096 | 0.8474 [0.8332, 0.8606] | 0.8622 | 0.8204 | 0.9598 | 0.9911 |
| text-embedding-3-small | 1536 | 0.7719 [0.7555, 0.7879] | 0.7886 | 0.7427 | 0.9210 | 0.9860 |
| google/gemini-embedding-2 | 3072 | 0.8424 [0.8270, 0.8578] | 0.8680 | 0.8110 | 0.9367 | 0.9728 |
| baai/bge-m3 | 1024 | 0.7975 [0.7816, 0.8137] | 0.8318 | 0.7598 | 0.9124 | 0.9642 |

## Primary dimension sweep

| Model | 128d nDCG@10 | 256d | 512d | Native |
|---|---:|---:|---:|---:|
| gemma2 | 0.7389 | 0.7925 | 0.7963 | 0.8002 |
| openai-large | 0.7272 | 0.7725 | 0.7945 | 0.8059 |
| qwen8b | 0.8015 | 0.8254 | 0.8362 | 0.8474 |
| openai-small | 0.6932 | 0.7557 | 0.7687 | 0.7719 |
| gemini2 | 0.7445 | 0.8005 | 0.8363 | 0.8424 |
| bge | 0.6359 | 0.7389 | 0.7845 | 0.7975 |

BGE reduced dimensions are exploratory prefixes, not validated MRL. Gemini's 128/256/512 treatments are below Google's recommended 768/1536/3072 operating points, although its MRL representation permits truncation. Google Gemma/Qwen use supported MRL; OpenAI dimensions are separately requested from its API. Native vectors are cached for MRL/prefix views and L2-normalized after truncation.

## Secondary chunk sweep: 20k passages only, native dimensions

| Model | 128 words | 256 words | 512 words |
|---|---:|---:|---:|
| gemma2 | 0.8186 | 0.8280 | 0.8299 |
| openai-large | 0.8155 | 0.8176 | 0.8164 |
| qwen8b | 0.8580 | 0.8567 | 0.8562 |
| openai-small | 0.7894 | 0.7860 | 0.7845 |
| gemini2 | 0.8631 | 0.8632 | 0.8621 |
| bge | 0.8160 | 0.8180 | 0.8174 |

The primary 100k corpus produces 101,192 chunks at target 256 (median 53 words). The secondary corpus produces 24,762/20,421/20,027 chunks at targets 128/256/512. No chunk was shortened by the common byte ceiling; maximum title-plus-text size was 3559 UTF-8 bytes. See results/token-overflow-audit.json. Compare chunk sizes only within this 20k table. Primary-to-secondary changes also change distractors and cannot identify a chunk-size effect. The unit is whitespace-delimited words, not provider tokenizer tokens; natural MIRACL passages are often shorter than a target, so many treatment inputs coincide. The 12.5% overlap and a conservative common 6000-byte title/text ceiling are identical across models.

## Timing, spend and storage

Successful embedding usage totals an estimated $10.3127. Turbopuffer metered writes/queries/returns add $13.5245. Retaining the study namespaces is approximately $4.2454/month (conservative logical-byte estimate). These are list-price estimates, not an invoice; see [machine-readable costs](results/costs.json), request receipts and the pre-run budget.

| Model | Embedding requests | Prompt tokens | Observed request p50 seconds | p95 seconds | Successful usage USD |
|---|---:|---:|---:|---:|---:|
| gemma2 | 3507 | 12074632 | 2.214 | 5.229 | 0.0000 |
| openai-large | 6944 | 44570869 | 0.455 | 1.132 | 5.7942 |
| qwen8b | 3481 | 11807650 | 0.586 | 3.412 | 1.1808 |
| openai-small | 6944 | 44570869 | 0.367 | 0.730 | 0.8914 |
| gemini2 | 3473 | 11622091 | 1.153 | 1.420 | 2.3244 |
| bge | 3481 | 12187564 | 0.971 | 1.754 | 0.1219 |

The local sidecar processed 12074632 model tokens in 8693.0 summed successful HTTP seconds: 1389.0 model tokens/s, including tokenization and HTTP serialization. This includes smoke/preflight work; warm inference dominates. Historical request receipts omit input counts, so a full-run HTTP inputs/s figure cannot be recovered from those receipts. Input-count rates in costs.json cover only requests that explicitly record input counts. CPU lane concurrency is one; API request-summed rates are not aggregate parallel throughput.

Across the completed local matrix, 110922 distinct formatted inputs (110123 document chunks plus799queries) took 8519.1 summed embedding/cache seconds: **13.02 distinct inputs/s**. This end-to-end inference phase includes deduplication, cache I/O, HTTP calls and repeated cache checks, and excludes turbopuffer ingestion/querying. The original BF16 run began with an empty runtime-specific cache; the distinct input count comes from the pre-run manifest. Full initial matrix wall time, including all16 indexing/evaluation treatments, was 157.4 minutes. See results/local-throughput.json.

The post-run isolated confirmation reloads the original pinned checkpoint directly in FP32 before BF16 conversion. See [precision/batch measurements](results/cpu-benchmark-confirmation.json) and [thread scaling](results/cpu-benchmark-confirmation-thread-scaling.json). These preserve the original pre-run receipts separately and correct their dependence on importing the production sidecar. Model initialization reads cached files; it is not a cold physical-disk measurement.

Study embedding totals include FOUR paid embedding passes for each OpenAI model because dimensions are requested explicitly; other models reuse one native embedding pass. These totals are experiment costs, not per-document deployment prices. Source token prices are OpenAI large $0.13/M, small $0.02/M, Qwen $0.10/M, Gemini $0.20/M and BGE $0.01/M. Local CPU has no API token charge but VM time is not free and is not priced here.

Observed API latency includes network, batching, queues and serialization, and differs in batch sizes; it is not controlled GPU execution or a direct latency ranking. Cache hits are excluded from request timings. Local CPU measurements include exact tokenization and serialization at the sidecar; standalone batch/precision/thread experiments are in results/cpu-benchmark.json and results/cpu-thread-scaling.json. Cold model initialization uses locally cached files, not cold physical disk. Parallel hosted work shares the VM, so CPU throughput is contextual.


## Turbopuffer execution mode

Primary queries use the strong-consistency ANN operator after the recorded settled-state check. The provider can still report exhaustive work even when index metadata is up-to-date; these are not uniformly pure ANN searches. Fresh-ingest observations are retained separately. Readiness probes warm the namespace, so these primary query timings are not cold-cache timings.

| Model / corpus / words / dimensions | Queries with zero exhaustive rows | Mixed queries | Fully exhaustive reported | Mean corpus fraction exhaustive |
|---|---:|---:|---:|---:|
| bge / 100000 / 256 / 1024 | 799 | 0 | 0 | 0.00% |
| bge / 20000 / 256 / 128 | 0 | 0 | 799 | 100.00% |
| gemini2 / 100000 / 256 / 3072 | 799 | 0 | 0 | 0.00% |
| gemini2 / 20000 / 256 / 128 | 0 | 0 | 799 | 100.00% |
| gemma2 / 100000 / 256 / 768 | 0 | 799 | 0 | 12.05% |
| gemma2 / 20000 / 256 / 128 | 0 | 0 | 799 | 100.00% |
| openai-large / 100000 / 256 / 3072 | 799 | 0 | 0 | 0.00% |
| openai-large / 20000 / 256 / 128 | 0 | 0 | 799 | 100.00% |
| openai-small / 100000 / 256 / 1536 | 799 | 0 | 0 | 0.00% |
| openai-small / 20000 / 256 / 128 | 0 | 0 | 799 | 100.00% |
| qwen8b / 100000 / 256 / 4096 | 799 | 0 | 0 | 0.00% |
| qwen8b / 20000 / 256 / 128 | 0 | 0 | 799 | 100.00% |

All96 cell-level execution summaries are in results/search-execution.json. The fully-exhaustive category is based on the service counter reaching the logical chunk count; it is not an inferred claim about undocumented internal algorithms. No local database replaces turbopuffer.

## Exact-search and index-state sensitivity

These are end-to-end turbopuffer configurations: model representation and search approximation both affect scores. Eight fixed sorted queries per cell diagnose exact cosine versus settled ANN. A post-hoc, hash-selected common64-query sample additionally covers each model's primary128d and native configuration; it was specified before computing those exact results. Neither diagnostic replaces the799-query primary evaluation.

| Primary configuration | Exact audit queries | Exact nDCG@10 | ANN on same queries | Top10 overlap | Settled minus fresh nDCG (all799) |
|---|---:|---:|---:|---:|---:|
| bge / 1024d | 64 | 0.8114 | 0.7731 | 94.5% | -0.0124 |
| bge / 128d | 64 | 0.7421 | 0.6504 | 84.1% | +0.0000 |
| gemini2 / 128d | 64 | 0.7874 | 0.7489 | 87.5% | +0.0000 |
| gemini2 / 3072d | 64 | 0.8455 | 0.8392 | 95.0% | -0.0042 |
| gemma2 / 128d | 64 | 0.7542 | 0.7194 | 89.5% | +0.0000 |
| gemma2 / 768d | 64 | 0.7973 | 0.7682 | 90.8% | +0.0000 |
| openai-large / 128d | 64 | 0.7293 | 0.6924 | 90.8% | -0.0114 |
| openai-large / 3072d | 64 | 0.8226 | 0.8204 | 98.6% | -0.0008 |
| openai-small / 128d | 64 | 0.7078 | 0.6826 | 90.3% | +0.0000 |
| openai-small / 1536d | 64 | 0.7535 | 0.7479 | 99.2% | -0.0011 |
| qwen8b / 128d | 64 | 0.7652 | 0.7376 | 96.4% | +0.0000 |
| qwen8b / 4096d | 64 | 0.8323 | 0.8371 | 98.0% | -0.0002 |

All cell-level differences are in results/search-audit-summary.json. Exact top10 overlap describes ranking agreement, not relevance recall. The64-query diagnostic is too small to establish population-wide ANN accuracy; conclusions apply to the measured model/dimension/index configuration. Fresh-versus-settled differences use identical vectors and demonstrate indexing-state sensitivity.

## Method, uncertainty and audit

The deterministic SHA-256 sample scans all 32,893,221 English passages, retaining all 7,921 unique dev-judged documents (positive and negative), then hash-selected distractors. This deliberately enriches judged/hard documents and removes most original distractors. It is not the official full-corpus MIRACL benchmark. Unjudged documents count as nonrelevant; incomplete judgments can penalize useful results. All 799 dev queries are included; no evaluation-score-based tuning was performed. The smoke pilot lacks complete qrels coverage and is excluded.

Chunks map to original passage IDs. ANN retrieves at least 500 chunks, increases overfetch until 100 unique passages are obtained, and ranks each passage by its best chunk. Metrics measure passage retrieval, not answer-bearing chunk relevance. Every query retains its top 100 doc IDs. Recall is macro-averaged; micro Recall@100 is also retained. Binary MIRACL nDCG normalizes by the ideal top min(10, number relevant).

All confidence intervals use 2,000 paired bootstrap samples, fixed seed 20261012 and the same sorted query order. results/bootstrap.json includes reduced-minus-native, model-native-minus-OpenAI-large-native, and chunk-minus-256 comparisons within the same corpus. Intervals are exploratory and unadjusted for multiple comparisons. They cover query sampling uncertainty, not provider drift or corpus-sampling variation.

Eight fixed queries per cell have exhaustive local float64 dot-product cosine audits over all normalized chunk vectors, followed by identical doc deduplication. Exact and ANN rankings/metrics are preserved under results/exact. This diagnoses ANN effects but does not exhaustively guarantee ANN accuracy for all 799 queries. Namespace row counts are exact Count aggregations; provider metadata counts/bytes are approximate and may lag ingestion. Vector dimensions are checked against returned vectors and namespace schema.

Reproduction: [RUNBOOK](RUNBOOK.md). Frozen corpus/chunk manifests, full attribution and license receipts accompany the recipe. Raw corpus text, weights, caches and vectors stay uncommitted. Training-set overlap with this public benchmark is not independently verifiable. Hosted providers do not expose immutable weight revisions; model names and returned model IDs/timestamps are recorded, so perfect future hosted reproducibility is unavailable. Original EmbeddingGemma 300M remains HTTP 403 gated and is not measured. Local Qwen0.6B was deferred to bound CPU time.

Final audit artifacts: results/audit.json, results/independent-audit.json, results/bootstrap.json and COMPLETION.json. The independent verifier separately recomputes headline metrics from ranked document IDs and qrels; it does not import the evaluator. Future GPU work requires a controlled GPU host, pinned runtime/weights and matched batching.
