// mg_xmen.go implements the shared Mutant Genesis player cards: the
// X-Men allies, supports, upgrades, events and basic resources that any
// hero can include (32011–32024, 32041–32054, 32066, 32089–32092, 32099).
package mutantgenesis

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() {
	registerXMenShared()
}

// xMenControlled reports whether the identity has the X-Men trait.
func xMenControlled(g *engine.Game, p *engine.Player) bool {
	return g.EntityHasTrait(p.ID, "x-men")
}

// mutOrXMen reports MUTANT or X-MEN identity trait.
func mutOrXMen(g *engine.Game, p *engine.Player) bool {
	return g.EntityHasTrait(p.ID, "x-men") || g.EntityHasTrait(p.ID, "mutant")
}

func registerXMenShared() {
	// 32011 Nightcrawler: his damage-prevention interrupt has no window;
	// he plays as a plain 2/2 X-Men ally.
	engine.RegisterBehavior("32011", &engine.Behavior{})

	// 32012 Polaris: after she enters play, give an X-Men character a
	// tough status card.
	engine.RegisterBehavior("32012", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			if xMenControlled(g, p) {
				choices = append(choices, engine.Choice{
					Label: engine.S("Tough on " + p.Name), Kind: engine.ChoiceTarget, SourceID: p.ID,
				}.Msgs(engine.ToughEntity{Target: p.ID}))
			}
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && a.EDef().HasTrait("x-men") {
					choices = append(choices, engine.Choice{
						Label: engine.S("Tough on " + a.EDef().Name), Kind: engine.ChoiceTarget, SourceID: id, CardCode: a.Code,
					}.Msgs(engine.ToughEntity{Target: id}))
				}
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.polarisGiveAnXMenCharacterAToughStatusCard"), choices...),
			}}
		},
	})

	// 32013 Protective Training: attach to an X-Men ally (max 1 Training
	// per ally): +3 hit points.
	registerTraining("32013", 0, 3)

	// 32015 Bait and Switch: the villain attacks you; remove 4 threat from
	// the main scheme.
	engine.RegisterBehavior("32015", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			msgs := []engine.Message{}
			if g.MainScheme != nil {
				msgs = append(msgs, engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 4, Source: pid})
			}
			if v := activeOrFirstVillain(g); v != nil {
				msgs = append(msgs, engine.AskAttack{Enemy: v.ID, Player: pid, Trigger: engine.TriggerVillainAttacksYou})
			}
			return msgs
		},
	})

	// 32016 Perseverance: after you change form, gain a tough status card.
	engine.RegisterBehavior("32016", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.ToughEntity{Target: e.EOwner()}}
		},
	})

	// 32017 Mutant Protectors: defense event — put an X-Men ally from hand
	// into play exhausted as the defender.
	engine.RegisterBehavior("32017", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			if !xMenControlled(g, p) || !p.IsHero() {
				return false
			}
			for _, c := range p.Hand {
				if c.Def().Type == "ally" && c.Def().HasTrait("x-men") {
					return true
				}
			}
			return false
		},
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			var choices []engine.Choice
			for _, c := range p.Hand {
				if c.Def().Type != "ally" || !c.Def().HasTrait("x-men") {
					continue
				}
				choices = append(choices, engine.Choice{
					Label: engine.S("Defend with " + c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code,
				}.Msgs(engine.AllyEntersPlayFree{Player: p.ID, Card: c}))
			}
			if len(choices) == 0 {
				return engine.Defends{}, nil, false
			}
			return engine.Defends{Defender: p.ID, Against: against},
				[]engine.Message{engine.AskQuestion{
					Player:   p.ID,
					Question: engine.Ask(engine.Tf("c.mutantProtectorsPutAnXMenAllyIntoPlayAsTheDefenderItMustBeEx"), append(choices, cardutil.Skip())...),
				}}, true
		},
	})

	// 32018 Defensive Energy: the spent-on-Defense-event rider is resolved
	// in handlePlayCard (hardcoded draw).
	engine.RegisterBehavior("32018", &engine.Behavior{})

	// 32019 Professor X: on enter — confuse the villain, stun a minion, or
	// ready an X-Men character; discarded at the end of the round.
	engine.RegisterBehavior("32019", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			if v := activeOrFirstVillain(g); v != nil {
				choices = append(choices, engine.Choice{
					Label: engine.S("Confuse " + v.EDef().Name), Kind: engine.ChoiceTarget, SourceID: v.ID,
				}.Msgs(engine.ConfuseEntity{Target: v.ID}))
			}
			for _, id := range cardutil.SortedIDs(g.Minions) {
				if mn := g.Minions[id]; mn != nil {
					choices = append(choices, engine.Choice{
						Label: engine.Tf("c.stun2", cardutil.EnemyLabel(mn)), Kind: engine.ChoiceTarget, SourceID: id,
					}.Msgs(engine.StunEntity{Target: id}))
				}
			}
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && a.EDef().HasTrait("x-men") {
					choices = append(choices, engine.Choice{
						Label: engine.S("Ready " + a.EDef().Name), Kind: engine.ChoiceTarget, SourceID: id,
					}.Msgs(engine.ReadyEntity{ID: id}))
				}
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.professorXChooseOne"), choices...),
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.EndRound); !ok {
				return nil
			}
			if a := g.Allies[e.EID()]; a != nil {
				g.Delete(a.ID)
				p := g.Player(a.Owner)
				if p != nil {
					p.Discard = append(p.Discard, engine.Card{ID: g.NextCardID(), Code: a.Code, Owner: p.ID})
				}
				g.TLogf("c.professorXLeavesPlayAtTheEndOfTheRound")
			}
			return nil
		},
	})

	// 32020 The X-Jet: exhaust → [wild] (approximation: for its owner).
	engine.RegisterBehavior("32020", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "wild"},
	})

	// 32021 Shadow and Steel (Colossus printing) and 32050 (Shadowcat
	// printing): defense event — prevent all damage and deal 4 to the
	// attacker.
	shadowAndSteel := &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			return engine.Defends{Defender: p.ID, Against: against, PreventAll: true},
				[]engine.Message{engine.DamageEntity{Target: against, Damage: 4, Source: p.ID}}, true
		},
	}
	engine.RegisterBehavior("32021", shadowAndSteel)
	engine.RegisterBehavior("32050", shadowAndSteel)

	// 32022-32024 / 32052-32054: basic resources.
	for _, code := range []string{"32022", "32023", "32024", "32052", "32053", "32054"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	// 32041 Wolverine: piercing is cosmetic; heals 1 when your turn begins.
	engine.RegisterBehavior("32041", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if m, ok := msg.(engine.PlayerTurnStart); ok && m.Player == e.EOwner() {
				return []engine.Message{engine.HealEntity{Target: e.EID(), N: 1}}
			}
			return nil
		},
	})

	// 32042 Magik: after she enters play, spend [mental] → shuffle a
	// non-Elite engaged minion into the encounter deck.
	engine.RegisterBehavior("32042", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn == nil || mn.EngagedWith != p.ID || mn.EDef().HasTrait("elite") {
					continue
				}
				choices = append(choices, engine.Choice{
					Label: engine.Tf("c.shuffleIntoTheEncounterDeck", cardutil.EnemyLabel(mn)), Kind: engine.ChoiceTarget, SourceID: id,
				}.Msgs(engine.ShuffleMinionIntoDeck{MinionID: id}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.magikShuffleANonEliteMinionEngagedWithYouIntoTheEncounterDec"), append(choices, cardutil.Skip())...),
			}}
		},
	})

	// 32043 Attack Training: attach to an X-Men ally: +1 ATK, +2 hit
	// points.
	registerTraining("32043", 1, 2)

	// 32044 Gatekeeper: attach to a minion: +2 hit points and patrol (not
	// modeled); when attached minion is defeated, remove 4 threat from the
	// main scheme.
	engine.RegisterBehavior("32044", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			var choices []engine.Choice
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn == nil {
					continue
				}
				choices = append(choices, engine.Choice{
					Label: engine.Tf("c.attachTo2", cardutil.EnemyLabel(mn)), Kind: engine.ChoiceTarget, SourceID: id,
				}.Msgs(engine.AttachUpgrade{ID: u.ID, Target: id, MaxHP: 2}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   e.EOwner(),
				Question: engine.Ask(engine.Tf("c.gatekeeperAttachToAMinion"), choices...),
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionDefeated)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil || m.MinionID != u.AttachTo || g.MainScheme == nil {
				return nil
			}
			g.Delete(u.ID)
			return []engine.Message{engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 4, Source: u.Owner}}
		},
	})

	// 32045 Team Strike: exhaust your hero and any number of X-Men allies
	// → deal their total ATK to an enemy (approximation: one target).
	engine.RegisterBehavior("32045", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return xMenControlled(g, p) && p.IsHero() && !p.Exhausted
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			total := p.AttackStat(g)
			var msgs []engine.Message
			msgs = append(msgs, engine.ExhaustEntity{ID: p.ID})
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a == nil || a.Exhausted || !a.EDef().HasTrait("x-men") {
					continue
				}
				total += a.AttackVal + a.BonusATK + a.PermATK
				msgs = append(msgs, engine.ExhaustEntity{ID: a.ID})
			}
			choices := cardutil.EnemyChoices(g, total, p.ID, func(target engine.EntityID) []engine.Message {
				return []engine.Message{engine.DamageEntity{Target: target, Damage: total, Source: p.ID}}
			})
			if len(choices) == 0 {
				return nil
			}
			return append(msgs, engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.teamStrikeDealDamage", total), choices...),
			})
		},
	})

	// 32046 Toe to Toe: the enemy attacks you; deal it 5 damage.
	engine.RegisterBehavior("32046", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			choices := cardutil.EnemyChoices(g, 5, pid, func(target engine.EntityID) []engine.Message {
				return []engine.Message{
					engine.DamageEntity{Target: target, Damage: 5, Source: pid},
					engine.AskAttack{Enemy: target, Player: pid},
				}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.toeToToeChooseAnEnemyItAttacksYouDealIt5Damage"), choices...),
			}}
		},
	})

	// 32047 Aggressive Energy: the +1 damage rider resolves in
	// handlePlayCard.
	engine.RegisterBehavior("32047", &engine.Behavior{})

	// 32048 Colossus ally: costs 1 less for MUTANT/X-MEN identities;
	// Toughness from data.
	engine.RegisterBehavior("32048", &engine.Behavior{
		CardCost: func(g *engine.Game, p *engine.Player, def *data.CardDef) int {
			if mutOrXMen(g, p) {
				return 1
			}
			return 0
		},
	})

	// 32049 X-Mansion: Alter-Ego action — exhaust → heal 1 from a
	// MUTANT/X-Men character.
	engine.RegisterBehavior("32049", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.xMansionHeal1DamageFromAMutantOrXMenCharacter"), Type: engine.AbilityAction,
				AlterEgoOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(g.ActiveTurn)
					if p == nil {
						return nil
					}
					var choices []engine.Choice
					add := func(id engine.EntityID, label string, mutant bool) {
						if mutant {
							choices = append(choices, engine.Choice{
								Label: engine.S("Heal " + label), Kind: engine.ChoiceTarget, SourceID: id,
							}.Msgs(engine.HealEntity{Target: id, N: 1}))
						}
					}
					add(p.ID, p.Name, mutOrXMen(g, p))
					for _, id := range p.Allies {
						if a := g.Allies[id]; a != nil {
							add(id, a.EDef().Name, a.EDef().HasTrait("x-men") || a.EDef().HasTrait("mutant"))
						}
					}
					if len(choices) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{
						Player:   p.ID,
						Question: engine.Ask(engine.Tf("c.xMansionHeal1Damage"), choices...),
					}}
				},
			}}
		},
	})

	// 32051 Ready to Rumble: after you change form, discard it → ready
	// your hero.
	engine.RegisterBehavior("32051", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.ChangeForm)
			u := g.Upgrades[e.EID()]
			if !ok || m.Player != e.EOwner() || u == nil {
				return nil
			}
			g.Delete(u.ID)
			p := g.Player(u.Owner)
			if p != nil {
				p.Discard = append(p.Discard, engine.Card{ID: g.NextCardID(), Code: u.Code, Owner: p.ID})
			}
			return []engine.Message{engine.ReadyEntity{ID: u.Owner}}
		},
	})

	// 32066 Robert Kelly: the undefended-attack redirect has no window;
	// he plays as a 0-cost ally (the Sabretooth scheme watches his HP).
	engine.RegisterBehavior("32066", &engine.Behavior{})

	// 32089 Rictor: after he attacks, 1 damage to the villain and each
	// minion engaged with you.
	engine.RegisterBehavior("32089", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyAttackWindow)
			if !ok || m.Ally != e.EID() {
				return nil
			}
			a := g.Allies[e.EID()]
			if a == nil {
				return nil
			}
			var msgs []engine.Message
			if v := activeOrFirstVillain(g); v != nil {
				msgs = append(msgs, engine.DamageEntity{Target: v.ID, Damage: 1, Source: a.ID})
			}
			for _, id := range cardutil.SortedIDs(g.Minions) {
				if mn := g.Minions[id]; mn != nil && mn.EngagedWith == a.Owner {
					msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: a.ID})
				}
			}
			return msgs
		},
	})

	// 32090 Boom Boom: after she attacks, a bomb counter; at the end of
	// the player phase each bomb deals 2 damage to every enemy
	// (approximation: a shared bomb pool instead of per-enemy counters).
	engine.RegisterBehavior("32090", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			a := g.Allies[e.EID()]
			if a == nil {
				return nil
			}
			switch m := msg.(type) {
			case engine.AllyAttackWindow:
				if m.Ally != e.EID() {
					return nil
				}
				g.MutantBombCounters++
				g.TLogf("c.boomBoomPlantsABombCounterInPlay", g.MutantBombCounters)
			case engine.EndPhase:
				if m.Phase != engine.PhasePlayer || g.MutantBombCounters <= 0 {
					return nil
				}
				n := g.MutantBombCounters
				g.MutantBombCounters = 0
				var msgs []engine.Message
				for _, id := range cardutil.SortedIDs(g.Villains) {
					if v := g.Villains[id]; v != nil {
						msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 2 * n, Source: a.ID})
					}
				}
				for _, id := range cardutil.SortedIDs(g.Minions) {
					if mn := g.Minions[id]; mn != nil {
						msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 2 * n, Source: a.ID})
					}
				}
				g.TLogf("c.boomBoomSBombsDetonate", n)
				return msgs
			}
			return nil
		},
	})

	// 32091 Cannonball: the conditional consequential rebate is not
	// modeled.
	engine.RegisterBehavior("32091", &engine.Behavior{})

	// 32092 Wolfsbane: piercing is cosmetic.
	engine.RegisterBehavior("32092", &engine.Behavior{})

	// 32099 Warn the Others: obligation — facedown under Operation Zero
	// Tolerance, or exhaust in alter-ego form to discard it.
	engine.RegisterBehavior("32099", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var choices []engine.Choice
			if !p.IsHero() && !p.Exhausted {
				choices = append(choices, engine.Choice{
					ID: "exhaust", Label: engine.Tf("c.exhaustYourIdentityDiscardWarnTheOthers"), Kind: engine.ChoiceLabel,
				}.Msgs(engine.ExhaustEntity{ID: p.ID}, engine.ObligationResolve{Player: p.ID, Card: card}))
			}
			choices = append(choices, engine.Choice{
				ID: "tuck", Label: engine.Tf("c.placeFacedownUnderOperationZeroTolerance"), Kind: engine.ChoiceLabel,
			}.Msgs(engine.ObligationResolve{Player: p.ID, Card: card}))
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.warnTheOthersChoose"), choices...),
			}}
		},
	})

	// 32014 Powerful Punch: defense-window damage has no event hook;
	// played as a plain card it does nothing extra (kept for collection
	// completeness — the attack/defense window is the defense prompt).
	engine.RegisterBehavior("32014", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			return engine.Defends{Defender: p.ID, Against: against},
				[]engine.Message{engine.DamageEntity{Target: against, Damage: 4, Source: p.ID}}, true
		},
	})
}

