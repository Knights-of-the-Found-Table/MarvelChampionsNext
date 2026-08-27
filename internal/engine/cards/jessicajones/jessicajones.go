// Package jessicajones registers Jessica Jones (61001), her Alias
// Investigations evidence engine, signature cards, obligation and nemesis.
package jessicajones

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

const evidenceCode = "61002"

func init() {
	registerIdentity()
	registerSignatures()
	registerGenericPlayerCards()
	registerNemesis()
	registerObligation()
}

func alias(g *engine.Game, p *engine.Player) *engine.Support {
	if p == nil {
		return nil
	}
	for _, id := range p.Supports {
		if s := g.Supports[id]; s != nil && s.Code == evidenceCode {
			return s
		}
	}
	return nil
}
func addEvidence(g *engine.Game, p *engine.Player, n int) []engine.Message {
	if s := alias(g, p); s != nil && n != 0 {
		return []engine.Message{engine.AddEntityCounter{ID: s.ID, N: n}}
	}
	return nil
}
func evidenceChoices(g *engine.Game, p *engine.Player, n int, prompt string, extra func() []engine.Message) []engine.Message {
	s := alias(g, p)
	if s == nil || s.Counters < n || s.Exhausted {
		return nil
	}
	choices := []engine.Choice{engine.Choice{ID: "keep", Label: engine.S("Do not spend evidence"), Kind: engine.ChoicePass}}
	choices = append(choices, engine.Choice{ID: "spend", Label: engine.S(fmt.Sprintf("Spend %d evidence", n)), Kind: engine.ChoiceLabel}.Msgs(append([]engine.Message{engine.ExhaustEntity{ID: s.ID}, engine.AddEntityCounter{ID: s.ID, N: -n}}, extra()...)...))
	return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.S(prompt), choices...)}}
}

func registerIdentity() {
	engine.RegisterBehavior("61001", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{} },
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			p := g.Player(e.EID())
			if p == nil {
				return nil
			}
			switch msg.(type) {
			case engine.BasicAttack, engine.BasicThwart, engine.BasicRecover:
				// The engine emits these only for a legal basic-power use;
				// retaining all three keeps Gather Evidence form-neutral.
				return addEvidence(g, p, 1)
			}
			return nil
		},
	})
	// Alias Investigations is permanent. The optional villain-stage defeat is
	// represented by damage equal to current HP; the engine's damage pipeline
	// then emits VillainDefeated and advances the current stage.
	engine.RegisterBehavior(evidenceCode, &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		s := g.Supports[e.EID()]
		if s == nil {
			return nil
		}
		m, ok := msg.(engine.SchemeDefeated)
		if !ok || s.Exhausted {
			return nil
		}
		if _, exists := g.SideSchemes[m.Scheme]; !exists {
			return nil
		}
		p := g.Player(s.Owner)
		if p == nil {
			return nil
		}
		s.Counters += 2
		v := g.Villains[g.ActiveVillain]
		if v == nil || v.HP() <= 0 || s.Counters < v.HP() {
			return []engine.Message{engine.ExhaustEntity{ID: s.ID}}
		}
		return evidenceChoices(g, p, v.HP(), "Alias Investigations — spend evidence to defeat the villain stage?", func() []engine.Message {
			return []engine.Message{engine.DamageEntity{Target: v.ID, Damage: v.HP(), Source: p.ID}}
		})
	}})
}

