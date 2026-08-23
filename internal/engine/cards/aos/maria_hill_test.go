package aos_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/aos"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
)

func mariaDeck() map[string]int {
	return map[string]int{
		"50002": 1, "50003": 2, "50004": 2, "50005": 3,
		"50006": 1, "50007": 2, "50008": 1, "50009": 1,
		"50010": 1, "50011": 1,
	}
}

func newAOSGame(t *testing.T, hero string, deck map[string]int) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed: 71, ScenarioID: "01097",
		Players: []engine.PlayerSpec{{Name: "AOS Hero", HeroBase: hero, Deck: deck}},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return g
}

func TestMariaHillImplemented(t *testing.T) {
	if !engine.Implemented("50001a") {
		t.Fatal("Maria Hill should count as implemented")
	}
}

// Contract test: call the identity hook and Execute directly. This avoids
// the recurring "Your turn" question while proving the alter-ego search
// emits a deterministic AskQuestion naming the available support.
func TestMariaHillSearchesForShieldSupport(t *testing.T) {
	g := newAOSGame(t, "50001", mariaDeck())
	p := g.Players[0]
	p.Side = engine.SideAlterEgo
	p.Deck = engine.CardList{{ID: g.NextCardID(), Code: "50008", Owner: p.ID}}

	b := engine.LookupBehavior("50001")
	if b == nil || b.HeroAbilities == nil {
		t.Fatal("Maria Hill should expose HeroAbilities")
	}
	abilities := b.HeroAbilities(g, p)
	if len(abilities) != 2 || abilities[1].Execute == nil {
		t.Fatalf("Maria Hill abilities = %d, want 2 with executable search", len(abilities))
	}
	if !abilities[1].AlterEgoOnly || !abilities[1].Exhaust {
		t.Fatal("Maria Hill's support search should be alter-ego-only and exhaust her")
	}

	msgs := abilities[1].Execute(g, p.ID)
	if len(msgs) != 1 {
		t.Fatalf("search returned %d messages, want 1", len(msgs))
	}
	ask, ok := msgs[0].(engine.AskQuestion)
	if !ok {
		t.Fatalf("search message = %T, want AskQuestion", msgs[0])
	}
	if !strings.Contains(ask.Question.Prompt, "S.H.I.E.L.D.") {
		t.Fatalf("search prompt = %q", ask.Question.Prompt)
	}
	if len(ask.Question.Choices) != 1 || ask.Question.Choices[0].CardCode != "50008" {
		t.Fatalf("search choices = %#v, want Support Staff", ask.Question.Choices)
	}
}