// registerTraining installs a Training upgrade (32013/32043 style) that
// attaches to an X-Men ally (max one Training per ally).
func registerTraining(code string, atk, hp int) {
	engine.RegisterBehavior(code, &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			u := g.Upgrades[e.EID()]
			p := g.Player(e.EOwner())
			if u == nil || p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a == nil || !a.EDef().HasTrait("x-men") {
					continue
				}
				if trainingAttachedTo(g, id) {
					continue // max 1 Training upgrade per ally
				}
				choices = append(choices, engine.Choice{
					Label: engine.S("Attach to " + a.EDef().Name), Kind: engine.ChoiceTarget, SourceID: id, CardCode: a.Code,
				}.Msgs(engine.AttachUpgrade{ID: u.ID, Target: id, ATK: atk, MaxHP: hp}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.S(u.EDef().Name+" — attach to an X-Men ally"), choices...),
			}}
		},
	})
}

// trainingAttachedTo reports whether any Training upgrade is attached to
// the ally.
func trainingAttachedTo(g *engine.Game, ally engine.EntityID) bool {
	for _, u := range g.Upgrades {
		if u != nil && u.AttachTo == ally && u.EDef().HasTrait("training") {
			return true
		}
	}
	return false
}

// activeOrFirstVillain returns the active villain or the first one.
func activeOrFirstVillain(g *engine.Game) *engine.Villain {
	if id := g.ActiveVillain; id != "" {
		if v := g.Villains[id]; v != nil {
			return v
		}
	}
	for _, v := range g.Villains {
		if v != nil {
			return v
		}
	}
	return nil
}
