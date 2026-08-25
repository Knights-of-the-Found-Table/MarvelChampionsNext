package riseofredskull

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// registerRedSkull installs the Red Skull scenario (04125–04144): the
// side-scheme deck drip (set-aside pool approximation).
func registerRedSkull() {
	for _, base := range []string{"04125", "04126", "04127"} {
		b := &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				if _, ok := msg.(engine.BeginRound); !ok {
					return nil
				}
				v := g.Villains[e.EID()]
				if v == nil {
					return nil
				}
				v.AttackVal = derefInt(v.EDef().Attack, 1) + len(g.SideSchemes)
				return nil
			},
		}
		if base != "04125" {
			b.VillainStage = func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
				var msgs []engine.Message
				for _, p := range g.Players {
					msgs = append(msgs, engine.DealEncounterToPlayer{Player: p.ID})
				}
				return msgs
			}
		}
		engine.RegisterBehavior(base, b)
	}

	// 04128/04129: one side scheme per round.
	drip := func() *engine.Behavior {
		return &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				bp, ok := msg.(engine.BeginPhase)
				if !ok || bp.Phase != engine.PhaseVillain {
					return nil
				}
				return revealSideSchemeFromPool(g)
			},
		}
	}
	engine.RegisterBehavior("04128", drip())
	engine.RegisterBehavior("04129", drip())

	// 04130 The Sleeper.
	engine.RegisterBehavior("04130", &engine.Behavior{})

	// 04131 Hydra Exo-Soldier.
	engine.RegisterBehavior("04131", &engine.Behavior{
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			for id := range g.Villains {
				return []engine.Message{
					engine.ToughEntity{Target: id},
					engine.DealBoost{Enemy: id},
				}
			}
			return nil
		},
	})

	// 04132/04133 Red Skull's gear.
	for _, code := range []string{"04132", "04133"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
				for id := range g.Villains {
					if v := g.Villains[id]; v != nil && v.Code[:5] == "04125" {
						t.Target = id
						return nil
					}
				}
				for id := range g.Villains {
					t.Target = id
					break
				}
				return nil
			},
			Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
				return []engine.Ability{{
					Label: engine.Tf("c.spendEnergyMentalPhysicalDiscardThisCard"), Type: engine.AbilityAction,
					CostIcons: "energy:1 mental:1 physical:1",
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						return []engine.Message{engine.DiscardAttachmentMsg{ID: self}}
					},
				}}
			},
			Boost: func(g *engine.Game, card engine.Card) []engine.Message {
				for id := range g.Villains {
					if v := g.Villains[id]; v != nil && v.Code[:5] == "04125" {
						g.SpawnAttachment(card.Code, id)
						break
					}
				}
				return nil
			},
		})
	}

	// 04134 Master Strategist.
	engine.RegisterBehavior("04134", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil && v.Code[:5] == "04125" {
					t.Target = id
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

	// 04135 Twisted Reality.
	engine.RegisterBehavior("04135", &engine.Behavior{})

	// 04136–04138 treacheries.
	engine.RegisterBehavior("04136", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var msgs []engine.Message
			n := len(g.SideSchemes)
			if n == 0 {
				n = 1
			}
			if !p.Exhausted {
				msgs = append(msgs, engine.ExhaustEntity{ID: p.ID})
				n--
			}
			for i := 0; i < n; i++ {
				for _, id := range p.Allies {
					if a := g.Allies[id]; a != nil && !a.Exhausted {
						msgs = append(msgs, engine.ExhaustEntity{ID: id})
						break
					}
				}
			}
			return msgs
		},
	})
	engine.RegisterBehavior("04137", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var msgs []engine.Message
			if g.MainScheme != nil {
				msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: t.ID})
			}
			for _, id := range sortedSchemeIDs(g) {
				msgs = append(msgs, engine.SchemeThreat{Scheme: id, N: 2, Source: t.ID})
			}
			return msgs
		},
	})
	engine.RegisterBehavior("04138", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil {
					v.Tough = true
				}
				return []engine.Message{engine.VillainActivates{VillainID: id, Player: p.ID}}
			}
			return nil
		},
	})

	// 04139–04144 side-scheme pool.
	engine.RegisterBehavior("04139", &engine.Behavior{})
	engine.RegisterBehavior("04140", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			m := &engine.Minion{ID: g.NextEntityID("minion"), Code: "04130", MaxHP: 8, AttackVal: 3, SchemeVal: 2}
			g.AddMinion(m, cardutil.FirstPlayerID(g))
			g.TLogf("c.theSleeperAwakens")
			return nil
		},
	})
	engine.RegisterBehavior("04141", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			p := g.Player(g.ActiveTurn)
			if p == nil {
				return nil
			}
			for i, c := range p.Deck {
				if c.Def().Type == "ally" {
					p.Deck = append(p.Deck[:i], p.Deck[i+1:]...)
					return []engine.Message{engine.PlayDiscardAlly{Player: p.ID, Card: c}}
				}
			}
			return nil
		},
	})
	engine.RegisterBehavior("04142", &engine.Behavior{})
	engine.RegisterBehavior("04143", &engine.Behavior{})
	engine.RegisterBehavior("04144", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			s := g.SideSchemes[e.EID()]
			if s == nil {
				return nil
			}
			for _, p := range g.Players {
				seen := map[string]bool{}
				for i := 0; i < 5 && len(p.Deck) > 0; i++ {
					c := p.Deck[0]
					p.Deck = p.Deck[1:]
					for _, r := range c.Def().Resources {
						seen[r] = true
					}
					p.Discard = append(p.Discard, c)
				}
				s.Threat += len(seen)
			}
			return nil
		},
	})
}

