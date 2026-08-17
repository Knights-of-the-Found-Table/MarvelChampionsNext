package core

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// registerAspectCards installs behaviors for frequently-played Core Set
// aspect and basic cards.
func registerAspectCards() {
	// For Justice!: remove 3 threat (4 if paid with a mental resource).
	engine.RegisterBehavior("01060", &engine.Behavior{
		OnPlay: chooseSchemeEffect("01060", "For Justice!", func(g *engine.Game, e engine.Entity) int {
			if ec, ok := e.(*engine.EventCard); ok && ec.Paid.PaidIcon("mental") {
				return 4
			}
			return 3
		}),
	})

	// Uppercut: deal 5 damage to an enemy.
	engine.RegisterBehavior("01054", &engine.Behavior{
		OnPlay: chooseEnemyEffect("Uppercut — deal 5 damage", func(g *engine.Game, e engine.Entity) (int, []engine.Message) {
			n := 5
			var extra []engine.Message
			if ec, ok := e.(*engine.EventCard); ok && ec.Paid.PaidIcon("physical") {
				// physical rider: +2 damage (approximation of the
				// printed "+2 if physical" line)
				n = 7
			}
			return n, extra
		}),
	})

	// Relentless Assault: deal 5 damage to a minion.
	engine.RegisterBehavior("01053", &engine.Behavior{
		OnPlay: chooseMinionEffect("Relentless Assault — deal 5 damage", 5),
	})

	// First Aid: heal 2 damage from any character (approximation: heal
	// yourself or one of your allies).
	engine.RegisterBehavior("01086", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			if p.Damage > 0 {
				choices = append(choices, engine.Choice{
					Label: "Heal " + p.Name, Kind: engine.ChoiceTarget, SourceID: p.ID,
				}.Msgs(engine.HealEntity{Target: p.ID, N: 2}))
			}
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && a.Damage > 0 {
					choices = append(choices, engine.Choice{
						Label: "Heal " + a.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id, CardCode: a.Code,
					}.Msgs(engine.HealEntity{Target: id, N: 2}))
				}
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("First Aid — heal 2 damage", choices...),
			}}
		},
	})

	// Maria Hill: response after she enters play — each player draws 1.
	engine.RegisterBehavior("01067", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, engine.DrawCards{Player: p.ID, N: 1})
			}
			return msgs
		},
	})

	// Nick Fury: draw 2, deal 2 damage, or remove 2 threat — the full
	// sequence is complex; approximation: draw 2 then discard at end of
	// round is skipped, player chooses one effect.
	engine.RegisterBehavior("01084", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			choices := []engine.Choice{
				engine.Choice{ID: "draw", Label: "Draw 2 cards", Kind: engine.ChoiceLabel}.
					Msgs(engine.DrawCards{Player: pid, N: 2}),
			}
			if len(g.Enemies()) > 0 {
				var dmgChoices []engine.Choice
				for _, id := range sortedEnemyIDs(g) {
					enemy := g.Entity(id)
					dmgChoices = append(dmgChoices, engine.Choice{
						Label: enemy.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id, CardCode: enemy.ECode(),
					}.Msgs(engine.DamageEntity{Target: id, Damage: 4, Source: pid}))
				}
				choices = append(choices, engine.Choice{
					ID: "damage", Label: "Deal 4 damage to an enemy", Kind: engine.ChoiceLabel,
				}.WithThen(engine.Ask("Choose an enemy", dmgChoices...)))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Nick Fury — choose one", choices...),
			}}
		},
	})
}

// chooseSchemeEffect builds a "remove N threat from a scheme" OnPlay hook.
func chooseSchemeEffect(code, name string, amount func(g *engine.Game, e engine.Entity) int) func(g *engine.Game, e engine.Entity) []engine.Message {
	return func(g *engine.Game, e engine.Entity) []engine.Message {
		pid := e.EOwner()
		p := g.Player(pid)
		if p == nil {
			return nil
		}
		schemes := g.Schemes()
		if len(schemes) == 0 {
			return nil
		}
		var choices []engine.Choice
		for _, id := range schemes {
			s := g.Entity(id)
			choices = append(choices, engine.Choice{
				Label: s.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id, CardCode: s.ECode(),
			}.Msgs(engine.ThwartScheme{Scheme: id, N: amount(g, e), Source: pid}))
		}
		return []engine.Message{engine.AskQuestion{
			Player:   pid,
			Question: engine.Ask(name + " — choose a scheme", choices...),
		}}
	}
}

// chooseEnemyEffect builds a "deal damage to an enemy" OnPlay hook; the
// callback returns the damage plus optional extra messages.
func chooseEnemyEffect(prompt string, f func(g *engine.Game, e engine.Entity) (int, []engine.Message)) func(g *engine.Game, e engine.Entity) []engine.Message {
	return func(g *engine.Game, e engine.Entity) []engine.Message {
		pid := e.EOwner()
		if len(g.Enemies()) == 0 {
			return nil
		}
		n, extra := f(g, e)
		_ = extra
		var choices []engine.Choice
		for _, id := range sortedEnemyIDs(g) {
			enemy := g.Entity(id)
			choices = append(choices, engine.Choice{
				Label: enemy.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id, CardCode: enemy.ECode(),
			}.Msgs(engine.DamageEntity{Target: id, Damage: n, Source: pid}))
		}
		return []engine.Message{engine.AskQuestion{
			Player:   pid,
			Question: engine.Ask(prompt, choices...),
		}}
	}
}

// chooseMinionEffect builds a "deal damage to a minion" OnPlay hook.
func chooseMinionEffect(prompt string, dmg int) func(g *engine.Game, e engine.Entity) []engine.Message {
	return func(g *engine.Game, e engine.Entity) []engine.Message {
		pid := e.EOwner()
		var choices []engine.Choice
		for _, id := range sortedIDsOf(g.Minions) {
			mn := g.Minions[id]
			choices = append(choices, engine.Choice{
				Label: mn.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id, CardCode: mn.Code,
			}.Msgs(engine.DamageEntity{Target: id, Damage: dmg, Source: pid}))
		}
		if len(choices) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{
			Player:   pid,
			Question: engine.Ask(prompt, choices...),
		}}
	}
}
