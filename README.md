# Marvel Champions Next

A full-stack web implementation of [Marvel Champions: The Card Game](https://fantasyflightgames.com/en/products/marvel-champions-the-card-game/): a Go game engine, a REST/WebSocket server, and a React frontend.

- Card data from the [marvelcdb.com](https://marvelcdb.com) public API (61 packs / 4000+ cards, snapshot bundled in the repo)
- Card images fetched on demand from marvelcdb and cached locally, with permanent hash-URL caching
- Five playable scenarios: Rhino, Klaw, Ultron, Green Goblin ×2; cards without hand-written logic fall back to generic behavior
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
cd deploy && docker compose up --build -d   # http://localhost:3000
```
