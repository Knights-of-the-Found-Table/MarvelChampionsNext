package hercules

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() { registerHerculesComplete() }

func hercInt(v *int, fallback int) int {
	if v == nil {
		return fallback
	}
	return *v
}

func hercFindEncounter(g *engine.Game, pred func(*data.CardDef) bool) (engine.Card, bool) {
	for i, c := range g.EncounterDeck {
		if pred(c.Def()) {
			g.EncounterDeck = append(g.EncounterDeck[:i:i], g.EncounterDeck[i+1:]...)
			return c, true
		}
	}
	for i, c := range g.EncounterDiscard {
		if pred(c.Def()) {
			g.EncounterDiscard = append(g.EncounterDiscard[:i:i], g.EncounterDiscard[i+1:]...)
			return c, true
		}
	}
	return engine.Card{}, false
}

// allVersusAll finds or spawns the All Versus All side scheme.
func allVersusAll(g *engine.Game) *engine.SideScheme {
	for _, s := range g.SideSchemes {
		if s != nil && s.Code == "59041" {
			return s
		}
	}
	if card, ok := hercFindEncounter(g, func(d *data.CardDef) bool { return d.Code == "59041" }); ok {
		s := &engine.SideScheme{ID: g.NextEntityID(engine.KindSideScheme), Code: card.Code,
			Threat: 2, MaxThreat: 2 + 2*len(g.Players)}
		g.SideSchemes[s.ID] = s
		return s
	}
	return nil
}

