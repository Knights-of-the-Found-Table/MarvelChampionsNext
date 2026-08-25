package aos

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() { registerShieldSuite() }

func registerShieldSuite() {
	// 50012 Victoria Hand: readies a S.H.I.E.L.D. support on entry.
	engine.RegisterBehavior("50012", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, s := range shieldSupports(g, p) {
				if s.Exhausted {
					choices = append(choices, engine.Choice{
						Label: engine.S("Ready " + s.EDef().Name), Kind: engine.ChoiceTarget, SourceID: s.ID, CardCode: s.Code,
					}.Msgs(engine.ReadyEntity{ID: s.ID}))
				}
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.victoriaHandReadyWhichSHIELDSupport"), choices...)}}
		},
	})

	// 50013 Slingshot: the hand-deployment action is approximated by her
	// normal ally entry (return-at-end-of-phase not modeled).
	engine.RegisterBehavior("50013", &engine.Behavior{})

	// 50014 Organizational Support: payment-time resource generation from
	// exhausted characters is not modeled.
	engine.RegisterBehavior("50014", &engine.Behavior{})

	// 50015 Agents of S.H.I.E.L.D.: encounter-card cancellation not modeled
	// (same approximation as Eyes in the Sky).
	engine.RegisterBehavior("50015", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			g.TLogf("c.agentsOfSHIELDCoordinatesTheTeam")
			return nil
		},
	})

	// 50016 Command Team: 3 command counters, spend one to ready an ally.
	engine.RegisterBehavior("50016", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			g.Supports[e.EID()].Counters = 3
			g.TLogf("c.commandTeamEntersPlayWith3CommandCounters")
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			if s == nil || s.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.commandTeamReadyAnAlly"), Type: engine.AbilityAction, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(g.Supports[self].Owner)
					if p == nil {
						return nil
					}
					var choices []engine.Choice
					for _, aid := range p.Allies {
						if a := g.Allies[aid]; a != nil && a.Exhausted {
							choices = append(choices, engine.Choice{
								Label: engine.S("Ready " + a.EDef().Name), Kind: engine.ChoiceTarget, SourceID: aid, CardCode: a.Code,
							}.Msgs(engine.ReadyEntity{ID: aid}))
						}
					}
					if len(choices) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.commandTeamReadyWhichAlly"), choices...)}}
				},
			}}
		},
	})

	// 50017 The Circe: 2 deploy counters, spend one to deploy an ally from
	// a player's hand for free.
	engine.RegisterBehavior("50017", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			g.Supports[e.EID()].Counters = 2
			g.TLogf("c.theCirceEntersPlayWith2DeployCounters")
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			if s == nil || s.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.theCirceDeployAnAllyFromAPlayerSHand"), Type: engine.AbilityAction, Exhaust: true,
				Execute: circeDeploy,
			}}
		},
	})

	// 50019 The Douglass: 3 operation counters, spend one to remove 2
	// threat from each scheme.
	engine.RegisterBehavior("50019", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			g.Supports[e.EID()].Counters = 3
			g.TLogf("c.theDouglassEntersPlayWith3OperationCounters")
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			if s == nil || s.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.theDouglassRemove2ThreatFromEachScheme"), Type: engine.AbilityAction, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					var msgs []engine.Message
					for _, id := range g.Schemes() {
						msgs = append(msgs, engine.ThwartScheme{Scheme: id, N: 2, Source: self})
					}
					return msgs
				},
			}}
		},
	})

	// 50020 The Pericles: 2 supply counters, spend one to hand out two
	// status cards (a tough for a friendly character, a stun or confuse for
	// an enemy).
	engine.RegisterBehavior("50020", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			g.Supports[e.EID()].Counters = 2
			g.TLogf("c.thePericlesEntersPlayWith2SupplyCounters")
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			if s == nil || s.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.thePericlesGiveOutStatusCards"), Type: engine.AbilityAction, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(g.Supports[self].Owner)
					if p == nil {
						return nil
					}
					var friendly, enemy []engine.Choice
					for _, pl := range g.Players {
						if pl.KOed {
							continue
						}
						friendly = append(friendly, engine.Choice{
							Label: engine.S("Tough: " + pl.Name), Kind: engine.ChoiceTarget, SourceID: pl.ID,
						}.Msgs(engine.ToughEntity{Target: pl.ID}))
						for _, aid := range pl.Allies {
							if a := g.Allies[aid]; a != nil {
								friendly = append(friendly, engine.Choice{
									Label: engine.S("Tough: " + a.EDef().Name), Kind: engine.ChoiceTarget, SourceID: aid, CardCode: a.Code,
								}.Msgs(engine.ToughEntity{Target: aid}))
							}
						}
					}
					for _, id := range cardutil.SortedEnemyIDs(g) {
						for _, st := range []struct {
							name string
							mk   func(engine.EntityID) engine.Message
						}{{"Stun", func(t engine.EntityID) engine.Message { return engine.StunEntity{Target: t} }},
							{"Confuse", func(t engine.EntityID) engine.Message { return engine.ConfuseEntity{Target: t} }}} {
							enemy = append(enemy, engine.Choice{
								Label: engine.Tf("c.nameColon", st.name, cardutil.EnemyLabel(g.Entity(id))), Kind: engine.ChoiceTarget, SourceID: id,
							}.Msgs(st.mk(id)))
						}
					}
					if len(friendly) == 0 || len(enemy) == 0 {
						return nil
					}
					return []engine.Message{
						engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.thePericlesWhichFriendlyCharacterGetsTough"), friendly...)},
						engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.thePericlesWhichEnemyGetsStunnedOrConfused"), enemy...)},
					}
				},
			}}
		},
	})

	// 50021 Dum Dum Dugan: the exhaust-for-power-boost interrupt is not
	// modeled; he enters play as a plain heavy ally.
	engine.RegisterBehavior("50021", &engine.Behavior{})

	// 50022 Grant Ward: cannot defend; after you reveal a treachery, spend
	// a mental resource (approximated: discard any card) or Grant Ward is
	// damaged and removed from the game.
	engine.RegisterBehavior("50022", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			r, ok := msg.(engine.RevealEncounterCard)
			if !ok || r.Player != e.EOwner() || r.Card.Def().Type != "treachery" {
				return nil
			}
			p := g.Player(e.EOwner())
			a := g.Allies[e.EID()]
			if p == nil || a == nil || len(p.Hand) == 0 {
				return nil
			}
			var choices []engine.Choice
			for _, c := range p.Hand {
				choices = append(choices, engine.Choice{
					Label: engine.S("Discard " + c.Def().Name + " (cover for Grant Ward)"), Kind: engine.ChoiceCard, CardCode: c.Code,
				}.Msgs(engine.ConsumeHandCard{Player: p.ID, CardID: c.ID}))
			}
			choices = append(choices, engine.Choice{
				Label: engine.Tf("c.refuseGrantWardTakesDamageAndLeaves"), Kind: engine.ChoicePass,
			}.Msgs(engine.DamageEntity{Target: a.ID, Damage: a.AttackVal, Source: a.ID},
				engine.DiscardControlled{Player: p.ID, ID: a.ID}))
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.grantWardSpendAResourceDiscardACardAfterTheTreacheryReveal"), choices...)}}
		},
	})

	// 50023 Melinda May: after she uses a basic power, look at the top
	// card of the encounter deck and optionally discard it.
	engine.RegisterBehavior("50023", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			var hit bool
			switch m := msg.(type) {
			case engine.AllyAttackWindow:
				hit = m.Ally == e.EID()
			case engine.AllyThwartWindow:
				hit = m.Ally == e.EID()
			}
			if !hit {
				return nil
			}
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			top, ok := g.PeekEncounterTop()
			if !ok {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(
				engine.S("Melinda May — top of the encounter deck: "+top.Def().Name),
				engine.Choice{Label: engine.Tf("c.discardIt"), Kind: engine.ChoicePass}.Msgs(engine.DiscardEncounterCard{Card: top}),
				engine.Choice{Label: engine.Tf("c.leaveIt"), Kind: engine.ChoicePass},
			)}}
		},
	})

	// 50024 Super Spies: distribute 3 all-purpose counters — approximated
	// as 3 counters on one S.H.I.E.L.D. support.
	engine.RegisterBehavior("50024", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			return supportCounterChoices(g, p, "Super Spies — place 3 counters on which S.H.I.E.L.D. support?", 3)
		},
	})

	// 50025/50026/50027 basic resources (Max 1 per deck).
	engine.RegisterBehavior("50025", &engine.Behavior{})
	engine.RegisterBehavior("50026", &engine.Behavior{})
	engine.RegisterBehavior("50027", &engine.Behavior{})

	// 50028 Front Organization: the discard-replacement interrupt window
	// is not modeled.
	engine.RegisterBehavior("50028", &engine.Behavior{})

	registerFurySuite()
}

