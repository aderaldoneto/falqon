# Falqon

Aplicacao full stack para criar, publicar e responder formulários dinamicos.

## Requisitos

- Docker
- Docker Compose

Não é necessário instalar Go, Node.js ou PostgreSQL no host.

## Primeira execução

```bash
cp .env.example .env
docker compose up --build
```

Serviços locais:

- Frontend: http://localhost:5173
- API: http://localhost:8080
- Health check: http://localhost:8080/health
- PostgreSQL: localhost:5432

Para encerrar os containers:

```bash
docker compose down
```

## Banco de dados e migrations

Migrations:

```bash
docker compose exec api sh -c 'goose -dir ./migrations postgres "$DATABASE_URL" up'
```

Consulte o status ou reverta a migration mais recente com:

```bash
docker compose exec api sh -c 'goose -dir ./migrations postgres "$DATABASE_URL" status'
docker compose exec api sh -c 'goose -dir ./migrations postgres "$DATABASE_URL" down'
```

As migrations ficam em `backend/migrations`.

## Dados iniciais

Depois de aplicar as migrations, crie ou atualize o usuario administrador de
desenvolvimento:

```bash
docker compose exec db sh -c 'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -f /seeds/admin.sql'
```

Credenciais locais:

```text
E-mail: user@falqon.com.br
Senha: 123456
```

## OpenAPI e cliente TypeScript

O contrato da API fica em `backend/openapi/openapi.yaml`. Depois de alterar o
contrato, regenere o codigo Go e o cliente TypeScript:

```bash
docker compose exec api go generate ./internal/api
docker compose exec web npm run generate:api
```

Os arquivos gerados ficam em `backend/internal/api/openapi.gen.go` e
`frontend/src/api/generated`. Eles devem ser versionados e nao devem ser
editados manualmente.
