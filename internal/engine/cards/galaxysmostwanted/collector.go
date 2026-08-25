package galaxysmostwanted

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// registerCollectorScenario installs the Collector scenarios: Infiltrate
// the Museum (16070–16079) and The Missing Milano / Escape the Museum
// (16080–16087). The Collection intercept for cards leaving play is
// engine-side (Game.cardLeavesPlay).
func registerCollectorScenario() {
	// Collector I (16070): the Biogram Image save lives here.
	engine.RegisterBehavior("16070", &engine.Behavior{
		VillainDamageable: biogramSave,
	})

	// Collector II (16071): on reveal, each player feeds the Collection
	// or takes 3 damage.
	engine.RegisterBehavior("16071", &engine.Behavior{
		VillainDamageable: biogramSave,
		VillainStage: func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, collectorToll(p)...)
			}
			return msgs
		},
	})

	// Collector III (16072): on reveal, everyone's top card feeds the
	// Collection; 1 threat per card collected (the leave-play rider adds
	// threat engine-side).
	engine.RegisterBehavior("16072", &engine.Behavior{
		VillainDamageable: biogramSave,
		VillainStage: func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				if len(p.Deck) == 0 {
					continue
				}
				c := p.Deck[0]
				p.Deck = p.Deck[1:]
				msgs = append(msgs, engine.CollectCard{Card: c})
				if g.MainScheme != nil {
					msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: v.ID})
				}
			}
			return msgs
		},
	})

	// 16073 The Grand Collection (stage 1): create the Collection with
	// everyone's top card.
	engine.RegisterBehavior("16073", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, topDeckCollect(p)...)
			}
			return msgs
		},
	})

	// 16074 Biogram Image: attach to the Collector; while attached the
	// save hook prevents all damage at the cost of the image (see
	// biogramSave). The card itself only carries the attach preference.
	engine.RegisterBehavior("16074", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil && v.Code[:5] == "16070" {
					t.Target = id
					g.TLogf("c.biogramImageAttachesToTheCollector")
					return nil
				}
			}
			for id := range g.Villains {
				t.Target = id
				break
			}
			return nil
		},
	})

	// 16075 Monarch Starstalker: Villainous keyword (engine-handled).
	engine.RegisterBehavior("16075", &engine.Behavior{})

	// 16076 Inconspicuous Box: lowest-cost card you control → Collection
	// (else surge). Boost: small Collection → your top card joins it.
	engine.RegisterBehavior("16076", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			lowest, lowCost := engine.EntityID(""), 99
			for _, id := range append(append([]engine.EntityID{}, p.Supports...), p.Upgrades...) {
				def := g.Entity(id).EDef()
				if c := cardutil.Cost(def); c < lowCost {
					lowest, lowCost = id, c
				}
			}
			if lowest == "" {
				g.TLogf("c.inconspicuousBoxHasNoTargetSurge")
				if c, ok := g.DrawEncounter(); ok {
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
				return nil
			}
			e := g.Entity(lowest)
			code := e.ECode()
			g.Delete(lowest)
			g.TLogf("c.isTakenIntoTheCollection", e)
			return []engine.Message{engine.CollectCard{Card: engine.Card{ID: g.NextCardID(), Code: code}}}
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if len(g.Collection) <= 3*len(g.Players) {
				for _, p := range g.Players {
					if len(p.Deck) > 0 {
						c := p.Deck[0]
						p.Deck = p.Deck[1:]
						return []engine.Message{engine.CollectCard{Card: c}}
					}
				}
			}
			return nil
		},
	})

	// 16077 View the Cosmos: feed your highest-cost hand card to the
	// Collection, or discard it and scheme for its cost.
	engine.RegisterBehavior("16077", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			hi, hiCost := "", -1
			for _, c := range p.Hand {
				if cost := cardutil.Cost(c.Def()); cost > hiCost {
					hi, hiCost = c.ID, cost
				}
			}
			if hi == "" || g.MainScheme == nil {
				return nil
			}
			card, _ := p.Hand.Find(hi)
			feed := engine.Choice{
				ID: "feed", Label: engine.Tf("c.putYourHighestCostCardIntoTheCollection"), Kind: engine.ChoiceLabel,
			}.Msgs(engine.CollectCard{Card: card},
				engine.DiscardCards{Player: p.ID, Cards: engine.CardList{card}})
			burn := engine.Choice{
				ID: "burn", Label: engine.Tf("c.discardItAndPlaceThreatEqualToItsCost"), Kind: engine.ChoiceLabel,
			}.Msgs(engine.DiscardCards{Player: p.ID, Cards: engine.CardList{card}},
				engine.SchemeThreat{Scheme: g.MainScheme.ID, N: hiCost, Source: t.ID})
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.viewTheCosmosChooseOne"), feed, burn)}}
		},
	})

	// 16078 Stay Awhile: alter-ego — pay [physical][physical]
	// (approximated as discarding 2 cards) or feed the Collection; hero —
	// the Collector attacks you.
	engine.RegisterBehavior("16078", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			if !p.IsHero() {
				var feed = engine.Choice{
					ID: "feed", Label: engine.Tf("c.putTheTopCardOfYourDeckIntoTheCollection"), Kind: engine.ChoiceLabel,
				}.Msgs(topDeckCollect(p)...)
				var opts []engine.Choice
				opts = append(opts, feed)
				if len(p.Hand) >= 2 {
					opts = append(opts, engine.Choice{
						ID: "pay", Label: engine.Tf("c.discard2CardsApproximatesSpendingPhysicalPhysical"), Kind: engine.ChoiceLabel,
					}.Msgs(engine.DiscardCards{Player: p.ID, Cards: engine.CardList{p.Hand[0], p.Hand[1]}}))
				}
				return []engine.Message{engine.AskQuestion{Player: p.ID,
					Question: engine.Ask(engine.Tf("c.stayAwhileChooseOne"), opts...)}}
			}
			for id := range g.Villains {
				return []engine.Message{engine.VillainActivates{VillainID: id, Player: p.ID}}
			}
			return nil
		},
	})

	// 16079 Caught Off Guard: alias core 01188.
	if b := engine.LookupBehavior("01188"); b != nil {
		engine.RegisterBehavior("16079", b)
	}

	registerEscapeTheMuseum()
}

