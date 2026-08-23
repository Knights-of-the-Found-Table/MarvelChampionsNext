// Package aoa registers the Bishop and Magik heroes from Age of Apocalypse.
package aoa

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerBishop()
	registerBishopSignatures()
	registerBishopObligation()
	registerBishopNemesis()
}

func resourceCards(cards engine.CardList) engine.CardList {
	var out engine.CardList
	for _, c := range cards {
		if c.Def().Type == "resource" {
			out = append(out, c)
		}
	}
	return out
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

func registerBishop() {
	engine.RegisterBehavior("45001", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			p := g.Player(e.EID())
			if p == nil {
				return nil
			}
			switch m := msg.(type) {
			case engine.WindowDefended:
				// WindowDefended is the engine's source-aware post-attack damage
				// window, and therefore avoids triggering Energy Absorption for
				// consequential or treachery damage.
				if !p.IsHero() || m.Defender != p.ID || m.DamageTaken <= 0 {
					return nil
				}
				n := min(m.DamageTaken, len(p.Deck))
				if n == 0 {
					return nil
				}
				discarded := append(engine.CardList(nil), p.Deck[:n]...)
				msgs := []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: n}}
				for _, c := range resourceCards(discarded) {
					msgs = append(msgs, engine.ReturnDiscardCard{Player: p.ID, CardID: c.ID})
				}
				// Bishop's Uniform responds to the same resolution. The engine has
				// no named-ability-resolved message, so its optional response is
				// folded into Energy Absorption and resolves automatically.
				if u := ownedUpgrade(g, p, "45005"); u != nil && !u.Exhausted {
					heal := len(resourceCards(p.Hand)) + len(resourceCards(discarded))
					if heal > 0 {
						msgs = append(msgs, engine.ExhaustEntity{ID: u.ID}, engine.HealEntity{Target: p.ID, N: heal})
					}
				}
				return msgs
			case engine.ChangeForm:
				if m.Player != p.ID || p.IsHero() {
					return nil
				}
				var choices []engine.Choice
				for _, c := range p.Discard {
					if c.Def().HasTrait("temporal") {
						choices = append(choices, engine.Choice{Label: "Return " + c.Def().Name, Kind: engine.ChoiceCard, CardCode: c.Code}.
							Msgs(engine.ReturnDiscardCard{Player: p.ID, CardID: c.ID}))
					}
				}
				if len(choices) == 0 {
					return nil
				}
				return []engine.Message{engine.AskQuestion{Player: p.ID,
					Question: engine.Ask("Temporally Displaced — return a Temporal card", choices...)}}
			}
			return nil
		},
	})
}

