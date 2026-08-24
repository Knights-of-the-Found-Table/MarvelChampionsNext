package civilwar_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/civilwar"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
)

func newCWGame(t *testing.T, seed int64, scenario string) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: scenario,
		Players:    []engine.PlayerSpec{{Name: "P1", HeroBase: "01001", Deck: map[string]int{"01088": 9, "01089": 9}}},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	for i := 0; i < 12; i++ {
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

// TestRegistrationActSetup: the villain-less scheme loads.
func TestRegistrationActSetup(t *testing.T) {
	g := newCWGame(t, 81, "56063")
	if g.MainScheme == nil {
		t.Fatal("the Registration Act main scheme should be in play")
	}
	if len(g.Villains) != 0 {
		t.Fatalf("Registration Act is villain-less, got %d villains", len(g.Villains))
	}
}

// TestLeaderScenarioSetup: Captain Marvel leads with Energy Channel.
func TestLeaderScenarioSetup(t *testing.T) {
	g := newCWGame(t, 82, "56096b")
	var v *engine.Villain
	for _, vv := range g.Villains {
		v = vv
	}
	if v == nil || engine.BaseCodeOf(v.Code) != "56092" {
		t.Fatalf("Captain Marvel should lead, got %v", v)
	}
	found := false
	for _, a := range g.Attachments {
		if a != nil && a.Code == "56098" {
			found = true
		}
	}
	if !found {
		t.Fatal("Energy Channel should start attached to Captain Marvel")
	}
}

// TestResistanceLeaderSetup: Cap's Shield attached.
func TestResistanceLeaderSetup(t *testing.T) {
	g := newCWGame(t, 83, "56141b")
	var v *engine.Villain
	for _, vv := range g.Villains {
		v = vv
	}
	if v == nil || engine.BaseCodeOf(v.Code) != "56137" {
		t.Fatalf("Captain America should lead, got %v", v)
	}
	found := false
	for _, a := range g.Attachments {
		if a != nil && a.Code == "56143" {
			found = true
		}
	}
	if !found {
		t.Fatal("Cap's Shield should start attached to Captain America")
	}
}

// TestCWGapCardsRegistered: spot-check the new registrations.
func TestCWGapCardsRegistered(t *testing.T) {
	for _, code := range []string{
		"56002", "56003", "56008", "56009", "56015", "56016", "56019", "56037", "56038",
		"56043", "56044", "56047", "56048", "56053", "56056", "56058",
		"56059", "56060", "56070", "56076", "56083", "56087", "56098", "56101",
		"56106", "56110", "56115", "56120", "56125", "56128", "56143", "56158",
		"56163", "56166", "56174", "56177", "56186", "56197",
		"56063", "56064", "56096", "56097", "56121", "56124", "56141", "56202",
	} {
		if !engine.Implemented(code) {
			t.Errorf("card %s should be implemented", code)
		}
	}
}
