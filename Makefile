APP=go-notes-api

.PHONY: dev run test lint build docker-up docker-down k6

dev:
	air

run:
	go run ./cmd/api

test:
	go test ./...

lint:
	golangci-lint run

build:
	CGO_ENABLED=0 go build -o bin/$(APP) ./cmd/api

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down -v

k6:
	docker run --rm -e BASE_URL=http://host.docker.internal:8080 -i -v $$PWD/tests/load:/scripts grafana/k6:0.51.0 run /scripts/smoke.js
