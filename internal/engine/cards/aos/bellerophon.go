package aos

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() { registerBellerophon() }

// registerBellerophon implements The Bellerophon (50018).
func registerBellerophon() {
	engine.RegisterBehavior("50018", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			s, ok := e.(*engine.Support)
			if !ok {
				return nil
			}
			s.Counters = 3
			g.TLogf("c.theBellerophonEntersPlayWith3MissileCounters")
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s, ok := e.(*engine.Support)
			if !ok || s.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label:   engine.Tf("c.theBellerophonFireAMissile"),
				Type:    engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					owner := g.Player(s.Owner)
					if owner == nil {
						return nil
					}
					var choices []engine.Choice
					for _, target := range g.Players {
						if target.KOed {
							continue
						}
						msgs := []engine.Message{engine.AddEntityCounter{ID: self, N: -1}}

						villainID := g.ActiveVillain
						if g.Villains[villainID] == nil {
							ids := cardutil.SortedIDs(g.Villains)
							if len(ids) > 0 {
								villainID = ids[0]
							}
						}
						if g.Villains[villainID] != nil {
							msgs = append(msgs,
								engine.ClearTough{Target: villainID},
								engine.DamageEntity{Target: villainID, Damage: 3, Source: self},
							)
						}
						for _, id := range cardutil.SortedIDs(g.Minions) {
							mn := g.Minions[id]
							if mn == nil || mn.EngagedWith != target.ID {
								continue
							}
							msgs = append(msgs,
								engine.ClearTough{Target: id},
								engine.DamageEntity{Target: id, Damage: 3, Source: self},
							)
						}
						choices = append(choices, engine.Choice{
							Label: engine.S(target.Name), Kind: engine.ChoiceTarget, SourceID: target.ID,
						}.Msgs(msgs...))
					}
					if len(choices) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{
						Player:   owner.ID,
						Question: engine.Ask(engine.Tf("c.theBellerophonChooseAPlayer"), choices...),
					}}
				},
			}}
		},
	})
}
