.PHONY: run build test test-race test-integration govulncheck migrate-up migrate-down seed lint docker-build docker-compose-up docker-compose-down

run:
	go run ./cmd/server

build:
	CGO_ENABLED=0 go build -o bin/server ./cmd/server

test:
	go test ./... -v -race -count=1

test-race:
	go test ./... -race -count=1

test-integration:
	go test -tags=integration ./internal/apptest -count=1

govulncheck:
	govulncheck ./...

migrate-up:
	migrate -path migrations -database "postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable" down

seed:
	psql -U postgres -d qr_restaurant -f scripts/seed.sql

lint:
	golangci-lint run ./...

docker-build:
	docker build -t qr-restaurant:latest .

docker-compose-up:
	docker-compose up -d

docker-compose-down:
	docker-compose down
