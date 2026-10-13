---
title: Embedding model access discovery
type: model-discovery
status: verified
created: 2026-10-12
updated: 2026-10-12
related: ["[[MODEL_CARD]]", "[[STUDY_PLAN]]", "[[RESULTS]]"]
---
Owner: Kyle Wild. [Execution issue](https://github.com/dorkitude/embedding-local-cpu-vs-hosted/issues/1).

Catalog membership and actual access are different. The harness snapshots
catalogs with study discover and performs real two-input requests with study
preflight. Full corpus outcomes are reported separately in RESULTS.md.

| Model | Route | Actual preflight | Selection |
|---|---|---|---|
| google/embeddinggemma-2 | pinned local CPU sidecar | 768d, two rows, finite | mandatory; selected |
| text-embedding-3-large | OpenAI integration | 3072d, two rows | mandatory; selected |
| accounts/fireworks/models/qwen3-embedding-8b | Fireworks integration | 4096d, two rows | mandatory; selected |
| text-embedding-3-small | OpenAI integration | 1536d, two rows | selected inexpensive OpenAI control |
| google/gemini-embedding-2 | OpenRouter integration | 3072d, two rows | selected current hosted Google family |
| baai/bge-m3 | OpenRouter integration | 1024d, two rows | selected non-MRL open family |
| voyageai/voyage-4-lite | OpenRouter integration | 1024d, two rows | accessible, excluded to bound suite |
| google/embeddinggemma-300m | Hugging Face integration | HTTP403 gated config download | inaccessible; not measured |
| Qwen/Qwen3-Embedding-0.6B | local | not attempted | optional baseline deferred to prioritize full Gemma2 CPU run |

The selected six-model suite was chosen from access, model-family coverage,
dimension support and resource constraints before comparative retrieval metrics.
Voyage exclusion was not based on its retrieval score. Other catalog names,
including E5, NVIDIA and Perplexity models, were not call-tested and are not
claimed accessible or inaccessible.

Recovered preflight problems: the initial local environment referenced a stale
Python installation, then lacked Pillow/torchvision dependencies required by the
multimodal processor even for text-only loading. The working runtime uses pinned
Python3.12.3 and uv.lock packages. Fireworks rejects truncate=false as an extra
parameter, so the harness omits it and enforces common input-length bounds.
Gemini2 uses retrieval instructions in the input text, as required by its current
official documentation; the full run does not rely on the older task_type field.

Discovery receipts: results/discovery.json, results/preflight/models.json,
results/preflight/extra-models.json and receipts/original-gemma-access.txt.
Model instructions, dimensionality, context bounds and revision limitations are
in MODEL_CARD.md. Hosted weight revisions are not exposed by these APIs.
