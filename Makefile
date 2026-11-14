.PHONY: build test tidy frontend install-browsers compose-up

build:
	go build -o bin/backend ./cmd/backend
	go build -o bin/scrape-engine ./cmd/scrape-engine
	go build -o bin/huginn ./cmd/huginn

test:
	go test ./...

tidy:
	go mod tidy

frontend:
	cd apps/frontend && npm install && npm run build

install-browsers:
	go run ./cmd/huginn install-browsers

compose-up:
	docker compose up --build
