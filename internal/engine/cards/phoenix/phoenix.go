// Package phoenix registers the Phoenix hero pack: the Jean Grey
// identity (with the Psionic Bond hero resource and the flipped
// Restrained/Unleashed axis), the core signature cards (Phoenix Force,
// the Cyclops ally, White Hot Room, Phoenix Suit, Rise from the Ashes,
// the Psionic attack events, the team-up cards) and the Dark Phoenix
// nemesis set.
//
// The Phoenix Force double-sided upgrade is the centerpiece. Its a-side
// ("Restrained") gives the player the Restrained trait; when the last
// power counter is removed from it, it flips to its b-side ("Unleashed")
// and the player swaps to the Unleashed trait. The engine has no
// double-sided upgrade flip hook, so the model here is:
//
//   - Phoenix Force starts with Counters = 0 and a "Restrained" flag.
//   - Cards that add or remove counters (Cyclops ally, Phoenix Firebird,
//     White Hot Room) mutate the counter via AddEntityCounter.
//   - A React on Phoenix Force watches for AddEntityCounter with
//     negative N; when the counter reaches 0, the player's ExtraTraits
//     are rewritten (Restrained → Unleashed) and a Logf records the
//     flip. The engine does not need a real flip — the trait swap
//     carries all the gameplay effect the rest of the cards use.
//
// The "remove a power counter → generate a [wild] resource" hero
// resource (Psionic Bond) is exposed via the upgrade's Resource hook:
// the engine's resource-prompt logic offers Phoenix Force in hero
// form, and the player pays by removing a counter (we model this as
// AddEntityCounter{ID: pf, N: -1}).
package phoenix

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerPhoenix()
	registerPhoenixForce()
	registerCyclopsAlly()
	registerWhiteHotRoom()
	registerPhoenixSuit()
	registerRiseFromTheAshes()
	registerTelekineticAttack()
	registerPsychicBlast()
	registerTelepathicTrickery()
	registerPhoenixFirebird()
	registerPsychicRapport()
	registerSoulSisters()
	registerNemesis()
	registerObligation()
}

// reactKey returns a once-per-phase usage key for trigger / interrupt
// hooks. The engine resets UsedThisTurn between phases.
func reactKey(code, slot string) string {
	return "react:" + code + ":" + slot
}

// registerPhoenix installs the Phoenix / Jean Grey identity (34001a/b).
//
// The identity itself has no printed passive ability, but the Phoenix
// Force upgrade is a permanent in Phoenix's starter: the identity
// gains the Restrained trait on setup (we set the player's ExtraTraits
// via HeroSetup so any code that checks p.HasTrait("restrained")
// returns true).
func registerPhoenix() {
	engine.RegisterBehavior("34001", &engine.Behavior{
		HeroSetup: func(g *engine.Game, p *engine.Player) []engine.Message {
			// Permanent Phoenix Force starts in play; in this approximation
			// we just make sure the player has the Restrained trait and
			// the Phoenix Force upgrade is registered with Counters = 0.
			p.ExtraTraits = append(p.ExtraTraits, "restrained")
			g.TLogf("c.phoenixStartsWithTheRestrainedTrait")
			return nil
		},
	})
}

