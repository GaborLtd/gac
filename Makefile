VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS = -s -w -X main.version=$(VERSION)

.PHONY: check test vet fmt-check docs-check build clean

check: fmt-check test vet docs-check
	sh -n install.sh
	sh scripts/test-install.sh

test:
	go test ./...

vet:
	go vet ./...

fmt-check:
	test -z "$$(gofmt -l cmd internal)"

docs-check:
	sh scripts/check-docs.sh

build:
	mkdir -p dist
	go build -ldflags "$(LDFLAGS)" -o dist/gac ./cmd/gac

clean:
	rm -rf dist
