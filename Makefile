.PHONY: up down logs test build run migrate-up migrate-down swag lint

up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f api

test:
	go test ./...

build:
	go build ./...

run:
	go run ./cmd/api

# Applies embedded migrations against DATABASE_URL (or the compose database).
migrate-up:
	DATABASE_URL=$${DATABASE_URL:-postgres://casemind:casemind@localhost:5432/casemind?sslmode=disable} go run ./cmd/migrate

# Migrations are forward-only by design; to reset locally, drop the volume.
migrate-down:
	docker compose down -v postgres 2>/dev/null || docker compose down -v

# The OpenAPI spec is maintained by hand at docs/openapi.yaml and served at
# /swagger; this target just validates it parses.
swag:
	@python3 -c "import yaml,sys; yaml.safe_load(open('docs/openapi.yaml')); print('openapi.yaml OK')" || \
	 echo "install python3+pyyaml or inspect docs/openapi.yaml manually"

lint:
	go vet ./...
