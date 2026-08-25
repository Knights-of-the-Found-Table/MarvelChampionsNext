// Package ironheart registers the Ironheart hero pack: the Version
// upgrade system (progress counters → swap hero sides) and the Lucia von
// Bardas nemesis set. Progress counters live on GrowthCounters.
package ironheart

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() {
	registerIronheart()
	registerNemesis()
}

// progress reads the identity's progress counters.
func progress(p *engine.Player) int { return p.GrowthCounters }

// version reports the current Ironheart version number (from the hero
// side's code order).
func version(p *engine.Player) int {
	// 29001/29002 are the v1/v2 hero sides' base codes; the third side
	// (29003) prints Maximum Efficiency instead of Level Up.
	switch p.HeroCode[:5] {
	case "29001":
		return 1
	case "29002":
		return 2
	case "29003":
		return 3
	}
	return 1
}

// addProgress returns n progress counters on the identity.
func addProgress(p *engine.Player, n int) {
	p.GrowthCounters += n
}

// levelUpAbility builds the Level Up! action for a version side.
func levelUpAbility(nextCode string, tough bool) func(g *engine.Game, p *engine.Player) []engine.Ability {
	return func(g *engine.Game, p *engine.Player) []engine.Ability {
		if p == nil || progress(p) < 6 {
			return nil
		}
		return []engine.Ability{{
			Label: engine.Tf("c.levelUpRemove6ProgressCountersSwapToTheNextVersion"), Type: engine.AbilityAction,
			Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
				p := g.Player(self)
				if p == nil || progress(p) < 6 {
					return nil
				}
				p.GrowthCounters -= 6
				def, ok := engine.DB.Lookup(nextCode)
				if !ok {
					return nil
				}
				p.HeroCode = def.Code
				p.Side = engine.SideHero
				msgs := []engine.Message{engine.ReadyEntity{ID: self}}
				if tough {
					msgs = append(msgs, engine.ToughEntity{Target: self})
				}
				return append(msgs, engine.SwapHeroSide{Player: self, HeroCode: def.Code})
			},
		}}
	}
}

