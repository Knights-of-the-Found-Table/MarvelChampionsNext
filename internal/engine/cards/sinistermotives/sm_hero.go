// Package sinistermotives registers the Sinister Motives box: Ghost-Spider
// and Miles Morales' aspect cards, the Sandman, Venom, Mysterio, Sinister
// Six and Venom Goblin scenarios, and the box's modular sets.
package sinistermotives

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// registerSMHeroCards installs the box's hero-pack aspect cards (27012–
// 27055).
func registerSMHeroCards() {
	// 27012 Spider-UK: retaliation-ish defense trigger.
	engine.RegisterBehavior("27012", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowDefended)
			if !ok || w.Defender != e.EID() {
				return nil
			}
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			n := 1 // the identity
			for _, id := range p.Supports {
				if g.Supports[id] != nil {
					n++
				}
			}
			for _, id := range p.Upgrades {
				if g.Upgrades[id] != nil {
					n++
				}
			}
			for _, id := range p.Allies {
				if g.Allies[id] != nil {
					n++
				}
			}
			g.TLogf("c.spiderUkRetaliatesFor", n)
			return []engine.Message{engine.DamageEntity{Target: w.Against, Damage: n, Source: e.EID()}}
		},
	})

	// 27013 Bait and Switch: the villain attacks you; remove 4 threat.
	engine.RegisterBehavior("27013", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var msgs []engine.Message
			for id := range g.Villains {
				msgs = append(msgs, engine.VillainActivates{VillainID: id, Player: e.EOwner()})
				break
			}
			if g.MainScheme != nil {
				msgs = append(msgs, engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 4, Source: e.EID()})
			}
			return msgs
		},
	})

	// 27014 Jump Flip: prevent 2 damage (energy rider approximated away).
	engine.RegisterBehavior("27014", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			if !p.IsHero() || p.Exhausted {
				return engine.Defends{}, nil, false
			}
			d := engine.Defends{Defender: p.ID, Against: against, ExtraPrevent: 2}
			return d, nil, true
		},
	})

	// 27015 Return the Favor: dig a treachery, then 5 damage.
	engine.RegisterBehavior("27015", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for len(g.EncounterDeck) > 0 {
				c := g.EncounterDeck[0]
				g.EncounterDeck = g.EncounterDeck[1:]
				if c.Def().Type == "treachery" {
					var msgs []engine.Message
					msgs = append(msgs, engine.RevealEncounterCard{Player: e.EOwner(), Card: c})
					for id := range g.Villains {
						msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 5, Source: e.EID()})
						break
					}
					return msgs
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			return nil
		},
	})

	// 27016 What Doesn't Kill Me: heal 2 → ready.
	engine.RegisterBehavior("27016", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || p.Damage < 2 {
				return nil
			}
			return []engine.Message{
				engine.HealEntity{Target: p.ID, N: 2},
				engine.ReadyEntity{ID: p.ID},
			}
		},
	})

	// 27017 Spider-Man (Miguel): on leaving play, mill 3 → damage per
	// boost icon.
	engine.RegisterBehavior("27017", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.AllyDestroyed)
			if !ok || d.AllyID != e.EID() {
				return nil
			}
			stars := 0
			for i := 0; i < 3 && len(g.EncounterDeck) > 0; i++ {
				c := g.EncounterDeck[0]
				g.EncounterDeck = g.EncounterDeck[1:]
				if b := c.Def().Boost; b != nil && *b > 0 {
					stars += *b
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			if stars == 0 {
				return nil
			}
			for id := range g.Villains {
				return []engine.Message{engine.DamageEntity{Target: id, Damage: stars, Source: e.EID()}}
			}
			return nil
		},
	})

	// 27018 Across the Spider-Verse: exhaust a Web-Warrior → revive one
	// from discard (the pay-more repeat is approximated away).
	engine.RegisterBehavior("27018", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var exhausts []engine.Choice
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && !a.Exhausted && g.EntityHasTrait(id, "web-warrior") {
					exhausts = append(exhausts, engine.Choice{
						Label: engine.S("Exhaust " + a.EDef().Name), Kind: engine.ChoiceTarget, SourceID: id,
					}.Msgs(engine.ExhaustEntity{ID: id}))
				}
			}
			if p.Exhausted == false {
				exhausts = append(exhausts, engine.Choice{
					ID: "self", Label: engine.S("Exhaust " + p.Name + " (Web-Warrior)"), Kind: engine.ChoiceLabel,
				}.Msgs(engine.ExhaustEntity{ID: p.ID}))
			}
			if len(exhausts) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.acrossTheSpiderVerseExhaustWhichWebWarrior"), exhausts...)}}
		},
	})

	// 27019/27050 Young Love: heal 3 from Gwen and Miles (approximated:
	// heal 3 from your identity when it is one of them).
	youngLove := &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil || p.IsHero() || p.Damage == 0 {
				return nil
			}
			if p.AlterEgoCode[:5] != "27001" && p.AlterEgoCode[:5] != "27030" {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.heal3DamageFromYourIdentity"), Type: engine.AbilityAction,
				AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.HealEntity{Target: e.EOwner(), N: 3}}
				},
			}}
		},
	}
	engine.RegisterBehavior("27019", youngLove)
	engine.RegisterBehavior("27050", youngLove)

	// 27020–27022 / 27051–27053 basic resources.
	for _, code := range []string{"27020", "27021", "27022", "27051", "27052", "27053"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	// 27023 Web of Life and Destiny: Web-Warrior allies leaving play draw
	// someone a card.
	engine.RegisterBehavior("27023", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.AllyDestroyed)
			if !ok {
				return nil
			}
			if a := g.Allies[d.AllyID]; a != nil && g.EntityHasTrait(d.AllyID, "web-warrior") {
				owner := g.Player(a.Owner)
				if owner != nil {
					return []engine.Message{engine.DrawCards{Player: owner.ID, N: 1}}
				}
			}
			return nil
		},
	})

	// 27024 Plan B: exhaust + random discard → 2 damage.
	engine.RegisterBehavior("27024", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil || len(p.Hand) == 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustPlanBDiscard1RandomCard2DamageToAnEnemy"), Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					c := p.Hand[0]
					return append([]engine.Message{engine.DiscardCards{Player: p.ID, Cards: engine.CardList{c}}},
						cardutil.ChooseEnemy(engine.Tf("c.planBDeal2Damage"),
							func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 2, nil })(
							g, g.Entity(self))...)
				},
			}}
		},
	})

	// 27041 Spider-Woman: -1 cost per confused enemy.
	engine.RegisterBehavior("27041", &engine.Behavior{})

	// 27042 Homeland Intervention: exhaust up to 3 S.H.I.E.L.D. cards →
	// remove 2 threat each.
	engine.RegisterBehavior("27042", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var opts []engine.Choice
			ids := append(append([]engine.EntityID{}, p.Supports...), p.Upgrades...)
			ids = append(ids, p.Allies...)
			for _, id := range ids {
				var def = g.Entity(id).EDef()
				if def != nil && def.HasTrait("shield") {
					opts = append(opts, engine.Choice{
						Label: engine.S(def.Name), Kind: engine.ChoiceTarget, SourceID: id,
					}.Msgs(engine.ExhaustEntity{ID: id}))
				}
			}
			if len(opts) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.AskN(engine.Tf("c.homelandInterventionExhaustUpTo3SHIELDCards2ThreatEach"), 3, opts...)}}
		},
	})

	// 27043 Global Logistics: peek 4 of a deck (approximated: log the
	// top 4 of your deck).
	engine.RegisterBehavior("27043", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			n := min(4, len(p.Deck))
			for i := 0; i < n; i++ {
				g.TLogf("c.globalLogisticsSees", p.Deck[i].Def().Name)
			}
			return nil
		},
	})

	// 27044 Field Agent: 3 backup counters; soak ally consequential
	// damage (approximation: auto-prevents 1 ally damage per counter).
	engine.RegisterBehavior("27044", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 3}}
		},
	})

	// 27046 Agent 13: after attacking or thwarting, ready a S.H.I.E.L.D.
	// support.
	engine.RegisterBehavior("27046", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			p := g.Player(e.EOwner())
			var hit bool
			switch m := msg.(type) {
			case engine.AllyAttackWindow:
				hit = m.Ally == e.EID()
			case engine.AllyThwartWindow:
				hit = m.Ally == e.EID()
			}
			if !hit || p == nil {
				return nil
			}
			for _, id := range p.Supports {
				if s := g.Supports[id]; s != nil && s.Exhausted && s.EDef().HasTrait("shield") {
					return []engine.Message{engine.ReadyEntity{ID: id}}
				}
			}
			return nil
		},
	})

	// 27047 Dum Dum Dugan: basic powers get +1 per exhausted S.H.I.E.L.D.
	// card (approximation: flat +1 ATK/+1 THW).
	engine.RegisterBehavior("27047", &engine.Behavior{})

	// 27048 Ghost-Spider (ally): on leaving play, tutor an event.
	engine.RegisterBehavior("27048", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.AllyDestroyed)
			if !ok || d.AllyID != e.EID() {
				return nil
			}
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			seen := map[string]bool{}
			for _, c := range p.Deck {
				def := c.Def()
				if def.Type == "event" && !seen[def.Code] {
					seen[def.Code] = true
					picks = append(picks, engine.Choice{
						Label: engine.S(def.Name + " (deck)"), Kind: engine.ChoiceCard, CardCode: def.Code,
					}.Msgs(engine.TakeDeckCard{Player: p.ID, CardID: c.ID}, engine.ShufflePlayerDeck{Player: p.ID}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.ghostSpiderAddWhichEventToHand"), picks...)}}
		},
	})

	// 27049 Spider-Man (Peter): after attacking or thwarting, ready
	// another Web-Warrior.
	engine.RegisterBehavior("27049", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			p := g.Player(e.EOwner())
			var tgt engine.EntityID
			switch m := msg.(type) {
			case engine.AllyAttackWindow:
				tgt = m.Ally
			case engine.AllyThwartWindow:
				tgt = m.Ally
			}
			if tgt != e.EID() || p == nil {
				return nil
			}
			for _, id := range p.Allies {
				if id != e.EID() && g.Allies[id] != nil && g.Allies[id].Exhausted && g.EntityHasTrait(id, "web-warrior") {
					return []engine.Message{engine.ReadyEntity{ID: id}}
				}
			}
			return nil
		},
	})

	// 27054 Government Liaison: play a S.H.I.E.L.D. card at -1 (windowed
	// plays are not modeled; registered for the discount approximation).
	engine.RegisterBehavior("27054", &engine.Behavior{})

	// 27055 Sky-Destroyer: after playing a S.H.I.E.L.D. card, 2 damage.
	engine.RegisterBehavior("27055", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			pc, ok := msg.(engine.PlayCard)
			if !ok || pc.Player != e.EOwner() {
				return nil
			}
			s := g.Supports[e.EID()]
			if s == nil || s.Exhausted {
				return nil
			}
			if def := pc.Card.Def(); def == nil || !def.HasTrait("shield") {
				return nil
			}
			return cardutil.ChooseEnemy(engine.Tf("c.skyDestroyerDeal2Damage"),
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 2, nil })(
				g, e)
		},
	})
}

