package drax_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/drax"
)

func newDraxGame(t *testing.T, seed int64, setup func(g *engine.Game)) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Drax", HeroBase: "19001", Deck: map[string]int{
				"19003": 1, "19004": 1, "19030": 1,
				"19022": 3, "19023": 3, "19024": 3,
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

// TestBringItDrawsPerMinion.
func TestBringItDrawsPerMinion(t *testing.T) {
	g := newDraxGame(t, 41, func(g *engine.Game) {
		p := g.Players[0]
		for i := 0; i < 2; i++ {
			mn := &engine.Minion{ID: g.NextEntityID("minion"), Code: "01101", MaxHP: 3, EngagedWith: p.ID}
			g.Minions[mn.ID] = mn
		}
	})
	p := g.Players[0]
	bi := engine.Card{ID: g.NextCardID(), Code: "19030", Owner: p.ID}
	p.Hand = append(p.Hand, bi)
	for len(p.Deck) > 2 {
		p.Deck = p.Deck[:2]
	}
	hand := len(p.Hand)
	g.Push(engine.PlayCard{Player: p.ID, Card: bi, Paid: engine.CostPaid{}})
	if pq := g.Pending(); pq != nil {
		_ = g.Answer(pq.Player, []string{"form"})
	}
	g.Run()
	if len(p.Hand) != hand+1 { // played card gone, drew 2
		t.Fatalf("Bring It! should draw 2, hand %d -> %d", hand, len(p.Hand))
	}
}

// TestTooStubbornToDie: defeat save sets HP to 4 in alter-ego form.
func TestTooStubbornToDie(t *testing.T) {
	g := newDraxGame(t, 42, nil)
	p := g.Players[0]
	p.Side = engine.SideHero
	ts := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "19011", Owner: p.ID}
	g.Upgrades[ts.ID] = ts
	p.Upgrades = append(p.Upgrades, ts.ID)
	g.Push(engine.DamageEntity{Target: p.ID, Damage: 999, Source: engine.EntityID("villain-2")})
	if pq := g.Pending(); pq != nil {
		_ = g.Answer(pq.Player, []string{"form"})
	}
	g.Run()
	if g.Over {
		t.Fatal("Too Stubborn to Die should prevent defeat")
	}
	if p.HP() != 4 {
		t.Fatalf("should survive at 4 HP, got %d", p.HP())
	}
	if p.IsHero() {
		t.Fatal("should flip to alter-ego form")
	}
	if g.Upgrades[ts.ID] != nil {
		t.Fatal("Too Stubborn to Die should be removed from the game")
	}
}

// TestVengeanceAfterVillainAttack: the villain attacking Drax grants a
// vengeance counter (+1 ATK).
func TestVengeanceAfterVillainAttack(t *testing.T) {
	g := newDraxGame(t, 43, func(g *engine.Game) {
		g.Players[0].Side = engine.SideHero // before the menu builds
	})
	p := g.Players[0]
	atk := p.AttackStat(g)
	var villain engine.EntityID
	for id := range g.Villains {
		villain = id
	}
	g.Push(engine.WindowAfterEnemyAttacked{Enemy: villain, Player: p.ID})
	// Unblock via a no-op drill into the attack question (the turn menu
	// choice carries no messages until a target leaf is answered).
	if pq := g.Pending(); pq != nil {
		_ = g.Answer(pq.Player, []string{"basic-attack"})
	}
	g.Run()
	if p.AttackStat(g) != atk+1 {
		t.Fatalf("vengeance should grant +1 ATK, %d -> %d", atk, p.AttackStat(g))
	}
	if p.GrowthCounters != 1 {
		t.Fatalf("expected 1 vengeance counter, got %d", p.GrowthCounters)
	}
	_ = strings.TrimSpace("")
}
