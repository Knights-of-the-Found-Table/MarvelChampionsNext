// Package valkyrie registers the Valkyrie hero pack: the Death-Glow
// mark-and-hunt economy and the Enchantress nemesis set.
package valkyrie

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerValkyrie()
	registerNemesis()
}

// deathGlowTarget finds the enemy with Death-Glow attached.
func deathGlowTarget(g *engine.Game) engine.EntityID {
	for _, a := range g.Attachments {
		if a.Code[:5] == "25002" && a.Target != "" {
			return a.Target
		}
	}
	return ""
}

func registerValkyrie() {
	// Valkyrie identity: Death Perception — play the set-aside Death-Glow
	// as an action (approximated: create it directly when none is in
	// play).
	engine.RegisterBehavior("25001", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			if deathGlowTarget(g) != "" {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.deathPerceptionAttachDeathGlowToAnEnemy"), Type: engine.AbilityAction, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(self)
					if p == nil {
						return nil
					}
					u := &engine.Upgrade{
						ID:    g.NextEntityID("upgrade"),
						Code:  "25002",
						Owner: p.ID,
					}
					g.Upgrades[u.ID] = u
					p.Upgrades = append(p.Upgrades, u.ID)
					g.TLogf("c.deathGlowEntersPlayAttachedByDeathPerception")
					return []engine.Message{engine.AttachUpgrade{ID: u.ID, Target: engine.EntityID("")}}
				},
			}}
		},
	})

	// Death-Glow: attach to an enemy; on its defeat, set aside (+ ready
	// Valkyrie if she dealt the killing blow).
	engine.RegisterBehavior("25002", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				enemy := g.Entity(id)
				if enemy != nil {
					picks = append(picks, engine.Choice{Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget, SourceID: id, CardCode: enemy.ECode()}.
						Msgs(engine.AttachUpgrade{ID: e.EID(), Target: id}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.attachDeathGlowToWhichEnemy"), picks...)}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			switch d := msg.(type) {
			case engine.VillainDefeated:
				if d.VillainID != u.AttachTo {
					return nil
				}
			case engine.MinionDefeated:
				if d.MinionID != u.AttachTo {
					return nil
				}
			default:
				return nil
			}
			g.Delete(u.ID) // set aside, out of play
			g.TLogf("c.deathGlowIsSetAsideAsItsEnemyIsDefeated")
			return []engine.Message{engine.ReadyEntity{ID: u.Owner}}
		},
	})

	// Annabelle Riggs: top-5 Valkyrie-card search.
	engine.RegisterBehavior("25003", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustAnnabelleRiggsSearchTop5ForAValkyrieCard"), Type: engine.AbilityAction,
				Exhaust: true, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					a := g.Allies[self]
					p := g.Player(a.EOwner())
					if p == nil {
						return nil
					}
					var picks []engine.Choice
					for i := 0; i < 5 && i < len(p.Deck); i++ {
						c := p.Deck[i]
						if c.Def().Code[:2] == "25" && c.Def().Type != "obligation" {
							picks = append(picks, engine.Choice{Label: engine.S(c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code}.
								Msgs(engine.TakeDeckCard{Player: p.ID, CardID: c.ID},
									engine.ShufflePlayerDeck{Player: p.ID}))
						}
					}
					if len(picks) == 0 {
						return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID,
						Question: engine.Ask(engine.Tf("c.addWhichCardToHand"), picks...)}}
				},
			}}
		},
	})

	// Valhalla: after the Death-Glow enemy falls, exhaust → draw + heal.
	engine.RegisterBehavior("25004", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			s := g.Supports[e.EID()]
			if s == nil || s.Exhausted {
				return nil
			}
			tgt := deathGlowTarget(g)
			if tgt == "" {
				// The glow already left play with the defeat; approximate
				// to any enemy defeat this react.
				switch msg.(type) {
				case engine.VillainDefeated, engine.MinionDefeated:
				default:
					return nil
				}
			} else {
				switch d := msg.(type) {
				case engine.VillainDefeated:
					if d.VillainID != tgt {
						return nil
					}
				case engine.MinionDefeated:
					if d.MinionID != tgt {
						return nil
					}
				default:
					return nil
				}
			}
			return []engine.Message{
				engine.ExhaustEntity{ID: e.EID()},
				engine.DrawCards{Player: s.Owner, N: 1},
				engine.HealEntity{Target: s.Owner, N: 1},
			}
		},
	})

	// Valkyrie's Spear / Dragonfang: conditional double bonuses
	// approximated to the base +1 each.
	engine.RegisterBehavior("25005", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{DEF: 1} },
	})
	engine.RegisterBehavior("25006", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{ATK: 1} },
	})

	// Aragorn: +4 HP + Aerial.
	engine.RegisterBehavior("25007", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p != nil {
				p.MaxHP += 4
			}
			return []engine.Message{engine.GrantTrait{Target: e.EOwner(), Trait: "aerial"}}
		},
	})

	// Flight of the Valkyrior: after the glow enemy falls, discard →
	// remove 5 threat.
	engine.RegisterBehavior("25008", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			switch msg.(type) {
			case engine.VillainDefeated, engine.MinionDefeated:
			default:
				return nil
			}
			return []engine.Message{
				engine.DiscardControlled{Player: u.Owner, ID: u.ID},
				engine.AskQuestion{Player: u.Owner, Question: engine.Ask(
					engine.Tf("c.flightOfTheValkyriorRemove5ThreatFromWhichScheme"), schemePicks(g, 5, u.Owner)...)},
			}
		},
	})

	// Visit Valhalla: Valkyrie card from discard to hand.
	engine.RegisterBehavior("25009", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			seen := map[string]bool{}
			for _, c := range p.Discard {
				def := c.Def()
				if def.Code[:2] == "25" && def.Type != "obligation" && !seen[c.Code] {
					seen[c.Code] = true
					picks = append(picks, engine.Choice{Label: engine.S(def.Name), Kind: engine.ChoiceCard, CardCode: def.Code}.
						Msgs(engine.ReturnDiscardCard{Player: p.ID, CardID: c.ID}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.returnWhichValkyrieCardToHand"), picks...)}}
		},
	})

	// Chooser of the Slain: fetch a minion engaged with you, draw 2.
	engine.RegisterBehavior("25010", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for i, c := range g.EncounterDeck {
				if c.Def().Type == "minion" {
					g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
					return []engine.Message{
						engine.RevealEncounterCard{Player: e.EOwner(), Card: c},
						engine.DrawCards{Player: e.EOwner(), N: 2},
					}
				}
			}
			return []engine.Message{engine.DrawCards{Player: e.EOwner(), N: 2}}
		},
	})

	// Shieldmaiden: defend the glow enemy without exhausting, +2 DEF.
	engine.RegisterBehavior("25011", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, ec *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			d := engine.Defends{Defender: p.ID, Against: against, DefBonus: 2}
			if deathGlowTarget(g) == against {
				d.NoExhaust = true
			}
			return d, nil, true
		},
	})

	// Have at Thee!: 7 damage (glow overkill skipped).
	engine.RegisterBehavior("25012", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy(engine.Tf("c.haveAtTheeDeal7DamageToWhichEnemy"),
			func(g *engine.Game, e engine.Entity) (int, []engine.Message) { return 7, nil }),
	})

	// Thor (ally): splash attack rider — multi-minion resolution
	// approximated to nothing extra ( Toughness printed).
	engine.RegisterBehavior("25013", &engine.Behavior{})

	// Throg: tough on entry while engaged.
	engine.RegisterBehavior("25014", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			for _, mn := range g.Minions {
				if mn.EngagedWith == p.ID {
					return []engine.Message{engine.ToughEntity{Target: e.EID()}}
				}
			}
			return nil
		},
	})

	// Angela: reprint of gam's 18011.
	if b := engine.LookupBehavior("18011"); b != nil {
		engine.RegisterBehavior("25015", b)
	}

	// Hall of Heroes reprint: alias thor 06017.
	if b := engine.LookupBehavior("06017"); b != nil {
		engine.RegisterBehavior("25016", b)
	}

	// Combat Training reprint: alias core 01057.
	if b := engine.LookupBehavior("01057"); b != nil {
		engine.RegisterBehavior("25017", b)
	}

	// Quick Strike: ATK damage.
	engine.RegisterBehavior("25018", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			n := 0
			if p != nil {
				n = max(0, p.AttackStat(g))
			}
			if n <= 0 {
				return nil
			}
			return cardutil.ChooseEnemy(engine.Tf("c.quickStrikeDealDamageToWhichEnemy"),
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return n, nil })(g, e)
		},
	})

	// Smash the Problem: exhaust → remove ATK threat.
	engine.RegisterBehavior("25019", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			n := max(0, p.AttackStat(g))
			if n <= 0 {
				return nil
			}
			return []engine.Message{
				engine.ExhaustEntity{ID: p.ID},
				engine.AskQuestion{Player: p.ID, Question: engine.Ask(
					engine.Tf("c.smashTheProblemRemoveThreatFromWhichScheme"), schemePicks(g, n, p.ID)...)},
			}
		},
	})

	// The Best Defense…: DEF substitution approximated as +2 DEF.
	engine.RegisterBehavior("25020", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, ec *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			return engine.Defends{Defender: p.ID, Against: against, DefBonus: 2}, nil, true
		},
	})

	// Audacity: spent-response — plain resource.
	engine.RegisterBehavior("25021", &engine.Behavior{})

	// The Power of Aggression reprint + basics.
	engine.RegisterBehavior("25022", &engine.Behavior{})
	engine.RegisterBehavior("25025", &engine.Behavior{})
	engine.RegisterBehavior("25026", &engine.Behavior{})
	engine.RegisterBehavior("25027", &engine.Behavior{})

	// The Bifrost: search an asgard ally and play it (payment question
	// simplified to free search+play — cost noted).
	engine.RegisterBehavior("25023", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustTheBifrostSearchYourDeckForAnAsgardAlly"), Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.EOwner())
					if p == nil {
						return nil
					}
					var picks []engine.Choice
					for _, c := range p.Deck {
						def := c.Def()
						if def.Type == "ally" && def.HasTrait("asgard") {
							picks = append(picks, engine.Choice{Label: engine.S(def.Name), Kind: engine.ChoiceCard, CardCode: def.Code}.
								Msgs(engine.TakeDeckCard{Player: p.ID, CardID: c.ID},
									engine.PlayCard{Player: p.ID, Card: c, Paid: engine.CostPaid{}},
									engine.ShufflePlayerDeck{Player: p.ID}))
						}
					}
					if len(picks) == 0 {
						return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID,
						Question: engine.Ask(engine.Tf("c.playWhichAsgardAlly"), picks...)}}
				},
			}}
		},
	})

	// Godlike Stamina: heal 2 + clear a status.
	engine.RegisterBehavior("25024", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			msgs := []engine.Message{engine.HealEntity{Target: e.EOwner(), N: 2},
				engine.ClearStun{Target: e.EOwner()}, engine.ClearConfuse{Target: e.EOwner()}}
			return msgs
		},
	})

	// Trouble in Otherworld obligation: energy+mental to remove.
	engine.RegisterBehavior("25028", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(
				engine.Tf("c.troubleInOtherworldSpendEnergyMentalToRemoveIt"),
				engine.Choice{ID: "pay", Label: engine.Tf("c.spendEnergyMental"), Kind: engine.ChoiceLabel}.
					Msgs(engine.ObligationResolve{Player: p.ID, Card: card}).
					WithThen(g.CustomPaymentQuestion(p, 2, engine.S("Spend 1 [energy] and 1 [mental]"),
						map[string]any{"player": p.ID.String(), "abilityIcons": "energy:1 mental:1"})),
				engine.Choice{ID: "keep", Label: engine.Tf("c.keepItInPlay"), Kind: engine.ChoicePass},
			)}}
		},
	})

	// Problem Solvers: exhaust avenger+guardian → combined-THW threat
	// removal (approximated to 4 flat).
	engine.RegisterBehavior("25033", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			n := 4
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(), Question: engine.Ask(
				engine.Tf("c.problemSolversRemoveThreatFromWhichScheme"), schemePicks(g, n, e.EOwner())...)}}
		},
	})

	// Leadership Training reprint: alias nebula's Defensive Training shape.
	engine.RegisterBehavior("25034", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 2}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			if s == nil || s.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustCounterShuffleALeadershipEventFromDiscard"), Type: engine.AbilityAction,
				Exhaust: true, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.EOwner())
					if p == nil {
						return nil
					}
					var picks []engine.Choice
					for _, c := range p.Discard {
						def := c.Def()
						if def.Type == "event" && def.Aspect == "leadership" {
							picks = append(picks, engine.Choice{Label: engine.S(def.Name), Kind: engine.ChoiceCard, CardCode: def.Code}.
								Msgs(engine.ShuffleIntoDeck{Player: p.ID, CardID: c.ID}))
						}
					}
					if len(picks) == 0 {
						return nil
					}
					return append([]engine.Message{engine.AddEntityCounter{ID: self, N: -1}},
						engine.AskQuestion{Player: p.ID, Question: engine.Ask(
							engine.Tf("c.shuffleWhichLeadershipEventIntoYourDeck"), picks...)})
				},
			}}
		},
	})

	// Anticipation: discard on engaging a minion → ready.
	engine.RegisterBehavior("25035", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionEntersPlay)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil || m.Player != u.Owner {
				return nil
			}
			return []engine.Message{
				engine.DiscardControlled{Player: u.Owner, ID: u.ID},
				engine.ReadyEntity{ID: u.Owner},
			}
		},
	})

	// Cosmic Alliance: ready an avenger and a guardian character.
	engine.RegisterBehavior("25036", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var picks []engine.Choice
			for _, q := range g.Players {
				for _, id := range q.Allies {
					a := g.Allies[id]
					if a != nil && a.Exhausted && (a.EDef().HasTrait("avenger") || a.EDef().HasTrait("guardian")) {
						picks = append(picks, engine.Choice{Label: engine.S(a.EDef().Name), Kind: engine.ChoiceTarget, SourceID: a.ID, CardCode: a.Code}.
							Msgs(engine.ReadyEntity{ID: a.ID}))
					}
				}
				if q.Exhausted && (g.EntityHasTrait(q.ID, "avenger") || g.EntityHasTrait(q.ID, "guardian")) {
					picks = append(picks, engine.Choice{Label: engine.S(q.Name + " (identity)"), Kind: engine.ChoiceTarget, SourceID: q.ID}.
						Msgs(engine.ReadyEntity{ID: q.ID}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			q := engine.AskN(engine.Tf("c.cosmicAllianceReadyWhichCharacters"), 2, picks...)
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(), Question: q}}
		},
	})
}

