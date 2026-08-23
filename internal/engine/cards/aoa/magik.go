package aoa

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerMagik()
	registerMagikSignatures()
	registerMagikObligation()
	registerMagikNemesis()
}

func topHasResource(p *engine.Player, icons ...string) bool {
	if p == nil || len(p.Deck) == 0 {
		return false
	}
	for _, have := range p.Deck[0].Def().Resources {
		for _, want := range icons {
			if have == want || have == "wild" {
				return true
			}
		}
	}
	return false
}

func registerMagik() {
	engine.RegisterBehavior("45030", &engine.Behavior{
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			if !p.IsHero() || len(p.Deck) == 0 {
				return nil
			}
			return []engine.Ability{{
				Label: "Play the top card of your deck with cost reduced by 1",
				Type:  engine.AbilityAction, HeroOnly: true, OncePerTurn: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					pl := g.Player(self)
					if pl == nil || len(pl.Deck) == 0 {
						return nil
					}
					// The turn menu cannot play directly from the deck. Move the
					// faceup card to hand and grant the printed one-shot discount.
					return []engine.Message{
						engine.TakeDeckCard{Player: pl.ID, CardID: pl.Deck[0].ID},
						engine.CostDiscountApply{Player: pl.ID, Amount: 1},
					}
				},
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			p := g.Player(e.EID())
			m, ok := msg.(engine.ChangeForm)
			if !ok || p == nil || m.Player != p.ID || !p.IsHero() {
				return nil
			}
			var choices []engine.Choice
			for _, c := range p.Discard {
				if c.Def().HasTrait("spell") {
					// No discard-to-deck-top message exists. ShuffleIntoDeck
					// preserves the zone change but randomizes its final position.
					choices = append(choices, engine.Choice{Label: "Return " + c.Def().Name + " to your deck", Kind: engine.ChoiceCard, CardCode: c.Code}.
						Msgs(engine.ShuffleIntoDeck{Player: p.ID, CardID: c.ID}))
				}
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask("Illyana Rasputin — return a Spell to your deck", choices...)}}
		},
	})
}

func registerMagikSignatures() {
	// Colossus's play-from-hand defender interrupt is outside the current
	// defense-event channel, so his printed stats and Toughness remain generic.
	engine.RegisterBehavior("45031", &engine.Behavior{})

	engine.RegisterBehavior("45032", &engine.Behavior{Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
		return []engine.Ability{{Label: "Limbo — swap a hand card with the top of your deck", Type: engine.AbilityAction, Exhaust: true,
			Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
				s := g.Supports[self]
				if s == nil {
					return nil
				}
				p := g.Player(s.Owner)
				var choices []engine.Choice
				for _, c := range p.Hand {
					choices = append(choices, engine.Choice{Label: "Put " + c.Def().Name + " on top", Kind: engine.ChoiceCard, CardCode: c.Code}.
						Msgs(engine.SwapHandWithDeckTop{Player: p.ID, CardID: c.ID}))
				}
				if len(choices) == 0 || len(p.Deck) == 0 {
					return nil
				}
				return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Limbo — choose a hand card", choices...)}}
			}}}
	}})

	engine.RegisterBehavior("45033", &engine.Behavior{IdentityStats: func(p *engine.Player) engine.StatBonus {
		if p.IsHero() && topHasResource(p, "mental") {
			return engine.StatBonus{THW: 1}
		}
		return engine.StatBonus{}
	}})
	engine.RegisterBehavior("45034", &engine.Behavior{IdentityStats: func(p *engine.Player) engine.StatBonus {
		if p.IsHero() && topHasResource(p, "physical") {
			return engine.StatBonus{ATK: 1}
		}
		return engine.StatBonus{}
	}})
	engine.RegisterBehavior("45035", &engine.Behavior{IdentityStats: func(p *engine.Player) engine.StatBonus {
		bonus := engine.StatBonus{Retaliate: 1}
		if p.IsHero() && topHasResource(p, "energy") {
			bonus.DEF = 1
		}
		return bonus
	}})

	engine.RegisterBehavior("45036", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		if p == nil || len(p.Deck) == 0 {
			return nil
		}
		n := min(3, len(p.Deck))
		var choices []engine.Choice
		for _, c := range p.Deck[:n] {
			choices = append(choices, engine.Choice{Label: "Draw " + c.Def().Name, Kind: engine.ChoiceCard, CardCode: c.Code}.
				Msgs(engine.TopDeckPick{Player: p.ID, CardID: c.ID}))
		}
		// TopDeckPick draws one and bottoms the others. This approximates
		// Scrying's separate discard-one / return-one ordering decisions.
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Scrying — choose a card to draw", choices...)}}
	}})

	engine.RegisterBehavior("45037", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		var choices []engine.Choice
		for _, c := range p.Discard {
			if c.Code != "45037" && c.Def().CardSet == "magik" {
				choices = append(choices, engine.Choice{Label: "Return " + c.Def().Name + " to your deck", Kind: engine.ChoiceCard, CardCode: c.Code}.
					Msgs(engine.ReadyEntity{ID: p.ID}, engine.ShuffleIntoDeck{Player: p.ID, CardID: c.ID}))
			}
		}
		if len(choices) == 0 {
			return []engine.Message{engine.ReadyEntity{ID: p.ID}}
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Stepping Disc — choose a Magik card", choices...)}}
	}})

	engine.RegisterBehavior("45038", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		choices := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
			msgs := []engine.Message{engine.ThwartScheme{Scheme: id, N: 4, Source: p.ID}}
			if topHasResource(p, "mental") {
				for _, vid := range cardutil.SortedIDs(g.Villains) {
					msgs = append(msgs, engine.ConfuseEntity{Target: vid})
					break
				}
			}
			return msgs
		})
		if len(choices) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Exorcism — choose a scheme", choices...)}}
	}})

	engine.RegisterBehavior("45039", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		choices := cardutil.EnemyChoices(g, 4, p.ID, func(id engine.EntityID) []engine.Message {
			msgs := []engine.Message{engine.DamageEntity{Target: id, Damage: 4, Source: p.ID}}
			if topHasResource(p, "physical") {
				msgs = append(msgs, engine.StunEntity{Target: id})
			}
			return msgs
		})
		if len(choices) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Soul Strike — choose an enemy", choices...)}}
	}})

	engine.RegisterBehavior("45040", &engine.Behavior{DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
		var msgs []engine.Message
		if topHasResource(p, "energy") {
			msgs = append(msgs, engine.DamageEntity{Target: against, Damage: 3, Source: p.ID})
		}
		return engine.Defends{Defender: p.ID, Against: against, ExtraPrevent: 3, NoExhaust: true}, msgs, true
	}})
}

