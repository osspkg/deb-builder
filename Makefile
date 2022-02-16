SHELL=/bin/bash

.PHONY: new-conf
new-conf:
	go run -race cmd/deb-builder/main.go new-conf

.PHONY: build
build:
	bash scripts/build.sh back

.PHONY: linter
linter:
	bash scripts/linter.sh

.PHONY: tests
tests:
	bash scripts/tests.sh

.PHONY: ci
ci:
	bash scripts/ci.sh

install: build
	@GO111MODULE=on GOOS=linux GOARCH=amd64 go build -o $(GOPATH)/bin/deb-builder ./cmd/deb-builder/