// Package deadpool registers the Deadpool hero pack (44001): the
// Deadpool / Wade Wilson identity built around the identity-level
// defeat save (The Regeneratin' Degenerate) and the alter-ego deck
// search (Break the Fourth Wall), the acceleration-token-fuelled
// signature cards, The Merc with the Mouth obligation and the Butler
// nemesis set.
package deadpool

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() {
	registerDeadpool()
	registerSignatures()
	registerObligation()
	registerNemesis()
	registerCorpsCards()
	registerDreadpool()
	registerCorpsNeutrals()
}

// accelerationTokens reports the acceleration tokens on the main scheme.
func accelerationTokens(g *engine.Game) int {
	if g.MainScheme == nil {
		return 0
	}
	return g.MainScheme.AccelerationTokens
}

// registerDeadpool installs the Deadpool / Wade Wilson identity
// (44001a/b).
func registerDeadpool() {
	engine.RegisterBehavior("44001", &engine.Behavior{
		// The Regeneratin' Degenerate — Forced Interrupt: when you would
		// be defeated, instead set your hit point dial to 1, change to
		// alter-ego form, and add 1 acceleration token to the main
		// scheme. (Identity-level save: applyDefeatSave consults it with
		// a nil upgrade, after upgrade saves. The card has no use limit:
		// every rescue feeds the main scheme another token.)
		DefeatSave: func(g *engine.Game, p *engine.Player, u *engine.Upgrade) bool {
			p.Damage = p.MaxHP - 1
			if p.IsHero() {
				p.Side = engine.SideAlterEgo
			}
			if g.MainScheme != nil {
				g.MainScheme.AccelerationTokens++
			}
			g.Logf("The Regeneratin' Degenerate — %s refuses to die (1 HP, alter-ego form); the main scheme gains an acceleration token", p.Name)
			return true
		},
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			return []engine.Ability{{
				// Break the Fourth Wall — Action: discard a card from
				// your hand → search your deck for a Deadpool event and
				// add it to your hand. (Limit once per round.)
				Label:        "Break the Fourth Wall — discard a card: search your deck for a Deadpool event",
				Type:         engine.AbilityAction,
				AlterEgoOnly: true,
				OncePerRound: true,
				Execute:      breakTheFourthWall,
			}}
		},
	})
}

// breakTheFourthWall discards a hand card, then searches the deck for a
// Deadpool event (the search may legally fail; the deck is shuffled
// either way).
func breakTheFourthWall(g *engine.Game, self engine.EntityID) []engine.Message {
	p := g.Player(self)
	if p == nil || len(p.Hand) == 0 {
		return nil
	}
	var finds []engine.Choice
	for _, c := range p.Deck {
		d := c.Def()
		if d.Type != "event" || d.CardSet != "deadpool" {
			continue
		}
		finds = append(finds, engine.Choice{
			Label: "Add " + d.Name + " to your hand", Kind: engine.ChoiceCard, CardCode: c.Code,
		}.Msgs(
			engine.TakeDeckCard{Player: p.ID, CardID: c.ID},
			engine.ShufflePlayerDeck{Player: p.ID},
		))
	}
	finds = append(finds, engine.Choice{
		ID: "decline", Label: "Take nothing (search fails) — shuffle", Kind: engine.ChoicePass,
	}.Msgs(engine.ShufflePlayerDeck{Player: p.ID}))
	search := engine.Ask("Break the Fourth Wall — search your deck for a Deadpool event", finds...)

	var choices []engine.Choice
	for _, c := range p.Hand {
		choices = append(choices, engine.Choice{
			Label: "Discard " + c.Def().Name, Kind: engine.ChoiceCard, CardCode: c.Code,
		}.Msgs(
			engine.DiscardCards{Player: p.ID, Cards: engine.CardList{c}},
		).WithThen(search))
	}
	return []engine.Message{engine.AskQuestion{
		Player:   p.ID,
		Question: engine.Ask("Break the Fourth Wall — discard a card from your hand", choices...),
	}}
}

// registerSignatures installs Deadpool's signature cards.
func registerSignatures() {
	registerCable()
	registerExhaustingPersonality()
	registerMaximumEffort()
	registerMetaknowledge()
	registerYooHoo()
	registerMontage()
	registerChimichangaTruck()
	registerArmedToTheTeeth()
	registerKatanas()
	registerItAintOver()
	registerThisCardIsFire()
}

