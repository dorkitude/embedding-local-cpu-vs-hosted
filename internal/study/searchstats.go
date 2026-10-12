package study

import (
	"fmt"
	"math"
	"strings"
)

func SearchStats(cells []Cell, records map[string][]Record) (string, error) {
	all := []map[string]any{}
	var table strings.Builder
	table.WriteString("\n## Turbopuffer execution mode\n\nPrimary queries use the strong-consistency ANN operator after the recorded settled-state check. The provider can still report exhaustive work even when index metadata is up-to-date; these are not uniformly pure ANN searches. Fresh-ingest observations are retained separately. Readiness probes warm the namespace, so these primary query timings are not cold-cache timings.\n\n| Model / corpus / words / dimensions | Queries with zero exhaustive rows | Mixed queries | Fully exhaustive reported | Mean corpus fraction exhaustive |\n|---|---:|---:|---:|---:|\n")
	for _, c := range cells {
		rs, ok := records[c.ID]
		if !ok {
			continue
		}
		zero, mixed, full, missing := 0, 0, 0, 0
		sum, lo, hi := 0.0, math.Inf(1), 0.0
		for _, r := range rs {
			v, ok := r.Performance["exhaustive_search_count"].(float64)
			if !ok {
				missing++
				continue
			}
			sum += v
			lo = math.Min(lo, v)
			hi = math.Max(hi, v)
			if v == 0 {
				zero++
			} else if v >= float64(c.Chunks) {
				full++
			} else {
				mixed++
			}
		}
		fraction := sum / float64(len(rs)*c.Chunks)
		all = append(all, map[string]any{"cell": c.ID, "queries": len(rs), "zero_exhaustive_queries": zero, "mixed_queries": mixed, "fully_exhaustive_reported_queries": full, "missing_performance": missing, "mean_exhaustive_fraction": fraction, "exhaustive_rows_min": lo, "exhaustive_rows_max": hi, "chunks": c.Chunks, "phase": c.Phase})
		native := 0
		for _, m := range Models {
			if m.ID == c.Model {
				native = m.Native
			}
		}
		if c.ChunkSize == 256 && ((c.Documents == 100000 && c.Dimension == native) || (c.Documents == 20000 && c.Dimension == 128)) {
			fmt.Fprintf(&table, "| %s / %d / %d / %d | %d | %d | %d | %.2f%% |\n", c.Model, c.Documents, c.ChunkSize, c.Dimension, zero, mixed, full, 100*fraction)
		}
	}
	table.WriteString("\nAll96 cell-level execution summaries are in results/search-execution.json. The fully-exhaustive category is based on the service counter reaching the logical chunk count; it is not an inferred claim about undocumented internal algorithms. No local database replaces turbopuffer.\n")
	return table.String(), Save("results/search-execution.json", all)
}
