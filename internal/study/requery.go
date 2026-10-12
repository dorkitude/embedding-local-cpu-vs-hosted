package study

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func Settled(c *Cell, q []float32) error {
	deadline := time.Now().Add(150 * time.Second)
	last := -1
	stable := 0
	for time.Now().Before(deadline) {
		var meta map[string]any
		if e := TP("GET", "/v1/namespaces/"+c.Namespace+"/metadata", nil, &meta); e != nil {
			return e
		}
		var probe struct {
			Performance map[string]any `json:"performance"`
		}
		if e := TP("POST", "/v2/namespaces/"+c.Namespace+"/query", map[string]any{"rank_by": []any{"vector", "ANN", q}, "top_k": 1, "include_attributes": false, "consistency": map[string]any{"level": "strong"}}, &probe); e != nil {
			return e
		}
		count, ok := probe.Performance["exhaustive_search_count"].(float64)
		if !ok {
			return fmt.Errorf("missing exhaustive-search receipt")
		}
		if int(count) == last {
			stable++
		} else {
			stable = 1
			last = int(count)
		}
		index, _ := meta["index"].(map[string]any)
		lastWrite, _ := meta["last_write_at"].(string)
		writeTime, e := time.Parse(time.RFC3339Nano, lastWrite)
		if e == nil && time.Since(writeTime) >= 60*time.Second && index["status"] == "up-to-date" && (count == 0 || stable >= 3) {
			c.Metadata = meta
			c.IndexProbe = map[string]any{"utc": time.Now().UTC(), "metadata": meta, "performance": probe.Performance, "stable_samples": stable, "policy": "metadata up-to-date; write age >=60s; zero or 3 stable exhaustive-search counts spaced 5s; bounded150s; persistent fallback is recorded, not relabeled as pure ANN"}
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("index did not meet settled-state criterion within150s")
}
func CachedVectors(cache *Cache, texts []string, query bool, dimension int) ([][]float32, error) {
	cache.query = query
	out := make([][]float32, len(texts))
	for i, s := range texts {
		v, ok := cache.values[cache.Key(s)]
		if !ok {
			return nil, fmt.Errorf("read-only requery cache miss")
		}
		var e error
		out[i], e = Normalize(v, dimension)
		if e != nil {
			return nil, e
		}
	}
	return out, nil
}
func Requery(modelID string) error {
	ds, qs, rels, e := LoadData()
	if e != nil {
		return e
	}
	for _, m := range Models {
		if modelID != "" && modelID != m.ID {
			continue
		}
		files, _ := filepath.Glob("results/cells/" + RunID() + "-" + m.ID + "-*.json")
		cells := []Cell{}
		for _, p := range files {
			b, e := os.ReadFile(p)
			if e != nil {
				return e
			}
			var c Cell
			if e = json.Unmarshal(b, &c); e != nil {
				return e
			}
			if c.Status != "passed" && c.Phase != "settled-v1" {
				return fmt.Errorf("wait for full model lane: %s", c.ID)
			}
			cells = append(cells, c)
		}
		if len(cells) != 16 {
			return fmt.Errorf("model %s has %d/16 cells; wait for lane", m.ID, len(cells))
		}
		sort.Slice(cells, func(i, j int) bool {
			if cells[i].Dimension != cells[j].Dimension {
				return cells[i].Dimension < cells[j].Dimension
			}
			return cells[i].ID < cells[j].ID
		})
		var cache *Cache
		priorDim := 0
		for i := range cells {
			c := &cells[i]
			if c.Phase == "settled-v1" && c.Status == "passed" {
				fmt.Println("resume settled", c.ID)
				continue
			}
			if cache == nil || (m.Mode == "api" && c.Dimension != priorDim) {
				cache, e = NewCache(m, c.Dimension)
				if e != nil {
					return e
				}
				priorDim = c.Dimension
			}
			chunks, e := Chunks(ds[:c.Documents], c.ChunkSize)
			if e != nil {
				return e
			}
			texts := make([]string, len(chunks))
			for i, ch := range chunks {
				texts[i] = m.Format(ch.Text, ch.Title, false)
			}
			qt := make([]string, len(qs))
			for i, q := range qs {
				qt[i] = m.Format(q.Text, "", true)
			}
			vs, e := CachedVectors(cache, texts, false, c.Dimension)
			if e != nil {
				return e
			}
			qv, e := CachedVectors(cache, qt, true, c.Dimension)
			if e != nil {
				return e
			}
			for _, p := range []string{"cells/" + c.ID + ".json", "queries/" + c.ID + ".jsonl", "exact/" + c.ID + ".json"} {
				dest := "results/fresh-ingest/" + p
				if _, e := os.Stat(dest); os.IsNotExist(e) {
					b, e := os.ReadFile("results/" + p)
					if e != nil {
						return e
					}
					if e = Atomic(dest, b); e != nil {
						return e
					}
				}
			}
			c.Phase = "settled-v1"
			c.Status = "running"
			if e = Save("results/cells/"+c.ID+".json", c); e != nil {
				return e
			}
			fmt.Println("settled requery", c.ID)
			e = Settled(c, qv[0])
			if e == nil {
				e = Evaluate(c, chunks, vs, qv, qs, rels)
			}
			c.Finished = time.Now().UTC()
			if e != nil {
				c.Status = "error"
				c.Reason = e.Error()
			} else {
				c.Status = "passed"
				c.Reason = ""
			}
			if se := Save("results/cells/"+c.ID+".json", c); se != nil {
				return se
			}
			if e != nil {
				return e
			}
			fmt.Printf("settled passed %s ndcg %.6f\n", c.ID, c.Metrics["ndcg10"])
		}
	}
	return nil
}
