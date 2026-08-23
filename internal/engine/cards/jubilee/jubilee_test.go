package jubilee_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/jubilee"
)

func jubileeDeck() map[string]int {
	return map[string]int{
		"47002": 1, "47003": 1, "47004": 1, "47005": 1, "47006": 1,
		"47007a": 3, "47008a": 3, "47009": 1, "47010a": 3,
	}
}

// Contract test: execute Mall Rat directly and verify the deterministic
// deck-to-play message pair without entering the turn menu.
func TestJubileeMallRatFindsShoppingSpree(t *testing.T) {
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed: 83, ScenarioID: "01097",
		Players: []engine.PlayerSpec{{Name: "Jubilee", HeroBase: "47001", Deck: jubileeDeck()}},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	p := g.Players[0]
	p.Side = engine.SideAlterEgo
	spree := engine.Card{ID: g.NextCardID(), Code: "47003", Owner: p.ID}
	p.Deck = engine.CardList{spree}

	b := engine.LookupBehavior("47001")
	if b == nil || b.HeroAbilities == nil {
		t.Fatal("Jubilee should expose Mall Rat")
	}
	abilities := b.HeroAbilities(g, p)
	if len(abilities) != 1 || abilities[0].Execute == nil {
		t.Fatalf("Jubilee alter-ego abilities = %d, want Mall Rat", len(abilities))
	}
	msgs := abilities[0].Execute(g, p.ID)
	if len(msgs) != 2 {
		t.Fatalf("Mall Rat returned %d messages, want 2", len(msgs))
	}
	take, ok := msgs[0].(engine.TakeDeckCard)
	if !ok || take.CardID != spree.ID {
		t.Fatalf("first message = %#v, want Shopping Spree TakeDeckCard", msgs[0])
	}
	play, ok := msgs[1].(engine.PlayCard)
	if !ok || play.Card.ID != spree.ID || play.Player != p.ID {
		t.Fatalf("second message = %#v, want free Shopping Spree PlayCard", msgs[1])
	}
}
