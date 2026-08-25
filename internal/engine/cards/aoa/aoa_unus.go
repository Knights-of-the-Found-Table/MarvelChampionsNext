package aoa

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// genePool finds the Gene Pool side scheme.
func genePool(g *engine.Game) *engine.SideScheme {
	for _, s := range g.SideSchemes {
		if s != nil && s.Code == "45071" {
			return s
		}
	}
	return nil
}

// geneThreat adds threat to Gene Pool when it exists.
func geneThreat(g *engine.Game, n int) []engine.Message {
	if s := genePool(g); s != nil {
		return []engine.Message{engine.ApplySchemeThreat{Scheme: s.ID, N: n, Source: s.ID}}
	}
	return nil
}

// geneTier returns the Gene Pool tier: 3/6/9 threat thresholds.
func geneTier(g *engine.Game) int {
	s := genePool(g)
	if s == nil {
		return 0
	}
	switch {
	case s.Threat >= 9:
		return 3
	case s.Threat >= 6:
		return 2
	case s.Threat >= 3:
		return 1
	}
	return 0
}

// unus finds the Unus villain.
func unus(g *engine.Game) *engine.Villain {
	for _, v := range g.Villains {
		if v != nil && engine.BaseCodeOf(v.Code) == "45059" {
			return v
		}
	}
	return nil
}

// pursuedEnv finds the Pursued by the Past environment.
func pursuedEnv(g *engine.Game) *engine.Environment {
	for _, e := range g.Environments {
		if e != nil && engine.BaseCodeOf(e.Code) == "45075" {
			return e
		}
	}
	return nil
}

// addPursuit places pursuit counters and triggers the payoff.
func addPursuit(g *engine.Game, n int) []engine.Message {
	env := pursuedEnv(g)
	if env == nil {
		return nil
	}
	env.Counters += n
	g.TLogf("c.pursuedByThePastGainsPursuitCounterS", n, env.Counters)
	if env.Counters >= len(g.Players)+3 {
		env.Counters = 0
		g.TLogf("c.yourPastCatchesUpWithYou")
		var msgs []engine.Message
		for _, p := range g.Players {
			for _, id := range p.NemesisInPlay {
				if mn := g.Minions[id]; mn != nil {
					msgs = append(msgs, engine.MinionActivates{MinionID: id, Player: p.ID})
					break
				}
			}
		}
		return msgs
	}
	return nil
}

func registerUnus() {
	// 45059-45061 Unus: scales with the Gene Pool tier (retaliate at 3;
	// stalwart/amplify riders are not modeled).
	unusBehavior := &engine.Behavior{}
	engine.RegisterBehavior("45059", unusBehavior)
	engine.RegisterBehavior("45060", unusBehavior)
	engine.RegisterBehavior("45061", unusBehavior)

	// 45062 Hunting Gene Traitors: step one grows the Gene Pool.
	engine.RegisterBehavior("45062", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.BeginPhase)
			if !ok || m.Phase != engine.PhaseVillain || g.MainScheme == nil {
				return nil
			}
			if g.MainScheme.EID() != e.EID() {
				return nil
			}
			return geneThreat(g, 1)
		},
	})

	// 45063 Prelate Sidearm: Unus's kills feed the Gene Pool.
	engine.RegisterBehavior("45063", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if v := unus(g); v != nil {
				t.Target = v.ID
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.AllyDestroyed); !ok {
				return nil
			}
			t := g.Attachments[e.EID()]
			v := unus(g)
			if t == nil || v == nil || t.Target != v.ID {
				return nil
			}
			// The source of the defeat is approximate: any ally death
			// while the sidearm is attached.
			return geneThreat(g, 1)
		},
	})

	// 45064 Prelate Armor: Unus toughs up after scheming.
	engine.RegisterBehavior("45064", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if v := unus(g); v != nil {
				t.Target = v.ID
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.ApplyVillainScheme); !ok {
				return nil
			}
			t := g.Attachments[e.EID()]
			v := unus(g)
			if t == nil || v == nil || t.Target != v.ID {
				return nil
			}
			return []engine.Message{engine.ToughEntity{Target: v.ID}}
		},
	})

	// 45065 Infinite Hunter: strike an ally; boost feeds the pool.
	engine.RegisterBehavior("45065", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return nil
			}
			if p := g.Player(mn.EngagedWith); p != nil {
				for _, id := range p.Allies {
					return []engine.Message{engine.DamageEntity{Target: id, Damage: 3, Source: mn.ID}}
				}
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return geneThreat(g, 2)
		},
	})

	// 45066 Genetic Experiments: +2 HP on an Infinite; death feeds the
	// pool.
	engine.RegisterBehavior("45066", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, mn := range g.Minions {
				if mn != nil && mn.EDef().HasTrait("Infinite") {
					t.Target = mn.ID
					mn.MaxHP += 2
					g.TLogf("c.geneticExperimentsBuffs2Hp", mn)
					return nil
				}
			}
			// No Infinite: surge.
			if c, ok := g.DrawEncounter(); ok {
				return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionDefeated)
			if !ok {
				return nil
			}
			t := g.Attachments[e.EID()]
			if t == nil || t.Target != m.MinionID {
				return nil
			}
			_ = m
			return geneThreat(g, 2)
		},
	})

	// 45067 Infinite Prelate: Unus activates with tier riders.
	engine.RegisterBehavior("45067", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			v := unus(g)
			if v == nil {
				return nil
			}
			tier := geneTier(g)
			msgs := []engine.Message{}
			if tier >= 1 {
				msgs = append(msgs, engine.ToughEntity{Target: v.ID})
			}
			if tier >= 2 {
				msgs = append(msgs, engine.HealEntity{Target: v.ID, N: 3})
			}
			if tier >= 3 {
				msgs = append(msgs, engine.DealBoost{Enemy: v.ID})
			}
			msgs = append(msgs, engine.VillainActivates{VillainID: v.ID, Player: p.ID})
			return msgs
		},
	})

	// 45068 Endless Ranks: defeat feeds the pool.
	engine.RegisterBehavior("45068", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			return geneThreat(g, 3)
		},
	})

	// 45069 Infinite Soldier: Guard; +3 HP at tier 3.
	engine.RegisterBehavior("45069", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if geneTier(g) >= 3 {
				mn := g.Minions[e.EID()]
				if mn != nil {
					mn.MaxHP += 3
					g.TLogf("c.infiniteSoldierGains3HitPoints")
				}
			}
			return nil
		},
	})

	// 45070 Culling the Weak: pool growth; boost feeds it too.
	engine.RegisterBehavior("45070", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			return geneThreat(g, 4)
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return geneThreat(g, 2)
		},
	})

	// 45071 Gene Pool: ally deaths feed it (consequential approximation).
	engine.RegisterBehavior("45071", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.AllyDefeated); !ok {
				return nil
			}
			s := g.SideSchemes[e.EID()]
			if s == nil {
				return nil
			}
			s.Threat += 3
			g.TLogf("c.genePoolGains3Threat", s.Threat, s.MaxThreat)
			return nil
		},
	})
}

