// Package spectrum registers Spectrum, her three Energy forms, signature cards,
// obligation, and nemesis set.
package spectrum

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

const (
	gammaSignal  = -21002
	photonSignal = -21003
	pulsarSignal = -21004
)

var energyForms = []string{"21002", "21003", "21004"}

func init() {
	registerIdentity()
	registerForms()
	registerSignatures()
	registerObligation()
	registerNemesis()
}

func formSignal(code string) int {
	switch code {
	case "21002":
		return gammaSignal
	case "21003":
		return photonSignal
	case "21004":
		return pulsarSignal
	}
	return 0
}

func signalForm(n int) string {
	switch n {
	case gammaSignal:
		return "21002"
	case photonSignal:
		return "21003"
	case pulsarSignal:
		return "21004"
	}
	return ""
}

func ownedForm(g *engine.Game, p *engine.Player, code string) *engine.Upgrade {
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

func activeForm(g *engine.Game, p *engine.Player) string {
	for _, code := range energyForms {
		if u := ownedForm(g, p, code); u != nil && u.Counters > 0 {
			return code
		}
	}
	return ""
}

func formChoices(g *engine.Game, p *engine.Player, prompt string, includeCurrent bool) []engine.Message {
	var choices []engine.Choice
	current := activeForm(g, p)
	for _, code := range energyForms {
		if !includeCurrent && code == current {
			continue
		}
		choices = append(choices, engine.Choice{Label: engine.S(engine.DB.MustLookup(code).Name), Kind: engine.ChoiceCard, CardCode: code}.
			Msgs(engine.AddEntityCounter{ID: p.ID, N: formSignal(code)}))
	}
	if len(choices) == 0 {
		return nil
	}
	return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.S(prompt), choices...)}}
}

func changeEnergyForm(g *engine.Game, p *engine.Player, code string) []engine.Message {
	if p == nil || formSignal(code) == 0 {
		return nil
	}
	wasActive := activeForm(g, p) == code
	for _, c := range energyForms {
		if u := ownedForm(g, p, c); u != nil {
			u.Counters = 0
		}
	}
	if u := ownedForm(g, p, code); u != nil {
		u.Counters = 1
	}
	g.TLogf("c.changesToEnergyForm", p.Name, engine.DB.MustLookup(code).Name)
	var msgs []engine.Message
	switch code {
	case "21002":
		choices := cardutil.EnemyChoices(g, 1, p.ID, func(id engine.EntityID) []engine.Message {
			return []engine.Message{engine.DamageEntity{Target: id, Damage: 1, Source: p.ID}}
		})
		if len(choices) > 0 {
			msgs = append(msgs, engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.gammaDeal1Damage"), choices...)})
		}
	case "21003":
		choices := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
			return []engine.Message{engine.ThwartScheme{Scheme: id, N: 1, Source: p.ID}}
		})
		if len(choices) > 0 {
			msgs = append(msgs, engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.photonRemove1Threat"), choices...)})
		}
	case "21004":
		msgs = append(msgs, engine.HealEntity{Target: p.ID, N: 1})
	}
	_ = wasActive // callers inspect the prior form for printed kicker approximations.
	return msgs
}

func registerIdentity() {
	engine.RegisterBehavior("21001", &engine.Behavior{
		HeroSetup: func(g *engine.Game, p *engine.Player) []engine.Message {
			for _, code := range energyForms {
				if ownedForm(g, p, code) != nil {
					continue
				}
				u := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: code, Owner: p.ID}
				g.Upgrades[u.ID] = u
				p.Upgrades = append(p.Upgrades, u.ID)
			}
			return formChoices(g, p, "Energy Transformation — choose a starting Energy form", true)
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			p := g.Player(e.EID())
			switch m := msg.(type) {
			case engine.AddEntityCounter:
				if p != nil && m.ID == p.ID {
					if code := signalForm(m.N); code != "" {
						return changeEnergyForm(g, p, code)
					}
				}
			case engine.ChangeForm:
				// ChangeForm is broadcast before the side flips. A hero-to-alter-ego
				// change turns every Energy form facedown; entering hero asks for one.
				if p == nil || m.Player != p.ID {
					return nil
				}
				if p.IsHero() {
					for _, code := range energyForms {
						if u := ownedForm(g, p, code); u != nil {
							u.Counters = 0
						}
					}
					return nil
				}
				return formChoices(g, p, "Energy Transformation — choose an Energy form", true)
			}
			return nil
		},
	})
}

