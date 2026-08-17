// Package daredevil registers the Daredevil hero from Fear No Evil: the
// Superhuman Senses mechanic, the signature cards and the Bullseye
// nemesis set.
package daredevil

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerDaredevil()
	registerSenses()
	registerNemesis()
	registerSignatures()
	registerDeckCards()
}

// registerDaredevil installs the identity (60001a/b).
func registerDaredevil() {
	engine.RegisterBehavior("60001", &engine.Behavior{
		// Matt Murdock begins the game with a Sense deck: one copy of each
		// Sense upgrade from the signature set.
		HeroSetup: func(g *engine.Game, p *engine.Player) []engine.Message {
			for _, def := range engine.DB.InSet("daredevil_sense_deck") {
				if def.Type == "upgrade" && def.HasTrait("sense") {
					p.SenseDeck = append(p.SenseDeck, engine.Card{ID: g.NextCardID(), Code: def.Code, Owner: p.ID})
				}
			}
			g.Logf("%s begins the game with a Sense deck of %d cards", p.Name, len(p.SenseDeck))
			return nil
		},
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			if len(p.SenseDeck) == 0 {
				return nil
			}
			return []engine.Ability{{
				// Superhuman Senses — Action: play the top card of the
				// Sense deck as if it were in your hand (paying its cost).
				Label:    fmt.Sprintf("Superhuman Senses — play the top Sense card (%s)", p.SenseDeck[0].Def().Name),
				Type:     engine.AbilityAction,
				HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(self)
					if p == nil || len(p.SenseDeck) == 0 {
						return nil
					}
					top := p.SenseDeck[0]
					cost := cardutil.Cost(top.Def())
					play := engine.Choice{
						ID: "play", Label: fmt.Sprintf("Play %s (cost %d)", top.Def().Name, cost),
						Kind: engine.ChoiceCard, CardCode: top.Code,
					}
					if cost > 0 {
						play = play.WithThen(g.CustomPaymentQuestion(p, cost,
							fmt.Sprintf("Pay %d resources for %s", cost, top.Def().Name),
							map[string]any{"senseCard": top.ID}))
					} else {
						play = play.Msgs(engine.SenseEnterPlay{Player: p.ID, Card: top})
					}
					return []engine.Message{engine.AskQuestion{
						Player:   p.ID,
						Question: engine.Ask("Superhuman Senses", play, cardutil.Skip()),
					}}
				},
			}}
		},
		// Matt Murdock: Sense upgrades leaving play return to the bottom
		// of the Sense deck (handled generically in discardControlled).
	})
}

// registerSenses installs the five Sense upgrades (60002-60006).
func registerSenses() {
	// 60002 Acute Tactility: when you defeat the attached enemy or remove
	// the last threat from the attached scheme, discard → ready your
	// identity.
	engine.RegisterBehavior("60002", &engine.Behavior{
		OnPlay: senseAttach(true, true),
		React:  senseDefeatTrigger("60002", func(g *engine.Game, p *engine.Player, u *engine.Upgrade) []engine.Message {
			return []engine.Message{engine.ReadyEntity{ID: p.ID}}
		}),
	})

	// 60003 Enhanced Olfaction: same trigger, discard → the next card you
	// play this phase costs 2 less.
	engine.RegisterBehavior("60003", &engine.Behavior{
		OnPlay: senseAttach(true, true),
		React:  senseDefeatTrigger("60003", func(g *engine.Game, p *engine.Player, u *engine.Upgrade) []engine.Message {
			p.CostDiscounts = append(p.CostDiscounts, engine.CostDiscount{Amount: 2})
			g.Logf("%s's next card this phase costs 2 less", p.Name)
			return nil
		}),
	})

	// 60004 Heightened Hearing: attached enemy's attack gets -3 ATK
	// (engine auto-consumes via AttachedEnemyAttackMod).
	engine.RegisterBehavior("60004", &engine.Behavior{
		OnPlay:                   senseAttach(true, false),
		AttachedEnemyAttackMod: -3,
	})

	// 60005 Radar Sense: after you attack the attached enemy, discard →
	// deal 3 damage to it.
	engine.RegisterBehavior("60005", &engine.Behavior{
		OnPlay: senseAttach(true, false),
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.DamageEntity)
			u, ok2 := e.(*engine.Upgrade)
			if !ok || !ok2 || d.Target != u.AttachTo || d.Source != u.Owner {
				return nil
			}
			g.Logf("Radar Sense: +3 damage")
			return []engine.Message{
				engine.DiscardControlled{Player: u.Owner, ID: u.ID},
				engine.DamageEntity{Target: u.AttachTo, Damage: 3, Source: u.ID},
			}
		},
	})

	// 60006 Superior Taste: after you thwart the attached scheme, discard
	// → remove 2 threat from it.
	engine.RegisterBehavior("60006", &engine.Behavior{
		OnPlay: senseAttach(false, true),
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowAfterThwarted)
			u, ok2 := e.(*engine.Upgrade)
			if !ok || !ok2 || w.Scheme != u.AttachTo || w.Player != u.Owner {
				return nil
			}
			g.Logf("Superior Taste: 2 additional threat removed")
			return []engine.Message{
				engine.DiscardControlled{Player: u.Owner, ID: u.ID},
				engine.ThwartScheme{Scheme: u.AttachTo, N: 2, Source: u.Owner},
			}
		},
	})
}

