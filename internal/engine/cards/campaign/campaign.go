// Package campaign implements the meta-campaign player cards of The Rise
// of Red Skull: the Hydra Campaign TECH upgrades (04155–04158), the
// Basic/Improved Condition upgrades (04159a–04162b) and the Expert
// Campaign obligation hazards (04163–04166). The obligations shuffle into
// a player's deck during campaign setup and enter play when drawn
// (engine-side in DrawCards); the upgrades carry the Setup keyword and
// begin the game in play (engine-side enterSetupCards).
package campaign

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerConditionUpgrades()
	registerTechUpgrades()
	registerObligations()
}

// conditionSpec describes one Condition upgrade pair (a = basic,
// b = improved). The improved side keeps the basic stats and adds an
// exhaust-to-draw response.
type conditionSpec struct {
	a, b  string
	hp    int
	stats engine.StatBonus
}

func registerConditionUpgrades() {
	specs := []conditionSpec{
		{a: "04159a", b: "04159b", hp: 2, stats: engine.StatBonus{THW: 1}},
		{a: "04160a", b: "04160b", hp: 1, stats: engine.StatBonus{ATK: 1}},
		{a: "04161a", b: "04161b", hp: 3, stats: engine.StatBonus{DEF: 1}},
		{a: "04162a", b: "04162b", hp: 4, stats: engine.StatBonus{REC: 1}},
	}
	for _, s := range specs {
		s := s
		base := &engine.Behavior{
			// Permanent: the +N hit points stay for the rest of the game.
			OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
				if p := g.Player(e.EOwner()); p != nil {
					p.MaxHP += s.hp
				}
				return nil
			},
			IdentityStats: func(p *engine.Player) engine.StatBonus { return s.stats },
		}
		engine.RegisterBehavior(s.a, base)
		engine.RegisterBehavior(s.b, &engine.Behavior{
			OnPlay:        base.OnPlay,
			IdentityStats: base.IdentityStats,
			// Improved riders: after the trigger, exhaust this card →
			// draw 1. Defeat attribution is not tracked engine-side, so
			// scheme/minion defeats count group-wide; defense and
			// recovery are owner-scoped.
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				u, ok := e.(*engine.Upgrade)
				if !ok || u.Exhausted || e.EOwner() == "" {
					return nil
				}
				mine := false
				switch m := msg.(type) {
				case engine.SchemeDefeated:
					mine = true
				case engine.MinionDefeated:
					mine = true
				case engine.WindowDefended:
					mine = m.Defender == e.EOwner()
				case engine.BasicRecover:
					mine = m.Player == e.EOwner()
				}
				if !mine {
					return nil
				}
				u.Exhausted = true
				return []engine.Message{engine.DrawCards{Player: e.EOwner(), N: 1}}
			},
		})
	}
}

// registerTechUpgrades installs the four Hydra Campaign TECH upgrades.
// Each is a passive Setup upgrade whose printed one-shot Hero Action
// discards it and removes it from the campaign log (engine-side the
// campaign layer marks a spent TECH upgrade found in the final discard).
func registerTechUpgrades() {
	discardSelf := func(self engine.EntityID, owner engine.PlayerID) engine.Message {
		return engine.DiscardControlled{Player: owner, ID: self}
	}
	engine.RegisterBehavior("04155", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.adrenalStimsReadyYourHeroAndHeal5"), Type: engine.AbilityAction, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					owner := g.Entity(self).EOwner()
					return []engine.Message{
						engine.ReadyEntity{ID: owner},
						engine.HealEntity{Target: owner, N: 5},
						discardSelf(self, owner),
					}
				},
			}}
		},
	})
	engine.RegisterBehavior("04156", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.tacticalScannerDraw5Cards"), Type: engine.AbilityAction, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					owner := g.Entity(self).EOwner()
					return []engine.Message{engine.DrawCards{Player: owner, N: 5}, discardSelf(self, owner)}
				},
			}}
		},
	})
	engine.RegisterBehavior("04157", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.emergencyTeleporterAllyEntersPlayWithATough"), Type: engine.AbilityAction, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					owner := g.Entity(self).EOwner()
					p := g.Player(owner)
					if p == nil {
						return nil
					}
					var picks []engine.Choice
					for _, zone := range []*engine.CardList{&p.Deck, &p.Discard} {
						for _, c := range *zone {
							if c.Def().Type == "ally" {
								picks = append(picks, engine.Choice{
									Label: engine.Tf("m.cardName", c), Kind: engine.ChoiceCard, CardCode: c.Code,
								}.Msgs(
									engine.AllyEntersPlayFree{Player: owner, Card: c, FromOwner: p.ID},
									engine.ToughEntity{Target: owner},
									discardSelf(self, owner),
								))
							}
						}
						if len(picks) > 0 {
							break
						}
					}
					if len(picks) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID,
						Question: engine.Ask(engine.Tf("c.emergencyTeleporterWhichAlly"), picks...)}}
				},
			}}
		},
	})
	engine.RegisterBehavior("04158", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.laserCannonDeal5ToTheVillainAndYourEnemies"), Type: engine.AbilityAction, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					owner := g.Entity(self).EOwner()
					var msgs []engine.Message
					for id := range g.Villains {
						msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 5, Source: self})
					}
					for _, id := range cardutil.SortedEnemyIDs(g) {
						if mn := g.Minions[id]; mn != nil && mn.EngagedWith == owner {
							msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 5, Source: self})
						}
					}
					return append(msgs, discardSelf(self, owner))
				},
			}}
		},
	})
}

