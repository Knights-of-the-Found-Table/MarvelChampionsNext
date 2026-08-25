package iceman

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func registerIcemanExtras() {
	// 46012 Shark-Girl: +1 ATK per upgrade on the target.
	engine.RegisterBehavior("46012", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyAttackWindow)
			if !ok || m.Ally != e.EID() {
				return nil
			}
			a := g.Allies[e.EID()]
			target := g.Entity(m.Target)
			if a == nil || target == nil {
				return nil
			}
			n := 0
			if mn := g.Minions[m.Target]; mn != nil {
				n = len(mn.Attachments)
			}
			if v := g.Villains[m.Target]; v != nil {
				n = len(v.Attachments)
			}
			if n > 0 {
				a.BonusATK += n
				g.TLogf("c.sharkGirlSmellsGearAtk", n)
			}
			return nil
		},
	})

	// 46013 Glob: 2 damage to an upgraded enemy.
	engine.RegisterBehavior("46013", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "X-Men")
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			a := g.Allies[e.EID()]
			p := g.Player(a.Owner)
			if p == nil {
				return nil
			}
			for _, id := range cardutil.SortedEnemyIDs(g) {
				if mn := g.Minions[id]; mn != nil && len(mn.Attachments) > 0 {
					return []engine.Message{engine.DamageEntity{Target: id, Damage: 2, Source: p.ID}}
				}
				if v := g.Villains[id]; v != nil && len(v.Attachments) > 0 {
					return []engine.Message{engine.DamageEntity{Target: id, Damage: 2, Source: p.ID}}
				}
			}
			return nil
		},
	})

	// 46014 Suppressing Fire: heal on the kill.
	engine.RegisterBehavior("46014", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, mn := range g.Minions {
				if mn != nil {
					t.Target = mn.ID
					return nil
				}
			}
			return nil
		},
	})

	// 46015 Surprise Move: +2 ATK vs upgraded enemies.
	engine.RegisterBehavior("46015", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			g.EventDamageBonus[p.ID] += 2
			g.TLogf("c.surpriseMove2Atk")
			return nil
		},
	})

	// 46016 Take That!: 7 damage to an upgraded enemy.
	engine.RegisterBehavior("46016", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				upgraded := false
				if mn := g.Minions[id]; mn != nil && len(mn.Attachments) > 0 {
					upgraded = true
				}
				if v := g.Villains[id]; v != nil && len(v.Attachments) > 0 {
					upgraded = true
				}
				if !upgraded {
					continue
				}
				enemy := g.Entity(id)
				choices = append(choices, engine.Choice{
					Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget, SourceID: id,
				}.Msgs(engine.DamageEntity{Target: id, Damage: 7, Source: p.ID}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.takeThat7DamageTo"), choices...)}}
		},
	})

	// 46017 Looking for Trouble: summon a minion, remove 3 threat.
	engine.RegisterBehavior("46017", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			for i := 0; i < 30; i++ {
				c, ok := g.DrawEncounter()
				if !ok {
					return nil
				}
				if c.Def().Type == "minion" {
					var msgs []engine.Message
					msgs = append(msgs, engine.RevealEncounterCard{Player: p.ID, Card: c})
					if g.MainScheme != nil {
						msgs = append(msgs, engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 3, Source: p.ID})
					}
					return msgs
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			return nil
		},
	})

	// 46018 Keep Up the Pressure: fetch Attack events.
	engine.RegisterBehavior("46018", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			for _, p := range g.Players {
				for _, c := range append(engine.CardList{}, p.Deck...) {
					if c.Def().Type == "event" && c.Def().HasTrait("Attack") {
						if _, ok := p.Deck.Remove(c.ID); ok {
							p.Hand = append(p.Hand, c)
							g.TLogf("c.draws", p.Name, c)
						}
						break
					}
				}
			}
			return nil
		},
	})

	// 46019 Shadowcat: strip a scheme's icons.
	engine.RegisterBehavior("46019", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "X-Men")
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for _, s := range g.SideSchemes {
				if s != nil {
					s.Crisis = false
					s.Hazard = 0
					g.TLogf("c.losesItsIconsShadowcat", s)
					return nil
				}
			}
			if g.MainScheme != nil {
				g.MainScheme.Crisis = false
				g.TLogf("c.theMainSchemeLosesItsIconsShadowcat")
			}
			return nil
		},
	})

	// 46020 Beak: threat per X-Men ally.
	engine.RegisterBehavior("46020", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			a := g.Allies[e.EID()]
			p := g.Player(a.Owner)
			if p == nil || len(g.Schemes()) == 0 {
				return nil
			}
			n := 0
			for _, id := range p.Allies {
				if x := g.Allies[id]; x != nil && x.EDef().HasTrait("X-Men") {
					n++
				}
			}
			return []engine.Message{engine.ThwartScheme{Scheme: g.Schemes()[0], N: n, Source: p.ID}}
		},
	})

	// 46021 Team-Building Exercise: play discount (approximated: flat
	// discount this phase).
	engine.RegisterBehavior("46021", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.teamBuildingExerciseNextMatchingCardCosts1Less"), Type: engine.AbilityAction,
				HeroOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					return []engine.Message{engine.CostDiscountApply{Player: s.Owner, Amount: 1}}
				},
			}}
		},
	})

	// 46022 Recuperation: heal REC.
	engine.RegisterBehavior("46022", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			rec := 0
			if r := p.HeroDef().Recover; r != nil {
				rec = *r
			}
			if rec == 0 {
				rec = 2
			}
			return []engine.Message{engine.HealEntity{Target: p.ID, N: rec}}
		},
	})

	// 46023 The Power in All of Us: engine powerOfBonus.
	engine.RegisterBehavior("46023", &engine.Behavior{})

	registerSauron()
}

