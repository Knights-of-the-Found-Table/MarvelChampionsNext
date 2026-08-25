package core

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// registerCoreObligations installs the five Core Set hero obligations. They
// share the template "flip to alter-ego form, then exhaust the identity to
// remove the obligation or take the penalty"; only the penalty differs.
func registerCoreObligations() {
	// 01155 Affairs of State (Black Panther): discard a Black Panther
	// upgrade you control.
	engine.RegisterBehavior("01155", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var choices []engine.Choice
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil && u.EDef().HasTrait("black panther") {
					choices = append(choices, engine.Choice{
						Label: engine.S("Discard " + u.EDef().Name), Kind: engine.ChoiceCard, CardCode: u.Code,
					}.Msgs(engine.DiscardControlled{Player: p.ID, ID: id}))
				}
			}
			var penalty []engine.Message
			if len(choices) > 0 {
				penalty = append(penalty, engine.AskQuestion{
					Player:   p.ID,
					Question: engine.Ask(engine.Tf("c.affairsOfStateDiscardABlackPantherUpgrade"), choices...),
				})
			}
			return cardutil.ExhaustOrPenalty(g, p, card, engine.S("Discard a Black Panther upgrade you control"), penalty...)
		},
	})

	// 01160 Legal Work (She-Hulk): the main scheme gains 1 acceleration
	// token.
	engine.RegisterBehavior("01160", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var penalty []engine.Message
			if g.MainScheme != nil {
				penalty = append(penalty, engine.AddAccelerationToken{Scheme: g.MainScheme.ID})
			}
			return cardutil.ExhaustOrPenalty(g, p, card, engine.S("Give the main scheme 1 acceleration token"), penalty...)
		},
	})

	// 01165 Eviction Notice (Spider-Man): discard 1 card at random from
	// your hand; this card gains surge.
	engine.RegisterBehavior("01165", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var penalty []engine.Message
			if len(p.Hand) > 0 {
				// random pick resolved when the question is built
				// (deterministic via the game's RNG)
				c := p.Hand[g.Random(len(p.Hand))]
				penalty = append(penalty, engine.DiscardCards{
					Player: p.ID, Cards: engine.CardList{c},
				})
			}
			penalty = append(penalty, engine.RevealNextEncounter{Player: p.ID})
			return cardutil.ExhaustOrPenalty(g, p, card, engine.S("Discard 1 card at random; surge"), penalty...)
		},
	})

	// 01170 Business Problems (Iron Man): exhaust each upgrade you control.
	engine.RegisterBehavior("01170", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var penalty []engine.Message
			for _, id := range p.Upgrades {
				penalty = append(penalty, engine.ExhaustEntity{ID: id})
			}
			return cardutil.ExhaustOrPenalty(g, p, card, engine.S(fmt.Sprintf("Exhaust each upgrade you control (%d)", len(penalty))), penalty...)
		},
	})

	// 01175 Family Emergency (Captain Marvel): you are stunned; surge.
	engine.RegisterBehavior("01175", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			penalty := []engine.Message{
				engine.StunEntity{Target: p.ID},
				engine.RevealNextEncounter{Player: p.ID},
			}
			return cardutil.ExhaustOrPenalty(g, p, card, engine.S("You are stunned; surge"), penalty...)
		},
	})
}