func registerForms() {
	engine.RegisterBehavior("21002", &engine.Behavior{IdentityStatsG: func(g *engine.Game, p *engine.Player, u *engine.Upgrade) engine.StatBonus {
		if u.Counters > 0 {
			return engine.StatBonus{ATK: 2}
		}
		return engine.StatBonus{}
	}})
	engine.RegisterBehavior("21003", &engine.Behavior{IdentityStatsG: func(g *engine.Game, p *engine.Player, u *engine.Upgrade) engine.StatBonus {
		if u.Counters > 0 {
			return engine.StatBonus{THW: 2}
		}
		return engine.StatBonus{}
	}})
	engine.RegisterBehavior("21004", &engine.Behavior{IdentityStatsG: func(g *engine.Game, p *engine.Player, u *engine.Upgrade) engine.StatBonus {
		if u.Counters > 0 {
			return engine.StatBonus{DEF: 2}
		}
		return engine.StatBonus{}
	}})
}

func registerSignatures() {
	engine.RegisterBehavior("21005", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		return formChoices(g, g.Player(e.EOwner()), "Blue Marvel — change Energy forms", false)
	}})
	// Energy Duplication cannot dynamically copy a printed icon; wild keeps
	// the same payment flexibility while still requiring the upgrade to exhaust.
	engine.RegisterBehavior("21006", &engine.Behavior{Resource: &engine.ResourceAbility{Icon: "wild", HeroOnly: true}})
	engine.RegisterBehavior("21007", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		already := activeForm(g, p) == "21002"
		msgs := changeEnergyForm(g, p, "21002")
		choices := cardutil.EnemyChoices(g, 7, p.ID, func(id engine.EntityID) []engine.Message {
			// Overkill is not represented on event damage; the 7 damage is exact.
			return []engine.Message{engine.DamageEntity{Target: id, Damage: 7, Source: p.ID}}
		})
		if len(choices) > 0 {
			msgs = append(msgs, engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.gammaBlastDeal7Damage"), choices...)})
		}
		_ = already
		return msgs
	}})
	engine.RegisterBehavior("21008", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		msgs := changeEnergyForm(g, p, "21003")
		choices := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
			// Crisis bypass is not expressible; the selected scheme receives the
			// full printed 4-threat removal.
			return []engine.Message{engine.ThwartScheme{Scheme: id, N: 4, Source: p.ID}}
		})
		if len(choices) > 0 {
			msgs = append(msgs, engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.photonSpeedRemove4Threat"), choices...)})
		}
		return msgs
	}})
	engine.RegisterBehavior("21009", &engine.Behavior{DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
		already := activeForm(g, p) == "21004"
		msgs := append(changeEnergyForm(g, p, "21004"), engine.ReadyEntity{ID: p.ID})
		if already {
			// Retaliate lasts through the phase via the existing stat bonus field.
			p.BonusDEF += 0
			msgs = append(msgs, engine.DamageEntity{Target: against, Damage: 1, Source: p.ID})
		}
		return engine.Defends{Defender: p.ID, Against: against}, msgs, true
	}})
	engine.RegisterBehavior("21010", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		msgs := formChoices(g, p, "Speed of Light — change Energy forms", false)
		return append(msgs, engine.DrawCards{Player: p.ID, N: 1})
	}})
}

func registerObligation() {
	engine.RegisterBehavior("21026", &engine.Behavior{ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
		if p.IsHero() {
			return []engine.Message{engine.ChangeForm{Player: p.ID}, engine.ExhaustEntity{ID: p.ID}, engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}}
		}
		return []engine.Message{engine.ExhaustEntity{ID: p.ID}, engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}}
	}})
}

func damageFriendly(g *engine.Game, p *engine.Player) []engine.Message {
	if p == nil {
		return nil
	}
	msgs := []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 1}}
	for _, id := range p.Allies {
		msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1})
	}
	return msgs
}

func registerNemesis() {
	engine.RegisterBehavior("21027", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionActivates)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			return damageFriendly(g, g.Player(m.Player))
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if len(g.Players) == 0 {
				return nil
			}
			return damageFriendly(g, g.Players[0])
		},
	})
	engine.RegisterBehavior("21028", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.SchemeDefeated)
		if !ok || m.Scheme != e.EID() {
			return nil
		}
		var msgs []engine.Message
		for _, p := range g.Players {
			msgs = append(msgs, damageFriendly(g, p)...)
		}
		return msgs
	}})
	engine.RegisterBehavior("21029", &engine.Behavior{
		OnAttach: func(g *engine.Game, a *engine.Attachment, target engine.EntityID) []engine.Message {
			a.Target = target
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			a := e.(*engine.Attachment)
			m, ok := msg.(engine.PlayerTurnEnd)
			if !ok || engine.EntityID(m.Player) != a.Target {
				return nil
			}
			return []engine.Message{engine.DamageEntity{Target: a.Target, Damage: 1, Source: a.ID}}
		},
	})
	engine.RegisterBehavior("21030", &engine.Behavior{ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		g.Delete(t.ID)
		if p.IsHero() {
			return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 2, Source: t.ID}}
		}
		if g.MainScheme != nil {
			return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: t.ID}}
		}
		return nil
	}})
}
