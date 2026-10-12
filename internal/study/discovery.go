package study

import (
	"fmt"
	"os"
	"time"
)

func Discover() error {
	receipts := []map[string]any{}
	for _, x := range []struct{ Name, URL, Key string }{
		{"openai", "https://openai.int.exe.xyz/v1/models", "OPENAI_API_KEY"},
		{"fireworks", "https://fireworks.int.exe.xyz/inference/v1/models", "FIREWORKS_API_KEY"},
		{"openrouter", "https://openrouter.int.exe.xyz/api/v1/embeddings/models", "OPENROUTER_API_KEY"},
		{"google-embeddinggemma-2", "https://huggingface.int.exe.xyz/api/models/google/embeddinggemma-2", ""},
	} {
		var out map[string]any
		e := Request("GET", x.URL, os.Getenv(x.Key), nil, &out, 0, "discover:"+x.Name)
		rec := map[string]any{"provider": x.Name, "url": x.URL, "utc": time.Now().UTC()}
		if e != nil {
			rec["error"] = e.Error()
		} else {
			rec["response"] = out
		}
		receipts = append(receipts, rec)
		fmt.Println("discovered", x.Name, e)
	}
	return Save("results/discovery.json", receipts)
}
