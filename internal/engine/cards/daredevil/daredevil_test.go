package daredevil_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/daredevil"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/extras"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/goblinfooblin"
)

// deck63988 mirrors marvelcdb decklist #63988 "The Defense Rests, Your
// Honor" (Daredevil / Protection).
func deck63988() map[string]int {
	return map[string]int{
		"01079": 2,                         // The Power of Protection
		"01081": 1,                         // Armored Vest
		"01088": 1, "01089": 1, "01090": 1, // basic resources
		"09020": 1, // Unflappable
		"16024": 1, // Deft Focus
		"32014": 2, // Powerful Punch
		"40020": 1, // Establish Perimeter
		"42017": 1, // Render Medical Aid
		"48014": 1, // Change of Fortune
		"48015": 3, // Under Control
		"56046": 1, // Defensive Conditioning
		"60007": 1, // Elektra
		"60008": 2, // Cross-Examination
		"60009": 2, // Deposition
		"60010": 2, // Living Lie Detector
		"60011": 1, // Raising Hell
		"60012": 1, // Focus the Senses
		"60013": 1, // Foggy Nelson
		"60014": 1, // Karen Page
		"60015": 1, // Nelson and Murdock
		"60016": 1, // Sister Maggie
		"60017": 1, // Daredevil's Billy Club
		"60018": 1, // The Man Without Fear
		"60030": 1, // Contingency Planning
		"60038": 1, // Innate Reflexes
		"60048": 3, // Army of One
		"60050": 3, // In Harm's Way
		"60052": 3, // The Best Offense...
		"60053": 1, // Ronin
		"60054": 1, // Stand Alone
	}
}

func newDaredevilGame(t *testing.T, scenario string, seed int64) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: scenario,
		Players: []engine.PlayerSpec{
			{Name: "Matt", HeroBase: "60001", Deck: deck63988()},
		},
	})
	if err != nil {
		t.Fatalf("NewGame(%s): %v", scenario, err)
	}
	return g
}

func pickDefault(q *engine.Question) []string {
	prefer := []string{"pass-interrupt", "skip", "take", "end-turn", "continue"}
	for _, id := range prefer {
		for _, c := range q.Choices {
			if c.ID == id && !c.Disabled {
				return []string{id}
			}
		}
	}
	if q.Type == "choose_n" {
		var out []string
		for i, c := range q.Choices {
			if q.N > 0 && i >= q.N {
				break
			}
			out = append(out, c.ID)
		}
		return out
	}
	if len(q.Choices) > 0 {
		return []string{q.Choices[0].ID}
	}
	return nil
}

func drive(t *testing.T, g *engine.Game, maxAnswers int) {
	t.Helper()
	for i := 0; i < maxAnswers; i++ {
		pq := g.Pending()
		if pq == nil {
			return
		}
		if err := g.Answer(pq.Player, pickDefault(pq.Question)); err != nil {
			t.Fatalf("answer %q: %v", pq.Question.Prompt, err)
		}
		if g.Over {
			return
		}
	}
}

func TestDaredevilImplemented(t *testing.T) {
	if !engine.Implemented("60001a") {
		t.Fatal("Daredevil should count as implemented")
	}
}

// TestSenseDeckBuilt: Matt Murdock starts with the 5-card Sense deck.
func TestSenseDeckBuilt(t *testing.T) {
	g := newDaredevilGame(t, "01097", 3)
	p := g.Players[0]
	if len(p.SenseDeck) != 5 {
		t.Fatalf("Sense deck should have 5 cards, got %d (%v)", len(p.SenseDeck), p.SenseDeck.Codes())
	}
	for _, c := range p.SenseDeck {
		if !c.Def().HasTrait("sense") {
			t.Fatalf("non-sense card %s in the Sense deck", c.Code)
		}
	}
}

// TestEveryDeckCardPlayable checks that each card in the reference
// decklist either has a hand-written behavior or works purely from the
// data layer (resource cards).
func TestEveryDeckCardPlayable(t *testing.T) {
	plainData := map[string]bool{
		"01088": true, "01089": true, "01090": true, // resources
	}
	for code := range deck63988() {
		if plainData[code] {
			continue
		}
		if !engine.Implemented(code) {
			t.Errorf("deck card %s has no hand-written behavior", code)
		}
	}
}

func TestDeck63988VersusRhino(t *testing.T) {
	g := newDaredevilGame(t, "01097", 11)
	drive(t, g, 900)
	if !g.Over {
		t.Fatalf("game did not end, round=%d", g.Round)
	}
	t.Logf("outcome: won=%v reason=%q rounds=%d", g.Won, g.Reason.Text, g.Round)
}

func TestDeck63988VersusMutagenFormula(t *testing.T) {
	g := newDaredevilGame(t, "02017", 29)
	drive(t, g, 900)
	if !g.Over {
		t.Fatalf("game did not end, round=%d", g.Round)
	}
	t.Logf("outcome: won=%v reason=%q rounds=%d", g.Won, g.Reason.Text, g.Round)
}

// TestSenseCardsCycle: after a Sense upgrade triggers, it returns to the
// bottom of the Sense deck instead of the discard pile.
func TestSenseCardsCycle(t *testing.T) {
	g := newDaredevilGame(t, "01097", 5)
	p := g.Players[0]
	// Put Superior Taste into play attached to the main scheme.
	u := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "60006", Owner: p.ID, AttachTo: g.MainScheme.ID}
	g.Upgrades[u.ID] = u
	p.Upgrades = append(p.Upgrades, u.ID)
	senseBefore := len(p.SenseDeck)

	g.Push(engine.WindowAfterThwarted{Player: p.ID, Scheme: g.MainScheme.ID})
	if pq := g.Pending(); pq != nil {
		if err := g.Answer(pq.Player, pickDefault(pq.Question)); err != nil {
			t.Fatalf("answer: %v", err)
		}
	} else {
		g.Run()
	}
	if len(p.SenseDeck) != senseBefore+1 {
		t.Fatalf("Superior Taste should have returned to the Sense deck, size %d -> %d", senseBefore, len(p.SenseDeck))
	}
	if !strings.Contains(g.LogText(), "Superior Taste") {
		t.Fatal("Superior Taste never triggered")
	}
}
