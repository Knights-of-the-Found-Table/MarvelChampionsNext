package blackpanther

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() { registerBPComplete() }

func bpInt(v *int, fallback int) int {
	if v == nil {
		return fallback
	}
	return *v
}

func bpEnemies(g *engine.Game) []engine.EntityID {
	var out []engine.EntityID
	for id := range g.Villains {
		out = append(out, id)
	}
	for id := range g.Minions {
		out = append(out, id)
	}
	return out
}

// joystick finds the Joystick minion.
func joystick(g *engine.Game) *engine.Minion {
	for _, mn := range g.Minions {
		if mn != nil && mn.Code == "51039" {
			return mn
		}
	}
	return nil
}

func registerBPComplete() {
	// 51006 Vibranium resource (textless).
	engine.RegisterBehavior("51006", &engine.Behavior{})

	// 51014 Manifold: fetch a player side scheme.
	engine.RegisterBehavior("51014", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			for i, c := range p.Deck {
				if c.Def().Type == "player_side_scheme" {
					card := c
					p.Deck = append(p.Deck[:i:i], p.Deck[i+1:]...)
					p.Hand = append(p.Hand, card)
					return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
				}
			}
			return nil
		},
	})
	// 51015 Infiltration: mill 1-5, threat per card (fixed at 3).
	engine.RegisterBehavior("51015", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var out []engine.Message
			n := 3
			for i := 0; i < n; i++ {
				if c, ok := g.DrawEncounter(); ok {
					if c.Def().Type == "minion" {
						if p := g.Player(e.EOwner()); p != nil {
							def := c.Def()
							mn := &engine.Minion{ID: g.NextEntityID(engine.KindMinion), Code: c.Code,
								MaxHP: bpInt(def.HP, 1), AttackVal: bpInt(def.Attack, 0), SchemeVal: bpInt(def.Scheme, 0),
								EngagedWith: p.ID}
							g.Minions[mn.ID] = mn
							out = append(out, engine.MinionEntersPlay{MinionID: mn.ID, Player: p.ID})
						}
					} else {
						g.EncounterDiscard = append(g.EncounterDiscard, c)
					}
				}
			}
			if g.MainScheme != nil {
				out = append(out, engine.ThwartScheme{Scheme: g.MainScheme.ID, N: n, Source: e.EID()})
			}
			return out
		},
	})
	// 51016 Going Undercover.
	engine.RegisterBehavior("51016", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			g.TLogf("c.goingUndercoverMapsTheEnemyOperation")
			return nil
		},
	})
	// 51017 Show of Empathy: threat-to-minion conversion approximated.
	engine.RegisterBehavior("51017", &engine.Behavior{})
	// 51018 The Raft: bank defeated minions.
	engine.RegisterBehavior("51018", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionDefeated)
			if !ok {
				return nil
			}
			s := g.Supports[e.EID()]
			if s == nil {
				return nil
			}
			if mn := g.Minions[m.MinionID]; mn != nil {
				s.AttachedCards = append(s.AttachedCards, engine.Card{ID: g.NextCardID(), Code: mn.Code})
				s.Counters = len(s.AttachedCards)
			}
			if g.MainScheme != nil {
				return []engine.Message{engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 1, Source: e.EID()}}
			}
			return nil
		},
	})
	// 51019 Invisibility Gear: attack-to-scheme swap not modeled.
	engine.RegisterBehavior("51019", &engine.Behavior{})
	// 51020 Sonic Rifle.
	engine.RegisterBehavior("51020", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			g.Upgrades[e.EID()].Counters = 2
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			u := g.Upgrades[e.EID()]
			if u == nil || u.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.sonicRifleConfuseAnEnemy3DamageIfAlreadyConfused"), Type: engine.AbilityAction,
				HeroOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					p := g.Player(u.Owner)
					if p == nil {
						return nil
					}
					u.Counters--
					var out []engine.Message
					for _, id := range bpEnemies(g) {
						if mn := g.Minions[id]; mn != nil && mn.Confused {
							out = append(out, engine.DamageEntity{Target: id, Damage: 3, Source: self})
						} else if g.Villains[id] != nil && g.Villains[id].Confused {
							out = append(out, engine.DamageEntity{Target: id, Damage: 3, Source: self})
						} else {
							out = append(out, engine.ConfuseEntity{Target: id})
						}
						break
					}
					return out
				},
			}}
		},
	})
	// 51021 Sting Operation.
	engine.RegisterBehavior("51021", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeThreat)
			if !ok {
				return nil
			}
			mn := g.Minions[m.Source]
			u := g.Upgrades[e.EID()]
			if mn == nil || u == nil || mn.EDef().HasTrait("elite") {
				return nil
			}
			g.Delete(u.ID)
			p := g.Player(u.Owner)
			p.Discard = append(p.Discard, engine.Card{ID: g.NextCardID(), Code: u.Code, Owner: p.ID})
			g.Delete(mn.ID)
			g.TLogf("c.stingOperationTakesDown", mn)
			return nil
		},
	})
	// 51022-51024 Dora Milaje: cross-ally Specials.
	dora := func(code, label string, exec func(g *engine.Game, self engine.EntityID) []engine.Message) {
		engine.RegisterBehavior(code, &engine.Behavior{
			Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
				return []engine.Ability{{
					Label: engine.S(label), Type: engine.AbilityAction, HeroOnly: true, Exhaust: true,
					Execute: exec,
				}}
			},
		})
	}
	dora("51022", "Aneka — Special: remove 1 threat from a scheme", func(g *engine.Game, self engine.EntityID) []engine.Message {
		if g.MainScheme != nil {
			return []engine.Message{engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 1, Source: self}}
		}
		return nil
	})
	dora("51023", "Ayo — Special: deal 1 damage to an enemy", func(g *engine.Game, self engine.EntityID) []engine.Message {
		for _, id := range bpEnemies(g) {
			return []engine.Message{engine.DamageEntity{Target: id, Damage: 1, Source: self}}
		}
		return nil
	})
	dora("51024", "Okoye — Special: a Wakanda character gets +1 THW/+1 ATK", func(g *engine.Game, self engine.EntityID) []engine.Message {
		a := g.Allies[self]
		if a == nil {
			return nil
		}
		return []engine.Message{engine.AllyStatBonus{Ally: self, ATK: 1, THW: 1}}
	})
	// 51025 Heart of the Panther.
	engine.RegisterBehavior("51025", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			for i, c := range p.Deck {
				if c.Def().Type == "upgrade" && c.Def().HasTrait("wakanda") {
					card := c
					p.Deck = append(p.Deck[:i:i], p.Deck[i+1:]...)
					return append([]engine.Message{engine.UpgradeEnterPlay{Player: p.ID, Card: card}},
						engine.ShufflePlayerDeck{Player: p.ID})
				}
			}
			return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
		},
	})
	// 51026 Build Support.
	engine.RegisterBehavior("51026", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			var out []engine.Message
			for _, p := range g.Players {
				for i, c := range p.Deck {
					if c.Def().Type == "support" && cardCostLE(c.Def(), 3) {
						card := c
						p.Deck = append(p.Deck[:i:i], p.Deck[i+1:]...)
						p.Hand = append(p.Hand, card)
						out = append(out, engine.ShufflePlayerDeck{Player: p.ID})
						break
					}
				}
			}
			return out
		},
	})
	// 51027-51029 basic resources.
	engine.RegisterBehavior("51027", &engine.Behavior{})
	engine.RegisterBehavior("51028", &engine.Behavior{})
	engine.RegisterBehavior("51029", &engine.Behavior{})
	// 51030 Dora Milaje support.
	engine.RegisterBehavior("51030", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.doraMilajeResolveAnAllySpecialAndHealThem"), Type: engine.AbilityAction, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.Owner)
					if p == nil {
						return nil
					}
					for _, aid := range p.Allies {
						if a := g.Allies[aid]; a != nil && a.EDef().HasTrait("dora milaje") {
							return []engine.Message{engine.HealEntity{Target: aid, N: 1}}
						}
					}
					return nil
				},
			}}
		},
	})
	// 51036 Redemption: minion-to-ally conversion.
	engine.RegisterBehavior("51036", &engine.Behavior{})
	// 51037 White Wolf.
	engine.RegisterBehavior("51037", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyAttackWindow)
			if !ok || m.Ally != e.EID() || g.MainScheme == nil {
				return nil
			}
			return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: e.EID()}}
		},
	})
	// 51038 Target Spotter.
	engine.RegisterBehavior("51038", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			g.Supports[e.EID()].Counters = 2
			return nil
		},
	})

	// --- Extreme Risk nemesis set (51039-51042) ---
	engine.RegisterBehavior("51039", &engine.Behavior{})
	engine.RegisterBehavior("51040", &engine.Behavior{})
	engine.RegisterBehavior("51041", &engine.Behavior{})
	engine.RegisterBehavior("51042", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			mn := joystick(g)
			if mn == nil {
				for i, c := range g.EncounterDeck {
					if c.Code == "51039" {
						g.EncounterDeck = append(g.EncounterDeck[:i:i], g.EncounterDeck[i+1:]...)
						def := c.Def()
						mn = &engine.Minion{ID: g.NextEntityID(engine.KindMinion), Code: c.Code,
							MaxHP: bpInt(def.HP, 1), AttackVal: bpInt(def.Attack, 0), SchemeVal: bpInt(def.Scheme, 0),
							EngagedWith: p.ID}
						g.Minions[mn.ID] = mn
						break
					}
				}
			}
			if mn == nil {
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			return []engine.Message{engine.MinionActivates{MinionID: mn.ID, Player: p.ID}}
		},
	})
}

// cardCostLE reports a cost at most n.
func cardCostLE(def *data.CardDef, n int) bool {
	return def.Cost != nil && *def.Cost <= n
}
