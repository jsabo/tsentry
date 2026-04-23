BINARY := tsentry
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

.PHONY: build vet test clean install workspace snapshot

build:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BINARY) ./cmd/tsentry

vet:
	go vet ./...

test:
	go test ./...

clean:
	rm -f $(BINARY)

install:
	go install ./cmd/tsentry

# Create a local Go workspace so builds use ../teleport/api instead of the
# pinned public pseudo-version. go.work is gitignored.
workspace:
	go work init
	go work use . ../teleport/api

# Local multi-platform snapshot build (requires goreleaser)
snapshot:
	goreleaser release --snapshot --clean
