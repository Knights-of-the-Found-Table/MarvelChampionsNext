// Package jubilee registers Jubilee, Shopping Spree, her signatures,
// obligation, and nemesis set.
package jubilee

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerJubilee()
	registerJubileeSignatures()
	registerJubileeObligation()
	registerJubileeNemesis()
}

func distinctIcons(icons []string) int {
	seen := map[string]bool{}
	for _, icon := range icons {
		switch icon {
		case "energy", "mental", "physical", "wild":
			seen[icon] = true
		}
	}
	return len(seen)
}

func handDistinctIcons(p *engine.Player) int {
	var icons []string
	for _, c := range p.Hand {
		icons = append(icons, c.Def().Resources...)
	}
	return distinctIcons(icons)
}

func registerJubilee() {
	engine.RegisterBehavior("47001", &engine.Behavior{HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
		if p.IsHero() {
			return []engine.Ability{{
				Label: "Like, totally! — exhaust Jubilee to generate a wild resource",
				Type:  engine.AbilityAction, HeroOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					// Identity resources are not enumerated by the payment prompt.
					// A one-shot unrestricted discount is resource-equivalent for
					// ordinary card payments, though it cannot satisfy icon riders.
					return []engine.Message{engine.CostDiscountApply{Player: engine.PlayerID(self), Amount: 1}}
				},
			}}
		}
		return []engine.Ability{{
			Label: "Mall Rat — find Shopping Spree", Type: engine.AbilityAction, AlterEgoOnly: true, OncePerTurn: true,
			Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
				pl := g.Player(self)
				if pl == nil {
					return nil
				}
				for _, c := range pl.Deck {
					if c.Code == "47003" {
						return []engine.Message{
							engine.TakeDeckCard{Player: pl.ID, CardID: c.ID},
							engine.PlayCard{Player: pl.ID, Card: c, Paid: engine.CostPaid{}},
						}
					}
				}
				return nil
			},
		}}
	}})
}

func registerJubileeSignatures() {
	engine.RegisterBehavior("47002", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.ChangeForm)
		p := g.Player(e.EOwner())
		if !ok || p == nil || m.Player != p.ID || p.IsHero() {
			return nil
		}
		return []engine.Message{engine.HealEntity{Target: e.EID(), N: 3}}
	}})

	engine.RegisterBehavior("47003", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.SideSchemes[e.EID()]
			if s == nil || s.Threat <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: "Shopping Spree — exhaust your identity to remove 1 threat",
				Type:  engine.AbilityAction, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					scheme := g.SideSchemes[self]
					if scheme == nil || scheme.Threat <= 0 {
						return nil
					}
					p := g.Player(scheme.Owner)
					if p == nil || p.IsHero() || p.Exhausted {
						return nil
					}
					return []engine.Message{engine.ExhaustEntity{ID: p.ID}, engine.ThwartScheme{Scheme: scheme.ID, N: 1, Source: p.ID}}
				},
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, c := range p.Deck {
				if c.Def().HasTrait("item") {
					choices = append(choices, engine.Choice{Label: "Put " + c.Def().Name + " into play", Kind: engine.ChoiceCard, CardCode: c.Code}.
						Msgs(engine.TakeDeckCard{Player: p.ID, CardID: c.ID}, engine.PlayCard{Player: p.ID, Card: c, Paid: engine.CostPaid{}}))
				}
			}
			for _, c := range p.Discard {
				if c.Def().HasTrait("item") {
					choices = append(choices, engine.Choice{Label: "Put " + c.Def().Name + " into play", Kind: engine.ChoiceCard, CardCode: c.Code}.
						Msgs(engine.ReturnDiscardCard{Player: p.ID, CardID: c.ID}, engine.PlayCard{Player: p.ID, Card: c, Paid: engine.CostPaid{}}))
				}
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Shopping Spree — put an Item into play", choices...)}}
		},
	})

	engine.RegisterBehavior("47004", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{THW: 1} },
		React:         eventAccessoryReaction("47004", "thwart", false),
	})
	engine.RegisterBehavior("47005", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{ATK: 1} },
		React:         eventAccessoryReaction("47005", "attack", true),
	})

	engine.RegisterBehavior("47006", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		ev := e.(*engine.EventCard)
		n := distinctIcons(ev.Paid.Icons)
		if n == 0 {
			n = 1
		}
		choices := cardutil.EnemyChoices(g, 0, ev.Owner, func(id engine.EntityID) []engine.Message {
			return []engine.Message{engine.StunEntity{Target: id}, engine.ConfuseEntity{Target: id}}
		})
		n = min(n, len(choices))
		if n == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: ev.Owner, Question: engine.AskN("Blinding Flash — choose enemies", n, choices...)}}
	}})

	firecracker := &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		ev := e.(*engine.EventCard)
		choices := cardutil.EnemyChoices(g, 4, ev.Owner, func(id engine.EntityID) []engine.Message {
			msgs := []engine.Message{engine.DamageEntity{Target: id, Damage: 4, Source: ev.Owner}}
			if distinctIcons(ev.Paid.Icons) >= 2 {
				msgs = append(msgs, engine.StunEntity{Target: id})
			}
			return msgs
		})
		if len(choices) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: ev.Owner, Question: engine.Ask("Firecracker — choose an enemy", choices...)}}
	}}
	engine.RegisterBehavior("47007", firecracker)
	engine.RegisterBehavior("47007a", firecracker)
	engine.RegisterBehavior("47007b", firecracker)
	engine.RegisterBehavior("47007c", firecracker)

	flash := &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		ev := e.(*engine.EventCard)
		var confuse *engine.Question
		if distinctIcons(ev.Paid.Icons) >= 2 {
			enemies := cardutil.EnemyChoices(g, 0, ev.Owner, func(id engine.EntityID) []engine.Message {
				return []engine.Message{engine.ConfuseEntity{Target: id}}
			})
			if len(enemies) > 0 {
				confuse = engine.Ask("Flash of Light — confuse an enemy", enemies...)
			}
		}
		choices := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
			return []engine.Message{engine.ThwartScheme{Scheme: id, N: 3, Source: ev.Owner}}
		})
		if confuse != nil {
			for i := range choices {
				choices[i] = choices[i].WithThen(confuse)
			}
		}
		if len(choices) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: ev.Owner, Question: engine.Ask("Flash of Light — choose a scheme", choices...)}}
	}}
	engine.RegisterBehavior("47008", flash)
	engine.RegisterBehavior("47008a", flash)
	engine.RegisterBehavior("47008b", flash)
	engine.RegisterBehavior("47008c", flash)

	engine.RegisterBehavior("47009", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		ev := e.(*engine.EventCard)
		n := distinctIcons(ev.Paid.Icons)
		if n == 0 {
			n = 1
		}
		additional := cardutil.EnemyChoices(g, 0, ev.Owner, func(id engine.EntityID) []engine.Message {
			return []engine.Message{engine.DamageEntity{Target: id, Damage: 2, Source: ev.Owner}}
		})
		n = min(n, len(additional))
		var follow *engine.Question
		if n > 0 {
			follow = engine.AskN("Grande Finale — choose additional fireworks", n, additional...)
		}
		base := cardutil.EnemyChoices(g, 2, ev.Owner, func(id engine.EntityID) []engine.Message {
			return []engine.Message{engine.DamageEntity{Target: id, Damage: 2, Source: ev.Owner}}
		})
		if follow != nil {
			for i := range base {
				base[i] = base[i].WithThen(follow)
			}
		}
		if len(base) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: ev.Owner, Question: engine.Ask("Grande Finale — choose the first target", base...)}}
	}})
	engine.RegisterBehavior("47010", &engine.Behavior{})
	engine.RegisterBehavior("47010a", &engine.Behavior{})
	engine.RegisterBehavior("47010b", &engine.Behavior{})
	engine.RegisterBehavior("47010c", &engine.Behavior{})
}