// circeDeploy lets each player with an ally in hand deploy one for free.
func circeDeploy(g *engine.Game, self engine.EntityID) []engine.Message {
	s := g.Supports[self]
	if s == nil {
		return nil
	}
	var choices []engine.Choice
	for _, p := range g.Players {
		if p.KOed {
			continue
		}
		for _, c := range p.Hand {
			if c.Def().Type != "ally" {
				continue
			}
			choices = append(choices, engine.Choice{
				Label: engine.S(p.Name + " deploys " + c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code,
			}.Msgs(engine.AllyEntersPlayFree{Player: p.ID, Card: c}))
		}
	}
	if len(choices) == 0 {
		return nil
	}
	return []engine.Message{engine.AskQuestion{Player: s.Owner, Question: engine.Ask(engine.Tf("c.theCirceWhichAllyEntersPlayForFree"), choices...)}}
}

func registerFurySuite() {
	// 50047 Agent Coulson: fetch a Preparation card from deck/discard.
	engine.RegisterBehavior("50047", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			for _, c := range p.Discard {
				if data.BaseCode(c.Code) != "" && c.Def().HasTrait("preparation") {
					return []engine.Message{
						engine.ReturnDiscardCard{Player: p.ID, CardID: c.ID},
					}
				}
			}
			for i, c := range p.Deck {
				if c.Def().HasTrait("preparation") {
					card := c
					p.Deck = append(p.Deck[:i:i], p.Deck[i+1:]...)
					p.Hand = append(p.Hand, card)
					g.TLogf("c.agentCoulsonFinds", card)
					return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
				}
			}
			return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
		},
	})

	// 50048 Quake: after a minion schemes, exhaust to deal it 2 damage.
	engine.RegisterBehavior("50048", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeThreat)
			if !ok {
				return nil
			}
			a := g.Allies[e.EID()]
			if a == nil || a.Exhausted {
				return nil
			}
			mn := g.Minions[m.Source]
			if mn == nil {
				return nil
			}
			a.Exhausted = true
			g.TLogf("c.quakeShocksAfterItsScheme", mn)
			return []engine.Message{engine.DamageEntity{Target: mn.ID, Damage: 2, Source: e.EID()}}
		},
	})

	// 50049 Global Logistics: approximate the top-4 arrangement as
	// discarding any one of the encounter deck's top 4, rest to the bottom.
	engine.RegisterBehavior("50049", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || len(g.EncounterDeck) == 0 {
				return nil
			}
			n := 4
			if len(g.EncounterDeck) < n {
				n = len(g.EncounterDeck)
			}
			top := append(engine.CardList(nil), g.EncounterDeck[:n]...)
			var choices []engine.Choice
			for _, c := range top {
				choices = append(choices, engine.Choice{
					Label: engine.S("Discard " + c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code,
				}.Msgs(engine.DiscardEncounterCard{Card: c}))
			}
			choices = append(choices, engine.Choice{Label: engine.Tf("c.discardNone"), Kind: engine.ChoicePass})
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.globalLogisticsDiscardOneOfTheEncounterDeckSTopCards"), choices...)}}
		},
	})

	// 50050 Informant: handled by the engine's minion-scheme window.
	engine.RegisterBehavior("50050", &engine.Behavior{})

	// 50051 Intelligence: the multi-card swap is approximated as peeking at
	// the encounter deck's top card with an optional discard.
	engine.RegisterBehavior("50051", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.RevealEncounterCard); !ok {
				return nil
			}
			p := g.Player(e.EOwner())
			u := g.Upgrades[e.EID()]
			if p == nil || u == nil {
				return nil
			}
			top, ok2 := g.PeekEncounterTop()
			if !ok2 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(
				engine.S("Intelligence — top of the encounter deck: "+top.Def().Name),
				engine.Choice{Label: engine.Tf("c.discardIntelligenceAndTheTopCard"), Kind: engine.ChoicePass}.Msgs(
					engine.DiscardControlled{Player: p.ID, ID: u.ID},
					engine.DiscardEncounterCard{Card: top}),
				engine.Choice{Label: engine.Tf("c.keepLooking"), Kind: engine.ChoicePass},
			)}}
		},
	})

	// 50052 Prism Dust: after a minion enters play, discard to confuse it
	// and deal it 2 damage.
	engine.RegisterBehavior("50052", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionEntersPlay)
			if !ok {
				return nil
			}
			p := g.Player(e.EOwner())
			u := g.Upgrades[e.EID()]
			if p == nil || u == nil || g.Minions[m.MinionID] == nil {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(
				engine.S("Prism Dust — discard to confuse and damage "+g.Minions[m.MinionID].EDef().Name+"?"),
				engine.Choice{Label: engine.Tf("c.discardPrismDust"), Kind: engine.ChoicePass}.Msgs(
					engine.DiscardControlled{Player: p.ID, ID: u.ID},
					engine.ConfuseEntity{Target: m.MinionID},
					engine.DamageEntity{Target: m.MinionID, Damage: 2, Source: u.ID}),
				engine.Choice{Label: engine.Tf("m.pass"), Kind: engine.ChoicePass},
			)}}
		},
	})

	// 50053 Under Surveillance: raise the main scheme's target threat by 4.
	engine.RegisterBehavior("50053", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			u := g.Upgrades[e.EID()]
			if g.MainScheme != nil && u != nil {
				u.AttachTo = g.MainScheme.ID
				g.MainScheme.MaxThreat += 4
				g.TLogf("c.underSurveillanceRaisesTheMainSchemeSTargetThreatBy4")
			}
			return nil
		},
	})

	// 50054 Nick Fury, Sr.: on entry choose a boon; leaves at end of round.
	engine.RegisterBehavior("50054", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var tough []engine.Choice
			for _, pl := range g.Players {
				if pl.KOed {
					continue
				}
				if hasShieldTrait(pl.EDef()) {
					tough = append(tough, engine.Choice{Label: engine.S("Tough: " + pl.Name), Kind: engine.ChoiceTarget, SourceID: pl.ID}.Msgs(engine.ToughEntity{Target: pl.ID}))
				}
				for _, aid := range pl.Allies {
					if a := g.Allies[aid]; a != nil && hasShieldTrait(a.EDef()) {
						tough = append(tough, engine.Choice{Label: engine.S("Tough: " + a.EDef().Name), Kind: engine.ChoiceTarget, SourceID: aid, CardCode: a.Code}.Msgs(engine.ToughEntity{Target: aid}))
					}
				}
			}
			var scheme []engine.Choice
			for _, sid := range g.Schemes() {
				scheme = append(scheme, engine.Choice{Label: engine.S("Remove 3 threat from " + g.Entity(sid).EDef().Name), Kind: engine.ChoiceTarget, SourceID: sid}.Msgs(engine.ThwartScheme{Scheme: sid, N: 3, Source: e.EID()}))
			}
			var choices []engine.Choice
			choices = append(choices, engine.Choice{Label: engine.Tf("c.draw2Cards"), Kind: engine.ChoicePass}.Msgs(engine.DrawCards{Player: p.ID, N: 2}))
			if len(scheme) > 0 {
				choices = append(choices, scheme...)
			}
			choices = append(choices, tough...)
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.nickFurySrChooseOne"), choices...)}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.EndRound); !ok {
				return nil
			}
			p := g.Player(e.EOwner())
			if p == nil || g.Allies[e.EID()] == nil {
				return nil
			}
			g.TLogf("c.nickFurySrDepartsAtTheEndOfTheRound")
			return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: e.EID()}}
		},
	})

	// 50055 Jemma Simmons: cost -2 with a S.H.I.E.L.D. identity; exhausts
	// for a mental resource (the Tech-only rider is not modeled).
	engine.RegisterBehavior("50055", &engine.Behavior{
		CardCost: func(g *engine.Game, p *engine.Player, def *data.CardDef) int {
			if def != nil && def.Code == "50055" && hasShieldTrait(p.EDef()) {
				return -2
			}
			return 0
		},
		Resource: &engine.ResourceAbility{Icon: "mental"},
	})

	// 50056 Leo Fitz: cost -2 with a S.H.I.E.L.D. identity; alter-ego
	// search for a Tech card.
	engine.RegisterBehavior("50056", &engine.Behavior{
		CardCost: func(g *engine.Game, p *engine.Player, def *data.CardDef) int {
			if def != nil && def.Code == "50056" && hasShieldTrait(p.EDef()) {
				return -2
			}
			return 0
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.leoFitzSearchYourDeckForATechCard"), Type: engine.AbilityAction,
				Exhaust: true, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(g.Supports[self].Owner)
					if p == nil {
						return nil
					}
					for i, c := range p.Deck {
						if c.Def().HasTrait("tech") {
							card := c
							p.Deck = append(p.Deck[:i:i], p.Deck[i+1:]...)
							p.Hand = append(p.Hand, card)
							g.TLogf("c.leoFitzFinds", card)
							return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
						}
					}
					return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
				},
			}}
		},
	})

	// 50057 Sky-Destroyer: after you play a S.H.I.E.L.D. card, exhaust to
	// deal 2 damage to an enemy (auto-targets the first enemy).
	engine.RegisterBehavior("50057", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.PlayCard)
			if !ok || m.Player != e.EOwner() {
				return nil
			}
			s := g.Supports[e.EID()]
			if s == nil || s.Exhausted || !hasShieldTrait(m.Card.Def()) {
				return nil
			}
			ids := cardutil.SortedEnemyIDs(g)
			if len(ids) == 0 {
				return nil
			}
			s.Exhausted = true
			g.TLogf("c.skyDestroyerOpensFireOn", cardutil.EnemyLabel(g.Entity(ids[0])))
			return []engine.Message{engine.DamageEntity{Target: ids[0], Damage: 2, Source: e.EID()}}
		},
	})

	// 50058 Practiced Plan: the Preparation-discard return window is not
	// modeled.
	engine.RegisterBehavior("50058", &engine.Behavior{})
}
