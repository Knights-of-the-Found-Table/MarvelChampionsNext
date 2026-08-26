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

// TestPowerInAllOfUsDoublesForBasic: The Power in All of Us (13024) pays 2
// when the card being paid for is Basic (gray) and only 1 otherwise, end
// to end through the payment question's validateSelection branch.
func TestPowerInAllOfUsDoublesForBasic(t *testing.T) {
	basic := Card{ID: "card-basic", Code: "01093"}   // Tenacity: basic upgrade, cost 2
	aspect := Card{ID: "card-aspect", Code: "01053"} // Relentless Assault: aggression event, cost 2
	power := Card{ID: "card-power", Code: "13024"}
	if bd := DB.MustLookup(basic.Code); bd.Aspect != "basic" || bd.Cost == nil || *bd.Cost != 2 {
		t.Fatalf("fixture drift: 01093 = %s cost %v", bd.Aspect, bd.Cost)
	}
	if ad := DB.MustLookup(aspect.Code); ad.Aspect != "aggression" || ad.Cost == nil || *ad.Cost != 2 {
		t.Fatalf("fixture drift: 01053 = %s cost %v", ad.Aspect, ad.Cost)
	}

	payWithPowerAlone := func(target Card, cost int) error {
		g, p := newDiscountGame()
		p.Hand = CardList{target, power}
		q := g.paymentQuestion(p, target, cost)
		q.assignIDs("")
		var path string
		for i := range q.Choices {
			if q.Choices[i].CardCode == power.Code {
				path = q.Choices[i].ID
			}
		}
		if path == "" {
			t.Fatal("13024 not offered as a payment choice")
		}
		_, err := g.resolveChooseN(q, []string{path})
		return err
	}

	if err := payWithPowerAlone(basic, 2); err != nil {
		t.Fatalf("13024 alone should cover a 2-cost basic card (doubled to 2): %v", err)
	}
	if err := payWithPowerAlone(aspect, 2); err == nil {
		t.Fatal("13024 should count 1 toward an aggression card: expected a payment error")
	}
}
