package aos

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() {
	registerBlackWidowVillain(); registerAIMSets() }

// widowAttackTax implements Black Widow's forced interrupt shared by all
// three stages: when a character you control attacks her, remove 1 threat
// from the main scheme and discard the encounter deck's top card. The
// "resolve Preparation abilities" rider is approximated by the discard.
func widowAttackTax(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
	atk, ok := msg.(engine.BasicAttack)
	if !ok || atk.Target != e.EID() || g.MainScheme == nil {
		return nil
	}
	var out []engine.Message
	if g.MainScheme.Threat > 0 {
		out = append(out, engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 1, Source: e.EID()})
	}
	if c, ok := g.DrawEncounter(); ok {
		g.Logf("Black Widow's intel: %s is discarded from the encounter deck", c.Def().Name)
		g.EncounterDiscard = append(g.EncounterDiscard, c)
	}
	return out
}

func registerBlackWidowVillain() {
	engine.RegisterBehavior("50064", &engine.Behavior{React: widowAttackTax})

	// 50065 Black Widow (II): entry tax — 2 threat per hero.
	engine.RegisterBehavior("50065", &engine.Behavior{
		React: widowAttackTax,
		VillainStage: func(g *engine.Game, v *engine.Villain, nextStage int) []engine.Message {
			if g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2 * len(g.Players), Source: v.ID}}
			}
			return nil
		},
	})

	// 50066 Black Widow (III): entry tax — 3 threat per hero.
	engine.RegisterBehavior("50066", &engine.Behavior{
		React: widowAttackTax,
		VillainStage: func(g *engine.Game, v *engine.Villain, nextStage int) []engine.Message {
			if g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 3 * len(g.Players), Source: v.ID}}
			}
			return nil
		},
	})

	// 50067a The Widow's Web main scheme — single stage, no extra effects.
	engine.RegisterBehavior("50067", &engine.Behavior{})

	// 50068 Black Widow's Gauntlet: retaliate handled by the engine; the
	// no-Preparation-resolved discard response is approximated away.
	engine.RegisterBehavior("50068", &engine.Behavior{})

	// 50069 Grappling Hook: event cancellation not modeled.
	engine.RegisterBehavior("50069", &engine.Behavior{})

	// 50070 Night Vision Goggles: global Preparation graft not modeled.
	engine.RegisterBehavior("50070", &engine.Behavior{})

	// 50071 Stun Net: attack gating for identity attachments is not
	// modeled; the card stays as a marker until the scenario ends.
	engine.RegisterBehavior("50071", &engine.Behavior{})

	// 50076 Attacrobatics: switch to hero form; Black Widow attacks you
	// (expert rider not modeled).
	engine.RegisterBehavior("50076", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var out []engine.Message
			if !p.IsHero() {
				out = append(out, engine.ChangeForm{Player: p.ID})
			}
			for id := range g.Villains {
				out = append(out, engine.AskAttack{Enemy: id, Player: p.ID})
				break
			}
			return out
		},
	})

	// 50077 Covert Ops: you are confused; Black Widow schemes.
	engine.RegisterBehavior("50077", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			out := []engine.Message{engine.ConfuseEntity{Target: p.ID}}
			for id, v := range g.Villains {
				base := data.BaseCode(v.Code)
				if base == "50064" || base == "50065" || base == "50066" {
					if g.MainScheme != nil {
						out = append(out, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: v.SchemeVal, Source: id})
					}
					break
				}
			}
			return out
		},
	})

	// 50078 Dance of Death: 1/2/3 damage spread over your characters
	// (approximated: identity takes 2).
	engine.RegisterBehavior("50078", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var out []engine.Message
			n := 1
			out = append(out, engine.DamageEntity{Target: p.ID, Damage: 2, Source: t.ID})
			for _, aid := range p.Allies {
				if a := g.Allies[aid]; a != nil {
					out = append(out, engine.DamageEntity{Target: aid, Damage: n, Source: t.ID})
					n++
					if n > 2 {
						break
					}
				}
			}
			return out
		},
	})

	// 50079 Widow's Bite: stunned plus 1 damage (2 if already stunned).
	engine.RegisterBehavior("50079", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			dmg := 1
			if p.Stunned {
				dmg = 2
			}
			return []engine.Message{engine.StunEntity{Target: p.ID}, engine.DamageEntity{Target: p.ID, Damage: dmg, Source: t.ID}}
		},
	})
}

