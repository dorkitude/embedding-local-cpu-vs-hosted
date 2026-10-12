package study

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestEmbeddingAPIContract(t *testing.T) {
	old, _ := os.Getwd()
	if e := os.Chdir(t.TempDir()); e != nil {
		t.Fatal(e)
	}
	defer os.Chdir(old)
	response := map[string]any{"data": []any{map[string]any{"index": 1, "embedding": []float64{0, 1}}, map[string]any{"index": 0, "embedding": []float64{1, 0}}}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Errorf("path %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["dimensions"] != float64(2) {
			t.Errorf("API dimensions not requested")
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	m := Model{ID: "fixture", Name: "fixture", Base: server.URL, Native: 3, Mode: "api"}
	vs, e := Embed(m, []string{"one", "two"}, 2)
	if e != nil {
		t.Fatal(e)
	}
	if vs[0][0] != 1 || vs[1][1] != 1 {
		t.Fatal("response indexes ignored")
	}
	response = map[string]any{"data": []any{map[string]any{"index": 0, "embedding": []float64{1, 0, 0}}, map[string]any{"index": 1, "embedding": []float64{0, 1, 0}}}}
	if _, e = Embed(m, []string{"one", "two"}, 2); e == nil {
		t.Fatal("wrong provider dimension accepted")
	}
	response = map[string]any{"data": []any{map[string]any{"index": 0, "embedding": []float64{1, 0}}, map[string]any{"index": 0, "embedding": []float64{0, 1}}}}
	if _, e = Embed(m, []string{"one", "two"}, 2); e == nil {
		t.Fatal("duplicate provider index accepted")
	}
}
