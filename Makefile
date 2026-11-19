# Build stamper. REPO enables `stamper self-update` from GitHub releases
# (assets named stamper_<os>_<arch>), e.g. make build REPO=you/stamper
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 0.1.0-dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null)
REPO    ?= pipejesus/stamper
GO      ?= go
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X github.com/pipejesus/stamper/internal/selfupdate.Repo=$(REPO)

.PHONY: build install test lint release clean
build:
	$(GO) build -trimpath -ldflags '$(LDFLAGS)' -o bin/stamper ./cmd/stamper

install:
	$(GO) install -trimpath -ldflags '$(LDFLAGS)' ./cmd/stamper

test:
	$(GO) test ./...

lint:
	$(GO) vet ./... && test -z "$$(gofmt -l .)"

# Cross-compiled raw binaries in the layout self-update expects.
release:
	@mkdir -p dist
	@for p in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do \
	  os=$${p%/*}; arch=$${p#*/}; \
	  GOOS=$$os GOARCH=$$arch $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o dist/stamper_$${os}_$${arch} ./cmd/stamper && echo dist/stamper_$${os}_$${arch}; \
	done

clean:
	rm -rf bin dist
