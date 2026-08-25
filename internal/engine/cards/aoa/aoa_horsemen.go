package aoa

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// horsemanByCode finds a Horsemen villain by base code.
func horsemanByCode(g *engine.Game, base string) *engine.Villain {
	for _, v := range g.Villains {
		if v != nil && engine.BaseCodeOf(v.Code) == base {
			return v
		}
	}
	return nil
}

// horsemanForced resolves the named Horseman's forced response against
// the player (War discard / Famine mill / Pestilence blank / Death chip).
func horsemanForced(g *engine.Game, base string, p *engine.Player) []engine.Message {
	if p == nil {
		return nil
	}
	switch base {
	case "45081": // War
		if len(p.Supports) > 0 {
			return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Supports[0]}}
		}
		if len(p.Upgrades) > 0 {
			return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Upgrades[0]}}
		}
	case "45082": // Famine
		return []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: 10}}
	case "45083": // Pestilence — text blanking not modeled
		g.TLogf("c.pestilenceSPlagueBlanksTheIdentityNotModeled")
	case "45084": // Death
		var msgs []engine.Message
		msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: 1, Source: engine.EntityID("")})
		for _, id := range p.Allies {
			msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: engine.EntityID("")})
		}
		return msgs
	}
	return nil
}

// horsemanVillainBehavior builds a Horseman's shared behavior: post-attack
// riders and the undamageable-while-siblings-live gate.
func horsemanVillainBehavior(base string) *engine.Behavior {
	return &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			v := g.Villains[e.EID()]
			if v == nil || v.HP() < 1 {
				return nil
			}
			// An ally may have defended; the rider hits its controller.
			pid := m.Player
			if a := g.Allies[engine.EntityID(pid)]; a != nil {
				pid = a.Owner
			}
			return horsemanForced(g, base, g.Player(pid))
		},
		VillainDamageable: func(g *engine.Game, v *engine.Villain, damage int) bool {
			for _, o := range g.Villains {
				if o != nil && o.ID != v.ID && o.HP() >= 1 {
					g.TLogf("c.cannotBeDefeatedWhileAnotherHorsemanLives", v)
					return false
				}
			}
			return true
		},
	}
}

func registerHorsemen() {
	for _, base := range []string{"45081", "45082", "45083", "45084"} {
		engine.RegisterBehavior(base, horsemanVillainBehavior(base))
	}

	// 45085 The Horsemen of Apocalypse: after a villain activates, rotate
	// the active counter (approximation: the engine activates villains in
	// order already; the rotation matters for Rough Riders).
	engine.RegisterBehavior("45085", &engine.Behavior{})

	// 45086-45089 side schemes: the defeat riders.
	riders := map[string]func(g *engine.Game, p *engine.Player) []engine.Message{
		"45086": func(g *engine.Game, p *engine.Player) []engine.Message {
			if len(p.Supports) > 0 {
				return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Supports[0]}}
			}
			if len(p.Upgrades) > 0 {
				return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Upgrades[0]}}
			}
			return nil
		},
		"45087": func(g *engine.Game, p *engine.Player) []engine.Message {
			return []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: 10}}
		},
		"45088": func(g *engine.Game, p *engine.Player) []engine.Message {
			g.TLogf("c.plagueAndPestilenceBlanksTheIdentityNotModeled")
			return nil
		},
		"45089": func(g *engine.Game, p *engine.Player) []engine.Message {
			var msgs []engine.Message
			msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: 1, Source: engine.EntityID("")})
			for _, id := range p.Allies {
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: engine.EntityID("")})
			}
			return msgs
		},
	}
	for code, rider := range riders {
		r := rider
		engine.RegisterBehavior(code, &engine.Behavior{
			SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
				return r(g, g.Player(cardutil.FirstPlayerID(g)))
			},
		})
	}

	// 45090 Golden Horse: the weakest non-Aerial villain gains Aerial and
	// deathless status (approximated: +nothing; the hero response is not
	// modeled).
	engine.RegisterBehavior("45090", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			best, bestID := 1<<30, engine.EntityID("")
			for id, v := range g.Villains {
				if v != nil && !v.EDef().HasTrait("Aerial") && v.HP() < best {
					best, bestID = v.HP(), id
				}
			}
			t.Target = bestID
			return nil
		},
	})

	// 45091 Metal Wings: attach to Death + retaliate 1 (engine retaliate
	// via attachment check would need code — approximated away).
	engine.RegisterBehavior("45091", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if v := horsemanByCode(g, "45084"); v != nil {
				t.Target = v.ID
				g.ActiveVillain = v.ID
			}
			return nil
		},
	})

	// 45092-45095 Horseman treacheries: heal + tough + activate; boost
	// re-activates.
	horsemanTreachery := func(base string) func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		return func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			v := horsemanByCode(g, base)
			if v == nil {
				return nil
			}
			return []engine.Message{
				engine.HealEntity{Target: v.ID, N: 2},
				engine.ToughEntity{Target: v.ID},
				engine.VillainActivates{VillainID: v.ID, Player: p.ID},
			}
		}
	}
	horsemanBoost := func(base string) func(g *engine.Game, card engine.Card) []engine.Message {
		return func(g *engine.Game, card engine.Card) []engine.Message {
			v := horsemanByCode(g, base)
			p := g.Player(cardutil.FirstPlayerID(g))
			if v == nil || p == nil {
				return nil
			}
			return []engine.Message{engine.VillainActivates{VillainID: v.ID, Player: p.ID}}
		}
	}
	for _, base := range []string{"45081", "45082", "45083", "45084"} {
		code := map[string]string{
			"45081": "45092", "45082": "45093", "45083": "45094", "45084": "45095",
		}[base]
		b := base
		engine.RegisterBehavior(code, &engine.Behavior{
			ResolveTreachery: horsemanTreachery(b),
			Boost:            horsemanBoost(b),
		})
	}

	// 45096 Rough Riders: the active villain's rider, then the next's.
	engine.RegisterBehavior("45096", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var msgs []engine.Message
			for _, base := range []string{"45081", "45082", "45083", "45084"} {
				if v := horsemanByCode(g, base); v != nil {
					msgs = append(msgs, horsemanForced(g, base, p)...)
				}
			}
			return msgs
		},
	})
}

