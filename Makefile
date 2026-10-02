BINARY_NAME := guardd

.PHONY: build test vet lint release

build:
	mkdir -p bin
	go build -o bin/$(BINARY_NAME) ./cmd/guardd
	# Note: client/ is a library package (no main); only cmd/guardd produces a binary.

test:
	go test ./...

vet:
	go vet ./...

lint:
	@if [ -f .golangci.yml ] || [ -f .golangci.yaml ]; then golangci-lint run; else echo "no golangci-lint config; skipping"; fi

release:
	goreleaser release --snapshot --clean
