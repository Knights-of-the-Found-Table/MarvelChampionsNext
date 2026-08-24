// Package iceman registers Iceman, his signature cards, obligation, and nemesis set.
package iceman

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerIceman()
	registerIcemanSignatures()
	registerIcemanObligation()
	registerIcemanNemesis()
	registerIcemanExtras()
}

func ownedUpgrade(g *engine.Game, p *engine.Player, code string) *engine.Upgrade {
	if p == nil {
		return nil
	}
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil && u.Code == code {
			return u
		}
	}
	return nil
}

func frostbiteCount(g *engine.Game, p *engine.Player) int {
	n := 0
	for _, u := range g.Upgrades {
		if u != nil && u.Owner == p.ID && u.Code == "46002" {
			n++
		}
	}
	return n
}

func frostbiteAttached(g *engine.Game, target engine.EntityID) bool {
	for _, u := range g.Upgrades {
		if u != nil && u.Code == "46002" && u.AttachTo == target {
			return true
		}
	}
	return false
}

func attachFrostbite(g *engine.Game, p *engine.Player, target engine.EntityID) []engine.Message {
	if p == nil || target == "" || frostbiteCount(g, p) >= 6 {
		return nil
	}
	// Frostbite begins outside all normal player zones. The engine has no
	// general set-aside-upgrade message, so create the entity here and let
	// AttachUpgrade perform the attachment bookkeeping.
	u := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "46002", Owner: p.ID}
	g.Upgrades[u.ID] = u
	p.Upgrades = append(p.Upgrades, u.ID)
	msgs := []engine.Message{engine.AttachUpgrade{ID: u.ID, Target: target}}
	if perception := ownedUpgrade(g, p, "46005"); perception != nil && !perception.Exhausted {
		msgs = append(msgs, engine.ExhaustEntity{ID: perception.ID}, engine.DrawCards{Player: p.ID, N: 1})
		if len(p.Deck) > 0 && p.Deck[0].Def().HasTrait("ice") {
			msgs = append(msgs, engine.ReadyEntity{ID: p.ID})
		}
	}
	return msgs
}

func registerIceman() {
	engine.RegisterBehavior("46001", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			p := g.Player(e.EID())
			if p == nil {
				return nil
			}
			switch m := msg.(type) {
			case engine.BasicAttack:
				if p.IsHero() && m.Player == p.ID {
					return attachFrostbite(g, p, m.Target)
				}
			case engine.Defends:
				if p.IsHero() && m.Defender == p.ID && !m.Undefended && m.Via == "" {
					return attachFrostbite(g, p, m.Against)
				}
			case engine.AddEntityCounter:
				// N=0 on an enemy is the package-local, serialization-safe
				// signal used by signature-card choices to attach Frostbite only
				// after the player actually selects that choice.
				if m.N == 0 && (g.Villains[m.ID] != nil || g.Minions[m.ID] != nil) {
					return attachFrostbite(g, p, m.ID)
				}
			case engine.ChangeForm:
				if m.Player != p.ID || p.IsHero() {
					return nil
				}
				n := frostbiteCount(g, p)
				var msgs []engine.Message
				for i := len(p.Discard) - 1; i >= 0 && len(msgs) < n; i-- {
					c := p.Discard[i]
					if c.Def().HasTrait("ice") {
						msgs = append(msgs, engine.ShuffleIntoDeck{Player: p.ID, CardID: c.ID})
					}
				}
				return msgs
			}
			return nil
		},
	})
}

