// Package winter registers Winter Soldier, his Cybernetic Arm suite,
// obligation, and Crossbones nemesis set.
package winter

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() { registerIdentity(); registerSignatures(); registerObligation(); registerNemesis() }

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
func lethal(g *engine.Game, id engine.EntityID, n int) bool {
	switch x := g.Entity(id).(type) {
	case *engine.Minion:
		return x.HP() <= n
	case *engine.Villain:
		return x.HP() <= n
	}
	return false
}
func thwartTwo(g *engine.Game, p *engine.Player, prompt string) []engine.Message {
	ch := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
		return []engine.Message{engine.ThwartScheme{Scheme: id, N: 2, Source: p.ID}}
	})
	if len(ch) == 0 {
		return nil
	}
	return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.S(prompt), ch...)}}
}

func registerIdentity() {
	engine.RegisterBehavior("54001", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		p := g.Player(e.EID())
		m, ok := msg.(engine.DamageEntity)
		if !ok || p == nil || m.Source != p.ID || !lethal(g, m.Target, m.Damage) {
			return nil
		}
		return thwartTwo(g, p, "Lethal Protector — remove 2 threat")
	}})
}

func registerSignatures() {
	// The payment engine cannot attach an event-damage rider to a resource
	// upgrade; wild-for-Attack is exact, +1 damage is approximated by events.
	engine.RegisterBehavior("54002", &engine.Behavior{Resource: &engine.ResourceAbility{Icon: "wild", EventOnly: true}})
	engine.RegisterBehavior("54003", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		var ch []engine.Choice
		for _, c := range p.Discard {
			if c.Def().Type == "event" && c.Def().HasTrait("attack") {
				ch = append(ch, engine.Choice{Label: engine.S(c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code}.Msgs(engine.ReturnDiscardCard{Player: p.ID, CardID: c.ID}))
			}
		}
		if len(ch) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.blackWidowReturnAnAttackEvent"), ch...)}}
	}})
	engine.RegisterBehavior("54004", &engine.Behavior{DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
		prevent := false
		if arm := ownedUpgrade(g, p, "54002"); arm != nil && arm.Exhausted {
			prevent = true
		}
		return engine.Defends{Defender: p.ID, Against: against, PreventAll: prevent}, []engine.Message{engine.DamageEntity{Target: against, Damage: 3, Source: p.ID}}, true
	}})
	engine.RegisterBehavior("54005", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		n := 7
		if arm := ownedUpgrade(g, p, "54002"); arm != nil && arm.Exhausted {
			n++
		}
		ch := cardutil.EnemyChoices(g, n, p.ID, func(id engine.EntityID) []engine.Message {
			return []engine.Message{engine.DamageEntity{Target: id, Damage: n, Source: p.ID}}
		})
		if len(ch) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.metalPunchChooseAnEnemy"), ch...)}}
	}})
	engine.RegisterBehavior("54006", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		ch := cardutil.EnemyChoices(g, 4, p.ID, func(id engine.EntityID) []engine.Message {
			return []engine.Message{engine.DamageEntity{Target: id, Damage: 4, Source: p.ID}}
		})
		if len(ch) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.electricalDischargeChooseAnEnemy"), ch...)}}
	}})
	engine.RegisterBehavior("54007", &engine.Behavior{Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
		return []engine.Ability{{Label: engine.Tf("c.safeHouse30FindAMinionAndDraw"), Type: engine.AbilityAction, AlterEgoOnly: true, Exhaust: true, Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
			s := g.Supports[self]
			p := g.Player(s.Owner)
			var ch []engine.Choice
			for _, c := range g.EncounterDeck {
				if c.Def().Type == "minion" {
					ch = append(ch, engine.Choice{Label: engine.S(c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code}.Msgs(engine.RevealEncounterCard{Player: p.ID, Card: c}, engine.DrawCards{Player: p.ID, N: 1}, engine.ShufflePlayerDeck{Player: p.ID}))
				}
			}
			if len(ch) == 0 {
				return []engine.Message{engine.DrawCards{Player: p.ID, N: 1}}
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.safeHouse30ChooseAMinion"), ch...)}}
		}}}
	}})
	engine.RegisterBehavior("54008", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		_, ok := msg.(engine.MinionDefeated)
		u := e.(*engine.Upgrade)
		if !ok {
			return nil
		}
		p := g.Player(u.Owner)
		ch := cardutil.EnemyChoices(g, 0, p.ID, func(id engine.EntityID) []engine.Message {
			return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: u.ID}, engine.ReadyEntity{ID: p.ID}, engine.ConfuseEntity{Target: id}}
		})
		if len(ch) == 0 {
			return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: u.ID}, engine.ReadyEntity{ID: p.ID}}
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.silentInfiltrationConfuseAnEnemy"), ch...)}}
	}})
	engine.RegisterBehavior("54009", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		p.MaxHP += 3
		return []engine.Message{engine.GrantTrait{Target: p.ID, Trait: "steady"}}
	}})
	engine.RegisterBehavior("54010", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		return []engine.Message{engine.GrantTrait{Target: e.EOwner(), Trait: "spy"}}
	}, React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		u := e.(*engine.Upgrade)
		if u.Exhausted {
			return nil
		}
		m, ok := msg.(engine.DamageEntity)
		if !ok || m.Source != u.Owner || !lethal(g, m.Target, m.Damage) {
			return nil
		}
		return []engine.Message{engine.ExhaustEntity{ID: u.ID}, engine.DrawCards{Player: u.Owner, N: 1}}
	}})
	engine.RegisterBehavior("54011", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.BasicAttack)
		u := e.(*engine.Upgrade)
		if !ok || m.Player != u.Owner || u.Exhausted {
			return nil
		}
		return []engine.Message{engine.ExhaustEntity{ID: u.ID}, engine.DamageEntity{Target: m.Target, Damage: 2, Source: u.Owner}}
	}})
}

