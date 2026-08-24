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
// generic play. HeroSetup installs the Solid mass-form upgrade (32031a).
func registerShadowcat() {
	engine.RegisterBehavior("32030", &engine.Behavior{
		HeroSetup: shadowcatSetup,
	})
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
		Setup: func(g *engine.Game) []engine.Message {
			// Robert Kelly joins the first player, hunted via Find the
			// Senator.
			pid := g.Players[0].ID
			a := &engine.Ally{
				ID: g.NextEntityID(engine.KindAlly), Code: "32066", Owner: pid,
				MaxHP: 6,
			}
			g.AddAlly(a, pid)
			return nil
		},
		OnMainSchemeMaxed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			if s.Stage == 1 {
				g.Logf("Sabretooth corners his prey — the main scheme advances")
				return []engine.Message{engine.ReplaceMainScheme{Scheme: s.ID}}
			}
			return []engine.Message{engine.GameOver{Won: false, Reason: "The Injured Senator scheme completed"}}
		},
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
		Setup: func(g *engine.Game) []engine.Message {
			// The Captive allies wait in the set-aside area for the
			// Abduction Protocols rescues.
			for _, code := range []string{"32089", "32090", "32091", "32092"} {
				g.SetAside = append(g.SetAside, engine.Card{ID: g.NextCardID(), Code: code})
			}
			// One Abduction Protocols starts in play.
			for i, c := range g.EncounterDeck {
				if c.Code == "32100" {
					g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
					return []engine.Message{engine.RevealEncounterCard{Player: g.Players[0].ID, Card: c}}
				}
			}
			return nil
		},
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
		Setup: func(g *engine.Game) []engine.Message {
			g.SpawnEnvironment("32130")
			return nil
		},
		OnMainSchemeDefeated: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			// A location saved: it goes to the victory display and the
			// next location reveals; three saved locations win the game.
			g.VictoryDisplay = append(g.VictoryDisplay, engine.Card{ID: g.NextCardID(), Code: s.Code})
			saved := 0
			for _, c := range g.VictoryDisplay {
				d := c.Def()
				if d != nil && d.Type == "main_scheme" {
					saved++
				}
			}
			if saved >= 3 {
				return []engine.Message{engine.GameOver{Won: true, Reason: "Three mansion locations saved"}}
			}
			return []engine.Message{engine.ReplaceMainScheme{Scheme: s.ID}}
		},
		OnMainSchemeMaxed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			return []engine.Message{engine.ReplaceMainScheme{Scheme: s.ID}}
		},
	})

	// Asteroid M (Magneto).
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "32141",
		Name:             "Magneto — Asteroid M",
		VillainBases:     []string{"32138"},
		MainSchemeStages: []string{"32141b", "32142b", "32143b"},
		ExtraSets:        []string{"magneto_villain", "acolytes", "standard"},
	})
}
