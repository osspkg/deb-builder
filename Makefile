SHELL=/bin/bash

.PHONY: new-conf
new-conf:
	go run -race cmd/deb-builder/main.go new-conf

.PHONY: build
build:
	bash scripts/build.sh amd64

.PHONY: linter
linter:
	bash scripts/linter.sh

.PHONY: tests
tests:
	bash scripts/tests.sh

.PHONY: ci
ci:
	bash scripts/ci.sh

deb:
	deb-builder build

install: build
	cp ./build/bin/deb-builder_amd64 $(GOPATH)/bin/deb-builder