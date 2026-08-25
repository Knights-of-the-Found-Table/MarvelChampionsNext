package echo

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// registerEchoCards installs the remaining Echo-side Fear No Evil cards:
// the Daredevil ally, the box's interrupt events and the Kingpin nemesis
// set (60039, 60049-60056, 60060-60064).
func registerEchoCards() {
	registerDaredevilAlly()
	registerGetTheirAttention()
	registerPowerfulPunch()
	registerSeeNoEvil()
	registerSuperpowerTraining()
	registerKingpinSet()
}

// fneInt dereferences an optional card stat.
func fneInt(v *int, fallback int) int {
	if v == nil {
		return fallback
	}
	return *v
}

// 60039 Daredevil: after he uses a basic power, the next event the owner
// plays this round costs 1 less.
func registerDaredevilAlly() {
	discount := func(g *engine.Game, pid engine.PlayerID) {
		p := g.Player(pid)
		if p == nil {
			return
		}
		p.CostDiscounts = append(p.CostDiscounts, engine.CostDiscount{Type: "event", Amount: 1})
		g.TLogf("c.daredevilSNextEventCosts1Less", p.Name)
	}
	engine.RegisterBehavior("60039", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			a, ok := e.(*engine.Ally)
			if !ok {
				return nil
			}
			switch m := msg.(type) {
			case engine.AllyThwartWindow:
				if m.Ally == a.ID {
					discount(g, a.Owner)
				}
			case engine.AllyAttackWindow:
				if m.Ally == a.ID {
					discount(g, a.Owner)
				}
			}
			return nil
		},
	})
}

// 60049 Get Their Attention: hero interrupt — when an enemy initiates an
// attack, remove 3 threat from a scheme.
func registerGetTheirAttention() {
	engine.RegisterBehavior("60049", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			var choices []engine.Choice
			for _, id := range g.Schemes() {
				s := g.Entity(id)
				choices = append(choices, engine.Choice{
					Label: engine.S(s.EDef().Name), Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: s.ECode(),
				}.Msgs(engine.ThwartScheme{Scheme: id, N: 3, Source: p.ID}))
			}
			if len(choices) == 0 {
				return engine.Defends{}, nil, false
			}
			return engine.Defends{Defender: p.ID, Against: against},
				[]engine.Message{engine.AskQuestion{
					Player:   p.ID,
					Question: engine.Ask(engine.Tf("c.getTheirAttentionRemove3ThreatFromWhichScheme"), choices...),
				}}, true
		},
	})
}

// 60051 Powerful Punch: hero interrupt — when an enemy initiates an
// attack, deal 4 damage to that enemy.
func registerPowerfulPunch() {
	engine.RegisterBehavior("60051", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			return engine.Defends{Defender: p.ID, Against: against},
				[]engine.Message{engine.DamageEntity{Target: against, Damage: 4, Source: p.ID}}, true
		},
	})
}

// 60055 See No Evil, Hear No Evil: choose 2 of {deal 3 damage to an
// enemy, remove 3 threat from a scheme}; options may repeat.
func registerSeeNoEvil() {
	pick := func(g *engine.Game, pid engine.PlayerID, n string) *engine.Question {
		damage := engine.Choice{
			ID: "damage", Label: engine.Tf("c.deal3DamageToAnEnemy"), Kind: engine.ChoiceLabel,
		}
		if enemies := cardutil.EnemyChoices(g, 3, pid, func(target engine.EntityID) []engine.Message {
			return []engine.Message{engine.DamageEntity{Target: target, Damage: 3, Source: pid}}
		}); len(enemies) > 0 {
			damage = damage.WithThen(engine.Ask(engine.Tf("c.seeNoEvilChooseAnEnemy"), enemies...))
		}
		threat := engine.Choice{
			ID: "threat", Label: engine.Tf("c.remove3ThreatFromAScheme"), Kind: engine.ChoiceLabel,
		}
		var schemes []engine.Choice
		for _, id := range g.Schemes() {
			s := g.Entity(id)
			schemes = append(schemes, engine.Choice{
				Label: engine.S(s.EDef().Name), Kind: engine.ChoiceTarget,
				SourceID: id, CardCode: s.ECode(),
			}.Msgs(engine.ThwartScheme{Scheme: id, N: 3, Source: pid}))
		}
		if len(schemes) > 0 {
			threat = threat.WithThen(engine.Ask(engine.Tf("c.seeNoEvilChooseAScheme"), schemes...))
		}
		return engine.Ask(engine.S(n), damage, threat)
	}
	engine.RegisterBehavior("60055", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			return []engine.Message{
				engine.AskQuestion{Player: pid, Question: pick(g, pid, "See No Evil — choose option 1 of 2")},
				engine.AskQuestion{Player: pid, Question: pick(g, pid, "See No Evil — choose option 2 of 2")},
			}
		},
	})
}

// 60056 Superpower Training: when defeated, each player may search their
// deck and discard pile for an identity-specific upgrade and put it into
// play (approximation: auto-takes the first hit, then shuffles; discard
// hits are taken without a shuffle).
func registerSuperpowerTraining() {
	engine.RegisterBehavior("60056", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			var out []engine.Message
			for _, p := range g.Players {
				heroDef, ok := engine.DB.Lookup(data.HeroSideCode(data.BaseCode(p.HeroCode)))
				if !ok || heroDef.CardSet == "" {
					continue
				}
				// UpgradeEnterPlay removes the card from deck/discard
				// itself; the deck is shuffled after the fetch.
				found := false
				for _, c := range p.Deck {
					if c.Def().Type == "upgrade" && c.Def().CardSet == heroDef.CardSet {
						out = append(out,
							engine.UpgradeEnterPlay{Player: p.ID, Card: c},
							engine.ShufflePlayerDeck{Player: p.ID})
						found = true
						break
					}
				}
				if !found {
					for _, c := range p.Discard {
						if c.Def().Type == "upgrade" && c.Def().CardSet == heroDef.CardSet {
							out = append(out, engine.UpgradeEnterPlay{Player: p.ID, Card: c})
							break
						}
					}
				}
			}
			return out
		},
	})
}

