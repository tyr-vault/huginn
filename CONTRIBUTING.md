# Contributing to Huginn

## Scope

Huginn is a focused lead-discovery engine: **search → extract → normalize → deduplicate → export**.

Please avoid contributions that reintroduce SaaS features, multi-user auth, billing, CAPTCHA bypasses, anti-bot evasion, or distributed worker infrastructure.

## Development setup

```bash
go mod tidy
go run ./cmd/huginn install-browsers
go test ./...
```

Live browser tests (opt-in):

```bash
HUGINN_LIVE=1 go test -tags=live ./internal/scraper -run Live -count=1
```

## Upstream boundary

Inherited Maps extraction lives under `internal/upstream/`. Prefer fixing extraction bugs there with minimal diffs and fixture updates. New BI/product behavior belongs in Huginn packages outside `upstream/`.

When syncing from [gosom/google-maps-scraper](https://github.com/gosom/google-maps-scraper), preserve MIT attribution in `NOTICE.md`.

## Pull requests

1. Keep changes small and purposeful.
2. Add or update tests for normalization, dedupe, exporters, and API behavior.
3. Do not make the default test suite depend on live Google Maps access.
4. Run `gofmt` and `go test ./...` before submitting.
