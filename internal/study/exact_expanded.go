package study

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// ExpandedExact is a post-hoc diagnostic, not an alternative primary database.
// Selection and dimensions were frozen in LEDGER before these calculations.
func ExpandedExact() error {
	ds, qs, rels, e := LoadData()
	if e != nil {
		return e
	}
	chunks, e := Chunks(ds, 256)
	if e != nil {
		return e
	}
	sort.Slice(qs, func(i, j int) bool {
		return Hash([]byte("exact-audit-20261012:"+qs[i].ID)) < Hash([]byte("exact-audit-20261012:"+qs[j].ID))
	})
	qs = qs[:64]
	for _, m := range Models {
		for _, dim := range []int{128, m.Native} {
			id := fmt.Sprintf("%s-%s-n100000-c256-d%d", RunID(), m.ID, dim)
			path := "results/exact-expanded/" + id + ".json"
			if _, e := os.Stat(path); e == nil {
				continue
			}
			b, e := os.ReadFile("results/cells/" + id + ".json")
			if e != nil {
				return e
			}
			var c Cell
			if e = json.Unmarshal(b, &c); e != nil {
				return e
			}
			if c.Status != "passed" || c.Phase != "settled-v1" {
				return fmt.Errorf("not settled: %s", id)
			}
			cache, e := NewCache(m, dim)
			if e != nil {
				return e
			}
			texts := make([]string, len(chunks))
			for i, ch := range chunks {
				texts[i] = m.Format(ch.Text, ch.Title, false)
			}
			vs, e := CachedVectors(cache, texts, false, dim)
			if e != nil {
				return e
			}
			qt := make([]string, len(qs))
			for i, q := range qs {
				qt[i] = m.Format(q.Text, "", true)
			}
			qv, e := CachedVectors(cache, qt, true, dim)
			if e != nil {
				return e
			}
			rs, e := ReadRecords("results/queries/" + id + ".jsonl")
			if e != nil {
				return e
			}
			byQuery := map[string]Record{}
			for _, r := range rs {
				byQuery[r.QID] = r
			}
			rows := []map[string]any{}
			for i, q := range qs {
				docs := Exact(chunks, vs, qv[i])
				ann, ok := byQuery[q.ID]
				if !ok {
					return fmt.Errorf("missing ANN query: %s", q.ID)
				}
				set := map[string]bool{}
				for _, d := range docs[:10] {
					set[d] = true
				}
				hits := 0
				for _, d := range ann.Docs[:10] {
					if set[d] {
						hits++
					}
				}
				rows = append(rows, map[string]any{"qid": q.ID, "exact_docs": docs, "exact_metrics": Score(docs, rels[q.ID]), "ann_metrics": ann.Metrics, "top10_overlap": float64(hits) / 10})
			}
			if e = Save(path, rows); e != nil {
				return e
			}
			fmt.Println("expanded exact", id, len(rows))
		}
	}
	return nil
}

func AuditSummary(cells []Cell) (string, error) {
	all := []map[string]any{}
	var table string
	table = "\n## Exact-search and index-state sensitivity\n\nThese are end-to-end turbopuffer configurations: model representation and search approximation both affect scores. Eight fixed sorted queries per cell diagnose exact cosine versus settled ANN. A post-hoc, hash-selected common64-query sample additionally covers each model's primary128d and native configuration; it was specified before computing those exact results. Neither diagnostic replaces the799-query primary evaluation.\n\n| Primary configuration | Exact audit queries | Exact nDCG@10 | ANN on same queries | Top10 overlap | Settled minus fresh nDCG (all799) |\n|---|---:|---:|---:|---:|---:|\n"
	for _, c := range cells {
		if c.Status != "passed" || c.Phase != "settled-v1" {
			continue
		}
		var fresh Cell
		b, e := os.ReadFile("results/fresh-ingest/cells/" + c.ID + ".json")
		if e != nil {
			return "", e
		}
		if e = json.Unmarshal(b, &fresh); e != nil {
			return "", e
		}
		row := map[string]any{"cell": c.ID, "settled_minus_fresh_ndcg10": c.Metrics["ndcg10"] - fresh.Metrics["ndcg10"]}
		for _, scope := range []string{"exact", "exact-expanded"} {
			b, e := os.ReadFile("results/" + scope + "/" + c.ID + ".json")
			if os.IsNotExist(e) && scope == "exact-expanded" {
				continue
			}
			if e != nil {
				return "", e
			}
			var rs []struct {
				Exact   Metrics `json:"exact_metrics"`
				ANN     Metrics `json:"ann_metrics"`
				Overlap float64 `json:"top10_overlap"`
			}
			if e = json.Unmarshal(b, &rs); e != nil {
				return "", e
			}
			var exact, ann, overlap float64
			for _, r := range rs {
				exact += r.Exact.NDCG10
				ann += r.ANN.NDCG10
				overlap += r.Overlap
			}
			n := float64(len(rs))
			row[scope] = map[string]any{"queries": len(rs), "mean_exact_ndcg10": exact / n, "mean_ann_ndcg10": ann / n, "mean_top10_overlap": overlap / n, "ann_minus_exact_ndcg10": (ann - exact) / n}
			if scope == "exact-expanded" {
				table += fmt.Sprintf("| %s / %dd | %d | %.4f | %.4f | %.1f%% | %+.4f |\n", c.Model, c.Dimension, len(rs), exact/n, ann/n, 100*overlap/n, c.Metrics["ndcg10"]-fresh.Metrics["ndcg10"])
			}
		}
		all = append(all, row)
	}
	table += "\nAll cell-level differences are in results/search-audit-summary.json. Exact top10 overlap describes ranking agreement, not relevance recall. The64-query diagnostic is too small to establish population-wide ANN accuracy; conclusions apply to the measured model/dimension/index configuration. Fresh-versus-settled differences use identical vectors and demonstrate indexing-state sensitivity.\n"
	return table, Save("results/search-audit-summary.json", all)
}
