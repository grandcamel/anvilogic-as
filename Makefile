VERSION ?= dev
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test lint smoke clean

build:
	go build -ldflags '$(LDFLAGS)' -o bin/anvilogic-as ./cmd/anvilogic-as

test:
	go test -race ./...

lint:
	golangci-lint run

smoke:
	ANVILOGIC_MOCK_MODE=1 go run ./cmd/anvilogic-as api GET /v1/ping

clean:
	rm -rf bin dist coverage.out