func registerDystopianNightmare() {
	// 45072 Hunted: discard a card to shed it.
	engine.RegisterBehavior("45072", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			if len(p.Hand) == 0 {
				return []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
			}
			var choices []engine.Choice
			for i, c := range p.Hand {
				hc := c
				choices = append(choices, engine.Choice{
					ID: "d-" + hc.ID, Label: engine.S("Discard " + hc.Def().Name), Kind: engine.ChoiceCard,
				}.Msgs(engine.DiscardCards{Player: p.ID, Cards: engine.CardList{hc}},
					engine.ObligationResolve{Player: p.ID, Card: card}))
				_ = i
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.huntedDiscardACardToRemoveThisObligation"), choices...),
			}}
		},
	})

	// 45073 War-Weary: stun or damage.
	engine.RegisterBehavior("45073", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			return warWearyHit(g, p)
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if p := g.Player(cardutil.FirstPlayerID(g)); p != nil {
				return warWearyHit(g, p)
			}
			return nil
		},
	})

	// 45074 Targeted for Extermination: defeater gets confused.
	engine.RegisterBehavior("45074", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			pid := cardutil.FirstPlayerID(g)
			return []engine.Message{engine.ConfuseEntity{Target: pid}}
		},
	})
}

func warWearyHit(g *engine.Game, p *engine.Player) []engine.Message {
	if p.Stunned {
		return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 2, Source: engine.EntityID("")}}
	}
	return []engine.Message{engine.StunEntity{Target: p.ID}}
}

func registerStandardIII() {
	// 45075a Pursued by the Past: counter payoffs live in addPursuit.
	engine.RegisterBehavior("45075", &engine.Behavior{})

	// 45076 Dark Designs: pursuit + villain schemes.
	engine.RegisterBehavior("45076", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			msgs := addPursuit(g, 1)
			if pursuedEnv(g) != nil {
				if id := firstVillainID(g); id != "" {
					msgs = append(msgs, engine.VillainActivates{VillainID: id, Player: p.ID})
				}
			}
			return msgs
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return addPursuit(g, 1)
		},
	})

	// 45077 Sinister Strike: pursuit + surge/attack.
	engine.RegisterBehavior("45077", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			msgs := addPursuit(g, 1)
			if pursuedEnv(g) == nil {
				return msgs
			}
			if !p.IsHero() {
				if c, ok := g.DrawEncounter(); ok {
					msgs = append(msgs, engine.RevealEncounterCard{Player: p.ID, Card: c})
				}
				return msgs
			}
			if id := firstVillainID(g); id != "" {
				msgs = append(msgs, engine.AskAttack{Enemy: id, Player: p.ID})
			}
			return msgs
		},
	})

	// 45078 Evil Alliance: nemeses activate.
	engine.RegisterBehavior("45078", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var msgs []engine.Message
			fired := false
			for _, o := range g.Players {
				for _, id := range o.NemesisInPlay {
					if mn := g.Minions[id]; mn != nil {
						msgs = append(msgs, engine.MinionActivates{MinionID: id, Player: p.ID})
						fired = true
					}
				}
			}
			if !fired {
				msgs = append(msgs, addPursuit(g, 3)...)
			}
			return msgs
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return addPursuit(g, 1)
		},
	})

	// 45079 Nowhere is Safe: pursuit + lose a card.
	engine.RegisterBehavior("45079", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			msgs := addPursuit(g, 1)
			if pursuedEnv(g) == nil {
				return msgs
			}
			if len(p.Supports) > 0 {
				msgs = append(msgs, engine.DiscardControlled{Player: p.ID, ID: p.Supports[0]})
			} else if len(p.Upgrades) > 0 {
				msgs = append(msgs, engine.DiscardControlled{Player: p.ID, ID: p.Upgrades[0]})
			}
			return msgs
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return addPursuit(g, 1)
		},
	})

	// 45080 Drawing Near: mill per turn; discard an identity card to
	// remove.
	engine.RegisterBehavior("45080", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			return []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
		},
	})
}

func firstVillainID(g *engine.Game) engine.EntityID {
	for id := range g.Villains {
		return id
	}
	return ""
}
