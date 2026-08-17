package core

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// registerSpiderMan installs Spider-Man's identity behavior (Spider-Sense).
func registerSpiderMan() {
	engine.RegisterBehavior("01001", &engine.Behavior{
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			return []engine.Ability{{
				Label:  "Spider-Sense — draw 1 card",
				Type:   engine.AbilityTrigger,
				Trigger: engine.TriggerVillainAttacksYou,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.DrawCards{Player: self, N: 1}}
				},
			}}
		},
	})
}

// registerCoreCards installs behaviors for notable Core Set player cards.
func registerCoreCards() {
	// Swinging Web Kick: deal 8 damage to an enemy.
	engine.RegisterBehavior("01005", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil || len(g.Enemies()) == 0 {
				return nil
			}
			var choices []engine.Choice
			for _, id := range sortedEnemyIDs(g) {
				enemy := g.Entity(id)
				label := fmt.Sprintf("%s", enemy.EDef().Name)
				if v, ok := enemy.(*engine.Villain); ok {
					label = fmt.Sprintf("%s — %d/%d HP", enemy.EDef().Name, v.HP(), v.MaxHP)
				}
				choices = append(choices, engine.Choice{
					Label: label, Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: enemy.ECode(),
				}.Msgs(engine.DamageEntity{Target: id, Damage: 8, Source: pid}))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Swinging Web Kick: choose an enemy", choices...),
			}}
		},
	})

	// Aunt May: Alter-Ego Action — exhaust to heal 4.
	engine.RegisterBehavior("01006", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label:        "Exhaust Aunt May → heal 4 damage",
				Type:         engine.AbilityAction,
				Exhaust:      true,
				AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.HealEntity{Target: e.EOwner(), N: 4}}
				},
			}}
		},
	})
}

func sortedEnemyIDs(g *engine.Game) []engine.EntityID {
	ids := g.Enemies()
	for i := 1; i < len(ids); i++ {
		for j := i; j > 0 && ids[j].Num() < ids[j-1].Num(); j-- {
			ids[j], ids[j-1] = ids[j-1], ids[j]
		}
	}
	return ids
}
