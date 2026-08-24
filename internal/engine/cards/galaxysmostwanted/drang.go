package galaxysmostwanted

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// registerDrangScenario installs the Planetary Invasion scenario content
// (16057–16069): Drang, the Badoon Ship, Drang's Spear and the Milano
// interaction side schemes.
func registerDrangScenario() {
	// 16057 Planetary Invasion treachery: dig a minion and reveal it
	// (the tough status rider is approximated away: it would need a
	// post-reveal window).
	engine.RegisterBehavior("16057", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			return discardUntilMinionEngaged(g, cardutil.FirstPlayerID(g))
		},
	})

	// Drang I–III (16058–16060): stage II fetches his spear; every scheme
	// (I/II) or activation (III) charges the Badoon Ship. The printed
	// stalwart from the spear is approximated away.
	for _, base := range []string{"16058", "16059", "16060"} {
		b := &engine.Behavior{}
		if base == "16059" {
			b.VillainStage = func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
				for i, c := range g.EncounterDeck {
					if c.Code[:5] == "16064" {
						g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
						g.ShuffleEncounterDeck()
						return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
					}
				}
				for i, c := range g.EncounterDiscard {
					if c.Code[:5] == "16064" {
						g.EncounterDiscard = append(g.EncounterDiscard[:i], g.EncounterDiscard[i+1:]...)
						return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
					}
				}
				return nil
			}
		}
		if base == "16060" {
			b.VillainStage = func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
				// Discard 4[per_hero] encounter cards; each minion joins
				// the player with the fewest minions.
				n := 4 * len(g.Players)
				var msgs []engine.Message
				for i := 0; i < n && len(g.EncounterDeck) > 0; i++ {
					c := g.EncounterDeck[0]
					g.EncounterDeck = g.EncounterDeck[1:]
					if c.Def().Type == "minion" {
						msgs = append(msgs, engine.RevealEncounterCard{Player: leastEngagedPlayer(g), Card: c})
					} else {
						g.EncounterDiscard = append(g.EncounterDiscard, c)
					}
				}
				return msgs
			}
		}
		b.React = func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			switch m := msg.(type) {
			case engine.ApplyVillainScheme:
				if m.VillainID == e.EID() && base != "16060" {
					return []engine.Message{engine.BarrageCharge{}}
				}
			case engine.VillainActivates:
				if m.VillainID == e.EID() && base == "16060" {
					return []engine.Message{engine.BarrageCharge{}}
				}
			}
			return nil
		}
		engine.RegisterBehavior(base, b)
	}

	// 16061 Terrestrial Invasion (stage 1 a-face): put the Badoon Ship and
	// the Milano into play.
	engine.RegisterBehavior("16061", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			if findEnvironment(g, "16063") == "" {
				g.SpawnEnvironment("16063")
			}
			if _, mil := findMilano(g); mil == nil {
				g.SpawnSupport("16142", cardutil.FirstPlayerID(g))
			}
			return nil
		},
	})

	// 16062 Protect the Planet (stage 2): when revealed, charge up.
	engine.RegisterBehavior("16062", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			return []engine.Message{engine.BarrageCharge{}}
		},
	})

	// 16063 Badoon Ship: the Charge Up special is engine-side
	// (BarrageCharge); the environment itself has no printed action.
	engine.RegisterBehavior("16063", &engine.Behavior{})

	// 16064 Drang's Spear: attaches to Drang; hero action buys it off.
	engine.RegisterBehavior("16064", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil && v.Code[:5] == "16058" {
					t.Target = id
					g.Logf("Drang's Spear attaches to Drang")
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
				Label: "Spend [mental][physical][physical] → discard Drang's Spear", Type: engine.AbilityAction,
				CostIcons: "mental:1 physical:2",
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.DiscardAttachmentMsg{ID: self}}
				},
			}}
		},
	})

	// 16065 Badoon Engineer: engaging or activating charges the ship.
	engine.RegisterBehavior("16065", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			switch m := msg.(type) {
			case engine.EngageMinion:
				if m.MinionID == e.EID() {
					return []engine.Message{engine.BarrageCharge{}}
				}
			case engine.MinionActivates:
				if m.MinionID == e.EID() {
					return []engine.Message{engine.BarrageCharge{}}
				}
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return []engine.Message{engine.BarrageCharge{}}
		},
	})

	// 16066–16069 side schemes: First Player Action — exhaust the Milano
	// → remove 3 threat.
	for _, code := range []string{"16066", "16067", "16068", "16069"} {
		engine.RegisterBehavior(code, milanoSchemeBehavior(3))
	}
}

// milanoSchemeBehavior builds the shared "exhaust the Milano → remove N
// threat" first-player action for side schemes.
func milanoSchemeBehavior(n int) *engine.Behavior {
	return &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			id, mil := findMilano(g)
			if mil == nil || mil.Exhausted {
				return nil
			}
			if s := g.SideSchemes[e.EID()]; s == nil || s.Threat <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: "Exhaust the Milano → remove 3 threat", Type: engine.AbilityAction,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{
						engine.ExhaustEntity{ID: id},
						engine.ThwartScheme{Scheme: self, N: n, Source: id},
					}
				},
			}}
		},
	}
}

// findMilano locates the Milano support (16142) in play.
func findMilano(g *engine.Game) (engine.EntityID, *engine.Support) {
	for id := range g.Supports {
		if s := g.Supports[id]; s != nil && s.Code[:5] == "16142" {
			return id, s
		}
	}
	return "", nil
}

// findEnvironment locates a scenario environment by base code.
func findEnvironment(g *engine.Game, base string) engine.EntityID {
	for id, env := range g.Environments {
		if env != nil && env.Code[:5] == base {
			return id
		}
	}
	return ""
}

// leastEngagedPlayer finds the player with the fewest engaged minions.
func leastEngagedPlayer(g *engine.Game) engine.PlayerID {
	best := engine.PlayerID("")
	bestN := -1
	for _, p := range g.Players {
		n := 0
		for _, m := range g.Minions {
			if m != nil && m.EngagedWith == p.ID {
				n++
			}
		}
		if bestN == -1 || n < bestN {
			best, bestN = p.ID, n
		}
	}
	return best
}