// senseAttach builds the attach-target question for a Sense upgrade.
func senseAttach(enemies, schemes bool) func(g *engine.Game, e engine.Entity) []engine.Message {
	return func(g *engine.Game, e engine.Entity) []engine.Message {
		pid := e.EOwner()
		var choices []engine.Choice
		if enemies {
			for _, id := range cardutil.SortedEnemyIDs(g) {
				enemy := g.Entity(id)
				choices = append(choices, engine.Choice{
					Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: enemy.ECode(),
				}.Msgs(engine.AttachUpgrade{ID: e.EID(), Target: id}))
			}
		}
		if schemes {
			for _, id := range g.Schemes() {
				s := g.Entity(id)
				choices = append(choices, engine.Choice{
					Label: s.EDef().Name, Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: s.ECode(),
				}.Msgs(engine.AttachUpgrade{ID: e.EID(), Target: id}))
			}
		}
		if len(choices) == 0 {
			// nothing to attach to: back to the Sense deck.
			return []engine.Message{engine.DiscardControlled{Player: pid, ID: e.EID()}}
		}
		return []engine.Message{engine.AskQuestion{
			Player:   pid,
			Question: engine.Ask(e.EDef().Name+" — attach to", choices...),
		}}
	}
}

// senseDefeatTrigger builds the "when you defeat the attached enemy or
// remove the last threat from the attached scheme" reaction.
func senseDefeatTrigger(code string, effect func(g *engine.Game, p *engine.Player, u *engine.Upgrade) []engine.Message) func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
	return func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		u, ok := e.(*engine.Upgrade)
		if !ok {
			return nil
		}
		hit := false
		switch m := msg.(type) {
		case engine.MinionDefeated:
			hit = m.MinionID == u.AttachTo
		case engine.VillainDefeated:
			hit = m.VillainID == u.AttachTo
		case engine.SchemeDefeated:
			hit = m.Scheme == u.AttachTo
		}
		if !hit {
			return nil
		}
		p := g.Player(u.Owner)
		if p == nil {
			return nil
		}
		name := u.EDef().Name
		var effectMsgs []engine.Message
		effectMsgs = append(effectMsgs, engine.DiscardControlled{Player: u.Owner, ID: u.ID})
		effectMsgs = append(effectMsgs, effect(g, p, u)...)
		return []engine.Message{engine.AskQuestion{
			Player: u.Owner,
			Question: engine.Ask(name + " — discard it for its effect?",
				engine.Choice{
					ID: "use", Label: "Discard " + name + " → use its effect", Kind: engine.ChoiceLabel,
				}.Msgs(effectMsgs...),
				engine.Choice{ID: "skip", Label: "Skip", Kind: engine.ChoicePass},
			),
		}}
	}
}

// upgradesAttachedTo counts upgrades attached to an entity.
func upgradesAttachedTo(g *engine.Game, target engine.EntityID) int {
	n := 0
	for _, u := range g.Upgrades {
		if u.AttachTo == target {
			n++
		}
	}
	return n
}

