// Package galaxysmostwanted registers The Galaxy's Most Wanted: Groot and
// Rocket Raccoon, plus the Drang, Collector and Nebula scenarios.
// Signature cards beyond the identity abilities fall back to generic
// behavior.
package galaxysmostwanted

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

func init() {
	registerGroot()
	registerRocket()
	registerScenarios()
}

// registerGroot installs Groot / Flora Colossus (16001a/b). Growth
// counters prevent damage; signature growth cards fall back to generic.
func registerGroot() {
	engine.RegisterBehavior("16001", &engine.Behavior{
		// Flora Colossus — when Groot would take damage, remove that many
		// growth counters, preventing 1 damage per counter
		// (approximation: identity-level counters; the signature cards
		// that add counters are approximated as generic).
		IdentityDamagePrevention: func(g *engine.Game, p *engine.Player, n int) int {
			use := min(p.GrowthCounters, n)
			if use <= 0 {
				return 0
			}
			p.GrowthCounters -= use
			g.Logf("Groot's growth counters prevent %d damage (%d left)", use, p.GrowthCounters)
			return use
		},
		// Approximation: without the Flora Colossus upgrades modeled,
		// Groot starts with 4 growth counters.
		HeroSetup: func(g *engine.Game, p *engine.Player) []engine.Message {
			p.GrowthCounters = 4
			g.Logf("Groot begins the game with 4 growth counters")
			return nil
		},
	})
}

// registerRocket installs Rocket Raccoon (16029a/b). The excess-damage
// rider is not modeled (overkill is absent); his setup draws extra cards.
func registerRocket() {
	engine.RegisterBehavior("16029", &engine.Behavior{})
}

// registerScenarios registers the box's three scenarios.
func registerScenarios() {
	// Planetary Invasion (Drang and the Brotherhood of Badoon).
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:   "16057",
		Name: "Drang — Planetary Invasion",
		// 16057 is a treachery in the card data; the scenario's actual
		// scheme is Terrestrial Invasion / Protect the Planet (16061/16062).
		VillainBases:     []string{"16058"},
		MainSchemeStages: []string{"16061b", "16062b"},
		ExtraSets:        []string{"brotherhood_of_badoon", "standard"},
	})

	// Infiltrate the Museum (the Collector's Grand Collection).
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "16073",
		Name:             "Collector — Infiltrate the Museum",
		VillainBases:     []string{"16070"},
		MainSchemeStages: []string{"16073b", "16083b", "16084b"},
		ExtraSets:        []string{"infiltrate_the_museum", "escape_the_museum", "standard"},
	})

	// The Missing Milano (Nebula's three-stage chase).
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "16082",
		Name:             "Nebula — The Missing Milano",
		VillainBases:     []string{"16088"},
		MainSchemeStages: []string{"16082b", "16091b", "16092b"},
		ExtraSets:        []string{"nebula", "standard"},
	})
}
