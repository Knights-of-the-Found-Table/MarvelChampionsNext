package wasp_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/wasp"
)

func newWspGame(t *testing.T, seed int64, setup func(g *engine.Game)) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Nadia", HeroBase: "13001", Deck: map[string]int{
				"13003": 1, "13004": 1, "13031": 1,
				"13021": 3, "13022": 3, "13023": 3,
			}},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	if setup != nil {
		setup(g)
	}
	for i := 0; i < 5; i++ {
		pq := g.Pending()
		if pq == nil || pq.Question.Prompt == "Your turn" {
			break
		}
		_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
	}
	return g
}

// TestFormQuestionAndGiantHelp: the flip asks Giant/Tiny; Giant Help
// removes 4 threat in giant form.
func TestFormQuestionAndGiantHelp(t *testing.T) {
	g := newWspGame(t, 151, nil)
	p := g.Players[0]
	gh := engine.Card{ID: g.NextCardID(), Code: "13003", Owner: p.ID}
	p.Hand = append(p.Hand, gh)
	threat := g.MainScheme.Threat

	// Queue Giant Help behind the flip.
	g.Push(engine.PlayCard{Player: p.ID, Card: gh, Paid: engine.CostPaid{}})
	pq := g.Pending()
	if err := g.Answer(pq.Player, []string{"form"}); err != nil {
		t.Fatalf("flip: %v", err)
	}
	pq = g.Pending()
	if pq == nil || pq.Question.Prompt != "Which hero form?" {
		t.Fatalf("expected the form question, got %q", func() string {
			if pq != nil {
				return pq.Question.Prompt
			}
			return "<none>"
		}())
	}
	if err := g.Answer(pq.Player, []string{"giant"}); err != nil {
		t.Fatalf("choose giant: %v", err)
	}
	if !g.EntityHasTrait(p.ID, "giant") {
		t.Fatal("should carry the giant trait")
	}
	// Giant Help processes: 4 threat in giant form.
	pq = g.Pending()
	if pq == nil || !strings.Contains(pq.Question.Prompt, "Giant Help") {
		t.Fatalf("expected Giant Help question, got %q", func() string {
			if pq != nil {
				return pq.Question.Prompt
			}
			return "<none>"
		}())
	}
	if err := g.Answer(pq.Player, []string{pq.Question.Choices[0].ID}); err != nil {
		t.Fatalf("answer Giant Help: %v", err)
	}
	if g.MainScheme.Threat != max(0, threat-4) {
		t.Fatalf("Giant Help (giant) should remove 4 threat, %d -> %d", threat, g.MainScheme.Threat)
	}
}

// TestRunningInterferenceScalesWithStage.
func TestRunningInterferenceScalesWithStage(t *testing.T) {
	g := newWspGame(t, 152, func(g *engine.Game) {
		for _, v := range g.Villains {
			v.Stage = 2 // 2 + 2 = 4 threat
		}
	})
	p := g.Players[0]
	ri := engine.Card{ID: g.NextCardID(), Code: "13031", Owner: p.ID}
	p.Hand = append(p.Hand, ri)
	threat := g.MainScheme.Threat
	g.Push(engine.PlayCard{Player: p.ID, Card: ri, Paid: engine.CostPaid{}})
	// Unlock with the form leaf (a plain choice whose answer run drains
	// the queue), then settle.
	if pq := g.Pending(); pq != nil {
		if err := g.Answer(pq.Player, []string{"form"}); err != nil {
			t.Fatalf("unblock: %v", err)
		}
	}
	// Answer the form question (giant).
	if pq := g.Pending(); pq != nil && pq.Question.Prompt == "Which hero form?" {
		_ = g.Answer(pq.Player, []string{"tiny"})
	}
	g.Run()
	if g.MainScheme.Threat != max(0, threat-4) {
		t.Fatalf("Running Interference should remove 4 threat at stage 2, %d -> %d", threat, g.MainScheme.Threat)
	}
}
