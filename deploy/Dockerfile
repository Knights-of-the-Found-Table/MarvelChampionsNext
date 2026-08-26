# syntax=docker/dockerfile:1

# --- Build the Go server (card data embedded, images NOT fetched) ---
FROM golang:1.26-alpine AS gobuild
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/server ./cmd/server

# --- Build the frontend ---
FROM node:24-alpine AS webbuild
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci --no-fund --no-audit
COPY web/ .
RUN npm run build

# --- Final image: single binary + static assets ---
# Card images are NOT downloaded at build time. At container runtime the
# server fetches them on demand through its source chain into MC_CACHE_DIR
# (/data/cache — persisted on the volume below); setting MC_PREWARM_IMAGES=1
# in the environment instead prewarms every card in the background at boot.
FROM alpine:3.20
RUN adduser -D -u 10001 marvel && mkdir -p /data /app && chown -R marvel /data /app
USER marvel
WORKDIR /app
COPY --from=gobuild /out/server /app/server
COPY --from=gobuild /src/tools/zh/out /app/tools/zh/out
COPY --from=webbuild /web/dist /app/web/dist
ENV MC_LISTEN=:3000 \
    MC_DB_PATH=/data/marvelchampions.db \
    MC_CACHE_DIR=/data/cache \
    MC_STATIC_DIR=/app/web/dist
VOLUME ["/data"]
EXPOSE 3000
CMD ["/app/server"]
