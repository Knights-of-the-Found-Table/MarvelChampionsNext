package galaxysmostwanted

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// registerRonanScenario installs the Ronan the Accuser scenario (16103–
// 16116): the Universal Weapon, Fanaticism and the Power Stone duel.
func registerRonanScenario() {
	// Ronan I–III (16103–16105): extra boost card when activating against
	// the Power Stone holder; stages II/III fetch their side scheme.
	for _, base := range []string{"16103", "16104", "16105"} {
		scheme := ""
		if base == "16104" {
			scheme = "16111"
		} else if base == "16105" {
			scheme = "16113"
		}
		b := &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				m, ok := msg.(engine.VillainActivates)
				if !ok || m.VillainID != e.EID() {
					return nil
				}
				holder := powerStoneHolder(g)
				if holder != "" && holder == m.Player {
					g.Logf("Ronan draws an extra boost card against the Power Stone bearer")
					return []engine.Message{engine.BoostActivation{Enemy: e.EID(), N: 1}}
				}
				return nil
			},
		}
		if scheme != "" {
			b.VillainStage = fetchSchemeStage(scheme)
		}
		engine.RegisterBehavior(base, b)
	}

	// 16106 Interception Imminent (stage 1): ship + Milano, Universal
	// Weapon on Ronan, Power Stone on the first player.
	engine.RegisterBehavior("16106", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			if findEnvironment(g, "16108") == "" {
				g.SpawnEnvironment("16108")
			}
			if _, mil := findMilano(g); mil == nil {
				g.SpawnSupport("16142", cardutil.FirstPlayerID(g))
			}
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil && v.Code[:5] == "16103" {
					g.SpawnAttachment("16109", id)
					break
				}
			}
			if ps := findPowerStone(g); ps == nil {
				g.SpawnAttachment("16149", cardutil.FirstPlayerID(g))
			}
			return nil
		},
	})

	// 16107 "Take What Is Mine" (stage 2): Power Stone to Ronan (or a
	// boost card).
	engine.RegisterBehavior("16107", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil && v.Code[:5] == "16103" {
					if ps := findPowerStone(g); ps != nil && ps.Target != id {
						ps.Target = id
						g.LogMajorf("The Power Stone leaps to Ronan the Accuser")
					} else if ps != nil {
						return []engine.Message{engine.DealBoost{Enemy: id}}
					}
				}
			}
			return nil
		},
	})

	// 16108 Kree Command Ship: cancel a revealed treachery via the
	// Milano (approximated as an always-available cancel prompt).
	engine.RegisterBehavior("16108", &engine.Behavior{})

	// 16109 Universal Weapon: attach to Ronan; buy-off shuffles it away.
	engine.RegisterBehavior("16109", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil && v.Code[:5] == "16103" {
					t.Target = id
					g.Logf("Universal Weapon attaches to Ronan the Accuser")
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
				Label: "Take 2 damage + 1 facedown encounter card → shuffle Universal Weapon into the encounter deck", Type: engine.AbilityAction,
				HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					a := g.Attachments[self]
					owner := cardutil.FirstPlayerID(g)
					code := "16109"
					if a != nil {
						code = a.Code
						if a.Target.Is(engine.KindPlayer) {
							owner = a.Target
						}
					}
					g.Delete(self)
					g.EncounterDeck = append(g.EncounterDeck, engine.Card{ID: g.NextCardID(), Code: code})
					g.ShuffleEncounterDeck()
					g.Logf("Universal Weapon shuffles into the encounter deck")
					return []engine.Message{
						engine.DamageEntity{Target: owner, Damage: 2, Source: self},
						engine.DealEncounterToPlayer{Player: owner},
					}
				},
			}}
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil && v.Code[:5] == "16103" {
					g.SpawnAttachment("16109", id)
					break
				}
			}
			return nil
		},
	})

	// 16110 Fanaticism: 1[per_hero] fury counters; attacks gain overkill
	// and piercing, burning a counter each time.
	engine.RegisterBehavior("16110", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			t.Counters = 1 + len(g.Players)
			g.Logf("Fanaticism enters play with %d fury counters", t.Counters)
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.VillainActivates)
			if !ok {
				return nil
			}
			f := g.Attachments[e.EID()]
			p := g.Player(m.Player)
			if f == nil || f.Counters <= 0 || p == nil || !p.IsHero() {
				return nil
			}
			if v := g.Villains[m.VillainID]; v != nil && v.Code[:5] == "16103" {
				f.Counters--
				g.Logf("Fanaticism empowers the attack (+2 ATK, overkill, piercing)")
				return []engine.Message{engine.BoostActivation{Enemy: m.VillainID, N: 2}}
			}
			return nil
		},
	})

	// 16111 Cut the Power: boost taxes the Milano or the scheme.
	engine.RegisterBehavior("16111", &engine.Behavior{
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			_, mil := findMilano(g)
			if mil == nil || mil.Exhausted || g.MainScheme == nil {
				if g.MainScheme != nil {
					return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2}}
				}
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: cardutil.FirstPlayerID(g), Question: engine.Ask(
				"Cut the Power: choose one",
				engine.Choice{
					ID: "milano", Label: "Exhaust the Milano", Kind: engine.ChoiceLabel,
				}.Msgs(engine.ExhaustEntity{ID: mil.ID}),
				engine.Choice{
					ID: "threat", Label: "Place 2 threat on the main scheme", Kind: engine.ChoiceLabel,
				}.Msgs(engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2}),
			)}}
		},
	})

	// 16112 Pincer Maneuver: Milano removal.
	engine.RegisterBehavior("16112", milanoSchemeBehavior(3))

	// 16113 Superior Tactics: Power Stone locks to Ronan.
	engine.RegisterBehavior("16113", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil && v.Code[:5] == "16103" {
					if ps := findPowerStone(g); ps != nil {
						ps.Target = id
						ps.Locked = true
						g.LogMajorf("The Power Stone cannot be unattached from Ronan")
					}
					break
				}
			}
			return nil
		},
	})

	// 16114 Single-Minded Fury: Ronan attacks the Power Stone holder.
	engine.RegisterBehavior("16114", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			holder := powerStoneHolder(g)
			if holder == "" {
				if c, ok := g.DrawEncounter(); ok {
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
				return nil
			}
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil && v.Code[:5] == "16103" {
					return []engine.Message{engine.VillainActivates{VillainID: id, Player: holder}}
				}
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil && v.Code[:5] == "16103" {
					if ps := findPowerStone(g); ps != nil {
						ps.Target = id
						g.Logf("The Power Stone flies back to Ronan")
					}
					break
				}
			}
			return nil
		},
	})

	// 16115 Kree Physiology: tough Ronan (or 1 damage).
	engine.RegisterBehavior("16115", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil {
					if v.Tough {
						return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 1, Source: t.ID}}
					}
					return []engine.Message{engine.ToughEntity{Target: id}}
				}
			}
			return nil
		},
	})

	// 16116 "You Stand Accused!": +1 scheme/attack from the form.
	engine.RegisterBehavior("16116", &engine.Behavior{
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
	})
}

// fetchSchemeStage builds the VillainStage hook that reveals a side
// scheme from the encounter deck or discard pile.
func fetchSchemeStage(code string) func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
	return func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
		for i, c := range g.EncounterDeck {
			if c.Code[:5] == code {
				g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
				g.ShuffleEncounterDeck()
				return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
			}
		}
		for i, c := range g.EncounterDiscard {
			if c.Code[:5] == code {
				g.EncounterDiscard = append(g.EncounterDiscard[:i], g.EncounterDiscard[i+1:]...)
				return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
			}
		}
		return nil
	}
}

// findPowerStone locates the Power Stone attachment (16149).
func findPowerStone(g *engine.Game) *engine.Attachment {
	for _, a := range g.Attachments {
		if a != nil && a.Code[:5] == "16149" {
			return a
		}
	}
	return nil
}

// powerStoneHolder returns the player currently attached to the Power
// Stone ("" when it sits on a villain).
func powerStoneHolder(g *engine.Game) engine.PlayerID {
	ps := findPowerStone(g)
	if ps == nil || !ps.Target.Is(engine.KindPlayer) {
		return ""
	}
	return ps.Target
}
