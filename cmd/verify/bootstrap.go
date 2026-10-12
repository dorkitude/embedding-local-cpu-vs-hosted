package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
)

type Boot struct {
	Difference float64   `json:"difference"`
	CI         []float64 `json:"ci95"`
	Samples    int       `json:"samples"`
	Seed       int64     `json:"seed"`
	Queries    int       `json:"queries"`
	Error      string    `json:"error"`
}

func verifyBootstrap(values map[string][]float64) error {
	var data struct {
		Absolute map[string]Boot `json:"absolute_ndcg95"`
		Pairs    []struct {
			A, B, Kind string
			Bootstrap  Boot
		} `json:"paired"`
	}
	b, e := os.ReadFile("results/bootstrap.json")
	if e != nil {
		return e
	}
	if e = json.Unmarshal(b, &data); e != nil {
		return e
	}
	// Independent formulation: bootstrap multiplicities, rather than the
	// evaluator's repeated summation of a newly drawn index sequence per pair.
	rng := rand.New(rand.NewSource(20261012))
	weights := make([][]uint16, 2000)
	for i := range weights {
		weights[i] = make([]uint16, 799)
		for k := 0; k < 799; k++ {
			weights[i][rng.Intn(799)]++
		}
	}
	checks := 0
	verify := func(a, b []float64, got Boot) error {
		if got.Error != "" || got.Samples != 2000 || got.Seed != 20261012 || got.Queries != 799 || len(got.CI) != 2 || len(a) != 799 || len(b) != 799 {
			return fmt.Errorf("bootstrap metadata/query counts")
		}
		d := make([]float64, 799)
		mean := 0.0
		for i := range d {
			d[i] = a[i] - b[i]
			mean += d[i]
		}
		mean /= 799
		distribution := make([]float64, 2000)
		for i, w := range weights {
			for j, n := range w {
				distribution[i] += float64(n) * d[j]
			}
			distribution[i] /= 799
		}
		sort.Float64s(distribution)
		if math.Abs(mean-got.Difference) > 1e-9 || math.Abs(distribution[49]-got.CI[0]) > 1e-9 || math.Abs(distribution[1949]-got.CI[1]) > 1e-9 {
			return fmt.Errorf("independent bootstrap mismatch")
		}
		checks++
		return nil
	}
	if len(data.Absolute) != 96 {
		return fmt.Errorf("absolute bootstrap count %d", len(data.Absolute))
	}
	for id, ci := range data.Absolute {
		if e := verify(values[id], make([]float64, 799), ci); e != nil {
			return fmt.Errorf("%s: %w", id, e)
		}
	}
	for _, p := range data.Pairs {
		if e := verify(values[p.A], values[p.B], p.Bootstrap); e != nil {
			return fmt.Errorf("%s vs %s: %w", p.A, p.B, e)
		}
	}
	out := map[string]any{"status": "passed", "intervals_independently_recomputed": checks, "absolute_intervals": len(data.Absolute), "paired_intervals": len(data.Pairs), "samples": 2000, "seed": 20261012, "queries": 799, "method": "independent rank-to-nDCG calculation, bootstrap multiplicity weights; tolerance1e-9"}
	b, _ = json.MarshalIndent(out, "", "  ")
	return os.WriteFile("results/independent-bootstrap-audit.json", append(b, '\n'), 0644)
}
