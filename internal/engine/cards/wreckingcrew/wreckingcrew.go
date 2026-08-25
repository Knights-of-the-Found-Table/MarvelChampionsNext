// Package wreckingcrew registers "The Wrecking Crew" scenario: four
// villains, each bound to a personal side scheme, and an active counter
// that cards move between them. Approximations are noted inline: the
// active counter is engine state (Game.ActiveVillain), villain schemes
// route threat to their own side scheme, and the players win only when
// all four villains are defeated.
package wreckingcrew

import (
	"sort"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// villain base -> their side scheme code.
var crewSchemes = map[string]string{
	"07002": "07004", // Wrecker -> Day of Reckoning
	"07017": "07019", // Thunderball -> Thunderstruck
	"07032": "07034", // Piledriver -> Pile It On!
	"07046": "07048", // Bulldozer -> Clear the Road
}

func init() {
	registerScenario()
	registerCrew()
	registerShared()
}

func registerScenario() {
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:   "07001",
		Name: "The Wrecking Crew — Breakout",
		// The four villain decks are gathered via ExtraSets; the villain
		// stages resolve from their bases.
		VillainBases:     []string{"07002", "07017", "07032", "07046"},
		MainSchemeStages: []string{"07001b"},
		ExtraSets:        []string{"wrecking_crew_shared", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			// Personal side schemes enter play and the active counter
			// starts on Wrecker.
			for _, vid := range cardutil.SortedIDs(g.Villains) {
				v := g.Villains[vid]
				if code, ok := crewSchemes[v.Code[:5]]; ok {
					spawnCrewScheme(g, code, v)
				}
			}
			for _, vid := range cardutil.SortedIDs(g.Villains) {
				if g.Villains[vid].Code[:5] == "07002" {
					g.ActiveVillain = vid
					g.TLogf("c.theActiveCounterStartsOn", g.Villains[vid].EDef().Name)
					break
				}
			}
			return nil
		},
		OnVillainDefeated: func(g *engine.Game, v *engine.Villain) []engine.Message {
			delete(g.Villains, v.ID)
			g.TLogf("c.isDefeatedForGood", v)
			if g.ActiveVillain == v.ID {
				setActiveLeastThreat(g)
			}
			if len(g.Villains) == 0 {
				return []engine.Message{engine.GameOver{Won: true, Reason: engine.Tf("reason.wreckingCrewDefeated")}}
			}
			return nil
		},
	})
	// 1A contents marker.
	engine.RegisterBehavior("07001", &engine.Behavior{})
}

func spawnCrewScheme(g *engine.Game, code string, v *engine.Villain) {
	def, ok := engine.DB.Lookup(code)
	if !ok {
		return
	}
	s := &engine.SideScheme{
		ID:        g.NextEntityID("side_scheme"),
		Code:      def.Code,
		Threat:    deref(def.BaseThreat, 3),
		MaxThreat: deref(def.Threat, 10),
	}
	g.SideSchemes[s.ID] = s
	g.TLogf("c.entersPlaySSideScheme", def.Name, v)
}

// schemeOfVillain finds the crew side scheme owned by the villain's base.
func schemeOfVillain(g *engine.Game, vid engine.EntityID) engine.EntityID {
	v := g.Villains[vid]
	if v == nil {
		return ""
	}
	code := crewSchemes[v.Code[:5]]
	for _, s := range g.SideSchemes {
		if s.Code[:5] == code {
			return s.ID
		}
	}
	return ""
}

// setActiveLeastThreat moves the active counter to the villain whose side
// scheme has the least threat.
func setActiveLeastThreat(g *engine.Game) {
	var best engine.EntityID
	bestThreat := -1
	for _, vid := range cardutil.SortedIDs(g.Villains) {
		if _, ok := g.Villains[vid]; !ok {
			continue
		}
		sid := schemeOfVillain(g, vid)
		t := 1 << 30
		if sid != "" {
			if s := g.SideSchemes[sid]; s != nil {
				t = s.Threat
			}
		}
		if bestThreat < 0 || t < bestThreat {
			best, bestThreat = vid, t
		}
	}
	if best != "" && best != g.ActiveVillain {
		g.ActiveVillain = best
		g.TLogf("c.theActiveCounterMovesTo", g.Villains[best].EDef().Name)
	}
}