// registerObligations installs the four Expert Campaign obligations. They
// shuffle into a player's deck and enter play when drawn (the engine's
// DrawCards branch spawns encounter-category cards as upgrades on the
// player); each offers its printed Alter-Ego action to discard itself.
// Resource icons are approximated as a generic 1-resource cost.
func registerObligations() {
	const aeCost = 1
	aeAction := func(labelKey string, extra func(g *engine.Game, self engine.EntityID, owner engine.PlayerID) []engine.Message) func(g *engine.Game, e engine.Entity) []engine.Ability {
		return func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf(labelKey), Type: engine.AbilityAction, AlterEgoOnly: true, Cost: aeCost,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					owner := g.Entity(self).EOwner()
					msgs := []engine.Message{engine.ExhaustEntity{ID: owner}}
					msgs = append(msgs, extra(g, self, owner)...)
					return append(msgs, engine.DiscardControlled{Player: owner, ID: self})
				},
			}}
		}
	}
	// 04163 Zola's Algorithm: exhaust your alter-ego + [mental] → discard.
	engine.RegisterBehavior("04163", &engine.Behavior{
		Abilities: aeAction("c.zolasAlgorithmDiscardThisCard", func(g *engine.Game, self engine.EntityID, owner engine.PlayerID) []engine.Message { return nil }),
	})
	// 04164 Medical Emergency: hero form takes 1 damage at end of turn;
	// mill 5 + [physical] → discard.
	engine.RegisterBehavior("04164", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.PlayerTurnEnd)
			if !ok || m.Player != e.EOwner() {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil || !p.IsHero() {
				return nil
			}
			return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 1, Source: e.EID()}}
		},
		Abilities: aeAction("c.medicalEmergencyDiscardThisCard", func(g *engine.Game, self engine.EntityID, owner engine.PlayerID) []engine.Message {
			return []engine.Message{engine.MillPlayerDeck{Player: owner, N: 5}}
		}),
	})
	// 04165 Martial Law: hand size -1; deal yourself an encounter card +
	// [energy] → discard.
	engine.RegisterBehavior("04165", &engine.Behavior{
		HandSizeBonus: func(g *engine.Game, p *engine.Player) int { return -1 },
		Abilities: aeAction("c.martialLawDiscardThisCard", func(g *engine.Game, self engine.EntityID, owner engine.PlayerID) []engine.Message {
			return []engine.Message{engine.RevealNextEncounter{Player: owner}}
		}),
	})
	// 04166 Anti-Hero Propaganda: hero -1 THW/-1 ATK/-1 DEF; 2 self
	// damage + [wild] → discard.
	engine.RegisterBehavior("04166", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{ATK: -1, THW: -1, DEF: -1} },
		Abilities: aeAction("c.antiHeroPropagandaDiscardThisCard", func(g *engine.Game, self engine.EntityID, owner engine.PlayerID) []engine.Message {
			return []engine.Message{engine.DamageEntity{Target: owner, Damage: 2, Source: self, Unpreventable: true}}
		}),
	})
}
