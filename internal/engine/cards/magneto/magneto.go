// Package magneto registers Magneto, his Magnetic suite, obligation, and nemesis set.
package magneto

import (
	"strings"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerMagneto()
	registerMagnetoSignatures()
	registerMagnetoObligation()
	registerMagnetoNemesis()
}

func ownedUpgrade(g *engine.Game, p *engine.Player, code string) *engine.Upgrade {
	if p == nil {
		return nil
	}
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil && u.Code == code {
			return u
		}
	}
	return nil
}

func pullMessages(g *engine.Game, p *engine.Player) []engine.Message {
	if p == nil {
		return nil
	}
	for i, c := range p.Deck {
		if !c.Def().HasTrait("magnetic") {
			continue
		}
		discarded := append(engine.CardList(nil), p.Deck[:i+1]...)
		msgs := []engine.Message{
			engine.MillPlayerDeck{Player: p.ID, N: i + 1},
			engine.ReturnDiscardCard{Player: p.ID, CardID: c.ID},
		}
		if ownedUpgrade(g, p, "49004") != nil {
			var mental, physical, energy bool
			for _, card := range discarded {
				for _, icon := range card.Def().Resources {
					switch icon {
					case "mental", "wild":
						mental = true
					case "physical":
						physical = true
					case "energy":
						energy = true
					}
					if icon == "wild" {
						physical, energy = true, true
					}
				}
			}
			bonus := engine.ApplyStatBonus{Target: p.ID}
			if mental {
				bonus.THW = 1
			}
			if physical {
				bonus.ATK = 1
			}
			if energy {
				bonus.DEF = 1
			}
			if bonus.THW+bonus.ATK+bonus.DEF > 0 {
				// ApplyStatBonus expires at phase end, earlier than the printed
				// end-of-round duration.
				msgs = append(msgs, bonus)
			}
		}
		if cape := ownedUpgrade(g, p, "49005"); cape != nil && !cape.Exhausted {
			// Cape's response is automatic because named ability resolutions do
			// not open an optional response question.
			msgs = append(msgs, engine.ExhaustEntity{ID: cape.ID}, engine.ReadyEntity{ID: p.ID})
		}
		return msgs
	}
	return nil
}

func registerMagneto() {
	engine.RegisterBehavior("49001", &engine.Behavior{
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			if !p.IsHero() {
				return nil
			}
			return []engine.Ability{{
				Label: "Magnetic Pull — discard until you find a Magnetic card",
				Type:  engine.AbilityAction, HeroOnly: true, OncePerRound: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return pullMessages(g, g.Player(self))
				},
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			p := g.Player(e.EID())
			m, ok := msg.(engine.ChangeForm)
			if !ok || p == nil || m.Player != p.ID || p.IsHero() {
				return nil
			}
			var msgs []engine.Message
			for i := len(p.Discard) - 1; i >= 0 && len(msgs) < 3; i-- {
				msgs = append(msgs, engine.ShuffleIntoDeck{Player: p.ID, CardID: p.Discard[i].ID})
			}
			return msgs
		},
	})
}

