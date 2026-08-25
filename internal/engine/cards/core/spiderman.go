package core

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// registerSpiderMan installs Spider-Man's identity behavior (Spider-Sense).
func registerSpiderMan() {
	engine.RegisterBehavior("01001", &engine.Behavior{
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			return []engine.Ability{{
				Label:   engine.Tf("c.spiderSenseDraw1Card"),
				Type:    engine.AbilityTrigger,
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
		OnPlay: cardutil.ChooseEnemy(engine.Tf("c.swingingWebKickChooseAnEnemy"), func(g *engine.Game, e engine.Entity) (int, []engine.Message) {
			return 8, nil
		}),
	})

	// Aunt May: Alter-Ego Action — exhaust to heal 4.
	engine.RegisterBehavior("01006", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label:        engine.Tf("c.exhaustAuntMayHeal4Damage"),
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
