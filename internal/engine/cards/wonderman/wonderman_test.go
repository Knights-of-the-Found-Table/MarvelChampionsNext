package wonderman_test

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/wonderman"
	"testing"
)

func deck() map[string]int {
	return map[string]int{"58003": 3, "58004": 2, "58005": 3, "58006": 2, "58007": 1, "58008": 1, "58009": 1, "58010": 1, "58011": 1}
}
func TestPowerRecyclingIonicAttackContract(t *testing.T) {
	g, err := engine.NewGame(engine.NewGameOptions{Seed: 58, ScenarioID: "01097", Players: []engine.PlayerSpec{{Name: "Wonder Man", HeroBase: "58001", Deck: deck()}}})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	p := g.Players[0]
	p.Side = engine.SideHero
	u := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "58002", Owner: p.ID}
	g.Upgrades[u.ID] = u
	p.Upgrades = append(p.Upgrades, u.ID)
	p.SenseDeck = engine.CardList{{ID: g.NextCardID(), Code: "58003", Owner: p.ID}, {ID: g.NextCardID(), Code: "58004", Owner: p.ID}}
	if got := p.AttackStat(g); got != 3 {
		t.Fatalf("ATK with two tucked cards = %d, want printed 1 + 2", got)
	}
	b := engine.LookupBehavior("58001")
	if b == nil || b.React == nil {
		t.Fatal("Wonder Man should expose Power Recycling React")
	}
	b.React(g, p, engine.BasicAttack{Player: p.ID, N: p.AttackStat(g), Target: g.ActiveVillain})
	if len(p.SenseDeck) != 0 {
		t.Fatalf("tucked cards after basic attack = %d, want 0", len(p.SenseDeck))
	}
}