func registerIcemanSignatures() {
	engine.RegisterBehavior("46002", &engine.Behavior{
		AttachedEnemyAttackMod: -1,
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			activated := false
			switch m := msg.(type) {
			case engine.VillainActivates:
				activated = m.VillainID == u.AttachTo
			case engine.MinionActivates:
				activated = m.MinionID == u.AttachTo
			case engine.MinionDefeated:
				activated = m.MinionID == u.AttachTo
			case engine.VillainDefeated:
				activated = m.VillainID == u.AttachTo
			}
			if activated {
				// Permanent Frostbite returns to the conceptual set-aside pool,
				// represented by deleting it without adding it to discard.
				g.Delete(u.ID)
			}
			return nil
		},
	})

	// Snow Clone's consequential reduction lacks a target-aware hook.
	engine.RegisterBehavior("46003", &engine.Behavior{})
	engine.RegisterBehavior("46004", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if p := g.Player(e.EOwner()); p != nil {
				p.MaxHP += 3
			}
			return nil
		},
		Resource: &engine.ResourceAbility{Icon: "wild", HeroOnly: true},
	})
	// Cryokinetic Perception is folded into attachFrostbite so it responds
	// to every supported Freeze resolution.
	engine.RegisterBehavior("46005", &engine.Behavior{})
	engine.RegisterBehavior("46006", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.GrantTrait{Target: e.EOwner(), Trait: "aerial"}}
		},
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			if p.IsHero() {
				return engine.StatBonus{ATK: 1, THW: 1, DEF: 1}
			}
			return engine.StatBonus{}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			u := g.Upgrades[e.EID()]
			m, ok := msg.(engine.ChangeForm)
			p := g.Player(e.EOwner())
			if !ok || u == nil || p == nil || m.Player != p.ID || p.IsHero() {
				return nil
			}
			// DiscardControlled cannot preserve the new card ID for a following
			// shuffle, so perform the set-aside-free zone move directly.
			g.Delete(u.ID)
			p.Deck = append(p.Deck, engine.Card{ID: g.NextCardID(), Code: u.Code, Owner: p.ID})
			return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
		},
	})

	engine.RegisterBehavior("46007", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			choices := cardutil.EnemyChoices(g, 0, p.ID, func(id engine.EntityID) []engine.Message {
				return []engine.Message{engine.AttachUpgrade{ID: e.EID(), Target: id}, engine.StunEntity{Target: id}}
			})
			if len(choices) == 0 {
				return nil
			}
			// Stun is applied at attachment time as an approximation for
			// cancelling the next activation; no activation-cancel hook exists.
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Frozen Solid — choose an enemy", choices...)}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			activated := false
			switch m := msg.(type) {
			case engine.VillainActivates:
				activated = m.VillainID == u.AttachTo
			case engine.MinionActivates:
				activated = m.MinionID == u.AttachTo
			}
			if !activated {
				return nil
			}
			p := g.Player(u.Owner)
			msgs := []engine.Message{engine.DiscardControlled{Player: p.ID, ID: u.ID}}
			msgs = append(msgs, attachFrostbite(g, p, u.AttachTo)...)
			return msgs
		},
	})

	engine.RegisterBehavior("46008", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		s := g.Supports[e.EID()]
		m, ok := msg.(engine.DamageEntity)
		if !ok || s == nil || m.Target != s.Owner || m.Damage <= 0 {
			return nil
		}
		if _, villain := g.Villains[m.Source]; !villain {
			if _, minion := g.Minions[m.Source]; !minion {
				return nil
			}
		}
		msgs := []engine.Message{engine.HealEntity{Target: s.Owner, N: m.Damage}, engine.AddEntityCounter{ID: s.ID, N: m.Damage}}
		if s.Counters+m.Damage >= 8 {
			msgs = append(msgs, engine.DiscardControlled{Player: s.Owner, ID: s.ID})
			msgs = append(msgs, attachFrostbite(g, g.Player(s.Owner), m.Source)...)
		}
		return msgs
	}})

	engine.RegisterBehavior("46009", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		var choices []engine.Choice
		for _, id := range cardutil.SortedEnemyIDs(g) {
			target := g.Entity(id)
			if target == nil {
				continue
			}
			msgs := []engine.Message{}
			damage := 6
			if !frostbiteAttached(g, id) {
				damage = 4
				msgs = append(msgs, engine.AddEntityCounter{ID: id, N: 0})
			}
			msgs = append([]engine.Message{engine.DamageEntity{Target: id, Damage: damage, Source: p.ID}}, msgs...)
			choices = append(choices, engine.Choice{Label: target.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id, CardCode: target.ECode()}.Msgs(msgs...))
		}
		if len(choices) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Arctic Attack — choose an enemy", choices...)}}
	}})

	engine.RegisterBehavior("46010", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		var choices []engine.Choice
		for _, targetPlayer := range g.Players {
			var targets []engine.EntityID
			for id := range g.Villains {
				targets = append(targets, id)
			}
			for id, mn := range g.Minions {
				if mn != nil && mn.EngagedWith == targetPlayer.ID {
					targets = append(targets, id)
				}
			}
			var msgs []engine.Message
			for _, id := range targets {
				msgs = append(msgs, engine.AddEntityCounter{ID: id, N: 0})
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 3, Source: p.ID})
			}
			choices = append(choices, engine.Choice{Label: targetPlayer.Name, Kind: engine.ChoiceTarget, SourceID: targetPlayer.ID}.Msgs(msgs...))
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Ice Blast — choose a player", choices...)}}
	}})

	engine.RegisterBehavior("46011", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		enemies := cardutil.EnemyChoices(g, 0, p.ID, func(id engine.EntityID) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: id, N: 0}}
		})
		var follow *engine.Question
		if len(enemies) > 0 {
			follow = engine.Ask("Chill Out! — attach Frostbite", enemies...)
		}
		choices := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
			return []engine.Message{engine.ThwartScheme{Scheme: id, N: 3, Source: p.ID}}
		})
		if follow != nil {
			for i := range choices {
				choices[i] = choices[i].WithThen(follow)
			}
		}
		if len(choices) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Chill Out! — choose a scheme", choices...)}}
	}})
}

func resourceIcons(cards engine.CardList) int {
	n := 0
	for _, c := range cards {
		n += len(c.Def().Resources)
	}
	return n
}

func registerIcemanObligation() {
	engine.RegisterBehavior("46024", &engine.Behavior{ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
		// The obligation is persistent in the tabletop game, but the engine
		// has no player-obligation play area. Resolve one representative
		// Frostbite penalty immediately, then discard it.
		return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 1, Source: p.ID}, engine.ObligationResolve{Player: p.ID, Card: card}}
	}})
}

func registerIcemanNemesis() {
	engine.RegisterBehavior("46025", &engine.Behavior{})
	engine.RegisterBehavior("46026", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.SchemeDefeated)
		if !ok || m.Scheme != e.EID() {
			return nil
		}
		p := g.Player(e.EOwner())
		if p == nil && len(g.Players) > 0 {
			p = g.Players[0]
		}
		if p == nil {
			return nil
		}
		n := min(3, len(p.Deck))
		damage := resourceIcons(p.Deck[:n])
		return []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: n}, engine.DamageEntity{Target: p.ID, Damage: damage, Source: e.EID()}}
	}})
	engine.RegisterBehavior("46027", &engine.Behavior{})
	engine.RegisterBehavior("46028", &engine.Behavior{ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		g.Delete(t.ID)
		n := 2
		for _, mn := range g.Minions {
			if mn != nil && mn.Code == "46025" {
				n = 3
				break
			}
		}
		n = min(n, len(p.Deck))
		damage := resourceIcons(p.Deck[:n])
		return []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: n}, engine.DamageEntity{Target: p.ID, Damage: damage, Source: t.ID}}
	}, Boost: func(g *engine.Game, card engine.Card) []engine.Message {
		if len(g.Players) == 0 {
			return nil
		}
		p := g.Players[0]
		return []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: 1}}
	}})
}
