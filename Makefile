VERSION=0.0.20
LDFLAGS=-ldflags "-w -s -X main.version=${VERSION}"
all: mackerel-plugin-linux-process-status

.PHONY: mackerel-plugin-linux-process-status

mackerel-plugin-linux-process-status: *.go
	go build $(LDFLAGS) -o mackerel-plugin-linux-process-status

linux: *.go
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o mackerel-plugin-linux-process-status

check:
	go test -v ./...

lint:
	golangci-lint run ./...