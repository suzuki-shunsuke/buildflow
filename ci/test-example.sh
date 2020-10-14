#!/usr/bin/env bash

set -eu

cd "$(dirname "$0")/.."

go build -o dist/buildflow ./cmd/buildflow
export PATH="$PWD/dist:$PATH"
cd examples
go test -race -covermode=atomic ./...
rm -R ../dist
