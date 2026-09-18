BIN := bin/herdr-tmux-session-navigator
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: build test fmt vet clean link

build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BIN) ./cmd/navigator

test:
	go test ./...

fmt:
	gofmt -l -w .

vet:
	go vet ./...

clean:
	rm -rf bin

# Install into herdr from this working tree (does not run [[build]]).
link: build
	herdr plugin link .
