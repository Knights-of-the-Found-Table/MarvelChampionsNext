package captainamerica

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// registerSignatures installs Captain America's signature cards
// (03002-03010).
func registerSignatures() {
	registerAgent13()
	registerFearlessDetermination()
	registerHeroicStrike()
	registerShieldBlock()
	registerShieldToss()
	registerStevesApartment()
	registerHelmet()
	registerShield()
	registerSuperSoldierSerum()
}

// 03002 Agent 13: Response — after she enters play, remove 2 threat from a
// scheme.
func registerAgent13() {
	engine.RegisterBehavior("03002", &engine.Behavior{
		OnPlay: cardutil.ChooseScheme(engine.Tf("c.chooseAScheme", "Agent 13"), func(g *engine.Game, e engine.Entity) int {
			return 2
		}),
	})
}

// 03003 Fearless Determination: Captain America gets +1 THW until the end
// of the phase. Draw 1 card.
func registerFearlessDetermination() {
	engine.RegisterBehavior("03003", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			if p := g.Player(pid); p != nil {
				p.BonusTHW++
				g.TLogf("c.gets1ThwUntilTheEndOfThePhase", p.Name)
			}
			return []engine.Message{engine.DrawCards{Player: pid, N: 1}}
		},
	})
}

// 03004 Heroic Strike: deal 6 damage to an enemy; stun it if paid with a
// [physical] resource.
func registerHeroicStrike() {
	engine.RegisterBehavior("03004", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			physical := false
			if ec, ok := e.(*engine.EventCard); ok {
				physical = ec.Paid.PaidIcon("physical")
			}
			choices := cardutil.EnemyChoices(g, 6, pid, func(target engine.EntityID) []engine.Message {
				msgs := []engine.Message{engine.DamageEntity{Target: target, Damage: 6, Source: pid}}
				if physical {
					msgs = append(msgs, engine.StunEntity{Target: target})
				}
				return msgs
			})
			if len(choices) == 0 {
				return nil
			}
			prompt := "Heroic Strike — deal 6 damage"
			if physical {
				prompt += " and stun"
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.S(prompt), choices...),
			}}
		},
	})
}

// 03005 Shield Block: when you would take any amount of damage, exhaust
// Captain America's Shield → prevent all of that damage (approximation:
// offered in the defense prompt).
func registerShieldBlock() {
	engine.RegisterBehavior("03005", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			shield := findShield(g, p)
			if shield == "" {
				return engine.Defends{}, nil, false
			}
			d := engine.Defends{
				Defender: p.ID, Against: against,
				Undefended: true, PreventAll: true,
			}
			return d, []engine.Message{engine.ExhaustEntity{ID: shield}}, true
		},
	})
}

func findShield(g *engine.Game, p *engine.Player) engine.EntityID {
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil && u.Code == "03009" && !u.Exhausted {
			return u.ID
		}
	}
	return ""
}

// 03006 Shield Toss: discard X cards from your hand, then return Captain
// America's Shield from play to your hand → deal 4 damage to X enemies.
// Implemented as a nested choose-one tree: enemy, discard card, enemy,
// discard card...; all effects resolve when the player finishes.
func registerShieldToss() {
	engine.RegisterBehavior("03006", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			var shield engine.EntityID
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil && u.Code == "03009" {
					shield = u.ID
				}
			}
			if shield == "" {
				g.TLogf("c.shieldTossNeedsCaptainAmericaSShieldInPlay")
				return nil
			}
			if len(g.Enemies()) == 0 || len(p.Hand) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: shieldTossEnemyStep(g, p, shield, nil, nil),
			}}
		},
	})
}