func registerIronheart() {
	// Version 1 → 2.
	engine.RegisterBehavior("29001", &engine.Behavior{
		HeroAbilities: levelUpAbility("29002a", false),
	})
	// Version 2 → 3.
	engine.RegisterBehavior("29002", &engine.Behavior{
		HeroAbilities: levelUpAbility("29003a", true),
	})
	// Version 3: Maximum Efficiency.
	engine.RegisterBehavior("29003", &engine.Behavior{
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			if p == nil || progress(p) <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.maximumEfficiency1ProgressCounterDeal2Damage"), Type: engine.AbilityAction, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(self)
					if p == nil || progress(p) <= 0 {
						return nil
					}
					p.GrowthCounters--
					return cardutil.ChooseEnemy(engine.Tf("c.maximumEfficiencyDeal2DamageToWhichEnemy"),
						func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 2, nil })(
						g, &engine.EventCard{Code: "29003", Owner: p.ID})
				},
			}}
		},
	})

	// Brawn: exhausted → mental resource once per phase.
	engine.RegisterBehavior("29004", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "mental"},
	})

	// Fly Over: 3 threat + 1-2 progress.
	engine.RegisterBehavior("29005", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			msgs := []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(
				engine.Tf("c.flyOverRemove3ThreatFromWhichScheme"), schemePicks(g, 3, p.ID)...)}}
			return append(msgs, engine.AddProgressCounters{Player: p.ID, N: 1})
		},
	})

	// Photon Beam: 4 damage + 1-2 progress.
	engine.RegisterBehavior("29006", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			msgs := cardutil.ChooseEnemy(engine.Tf("c.photonBeamDeal4DamageToWhichEnemy"),
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 4, nil })(g, e)
			return append(msgs, engine.AddProgressCounters{Player: p.ID, N: 1})
		},
	})

	// New and Improved: X = version, choose X different options.
	engine.RegisterBehavior("29007", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			x := min(3, version(p))
			var picks []engine.Choice
			seen := map[string]bool{}
			for _, c := range append(engine.CardList{}, p.Deck...) {
				def := c.Def()
				if def.Code[:2] == "29" && def.Type != "obligation" && !seen[c.Code] {
					seen[c.Code] = true
					picks = append(picks, engine.Choice{Label: engine.S("Search — " + def.Name), Kind: engine.ChoiceCard, CardCode: def.Code}.
						Msgs(engine.TakeDeckCard{Player: p.ID, CardID: c.ID}, engine.ShufflePlayerDeck{Player: p.ID}))
				}
			}
			picks = append(picks,
				engine.Choice{ID: "tough", Label: engine.Tf("c.giveIronheartTough"), Kind: engine.ChoiceLabel}.
					Msgs(engine.ToughEntity{Target: p.ID}),
				engine.Choice{ID: "ready", Label: engine.Tf("c.readyIronheart"), Kind: engine.ChoiceLabel}.
					Msgs(engine.ReadyEntity{ID: p.ID}))
			q := engine.AskN(engine.Tf("c.newAndImprovedChooseOptions"), x, picks...)
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: q}}
		},
	})

	// Sector Scan: look at the encounter top — approximated to a one-shot
	// peek log.
	engine.RegisterBehavior("29008", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if len(g.EncounterDeck) > 0 {
				g.TLogf("c.sectorScanTheTopEncounterCardIs", g.EncounterDeck[0].Def().Name)
			}
			return nil
		},
	})

	// Stroke of Genius: spent-response — plain resource.
	engine.RegisterBehavior("29009", &engine.Behavior{})

	// Ronnie Williams: heal 2 or 1 progress.
	engine.RegisterBehavior("29010", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustRonnieWilliamsHeal2OrAdd1ProgressCounter"), Type: engine.AbilityAction,
				Exhaust: true, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.EOwner())
					if p == nil {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(
						engine.Tf("c.ronnieWilliams"),
						engine.Choice{ID: "heal", Label: engine.Tf("c.heal2DamageFromRiri"), Kind: engine.ChoiceLabel}.
							Msgs(engine.HealEntity{Target: p.ID, N: 2}),
						engine.Choice{ID: "progress", Label: engine.Tf("c.add1ProgressCounter"), Kind: engine.ChoiceLabel}.
							Msgs(engine.AddProgressCounters{Player: p.ID, N: 1}),
					)}}
				},
			}}
		},
	})

	// Tony Stark A.I.: top 2 → 1 hand 1 discard.
	engine.RegisterBehavior("29011", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustTonyStarkAITop21ToHand1Discarded"), Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.EOwner())
					if p == nil || len(p.Deck) < 2 {
						return nil
					}
					top := append(engine.CardList{}, p.Deck[:2]...)
					p.Deck = p.Deck[2:]
					p.Hand = append(p.Hand, top[0])
					p.Discard = append(p.Discard, top[1])
					g.TLogf("c.tonyStarkAIToHandDiscarded", top[0].Def().Name, top[1].Def().Name)
					return nil
				},
			}}
		},
	})

	// Photon Blasters / Propulsion Jets: +2 HP; version-scaled shots.
	engine.RegisterBehavior("29012", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if p := g.Player(e.EOwner()); p != nil {
				p.MaxHP += 2
			}
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustPhotonBlastersDealVersionNumberDamage"), Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					p := g.Player(u.EOwner())
					if p == nil {
						return nil
					}
					n := version(p)
					return cardutil.ChooseEnemy(engine.Tf("c.photonBlastersDealDamageToWhichEnemy"),
						func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return n, nil })(
						g, &engine.EventCard{Code: "29012", Owner: p.ID})
				},
			}}
		},
	})
	engine.RegisterBehavior("29013", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if p := g.Player(e.EOwner()); p != nil {
				p.MaxHP += 2
			}
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustPropulsionJetsRemoveVersionNumberThreat"), Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					p := g.Player(u.EOwner())
					if p == nil {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(
						engine.Tf("c.propulsionJetsRemoveThreatFromWhichScheme"), schemePicks(g, version(p), p.ID)...)}}
				},
			}}
		},
	})

	// Cloud 9: a player's Aerial characters +1 THW this phase.
	engine.RegisterBehavior("29014", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustCloud9APlayerSAerialCharacters1Thw"), Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					var picks []engine.Choice
					for _, q := range g.Players {
						msgs := []engine.Message{engine.ApplyStatBonus{Target: q.ID, THW: 1}}
						for _, id := range q.Allies {
							if a := g.Allies[id]; a != nil && a.EDef().HasTrait("aerial") {
								msgs = append(msgs, engine.AllyStatBonus{Ally: id, THW: 1})
							}
						}
						picks = append(picks, engine.Choice{Label: engine.S(q.Name), Kind: engine.ChoiceTarget, SourceID: q.ID}.
							Msgs(msgs...))
					}
					return []engine.Message{engine.AskQuestion{Player: g.Entity(self).EOwner(),
						Question: engine.Ask(engine.Tf("c.cloud9WhichPlayer"), picks...)}}
				},
			}}
		},
	})

	// Falcon: energy-spending ready of another champion.
	engine.RegisterBehavior("29015", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.spendEnergyReadyAnotherChampionCharacter"), Type: engine.AbilityAction,
				Cost: 1, CostIcons: "energy:1", HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					a := g.Allies[self]
					p := g.Player(a.EOwner())
					if p == nil {
						return nil
					}
					var picks []engine.Choice
					for _, id := range p.Allies {
						if q := g.Allies[id]; q != nil && q.Exhausted && id != self && q.EDef().HasTrait("champion") {
							picks = append(picks, engine.Choice{Label: engine.S(q.EDef().Name), Kind: engine.ChoiceTarget, SourceID: id, CardCode: q.Code}.
								Msgs(engine.ReadyEntity{ID: id}))
						}
					}
					if p.Exhausted && g.EntityHasTrait(p.ID, "champion") {
						picks = append(picks, engine.Choice{Label: engine.S(p.Name + " (identity)"), Kind: engine.ChoiceTarget, SourceID: p.ID}.
							Msgs(engine.ReadyEntity{ID: p.ID}))
					}
					if len(picks) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID,
						Question: engine.Ask(engine.Tf("c.readyWhichChampion"), picks...)}}
				},
			}}
		},
	})

	// Patriot: a champion gets +1 to all basic powers this round.
	engine.RegisterBehavior("29016", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			for _, q := range g.Players {
				for _, id := range q.Allies {
					if a := g.Allies[id]; a != nil && a.EDef().HasTrait("champion") {
						picks = append(picks, engine.Choice{Label: engine.S(a.EDef().Name), Kind: engine.ChoiceTarget, SourceID: id, CardCode: a.Code}.
							Msgs(engine.AllyStatBonus{Ally: id, THW: 1, ATK: 1}))
					}
				}
			}
			if g.EntityHasTrait(p.ID, "champion") {
				picks = append(picks, engine.Choice{Label: engine.S(p.Name + " (identity)"), Kind: engine.ChoiceTarget, SourceID: p.ID}.
					Msgs(engine.ApplyStatBonus{Target: p.ID, THW: 1, ATK: 1, DEF: 1}))
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.patriotEmpowerWhichChampion"), picks...)}}
		},
	})

	// Go All Out / Push Ahead: THW+ATK+DEF totals.
	engine.RegisterBehavior("29017", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			n := max(0, p.ThwartStat(g)) + max(0, p.AttackStat(g)) + max(0, p.DefenseStat(g))
			return cardutil.ChooseEnemy(engine.Tf("c.goAllOutDealDamageToWhichEnemy"),
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return n, nil })(g, e)
		},
	})
	engine.RegisterBehavior("29018", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			n := max(0, p.ThwartStat(g)) + max(0, p.AttackStat(g)) + max(0, p.DefenseStat(g))
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(
				engine.Tf("c.pushAheadRemoveThreatFromWhichScheme"), schemePicks(g, n, p.ID)...)}}
		},
	})

	// Morale Boost: a hero +1/+1/+1 this round (phase here).
	engine.RegisterBehavior("29019", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var picks []engine.Choice
			for _, q := range g.Players {
				q := q
				picks = append(picks, engine.Choice{Label: engine.S(q.Name), Kind: engine.ChoiceTarget, SourceID: q.ID}.
					Msgs(engine.ApplyStatBonus{Target: q.ID, THW: 1, ATK: 1, DEF: 1}))
			}
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(),
				Question: engine.Ask(engine.Tf("c.moraleBoostWhichHero2"), picks...)}}
		},
	})

	// R&D Facility: 3 research counters; +1 THW/+1 ATK to a character.
	engine.RegisterBehavior("29020", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 3}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			if s == nil || s.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustRDFacilityCounterACharacter1Thw1Atk"), Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.EOwner())
					if p == nil {
						return nil
					}
					var picks []engine.Choice
					for _, q := range g.Players {
						picks = append(picks, engine.Choice{Label: engine.S(q.Name), Kind: engine.ChoiceTarget, SourceID: q.ID}.
							Msgs(engine.AddEntityCounter{ID: self, N: -1},
								engine.ApplyStatBonus{Target: q.ID, THW: 1, ATK: 1}))
						for _, id := range q.Allies {
							if a := g.Allies[id]; a != nil {
								picks = append(picks, engine.Choice{Label: engine.S(a.EDef().Name), Kind: engine.ChoiceTarget, SourceID: id, CardCode: a.Code}.
									Msgs(engine.AddEntityCounter{ID: self, N: -1},
										engine.AllyStatBonus{Ally: id, THW: 1, ATK: 1}))
							}
						}
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID,
						Question: engine.Ask(engine.Tf("c.empowerWhichCharacter"), picks...)}}
				},
			}}
		},
	})

	// The Power of Leadership reprint.
	engine.RegisterBehavior("29021", &engine.Behavior{})

	// Agent 13: ready a SHIELD support after attacking/thwarting.
	engine.RegisterBehavior("29022", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			var hit bool
			switch w := msg.(type) {
			case engine.AllyAttackWindow:
				hit = w.Ally == e.EID()
			case engine.AllyThwartWindow:
				hit = w.Ally == e.EID()
			}
			if !hit {
				return nil
			}
			a := g.Allies[e.EID()]
			p := g.Player(a.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			for _, id := range p.Supports {
				if s := g.Supports[id]; s != nil && s.Exhausted && s.EDef().HasTrait("s.h.i.e.l.d.") {
					picks = append(picks, engine.Choice{Label: engine.S(s.EDef().Name), Kind: engine.ChoiceTarget, SourceID: id, CardCode: s.Code}.
						Msgs(engine.ReadyEntity{ID: id}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.agent13ReadyWhichShieldSupport"), picks...)}}
		},
	})

	// Snowguard: shift counters (mode 1 approximated — +3 ATK).
	engine.RegisterBehavior("29023", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 1}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if ac, ok := msg.(engine.AddEntityCounter); ok && ac.ID == e.EID() && ac.N > 0 {
				if a := g.Allies[e.EID()]; a != nil {
					a.PermATK += 3 * ac.N
				}
			}
			return nil
		},
	})

	// Vivian: blank a card's text — not enforceable; marker behavior.
	engine.RegisterBehavior("29024", &engine.Behavior{})

	// "Go for Champions!": champions cannot take damage this round —
	// approximated to a one-shot prevention marker.
	engine.RegisterBehavior("29025", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			g.UsedThisRound["champions-shield"] = true
			g.TLogf("c.goForChampionsChampionsCannotTakeDamageThisRound")
			return nil
		},
	})

	// Helicarrier reprint.
	if b := engine.LookupBehavior("01092"); b != nil {
		engine.RegisterBehavior("29026", b)
	}

	// Ingenuity: mental resource.
	engine.RegisterBehavior("29027", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "mental"},
	})

	// A Minor Setback obligation.
	engine.RegisterBehavior("29028", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			if progress(p) > 0 {
				p.GrowthCounters--
				return []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
			}
			return []engine.Message{
				engine.DealEncounterToPlayer{Player: p.ID},
				engine.ObligationResolve{Player: p.ID, Card: card},
			}
		},
	})

	// Bombshell / Wasp (champions): keyword riders approximated.
	engine.RegisterBehavior("29033", &engine.Behavior{})
	engine.RegisterBehavior("29034", &engine.Behavior{})

	// Pinpoint: placement-replacement — not modeled.
	engine.RegisterBehavior("29035", &engine.Behavior{})
}

