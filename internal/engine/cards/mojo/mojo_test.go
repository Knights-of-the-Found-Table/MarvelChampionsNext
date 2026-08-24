package mojo_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/mojo"
)

// TestMojoAllRegistered sweeps every texted card of the box.
func TestMojoAllRegistered(t *testing.T) {
	checked := 0
	for _, def := range engine.DB.All() {
		if def.PackCode != "mojo" {
			continue
		}
		if def.Text == "" {
			continue
		}
		if !engine.Implemented(def.Code) {
			t.Errorf("card %s (%s) has no registered behavior", def.Code, def.Name)
		}
		checked++
	}
	if checked < 70 {
		t.Fatalf("only %d texted mojo cards swept", checked)
	}
}

func newMojoGame(t *testing.T, seed int64, scenario string) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: scenario,
		Players: []engine.PlayerSpec{
			{Name: "P1", HeroBase: "01001", Deck: map[string]int{"01088": 9, "01089": 9}},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	for i := 0; i < 10; i++ {
		pq := g.Pending()
		if pq == nil || pq.Question.Prompt == "Your turn" {
			break
		}
		idx := 0
		for j, c := range pq.Question.Choices {
			if c.Then == nil && !c.Disabled {
				idx = j
				break
			}
		}
		_ = g.Answer(pq.Player, []string{pq.Question.Choices[idx].ID})
	}
	return g
}

// TestMaGogSetup: the crowd boards enter play.
func TestMaGogSetup(t *testing.T) {
	g := newMojoGame(t, 51, "39002")
	if envByCodePub(g, "39003") == nil || envByCodePub(g, "39004") == nil {
		t.Fatal("The Champion and The Challengers should start in play")
	}
}

// TestSpiralSetup: The Search for Spiral starts revealed.
func TestSpiralSetup(t *testing.T) {
	g := newMojoGame(t, 52, "39015")
	found := false
	for _, s := range g.SideSchemes {
		if s != nil && s.Code == "39016" {
			found = true
		}
	}
	if !found {
		t.Fatal("The Search for Spiral should start in play")
	}
}

// TestMojoSetup: the Wheel of Genres spins up.
func TestMojoSetup(t *testing.T) {
	g := newMojoGame(t, 53, "39025")
	found := false
	for _, e := range g.Environments {
		if e != nil && engine.BaseCodeOf(e.Code) == "39026" {
			found = true
		}
	}
	if !found {
		t.Fatal("the Wheel of Genres should start in play")
	}
}

func envByCodePub(g *engine.Game, base string) *engine.Environment {
	for _, e := range g.Environments {
		if e != nil && engine.BaseCodeOf(e.Code) == base {
			return e
		}
	}
	return nil
}
