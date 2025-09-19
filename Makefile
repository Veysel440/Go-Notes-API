APP=go-notes-api
MIGRATE_IMG=migrate/migrate:4
MIGRATE_DSN?=mysql://$(DB_USERNAME):$(DB_PASSWORD)@tcp($(DB_HOST):$(DB_PORT))/$(DB_DATABASE)?multiStatements=true

.PHONY: dev run test lint vuln build docker-up docker-down k6 spectral sbom hadolint migrate-up migrate-down

dev: ; air
run: ; go run ./cmd/api

test:
	go test -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/$(APP) ./cmd/api

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down -v

k6:
	docker run --rm -e BASE_URL=http://host.docker.internal:8080 -i -v $$PWD/tests/load:/scripts grafana/k6:0.51.0 run /scripts/smoke.js

spectral:
	npx -y @stoplight/spectral-cli lint openapi/openapi.yaml --fail-severity=error

sbom:
	docker build -t $(APP):local .
	docker run --rm -v $$PWD:/out anchore/syft:latest $(APP):local -o spdx-json=/out/sbom.spdx.json

hadolint:
	docker run --rm -i hadolint/hadolint < Dockerfile

migrate-up:
	docker run --rm -v $$PWD/migrations:/migrations --network host $(MIGRATE_IMG) \
	  -path=/migrations -database '$(MIGRATE_DSN)' up

migrate-down:
	docker run --rm -v $$PWD/migrations:/migrations --network host $(MIGRATE_IMG) \
	  -path=/migrations -database '$(MIGRATE_DSN)' down 1
