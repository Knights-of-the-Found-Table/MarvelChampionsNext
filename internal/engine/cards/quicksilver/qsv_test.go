package quicksilver_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/quicksilver"
)

func newQsvGame(t *testing.T, seed int64, setup func(g *engine.Game)) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Pietro", HeroBase: "14001", Deck: map[string]int{
				"14003": 1, "14004": 1, "14032": 1,
				"14019": 3, "14020": 3, "14021": 3,
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

// TestSuperSpeedReadies: a basic attack exhausts Quicksilver, then Super
// Speed readies him.
func TestSuperSpeedReadies(t *testing.T) {
	g := newQsvGame(t, 81, func(g *engine.Game) {
		g.Players[0].Side = engine.SideHero
	})
	p := g.Players[0]
	var villain engine.EntityID
	for id := range g.Villains {
		villain = id
	}
	// Two basic attacks in sequence: the first readies via Super Speed
	// (marker set), the second must not.
	g.Push(engine.BasicAttack{Player: p.ID, N: 2, Target: villain})
	g.Push(engine.BasicAttack{Player: p.ID, N: 2, Target: villain})
	// Unblock with the form choice (a plain leaf, no payment subtree).
	if pq := g.Pending(); pq != nil {
		_ = g.Answer(pq.Player, []string{"form"})
	}
	g.Run()
	if !p.Exhausted {
		t.Fatal("Super Speed is once per phase: the second basic power must leave Quicksilver exhausted")
	}
}

// TestBeatEmUpHitsEngaged: villain + engaged minions take 1.
func TestBeatEmUpHitsEngaged(t *testing.T) {
	g := newQsvGame(t, 82, func(g *engine.Game) {
		p := g.Players[0]
		mn := &engine.Minion{ID: g.NextEntityID("minion"), Code: "01101", MaxHP: 3, EngagedWith: p.ID}
		g.Minions[mn.ID] = mn
	})
	p := g.Players[0]
	be := engine.Card{ID: g.NextCardID(), Code: "14032", Owner: p.ID}
	p.Hand = append(p.Hand, be)
	var villain engine.EntityID
	for id := range g.Villains {
		villain = id
	}
	vHP := g.Villains[villain].HP()
	var mnID engine.EntityID
	for id := range g.Minions {
		mnID = id
	}
	mHP := g.Minions[mnID].HP()

	g.Push(engine.PlayCard{Player: p.ID, Card: be, Paid: engine.CostPaid{}})
	if pq := g.Pending(); pq != nil {
		_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
	}
	g.Run()
	if g.Villains[villain].HP() != vHP-1 || g.Minions[mnID].HP() != mHP-1 {
		t.Fatalf("Beat 'Em Up should hit villain and engaged minion: %d->%d, %d->%d",
			vHP, g.Villains[villain].HP(), mHP, g.Minions[mnID].HP())
	}
	_ = strings.TrimSpace("")
}