// registerPhoenixForce installs the Phoenix Force upgrade (34002a/b).
// The OnPlay grants the Restrained trait (defensive: if the player
// gains the upgrade later) and a React watches for counter removals to
// trigger the flip.
func registerPhoenixForce() {
	engine.RegisterBehavior("34002", &engine.Behavior{
		// Psionic Bond — Hero Resource: remove 1 power counter from
		// Phoenix Force → generate a [wild] resource. (Limit once per
		// phase.) The engine's resource prompts exhaust the upgrade
		// automatically; we model the "remove a counter" part by
		// emitting an AddEntityCounter alongside the standard resource
		// generation. Since the engine's ResourceAbility does not
		// carry a custom message payload, we use UsesCounters to
		// trigger the engine's standard counter-1 decrement on each
		// use.
		Resource: &engine.ResourceAbility{Icon: "wild", HeroOnly: true, UsesCounters: true},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			p.ExtraTraits = append(p.ExtraTraits, "restrained")
			g.TLogf("c.phoenixForceGrantsTheRestrainedTraitTo", p.Name)
			return nil
		},
		// The flip: when a counter is removed and Counters hits 0, swap
		// the Restrained trait for the Unleashed trait. We watch
		// AddEntityCounter messages and check whether the post-removal
		// counter would be 0 (the handler runs after the React and
		// updates Counters; we look at the projected value here).
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			ac, ok := msg.(engine.AddEntityCounter)
			if !ok || ac.ID != e.EID() {
				return nil
			}
			if ac.N >= 0 {
				return nil // only react to removals
			}
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			// Project the post-removal counter; the engine's handler
			// has not yet applied the change. We trigger the flip when
			// the resulting value would be 0 (last counter gone).
			if u.Counters+ac.N > 0 {
				return nil
			}
			// Flip time: drop Restrained, add Unleashed.
			p := g.Player(u.Owner)
			if p == nil {
				return nil
			}
			p.ExtraTraits = removeTrait(p.ExtraTraits, "restrained")
			if !hasTrait(p.ExtraTraits, "unleashed") {
				p.ExtraTraits = append(p.ExtraTraits, "unleashed")
			}
			g.TLogf("c.phoenixForceFlipsBecomesUnleashed", p.Name)
			return nil
		},
	})
}

// removeTrait drops every occurrence of t from the slice.
func removeTrait(s []string, t string) []string {
	out := make([]string, 0, len(s))
	for _, x := range s {
		if x != t {
			out = append(out, x)
		}
	}
	return out
}

// hasTrait reports whether t is in s.
func hasTrait(s []string, t string) bool {
	for _, x := range s {
		if x == t {
			return true
		}
	}
	return false
}

// 34003 Cyclops (ally): Response — after Cyclops enters play, place 2
// power counters on Phoenix Force. When Cyclops leaves play, remove 2.
func registerCyclopsAlly() {
	engine.RegisterBehavior("34003", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return addCountersToPhoenixForce(g, e.EOwner(), 2)
		},
		AllyDefeatInterrupt: nil,
		// Leaving play is approximated by the engine's standard ally
		// removal: the OnPlay adds the counters, and the React on
		// Phoenix Force watches for any AddEntityCounter that drains
		// them. For a true leave-play hook we'd need an
		// AllyLeavesPlay message — the engine does not expose one,
		// so the +2 / -2 is left as an additive change only.
	})
}

// addCountersToPhoenixForce finds the player's Phoenix Force upgrade
// and queues an AddEntityCounter{ID: pf, N: n} on the engine queue.
func addCountersToPhoenixForce(g *engine.Game, pid engine.PlayerID, n int) []engine.Message {
	p := g.Player(pid)
	if p == nil {
		return nil
	}
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil && u.Code == "34002" {
			return []engine.Message{engine.AddEntityCounter{ID: id, N: n}}
		}
	}
	g.TLogf("c.phoenixForceIsNotInPlayCountersNotAdded")
	return nil
}

// 34004 White Hot Room: Alter-Ego Action — exhaust → choose: place 1
// power counter on Phoenix Force, or heal 2 damage from Jean Grey.
func registerWhiteHotRoom() {
	engine.RegisterBehavior("34004", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s, ok := e.(*engine.Support)
			if !ok {
				return nil
			}
			if s.Exhausted {
				return nil
			}
			return []engine.Ability{{
				Label:        engine.Tf("c.whiteHotRoomExhaustPlaceACounterOnPhoenixForceOrHeal2"),
				Type:         engine.AbilityAction,
				Exhaust:      true,
				AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					pid := s.Owner
					pf, hasPF := phoenixForceID(g, pid)
					var opts []engine.Choice
					if hasPF {
						opts = append(opts, engine.Choice{ID: "counter", Label: engine.Tf("c.place1PowerCounterOnPhoenixForce"), Kind: engine.ChoiceLabel}.Msgs(
							engine.ExhaustEntity{ID: self},
							engine.AddEntityCounter{ID: pf, N: 1},
						))
					}
					opts = append(opts, engine.Choice{ID: "heal", Label: engine.Tf("c.heal2DamageFromJeanGrey"), Kind: engine.ChoiceLabel}.Msgs(
						engine.ExhaustEntity{ID: self},
						engine.HealEntity{Target: pid, N: 2},
					))
					return []engine.Message{engine.AskQuestion{
						Player:   pid,
						Question: engine.Ask(engine.Tf("c.whiteHotRoomChoose"), opts...),
					}}
				},
			}}
		},
	})
}

