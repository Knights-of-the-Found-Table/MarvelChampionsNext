package x23_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/x23"
)

func x23Deck() map[string]int {
	return map[string]int{
		"43002": 1, "43003": 1, "43004": 1, "43005": 1, "43006": 1,
		"43007": 1, "43008": 1, "43009": 1, "43010": 1, "43011": 1,
		"43012": 1,
	}
}

func newX23Game(t *testing.T, seed int64) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "X-23", HeroBase: "43001", Deck: x23Deck()},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return g
}

// TestX23Implemented: the identity is registered.
func TestX23Implemented(t *testing.T) {
	if !engine.Implemented("43001a") {
		t.Fatal("X-23 identity should be implemented")
	}
}

// TestLivingWeapon: taking damage readies an exhausted X-23, once per
// phase. Contract test via the identity React hook.
func TestLivingWeapon(t *testing.T) {
	g := newX23Game(t, 23)
	p := g.Players[0]
	p.Side = engine.SideHero
	p.Exhausted = true

	b := engine.LookupBehavior("43001")
	if b == nil || b.React == nil {
		t.Fatal("X-23 identity should expose React")
	}
	msgs := b.React(g, p, engine.DamageEntity{Target: p.ID, Damage: 2, Source: "villain-1"})
	if len(msgs) != 1 {
		t.Fatalf("Living Weapon returned %d messages, want 1", len(msgs))
	}
	if rd, ok := msgs[0].(engine.ReadyEntity); !ok || rd.ID != p.ID {
		t.Fatalf("Living Weapon message = %#v, want ReadyEntity on X-23", msgs[0])
	}
	// Once per phase: a second damage in the same phase stays silent.
	if msgs := b.React(g, p, engine.DamageEntity{Target: p.ID, Damage: 1, Source: "villain-1"}); len(msgs) != 0 {
		t.Fatalf("Living Weapon twice in a phase returned %d messages, want 0", len(msgs))
	}
	// A ready identity does not need the ready.
	g.UsedThisTurn = map[string]bool{}
	p.Exhausted = false
	if msgs := b.React(g, p, engine.DamageEntity{Target: p.ID, Damage: 1, Source: "villain-1"}); len(msgs) != 0 {
		t.Fatalf("Living Weapon while ready returned %d messages, want 0", len(msgs))
	}
}

// TestSetupPlaysClaws: Shhnk! puts X-23's Claws into play from the
// deck. Contract test via HeroSetup.
func TestSetupPlaysClaws(t *testing.T) {
	g := newX23Game(t, 23)
	p := g.Players[0]
	p.Deck = append(p.Deck, engine.Card{ID: g.NextCardID(), Code: "43002", Owner: p.ID})

	b := engine.LookupBehavior("43001")
	if b == nil || b.HeroSetup == nil {
		t.Fatal("X-23 identity should expose HeroSetup")
	}
	msgs := b.HeroSetup(g, p)
	if len(msgs) != 1 {
		t.Fatalf("Shhnk! returned %d messages, want 1", len(msgs))
	}
	if up, ok := msgs[0].(engine.UpgradeEnterPlay); !ok || up.Card.Code != "43002" {
		t.Fatalf("Shhnk! message = %#v, want UpgradeEnterPlay for X-23's Claws", msgs[0])
	}
}

// TestLauraKinneyRecursion: the alter-ego action shuffles Honey Badger
// or Sisterly Bond from the discard into the deck and draws 1.
func TestLauraKinneyRecursion(t *testing.T) {
	g := newX23Game(t, 23)
	p := g.Players[0]
	p.Side = engine.SideAlterEgo
	p.Discard = append(p.Discard, engine.Card{ID: g.NextCardID(), Code: "43003", Owner: p.ID})

	b := engine.LookupBehavior("43001")
	if b == nil || b.HeroAbilities == nil {
		t.Fatal("X-23 identity should expose HeroAbilities")
	}
	abs := b.HeroAbilities(g, p)
	if len(abs) != 1 {
		t.Fatalf("HeroAbilities = %d, want 1", len(abs))
	}
	ab := abs[0]
	if !ab.AlterEgoOnly || !ab.OncePerRound {
		t.Fatal("Laura's action should be AlterEgoOnly and OncePerRound")
	}
	msgs := ab.Execute(g, p.ID)
	if len(msgs) != 1 {
		t.Fatalf("Laura's action returned %d messages, want 1", len(msgs))
	}
	ask, ok := msgs[0].(engine.AskQuestion)
	if !ok || ask.Question == nil || len(ask.Question.Choices) != 1 {
		t.Fatalf("Laura's action message = %#v, want a 1-choice question", msgs[0])
	}
	if !strings.Contains(ask.Question.Choices[0].Label.Text, "Honey Badger") {
		t.Fatalf("choice label = %q, want Honey Badger", ask.Question.Choices[0].Label.Text)
	}
}
