package aoa_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

func magikDeck() map[string]int {
	return map[string]int{
		"45031": 1, "45032": 1, "45033": 1, "45034": 1, "45035": 1,
		"45036": 2, "45037": 2, "45038": 2, "45039": 2, "45040": 2,
	}
}

// Contract test: execute the top-deck ability directly and verify that the
// engine receives a deck-to-hand move followed by the one-cost discount.
func TestMagikPlaysTopCardAtDiscount(t *testing.T) {
	g := newAOAGame(t, "45030", magikDeck())
	p := g.Players[0]
	p.Side = engine.SideHero
	top := engine.Card{ID: g.NextCardID(), Code: "45039", Owner: p.ID}
	p.Deck = engine.CardList{top}

	b := engine.LookupBehavior("45030")
	if b == nil || b.HeroAbilities == nil {
		t.Fatal("Magik should expose the top-deck HeroAbility")
	}
	abilities := b.HeroAbilities(g, p)
	if len(abilities) != 1 || abilities[0].Execute == nil {
		t.Fatalf("Magik abilities = %d, want one executable ability", len(abilities))
	}
	msgs := abilities[0].Execute(g, p.ID)
	if len(msgs) != 2 {
		t.Fatalf("top-deck ability returned %d messages, want 2", len(msgs))
	}
	take, ok := msgs[0].(engine.TakeDeckCard)
	if !ok || take.CardID != top.ID || take.Player != p.ID {
		t.Fatalf("first message = %#v, want TakeDeckCard for top card", msgs[0])
	}
	discount, ok := msgs[1].(engine.CostDiscountApply)
	if !ok || discount.Player != p.ID || discount.Amount != 1 {
		t.Fatalf("second message = %#v, want one-cost discount", msgs[1])
	}
}
