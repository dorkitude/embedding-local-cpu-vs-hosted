#!/usr/bin/env bash
set -euo pipefail
for model in openai-large openai-small; do
 .local/study matrix --model "$model" --run pilot1 --pilot
 .local/study matrix --model "$model" --run r20261012a
done
