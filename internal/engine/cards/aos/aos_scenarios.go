package aos

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() { registerAOSScenarios() }

// advanceSchemeOrLose advances the main scheme on completion and loses on
// the final stage.
func advanceSchemeOrLose(reason string) func(g *engine.Game, s *engine.MainScheme) []engine.Message {
	return func(g *engine.Game, s *engine.MainScheme) []engine.Message {
		if s.Stage < len(s.StageCodes) {
			return []engine.Message{engine.ReplaceMainScheme{Scheme: s.ID}}
		}
		return []engine.Message{engine.GameOver{Won: false, Reason: reason}}
	}
}

func registerAOSScenarios() {
	// The Widow's Web — Black Widow I→III (separate stage cards).
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "50067a",
		Name:             "Black Widow — The Widow's Web",
		VillainBases:     []string{"50064"},
		MainSchemeStages: []string{"50067b"},
		ExtraSets:        []string{"a.i.m._abduction", "a.i.m._science", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			for _, v := range g.Villains {
				v.SetVillainStages([]string{"50064", "50065", "50066"})
			}
			// Each player is ambushed by a minion from the encounter deck.
			for _, p := range g.Players {
				for i, c := range g.EncounterDeck {
					if c.Def().Type == "minion" {
						card := c
						g.EncounterDeck = append(g.EncounterDeck[:i:i], g.EncounterDeck[i+1:]...)
						def := card.Def()
						mn := &engine.Minion{
							ID: g.NextEntityID(engine.KindMinion), Code: card.Code,
							MaxHP:     intValue(def.HP, 1),
							AttackVal: intValue(def.Attack, 0), SchemeVal: intValue(def.Scheme, 0),
							EngagedWith: p.ID,
						}
						g.Minions[mn.ID] = mn
						g.Logf("%s is ambushed by %s", p.Name, def.Name)
						break
					}
				}
			}
			return []engine.Message{engine.ShuffleEncounterDeck{}}
		},
	})

	// Infiltrate A.I.M. Island Embassy — Batroc over three scheme stages.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "50087a",
		Name:             "Batroc — Infiltrate A.I.M. Island Embassy",
		VillainBases:     []string{"50086"},
		MainSchemeStages: []string{"50087b", "50088b", "50089b"},
		ExtraSets:        []string{"a.i.m._science", "batrocs_brigade", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			// Alert Level environment, Low side up.
			g.SpawnEnvironment("50090a")
			// The captives wait in the wings: one Rescued Captive per
			// player joins them (approximation of the set-aside pool).
			for _, p := range g.Players {
				if card, ok := takeEncounterBy(g, func(def *data.CardDef) bool { return def.Code == "50091" }); ok {
					a := &engine.Ally{
						ID: g.NextEntityID(engine.KindAlly), Code: card.Code,
						Owner: p.ID, MaxHP: intValue(card.Def().HP, 3),
						AttackVal: intValue(card.Def().Attack, 1), ThwartVal: intValue(card.Def().Thwart, 1),
					}
					g.AddAlly(a, p.ID)
					g.Logf("Rescued Captive joins %s", p.Name)
				}
			}
			return nil
		},
		OnMainSchemeMaxed: advanceSchemeOrLose("The A.I.M. embassy operation completed"),
	})

	// Upgrading Adaptoids — M.O.D.O.K. behind his Holding Cells.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "50104a",
		Name:             "M.O.D.O.K. — Upgrading Adaptoids",
		VillainBases:     []string{"50103"},
		MainSchemeStages: []string{"50104b"},
		ExtraSets:        []string{"scientist_supreme", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			// The four Holding Cells enter play with lock counters.
			for _, code := range []string{"50105a", "50106a", "50107a", "50108a"} {
				cell := g.SpawnEnvironment(code)
				cell.Counters = 2 * len(g.Players)
			}
			// One random Adaptoid Upgrade environment joins; the rest
			// wait in the deck (approximation).
			adaptoids := []string{"50109", "50110", "50111", "50112"}
			pick := adaptoids[g.Random(len(adaptoids))]
			for _, code := range adaptoids {
				if code == pick {
					takeEncounterBy(g, func(def *data.CardDef) bool { return def.Code == code })
					g.SpawnEnvironment(code)
					g.Logf("%s empowers the Adaptoids", code)
				}
			}
			// Each player is engaged by an Adaptoid.
			for _, p := range g.Players {
				for i, c := range g.EncounterDeck {
					if c.Code == "50113" {
						card := c
						g.EncounterDeck = append(g.EncounterDeck[:i:i], g.EncounterDeck[i+1:]...)
						def := card.Def()
						mn := &engine.Minion{
							ID: g.NextEntityID(engine.KindMinion), Code: card.Code,
							MaxHP:     intValue(def.HP, 1),
							AttackVal: intValue(def.Attack, 0), SchemeVal: intValue(def.Scheme, 0),
							EngagedWith: p.ID,
						}
						g.Minions[mn.ID] = mn
						g.Logf("%s is engaged by an Adaptoid", p.Name)
						break
					}
				}
			}
			return []engine.Message{engine.ShuffleEncounterDeck{}}
		},
	})

	// Apprehending Rogue Agents — Citizen V and the Thunderbolts.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "50130a",
		Name:             "Citizen V — Apprehending Rogue Agents",
		VillainBases:     []string{"50129"},
		MainSchemeStages: []string{"50130b"},
		ExtraSets: []string{
			"gravitational_pull", "hard_sound", "pale_little_spider",
			"power_of_the_atom", "supersonic", "the_leaper", "standard",
		},
		Setup: func(g *engine.Game) []engine.Message {
			g.SpawnEnvironment("50131a")
			// One random Elite Thunderbolt joins the hunt immediately.
			elites := []string{"50139", "50143", "50148", "50152", "50156", "50161"}
			pick := elites[g.Random(len(elites))]
			if card, ok := takeEncounterBy(g, func(def *data.CardDef) bool { return def.Code == pick }); ok {
				def := card.Def()
				p := g.Players[0]
				mn := &engine.Minion{
					ID: g.NextEntityID(engine.KindMinion), Code: card.Code,
					MaxHP:     intValue(def.HP, 1),
					AttackVal: intValue(def.Attack, 0), SchemeVal: intValue(def.Scheme, 0),
					EngagedWith: p.ID,
				}
				g.Minions[mn.ID] = mn
				g.Logf("%s joins Citizen V's manhunt", def.Name)
			}
			return nil
		},
	})

	// Zemo's Manipulations — Baron Zemo hides behind the executive board.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "50167a",
		Name:             "Baron Zemo — Zemo's Manipulations",
		VillainBases:     []string{"50165"},
		MainSchemeStages: []string{"50167b", "50168b", "50169b"},
		ExtraSets:        []string{"s.h.i.e.l.d.", "scientist_supreme", "s.h.i.e.l.d._executive_board", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			// The executive board convenes with 2 secret counters each.
			for _, code := range []string{"50181a", "50182a", "50183a"} {
				env := g.SpawnEnvironment(code)
				env.Counters = 2
			}
			// The evidence is prepared and set aside (the evidence set never
			// shuffles into the encounter deck; kept as a no-op filter).
			var kept engine.CardList
			for _, c := range g.EncounterDeck {
				if c.Code >= "50185" && c.Code <= "50193" {
					g.SetAside = append(g.SetAside, c)
					continue
				}
				kept = append(kept, c)
			}
			g.EncounterDeck = kept
			g.Logf("The executive board's evidence is set aside")
			return nil
		},
		OnMainSchemeMaxed: advanceSchemeOrLose("Baron Zemo's manipulations succeed"),
	})

	// The Accusation (stage two): the heroes name their suspect.
	engine.RegisterBehavior("50168", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			g.LogMajorf("The heroes make their accusation — who is Zemo's insider?")
			return nil
		},
	})

	// Fighting Zemo (stage three): unmask him and arm his sword.
	engine.RegisterBehavior("50169", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			for _, v := range g.Villains {
				if v == nil || data.BaseCode(v.Code) != "50165" {
					continue
				}
				v.Code = "50165b"
				v.MaxHP = 16
				v.Damage = 0
				g.LogMajorf("Baron Zemo is unmasked!")
				g.SpawnAttachment("50170", v.ID)
			}
			return nil
		},
	})
}
