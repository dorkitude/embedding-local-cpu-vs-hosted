---
title: Local CPU versus hosted embedding retrieval
type: study-plan
status: preregistered
created: 2026-10-12
updated: 2026-10-12
related: ["[[CORPUS_CARD]]", "[[LEDGER]]", "[[RESULTS]]"]
---
Owner: Kyle Wild. Issue: https://github.com/dorkitude/embedding-local-cpu-vs-hosted/issues/1

Primary 100,000 MIRACL English passages; all 799 dev queries. Secondary 20,000
nested passages for chunk sweeps. Use 256 whitespace-delimited token-equivalents
primary; 128/256/512 secondary with 12.5% overlap. The unit is explicitly NOT a
model token. Include title in every document input and model-specific retrieval
prefixes. Check provider context constraints and never silently truncate.
No tuning on observed evaluation outcomes.

Mandatory: local CPU Google EmbeddingGemma 2, OpenAI text-embedding-3-large,
Fireworks Qwen3-Embedding-8B if callable. Also OpenAI small and two other
accessible families. Preflight actual requests before choosing extras.
Four dimensions 128/256/512/native. MRL prefix + normalization only where supported;
other prefixes labeled exploratory. OpenAI dimensions requested from API.
CPU host: 8 vCPUs AMD EPYC 9554P, 31 GiB RAM, no GPU. Hosted wall time is observed
API time, never a controlled GPU benchmark.

Budget maximum $150 estimated provider usage, including conservative retry
allowance and turbopuffer. Freeze actual estimates once corpus counts and model
prices are known, BEFORE launching matrix. Request retries at most 3 with timeouts.
Cache/resume native embeddings; terminal per-cell outcomes; no endless paid loop.
Metrics: doc-deduplicated nDCG@10, MRR@10, Recall@5/10/100; retain ranked doc IDs
and per-query metrics. Paired 2,000 bootstrap samples, seed 20261012, same queries.
Strong-consistency turbopuffer ANN with adaptive overfetch up to 10,000 chunks
if needed for 100 unique documents; independent exact cosine audits for fixed
query subset. Freeze plan/manifest before metrics.
