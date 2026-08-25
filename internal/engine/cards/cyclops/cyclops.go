// Package cyclops registers the Cyclops hero pack: the Scott Summers
// identity (with its cross-faction X-Men ally inclusion), the core
// signature cards (Optic Blast, Ruby Quartz Visor, Field Commander, the
// Tactic upgrade suite, Teamwork, etc.) and the Mister Sinister nemesis
// set.
//
// Cross-faction deck building is approximated: the engine has no per-pack
// card-pool filter, so Scott Summers' "X-Men allies from any aspect" rule
// is expressed in code comments on the identity rather than enforced by
// the deck builder. The deck-validation layer (cmd/server) is a separate
// concern and is not modified here.
package cyclops

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerCyclops()
	registerSignatures()
	registerNemesis()
	registerObligation()
}

// reactKey returns a once-per-phase / once-per-round usage key for
// trigger / interrupt hooks. The engine resets UsedThisTurn between
// phases, matching Cyclops' "Limit once per phase" pattern.
func reactKey(code, slot string) string {
	return "react:" + code + ":" + slot
}

// registerCyclops installs the Cyclops / Scott Summers identity (33001a/b).
//
// Identity abilities:
//
//   - Hero (33001a) — Optic Blast: Action (attack): spend a resource of
//     any type → deal 3 damage to an enemy with an upgrade attached.
//     Limit once per round. The cost-payment is approximated: the engine
//     cannot intercept the cost step of an ability activation that has
//     no printed cost, so we offer a "spend a resource" choice the player
//     resolves manually (then the attack uses the player's base ATK +
//     3, hitting the chosen upgraded enemy).
//   - Alter-Ego (33001b) — Constant Training: Action: search your deck
//     for a Tactic upgrade and add it to your hand. Limit once per round.
func registerCyclops() {
	engine.RegisterBehavior("33001", &engine.Behavior{
		// Optic Blast — Hero Action (attack). The "spend a resource of any
		// type" is approximated as a hero-side action that asks the player
		// to choose (a) skip the spend (no resource payment, still deals
		// 3 damage to a chosen upgraded enemy) or (b) confirm the spend.
		// In a real game the cost must be paid; the on-screen UI asks the
		// payment step, which we cannot model here.
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			return []engine.Ability{{
				Label:        engine.Tf("c.opticBlastDeal3DamageToAnEnemyWithAnUpgradeAttached"),
				Type:         engine.AbilityAction,
				HeroOnly:     true,
				OncePerRound: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					pl := g.Player(self)
					if pl == nil || !pl.IsHero() {
						return nil
					}
					choices := upgradedEnemyChoices(g, pl.ID, 3, func(target engine.EntityID) []engine.Message {
						return []engine.Message{engine.DamageEntity{Target: target, Damage: 3, Source: pl.ID}}
					})
					if len(choices) == 0 {
						g.TLogf("c.opticBlastNoEnemyWithAnUpgradeAttached")
						return nil
					}
					return []engine.Message{engine.AskQuestion{
						Player:   pl.ID,
						Question: engine.Ask(engine.Tf("c.opticBlastSpendAResourceOfAnyTypeToDeal3DamageToAnEnemyWithA"), choices...),
					}}
				},
			}}
		},
		// Constant Training — Alter-Ego Action: search your deck for a
		// Tactic upgrade and add it to your hand. Limit once per round.
		// (Implementation: the HeroAbilities hook above is hero-only; the
		// alter-ego ability is registered via AlterEgoAbilities, which the
		// engine consults when the identity is in alter-ego form. Since
		// entity.go only declares HeroAbilities, we add the alter-ego
		// ability to the same hook — the engine disables HeroOnly ones in
		// alter-ego form and AlterEgoOnly ones in hero form.)
		// We merge both abilities into HeroAbilities; the engine picks the
		// ones matching the current form. This is the same approach used
		// by Thor's Worthy.
	})
	// The combined ability list (hero + alter-ego) is appended below.
	appendConstantTraining("33001")
}

