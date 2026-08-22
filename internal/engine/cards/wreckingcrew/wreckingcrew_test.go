package wreckingcrew_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register core + wrecking crew content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/wreckingcrew"
)

// TestWreckingCrewSetup: four villains spawn, each with a personal side
// scheme, and the active counter starts on Wrecker.
func TestWreckingCrewSetup(t *testing.T) {
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       7,
		ScenarioID: "07001",
		Players: []engine.PlayerSpec{
			{Name: "P1", HeroBase: "01001", Deck: map[string]int{"01088": 9, "01089": 9}},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	if len(g.Villains) != 4 {
		t.Fatalf("expected 4 villains, got %d", len(g.Villains))
	}
	if len(g.SideSchemes) != 4 {
		t.Fatalf("expected 4 personal side schemes, got %d", len(g.SideSchemes))
	}
	if g.ActiveVillain == "" || g.Villains[g.ActiveVillain] == nil {
		t.Fatal("active counter should start on Wrecker")
	}
	if base := g.Villains[g.ActiveVillain].Code[:5]; base != "07002" {
		t.Fatalf("active counter should start on Wrecker (07002), got %s", base)
	}
	// Keep the mulligan and confirm the game is answerable.
	if pq := g.Pending(); pq != nil {
		if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
			t.Fatalf("keep: %v", err)
		}
	}
	if pq := g.Pending(); pq == nil || pq.Question.Prompt != "Your turn" {
		t.Fatal("game should reach the first turn menu")
	}
}

// TestWreckingCrewWinRequiresAllFour: with three villains removed, the
// scenario override does not end the game; removing the last wins.
func TestWreckingCrewWinRequiresAllFour(t *testing.T) {
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       8,
		ScenarioID: "07001",
		Players: []engine.PlayerSpec{
			{Name: "P1", HeroBase: "01001", Deck: map[string]int{"01088": 9, "01089": 9}},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	// Remove three villains directly.
	i := 0
	for id := range g.Villains {
		if i < 3 {
			delete(g.Villains, id)
			i++
		}
	}
	var last engine.EntityID
	for id := range g.Villains {
		last = id
	}
	// Keep the mulligan so the turn menu is pending, then queue lethal
	// damage and unblock with the harmless form choice.
	if pq := g.Pending(); pq != nil && pq.Question.Prompt == "Mulligan?" {
		if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
			t.Fatalf("keep: %v", err)
		}
	}
	g.Push(engine.DamageEntity{Target: last, Damage: 999, Source: g.Players[0].ID})
	if pq := g.Pending(); pq != nil {
		if err := g.Answer(pq.Player, []string{"form"}); err != nil {
			t.Fatalf("unblock: %v", err)
		}
	}
	g.Run()
	if !g.Over || !g.Won {
		t.Fatalf("defeating the last villain should win, over=%v won=%v", g.Over, g.Won)
	}
}
