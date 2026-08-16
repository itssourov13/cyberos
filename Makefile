# CYBEROS Makefile — Phase 0 Foundation
# Do not add real tool execution or Phase 1 business targets here yet.

MODULE      := cyberos
BINARY      := bin/cyberos
CMD         := ./cmd/cyberos
GO          := go
GOFLAGS     :=
LDFLAGS     := -X cyberos/pkg/version.Version=0.0.0-foundation \
               -X cyberos/pkg/version.Commit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown) \
               -X cyberos/pkg/version.BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)

.PHONY: all build run version test test-unit clean fmt vet mod-tidy help

all: build

build:
	@mkdir -p bin
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

run: build
	./$(BINARY)

version: build
	./$(BINARY) version

test:
	$(GO) test ./...

test-unit:
	$(GO) test ./tests/unit/... ./pkg/... ./internal/... -count=1

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

mod-tidy:
	$(GO) mod tidy

clean:
	rm -rf bin/ dist/ coverage.txt coverage.html

help:
	@echo "CYBEROS Foundation Makefile"
	@echo ""
	@echo "Targets:"
	@echo "  build       Build the cyberos binary"
	@echo "  run         Build and run"
	@echo "  version     Print version information"
	@echo "  test        Run all tests"
	@echo "  test-unit   Run unit tests"
	@echo "  fmt         Format code"
	@echo "  vet         Run go vet"
	@echo "  mod-tidy    Tidy go.mod"
	@echo "  clean       Remove build artifacts"
