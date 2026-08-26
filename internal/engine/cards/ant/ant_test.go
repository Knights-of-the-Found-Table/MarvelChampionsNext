package ant_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register core + ant content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/ant"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
)

func newAntGame(t *testing.T, seed int64, setup func(g *engine.Game)) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Scott", HeroBase: "12001", Deck: map[string]int{
				"12003": 1, "12005": 1, "12019": 1, "12031": 1,
				"12021": 3, "12022": 3, "12023": 3,
			}},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	if setup != nil {
		setup(g)
	}
	// Setup ability + mulligan prompts, stopping at the turn menu.
	for i := 0; i < 6; i++ {
		pq := g.Pending()
		if pq == nil || pq.Question.Prompt == "Your turn" {
			break
		}
		_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
	}
	return g
}

// TestAntManFormChoice: changing to hero form asks Giant/Tiny and grants
// the trait.
func TestAntManFormChoice(t *testing.T) {
	g := newAntGame(t, 21, nil)
	p := g.Players[0]
	p.Side = engine.SideHero // stale-proofing; the flip below is manual
	p.Side = engine.SideAlterEgo

	pq := g.Pending()
	if pq == nil || pq.Question.Prompt != "Your turn" {
		t.Fatal("expected the turn menu")
	}
	// Flip to hero via the form choice.
	if err := g.Answer(pq.Player, []string{"form"}); err != nil {
		t.Fatalf("flip: %v", err)
	}
	pq = g.Pending()
	if pq == nil || pq.Question.Prompt != "Which hero form?" {
		t.Fatalf("expected the form question, got %q", promptOf(pq))
	}
	var giantPath string
	for _, c := range pq.Question.Choices {
		if c.ID == "giant" {
			giantPath = c.ID
		}
	}
	if err := g.Answer(pq.Player, []string{giantPath}); err != nil {
		t.Fatalf("choose giant: %v", err)
	}
	if !p.IsHero() || !g.EntityHasTrait(p.ID, "giant") {
		t.Fatal("player should be in hero form with the giant trait")
	}
	// The Giant Nuisance rider question follows.
	if pq = g.Pending(); pq == nil || !strings.Contains(pq.Question.Prompt, "Giant Nuisance") {
		t.Fatalf("expected Giant Nuisance damage question, got %q", promptOf(pq))
	}
	var villain engine.EntityID
	for id := range g.Villains {
		villain = id
	}
	hp := g.Villains[villain].HP()
	if err := g.Answer(pq.Player, []string{pq.Question.Choices[0].ID}); err != nil {
		t.Fatalf("nuisance: %v", err)
	}
	if g.Villains[villain].HP() != hp-1 {
		t.Fatalf("Giant Nuisance should deal 1 damage, %d -> %d", hp, g.Villains[villain].HP())
	}
}

// TestLayDownTheLawRemovesThreat.
func TestLayDownTheLawRemovesThreat(t *testing.T) {
	g := newAntGame(t, 22, nil)
	p := g.Players[0]
	ldl := engine.Card{ID: g.NextCardID(), Code: "12031", Owner: p.ID}
	p.Hand = append(p.Hand, ldl)
	threat := g.MainScheme.Threat
	g.Push(engine.PlayCard{Player: p.ID, Card: ldl, Paid: engine.CostPaid{}})
	for i := 0; i < 8; i++ {
		pq := g.Pending()
		if pq == nil || g.Over {
			break
		}
		if strings.Contains(pq.Question.Prompt, "Lay Down the Law") {
			if err := g.Answer(pq.Player, []string{pq.Question.Choices[0].ID}); err != nil {
				t.Fatalf("ldl: %v", err)
			}
			break
		}
		if err := g.Answer(pq.Player, []string{pq.Question.Choices[0].ID}); err != nil {
			break
		}
	}
	if g.MainScheme.Threat != max(0, threat-3) {
		t.Fatalf("Lay Down the Law should remove 3 threat, %d -> %d", threat, g.MainScheme.Threat)
	}
}

func promptOf(pq *engine.PendingQuestion) string {
	if pq == nil {
		return "<none>"
	}
	return pq.Question.Prompt
}

// TestPowerGlovesOnlyAttachesToAvengers: Mockingbird (01083) is not an
// Avenger in print, while Blade (21019) is; the attach question must offer
// Blade but not Mockingbird. Honorary Avenger's dynamic ExtraTraits are
// included by the helper.
func TestPowerGlovesOnlyAttachesToAvengers(t *testing.T) {
	g := newAntGame(t, 23, nil)
	p := g.Players[0]
	mock := &engine.Ally{ID: engine.EntityID(g.NextCardID()), Code: "01083", Owner: p.ID, MaxHP: 3}
	blade := &engine.Ally{ID: engine.EntityID(g.NextCardID()), Code: "21019", Owner: p.ID, MaxHP: 2}
	g.Allies[mock.ID] = mock
	g.Allies[blade.ID] = blade
	p.Allies = append(p.Allies, mock.ID, blade.ID)

	up := &engine.Upgrade{ID: engine.EntityID(g.NextCardID()), Code: "12017", Owner: p.ID}
	g.Upgrades[up.ID] = up
	b := engine.LookupBehavior("12017")
	if b == nil || b.OnPlay == nil {
		t.Fatal("Power Gloves behavior missing")
	}
	out := b.OnPlay(g, up)
	if len(out) != 1 {
		t.Fatalf("expected attach question, got %#v", out)
	}
	aq, ok := out[0].(engine.AskQuestion)
	if !ok || len(aq.Question.Choices) != 1 {
		t.Fatalf("expected one attach choice, got %#v", out)
	}
	if aq.Question.Choices[0].SourceID != blade.ID {
		t.Fatalf("Power Gloves offered %s, want Blade %s", aq.Question.Choices[0].SourceID, blade.ID)
	}
}
