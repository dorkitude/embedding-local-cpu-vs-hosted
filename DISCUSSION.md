---
title: Practical interpretation of the completed embedding study
type: research-discussion
status: complete
created: 2026-10-12
updated: 2026-10-12
related: ["[[RESULTS]]", "[[CORPUS_CARD]]", "[[MODEL_CARD]]"]
---

Owner: Kyle Wild. Run: r20261012a, settled-v1.
[Execution issue](https://github.com/dorkitude/embedding-local-cpu-vs-hosted/issues/1).

All 96 configurations completed on turbopuffer, with 799 judged queries each.
The primary comparison uses 100,000 passages; chunk comparisons use the same
nested 20,000 passages. These are measured retrieval configurations, with
different model representations and different service search behavior.

Qwen3-8B had the highest observed native nDCG@10, 0.8474, followed by Gemini 2
at 0.8424. Their native differences against OpenAI-large were +0.0415
[0.0298, 0.0537] and +0.0365 [0.0233, 0.0511], respectively, using exploratory,
unadjusted paired 95% bootstrap intervals. This does not establish a significant
Qwen-versus-Gemini difference: that pair was not directly tested. Gemini had
slightly higher MRR@10, while Qwen had higher Recall@100.

Qwen at 512 dimensions is a useful configuration to test on new deployment data:
its nDCG was 0.8362 versus 0.8474 at 4096 dimensions, with one eighth of the raw
vector payload. Gemini at 512 dimensions scored 0.8363 versus 0.8424 at 3072,
but 512 is below Google's recommended operating dimensions and is treated here
as exploratory. Total database storage does not shrink by exactly the vector
ratio because IDs and attributes also occupy space. API token charges generally
do not decrease with output dimension.

Local EmbeddingGemma 2 at 768 dimensions scored 0.8002 versus OpenAI-large's
0.8059. The paired nDCG difference was -0.0056 [-0.0207, 0.0088]; this does not
prove equivalence. Recall@100 was lower locally, 0.9567 versus 0.9908. The full
local run processed 110,922 distinct formatted inputs at 13.02 inputs/second
across measured embedding/cache phases, taking 142.0 minutes of those phases
and 157.4 minutes for all initial indexing/evaluation treatments. This is real
CPU inference on eight vCPUs, with BF16 weights and batch 8. VM cost is unpriced;
zero API token fees do not imply free computation. The isolated confirmation
used an explicit original FP32 checkpoint load and retained batch/thread results.

BGE-M3 at native 1024 dimensions scored 0.7975 at a low observed token price,
but its unsupported 128-dimensional prefix fell to 0.6359. Do not generalize
MRL compression behavior to arbitrary embedding families. OpenAI-small scored
0.7719 versus large's 0.8059; both use the same input tokenizer here, with listed
token prices of $0.02/M and $0.13/M. The experiment paid for four dimension passes
per OpenAI model, so aggregate study costs are not single-pass deployment costs.

The exact audits materially limit model-only conclusions. On the common 64-query
diagnostic subset, native Gemma's exact nDCG was 0.7973 versus ANN's 0.7682;
OpenAI-large's corresponding values were 0.8226 and 0.8204. Approximation losses
are uneven. Native Qwen's ANN nDCG on that subset was slightly above exact cosine
(0.8371 versus 0.8323): exact nearest-neighbor ordering is not an upper bound on
judgment-based relevance metrics. These 64 queries diagnose search behavior;
they do not replace the 799-query primary comparison. Some namespaces retained
mixed or exhaustive fallback even after their recorded settled-state checks.

Chunk effects were usually small on these short passages, but not uniformly
absent. For native Gemma on the 20k corpus, 128 minus 256 words was -0.0094
[-0.0153, -0.0032], and 512 minus 256 was +0.0019 [0.0004, 0.0036]. These are
unadjusted exploratory intervals, not a universal chunk-size recommendation.
The primary median is only 53 words; many target sizes produce identical text.
Do not extrapolate to long documents or infer a chunk effect from changing the
100k corpus to the easier 20k corpus. Choosing a deployment setting from this
matrix should be followed by held-out domain evaluation.

Successful embedding usage was approximately $10.31. Discount-adjusted
turbopuffer usage brings the estimate to $23.84, versus a $36.01 ceiling using
undiscounted writes; both exclude retained storage, fixed account minimums,
taxes and any unknown failed-call charges. The conservative embedding reservation
ledger also remains below its $120 cap. Primary namespace retention is about
$4.25/month using the conservative estimate; actual service-reported sizes are
in results/storage-summary.json. Namespaces are retained for reproducibility.

No GPU hardware was controlled or measured. Hosted latencies include network,
queues, batching and provider behavior. The corpus deliberately retains all
judged positives and negatives, excludes most original distractors, and has
incomplete relevance labels. Hosted weights cannot be pinned immutably. These
limits, together with ANN effects, make the study useful for choosing follow-up
configurations rather than declaring a universal embedding-model winner.
