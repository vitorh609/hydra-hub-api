# api-hydra-hub

API REST em Go usando `chi` para roteamento HTTP e `pgx/v5` (`pgxpool`) para acesso ao Postgres (ex.: Supabase).

## Requisitos

- Go (o projeto está com `go 1.25.7` em `go.mod`)
- Banco Postgres acessível via `DATABASE_URL`

## Estrutura Do Projeto

- `cmd/api/main.go`
  - Entry-point da aplicação.
  - Carrega `.env` (via `godotenv`), cria conexão com o banco (`internal/db`), monta o router (`internal/httpx`) e sobe o servidor HTTP.
- `internal/db/db.go`
  - Conexão com Postgres via `pgxpool`.
  - Ajusta `DefaultQueryExecMode` para `SimpleProtocol` (necessário em alguns cenários de pooler do Supabase).
- `internal/httpx/router.go`
  - Define rotas e faz o "wiring" (cria `Repo`, cria `Handler`, registra endpoints).
- `internal/users/`
  - Um recurso completo como exemplo (CRUD).
  - `internal/users/model.go`: structs de domínio e inputs (com tags JSON).
  - `internal/users/repo.go`: queries SQL e acesso ao banco.
  - `internal/users/handler.go`: handlers HTTP, validação básica, timeouts e serialização JSON.

## Configuração (ENV)

Variáveis esperadas:

- `DATABASE_URL` (obrigatória)
- `CORS_ALLOWED_ORIGINS` (opcional; lista separada por vírgula)
- `PORT` (opcional, default `3000`)
- `CLICKUP_CREDENTIALS_ENCRYPTION_KEY` (opcional para a API geral, mas obrigatória para habilitar a integração ClickUp; valor deve ser uma chave AES-256 em Base64)

Exemplo:

```env
DATABASE_URL=postgresql://user:pass@host:5432/dbname?sslmode=require
CORS_ALLOWED_ORIGINS=http://localhost:4200,http://127.0.0.1:4200
PORT=3000
CLICKUP_CREDENTIALS_ENCRYPTION_KEY=<base64-de-32-bytes>
```

Observação: atualmente existe um `.env` no repositório. Em projetos reais, evite commitar credenciais.

## Rodando Localmente

```bash
go run ./cmd/api
```

Se `DATABASE_URL` não estiver configurada, a aplicação falha na inicialização com erro explícito.

Endpoints:

- `GET /health` -> `{"status":"ok"}`
- `POST /auth/login`
- `POST /users`
- `GET /users`
- `GET /users/{id}`
- `PUT /users/{id}`
- `DELETE /users/{id}`
- `POST /notes`
- `GET /notes`
- `GET /notes/{id}`
- `PUT /notes/{id}`
- `DELETE /notes/{id}`
- `POST /tickets`
- `GET /tickets`
- `GET /tickets/{id}`
- `PUT /tickets/{id}`
- `DELETE /tickets/{id}`
- `POST /integrations/clickup/connect`
- `GET /integrations/clickup/status`
- `GET /integrations/clickup/spaces`
- `GET /integrations/clickup/spaces/{spaceId}/folders`
- `GET /integrations/clickup/lists/{listId}/tasks`
- `POST /integrations/clickup/lists/{listId}/tasks`

## Autenticação

- Todos os endpoints de negócio (`/users`, `/notes`, `/tickets`, `/account-settings`) exigem login.
- Os endpoints de integração (`/integrations/clickup/*`) também exigem login e são escopados ao usuário autenticado na arquitetura atual.
- O login é feito em `POST /auth/login` com `login` e `password`.
- O token deve ser enviado em `Authorization: Bearer <token>`.
- A sessão expira após `15 minutos` sem requisições autenticadas.
- Cada requisição autenticada renova a sessão por mais `15 minutos`.

Exemplo de login:

```bash
curl -X POST http://localhost:3000/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"login":"user@email.com","password":"123456"}'
```

Depois use o token retornado:

```bash
curl http://localhost:3000/users \
  -H 'Authorization: Bearer SEU_TOKEN'
```