func registerNemesis() {
	// Rule by Force: hazard while Lucia lives, acceleration otherwise
	// (dynamic switch approximated to acceleration always).
	engine.RegisterBehavior("29029", &engine.Behavior{})

	// Lucia von Bardas: +1/+1 while tough; regains tough each villain
	// phase.
	engine.RegisterBehavior("29030", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.ToughEntity{Target: e.EID()}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.ToughEntity); ok {
				if mn := g.Minions[e.EID()]; mn != nil {
					mn.AttackVal++
					mn.SchemeVal++
				}
			}
			if _, ok := msg.(engine.EndPhase); ok {
				if mn := g.Minions[e.EID()]; mn != nil {
					return []engine.Message{engine.ToughEntity{Target: e.EID()}}
				}
			}
			return nil
		},
	})

	// Cyborg Tech: attach to the most-traits minion (+3 HP, retaliate).
	engine.RegisterBehavior("29031", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			var best *engine.Minion
			traits := -1
			for _, mn := range g.Minions {
				if n := len(mn.EDef().Traits); n > traits {
					best, traits = mn, n
				}
			}
			if best == nil {
				g.Delete(t.ID)
				g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: t.Code})
				return []engine.Message{engine.RevealNextEncounter{Player: cardutil.FirstPlayerID(g)}}
			}
			t.Target = best.ID
			best.MaxHP += 3
			return nil
		},
	})

	// Political Retribution.
	engine.RegisterBehavior("29032", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var msgs []engine.Message
			fired := false
			for _, mn := range g.Minions {
				if mn.Code[:5] == "29030" {
					fired = true
					msgs = append(msgs, engine.SchemeThreat{Scheme: mainScheme(g), N: mn.SchemeVal, Source: mn.ID})
					break
				}
			}
			for id, s := range g.SideSchemes {
				if s.Code[:5] == "29029" {
					fired = true
					msgs = append(msgs, engine.SchemeThreat{Scheme: id, N: 3, Source: t.ID})
					break
				}
			}
			if !fired {
				msgs = append(msgs, engine.RevealNextEncounter{Player: p.ID})
			}
			return msgs
		},
	})

	// Feedback Loop: threat per energy resource in hand + in play.
	engine.RegisterBehavior("29036", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			n := 0
			for _, p := range g.Players {
				n += energyResources(g, p)
			}
			if n > 0 {
				return []engine.Message{engine.SchemeThreat{Scheme: e.EID(), N: n, Source: e.EID()}}
			}
			return nil
		},
	})

	// Zzzax: +X/+X per energy resource the engaged player controls.
	engine.RegisterBehavior("29037", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return nil
			}
			p := g.Player(mn.EngagedWith)
			if p == nil {
				return nil
			}
			x := min(energyResources(g, p), 5)
			mn.AttackVal += x
			mn.MaxHP += x
			return nil
		},
	})

	// Haywire / Air Static: energy-count penalties + removal.
	engine.RegisterBehavior("29038", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, p := range g.Players {
				t.Target = p.ID
				break
			}
			return nil
		},
		Abilities: energyRemoval(),
	})
	engine.RegisterBehavior("29039", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return nil
		},
		Abilities: energyRemoval(),
	})

	// Zzzap!: indirect damage per energy resource in hand.
	engine.RegisterBehavior("29040", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			n := 0
			for _, c := range p.Hand {
				for _, r := range c.Def().Resources {
					if r == "energy" {
						n++
					}
				}
			}
			if n == 0 {
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: n, Source: t.ID}}
		},
	})
}

