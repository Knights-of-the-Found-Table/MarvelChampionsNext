package silk_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/silk"
)

func silkDeck() map[string]int {
	return map[string]int{"52002": 2, "52003": 3, "52004": 2, "52005": 1, "52006": 1, "52007": 1, "52008": 1, "52009": 1, "52010": 1, "52011": 1, "52012": 1}
}

func TestSilkSenseTucksDefeatedCardsAndCapsAtFour(t *testing.T) {
	g, err := engine.NewGame(engine.NewGameOptions{Seed: 52, ScenarioID: "01097", Players: []engine.PlayerSpec{{Name: "Silk", HeroBase: "52001", Deck: silkDeck()}}})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	p := g.Players[0]
	b := engine.LookupBehavior("52001")
	if b == nil || b.React == nil {
		t.Fatal("Silk should expose Silk Sense React")
	}
	for i := 0; i < 5; i++ {
		id := g.NextEntityID("minion")
		mn := &engine.Minion{ID: id, Code: "01104", EngagedWith: p.ID}
		g.Minions[id] = mn
		b.React(g, p, engine.MinionDefeated{MinionID: id})
	}
	if len(p.SenseDeck) != 4 {
		t.Fatalf("tucked cards = %d, want cap 4", len(p.SenseDeck))
	}
	for _, c := range p.SenseDeck {
		if c.Code != "01104" {
			t.Fatalf("tucked code = %s, want defeated minion", c.Code)
		}
	}
}
