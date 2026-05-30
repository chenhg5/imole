.PHONY: build test fmt vet clean

BIN_DIR := bin
BIN := imole

build:
	go build -o $(BIN_DIR)/$(BIN) ./cmd/imole

test:
	go test ./...

fmt:
	gofmt -w cmd internal

vet:
	go vet ./...

clean:
	rm -rf $(BIN_DIR)
