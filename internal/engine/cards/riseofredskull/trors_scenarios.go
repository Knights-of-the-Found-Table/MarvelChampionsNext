package riseofredskull

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// absorbingTraits maps each environment to the trait Absorbing Man gains.
func absorbingHasTrait(g *engine.Game, trait string) bool {
	for _, env := range g.Environments {
		if env != nil && env.EDef().HasTrait(trait) {
			return true
		}
	}
	for _, s := range g.SideSchemes {
		if s != nil && s.Code[:5] == "04092" {
			return true // Super Absorbing Power grants them all
		}
	}
	return false
}

// registerAbsorbingMan installs the Absorbing Man scenario (04076–
// 04092) on the delay-counter economy.
func registerAbsorbingMan() {
	for _, base := range []string{"04076", "04077", "04078"} {
		b := &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				w, ok := msg.(engine.WindowAfterEnemyAttacked)
				if !ok || w.Enemy != e.EID() || base != "04078" {
					return nil
				}
				if absorbingHasTrait(g, "ice") || absorbingHasTrait(g, "stone") {
					if g.MainScheme != nil {
						g.Push(engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: e.EID()})
					}
				}
				if absorbingHasTrait(g, "metal") || absorbingHasTrait(g, "wood") {
					return []engine.Message{engine.IndirectDamage{Player: w.Player, N: 1}}
				}
				return nil
			},
		}
		if base == "04077" {
			b.VillainStage = func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
				for _, s := range g.SideSchemes {
					if s != nil && s.Code[:5] == "04092" {
						var msgs []engine.Message
						for _, p := range g.Players {
							msgs = append(msgs, engine.DealEncounterToPlayer{Player: p.ID})
						}
						return msgs
					}
				}
				for i, c := range g.EncounterDeck {
					if c.Code[:5] == "04092" {
						g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
						g.ShuffleEncounterDeck()
						return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
					}
				}
				return nil
			}
		}
		engine.RegisterBehavior(base, b)
	}

	// 04079 None Shall Pass: delay counters each round; environments
	// replace each other.
	engine.RegisterBehavior("04079", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			// Setup: dig an environment into play.
			for len(g.EncounterDeck) > 0 {
				c := g.EncounterDeck[0]
				g.EncounterDeck = g.EncounterDeck[1:]
				if c.Def().Type == "environment" {
					g.SpawnEnvironment(c.Code)
					break
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			switch m := msg.(type) {
			case engine.BeginPhase:
				if m.Phase != engine.PhaseVillain || g.MainScheme == nil {
					return nil
				}
				g.MainScheme.Counters++
				g.Logf("A delay counter accumulates (%d)", g.MainScheme.Counters)
			case engine.RevealEncounterCard:
				if m.Card.Def().Type != "environment" {
					return nil
				}
				// Discard each other environment.
				for id, env := range g.Environments {
					if env != nil && env.Code != m.Card.Code {
						g.Delete(id)
						g.Logf("%s washes away", env.EDef().Name)
					}
				}
			}
			return nil
		},
	})

	// 04080–04083 environments: undefended-attack riders.
	for _, code := range []string{"04080", "04081", "04082", "04083"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	// 04084 Ball and Chain.
	engine.RegisterBehavior("04084", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil && v.Code[:5] == "04076" {
					t.Target = id
					return nil
				}
			}
			for id := range g.Villains {
				t.Target = id
				break
			}
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Spend 1 [physical] → shuffle Ball and Chain into the encounter deck", Type: engine.AbilityAction,
				CostIcons: "physical:1",
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					a := g.Attachments[self]
					code := "04084"
					if a != nil {
						code = a.Code
					}
					g.Delete(self)
					g.EncounterDeck = append(g.EncounterDeck, engine.Card{ID: g.NextCardID(), Code: code})
					g.ShuffleEncounterDeck()
					return nil
				},
			}}
		},
	})

	// 04085 Stall Tactics.
	engine.RegisterBehavior("04085", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			if g.MainScheme == nil {
				return nil
			}
			n := g.MainScheme.Counters / 2
			if n == 0 {
				if c, ok := g.DrawEncounter(); ok {
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
				return nil
			}
			return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: n, Source: t.ID}}
		},
	})

	// 04086–04090 trait riders.
	traitRider := func(stone, ice, metal, wood bool) *engine.Behavior {
		return &engine.Behavior{}
	}
	_ = traitRider
	engine.RegisterBehavior("04086", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for id := range g.Villains {
				n := 0
				if absorbingHasTrait(g, "stone") {
					n = 1
				}
				return []engine.Message{
					engine.BoostActivation{Enemy: id, N: n},
					engine.VillainActivates{VillainID: id, Player: p.ID},
				}
			}
			return nil
		},
	})
	engine.RegisterBehavior("04087", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			if !p.IsHero() {
				n := 2
				if absorbingHasTrait(g, "metal") {
					n = 3
				}
				if g.MainScheme != nil {
					return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: n, Source: t.ID}}
				}
				return nil
			}
			n := 3
			if absorbingHasTrait(g, "metal") {
				n = 4
			}
			return []engine.Message{engine.IndirectDamage{Player: p.ID, N: n}}
		},
	})
	engine.RegisterBehavior("04088", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var msgs []engine.Message
			if len(p.Hand) > 0 {
				msgs = append(msgs, engine.DiscardCards{Player: p.ID, Cards: engine.CardList{p.Hand[0]}})
			}
			if absorbingHasTrait(g, "wood") && len(p.Upgrades) > 0 {
				msgs = append(msgs, engine.DiscardControlled{Player: p.ID, ID: p.Upgrades[len(p.Upgrades)-1]})
			}
			return msgs
		},
	})
	engine.RegisterBehavior("04089", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			switch {
			case absorbingHasTrait(g, "ice"):
				return []engine.Message{engine.ExhaustEntity{ID: p.ID}}
			case absorbingHasTrait(g, "metal"):
				for id, v := range g.Villains {
					if v != nil {
						v.Tough = true
						return []engine.Message{engine.HealEntity{Target: id, N: 1}}
					}
				}
			case absorbingHasTrait(g, "stone"):
				for id := range g.Villains {
					return []engine.Message{engine.DealBoost{Enemy: id}}
				}
			case absorbingHasTrait(g, "wood"):
				if len(p.Hand) > 0 {
					return []engine.Message{engine.DiscardCards{Player: p.ID, Cards: engine.CardList{p.Hand[0]}}}
				}
			}
			return nil
		},
	})
	engine.RegisterBehavior("04090", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			if absorbingHasTrait(g, "ice") {
				return []engine.Message{engine.StunEntity{Target: p.ID}, engine.IndirectDamage{Player: p.ID, N: 2}}
			}
			return []engine.Message{engine.StunEntity{Target: p.ID}}
		},
	})

	// 04091/04092 schemes.
	engine.RegisterBehavior("04091", &engine.Behavior{})
	engine.RegisterBehavior("04092", &engine.Behavior{})
}

