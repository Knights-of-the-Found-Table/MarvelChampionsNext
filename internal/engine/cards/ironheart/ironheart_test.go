package ironheart_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/ironheart"
)

func newIHGame(t *testing.T, seed int64, setup func(g *engine.Game)) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Riri", HeroBase: "29001", Deck: map[string]int{
				"29005": 1, "29006": 1, "29017": 1,
				"29009": 2, "29021": 2, "29027": 2,
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

// TestPhotonBeamAddsProgress: 4 damage + 1 progress counter.
func TestPhotonBeamAddsProgress(t *testing.T) {
	g := newIHGame(t, 161, nil)
	p := g.Players[0]
	pb := engine.Card{ID: g.NextCardID(), Code: "29006", Owner: p.ID}
	p.Hand = append(p.Hand, pb)
	var villain engine.EntityID
	for id := range g.Villains {
		villain = id
	}
	hp := g.Villains[villain].HP()
	prog := p.GrowthCounters

	g.Push(engine.PlayCard{Player: p.ID, Card: pb, Paid: engine.CostPaid{}})
	// Unlock with the form leaf; answer the Photon Beam question.
	if pq := g.Pending(); pq != nil {
		_ = g.Answer(pq.Player, []string{"form"})
	}
	for i := 0; i < 8; i++ {
		pq := g.Pending()
		if pq == nil || g.Over {
			break
		}
		if strings.Contains(pq.Question.Prompt, "Photon Beam") {
			_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
			break
		}
		_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
	}
	if g.Villains[villain].HP() != hp-4 {
		t.Fatalf("Photon Beam should deal 4 damage, %d -> %d", hp, g.Villains[villain].HP())
	}
	if p.GrowthCounters != prog+1 {
		t.Fatalf("Photon Beam should add 1 progress counter, %d -> %d", prog, p.GrowthCounters)
	}
}

// TestLevelUpSwapsSide: with 6 progress counters, Level Up! swaps the
// identity to Version 2 and readies her.
func TestLevelUpSwapsSide(t *testing.T) {
	g := newIHGame(t, 162, func(g *engine.Game) {
		g.Players[0].Side = engine.SideHero
		g.Players[0].GrowthCounters = 6
		g.Players[0].Exhausted = true
	})
	p := g.Players[0]
	pq := g.Pending()
	if pq == nil {
		t.Fatal("no pending menu")
	}
	var levelUpID string
	for _, c := range pq.Question.Choices {
		if strings.Contains(c.Label, "Level Up") {
			levelUpID = c.ID
		}
	}
	if levelUpID == "" {
		var labels []string
		for _, c := range pq.Question.Choices {
			labels = append(labels, c.Label)
		}
		t.Fatalf("Level Up! not offered; menu: %v", labels)
	}
	if err := g.Answer(pq.Player, []string{levelUpID}); err != nil {
		t.Fatalf("level up: %v", err)
	}
	if p.GrowthCounters != 0 {
		t.Fatalf("Level Up! should spend all 6 counters, got %d", p.GrowthCounters)
	}
	if p.HeroCode[:5] != "29002" {
		t.Fatalf("should swap to Version 2 (29002), got %s", p.HeroCode)
	}
	if p.Exhausted {
		t.Fatal("Level Up! should ready Ironheart")
	}
}
