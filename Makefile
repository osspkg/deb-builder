
.PHONY: install
install:
	go install go.osspkg.com/goppy/v2/cmd/goppy@latest
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
	pkg-build build --base-dir=./build --tmp-dir=/tmp/deb-build --no-revision

local: build
	cp ./build/pkg-build_amd64 $(GOPATH)/bin/pkg-build