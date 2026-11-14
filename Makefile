.PHONY: build test tidy install-browsers run-serve

build:
	go build -o bin/huginn ./cmd/huginn

test:
	go test ./...

tidy:
	go mod tidy

install-browsers:
	go run ./cmd/huginn install-browsers

run-serve: build
	./bin/huginn serve --addr :8080
