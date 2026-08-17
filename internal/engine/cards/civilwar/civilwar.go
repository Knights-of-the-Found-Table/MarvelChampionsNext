// Package civilwar registers the Civil War box: Tigra and Hulkling, plus
// the Superhero Registration Act scenario (villain-less: defeating the
// main scheme wins).
package civilwar

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

func init() {
	registerTigra()
	registerHulkling()
	registerScenarios()
}

// registerTigra installs Tigra (56001a/b): after the player phase begins,
// draw 1 card per minion engaged with her (max 3). The Hunted rider is
// not modeled.
func registerTigra() {
	engine.RegisterBehavior("56001", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.BeginPhase)
			if !ok || m.Phase != engine.PhasePlayer {
				return nil
			}
			p := g.Player(e.EID())
			if p == nil {
				return nil
			}
			n := 0
			for _, mn := range g.Minions {
				if mn.EngagedWith == p.ID {
					n++
				}
			}
			n = min(n, 3)
			if n <= 0 {
				return nil
			}
			g.Logf("On the Hunt: %s draws %d", p.Name, n)
			return []engine.Message{engine.DrawCards{Player: p.ID, N: n}}
		},
	})
}

// registerHulkling installs Hulkling (56029a/b). Chosen Shape's
// shapeshift cycling is approximated away (shapeshift upgrades play
// normally).
func registerHulkling() {
	engine.RegisterBehavior("56029", &engine.Behavior{})
}

// registerScenarios registers the box's scenarios.
func registerScenarios() {
	// Superhero Registration Act: no villain; defeat the main scheme to
	// win.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "56063",
		Name:             "Civil War — Superhero Registration Act",
		VillainBases:     []string{},
		MainSchemeStages: []string{"56063a"},
		ExtraSets:        []string{"registration", "cape_killer", "standard"},
		OnMainSchemeDefeated: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			return []engine.Message{engine.GameOver{Won: true, Reason: "The Superhero Registration Act was defeated"}}
		},
	})
}
