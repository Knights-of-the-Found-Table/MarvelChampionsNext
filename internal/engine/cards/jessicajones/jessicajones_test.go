package jessicajones_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/jessicajones"
)

func newJJGame(t *testing.T) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{Seed: 61001, ScenarioID: "01097", Players: []engine.PlayerSpec{{Name: "Jessica", HeroBase: "61001", Deck: map[string]int{"61002": 1, "61004": 1}}}})
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestJessicaJonesContractEvidenceAfterBasicPower(t *testing.T) {
	g := newJJGame(t)
	p := g.Players[0]
	s := &engine.Support{ID: g.NextEntityID(engine.KindSupport), Code: "61002", Owner: p.ID}
	g.Supports[s.ID] = s
	p.Supports = append(p.Supports, s.ID)
	// NewGame opens in alter-ego; flip once directly, without entering the
	// turn state machine, so this contract exercises the hero-only response.
	g.Push(engine.ChangeForm{Player: p.ID})
	g.Run()
	b := engine.LookupBehavior("61001")
	if b.React == nil {
		t.Fatal("Jessica identity must expose Gather Evidence")
	}
	msgs := b.React(g, p, engine.BasicThwart{Player: p.ID, N: 1, Target: g.MainScheme.ID})
	if len(msgs) != 1 {
		t.Fatalf("Gather Evidence messages = %d, want 1", len(msgs))
	}
	counter, ok := msgs[0].(engine.AddEntityCounter)
	if !ok || counter.ID != s.ID || counter.N != 1 {
		t.Fatalf("Gather Evidence message = %#v, want +1 on Alias Investigations", msgs[0])
	}
}

func TestAliasInvestigationsContractAddsTwoOnSideSchemeDefeat(t *testing.T) {
	g := newJJGame(t)
	p := g.Players[0]
	s := &engine.Support{ID: g.NextEntityID(engine.KindSupport), Code: "61002", Owner: p.ID}
	g.Supports[s.ID] = s
	p.Supports = append(p.Supports, s.ID)
	scheme := &engine.SideScheme{ID: g.NextEntityID(engine.KindSideScheme), Code: "01116", Threat: 1, MaxThreat: 1}
	g.SideSchemes[scheme.ID] = scheme
	b := engine.LookupBehavior("61002")
	msgs := b.React(g, s, engine.SchemeDefeated{Scheme: scheme.ID})
	if s.Counters != 2 {
		t.Fatalf("evidence counters = %d, want 2", s.Counters)
	}
	if len(msgs) == 0 {
		t.Fatal("Alias Investigations should exhaust after a defeated side scheme")
	}
}

func TestJessicaJonesPackRegistersNemesisAndObligationHooks(t *testing.T) {
	for _, code := range []string{"61030", "61031", "61032", "61033a", "61033b", "61033c", "61040"} {
		if b := engine.LookupBehavior(code); b == nil {
			t.Fatalf("missing behavior registration for %s", code)
		}
	}
	if engine.LookupBehavior("61031").EnemyStatBonus == nil {
		t.Fatal("Purple Man must expose dynamic stats")
	}
}
