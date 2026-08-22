package valkyrie_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/valkyrie"
)

func newValkGame(t *testing.T, seed int64, setup func(g *engine.Game)) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Brunnhilde", HeroBase: "25001", Deck: map[string]int{
				"25012": 1, "25018": 1, "25024": 1,
				"25025": 3, "25026": 3, "25027": 3,
			}},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	if setup != nil {
		setup(g)
	}
	for i := 0; i < 5; i++ {
		pq := g.Pending()
		if pq == nil || pq.Question.Prompt == "Your turn" {
			break
		}
		_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
	}
	return g
}

// TestDeathGlowReadiesOnDefeat: the marked minion's defeat sets the glow
// aside and readies Valkyrie.
func TestDeathGlowReadiesOnDefeat(t *testing.T) {
	g := newValkGame(t, 111, nil)
	p := g.Players[0]
	p.Side = engine.SideHero
	mn := &engine.Minion{ID: g.NextEntityID("minion"), Code: "01101", MaxHP: 3, EngagedWith: p.ID}
	g.Minions[mn.ID] = mn
	glow := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "25002", Owner: p.ID, AttachTo: mn.ID}
	g.Upgrades[glow.ID] = glow
	p.Upgrades = append(p.Upgrades, glow.ID)
	p.Exhausted = true

	g.Push(engine.DamageEntity{Target: mn.ID, Damage: 9, Source: p.ID})
	if pq := g.Pending(); pq != nil {
		_ = g.Answer(pq.Player, []string{"form"})
	}
	g.Run()
	if g.Minions[mn.ID] != nil {
		t.Fatal("minion should be defeated")
	}
	if g.Upgrades[glow.ID] != nil {
		t.Fatal("Death-Glow should be set aside (removed from play)")
	}
	if p.Exhausted {
		t.Fatal("Valkyrie should ready when her marked enemy falls")
	}
}

// TestHaveAtTheeDealsSeven.
func TestHaveAtTheeDealsSeven(t *testing.T) {
	g := newValkGame(t, 112, nil)
	p := g.Players[0]
	hat := engine.Card{ID: g.NextCardID(), Code: "25012", Owner: p.ID}
	p.Hand = append(p.Hand, hat)
	var villain engine.EntityID
	for id := range g.Villains {
		villain = id
	}
	hp := g.Villains[villain].HP()
	g.Push(engine.PlayCard{Player: p.ID, Card: hat, Paid: engine.CostPaid{}})
	for i := 0; i < 6; i++ {
		pq := g.Pending()
		if pq == nil || g.Over {
			break
		}
		if strings.Contains(pq.Question.Prompt, "Have at Thee") {
			_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
			break
		}
		_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
	}
	if g.Villains[villain].HP() != hp-7 {
		t.Fatalf("Have at Thee! should deal 7 damage, %d -> %d", hp, g.Villains[villain].HP())
	}
}
