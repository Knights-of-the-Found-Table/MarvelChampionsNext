package aoa_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/aoa"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
)

func bishopDeck() map[string]int {
	return map[string]int{
		"45002": 1, "45003": 1, "45004": 1, "45005": 1, "45006": 1,
		"45007": 2, "45008": 2, "45009": 2, "45010": 3,
	}
}

func newAOAGame(t *testing.T, hero string, deck map[string]int) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed: 81, ScenarioID: "01097",
		Players: []engine.PlayerSpec{{Name: "AOA Hero", HeroBase: hero, Deck: deck}},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return g
}

// Contract test: invoke React directly and inspect the exact mill/recovery
// messages. It does not enter the recurring player-turn menu.
func TestBishopEnergyAbsorptionRecoversResources(t *testing.T) {
	g := newAOAGame(t, "45001", bishopDeck())
	p := g.Players[0]
	p.Side = engine.SideHero
	p.Deck = engine.CardList{
		{ID: g.NextCardID(), Code: "45010", Owner: p.ID},
		{ID: g.NextCardID(), Code: "45007", Owner: p.ID},
		{ID: g.NextCardID(), Code: "45010", Owner: p.ID},
	}

	b := engine.LookupBehavior("45001")
	if b == nil || b.React == nil {
		t.Fatal("Bishop should expose Energy Absorption through React")
	}
	msgs := b.React(g, p, engine.WindowDefended{Defender: p.ID, DamageTaken: 3})
	if len(msgs) != 3 {
		t.Fatalf("Energy Absorption returned %d messages, want mill + 2 recoveries", len(msgs))
	}
	mill, ok := msgs[0].(engine.MillPlayerDeck)
	if !ok || mill.Player != p.ID || mill.N != 3 {
		t.Fatalf("first message = %#v, want MillPlayerDeck N=3", msgs[0])
	}
	for i, want := range []string{p.Deck[0].ID, p.Deck[2].ID} {
		got, ok := msgs[i+1].(engine.ReturnDiscardCard)
		if !ok || got.CardID != want {
			t.Fatalf("message %d = %#v, want resource recovery %s", i+1, msgs[i+1], want)
		}
	}
}
