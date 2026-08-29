package rooms

import (
	"fmt"
	"sort"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// PileCard is one entry of a pile listing.
type PileCard struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// PileList returns the contents of a pile for display. Deck listings are
// sorted by card code: the sort scrambles the stored order, so the client
// cannot infer draw order, and the listing stays stable across refreshes.
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
		// Empty player id = the shared encounter discard pile.
		if playerID == "" {
			list = r.game.EncounterDiscard
			break
		}
		p := r.game.Player(engine.PlayerID(playerID))
		if p == nil {
			return nil, fmt.Errorf("unknown player %q", playerID)
		}
		list = p.Discard
	case "sideDeck":
		p := r.game.Player(engine.PlayerID(playerID))
		if p == nil {
			return nil, fmt.Errorf("unknown player %q", playerID)
		}
		list = p.SenseDeck
	case "sideDiscard":
		p := r.game.Player(engine.PlayerID(playerID))
		if p == nil {
			return nil, fmt.Errorf("unknown player %q", playerID)
		}
		list = p.SideDiscard
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
	if pile == "deck" || pile == "sideDeck" {
		// 字典序打乱存储顺序且不泄露抽牌次序，比洗牌更稳定可读。
		sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	} else if pile == "discard" || pile == "sideDiscard" {
		for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
			out[i], out[j] = out[j], out[i]
		}
	}
	return out, nil
}
