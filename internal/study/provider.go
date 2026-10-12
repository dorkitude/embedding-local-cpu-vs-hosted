package study

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Model struct {
	ID, Provider, Name, Base, KeyEnv, Revision, Mode string
	Native                                           int
	Price                                            float64
	Batch, Workers                                   int
}

var Models = []Model{
	{"gemma2", "local", "google/embeddinggemma-2", "http://127.0.0.1:8765/v1", "", "914f7f89142e33e77833254d9c9b90c3cef7303b", "mrl", 768, 0, 32, 1},
	{"openai-large", "openai", "text-embedding-3-large", "https://openai.int.exe.xyz/v1", "OPENAI_API_KEY", "provider-unpinned", "api", 3072, .13, 64, 4},
	{"qwen8b", "fireworks", "accounts/fireworks/models/qwen3-embedding-8b", "https://fireworks.int.exe.xyz/inference/v1", "FIREWORKS_API_KEY", "provider-unpinned", "mrl", 4096, .15, 32, 4},
	{"openai-small", "openai", "text-embedding-3-small", "https://openai.int.exe.xyz/v1", "OPENAI_API_KEY", "provider-unpinned", "api", 1536, .02, 64, 4},
	{"gemini2", "openrouter", "google/gemini-embedding-2", "https://openrouter.int.exe.xyz/api/v1", "OPENROUTER_API_KEY", "provider-unpinned", "mrl", 3072, .20, 32, 4},
	{"bge", "openrouter", "baai/bge-m3", "https://openrouter.int.exe.xyz/api/v1", "OPENROUTER_API_KEY", "provider-unpinned", "exploratory-prefix", 1024, .01, 32, 4},
}

func (m Model) Endpoint() string {
	if b := os.Getenv(strings.ToUpper(m.Provider) + "_BASE_URL"); b != "" {
		return strings.TrimRight(b, "/")
	}
	return m.Base
}
func (m Model) Format(text, title string, query bool) string {
	if query {
		switch m.ID {
		case "gemma2", "gemini2":
			return "task: search result | query: " + text
		case "qwen8b":
			return "Instruct: Given a web search query, retrieve relevant passages that answer the query\nQuery: " + text
		default:
			return text
		}
	}
	if m.ID == "gemma2" || m.ID == "gemini2" {
		return "title: " + title + " | text: " + text
	}
	return title + "\n" + text
}

var client = &http.Client{Timeout: 180 * time.Second}
var logMu sync.Mutex

