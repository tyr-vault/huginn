# Huginn Extraction Feasibility Report

**Phase:** 1 — Repository analysis  
**Source:** [`github.com/gosom/google-maps-scraper`](https://github.com/gosom/google-maps-scraper) (local mirror: `../google-maps-scraper`)  
**License:** MIT — Copyright (c) 2023 Georgios Komninos  
**Date:** 2026-11-13  

This report documents what can be reused for Huginn. The upstream repository must not be modified; Huginn is a sibling product under `personal-tools/huginn`.

---

## 1. Executive verdict

**Feasible.** The reliable Google Maps extraction path is concentrated in the `gmaps` package, driven by `github.com/gosom/scrapemate` and Playwright. Huginn should preserve that path behind a small interface, discard SaaS/web/telemetry surfaces, and add BI-oriented normalize → dedupe → export layers.

Default v1 mode: **full Playwright crawl** (`GmapJob` → feed scroll → `PlaceJob` → `EntryFromJSON`). Fast HTTP mode is deferred (weaker Place ID coverage).

---

## 2. Scraping engine

| Component | Location | Role |
|-----------|----------|------|
| Job orchestrator | `github.com/gosom/scrapemate` | Concurrency, retries, browser pool, result writers |
| Maps search job | `gmaps/job.go` (`GmapJob`) | Navigate search URL, scroll results feed, enqueue place jobs |
| Place extract job | `gmaps/place.go` (`PlaceJob`) | Open place page, read `APP_INITIALIZATION_STATE` |
| Parser | `gmaps/entry.go` (`Entry`, `EntryFromJSON`) | Nested-array JSON → structured place record |
| Fast mode (out of v1) | `gmaps/searchjob.go`, `multiple.go` | Stealth HTTP search; thinner fields |
| Optional email | `gmaps/emailjob.go` | Website harvest; not core |
| Optional reviews | `gmaps/reviews.go` | Extra review fetch; not core |

There is **no second browser automation implementation**. Do not introduce rod/chromedp.

---

## 3. Browser automation

- **Library:** Playwright Chromium via scrapemate `WithJS` (`github.com/mxschmitt/playwright-go`).
- **Lifecycle:** scrapemate app `New` / `Start` / `Close`; process signals cancel root context.
- **Pool:** concurrency (`-c`), browser pool size, pages-per-browser, page/browser reuse limits.
- **Install:** dedicated install-playwright run mode / env flag.
- **Huginn:** expose bounded concurrency + install helper; always call `Close` on shutdown.

---

## 4. Navigation, pagination, search workflow

```
SearchRequest
  → seed GmapJob(s)
  → BrowserActions: goto Maps search URL, reject cookies
  → wait for div[role=feed] OR single /maps/place/ redirect
  → scroll feed (MaxDepth iterations; stop on unchanged height / ctx)
  → Process: parse feed links → PlaceJob each (href dedupe)
  → PlaceJob: extract JSON → EntryFromJSON → results
```

**URL construction** (`NewGmapJob`):

- Raw Maps URL → sanitized place URL.
- Else `geo` + `zoom` → `/maps/search/{query}/@{lat,lon},{zoom}z`.
- Else → `/maps/search/{query}` with `hl` language param.

**Pagination:** infinite scroll on the feed, controlled by depth — not classic page buttons.

**Location mapping for Huginn:** encode human location into the query string (e.g. `barbershops in São José dos Campos, SP`) and/or optional lat/lon+zoom when provided. Grid (`grid/`) can expand coverage beyond ~120 results per search.

---

## 5. Data model (`gmaps.Entry`) — BI-relevant fields

| Huginn need | Upstream field(s) | Available in full mode |
|-------------|-------------------|------------------------|
| Place ID | `PlaceID` | Yes |
| Name | `Title` | Yes |
| Category | `Category`, `Categories` | Yes |
| Address | `Address`, `CompleteAddress` | Yes (city/state/country/postal) |
| Lat / Lon | `Latitude`, `Longtitude` | Yes |
| Phone | `Phone` | Yes |
| Website | `WebSite` | Yes |
| Maps URL | `Link` | Yes |
| Rating | `ReviewRating` | Yes |
| Review count | `ReviewCount` | Yes |
| Opening hours | `OpenHours` | Yes |
| Status | `Status` | Yes |
| Email | `Emails` | Only with optional email job |

Do not fabricate unsupported fields. Preserve raw `Entry` (or JSON) when useful for audit.

---

## 6. Concurrency, errors, retries, rate limiting

| Concern | Upstream | Huginn plan |
|---------|----------|-------------|
| Concurrency | scrapemate worker pool | Config `concurrency` (conservative default) |
| Job retries | `MaxRetries: 3` on Maps jobs | Preserve |
| Place JSON | Poll + reload up to 2 attempts | Preserve |
| Cancellation | Context + `exiter` when seeds/places done | Preserve + graceful CLI/API shutdown |
| Rate limit | None in core; proxies optional | Huginn pacing delay config; optional proxies later |
| CAPTCHA / anti-bot | Not implemented | **Must not** add bypasses |

---

## 7. Existing normalization and dedupe

**Present upstream (keep inside extraction):**

- Place URL sanitize; Google redirect URL unwrap for websites.
- Address title-prefix trim; hours/status schema fallbacks.
- Fast-mode phone space strip; plus-code helpers.
- In-run href FNV dedupe (`deduper`).

**Missing (build in Huginn):**

- Phone E.164-style normalization.
- Website URL canonicalization.
- Structured geo standardization and missing-value policy.
- Business-level dedupe (Place ID primary; fallback website + phone + name + geo).
- Extraction timestamps.

---

## 8. Essential vs exclude inventory

### Copy / adapt into Huginn (`internal/upstream/...`)

| Path | Reason |
|------|--------|
| `gmaps/*.go` (+ `gmaps/testdata/`) | Extraction core |
| Root `testdata/` (raw/panic/output fixtures) | Deterministic parser tests |
| `deduper/` | Crawl-time href dedupe |
| `exiter/` | Completion → cancel |
| `grid/` | Optional geo cell expansion |
| `internal/proxyconfig/` | Optional proxy list parsing |
| Patterns from `runner/jobs.go`, `runner/filerunner` | Seed creation / scrapemate wiring (rewrite, do not copy SaaS) |

### Exclude

`cmd/gmapssaas/`, `api/`, `admin/`, `rqueue/`, `saas/`, `migrations/`, `infra/`, `web/` + `webdata/`, `tlmt/`, `ratelimit/`, `leadsdb/`, `s3uploader/`, `runner/lambdaaws/`, `runner/databaserunner/`, `runner/webrunner/`, `postgres/` (SaaS path), sponsors/marketing markdown, PostHog telemetry.

---

## 9. License obligations

MIT requires retention of copyright and permission notice in all copies or substantial portions. Huginn will:

1. Ship its own `LICENSE` (project license).
2. Ship `NOTICE.md` attributing Georgios Komninos / google-maps-scraper and listing adapted files.
3. Keep copyright headers / notices on adapted upstream sources where present.

Runtime dependency on the **original git repository** is not required. Build-time deps: `scrapemate`, Playwright, goquery, etc.

---

## 10. Risks and technical debt

1. **Schema drift:** `EntryFromJSON` depends on brittle nested indexes; Google can break parsing — mitigate with fixture tests and careful upstream syncs.
2. **Playwright weight:** larger Docker image; document install steps.
3. **Terms of use:** document that users must respect Google terms, privacy, and data-use rules; prefer official APIs where they fit; no evasion features.
4. **Location without geocoding:** human location strings are folded into the Maps query; precise radius requires geo+zoom (and/or grid).
5. **Fast mode omitted in v1:** may revisit if Place ID coverage improves.

---

## 11. Recommended Huginn architecture (post Phase 1)

```
SEARCH → EXTRACT (upstream gmaps/scrapemate) → NORMALIZE → DEDUPLICATE → EXPORT
```

```go
type Scraper interface {
    Search(ctx context.Context, req SearchRequest) ([]Business, error)
}
```

Package boundary: `internal/upstream` = inherited extraction; all other `internal/*` = Huginn product code.

---

## 12. Phase 1 exit criteria

- [x] Scraping engine identified
- [x] Navigation / extraction / browser / models / workflow documented
- [x] Concurrency, retries, rate limiting, lifecycle documented
- [x] Normalization gaps and license obligations documented
- [x] Keep vs exclude inventory produced
- [x] Upstream repo left unmodified

**Next:** Phase 2 — engine isolation into `huginn` module.
