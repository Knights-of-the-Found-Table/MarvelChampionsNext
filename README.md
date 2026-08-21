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
| `MC_PREWARM_IMAGES` | `auto` | Background cache prewarm at startup so every image URL is content-hashed. `auto` = on for every mirror-backed cache, off against bare marvelcdb; `1`/`0` force it. |
| `IMAGE_MIRROR` | `https://marvelcdb.com` | Default-language (English) image mirror root. Must serve marvelcdb's path layout (`/bundles/cards/{code}.png`). |
| `ZH_IMAGE_MIRROR` | empty | Chinese image mirror root — a separate language source with the same paths, like marvelcdb's language domains. |

A repo-root `.env` file (see `.env.example`) is loaded by the server on
startup — real environment variables win — and passed into the docker
container by `deploy/docker-compose.yml` via `env_file`.

### Image mirrors

Card images are fetched on demand and cached locally. Mirrors are plain
HTTP site roots serving marvelcdb's exact path layout
(`/bundles/cards/{code}.png` — the mirror decides the file format, png,
webp or otherwise; the MIME type is detected from content). Languages are
separate sources, like marvelcdb's own language domains:

- **default (English)**: `IMAGE_MIRROR` (default marvelcdb.com),
- **Chinese**: `ZH_IMAGE_MIRROR` — the zh cache fetches from it on demand;
  codes it lacks fall back to the default root. Locally seeded faces in
  `cache/images/zh` (see `tools/seed_zh_images.py`) always take priority.

With a mirror configured the server prewarms that language's whole card
set in the background (`MC_PREWARM_IMAGES`), completing the hash manifest
so image URLs are content-addressed (`/img/cards/{code}.{hash}.png`) and
cached immutably by browsers; un-hashed URLs and manifests revalidate via
ETag. The cache persists in `MC_CACHE_DIR` (the docker image keeps it on
the `/data` volume), so repeat boots only fetch what is missing.
