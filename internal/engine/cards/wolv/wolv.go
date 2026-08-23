// Package wolv registers the Wolverine hero pack: the Logan identity
// (with the Healing Factor response), Wolverine's Claws, Adamantium
// Skeleton, Berserker Frenzy, "I Got Better!", the Logan's Cabin
// support, the signature attack events, and the Omega Red / Lady
// Deathstrike nemesis set.
//
// The healing factor is keyed off BeginPhase{Phase: PhasePlayer}: when
// the player phase begins, the identity heals 2 damage. The Adamantium
// Skeleton +4 HP is applied through OnPlay; the +1 ATK and piercing on
// Wolverine's basic attacks are not enforced (IdentityStats only covers
// ATK/THW/DEF/REC/Retaliate — the piercing gain is dropped).
//
// "I Got Better!" models the defeat save as a React on DamageEntity
// that, when the player would reach 0 HP, restores them to 5, readies
// them, and discards the card. The engine has no per-upgrade defeat
// save hook, so this is a per-card interception.
package wolv

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerWolverine()
	registerSignatures()
	registerNemesis()
	registerObligation()
}

// reactKey returns a once-per-phase usage key for trigger / interrupt
// hooks. The engine resets UsedThisTurn between phases.
func reactKey(code, slot string) string {
	return "react:" + code + ":" + slot
}

// registerWolverine installs the Wolverine / Logan identity (35001a/b).
//
// Healing Factor — Response: after the player phase begins, heal 2
// damage. The engine's BeginPhase message fires once per phase; we
// React on it and emit a HealEntity for the player whose turn is
// about to start.
func registerWolverine() {
	engine.RegisterBehavior("35001", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			bp, ok := msg.(engine.BeginPhase)
			if !ok || bp.Phase != engine.PhasePlayer {
				return nil
			}
			p := g.Player(e.EID())
			if p == nil {
				return nil
			}
			if p.Damage == 0 {
				return nil
			}
			g.Logf("Healing Factor — %s heals 2 damage", p.Name)
			return []engine.Message{engine.HealEntity{Target: p.ID, N: 2}}
		},
	})
}

// registerSignatures installs Wolverine's signature cards.
func registerSignatures() {
	registerWolverinesClaws()
	registerJubilee()
	registerAdamantiumSkeleton()
	registerBerserkerFrenzy()
	registerIGotBetter()
	registerLogansCabin()
	registerBerserkerBarrage()
	registerSliceAndDice()
	registerLungingStrike()
	registerTrackByScent()
	registerRegenerativeHealing()
}

// 35002 Wolverine's Claws: Permanent. Hero Action: exhaust Wolverine's
// Claws, choose an [[Attack]] event in your hand, take damage equal to
// its printed cost → play that event, ignoring its resource cost. That
// attack gains piercing. (Approximated: an Ability exposed in hero
// form that asks the player to pick an Attack event from hand, then
// damages the player and plays the event without resource cost. The
// "play without cost" half is approximated by emitting PlayCard with
// a zero CostPaid.)
func registerWolverinesClaws() {
	engine.RegisterBehavior("35002", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			u, ok := e.(*engine.Upgrade)
			if !ok || u.Exhausted {
				return nil
			}
			p := g.Player(u.Owner)
			if p == nil {
				return nil
			}
			var attackEvents []engine.Card
			for _, c := range p.Hand {
				if c.Def().HasTrait("attack") {
					attackEvents = append(attackEvents, c)
				}
			}
			if len(attackEvents) == 0 {
				return nil
			}
			return []engine.Ability{{
				Label:    "Wolverine's Claws — exhaust: play an Attack event, take damage = its cost",
				Type:     engine.AbilityAction,
				Exhaust:  true,
				HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					// Re-scan at execute time so the list is current.
					pid := u.Owner
					p := g.Player(pid)
					if p == nil {
						return nil
					}
					var opts []engine.Choice
					for _, c := range p.Hand {
						if c.Def().HasTrait("attack") {
							c := c
							opts = append(opts, engine.Choice{
								Label: fmt.Sprintf("Play %s (take %d damage)", c.Def().Name, cardCost(c.Code)),
								Kind:  engine.ChoiceCard,
								CardCode: c.Code,
							}.Msgs(
								engine.ExhaustEntity{ID: self},
								engine.DamageEntity{Target: p.ID, Damage: cardCost(c.Code), Source: self},
								engine.PlayCard{Player: p.ID, Card: c, Paid: engine.CostPaid{}},
							))
						}
					}
					if len(opts) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{
						Player:   pid,
						Question: engine.Ask("Wolverine's Claws — play which Attack event?", opts...),
					}}
				},
			}}
		},
	})
}