func registerObligation() {
	engine.RegisterBehavior("54027", &engine.Behavior{ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
		var choices []engine.Choice
		if !p.IsHero() && !p.Exhausted {
			choices = append(choices, engine.Choice{ID: "remove", Label: engine.Tf("c.exhaustBuckyBarnesAndRemove"), Kind: engine.ChoiceLabel}.Msgs(engine.ExhaustEntity{ID: p.ID}, engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}))
		}
		if len(p.Hand) > 0 {
			best := p.Hand[0]
			cost := cardutil.Cost(best.Def())
			for _, c := range p.Hand[1:] {
				if n := cardutil.Cost(c.Def()); n > cost {
					best = c
					cost = n
				}
			}
			choices = append(choices, engine.Choice{ID: "discard", Label: engine.S("Discard " + best.Def().Name + " and take damage"), Kind: engine.ChoiceCard, CardCode: best.Code}.Msgs(engine.DiscardCards{Player: p.ID, Cards: engine.CardList{best}}, engine.DamageEntity{Target: p.ID, Damage: cost, Source: p.ID}, engine.ObligationResolve{Player: p.ID, Card: card}))
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.redRoomProgrammingChoose"), choices...)}}
	}})
}
func registerNemesis() {
	engine.RegisterBehavior("54028", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.WindowAfterEnemyAttacked)
		if !ok || m.Enemy != e.EID() {
			return nil
		}
		if g.MainScheme != nil {
			return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: e.EID()}}
		}
		return nil
	}})
	engine.RegisterBehavior("54029", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.SchemeDefeated)
		if !ok || m.Scheme != e.EID() {
			return nil
		}
		p := cardutil.FirstPlayerID(g)
		for _, c := range g.EncounterDeck {
			if c.Def().Type == "minion" && c.Def().HasTrait("hydra") {
				return []engine.Message{engine.RevealEncounterCard{Player: p, Card: c}}
			}
		}
		return nil
	}})
	engine.RegisterBehavior("54030", &engine.Behavior{OnAttach: func(g *engine.Game, a *engine.Attachment, target engine.EntityID) []engine.Message {
		if target == "" {
			for _, mn := range g.Minions {
				if mn != nil && mn.Code == "54028" {
					target = mn.ID
					break
				}
			}
		}
		if target == "" {
			target = g.ActiveVillain
		}
		a.Target = target
		pid := cardutil.FirstPlayerID(g)
		if g.Minions[target] != nil {
			return []engine.Message{engine.MinionActivates{MinionID: target, Player: pid}}
		}
		if g.Villains[target] != nil {
			return []engine.Message{engine.VillainActivates{VillainID: target, Player: pid}}
		}
		return nil
	}})
	engine.RegisterBehavior("54031", &engine.Behavior{})
}
