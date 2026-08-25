package goblinfooblin

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// registerGoblinEncounterCards installs behaviors for the pack's minions,
// attachments, side schemes and treacheries.
func registerGoblinEncounterCards() {
	// Monster: when revealed, you are stunned (2 damage if already).
	engine.RegisterBehavior("02025", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn, ok := e.(*engine.Minion)
			if !ok || mn.EngagedWith == "" {
				return nil
			}
			p := g.Player(mn.EngagedWith)
			if p == nil {
				return nil
			}
			if p.Stunned {
				return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 2, Source: mn.ID}}
			}
			return []engine.Message{engine.StunEntity{Target: p.ID}}
		},
	})

	// Goblin Soldier: when defeated, 1 damage to the engaged player.
	engine.RegisterBehavior("02023", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionDefeated)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			mn, ok := e.(*engine.Minion)
			if !ok || mn.EngagedWith == "" {
				return nil
			}
			return []engine.Message{engine.DamageEntity{Target: mn.EngagedWith, Damage: 1, Source: e.EID()}}
		},
	})

	// Goblin Knight: after he attacks you, discard from the encounter deck
	// until a Goblin minion appears and put it into play.
	engine.RegisterBehavior("02022", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			mn, ok := e.(*engine.Minion)
			if !ok || mn.EngagedWith == "" {
				return nil
			}
			for i, c := range g.EncounterDeck {
				def := c.Def()
				if def.Type == "minion" && def.HasTrait("goblin") {
					g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
					g.TLogf("c.goblinKnightSummons", def.Name)
					return []engine.Message{engine.RevealEncounterCard{Player: mn.EngagedWith, Card: c}}
				}
			}
			return nil
		},
	})

	// Goblin Glider: attach to the enemy with the highest printed HP;
	// it gets +1 ATK.
	attachHighestHP("02019")
	attachHighestHP("02033")

	// Pumpkin Bombs: after the villain attacks you, 2 damage and discard.
	pumpkinBombs("02021")
	pumpkinBombs("02034")

	// Goblin Reinforcements: extra threat per Goblin minion in play.
	engine.RegisterBehavior("02026", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if s, ok := e.(*engine.SideScheme); ok {
				n := 0
				for _, mn := range g.Minions {
					if mn.EDef().HasTrait("goblin") {
						n++
					}
				}
				if n > 0 {
					s.Threat += n
					g.TLogf("c.goblinReinforcementsThreat", n)
				}
			}
			return nil
		},
	})

	// Death from Above (Mutagen Formula): the villain attacks the first
	// player with +2 ATK (approximation).
	engine.RegisterBehavior("02029", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for id := range g.Villains {
				return []engine.Message{engine.VillainActivates{VillainID: id, Player: p.ID}}
			}
			return nil
		},
	})

	// I See You: reveal the top card of the encounter deck as a boost for
	// the villain (approximation: just an extra boost card).
	engine.RegisterBehavior("02030", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for id := range g.Villains {
				return []engine.Message{engine.DealBoost{Enemy: id}}
			}
			return nil
		},
	})

	// Overconfidence: place 1 threat on the main scheme for each Goblin
	// enemy in play.
	engine.RegisterBehavior("02031", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			if g.MainScheme == nil {
				return nil
			}
			n := 0
			for _, mn := range g.Minions {
				if mn.EDef().HasTrait("goblin") {
					n++
				}
			}
			if n > 0 {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: n, Source: t.ID}}
			}
			return nil
		},
	})
}

func attachHighestHP(code string) {
	engine.RegisterBehavior(code, &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			var best engine.EntityID
			bestHP := -1
			for id, v := range g.Villains {
				if v.MaxHP > bestHP && !hasAttachmentCode(g, id, code) {
					best, bestHP = id, v.MaxHP
				}
			}
			for id, mn := range g.Minions {
				if mn.MaxHP > bestHP && !hasAttachmentCode(g, id, code) {
					best, bestHP = id, mn.MaxHP
				}
			}
			if best == "" {
				// no legal target: surge
				g.Delete(t.ID)
				if c, ok := g.DrawEncounter(); ok {
					return []engine.Message{engine.RevealEncounterCard{Player: firstPlayerID(g), Card: c}}
				}
				return nil
			}
			t.Target = best
			if e := g.Entity(best); e != nil {
				g.TLogf("log.attachesTo", t, e)
			}
			return []engine.Message{engine.BoostEnemyAttack{Enemy: best, N: 1}}
		},
	})
}

func hasAttachmentCode(g *engine.Game, target engine.EntityID, code string) bool {
	for _, a := range g.Attachments {
		if a.Target == target && a.Code == code {
			return true
		}
	}
	return false
}

func pumpkinBombs(code string) {
	engine.RegisterBehavior(code, &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for id := range g.Villains {
				t.Target = id
				break
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok {
				return nil
			}
			if m.Enemy != e.EID() && !isAttachedTo(g, e.EID(), m.Enemy) {
				return nil
			}
			g.Delete(e.EID())
			return []engine.Message{engine.DamageEntity{Target: m.Player, Damage: 2, Source: e.EID()}}
		},
	})
}

func isAttachedTo(g *engine.Game, attachment, enemy engine.EntityID) bool {
	if a := g.Attachments[attachment]; a != nil {
		return a.Target == enemy
	}
	return false
}

func firstPlayerID(g *engine.Game) engine.PlayerID {
	for _, p := range g.Players {
		if p.FirstPlayer {
			return p.ID
		}
	}
	if len(g.Players) > 0 {
		return g.Players[0].ID
	}
	return ""
}