// 44002 Cable: [star] +1 THW and +1 ATK per acceleration token on the
// main scheme (max +3). (Approximation: the scaling aura is re-applied
// at each player phase begin as an until-end-of-phase bonus.)
func registerCable() {
	engine.RegisterBehavior("44002", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.BeginPhase)
			if !ok || m.Phase != engine.PhasePlayer {
				return nil
			}
			a := g.Allies[e.EID()]
			if a == nil {
				return nil
			}
			n := accelerationTokens(g)
			if n > 3 {
				n = 3
			}
			if n <= 0 {
				return nil
			}
			g.Logf("Cable — +%d THW / +%d ATK from acceleration tokens this phase", n, n)
			return []engine.Message{engine.AllyStatBonus{Ally: a.ID, ATK: n, THW: n}}
		},
	})
}

// 44003 Exhausting Personality: Hero Action — choose: place 1
// acceleration token on the main scheme → stun and confuse the villain;
// or exhaust a player's identity → that player draws 1 card per
// acceleration token on the main scheme. (Approximation: the draw count
// is fixed when the card is played.)
func registerExhaustingPersonality() {
	engine.RegisterBehavior("44003", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			var choices []engine.Choice
			if g.MainScheme != nil {
				for vid := range g.Villains {
					choices = append(choices, engine.Choice{
						ID: "accelerate", Kind: engine.ChoiceLabel,
						Label: "Place 1 acceleration token on the main scheme → stun and confuse the villain",
					}.Msgs(
						engine.AddAccelerationToken{Scheme: g.MainScheme.ID},
						engine.StunEntity{Target: vid},
						engine.ConfuseEntity{Target: vid},
					))
					break
				}
			}
			n := accelerationTokens(g)
			var targets []engine.Choice
			for _, pl := range g.Players {
				if pl.KOed || pl.Exhausted {
					continue
				}
				targets = append(targets, engine.Choice{
					Label: fmt.Sprintf("Exhaust %s → draw %d", pl.Name, n), Kind: engine.ChoiceLabel,
				}.Msgs(
					engine.ExhaustEntity{ID: pl.ID},
					engine.DrawCards{Player: pl.ID, N: n},
				))
			}
			if len(targets) > 0 {
				choices = append(choices, engine.Choice{
					ID: "exhaust-draw", Kind: engine.ChoiceLabel,
					Label: fmt.Sprintf("Exhaust a player's identity → that player draws %d", n),
				}.WithThen(engine.Ask("Exhausting Personality — exhaust whose identity?", targets...)))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Exhausting Personality — choose:", choices...),
			}}
		},
	})
}

// anyDamageQuestion builds the shared "take any amount of damage up to
// your remaining hit points → effect" choice tree (Maximum Effort,
// "Yoo-Hoo!"). effect builds the follow-up targeting question for a
// chosen amount; taking lethal damage routes through The Regeneratin'
// Degenerate like any other defeat.
func anyDamageQuestion(g *engine.Game, p *engine.Player, name string, effect func(n int) *engine.Question) []engine.Message {
	hp := p.HP()
	if hp <= 0 {
		return nil
	}
	var choices []engine.Choice
	for n := 1; n <= hp; n++ {
		q := effect(n)
		if q == nil {
			continue
		}
		choices = append(choices, engine.Choice{
			ID: fmt.Sprintf("take-%d", n), Kind: engine.ChoiceLabel,
			Label: fmt.Sprintf("Take %d damage", n),
		}.Msgs(
			engine.DamageEntity{Target: p.ID, Damage: n, Source: p.ID},
		).WithThen(q))
	}
	if len(choices) == 0 {
		return nil
	}
	return []engine.Message{engine.AskQuestion{
		Player:   p.ID,
		Question: engine.Ask(name+" — take how much damage?", choices...),
	}}
}

// 44004 Maximum Effort: Hero Action (attack) — take any amount of
// damage up to your remaining hit points → deal an equal amount of
// damage to an enemy.
func registerMaximumEffort() {
	engine.RegisterBehavior("44004", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			return anyDamageQuestion(g, p, "Maximum Effort", func(n int) *engine.Question {
				choices := cardutil.EnemyChoices(g, n, p.ID, func(id engine.EntityID) []engine.Message {
					return []engine.Message{engine.DamageEntity{Target: id, Damage: n, Source: p.ID}}
				})
				if len(choices) == 0 {
					return nil
				}
				return engine.Ask(fmt.Sprintf("Maximum Effort — deal %d damage to which enemy?", n), choices...)
			})
		},
	})
}

