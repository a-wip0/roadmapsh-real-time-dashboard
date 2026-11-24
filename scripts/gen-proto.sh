#!/usr/bin/env bash
# Regenerates the Go (backend/gen) and Python (simulator/gen) gRPC bindings.
# Requires: python3 with grpcio-tools (pip install grpcio-tools) and, on PATH,
#   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
#   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
set -euo pipefail
cd "$(dirname "$0")/.."
PY=${PYTHON:-python3}

rm -rf backend/gen simulator/gen
mkdir -p backend/gen simulator/gen

"$PY" -m grpc_tools.protoc -Iproto \
  --go_out=backend/gen --go_opt=paths=source_relative \
  --go-grpc_out=backend/gen --go-grpc_opt=paths=source_relative \
  --python_out=simulator/gen --grpc_python_out=simulator/gen \
  proto/leaderboard/v1/leaderboard.proto
echo "generated backend/gen and simulator/gen"