// shieldTossEnemyStep asks for the next enemy to hit (or finish).
func shieldTossEnemyStep(g *engine.Game, p *engine.Player, shield engine.EntityID, cards []engine.Card, enemies []engine.EntityID) *engine.Question {
	picked := map[engine.EntityID]bool{}
	for _, en := range enemies {
		picked[en] = true
	}
	var choices []engine.Choice
	choices = append(choices, engine.Choice{
		ID: "done", Label: engine.Tf("c.finishShieldTossDamageQueued", 4*len(enemies)), Kind: engine.ChoicePass,
	}.Msgs(shieldTossEffects(p, shield, cards, enemies)...))
	for _, id := range cardutil.SortedEnemyIDs(g) {
		if picked[id] {
			continue
		}
		enemy := g.Entity(id)
		choices = append(choices, engine.Choice{
			Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget,
			SourceID: id, CardCode: enemy.ECode(),
		}.WithThen(shieldTossCardStep(g, p, shield, cards, append(enemies, id))))
	}
	return engine.Ask(engine.Tf("c.shieldTossChooseAnEnemyToHitFor4"), choices...)
}

// shieldTossCardStep asks which card to discard for the just-picked enemy.
func shieldTossCardStep(g *engine.Game, p *engine.Player, shield engine.EntityID, cards []engine.Card, enemies []engine.EntityID) *engine.Question {
	used := map[string]bool{}
	for _, c := range cards {
		used[c.ID] = true
	}
	var choices []engine.Choice
	for _, c := range p.Hand {
		if used[c.ID] {
			continue
		}
		choices = append(choices, engine.Choice{
			Label: engine.Tf("m.discardCard", c), Kind: engine.ChoiceCard, CardCode: c.Code,
		}.WithThen(shieldTossEnemyStep(g, p, shield, append(cards, c), enemies)))
	}
	// A discard is mandatory per enemy; if the hand ran dry the chain
	// simply ends here with the accumulated effects.
	choices = append(choices, engine.Choice{
		ID: "done", Label: engine.Tf("c.finishNoMoreCardsToDiscard"), Kind: engine.ChoicePass,
	}.Msgs(shieldTossEffects(p, shield, cards, enemies)...))
	return engine.Ask(engine.Tf("c.shieldTossDiscardACardForThisEnemy"), choices...)
}

func shieldTossEffects(p *engine.Player, shield engine.EntityID, cards []engine.Card, enemies []engine.EntityID) []engine.Message {
	var msgs []engine.Message
	for _, c := range cards {
		msgs = append(msgs, engine.DiscardCards{Player: p.ID, Cards: engine.CardList{c}})
	}
	for _, en := range enemies {
		msgs = append(msgs, engine.DamageEntity{Target: en, Damage: 4, Source: p.ID})
	}
	msgs = append(msgs, engine.ReturnControlled{Player: p.ID, ID: shield})
	return msgs
}

// 03007 Steve's Apartment: Alter-Ego Action — exhaust → draw 1 card and
// heal 1 damage from Steve Rogers.
func registerStevesApartment() {
	engine.RegisterBehavior("03007", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label:        engine.Tf("c.exhaustSteveSApartmentDraw1CardAndHeal1Damage"),
				Type:         engine.AbilityAction,
				Exhaust:      true,
				AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{
						engine.DrawCards{Player: e.EOwner(), N: 1},
						engine.HealEntity{Target: e.EOwner(), N: 1},
					}
				},
			}}
		},
	})
}

// 03008 Captain America's Helmet: Interrupt — when Captain America would
// be defeated, set his hit point dial to 1 instead. Then discard this card.
func registerHelmet() {
	engine.RegisterBehavior("03008", &engine.Behavior{
		DefeatSave: func(g *engine.Game, p *engine.Player, u *engine.Upgrade) bool {
			p.Damage = p.MaxHP - 1
			g.Delete(u.ID)
			p.Discard = append(p.Discard, engine.Card{ID: g.NextCardID(), Code: u.Code, Owner: p.ID})
			g.TLogf("c.captainAmericaSHelmetSaves1HpTheHelmetIsDiscarded", p.Name)
			return true
		},
	})
}

// 03009 Captain America's Shield: Restricted. Captain America gets +1 DEF
// and gains retaliate 1. (Restricted limit not enforced.)
func registerShield() {
	engine.RegisterBehavior("03009", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{DEF: 1, Retaliate: 1}
		},
	})
}

// 03010 Super-Soldier Serum: Resource — exhaust → generate a [physical]
// resource.
func registerSuperSoldierSerum() {
	engine.RegisterBehavior("03010", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "physical"},
	})
}
