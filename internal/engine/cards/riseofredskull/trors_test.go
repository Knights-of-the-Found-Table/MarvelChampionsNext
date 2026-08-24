package riseofredskull_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/onceandfuturekang"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/riseofredskull"
)

func TestTrorsAllRegistered(t *testing.T) {
	for _, code := range []string{
		"04012", "04013", "04014", "04015", "04016", "04017", "04018",
		"04019", "04020", "04021", "04022", "04023", "04024", "04025",
		"04041", "04042", "04043", "04044", "04045", "04046", "04047",
		"04048", "04049", "04050", "04051", "04052", "04058", "04059",
		"04060", "04061", "04062", "04063", "04064", "04065", "04066",
		"04067", "04068", "04069", "04070", "04071", "04072", "04073",
		"04074", "04075", "04076", "04077", "04078", "04079", "04080",
		"04081", "04082", "04083", "04084", "04085", "04086", "04087",
		"04088", "04089", "04090", "04091", "04092", "04093", "04094",
		"04095", "04096", "04097", "04098", "04099", "04100", "04101",
		"04102", "04103", "04104", "04105", "04106", "04107", "04108",
		"04109", "04110", "04111", "04112", "04113", "04114", "04115",
		"04116", "04117", "04118", "04119", "04120", "04121", "04122",
		"04123", "04124", "04125", "04126", "04127", "04128", "04129",
		"04130", "04131", "04132", "04133", "04134", "04135", "04136",
		"04137", "04138", "04139", "04140", "04141", "04142", "04143",
		"04144", "04145", "04146", "04147", "04148", "04149", "04150",
		"04151", "04152", "04153", "04154", "04155", "04156", "04157",
		"04158", "04159", "04160", "04161", "04162", "04163", "04164",
		"04165", "04166", "10098",
	} {
		if !engine.Implemented(code) {
			t.Errorf("card %s has no registered behavior", code)
		}
	}
}

func newTrorsGame(t *testing.T, seed int64, scenario string) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: scenario,
		Players: []engine.PlayerSpec{
			{Name: "Sam", HeroBase: "08001", Deck: map[string]int{
				"08010": 2, "08025": 3, "08026": 3, "08027": 3,
			}},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	for i := 0; i < 8; i++ {
		pq := g.Pending()
		if pq == nil || pq.Question.Prompt == "Your turn" {
			break
		}
		_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
	}
	return g
}

func unblock(t *testing.T, g *engine.Game, limit int) {
	t.Helper()
	for i := 0; i < limit; i++ {
		pq := g.Pending()
		if pq == nil || g.Over {
			return
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
}

// TestCrossbonesSetup: the experimental weapons pool is seeded.
func TestCrossbonesSetup(t *testing.T) {
	g := newTrorsGame(t, 51, "04061")
	if len(g.SetAside) != 4 {
		t.Fatalf("the experimental weapons deck should hold 4 weapons, got %d", len(g.SetAside))
	}
}

// TestAbsorbingManSetup: an environment starts in play.
func TestAbsorbingManSetup(t *testing.T) {
	g := newTrorsGame(t, 52, "04079")
	found := false
	for _, env := range g.Environments {
		if env != nil {
			found = true
		}
	}
	if !found {
		t.Fatal("an environment should start in play")
	}
}

// TestZolaSetup: Hydra Prison reveals and Bio-Servants engage.
func TestZolaSetup(t *testing.T) {
	g := newTrorsGame(t, 53, "04112")
	unblock(t, g, 4)
	servants := 0
	for _, m := range g.Minions {
		if m != nil && m.Code[:5] == "04114" {
			servants++
		}
	}
	if servants == 0 {
		t.Fatal("Ultimate Bio-Servants should engage the players")
	}
}

// TestRedSkullSetup: the side-scheme deck + Sleeper benched.
func TestRedSkullSetup(t *testing.T) {
	g := newTrorsGame(t, 54, "04128")
	bench := len(g.SetAside)
	if bench < 6 {
		t.Fatalf("5 side schemes + The Sleeper should be benched, got %d", bench)
	}
}

// TestKangTribute: the final Kang extorts on attack.
func TestKangTribute(t *testing.T) {
	b := engine.LookupBehavior("11006")
	if b == nil || b.React == nil {
		t.Fatal("Kang the Conqueror should react to his attacks")
	}
}
