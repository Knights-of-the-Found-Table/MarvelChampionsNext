// Package aos registers the Agents of S.H.I.E.L.D. hero pack.
package aos

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerMariaHill()
	registerMariaSignatures()
	registerMariaNemesis()
	registerMariaObligation()
}

// registerMariaHill installs Maria Hill (50001a/b).
func registerMariaHill() {
	engine.RegisterBehavior("50001", &engine.Behavior{
		// Maria's constant ability grants S.H.I.E.L.D. to every ally she
		// controls. There is no ally-entered-play notification for normally
		// paid allies, so the identity keeps the trait synchronized whenever
		// the game processes a message.
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			p := g.Player(e.EID())
			if p == nil {
				return nil
			}
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil {
					addTraitOnce(a, shieldTrait)
				}
			}
			return nil
		},
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			return []engine.Ability{
				{
					Label: engine.Tf("c.reassignmentMove1AllPurposeCounterBetweenSHIELDSupports"),
					Type:  engine.AbilityAction, HeroOnly: true, OncePerRound: true,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						pl := g.Player(self)
						var choices []engine.Choice
						for _, from := range shieldSupports(g, pl) {
							if from.Counters <= 0 {
								continue
							}
							for _, to := range shieldSupports(g, pl) {
								if from.ID == to.ID {
									continue
								}
								choices = append(choices, engine.Choice{
									Label: engine.S(fmt.Sprintf("%s → %s", from.EDef().Name, to.EDef().Name)),
									Kind:  engine.ChoiceTarget, SourceID: to.ID, CardCode: to.Code,
								}.Msgs(
									engine.AddEntityCounter{ID: from.ID, N: -1},
									engine.AddEntityCounter{ID: to.ID, N: 1},
								))
							}
						}
						if len(choices) == 0 || pl == nil {
							return nil
						}
						return []engine.Message{engine.AskQuestion{Player: pl.ID,
							Question: engine.Ask(engine.Tf("c.reassignmentMoveWhichCounter"), choices...)}}
					},
				},
				{
					Label: engine.Tf("c.searchYourDeckForASHIELDSupport"),
					Type:  engine.AbilityAction, Exhaust: true, AlterEgoOnly: true,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						pl := g.Player(self)
						if pl == nil {
							return nil
						}
						var choices []engine.Choice
						for _, c := range pl.Deck {
							def := c.Def()
							if def.Type == "support" && hasShieldTrait(def) {
								choices = append(choices, engine.Choice{
									Label: engine.S("Add " + def.Name + " to your hand"),
									Kind:  engine.ChoiceCard, CardCode: c.Code,
								}.Msgs(
									engine.TakeDeckCard{Player: pl.ID, CardID: c.ID},
									engine.ShufflePlayerDeck{Player: pl.ID},
								))
							}
						}
						if len(choices) == 0 {
							return []engine.Message{engine.ShufflePlayerDeck{Player: pl.ID}}
						}
						return []engine.Message{engine.AskQuestion{Player: pl.ID,
							Question: engine.Ask(engine.Tf("c.mariaHillFindASHIELDSupport"), choices...)}}
					},
				},
			}
		},
	})
}

