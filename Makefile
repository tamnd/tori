BIN     := tori
PKG     := ./cmd/tori
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/tamnd/tori/cli.Version=$(VERSION) \
	-X github.com/tamnd/tori/cli.Commit=$(COMMIT) \
	-X github.com/tamnd/tori/cli.Date=$(DATE)

export CGO_ENABLED := 0

.PHONY: build install test vet tidy clean run

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BIN) $(PKG)

install:
	go install -ldflags "$(LDFLAGS)" $(PKG)

test:
	go test -race ./...

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf bin

run: build
	./bin/$(BIN)
