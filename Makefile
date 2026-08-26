.PHONY: up down logs ps test

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

