// Campaign persistence: one campaigns row per campaign (the JSON state
// lives in the state column), campaign_players holds the seats claimed
// with their deck at join time, and games.campaign_id links every chapter
// game back to its campaign.
package store

import (
	"database/sql"
	"errors"
	"time"
)

// CampaignRow's public identity is Token (serialized as "id").
type CampaignRow struct {
	ID          int64  `json:"-"`
	Token       string `json:"id"`
	Box         string `json:"box"`
	Difficulty  string `json:"difficulty"`
	Status      string `json:"status"`
	Index       int    `json:"index"`
	State       string `json:"-"`
	HostUserID  int64  `json:"-"`
	PlayerCount int    `json:"playerCount"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type CampaignPlayer struct {
	Slot     int    `json:"slot"`
	UserID   *int64 `json:"userId,omitempty"`
	DeckID   int64  `json:"-"`
	HeroBase string `json:"heroBase"`
}

func (s *Store) migrateCampaigns() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS campaigns (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  token TEXT,
  box TEXT NOT NULL,
  difficulty TEXT NOT NULL DEFAULT 'standard',
  status TEXT NOT NULL DEFAULT 'forming',
  scenario_index INTEGER NOT NULL DEFAULT 0,
  state TEXT NOT NULL,
  host_user_id INTEGER NOT NULL DEFAULT 0,
  player_count INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS campaign_players (
  campaign_id INTEGER NOT NULL REFERENCES campaigns(id),
  slot INTEGER NOT NULL,
  user_id INTEGER REFERENCES users(id),
  deck_id INTEGER NOT NULL REFERENCES decks(id),
  hero_base TEXT NOT NULL,
  PRIMARY KEY (campaign_id, slot)
);
`)
	if err != nil {
		return err
	}
	if err := s.ensureTokenColumn("campaigns"); err != nil {
		return err
	}
	return s.ensureColumn("games", "campaign_id INTEGER NOT NULL DEFAULT 0")
}

const campaignColumns = `id, token, box, difficulty, status, scenario_index, state, host_user_id, player_count, created_at, updated_at`

