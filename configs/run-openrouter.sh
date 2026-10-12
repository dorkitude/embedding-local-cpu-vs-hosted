#!/usr/bin/env bash
set -euo pipefail
for model in gemini2 bge; do
 .local/study matrix --model "$model" --run pilot2 --pilot
 .local/study matrix --model "$model" --run r20261012a
done
