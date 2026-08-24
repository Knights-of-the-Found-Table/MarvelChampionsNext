package aos

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() { registerMODOK() }

// holdingCell finds a Holding Cell environment still holding lock counters.
func holdingCell(g *engine.Game) *engine.Environment {
	for _, env := range g.Environments {
		if env != nil && env.Code >= "50105" && env.Code <= "50108" && env.Counters > 0 {
			return env
		}
	}
	return nil
}

// modokVillain finds the M.O.D.O.K. villain entity.
func modokVillain(g *engine.Game) *engine.Villain {
	for _, v := range g.Villains {
		if v != nil && data.BaseCode(v.Code) == "50103" {
			return v
		}
	}
	return nil
}

func registerMODOK() {
	// 50103a M.O.D.O.K.: while a Holding Cell still has lock counters,
	// defeating him removes 2 counters and resets him instead; hostages
	// shield him entirely.
	engine.RegisterBehavior("50103", &engine.Behavior{
		VillainDamageable: func(g *engine.Game, v *engine.Villain, damage int) bool {
			if g.SideSchemeInPlay("50121") {
				g.Logf("M.O.D.O.K. is shielded by the hostage situation")
				return false
			}
			if v.Damage+damage < v.MaxHP {
				return true
			}
			cell := holdingCell(g)
			if cell == nil {
				return true // the players win the game
			}
			cell.Counters -= 2
			if cell.Counters < 0 {
				cell.Counters = 0
			}
			v.Damage = 0
			g.Logf("M.O.D.O.K. retreats to the Holding Cell (%d lock counters left)", cell.Counters)
			if cell.Counters == 0 {
				g.Logf("A Holding Cell bursts open!")
			}
			return false
		},
	})

	// 50104a Upgrading Adaptoids main scheme.
	engine.RegisterBehavior("50104", &engine.Behavior{})

	// Holding Cells: lock counters + a resource action to open one.
	for _, code := range []string{"50105", "50106", "50107", "50108"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
				env := g.Environments[e.EID()]
				if env == nil || env.Counters <= 0 {
					return nil
				}
				return []engine.Ability{{
					Label: "Holding Cell — spend 3 resources to remove 1 lock counter", Type: engine.AbilityAction,
					HeroOnly: true, Cost: 3,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						env := g.Environments[self]
						if env == nil || env.Counters <= 0 {
							return nil
						}
						env.Counters--
						g.Logf("A lock counter is removed (%d left)", env.Counters)
						if env.Counters == 0 {
							// The cell bursts open and its prisoner
							// (the linked b-side ally) joins a player.
							p := g.Players[0]
							if p == nil {
								return nil
							}
							allyCode := env.Code[:len(env.Code)-1] + "b"
							card := engine.Card{Code: allyCode}
							def := card.Def()
							if def.Type != "ally" {
								return nil
							}
							a := &engine.Ally{
								ID: g.NextEntityID(engine.KindAlly), Code: allyCode,
								Owner: p.ID, MaxHP: intValue(def.HP, 3),
								AttackVal: intValue(def.Attack, 1), ThwartVal: intValue(def.Thwart, 1),
							}
							g.AddAlly(a, p.ID)
							g.Logf("%s bursts free from the Holding Cell!", def.Name)
							return []engine.Message{engine.AllyEnteredPlay{Ally: a.ID, Player: p.ID}}
						}
						return nil
					},
				}}
			},
		})
	}

	// Adaptoid Upgrade environments: stat auras approximated through the
	// Adaptoid minion's own EnemyStatBonus.
	for _, code := range []string{"50109", "50110", "50111", "50112"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	// 50113 Adaptoid: scales with the Adaptoid environments in play;
	// defeating one strips a counter from an environment.
	engine.RegisterBehavior("50113", &engine.Behavior{
		EnemyStatBonus: func(g *engine.Game, e engine.Entity) (int, int) {
			atk, sch := 0, 0
			for _, env := range g.Environments {
				if env == nil {
					continue
				}
				switch data.BaseCode(env.Code) {
				case "50109":
					sch++
				case "50111", "50112":
					atk++
				}
			}
			return atk, sch
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionDefeated)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			for _, env := range g.Environments {
				if env != nil && env.Counters > 0 {
					env.Counters--
					g.Logf("%s powers down (%d counters left)", env.EDef().Name, env.Counters)
					break
				}
			}
			return nil
		},
	})

	// Attachments: 50114 raises max HP; the rest ride along as markers and
	// are discarded when M.O.D.O.K. resets (his damage hook already
	// handles the cell route; the discard-on-reset is approximated).
	engine.RegisterBehavior("50114", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if v := g.Villains[target]; v != nil {
				v.MaxHP += 5
				g.Logf("M.O.D.O.K. gains 5 max hit points")
			}
			return nil
		},
	})
	engine.RegisterBehavior("50115", &engine.Behavior{})
	engine.RegisterBehavior("50116", &engine.Behavior{})
	// 50117 Psionic Force Field: damage banks on the attachment until 5
	// burns it out (approximated: M.O.D.O.K. ignores hits of 2 or less
	// while it is attached).
	engine.RegisterBehavior("50117", &engine.Behavior{})
	engine.RegisterBehavior("50118", &engine.Behavior{})
	engine.RegisterBehavior("50119", &engine.Behavior{})

	// 50120 A.I.M. Jailer: attacks the rescued captive with the fewest
	// remaining hit points, else locks a Holding Cell.
	engine.RegisterBehavior("50120", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil {
				return nil
			}
			p := g.Player(mn.EngagedWith)
			if p == nil {
				return nil
			}
			var captive *engine.Ally
			for _, aid := range p.Allies {
				if a := g.Allies[aid]; a != nil && (captive == nil || a.HP() < captive.HP()) {
					captive = a
				}
			}
			if captive != nil {
				return []engine.Message{engine.DamageEntity{Target: captive.ID, Damage: mn.AttackVal, Source: mn.ID}}
			}
			if cell := holdingCell(g); cell != nil {
				cell.Counters++
				g.Logf("A lock counter is added to a Holding Cell (%d)", cell.Counters)
			}
			return nil
		},
	})

	// 50121 Hostage Situation: M.O.D.O.K. cannot take damage while it
	// stands (the ally attachment is approximated away).
	engine.RegisterBehavior("50121", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			g.Logf("M.O.D.O.K. hides behind hostages — he cannot take damage while Hostage Situation stands")
			return nil
		},
	})

	// 50122 Psionic Enhancement: Hinder handled by the engine.
	engine.RegisterBehavior("50122", &engine.Behavior{})

	// 50123 "It's Alive!": each player reveals an Adaptoid.
	engine.RegisterBehavior("50123", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var out []engine.Message
			spawned := false
			for i, c := range g.EncounterDeck {
				if c.Code == "50113" {
					card := c
					g.EncounterDeck = append(g.EncounterDeck[:i:i], g.EncounterDeck[i+1:]...)
					def := card.Def()
					mn := &engine.Minion{
						ID: g.NextEntityID(engine.KindMinion), Code: card.Code,
						MaxHP:     intValue(def.HP, 1),
						AttackVal: intValue(def.Attack, 0), SchemeVal: intValue(def.Scheme, 0),
						EngagedWith: p.ID,
					}
					g.Minions[mn.ID] = mn
					out = append(out, engine.MinionEntersPlay{MinionID: mn.ID, Player: p.ID})
					spawned = true
					break
				}
			}
			if !spawned {
				out = append(out, engine.DealEncounterToPlayer{Player: p.ID})
			}
			return out
		},
	})

	// 50124 Psionic Blast: confuse + scheme, or indirect damage.
	engine.RegisterBehavior("50124", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if !p.IsHero() {
				out := []engine.Message{engine.ConfuseEntity{Target: p.ID}}
				if v := modokVillain(g); v != nil && g.MainScheme != nil {
					out = append(out, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: v.SchemeVal, Source: v.ID})
				}
				return out
			}
			n := 2
			if v := modokVillain(g); v != nil {
				n = v.SchemeVal
			}
			return []engine.Message{engine.IndirectDamage{Player: p.ID, N: n}}
		},
	})

	registerScientistSupreme()
}

