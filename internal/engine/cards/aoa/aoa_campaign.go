package aoa

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// missionScheme finds the in-play Mission side scheme.
func missionScheme(g *engine.Game) *engine.SideScheme {
	for _, s := range g.SideSchemes {
		if s != nil && s.EDef().HasTrait("Mission") {
			return s
		}
	}
	return nil
}

// missionTeam finds the Mission Team support.
func missionTeam(g *engine.Game) *engine.Support {
	for _, s := range g.Supports {
		if s != nil && engine.BaseCodeOf(s.Code) == "45171" {
			return s
		}
	}
	return nil
}

// makeMissionAttempt resolves a mission attempt (approximation: an
// exhausted ally leads the attempt; the scheme gains an attempt counter
// and allies at the mission take a damage).
func makeMissionAttempt(g *engine.Game, p *engine.Player) []engine.Message {
	s := missionScheme(g)
	if s == nil {
		g.TLogf("c.noMissionInPlay")
		return nil
	}
	s.Counters++
	g.TLogf("c.missionAttempt4", s.Counters, s)
	var msgs []engine.Message
	for _, id := range p.Allies {
		if a := g.Allies[id]; a != nil && !a.Exhausted {
			msgs = append(msgs, engine.ExhaustEntity{ID: id}, engine.DamageEntity{Target: id, Damage: 1, Source: s.ID})
			break
		}
	}
	if s.Counters >= 4 {
		g.TLogf("c.theMissionSucceeds")
		if mt := missionTeam(g); mt != nil {
			g.Delete(mt.ID)
			g.TLogf("c.missionTeamLeavesTheGame")
		}
		msgs = append(msgs, engine.ThwartScheme{Scheme: s.ID, N: s.Threat, Source: p.ID})
	}
	return msgs
}

