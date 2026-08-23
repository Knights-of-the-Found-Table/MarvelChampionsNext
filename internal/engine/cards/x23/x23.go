// Package x23 registers the X-23 hero pack (43001): the X-23 / Laura
// Kinney identity built around self-damage readies (Living Weapon) and
// the Honey Badger loop, the signature cards, the Self-Isolation
// obligation and the Lady Deathstrike nemesis set.
package x23

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerX23()
	registerSignatures()
	registerNemesis()
	registerObligation()
}

// honeyBadger returns the owner's in-play Honey Badger ally, if any.
func honeyBadger(g *engine.Game, p *engine.Player) *engine.Ally {
	for _, id := range p.Allies {
		if a := g.Allies[id]; a != nil && a.Code == "43003" {
			return a
		}
	}
	return nil
}

// registerX23 installs the X-23 / Laura Kinney identity (43001a/b).
func registerX23() {
	engine.RegisterBehavior("43001", &engine.Behavior{
		// Shhnk! — Setup: put X-23's Claws into play.
		HeroSetup: func(g *engine.Game, p *engine.Player) []engine.Message {
			for _, zone := range []engine.CardList{p.Hand, p.Deck, p.Discard} {
				for _, c := range zone {
					if c.Code == "43002" {
						return []engine.Message{engine.UpgradeEnterPlay{Player: p.ID, Card: c}}
					}
				}
			}
			return nil
		},
		// Living Weapon — Response: after X-23 takes any amount of
		// damage, ready her (once per phase; UsedThisTurn resets each
		// phase).
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.DamageEntity)
			if !ok || m.Target != e.EID() || m.Damage <= 0 {
				return nil
			}
			p := g.Player(e.EID())
			if p == nil || !p.IsHero() || !p.Exhausted {
				return nil
			}
			if g.UsedThisTurn["x23-living-weapon"] {
				return nil
			}
			g.UsedThisTurn["x23-living-weapon"] = true
			g.Logf("Living Weapon — %s readies", p.Name)
			return []engine.Message{engine.ReadyEntity{ID: p.ID}}
		},
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			return []engine.Ability{{
				// Laura Kinney — Action: shuffle Honey Badger or
				// Sisterly Bond from your discard into your deck →
				// draw 1 card (once per round).
				Label:        "Shuffle Honey Badger or Sisterly Bond from your discard into your deck → draw 1 card",
				Type:         engine.AbilityAction,
				AlterEgoOnly: true,
				OncePerRound: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					pl := g.Player(self)
					if pl == nil {
						return nil
					}
					var choices []engine.Choice
					for _, c := range pl.Discard {
						if c.Code == "43003" || c.Code == "43007" {
							choices = append(choices, engine.Choice{
								Label: "Shuffle in " + c.Def().Name, Kind: engine.ChoiceCard, CardCode: c.Code,
							}.Msgs(
								engine.ShuffleIntoDeck{Player: pl.ID, CardID: c.ID},
								engine.DrawCards{Player: pl.ID, N: 1},
							))
						}
					}
					if len(choices) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{
						Player:   pl.ID,
						Question: engine.Ask("Laura Kinney — shuffle which card into your deck?", choices...),
					}}
				},
			}}
		},
	})
}

// registerSignatures installs X-23's signature cards.
func registerSignatures() {
	registerClaws()
	registerHoneyBadger()
	registerAnimalInstinct()
	registerClawMastery()
	registerRegenerativeLongevity()
	registerSisterlyBond()
	registerSisterhood()
	registerAdamantiumLacing()
	registerGrimResolve()
	registerPainTolerance()
	registerPunctureWound()
}

// 43002 X-23's Claws: permanent. Hero Action — exhaust and take 2
// damage → X-23 gets +2 ATK until the end of the round.
// (Approximation: the engine's stat bonus channel expires at the end
// of the phase instead of the round.)
func registerClaws() {
	engine.RegisterBehavior("43002", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			return []engine.Ability{{
				Label:    "X-23's Claws — exhaust and take 2 damage: +2 ATK",
				Type:     engine.AbilityAction,
				Exhaust:  true,
				HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(u.Owner)
					if p == nil {
						return nil
					}
					g.Logf("X-23's Claws — %s takes 2 damage for +2 ATK", p.Name)
					return []engine.Message{
						engine.DamageEntity{Target: p.ID, Damage: 2, Source: u.ID},
						engine.ApplyStatBonus{Target: p.ID, ATK: 2},
					}
				},
			}}
		},
	})
}