func registerSignatures() {
	// Luke Cage: Toughness plus exhaust + 1 damage to replace his tough card.
	engine.RegisterBehavior("61003", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			a := g.Allies[e.EID()]
			if a == nil || a.Tough {
				return nil
			}
			return []engine.Ability{{
				Label: engine.S("Luke Cage — take 1 damage to gain tough"), Type: engine.AbilityAction, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.DamageEntity{Target: self, Damage: 1, Source: self}, engine.ToughEntity{Target: self}}
				},
			}}
		},
	})
	engine.RegisterBehavior("61004", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		pid := e.EOwner()
		choices := cardutil.EnemyChoices(g, 3, pid, func(id engine.EntityID) []engine.Message {
			base := []engine.Message{engine.DamageEntity{Target: id, Damage: 3, Source: pid}}
			extra := evidenceChoices(g, g.Player(pid), 1, "Big Mistake — spend evidence for +2 damage?", func() []engine.Message {
				return []engine.Message{engine.DamageEntity{Target: id, Damage: 2, Source: pid}}
			})
			return append(base, extra...)
		})
		if len(choices) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: pid, Question: engine.Ask(engine.S("Big Mistake — choose an enemy"), choices...)}}
	}})
	engine.RegisterBehavior("61005", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		_, ok := msg.(engine.AddEntityCounter)
		if !ok {
			return nil
		}
		s := alias(g, g.Player(e.EOwner()))
		if s == nil {
			return nil
		}
		// The engine has no event-card shuffle message; readiness is exact,
		// while shuffling Breakthrough itself is an explicit approximation.
		return []engine.Message{engine.ReadyEntity{ID: e.EOwner()}}
	}})
	engine.RegisterBehavior("61006", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		pid := e.EOwner()
		choices := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
			return []engine.Message{engine.ThwartScheme{Scheme: id, N: 4, Source: pid}}
		})
		if len(choices) == 0 {
			return nil
		}
		msgs := []engine.Message{engine.AskQuestion{Player: pid, Question: engine.Ask(engine.S("Snooping Around — remove 4 threat"), choices...)}}
		if len(g.EncounterDeck) > 0 {
			c := g.EncounterDeck[0]
			msgs = append(msgs, engine.DiscardEncounterCard{Card: c})
			if c.Def().Type == "minion" || c.Def().Type == "side_scheme" {
				msgs = append(msgs, engine.RevealEncounterCard{Player: pid, Card: c})
			}
		}
		return msgs
	}})
	engine.RegisterBehavior("61007", &engine.Behavior{SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
		return []engine.Message{engine.DrawCards{Player: s.Owner, N: 3}}
	}})
	engine.RegisterBehavior("61008", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			p := g.Player(e.EOwner())
			a := alias(g, p)
			if s == nil || p == nil || p.IsHero() || s.Exhausted || a == nil || a.Counters < 2 {
				return nil
			}
			return []engine.Ability{{Label: engine.S("Calling in Favors — spend 2 evidence for an ally"), Type: engine.AbilityAction, AlterEgoOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					for i, c := range p.Deck {
						if c.Def().Type == "ally" {
							p.Deck = append(p.Deck[:i], p.Deck[i+1:]...)
							return []engine.Message{engine.AddEntityCounter{ID: a.ID, N: -2}, engine.AllyEntersPlayFree{Player: p.ID, Card: c, FromOwner: p.ID}}
						}
					}
					return nil
				},
			}}
		},
	})
	engine.RegisterBehavior("61009", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			u := g.Upgrades[e.EID()]
			p := g.Player(e.EOwner())
			a := alias(g, p)
			if u == nil || p == nil || p.IsHero() || u.Exhausted || a == nil {
				return nil
			}
			var cs []engine.Choice
			for id, m := range g.Minions {
				if m != nil && !m.EDef().HasKeyword("Elite") {
					n := m.HP() - 2
					if n < 0 {
						n = 0
					}
					cs = append(cs, engine.Choice{Label: engine.Tf("m.cardName", m), Kind: engine.ChoiceTarget, SourceID: id}.Msgs(engine.ExhaustEntity{ID: u.ID}, engine.AddEntityCounter{ID: a.ID, N: -n}, engine.DiscardControlled{Player: p.ID, ID: id}))
				}
			}
			if len(cs) == 0 {
				return nil
			}
			return []engine.Ability{{Label: engine.S("4K Digital Camcorder — discard a minion"), Type: engine.AbilityAction, AlterEgoOnly: true, Exhaust: true, Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
				return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.S("4K Digital Camcorder — choose a minion"), cs...)}}
			}}}
		},
	})
	engine.RegisterBehavior("61010", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		if _, ok := msg.(engine.MinionDefeated); ok {
			if a := alias(g, g.Player(e.EOwner())); a != nil {
				return []engine.Message{engine.ExhaustEntity{ID: e.EID()}, engine.AddEntityCounter{ID: a.ID, N: 1}}
			}
		}
		if _, ok := msg.(engine.VillainDefeated); ok {
			if a := alias(g, g.Player(e.EOwner())); a != nil {
				return []engine.Message{engine.ExhaustEntity{ID: e.EID()}, engine.AddEntityCounter{ID: a.ID, N: 1}}
			}
		}
		return nil
	}})
	engine.RegisterBehavior("61011", &engine.Behavior{IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{DEF: 2} }, React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		if _, ok := msg.(engine.WindowDefended); ok {
			return addEvidence(g, g.Player(e.EOwner()), 1)
		}
		return nil
	}})
	engine.RegisterBehavior("61012", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.VillainActivates)
		p := g.Player(e.EOwner())
		if !ok || p == nil || m.Player != p.ID || p.IsHero() {
			return nil
		}
		var out []engine.Message
		out = append(out, engine.ChangeForm{Player: p.ID}, engine.DamageEntity{Target: m.VillainID, Damage: 5, Source: p.ID})
		for id, mn := range g.Minions {
			if mn.EngagedWith == p.ID {
				out = append(out, engine.DamageEntity{Target: id, Damage: 5, Source: p.ID})
			}
		}
		out = append(out, engine.DiscardControlled{Player: p.ID, ID: e.EID()})
		return out
	}})
	engine.RegisterBehavior("61013", &engine.Behavior{DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
		if !p.IsHero() {
			return engine.Defends{}, nil, false
		}
		return engine.Defends{Defender: p.ID, Against: against, PreventAll: true}, []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 1, Source: p.ID}}, true
	}})
	engine.RegisterBehavior("61014", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		u := g.Upgrades[e.EID()]
		if u == nil {
			return nil
		}
		if m, ok := msg.(engine.SchemeDefeated); ok && m.Scheme == u.AttachTo {
			return addEvidence(g, g.Player(u.Owner), 2)
		}
		return nil
	}})
	engine.RegisterBehavior("61015", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		if _, ok := msg.(engine.AllyEnteredPlay); ok {
			return []engine.Message{engine.DiscardCards{Player: e.EOwner(), Cards: g.Player(e.EOwner()).Hand[:min(4, len(g.Player(e.EOwner()).Hand))]}}
		}
		return nil
	}})
	engine.RegisterBehavior("61016", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		a := g.Allies[e.EID()]
		if a == nil {
			return nil
		}
		switch m := msg.(type) {
		case engine.AllyThwartWindow:
			if m.Ally == a.ID {
				a.Counters++
				if a.Counters > 8 {
					a.Counters = 8
				}
			}
		case engine.AllyDefeated:
			if m.AllyID == a.ID {
				return []engine.Message{engine.DamageEntity{Target: g.ActiveVillain, Damage: a.Counters, Source: e.EID()}}
			}
		}
		return nil
	}})
	engine.RegisterBehavior("61017", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			a := g.Allies[e.EID()]
			p := g.Player(e.EOwner())
			if a != nil && p != nil {
				a.Counters = len(p.Hand)
				if a.Counters > 4 {
					a.Counters = 4
				}
			}
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			a := g.Allies[e.EID()]
			if a == nil || a.Counters < 1 {
				return nil
			}
			return []engine.Ability{{Label: engine.S("Squirrel Girl — remove 1 threat"), Type: engine.AbilityAction, Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
				cs := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
					return []engine.Message{engine.AddEntityCounter{ID: self, N: -1}, engine.ThwartScheme{Scheme: id, N: 1, Source: self}}
				})
				if len(cs) == 0 {
					return nil
				}
				return []engine.Message{engine.AskQuestion{Player: a.Owner, Question: engine.Ask(engine.S("Squirrel Girl — choose a scheme"), cs...)}}
			}}}
		},
	})
	engine.RegisterBehavior("61018", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		if _, ok := msg.(engine.ChangeForm); ok {
			return []engine.Message{engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 3, Source: e.EOwner()}}
		}
		return nil
	}})
	engine.RegisterBehavior("61019", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		pid := e.EOwner()
		return []engine.Message{engine.DrawCards{Player: pid, N: 1}}
	}})
	engine.RegisterBehavior("61020", &engine.Behavior{SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
		var out []engine.Message
		for _, p := range g.Players {
			out = append(out, engine.DealEncounterToPlayer{Player: p.ID})
		}
		return out
	}})
	engine.RegisterBehavior("61021", &engine.Behavior{SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
		if v := g.Villains[g.ActiveVillain]; v != nil {
			return []engine.Message{engine.DamageEntity{Target: v.ID, Damage: 5 * len(g.Players), Source: s.Owner}}
		}
		return nil
	}})
	engine.RegisterBehavior("61022", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		if _, ok := msg.(engine.ResourcePay); ok && g.MainScheme != nil {
			return []engine.Message{engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 1, Source: e.EOwner()}}
		}
		return nil
	}})
	engine.RegisterBehavior("61023", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		pid := e.EOwner()
		return []engine.Message{engine.BasicThwart{Player: pid, N: 2, Target: g.MainScheme.ID}}
	}})
	engine.RegisterBehavior("61024", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		if m, ok := msg.(engine.ApplySchemeThreat); ok && g.Player(e.EOwner()) != nil {
			return []engine.Message{engine.ChangeForm{Player: e.EOwner()}, engine.DamageEntity{Target: g.ActiveVillain, Damage: m.N, Source: e.EOwner()}, engine.DiscardControlled{Player: e.EOwner(), ID: e.EID()}}
		}
		return nil
	}})
	engine.RegisterBehavior("61025", &engine.Behavior{})
	engine.RegisterBehavior("61026", &engine.Behavior{})
	engine.RegisterBehavior("61027", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		pid := e.EOwner()
		return append([]engine.Message{engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 3, Source: pid}, engine.HealEntity{Target: pid, N: 3}}, nil...)
	}})
	engine.RegisterBehavior("61028", &engine.Behavior{SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
		for _, p := range g.Players {
			p.Deck = append(p.Deck, p.Discard...)
			p.Discard = nil
		}
		return nil
	}})
	engine.RegisterBehavior("61029", &engine.Behavior{SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
		return []engine.Message{engine.DrawCards{Player: s.Owner, N: 1}}
	}})
}