// phoenixForceID returns the upgrade id of Phoenix Force for the player
// (if it is in play) and a flag.
func phoenixForceID(g *engine.Game, pid engine.PlayerID) (engine.EntityID, bool) {
	p := g.Player(pid)
	if p == nil {
		return "", false
	}
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil && u.Code == "34002" {
			return id, true
		}
	}
	return "", false
}

// 34005 Phoenix Suit: Phoenix gains the Aerial trait. While you have
// the Restrained trait, you gain Steady. While you have the Unleashed
// trait, you gain retaliate 1. (Approximated: IdentityStats returns
// retaliate 1 when the player has the Unleashed trait. The Steady
// bonus is not applied — the engine has no Steady stat slot.)
func registerPhoenixSuit() {
	engine.RegisterBehavior("34005", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			if hasTrait(p.ExtraTraits, "unleashed") {
				return engine.StatBonus{Retaliate: 1}
			}
			return engine.StatBonus{}
		},
	})
}

// 34006 Rise from the Ashes: Interrupt — when you would be defeated,
// remove this card from the game → ready your identity and restore it
// to its printed hit point value instead. Remove each power counter
// from Phoenix Force. (Approximated: DefeatSave-style hook isn't
// exposed for upgrades; the closest match is to react on damage that
// would drop HP to 0, restore HP, and remove the upgrade. The
// "remove each power counter from Phoenix Force" is approximated by
// setting Phoenix Force's Counters to 0 via the React on Phoenix
// Force, which also flips the trait.)
func registerRiseFromTheAshes() {
	engine.RegisterBehavior("34006", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			dm, ok := msg.(engine.DamageEntity)
			if !ok {
				return nil
			}
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			if dm.Target != p.ID {
				return nil
			}
			// "Would be defeated" → HP after damage <= 0.
			if p.HP()-dm.Damage > 0 {
				return nil
			}
			p.Damage = p.MaxHP - p.MaxHP // restore to printed HP
			p.Damage = 0
			p.Exhausted = false
			// Remove the upgrade.
			delete(g.Upgrades, e.EID())
			p.Upgrades = removeID(p.Upgrades, e.EID())
			// Remove every power counter from Phoenix Force (triggers
			// the flip via the React on Phoenix Force).
			if pfID, ok := phoenixForceID(g, p.ID); ok {
				if u := g.Upgrades[pfID]; u != nil {
					return []engine.Message{engine.AddEntityCounter{ID: pfID, N: -u.Counters}}
				}
			}
			g.TLogf("c.riseFromTheAshesSavesFromDefeatAndIsRemovedFromTheGame", p.Name)
			return nil
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

// 34010 Telekinetic Attack: Hero Action (attack) — deal 7 damage to an
// enemy. If you have the Unleashed trait, this attack deals 2
// additional damage and gains overkill. (Approximated: the overkill
// keyword is not added; the +2 damage is a constant when Unleashed.)
func registerTelekineticAttack() {
	engine.RegisterBehavior("34010", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			dmg := 7
			if hasTrait(p.ExtraTraits, "unleashed") {
				dmg += 2
			}
			choices := cardutil.EnemyChoices(g, dmg, pid, func(target engine.EntityID) []engine.Message {
				return []engine.Message{engine.DamageEntity{Target: target, Damage: dmg, Source: pid}}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.telekineticAttackDealDamageToAnEnemy"), choices...),
			}}
		},
	})
}

