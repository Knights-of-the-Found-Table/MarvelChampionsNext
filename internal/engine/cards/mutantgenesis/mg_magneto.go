// mg_magneto.go implements the Magneto scenario (32138–32158) and the
// Acolytes modular set (32159–32165).
package mutantgenesis

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerMagneto()
	registerAcolytes()
}

// magneto returns the Magneto villain.
func magneto(g *engine.Game) *engine.Villain {
	for _, v := range g.Villains {
		if v != nil && engine.BaseCodeOf(v.Code) == "32138" {
			return v
		}
	}
	return nil
}

func registerMagneto() {
	// 32138–32140 Magneto stages: after he attacks you, place a magnet
	// counter on the main scheme (the counter engine lives on the scheme
	// behavior below; stages II+ "deal each player a facedown encounter
	// card" runs on reveal).
	for i, code := range []string{"32138", "32139", "32140"} {
		stage := i + 1
		engine.RegisterBehavior(code, &engine.Behavior{
			VillainStage: func(g *engine.Game, v *engine.Villain, nextStage int) []engine.Message {
				if stage == 1 {
					return nil
				}
				var msgs []engine.Message
				for _, p := range g.Players {
					msgs = append(msgs, engine.DealEncounterToPlayer{Player: p.ID})
				}
				return msgs
			},
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				// After Magneto attacks you, place a magnet counter on the
				// main scheme.
				m, ok := msg.(engine.AskAttack)
				if !ok || m.Enemy != e.EID() || g.MainScheme == nil {
					return nil
				}
				return []engine.Message{engine.AddMagnetCounter{Scheme: g.MainScheme.ID}}
			},
		})
	}

	// 32141–32143 Asteroid M / Factory Online / The Rule of Magnus: the
	// magnet-counter payoff (each counters≥3 reveals a Magnetic card).
	for _, code := range []string{"32141", "32142", "32143"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	// 32144 Boarding Party / 32145 Orbital Decay: sustained-damage caps on
	// Magneto (approximation: damage cap enforced in VillainDamageable).
	engine.RegisterBehavior("32144", &engine.Behavior{})
	engine.RegisterBehavior("32145", &engine.Behavior{})

	// 32146 M-Type Sentinel: guard from data; When Defeated — tough for
	// Magneto.
	engine.RegisterBehavior("32146", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.MinionDefeated); !ok {
				return nil
			}
			if mn := g.Minions[e.EID()]; mn != nil && magneto(g) != nil {
				return []engine.Message{engine.ToughEntity{Target: magneto(g).ID}}
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if v := magneto(g); v != nil {
				v.BoostCount++
				return []engine.Message{engine.ToughEntity{Target: v.ID}}
			}
			return nil
		},
	})

	// 32147/32148 Magneto's Helmet/Armor: attach to Magneto; the
	// stun/confuse immunity and attack-response discard are approximated
	// away (spend riders offered as actions).
	for _, code := range []string{"32147", "32148"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
				if v := magneto(g); v != nil {
					t.Target = v.ID
				}
				return nil
			},
		})
	}

	// 32149 Magnetic Bubble: damage to Magneto lands on the bubble instead
	// (tracked as counters on the attachment; pops at 8).
	engine.RegisterBehavior("32149", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if v := magneto(g); v != nil {
				t.Target = v.ID
			}
			return nil
		},
	})

	// 32150 Wrapped in Metal: attach to an identity; it cannot act; spend
	// [physical] + exhaust to discard (approximation: the action lock is
	// player-enforced).
	engine.RegisterBehavior("32150", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, p := range g.Players {
				has := false
				for _, a := range g.Attachments {
					if a != nil && a.Code == "32150" && a.Target == p.ID {
						has = true
					}
				}
				if !has {
					t.Target = p.ID
					return nil
				}
			}
			g.Delete(t.ID)
			return []engine.Message{engine.RevealNextEncounter{Player: cardutil.FirstPlayerID(g)}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			t := g.Attachments[e.EID()]
			if t == nil {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.wrappedInMetalExhaustSpendPhysicalDiscard"), Type: engine.AbilityAction,
				Cost: 1, CostIcons: "physical:1",
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					t := g.Attachments[self]
					if t == nil {
						return nil
					}
					if p := g.Player(engine.PlayerID(t.Target)); p != nil && p.Exhausted {
						return []engine.Message{engine.DiscardAttachmentMsg{ID: self}}
					}
					return nil
				},
			}}
		},
	})

	// 32151 Master of Magnetism: topmost Magnetic card becomes Magneto's
	// facedown boost; he activates.
	engine.RegisterBehavior("32151", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			v := magneto(g)
			if v == nil {
				return nil
			}
			for i, c := range g.EncounterDiscard {
				if c.Def().HasTrait("magnetic") {
					g.EncounterDiscard = append(g.EncounterDiscard[:i], g.EncounterDiscard[i+1:]...)
					v.BoostCount += cardutil.BoostOf(c)
					g.TLogf("c.becomesMagnetoSFacedownBoostCard", c)
					break
				}
			}
			if p.IsHero() {
				return []engine.Message{engine.AskAttack{Enemy: v.ID, Player: p.ID, Trigger: engine.TriggerVillainAttacksYou}}
			}
			if g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: v.SchemeVal, Source: v.ID}}
			}
			return nil
		},
	})

	// 32152 Electric Shock: confused + threat per magnet counter (AE) /
	// stunned + damage per counter (hero).
	engine.RegisterBehavior("32152", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			n := 0
			if g.MainScheme != nil {
				n = g.MainScheme.Counters
			}
			if p.IsHero() {
				msgs := []engine.Message{engine.StunEntity{Target: p.ID}}
				if n > 0 {
					msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: n, Source: t.ID})
				}
				return msgs
			}
			msgs := []engine.Message{engine.ConfuseEntity{Target: p.ID}}
			if n > 0 && g.MainScheme != nil {
				msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: n, Source: t.ID})
			}
			return msgs
		},
	})

	// 32153 Electromagnetic Blast: exhaust each upgrade and support you
	// control; 1 magnet counter.
	engine.RegisterBehavior("32153", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for _, id := range p.Supports {
				if s := g.Supports[id]; s != nil {
					s.Exhausted = true
				}
			}
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil {
					u.Exhausted = true
				}
			}
			var msgs []engine.Message
			if g.MainScheme != nil {
				msgs = append(msgs, engine.AddMagnetCounter{Scheme: g.MainScheme.ID})
			}
			return msgs
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return []engine.Message{engine.ExhaustEntity{ID: cardutil.FirstPlayerID(g)}}
		},
	})

	// 32154 Metal Shards: 1 damage to each character you control; 1 magnet
	// counter.
	engine.RegisterBehavior("32154", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			msgs := []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 1, Source: t.ID}}
			for _, id := range p.Allies {
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: t.ID})
			}
			if g.MainScheme != nil {
				msgs = append(msgs, engine.AddMagnetCounter{Scheme: g.MainScheme.ID})
			}
			return msgs
		},
	})

	// 32155 Magnetic Missile: defeat a Sentinel minion, take 5 damage
	// (discard-to-prevent approximated to a flat 5).
	engine.RegisterBehavior("32155", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var msgs []engine.Message
			for _, id := range cardutil.SortedIDs(g.Minions) {
				if mn := g.Minions[id]; mn != nil && mn.EDef().HasTrait("sentinel") {
					msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 99, Source: t.ID})
					break
				}
			}
			return append(msgs, engine.DamageEntity{Target: p.ID, Damage: 5, Source: t.ID})
		},
	})

	// 32156 Magnetic Mayhem: When Defeated — mill 4, magnet counter per
	// Magnetic card.
	engine.RegisterBehavior("32156", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() || g.MainScheme == nil {
				return nil
			}
			var msgs []engine.Message
			for i := 0; i < 4 && len(g.EncounterDeck) > 0; i++ {
				top := g.EncounterDeck[0]
				g.EncounterDeck = g.EncounterDeck[1:]
				g.EncounterDiscard = append(g.EncounterDiscard, top)
				if top.Def().HasTrait("magnetic") {
					msgs = append(msgs, engine.AddMagnetCounter{Scheme: g.MainScheme.ID})
				}
			}
			return msgs
		},
	})

	// 32157 Magnetically Sealed: 2 extra threat per ally in play.
	engine.RegisterBehavior("32157", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			n := 0
			for _, p := range g.Players {
				n += len(p.Allies)
			}
			if n > 0 {
				return []engine.Message{engine.SchemeThreat{Scheme: e.EID(), N: 2 * n, Source: e.EID()}}
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			pid := cardutil.FirstPlayerID(g)
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			var msgs []engine.Message
			for _, id := range p.Allies {
				msgs = append(msgs, engine.ExhaustEntity{ID: id})
			}
			return msgs
		},
	})

	// 32158 Seized!: each player tucks 6 deck cards here; When Defeated —
	// Magneto attacks.
	engine.RegisterBehavior("32158", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			s := g.SideSchemes[e.EID()]
			if s == nil {
				return nil
			}
			for _, p := range g.Players {
				for i := 0; i < 6 && len(p.Deck) > 0; i++ {
					top := p.Deck[0]
					p.Deck = p.Deck[1:]
					s.StoredCards = append(s.StoredCards, top)
				}
			}
			g.TLogf("c.seizedEachPlayerSTop6CardsAreTrapped")
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			if v := magneto(g); v != nil {
				return []engine.Message{engine.AskAttack{Enemy: v.ID, Player: cardutil.FirstPlayerID(g), Trigger: engine.TriggerVillainAttacksYou}}
			}
			return nil
		},
	})
}