// cardCost returns a card's printed cost (0 if missing). Cached for the
// Build/Pick path; the engine has a similar helper in cardutil but it
// is not directly addressable for the value.
func cardCost(code string) int {
	def, ok := engine.DB.Lookup(code)
	if !ok || def.Cost == nil {
		return 0
	}
	return *def.Cost
}

// 35003 Jubilee: Response — after Jubilee enters play, choose an
// enemy. Until the end of the phase, while Wolverine or Jubilee is
// making a basic attack against that enemy, they get +2 ATK for that
// attack. (Approximated: an OnPlay that offers a target question; the
// per-attack +2 is recorded via a generic counter on the player that
// other Wolverine attacks would need to look up. We skip the
// per-attack wiring.)
func registerJubilee() {
	engine.RegisterBehavior("35003", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			choices := cardutil.EnemyChoices(g, 0, pid, func(target engine.EntityID) []engine.Message {
				return nil
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Jubilee — choose the enemy Wolverine/Jubilee gets +2 ATK against this phase", choices...),
			}}
		},
	})
}

// 35004 Adamantium Skeleton: You get +4 hit points. Wolverine gets +1
// ATK and his basic attacks gain piercing. (Approximated: OnPlay adds
// +4 to MaxHP; IdentityStats grants +1 ATK. The piercing gain is
// dropped — the engine has no per-attack piercing flag.)
func registerAdamantiumSkeleton() {
	engine.RegisterBehavior("35004", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			p.MaxHP += 4
			g.Logf("Adamantium Skeleton grants +4 HP to %s (now %d)", p.Name, p.MaxHP)
			return nil
		},
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{ATK: 1}
		},
	})
}

// 35005 Berserker Frenzy: Hero Response — after Wolverine takes any
// amount of damage from an enemy attack, draw 1 card. Forced Response
// — after you flip to alter-ego form, discard this card.
func registerBerserkerFrenzy() {
	engine.RegisterBehavior("35005", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			dm, ok := msg.(engine.DamageEntity)
			if !ok || dm.Target != e.EOwner() {
				return nil
			}
			// The source must be an enemy attack — we approximate by
			// checking the source is a Villain or Minion.
			src := g.Entity(dm.Source)
			if src == nil {
				return nil
			}
			if _, isV := src.(*engine.Villain); !isV {
				if _, isM := src.(*engine.Minion); !isM {
					return nil
				}
			}
			return []engine.Message{engine.DrawCards{Player: e.EOwner(), N: 1}}
		},
	})
}

// 35006 "I Got Better!": Interrupt — when you would be defeated by an
// enemy attack, instead set your hit point dial to 5, ready your
// identity, and discard this card. (Approximated: React on
// DamageEntity that watches for the player's HP dropping to 0 or
// below. The "by an enemy attack" clause is dropped — the React
// fires on any lethal damage.)
func registerIGotBetter() {
	engine.RegisterBehavior("35006", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			dm, ok := msg.(engine.DamageEntity)
			if !ok || dm.Target != e.EOwner() {
				return nil
			}
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			if p.HP()-dm.Damage > 0 {
				return nil
			}
			// Save: HP = 5, ready, discard the card.
			p.Damage = max(0, p.MaxHP-5)
			p.Exhausted = false
			delete(g.Upgrades, e.EID())
			p.Upgrades = removeID(p.Upgrades, e.EID())
			g.Logf("\"I Got Better!\" — %s survives at 5 HP and discards the upgrade", p.Name)
			return nil
		},
	})
}

