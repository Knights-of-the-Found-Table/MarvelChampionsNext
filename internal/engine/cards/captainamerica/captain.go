// Package captainamerica registers the Captain America hero pack: the
// identity, signature cards, aspect cards and the Baron Zemo nemesis set.
package captainamerica

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() {
	registerCaptainAmerica()
	registerSignatures()
	registerAspectCards()
	registerCapNemesis()
	registerCapObligation()
}

// registerCaptainAmerica installs the Captain America / Steve Rogers
// identity (03001a/b).
func registerCaptainAmerica() {
	engine.RegisterBehavior("03001", &engine.Behavior{
		// Living Legend — the first ally played each round costs 1 less.
		CardCost: func(g *engine.Game, p *engine.Player, def *data.CardDef) int {
			if def.Type == "ally" && !p.AllyPlayedThisRound {
				return 1
			}
			return 0
		},
		// Setup: search your deck and discard pile for Captain America's
		// Shield and add it to your hand. Shuffle your deck.
		HeroSetup: func(g *engine.Game, p *engine.Player) []engine.Message {
			for _, c := range p.Discard {
				if c.Code == "03009" {
					if _, ok := p.Discard.Remove(c.ID); ok {
						p.Hand = append(p.Hand, c)
						g.TLogf("c.takesCaptainAmericaSShieldFromTheirDiscardPile", p.Name)
					}
					break
				}
			}
			var msgs []engine.Message
			for _, c := range p.Deck {
				if c.Code == "03009" {
					msgs = append(msgs, engine.TakeDeckCard{Player: p.ID, CardID: c.ID})
					break
				}
			}
			msgs = append(msgs, engine.ShufflePlayerDeck{Player: p.ID})
			return msgs
		},
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			if len(p.Hand) == 0 {
				return nil
			}
			return []engine.Ability{{
				// "I Can Do This All Day!" — Action: discard 1 card from
				// your hand → ready Captain America (limit once per round).
				Label:        engine.Tf("c.iCanDoThisAllDayDiscard1CardReadyCaptainAmerica"),
				Type:         engine.AbilityAction,
				HeroOnly:     true,
				OncePerRound: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(self)
					if p == nil {
						return nil
					}
					var choices []engine.Choice
					for _, c := range p.Hand {
						choices = append(choices, engine.Choice{
							Label: engine.Tf("m.discardCard", c), Kind: engine.ChoiceCard, CardCode: c.Code,
						}.Msgs(
							engine.DiscardCards{Player: p.ID, Cards: engine.CardList{c}},
							engine.ReadyEntity{ID: self},
						))
					}
					return []engine.Message{engine.AskQuestion{
						Player:   p.ID,
						Question: engine.Ask(engine.Tf("c.iCanDoThisAllDayChooseACardToDiscard"), choices...),
					}}
				},
			}}
		},
	})
}

// registerCapNemesis installs the Baron Zemo nemesis encounter set.
func registerCapNemesis() {
	// Hit Squad: When Revealed — in player order, each player discards the
	// top card of the encounter deck and takes 1 damage for each boost
	// icon discarded this way (approximation: seat order).
	engine.RegisterBehavior("03027", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				c, ok := g.DrawEncounter()
				if !ok {
					continue
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
				n := cardutil.BoostOf(c)
				g.TLogf("c.discardsFromTheEncounterDeckBoostIcons", p.Name, c, n)
				if n > 0 {
					msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: n, Source: e.EID()})
				}
			}
			return msgs
		},
	})

	// Baron Zemo: Quickstrike (keyword, generic) + while engaged with you,
	// you cannot thwart.
	engine.RegisterBehavior("03028", &engine.Behavior{
		EngagedBlocksThwart: true,
	})

	// Hydra Soldier: Guard (keyword, generic) + When Defeated — deal the
	// engaged player an encounter card.
	engine.RegisterBehavior("03029", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionDefeated)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			mn, ok := e.(*engine.Minion)
			if !ok || mn.EngagedWith == "" {
				return nil
			}
			return []engine.Message{engine.DealEncounterToPlayer{Player: mn.EngagedWith}}
		},
	})

	// Hail Hydra!: each Hydra minion engaged with a hero attacks that
	// hero; each other player searches deck and discard for a Hydra minion
	// and puts it into play engaged with them.
	engine.RegisterBehavior("03030", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var msgs []engine.Message
			for _, pl := range g.Players {
				attacked := false
				if pl.IsHero() {
					for _, id := range cardutil.SortedIDs(g.Minions) {
						mn := g.Minions[id]
						if mn.EngagedWith == pl.ID && mn.EDef().HasTrait("hydra") {
							msgs = append(msgs, engine.MinionActivates{MinionID: id, Player: pl.ID})
							attacked = true
							break
						}
					}
				}
				if attacked {
					continue
				}
				if c, ok := takeHydraMinion(g, &g.EncounterDeck); ok {
					msgs = append(msgs, engine.RevealEncounterCard{Player: pl.ID, Card: c})
					g.ShuffleEncounterDeck()
					continue
				}
				if c, ok := takeHydraMinion(g, &g.EncounterDiscard); ok {
					msgs = append(msgs, engine.RevealEncounterCard{Player: pl.ID, Card: c})
					g.ShuffleEncounterDeck()
				}
			}
			return msgs
		},
	})
}

// takeHydraMinion removes the first Hydra minion card from a zone.
func takeHydraMinion(g *engine.Game, zone *engine.CardList) (engine.Card, bool) {
	for i, c := range *zone {
		def := c.Def()
		if def.Type == "minion" && def.HasTrait("hydra") {
			*zone = append((*zone)[:i], (*zone)[i+1:]...)
			return c, true
		}
	}
	return engine.Card{}, false
}

// registerCapObligation installs Man Out of Time.
func registerCapObligation() {
	// You may flip to alter-ego form. Choose: exhaust Steve Rogers →
	// remove Man Out of Time from the game, or discard half of your hand
	// rounded down.
	engine.RegisterBehavior("03026", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			n := len(p.Hand) / 2
			var choices []engine.Choice
			for _, c := range p.Hand {
				choices = append(choices, engine.Choice{
					Label: engine.Tf("m.discardCard", c), Kind: engine.ChoiceCard, CardCode: c.Code,
				}.Msgs(engine.DiscardCards{Player: p.ID, Cards: engine.CardList{c}}))
			}
			var penalty []engine.Message
			if n > 0 && len(choices) > 0 {
				penalty = append(penalty, engine.AskQuestion{
					Player:   p.ID,
					Question: engine.AskN(engine.Tf("c.manOutOfTimeDiscardCards", n), n, choices...),
				})
			}
			return cardutil.ExhaustOrPenalty(g, p, card,
				engine.S(fmt.Sprintf("Discard %d cards from your hand", n)), penalty...)
		},
	})
}
