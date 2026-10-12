package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
)

func independentMetrics(docs []string, rel map[string]bool) map[string]float64 {
	n, h5, h10, h100 := 0, 0, 0, 0
	var dcg, ideal, rr float64
	for _, relevant := range rel {
		if relevant {
			n++
		}
	}
	for i, d := range docs {
		if !rel[d] {
			continue
		}
		if i < 100 {
			h100++
		}
		if i < 10 {
			h10++
			dcg += math.Log(2) / math.Log(float64(i+2))
			if rr == 0 {
				rr = 1 / float64(i+1)
			}
		}
		if i < 5 {
			h5++
		}
	}
	for i := 0; i < n && i < 10; i++ {
		ideal += math.Log(2) / math.Log(float64(i+2))
	}
	return map[string]float64{"ndcg10": dcg / ideal, "mrr10": rr, "recall5": float64(h5) / float64(n), "recall10": float64(h10) / float64(n), "recall100": float64(h100) / float64(n)}
}

func verifyExpanded(run string, rel map[string]map[string]bool, queries, selected map[string]bool) error {
	qids := []string{}
	for q := range queries {
		qids = append(qids, q)
	}
	sort.Slice(qids, func(i, j int) bool {
		return fmt.Sprintf("%x", sha256.Sum256([]byte("exact-audit-20261012:"+qids[i]))) < fmt.Sprintf("%x", sha256.Sum256([]byte("exact-audit-20261012:"+qids[j])))
	})
	expected := map[string]bool{}
	for _, q := range qids[:64] {
		expected[q] = true
	}
	checked := 0
	for model, native := range map[string]int{"gemma2": 768, "openai-large": 3072, "qwen8b": 4096, "openai-small": 1536, "gemini2": 3072, "bge": 1024} {
		for _, dim := range []int{128, native} {
			id := fmt.Sprintf("%s-%s-n100000-c256-d%d", run, model, dim)
			ann := map[string][]string{}
			if e := read("results/queries/"+id+".jsonl", func(b []byte) error {
				var r struct {
					QID  string   `json:"qid"`
					Docs []string `json:"ranked_docs"`
				}
				if e := json.Unmarshal(b, &r); e != nil {
					return e
				}
				ann[r.QID] = r.Docs
				return nil
			}); e != nil {
				return e
			}
			b, e := os.ReadFile("results/exact-expanded/" + id + ".json")
			if e != nil {
				return e
			}
			var rows []struct {
				QID     string             `json:"qid"`
				Docs    []string           `json:"exact_docs"`
				Exact   map[string]float64 `json:"exact_metrics"`
				ANN     map[string]float64 `json:"ann_metrics"`
				Overlap float64            `json:"top10_overlap"`
			}
			if e = json.Unmarshal(b, &rows); e != nil {
				return e
			}
			if len(rows) != 64 {
				return fmt.Errorf("expanded sample count: %s", id)
			}
			seenq := map[string]bool{}
			for _, r := range rows {
				if !expected[r.QID] || seenq[r.QID] || len(r.Docs) != 100 || len(ann[r.QID]) != 100 {
					return fmt.Errorf("expanded query/row count: %s", id)
				}
				seenq[r.QID] = true
				seen := map[string]bool{}
				for _, d := range r.Docs {
					if !selected[d] || seen[d] {
						return fmt.Errorf("invalid exact document")
					}
					seen[d] = true
				}
				for k, v := range independentMetrics(r.Docs, rel[r.QID]) {
					if math.Abs(v-r.Exact[k]) > 1e-10 {
						return fmt.Errorf("exact score mismatch %s %s", id, k)
					}
				}
				for k, v := range independentMetrics(ann[r.QID], rel[r.QID]) {
					if math.Abs(v-r.ANN[k]) > 1e-10 {
						return fmt.Errorf("ANN diagnostic score mismatch %s %s", id, k)
					}
				}
				top := map[string]bool{}
				for _, d := range r.Docs[:10] {
					top[d] = true
				}
				hits := 0
				for _, d := range ann[r.QID][:10] {
					if top[d] {
						hits++
					}
				}
				if math.Abs(float64(hits)/10-r.Overlap) > 1e-10 {
					return fmt.Errorf("exact overlap mismatch")
				}
				checked++
			}
		}
	}
	b, _ := json.MarshalIndent(map[string]any{"status": "passed", "configurations": 12, "queries_per_configuration": 64, "query_configurations_checked": checked, "checks": "independent deterministic query selection; exact and ANN rank-based metrics; top10 overlap; passage IDs and uniqueness; primary corpus"}, "", "  ")
	return os.WriteFile("results/independent-expanded-exact-audit.json", append(b, '\n'), 0644)
}