// 43003 Honey Badger: Hero Response — after Honey Badger takes any
// amount of damage, ready X-23.
func registerHoneyBadger() {
	engine.RegisterBehavior("43003", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.DamageEntity)
			if !ok || m.Target != e.EID() || m.Damage <= 0 {
				return nil
			}
			p := g.Player(e.EOwner())
			if p == nil || !p.IsHero() || !p.Exhausted {
				return nil
			}
			g.Logf("Honey Badger — %s readies", p.Name)
			return []engine.Message{engine.ReadyEntity{ID: p.ID}}
		},
	})
}

// 43004 Animal Instinct: Hero Interrupt — when X-23 makes a basic
// thwart, she gets +X THW, X = her ATK. (Approximation: interrupts mid-
// basic-power are not hooked; played as an event it grants +ATK-as-THW
// until the end of the phase.)
func registerAnimalInstinct() {
	engine.RegisterBehavior("43004", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			atk := p.AttackStat(g)
			if atk < 0 {
				atk = 0
			}
			g.Logf("Animal Instinct — +%d THW this phase", atk)
			return []engine.Message{engine.ApplyStatBonus{Target: p.ID, THW: atk}}
		},
	})
}

// 43005 Claw Mastery: Hero Action — +2 ATK until the end of the round
// (approximated to end of phase; the overkill rider with Honey Badger
// and the max-1-per-round limit are not modeled).
func registerClawMastery() {
	engine.RegisterBehavior("43005", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.ApplyStatBonus{Target: e.EOwner(), ATK: 2}}
		},
	})
}

// 43006 Regenerative Longevity: Action — heal a total of 4 damage from
// your identity and Honey Badger, split as chosen.
func registerRegenerativeLongevity() {
	engine.RegisterBehavior("43006", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			hb := honeyBadger(g, p)
			var choices []engine.Choice
			choices = append(choices, engine.Choice{
				ID: "self-4", Label: "Heal 4 from " + p.Name, Kind: engine.ChoiceLabel,
			}.Msgs(engine.HealEntity{Target: pid, N: 4}))
			if hb != nil {
				choices = append(choices,
					engine.Choice{
						ID: "hb-4", Label: "Heal 4 from Honey Badger", Kind: engine.ChoiceLabel,
					}.Msgs(engine.HealEntity{Target: hb.ID, N: 4}),
					engine.Choice{
						ID: "split", Label: "Heal 2 from each", Kind: engine.ChoiceLabel,
					}.Msgs(
						engine.HealEntity{Target: pid, N: 2},
						engine.HealEntity{Target: hb.ID, N: 2},
					))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Regenerative Longevity — heal a total of 4 damage", choices...),
			}}
		},
	})
}

// 43007 Sisterly Bond: Hero Interrupt — when Honey Badger thwarts or
// attacks, add X-23's matching power to hers. (Approximation: played as
// an event, Honey Badger gets X-23's current ATK/THW as an
// until-end-of-phase bonus.)
func registerSisterlyBond() {
	engine.RegisterBehavior("43007", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			hb := honeyBadger(g, p)
			if hb == nil {
				g.Logf("Sisterly Bond — Honey Badger is not in play")
				return nil
			}
			atk, thw := p.AttackStat(g), p.ThwartStat(g)
			if atk < 0 {
				atk = 0
			}
			if thw < 0 {
				thw = 0
			}
			g.Logf("Sisterly Bond — Honey Badger gets +%d ATK / +%d THW this phase", atk, thw)
			return []engine.Message{engine.AllyStatBonus{Ally: hb.ID, ATK: atk, THW: thw}}
		},
	})
}

