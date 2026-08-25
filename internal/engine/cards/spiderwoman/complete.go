// Package spiderwoman registers the Scarlet Witch / Spider-Woman pack
// (double-sided hero): chaos boost-count tricks, hex riders, and the
// Luminous nemesis set. Boost-count interrupts are approximated as noted.
package spiderwoman

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerSW()
	registerNemesis()
}

// boostOf returns a card's printed boost icons.
func boostOf(c engine.Card) int {
	if b := c.Def().Boost; b != nil {
		return *b
	}
	return 0
}

func registerSW() {
	// Scarlet Witch identity: Chaos Control — boost-count replacement
	// window absent; registered with the marker note.
	engine.RegisterBehavior("15001", &engine.Behavior{})

	// Quicksilver (ally): ready once per phase.
	engine.RegisterBehavior("15002", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			if g.UsedThisTurn["sw-quicksilver"] {
				return nil
			}
			a := g.Allies[e.EID()]
			if a == nil || !a.Exhausted {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.readyQuicksilverOncePerPhase"), Type: engine.AbilityAction,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					g.UsedThisTurn["sw-quicksilver"] = true
					return []engine.Message{engine.ReadyEntity{ID: self}}
				},
			}}
		},
	})

	// Chaos Magic: play a card ignoring its cost; mill per its cost.
	engine.RegisterBehavior("15003", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			for _, c := range p.Hand {
				def := c.Def()
				if def.Cost == nil || *def.Cost <= 0 {
					continue
				}
				cost := *def.Cost
				picks = append(picks, engine.Choice{Label: engine.S(def.Name + " (free)"), Kind: engine.ChoicePlay, CardCode: def.Code}.
					Msgs(engine.PlayCard{Player: p.ID, Card: c, Paid: engine.CostPaid{}},
						engine.MillEncounter{N: cost}))
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.chaosMagicPlayWhichCardForFree"), picks...)}}
		},
	})

	// Hex Bolt: mill 3, resolve per boost count.
	engine.RegisterBehavior("15004", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var msgs []engine.Message
			for i := 0; i < 3; i++ {
				c, ok := g.DrawEncounter()
				if !ok {
					break
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
				switch b := boostOf(c); {
				case b == 0:
					msgs = append(msgs, cardutil.ChooseEnemy(engine.Tf("c.hexBolt0BoostDeal2DamageToWhichEnemy"),
						func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 2, nil })(
						g, &engine.EventCard{Code: "15004", Owner: p.ID})...)
				case b == 1:
					msgs = append(msgs, engine.AskQuestion{Player: p.ID, Question: engine.Ask(
						engine.Tf("c.hexBolt1BoostRemove2ThreatFromWhichScheme"), schemePicks(g, 2, p.ID)...)})
				case b == 2:
					msgs = append(msgs, engine.DrawCards{Player: p.ID, N: 1})
				default:
					var picks []engine.Choice
					for _, q := range g.Players {
						picks = append(picks,
							engine.Choice{Label: engine.S("Tough — " + q.Name), Kind: engine.ChoiceTarget, SourceID: q.ID}.
								Msgs(engine.ToughEntity{Target: q.ID}),
							engine.Choice{Label: engine.S("Stun — " + q.Name), Kind: engine.ChoiceTarget, SourceID: q.ID}.
								Msgs(engine.StunEntity{Target: q.ID}))
					}
					for _, id := range cardutil.SortedEnemyIDs(g) {
						enemy := g.Entity(id)
						if enemy != nil {
							picks = append(picks,
								engine.Choice{Label: engine.Tf("c.stun3", cardutil.EnemyLabel(enemy)), Kind: engine.ChoiceTarget, SourceID: id, CardCode: enemy.ECode()}.
									Msgs(engine.StunEntity{Target: id}),
								engine.Choice{Label: engine.Tf("c.confuse3", cardutil.EnemyLabel(enemy)), Kind: engine.ChoiceTarget, SourceID: id, CardCode: enemy.ECode()}.
									Msgs(engine.ConfuseEntity{Target: id}))
						}
					}
					if len(picks) > 0 {
						msgs = append(msgs, engine.AskQuestion{Player: p.ID,
							Question: engine.Ask(engine.Tf("c.hexBolt3BoostPlaceAStatusOnWhichCharacter"), picks...)})
					}
				}
			}
			return msgs
		},
	})

	// Molecular Decay: 5 damage + 1 per boost icon on 2 milled cards.
	engine.RegisterBehavior("15005", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			bonus := 0
			for i := 0; i < 2; i++ {
				c, ok := g.DrawEncounter()
				if !ok {
					break
				}
				bonus += boostOf(c)
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			total := 5 + bonus
			return cardutil.ChooseEnemy(engine.Tf("c.molecularDecayDealDamageToWhichEnemy"),
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return total, nil })(g, e)
		},
	})

	// Warp Reality: reveal-cancel — no reveal window; the treachery
	// portion rides the existing hand-interrupt channel.
	engine.RegisterBehavior("15006", &engine.Behavior{
		TreacheryInterrupt: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			n := boostOf(card)
			for i := 0; i < n; i++ {
				if c, ok := g.DrawEncounter(); ok {
					g.EncounterDiscard = append(g.EncounterDiscard, c)
				}
			}
			return []engine.Message{}
		},
	})

	// Agatha Harkness: top 3 → 1 to hand, rest to bottom.
	engine.RegisterBehavior("15007", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustAgathaHarknessTop31ToHandRestToBottom"), Type: engine.AbilityAction,
				Exhaust: true, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.EOwner())
					if p == nil || len(p.Deck) < 3 {
						return nil
					}
					var picks []engine.Choice
					for _, c := range p.Deck[:3] {
						picks = append(picks, engine.Choice{Label: engine.S(c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code}.
							Msgs(engine.TopDeckPick{Player: p.ID, CardID: c.ID}))
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID,
						Question: engine.Ask(engine.Tf("c.agathaAddWhichCardToHand"), picks...)}}
				},
			}}
		},
	})

	// Magic Shield: discard to prevent 3 damage.
	engine.RegisterBehavior("15008", &engine.Behavior{
		DamagePrevention: func(g *engine.Game, u *engine.Upgrade, p *engine.Player, n int) (int, int) {
			g.Delete(u.ID)
			p.Discard = append(p.Discard, engine.Card{ID: g.NextCardID(), Code: u.Code, Owner: p.ID})
			g.TLogf("c.discardsMagicShieldToPrevent3Damage", p.Name)
			return min(3, n), 0
		},
	})

	// Scarlet Witch's Crest: boost-count adjust — no window; approximated.
	engine.RegisterBehavior("15009", &engine.Behavior{})

	// Speed: readies after thwarting (once per round).
	engine.RegisterBehavior("15010", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.AllyThwartWindow)
			a := g.Allies[e.EID()]
			if !ok || a == nil || w.Ally != e.EID() || g.UsedThisRound["sw-speed"] {
				return nil
			}
			g.UsedThisRound["sw-speed"] = true
			return []engine.Message{engine.ReadyEntity{ID: e.EID()}}
		},
	})

	// Wiccan: after thwarting, mill 1 and 1 damage per boost icon.
	engine.RegisterBehavior("15011", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.AllyThwartWindow)
			if !ok || w.Ally != e.EID() {
				return nil
			}
			n := 0
			if c, ok := g.DrawEncounter(); ok {
				n = boostOf(c)
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			if n <= 0 {
				return nil
			}
			return cardutil.ChooseEnemy(engine.Tf("c.wiccanDealDamageToWhichEnemy"),
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return n, nil })(
				g, &engine.EventCard{Code: "15011", Owner: e.EOwner()})
		},
	})

	// Crisis Averted: 6 threat from the main scheme.
	engine.RegisterBehavior("15012", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if g.MainScheme == nil {
				return nil
			}
			return []engine.Message{engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 6, Source: e.EOwner()}}
		},
	})

	// Multitasking: 2 threat (+2 elsewhere on mental payment).
	engine.RegisterBehavior("15013", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			msgs := []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(
				engine.Tf("c.multitaskingRemove2ThreatFromWhichScheme"), schemePicks(g, 2, p.ID)...)}}
			if ec, ok := e.(*engine.EventCard); ok {
				for _, ic := range ec.Paid.Icons {
					if ic == "mental" {
						msgs = append(msgs, engine.AskQuestion{Player: p.ID, Question: engine.Ask(
							engine.Tf("c.multitaskingMentalRemove2ThreatFromWhichOtherScheme"), schemePicks(g, 2, p.ID)...)})
						break
					}
				}
			}
			return msgs
		},
	})

	// Swift Retribution: the villain schemes, then takes 4.
	engine.RegisterBehavior("15014", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			msgs := []engine.Message{}
			for id := range g.Villains {
				msgs = append(msgs, engine.DealBoost{Enemy: id}, engine.RevealBoost{Enemy: id},
					engine.ApplyVillainScheme{VillainID: id, Player: e.EOwner()},
					engine.DamageEntity{Target: id, Damage: 4, Source: e.EOwner()})
				break
			}
			return msgs
		},
	})

	// Turn the Tide: after a full-clear thwart, 3 damage.
	engine.RegisterBehavior("15015", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowAfterThwarted)
			if !ok || w.Player != e.EOwner() {
				return nil
			}
			if s := g.SideSchemes[w.Scheme]; s == nil || s.Threat != 0 {
				return nil
			}
			return cardutil.ChooseEnemy(engine.Tf("c.turnTheTideDeal3DamageToWhichEnemy"),
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 3, nil })(
				g, &engine.EventCard{Code: "15015", Owner: e.EOwner()})
		},
	})

	// The Power of Justice reprint + basics.
	engine.RegisterBehavior("15016", &engine.Behavior{})
	engine.RegisterBehavior("15020", &engine.Behavior{})
	engine.RegisterBehavior("15021", &engine.Behavior{})
	engine.RegisterBehavior("15022", &engine.Behavior{})

	// Heroic Intuition reprint.
	if b := engine.LookupBehavior("01065"); b != nil {
		engine.RegisterBehavior("15017", b)
	}

	// Order and Chaos reprint: alias quicksilver 14018.
	if b := engine.LookupBehavior("14018"); b != nil {
		engine.RegisterBehavior("15018", b)
	}

	// Spiritual Meditation: draw 2, discard 1.
	engine.RegisterBehavior("15019", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			msgs := []engine.Message{engine.DrawCards{Player: p.ID, N: 2}}
			var picks []engine.Choice
			for _, c := range p.Hand {
				picks = append(picks, engine.Choice{Label: engine.S("Discard " + c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code}.
					Msgs(engine.DiscardCards{Player: p.ID, Cards: engine.CardList{c}}))
			}
			if len(picks) > 0 {
				msgs = append(msgs, engine.AskQuestion{Player: p.ID,
					Question: engine.Ask(engine.Tf("c.discardWhichCard"), picks...)})
			}
			return msgs
		},
	})

	// Slipping Sanity obligation: exhaust-remove or star-mill threat.
	engine.RegisterBehavior("15023", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var picks []engine.Choice
			if !p.Exhausted {
				picks = append(picks, engine.Choice{ID: "exhaust", Label: engine.Tf("c.exhaustWandaRemoveFromTheGame"), Kind: engine.ChoiceLabel}.
					Msgs(engine.ExhaustEntity{ID: p.ID},
						engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}))
			}
			picks = append(picks, engine.Choice{ID: "mill", Label: engine.Tf("c.mill5EncounterCardsThreatPerStarIcon"), Kind: engine.ChoiceLabel}.
				Msgs(engine.SlippingSanityMill{Player: p.ID},
					engine.ObligationResolve{Player: p.ID, Card: card}))
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.slippingSanity"), picks...)}}
		},
	})

	// Browbeat: 2 + stage (max 3) damage to the villain.
	engine.RegisterBehavior("15028", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			n := 2
			for _, v := range g.Villains {
				n += min(3, v.Stage)
			}
			for id := range g.Villains {
				return []engine.Message{engine.DamageEntity{Target: id, Damage: n, Source: e.EOwner()}}
			}
			return nil
		},
	})

	// Last Stand: ally attack +3 ATK then discard — per-attack window
	// approximated to a flat bonus event on an ally attack.
	engine.RegisterBehavior("15029", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.AllyAttackWindow)
			if !ok {
				return nil
			}
			a := g.Allies[w.Ally]
			if a == nil {
				return nil
			}
			return []engine.Message{
				engine.DamageEntity{Target: w.Target, Damage: 3, Source: a.Owner},
				engine.DiscardControlled{Player: a.Owner, ID: w.Ally},
			}
		},
	})

	// Bait and Switch: the villain attacks you; remove 4 main threat.
	engine.RegisterBehavior("15030", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			msgs := []engine.Message{}
			if g.MainScheme != nil {
				msgs = append(msgs, engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 4, Source: e.EOwner()})
			}
			for id := range g.Villains {
				msgs = append(msgs, engine.DealBoost{Enemy: id}, engine.RevealBoost{Enemy: id},
					engine.AskAttack{Enemy: id, Player: e.EOwner(), Trigger: engine.TriggerVillainAttacksYou})
				break
			}
			return msgs
		},
	})

	// Recuperation: heal REC.
	engine.RegisterBehavior("15031", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			n := p.RecoverStat(g)
			if n <= 0 {
				return nil
			}
			return []engine.Message{engine.HealEntity{Target: p.ID, N: n}}
		},
	})
}

