#!/bin/bash

cd $PWD

go clean -testcache
go test -v -race ./...