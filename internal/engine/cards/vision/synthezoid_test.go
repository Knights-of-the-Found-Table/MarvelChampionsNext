package vision_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/vision"
)

func newSynGame(t *testing.T, seed int64, scenario string) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: scenario,
		Players:    []engine.PlayerSpec{{Name: "P1", HeroBase: "01001", Deck: map[string]int{"01088": 9, "01089": 9}}},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	for i := 0; i < 12; i++ {
		pq := g.Pending()
		if pq == nil || pq.Question.Prompt == "Your turn" {
			break
		}
		idx := 0
		for j, c := range pq.Question.Choices {
			if c.Then == nil && !c.Disabled {
				idx = j
				break
			}
		}
		_ = g.Answer(pq.Player, []string{pq.Question.Choices[idx].ID})
	}
	return g
}

// TestSheHulkSetup: She-Hulk leads with Superhuman Strength.
func TestSheHulkSetup(t *testing.T) {
	g := newSynGame(t, 91, "57005b")
	var v *engine.Villain
	for _, vv := range g.Villains {
		v = vv
	}
	if v == nil || engine.BaseCodeOf(v.Code) != "57001" {
		t.Fatalf("She-Hulk should lead, got %v", v)
	}
	found := false
	for _, a := range g.Attachments {
		if a != nil && a.Code == "57007" {
			found = true
		}
	}
	if !found {
		t.Fatal("Superhuman Strength should start attached to She-Hulk")
	}
}

// TestVisionLeaderSetup: Vision leads with the Dense attachment.
func TestVisionLeaderSetup(t *testing.T) {
	g := newSynGame(t, 92, "57044b")
	var v *engine.Villain
	for _, vv := range g.Villains {
		v = vv
	}
	if v == nil || engine.BaseCodeOf(v.Code) != "57040" {
		t.Fatalf("Vision should lead, got %v", v)
	}
	found := false
	for _, a := range g.Attachments {
		if a != nil && engine.BaseCodeOf(a.Code) == "57046" {
			found = true
		}
	}
	if !found {
		t.Fatal("the Dense mass form should start attached to Vision")
	}
}

// TestSynthezoidGapCardsRegistered: spot-check.
func TestSynthezoidGapCardsRegistered(t *testing.T) {
	for _, code := range []string{
		"57001", "57002", "57007", "57009", "57011", "57013", "57015",
		"57020", "57027", "57031", "57036", "57040", "57046", "57051",
		"57055", "57060", "57065", "57069", "57072", "57005", "57045",
	} {
		if !engine.Implemented(code) {
			t.Errorf("card %s should be implemented", code)
		}
	}
}
