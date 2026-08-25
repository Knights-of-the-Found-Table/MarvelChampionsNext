// Package gambit registers the Gambit hero pack (37001): the
// Gambit / Remy LeBeau identity with its charge-counter economy
// (Charge de Card / Throw de Card / Thief Extraordinaire), the
// signature cards, the Guild Business obligation and the Belladonna
// nemesis set.
package gambit

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() {
	registerGambit()
	registerSignatures()
	registerNemesis()
	registerObligation()
}

// chargeOf reads the charge counters on Gambit's identity.
func chargeOf(p *engine.Player) int { return p.Counters }

// registerGambit installs the Gambit / Remy LeBeau identity (37001a/b).
//
// Ability order matters: Thief Extraordinaire must stay at index 0 —
// The Thieves Guild's response keys off RunAbility{Source: identity,
// Index: 0}.
func registerGambit() {
	engine.RegisterBehavior("37001", &engine.Behavior{
		// Throw de Card — Interrupt: when you play an [attack] event,
		// remove up to 3 charge counters → that event deals +1 damage
		// for each counter removed. The bonus rides the engine's
		// per-event damage channel (SetEventBonus, the Embiggen!
		// mechanism) so it lands on the event's first damage.
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			p := g.Player(e.EID())
			if p == nil {
				return nil
			}
			// Molecular Acceleration rider: when a 37010 resource card is
			// spent from hand, place 1 charge counter. (Resource cards are
			// never in-play entities, so the rider lives on the identity.)
			if pay, ok := msg.(engine.ResourcePay); ok && pay.Player == p.ID {
				n := 0
				for _, c := range pay.Cards {
					if c.Code == "37010" {
						n++
					}
				}
				if n > 0 {
					g.TLogf("c.molecularAccelerationChargeCounterS", n)
					return []engine.Message{engine.AddEntityCounter{ID: p.ID, N: n}}
				}
				return nil
			}
			m, ok := msg.(engine.EventPlayed)
			if !ok || m.Player != e.EID() {
				return nil
			}
			if !p.IsHero() || p.Counters <= 0 {
				return nil
			}
			def := m.Card.Def()
			if def.Type != "event" || !def.HasTrait("attack") {
				return nil
			}
			max := p.Counters
			if max > 3 {
				max = 3
			}
			choices := []engine.Choice{{
				ID: "keep", Label: engine.Tf("c.keepTheCounters"), Kind: engine.ChoicePass,
			}}
			for n := 1; n <= max; n++ {
				choices = append(choices, engine.Choice{
					ID:    fmt.Sprintf("throw-%d", n),
					Label: engine.Tf("c.removeChargeCounterSDamage", n, n),
					Kind:  engine.ChoiceLabel,
				}.Msgs(
					engine.AddEntityCounter{ID: p.ID, N: -n},
					engine.SetEventBonus{Player: p.ID, Damage: n},
				))
			}
			g.TLogf("c.throwDeCardMayBoost", p.Name, def)
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.throwDeCardRemoveUpTo3ChargeCounters"), choices...),
			}}
		},
		// Rogue (the signature ally) costs 1 less per charge counter on
		// the identity.
		CardCost: func(g *engine.Game, p *engine.Player, def *data.CardDef) int {
			if def.Code == "37002" && p.Counters > 0 {
				return p.Counters
			}
			return 0
		},
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			return []engine.Ability{
				// [0] Remy LeBeau — Thief Extraordinaire: Action
				// (thwart): Exhaust → look at the top 2 cards of the
				// encounter deck, discard 1 → remove threat from a
				// scheme equal to that card's boost icons.
				{
					Label:        engine.Tf("c.thiefExtraordinaireDiscard1OfTheTop2EncounterCardsRemoveThre"),
					Type:         engine.AbilityAction,
					Exhaust:      true,
					AlterEgoOnly: true,
					Execute:      thiefExtraordinaire,
				},
				// [1] Gambit — Charge de Card: Action: place 1 charge
				// counter here (once per round).
				{
					Label:        engine.Tf("c.chargeDeCardPlace1ChargeCounterOnGambit"),
					Type:         engine.AbilityAction,
					HeroOnly:     true,
					OncePerRound: true,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						g.TLogf("c.chargeDeCardGambitGainsAChargeCounter")
						return []engine.Message{engine.AddEntityCounter{ID: self, N: 1}}
					},
				},
			}
		},
	})
}

