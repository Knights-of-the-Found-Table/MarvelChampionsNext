package rooms

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// PileCard is one entry of a pile listing.
type PileCard struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// PileList returns the contents of a pile for display. Deck listings are
// shuffled with a wall-clock RNG (never the game's seeded PCG, which must
// stay untouched for determinism), so the client cannot infer draw order.
// Discard listings are returned top-card first.
func (m *Manager) PileList(gameID int64, playerID, pile string) ([]PileCard, error) {
	r, err := m.Get(gameID)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	var list engine.CardList
	switch pile {
	case "deck":
		if playerID == "" {
			list = r.game.EncounterDeck
		} else {
			p := r.game.Player(engine.PlayerID(playerID))
			if p == nil {
				return nil, fmt.Errorf("unknown player %q", playerID)
			}
			list = p.Deck
		}
	case "discard":
		p := r.game.Player(engine.PlayerID(playerID))
		if p == nil {
			return nil, fmt.Errorf("unknown player %q", playerID)
		}
		list = p.Discard
	default:
		return nil, fmt.Errorf("unknown pile %q", pile)
	}

	out := make([]PileCard, len(list))
	for i, c := range list {
		name := c.Code
		if def, ok := engine.DB.Lookup(c.Code); ok && def.Name != "" {
			name = def.Name
		}
		out[i] = PileCard{Code: c.Code, Name: name}
	}
	if pile == "deck" && len(out) > 1 {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	} else if pile == "discard" {
		for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
			out[i], out[j] = out[j], out[i]
		}
	}
	return out, nil
}
