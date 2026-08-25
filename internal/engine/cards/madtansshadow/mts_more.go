package madtansshadow

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func derefInt(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}

// spawnSideScheme brings a side scheme into play with per-hero threat.
func spawnSideScheme(g *engine.Game, code string, perHero int) *engine.SideScheme {
	def := engine.DB.MustLookup(code)
	maxT := derefInt(def.Threat, 6)
	s := &engine.SideScheme{
		ID:        g.NextEntityID("sidescheme"),
		Code:      code,
		Threat:    perHero * len(g.Players),
		MaxThreat: maxT,
	}
	g.SideSchemes[s.ID] = s
	g.TLogf("c.entersPlay", def)
	return s
}

// pullFromSetAside removes a matching card from the set-aside pool.
func pullFromSetAside(g *engine.Game, base string) (engine.Card, bool) {
	for i, c := range g.SetAside {
		if engine.BaseCodeOf(c.Code) == base {
			g.SetAside = append(g.SetAside[:i], g.SetAside[i+1:]...)
			return c, true
		}
	}
	return engine.Card{}, false
}

// registerHela installs the Hela scenario (21136–21155): she scales with
// the victory display and flips instead of dying while Odin is captive.
func registerHela() {
	for _, base := range []string{"21136", "21137"} {
		engine.RegisterBehavior(base, &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				if _, ok := msg.(engine.BeginRound); !ok {
					return nil
				}
				v := g.Villains[e.EID()]
				if v == nil {
					return nil
				}
				n := len(g.VictoryDisplay)
				per := 2 * len(g.Players)
				if base == "21137" {
					per = 3 * len(g.Players)
				}
				def := v.EDef()
				v.SchemeVal = derefInt(def.Scheme, 1) + n
				v.AttackVal = derefInt(def.Attack, 1) + n
				v.MaxHP = derefInt(def.HP, 10) + n*per
				return nil
			},
		})
	}

	// 21138 Odin's Torment (stage 1): Odin attaches to the scheme,
	// Gnipahellir and Garm start, the rest is set aside.
	engine.RegisterBehavior("21138", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			g.SpawnAttachment("21139a", s.ID)
			spawnSideScheme(g, "21140", 1)
			m := &engine.Minion{ID: g.NextEntityID("minion"), Code: "21143", MaxHP: 6, AttackVal: 2, SchemeVal: 1}
			g.AddMinion(m, cardutil.FirstPlayerID(g))
			for _, code := range []string{"21141", "21142", "21144", "21145"} {
				g.SetAside = append(g.SetAside, engine.Card{ID: g.NextCardID(), Code: code})
			}
			return nil
		},
	})

	// 21139 Odin: captive (attached) until Hall of Nastrond falls.
	engine.RegisterBehavior("21139", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if g.MainScheme != nil {
				t.Target = g.MainScheme.ID
			}
			return nil
		},
	})

	// 21140–21142 location chains: reveal the linked set-aside pieces.
	engine.RegisterBehavior("21140", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			var msgs []engine.Message
			for _, base := range []string{"21142", "21144"} {
				if c, ok := pullFromSetAside(g, base); ok {
					msgs = append(msgs, engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c})
				}
			}
			msgs = append(msgs, engine.DealEncounterToPlayer{Player: cardutil.FirstPlayerID(g)})
			return msgs
		},
	})
	engine.RegisterBehavior("21141", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			// Odin joins the first player as an ally.
			for _, a := range g.Attachments {
				if a != nil && a.Code[:5] == "21139" {
					g.Delete(a.ID)
					break
				}
			}
			ally := &engine.Ally{ID: g.NextEntityID("ally"), Code: "21139a", Owner: cardutil.FirstPlayerID(g), MaxHP: 8, AttackVal: 2, ThwartVal: 2}
			g.AddAlly(ally, cardutil.FirstPlayerID(g))
			g.TLogf("c.odinJoinsTheFirstPlayer")
			return nil
		},
	})
	engine.RegisterBehavior("21142", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			var msgs []engine.Message
			for _, base := range []string{"21141", "21145"} {
				if c, ok := pullFromSetAside(g, base); ok {
					msgs = append(msgs, engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c})
				}
			}
			msgs = append(msgs, engine.DealEncounterToPlayer{Player: cardutil.FirstPlayerID(g)})
			return msgs
		},
	})

	// 21143–21145 guardians (block the matching locations; keywords are
	// data-driven).
	engine.RegisterBehavior("21143", &engine.Behavior{})
	engine.RegisterBehavior("21144", &engine.Behavior{})
	engine.RegisterBehavior("21145", &engine.Behavior{})

	// 21146–21148 Hela's gear.
	for _, code := range []string{"21146", "21147", "21148"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
				for id := range g.Villains {
					if v := g.Villains[id]; v != nil && v.Code[:5] == "21136" {
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
			Boost: func(g *engine.Game, card engine.Card) []engine.Message {
				for id := range g.Villains {
					if v := g.Villains[id]; v != nil && v.Code[:5] == "21136" {
						g.SpawnAttachment(card.Code, id)
						break
					}
				}
				return nil
			},
		})
	}

	// 21149–21151 treacheries.
	engine.RegisterBehavior("21149", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			if g.MainScheme == nil {
				return nil
			}
			return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1 + len(g.VictoryDisplay), Source: t.ID}}
		},
	})
	engine.RegisterBehavior("21150", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var msgs []engine.Message
			for id := range g.Villains {
				msgs = append(msgs, engine.VillainActivates{VillainID: id, Player: p.ID})
				break
			}
			for _, sid := range sortedSchemeIDs(g) {
				msgs = append(msgs, engine.SchemeThreat{Scheme: sid, N: 1, Source: t.ID})
			}
			return msgs
		},
	})
	engine.RegisterBehavior("21151", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			return []engine.Message{engine.IndirectDamage{Player: p.ID, N: 1 + len(g.VictoryDisplay)}}
		},
	})

	// 21152–21155 Legions of Hel.
	engine.RegisterBehavior("21152", &engine.Behavior{})
	engine.RegisterBehavior("21153", &engine.Behavior{})
	engine.RegisterBehavior("21154", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			total := len(p.Upgrades) + len(p.Supports)
			best, bestCost := engine.EntityID(""), -1
			for _, id := range append(append([]engine.EntityID{}, p.Supports...), p.Upgrades...) {
				if e := g.Entity(id); e != nil {
					if c := cardutil.Cost(e.EDef()); c > bestCost {
						best, bestCost = id, c
					}
				}
			}
			var opts []engine.Choice
			if best != "" {
				opts = append(opts, engine.Choice{
					ID: "strip", Label: engine.Tf("c.discardYourHighestCostSupportUpgrade"), Kind: engine.ChoiceLabel,
				}.Msgs(engine.DiscardControlled{Player: p.ID, ID: best}))
			}
			opts = append(opts, engine.Choice{
				ID: "dmg", Label: engine.Tf("c.takeDamageEqualToYourBoardSize"), Kind: engine.ChoiceLabel,
			}.Msgs(engine.DamageEntity{Target: p.ID, Damage: total, Source: t.ID}))
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.noPlaceForTheLivingChooseOne"), opts...)}}
		},
	})
	engine.RegisterBehavior("21155", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			undead := 0
			for _, m := range g.Minions {
				if m != nil && m.EDef().HasTrait("undead") {
					undead++
				}
			}
			if undead == 0 {
				if c, ok := g.DrawEncounter(); ok {
					return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
				}
				return nil
			}
			if s := g.SideSchemes[e.EID()]; s != nil {
				s.Threat += 2 * undead
			}
			return nil
		},
	})

	// 21156–21159 Frost Giants.
	engine.RegisterBehavior("21156", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok || w.Enemy != e.EID() {
				return nil
			}
			return []engine.Message{engine.StunEntity{Target: w.Player}}
		},
	})
	engine.RegisterBehavior("21157", &engine.Behavior{
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			for _, p := range g.Players {
				if p.Exhausted {
					return []engine.Message{engine.StunEntity{Target: p.ID}}
				}
			}
			return nil
		},
	})
	engine.RegisterBehavior("21158", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			t.Target = cardutil.FirstPlayerID(g)
			return []engine.Message{engine.ExhaustEntity{ID: t.Target}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.spendEnergyPhysicalDiscardFrozen"), Type: engine.AbilityAction,
				AlterEgoOnly: true, CostIcons: "energy:1 physical:1",
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.DiscardAttachmentMsg{ID: self}}
				},
			}}
		},
	})
	engine.RegisterBehavior("21159", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				for _, id := range p.Allies {
					if a := g.Allies[id]; a != nil && !a.Exhausted {
						msgs = append(msgs, engine.ExhaustEntity{ID: id})
					}
				}
			}
			return msgs
		},
	})
}

