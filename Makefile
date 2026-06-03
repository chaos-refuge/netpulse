.PHONY: build run clean

APP=netpulse
SRC=$(wildcard *.go) frontend/index.html

build: $(APP)

$(APP): $(SRC)
	go mod tidy
	go build -ldflags="-s -w" -o $(APP) .

run: $(APP)
	./$(APP)

clean:
	rm -f $(APP)

# cross-compile for release
release:
	GOOS=darwin  GOARCH=arm64 go build -ldflags="-s -w" -o $(APP)-darwin-arm64 .
	GOOS=darwin  GOARCH=amd64 go build -ldflags="-s -w" -o $(APP)-darwin-amd64 .
	GOOS=linux   GOARCH=amd64 go build -ldflags="-s -w" -o $(APP)-linux-amd64 .
	@echo "✅ Release binaries built"
