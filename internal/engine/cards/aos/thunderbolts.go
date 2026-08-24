package aos

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() { registerThunderbolts() }

// thunderboltVictories counts Thunderbolt cards in the victory display.
func thunderboltVictories(g *engine.Game) int {
	n := 0
	for _, c := range g.VictoryDisplay {
		if c.Def().HasTrait("thunderbolt") {
			n++
		}
	}
	return n
}

// citizenV finds Citizen V.
func citizenV(g *engine.Game) *engine.Villain {
	for _, v := range g.Villains {
		if v != nil && data.BaseCode(v.Code) == "50129" {
			return v
		}
	}
	return nil
}

// thunderboltMinion reveals a Thunderbolt minion from deck/discard and
// engages it with the player (the "Find X and reveal it" rider).
func thunderboltMinion(g *engine.Game, code string, p *engine.Player) []engine.Message {
	for i, c := range g.EncounterDeck {
		if c.Code == code {
			g.EncounterDeck = append(g.EncounterDeck[:i:i], g.EncounterDeck[i+1:]...)
			return spawnThunderbolt(g, code, p)
		}
	}
	for i, c := range g.EncounterDiscard {
		if c.Code == code {
			g.EncounterDiscard = append(g.EncounterDiscard[:i:i], g.EncounterDiscard[i+1:]...)
			return spawnThunderbolt(g, code, p)
		}
	}
	return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
}

func spawnThunderbolt(g *engine.Game, code string, p *engine.Player) []engine.Message {
	card := engine.Card{Code: code}
	def := card.Def()
	mn := &engine.Minion{
		ID: g.NextEntityID(engine.KindMinion), Code: code,
		MaxHP:     intValue(def.HP, 1),
		AttackVal: intValue(def.Attack, 0), SchemeVal: intValue(def.Scheme, 0),
		EngagedWith: p.ID,
	}
	g.Minions[mn.ID] = mn
	return []engine.Message{engine.MinionEntersPlay{MinionID: mn.ID, Player: p.ID}}
}

