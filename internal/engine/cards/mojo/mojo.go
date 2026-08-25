// Package mojo registers Mojo Mania: the Melee in the Mojo-seum (MaGog),
// Across the Mojoverse (Spiral) and MojoMania (Mojo) scenarios plus the
// genre Show modular sets.
//
// The box's "place N threat on a character" rider (incite) is not
// representable — those effects deal damage instead, noted per card.
package mojo

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerMaGog()
	registerSpiral()
	registerMojoVillain()
	registerMojoScenarios()
}

// envByCode finds an environment by base code.
func envByCode(g *engine.Game, base string) *engine.Environment {
	for _, e := range g.Environments {
		if e != nil && engine.BaseCodeOf(e.Code) == base {
			return e
		}
	}
	return nil
}

// ratings adds n counters to The Champion (39003) or The Challengers
// (39004), flipping the environment at 5[per_hero].
func ratings(g *engine.Game, base string, n int) {
	env := envByCode(g, base)
	if env == nil || n == 0 {
		return
	}
	env.Counters += n
	g.TLogf("c.gainsRatingsCounters", env, n, env.Counters)
	if env.Counters >= 5*len(g.Players) {
		if env.Code == base+"a" {
			env.Code = base + "b"
		} else if env.Code == base+"b" {
			env.Code = base + "a"
		}
		env.Counters = 0
		g.TLogf("c.theCrowdFlips", env)
	}
}

// magog returns the MaGog villain.
func magog(g *engine.Game) *engine.Villain {
	for _, v := range g.Villains {
		if v != nil && engine.BaseCodeOf(v.Code) == "39001" {
			return v
		}
	}
	return nil
}

func registerMaGog() {
	// Main scheme and crowd boards: the ratings flips live in ratings().
	for _, code := range []string{"39002", "39003", "39004"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	// 39001 MaGog: after he attacks and damages, 1 rating on The Champion;
	// a would-be defeat resets his hit points and rewards The Challengers.
	engine.RegisterBehavior("39001", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.AskAttack); !ok {
				return nil
			}
			if v := g.Villains[e.EID()]; v != nil && engine.BaseCodeOf(v.Code) == "39001" {
				ratings(g, "39003", 1)
			}
			return nil
		},
		VillainDamageable: func(g *engine.Game, v *engine.Villain, damage int) bool {
			if v.HP()-damage > 0 {
				return true
			}
			v.Damage = 0
			if v.MaxHP > 10*len(g.Players) {
				v.Damage = v.MaxHP - 10*len(g.Players)
			}
			ratings(g, "39004", 3*len(g.Players))
			g.TLogf("c.magogPowersThroughHisHitPointsResetTheCrowdGoesWild")
			return false
		},
	})

	// 39005/39006 Jolt of Adrenaline & Surge of Aggression: attach to
	// MaGog; after the reset, +1[per_hero] ratings on The Challengers and
	// discard.
	for _, code := range []string{"39005", "39006"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
				if v := magog(g); v != nil {
					t.Target = v.ID
				}
				return nil
			},
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				// The reset itself fires through MaGog's damage gate; the
				// riders piggyback on the same window (any damage event
				// while MaGog is at the reset threshold).
				t := g.Attachments[e.EID()]
				if t == nil {
					return nil
				}
				if v := magog(g); v != nil && v.Damage == v.MaxHP-10*len(g.Players) {
					ratings(g, "39004", len(g.Players))
					g.Delete(t.ID)
				}
				return nil
			},
		})
	}

	// 39007 Surprise Contender: attacks → 1 rating on The Champion;
	// defeated → 2[per_hero] ratings on The Challengers.
	engine.RegisterBehavior("39007", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			switch m := msg.(type) {
			case engine.AskAttack:
				if m.Enemy == e.EID() {
					ratings(g, "39003", 1)
				}
			case engine.MinionDefeated:
				if m.MinionID == e.EID() {
					ratings(g, "39004", 2*len(g.Players))
				}
			}
			return nil
		},
	})

	// 39008 Pump Up the Crowd: +1[per_hero] threat on reveal while The
	// Champion crowd cheers; defeated → ratings for The Challengers.
	engine.RegisterBehavior("39008", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if env := envByCode(g, "39003"); env != nil && env.Code == "39003b" {
				return []engine.Message{engine.SchemeThreat{Scheme: e.EID(), N: len(g.Players), Source: e.EID()}}
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			ratings(g, "39004", len(g.Players))
			return nil
		},
	})

	// 39009 Break a Leg: stunned + 2 damage (4 when the Challengers lead).
	engine.RegisterBehavior("39009", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			dmg := 2
			ch := envByCode(g, "39004")
			cp := envByCode(g, "39003")
			if ch != nil && cp != nil && ch.Counters > cp.Counters {
				dmg = 4
			}
			return []engine.Message{
				engine.StunEntity{Target: p.ID},
				engine.DamageEntity{Target: p.ID, Damage: dmg, Source: t.ID},
			}
		},
	})

	// 39010 Defend the Title: alter-ego → 2 ratings on The Champion;
	// hero → MaGog attacks you.
	engine.RegisterBehavior("39010", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			if !p.IsHero() {
				ratings(g, "39003", 2)
				return nil
			}
			if v := magog(g); v != nil {
				return []engine.Message{engine.AskAttack{Enemy: v.ID, Player: p.ID, Trigger: engine.TriggerVillainAttacksYou}}
			}
			return nil
		},
	})

	// 39011 Stage Fright: confused + 2 threat (4 when the Challengers
	// lead).
	engine.RegisterBehavior("39011", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			msgs := []engine.Message{engine.ConfuseEntity{Target: p.ID}}
			if g.MainScheme != nil {
				n := 2
				ch := envByCode(g, "39004")
				cp := envByCode(g, "39003")
				if ch != nil && cp != nil && ch.Counters > cp.Counters {
					n = 4
				}
				msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: n, Source: t.ID})
			}
			return msgs
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return []engine.Message{engine.ConfuseEntity{Target: cardutil.FirstPlayerID(g)}}
		},
	})
}

