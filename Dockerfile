FROM golang:1.27-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/huginn ./cmd/huginn

FROM mcr.microsoft.com/playwright:v1.49.0-jammy

WORKDIR /app
COPY --from=build /out/huginn /usr/local/bin/huginn
COPY .env.example /app/.env.example

ENV HUGINN_HTTP_ADDR=:8080 \
    HUGINN_CONCURRENCY=1 \
    HUGINN_DATA_DIR=/data

VOLUME ["/data"]
EXPOSE 8080

ENTRYPOINT ["huginn"]
CMD ["serve", "--addr", ":8080"]
