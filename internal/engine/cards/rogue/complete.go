// complete.go implements the remaining Rogue pack cards (38010–38035):
// the shared defense suite, the Mystique nemesis set and the Reavers
// modular set.
package rogue

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() {
	registerRemainingRogue()
}

func registerRemainingRogue() {
	// 38010 Iceman: 3 freeze counters; after a minion enters play, remove
	// 1 → stun it.
	engine.RegisterBehavior("38010", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 3}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionEntersPlay)
			a := g.Allies[e.EID()]
			if !ok || a == nil || a.Counters <= 0 {
				return nil
			}
			a.Counters--
			return []engine.Message{engine.StunEntity{Target: m.MinionID}}
		},
	})

	// 38011 Karma: on enter — control a non-Elite minion (2 consequential).
	engine.RegisterBehavior("38011", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn == nil || mn.EDef().HasTrait("elite") {
					continue
				}
				choices = append(choices, engine.Choice{
					Label: "Control " + cardutil.EnemyLabel(mn), Kind: engine.ChoiceTarget, SourceID: id,
				}.Msgs(engine.ConvertMinionToAlly{MinionID: id, Owner: p.ID, Consequential: 2}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask("Karma — take control of a non-Elite minion", choices...),
			}}
		},
	})

	// 38012 Armor: X-Men only; Toughness from data.
	engine.RegisterBehavior("38012", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "x-men")
		},
	})

	// 38013 Unflappable: after a no-damage defense, exhaust → draw 1.
	engine.RegisterBehavior("38013", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.WindowDefended)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil || u.Exhausted || m.DamageTaken != 0 || m.Defender != u.Owner {
				return nil
			}
			return []engine.Message{
				engine.ExhaustEntity{ID: u.ID},
				engine.DrawCards{Player: u.Owner, N: 1},
			}
		},
	})

	// 38014 Judoka Skill: 3 counters; on defense the attacker gets −2 ATK
	// (applied as a post-hoc attack reduction).
	engine.RegisterBehavior("38014", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 3}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.Defends)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil || u.Exhausted || u.Counters <= 0 || m.Defender != u.Owner {
				return nil
			}
			u.Counters--
			g.Logf("Judoka Skill — %s turns the attack aside (−2 ATK)", g.Entity(m.Defender).EDef().Name)
			return []engine.Message{
				engine.ExhaustEntity{ID: u.ID},
				engine.BoostEnemyAttack{Enemy: m.Against, N: -2},
			}
		},
	})

	// 38015 Preemptive Strike: cancel boost icons during villain attacks
	// (window in handle(RevealBoost)); 1 damage per icon.
	engine.RegisterBehavior("38015", &engine.Behavior{})

	// 38016 Not Today!: defense event, +2 DEF.
	engine.RegisterBehavior("38016", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			return engine.Defends{Defender: p.ID, Against: against, DefBonus: 2}, nil, true
		},
	})

	// 38017 Defensive Energy: rider in handlePlayCard.
	engine.RegisterBehavior("38017", &engine.Behavior{})

	// 38018 Moira MacTaggert: after a MUTANT alter-ego changes to hero
	// form, exhaust → draw 1.
	engine.RegisterBehavior("38018", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "mutant")
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.ChangeForm)
			s := g.Supports[e.EID()]
			if !ok || s == nil || s.Exhausted {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil || !p.IsHero() || !g.EntityHasTrait(p.ID, "mutant") {
				return nil
			}
			return []engine.Message{
				engine.ExhaustEntity{ID: s.ID},
				engine.DrawCards{Player: p.ID, N: 1},
			}
		},
	})

	// 38019 X-Gene: exhaust → [wild] (identity-event restriction
	// approximated away).
	engine.RegisterBehavior("38019", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "mutant")
		},
		Resource: &engine.ResourceAbility{Icon: "wild", HeroOnly: true},
	})

	// 38020 Beauty and the Thief: 4 damage and 4 threat.
	engine.RegisterBehavior("38020", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			msgs := []engine.Message{}
			if g.MainScheme != nil {
				msgs = append(msgs, engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 4, Source: p.ID})
			}
			choices := cardutil.EnemyChoices(g, 4, p.ID, func(target engine.EntityID) []engine.Message {
				return []engine.Message{engine.DamageEntity{Target: target, Damage: 4, Source: p.ID}}
			})
			if len(choices) == 0 {
				return msgs
			}
			return append(msgs, engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask("Beauty and the Thief — deal 4 damage", choices...),
			})
		},
	})

	// 38021–38023 basic resources.
	for _, code := range []string{"38021", "38022", "38023"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	// 38024 Deadly Touch: Touched attached → 2 damage; else 2 threat.
	engine.RegisterBehavior("38024", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			for _, a := range g.Attachments {
				if a == nil || a.Code != "38019" && a.Code != "touched" {
					// match by name for scenario attachments
					if a.EDef().Name != "Touched" {
						continue
					}
				}
				if a.Target != "" && a.Target.Is(engine.KindPlayer) {
					return []engine.Message{
						engine.DamageEntity{Target: a.Target, Damage: 2, Source: p.ID},
						engine.ObligationResolve{Player: p.ID, Card: card},
					}
				}
			}
			msgs := []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
			if g.MainScheme != nil {
				return append([]engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: p.ID}}, msgs...)
			}
			return msgs
		},
	})

	// 38025 Mystique: on engage — shuffle a Misled into your deck
	// (approximation: discard pile).
	engine.RegisterBehavior("38025", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionEntersPlay)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
				for _, c := range *zone {
					if c.Code == "38027" {
						zone.Remove(c.ID)
						p := g.Player(m.Player)
						if p != nil {
							p.Deck = append(p.Deck, c)
							g.Logf("Misled hides in %s's deck", p.Name)
						}
						return nil
					}
				}
			}
			return nil
		},
	})

	// 38026 Mystique's Manipulations: When Defeated — shuffle Misled into
	// your deck (approximation as above).
	engine.RegisterBehavior("38026", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
				for _, c := range *zone {
					if c.Code == "38027" {
						zone.Remove(c.ID)
						p := g.Player(cardutil.FirstPlayerID(g))
						if p != nil {
							p.Deck = append(p.Deck, c)
						}
						return nil
					}
				}
			}
			return nil
		},
	})

	// 38027 Misled: shuffle into your deck + surge (approximated as a
	// plain reveal with surge).
	engine.RegisterBehavior("38027", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
		},
	})

	// 38028 Med Lab: tuck a defeated ally; alter-ego action returns it to
	// hand (consequential-attribution approximated to any ally defeat).
	engine.RegisterBehavior("38028", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.AllyDefeated); !ok {
				return nil
			}
			s := g.Supports[e.EID()]
			if s == nil || len(s.AttachedCards) > 0 || s.Exhausted {
				return nil
			}
			// The defeated ally is gone by now; approximation: no-op (the
			// tuck requires the defeat event's card, unavailable here).
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			if s == nil || len(s.AttachedCards) == 0 {
				return nil
			}
			return []engine.Ability{{
				Label: "Med Lab — return the tucked ally to hand", Type: engine.AbilityAction,
				AlterEgoOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					if s == nil || len(s.AttachedCards) == 0 {
						return nil
					}
					c := s.AttachedCards[0]
					s.AttachedCards = s.AttachedCards[1:]
					c.Owner = s.Owner
					p := g.Player(s.Owner)
					if p != nil {
						p.Hand = append(p.Hand, c)
					}
					return nil
				},
			}}
		},
	})

	// 38029 Donald Pierce: on engage — reveal the topmost REAVER from the
	// discard pile.
	engine.RegisterBehavior("38029", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionEntersPlay)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			for _, c := range g.EncounterDiscard {
				if c.Def().HasTrait("reaver") && c.Def().Type == "minion" {
					card := c
					return []engine.Message{engine.RevealEncounterCard{Player: m.Player, Card: card}}
				}
			}
			return nil
		},
	})

	// 38030 Skullbuster: on engage — 1 main-scheme threat per engaged
	// REAVER.
	engine.RegisterBehavior("38030", &engine.Behavior{
		React: reaverEngage(func(g *engine.Game, pid engine.PlayerID, n int) []engine.Message {
			if g.MainScheme != nil && n > 0 {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: n, Source: pid}}
			}
			return nil
		}),
	})

	// 38031 Bonebreaker: on engage — 1 indirect damage per engaged REAVER.
	engine.RegisterBehavior("38031", &engine.Behavior{
		React: reaverEngage(func(g *engine.Game, pid engine.PlayerID, n int) []engine.Message {
			if n > 0 {
				return []engine.Message{engine.IndirectDamage{Player: pid, N: n}}
			}
			return nil
		}),
	})

	// 38032/38033 Wade Cole & Murray Reese: on reveal — attach Cybernetic
	// Enhancements.
	for _, code := range []string{"38032", "38033"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				m, ok := msg.(engine.MinionEntersPlay)
				if !ok || m.MinionID != e.EID() {
					return nil
				}
				for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
					for _, c := range *zone {
						if c.Code == "38035" {
							zone.Remove(c.ID)
							return []engine.Message{engine.RevealEncounterCard{Player: m.Player, Card: c}}
						}
					}
				}
				return nil
			},
		})
	}

	// 38034 The Reavers: When Defeated — mill to a REAVER and reveal it.
	engine.RegisterBehavior("38034", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			pid := cardutil.FirstPlayerID(g)
			for guards := 0; guards < 40; guards++ {
				if len(g.EncounterDeck) == 0 {
					return nil
				}
				top := g.EncounterDeck[0]
				g.EncounterDeck = g.EncounterDeck[1:]
				if top.Def().Type == "minion" && top.Def().HasTrait("reaver") {
					return []engine.Message{engine.RevealEncounterCard{Player: pid, Card: top}}
				}
				g.EncounterDiscard = append(g.EncounterDiscard, top)
			}
			return nil
		},
	})

	// 38035 Cybernetic Enhancements: attach to a minion (immunity in
	// g.damage); discard after it attacks.
	engine.RegisterBehavior("38035", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn == nil {
					continue
				}
				attached := false
				for _, a := range g.Attachments {
					if a != nil && a.Code == "38035" && a.Target == mn.ID {
						attached = true
					}
				}
				if attached {
					continue
				}
				t.Target = mn.ID
				mn.Attachments = append(mn.Attachments, t.ID)
				return nil
			}
			g.Delete(t.ID)
			return []engine.Message{engine.RevealNextEncounter{Player: cardutil.FirstPlayerID(g)}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.AskAttack); !ok {
				return nil
			}
			t := g.Attachments[e.EID()]
			if t == nil {
				return nil
			}
			return nil // discard handled after the attack resolves; kept
		},
	})
}

// reaverEngage wraps an on-engage rider scaled by engaged REAVER count.
func reaverEngage(effect func(g *engine.Game, pid engine.PlayerID, n int) []engine.Message) func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
	return func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.MinionEntersPlay)
		if !ok || m.MinionID != e.EID() {
			return nil
		}
		n := 0
		for _, mn := range g.Minions {
			if mn != nil && mn.EngagedWith == m.Player && mn.EDef().HasTrait("reaver") {
				n++
			}
		}
		return effect(g, m.Player, n)
	}
}