func registerMagnetoSignatures() {
	engine.RegisterBehavior("49002", &engine.Behavior{Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
		return []engine.Ability{{Label: "Asteroid M — recycle a Magnetic card and heal 1", Type: engine.AbilityAction, AlterEgoOnly: true, Exhaust: true,
			Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
				s := g.Supports[self]
				if s == nil {
					return nil
				}
				p := g.Player(s.Owner)
				for i := len(p.Discard) - 1; i >= 0; i-- {
					c := p.Discard[i]
					if c.Def().HasTrait("magnetic") {
						return []engine.Message{engine.ShuffleIntoDeck{Player: p.ID, CardID: c.ID}, engine.HealEntity{Target: p.ID, N: 1}}
					}
				}
				return []engine.Message{engine.HealEntity{Target: p.ID, N: 1}}
			}}}
	}})
	// The Magnetic-only payment gate is not expressible by ResourceAbility.
	engine.RegisterBehavior("49003", &engine.Behavior{Resource: &engine.ResourceAbility{Icon: "wild"}})
	// Magneto's Armor and Cape are folded into pullMessages.
	engine.RegisterBehavior("49004", &engine.Behavior{})
	engine.RegisterBehavior("49005", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		return []engine.Message{engine.GrantTrait{Target: e.EOwner(), Trait: "aerial"}}
	}})

	engine.RegisterBehavior("49006", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{Retaliate: 1} },
		DamagePrevention: func(g *engine.Game, u *engine.Upgrade, p *engine.Player, n int) (int, int) {
			u.Counters += n
			if u.Counters >= 6 {
				g.Delete(u.ID)
				p.Discard = append(p.Discard, engine.Card{ID: g.NextCardID(), Code: u.Code, Owner: p.ID})
			}
			return n, 0
		},
	})

	engine.RegisterBehavior("49007", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			var choices []engine.Choice
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn == nil || mn.EDef().HasTrait("elite") {
					continue
				}
				choices = append(choices, engine.Choice{Label: "Wrap " + mn.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id, CardCode: mn.Code}.
					Msgs(engine.AttachUpgrade{ID: e.EID(), Target: id}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Wrapped in Metal — choose a minion", choices...)}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AttachUpgrade)
			if !ok || m.ID != e.EID() {
				return nil
			}
			if mn := g.Minions[m.Target]; mn != nil {
				// The activation replacement hook cannot cancel an activation.
				// Zeroing printed SCH/ATK and blanking text is a persistent lock;
				// detach restoration is unavailable because upgrades lack OnDetach.
				mn.AttackVal, mn.SchemeVal, mn.BlankText = 0, 0, true
			}
			return nil
		},
	})

	engine.RegisterBehavior("49008", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		var choices []engine.Choice
		for _, id := range g.Schemes() {
			scheme := g.Entity(id)
			if scheme == nil {
				continue
			}
			msgs := []engine.Message{engine.ThwartScheme{Scheme: id, N: 3, Source: p.ID}}
			remaining := 4
			switch s := scheme.(type) {
			case *engine.SideScheme:
				remaining = s.Threat
			case *engine.MainScheme:
				remaining = s.Threat
			}
			choice := engine.Choice{Label: "Remove 3 threat from " + scheme.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id, CardCode: scheme.ECode()}.Msgs(msgs...)
			if remaining <= 3 {
				var attachments []engine.Choice
				for aid, a := range g.Attachments {
					if a == nil {
						continue
					}
					text := strings.ToLower(a.EDef().Text)
					if strings.Contains(text, "hero action") || strings.Contains(text, "hero response") {
						attachments = append(attachments, engine.Choice{Label: "Discard " + a.EDef().Name, Kind: engine.ChoiceCard, SourceID: aid, CardCode: a.Code}.
							Msgs(engine.DiscardAttachmentMsg{ID: aid}))
					}
				}
				if len(attachments) > 0 {
					choice = choice.WithThen(engine.Ask("Electromagnetic Blast — discard an attachment", attachments...))
				}
			}
			choices = append(choices, choice)
		}
		if len(choices) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Electromagnetic Blast — choose a scheme", choices...)}}
	}})

	engine.RegisterBehavior("49009", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		choices := cardutil.EnemyChoices(g, 7, p.ID, func(id engine.EntityID) []engine.Message {
			msgs := []engine.Message{engine.DamageEntity{Target: id, Damage: 7, Source: p.ID}}
			lethal := false
			switch target := g.Entity(id).(type) {
			case *engine.Minion:
				lethal = target.HP() <= 7
			case *engine.Villain:
				lethal = target.HP() <= 7
			}
			if lethal {
				msgs = append(msgs, engine.ToughEntity{Target: p.ID})
			}
			return msgs
		})
		if len(choices) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Metal Shards — choose an enemy", choices...)}}
	}})

	engine.RegisterBehavior("49010", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		var wrapped []engine.Choice
		for _, id := range cardutil.SortedIDs(g.Minions) {
			mn := g.Minions[id]
			if mn == nil || !wrappedInMetal(g, id) {
				continue
			}
			targets := cardutil.EnemyChoices(g, 5, p.ID, func(target engine.EntityID) []engine.Message {
				return []engine.Message{engine.DamageEntity{Target: id, Damage: mn.HP(), Source: p.ID}, engine.DamageEntity{Target: target, Damage: 5, Source: p.ID}, engine.StunEntity{Target: target}}
			})
			if len(targets) > 0 {
				wrapped = append(wrapped, engine.Choice{Label: "Discard " + mn.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id, CardCode: mn.Code}.
					WithThen(engine.Ask("Magnetic Missile — choose the target", targets...)))
			}
		}
		if len(wrapped) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Magnetic Missile — choose a wrapped minion", wrapped...)}}
	}})
	engine.RegisterBehavior("49011", &engine.Behavior{})
}

