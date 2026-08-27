package msmarvel

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// registerAspectCards installs the pack's Protection, basic and other
// aspect cards (05012-05024, 05030-05033).
func registerAspectCards() {
	registerNova()
	registerGetBehindMe()
	registerPreemptiveStrike()
	registerTackle()
	registerPowerOfProtection()
	registerEnergyBarrier()
	registerLockjaw()
	registerBasicResources()
	registerAvengersMansion()
	registerEndurance()
	registerEnhancedReflexes()
	registerMelee()
	registerConcussiveBlow()
	registerMoraleBoost()
	registerDownTime()
}

// 05012 Nova: interrupt — when an enemy initiates an attack against you,
// spend a [energy] resource → deal 2 damage to that enemy (approximation:
// the energy icon is checked in hand, the discard happens on confirm).
func registerNova() {
	engine.RegisterBehavior("05012", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil || !p.IsHero() {
				return nil
			}
			hasEnergy := false
			for _, c := range p.Hand {
				for _, r := range c.Def().Resources {
					if r == "energy" || r == "wild" {
						hasEnergy = true
					}
				}
			}
			if !hasEnergy {
				return nil
			}
			return []engine.Ability{{
				Label:    engine.Tf("c.novaSpendAnEnergyResourceDeal2DamageToTheAttacker"),
				Type:     engine.AbilityTrigger,
				Trigger:  engine.TriggerVillainAttacksYou,
				Exhaust:  true,
				HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					// The attacking enemy: prefer a villain, else a minion
					// (approximation when several enemies are present).
					var atk engine.EntityID
					for id := range g.Villains {
						atk = id
						break
					}
					if atk == "" {
						for id := range g.Minions {
							atk = id
							break
						}
					}
					if atk == "" {
						return nil
					}
					var out []engine.Choice
					for _, c := range p.Hand {
						ok := false
						for _, r := range c.Def().Resources {
							if r == "energy" || r == "wild" {
								ok = true
							}
						}
						if !ok {
							continue
						}
						out = append(out, engine.Choice{
							Label: engine.Tf("c.spendName", c), Kind: engine.ChoiceCard, CardCode: c.Code,
						}.Msgs(
							engine.DiscardCards{Player: p.ID, Cards: engine.CardList{c}},
							engine.DamageEntity{Target: atk, Damage: 2, Source: p.ID},
						))
					}
					if len(out) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{
						Player:   p.ID,
						Question: engine.Ask(engine.Tf("c.novaSpendAnEnergyResource"), out...),
					}}
				},
			}}
		},
	})
}

// 05013 Get Behind Me!: hero interrupt — when a treachery is revealed
// from the encounter deck, cancel its When Revealed effects; the villain
// attacks you instead.
func registerGetBehindMe() {
	engine.RegisterBehavior("05013", &engine.Behavior{
		TreacheryInterrupt: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			if !p.IsHero() {
				return nil
			}
			for id := range g.Villains {
				g.TLogf("c.playsGetBehindMe", p.Name)
				return []engine.Message{engine.VillainActivates{VillainID: id, Player: p.ID}}
			}
			return nil
		},
	})
}

// 05014 Preemptive Strike: cancel all boost icons on the villain's attack;
// deal 1 damage per icon cancelled (approximation: played during the
// defense prompt, which is after boosts are revealed).
func registerPreemptiveStrike() {
	engine.RegisterBehavior("05014", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			v := g.Villains[against]
			if v == nil || v.BoostCount == 0 {
				return engine.Defends{}, nil, false
			}
			n := v.BoostCount
			v.BoostCount = 0
			g.TLogf("c.preemptiveStrikeCancelsBoostIcons", n)
			d := engine.Defends{Defender: p.ID, Against: against, Undefended: true}
			return d, []engine.Message{engine.DamageEntity{Target: against, Damage: n, Source: p.ID}}, true
		},
	})
}

