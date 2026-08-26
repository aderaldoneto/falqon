#!/bin/sh
set -eu

docker compose exec api gofmt -w ./cmd ./internal
docker compose exec api go test ./...
docker compose exec web npm test
docker compose exec web npm run lint
docker compose exec web npm run format:check
docker compose exec web npm run build
