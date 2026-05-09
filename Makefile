.PHONY: build run test test-integration lint swagger generate docker-up docker-down

build: swagger
	go build -o bin/api ./cmd/api

run: swagger
	go run ./cmd/api

test:
	go test ./... -race -count=1

test-integration:
	go test ./... -race -count=1 -tags=integration

lint:
	golangci-lint run ./...

swagger:
	swag init -g cmd/api/main.go -o docs

generate:
	go generate ./...

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down -v
