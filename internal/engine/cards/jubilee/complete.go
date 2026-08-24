package jubilee

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func registerJubileeExtras() {
	// 47011 Chamber: -1 consequential vs confused (approximated flat).
	engine.RegisterBehavior("47011", &engine.Behavior{})

	// 47012 Husk: adaptive basic powers (approximated: +1 to powers this
	// phase on use).
	engine.RegisterBehavior("47012", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			var ally engine.EntityID
			switch m := msg.(type) {
			case engine.AllyAttackWindow:
				ally = m.Ally
			case engine.AllyThwartWindow:
				ally = m.Ally
			default:
				return nil
			}
			if ally != e.EID() {
				return nil
			}
			a := g.Allies[e.EID()]
			if a == nil {
				return nil
			}
			a.BonusATK++
			a.BonusTHW++
			return nil
		},
	})

	// 47013 Disguise: 2 threat for identity exhaust.
	engine.RegisterBehavior("47013", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Disguise — remove 2 threat", Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					p := g.Player(u.Owner)
					if p == nil || len(g.Schemes()) == 0 {
						return nil
					}
					var choices []engine.Choice
					for _, id := range g.Schemes() {
						s := g.Entity(id)
						choices = append(choices, engine.Choice{
							Label: s.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id,
						}.Msgs(engine.ExhaustEntity{ID: p.ID}, engine.ThwartScheme{Scheme: id, N: 2, Source: p.ID}))
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Disguise — remove 2 threat from:", choices...)}}
				},
			}}
		},
	})

	// 47014 Waylay: 4 damage after thwarting (approximated: immediate 4).
	engine.RegisterBehavior("47014", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy("Waylay", func(g *engine.Game, e engine.Entity) (int, []engine.Message) {
			return 4, nil
		}),
	})

	// 47015 Three Steps Ahead: 2 threat per resource type (approximated:
	// 2 threat × 2 schemes).
	engine.RegisterBehavior("47015", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var msgs []engine.Message
			for i, id := range g.Schemes() {
				if i >= 2 {
					break
				}
				msgs = append(msgs, engine.ThwartScheme{Scheme: id, N: 2, Source: p.ID})
			}
			return msgs
		},
	})

	// 47016 Generation X: fetch identity events on defeat.
	engine.RegisterBehavior("47016", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			for _, p := range g.Players {
				heroSet := ""
				if d, ok := engine.DB.Lookup(p.HeroCode); ok {
					heroSet = d.CardSet
				}
				for _, c := range append(engine.CardList{}, p.Deck...) {
					if c.Def().Type == "event" && c.Def().CardSet == heroSet {
						if _, ok := p.Deck.Remove(c.ID); ok {
							p.Hand = append(p.Hand, c)
						}
						break
					}
				}
			}
			return nil
		},
	})

	// 47017 The Power of Justice: engine powerOfBonus.
	engine.RegisterBehavior("47017", &engine.Behavior{})

	// 47018 Synch: +1 to basic powers.
	engine.RegisterBehavior("47018", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "X-Men")
		},
	})

	// 47019 Cell Phone: charges for boosted actions.
	engine.RegisterBehavior("47019", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			g.Upgrades[e.EID()].Counters = 3
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			u := g.Upgrades[e.EID()]
			if u == nil || u.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: "Cell Phone — boosted basic power (+1)", Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					p := g.Player(u.Owner)
					if p == nil {
						return nil
					}
					u.Counters--
					g.EventDamageBonus[p.ID] += 1
					g.EventThreatBonus[p.ID] += 1
					return nil
				},
			}}
		},
	})

	// 47020 X-Gene: wild resource for identity events.
	engine.RegisterBehavior("47020", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "Mutant")
		},
		Resource: &engine.ResourceAbility{Icon: "wild", EventOnly: true},
	})

	// 47021 Multitalented: pay-mix effects (approximated: all three).
	engine.RegisterBehavior("47021", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var msgs []engine.Message
			if len(g.Enemies()) > 0 {
				msgs = append(msgs, engine.DamageEntity{Target: g.Enemies()[0], Damage: 2, Source: p.ID})
			}
			if len(g.Schemes()) > 0 {
				msgs = append(msgs, engine.ThwartScheme{Scheme: g.Schemes()[0], N: 2, Source: p.ID})
			}
			msgs = append(msgs, engine.HealEntity{Target: p.ID, N: 2})
			return msgs
		},
	})

	// 47022 Unlikely Duo: confuse + hit.
	engine.RegisterBehavior("47022", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || len(g.Enemies()) == 0 {
				return nil
			}
			var choices []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				enemy := g.Entity(id)
				choices = append(choices, engine.Choice{
					Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget, SourceID: id,
				}.Msgs(engine.ConfuseEntity{Target: id}, engine.DamageEntity{Target: id, Damage: 4, Source: p.ID}))
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Unlikely Duo — target:", choices...)}}
		},
	})

	// 47028 Mutant Mayhem: bounce and replay two allies (approximated:
	// ready + heal them).
	engine.RegisterBehavior("47028", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var msgs []engine.Message
			n := 0
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a == nil {
					continue
				}
				if (a.EDef().HasTrait("X-Force") || a.EDef().HasTrait("X-Men")) && n < 2 {
					msgs = append(msgs, engine.ReadyEntity{ID: id}, engine.HealEntity{Target: id, N: 1})
					n++
				}
			}
			return msgs
		},
	})

	// 47029 Serve and Protect: prevent threat with toughs (approximated:
	// threat prevention window not modeled; grants tough).
	engine.RegisterBehavior("47029", &engine.Behavior{})

	registerArcade()
}

