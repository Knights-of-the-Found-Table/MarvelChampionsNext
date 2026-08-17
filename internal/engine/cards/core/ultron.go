package core

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// registerUltron registers "The Ultron Imperative" scenario.
func registerUltron() {
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "01137",
		Name:             "Ultron — The Imperative",
		VillainBases:     []string{"01134"},
		MainSchemeStages: []string{"01137", "01138", "01139"},
		ExtraSets:        []string{"under_attack", "standard"},
	})

	// Stage I: after Ultron attacks you, place 1 threat or spawn a Drone.
	engine.RegisterBehavior("01134", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player: p.ID,
				Question: engine.Ask("Ultron: place 1 threat on the main scheme or spawn a Drone?",
					engine.Choice{
						ID: "threat", Label: "Place 1 threat on the main scheme", Kind: engine.ChoiceLabel,
					}.Msgs(engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: e.EID()}),
					engine.Choice{
						ID: "drone", Label: "Put the top card of your deck into play as a Drone", Kind: engine.ChoiceLabel,
					}.Msgs(engine.SpawnDrone{Player: p.ID}),
				),
			}}
		},
	})

	// Stage II: forced interrupt when Ultron attacks — spawn a Drone first.
	engine.RegisterBehavior("01135", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.VillainActivates)
			if !ok || m.VillainID != e.EID() {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil || !p.IsHero() {
				return nil
			}
			return []engine.Message{engine.SpawnDrone{Player: p.ID}}
		},
	})

	// Stage III: Ultron cannot take damage while a Drone is in play.
	// (Drone +1/+1 is applied by the engine's droneBonus.)
	engine.RegisterBehavior("01136", &engine.Behavior{
		VillainDamageable: func(g *engine.Game, v *engine.Villain, damage int) bool {
			for _, mn := range g.Minions {
				if mn.IsDrone {
					return false
				}
			}
			return true
		},
	})

	// The Crimson Cowl (main scheme I): when revealed, each player spawns
	// a Drone. Handled via the scheme stage hook below.
}
