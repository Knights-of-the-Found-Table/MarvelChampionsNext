package psylocke

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func registerPsylockeExtras() {
	// 41002a Psi-Knife: +1 THW + mental resource.
	engine.RegisterBehavior("41002", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{THW: 1} },
		Resource:      &engine.ResourceAbility{Icon: "mental", HeroOnly: true},
	})

	// 41012 Captain Britain: cheaper consequential (approximated flat).
	engine.RegisterBehavior("41012", &engine.Behavior{})

	// 41013 Cypher: draw after hitting a confused enemy.
	engine.RegisterBehavior("41013", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyAttackWindow)
			if !ok || m.Ally != e.EID() {
				return nil
			}
			a := g.Allies[e.EID()]
			mn := g.Minions[m.Target]
			if a == nil || mn == nil || !mn.Confused {
				return nil
			}
			return []engine.Message{engine.DrawCards{Player: a.Owner, N: 1}}
		},
	})

	// 41014 Concussive Blow: confuse (+3 with a physical payment,
	// approximated when a physical card sits in hand).
	engine.RegisterBehavior("41014", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || len(g.Enemies()) == 0 {
				return nil
			}
			n := 0
			for _, c := range p.Hand {
				for _, r := range c.Def().Resources {
					if r == "physical" || r == "wild" {
						n = 3
					}
				}
			}
			var choices []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				enemy := g.Entity(id)
				msgs := []engine.Message{engine.ConfuseEntity{Target: id}}
				if n > 0 {
					msgs = append(msgs, engine.DamageEntity{Target: id, Damage: n, Source: p.ID})
				}
				choices = append(choices, engine.Choice{
					Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget, SourceID: id,
				}.Msgs(msgs...))
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Concussive Blow — target:", choices...)}}
		},
	})

	// 41015 Upside the Head: confuse/stun after basic attacks.
	engine.RegisterBehavior("41015", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.BasicAttack)
			if !ok {
				return nil
			}
			p := g.Player(e.EOwner())
			_ = p
			mn := g.Minions[m.Target]
			if mn != nil {
				if mn.Confused {
					return []engine.Message{engine.StunEntity{Target: mn.ID}}
				}
				return []engine.Message{engine.ConfuseEntity{Target: mn.ID}}
			}
			if v := g.Villains[m.Target]; v != nil {
				if v.Confused {
					return []engine.Message{engine.StunEntity{Target: v.ID}}
				}
				return []engine.Message{engine.ConfuseEntity{Target: v.ID}}
			}
			return nil
		},
	})

	// 41016 Lay the Trap: defeat nukes the villain.
	engine.RegisterBehavior("41016", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			var msgs []engine.Message
			for id := range g.Villains {
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 5 * len(g.Players), Source: engine.PlayerID("")})
				break
			}
			return msgs
		},
	})

	// 41017 Float Like a Butterfly: +1 damage vs confused (approximated:
	// +1 ATK aura).
	engine.RegisterBehavior("41017", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{ATK: 1} },
	})

	// 41018 Pete Wisdom: heal after treacheries resolve.
	engine.RegisterBehavior("41018", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "X-Force")
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.TreacheryResolve); !ok {
				return nil
			}
			a := g.Allies[e.EID()]
			if a == nil {
				return nil
			}
			return []engine.Message{engine.HealEntity{Target: a.ID, N: 1}}
		},
	})

	// 41019 Directed Force: +2 damage on keyword attacks (approximated
	// flat +2 on basic attacks).
	engine.RegisterBehavior("41019", &engine.Behavior{})

	// 41021 The Power of the Mind: engine powerOfBonus.
	engine.RegisterBehavior("41021", &engine.Behavior{})

	// 41022/41023 IPAC & X-Bunker: same engine as the X-23 versions.
	engine.RegisterBehavior("41022", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "X-Force")
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "IPAC — trade a facedown encounter for 2 cards", Type: engine.AbilityAction,
				HeroOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.Owner)
					if p == nil {
						return nil
					}
					return []engine.Message{
						engine.DealEncounterToPlayer{Player: p.ID},
						engine.DrawCards{Player: p.ID, N: 2},
					}
				},
			}}
		},
	})
	engine.RegisterBehavior("41023", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			n := 0
			for _, c := range g.VictoryDisplay {
				if c.Def().Type == "side_scheme" || c.Def().Type == "player_side_scheme" {
					n++
				}
			}
			if n == 0 {
				return nil
			}
			return []engine.Ability{{
				Label: "X-Bunker — dig the top card", Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.Owner)
					if p == nil || len(p.Deck) == 0 {
						return nil
					}
					c := p.Deck[0]
					p.Deck = p.Deck[1:]
					p.Hand = append(p.Hand, c)
					g.Logf("X-Bunker recovers %s", c.Def().Name)
					return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
				},
			}}
		},
	})

	// 41024 Telepathy: 2 threat for mental x2.
	engine.RegisterBehavior("41024", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "Psionic")
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Telepathy — remove 2 threat (mental x2)", Type: engine.AbilityAction,
				HeroOnly: true, Exhaust: true, CostIcons: "mental:2",
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
						}.Msgs(engine.ThwartScheme{Scheme: id, N: 2, Source: p.ID}))
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Telepathy — remove 2 threat from:", choices...)}}
				},
			}}
		},
	})

	// 41030 Psi-Bow Attack: 4 damage.
	engine.RegisterBehavior("41030", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "Psionic")
		},
		OnPlay: cardutil.ChooseEnemy("Psi-Bow Attack", func(g *engine.Game, e engine.Entity) (int, []engine.Message) {
			return 4, nil
		}),
	})

	// 41031 Domino (psylocke printing): swap after basic powers.
	engine.RegisterBehavior("41031", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			switch msg.(type) {
			case engine.AllyAttackWindow, engine.AllyThwartWindow:
			default:
				return nil
			}
			a := g.Allies[e.EID()]
			p := g.Player(a.Owner)
			if p == nil || len(p.Hand) == 0 || len(p.Deck) == 0 {
				return nil
			}
			c := p.Hand[0]
			g.Logf("Domino swaps %s with the deck top", c.Def().Name)
			return []engine.Message{engine.SwapHandWithDeckTop{Player: p.ID, CardID: c.ID}}
		},
	})

	// 41032 Psi-Flail Strike: counter-attack after defending.
	engine.RegisterBehavior("41032", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "Psionic")
		},
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			d := engine.Defends{Defender: p.ID, Against: against}
			return d, []engine.Message{
				engine.DamageEntity{Target: against, Damage: 3, Source: p.ID},
				engine.StunEntity{Target: against},
			}, true
		},
	})

	// 41033 Telekinesis: 3 damage for mental x2.
	engine.RegisterBehavior("41033", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "Psionic")
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Telekinesis — 3 damage (mental x2)", Type: engine.AbilityAction,
				HeroOnly: true, Exhaust: true, CostIcons: "mental:2",
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					p := g.Player(u.Owner)
					if p == nil || len(g.Enemies()) == 0 {
						return nil
					}
					var choices []engine.Choice
					for _, id := range cardutil.SortedEnemyIDs(g) {
						enemy := g.Entity(id)
						choices = append(choices, engine.Choice{
							Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget, SourceID: id,
						}.Msgs(engine.DamageEntity{Target: id, Damage: 3, Source: p.ID}))
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Telekinesis — target:", choices...)}}
				},
			}}
		},
	})
}
