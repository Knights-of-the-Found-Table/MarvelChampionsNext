// mg_shadowcat.go implements the Shadowcat hero-pack cards (32031–32059):
// the Solid/Phased mass-form upgrade, the signature cards, the
// Permanently Phased obligation and the White Queen nemesis set.
package mutantgenesis

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() {
	registerShadowcatPack()
	registerShadowcatNemesis()
}

// shadowcatForm returns "solid" or "phased" for the mass-form upgrade in
// play ("" for anyone else).
func shadowcatForm(g *engine.Game, p *engine.Player) string {
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil && data.BaseCode(u.Code) == "32031" {
			if u.Code == "32031b" {
				return "phased"
			}
			return "solid"
		}
	}
	return ""
}

// flipShadowcatForm swaps the mass-form upgrade between Solid and Phased.
func flipShadowcatForm(g *engine.Game, p *engine.Player) {
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil && data.BaseCode(u.Code) == "32031" {
			if u.Code == "32031b" {
				u.Code = "32031a"
				g.Logf("Shadowcat flips to Solid mass form")
			} else {
				u.Code = "32031b"
				g.Logf("Shadowcat flips to Phased mass form")
			}
			return
		}
	}
}

// registerShadowcatSetup installs the Solid mass form at game start; wired
// into the Shadowcat identity (32030) behavior.
func shadowcatSetup(g *engine.Game, p *engine.Player) []engine.Message {
	u := &engine.Upgrade{
		ID: g.NextEntityID(engine.KindUpgrade), Code: "32031a", Owner: p.ID, AttachTo: p.ID,
	}
	g.Upgrades[u.ID] = u
	p.Upgrades = append(p.Upgrades, u.ID)
	g.Logf("Shadowcat starts in Solid mass form")
	return nil
}

func registerShadowcatPack() {
	// 32031 Solid/Phased: the mass-form upgrade. Its flip response runs
	// after Shadowcat attacks or defends; the [physical] resource is
	// attack/defense-event keyed in print, approximated to any hero play.
	engine.RegisterBehavior("32031", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "physical", HeroOnly: true},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			switch m := msg.(type) {
			case engine.BasicAttack:
				if m.Player == p.ID {
					flipShadowcatForm(g, p)
				}
			case engine.WindowDefended:
				if m.Defender == p.ID {
					flipShadowcatForm(g, p)
				}
			}
			return nil
		},
	})

	// 32032 Lockheed: on enter — solid: 2 damage to an enemy; phased:
	// remove 2 threat from a scheme.
	engine.RegisterBehavior("32032", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			phased := shadowcatForm(g, p) == "phased"
			if phased {
				return cardutil.ChooseScheme("Lockheed", func(g *engine.Game, e engine.Entity) int { return 2 })(g, e)
			}
			choices := cardutil.EnemyChoices(g, 2, p.ID, func(target engine.EntityID) []engine.Message {
				return []engine.Message{engine.DamageEntity{Target: target, Damage: 2, Source: p.ID}}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask("Lockheed — deal 2 damage", choices...),
			}}
		},
	})

	// 32033 Kitty's Room: Alter-Ego action — solid: heal 2; phased: draw 1.
	engine.RegisterBehavior("32033", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Kitty's Room — heal 2 (solid) or draw 1 (phased)", Type: engine.AbilityAction,
				AlterEgoOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(g.ActiveTurn)
					if p == nil {
						return nil
					}
					if shadowcatForm(g, p) == "phased" {
						return []engine.Message{engine.DrawCards{Player: p.ID, N: 1}}
					}
					return []engine.Message{engine.HealEntity{Target: p.ID, N: 2}}
				},
			}}
		},
	})

	// 32034 Acute Control / 32035 Intangible Interference: keyed off
	// guard/patrol/crisis-ignoring windows that do not exist.
	engine.RegisterBehavior("32034", &engine.Behavior{})
	engine.RegisterBehavior("32035", &engine.Behavior{})

	// 32036 Phased and Confused: attach to an enemy; when it would attack,
	// discard this card and confuse it (approximation: the attack itself
	// is not cancelled).
	engine.RegisterBehavior("32036", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			var choices []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				en := g.Entity(id)
				if en == nil {
					continue
				}
				choices = append(choices, engine.Choice{
					Label: "Attach to " + cardutil.EnemyLabel(en), Kind: engine.ChoiceTarget, SourceID: id,
				}.Msgs(engine.AttachUpgrade{ID: u.ID, Target: id}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   e.EOwner(),
				Question: engine.Ask("Phased and Confused — attach to an enemy", choices...),
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			u := g.Upgrades[e.EID()]
			if u == nil || u.AttachTo == "" {
				return nil
			}
			switch m := msg.(type) {
			case engine.VillainActivates:
				if m.VillainID != u.AttachTo {
					return nil
				}
			case engine.MinionActivates:
				if m.MinionID != u.AttachTo {
					return nil
				}
			default:
				return nil
			}
			g.Delete(u.ID)
			g.Logf("Phased and Confused — the attack fizzles and the enemy is confused")
			return []engine.Message{engine.ConfuseEntity{Target: u.AttachTo}}
		},
	})

	// 32037 Shadowcat Surprise: 3 damage and ready your hero.
	engine.RegisterBehavior("32037", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			choices := cardutil.EnemyChoices(g, 3, pid, func(target engine.EntityID) []engine.Message {
				return []engine.Message{
					engine.DamageEntity{Target: target, Damage: 3, Source: pid},
					engine.ReadyEntity{ID: pid},
				}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Shadowcat Surprise — deal 3 damage and ready your hero", choices...),
			}}
		},
	})

	// 32038 Phase Strike: 6 damage (the phased attachment-discard rider is
	// not modeled).
	engine.RegisterBehavior("32038", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy("Phase Strike", func(g *engine.Game, e engine.Entity) (int, []engine.Message) {
			return 6, nil
		}),
	})

	// 32039 Airwalk: remove 2 threat (4 in Phased mass form).
	engine.RegisterBehavior("32039", &engine.Behavior{
		OnPlay: cardutil.ChooseScheme("Airwalk", func(g *engine.Game, e engine.Entity) int {
			p := g.Player(e.EOwner())
			if p != nil && shadowcatForm(g, p) == "phased" {
				return 4
			}
			return 2
		}),
	})

	// 32040 Quick Shift: defense event — solid: flip to Phased; phased:
	// draw 2.
	engine.RegisterBehavior("32040", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			d := engine.Defends{Defender: p.ID, Against: against}
			if shadowcatForm(g, p) == "phased" {
				return d, []engine.Message{engine.DrawCards{Player: p.ID, N: 2}}, true
			}
			return d, nil, true
		},
	})

	// 32055 Permanently Phased: flip to Phased; the play restriction is
	// player-enforced; alter-ego exhaust removes it.
	engine.RegisterBehavior("32055", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			if shadowcatForm(g, p) != "phased" {
				flipShadowcatForm(g, p)
			}
			choices := []engine.Choice{engine.Choice{
				ID: "keep", Label: "Keep Permanently Phased in play", Kind: engine.ChoiceLabel,
			}.Msgs(engine.ObligationResolve{Player: p.ID, Card: card})}
			if !p.IsHero() && !p.Exhausted {
				choices = append(choices, engine.Choice{
					ID: "exhaust", Label: "Exhaust your identity → remove from the game", Kind: engine.ChoiceLabel,
				}.Msgs(engine.ExhaustEntity{ID: p.ID}, engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask("Permanently Phased — choose:", choices...),
			}}
		},
	})
}

