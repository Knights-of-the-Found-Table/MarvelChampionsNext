# syntax=docker/dockerfile:1

# --- Build Go binaries (server + fetchimages with embedded card data) ---
FROM golang:1.26-alpine AS gobuild
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/server ./cmd/server \
 && CGO_ENABLED=0 go build -trimpath -o /out/fetchimages ./cmd/fetchimages

# --- Download card images from marvelcdb (content-hash manifest) ---
# MC_PACKS limits the fetch to specific pack codes (comma separated).
FROM alpine:3.20 AS images
COPY --from=gobuild /out/fetchimages /usr/local/bin/fetchimages
RUN mkdir -p /cards && fetchimages -out /cards

# --- Build the frontend with images in place ---
FROM node:24-alpine AS webbuild
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci --no-fund --no-audit
COPY web/ .
COPY --from=images /cards public/img/cards
RUN npm run build

# --- Final image: single binary + static assets ---
FROM alpine:3.20
RUN adduser -D -u 10001 marvel && mkdir -p /data /app && chown -R marvel /data /app
USER marvel
WORKDIR /app
COPY --from=gobuild /out/server /app/server
COPY --from=webbuild /web/dist /app/web/dist
ENV MC_LISTEN=:3000 \
    MC_DB_PATH=/data/marvelchampions.db \
    MC_CACHE_DIR=/data/cache \
    MC_STATIC_DIR=/app/web/dist
VOLUME ["/data"]
EXPOSE 3000
CMD ["/app/server"]