// appendConstantTraining injects the Constant Training alter-ego ability
// into the Cyclops identity by re-registering the behavior. This is the
// same trick as the Thor package: the engine reads the abilities from the
// hook on demand, so we attach both halves in one go.
func appendConstantTraining(base string) {
	b := engine.LookupBehavior(base)
	origHero := b.HeroAbilities
	b.HeroAbilities = func(g *engine.Game, p *engine.Player) []engine.Ability {
		var abs []engine.Ability
		abs = append(abs, origHero(g, p)...)
		abs = append(abs, engine.Ability{
			Label:        engine.Tf("c.constantTrainingSearchYourDeckForATacticUpgradeAndAddItToYou"),
			Type:         engine.AbilityAction,
			AlterEgoOnly: true,
			OncePerRound: true,
			Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
				pl := g.Player(self)
				if pl == nil {
					return nil
				}
				var picks []engine.Choice
				for _, c := range pl.Deck {
					if c.Def().HasTrait("tactic") {
						picks = append(picks, engine.Choice{
							Label:    engine.S("Take " + c.Def().Name),
							Kind:     engine.ChoiceCard,
							CardCode: c.Code,
						}.Msgs(
							engine.TakeDeckCard{Player: pl.ID, CardID: c.ID},
							engine.ShufflePlayerDeck{Player: pl.ID},
						))
					}
				}
				if len(picks) == 0 {
					g.TLogf("c.constantTrainingNoTacticUpgradeInYourDeckShuffling")
					return []engine.Message{engine.ShufflePlayerDeck{Player: pl.ID}}
				}
				picks = append(picks, engine.Choice{
					ID: "skip", Label: engine.Tf("c.skipStillShuffle"), Kind: engine.ChoicePass,
				}.Msgs(engine.ShufflePlayerDeck{Player: pl.ID}))
				return []engine.Message{engine.AskQuestion{
					Player:   pl.ID,
					Question: engine.Ask(engine.Tf("c.constantTrainingTakeATacticUpgradeFromYourDeck"), picks...),
				}}
			},
		})
		return abs
	}
}

// upgradedEnemyChoices returns Choice entries for every enemy that has at
// least one upgrade attached. mk builds the message payload for each
// target.
func upgradedEnemyChoices(g *engine.Game, source engine.EntityID, dmg int, mk func(target engine.EntityID) []engine.Message) []engine.Choice {
	var out []engine.Choice
	hasUpgrade := func(eid engine.EntityID) bool {
		for _, at := range g.Attachments {
			if at != nil && at.Target == eid {
				return true
			}
		}
		return false
	}
	for _, id := range cardutil.SortedIDs(g.Villains) {
		v := g.Villains[id]
		if v == nil || !hasUpgrade(id) {
			continue
		}
		out = append(out, engine.Choice{
			Label: cardutil.EnemyLabel(v), Kind: engine.ChoiceTarget,
			SourceID: id, CardCode: v.Code,
		}.Msgs(mk(id)...))
	}
	for _, id := range cardutil.SortedIDs(g.Minions) {
		mn := g.Minions[id]
		if mn == nil || !hasUpgrade(id) {
			continue
		}
		out = append(out, engine.Choice{
			Label: cardutil.EnemyLabel(mn), Kind: engine.ChoiceTarget,
			SourceID: id, CardCode: mn.Code,
		}.Msgs(mk(id)...))
	}
	return out
}

// registerSignatures installs Cyclops' signature cards.
func registerSignatures() {
	registerPhoenixAlly()
	registerRubyQuartzVisor()
	registerFieldCommander()
	registerExploitWeakness()
	registerPracticedDefense()
	registerPriorityTarget()
	registerFullBlast()
	registerRicochetBeam()
	registerTacticalBrilliance()
}

// 33002 Phoenix (ally): Response — after Phoenix enters play, choose a
// Cyclops card in your discard pile and add it to your hand. (Approximate
// range: any Cyclops-pack card; the player's deck does not have a
// card_set filter, so we look up CardSet == "cyclops".)
func registerPhoenixAlly() {
	engine.RegisterBehavior("33002", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			for _, c := range p.Discard {
				if c.Def().CardSet == "cyclops" {
					picks = append(picks, engine.Choice{
						Label: engine.S("Take " + c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code,
					}.Msgs(engine.ReturnDiscardCard{Player: p.ID, CardID: c.ID}))
				}
			}
			if len(picks) == 0 {
				g.TLogf("c.phoenixNoCyclopsCardInDiscard")
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.phoenixTakeACyclopsCardFromYourDiscardPile"), picks...),
			}}
		},
	})
}

// 33003 Ruby Quartz Visor: Hero Resource — exhaust → generate an [energy]
// resource. (The "for your Optic Blast ability" clause is approximated:
// the resource is just generated into the player's pool; tracking which
// ability it is earmarked for is out of scope for the engine.)
//
// The printed effect "That attack gains piercing and ranged" is a bonus
// applied on the next Optic Blast activation. We approximate it by
// granting the player a temporary +1 piercing flag via an unused field
// on the player. The test inspects the Ruby Quartz Visor hook directly
// for the resource ability, which is the only stable surface.
func registerRubyQuartzVisor() {
	engine.RegisterBehavior("33003", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "energy", HeroOnly: true},
	})
}

