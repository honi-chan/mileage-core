.PHONY: run build test migrate migrate-down docker-up docker-down k6-balance k6-contention

run:
	go run ./cmd/api/

build:
	go build -o bin/api ./cmd/api/

test:
	go test ./... -v

migrate:
	mysql -h 127.0.0.1 -P 3306 -u mileage -pmileage mileage < migrations/001_create_tables.up.sql

migrate-down:
	mysql -h 127.0.0.1 -P 3306 -u mileage -pmileage mileage < migrations/001_create_tables.down.sql

docker-up:
	docker compose up -d

docker-down:
	docker compose down

k6-balance:
	k6 run k6/scenarios/balance.js

k6-contention:
	k6 run k6/scenarios/redeem_contention.js

lint:
	golangci-lint run ./...