func registerBishopSignatures() {
	registerBishopAlly := func(code, icon string) {
		engine.RegisterBehavior(code, &engine.Behavior{Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Discard a resource card → ready " + e.EDef().Name,
				Type:  engine.AbilityAction, OncePerTurn: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					a := g.Allies[self]
					if a == nil {
						return nil
					}
					p := g.Player(a.Owner)
					var choices []engine.Choice
					for _, c := range resourceCards(p.Hand) {
						msgs := []engine.Message{engine.DiscardCards{Player: p.ID, Cards: engine.CardList{c}}, engine.ReadyEntity{ID: a.ID}}
						for _, r := range c.Def().Resources {
							if r == icon || r == "wild" {
								msgs = append(msgs, engine.HealEntity{Target: a.ID, N: 1})
								break
							}
						}
						choices = append(choices, engine.Choice{Label: "Discard " + c.Def().Name, Kind: engine.ChoiceCard, CardCode: c.Code}.Msgs(msgs...))
					}
					if len(choices) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Ready "+a.EDef().Name, choices...)}}
				},
			}}
		}})
	}
	registerBishopAlly("45002", "physical")
	registerBishopAlly("45003", "energy")

	engine.RegisterBehavior("45004", &engine.Behavior{Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
		return []engine.Ability{{Label: "Bishop's Rifle — damage for each resource card in hand", Type: engine.AbilityAction, HeroOnly: true, Exhaust: true,
			Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
				u := g.Upgrades[self]
				if u == nil {
					return nil
				}
				n := len(resourceCards(g.Player(u.Owner).Hand))
				choices := cardutil.EnemyChoices(g, n, u.Owner, func(id engine.EntityID) []engine.Message {
					return []engine.Message{engine.DamageEntity{Target: id, Damage: n, Source: u.Owner}}
				})
				if len(choices) == 0 {
					return nil
				}
				return []engine.Message{engine.AskQuestion{Player: u.Owner, Question: engine.Ask("Bishop's Rifle — choose an enemy", choices...)}}
			}}}
	}})

	// 45005 Bishop's Uniform is resolved by the identity hook above.
	engine.RegisterBehavior("45005", &engine.Behavior{})

	engine.RegisterBehavior("45006", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{Label: "Super-Charged — discard a resource card", Type: engine.AbilityAction,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					if u == nil {
						return nil
					}
					p := g.Player(u.Owner)
					var choices []engine.Choice
					for _, c := range resourceCards(p.Hand) {
						n := len(c.Def().Resources)
						choices = append(choices, engine.Choice{Label: fmt.Sprintf("Discard %s — %d charge", c.Def().Name, n), Kind: engine.ChoiceCard, CardCode: c.Code}.
							Msgs(engine.DiscardCards{Player: p.ID, Cards: engine.CardList{c}}, engine.AddEntityCounter{ID: u.ID, N: n}))
					}
					if len(choices) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Super-Charged — choose a resource card", choices...)}}
				}}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			u := g.Upgrades[e.EID()]
			m, ok := msg.(engine.BasicAttack)
			if !ok || u == nil || m.Player != u.Owner || u.Counters <= 0 {
				return nil
			}
			// The interrupt is automatic because the engine has no optional
			// pre-basic-power interrupt menu.
			bonus := min(8, 2*u.Counters)
			return []engine.Message{engine.ApplyStatBonus{Target: u.Owner, ATK: bonus}, engine.DiscardControlled{Player: u.Owner, ID: u.ID}}
		},
	})

	engine.RegisterBehavior("45007", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		ev := e.(*engine.EventCard)
		choices := cardutil.EnemyChoices(g, 6, ev.Owner, func(id engine.EntityID) []engine.Message {
			msgs := []engine.Message{engine.DamageEntity{Target: id, Damage: 6, Source: ev.Owner}}
			// Payment card types are not retained after payment; any card payment
			// is the closest stable approximation to "paid with a resource card."
			if len(ev.Paid.CardIDs) > 0 {
				msgs = append(msgs, engine.ReadyEntity{ID: ev.Owner})
			}
			return msgs
		})
		if len(choices) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: ev.Owner, Question: engine.Ask("Concussive Blast — choose an enemy", choices...)}}
	}})

	engine.RegisterBehavior("45008", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		ev := e.(*engine.EventCard)
		choices := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
			msgs := []engine.Message{engine.ThwartScheme{Scheme: id, N: 3, Source: ev.Owner}}
			if len(ev.Paid.CardIDs) > 0 {
				msgs = append(msgs, engine.DrawCards{Player: ev.Owner, N: 1})
			}
			return msgs
		})
		if len(choices) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: ev.Owner, Question: engine.Ask("Command Authority — choose a scheme", choices...)}}
	}})

	engine.RegisterBehavior("45009", &engine.Behavior{DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
		var msgs []engine.Message
		for _, c := range resourceCards(p.Discard) {
			msgs = append(msgs, engine.ShuffleIntoDeck{Player: p.ID, CardID: c.ID})
		}
		// DefBonus 99 caps ordinary attacks at zero rather than the printed
		// three because the incoming attack value is not exposed here.
		return engine.Defends{Defender: p.ID, Against: against, DefBonus: 99}, msgs, true
	}})
	engine.RegisterBehavior("45010", &engine.Behavior{})
}

func registerBishopObligation() {
	engine.RegisterBehavior("45025", &engine.Behavior{ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
		var discard engine.CardList
		for _, c := range resourceCards(p.Hand) {
			discard = append(discard, c)
		}
		penalty := []engine.Message{}
		if len(discard) > 0 {
			penalty = append(penalty, engine.DiscardCards{Player: p.ID, Cards: discard})
		} else {
			penalty = append(penalty, engine.DealEncounterToPlayer{Player: p.ID})
		}
		penalty = append(penalty, engine.ObligationResolve{Player: p.ID, Card: card})
		choices := []engine.Choice{engine.Choice{ID: "fear", Label: "Discard each resource card from your hand", Kind: engine.ChoiceLabel}.Msgs(penalty...)}
		if !p.IsHero() && !p.Exhausted {
			choices = append(choices, engine.Choice{ID: "remove", Label: "Exhaust Lucas Bishop and remove Fear the Future", Kind: engine.ChoiceLabel}.
				Msgs(engine.ExhaustEntity{ID: p.ID}, engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}))
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Fear the Future — choose", choices...)}}
	}})
}

func registerBishopNemesis() {
	engine.RegisterBehavior("45026", &engine.Behavior{})
	engine.RegisterBehavior("45027", &engine.Behavior{})
	engine.RegisterBehavior("45028", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		for _, s := range g.SideSchemes {
			if s != nil && s.Code == "45027" {
				return []engine.Message{engine.SchemeThreat{Scheme: s.ID, N: 2, Source: e.EID()}}
			}
		}
		if g.MainScheme != nil {
			return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: e.EID()}}
		}
		return nil
	}})
	engine.RegisterBehavior("45029", &engine.Behavior{ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		g.Delete(t.ID)
		if len(p.Hand) == 0 {
			return nil
		}
		chosen := p.Hand[0]
		for _, c := range p.Hand[1:] {
			if len(c.Def().Resources) > len(chosen.Def().Resources) {
				chosen = c
			}
		}
		n := len(chosen.Def().Resources)
		msgs := []engine.Message{engine.DiscardCards{Player: p.ID, Cards: engine.CardList{chosen}}}
		for _, id := range g.Schemes() {
			msgs = append(msgs, engine.SchemeThreat{Scheme: id, N: n, Source: t.ID})
		}
		return msgs
	}})
}
