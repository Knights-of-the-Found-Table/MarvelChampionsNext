package aos_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/aos"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
)

func newAOSScenarioGame(t *testing.T, seed int64, scenario string) *engine.Game {
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

// TestWidowWebSetup: Black Widow I in play with the three-stage ladder.
func TestWidowWebSetup(t *testing.T) {
	g := newAOSScenarioGame(t, 71, "50067a")
	var v *engine.Villain
	for _, vv := range g.Villains {
		v = vv
	}
	if v == nil || engine.BaseCodeOf(v.Code) != "50064" {
		t.Fatalf("Black Widow I should be in play, got %v", v)
	}
	engaged := false
	for _, mn := range g.Minions {
		if mn != nil {
			engaged = true
		}
	}
	if !engaged {
		t.Fatal("each player should start engaged with a minion")
	}
}

// TestBatrocSetup: Alert Level in play, a Rescued Captive joined.
func TestBatrocSetup(t *testing.T) {
	g := newAOSScenarioGame(t, 72, "50087a")
	if env := g.EnvironmentByCode("50090a"); env == nil {
		t.Fatal("Alert Level should start in play")
	}
	found := false
	for _, p := range g.Players {
		for _, aid := range p.Allies {
			if a := g.Allies[aid]; a != nil && a.Code == "50091" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("a Rescued Captive should join the players")
	}
}

// TestMODOKSetup: Holding Cells and an Adaptoid Upgrade environment in
// play, one Adaptoid engaged.
func TestMODOKSetup(t *testing.T) {
	g := newAOSScenarioGame(t, 73, "50104a")
	cells, upgrades, adaptoids := 0, 0, 0
	for _, env := range g.Environments {
		if env == nil {
			continue
		}
		switch {
		case env.Code >= "50105a" && env.Code <= "50108a":
			cells++
			if env.Counters != 2*len(g.Players) {
				t.Fatalf("Holding Cell should hold %d lock counters, got %d", 2*len(g.Players), env.Counters)
			}
		case env.Code >= "50109" && env.Code <= "50112":
			upgrades++
		}
	}
	for _, mn := range g.Minions {
		if mn != nil && mn.Code == "50113" {
			adaptoids++
		}
	}
	if cells != 4 {
		t.Fatalf("4 Holding Cells expected, got %d", cells)
	}
	if upgrades != 1 {
		t.Fatalf("exactly one Adaptoid Upgrade environment expected, got %d", upgrades)
	}
	if adaptoids != 1 {
		t.Fatalf("one Adaptoid should be engaged with the player, got %d", adaptoids)
	}
}

// TestZemoSetup: the executive board convenes with secrets; evidence is
// set aside.
func TestZemoSetup(t *testing.T) {
	g := newAOSScenarioGame(t, 74, "50167a")
	board := 0
	for _, env := range g.Environments {
		if env != nil && env.Code >= "50181a" && env.Code <= "50183a" {
			board++
			if env.Counters != 2 {
				t.Fatalf("board member should hold 2 secret counters, got %d", env.Counters)
			}
		}
	}
	if board != 3 {
		t.Fatalf("3 board members expected, got %d", board)
	}
	for _, c := range g.EncounterDeck {
		if c.Code >= "50185" && c.Code <= "50193" {
			t.Fatal("evidence cards must never enter the encounter deck")
		}
	}
}

// TestThunderboltsSetup: Justice, Like Lightning and one Elite join.
func TestThunderboltsSetup(t *testing.T) {
	g := newAOSScenarioGame(t, 75, "50130a")
	if env := g.EnvironmentByCode("50131a"); env == nil {
		t.Fatal("Justice, Like Lightning should start in play")
	}
	elite := false
	for _, mn := range g.Minions {
		if mn != nil && mn.EDef().HasTrait("thunderbolt") {
			elite = true
		}
	}
	if !elite {
		t.Fatal("one Elite Thunderbolt should join the manhunt")
	}
}

// TestAOSGapCardsRegistered: spot-check the newly registered behaviors.
func TestAOSGapCardsRegistered(t *testing.T) {
	for _, code := range []string{
		"50012", "50016", "50017", "50019", "50020", "50022", "50023", "50024",
		"50047", "50048", "50050", "50053", "50054", "50057",
		"50064", "50065", "50066", "50076", "50081", "50084",
		"50086a", "50090a", "50095", "50101",
		"50103a", "50113", "50121", "50127",
		"50129a", "50133", "50137", "50154",
		"50165a", "50170", "50173", "50184a", "50185",
	} {
		if !engine.Implemented(code) {
			t.Errorf("card %s should be implemented", code)
		}
	}
}