// biogramSave implements the Biogram Image (16074) forced interrupt: while
// it is attached, all damage to the Collector is prevented, the image goes
// into The Collection and threat equal to the prevented amount schemes.
func biogramSave(g *engine.Game, v *engine.Villain, damage int) bool {
	for id, a := range g.Attachments {
		if a == nil || a.Target != v.ID || a.Code[:5] != "16074" {
			continue
		}
		g.Delete(id)
		g.Collection = append(g.Collection, engine.Card{ID: g.NextCardID(), Code: a.Code})
		if g.MainScheme != nil {
			g.Push(engine.SchemeThreat{Scheme: g.MainScheme.ID, N: damage, Source: id})
		}
		g.TLogMajorf("c.biogramImagePreventsDamageToTheCollector", damage)
		return false
	}
	return true
}

// registerEscapeTheMuseum installs The Missing Milano stages (16080–
// 16087).
func registerEscapeTheMuseum() {
	// Collector A1/B1: stats scale with the main scheme stage; "death"
	// flips the card instead after removing 3[per_hero] threat.
	flip := func(next string) *engine.Behavior {
		return &engine.Behavior{
			VillainDamageable: func(g *engine.Game, v *engine.Villain, damage int) bool {
				if v.HP()-damage > 0 {
					return true
				}
				if g.MainScheme != nil {
					n := min(3*len(g.Players), g.MainScheme.Threat)
					g.Push(engine.ThwartScheme{Scheme: g.MainScheme.ID, N: n, Source: v.ID})
				}
				v.Code = next
				v.Damage = 0
				def := v.EDef()
				if def.HP != nil {
					v.MaxHP = *def.HP
				}
				if def.Scheme != nil {
					v.SchemeVal = *def.Scheme
				}
				if def.Attack != nil {
					v.AttackVal = *def.Attack
				}
				g.TLogMajorf("c.theCollectorSlipsAwayAndFlipsTo", def.StageLabel)
				return false
			},
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				// +X SCH/+X ATK where X = main scheme stage, applied
				// before activation resolution reads the stats.
				if m, ok := msg.(engine.VillainActivates); ok && m.VillainID == e.EID() && g.MainScheme != nil {
					v := g.Villains[m.VillainID]
					def := v.EDef()
					x := g.MainScheme.Stage
					if def.Scheme != nil {
						v.SchemeVal = *def.Scheme + x
					}
					if def.Attack != nil {
						v.AttackVal = *def.Attack + x
					}
				}
				return nil
			},
		}
	}
	engine.RegisterBehavior("16080", flip("16081a"))
	engine.RegisterBehavior("16081", flip("16080a"))

	// 16082 The Missing Milano (stage 1): Library Labyrinth enters play;
	// the Ship Command set is set aside (removed from the encounter deck).
	engine.RegisterBehavior("16082", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			if findEnvironment(g, "16085") == "" {
				g.SpawnEnvironment("16085a")
			}
			var kept engine.CardList
			for _, c := range g.EncounterDeck {
				if c.Def().CardSet == "ship_command" {
					g.SetAside = append(g.SetAside, c)
					g.TLogf("c.isSetAside", c)
				} else {
					kept = append(kept, c)
				}
			}
			g.EncounterDeck = kept
			return nil
		},
	})

	// 16083 Lost in the Museum (stage 2): the set-aside Milano joins the
	// first player.
	engine.RegisterBehavior("16083", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			if _, mil := findMilano(g); mil == nil {
				g.SpawnSupport("16142", cardutil.FirstPlayerID(g))
			}
			return nil
		},
	})

	// 16084 The Great Escape (stage 3): acceleration token; the remaining
	// set-aside Ship Command cards shuffle in.
	engine.RegisterBehavior("16084", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			g.EncounterDeck = append(g.EncounterDeck, g.SetAside...)
			g.SetAside = nil
			g.ShuffleEncounterDeck()
			if s != nil {
				return []engine.Message{engine.AddAccelerationToken{Scheme: s.ID}}
			}
			return nil
		},
	})

	// 16085 Library Labyrinth: "This way?" — take a facedown encounter
	// card to remove 5 threat (once per round).
	engine.RegisterBehavior("16085", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			if g.MainScheme == nil || g.MainScheme.Threat <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.take1FacedownEncounterCardRemove5ThreatFromTheMainScheme"), Type: engine.AbilityAction,
				HeroOnly: true, OncePerRound: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{
						engine.DealEncounterToPlayer{Player: g.ActiveTurn},
						engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 5, Source: self},
					}
				},
			}}
		},
	})

	// 16086 "I Have You Now!": alter-ego — exhaust and the Collector
	// schemes; hero — stunned and attacked. Boost: Collector tough.
	engine.RegisterBehavior("16086", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for id := range g.Villains {
				if !p.IsHero() {
					return []engine.Message{
						engine.ExhaustEntity{ID: p.ID},
						engine.VillainActivates{VillainID: id, Player: p.ID},
					}
				}
				return []engine.Message{
					engine.StunEntity{Target: p.ID},
					engine.VillainActivates{VillainID: id, Player: p.ID},
				}
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil {
					v.Tough = true
					g.TLogf("c.gainsAToughStatusCard", v)
				}
			}
			return nil
		},
	})

	// 16087 Impossible Geometry: incite 1; confused (or discard a card
	// you control when already confused).
	engine.RegisterBehavior("16087", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var msgs []engine.Message
			if g.MainScheme != nil {
				msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: t.ID})
			}
			if p.Confused {
				if len(p.Upgrades) > 0 {
					msgs = append(msgs, engine.DiscardControlled{Player: p.ID, ID: p.Upgrades[len(p.Upgrades)-1]})
				} else if len(p.Supports) > 0 {
					msgs = append(msgs, engine.DiscardControlled{Player: p.ID, ID: p.Supports[len(p.Supports)-1]})
				}
			} else {
				msgs = append(msgs, engine.ConfuseEntity{Target: p.ID})
			}
			return msgs
		},
	})
}

// collectorToll asks one player: top card of deck → Collection, or take 3
// damage.
func collectorToll(p *engine.Player) []engine.Message {
	feed := engine.Choice{
		ID: "feed", Label: engine.Tf("c.putTheTopCardOfYourDeckIntoTheCollection"), Kind: engine.ChoiceLabel,
	}
	if len(p.Deck) > 0 {
		feed = feed.Msgs(topDeckCollect(p)...)
	}
	return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(
		engine.Tf("c.theCollectorDemandsTributeChooseOne"),
		feed,
		engine.Choice{
			ID: "dmg", Label: engine.Tf("c.take3Damage"), Kind: engine.ChoiceLabel,
		}.Msgs(engine.DamageEntity{Target: p.ID, Damage: 3}),
	)}}
}

// topDeckCollect builds the message that moves the player's deck top into
// The Collection (the handler removes it from the deck when it resolves).
func topDeckCollect(p *engine.Player) []engine.Message {
	if len(p.Deck) == 0 {
		return nil
	}
	return []engine.Message{engine.CollectCard{Card: p.Deck[0]}}
}
