// Package adamwarlock registers Adam Warlock, his signature cards,
// four-aspect Battle Mage ability, obligation, and nemesis set.
package adamwarlock

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

const battleMageSignal = -21031

func init() {
	registerIdentity()
	registerSignatures()
	registerObligation()
	registerNemesis()
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

func battleMageEffect(g *engine.Game, p *engine.Player, aspect string) []engine.Message {
	var choices []engine.Choice
	switch aspect {
	case "aggression":
		choices = cardutil.EnemyChoices(g, 2, p.ID, func(id engine.EntityID) []engine.Message {
			return []engine.Message{engine.DamageEntity{Target: id, Damage: 2, Source: p.ID}, engine.AddEntityCounter{ID: p.ID, N: battleMageSignal}}
		})
	case "justice":
		choices = cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
			return []engine.Message{engine.ThwartScheme{Scheme: id, N: 2, Source: p.ID}, engine.AddEntityCounter{ID: p.ID, N: battleMageSignal}}
		})
	case "protection":
		for _, pl := range g.Players {
			for _, id := range pl.Allies {
				if a := g.Allies[id]; a != nil && a.Damage > 0 {
					choices = append(choices, engine.Choice{Label: a.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id, CardCode: a.Code}.
						Msgs(engine.HealEntity{Target: id, N: 1}, engine.AddEntityCounter{ID: p.ID, N: battleMageSignal}))
				}
			}
		}
	case "leadership":
		for _, pl := range g.Players {
			choices = append(choices, engine.Choice{Label: pl.Name, Kind: engine.ChoiceTarget, SourceID: pl.ID}.
				Msgs(engine.ApplyStatBonus{Target: pl.ID, THW: 1, ATK: 1, DEF: 1}, engine.AddEntityCounter{ID: p.ID, N: battleMageSignal}))
		}
	}
	if len(choices) == 0 {
		return []engine.Message{engine.AddEntityCounter{ID: p.ID, N: battleMageSignal}}
	}
	return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Battle Mage — resolve "+aspect, choices...)}}
}

func battleMageChoices(g *engine.Game, p *engine.Player) []engine.Message {
	if p == nil || len(p.Hand) == 0 {
		return nil
	}
	var choices []engine.Choice
	for _, c := range p.Hand {
		aspect := c.Def().Aspect
		if aspect != "aggression" && aspect != "justice" && aspect != "protection" && aspect != "leadership" {
			continue
		}
		msgs := []engine.Message{engine.DiscardCards{Player: p.ID, Cards: engine.CardList{c}}}
		msgs = append(msgs, battleMageEffect(g, p, aspect)...)
		choices = append(choices, engine.Choice{Label: fmt.Sprintf("Discard %s (%s)", c.Def().Name, aspect), Kind: engine.ChoiceCard, CardCode: c.Code}.Msgs(msgs...))
	}
	if len(choices) == 0 {
		return nil
	}
	return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Battle Mage — discard an aspect card", choices...)}}
}

func registerIdentity() {
	engine.RegisterBehavior("21031", &engine.Behavior{
		HeroSetup: func(g *engine.Game, p *engine.Player) []engine.Message {
			// The engine has no pre-game deck rejection hook. Audit the assembled
			// deck and log an explicit warning when the four aspect counts are not
			// equal; singleton and no-basic-aspect constraints remain UI concerns.
			counts := map[string]int{}
			for _, zone := range []engine.CardList{p.Deck, p.Hand, p.Discard} {
				for _, c := range zone {
					if a := c.Def().Aspect; a == "aggression" || a == "justice" || a == "protection" || a == "leadership" {
						counts[a]++
					}
				}
			}
			if counts["aggression"] != counts["justice"] || counts["aggression"] != counts["protection"] || counts["aggression"] != counts["leadership"] {
				g.Logf("Adam Warlock deckbuilding warning: aspect counts are not equal: %v", counts)
			}
			return nil
		},
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			if p == nil || !p.IsHero() || len(battleMageChoices(g, p)) == 0 {
				return nil
			}
			return []engine.Ability{{Label: "Battle Mage — discard an aspect card", Type: engine.AbilityAction, HeroOnly: true, OncePerTurn: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return battleMageChoices(g, g.Player(self))
				}}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			p := g.Player(e.EID())
			m, ok := msg.(engine.AddEntityCounter)
			if !ok || p == nil || m.ID != p.ID || m.N != battleMageSignal {
				return nil
			}
			var msgs []engine.Message
			if cape := ownedUpgrade(g, p, "21035"); cape != nil && !cape.Exhausted {
				// Optional responses are automatic after the named ability resolves.
				msgs = append(msgs, engine.ExhaustEntity{ID: cape.ID}, engine.ReadyEntity{ID: p.ID})
			}
			for _, id := range p.Upgrades {
				if senses := g.Upgrades[id]; senses != nil && senses.Code == "21037" && !senses.Exhausted {
					msgs = append(msgs, engine.DrawCards{Player: p.ID, N: 1})
				}
			}
			return msgs
		},
	})
}

func aspectVariety(cards engine.CardList) int {
	seen := map[string]bool{}
	for _, c := range cards {
		switch c.Def().Aspect {
		case "aggression", "justice", "protection", "leadership":
			seen[c.Def().Aspect] = true
		}
	}
	return len(seen)
}