func registerMagikObligation() {
	engine.RegisterBehavior("45053", &engine.Behavior{ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
		penalty := []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 1, Source: p.ID}}
		for _, id := range p.Allies {
			penalty = append(penalty, engine.DamageEntity{Target: id, Damage: 1, Source: p.ID})
		}
		penalty = append(penalty, engine.ObligationResolve{Player: p.ID, Card: card})
		choices := []engine.Choice{engine.Choice{ID: "darkchilde", Label: "Deal 1 damage to each character you control", Kind: engine.ChoiceLabel}.Msgs(penalty...)}
		if !p.IsHero() && !p.Exhausted {
			choices = append(choices, engine.Choice{ID: "remove", Label: "Exhaust Illyana Rasputin and remove Darkchilde", Kind: engine.ChoiceLabel}.
				Msgs(engine.ExhaustEntity{ID: p.ID}, engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}))
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Darkchilde — choose", choices...)}}
	}})
}

func registerMagikNemesis() {
	engine.RegisterBehavior("45054", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.MinionActivates)
		if !ok || m.MinionID != e.EID() {
			return nil
		}
		return []engine.Message{engine.MillPlayerDeck{Player: m.Player, N: 3}}
	}})
	// Ruler of Limbo's threat lock and facedown Limbo attachment require a
	// side-scheme damage gate / attached-player-card zone the engine lacks.
	engine.RegisterBehavior("45055", &engine.Behavior{})
	engine.RegisterBehavior("45056", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		for _, s := range g.SideSchemes {
			if s != nil && s.Code == "45055" {
				return []engine.Message{engine.SchemeThreat{Scheme: s.ID, N: 2, Source: e.EID()}}
			}
		}
		if g.MainScheme != nil {
			return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: e.EID()}}
		}
		return nil
	}})
	// Witchfire's ally-defeat attribution is not exposed after an attack.
	engine.RegisterBehavior("45057", &engine.Behavior{})
	engine.RegisterBehavior("45058", &engine.Behavior{ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		g.Delete(t.ID)
		var msgs []engine.Message
		for _, mn := range g.Minions {
			if mn != nil && mn.EngagedWith == p.ID && mn.EDef().HasTrait("limbo") {
				msgs = append(msgs, engine.MinionActivates{MinionID: mn.ID, Player: p.ID})
			}
		}
		if len(msgs) == 0 {
			msgs = append(msgs, engine.RevealNextEncounter{Player: p.ID})
		}
		return msgs
	}, Boost: func(g *engine.Game, card engine.Card) []engine.Message {
		for _, s := range g.SideSchemes {
			if s != nil && s.Code == "45055" {
				return []engine.Message{engine.SchemeThreat{Scheme: s.ID, N: 2}}
			}
		}
		return nil
	}})
}