// thiefExtraordinaire runs Remy LeBeau's alter-ego action.
func thiefExtraordinaire(g *engine.Game, self engine.EntityID) []engine.Message {
	p := g.Player(self)
	if p == nil || len(g.EncounterDeck) == 0 {
		return nil
	}
	top := g.EncounterDeck
	if len(top) > 2 {
		top = top[:2]
	}
	var choices []engine.Choice
	for _, c := range top {
		c := c
		def := c.Def()
		n := cardutil.BoostOf(c)
		schemes := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
			return []engine.Message{
				engine.DiscardEncounterCard{Card: c},
				engine.ThwartScheme{Scheme: id, N: n, Source: self},
			}
		})
		if len(schemes) == 0 {
			continue
		}
		label := fmt.Sprintf("Discard %s (%d boost icon(s))", def.Name, n)
		choices = append(choices, engine.Choice{
			Label: engine.S(label), Kind: engine.ChoiceCard, CardCode: def.Code,
		}.WithThen(engine.Ask(engine.Tf("c.thiefExtraordinaireRemoveTheThreatFromWhichScheme"), schemes...)))
	}
	if len(choices) == 0 {
		return nil
	}
	return []engine.Message{engine.AskQuestion{
		Player:   p.ID,
		Question: engine.Ask(engine.Tf("c.thiefExtraordinaireDiscard1OfTheTop2EncounterCards"), choices...),
	}}
}

// registerSignatures installs Gambit's signature cards.
func registerSignatures() {
	registerRogueAlly()
	registerThievesGuild()
	registerGambitsStaff()
	registerGuildArmor()
	registerChargedCard()
	registerRoyalFlush()
	registerNaturalAgility()
	registerCreoleCharmer()
	registerMolecularAcceleration()
}

// 37002 Rogue (ally): cost reduction lives in the identity's CardCost
// hook; Toughness comes from the data layer keywords.
func registerRogueAlly() {
	engine.RegisterBehavior("37002", &engine.Behavior{})
}

// 37003 The Thieves Guild: Alter-Ego Response — after you resolve your
// "Thief Extraordinaire" ability, exhaust → remove 1 threat from a
// scheme; if this removes the last threat, draw 1 card. (Keys off the
// identity ability's RunAbility record, index 0 — see registerGambit.)
func registerThievesGuild() {
	engine.RegisterBehavior("37003", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.RunAbility)
			if !ok {
				return nil
			}
			s := g.Supports[e.EID()]
			if s == nil || s.Exhausted {
				return nil
			}
			p := g.Player(s.Owner)
			if p == nil || p.IsHero() {
				return nil
			}
			if m.Source != p.ID || m.Index != 0 {
				return nil
			}
			var choices []engine.Choice
			for _, id := range g.Schemes() {
				sc := g.Entity(id)
				msgs := []engine.Message{
					engine.ExhaustEntity{ID: s.ID},
					engine.ThwartScheme{Scheme: id, N: 1, Source: s.ID},
				}
				if t := schemeThreat(g, id); t <= 1 {
					msgs = append(msgs, engine.DrawCards{Player: p.ID, N: 1})
				}
				choices = append(choices, engine.Choice{
					Label: engine.S(sc.EDef().Name), Kind: engine.ChoiceTarget, SourceID: id, CardCode: sc.ECode(),
				}.Msgs(msgs...))
			}
			if len(choices) == 0 {
				return nil
			}
			g.TLogf("c.theThievesGuildRemove1ThreatAfterThiefExtraordinaire")
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.theThievesGuildRemove1ThreatFromAScheme"), choices...),
			}}
		},
	})
}

// schemeThreat reads a scheme's current threat.
func schemeThreat(g *engine.Game, id engine.EntityID) int {
	switch s := g.Entity(id).(type) {
	case *engine.MainScheme:
		return s.Threat
	case *engine.SideScheme:
		return s.Threat
	}
	return 0
}

// 37004 Gambit's Staff: Hero Interrupt — when an enemy attacks,
// exhaust → deal 1 damage to that enemy. (Approximation: fires after
// the attack resolves, off WindowAfterEnemyAttacked; the exhaust gates
// it to once per round.)
func registerGambitsStaff() {
	engine.RegisterBehavior("37004", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if u == nil || u.Exhausted {
				return nil
			}
			p := g.Player(u.Owner)
			if p == nil || !p.IsHero() {
				return nil
			}
			if g.Entity(w.Enemy) == nil {
				return nil
			}
			g.TLogf("c.gambitSStaff1DamageToTheAttacker")
			return []engine.Message{
				engine.ExhaustEntity{ID: u.ID},
				engine.DamageEntity{Target: w.Enemy, Damage: 1, Source: u.ID},
			}
		},
	})
}

