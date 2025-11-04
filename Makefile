.PHONY: dev test lint sqlc migrate-up migrate-down

GOFILES := $(shell find . -name '*.go' -not -path './vendor/*')

mod:
go mod tidy

fmt:
gofmt -w $(GOFILES)

lint:
golangci-lint run ./...

sqlc:
sqlc generate -f configs/sqlc.yaml

migrate-up:
golang-migrate -path migrations -database "${DATABASE_URL}" up

migrate-down:
golang-migrate -path migrations -database "${DATABASE_URL}" down 1

test:
go test ./...

dev:
docker compose up -d db redis nats
air
