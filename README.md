# Huginn

Self-hosted lead discovery application: configure BI sessions in the UI, extract businesses from Google Maps, visualize them on a map, export CSV, and erase sessions.

```
SEARCH → EXTRACT → NORMALIZE → DEDUPLICATE → STORE → MAP / CSV
```

## Quick start

```bash
cp .env.example .env
docker compose up --build
```

Open **http://localhost:3000** and sign in with `ADMIN_USER` / `ADMIN_PASSWORD` from `.env`.

Services started by Compose:

| Service | Role |
|---------|------|
| `frontend` | React + Leaflet UI (port 3000) |
| `backend` | Auth, sessions, Postgres persistence, gateway |
| `scrape-engine` | Playwright extraction only (internal) |
| `postgres` | Sessions + leads |

## What Huginn does

- Admin login (single user from env + JWT)
- Create BI **sessions** (query, location, limits)
- Run extractions via the scrape engine
- Persist leads in PostgreSQL
- Map visualization (Leaflet / OpenStreetMap)
- CSV export per session
- Erase sessions (cascade deletes leads)

## What it deliberately does not do

- Multi-tenant SaaS / billing
- CAPTCHA bypass or anti-bot evasion
- Redis / Kafka / Kubernetes
- Public scrape-engine exposure

Respect Google terms, privacy, and data-use rules. Prefer official Maps Platform APIs when they fit.

## Architecture

```
Browser → frontend (nginx) → backend → postgres
                           ↘ scrape-engine → Playwright / Google Maps
```

Go module layout:

- `cmd/backend` — public API
- `cmd/scrape-engine` — internal extract API
- `cmd/huginn` — optional local CLI
- `apps/frontend` — Vite React SPA
- `internal/upstream` — inherited Maps extraction (MIT)
- `migrations/` — Postgres schema

## Configuration

See [`.env.example`](.env.example).

| Variable | Purpose |
|----------|---------|
| `ADMIN_USER` / `ADMIN_PASSWORD` | UI login |
| `JWT_SECRET` | HS256 signing key (≥16 chars) |
| `DATABASE_URL` | Postgres DSN |
| `SCRAPE_ENGINE_URL` | Backend → engine |
| `HUGINN_CONCURRENCY` | Scrape workers |

## API (backend)

| Method | Path | Auth |
|--------|------|------|
| `POST` | `/api/v1/auth/login` | no |
| `GET` | `/api/v1/health` | no |
| `GET/PUT` | `/api/v1/settings` | JWT |
| `POST/GET` | `/api/v1/sessions` | JWT |
| `GET/DELETE` | `/api/v1/sessions/{id}` | JWT |
| `GET` | `/api/v1/sessions/{id}/leads` | JWT |
| `GET` | `/api/v1/sessions/{id}/export.csv` | JWT |

## Local development

```bash
# Postgres + engine + backend via compose, or run pieces locally:
go run ./cmd/scrape-engine
DATABASE_URL=... SCRAPE_ENGINE_URL=http://localhost:8090 go run ./cmd/backend

cd apps/frontend && npm install && npm run dev
```

Optional CLI:

```bash
go run ./cmd/huginn install-browsers
go run ./cmd/huginn search "barbershops" --location "São José dos Campos, SP"
```

## Testing

```bash
go test ./...
```

Live browser extraction (opt-in):

```bash
HUGINN_LIVE=1 go test -tags=live ./internal/scraper -run Live -count=1
```

## License & attribution

- Huginn: [LICENSE](LICENSE)
- Upstream extraction: [NOTICE.md](NOTICE.md) — [gosom/google-maps-scraper](https://github.com/gosom/google-maps-scraper)