func registerGenericPlayerCards() {
	for _, code := range []string{"61034", "61037", "61038", "61040", "61025", "61026", "61027"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}
	engine.RegisterBehavior("61034", &engine.Behavior{IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{ATK: 1} }})
	engine.RegisterBehavior("61036", &engine.Behavior{IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{THW: 1} }})
	// Innate Inspiration's ally hit-point increase has no persistent ally-HP
	// modifier hook; the data-layer printed text remains the visible fallback.
	engine.RegisterBehavior("61039", &engine.Behavior{SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
		for _, p := range g.Players {
			if !p.IsHero() {
				p.Damage -= p.RecoverStat(g)
				if p.Damage < 0 {
					p.Damage = 0
				}
			}
		}
		return nil
	}})
}

func registerObligation() {
	engine.RegisterBehavior("61030", &engine.Behavior{ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
		var out []engine.Message
		if p.IsHero() {
			out = append(out, engine.ChangeForm{Player: p.ID})
		}
		out = append(out, engine.DiscardCards{Player: p.ID, Cards: p.Hand[:min(1, len(p.Hand))]}, engine.ObligationResolve{Player: p.ID, Card: card})
		return out
	}})
}

func registerNemesis() {
	engine.RegisterBehavior("61031", &engine.Behavior{EnemyStatBonus: func(g *engine.Game, e engine.Entity) (int, int) {
		atk, sch := 0, 0
		for _, p := range g.Players {
			if p != nil {
				if p.AttackStat(g) > atk {
					atk = p.AttackStat(g)
				}
				if p.ThwartStat(g) > sch {
					sch = p.ThwartStat(g)
				}
			}
		}
		return atk, sch
	}})
	// Indomitable Will's pheromone placement is approximated because
	// obligations have no persistent in-play entity in this engine.
	engine.RegisterBehavior("61032", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		return []engine.Message{engine.SchemeThreat{Scheme: e.EID(), N: 3 * len(g.Players), Source: e.EID()}}
	}})
	for _, code := range []string{"61033a", "61033b", "61033c"} {
		engine.RegisterBehavior(code, &engine.Behavior{ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			out := []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
			if g.MainScheme != nil {
				out = append(out, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: p.ID})
			}
			return out
		}})
	}
}
