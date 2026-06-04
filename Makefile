APP=netpulse

.PHONY: build build-all run clean release vet lint test test-race test-cover coverage

# --- Quality ---
vet:
	go vet ./...

lint:
	gosec -quiet ./... 2>/dev/null || echo "⚠️  gosec not installed; run: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"

test:
	go test -count=1 -short ./...

test-race:
	go test -race -count=1 -short ./...

test-cover:
	go test -cover -count=1 -short ./...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

# --- Build ---
build: vet
	go mod tidy
	go build -ldflags="-s -w" -o $(APP) ./cmd/netpulse

build-all: build-darwin-arm64 build-darwin-amd64 build-windows-amd64 build-linux-amd64

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(APP)-darwin-arm64 ./cmd/netpulse

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(APP)-darwin-amd64 ./cmd/netpulse

build-windows-amd64:
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(APP)-windows-amd64.exe ./cmd/netpulse

build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(APP)-linux-amd64 ./cmd/netpulse

run: build
	./$(APP)

clean:
	rm -f $(APP) $(APP)-darwin-* $(APP)-windows-* $(APP)-linux-*
	rm -f coverage.out coverage.html

release: clean test vet lint
	$(MAKE) build-all
	@echo "✅ Release binaries built"
	@ls -lh $(APP)-*