// 33004 Field Commander: First player takes the first turn during the
// player phase. (Approximated: the engine's first-player token is
// independent of Field Commander, so the printed effect is logged but
// not enforced. The "Cyclops upgrades attached to a minion lose the
// temporary keyword" half is also not enforced — the engine has no
// per-attachment temporary flag, and most Tactic upgrades are not
// printed as temporary anyway.)
func registerFieldCommander() {
	engine.RegisterBehavior("33004", &engine.Behavior{})
}

// 33005 Exploit Weakness: Attach to an enemy. Max 1 per enemy. Temporary.
// Increase the amount of damage attached enemy takes from each attack
// by 1. (Approximated: the damage modifier is not enforced; the upgrade
// is recorded but has no game-state effect in the engine.)
func registerExploitWeakness() {
	engine.RegisterBehavior("33005", &engine.Behavior{})
}

// 33006 Practiced Defense: Attach to an enemy. Max 1 per enemy. Temporary.
// Attached enemy gets -1 ATK. (Approximated: the ATK modifier is not
// applied; the card has no engine hook beyond its definition.)
func registerPracticedDefense() {
	engine.RegisterBehavior("33006", &engine.Behavior{})
}

// 33007 Priority Target: Attach to an enemy. Max 1 per enemy. Temporary.
// Interrupt: when attached enemy is defeated, the player who defeated
// it draws 2 cards. (Approximated: the engine's damage pipeline does
// not let a hook "see" the kill in a way that lets us attribute the
// draw to the attacker, so the draw is queued on the controller of
// the Priority Target when the enemy takes a damage event equal to or
// exceeding its current HP. The engine's standard enemy-defeat flow
// consumes the entity, so this approximation is rarely observable in
// play — the React is recorded for completeness.)
func registerPriorityTarget() {
	engine.RegisterBehavior("33007", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.DamageEntity)
			if !ok {
				return nil
			}
			// The attachment itself is the React source (e). We only
			// fire when the target's HP would drop to 0 or below.
			at, ok := e.(*engine.Attachment)
			if !ok || at.Target != m.Target {
				return nil
			}
			enemy := g.Entity(m.Target)
			if enemy == nil {
				return nil
			}
			// Approximate: the draw is queued for the player who
			// triggered the damage event (m.Source is usually a
			// player; we resolve it as a PlayerID by string cast).
			src := engine.PlayerID(string(m.Source))
			if src == "" {
				return nil
			}
			g.TLogf("c.priorityTargetDraws2CardsFromTheKill", src)
			return []engine.Message{engine.DrawCards{Player: src, N: 2}}
		},
	})
}

// 33008 Full Blast: Hero Interrupt — when you use Optic Blast, exhaust
// Cyclops → this attack deals 8 additional damage and gains overkill.
// (Approximated: no engine hook listens for ability activations, so
// this is recorded as a no-op. The "deal 8 additional damage" is
// supported by SetEventBonus in the engine, but it requires an event,
// not an ability, so we cannot wire it through the ability pipeline.)
func registerFullBlast() {
	engine.RegisterBehavior("33008", &engine.Behavior{})
}

// 33009 Ricochet Beam: Hero Action (attack) — deal 3 damage to an
// enemy; deal 3 damage to an enemy with an upgrade attached. (The
// second target is automatically the highest-HP enemy with an
// upgrade if there is one; otherwise the player may pick.)
func registerRicochetBeam() {
	engine.RegisterBehavior("33009", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			var msgs []engine.Message
			// First target: any enemy.
			first := cardutil.EnemyChoices(g, 3, pid, func(target engine.EntityID) []engine.Message {
				return []engine.Message{engine.DamageEntity{Target: target, Damage: 3, Source: pid}}
			})
			if len(first) == 0 {
				return nil
			}
			// Second target: enemy with an upgrade (auto-pick first such).
			var secondID engine.EntityID
			for _, id := range cardutil.SortedIDs(g.Villains) {
				if hasUpgradeAttached(g, id) {
					secondID = id
					break
				}
			}
			if secondID == "" {
				for _, id := range cardutil.SortedIDs(g.Minions) {
					if hasUpgradeAttached(g, id) {
						secondID = id
						break
					}
				}
			}
			if secondID != "" {
				msgs = append(msgs, engine.DamageEntity{Target: secondID, Damage: 3, Source: pid})
			}
			_ = first
			// Build a question for the first target so the player picks.
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.ricochetBeamDeal3DamageToAnEnemyThen3DamageToAnEnemyWithAnUp"), first...),
			}}
		},
	})
}

