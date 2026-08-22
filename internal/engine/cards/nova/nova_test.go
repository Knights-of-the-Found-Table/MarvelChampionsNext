package nova_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/nova"
)

func newNovaGame(t *testing.T, seed int64, setup func(g *engine.Game)) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Sam", HeroBase: "28001", Deck: map[string]int{
				"28004": 1, "28005": 1, "28006": 1,
				"28024": 3, "28025": 3, "28026": 3,
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

// TestPotShotDealsFour.
func TestPotShotDealsFour(t *testing.T) {
	g := newNovaGame(t, 71, nil)
	p := g.Players[0]
	ps := engine.Card{ID: g.NextCardID(), Code: "28005", Owner: p.ID}
	p.Hand = append(p.Hand, ps)
	var villain engine.EntityID
	for id := range g.Villains {
		villain = id
	}
	hp := g.Villains[villain].HP()
	g.Push(engine.PlayCard{Player: p.ID, Card: ps, Paid: engine.CostPaid{}})
	for i := 0; i < 6; i++ {
		pq := g.Pending()
		if pq == nil || g.Over {
			break
		}
		if strings.Contains(pq.Question.Prompt, "Pot Shot") {
			_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
			break
		}
		_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
	}
	if g.Villains[villain].HP() != hp-4 {
		t.Fatalf("Pot Shot should deal 4 damage, %d -> %d", hp, g.Villains[villain].HP())
	}
}

// TestUnleashNovaForceReadiesAndDraws: after activation, defeating a
// minion readies Nova and draws 1.
func TestUnleashNovaForceReadiesAndDraws(t *testing.T) {
	g := newNovaGame(t, 72, func(g *engine.Game) {
		g.Players[0].Side = engine.SideHero
		g.Players[0].Exhausted = true
	})
	p := g.Players[0]
	unf := engine.Card{ID: g.NextCardID(), Code: "28006", Owner: p.ID}
	p.Hand = append(p.Hand, unf)
	for len(p.Deck) > 2 {
		p.Deck = p.Deck[:2]
	}
	mn := &engine.Minion{ID: g.NextEntityID("minion"), Code: "01101", MaxHP: 1, EngagedWith: p.ID}
	g.Minions[mn.ID] = mn

	g.Push(engine.PlayCard{Player: p.ID, Card: unf, Paid: engine.CostPaid{}})
	g.Push(engine.DamageEntity{Target: mn.ID, Damage: 9, Source: p.ID})
	hand := len(p.Hand)
	// Unblock the queue with a no-op root answer (a costed hand-play
	// choice carries no messages until its payment leaf).
	if pq := g.Pending(); pq != nil {
		_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
	}
	g.Run()
	// unf left hand when played; the marker draw brings it back to par.
	if len(p.Hand) != hand {
		t.Fatalf("Unleash Nova Force should draw 1 on defeat (hand %d -> %d)", hand, len(p.Hand))
	}
	if p.Exhausted {
		t.Fatal("Unleash Nova Force should ready Nova on defeat")
	}
}
