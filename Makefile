VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.Version=$(VERSION)"

build:
	go build $(LDFLAGS) -o gstash ./cmd/gstash

release:
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o dist/gstash-darwin-arm64 ./cmd/gstash
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o dist/gstash-darwin-amd64 ./cmd/gstash
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o dist/gstash-linux-amd64  ./cmd/gstash
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/gstash-windows-amd64.exe ./cmd/gstash

clean:
	rm -f gstash dist/*