// Independent streaming audit of the full-corpus smallest-hash selection.
package main

import (
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type Attribution struct {
	ID        string `json:"docid"`
	Title     string `json:"title"`
	SHA       string `json:"text_sha256"`
	Secondary bool   `json:"secondary"`
}

func hash(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run() error {
	selected := map[string]Attribution{}
	judged := map[string]bool{}
	f, e := os.Open("results/manifests/qrels.tsv")
	if e != nil {
		return e
	}
	s := bufio.NewScanner(f)
	for s.Scan() {
		cols := strings.Fields(s.Text())
		judged[cols[2]] = true
	}
	if e = s.Err(); e != nil {
		return e
	}
	f.Close()
	f, e = os.Open("results/manifests/attribution.jsonl.gz")
	if e != nil {
		return e
	}
	z, e := gzip.NewReader(f)
	if e != nil {
		return e
	}
	dec := json.NewDecoder(z)
	threshold, secondaryThreshold := "", ""
	secondaryCount := 0
	for {
		var a Attribution
		e = dec.Decode(&a)
		if e == io.EOF {
			break
		}
		if e != nil {
			return e
		}
		if _, ok := selected[a.ID]; ok {
			return fmt.Errorf("duplicate attribution")
		}
		selected[a.ID] = a
		if a.Secondary {
			secondaryCount++
		}
		if !judged[a.ID] {
			h := hash("embedding-study-20261012:" + a.ID)
			if h > threshold {
				threshold = h
			}
			if a.Secondary && h > secondaryThreshold {
				secondaryThreshold = h
			}
		}
	}
	z.Close()
	f.Close()
	if len(selected) != 100000 || secondaryCount != 20000 {
		return fmt.Errorf("sample sizes")
	}
	for id := range judged {
		a, ok := selected[id]
		if !ok || !a.Secondary {
			return fmt.Errorf("judged doc missing from nested sample")
		}
	}
	for id, a := range selected {
		if !judged[id] && a.Secondary != (hash("embedding-study-20261012:"+id) <= secondaryThreshold) {
			return fmt.Errorf("secondary threshold violation")
		}
	}
	count, matched := 0, 0
	seen := map[string]bool{}
	for shard := 0; shard < 66; shard++ {
		f, e = os.Open(fmt.Sprintf("data/raw/docs-%d.jsonl.gz", shard))
		if e != nil {
			return e
		}
		z, e = gzip.NewReader(f)
		if e != nil {
			return e
		}
		dec = json.NewDecoder(z)
		for {
			var d struct {
				ID    string `json:"docid"`
				Title string `json:"title"`
				Text  string `json:"text"`
			}
			e = dec.Decode(&d)
			if e == io.EOF {
				break
			}
			if e != nil {
				return e
			}
			count++
			a, ok := selected[d.ID]
			if ok {
				if seen[d.ID] {
					return fmt.Errorf("selected doc duplicated in source")
				}
				seen[d.ID] = true
				matched++
				if a.Title != d.Title || a.SHA != hash(d.Text) {
					return fmt.Errorf("source attribution mismatch: %s", d.ID)
				}
			} else if hash("embedding-study-20261012:"+d.ID) <= threshold {
				return fmt.Errorf("eligible distractor omitted: %s", d.ID)
			}
		}
		z.Close()
		f.Close()
		if shard%10 == 0 {
			fmt.Printf("independent source audit: shard %d, %d passages\n", shard, count)
		}
	}
	if count != 32893221 || matched != 100000 {
		return fmt.Errorf("source/sample counts %d/%d", count, matched)
	}
	out := map[string]any{"status": "passed", "utc": time.Now().UTC(), "source_passages": count, "selected": matched, "secondary": secondaryCount, "judged_unique": len(judged), "primary_max_distractor_hash": threshold, "secondary_max_distractor_hash": secondaryThreshold, "checks": []string{"Every selected doc appears exactly once in source", "Title and UTF-8 text checksum match attribution manifest", "Every omitted source doc has hash above selected threshold", "Both samples include every positive and negative judged doc", "Secondary sample equals smallest-hash nested distractors"}}
	b, _ := json.MarshalIndent(out, "", "  ")
	if e = os.WriteFile("results/independent-sampling-audit.json", append(b, '\n'), 0644); e != nil {
		return e
	}
	fmt.Printf("Independent sample audit passed: %d source passages, %d selected\n", count, matched)
	return nil
}
