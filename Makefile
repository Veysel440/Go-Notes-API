APP ?= go-notes-api
REG ?= ghcr.io
REPO ?= $(shell git config --get remote.origin.url | sed -E 's#.*github.com[:/](.*)\.git#\1#')
TAG ?= latest

.PHONY: dev run test lint build tidy vuln docker-up docker-down docker-build docker-push sbom k6

dev:
	air

run:
	go run ./cmd/api

test:
	go test ./... -count=1

lint:
	golangci-lint run

tidy:
	go mod tidy

vuln:
	govulncheck ./...

build:
	CGO_ENABLED=0 go build -o bin/$(APP) ./cmd/api

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down -v

docker-build:
	docker build -t $(APP):$(TAG) .

docker-push:
	docker buildx build --platform linux/amd64,linux/arm64 -t $(REG)/$(REPO):$(TAG) --push .

sbom:
	syft packages docker:$(APP):$(TAG) -o spdx-json > sbom.spdx.json

k6:
	docker run --rm -e BASE_URL=http://host.docker.internal:8080 -i -v $$PWD/tests/load:/scripts grafana/k6:0.51.0 run /scripts/smoke.js
openapi:
	npx -y @stoplight/spectral-cli lint openapi/openapi.yaml --fail-severity=error