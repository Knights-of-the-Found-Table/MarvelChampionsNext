// Package doctorstrange registers the Doctor Strange hero pack: the
// Invocation deck mechanic, the signature cards and the Baron Mordo
// nemesis set.
package doctorstrange

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// invocationCodes lists the five Invocation cards making up the side deck.
var invocationCodes = []string{"09032", "09033", "09034", "09035", "09036"}

func init() {
	registerDoctorStrange()
	registerSignatures()
	registerInvocations()
	registerNemesis()
	registerObligation()
}

// registerDoctorStrange installs the identity (09001a/b).
func registerDoctorStrange() {
	engine.RegisterBehavior("09001", &engine.Behavior{
		// Stephen Strange begins the game with an Invocation deck.
		HeroSetup: func(g *engine.Game, p *engine.Player) []engine.Message {
			for _, code := range invocationCodes {
				p.SenseDeck = append(p.SenseDeck, engine.Card{ID: g.NextCardID(), Code: code, Owner: p.ID})
			}
			g.ShuffleSideDeck(p)
			g.Logf("%s begins the game with an Invocation deck of %d cards", p.Name, len(p.SenseDeck))
			return nil
		},
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			var abs []engine.Ability
			if len(p.SenseDeck) > 0 {
				top := p.SenseDeck[0]
				cost := cardutil.Cost(top.Def())
				abs = append(abs, engine.Ability{
					// Spell Mastery — Action: exhaust and pay the cost of
					// the top card of the Invocation deck → resolve its
					// Special ability.
					Label:    fmt.Sprintf("Spell Mastery — resolve %s (cost %d)", top.Def().Name, cost),
					Type:     engine.AbilityAction,
					HeroOnly: true,
					Exhaust:  true,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						return invokeQuestion(g, g.Player(self), top, false)
					},
				})
			}
			// Natural Talent — Action: discard the top card of the
			// Invocation deck (limit once per phase).
			abs = append(abs, engine.Ability{
				Label:        "Natural Talent — discard the top Invocation card",
				Type:         engine.AbilityAction,
				AlterEgoOnly: true,
				OncePerRound: true, // approximation of once per phase
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(self)
					if p == nil || len(p.SenseDeck) == 0 {
						return nil
					}
					top := p.SenseDeck[0]
					p.SenseDeck = p.SenseDeck[1:]
					p.SideDiscard = append(p.SideDiscard, top)
					g.Logf("%s discards %s from the Invocation deck", p.Name, top.Def().Name)
					return nil
				},
			})
			return abs
		},
	})
}

// invokeQuestion builds the payment question for resolving an invocation.
func invokeQuestion(g *engine.Game, p *engine.Player, card engine.Card, returnToTop bool) []engine.Message {
	if p == nil {
		return nil
	}
	cost := cardutil.Cost(card.Def())
	ctx := map[string]any{"invocationCard": card.ID}
	if returnToTop {
		ctx["returnToTop"] = true
	}
	var invoke engine.Choice
	if cost > 0 {
		invoke = engine.Choice{
			ID: "resolve", Label: fmt.Sprintf("Resolve %s (cost %d)", card.Def().Name, cost),
			Kind: engine.ChoiceCard, CardCode: card.Code,
		}.WithThen(g.CustomPaymentQuestion(p, cost,
			fmt.Sprintf("Pay %d resources for %s", cost, card.Def().Name), ctx))
	} else {
		invoke = engine.Choice{
			ID: "resolve", Label: fmt.Sprintf("Resolve %s (cost 0)", card.Def().Name),
			Kind: engine.ChoiceCard, CardCode: card.Code,
		}.Msgs(engine.InvokeSpecial{Player: p.ID, Card: card, ReturnToTop: returnToTop})
	}
	return []engine.Message{engine.AskQuestion{
		Player:   p.ID,
		Question: engine.Ask(fmt.Sprintf("Resolve %s?", card.Def().Name), invoke, cardutil.Skip()),
	}}
}

