.PHONY: build install clean test

build:
	go build -o replicateme ./cmd/replicateme

install:
	go install ./cmd/replicateme

clean:
	rm -f replicateme

test:
	go test ./...
