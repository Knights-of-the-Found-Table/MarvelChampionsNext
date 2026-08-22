// Package core registers Core Set content: scenarios, villains and hero
// behaviors. Import for side effects.
package core

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

func init() {
	registerRhino()
	registerKlaw()
	registerUltron()
	registerSpiderMan()
	registerCoreCards()
	registerHeroes()
	registerTreacheries()
	registerAspectCards()
	registerCoreObligations()
}

// registerRhino registers "The Break-In" (Rhino scenario).
func registerRhino() {
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "01097",
		Name:             "Rhino — The Break-In",
		VillainBases:     []string{"01094"},
		MainSchemeStages: []string{"01097b"},
		ExtraSets:        []string{"bomb_scare", "standard"},
	})

	// Rhino II: search for Breakin' & Takin' side scheme.
	engine.RegisterBehavior("01095", &engine.Behavior{
		VillainStage: func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
			return findAndRevealSideScheme(g, "01107")
		},
	})

	// Rhino III: When Revealed: stun each hero.
	engine.RegisterBehavior("01096", &engine.Behavior{
		VillainStage: func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, engine.StunEntity{Target: p.ID})
			}
			return msgs
		},
	})
}

// registerKlaw registers "Underground Distribution" (Klaw scenario).
func registerKlaw() {
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "01116",
		Name:             "Klaw — Underground Distribution",
		VillainBases:     []string{"01113"},
		MainSchemeStages: []string{"01116b", "01117b"},
		ExtraSets:        []string{"masters_of_evil", "standard"},
		OnMainSchemeDefeated: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			msgs := []engine.Message{engine.ReplaceMainScheme{Scheme: s.ID}}
			for id := range g.Villains {
				msgs = append(msgs, engine.AdvanceVillainStage{VillainID: id})
			}
			return msgs
		},
	})

	// All Klaw stages: Forced Interrupt — when Klaw attacks, he gets 1
	// additional boost card.
	klawBoost := &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.VillainActivates)
			if !ok || m.VillainID != e.EID() {
				return nil
			}
			if p := g.Player(m.Player); p != nil && p.IsHero() {
				return []engine.Message{engine.DealBoost{Enemy: e.EID()}}
			}
			return nil
		},
	}
	engine.RegisterBehavior("01113", klawBoost)
	engine.RegisterBehavior("01114", klawBoost)
	engine.RegisterBehavior("01115", klawBoost)

	// Stage 1A setup: search the encounter deck for the Defense Network
	// side scheme and reveal it (the deck is shuffled during game start,
	// right after the a-face effects resolve).
	engine.RegisterBehavior("01116a", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			return findAndRevealSideScheme(g, "01125")
		},
	})

	// Stage 2A: discard encounter cards until a minion shows up; it enters
	// play engaged with the first player.
	engine.RegisterBehavior("01117a", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			for len(g.EncounterDeck) > 0 {
				c := g.EncounterDeck[0]
				g.EncounterDeck = g.EncounterDeck[1:]
				if c.Def().Type == "minion" {
					return []engine.Message{engine.RevealEncounterCard{Player: firstPlayer(g), Card: c}}
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
				g.Logf("%s is discarded", c.Def().Name)
			}
			return nil
		},
	})

	// Klaw II: When Revealed: search for The "Immortal" Klaw side scheme.
	engine.RegisterBehavior("01114", &engine.Behavior{
		React: klawBoost.React,
		VillainStage: func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
			return findAndRevealSideScheme(g, "01127")
		},
	})
}

// findAndRevealSideScheme searches deck+discard for a side scheme card and
// reveals it against the first player.
func findAndRevealSideScheme(g *engine.Game, code string) []engine.Message {
	for i, c := range g.EncounterDeck {
		if c.Code == code {
			g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
			return []engine.Message{engine.RevealEncounterCard{Player: firstPlayer(g), Card: c}}
		}
	}
	for i, c := range g.EncounterDiscard {
		if c.Code == code {
			g.EncounterDiscard = append(g.EncounterDiscard[:i], g.EncounterDiscard[i+1:]...)
			return []engine.Message{engine.RevealEncounterCard{Player: firstPlayer(g), Card: c}}
		}
	}
	return nil
}

func firstPlayer(g *engine.Game) engine.PlayerID {
	for _, p := range g.Players {
		if p.FirstPlayer {
			return p.ID
		}
	}
	return g.Players[0].ID
}
