package doctorstrange_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register core + hero pack content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/doctorstrange"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/hercules"
)

// newSideDeckGame starts a game and answers through the setup prompts (to
// the turn menu). Messages queued by push(g) process while the first setup
// answer resumes the flow — the engine blocks pushed messages behind a
// pending question.
func newSideDeckGame(
	t *testing.T,
	hero string,
	deck map[string]int,
	push func(g *engine.Game),
) (*engine.Game, *engine.Player) {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       90,
		ScenarioID: "01097",
		Players:    []engine.PlayerSpec{{Name: hero, HeroBase: hero, Deck: deck}},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	if push != nil {
		push(g)
	}
	for i := 0; i < 6; i++ {
		pq := g.Pending()
		if pq == nil || pq.Question.Prompt == "Your turn" {
			break
		}
		_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
	}
	return g, g.Players[0]
}

func strangeDeck() map[string]int {
	return map[string]int{
		"09002": 1, "09003": 2, "09004": 2, "09005": 2, "09006": 2,
		"09007": 1, "09008": 1, "09009": 1, "09010": 1, "09011": 1, "09012": 1,
	}
}

func herculesDeck() map[string]int {
	return map[string]int{
		"59008": 1, "59009": 3, "59010": 2, "59011": 2, "59012": 1,
		"59013": 1, "59014": 1, "59015": 1, "59016": 2, "59017": 1,
	}
}

// TestInvocationDeckRecyclesWhenEmpty pins the Invocation rule: discarding
// or resolving the last card shuffles the side discard pile back into the
// side deck with no penalty, so Spell Mastery keeps working all game.
func TestInvocationDeckRecyclesWhenEmpty(t *testing.T) {
	g, p := newSideDeckGame(t, "09001", strangeDeck(), func(g *engine.Game) {
		for i := 0; i < 5; i++ {
			g.Push(engine.SideDeckDiscardTop{Player: g.Players[0].ID})
		}
	})
	if len(p.SenseDeck) != 5 || len(p.SideDiscard) != 0 {
		t.Fatalf("after emptying the deck: deck=%d discard=%d, want refill 5/0", len(p.SenseDeck), len(p.SideDiscard))
	}
	if !strings.Contains(g.LogText(), "shuffles 5 cards from the side deck discard pile") {
		t.Fatalf("missing refill log line: %s", g.LogText())
	}
	// Spell Mastery (offered only while the deck has cards) is still around
	found := false
	for _, ab := range engine.LookupBehavior("09001").HeroAbilities(g, p) {
		if ab.Exhaust { // Spell Mastery exhausts; Natural Talent does not
			found = true
		}
	}
	if !found {
		t.Fatal("Spell Mastery should still be offered after the refill")
	}
}

// TestSideDeckFaceupFlag pins the view-visibility contract: Doctor Strange's
// Invocation deck is faceup with a public discard pile; Hercules' Labor/Gift
// piles reuse the same fields but stay hidden and never recycle.
func TestSideDeckFaceupFlag(t *testing.T) {
	_, sp := newSideDeckGame(t, "09001", strangeDeck(), nil)
	if !engine.SideDeckFaceup(sp) {
		t.Fatal("Invocation deck should be faceup")
	}

	hg, hp := newSideDeckGame(t, "59001", herculesDeck(), func(g *engine.Game) {
		g.Push(engine.SideDeckDiscardTop{Player: g.Players[0].ID})
	})
	_ = hg
	if engine.SideDeckFaceup(hp) {
		t.Fatal("Labor/Gift decks should stay hidden")
	}
	// setup auto-reveals the top Labor into play (3→2); our discard moves
	// one more into the gift pile (2→1 labor, 3→4 gift). A recycle would
	// have shuffled the gift pile back (labor high, gift 0) — it must not.
	if len(hp.SenseDeck) != 1 || len(hp.SideDiscard) != 4 {
		t.Fatalf("hercules after discard: labor=%d gift=%d, want 1/4 (no recycle)", len(hp.SenseDeck), len(hp.SideDiscard))
	}
}
