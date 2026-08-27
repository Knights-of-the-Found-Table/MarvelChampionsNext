package captainamerica

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// registerAspectCards installs the pack's Leadership, basic and other
// aspect cards (03011-03025, 03031-03034).
func registerAspectCards() {
	registerFalcon()
	registerHawkeye()
	registerSquirrelGirl()
	registerWonderMan()
	registerAvengersAssemble()
	registerMakeTheCall()
	registerStrengthInNumbers()
	registerPowerOfLeadership()
	registerQuinjet()
	registerMockingbird()
	registerAvengersTower()
	registerHonoraryAvenger()
	registerBasicResources()
	registerEnraged()
	registerFollowed()
	registerExpertDefense()
	registerEnhancedAwareness()
}

// 03011 Falcon: after he enters play, look at the top 3 cards of the
// encounter deck; remove 1 threat from a scheme for each treachery looked
// at (approximation: the looked cards go to the bottom of the deck).
func registerFalcon() {
	engine.RegisterBehavior("03011", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			n := min(3, len(g.EncounterDeck))
			treacheries := 0
			var names []string
			for i := 0; i < n; i++ {
				def := g.EncounterDeck[i].Def()
				names = append(names, def.Name)
				if def.Type == "treachery" {
					treacheries++
				}
			}
			if n > 0 {
				// rotate the looked cards to the bottom
				rest := make(engine.CardList, 0, len(g.EncounterDeck))
				rest = append(rest, g.EncounterDeck[n:]...)
				rest = append(rest, g.EncounterDeck[:n]...)
				g.EncounterDeck = rest
				g.TLogf("c.falconLooksAt", names)
			}
			if treacheries == 0 || len(g.Schemes()) == 0 {
				return nil
			}
			var choices []engine.Choice
			for _, id := range g.Schemes() {
				s := g.Entity(id)
				choices = append(choices, engine.Choice{
					Label: engine.Tf("m.cardName", s), Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: s.ECode(),
				}.Msgs(engine.ThwartScheme{Scheme: id, N: treacheries, Source: pid}))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.falconRemoveThreatFromAScheme", treacheries), choices...),
			}}
		},
	})
}

// 03012 Hawkeye: enters play with 4 arrow counters; after a minion enters
// play, remove 1 arrow counter → deal 2 damage to it.
func registerHawkeye() {
	engine.RegisterBehavior("03012", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if a, ok := e.(*engine.Ally); ok {
				a.Counters = 4
				g.TLogf("c.hawkeyeEntersPlayWith4ArrowCounters")
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionEntersPlay)
			if !ok {
				return nil
			}
			a, ok := e.(*engine.Ally)
			if !ok || a.Counters <= 0 {
				return nil
			}
			mn := g.Entity(m.MinionID)
			if mn == nil {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player: a.Owner,
				Question: engine.Ask(engine.Tf("c.hawkeyeRemove1ArrowCounterToShootFor2Arrows", mn, a.Counters),
					engine.Choice{
						ID: "shoot", Label: engine.Tf("c.shoot"), Kind: engine.ChoiceLabel,
					}.Msgs(
						engine.AddEntityCounter{ID: a.ID, N: -1},
						engine.DamageEntity{Target: m.MinionID, Damage: 2, Source: a.Owner},
					),
					engine.Choice{ID: "skip", Label: engine.Tf("c.skip"), Kind: engine.ChoicePass},
				),
			}}
		},
	})
}

// 03013 Squirrel Girl: after she enters play, deal 1 damage to each enemy.
func registerSquirrelGirl() {
	engine.RegisterBehavior("03013", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var msgs []engine.Message
			for _, id := range cardutil.SortedEnemyIDs(g) {
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: e.EOwner()})
			}
			return msgs
		},
	})
}

// 03014 Wonder Man: as an additional cost for him to attack, discard 1
// card from your hand.
func registerWonderMan() {
	engine.RegisterBehavior("03014", &engine.Behavior{
		AllyAttackDiscardCost: true,
	})
}

// 03015 Avengers Assemble!: ready each Avenger character you control;
// until the end of the phase each Avenger character in play gets +1 THW
// and +1 ATK (max 1 per round — the second copy's effect is skipped).
func registerAvengersAssemble() {
	engine.RegisterBehavior("03015", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			if g.UsedThisRound["03015"] {
				g.TLogf("c.avengersAssembleWasAlreadyPlayedThisRoundMax1")
				return nil
			}
			g.UsedThisRound["03015"] = true
			var msgs []engine.Message
			for _, pl := range g.Players {
				for _, id := range pl.Allies {
					if a := g.Allies[id]; a != nil && g.EntityHasTrait(id, "avenger") {
						a.BonusTHW++
						a.BonusATK++
						if pl.ID == pid {
							msgs = append(msgs, engine.ReadyEntity{ID: id})
						}
					}
				}
				if g.EntityHasTrait(pl.ID, "avenger") {
					pl.BonusTHW++
					pl.BonusATK++
				}
			}
			g.TLogf("c.avengersAssembleAvengersGet1ThwAnd1AtkUntilTheEndOfThePhase")
			return msgs
		},
	})
}

