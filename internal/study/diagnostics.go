package study

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func Diagnostics() error {
	ds, qs, rels, e := LoadData()
	if e != nil {
		return e
	}
	rows := []map[string]any{}
	for _, t := range [][2]int{{100000, 256}, {20000, 128}, {20000, 256}, {20000, 512}} {
		cs, e := Chunks(ds[:t[0]], t[1])
		if e != nil {
			return e
		}
		lengths := make([]int, len(cs))
		byteMax, capCount, multi := 0, 0, 0
		for i, c := range cs {
			lengths[i] = len(strings.Fields(c.Text))
			byteMax = max(byteMax, len(c.Text)+len(c.Title))
			if i+1 < len(cs) && cs[i+1].DocID == c.DocID {
				if lengths[i] < t[1] {
					capCount++
				}
				multi++
			}
		}
		sort.Ints(lengths)
		rows = append(rows, map[string]any{"documents": t[0], "target_words": t[1], "chunks": len(cs), "words_min": lengths[0], "words_median": lengths[len(lengths)/2], "words_p95": lengths[int(float64(len(lengths)-1)*.95)], "words_max": lengths[len(lengths)-1], "title_text_max_utf8_bytes": byteMax, "byte_ceiling_constrained_chunks": capCount, "extra_chunks_beyond_first": multi})
	}
	qMax := 0
	positive, negative := 0, 0
	for _, q := range qs {
		qMax = max(qMax, len(q.Text))
		for _, v := range rels[q.ID] {
			if v > 0 {
				positive++
			} else {
				negative++
			}
		}
	}
	return Save("results/token-overflow-audit.json", map[string]any{"utc": time.Now().UTC(), "unit": "whitespace-delimited words; not model tokens", "text_changed_after_freeze": false, "maximum_title_text_bytes": 6000, "query_max_utf8_bytes": qMax, "positive_judgments": positive, "negative_judgments": negative, "queries": len(qs), "treatments": rows, "note": "Common byte ceiling subdivides instead of discarding text. Raw UTF-8 bytes plus prefix room conservatively bound byte/subword tokenizer lengths; local model additionally checks exact tokens. No provider truncation is requested."})
}
func RefreshMetadata() error {
	files, e := filepath.Glob("results/cells/" + RunID() + "-*.json")
	if e != nil {
		return e
	}
	all := []map[string]any{}
	for _, p := range files {
		b, e := os.ReadFile(p)
		if e != nil {
			return e
		}
		var c Cell
		if e = json.Unmarshal(b, &c); e != nil {
			return e
		}
		if c.Status != "passed" {
			continue
		}
		if !strings.HasPrefix(c.Namespace, "egvl-"+RunID()+"-") {
			return fmt.Errorf("foreign namespace")
		}
		var metadata map[string]any
		if e = TP("GET", "/v1/namespaces/"+c.Namespace+"/metadata", nil, &metadata); e != nil {
			return e
		}
		all = append(all, map[string]any{"cell": c.ID, "namespace": c.Namespace, "metadata": metadata, "raw_vector_bytes": int64(c.Chunks) * int64(c.Dimension) * 4, "exact_ingested_rows": c.Count, "utc": time.Now().UTC()})
	}
	return Save("results/final-storage-metadata.json", all)
}

// Sentinel re-embeds fixed queries after the run and compares with the actual
// cached run vectors. It detects gross provider alias/runtime drift without
// pretending hosted weights are pinned. It performs one bounded call per model.
func Sentinel() error {
	_, qs, _, e := LoadData()
	if e != nil {
		return e
	}
	all := []map[string]any{}
	for _, m := range Models {
		c, e := NewCache(m, m.Native)
		if e != nil {
			return e
		}
		c.query = true
		indexes := []int{0, 400}
		texts := []string{}
		old := [][]float32{}
		for _, i := range indexes {
			s := m.Format(qs[i].Text, "", true)
			v, ok := c.values[c.Key(s)]
			if !ok {
				return fmt.Errorf("sentinel native cache absent: %s", m.ID)
			}
			texts = append(texts, s)
			old = append(old, v)
		}
		fresh, e := EmbedRole(m, texts, m.Native, true)
		if e != nil {
			return e
		}
		comparisons := []map[string]any{}
		for i, v := range fresh {
			a, e := Normalize(old[i], m.Native)
			if e != nil {
				return e
			}
			b, e := Normalize(v, m.Native)
			if e != nil {
				return e
			}
			var cosine float64
			for j, x := range a {
				cosine += float64(x) * float64(b[j])
			}
			ab, _ := json.Marshal(old[i])
			bb, _ := json.Marshal(v)
			comparisons = append(comparisons, map[string]any{"qid": qs[indexes[i]].ID, "cosine_to_cached_run": cosine, "cached_vector_sha256": Hash(ab), "fresh_vector_sha256": Hash(bb), "dimension": len(v)})
		}
		all = append(all, map[string]any{"model": m.ID, "utc": time.Now().UTC(), "comparisons": comparisons})
	}
	return Save("results/end-to-end-sentinel.json", all)
}
