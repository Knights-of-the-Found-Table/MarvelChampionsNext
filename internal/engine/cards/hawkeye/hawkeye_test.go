package hawkeye_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/hawkeye"
)

func hawkeyeDeck() map[string]int {
	return map[string]int{
		"04004": 1, "04005": 1, "04006": 1, "04007": 1, "04008": 1,
		"04009": 1, "04010": 1, "04011": 1, "04014": 1, "04019": 1,
	}
}

func newHawkeyeGame(t *testing.T, seed int64) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Hawkeye", HeroBase: "04001", Deck: hawkeyeDeck()},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return g
}

// TestHawkeyeImplemented: the identity is registered.
func TestHawkeyeImplemented(t *testing.T) {
	if !engine.Implemented("04001a") {
		t.Fatal("Hawkeye identity should be implemented")
	}
}

// TestWeaponOfChoiceSearchesBow: in alter-ego form with the bow in the
// deck, the ability offers a Take choice; with no bow anywhere it falls
// back without a question. Contract test, no full game walk.
func TestWeaponOfChoiceSearchesBow(t *testing.T) {
	g := newHawkeyeGame(t, 11)
	p := g.Players[0]
	p.Side = engine.SideAlterEgo

	b := engine.LookupBehavior("04001")
	if b == nil || b.HeroAbilities == nil {
		t.Fatal("Hawkeye identity should expose HeroAbilities")
	}

	// Case 1: bow in deck -> question with a Take choice.
	p.Deck = engine.CardList{{ID: g.NextCardID(), Code: "04002", Owner: p.ID}}
	p.Hand = engine.CardList{}
	p.Discard = engine.CardList{}
	var weapon *engine.Ability
	for i := range b.HeroAbilities(g, p) {
		abs := b.HeroAbilities(g, p)
		if strings.Contains(abs[i].Label.Text, "Weapon of Choice") {
			weapon = &abs[i]
			break
		}
	}
	if weapon == nil {
		t.Fatal("Weapon of Choice ability should be offered")
	}
	if !weapon.AlterEgoOnly {
		t.Fatal("Weapon of Choice should be AlterEgoOnly")
	}
	msgs := weapon.Execute(g, p.ID)
	if len(msgs) != 1 {
		t.Fatalf("Execute returned %d messages, want 1", len(msgs))
	}
	ask, ok := msgs[0].(engine.AskQuestion)
	if !ok {
		t.Fatalf("message = %T, want AskQuestion", msgs[0])
	}
	hasTake := false
	for _, c := range ask.Question.Choices {
		if strings.Contains(c.Label.Text, "Take") {
			hasTake = true
		}
	}
	if !hasTake {
		t.Fatal("Weapon of Choice question should offer a Take choice")
	}

	// Case 2: no bow anywhere -> no question.
	p.Deck = engine.CardList{}
	msgs = weapon.Execute(g, p.ID)
	for _, m := range msgs {
		if _, isAsk := m.(engine.AskQuestion); isAsk {
			t.Fatal("Weapon of Choice without a bow should not ask a question")
		}
	}
}