// 44005 Metaknowledge: Hero Interrupt — when an encounter card is
// revealed, cancel all of its effects and discard it; take 1 damage per
// boost icon on that card. (Approximation: hooks the treachery
// interrupt window, so only treacheries can be cancelled; the revealed
// card is not exposed to the hook, so the boost-icon damage is not
// modeled.)
func registerMetaknowledge() {
	engine.RegisterBehavior("44005", &engine.Behavior{
		TreacheryInterrupt: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			if !p.IsHero() {
				return nil
			}
			g.Logf("Metaknowledge — the fourth wall eats the encounter card")
			return []engine.Message{}
		},
	})
}

// 44006 "Yoo-Hoo!": Hero Action (thwart) — take any amount of damage up
// to your remaining hit points → remove an equal amount of threat from
// a scheme.
func registerYooHoo() {
	engine.RegisterBehavior("44006", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			return anyDamageQuestion(g, p, `"Yoo-Hoo!"`, func(n int) *engine.Question {
				choices := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
					return []engine.Message{engine.ThwartScheme{Scheme: id, N: n, Source: p.ID}}
				})
				if len(choices) == 0 {
					return nil
				}
				return engine.Ask(fmt.Sprintf(`"Yoo-Hoo!" — remove %d threat from which scheme?`, n), choices...)
			})
		},
	})
}

// 44007 Montage: generates 1 additional [wild] resource per
// acceleration token on the main scheme (max 3 additional).
// (Approximation: resource payment counts printed icons only, so the
// flat printed resource applies; the acceleration scaling is not
// modeled.)
func registerMontage() {
	engine.RegisterBehavior("44007", &engine.Behavior{})
}

// 44008 Chimichanga Truck: Response — after an identity makes a basic
// recovery, exhaust Chimichanga Truck → ready that identity.
// (Approximation: the optional response auto-triggers.)
func registerChimichangaTruck() {
	engine.RegisterBehavior("44008", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.BasicRecover)
			if !ok {
				return nil
			}
			s := g.Supports[e.EID()]
			if s == nil || s.Exhausted {
				return nil
			}
			rp := g.Player(m.Player)
			if rp == nil || !rp.Exhausted {
				return nil
			}
			g.Logf("Chimichanga Truck — %s readies after recovering", rp.Name)
			return []engine.Message{
				engine.ExhaustEntity{ID: s.ID},
				engine.ReadyEntity{ID: rp.ID},
			}
		},
	})
}

// 44009 Armed to the Teeth: searches the player's collection for a
// [[WEAPON]] upgrade and swaps weapons in and out. (Collection search
// and facedown attachment are not modeled.)
func registerArmedToTheTeeth() {
	engine.RegisterBehavior("44009", &engine.Behavior{})
}

// 44010 Deadpool's Katana: Restricted. Hero Action (attack) — exhaust
// and take 1 damage → deal 2 damage to an enemy. (The piercing rider is
// not modeled.)
func registerKatanas() {
	engine.RegisterBehavior("44010", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			return []engine.Ability{{
				Label:    "Deadpool's Katana — exhaust and take 1 damage: deal 2 damage to an enemy",
				Type:     engine.AbilityAction,
				Exhaust:  true,
				HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					if u == nil {
						return nil
					}
					choices := cardutil.EnemyChoices(g, 2, u.Owner, func(id engine.EntityID) []engine.Message {
						return []engine.Message{engine.DamageEntity{Target: id, Damage: 2, Source: u.ID}}
					})
					if len(choices) == 0 {
						return nil
					}
					return []engine.Message{
						engine.DamageEntity{Target: u.Owner, Damage: 1, Source: u.ID},
						engine.AskQuestion{
							Player:   u.Owner,
							Question: engine.Ask("Deadpool's Katana — deal 2 damage to which enemy?", choices...),
						},
					}
				},
			}}
		},
	})
}

// 44011 It Ain't Over...: attach to the main scheme; the attached
// scheme's target threat is increased by 2 per acceleration token on
// it. (Approximation: the card attaches to the main scheme; the
// target-threat modification is not modeled.)
func registerItAintOver() {
	engine.RegisterBehavior("44011", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if g.MainScheme == nil {
				return nil
			}
			return []engine.Message{engine.AttachUpgrade{ID: e.EID(), Target: g.MainScheme.ID}}
		},
	})
}