func villainLeastThreat(g *engine.Game) engine.EntityID {
	var best engine.EntityID
	bestThreat := -1
	for _, vid := range cardutil.SortedIDs(g.Villains) {
		if _, ok := g.Villains[vid]; !ok {
			continue
		}
		t := 1 << 30
		if sid := schemeOfVillain(g, vid); sid != "" {
			if s := g.SideSchemes[sid]; s != nil {
				t = s.Threat
			}
		}
		if bestThreat < 0 || t < bestThreat {
			best, bestThreat = vid, t
		}
	}
	return best
}

func villainMostThreat(g *engine.Game) engine.EntityID {
	var best engine.EntityID
	bestThreat := -1
	for _, vid := range cardutil.SortedIDs(g.Villains) {
		if _, ok := g.Villains[vid]; !ok {
			continue
		}
		t := -1
		if sid := schemeOfVillain(g, vid); sid != "" {
			if s := g.SideSchemes[sid]; s != nil {
				t = s.Threat
			}
		}
		if t > bestThreat {
			best, bestThreat = vid, t
		}
	}
	return best
}

// registerCrew installs the four villains and their side schemes.
func registerCrew() {
	// Wrecker (07002): schemes to his scheme; +2 ATK undefended.
	for _, base := range []string{"07002", "07003"} {
		engine.RegisterBehavior(base, &engine.Behavior{
			VillainActivate: crewActivate(base, "07002"),
		})
	}
	// Thunderball (07017): splash 1 to controlled characters after attack.
	for _, base := range []string{"07017", "07018"} {
		engine.RegisterBehavior(base, &engine.Behavior{
			VillainActivate: crewActivate(base, "07017"),
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				d, ok := msg.(engine.DamageEntity)
				if !ok || d.Source != e.EID() || !d.Target.Is(engine.KindPlayer) {
					return nil
				}
				p := g.Player(d.Target)
				if p == nil {
					return nil
				}
				var msgs []engine.Message
				msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: 1, Source: e.EID()})
				for _, id := range p.Allies {
					msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: e.EID()})
				}
				return msgs
			},
		})
	}
	// Piledriver (07032): Retaliate 1 printed.
	for _, base := range []string{"07032", "07033"} {
		engine.RegisterBehavior(base, &engine.Behavior{
			VillainActivate: crewActivate(base, "07032"),
		})
	}
	// Bulldozer (07046): overkill approximated as +1 spill to the player.
	for _, base := range []string{"07046", "07047"} {
		engine.RegisterBehavior(base, &engine.Behavior{
			VillainActivate: crewActivate(base, "07046"),
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				w, ok := msg.(engine.WindowDefended)
				if !ok || w.Against != e.EID() || w.DamageTaken <= 0 {
					return nil
				}
				// Overkill: spill hits the controlling player (ally
				// defense). WindowAfterEnemyAttacked does not carry the
				// damage amount; 1 spill as a minimal approximation.
				if w.Defender.Is(engine.KindAlly) {
					if a := g.Allies[w.Defender]; a != nil {
						if p := g.Player(a.Owner); p != nil {
							return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 1, Source: e.EID()}}
						}
					}
				}
				return nil
			},
		})
	}

	// Personal side schemes: cannot leave play while the owner lives;
	// at 10+ threat the Forced Response fires and resets to 3.
	for base, scheme := range crewSchemes {
		schemeCode := scheme
		ownerBase := base
		engine.RegisterBehavior(schemeCode, &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				s := g.SideSchemes[e.EID()]
				if s == nil || s.Threat < 10 {
					return nil
				}
				// Trigger and reset.
				var msgs []engine.Message
				switch schemeCode {
				case "07004": // Day of Reckoning: 2 damage to each friendly.
					for _, p := range g.Players {
						msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: 2, Source: e.EID()})
						for _, id := range p.Allies {
							msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 2, Source: e.EID()})
						}
					}
				case "07019": // Thunderstruck: stun each friendly.
					for _, p := range g.Players {
						msgs = append(msgs, engine.StunEntity{Target: p.ID})
						for _, id := range p.Allies {
							msgs = append(msgs, engine.StunEntity{Target: id})
						}
					}
				case "07034": // Pile It On!: highest-cost support/upgrade discarded.
					for _, p := range g.Players {
						discardHighestCost(g, p)
					}
				case "07048": // Clear the Road: each player mills 10.
					for _, p := range g.Players {
						msgs = append(msgs, engine.MillPlayerDeck{Player: p.ID, N: 10})
					}
				}
				s.Threat = 3
				g.TLogf("c.triggersAndResetsTo3Threat", s)
				_ = ownerBase
				return msgs
			},
		})
	}
}

