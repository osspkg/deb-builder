#!/bin/bash

PRJROOT="$PWD"
GOMAIN="$PWD/cmd/deb-builder"

cd $PWD

back() {
  rm -rf $PRJROOT/build/bin/deb-builder_*

  go generate ./...

  GO111MODULE=on GOOS=linux GOARCH=amd64 go build -o $PRJROOT/build/bin/deb-builder_amd64 $GOMAIN
}

case $1 in
back)
  back
  ;;
*)
  echo "back"
  ;;
esac
