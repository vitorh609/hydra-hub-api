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
- `PORT` (opcional, default `3000`)

Exemplo:

```env
DATABASE_URL=postgresql://user:pass@host:5432/dbname?sslmode=require
PORT=3000
```

Observação: atualmente existe um `.env` no repositório. Em projetos reais, evite commitar credenciais.

## Rodando Localmente

```bash
go run ./cmd/api
```

Endpoints:

- `GET /health` -> `{"status":"ok"}`
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
