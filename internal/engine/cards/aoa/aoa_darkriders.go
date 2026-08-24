package aoa

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func registerDarkRiders() {
	// 45112-45116 Dark Riders: shared after-attack riders.
	rider := func(apply func(g *engine.Game, p *engine.Player) []engine.Message) *engine.Behavior {
		return &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				m, ok := msg.(engine.WindowAfterEnemyAttacked)
				if !ok || m.Enemy != e.EID() {
					return nil
				}
				pid := m.Player
				if a := g.Allies[engine.EntityID(pid)]; a != nil {
					pid = a.Owner
				}
				return apply(g, g.Player(pid))
			},
		}
	}
	// Gauntlet: discard an upgrade.
	engine.RegisterBehavior("45112", rider(func(g *engine.Game, p *engine.Player) []engine.Message {
		if p == nil || len(p.Upgrades) == 0 {
			return nil
		}
		return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Upgrades[0]}}
	}))
	// Barrage: 1 damage to each character.
	engine.RegisterBehavior("45113", rider(func(g *engine.Game, p *engine.Player) []engine.Message {
		if p == nil {
			return nil
		}
		var msgs []engine.Message
		msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: 1, Source: engine.EntityID("")})
		for _, id := range p.Allies {
			msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: engine.EntityID("")})
		}
		return msgs
	}))
	// Hard-Drive: 1 threat on each scheme.
	engine.RegisterBehavior("45114", rider(func(g *engine.Game, p *engine.Player) []engine.Message {
		var msgs []engine.Message
		if g.MainScheme != nil {
			msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: engine.EntityID("")})
		}
		for _, id := range cardutil.SortedIDs(g.SideSchemes) {
			msgs = append(msgs, engine.ApplySchemeThreat{Scheme: id, N: 1, Source: engine.EntityID("")})
		}
		return msgs
	}))
	// Tusk: stunned; boost stuns too.
	engine.RegisterBehavior("45115", rider(func(g *engine.Game, p *engine.Player) []engine.Message {
		if p == nil {
			return nil
		}
		return []engine.Message{engine.StunEntity{Target: p.ID}}
	}))
	engine.LookupBehavior("45115").Boost = func(g *engine.Game, card engine.Card) []engine.Message {
		return stunBoostAoa(g, card)
	}
	// Psynapse: confused; boost confuses too.
	engine.RegisterBehavior("45116", rider(func(g *engine.Game, p *engine.Player) []engine.Message {
		if p == nil {
			return nil
		}
		return []engine.Message{engine.ConfuseEntity{Target: p.ID}}
	}))
	engine.LookupBehavior("45116").Boost = func(g *engine.Game, card engine.Card) []engine.Message {
		if p := g.Player(cardutil.FirstPlayerID(g)); p != nil {
			return []engine.Message{engine.ConfuseEntity{Target: p.ID}}
		}
		return nil
	}

	// 45117 The Dark Riders: Hinder parsed; toughness aura approximated
	// on reveal; summon a Dark Rider.
	engine.RegisterBehavior("45117", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			s := g.SideSchemes[e.EID()]
			if s == nil {
				return nil
			}
			for _, mn := range g.Minions {
				if mn != nil && mn.EDef().HasTrait("Dark Riders") {
					mn.Tough = true
				}
			}
			for i := 0; i < 30; i++ {
				c, ok := g.DrawEncounter()
				if !ok {
					return nil
				}
				if c.Def().Type == "minion" && c.Def().HasTrait("Dark Riders") {
					return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			return nil
		},
	})
}

func stunBoostAoa(g *engine.Game, card engine.Card) []engine.Message {
	if p := g.Player(cardutil.FirstPlayerID(g)); p != nil {
		return []engine.Message{engine.StunEntity{Target: p.ID}}
	}
	return nil
}