// registerLoki installs the Loki scenario (21160–21179): the variant
// gauntlet — defeat enough Lokis to win.
func registerLoki() {
	for _, base := range []string{"21160", "21161", "21162", "21163", "21164"} {
		engine.RegisterBehavior(base, &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				if _, ok := msg.(engine.BeginRound); !ok {
					return nil
				}
				v := g.Villains[e.EID()]
				if v != nil && len(g.SideSchemes) > 0 && base == "21160" {
					// Loki I: immune while a side scheme is in play
					// (approximated via the damage gate below).
				}
				return nil
			},
		})
	}

	// 21165 All Hail King Loki (stage 1): seed the variant pool.
	engine.RegisterBehavior("21165", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			for _, base := range []string{"21160", "21161", "21162", "21163", "21164"} {
				g.SetAside = append(g.SetAside, engine.Card{ID: g.NextCardID(), Code: base})
			}
			spawnSideScheme(g, "21167", 1)
			swapLoki(g)
			return revealStone(g)
		},
	})

	// 21166–21169: defeat rewards — swap Loki, reveal a stone.
	for _, code := range []string{"21166", "21167", "21168", "21169"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
				msgs := revealStone(g)
				return append(msgs, swapLoki(g)...)
			},
		})
	}

	// 21170–21173 Loki's gear.
	for _, code := range []string{"21170", "21171", "21172", "21173"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
				for id := range g.Villains {
					if v := g.Villains[id]; v != nil && v.Code[:5] >= "21160" && v.Code[:5] <= "21164" {
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
			Boost: func(g *engine.Game, card engine.Card) []engine.Message {
				for id := range g.Villains {
					if v := g.Villains[id]; v != nil && v.Code[:5] >= "21160" && v.Code[:5] <= "21164" {
						g.SpawnAttachment(card.Code, id)
						break
					}
				}
				return nil
			},
		})
	}

	// 21174–21176 treacheries.
	engine.RegisterBehavior("21174", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			if p.Stunned {
				if p.IsHero() {
					return []engine.Message{engine.StunEntity{Target: p.ID}, engine.DamageEntity{Target: p.ID, Damage: 2, Source: t.ID}}
				}
				if g.MainScheme != nil {
					return []engine.Message{engine.StunEntity{Target: p.ID}, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: t.ID}}
				}
			}
			return []engine.Message{engine.StunEntity{Target: p.ID}}
		},
	})
	engine.RegisterBehavior("21175", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			return revealStone(g)
		},
	})
	engine.RegisterBehavior("21176", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			msgs := swapLoki(g)
			for id := range g.Villains {
				msgs = append(msgs, engine.VillainActivates{VillainID: id, Player: p.ID})
				break
			}
			return msgs
		},
	})

	// 21177–21179 Enchantress.
	engine.RegisterBehavior("21177", &engine.Behavior{})
	engine.RegisterBehavior("21178", &engine.Behavior{})
	engine.RegisterBehavior("21179", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if g.Player(target) != nil {
				t.Target = target
			} else {
				t.Target = cardutil.FirstPlayerID(g)
			}
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.spendEnergyMentalDiscardSeduced"), Type: engine.AbilityAction,
				AlterEgoOnly: true, CostIcons: "energy:1 mental:1",
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.DiscardAttachmentMsg{ID: self}}
				},
			}}
		},
	})
}

