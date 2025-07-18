#!/bin/bash
set -e

go mod tidy
if ! command -v staticcheck &> /dev/null; then
  go install honnef.co/go/tools/cmd/staticcheck@latest
fi
staticcheck ./...
go build -v -o pihole-sync ./cmd/main.go
