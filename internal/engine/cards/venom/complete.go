// Package venom registers the Venom hero pack: the weapon/restricted
// economy, symbiote bond and the Enraged Symbiote nemesis set.
package venom

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerVenom()
	registerNemesis()
}

// paidOnlyWith reports an event's payment being exclusively one icon.
func paidOnlyWith(ec *engine.EventCard, icon string) bool {
	if len(ec.Paid.Icons) == 0 {
		return false
	}
	for _, ic := range ec.Paid.Icons {
		if ic != icon {
			return false
		}
	}
	return true
}

func registerVenom() {
	// Venom identity: Symbiotic Bond — take 1 damage for a wild resource
	// (once per phase). The extra restricted slot is not enforced.
	engine.RegisterBehavior("20001", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			if g.UsedThisTurn["venom-bond"] {
				return nil
			}
			return []engine.Ability{{
				Label: "Symbiotic Bond — take 1 damage → wild resource", Type: engine.AbilityAction,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					g.UsedThisTurn["venom-bond"] = true
					return []engine.Message{
						engine.DamageEntity{Target: self, Damage: 1, Source: self},
					}
				},
			}}
		},
		// The wild-resource generation rides the payment stub channel:
		// registered as a producer via a support-like hook is not
		// available for identities; the ability above tracks the limit
		// while the resource icon is granted via a marker consumed in
		// payment choices.
		Resource: nil,
	})

	// Behind Enemy Lines: 3 threat (+ confuse on pure-mental payment).
	engine.RegisterBehavior("20002", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			msgs := []engine.Message{engine.AskQuestion{Player: e.EOwner(), Question: engine.Ask(
				"Behind Enemy Lines: remove 3 threat from which scheme?", schemePicks(g, 3, e.EOwner())...)}}
			if ec, ok := e.(*engine.EventCard); ok && paidOnlyWith(ec, "mental") && p != nil {
				msgs = append(msgs, engine.AskQuestion{Player: p.ID, Question: engine.Ask(
					"Confuse which enemy?", enemyStatusChoices(g, p.ID,
						func(id engine.EntityID) engine.Message { return engine.ConfuseEntity{Target: id} })...)})
			}
			return msgs
		},
	})

	// Grasping Tendrils: cancel the attack (defense event; pure-physical
	// adds stun).
	engine.RegisterBehavior("20003", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, ec *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			var extra []engine.Message
			if paidOnlyWith(ec, "physical") {
				extra = append(extra, engine.StunEntity{Target: against})
			}
			return engine.Defends{Defender: p.ID, Against: against, PreventAll: true}, extra, true
		},
	})

	// Locked and Loaded: weapon upgrade search.
	engine.RegisterBehavior("20004", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			seen := map[string]bool{}
			for _, c := range p.Deck {
				def := c.Def()
				if def.Type == "upgrade" && def.HasTrait("weapon") && !seen[c.Code] {
					seen[c.Code] = true
					picks = append(picks, engine.Choice{Label: def.Name, Kind: engine.ChoiceCard, CardCode: def.Code}.
						Msgs(engine.TakeDeckCard{Player: p.ID, CardID: c.ID},
							engine.ShufflePlayerDeck{Player: p.ID}))
				}
			}
			if len(picks) == 0 {
				return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask("Add which weapon to hand?", picks...)}}
		},
	})

	// Run and Gun: ready Venom + each weapon upgrade.
	engine.RegisterBehavior("20005", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			msgs := []engine.Message{engine.ReadyEntity{ID: p.ID}}
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil && u.EDef().HasTrait("weapon") && u.Exhausted {
					msgs = append(msgs, engine.ReadyEntity{ID: id})
				}
			}
			return msgs
		},
	})

	// Savage Attack: 5 damage (pure-energy overkill skipped).
	engine.RegisterBehavior("20006", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy("Savage Attack: deal 5 damage to which enemy?",
			func(g *engine.Game, e engine.Entity) (int, []engine.Message) { return 5, nil }),
	})

	// Project Rebirth 2.0: draw 1 or heal 3.
	engine.RegisterBehavior("20007", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Exhaust Project Rebirth 2.0 → draw 1 or heal 3", Type: engine.AbilityAction,
				Exhaust: true, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.EOwner())
					if p == nil {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(
						"Project Rebirth 2.0:",
						engine.Choice{ID: "draw", Label: "Draw 1 card", Kind: engine.ChoiceLabel}.
							Msgs(engine.DrawCards{Player: p.ID, N: 1}),
						engine.Choice{ID: "heal", Label: "Heal 3 damage", Kind: engine.ChoiceLabel}.
							Msgs(engine.HealEntity{Target: p.ID, N: 3}),
					)}}
				},
			}}
		},
	})

	// Multi-Gun: exhaust → 2 damage / splash 1 / 2 threat.
	engine.RegisterBehavior("20008", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Exhaust Multi-Gun → 2 damage, splash 1, or 2 threat", Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					p := g.Player(u.EOwner())
					if p == nil {
						return nil
					}
					var opts []engine.Choice
					for _, id := range cardutil.SortedEnemyIDs(g) {
						enemy := g.Entity(id)
						if enemy != nil {
							opts = append(opts, engine.Choice{Label: "2 damage — " + cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget, SourceID: id, CardCode: enemy.ECode()}.
								Msgs(engine.DamageEntity{Target: id, Damage: 2, Source: p.ID}))
						}
					}
					for _, q := range g.Players {
						q := q
						var msgs []engine.Message
						for _, mn := range g.Minions {
							if mn.EngagedWith == q.ID {
								msgs = append(msgs, engine.DamageEntity{Target: mn.ID, Damage: 1, Source: p.ID})
							}
						}
						if len(msgs) > 0 {
							opts = append(opts, engine.Choice{Label: "Splash 1 — " + q.Name + "'s minions", Kind: engine.ChoiceTarget, SourceID: q.ID}.
								Msgs(msgs...))
						}
					}
					for _, sid := range g.Schemes() {
						s := g.Entity(sid)
						opts = append(opts, engine.Choice{Label: "2 threat — " + s.EDef().Name, Kind: engine.ChoiceTarget, SourceID: sid, CardCode: s.ECode()}.
							Msgs(engine.ThwartScheme{Scheme: sid, N: 2, Source: p.ID}))
					}
					if len(opts) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID,
						Question: engine.Ask("Multi-Gun: choose one", opts...)}}
				},
			}}
		},
	})

	// Spider-Sense: draw when the villain initiates an attack against
	// you (rides the AskAttack reaction).
	engine.RegisterBehavior("20009", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			aa, ok := msg.(engine.AskAttack)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil || aa.Player != u.Owner || g.Villains[aa.Enemy] == nil {
				return nil
			}
			return []engine.Message{engine.DrawCards{Player: u.Owner, N: 1}}
		},
	})

	// Venom's Pistol: +1 to basic powers — per-use window approximated to
	// +1 ATK/THW/DEF until end of phase on use.
	engine.RegisterBehavior("20010", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Exhaust Venom's Pistol → +1 THW/ATK/DEF this phase", Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					if u == nil {
						return nil
					}
					return []engine.Message{engine.ApplyStatBonus{Target: u.Owner, THW: 1, ATK: 1, DEF: 1}}
				},
			}}
		},
	})

	// Jack Flag: ammo counters on thwart; spend for 2 damage.
	engine.RegisterBehavior("20011", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.AllyThwartWindow)
			if !ok || w.Ally != e.EID() {
				return nil
			}
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 1}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			a := g.Allies[e.EID()]
			if a == nil || a.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: "Exhaust Jack Flag + counter → deal 2 damage", Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return append([]engine.Message{engine.AddEntityCounter{ID: self, N: -1}},
						cardutil.ChooseEnemy("Jack Flag: deal 2 damage to which enemy?",
							func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 2, nil })(
							g, g.Entity(self))...)
				},
			}}
		},
	})

	// Scare Tactic: 3 damage to a confused enemy.
	engine.RegisterBehavior("20012", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var picks []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				enemy := g.Entity(id)
				if enemy == nil {
					continue
				}
				if mn := g.Minions[id]; mn != nil && !mn.Confused {
					continue
				}
				if v := g.Villains[id]; v != nil && !v.Confused {
					continue
				}
				picks = append(picks, engine.Choice{Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget, SourceID: id, CardCode: enemy.ECode()}.
					Msgs(engine.DamageEntity{Target: id, Damage: 3, Source: e.EOwner()}))
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(),
				Question: engine.Ask("Scare Tactic: damage which confused enemy?", picks...)}}
		},
	})

	// Making an Entrance: +2 THW; heal 2 on a full clear (rider on the
	// thwart window).
	engine.RegisterBehavior("20013", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.ApplyStatBonus{Target: e.EOwner(), THW: 2}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowAfterThwarted)
			ec, isEv := e.(*engine.EventCard)
			if !ok || !isEv || w.Player != ec.Owner {
				return nil
			}
			if s := g.SideSchemes[w.Scheme]; s == nil || s.Threat != 0 {
				return nil
			}
			return []engine.Message{engine.HealEntity{Target: ec.Owner, N: 2}}
		},
	})

	// The Power of Justice reprint + basics.
	engine.RegisterBehavior("20014", &engine.Behavior{})
	engine.RegisterBehavior("20017", &engine.Behavior{})
	engine.RegisterBehavior("20018", &engine.Behavior{})
	engine.RegisterBehavior("20019", &engine.Behavior{})

	// Sonic Rifle: 2 charges; confuse or 3 damage to the confused.
	engine.RegisterBehavior("20015", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 2}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			u := g.Upgrades[e.EID()]
			if u == nil || u.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: "Exhaust Sonic Rifle + counter → confuse (3 damage if confused)", Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					var picks []engine.Choice
					for _, id := range cardutil.SortedEnemyIDs(g) {
						enemy := g.Entity(id)
						if enemy == nil {
							continue
						}
						var msg engine.Message = engine.ConfuseEntity{Target: id}
						label := "Confuse"
						if mn := g.Minions[id]; mn != nil && mn.Confused {
							msg = engine.DamageEntity{Target: id, Damage: 3, Source: u.Owner}
							label = "3 damage"
						}
						if v := g.Villains[id]; v != nil && v.Confused {
							msg = engine.DamageEntity{Target: id, Damage: 3, Source: u.Owner}
							label = "3 damage"
						}
						picks = append(picks, engine.Choice{Label: label + " — " + cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget, SourceID: id, CardCode: enemy.ECode()}.
							Msgs(engine.AddEntityCounter{ID: self, N: -1}, msg))
					}
					if len(picks) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: u.Owner,
						Question: engine.Ask("Sonic Rifle: target which enemy?", picks...)}}
				},
			}}
		},
	})

	// Star-Lord (ally): facedown encounter card on entry.
	engine.RegisterBehavior("20016", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.DealEncounterToPlayer{Player: e.EOwner()}}
		},
	})

	// Resourceful reprint: alias hlk 10032.
	if b := engine.LookupBehavior("10032"); b != nil {
		engine.RegisterBehavior("20020", b)
	}

	// Side Holster: extra restricted slot (unenforced); marker behavior.
	engine.RegisterBehavior("20021", &engine.Behavior{})

	// Plasma Pistol: 3 charges, 1 damage each.
	engine.RegisterBehavior("20022", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 3}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			u := g.Upgrades[e.EID()]
			if u == nil || u.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: "Exhaust Plasma Pistol + counter → deal 1 damage", Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return append([]engine.Message{engine.AddEntityCounter{ID: self, N: -1}},
						cardutil.ChooseEnemy("Plasma Pistol: deal 1 damage to which enemy?",
							func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 1, nil })(
							g, g.Entity(self))...)
				},
			}}
		},
	})

	// Struggle for Control obligation: exhaust + 2 damage, or surge in
	// the Symbiote.
	engine.RegisterBehavior("20023", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var picks []engine.Choice
			if !p.Exhausted {
				picks = append(picks, engine.Choice{ID: "exhaust", Label: "Exhaust Flash + take 2 damage", Kind: engine.ChoiceLabel}.
					Msgs(engine.ExhaustEntity{ID: p.ID},
						engine.DamageEntity{Target: p.ID, Damage: 2, Source: engine.EntityID("20023")},
						engine.ObligationResolve{Player: p.ID, Card: card}))
			}
			picks = append(picks, engine.Choice{ID: "symbiote", Label: "Put an Enraged Symbiote into play", Kind: engine.ChoiceLabel}.
				Msgs(engine.SpawnSymbiote{},
					engine.ObligationResolve{Player: p.ID, Card: card}))
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask("Struggle for Control:", picks...)}}
		},
	})

	// Fusillade: exhaust a weapon → 5 damage.
	engine.RegisterBehavior("20026", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var weapons []engine.Choice
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil && u.EDef().HasTrait("weapon") && !u.Exhausted {
					weapons = append(weapons, engine.Choice{Label: "Exhaust " + u.EDef().Name, Kind: engine.ChoiceCard, CardCode: u.Code}.
						Msgs(engine.ExhaustEntity{ID: id}))
				}
			}
			if len(weapons) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(
				"Fusillade: exhaust which weapon?", weapons...)},
				engine.AskQuestion{Player: p.ID, Question: engine.Ask(
					"Fusillade: deal 5 damage to which enemy?", cardutil.EnemyChoices(g, 5, p.ID,
						func(t engine.EntityID) []engine.Message {
							return []engine.Message{engine.DamageEntity{Target: t, Damage: 5, Source: p.ID}}
						})...)}}
		},
	})

	// "Welcome Aboard": next ally this phase costs 2 less (all players).
	engine.RegisterBehavior("20027", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			msgs := []engine.Message{}
			for _, q := range g.Players {
				msgs = append(msgs, engine.CostDiscountApply{Player: q.ID, Amount: 2})
			}
			return msgs
		},
	})

	// Shake it Off: tough on a damaged guardian.
	engine.RegisterBehavior("20028", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			for _, q := range g.Players {
				q := q
				if q.Damage > 0 {
					isG := g.EntityHasTrait(q.ID, "guardian")
					hasG := false
					for _, id := range q.Allies {
						if a := g.Allies[id]; a != nil && a.EDef().HasTrait("guardian") {
							hasG = true
						}
					}
					if isG || hasG {
						picks = append(picks, engine.Choice{Label: q.Name, Kind: engine.ChoiceTarget, SourceID: q.ID}.
							Msgs(engine.ToughEntity{Target: q.ID}))
					}
				}
				for _, id := range q.Allies {
					a := g.Allies[id]
					if a != nil && a.Damage > 0 && a.EDef().HasTrait("guardian") {
						picks = append(picks, engine.Choice{Label: a.EDef().Name, Kind: engine.ChoiceTarget, SourceID: a.ID, CardCode: a.Code}.
							Msgs(engine.ToughEntity{Target: a.ID}))
					}
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask("Shake it Off: tough on which damaged guardian?", picks...)}}
		},
	})

	// Crew Quarters: heal 1 from an alter-ego.
	engine.RegisterBehavior("20029", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Exhaust Crew Quarters → heal 1 from an alter-ego", Type: engine.AbilityAction,
				Exhaust: true, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.EOwner())
					if p == nil {
						return nil
					}
					var picks []engine.Choice
					for _, q := range g.Players {
						if q.Damage > 0 {
							picks = append(picks, engine.Choice{Label: q.Name + " (alter-ego)", Kind: engine.ChoiceTarget, SourceID: q.ID}.
								Msgs(engine.HealEntity{Target: q.ID, N: 1}))
						}
					}
					if len(picks) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID,
						Question: engine.Ask("Heal which alter-ego?", picks...)}}
				},
			}}
		},
	})
}

func registerNemesis() {
	// Klyntar Frenzy: threat lock while a Symbiote enemy lives (wired
	// engine-side in removeThreat via this marker code).
	engine.RegisterBehavior("20024", &engine.Behavior{})

	// Enraged Symbiote: Guard + Patrol printed; boost put-into-play
	// covered engine-side.
	engine.RegisterBehavior("20025", &engine.Behavior{})
}

// ---- helpers ----

func schemePicks(g *engine.Game, n int, pid engine.PlayerID) []engine.Choice {
	return cardutil.SchemeChoices(g, func(s engine.EntityID) []engine.Message {
		return []engine.Message{engine.ThwartScheme{Scheme: s, N: n, Source: pid}}
	})
}

func enemyStatusChoices(g *engine.Game, pid engine.PlayerID, mk func(id engine.EntityID) engine.Message) []engine.Choice {
	var out []engine.Choice
	for _, id := range cardutil.SortedEnemyIDs(g) {
		enemy := g.Entity(id)
		if enemy != nil {
			out = append(out, engine.Choice{Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget, SourceID: id, CardCode: enemy.ECode()}.
				Msgs(mk(id)))
		}
	}
	return out
}