// 37005 Gambit's Guild Armor: Hero Response — after Gambit defends
// against an attack and takes no damage, exhaust → ready Gambit.
func registerGuildArmor() {
	engine.RegisterBehavior("37005", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowDefended)
			if !ok {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if u == nil || u.Exhausted {
				return nil
			}
			p := g.Player(u.Owner)
			if p == nil || !p.IsHero() {
				return nil
			}
			if w.Defender != p.ID || w.DamageTaken != 0 {
				return nil
			}
			g.TLogf("c.gambitSGuildArmorReadies", p.Name)
			return []engine.Message{
				engine.ExhaustEntity{ID: u.ID},
				engine.ReadyEntity{ID: p.ID},
			}
		},
	})
}

// 37006 Charged Card: Hero Action (attack) — deal 4 damage to an
// enemy. (Approximation: the ranged/piercing/overkill riders from
// Throw de Card counters are not modeled; Throw de Card's +1-per-
// counter damage boost still applies through the event bonus.)
func registerChargedCard() {
	engine.RegisterBehavior("37006", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy(engine.Tf("c.chargedCardDeal4DamageToAnEnemy"),
			func(g *engine.Game, e engine.Entity) (int, []engine.Message) { return 4, nil }),
	})
}

// 37007 Royal Flush: Hero Action (attack) — place 1 charge counter on
// Gambit; deal 0 damage to an enemy three times. (Approximation: the
// three 0-damage pings collapse into a single 1-damage hit, which
// Throw de Card's event bonus still boosts.)
func registerRoyalFlush() {
	engine.RegisterBehavior("37007", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			msgs := []engine.Message{engine.AddEntityCounter{ID: pid, N: 1}}
			choices := cardutil.EnemyChoices(g, 1, pid, func(id engine.EntityID) []engine.Message {
				return []engine.Message{engine.DamageEntity{Target: id, Damage: 1, Source: pid}}
			})
			if len(choices) > 0 {
				msgs = append(msgs, engine.AskQuestion{
					Player:   pid,
					Question: engine.Ask(engine.Tf("c.royalFlushDeal1DamageToAnEnemy"), choices...),
				})
			}
			return msgs
		},
	})
}

// 37008 Natural Agility: Hero Interrupt (defense) — place 1 charge
// counter on Gambit → +1 DEF per charge counter on Gambit.
func registerNaturalAgility() {
	engine.RegisterBehavior("37008", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			if !p.IsHero() || p.Exhausted {
				return engine.Defends{}, nil, false
			}
			// The counter placed by this event counts for the bonus.
			d := engine.Defends{Defender: p.ID, Against: against, DefBonus: p.Counters + 1}
			return d, []engine.Message{engine.AddEntityCounter{ID: p.ID, N: 1}}, true
		},
	})
}

// 37009 Creole Charmer: Alter-Ego Action (thwart) — remove 3 threat
// from a scheme; if this removes the last threat, confuse the villain.
func registerCreoleCharmer() {
	engine.RegisterBehavior("37009", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			var choices []engine.Choice
			for _, id := range g.Schemes() {
				s := g.Entity(id)
				msgs := []engine.Message{engine.ThwartScheme{Scheme: id, N: 3, Source: pid}}
				if schemeThreat(g, id) <= 3 {
					for _, vid := range cardutil.SortedIDs(g.Villains) {
						msgs = append(msgs, engine.ConfuseEntity{Target: vid})
					}
				}
				choices = append(choices, engine.Choice{
					Label: engine.S(s.EDef().Name), Kind: engine.ChoiceTarget, SourceID: id, CardCode: s.ECode(),
				}.Msgs(msgs...))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.creoleCharmerRemove3ThreatFromAScheme"), choices...),
			}}
		},
	})
}

// 37010 Molecular Acceleration: resource card (energy + physical,
// from the data layer). The "when you spend this card, place 1 charge
// counter" rider is implemented on the identity's React (ResourcePay),
// since spent resource cards never become in-play entities.
func registerMolecularAcceleration() {
	engine.RegisterBehavior("37010", &engine.Behavior{})
}

