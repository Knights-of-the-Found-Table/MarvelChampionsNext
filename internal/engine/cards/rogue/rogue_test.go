package rogue_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/rogue"
)

func rogueDeck() map[string]int {
	return map[string]int{
		"38002": 1, "38003": 1, "38004": 1, "38005": 1, "38006": 1,
		"38007": 1, "38008": 1, "38009": 1,
	}
}

func newRogueGame(t *testing.T, seed int64) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Rogue", HeroBase: "38001", Deck: rogueDeck()},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return g
}

// TestRogueImplemented: the identity is registered.
func TestRogueImplemented(t *testing.T) {
	if !engine.Implemented("38001a") {
		t.Fatal("Rogue identity should be implemented")
	}
}

// TestSetupSetsTouchedAside: HeroSetup moves Touched from the deck into
// the set-aside slot (modeled by the side deck). Contract test via the
// identity hook directly.
func TestSetupSetsTouchedAside(t *testing.T) {
	g := newRogueGame(t, 17)
	p := g.Players[0]

	b := engine.LookupBehavior("38001")
	if b == nil || b.HeroSetup == nil {
		t.Fatal("Rogue identity should expose HeroSetup")
	}
	b.HeroSetup(g, p)

	if len(p.SenseDeck) != 1 || p.SenseDeck[0].Code != "38002" {
		t.Fatalf("SenseDeck = %v, want Touched set aside", p.SenseDeck)
	}
	for _, c := range p.Deck {
		if c.Code == "38002" {
			t.Fatal("Touched should have left the deck")
		}
	}
}

// TestSkinContactAttachesTouched: the hero action brings Touched into
// play from the set-aside slot and offers every other character as an
// attach target. Contract test via HeroAbilities.
func TestSkinContactAttachesTouched(t *testing.T) {
	g := newRogueGame(t, 17)
	p := g.Players[0]
	p.Side = engine.SideHero

	b := engine.LookupBehavior("38001")
	if b == nil || b.HeroAbilities == nil {
		t.Fatal("Rogue identity should expose HeroAbilities")
	}
	abs := b.HeroAbilities(g, p)
	if len(abs) != 1 || !strings.Contains(abs[0].Label.Text, "Skin Contact") {
		t.Fatalf("HeroAbilities = %v, want Skin Contact", abs)
	}
	if !abs[0].HeroOnly || !abs[0].OncePerRound {
		t.Fatal("Skin Contact should be HeroOnly and OncePerRound")
	}

	msgs := abs[0].Execute(g, p.ID)
	if len(msgs) != 1 {
		t.Fatalf("Skin Contact returned %d messages, want 1", len(msgs))
	}
	ask, ok := msgs[0].(engine.AskQuestion)
	if !ok || ask.Question == nil {
		t.Fatalf("Skin Contact message = %#v, want AskQuestion", msgs[0])
	}
	// The scenario villain is in play: at least one attach target.
	if len(ask.Question.Choices) == 0 {
		t.Fatal("Skin Contact offered no attach targets")
	}
	// Touched materialized in play; each choice attaches it.
	if len(p.Upgrades) != 1 {
		t.Fatalf("Touched not in play; upgrades = %v", p.Upgrades)
	}
	u := g.Upgrades[p.Upgrades[0]]
	if u == nil || u.Code != "38002" {
		t.Fatalf("upgrade = %#v, want Touched", u)
	}
	// Every choice targets a character and carries its attach payload
	// (choice msgs are engine-internal; structural contract only).
	for i, ch := range ask.Question.Choices {
		if ch.Kind != engine.ChoiceTarget || ch.SourceID == "" {
			t.Fatalf("choice %d = %#v, want a target choice", i, ch)
		}
	}
}

// TestDeadlyTouchObligation: with Touched not on a friendly character,
// the obligation places 2 threat on the main scheme and discards.
func TestDeadlyTouchObligation(t *testing.T) {
	g := newRogueGame(t, 17)
	p := g.Players[0]

	b := engine.LookupBehavior("38024")
	if b == nil || b.ResolveObligation == nil {
		t.Fatal("Deadly Touch should expose ResolveObligation")
	}
	card := engine.Card{ID: g.NextCardID(), Code: "38024", Owner: p.ID}
	msgs := b.ResolveObligation(g, p, card)
	if len(msgs) != 2 {
		t.Fatalf("Deadly Touch returned %d messages, want 2", len(msgs))
	}
	threat, ok := msgs[0].(engine.SchemeThreat)
	if !ok || threat.N != 2 || threat.Scheme != g.MainScheme.ID {
		t.Fatalf("Deadly Touch first message = %#v, want 2 threat on the main scheme", msgs[0])
	}
	if _, ok := msgs[1].(engine.ObligationResolve); !ok {
		t.Fatalf("Deadly Touch second message = %#v, want ObligationResolve", msgs[1])
	}
}

// TestRemainingrogueRegistered sweeps the pack's remaining cards.
func TestRemainingrogueSweep(t *testing.T) {
	for _, def := range engine.DB.All() {
		if def.PackCode != "rogue" {
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
