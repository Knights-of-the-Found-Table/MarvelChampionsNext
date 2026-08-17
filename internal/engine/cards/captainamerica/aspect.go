package captainamerica

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// registerAspectCards installs the pack's Leadership, basic and other
// aspect cards (03011-03025, 03031-03034).
func registerAspectCards() {
	registerFalcon()
	registerHawkeye()
	registerSquirrelGirl()
	registerWonderMan()
	registerAvengersAssemble()
	registerMakeTheCall()
	registerStrengthInNumbers()
	registerPowerOfLeadership()
	registerQuinjet()
	registerMockingbird()
	registerAvengersTower()
	registerHonoraryAvenger()
	registerBasicResources()
	registerEnraged()
	registerFollowed()
	registerExpertDefense()
	registerEnhancedAwareness()
}

// 03011 Falcon: after he enters play, look at the top 3 cards of the
// encounter deck; remove 1 threat from a scheme for each treachery looked
// at (approximation: the looked cards go to the bottom of the deck).
func registerFalcon() {
	engine.RegisterBehavior("03011", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			n := min(3, len(g.EncounterDeck))
			treacheries := 0
			var names []string
			for i := 0; i < n; i++ {
				def := g.EncounterDeck[i].Def()
				names = append(names, def.Name)
				if def.Type == "treachery" {
					treacheries++
				}
			}
			if n > 0 {
				// rotate the looked cards to the bottom
				rest := make(engine.CardList, 0, len(g.EncounterDeck))
				rest = append(rest, g.EncounterDeck[n:]...)
				rest = append(rest, g.EncounterDeck[:n]...)
				g.EncounterDeck = rest
				g.Logf("Falcon looks at: %v", names)
			}
			if treacheries == 0 || len(g.Schemes()) == 0 {
				return nil
			}
			var choices []engine.Choice
			for _, id := range g.Schemes() {
				s := g.Entity(id)
				choices = append(choices, engine.Choice{
					Label: s.EDef().Name, Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: s.ECode(),
				}.Msgs(engine.ThwartScheme{Scheme: id, N: treacheries, Source: pid}))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(fmt.Sprintf("Falcon — remove %d threat from a scheme", treacheries), choices...),
			}}
		},
	})
}

// 03012 Hawkeye: enters play with 4 arrow counters; after a minion enters
// play, remove 1 arrow counter → deal 2 damage to it.
func registerHawkeye() {
	engine.RegisterBehavior("03012", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if a, ok := e.(*engine.Ally); ok {
				a.Counters = 4
				g.Logf("Hawkeye enters play with 4 arrow counters")
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionEntersPlay)
			if !ok {
				return nil
			}
			a, ok := e.(*engine.Ally)
			if !ok || a.Counters <= 0 {
				return nil
			}
			mn := g.Entity(m.MinionID)
			if mn == nil {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player: a.Owner,
				Question: engine.Ask(fmt.Sprintf("Hawkeye — remove 1 arrow counter to shoot %s for 2? (%d arrows)", mn.EDef().Name, a.Counters),
					engine.Choice{
						ID: "shoot", Label: "Shoot", Kind: engine.ChoiceLabel,
					}.Msgs(
						engine.AddEntityCounter{ID: a.ID, N: -1},
						engine.DamageEntity{Target: m.MinionID, Damage: 2, Source: a.Owner},
					),
					engine.Choice{ID: "skip", Label: "Skip", Kind: engine.ChoicePass},
				),
			}}
		},
	})
}

// 03013 Squirrel Girl: after she enters play, deal 1 damage to each enemy.
func registerSquirrelGirl() {
	engine.RegisterBehavior("03013", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var msgs []engine.Message
			for _, id := range cardutil.SortedEnemyIDs(g) {
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: e.EOwner()})
			}
			return msgs
		},
	})
}

// 03014 Wonder Man: as an additional cost for him to attack, discard 1
// card from your hand.
func registerWonderMan() {
	engine.RegisterBehavior("03014", &engine.Behavior{
		AllyAttackDiscardCost: true,
	})
}