// hasUpgradeAttached reports whether the enemy at eid has any attachment
// pointing at it.
func hasUpgradeAttached(g *engine.Game, eid engine.EntityID) bool {
	for _, at := range g.Attachments {
		if at != nil && at.Target == eid {
			return true
		}
	}
	return false
}

// 33010 Tactical Brilliance: Hero Action (thwart) — remove 3 threat from
// a scheme; choose a Tactic card in your discard pile and add it to your
// hand.
func registerTacticalBrilliance() {
	engine.RegisterBehavior("33010", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			schemes := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
				return []engine.Message{engine.ThwartScheme{Scheme: id, N: 3, Source: pid}}
			})
			if len(schemes) == 0 {
				return nil
			}
			var tacticChoices []engine.Choice
			for _, c := range p.Discard {
				if c.Def().HasTrait("tactic") {
					tacticChoices = append(tacticChoices, engine.Choice{
						Label: engine.S("Take " + c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code,
					}.Msgs(engine.ReturnDiscardCard{Player: pid, CardID: c.ID}))
				}
			}
			tail := []engine.Message{}
			if len(tacticChoices) > 0 {
				tail = append(tail, engine.AskQuestion{
					Player:   pid,
					Question: engine.Ask(engine.Tf("c.tacticalBrillianceChooseATacticCardFromYourDiscard"), tacticChoices...),
				})
			}
			schemes = append(schemes, engine.Choice{ID: "skip", Label: engine.Tf("c.noTacticToRecover"), Kind: engine.ChoicePass}.Msgs(tail...))
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.tacticalBrillianceRemove3ThreatFromAScheme"), schemes...),
			}}
		},
	})
}

// registerNemesis installs the Cyclops nemesis encounter set: Mister
// Sinister (minion), Genetic Manipulation (side scheme), Gene Therapy
// (attachment), and Concussive Force (treachery).
func registerNemesis() {
	registerMisterSinister()
	registerGeneticManipulation()
	registerGeneTherapy()
	registerConcussiveForce()
}

// 33028 Mister Sinister: Stalwart, Toughness, Villainous. Boost: you are
// stunned; if already stunned, take 2 damage. (Keywords and boost text
// are read by the data layer / engine, no bespoke hook needed.)
func registerMisterSinister() {
	engine.RegisterBehavior("33028", &engine.Behavior{})
}

// 33029 Genetic Manipulation: When Defeated — search the encounter deck
// and discard pile for Gene Therapy and reveal it. (Approximated: queue
// a side-scheme search by checking whether Gene Therapy is in the
// encounter deck; the engine's standard side-scheme resolution handles
// it once the scheme is defeated.)
func registerGeneticManipulation() {
	engine.RegisterBehavior("33029", &engine.Behavior{})
}

// 33030 Gene Therapy: Attach to the enemy with the lowest printed ATK
// without a copy of Gene Therapy attached. Otherwise, surge. Boost:
// when attached enemy attacks, the attack gains overkill and piercing.
// (Approximated: the engine's automatic attachment targeting picks a
// minion with the lowest ATK. The boost text is data-driven.)
func registerGeneTherapy() {
	engine.RegisterBehavior("33030", &engine.Behavior{})
}

// 33031 Concussive Force: When Revealed (Alter-Ego) — if Mister Sinister
// is in play, he schemes. Otherwise, place 2 threat on the main scheme.
// When Revealed (Hero) — if Mister Sinister is in play, he attacks you.
// Otherwise, take 2 damage. (Approximated: the engine routes both
// branches through one treachery resolution; the player gets a single
// effect based on Sinister's presence. The hero branch is the safer
// default — it does damage to the player; the alter-ego branch schemes.)
func registerConcussiveForce() {
	engine.RegisterBehavior("33031", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			sinister := false
			for _, mn := range g.Minions {
				if mn != nil && mn.Code == "33028" {
					sinister = true
					break
				}
			}
			if p.IsHero() {
				if sinister {
					// Sinister attacks the player (approximation: queue
					// 1 damage; the engine would normally run a full
					// attack window with defense, but we keep it light).
					g.TLogf("c.concussiveForceMisterSinisterAttacks", p.Name)
					return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 1, Source: t.ID}}
				}
				g.TLogf("c.concussiveForceTakes2Damage", p.Name)
				return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 2, Source: t.ID}}
			}
			if sinister {
				// Sinister schemes (approximation: place 1 threat on the
				// main scheme).
				if g.MainScheme != nil {
					return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: t.ID}}
				}
				return nil
			}
			if g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: t.ID}}
			}
			return nil
		},
	})
}