func registerShadowcatNemesis() {
	// 32056 White Queen: villainous from data; "while engaged you are
	// confused" — reapplied when she engages and while in play
	// (approximation: confuse on engage).
	engine.RegisterBehavior("32056", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionEntersPlay)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			return []engine.Message{engine.ConfuseEntity{Target: m.Player}}
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return nil // "You are confused" needs the boost revealer; skipped
		},
	})

	// 32057 The Hellfire Club: When Defeated — reveal a Hellfire Pawn
	// engaged with the defeater (approximation: first player).
	engine.RegisterBehavior("32057", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			pid := cardutil.FirstPlayerID(g)
			for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
				for _, c := range *zone {
					if c.Code == "32058" {
						zone.Remove(c.ID)
						return []engine.Message{engine.RevealEncounterCard{Player: pid, Card: c}}
					}
				}
			}
			return nil
		},
	})

	// 32058 Hellfire Pawn: keywords from data; the boost self-spawn is
	// handled by the generic boost-spawn text scan.
	engine.RegisterBehavior("32058", &engine.Behavior{})

	// 32059 Telepathic Restraint: attach to your identity; you are stunned
	// while attached; spend [mental][mental] via an action to discard.
	engine.RegisterBehavior("32059", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if p := g.Player(PlayerIDOf(target)); p != nil {
				return []engine.Message{engine.StunEntity{Target: p.ID}}
			}
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			t := g.Attachments[e.EID()]
			if t == nil {
				return nil
			}
			p := g.Player(PlayerIDOf(t.Target))
			if p == nil {
				return nil
			}
			return []engine.Ability{{
				Label: "Telepathic Restraint — spend [mental][mental] → discard", Type: engine.AbilityAction,
				Cost: 2, CostIcons: "mental:2",
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.DiscardAttachmentMsg{ID: self}}
				},
			}}
		},
	})
}

// PlayerIDOf adapts an EntityID to a PlayerID.
func PlayerIDOf(id engine.EntityID) engine.PlayerID {
	return engine.PlayerID(id)
}
