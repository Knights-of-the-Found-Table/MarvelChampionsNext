// Package wonderman registers Wonder Man, Ionic Physiology and its tucked
// energy-card engine, obligation, and Grim Reaper nemesis set.
package wonderman

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() { registerIdentity(); registerSignatures(); registerObligation(); registerNemesis() }
func hasEnergy(c engine.Card) bool {
	for _, r := range c.Def().Resources {
		if r == "energy" || r == "wild" {
			return true
		}
	}
	return false
}
func ionic(g *engine.Game, p *engine.Player) *engine.Upgrade {
	if p == nil {
		return nil
	}
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil && u.Code == "58002" {
			return u
		}
	}
	return nil
}
func tuckEnergy(g *engine.Game, p *engine.Player, c engine.Card) bool {
	if p == nil || ionic(g, p) == nil || len(p.SenseDeck) >= 3 || !hasEnergy(c) {
		return false
	}
	if c.ID == "" {
		c.ID = g.NextCardID()
	}
	c.Owner = p.ID
	p.SenseDeck = append(p.SenseDeck, c)
	return true
}
func takeTucked(p *engine.Player, i int) engine.Card {
	c := p.SenseDeck[i]
	p.SenseDeck = append(p.SenseDeck[:i], p.SenseDeck[i+1:]...)
	return c
}

