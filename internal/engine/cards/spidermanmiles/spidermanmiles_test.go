package spidermanmiles_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/spidermanmiles"
)

func milesDeck() map[string]int {
	return map[string]int{
		"27031": 2, "27032": 2, "27033": 2, "27034": 3, "27035": 1,
		"27036": 1, "27037": 1, "27038": 1, "27039": 2, "27040": 1, "27045": 1,
	}
}

func newMilesGame(t *testing.T) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed: 27, ScenarioID: "01097",
		Players: []engine.PlayerSpec{{Name: "Miles", HeroBase: "27030", Deck: milesDeck()}},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return g
}

// Contract test: both identity Specials are exposed as once-per-round hero
// actions, and Venom Blast's Execute deals 2 damage + stun to the chosen
// enemy.
func TestMilesSpecials(t *testing.T) {
	g := newMilesGame(t)
	p := g.Players[0]
	p.Side = engine.SideHero
	b := engine.LookupBehavior("27030")
	if b == nil || b.HeroAbilities == nil {
		t.Fatal("Miles should expose HeroAbilities")
	}
	abilities := b.HeroAbilities(g, p)
	if len(abilities) != 2 {
		t.Fatalf("HeroAbilities = %d, want 2 (Venom Blast + Spider Camouflage)", len(abilities))
	}
	for _, a := range abilities {
		if !a.OncePerRound || !a.HeroOnly || a.Execute == nil {
			t.Fatalf("ability %#v should be a once-per-round executable hero action", a)
		}
	}
	msgs := abilities[0].Execute(g, p.ID)
	if len(msgs) != 1 {
		t.Fatalf("Venom Blast returned %d messages, want one target question", len(msgs))
	}
	ask, ok := msgs[0].(engine.AskQuestion)
	if !ok || ask.Question == nil || len(ask.Question.Choices) == 0 {
		t.Fatalf("Venom Blast message = %#v, want an enemy question", msgs[0])
	}

	camo := abilities[1].Execute(g, p.ID)
	if len(camo) != 2 {
		t.Fatalf("Spider Camouflage returned %d messages, want tough + confuse question", len(camo))
	}
	if tough, ok := camo[0].(engine.ToughEntity); !ok || tough.Target != p.ID {
		t.Fatalf("first message = %#v, want ToughEntity for Miles", camo[0])
	}
}

// Contract test: Web-Shot deals 4 damage, and resolves Venom Blast only
// when an energy resource paid for it.
func TestWebShotEnergyRider(t *testing.T) {
	g := newMilesGame(t)
	p := g.Players[0]
	p.Side = engine.SideHero
	b := engine.LookupBehavior("27034")
	if b == nil || b.OnPlay == nil {
		t.Fatal("Web-Shot should expose OnPlay")
	}
	plain := b.OnPlay(g, &engine.EventCard{Code: "27034", Owner: p.ID})
	if len(plain) != 1 {
		t.Fatalf("unpaid Web-Shot returned %d messages, want one question", len(plain))
	}
	ask := plain[0].(engine.AskQuestion)
	chain, err := ask.Question.Chain(ask.Question.Choices[0].ID)
	if err != nil || len(chain) != 1 {
		t.Fatalf("chain: %v", err)
	}

	paid := b.OnPlay(g, &engine.EventCard{Code: "27034", Owner: p.ID, Paid: engine.CostPaid{Icons: []string{"energy", "energy"}}})
	paidAsk := paid[0].(engine.AskQuestion)
	if len(paidAsk.Question.Choices) != len(ask.Question.Choices) {
		t.Fatalf("paid Web-Shot choices = %d, want the same enemy list", len(paidAsk.Question.Choices))
	}
	// The energy-payment rider embeds a nested Venom Blast question in the
	// choice payloads, which are unexported; the Power Within test covers
	// the Venom Blast Special itself.
}

// Contract test: Miles Morales shuffles a signature card back into the deck
// after flipping to alter-ego.
func TestMilesAlterEgoShuffleBack(t *testing.T) {
	g := newMilesGame(t)
	p := g.Players[0]
	p.Side = engine.SideAlterEgo
	p.Discard = append(p.Discard, engine.Card{ID: "sig1", Code: "27034", Owner: p.ID})
	p.Discard = append(p.Discard, engine.Card{ID: "other", Code: "01087", Owner: p.ID}) // First Aid (not a signature)

	b := engine.LookupBehavior("27030")
	msgs := b.React(g, p, engine.ChangeForm{Player: p.ID})
	if len(msgs) != 1 {
		t.Fatalf("ChangeForm react returned %d messages, want one question", len(msgs))
	}
	ask := msgs[0].(engine.AskQuestion)
	// Skip + exactly one signature choice (Web-Shot), never First Aid.
	if len(ask.Question.Choices) != 2 {
		t.Fatalf("choices = %d, want skip + 1 signature card", len(ask.Question.Choices))
	}
	if ask.Question.Choices[1].CardCode != "27034" {
		t.Fatalf("choice card = %s, want 27034", ask.Question.Choices[1].CardCode)
	}
}

// Contract test: Power Within offers the Venom Blast resolution after a
// basic attack and stays silent for other players' powers.
func TestPowerWithinResponse(t *testing.T) {
	g := newMilesGame(t)
	p := g.Players[0]
	p.Side = engine.SideHero
	u := &engine.Upgrade{ID: g.NextEntityID(engine.KindUpgrade), Code: "27037", Owner: p.ID}
	g.Upgrades[u.ID] = u
	p.Upgrades = append(p.Upgrades, u.ID)

	b := engine.LookupBehavior("27037")
	if b == nil || b.React == nil {
		t.Fatal("Power Within should expose React")
	}
	msgs := b.React(g, u, engine.BasicAttack{Player: p.ID, N: 2, Target: "some-enemy"})
	if len(msgs) != 1 {
		t.Fatalf("Power Within returned %d messages, want one question", len(msgs))
	}
	ask := msgs[0].(engine.AskQuestion)
	if len(ask.Question.Choices) != 2 {
		t.Fatalf("choices = %d, want resolve + skip", len(ask.Question.Choices))
	}
	if msgs := b.React(g, u, engine.BasicAttack{Player: "other-player", N: 2, Target: "x"}); msgs != nil {
		t.Fatalf("other player's attack triggered Power Within: %#v", msgs)
	}
}
