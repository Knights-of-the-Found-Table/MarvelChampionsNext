package spiderwoman_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/spiderwoman"
)

func newScwGame(t *testing.T, seed int64, setup func(g *engine.Game)) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Wanda", HeroBase: "15001", Deck: map[string]int{
				"15012": 1, "15014": 1, "15028": 1,
				"15020": 3, "" + "15021": 3, "15022": 3,
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

// TestCrisisAvertedRemovesSix.
func TestCrisisAvertedRemovesSix(t *testing.T) {
	g := newScwGame(t, 91, nil)
	p := g.Players[0]
	ca := engine.Card{ID: g.NextCardID(), Code: "15012", Owner: p.ID}
	p.Hand = append(p.Hand, ca)
	threat := g.MainScheme.Threat
	g.Push(engine.PlayCard{Player: p.ID, Card: ca, Paid: engine.CostPaid{}})
	if pq := g.Pending(); pq != nil {
		_ = g.Answer(pq.Player, []string{"form"})
	}
	g.Run()
	if g.MainScheme.Threat != max(0, threat-6) {
		t.Fatalf("Crisis Averted should remove 6 threat, %d -> %d", threat, g.MainScheme.Threat)
	}
}

// TestBrowbeatScalesWithStage: 2 + min(3, stage).
func TestBrowbeatScalesWithStage(t *testing.T) {
	g := newScwGame(t, 92, func(g *engine.Game) {
		for _, v := range g.Villains {
			v.Stage = 2 // expert Rhino II: 2 + 2 = 4 damage
		}
	})
	p := g.Players[0]
	bb := engine.Card{ID: g.NextCardID(), Code: "15028", Owner: p.ID}
	p.Hand = append(p.Hand, bb)
	var villain engine.EntityID
	for id := range g.Villains {
		villain = id
	}
	hp := g.Villains[villain].HP()
	g.Push(engine.PlayCard{Player: p.ID, Card: bb, Paid: engine.CostPaid{}})
	if pq := g.Pending(); pq != nil {
		_ = g.Answer(pq.Player, []string{"form"})
	}
	g.Run()
	if g.Villains[villain].HP() != hp-4 {
		t.Fatalf("Browbeat should deal 4 at villain stage 2, %d -> %d", hp, g.Villains[villain].HP())
	}
	_ = strings.TrimSpace("")
}
