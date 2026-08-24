// Package nextevolution registers NeXt Evolution: Cable and Domino, plus
// the Marauders, Juggernaut, Mister Sinister and Stryfe scenarios.
package nextevolution

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() {
	registerCable()
	registerDomino()
	registerCableCards()
	registerDominoCards()
	registerXForceCards()
	registerMarauders()
	registerOnTheRun()
	registerNastyBoys()
	registerJuggernaut()
	registerSinister()
	registerStryfe()
	registerCampaign()
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

// marauderNextVillain spawns the next villain from the set-aside villain
// deck, discarding minions that share its title.
func marauderNextVillain(g *engine.Game) {
	for i, c := range g.SetAside {
		if c.Def().Type != "villain" {
			continue
		}
		g.SetAside = append(g.SetAside[:i:i], g.SetAside[i+1:]...)
		name := c.Def().Name
		_ = name
		// Discard minions sharing the incoming villain's title.
		for _, mn := range g.Minions {
			if mn != nil && mn.EDef().Name == name {
				g.Delete(mn.ID)
				g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: mn.Code})
			}
		}
		if v := g.SpawnVillainFromCard(c.Code); v != nil {
			g.Logf("Next Marauder: %s", v.EDef().Name)
		}
		return
	}
}

// registerScenarios registers the box's scenarios.
func registerScenarios() {
	// Mutant Massacre (the Marauders, one villain at a time under Routed).
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "40077",
		Name:             "Marauders — Mutant Massacre",
		VillainBases:     []string{"40070a", "40071a", "40072a", "40073a", "40074a", "40075a", "40076a"},
		MainSchemeStages: []string{"40077", "40078"},
		ExtraSets:        []string{"marauders", "morlock_siege", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			// Routed environment.
			g.SpawnEnvironment("40081a")
			// Set aside the Morlock allies and Hide!; they return on stage
			// two's reveal.
			tuckFromEncounter(g, "40079")
			tuckFromEncounter(g, "40080")
			// Build the villain deck: keep one random villain in play, tuck
			// the rest.
			var ids []engine.EntityID
			for id := range g.Villains {
				ids = append(ids, id)
			}
			if len(ids) > 1 {
				keep := g.Random(len(ids))
				for i, id := range ids {
					if i == keep {
						continue
					}
					if v := g.Villains[id]; v != nil {
						// Keep the full a-suffixed code: the unsuffixed
						// form is not a card in the database.
						g.SetAside = append(g.SetAside, engine.Card{ID: g.NextCardID(), Code: v.Code})
						g.Delete(id)
					}
				}
				g.Logf("The Marauders form a villain deck — only its top card is in play")
			}
			return nil
		},
		OnVillainDefeated: func(g *engine.Game, v *engine.Villain) []engine.Message {
			// Tuck the defeated villain under Routed; three banked villains
			// win the game.
			code := data.BaseCode(v.Code)
			name := v.EDef().Name
			g.Delete(v.ID)
			for _, env := range g.Environments {
				if env != nil && data.BaseCode(env.Code) == "40081" {
					env.StoredCards = append(env.StoredCards, engine.Card{ID: g.NextCardID(), Code: code})
					g.Logf("%s is routed (%d villains under Routed)", name, len(env.StoredCards))
					if len(env.StoredCards) >= 3 {
						return []engine.Message{engine.GameOver{Won: true, Reason: engine.Tf("reason.threeMaraudersRouted")}}
					}
				}
			}
			marauderNextVillain(g)
			return nil
		},
		OnMainSchemeMaxed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			if s.Stage < len(s.StageCodes) {
				return []engine.Message{engine.ReplaceMainScheme{Scheme: s.ID}}
			}
			return []engine.Message{engine.GameOver{Won: false, Reason: engine.Tf("reason.mainSchemeCompleted")}}
		},
	})

	// On the Run (the Marauders escaping with Hope).
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "40103",
		Name:             "Marauders — On the Run",
		VillainBases:     []string{"40070a", "40071a", "40072a", "40073a", "40074a", "40075a", "40076a"},
		MainSchemeStages: []string{"40103", "40104"},
		ExtraSets:        []string{"marauders", "on_the_run", "mutant_slayers", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			// One random Marauder villain in play; the villain with the
			// same title (minion) and every other villain leave the game.
			var ids []engine.EntityID
			for id := range g.Villains {
				ids = append(ids, id)
			}
			keep := g.Random(len(ids))
			var keptName string
			for i, id := range ids {
				v := g.Villains[id]
				if v == nil {
					continue
				}
				if i == keep {
					keptName = v.EDef().Name
					continue
				}
				g.Delete(id)
			}
			for _, mn := range g.Minions {
				if mn != nil && keptName != "" && mn.EDef().Name == keptName {
					g.Delete(mn.ID)
				}
			}
			// Hope's Captor, confident side up.
			if v := firstVillain(g); v != nil {
				t := &engine.Attachment{ID: g.NextEntityID(engine.KindAttachment), Code: "40105a", Target: v.ID}
				g.Attachments[t.ID] = t
				v.Attachments = append(v.Attachments, t.ID)
				g.Logf("Hope's Captor attaches to %s (confident)", v.EDef().Name)
			}
			return nil
		},
		OnVillainDefeated: func(g *engine.Game, v *engine.Villain) []engine.Message {
			return []engine.Message{engine.GameOver{Won: true, Reason: engine.Tf("reason.marauderStopped")}}
		},
		OnMainSchemeMaxed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			if s.Stage < len(s.StageCodes) {
				return []engine.Message{engine.ReplaceMainScheme{Scheme: s.ID}}
			}
			return []engine.Message{engine.GameOver{Won: false, Reason: engine.Tf("reason.maraudersEscaped")}}
		},
	})

	// The Unstoppable Juggernaut.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "40121",
		Name:             "Juggernaut — The Unstoppable Juggernaut",
		VillainBases:     []string{"40118"},
		MainSchemeStages: []string{"40121"},
		ExtraSets:        []string{"juggernaut", "hope_summers", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			hopeSummers(g)
			if v := firstVillain(g); v != nil {
				t := &engine.Attachment{ID: g.NextEntityID(engine.KindAttachment), Code: "40122a", Target: v.ID}
				g.Attachments[t.ID] = t
				v.Attachments = append(v.Attachments, t.ID)
				v.Counters++
				v.Tough = true
				g.Logf("Juggernaut's Helmet attaches; 1 momentum counter")
			}
			return nil
		},
		OnMainSchemeMaxed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			// The scheme never completes: clear threat, flip the helmet,
			// gain momentum, Juggernaut attacks everyone.
			s.Threat = 0
			g.Logf("The Unstoppable Juggernaut shrugs off the scheme!")
			var msgs []engine.Message
			if v := firstVillain(g); v != nil {
				for _, a := range g.Attachments {
					if a != nil && a.Code == "40122a" {
						g.Delete(a.ID)
						g.Logf("Juggernaut's Helmet flips (Exposed)")
					}
				}
				v.Counters++
				for _, p := range g.Players {
					msgs = append(msgs, engine.AskAttack{Enemy: v.ID, Player: p.ID})
				}
			}
			return msgs
		},
	})

	// Sinister Intent (Mister Sinister).
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "40139",
		Name:             "Mister Sinister — Sinister Intent",
		VillainBases:     []string{"40136"},
		MainSchemeStages: []string{"40139", "40140", "40141", "40142", "40143"},
		ExtraSets:        []string{"mister_sinister", "flight", "super_strength", "telepathy", "hope_summers", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			// The superpower flagships start set aside; stage 2's reveal
			// attaches them (see nx_sinister). Put Hope into play.
			hopeSummers(g)
			return nil
		},
		OnMainSchemeMaxed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			if s.Stage < len(s.StageCodes) {
				g.Logf("The stage completes — advancing")
				return []engine.Message{engine.ReplaceMainScheme{Scheme: s.ID}}
			}
			return []engine.Message{engine.GameOver{Won: false, Reason: engine.Tf("reason.sinisterEnds")}}
		},
	})

	// Uncontrollable Power (Stryfe).
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "40166",
		Name:             "Stryfe — Uncontrollable Power",
		VillainBases:     []string{"40163"},
		MainSchemeStages: []string{"40166", "40167"},
		ExtraSets:        []string{"stryfe", "hope_summers", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			hopeSummers(g)
			// Stryfe's Grasp starts in play.
			s := spawnSideSchemeCard(g, "40168a", 6*len(g.Players))
			_ = s
			return nil
		},
		OnMainSchemeMaxed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			if s.Stage < len(s.StageCodes) {
				return []engine.Message{engine.ReplaceMainScheme{Scheme: s.ID}}
			}
			return []engine.Message{engine.GameOver{Won: false, Reason: engine.Tf("reason.stryfeUncontrollable")}}
		},
	})
}

func firstVillain(g *engine.Game) *engine.Villain {
	for _, v := range g.Villains {
		if v != nil {
			return v
		}
	}
	return nil
}
