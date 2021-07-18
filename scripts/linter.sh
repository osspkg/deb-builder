#!/bin/bash

GO_FILES=$(find . -name '*.go' | grep -vE 'vendor|easyjson|static')
TOOLS_BIN=$PWD/.tools

curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $TOOLS_BIN v1.38.0

cd $PWD

go generate ./...
goimports -w $GO_FILES
go fmt ./...
$TOOLS_BIN/golangci-lint -v run ./...