---
title: Execution ledger
type: research-ledger
status: active
created: 2026-10-12
updated: 2026-10-12
related: ["[[STUDY_PLAN]]", "[[RUNBOOK]]"]
---
- 2026-10-12: Repository initially contains AGENTS.md only. Existing issue #1 is
  operational source of truth. Verified 8 vCPUs, 31 GiB RAM, no GPU.
- 2026-10-12: License review completed before corpus acquisition. Pinned MIRACL
  revisions; retain underlying Wikipedia CC BY-SA and attribution.
- 2026-10-12: Precommitted bounded design uses allowed 100k primary / 20k
  secondary compromise to make CPU inference and provider spend practical.
- 2026-10-12T20:44Z: Frozen corpus has 100,000 passages including 7,921 unique
  judged documents; source scan verified 32,893,221 passages. All 799 queries.
  Input manifests frozen before metrics. The byte safety ceiling is 6,000 bytes
  including title, with extra room for prefixes/special tokens; whitespace chunk
  unit and 12.5% overlap are held constant for every model.
- 2026-10-12T20:45Z: All six model preflights pass. Fireworks rejects the optional
  truncate=false argument, so omit it and enforce a common conservative byte
  ceiling locally. Qwen price page says $0.10/M; reserve at $0.15/M conservatively.
  First 1,000-document smoke is plumbing only: it does not retain all judged
  documents and is excluded from comparisons.
- 2026-10-12T20:46Z: BEFORE full run: embedding upper-bound estimate $50.7911,
  20% retries = $60.9494. Reserve $30 for turbopuffer ($2/GB writes, up to 50%
  batch discount; $0.001/TB scanned, minimum 1.28GB/query; $0.05/GB returned;
  $0.33/GB-month storage). Public pricing calculator JS captured as receipt.
  ~12 GB raw vectors over matrix => <=$24 writes before batch discounts, plus
  <=$4 monthly retained storage and < $1 queries. Total estimate < $92.
  Hard conservative embedding reservation limit $120, leaving $30 for storage.
  Existing account plan minimums are not incremental study usage.
- 2026-10-12T20:52Z BEFORE Google full run: choose supported CPU bfloat16,
  8 PyTorch threads, inference batch8, HTTP batch32, based on fixed synthetic
  225-token input (not retrieval labels). ~10–12 docs/s BF16 vs ~5–7 FP32.
  Cosine to FP32 >0.99995, all finite. Native AVX512_BF16 is present.
  CPU thread sweep 1/2/4/8 favors8. Float32 pilot remains a separate smoke run;
  cache keys add runtime/precision/batch tag so its vectors cannot leak into full run.
- 2026-10-12T20:52Z BEFORE Gemini full run: official current Gemini2 documentation
  requires task instructions in TEXT, not the old task_type field. Corrected to
  task: search result | query and title: ... | text, identical to Google Gemma.
  OpenRouter input_type removed to avoid undocumented adapter behavior. Hosted
  Gemini2 has not run its matrix yet. 128/256/512 are exploratory MRL sizes below
  Google's recommended 768/1536/3072 (not claimed recommended dimensions).
- 2026-10-12T20:56Z: Corrected the two runtime-decision timestamps above from
  hand-entered 20:55 to 20:52, using commit 21b8151 at 20:52:57Z as evidence.
  The full local cell started at 20:53:08Z; decisions preceded it. Other primary
  timestamps are captured directly in request/cell/manifests, not inferred.
- 2026-10-12: Scheduling-only revision: split hosted jobs by provider
  (OpenAI large/small; Fireworks Qwen; OpenRouter Gemini/BGE), concurrent with
  local CPU. Existing OpenAI job stopped and resumed through checkpoints to
  deploy the atomic shared budget lock before starting simultaneous paid lanes.
  Up to4 requests per hosted lane. Memory estimate <20GiB of31GiB. The fixed
  corpus, prompts, dimensions and evaluation are unchanged; observed timings
  include shared VM contention. OpenRouter smoke ID pilot2 uses corrected Gemini
  prompts. No quality metrics motivated this scheduling revision.
- 2026-10-12: Post-freeze diagnostics (no input changes) show 2,326 positive and
  6,024 negative judgments; all 799 queries retained. No byte-ceiling contractions.
  Primary 256-word treatment has 101,192 chunks, median 53 words. Secondary chunk
  counts 128/256/512 are 24,762/20,421/20,027. These distributions limit conclusions
  about longer-document chunking.
- 2026-10-12: Captured official historical Wikimedia Terms oldid=98524, applicable
  2014–2023, confirming CC BY-SA 3.0 for the 2019 source dump. This supplements the
  initial license review with historical primary-source evidence.
- 2026-10-12: Retrieval audit amendment BEFORE second-pass queries. API receipts
  showed fresh queries scanning substantial unindexed tails. An older20k/128d
  namespace still reports20,421 exhaustive rows after30minutes despite current
  metadata saying up-to-date. Therefore zero exhaustive fallback is not a
  defensible universal readiness assumption. No undocumented force-index API
  will be used. Primary settled-v1 pass uses the SAME cached vectors/namespaces,
  strong ANN operator, metadata up-to-date, last-write age>=60s and zero or three
  stable exhaustive counts5s apart (maximum150s wait). Every primary query
  records full performance counters. Persistent exhaustive/mixed execution is
  labeled explicitly. Original fresh-ingest cells/rankings/exact audits remain
  under results/fresh-ingest; this is an indexing-state correction, not tuning
  based on relevance scores. Additional query cost remains well within budget.
