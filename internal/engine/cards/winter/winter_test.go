package winter_test

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/winter"
	"testing"
)

func deck() map[string]int {
	return map[string]int{"54002": 1, "54003": 1, "54004": 2, "54005": 3, "54006": 2, "54007": 1, "54008": 2, "54009": 1, "54010": 1, "54011": 1}
}
func TestLethalProtectorDirectReactionContract(t *testing.T) {
	g, err := engine.NewGame(engine.NewGameOptions{Seed: 54, ScenarioID: "01097", Players: []engine.PlayerSpec{{Name: "Bucky", HeroBase: "54001", Deck: deck()}}})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	p := g.Players[0]
	id := g.NextEntityID("minion")
	g.Minions[id] = &engine.Minion{ID: id, Code: "01104", MaxHP: 2, EngagedWith: p.ID}
	b := engine.LookupBehavior("54001")
	if b == nil || b.React == nil {
		t.Fatal("Winter Soldier should expose Lethal Protector")
	}
	msgs := b.React(g, p, engine.DamageEntity{Target: id, Damage: 2, Source: p.ID})
	if len(msgs) != 1 {
		t.Fatalf("Lethal Protector returned %d messages, want scheme question", len(msgs))
	}
	ask, ok := msgs[0].(engine.AskQuestion)
	if !ok || ask.Question == nil || len(ask.Question.Choices) == 0 {
		t.Fatalf("message = %#v, want scheme choices", msgs[0])
	}
}
