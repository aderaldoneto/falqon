.PHONY: up down logs ps test migrate-up migrate-down migrate-status migrate-create

up:
	docker compose up --build

down:
	docker compose down

logs:
	docker compose logs -f

ps:
	docker compose ps

test:
	docker compose exec api go test ./...
	docker compose exec web npm test

migrate-up:
	docker compose exec api sh -c 'goose -dir ./migrations postgres "$$DATABASE_URL" up'

migrate-down:
	docker compose exec api sh -c 'goose -dir ./migrations postgres "$$DATABASE_URL" down'

migrate-status:
	docker compose exec api sh -c 'goose -dir ./migrations postgres "$$DATABASE_URL" status'

migrate-create:
	@test -n "$(name)" || (echo "Use: make migrate-create name=migration_name" && exit 1)
	docker compose exec api goose -dir ./migrations create "$(name)" sql
