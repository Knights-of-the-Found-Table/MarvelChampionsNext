// Package goblinfooblin registers The Green Goblin pack content: the
// Mutagen Formula and Risky Business scenarios with their encounter sets.
package goblinfooblin

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

func init() {
	registerRemainingGob()
	registerMutagenFormula()
	registerRiskyBusiness()
	registerGoblinEncounterCards()
}

// registerMutagenFormula registers scenario 1 of the pack.
func registerMutagenFormula() {
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "02017",
		Name:             "Green Goblin — The Mutagen Formula",
		VillainBases:     []string{"02014"},
		MainSchemeStages: []string{"02017b", "02018b"},
		ExtraSets:        []string{"goblin_gimmicks", "a_mess_of_things", "standard"},
		OnMainSchemeMaxed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			if s.Stage == 1 {
				// Stage 1 completes: the mutagen is unleashed and the
				// cloud forms (the scheme replaces; the villain does
				// not advance).
				g.Logf("The mutagen is unleashed!")
				msgs := []engine.Message{engine.ReplaceMainScheme{Scheme: s.ID}}
				return msgs
			}
			return []engine.Message{engine.GameOver{Won: false, Reason: engine.Tf("reason.mutagenCloud")}}
		},
	})

	// Stage I: after Green Goblin attacks and damages you, +1 threat.
	engine.RegisterBehavior("02014", &engine.Behavior{
		React: goblinAttackRider(1),
		VillainStage: func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
			return dealIndirectToAll(g, 0) // stage 1 has no reveal rider
		},
	})
	// Stage II: when revealed, deal 2 encounter cards to each player.
	engine.RegisterBehavior("02015", &engine.Behavior{
		React:        goblinAttackRider(1),
		VillainStage: dealEncounterCardsPerPlayer(2),
	})
	// Stage III: 3 encounter cards; +2 threat per attack.
	engine.RegisterBehavior("02016", &engine.Behavior{
		React:        goblinAttackRider(2),
		VillainStage: dealEncounterCardsPerPlayer(3),
	})
}

// registerRiskyBusiness registers scenario 2 of the pack: the
// Norman Osborn <-> Green Goblin flip driven by the Criminal Enterprise /
// State of Madness environment.
func registerRiskyBusiness() {
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "02004",
		Name:             "Green Goblin — Risky Business",
		VillainBases:     []string{"02001"},
		MainSchemeStages: []string{"02004b", "02005b"},
		ExtraSets:        []string{"power_drain", "running_interference", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			// The villain starts on his Norman Osborn side; Criminal
			// Enterprise enters play with 2 infamy counters per player.
			msgs := []engine.Message{engine.FlipVillainPersona{FlipToNorman: true}}
			env := &engine.Environment{
				ID:       g.NextEntityID("environment"),
				Code:     "02006a",
				Counters: 2 * len(g.Players),
			}
			g.AddEnvironment(env)
			g.Logf("Criminal Enterprise enters play with %d infamy counters", env.Counters)
			return msgs
		},
		OnMainSchemeMaxed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			if s.Stage == 1 {
				// Hostile Takeover completes: stockpile infamy, mill
				// decks, then move to Corporate Acquisition (scheme
				// only; the villain does not advance).
				msgs := []engine.Message{engine.ReplaceMainScheme{Scheme: s.ID}}
				if env := g.EnvironmentByCode("02006a", "02006b"); env != nil {
					env.Counters += len(g.Players)
					g.Logf("Criminal Enterprise gains %d infamy counters", len(g.Players))
				}
				for _, p := range g.Players {
					msgs = append(msgs, engine.MillPlayerDeck{Player: p.ID, N: 2})
				}
				return msgs
			}
			return []engine.Message{engine.GameOver{Won: false, Reason: engine.Tf("reason.osbornAcquisition")}}
		},
	})

	// Norman Osborn side (stage I): attacks become +1 infamy instead;
	// damage removes infamy; scheme normally.
	registerPersona("02001", 1)
	registerPersona("02002", 2)
	registerPersona("02003", 3)
}

