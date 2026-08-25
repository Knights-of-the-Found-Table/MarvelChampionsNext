package nextevolution

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// hopeSummers spawns Hope Summers (40130) under the first player with
// stats mirroring their hero.
func hopeSummers(g *engine.Game) {
	p := g.Player(cardutil.FirstPlayerID(g))
	if p == nil {
		return
	}
	c := engine.Card{ID: g.NextCardID(), Code: "40130", Owner: p.ID}
	spawnAllyFor(g, p, c)
	// Mirror the hero's THW/ATK.
	for _, id := range p.Allies {
		if a := g.Allies[id]; a != nil && engine.BaseCodeOf(a.Code) == "40130" {
			a.ThwartVal = p.ThwartStat(g)
			a.AttackVal = p.AttackStat(g)
			g.TLogf("c.hopeSummersMirrorsThwAtk", p.HeroDef(), a.ThwartVal, a.AttackVal)
		}
	}
}

// drawFromSetAside removes the first set-aside card with a base code.
func drawFromSetAside(g *engine.Game, base string) (engine.Card, bool) {
	for i, c := range g.SetAside {
		if data.BaseCode(c.Code) == base {
			g.SetAside = append(g.SetAside[:i:i], g.SetAside[i+1:]...)
			return c, true
		}
	}
	return engine.Card{}, false
}

// tuckFromEncounter removes every card with a base code from the
// encounter deck into the set-aside area.
func tuckFromEncounter(g *engine.Game, base string) int {
	n := 0
	var kept engine.CardList
	for _, c := range g.EncounterDeck {
		if data.BaseCode(c.Code) == base {
			g.SetAside = append(g.SetAside, c)
			n++
			continue
		}
		kept = append(kept, c)
	}
	g.EncounterDeck = kept
	return n
}

