#!/usr/bin/env sh
set -eu

mkdir -p internal/protocol/pb

protoc \
  --proto_path=api/proto \
  --go_out=internal/protocol/pb \
  --go_opt=paths=source_relative \
  kick.proto
