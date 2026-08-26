# Falqon

Aplicação full stack para criação, publicação e resposta de formulários dinâmicos de reviews de filmes.

## Funcionalidades

- Cadastro por e-mail e autenticação com Google OAuth 2.0.
- Área administrativa protegida por sessão.
- Criação e edição de formulários em rascunho.
- Geração e validação de slug único.
- Campos de texto curto, texto longo, número, escolha única, múltipla escolha e avaliação.
- Publicação e exclusão lógica de formulários.
- Catálogo público e acesso direto em `/forms/{slug}`.
- Validação e persistência transacional das respostas.
- Visualização das respostas na área administrativa.
- Contrato OpenAPI 3 com servidor Go e cliente TypeScript gerados.

## Tecnologias

- Backend: Go, Chi, pgx e oapi-codegen.
- Frontend: React, TypeScript, Vite, Material UI e TanStack Query.
- Banco de dados: PostgreSQL 17.
- Infraestrutura local: Docker Compose.
- Qualidade: Go test, Vitest, ESLint, Prettier e GitHub Actions.

## Arquitetura

```text
falqon/
├── backend/
│   ├── cmd/api/              # Inicialização da API
│   ├── internal/api/         # Handlers e código OpenAPI gerado
│   ├── internal/auth/        # Autenticação e sessões
│   ├── internal/forms/       # Domínio, validações e persistência
│   ├── migrations/           # Evolução do banco
│   ├── openapi/              # Contrato OpenAPI 3
│   └── seeds/                # Dados iniciais
├── frontend/src/
│   ├── api/                  # Cliente TypeScript gerado
│   └── pages/                # Páginas públicas e administrativas
├── scripts/                  # Automação de qualidade e geração
└── compose.yaml
```

O OpenAPI é a fonte de verdade entre backend e frontend. As regras de campos e respostas são validadas no backend. Formulários não publicados não são expostos pelas rotas públicas.

## Pré-requisitos

- Docker
- Docker Compose

Go, Node.js e PostgreSQL não precisam estar instalados no host.

## Configuração

```bash
cp .env.example .env
```

## Executando o projeto

```bash
docker compose up --build -d
docker compose exec api sh -c 'goose -dir ./migrations postgres "$DATABASE_URL" up'
```

Opcionalmente, crie o usuário administrador local:

```bash
docker compose exec db sh -c 'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -f /seeds/admin.sql'
```

Credenciais do seed:

```text
E-mail: user@falqon.com.br
Senha: 123456
```

Serviços locais:

- Frontend: http://localhost:5173
- API: http://localhost:8080
- Health check: http://localhost:8080/health
- PostgreSQL: `localhost:15432`

A API possui hot reload. Para acompanhar os logs ou encerrar o ambiente:

```bash
docker compose logs -f
docker compose down
```

## Google OAuth

Crie uma credencial OAuth 2.0 do tipo aplicação web no Google Cloud:

```text
Authorized JavaScript origin: http://localhost:5173
Authorized redirect URI:      http://localhost:8080/auth/google/callback
```

Preencha `GOOGLE_CLIENT_ID` e `GOOGLE_CLIENT_SECRET` no `.env` e recrie a API:

```bash
docker compose up -d --force-recreate api
```

Sem essas variáveis, o cadastro por e-mail continua disponível e a API retorna `google_auth_not_configured` nas rotas do Google.

## Fluxo de uso

1. Acesse http://localhost:5173 e cadastre-se.
2. Entre na área administrativa.
3. Crie um formulário e configure os campos.
4. Publique o formulário.
5. Compartilhe `http://localhost:5173/forms/{slug}`.
6. Consulte as respostas em **Meus formulários → Ver respostas**.

O catálogo público está em http://localhost:5173/forms.

## OpenAPI e geração de código

O contrato está em `backend/openapi/openapi.yaml`. Após alterá-lo:

```bash
./scripts/generate-api.sh
```

Ou:

```bash
docker compose exec api go generate ./internal/api
docker compose exec web npm run generate:api
```

Os arquivos gerados em `backend/internal/api/openapi.gen.go` e `frontend/src/api/generated` devem ser versionados e não editados manualmente.

## Testes e qualidade

```bash
./scripts/quality.sh
```

Comandos individuais:

```bash
docker compose exec api go test ./...
docker compose exec web npm test
docker compose exec web npm run lint
docker compose exec web npm run format:check
docker compose exec web npm run build
```

O pipeline `.github/workflows/quality.yml` executa testes, análise estática, formatação e build.

## Migrations

```bash
docker compose exec api sh -c 'goose -dir ./migrations postgres "$DATABASE_URL" status'
docker compose exec api sh -c 'goose -dir ./migrations postgres "$DATABASE_URL" down'
```

`docker compose down` preserva o volume `postgres_data`. Use `docker compose down -v` somente quando quiser excluir os dados.

## Decisões técnicas

- IDs numéricos auto incrementais e slug globalmente único.
- Estados `DRAFT`, `PUBLISHED` e `CANCELED` como enum no PostgreSQL.
- Auditoria com campos de criação, atualização e exclusão.
- Exclusão lógica para preservar histórico.
- Cookies de sessão `HttpOnly` e fluxo OAuth assinado.
- Submissão e respostas persistidas na mesma transação.
- Configurações específicas dos campos armazenadas em JSONB.
