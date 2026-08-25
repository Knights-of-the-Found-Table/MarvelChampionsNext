// Package ghostspider registers the Ghost-Spider hero (27001) from the
// Sinister Motives box: the Ghost-Spider / Gwen Stacy identity with its
// interrupt/response-readiness engine (Dizzying Reflexes, Web-Bracelet),
// the George Stacy event-tucking support, Ticket to the Multiverse, the
// Worried Father obligation and The Lizard nemesis set.
package ghostspider

import (
	"strings"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerGhostSpider()
	registerGWSignatures()
	registerGWObligation()
	registerGWNemesis()
}

// isInterruptResponseEvent reports whether the event's text box carries an
// Interrupt or Response ability.
func isInterruptResponseEvent(code string) bool {
	text := engine.DB.MustLookup(code).Text
	return strings.Contains(text, "Interrupt</b>") || strings.Contains(text, "Response</b>")
}

// registerGhostSpider installs the Ghost-Spider / Gwen Stacy identity
// (27001a/b).
func registerGhostSpider() {
	engine.RegisterBehavior("27001", &engine.Behavior{
		// Dizzying Reflexes — Response: after you resolve an Interrupt or
		// Response ability on an event, ready Ghost-Spider (limit once per
		// phase). (Approximation: hooked on the event's play announcement,
		// which stands in for resolving its ability; UsedThisTurn resets at
		// each phase begin, matching the per-phase limit.)
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			p := g.Player(e.EID())
			if p == nil || !p.IsHero() || !p.Exhausted {
				return nil
			}
			var code string
			switch m := msg.(type) {
			case engine.EventPlayed:
				if m.Player == p.ID {
					code = m.Card.Code
				}
			case engine.PlayDefenseEvent:
				if m.Player == p.ID {
					code = m.Card.Code
				}
			}
			if code == "" || !isInterruptResponseEvent(code) {
				return nil
			}
			if g.UsedThisTurn == nil {
				g.UsedThisTurn = map[string]bool{}
			}
			if g.UsedThisTurn["gw-reflexes"] {
				return nil
			}
			g.UsedThisTurn["gw-reflexes"] = true
			g.TLogf("c.dizzyingReflexesGhostSpiderReadies")
			return []engine.Message{engine.ReadyEntity{ID: p.ID}}
		},
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			return []engine.Ability{{
				// Gwen Stacy — Action: shuffle Ticket to the Multiverse
				// from your discard pile into your deck, or ready George
				// Stacy (limit once per round).
				Label:        engine.Tf("c.gwenStacyRecoverTicketToTheMultiverseOrReadyGeorgeStacy"),
				Type:         engine.AbilityAction,
				AlterEgoOnly: true,
				OncePerRound: true,
				Execute:      gwenStacyAction,
			}}
		},
	})
}

// gwenStacyAction runs Gwen Stacy's alter-ego action.
func gwenStacyAction(g *engine.Game, self engine.EntityID) []engine.Message {
	p := g.Player(self)
	if p == nil {
		return nil
	}
	var choices []engine.Choice
	for _, c := range p.Discard {
		if c.Code == "27008" {
			choices = append(choices, engine.Choice{
				ID: "ticket", Label: engine.Tf("c.shuffleTicketToTheMultiverseIntoYourDeck"), Kind: engine.ChoiceCard, CardCode: c.Code,
			}.Msgs(engine.ShuffleIntoDeck{Player: p.ID, CardID: c.ID}))
			break
		}
	}
	for _, id := range p.Supports {
		if s := g.Supports[id]; s != nil && s.Code == "27007" && s.Exhausted {
			choices = append(choices, engine.Choice{
				ID: "george", Label: engine.Tf("c.readyGeorgeStacy"), Kind: engine.ChoiceTarget, SourceID: s.ID, CardCode: s.Code,
			}.Msgs(engine.ReadyEntity{ID: s.ID}))
		}
	}
	if len(choices) == 0 {
		return nil
	}
	return []engine.Message{engine.AskQuestion{
		Player:   p.ID,
		Question: engine.Ask(engine.Tf("c.gwenStacyChoose"), choices...),
	}}
}

