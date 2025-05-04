
.PHONY: install
install:
	go install go.osspkg.com/goppy/v2/cmd/goppy@latest

.PHONY: setup
setup:
	goppy setup-lib

.PHONY: lint
lint:
	goppy lint

.PHONY: license
license:
	goppy license

.PHONY: build
build:
	goppy build --arch=amd64

.PHONY: tests
tests:
	goppy test

.PHONY: pre-commite
pre-commit: license setup lint build tests

.PHONY: ci
ci: install setup lint build tests

deb:
	deb-builder build

local: build
	cp ./build/deb-builder_amd64 $(GOPATH)/bin/deb-builder