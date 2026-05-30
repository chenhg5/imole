.PHONY: build test fmt vet clean release release-all

BIN_DIR := bin
BIN := imole

build:
	go build -o $(BIN_DIR)/$(BIN) ./cmd/imole

release-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(BIN)-darwin-amd64 ./cmd/imole

release-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(BIN)-darwin-arm64 ./cmd/imole

release-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(BIN)-linux-amd64 ./cmd/imole

release-linux-arm64:
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(BIN)-linux-arm64 ./cmd/imole

release-windows-amd64:
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(BIN)-windows-amd64.exe ./cmd/imole

release: release-darwin-amd64 release-darwin-arm64 release-linux-amd64 release-linux-arm64 release-windows-amd64
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