func registerHerculesComplete() {
	// 59018 Deathcry: the self-damage discount is approximated away.
	engine.RegisterBehavior("59018", &engine.Behavior{})
	// 59019 Namora: +1 HP per other ally.
	engine.RegisterBehavior("59019", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			a := g.Allies[e.EID()]
			if a == nil {
				return nil
			}
			n := 0
			if p := g.Player(a.Owner); p != nil {
				n = len(p.Allies) - 1
			}
			if n > 0 {
				a.MaxHP += n
			}
			return nil
		},
	})
	// 59020 Thor: thwarts drag a minion in.
	engine.RegisterBehavior("59020", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyThwartWindow)
			if !ok || m.Ally != e.EID() {
				return nil
			}
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			if card, ok := hercFindEncounter(g, func(d *data.CardDef) bool { return d.Type == "minion" }); ok {
				def := card.Def()
				mn := &engine.Minion{ID: g.NextEntityID(engine.KindMinion), Code: card.Code,
					MaxHP: hercInt(def.HP, 1), AttackVal: hercInt(def.Attack, 0), SchemeVal: hercInt(def.Scheme, 0),
					EngagedWith: p.ID}
				g.Minions[mn.ID] = mn
				return []engine.Message{engine.MinionEntersPlay{MinionID: mn.ID, Player: p.ID}}
			}
			return nil
		},
	})
	// 59021 Teamwork: power-share interrupt approximated as a flat +2
	// event bonus.
	engine.RegisterBehavior("59021", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			return []engine.Message{engine.SetEventBonus{Player: p.ID, Damage: 2, Threat: 2}}
		},
	})
	// 59022 Call for Backup: each player fields an ally (approximated:
	// the owner fields one from discard).
	engine.RegisterBehavior("59022", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			var out []engine.Message
			for _, p := range g.Players {
				for _, c := range p.Discard {
					if c.Def().Type == "ally" {
						out = append(out, engine.AllyEntersPlayFree{Player: p.ID, Card: c, FromOwner: p.ID})
						break
					}
				}
			}
			return out
		},
	})
	// 59023 Recruitment Drive.
	engine.RegisterBehavior("59023", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Recruitment Drive — discard: next ally is free", Type: engine.AbilityAction, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.Owner)
					if p == nil {
						return nil
					}
					g.Delete(self)
					p.Discard = append(p.Discard, engine.Card{ID: g.NextCardID(), Code: s.Code, Owner: p.ID})
					return []engine.Message{engine.CostDiscountApply{Player: p.ID, Amount: 4}}
				},
			}}
		},
	})
	// 59024 "Avenge Me!": draw 2 when the attached ally falls.
	engine.RegisterBehavior("59024", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyDestroyed)
			if !ok {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if u == nil || u.AttachTo != m.AllyID {
				return nil
			}
			return []engine.Message{engine.DrawCards{Player: u.Owner, N: 2}}
		},
	})
	// 59025 Gilgamesh.
	engine.RegisterBehavior("59025", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p != nil && !p.EDef().HasTrait("eternal") {
				return []engine.Message{engine.ConfuseEntity{Target: p.ID}}
			}
			return nil
		},
	})
	// 59026 Ancient Rivalry: identity-upgrade fetch + ready.
	engine.RegisterBehavior("59026", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var out []engine.Message
			for _, c := range p.Discard {
				if c.Def().Type == "upgrade" && c.Def().CardSet != "" && c.Def().CardSet == "hercules" {
					out = append(out, engine.ReturnDiscardCard{Player: p.ID, CardID: c.ID})
					break
				}
			}
			if !p.Exhausted {
				out = append(out, engine.ReadyEntity{ID: p.ID})
			}
			return out
		},
	})
	// 59027 Limitless Stamina.
	engine.RegisterBehavior("59027", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.ReadyEntity{ID: e.EOwner()}}
		},
	})
	// 59028 Evaluate Threat.
	engine.RegisterBehavior("59028", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			var out []engine.Message
			for _, p := range g.Players {
				for i, c := range p.Deck {
					if c.Def().HasTrait("avenger") || c.Def().HasTrait("s.h.i.e.l.d.") {
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
	// 59029-59031 basic resources.
	engine.RegisterBehavior("59029", &engine.Behavior{})
	engine.RegisterBehavior("59030", &engine.Behavior{})
	engine.RegisterBehavior("59031", &engine.Behavior{})
	// 59032/58034 Avengers Compound: tuck-and-deploy.
	compound := func() *engine.Behavior {
		return &engine.Behavior{
			Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
				s := g.Supports[e.EID()]
				if s == nil {
					return nil
				}
				if len(s.AttachedCards) == 0 {
					return []engine.Ability{{
						Label: "Avengers Compound — tuck an ally from your hand", Type: engine.AbilityAction, Exhaust: true,
						Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
							s := g.Supports[self]
							p := g.Player(s.Owner)
							if p == nil {
								return nil
							}
							for i, c := range p.Hand {
								if c.Def().Type == "ally" {
									card := c
									p.Hand = append(p.Hand[:i:i], p.Hand[i+1:]...)
									s.AttachedCards = append(s.AttachedCards, card)
									s.Counters = 1
									g.Logf("%s waits at the compound", card.Def().Name)
									return nil
								}
							}
							return nil
						},
					}}
				}
				return []engine.Ability{{
					Label: "Avengers Compound — deploy the tucked ally", Type: engine.AbilityAction, Exhaust: true,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						s := g.Supports[self]
						p := g.Player(s.Owner)
						if s == nil || p == nil || len(s.AttachedCards) == 0 {
							return nil
						}
						card := s.AttachedCards[0]
						s.AttachedCards = nil
						s.Counters = 0
						return []engine.Message{engine.AllyEntersPlayFree{Player: p.ID, Card: card}}
					},
				}}
			},
		}
	}
	engine.RegisterBehavior("59032", compound())
	// 59033 Helicarrier: discount.
	engine.RegisterBehavior("59033", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Helicarrier — next card costs 1 less", Type: engine.AbilityAction, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					return []engine.Message{engine.CostDiscountApply{Player: s.Owner, Amount: 1}}
				},
			}}
		},
	})
	// 59034 Quincarrier.
	engine.RegisterBehavior("59034", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "wild"},
	})

	// --- All Versus All nemesis set (59041-59045) ---
	engine.RegisterBehavior("59041", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.MinionDefeated); !ok {
				return nil
			}
			if _, ok := msg.(engine.AllyDestroyed); ok {
				return nil
			}
			s := g.SideSchemes[e.EID()]
			if s == nil {
				return nil
			}
			return []engine.Message{engine.SchemeThreat{Scheme: s.ID, N: 2, Source: s.ID}}
		},
	})
	engine.RegisterBehavior("59042", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if s := allVersusAll(g); s != nil {
				g.Logf("All Versus All joins the fray")
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			if s := allVersusAll(g); s != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: s.ID, N: 2, Source: e.EID()}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("59043", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			if s := allVersusAll(g); s != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: s.ID, N: 2, Source: e.EID()}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("59044", &engine.Behavior{})
	engine.RegisterBehavior("59045", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			allVersusAll(g)
			return nil
		},
	})
}
