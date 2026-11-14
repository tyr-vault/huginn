# Legacy single-binary image. Prefer docker compose (Dockerfile.backend / scrape-engine / frontend).
FROM golang:1.27-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/huginn ./cmd/huginn

FROM mcr.microsoft.com/playwright:v1.49.0-jammy

WORKDIR /app
COPY --from=build /out/huginn /usr/local/bin/huginn

ENTRYPOINT ["huginn"]
CMD ["--help"]
