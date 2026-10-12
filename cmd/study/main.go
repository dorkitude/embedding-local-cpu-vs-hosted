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
	}{{"acquire", s.Acquire}, {"prepare", s.Prepare}} {
		f := c.fn
		r.AddCommand(&cobra.Command{Use: c.name, RunE: func(*cobra.Command, []string) error { return f() }})
	}
	if e := r.Execute(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
