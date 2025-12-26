#!/bin/bash
set -e
cd "$(dirname "$0")/.."
go build -o bin/hub-tui ./cmd/hub-tui
echo "Built: bin/hub-tui"
