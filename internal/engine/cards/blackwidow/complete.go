// Package blackwidow registers the Black Widow hero pack. Boost-cancel
// and reveal-cancel windows do not exist in the engine yet; those cards
// are registered with documented approximations.
package blackwidow

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() {
	registerBlackWidow()
}

func registerBlackWidow() {
	// Black Widow identity: Widowmaker (response after triggering a
	// Preparation ability) has no window; the identity plays normally.
	engine.RegisterBehavior("08001", &engine.Behavior{})

	// Winter Soldier: 1 less per Preparation controlled.
	engine.RegisterBehavior("08002", &engine.Behavior{
		CardCost: func(g *engine.Game, p *engine.Player, def *data.CardDef) int {
			if def.Code != "08002" {
				return 0
			}
			n := 0
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil && u.EDef().HasTrait("preparation") {
					n++
				}
			}
			return min(n, 4)
		},
	})

	// Covert Ops: remove 4 threat + confuse the villain.
	engine.RegisterBehavior("08003", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			msgs := []engine.Message{engine.AskQuestion{Player: e.EOwner(), Question: engine.Ask(
				"Covert Ops: remove 4 threat from which scheme?", schemePicks(g, 4, e.EOwner())...)}}
			for id := range g.Villains {
				msgs = append(msgs, engine.ConfuseEntity{Target: id})
				break
			}
			return msgs
		},
	})

	// Dance of Death: three attacks of 1/2/3.
	engine.RegisterBehavior("08004", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var msgs []engine.Message
			for _, n := range []int{1, 2, 3} {
				msgs = append(msgs, cardutil.ChooseEnemy(
					fmt.Sprintf("Dance of Death: deal %d damage to which enemy?", n),
					func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return n, nil })(g, e)...)
			}
			return msgs
		},
	})

	// Safe House #29: alter-ego exhaust → Preparation from discard.
	engine.RegisterBehavior("08005", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Exhaust Safe House #29 → add a Preparation from your discard", Type: engine.AbilityAction,
				Exhaust: true, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(g.Entity(self).EOwner())
					if p == nil {
						return nil
					}
					var picks []engine.Choice
					seen := map[string]bool{}
					for _, c := range p.Discard {
						def := c.Def()
						if !def.HasTrait("preparation") || seen[c.Code] {
							continue
						}
						seen[c.Code] = true
						picks = append(picks, engine.Choice{Label: def.Name, Kind: engine.ChoiceCard, CardCode: def.Code}.
							Msgs(engine.ReturnDiscardCard{Player: p.ID, CardID: c.ID}))
					}
					if len(picks) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID,
						Question: engine.Ask("Add which Preparation to hand?", picks...)}}
				},
			}}
		},
	})

	// Attacrobatics: boost-icon cancel — no boost window; approximated
	// to nothing.
	engine.RegisterBehavior("08006", &engine.Behavior{})

	// Black Widow's Gauntlet: wild resource (the Preparation-only gate is
	// approximated away).
	engine.RegisterBehavior("08007", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "wild"},
	})

	// Grappling Hook: treachery cancel from play — the treachery window
	// only enumerates hand cards; approximated to nothing.
	engine.RegisterBehavior("08008", &engine.Behavior{})

	// Synth-Suit: +1 DEF (the ready rider lacks a window).
	engine.RegisterBehavior("08009", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{DEF: 1} },
	})

	// Widow's Bite: a minion entering play may be shot for 2 + stun.
	engine.RegisterBehavior("08010", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionEntersPlay)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil || m.Player != u.Owner {
				return nil
			}
			mn := g.Minions[m.MinionID]
			if mn == nil {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: u.Owner, Question: engine.Ask(
				"Discard Widow's Bite → deal 2 damage to "+mn.EDef().Name+" and stun it?",
				engine.Choice{ID: "use", Label: "Use Widow's Bite", Kind: engine.ChoiceAbility, SourceID: e.EID(), CardCode: "08010"}.
					Msgs(engine.DiscardControlled{Player: u.Owner, ID: e.EID()},
						engine.DamageEntity{Target: m.MinionID, Damage: 2, Source: u.Owner},
						engine.StunEntity{Target: m.MinionID}),
				engine.Choice{ID: "skip", Label: "Skip", Kind: engine.ChoicePass},
			)}}
		},
	})

	// Agent Coulson: search deck and discard for a Preparation.
	engine.RegisterBehavior("08011", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			seen := map[string]bool{}
			for _, c := range append(append(engine.CardList{}, p.Deck...), p.Discard...) {
				def := c.Def()
				if !def.HasTrait("preparation") || seen[c.Code] {
					continue
				}
				seen[c.Code] = true
				_, fromDeck := p.Deck.Find(c.ID)
				label := def.Name
				if fromDeck {
					label += " (deck)"
				} else {
					label += " (discard)"
				}
				if fromDeck {
					picks = append(picks, engine.Choice{Label: label, Kind: engine.ChoiceCard, CardCode: def.Code}.
						Msgs(engine.TakeDeckCard{Player: p.ID, CardID: c.ID}, engine.ShufflePlayerDeck{Player: p.ID}))
				} else {
					picks = append(picks, engine.Choice{Label: label, Kind: engine.ChoiceCard, CardCode: def.Code}.
						Msgs(engine.ReturnDiscardCard{Player: p.ID, CardID: c.ID}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask("Coulson: add which Preparation to hand?", picks...)}}
		},
	})

	// Quake: after a minion schemes, exhaust to deal 2 to it.
	engine.RegisterBehavior("08012", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			st, ok := msg.(engine.SchemeThreat)
			a := g.Allies[e.EID()]
			if !ok || a == nil || a.Exhausted {
				return nil
			}
			if mn := g.Minions[st.Source]; mn == nil {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: a.Owner, Question: engine.Ask(
				"Exhaust Quake → deal 2 damage to the scheming minion?",
				engine.Choice{ID: "use", Label: "Exhaust Quake (2 damage)", Kind: engine.ChoiceAbility, SourceID: e.EID(), CardCode: "08012"}.
					Msgs(engine.ExhaustEntity{ID: e.EID()},
						engine.DamageEntity{Target: st.Source, Damage: 2, Source: a.Owner}),
				engine.Choice{ID: "skip", Label: "Skip", Kind: engine.ChoicePass},
			)}}
		},
	})

	// Stealth Strike: 4 damage (the defeat-rider threat removal is
	// approximated away).
	engine.RegisterBehavior("08013", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy("Stealth Strike: deal 4 damage to which enemy?",
			func(g *engine.Game, e engine.Entity) (int, []engine.Message) { return 4, nil }),
	})

	// Power of Justice reprint + core reprints.
	engine.RegisterBehavior("08014", &engine.Behavior{})
	if b := engine.LookupBehavior("01063"); b != nil {
		engine.RegisterBehavior("08015", b)
	}
	if b := engine.LookupBehavior("01064"); b != nil {
		engine.RegisterBehavior("08016", b)
	}

	// Counterintelligence: threat prevention — no threat window; approx.
	engine.RegisterBehavior("08017", &engine.Behavior{})

	// Spycraft: reveal cancel — no reveal window; approximated.
	engine.RegisterBehavior("08018", &engine.Behavior{})

	// Nick Fury: on play choose one; discarded at the round's end.
	engine.RegisterBehavior("08019", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(
				"Nick Fury: choose one",
				engine.Choice{ID: "threat", Label: "Remove 2 threat from a scheme", Kind: engine.ChoiceLabel}.
					WithThen(engine.Ask("Remove 2 threat from which scheme?", schemePicks(g, 2, p.ID)...)),
				engine.Choice{ID: "draw", Label: "Draw 3 cards", Kind: engine.ChoiceLabel}.
					Msgs(engine.DrawCards{Player: p.ID, N: 3}),
				engine.Choice{ID: "damage", Label: "Deal 4 damage to an enemy", Kind: engine.ChoiceLabel}.
					WithThen(engine.Ask("Deal 4 damage to which enemy?", cardutil.EnemyChoices(g, 4, p.ID,
						func(t engine.EntityID) []engine.Message {
							return []engine.Message{engine.DamageEntity{Target: t, Damage: 4, Source: p.ID}}
						})...)),
			)}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.BeginRound); ok {
				if a := g.Allies[e.EID()]; a != nil {
					return []engine.Message{engine.DiscardControlled{Player: a.Owner, ID: e.EID()}}
				}
			}
			return nil
		},
	})

	// Basic resources + Quincarrier (wild; trait gate approximated).
	engine.RegisterBehavior("08020", &engine.Behavior{})
	engine.RegisterBehavior("08021", &engine.Behavior{})
	engine.RegisterBehavior("08022", &engine.Behavior{})
	engine.RegisterBehavior("08023", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "wild"},
	})

	// Target Acquired: boost-ability cancel — no window; approximated.
	engine.RegisterBehavior("08024", &engine.Behavior{})

	// Burn Notice: exhaust to remove, or discard the highest-cost
	// Preparation (surge otherwise).
	engine.RegisterBehavior("08025", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var picks []engine.Choice
			if !p.Exhausted {
				picks = append(picks, engine.Choice{ID: "exhaust", Label: "Exhaust Natasha → remove from the game", Kind: engine.ChoiceLabel}.
					Msgs(engine.ExhaustEntity{ID: p.ID},
						engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}))
			}
			best, bestCost := "", -1
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil && u.EDef().HasTrait("preparation") {
					if c := deref(u.EDef().Cost, 0); c > bestCost {
						best, bestCost = string(id), c
					}
				}
			}
			if best != "" {
				picks = append(picks, engine.Choice{ID: "discard", Label: "Discard the highest-cost Preparation", Kind: engine.ChoiceLabel}.
					Msgs(engine.DiscardControlled{Player: p.ID, ID: engine.EntityID(best)},
						engine.ObligationResolve{Player: p.ID, Card: card}))
			}
			if len(picks) == 0 {
				return []engine.Message{
					engine.RevealNextEncounter{Player: p.ID},
					engine.ObligationResolve{Player: p.ID, Card: card},
				}
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask("Burn Notice:", picks...)}}
		},
	})

	// Taskmaster: +1 SCH/+1 ATK per upgrade the engaged player controls
	// (synced on relevant messages); boost grants the same to the
	// villain for this activation.
	engine.RegisterBehavior("08026", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return nil
			}
			switch msg.(type) {
			case engine.PlayCard, engine.DiscardControlled, engine.AttachUpgrade, engine.MinionActivates:
				p := g.Player(mn.EngagedWith)
				n := 0
				if p != nil {
					n = min(len(p.Upgrades), 5)
				}
				baseS, baseA := printed(mn.EDef(), "scheme"), printed(mn.EDef(), "attack")
				mn.SchemeVal = baseS + n
				mn.AttackVal = baseA + n
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			n := 0
			for _, p := range g.Players {
				n = max(n, min(len(p.Upgrades), 5))
			}
			if n > 0 {
				return []engine.Message{engine.BoostActivation{Enemy: boostEnemy(g), N: n}}
			}
			return nil
		},
	})

	// Killer for Hire: per-hero threat.
	engine.RegisterBehavior("08027", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.SchemeThreat{Scheme: e.EID(), N: len(g.Players), Source: e.EID()}}
		},
	})

	// Hydra Mercenary: Guard only.
	engine.RegisterBehavior("08028", &engine.Behavior{})

	// Deadly Shot: form-dependent.
	engine.RegisterBehavior("08029", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var picks []engine.Choice
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil {
					picks = append(picks, engine.Choice{Label: "Discard " + u.EDef().Name, Kind: engine.ChoiceCard, CardCode: u.Code}.
						Msgs(engine.DiscardControlled{Player: p.ID, ID: id}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			rider := []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 1, Source: t.ID}}
			if !p.IsHero() {
				rider = nil
				if g.MainScheme != nil {
					rider = []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: t.ID}}
				}
			}
			ask := engine.Ask("Deadly Shot: discard which upgrade?", picks...)
			if len(rider) > 0 {
				ask = engine.Ask("Deadly Shot: discard which upgrade?", append(picks, engine.Choice{
					ID: "skip", Label: "Skip discard", Kind: engine.ChoicePass,
				}.Msgs(rider...))...)
				return []engine.Message{engine.AskQuestion{Player: p.ID, Question: ask}}
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: ask}}
		},
	})

	// Counterattack: after taking enemy damage, strike back equally.
	engine.RegisterBehavior("08030", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.DamageEntity)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil || d.Target != u.Owner || d.Damage <= 0 {
				return nil
			}
			if !(d.Source.Is(engine.KindVillain) || d.Source.Is(engine.KindMinion)) {
				return nil
			}
			return []engine.Message{
				engine.DiscardControlled{Player: u.Owner, ID: e.EID()},
				engine.DamageEntity{Target: d.Source, Damage: d.Damage, Source: u.Owner},
			}
		},
	})

	// Rapid Response: after an ally is defeated, return it to play with
	// 1 damage.
	engine.RegisterBehavior("08031", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.AllyDefeated)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil {
				return nil
			}
			a := g.Allies[w.AllyID]
			if a == nil || a.Owner != u.Owner {
				return nil
			}
			code := a.Code
			return []engine.Message{engine.AskQuestion{Player: u.Owner, Question: engine.Ask(
				"Discard Rapid Response → return "+a.EDef().Name+" to play with 1 damage?",
				engine.Choice{ID: "use", Label: "Use Rapid Response", Kind: engine.ChoiceAbility, SourceID: e.EID(), CardCode: "08031"}.
					Msgs(engine.DiscardControlled{Player: u.Owner, ID: e.EID()},
						engine.RapidReturn{Player: u.Owner, Code: code}),
				engine.Choice{ID: "skip", Label: "Skip", Kind: engine.ChoicePass},
			)}}
		},
	})

	// Defensive Stance: discard to prevent 3 damage.
	engine.RegisterBehavior("08032", &engine.Behavior{
		DamagePrevention: func(g *engine.Game, u *engine.Upgrade, p *engine.Player, n int) (int, int) {
			g.Delete(u.ID)
			p.Discard = append(p.Discard, engine.Card{ID: g.NextCardID(), Code: u.Code, Owner: p.ID})
			g.Logf("%s discards Defensive Stance to prevent 3 damage", p.Name)
			return min(3, n), 0
		},
	})

	// Espionage: surge-cancel draw — no surge window; approximated.
	engine.RegisterBehavior("08033", &engine.Behavior{})
}

// ---- helpers ----

func schemePicks(g *engine.Game, n int, pid engine.PlayerID) []engine.Choice {
	return cardutil.SchemeChoices(g, func(s engine.EntityID) []engine.Message {
		return []engine.Message{engine.ThwartScheme{Scheme: s, N: n, Source: pid}}
	})
}

func deref(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}

func printed(def *data.CardDef, stat string) int {
	switch stat {
	case "scheme":
		return deref(def.Scheme, 0)
	case "attack":
		return deref(def.Attack, 0)
	}
	return 0
}

func boostEnemy(g *engine.Game) engine.EntityID {
	for id := range g.Villains {
		return id
	}
	for id := range g.Minions {
		return id
	}
	return ""
}