func registerCampaign() {
	// 40190a Assemble the Team: flip → each player plays an ally from
	// their deck for free (auto-picked).
	engine.RegisterBehavior("40190", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			for _, p := range g.Players {
				searchDeckDiscard(g, p, func(d *data.CardDef) bool { return d.Type == "ally" },
					func(p *engine.Player, c engine.Card) { spawnAllyFor(g, p, c) })
			}
			g.TLogf("c.teamAssembled")
			return nil
		},
	})

	// 40191a Establish Safehouse: flip → the players gain the Safehouse
	// support.
	engine.RegisterBehavior("40191", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			p := g.Player(cardutil.FirstPlayerID(g))
			if p == nil {
				return nil
			}
			spawnSupportFor(g, p, engine.Card{ID: g.NextCardID(), Code: "40197", Owner: p.ID})
			g.TLogf("c.safehouseEstablished")
			return nil
		},
	})

	// 40192a Gear Up: flip → each player gains a Pouches.
	engine.RegisterBehavior("40192", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			for _, p := range g.Players {
				p.Hand = append(p.Hand, engine.Card{ID: g.NextCardID(), Code: "40196", Owner: p.ID})
			}
			g.TLogf("c.gearedUpEveryoneGainsAPouches")
			return nil
		},
	})

	// 40193a Mission Prep: flip → each player plays an upgrade from their
	// deck for free (auto-picked).
	engine.RegisterBehavior("40193", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			for _, p := range g.Players {
				searchDeckDiscard(g, p, func(d *data.CardDef) bool { return d.Type == "upgrade" },
					func(p *engine.Player, c engine.Card) { spawnUpgradeFor(g, p, c) })
			}
			g.TLogf("c.missionPrepped")
			return nil
		},
	})

	// 40194a Practice Maneuvers: the permanent event discount is
	// approximated away (log only).
	engine.RegisterBehavior("40194", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			g.TLogf("c.practicedManeuversPermanentEventDiscountNotModeled")
			return nil
		},
	})

	// 40195a Prepare Defenses: the permanent +1 DEF / retaliate 1 is
	// approximated away (log only).
	engine.RegisterBehavior("40195", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			g.TLogf("c.preparedDefenses1DefRetaliate1NotModeled")
			return nil
		},
	})

	// 40196 Pouches: textless resource.
	engine.RegisterBehavior("40196", &engine.Behavior{})

	// 40197 Safehouse: alter-ego action — heal 2 or draw 1.
	engine.RegisterBehavior("40197", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.safehouseHeal2DamageOrDraw1Card"), Type: engine.AbilityAction,
				AlterEgoOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.Owner)
					if p == nil {
						return nil
					}
					return []engine.Message{engine.AskQuestion{
						Player: p.ID,
						Question: engine.Ask(engine.Tf("c.safehouseChoose"),
							engine.Choice{ID: "heal", Label: engine.Tf("c.heal2DamageFromYourIdentity"), Kind: engine.ChoiceLabel}.
								Msgs(engine.HealEntity{Target: p.ID, N: 2}),
							engine.Choice{ID: "draw", Label: engine.Tf("c.draw1Card"), Kind: engine.ChoiceLabel}.
								Msgs(engine.DrawCards{Player: p.ID, N: 1}),
						),
					}}
				},
			}}
		},
	})

	// 40198 Lady Mastermind: damage = highest-cost event in hand.
	engine.RegisterBehavior("40198", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return nil
			}
			p := g.Player(mn.EngagedWith)
			if p == nil {
				return nil
			}
			best := 0
			for _, c := range p.Hand {
				if c.Def().Type == "event" {
					if n := cardutil.Cost(c.Def()); n > best {
						best = n
					}
				}
			}
			if best > 0 {
				return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: best, Source: mn.ID}}
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if p := g.Player(cardutil.FirstPlayerID(g)); p != nil {
				for _, c := range p.Hand {
					if c.Def().Type == "event" {
						return []engine.Message{engine.DiscardCards{Player: p.ID, Cards: engine.CardList{c}}}
					}
				}
			}
			return nil
		},
	})

	// 40199 Malice: on defeat, possess the costliest non-PSIONIC ally
	// (converted into a minion).
	engine.RegisterBehavior("40199", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionDefeated)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			best, pick := -1, engine.EntityID("")
			var owner engine.PlayerID
			for _, p := range g.Players {
				for _, id := range p.Allies {
					a := g.Allies[id]
					if a == nil || a.EDef().HasTrait("Psionic") {
						continue
					}
					if c := cardutil.Cost(a.EDef()); c > best {
						best, pick, owner = c, id, p.ID
					}
				}
			}
			if pick == "" {
				return nil
			}
			a := g.Allies[pick]
			hp, atk, sch := a.MaxHP-a.Damage, a.AttackVal, a.ThwartVal
			code := a.Code
			g.Delete(pick)
			mn := &engine.Minion{
				ID:        g.NextEntityID(engine.KindMinion),
				Code:      code,
				MaxHP:     hp,
				AttackVal: atk,
				SchemeVal: sch,
			}
			g.Minions[mn.ID] = mn
			mn.EngagedWith = owner
			g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: "40199"})
			g.TLogf("c.malicePossessesTreatedAsAPossessedMinion", engine.DB.MustLookup(code))
			return nil
		},
	})

	// 40200 Scrambler: discard an upgrade you control.
	engine.RegisterBehavior("40200", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return nil
			}
			p := g.Player(mn.EngagedWith)
			if p == nil || len(p.Upgrades) == 0 {
				return nil
			}
			return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Upgrades[0]}}
		},
	})

	// 40201 Vanisher: return the costliest support to hand.
	engine.RegisterBehavior("40201", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return vanishBounce(g, e)
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			// Boost: return the support you control with the highest cost
			// (first player approximation).
			p := g.Player(cardutil.FirstPlayerID(g))
			if p == nil {
				return nil
			}
			best, pick := -1, engine.EntityID("")
			for _, id := range p.Supports {
				if s := g.Supports[id]; s != nil {
					if c := cardutil.Cost(s.EDef()); c > best {
						best, pick = c, id
					}
				}
			}
			if pick != "" {
				bounceToHand(g, p, "support", pick)
			}
			return nil
		},
	})

	// 40202 Under Pressure: surge (keyword); boost rider approximated.
	engine.RegisterBehavior("40202", &engine.Behavior{
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if id := boostEnemy(g); id != "" {
				return []engine.Message{engine.DealBoost{Enemy: id}}
			}
			return nil
		},
	})

	// 40203 Overburdened: discard a resource or take 2 damage.
	engine.RegisterBehavior("40203", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			return overburdenChoice(g, p)
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if p := g.Player(cardutil.FirstPlayerID(g)); p != nil {
				return overburdenChoice(g, p)
			}
			return nil
		},
	})

	// 40204 Hope Summers (player deck copy): gains the identity's traits;
	// on play, fetch a SUPERPOWER card.
	engine.RegisterBehavior("40204", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			a := g.Allies[e.EID()]
			p := g.Player(a.Owner)
			if p == nil {
				return nil
			}
			// Grant the identity's traits.
			hero := p.HeroDef()
			a.ExtraTraits = append(a.ExtraTraits, hero.Traits...)
			searchDeckDiscard(g, p, func(d *data.CardDef) bool { return d.HasTrait("Superpower") },
				func(p *engine.Player, c engine.Card) {
					p.Hand = append(p.Hand, c)
					g.TLogf("c.addsToHand", p.Name, c)
				})
			return nil
		},
	})
}

func overburdenChoice(g *engine.Game, p *engine.Player) []engine.Message {
	var resource engine.Card
	found := false
	for _, c := range p.Hand {
		if c.Def().Type == "resource" {
			resource, found = c, true
			break
		}
	}
	choices := []engine.Choice{
		engine.Choice{ID: "dmg", Label: engine.Tf("c.take2Damage"), Kind: engine.ChoiceLabel}.
			Msgs(engine.DamageEntity{Target: p.ID, Damage: 2, Source: engine.EntityID("")}),
	}
	if found {
		choices = append(choices, engine.Choice{
			ID: "disc", Label: engine.S("Discard " + resource.Def().Name), Kind: engine.ChoiceLabel,
		}.Msgs(engine.DiscardCards{Player: p.ID, Cards: engine.CardList{resource}}))
	}
	return []engine.Message{engine.AskQuestion{
		Player:   p.ID,
		Question: engine.Ask(engine.Tf("c.overburdenedChoose"), choices...),
	}}
}

func vanishBounce(g *engine.Game, e engine.Entity) []engine.Message {
	mn := g.Minions[e.EID()]
	if mn == nil || mn.EngagedWith == "" {
		return nil
	}
	p := g.Player(mn.EngagedWith)
	if p == nil {
		return nil
	}
	best, pick := -1, engine.EntityID("")
	for _, id := range p.Supports {
		if s := g.Supports[id]; s != nil {
			if c := cardutil.Cost(s.EDef()); c > best {
				best, pick = c, id
			}
		}
	}
	if pick != "" {
		return bounceToHand(g, p, "support", pick)
	}
	return nil
}
