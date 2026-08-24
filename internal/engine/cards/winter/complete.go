package winter

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() { registerWinterComplete() }

func winInt(v *int, fallback int) int {
	if v == nil {
		return fallback
	}
	return *v
}

func winterEnemies(g *engine.Game) []engine.EntityID {
	var out []engine.EntityID
	for id := range g.Villains {
		out = append(out, id)
	}
	for id := range g.Minions {
		out = append(out, id)
	}
	return out
}

func registerWinterComplete() {
	// 54012 Captain America ally: ready a S.H.I.E.L.D. character.
	engine.RegisterBehavior("54012", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			if p.Exhausted && hasShield(p.EDef()) {
				return []engine.Message{engine.ReadyEntity{ID: p.ID}}
			}
			for _, aid := range p.Allies {
				if a := g.Allies[aid]; a != nil && a.Exhausted && hasShield(a.EDef()) {
					return []engine.Message{engine.ReadyEntity{ID: aid}}
				}
			}
			return nil
		},
	})
	// 54013 Deathlok: fetch a Sidearm.
	engine.RegisterBehavior("54013", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			for i, c := range p.Deck {
				if c.Code == "54020" {
					card := c
					p.Deck = append(p.Deck[:i:i], p.Deck[i+1:]...)
					p.Hand = append(p.Hand, card)
					return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
				}
			}
			return nil
		},
	})
	// 54014 Firepower: up to 3 weapons, 3 damage each (approximated: one
	// strike per weapon controlled, capped at 3).
	engine.RegisterBehavior("54014", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			weapons := 0
			for _, uid := range p.Upgrades {
				if u := g.Upgrades[uid]; u != nil && u.EDef().HasTrait("weapon") {
					weapons++
				}
			}
			if weapons > 3 {
				weapons = 3
			}
			var out []engine.Message
			for i := 0; i < weapons; i++ {
				for _, id := range winterEnemies(g) {
					out = append(out, engine.DamageEntity{Target: id, Damage: 3, Source: e.EID()})
					break
				}
			}
			return out
		},
	})
	// 54015 One by One.
	engine.RegisterBehavior("54015", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var out []engine.Message
			for _, id := range winterEnemies(g) {
				out = append(out, engine.DamageEntity{Target: id, Damage: 2, Source: e.EID()},
					engine.DamageEntity{Target: id, Damage: 2, Source: e.EID()})
				break
			}
			return out
		},
	})
	// 54016 Spoiling for a Fight.
	engine.RegisterBehavior("54016", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			for i, c := range g.EncounterDeck {
				if c.Def().Type == "minion" {
					card := c
					g.EncounterDeck = append(g.EncounterDeck[:i:i], g.EncounterDeck[i+1:]...)
					def := card.Def()
					mn := &engine.Minion{ID: g.NextEntityID(engine.KindMinion), Code: card.Code,
						MaxHP: winInt(def.HP, 1), AttackVal: winInt(def.Attack, 0), SchemeVal: winInt(def.Scheme, 0),
						EngagedWith: p.ID}
					g.Minions[mn.ID] = mn
					return []engine.Message{engine.MinionEntersPlay{MinionID: mn.ID, Player: p.ID},
						engine.ReadyEntity{ID: p.ID}}
				}
			}
			return nil
		},
	})
	// 54017 Aggressive Stance.
	engine.RegisterBehavior("54017", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.EngageMinion)
			if !ok || g.Player(m.Player) == nil || g.Player(m.Player).ID != e.EOwner() {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			g.Delete(u.ID)
			p := g.Player(u.Owner)
			p.Discard = append(p.Discard, engine.Card{ID: g.NextCardID(), Code: u.Code, Owner: p.ID})
			for i, c := range p.Deck {
				if c.Def().Type == "event" && c.Def().HasTrait("attack") {
					card := c
					p.Deck = append(p.Deck[:i:i], p.Deck[i+1:]...)
					p.Hand = append(p.Hand, card)
					return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
				}
			}
			return nil
		},
	})
	// 54018 Bambino: counters.
	engine.RegisterBehavior("54018", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			g.Upgrades[e.EID()].Counters = 3
			return nil
		},
	})
	// 54019 Man on the Wall.
	engine.RegisterBehavior("54019", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Man on the Wall — discount per engaged minion", Type: engine.AbilityAction,
				HeroOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					p := g.Player(u.Owner)
					if p == nil {
						return nil
					}
					n := 0
					for _, mn := range g.Minions {
						if mn != nil && mn.EngagedWith == p.ID {
							n++
						}
					}
					if n == 0 {
						return nil
					}
					return []engine.Message{engine.CostDiscountApply{Player: p.ID, Amount: n}}
				},
			}}
		},
	})
	// 54020 S.H.I.E.L.D. Sidearm.
	engine.RegisterBehavior("54020", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			g.Upgrades[e.EID()].Counters = 3
			return nil
		},
	})
	// 54021 Nick Fury, Sr. (same rider as the aos printing).
	engine.RegisterBehavior("54021", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			choices = append(choices, engine.Choice{Label: "Draw 2 cards", Kind: engine.ChoicePass}.Msgs(engine.DrawCards{Player: p.ID, N: 2}))
			if g.MainScheme != nil {
				choices = append(choices, engine.Choice{Label: "Remove 3 threat from the main scheme", Kind: engine.ChoicePass}.
					Msgs(engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 3, Source: e.EID()}))
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Nick Fury, Sr. — choose one:", choices...)}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.EndRound); !ok {
				return nil
			}
			p := g.Player(e.EOwner())
			if p == nil || g.Allies[e.EID()] == nil {
				return nil
			}
			return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: e.EID()}}
		},
	})
	// 54022 Super-Soldiers.
	engine.RegisterBehavior("54022", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var out []engine.Message
			for _, id := range winterEnemies(g) {
				out = append(out, engine.DamageEntity{Target: id, Damage: 6, Source: e.EID()},
					engine.ToughEntity{Target: p.ID})
				break
			}
			return out
		},
	})
	// 54023 Winter, Widow, Soldier, Spy.
	engine.RegisterBehavior("54023", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var out []engine.Message
			for _, c := range p.Discard {
				if c.Def().HasTrait("preparation") {
					out = append(out, engine.UpgradeEnterPlay{Player: p.ID, Card: c})
					break
				}
			}
			for _, id := range winterEnemies(g) {
				out = append(out, engine.DamageEntity{Target: id, Damage: 4, Source: e.EID()})
				break
			}
			return out
		},
	})
	// 54024-54026 basic resources.
	engine.RegisterBehavior("54024", &engine.Behavior{})
	engine.RegisterBehavior("54025", &engine.Behavior{})
	engine.RegisterBehavior("54026", &engine.Behavior{})
	// 54032 White Widow.
	engine.RegisterBehavior("54032", &engine.Behavior{})
	// 54033 S.H.I.E.L.D. Deputy.
	engine.RegisterBehavior("54033", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			if p := g.Player(u.Owner); p != nil && u.AttachTo == p.ID {
				p.MaxHP += 1
			} else if a := g.Allies[u.AttachTo]; a != nil {
				a.MaxHP += 1
			}
			return nil
		},
	})

	// --- Whiteout nemesis set (54034-54037) ---
	engine.RegisterBehavior("54034", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			g.Logf("Blizzard's victim is encased in ice")
			return []engine.Message{engine.StunEntity{Target: m.Player}}
		},
	})
	engine.RegisterBehavior("54035", &engine.Behavior{})
	engine.RegisterBehavior("54036", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyEnteredPlay)
			if !ok {
				return nil
			}
			if a := g.Allies[m.Ally]; a != nil {
				return []engine.Message{engine.ExhaustEntity{ID: a.ID}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("54037", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var mn *engine.Minion
			for _, m := range g.Minions {
				if m != nil && m.Code == "54034" {
					mn = m
					break
				}
			}
			if mn == nil {
				for i, c := range g.EncounterDeck {
					if c.Code == "54034" {
						g.EncounterDeck = append(g.EncounterDeck[:i:i], g.EncounterDeck[i+1:]...)
						def := c.Def()
						mn = &engine.Minion{ID: g.NextEntityID(engine.KindMinion), Code: c.Code,
							MaxHP: winInt(def.HP, 1), AttackVal: winInt(def.Attack, 0), SchemeVal: winInt(def.Scheme, 0),
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

// hasShield checks the dotted S.H.I.E.L.D. acronym trait.
func hasShield(def *data.CardDef) bool {
	if def == nil {
		return false
	}
	if def.HasTrait("s.h.i.e.l.d.") {
		return true
	}
	want := []string{"s", "h", "i", "e", "l", "d"}
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