// 03016 Make the Call: pay the printed cost of an ally in any player's
// discard pile → put that ally into play under your control.
func registerMakeTheCall() {
	engine.RegisterBehavior("03016", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, pl := range g.Players {
				for _, c := range pl.Discard {
					def := c.Def()
					if def.Type != "ally" {
						continue
					}
					cost := cardutil.Cost(def)
					choices = append(choices, engine.Choice{
						Label: engine.Tf("c.costFromSDiscardPile", def.Name, cost, pl.Name),
						Kind:  engine.ChoiceCard, CardCode: def.Code,
					}.WithThen(g.CustomPaymentQuestion(p, cost,
						engine.Tf("q.payGeneric", cost, def.Name),
						map[string]any{
							"makeCallFrom": pl.ID.String(),
							"makeCallCard": c.ID,
						})))
				}
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.makeTheCallChooseAnAllyFromAnyDiscardPile"), choices...),
			}}
		},
	})
}

// 03017 Strength In Numbers: exhaust any number of allies you control →
// draw 1 card for each ally exhausted this way.
func registerStrengthInNumbers() {
	engine.RegisterBehavior("03017", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && !a.Exhausted {
					choices = append(choices, engine.Choice{
						Label: engine.Tf("c.exhaustName", a), Kind: engine.ChoiceTarget,
						SourceID: id, CardCode: a.Code,
					}.Msgs(
						engine.ExhaustEntity{ID: id},
						engine.DrawCards{Player: pid, N: 1},
					))
				}
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.AskN(engine.Tf("c.strengthInNumbersExhaustAlliesDraw1PerAlly"), len(choices), choices...),
			}}
		},
	})
}

// 03018 The Power of Leadership: doubles its resource while paying for a
// Leadership card — implemented generically in the payment validator.
func registerPowerOfLeadership() {
	engine.RegisterBehavior("03018", &engine.Behavior{})
}

// 03019 Quinjet: after your turn begins place 1 time counter on Quinjet;
// action: put an Avenger ally from your hand into play with printed cost
// ≤ the number of time counters, then discard Quinjet.
func registerQuinjet() {
	engine.RegisterBehavior("03019", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.PlayerTurnStart)
			if !ok || m.Player != e.EOwner() {
				return nil
			}
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 1}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s, ok := e.(*engine.Support)
			if !ok {
				return nil
			}
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, c := range p.Hand {
				def := c.Def()
				if def.Type != "ally" || !def.HasTrait("avenger") || cardutil.Cost(def) > s.Counters {
					continue
				}
				choices = append(choices, engine.Choice{
					Label: engine.Tf("m.cardName", def), Kind: engine.ChoiceCard, CardCode: def.Code,
				}.Msgs(
					engine.AllyEntersPlayFree{Player: p.ID, Card: c},
					engine.DiscardControlled{Player: p.ID, ID: e.EID()},
				))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.quinjetPutAnAvengerAllyCostIntoPlayThenDiscardQuinjet", s.Counters),
				Type:  engine.AbilityAction,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.AskQuestion{
						Player:   p.ID,
						Question: engine.Ask(engine.Tf("c.quinjetChooseAnAvengerAlly"), choices...),
					}}
				},
			}}
		},
	})
}

// 03020 Mockingbird: after she enters play, stun an enemy.
func registerMockingbird() {
	engine.RegisterBehavior("03020", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			choices := cardutil.EnemyChoices(g, 0, pid, func(target engine.EntityID) []engine.Message {
				return []engine.Message{engine.StunEntity{Target: target}}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.mockingbirdStunAnEnemy"), choices...),
			}}
		},
	})
}

// 03024 Avengers Tower: if each of your allies has the Avenger trait,
// increase your ally limit by 1 (ally limit not enforced — approximation);
// action: exhaust → the next Avenger ally played this phase costs 1 less.
func registerAvengersTower() {
	engine.RegisterBehavior("03024", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label:   engine.Tf("c.avengersTowerTheNextAvengerAllyThisPhaseCosts1Less"),
				Type:    engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					if p := g.Player(e.EOwner()); p != nil {
						p.CostDiscounts = append(p.CostDiscounts, engine.CostDiscount{
							Type: "ally", Trait: "avenger", Amount: 1,
						})
						g.TLogf("c.theNextAvengerAllyThisPhaseCosts1Less")
					}
					return nil
				},
			}}
		},
	})
}

