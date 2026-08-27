package core

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// registerHeroes installs identity behaviors for the remaining Core Set
// heroes (Spider-Man lives in spiderman.go).
func registerHeroes() {
	registerCaptainMarvel()
	registerSheHulk()
	registerIronMan()
	registerBlackPanther()
}

// Captain Marvel / Carol Danvers.
func registerCaptainMarvel() {
	engine.RegisterBehavior("01010", &engine.Behavior{
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			return []engine.Ability{
				{
					// Rechannel: Action — pay 1 resource + heal 1 → draw 1
					// (limit once per round).
					Label:        engine.Tf("c.rechannelPay1Heal1DamageDraw1"),
					Type:         engine.AbilityAction,
					Cost:         1,
					HeroOnly:     true,
					OncePerRound: true,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						return []engine.Message{
							engine.HealEntity{Target: self, N: 1},
							engine.DrawCards{Player: self, N: 1},
						}
					},
				},
				{
					// Commander: Action — choose a player to draw 1
					// (solo: you).
					Label:        engine.Tf("c.commanderAPlayerDraws1"),
					Type:         engine.AbilityAction,
					AlterEgoOnly: true,
					OncePerRound: true,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						var msgs []engine.Message
						for _, p := range g.Players {
							msgs = append(msgs, engine.DrawCards{Player: p.ID, N: 1})
						}
						return msgs
					},
				},
			}
		},
	})
}

// She-Hulk / Jennifer Walters. "I Object!" threat prevention is an engine
// approximation inside addThreat.
func registerSheHulk() {
	engine.RegisterBehavior("01019", &engine.Behavior{
		// "Do You Even Lift?" — Response: after changing to hero form,
		// deal 2 damage to an enemy.
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.ChangeForm)
			if !ok || m.Player != e.EID() {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil || !p.IsHero() || len(g.Enemies()) == 0 {
				return nil
			}
			var choices []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				enemy := g.Entity(id)
				choices = append(choices, engine.Choice{
					Label: engine.Tf("m.cardName", enemy), Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: enemy.ECode(),
				}.Msgs(engine.DamageEntity{Target: id, Damage: 2, Source: p.ID}))
			}
			choices = append(choices, engine.Choice{
				ID: "skip", Label: engine.Tf("c.skip"), Kind: engine.ChoicePass,
			})
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.doYouEvenLiftDeal2DamageToAnEnemy"), choices...),
			}}
		},
	})
}

// Iron Man / Tony Stark.
func registerIronMan() {
	engine.RegisterBehavior("01029", &engine.Behavior{
		// +1 hand size per Tech upgrade while in hero form (max 7).
		HandSizeBonus: func(g *engine.Game, p *engine.Player) int {
			if !p.IsHero() {
				return 0
			}
			n := 0
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil && u.EDef().HasTrait("tech") {
					n++
				}
			}
			return n
		},
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			return []engine.Ability{{
				// Futurist — Action: look at top 3 of your deck, keep 1,
				// discard the rest (limit once per round).
				Label:        engine.Tf("c.futuristLookAtTop3Keep1"),
				Type:         engine.AbilityAction,
				AlterEgoOnly: true,
				OncePerRound: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(self)
					if p == nil || len(p.Deck) == 0 {
						return nil
					}
					n := min(3, len(p.Deck))
					var choices []engine.Choice
					for i := 0; i < n; i++ {
						c := p.Deck[i]
						choices = append(choices, engine.Choice{
							Label: engine.Tf("c.keep", c),
							Kind:  engine.ChoiceCard, CardCode: c.Code,
						}.Msgs(engine.TakeDeckCard{Player: self, CardID: c.ID, FromTop: 3}))
					}
					return []engine.Message{engine.AskQuestion{
						Player:   self,
						Question: engine.Ask(engine.Tf("c.futuristKeep1OfTheTop3"), choices...),
					}}
				},
			}}
		},
	})
}

// Black Panther / T'Challa. Retaliate 1 comes from the printed keyword.
func registerBlackPanther() {
	engine.RegisterBehavior("01040", &engine.Behavior{
		// Foresight — Setup: search your deck for a Black Panther
		// upgrade, add it to your hand, shuffle.
		HeroSetup: func(g *engine.Game, p *engine.Player) []engine.Message {
			var choices []engine.Choice
			for _, c := range p.Deck {
				def := c.Def()
				if def.Type == "upgrade" && def.CardSet == "black_panther" {
					choices = append(choices, engine.Choice{
						Label: engine.Tf("c.addToHand", def.Name),
						Kind:  engine.ChoiceCard, CardCode: def.Code,
					}.Msgs(engine.TakeDeckCard{Player: p.ID, CardID: c.ID}))
				}
			}
			if len(choices) == 0 {
				return nil
			}
			choices = append(choices, engine.Choice{
				ID: "skip", Label: engine.Tf("c.skip"), Kind: engine.ChoicePass,
			})
			for i := range choices {
				// shuffle after taking (or skipping)
				choices[i] = choices[i].Msgs(engine.ShufflePlayerDeck{Player: p.ID})
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.foresightTakeABlackPantherUpgrade"), choices...),
			}}
		},
	})
}