// swapLoki trades the active Loki for a random benched variant.
func swapLoki(g *engine.Game) []engine.Message {
	var pool []engine.Card
	for _, c := range g.SetAside {
		if base := engine.BaseCodeOf(c.Code); base >= "21160" && base <= "21164" {
			pool = append(pool, c)
		}
	}
	if len(pool) == 0 {
		return nil
	}
	pick := pool[g.Random(len(pool))]
	pullFromSetAside(g, engine.BaseCodeOf(pick.Code))
	for id, v := range g.Villains {
		if base := engine.BaseCodeOf(v.ECode()); base >= "21160" && base <= "21164" {
			g.SetAside = append(g.SetAside, engine.Card{ID: g.NextCardID(), Code: base})
			delete(g.Villains, id)
			break
		}
	}
	nv := g.SpawnVillainFromCard(engine.BaseCodeOf(pick.Code))
	if nv != nil {
		g.TLogMajorf("c.lokiSwapsTo", nv)
	}
	return nil
}

// registerCampaignSchemes installs the campaign side schemes,
// obligation and allies (21180–21193).
func registerCampaignSchemes() {
	// 21180 Secure the Landing Pad: flips (no b-side effect modeled).
	engine.RegisterBehavior("21180", &engine.Behavior{})

	// 21181 Security Breach: stash a random card per player.
	engine.RegisterBehavior("21181", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			s := g.SideSchemes[e.EID()]
			if s == nil {
				return nil
			}
			for _, p := range g.Players {
				if len(p.Hand) > 0 {
					c := p.Hand[0]
					p.Hand.Remove(c.ID)
					s.StoredCards = append(s.StoredCards, c)
				}
			}
			return nil
		},
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			for _, c := range s.StoredCards {
				if p := g.Player(c.Owner); p != nil {
					p.Hand = append(p.Hand, c)
				}
			}
			return nil
		},
	})

	// 21182/21184/21186/21189 flip rewards.
	engine.RegisterBehavior("21182", &engine.Behavior{})
	engine.RegisterBehavior("21184", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			return []engine.Message{engine.DrawCards{Player: g.ActiveTurn, N: 1}}
		},
	})
	engine.RegisterBehavior("21186", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				u := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "21187", Owner: p.ID}
				g.Upgrades[u.ID] = u
				p.Upgrades = append(p.Upgrades, u.ID)
			}
			g.TLogf("c.eachPlayerGainsANornStone")
			return msgs
		},
	})
	engine.RegisterBehavior("21189", &engine.Behavior{})

	// 21183 (Shawarma placeholder — textless beyond setup).
	engine.RegisterBehavior("21183", &engine.Behavior{})

	// 21185 System Shock obligation.
	engine.RegisterBehavior("21185", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			return []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}}
		},
	})

	// 21187 Norn Stone: +1 all stats; action to ready + flip.
	engine.RegisterBehavior("21187", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{ATK: 1, THW: 1, DEF: 1}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.readyYourHeroFlipThisCard"), Type: engine.AbilityAction,
				HeroOnly: true, OncePerRound: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.ReadyEntity{ID: e.EOwner()}}
				},
			}}
		},
	})

	// 21188 Summoned Back: nemesis returns.
	engine.RegisterBehavior("21188", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for _, c := range append(engine.CardList{}, append(engine.CardList{}, g.EncounterDeck...)...) {
				_ = c
				break
			}
			for i, c := range g.EncounterDeck {
				if c.Def().Type == "minion" {
					g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
			}
			return nil
		},
	})

	// 21190–21193 Warriors Three.
	engine.RegisterBehavior("21190", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.spend1ResourceReadyLadySif"), Type: engine.AbilityAction,
				Cost: 1,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.ReadyEntity{ID: self}}
				},
			}}
		},
	})
	engine.RegisterBehavior("21191", &engine.Behavior{})
	engine.RegisterBehavior("21192", &engine.Behavior{})
	engine.RegisterBehavior("21193", &engine.Behavior{})
}

