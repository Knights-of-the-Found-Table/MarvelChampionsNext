package sinistermotives

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// registerVenomScenario installs the Venom scenario (27073–27083) with
// the Bell Tower chime-counter race.
func registerVenomScenario() {
	// Venom I–III (27073–27075): Vengeance stacks boost cards on the
	// attacker (banked directly on the villain — per-identity piles are
	// approximated away).
	for _, base := range []string{"27073", "27074", "27075"} {
		b := &engine.Behavior{}
		b.React = func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.DamageEntity)
			if !ok || d.Target != e.EID() || d.Damage <= 0 {
				return nil
			}
			if !(d.Source.Is(engine.KindPlayer) || d.Source.Is(engine.KindAlly)) {
				return nil
			}
			g.TLogf("c.vengeanceVenomBanksAFacedownBoostCard")
			return []engine.Message{engine.DealBoost{Enemy: e.EID()}}
		}
		if base != "27073" {
			b.VillainStage = func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
				if base == "27074" {
					// Search for Tooth and Nail.
					for i, c := range g.EncounterDeck {
						if c.Code[:5] == "27081" {
							g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
							g.ShuffleEncounterDeck()
							return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
						}
					}
					for i, c := range g.EncounterDiscard {
						if c.Code[:5] == "27081" {
							g.EncounterDiscard = append(g.EncounterDiscard[:i], g.EncounterDiscard[i+1:]...)
							return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
						}
					}
					return nil
				}
				// 27075: 2 facedown encounter cards each.
				var msgs []engine.Message
				for range g.Players {
					msgs = append(msgs, engine.DealBoost{Enemy: v.ID}, engine.DealBoost{Enemy: v.ID})
				}
				return msgs
			}
		}
		engine.RegisterBehavior(base, b)
	}

	// 27076 "Leave Us Alone!": Bell Tower enters play (QUIET side).
	engine.RegisterBehavior("27076", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			if g.EnvironmentByCode("27077a") == nil && g.EnvironmentByCode("27077b") == nil {
				g.SpawnEnvironment("27077a")
			}
			return nil
		},
	})

	// 27077 Bell Tower: the chime race (flip at 3[per_hero], represented
	// by swapping the environment code; damage is NOT prevented — the
	// counters only track the race).
	engine.RegisterBehavior("27077", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.DamageEntity)
			if !ok || d.Target == "" {
				return nil
			}
			v := g.Villains[d.Target]
			if v == nil || v.Code[:5] != "27073" {
				return nil
			}
			env := g.Environments[e.EID()]
			if env == nil {
				return nil
			}
			// Chiming converts the damage into counters (approximated:
			// prevent the damage by pre-marking; the handler still runs
			// so damage is applied — the counters are the tracked race).
			env.Counters += d.Damage
			g.TLogf("c.theBellTowerChimesCounters", env.Counters)
			if env.Counters >= 3*len(g.Players) && env.Code == "27077a" {
				env.Code = "27077b"
				g.TLogMajorf("c.theBellTowerRings")
			}
			return nil
		},
	})

	// 27078 "Now We're Angry!": overkill rage (2 counters).
	engine.RegisterBehavior("27078", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			t.Counters = 2
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil && v.Code[:5] == "27073" {
					t.Target = id
					break
				}
			}
			return nil
		},
	})

	// 27079 Guard the Bell Tower: chime wipe + reset to QUIET.
	engine.RegisterBehavior("27079", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for _, env := range g.Environments {
				if env != nil && env.Code[:5] == "27077" {
					env.Counters = 0
					env.Code = "27077a"
					g.TLogf("c.theBellTowerFallsSilent")
				}
			}
			return nil
		},
	})

	// 27080/27081 Lashing Out / Tooth and Nail: threat melts as Venom is
	// hurt (approximated: each attack against Venom removes 2).
	for _, code := range []string{"27080", "27081"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				w, ok := msg.(engine.WindowAfterEnemyAttacked)
				if !ok {
					return nil
				}
				s := g.SideSchemes[e.EID()]
				if s == nil || s.Threat <= 0 {
					return nil
				}
				v := g.Villains[w.Enemy]
				if v == nil || v.Code[:5] != "27073" {
					return nil
				}
				return []engine.Message{engine.ThwartScheme{Scheme: e.EID(), N: min(2, s.Threat), Source: e.EID()}}
			},
		})
	}

	// 27082 Biting Retort: Venom activates with juiced boosts.
	engine.RegisterBehavior("27082", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for id := range g.Villains {
				return []engine.Message{
					engine.BoostActivation{Enemy: id, N: 1},
					engine.VillainActivates{VillainID: id, Player: p.ID},
				}
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			for _, env := range g.Environments {
				if env != nil && env.Code[:5] == "27077" && env.Counters > 0 {
					env.Counters--
					g.TLogf("c.bitingRetortRemovesAChimeCounter")
				}
			}
			return nil
		},
	})

	// 27083 For Whom the Bell Tolls: chime tax.
	engine.RegisterBehavior("27083", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			env := g.EnvironmentByCode("27077b")
			if env == nil {
				env = g.EnvironmentByCode("27077a")
			}
			if env == nil {
				return nil
			}
			drop := min(2, env.Counters)
			env.Counters -= drop
			if env.Code == "27077b" {
				if g.MainScheme != nil {
					return []engine.Message{engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 1, Source: t.ID}}
				}
				return nil
			}
			return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 1, Source: t.ID}}
		},
	})
}

