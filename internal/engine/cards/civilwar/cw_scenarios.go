package civilwar

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

func init() {
	registerCWSchemes()
	registerCWLeaderScenarios()
}

// registerCWSchemes registers the registration/resistance main-scheme
// families (single-face in the snapshot; the b faces carry threat stats).
func registerCWSchemes() {
	for _, code := range []string{
		"56063", "56064", "56096", "56097", "56121", "56122", "56123",
		"56124", "56141", "56142", "56172", "56173", "56199", "56200",
		"56201", "56202",
	} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}
}

// registerCWLeaderScenarios registers the leader showdowns: each Civil War
// leader plays as the scenario's villain (PvP approximated as co-op).
func registerCWLeaderScenarios() {
	leaderSetup := func(attachCode string) func(g *engine.Game) []engine.Message {
		return func(g *engine.Game) []engine.Message {
			if v := cwVillain(g); v != nil && attachCode != "" {
				// Non-expert leaders climb I→II, expert III→IV.
				if g.Difficulty == "expert" {
					v.SetVillainStages(expertLadder(v.Code))
				} else {
					v.SetVillainStages(normalLadder(v.Code))
				}
				cwFindAttach(g, attachCode, v.ID)
				g.TLogf("c.preparesForTheShowdown", v)
			}
			return nil
		}
	}

	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "56096b",
		Name:             "Civil War — Registration: Captain Marvel",
		VillainBases:     []string{"56092"},
		MainSchemeStages: []string{"56096b", "56097b"},
		ExtraSets:        []string{"mighty_avengers", "the_initiative", "standard"},
		Setup:            leaderSetup("56098"),
		OnMainSchemeMaxed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			if s.Stage < len(s.StageCodes) {
				return []engine.Message{engine.ReplaceMainScheme{Scheme: s.ID}}
			}
			return []engine.Message{engine.GameOver{Won: false, Reason: engine.Tf("reason.registrationAct")}}
		},
	})

	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "56121b",
		Name:             "Civil War — Registration: Iron Man",
		VillainBases:     []string{"56059"},
		MainSchemeStages: []string{"56121b", "56123b"},
		ExtraSets:        []string{"maria_hill_modular", "dangerous_recruits", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			if v := cwVillain(g); v != nil {
				if g.Difficulty == "expert" {
					v.SetVillainStages([]string{"56061", "56062"})
				} else {
					v.SetVillainStages([]string{"56059", "56060"})
				}
			}
			return nil
		},
		OnMainSchemeMaxed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			if s.Stage < len(s.StageCodes) {
				return []engine.Message{engine.ReplaceMainScheme{Scheme: s.ID}}
			}
			return []engine.Message{engine.GameOver{Won: false, Reason: engine.Tf("reason.registrationAct")}}
		},
	})

	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "56141b",
		Name:             "Civil War — Resistance: Captain America",
		VillainBases:     []string{"56137"},
		MainSchemeStages: []string{"56141b", "56142b"},
		ExtraSets:        []string{"secret_avengers", "new_avengers", "standard"},
		Setup:            leaderSetup("56143"),
		OnMainSchemeMaxed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			if s.Stage < len(s.StageCodes) {
				return []engine.Message{engine.ReplaceMainScheme{Scheme: s.ID}}
			}
			return []engine.Message{engine.GameOver{Won: false, Reason: engine.Tf("reason.resistanceCrushed")}}
		},
	})

	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "56172b",
		Name:             "Civil War — Resistance: Spider-Woman",
		VillainBases:     []string{"56168"},
		MainSchemeStages: []string{"56172b", "56173b"},
		ExtraSets:        []string{"spider_man_modular", "defenders", "standard"},
		Setup:            leaderSetup("56174"),
		OnMainSchemeMaxed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			if s.Stage < len(s.StageCodes) {
				return []engine.Message{engine.ReplaceMainScheme{Scheme: s.ID}}
			}
			return []engine.Message{engine.GameOver{Won: false, Reason: engine.Tf("reason.resistanceCrushed")}}
		},
	})
}

// normalLadder maps a leader's on-table code to its I→II stage pair.
func normalLadder(code string) []string {
	switch engine.BaseCodeOf(code) {
	case "56092":
		return []string{"56092", "56093"}
	case "56137":
		return []string{"56137", "56138"}
	case "56168":
		return []string{"56168", "56169"}
	}
	return nil
}

// expertLadder maps a leader's on-table code to its III→IV stage pair.
func expertLadder(code string) []string {
	switch engine.BaseCodeOf(code) {
	case "56092":
		return []string{"56094", "56095"}
	case "56137":
		return []string{"56139", "56140"}
	case "56168":
		return []string{"56170", "56171"}
	}
	return nil
}
