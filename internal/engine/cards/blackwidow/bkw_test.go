package blackwidow_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/blackwidow"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
)

func newBkwGame(t *testing.T, seed int64, setup func(g *engine.Game)) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Natasha", HeroBase: "08001", Deck: map[string]int{
				"08003": 2, "08013": 1, "08019": 1,
				"08020": 3, "08021": 3, "08022": 3,
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

// TestCovertOpsConfuses: 4 threat removed and the villain confused.
func TestCovertOpsConfuses(t *testing.T) {
	var villain engine.EntityID
	g := newBkwGame(t, 31, func(g *engine.Game) {
		for id := range g.Villains {
			villain = id
		}
	})
	p := g.Players[0]
	co := engine.Card{ID: g.NextCardID(), Code: "08003", Owner: p.ID}
	p.Hand = append(p.Hand, co)
	threat := g.MainScheme.Threat
	g.Push(engine.PlayCard{Player: p.ID, Card: co, Paid: engine.CostPaid{}})
	for i := 0; i < 8; i++ {
		pq := g.Pending()
		if pq == nil || g.Over {
			break
		}
		if strings.Contains(pq.Question.Prompt, "Covert Ops") {
			_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
			break
		}
		_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
	}
	if g.MainScheme.Threat != max(0, threat-4) {
		t.Fatalf("Covert Ops should remove 4 threat, %d -> %d", threat, g.MainScheme.Threat)
	}
	if !g.Villains[villain].Confused {
		t.Fatal("Covert Ops should confuse the villain")
	}
}

// TestDefensiveStancePrevents: discarding prevents 3 damage.
func TestDefensiveStancePrevents(t *testing.T) {
	g := newBkwGame(t, 32, nil)
	p := g.Players[0]
	ds := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "08032", Owner: p.ID}
	g.Upgrades[ds.ID] = ds
	p.Upgrades = append(p.Upgrades, ds.ID)
	dmg := p.Damage
	g.Push(engine.DamageEntity{Target: p.ID, Damage: 5, Source: engine.EntityID("villain-2")})
	if pq := g.Pending(); pq != nil {
		_ = g.Answer(pq.Player, []string{"form"})
	}
	g.Run()
	if p.Damage != dmg+2 {
		t.Fatalf("Defensive Stance should reduce 5 damage to 2, %d -> %d", dmg, p.Damage)
	}
	if g.Upgrades[ds.ID] != nil {
		t.Fatal("Defensive Stance should be discarded after use")
	}
}
