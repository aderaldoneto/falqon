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

Terminar de documentar depois