func registerArcade() {
	// 47030 Arcade: immune while a Trap! lives; fetch one on reveal.
	engine.RegisterBehavior("47030", &engine.Behavior{
		MinionDamageable: func(g *engine.Game, m *engine.Minion, damage int) bool {
			for _, s := range g.SideSchemes {
				if s != nil && s.EDef().HasTrait("Trap!") {
					g.Logf("Arcade cannot take damage while a Trap! is in play")
					return false
				}
			}
			return true
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return nil
			}
			for i := 0; i < 30; i++ {
				c, ok := g.DrawEncounter()
				if !ok {
					return nil
				}
				if c.Def().Type == "side_scheme" && c.Def().HasTrait("Trap!") {
					return []engine.Message{engine.RevealEncounterCard{Player: mn.EngagedWith, Card: c}}
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			return nil
		},
	})

	// 47031 Welcome to Murderworld.
	engine.RegisterBehavior("47031", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			pid := cardutil.FirstPlayerID(g)
			return []engine.Message{engine.DamageEntity{Target: pid, Damage: 2, Source: s.ID}}
		},
	})

	// 47032 Arcade's Funhouse.
	engine.RegisterBehavior("47032", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			p := g.Player(cardutil.FirstPlayerID(g))
			if p == nil {
				return nil
			}
			if p.Stunned {
				if len(p.Allies) > 0 {
					return []engine.Message{engine.AllyDestroyed{AllyID: p.Allies[0]}}
				}
				if len(p.Upgrades) > 0 {
					return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Upgrades[0]}}
				}
				return nil
			}
			return []engine.Message{engine.StunEntity{Target: p.ID}}
		},
	})

	// 47033 Hall of Mirrors.
	engine.RegisterBehavior("47033", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			p := g.Player(cardutil.FirstPlayerID(g))
			if p == nil {
				return nil
			}
			if p.Confused && g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: s.ID}}
			}
			return []engine.Message{engine.ConfuseEntity{Target: p.ID}}
		},
	})

	// 47034 Elaborate Trap: resolve every Trap!'s defeat rider.
	engine.RegisterBehavior("47034", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var msgs []engine.Message
			for _, s := range g.SideSchemes {
				if s == nil || !s.EDef().HasTrait("Trap!") {
					continue
				}
				if b := engine.LookupBehavior(s.Code); b.SideSchemeDefeated != nil {
					msgs = append(msgs, b.SideSchemeDefeated(g, s)...)
				}
			}
			if len(msgs) > 0 {
				return msgs
			}
			for i := 0; i < 30; i++ {
				c, ok := g.DrawEncounter()
				if !ok {
					return nil
				}
				if c.Def().Type == "side_scheme" && c.Def().HasTrait("Trap!") {
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			return nil
		},
	})
}
