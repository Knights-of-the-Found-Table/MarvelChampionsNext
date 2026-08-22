package venom_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/venom"
)

func villainOf(g *engine.Game) engine.EntityID {
	for id := range g.Villains {
		return id
	}
	return ""
}

func newVnmGame(t *testing.T, seed int64, setup func(g *engine.Game)) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Flash", HeroBase: "20001", Deck: map[string]int{
				"20002": 1, "20006": 1, "20012": 1,
				"20017": 3, "20018": 3, "20019": 3,
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

// unblockNoOp answers any plain leaf of the pending question so the
// queue drains.
func unblockNoOp(t *testing.T, g *engine.Game) {
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

// TestSavageAttackDealsFive.
func TestSavageAttackDealsFive(t *testing.T) {
	g := newVnmGame(t, 131, nil)
	p := g.Players[0]
	sa := engine.Card{ID: g.NextCardID(), Code: "20006", Owner: p.ID}
	p.Hand = append(p.Hand, sa)
	var villain engine.EntityID
	for id := range g.Villains {
		villain = id
	}
	hp := g.Villains[villain].HP()
	g.Push(engine.PlayCard{Player: p.ID, Card: sa, Paid: engine.CostPaid{}})
	unblockNoOp(t, g)
	for i := 0; i < 6; i++ {
		pq := g.Pending()
		if pq == nil || g.Over {
			break
		}
		if strings.Contains(pq.Question.Prompt, "Savage Attack") {
			_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
			break
		}
		_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
	}
	if g.Villains[villain].HP() != hp-5 {
		t.Fatalf("Savage Attack should deal 5 damage, %d -> %d", hp, g.Villains[villain].HP())
	}
}

// TestScareTacticSkipsNonConfused: no confused enemies → no damage.
func TestScareTacticSkipsNonConfused(t *testing.T) {
	g := newVnmGame(t, 132, nil)
	p := g.Players[0]
	villain := villainOf(g)
	st := engine.Card{ID: g.NextCardID(), Code: "20012", Owner: p.ID}
	p.Hand = append(p.Hand, st)
	hp := g.Villains[villain].HP()
	g.Push(engine.PlayCard{Player: p.ID, Card: st, Paid: engine.CostPaid{}})
	unblockNoOp(t, g)
	g.Run()
	if g.Villains[villain].HP() != hp {
		t.Fatal("Scare Tactic should not damage a non-confused enemy")
	}
}

// TestScareTacticHitsConfused: 3 damage to a confused enemy.
func TestScareTacticHitsConfused(t *testing.T) {
	g := newVnmGame(t, 133, func(g *engine.Game) {
		for id := range g.Villains {
			g.Villains[id].Confused = true
		}
	})
	p := g.Players[0]
	villain := villainOf(g)
	st := engine.Card{ID: g.NextCardID(), Code: "20012", Owner: p.ID}
	p.Hand = append(p.Hand, st)
	hp := g.Villains[villain].HP()
	g.Push(engine.PlayCard{Player: p.ID, Card: st, Paid: engine.CostPaid{}})
	unblockNoOp(t, g)
	if pq := g.Pending(); pq != nil && strings.Contains(pq.Question.Prompt, "Scare Tactic") {
		if err := g.Answer(pq.Player, []string{pq.Question.Choices[0].ID}); err != nil {
			t.Fatalf("answer: %v", err)
		}
	}
	g.Run()
	if g.Villains[villain].HP() != hp-3 {
		t.Fatalf("Scare Tactic should deal 3 damage to a confused enemy, %d -> %d", hp, g.Villains[villain].HP())
	}
}