func Journal(path string, x any) error {
	logMu.Lock()
	defer logMu.Unlock()
	os.MkdirAll(filepath.Dir(path), 0755)
	f, e := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if e != nil {
		return e
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(x)
}

// Conservative byte-as-token reservation charged on EACH attempt, including failed calls.
func Reserve(cost float64) error {
	os.MkdirAll("data", 0755)
	lock, e := os.OpenFile("data/budget.lock", os.O_CREATE|os.O_RDWR, 0644)
	if e != nil {
		return e
	}
	defer lock.Close()
	if e = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); e != nil {
		return e
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	var used float64
	b, e := os.ReadFile("data/budget.json")
	if e == nil {
		if e = json.Unmarshal(b, &used); e != nil {
			return fmt.Errorf("invalid budget ledger: %w", e)
		}
	} else if !os.IsNotExist(e) {
		return e
	}
	if used+cost > 120 {
		return fmt.Errorf("embedding budget cap: %.4f + %.4f > 120 USD", used, cost)
	}
	return Save("data/budget.json", used+cost)
}
func Request(method, u, key string, payload any, out any, charge float64, tag string) error {
	var b []byte
	var e error
	if payload != nil {
		b, e = json.Marshal(payload)
		if e != nil {
			return e
		}
	}
	for attempt := 1; attempt <= 3; attempt++ {
		if charge > 0 {
			if e = Reserve(charge); e != nil {
				return e
			}
		}
		req, e := http.NewRequest(method, u, bytes.NewReader(b))
		if e != nil {
			return e
		}
		req.Header.Set("Content-Type", "application/json")
		if key != "" {
			req.Header.Set("Authorization", "Bearer "+key)
		}
		start := time.Now()
		r, err := client.Do(req)
		status := 0
		var raw []byte
		if err == nil {
			status = r.StatusCode
			raw, err = io.ReadAll(io.LimitReader(r.Body, 128<<20))
			r.Body.Close()
		}
		rec := map[string]any{"utc": time.Now().UTC(), "tag": tag, "attempt": attempt, "http_status": status, "seconds": time.Since(start).Seconds(), "request_bytes": len(b), "response_bytes": len(raw), "reserved_usd": charge}
		if p, ok := payload.(map[string]any); ok {
			if inputs, ok := p["input"].([]string); ok {
				rec["input_count"] = len(inputs)
			}
		}
		if err == nil && status >= 200 && status < 300 {
			var meta map[string]json.RawMessage
			json.Unmarshal(raw, &meta)
			for _, k := range []string{"usage", "billing", "performance", "timing", "model"} {
				if v, ok := meta[k]; ok {
					rec[k] = v
				}
			}
			if e = Journal("results/requests.jsonl", rec); e != nil {
				return e
			}
			if out != nil {
				return json.Unmarshal(raw, out)
			}
			return nil
		}
		if err == nil {
			err = fmt.Errorf("HTTP %d: %.1600s", status, string(raw))
		}
		if key != "" {
			err = fmt.Errorf("%s", strings.ReplaceAll(err.Error(), key, "[REDACTED]"))
		}
		rec["error"] = err.Error()
		Journal("results/requests.jsonl", rec)
		if attempt == 3 || (status >= 400 && status < 500 && status != 429) {
			return fmt.Errorf("%s: %w", tag, err)
		}
		time.Sleep(time.Duration(attempt*2) * time.Second)
	}
	return fmt.Errorf("request exhausted")
}

type EmbResponse struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Model string         `json:"model"`
	Usage map[string]any `json:"usage"`
}

func Embed(m Model, texts []string, dim int) ([][]float32, error) {
	return EmbedRole(m, texts, dim, false)
}
func EmbedRole(m Model, texts []string, dim int, query bool) ([][]float32, error) {
	p := map[string]any{"model": m.Name, "input": texts, "encoding_format": "float"}

	if m.Mode == "api" {
		p["dimensions"] = dim
	}
	// BGE does not require retrieval instructions. Gemini is explicitly formatted as retrieval task.
	if m.ID == "gemini2" {
		p["dimensions"] = m.Native
	}
	nbytes := 0
	for _, s := range texts {
		nbytes += len(s) + 32
	}
	var r EmbResponse
	e := Request("POST", m.Endpoint()+"/embeddings", os.Getenv(m.KeyEnv), p, &r, float64(nbytes)*m.Price/1e6, "embed:"+m.ID+fmt.Sprint(":", dim))
	if e != nil {
		return nil, e
	}
	if len(r.Data) != len(texts) {
		return nil, fmt.Errorf("provider rows %d expected %d", len(r.Data), len(texts))
	}
	want := m.Native
	if m.Mode == "api" {
		want = dim
	}
	vs := make([][]float32, len(texts))
	seen := map[int]bool{}
	for _, x := range r.Data {
		if x.Index < 0 || x.Index >= len(vs) || seen[x.Index] {
			return nil, fmt.Errorf("invalid response index")
		}
		seen[x.Index] = true
		if len(x.Embedding) != want {
			return nil, fmt.Errorf("returned dim %d want %d", len(x.Embedding), want)
		}
		if _, e := Normalize(x.Embedding, want); e != nil {
			return nil, e
		}
		vs[x.Index] = x.Embedding
	}
	return vs, nil
}
func Normalize(v []float32, d int) ([]float32, error) {
	if len(v) < d || d <= 0 {
		return nil, fmt.Errorf("invalid dimension")
	}
	var norm float64
	for _, x := range v[:d] {
		if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
			return nil, fmt.Errorf("nonfinite vector")
		}
		norm += float64(x) * float64(x)
	}
	if norm == 0 {
		return nil, fmt.Errorf("zero vector")
	}
	norm = math.Sqrt(norm)
	out := make([]float32, d)
	for i := range out {
		out[i] = float32(float64(v[i]) / norm)
	}
	return out, nil
}