// crewActivate builds a VillainActivate that routes schemes to the
// villain's own side scheme and runs standard attack flow otherwise.
func crewActivate(base, ownerBase string) func(g *engine.Game, v *engine.Villain, p *engine.Player) []engine.Message {
	return func(g *engine.Game, v *engine.Villain, p *engine.Player) []engine.Message {
		if p.IsHero() {
			if v.Stunned {
				v.Stunned = false
				g.TLogf("log.stunnedCanceled", v)
				return nil
			}
			g.TLogf("log.attacks", v, p.Name)
			g.Push(engine.DealBoost{Enemy: v.ID})
			g.Push(engine.RevealBoost{Enemy: v.ID})
			g.Push(engine.AskAttack{Enemy: v.ID, Player: p.ID, Trigger: engine.TriggerVillainAttacksYou})
			return nil
		}
		// Scheme onto his own side scheme instead of the main scheme.
		if v.Confused {
			v.Confused = false
			g.TLogf("log.confusedCanceled", v)
			return nil
		}
		sid := schemeOfVillain(g, v.ID)
		if sid == "" {
			sid = engine.EntityID("")
		}
		target := sid
		n := v.SchemeVal + v.BoostCount
		if target == "" && g.MainScheme != nil {
			target = g.MainScheme.ID
		}
		g.TLogf("c.schemesAgainstThreatToHisSideScheme", v, p.Name)
		g.Push(engine.SchemeThreat{Scheme: target, N: n, Source: v.ID})
		g.Push(engine.ClearBoosts{Enemy: v.ID})
		// No boost cards are dealt for scheming villains per scenario
		// intent; keep the standard flow minimal.
		return nil
	}
}

func discardHighestCost(g *engine.Game, p *engine.Player) {
	best := ""
	bestCost := -1
	for _, id := range p.Supports {
		if s := g.Supports[id]; s != nil {
			if c := deref(s.EDef().Cost, 0); c > bestCost {
				best, bestCost = "support:"+string(id), c
			}
		}
	}
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil {
			if c := deref(u.EDef().Cost, 0); c > bestCost {
				best, bestCost = "upgrade:"+string(id), c
			}
		}
	}
	if best == "" {
		return
	}
	kind, id := best[:7], engine.EntityID(best[7:])
	if kind == "support" {
		g.Push(engine.DiscardControlled{Player: p.ID, ID: id})
	} else {
		g.Push(engine.DiscardControlled{Player: p.ID, ID: id})
	}
}