// revealSideSchemeFromPool pops a scheme from the side-scheme deck.
func revealSideSchemeFromPool(g *engine.Game) []engine.Message {
	for i, c := range g.SetAside {
		if c.Def().Type == "side_scheme" {
			g.SetAside = append(g.SetAside[:i], g.SetAside[i+1:]...)
			g.TLogMajorf("c.isRevealedFromTheSideSchemeDeck", c)
			return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
		}
	}
	return nil
}

// registerHydraModulars installs the Hydra modular sets (04145–04154)
// and Weapon Master (04148–04151).
func registerHydraModulars() {
	engine.RegisterBehavior("04145", &engine.Behavior{})
	engine.RegisterBehavior("04146", &engine.Behavior{})
	engine.RegisterBehavior("04147", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var msgs []engine.Message
			for _, id := range sortedMinionIDs(g) {
				if m := g.Minions[id]; m != nil && m.EDef().HasTrait("hydra") && m.EngagedWith != "" {
					msgs = append(msgs, engine.MinionActivates{MinionID: id, Player: m.EngagedWith})
				}
			}
			return msgs
		},
	})
	for _, spec := range []struct{ code, icons string }{
		{"04148", "mental:1 physical:1"},
		{"04149", "mental:1 physical:1"},
	} {
		engine.RegisterBehavior(spec.code, &engine.Behavior{
			OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
				for id := range g.Villains {
					t.Target = id
					break
				}
				return nil
			},
			Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
				return []engine.Ability{{
					Label: engine.Tf("c.spendResourcesDiscardThisWeapon"), Type: engine.AbilityAction,
					CostIcons: spec.icons,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						return []engine.Message{engine.DiscardAttachmentMsg{ID: self}}
					},
				}}
			},
		})
	}
	engine.RegisterBehavior("04150", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for id := range g.Villains {
				return []engine.Message{engine.VillainActivates{VillainID: id, Player: p.ID}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("04151", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			if !p.IsHero() {
				if p.Confused && g.MainScheme != nil {
					return []engine.Message{engine.ConfuseEntity{Target: p.ID}, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: t.ID}}
				}
				if g.MainScheme != nil {
					return []engine.Message{engine.ConfuseEntity{Target: p.ID}, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: t.ID}}
				}
				return []engine.Message{engine.ConfuseEntity{Target: p.ID}}
			}
			if p.Stunned {
				return []engine.Message{engine.StunEntity{Target: p.ID}, engine.DamageEntity{Target: p.ID, Damage: 2, Source: t.ID}}
			}
			return []engine.Message{engine.StunEntity{Target: p.ID}, engine.DamageEntity{Target: p.ID, Damage: 1, Source: t.ID}}
		},
	})
	engine.RegisterBehavior("04152", &engine.Behavior{})
	engine.RegisterBehavior("04153", &engine.Behavior{})
	engine.RegisterBehavior("04154", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				for i, c := range g.EncounterDeck {
					if c.Def().Type == "minion" && c.Def().HasTrait("hydra") {
						g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
						msgs = append(msgs, engine.RevealEncounterCard{Player: p.ID, Card: c})
						break
					}
				}
			}
			return msgs
		},
	})
}

