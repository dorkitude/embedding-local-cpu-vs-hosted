#!/usr/bin/env bash
set -euo pipefail
.local/study matrix --model qwen8b --run pilot1 --pilot
exec .local/study matrix --model qwen8b --run r20261012a