// webWarriorCount counts the Web-Warrior cards a player controls
// (identity on its Web-Warrior side, allies, supports, upgrades).
func webWarriorCount(g *engine.Game, p *engine.Player) int {
	n := 0
	if p.EDef().HasTrait("web-warrior") {
		n++
	}
	for _, id := range p.Allies {
		if a := g.Allies[id]; a != nil && a.EDef().HasTrait("web-warrior") {
			n++
		}
	}
	for _, id := range p.Supports {
		if s := g.Supports[id]; s != nil && s.EDef().HasTrait("web-warrior") {
			n++
		}
	}
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil && u.EDef().HasTrait("web-warrior") {
			n++
		}
	}
	return n
}

func registerGWSignatures() {
	// 27002 Ghost Kick: Hero Response (attack) — after Ghost-Spider uses
	// a basic power, deal 6 damage to an enemy. (Approximation: the
	// basic-power trigger and the max-1-per-use limit are not enforced by
	// the event-play flow.)
	engine.RegisterBehavior("27002", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy(engine.Tf("c.ghostKickDeal6DamageToAnEnemy"),
			func(g *engine.Game, e engine.Entity) (int, []engine.Message) { return 6, nil }),
	})

	// 27003 Parental Guidance: Alter-Ego Action — if George Stacy is in
	// play, attach 1 event from your hand or discard pile facedown to
	// him; otherwise search your deck and discard pile for him and add
	// him to your hand (shuffle).
	engine.RegisterBehavior("27003", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			var george *engine.Support
			for _, id := range p.Supports {
				if s := g.Supports[id]; s != nil && s.Code == "27007" {
					george = s
				}
			}
			if george != nil {
				var choices []engine.Choice
				for _, c := range p.Hand {
					if c.Def().Type == "event" {
						choices = append(choices, engine.Choice{
							Label: engine.S(c.Def().Name + " (hand)"), Kind: engine.ChoiceCard, CardCode: c.Code,
						}.Msgs(engine.SupportStoreCard{ID: george.ID, Card: c}))
					}
				}
				for _, c := range p.Discard {
					if c.Def().Type == "event" {
						choices = append(choices, engine.Choice{
							Label: engine.S(c.Def().Name + " (discard)"), Kind: engine.ChoiceCard, CardCode: c.Code,
						}.Msgs(
							engine.ReturnDiscardCard{Player: pid, CardID: c.ID},
							engine.SupportStoreCard{ID: george.ID, Card: c},
						))
					}
				}
				if len(choices) == 0 {
					return nil
				}
				return []engine.Message{engine.AskQuestion{
					Player:   pid,
					Question: engine.Ask(engine.Tf("c.parentalGuidanceAttachAnEventToGeorgeStacy"), choices...),
				}}
			}
			var choices []engine.Choice
			for _, c := range p.Deck {
				if c.Code == "27007" {
					choices = append(choices, engine.Choice{
						Label: engine.Tf("c.georgeStacyDeck"), Kind: engine.ChoiceCard, CardCode: c.Code,
					}.Msgs(
						engine.TakeDeckCard{Player: pid, CardID: c.ID},
						engine.ShufflePlayerDeck{Player: pid},
					))
				}
			}
			for _, c := range p.Discard {
				if c.Code == "27007" {
					choices = append(choices, engine.Choice{
						Label: engine.Tf("c.georgeStacyDiscard"), Kind: engine.ChoiceCard, CardCode: c.Code,
					}.Msgs(engine.ReturnDiscardCard{Player: pid, CardID: c.ID}))
				}
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.parentalGuidanceFindGeorgeStacy"), choices...),
			}}
		},
	})

	// 27004 Phantom Flip: Hero Response (thwart) — after Ghost-Spider
	// uses a basic power, remove 5 threat from a scheme. (Same trigger
	// approximation as Ghost Kick.)
	engine.RegisterBehavior("27004", &engine.Behavior{
		OnPlay: cardutil.ChooseScheme(engine.Tf("c.chooseAScheme", "Phantom Flip"), func(g *engine.Game, e engine.Entity) int { return 5 }),
	})

	// 27005 Pirouette and Punch: Hero Interrupt — when a card is revealed
	// from the encounter deck, deal damage to the villain equal to 1 plus
	// that card's boost icons and cancel its When Revealed effects.
	// (Approximation: covers treachery reveals through the treachery
	// interrupt window; other card types' reveal windows are not
	// exposed.)
	engine.RegisterBehavior("27005", &engine.Behavior{
		TreacheryInterrupt: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			if !p.IsHero() {
				return nil
			}
			var msgs []engine.Message
			n := cardutil.BoostOf(card) + 1
			for _, id := range cardutil.SortedIDs(g.Villains) {
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: n, Source: p.ID})
			}
			return append(msgs, engine.TreacheryResolve{Player: p.ID, Card: card, Cancelled: true})
		},
	})

	// 27006 Web Binding: Hero Interrupt — when an enemy would activate,
	// cancel that activation; if it was a minion's, deal 4 damage to it.
	// (Approximation: activation cancellation is not exposed; standing in
	// for it, the event stuns the chosen enemy and pings a minion for 4.
	// The [mental] requirement is not enforceable by the payment hook.)
	engine.RegisterBehavior("27006", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			var choices []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				enemy := g.Entity(id)
				msgs := []engine.Message{engine.StunEntity{Target: id}}
				if _, isMinion := enemy.(*engine.Minion); isMinion {
					msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 4, Source: pid})
				}
				choices = append(choices, engine.Choice{
					Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: enemy.ECode(),
				}.Msgs(msgs...))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.webBindingCancelAnEnemySActivationStunIt"), choices...),
			}}
		},
	})

	// 27007 George Stacy: events attached here may be played as if in
	// hand. Action — exhaust → attach 1 event from your hand facedown
	// here (max 3). (Approximation: playing directly from under George is
	// not wired into the play flow; a second action retrieves the stored
	// events to hand, the Bruno Carrelli precedent.)
	engine.RegisterBehavior("27007", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			p := g.Player(e.EOwner())
			if s == nil || p == nil {
				return nil
			}
			var out []engine.Ability
			if !s.Exhausted && len(s.AttachedCards) < 3 {
				hasEvent := false
				for _, c := range p.Hand {
					if c.Def().Type == "event" {
						hasEvent = true
						break
					}
				}
				if hasEvent {
					out = append(out, engine.Ability{
						Label: engine.Tf("c.georgeStacyAttachAnEventFromYourHand"), Type: engine.AbilityAction, Exhaust: true,
						Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
							s := g.Supports[self]
							p := g.Player(s.Owner)
							var choices []engine.Choice
							for _, c := range p.Hand {
								if c.Def().Type != "event" {
									continue
								}
								choices = append(choices, engine.Choice{
									Label: engine.S(c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code,
								}.Msgs(engine.SupportStoreCard{ID: s.ID, Card: c}))
							}
							if len(choices) == 0 {
								return nil
							}
							return []engine.Message{engine.AskQuestion{
								Player:   p.ID,
								Question: engine.Ask(engine.Tf("c.georgeStacyAttachWhichEvent"), choices...),
							}}
						},
					})
				}
			}
			if len(s.AttachedCards) > 0 {
				out = append(out, engine.Ability{
					Label: engine.Tf("c.georgeStacyTakeTheStoredEventsIntoYourHand"), Type: engine.AbilityAction,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						s := g.Supports[self]
						if s == nil || len(s.AttachedCards) == 0 {
							return nil
						}
						return []engine.Message{engine.SupportRetrieveCards{ID: s.ID, Cards: s.AttachedCards}}
					},
				})
			}
			return out
		},
	})

	// 27008 Ticket to the Multiverse: Action — remove from game → discard
	// your hand, shuffle your discard pile into your deck, draw up to
	// your hand size, and ready each Ghost-Spider card you control.
	// (Approximation: remove-from-game becomes discard; "each Ghost-
	// Spider card you control" is the identity, as no other card shares
	// the title in the engine.)
	engine.RegisterBehavior("27008", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.ticketToTheMultiverseResetYourHandAndReadyGhostSpider"), Type: engine.AbilityAction,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					p := g.Player(e.EOwner())
					if u == nil || p == nil {
						return nil
					}
					g.TLogf("c.ticketToTheMultiverseResetsTheirHand", p.Name)
					msgs := []engine.Message{engine.DiscardControlled{Player: p.ID, ID: u.ID}}
					// Discard hand, then shuffle the whole discard pile
					// back (the ticket lands there too and rides along).
					if len(p.Hand) > 0 {
						msgs = append(msgs, engine.DiscardCards{Player: p.ID, Cards: append(engine.CardList{}, p.Hand...)})
					}
					p.Deck = append(p.Deck, p.Discard...)
					p.Discard = nil
					msgs = append(msgs,
						engine.ShufflePlayerDeck{Player: p.ID},
						engine.DrawCards{Player: p.ID, N: p.HandSize(g)},
						engine.ReadyEntity{ID: p.ID},
					)
					return msgs
				},
			}}
		},
	})

	// 27009 Web-Bracelet: Hero Response — after you resolve an Interrupt
	// or Response ability on an event, exhaust → draw 1 card.
	engine.RegisterBehavior("27009", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			u := g.Upgrades[e.EID()]
			if u == nil || u.Exhausted {
				return nil
			}
			p := g.Player(u.Owner)
			if p == nil || !p.IsHero() {
				return nil
			}
			var code string
			switch m := msg.(type) {
			case engine.EventPlayed:
				if m.Player == p.ID {
					code = m.Card.Code
				}
			case engine.PlayDefenseEvent:
				if m.Player == p.ID {
					code = m.Card.Code
				}
			}
			if code == "" || !isInterruptResponseEvent(code) {
				return nil
			}
			g.TLogf("c.webBraceletDraws1Card", p.Name)
			return []engine.Message{
				engine.ExhaustEntity{ID: u.ID},
				engine.DrawCards{Player: p.ID, N: 1},
			}
		},
	})

	// 27010 Silk (ally): Response — after you play her, if you control
	// another Web-Warrior card, search the encounter deck for a treachery
	// and discard it (shuffle).
	engine.RegisterBehavior("27010", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || webWarriorCount(g, p) < 2 {
				return nil
			}
			for _, c := range g.EncounterDeck {
				if c.Def().Type == "treachery" {
					g.TLogf("c.silkDiscardsFromTheEncounterDeck", c)
					return []engine.Message{engine.DiscardEncounterCard{Card: c}}
				}
			}
			return nil
		},
	})

	// 27011 Spider-Man (ally): Response — after you play him, stun and
	// confuse an enemy if you control at least 3 Web-Warrior cards.
	engine.RegisterBehavior("27011", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || webWarriorCount(g, p) < 3 {
				return nil
			}
			choices := cardutil.EnemyChoices(g, 0, p.ID, func(id engine.EntityID) []engine.Message {
				return []engine.Message{
					engine.StunEntity{Target: id},
					engine.ConfuseEntity{Target: id},
				}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.spiderManStunAndConfuseAnEnemy"), choices...),
			}}
		},
	})
}