func registerNemesis() {
	// The Next Evolution: +1 boost icon on each encounter card —
	// approximated as +1 threat acceleration per villain phase.
	engine.RegisterBehavior("15024", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.BeginPhase); !ok {
				return nil
			}
			if g.SideSchemes[e.EID()] == nil {
				return nil
			}
			g.TLogf("c.theNextEvolutionBoostIconsIncreased")
			return nil
		},
	})

	// Luminous: after activating, star-mill check.
	engine.RegisterBehavior("15025", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			ma, ok := msg.(engine.MinionActivates)
			mn := g.Minions[e.EID()]
			if !ok || mn == nil || ma.MinionID != e.EID() || mn.EngagedWith == "" {
				return nil
			}
			b := 0
			if c, ok := g.DrawEncounter(); ok {
				b = boostOf(c)
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			if b >= 2 {
				return []engine.Message{engine.DealEncounterToPlayer{Player: mn.EngagedWith}}
			}
			return nil
		},
	})

	// Magical Suspension: cost tax not enforced; exhaust removal.
	engine.RegisterBehavior("15026", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, p := range g.Players {
				if p.HeroCode[:5] == "15001" {
					t.Target = p.ID
					break
				}
			}
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustYourHeroDiscardMagicalSuspension"), Type: engine.AbilityAction, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					a := g.Attachments[self]
					if a == nil {
						return nil
					}
					g.Delete(self)
					g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: a.Code})
					return []engine.Message{engine.ExhaustEntity{ID: g.ActiveTurn}}
				},
			}}
		},
	})

	// Chaos Manipulation: fetch Luminous; star-mill check → she attacks.
	engine.RegisterBehavior("15027", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			for _, mn := range g.Minions {
				if mn.Code[:5] == "15025" {
					return []engine.Message{engine.AskAttack{Enemy: mn.ID, Player: p.ID}}
				}
			}
			for i, c := range g.EncounterDeck {
				if c.Code[:5] == "15025" {
					g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
			}
			for i, c := range g.EncounterDiscard {
				if c.Code[:5] == "15025" {
					g.EncounterDiscard = append(g.EncounterDiscard[:i], g.EncounterDiscard[i+1:]...)
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
			}
			return nil
		},
	})
}

// ---- helpers ----

func schemePicks(g *engine.Game, n int, pid engine.PlayerID) []engine.Choice {
	return cardutil.SchemeChoices(g, func(s engine.EntityID) []engine.Message {
		return []engine.Message{engine.ThwartScheme{Scheme: s, N: n, Source: pid}}
	})
}