func registerSauron() {
	// 46029 Sauron: fetch Life Drain.
	engine.RegisterBehavior("46029", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return nil
			}
			for _, c := range append(engine.CardList{}, g.EncounterDeck...) {
				if c.Code == "46031" {
					g.EncounterDeck.Remove(c.ID)
					g.ShuffleEncounterDeck()
					return []engine.Message{engine.RevealEncounterCard{Player: mn.EngagedWith, Card: c}}
				}
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if id := firstVillain(g); id != "" {
				return []engine.Message{engine.HealEntity{Target: id, N: 3}, engine.ToughEntity{Target: id}}
			}
			return nil
		},
	})

	// 46030 Sauron Lives!: defeat deals him facedown.
	engine.RegisterBehavior("46030", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			p := g.Player(cardutil.FirstPlayerID(g))
			if p == nil {
				return nil
			}
			for _, c := range append(engine.CardList{}, g.EncounterDeck...) {
				if c.Code == "46029" {
					g.EncounterDeck.Remove(c.ID)
					p.EncounterDown = append(p.EncounterDown, c)
					g.TLogf("c.sauronIsDealtFacedownTo", p.Name)
					return nil
				}
			}
			return nil
		},
	})

	// 46031 Life Drain: big minion activates; attack rider.
	engine.RegisterBehavior("46031", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			best, bestID := -1, engine.EntityID("")
			engaged := engine.PlayerID("")
			for _, mn := range g.Minions {
				if mn != nil && mn.MaxHP > best {
					best, bestID, engaged = mn.MaxHP, mn.ID, mn.EngagedWith
				}
			}
			t.Target = bestID
			if bestID != "" && engaged != "" {
				return []engine.Message{engine.MinionActivates{MinionID: bestID, Player: engaged}}
			}
			if c, ok := g.DrawEncounter(); ok {
				return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AskAttack)
			if !ok {
				return nil
			}
			t := g.Attachments[e.EID()]
			if t == nil || t.Target == "" || m.Enemy != t.Target {
				return nil
			}
			return []engine.Message{
				engine.DamageEntity{Target: m.Player, Damage: 2, Source: t.Target},
				engine.ToughEntity{Target: t.Target},
			}
		},
	})

	// 46032 The Eye of Sauron: icon-based punishment.
	engine.RegisterBehavior("46032", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			n := 2
			for _, mn := range g.Minions {
				if mn != nil && mn.Code == "46029" {
					n = 3
				}
			}
			var discarded engine.CardList
			for i := 0; i < n && len(p.Deck) > 0; i++ {
				discarded = append(discarded, p.Deck[0])
				p.Deck = p.Deck[1:]
			}
			var msgs []engine.Message
			if len(discarded) > 0 {
				msgs = append(msgs, engine.DiscardCards{Player: p.ID, Cards: discarded})
			}
			for _, c := range discarded {
				for _, r := range c.Def().Resources {
					switch r {
					case "energy":
						if g.MainScheme != nil {
							msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: t.ID})
						}
					case "mental":
						if len(p.Hand) > 0 {
							msgs = append(msgs, engine.DiscardCards{Player: p.ID, Cards: engine.CardList{p.Hand[0]}})
						}
					case "physical":
						msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: 1, Source: t.ID})
					case "wild":
						msgs = append(msgs, engine.ExhaustEntity{ID: p.ID})
					}
				}
			}
			return msgs
		},
	})
}

func firstVillain(g *engine.Game) engine.EntityID {
	for id := range g.Villains {
		return id
	}
	return ""
}
