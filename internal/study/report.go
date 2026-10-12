package study

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func Report() error {
	files, e := filepath.Glob("results/cells/r20261012a-*.json")
	if e != nil {
		return e
	}
	cells := []Cell{}
	records := map[string][]Record{}
	by := map[string]Cell{}
	for _, p := range files {
		var c Cell
		b, e := os.ReadFile(p)
		if e != nil {
			return e
		}
		if e = json.Unmarshal(b, &c); e != nil {
			return e
		}
		cells = append(cells, c)
		by[c.ID] = c
		if c.Status == "passed" {
			rs, e := ReadRecords("results/queries/" + c.ID + ".jsonl")
			if e != nil {
				return e
			}
			records[c.ID] = rs
		}
	}
	sort.Slice(cells, func(i, j int) bool { return cells[i].ID < cells[j].ID })
	pairs := []map[string]any{}
	absolute := map[string]any{}
	for _, c := range cells {
		if c.Status != "passed" {
			continue
		}
		rs := records[c.ID]
		zeros := make([]Record, len(rs))
		for i := range rs {
			zeros[i].QID = rs[i].QID
		}
		absolute[c.ID] = Bootstrap(rs, zeros)
		var native int
		for _, m := range Models {
			if m.ID == c.Model {
				native = m.Native
			}
		}
		ref := fmt.Sprintf("%s-%s-n%d-c%d-d%d", c.Run, c.Model, c.Documents, c.ChunkSize, native)
		if b, ok := records[ref]; ok && ref != c.ID {
			pairs = append(pairs, map[string]any{"a": c.ID, "b": ref, "kind": "dimension-minus-native", "bootstrap": Bootstrap(rs, b)})
		}
		if c.Dimension != native && c.Model != "openai-large" {
			ref = fmt.Sprintf("%s-openai-large-n%d-c%d-d%d", c.Run, c.Documents, c.ChunkSize, c.Dimension)
			if b, ok := records[ref]; ok {
				pairs = append(pairs, map[string]any{"a": c.ID, "b": ref, "kind": "model-minus-openai-large-same-dimension", "bootstrap": Bootstrap(rs, b)})
			}
		}
		if c.Dimension == native && c.Model != "openai-large" {
			ref = fmt.Sprintf("%s-openai-large-n%d-c%d-d3072", c.Run, c.Documents, c.ChunkSize)
			if b, ok := records[ref]; ok {
				pairs = append(pairs, map[string]any{"a": c.ID, "b": ref, "kind": "model-native-minus-openai-large-native", "bootstrap": Bootstrap(rs, b)})
			}
		}
		if c.Documents == 20000 && c.ChunkSize != 256 {
			ref = fmt.Sprintf("%s-%s-n20000-c256-d%d", c.Run, c.Model, c.Dimension)
			if b, ok := records[ref]; ok {
				pairs = append(pairs, map[string]any{"a": c.ID, "b": ref, "kind": "chunk-minus-256-same-20k-corpus", "bootstrap": Bootstrap(rs, b)})
			}
		}
	}
	if e = Save("results/matrix.json", cells); e != nil {
		return e
	}
	Save("results/bootstrap.json", map[string]any{"absolute_ndcg95": absolute, "paired": pairs, "seed": 20261012, "samples": 2000, "warning": "unadjusted exploratory intervals; multiple comparisons; no test-set tuning"})
	// Token and latency accounting from each real request, including failures and retries.
	type Spend struct {
		Requests, Errors                   int
		Tokens                             float64
		EstimatedUSD, ReservedUSD, Seconds float64
		Latencies                          []float64 `json:"-"`
		P50, P95                           float64
	}
	costs := map[string]*Spend{}
	tpBilling := map[string]float64{}
	total := 0.0
	e = Scan("results/requests.jsonl", func(b []byte) error {
		var r struct {
			Tag      string  `json:"tag"`
			Seconds  float64 `json:"seconds"`
			Status   int     `json:"http_status"`
			Reserved float64 `json:"reserved_usd"`
			Usage    struct {
				Tokens float64  `json:"prompt_tokens"`
				Cost   *float64 `json:"cost"`
			} `json:"usage"`
			Billing map[string]float64 `json:"billing"`
		}
		if e := json.Unmarshal(b, &r); e != nil {
			return e
		}
		if strings.HasPrefix(r.Tag, "embed:") {
			f := strings.Split(r.Tag, ":")
			id := f[1]
			c := costs[id]
			if c == nil {
				c = &Spend{}
				costs[id] = c
			}
			c.Requests++
			c.ReservedUSD += r.Reserved
			c.Seconds += r.Seconds
			c.Latencies = append(c.Latencies, r.Seconds)
			if r.Status != 200 {
				c.Errors++
				return nil
			}
			c.Tokens += r.Usage.Tokens
			price := 0.0
			for _, m := range Models {
				if m.ID == id {
					price = m.Price
					if id == "qwen8b" {
						price = .10
					}
				}
			}
			if r.Usage.Cost != nil {
				c.EstimatedUSD += *r.Usage.Cost
			} else {
				c.EstimatedUSD += r.Usage.Tokens * price / 1e6
			}
		} else if strings.HasPrefix(r.Tag, "tp:") {
			for k, v := range r.Billing {
				tpBilling[k] += v
			}
		}
		return nil
	})
	if e != nil {
		return e
	}
	for _, c := range costs {
		sort.Float64s(c.Latencies)
		c.P50 = c.Latencies[len(c.Latencies)/2]
		c.P95 = c.Latencies[int(math.Ceil(float64(len(c.Latencies))*.95))-1]
		total += c.EstimatedUSD
	}
	// API billing already applies write discounts. Decimal GB/TB units from official calculator.
	tpWrite := tpBilling["billable_logical_bytes_written"] / 1e9 * 2
	tpQuery := tpBilling["billable_logical_bytes_queried"] / 1e12 * .001
	tpReturned := tpBilling["billable_logical_bytes_returned"] / 1e9 * .05
	var storage float64
	for _, c := range cells {
		if c.Status == "passed" {
			storage += float64(c.Chunks * (c.Dimension*4 + 64))
		}
	}
	spend := map[string]any{"embedding_by_model": costs, "embedding_success_estimate_usd": total, "turbopuffer_billing_bytes": tpBilling, "turbopuffer_write_usd": tpWrite, "turbopuffer_query_usd": tpQuery, "turbopuffer_return_usd": tpReturned, "retained_storage_conservative_gb": storage / 1e9, "retained_storage_usd_per_month": storage / 1e9 * .33, "total_usage_estimate_excluding_storage_usd": total + tpWrite + tpQuery + tpReturned, "limitations": "List-price estimate; excludes account minimums, taxes, unknown failed-call provider billing and manual discovery calls (<$0.01). Conservative reservation counts every retry separately. Storage billed over retention time, not fixed experiment expense."}
	Save("results/costs.json", spend)
	var out strings.Builder
	passed := 0
	for _, c := range cells {
		if c.Status == "passed" {
			passed++
		}
	}
	status := "partial"
	if passed == 96 {
		status = "complete-pending-independent-audit"
	}
	fmt.Fprintf(&out, "---\ntitle: Local CPU versus hosted embedding retrieval\ntype: research-report\nstatus: %s\ncreated: 2026-10-12\nupdated: %s\nrelated: [\"[[STUDY_PLAN]]\", \"[[CORPUS_CARD]]\", \"[[MODEL_CARD]]\"]\n---\n\nOwner: Kyle Wild. [Repository](https://github.com/dorkitude/embedding-gpu-vs-local), [execution issue](https://github.com/dorkitude/embedding-gpu-vs-local/issues/1).\n\n%d of 96 full-matrix cells passed. Run ID: r20261012a. All reported cells use 799 native MIRACL English dev queries and turbopuffer ANN retrieval. Primary corpus: 100,000 passages; chunk sweep: a nested 20,000-passage corpus. No GPU hardware was measured.\n\n", status, time.Now().UTC().Format("2006-01-02"), passed)
	out.WriteString("## Primary results: 100k passages, 256-word chunks, native dimensions\n\n| Model | Native dimensions | nDCG@10 (95% bootstrap CI) | MRR@10 | Recall@5 | Recall@10 | Recall@100 |\n|---|---:|---:|---:|---:|---:|---:|\n")
	for _, m := range Models {
		id := fmt.Sprintf("r20261012a-%s-n100000-c256-d%d", m.ID, m.Native)
		c, ok := by[id]
		if !ok || c.Status != "passed" {
			fmt.Fprintf(&out, "| %s | %d | pending | | | | |\n", m.Name, m.Native)
			continue
		}
		ci := absolute[id].(map[string]any)["ci95"].([]float64)
		fmt.Fprintf(&out, "| %s | %d | %.4f [%.4f, %.4f] | %.4f | %.4f | %.4f | %.4f |\n", m.Name, m.Native, c.Metrics["ndcg10"], ci[0], ci[1], c.Metrics["mrr10"], c.Metrics["recall5"], c.Metrics["recall10"], c.Metrics["recall100"])
	}
	out.WriteString("\n## Primary dimension sweep\n\n| Model | 128d nDCG@10 | 256d | 512d | Native |\n|---|---:|---:|---:|---:|\n")
	for _, m := range Models {
		fmt.Fprintf(&out, "| %s", m.ID)
		for _, d := range []int{128, 256, 512, m.Native} {
			id := fmt.Sprintf("r20261012a-%s-n100000-c256-d%d", m.ID, d)
			if c, ok := by[id]; ok && c.Status == "passed" {
				fmt.Fprintf(&out, " | %.4f", c.Metrics["ndcg10"])
			} else {
				out.WriteString(" | pending")
			}
		}
		out.WriteString(" |\n")
	}
	out.WriteString("\nBGE reduced dimensions are exploratory prefixes, not validated MRL. Gemini's 128/256/512 treatments are below Google's recommended 768/1536/3072 operating points, although its MRL representation permits truncation. Google Gemma/Qwen use supported MRL; OpenAI dimensions are separately requested from its API. Native vectors are cached for MRL/prefix views and L2-normalized after truncation.\n\n## Secondary chunk sweep: 20k passages only, native dimensions\n\n| Model | 128 words | 256 words | 512 words |\n|---|---:|---:|---:|\n")
	for _, m := range Models {
		fmt.Fprintf(&out, "| %s", m.ID)
		for _, size := range []int{128, 256, 512} {
			id := fmt.Sprintf("r20261012a-%s-n20000-c%d-d%d", m.ID, size, m.Native)
			if c, ok := by[id]; ok && c.Status == "passed" {
				fmt.Fprintf(&out, " | %.4f", c.Metrics["ndcg10"])
			} else {
				out.WriteString(" | pending")
			}
		}
		out.WriteString(" |\n")
	}
	out.WriteString("\nThe primary 100k corpus produces 101,192 chunks at target256 (median53 words). The secondary corpus produces 24,762/20,421/20,027 chunks at targets128/256/512. No chunk was shortened by the common byte ceiling; maximum title-plus-text size was3559 UTF-8 bytes. See results/token-overflow-audit.json. Compare chunk sizes only within this 20k table. Primary-to-secondary changes also change distractors and cannot identify a chunk-size effect. The unit is whitespace-delimited words, not provider tokenizer tokens; natural MIRACL passages are often shorter than a target, so many treatment inputs coincide. The 12.5% overlap and a conservative common 6000-byte title/text ceiling are identical across models.\n\n## Timing, spend and storage\n\n")
	fmt.Fprintf(&out, "Successful embedding usage totals an estimated $%.4f. Turbopuffer metered writes/queries/returns add $%.4f. Retaining the study namespaces is approximately $%.4f/month (conservative logical-byte estimate). These are list-price estimates, not an invoice; see [machine-readable costs](results/costs.json), request receipts and the pre-run budget.\n\n", total, tpWrite+tpQuery+tpReturned, storage/1e9*.33)
	out.WriteString("| Model | Embedding requests | Prompt tokens | Observed request p50 seconds | p95 seconds | Successful usage USD |\n|---|---:|---:|---:|---:|---:|\n")
	for _, m := range Models {
		if c := costs[m.ID]; c != nil {
			fmt.Fprintf(&out, "| %s | %d | %.0f | %.3f | %.3f | %.4f |\n", m.ID, c.Requests, c.Tokens, c.P50, c.P95, c.EstimatedUSD)
		}
	}
	out.WriteString("\nStudy embedding totals include FOUR paid embedding passes for each OpenAI model because dimensions are requested explicitly; other models reuse one native embedding pass. These totals are experiment costs, not per-document deployment prices. Source token prices are OpenAI large $0.13/M, small $0.02/M, Qwen $0.10/M, Gemini $0.20/M and BGE $0.01/M. Local CPU has no API token charge but VM time is not free and is not priced here.\n\nObserved API latency includes network, batching, queues and serialization, and differs in batch sizes; it is not controlled GPU execution or a direct latency ranking. Cache hits are excluded from request timings. Local CPU measurements include exact tokenization and serialization at the sidecar; standalone batch/precision/thread experiments are in results/cpu-benchmark.json and results/cpu-thread-scaling.json. Cold model initialization uses locally cached files, not cold physical disk. Parallel hosted work shares the VM, so CPU throughput is contextual.\n\n## Method, uncertainty and audit\n\n")
	out.WriteString("The deterministic SHA-256 sample scans all 32,893,221 English passages, retaining all 7,921 unique dev-judged documents (positive and negative), then hash-selected distractors. This deliberately enriches judged/hard documents and removes most original distractors. It is not the official full-corpus MIRACL benchmark. Unjudged documents count as nonrelevant; incomplete judgments can penalize useful results. All 799 dev queries are included; no evaluation-score-based tuning was performed. The smoke pilot lacks complete qrels coverage and is excluded.\n\n")
	out.WriteString("Chunks map to original passage IDs. ANN retrieves at least 500 chunks, increases overfetch until 100 unique passages are obtained, and ranks each passage by its best chunk. Metrics measure passage retrieval, not answer-bearing chunk relevance. Every query retains its top 100 doc IDs. Recall is macro-averaged; micro Recall@100 is also retained. Binary MIRACL nDCG normalizes by the ideal top min(10, number relevant).\n\n")
	out.WriteString("All confidence intervals use 2,000 paired bootstrap samples, fixed seed 20261012 and the same sorted query order. results/bootstrap.json includes reduced-minus-native, model-native-minus-OpenAI-large-native, and chunk-minus-256 comparisons within the same corpus. Intervals are exploratory and unadjusted for multiple comparisons. They cover query sampling uncertainty, not provider drift or corpus-sampling variation.\n\n")
	out.WriteString("Eight fixed queries per cell have exhaustive local float64 dot-product cosine audits over all normalized chunk vectors, followed by identical doc deduplication. Exact and ANN rankings/metrics are preserved under results/exact. This diagnoses ANN effects but does not exhaustively guarantee ANN accuracy for all 799 queries. Namespace row counts are exact Count aggregations; provider metadata counts/bytes are approximate and may lag ingestion. Vector dimensions are checked against returned vectors and namespace schema.\n\n")
	out.WriteString("Reproduction: [RUNBOOK](RUNBOOK.md). Frozen corpus/chunk manifests, full attribution and license receipts accompany the recipe. Raw corpus text, weights, caches and vectors stay uncommitted. Hosted providers do not expose immutable weight revisions; model names and returned model IDs/timestamps are recorded, so perfect future hosted reproducibility is unavailable. Original EmbeddingGemma 300M remains HTTP 403 gated and is not measured. Local Qwen0.6B was deferred to bound CPU time.\n\n")
	out.WriteString("Final audit artifacts: results/audit.json, results/independent-audit.json, results/bootstrap.json and COMPLETION.json. The independent verifier separately recomputes headline metrics from ranked document IDs and qrels; it does not import the evaluator. Future GPU work requires a controlled GPU host, pinned runtime/weights and matched batching.\n")
	return Atomic("RESULTS.md", []byte(out.String()))
}