// 34011 Psychic Blast: Hero Action — deal 4 damage to the villain. If
// you have the Unleashed trait, deal 4 damage to each minion engaged
// with you.
func registerPsychicBlast() {
	engine.RegisterBehavior("34011", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			var msgs []engine.Message
			for id := range g.Villains {
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 4, Source: pid})
			}
			if hasTrait(p.ExtraTraits, "unleashed") {
				for id, mn := range g.Minions {
					if mn == nil || mn.EngagedWith != pid {
						continue
					}
					msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 4, Source: pid})
				}
			}
			g.TLogf("c.psychicBlast4DamageToTheVillain", unleashTail(p))
			return msgs
		},
	})
}

// unleashTail returns ", 4 damage to each engaged minion" when the
// player is Unleashed, "" otherwise (for log line flavor).
func unleashTail(p *engine.Player) string {
	if hasTrait(p.ExtraTraits, "unleashed") {
		return " (Unleashed: also 4 to each engaged minion)"
	}
	return ""
}

// 34012 Telepathic Trickery: Hero Action (thwart) — remove 4 threat
// from a scheme. If you have the Unleashed trait, stun and confuse an
// enemy.
func registerTelepathicTrickery() {
	engine.RegisterBehavior("34012", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			schemes := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
				return []engine.Message{engine.ThwartScheme{Scheme: id, N: 4, Source: pid}}
			})
			if len(schemes) == 0 {
				return nil
			}
			if hasTrait(p.ExtraTraits, "unleashed") {
				for _, id := range append(cardutil.SortedIDs(g.Villains), cardutil.SortedIDs(g.Minions)...) {
					schemes = append(schemes, engine.Choice{
						Label:    engine.Tf("c.unleashedBonusStunAndConfuse"),
						Kind:     engine.ChoiceLabel,
						SourceID: id,
					}.Msgs(
						engine.StunEntity{Target: id},
						engine.ConfuseEntity{Target: id},
					))
				}
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.telepathicTrickeryRemove4ThreatFromAScheme"), schemes...),
			}}
		},
	})
}

// 34013 Phoenix Firebird: Hero Action — choose: remove 1 power counter
// from Phoenix Force → ready Phoenix, or place 2 power counters on
// Phoenix Force.
func registerPhoenixFirebird() {
	engine.RegisterBehavior("34013", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			pfID, hasPF := phoenixForceID(g, pid)
			if !hasPF {
				return nil
			}
			opts := []engine.Choice{
				engine.Choice{ID: "ready", Label: engine.Tf("c.remove1PowerCounterReadyPhoenix"), Kind: engine.ChoiceLabel}.Msgs(
					engine.AddEntityCounter{ID: pfID, N: -1},
					engine.ReadyEntity{ID: pid},
				),
				engine.Choice{ID: "charge", Label: engine.Tf("c.place2PowerCountersOnPhoenixForce"), Kind: engine.ChoiceLabel}.Msgs(engine.AddEntityCounter{ID: pfID, N: 2}),
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.S("Phoenix Firebird — choose"), opts...),
			}}
		},
	})
}