func registerMariaSignatures() {
	// 50002 Nick Fury: after a basic attack or thwart, place a counter.
	// Basic defense has no ally-use window in the current engine.
	engine.RegisterBehavior("50002", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			hit := false
			switch m := msg.(type) {
			case engine.AllyAttackWindow:
				hit = m.Ally == e.EID()
			case engine.AllyThwartWindow:
				hit = m.Ally == e.EID()
			}
			if !hit {
				return nil
			}
			return supportCounterChoices(g, g.Player(e.EOwner()),
				"Nick Fury — place 1 all-purpose counter", 1)
		},
	})

	// 50003 All-Points Bulletin. The printed card permits an independent
	// damage/thwart choice for every support. Multiple simultaneous nested
	// prompts are not representable, so all supports share one chosen mode.
	engine.RegisterBehavior("50003", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			n := len(shieldSupports(g, p))
			if n == 0 {
				return nil
			}
			choices := cardutil.EnemyChoices(g, n, e.EOwner(), func(id engine.EntityID) []engine.Message {
				return []engine.Message{engine.DamageEntity{Target: id, Damage: n, Source: e.EOwner()}}
			})
			choices = append(choices, cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
				return []engine.Message{engine.ThwartScheme{Scheme: id, N: n, Source: e.EOwner()}}
			})...)
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(),
				Question: engine.Ask(engine.Tf("c.allPointsBulletinResolveSHIELDSupports", n), choices...)}}
		},
	})

	// 50004 On the Double: ready a stable subset costing at most 6.
	engine.RegisterBehavior("50004", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			remaining := 6
			var msgs []engine.Message
			for _, s := range shieldSupports(g, p) {
				cost := cardutil.Cost(s.EDef())
				if s.Exhausted && cost <= remaining {
					remaining -= cost
					msgs = append(msgs, engine.ReadyEntity{ID: s.ID})
				}
			}
			return msgs
		},
	})

	// 50005 Reinforcements: place counters on a stable subset costing at
	// most 6. This deterministic subset approximates the free multi-select.
	engine.RegisterBehavior("50005", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			remaining := 6
			var msgs []engine.Message
			for _, s := range shieldSupports(g, p) {
				cost := cardutil.Cost(s.EDef())
				if cost <= remaining {
					remaining -= cost
					msgs = append(msgs, engine.AddEntityCounter{ID: s.ID, N: 1})
				}
			}
			return msgs
		},
	})

	// 50006 The Hard Call.
	engine.RegisterBehavior("50006", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			var choices []engine.Choice
			for _, s := range shieldSupports(g, p) {
				n := cardutil.Cost(s.EDef())
				msgs := []engine.Message{engine.DiscardControlled{Player: p.ID, ID: s.ID}}
				msgs = append(msgs, allEnemyDamage(g, p.ID, n)...)
				choices = append(choices, engine.Choice{
					Label: engine.Tf("c.discardDamageToEachEnemy", s, n),
					Kind:  engine.ChoiceCard, SourceID: s.ID, CardCode: s.Code,
				}.Msgs(msgs...))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.theHardCallDiscardWhichSupport"), choices...)}}
		},
	})

	// 50007 Special Funding: payment provenance is not exposed to card
	// behaviors, so its after-payment counter is not enforceable.
	engine.RegisterBehavior("50007", &engine.Behavior{})

	// 50008 Support Staff. The generic resource channel pays for its owner;
	// choosing another S.H.I.E.L.D. player is not currently exposed.
	engine.RegisterBehavior("50008", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 3}}
		},
		Resource: &engine.ResourceAbility{Icon: "wild", UsesCounters: true},
	})

	// 50009 The Iliad.
	engine.RegisterBehavior("50009", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 3}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			if s == nil || s.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.theIliadSpend1MissionCounter"), Type: engine.AbilityAction, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					if s == nil || s.Counters <= 0 {
						return nil
					}
					var choices []engine.Choice
					for _, c := range cardutil.EnemyChoices(g, 5, s.Owner, func(id engine.EntityID) []engine.Message {
						return []engine.Message{engine.AddEntityCounter{ID: self, N: -1},
							engine.DamageEntity{Target: id, Damage: 5, Source: s.Owner}}
					}) {
						choices = append(choices, c)
					}
					for _, c := range cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
						return []engine.Message{engine.AddEntityCounter{ID: self, N: -1},
							engine.ThwartScheme{Scheme: id, N: 4, Source: s.Owner}}
					}) {
						choices = append(choices, c)
					}
					for _, pl := range g.Players {
						choices = append(choices, engine.Choice{
							Label: engine.S("Heal 3 damage from " + pl.Name), Kind: engine.ChoiceTarget, SourceID: pl.ID,
						}.Msgs(engine.AddEntityCounter{ID: self, N: -1}, engine.HealEntity{Target: pl.ID, N: 3}))
					}
					return []engine.Message{engine.AskQuestion{Player: s.Owner,
						Question: engine.Ask(engine.Tf("c.theIliadChooseAMissionEffect"), choices...)}}
				},
			}}
		},
	})

	// 50010 Life Model Decoy. Damage attribution does not distinguish an
	// attack here, so it automatically prevents the next damage instance.
	engine.RegisterBehavior("50010", &engine.Behavior{
		DamagePrevention: func(g *engine.Game, u *engine.Upgrade, p *engine.Player, n int) (int, int) {
			g.Delete(u.ID)
			p.Discard = append(p.Discard, engine.Card{ID: g.NextCardID(), Code: u.Code, Owner: p.ID})
			return n, 0
		},
	})

	// 50011 S.H.I.E.L.D. Director.
	engine.RegisterBehavior("50011", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.sHIELDDirectorPlace1AllPurposeCounter"),
				Type:  engine.AbilityAction, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					if u == nil {
						return nil
					}
					return supportCounterChoices(g, g.Player(u.Owner),
						"S.H.I.E.L.D. Director — choose a support", 1)
				},
			}}
		},
	})
}

