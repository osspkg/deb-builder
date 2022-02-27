#!/bin/bash

PRJROOT="$PWD"
GOMAIN="$PWD/cmd/deb-builder"

cd $PWD

rm -rf $PRJROOT/build/bin/deb-builder_$1

go generate ./...

GO111MODULE=on GOOS=linux GOARCH=$1 go build -o $PRJROOT/build/bin/deb-builder_$1 $GOMAIN
