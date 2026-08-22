package gamora_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/gamora"
)

func newGamGame(t *testing.T, seed int64, setup func(g *engine.Game)) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Gamora", HeroBase: "18001", Deck: map[string]int{
				"18003": 2, "18005": 1, "18014": 1, "18029": 1,
				"18021": 3, "18022": 3, "18023": 3,
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

// TestPrecisionTriggersOnThwartEvent: playing a thwart event fires the
// identity's Precision rider (1 damage to an enemy).
func TestPrecisionTriggersOnThwartEvent(t *testing.T) {
	g := newGamGame(t, 51, nil)
	p := g.Players[0]
	p.Side = engine.SideHero
	p.Exhausted = false
	stp := engine.Card{ID: g.NextCardID(), Code: "18005", Owner: p.ID} // Set the Pace (thwart)
	p.Hand = append(p.Hand, stp)
	var villain engine.EntityID
	for id := range g.Villains {
		villain = id
	}
	hp := g.Villains[villain].HP()

	g.Push(engine.PlayCard{Player: p.ID, Card: stp, Paid: engine.CostPaid{}})
	// Drain: Set the Pace scheme question, then Precision enemy question.
	for i := 0; i < 8; i++ {
		pq := g.Pending()
		if pq == nil || g.Over {
			break
		}
		if strings.Contains(pq.Question.Prompt, "Set the Pace") {
			_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
			continue
		}
		if strings.Contains(pq.Question.Prompt, "Precision") {
			_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
			break
		}
		_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
	}
	if g.Villains[villain].HP() != hp-1 {
		t.Fatalf("Precision should deal 1 damage, %d -> %d", hp, g.Villains[villain].HP())
	}
}

// TestUppercutDealsFive.
func TestUppercutDealsFive(t *testing.T) {
	g := newGamGame(t, 52, nil)
	p := g.Players[0]
	up := engine.Card{ID: g.NextCardID(), Code: "18014", Owner: p.ID}
	p.Hand = append(p.Hand, up)
	var villain engine.EntityID
	for id := range g.Villains {
		villain = id
	}
	hp := g.Villains[villain].HP()
	g.Push(engine.PlayCard{Player: p.ID, Card: up, Paid: engine.CostPaid{}})
	for i := 0; i < 6; i++ {
		pq := g.Pending()
		if pq == nil || g.Over {
			break
		}
		// Answer the rider/turn prompts but stop once Uppercut resolves
		// so no further cards get played.
		if strings.Contains(pq.Question.Prompt, "Uppercut") {
			_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
			break
		}
		if err := g.Answer(pq.Player, []string{pq.Question.Choices[0].ID}); err != nil {
			break
		}
	}
	g.Run()
	if g.Villains[villain].HP() != hp-5 {
		t.Fatalf("Uppercut should deal 5 damage, %d -> %d", hp, g.Villains[villain].HP())
	}
}

// TestPivotalMomentBonusWithCleanScheme: 5 damage when the main scheme
// has no threat.
func TestPivotalMomentBonusWithCleanScheme(t *testing.T) {
	g := newGamGame(t, 53, func(g *engine.Game) {
		if g.MainScheme != nil {
			g.MainScheme.Threat = 0
		}
	})
	p := g.Players[0]
	pm := engine.Card{ID: g.NextCardID(), Code: "18029", Owner: p.ID}
	p.Hand = append(p.Hand, pm)
	var villain engine.EntityID
	for id := range g.Villains {
		villain = id
	}
	hp := g.Villains[villain].HP()
	g.Push(engine.PlayCard{Player: p.ID, Card: pm, Paid: engine.CostPaid{}})
	// The attack event triggers Finesse first (threat question); answer
	// through until the queued damage lands.
	for i := 0; i < 6 && g.Villains[villain].HP() == hp; i++ {
		pq := g.Pending()
		if pq == nil || g.Over {
			break
		}
		if err := g.Answer(pq.Player, []string{pq.Question.Choices[0].ID}); err != nil {
			break
		}
	}
	g.Run()
	if g.Villains[villain].HP() != hp-5 {
		t.Fatalf("Pivotal Moment should deal 5 with a clean main scheme, %d -> %d", hp, g.Villains[villain].HP())
	}
}
