VERSION=0.0.11
GITCOMMIT?=$(shell git describe --dirty --always 2>/dev/null)
LDFLAGS=-ldflags "-w -s -X main.version=${VERSION} -X main.commit=${GITCOMMIT}"
all: mackerel-plugin-linux-process-status

.PHONY: mackerel-plugin-linux-process-status

mackerel-plugin-linux-process-status: linux.go
	go build $(LDFLAGS) -o mackerel-plugin-linux-process-status

linux: linux.go
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o mackerel-plugin-linux-process-status

check:
	go test ./...

clean:
	rm -rf mackerel-plugin-linux-process-status

