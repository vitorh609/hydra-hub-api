# Configuracao Angular + API Hydra Hub

Este guia cobre apenas a configuracao para consumir a API Go a partir de um app Angular em ambiente local.

## 1) Backend (API)

No backend, o CORS agora esta habilitado via middleware global.

Variavel de ambiente suportada:

- `CORS_ALLOWED_ORIGINS`
  - Lista separada por virgula.
  - Exemplo: `http://localhost:4200,http://127.0.0.1:4200`
  - Default (quando vazio): `http://localhost:4200,http://127.0.0.1:4200`

Exemplo no `.env`:

```env
DATABASE_URL=postgresql://user:pass@host:5432/dbname?sslmode=require
PORT=3000
CORS_ALLOWED_ORIGINS=http://localhost:4200
```

Suba a API:

```bash
go run ./cmd/api
```

## 2) Angular (cliente)

Use `HttpClient` apontando para a API:

```ts
// example.service.ts
import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';

@Injectable({ providedIn: 'root' })
export class ExampleService {
  private apiUrl = 'http://localhost:3000';

  constructor(private http: HttpClient) {}

  listUsers() {
    return this.http.get(`${this.apiUrl}/users`);
  }
}
```

## 3) Teste rapido

Com API em `:3000` e Angular em `:4200`:

- Abra o app Angular no navegador.
- Dispare uma chamada para `GET /users`.
- Verifique que a requisicao nao e bloqueada por CORS.

## 4) Problemas comuns

- Erro CORS no browser:
  - Confirme se a origem do Angular esta incluida em `CORS_ALLOWED_ORIGINS`.
- Preflight `OPTIONS`:
  - A API responde `204 No Content` automaticamente.
- Porta diferente no Angular:
  - Atualize `CORS_ALLOWED_ORIGINS` com a origem correta.