// registerShared installs the shared minions, attachments and treacheries.
func registerShared() {
	// Corrupt Prison Guard: Guard only.
	for _, code := range []string{"07008", "07023", "07037", "07052"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	// Escaped Convict: Surge (engine) + boost moves the active counter and
	// the villain attacks the hero without a boost card.
	esc := func(g *engine.Game, card engine.Card) []engine.Message {
		setActiveLeastThreat(g)
		var msgs []engine.Message
		if g.ActiveVillain != "" {
			msgs = append(msgs, engine.AskAttack{Enemy: g.ActiveVillain, Player: engine.PlayerID(card.Owner)})
		}
		return msgs
	}
	for _, code := range []string{"07009", "07024", "07038", "07053"} {
		engine.RegisterBehavior(code, &engine.Behavior{Boost: esc})
	}

	// Buddy System: the least-threat villain reveals encounter cards
	// (approximated: reveal to the first player).
	buddy := func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		n := 1
		if len(g.Villains) == 1 {
			n = 2
		}
		var msgs []engine.Message
		for i := 0; i < n; i++ {
			if c, ok := g.DrawEncounter(); ok {
				msgs = append(msgs, engine.RevealEncounterCard{Player: p.ID, Card: c})
			}
		}
		return msgs
	}
	for _, code := range []string{"07010", "07025", "07039", "07054"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			ResolveTreachery: buddy,
			Boost: func(g *engine.Game, card engine.Card) []engine.Message {
				setActiveLeastThreat(g)
				return nil
			},
		})
	}

	// Chaos In the Prison: discard an upgrade or 1 threat per upgrade.
	chaos := func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		if len(p.Upgrades) == 0 {
			return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
		}
		var picks []engine.Choice
		for _, id := range p.Upgrades {
			u := g.Upgrades[id]
			if u != nil {
				picks = append(picks, engine.Choice{Label: engine.S("Discard " + u.EDef().Name), Kind: engine.ChoiceCard, CardCode: u.Code}.
					Msgs(engine.DiscardControlled{Player: p.ID, ID: id}))
			}
		}
		active := g.ActiveVillain
		sid := schemeOfVillain(g, active)
		if sid != "" {
			picks = append(picks, engine.Choice{
				ID: "threat", Label: engine.Tf("c.placeThreatOnTheActiveVillainSScheme", len(p.Upgrades)),
				Kind: engine.ChoiceLabel,
			}.Msgs(engine.SchemeThreat{Scheme: sid, N: len(p.Upgrades), Source: t.ID}))
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID,
			Question: engine.Ask(engine.Tf("c.chaosInThePrison"), picks...)}}
	}
	for _, code := range []string{"07011", "07026", "07056"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			ResolveTreachery: chaos,
			Boost: func(g *engine.Game, card engine.Card) []engine.Message {
				// Undefended-detection unavailable; no boost effect.
				return nil
			},
		})
	}

	// Energy Projectiles: 1 damage to each friendly; boost hits the
	// defender (approximated: the first player).
	for _, code := range []string{"07027"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
				var msgs []engine.Message
				msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: 1, Source: t.ID})
				for _, id := range p.Allies {
					msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: t.ID})
				}
				return msgs
			},
		})
	}

	// Get Wrecked!: most-threat schemes / least-threat attacks.
	gw := func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		if !p.IsHero() {
			vid := villainMostThreat(g)
			return villainSchemes(g, vid, p)
		}
		vid := villainLeastThreat(g)
		return villainAttacks(g, vid, p)
	}
	for _, code := range []string{"07013", "07028", "07040", "07057"} {
		engine.RegisterBehavior(code, &engine.Behavior{ResolveTreachery: gw})
	}

	// I've Been Waiting For This!: active villain heals 3 + tough; boost
	// moves the counter and the new villain schemes.
	wait := func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		if v := g.Villains[g.ActiveVillain]; v != nil {
			if v.Damage > 0 {
				v.Damage -= min(3, v.Damage)
			}
			return []engine.Message{engine.ToughEntity{Target: v.ID}}
		}
		return nil
	}
	for _, code := range []string{"07014", "07029", "07041"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			ResolveTreachery: wait,
			Boost: func(g *engine.Game, card engine.Card) []engine.Message {
				setActiveLeastThreat(g)
				if g.ActiveVillain != "" {
					return []engine.Message{engine.ApplyVillainScheme{VillainID: g.ActiveVillain, Player: engine.PlayerID(card.Owner)}}
				}
				return nil
			},
		})
	}

	// Mystical Link: 2 threat on each crew scheme.
	engine.RegisterBehavior("07015", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var msgs []engine.Message
			for _, s := range g.SideSchemes {
				msgs = append(msgs, engine.SchemeThreat{Scheme: s.ID, N: 2, Source: t.ID})
			}
			return msgs
		},
	})

	// You're Dead Meat!: 1 damage to the weakest friendly.
	engine.RegisterBehavior("07016", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var weakest engine.EntityID
			hp := 1 << 30
			if p.HP() < hp {
				weakest, hp = p.ID, p.HP()
			}
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && a.HP() < hp {
					weakest, hp = a.ID, a.HP()
				}
			}
			if weakest == "" {
				return nil
			}
			dmg := []engine.Message{engine.DamageEntity{Target: weakest, Damage: 1, Source: t.ID}}
			// If that defeats the character: 3 threat on Wrecker's scheme.
			if a := g.Allies[weakest]; a != nil && a.HP()-1 <= 0 {
				if sid := schemeOfVillainByBase(g, "07002"); sid != "" {
					dmg = append(dmg, engine.SchemeThreat{Scheme: sid, N: 3, Source: t.ID})
				}
			} else if weakest == p.ID && p.HP()-1 <= 0 {
				if sid := schemeOfVillainByBase(g, "07002"); sid != "" {
					dmg = append(dmg, engine.SchemeThreat{Scheme: sid, N: 3, Source: t.ID})
				}
			}
			return dmg
		},
	})

	// Lightning Blast: Thunderball focus.
	engine.RegisterBehavior("07030", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if !p.IsHero() {
				sid := schemeOfVillainByBase(g, "07017")
				if sid == "" {
					return nil
				}
				return []engine.Message{engine.SchemeThreat{Scheme: sid, N: 3, Source: t.ID}}
			}
			if vid := villainByBase(g, "07017"); vid != "" {
				return villainAttacks(g, vid, p)
			}
			return nil
		},
	})

	// Tactical Prowess: move all threat least -> most.
	engine.RegisterBehavior("07031", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var schemes []*engine.SideScheme
			for _, id := range cardutil.SortedIDs(g.SideSchemes) {
				if s := g.SideSchemes[id]; s != nil {
					schemes = append(schemes, s)
				}
			}
			if len(schemes) < 2 {
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			sort.SliceStable(schemes, func(i, j int) bool { return schemes[i].Threat < schemes[j].Threat })
			least, most := schemes[0], schemes[len(schemes)-1]
			n := least.Threat
			least.Threat = 0
			most.Threat += n
			g.TLogf("c.tacticalProwessMovesThreatFromTo", n, least, most)
			return nil
		},
	})

	// Crowbar Toss: Wrecker focus.
	engine.RegisterBehavior("07012", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			vid := villainByBase(g, "07002")
			var msgs []engine.Message
			if vid != "" {
				if !p.IsHero() {
					msgs = villainSchemes(g, vid, p)
				} else {
					msgs = villainAttacks(g, vid, p)
				}
			}
			setActiveLeastThreat(g)
			return msgs
		},
	})

	// Oversized Hands: discard highest-cost support or +2 threat.
	engine.RegisterBehavior("07042", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			best := ""
			bestCost := -1
			for _, id := range p.Supports {
				if s := g.Supports[id]; s != nil {
					if c := deref(s.EDef().Cost, 0); c > bestCost {
						best, bestCost = string(id), c
					}
				}
			}
			if best != "" {
				return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: engine.EntityID(best)}}
			}
			if sid := schemeOfVillain(g, g.ActiveVillain); sid != "" {
				return []engine.Message{engine.SchemeThreat{Scheme: sid, N: 2, Source: t.ID}}
			}
			return nil
		},
	})

	// Escape Plan: confused; if already confused Piledriver schemes.
	engine.RegisterBehavior("07043", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if p.Confused {
				if vid := villainByBase(g, "07032"); vid != "" {
					return villainSchemes(g, vid, p)
				}
			}
			return []engine.Message{engine.ConfuseEntity{Target: p.ID}}
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if v := g.Villains[g.ActiveVillain]; v != nil {
				if v.Tough {
					if sid := schemeOfVillain(g, v.ID); sid != "" {
						return []engine.Message{engine.SchemeThreat{Scheme: sid, N: 2, Source: engine.EntityID("07043")}}
					}
				}
				return []engine.Message{engine.ToughEntity{Target: v.ID}}
			}
			return nil
		},
	})

	// Pummel: Piledriver focus with tough riders.
	engine.RegisterBehavior("07044", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			vid := villainByBase(g, "07032")
			if vid == "" {
				return nil
			}
			v := g.Villains[vid]
			if !p.IsHero() {
				msgs := villainSchemes(g, vid, p)
				if v.Tough {
					// +2 SCH: add threat directly.
					if sid := schemeOfVillain(g, vid); sid != "" {
						msgs = append(msgs, engine.SchemeThreat{Scheme: sid, N: 2, Source: t.ID})
					}
				}
				return msgs
			}
			msgs := villainAttacks(g, vid, p)
			if v.Tough {
				msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: 2, Source: t.ID})
			}
			return msgs
		},
	})

	// Uncanny Resilience: clear villain statuses or surge.
	engine.RegisterBehavior("07045", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			cleared := false
			for _, v := range g.Villains {
				if v.Stunned || v.Confused || v.Tough {
					cleared = true
				}
				v.Stunned, v.Confused, v.Tough = false, false, false
			}
			if !cleared {
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			return nil
		},
	})

	// Bull Rush: Bulldozer focus with mill riders.
	engine.RegisterBehavior("07055", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			vid := villainByBase(g, "07046")
			if vid == "" {
				return nil
			}
			v := g.Villains[vid]
			if !p.IsHero() {
				msgs := villainSchemes(g, vid, p)
				msgs = append(msgs, engine.MillPlayerDeck{Player: p.ID, N: v.SchemeVal})
				return msgs
			}
			msgs := villainAttacks(g, vid, p)
			msgs = append(msgs, engine.MillPlayerDeck{Player: p.ID, N: v.AttackVal})
			return msgs
		},
	})

	// Headbutt: random discard, cost-based damage/threat.
	engine.RegisterBehavior("07058", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if len(p.Hand) == 0 {
				return nil
			}
			i := g.Random(len(p.Hand))
			c := p.Hand[i]
			p.Hand = append(p.Hand[:i], p.Hand[i+1:]...)
			p.Discard = append(p.Discard, c)
			cost := deref(c.Def().Cost, 0)
			if p.IsHero() {
				return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: cost, Source: t.ID}}
			}
			if sid := schemeOfVillainByBase(g, "07046"); sid != "" {
				return []engine.Message{engine.SchemeThreat{Scheme: sid, N: cost, Source: t.ID}}
			}
			return nil
		},
	})

	// Leading the Charge: mill X = Bulldozer ATK, +1 threat per type.
	engine.RegisterBehavior("07059", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			vid := villainByBase(g, "07046")
			n := 2
			if v := g.Villains[vid]; v != nil {
				n = v.AttackVal
			}
			types := map[string]bool{}
			for i := 0; i < n && len(p.Deck) > 0; i++ {
				c := p.Deck[0]
				p.Deck = p.Deck[1:]
				p.Discard = append(p.Discard, c)
				types[c.Def().Type] = true
			}
			if len(types) > 0 {
				if sid := schemeOfVillainByBase(g, "07046"); sid != "" {
					return []engine.Message{engine.SchemeThreat{Scheme: sid, N: len(types), Source: t.ID}}
				}
			}
			return nil
		},
	})

	registerCrewAttachments()
}