func wrappedInMetal(g *engine.Game, minion engine.EntityID) bool {
	for _, u := range g.Upgrades {
		if u != nil && u.Code == "49007" && u.AttachTo == minion {
			return true
		}
	}
	return false
}

func registerMagnetoObligation() {
	engine.RegisterBehavior("49027", &engine.Behavior{ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
		// Old Grievances is persistent and keys damage to a future named
		// ability resolution. With no obligation play area, resolve a minimum
		// one-card Pull penalty now, or let alter ego exhaust to remove it.
		choices := []engine.Choice{engine.Choice{ID: "damage", Label: "Take 1 damage and discard Old Grievances", Kind: engine.ChoiceLabel}.
			Msgs(engine.DamageEntity{Target: p.ID, Damage: 1, Source: p.ID}, engine.ObligationResolve{Player: p.ID, Card: card})}
		if !p.IsHero() && !p.Exhausted {
			choices = append(choices, engine.Choice{ID: "remove", Label: "Exhaust Erik Lehnsherr and discard Old Grievances", Kind: engine.ChoiceLabel}.
				Msgs(engine.ExhaustEntity{ID: p.ID}, engine.ObligationResolve{Player: p.ID, Card: card}))
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Old Grievances — choose", choices...)}}
	}})
}

func registerMagnetoNemesis() {
	engine.RegisterBehavior("49028", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.WindowAfterEnemyAttacked)
		mn := g.Minions[e.EID()]
		if !ok || mn == nil || m.Enemy != mn.ID {
			return nil
		}
		return []engine.Message{engine.MillPlayerDeck{Player: m.Player, N: max(0, mn.AttackVal)}}
	}})
	engine.RegisterBehavior("49029", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.SchemeDefeated)
		if !ok || m.Scheme != e.EID() || len(g.Players) == 0 {
			return nil
		}
		return []engine.Message{engine.MillPlayerDeck{Player: g.Players[0].ID, N: 9}}
	}})
	engine.RegisterBehavior("49030", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionDefeated)
			mn := g.Minions[e.EID()]
			if !ok || mn == nil || m.MinionID != mn.ID {
				return nil
			}
			return []engine.Message{engine.MillPlayerDeck{Player: mn.EngagedWith, N: 4}}
		},
		Boost: millFirstPlayer(4),
	})
	engine.RegisterBehavior("49031", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.WindowAfterEnemyAttacked)
			mn := g.Minions[e.EID()]
			if !ok || mn == nil || m.Enemy != mn.ID {
				return nil
			}
			return []engine.Message{engine.MillPlayerDeck{Player: m.Player, N: 2}}
		},
		Boost: millFirstPlayer(4),
	})
	engine.RegisterBehavior("49032", &engine.Behavior{ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		g.Delete(t.ID)
		var msgs []engine.Message
		for _, mn := range g.Minions {
			if mn != nil && mn.EngagedWith != "" && mn.EDef().HasTrait("acolyte") {
				msgs = append(msgs, engine.MinionActivates{MinionID: mn.ID, Player: mn.EngagedWith})
			}
		}
		if len(msgs) == 0 {
			// Encounter-deck search-until-minion is unavailable; surge is the
			// deterministic fallback for revealing another encounter card.
			msgs = append(msgs, engine.RevealNextEncounter{Player: p.ID})
		}
		return msgs
	}})
}

func millFirstPlayer(n int) func(*engine.Game, engine.Card) []engine.Message {
	return func(g *engine.Game, card engine.Card) []engine.Message {
		if len(g.Players) == 0 {
			return nil
		}
		return []engine.Message{engine.MillPlayerDeck{Player: g.Players[0].ID, N: n}}
	}
}