// registerPromos installs the promo Venom ally and Symbiote Suit (27190–
// 27191).
func registerPromos() {
	// 27190 Venom (ally): pay 1 damage to Venom → hit an enemy for the
	// boost area of the revealed card.
	engine.RegisterBehavior("27190", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			r, ok := msg.(engine.RevealEncounterCard)
			if !ok {
				return nil
			}
			a := g.Allies[e.EID()]
			if a == nil || a.HP() <= 1 {
				return nil
			}
			stars := 0
			if b := r.Card.Def().Boost; b != nil {
				stars = *b
			}
			if stars <= 0 {
				return nil
			}
			g.TLogf("c.venomAllyChannelsForDamage", r.Card, stars)
			return cardutil.ChooseEnemy(engine.Tf("c.venomDealDamageEqualToBoostIcons"),
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) {
					return stars, []engine.Message{engine.DamageEntity{Target: e.EID(), Damage: 1}}
				})(g, e)
		},
	})

	// 27191 Symbiote Suit: +1 each power, +1 hand size, +10 HP.
	engine.RegisterBehavior("27191", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p != nil {
				p.MaxHP += 10
				g.TLogf("c.gets10HitPoints", p.Name)
			}
			return nil
		},
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{ATK: 1, THW: 1, DEF: 1}
		},
		HandSizeBonus: func(g *engine.Game, p *engine.Player) int { return 1 },
	})
}
