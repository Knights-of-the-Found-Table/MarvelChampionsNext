package spiderham_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/spiderham"
)

func hamDeck() map[string]int {
	return map[string]int{
		"30002": 1, "30003": 2, "30004": 1, "30005": 1, "30006": 2,
		"30007": 3, "30008": 1, "30009": 2, "30010": 1, "30011": 1,
	}
}

func newHamGame(t *testing.T) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed: 30, ScenarioID: "01097",
		Players: []engine.PlayerSpec{{Name: "Spider-Ham", HeroBase: "30001", Deck: hamDeck()}},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return g
}

// Contract test: Spider-Nonsense places a toon counter after Spider-Ham
// takes damage (hero form only), and Cartoon Power places one after Peter
// Porker recovers.
func TestToonCounterGain(t *testing.T) {
	g := newHamGame(t)
	p := g.Players[0]
	b := engine.LookupBehavior("30001")
	if b == nil || b.React == nil {
		t.Fatal("Spider-Ham should expose a React hook")
	}
	p.Side = engine.SideHero
	msgs := b.React(g, p, engine.DamageEntity{Target: p.ID, Damage: 2, Source: "villain"})
	if len(msgs) != 1 {
		t.Fatalf("damage react returned %d messages, want one AddEntityCounter", len(msgs))
	}
	if c, ok := msgs[0].(engine.AddEntityCounter); !ok || c.ID != p.ID || c.N != 1 {
		t.Fatalf("message = %#v, want +1 toon counter on the identity", msgs[0])
	}
	// No counter from zero-damage hits or in alter-ego form.
	if msgs := b.React(g, p, engine.DamageEntity{Target: p.ID, Damage: 0}); msgs != nil {
		t.Fatalf("zero damage triggered Spider-Nonsense: %#v", msgs)
	}
	p.Side = engine.SideAlterEgo
	if msgs := b.React(g, p, engine.DamageEntity{Target: p.ID, Damage: 2}); msgs != nil {
		t.Fatalf("alter-ego damage triggered Spider-Nonsense: %#v", msgs)
	}
	msgs = b.React(g, p, engine.BasicRecover{Player: p.ID})
	if len(msgs) != 1 {
		t.Fatalf("recovery react returned %d messages, want one AddEntityCounter", len(msgs))
	}
}

// Contract test: toon counters are spendable as wild resources through the
// identity Resource hook (NoExhaust + UsesCounters), in either form.
func TestToonCountersAsWildResources(t *testing.T) {
	g := newHamGame(t)
	p := g.Players[0]
	b := engine.LookupBehavior("30001")
	if b == nil || b.Resource == nil {
		t.Fatal("Spider-Ham should expose an identity Resource hook")
	}
	if b.Resource.Icon != "wild" || !b.Resource.UsesCounters || !b.Resource.NoExhaust {
		t.Fatalf("Resource = %#v, want wild / UsesCounters / NoExhaust", b.Resource)
	}
	// With counters, the payment choices offer the toon counter option.
	p.Counters = 2
	card := engine.Card{ID: "play-me", Code: "30007", Owner: p.ID}
	p.Hand = append(p.Hand, card)
	choices := g.PaymentChoicesFor(p, card)
	found := false
	for _, c := range choices {
		if c.SourceID == p.ID && c.Kind == engine.ChoiceAbility {
			found = true
		}
	}
	if !found {
		t.Fatal("toon counters were not offered as a wild resource")
	}
	// Without counters the identity offers nothing.
	p.Counters = 0
	for _, c := range g.PaymentChoicesFor(p, card) {
		if c.SourceID == p.ID && c.Kind == engine.ChoiceAbility {
			t.Fatal("toon counter resource offered with zero counters")
		}
	}
}

// Contract test: "I Don't Think So!" cancels a treachery for 1 toon
// counter, and is unavailable without counters.
func TestIDontThinkSoCancels(t *testing.T) {
	g := newHamGame(t)
	p := g.Players[0]
	p.Side = engine.SideHero
	b := engine.LookupBehavior("30005")
	if b == nil || b.TreacheryInterrupt == nil {
		t.Fatal("\"I Don't Think So!\" should expose TreacheryInterrupt")
	}
	p.Counters = 0
	if repl := b.TreacheryInterrupt(g, p, engine.Card{Code: "30028"}); repl != nil {
		t.Fatalf("no-counter replacement = %#v, want nil", repl)
	}
	p.Counters = 2
	repl := b.TreacheryInterrupt(g, p, engine.Card{Code: "30028"})
	if len(repl) != 2 {
		t.Fatalf("replacement = %d messages, want counter payment + cancellation", len(repl))
	}
	if c, ok := repl[0].(engine.AddEntityCounter); !ok || c.N != -1 {
		t.Fatalf("first message = %#v, want -1 toon counter", repl[0])
	}
	if res, ok := repl[1].(engine.TreacheryResolve); !ok || !res.Cancelled {
		t.Fatalf("second message = %#v, want cancelled TreacheryResolve", repl[1])
	}
}

// Contract test: Cartoon Physics prevents all but 1 damage and discards
// itself.
func TestCartoonPhysicsPrevention(t *testing.T) {
	g := newHamGame(t)
	p := g.Players[0]
	u := &engine.Upgrade{ID: g.NextEntityID(engine.KindUpgrade), Code: "30009", Owner: p.ID}
	g.Upgrades[u.ID] = u
	p.Upgrades = append(p.Upgrades, u.ID)

	b := engine.LookupBehavior("30009")
	if b == nil || b.DamagePrevention == nil {
		t.Fatal("Cartoon Physics should expose DamagePrevention")
	}
	prevented, reflect := b.DamagePrevention(g, u, p, 4)
	if prevented != 3 || reflect != 0 {
		t.Fatalf("prevented = %d, reflect = %d; want 3, 0", prevented, reflect)
	}
	if g.Upgrades[u.ID] != nil {
		t.Fatal("Cartoon Physics should have been discarded")
	}
}
