# Flowmi CLI

BINARY := flowmi

# Build metadata
VERSION ?= $(shell V=$$(git describe --tags --abbrev=0 2>/dev/null || echo dev); echo $${V\#v})
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Go build flags
MODULE  := github.com/flowmi-ai/flowmi/cmd
LDFLAGS := -s -w \
	-X $(MODULE).version=$(VERSION) \
	-X $(MODULE).commit=$(COMMIT) \
	-X $(MODULE).date=$(DATE)

.PHONY: all build install test lint fmt vet clean dev

all: build

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARY) .

install:
	go install -trimpath -ldflags "$(LDFLAGS)" .

test:
	go test ./... -v -race -cover

lint:
	golangci-lint run

fmt:
	gofmt -s -l -w .

vet:
	go vet ./...

clean:
	rm -rf bin/

# Development build (no symbol stripping, for debugging)
dev:
	go build -ldflags "-X $(MODULE).version=$(VERSION) -X $(MODULE).commit=$(COMMIT) -X $(MODULE).date=$(DATE)" -o bin/$(BINARY) .
