// Package store persists users, decks and games. SQLite keeps deployment a
// single binary; the schema is deliberately plain so a Postgres driver can
// be swapped in behind the same interface.
package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // sqlite writes serialize better with a single conn
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  username TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS decks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL REFERENCES users(id),
  name TEXT NOT NULL,
  investigator_code TEXT NOT NULL,
  slots TEXT NOT NULL,          -- JSON {cardCode: count}
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS games (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  scenario_id TEXT NOT NULL,
  difficulty TEXT NOT NULL DEFAULT 'standard',
  seed INTEGER NOT NULL,
  state TEXT NOT NULL,          -- serialized engine.Game
  status TEXT NOT NULL,        -- lobby | active | finished
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS game_players (
  game_id INTEGER NOT NULL REFERENCES games(id),
  slot INTEGER NOT NULL,
  user_id INTEGER REFERENCES users(id),  -- claimed owner; NULL = open
  deck_id INTEGER NOT NULL REFERENCES decks(id),
  hero_base TEXT NOT NULL,
  PRIMARY KEY (game_id, slot)
);
CREATE TABLE IF NOT EXISTS game_snapshots (
  game_id INTEGER NOT NULL REFERENCES games(id),
  seq INTEGER NOT NULL,
  state TEXT NOT NULL,
  PRIMARY KEY (game_id, seq)
);
CREATE TABLE IF NOT EXISTS game_actions (
  game_id INTEGER NOT NULL REFERENCES games(id),
  seq INTEGER NOT NULL,
  player TEXT NOT NULL,
  paths TEXT NOT NULL,          -- JSON [string]
  PRIMARY KEY (game_id, seq)
);
`)
	return err
}

// ---------------------------------------------------------------- users

type User struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
	CreatedAt    string `json:"createdAt"`
}

func (s *Store) CreateUser(username, passwordHash string) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO users (username, password_hash, created_at) VALUES (?, ?, ?)`,
		username, passwordHash, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UserByName(username string) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(`SELECT id, username, password_hash, created_at FROM users WHERE username = ?`, username).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (s *Store) UserByID(id int64) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(`SELECT id, username, password_hash, created_at FROM users WHERE id = ?`, id).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

// ---------------------------------------------------------------- decks

type Deck struct {
	ID               int64          `json:"id"`
	UserID           int64          `json:"userId"`
	Name             string         `json:"name"`
	InvestigatorCode string         `json:"investigatorCode"`
	Slots            map[string]int `json:"slots"`
	CreatedAt        string         `json:"createdAt"`
}

func (s *Store) CreateDeck(userID int64, name, investigatorCode string, slots map[string]int) (int64, error) {
	slotsJSON, err := json.Marshal(slots)
	if err != nil {
		return 0, err
	}
	res, err := s.db.Exec(
		`INSERT INTO decks (user_id, name, investigator_code, slots, created_at) VALUES (?, ?, ?, ?, ?)`,
		userID, name, investigatorCode, string(slotsJSON), time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) DecksForUser(userID int64) ([]Deck, error) {
	rows, err := s.db.Query(`SELECT id, user_id, name, investigator_code, slots, created_at FROM decks WHERE user_id = ? ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Deck
	for rows.Next() {
		d := Deck{}
		var slots string
		if err := rows.Scan(&d.ID, &d.UserID, &d.Name, &d.InvestigatorCode, &slots, &d.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(slots), &d.Slots); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) DeckByID(id int64) (*Deck, error) {
	d := &Deck{}
	var slots string
	err := s.db.QueryRow(`SELECT id, user_id, name, investigator_code, slots, created_at FROM decks WHERE id = ?`, id).
		Scan(&d.ID, &d.UserID, &d.Name, &d.InvestigatorCode, &slots, &d.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(slots), &d.Slots); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Store) DeleteDeck(userID, deckID int64) error {
	res, err := s.db.Exec(`DELETE FROM decks WHERE id = ? AND user_id = ?`, deckID, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ---------------------------------------------------------------- games

type GameRow struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	ScenarioID string `json:"scenarioId"`
	Difficulty string `json:"difficulty"`
	Seed       int64  `json:"seed"`
	State      string `json:"-"`
	Status     string `json:"status"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

type GamePlayer struct {
	Slot     int    `json:"slot"`
	UserID   *int64 `json:"userId,omitempty"`
	DeckID   int64  `json:"deckId"`
	HeroBase string `json:"heroBase"`
}

func (s *Store) CreateGame(name, scenarioID, difficulty string, seed int64, state string, players []GamePlayer) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(
		`INSERT INTO games (name, scenario_id, difficulty, seed, state, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, 'active', ?, ?)`,
		name, scenarioID, difficulty, seed, state, now, now,
	)
	if err != nil {
		return 0, err
	}
	gameID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	for _, p := range players {
		if _, err := tx.Exec(
			`INSERT INTO game_players (game_id, slot, user_id, deck_id, hero_base) VALUES (?, ?, ?, ?, ?)`,
			gameID, p.Slot, nullableInt64(p.UserID), p.DeckID, p.HeroBase,
		); err != nil {
			return 0, err
		}
	}
	return gameID, tx.Commit()
}

func nullableInt64(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

func (s *Store) GameByID(id int64) (*GameRow, error) {
	g := &GameRow{}
	err := s.db.QueryRow(
		`SELECT id, name, scenario_id, difficulty, seed, state, status, created_at, updated_at FROM games WHERE id = ?`, id,
	).Scan(&g.ID, &g.Name, &g.ScenarioID, &g.Difficulty, &g.Seed, &g.State, &g.Status, &g.CreatedAt, &g.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return g, err
}

func (s *Store) ListGames() ([]GameRow, error) {
	rows, err := s.db.Query(`SELECT id, name, scenario_id, difficulty, seed, state, status, created_at, updated_at FROM games ORDER BY id DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GameRow
	for rows.Next() {
		var g GameRow
		if err := rows.Scan(&g.ID, &g.Name, &g.ScenarioID, &g.Difficulty, &g.Seed, &g.State, &g.Status, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (s *Store) GamePlayers(gameID int64) ([]GamePlayer, error) {
	rows, err := s.db.Query(`SELECT slot, user_id, deck_id, hero_base FROM game_players WHERE game_id = ? ORDER BY slot`, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GamePlayer
	for rows.Next() {
		var p GamePlayer
		var uid sql.NullInt64
		if err := rows.Scan(&p.Slot, &uid, &p.DeckID, &p.HeroBase); err != nil {
			return nil, err
		}
		if uid.Valid {
			v := uid.Int64
			p.UserID = &v
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) SaveGameState(gameID int64, state string, status string) error {
	_, err := s.db.Exec(`UPDATE games SET state = ?, status = ?, updated_at = ? WHERE id = ?`,
		state, status, time.Now().UTC().Format(time.RFC3339), gameID)
	return err
}

func (s *Store) ClaimPlayerSlot(gameID int64, slot int, userID int64) error {
	res, err := s.db.Exec(`UPDATE game_players SET user_id = ? WHERE game_id = ? AND slot = ? AND user_id IS NULL`, userID, gameID, slot)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("slot %d not available", slot)
	}
	return nil
}

// ---------------------------------------------------------------- snapshots

func (s *Store) PushSnapshot(gameID int64, state string) error {
	var seq int
	if err := s.db.QueryRow(`SELECT COALESCE(MAX(seq), 0) FROM game_snapshots WHERE game_id = ?`, gameID).Scan(&seq); err != nil {
		return err
	}
	_, err := s.db.Exec(`INSERT INTO game_snapshots (game_id, seq, state) VALUES (?, ?, ?)`, gameID, seq+1, state)
	return err
}

// PopSnapshot removes and returns the latest snapshot.
func (s *Store) PopSnapshot(gameID int64) (string, error) {
	var seq int
	if err := s.db.QueryRow(`SELECT COALESCE(MAX(seq), 0) FROM game_snapshots WHERE game_id = ?`, gameID).Scan(&seq); err != nil {
		return "", err
	}
	if seq == 0 {
		return "", ErrNotFound
	}
	var state string
	if err := s.db.QueryRow(`SELECT state FROM game_snapshots WHERE game_id = ? AND seq = ?`, gameID, seq).Scan(&state); err != nil {
		return "", err
	}
	if _, err := s.db.Exec(`DELETE FROM game_snapshots WHERE game_id = ? AND seq = ?`, gameID, seq); err != nil {
		return "", err
	}
	return state, nil
}

// ---------------------------------------------------------------- actions (replay log)

func (s *Store) RecordAction(gameID int64, player string, paths []string) error {
	pathsJSON, err := json.Marshal(paths)
	if err != nil {
		return err
	}
	var seq int
	if err := s.db.QueryRow(`SELECT COALESCE(MAX(seq), 0) FROM game_actions WHERE game_id = ?`, gameID).Scan(&seq); err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO game_actions (game_id, seq, player, paths) VALUES (?, ?, ?, ?)`, gameID, seq+1, player, string(pathsJSON))
	return err
}

type RecordedAction struct {
	Seq    int      `json:"seq"`
	Player string   `json:"player"`
	Paths  []string `json:"paths"`
}

func (s *Store) GameActions(gameID int64) ([]RecordedAction, error) {
	rows, err := s.db.Query(`SELECT seq, player, paths FROM game_actions WHERE game_id = ? ORDER BY seq`, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RecordedAction
	for rows.Next() {
		var a RecordedAction
		var paths string
		if err := rows.Scan(&a.Seq, &a.Player, &paths); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(paths), &a.Paths); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
