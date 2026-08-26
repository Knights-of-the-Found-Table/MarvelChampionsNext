package store

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// TestTokenPersistsAcrossReopen guards the migration contract: tokens are
// generated once, survive a close/reopen cycle, and ensureTokenColumn is
// idempotent on every boot.
func TestTokenPersistsAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.db")
	st, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := st.CreateUser("u1", "hash"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	deck, err := st.CreateDeck(1, "Deck", "01001a", map[string]int{"01088": 1})
	if err != nil {
		t.Fatalf("create deck: %v", err)
	}
	if len(deck.Token) != 16 {
		t.Fatalf("token should be 16 chars, got %q", deck.Token)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	st2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer st2.Close()
	got, err := st2.DeckByToken(deck.Token)
	if err != nil {
		t.Fatalf("deck by token after reopen: %v", err)
	}
	if got.ID != deck.ID || got.Name != deck.Name {
		t.Fatalf("wrong deck for token: got %+v", got)
	}
}

// TestMigrateBackfillsPreTokenDatabase opens the store on a database that
// still has the old schema (no token columns) and verifies the columns are
// added and every existing row gets a distinct token.
func TestMigrateBackfillsPreTokenDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}
	if _, err := raw.Exec(`
CREATE TABLE decks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL REFERENCES users(id),
  name TEXT NOT NULL,
  investigator_code TEXT NOT NULL,
  slots TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE games (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  scenario_id TEXT NOT NULL,
  difficulty TEXT NOT NULL DEFAULT 'standard',
  seed INTEGER NOT NULL,
  state TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
INSERT INTO decks (user_id, name, investigator_code, slots, created_at)
  VALUES (1, 'Old', '01001a', '{}', '2025-01-01T00:00:00Z');
INSERT INTO decks (user_id, name, investigator_code, slots, created_at)
  VALUES (1, 'Old2', '01001a', '{}', '2025-01-01T00:00:00Z');
INSERT INTO games (name, scenario_id, difficulty, seed, state, status, created_at, updated_at)
  VALUES ('Legacy game', '01097', 'standard', 1, '{}', 'active', '2025-01-01T00:00:00Z', '2025-01-01T00:00:00Z');
`); err != nil {
		t.Fatalf("seed legacy schema: %v", err)
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("close raw: %v", err)
	}

	st, err := Open(path)
	if err != nil {
		t.Fatalf("migrate legacy db: %v", err)
	}
	defer st.Close()

	decks, err := st.DecksForUser(1)
	if err != nil {
		t.Fatalf("decks: %v", err)
	}
	if len(decks) != 2 {
		t.Fatalf("expected 2 legacy decks, got %d", len(decks))
	}
	seen := map[string]bool{}
	for _, d := range decks {
		if len(d.Token) != 16 {
			t.Fatalf("legacy deck not backfilled: %+v", d)
		}
		if seen[d.Token] {
			t.Fatalf("duplicate token %q", d.Token)
		}
		seen[d.Token] = true
	}
	games, err := st.ListGames()
	if err != nil {
		t.Fatalf("games: %v", err)
	}
	if len(games) != 1 || len(games[0].Token) != 16 {
		t.Fatalf("legacy game not backfilled: %+v", games)
	}

	// The backfilled tokens must resolve through the public lookup.
	if _, err := st.GameIDByToken(games[0].Token); err != nil {
		t.Fatalf("game token does not resolve: %v", err)
	}
}