## Integração ClickUp

A integração do ClickUp é totalmente server-side: o front-end envia requests apenas para a Hydra, e a Hydra é quem guarda o token criptografado, autentica no ClickUp, aplica timeout/retry e normaliza as respostas.

Premissa adotada nesta versão:

- Como a API atual ainda não possui um modelo explícito de tenant/workspace Hydra, a conexão do ClickUp foi isolada por usuário autenticado (`users.id`).
- O token do ClickUp nunca é retornado pela API nem escrito em logs.
- A tabela `clickup_connections` é criada automaticamente no bootstrap da API.

### Configuração

Defina `CLICKUP_CREDENTIALS_ENCRYPTION_KEY` com uma chave AES-256 em Base64. Exemplo para gerar localmente:

```bash
openssl rand -base64 32
```

Sem essa variável, a API sobe normalmente, mas os endpoints do ClickUp respondem `503`.

### Endpoints

`POST /integrations/clickup/connect`

Request:

```json
{
  "token": "pk_xxxxxxxxx",
  "defaultWorkspaceId": "90123456",
  "defaultWorkspaceName": "Hydra"
}
```

Response:

```json
{
  "connected": true,
  "status": "connected",
  "defaultWorkspaceId": "90123456",
  "defaultWorkspaceName": "Hydra",
  "lastCheckedAt": "2026-04-05T12:00:00Z"
}
```

`GET /integrations/clickup/status`

Response:

```json
{
  "connected": true,
  "status": "connected",
  "defaultWorkspaceId": "90123456",
  "defaultWorkspaceName": "Hydra",
  "lastCheckedAt": "2026-04-05T12:03:00Z"
}
```

`GET /integrations/clickup/spaces`

Response:

```json
[
  {
    "id": "445566",
    "name": "Engineering",
    "private": false,
    "workspace": {
      "id": "90123456",
      "name": "Hydra"
    }
  }
]
```

`GET /integrations/clickup/spaces/{spaceId}/folders`

Response:

```json
{
  "spaceId": "445566",
  "folders": [
    {
      "id": "100",
      "name": "Backlog",
      "hidden": false,
      "spaceId": "445566"
    }
  ],
  "lists": [
    {
      "id": "200",
      "name": "Sprint",
      "spaceId": "445566",
      "folderId": "100"
    }
  ]
}
```

`GET /integrations/clickup/lists/{listId}/tasks?page=0`

Response:

```json
{
  "listId": "200",
  "pagination": {
    "page": 0,
    "hasMore": true
  },
  "tasks": [
    {
      "id": "task_1",
      "name": "Implementar integração",
      "status": "open",
      "priority": "high",
      "listId": "200",
      "dateCreated": "2026-04-05T12:00:00Z",
      "dateUpdated": "2026-04-05T12:10:00Z",
      "assigneeIds": ["123"]
    }
  ]
}
```

`POST /integrations/clickup/lists/{listId}/tasks`

Request:

```json
{
  "name": "Nova task via Hydra",
  "description": "Criada pela API interna",
  "priority": 2,
  "assigneeIds": ["123"]
}
```

Response:

```json
{
  "id": "task_2",
  "name": "Nova task via Hydra",
  "description": "Criada pela API interna",
  "status": "to do",
  "listId": "200",
  "dateCreated": "2026-04-05T12:20:00Z",
  "dateUpdated": "2026-04-05T12:20:00Z",
  "assigneeIds": ["123"]
}
```

Documentação mais objetiva do fluxo: `internal/auth/README.md`.

## Deploy no Render

- Build Command: `go build -tags netgo -ldflags '-s -w' -o app ./cmd/api`
- Start Command: `./app`
- Health Check Path: `/health`

Variáveis de ambiente necessárias:

- `DATABASE_URL`: obrigatória. A API falha na inicialização se não estiver configurada.
- `CORS_ALLOWED_ORIGINS`: recomendada em produção. Informe os domínios permitidos separados por vírgula.
- `PORT`: o Render fornece essa variável automaticamente. Localmente, o default continua sendo `3000`.

