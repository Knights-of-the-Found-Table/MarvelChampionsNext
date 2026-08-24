package aoa

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

func registerAoaScenarios() {
	// Unus — Hunting Gene Traitors.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "45062",
		Name:             "Unus — Hunting Gene Traitors",
		VillainBases:     []string{"45059"},
		MainSchemeStages: []string{"45062"},
		ExtraSets:        []string{"unus", "infinites", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			s := &engine.SideScheme{
				ID:        g.NextEntityID(engine.KindSideScheme),
				Code:      "45071",
				Threat:    0,
				MaxThreat: 99,
			}
			g.SideSchemes[s.ID] = s
			g.Logf("Gene Pool enters play")
			return nil
		},
	})

	// The Four Horsemen of Apocalypse.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "45085",
		Name:             "The Horsemen of Apocalypse",
		VillainBases:     []string{"45081a", "45082a", "45083a", "45084a"},
		MainSchemeStages: []string{"45085"},
		ExtraSets:        []string{"four_horsemen", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			// The active counter starts on the first Horseman.
			for id := range g.Villains {
				g.ActiveVillain = id
				break
			}
			return nil
		},
		OnVillainDefeated: func(g *engine.Game, v *engine.Villain) []engine.Message {
			g.Delete(v.ID)
			for _, o := range g.Villains {
				if o != nil && o.HP() >= 1 {
					g.Logf("%s remains standing", o.EDef().Name)
					return nil
				}
			}
			return []engine.Message{engine.GameOver{Won: true, Reason: "All four Horsemen have fallen"}}
		},
	})

	// Apocalypse — The Age of Apocalypse.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "45103",
		Name:             "Apocalypse — The Age of Apocalypse",
		VillainBases:     []string{"45101a"},
		MainSchemeStages: []string{"45103"},
		ExtraSets:        []string{"apocalypse", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			// Reveal the Heart of the Empire; tuck the Tyrant's Throne.
			var msgs []engine.Message
			for i, c := range g.EncounterDeck {
				if c.Code == "45104" {
					g.EncounterDeck = append(g.EncounterDeck[:i:i], g.EncounterDeck[i+1:]...)
					msgs = append(msgs, engine.RevealEncounterCard{Player: cardutilFirst(g), Card: c})
					break
				}
			}
			for i, c := range g.EncounterDeck {
				if c.Code == "45105" {
					g.SetAside = append(g.SetAside, c)
					g.EncounterDeck = append(g.EncounterDeck[:i:i], g.EncounterDeck[i+1:]...)
					break
				}
			}
			return msgs
		},
	})

	// Dark Beast — Bogus Journey.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "45121",
		Name:             "Dark Beast — Bogus Journey",
		VillainBases:     []string{"45118"},
		MainSchemeStages: []string{"45121"},
		ExtraSets:        []string{"dark_beast", "blue_moon", "genosha", "savage_land", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			// The three Settings wait set aside; Dark Beast's reveal
			// brings a random one out (VillainStage hook below).
			for _, code := range []string{"45127", "45133", "45139"} {
				for i, c := range g.EncounterDeck {
					if c.Code == code {
						g.SetAside = append(g.SetAside, c)
						g.EncounterDeck = append(g.EncounterDeck[:i:i], g.EncounterDeck[i+1:]...)
						break
					}
				}
			}
			// Blue Moon, Genosha and Savage Land cards stay in the deck;
			// only the Setting environments are benched.
			if v := darkBeast(g); v != nil {
				return revealRandomSetting(g)
			}
			return nil
		},
	})

	// En Sabah Nur — the pyramid of power.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "45147",
		Name:             "En Sabah Nur — The Rise of Apocalypse",
		VillainBases:     []string{"45184a"},
		MainSchemeStages: []string{"45147", "45148"},
		ExtraSets:        []string{"en_sabah_nur", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			// Apocalypse begins in Biomorph form (his a side, the spawn
			// default) and everyone takes a facedown encounter card.
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, engine.DealEncounterToPlayer{Player: p.ID})
			}
			return msgs
		},
		OnMainSchemeMaxed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			if s.Stage < len(s.StageCodes) {
				return []engine.Message{engine.ReplaceMainScheme{Scheme: s.ID}}
			}
			return []engine.Message{engine.GameOver{Won: false, Reason: "Apocalypse rises"}}
		},
	})
}

func cardutilFirst(g *engine.Game) engine.PlayerID {
	for _, p := range g.Players {
		if p.FirstPlayer {
			return p.ID
		}
	}
	if len(g.Players) > 0 {
		return g.Players[0].ID
	}
	return ""
}