// registerNemesis installs the Bullseye nemesis encounter set.
func registerNemesis() {
	// 60033 Bullseye: When Revealed — discard a Persona support you
	// control, or Bullseye attacks you (even in alter-ego form).
	engine.RegisterBehavior("60033", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn, ok := e.(*engine.Minion)
			if !ok || mn.EngagedWith == "" {
				return nil
			}
			p := g.Player(mn.EngagedWith)
			if p == nil {
				return nil
			}
			var discard []engine.Choice
			for _, id := range p.Supports {
				if s := g.Supports[id]; s != nil && s.EDef().HasTrait("persona") {
					discard = append(discard, engine.Choice{
						Label: "Discard " + s.EDef().Name, Kind: engine.ChoiceCard, CardCode: s.Code,
					}.Msgs(engine.DiscardControlled{Player: p.ID, ID: id}))
				}
			}
			attack := engine.Choice{
				ID: "attack", Label: "Bullseye attacks you", Kind: engine.ChoiceLabel,
			}.Msgs(engine.MinionActivates{MinionID: mn.ID, Player: p.ID})
			if len(discard) == 0 {
				return []engine.Message{engine.MinionActivates{MinionID: mn.ID, Player: p.ID}}
			}
			return []engine.Message{engine.AskQuestion{
				Player: p.ID,
				Question: engine.Ask("Bullseye — discard a Persona support or take the attack?",
					append(discard, attack)...),
			}}
		},
	})

	// 60034 Deadliest Man Alive: approximation — when revealed, Bullseye
	// permanently gets +1 ATK (the facedown-boost rider is not modeled
	// for minion attacks).
	engine.RegisterBehavior("60034", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for id, mn := range g.Minions {
				if mn.Code == "60033" {
					g.Logf("Deadliest Man Alive: Bullseye gets +1 ATK")
					return []engine.Message{engine.BoostEnemyAttack{Enemy: id, N: 1}}
				}
			}
			return nil
		},
	})

	// 60035 Stolen Sai: attach to the enemy with the highest ATK
	// (piercing not modeled); hero action — spend 2 [physical] → discard.
	engine.RegisterBehavior("60035", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			var best engine.EntityID
			bestATK := -1
			for id, v := range g.Villains {
				if v.AttackVal > bestATK {
					best, bestATK = id, v.AttackVal
				}
			}
			for id, mn := range g.Minions {
				if mn.AttackVal > bestATK {
					best, bestATK = id, mn.AttackVal
				}
			}
			if best != "" {
				t.Target = best
				if e := g.Entity(best); e != nil {
					g.Logf("Stolen Sai attaches to %s (piercing not modeled)", e.EDef().Name)
				}
			}
			return nil
		},
	})

	// 60036 Eye on the Target: When Revealed — remove an ally or Persona
	// support you control from the game, or Bullseye attacks you (reveal
	// his set if he is not in play).
	engine.RegisterBehavior("60036", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			s, ok := e.(*engine.SideScheme)
			_ = s
			if !ok {
				return nil
			}
			// side schemes revealed against a player: use the first
			// player (approximation for nemesis reveals in multiplayer).
			p := g.Player(cardutil.FirstPlayerID(g))
			if p == nil {
				return nil
			}
			var rfg []engine.Choice
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil {
					rfg = append(rfg, engine.Choice{
						Label: "Remove " + a.EDef().Name, Kind: engine.ChoiceCard, CardCode: a.Code,
					}.Msgs(engine.DiscardControlled{Player: p.ID, ID: id}))
				}
			}
			for _, id := range p.Supports {
				if sp := g.Supports[id]; sp != nil && sp.EDef().HasTrait("persona") {
					rfg = append(rfg, engine.Choice{
						Label: "Remove " + sp.EDef().Name, Kind: engine.ChoiceCard, CardCode: sp.Code,
					}.Msgs(engine.DiscardControlled{Player: p.ID, ID: id}))
				}
			}
			attack := func() []engine.Message {
				for id, mn := range g.Minions {
					if mn.Code == "60033" {
						return []engine.Message{engine.MinionActivates{MinionID: id, Player: p.ID}}
					}
				}
				return []engine.Message{engine.RevealNemesisSet{Player: p.ID}}
			}()
			if len(rfg) == 0 {
				return attack
			}
			rfg = append(rfg, engine.Choice{
				ID: "attack", Label: "Bullseye attacks you instead", Kind: engine.ChoiceLabel,
			}.Msgs(attack...))
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask("Eye on the Target — remove an ally or Persona support from the game?", rfg...),
			}}
		},
	})
}
