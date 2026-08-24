package doctorstrange

// complete.go implements the remaining Doctor Strange pack cards.
// Warning, Foiled! and Iron Man resolve through engine-level interrupt
// windows and cost hooks (see handle.go and costFor).

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func registerRemainingDRS() {
	// Med Team reprint: alias the core behavior.
	if b := engine.LookupBehavior("01080"); b != nil {
		engine.RegisterBehavior("09018", b)
	}

	// The Night Nurse: 3 medical counters, exhaust + counter → heal 1 and
	// discard a status card from a hero.
	engine.RegisterBehavior("09019", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 3}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			if s == nil || s.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: "Exhaust The Night Nurse + counter → heal 1 + clear a status", Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					if s == nil {
						return nil
					}
					var picks []engine.Choice
					for _, q := range g.Players {
						msgs := []engine.Message{engine.AddEntityCounter{ID: self, N: -1}, engine.HealEntity{Target: q.ID, N: 1}}
						var status []engine.Choice
						if q.Stunned {
							status = append(status, engine.Choice{Label: "clear stunned", Kind: engine.ChoiceLabel}.
								Msgs(append(append([]engine.Message{}, msgs...), engine.ClearStun{Target: q.ID})...))
						}
						if q.Confused {
							status = append(status, engine.Choice{Label: "clear confused", Kind: engine.ChoiceLabel}.
								Msgs(append(append([]engine.Message{}, msgs...), engine.ClearConfuse{Target: q.ID})...))
						}
						status = append(status, engine.Choice{Label: "no status", Kind: engine.ChoicePass}.Msgs(msgs...))
						picks = append(picks, engine.Choice{
							Label: q.Name, Kind: engine.ChoiceTarget, SourceID: q.ID,
						}.WithThen(engine.Ask("Clear which status from "+q.Name+"?", status...)))
					}
					return []engine.Message{engine.AskQuestion{Player: s.Owner,
						Question: engine.Ask("The Night Nurse: heal which hero?", picks...)}}
				},
			}}
		},
	})

	// Basic resources + Avengers Mansion reprint.
	engine.RegisterBehavior("09022", &engine.Behavior{})
	engine.RegisterBehavior("09023", &engine.Behavior{})
	engine.RegisterBehavior("09024", &engine.Behavior{})
	if b := engine.LookupBehavior("01091"); b != nil {
		engine.RegisterBehavior("09025", b)
	}

	// The Sorcerer Supreme: +1 hand size while in hero form.
	engine.RegisterBehavior("09026", &engine.Behavior{
		HandSizeBonus: func(g *engine.Game, p *engine.Player) int {
			if p.IsHero() {
				return 1
			}
			return 0
		},
	})

	// Skilled Strike: +2 ATK until end of phase (approximation of the
	// per-attack interrupt, consistent with Mean Swing / Hulk Smash).
	engine.RegisterBehavior("09037", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.ApplyStatBonus{Target: e.EOwner(), ATK: 2}}
		},
	})

	// 09021 Warning: Interrupt — when a hero would take any amount of
	// damage, reduce that amount by 1. Resolved by the interrupt window in
	// handle(DamageEntity) whenever a copy sits in the damaged player's
	// hand.
	engine.RegisterBehavior("09021", &engine.Behavior{})

	// 09038 Foiled!: Interrupt — when a boost card is turned faceup during
	// a scheme activation, cancel its boost icons. Resolved by the window
	// in handle(RevealBoost).
	engine.RegisterBehavior("09038", &engine.Behavior{})

	// 09039 Iron Man: reduce the cost to play each upgrade on Iron Man by
	// 1. Approximation: while Iron Man is in play, every upgrade his
	// controller plays costs 1 less (costFor consults ally CardCost
	// hooks).
	engine.RegisterBehavior("09039", &engine.Behavior{
		CardCost: func(g *engine.Game, p *engine.Player, def *data.CardDef) int {
			if def.Type == "upgrade" {
				return 1
			}
			return 0
		},
	})
}
