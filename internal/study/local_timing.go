package study

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// LocalTiming combines the frozen distinct-input count with completed initial
// embedding/cache phase timers. It does not invent per-request input counts.
func LocalTiming(cells []Cell) (string, error) {
	var seconds float64
	count := 0
	var start, finish time.Time
	for _, c := range cells {
		if c.Model != "gemma2" || c.Status != "passed" || c.Phase != "settled-v1" {
			continue
		}
		b, e := os.ReadFile("results/fresh-ingest/cells/" + c.ID + ".json")
		if e != nil {
			return "", e
		}
		var fresh Cell
		if e = json.Unmarshal(b, &fresh); e != nil {
			return "", e
		}
		seconds += fresh.EmbedSeconds
		count++
		if start.IsZero() || fresh.Started.Before(start) {
			start = fresh.Started
		}
		if finish.IsZero() || fresh.Finished.After(finish) {
			finish = fresh.Finished
		}
	}
	if count != 16 {
		return "", nil
	}
	var budget struct {
		Models []struct {
			Model  string
			Inputs int `json:"unique_document_inputs"`
		}
	}
	b, e := os.ReadFile("results/budget-estimate.json")
	if e != nil {
		return "", e
	}
	if e = json.Unmarshal(b, &budget); e != nil {
		return "", e
	}
	docInputs := 0
	for _, m := range budget.Models {
		if m.Model == "gemma2" {
			docInputs = m.Inputs
		}
	}
	if docInputs == 0 || seconds <= 0 {
		return "", fmt.Errorf("missing frozen local input/timing evidence")
	}
	requests, tokens := 0, 0.0
	e = ScanJournal("results/requests.jsonl", func(b []byte) error {
		var r struct {
			UTC    time.Time
			Tag    string
			Status int `json:"http_status"`
			Usage  struct {
				Tokens float64 `json:"prompt_tokens"`
			}
		}
		if e := json.Unmarshal(b, &r); e != nil {
			return e
		}
		if strings.HasPrefix(r.Tag, "embed:gemma2:") && !r.UTC.Before(start) && !r.UTC.After(finish) && r.Status == 200 {
			requests++
			tokens += r.Usage.Tokens
		}
		return nil
	})
	if e != nil {
		return "", e
	}
	inputs := docInputs + 799
	receipt := map[string]any{"model": "gemma2", "run": RunID(), "distinct_document_inputs": docInputs, "query_inputs": 799, "distinct_inputs": inputs, "embedding_and_cache_seconds": seconds, "distinct_inputs_per_embedding_cache_second": float64(inputs) / seconds, "initial_lane_wall_seconds": finish.Sub(start).Seconds(), "successful_requests_in_initial_lane": requests, "model_tokens_in_initial_lane": tokens, "start": start, "finish": finish, "method": "Frozen distinct formatted document input count plus799queries divided by sum of initial16cells embedding/cache timers. Includes cache/dedup/I/O overhead and reused-view cache checks; excludes ingestion, querying, exact audits. Original run BF16 cache started empty, separate from FP32 pilot. Not a per-request input-count reconstruction."}
	if e = Save("results/local-throughput.json", receipt); e != nil {
		return "", e
	}
	return fmt.Sprintf("\nAcross the completed local matrix, %d distinct formatted inputs (%d document chunks plus799queries) took %.1f summed embedding/cache seconds: **%.2f distinct inputs/s**. This end-to-end inference phase includes deduplication, cache I/O, HTTP calls and repeated cache checks, and excludes turbopuffer ingestion/querying. The original BF16 run began with an empty runtime-specific cache; the distinct input count comes from the pre-run manifest. Full initial matrix wall time, including all16 indexing/evaluation treatments, was %.1f minutes. See results/local-throughput.json.\n", inputs, docInputs, seconds, float64(inputs)/seconds, finish.Sub(start).Minutes()), nil
}