// registerCampaignExtras installs the campaign upgrades and obligations
// (04155–04166) and the duplicate Shang-Chi (10098).
func registerCampaignExtras() {
	oneShot := func(label string, exec func(g *engine.Game, self engine.EntityID) []engine.Message) *engine.Behavior {
		return &engine.Behavior{
			Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
				return []engine.Ability{{
					Label: engine.S(label + " (discard from the campaign)"), Type: engine.AbilityAction,
					HeroOnly: true,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						msgs := exec(g, self)
						return append([]engine.Message{engine.DiscardControlled{Player: g.Entity(self).EOwner(), ID: self}}, msgs...)
					},
				}}
			},
		}
	}
	engine.RegisterBehavior("04155", oneShot("Ready your hero and heal 5", func(g *engine.Game, self engine.EntityID) []engine.Message {
		return []engine.Message{engine.ReadyEntity{ID: g.Entity(self).EOwner()}, engine.HealEntity{Target: g.Entity(self).EOwner(), N: 5}}
	}))
	engine.RegisterBehavior("04156", oneShot("Draw 5 cards", func(g *engine.Game, self engine.EntityID) []engine.Message {
		return []engine.Message{engine.DrawCards{Player: g.Entity(self).EOwner(), N: 5}}
	}))
	engine.RegisterBehavior("04157", oneShot("Put an ally into play with tough", func(g *engine.Game, self engine.EntityID) []engine.Message {
		p := g.Player(g.Entity(self).EOwner())
		if p == nil {
			return nil
		}
		for i, c := range p.Deck {
			if c.Def().Type == "ally" {
				p.Deck = append(p.Deck[:i], p.Deck[i+1:]...)
				return []engine.Message{engine.PlayDiscardAlly{Player: p.ID, Card: c}}
			}
		}
		return nil
	}))
	engine.RegisterBehavior("04158", oneShot("Deal 5 damage to the villain", func(g *engine.Game, self engine.EntityID) []engine.Message {
		var msgs []engine.Message
		for id := range g.Villains {
			msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 5, Source: self})
			break
		}
		return msgs
	}))

	// 04159–04162 permanent stat upgrades.
	engine.RegisterBehavior("04159", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if p := g.Player(e.EOwner()); p != nil {
				p.MaxHP += 2
			}
			return nil
		},
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{THW: 1} },
	})
	engine.RegisterBehavior("04160", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if p := g.Player(e.EOwner()); p != nil {
				p.MaxHP++
			}
			return nil
		},
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{ATK: 1} },
	})
	engine.RegisterBehavior("04161", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if p := g.Player(e.EOwner()); p != nil {
				p.MaxHP += 3
			}
			return nil
		},
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{DEF: 1} },
	})
	engine.RegisterBehavior("04162", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if p := g.Player(e.EOwner()); p != nil {
				p.MaxHP += 4
			}
			return nil
		},
	})

	// 04163–04166 obligations.
	obligation := func(clear func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message) *engine.Behavior {
		return &engine.Behavior{ResolveObligation: clear}
	}
	engine.RegisterBehavior("04163", obligation(func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
		return []engine.Message{
			engine.ExhaustEntity{ID: p.ID},
			engine.ObligationResolve{Player: p.ID, Card: card, Remove: true},
		}
	}))
	engine.RegisterBehavior("04164", obligation(func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
		return []engine.Message{
			engine.MillPlayerDeck{Player: p.ID, N: 5},
			engine.ObligationResolve{Player: p.ID, Card: card, Remove: true},
		}
	}))
	engine.RegisterBehavior("04165", obligation(func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
		return []engine.Message{
			engine.DealEncounterToPlayer{Player: p.ID},
			engine.ObligationResolve{Player: p.ID, Card: card, Remove: true},
		}
	}))
	engine.RegisterBehavior("04166", obligation(func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
		return []engine.Message{
			engine.DamageEntity{Target: p.ID, Damage: 2},
			engine.ObligationResolve{Player: p.ID, Card: card, Remove: true},
		}
	}))

	// 10098 Shang-Chi duplicate.
	if b := engine.LookupBehavior("04098"); b != nil {
		engine.RegisterBehavior("10098", b)
	}
}

