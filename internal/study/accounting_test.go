package study

import (
	"compress/gzip"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWritePricing(t *testing.T) {
	for _, c := range []struct{ bytes, want float64 }{
		{0, 0}, {100, .00002}, {10000, .00002},
		{100000, .00016}, {1000000, .0012}, {10000000, .01},
	} {
		got, _ := WriteCost(c.bytes)
		if math.Abs(got-c.want) > 1e-12 {
			t.Fatalf("bytes=%g: got %g want %g", c.bytes, got, c.want)
		}
	}
}

func TestArchivedJournalWithNewRequests(t *testing.T) {
	p := filepath.Join(t.TempDir(), "requests.jsonl")
	f, e := os.Create(p + ".gz")
	if e != nil {
		t.Fatal(e)
	}
	z := gzip.NewWriter(f)
	z.Write([]byte("old1\nold2\n"))
	z.Close()
	f.Close()
	for _, live := range []bool{false, true} {
		if live {
			os.WriteFile(p, []byte("new1\n"), 0644)
		}
		var records []string
		e := ScanJournal(p, func(b []byte) error { records = append(records, string(b)); return nil })
		want := "old1,old2"
		if live {
			want += ",new1"
		}
		if e != nil || strings.Join(records, ",") != want {
			t.Fatalf("%v: %v", records, e)
		}
	}
}
