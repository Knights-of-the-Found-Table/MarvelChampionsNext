package daredevil

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// registerDeckCards installs the Fear No Evil cards used by the reference
// Protection decklist (60030, 60038, 60048-60054).
func registerDeckCards() {
	registerContingencyPlanning()
	registerInnateReflexes()
	registerArmyOfOne()
	registerInHarmsWay()
	registerBestOffense()
	registerRonin()
	registerStandAlone()
}

// 60030 Contingency Planning: tuck 1 upgrade under here (approximation:
// stored cards are retrieved to hand instead of played directly).
func registerContingencyPlanning() {
	engine.RegisterBehavior("60030", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s, ok := e.(*engine.Support)
			if !ok {
				return nil
			}
			p := g.Player(s.Owner)
			if p == nil {
				return nil
			}
			var abs []engine.Ability
			hasUpgrade := false
			for _, c := range p.Hand {
				if c.Def().Type == "upgrade" {
					hasUpgrade = true
					break
				}
			}
			if hasUpgrade {
				abs = append(abs, engine.Ability{
					Label:   engine.Tf("c.contingencyPlanningTuckAnUpgradeFromYourHand"),
					Type:    engine.AbilityAction,
					Exhaust: true,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						var choices []engine.Choice
						for _, c := range p.Hand {
							if c.Def().Type != "upgrade" {
								continue
							}
							choices = append(choices, engine.Choice{
								Label: engine.S("Tuck " + c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code,
							}.Msgs(engine.SupportStoreCard{ID: s.ID, Card: c}))
						}
						if len(choices) == 0 {
							return nil
						}
						return []engine.Message{engine.AskQuestion{
							Player:   p.ID,
							Question: engine.Ask(engine.Tf("c.contingencyPlanningTuckWhichUpgrade"), choices...),
						}}
					},
				})
			}
			if s.Counters > 0 {
				abs = append(abs, engine.Ability{
					Label:   engine.Tf("c.contingencyPlanningTakeTheTuckedUpgradeBackToHand"),
					Type:    engine.AbilityAction,
					Exhaust: true,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						return []engine.Message{engine.SupportRetrieveCards{ID: s.ID, Cards: engine.CardList{s.AttachedCards[0]}}}
					},
				})
			}
			return abs
		},
	})
}

// 60038 Innate Reflexes: your hero gets +1 DEF (the "Starting" option to
// open with it in hand is not modeled).
func registerInnateReflexes() {
	engine.RegisterBehavior("60038", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{DEF: 1}
		},
	})
}

// allyTax is the "costs +1 per ally you control" rider.
func allyTax() func(g *engine.Game, p *engine.Player, def *data.CardDef) int {
	return func(g *engine.Game, p *engine.Player, def *data.CardDef) int {
		return -len(p.Allies)
	}
}

// 60048 Army of One: costs +1 per ally you control; ready your hero.
func registerArmyOfOne() {
	engine.RegisterBehavior("60048", &engine.Behavior{
		CardCost: allyTax(),
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.ReadyEntity{ID: e.EOwner()}}
		},
	})
}

// 60050 In Harm's Way: costs +1 per ally you control; deal X damage to an
// enemy and remove X threat from a scheme, where X is your DEF.
func registerInHarmsWay() {
	engine.RegisterBehavior("60050", &engine.Behavior{
		CardCost: allyTax(),
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			x := p.DefenseStat(g)
			if x <= 0 {
				return nil
			}
			if len(g.Enemies()) == 0 || len(g.Schemes()) == 0 {
				return nil
			}
			var first []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				enemy := g.Entity(id)
				var inner []engine.Choice
				for _, sid := range g.Schemes() {
					s := g.Entity(sid)
					inner = append(inner, engine.Choice{
						Label: engine.S(s.EDef().Name), Kind: engine.ChoiceTarget,
						SourceID: sid, CardCode: s.ECode(),
					}.Msgs(
						engine.DamageEntity{Target: id, Damage: x, Source: pid},
						engine.ThwartScheme{Scheme: sid, N: x, Source: pid},
					))
				}
				first = append(first, engine.Choice{
					Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: enemy.ECode(),
				}.WithThen(engine.Ask(engine.Tf("c.inHarmSWayChooseAScheme"), inner...)))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.inHarmSWayChooseAnEnemy"), first...),
			}}
		},
	})
}

// 60052 The Best Offense...: +1 DEF; Temporary (returns to hand at the
// end of the round). The "use DEF in place of THW and ATK" rider is not
// modeled (approximation).
func registerBestOffense() {
	engine.RegisterBehavior("60052", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{DEF: 1}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.EndRound); !ok {
				return nil
			}
			u, ok := e.(*engine.Upgrade)
			if !ok {
				return nil
			}
			return []engine.Message{engine.ReturnControlled{Player: u.Owner, ID: u.ID}}
		},
	})
}

// 60053 Ronin: while you control no allies, +1 DEF and retaliate 1.
func registerRonin() {
	engine.RegisterBehavior("60053", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			if len(p.Allies) == 0 {
				return engine.StatBonus{DEF: 1, Retaliate: 1}
			}
			return engine.StatBonus{}
		},
	})
}

// 60054 Stand Alone: hero interrupt — when an enemy attacks you, if you
// control no allies, exhaust → ready your hero.
func registerStandAlone() {
	engine.RegisterBehavior("60054", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil || len(p.Allies) > 0 {
				return nil
			}
			return []engine.Ability{{
				Label:    engine.Tf("c.standAloneReadyYourHero"),
				Type:     engine.AbilityTrigger,
				Trigger:  engine.TriggerVillainAttacksYou,
				Exhaust:  true,
				HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.ReadyEntity{ID: e.EOwner()}}
				},
			}}
		},
	})
}