// 05015 Tackle: stun an enemy; if paid with a [physical] resource, deal 3
// damage to it.
func registerTackle() {
	engine.RegisterBehavior("05015", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			physical := false
			if ec, ok := e.(*engine.EventCard); ok {
				physical = ec.Paid.PaidIcon("physical")
			}
			choices := cardutil.EnemyChoices(g, 0, pid, func(target engine.EntityID) []engine.Message {
				msgs := []engine.Message{engine.StunEntity{Target: target}}
				if physical {
					msgs = append(msgs, engine.DamageEntity{Target: target, Damage: 3, Source: pid})
				}
				return msgs
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.tackleStunAnEnemy"), choices...),
			}}
		},
	})
}

// 05016 The Power of Protection: doubles its resource while paying for a
// Protection card — implemented generically in the payment validator.
func registerPowerOfProtection() {
	engine.RegisterBehavior("05016", &engine.Behavior{})
}

// 05017 Energy Barrier: uses (3 reflection counters); when you would take
// damage, remove 1 counter → prevent 1 and deal 1 damage to the enemy
// (approximation: auto-used and reflected at the damage source).
func registerEnergyBarrier() {
	engine.RegisterBehavior("05017", &engine.Behavior{
		DamagePrevention: func(g *engine.Game, u *engine.Upgrade, p *engine.Player, n int) (int, int) {
			use := min(u.Counters, n)
			if use <= 0 {
				return 0, 0
			}
			u.Counters -= use
			g.TLogf("c.energyBarrierPreventsDamageCountersLeft", use, u.Counters)
			return use, use
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if u, ok := e.(*engine.Upgrade); ok {
				u.Counters = 3
				g.TLogf("c.energyBarrierEntersPlayWith3ReflectionCounters")
			}
			return nil
		},
	})
}

// 05018 Lockjaw: you may play him from your discard pile during your turn.
func registerLockjaw() {
	engine.RegisterBehavior("05018", &engine.Behavior{
		PlayableFromDiscard: true,
	})
}

// 05019-05021 Energy/Genius/Strength: plain resource cards, handled
// generically by the data layer.
func registerBasicResources() {
	engine.RegisterBehavior("05019", &engine.Behavior{})
	engine.RegisterBehavior("05020", &engine.Behavior{})
	engine.RegisterBehavior("05021", &engine.Behavior{})
}

// 05022 Avengers Mansion: action — exhaust → choose a player; that player
// draws 1 card.
func registerAvengersMansion() {
	engine.RegisterBehavior("05022", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label:   engine.Tf("c.avengersMansionAPlayerDraws1Card"),
				Type:    engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(e.EOwner())
					if p == nil {
						return nil
					}
					if len(g.Players) == 1 {
						return []engine.Message{engine.DrawCards{Player: p.ID, N: 1}}
					}
					var choices []engine.Choice
					for _, pl := range g.Players {
						choices = append(choices, engine.Choice{
							Label: engine.S(pl.Name), Kind: engine.ChoiceTarget, SourceID: pl.ID,
						}.Msgs(engine.DrawCards{Player: pl.ID, N: 1}))
					}
					return []engine.Message{engine.AskQuestion{
						Player:   p.ID,
						Question: engine.Ask(engine.Tf("c.avengersMansionWhoDraws"), choices...),
					}}
				},
			}}
		},
	})
}

// 05023 Endurance: play under any player's control; that player gets +3
// hit points.
func registerEndurance() {
	engine.RegisterBehavior("05023", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			var choices []engine.Choice
			for _, pl := range g.Players {
				choices = append(choices, engine.Choice{
					Label: engine.S(pl.Name), Kind: engine.ChoiceTarget, SourceID: pl.ID,
				}.Msgs(engine.AttachUpgrade{ID: e.EID(), Target: pl.ID, MaxHP: 3}))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.enduranceWhoGets3HitPoints"), choices...),
			}}
		},
	})
}