func topCards(p *engine.Player, n int) engine.CardList {
	if n > len(p.Deck) {
		n = len(p.Deck)
	}
	return append(engine.CardList(nil), p.Deck[:n]...)
}

func registerSignatures() {
	engine.RegisterBehavior("21032", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		return []engine.Message{engine.ToughEntity{Target: e.EID()}}
	}})
	engine.RegisterBehavior("21033", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.ShufflePlayerDeck)
			s := e.(*engine.Support)
			if ok && m.Player == s.Owner {
				return []engine.Message{engine.AddEntityCounter{ID: s.ID, N: 1}}
			}
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := e.(*engine.Support)
			if s.Counters < 1 {
				return nil
			}
			return []engine.Ability{{Label: "Soul World — remove a soul counter and heal all damage", Type: engine.AbilityAction, AlterEgoOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.Owner)
					return []engine.Message{engine.AddEntityCounter{ID: s.ID, N: -1}, engine.HealEntity{Target: p.ID, N: p.Damage}}
				}}}
		},
	})
	engine.RegisterBehavior("21034", &engine.Behavior{Resource: &engine.ResourceAbility{Icon: "wild"}})
	engine.RegisterBehavior("21035", &engine.Behavior{})
	engine.RegisterBehavior("21036", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.RevealEncounterCard)
		u := e.(*engine.Upgrade)
		if !ok || m.Player != u.Owner || m.Card.Def().Type != "treachery" {
			return nil
		}
		// The reveal message cannot be canceled by a reaction. Discarding the
		// Ward is exact; cancellation of When Revealed is the known hook gap.
		return []engine.Message{engine.DiscardControlled{Player: u.Owner, ID: u.ID}}
	}})
	engine.RegisterBehavior("21037", &engine.Behavior{})
	engine.RegisterBehavior("21038", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		n := min(4, len(p.Deck))
		damage := 4 + aspectVariety(topCards(p, n))
		choices := cardutil.EnemyChoices(g, damage, p.ID, func(id engine.EntityID) []engine.Message {
			return []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: n}, engine.DamageEntity{Target: id, Damage: damage, Source: p.ID}}
		})
		if len(choices) == 0 {
			return []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: n}}
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Karmic Blast — choose an enemy", choices...)}}
	}})
	engine.RegisterBehavior("21039", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		n := min(4, len(p.Deck))
		amount := 3 + aspectVariety(topCards(p, n))
		choices := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
			return []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: n}, engine.ThwartScheme{Scheme: id, N: amount, Source: p.ID}}
		})
		if len(choices) == 0 {
			return []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: n}}
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Cosmic Awareness — choose a scheme", choices...)}}
	}})
	engine.RegisterBehavior("21040", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		var choices []engine.Choice
		for _, c := range p.Discard {
			if c.ID == "" || c.Code == "21040" {
				continue
			}
			choices = append(choices, engine.Choice{Label: c.Def().Name, Kind: engine.ChoiceCard, CardCode: c.Code}.
				Msgs(engine.ReturnDiscardCard{Player: p.ID, CardID: c.ID}))
		}
		if len(choices) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Quantum Magic — return a card", choices...)}}
	}})
}

func registerObligation() {
	engine.RegisterBehavior("21066", &engine.Behavior{ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
		var choices []engine.Choice
		if !p.IsHero() && !p.Exhausted {
			choices = append(choices, engine.Choice{ID: "remove", Label: "Exhaust Adam Warlock and remove Regeneration Cycle", Kind: engine.ChoiceLabel}.
				Msgs(engine.ExhaustEntity{ID: p.ID}, engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}))
		}
		top := topCards(p, 5)
		choices = append(choices, engine.Choice{ID: "mill", Label: "Discard 5 cards and place threat", Kind: engine.ChoiceLabel}.
			Msgs(engine.MillPlayerDeck{Player: p.ID, N: len(top)}, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: aspectVariety(top), Source: p.ID}, engine.ObligationResolve{Player: p.ID, Card: card}))
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Regeneration Cycle — choose", choices...)}}
	}})
}

func registerNemesis() {
	engine.RegisterBehavior("21067", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.MinionActivates)
		if !ok || m.MinionID != e.EID() {
			return nil
		}
		return []engine.Message{engine.MillPlayerDeck{Player: m.Player, N: 5}}
	}})
	engine.RegisterBehavior("21068", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.ShufflePlayerDeck)
		if !ok {
			return nil
		}
		return []engine.Message{engine.ExhaustEntity{ID: m.Player}, engine.StunEntity{Target: m.Player}}
	}, Boost: func(g *engine.Game, card engine.Card) []engine.Message {
		return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: card}}
	}})
	// Threat immunity while Universal Church is in play has no thwart gate;
	// Zealot's boost put-into-play is supported by generic boost handling.
	engine.RegisterBehavior("21069", &engine.Behavior{})
	engine.RegisterBehavior("21070", &engine.Behavior{ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		g.Delete(t.ID)
		for _, s := range g.SideSchemes {
			if s != nil && s.Code == "21068" {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: t.ID}, engine.MillPlayerDeck{Player: p.ID, N: 10}}
			}
		}
		for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
			for _, c := range *zone {
				if c.Code == "21068" {
					zone.Remove(c.ID)
					return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: t.ID}, engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
			}
		}
		return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: t.ID}, engine.RevealEncounterCard{Player: p.ID, Card: engine.Card{ID: g.NextCardID(), Code: "21068"}}}
	}})
}
