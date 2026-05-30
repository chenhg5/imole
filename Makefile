.PHONY: build test fmt vet clean release

BIN_DIR := bin
BIN := imole

build:
	go build -o $(BIN_DIR)/$(BIN) ./cmd/imole

release-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(BIN)-darwin-amd64 ./cmd/imole

release-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(BIN)-darwin-arm64 ./cmd/imole

release: release-darwin-amd64 release-darwin-arm64
	@echo "Build complete:"
	@ls -lh $(BIN_DIR)/

test:
	go test ./...

fmt:
	gofmt -w cmd internal

vet:
	go vet ./...

clean:
	rm -rf $(BIN_DIR)