package blackpanther_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/blackpanther"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
)

func bpDeck() map[string]int {
	return map[string]int{
		"51002": 1, "51003": 1, "51004": 1, "51005": 1, "51007": 1,
		"51008": 1, "51009": 1, "51010": 1, "51011": 1, "51012": 1,
	}
}

func newBPGame(t *testing.T, seed int64) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "BlackPanther", HeroBase: "51001", Deck: bpDeck()},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return g
}

// TestBlackPantherImplemented: the identity is registered.
func TestBlackPantherImplemented(t *testing.T) {
	if !engine.Implemented("51001a") {
		t.Fatal("Black Panther identity should be implemented")
	}
}

// TestBasicPowerResponseNeedsUpgrade: the hero-side response (resolve a
// Black Panther upgrade's Special after a basic power) stays silent when
// no Black Panther upgrade is in play. Contract test via React.
func TestBasicPowerResponseNeedsUpgrade(t *testing.T) {
	g := newBPGame(t, 13)
	p := g.Players[0]
	p.Side = engine.SideHero

	b := engine.LookupBehavior("51001")
	if b == nil || b.React == nil {
		t.Fatal("Black Panther identity should expose React")
	}
	// No BP upgrades in play: a basic attack produces no question.
	msgs := b.React(g, p, engine.BasicAttack{Player: p.ID, Target: engine.EntityID("test-target"), N: 1})
	if len(msgs) != 0 {
		t.Fatalf("React without a BP upgrade returned %d messages, want 0", len(msgs))
	}
	// Alter-ego form: inactive.
	p.Side = engine.SideAlterEgo
	if msgs := b.React(g, p, engine.BasicThwart{Player: p.ID}); len(msgs) != 0 {
		t.Fatalf("React in alter-ego form returned %d messages, want 0", len(msgs))
	}
}
