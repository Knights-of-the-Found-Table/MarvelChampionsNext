package thor

// complete.go implements the remaining Thor hero-pack cards.
// Approximations are noted inline.

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func registerRemainingThor() {
	// Hercules: costs 1 less per minion engaged with you.
	engine.RegisterBehavior("06011", &engine.Behavior{
		CardCost: func(g *engine.Game, p *engine.Player, def *data.CardDef) int {
			if def.Code != "06011" {
				return 0
			}
			n := 0
			for _, mn := range g.Minions {
				if mn.EngagedWith == p.ID {
					n++
				}
			}
			return min(n, 4)
		},
	})

	// Valkyrie: 2 damage to a minion on entry (3 with energy payment).
	engine.RegisterBehavior("06012", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			// The 3-damage energy-payment rider is approximated at 2
			// (ally payments are not recorded by the engine).
			return cardutil.ChooseMinion(engine.Tf("c.valkyrieDeal2DamageToWhichMinion"), 2)(g, e)
		},
	})

	// Chase Them Down (thor reprint): action approximation like the core
	// version.
	if b := engine.LookupBehavior("01052"); b != nil {
		engine.RegisterBehavior("06013", b)
	}

	// Get Over Here!: 1 damage to a minion; engage it if Aerial.
	engine.RegisterBehavior("06014", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			var picks []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				mn := g.Minions[id]
				if mn == nil {
					continue
				}
				msgs := []engine.Message{engine.DamageEntity{Target: id, Damage: 1, Source: p.ID}}
				if p != nil && g.EntityHasTrait(p.ID, "aerial") {
					msgs = append(msgs, engine.EngageMinion{MinionID: id, Player: p.ID})
				}
				picks = append(picks, engine.Choice{
					Label: cardutil.EnemyLabel(mn), Kind: engine.ChoiceTarget, SourceID: id, CardCode: mn.Code,
				}.Msgs(msgs...))
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(),
				Question: engine.Ask(engine.Tf("c.getOverHereWhichMinion"), picks...)}}
		},
	})

	// Mean Swing: +3 ATK until end of phase (approximation of the
	// per-attack interrupt; the Weapon-exhaust cost is skipped).
	engine.RegisterBehavior("06015", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.ApplyStatBonus{Target: e.EOwner(), ATK: 3}}
		},
	})

	// The Power of Aggression reprint.
	engine.RegisterBehavior("06016", &engine.Behavior{})

	// Hall of Heroes: glory counters on minion defeats; 3 counters → 3 cards.
	engine.RegisterBehavior("06017", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.MinionDefeated); !ok {
				return nil
			}
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 1}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			if s == nil || s.Counters < 3 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustHallOfHeroes3GloryCountersDraw3Cards"), Type: engine.AbilityAction,
				Exhaust: true, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{
						engine.AddEntityCounter{ID: self, N: -3},
						engine.DrawCards{Player: g.Entity(self).EOwner(), N: 3},
					}
				},
			}}
		},
	})

	// Battle Fury: after your hero attacks and defeats a minion, discard
	// this + 1 damage to ready your hero.
	// Approximation: triggers on any basic attack against a minion.
	engine.RegisterBehavior("06018", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			ba, ok := msg.(engine.BasicAttack)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil || ba.Player != u.Owner {
				return nil
			}
			if g.Minions[ba.Target] == nil {
				return nil
			}
			return []engine.Message{
				engine.DiscardControlled{Player: u.Owner, ID: u.ID},
				engine.DamageEntity{Target: u.Owner, Damage: 1, Source: u.ID},
				engine.ReadyEntity{ID: u.Owner},
			}
		},
	})

	// Jarnbjorn: after your hero attacks, spend [physical] → 2 damage to
	// an enemy.
	// Approximation: an exhaust-free action usable once per turn after
	// attacking (OncePerTurn; the response window is not modeled).
	engine.RegisterBehavior("06019", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.spendPhysicalDeal2DamageToAnEnemy"), Type: engine.AbilityAction,
				Cost: 1, CostIcons: "physical:1",
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return cardutil.ChooseEnemy(engine.Tf("c.jarnbjornDeal2DamageToWhichEnemy"),
						func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 2, nil })(g, g.Entity(self))
				},
			}}
		},
	})

	// Heimdall: look at the top 3 encounter cards, discard 1, reorder.
	engine.RegisterBehavior("06020", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if len(g.EncounterDeck) < 3 {
				return nil
			}
			top := append(engine.CardList{}, g.EncounterDeck[:3]...)
			var picks []engine.Choice
			for _, c := range top {
				picks = append(picks, engine.Choice{
					Label: engine.S("Discard " + c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code,
				}.Msgs(engine.DiscardEncounterCard{Card: c}))
			}
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(),
				Question: engine.Ask(engine.Tf("c.heimdallDiscardWhichOfTheTop3EncounterCards"), picks...)}}
		},
	})

	// Invulnerability: tough status card.
	engine.RegisterBehavior("06021", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.ToughEntity{Target: e.EOwner()}}
		},
	})

	// Basic resources + Avengers Mansion reprint.
	engine.RegisterBehavior("06022", &engine.Behavior{})
	engine.RegisterBehavior("06023", &engine.Behavior{})
	engine.RegisterBehavior("06024", &engine.Behavior{})
	if b := engine.LookupBehavior("01091"); b != nil {
		engine.RegisterBehavior("06025", b)
	}

	// Under Surveillance: attach to the main scheme, +4 target threat.
	engine.RegisterBehavior("06031", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if g.MainScheme == nil {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			u.AttachTo = g.MainScheme.ID
			g.MainScheme.MaxThreat += 4
			g.TLogf("c.targetThreatIncreasedBy4UnderSurveillance", g.MainScheme)
			return nil
		},
	})

	// Teamwork: add an ally's power to your hero's basic power this phase.
	// Approximation: grants +2 (a typical ally power) until end of phase;
	// the exact per-use window is not modeled.
	engine.RegisterBehavior("06032", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a == nil || a.Exhausted {
					continue
				}
				atk, thw := a.AttackVal+a.BonusATK+a.PermATK, a.ThwartVal+a.BonusTHW+a.PermTHW
				picks = append(picks,
					engine.Choice{ID: "tw-atk", Label: engine.Tf("c.atk2", atk, a), Kind: engine.ChoiceBasicPower, SourceID: a.ID}.
						Msgs(engine.ExhaustEntity{ID: a.ID}, engine.ApplyStatBonus{Target: p.ID, ATK: atk}),
					engine.Choice{ID: "tw-thw", Label: engine.Tf("c.thw", thw, a), Kind: engine.ChoiceBasicPower, SourceID: a.ID}.
						Msgs(engine.ExhaustEntity{ID: a.ID}, engine.ApplyStatBonus{Target: p.ID, THW: thw}))
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.teamworkExhaustAnAllyToAddItsPowerToYourHeroThisPhase"), picks...)}}
		},
	})

	// Second Wind: heal 4 (the mental-payment 5 rider is approximated at
	// 4 — event payment icons are recorded but kept simple here).
	engine.RegisterBehavior("06033", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			for _, q := range g.Players {
				if q.Damage > 0 {
					picks = append(picks, engine.Choice{Label: engine.S(q.Name), Kind: engine.ChoiceTarget, SourceID: q.ID}.
						Msgs(engine.HealEntity{Target: q.ID, N: 4}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.secondWindHealWhichIdentity"), picks...)}}
		},
	})

	// Enhanced Physique: 3 physical counters, hero resource.
	engine.RegisterBehavior("06034", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 3}}
		},
		Resource: &engine.ResourceAbility{Icon: "physical", HeroOnly: true, UsesCounters: true},
	})
}