// 43008 Sisterhood: Action — exhaust and discard an X-23 card from
// your hand → search your deck and discard pile for Honey Badger and
// add her to your hand (shuffle).
func registerSisterhood() {
	engine.RegisterBehavior("43008", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			if s == nil {
				return nil
			}
			p := g.Player(s.Owner)
			if p == nil {
				return nil
			}
			hasX23Card := false
			for _, c := range p.Hand {
				if c.Def().CardSet == "x23" {
					hasX23Card = true
					break
				}
			}
			if !hasX23Card {
				return nil
			}
			return []engine.Ability{{
				Label:   "Sisterhood — discard an X-23 card: search for Honey Badger",
				Type:    engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					pl := g.Player(s.Owner)
					if pl == nil {
						return nil
					}
					// Find Honey Badger first.
					type hbZone struct {
						card   engine.Card
						inDeck bool
					}
					var found *hbZone
					for _, c := range pl.Deck {
						if c.Code == "43003" {
							found = &hbZone{card: c, inDeck: true}
							break
						}
					}
					if found == nil {
						for _, c := range pl.Discard {
							if c.Code == "43003" {
								found = &hbZone{card: c}
								break
							}
						}
					}
					if found == nil {
						g.Logf("Sisterhood — Honey Badger is not available")
						return nil
					}
					hb := found
					var choices []engine.Choice
					for _, c := range pl.Hand {
						if c.Def().CardSet != "x23" {
							continue
						}
						fetch := []engine.Message{
							engine.DiscardCards{Player: pl.ID, Cards: engine.CardList{c}},
						}
						if hb.inDeck {
							fetch = append(fetch,
								engine.TakeDeckCard{Player: pl.ID, CardID: hb.card.ID},
								engine.ShufflePlayerDeck{Player: pl.ID})
						} else {
							fetch = append(fetch,
								engine.ReturnDiscardCard{Player: pl.ID, CardID: hb.card.ID},
								engine.ShufflePlayerDeck{Player: pl.ID})
						}
						choices = append(choices, engine.Choice{
							Label: "Discard " + c.Def().Name, Kind: engine.ChoiceCard, CardCode: c.Code,
						}.Msgs(fetch...))
					}
					if len(choices) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{
						Player:   pl.ID,
						Question: engine.Ask("Sisterhood — discard an X-23 card to fetch Honey Badger", choices...),
					}}
				},
			}}
		},
	})
}

// 43009 Adamantium Lacing: +2 hit points; X-23 gains retaliate 1 and
// her basic attacks gain piercing (piercing is not modeled).
func registerAdamantiumLacing() {
	engine.RegisterBehavior("43009", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if p := g.Player(e.EOwner()); p != nil {
				p.MaxHP += 2
				g.Logf("Adamantium Lacing grants +2 HP to %s (now %d)", p.Name, p.MaxHP)
			}
			return nil
		},
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{Retaliate: 1}
		},
	})
}

// 43010 Grim Resolve: Resource — exhaust and take 1 damage → generate
// a [wild] resource. (Approximation: the self-damage is not modeled;
// the resource is generated via the declarative payment channel.)
func registerGrimResolve() {
	engine.RegisterBehavior("43010", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "wild"},
	})
}

// 43011 Pain Tolerance: Response — after you play an X-23 card
// (including this one), heal 1 damage from your identity.
// (Approximation: the hook covers event plays via EventPlayed and this
// upgrade's own entry; ally/support/upgrade plays emit no
// announcement.)
func registerPainTolerance() {
	engine.RegisterBehavior("43011", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.HealEntity{Target: e.EOwner(), N: 1}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.EventPlayed)
			if !ok {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if u == nil || m.Player != u.Owner || m.Card.Code == "43011" {
				return nil
			}
			if m.Card.Def().CardSet != "x23" {
				return nil
			}
			return []engine.Message{engine.HealEntity{Target: u.Owner, N: 1}}
		},
	})
}

// 43012 Puncture Wound: attach to an enemy X-23 or Honey Badger
// attacked this turn (the attacked-this-turn gate is not modeled).
// Attached enemy gets -1 ATK. Forced Response — after the player phase
// begins, discard this and deal 3 damage to the attached enemy.
func registerPunctureWound() {
	engine.RegisterBehavior("43012", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			choices := cardutil.EnemyChoices(g, 0, pid, func(id engine.EntityID) []engine.Message {
				return []engine.Message{
					engine.AttachUpgrade{ID: e.EID(), Target: id},
					engine.BoostEnemyAttack{Enemy: id, N: -1},
				}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Puncture Wound — attach to which enemy?", choices...),
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.BeginPhase)
			if !ok || m.Phase != engine.PhasePlayer {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if u == nil || u.AttachTo == "" || g.Entity(u.AttachTo) == nil {
				return nil
			}
			g.Logf("Puncture Wound — 3 damage to the attached enemy")
			return []engine.Message{
				engine.DamageEntity{Target: u.AttachTo, Damage: 3, Source: u.ID},
				engine.DiscardControlled{Player: u.Owner, ID: u.ID},
			}
		},
	})
}