// ---- helpers ----

// energyResources counts printed energy resources in hand plus on
// controlled permanents.
func energyResources(g *engine.Game, p *engine.Player) int {
	n := 0
	for _, c := range p.Hand {
		for _, r := range c.Def().Resources {
			if r == "energy" {
				n++
			}
		}
	}
	return n
}

func energyRemoval() func(g *engine.Game, e engine.Entity) []engine.Ability {
	return func(g *engine.Game, e engine.Entity) []engine.Ability {
		return []engine.Ability{{
			Label: engine.Tf("c.discardAnEnergyResourceCardOrTake2DamageDiscardThis"), Type: engine.AbilityAction,
			HeroOnly: true,
			Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
				a := g.Attachments[self]
				env := g.Environments[self]
				pid := engine.PlayerID("")
				if a != nil {
					pid = a.Target
				} else if env != nil {
					pid = g.ActiveTurn
				}
				p := g.Player(pid)
				if p == nil {
					return nil
				}
				// Default: take 2 damage.
				msgs := []engine.Message{engine.DamageEntity{Target: pid, Damage: 2, Source: self}}
				for _, c := range p.Hand {
					for _, r := range c.Def().Resources {
						if r == "energy" {
							msgs = []engine.Message{
								engine.DiscardCards{Player: pid, Cards: engine.CardList{c}},
							}
							break
						}
					}
					if len(msgs) == 1 && msgs[0].(engine.DamageEntity).Damage != 2 {
						break
					}
				}
				g.Delete(self)
				if a != nil {
					g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: a.Code})
				}
				return msgs
			},
		}}
	}
}

func mainScheme(g *engine.Game) engine.EntityID {
	if g.MainScheme != nil {
		return g.MainScheme.ID
	}
	return ""
}

func schemePicks(g *engine.Game, n int, pid engine.PlayerID) []engine.Choice {
	return cardutil.SchemeChoices(g, func(s engine.EntityID) []engine.Message {
		return []engine.Message{engine.ThwartScheme{Scheme: s, N: n, Source: pid}}
	})
}

var _ = data.BaseCode
