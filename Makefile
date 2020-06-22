BIN := bin/
GOARCH := amd64
GOOS := linux

bin:
	GO111MODULE=on CGO_ENABLED=0 GOARCH=$(GOARCH) GOOS=$(GOOS) go build -mod=vendor -o $(BIN) ./

.PHONY: bin