// registerMysterio installs the Mysterio scenario (27084–27093). The
// illusion-boost redirection targets the first player (approximation).
func registerMysterio() {
	routeIllusion := func(g *engine.Game, v *engine.Villain, to string) {
		fp := g.Player(cardutil.FirstPlayerID(g))
		if fp == nil {
			return
		}
		var kept engine.CardList
		for _, c := range v.RevealedBoosts {
			if c.Def().HasTrait("illusion") {
				switch to {
				case "discard":
					fp.Discard = append(fp.Discard, c)
					g.TLogf("c.joinsSDiscardPile", c, fp.Name)
					continue
				case "bottom":
					fp.Deck = append(fp.Deck, c)
					g.TLogf("c.goesToTheBottomOfSDeck", c, fp.Name)
					continue
				case "top":
					fp.Deck = append([]engine.Card{c}, fp.Deck...)
					g.TLogf("c.goesOnTopOfSDeck", c, fp.Name)
					continue
				}
			}
			kept = append(kept, c)
		}
		v.RevealedBoosts = kept
	}

	for _, spec := range []struct{ base, dest string }{
		{"27084", "discard"}, {"27085", "bottom"}, {"27086", "top"},
	} {
		dest := spec.dest
		b := &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				if cb, ok := msg.(engine.ClearBoosts); ok && cb.Enemy == e.EID() {
					if v := g.Villains[cb.Enemy]; v != nil {
						routeIllusion(g, v, dest)
					}
				}
				return nil
			},
		}
		if spec.base != "27084" {
			b.VillainStage = func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
				if spec.base == "27085" {
					// Shuffle the encounter top into each player deck —
					// skipped (encounter cards in player decks are not
					// modeled); noted approximation.
					g.TLogf("c.creepingFearWhispersDeckSeedingApproximatedAway")
					return nil
				}
				var msgs []engine.Message
				for _, p := range g.Players {
					msgs = append(msgs, engine.MillPlayerDeck{Player: p.ID, N: 5})
				}
				return msgs
			}
		}
		engine.RegisterBehavior(spec.base, b)
	}

	// 27087 Maze of Mirrors (stage 1): Apparitions engage everyone.
	engine.RegisterBehavior("27087", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				m := &engine.Minion{ID: g.NextEntityID("minion"), Code: "27091", MaxHP: 3, AttackVal: 1, SchemeVal: 1}
				g.AddMinion(m, p.ID)
				g.TLogf("c.aShiftingApparitionEngages", p.Name)
			}
			return msgs
		},
	})

	// 27088 Edge of Reality (stage 2): the deck-seeding rider is
	// approximated away.
	engine.RegisterBehavior("27088", &engine.Behavior{})

	// 27089 Humongous Hallucination: buy-off shuffles encounter cards
	// into your deck (approximated: mills your deck by 2 instead).
	engine.RegisterBehavior("27089", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil && v.Code[:5] == "27084" {
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
				Label: engine.Tf("c.spend1ResourceDiscardHumongousHallucination"), Type: engine.AbilityAction,
				Cost: 1,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{
						engine.MillEncounter{N: 2},
						engine.DiscardAttachmentMsg{ID: self},
					}
				},
			}}
		},
	})

	// 27090 Masterful Mirage: damage to Mysterio mills you 4 instead.
	engine.RegisterBehavior("27090", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil && v.Code[:5] == "27084" {
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

	// 27091 Shifting Apparition: Guard keyword (data-driven).
	engine.RegisterBehavior("27091", &engine.Behavior{})

	// 27092 Déjà Vu: damage or threat, then shuffles into a deck
	// (approximated: into the encounter deck).
	engine.RegisterBehavior("27092", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			g.EncounterDeck = append(g.EncounterDeck, engine.Card{ID: g.NextCardID(), Code: "27092"})
			g.ShuffleEncounterDeck()
			var msgs []engine.Message
			if g.MainScheme != nil {
				msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: t.ID})
			}
			return msgs
		},
	})

	// 27093 Fearmonger: discard hand, redraw.
	engine.RegisterBehavior("27093", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			n := len(p.Hand)
			p.Discard = append(p.Discard, p.Hand...)
			p.Hand = nil
			return []engine.Message{engine.DrawCards{Player: p.ID, N: min(n, p.HandSize(g))}}
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return []engine.Message{engine.DealEncounterToPlayer{Player: cardutil.FirstPlayerID(g)}}
		},
	})
}
