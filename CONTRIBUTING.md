# Contributing to Huginn

## Scope

Huginn is a compose-first lead discovery app: **frontend → backend → scrape-engine → postgres**.

Avoid SaaS multi-tenancy, CAPTCHA bypasses, anti-bot evasion, or distributed worker infrastructure.

## Layout

- `apps/frontend` — Vite React + Leaflet
- `cmd/backend` — JWT gateway + sessions/leads
- `cmd/scrape-engine` — extraction only
- `internal/upstream` — inherited Maps engine (MIT boundary)

## Development

```bash
go test ./...
cd apps/frontend && npm install && npm run build
docker compose up --build
```

Live browser tests (opt-in):

```bash
HUGINN_LIVE=1 go test -tags=live ./internal/scraper -run Live -count=1
```

When syncing extraction code from [gosom/google-maps-scraper](https://github.com/gosom/google-maps-scraper), update `NOTICE.md` and keep product logic out of `internal/upstream`.
