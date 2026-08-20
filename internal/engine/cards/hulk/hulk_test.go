package hulk_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/hulk"
)

// hulkDeck is a minimal legal-ish Hulk deck for engine tests.
func hulkDeck() map[string]int {
	return map[string]int{
		"10002": 1, "10003": 1, "10004": 1, "10005": 1, "10006": 1,
		"10008": 1, "10010": 1, "10014": 1, "10015": 1, "10019": 1,
	}
}

func newHulkGame(t *testing.T, scenario string, seed int64) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: scenario,
		Players: []engine.PlayerSpec{
			{Name: "Hulk", HeroBase: "10001", Deck: hulkDeck()},
		},
	})
	if err != nil {
		t.Fatalf("NewGame(%s): %v", scenario, err)
	}
	return g
}

// TestHulkImplemented: the identity is registered.
func TestHulkImplemented(t *testing.T) {
	if !engine.Implemented("10001a") {
		t.Fatal("Hulk identity should be implemented")
	}
}

// TestExperimentalResearchOffersChoices: in alter-ego form with cards in
// hand, the ability offers a draw+discard question; with an empty hand it
// just draws. Contract test, no full game walk.
func TestExperimentalResearchOffersChoices(t *testing.T) {
	g := newHulkGame(t, "01097", 7)
	p := g.Players[0]
	p.Side = engine.SideAlterEgo

	b := engine.LookupBehavior("10001")
	if b == nil || b.HeroAbilities == nil {
		t.Fatal("Hulk identity should expose HeroAbilities")
	}
	abs := b.HeroAbilities(g, p)
	if len(abs) == 0 || abs[0].Execute == nil {
		t.Fatal("Experimental Research should be the first HeroAbility")
	}
	if !abs[0].AlterEgoOnly || !abs[0].OncePerRound {
		t.Fatal("Experimental Research should be AlterEgoOnly and OncePerRound")
	}

	// Case 1: hand has cards -> question.
	p.Hand = engine.CardList{
		{ID: g.NextCardID(), Code: "10014", Owner: p.ID},
		{ID: g.NextCardID(), Code: "10019", Owner: p.ID},
	}
	msgs := abs[0].Execute(g, p.ID)
	if len(msgs) != 1 {
		t.Fatalf("Execute returned %d messages, want 1", len(msgs))
	}
	ask, ok := msgs[0].(engine.AskQuestion)
	if !ok {
		t.Fatalf("message = %T, want AskQuestion", msgs[0])
	}
	if !strings.Contains(ask.Question.Prompt, "Experimental Research") {
		t.Fatalf("prompt = %q", ask.Question.Prompt)
	}
	if len(ask.Question.Choices) != 2 {
		t.Fatalf("choices = %d, want 2 (one per hand card)", len(ask.Question.Choices))
	}

	// Case 2: empty hand -> plain draw.
	p.Hand = engine.CardList{}
	msgs = abs[0].Execute(g, p.ID)
	if len(msgs) != 1 {
		t.Fatalf("Execute (empty hand) returned %d messages, want 1", len(msgs))
	}
	draw, ok := msgs[0].(engine.DrawCards)
	if !ok || draw.N != 1 {
		t.Fatalf("message = %#v, want DrawCards{N:1}", msgs[0])
	}
}

// TestEnragedDiscardsHand: at the end of a hero-form turn, Hulk discards
// his hand; in alter-ego form nothing happens. Contract test via the
// React hook.
func TestEnragedDiscardsHand(t *testing.T) {
	g := newHulkGame(t, "01097", 7)
	p := g.Players[0]

	b := engine.LookupBehavior("10001")
	if b == nil || b.React == nil {
		t.Fatal("Hulk identity should expose React")
	}
	p.Side = engine.SideHero
	p.Hand = engine.CardList{
		{ID: g.NextCardID(), Code: "10014", Owner: p.ID},
		{ID: g.NextCardID(), Code: "10015", Owner: p.ID},
	}
	msgs := b.React(g, p, engine.PlayerTurnEnd{Player: p.ID})
	if len(msgs) != 1 {
		t.Fatalf("Enraged returned %d messages, want 1", len(msgs))
	}
	dc, ok := msgs[0].(engine.DiscardCards)
	if !ok || len(dc.Cards) != 2 {
		t.Fatalf("message = %#v, want DiscardCards with 2 cards", msgs[0])
	}

	// Alter-ego form: no discard.
	p.Side = engine.SideAlterEgo
	if msgs := b.React(g, p, engine.PlayerTurnEnd{Player: p.ID}); len(msgs) != 0 {
		t.Fatalf("Enraged in alter-ego form returned %d messages, want 0", len(msgs))
	}
}

// TestImmovableObjectGrantsRetaliate: contract test for the upgrade hook.
func TestImmovableObjectGrantsRetaliate(t *testing.T) {
	b := engine.LookupBehavior("10010")
	if b == nil || b.IdentityStats == nil {
		t.Fatal("Immovable Object should expose IdentityStats")
	}
	bonus := b.IdentityStats(nil)
	if bonus.Retaliate != 1 {
		t.Fatalf("Retaliate = %d, want 1", bonus.Retaliate)
	}
}

// TestBannerLaboratoryResource: contract test for the resource hook.
func TestBannerLaboratoryResource(t *testing.T) {
	b := engine.LookupBehavior("10008")
	if b == nil || b.Resource == nil {
		t.Fatal("Banner's Laboratory should expose a Resource ability")
	}
	if b.Resource.Icon != "mental" {
		t.Fatalf("icon = %q, want mental", b.Resource.Icon)
	}
}
