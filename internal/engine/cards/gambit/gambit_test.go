package gambit_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/gambit"
)

func gambitDeck() map[string]int {
	return map[string]int{
		"37002": 1, "37003": 1, "37004": 1, "37005": 1, "37006": 1,
		"37007": 1, "37008": 1, "37009": 1, "37010": 1,
	}
}

func newGambitGame(t *testing.T, seed int64) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Gambit", HeroBase: "37001", Deck: gambitDeck()},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return g
}

// TestGambitImplemented: the identity is registered.
func TestGambitImplemented(t *testing.T) {
	if !engine.Implemented("37001a") {
		t.Fatal("Gambit identity should be implemented")
	}
}

// TestChargeDeCard: the hero-side action places 1 charge counter on the
// identity and is limited to once per round. Contract test via
// HeroAbilities, no full game walk.
func TestChargeDeCard(t *testing.T) {
	g := newGambitGame(t, 11)
	p := g.Players[0]
	p.Side = engine.SideHero

	b := engine.LookupBehavior("37001")
	if b == nil || b.HeroAbilities == nil {
		t.Fatal("Gambit identity should expose HeroAbilities")
	}
	var charge *engine.Ability
	for i := range b.HeroAbilities(g, p) {
		abs := b.HeroAbilities(g, p)
		if strings.Contains(abs[i].Label, "Charge de Card") {
			charge = &abs[i]
			break
		}
	}
	if charge == nil {
		t.Fatal("Charge de Card ability should be offered")
	}
	if !charge.HeroOnly || !charge.OncePerRound {
		t.Fatal("Charge de Card should be HeroOnly and OncePerRound")
	}
	msgs := charge.Execute(g, p.ID)
	if len(msgs) != 1 {
		t.Fatalf("Charge de Card returned %d messages, want 1", len(msgs))
	}
	add, ok := msgs[0].(engine.AddEntityCounter)
	if !ok || add.ID != p.ID || add.N != 1 {
		t.Fatalf("Charge de Card message = %#v, want +1 counter on the identity", msgs[0])
	}
}

// TestThrowDeCard: playing an attack event offers to remove up to 3
// charge counters for +1 damage each; non-attack events and an empty
// counter pool stay silent. Contract test via React.
func TestThrowDeCard(t *testing.T) {
	g := newGambitGame(t, 11)
	p := g.Players[0]
	p.Side = engine.SideHero

	b := engine.LookupBehavior("37001")
	if b == nil || b.React == nil {
		t.Fatal("Gambit identity should expose React")
	}

	attackEvent := engine.EventPlayed{Player: p.ID, Card: engine.Card{Code: "37006", Owner: p.ID}}
	thwartEvent := engine.EventPlayed{Player: p.ID, Card: engine.Card{Code: "37009", Owner: p.ID}}

	// No counters: silent.
	if msgs := b.React(g, p, attackEvent); len(msgs) != 0 {
		t.Fatalf("Throw de Card with 0 counters returned %d messages, want 0", len(msgs))
	}
	// Counters available: an attack event offers the boost question.
	p.Counters = 5
	msgs := b.React(g, p, attackEvent)
	if len(msgs) != 1 {
		t.Fatalf("Throw de Card returned %d messages, want 1", len(msgs))
	}
	ask, ok := msgs[0].(engine.AskQuestion)
	if !ok || ask.Question == nil {
		t.Fatalf("Throw de Card message = %#v, want AskQuestion", msgs[0])
	}
	// keep + remove 1..3 (capped at 3 even with 5 counters).
	if len(ask.Question.Choices) != 4 {
		t.Fatalf("Throw de Card offered %d choices, want 4 (keep + 1..3)", len(ask.Question.Choices))
	}
	// A non-attack event does not trigger.
	if msgs := b.React(g, p, thwartEvent); len(msgs) != 0 {
		t.Fatalf("Throw de Card on a thwart event returned %d messages, want 0", len(msgs))
	}
	// Alter-ego form: inactive.
	p.Side = engine.SideAlterEgo
	if msgs := b.React(g, p, attackEvent); len(msgs) != 0 {
		t.Fatalf("Throw de Card in alter-ego returned %d messages, want 0", len(msgs))
	}
}

// TestMolecularAccelerationRider: spending a Molecular Acceleration
// resource places 1 charge counter on the identity. Contract test via
// the identity's React on ResourcePay.
func TestMolecularAccelerationRider(t *testing.T) {
	g := newGambitGame(t, 11)
	p := g.Players[0]

	b := engine.LookupBehavior("37001")
	msgs := b.React(g, p, engine.ResourcePay{Player: p.ID, Cards: engine.CardList{
		{Code: "37010", Owner: p.ID},
		{Code: "37009", Owner: p.ID},
	}})
	if len(msgs) != 1 {
		t.Fatalf("Molecular Acceleration rider returned %d messages, want 1", len(msgs))
	}
	add, ok := msgs[0].(engine.AddEntityCounter)
	if !ok || add.ID != p.ID || add.N != 1 {
		t.Fatalf("rider message = %#v, want +1 counter on the identity", msgs[0])
	}
	// A payment without 37010 stays silent.
	if msgs := b.React(g, p, engine.ResourcePay{Player: p.ID, Cards: engine.CardList{{Code: "37009", Owner: p.ID}}}); len(msgs) != 0 {
		t.Fatalf("rider without Molecular Acceleration returned %d messages, want 0", len(msgs))
	}
}

// TestRemaininggambitRegistered sweeps the pack's remaining cards.
func TestRemaininggambitSweep(t *testing.T) {
	for _, def := range engine.DB.All() {
		if def.PackCode != "gambit" {
			continue
		}
		if def.Text == "" {
			continue
		}
		if !engine.Implemented(def.Code) {
			t.Errorf("card %s (%s) has no registered behavior", def.Code, def.Name)
		}
	}
}
