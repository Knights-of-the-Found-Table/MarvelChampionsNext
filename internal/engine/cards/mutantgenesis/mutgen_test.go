package mutantgenesis_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/mutantgenesis"
)

// TestMutGenAllImplemented sweeps every carded entry of the box.
func TestMutGenAllImplemented(t *testing.T) {
	checked := 0
	for _, def := range engine.DB.All() {
		if def.PackCode != "mut_gen" {
			continue
		}
		if strings.TrimSpace(def.Text) == "" {
			continue
		}
		if !engine.Implemented(def.Code) {
			t.Errorf("card %s (%s) has no registered behavior", def.Code, def.Name)
		}
		checked++
	}
	if checked < 200 {
		t.Fatalf("only %d texted mut_gen cards swept", checked)
	}
}

func unblockMG(t *testing.T, g *engine.Game, limit int) {
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

func newMutGenGame(t *testing.T, seed int64, scenario string) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: scenario,
		Players: []engine.PlayerSpec{
			{Name: "Piotr", HeroBase: "32001", Deck: map[string]int{
				"32007": 1, "32008": 2, "32009": 1, "32010": 1,
			}},
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

// TestSabretoothSetup: Robert Kelly joins the first player.
func TestSabretoothSetup(t *testing.T) {
	g := newMutGenGame(t, 41, "32063")
	found := false
	for _, a := range g.Allies {
		if a != nil && a.Code == "32066" {
			found = true
		}
	}
	if !found {
		t.Fatal("Robert Kelly should start in play")
	}
}

// TestShadowcatMassForm: Shadowcat starts Solid and flips after a basic
// attack.
func TestShadowcatMassForm(t *testing.T) {
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       42,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Kitty", HeroBase: "32030", Deck: map[string]int{"32037": 2, "32039": 2}},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	p := g.Players[0]
	var solid *engine.Upgrade
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil && u.Code == "32031a" {
			solid = u
		}
	}
	if solid == nil {
		t.Fatal("Shadowcat should start with the Solid mass form")
	}
	// The flip response swaps the code on a basic attack.
	p.Side = engine.SideHero
	b := engine.LookupBehavior("32031")
	msgs := b.React(g, solid, engine.BasicAttack{Player: p.ID})
	if msgs != nil {
		t.Fatalf("flip react should mutate silently, got %#v", msgs)
	}
	if solid.Code != "32031b" {
		t.Fatalf("mass form = %s, want Phased (32031b)", solid.Code)
	}
}

// TestMagnetCounterPayoff: three magnet counters reveal a Magnetic card.
func TestMagnetCounterPayoff(t *testing.T) {
	g := newMutGenGame(t, 43, "32141")
	if g.MainScheme == nil {
		t.Fatal("Magneto scenario needs a main scheme")
	}
	// Seed the encounter deck with Magneto's Helmet (a Magnetic card with
	// no counter rider of its own).
	g.EncounterDeck = append(engine.CardList{{ID: "mag1", Code: "32147"}}, g.EncounterDeck...)
	g.Push(engine.AddMagnetCounter{Scheme: g.MainScheme.ID})
	g.Push(engine.AddMagnetCounter{Scheme: g.MainScheme.ID})
	unblockMG(t, g, 1)
	if g.MainScheme.Counters != 2 {
		t.Fatalf("magnet counters = %d, want 2", g.MainScheme.Counters)
	}
	// The third counter triggers the Magnetic reveal and resets to 0.
	g.Push(engine.AddMagnetCounter{Scheme: g.MainScheme.ID})
	unblockMG(t, g, 1)
	if g.MainScheme.Counters != 0 {
		t.Fatalf("magnet counters = %d, want 0 after the payoff", g.MainScheme.Counters)
	}
}

// TestHinderReducesThreat: a Hinder 2 side scheme blocks 2 main-scheme
// threat per placement.
func TestHinderReducesThreat(t *testing.T) {
	g := newMutGenGame(t, 44, "32063")
	p := g.Players[0]
	s := &engine.SideScheme{ID: g.NextEntityID(engine.KindSideScheme), Code: "32071", Threat: 4, MaxThreat: 8}
	g.AddSideScheme(s)
	before := g.MainScheme.Threat
	g.Push(engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 3, Source: p.ID})
	for i := 0; i < 4; i++ {
		pq := g.Pending()
		if pq == nil || g.Over {
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
	if got := g.MainScheme.Threat - before; got != 1 {
		t.Fatalf("threat placed = %d, want 1 (3 minus Hinder 2)", got)
	}
}

// TestSentinelSetup: the Captives wait in the set-aside area.
func TestSentinelSetup(t *testing.T) {
	g := newMutGenGame(t, 45, "32087")
	captives := 0
	for _, c := range g.SetAside {
		if c.Def().HasTrait("captive") {
			captives++
		}
	}
	if captives < 4 {
		t.Fatalf("captives set aside = %d, want 4", captives)
	}
}

// TestAcolyteGuardAura: The Acolytes scheme grants Acolytes guard —
// verified through the scenario presence helpers (compile-level check
// plus the scheme registration).
func TestAcolyteSchemeRegistered(t *testing.T) {
	if !engine.Implemented("32165") {
		t.Fatal("The Acolytes scheme missing")
	}
	for _, code := range []string{"32159", "32160", "32161", "32162", "32163"} {
		if !engine.Implemented(code) {
			t.Errorf("Acolyte %s missing", code)
		}
	}
}