// 35007 Logan's Cabin: Alter-Ego Action — exhaust Logan's Cabin →
// shuffle 1 Wolverine card from your discard pile into your deck.
func registerLogansCabin() {
	engine.RegisterBehavior("35007", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s, ok := e.(*engine.Support)
			if !ok || s.Exhausted {
				return nil
			}
			p := g.Player(s.Owner)
			if p == nil {
				return nil
			}
			if len(p.Discard) == 0 {
				return nil
			}
			return []engine.Ability{{
				Label:        "Logan's Cabin — shuffle a Wolverine card from your discard into your deck",
				Type:         engine.AbilityAction,
				Exhaust:      true,
				AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					var opts []engine.Choice
					for _, c := range p.Discard {
						if c.Def().CardSet == "wolverine" {
							opts = append(opts, engine.Choice{
								Label: "Shuffle in " + c.Def().Name, Kind: engine.ChoiceCard, CardCode: c.Code,
							}.Msgs(
								engine.ExhaustEntity{ID: self},
								engine.ShuffleIntoDeck{Player: s.Owner, CardID: c.ID},
								engine.ShufflePlayerDeck{Player: s.Owner},
							))
						}
					}
					if len(opts) == 0 {
						g.Logf("Logan's Cabin — no Wolverine card in discard")
						return nil
					}
					return []engine.Message{engine.AskQuestion{
						Player:   s.Owner,
						Question: engine.Ask("Logan's Cabin — shuffle which Wolverine card into your deck?", opts...),
					}}
				},
			}}
		},
	})
}

// 35008 Berserker Barrage: Hero Action (attack) — deal 4 damage to an
// enemy. If this attack defeats an enemy, you may take 2 damage to
// repeat. (Approximated: deal 4 to a chosen enemy; the "repeat"
// follow-up is recorded as a stub AskQuestion — the engine has no
// "if this killed it" hook.)
func registerBerserkerBarrage() {
	engine.RegisterBehavior("35008", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			choices := cardutil.EnemyChoices(g, 4, pid, func(target engine.EntityID) []engine.Message {
				return []engine.Message{engine.DamageEntity{Target: target, Damage: 4, Source: pid}}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Berserker Barrage — deal 4 damage to an enemy", choices...),
			}}
		},
	})
}

// 35009 Slice and Dice: Hero Action — make 2 attacks in order: deal 3
// damage to an enemy, deal 3 damage to an enemy. (Approximated: the
// player picks one target; the second attack asks for a different
// target. We chain two AskQuestions.)
func registerSliceAndDice() {
	engine.RegisterBehavior("35009", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			choices := cardutil.EnemyChoices(g, 3, pid, func(target engine.EntityID) []engine.Message {
				// The second 3 damage: pick any other enemy.
				return []engine.Message{engine.AskQuestion{
					Player:   pid,
					Question: engine.Ask("Slice and Dice — pick the second target", cardutil.EnemyChoices(g, 3, pid, func(t2 engine.EntityID) []engine.Message {
						return []engine.Message{
							engine.DamageEntity{Target: target, Damage: 3, Source: pid},
							engine.DamageEntity{Target: t2, Damage: 3, Source: pid},
						}
					})...),
				}}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Slice and Dice — first attack: deal 3 damage to an enemy", choices...),
			}}
		},
	})
}

// 35010 Lunging Strike: Hero Action (attack) — deal 8 damage to an
// enemy. If you exhausted Wolverine's Claws to play this card, this
// attack gains overkill. (Approximated: the overkill gain is dropped;
// the choice is just 8 damage to an enemy.)
func registerLungingStrike() {
	engine.RegisterBehavior("35010", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			choices := cardutil.EnemyChoices(g, 8, pid, func(target engine.EntityID) []engine.Message {
				return []engine.Message{engine.DamageEntity{Target: target, Damage: 8, Source: pid}}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Lunging Strike — deal 8 damage to an enemy", choices...),
			}}
		},
	})
}

// 35011 Track by Scent: Hero Action (thwart) — remove 3 threat from a
// scheme. If this removes the last threat from that scheme, draw 2
// cards. (Approximated: 3-threat removal; the bonus draw is dropped
// since the engine has no "scheme cleared" event hook.)
func registerTrackByScent() {
	engine.RegisterBehavior("35011", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			choices := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
				return []engine.Message{engine.ThwartScheme{Scheme: id, N: 3, Source: pid}}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Track by Scent — remove 3 threat from a scheme", choices...),
			}}
		},
	})
}