// registerCrewAttachments installs the villain attachments.
func registerCrewAttachments() {
	// Held Hostage (×4): attach to the active villain's scheme; that
	// scheme cannot be thwarted; hero action provokes the attack then
	// discards.
	for _, code := range []string{"07005", "07021", "07036", "07050"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
				if sid := schemeOfVillain(g, g.ActiveVillain); sid != "" {
					t.Target = sid
					g.TLogf("c.heldHostageAttachesTo", g.SideSchemes[sid].EDef().Name)
				}
				return nil
			},
			Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
				return []engine.Ability{{
					Label: engine.Tf("c.provokeTheActiveVillainSAttackThenDiscardHeldHostage"), Type: engine.AbilityAction,
					HeroOnly: true,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						a := g.Attachments[self]
						if a == nil {
							return nil
						}
						// Find the villain owning the attached scheme.
						var msgs []engine.Message
						for vid := range g.Villains {
							if schemeOfVillain(g, vid) == a.Target {
								msgs = append(msgs, engine.DealBoost{Enemy: vid}, engine.RevealBoost{Enemy: vid},
									engine.AskAttack{Enemy: vid, Player: engine.PlayerID(actingOwner(g, self)), Trigger: engine.TriggerVillainAttacksYou})
								break
							}
						}
						g.Delete(self)
						g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: a.Code})
						return msgs
					},
				}}
			},
		})
	}

	// Magic Crowbar: Wrecker +1 threat to least scheme after attacks.
	engine.RegisterBehavior("07006", &engine.Behavior{
		React: afterVillainAttacks("07002", func(g *engine.Game, a *engine.Attachment, w engine.WindowAfterEnemyAttacked) []engine.Message {
			var msgs []engine.Message
			if sid := leastThreatScheme(g); sid != "" {
				msgs = append(msgs, engine.SchemeThreat{Scheme: sid, N: 1, Source: a.ID})
			}
			return msgs
		}),
		Abilities: exhaustRandomDiscardRemoval("Magic Crowbar"),
	})

	// Wrecker's Command: after Wrecker schemes, +1 threat to each other
	// crew scheme; physical×2 removal.
	engine.RegisterBehavior("07007", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			a := g.Attachments[e.EID()]
			sch, ok := msg.(engine.ApplyVillainScheme)
			if !ok || a == nil || sch.VillainID != a.Target {
				return nil
			}
			var msgs []engine.Message
			for _, s := range g.SideSchemes {
				if s.ID != schemeOfVillain(g, a.Target) {
					msgs = append(msgs, engine.SchemeThreat{Scheme: s.ID, N: 1, Source: a.ID})
				}
			}
			return msgs
		},
		Abilities: iconRemoval("Spend [physical] [physical] → discard Wrecker's Command", "physical:2", 2),
	})

	// Ball and Chain: after Thunderball attacks, +1 threat main scheme.
	engine.RegisterBehavior("07020", &engine.Behavior{
		React: afterVillainAttacks("07017", func(g *engine.Game, a *engine.Attachment, w engine.WindowAfterEnemyAttacked) []engine.Message {
			if g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: a.ID}}
			}
			return nil
		}),
		Abilities: exhaustRandomDiscardRemoval("Ball and Chain"),
	})

	// Radioactive Buildup: excess damage as threat; discarded after the
	// attack (approximated: discard after any Thunderball attack).
	engine.RegisterBehavior("07022", &engine.Behavior{
		React: afterVillainAttacks("07017", func(g *engine.Game, a *engine.Attachment, w engine.WindowAfterEnemyAttacked) []engine.Message {
			g.Delete(a.ID)
			g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: a.Code})
			return nil
		}),
	})

	// Distracting Taunts: Piledriver +3 HP; other villains untargetable
	// (not enforced — guard-style targeting lock unsupported).
	engine.RegisterBehavior("07035", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if v := g.Villains[target]; v != nil {
				v.MaxHP += 3
			}
			return nil
		},
		Abilities: iconRemoval("Spend [physical] [physical] → discard Distracting Taunts", "physical:2", 2),
	})

	// Bulldozer's Helmet: mill per damage dealt.
	engine.RegisterBehavior("07049", &engine.Behavior{
		React: afterVillainAttacks("07046", func(g *engine.Game, a *engine.Attachment, w engine.WindowAfterEnemyAttacked) []engine.Message {
			// Damage amount is not carried on this window; mill 1 as a
			// minimal approximation of "per point of damage".
			return []engine.Message{engine.MillPlayerDeck{Player: w.Player, N: 1}}
		}),
		Abilities: exhaustRandomDiscardRemoval("Bulldozer's Helmet"),
	})

	// Ramming Speed: force-ally defense + discard after attack.
	engine.RegisterBehavior("07051", &engine.Behavior{
		React: afterVillainAttacks("07046", func(g *engine.Game, a *engine.Attachment, w engine.WindowAfterEnemyAttacked) []engine.Message {
			g.Delete(a.ID)
			g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: a.Code})
			return nil
		}),
	})
}

