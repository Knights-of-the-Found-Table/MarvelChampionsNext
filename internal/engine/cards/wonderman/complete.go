package wonderman

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

func init() { registerWonderManComplete() }

func wmEnemies(g *engine.Game) []engine.EntityID {
	var out []engine.EntityID
	for id := range g.Villains {
		out = append(out, id)
	}
	for id := range g.Minions {
		out = append(out, id)
	}
	return out
}

func registerWonderManComplete() {
	// 58012 Firebird: overpay riders not modeled.
	engine.RegisterBehavior("58012", &engine.Behavior{})
	// 58013 Hawkeye: 4 arrow counters, 2 damage per ally thwart.
	engine.RegisterBehavior("58013", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			g.Allies[e.EID()].Counters = 4
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyThwartWindow)
			if !ok || m.Ally == e.EID() {
				return nil
			}
			a := g.Allies[e.EID()]
			if a == nil || a.Counters <= 0 {
				return nil
			}
			a.Counters--
			for _, id := range wmEnemies(g) {
				if mn := g.Minions[id]; mn != nil {
					return []engine.Message{engine.DamageEntity{Target: id, Damage: 2, Source: e.EID()}}
				}
			}
			return nil
		},
	})
	// 58014 Scarlet Witch: mills and resolves a treachery.
	engine.RegisterBehavior("58014", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for i, c := range g.EncounterDeck {
				if c.Def().Type == "treachery" {
					card := c
					g.EncounterDeck = append(g.EncounterDeck[:i:i], g.EncounterDeck[i+1:]...)
					g.TLogf("c.scarletWitchReveals", card)
					return []engine.Message{engine.RevealEncounterCard{Player: e.EOwner(), Card: card}}
				}
			}
			return nil
		},
	})
	// 58015 Sentry: drags a side scheme out.
	engine.RegisterBehavior("58015", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for i, c := range g.EncounterDeck {
				if c.Def().Type == "side_scheme" {
					g.EncounterDeck = append(g.EncounterDeck[:i:i], g.EncounterDeck[i+1:]...)
					return []engine.Message{engine.RevealEncounterCard{Player: e.EOwner(), Card: c}}
				}
			}
			if g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 6, Source: e.EID()}}
			}
			return nil
		},
	})
	// 58016 Battlefield Benevolence.
	engine.RegisterBehavior("58016", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var out []engine.Message
			for _, id := range wmEnemies(g) {
				out = append(out, engine.HealEntity{Target: id, N: 2}, engine.ConfuseEntity{Target: id})
				break
			}
			return out
		},
	})
	// 58017 Bombs Away.
	engine.RegisterBehavior("58017", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var out []engine.Message
			for _, id := range wmEnemies(g) {
				out = append(out, engine.DamageEntity{Target: id, Damage: 3, Source: e.EID()})
			}
			return out
		},
	})
	// 58018 Everywhere All at Once.
	engine.RegisterBehavior("58018", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var out []engine.Message
			n := 0
			for _, sid := range g.Schemes() {
				out = append(out, engine.ThwartScheme{Scheme: sid, N: 2, Source: e.EID()})
				n++
				if n >= 2 {
					break
				}
			}
			return out
		},
	})
	// 58019 Stronger Together: trait-share damage reduction not modeled.
	engine.RegisterBehavior("58019", &engine.Behavior{})
	// 58020 Unified Strike: power share approximated as +2/+2 event
	// bonus.
	engine.RegisterBehavior("58020", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			return []engine.Message{engine.SetEventBonus{Player: p.ID, Damage: 2, Threat: 2}}
		},
	})
	// 58021 Heroic Conditioning.
	engine.RegisterBehavior("58021", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if p := g.Player(e.EOwner()); p != nil {
				p.MaxHP += 3
			}
			return nil
		},
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{THW: 1}
		},
	})
	// 58022 Swordsman: piercing + substitute defense not modeled.
	engine.RegisterBehavior("58022", &engine.Behavior{})
	// 58023 Energy resource.
	engine.RegisterBehavior("58023", &engine.Behavior{})
	// 58024 Jarvis.
	engine.RegisterBehavior("58024", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.jarvisTendAnAlterEgoHeal2AndClearAStatus"), Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.Owner)
					if p == nil {
						return nil
					}
					return []engine.Message{
						engine.HealEntity{Target: p.ID, N: 2},
						engine.ClearStun{Target: p.ID},
						engine.ClearConfuse{Target: p.ID},
					}
				},
			}}
		},
	})
	// 58030 Caught in the Crossfire.
	engine.RegisterBehavior("58030", &engine.Behavior{})
	// 58031 Cameo: collection draft not modeled.
	engine.RegisterBehavior("58031", &engine.Behavior{})
	// 58032 Coordinated Effort: shared payment not modeled.
	engine.RegisterBehavior("58032", &engine.Behavior{})
	// 58033 Disarming Defense.
	engine.RegisterBehavior("58033", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			return engine.Defends{Defender: p.ID, Against: against, DefBonus: 2}, nil, true
		},
	})
	// 58034 Avengers Compound (shared card with the hercules printing).
	engine.RegisterBehavior("58034", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			if s == nil {
				return nil
			}
			if len(s.AttachedCards) == 0 {
				return []engine.Ability{{
					Label: engine.Tf("c.avengersCompoundTuckAnAllyFromYourHand"), Type: engine.AbilityAction, Exhaust: true,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						s := g.Supports[self]
						p := g.Player(s.Owner)
						if p == nil {
							return nil
						}
						for i, c := range p.Hand {
							if c.Def().Type == "ally" {
								card := c
								p.Hand = append(p.Hand[:i:i], p.Hand[i+1:]...)
								s.AttachedCards = append(s.AttachedCards, card)
								s.Counters = 1
								return nil
							}
						}
						return nil
					},
				}}
			}
			return []engine.Ability{{
				Label: engine.Tf("c.avengersCompoundDeployTheTuckedAlly"), Type: engine.AbilityAction, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.Owner)
					if s == nil || p == nil || len(s.AttachedCards) == 0 {
						return nil
					}
					card := s.AttachedCards[0]
					s.AttachedCards = nil
					s.Counters = 0
					return []engine.Message{engine.AllyEntersPlayFree{Player: p.ID, Card: card}}
				},
			}}
		},
	})
}
