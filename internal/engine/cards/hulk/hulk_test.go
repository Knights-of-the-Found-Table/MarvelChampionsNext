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

// answerUntil answers prompts with a simple policy until a prompt
// containing want appears (answered with its first choice) and resolution
// settles.
func answerUntil(t *testing.T, g *engine.Game, want string, max int) {
	t.Helper()
	for i := 0; i < max; i++ {
		pq := g.Pending()
		if pq == nil {
			g.Run()
			if g.Pending() == nil || g.Over {
				return
			}
			pq = g.Pending()
		}
		if g.Over {
			return
		}
		if strings.Contains(pq.Question.Prompt, want) {
			if err := g.Answer(pq.Player, []string{pq.Question.Choices[0].ID}); err != nil {
				t.Fatalf("answer target: %v", err)
			}
			g.Run()
			return
		}
		var ans []string
		if len(pq.Question.Choices) > 0 {
			ans = []string{pq.Question.Choices[0].ID}
		}
		if ans == nil {
			return
		}
		if err := g.Answer(pq.Player, ans); err != nil {
			return
		}
	}
}

// drain answers prompts with a simple policy until the queue settles.
func drain(t *testing.T, g *engine.Game, max int) {
	t.Helper()
	for i := 0; i < max; i++ {
		pq := g.Pending()
		if pq == nil {
			g.Run()
			if g.Pending() == nil || g.Over {
				return
			}
			pq = g.Pending()
		}
		if g.Over {
			return
		}
		prefer := []string{"pass-interrupt", "skip", "continue", "keep", "take", "end-turn"}
		var ans []string
		for _, id := range prefer {
			for _, c := range pq.Question.Choices {
				if c.ID == id && !c.Disabled {
					ans = []string{id}
					break
				}
			}
			if ans != nil {
				break
			}
		}
		if ans == nil && pq.Question.Type == "choose_n" {
			n := pq.Question.N
			if n == 0 {
				n = 1
			}
			for j := 0; j < n && j < len(pq.Question.Choices); j++ {
				ans = append(ans, pq.Question.Choices[j].ID)
			}
		}
		if ans == nil && len(pq.Question.Choices) > 0 {
			ans = []string{pq.Question.Choices[0].ID}
		}
		if ans == nil {
			return
		}
		if err := g.Answer(pq.Player, ans); err != nil {
			return
		}
	}
}

// keepHand answers the setup mulligan if pending.
func keepHand(t *testing.T, g *engine.Game) {
	t.Helper()
	if pq := g.Pending(); pq != nil && pq.Question.Prompt == "Mulligan?" {
		if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
			t.Fatalf("keep: %v", err)
		}
	}
}

// TestDropKickStunsOnPhysical: paying only physical resources adds the
// stun + draw riders.
func TestDropKickStunsOnPhysical(t *testing.T) {
	g := newHulkGame(t, "01097", 41)
	keepHand(t, g)
	p := g.Players[0]
	p.Side = engine.SideHero
	dk := engine.Card{ID: g.NextCardID(), Code: "10014", Owner: p.ID}
	for len(p.Deck) > 1 {
		p.Deck = p.Deck[:1]
	}
	p.Hand = append(p.Hand, dk)
	hand := len(p.Hand)

	var villain engine.EntityID
	for id := range g.Villains {
		villain = id
	}
	v := g.Villains[villain]

	g.Push(engine.PlayCard{Player: p.ID, Card: dk,
		Paid: engine.CostPaid{Icons: []string{"physical", "physical", "physical"}}})
	answerUntil(t, g, "Drop Kick", 12)
	if v.Damage < 4 {
		t.Fatalf("Drop Kick should deal 4 damage, got %d", v.Damage)
	}
	if !v.Stunned {
		t.Fatal("villain should be stunned when paid only with physical resources")
	}
	if len(p.Hand) != hand+1-1 { // played card left hand, rider drew 1
		t.Fatalf("rider should draw 1 card, hand %d -> %d", hand, len(p.Hand))
	}
}

// TestToTheRescueRemovesThreat.
func TestToTheRescueRemovesThreat(t *testing.T) {
	g := newHulkGame(t, "01097", 42)
	keepHand(t, g)
	p := g.Players[0]
	tr := engine.Card{ID: g.NextCardID(), Code: "10019", Owner: p.ID}
	p.Hand = append(p.Hand, tr)
	if g.MainScheme == nil {
		t.Fatal("expected main scheme")
	}
	threat := g.MainScheme.Threat
	g.Push(engine.PlayCard{Player: p.ID, Card: tr, Paid: engine.CostPaid{}})
	answerUntil(t, g, "To the Rescue —", 12)
	if g.MainScheme.Threat != max(0, threat-2) {
		t.Fatalf("To the Rescue should remove 2 threat, %d -> %d", threat, g.MainScheme.Threat)
	}
}