// ---- shared helpers ----

func deref(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}

func villainByBase(g *engine.Game, base string) engine.EntityID {
	for id, v := range g.Villains {
		if v.Code[:5] == base {
			return id
		}
	}
	return ""
}

func schemeOfVillainByBase(g *engine.Game, base string) engine.EntityID {
	if vid := villainByBase(g, base); vid != "" {
		return schemeOfVillain(g, vid)
	}
	return ""
}

func villainAttacks(g *engine.Game, vid engine.EntityID, p *engine.Player) []engine.Message {
	if vid == "" || g.Villains[vid] == nil {
		return nil
	}
	return []engine.Message{
		engine.DealBoost{Enemy: vid}, engine.RevealBoost{Enemy: vid},
		engine.AskAttack{Enemy: vid, Player: p.ID, Trigger: engine.TriggerVillainAttacksYou},
	}
}

func villainSchemes(g *engine.Game, vid engine.EntityID, p *engine.Player) []engine.Message {
	if vid == "" || g.Villains[vid] == nil {
		return nil
	}
	return []engine.Message{
		engine.DealBoost{Enemy: vid}, engine.RevealBoost{Enemy: vid},
		engine.ApplyVillainScheme{VillainID: vid, Player: p.ID},
	}
}

func leastThreatScheme(g *engine.Game) engine.EntityID {
	var best engine.EntityID
	t := 1 << 30
	for _, s := range g.SideSchemes {
		if s.Threat < t {
			best, t = s.ID, s.Threat
		}
	}
	return best
}