func scanCampaign(scanner interface{ Scan(dest ...any) error }) (*CampaignRow, error) {
	c := &CampaignRow{}
	err := scanner.Scan(&c.ID, &c.Token, &c.Box, &c.Difficulty, &c.Status, &c.Index, &c.State, &c.HostUserID, &c.PlayerCount, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

// CreateCampaign inserts a campaign with its initial state JSON.
func (s *Store) CreateCampaign(box, difficulty string, state string, hostUserID int64, playerCount int) (*CampaignRow, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	tok, err := newToken()
	if err != nil {
		return nil, err
	}
	res, err := s.db.Exec(
		`INSERT INTO campaigns (token, box, difficulty, status, scenario_index, state, host_user_id, player_count, created_at, updated_at) VALUES (?, ?, ?, 'forming', 0, ?, ?, ?, ?, ?)`,
		tok, box, difficulty, state, hostUserID, playerCount, now, now,
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.CampaignByID(id)
}

func (s *Store) CampaignByID(id int64) (*CampaignRow, error) {
	c, err := scanCampaign(s.db.QueryRow(`SELECT `+campaignColumns+` FROM campaigns WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return c, err
}

// CampaignIDByToken resolves a public campaign token.
func (s *Store) CampaignIDByToken(token string) (int64, error) {
	var id int64
	err := s.db.QueryRow(`SELECT id FROM campaigns WHERE token = ?`, token).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return id, err
}

// CampaignsByUser lists campaigns the user hosts or plays in.
func (s *Store) CampaignsByUser(userID int64) ([]CampaignRow, error) {
	rows, err := s.db.Query(`SELECT DISTINCT `+campaignColumns+` FROM campaigns c
LEFT JOIN campaign_players cp ON cp.campaign_id = c.id
WHERE c.host_user_id = ? OR cp.user_id = ?
ORDER BY c.id DESC LIMIT 100`, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CampaignRow
	for rows.Next() {
		c, err := scanCampaign(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// SaveCampaignState persists the campaign state JSON plus status/index.
func (s *Store) SaveCampaignState(id int64, status string, index int, state string) error {
	_, err := s.db.Exec(`UPDATE campaigns SET status = ?, scenario_index = ?, state = ?, updated_at = ? WHERE id = ?`,
		status, index, state, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

// CampaignPlayers lists the seats of a campaign.
func (s *Store) CampaignPlayers(campaignID int64) ([]CampaignPlayer, error) {
	rows, err := s.db.Query(`SELECT slot, user_id, deck_id, hero_base FROM campaign_players WHERE campaign_id = ? ORDER BY slot`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CampaignPlayer
	for rows.Next() {
		var p CampaignPlayer
		if err := rows.Scan(&p.Slot, &p.UserID, &p.DeckID, &p.HeroBase); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// JoinCampaignSlot claims the first open seat with a deck.
func (s *Store) JoinCampaignSlot(campaignID int64, userID int64, deckID int64, heroBase string) (int, error) {
	players, err := s.CampaignPlayers(campaignID)
	if err != nil {
		return 0, err
	}
	row := s.db.QueryRow(`SELECT player_count FROM campaigns WHERE id = ?`, campaignID)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	claimed := map[int]bool{}
	for _, p := range players {
		if p.UserID != nil {
			claimed[p.Slot] = true
		}
	}
	slot := -1
	for i := 0; i < count; i++ {
		if !claimed[i] {
			slot = i
			break
		}
	}
	if slot < 0 {
		return 0, errors.New("campaign is full")
	}
	if _, err := s.db.Exec(`INSERT INTO campaign_players (campaign_id, slot, user_id, deck_id, hero_base) VALUES (?, ?, ?, ?, ?)`,
		campaignID, slot, userID, deckID, heroBase); err != nil {
		return 0, err
	}
	return slot, nil
}

// CampaignSlotByUser finds the seat a user claimed (-1 = none).
func (s *Store) CampaignSlotByUser(campaignID, userID int64) (int, error) {
	var slot int
	err := s.db.QueryRow(`SELECT slot FROM campaign_players WHERE campaign_id = ? AND user_id = ?`, campaignID, userID).Scan(&slot)
	if errors.Is(err, sql.ErrNoRows) {
		return -1, nil
	}
	return slot, err
}

// UpdateCampaignDeck re-seats a player's deck between chapters (the
// Watcher's Team rebuilds identities every chapter; several contest
// campaigns allow deck customization in the interlude).
func (s *Store) UpdateCampaignDeck(campaignID int64, slot int, deckID int64, heroBase string) error {
	res, err := s.db.Exec(`UPDATE campaign_players SET deck_id = ?, hero_base = ? WHERE campaign_id = ? AND slot = ?`,
		deckID, heroBase, campaignID, slot)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return errors.New("seat not found")
	}
	return nil
}

// SetGameCampaign links a game row to its campaign chapter.
func (s *Store) SetGameCampaign(gameID, campaignID int64) error {
	_, err := s.db.Exec(`UPDATE games SET campaign_id = ? WHERE id = ?`, campaignID, gameID)
	return err
}

// CampaignIDByGame reads the campaign link of a game (0 = none).
func (s *Store) CampaignIDByGame(gameID int64) (int64, error) {
	var id int64
	err := s.db.QueryRow(`SELECT campaign_id FROM games WHERE id = ?`, gameID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return id, err
}

// CampaignGames lists the chapter games of a campaign, newest first.
func (s *Store) CampaignGames(campaignID int64) ([]GameRow, error) {
	rows, err := s.db.Query(`SELECT `+gameColumns+` FROM games WHERE campaign_id = ? ORDER BY id DESC`, campaignID)
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

// KickCampaignSlot frees a seat (host action).
func (s *Store) KickCampaignSlot(campaignID int64, slot int) error {
	_, err := s.db.Exec(`DELETE FROM campaign_players WHERE campaign_id = ? AND slot = ?`, campaignID, slot)
	return err
}