// registerTaskmaster installs the Taskmaster scenario (04093–04108).
func registerTaskmaster() {
	for _, base := range []string{"04093", "04094", "04095"} {
		b := &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				cf, ok := msg.(engine.ChangeForm)
				if !ok {
					return nil
				}
				p := g.Player(cf.Player)
				if p == nil || !p.IsHero() {
					return nil
				}
				stars := 0
				if len(g.EncounterDeck) > 0 {
					c := g.EncounterDeck[0]
					g.EncounterDeck = g.EncounterDeck[1:]
					if bs := c.Def().Boost; bs != nil {
						stars = *bs
					}
					g.EncounterDiscard = append(g.EncounterDiscard, c)
				}
				if stars > 0 {
					return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: stars, Source: e.EID()}}
				}
				return nil
			},
		}
		if base != "04093" {
			b.VillainStage = func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
				var msgs []engine.Message
				for _, p := range g.Players {
					msgs = append(msgs, engine.DealEncounterToPlayer{Player: p.ID})
				}
				return msgs
			}
		}
		engine.RegisterBehavior(base, b)
	}

	// 04096 Hunting Down Heroes: hero-form tax.
	engine.RegisterBehavior("04096", &engine.Behavior{})

	// 04097–04100 hireling allies with resource riders.
	engine.RegisterBehavior("04097", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.DrawCards{Player: e.EOwner(), N: 2}}
		},
	})
	engine.RegisterBehavior("04098", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return cardutil.ChooseEnemy("Shang-Chi: stun which enemy?",
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) {
					return 0, []engine.Message{engine.StunEntity{Target: tgt.EID()}}
				})(g, e)
		},
	})
	engine.RegisterBehavior("04099", &engine.Behavior{
		OnPlay: cardutil.ChooseScheme("White Tiger", func(g *engine.Game, s engine.Entity) int { return 3 }),
	})
	engine.RegisterBehavior("04100", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy("Elektra: deal 3 damage",
			func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 3, nil }),
	})

	// 04101–04104.
	engine.RegisterBehavior("04101", &engine.Behavior{})
	engine.RegisterBehavior("04102", &engine.Behavior{})
	engine.RegisterBehavior("04103", &engine.Behavior{})
	engine.RegisterBehavior("04104", &engine.Behavior{})

	// 04105 Mimicry.
	engine.RegisterBehavior("04105", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var msgs []engine.Message
			found := false
			for i := 0; i < 5 && len(p.Deck) > 0; i++ {
				c := p.Deck[0]
				p.Deck = p.Deck[1:]
				if !p.IsHero() && c.Def().HasTrait("thwart") {
					found = true
				}
				if p.IsHero() && c.Def().HasTrait("attack") {
					found = true
				}
				p.Discard = append(p.Discard, c)
			}
			if found {
				for id := range g.Villains {
					msgs = append(msgs, engine.VillainActivates{VillainID: id, Player: p.ID})
					break
				}
			}
			return msgs
		},
	})

	// 04106 Hunted by Hydra.
	engine.RegisterBehavior("04106", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var msgs []engine.Message
			if g.MainScheme != nil {
				msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: t.ID})
			}
			for _, tp := range g.Players {
				if tp.IsHero() {
					msgs = append(msgs, engine.DamageEntity{Target: tp.ID, Damage: 1, Source: t.ID})
					if len(tp.Hand) > 0 {
						msgs = append(msgs, engine.DiscardCards{Player: tp.ID, Cards: engine.CardList{tp.Hand[0]}})
					}
				}
			}
			return msgs
		},
	})

	// 04107 Captured by Hydra: stash a hero-specific ally.
	engine.RegisterBehavior("04107", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			s := g.SideSchemes[e.EID()]
			if s == nil {
				return nil
			}
			for _, p := range g.Players {
				for i, c := range p.Deck {
					if c.Def().Type == "ally" {
						p.Deck = append(p.Deck[:i], p.Deck[i+1:]...)
						s.StoredCards = append(s.StoredCards, c)
						break
					}
				}
			}
			return nil
		},
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			for _, c := range s.StoredCards {
				if p := g.Player(c.Owner); p != nil {
					p.Hand = append(p.Hand, c)
				}
			}
			return nil
		},
	})

	// 04108 Training Camp: minions arrive tough.
	engine.RegisterBehavior("04108", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if m, ok := msg.(engine.MinionEntersPlay); ok {
				if mn := g.Minions[m.MinionID]; mn != nil {
					mn.Tough = true
				}
			}
			return nil
		},
	})
}

