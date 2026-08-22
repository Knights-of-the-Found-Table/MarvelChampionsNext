package hood_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/hood"
)

// TestHoodScenarioSetup: the scenario registers, folds one modular set
// in, and starts with The Hood I.
func TestHoodScenarioSetup(t *testing.T) {
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       21,
		ScenarioID: "24004",
		Players: []engine.PlayerSpec{
			{Name: "P1", HeroBase: "01001", Deck: map[string]int{"01088": 9, "01089": 9}},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	if len(g.Villains) != 1 {
		t.Fatalf("expected one villain, got %d", len(g.Villains))
	}
	for _, v := range g.Villains {
		if v.Code[:5] != "24001" {
			t.Fatalf("expected The Hood I, got %s", v.Code)
		}
	}
	if len(g.EncounterDeck) < 20 {
		t.Fatalf("encounter deck should include a folded modular set, got %d cards", len(g.EncounterDeck))
	}
	if pq := g.Pending(); pq != nil && pq.Question.Prompt == "Mulligan?" {
		if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
			t.Fatalf("keep: %v", err)
		}
	}
	if pq := g.Pending(); pq == nil || pq.Question.Prompt != "Your turn" {
		t.Fatal("game should reach the first turn menu")
	}
}

// TestFoulPlayDealsFacedown: non-Hood cards are dealt facedown.
func TestFoulPlayDealsFacedown(t *testing.T) {
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       22,
		ScenarioID: "24004",
		Players: []engine.PlayerSpec{
			{Name: "P1", HeroBase: "01001", Deck: map[string]int{"01088": 9, "01089": 9}},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	p := g.Players[0]
	if pq := g.Pending(); pq != nil && pq.Question.Prompt == "Mulligan?" {
		_ = g.Answer(pq.Player, []string{"keep"})
	}
	// The deck top is non-Hood (modular/standard cards dominate).
	before := len(p.EncounterDown)
	g.Push(engine.HoodFoulPlay{Player: p.ID, N: 3})
	if pq := g.Pending(); pq != nil {
		for _, c := range pq.Question.Choices {
			if c.Then == nil && !c.Disabled {
				_ = g.Answer(pq.Player, []string{c.ID})
				break
			}
		}
	}
	g.Run()
	if len(p.EncounterDown) <= before {
		t.Fatalf("Foul Play should deal facedown cards, %d -> %d", before, len(p.EncounterDown))
	}
}
