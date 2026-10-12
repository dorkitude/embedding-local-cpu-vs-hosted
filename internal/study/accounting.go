package study

import (
	"math"
	"os"
)

// WriteCost applies the captured official list-price calculator to one write.
// API billing bytes exclude batch discounts (receipts/turbopuffer-write.md).
func WriteCost(bytes float64) (discounted, gross float64) {
	if bytes <= 0 {
		return 0, 0
	}
	bytes = math.Max(bytes, 10000)
	discount := math.Min(.5, math.Max(0, (math.Log10(bytes/1000)-1)*.2))
	gross = bytes / 1e9 * 2
	return gross * (1 - discount), gross
}

// ScanJournal reads a compressed historical prefix followed by a live suffix.
// Unlike corpus files, journals may legitimately have both after resuming.
// Rotate only with all writers stopped; gzip -n removes the original prefix.
func ScanJournal(path string, fn func([]byte) error) error {
	found := false
	for _, p := range []string{path + ".gz", path} {
		if _, e := os.Stat(p); os.IsNotExist(e) {
			continue
		} else if e != nil {
			return e
		}
		found = true
		if e := Scan(p, fn); e != nil {
			return e
		}
	}
	if !found {
		return os.ErrNotExist
	}
	return nil
}
