---
title: Model access, prompts, dimensions and runtime
type: model-card
status: active
created: 2026-10-12
updated: 2026-10-12
related: ["[[STUDY_PLAN]]", "[[RESULTS]]"]
---
Owner: Kyle Wild. Issue: https://github.com/dorkitude/embedding-gpu-vs-local/issues/1

| Model | Runtime | Native d | Reduced dimensions | Retrieval formatting |
|---|---|---:|---|---|
| Google EmbeddingGemma 2 | local CPU | 768 | supported MRL 128/256/512 + L2 | query: task: search result / query; document: title / text |
| OpenAI text-embedding-3-large | OpenAI API | 3072 | API dimensions=128/256/512/native | plain query; title + newline + passage |
| Qwen3-Embedding-8B | Fireworks | 4096 | supported MRL prefix + L2 | official retrieval Instruct / Query; title + passage document |
| OpenAI text-embedding-3-small | OpenAI API | 1536 | API dimensions=128/256/512/native | as large |
| Google Gemini Embedding 2 | OpenRouter | 3072 | MRL prefix + L2 | query: task: search result / query; document: title / text |
| BAAI BGE-M3 | OpenRouter | 1024 | exploratory prefix + L2, NOT validated MRL | no special instruction required; title + passage |

Local revision 914f7f89142e33e77833254d9c9b90c3cef7303b is pinned. Python runtime
dependencies pinned by sidecar/uv.lock. Text-only configuration disables vision
and audio encoders; processor nevertheless needs Pillow and torchvision.
Full run uses bfloat16 (AVX512_BF16 CPU), eight threads and batch8, selected by a fixed-input precision/throughput benchmark before full local retrieval. The separate pilot used float32.
Full model is 740M parameters; text-only is 270M, not 740M text parameters.
Latest official documentation retrieved 2026-10-12 confirms EmbeddingGemma 2.

Hosted model IDs are stable names, NOT pinned weight revisions. Providers do not
expose weight hashes; results are tied to timestamps, endpoint and returned model
strings. Do not imply controlled equality with public weights or GPU hardware.

All planned contexts are >=8192 tokens (BGE 8194, Qwen 32768). Common text plus
title ceiling 6000 UTF-8 bytes, leaving prefix/special-token room, is deliberately
conservative. Sidecar additionally counts exact model tokens without truncation.
Fireworks rejects an unsupported truncate parameter (recorded), so protection is
in the harness. Every API-returned vector is checked for correct dimensions,
finite values and nonzero norm. OpenAI sends every dimension explicitly; no
assumption of API-prefix equivalence. Other models reuse native full vectors.

Accessible exploratory candidates: Voyage 4 Lite actual call succeeds at 1024d.
Chosen extras are Gemini and BGE to span current Google hosted and an older,
non-MRL open family, rather than enumerate every catalog entry. Catalog snapshots
do not establish access. Original EmbeddingGemma 300M is gated; record a fresh
bounded check before final status. Local Qwen0.6B is desirable but not part of
the fixed six-model matrix, given CPU budget.

Observed network/API latencies include provider queueing, network, serialization
and shared-host effects; they are not pure model/GPU execution benchmarks.

Final official OpenAI API review confirms 8192 tokens per input, at most2048
inputs and300,000 summed tokens per request, with dimensions supported for the
text-embedding-3 family. The64-input batches on this frozen corpus are below
that total even under the conservative observed maximum3559-byte input bound.
The larger6000-byte general ceiling alone would not prove a64-input batch safe;
the frozen input diagnostics supply the tighter bound here. Full native sizes
are1536/3072. Receipts: openai-create-embeddings.md and openai-embedding-guide.md.
All four dimension settings were actually requested and verified in this study.
