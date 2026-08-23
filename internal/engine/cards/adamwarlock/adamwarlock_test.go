package adamwarlock_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/adamwarlock"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
)

func warlockDeck() map[string]int {
	return map[string]int{"21032": 1, "21033": 1, "21034": 1, "21035": 1, "21036": 2, "21037": 2, "21038": 3, "21039": 2, "21040": 2}
}

func TestBattleMageAbilityAndSignatureResponsesContract(t *testing.T) {
	g, err := engine.NewGame(engine.NewGameOptions{Seed: 31, ScenarioID: "01097", Players: []engine.PlayerSpec{{Name: "Adam", HeroBase: "21031", Deck: warlockDeck()}}})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	p := g.Players[0]
	p.Side = engine.SideHero
	p.Hand = append(p.Hand, engine.Card{ID: g.NextCardID(), Code: "21044", Owner: p.ID})
	for _, code := range []string{"21035", "21037"} {
		u := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: code, Owner: p.ID}
		g.Upgrades[u.ID] = u
		p.Upgrades = append(p.Upgrades, u.ID)
	}
	b := engine.LookupBehavior("21031")
	if b == nil || b.HeroAbilities == nil || b.React == nil {
		t.Fatal("Adam Warlock must expose HeroAbilities and React")
	}
	abilities := b.HeroAbilities(g, p)
	if len(abilities) != 1 || abilities[0].Execute == nil || !abilities[0].OncePerTurn {
		t.Fatalf("Battle Mage ability = %#v", abilities)
	}
	msgs := abilities[0].Execute(g, p.ID)
	if len(msgs) != 1 {
		t.Fatalf("Battle Mage returned %d messages, want a discard question", len(msgs))
	}
	ask, ok := msgs[0].(engine.AskQuestion)
	if !ok || ask.Question == nil || len(ask.Question.Choices) == 0 {
		t.Fatalf("Battle Mage = %#v, want aspect-card choices", msgs[0])
	}
	responses := b.React(g, p, engine.AddEntityCounter{ID: p.ID, N: -21031})
	if len(responses) != 3 {
		t.Fatalf("Cape + Mystic Senses responses = %d messages, want exhaust, ready, draw", len(responses))
	}
	if _, ok := responses[2].(engine.DrawCards); !ok {
		t.Fatalf("last response = %T, want DrawCards", responses[2])
	}
}
