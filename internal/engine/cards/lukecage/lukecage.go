// Package lukecage registers Luke Cage's hero pack (62001-62037).
package lukecage

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() {
	registerIdentity()
	registerCards()
	registerObligation()
	registerNemesis()
}

func registerIdentity() {
	// Piercing is not represented by the combat message model. Tough stacking
	// and the forced unpreventable damage are represented exactly.
	engine.RegisterBehavior("62001", &engine.Behavior{
		UnlimitedTough: true,
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.ToughDiscarded)
			if !ok || m.Target != e.EID() {
				return nil
			}
			return []engine.Message{engine.DamageEntity{
				Target: e.EID(), Damage: 1, Source: e.EID(), Unpreventable: true,
			}}
		},
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			if p == nil || p.IsHero() || g.UsedThisRound["62001-burstein"] {
				return nil
			}
			var choices []engine.Choice
			for _, c := range append(append(engine.CardList(nil), p.Deck...), p.Discard...) {
				if c.Code == "62008" {
					choices = append(choices, engine.Choice{Label: engine.S("Take Burstein Process"), Kind: engine.ChoiceCard, CardCode: c.Code}.
						Msgs(engine.ReturnDiscardCard{Player: p.ID, CardID: c.ID}))
				}
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.S("Burstein Process — search your deck and discard pile"),
				Type:  engine.AbilityAction, AlterEgoOnly: true, OncePerRound: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					g.UsedThisRound["62001-burstein"] = true
					pl := g.Player(self)
					for _, c := range pl.Deck {
						if c.Code == "62008" {
							return []engine.Message{engine.TakeDeckCard{Player: self, CardID: c.ID}, engine.ShufflePlayerDeck{Player: self}}
						}
					}
					for _, c := range pl.Discard {
						if c.Code == "62008" {
							return []engine.Message{engine.ReturnDiscardCard{Player: self, CardID: c.ID}}
						}
					}
					return nil
				},
			}}
		},
	})
}

