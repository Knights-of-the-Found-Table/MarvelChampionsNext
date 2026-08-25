// Package quicksilver registers the Quicksilver hero pack: Super Speed,
// the basic-power reward loops, and the Avalanche nemesis set.
package quicksilver

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerQuicksilver()
	registerNemesis()
}

func registerQuicksilver() {
	// Quicksilver: Super Speed — ready after a basic power (once per
	// phase).
	engine.RegisterBehavior("14001", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if g.UsedThisTurn["super-speed"] {
				return nil
			}
			switch msg.(type) {
			case engine.BasicAttack, engine.BasicThwart, engine.Defends:
				g.UsedThisTurn["super-speed"] = true
				return []engine.Message{engine.ReadyEntity{ID: e.EID()}}
			}
			return nil
		},
	})

	// Scarlet Witch: boost-icon scaling on her basic power use —
	// approximated to a flat +1 on her menu stats.
	engine.RegisterBehavior("14002", &engine.Behavior{})

	// Always Be Running: ready Quicksilver.
	engine.RegisterBehavior("14003", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.ReadyEntity{ID: e.EOwner()}}
		},
	})

	// Double Time: choose two of damage/threat (may repeat).
	engine.RegisterBehavior("14004", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var options []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				enemy := g.Entity(id)
				if enemy != nil {
					options = append(options, engine.Choice{Label: engine.Tf("c.deal2", cardutil.EnemyLabel(enemy)), Kind: engine.ChoiceTarget, SourceID: id, CardCode: enemy.ECode()}.
						Msgs(engine.DamageEntity{Target: id, Damage: 2, Source: p.ID}))
				}
			}
			for _, sid := range g.Schemes() {
				s := g.Entity(sid)
				options = append(options, engine.Choice{Label: engine.S("Remove 2 — " + s.EDef().Name), Kind: engine.ChoiceTarget, SourceID: sid, CardCode: s.ECode()}.
					Msgs(engine.ThwartScheme{Scheme: sid, N: 2, Source: p.ID}))
			}
			if len(options) == 0 {
				return nil
			}
			q := engine.AskN(engine.Tf("c.doubleTimeChooseTwoOptions"), 2, options...)
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: q}}
		},
	})

	// Maximum Velocity: +2/+2/+2 until end of phase (round approximated
	// to phase).
	engine.RegisterBehavior("14005", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.ApplyStatBonus{Target: e.EOwner(), THW: 2, ATK: 2, DEF: 2}}
		},
	})

	// Speed Cyclone: stun X enemies (X = players).
	engine.RegisterBehavior("14006", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			x := len(g.Players)
			var picks []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				enemy := g.Entity(id)
				if enemy != nil {
					picks = append(picks, engine.Choice{Label: engine.Tf("c.stun2", cardutil.EnemyLabel(enemy)), Kind: engine.ChoiceTarget, SourceID: id, CardCode: enemy.ECode()}.
						Msgs(engine.StunEntity{Target: id}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			q := engine.AskN(engine.Tf("c.speedCycloneStunWhichEnemies"), min(x, len(picks)), picks...)
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(), Question: q}}
		},
	})

	// Serval Industries: shuffle 2 Quicksilver cards from discard.
	engine.RegisterBehavior("14007", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustServalIndustriesShuffle2QuicksilverCardsBack"), Type: engine.AbilityAction,
				Exhaust: true, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.EOwner())
					if p == nil {
						return nil
					}
					var picks []engine.Choice
					for _, c := range p.Discard {
						def := c.Def()
						if def.Code[:2] == "14" && def.Type != "obligation" {
							picks = append(picks, engine.Choice{Label: engine.S(def.Name), Kind: engine.ChoiceCard, CardCode: def.Code}.
								Msgs(engine.ShuffleIntoDeck{Player: p.ID, CardID: c.ID}))
						}
					}
					if len(picks) == 0 {
						return nil
					}
					q := engine.AskN(engine.Tf("c.shuffleWhichQuicksilverCardsBack"), min(2, len(picks)), picks...)
					return []engine.Message{engine.AskQuestion{Player: p.ID, Question: q}}
				},
			}}
		},
	})

	// Stat upgrades.
	engine.RegisterBehavior("14008", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{DEF: 1} },
	})
	engine.RegisterBehavior("14010", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{THW: 1} },
	})
	engine.RegisterBehavior("14011", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{ATK: 1} },
	})

	// Friction Resistance: after Quicksilver readies, ready this card;
	// physical resource.
	engine.RegisterBehavior("14009", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "physical"},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			r, ok := msg.(engine.ReadyEntity)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil || r.ID != u.Owner {
				return nil
			}
			if u.Exhausted {
				return []engine.Message{engine.ReadyEntity{ID: e.EID()}}
			}
			return nil
		},
	})

	// Multiple Man: recruit a copy from deck or hand.
	engine.RegisterBehavior("14012", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			for _, c := range p.Deck {
				if c.Code[:5] == "14012" {
					picks = append(picks, engine.Choice{Label: engine.Tf("c.multipleManDeck"), Kind: engine.ChoiceCard, CardCode: c.Code}.
						Msgs(engine.TakeDeckCard{Player: p.ID, CardID: c.ID},
							engine.PlayCard{Player: p.ID, Card: c, Paid: engine.CostPaid{}},
							engine.ShufflePlayerDeck{Player: p.ID}))
					break
				}
			}
			for _, c := range p.Hand {
				if c.Code[:5] == "14012" {
					picks = append(picks, engine.Choice{Label: engine.Tf("c.multipleManHand"), Kind: engine.ChoiceCard, CardCode: c.Code}.
						Msgs(engine.PlayCard{Player: p.ID, Card: c, Paid: engine.CostPaid{}}))
					break
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.multipleManPutACopyIntoPlay"), picks...)}}
		},
	})

	// Warlock: spend [mental] → heal up to 2.
	engine.RegisterBehavior("14013", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.spendMentalHealUpTo2DamageFromWarlock"), Type: engine.AbilityAction,
				Cost: 1, CostIcons: "mental:1",
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					a := g.Allies[self]
					if a == nil {
						return nil
					}
					n := min(2, a.Damage)
					if n <= 0 {
						return nil
					}
					return []engine.Message{engine.HealEntity{Target: self, N: n}}
				},
			}}
		},
	})

	// Never Back Down: +2 DEF; stun the attacker if no damage taken
	// (tough-rider approximated: stun only when full prevention).
	engine.RegisterBehavior("14014", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, ec *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			return engine.Defends{Defender: p.ID, Against: against, DefBonus: 2, PreventAll: true},
				[]engine.Message{engine.StunEntity{Target: against}}, true
		},
	})

	// Side Step: prevent 3; energy payment adds 1 damage.
	engine.RegisterBehavior("14015", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, ec *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			var extra []engine.Message
			for _, ic := range ec.Paid.Icons {
				if ic == "energy" {
					extra = append(extra, engine.DamageEntity{Target: against, Damage: 1, Source: p.ID})
					break
				}
			}
			return engine.Defends{Defender: p.ID, Against: against, ExtraPrevent: 3}, extra, true
		},
	})

	// Armored Vest: +1 DEF reprint.
	if b := engine.LookupBehavior("14008"); b != nil {
		engine.RegisterBehavior("14016", b)
	}

	// Nerves of Steel: energy for Defense events (gate approximated).
	engine.RegisterBehavior("14017", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "energy"},
	})

	// Order and Chaos: cancel a treachery + 2 damage to the villain.
	engine.RegisterBehavior("14018", &engine.Behavior{
		TreacheryInterrupt: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var msgs []engine.Message
			for id := range g.Villains {
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 2, Source: p.ID})
				break
			}
			return msgs
		},
	})

	// Basic resources.
	engine.RegisterBehavior("14019", &engine.Behavior{})
	engine.RegisterBehavior("14020", &engine.Behavior{})
	engine.RegisterBehavior("14021", &engine.Behavior{})

	// Adrenaline Rush / Civic Duty: discard for +1 ATK / +1 THW.
	engine.RegisterBehavior("14022", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.discardAdrenalineRush1AtkThisPhase"), Type: engine.AbilityAction, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					if u == nil {
						return nil
					}
					return []engine.Message{
						engine.DiscardControlled{Player: u.Owner, ID: self},
						engine.ApplyStatBonus{Target: u.Owner, ATK: 1},
					}
				},
			}}
		},
	})
	engine.RegisterBehavior("14023", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.discardCivicDuty1ThwThisPhase"), Type: engine.AbilityAction, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					if u == nil {
						return nil
					}
					return []engine.Message{
						engine.DiscardControlled{Player: u.Owner, ID: self},
						engine.ApplyStatBonus{Target: u.Owner, THW: 1},
					}
				},
			}}
		},
	})

	// Need for Speed obligation.
	engine.RegisterBehavior("14024", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			if !p.Exhausted {
				return []engine.Message{
					engine.ExhaustEntity{ID: p.ID},
					engine.ObligationResolve{Player: p.ID, Card: card, Remove: true},
				}
			}
			return []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
		},
	})

	// Brute Force: +1 ATK; discard after a basic attack (piercing
	// skipped).
	engine.RegisterBehavior("14029", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{ATK: 1} },
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			ba, ok := msg.(engine.BasicAttack)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil || ba.Player != u.Owner {
				return nil
			}
			return []engine.Message{engine.DiscardControlled{Player: u.Owner, ID: u.ID}}
		},
	})

	// Sense of Justice: mental for Thwart events (gate approximated).
	engine.RegisterBehavior("14030", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "mental"},
	})

	// United We Stand: heal 1 from up to X characters (X = villain
	// stage, max 3).
	engine.RegisterBehavior("14031", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			x := 1
			for _, v := range g.Villains {
				x = v.Stage
			}
			x = min(3, x)
			var picks []engine.Choice
			for _, q := range g.Players {
				if q.Damage > 0 {
					picks = append(picks, engine.Choice{Label: engine.S(q.Name), Kind: engine.ChoiceTarget, SourceID: q.ID}.
						Msgs(engine.HealEntity{Target: q.ID, N: 1}))
				}
				for _, id := range q.Allies {
					if a := g.Allies[id]; a != nil && a.Damage > 0 {
						picks = append(picks, engine.Choice{Label: engine.S(a.EDef().Name), Kind: engine.ChoiceTarget, SourceID: a.ID, CardCode: a.Code}.
							Msgs(engine.HealEntity{Target: a.ID, N: 1}))
					}
				}
			}
			if len(picks) == 0 {
				return nil
			}
			q := engine.AskN(engine.Tf("c.unitedWeStandHealWhichCharacters"), x, picks...)
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(), Question: q}}
		},
	})

	// Beat 'Em Up: 1 damage to the villain + minions engaged with you.
	engine.RegisterBehavior("14032", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var msgs []engine.Message
			for id := range g.Villains {
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: e.EOwner()})
				break
			}
			for _, mn := range g.Minions {
				if mn.EngagedWith == e.EOwner() {
					msgs = append(msgs, engine.DamageEntity{Target: mn.ID, Damage: 1, Source: e.EOwner()})
				}
			}
			return msgs
		},
	})
}

