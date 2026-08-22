package warmachine_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/warmachine"
)

func newWarmGame(t *testing.T, seed int64, setup func(g *engine.Game)) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Rhodes", HeroBase: "23001", Deck: map[string]int{
				"23008": 1, "23010": 1, "23011": 1,
				"23025": 3, "23026": 3, "23027": 3,
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

func unblock(t *testing.T, g *engine.Game) {
	t.Helper()
	if pq := g.Pending(); pq != nil {
		for _, c := range pq.Question.Choices {
			if c.Then == nil && !c.Disabled {
				if err := g.Answer(pq.Player, []string{c.ID}); err == nil {
					return
				}
			}
		}
	}
}

// TestLockedAndLoadedGivesAmmo: entering hero form loads 5 ammo, and
// Full Auto (queued behind the flip) spends 4 of them for 8 damage.
func TestLockedAndLoadedGivesAmmo(t *testing.T) {
	g := newWarmGame(t, 141, nil)
	p := g.Players[0]
	fa := engine.Card{ID: g.NextCardID(), Code: "23011", Owner: p.ID}
	p.Hand = append(p.Hand, fa)
	var villain engine.EntityID
	for id := range g.Villains {
		villain = id
	}
	hp := g.Villains[villain].HP()

	g.Push(engine.PlayCard{Player: p.ID, Card: fa, Paid: engine.CostPaid{}})
	unblock(t, g) // "form": flips to hero, +5 ammo, then Full Auto runs
	pq := g.Pending()
	if pq == nil || !strings.Contains(pq.Question.Prompt, "Full Auto") {
		t.Fatalf("expected the Full Auto question, got %q", func() string {
			if pq != nil {
				return pq.Question.Prompt
			}
			return "<none>"
		}())
	}
	if err := g.Answer(pq.Player, []string{pq.Question.Choices[0].ID}); err != nil {
		t.Fatalf("answer Full Auto: %v", err)
	}
	if g.Villains[villain].HP() != hp-8 {
		t.Fatalf("Full Auto should deal 8 damage, %d -> %d", hp, g.Villains[villain].HP())
	}
	if p.GrowthCounters != 1 {
		t.Fatalf("Full Auto should spend 4 ammo (5 -> 1), got %d", p.GrowthCounters)
	}
}

// TestScorchedEarthNeedsThreeAmmo: with fewer than 3 ammo the event does
// nothing.
func TestScorchedEarthNeedsThreeAmmo(t *testing.T) {
	g := newWarmGame(t, 142, func(g *engine.Game) {
		g.Players[0].GrowthCounters = 2
		g.Players[0].Side = engine.SideHero // no flip → no ammo reload
	})
	p := g.Players[0]
	se := engine.Card{ID: g.NextCardID(), Code: "23010", Owner: p.ID}
	p.Hand = append(p.Hand, se)
	var villain engine.EntityID
	for id := range g.Villains {
		villain = id
	}
	hp := g.Villains[villain].HP()
	g.Push(engine.PlayCard{Player: p.ID, Card: se, Paid: engine.CostPaid{}})
	unblock(t, g)
	g.Run()
	if g.Villains[villain].HP() != hp {
		t.Fatal("Scorched Earth should do nothing without 3 ammo")
	}
}