func registerCards() {
	// Jessica Jones scales with the identity's actual tough count.
	engine.RegisterBehavior("62002", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		if a, ok := e.(*engine.Ally); ok {
			if p := g.Player(a.Owner); p != nil {
				n := p.Tough
				if n > 4 {
					n = 4
				}
				a.PermTHW = n
			}
		}
		return nil
	}})
	engine.RegisterBehavior("62003", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		if p == nil {
			return nil
		}
		msgs := []engine.Message{engine.ToughEntity{Target: p.ID}}
		for _, id := range p.Allies {
			msgs = append(msgs, engine.ReadyEntity{ID: id})
		}
		msgs = append(msgs, engine.ReadyEntity{ID: p.ID})
		return msgs
	}})
	engine.RegisterBehavior("62004", &engine.Behavior{OnPlay: cardutil.ChooseEnemy(engine.S("Knuckle Sandwich — choose an enemy"), func(g *engine.Game, e engine.Entity) (int, []engine.Message) { return 3, nil })})
	engine.RegisterBehavior("62005", &engine.Behavior{OnPlay: cardutil.ChooseScheme(engine.Tf("c.chooseAScheme", "Stand with Me! — choose a scheme"), func(g *engine.Game, e engine.Entity) int {
		p := g.Player(e.EOwner())
		if p == nil {
			return 0
		}
		n := p.Tough
		if n > 5 {
			n = 5
		}
		return n
	})})
	engine.RegisterBehavior("62006", &engine.Behavior{CardCost: func(g *engine.Game, p *engine.Player, def *data.CardDef) int {
		if def == nil || def.Code != "62006" || def.Cost == nil {
			return 0
		}
		n := p.Tough
		if n > *def.Cost {
			n = *def.Cost
		}
		return n
	}})
	engine.RegisterBehavior("62007", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.ToughDiscarded)
		if !ok {
			return nil
		}
		if ch := g.Entity(m.Target); ch != nil && ch.EDef().HasTrait("defender") {
			return []engine.Message{engine.ReadyEntity{ID: m.Target}}
		}
		return nil
	}})
	engine.RegisterBehavior("62008", &engine.Behavior{Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
		u := e.(*engine.Upgrade)
		p := g.Player(u.Owner)
		if p == nil || p.IsHero() {
			return nil
		}
		return []engine.Ability{{Label: engine.S("Burstein Process — give Luke Cage tough"), Type: engine.AbilityAction, AlterEgoOnly: true, Exhaust: true, Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
			p := g.Player(u.Owner)
			n := 1
			if p.Tough == 0 {
				n = 2
			}
			out := []engine.Message{}
			for i := 0; i < n; i++ {
				out = append(out, engine.ToughEntity{Target: p.ID})
			}
			return out
		}}}
	}})
	engine.RegisterBehavior("62010", &engine.Behavior{IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{Retaliate: 1} }, DamagePrevention: func(g *engine.Game, u *engine.Upgrade, p *engine.Player, n int) (int, int) {
		if u.Exhausted {
			return 0, 0
		}
		u.Exhausted = true
		return 1, 0
	}})
	engine.RegisterBehavior("62011", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.ToughDiscarded)
		u := e.(*engine.Upgrade)
		if !ok || m.Target != u.Owner || u.Exhausted {
			return nil
		}
		return []engine.Message{engine.ExhaustEntity{ID: u.ID}, engine.DrawCards{Player: u.Owner, N: 1}}
	}})
	engine.RegisterBehavior("62015", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		if _, ok := msg.(engine.Defends); ok {
			return []engine.Message{engine.ApplyStatBonus{Target: e.EOwner(), ATK: 1, THW: 1}}
		}
		return nil
	}})
	engine.RegisterBehavior("62018", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 3}}
	}, Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
		s := e.(*engine.Support)
		if s.Counters <= 0 || s.Exhausted {
			return nil
		}
		return []engine.Ability{{Label: engine.S("Defensive Formation — give a Defender ally tough"), Type: engine.AbilityAction, Exhaust: true, Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
			s := g.Supports[self]
			s.Counters--
			for _, id := range g.Player(s.Owner).Allies {
				if g.Allies[id].EDef().HasTrait("defender") {
					return []engine.Message{engine.ToughEntity{Target: id}}
				}
			}
			return nil
		}}}
	}})
	engine.RegisterBehavior("62021", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		if m, ok := msg.(engine.AllyEnteredPlay); ok && m.Ally == e.EID() {
			return []engine.Message{engine.ToughEntity{Target: e.EOwner()}}
		}
		return nil
	}})
	engine.RegisterBehavior("62023", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		if p == nil {
			return nil
		}
		return []engine.Message{engine.ToughEntity{Target: p.ID}}
	}})
	engine.RegisterBehavior("62028", &engine.Behavior{ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
		return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 3, Source: p.ID, Unpreventable: true}, engine.ObligationResolve{Player: p.ID, Card: card}}
	}})
	engine.RegisterBehavior("62034", &engine.Behavior{OnPlay: cardutil.ChooseEnemy(engine.S("Size Advantage — choose an enemy"), func(g *engine.Game, e engine.Entity) (int, []engine.Message) { return 3, nil })})
	engine.RegisterBehavior("62035", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		if p == nil {
			return nil
		}
		for id := range g.Villains {
			return []engine.Message{engine.ApplyVillainScheme{VillainID: id, Player: p.ID}, engine.ToughEntity{Target: p.ID}}
		}
		return []engine.Message{engine.ToughEntity{Target: p.ID}}
	}})
	engine.RegisterBehavior("62036", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		if _, ok := msg.(engine.BasicRecover); ok {
			u := e.(*engine.Upgrade)
			if !u.Exhausted {
				return []engine.Message{engine.ExhaustEntity{ID: u.ID}, engine.ToughEntity{Target: u.Owner}}
			}
		}
		return nil
	}})
	for _, code := range []string{"62009", "62012", "62013", "62014", "62016", "62017", "62019", "62020", "62022", "62024", "62025", "62026", "62027", "62031", "62032", "62033", "62037"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}
}

func registerObligation() { /* 62028 is registered with the shared obligation behavior above. */ }

func registerNemesis() {
	engine.RegisterBehavior("62029", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		mn := e.(*engine.Minion)
		if _, ok := msg.(engine.BeginPhase); ok {
			mn.Counters = 0
			return nil
		}
		m, ok := msg.(engine.WindowAfterEnemyAttacked)
		if !ok || m.Enemy != mn.ID || mn.Counters > 0 {
			return nil
		}
		mn.Counters = 1
		return []engine.Message{engine.AskAttack{Enemy: mn.ID, Player: m.Player, Trigger: "cottonmouth"}, engine.AskAttack{Enemy: mn.ID, Player: m.Player, Trigger: "cottonmouth"}}
	}})
	engine.RegisterBehavior("62030", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		if m, ok := msg.(engine.ThwartScheme); ok && m.Scheme == e.EID() {
			return nil
		}
		return nil
	}})
	engine.RegisterBehavior("62031", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		if m, ok := msg.(engine.Defends); ok {
			return []engine.Message{engine.ClearAllTough{Target: m.Defender}}
		}
		return nil
	}})
}
