# Huginn

Huginn is a lightweight, self-hosted business intelligence **lead discovery** engine.

It discovers local businesses through Google Maps searches, extracts structured fields, normalizes and deduplicates results, and exports lead datasets for downstream BI or CRM use.

```
SEARCH → EXTRACT → NORMALIZE → DEDUPLICATE → EXPORT
```

## What Huginn does

- Local business discovery from Google Maps queries
- Lead-oriented field extraction (Place ID, contact, geo, ratings, hours, status)
- Deterministic normalization and duplicate detection
- JSON / CSV / SQLite export
- CLI and optional REST API with in-memory async jobs

## What Huginn deliberately does not do

- General-purpose scraping platform features
- SaaS multi-tenant product (auth, billing, subscriptions, admin)
- CAPTCHA bypass, authentication circumvention, or anti-bot evasion
- Redis / Kafka / Kubernetes / distributed worker fleets
- Recreating the full upstream scraper product surface

Users must respect applicable Google terms, privacy requirements, and data-use restrictions. Prefer official Google Maps Platform APIs when they fit the use case.

## Architecture

```
cmd/huginn                 CLI + serve entrypoint
internal/scraper           Scraper interface + scrapemate adapter
internal/upstream/gmaps    Inherited Maps extraction (MIT)
internal/normalization     Phone/URL/geo/timestamp normalization
internal/deduplication     Place ID + fallback identity
internal/exporters         JSON, CSV, SQLite
internal/pipeline          End-to-end orchestration
internal/jobs              In-memory async job manager
internal/api               REST endpoints
```

Extraction is powered by adapted code from [gosom/google-maps-scraper](https://github.com/gosom/google-maps-scraper) behind a small interface. See [docs/extraction-feasibility.md](docs/extraction-feasibility.md) and [NOTICE.md](NOTICE.md).

## Installation

Requirements: Go 1.22+ (developed with Go 1.27), Playwright Chromium.

```bash
git clone <this-repo> huginn
cd huginn
go mod tidy
go run ./cmd/huginn install-browsers
go build -o huginn ./cmd/huginn
```

Docker:

```bash
docker compose up --build
```

## CLI usage

```bash
# Discover leads
./huginn search "barbershops" --location "São José dos Campos, SP" --limit 100 --output leads.json

./huginn search "dentists" --location "São Paulo, SP" --limit 200 --format csv --output leads.csv

# Optional website email enrichment
./huginn search "clinics" --location "Campinas, SP" --email --output leads.json

# Re-export
./huginn export --input leads.json --format csv --output leads.csv
./huginn export --input leads.json --format sqlite --output leads.db

# REST API
./huginn serve --addr :8080
```

Useful flags: `--depth`, `--geo lat,lon`, `--zoom`, `--concurrency`, `--pace-ms`, `--grid-bbox`, `--debug`.

## REST API usage

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/search` | Start async search (`202` + job) |
| `GET` | `/api/v1/jobs` | List jobs |
| `GET` | `/api/v1/jobs/{id}` | Job status |
| `GET` | `/api/v1/jobs/{id}/results` | Completed results |
| `GET` | `/api/v1/health` | Health check |

Example:

```bash
curl -s -X POST localhost:8080/api/v1/search \
  -H 'content-type: application/json' \
  -d '{"query":"barbershops","location":"São José dos Campos, SP","max_results":50}'
```

Search is asynchronous. Poll the job until `status` is `completed`, then fetch `/results`.

## Configuration

See [`.env.example`](.env.example). Common variables:

| Variable | Default | Meaning |
|----------|---------|---------|
| `HUGINN_CONCURRENCY` | `1` | Scrape workers |
| `HUGINN_DEPTH` | `10` | Feed scroll depth |
| `HUGINN_LANG` | `en` | Maps language |
| `HUGINN_REQUEST_PACE_MS` | `500` | Pre-scrape delay |
| `HUGINN_HTTP_ADDR` | `:8080` | API listen address |
| `HUGINN_PROXIES` | empty | Optional proxy list |

## Output schema

Each lead is a `Business` object:

| Field | Type | Notes |
|-------|------|-------|
| `place_id` | string | Preferred identity key |
| `name` | string | |
| `category` / `categories` | string / array | |
| `address`, `city`, `state`, `country`, `postal_code` | string | |
| `latitude`, `longitude` | number | |
| `phone` | string | Digits with optional leading `+` |
| `website` | string | Canonical URL |
| `maps_url` | string | |
| `rating`, `review_count` | number / int | |
| `opening_hours` | object | Day → time ranges |
| `business_status` | string | When available |
| `emails` | array | Only with `--email` |
| `extracted_at` | RFC3339 | |
| `raw` | object | Original extraction payload |

Deduplication prefers `place_id`, then website / phone / name+geo.

## Testing

Default suite is offline (fixtures + unit tests):

```bash
go test ./...
```

Opt-in live browser extraction:

```bash
HUGINN_LIVE=1 go test -tags=live ./internal/scraper -run Live -count=1
```

## Upstream attribution and licensing

- Huginn: MIT — see [LICENSE](LICENSE)
- Inherited extraction: MIT — [gosom/google-maps-scraper](https://github.com/gosom/google-maps-scraper), Georgios Komninos — see [NOTICE.md](NOTICE.md)

Huginn is independently buildable and does not require the upstream repository at runtime.
