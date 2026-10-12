"""Standalone CPU precision/batch benchmark, fixed synthetic input; no retrieval tuning."""
import json
import os
import time

os.environ.setdefault("HF_ENDPOINT", "https://huggingface.int.exe.xyz")
os.environ.setdefault("HF_HUB_DISABLE_XET", "1")
import numpy as np
import torch
from sentence_transformers import SentenceTransformer

torch.set_num_threads(8)
torch.set_num_interop_threads(1)
started = time.perf_counter()
# Load original checkpoint values directly into FP32; importing the production
# BF16 sidecar and widening its rounded weights does not provide a FP32 reference.
model = SentenceTransformer(
    "google/embeddinggemma-2",
    revision="914f7f89142e33e77833254d9c9b90c3cef7303b",
    token=os.environ.get("HF_TOKEN", False),
    device="cpu",
    config_kwargs={"vision_config": None, "audio_config": None},
    model_kwargs={"dtype": torch.float32},
)
load_seconds = time.perf_counter() - started
assert next(model.parameters()).dtype == torch.float32
text = "title: Aurora | text: " + "Charged particles from the sun interact with Earth's magnetic field to produce polar auroras. " * 12
rows = []
references = {}
for name, dtype in [("float32", torch.float32), ("bfloat16", torch.bfloat16)]:
    model.to(dtype)
    for batch in [1, 8, 32]:
        for repetition in range(3):
            started = time.perf_counter()
            v = model.encode([text] * batch, batch_size=batch, normalize_embeddings=True, show_progress_bar=False)
            seconds = time.perf_counter() - started
            if name == "float32":
                references[batch] = v.astype(np.float32)
            cosine = (v.astype(np.float32) * references[batch]).sum(axis=1) / (np.linalg.norm(v.astype(np.float32), axis=1) * np.linalg.norm(references[batch], axis=1))
            row = {"dtype": name, "batch": batch, "repetition": repetition, "seconds": seconds, "documents_per_second": batch/seconds, "minimum_cosine_to_fp32": float(cosine.min()), "finite": bool(np.isfinite(v).all()), "model_load_seconds": load_seconds, "model_tokens_per_document": len(model.tokenizer(text)["input_ids"])}
            rows.append(row)
            print(json.dumps(row), flush=True)
output_prefix = os.environ.get("BENCHMARK_OUTPUT_PREFIX", "results/cpu-benchmark-confirmation")
with open(output_prefix + ".json", "w") as f:
    json.dump({"rows": rows, "threads": torch.get_num_threads(), "cuda": torch.cuda.is_available(), "input": text, "fp32_reference": "checkpoint loaded directly as float32 before BF16 conversion", "load_scope": "process/model initialization from cached files; not cold physical disk"}, f, indent=2)
scaling = []
for threads in [1, 2, 4, 8]:
    torch.set_num_threads(threads)
    for batch in [8, 32]:
        for repeat in range(2):
            started = time.perf_counter()
            model.encode([text]*batch, batch_size=batch, normalize_embeddings=True, show_progress_bar=False)
            row = {"threads": threads, "batch": batch, "repetition": repeat, "seconds": time.perf_counter()-started, "dtype": "bfloat16"}
            scaling.append(row)
            print(json.dumps(row), flush=True)
with open(output_prefix + "-thread-scaling.json", "w") as f:
    json.dump(scaling, f, indent=2)