// 03015 Avengers Assemble!: ready each Avenger character you control;
// until the end of the phase each Avenger character in play gets +1 THW
// and +1 ATK (max 1 per round — the second copy's effect is skipped).
func registerAvengersAssemble() {
	engine.RegisterBehavior("03015", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			if g.UsedThisRound["03015"] {
				g.Logf("Avengers Assemble! was already played this round (max 1)")
				return nil
			}
			g.UsedThisRound["03015"] = true
			var msgs []engine.Message
			for _, pl := range g.Players {
				for _, id := range pl.Allies {
					if a := g.Allies[id]; a != nil && g.EntityHasTrait(id, "avenger") {
						a.BonusTHW++
						a.BonusATK++
						if pl.ID == pid {
							msgs = append(msgs, engine.ReadyEntity{ID: id})
						}
					}
				}
				if g.EntityHasTrait(pl.ID, "avenger") {
					pl.BonusTHW++
					pl.BonusATK++
				}
			}
			g.Logf("Avengers Assemble! — Avengers get +1 THW and +1 ATK until the end of the phase")
			return msgs
		},
	})
}

// 03016 Make the Call: pay the printed cost of an ally in any player's
// discard pile → put that ally into play under your control.
func registerMakeTheCall() {
	engine.RegisterBehavior("03016", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, pl := range g.Players {
				for _, c := range pl.Discard {
					def := c.Def()
					if def.Type != "ally" {
						continue
					}
					cost := cardutil.Cost(def)
					choices = append(choices, engine.Choice{
						Label:    fmt.Sprintf("%s (cost %d) — from %s's discard pile", def.Name, cost, pl.Name),
						Kind:     engine.ChoiceCard, CardCode: def.Code,
					}.WithThen(g.CustomPaymentQuestion(p, cost,
						fmt.Sprintf("Pay %d resources for %s", cost, def.Name),
						map[string]any{
							"makeCallFrom": pl.ID.String(),
							"makeCallCard": c.ID,
						})))
				}
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Make the Call — choose an ally from any discard pile", choices...),
			}}
		},
	})
}

// 03017 Strength In Numbers: exhaust any number of allies you control →
// draw 1 card for each ally exhausted this way.
func registerStrengthInNumbers() {
	engine.RegisterBehavior("03017", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && !a.Exhausted {
					choices = append(choices, engine.Choice{
						Label: "Exhaust " + a.EDef().Name, Kind: engine.ChoiceTarget,
						SourceID: id, CardCode: a.Code,
					}.Msgs(
						engine.ExhaustEntity{ID: id},
						engine.DrawCards{Player: pid, N: 1},
					))
				}
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.AskN("Strength In Numbers — exhaust allies (draw 1 per ally)", len(choices), choices...),
			}}
		},
	})
}

// 03018 The Power of Leadership: doubles its resource while paying for a
// Leadership card — implemented generically in the payment validator.
func registerPowerOfLeadership() {
	engine.RegisterBehavior("03018", &engine.Behavior{})
}

// 03019 Quinjet: after your turn begins place 1 time counter on Quinjet;
// action: put an Avenger ally from your hand into play with printed cost
// ≤ the number of time counters, then discard Quinjet.
func registerQuinjet() {
	engine.RegisterBehavior("03019", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.PlayerTurnStart)
			if !ok || m.Player != e.EOwner() {
				return nil
			}
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 1}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s, ok := e.(*engine.Support)
			if !ok {
				return nil
			}
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, c := range p.Hand {
				def := c.Def()
				if def.Type != "ally" || !def.HasTrait("avenger") || cardutil.Cost(def) > s.Counters {
					continue
				}
				choices = append(choices, engine.Choice{
					Label: def.Name, Kind: engine.ChoiceCard, CardCode: def.Code,
				}.Msgs(
					engine.AllyEntersPlayFree{Player: p.ID, Card: c},
					engine.DiscardControlled{Player: p.ID, ID: e.EID()},
				))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Ability{{
				Label: fmt.Sprintf("Quinjet — put an Avenger ally (cost ≤ %d) into play, then discard Quinjet", s.Counters),
				Type:  engine.AbilityAction,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.AskQuestion{
						Player:   p.ID,
						Question: engine.Ask("Quinjet — choose an Avenger ally", choices...),
					}}
				},
			}}
		},
	})
}

// 03020 Mockingbird: after she enters play, stun an enemy.
func registerMockingbird() {
	engine.RegisterBehavior("03020", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			choices := cardutil.EnemyChoices(g, 0, pid, func(target engine.EntityID) []engine.Message {
				return []engine.Message{engine.StunEntity{Target: target}}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Mockingbird — stun an enemy", choices...),
			}}
		},
	})
}

