package msmarvel

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// registerSignatures installs Ms. Marvel's signature cards (05002-05011).
func registerSignatures() {
	registerRedDagger()
	registerBigHands()
	registerSneakBy()
	registerWiggleRoom()
	registerAamirKhan()
	registerBrunoCarrelli()
	registerNakiaBahadir()
	registerPolymerSuit()
	registerEmbiggen()
	registerShrink()
}

// 05002 Red Dagger: when he would be defeated, spend 2 resources of
// different types → deal 2 damage to an enemy and return him to your hand
// (approximation: any 2 resources).
func registerRedDagger() {
	engine.RegisterBehavior("05002", &engine.Behavior{
		AllyDefeatInterrupt: func(g *engine.Game, a *engine.Ally, destroy func()) []engine.Message {
			p := g.Player(a.Owner)
			if p == nil {
				return nil
			}
			icons := 0
			for _, c := range p.Hand {
				icons += len(c.Def().Resources)
			}
			if icons < 2 {
				return nil // save impossible
			}
			return []engine.Message{engine.AskQuestion{
				Player: a.Owner,
				Question: engine.Ask(engine.Tf("c.redDaggerSpend2ResourcesToDeal2DamageAndReturnHimToYourHand"),
					engine.Choice{
						ID: "save", Label: engine.Tf("c.saveRedDaggerPay2Resources"), Kind: engine.ChoiceLabel,
					}.WithThen(g.CustomPaymentQuestion(p, 2, engine.S("Pay 2 resources to save Red Dagger"),
						map[string]any{"saveAlly": a.ID.String(), "saveDamage": 2})),
					engine.Choice{
						ID: "defeat", Label: engine.Tf("c.letRedDaggerBeDefeated"), Kind: engine.ChoicePass,
					}.Msgs(engine.AllyDestroyed{AllyID: a.ID}),
				),
			}}
		},
	})
}

// 05003 Big Hands: deal 4 damage to an enemy.
func registerBigHands() {
	engine.RegisterBehavior("05003", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy(engine.Tf("c.bigHandsDeal4Damage"), func(g *engine.Game, e engine.Entity) (int, []engine.Message) {
			return 4, nil
		}),
	})
}

// 05004 Sneak By: remove 3 threat from a scheme.
func registerSneakBy() {
	engine.RegisterBehavior("05004", &engine.Behavior{
		OnPlay: cardutil.ChooseScheme(engine.Tf("c.chooseAScheme", "Sneak By"), func(g *engine.Game, e engine.Entity) int {
			return 3
		}),
	})
}

// 05005 Wiggle Room: when you would take any amount of damage, prevent 3
// of that damage. Draw 1 card (approximation: offered in the defense
// prompt).
func registerWiggleRoom() {
	engine.RegisterBehavior("05005", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			d := engine.Defends{
				Defender: p.ID, Against: against,
				Undefended: true, ExtraPrevent: 3,
			}
			return d, []engine.Message{engine.DrawCards{Player: p.ID, N: 1}}, true
		},
	})
}

// 05006 Aamir Khan: Alter-Ego Action — exhaust → place 1 card from your
// discard pile on the bottom of your deck, then draw 1 card.
func registerAamirKhan() {
	engine.RegisterBehavior("05006", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil || len(p.Discard) == 0 {
				return nil
			}
			return []engine.Ability{{
				Label:        engine.Tf("c.aamirKhanPutADiscardCardOnTheBottomOfYourDeckDraw1"),
				Type:         engine.AbilityAction,
				Exhaust:      true,
				AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					var choices []engine.Choice
					for _, c := range p.Discard {
						choices = append(choices, engine.Choice{
							Label: engine.S(c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code,
						}.Msgs(
							engine.DiscardToBottom{Player: p.ID, CardID: c.ID},
							engine.DrawCards{Player: p.ID, N: 1},
						))
					}
					return []engine.Message{engine.AskQuestion{
						Player:   p.ID,
						Question: engine.Ask(engine.Tf("c.aamirKhanChooseADiscardCard"), choices...),
					}}
				},
			}}
		},
	})
}