func registerNemesis() {
	// Extortion of Seismic Proportion: Incite 1 on reveal.
	engine.RegisterBehavior("14025", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if g.MainScheme == nil {
				return nil
			}
			return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: e.EID()}}
		},
	})

	// Avalanche: Incite 2 + damage-or-exhaust choice.
	engine.RegisterBehavior("14026", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var msgs []engine.Message
			if g.MainScheme != nil {
				msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: e.EID()})
			}
			for _, p := range g.Players {
				p := p
				msgs = append(msgs, engine.AskQuestion{Player: p.ID, Question: engine.Ask(
					engine.Tf("c.avalancheTake2DamageOrExhaustYourIdentity"),
					engine.Choice{ID: "dmg", Label: engine.Tf("c.take2Damage"), Kind: engine.ChoiceLabel}.
						Msgs(engine.DamageEntity{Target: p.ID, Damage: 2, Source: e.EID()}),
					engine.Choice{ID: "exh", Label: engine.Tf("c.exhaustYourIdentity"), Kind: engine.ChoiceLabel}.
						Msgs(engine.ExhaustEntity{ID: p.ID}),
				)})
			}
			return msgs
		},
	})

	// Vibration Resistance: -1 damage on the attached enemy (via
	// VillainDamageable/Minion gates); exhaust removal.
	engine.RegisterBehavior("14027", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, mn := range g.Minions {
				if mn.Code[:5] == "14026" {
					t.Target = mn.ID
					return nil
				}
			}
			for id := range g.Villains {
				t.Target = id
				return nil
			}
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustYourHeroDiscardVibrationResistance"), Type: engine.AbilityAction, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					a := g.Attachments[self]
					if a == nil {
						return nil
					}
					g.Delete(self)
					g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: a.Code})
					return []engine.Message{engine.ExhaustEntity{ID: g.ActiveTurn}}
				},
			}}
		},
	})

	// Earthquake: Incite 1 + discard 2 + exhaust; boost rider.
	engine.RegisterBehavior("14028", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			msgs := []engine.Message{}
			if g.MainScheme != nil {
				msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: t.ID})
			}
			for i := 0; i < 2 && len(p.Hand) > 0; i++ {
				c := p.Hand[0]
				p.Hand = p.Hand[1:]
				p.Discard = append(p.Discard, c)
			}
			msgs = append(msgs, engine.ExhaustEntity{ID: p.ID})
			return msgs
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return []engine.Message{engine.ExhaustEntity{ID: engine.PlayerID(card.Owner)}}
		},
	})
}