func actingOwner(g *engine.Game, id engine.EntityID) string {
	if g.ActiveTurn != "" {
		return string(g.ActiveTurn)
	}
	return string(g.Players[0].ID)
}

// afterVillainAttacks builds a React on WindowAfterEnemyAttacked filtered
// to the attached villain base.
func afterVillainAttacks(base string, run func(g *engine.Game, a *engine.Attachment, w engine.WindowAfterEnemyAttacked) []engine.Message) func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
	return func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		w, ok := msg.(engine.WindowAfterEnemyAttacked)
		a := g.Attachments[e.EID()]
		if !ok || a == nil {
			return nil
		}
		v := g.Villains[w.Enemy]
		if v == nil || v.Code[:5] != base {
			return nil
		}
		return run(g, a, w)
	}
}

func exhaustRandomDiscardRemoval(name string) func(g *engine.Game, e engine.Entity) []engine.Ability {
	return func(g *engine.Game, e engine.Entity) []engine.Ability {
		return []engine.Ability{{
			Label: engine.S("Exhaust your hero + discard 1 random card → discard " + name), Type: engine.AbilityAction,
			HeroOnly: true,
			Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
				a := g.Attachments[self]
				if a == nil {
					return nil
				}
				pid := engine.PlayerID(actingOwner(g, self))
				p := g.Player(pid)
				if p != nil && len(p.Hand) > 0 {
					i := g.Random(len(p.Hand))
					c := p.Hand[i]
					p.Hand = append(p.Hand[:i], p.Hand[i+1:]...)
					p.Discard = append(p.Discard, c)
				}
				g.Delete(self)
				g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: a.Code})
				return []engine.Message{engine.ExhaustEntity{ID: pid}}
			},
		}}
	}
}

func iconRemoval(label, icons string, cost int) func(g *engine.Game, e engine.Entity) []engine.Ability {
	return func(g *engine.Game, e engine.Entity) []engine.Ability {
		return []engine.Ability{{
			Label: engine.S(label), Type: engine.AbilityAction, Cost: cost, CostIcons: icons, HeroOnly: true,
			Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
				a := g.Attachments[self]
				if a == nil {
					return nil
				}
				g.Delete(self)
				g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: a.Code})
				return nil
			},
		}}
	}
}