func registerMariaNemesis() {
	// 50030 Controller. Controlled Innocents uses a facedown player card as
	// a 1/1/1 minion; the proxy is marked BlankText so it cannot retrigger
	// Controller's own response.
	engine.RegisterBehavior("50030", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.BlankText {
				return nil
			}
			m, ok := msg.(engine.MinionActivates)
			if !ok || m.MinionID != mn.ID {
				return nil
			}
			p := g.Player(m.Player)
			removeCounterFromFirstSupport(g, p)
			if controlledInnocentsInPlay(g) {
				controlledMinion(g, p)
			}
			return nil
		},
	})

	// 50031 Army of the Controlled: put Controlled Innocents into play.
	// The environment is created directly because the engine has no generic
	// encounter-card-to-environment message.
	engine.RegisterBehavior("50031", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if controlledInnocentsInPlay(g) {
				return nil
			}
			env := &engine.Environment{ID: g.NextEntityID("environment"), Code: "50032"}
			g.Environments[env.ID] = env
			g.TLogf("c.controlledInnocentsEntersPlay")
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			count := 0
			for id, mn := range g.Minions {
				if mn != nil && mn.BlankText && mn.Source != nil {
					if p := g.Player(mn.EngagedWith); p != nil {
						p.Discard = append(p.Discard, *mn.Source)
					}
					g.Delete(id)
					count++
				}
			}
			if count == 0 || len(g.Players) == 0 {
				return nil
			}
			return supportCounterChoices(g, g.Players[0],
				fmt.Sprintf("Army of the Controlled — place %d counters", count), count)
		},
	})

	// 50032 Controlled Innocents: defeated Controlled cards return to their
	// owner's discard and add 1 main-scheme threat.
	engine.RegisterBehavior("50032", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionDefeated)
			if !ok {
				return nil
			}
			mn := g.Minions[m.MinionID]
			if mn == nil || !mn.BlankText || mn.Source == nil {
				return nil
			}
			if p := g.Player(mn.EngagedWith); p != nil {
				p.Discard = append(p.Discard, *mn.Source)
			}
			if g.MainScheme == nil {
				return nil
			}
			return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: e.EID()}}
		},
	})

	// 50033 Diabolical Discs. Surge is approximated by dealing one more
	// encounter card to the resolving player.
	engine.RegisterBehavior("50033", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			removeCounterFromFirstSupport(g, p)
			if controlledInnocentsInPlay(g) {
				controlledMinion(g, p)
			}
			return []engine.Message{engine.DealEncounterToPlayer{Player: p.ID}}
		},
	})
}

func registerMariaObligation() {
	// 50029 Press Conference normally remains in a player's play area. The
	// engine has no persistent obligation zone, so resolve it immediately:
	// either exhaust in alter ego to remove it, or lose one counter from
	// every support and discard it.
	engine.RegisterBehavior("50029", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var loss []engine.Message
			for _, s := range shieldSupports(g, p) {
				if s.Counters > 0 {
					loss = append(loss, engine.AddEntityCounter{ID: s.ID, N: -1})
				}
			}
			loss = append(loss, engine.ObligationResolve{Player: p.ID, Card: card})
			choices := []engine.Choice{(engine.Choice{
				ID: "pressure", Label: engine.Tf("c.remove1CounterFromEachSHIELDSupport"),
				Kind: engine.ChoiceLabel,
			}).Msgs(loss...)}
			if !p.IsHero() && !p.Exhausted {
				choices = append(choices, engine.Choice{
					ID: "remove", Label: engine.Tf("c.exhaustMariaHillAndRemovePressConference"),
					Kind: engine.ChoiceLabel,
				}.Msgs(engine.ExhaustEntity{ID: p.ID},
					engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}))
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.pressConferenceChoose"), choices...)}}
		},
	})
}
