# Login e Sessão

## O que foi implementado

- `POST /auth/login`
- Autenticação por `Bearer token`
- Sessão com expiração por inatividade de 15 minutos
- Proteção dos demais endpoints via middleware

## Payload de login

```json
{
  "login": "email-do-usuario",
  "password": "senha-do-usuario"
}
```

Hoje o campo `login` usa o e-mail salvo em `users.email`.

## Resposta

```json
{
  "token": "TOKEN_AQUI",
  "type": "Bearer",
  "expires_at": "2026-03-15T12:00:00Z",
  "user": {
    "id": "uuid",
    "name": "Nome",
    "email": "user@email.com"
  }
}
```

Use o token nas próximas requisições:

```http
Authorization: Bearer TOKEN_AQUI
```

## Regra de expiração

- Se o usuário ficar 15 minutos sem fazer nenhuma requisição autenticada, a sessão expira.
- Cada nova requisição autenticada renova a validade por mais 15 minutos.
- O backend devolve o novo vencimento no header `X-Session-Expires-At`.

## Tabela nova no Supabase/Postgres

Para essa estratégia funcionar de forma persistente, crie a tabela abaixo:

```sql
create extension if not exists pgcrypto;

create table if not exists auth_sessions (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  token_hash text not null unique,
  last_activity_at timestamptz not null default now(),
  expires_at timestamptz not null,
  revoked_at timestamptz,
  created_at timestamptz not null default now()
);

create index if not exists idx_auth_sessions_user_id on auth_sessions(user_id);
create index if not exists idx_auth_sessions_expires_at on auth_sessions(expires_at);
```

## Observação sobre senha

O login compara a senha enviada com `users.password_hash`.

- Se o valor salvo já estiver em SHA-256, ele funciona normalmente.
- Se existir dado legado em texto puro, o login ainda aceita para não quebrar o ambiente atual.
- Para novos registros, o ideal é salvar a senha já tratada pelo backend.