// 03024 Avengers Tower: if each of your allies has the Avenger trait,
// increase your ally limit by 1 (ally limit not enforced — approximation);
// action: exhaust → the next Avenger ally played this phase costs 1 less.
func registerAvengersTower() {
	engine.RegisterBehavior("03024", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label:   "Avengers Tower — the next Avenger ally this phase costs 1 less",
				Type:    engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					if p := g.Player(e.EOwner()); p != nil {
						p.CostDiscounts = append(p.CostDiscounts, engine.CostDiscount{
							Type: "ally", Trait: "avenger", Amount: 1,
						})
						g.Logf("The next Avenger ally this phase costs 1 less")
					}
					return nil
				},
			}}
		},
	})
}

// 03025 Honorary Avenger: attach to a friendly character; it gets +1 hit
// point and gains the Avenger trait.
func registerHonoraryAvenger() {
	engine.RegisterBehavior("03025", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			choices = append(choices, engine.Choice{
				Label: p.Name + " (identity)", Kind: engine.ChoiceTarget, SourceID: p.ID,
			}.Msgs(engine.AttachUpgrade{ID: e.EID(), Target: p.ID, MaxHP: 1, GrantTrait: "avenger"}))
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil {
					choices = append(choices, engine.Choice{
						Label: a.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id, CardCode: a.Code,
					}.Msgs(engine.AttachUpgrade{ID: e.EID(), Target: id, MaxHP: 1, GrantTrait: "avenger"}))
				}
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Honorary Avenger — attach to a friendly character", choices...),
			}}
		},
	})
}

// 03021-03023 Energy/Genius/Strength: plain resource cards, handled
// generically by the data layer.
func registerBasicResources() {
	engine.RegisterBehavior("03021", &engine.Behavior{})
	engine.RegisterBehavior("03022", &engine.Behavior{})
	engine.RegisterBehavior("03023", &engine.Behavior{})
}

// 03031 Enraged: attach to an ally; it gets +2 ATK and takes +1
// consequential damage after it attacks.
func registerEnraged() {
	engine.RegisterBehavior("03031", &engine.Behavior{
		ConsequentialBonus: 1,
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil || len(p.Allies) == 0 {
				return nil
			}
			var choices []engine.Choice
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil {
					choices = append(choices, engine.Choice{
						Label: a.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id, CardCode: a.Code,
					}.Msgs(engine.AttachUpgrade{ID: e.EID(), Target: id, ATK: 2}))
				}
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Enraged — attach to an ally (+2 ATK)", choices...),
			}}
		},
	})
}

// 03032 Followed: attach to a side scheme; when the scheme is defeated,
// deal 4 damage to an enemy.
func registerFollowed() {
	engine.RegisterBehavior("03032", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			var choices []engine.Choice
			for _, id := range cardutil.SortedIDs(g.SideSchemes) {
				s := g.SideSchemes[id]
				choices = append(choices, engine.Choice{
					Label: s.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id, CardCode: s.Code,
				}.Msgs(engine.AttachUpgrade{ID: e.EID(), Target: id}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Followed — attach to a side scheme", choices...),
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok {
				return nil
			}
			u, ok := e.(*engine.Upgrade)
			if !ok || u.AttachTo != m.Scheme {
				return nil
			}
			pid := u.Owner
			var out []engine.Message
			out = append(out, engine.DiscardControlled{Player: pid, ID: u.ID})
			choices := cardutil.EnemyChoices(g, 4, pid, func(target engine.EntityID) []engine.Message {
				return []engine.Message{engine.DamageEntity{Target: target, Damage: 4, Source: pid}}
			})
			if len(choices) > 0 {
				out = append(out, engine.AskQuestion{
					Player:   pid,
					Question: engine.Ask("Followed — deal 4 damage to an enemy", choices...),
				})
			}
			return out
		},
	})
}

// 03033 Expert Defense: when your hero defends against an attack, it gets
// +3 DEF for that attack.
func registerExpertDefense() {
	engine.RegisterBehavior("03033", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			if !p.IsHero() || p.Exhausted {
				return engine.Defends{}, nil, false
			}
			return engine.Defends{Defender: p.ID, Against: against, DefBonus: 3}, nil, true
		},
	})
}

// 03034 Enhanced Awareness: uses (3 mental counters); hero resource —
// exhaust and remove 1 counter → generate a [mental] resource.
func registerEnhancedAwareness() {
	engine.RegisterBehavior("03034", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "mental", HeroOnly: true, UsesCounters: true},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if u, ok := e.(*engine.Upgrade); ok {
				u.Counters = 3
				g.Logf("Enhanced Awareness enters play with 3 mental counters")
			}
			return nil
		},
	})
}
