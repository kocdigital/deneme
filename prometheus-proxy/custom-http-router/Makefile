# Makefile for custom HTTP router project

.PHONY: all build run test clean

all: build

build:
	go build -o custom-http-router ./cmd/server

run: build
	./custom-http-router

test:
	go test ./...

clean:
	go clean
	rm -f custom-http-router