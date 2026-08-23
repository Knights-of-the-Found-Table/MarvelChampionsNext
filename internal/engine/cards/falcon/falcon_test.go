package falcon_test

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/falcon"
	"testing"
)

func falconDeck() map[string]int {
	return map[string]int{"53002": 1, "53003": 2, "53004": 2, "53005": 2, "53006": 1, "53007": 1, "53008": 1, "53009": 1, "53010": 1, "53011": 1, "53012": 1, "53013": 1}
}
func TestEagleEyedDirectReactionContract(t *testing.T) {
	g, err := engine.NewGame(engine.NewGameOptions{Seed: 53, ScenarioID: "01097", Players: []engine.PlayerSpec{{Name: "Falcon", HeroBase: "53001", Deck: falconDeck()}}})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	p := g.Players[0]
	b := engine.LookupBehavior("53001")
	if b == nil || b.React == nil {
		t.Fatal("Falcon should expose Eagle-Eyed React")
	}
	card := engine.Card{ID: g.NextCardID(), Code: "53003", Owner: p.ID}
	if !card.Def().HasTrait("aerial") {
		t.Fatal("contract fixture must be Aerial")
	}
	msgs := b.React(g, p, engine.PlayCard{Player: p.ID, Card: card})
	if len(msgs) != 2 {
		t.Fatalf("Eagle-Eyed returned %d messages, want mill + icon signal", len(msgs))
	}
	if mill, ok := msgs[0].(engine.MillEncounter); !ok || mill.N != 1 {
		t.Fatalf("first message = %#v, want MillEncounter 1", msgs[0])
	}
}
