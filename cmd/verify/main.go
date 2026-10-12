// Independent audit: deliberately does not import the harness/evaluation package.
package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func read(path string, fn func([]byte) error) error {
	f, e := os.Open(path)
	if os.IsNotExist(e) {
		path += ".gz"
		f, e = os.Open(path)
	}
	if e != nil {
		return e
	}
	defer f.Close()
	var r io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		z, e := gzip.NewReader(f)
		if e != nil {
			return e
		}
		defer z.Close()
		r = z
	}
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 65536), 16<<20)
	for s.Scan() {
		if e = fn(s.Bytes()); e != nil {
			return e
		}
	}
	return s.Err()
}
func main() {
	if e := verify(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func verify() error {
	rel := map[string]map[string]bool{}
	queries := map[string]bool{}
	selected := map[string]bool{}
	secondary := map[string]bool{}
	if e := read("results/manifests/qrels.tsv", func(b []byte) error {
		f := strings.Fields(string(b))
		if len(f) != 4 {
			return fmt.Errorf("qrels format")
		}
		if rel[f[0]] == nil {
			rel[f[0]] = map[string]bool{}
		}
		rel[f[0]][f[2]] = f[3] != "0"
		return nil
	}); e != nil {
		return e
	}
	if e := read("results/manifests/queries.tsv", func(b []byte) error {
		q := strings.SplitN(string(b), "\t", 2)[0]
		if queries[q] {
			return fmt.Errorf("duplicate qid")
		}
		queries[q] = true
		return nil
	}); e != nil {
		return e
	}
	if e := read("results/manifests/attribution.jsonl.gz", func(b []byte) error {
		var a struct {
			ID        string `json:"docid"`
			URL       string `json:"source_url"`
			License   string `json:"license"`
			Secondary bool   `json:"secondary"`
		}
		if e := json.Unmarshal(b, &a); e != nil {
			return e
		}
		if selected[a.ID] || a.URL == "" || a.License != "CC-BY-SA-3.0" {
			return fmt.Errorf("attribution invalid")
		}
		selected[a.ID] = true
		if a.Secondary {
			secondary[a.ID] = true
		}
		return nil
	}); e != nil {
		return e
	}
	if len(selected) != 100000 || len(secondary) != 20000 || len(queries) != 799 {
		return fmt.Errorf("counts %d %d", len(selected), len(queries))
	}
	for _, rs := range rel {
		for id := range rs {
			if !selected[id] || !secondary[id] {
				return fmt.Errorf("judged doc absent")
			}
		}
	}
	runID := os.Getenv("STUDY_RUN_ID")
	if runID == "" {
		runID = "r20261012a"
	}
	files, _ := filepath.Glob("results/cells/" + runID + "-*.json")
	expected := map[string]int{}
	for model, native := range map[string]int{"gemma2": 768, "openai-large": 3072, "qwen8b": 4096, "openai-small": 1536, "gemini2": 3072, "bge": 1024} {
		for _, dim := range []int{128, 256, 512, native} {
			for _, treatment := range [][3]int{{100000, 256, 101192}, {20000, 128, 24762}, {20000, 256, 20421}, {20000, 512, 20027}} {
				expected[fmt.Sprintf("%s-%s-n%d-c%d-d%d", runID, model, treatment[0], treatment[1], dim)] = treatment[2]
			}
		}
	}
	sort.Strings(files)
	results := []map[string]any{}
	independentValues := map[string][]float64{}
	for _, path := range files {
		var cell struct {
			Phase      string         `json:"evaluation_phase"`
			IndexProbe map[string]any `json:"index_probe"`
			ID, Status string
			Queries    int
			Count      int `json:"verified_row_count"`
			Chunks     int
			Dimension  int
			Documents  int
			Metrics    map[string]float64
			Metadata   struct {
				Schema struct{ Vector struct{ Type string } }
			}
		}
		b, _ := os.ReadFile(path)
		if e := json.Unmarshal(b, &cell); e != nil {
			return e
		}
		if cell.Status != "passed" {
			return fmt.Errorf("cell not passed %s", path)
		}
		if expected[cell.ID] == 0 || cell.Chunks != expected[cell.ID] || cell.Queries != 799 || filepath.Base(path) != cell.ID+".json" {
			return fmt.Errorf("unexpected cell design: %s", cell.ID)
		}
		delete(expected, cell.ID)
		if cell.Phase != "settled-v1" || cell.IndexProbe == nil {
			return fmt.Errorf("primary settled phase missing: %s", cell.ID)
		}
		if cell.Count != cell.Chunks || cell.Count <= 0 {
			return fmt.Errorf("row count mismatch %s", cell.ID)
		}
		if cell.Metadata.Schema.Vector.Type != fmt.Sprintf("[%d]f32", cell.Dimension) {
			return fmt.Errorf("dimension mismatch %s", cell.ID)
		}
		seenq := map[string]bool{}
		sums := map[string]float64{}
		totalHits, totalRelevant := 0, 0
		queryValues := map[string]float64{}
		e := read("results/queries/"+cell.ID+".jsonl", func(b []byte) error {
			var row struct {
				Performance map[string]any     `json:"performance"`
				QID         string             `json:"qid"`
				Docs        []string           `json:"ranked_docs"`
				Metrics     map[string]float64 `json:"metrics"`
			}
			if e := json.Unmarshal(b, &row); e != nil {
				return e
			}
			if v, ok := row.Performance["exhaustive_search_count"].(float64); !ok || v < 0 {
				return fmt.Errorf("missing exhaustive-search receipt: %s", cell.ID)
			}
			if seenq[row.QID] || !queries[row.QID] {
				return fmt.Errorf("unexpected query")
			}
			seenq[row.QID] = true
			nrel := 0
			for _, r := range rel[row.QID] {
				if r {
					nrel++
				}
			}
			if nrel == 0 {
				return fmt.Errorf("no positive qrels")
			}
			seen := map[string]bool{}
			var gain, ideal, rr float64
			h5, h10, h100 := 0, 0, 0
			if len(row.Docs) != 100 {
				return fmt.Errorf("rank length")
			}
			for i, d := range row.Docs {
				if seen[d] || !selected[d] || (cell.Documents == 20000 && !secondary[d]) {
					return fmt.Errorf("duplicate or foreign document")
				}
				seen[d] = true
				if rel[row.QID][d] {
					if i < 5 {
						h5++
					}
					if i < 10 {
						h10++
						gain += math.Log(2) / math.Log(float64(i+2))
						if rr == 0 {
							rr = 1 / float64(i+1)
						}
					}
					if i < 100 {
						h100++
					}
				}
			}
			for i := 0; i < nrel && i < 10; i++ {
				ideal += math.Log(2) / math.Log(float64(i+2))
			}
			vals := map[string]float64{"ndcg10": gain / ideal, "mrr10": rr, "recall5": float64(h5) / float64(nrel), "recall10": float64(h10) / float64(nrel), "recall100": float64(h100) / float64(nrel)}
			queryValues[row.QID] = gain / ideal
			for k, v := range vals {
				if math.Abs(v-row.Metrics[k]) > 1e-10 {
					return fmt.Errorf("per query mismatch %s %s", cell.ID, k)
				}
				sums[k] += v
			}
			totalHits += h100
			totalRelevant += nrel
			return nil
		})
		if e != nil {
			return e
		}
		if len(seenq) != 799 {
			return fmt.Errorf("incomplete queries")
		}
		for k, v := range sums {
			sums[k] = v / 799
			if math.Abs(sums[k]-cell.Metrics[k]) > 1e-10 {
				return fmt.Errorf("headline mismatch %s %s", cell.ID, k)
			}
		}
		sums["micro_recall100"] = float64(totalHits) / float64(totalRelevant)
		if math.Abs(sums["micro_recall100"]-cell.Metrics["micro_recall100"]) > 1e-10 {
			return fmt.Errorf("micro recall mismatch: %s", cell.ID)
		}
		ids := make([]string, 0, len(queryValues))
		for id := range queryValues {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			independentValues[cell.ID] = append(independentValues[cell.ID], queryValues[id])
		}
		results = append(results, map[string]any{"cell": cell.ID, "queries": len(seenq), "independent_metrics": sums})
	}
	status := "partial"
	if len(files) == 96 {
		if len(expected) != 0 {
			return fmt.Errorf("missing design cells")
		}
		if e := verifyBootstrap(independentValues); e != nil {
			return e
		}
		status = "passed"
	}
	out := map[string]any{"run_id": runID, "status": status, "cells": len(files), "expected": 96, "corpus": len(selected), "queries": len(queries), "checks": "Independent metric calculation from ranked doc IDs; attribution; judged-document inclusion; query set; uniqueness; exact namespace rowcounts; dimensions", "results": results}
	b, _ := json.MarshalIndent(out, "", "  ")
	if e := os.WriteFile("results/independent-audit.json", append(b, '\n'), 0644); e != nil {
		return e
	}
	fmt.Printf("Independent audit: %d/96 cells, %s\n", len(files), status)
	return nil
}
