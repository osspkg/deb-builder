#!/bin/bash

TOOLS_BIN=$PWD/.tools

cd $PWD

rm -rf $TOOLS_BIN
mkdir -p $TOOLS_BIN

curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $TOOLS_BIN v1.50.0
GO111MODULE=off GOBIN=$TOOLS_BIN go get github.com/mattn/goveralls

go mod download
$TOOLS_BIN/golangci-lint -v run ./...
go build -race -v ./...
go test -race -v -covermode=atomic -coverprofile=coverage.out ./...
$TOOLS_BIN/goveralls -coverprofile=coverage.out -repotoken $COVERALLS_TOKEN