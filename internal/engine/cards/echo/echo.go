// Package echo registers Echo (Maya Lopez) from Fear No Evil: the Watch
// and Learn event-tucking mechanic and her signature cards. Tucked events
// are stored on the player's side deck slot.
package echo

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerEcho()
	registerSignatures()
	registerEchoCards()
}

// registerEcho installs the identity (60037a/b).
func registerEcho() {
	engine.RegisterBehavior("60037", &engine.Behavior{
		// Watch and Learn — after a player plays an aspect or basic
		// event, tuck it under Echo (max 3 tucked; approximation: stored
		// in the side deck slot, auto-tucked).
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.EventPlayed)
			if !ok {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil {
				return nil
			}
			def := m.Card.Def()
			if def.Aspect == "" { // only aspect or basic events
				return nil
			}
			echo := g.Player(e.EID())
			if echo == nil {
				return nil
			}
			// the event must be in that player's discard to tuck
			if _, ok := p.Discard.Find(m.Card.ID); !ok {
				return nil
			}
			if len(echo.SenseDeck) >= 3 {
				return nil
			}
			if _, ok := p.Discard.Remove(m.Card.ID); ok {
				echo.SenseDeck = append(echo.SenseDeck, m.Card)
				g.TLogf("c.echoTucksTucked", def.Name, len(echo.SenseDeck))
			}
			return nil
		},
	})
}

