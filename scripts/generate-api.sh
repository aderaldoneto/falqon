#!/bin/sh
set -eu

docker compose exec api go generate ./internal/api
docker compose exec web npm run generate:api