// registerTRORSScenarios registers the box's five scenarios.
func registerTRORSScenarios() {
	// Crossbones — Attack on Mount Athena.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "04061",
		Name:             "Crossbones — Attack on Mount Athena",
		VillainBases:     []string{"04058"},
		MainSchemeStages: []string{"04061", "04062", "04063"},
		ExtraSets:        []string{"exper_weapon", "hydra_assault", "weap_master", "standard"},
	})

	// Absorbing Man — None Shall Pass.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "04079",
		Name:             "Absorbing Man — None Shall Pass",
		VillainBases:     []string{"04076"},
		MainSchemeStages: []string{"04079"},
		ExtraSets:        []string{"hydra_patrol", "standard"},
	})

	// Taskmaster — Hunting Down Heroes.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "04096",
		Name:             "Taskmaster — Hunting Down Heroes",
		VillainBases:     []string{"04093"},
		MainSchemeStages: []string{"04096"},
		ExtraSets:        []string{"hydra_patrol", "weap_master", "standard"},
	})

	// Zola — The Island of Dr. Zola.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "04112",
		Name:             "Zola — The Island of Dr. Zola",
		VillainBases:     []string{"04109"},
		MainSchemeStages: []string{"04112", "04113"},
		ExtraSets:        []string{"standard"},
		Setup: func(g *engine.Game) []engine.Message {
			// Hydra Prison reveals; one Bio-Servant per player.
			for i, c := range g.EncounterDeck {
				if c.Code[:5] == "04122" {
					g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
					g.Push(engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c})
					break
				}
			}
			for _, p := range g.Players {
				for i, c := range g.EncounterDeck {
					if c.Code[:5] == "04114" {
						g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
						m := &engine.Minion{ID: g.NextEntityID("minion"), Code: "04114", MaxHP: 4, AttackVal: 2, SchemeVal: 1}
						g.AddMinion(m, p.ID)
						break
					}
				}
			}
			g.ShuffleEncounterDeck()
			return nil
		},
	})

	// Red Skull — The Rise of Red Skull.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "04128",
		Name:             "Red Skull — The Rise of Red Skull",
		VillainBases:     []string{"04125"},
		MainSchemeStages: []string{"04128", "04129"},
		ExtraSets:        []string{"hydra_assault", "hydra_patrol", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			// The Red House starts in play; the other schemes form the
			// side-scheme deck; The Sleeper is set aside.
			for _, code := range []string{"04140", "04141", "04142", "04143", "04144"} {
				var keep engine.CardList
				for _, c := range g.EncounterDeck {
					if c.Code[:5] == code {
						g.SetAside = append(g.SetAside, c)
					} else {
						keep = append(keep, c)
					}
				}
				g.EncounterDeck = keep
			}
			g.SetAside = append(g.SetAside, engine.Card{ID: g.NextCardID(), Code: "04130"})
			g.EncounterDeck = append(g.EncounterDeck, engine.Card{ID: g.NextCardID(), Code: "04139"})
			g.ShuffleEncounterDeck()
			return nil
		},
	})
}

func derefInt(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}
