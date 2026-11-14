# Huginn

<p align="center">
  <img src="apps/frontend/public/favicon.ico" alt="Huginn logo" width="96" height="96" />
</p>

<p align="center">
  <strong>Self-hosted lead discovery for local business intelligence</strong><br />
  Configure BI sessions, extract businesses from Google Maps, map the results, export CSV, and erase sessions when done.
</p>

<p align="center">
  <a href="#-english">🇺🇸 English</a> ·
  <a href="#-português">🇧🇷 Português</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/React-20232A?style=for-the-badge&logo=react&logoColor=61DAFB" alt="React" />
  <img src="https://img.shields.io/badge/Vite-646CFF?style=for-the-badge&logo=vite&logoColor=white" alt="Vite" />
  <img src="https://img.shields.io/badge/Leaflet-199900?style=for-the-badge&logo=leaflet&logoColor=white" alt="Leaflet" />
  <img src="https://img.shields.io/badge/PostgreSQL-316192?style=for-the-badge&logo=postgresql&logoColor=white" alt="PostgreSQL" />
  <img src="https://img.shields.io/badge/Docker-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="Docker" />
  <img src="https://img.shields.io/badge/Playwright-2EAD33?style=for-the-badge&logo=playwright&logoColor=white" alt="Playwright" />
  <img src="https://img.shields.io/badge/License-MIT-yellow?style=for-the-badge" alt="MIT" />
</p>

---

# 🇺🇸 English

## Table of contents