// 05024 Enhanced Reflexes: uses (3 energy counters); hero resource —
// exhaust and remove 1 counter → generate an [energy] resource.
func registerEnhancedReflexes() {
	engine.RegisterBehavior("05024", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "energy", HeroOnly: true, UsesCounters: true},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if u, ok := e.(*engine.Upgrade); ok {
				u.Counters = 3
				g.TLogf("c.enhancedReflexesEntersPlayWith3EnergyCounters")
			}
			return nil
		},
	})
}

// 05030 Melee: deal 3 damage to an enemy and 3 damage to another enemy.
func registerMelee() {
	engine.RegisterBehavior("05030", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			ids := cardutil.SortedEnemyIDs(g)
			if len(ids) < 2 {
				return nil
			}
			var first []engine.Choice
			for _, id := range ids {
				enemy := g.Entity(id)
				var second []engine.Choice
				for _, id2 := range ids {
					if id2 == id {
						continue
					}
					enemy2 := g.Entity(id2)
					second = append(second, engine.Choice{
						Label: cardutil.EnemyLabel(enemy2), Kind: engine.ChoiceTarget,
						SourceID: id2, CardCode: enemy2.ECode(),
					}.Msgs(
						engine.DamageEntity{Target: id, Damage: 3, Source: pid},
						engine.DamageEntity{Target: id2, Damage: 3, Source: pid},
					))
				}
				first = append(first, engine.Choice{
					Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: enemy.ECode(),
				}.WithThen(engine.Ask(engine.Tf("c.meleeTheOtherEnemy"), second...)))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.meleeFirstEnemy3DamageEach"), first...),
			}}
		},
	})
}

// 05031 Concussive Blow: confuse an enemy; if paid with a [physical]
// resource, deal 3 damage to it.
func registerConcussiveBlow() {
	engine.RegisterBehavior("05031", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			physical := false
			if ec, ok := e.(*engine.EventCard); ok {
				physical = ec.Paid.PaidIcon("physical")
			}
			choices := cardutil.EnemyChoices(g, 0, pid, func(target engine.EntityID) []engine.Message {
				msgs := []engine.Message{engine.ConfuseEntity{Target: target}}
				if physical {
					msgs = append(msgs, engine.DamageEntity{Target: target, Damage: 3, Source: pid})
				}
				return msgs
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.concussiveBlowConfuseAnEnemy"), choices...),
			}}
		},
	})
}

// 05032 Morale Boost: choose a hero; until the end of the round that hero
// gets +1 THW, +1 ATK and +1 DEF (approximation: expires at end of phase).
func registerMoraleBoost() {
	engine.RegisterBehavior("05032", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			var choices []engine.Choice
			for _, pl := range g.Players {
				pl := pl
				choices = append(choices, engine.Choice{
					Label: engine.S(pl.Name), Kind: engine.ChoiceTarget, SourceID: pl.ID,
				}.Msgs(engine.ApplyStatBonus{
					Target: pl.ID, THW: 1, ATK: 1, DEF: 1,
				}))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.moraleBoostWhichHero"), choices...),
			}}
		},
	})
}

// 05033 Down Time: play under any player's control; that player's
// alter-ego gets +2 REC.
func registerDownTime() {
	engine.RegisterBehavior("05033", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{REC: 2}
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			if len(g.Players) == 1 {
				return []engine.Message{engine.AttachUpgrade{ID: e.EID(), Target: pid}}
			}
			var choices []engine.Choice
			for _, pl := range g.Players {
				choices = append(choices, engine.Choice{
					Label: engine.S(pl.Name), Kind: engine.ChoiceTarget, SourceID: pl.ID,
				}.Msgs(engine.AttachUpgrade{ID: e.EID(), Target: pl.ID}))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.downTimeUnderWhoseControl"), choices...),
			}}
		},
	})
}