// spiral returns the Spiral villain.
func spiral(g *engine.Game) *engine.Villain {
	for _, v := range g.Villains {
		if v != nil && engine.BaseCodeOf(v.Code) == "39012" {
			return v
		}
	}
	return nil
}

func registerSpiral() {
	engine.RegisterBehavior("39015", &engine.Behavior{})

	// 39012 Spiral: cannot take damage (the stun immunity is folded in);
	// main-scheme threat is locked (enforced in removeThreat); attacks
	// become schemes.
	for _, code := range []string{"39012", "39013", "39014"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			// Spiral cannot take damage.
			VillainDamageable: func(g *engine.Game, v *engine.Villain, damage int) bool {
				return false
			},
			// When she would attack, she schemes instead (both forms).
			VillainActivate: func(g *engine.Game, v *engine.Villain, p *engine.Player) []engine.Message {
				if g.MainScheme != nil {
					return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: v.SchemeVal, Source: v.ID}}
				}
				return nil
			},
		})
	}

	// 39016 The Search for Spiral: when cleared, reveal the top of the
	// show deck (approximated by the encounter deck) and re-set
	// 3[per_hero] threat; hero action: take 2 damage → remove 3.
	engine.RegisterBehavior("39016", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.ThwartScheme)
			s := g.SideSchemes[e.EID()]
			if !ok || s == nil || m.Scheme != s.ID || s.Threat > m.N {
				return nil
			}
			pid := m.Source
			msgs := []engine.Message{engine.SchemeThreat{Scheme: s.ID, N: 3 * len(g.Players), Source: s.ID}}
			if pid.Is(engine.KindPlayer) {
				msgs = append(msgs, engine.RevealNextEncounter{Player: engine.PlayerID(pid)})
			}
			return msgs
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.theSearchForSpiralTake2DamageRemove3Threat"), Type: engine.AbilityAction,
				HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{
						engine.DamageEntity{Target: g.ActiveTurn, Damage: 2, Source: self},
						engine.ThwartScheme{Scheme: self, N: 3, Source: g.ActiveTurn},
					}
				},
			}}
		},
	})

	// 39017 Cornered!: flip Spiral to her Cornered side and reveal the
	// next show card.
	engine.RegisterBehavior("39017", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			if v := spiral(g); v != nil && !v.EDef().HasTrait("cornered") {
				// Flip to the b-side (Cornered).
				for _, c := range []string{"39012b", "39013b", "39014b"} {
					if engine.BaseCodeOf(v.Code)+"b" == c {
						v.Code = c
						break
					}
				}
				g.TLogf("c.spiralIsCornered")
			}
			return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
		},
	})

	// 39018 Spiral's Swords: attach to Spiral, 3 sword counters (+1 ATK
	// each on attach); spend [physical][physical] → remove one.
	engine.RegisterBehavior("39018", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			t.Counters = 3
			if v := spiral(g); v != nil {
				t.Target = v.ID
				return []engine.Message{engine.BoostEnemyAttack{Enemy: v.ID, N: 3}}
			}
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			t := g.Attachments[e.EID()]
			if t == nil || t.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.spiralSSwordsSpendPhysicalPhysicalRemove1SwordCounter"), Type: engine.AbilityAction,
				Cost: 2, CostIcons: "physical:2",
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					t := g.Attachments[self]
					if t == nil || t.Counters <= 0 || t.Target == "" {
						return nil
					}
					t.Counters--
					return []engine.Message{engine.BoostEnemyAttack{Enemy: t.Target, N: -1}}
				},
			}}
		},
	})

	// 39019 Erratic Teleportation: Surge from data; the peek/counter
	// riders are approximated away.
	engine.RegisterBehavior("39019", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			return nil
		},
	})

	// 39020 The Show Must Go On: each player is dealt a facedown
	// encounter card.
	engine.RegisterBehavior("39020", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var msgs []engine.Message
			for _, tp := range g.Players {
				msgs = append(msgs, engine.DealEncounterToPlayer{Player: tp.ID})
			}
			return msgs
		},
	})

	// 39021 Well-Armed: swords attached → Spiral attacks; otherwise
	// attach one.
	engine.RegisterBehavior("39021", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			v := spiral(g)
			if v == nil {
				return nil
			}
			for _, id := range cardutil.SortedIDs(g.Attachments) {
				if a := g.Attachments[id]; a != nil && a.Code == "39018" && a.Target == v.ID {
					return []engine.Message{engine.AskAttack{Enemy: v.ID, Player: p.ID, Trigger: engine.TriggerVillainAttacksYou}}
				}
			}
			for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
				for _, c := range *zone {
					if c.Code == "39018" {
						zone.Remove(c.ID)
						return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
					}
				}
			}
			return nil
		},
	})
}