// registerInvocations installs the five Invocation cards' Specials.
func registerInvocations() {
	// 09032 Crimson Bands of Cyttorak: stun an enemy and deal 7 damage.
	engine.RegisterBehavior("09032", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			choices := cardutil.EnemyChoices(g, 0, pid, func(target engine.EntityID) []engine.Message {
				return []engine.Message{
					engine.StunEntity{Target: target},
					engine.DamageEntity{Target: target, Damage: 7, Source: pid},
				}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: pid, Question: engine.Ask("Crimson Bands of Cyttorak — stun and deal 7 damage", choices...)}}
		},
	})
	// 09033 Images of Ikonn: confuse the villain, remove 4 threat.
	engine.RegisterBehavior("09033", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			var msgs []engine.Message
			for id := range g.Villains {
				msgs = append(msgs, engine.ConfuseEntity{Target: id})
			}
			var choices []engine.Choice
			for _, id := range g.Schemes() {
				s := g.Entity(id)
				choices = append(choices, engine.Choice{
					Label: s.EDef().Name, Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: s.ECode(),
				}.Msgs(engine.ThwartScheme{Scheme: id, N: 4, Source: pid}))
			}
			if len(choices) == 0 {
				return msgs
			}
			msgs = append(msgs, engine.AskQuestion{Player: pid, Question: engine.Ask("Images of Ikonn — remove 4 threat", choices...)})
			return msgs
		},
	})
	// 09034 Seven Rings of Raggadorr: up to 3 characters gain tough.
	engine.RegisterBehavior("09034", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			choices = append(choices, engine.Choice{
				Label: p.Name, Kind: engine.ChoiceTarget, SourceID: p.ID,
			}.Msgs(engine.ToughEntity{Target: p.ID}))
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil {
					choices = append(choices, engine.Choice{
						Label: a.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id, CardCode: a.Code,
					}.Msgs(engine.ToughEntity{Target: id}))
				}
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.AskN("Seven Rings of Raggadorr — tough status (up to 3)", 3, choices...),
			}}
		},
	})
	// 09035 Vapors of Valtorr: replace a status card (approximation: clear
	// one of your own status effects).
	engine.RegisterBehavior("09035", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var msgs []engine.Message
			if p.Stunned {
				msgs = append(msgs, engine.ClearStun{Target: p.ID})
			} else if p.Confused {
				msgs = append(msgs, engine.ClearConfuse{Target: p.ID})
			} else {
				g.Logf("Vapors of Valtorr: no status card to replace")
			}
			return msgs
		},
	})
	// 09036 Winds of Watoomb: draw 3 cards.
	engine.RegisterBehavior("09036", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.DrawCards{Player: e.EOwner(), N: 3}}
		},
	})
}

