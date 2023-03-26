
.PHONY: install
install:
	go install github.com/dewep-online/devtool@latest

.PHONY: setup
setup:
	devtool setup-lib

.PHONY: lint
lint:
	devtool lint

.PHONY: build
build:
	devtool build --arch=amd64

.PHONY: tests
tests:
	devtool test

.PHONY: pre-commite
pre-commite: setup lint build tests

.PHONY: ci
ci: install setup lint build tests

deb:
	deb-builder build

local: build
	cp ./build/deb-builder_amd64 $(GOPATH)/bin/deb-builder