// 44012 This Card is Fire: Hero Action (attack) — deal X damage to an
// enemy, X = the damage you have sustained. (The Forced Response that
// pings you for holding it at turn end is not modeled: in-hand reaction
// windows are not hooked.)
func registerThisCardIsFire() {
	engine.RegisterBehavior("44012", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || p.Damage <= 0 {
				return nil
			}
			x := p.Damage
			choices := cardutil.EnemyChoices(g, x, p.ID, func(id engine.EntityID) []engine.Message {
				return []engine.Message{engine.DamageEntity{Target: id, Damage: x, Source: p.ID}}
			})
			if len(choices) == 0 {
				return nil
			}
			g.Logf("This Card is Fire — X is %d", x)
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(fmt.Sprintf("This Card is Fire — deal %d damage to which enemy?", x), choices...),
			}}
		},
	})
}

// registerObligation installs The Merc with the Mouth (44032): the
// resolving player exhausts each ally they control. (The allies-cannot-
// ready lock, the silence on other players' abilities and the "talk or
// discard" forfeit are not modeled; the obligation resolves to the
// discard after the exhaust.)
func registerObligation() {
	engine.RegisterBehavior("44032", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var msgs []engine.Message
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && !a.Exhausted {
					msgs = append(msgs, engine.ExhaustEntity{ID: id})
				}
			}
			g.Logf("The Merc with the Mouth — %s's allies are exhausted", p.Name)
			return append(msgs, engine.ObligationResolve{Player: p.ID, Card: card})
		},
	})
}

// registerNemesis installs the Deadpool nemesis set (deadpool_nemesis):
// Butler, Involuntary Procedures, Tabula Rasa 16 and Mutated Soldier.
// (The set-aside Dreadpool encounter set, 44037-44042, is a modular
// encounter set and out of scope for the hero pack.)
func registerNemesis() {
	// 44033 Butler: Forced Interrupt — when Butler schemes, place that
	// threat on Involuntary Procedures instead (not modeled). Boost —
	// you are confused. (Approximation: the boost hook has no target
	// context; the first player stands in, matching the nemesis
	// treachery precedent.)
	engine.RegisterBehavior("44033", &engine.Behavior{
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			p := g.Player(cardutil.FirstPlayerID(g))
			if p == nil {
				return nil
			}
			g.Logf("Butler's boost — %s is confused", p.Name)
			return []engine.Message{engine.ConfuseEntity{Target: p.ID}}
		},
	})

	// 44034 Involuntary Procedures: Forced Response — after Deadpool
	// takes any amount of damage, place 1 threat here; at 10+ threat
	// remove this card from the game. (Approximation: the threat is
	// placed; no remove-from-game channel exists for schemes, so the
	// 10-threat rider is not modeled.)
	engine.RegisterBehavior("44034", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.DamageEntity)
			if !ok || m.Damage <= 0 {
				return nil
			}
			s := g.SideSchemes[e.EID()]
			if s == nil {
				return nil
			}
			victim, ok2 := g.Entity(m.Target).(*engine.Player)
			if !ok2 || data.BaseCode(victim.HeroCode) != "44001" {
				return nil
			}
			g.Logf("Involuntary Procedures — Deadpool's damage feeds the scheme")
			return []engine.Message{engine.SchemeThreat{Scheme: s.ID, N: 1, Source: s.ID}}
		},
	})

	// 44035 Tabula Rasa 16: attach to your identity; treat its printed
	// text box as blank; Alter-Ego Action — spend [mental][mental] →
	// discard. Boost — attach to your identity. (Text-box blanking and
	// the discard action are not modeled; the boost attach follows the
	// generic attachment handling.)
	engine.RegisterBehavior("44035", &engine.Behavior{})

	// 44036 Mutated Soldier: Toughness. Forced Response — after Mutated
	// Soldier activates, heal all damage from it. (Approximation: the
	// generic heal channel does not cover minions, so the hook repairs
	// the damage directly; the heal lands when the activation message
	// is dispatched, before its attack resolves.)
	engine.RegisterBehavior("44036", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionActivates)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			mn := g.Minions[e.EID()]
			if mn == nil || mn.Damage <= 0 {
				return nil
			}
			g.Logf("Mutated Soldier — heals all damage after activating")
			mn.Damage = 0
			return nil
		},
	})
}
