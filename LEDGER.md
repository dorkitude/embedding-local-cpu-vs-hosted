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