func eventAccessoryReaction(code, trait string, damage bool) func(*engine.Game, engine.Entity, engine.Message) []engine.Message {
	return func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		u := g.Upgrades[e.EID()]
		m, ok := msg.(engine.EventPlayed)
		if !ok || u == nil || u.Exhausted || m.Player != u.Owner || !m.Card.Def().HasTrait(trait) {
			return nil
		}
		// EventPlayed omits CostPaid, so the accessory supplies the minimum
		// one-point rider rather than counting distinct payment types.
		var choices []engine.Choice
		if damage {
			choices = cardutil.EnemyChoices(g, 1, u.Owner, func(id engine.EntityID) []engine.Message {
				return []engine.Message{engine.ExhaustEntity{ID: u.ID}, engine.DamageEntity{Target: id, Damage: 1, Source: u.Owner}}
			})
		} else {
			choices = cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
				return []engine.Message{engine.ExhaustEntity{ID: u.ID}, engine.ThwartScheme{Scheme: id, N: 1, Source: u.Owner}}
			})
		}
		if len(choices) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: u.Owner, Question: engine.Ask(fmt.Sprintf("%s — choose a target", e.EDef().Name), choices...)}}
	}
}

func registerJubileeObligation() {
	engine.RegisterBehavior("47023", &engine.Behavior{ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
		// Grounded should remain in a player play area until a Jubilee event
		// removes it. No persistent-obligation zone exists, so force the form
		// change and discard it immediately; the extra flip cost is omitted.
		msgs := []engine.Message{}
		if p.IsHero() {
			msgs = append(msgs, engine.ChangeForm{Player: p.ID})
		}
		msgs = append(msgs, engine.ObligationResolve{Player: p.ID, Card: card})
		return msgs
	}})
}

func registerJubileeNemesis() {
	// Nanny's Lost Child effect requires converting an ally into a minion,
	// a zone/type transition the engine cannot currently represent.
	engine.RegisterBehavior("47024", &engine.Behavior{})
	engine.RegisterBehavior("47025", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		if len(g.Players) == 0 {
			return nil
		}
		n := handDistinctIcons(g.Players[0])
		if n == 0 {
			return nil
		}
		return []engine.Message{engine.SchemeThreat{Scheme: e.EID(), N: n, Source: e.EID()}}
	}})
	engine.RegisterBehavior("47026", &engine.Behavior{OnAttach: func(g *engine.Game, a *engine.Attachment, target engine.EntityID) []engine.Message {
		if mn := g.Minions[target]; mn != nil {
			mn.MaxHP += 3
		}
		return nil
	}})
	engine.RegisterBehavior("47027", &engine.Behavior{})
}
