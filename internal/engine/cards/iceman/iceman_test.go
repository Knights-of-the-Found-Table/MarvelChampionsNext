package iceman_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/iceman"
)

func icemanDeck() map[string]int {
	return map[string]int{
		"46003": 1, "46004": 1, "46005": 1, "46006": 1, "46007": 2,
		"46008": 1, "46009": 2, "46010": 2, "46011": 2,
	}
}

// Contract test: call Freeze directly and verify that it creates a Frostbite
// entity and emits an attachment message targeting the attacked enemy.
func TestIcemanFreezeAttachesFrostbite(t *testing.T) {
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed: 82, ScenarioID: "01097",
		Players: []engine.PlayerSpec{{Name: "Iceman", HeroBase: "46001", Deck: icemanDeck()}},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	p := g.Players[0]
	p.Side = engine.SideHero
	var target engine.EntityID
	for id := range g.Villains {
		target = id
		break
	}

	b := engine.LookupBehavior("46001")
	if b == nil || b.React == nil {
		t.Fatal("Iceman should expose Freeze through React")
	}
	msgs := b.React(g, p, engine.BasicAttack{Player: p.ID, Target: target, N: 2})
	if len(msgs) == 0 {
		t.Fatal("Freeze should emit an attachment message")
	}
	attach, ok := msgs[0].(engine.AttachUpgrade)
	if !ok || attach.Target != target {
		t.Fatalf("Freeze message = %#v, want AttachUpgrade to %s", msgs[0], target)
	}
	u := g.Upgrades[attach.ID]
	if u == nil || u.Code != "46002" || u.Owner != p.ID {
		t.Fatalf("created upgrade = %#v, want owned Frostbite", u)
	}
}