// registerGWObligation installs Worried Father (27025): George Stacy is
// stripped from wherever he is; Gwen may exhaust herself to remove the
// obligation and take George back into hand.
func registerGWObligation() {
	engine.RegisterBehavior("27025", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			// Strip George Stacy from play / deck / hand into the discard
			// pile (attaching him facedown to the obligation is not
			// representable; the discard pile stands in for it).
			georgeCardID := ""
			for _, id := range p.Supports {
				if s := g.Supports[id]; s != nil && s.Code == "27007" {
					g.TLogf("c.worriedFatherGeorgeStacyLeavesPlay")
					g.Delete(s.ID)
					for i, sid := range p.Supports {
						if sid == s.ID {
							p.Supports = append(p.Supports[:i], p.Supports[i+1:]...)
							break
						}
					}
					george := engine.Card{ID: g.NextCardID(), Code: "27007", Owner: p.ID}
					p.Discard = append(p.Discard, george)
					georgeCardID = george.ID
				}
			}
			for _, zone := range []*engine.CardList{&p.Deck, &p.Hand, &p.Discard} {
				for _, c := range *zone {
					if c.Code == "27007" && c.ID != georgeCardID {
						zone.Remove(c.ID)
						p.Discard = append(p.Discard, c)
						georgeCardID = c.ID
					}
				}
			}
			removeMsgs := []engine.Message{engine.ExhaustEntity{ID: p.ID}}
			if georgeCardID != "" {
				removeMsgs = append(removeMsgs, engine.ReturnDiscardCard{Player: p.ID, CardID: georgeCardID})
			}
			removeMsgs = append(removeMsgs, engine.ObligationResolve{Player: p.ID, Card: card, Remove: true})
			exhaustChoice := engine.Choice{
				ID: "exhaust", Label: engine.Tf("c.exhaustGwenStacyRemoveWorriedFatherAndTakeGeorgeStacyIntoHan"), Kind: engine.ChoiceLabel,
			}.Msgs(removeMsgs...)
			discardChoice := engine.Choice{
				ID: "discard", Label: engine.Tf("c.discardWorriedFather"), Kind: engine.ChoiceLabel,
			}.Msgs(engine.ObligationResolve{Player: p.ID, Card: card})
			choices := []engine.Choice{exhaustChoice, discardChoice}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.worriedFatherChoose"), choices...),
			}}
		},
	})
}

