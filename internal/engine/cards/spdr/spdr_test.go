package spdr_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/spdr"
)

func spdrDeck() map[string]int {
	return map[string]int{
		"31003": 1, "31004": 3, "31005": 2, "31006": 2, "31007": 1,
		"31008": 1, "31009": 1, "31010": 1, "31011": 1, "31012": 1, "31013": 1,
	}
}

func newSPDRGame(t *testing.T) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed: 31, ScenarioID: "01097",
		Players: []engine.PlayerSpec{{Name: "SP//dr", HeroBase: "31001", Deck: spdrDeck()}},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return g
}

// Contract test: Interface upgrades expose hero-only Sync Ratio resources
// with their printed icons.
func TestSyncRatioResources(t *testing.T) {
	g := newSPDRGame(t)
	p := g.Players[0]
	p.Side = engine.SideHero
	want := map[string]string{"31010": "wild", "31011": "mental", "31012": "physical", "31013": "energy"}
	for code, icon := range want {
		b := engine.LookupBehavior(code)
		if b == nil || b.Resource == nil {
			t.Fatalf("%s should expose a Resource hook", code)
		}
		if b.Resource.Icon != icon || !b.Resource.HeroOnly {
			t.Fatalf("%s resource = %#v, want hero-only [%s]", code, b.Resource, icon)
		}
	}
	// They show up as payment producers once in play.
	u := &engine.Upgrade{ID: g.NextEntityID(engine.KindUpgrade), Code: "31011", Owner: p.ID}
	g.Upgrades[u.ID] = u
	p.Upgrades = append(p.Upgrades, u.ID)
	card := engine.Card{ID: "play-me", Code: "31006", Owner: p.ID}
	found := false
	for _, c := range g.PaymentChoicesFor(p, card) {
		if c.SourceID == u.ID && c.Kind == engine.ChoiceAbility {
			found = true
		}
	}
	if !found {
		t.Fatal("Psychic Link was not offered as a Sync Ratio resource")
	}
}

// Contract test: Psychic Link adds follow-up threat removal after SP//dr's
// basic thwart and exhausts itself; Web-Fluid Compressor mirrors it for
// attacks.
func TestInterfaceBoosts(t *testing.T) {
	g := newSPDRGame(t)
	p := g.Players[0]
	p.Side = engine.SideHero
	link := &engine.Upgrade{ID: g.NextEntityID(engine.KindUpgrade), Code: "31011", Owner: p.ID}
	comp := &engine.Upgrade{ID: g.NextEntityID(engine.KindUpgrade), Code: "31013", Owner: p.ID}
	g.Upgrades[link.ID] = link
	g.Upgrades[comp.ID] = comp
	p.Upgrades = append(p.Upgrades, link.ID, comp.ID)

	lb := engine.LookupBehavior("31011")
	msgs := lb.React(g, link, engine.BasicThwart{Player: p.ID, N: 2, Target: g.MainScheme.ID})
	if len(msgs) != 2 {
		t.Fatalf("Psychic Link returned %d messages, want exhaust + ThwartScheme", len(msgs))
	}
	if tw, ok := msgs[1].(engine.ThwartScheme); !ok || tw.N != 2 || tw.Scheme != g.MainScheme.ID {
		t.Fatalf("second message = %#v, want +2 threat removal on the thwarted scheme", msgs[1])
	}
	// Another player's thwart does not trigger it.
	if msgs := lb.React(g, link, engine.BasicThwart{Player: "other", N: 2, Target: g.MainScheme.ID}); msgs != nil {
		t.Fatalf("other player's thwart triggered Psychic Link: %#v", msgs)
	}

	cb := engine.LookupBehavior("31013")
	msgs = cb.React(g, comp, engine.BasicAttack{Player: p.ID, N: 2, Target: "some-enemy"})
	if len(msgs) != 2 {
		t.Fatalf("Web-Fluid Compressor returned %d messages, want exhaust + DamageEntity", len(msgs))
	}
	if d, ok := msgs[1].(engine.DamageEntity); !ok || d.Damage != 2 {
		t.Fatalf("second message = %#v, want 2 follow-up damage", msgs[1])
	}
}

// Contract test: Ejection Protocol discards itself, exhausts each Interface
// upgrade, sets HP to 6, grants tough and flips to alter-ego.
func TestEjectionProtocol(t *testing.T) {
	g := newSPDRGame(t)
	p := g.Players[0]
	p.Side = engine.SideHero
	p.Damage = 0
	s := &engine.Support{ID: g.NextEntityID(engine.KindSupport), Code: "31008", Owner: p.ID}
	g.Supports[s.ID] = s
	p.Supports = append(p.Supports, s.ID)
	u := &engine.Upgrade{ID: g.NextEntityID(engine.KindUpgrade), Code: "31010", Owner: p.ID}
	g.Upgrades[u.ID] = u
	p.Upgrades = append(p.Upgrades, u.ID)

	b := engine.LookupBehavior("31008")
	abilities := b.Abilities(g, s)
	if len(abilities) != 1 {
		t.Fatalf("Ejection Protocol abilities = %d, want 1", len(abilities))
	}
	msgs := abilities[0].Execute(g, s.ID)
	// DiscardControlled + ExhaustEntity(Host Spider) + ToughEntity + ChangeForm.
	if len(msgs) != 4 {
		t.Fatalf("Ejection Protocol returned %d messages, want 4", len(msgs))
	}
	if p.Damage != p.MaxHP-6 {
		t.Fatalf("damage = %d, want dial set to 6 HP (%d)", p.Damage, p.MaxHP-6)
	}
	if _, ok := msgs[2].(engine.ToughEntity); !ok {
		t.Fatalf("third message = %#v, want ToughEntity", msgs[2])
	}
	if cf, ok := msgs[3].(engine.ChangeForm); !ok || cf.Player != p.ID {
		t.Fatalf("fourth message = %#v, want ChangeForm", msgs[3])
	}
}

// Contract test: M.O.R.B.I.U.S. damages the engaged hero for each resource
// icon they spend from hand.
func TestMorbiusResourceBacklash(t *testing.T) {
	g := newSPDRGame(t)
	p := g.Players[0]
	p.Side = engine.SideHero
	mn := &engine.Minion{ID: g.NextEntityID(engine.KindMinion), Code: "31027", MaxHP: 6, EngagedWith: p.ID}
	g.Minions[mn.ID] = mn

	b := engine.LookupBehavior("31027")
	if b == nil || b.React == nil {
		t.Fatal("M.O.R.B.I.U.S. should expose a React hook")
	}
	pay := engine.ResourcePay{Player: p.ID, Cards: engine.CardList{
		{Code: "31006", Owner: p.ID}, // 1 energy icon
		{Code: "31010", Owner: p.ID}, // 1 wild icon
	}}
	msgs := b.React(g, mn, pay)
	if len(msgs) != 1 {
		t.Fatalf("M.O.R.B.I.U.S. returned %d messages, want 1", len(msgs))
	}
	if d, ok := msgs[0].(engine.DamageEntity); !ok || d.Damage != 2 || d.Target != p.ID {
		t.Fatalf("message = %#v, want 2 damage to the engaged hero", msgs[0])
	}
	// Not engaged with the payer → silent.
	mn.EngagedWith = "someone-else"
	if msgs := b.React(g, mn, pay); msgs != nil {
		t.Fatalf("M.O.R.B.I.U.S. reacted to another player's payment: %#v", msgs)
	}
}
