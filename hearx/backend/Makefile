.PHONY: build run test clean

build:
	go build -o bin/hearx ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./... -v

clean:
	rm -rf bin/ tmp/