- [Overview](#overview)
- [Features and usability](#features-and-usability)
- [Technical stack](#technical-stack)
- [Architecture](#architecture)
- [Implementation notes](#implementation-notes)
- [Project structure](#project-structure)
- [Getting started](#getting-started)
- [Usage](#usage)
- [API reference](#api-reference)
- [Development](#development)
- [Testing](#testing)
- [Security](#security)
- [Contributing](#contributing)
- [License and acknowledgements](#license-and-acknowledgements)

## Overview

Huginn is a compose-first, self-hosted application for **business lead discovery**. Named after the raven that gathers information and returns it home, the product focuses on a single workflow:

**SEARCH → EXTRACT → NORMALIZE → DEDUPLICATE → STORE → MAP / CSV**

It is intended for operators who want a local control plane: sign in, configure an extraction session, run the scrape engine, inspect leads on a map, export CSV, and delete the BI session when finished.

**In scope:** single-admin UI, sessions, Postgres persistence, isolated scrape engine, Leaflet map.

**Out of scope:** multi-tenant SaaS, billing, CAPTCHA bypass / anti-bot evasion, Redis/Kafka/Kubernetes, publishing the scrape engine to the public internet.

Users must respect applicable Google terms, privacy requirements, and data-use restrictions. Prefer official Google Maps Platform APIs when they fit the use case. See [docs/extraction-feasibility.md](docs/extraction-feasibility.md).

## Features and usability

| Capability | Status |
|------------|--------|
| Branded landing / sign-in hero (logo + catchphrase) | Implemented |
| Admin login (`ADMIN_USER` / `ADMIN_PASSWORD` → JWT) | Implemented |
| Create BI sessions (query, location, limits, depth, lang) | Implemented |
| Async extraction via scrape-engine | Implemented |
| Persist leads in PostgreSQL | Implemented |
| Session list with status polling | Implemented |
| Leaflet / OpenStreetMap map + lead table | Implemented |
| CSV export per session | Implemented |
| Erase session (cascade deletes leads) | Implemented |
| Optional local CLI (`cmd/huginn`) | Implemented |
| Multi-user accounts / OAuth | Not implemented |

**Primary journey**

1. Open the UI at `http://localhost:3000` (branded landing hero) and sign in.
2. Create a **New session** with search parameters.
3. Wait while status moves `pending` → `running` → `completed` (or `failed`).
4. Review markers and the lead table; **Export CSV** or **Erase session**.

## Technical stack

| Layer | Technologies | Role |
|-------|--------------|------|
| Frontend | React, Vite, React Router, Leaflet / react-leaflet, nginx | SPA + map; reverse-proxy `/api` |
| Backend | Go, JWT (HS256), pgx | Auth, sessions/leads, gateway to engine |
| Scrape engine | Go, scrapemate, Playwright | Maps extraction only (no Postgres) |
| Database | PostgreSQL 16 | Sessions, leads, settings JSON |
| Packaging | Docker Compose | Four services: frontend, backend, scrape-engine, postgres |

## Architecture

### System context (Docker Compose)

```mermaid
flowchart TB
  User[Operator_Browser]

  subgraph publicEdge [Published]
    FE["frontend :3000<br/>React SPA + nginx"]
  end

  subgraph composeNet [Internal_Compose_Network]
    BE["backend :8080<br/>JWT + sessions API"]
    SE["scrape-engine :8090<br/>extract jobs only"]
    DB[(postgres :5432<br/>sessions + leads)]
  end

  ExtMaps[Google_Maps_via_Playwright]

  User -->|"HTTPS/HTTP UI + /api proxy"| FE
  FE -->|"/api/v1/*"| BE
  BE -->|"SQL"| DB
  BE -->|"POST /v1/extract + poll"| SE
  SE -->|"browser crawl"| ExtMaps
```

### Session extraction sequence

```mermaid
sequenceDiagram
  participant UI as Frontend
  participant BE as Backend
  participant DB as Postgres
  participant SE as ScrapeEngine

  UI->>BE: POST /api/v1/auth/login
  BE-->>UI: JWT
  UI->>BE: POST /api/v1/sessions + JWT
  BE->>DB: INSERT session pending
  BE-->>UI: 202 session
  BE->>SE: POST /v1/extract
  SE-->>BE: job_id
  loop Poll until terminal
    BE->>SE: GET /v1/jobs/id
    SE-->>BE: status
  end
  BE->>SE: GET /v1/jobs/id/results
  SE-->>BE: businesses JSON
  BE->>DB: INSERT leads + UPDATE session completed
  UI->>BE: GET session + leads
  BE->>DB: SELECT
  BE-->>UI: map and table data
```

### Data model

```mermaid
erDiagram
  sessions ||--o{ leads : contains
  sessions {
    uuid id PK
    text name
    text status
    jsonb config
    text error
    text engine_job_id
    timestamptz created_at
    timestamptz updated_at
  }
  leads {
    uuid id PK
    uuid session_id FK
    text place_id
    text name
    text category
    float latitude
    float longitude
    text phone
    text website
    jsonb raw
  }
```

**Boundaries:** only `frontend` is published (`:3000`). `backend` and `scrape-engine` stay on the Compose network. The scrape-engine never opens Postgres; the backend owns persistence and polls the engine until completion, then stores normalized leads.

## Implementation notes

Observable choices in this repository (not formal ADRs):

- **Single admin from env** — simplest self-hosted auth; JWT middleware on sessions/settings.
- **Engine isolation** — extraction binary has no login and no DB; only reachable on the Compose network.
- **Poll-based completion** — avoids Redis/webhooks for v1.
- **Upstream extraction** — Maps parsing lives under `internal/upstream` (MIT; see [NOTICE.md](NOTICE.md)).

## Project structure

```text
huginn/
├── apps/frontend/          # Vite React SPA + public/favicon.ico logo
├── cmd/backend/            # Public API
├── cmd/scrape-engine/      # Internal extract API
├── cmd/huginn/             # Optional CLI
├── internal/               # auth, store, pipeline, scraper, upstream, …
├── migrations/             # Postgres SQL
├── docs/                   # Feasibility and deeper notes
├── docker-compose.yml
├── Dockerfile.backend
├── Dockerfile.frontend
├── Dockerfile.scrape-engine
└── .env.example
```

Brand asset: [`apps/frontend/public/favicon.ico`](apps/frontend/public/favicon.ico) (used as favicon and product logo in the UI).

## Getting started

### Prerequisites

- Docker and Docker Compose
- (Optional local dev) Go 1.27+, Node.js 22+

### Compose (recommended)

```bash
cp .env.example .env
# edit ADMIN_PASSWORD and JWT_SECRET before any shared deployment
docker compose up --build
```

1. Open **http://localhost:3000**
2. Sign in with `ADMIN_USER` / `ADMIN_PASSWORD` from `.env`
3. Verify API through the UI proxy: `GET /api/v1/health` should return `{"status":"ok",...}`

Services: `frontend` (`:3000`), `backend` (internal `8080`), `scrape-engine` (internal `8090`), `postgres`.

### Configuration

Variable names only (see [`.env.example`](.env.example)):

| Variable | Purpose |
|----------|---------|
| `ADMIN_USER` / `ADMIN_PASSWORD` | UI login |
| `JWT_SECRET` | HS256 signing key (≥16 characters) |
| `POSTGRES_*` / `DATABASE_URL` | Database |
| `SCRAPE_ENGINE_URL` | Backend → engine |
| `HUGINN_CONCURRENCY`, `HUGINN_DEPTH`, `HUGINN_LANG`, … | Engine defaults |

## Usage

1. **Sessions** → **New session** → set query/location/limits → **Run extraction**
2. Open the session detail page; map and table update as leads are stored
3. **Export CSV** downloads `session-{id}.csv`
4. **Erase session** deletes the session and all leads (`ON DELETE CASCADE`)

Example API login (via frontend proxy or backend):

```bash
curl -s -X POST http://localhost:3000/api/v1/auth/login \
  -H 'content-type: application/json' \
  -d '{"username":"admin","password":"YOUR_PASSWORD"}'
```

```bash
TOKEN=... # from login response
curl -s http://localhost:3000/api/v1/sessions \
  -H "Authorization: Bearer $TOKEN"
```

## API reference

Base path (through frontend nginx): `/api/v1`. JWT required except login and health.

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| `POST` | `/api/v1/auth/login` | no | `{username,password}` → `{token}` |
| `GET` | `/api/v1/health` | no | Liveness |
| `GET` / `PUT` | `/api/v1/settings` | JWT | Defaults JSON in DB |
| `POST` | `/api/v1/sessions` | JWT | Create session + start extract |
| `GET` | `/api/v1/sessions` | JWT | List sessions |
| `GET` | `/api/v1/sessions/{id}` | JWT | Session detail |
| `GET` | `/api/v1/sessions/{id}/leads` | JWT | Leads for map/table |
| `GET` | `/api/v1/sessions/{id}/export.csv` | JWT | CSV download |
| `DELETE` | `/api/v1/sessions/{id}` | JWT | Erase session + leads |

Internal scrape-engine (Compose network only): `GET /health`, `POST /v1/extract`, `GET /v1/jobs/{id}`, `GET /v1/jobs/{id}/results`.

## Development

```bash
# Engine + backend (Postgres required)
go run ./cmd/scrape-engine
DATABASE_URL=postgres://huginn:huginn@localhost:5432/huginn?sslmode=disable \
  SCRAPE_ENGINE_URL=http://localhost:8090 \
  go run ./cmd/backend

# Frontend
cd apps/frontend && npm install && npm run dev
```

Optional CLI:

```bash
go run ./cmd/huginn install-browsers
go run ./cmd/huginn search "barbershops" --location "São José dos Campos, SP"
```

See [CONTRIBUTING.md](CONTRIBUTING.md).

## Testing

```bash
go test ./...
```

Opt-in live browser extraction:

```bash
HUGINN_LIVE=1 go test -tags=live ./internal/scraper -run Live -count=1
```

Frontend unit tests are not configured; validate UI with `npm run build` / Compose.

## Security

- Keep `ADMIN_PASSWORD` and `JWT_SECRET` out of git; use `.env` locally (gitignored).
- Scrape-engine port is not published to the host in Compose.
- JWT protects sessions and settings; login uses constant-time credential compare.
- Do not add CAPTCHA bypass or anti-bot evasion features.
- Extraction inherits upstream MIT code; comply with [NOTICE.md](NOTICE.md) and Google’s applicable terms.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Prefer small changes; keep product logic out of `internal/upstream`.

## License and acknowledgements

- Huginn: [LICENSE](LICENSE) (MIT)
- Upstream Maps extraction: [NOTICE.md](NOTICE.md) — [gosom/google-maps-scraper](https://github.com/gosom/google-maps-scraper)
- Feasibility analysis: [docs/extraction-feasibility.md](docs/extraction-feasibility.md)

---

# 🇧🇷 Português

## Sumário

- [Visão geral](#visão-geral)
- [Funcionalidades e usabilidade](#funcionalidades-e-usabilidade)
- [Stack tecnológica](#stack-tecnológica)
- [Arquitetura](#arquitetura)
- [Notas de implementação](#notas-de-implementação)
- [Estrutura do projeto](#estrutura-do-projeto)
- [Primeiros passos](#primeiros-passos)
- [Utilização](#utilização)
- [Referência da API](#referência-da-api)
- [Desenvolvimento](#desenvolvimento)
- [Testes](#testes)
- [Segurança](#segurança)
- [Contribuição](#contribuição)
- [Licença e agradecimentos](#licença-e-agradecimentos)

## Visão geral

Huginn é uma aplicação self-hosted, orientada a Docker Compose, para **descoberta de leads de negócios**. O nome evoca o corvo que coleta informação e a traz de volta; o produto concentra-se no fluxo:

**SEARCH → EXTRACT → NORMALIZE → DEDUPLICATE → STORE → MAP / CSV**

Serve operadores que querem um painel local: autenticar, configurar uma sessão de extração, executar o motor de scrape, ver leads no mapa, exportar CSV e apagar a sessão de BI.

**No escopo:** UI de admin único, sessões, Postgres, motor de scrape isolado, mapa Leaflet.

**Fora de escopo:** SaaS multi-inquilino, billing, bypass de CAPTCHA / evasão anti-bot, Redis/Kafka/Kubernetes, exposição pública do scrape-engine.

Respeite os termos aplicáveis do Google, privacidade e restrições de uso de dados. Prefira as APIs oficiais do Google Maps Platform quando forem adequadas. Ver [docs/extraction-feasibility.md](docs/extraction-feasibility.md).

## Funcionalidades e usabilidade

| Capacidade | Status |
|------------|--------|
| Landing / sign-in com hero (logo + catchphrase) | Implementado |
| Login admin (`ADMIN_USER` / `ADMIN_PASSWORD` → JWT) | Implementado |
| Criar sessões BI (query, localização, limites, depth, lang) | Implementado |
| Extração assíncrona via scrape-engine | Implementado |
| Persistência de leads no PostgreSQL | Implementado |
| Lista de sessões com polling de status | Implementado |
| Mapa Leaflet / OpenStreetMap + tabela | Implementado |
| Exportação CSV por sessão | Implementado |
| Apagar sessão (cascade nos leads) | Implementado |
| CLI local opcional (`cmd/huginn`) | Implementado |
| Multi-usuário / OAuth | Não implementado |

**Jornada principal**

1. Abra `http://localhost:3000` (landing com hero da marca) e faça login.
2. Crie uma **New session** com os parâmetros de busca.
3. Aguarde o status `pending` → `running` → `completed` (ou `failed`).
4. Revise o mapa e a tabela; **Export CSV** ou **Erase session**.

## Stack tecnológica

| Camada | Tecnologias | Papel |
|--------|-------------|-------|
| Frontend | React, Vite, React Router, Leaflet / react-leaflet, nginx | SPA + mapa; proxy de `/api` |
| Backend | Go, JWT (HS256), pgx | Auth, sessões/leads, gateway |
| Scrape engine | Go, scrapemate, Playwright | Só extração (sem Postgres) |
| Banco | PostgreSQL 16 | Sessões, leads, settings |
| Empacotamento | Docker Compose | frontend, backend, scrape-engine, postgres |

## Arquitetura

Os diagramas Mermaid da [seção Architecture](#architecture) (inglês) são a representação canônica da arquitetura atual:

1. **Contexto do sistema** — Compose com borda pública (`frontend :3000`) e rede interna (`backend`, `scrape-engine`, `postgres`).
2. **Sequência de extração** — login JWT → criar sessão → `POST /v1/extract` → polling → gravar leads.
3. **Modelo de dados** — `sessions` 1‑N `leads`.

**Limites:** só o frontend é publicado. O scrape-engine não acessa o Postgres; o backend persiste e faz polling até concluir.

## Notas de implementação

Escolhas observáveis neste repositório (não são ADRs formais):

- **Admin único via env** — auth simples para self-host; JWT nas rotas de sessão/settings.
- **Motor isolado** — sem login e sem DB; só na rede Compose.
- **Polling** — evita Redis/webhooks na v1.
- **Extração upstream** — parsing em `internal/upstream` (MIT; [NOTICE.md](NOTICE.md)).

## Estrutura do projeto

A árvore listada em [Project structure](#project-structure) corresponde ao repositório atual. O logo do produto está em [`apps/frontend/public/favicon.ico`](apps/frontend/public/favicon.ico).

## Primeiros passos

### Pré-requisitos

- Docker e Docker Compose
- (Dev local opcional) Go 1.27+, Node.js 22+

### Compose (recomendado)

```bash
cp .env.example .env
# altere ADMIN_PASSWORD e JWT_SECRET antes de compartilhar o ambiente
docker compose up --build
```

1. Abra **http://localhost:3000**
2. Entre com `ADMIN_USER` / `ADMIN_PASSWORD` do `.env`
3. Confirme `GET /api/v1/health` → `{"status":"ok",...}`

### Configuração

Nomes de variáveis (sem segredos) em [`.env.example`](.env.example) — mesma tabela da [seção em inglês](#configuration).

## Utilização

1. **Sessions** → **New session** → query/localização/limites → **Run extraction**
2. Abra o detalhe da sessão; mapa e tabela atualizam conforme os leads são gravados
3. **Export CSV** baixa `session-{id}.csv`
4. **Erase session** remove sessão e leads

Exemplos `curl` iguais aos da [seção Usage](#usage) (mantidos em inglês técnico).

## Referência da API

Mesmos endpoints documentados em [API reference](#api-reference). Autenticação JWT exceto login e health. Motor interno apenas na rede Compose.

## Desenvolvimento

```bash
go run ./cmd/scrape-engine
DATABASE_URL=postgres://huginn:huginn@localhost:5432/huginn?sslmode=disable \
  SCRAPE_ENGINE_URL=http://localhost:8090 \
  go run ./cmd/backend

cd apps/frontend && npm install && npm run dev
```

CLI opcional e detalhes em [CONTRIBUTING.md](CONTRIBUTING.md).

## Testes

```bash
go test ./...
```

Extração live (opt-in):

```bash
HUGINN_LIVE=1 go test -tags=live ./internal/scraper -run Live -count=1
```

## Segurança

- Não versionar `ADMIN_PASSWORD` / `JWT_SECRET`; use `.env` (ignorado pelo git).
- Porta do scrape-engine não é publicada no host no Compose.
- JWT protege sessões e settings.
- Não implementar bypass de CAPTCHA ou evasão anti-bot.
- Extração herda código MIT upstream — [NOTICE.md](NOTICE.md) e termos do Google.

## Contribuição

Ver [CONTRIBUTING.md](CONTRIBUTING.md). Prefira mudanças pequenas; mantenha lógica de produto fora de `internal/upstream`.

## Licença e agradecimentos

- Huginn: [LICENSE](LICENSE) (MIT)
- Extração upstream: [NOTICE.md](NOTICE.md) — [gosom/google-maps-scraper](https://github.com/gosom/google-maps-scraper)
- Análise de viabilidade: [docs/extraction-feasibility.md](docs/extraction-feasibility.md)
