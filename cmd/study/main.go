package main

import (
	"fmt"
	s "github.com/dorkitude/embedding-gpu-vs-local/internal/study"
	"github.com/spf13/cobra"
	"os"
)

func main() {
	r := &cobra.Command{Use: "study"}
	for _, c := range []struct {
		name string
		fn   func() error
	}{{"discover", s.Discover}, {"acquire", s.Acquire}, {"prepare", s.Prepare}, {"preflight", s.Preflight}, {"estimate", s.Estimate}, {"integration", s.Integration}, {"audit", s.Audit}, {"report", s.Report}, {"diagnostics", s.Diagnostics}, {"sentinel", s.Sentinel}, {"refresh-metadata", s.RefreshMetadata}} {
		f := c.fn
		r.AddCommand(&cobra.Command{Use: c.name, RunE: func(*cobra.Command, []string) error { return f() }})
	}
	var model, run string
	var pilot bool
	m := &cobra.Command{Use: "matrix", RunE: func(*cobra.Command, []string) error { return s.Matrix(model, run, pilot) }}
	m.Flags().StringVar(&model, "model", "", "Model ID or hosted")
	m.Flags().StringVar(&run, "run", "r1", "Frozen run ID")
	m.Flags().BoolVar(&pilot, "pilot", false, "1000 documents / 20 queries smoke only")
	r.AddCommand(m)
	if e := r.Execute(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
