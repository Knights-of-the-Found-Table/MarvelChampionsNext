package galaxysmostwanted

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// registerChallenge installs the standard-mode challenge side schemes
// (16178–16182) and the Badoon Headhunter set (16183–16187). Expert
// variants of the challenge schemes share the base behaviors.
func registerChallenge() {
	// 16178 Badoon Blitz: each player draws 1 on defeat.
	blitz := &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, engine.DrawCards{Player: p.ID, N: 1})
			}
			return msgs
		},
	}
	engine.RegisterBehavior("16178", blitz)

	// 16179 Gallery of Splendor: feed each deck top to the Collection.
	gallery := &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, topDeckCollect(p)...)
			}
			return msgs
		},
	}
	engine.RegisterBehavior("16179", gallery)

	// 16180 "There is No Escape": 1 damage to each player on defeat.
	noEscape := &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: 1, Source: s.ID})
			}
			return msgs
		},
	}
	engine.RegisterBehavior("16180", noEscape)

	// 16181 Guerrilla Tactics: 2 evasion counters on Nebula's Ship.
	guerrilla := &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			if ship := g.EnvironmentByCode("16093"); ship != nil {
				ship.Counters += 2
				g.Logf("Nebula's Ship gains 2 evasion counters (%d total)", ship.Counters)
			}
			return nil
		},
	}
	engine.RegisterBehavior("16181", guerrilla)

	// 16182 Kree Supremacy: plain hinder scheme.
	engine.RegisterBehavior("16182", &engine.Behavior{})

	// 16183 Badoon Headhunter: Villainous; boost spawns itself
	// (engine-side "Boost: put into play" handling).
	engine.RegisterBehavior("16183", &engine.Behavior{})

	// 16184 On the Hunt: damage or random discard; boost feeds the
	// villain.
	engine.RegisterBehavior("16184", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var opts []engine.Choice
			opts = append(opts, engine.Choice{
				ID: "dmg", Label: "Take 2 damage", Kind: engine.ChoiceLabel,
			}.Msgs(engine.DamageEntity{Target: p.ID, Damage: 2, Source: t.ID}))
			if len(p.Hand) > 0 {
				opts = append(opts, engine.Choice{
					ID: "discard", Label: "Discard 1 card at random", Kind: engine.ChoiceLabel,
				}.Msgs(engine.DiscardCards{Player: p.ID, Cards: engine.CardList{p.Hand[0]}}))
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask("On the Hunt: choose one", opts...)}}
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			for id := range g.Villains {
				return []engine.Message{engine.DealBoost{Enemy: id}}
			}
			return nil
		},
	})

	// 16185 Dead to Rights: exhaust (or 2 threat); boost feeds the
	// villain.
	engine.RegisterBehavior("16185", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			if !p.Exhausted {
				return []engine.Message{engine.ExhaustEntity{ID: p.ID}}
			}
			if g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: t.ID}}
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			for id := range g.Villains {
				return []engine.Message{engine.DealBoost{Enemy: id}}
			}
			return nil
		},
	})

	// 16186 Headhunter's Henchman: Patrol (engine-side thwart block).
	engine.RegisterBehavior("16186", &engine.Behavior{})

	// 16187 Fugitive Recovery: plain hinder scheme with surge.
	engine.RegisterBehavior("16187", &engine.Behavior{})
}
