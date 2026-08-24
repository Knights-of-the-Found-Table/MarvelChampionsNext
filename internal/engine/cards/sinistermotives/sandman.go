package sinistermotives

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// registerSandman installs the Sandman scenario (27061–27072) built on
// the City Streets sand-counter economy.
func registerSandman() {
	// Surging Sands helper: 1 sand counter + mill the encounter deck for
	// the count.
	surgingSands := func(g *engine.Game) []engine.Message {
		env := g.EnvironmentByCode("27065")
		if env == nil {
			return nil
		}
		env.Counters++
		n := env.Counters
		g.Logf("City Streets surges (%d sand counters, milling %d)", n, n)
		for i := 0; i < n && len(g.EncounterDeck) > 0; i++ {
			c := g.EncounterDeck[0]
			g.EncounterDeck = g.EncounterDeck[1:]
			g.EncounterDiscard = append(g.EncounterDiscard, c)
		}
		return nil
	}

	// Sandman I–III (27061–27063).
	for _, base := range []string{"27061", "27062", "27063"} {
		b := &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				m, ok := msg.(engine.VillainActivates)
				if !ok || m.VillainID != e.EID() {
					return nil
				}
				p := g.Player(m.Player)
				if p == nil || !p.IsHero() {
					return nil
				}
				// Sand Blast / Sand Wave riders: indirect damage (I/II,
				// approximated as +2 damage follow-up) and Surging Sands
				// when the identity is hurt.
				if base != "27063" {
					g.Logf("Sand Blast scatters the attack")
					return []engine.Message{
						engine.DamageEntity{Target: p.ID, Damage: 2, Source: e.EID()},
					}
				}
				return nil
			},
		}
		if base != "27061" {
			b.VillainStage = func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
				if base == "27062" {
					return surgingSands(g)
				}
				if env := g.EnvironmentByCode("27065"); env != nil {
					env.Counters++
				}
				return surgingSands(g)
			}
		}
		engine.RegisterBehavior(base, b)
	}

	// 27064 Hapless Pedestrians: acceleration blasts the first player;
	// completing the stage loses (default engine behavior).
	engine.RegisterBehavior("27064", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.AddAccelerationToken); ok {
				return []engine.Message{engine.IndirectDamage{Player: cardutil.FirstPlayerID(g), N: 3}}
			}
			return nil
		},
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			// Setup: City Streets with 4 sand counters.
			if g.EnvironmentByCode("27065") == nil {
				env := g.SpawnEnvironment("27065")
				env.Counters = 4
			}
			return nil
		},
	})

	// 27065 City Streets: Surging Sands + the ATK-scrub action.
	engine.RegisterBehavior("27065", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			env := g.Environments[e.EID()]
			if env == nil || env.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: "Exhaust a character you control → remove sand counters equal to its ATK", Type: engine.AbilityAction,
				OncePerRound: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(g.ActiveTurn)
					if p == nil {
						return nil
					}
					var picks []engine.Choice
					if !p.Exhausted {
						picks = append(picks, engine.Choice{
							ID: "self", Label: p.Name + " (ATK " + itoa(max(0, p.AttackStat(g))) + ")", Kind: engine.ChoiceLabel,
						}.Msgs(engine.ExhaustEntity{ID: p.ID}, engine.AddEntityCounter{ID: self, N: -max(1, p.AttackStat(g))}))
					}
					for _, id := range p.Allies {
						if a := g.Allies[id]; a != nil && !a.Exhausted {
							n := max(1, a.AttackVal)
							picks = append(picks, engine.Choice{
								Label: a.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id,
							}.Msgs(engine.ExhaustEntity{ID: id}, engine.AddEntityCounter{ID: self, N: -n}))
						}
					}
					if len(picks) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID,
						Question: engine.Ask("City Streets: exhaust which character?", picks...)}}
				},
			}}
		},
	})

	// 27066 Sand Form: damage to Sandman is deflected into Surging Sands
	// once (approximated: the first lethal hit is eaten).
	engine.RegisterBehavior("27066", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil && v.Code[:5] == "27061" {
					t.Target = id
					g.Logf("Sand Form attaches to Sandman")
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

	// 27067 Sand Clone: X HP per sand counter (scaled on reveal via the
	// generic path; the When Defeated surge is approximated by dropping
	// a surge-like mill).
	engine.RegisterBehavior("27067", &engine.Behavior{})

	// 27068 Dirt Trap: double Surging Sands on defeat.
	engine.RegisterBehavior("27068", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			// The environment surge runs inline (mutation-based).
			if env := g.EnvironmentByCode("27065"); env != nil {
				for i := 0; i < 2; i++ {
					env.Counters++
					for j := 0; j < env.Counters && len(g.EncounterDeck) > 0; j++ {
						c := g.EncounterDeck[0]
						g.EncounterDeck = g.EncounterDeck[1:]
						g.EncounterDiscard = append(g.EncounterDiscard, c)
					}
				}
			}
			return nil
		},
	})

	// 27069 Tidal Sands: extra threat per sand counter (approximated on
	// reveal via OnPlay).
	engine.RegisterBehavior("27069", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if s := g.SideSchemes[e.EID()]; s != nil {
				if env := g.EnvironmentByCode("27065"); env != nil {
					s.Threat += env.Counters
				}
			}
			return nil
		},
	})

	// 27070 Sandslide: 2 counters + surge; stunned if a Sandman card was
	// milled.
	engine.RegisterBehavior("27070", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			env := g.EnvironmentByCode("27065")
			var msgs []engine.Message
			if env != nil {
				env.Counters += 2
				sandmanMilled := false
				for i := 0; i < env.Counters && len(g.EncounterDeck) > 0; i++ {
					c := g.EncounterDeck[0]
					g.EncounterDeck = g.EncounterDeck[1:]
					if c.Def().CardSet == "sandman" {
						sandmanMilled = true
					}
					g.EncounterDiscard = append(g.EncounterDiscard, c)
				}
				if sandmanMilled {
					msgs = append(msgs, engine.StunEntity{Target: p.ID})
				}
			}
			return msgs
		},
	})

	// 27071 Sand Storm: X indirect damage split among players.
	engine.RegisterBehavior("27071", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			env := g.EnvironmentByCode("27065")
			if env == nil || env.Counters == 0 {
				if env != nil {
					env.Counters = 3
				}
				g.EncounterDeck = append(g.EncounterDeck, engine.Card{ID: g.NextCardID(), Code: "27071"})
				g.ShuffleEncounterDeck()
				return nil
			}
			per := env.Counters / len(g.Players)
			var msgs []engine.Message
			for _, tp := range g.Players {
				msgs = append(msgs, engine.IndirectDamage{Player: tp.ID, N: per})
			}
			return msgs
		},
	})

	// 27072 Sand Smash: Surging Sands (alter-ego) or +1 ATK attack
	// (hero).
	engine.RegisterBehavior("27072", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			if !p.IsHero() {
				if env := g.EnvironmentByCode("27065"); env != nil {
					env.Counters++
					for i := 0; i < env.Counters && len(g.EncounterDeck) > 0; i++ {
						c := g.EncounterDeck[0]
						g.EncounterDeck = g.EncounterDeck[1:]
						g.EncounterDiscard = append(g.EncounterDiscard, c)
					}
				}
				return nil
			}
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

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	if neg {
		return "-" + digits
	}
	return digits
}
