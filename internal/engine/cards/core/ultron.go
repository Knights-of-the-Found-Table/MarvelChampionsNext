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
		MainSchemeStages: []string{"01137b", "01138b", "01139b"},
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
				Question: engine.Ask(engine.Tf("c.ultronPlace1ThreatOnTheMainSchemeOrSpawnADrone"),
					engine.Choice{
						ID: "threat", Label: engine.Tf("c.place1ThreatOnTheMainScheme"), Kind: engine.ChoiceLabel,
					}.Msgs(engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: e.EID()}),
					engine.Choice{
						ID: "drone", Label: engine.Tf("c.putTheTopCardOfYourDeckIntoPlayAsADrone"), Kind: engine.ChoiceLabel,
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

	// Main scheme 1A setup: put the Ultron Drones environment into play
	// (the encounter deck is shuffled during game start, right after).
	engine.RegisterBehavior("01137a", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			env := &engine.Environment{ID: g.NextEntityID("environment"), Code: "01140"}
			g.AddEnvironment(env)
			g.TLogf("c.entersPlay", env)
			return nil
		},
	})

	// Main scheme stages 2A/3A: each player puts the top card of their
	// deck into play facedown as a Drone engaged with them.
	droneStage := &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			msgs := make([]engine.Message, 0, len(g.Players))
			for _, p := range g.Players {
				msgs = append(msgs, engine.SpawnDrone{Player: p.ID})
			}
			return msgs
		},
	}
	engine.RegisterBehavior("01138a", droneStage)
	engine.RegisterBehavior("01139a", droneStage)
}