// registerPersona wires the flip mechanics for one Risky Business stage.
// goblinCode/normanCode are the b/a sides of the stage card.
func registerPersona(stageBase string, perStage int) {
	goblinCode := stageBase + "b"
	normanCode := stageBase + "a"

	// Norman side: attack -> infamy; damage -> remove infamy (and flip
	// when empty).
	engine.RegisterBehavior(normanCode, &engine.Behavior{
		VillainActivate: func(g *engine.Game, v *engine.Villain, p *engine.Player) []engine.Message {
			if p.IsHero() {
				// Norman never attacks; he gains infamy instead.
				if env := g.EnvironmentByCode("02006a", "02006b"); env != nil && env.Code == "02006a" {
					env.Counters += perStage
					g.Logf("Norman Osborn plots: +%d infamy on Criminal Enterprise", perStage)
					return nil
				}
				return nil
			}
			// Norman schemes with his own scheme value.
			g.Logf("Norman Osborn schemes against %s", p.Name)
			g.Push(engine.DealBoost{Enemy: v.ID})
			g.Push(engine.RevealBoost{Enemy: v.ID})
			return []engine.Message{engine.ApplyVillainScheme{VillainID: v.ID, Player: p.ID}}
		},
		VillainDamageable: func(g *engine.Game, v *engine.Villain, damage int) bool {
			env := g.EnvironmentByCode("02006a", "02006b")
			if env == nil || env.Code != "02006a" {
				return true // no Criminal Enterprise: damage applies
			}
			// Damage removes infamy; when empty, flip to Green Goblin.
			toRemove := damage
			if toRemove <= 0 {
				toRemove = 1
			}
			if env.Counters >= toRemove {
				env.Counters -= toRemove
				g.Logf("Damage converted: -%d infamy from Criminal Enterprise (%d left)", toRemove, env.Counters)
			} else {
				env.Counters = 0
			}
			if env.Counters == 0 {
				flipToGoblin(g, v, env)
			}
			return false
		},
	})

	// Green Goblin side: schemes remove madness counters from State of
	// Madness (flipping back to Norman when empty); attacks resolve
	// normally with the stage reveal riders.
	engine.RegisterBehavior(goblinCode, &engine.Behavior{
		VillainActivate: func(g *engine.Game, v *engine.Villain, p *engine.Player) []engine.Message {
			if p.IsHero() {
				// Normal attack flow.
				if v.Stunned {
					v.Stunned = false
					g.Logf("Green Goblin is stunned; attack canceled")
					return nil
				}
				g.Logf("Green Goblin attacks %s", p.Name)
				g.Push(engine.DealBoost{Enemy: v.ID})
				g.Push(engine.RevealBoost{Enemy: v.ID})
				return []engine.Message{engine.AskQuestion{
					Player:   p.ID,
					Question: g.AttackQuestion(v.ID, v.AttackVal, p, ""),
				}}
			}
			// Goblin would scheme: remove madness counters instead.
			env := g.EnvironmentByCode("02006a", "02006b")
			if env != nil && env.Code == "02006b" && env.Counters > 0 {
				n := 1
				if perStage == 3 {
					n = 2
				}
				env.Counters -= n
				if env.Counters < 0 {
					env.Counters = 0
				}
				g.Logf("Madness consumes the Goblin: -%d madness counters (%d left)", n, env.Counters)
				if env.Counters == 0 {
					flipToNorman(g, v, env)
				}
				return nil
			}
			// No State of Madness: scheme for real.
			g.Logf("Green Goblin schemes against %s", p.Name)
			g.Push(engine.DealBoost{Enemy: v.ID})
			g.Push(engine.RevealBoost{Enemy: v.ID})
			return []engine.Message{engine.ApplyVillainScheme{VillainID: v.ID, Player: p.ID}}
		},
		// When Revealed (i.e. on flip / stage entry): indirect damage.
		VillainStage: func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
			return dealIndirectToAll(g, 3)
		},
	})
}

func flipToGoblin(g *engine.Game, v *engine.Villain, env *engine.Environment) {
	v.Code = v.Code[:5] + "b"
	def := v.EDef()
	v.SchemeVal = derefOr(def.Scheme, 0)
	v.AttackVal = derefOr(def.Attack, 0)
	v.Tough = def.HasKeyword("Toughness")
	env.Code = "02006b"
	env.Counters = 2 * len(g.Players)
	g.Logf("Norman snaps: the Green Goblin emerges! (%d madness counters)", env.Counters)
	for _, msg := range dealIndirectToAll(g, 3) {
		g.Push(msg)
	}
}

func flipToNorman(g *engine.Game, v *engine.Villain, env *engine.Environment) {
	v.Code = v.Code[:5] + "a"
	def := v.EDef()
	v.SchemeVal = derefOr(def.Scheme, 0)
	v.AttackVal = derefOr(def.Attack, 0)
	env.Code = "02006a"
	env.Counters = 2 * len(g.Players)
	g.Logf("The madness subsides: Norman Osborn regains control. (%d infamy counters)", env.Counters)
}

func derefOr(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}

// goblinAttackRider builds the "after Green Goblin attacks and damages
// you, place N threat" forced response.
func goblinAttackRider(n int) func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
	return func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.WindowAfterEnemyAttacked)
		if !ok || m.Enemy != e.EID() {
			return nil
		}
		if g.MainScheme == nil {
			return nil
		}
		return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: n, Source: e.EID()}}
	}
}

// dealEncounterCardsPerPlayer builds the stage When-Revealed effect.
func dealEncounterCardsPerPlayer(n int) func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
	return func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
		var msgs []engine.Message
		for _, p := range g.Players {
			for i := 0; i < n; i++ {
				msgs = append(msgs, engine.DealEncounterToPlayer{Player: p.ID})
			}
		}
		return msgs
	}
}

// dealIndirectToAll damages each player in hero form (approximation of
// indirect damage assignment).
func dealIndirectToAll(g *engine.Game, n int) []engine.Message {
	if n <= 0 {
		return nil
	}
	var msgs []engine.Message
	for _, p := range g.Players {
		if p.IsHero() {
			msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: n, Source: EntityIDZero()})
		}
	}
	return msgs
}

func EntityIDZero() engine.EntityID { return engine.EntityID("scenario") }