func registerSignatures() {
	// 60040 Photographic Reflexes: discard it from hand → play an event
	// tucked under Echo with its cost reduced by 2 (approximation: take a
	// tucked event to hand at a 2 discount via CostDiscounts).
	engine.RegisterBehavior("60040", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil || len(p.SenseDeck) == 0 {
				return nil
			}
			var choices []engine.Choice
			for _, c := range p.SenseDeck {
				choices = append(choices, engine.Choice{
					Label: engine.S("Take " + c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code,
				}.Msgs(engine.SideDeckToHand{Player: pid, CardID: c.ID}))
			}
			choices = append(choices, cardutil.Skip())
			return []engine.Message{engine.AskQuestion{
				Player: pid,
				Question: engine.Ask(engine.Tf("c.photographicReflexesTakeATuckedEventToHandItsCostIsReducedBy"),
					choices...),
			}}
		},
	})

	// 60041 Study the Tape: take an aspect/basic event from any discard
	// pile to hand.
	engine.RegisterBehavior("60041", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			var choices []engine.Choice
			for _, pl := range g.Players {
				for _, c := range pl.Discard {
					def := c.Def()
					if def.Type != "event" || def.Aspect == "" {
						continue
					}
					choices = append(choices, engine.Choice{
						Label: engine.S(def.Name + " (" + pl.Name + ")"), Kind: engine.ChoiceCard, CardCode: def.Code,
					}.Msgs(engine.RecycleFromDiscard{Player: pid, From: pl.ID, CardID: c.ID}))
				}
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: pid, Question: engine.Ask(engine.Tf("c.studyTheTapeTakeAnEvent"), choices...)}}
		},
	})

	// 60042 The Rez: AE exhaust → heal X, X = highest cost among tucked
	// cards.
	engine.RegisterBehavior("60042", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil || len(p.SenseDeck) == 0 {
				return nil
			}
			return []engine.Ability{{
				Label:        engine.Tf("c.theRezHeal1PerTheHighestTuckedCost"),
				Type:         engine.AbilityAction,
				Exhaust:      true,
				AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					best := 0
					for _, c := range p.SenseDeck {
						if n := cardutil.Cost(c.Def()); n > best {
							best = n
						}
					}
					return []engine.Message{engine.HealEntity{Target: p.ID, N: best}}
				},
			}}
		},
	})

	// 60043 American Sign Language: resource — [wild] for any player's
	// event (approximation: own events).
	engine.RegisterBehavior("60043", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "wild", EventOnly: true},
	})

	// 60044 Choreography: shuffle an aspect/basic event from discard into
	// your deck; AE: draw 1.
	engine.RegisterBehavior("60044", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			hasEvent := false
			for _, c := range p.Discard {
				if c.Def().Type == "event" && c.Def().Aspect != "" {
					hasEvent = true
					break
				}
			}
			if !hasEvent {
				return nil
			}
			return []engine.Ability{{
				Label:   engine.Tf("c.choreographyShuffleAnEventIntoYourDeck"),
				Type:    engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					var choices []engine.Choice
					for _, c := range p.Discard {
						if c.Def().Type != "event" || c.Def().Aspect == "" {
							continue
						}
						msgs := []engine.Message{engine.ShuffleIntoDeck{Player: p.ID, CardID: c.ID}}
						if !p.IsHero() {
							msgs = append(msgs, engine.DrawCards{Player: p.ID, N: 1})
						}
						choices = append(choices, engine.Choice{
							Label: engine.S("Shuffle in " + c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code,
						}.Msgs(msgs...))
					}
					if len(choices) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.choreographyWhichEvent"), choices...)}}
				},
			}}
		},
	})

	// 60045 Echo's Katana: after you play an event, exhaust → deal damage
	// to an enemy equal to the event's printed cost.
	engine.RegisterBehavior("60045", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.EventPlayed)
			u, ok2 := e.(*engine.Upgrade)
			if !ok || !ok2 || m.Player != u.Owner || u.Exhausted {
				return nil
			}
			p := g.Player(u.Owner)
			if p == nil {
				return nil
			}
			n := cardutil.Cost(m.Card.Def())
			if n <= 0 {
				return nil
			}
			choices := cardutil.EnemyChoices(g, n, p.ID, func(t engine.EntityID) []engine.Message {
				return []engine.Message{engine.DamageEntity{Target: t, Damage: n, Source: p.ID}}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player: p.ID,
				Question: engine.Ask(engine.Tf("c.echoSKatanaDealDamageEqualToTheEventSCost"),
					engine.Choice{ID: "hit", Label: engine.Tf("c.hit"), Kind: engine.ChoiceLabel}.WithThen(
						engine.Ask(engine.Tf("q.chooseEnemy"), choices...)),
					engine.Choice{ID: "skip", Label: engine.Tf("c.skip"), Kind: engine.ChoicePass},
				),
			}}
		},
	})

	// 60046 Improvisation: after you play an event — Attack: heal 1;
	// Defense: remove 1 threat; Thwart: deal 1 damage.
	engine.RegisterBehavior("60046", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.EventPlayed)
			if !ok || m.Player != e.EOwner() {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil {
				return nil
			}
			def := m.Card.Def()
			switch {
			case def.HasTrait("attack"):
				return []engine.Message{engine.HealEntity{Target: p.ID, N: 1}}
			case def.HasTrait("defense"):
				if g.MainScheme != nil {
					return []engine.Message{engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 1, Source: p.ID}}
				}
			case def.HasTrait("thwart"):
				if enemies := cardutil.SortedEnemyIDs(g); len(enemies) > 0 {
					return []engine.Message{engine.DamageEntity{Target: enemies[0], Damage: 1, Source: p.ID}}
				}
			}
			return nil
		},
	})

	// 60047 Muscle Memory: exhaust → take a tucked event to hand.
	engine.RegisterBehavior("60047", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil || len(p.SenseDeck) == 0 {
				return nil
			}
			return []engine.Ability{{
				Label:   engine.Tf("c.muscleMemoryTakeATuckedEventToHand"),
				Type:    engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					var choices []engine.Choice
					for _, c := range p.SenseDeck {
						choices = append(choices, engine.Choice{
							Label: engine.S(c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code,
						}.Msgs(engine.SideDeckToHand{Player: p.ID, CardID: c.ID}))
					}
					if len(choices) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.muscleMemoryWhichEvent"), choices...)}}
				},
			}}
		},
	})
}
