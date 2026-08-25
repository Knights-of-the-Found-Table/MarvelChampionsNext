// Package extras registers individual cards from packs that are not
// otherwise implemented, as required by imported decklists (the Daredevil
// Protection reference deck). Each card carries its pack of origin in the
// comment.
package extras

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	// Core Set
	registerPowerOfProtection()
	registerArmoredVest()
	// Doctor Strange? No — 09020 Unflappable ( its pack carries no other
	// cards used here).
	registerUnflappable()
	registerDeftFocus()
	registerPowerfulPunch()
	registerEstablishPerimeter()
	registerRenderMedicalAid()
	registerChangeOfFortune()
	registerUnderControl()
	registerDefensiveConditioning()
}

// 01079 The Power of Protection (Core Set): doubles while paying for a
// Protection card — handled generically in the payment validator.
func registerPowerOfProtection() {
	engine.RegisterBehavior("01079", &engine.Behavior{})
}

// 01081 Armored Vest (Core Set): play under any player's control
// (approximation: your identity only); your hero gets +1 DEF.
func registerArmoredVest() {
	engine.RegisterBehavior("01081", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{DEF: 1}
		},
	})
}

// 09020 Unflappable: response — after you defend against an attack and
// take no damage, exhaust → draw 1 card.
func registerUnflappable() {
	engine.RegisterBehavior("09020", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowDefended)
			s, ok2 := e.(*engine.Support)
			if !ok || !ok2 || s.Exhausted {
				return nil
			}
			p := g.Player(s.Owner)
			if p == nil || w.Defender != p.ID || w.DamageTaken != 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player: p.ID,
				Question: engine.Ask(engine.Tf("c.unflappableExhaustToDraw1Card"),
					engine.Choice{
						ID: "draw", Label: engine.Tf("c.exhaustUnflappableDraw1"), Kind: engine.ChoiceLabel,
					}.Msgs(
						engine.ExhaustEntity{ID: s.ID},
						engine.DrawCards{Player: p.ID, N: 1},
					),
					engine.Choice{ID: "skip", Label: engine.Tf("c.skip"), Kind: engine.ChoicePass},
				),
			}}
		},
	})
}

// 16024 Deft Focus: hero action — exhaust → the next [[superpower]] card
// you play this turn costs 1 less (approximation: this phase).
func registerDeftFocus() {
	engine.RegisterBehavior("16024", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label:    engine.Tf("c.deftFocusTheNextSuperpowerCardCosts1Less"),
				Type:     engine.AbilityAction,
				Exhaust:  true,
				HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					if p := g.Player(e.EOwner()); p != nil {
						p.CostDiscounts = append(p.CostDiscounts, engine.CostDiscount{Trait: "superpower", Amount: 1})
						g.TLogf("c.sNextSuperpowerCardThisPhaseCosts1Less", p.Name)
					}
					return nil
				},
			}}
		},
	})
}

// 32014 Powerful Punch: hero interrupt — when an enemy initiates an
// attack, deal 4 damage to it (approximation: offered in the defense
// prompt instead of the attack declaration).
func registerPowerfulPunch() {
	engine.RegisterBehavior("32014", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			d := engine.Defends{Defender: p.ID, Against: against, Undefended: true}
			return d, []engine.Message{engine.DamageEntity{Target: against, Damage: 4, Source: p.ID}}, true
		},
	})
}

// 40020 Establish Perimeter (player side scheme): when defeated, each
// identity gains a tough status card.
func registerEstablishPerimeter() {
	engine.RegisterBehavior("40020", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, engine.ToughEntity{Target: p.ID})
			}
			return msgs
		},
	})
}

// 42017 Render Medical Aid (player side scheme): when defeated, each
// player heals a total of 5 damage (approximation: the identity heals 5).
func registerRenderMedicalAid() {
	engine.RegisterBehavior("42017", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, engine.HealEntity{Target: p.ID, N: 5})
			}
			return msgs
		},
	})
}

// 48014 Change of Fortune: response — after you defeat an enemy during
// the villain phase, exhaust → draw 2 cards.
func registerChangeOfFortune() {
	engine.RegisterBehavior("48014", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionDefeated)
			s, ok2 := e.(*engine.Support)
			if !ok || !ok2 || s.Exhausted {
				return nil
			}
			if g.Phase != engine.PhaseVillain {
				return nil
			}
			p := g.Player(s.Owner)
			if p == nil {
				return nil
			}
			defeated := false
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && a.ID == m.MinionID {
					defeated = true
				}
			}
			// approximate attribution: any minion defeated while you
			// control an ally counts as your defeat
			if !defeated && len(p.Allies) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player: p.ID,
				Question: engine.Ask(engine.Tf("c.changeOfFortuneExhaustToDraw2Cards"),
					engine.Choice{
						ID: "draw", Label: engine.Tf("c.exhaustChangeOfFortuneDraw2"), Kind: engine.ChoiceLabel,
					}.Msgs(
						engine.ExhaustEntity{ID: s.ID},
						engine.DrawCards{Player: p.ID, N: 2},
					),
					engine.Choice{ID: "skip", Label: engine.Tf("c.skip"), Kind: engine.ChoicePass},
				),
			}}
		},
	})
}

// 48015 Under Control: attach to a minion; after a hero defends against
// that minion's attack and takes no damage, deal 4 damage to it.
func registerUnderControl() {
	engine.RegisterBehavior("48015", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			choices := cardutil.MinionChoices(g, func(target engine.EntityID) []engine.Message {
				return []engine.Message{engine.AttachUpgrade{ID: e.EID(), Target: target}}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.underControlAttachToAMinion"), choices...),
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowDefended)
			u, ok2 := e.(*engine.Upgrade)
			if !ok || !ok2 || w.Against != u.AttachTo || w.DamageTaken != 0 {
				return nil
			}
			if !w.Defender.Is(engine.KindPlayer) {
				return nil
			}
			g.TLogf("c.underControl4DamageTo", g.Entity(u.AttachTo).EDef().Name)
			return []engine.Message{engine.DamageEntity{Target: u.AttachTo, Damage: 4, Source: u.ID}}
		},
	})
}

// 56046 Defensive Conditioning: you get +3 hit points and your hero gets
// +1 DEF.
func registerDefensiveConditioning() {
	engine.RegisterBehavior("56046", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{DEF: 1}
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if p := g.Player(e.EOwner()); p != nil {
				p.MaxHP += 3
				g.TLogf("c.gets3MaxHitPointsDefensiveConditioning", p.Name)
			}
			return nil
		},
	})
}
