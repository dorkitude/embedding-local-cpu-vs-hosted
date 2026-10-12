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
	files, e := filepath.Glob("results/cells/r20261012a-*.json")
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
		if !strings.HasPrefix(c.Namespace, "egvl-r20261012a-") {
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
