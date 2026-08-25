// Package hulk registers the Hulk hero pack: the identity, signature
// cards, and the Abomination nemesis set.
package hulk

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerHulk()
	registerSignatures()
	registerNemesis()
	registerObligation()
	registerRemaining()
}

// registerHulk installs the Hulk / Bruce Banner identity (10001a/b).
func registerHulk() {
	engine.RegisterBehavior("10001", &engine.Behavior{
		// Enraged — Forced Interrupt: when your turn ends, discard your
		// hand. Applies while in hero form (the ability is printed on the
		// Hulk side).
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.PlayerTurnEnd)
			if !ok || m.Player != e.EID() {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil || !p.IsHero() || len(p.Hand) == 0 {
				return nil
			}
			g.TLogf("c.enragedDiscardsTheirHand", p.Name)
			return []engine.Message{engine.DiscardCards{
				Player: p.ID,
				Cards:  append(engine.CardList(nil), p.Hand...),
			}}
		},
		// Bruce Banner — Experimental Research: Action: draw 1 card,
		// choose and discard 1 card from your hand (limit once per round).
		// Approximation: the discard choice is picked from the pre-draw
		// hand, so the drawn card itself cannot be the one discarded.
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			return []engine.Ability{{
				Label:        engine.Tf("c.experimentalResearchDraw1ThenDiscard1"),
				Type:         engine.AbilityAction,
				AlterEgoOnly: true,
				OncePerRound: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					pl := g.Player(self)
					if pl == nil {
						return nil
					}
					if len(pl.Hand) == 0 {
						return []engine.Message{engine.DrawCards{Player: pl.ID, N: 1}}
					}
					var choices []engine.Choice
					for _, c := range pl.Hand {
						choices = append(choices, engine.Choice{
							Label: engine.S("Draw 1, discard " + c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code,
						}.Msgs(
							engine.DrawCards{Player: pl.ID, N: 1},
							engine.DiscardCards{Player: pl.ID, Cards: engine.CardList{c}},
						))
					}
					return []engine.Message{engine.AskQuestion{
						Player:   pl.ID,
						Question: engine.Ask(engine.Tf("c.experimentalResearchDraw1ThenDiscardWhichCard"), choices...),
					}}
				},
			}}
		},
	})
}

// registerSignatures installs Hulk's signature cards.
func registerSignatures() {
	registerCrushingBlow()
	registerSubOrbitalLeap()
	registerThunderclap()
	registerUnstoppableForce()
	registerBannersLaboratory()
	registerImmovableObject()
}

// 10002 Crushing Blow: Hero Action (attack) — deal damage to an enemy
// equal to your ATK. (Approximation: the [physical]-only payment
// restriction is not enforced; the cost is paid normally.)
func registerCrushingBlow() {
	engine.RegisterBehavior("10002", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			dmg := p.AttackStat(g)
			if dmg < 1 {
				dmg = 1
			}
			choices := cardutil.EnemyChoices(g, dmg, pid, func(target engine.EntityID) []engine.Message {
				return []engine.Message{engine.DamageEntity{Target: target, Damage: dmg, Source: pid}}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.crushingBlowDealDamageEqualToYourAtk"), choices...),
			}}
		},
	})
}

// 10004 Sub-Orbital Leap: Hero Action (thwart) — remove 3 threat from a
// scheme. (Approximation: the 5-threat upgrade when paid with only
// [physical] resources is not tracked; always 3.)
func registerSubOrbitalLeap() {
	engine.RegisterBehavior("10004", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			var choices []engine.Choice
			if g.MainScheme != nil {
				id := g.MainScheme.ID
				choices = append(choices, engine.Choice{
					Label: engine.S("Remove 3 threat from " + g.MainScheme.EDef().Name), Kind: engine.ChoiceLabel,
				}.Msgs(engine.ThwartScheme{Scheme: id, N: 3, Source: pid}))
			}
			for _, id := range cardutil.SortedIDs(g.SideSchemes) {
				ss := g.SideSchemes[id]
				if ss == nil {
					continue
				}
				id := id
				choices = append(choices, engine.Choice{
					Label: engine.S("Remove 3 threat from " + ss.EDef().Name), Kind: engine.ChoiceLabel,
				}.Msgs(engine.ThwartScheme{Scheme: id, N: 3, Source: pid}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.subOrbitalLeapRemove3ThreatFromAScheme"), choices...),
			}}
		},
	})
}

// 10005 Thunderclap: Hero Action — choose up to 3 different enemies,
// deal 3 damage to each. (Approximation: targets are the first three
// enemies in stable order rather than a free multi-select.)
func registerThunderclap() {
	engine.RegisterBehavior("10005", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			ids := cardutil.SortedEnemyIDs(g)
			if len(ids) > 3 {
				ids = ids[:3]
			}
			var msgs []engine.Message
			for _, id := range ids {
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 3, Source: pid})
			}
			if len(msgs) == 0 {
				g.TLogf("c.thunderclapNoEnemiesInPlay")
			}
			return msgs
		},
	})
}