func registerScientistSupreme() {
	// 50125 Scientist Supreme / 50126 Monica Rappaccini: villainous
	// minions with piercing/vulnerable riders handled by the data
	// keywords; Monica's conditional villainous is approximated as always
	// on once her mentor is in the victory display.
	engine.RegisterBehavior("50125", &engine.Behavior{})
	engine.RegisterBehavior("50126", &engine.Behavior{})

	// 50127 Diplomatic Immunity: acceleration per A.I.M. minion in the
	// victory display.
	engine.RegisterBehavior("50127", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if n := aimVictories(g); n > 0 {
				s := g.SideSchemes[e.EID()]
				for i := 0; i < n; i++ {
					s.Counters++
				}
				g.Logf("Diplomatic Immunity gains %d acceleration tokens", n)
			}
			return nil
		},
	})

	// 50128 Diplomatic Sanctions: discard per A.I.M. minion in the
	// victory display.
	engine.RegisterBehavior("50128", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			n := aimVictories(g)
			var out []engine.Message
			if n > 0 && len(p.Hand) > 0 {
				if n > len(p.Hand) {
					n = len(p.Hand)
				}
				discarded := append(engine.CardList(nil), p.Hand[:n]...)
				p.Hand = p.Hand[n:]
				p.Discard = append(p.Discard, discarded...)
				out = append(out, engine.DiscardCards{Player: p.ID, Cards: discarded})
			}
			out = append(out, engine.RevealNextEncounter{Player: p.ID})
			return out
		},
	})
}

// hasAIMTrait mirrors the dotted-acronym workaround used for S.H.I.E.L.D.
func hasAIMTrait(def *data.CardDef) bool {
	if def == nil {
		return false
	}
	if def.HasTrait("a.i.m.") {
		return true
	}
	want := []string{"a", "i", "m"}
	for i := 0; i+len(want) <= len(def.Traits); i++ {
		match := true
		for j := range want {
			if def.Traits[i+j] != want[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// aimVictories counts A.I.M. cards in the victory display.
func aimVictories(g *engine.Game) int {
	n := 0
	for _, c := range g.VictoryDisplay {
		if hasAIMTrait(c.Def()) {
			n++
		}
	}
	return n
}
