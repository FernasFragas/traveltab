# ---------- Browser libraries ----------
FROM node:24.18.1-bookworm-slim AS assets

WORKDIR /app
COPY package.json package-lock.json ./
# npm checks each package against the lockfile's integrity hash; package scripts never run.
# Only the files the page loads are copied to public/vendor.
RUN npm ci --ignore-scripts && npm run vendor

# ---------- Build Stage ----------
FROM golang:1.27.2-bookworm AS builder

# SQLite requires CGO; the Go image includes the native C compiler.
ENV CGO_ENABLED=1 GOOS=linux

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /app/bin/app ./cmd/web

# ---------- Final Stage ----------
FROM debian:bookworm-slim

RUN apt-get update && apt-get upgrade -y && apt-get install -y ca-certificates libsqlite3-0 && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/bin/app /app/bin/app
COPY --from=builder /app/views /app/views
COPY --from=builder /app/public /app/public
COPY --from=assets /app/public/vendor /app/public/vendor

WORKDIR /app
EXPOSE 8080

CMD ["/app/bin/app"]