// 10006 Unstoppable Force: Hero Action — ready Hulk. (Approximation:
// the extra draw when paid with only [physical] resources is not
// tracked.)
func registerUnstoppableForce() {
	engine.RegisterBehavior("10006", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			g.TLogf("c.unstoppableForceReadies", p.Name)
			return []engine.Message{engine.ReadyEntity{ID: p.ID}}
		},
	})
}

// 10008 Banner's Laboratory: Alter-Ego Resource — exhaust → generate a
// [mental] resource; Bruce Banner gets +2 REC. (Approximation: the
// resource can be generated in either form, and the REC bonus is not
// applied.)
func registerBannersLaboratory() {
	engine.RegisterBehavior("10008", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "mental"},
	})
}

// 10010 Immovable Object: You get +4 hit points; Hulk gains retaliate
// 1. (Approximation: only the retaliate bonus is representable via
// IdentityStats; the +4 HP is not applied.)
func registerImmovableObject() {
	engine.RegisterBehavior("10010", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{Retaliate: 1}
		},
	})
}

// registerNemesis installs Hulk's nemesis set (Abomination).
func registerNemesis() {
	// 10026 Abomination: after Abomination attacks you, discard the top
	// card of your deck; if a [physical] resource was discarded this
	// way, take 2 damage.
	engine.RegisterBehavior("10026", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil {
				return nil
			}
			physical := false
			if len(p.Deck) > 0 {
				for _, r := range p.Deck[0].Def().Resources {
					if r == "physical" {
						physical = true
					}
				}
			}
			msgs := []engine.Message{engine.MillPlayerDeck{Player: m.Player, N: 1}}
			if physical {
				g.TLogf("c.abominationAPhysicalResourceWasDiscardedTakes2Damage", p.Name)
				msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: 2, Source: e.EID()})
			}
			return msgs
		},
	})

	// 10027 Total Destruction: threat cannot be removed from this scheme
	// while Abomination is in play. (Approximation: not enforced; the
	// scheme resolves with generic threat rules.)

	// 10028 Clash of the Titans: When Revealed — the enemy with the
	// highest ATK attacks the hero or ally with the highest ATK.
	// (Approximation: resolved as direct damage equal to the enemy's
	// ATK against the highest-ATK hero/ally, without a defense window.)
	engine.RegisterBehavior("10028", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var enemyID engine.EntityID
			best := -1
			for _, id := range cardutil.SortedIDs(g.Villains) {
				if v := g.Villains[id]; v != nil && v.AttackVal > best {
					best, enemyID = v.AttackVal, id
				}
			}
			for _, id := range cardutil.SortedIDs(g.Minions) {
				if mn := g.Minions[id]; mn != nil && mn.AttackVal > best {
					best, enemyID = mn.AttackVal, id
				}
			}
			if best < 0 {
				g.TLogf("c.clashOfTheTitansNoEnemiesInPlayGainsSurge")
				return nil
			}
			var target engine.EntityID
			tbest := -1
			for _, pl := range g.Players {
				if pl.IsHero() {
					if atk := pl.AttackStat(g); atk > tbest {
						tbest, target = atk, pl.ID
					}
				}
				for _, id := range pl.Allies {
					if a := g.Allies[id]; a != nil && a.AttackVal > tbest {
						tbest, target = a.AttackVal, id
					}
				}
			}
			if target == "" {
				return nil
			}
			g.TLogf("c.clashOfTheTitansTheStrongestEnemyStrikesTheStrongestHero")
			return []engine.Message{engine.DamageEntity{Target: target, Damage: best, Source: enemyID}}
		},
	})
}

// registerObligation installs Inner Demons.
func registerObligation() {
	// 10025 Inner Demons: change form. If you are Bruce Banner (after
	// the flip), discard 2 cards; if you are Hulk, exhaust your hero.
	// Then discard this obligation. (Approximation: the 2-card discard
	// takes the first two cards of the hand instead of a free choice.)
	engine.RegisterBehavior("10025", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			if p == nil {
				return nil
			}
			msgs := []engine.Message{engine.ChangeForm{Player: p.ID}}
			if p.IsHero() {
				// Currently Hulk: flips to Banner and discards 2.
				if len(p.Hand) > 0 {
					n := 2
					if len(p.Hand) < n {
						n = len(p.Hand)
					}
					msgs = append(msgs, engine.DiscardCards{
						Player: p.ID,
						Cards:  append(engine.CardList(nil), p.Hand[:n]...),
					})
				}
			} else {
				// Currently Banner: flips to Hulk and exhausts.
				msgs = append(msgs, engine.ExhaustEntity{ID: p.ID})
			}
			msgs = append(msgs, engine.ObligationResolve{Player: p.ID, Card: card})
			return msgs
		},
	})
}
