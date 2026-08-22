package nebula_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/nebula"
)

func newNebGame(t *testing.T, seed int64, setup func(g *engine.Game)) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Nebula", HeroBase: "22001", Deck: map[string]int{
				"22010": 1, "22022": 1,
				"22024": 3, "22025": 3, "22026": 3,
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

// TestCombatProtocols: a technique upgrade resolves its Special and is
// discarded when the turn begins.
func TestCombatProtocols(t *testing.T) {
	g := newNebGame(t, 61, nil)
	// Place the technique while the turn menu is pending, so Combat
	// Protocols fires on the NEXT turn start.
	p := g.Players[0]
	weapon := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "22007", Owner: p.ID}
	g.Upgrades[weapon.ID] = weapon
	p.Upgrades = append(p.Upgrades, weapon.ID)
	var villain engine.EntityID
	for id := range g.Villains {
		villain = id
	}
	hp := g.Villains[villain].HP()

	// End the turn: discard prompts then Combat Protocols fire in the
	// next round's PlayerTurnStart.
	pq := g.Pending()
	if err := g.Answer(pq.Player, []string{"end-turn"}); err != nil {
		t.Fatalf("end turn: %v", err)
	}
	for i := 0; i < 10; i++ {
		pq = g.Pending()
		if pq == nil || g.Over {
			break
		}
		if strings.Contains(pq.Question.Prompt, "Weapons Master") {
			_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
			continue
		}
		_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
	}
	if g.Villains[villain].HP() != hp-4 {
		t.Fatalf("Weapons Master Special should deal 4 damage, %d -> %d", hp, g.Villains[villain].HP())
	}
	if g.Upgrades[weapon.ID] != nil {
		t.Fatal("the technique should be discarded after resolving")
	}
}

// TestDaughtersOfThanosDraws3.
func TestDaughtersOfThanosDraws3(t *testing.T) {
	g := newNebGame(t, 62, nil)
	p := g.Players[0]
	dot := engine.Card{ID: g.NextCardID(), Code: "22022", Owner: p.ID}
	p.Hand = append(p.Hand, dot)
	for len(p.Deck) > 3 {
		p.Deck = p.Deck[:3]
	}
	hand := len(p.Hand)
	g.Push(engine.PlayCard{Player: p.ID, Card: dot, Paid: engine.CostPaid{}})
	if pq := g.Pending(); pq != nil {
		_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
	}
	g.Run()
	if len(p.Hand) != hand+3-1 { // played 1, drew 3
		t.Fatalf("Daughters of Thanos should draw 3, hand %d -> %d", hand, len(p.Hand))
	}
}