func registerAIMSets() {
	// 50080 A.I.M. Abductor: tuck the healthiest ally under Abduct
	// Superhumans, else grow the scheme.
	engine.RegisterBehavior("50080", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			p := g.Player(mn.EngagedWith)
			if p == nil {
				return nil
			}
			s := abductScheme(g)
			if s == nil {
				return nil
			}
			best, bestHP := engine.EntityID(""), -1
			for _, aid := range p.Allies {
				if a := g.Allies[aid]; a != nil && a.HP() > bestHP {
					best, bestHP = aid, a.HP()
				}
			}
			if best != "" {
				a := g.Allies[best]
				s.StoredCards = append(s.StoredCards, engine.Card{ID: g.NextCardID(), Code: a.Code})
				g.Delete(best)
				g.Logf("A.I.M. abducts %s", a.EDef().Name)
				return []engine.Message{engine.SchemeThreat{Scheme: s.ID, N: cardutil.Cost(a.EDef()), Source: e.EID()}}
			}
			return []engine.Message{engine.SchemeThreat{Scheme: s.ID, N: 2, Source: e.EID()}}
		},
	})

	// 50081 Abduct Superhumans: steals leaving allies; defeating it
	// returns them.
	engine.RegisterBehavior("50081", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyDestroyed)
			if !ok {
				return nil
			}
			s := g.SideSchemes[e.EID()]
			a := g.Allies[m.AllyID] // the ally still exists at reaction time
			if s == nil || a == nil {
				return nil
			}
			s.StoredCards = append(s.StoredCards, engine.Card{ID: g.NextCardID(), Code: a.Code})
			g.Logf("%s is abducted into Abduct Superhumans", a.EDef().Name)
			return []engine.Message{
				engine.SchemeThreat{Scheme: s.ID, N: cardutil.Cost(a.EDef()), Source: s.ID},
				engine.AddAccelerationToken{Scheme: s.ID},
			}
		},
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			var out []engine.Message
			for _, c := range s.StoredCards {
				for _, p := range g.Players {
					if p == nil {
						continue
					}
					for _, hc := range p.Discard {
						if hc.Code == c.Code {
							out = append(out, engine.AllyEntersPlayFree{Player: p.ID, Card: hc, FromOwner: p.ID})
							break
						}
					}
				}
			}
			s.StoredCards = nil
			return out
		},
	})

	// 50082 Nabbed!: find the scheme, tuck a milled ally under it.
	engine.RegisterBehavior("50082", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			s := abductScheme(g)
			if s == nil {
				return nil
			}
			for i, c := range p.Deck {
				if c.Def().Type == "ally" {
					card := c
					p.Deck = append(p.Deck[:i:i], p.Deck[i+1:]...)
					s.StoredCards = append(s.StoredCards, card)
					g.Logf("%s is nabbed into Abduct Superhumans", card.Def().Name)
					return []engine.Message{
						engine.SchemeThreat{Scheme: s.ID, N: cardutil.Cost(card.Def()), Source: t.ID},
						engine.AddAccelerationToken{Scheme: s.ID},
					}
				}
			}
			return nil
		},
	})

	// 50083 A.I.M. Scientist: cannot be attacked while the engaged player
	// has another minion.
	engine.RegisterBehavior("50083", &engine.Behavior{
		MinionDamageable: func(g *engine.Game, m *engine.Minion, damage int) bool {
			for _, other := range g.Minions {
				if other != nil && other.ID != m.ID && other.EngagedWith == m.EngagedWith {
					return false
				}
			}
			return true
		},
	})

	// 50084 A.I.M. Soldier: fetches an A.I.M. Scientist.
	engine.RegisterBehavior("50084", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil {
				return nil
			}
			for i, c := range g.EncounterDeck {
				if c.Code == "50083" {
					g.EncounterDeck = append(g.EncounterDeck[:i:i], g.EncounterDeck[i+1:]...)
					def := c.Def()
					m2 := &engine.Minion{
						ID: g.NextEntityID(engine.KindMinion), Code: c.Code,
						MaxHP:     intValue(def.HP, 1),
						AttackVal: intValue(def.Attack, 0), SchemeVal: intValue(def.Scheme, 0),
						EngagedWith: mn.EngagedWith,
					}
					g.Minions[m2.ID] = m2
					g.Logf("A.I.M. Scientist joins the fight")
					return []engine.Message{engine.MinionEntersPlay{MinionID: m2.ID, Player: mn.EngagedWith}}
				}
			}
			if g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: e.EID()}}
			}
			return nil
		},
	})

	// 50085 Mad Science: Hinder handled by the engine; the acceleration
	// icon aura is approximated away.
	engine.RegisterBehavior("50085", &engine.Behavior{})

	// 50072 A.I.M. Commando: Quickstrike handled by the engine; the
	// Preparation boost rider is approximated away.
	engine.RegisterBehavior("50072", &engine.Behavior{})

	// 50073 A.I.M. Grunt: Guard handled by the engine.
	engine.RegisterBehavior("50073", &engine.Behavior{})

	// 50074 Automated Defenses: Hinder handled by the engine; the
	// Preparation graft is approximated away.
	engine.RegisterBehavior("50074", &engine.Behavior{})

	// 50075 Destroy Evidence: Hinder handled by the engine; the incite
	// graft is approximated away.
	engine.RegisterBehavior("50075", &engine.Behavior{})
}

// abductScheme finds Abduct Superhumans in play (or puts it into play).
func abductScheme(g *engine.Game) *engine.SideScheme {
	for _, s := range g.SideSchemes {
		if s != nil && s.Code == "50081" {
			return s
		}
	}
	var found bool
	for i, c := range g.EncounterDeck {
		if c.Code == "50081" {
			g.EncounterDeck = append(g.EncounterDeck[:i:i], g.EncounterDeck[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		for i, c := range g.EncounterDiscard {
			if c.Code == "50081" {
				g.EncounterDiscard = append(g.EncounterDiscard[:i:i], g.EncounterDiscard[i+1:]...)
				found = true
				break
			}
		}
	}
	if !found {
		return nil
	}
	s := &engine.SideScheme{
		ID: g.NextEntityID(engine.KindSideScheme), Code: "50081",
		Threat: len(g.Players), MaxThreat: len(g.Players) + 2,
	}
	g.SideSchemes[s.ID] = s
	g.Logf("Abduct Superhumans enters play")
	return s
}