// findKingpin pulls Kingpin (60061) from the player's nemesis deck or
// discard and returns the enter-play messages. No-op when he is already
// in play or not findable.
func findKingpin(g *engine.Game, p *engine.Player) []engine.Message {
	for _, mn := range g.Minions {
		if mn != nil && mn.Code == "60061" {
			g.TLogf("c.kingpinIsAlreadyInPlay")
			return nil
		}
	}
	var card engine.Card
	found := false
	for i, c := range p.NemesisDeck {
		if c.Code == "60061" {
			card = c
			p.NemesisDeck = append(p.NemesisDeck[:i:i], p.NemesisDeck[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		for i, c := range p.NemesisDiscard {
			if c.Code == "60061" {
				card = c
				p.NemesisDiscard = append(p.NemesisDiscard[:i:i], p.NemesisDiscard[i+1:]...)
				found = true
				break
			}
		}
	}
	if !found {
		g.TLogf("c.cannotFindKingpin", p.Name)
		return nil
	}
	def := card.Def()
	mn := &engine.Minion{
		ID:          g.NextEntityID(engine.KindMinion),
		Code:        card.Code,
		MaxHP:       fneInt(def.HP, 1),
		AttackVal:   fneInt(def.Attack, 0),
		SchemeVal:   fneInt(def.Scheme, 0),
		EngagedWith: p.ID,
	}
	g.Minions[mn.ID] = mn
	g.TLogf("c.kingpinEntersPlayEngagedWith", p.Name)
	return []engine.Message{engine.MinionEntersPlay{MinionID: mn.ID, Player: p.ID}}
}

// registerKingpinSet installs the echo_nemesis encounter set
// (60060-60064).
func registerKingpinSet() {
	// 60060 Raised by the Kingpin: obligation — find Kingpin and put him
	// into play engaged with the owner (approximation: obligations cannot
	// persist, so the "cannot deal damage to Kingpin" rider and the
	// threat-tallying removal are not modeled).
	engine.RegisterBehavior("60060", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			out := findKingpin(g, p)
			out = append(out, engine.ObligationResolve{Player: p.ID, Card: card})
			return out
		},
	})

	// 60061 Kingpin: he schemes instead of attacking the Maya Lopez
	// player; he cannot take damage while Master Manipulator or a
	// Kingpin's Henchman is in play.
	engine.RegisterBehavior("60061", &engine.Behavior{
		MinionActivate: func(g *engine.Game, mn *engine.Minion, p *engine.Player) []engine.Message {
			if p.IsHero() && data.BaseCode(p.HeroCode) == "60037" {
				g.TLogf("c.kingpinSchemesAgainstInsteadOfAttacking", p.Name)
				if g.MainScheme != nil {
					return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: mn.SchemeVal, Source: mn.ID}}
				}
				return nil
			}
			// default activation (the hook replaces it entirely)
			if p.IsHero() {
				if mn.Stunned {
					mn.Stunned = false
					g.TLogf("c.kingpinIsStunnedAttackCanceled")
					return nil
				}
				return []engine.Message{engine.AskAttack{Enemy: mn.ID, Player: p.ID}}
			}
			if mn.Confused {
				mn.Confused = false
				return nil
			}
			if g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: mn.SchemeVal, Source: mn.ID}}
			}
			return nil
		},
		MinionDamageable: func(g *engine.Game, m *engine.Minion, damage int) bool {
			for _, s := range g.SideSchemes {
				if s != nil && s.Code == "60062" {
					g.TLogf("c.kingpinCannotTakeDamageMasterManipulator")
					return false
				}
			}
			for _, mn := range g.Minions {
				if mn != nil && mn.Code == "60063" {
					g.TLogf("c.kingpinCannotTakeDamageKingpinSHenchman")
					return false
				}
			}
			return true
		},
	})

	// 60062 Master Manipulator: when revealed, find Kingpin and put him
	// into play engaged with the revealer (approximation: the first
	// player; the boost rider is not modeled).
	engine.RegisterBehavior("60062", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(cardutil.FirstPlayerID(g))
			if p == nil {
				return nil
			}
			return findKingpin(g, p)
		},
	})

	// 60063 Kingpin's Henchman: Guard is generic; the "Kingpin cannot
	// take damage" aura is modeled on Kingpin's behavior.
	engine.RegisterBehavior("60063", &engine.Behavior{})

	// 60064 Pawn of the Kingpin: alter-ego — Kingpin schemes (surge if he
	// is not in play); hero — deal yourself damage equal to your ATK.
	engine.RegisterBehavior("60064", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			if p == nil {
				return nil
			}
			if p.IsHero() {
				n := p.AttackStat(g)
				g.TLogf("c.pawnOfTheKingpinTakesDamage", p.Name, n)
				return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: n, Source: engine.EntityID("treachery")}}
			}
			var kingpin *engine.Minion
			for _, mn := range g.Minions {
				if mn != nil && mn.Code == "60061" {
					kingpin = mn
					break
				}
			}
			if kingpin != nil && g.MainScheme != nil {
				g.TLogf("c.pawnOfTheKingpinKingpinSchemes")
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: kingpin.SchemeVal, Source: kingpin.ID}}
			}
			return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
		},
	})
}