type Cache struct {
	query  bool
	m      Model
	dim    int
	path   string
	values map[string][]float32
	mu     sync.Mutex
}

func NewCache(m Model, dim int) (*Cache, error) {
	if m.Mode != "api" {
		dim = m.Native
	}
	p := fmt.Sprintf("data/cache/%s-%d.bin", m.ID, dim)
	os.MkdirAll(filepath.Dir(p), 0755)
	c := &Cache{m: m, dim: dim, path: p, values: map[string][]float32{}}
	f, e := os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0644)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	var offset int64
	for {
		key := make([]byte, 64)
		_, e = io.ReadFull(f, key)
		if e == io.EOF {
			break
		}
		if e != nil {
			f.Truncate(offset)
			break
		}
		v := make([]float32, dim)
		if e = binary.Read(f, binary.LittleEndian, v); e != nil {
			f.Truncate(offset)
			break
		}
		c.values[string(key)] = v
		offset += 64 + int64(dim*4)
	}
	return c, nil
}
func (c *Cache) Key(s string) string {
	precision := ""
	if c.m.ID == "gemma2" {
		precision = "|bf16-torch2.14.1-st6.1.0-b8-t8|"
	}
	return Hash([]byte(precision + c.m.Name + "|" + c.m.Revision + fmt.Sprint(c.query) + "|format-v2|" + fmt.Sprint(c.dim) + "|" + s))
}
func (c *Cache) Get(texts []string, query bool) ([][]float32, error) {
	c.query = query
	missing := []string{}
	keys := map[string]bool{}
	for _, s := range texts {
		k := c.Key(s)
		if _, ok := c.values[k]; !ok && !keys[k] {
			keys[k] = true
			missing = append(missing, s)
		}
	}
	// Sort by length within stable global order for reduced local padding; key identity is text-based.
	sort.SliceStable(missing, func(i, j int) bool { return len(missing[i]) < len(missing[j]) })
	jobs := make(chan []string, len(missing)/c.m.Batch+1)
	for i := 0; i < len(missing); i += c.m.Batch {
		end := min(i+c.m.Batch, len(missing))
		jobs <- missing[i:end]
	}
	close(jobs)
	var wg sync.WaitGroup
	errs := make(chan error, c.m.Workers)
	done := 0
	for w := 0; w < c.m.Workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ss := range jobs {
				vs, e := EmbedRole(c.m, ss, c.dim, query)
				if e != nil {
					errs <- e
					return
				}
				c.mu.Lock()
				f, e := os.OpenFile(c.path, os.O_WRONLY|os.O_APPEND, 0644)
				if e == nil {
					for i, s := range ss {
						k := c.Key(s)
						if _, e = f.Write([]byte(k)); e != nil {
							break
						}
						if e = binary.Write(f, binary.LittleEndian, vs[i]); e != nil {
							break
						}
						c.values[k] = vs[i]
					}
					if e == nil {
						e = f.Sync()
					}
					f.Close()
				}
				done += len(ss)
				if done%1024 < c.m.Batch {
					fmt.Printf("embed %s d%d %d/%d new\n", c.m.ID, c.dim, done, len(missing))
				}
				c.mu.Unlock()
				if e != nil {
					errs <- e
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		return nil, e
	}
	out := make([][]float32, len(texts))
	for i, s := range texts {
		v, ok := c.values[c.Key(s)]
		if !ok {
			return nil, fmt.Errorf("cache missing")
		}
		out[i] = v
	}
	return out, nil
}
func Preflight() error {
	os.MkdirAll("results/preflight", 0755)
	all := []map[string]any{}
	for _, m := range Models {
		t := time.Now()
		v, e := Embed(m, []string{m.Format("What causes auroras?", "", true), m.Format("Auroras arise from charged solar particles.", "Aurora", false)}, m.Native)
		rec := map[string]any{"model": m, "utc": time.Now().UTC(), "seconds": time.Since(t).Seconds(), "status": "passed"}
		if e != nil {
			rec["status"] = "blocked"
			rec["reason"] = e.Error()
		} else {
			rec["returned_dimension"] = len(v[0])
			rec["rows"] = len(v)
		}
		all = append(all, rec)
		fmt.Println(m.ID, rec["status"])
	}
	return Save("results/preflight/models.json", all)
}
