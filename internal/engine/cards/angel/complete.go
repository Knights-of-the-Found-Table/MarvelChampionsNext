package angel

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func registerAngelExtras() {
	// 42011 Elixir: heal after acting.
	engine.RegisterBehavior("42011", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "X-Force") || g.EntityHasTrait(p.ID, "X-Men")
		},
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
			p := g.Player(a.Owner)
			if p == nil {
				return nil
			}
			for _, id := range p.Allies {
				if id != e.EID() {
					return []engine.Message{engine.HealEntity{Target: id, N: 1}}
				}
			}
			return []engine.Message{engine.HealEntity{Target: p.ID, N: 1}}
		},
	})

	// 42012 Siryn: stun a minion after attacking.
	engine.RegisterBehavior("42012", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyAttackWindow)
			if !ok || m.Ally != e.EID() {
				return nil
			}
			if mn := g.Minions[m.Target]; mn != nil {
				return []engine.Message{engine.StunEntity{Target: mn.ID}}
			}
			for _, mn := range g.Minions {
				if mn != nil {
					return []engine.Message{engine.StunEntity{Target: mn.ID}}
				}
			}
			return nil
		},
	})

	// 42013 Warpath: post-defense event discount is approximated away.
	engine.RegisterBehavior("42013", &engine.Behavior{})

	// 42014 Aerial Intervention: prevent 3 by exhausting an Aerial
	// character (defense substitute).
	engine.RegisterBehavior("42014", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			hasAerial := g.EntityHasTrait(p.ID, "Aerial")
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && a.EDef().HasTrait("Aerial") && !a.Exhausted {
					hasAerial = true
				}
			}
			if !hasAerial {
				return engine.Defends{}, nil, false
			}
			d := engine.Defends{Defender: p.ID, Against: against, Undefended: true, ExtraPrevent: 3}
			return d, nil, true
		},
	})

	// 42015 Ever Vigilant: ready + 2 threat.
	engine.RegisterBehavior("42015", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "Aerial")
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || g.MainScheme == nil {
				return nil
			}
			return []engine.Message{
				engine.ReadyEntity{ID: p.ID},
				engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 2, Source: p.ID},
			}
		},
	})

	// 42016 Taunt: villain attacks you; draw 3.
	engine.RegisterBehavior("42016", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			for id := range g.Villains {
				return []engine.Message{
					engine.AskAttack{Enemy: id, Player: p.ID},
					engine.DrawCards{Player: p.ID, N: 3},
				}
			}
			return []engine.Message{engine.DrawCards{Player: p.ID, N: 3}}
		},
	})

	// 42019 Containment Strategy: shed threat on defense.
	engine.RegisterBehavior("42019", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			u := g.Upgrades[e.EID()]
			p := g.Player(u.Owner)
			if p == nil || len(g.Schemes()) == 0 {
				return nil
			}
			var choices []engine.Choice
			for _, id := range g.Schemes() {
				s := g.Entity(id)
				choices = append(choices, engine.Choice{
					ID: "sch-" + id.String(), Label: engine.S(s.EDef().Name), Kind: engine.ChoiceTarget,
				}.Msgs(engine.AttachUpgrade{ID: u.ID, Target: id}))
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.containmentStrategyAttachTo"), choices...)}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowDefended)
			if !ok {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if u == nil || u.AttachTo == "" {
				return nil
			}
			n := 1
			if w.DamageTaken == 0 {
				n = 2
			}
			return []engine.Message{engine.ThwartScheme{Scheme: u.AttachTo, N: n, Source: u.Owner}}
		},
	})

	// 42020 Cannonball: consequential reduction not modeled per-icon;
	// flat -all (allies take 0 from his consequential).
	engine.RegisterBehavior("42020", &engine.Behavior{})

	// 42021 Soaring Hearts: fetch an identity event + ready the duo.
	soaring := func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		if p == nil {
			return nil
		}
		heroSet := ""
		if d, ok := engine.DB.Lookup(p.HeroCode); ok {
			heroSet = d.CardSet
		}
		msgs := []engine.Message{engine.ReadyEntity{ID: p.ID}}
		for _, c := range append(engine.CardList{}, p.Discard...) {
			if c.Def().Type == "event" && c.Def().CardSet == heroSet {
				if _, ok := p.Discard.Remove(c.ID); ok {
					p.Hand = append(p.Hand, c)
					g.TLogf("c.soaringHeartsRecovers", c)
				}
				break
			}
		}
		for _, id := range p.Allies {
			if a := g.Allies[id]; a != nil {
				msgs = append(msgs, engine.ReadyEntity{ID: id})
			}
		}
		return msgs
	}
	engine.RegisterBehavior("42021", &engine.Behavior{OnPlay: soaring})
	engine.RegisterBehavior("41020", &engine.Behavior{OnPlay: soaring})

	// 42022 The Power of Flight: engine powerOfBonus.
	engine.RegisterBehavior("42022", &engine.Behavior{})

	// 42023 Soaring Acrobatics: +1 to Aerial basic powers.
	engine.RegisterBehavior("42023", &engine.Behavior{})

	// 42026 Hook, Line, and Sinker: indirect exhaust rider not modeled.
	engine.RegisterBehavior("42026", &engine.Behavior{})

	// 42029 Bombs Away: blast the villain + engaged minions.
	engine.RegisterBehavior("42029", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			// Exhaust the first ready Aerial character.
			var src engine.EntityID
			if g.EntityHasTrait(p.ID, "Aerial") && !p.Exhausted {
				src = p.ID
			} else {
				for _, id := range p.Allies {
					if a := g.Allies[id]; a != nil && a.EDef().HasTrait("Aerial") && !a.Exhausted {
						src = id
						break
					}
				}
			}
			if src == "" {
				return nil
			}
			var msgs []engine.Message
			msgs = append(msgs, engine.ExhaustEntity{ID: src})
			for id := range g.Villains {
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 3, Source: p.ID})
				break
			}
			for _, id := range cardutil.SortedIDs(g.Minions) {
				if mn := g.Minions[id]; mn != nil && mn.EngagedWith == p.ID {
					msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 3, Source: p.ID})
				}
			}
			return msgs
		},
	})

	// 42030 Eyes in the Sky: minion reveal cancel not modeled.
	engine.RegisterBehavior("42030", &engine.Behavior{})

	// 42031 Flying Formation: ready up to 3 Aerial characters.
	engine.RegisterBehavior("42031", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var msgs []engine.Message
			n := 0
			if g.EntityHasTrait(p.ID, "Aerial") && p.Exhausted {
				msgs = append(msgs, engine.ReadyEntity{ID: p.ID})
				n++
			}
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && a.EDef().HasTrait("Aerial") && a.Exhausted && n < 3 {
					msgs = append(msgs, engine.ReadyEntity{ID: id})
					n++
				}
			}
			return msgs
		},
	})

	// 42032 X-Force Recruit: +1 HP + X-Force trait.
	engine.RegisterBehavior("42032", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "X-Force")
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			u := g.Upgrades[e.EID()]
			p := g.Player(u.Owner)
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a == nil {
					continue
				}
				choices = append(choices, engine.Choice{
					ID: "ally-" + id.String(), Label: engine.S(a.EDef().Name), Kind: engine.ChoiceTarget,
				}.Msgs(engine.AttachUpgrade{ID: u.ID, Target: id, MaxHP: 1, GrantTrait: "X-Force"}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.xForceRecruitAttachTo"), choices...)}}
		},
	})
}
