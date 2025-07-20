#!/bin/bash
set -e

go mod tidy
if ! command -v staticcheck &> /dev/null; then
  go install honnef.co/go/tools/cmd/staticcheck@latest
fi
echo staticcheck 
staticcheck ./...
echo go test
go test ./...
echo go build
go build -v -o pihole-sync ./cmd/main.go
