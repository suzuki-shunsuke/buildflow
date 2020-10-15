#!/usr/bin/env bash

set -eu

cd "$(dirname "$0")/.."

tempdir=$(mktemp -d)
go build -o "$tempdir/buildflow" ./cmd/buildflow
export PATH="$tempdir:$PATH"
cd examples
# command -v buildflow
go test -race -covermode=atomic ./...
rm -R "$tempdir"
