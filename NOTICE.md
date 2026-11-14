# NOTICE

Huginn includes adapted source code originally from:

**google-maps-scraper**  
https://github.com/gosom/google-maps-scraper  
Copyright (c) 2023 Georgios Komninos  
Licensed under the MIT License (see upstream `LICENSE`).

## Adapted packages

The following Huginn packages contain substantial portions of the upstream project:

- `internal/upstream/gmaps`
- `internal/upstream/deduper`
- `internal/upstream/exiter`
- `internal/upstream/grid`
- `internal/upstream/proxyconfig`
- Fixture data under `testdata/` and `internal/upstream/testdata/`

Huginn product packages (`internal/models`, `normalization`, `deduplication`, `exporters`, `scraper` adapter, `pipeline`, `jobs`, `api`, `config`, `discovery`, and `cmd/huginn`) are original to this project.

Runtime extraction depends on:

- `github.com/gosom/scrapemate`
- Playwright (via `github.com/mxschmitt/playwright-go`)

Huginn does **not** depend on the google-maps-scraper repository at runtime.

The MIT copyright notice and permission text for the upstream project are reproduced below.

---

MIT License

Copyright (c) 2023 Georgios Komninos

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