func registerIdentity() {
	engine.RegisterBehavior("58001", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		p := g.Player(e.EID())
		m, ok := msg.(engine.BasicAttack)
		if !ok || p == nil || m.Player != p.ID {
			return nil
		}
		p.SenseDeck = nil
		return nil
	}})
}
func registerSignatures() {
	engine.RegisterBehavior("58002", &engine.Behavior{IdentityStatsG: func(g *engine.Game, p *engine.Player, u *engine.Upgrade) engine.StatBonus {
		return engine.StatBonus{ATK: len(p.SenseDeck)}
	}, React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.EventPlayed)
		u := e.(*engine.Upgrade)
		if !ok || m.Player != u.Owner || !hasEnergy(m.Card) {
			return nil
		}
		p := g.Player(u.Owner)
		if _, found := p.Discard.Find(m.Card.ID); !found {
			return nil
		}
		if len(p.SenseDeck) >= 3 {
			return nil
		}
		p.Discard.Remove(m.Card.ID)
		tuckEnergy(g, p, m.Card)
		return []engine.Message{engine.HealEntity{Target: p.ID, N: 1}}
	}})
	// Overpayment amounts are not exposed by PlayCard; signature events use
	// their guaranteed base values and document the omitted energy kickers.
	engine.RegisterBehavior("58003", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		ch := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
			return []engine.Message{engine.ThwartScheme{Scheme: id, N: 1, Source: p.ID}}
		})
		if len(ch) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.activeAltruismChooseAScheme"), ch...)}}
	}})
	engine.RegisterBehavior("58004", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		ch := cardutil.EnemyChoices(g, 3, p.ID, func(id engine.EntityID) []engine.Message {
			return []engine.Message{engine.DamageEntity{Target: id, Damage: 3, Source: p.ID}}
		})
		if len(ch) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.ionicBlastChooseAnEnemy"), ch...)}}
	}})
	engine.RegisterBehavior("58005", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		n := p.AttackStat(g)
		ch := cardutil.EnemyChoices(g, n, p.ID, func(id engine.EntityID) []engine.Message {
			return []engine.Message{engine.DamageEntity{Target: id, Damage: n, Source: p.ID}}
		})
		if len(ch) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.starstruckChooseAnEnemy"), ch...)}}
	}})
	// Resource cards have no spend hook; Energy Siphon's self-damage scaling
	// cannot be represented without engine changes.
	engine.RegisterBehavior("58006", &engine.Behavior{})
	engine.RegisterBehavior("58007", &engine.Behavior{Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
		return []engine.Ability{{Label: engine.Tf("c.wonderFansTuckAnEnergyEventOrDraw"), Type: engine.AbilityAction, AlterEgoOnly: true, Exhaust: true, Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
			s := g.Supports[self]
			p := g.Player(s.Owner)
			var ch []engine.Choice
			for _, c := range p.Discard {
				if c.Def().Type == "event" && hasEnergy(c) && len(p.SenseDeck) < 3 {
					card := c
					ch = append(ch, engine.Choice{Label: engine.Tf("c.tuckName", c), Kind: engine.ChoiceCard, CardCode: c.Code}.Msgs(engine.ReturnDiscardCard{Player: p.ID, CardID: card.ID}))
				}
			}
			ch = append(ch, engine.Choice{ID: "draw", Label: engine.Tf("c.draw1Card"), Kind: engine.ChoiceLabel}.Msgs(engine.DrawCards{Player: p.ID, N: 1}))
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.wonderFansChoose"), ch...)}}
		}}}
	}})
	engine.RegisterBehavior("58008", &engine.Behavior{Resource: &engine.ResourceAbility{Icon: "energy", HeroOnly: true, EventOnly: true}})
	engine.RegisterBehavior("58009", &engine.Behavior{Resource: &engine.ResourceAbility{Icon: "energy"}})
	engine.RegisterBehavior("58010", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.ChangeForm)
		u := e.(*engine.Upgrade)
		if !ok || m.Player != u.Owner {
			return nil
		}
		p := g.Player(u.Owner)
		var ch []engine.Choice
		for _, c := range p.Discard {
			if (c.Def().Type == "event" || c.Def().Type == "resource") && hasEnergy(c) && len(p.SenseDeck) < 3 {
				ch = append(ch, engine.Choice{Label: engine.Tf("c.tuckName", c), Kind: engine.ChoiceCard, CardCode: c.Code}.Msgs(engine.ReturnDiscardCard{Player: p.ID, CardID: c.ID}))
			}
		}
		for _, c := range p.SenseDeck {
			ch = append(ch, engine.Choice{Label: engine.Tf("c.takeName", c), Kind: engine.ChoiceCard, CardCode: c.Code}.Msgs(engine.SideDeckToHand{Player: p.ID, CardID: c.ID}))
		}
		if len(ch) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.signatureSunglassesChoose"), ch...)}}
	}})
	engine.RegisterBehavior("58011", &engine.Behavior{DefeatSave: func(g *engine.Game, p *engine.Player, u *engine.Upgrade) bool {
		p.Damage = max(0, p.MaxHP-4)
		p.Side = engine.SideAlterEgo
		for _, c := range append(engine.CardList(nil), p.Discard...) {
			if len(p.SenseDeck) >= 3 {
				break
			}
			if c.Def().Type == "event" && hasEnergy(c) {
				p.Discard.Remove(c.ID)
				tuckEnergy(g, p, c)
			}
		}
		g.Delete(u.ID)
		return true
	}})
}
func registerObligation() {
	engine.RegisterBehavior("58025", &engine.Behavior{ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
		var ch []engine.Choice
		if !p.IsHero() && !p.Exhausted {
			ch = append(ch, engine.Choice{ID: "exhaust", Label: engine.Tf("c.exhaustSimonWilliams"), Kind: engine.ChoiceLabel}.Msgs(engine.ExhaustEntity{ID: p.ID}, engine.ObligationResolve{Player: p.ID, Card: card}))
		}
		if len(p.SenseDeck) >= 3 {
			ch = append(ch, engine.Choice{ID: "discard", Label: engine.Tf("c.discard3TuckedCards"), Kind: engine.ChoiceLabel}.Msgs(engine.ObligationResolve{Player: p.ID, Card: card}))
		}
		if len(ch) == 0 {
			return []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.pacifismChoose"), ch...)}}
	}})
}
func grim(g *engine.Game) *engine.Minion {
	for _, m := range g.Minions {
		if m != nil && m.Code == "58026" {
			return m
		}
	}
	return nil
}
func registerNemesis() {
	// Ally-defeat attribution is absent from WindowAfterEnemyAttacked; Grim
	// Reaper deals an encounter card after each completed attack.
	engine.RegisterBehavior("58026", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.WindowAfterEnemyAttacked)
		if !ok || m.Enemy != e.EID() {
			return nil
		}
		return []engine.Message{engine.DealEncounterToPlayer{Player: m.Player}}
	}})
	// Brother vs. Brother's hand discard is an attack declaration cost with no side-scheme cost hook.
	engine.RegisterBehavior("58027", &engine.Behavior{})
	engine.RegisterBehavior("58028", &engine.Behavior{ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		g.Delete(t.ID)
		if m := grim(g); m != nil {
			return []engine.Message{engine.MinionActivates{MinionID: m.ID, Player: p.ID}}
		}
		return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 2, Source: t.ID}}
	}})
	engine.RegisterBehavior("58029", &engine.Behavior{ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		g.Delete(t.ID)
		if m := grim(g); m != nil {
			return []engine.Message{engine.MinionActivates{MinionID: m.ID, Player: p.ID}}
		}
		for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
			for _, c := range *zone {
				if c.Code == "58026" {
					zone.Remove(c.ID)
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
			}
		}
		return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: engine.Card{ID: g.NextCardID(), Code: "58026"}}}
	}, Boost: func(g *engine.Game, card engine.Card) []engine.Message {
		if m := grim(g); m != nil {
			return []engine.Message{engine.MinionActivates{MinionID: m.ID, Player: cardutil.FirstPlayerID(g)}}
		}
		return nil
	}})
}
