package study

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type Cell struct {
	Phase         string             `json:"evaluation_phase,omitempty"`
	IndexProbe    map[string]any     `json:"index_probe,omitempty"`
	ID            string             `json:"id"`
	Run           string             `json:"run"`
	Model         string             `json:"model"`
	Mode          string             `json:"dimension_mode"`
	Status        string             `json:"status"`
	Reason        string             `json:"reason,omitempty"`
	Documents     int                `json:"documents"`
	ChunkSize     int                `json:"chunk_size"`
	Chunks        int                `json:"chunks"`
	Dimension     int                `json:"dimension"`
	Queries       int                `json:"queries"`
	Namespace     string             `json:"namespace"`
	Started       time.Time          `json:"started"`
	Finished      time.Time          `json:"finished"`
	EmbedSeconds  float64            `json:"embedding_seconds_including_cache"`
	IngestSeconds float64            `json:"ingest_seconds"`
	Metrics       map[string]float64 `json:"metrics,omitempty"`
	Metadata      map[string]any     `json:"metadata,omitempty"`
	Count         int                `json:"verified_row_count"`
}

func TPBase() string {
	if s := os.Getenv("TURBOPUFFER_BASE_URL"); s != "" {
		return strings.TrimRight(s, "/")
	}
	return "https://turbopuffer.int.exe.xyz"
}
func TP(method, path string, payload, out any) error {
	return Request(method, TPBase()+path, os.Getenv("TURBOPUFFER_API_KEY"), payload, out, 0, "tp:"+path)
}
func Ingest(c *Cell, chunks []Chunk, vectors [][]float32) error {
	checkpoint := "data/ingest/" + c.ID + ".json"
	start := 0
	b, e := os.ReadFile(checkpoint)
	if e == nil {
		json.Unmarshal(b, &start)
	}
	for i := start; i < len(chunks); i += 1000 {
		end := min(i+1000, len(chunks))
		rows := make([]map[string]any, 0, end-i)
		for j := i; j < end; j++ {
			rows = append(rows, map[string]any{"id": chunks[j].ID, "vector": vectors[j], "docid": chunks[j].DocID})
		}
		var response map[string]any
		if e := TP("POST", "/v2/namespaces/"+c.Namespace, map[string]any{"upsert_rows": rows, "distance_metric": "cosine_distance", "schema": map[string]any{"docid": map[string]any{"type": "string", "filterable": true}}}, &response); e != nil {
			return e
		}
		if e := Save(checkpoint, end); e != nil {
			return e
		}
	}
	// Exact aggregation, unlike approximate metadata count.
	var count struct {
		Aggregations struct {
			Count int `json:"count"`
		} `json:"aggregations"`
	}
	if e := TP("POST", "/v2/namespaces/"+c.Namespace+"/query", map[string]any{"aggregate_by": map[string]any{"count": []any{"Count"}}}, &count); e != nil {
		return e
	}
	if count.Aggregations.Count != len(chunks) {
		return fmt.Errorf("namespace count %d != %d", count.Aggregations.Count, len(chunks))
	}
	c.Count = count.Aggregations.Count
	return TP("GET", "/v1/namespaces/"+c.Namespace+"/metadata", nil, &c.Metadata)
}
func Search(ns string, v []float32) ([]string, int, map[string]any, error) {
	for k := 500; ; k = min(k*2, 10000) {
		var response struct {
			Performance map[string]any `json:"performance"`
			Rows        []struct {
				DocID string `json:"docid"`
			} `json:"rows"`
		}
		e := TP("POST", "/v2/namespaces/"+ns+"/query", map[string]any{"rank_by": []any{"vector", "ANN", v}, "top_k": k, "include_attributes": []string{"docid"}, "consistency": map[string]any{"level": "strong"}}, &response)
		if e != nil {
			return nil, k, nil, e
		}
		seen := map[string]bool{}
		ds := []string{}
		for _, r := range response.Rows {
			if r.DocID == "" {
				return nil, k, nil, fmt.Errorf("missing docid")
			}
			if !seen[r.DocID] {
				seen[r.DocID] = true
				ds = append(ds, r.DocID)
				if len(ds) == 100 {
					return ds, k, response.Performance, nil
				}
			}
		}
		if k == 10000 {
			return nil, k, nil, fmt.Errorf("overfetch exhausted with %d unique docs", len(ds))
		}
	}
}
func Exact(chunks []Chunk, vs [][]float32, q []float32) []string {
	scores := map[string]float64{}
	for i, v := range vs {
		dot := float64(0)
		for j, x := range v {
			dot += float64(x) * float64(q[j])
		}
		old, ok := scores[chunks[i].DocID]
		if !ok || dot > old {
			scores[chunks[i].DocID] = dot
		}
	}
	ds := make([]string, 0, len(scores))
	for d := range scores {
		ds = append(ds, d)
	}
	sort.Slice(ds, func(i, j int) bool {
		if scores[ds[i]] == scores[ds[j]] {
			return ds[i] < ds[j]
		}
		return scores[ds[i]] > scores[ds[j]]
	})
	return ds[:min(100, len(ds))]
}
func Evaluate(c *Cell, chunks []Chunk, vs, qv [][]float32, qs []Query, rels map[string]map[string]int) error {
	rs := make([]Record, len(qs))
	jobs := make(chan int, len(qs))
	for i := range qs {
		jobs <- i
	}
	close(jobs)
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				t := time.Now()
				docs, k, performance, e := Search(c.Namespace, qv[i])
				if e != nil {
					errs <- e
					return
				}
				rs[i] = Record{QID: qs[i].ID, Metrics: Score(docs, rels[qs[i].ID]), Docs: docs, Seconds: time.Since(t).Seconds(), Overfetch: k, Performance: performance}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		return e
	}
	var b strings.Builder
	enc := json.NewEncoder(&b)
	for _, r := range rs {
		if e := enc.Encode(r); e != nil {
			return e
		}
	}
	if e := Atomic("results/queries/"+c.ID+".jsonl", []byte(b.String())); e != nil {
		return e
	}
	c.Metrics = Mean(rs)
	audits := []map[string]any{}
	for i := 0; i < min(8, len(qs)); i++ {
		t := time.Now()
		exact := Exact(chunks, vs, qv[i])
		overlap := 0
		set := map[string]bool{}
		for _, d := range exact[:10] {
			set[d] = true
		}
		for _, d := range rs[i].Docs[:10] {
			if set[d] {
				overlap++
			}
		}
		audits = append(audits, map[string]any{"qid": qs[i].ID, "exact_docs": exact, "exact_metrics": Score(exact, rels[qs[i].ID]), "ann_metrics": rs[i].Metrics, "top10_overlap": float64(overlap) / 10, "seconds": time.Since(t).Seconds()})
	}
	return Save("results/exact/"+c.ID+".json", audits)
}
func RunCell(m Model, c *Cell, chunks []Chunk, qs []Query, rels map[string]map[string]int, cache *Cache) error {
	texts := make([]string, len(chunks))
	for i, ch := range chunks {
		texts[i] = m.Format(ch.Text, ch.Title, false)
	}
	qt := make([]string, len(qs))
	for i, q := range qs {
		qt[i] = m.Format(q.Text, "", true)
		if len(qt[i]) > 6000 {
			return fmt.Errorf("query too long")
		}
	}
	t := time.Now()
	dv, e := cache.Get(texts, false)
	if e != nil {
		return e
	}
	qv, e := cache.Get(qt, true)
	if e != nil {
		return e
	}
	c.EmbedSeconds = time.Since(t).Seconds()
	vs := make([][]float32, len(dv))
	for i, v := range dv {
		vs[i], e = Normalize(v, c.Dimension)
		if e != nil {
			return e
		}
	}
	nq := make([][]float32, len(qv))
	for i, v := range qv {
		nq[i], e = Normalize(v, c.Dimension)
		if e != nil {
			return e
		}
	}
	t = time.Now()
	if e = Ingest(c, chunks, vs); e != nil {
		return e
	}
	c.IngestSeconds = time.Since(t).Seconds()
	return Evaluate(c, chunks, vs, nq, qs, rels)
}
func Matrix(modelID, run string, pilot bool) error {
	ds, qs, rels, e := LoadData()
	if e != nil {
		return e
	}
	if len(ds) != 100000 || len(qs) != 799 {
		return fmt.Errorf("frozen counts violated")
	}
	b, e := os.ReadFile("data/corpus.jsonl")
	if e != nil {
		return e
	}
	var manifest map[string]any
	mb, e := os.ReadFile("results/manifests/corpus.json")
	if e != nil {
		return e
	}
	json.Unmarshal(mb, &manifest)
	if Hash(b) != manifest["corpus_sha256"] {
		return fmt.Errorf("corpus checksum changed")
	}
	b = nil
	treatments := [][2]int{{100000, 256}, {20000, 128}, {20000, 256}, {20000, 512}}
	if pilot {
		treatments = [][2]int{{1000, 256}}
		qs = qs[:20]
	}
	var last error
	for _, m := range Models {
		if modelID != "" && modelID != m.ID && !(modelID == "hosted" && m.Provider != "local") {
			continue
		}
		dims := []int{128, 256, 512, m.Native}
		if pilot {
			dims = []int{128}
		}
		var cache *Cache
		prevdim := 0
		for _, dim := range dims {
			if cache == nil || (m.Mode == "api" && prevdim != dim) {
				cache, e = NewCache(m, dim)
				if e != nil {
					return e
				}
				prevdim = dim
			}
			for _, t := range treatments {
				id := fmt.Sprintf("%s-%s-n%d-c%d-d%d", run, m.ID, t[0], t[1], dim)
				path := "results/cells/" + id + ".json"
				var old Cell
				if b, e := os.ReadFile(path); e == nil && json.Unmarshal(b, &old) == nil && old.Status == "passed" {
					fmt.Println("resume passed", id)
					continue
				}
				chunks, e := Chunks(ds[:t[0]], t[1])
				if e != nil {
					return e
				}
				c := Cell{ID: id, Run: run, Model: m.ID, Mode: m.Mode, Status: "running", Documents: t[0], ChunkSize: t[1], Chunks: len(chunks), Dimension: dim, Queries: len(qs), Namespace: "egvl-" + id, Started: time.Now().UTC()}
				if e = Save(path, c); e != nil {
					return e
				}
				fmt.Println("cell start", id)
				e = RunCell(m, &c, chunks, qs, rels, cache)
				c.Finished = time.Now().UTC()
				if e != nil {
					c.Status = "error"
					c.Reason = e.Error()
					last = e
					fmt.Println("cell error", id, e)
				} else {
					c.Status = "passed"
					fmt.Printf("cell passed %s ndcg %.5f\n", id, c.Metrics["ndcg10"])
				}
				if e = Save(path, c); e != nil {
					return e
				}
				if c.Status == "error" && strings.Contains(c.Reason, "embed:") { // bounded: do not repeat a known failed provider for subsequent cells
					return fmt.Errorf("model stopped after embedding failure: %s", c.Reason)
				}
			}
		}
	}
	return last
}
func Integration() error {
	ns := "egvl-integration-20261012"
	p := map[string]any{"upsert_rows": []any{map[string]any{"id": "a", "vector": []float64{1, 0}, "docid": "d1"}, map[string]any{"id": "b", "vector": []float64{0, 1}, "docid": "d2"}}, "distance_metric": "cosine_distance"}
	var out map[string]any
	if e := TP("POST", "/v2/namespaces/"+ns, p, &out); e != nil {
		return e
	}
	out = map[string]any{}
	if e := TP("POST", "/v2/namespaces/"+ns+"/query", map[string]any{"rank_by": []any{"vector", "ANN", []float64{1, 0}}, "top_k": 2, "include_attributes": true}, &out); e != nil {
		return e
	}
	rows, ok := out["rows"].([]any)
	if !ok || len(rows) != 2 || rows[0].(map[string]any)["id"] != "a" {
		return fmt.Errorf("integration unexpected response")
	}
	for i, row := range rows {
		r, ok := row.(map[string]any)
		if !ok {
			return fmt.Errorf("integration malformed row")
		}
		distance, ok := r["$dist"].(float64)
		if !ok || math.Abs(distance-float64(i)) > 1e-6 {
			return fmt.Errorf("integration cosine distance mismatch: row %d", i)
		}
	}
	return Save("results/integration.json", map[string]any{"status": "passed", "utc": time.Now().UTC(), "namespace": ns, "response": out, "checks": "two rows; correct nearest ID; returned cosine distances 0 and 1 within1e-6"})
}