// registerZola installs the Zola scenario (04109–04124).
func registerZola() {
	for _, base := range []string{"04109", "04110", "04111"} {
		b := &engine.Behavior{}
		if base == "04110" {
			b.VillainStage = func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
				for i, c := range g.EncounterDeck {
					if c.Code[:5] == "04123" {
						g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
						g.ShuffleEncounterDeck()
						return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
					}
				}
				return nil
			}
		}
		if base == "04111" {
			b.VillainStage = func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
				var msgs []engine.Message
				for _, p := range g.Players {
					for i, c := range g.EncounterDeck {
						if c.Def().Type == "minion" {
							g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
							msgs = append(msgs, engine.RevealEncounterCard{Player: p.ID, Card: c})
							break
						}
					}
				}
				g.ShuffleEncounterDeck()
				return msgs
			}
		}
		engine.RegisterBehavior(base, b)
	}

	// 04112/04113: test counters spawn minions.
	testTimer := func() *engine.Behavior {
		return &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				bp, ok := msg.(engine.BeginPhase)
				if !ok || bp.Phase != engine.PhaseVillain || g.MainScheme == nil {
					return nil
				}
				g.MainScheme.Counters++
				if g.MainScheme.Counters >= 3 {
					g.MainScheme.Counters -= 3
					for len(g.EncounterDeck) > 0 {
						c := g.EncounterDeck[0]
						g.EncounterDeck = g.EncounterDeck[1:]
						if c.Def().Type == "minion" {
							return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
						}
						g.EncounterDiscard = append(g.EncounterDiscard, c)
					}
				}
				return nil
			},
		}
	}
	engine.RegisterBehavior("04112", testTimer())
	engine.RegisterBehavior("04113", testTimer())

	// 04114–04119.
	engine.RegisterBehavior("04114", &engine.Behavior{})
	engine.RegisterBehavior("04115", &engine.Behavior{})
	engine.RegisterBehavior("04116", &engine.Behavior{
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if g.MainScheme != nil {
				g.MainScheme.Counters++
			}
			return nil
		},
	})
	for _, code := range []string{"04117", "04118", "04119"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
				best, bestHP := engine.EntityID(""), -1
				for _, id := range sortedMinionIDs(g) {
					if m := g.Minions[id]; m != nil && m.HP() > bestHP {
						best, bestHP = id, m.HP()
					}
				}
				if best == "" {
					return []engine.Message{engine.DiscardAttachmentMsg{ID: t.ID}}
				}
				t.Target = best
				if m := g.Minions[best]; m != nil {
					m.MaxHP += 2
					switch code {
					case "04117":
						m.Guard = true
					case "04118":
						// retaliate 1 aura approximated as nothing extra here
					}
				}
				return nil
			},
		})
	}

	// 04120–04124.
	engine.RegisterBehavior("04120", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for id := range g.Villains {
				if !p.IsHero() {
					return []engine.Message{
						engine.VillainActivates{VillainID: id, Player: p.ID},
						engine.ConfuseEntity{Target: p.ID},
					}
				}
				return []engine.Message{
					engine.VillainActivates{VillainID: id, Player: p.ID},
					engine.StunEntity{Target: p.ID},
				}
			}
			return nil
		},
	})
	engine.RegisterBehavior("04121", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			if g.MainScheme != nil {
				g.MainScheme.Counters++
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if g.MainScheme != nil {
				g.MainScheme.Counters++
			}
			return nil
		},
	})
	engine.RegisterBehavior("04122", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			s := g.SideSchemes[e.EID()]
			if s == nil {
				return nil
			}
			for _, p := range g.Players {
				for i, c := range p.Deck {
					if c.Def().Type == "ally" {
						p.Deck = append(p.Deck[:i], p.Deck[i+1:]...)
						s.StoredCards = append(s.StoredCards, c)
						s.Threat += cardutil.Cost(c.Def())
						break
					}
				}
			}
			return nil
		},
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			for _, c := range s.StoredCards {
				if p := g.Player(c.Owner); p != nil {
					p.Hand = append(p.Hand, c)
				}
			}
			return nil
		},
	})
	engine.RegisterBehavior("04123", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			for len(g.EncounterDeck) > 0 {
				c := g.EncounterDeck[0]
				g.EncounterDeck = g.EncounterDeck[1:]
				if c.Def().Type == "minion" {
					return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			return nil
		},
	})
	engine.RegisterBehavior("04124", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionEntersPlay)
			if !ok {
				return nil
			}
			for i, c := range g.EncounterDiscard {
				if c.Def().Type == "attachment" && c.Def().HasTrait("tech") {
					g.EncounterDiscard = append(g.EncounterDiscard[:i], g.EncounterDiscard[i+1:]...)
					g.SpawnAttachment(c.Code, m.MinionID)
					break
				}
			}
			return nil
		},
	})
}

// sortedMinionIDs lists minion ids in stable order.
func sortedMinionIDs(g *engine.Game) []engine.EntityID {
	var out []engine.EntityID
	for id := range g.Minions {
		out = append(out, id)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