// 03025 Honorary Avenger: attach to a friendly character; it gets +1 hit
// point and gains the Avenger trait.
func registerHonoraryAvenger() {
	engine.RegisterBehavior("03025", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			choices = append(choices, engine.Choice{
				Label: engine.Tf("c.nameIdentity", p.Name), Kind: engine.ChoiceTarget, SourceID: p.ID,
			}.Msgs(engine.AttachUpgrade{ID: e.EID(), Target: p.ID, MaxHP: 1, GrantTrait: "avenger"}))
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil {
					choices = append(choices, engine.Choice{
						Label: engine.Tf("m.cardName", a), Kind: engine.ChoiceTarget, SourceID: id, CardCode: a.Code,
					}.Msgs(engine.AttachUpgrade{ID: e.EID(), Target: id, MaxHP: 1, GrantTrait: "avenger"}))
				}
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.honoraryAvengerAttachToAFriendlyCharacter"), choices...),
			}}
		},
	})
}

// 03021-03023 Energy/Genius/Strength: plain resource cards, handled
// generically by the data layer.
func registerBasicResources() {
	engine.RegisterBehavior("03021", &engine.Behavior{})
	engine.RegisterBehavior("03022", &engine.Behavior{})
	engine.RegisterBehavior("03023", &engine.Behavior{})
}

// 03031 Enraged: attach to an ally; it gets +2 ATK and takes +1
// consequential damage after it attacks.
func registerEnraged() {
	engine.RegisterBehavior("03031", &engine.Behavior{
		ConsequentialBonus: 1,
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil || len(p.Allies) == 0 {
				return nil
			}
			var choices []engine.Choice
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil {
					choices = append(choices, engine.Choice{
						Label: engine.Tf("m.cardName", a), Kind: engine.ChoiceTarget, SourceID: id, CardCode: a.Code,
					}.Msgs(engine.AttachUpgrade{ID: e.EID(), Target: id, ATK: 2}))
				}
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.enragedAttachToAnAlly2Atk"), choices...),
			}}
		},
	})
}

// 03032 Followed: attach to a side scheme; when the scheme is defeated,
// deal 4 damage to an enemy.
func registerFollowed() {
	engine.RegisterBehavior("03032", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			var choices []engine.Choice
			for _, id := range cardutil.SortedIDs(g.SideSchemes) {
				s := g.SideSchemes[id]
				choices = append(choices, engine.Choice{
					Label: engine.Tf("m.cardName", s), Kind: engine.ChoiceTarget, SourceID: id, CardCode: s.Code,
				}.Msgs(engine.AttachUpgrade{ID: e.EID(), Target: id}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.followedAttachToASideScheme"), choices...),
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok {
				return nil
			}
			u, ok := e.(*engine.Upgrade)
			if !ok || u.AttachTo != m.Scheme {
				return nil
			}
			pid := u.Owner
			var out []engine.Message
			out = append(out, engine.DiscardControlled{Player: pid, ID: u.ID})
			choices := cardutil.EnemyChoices(g, 4, pid, func(target engine.EntityID) []engine.Message {
				return []engine.Message{engine.DamageEntity{Target: target, Damage: 4, Source: pid}}
			})
			if len(choices) > 0 {
				out = append(out, engine.AskQuestion{
					Player:   pid,
					Question: engine.Ask(engine.Tf("c.followedDeal4DamageToAnEnemy"), choices...),
				})
			}
			return out
		},
	})
}

// 03033 Expert Defense: when your hero defends against an attack, it gets
// +3 DEF for that attack.
func registerExpertDefense() {
	engine.RegisterBehavior("03033", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			if !p.IsHero() || p.Exhausted {
				return engine.Defends{}, nil, false
			}
			return engine.Defends{Defender: p.ID, Against: against, DefBonus: 3}, nil, true
		},
	})
}

// 03034 Enhanced Awareness: uses (3 mental counters); hero resource —
// exhaust and remove 1 counter → generate a [mental] resource.
func registerEnhancedAwareness() {
	engine.RegisterBehavior("03034", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "mental", HeroOnly: true, UsesCounters: true},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if u, ok := e.(*engine.Upgrade); ok {
				u.Counters = 3
				g.TLogf("c.enhancedAwarenessEntersPlayWith3MentalCounters")
			}
			return nil
		},
	})
}