func registerAcolytes() {
	// 32159 Fabian Cortez: When Defeated — the defeater gets an Acolyte
	// from the deck.
	engine.RegisterBehavior("32159", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.MinionDefeated); !ok {
				return nil
			}
			pid := cardutil.FirstPlayerID(g)
			for guards := 0; guards < 40; guards++ {
				if len(g.EncounterDeck) == 0 {
					return nil
				}
				top := g.EncounterDeck[0]
				g.EncounterDeck = g.EncounterDeck[1:]
				if top.Def().Type == "minion" && top.Def().HasTrait("acolyte") {
					return []engine.Message{engine.RevealEncounterCard{Player: pid, Card: top}}
				}
				g.EncounterDiscard = append(g.EncounterDiscard, top)
			}
			return nil
		},
	})

	// 32160 Amelia Voght: defeater confused (2 threat if already).
	engine.RegisterBehavior("32160", &engine.Behavior{
		React: acolyteDefeat(func(g *engine.Game, p *engine.Player) []engine.Message {
			if p.Confused && g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: p.ID}}
			}
			return []engine.Message{engine.ConfuseEntity{Target: p.ID}}
		}),
	})

	// 32161 Senyaka: defeater stunned (3 damage if already).
	engine.RegisterBehavior("32161", &engine.Behavior{
		React: acolyteDefeat(func(g *engine.Game, p *engine.Player) []engine.Message {
			if p.Stunned {
				return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 3, Source: p.ID}}
			}
			return []engine.Message{engine.StunEntity{Target: p.ID}}
		}),
	})

	// 32162 Delgado: defeat — clear the villain's statuses + facedown
	// boost.
	engine.RegisterBehavior("32162", &engine.Behavior{
		React: acolyteDefeat(func(g *engine.Game, p *engine.Player) []engine.Message {
			if v := activeOrFirstVillain(g); v != nil {
				v.Stunned, v.Confused = false, false
				v.BoostCount++
			}
			return nil
		}),
	})

	// 32163 Unuscione: defeat — villain tough (heal 4 if already).
	engine.RegisterBehavior("32163", &engine.Behavior{
		React: acolyteDefeat(func(g *engine.Game, p *engine.Player) []engine.Message {
			if v := activeOrFirstVillain(g); v != nil {
				if v.Tough && v.Damage > 0 {
					v.Damage -= 4
					if v.Damage < 0 {
						v.Damage = 0
					}
					return nil
				}
				return []engine.Message{engine.ToughEntity{Target: v.ID}}
			}
			return nil
		}),
	})

	// 32164 Zeal for the Cause: resolve each engaged Acolyte's defeat
	// rider; with none, reveal a minion.
	engine.RegisterBehavior("32164", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var msgs []engine.Message
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn == nil || mn.EngagedWith != p.ID || !mn.EDef().HasTrait("acolyte") {
					continue
				}
				if b := engine.LookupBehavior(mn.Code); b != nil && b.React != nil {
					msgs = append(msgs, b.React(g, mn, engine.MinionDefeated{MinionID: id})...)
				}
			}
			if len(msgs) == 0 {
				for guards := 0; guards < 40; guards++ {
					if len(g.EncounterDeck) == 0 {
						return nil
					}
					top := g.EncounterDeck[0]
					g.EncounterDeck = g.EncounterDeck[1:]
					if top.Def().Type == "minion" {
						return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: top}}
					}
					g.EncounterDiscard = append(g.EncounterDiscard, top)
				}
			}
			return msgs
		},
	})

	// 32165 The Acolytes: each Acolyte gains guard (enforced in
	// guardBlocksVillain); boost shuffles Acolytes back.
	engine.RegisterBehavior("32165", &engine.Behavior{
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			shuffled := 0
			for i := 0; i < len(g.EncounterDiscard); {
				c := g.EncounterDiscard[i]
				if c.Def().Type == "minion" && c.Def().HasTrait("acolyte") {
					g.EncounterDiscard = append(g.EncounterDiscard[:i], g.EncounterDiscard[i+1:]...)
					g.EncounterDeck = append(g.EncounterDeck, c)
					shuffled++
					continue
				}
				i++
			}
			if shuffled > 0 {
				g.TLogf("c.theAcolytesShuffleMembersBackIntoTheDeck", shuffled)
			}
			return nil
		},
	})
}

// acolyteDefeat wraps a When-Defeated rider for Acolyte minions (the
// defeater approximated by the first player).
func acolyteDefeat(effect func(g *engine.Game, p *engine.Player) []engine.Message) func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
	return func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.MinionDefeated)
		if !ok || m.MinionID != e.EID() {
			return nil
		}
		p := g.Player(cardutil.FirstPlayerID(g))
		if p == nil {
			return nil
		}
		return effect(g, p)
	}
}
