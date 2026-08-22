package vision_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/vision"
)

func newVisionGame(t *testing.T, seed int64, setup func(g *engine.Game)) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Vision", HeroBase: "26001", Deck: map[string]int{
				"26008": 1, "26009": 1, "26024": 1,
				"26025": 3, "26026": 3, "26027": 3,
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

// TestMassFormFlip: the upgrade grants intangible; the identity ability
// flips to dense; Solar Beam resolves per the current form.
func TestMassFormFlip(t *testing.T) {
	g := newVisionGame(t, 121, func(g *engine.Game) {
		p := g.Players[0]
		p.Side = engine.SideHero
		intang := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "26002", Owner: p.ID}
		g.Upgrades[intang.ID] = intang
		p.Upgrades = append(p.Upgrades, intang.ID)
		p.ExtraTraits = append(p.ExtraTraits, "intangible")
	})
	p := g.Players[0]

	// Solar Beam in intangible form removes 5 threat.
	sb := engine.Card{ID: g.NextCardID(), Code: "26008", Owner: p.ID}
	p.Hand = append(p.Hand, sb)
	threat := g.MainScheme.Threat
	g.Push(engine.PlayCard{Player: p.ID, Card: sb, Paid: engine.CostPaid{}})
	for i := 0; i < 6; i++ {
		pq := g.Pending()
		if pq == nil || g.Over {
			break
		}
		if strings.Contains(pq.Question.Prompt, "Solar Beam") {
			_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
			break
		}
		_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
	}
	if g.MainScheme.Threat != max(0, threat-5) {
		t.Fatalf("Solar Beam (intangible) should remove 5 threat, %d -> %d", threat, g.MainScheme.Threat)
	}

	// Flip to dense via the message; Solar Beam now deals 7. The first
	// section's unblock consumed the once-per-turn form change, so reset
	// it before unblocking again.
	p.FormChanged = false
	p.Side = engine.SideAlterEgo             // ensure the form choice exists
	g.Push(engine.SetMassForm{Player: p.ID}) // flip
	sb2 := engine.Card{ID: g.NextCardID(), Code: "26008", Owner: p.ID}
	p.Hand = append(p.Hand, sb2)
	var villain engine.EntityID
	for id := range g.Villains {
		villain = id
	}
	hp := g.Villains[villain].HP()
	g.Push(engine.PlayCard{Player: p.ID, Card: sb2, Paid: engine.CostPaid{}})
	// Unblock with any plain leaf from the (possibly stale) pending menu
	// — a no-op answer whose queue run then drains our pushed messages.
	if pq := g.Pending(); pq != nil {
		for _, c := range pq.Question.Choices {
			if c.Then == nil && !c.Disabled {
				if err := g.Answer(pq.Player, []string{c.ID}); err == nil {
					break
				}
			}
		}
	}
	for i := 0; i < 8; i++ {
		pq := g.Pending()
		if pq == nil || g.Over {
			break
		}
		if strings.Contains(pq.Question.Prompt, "Solar Beam") {
			_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
			break
		}
		_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
	}
	if g.Villains[villain].HP() != hp-7 {
		t.Fatalf("Solar Beam (dense) should deal 7 damage, %d -> %d", hp, g.Villains[villain].HP())
	}
}