// registerSignatures installs Doctor Strange's signature cards.
func registerSignatures() {
	// 09002 Wong: exhaust → heal 1 from your identity or discard the top
	// Invocation card.
	engine.RegisterBehavior("09002", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label:   "Wong — heal 1 damage or discard the top Invocation card",
				Type:    engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(e.EOwner())
					if p == nil {
						return nil
					}
					heal := engine.Choice{
						ID: "heal", Label: "Heal 1 damage from your identity", Kind: engine.ChoiceLabel,
					}.Msgs(engine.HealEntity{Target: p.ID, N: 1})
					choices := []engine.Choice{heal}
					if len(p.SenseDeck) > 0 {
						top := p.SenseDeck[0]
						choices = append(choices, engine.Choice{
							ID: "discard", Label: "Discard " + top.Def().Name + " from the Invocation deck", Kind: engine.ChoiceLabel,
						}.Msgs(engine.SideDeckDiscardTop{Player: p.ID}))
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Wong — choose", choices...)}}
				},
			}}
		},
	})
	// NOTE: discarding via InvokeSpecial runs the Special too, which is a
	// small approximation (the printed discard has no effect); acceptable
	// because discarding Winds of Watoomb simply draws nothing extra.

	// 09003 Astral Projection: remove 3 threat (+1 per boost icon on the
	// top encounter card).
	engine.RegisterBehavior("09003", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			n := 3
			if len(g.EncounterDeck) > 0 {
				n += cardutil.BoostOf(g.EncounterDeck[0])
			}
			var choices []engine.Choice
			for _, id := range g.Schemes() {
				s := g.Entity(id)
				choices = append(choices, engine.Choice{
					Label: s.EDef().Name, Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: s.ECode(),
				}.Msgs(engine.ThwartScheme{Scheme: id, N: n, Source: pid}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: pid, Question: engine.Ask(fmt.Sprintf("Astral Projection — remove %d threat", n), choices...)}}
		},
	})

	// 09004 Magic Blast: deal 5 damage + rider by the discarded card's
	// resource icon.
	engine.RegisterBehavior("09004", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			var rider []engine.Message
			icon := ""
			if len(p.Deck) > 0 {
				milled := p.Deck[0]
				p.Deck = p.Deck[1:]
				p.Discard = append(p.Discard, milled)
				for _, r := range milled.Def().Resources {
					icon = r
				}
				g.Logf("Magic Blast discards %s [%s]", milled.Def().Name, icon)
			}
			mk := func(target engine.EntityID) []engine.Message {
				msgs := []engine.Message{engine.DamageEntity{Target: target, Damage: 5, Source: pid}}
				switch icon {
				case "physical":
					msgs = append(msgs, engine.StunEntity{Target: target})
				case "energy":
					msgs = append(msgs, engine.DamageEntity{Target: target, Damage: 2, Source: pid})
				case "mental":
					msgs = append(msgs, engine.ConfuseEntity{Target: target})
				case "wild":
					msgs = append(msgs, engine.StunEntity{Target: target}, engine.DamageEntity{Target: target, Damage: 2, Source: pid}, engine.ConfuseEntity{Target: target})
				}
				return msgs
			}
			_ = rider
			choices := cardutil.EnemyChoices(g, 0, pid, mk)
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: pid, Question: engine.Ask("Magic Blast — deal 5 damage", choices...)}}
		},
	})

	// 09005 Master of the Mystic Arts: pay the top Invocation's cost →
	// resolve it and place it back on top.
	engine.RegisterBehavior("09005", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || len(p.SenseDeck) == 0 {
				return nil
			}
			return invokeQuestion(g, p, p.SenseDeck[0], true)
		},
	})

	// 09006 Mystical Studies: search deck and discard for a Doctor
	// Strange card (approximation: signature-set cards only).
	engine.RegisterBehavior("09006", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, c := range p.Deck {
				def := c.Def()
				if def.CardSet == "doctor_strange" && def.Code != "09006" {
					choices = append(choices, engine.Choice{
						Label: "Take " + def.Name, Kind: engine.ChoiceCard, CardCode: def.Code,
					}.Msgs(engine.TakeDeckCard{Player: pid, CardID: c.ID}, engine.ShufflePlayerDeck{Player: pid}))
				}
			}
			for _, c := range p.Discard {
				def := c.Def()
				if def.CardSet == "doctor_strange" && def.Code != "09006" {
					msgs := []engine.Message{engine.ShuffleIntoDeck{Player: pid, CardID: c.ID}}
					choices = append(choices, engine.Choice{
						Label: "Shuffle in " + def.Name, Kind: engine.ChoiceCard, CardCode: def.Code,
					}.Msgs(msgs...))
				}
			}
			if len(choices) == 0 {
				return nil
			}
			choices = append(choices, cardutil.Skip())
			return []engine.Message{engine.AskQuestion{Player: pid, Question: engine.Ask("Mystical Studies — take a Doctor Strange card", choices...)}}
		},
	})

	// 09007 Protective Ward: cancel a treachery's When Revealed.
	engine.RegisterBehavior("09007", &engine.Behavior{
		TreacheryInterrupt: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			if !p.IsHero() {
				return nil
			}
			return []engine.Message{} // cancel; nothing replaces it
		},
	})

	// 09008 Sanctum Sanctorum: AE exhaust → shuffle a Spell card from
	// your discard into your deck, draw 1.
	engine.RegisterBehavior("09008", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			hasSpell := false
			for _, c := range p.Discard {
				if c.Def().HasTrait("spell") {
					hasSpell = true
					break
				}
			}
			if !hasSpell {
				return nil
			}
			return []engine.Ability{{
				Label:        "Sanctum Sanctorum — shuffle a Spell into your deck, draw 1",
				Type:         engine.AbilityAction,
				Exhaust:      true,
				AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					var choices []engine.Choice
					for _, c := range p.Discard {
						if !c.Def().HasTrait("spell") {
							continue
						}
						choices = append(choices, engine.Choice{
							Label: "Shuffle in " + c.Def().Name, Kind: engine.ChoiceCard, CardCode: c.Code,
						}.Msgs(
							engine.ShuffleIntoDeck{Player: p.ID, CardID: c.ID},
							engine.DrawCards{Player: p.ID, N: 1},
						))
					}
					if len(choices) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Sanctum Sanctorum — shuffle which Spell?", choices...)}}
				},
			}}
		},
	})

	// 09009 Cloak of Levitation: Aerial trait + hero action exhaust →
	// ready Doctor Strange.
	engine.RegisterBehavior("09009", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.GrantTrait{Target: e.EOwner(), Trait: "aerial"}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label:    "Cloak of Levitation — ready Doctor Strange",
				Type:     engine.AbilityAction,
				Exhaust:  true,
				HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.ReadyEntity{ID: e.EOwner()}}
				},
			}}
		},
	})

	// 09010 Magical Enhancements: +1 THW/ATK/DEF; discarded at end of
	// round.
	engine.RegisterBehavior("09010", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if p := g.Player(e.EOwner()); p != nil {
				p.BonusTHW++
				p.BonusATK++
				p.BonusDEF++
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.EndRound); !ok {
				return nil
			}
			if p := g.Player(e.EOwner()); p != nil {
				p.BonusTHW--
				p.BonusATK--
				p.BonusDEF--
			}
			return []engine.Message{engine.DiscardControlled{Player: e.EOwner(), ID: e.EID()}}
		},
	})

	// 09011 The Eye of Agamotto: hero resource — exhaust → [wild].
	engine.RegisterBehavior("09011", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "wild", HeroOnly: true},
	})

	// 09012 Brother Voodoo: enters play → search top 5 for an event.
	engine.RegisterBehavior("09012", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			n := min(5, len(p.Deck))
			var choices []engine.Choice
			for i := 0; i < n; i++ {
				c := p.Deck[i]
				if c.Def().Type != "event" {
					continue
				}
				choices = append(choices, engine.Choice{
					Label: "Take " + c.Def().Name, Kind: engine.ChoiceCard, CardCode: c.Code,
				}.Msgs(engine.TakeDeckCard{Player: pid, CardID: c.ID}, engine.ShufflePlayerDeck{Player: pid}))
			}
			if len(choices) == 0 {
				return []engine.Message{engine.ShufflePlayerDeck{Player: pid}}
			}
			choices = append(choices, cardutil.Skip().Msgs(engine.ShufflePlayerDeck{Player: pid}))
			return []engine.Message{engine.AskQuestion{Player: pid, Question: engine.Ask("Brother Voodoo — take an event from the top 5", choices...)}}
		},
	})

	// 09013 Clea: when defeated, shuffle into her owner's deck.
	engine.RegisterBehavior("09013", &engine.Behavior{
		AllyDefeatInterrupt: func(g *engine.Game, a *engine.Ally, destroy func()) []engine.Message {
			owner := g.Player(a.Owner)
			if owner == nil {
				return nil
			}
			g.Delete(a.ID)
			owner.Deck = append(owner.Deck, engine.Card{ID: g.NextCardID(), Code: a.Code, Owner: owner.ID})
			g.Logf("Clea shuffles into %s's deck", owner.Name)
			return nil
		},
	})

	// 09014 Iron Fist: 2 mystic counters; when he attacks, remove 1 →
	// stun and 1 extra damage.
	engine.RegisterBehavior("09014", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if a, ok := e.(*engine.Ally); ok {
				a.Counters = 2
				g.Logf("Iron Fist enters play with 2 mystic counters")
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.AllyAttackWindow)
			a, ok2 := e.(*engine.Ally)
			if !ok || !ok2 || w.Ally != a.ID || a.Counters <= 0 {
				return nil
			}
			a.Counters--
			g.Logf("Iron Fist removes a mystic counter (stun +1 damage)")
			return []engine.Message{
				engine.StunEntity{Target: w.Target},
				engine.DamageEntity{Target: w.Target, Damage: 1, Source: a.Owner},
			}
		},
	})

	// 09015 Desperate Defense: +2 DEF; ready if no damage expected.
	engine.RegisterBehavior("09015", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			if !p.IsHero() || p.Exhausted {
				return engine.Defends{}, nil, false
			}
			d := engine.Defends{Defender: p.ID, Against: against, DefBonus: 2}
			attack := 0
			switch t := g.Entity(against).(type) {
			case *engine.Villain:
				attack = t.AttackVal + t.BoostCount
			case *engine.Minion:
				attack = t.AttackVal
			}
			var extra []engine.Message
			if attack <= p.DefenseStat(g)+2 {
				extra = append(extra, engine.ReadyEntity{ID: p.ID})
			}
			return d, extra, true
		},
	})

	// 09016 Momentum Shift: heal 2 → deal 2.
	engine.RegisterBehavior("09016", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			var msgs []engine.Message
			msgs = append(msgs, engine.HealEntity{Target: pid, N: 2})
			choices := cardutil.EnemyChoices(g, 2, pid, func(target engine.EntityID) []engine.Message {
				return []engine.Message{engine.DamageEntity{Target: target, Damage: 2, Source: pid}}
			})
			if len(choices) > 0 {
				msgs = append(msgs, engine.AskQuestion{Player: pid, Question: engine.Ask("Momentum Shift — deal 2 damage", choices...)})
			}
			return msgs
		},
	})

	// 09017 The Power of Protection: generic (validator doubling).
	engine.RegisterBehavior("09017", &engine.Behavior{})
}