// registerNemesis installs the Gambit nemesis encounter set
// (gambit_nemesis): Belladonna, The Assassins Guild, Guild Assassin
// and Assassination Attempt.
func registerNemesis() {
	// 37026 Belladonna: Quickstrike + Toughness come from the data
	// layer. Forced Response — after Belladonna attacks and defeats a
	// character, place 2 threat on the main scheme. (Approximation:
	// the engine has no "attack defeats a character" window, so an
	// ally of the engaged player being defeated while Belladonna is
	// engaged with them stands in for it.)
	engine.RegisterBehavior("37026", &engine.Behavior{
		React: nemesisAssassinReact("37026", 2),
	})

	// 37027 The Assassins Guild: Forced Response — after an [assassin]
	// minion attacks and defeats a character, place 2 threat here.
	// (Same approximation as Belladonna.)
	engine.RegisterBehavior("37027", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyDefeated)
			if !ok {
				return nil
			}
			s := g.SideSchemes[e.EID()]
			if s == nil {
				return nil
			}
			a := g.Allies[m.AllyID]
			if a == nil {
				return nil
			}
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn != nil && mn.EngagedWith == a.Owner && mn.EDef().HasTrait("assassin") {
					g.TLogf("c.theAssassinsGuildAnAssassinFelledACharacter2Threat")
					return []engine.Message{engine.SchemeThreat{Scheme: s.ID, N: 2, Source: s.ID}}
				}
			}
			return nil
		},
	})

	// 37028 Guild Assassin: Quickstrike. Forced Response — after Guild
	// Assassin attacks and defeats a character, place 1 threat on the
	// main scheme. (Same approximation as Belladonna.)
	engine.RegisterBehavior("37028", &engine.Behavior{
		React: nemesisAssassinReact("37028", 1),
	})

	// 37029 Assassination Attempt: When Revealed — each [assassin]
	// minion attacks you (even in alter-ego); if none is in play,
	// search the encounter deck and discard pile for one and reveal it.
	engine.RegisterBehavior("37029", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var msgs []engine.Message
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn != nil && mn.EDef().HasTrait("assassin") {
					msgs = append(msgs, engine.MinionActivates{MinionID: id, Player: p.ID})
				}
			}
			if len(msgs) > 0 {
				g.TLogf("c.assassinationAttemptEachAssassinAttacks", p.Name)
				return msgs
			}
			if card, ok := findAssassin(g); ok {
				mn := &engine.Minion{
					ID:        g.NextEntityID(engine.KindMinion),
					Code:      card.Code,
					MaxHP:     derefInt(card.Def().HP, 1),
					AttackVal: derefInt(card.Def().Attack, 0),
					SchemeVal: derefInt(card.Def().Scheme, 0),
				}
				g.Minions[mn.ID] = mn
				mn.EngagedWith = p.ID
				g.TLogf("c.assassinationAttemptIsFoundAndEngages", card, p.Name)
				return []engine.Message{engine.MinionEntersPlay{MinionID: mn.ID, Player: p.ID}}
			}
			return nil
		},
	})
}

// nemesisAssassinReact builds the shared "after this assassin attacks
// and defeats a character → N threat on the main scheme" approximation.
func nemesisAssassinReact(self string, n int) func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
	return func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.AllyDefeated)
		if !ok {
			return nil
		}
		mn := g.Minions[e.EID()]
		a := g.Allies[m.AllyID]
		if mn == nil || a == nil || mn.EngagedWith != a.Owner {
			return nil
		}
		if g.MainScheme == nil {
			return nil
		}
		g.TLogf("c.fellsACharacterThreatOnTheMainScheme", mn, n)
		return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: n, Source: e.EID()}}
	}
}

// findAssassin pulls the first assassin minion out of the encounter
// deck or discard pile.
func findAssassin(g *engine.Game) (engine.Card, bool) {
	for _, c := range g.EncounterDeck {
		if c.Def().Type == "minion" && c.Def().HasTrait("assassin") {
			g.EncounterDeck.Remove(c.ID)
			return c, true
		}
	}
	for _, c := range g.EncounterDiscard {
		if c.Def().Type == "minion" && c.Def().HasTrait("assassin") {
			g.EncounterDiscard.Remove(c.ID)
			return c, true
		}
	}
	return engine.Card{}, false
}

// registerObligation installs Guild Business (37025). The printed card
// lingers in play until removed; the engine has no persistent
// obligation zone, so it resolves immediately: exhaust Remy LeBeau to
// remove it from the game, or discard it. (The [energy] resource part
// of the removal cost is not modeled.)
func registerObligation() {
	engine.RegisterBehavior("37025", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			return cardutil.ExhaustOrPenalty(g, p, card, engine.Tf("c.discardGuildBusiness"))
		},
	})
}

// derefInt reads an optional printed stat.
func derefInt(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}