// registerGWNemesis installs the Ghost-Spider nemesis set (Regenerative
// Research, The Lizard, Experimental Injection, In Cold Blood).
func registerGWNemesis() {
	// 27026 Regenerative Research: Forced Interrupt — when the villain
	// phase begins, heal 1 damage from each enemy.
	engine.RegisterBehavior("27026", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.BeginPhase)
			if !ok || m.Phase != engine.PhaseVillain {
				return nil
			}
			var msgs []engine.Message
			for _, id := range cardutil.SortedEnemyIDs(g) {
				msgs = append(msgs, engine.HealEntity{Target: id, N: 1})
			}
			return msgs
		},
	})

	// 27027 The Lizard: Forced Interrupt — when the villain phase
	// begins, heal 1 damage from The Lizard.
	engine.RegisterBehavior("27027", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.BeginPhase)
			if !ok || m.Phase != engine.PhaseVillain {
				return nil
			}
			return []engine.Message{engine.HealEntity{Target: e.EID(), N: 1}}
		},
	})

	// 27028 Experimental Injection: attach to the minion with the most
	// remaining hit points (surge if none); attached minion gains the
	// Creature trait and +4 hit points. (Approximation: the trait grant
	// is cosmetic in the engine; only the +4 HP is modeled.)
	engine.RegisterBehavior("27028", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			var best *engine.Minion
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn == nil {
					continue
				}
				if best == nil || mn.HP() > best.HP() {
					best = mn
				}
			}
			if best == nil {
				g.Delete(t.ID)
				return []engine.Message{engine.RevealNextEncounter{Player: cardutil.FirstPlayerID(g)}}
			}
			t.Target = best.ID
			best.Attachments = append(best.Attachments, t.ID)
			best.MaxHP += 4
			g.TLogf("c.experimentalInjectionGets4HitPoints", best)
			return nil
		},
	})

	// 27029 In Cold Blood: The Lizard attacks you (surge if he is not in
	// play). (Approximation: the "cannot play events until after the
	// attack" rider is not modeled.)
	engine.RegisterBehavior("27029", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn != nil && mn.Code == "27027" {
					g.TLogf("c.inColdBloodTheLizardAttacks", p.Name)
					return []engine.Message{engine.MinionActivates{MinionID: id, Player: p.ID}}
				}
			}
			return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
		},
	})
}
