package starlord_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/starlord"
)

func newStldGame(t *testing.T, seed int64, setup func(g *engine.Game)) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Peter", HeroBase: "17001", Deck: map[string]int{
				"17003": 1, "17004": 1, "17028": 1,
				"17018": 2, "17031": 2, "17020": 2,
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

// TestGutsyMoveScalesWithFacedown: 2 + 2 per facedown encounter card.
func TestGutsyMoveScalesWithFacedown(t *testing.T) {
	g := newStldGame(t, 101, func(g *engine.Game) {
		// Two facedown encounter cards → 2 + 4 = 6 threat removal.
		p := g.Players[0]
		p.EncounterDown = append(p.EncounterDown,
			engine.Card{ID: g.NextCardID(), Code: "01106"},
			engine.Card{ID: g.NextCardID(), Code: "01106"})
	})
	p := g.Players[0]
	gm := engine.Card{ID: g.NextCardID(), Code: "17004", Owner: p.ID}
	p.Hand = append(p.Hand, gm)
	threat := g.MainScheme.Threat
	g.Push(engine.PlayCard{Player: p.ID, Card: gm, Paid: engine.CostPaid{}})
	for i := 0; i < 6; i++ {
		pq := g.Pending()
		if pq == nil || g.Over {
			break
		}
		if strings.Contains(pq.Question.Prompt, "Gutsy Move") {
			_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
			break
		}
		_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
	}
	if g.MainScheme.Threat != max(0, threat-6) {
		t.Fatalf("Gutsy Move should remove 6 (2+2/facedown), %d -> %d", threat, g.MainScheme.Threat)
	}
}

// TestHelmetHandSizeBonus: +1 hand size per facedown card in hero form.
func TestHelmetHandSizeBonus(t *testing.T) {
	g := newStldGame(t, 102, func(g *engine.Game) {
		p := g.Players[0]
		p.Side = engine.SideHero
		helm := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "17010", Owner: p.ID}
		g.Upgrades[helm.ID] = helm
		p.Upgrades = append(p.Upgrades, helm.ID)
		p.EncounterDown = append(p.EncounterDown,
			engine.Card{ID: g.NextCardID(), Code: "01106"},
			engine.Card{ID: g.NextCardID(), Code: "01106"})
	})
	p := g.Players[0]
	base := 5 // Star-Lord hero hand size
	if got := p.HandSize(g); got != base+2 {
		t.Fatalf("Helmet should give +2 hand size with 2 facedown cards, got %d (want %d)", got, base+2)
	}
}
