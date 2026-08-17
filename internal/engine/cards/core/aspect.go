package core

import (

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// registerAspectCards installs behaviors for frequently-played Core Set
// aspect and basic cards.
func registerAspectCards() {
	// For Justice!: remove 3 threat (4 if paid with a mental resource).
	engine.RegisterBehavior("01060", &engine.Behavior{
		OnPlay: cardutil.ChooseScheme("For Justice!", func(g *engine.Game, e engine.Entity) int {
			if ec, ok := e.(*engine.EventCard); ok && ec.Paid.PaidIcon("mental") {
				return 4
			}
			return 3
		}),
	})

	// Uppercut: deal 5 damage to an enemy.
	engine.RegisterBehavior("01054", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy("Uppercut — deal 5 damage", func(g *engine.Game, e engine.Entity) (int, []engine.Message) {
			n := 5
			if ec, ok := e.(*engine.EventCard); ok && ec.Paid.PaidIcon("physical") {
				// physical rider: +2 damage (approximation of the
				// printed "+2 if physical" line)
				n = 7
			}
			return n, nil
		}),
	})

	// Relentless Assault: deal 5 damage to a minion.
	engine.RegisterBehavior("01053", &engine.Behavior{
		OnPlay: cardutil.ChooseMinion("Relentless Assault — deal 5 damage", 5),
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
				for _, id := range cardutil.SortedEnemyIDs(g) {
					enemy := g.Entity(id)
					dmgChoices = append(dmgChoices, engine.Choice{
						Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget, SourceID: id, CardCode: enemy.ECode(),
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
