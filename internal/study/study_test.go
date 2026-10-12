package study

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScore(t *testing.T) {
	m := Score([]string{"bad", "a", "a", "b"}, map[string]int{"a": 1, "b": 1, "negative": 0})
	want := (1/math.Log2(3) + 1/math.Log2(4)) / (1 + 1/math.Log2(3))
	if math.Abs(m.NDCG10-want) > 1e-12 || m.MRR10 != .5 || m.Recall100 != 1 {
		t.Fatalf("%+v", m)
	}
	if Score([]string{"a"}, nil).NDCG10 != 0 {
		t.Fatal("empty qrels")
	}
}
func TestChunkCoverage(t *testing.T) {
	ws := make([]string, 1000)
	for i := range ws {
		ws[i] = "word"
	}
	cs, e := Chunks([]Doc{{ID: "a", Title: "title", Text: strings.Join(ws, " ")}}, 128)
	if e != nil {
		t.Fatal(e)
	}
	if len(cs) != 9 {
		t.Fatalf("chunks %d", len(cs))
	}
	if len(strings.Fields(cs[1].Text)) != 128 {
		t.Fatal("size")
	}
	if cs[0].DocID != "a" || cs[1].ID != "a-c1" {
		t.Fatal("mapping")
	}
	// Verify overlapping text boundaries with distinct words.
	words := strings.Fields("a b c d e f g h i j k l m n o p q r s t")
	cs, e = Chunks([]Doc{{ID: "b", Text: strings.Join(words, " ")}}, 8)
	if e != nil {
		t.Fatal(e)
	}
	if cs[0].Text != "a b c d e f g h" || cs[1].Text != "h i j k l m n o" {
		t.Fatal(cs)
	}
}
func TestNormalizeAndReject(t *testing.T) {
	v, e := Normalize([]float32{3, 4, 99}, 2)
	if e != nil || math.Abs(float64(v[0])-.6) > 1e-6 {
		t.Fatal(v, e)
	}
	for _, v := range [][]float32{{0, 0}, {float32(math.NaN()), 1}} {
		if _, e = Normalize(v, 2); e == nil {
			t.Fatal("invalid accepted")
		}
	}
	if _, e = Normalize([]float32{1}, 2); e == nil {
		t.Fatal("dim accepted")
	}
}
func TestExactDedup(t *testing.T) {
	cs := []Chunk{{DocID: "a"}, {DocID: "a"}, {DocID: "b"}}
	ds := Exact(cs, [][]float32{{1, 0}, {.8, .2}, {0, 1}}, []float32{1, 0})
	if len(ds) != 2 || ds[0] != "a" {
		t.Fatal(ds)
	}
}
func TestBootstrapPairing(t *testing.T) {
	a := []Record{{QID: "a", Metrics: Metrics{NDCG10: .7}}, {QID: "b", Metrics: Metrics{NDCG10: .3}}}
	b := []Record{{QID: "a", Metrics: Metrics{NDCG10: .6}}, {QID: "b", Metrics: Metrics{NDCG10: .2}}}
	r := Bootstrap(a, b)
	if math.Abs(r["difference"].(float64)-.1) > 1e-9 {
		t.Fatal(r)
	}
	b[0].QID = "x"
	if Bootstrap(a, b)["error"] == nil {
		t.Fatal("pairing")
	}
}
func TestCacheCrashRecovery(t *testing.T) {
	old, _ := os.Getwd()
	dir := t.TempDir()
	os.Chdir(dir)
	defer os.Chdir(old)
	m := Model{ID: "test", Name: "test", Native: 2}
	os.MkdirAll("data/cache", 0755)
	f, _ := os.Create("data/cache/test-2.bin")
	f.Write([]byte(strings.Repeat("a", 64)))
	binary.Write(f, binary.LittleEndian, []float32{1, 0})
	f.Write([]byte("partial"))
	f.Close()
	c, e := NewCache(m, 2)
	if e != nil || len(c.values) != 1 {
		t.Fatal(e)
	}
	st, _ := os.Stat(filepath.Join(dir, "data/cache/test-2.bin"))
	if st.Size() != 72 {
		t.Fatal(st.Size())
	}
}
func TestIntegrationLive(t *testing.T) {
	if os.Getenv("LIVE_INTEGRATION") != "1" {
		t.Skip("LIVE_INTEGRATION=1 enables actual turbopuffer calls")
	}
	if e := Integration(); e != nil {
		t.Fatal(e)
	}
}
func TestBudgetFailsClosed(t *testing.T) {
	old, _ := os.Getwd()
	os.Chdir(t.TempDir())
	defer os.Chdir(old)
	if e := Reserve(119); e != nil {
		t.Fatal(e)
	}
	if e := Reserve(2); e == nil {
		t.Fatal("overspend accepted")
	}
	os.WriteFile("data/budget.json", []byte("broken"), 0644)
	if e := Reserve(.01); e == nil {
		t.Fatal("corrupt ledger accepted")
	}
}
