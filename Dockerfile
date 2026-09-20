# ---------- Build Stage ----------
FROM debian:bookworm AS builder

RUN apt-get update && apt-get install -y wget tar gcc libc6-dev ca-certificates

# Match Go to the target image and its native C compiler (SQLite requires CGO).
ARG TARGETARCH
ENV GOLANG_VERSION=1.23.7
RUN wget -q https://go.dev/dl/go${GOLANG_VERSION}.linux-${TARGETARCH}.tar.gz && \
    tar -C /usr/local -xzf go${GOLANG_VERSION}.linux-${TARGETARCH}.tar.gz

ENV PATH="/usr/local/go/bin:${PATH}"
ENV CGO_ENABLED=1 GOOS=linux

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /app/bin/app ./cmd/web

# ---------- Final Stage ----------
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates libsqlite3-0 && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/bin/app /app/bin/app
COPY --from=builder /app/views /app/views
COPY --from=builder /app/public /app/public

WORKDIR /app
EXPOSE 8080

CMD ["/app/bin/app"]