func registerAoaCampaign() {
	// 45164 Agent of Apocalypse: join the mission or attack.
	engine.RegisterBehavior("45164", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return nil
			}
			p := g.Player(mn.EngagedWith)
			if p == nil {
				return nil
			}
			// "Add to the mission area" is approximated as the minion
			// leaving play (it joins the mission board).
			return []engine.Message{engine.AskQuestion{
				Player: p.ID,
				Question: engine.Ask(engine.Tf("c.agentOfApocalypseChoose"),
					engine.Choice{ID: "mission", Label: engine.Tf("c.addItToTheMissionArea"), Kind: engine.ChoiceLabel}.
						Msgs(engine.MinionDefeated{MinionID: mn.ID}),
					engine.Choice{ID: "attack", Label: engine.Tf("c.itActivatesAgainstYou"), Kind: engine.ChoiceLabel}.
						Msgs(engine.MinionActivates{MinionID: mn.ID, Player: p.ID}),
				),
			}}
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if p := g.Player(cardutil.FirstPlayerID(g)); p != nil {
				for _, id := range p.Allies {
					return []engine.Message{
						engine.DamageEntity{Target: id, Damage: 1, Source: engine.EntityID("")},
						engine.DealBoost{Enemy: boostEnemyID(g)},
					}
				}
			}
			return nil
		},
	})

	// 45165 Worldwide Crisis: threaten the mission or eat damage + surge.
	engine.RegisterBehavior("45165", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			return []engine.Message{engine.AskQuestion{
				Player: p.ID,
				Question: engine.Ask(engine.Tf("c.worldwideCrisisChoose"),
					engine.Choice{ID: "threat", Label: engine.Tf("c.place3ThreatOnTheMission"), Kind: engine.ChoiceLabel}.
						Msgs(engine.ApplySchemeThreat{Scheme: missionID(g), N: 3, Source: t.ID}),
					engine.Choice{ID: "dmg", Label: engine.Tf("c.take1DamageAndSurge"), Kind: engine.ChoiceLabel}.
						Msgs(engine.DamageEntity{Target: p.ID, Damage: 1, Source: t.ID}, engine.RevealNextEncounter{Player: p.ID}),
				),
			}}
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return []engine.Message{
				engine.ApplySchemeThreat{Scheme: missionID(g), N: 1, Source: engine.EntityID("")},
				engine.DealBoost{Enemy: boostEnemyID(g)},
			}
		},
	})

	// 45166a-45170a Mission side schemes: attempt counters live in
	// makeMissionAttempt; their defeat shuffles mission cards home
	// (approximated as a plain defeat).
	for _, code := range []string{"45166", "45167", "45168", "45169", "45170"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	// 45171a Mission Team: cost reduction or mission attempts.
	engine.RegisterBehavior("45171", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.missionTeamReduceTheNextAllySCostBy2"), Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.CostDiscountApply{Player: g.Supports[self].Owner, Amount: 2}}
				},
			}, {
				Label: engine.Tf("c.missionTeamMakeAMissionAttempt"), Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return makeMissionAttempt(g, g.Player(g.Supports[self].Owner))
				},
			}}
		},
	})

	// 45172-45175 campaign allies: "after this enters your hand" riders
	// have no engine hook (cards enter hands synchronously); approximated
	// as play-time effects.
	engine.RegisterBehavior("45172", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			a := g.Allies[e.EID()]
			p := g.Player(a.Owner)
			if p == nil || g.MainScheme == nil {
				return nil
			}
			g.TLogf("c.destinyForesees2ThreatRemovedFromTheMainScheme")
			return []engine.Message{engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 2, Source: p.ID}}
		},
	})
	engine.RegisterBehavior("45173", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			a := g.Allies[e.EID()]
			p := g.Player(a.Owner)
			if p == nil {
				return nil
			}
			for id := range g.Villains {
				return []engine.Message{engine.DamageEntity{Target: id, Damage: 2, Source: p.ID}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("45174", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for id := range g.Villains {
				return []engine.Message{engine.ConfuseEntity{Target: id}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("45175", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			a := g.Allies[e.EID()]
			if a == nil {
				return nil
			}
			return []engine.Message{engine.ToughEntity{Target: a.Owner}}
		},
	})

	// 45176 Desperate Measures: ally stat glue.
	engine.RegisterBehavior("45176", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			u := g.Upgrades[e.EID()]
			p := g.Player(u.Owner)
			if p == nil || len(p.Allies) == 0 {
				return nil
			}
			var choices []engine.Choice
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a == nil {
					continue
				}
				choices = append(choices, engine.Choice{
					ID: "ally-" + id.String(), Label: engine.S(a.EDef().Name), Kind: engine.ChoiceTarget,
				}.Msgs(engine.AttachUpgrade{ID: u.ID, Target: id, ATK: 1, THW: 1, MaxHP: 1}))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.desperateMeasuresAttachTo"), choices...),
			}}
		},
	})

	// 45177 North American Sea Wall: villain undamageable while it lives
	// (engine damage check); boost self-deals facedown.
	engine.RegisterBehavior("45177", &engine.Behavior{
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if p := g.Player(cardutil.FirstPlayerID(g)); p != nil {
				p.EncounterDown = append(p.EncounterDown, card)
			}
			return nil
		},
	})

	// 45178 Panicked Refugees: reveal on entry (approximated at
	// resolution) + draw.
	engine.RegisterBehavior("45178", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			return []engine.Message{
				engine.ObligationResolve{Player: p.ID, Card: card},
				engine.DrawCards{Player: p.ID, N: 1},
			}
		},
	})

	registerOverseers()
}

func missionID(g *engine.Game) engine.EntityID {
	if s := missionScheme(g); s != nil {
		return s.ID
	}
	return ""
}

func boostEnemyID(g *engine.Game) engine.EntityID {
	for id := range g.Villains {
		return id
	}
	return ""
}

func registerOverseers() {
	// 45179a-45183a Overseers: immune while another minion stands; the
	// mission-response riders are approximated away.
	overseer := func(code string) {
		engine.RegisterBehavior(code, &engine.Behavior{
			MinionDamageable: func(g *engine.Game, m *engine.Minion, damage int) bool {
				for _, o := range g.Minions {
					if o != nil && o.ID != m.ID {
						g.TLogf("c.theOverseerCannotTakeDamageWhileAnotherMinionStands")
						return false
					}
				}
				return true
			},
		})
	}
	for _, code := range []string{"45179", "45180", "45181", "45182", "45183"} {
		overseer(code)
	}
}