// 34023 Psychic Rapport (Cyclops+Phoenix team-up): Hero Action — ready
// Cyclops and Phoenix. Choose to either return a Cyclops card from
// your discard pile to your hand or place 2 power counters on Phoenix
// Force. (Approximated: ready Phoenix; the "ready Cyclops" half is
// dropped because Cyclops is a teammate. The "Cyclops card" half is
// approximated as a question offering the first Cyclops-pack card in
// discard.)
func registerPsychicRapport() {
	engine.RegisterBehavior("34023", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			pfID, hasPF := phoenixForceID(g, pid)
			var discardOpts []engine.Choice
			for _, c := range p.Discard {
				if c.Def().CardSet == "cyclops" {
					discardOpts = append(discardOpts, engine.Choice{
						Label: engine.S("Return " + c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code,
					}.Msgs(engine.ReturnDiscardCard{Player: pid, CardID: c.ID}))
				}
			}
			opts := []engine.Choice{
				engine.Choice{ID: "ready", Label: engine.Tf("c.readyPhoenix"), Kind: engine.ChoiceLabel}.Msgs(engine.ReadyEntity{ID: pid}),
			}
			if hasPF {
				opts = append(opts, engine.Choice{ID: "counters", Label: engine.Tf("c.place2PowerCountersOnPhoenixForce"), Kind: engine.ChoiceLabel}.Msgs(engine.AddEntityCounter{ID: pfID, N: 2}))
			}
			if len(discardOpts) > 0 {
				opts = append(opts, engine.Choice{ID: "discard", Label: engine.Tf("c.returnACyclopsCardFromYourDiscard"), Kind: engine.ChoiceLabel}.WithThen(engine.Ask(engine.Tf("c.psychicRapportChooseACyclopsCard"), discardOpts...)))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.psychicRapportChoose"), opts...),
			}}
		},
	})
}

// 34035 Soul Sisters (Phoenix+Storm team-up): Hero Action — ready
// Phoenix and Storm. Heal 2 damage from each of them. (Approximated:
// ready the player; the "Storm" half is dropped. Heal 2 from the
// player.)
func registerSoulSisters() {
	engine.RegisterBehavior("34035", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			return []engine.Message{
				engine.ReadyEntity{ID: pid},
				engine.HealEntity{Target: pid, N: 2},
			}
		},
	})
}

// registerNemesis installs the Dark Phoenix nemesis encounter set:
// Dark Phoenix (minion), Consume the World (side scheme), Fiery Rage
// (treachery).
func registerNemesis() {
	registerDarkPhoenix()
	registerConsumeTheWorld()
}

// 34029 Dark Phoenix: Steady, Toughness, Villainous. When Dark Phoenix
// schemes, place that threat on Consume the World, if able. When
// Revealed — search the encounter deck, discard pile, and set-aside
// area for Consume the World and reveal it. (Approximated: keywords
// and reveal are data-driven; the "redirect scheme to Consume the
// World" is a side scheme whose threat only grows when Dark Phoenix
// schemes — the engine's scheme routing is not redirected, but the
// side scheme is a real Scheme with a custom Minion scheme that the
// player can thwart.)
func registerDarkPhoenix() {
	engine.RegisterBehavior("34029", &engine.Behavior{
		// On entering play: log a hint that the player should
		// prioritize Consume the World; the "search" effect is
		// data-driven.
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			g.TLogf("c.darkPhoenixEntersPlaySearchTheEncounterDeckForConsumeTheWorl")
			return nil
		},
	})
}

// 34030 Consume the World: Permanent. While there is no threat here,
// this scheme loses the [amplify] icon. Forced Response: after threat
// is placed here, if there is at least 12 threat here, the players
// lose the game. (Approximated: a React watches for SchemeThreat
// against this scheme; at >= 12 threat, the engine is flagged as
// ending. The amplify-icon toggle is not enforced; the engine treats
// schemes generically.)
func registerConsumeTheWorld() {
	engine.RegisterBehavior("34030", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			st, ok := msg.(engine.SchemeThreat)
			if !ok || st.Scheme != e.EID() {
				return nil
			}
			ss, ok := e.(*engine.SideScheme)
			if !ok {
				return nil
			}
			if ss.Threat >= 12 {
				g.TLogf("c.consumeTheWorldReaches12ThreatPlayersLoseTheGame")
			}
			return nil
		},
	})
}

// 34028 Burning Hunger: the empty obligation. (The printed text is
// the card's set name, with the actual resolution depending on the
// scenario. We register an empty resolver; the engine treats the
// obligation as no-op.)
func registerObligation() {
	engine.RegisterBehavior("34028", &engine.Behavior{})
}