// registerNemesis installs the Baron Mordo set.
func registerNemesis() {
	// 09028 Baron Mordo: when he attacks you, discard the top card of
	// your deck; riders by resource icon.
	engine.RegisterBehavior("09028", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok || w.Enemy != e.EID() {
				return nil
			}
			p := g.Player(w.Player)
			if p == nil || len(p.Deck) == 0 {
				return nil
			}
			milled := p.Deck[0]
			p.Deck = p.Deck[1:]
			p.Discard = append(p.Discard, milled)
			icon := ""
			for _, r := range milled.Def().Resources {
				icon = r
			}
			g.Logf("Baron Mordo's attack discards %s [%s]", milled.Def().Name, icon)
			var msgs []engine.Message
			switch icon {
			case "physical":
				msgs = append(msgs, engine.StunEntity{Target: p.ID})
			case "energy":
				msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: 2, Source: e.EID()})
			case "mental":
				msgs = append(msgs, engine.ConfuseEntity{Target: p.ID})
			case "wild":
				msgs = append(msgs, engine.StunEntity{Target: p.ID}, engine.DamageEntity{Target: p.ID, Damage: 2, Source: e.EID()}, engine.ConfuseEntity{Target: p.ID})
			}
			return msgs
		},
	})

	// 09029 Open the Dark Dimension: steals the top Invocation card while
	// in play (approximation: the card returns when the scheme is
	// defeated).
	engine.RegisterBehavior("09029", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			s, ok := e.(*engine.SideScheme)
			if !ok {
				return nil
			}
			// find the Strange player (approximation: first player with a
			// side deck)
			var p *engine.Player
			for _, pl := range g.Players {
				if pl.HeroCode == "09001a" || len(pl.SenseDeck) > 0 {
					p = pl
					break
				}
			}
			if p == nil || len(p.SenseDeck) == 0 {
				return nil
			}
			stolen := p.SenseDeck[0]
			p.SenseDeck = p.SenseDeck[1:]
			s.StoredCards = append(s.StoredCards, stolen)
			g.Logf("Open the Dark Dimension steals %s", stolen.Def().Name)
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			s := g.SideSchemes[m.Scheme]
			if s == nil || len(s.StoredCards) == 0 {
				return nil
			}
			for _, pl := range g.Players {
				if len(pl.SenseDeck) > 0 || pl.HeroCode == "09001a" {
					pl.SenseDeck = append(pl.SenseDeck, s.StoredCards...)
					g.Logf("The stolen Invocation card returns")
					return nil
				}
			}
			return nil
		},
	})

	// 09030 Counterspell: approximation — when you play an event it is
	// returned to your hand (effects resolve) and Counterspell discards.
	engine.RegisterBehavior("09030", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			// stay in play attached to the player
			t.Target = p.ID
			g.Logf("Counterspell attaches to %s", p.Name)
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.EventPlayed)
			t, ok2 := e.(*engine.Treachery)
			if !ok || !ok2 || m.Player != t.Target {
				return nil
			}
			g.Delete(t.ID)
			return []engine.Message{engine.ReturnDiscardCard{Player: m.Player, CardID: m.Card.ID}}
		},
	})

	// 09031 Thoughtcasting: discard your highest-cost card; threat or
	// damage equal to its cost.
	engine.RegisterBehavior("09031", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			if len(p.Hand) == 0 {
				return nil
			}
			worst := p.Hand[0]
			for _, c := range p.Hand {
				if cardutil.Cost(c.Def()) > cardutil.Cost(worst.Def()) {
					worst = c
				}
			}
			if _, ok := p.Hand.Remove(worst.ID); ok {
				p.Discard = append(p.Discard, worst)
			}
			n := cardutil.Cost(worst.Def())
			g.Logf("Thoughtcasting discards %s (cost %d)", worst.Def().Name, n)
			if p.IsHero() {
				return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: n, Source: t.ID}}
			}
			if g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: n, Source: t.ID}}
			}
			return nil
		},
	})
}

// registerObligation installs Physical Toll.
func registerObligation() {
	// You may flip to alter-ego form. Choose: exhaust Stephen Strange →
	// remove from game, or your next event costs 3 additional resources.
	engine.RegisterBehavior("09027", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			penalty := []engine.Message{}
			if p != nil {
				p.CostDiscounts = append(p.CostDiscounts, engine.CostDiscount{Type: "event", Amount: -3})
				g.Logf("%s's next event costs 3 additional resources", p.Name)
			}
			return cardutil.ExhaustOrPenalty(g, p, card, "Your next event costs 3 additional resources", penalty...)
		},
	})
}
