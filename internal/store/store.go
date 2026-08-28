// Package store persists users, decks and games. SQLite keeps deployment a
// single binary; the schema is deliberately plain so a Postgres driver can
// be swapped in behind the same interface.
package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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
  token TEXT,                   -- opaque public identifier (URL-safe)
  user_id INTEGER NOT NULL REFERENCES users(id),
  name TEXT NOT NULL,
  investigator_code TEXT NOT NULL,
  slots TEXT NOT NULL,          -- JSON {cardCode: count}
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS games (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  token TEXT,                   -- opaque public identifier (URL-safe)
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
	if err != nil {
		return err
	}
	// 库里出现过 id 的表都要有公开 token 列（老库在此处一次性补齐）。
	if err := s.ensureTokenColumn("decks"); err != nil {
		return err
	}
	if err := s.ensureTokenColumn("games"); err != nil {
		return err
	}
	// 大厅字段：玩家人数上限与房主（老库默认单人/无房主）。
	if err := s.ensureColumn("games", "player_count INTEGER NOT NULL DEFAULT 1"); err != nil {
		return err
	}
	if err := s.ensureColumn("games", "host_user_id INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	return s.migrateCampaigns()
}

// columnExists reports whether a table has the named column.
func (s *Store) columnExists(table, column string) (bool, error) {
	rows, err := s.db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notNull int
		var dflt, pk any
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

// ensureColumn adds a column declaration ("name TYPE CONSTRAINTS") when the
// table predates it. table/decl come from literal call sites only.
func (s *Store) ensureColumn(table, decl string) error {
	col, _, _ := strings.Cut(decl, " ")
	exists, err := s.columnExists(table, col)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = s.db.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + decl)
	return err
}

// ensureTokenColumn adds the opaque `token` column to a table created before
// it existed, backfills one random token per row, and indexes it. Idempotent:
// safe to run on every boot.
func (s *Store) ensureTokenColumn(table string) error {
	// table comes from literal call sites, never user input.
	hasToken, err := s.columnExists(table, "token")
	if err != nil {
		return err
	}
	if !hasToken {
		if _, err := s.db.Exec(`ALTER TABLE ` + table + ` ADD COLUMN token TEXT`); err != nil {
			return err
		}
	}
	backfill, err := s.db.Query(`SELECT id FROM ` + table + ` WHERE token IS NULL OR token = ''`)
	if err != nil {
		return err
	}
	var pending []int64
	for backfill.Next() {
		var id int64
		if err := backfill.Scan(&id); err != nil {
			backfill.Close()
			return err
		}
		pending = append(pending, id)
	}
	if err := backfill.Err(); err != nil {
		backfill.Close()
		return err
	}
	backfill.Close()
	for _, id := range pending {
		tok, err := newToken()
		if err != nil {
			return err
		}
		if _, err := s.db.Exec(`UPDATE `+table+` SET token = ? WHERE id = ?`, tok, id); err != nil {
			return err
		}
	}
	_, err = s.db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_` + table + `_token ON ` + table + `(token)`)
	return err
}

// newToken returns an unguessable 16-char URL-safe identifier (96 bits of
// entropy) used in place of sequential ids in public URLs and API payloads.
func newToken() (string, error) {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf[:]), nil
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

// Deck's public identity is Token (serialized as "id"); the sequential int64
// primary key never leaves the server.
type Deck struct {
	ID               int64          `json:"-"`
	Token            string         `json:"id"`
	UserID           int64          `json:"userId"`
	Name             string         `json:"name"`
	InvestigatorCode string         `json:"investigatorCode"`
	Slots            map[string]int `json:"slots"`
	CreatedAt        string         `json:"createdAt"`
}

func (s *Store) CreateDeck(userID int64, name, investigatorCode string, slots map[string]int) (*Deck, error) {
	slotsJSON, err := json.Marshal(slots)
	if err != nil {
		return nil, err
	}
	tok, err := newToken()
	if err != nil {
		return nil, err
	}
	res, err := s.db.Exec(
		`INSERT INTO decks (token, user_id, name, investigator_code, slots, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		tok, userID, name, investigatorCode, string(slotsJSON), time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Deck{
		ID:               id,
		Token:            tok,
		UserID:           userID,
		Name:             name,
		InvestigatorCode: investigatorCode,
		Slots:            slots,
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
	}, nil
}

const deckColumns = `id, token, user_id, name, investigator_code, slots, created_at`

func scanDeck(scanner interface{ Scan(dest ...any) error }) (*Deck, error) {
	d := &Deck{}
	var slots string
	if err := scanner.Scan(&d.ID, &d.Token, &d.UserID, &d.Name, &d.InvestigatorCode, &slots, &d.CreatedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(slots), &d.Slots); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Store) DecksForUser(userID int64) ([]Deck, error) {
	rows, err := s.db.Query(`SELECT `+deckColumns+` FROM decks WHERE user_id = ? ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Deck
	for rows.Next() {
		d, err := scanDeck(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (s *Store) DeckByID(id int64) (*Deck, error) {
	d, err := scanDeck(s.db.QueryRow(`SELECT `+deckColumns+` FROM decks WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return d, err
}

// DeckByToken resolves an opaque public deck identifier.
func (s *Store) DeckByToken(token string) (*Deck, error) {
	d, err := scanDeck(s.db.QueryRow(`SELECT `+deckColumns+` FROM decks WHERE token = ?`, token))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return d, err
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

// GameRow's public identity is Token (serialized as "id"); the sequential
// int64 primary key never leaves the server.
type GameRow struct {
	ID          int64  `json:"-"`
	Token       string `json:"id"`
	Name        string `json:"name"`
	ScenarioID  string `json:"scenarioId"`
	Difficulty  string `json:"difficulty"`
	Seed        int64  `json:"seed"`
	State       string `json:"-"`
	Status      string `json:"status"` // lobby | active | finished
	HostUserID  int64  `json:"-"`      // lobby owner; 0 for pre-lobby-era rows
	PlayerCount int    `json:"playerCount"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type GamePlayer struct {
	Slot     int    `json:"slot"`
	UserID   *int64 `json:"userId,omitempty"`
	DeckID   int64  `json:"-"` // internal; the replay API maps it to the deck token
	HeroBase string `json:"heroBase"`
}

// CreateGame inserts a game row. status is 'active' for solo games (state
// already built) or 'lobby' for multiplayer games (state built at start).
func (s *Store) CreateGame(name, scenarioID, difficulty string, seed int64, state, status string, hostUserID int64, playerCount int, players []GamePlayer) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	tok, err := newToken()
	if err != nil {
		return 0, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(
		`INSERT INTO games (token, name, scenario_id, difficulty, seed, state, status, host_user_id, player_count, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tok, name, scenarioID, difficulty, seed, state, status, hostUserID, playerCount, now, now,
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

const gameColumns = `id, token, name, scenario_id, difficulty, seed, state, status, host_user_id, player_count, created_at, updated_at`

func scanGame(scanner interface{ Scan(dest ...any) error }) (*GameRow, error) {
	g := &GameRow{}
	err := scanner.Scan(&g.ID, &g.Token, &g.Name, &g.ScenarioID, &g.Difficulty, &g.Seed, &g.State, &g.Status, &g.HostUserID, &g.PlayerCount, &g.CreatedAt, &g.UpdatedAt)
	return g, err
}

func (s *Store) GameByID(id int64) (*GameRow, error) {
	g, err := scanGame(s.db.QueryRow(`SELECT `+gameColumns+` FROM games WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return g, err
}

// GameIDByToken resolves an opaque public game identifier to its internal id.
func (s *Store) GameIDByToken(token string) (int64, error) {
	var id int64
	err := s.db.QueryRow(`SELECT id FROM games WHERE token = ?`, token).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return id, err
}

func (s *Store) ListGames() ([]GameRow, error) {
	rows, err := s.db.Query(`SELECT ` + gameColumns + ` FROM games ORDER BY id DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GameRow
	for rows.Next() {
		g, err := scanGame(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *g)
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

// JoinLobbySlot claims a lobby slot for a user with their chosen deck. The
// row is upserted so an already-joined player can change their deck.
func (s *Store) JoinLobbySlot(gameID, userID, deckID int64, heroBase string, slot int) error {
	var existing int
	err := s.db.QueryRow(`SELECT slot FROM game_players WHERE game_id = ? AND user_id = ?`, gameID, userID).Scan(&existing)
	if err == nil {
		_, err = s.db.Exec(`UPDATE game_players SET deck_id = ?, hero_base = ? WHERE game_id = ? AND user_id = ?`, deckID, heroBase, gameID, userID)
		return err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO game_players (game_id, slot, user_id, deck_id, hero_base) VALUES (?, ?, ?, ?, ?)`,
		gameID, slot, userID, deckID, heroBase,
	)
	return err
}

// RemoveLobbyPlayer drops a joined player from a lobby (host kick).
func (s *Store) RemoveLobbyPlayer(gameID int64, slot int) error {
	res, err := s.db.Exec(`DELETE FROM game_players WHERE game_id = ? AND slot = ?`, gameID, slot)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// StartGame flips a lobby to an active game with freshly built engine state.
func (s *Store) StartGame(gameID, seed int64, state string) error {
	_, err := s.db.Exec(`UPDATE games SET seed = ?, state = ?, status = 'active', updated_at = ? WHERE id = ?`,
		seed, state, time.Now().UTC().Format(time.RFC3339), gameID)
	return err
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
