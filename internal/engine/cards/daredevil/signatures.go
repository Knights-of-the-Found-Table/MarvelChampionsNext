package daredevil

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// registerSignatures installs Daredevil's signature cards (60007-60018).
func registerSignatures() {
	registerElektra()
	registerCrossExamination()
	registerDeposition()
	registerLivingLieDetector()
	registerRaisingHell()
	registerFocusTheSenses()
	registerFoggyNelson()
	registerKarenPage()
	registerNelsonAndMurdock()
	registerSisterMaggie()
	registerBillyClub()
	registerManWithoutFear()
}

// 60007 Elektra: forced interrupt — when she would take consequential
// damage, Daredevil takes it instead.
func registerElektra() {
	engine.RegisterBehavior("60007", &engine.Behavior{
		ConsequentialToOwner: true,
	})
}

// 60008 Cross-Examination: deal 3 damage to an enemy, +1 per upgrade
// attached to it.
func registerCrossExamination() {
	engine.RegisterBehavior("60008", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			choices := cardutil.EnemyChoices(g, 0, pid, func(target engine.EntityID) []engine.Message {
				dmg := 3 + upgradesAttachedTo(g, target)
				return []engine.Message{engine.DamageEntity{Target: target, Damage: dmg, Source: pid}}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.crossExaminationDeal3Damage1PerAttachedUpgrade"), choices...),
			}}
		},
	})
}

// chooseSenseFree builds a question to play any Sense upgrade from the
// deck, ignoring its cost.
func chooseSenseFree(g *engine.Game, p *engine.Player, prompt string) []engine.Message {
	if len(p.SenseDeck) == 0 {
		return nil
	}
	var choices []engine.Choice
	for _, c := range p.SenseDeck {
		choices = append(choices, engine.Choice{
			Label: engine.S("Play " + c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code,
		}.Msgs(engine.SenseEnterPlay{Player: p.ID, Card: c}))
	}
	choices = append(choices, cardutil.Skip())
	return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.S(prompt), choices...)}}
}

// 60009 Deposition: alter-ego action — choose any upgrade from the Sense
// deck and play it ignoring its cost; remove 2 threat from a scheme.
func registerDeposition() {
	engine.RegisterBehavior("60009", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			var msgs []engine.Message
			msgs = append(msgs, chooseSenseFree(g, p, "Deposition — play a Sense card for free")...)
			msgs = append(msgs, cardutil.ChooseScheme(engine.Tf("c.chooseAScheme", "Deposition"), func(g *engine.Game, e engine.Entity) int {
				return 2
			})(g, e)...)
			return msgs
		},
	})
}

// 60010 Living Lie Detector: remove 2 threat from a scheme, +1 per
// upgrade attached to it.
func registerLivingLieDetector() {
	engine.RegisterBehavior("60010", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			var choices []engine.Choice
			for _, id := range g.Schemes() {
				s := g.Entity(id)
				n := 2 + upgradesAttachedTo(g, id)
				choices = append(choices, engine.Choice{
					Label: engine.Tf("c.threat", s, n), Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: s.ECode(),
				}.Msgs(engine.ThwartScheme{Scheme: id, N: n, Source: pid}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.livingLieDetectorRemove2Threat1PerAttachedUpgrade"), choices...),
			}}
		},
	})
}

// 60011 Raising Hell: deal 2 damage to each enemy for each upgrade
// attached to it (3 if Daredevil has the Aerial trait).
func registerRaisingHell() {
	engine.RegisterBehavior("60011", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			per := 2
			if p != nil && g.EntityHasTrait(p.ID, "aerial") {
				per = 3
			}
			var msgs []engine.Message
			for _, id := range cardutil.SortedEnemyIDs(g) {
				n := per * upgradesAttachedTo(g, id)
				if n > 0 {
					msgs = append(msgs, engine.DamageEntity{Target: id, Damage: n, Source: pid})
				}
			}
			return msgs
		},
	})
}

