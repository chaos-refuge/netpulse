APP=netpulse
SRC=$(wildcard *.go) frontend/index.html

.PHONY: build build-all run clean release

build: $(APP)

$(APP): $(SRC)
	go mod tidy
	go build -ldflags="-s -w" -o $(APP) .

build-all: build-darwin-arm64 build-darwin-amd64 build-windows-amd64 build-linux-amd64

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(APP)-darwin-arm64 .

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(APP)-darwin-amd64 .

build-windows-amd64:
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(APP)-windows-amd64.exe .

build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(APP)-linux-amd64 .

run: $(APP)
	./$(APP)

clean:
	rm -f $(APP) $(APP)-darwin-* $(APP)-windows-* $(APP)-linux-*

# Build all platforms for release
release: clean build-all
	@echo "✅ Release binaries built"
	@ls -lh $(APP)-*
