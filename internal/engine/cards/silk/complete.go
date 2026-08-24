package silk

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

func init() { registerSilkComplete() }

func silkInt(v *int, fallback int) int {
	if v == nil {
		return fallback
	}
	return *v
}

// atlasMinion finds the growing Atlas.
func atlasMinion(g *engine.Game) *engine.Minion {
	for _, mn := range g.Minions {
		if mn != nil && mn.Code == "52035" {
			return mn
		}
	}
	return nil
}

func registerSilkComplete() {
	// 52013 Scarlet Spider: bodyguard interrupt approximated away.
	engine.RegisterBehavior("52013", &engine.Behavior{})
	// 52014 Spider-Byte: cost scaling not modeled.
	engine.RegisterBehavior("52014", &engine.Behavior{})
	// 52015 Not Today!: +2 DEF defense event with threat payoff.
	engine.RegisterBehavior("52015", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			out := []engine.Message{}
			if g.MainScheme != nil {
				out = append(out, engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 2, Source: p.ID})
			}
			return engine.Defends{Defender: p.ID, Against: against, DefBonus: 2}, out, true
		},
	})
	// 52016 "Stop Hitting Yourself": post-defense counter.
	engine.RegisterBehavior("52016", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			return engine.Defends{Defender: p.ID, Against: against},
				[]engine.Message{engine.DamageEntity{Target: against, Damage: 2, Source: p.ID}}, true
		},
	})
	// 52017 Dr. Sinclair.
	engine.RegisterBehavior("52017", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Dr. Sinclair — heal equal to REC and clear a status", Type: engine.AbilityAction,
				Exhaust: true, AlterEgoOnly: true, Cost: 1,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.Owner)
					if p == nil {
						return nil
					}
					return []engine.Message{
						engine.HealEntity{Target: p.ID, N: 2},
						engine.ClearStun{Target: p.ID},
						engine.ClearConfuse{Target: p.ID},
					}
				},
			}}
		},
	})
	// 52018 Energy Shield: X-payment prevention not modeled.
	engine.RegisterBehavior("52018", &engine.Behavior{})
	// 52019 Ready for a Fight: scheme-to-attack swap not modeled.
	engine.RegisterBehavior("52019", &engine.Behavior{})
	// 52020 Stun Gun.
	engine.RegisterBehavior("52020", &engine.Behavior{
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
				Label: "Stun Gun — 1 charge: stun a minion; 2: stun the villain", Type: engine.AbilityAction,
				HeroOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					p := g.Player(u.Owner)
					if p == nil {
						return nil
					}
					var choices []engine.Choice
					for _, id := range enemiesOf(g) {
						target := id
						if g.Minions[id] != nil && u.Counters >= 1 {
							choices = append(choices, engine.Choice{Label: "Stun " + g.Minions[id].EDef().Name + " (1 charge)", Kind: engine.ChoiceTarget, SourceID: id}.
								Msgs(engine.AddEntityCounter{ID: self, N: -1}, engine.StunEntity{Target: target}))
						}
						if g.Villains[id] != nil && u.Counters >= 2 {
							choices = append(choices, engine.Choice{Label: "Stun the villain (2 charges)", Kind: engine.ChoiceTarget, SourceID: id}.
								Msgs(engine.AddEntityCounter{ID: self, N: -2}, engine.StunEntity{Target: target}))
						}
					}
					if len(choices) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Stun Gun — stun whom?", choices...)}}
				},
			}}
		},
	})
	// 52021 Madame Web: peek-and-discard.
	engine.RegisterBehavior("52021", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || len(g.EncounterDeck) == 0 {
				return nil
			}
			top, _ := g.PeekEncounterTop()
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(
				"Madame Web — top of the encounter deck: "+top.Def().Name,
				engine.Choice{Label: "Discard it", Kind: engine.ChoicePass}.Msgs(engine.DiscardEncounterCard{Card: top}),
				engine.Choice{Label: "Leave it", Kind: engine.ChoicePass},
			)}}
		},
	})
	// 52022 Spider-Man: ready a Web-Warrior after acting.
	engine.RegisterBehavior("52022", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			var used bool
			switch m := msg.(type) {
			case engine.AllyAttackWindow:
				used = m.Ally == e.EID()
			case engine.AllyThwartWindow:
				used = m.Ally == e.EID()
			}
			if !used {
				return nil
			}
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			for _, aid := range p.Allies {
				if a := g.Allies[aid]; a != nil && aid != e.EID() && a.Exhausted && a.EDef().HasTrait("web-warrior") {
					return []engine.Message{engine.ReadyEntity{ID: aid}}
				}
			}
			return nil
		},
	})
	// 52023 Across the Spider-Verse.
	engine.RegisterBehavior("52023", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			for _, c := range p.Discard {
				if c.Def().Type == "ally" && c.Def().HasTrait("web-warrior") {
					return []engine.Message{engine.AllyEntersPlayFree{Player: p.ID, Card: c, FromOwner: p.ID}}
				}
			}
			return nil
		},
	})
	// 52024 Investigative Journalism: scheme-cancel not modeled.
	engine.RegisterBehavior("52024", &engine.Behavior{})
	// 52025-52027 basic resources.
	engine.RegisterBehavior("52025", &engine.Behavior{})
	engine.RegisterBehavior("52026", &engine.Behavior{})
	engine.RegisterBehavior("52027", &engine.Behavior{})
	// 52032 Spider-Man 2099: bounce a Web-Warrior after acting.
	engine.RegisterBehavior("52032", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			var used bool
			switch m := msg.(type) {
			case engine.AllyAttackWindow:
				used = m.Ally == e.EID()
			case engine.AllyThwartWindow:
				used = m.Ally == e.EID()
			}
			if !used {
				return nil
			}
			for _, p := range g.Players {
				for _, aid := range p.Allies {
					if a := g.Allies[aid]; a != nil && aid != e.EID() && a.EDef().HasTrait("web-warrior") {
						g.Delete(aid)
						p.Hand = append(p.Hand, engine.Card{ID: g.NextCardID(), Code: a.Code, Owner: p.ID})
						g.Logf("%s swings back to %s's hand", a.EDef().Name, p.Name)
						return nil
					}
				}
			}
			return nil
		},
	})
	// 52033 Spider-Woman: 1 damage per Web-Warrior entry.
	engine.RegisterBehavior("52033", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyEnteredPlay)
			if !ok || g.Allies[m.Ally] == nil || !g.Allies[m.Ally].EDef().HasTrait("web-warrior") {
				return nil
			}
			for _, id := range enemiesOf(g) {
				return []engine.Message{engine.DamageEntity{Target: id, Damage: 1, Source: e.EID()}}
			}
			return nil
		},
	})
	// 52034 Quick Quip.
	engine.RegisterBehavior("52034", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var out []engine.Message
			n := 0
			for _, id := range enemiesOf(g) {
				out = append(out, engine.ConfuseEntity{Target: id})
				n++
				if n >= 2 {
					break
				}
			}
			return out
		},
	})

	// --- Growing Strong nemesis set (52035-52038) ---
	engine.RegisterBehavior("52035", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.EndRound); !ok {
				return nil
			}
			mn := g.Minions[e.EID()]
			if mn == nil {
				return nil
			}
			mn.Counters++
			mn.MaxHP += 2
			g.Logf("Atlas grows (%d growth counters)", mn.Counters)
			return nil
		},
	})
	engine.RegisterBehavior("52036", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			s := g.SideSchemes[e.EID()]
			if s == nil {
				return nil
			}
			if mn := atlasMinion(g); mn != nil {
				s.Threat += mn.Counters
				if mn.Counters >= 10 {
					return []engine.Message{engine.GameOver{Won: false, Reason: "Atlas grew beyond control"}}
				}
			}
			return nil
		},
	})
	engine.RegisterBehavior("52037", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var mn *engine.Minion
			for _, m := range g.Minions {
				if m != nil && m.Code == "52035" {
					mn = m
					break
				}
			}
			if mn == nil {
				for i, c := range g.EncounterDeck {
					if c.Code == "52035" {
						g.EncounterDeck = append(g.EncounterDeck[:i:i], g.EncounterDeck[i+1:]...)
						def := c.Def()
						mn = &engine.Minion{ID: g.NextEntityID(engine.KindMinion), Code: c.Code,
							MaxHP: silkInt(def.HP, 1), AttackVal: silkInt(def.Attack, 0), SchemeVal: silkInt(def.Scheme, 0),
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
	engine.RegisterBehavior("52038", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			n := 0
			if mn := atlasMinion(g); mn != nil {
				n = mn.Counters
				mn.Counters++
				mn.MaxHP += 2
			}
			if n < 3 {
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			return []engine.Message{engine.IndirectDamage{Player: p.ID, N: n}}
		},
	})
}

// enemiesOf lists enemy ids.
func enemiesOf(g *engine.Game) []engine.EntityID {
	var out []engine.EntityID
	for id := range g.Villains {
		out = append(out, id)
	}
	for id := range g.Minions {
		out = append(out, id)
	}
	return out
}