// 05007 Bruno Carrelli: Alter-Ego Action — exhaust → attach 1 card from
// your hand facedown here; Action — exhaust → add up to 3 cards attached
// here to your hand (the stored cards are facedown; the player only
// chooses how many to take).
func registerBrunoCarrelli() {
	engine.RegisterBehavior("05007", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s, ok := e.(*engine.Support)
			if !ok {
				return nil
			}
			p := g.Player(s.Owner)
			if p == nil {
				return nil
			}
			var abs []engine.Ability
			if len(p.Hand) > 0 {
				abs = append(abs, engine.Ability{
					Label:        engine.Tf("c.brunoCarrelliTuckACardFromYourHandFacedown"),
					Type:         engine.AbilityAction,
					Exhaust:      true,
					AlterEgoOnly: true,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						var choices []engine.Choice
						for _, c := range p.Hand {
							choices = append(choices, engine.Choice{
								Label: engine.S("Tuck " + c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code,
							}.Msgs(engine.SupportStoreCard{ID: s.ID, Card: c}))
						}
						return []engine.Message{engine.AskQuestion{
							Player:   p.ID,
							Question: engine.Ask(engine.Tf("c.brunoCarrelliTuckACard"), choices...),
						}}
					},
				})
			}
			if s.Counters > 0 {
				abs = append(abs, engine.Ability{
					Label:   engine.Tf("c.brunoCarrelliTakeUpTo3StoredCardsBack"),
					Type:    engine.AbilityAction,
					Exhaust: true,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						n := min(3, s.Counters)
						var choices []engine.Choice
						for i := 1; i <= n; i++ {
							cards := append(engine.CardList{}, s.AttachedCards[:i]...)
							choices = append(choices, engine.Choice{
								ID: fmt.Sprintf("take-%d", i), Label: engine.Tf("c.takeCardS", i),
								Kind: engine.ChoiceLabel,
							}.Msgs(engine.SupportRetrieveCards{ID: s.ID, Cards: cards}))
						}
						return []engine.Message{engine.AskQuestion{
							Player:   p.ID,
							Question: engine.Ask(engine.Tf("c.brunoCarrelliHowManyStoredCards"), choices...),
						}}
					},
				})
			}
			return abs
		},
	})
}

// 05008 Nakia Bahadir: Alter-Ego Action — exhaust → reduce the cost of the
// next card you play this phase by 1.
func registerNakiaBahadir() {
	engine.RegisterBehavior("05008", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label:        engine.Tf("c.nakiaBahadirYourNextCardThisPhaseCosts1Less"),
				Type:         engine.AbilityAction,
				Exhaust:      true,
				AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					if p := g.Player(e.EOwner()); p != nil {
						p.CostDiscounts = append(p.CostDiscounts, engine.CostDiscount{Amount: 1})
						g.TLogf("c.sNextCardThisPhaseCosts1Less", p.Name)
					}
					return nil
				},
			}}
		},
	})
}

// 05009 Biokinetic Polymer Suit: hero resource — exhaust → generate a
// [wild] resource for an event.
func registerPolymerSuit() {
	engine.RegisterBehavior("05009", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "wild", EventOnly: true},
	})
}

// 05010 Embiggen!: hero interrupt — when you play an [[Attack]] event,
// exhaust → that event deals +2 damage.
func registerEmbiggen() {
	engine.RegisterBehavior("05010", &engine.Behavior{
		React: eventBoost("05010", "attack", "damage", 2),
	})
}

// 05011 Shrink: hero interrupt — when you play a [[Thwart]] event, exhaust
// → that event removes +2 threat.
func registerShrink() {
	engine.RegisterBehavior("05011", &engine.Behavior{
		React: eventBoost("05011", "thwart", "threat", 2),
	})
}

// eventBoost builds the Embiggen!/Shrink interrupt on EventPlayed.
func eventBoost(code, trait, kind string, n int) func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
	return func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.EventPlayed)
		if !ok || m.Player != e.EOwner() {
			return nil
		}
		if e.EExhausted() {
			return nil
		}
		def := m.Card.Def()
		if !def.HasTrait(trait) {
			return nil
		}
		p := g.Player(m.Player)
		if p == nil || !p.IsHero() {
			return nil
		}
		var effect []engine.Message
		if kind == "damage" {
			effect = []engine.Message{engine.SetEventBonus{Player: p.ID, Damage: n}}
		} else {
			effect = []engine.Message{engine.SetEventBonus{Player: p.ID, Threat: n}}
		}
		return []engine.Message{engine.AskQuestion{
			Player: p.ID,
			Question: engine.Ask(engine.Tf("c.exhaustGets", e, def.Name, n, kind),
				engine.Choice{
					ID: "boost", Label: engine.Tf("c.exhaust2", e, n, kind),
					Kind: engine.ChoiceLabel,
				}.Msgs(append([]engine.Message{engine.ExhaustEntity{ID: e.EID()}}, effect...)...),
				engine.Choice{ID: "skip", Label: engine.Tf("c.skip"), Kind: engine.ChoicePass},
			),
		}}
	}
}
