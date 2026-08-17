// Package nextevolution registers NeXt Evolution: Cable and Domino, plus
// the Marauders, Juggernaut and Mister Sinister scenarios.
package nextevolution

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

func init() {
	registerCable()
	registerDomino()
	registerScenarios()
}

// registerCable installs Cable (40001a/b): after he defeats a side scheme
// he readies (attribution approximated: any side-scheme defeat while
// Cable is in hero form, once per round).
func registerCable() {
	engine.RegisterBehavior("40001", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || g.SideSchemes[m.Scheme] == nil {
				return nil
			}
			p := g.Player(e.EID())
			if p == nil || !p.IsHero() || p.Exhausted {
				return nil
			}
			if g.UsedThisRound["40001-ready"] {
				return nil
			}
			g.UsedThisRound["40001-ready"] = true
			g.Logf("Cable readies after defeating a side scheme")
			return []engine.Message{engine.ReadyEntity{ID: p.ID}}
		},
	})
}

// registerDomino installs Domino (40037a/b): action — swap a card in hand
// with the top card of your deck (once per round). The double-wild
// counting rider is not modeled.
func registerDomino() {
	engine.RegisterBehavior("40037", &engine.Behavior{
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			if len(p.Hand) == 0 || len(p.Deck) == 0 {
				return nil
			}
			return []engine.Ability{{
				Label:        "Domino — swap a card in hand with the top card of your deck",
				Type:         engine.AbilityAction,
				OncePerRound: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(self)
					if p == nil {
						return nil
					}
					var choices []engine.Choice
					for _, c := range p.Hand {
						choices = append(choices, engine.Choice{
							Label: "Swap out " + c.Def().Name, Kind: engine.ChoiceCard, CardCode: c.Code,
						}.Msgs(engine.SwapHandWithDeckTop{Player: p.ID, CardID: c.ID}))
					}
					return []engine.Message{engine.AskQuestion{
						Player:   p.ID,
						Question: engine.Ask("Domino — which hand card swaps with the deck top?", choices...),
					}}
				},
			}}
		},
	})
}

// registerScenarios registers the box's scenarios.
func registerScenarios() {
	// Mutant Massacre (the Marauders, seven simultaneous villains).
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "40077",
		Name:             "Marauders — Mutant Massacre",
		VillainBases:     []string{"40070", "40071", "40072", "40073", "40074", "40075", "40076"},
		MainSchemeStages: []string{"40077a", "40078a"},
		ExtraSets:        []string{"marauders", "standard"},
	})

	// The Unstoppable Juggernaut.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "40121",
		Name:             "Juggernaut — The Unstoppable Juggernaut",
		VillainBases:     []string{"40118"},
		MainSchemeStages: []string{"40121a"},
		ExtraSets:        []string{"juggernaut", "standard"},
	})

	// Sinister Intent (Mister Sinister).
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "40139",
		Name:             "Mister Sinister — Sinister Intent",
		VillainBases:     []string{"40136"},
		MainSchemeStages: []string{"40139a", "40140a", "40141a"},
		ExtraSets:        []string{"mister_sinister", "marauders", "standard"},
	})
}
