package nextevolution

import (
	"strings"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func registerDominoCards() {
	// 40038 Diamondback: exhaust + 1 self damage + discard deck top → 1
	// damage to each enemy per resource icon discarded.
	engine.RegisterBehavior("40038", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			a := g.Allies[e.EID()]
			if a == nil || len(g.Enemies()) == 0 {
				return nil
			}
			p := g.Player(a.Owner)
			if p == nil || len(p.Deck) == 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.diamondbackLuckShotAtEveryEnemy"), Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					a := g.Allies[self]
					p := g.Player(a.Owner)
					if p == nil {
						return nil
					}
					c, _, _ := deckTopIcons(p)
					n := iconCountOf(c)
					var msgs []engine.Message
					msgs = append(msgs,
						engine.DamageEntity{Target: a.ID, Damage: 1, Source: p.ID},
						engine.MillPlayerDeck{Player: p.ID, N: 1})
					for _, id := range cardutil.SortedEnemyIDs(g) {
						msgs = append(msgs, engine.DamageEntity{Target: id, Damage: n, Source: p.ID})
					}
					g.TLogf("c.diamondbankDiscardsIcons", c, n)
					return msgs
				},
			}}
		},
	})

	// 40039 Outlaw: when she attacks, discard deck top → +1 ATK per icon.
	engine.RegisterBehavior("40039", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyAttackWindow)
			if !ok || m.Ally != e.EID() {
				return nil
			}
			a := g.Allies[e.EID()]
			p := g.Player(a.Owner)
			if p == nil || len(p.Deck) == 0 {
				return nil
			}
			c, _, _ := deckTopIcons(p)
			n := iconCountOf(c)
			a.BonusATK += n
			g.TLogf("c.outlawDiscardsAtkForThisAttack", c, n)
			return []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: 1}}
		},
	})

	// 40040 A Good Workout: 4 damage + discard deck top → +1 damage per
	// icon to that enemy.
	engine.RegisterBehavior("40040", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || len(g.Enemies()) == 0 {
				return nil
			}
			c, _, _ := deckTopIcons(p)
			n := 4 + iconCountOf(c)
			var choices []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				enemy := g.Entity(id)
				choices = append(choices, engine.Choice{
					Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: enemy.ECode(),
				}.Msgs(engine.MillPlayerDeck{Player: p.ID, N: 1},
					engine.DamageEntity{Target: id, Damage: n, Source: p.ID}))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.aGoodWorkoutDealDamageTo", n), choices...),
			}}
		},
	})

	// 40041 Luck Be a Lady: discard deck top; per icon: energy heal 2,
	// mental thwart 2, physical damage 3, wild choose (approximation: wilds
	// ask once and apply to all wild icons).
	engine.RegisterBehavior("40041", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || len(p.Deck) == 0 {
				return nil
			}
			c, icons, _ := deckTopIcons(p)
			msgs := []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: 1}}
			energy, mental, physical, wild := 0, 0, 0, 0
			for _, r := range icons {
				switch r {
				case "energy":
					energy++
				case "mental":
					mental++
				case "physical":
					physical++
				case "wild":
					wild++
				}
			}
			g.TLogf("c.luckBeALadyDiscardsEnergyMentalPhysicalWild", c, energy, mental, physical, wild)
			if energy > 0 {
				msgs = append(msgs, engine.HealEntity{Target: p.ID, N: 2 * energy})
			}
			if mental > 0 && len(g.Schemes()) > 0 {
				var choices []engine.Choice
				for _, id := range g.Schemes() {
					s := g.Entity(id)
					choices = append(choices, engine.Choice{
						Label: engine.S(s.EDef().Name), Kind: engine.ChoiceTarget, SourceID: id, CardCode: s.ECode(),
					}.Msgs(engine.ThwartScheme{Scheme: id, N: 2 * mental, Source: p.ID}))
				}
				msgs = append(msgs, engine.AskQuestion{Player: p.ID, Question: engine.Ask(
					engine.Tf("c.luckBeALadyRemoveThreatFrom", 2*mental), choices...)})
			}
			if physical > 0 && len(g.Enemies()) > 0 {
				var choices []engine.Choice
				for _, id := range cardutil.SortedEnemyIDs(g) {
					enemy := g.Entity(id)
					choices = append(choices, engine.Choice{
						Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget,
						SourceID: id, CardCode: enemy.ECode(),
					}.Msgs(engine.DamageEntity{Target: id, Damage: 3 * physical, Source: p.ID}))
				}
				msgs = append(msgs, engine.AskQuestion{Player: p.ID, Question: engine.Ask(
					engine.Tf("c.luckBeALadyDealDamageTo", 3*physical), choices...)})
			}
			if wild > 0 {
				var wildChoices []engine.Choice
				wildChoices = append(wildChoices,
					engine.Choice{ID: "w-heal", Label: engine.Tf("c.wildIconsHealDamage", 2*wild), Kind: engine.ChoiceLabel}.
						Msgs(engine.HealEntity{Target: p.ID, N: 2 * wild}))
				if len(g.Schemes()) > 0 {
					var choices []engine.Choice
					for _, id := range g.Schemes() {
						s := g.Entity(id)
						choices = append(choices, engine.Choice{
							Label: engine.S(s.EDef().Name), Kind: engine.ChoiceTarget, SourceID: id, CardCode: s.ECode(),
						}.Msgs(engine.ThwartScheme{Scheme: id, N: 2 * wild, Source: p.ID}))
					}
					wildChoices = append(wildChoices, engine.Choice{ID: "w-thw", Label: engine.Tf("c.wildIconsRemoveThreat", 2*wild), Kind: engine.ChoiceLabel}.
						Msgs(engine.AskQuestion{Player: p.ID, Question: engine.Ask(
							engine.Tf("c.luckBeALadyRemoveThreatFrom", 2*wild), choices...)}))
				}
				if len(g.Enemies()) > 0 {
					var choices []engine.Choice
					for _, id := range cardutil.SortedEnemyIDs(g) {
						enemy := g.Entity(id)
						choices = append(choices, engine.Choice{
							Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget,
							SourceID: id, CardCode: enemy.ECode(),
						}.Msgs(engine.DamageEntity{Target: id, Damage: 3 * wild, Source: p.ID}))
					}
					wildChoices = append(wildChoices, engine.Choice{ID: "w-dmg", Label: engine.Tf("c.wildIconsDealDamage", 3*wild), Kind: engine.ChoiceLabel}.
						Msgs(engine.AskQuestion{Player: p.ID, Question: engine.Ask(
							engine.Tf("c.luckBeALadyDealDamageTo", 3*wild), choices...)}))
				}
				msgs = append(msgs, engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.luckBeALadyWildIcons"), wildChoices...)})
			}
			return msgs
		},
	})

	// 40042 Right Place, Right Time: remove 3 threat + deck top icons.
	engine.RegisterBehavior("40042", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || len(p.Deck) == 0 {
				return nil
			}
			c, _, _ := deckTopIcons(p)
			n := 3 + iconCountOf(c)
			g.TLogf("c.rightPlaceRightTimeDiscardsRemovesThreat", c, n)
			return cardutil.ChooseScheme(engine.Tf("c.rightPlaceRightTimeChooseAScheme"), func(g *engine.Game, e engine.Entity) int {
				return n
			})(g, e)
		},
	})

	// 40043 Jackpot!: after discarded from the deck top, shuffle back in.
	engine.RegisterBehavior("40043", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.DeckTopDiscarded)
			if !ok || m.Card.Def() == nil || data.BaseCode(m.Card.Code) != "40043" {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil {
				return nil
			}
			if _, ok := p.Discard.Remove(m.Card.ID); ok {
				g.TLogf("c.jackpotShufflesItselfBackIntoTheDeck")
				return []engine.Message{engine.ShuffleIntoDeck{Player: p.ID, CardID: m.Card.ID}}
			}
			return nil
		},
	})

	// 40044 Pip the Pug: alter-ego action — top-deck a Domino or POSSE card
	// from the discard pile (auto-picks the first).
	engine.RegisterBehavior("40044", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			p := g.Player(s.Owner)
			if p == nil {
				return nil
			}
			for _, c := range p.Discard {
				d := c.Def()
				if d.HasTrait("Posse") || strings.Contains(d.Name, "Domino") {
					card := c
					return []engine.Ability{{
						Label: engine.S("Pip the Pug — put " + card.Def().Name + " on top of your deck"), Type: engine.AbilityAction,
						AlterEgoOnly: true, Exhaust: true,
						Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
							s := g.Supports[self]
							p := g.Player(s.Owner)
							if p == nil {
								return nil
							}
							if _, ok := p.Discard.Remove(card.ID); ok {
								p.Deck = append(engine.CardList{card}, p.Deck...)
								g.TLogf("c.putsOnTopOfTheirDeck", p.Name, card)
							}
							return nil
						},
					}}
				}
			}
			return nil
		},
	})

	// 40045 The Painted Lady: bank deck-top discards facedown (max 3);
	// alter-ego action retrieves one to hand.
	engine.RegisterBehavior("40045", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.DeckTopDiscarded)
			if !ok {
				return nil
			}
			s := g.Supports[e.EID()]
			if s == nil || s.Owner != m.Player || len(s.AttachedCards) >= 3 {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil {
				return nil
			}
			if _, ok := p.Discard.Remove(m.Card.ID); ok {
				c := m.Card
				c.FaceDown = true
				s.AttachedCards = append(s.AttachedCards, c)
				s.Counters = len(s.AttachedCards)
				g.TLogf("c.thePaintedLadyStoresFacedownStored", c, s.Counters)
			}
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			if s == nil || len(s.AttachedCards) == 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.thePaintedLadyRetrieveAStoredCard"), Type: engine.AbilityAction,
				AlterEgoOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.Owner)
					if p == nil || len(s.AttachedCards) == 0 {
						return nil
					}
					c := s.AttachedCards[0]
					s.AttachedCards = s.AttachedCards[1:]
					s.Counters = len(s.AttachedCards)
					p.Hand = append(p.Hand, c)
					g.TLogf("c.retrievesACardFromThePaintedLady", p.Name)
					return nil
				},
			}}
		},
	})

	// 40046 Domino's Pistol: exhaust + choose enemy + discard deck top →
	// damage per icon.
	engine.RegisterBehavior("40046", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			if len(g.Enemies()) == 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.dominoSPistolLuckShot"), Type: engine.AbilityAction,
				HeroOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					p := g.Player(u.Owner)
					if p == nil || len(p.Deck) == 0 {
						return nil
					}
					c, _, _ := deckTopIcons(p)
					n := iconCountOf(c)
					var choices []engine.Choice
					for _, id := range cardutil.SortedEnemyIDs(g) {
						enemy := g.Entity(id)
						choices = append(choices, engine.Choice{
							Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget,
							SourceID: id, CardCode: enemy.ECode(),
						}.Msgs(engine.MillPlayerDeck{Player: p.ID, N: 1},
							engine.DamageEntity{Target: id, Damage: n, Source: p.ID}))
					}
					return []engine.Message{engine.AskQuestion{
						Player:   p.ID,
						Question: engine.Ask(engine.Tf("c.dominoSPistolDealDamageTo", n), choices...),
					}}
				},
			}}
		},
	})

	// 40047 Lucky and Good: cancel a boost card's icons during an attack
	// against you; the enemy draws a replacement boost card.
	engine.RegisterBehavior("40047", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.RevealBoost)
			if !ok || e.EExhausted() {
				return nil
			}
			if !engine.AttackActivationPending(g, m.Enemy) {
				return nil
			}
			u := g.Upgrades[e.EID()]
			p := g.Player(u.Owner)
			if p == nil {
				return nil
			}
			add := 0
			if v := g.Villains[m.Enemy]; v != nil {
				for _, c := range v.RevealedBoosts {
					add += cardutil.BoostOf(c)
				}
			}
			if add <= 0 {
				return nil
			}
			return []engine.Message{
				engine.ExhaustEntity{ID: u.ID},
				engine.CancelBoostIcons{Enemy: m.Enemy, N: add},
				engine.DealBoost{Enemy: m.Enemy},
				engine.RevealBoost{Enemy: m.Enemy},
			}
		},
	})

	// 40048 Lucky Break: when you would reveal an encounter card, discard
	// it and reveal another instead — window lives in the engine.
	engine.RegisterBehavior("40048", &engine.Behavior{})

	// 40049 Probability Field: when you use a basic power, discard deck
	// top → +1 to that power per icon (approximation: follow-up effect).
	engine.RegisterBehavior("40049", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if e.EExhausted() {
				return nil
			}
			u := g.Upgrades[e.EID()]
			p := g.Player(u.Owner)
			if p == nil || len(p.Deck) == 0 {
				return nil
			}
			var base engine.Message
			switch m := msg.(type) {
			case engine.BasicAttack:
				if m.Player != p.ID {
					return nil
				}
				base = engine.DamageEntity{Target: m.Target, Damage: iconCountOf(p.Deck[0]), Source: p.ID}
			case engine.BasicThwart:
				if m.Player != p.ID {
					return nil
				}
				base = engine.ThwartScheme{Scheme: m.Target, N: iconCountOf(p.Deck[0]), Source: p.ID}
			case engine.BasicRecover:
				if m.Player != p.ID {
					return nil
				}
				base = engine.HealEntity{Target: p.ID, N: iconCountOf(p.Deck[0])}
			default:
				return nil
			}
			c, _, _ := deckTopIcons(p)
			g.TLogf("c.probabilityFieldDiscardsForTheBasicPower", c)
			return []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: 1}, base}
		},
	})

	// 40065 Memories of Armageddon: blank text not modeled; the alter-ego
	// discard action is offered.
	engine.RegisterBehavior("40065", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			return []engine.Message{engine.AskQuestion{
				Player: p.ID,
				Question: engine.Ask(engine.Tf("c.memoriesOfArmageddonExhaustYourIdentityInAlterEgoFormToDisca"),
					engine.Choice{ID: "discard", Label: engine.S("Exhaust " + p.AlterEgoDef().Name + " → discard"), Kind: engine.ChoiceLabel}.
						Msgs(engine.ExhaustEntity{ID: p.ID}, engine.ObligationResolve{Player: p.ID, Card: card}),
					engine.Choice{ID: "keep", Label: engine.Tf("c.keepItInPlay"), Kind: engine.ChoicePass}),
			}}
		},
	})

	// 40066 Topaz: fetch Superpower Feedback onto the revealer.
	engine.RegisterBehavior("40066", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return nil
			}
			for _, c := range g.EncounterDeck {
				if c.Code == "40069" {
					g.EncounterDeck.Remove(c.ID)
					t := &engine.Attachment{ID: g.NextEntityID(engine.KindAttachment), Code: "40069", Target: engine.EntityID(mn.EngagedWith)}
					g.Attachments[t.ID] = t
					g.TLogf("c.superpowerFeedbackAttachesTo", g.Player(mn.EngagedWith).Name)
					return nil
				}
			}
			for _, c := range g.EncounterDiscard {
				if c.Code == "40069" {
					g.EncounterDiscard.Remove(c.ID)
					t := &engine.Attachment{ID: g.NextEntityID(engine.KindAttachment), Code: "40069", Target: engine.EntityID(mn.EngagedWith)}
					g.Attachments[t.ID] = t
					g.TLogf("c.superpowerFeedbackAttachesTo", g.Player(mn.EngagedWith).Name)
					return nil
				}
			}
			return nil
		},
	})

	// 40067 Not My Lucky Day: each player takes 1 damage or adds 2 threat.
	engine.RegisterBehavior("40067", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			s := g.SideSchemes[e.EID()]
			var msgs []engine.Message
			for _, p := range g.Players {
				if p.KOed {
					continue
				}
				msgs = append(msgs, engine.AskQuestion{
					Player: p.ID,
					Question: engine.Ask(engine.Tf("c.notMyLuckyDayChoose"),
						engine.Choice{ID: "dmg", Label: engine.Tf("c.take1Damage"), Kind: engine.ChoiceLabel}.
							Msgs(engine.DamageEntity{Target: p.ID, Damage: 1, Source: s.ID}),
						engine.Choice{ID: "threat", Label: engine.Tf("c.place2ThreatOnThisScheme"), Kind: engine.ChoiceLabel}.
							Msgs(engine.ApplySchemeThreat{Scheme: s.ID, N: 2, Source: s.ID}),
					),
				})
			}
			return msgs
		},
	})

	// 40068 Prototype: luck counters equal to the revealer's damage; +1 HP
	// each.
	engine.RegisterBehavior("40068", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return nil
			}
			if p := g.Player(mn.EngagedWith); p != nil {
				mn.Counters = p.Damage
				mn.MaxHP += p.Damage
				g.TLogf("c.prototypeGainsLuckCountersHp", mn.Counters, mn.MaxHP)
			}
			return nil
		},
	})

	// 40069 Superpower Feedback: 1 damage after resolving an ability; the
	// alter-ego discard-out is not modeled.
	engine.RegisterBehavior("40069", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			t.Target = target
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.RunAbility); !ok {
				return nil
			}
			t := g.Attachments[e.EID()]
			if t == nil || t.Target == "" {
				return nil
			}
			g.TLogf("c.superpowerFeedback1DamageAfterTheAbility")
			return []engine.Message{engine.DamageEntity{Target: t.Target, Damage: 1, Source: t.ID}}
		},
	})
}