func registerThunderbolts() {
	// 50129a Citizen V: only defeatable once enough Thunderbolts have
	// been swept into the victory display.
	engine.RegisterBehavior("50129", &engine.Behavior{
		VillainDamageable: func(g *engine.Game, v *engine.Villain, damage int) bool {
			if v.Damage+damage < v.MaxHP {
				return true
			}
			if thunderboltVictories(g) >= len(g.Players) {
				return true
			}
			g.Logf("Citizen V cannot be defeated yet (%d/%d Thunderbolts in the victory display)",
				thunderboltVictories(g), len(g.Players))
			v.Damage = v.MaxHP - 1
			return false
		},
	})

	// 50130a Apprehending Rogue Agents main scheme.
	engine.RegisterBehavior("50130", &engine.Behavior{})

	// 50131a Justice, Like Lightning: a marker for the set-aside
	// Thunderbolts (their reveal is handled by scenario setup).
	engine.RegisterBehavior("50131", &engine.Behavior{})

	// 50132 Citizen V's Sword: attach + Citizen V activates.
	engine.RegisterBehavior("50132", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if v := g.Villains[target]; v != nil {
				v.AttackVal++
				g.Logf("Citizen V draws his sword (+1 ATK)")
			}
			return nil
		},
	})

	// 50133 Jolt: parley counters buy her removal; defeat costs threat.
	engine.RegisterBehavior("50133", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionDefeated)
			if !ok || m.MinionID != e.EID() || g.MainScheme == nil {
				return nil
			}
			return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 3, Source: m.MinionID}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Jolt — exhaust your hero to add a parley counter (3+ removes her)", Type: engine.AbilityAction,
				HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					mn := g.Minions[self]
					p := g.Player(mn.EngagedWith)
					if mn == nil || p == nil || p.Exhausted {
						return nil
					}
					mn.Counters++
					g.Logf("Jolt gains a parley counter (%d)", mn.Counters)
					if mn.Counters >= 3 {
						g.Delete(mn.ID)
						g.Logf("Jolt stands down and leaves the fight")
					}
					return []engine.Message{engine.ExhaustEntity{ID: p.ID}}
				},
			}}
		},
	})

	// 50134 Innocent Bystanders obligation: 4 bystander counters; each
	// clash costs a resource or threat.
	engine.RegisterBehavior("50134", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			g.Logf("Innocent Bystanders: 4 bystander counters")
			return []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
		},
	})

	// 50135/50136 minion-rotation schemes (clockwise engagement not
	// modeled beyond a log).
	engine.RegisterBehavior("50135", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			g.Logf("The Coming Storm shifts the battlefield")
			return nil
		},
	})
	engine.RegisterBehavior("50136", &engine.Behavior{})

	// 50137 Down but Not Out: a Thunderbolt returns from the victory
	// display wounded to 5 remaining hit points.
	engine.RegisterBehavior("50137", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			for _, c := range g.VictoryDisplay {
				if c.Def().Type == "minion" && c.Def().HasTrait("thunderbolt") {
					out := spawnThunderbolt(g, c.Code, p)
					for _, mn := range g.Minions {
						if mn != nil && mn.Code == c.Code && mn.EngagedWith == p.ID {
							if rem := mn.MaxHP - 5; rem > mn.Damage {
								mn.Damage = rem
							}
							g.Logf("%s returns from the victory display", mn.EDef().Name)
						}
					}
					return out
				}
			}
			return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
		},
	})

	// 50138 Tap In: a Thunderbolt activates against you (approximated:
	// the villain activates instead).
	engine.RegisterBehavior("50138", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var least engine.EntityID
			leastDmg := 1 << 30
			for _, mn := range g.Minions {
				if mn != nil && mn.EDef().HasTrait("thunderbolt") && mn.EngagedWith != p.ID && mn.Damage < leastDmg {
					least, leastDmg = mn.ID, mn.Damage
				}
			}
			if least != "" {
				return []engine.Message{engine.EngageMinion{MinionID: least, Player: p.ID},
					engine.MinionActivates{MinionID: least, Player: p.ID}}
			}
			if v := citizenV(g); v != nil {
				return []engine.Message{engine.VillainActivates{VillainID: v.ID, Player: p.ID}}
			}
			return nil
		},
	})

	// --- Gravitational Pull (Moonstone) ---
	// 50139 Moonstone: tough after activating.
	engine.RegisterBehavior("50139", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			g.Logf("Moonstone hardens into a tough card")
			return []engine.Message{engine.ToughEntity{Target: e.EID()}}
		},
	})
	// 50140 Rule the Skies: Aerial characters get +1 ATK.
	engine.RegisterBehavior("50140", &engine.Behavior{})
	// 50141 Gravitational Pull: find Moonstone, she activates.
	engine.RegisterBehavior("50141", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			return thunderboltMinion(g, "50139", p)
		},
	})
	// 50142 Psychological Manipulation.
	engine.RegisterBehavior("50142", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if !p.IsHero() {
				if len(p.Allies) > 0 {
					return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Allies[0]}}
				}
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 2, Source: t.ID}}
		},
	})

	// --- Hard Sound (Songbird) ---
	// 50143 Songbird.
	engine.RegisterBehavior("50143", &engine.Behavior{})
	// 50144 Solid Sound Constructs.
	engine.RegisterBehavior("50144", &engine.Behavior{})
	// 50145 Hard Sound Bindings.
	engine.RegisterBehavior("50145", &engine.Behavior{})
	// 50146 Sonic Bubble: damage to enemies becomes threat removal here.
	engine.RegisterBehavior("50146", &engine.Behavior{})
	// 50147 Hard Sound.
	engine.RegisterBehavior("50147", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			return thunderboltMinion(g, "50143", p)
		},
	})

	// --- Pale Little Spider (Black Widow) ---
	// 50148 Black Widow minion.
	engine.RegisterBehavior("50148", &engine.Behavior{})
	// 50149 Handspring.
	engine.RegisterBehavior("50149", &engine.Behavior{})
	// 50150 Pride of the Red Room.
	engine.RegisterBehavior("50150", &engine.Behavior{})
	// 50151 Pale Little Spider.
	engine.RegisterBehavior("50151", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			return thunderboltMinion(g, "50148", p)
		},
	})

	// --- Power of the Atom (Radioactive Man) ---
	// 50152 Radioactive Man.
	engine.RegisterBehavior("50152", &engine.Behavior{})
	// 50153 Radiation Exposure.
	engine.RegisterBehavior("50153", &engine.Behavior{})
	// 50154 Runaway Nuclear Reaction: damage he takes becomes threat.
	engine.RegisterBehavior("50154", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.DamageEntity)
			if !ok || g.Minions[m.Target] == nil || g.Minions[m.Target].Code != "50152" {
				return nil
			}
			s := g.SideSchemes[e.EID()]
			if s == nil || m.Damage <= 0 {
				return nil
			}
			return []engine.Message{engine.SchemeThreat{Scheme: s.ID, N: m.Damage, Source: s.ID}}
		},
	})
	// 50155 Power of the Atom.
	engine.RegisterBehavior("50155", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			out := thunderboltMinion(g, "50152", p)
			return append(out, engine.IndirectDamage{Player: p.ID, N: 2})
		},
	})

	// --- Supersonic (MACH-IV) ---
	// 50156 MACH-IV.
	engine.RegisterBehavior("50156", &engine.Behavior{})
	// 50157 Blasters.
	engine.RegisterBehavior("50157", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if v := g.Villains[target]; v != nil {
				v.AttackVal++
			} else if mn := g.Minions[target]; mn != nil {
				mn.AttackVal++
			}
			return nil
		},
	})
	// 50158 Heat-Seeking Missiles: 4 counters, 2 indirect per attack.
	engine.RegisterBehavior("50158", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			t.Counters = 4
			g.Logf("Heat-Seeking Missiles arm with 4 missile counters")
			return nil
		},
	})
	// 50159 Aerial Dogfight: Hinder handled by the engine.
	engine.RegisterBehavior("50159", &engine.Behavior{})
	// 50160 Supersonic.
	engine.RegisterBehavior("50160", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			return thunderboltMinion(g, "50156", p)
		},
	})

	// --- The Leaper (Batroc) ---
	// 50161 Batroc the Leaper.
	engine.RegisterBehavior("50161", &engine.Behavior{})
	// 50162 Coup de Foudre.
	engine.RegisterBehavior("50162", &engine.Behavior{})
	// 50163 Batroc the Leaper treachery.
	engine.RegisterBehavior("50163", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			return thunderboltMinion(g, "50161", p)
		},
	})
	// 50164 Parcours du Combattant.
	engine.RegisterBehavior("50164", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			return []engine.Message{engine.RevealNextEncounter{Player: p.ID}, engine.RevealNextEncounter{Player: p.ID}}
		},
	})
}