// mojoVillain returns the Mojo villain.
func mojoVillain(g *engine.Game) *engine.Villain {
	for _, v := range g.Villains {
		if v != nil && engine.BaseCodeOf(v.Code) == "39022" {
			return v
		}
	}
	return nil
}

func registerMojoVillain() {
	engine.RegisterBehavior("39025", &engine.Behavior{})

	// 39022–39024 Mojo: stage reveals place "threat" on friendly
	// characters (approximated as damage); after a hero turn ends, mill
	// and pay per non-Mojo card.
	for i, code := range []string{"39022", "39023", "39024"} {
		stage := i + 1
		mill := 2 + stage
		engine.RegisterBehavior(code, &engine.Behavior{
			// Undercover Mojo: damage to Mojo removes threat from that
			// scheme instead.
			VillainDamageable: func(g *engine.Game, v *engine.Villain, damage int) bool {
				for _, s := range g.SideSchemes {
					if s == nil || s.Code != "39031" {
						continue
					}
					divert := damage
					if divert > s.Threat {
						divert = s.Threat
					}
					if divert > 0 {
						s.Threat -= divert
						remaining := damage - divert
						g.TLogf("c.undercoverMojoAbsorbsDamage", divert)
						if remaining > 0 {
							v.Damage += remaining
						}
						return false
					}
				}
				return true
			},
			VillainStage: func(g *engine.Game, v *engine.Villain, nextStage int) []engine.Message {
				if stage == 1 {
					return nil
				}
				var msgs []engine.Message
				for _, p := range g.Players {
					n := stage // 2 on II, 3 on III (threat approximated)
					msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: n, Source: v.ID})
				}
				return msgs
			},
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				m, ok := msg.(engine.PlayerTurnEnd)
				v := g.Villains[e.EID()]
				if !ok || v == nil || v.ID != e.EID() {
					return nil
				}
				p := g.Player(m.Player)
				if p == nil || !p.IsHero() {
					return nil
				}
				hits := 0
				for i := 0; i < mill; i++ {
					if len(g.EncounterDeck) == 0 {
						break
					}
					top := g.EncounterDeck[0]
					g.EncounterDeck = g.EncounterDeck[1:]
					g.EncounterDiscard = append(g.EncounterDiscard, top)
					if top.Def().CardSet != "mojo" {
						hits++
					}
				}
				if hits > 0 {
					g.TLogf("c.theRatingsWarDealsDamage", p.Name, hits)
					return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: hits, Source: v.ID}}
				}
				return nil
			},
		})
	}

	// 39026 Wheel of Genres: the reshuffle hook is not modeled; the
	// set-aside watch is handled at setup.
	engine.RegisterBehavior("39026", &engine.Behavior{})

	// 39027 Major Domo: attach to Mojo; spend [energy][mental][physical]
	// to discard (the mill rider approximated away).
	engine.RegisterBehavior("39027", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if v := mojoVillain(g); v != nil {
				t.Target = v.ID
			}
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			t := g.Attachments[e.EID()]
			if t == nil {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.majorDomoSpendEnergyMentalPhysicalDiscard"), Type: engine.AbilityAction,
				Cost: 3, CostIcons: "energy:1 mental:1 physical:1",
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.DiscardAttachmentMsg{ID: self}}
				},
			}}
		},
	})

	// 39028 Stinger Tail: attach to Mojo, retaliate 2, damage stored here
	// (pops at 5).
	engine.RegisterBehavior("39028", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if v := mojoVillain(g); v != nil {
				t.Target = v.ID
			}
			return nil
		},
	})

	// 39029 Supporting Actor: threat-on-self and boost riders approximate
	// to nothing.
	engine.RegisterBehavior("39029", &engine.Behavior{})

	// 39030 Paparazzi: when the turn ends its threat moves to the main
	// scheme (approximated: 2 threat).
	engine.RegisterBehavior("39030", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			msgs := []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
			if g.MainScheme != nil {
				return append([]engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: p.ID}}, msgs...)
			}
			return msgs
		},
	})

	// 39031 Undercover Mojo: damage to Mojo removes threat here instead.
	engine.RegisterBehavior("39031", &engine.Behavior{})

	// 39032 Curtain Call: 1 damage to each controlled character
	// (threat-on-characters approximation).
	engine.RegisterBehavior("39032", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			msgs := []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 1, Source: t.ID}}
			for _, id := range p.Allies {
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: t.ID})
			}
			return msgs
		},
	})

	// 39033 Director's Directions: spend 2, Mojo schemes, or take damage.
	engine.RegisterBehavior("39033", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			choices := []engine.Choice{
				engine.Choice{ID: "scheme", Label: engine.Tf("c.mojoSchemes"), Kind: engine.ChoiceLabel},
				engine.Choice{ID: "damage", Label: engine.Tf("c.take1DamageThreatOnYourIdentityApproximated"), Kind: engine.ChoiceLabel}.
					Msgs(engine.DamageEntity{Target: p.ID, Damage: 1, Source: t.ID}),
			}
			choices = append(choices, engine.Choice{
				ID: "spend", Label: engine.Tf("c.spend2ResourcesDiscard2Cards"), Kind: engine.ChoiceLabel,
			}.Msgs(engine.DiscardCards{Player: p.ID, Cards: handTake(p, 2)}))
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.directorSDirectionsChoose"), choices...),
			}}
		},
	})

	// 39034 Top Billing: heal 1 each friendly character, then 2
	// "threat" (1 damage approximation).
	engine.RegisterBehavior("39034", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			msgs := []engine.Message{engine.HealEntity{Target: p.ID, N: 1}, engine.DamageEntity{Target: p.ID, Damage: 1, Source: t.ID}}
			for _, id := range p.Allies {
				msgs = append(msgs, engine.HealEntity{Target: id, N: 1}, engine.DamageEntity{Target: id, Damage: 1, Source: t.ID})
			}
			return msgs
		},
	})
}

