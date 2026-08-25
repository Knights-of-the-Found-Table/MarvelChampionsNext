package galaxysmostwanted

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// registerMarket installs The Market's neutral support crew (16150–16177).
// These cards are bought with "units" during the GMW campaign; outside it
// they only appear via scenario effects, so they behave as ordinary player
// cards (the printed unit cost approximates their resource cost).
func registerMarket() {
	// 16150 Brainstorm: name a card type, peek your deck top; match →
	// remove 3 threat, then top-or-bottom, draw 1.
	engine.RegisterBehavior("16150", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || len(p.Deck) == 0 {
				return []engine.Message{engine.DrawCards{Player: e.EOwner(), N: 1}}
			}
			top := p.Deck[0]
			var picks []engine.Choice
			for _, typ := range []string{"ally", "event", "support", "upgrade", "resource"} {
				removal := []engine.Message{engine.DrawCards{Player: p.ID, N: 1}}
				if g.MainScheme != nil && top.Def().Type == typ {
					removal = append([]engine.Message{engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 3, Source: e.EID()}}, removal...)
				}
				picks = append(picks, engine.Choice{
					ID: typ, Label: "Name: " + typ, Kind: engine.ChoiceLabel,
				}.Msgs(removal...))
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask("Brainstorm: name a card type (top card: "+top.Def().Name+")", picks...)}}
		},
	})

	// 16151 By Any Means: 2 threat, 3 damage to the villain, draw 1.
	engine.RegisterBehavior("16151", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			msgs := []engine.Message{engine.DrawCards{Player: e.EOwner(), N: 1}}
			for id := range g.Villains {
				msgs = append([]engine.Message{engine.DamageEntity{Target: id, Damage: 3, Source: e.EID()}}, msgs...)
				break
			}
			if g.MainScheme != nil {
				msgs = append([]engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: e.EID()}}, msgs...)
			}
			return msgs
		},
	})

	// 16152 Contingency Plan: discard top 4, 1 damage per distinct
	// resource type, draw 1.
	engine.RegisterBehavior("16152", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			seen := map[string]bool{}
			for i := 0; i < 4 && len(p.Deck) > 0; i++ {
				c := p.Deck[0]
				p.Deck = p.Deck[1:]
				for _, r := range c.Def().Resources {
					seen[r] = true
				}
				p.Discard = append(p.Discard, c)
			}
			n := len(seen)
			if n == 0 {
				return []engine.Message{engine.DrawCards{Player: p.ID, N: 1}}
			}
			return cardutil.ChooseEnemy("Contingency Plan: deal damage equal to distinct resource types",
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) {
					return n, []engine.Message{engine.DrawCards{Player: p.ID, N: 1}}
				})(g, e)
		},
	})

	// 16153 In Defiance: prevent 2 damage to an identity, draw 1.
	engine.RegisterBehavior("16153", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.DrawCards{Player: e.EOwner(), N: 1}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			// Approximation: used from hand when damage is imminent is not
			// modelable; the play effect covers the draw half.
			return nil
		},
	})

	// 16154 Calculate the Odds: draw 1; a player draws then discards.
	engine.RegisterBehavior("16154", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			for _, tp := range g.Players {
				tp := tp
				inner := []engine.Message{engine.DrawCards{Player: tp.ID, N: 1}}
				if len(tp.Hand) > 0 {
					inner = append(inner, engine.DiscardCards{Player: tp.ID, Cards: engine.CardList{tp.Hand[0]}})
				}
				picks = append(picks, engine.Choice{
					ID: string(tp.ID), Label: tp.Name, Kind: engine.ChoiceLabel,
				}.Msgs(inner...))
			}
			return []engine.Message{
				engine.DrawCards{Player: p.ID, N: 1},
				engine.AskQuestion{Player: p.ID, Question: engine.Ask("Calculate the Odds: who draws and discards?", picks...)},
			}
		},
	})

	// 16155 Creative Solution: draw 1; remove a status card with a bonus.
	engine.RegisterBehavior("16155", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			if p.Tough > 0 {
				picks = append(picks, engine.Choice{
					ID: "tough", Label: "Remove your tough → 3 damage to an enemy", Kind: engine.ChoiceLabel,
				}.Msgs())
			}
			if p.Stunned {
				picks = append(picks, engine.Choice{ID: "stun", Label: "Remove your stunned", Kind: engine.ChoiceLabel}.
					Msgs(engine.ClearStun{Target: p.ID}))
			}
			if p.Confused {
				picks = append(picks, engine.Choice{ID: "confused", Label: "Remove your confused", Kind: engine.ChoiceLabel}.
					Msgs(engine.ClearConfuse{Target: p.ID}))
			}
			for _, id := range cardutil.SortedEnemyIDs(g) {
				if m := g.Minions[id]; m != nil {
					if m.Tough {
						picks = append(picks, engine.Choice{ID: "mt" + id.String(), Label: "Remove a minion's tough → 3 damage", Kind: engine.ChoiceLabel})
					}
					if m.Stunned {
						picks = append(picks, engine.Choice{ID: "ms" + id.String(), Label: "Remove a minion's stunned → 3 threat", Kind: engine.ChoiceLabel}.
							Msgs(engine.ClearStun{Target: id}))
					}
					if m.Confused {
						picks = append(picks, engine.Choice{ID: "mc" + id.String(), Label: "Remove a minion's confused → heal 3", Kind: engine.ChoiceLabel}.
							Msgs(engine.ClearConfuse{Target: id}))
					}
				}
			}
			msgs := []engine.Message{engine.DrawCards{Player: p.ID, N: 1}}
			if len(picks) > 0 {
				msgs = append(msgs, engine.AskQuestion{Player: p.ID,
					Question: engine.Ask("Creative Solution: remove which status card?", picks...)})
			}
			return msgs
		},
	})

	// 16156 Grapple: 1 damage + stun enemy, stun self, draw 1.
	engine.RegisterBehavior("16156", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return cardutil.ChooseEnemy("Grapple: stun which enemy?",
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) {
					return 1, []engine.Message{
						engine.StunEntity{Target: tgt.EID()},
						engine.StunEntity{Target: e.EOwner()},
						engine.DrawCards{Player: e.EOwner(), N: 1},
					}
				})(g, e)
		},
	})

	// 16157 Wing It: 1 damage + confuse enemy, confuse self, draw 1.
	engine.RegisterBehavior("16157", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return cardutil.ChooseEnemy("Wing It: confuse which enemy?",
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) {
					return 1, []engine.Message{
						engine.ConfuseEntity{Target: tgt.EID()},
						engine.ConfuseEntity{Target: e.EOwner()},
						engine.DrawCards{Player: e.EOwner(), N: 1},
					}
				})(g, e)
		},
	})

	// 16158 Close Call: cancel a boost card's ability and icons, draw 1
	// (approximation: play window not modeled — the draw stands in).
	engine.RegisterBehavior("16158", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.DrawCards{Player: e.EOwner(), N: 1}}
		},
	})

	// 16159 Defy Danger: 5 damage; take damage per milled boost icon.
	engine.RegisterBehavior("16159", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			stars := 0
			for i := 0; i < 1 && len(g.EncounterDeck) > 0; i++ {
				c := g.EncounterDeck[0]
				g.EncounterDeck = g.EncounterDeck[1:]
				if b := c.Def().Boost; b != nil && *b > 0 {
					stars += *b
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			msgs := cardutil.ChooseEnemy("Defy Danger: deal 5 damage",
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 5, nil })(g, e)
			return append(msgs,
				engine.DamageEntity{Target: e.EOwner(), Damage: stars, Source: e.EID()},
				engine.DrawCards{Player: e.EOwner(), N: 1})
		},
	})

	// 16160 In Harm's Way: take 2, remove 5 threat, draw 1.
	engine.RegisterBehavior("16160", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if g.MainScheme == nil {
				return []engine.Message{engine.DrawCards{Player: e.EOwner(), N: 1}}
			}
			return []engine.Message{
				engine.DamageEntity{Target: e.EOwner(), Damage: 2, Source: e.EID()},
				engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 5, Source: e.EID()},
				engine.DrawCards{Player: e.EOwner(), N: 1},
			}
		},
	})

	// 16161 Take the Fight to Them: scry the encounter deck, draw 1.
	engine.RegisterBehavior("16161", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			n := min(2*len(g.Players), len(g.EncounterDeck))
			var names []byte
			for i := 0; i < n; i++ {
				names = append(names, (", " + g.EncounterDeck[i].Def().Name)...)
			}
			g.Logf("Take the Fight to Them reveals:%s", string(names))
			return []engine.Message{engine.DrawCards{Player: e.EOwner(), N: 1}}
		},
	})

	// 16162 Armor Plating: exhaust → prevent 1 damage (2 with the
	// Milano).
	engine.RegisterBehavior("16162", &engine.Behavior{
		DamagePrevention: func(g *engine.Game, u *engine.Upgrade, p *engine.Player, n int) (int, int) {
			if u.Exhausted {
				return 0, 0
			}
			u.Exhausted = true
			if _, mil := findMilano(g); mil != nil && mil.EOwner() == p.ID {
				g.Logf("Armor Plating prevents 2 damage")
				return 2, 0
			}
			g.Logf("Armor Plating prevents 1 damage")
			return 1, 0
		},
	})

	// 16163 Heavy Cannon: exhaust → 1 damage to each enemy (+1 villain
	// with the Milano).
	engine.RegisterBehavior("16163", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Exhaust Heavy Cannon → 1 damage to each enemy", Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					var msgs []engine.Message
					bonus := 0
					if _, mil := findMilano(g); mil != nil && mil.EOwner() == e.EOwner() {
						bonus = 1
					}
					for _, id := range cardutil.SortedEnemyIDs(g) {
						dmg := 1
						if g.Villains[id] != nil && bonus > 0 {
							dmg = 2
						}
						msgs = append(msgs, engine.DamageEntity{Target: id, Damage: dmg, Source: self})
					}
					return msgs
				},
			}}
		},
	})

	// 16164 Hyper Thrusters: exhaust → remove 1 threat from each scheme.
	engine.RegisterBehavior("16164", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Exhaust Hyper Thrusters → remove 1 threat from each scheme", Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					var msgs []engine.Message
					if g.MainScheme != nil {
						n := 1
						if _, mil := findMilano(g); mil != nil && mil.EOwner() == e.EOwner() {
							n = 2
						}
						msgs = append(msgs, engine.ThwartScheme{Scheme: g.MainScheme.ID, N: n, Source: self})
					}
					for _, id := range sortedSchemeIDs(g) {
						msgs = append(msgs, engine.ThwartScheme{Scheme: id, N: 1, Source: self})
					}
					return msgs
				},
			}}
		},
	})

	// 16165 Reactor Core: exhaust + discard 2 → next event this turn
	// costs 1 less.
	engine.RegisterBehavior("16165", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil || len(p.Deck) < 1 {
				return nil
			}
			return []engine.Ability{{
				Label: "Exhaust Reactor Core + discard top 2 of your deck → next event costs 1 less", Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					n := 2
					if _, mil := findMilano(g); mil != nil && mil.EOwner() == e.EOwner() {
						n = 1
					}
					return []engine.Message{
						engine.MillPlayerDeck{Player: e.EOwner(), N: n},
						engine.CostDiscountApply{Player: e.EOwner(), Amount: 1},
					}
				},
			}}
		},
	})

	// 16166 Ardent Resolve: ready a friendly character, draw 1.
	engine.RegisterBehavior("16166", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			if p.Exhausted {
				picks = append(picks, engine.Choice{ID: "self", Label: p.Name, Kind: engine.ChoiceLabel}.
					Msgs(engine.ReadyEntity{ID: p.ID}))
			}
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && a.Exhausted {
					picks = append(picks, engine.Choice{Label: a.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id}.
						Msgs(engine.ReadyEntity{ID: id}))
				}
			}
			msgs := []engine.Message{engine.DrawCards{Player: p.ID, N: 1}}
			if len(picks) > 0 {
				msgs = append(msgs, engine.AskQuestion{Player: p.ID,
					Question: engine.Ask("Ardent Resolve: ready which friendly character?", picks...)})
			}
			return msgs
		},
	})

	// 16167 Onrush: cancel a revealed encounter card (not modeled — the
	// cancel window is outside the current interrupt architecture).
	engine.RegisterBehavior("16167", &engine.Behavior{})

	// 16168 Safeguard: tough to up to 2 friendly characters, draw 1.
	engine.RegisterBehavior("16168", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			picks = append(picks, engine.Choice{ID: "self", Label: p.Name, Kind: engine.ChoiceLabel}.
				Msgs(engine.ToughEntity{Target: p.ID}))
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil {
					picks = append(picks, engine.Choice{Label: a.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id}.
						Msgs(engine.ToughEntity{Target: id}))
				}
			}
			return []engine.Message{
				engine.AskQuestion{Player: p.ID,
					Question: engine.AskN("Safeguard: give tough status cards (up to 2)", 2, picks...)},
				engine.DrawCards{Player: p.ID, N: 1},
			}
		},
	})

	// 16169 Sure Gamble: next card this phase costs 3 less.
	engine.RegisterBehavior("16169", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.CostDiscountApply{Player: e.EOwner(), Amount: 3}}
		},
	})

	// 16170 Cargo Hold: exhaust → heal 1 (+1 self with the Milano).
	engine.RegisterBehavior("16170", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			if p.Damage > 0 {
				picks = append(picks, engine.Choice{ID: "self", Label: p.Name, Kind: engine.ChoiceLabel}.
					Msgs(engine.HealEntity{Target: p.ID, N: 1}))
			}
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && a.Damage > 0 {
					picks = append(picks, engine.Choice{Label: a.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id}.
						Msgs(engine.HealEntity{Target: id, N: 1}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Ability{{
				Label: "Exhaust Cargo Hold → heal 1 damage from a friendly character", Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					msgs := []engine.Message{engine.AskQuestion{Player: p.ID,
						Question: engine.Ask("Cargo Hold: heal which friendly character?", picks...)}}
					if _, mil := findMilano(g); mil != nil && mil.EOwner() == p.ID {
						msgs = append(msgs, engine.HealEntity{Target: p.ID, N: 1})
					}
					return msgs
				},
			}}
		},
	})

	// 16171 Mounted Laser: exhaust → 2 damage (3 with the Milano).
	engine.RegisterBehavior("16171", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Exhaust Mounted Laser → deal 2 damage to an enemy", Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					dmg := 2
					if _, mil := findMilano(g); mil != nil && mil.EOwner() == e.EOwner() {
						dmg = 3
					}
					return cardutil.ChooseEnemy("Mounted Laser: deal damage",
						func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return dmg, nil })(g, g.Entity(self))
				},
			}}
		},
	})

	// 16172 Navigation Column: exhaust + discard → draw 1.
	engine.RegisterBehavior("16172", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			return []engine.Ability{{
				Label: "Exhaust Navigation Column + discard 1 card → draw 1 card", Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					if _, mil := findMilano(g); mil != nil && mil.EOwner() == p.ID {
						return []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: 1}, engine.DrawCards{Player: p.ID, N: 1}}
					}
					var picks []engine.Choice
					for _, c := range p.Hand {
						picks = append(picks, engine.Choice{Label: "Discard " + c.Def().Name, Kind: engine.ChoiceCard, CardCode: c.Code}.
							Msgs(engine.DiscardCards{Player: p.ID, Cards: engine.CardList{c}}))
					}
					if len(picks) == 0 {
						return nil
					}
					return []engine.Message{
						engine.AskQuestion{Player: p.ID, Question: engine.Ask("Navigation Column: discard which card?", picks...)},
						engine.DrawCards{Player: p.ID, N: 1},
					}
				},
			}}
		},
	})

	// 16173 Targeting Screen: exhaust → remove 2 threat (3 with the
	// Milano).
	engine.RegisterBehavior("16173", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Exhaust Targeting Screen → remove 2 threat from a scheme", Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					n := 2
					if _, mil := findMilano(g); mil != nil && mil.EOwner() == e.EOwner() {
						n = 3
					}
					return cardutil.ChooseScheme("Targeting Screen", func(g *engine.Game, s engine.Entity) int { return n })(g, g.Entity(self))
				},
			}}
		},
	})

	// 16174 Grand Strategy: draw to max hand size, remove from game.
	engine.RegisterBehavior("16174", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			n := p.HandSize(g) - len(p.Hand)
			if n <= 0 {
				return nil
			}
			g.Logf("Grand Strategy is removed from the game")
			return []engine.Message{engine.DrawCards{Player: p.ID, N: n}}
		},
	})

	// 16175 Power Unleashed: 5 damage + 5 threat removal, removed from
	// game.
	engine.RegisterBehavior("16175", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var msgs []engine.Message
			for id := range g.Villains {
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 5, Source: e.EID()})
				break
			}
			if g.MainScheme != nil {
				msgs = append(msgs, engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 5, Source: e.EID()})
			}
			g.Logf("Power Unleashed is removed from the game")
			return msgs
		},
	})

	// 16176 Tried and True: a player recovers up to 3 discard cards,
	// removed from game.
	engine.RegisterBehavior("16176", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			for _, tp := range g.Players {
				var subs []engine.Choice
				for _, c := range tp.Discard {
					subs = append(subs, engine.Choice{Label: c.Def().Name, Kind: engine.ChoiceCard, CardCode: c.Code}.
						Msgs(engine.ReturnDiscardCard{Player: tp.ID, CardID: c.ID}))
				}
				if len(subs) == 0 {
					continue
				}
				picks = append(picks, engine.Choice{
					ID: string(tp.ID), Label: tp.Name, Kind: engine.ChoiceLabel,
				}.WithThen(engine.AskN("Tried and True: add up to 3 cards to hand", 3, subs...)))
			}
			if len(picks) == 0 {
				return nil
			}
			g.Logf("Tried and True is removed from the game")
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask("Tried and True: which player recovers cards?", picks...)}}
		},
	})

	// 16177 Triple Threat: ready up to 3 characters, removed from game.
	engine.RegisterBehavior("16177", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			if p.Exhausted {
				picks = append(picks, engine.Choice{ID: "self", Label: p.Name, Kind: engine.ChoiceLabel}.
					Msgs(engine.ReadyEntity{ID: p.ID}))
			}
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && a.Exhausted {
					picks = append(picks, engine.Choice{Label: a.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id}.
						Msgs(engine.ReadyEntity{ID: id}))
				}
			}
			var msgs []engine.Message
			if len(picks) > 0 {
				msgs = append(msgs, engine.AskQuestion{Player: p.ID,
					Question: engine.AskN("Triple Threat: ready up to 3 characters", 3, picks...)})
			}
			g.Logf("Triple Threat is removed from the game")
			return msgs
		},
	})
}

// sortedSchemeIDs lists side scheme ids in stable order.
func sortedSchemeIDs(g *engine.Game) []engine.EntityID {
	var out []engine.EntityID
	for id := range g.SideSchemes {
		out = append(out, id)
	}
	// lexical sort for determinism
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
