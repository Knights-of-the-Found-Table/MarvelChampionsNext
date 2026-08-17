// Package msmarvel registers the Ms. Marvel hero pack: the identity,
// signature cards, aspect cards and the Thomas Edison nemesis set.
package msmarvel

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerMsMarvel()
	registerSignatures()
	registerAspectCards()
	registerNemesis()
	registerObligation()
}

// registerMsMarvel installs the Ms. Marvel / Kamala Khan identity
// (05001a/b).
func registerMsMarvel() {
	engine.RegisterBehavior("05001", &engine.Behavior{
		// Morphogenetics — Response: after you play an [[Attack]],
		// [[Thwart]] or [[Defense]] event, exhaust Ms. Marvel → return
		// that event to your hand.
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.EventPlayed)
			if !ok || m.Player != e.EID() {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil || p.Exhausted || !p.IsHero() {
				return nil
			}
			def := m.Card.Def()
			if !(def.HasTrait("attack") || def.HasTrait("thwart") || def.HasTrait("defense")) {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player: p.ID,
				Question: engine.Ask(fmt.Sprintf("Morphogenetics — exhaust Ms. Marvel to return %s to your hand?", def.Name),
					engine.Choice{
						ID: "return", Label: "Exhaust Ms. Marvel — return " + def.Name + " to hand",
						Kind: engine.ChoiceLabel,
					}.Msgs(
						engine.ExhaustEntity{ID: p.ID},
						engine.ReturnDiscardCard{Player: p.ID, CardID: m.Card.ID},
					),
					engine.Choice{ID: "skip", Label: "Skip", Kind: engine.ChoicePass},
				),
			}}
		},
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			return []engine.Ability{{
				// Teen Spirit — Action: discard cards from the top of
				// your deck until you discard a Ms. Marvel card, then add
				// it to your hand (limit once per round).
				Label:        "Teen Spirit — mill until a Ms. Marvel card, add it to your hand",
				Type:         engine.AbilityAction,
				AlterEgoOnly: true,
				OncePerRound: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(self)
					if p == nil {
						return nil
					}
					var discarded engine.CardList
					found := engine.Card{}
					for _, c := range p.Deck {
						if c.Def().CardSet == "ms_marvel" {
							found = c
							break
						}
						discarded = append(discarded, c)
					}
					if found.Code != "" {
						p.Deck = p.Deck[len(discarded)+1:]
						p.Hand = append(p.Hand, found)
						g.Logf("Teen Spirit finds %s", found.Def().Name)
					} else {
						p.Deck = p.Deck[len(discarded):]
						g.Logf("Teen Spirit discards the whole deck and finds nothing")
					}
					if len(discarded) > 0 {
						return []engine.Message{engine.DiscardCards{Player: p.ID, Cards: discarded}}
					}
					return nil
				},
			}}
		},
	})
}

// registerNemesis installs the Thomas Edison nemesis encounter set.
func registerNemesis() {
	// 05026 Generation Why?: When Revealed — discard the top card of each
	// player's deck for each ally and [[Persona]] support in play.
	engine.RegisterBehavior("05026", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			n := 0
			for _, p := range g.Players {
				n += len(p.Allies)
				for _, id := range p.Supports {
					if s := g.Supports[id]; s != nil && isPersonaSupport(s) {
						n++
					}
				}
			}
			if n == 0 {
				return nil
			}
			g.Logf("Generation Why? discards %d cards from each player's deck", n)
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, engine.MillPlayerDeck{Player: p.ID, N: n})
			}
			return msgs
		},
	})

	// 05027 Thomas Edison: cannot take damage while you are engaged with
	// another minion.
	engine.RegisterBehavior("05027", &engine.Behavior{
		MinionDamageable: func(g *engine.Game, mn *engine.Minion, damage int) bool {
			if mn.EngagedWith == "" {
				return true
			}
			for _, other := range g.Minions {
				if other.ID != mn.ID && other.EngagedWith == mn.EngagedWith {
					g.Logf("%s cannot take damage while another minion is engaged", mn.EDef().Name)
					return false
				}
			}
			return true
		},
	})

	// 05028 Edison's Giant Robot: cannot take damage; hero action spend a
	// [mental] resource → treat its text box as blank until end of phase
	// (approximation: blanking only disables the damage immunity).
	engine.RegisterBehavior("05028", &engine.Behavior{
		MinionDamageable: func(g *engine.Game, mn *engine.Minion, damage int) bool {
			if mn.BlankText {
				return true
			}
			g.Logf("Edison's Giant Robot cannot take damage")
			return false
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			if mn, ok := e.(*engine.Minion); ok && mn.BlankText {
				return nil
			}
			return []engine.Ability{{
				Label:    "Spend a resource — blank Edison's Giant Robot until the end of the phase",
				Type:     engine.AbilityAction,
				HeroOnly: true,
				Cost:     1,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					if mn := g.Minions[self]; mn != nil {
						mn.BlankText = true
						g.Logf("Edison's Giant Robot's text box is blank until the end of the phase")
					}
					return nil
				},
			}}
		},
	})

	// 05029 Harvest: exhaust each [[Persona]] support in play; the villain
	// heals 1 for each support exhausted this way. If none, surge.
	engine.RegisterBehavior("05029", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			n := 0
			for _, pl := range g.Players {
				for _, id := range pl.Supports {
					if s := g.Supports[id]; s != nil && isPersonaSupport(s) && !s.Exhausted {
						s.Exhausted = true
						n++
					}
				}
			}
			var msgs []engine.Message
			if n > 0 {
				for id := range g.Villains {
					v := g.Villains[id]
					heal := min(n, v.Damage)
					if heal > 0 {
						v.Damage -= heal
						g.Logf("%s heals %d damage (Harvest)", v.EDef().Name, heal)
					}
					break
				}
				g.Logf("Harvest exhausts %d Persona supports", n)
				return msgs
			}
			// no support exhausted: surge
			return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
		},
	})
}

// isPersonaSupport reports whether a support carries the printed Persona
// trait (Aamir, Bruno, Nakia, Aunt May, Steve's Apartment...).
func isPersonaSupport(s *engine.Support) bool {
	return s.EDef().HasTrait("persona")
}

// registerObligation installs Home by Dawn.
func registerObligation() {
	// You may flip to alter-ego form. Choose: exhaust Kamala Khan →
	// remove Home by Dawn from the game, or discard 1 [[Persona]] support
	// you control (surge if none was discarded).
	engine.RegisterBehavior("05025", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var supportChoices []engine.Choice
			for _, id := range p.Supports {
				if s := g.Supports[id]; s != nil && isPersonaSupport(s) {
					supportChoices = append(supportChoices, engine.Choice{
						Label: "Discard " + s.EDef().Name, Kind: engine.ChoiceCard, CardCode: s.Code,
					}.Msgs(engine.DiscardControlled{Player: p.ID, ID: id}))
				}
			}
			penalty := []engine.Message{}
			if len(supportChoices) > 0 {
				penalty = append(penalty, engine.AskQuestion{
					Player:   p.ID,
					Question: engine.Ask("Home by Dawn — discard a Persona support", supportChoices...),
				})
			} else {
				penalty = append(penalty, engine.RevealNextEncounter{Player: p.ID})
			}
			return cardutil.ExhaustOrPenalty(g, p, card, "Discard 1 Persona support you control (surge if none)", penalty...)
		},
	})
}