Exemplo de `CORS_ALLOWED_ORIGINS` em produção:

```env
CORS_ALLOWED_ORIGINS=https://app.exemplo.com,https://admin.exemplo.com
```

## Como Adicionar Um Novo Recurso (Novo CRUD Para Uma Nova Tabela)

A forma mais rápida e consistente é copiar o padrão do recurso `users`.

### 1) Crie/Ajuste A Tabela No Banco

O `Repo` usa SQL direto, então a tabela precisa existir antes.

Exemplo (tabela `posts`, apenas como referência):

```sql
create table if not exists posts (
  id uuid primary key default gen_random_uuid(),
  title text not null,
  body text not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
```

Dica: o recurso `users` assume que o banco retorna `id`, `created_at` e `updated_at` no `returning ...`.

### 2) Crie O Pacote Do Recurso

Crie uma pasta `internal/<recurso>` com os mesmos 3 arquivos:

- `internal/posts/model.go`
- `internal/posts/repo.go`
- `internal/posts/handler.go`

#### `model.go` (domínio + inputs)

Use o padrão atual:

- struct principal com tags JSON
- input de create com campos obrigatórios (tipos não ponteiro)
- input de update com campos opcionais (ponteiros), para permitir update parcial via `coalesce(...)`

Exemplo:

```go
package posts

import "time"

type Post struct {
  ID        string    `json:"id"`
  Title     string    `json:"title"`
  Body      string    `json:"body"`
  CreatedAt time.Time `json:"created_at"`
  UpdatedAt time.Time `json:"updated_at"`
}

type CreatePostInput struct {
  Title string `json:"title"`
  Body  string `json:"body"`
}

type UpdatePostInput struct {
  Title *string `json:"title"`
  Body  *string `json:"body"`
}
```

#### `repo.go` (SQL)

Siga o padrão de `internal/users/repo.go`:

- `type Repo struct { pool *pgxpool.Pool }`
- `NewRepo(pool)`
- `Create/List/GetByID/Update/Delete`
- em `GetByID` e `Update`, traduza `pgx.ErrNoRows` para `ErrNotFound`
- em `Delete`, valide `RowsAffected()`

Pontos importantes:

- Em `Update`, use `coalesce($2, title)` etc para update parcial quando o input é ponteiro.
- Atualize `updated_at` no update (o `users` usa `time.Now().UTC()`).

#### `handler.go` (HTTP)

Siga o padrão de `internal/users/handler.go`:

- decode JSON com `json.NewDecoder(r.Body).Decode(&in)`
- validação simples (campos obrigatórios no create)
- `context.WithTimeout(..., 5*time.Second)` antes de chamar o repo
- mapeie `ErrNotFound` para `404`
- para respostas JSON, use um helper tipo `writeJSON`

Observação: hoje o helper `writeJSON` está definido dentro de `internal/users/handler.go`. Para um novo recurso você pode:

- duplicar o helper no novo `handler.go`, ou
- extrair para um helper compartilhado (ex.: `internal/httpx/response.go`) e reutilizar.

### 3) Registre As Rotas No Router

Edite `internal/httpx/router.go` para:

1. Importar o pacote do recurso (ex.: `api-hydra-hub/internal/posts`)
2. Instanciar `Repo` e `Handler`
3. Criar o bloco `r.Route("/posts", ...)`

Exemplo:

```go
postRepo := posts.NewRepo(pool)
postHandler := posts.NewHandler(postRepo)

r.Route("/posts", func(r chi.Router) {
  r.Post("/", postHandler.Create)
  r.Get("/", postHandler.List)
  r.Get("/{id}", postHandler.GetByID)
  r.Put("/{id}", postHandler.Update)
  r.Delete("/{id}", postHandler.Delete)
})
```

### 4) Teste Rapidamente Com curl

```bash
curl -sS localhost:3000/health
curl -sS -X POST localhost:3000/posts -H 'content-type: application/json' -d '{"title":"t","body":"b"}'
curl -sS localhost:3000/posts
```
