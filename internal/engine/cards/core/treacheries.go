package core

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// registerTreacheries installs Standard-set and Bomb Scare treacheries plus
// key encounter minions.
func registerTreacheries() {
	// Advance: place 1 threat on the main scheme (2 if it has none).
	engine.RegisterBehavior("01186", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if g.MainScheme == nil {
				return nil
			}
			n := 1
			if g.MainScheme.Threat == 0 {
				n = 2
			}
			g.Delete(t.ID)
			return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: n, Source: t.ID}}
		},
	})

	// Assault: the villain attacks you.
	engine.RegisterBehavior("01187", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for id := range g.Villains {
				return []engine.Message{engine.VillainActivates{VillainID: id, Player: p.ID}}
			}
			return nil
		},
	})

	// Caught Off Guard: discard one of your supports or upgrades
	// (approximation: the most recently played one).
	engine.RegisterBehavior("01188", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			if len(p.Supports) > 0 {
				id := p.Supports[len(p.Supports)-1]
				s := g.Supports[id]
				g.Delete(id)
				g.Logf("Caught Off Guard discards %s", s.EDef().Name)
			} else if len(p.Upgrades) > 0 {
				id := p.Upgrades[len(p.Upgrades)-1]
				u := g.Upgrades[id]
				g.Delete(id)
				g.Logf("Caught Off Guard discards %s", u.EDef().Name)
			} else {
				g.Logf("Caught Off Guard has no target")
			}
			return nil
		},
	})

	// Gang-Up: each villain and minion attacks you.
	engine.RegisterBehavior("01189", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var msgs []engine.Message
			for _, id := range cardutil.SortedIDs(g.Villains) {
				msgs = append(msgs, engine.VillainActivates{VillainID: id, Player: p.ID})
			}
			for _, id := range cardutil.SortedIDs(g.Minions) {
				msgs = append(msgs, engine.MinionActivates{MinionID: id, Player: p.ID})
			}
			return msgs
		},
	})

	// Shadow of the Past: reveal your nemesis set.
	engine.RegisterBehavior("01190", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			return []engine.Message{engine.RevealNemesisSet{Player: p.ID}}
		},
	})

	// Bomb Scare side scheme: acceleration token on the main scheme.
	engine.RegisterBehavior("01109", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if g.MainScheme != nil {
				g.MainScheme.AccelerationTokens++
				g.Logf("Bomb Scare: acceleration token on %s", g.MainScheme.EDef().Name)
			}
			return nil
		},
	})

	// Hydra Bomber: when revealed, take 2 damage or place 1 threat.
	engine.RegisterBehavior("01110", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn, ok := e.(*engine.Minion)
			if !ok || mn.EngagedWith == "" {
				return nil
			}
			p := g.Player(mn.EngagedWith)
			if p == nil {
				return nil
			}
			var threatChoice engine.Choice
			if g.MainScheme != nil {
				threatChoice = engine.Choice{
					ID: "threat", Label: "Place 1 threat on the main scheme", Kind: engine.ChoiceLabel,
				}.Msgs(engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: e.EID()})
			} else {
				threatChoice = engine.Choice{
					ID: "threat", Label: "Place 1 threat (no scheme)", Kind: engine.ChoicePass,
				}
			}
			return []engine.Message{engine.AskQuestion{
				Player: p.ID,
				Question: engine.Ask("Hydra Bomber: take 2 damage or place 1 threat?",
					threatChoice,
					engine.Choice{
						ID: "damage", Label: "Take 2 damage", Kind: engine.ChoiceLabel,
					}.Msgs(engine.DamageEntity{Target: p.ID, Damage: 2, Source: e.EID()}),
				),
			}}
		},
	})

	// Explosion: assign damage equal to Bomb Scare's threat (approximation:
	// all to the revealing hero).
	engine.RegisterBehavior("01111", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			n := 0
			for _, s := range g.SideSchemes {
				if s.Code == "01109" {
					n = s.Threat
					break
				}
			}
			if n == 0 {
				g.Logf("Explosion fizzles (Bomb Scare not in play)")
				return nil
			}
			return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: n, Source: t.ID}}
		},
	})

	// False Alarm: you are confused.
	engine.RegisterBehavior("01112", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			if p.Confused {
				// already confused: surge
				if c, ok := g.DrawEncounter(); ok {
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
				return nil
			}
			return []engine.Message{engine.ConfuseEntity{Target: p.ID}}
		},
	})

	// Rhino set treacheries.
	// "I'm Tough": remove 3 threat from the villain (Rhino heals? actual:
	// remove 3 tokens... treat as heal villain 3).
	engine.RegisterBehavior("01105", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var msgs []engine.Message
			for _, v := range g.Villains {
				if v.Damage >= 3 {
					msgs = append(msgs, engine.HealEntity{Target: v.ID, N: 3})
				} else if v.Damage > 0 {
					msgs = append(msgs, engine.HealEntity{Target: v.ID, N: v.Damage})
				}
				break
			}
			return msgs
		},
	})

	// Stampede: the villain attacks you (approximation of "Rhino attacks
	// and gets +1 ATK per Hydra Mercenary engaged with you").
	engine.RegisterBehavior("01106", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for id := range g.Villains {
				return []engine.Message{engine.VillainActivates{VillainID: id, Player: p.ID}}
			}
			return nil
		},
	})

	// Hard to Keep Down: spawn the top Hydra Mercenary (approximation:
	// spawn the top minion of the encounter deck).
	engine.RegisterBehavior("01104", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for i, c := range g.EncounterDeck {
				if c.Def().Type == "minion" {
					g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
			}
			return nil
		},
	})
}
