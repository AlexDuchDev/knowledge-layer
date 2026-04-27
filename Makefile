.PHONY: dev test lint typecheck db-up db-down

export DATABASE_URL ?= postgres://knowledge:knowledge@localhost:25432/knowledge?sslmode=disable

db-up:
	docker compose up -d postgres redis opensearch

db-down:
	docker compose down

dev: db-up
	@echo "Infra up (postgres, redis, opensearch). API: cd apps/api && go run ./cmd/api"
	@echo "Web: cd apps/web && npm install && npm run dev"
	@echo "Workers: cd apps/api && go run ./cmd/jobworker"
	@echo "         cd apps/api && go run ./cmd/connectorworker"

test:
	cd apps/api && go test ./... -count=1
	cd apps/api && go build -o /dev/null ./cmd/jobworker
	cd apps/api && go build -o /dev/null ./cmd/connectorworker

lint:
	cd apps/api && go vet ./...
	cd apps/web && npm run lint
	cd packages/shared && npm run lint

typecheck:
	cd apps/web && npm run typecheck
	cd packages/shared && npm run typecheck