// handTake returns up to n hand cards (for forced discards).
func handTake(p *engine.Player, n int) engine.CardList {
	var out engine.CardList
	for i := 0; i < n && i < len(p.Hand); i++ {
		out = append(out, p.Hand[i])
	}
	return out
}

func registerMojoScenarios() {
	// Melee in the Mojo-seum (MaGog).
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "39002",
		Name:             "MaGog — Melee in the Mojo-seum",
		VillainBases:     []string{"39001"},
		MainSchemeStages: []string{"39002b"},
		ExtraSets:        []string{"magog", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			g.SpawnEnvironment("39003a")
			g.SpawnEnvironment("39004a")
			return nil
		},
		OnVillainDefeated: func(g *engine.Game, v *engine.Villain) []engine.Message {
			return []engine.Message{engine.GameOver{Won: true, Reason: engine.Tf("reason.magogOut")}}
		},
	})

	// Across the Mojoverse (Spiral).
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "39015",
		Name:             "Spiral — Across the Mojoverse",
		VillainBases:     []string{"39012"},
		MainSchemeStages: []string{"39015b"},
		ExtraSets:        []string{"spiral", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			// The Search for Spiral starts in play.
			for i, c := range g.EncounterDeck {
				if c.Code == "39016" {
					g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
					return []engine.Message{engine.RevealEncounterCard{Player: g.Players[0].ID, Card: c}}
				}
			}
			return nil
		},
	})

	// MojoMania (Mojo with the Wheel of Genres).
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "39025",
		Name:             "Mojo — MojoMania",
		VillainBases:     []string{"39022"},
		MainSchemeStages: []string{"39025b"},
		ExtraSets:        []string{"mojo", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			g.SpawnEnvironment("39026a")
			return nil
		},
	})
}