// 60012 Focus the Senses: player side scheme (threat 4). When defeated:
// Sense upgrades in play return to the Sense deck and any number of Sense
// upgrades from the deck enter play free (approximation: choose up to 2;
// the only-Daredevil-can-thwart rider is not modeled).
func registerFocusTheSenses() {
	engine.RegisterBehavior("60012", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			s, ok := e.(*engine.SideScheme)
			if !ok {
				return nil
			}
			p := g.Player(s.Owner)
			if p == nil {
				return nil
			}
			// Recall all in-play Sense upgrades to the Sense deck.
			var recall []engine.Message
			for _, uid := range p.Upgrades {
				if u := g.Upgrades[uid]; u != nil && u.EDef().HasTrait("sense") {
					recall = append(recall, engine.DiscardControlled{Player: p.ID, ID: uid})
				}
			}
			var choices []engine.Choice
			for _, c := range p.SenseDeck {
				choices = append(choices, engine.Choice{
					Label: engine.S("Play " + c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code,
				}.Msgs(engine.SenseEnterPlay{Player: p.ID, Card: c}))
			}
			out := recall
			if len(choices) > 0 {
				n := min(2, len(choices))
				out = append(out, engine.AskQuestion{
					Player:   p.ID,
					Question: engine.AskN(engine.Tf("c.focusTheSensesDeploySenseUpgradesForFree"), n, choices...),
				})
			}
			return out
		},
	})
}

// 60013 Foggy Nelson: alter-ego action — exhaust → remove 2 threat.
func registerFoggyNelson() {
	engine.RegisterBehavior("60013", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label:        engine.Tf("c.foggyNelsonRemove2ThreatFromAScheme"),
				Type:         engine.AbilityAction,
				Exhaust:      true,
				AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					pid := e.EOwner()
					var choices []engine.Choice
					for _, id := range g.Schemes() {
						s := g.Entity(id)
						choices = append(choices, engine.Choice{
							Label: engine.S(s.EDef().Name), Kind: engine.ChoiceTarget,
							SourceID: id, CardCode: s.ECode(),
						}.Msgs(engine.ThwartScheme{Scheme: id, N: 2, Source: pid}))
					}
					if len(choices) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{
						Player:   pid,
						Question: engine.Ask(engine.Tf("c.foggyNelsonChooseAScheme"), choices...),
					}}
				},
			}}
		},
	})
}

// 60014 Karen Page: action — exhaust → shuffle 1 Daredevil card from your
// discard pile into your deck; if in alter-ego form, draw 1.
func registerKarenPage() {
	engine.RegisterBehavior("60014", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			hasTarget := false
			for _, c := range p.Discard {
				if c.Def().CardSet == "daredevil" {
					hasTarget = true
					break
				}
			}
			if !hasTarget {
				return nil
			}
			return []engine.Ability{{
				Label:   engine.Tf("c.karenPageShuffleADaredevilCardFromYourDiscardIntoYourDeck"),
				Type:    engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					var choices []engine.Choice
					for _, c := range p.Discard {
						if c.Def().CardSet != "daredevil" {
							continue
						}
						msgs := []engine.Message{engine.ShuffleIntoDeck{Player: p.ID, CardID: c.ID}}
						if !p.IsHero() {
							msgs = append(msgs, engine.DrawCards{Player: p.ID, N: 1})
						}
						choices = append(choices, engine.Choice{
							Label: engine.S(c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code,
						}.Msgs(msgs...))
					}
					if len(choices) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{
						Player:   p.ID,
						Question: engine.Ask(engine.Tf("c.karenPageShuffleWhichDaredevilCardIntoYourDeck"), choices...),
					}}
				},
			}}
		},
	})
}

// 60015 Nelson and Murdock: response — after an Attorney character or
// support defeats a side scheme, confuse an enemy (approximation: any
// side-scheme defeat while you control an Attorney).
func registerNelsonAndMurdock() {
	engine.RegisterBehavior("60015", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok {
				return nil
			}
			s := g.SideSchemes[m.Scheme]
			if s == nil {
				return nil
			}
			u, ok := e.(*engine.Support)
			if !ok {
				return nil
			}
			p := g.Player(u.Owner)
			if p == nil {
				return nil
			}
			attorney := g.EntityHasTrait(p.ID, "attorney")
			for _, id := range p.Supports {
				if sp := g.Supports[id]; sp != nil && sp.EDef().HasTrait("attorney") {
					attorney = true
				}
			}
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && a.EDef().HasTrait("attorney") {
					attorney = true
				}
			}
			if !attorney {
				return nil
			}
			choices := cardutil.EnemyChoices(g, 0, p.ID, func(target engine.EntityID) []engine.Message {
				return []engine.Message{engine.ConfuseEntity{Target: target}}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.nelsonAndMurdockConfuseAnEnemy"), append(choices, cardutil.Skip())...),
			}}
		},
	})
}

