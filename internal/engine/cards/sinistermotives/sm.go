package sinistermotives

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// registerSMScenarios registers the box's five scenarios.
func registerSMScenarios() {
	// Sandman — Hapless Pedestrians.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "27064",
		Name:             "Sandman — Hapless Pedestrians",
		VillainBases:     []string{"27061"},
		MainSchemeStages: []string{"27064"},
		ExtraSets:        []string{"city_in_chaos", "standard"},
	})

	// Venom — "Leave Us Alone!".
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "27076",
		Name:             "Venom — Leave Us Alone!",
		VillainBases:     []string{"27073"},
		MainSchemeStages: []string{"27076a"},
		ExtraSets:        []string{"symbiotic_strength", "standard"},
	})

	// Mysterio — Maze of Mirrors.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "27087",
		Name:             "Mysterio — Maze of Mirrors",
		VillainBases:     []string{"27084"},
		MainSchemeStages: []string{"27087", "27088"},
		ExtraSets:        []string{"personal_nightmare", "standard"},
	})

	// The Sinister Six.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:   "27100",
		Name: "The Sinister Six",
		VillainBases: []string{
			"27094", "27095", "27096", "27097", "27098", "27099",
		},
		MainSchemeStages: []string{"27100", "27101a"},
		ExtraSets:        []string{"guerrilla_tactics", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			// The Light at the End starts faceup.
			s := &engine.SideScheme{
				ID:        g.NextEntityID("sidescheme"),
				Code:      "27102a",
				Threat:    10 * len(g.Players),
				MaxThreat: 10 * len(g.Players),
			}
			g.SideSchemes[s.ID] = s
			g.TLogf("c.lightAtTheEndEntersPlay")
			return nil
		},
		OnVillainDefeated: func(g *engine.Game, v *engine.Villain) []engine.Message {
			// Set the villain aside instead of staging up.
			base := engine.BaseCodeOf(v.ECode())
			delete(g.Villains, v.ID)
			g.SetAside = append(g.SetAside, engine.Card{ID: g.NextCardID(), Code: base})
			g.TLogf("c.isSetAside", v)
			if g.ActiveVillain == v.ID {
				advanceSixCounter(g, v.ID)
			}
			if len(g.Villains) == 0 {
				if msgs := sixAmbush(g); len(msgs) == 0 && len(g.SetAside) == 0 {
					return []engine.Message{engine.GameOver{Won: true, Reason: engine.Tf("reason.sinisterSixDefeated")}}
				}
			}
			return nil
		},
	})

	// Venom Goblin — Skies Over New York.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "27116",
		Name:             "Venom Goblin — Skies Over New York",
		VillainBases:     []string{"27113"},
		MainSchemeStages: []string{"27116a"},
		ExtraSets:        []string{"venom_goblin", "symbiotic_strength", "goblin_gear", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			// Lower and Upper Manhattan ride as high-threat surrogates;
			// the glider starts on Midtown (the main scheme).
			for _, code := range []string{"27117a", "27119a"} {
				def := engine.DB.MustLookup(code)
				maxT := 11
				if def.Threat != nil {
					maxT = *def.Threat
				}
				s := &engine.SideScheme{
					ID:        g.NextEntityID("sidescheme"),
					Code:      code,
					Threat:    maxT / 2,
					MaxThreat: maxT,
				}
				g.SideSchemes[s.ID] = s
				g.TLogf("c.entersPlay", def)
			}
			if g.MainScheme != nil {
				g.GliderCounter = g.MainScheme.ID
			}
			return nil
		},
	})
}

func init() {
	registerSMHeroCards()
	registerPromos()
	registerSandman()
	registerVenomScenario()
	registerMysterio()
	registerSinisterSix()
	registerVenomGoblin()
	registerSMModulars()
	registerSMScenarios()
}