// registerNemesis installs the X-23 nemesis set (x23_nemesis): Lady
// Deathstrike, In the Name of Vengeance, Cybermods, Critical Wound and
// Hack 'n' Slash.
func registerNemesis() {
	// 43029 Lady Deathstrike: When Defeated — the defeating player
	// discards the top card of the encounter deck and takes 1 indirect
	// damage per boost icon on it. (The defeater is not on the
	// message; the engaged player stands in. Indirect damage lands on
	// the identity.)
	engine.RegisterBehavior("43029", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionDefeated)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			mn := g.Minions[e.EID()]
			pid := cardutil.FirstPlayerID(g)
			if mn != nil && mn.EngagedWith != "" {
				pid = mn.EngagedWith
			}
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			card, ok := g.DrawEncounter()
			if !ok {
				return nil
			}
			g.EncounterDiscard = append(g.EncounterDiscard, card)
			n := cardutil.BoostOf(card)
			g.Logf("Lady Deathstrike — %s discards %s and takes %d damage", p.Name, card.Def().Name, n)
			if n == 0 {
				return nil
			}
			return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: n, Source: e.EID()}}
		},
	})

	// 43030 In the Name of Vengeance: each enemy gains retaliate 1.
	// (A global enemy retaliate aura is not modeled.)
	engine.RegisterBehavior("43030", &engine.Behavior{})

	// 43031 Cybermods: attach to Lady Deathstrike; when the attached
	// minion would be discarded, shuffle it into the encounter deck.
	// (The shuffle-back interrupt is not modeled.)
	engine.RegisterBehavior("43031", &engine.Behavior{})

	// 43032 Critical Wound: attach to your identity. Forced Interrupt —
	// when your turn ends, discard this card and take 4 damage.
	engine.RegisterBehavior("43032", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.PlayerTurnEnd)
			if !ok {
				return nil
			}
			t, ok2 := e.(*engine.Attachment)
			if !ok2 || t.Target != m.Player {
				return nil
			}
			g.Logf("Critical Wound — 4 damage at the end of the turn")
			return []engine.Message{
				engine.DamageEntity{Target: t.Target, Damage: 4, Source: t.ID},
				engine.DiscardAttachmentMsg{ID: t.ID},
			}
		},
	})

	// 43033 Hack 'n' Slash: When Revealed — discard 1 random card from
	// your hand and take damage equal to its printed resources.
	engine.RegisterBehavior("43033", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			return hackAndSlash(g, p, t.ID)
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			p := g.Player(cardutil.FirstPlayerID(g))
			if p == nil {
				return nil
			}
			return hackAndSlash(g, p, "")
		},
	})
}

// hackAndSlash discards a random hand card and damages the player by
// its printed resource count.
func hackAndSlash(g *engine.Game, p *engine.Player, src engine.EntityID) []engine.Message {
	if len(p.Hand) == 0 {
		return nil
	}
	c := p.Hand[g.Random(len(p.Hand))]
	p.Hand.Remove(c.ID)
	n := len(c.Def().Resources)
	g.Logf("Hack 'n' Slash — %s discards %s and takes %d damage", p.Name, c.Def().Name, n)
	msgs := []engine.Message{engine.DiscardCards{Player: p.ID, Cards: engine.CardList{c}}}
	if n > 0 {
		msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: n, Source: src})
	}
	return msgs
}

// registerObligation installs Self-Isolation (43028): Honey Badger is
// found and locked away (approximated: removed from the game with the
// obligation discarded); when she cannot be found, the obligation is
// discarded and a facedown encounter card is dealt. The
// recovery-discard response is not modeled.
func registerObligation() {
	engine.RegisterBehavior("43028", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			// Honey Badger in play?
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && a.Code == "43003" {
					g.Delete(id)
					g.Logf("Self-Isolation — Honey Badger is locked away")
					return []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
				}
			}
			for _, z := range []*engine.CardList{&p.Hand, &p.Deck, &p.Discard} {
				for _, c := range *z {
					if c.Code == "43003" {
						z.Remove(c.ID)
						g.Logf("Self-Isolation — Honey Badger is locked away")
						return []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
					}
				}
			}
			g.Logf("Self-Isolation — Honey Badger not found; a facedown encounter card is dealt")
			return []engine.Message{
				engine.ObligationResolve{Player: p.ID, Card: card},
				engine.DealEncounterToPlayer{Player: p.ID},
			}
		},
	})
}
