-include .env
export

.PHONY: run build tidy test migrate

run:
	go run ./cmd/server/main.go

build:
	go build -o bin/gym-crm ./cmd/server/main.go

tidy:
	go mod tidy

test:
	go test ./... -v -count=1

migrate:
	go run ./cmd/migrate/main.go
