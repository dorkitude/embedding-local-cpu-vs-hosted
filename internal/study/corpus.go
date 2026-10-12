package study

import (
	"bufio"
	"compress/gzip"
	"container/heap"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

func HFBase() string {
	if b := os.Getenv("HF_ENDPOINT"); b != "" {
		return strings.TrimRight(b, "/")
	}
	return "https://huggingface.int.exe.xyz"
}

const CorpusRev = "d921ec7e349ce0d28daf30b2da9da5ee698bef0d"
const QueryRev = "5be20db9509754dadad47689368639fcec739c00"

type Doc struct {
	ID    string `json:"docid"`
	Title string `json:"title"`
	Text  string `json:"text"`
}
type pick struct {
	D Doc
	H string
}
type picks []pick

func (p picks) Len() int           { return len(p) }
func (p picks) Less(i, j int) bool { return p[i].H > p[j].H }
func (p picks) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p *picks) Push(x any)        { *p = append(*p, x.(pick)) }
func (p *picks) Pop() any          { x := *p; n := len(x); v := x[n-1]; *p = x[:n-1]; return v }
func Hash(b []byte) string         { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func Save(path string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	return Atomic(path, append(b, '\n'))
}
func Atomic(path string, b []byte) error {
	if e := os.MkdirAll(filepath.Dir(path), 0755); e != nil {
		return e
	}
	if e := os.WriteFile(path+".tmp", b, 0644); e != nil {
		return e
	}
	return os.Rename(path+".tmp", path)
}
func Download(u, path string) error {
	if s, e := os.Stat(path); e == nil && s.Size() > 0 {
		return nil
	}
	c := http.Client{Timeout: 20 * time.Minute}
	for n := 0; n < 3; n++ {
		e := func() error {
			r, e := c.Get(u)
			if e != nil {
				return e
			}
			defer r.Body.Close()
			if r.StatusCode != 200 {
				return fmt.Errorf("download %s: %s", u, r.Status)
			}
			f, e := os.Create(path + ".tmp")
			if e != nil {
				return e
			}
			_, e = io.Copy(f, r.Body)
			ce := f.Close()
			if e != nil {
				return e
			}
			if ce != nil {
				return ce
			}
			return os.Rename(path+".tmp", path)
		}()
		if e == nil {
			return nil
		} else if n == 2 {
			return e
		}
		time.Sleep(time.Duration(n+1) * time.Second)
	}
	return nil
}
func Scan(path string, fn func([]byte) error) error {
	f, e := os.Open(path)
	if os.IsNotExist(e) {
		path += ".gz"
		f, e = os.Open(path)
	}
	if e != nil {
		return e
	}
	defer f.Close()
	var r io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		z, e := gzip.NewReader(f)
		if e != nil {
			return e
		}
		defer z.Close()
		r = z
	}
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 65536), 16<<20)
	for s.Scan() {
		if e := fn(s.Bytes()); e != nil {
			return e
		}
	}
	return s.Err()
}
func Acquire() error {
	os.MkdirAll("data/raw", 0755)
	base := HFBase() + "/datasets/miracl/miracl/resolve/" + QueryRev + "/miracl-v1.0-en/"
	for _, x := range []struct{ remote, local string }{{"topics/topics.miracl-v1.0-en-dev.tsv", "queries.tsv"}, {"qrels/qrels.miracl-v1.0-en-dev.tsv", "qrels.tsv"}} {
		if e := Download(base+x.remote, "data/"+x.local); e != nil {
			return e
		}
	}
	jobs := make(chan int, 66)
	for i := 0; i < 66; i++ {
		jobs <- i
	}
	close(jobs)
	var wg sync.WaitGroup
	errs := make(chan error, 66)
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				p := fmt.Sprintf("data/raw/docs-%d.jsonl.gz", i)
				u := fmt.Sprintf("%s/datasets/miracl/miracl-corpus/resolve/%s/miracl-corpus-v1.0-en/docs-%d.jsonl.gz", HFBase(), CorpusRev, i)
				if e := Download(u, p); e != nil {
					errs <- e
				} else {
					fmt.Printf("acquired shard %d\n", i)
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		return e
	}
	return nil
}
func Prepare() error {
	judged := map[string]bool{}
	if e := Scan("data/qrels.tsv", func(b []byte) error {
		f := strings.Fields(string(b))
		if len(f) != 4 {
			return fmt.Errorf("invalid qrel")
		}
		judged[f[2]] = true
		return nil
	}); e != nil {
		return e
	}
	chosen := map[string]Doc{}
	p := &picks{}
	heap.Init(p)
	keep := 100000 - len(judged)
	count := 0
	checks := map[string]string{}
	for i := 0; i < 66; i++ {
		path := fmt.Sprintf("data/raw/docs-%d.jsonl.gz", i)
		f, e := os.Open(path)
		if e != nil {
			return e
		}
		h := sha256.New()
		_, e = io.Copy(h, f)
		f.Close()
		if e != nil {
			return e
		}
		checks[path] = hex.EncodeToString(h.Sum(nil))
		e = Scan(path, func(b []byte) error {
			var d Doc
			if e := json.Unmarshal(b, &d); e != nil {
				return e
			}
			count++
			if judged[d.ID] {
				chosen[d.ID] = d
				return nil
			}
			k := Hash([]byte("embedding-study-20261012:" + d.ID))
			if p.Len() < keep {
				heap.Push(p, pick{d, k})
			} else if k < (*p)[0].H {
				heap.Pop(p)
				heap.Push(p, pick{d, k})
			}
			return nil
		})
		if e != nil {
			return e
		}
		fmt.Printf("scanned %d: %d passages\n", i, count)
	}
	if len(chosen) != len(judged) {
		return fmt.Errorf("missing judged documents: have %d want %d", len(chosen), len(judged))
	}
	ds := make([]Doc, 0, 100000)
	for _, d := range chosen {
		ds = append(ds, d)
	}
	sort.Slice(ds, func(i, j int) bool { return ds[i].ID < ds[j].ID })
	sort.Slice(*p, func(i, j int) bool { return (*p)[i].H < (*p)[j].H })
	for _, v := range *p {
		ds = append(ds, v.D)
	}
	if len(ds) != 100000 {
		return fmt.Errorf("count %d", len(ds))
	}
	out, e := os.Create("data/corpus.jsonl")
	if e != nil {
		return e
	}
	enc := json.NewEncoder(out)
	os.MkdirAll("results/manifests", 0755)
	af, e := os.Create("results/manifests/attribution.jsonl.gz")
	if e != nil {
		return e
	}
	az := gzip.NewWriter(af)
	ae := json.NewEncoder(az)
	h := sha256.New()
	for i, d := range ds {
		b, _ := json.Marshal(d)
		h.Write(append(b, '\n'))
		if e = enc.Encode(d); e != nil {
			return e
		}
		page := "https://en.wikipedia.org/wiki/" + url.PathEscape(strings.ReplaceAll(d.Title, " ", "_"))
		if e = ae.Encode(map[string]any{"docid": d.ID, "title": d.Title, "source_url": page, "history_url": page + "?action=history", "authors": "Wikipedia contributors", "license": "CC-BY-SA-3.0", "text_sha256": Hash([]byte(d.Text)), "secondary": i < 20000}); e != nil {
			return e
		}
	}
	out.Close()
	az.Close()
	af.Close()
	q, _ := os.ReadFile("data/queries.tsv")
	qr, _ := os.ReadFile("data/qrels.tsv")
	return Save("results/manifests/corpus.json", map[string]any{"frozen_utc": time.Now().UTC(), "corpus_revision": CorpusRev, "query_revision": QueryRev, "seed": "embedding-study-20261012", "source_passages": count, "selected": len(ds), "secondary": 20000, "judged_unique": len(judged), "corpus_sha256": hex.EncodeToString(h.Sum(nil)), "queries_sha256": Hash(q), "qrels_sha256": Hash(qr), "shards_sha256": checks, "source_dump": "https://archive.org/download/enwiki-20190201/enwiki-20190201-pages-articles-multistream.xml.bz2", "license": "CC-BY-SA-3.0 text; Apache-2.0 MIRACL contributions"})
}
