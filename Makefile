VERSION ?= dev
LDFLAGS := -s -w -X main.version=$(VERSION)
BIN := cronk

.PHONY: build install check fmt lint clean

build:
	go build -ldflags="$(LDFLAGS)" -o $(BIN) ./cmd/cronk

install: build
	mkdir -p $(HOME)/.local/bin
	mv $(BIN) $(HOME)/.local/bin/$(BIN)

fmt:
	gofmt -w .

lint:
	golangci-lint run

check: fmt lint build

clean:
	rm -f $(BIN)
