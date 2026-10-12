package study

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Query struct{ ID, Text string }
type Chunk struct{ ID, DocID, Title, Text string }
type Metrics struct {
	NDCG10    float64 `json:"ndcg10"`
	MRR10     float64 `json:"mrr10"`
	Recall5   float64 `json:"recall5"`
	Recall10  float64 `json:"recall10"`
	Recall100 float64 `json:"recall100"`
	Hits100   int     `json:"hits100"`
	Relevant  int     `json:"relevant"`
}
type Record struct {
	Performance map[string]any `json:"performance,omitempty"`
	QID         string         `json:"qid"`
	Metrics     Metrics        `json:"metrics"`
	Docs        []string       `json:"ranked_docs"`
	Seconds     float64        `json:"query_seconds"`
	Overfetch   int            `json:"overfetch"`
}

func LoadData() ([]Doc, []Query, map[string]map[string]int, error) {
	var ds []Doc
	var qs []Query
	rels := map[string]map[string]int{}
	e := Scan("data/corpus.jsonl", func(b []byte) error {
		var d Doc
		if e := json.Unmarshal(b, &d); e != nil {
			return e
		}
		ds = append(ds, d)
		return nil
	})
	if e != nil {
		return nil, nil, nil, e
	}
	e = Scan("data/queries.tsv", func(b []byte) error {
		f := strings.SplitN(string(b), "\t", 2)
		if len(f) != 2 {
			return fmt.Errorf("invalid query")
		}
		qs = append(qs, Query{f[0], f[1]})
		return nil
	})
	if e != nil {
		return nil, nil, nil, e
	}
	e = Scan("data/qrels.tsv", func(b []byte) error {
		f := strings.Fields(string(b))
		if len(f) != 4 {
			return fmt.Errorf("invalid qrels")
		}
		v, e := strconv.Atoi(f[3])
		if e != nil {
			return e
		}
		if rels[f[0]] == nil {
			rels[f[0]] = map[string]int{}
		}
		rels[f[0]][f[2]] = v
		return nil
	})
	sort.Slice(qs, func(i, j int) bool { return qs[i].ID < qs[j].ID })
	return ds, qs, rels, e
}
func Chunks(ds []Doc, size int) ([]Chunk, error) {
	if size < 8 {
		return nil, fmt.Errorf("chunk size too small")
	}
	out := []Chunk{}
	for _, d := range ds {
		ws := strings.Fields(d.Text)
		if len(ws) == 0 {
			return nil, fmt.Errorf("empty %s", d.ID)
		}
		j := 0
		for start := 0; start < len(ws); {
			end := min(start+size, len(ws))
			text := strings.Join(ws[start:end], " ")
			// A transparent common byte ceiling ensures <=8192 model tokens, including title/prefix.
			for len(text)+len(d.Title) > 6000 && end > start+1 {
				end--
				text = strings.Join(ws[start:end], " ")
			}
			if len(text)+len(d.Title) > 6000 {
				return nil, fmt.Errorf("unsplittable oversized word/title in %s", d.ID)
			}
			out = append(out, Chunk{fmt.Sprintf("%s-c%d", d.ID, j), d.ID, d.Title, text})
			j++
			if end == len(ws) {
				break
			}
			overlap := min(size/8, (end-start)/8)
			start = end - overlap
		}
	}
	return out, nil
}
func Score(ranked []string, rels map[string]int) Metrics {
	m := Metrics{}
	for _, r := range rels {
		if r > 0 {
			m.Relevant++
		}
	}
	if m.Relevant == 0 {
		return m
	}
	seen := map[string]bool{}
	rank := 0
	for _, d := range ranked {
		if seen[d] {
			continue
		}
		seen[d] = true
		rank++
		if rank > 100 {
			break
		}
		if rels[d] > 0 {
			m.Hits100++
			if rank <= 5 {
				m.Recall5++
			}
			if rank <= 10 {
				m.Recall10++
				m.NDCG10 += 1 / math.Log2(float64(rank+1))
				if m.MRR10 == 0 {
					m.MRR10 = 1 / float64(rank)
				}
			}
		}
	}
	idcg := 0.0
	for i := 1; i <= min(m.Relevant, 10); i++ {
		idcg += 1 / math.Log2(float64(i+1))
	}
	m.NDCG10 /= idcg
	m.Recall5 /= float64(m.Relevant)
	m.Recall10 /= float64(m.Relevant)
	m.Recall100 = float64(m.Hits100) / float64(m.Relevant)
	return m
}
func Mean(rs []Record) map[string]float64 {
	out := map[string]float64{}
	hits, rel := 0, 0
	for _, r := range rs {
		m := r.Metrics
		out["ndcg10"] += m.NDCG10
		out["mrr10"] += m.MRR10
		out["recall5"] += m.Recall5
		out["recall10"] += m.Recall10
		out["recall100"] += m.Recall100
		out["query_seconds"] += r.Seconds
		hits += m.Hits100
		rel += m.Relevant
	}
	for k := range out {
		out[k] /= float64(len(rs))
	}
	out["micro_recall100"] = float64(hits) / float64(rel)
	return out
}
func Bootstrap(a, b []Record) map[string]any {
	if len(a) != len(b) || len(a) == 0 {
		return map[string]any{"error": "unequal query count"}
	}
	delta := make([]float64, len(a))
	for i := range a {
		if a[i].QID != b[i].QID {
			return map[string]any{"error": "unpaired queries"}
		}
		delta[i] = a[i].Metrics.NDCG10 - b[i].Metrics.NDCG10
	}
	rng := rand.New(rand.NewSource(20261012))
	dist := make([]float64, 2000)
	mean := 0.0
	for _, x := range delta {
		mean += x
	}
	mean /= float64(len(delta))
	for i := range dist {
		for range delta {
			dist[i] += delta[rng.Intn(len(delta))]
		}
		dist[i] /= float64(len(delta))
	}
	sort.Float64s(dist)
	return map[string]any{"metric": "ndcg10", "difference": mean, "ci95": []float64{dist[49], dist[1949]}, "samples": 2000, "seed": 20261012, "queries": len(delta)}
}
func ReadRecords(path string) ([]Record, error) {
	rs := []Record{}
	e := Scan(path, func(b []byte) error {
		var r Record
		if e := json.Unmarshal(b, &r); e != nil {
			return e
		}
		rs = append(rs, r)
		return nil
	})
	return rs, e
}
func Estimate() error {
	ds, qs, _, e := LoadData()
	if e != nil {
		return e
	}
	all := []map[string]any{}
	total := 0.0
	for _, m := range Models {
		unique := map[string]bool{}
		var bytes int
		for _, t := range [][2]int{{100000, 256}, {20000, 128}, {20000, 256}, {20000, 512}} {
			cs, e := Chunks(ds[:t[0]], t[1])
			if e != nil {
				return e
			}
			for _, c := range cs {
				s := m.Format(c.Text, c.Title, false)
				if !unique[s] {
					unique[s] = true
					bytes += len(s) + 32
				}
			}
		}
		for _, q := range qs {
			bytes += len(m.Format(q.Text, "", true)) + 32
		}
		mult := 1
		if m.Mode == "api" {
			mult = 4
		}
		cost := float64(bytes) * m.Price / 1e6 * float64(mult)
		total += cost
		all = append(all, map[string]any{"model": m.ID, "unique_document_inputs": len(unique), "upper_bound_byte_tokens": bytes, "embedding_passes": mult, "usd_per_million_tokens": m.Price, "conservative_cost_usd": cost})
	}
	for _, t := range [][2]int{{100000, 256}, {20000, 128}, {20000, 256}, {20000, 512}} {
		cs, e := Chunks(ds[:t[0]], t[1])
		if e != nil {
			return e
		}
		texts := []string{}
		for _, c := range cs {
			texts = append(texts, c.ID+"|"+c.Title+"|"+c.Text)
		}
		b, _ := json.Marshal(texts)
		Save(fmt.Sprintf("results/manifests/chunks-%d-%d.json", t[0], t[1]), map[string]any{"documents": t[0], "chunks": len(cs), "words_per_chunk": t[1], "overlap": 0.125, "input_sha256": Hash(b), "max_text_title_bytes": 6000})
	}
	if total > 100 {
		return fmt.Errorf("estimate %.2f exceeds $100 planned embedding allowance", total)
	}
	return Save("results/budget-estimate.json", map[string]any{"estimated_before_run": true, "embedding_conservative_usd": total, "retry_factor": 1.2, "storage_query_allowance_usd": 30, "upper_budget_usd": 150, "models": all, "note": "bytes as upper token bound; all 4 OpenAI dimension calls charged separately; cache sharing across chunks; actual usage recorded; $120 hard embedding reservation cap"})
}
func Audit() error {
	_, qs, rels, e := LoadData()
	if e != nil {
		return e
	}
	entries, e := os.ReadDir("results/cells")
	if e != nil {
		return e
	}
	all := []map[string]any{}
	for _, en := range entries {
		if !strings.HasSuffix(en.Name(), ".json") {
			continue
		}
		var cell Cell
		b, e := os.ReadFile("results/cells/" + en.Name())
		if e != nil {
			return e
		}
		if e = json.Unmarshal(b, &cell); e != nil {
			return e
		}
		if cell.Status != "passed" {
			all = append(all, map[string]any{"cell": cell.ID, "status": cell.Status})
			continue
		}
		rs, e := ReadRecords("results/queries/" + cell.ID + ".jsonl")
		if e != nil {
			return e
		}
		if cell.Queries <= 0 || cell.Queries > len(qs) || len(rs) != cell.Queries {
			return fmt.Errorf("query count %s", cell.ID)
		}
		for i, r := range rs {
			if r.QID != qs[i].ID {
				return fmt.Errorf("query order mismatch")
			}
			s := Score(r.Docs, rels[r.QID])
			if s != r.Metrics {
				return fmt.Errorf("recompute mismatch %s %s", cell.ID, r.QID)
			}
		}
		all = append(all, map[string]any{"cell": cell.ID, "status": "audited", "recomputed": Mean(rs), "queries": len(rs)})
	}
	return Save("results/audit.json", all)
}
