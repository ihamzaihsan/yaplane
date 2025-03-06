FROM golang:1.27.1-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -trimpath -o /out/forum . && \
    go build -trimpath -o /out/seed-demo ./cmd/seed-demo && \
    go build -trimpath -o /out/admin ./cmd/admin && \
    go build -trimpath -o /out/certgen ./cmd/certgen

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --gid 10001 forum \
    && useradd --uid 10001 --gid forum --no-create-home forum \
    && mkdir -p /data /app/HTTPS /app/static/uploads \
    && chown -R forum:forum /data /app
WORKDIR /app
COPY --from=build /out/ /app/
COPY --chown=forum:forum static /app/static
USER forum
RUN /app/certgen -cert /app/HTTPS/server.crt -key /app/HTTPS/server.key
ENV DATABASE_PATH=/data/forum.db TLS_CERT_FILE=/app/HTTPS/server.crt TLS_KEY_FILE=/app/HTTPS/server.key
EXPOSE 8080
CMD ["/app/forum"]