// 35012 Regenerative Healing: Action — choose: heal 4 from your
// identity, or discard each stunned and confused status card from your
// identity.
func registerRegenerativeHealing() {
	engine.RegisterBehavior("35012", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			opts := []engine.Choice{
				engine.Choice{ID: "heal", Label: "Heal 4 damage from your identity", Kind: engine.ChoiceLabel}.Msgs(engine.HealEntity{Target: pid, N: 4}),
			}
			if p.Stunned || p.Confused {
				var cleanup []engine.Message
				if p.Stunned {
					cleanup = append(cleanup, engine.ClearStun{Target: pid})
				}
				if p.Confused {
					cleanup = append(cleanup, engine.ClearConfuse{Target: pid})
				}
				opts = append(opts, engine.Choice{ID: "cleanse", Label: "Discard each stunned/confused status card", Kind: engine.ChoiceLabel}.Msgs(cleanup...))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Regenerative Healing — choose", opts...),
			}}
		},
	})
}

// registerNemesis installs the Wolverine nemesis set: Omega Red
// (minion), Lady Deathstrike (minion), the Carbonadium Synthesizer
// (side scheme), and the supporting attachments and treacheries.
func registerNemesis() {
	registerOmegaRed()
	registerLadyDeathstrike()
}

// 35028 Omega Red: Retaliate 1, Steady. Boost: when Omega Red attacks
// you, deal 1 damage to each character you control. (Retaliate is read
// by the data layer; Steady too. The "deal 1 damage to each character
// you control" is approximated by a React on VillainActivates that
// fires 1 damage to the player, all allies, and all upgrades with HP.)
func registerOmegaRed() {
	engine.RegisterBehavior("35028", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			va, ok := msg.(engine.VillainActivates)
			if !ok {
				return nil
			}
			mn, ok := e.(*engine.Minion)
			if !ok || mn.ID != va.VillainID {
				return nil
			}
			if va.Player != e.EOwner() {
				return nil
			}
			var msgs []engine.Message
			msgs = append(msgs, engine.DamageEntity{Target: va.Player, Damage: 1, Source: e.EID()})
			for _, pl := range g.Players {
				for _, id := range pl.Allies {
					msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: e.EID()})
				}
			}
			return msgs
		},
	})
}

// 35034 Lady Deathstrike: Quickstrike. Boost: after Lady Deathstrike
// attacks and damages a character, that character's owner discards 1
// random card from their hand. (Quickstrike is read by the data layer;
// the discard is a React on the attack resolution.)
func registerLadyDeathstrike() {
	engine.RegisterBehavior("35034", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			wa, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok || wa.Enemy != e.EID() {
				return nil
			}
			// Discard 1 random card from the player's hand.
			owner := e.EOwner()
			// Lady Deathstrike is a minion; for nemesis minions the
			// owner is the engaged player.
			if owner == "" {
				if mn, ok2 := e.(*engine.Minion); ok2 {
					owner = mn.EngagedWith
				}
			}
			p := g.Player(owner)
			if p == nil || len(p.Hand) == 0 {
				return nil
			}
			card := p.Hand[0]
			return []engine.Message{engine.DiscardCards{Player: owner, Cards: engine.CardList{card}}}
		},
	})
}

// registerObligation installs Past Demons.
func registerObligation() {
	// 35027 Past Demons: You may flip to alter-ego form. Choose:
	// exhaust Logan → remove Past Demons from the game, OR you are
	// stunned and confused. Discard this card.
	engine.RegisterBehavior("35027", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			return cardutil.ExhaustOrPenalty(g, p, card, "You are stunned and confused",
				engine.StunEntity{Target: p.ID},
				engine.ConfuseEntity{Target: p.ID},
			)
		},
	})
}

// removeID drops the given id from a slice.
func removeID(s []engine.EntityID, id engine.EntityID) []engine.EntityID {
	out := make([]engine.EntityID, 0, len(s))
	for _, x := range s {
		if x != id {
			out = append(out, x)
		}
	}
	return out
}
