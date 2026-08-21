# Marvel Champions Next

A full-stack web implementation of [Marvel Champions: The Card Game](https://fantasyflightgames.com/en/products/marvel-champions-the-card-game/): a Go game engine, a REST/WebSocket server, and a React frontend.

- Card data from the [marvelcdb.com](https://marvelcdb.com) public API (61 packs / 4000+ cards, snapshot bundled in the repo)
- Card images fetched on demand from marvelcdb and cached locally, with permanent hash-URL caching
- Twenty playable heroes: Core Set five, Captain America, Ms. Marvel, Daredevil (Sense deck), Doctor Strange (Invocation deck), Angel, Nightcrawler (Bamf!), Echo, Groot, Rocket Raccoon, Colossus, Shadowcat, Cable, Domino, Tigra and Hulkling
- Sixteen playable scenarios: Rhino, Klaw, Ultron, Green Goblin ×2, Drang, Collector, Nebula, Sabretooth, Sentinels, Master Mold, Brotherhood, Marauders, Juggernaut, Mister Sinister and the Superhero Registration Act; cards without hand-written logic fall back to generic behavior
- Up to 4 players, spectating, undo and replay

## Quick Start

One-click dev server (backend on :3000, frontend on :8080):

```bash
./dev.sh   # Linux / macOS / Git Bash
dev.cmd    # Windows
```

Open http://localhost:8080, register, import a deck from a marvelcdb URL on the Decks page, and start a game.

Docker deployment:

```bash
cd deploy
cp .env.example .env   # then edit MC_JWT_SECRET
docker compose up --build -d   # http://localhost:3000
```

## Configuration (environment variables)

| Variable | Default | Description |
| --- | --- | --- |
| `MC_JWT_SECRET` | random per start | Secret used to sign login tokens. **Set a stable value in production** — otherwise every restart invalidates all logins. The dev scripts fix it to a known dev-only value; `deploy/` requires it (see `deploy/.env.example`). |
| `MC_LISTEN` | `:3000` | Listen address. |
| `MC_DB_PATH` | `marvelchampions.db` | SQLite database file. |
| `MC_STATIC_DIR` | `web/dist` | Directory with the built frontend. |
| `MC_CACHE_DIR` | `cache` | On-demand card image cache. |
| `MC_PREWARM_IMAGES` | `auto` | Background cache prewarm at startup so every image URL is content-hashed. `auto` = on when a mirror is configured, off against bare marvelcdb; `1`/`0` force it. |
| `R2_ACCESS_KEY_ID` / `R2_SECRET_ACCESS_KEY` | empty | Read-only credentials for the private Cloudflare R2 image mirror (S3 SigV4). |
| `R2_ENDPOINT` | derived | R2 S3 endpoint; when empty it is derived from `CLOUDFLARE_ACCOUNT_ID` as `https://{id}.r2.cloudflarestorage.com`. |
| `R2_BUCKET` | empty | Mirror bucket, using marvelcdb's exact paths. Activates when credentials, endpoint and bucket are all set. |
| `MC_MARVELCDB_IMAGES` | `https://marvelcdb.com` | HTTP image mirror (site root), used as the fallback root. Must serve marvelcdb's path layout. |

A repo-root `.env` file (see `.env.example`) is loaded by the server on
startup — real environment variables win — and passed into the docker
container by `deploy/docker-compose.yml` via `env_file`.

### Image mirrors

Card images are fetched on demand and cached locally, always under
marvelcdb's exact paths (`/bundles/cards/{code}.png`). One source chain is
shared by everything the server serves:

1. the R2 mirror when `R2_*` is configured (read-only, SigV4-signed GETs;
   `.png` paths from card data are retried as `.jpg`, the pack's original
   filenames),
2. then the HTTP root — marvelcdb.com, or `MC_MARVELCDB_IMAGES`.

The mirror's content defines what gets distributed: a bucket holding the
Chinese community card pack makes every served image Chinese, with no
language-specific source or path anywhere in the server. Codes the mirror
lacks fall back to the HTTP root. Locally seeded faces in
`cache/images/zh` (see `tools/seed_zh_images.py`) still take priority over
the chain on the zh routes.

With a mirror configured the server prewarms the whole card set in the
background (`MC_PREWARM_IMAGES`), completing the hash manifest so image
URLs are content-addressed (`/img/cards/{code}.{hash}.png`) and cached
immutably by browsers; un-hashed URLs and manifests revalidate via ETag.
The cache persists in `MC_CACHE_DIR` (the docker image keeps it on the
`/data` volume), so repeat boots only fetch what is missing.