// 60016 Sister Maggie: Matt Murdock gets +3 REC; response — after you
// recover, discard a status card from your identity.
func registerSisterMaggie() {
	engine.RegisterBehavior("60016", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{REC: 3}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.BasicRecover)
			u, ok2 := e.(*engine.Support)
			if !ok || !ok2 || m.Player != u.Owner {
				return nil
			}
			p := g.Player(u.Owner)
			if p == nil {
				return nil
			}
			if p.Stunned {
				return []engine.Message{engine.ClearStun{Target: p.ID}}
			}
			if p.Confused {
				return []engine.Message{engine.ClearConfuse{Target: p.ID}}
			}
			return nil
		},
	})
}

// 60017 Daredevil's Billy Club: Daredevil gets +1 ATK; hero action —
// return it to your hand → deal 1 damage to an enemy, or Daredevil gains
// the Aerial trait until the end of the round.
func registerBillyClub() {
	engine.RegisterBehavior("60017", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{ATK: 1}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			damage := engine.Choice{
				ID: "damage", Label: engine.Tf("c.deal1DamageToAnEnemy"), Kind: engine.ChoiceLabel,
			}
			if enemies := cardutil.EnemyChoices(g, 1, p.ID, func(target engine.EntityID) []engine.Message {
				return []engine.Message{engine.DamageEntity{Target: target, Damage: 1, Source: p.ID}}
			}); len(enemies) > 0 {
				damage = damage.WithThen(engine.Ask(engine.Tf("c.billyClubChooseAnEnemy"), enemies...))
			}
			aerial := engine.Choice{
				ID: "aerial", Label: engine.Tf("c.daredevilGainsTheAerialTraitUntilTheEndOfTheRound"), Kind: engine.ChoiceLabel,
			}.Msgs(engine.GrantTrait{Target: p.ID, Trait: "aerial"})
			return []engine.Ability{{
				Label:    engine.Tf("c.billyClubReturnToHand1DamageOrGainAerial"),
				Type:     engine.AbilityAction,
				HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.AskQuestion{
						Player: p.ID,
						Question: engine.Ask(engine.Tf("c.billyClubChoose"),
							damage,
							aerial,
						),
					}, engine.ReturnControlled{Player: p.ID, ID: e.EID()}}
				},
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			// Aerial grant expires at the end of the round.
			if _, ok := msg.(engine.EndRound); !ok {
				return nil
			}
			u, ok := e.(*engine.Upgrade)
			if !ok {
				return nil
			}
			p := g.Player(u.Owner)
			if p == nil {
				return nil
			}
			for i, t := range p.ExtraTraits {
				if t == "aerial" {
					p.ExtraTraits = append(p.ExtraTraits[:i], p.ExtraTraits[i+1:]...)
					return nil
				}
			}
			return nil
		},
	})
}

// 60018 The Man Without Fear: hero action — exhaust and deal 1 damage to
// Daredevil → play any upgrade from the Sense deck for free, or ready
// Daredevil.
func registerManWithoutFear() {
	engine.RegisterBehavior("60018", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			return []engine.Ability{{
				Label:    engine.Tf("c.theManWithoutFear1DamageFreeSenseCardOrReadyDaredevil"),
				Type:     engine.AbilityAction,
				Exhaust:  true,
				HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(self)
					if p == nil {
						return nil
					}
					var choices []engine.Choice
					for _, c := range p.SenseDeck {
						choices = append(choices, engine.Choice{
							Label: engine.S("Play " + c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code,
						}.Msgs(
							engine.DamageEntity{Target: p.ID, Damage: 1, Source: p.ID},
							engine.SenseEnterPlay{Player: p.ID, Card: c},
						))
					}
					choices = append(choices, engine.Choice{
						ID: "ready", Label: engine.Tf("c.readyDaredevil"), Kind: engine.ChoiceLabel,
					}.Msgs(
						engine.DamageEntity{Target: p.ID, Damage: 1, Source: p.ID},
						engine.ReadyEntity{ID: p.ID},
					))
					return []engine.Message{engine.AskQuestion{
						Player:   p.ID,
						Question: engine.Ask(engine.Tf("c.theManWithoutFearChoose"), choices...),
					}}
				},
			}}
		},
	})
}
