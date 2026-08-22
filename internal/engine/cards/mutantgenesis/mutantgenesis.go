// Package mutantgenesis registers Mutant Genesis: Colossus and Shadowcat,
// plus the Sabretooth, Sentinel and Brotherhood scenarios.
package mutantgenesis

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

func init() {
	registerColossus()
	registerShadowcat()
	registerScenarios()
}

// registerColossus installs Colossus (32001a/b): after changing to hero
// form he gains a tough status card (the extra-tough-slot rider is not
// modeled).
func registerColossus() {
	engine.RegisterBehavior("32001", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.ChangeForm)
			if !ok || m.Player != e.EID() {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil || !p.IsHero() {
				return nil
			}
			g.Logf("Steel Skin: Colossus gains a tough status card")
			return []engine.Message{engine.ToughEntity{Target: p.ID}}
		},
	})
}

// registerShadowcat installs Shadowcat (32030a/b). Selective
// Intangibility's keyword/crisis immunity is approximated: she may thwart
// crisis schemes is not modeled; the phased-form switching falls back to
// generic play.
func registerShadowcat() {
	engine.RegisterBehavior("32030", &engine.Behavior{})
}

// registerScenarios registers the box's scenarios.
func registerScenarios() {
	// Stalked by Sabretooth.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "32063",
		Name:             "Sabretooth — Stalked by Sabretooth",
		VillainBases:     []string{"32060"},
		MainSchemeStages: []string{"32063b", "32064b"},
		ExtraSets:        []string{"sabretooth", "standard"},
	})

	// Night of the Sentinels (Sentinel I-III, then Master Mold).
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:           "32087",
		Name:         "Sentinels — Night of the Sentinels",
		VillainBases: []string{"32084"},
		// Stage 2 reuses the Mutants at the Mall flip card (32088a, a side
		// scheme whose b face is an ally), so it keeps its a-side code.
		MainSchemeStages: []string{"32087b", "32088a"},
		ExtraSets:        []string{"project_wideawake", "standard"},
	})

	// The Sentinel Factory (Master Mold).
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "32112",
		Name:             "Master Mold — The Sentinel Factory",
		VillainBases:     []string{"32109"},
		MainSchemeStages: []string{"32112b", "32113b"},
		ExtraSets:        []string{"master_mold", "project_wideawake", "standard"},
	})

	// The Brotherhood Strikes (four simultaneous villains storming the
	// X-Mansion).
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "32125",
		Name:             "Brotherhood — The Brotherhood Strikes",
		VillainBases:     []string{"32121", "32122", "32123", "32124"},
		MainSchemeStages: []string{"32125b", "32126b", "32127b"},
		ExtraSets:        []string{"mansion_attack", "standard"},
	})
}