func registerNemesis() {
	// Enchantress: fetch Seduced on reveal.
	engine.RegisterBehavior("25029", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return nil
			}
			p := g.Player(mn.EngagedWith)
			if p == nil {
				return nil
			}
			for i, c := range g.EncounterDeck {
				if c.Code[:5] == "25032" {
					g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
			}
			for i, c := range g.EncounterDiscard {
				if c.Code[:5] == "25032" {
					g.EncounterDiscard = append(g.EncounterDiscard[:i], g.EncounterDiscard[i+1:]...)
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
			}
			return nil
		},
	})

	// Powerful Enchantments: Hinder 1 per hero; the no-discard lock is
	// not enforced.
	engine.RegisterBehavior("25030", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.SchemeThreat{Scheme: e.EID(), N: len(g.Players), Source: e.EID()}}
		},
	})

	// Beguiled: attach to the highest-cost ally; converts it to an
	// enthralled minion — approximate to discarding the ally.
	engine.RegisterBehavior("25031", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			var best engine.EntityID
			bestCost := -1
			for _, q := range g.Players {
				for _, id := range q.Allies {
					a := g.Allies[id]
					if a == nil {
						continue
					}
					taken := false
					for _, at := range g.Attachments {
						if at.Code[:5] == "25031" && at.Target == id {
							taken = true
						}
					}
					if taken {
						continue
					}
					if c := deref(a.EDef().Cost, 0); c > bestCost {
						best, bestCost = id, c
					}
				}
			}
			if best == "" {
				g.Delete(t.ID)
				g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: t.Code})
				return []engine.Message{engine.RevealNextEncounter{Player: cardutil.FirstPlayerID(g)}}
			}
			t.Target = best
			// Enthrall: remove the ally from the owner's control.
			if a := g.Allies[best]; a != nil {
				owner := g.Player(a.Owner)
				g.Delete(best)
				if owner != nil {
					owner.Discard = append(owner.Discard, engine.Card{ID: g.NextCardID(), Code: a.Code, Owner: a.Owner})
				}
			}
			g.TLogf("c.beguiledEnthrallsTheHighestCostAlly")
			return nil
		},
	})

	// Seduced: attach identity; no basic attacks / attack events —
	// restriction approximated away; removal via icon payment.
	engine.RegisterBehavior("25032", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if p := g.Player(engine.PlayerID(target)); p != nil {
				t.Target = p.ID
			}
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.spendEnergyMentalDiscardSeduced"), Type: engine.AbilityAction,
				Cost: 2, CostIcons: "energy:1 mental:1", AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					a := g.Attachments[self]
					if a == nil {
						return nil
					}
					g.Delete(self)
					g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: a.Code})
					return nil
				},
			}}
		},
	})
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