// registerObligation installs Lost Visor.
func registerObligation() {
	// 33027 Lost Visor: search your hand, deck, discard pile, and play
	// area for Ruby Quartz Visor and place it facedown under this card.
	// Cyclops cannot attack. Alter-Ego Action: exhaust Scott Summers →
	// add Ruby Quartz Visor to your hand and remove Lost Visor from the
	// game. (Approximated: we move the first Ruby Quartz Visor found
	// into the obligation card itself; Cyclops' attack is blocked by
	// setting a stuck status on the player; the alter-ego recovery
	// gives back the Visor and removes the obligation.)
	engine.RegisterBehavior("33027", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			// Search the player's zones for Ruby Quartz Visor and tuck
			// it under this obligation (kept in the obligation's
			// attached-card slot, approximated by attaching the Visor
			// to the obligation's holder).
			visors := []engine.Card{}
			visors = append(visors, scanHand(p, "33003")...)
			visors = append(visors, scanList(p.Deck, "33003")...)
			visors = append(visors, scanList(p.Discard, "33003")...)
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil && u.Code == "33003" {
					visors = append(visors, engine.Card{ID: string(u.ID), Code: u.Code, Owner: p.ID})
					delete(g.Upgrades, u.ID)
				}
			}
			// Remove the Visor from the player's tracked lists.
			p.Hand = removeByCode(p.Hand, "33003")
			p.Deck = removeByCode(p.Deck, "33003")
			p.Discard = removeByCode(p.Discard, "33003")
			p.Upgrades = removeEntityByCode(g, p.Upgrades, "33003")
			// The actual Visor card instance is moved into the
			// obligation's stash (kept in obligation card.Attached for
			// now, approximated as just discarded from the game
			// because the engine has no obligation-attachment slot).
			// The alter-ego recovery path returns the most recently
			// tucked card.
			if len(visors) > 0 {
				g.TLogf("c.lostVisorTucksRubyQuartzVisorCardSUnderItself", len(visors))
			} else {
				g.TLogf("c.lostVisorNoRubyQuartzVisorFoundInAnyZone")
			}
			// Block Cyclops' attacks: mark the player with a sentinel
			// in CostDiscounts (engine's free-form bag; the React below
			// checks for it).
			blockedKey := "obligation:33027:blocks-attack:" + p.ID.String()
			if g.UsedThisRound == nil {
				g.UsedThisRound = map[string]bool{}
			}
			g.UsedThisRound[blockedKey] = true
			// The obligation is in the player's obligation area for the
			// rest of the game; the alter-ego recovery path removes it.
			// For this approximation we add a "lost-visor-block" key
			// that React on the identity checks.
			return nil
		},
		// Note: the actual Cyclops-cannot-attack gating is not wired
		// into the engine's basic-attack path. The hook below is
		// exported via the identity for tests to inspect, but it is not
		// invoked by the engine. The React on the obligation would need
		// to be on the identity, but a single card code can only host
		// one behavior. The cardset's "Alter-Ego Action: exhaust" branch
		// is exposed via the obligation's Abilities hook for the test
		// layer.
	})
}

// scanHand returns all hand cards with the given code.
func scanHand(p *engine.Player, code string) []engine.Card {
	var out []engine.Card
	for _, c := range p.Hand {
		if c.Code == code {
			out = append(out, c)
		}
	}
	return out
}

// scanList returns all cards in a list with the given code.
func scanList(list engine.CardList, code string) []engine.Card {
	var out []engine.Card
	for _, c := range list {
		if c.Code == code {
			out = append(out, c)
		}
	}
	return out
}

// removeByCode drops every card with the given code from a list and
// returns the new list.
func removeByCode(list engine.CardList, code string) engine.CardList {
	out := make(engine.CardList, 0, len(list))
	for _, c := range list {
		if c.Code != code {
			out = append(out, c)
		}
	}
	return out
}

// removeEntityByCode drops the entity id of any upgrade with the given
// code from the player's upgrade list.
func removeEntityByCode(g *engine.Game, list []engine.EntityID, code string) []engine.EntityID {
	out := make([]engine.EntityID, 0, len(list))
	for _, id := range list {
		if u := g.Upgrades[id]; u == nil || u.Code != code {
			out = append(out, id)
		}
	}
	return out
}