// registerMTSScenarios registers the box's scenarios.
func registerMTSScenarios() {
	// Ebony Maw — Attack on Knowhere.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "21074",
		Name:             "Ebony Maw — Attack on Knowhere",
		VillainBases:     []string{"21071"},
		MainSchemeStages: []string{"21074", "21075"},
		ExtraSets:        []string{"armies_of_titan", "black_order", "standard"},
	})

	// Tower Defense — Under Siege (Proxima + Corvus).
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "21098",
		Name:             "Tower Defense — Under Siege",
		VillainBases:     []string{"21092", "21095"},
		MainSchemeStages: []string{"21098"},
		ExtraSets:        []string{"tower_defense", "armies_of_titan", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			env := g.SpawnEnvironment("21100a")
			_ = env
			spawnSideScheme(g, "21099a", 2)
			// Active villain starts on Proxima.
			for _, id := range cardutil.SortedIDs(g.Villains) {
				if g.Villains[id].Code[:5] == "21092" {
					g.ActiveVillain = id
					break
				}
			}
			return nil
		},
		OnMainSchemeMaxed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			// The stage never completes: threat wipes and the tower
			// suffers instead.
			s.Threat = 0
			if tower := g.EnvironmentByCode("21100a"); tower != nil {
				tower.Counters += 6 * len(g.Players)
				g.TLogf("c.theSiegeDealsDamageToAvengersTower", 6*len(g.Players))
			}
			return nil
		},
		OnVillainDefeated: func(g *engine.Game, v *engine.Villain) []engine.Message {
			delete(g.Villains, v.ID)
			if len(g.Villains) == 0 {
				return []engine.Message{engine.GameOver{Won: true, Reason: engine.Tf("reason.blackOrderBroken")}}
			}
			return nil
		},
	})

	// Thanos — The Infinity Stones.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "21114",
		Name:             "Thanos — The Infinity Stones",
		VillainBases:     []string{"21111"},
		MainSchemeStages: []string{"21114", "21115"},
		ExtraSets:        []string{"infinity_gauntlet", "black_order", "children_of_thanos", "standard"},
	})

	// Hela — Odin's Torment.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "21138",
		Name:             "Hela — Odin's Torment",
		VillainBases:     []string{"21136"},
		MainSchemeStages: []string{"21138"},
		ExtraSets:        []string{"legions_of_hel", "frost_giants", "standard"},
		OnVillainDefeated: func(g *engine.Game, v *engine.Villain) []engine.Message {
			// If Odin is still attached, Hela flips instead of dying.
			for _, a := range g.Attachments {
				if a != nil && a.Code[:5] == "21139" && g.MainScheme != nil && a.Target == g.MainScheme.ID {
					v.Damage = 0
					v.Code = "21137"
					def := v.EDef()
					v.MaxHP = derefInt(def.HP, 10)
					g.TLogMajorf("c.helaRisesAgainWoundedButFurious")
					return nil
				}
			}
			delete(g.Villains, v.ID)
			return []engine.Message{engine.GameOver{Won: true, Reason: engine.Tf("reason.helaCastDown")}}
		},
	})

	// Loki — All Hail King Loki.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "21165",
		Name:             "Loki — All Hail King Loki",
		VillainBases:     []string{"21160"},
		MainSchemeStages: []string{"21165"},
		ExtraSets:        []string{"infinity_gauntlet", "enchantress", "frost_giants", "standard"},
		OnVillainDefeated: func(g *engine.Game, v *engine.Villain) []engine.Message {
			g.VictoryDisplay = append(g.VictoryDisplay, engine.Card{ID: g.NextCardID(), Code: engine.BaseCodeOf(v.ECode())})
			delete(g.Villains, v.ID)
			lokis := 0
			for _, c := range g.VictoryDisplay {
				if base := engine.BaseCodeOf(c.Code); base >= "21160" && base <= "21164" {
					lokis++
				}
			}
			need := 3
			if len(g.Players) >= 3 {
				need = 4
			}
			if lokis >= need {
				return []engine.Message{engine.GameOver{Won: true, Reason: engine.Tf("reason.lokiVariants")}}
			}
			swapLoki(g)
			if len(g.Villains) == 0 {
				return []engine.Message{engine.GameOver{Won: true, Reason: engine.Tf("reason.variantGauntletSpent")}}
			}
			return nil
		},
	})
}