func registerHounds() {
	// 45097 Ahab: fetches Release the Hounds.
	engine.RegisterBehavior("45097", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return nil
			}
			for _, s := range g.SideSchemes {
				if s != nil && s.Code == "45100" {
					return []engine.Message{engine.ApplySchemeThreat{Scheme: s.ID, N: 3, Source: mn.ID}}
				}
			}
			for _, c := range append(engine.CardList{}, g.EncounterDeck...) {
				if c.Code == "45100" {
					g.EncounterDeck.Remove(c.ID)
					return []engine.Message{engine.RevealEncounterCard{Player: mn.EngagedWith, Card: c}}
				}
			}
			for _, c := range append(engine.CardList{}, g.EncounterDiscard...) {
				if c.Code == "45100" {
					g.EncounterDiscard.Remove(c.ID)
					return []engine.Message{engine.RevealEncounterCard{Player: mn.EngagedWith, Card: c}}
				}
			}
			return nil
		},
	})

	// 45098 Hound: attacks heroes; flips alter-egos.
	engine.RegisterBehavior("45098", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return nil
			}
			p := g.Player(mn.EngagedWith)
			if p == nil {
				return nil
			}
			if p.IsHero() {
				return []engine.Message{engine.AskAttack{Enemy: mn.ID, Player: p.ID}}
			}
			return []engine.Message{engine.ChangeForm{Player: p.ID}}
		},
	})

	// 45099 Ahab's Energy Spear: overkill/piercing riders not modeled;
	// discards after the attack.
	engine.RegisterBehavior("45099", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, mn := range g.Minions {
				if mn != nil && engine.BaseCodeOf(mn.Code) == "45097" {
					t.Target = mn.ID
					return nil
				}
			}
			if id := firstVillainID(g); id != "" {
				t.Target = id
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok {
				return nil
			}
			t := g.Attachments[e.EID()]
			if t == nil || m.Enemy != t.Target {
				return nil
			}
			g.Delete(t.ID)
			g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: t.Code})
			g.TLogf("c.ahabSEnergySpearIsDiscarded")
			return nil
		},
	})

	// 45100 Release the Hounds: defeat feeds a facedown Hound.
	engine.RegisterBehavior("45100", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			for _, c := range append(engine.CardList{}, g.EncounterDeck...) {
				if c.Code == "45098" {
					g.EncounterDeck.Remove(c.ID)
					if p := g.Player(cardutil.FirstPlayerID(g)); p != nil {
						p.EncounterDown = append(p.EncounterDown, c)
						g.TLogf("c.aHoundIsDealtFacedownTo", p.Name)
					}
					return nil
				}
			}
			for _, c := range append(engine.CardList{}, g.EncounterDiscard...) {
				if c.Code == "45098" {
					g.EncounterDiscard.Remove(c.ID)
					if p := g.Player(cardutil.FirstPlayerID(g)); p != nil {
						p.EncounterDown = append(p.EncounterDown, c)
						g.TLogf("c.aHoundIsDealtFacedownTo", p.Name)
					}
					return nil
				}
			}
			return nil
		},
	})
}
