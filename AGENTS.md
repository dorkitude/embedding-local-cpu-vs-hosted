# Embedding Local CPU vs Hosted

Owner: Kyle Wild (dorkitude). Public research study requested 2026-10-12.

Read /home/exedev/a/AGENTS.md and /home/exedev/a/dev/phd/AGENTS.md and README.md. Never read or print secrets. Use exe.dev integrations with injected auth; direct-provider env overrides for portability. Python only via uv. Primary harness is Go/Cobra; local inference sidecars may use Python.

Keep durable research docs Markdown with YAML frontmatter (title, type, status, created/updated dates, related Obsidian links). GitHub issues are operational truth. Commit/push scoped progress and completed results. Do not change unrelated parent workspace files. No fabricated results; label partial/failed/blocked cells honestly. Do not call hosted inference controlled GPU benchmarking; this VM has no GPU. Pin model/runtime/data revisions and record hardware.

Raw model weights, caches, embeddings and huge corpus files are ignored, not committed. Publish only licensed corpus artifacts with attribution/license receipts. Commit compact machine-readable per-query results, manifests, report, reproducible configs and commands. No silent API truncation; model-specific query/document prompts and token limits matter.
