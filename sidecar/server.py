"""Local CPU embedding sidecar. No corpus preparation or retrieval in Python."""
import os
os.environ.setdefault("HF_ENDPOINT", "https://huggingface.int.exe.xyz")
os.environ.setdefault("HF_HUB_DISABLE_XET", "1")
import time
import threading
import torch
from sentence_transformers import SentenceTransformer
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import uvicorn

torch.set_num_threads(8)
torch.set_num_interop_threads(1)
started = time.perf_counter()
model = SentenceTransformer(
    "google/embeddinggemma-2",
    revision="914f7f89142e33e77833254d9c9b90c3cef7303b",
    device="cpu",
    config_kwargs={"vision_config": None, "audio_config": None},
    model_kwargs={"dtype": torch.bfloat16},
)
load_seconds = time.perf_counter() - started
app = FastAPI()
lock = threading.Lock()

class Request(BaseModel):
    input: list[str]
    model: str = "google/embeddinggemma-2"
    encoding_format: str = "float"

@app.get("/health")
def health():
    return {"model": "google/embeddinggemma-2", "revision": "914f7f89142e33e77833254d9c9b90c3cef7303b", "load_seconds": load_seconds, "threads": torch.get_num_threads(), "dtype": str(next(model.parameters()).dtype), "cuda": torch.cuda.is_available(), "max_seq_length": model.max_seq_length, "inference_batch_size": 8}

@app.post("/v1/embeddings")
def embeddings(req: Request):
    with lock:
        t = time.perf_counter()
        counts = [len(model.tokenizer(x, truncation=False)["input_ids"]) for x in req.input]
        if max(counts, default=0) > 8192:
            raise HTTPException(400, "Context overflow; truncation forbidden")
        vectors = model.encode(req.input, batch_size=min(8, len(req.input)), normalize_embeddings=True, show_progress_bar=False)
        if not torch.isfinite(torch.from_numpy(vectors)).all():
            raise HTTPException(500, "Nonfinite embedding")
        return {"model": req.model, "data": [{"index": i, "embedding": x.tolist()} for i, x in enumerate(vectors)], "usage": {"prompt_tokens": sum(counts), "total_tokens": sum(counts)}, "timing": {"seconds": time.perf_counter()-t, "max_tokens": max(counts), "load_seconds": load_seconds}}

if __name__ == "__main__":
    uvicorn.run(app, host="127.0.0.1", port=8765)
