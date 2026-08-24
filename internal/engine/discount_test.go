package engine

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func discountDef(typ string, cost int, traits ...string) *data.CardDef {
	return &data.CardDef{Code: "test" + typ, Name: "Test " + typ, Type: typ, Cost: &cost, Traits: traits}
}

func newDiscountGame() (*Game, *Player) {
	g := &Game{}
	p := &Player{ID: PlayerID("player-1"), Name: "P1"}
	g.Players = []*Player{p}
	return g, p
}

// TestCostDiscountsStack: Helicarrier's untyped discount (01092) and
// Avengers Tower's Avenger-ally discount (03024) apply to the same card —
// cost reductions from multiple sources are cumulative.
func TestCostDiscountsStack(t *testing.T) {
	g, p := newDiscountGame()
	avenger := discountDef("ally", 3, "avenger")
	p.CostDiscounts = []CostDiscount{
		{Amount: 1}, // Helicarrier: next card costs 1 less
		{Type: "ally", Trait: "avenger", Amount: 1}, // Avengers Tower
	}
	if got := g.costFor(p, avenger); got != 1 {
		t.Fatalf("cost 3 - 1 - 1 = 1, got %d", got)
	}
	g.consumeDiscount(p, avenger)
	if len(p.CostDiscounts) != 0 {
		t.Fatalf("both discounts should be spent by the play, left %+v", p.CostDiscounts)
	}
}

// TestCostDiscountMultipleUntyped: two untyped discounts (two Helicarrier
// uses) stack and are spent together; the total cannot go below 0.
func TestCostDiscountMultipleUntyped(t *testing.T) {
	g, p := newDiscountGame()
	cheap := discountDef("event", 1)
	p.CostDiscounts = []CostDiscount{{Amount: 1}, {Amount: 1}}
	if got := g.costFor(p, cheap); got != 0 {
		t.Fatalf("cost should floor at 0, got %d", got)
	}
	g.consumeDiscount(p, cheap)
	if len(p.CostDiscounts) != 0 {
		t.Fatalf("both discounts should be spent, left %+v", p.CostDiscounts)
	}
}

// TestCostDiscountSelective: a discount whose type/trait filter does not
// match the played card neither applies nor is consumed.
func TestCostDiscountSelective(t *testing.T) {
	g, p := newDiscountGame()
	support := discountDef("support", 2)
	p.CostDiscounts = []CostDiscount{{Type: "ally", Trait: "avenger", Amount: 1}}
	if got := g.costFor(p, support); got != 2 {
		t.Fatalf("non-matching discount should not apply, got %d", got)
	}
	g.consumeDiscount(p, support)
	if len(p.CostDiscounts) != 1 {
		t.Fatalf("non-matching discount should survive, got %+v", p.CostDiscounts)
	}
}

// TestCostDiscountPenalty: negative amounts (Physical Toll 09027: next
// event costs 3 more) raise the cost and are consumed by that play.
func TestCostDiscountPenalty(t *testing.T) {
	g, p := newDiscountGame()
	event := discountDef("event", 2)
	p.CostDiscounts = []CostDiscount{{Type: "event", Amount: -3}}
	if got := g.costFor(p, event); got != 5 {
		t.Fatalf("cost 2 + 3 = 5, got %d", got)
	}
	g.consumeDiscount(p, event)
	if len(p.CostDiscounts) != 0 {
		t.Fatalf("penalty should be spent, left %+v", p.CostDiscounts)
	}
}
