package magneto_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/magneto"
)

func magnetoDeck() map[string]int {
	return map[string]int{
		"49002": 1, "49003": 1, "49004": 1, "49005": 1, "49006": 1,
		"49007": 2, "49008": 2, "49009": 2, "49010": 2, "49011": 2,
	}
}

// Contract test: execute Magnetic Pull directly and verify it mills through
// the first Magnetic card, then returns exactly that card from discard.
func TestMagnetoMagneticPull(t *testing.T) {
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed: 84, ScenarioID: "01097",
		Players: []engine.PlayerSpec{{Name: "Magneto", HeroBase: "49001", Deck: magnetoDeck()}},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	p := g.Players[0]
	p.Side = engine.SideHero
	magnetic := engine.Card{ID: g.NextCardID(), Code: "49008", Owner: p.ID}
	p.Deck = engine.CardList{
		{ID: g.NextCardID(), Code: "49024", Owner: p.ID},
		{ID: g.NextCardID(), Code: "49021", Owner: p.ID},
		magnetic,
		{ID: g.NextCardID(), Code: "49009", Owner: p.ID},
	}

	b := engine.LookupBehavior("49001")
	if b == nil || b.HeroAbilities == nil {
		t.Fatal("Magneto should expose Magnetic Pull")
	}
	abilities := b.HeroAbilities(g, p)
	if len(abilities) != 1 || abilities[0].Execute == nil || !abilities[0].OncePerRound {
		t.Fatalf("Magnetic Pull ability contract is incomplete: %#v", abilities)
	}
	msgs := abilities[0].Execute(g, p.ID)
	if len(msgs) != 2 {
		t.Fatalf("Magnetic Pull returned %d messages, want mill + recovery", len(msgs))
	}
	mill, ok := msgs[0].(engine.MillPlayerDeck)
	if !ok || mill.N != 3 || mill.Player != p.ID {
		t.Fatalf("first message = %#v, want MillPlayerDeck N=3", msgs[0])
	}
	back, ok := msgs[1].(engine.ReturnDiscardCard)
	if !ok || back.CardID != magnetic.ID || back.Player != p.ID {
		t.Fatalf("second message = %#v, want recovery of first Magnetic card", msgs[1])
	}
}
