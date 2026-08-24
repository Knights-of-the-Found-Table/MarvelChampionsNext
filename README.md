# Marvel Champions Next

A full-stack web implementation of [Marvel Champions: The Card Game](https://fantasyflightgames.com/en/products/marvel-champions-the-card-game/): a Go game engine, a REST/WebSocket server, and a React frontend.

- Card data from the [marvelcdb.com](https://marvelcdb.com) public API (61 packs / 4000+ cards, snapshot bundled in the repo)
- Card images fetched on demand from marvelcdb and cached locally, with permanent hash-URL caching
- Twenty playable heroes: Core Set five, Captain America, Ms. Marvel, Daredevil (Sense deck), Doctor Strange (Invocation deck), Angel, Nightcrawler (Bamf!), Echo, Groot, Rocket Raccoon, Colossus, Shadowcat, Cable, Domino, Tigra and Hulkling
- Sixteen playable scenarios: Rhino, Klaw, Ultron, Green Goblin ×2, Drang, Collector, Nebula, Sabretooth, Sentinels, Master Mold, Brotherhood, Marauders, Juggernaut, Mister Sinister and the Superhero Registration Act; cards without hand-written logic fall back to generic behavior
- Up to 4 players, spectating, undo and replay
- Full-viewport, Marvel Duel-inspired battle table with large card previews, responsive enemy/player zones, payment drawer, and reduced-motion support

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

## Battle UI

The game board uses a full-viewport table layout with dedicated villain, shared,
player-asset, hero, and hand zones. Hovered cards open a pointer-transparent
preview up to `72vh`; heroes and villains receive restrained character effects
that respect reduced-motion settings. See [docs/BATTLE_UI.md](docs/BATTLE_UI.md)
for the design, interaction guarantees, implementation map, and validation
checklist.

## Configuration (environment variables)

| Variable | Default | Description |
| --- | --- | --- |
| `MC_JWT_SECRET` | random per start | Secret used to sign login tokens. **Set a stable value in production** — otherwise every restart invalidates all logins. The dev scripts fix it to a known dev-only value; `deploy/` requires it (see `deploy/.env.example`). |
| `MC_LISTEN` | `:3000` | Listen address. |
| `MC_DB_PATH` | `marvelchampions.db` | SQLite database file. |
| `MC_STATIC_DIR` | `web/dist` | Directory with the built frontend. |
| `MC_CACHE_DIR` | `cache` | On-demand card image cache. |
| `MC_LOG_LEVEL` | `info` | Server log level for the structured (text) stderr log: `debug`, `info`, `warn` or `error`. |
| `MC_PREWARM_IMAGES` | off | Background cache prewarm at startup so every image URL is content-hashed. Off by default; set `1` to prewarm the whole card set at boot (the default chain, and the zh chain when a Chinese mirror is configured). |
| `IMAGE_MIRROR` | `https://marvelcdb.com` | Default-language (English) image mirror root, keyed by the face convention (`/bundles/cards/{code}.png`). The marvelcdb.com default is special-cased to its own legacy per-face paths. |
| `ZH_IMAGE_MIRROR` | empty | Chinese image mirror root — a separate language source keyed by the face convention (`/bundles/cards/{code}.png`, `{base}a`/`{base}b` for double-sided cards). |

A repo-root `.env` file (see `.env.example`) is loaded by the server on
startup — real environment variables win — and passed into the docker
container by `deploy/docker-compose.yml` via `env_file`.

### Image mirrors

Card images are fetched on demand and cached locally. The repo standardizes
on the double-sided face convention — `{base}a.png` is the A face,
`{base}b.png` the B face — and main schemes are registered by their b-face
code (the gameplay side, which carries the threat stats). marvelcdb's own
storage predates and often contradicts that convention (for some schemes
`{base}.png` holds the B-face image and `{base}b.png` the A-face image), so
the two language chains resolve paths differently:

- **default (English)**: `IMAGE_MIRROR` — always requested by convention
  path `/bundles/cards/{code}.png`, so the mirror must be keyed that way.
  The marvelcdb.com default (no `IMAGE_MIRROR` configured) is the one
  special case: it is requested through the per-face paths recorded in the
  normalized card data (`tools/normalize_faces.py` keeps them correct),
  because marvelcdb's own layout predates and often contradicts the
  convention (for some schemes `{base}.png` holds the B-face image and
  `{base}b.png` the A-face image).
- **Chinese**: `ZH_IMAGE_MIRROR` — same convention paths, a separate
  language source. Codes it lacks fall back to the default root. Locally
  seeded faces in `cache/images/zh` (see `tools/seed_zh_images.py`) always
  take priority, and `tools/zh/audit_mirror.py` writes a gap report for
  the mirror maintainer.

A missing path is retried with the other extensions (`.png`/`.jpg`/`.webp`)
so mirrors keyed differently still work, and the MIME type is detected from
content regardless of the URL.

Set `MC_PREWARM_IMAGES=1` to prewarm the whole card set in the background
at startup, completing the hash manifest so image
URLs are content-addressed (`/img/cards/{code}.{hash}.png`) and cached
immutably by browsers; un-hashed URLs and manifests revalidate via ETag,
and a zh fallback (no Chinese face exists, the default-language image is
served) caches for a day. The cache persists in `MC_CACHE_DIR` (the docker
image keeps it on the `/data` volume), so repeat boots only fetch what is
missing.
