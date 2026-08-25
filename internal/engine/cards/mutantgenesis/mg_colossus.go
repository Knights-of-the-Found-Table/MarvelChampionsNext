// mg_colossus.go implements the Colossus hero-pack cards (32002–32029):
// signature allies/supports/upgrades/events, the Homesick obligation and
// the Juggernaut nemesis set.
package mutantgenesis

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() {
	registerColossusPack()
	registerColossusNemesis()
}

func registerColossusPack() {
	// 32002 Shadowcat ally: guard/patrol/crisis immunity has no engine
	// hook; she plays as a plain 2/2 X-Men ally.
	engine.RegisterBehavior("32002", &engine.Behavior{})

	// 32003 Piotr's Studio: Alter-Ego action — exhaust → mill until a
	// Colossus card, add it to hand.
	engine.RegisterBehavior("32003", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Piotr's Studio — dig for a Colossus card", Type: engine.AbilityAction,
				AlterEgoOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(g.ActiveTurn)
					if p == nil || len(p.Deck) == 0 {
						return nil
					}
					for i, c := range p.Deck {
						if c.Def().CardSet == "colossus" {
							p.Deck = append(p.Deck[:i], p.Deck[i+1:]...)
							c.Owner = p.ID
							p.Hand = append(p.Hand, c)
							g.Logf("Piotr's Studio finds %s", c.Def().Name)
							return nil
						}
						p.Discard = append(p.Discard, c)
					}
					p.Deck = nil
					return nil
				},
			}}
		},
	})

	// 32004 Iron Will: +1 THW; after a tough status card is discarded from
	// Colossus, draw 1 card (the tough-discard window is approximated by
	// reacting to incoming damage while tough).
	engine.RegisterBehavior("32004", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{THW: 1} },
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.DamageEntity)
			p := g.Player(e.EOwner())
			if !ok || p == nil || m.Target != p.ID || m.Damage <= 0 || p.Tough == 0 {
				return nil
			}
			return []engine.Message{engine.DrawCards{Player: p.ID, N: 1}}
		},
	})

	// 32005 Titanium Muscles: +1 ATK; exhaust → [physical] (the
	// per-tough multiplier is moot with boolean tough tracking).
	engine.RegisterBehavior("32005", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{ATK: 1} },
		Resource:      &engine.ResourceAbility{Icon: "physical", HeroOnly: true},
	})

	// 32006 Organic Steel: 2 steel counters; after a tough is discarded,
	// exhaust + counter → regain tough.
	engine.RegisterBehavior("32006", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 2}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.DamageEntity)
			u := g.Upgrades[e.EID()]
			p := g.Player(e.EOwner())
			if !ok || u == nil || p == nil || m.Target != p.ID || m.Damage <= 0 || p.Tough == 0 {
				return nil
			}
			if u.Exhausted || u.Counters <= 0 {
				return nil
			}
			g.Logf("Organic Steel — Colossus regains his tough status card")
			return []engine.Message{
				engine.ExhaustEntity{ID: u.ID},
				engine.AddEntityCounter{ID: u.ID, N: -1},
				engine.ToughEntity{Target: p.ID},
			}
		},
	})

	// 32007 Made of Rage: when you make a basic attack, discard a tough
	// → +6 damage for that attack (follow-up damage, the Huge Wooden
	// Hammer precedent; overkill not modeled).
	engine.RegisterBehavior("32007", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.BasicAttack)
			p := g.Player(e.EOwner())
			if !ok || p == nil || m.Player != p.ID || p.Tough == 0 || !p.IsHero() {
				return nil
			}
			p.Tough = 0
			g.Logf("Made of Rage — +6 damage for this attack")
			return []engine.Message{engine.DamageEntity{Target: m.Target, Damage: 6, Source: p.ID}}
		},
	})

	// 32008 Steel Fist: 5 damage; optionally discard a tough → stun and
	// confuse.
	engine.RegisterBehavior("32008", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			rider := p.Tough > 0
			var choices []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				en := g.Entity(id)
				if en == nil {
					continue
				}
				label := "Deal 5 damage to " + cardutil.EnemyLabel(en)
				msgs := []engine.Message{engine.DamageEntity{Target: id, Damage: 5, Source: p.ID}}
				if rider {
					label += " (discard a tough → stun and confuse)"
					msgs = append(msgs, engine.StunEntity{Target: id}, engine.ConfuseEntity{Target: id})
				}
				choices = append(choices, engine.Choice{
					Label: label, Kind: engine.ChoiceTarget, SourceID: id,
				}.Msgs(msgs...))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask("Steel Fist", choices...),
			}}
		},
	})

	// 32009 Bulletproof Protector: discard a tough → 2 tough status cards
	// (boolean approximation: one tough) or ready your hero.
	engine.RegisterBehavior("32009", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool { return p.Tough > 0 },
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player: p.ID,
				Question: engine.Ask("Bulletproof Protector — choose:",
					engine.Choice{
						ID: "tough", Label: "Gain tough status cards (boolean approximation)", Kind: engine.ChoiceLabel,
					}.Msgs(engine.ToughEntity{Target: p.ID}),
					engine.Choice{
						ID: "ready", Label: "Ready your hero", Kind: engine.ChoiceLabel,
					}.Msgs(engine.ReadyEntity{ID: p.ID}),
				),
			}}
		},
	})

	// 32010 Armor Up: the alter-ego villain-activation interrupt has no
	// event window from hand; kept as a plain card.
	engine.RegisterBehavior("32010", &engine.Behavior{})

	// 32025 Homesick: exhaust to remove, or discard your toughs (surge if
	// none).
	engine.RegisterBehavior("32025", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			choices := []engine.Choice{engine.Choice{
				ID: "exhaust", Label: "Exhaust Piotr Rasputin → remove from the game", Kind: engine.ChoiceLabel,
			}.Msgs(engine.ExhaustEntity{ID: p.ID}, engine.ObligationResolve{Player: p.ID, Card: card, Remove: true})}
			penalty := []engine.Message{}
			if p.Tough == 0 {
				penalty = append(penalty, engine.RevealNextEncounter{Player: p.ID})
			}
			penalty = append(penalty, engine.ObligationResolve{Player: p.ID, Card: card})
			choices = append(choices, engine.Choice{
				ID: "discard", Label: "Discard this card and each tough status card from your identity", Kind: engine.ChoiceLabel,
			}.Msgs(penalty...))
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask("Homesick — choose:", choices...),
			}}
		},
	})
}

func registerColossusNemesis() {
	// 32026 Juggernaut: stalwart/toughness from data; the boost overkill
	// and piercing rider is not modeled.
	engine.RegisterBehavior("32026", &engine.Behavior{})

	// 32027 Rampaging Juggernaut: When Revealed — discard every tough
	// from friendly characters; 2 threat here per tough discarded.
	engine.RegisterBehavior("32027", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			n := 0
			for _, p := range g.Players {
				if p.Tough > 0 {
					n += p.Tough
					p.Tough = 0
				}
				for _, id := range p.Allies {
					if a := g.Allies[id]; a != nil && a.Tough {
						a.Tough = false
						n++
					}
				}
			}
			if n > 0 {
				g.Logf("Rampaging Juggernaut discards %d tough status cards", n)
				return []engine.Message{engine.SchemeThreat{Scheme: e.EID(), N: 2 * n, Source: e.EID()}}
			}
			return nil
		},
	})

	// 32028 Unstoppable: attach to the highest-ATK enemy; the
	// overkill/piercing rider and end-of-attack discard are not modeled.
	engine.RegisterBehavior("32028", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			var best *engine.Minion
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn == nil {
					continue
				}
				attached := false
				for _, aid := range mn.Attachments {
					if a := g.Attachments[aid]; a != nil && a.Code == "32028" {
						attached = true
					}
				}
				if attached {
					continue
				}
				if best == nil || mn.AttackVal > best.AttackVal {
					best = mn
				}
			}
			if best == nil {
				for _, id := range cardutil.SortedIDs(g.Villains) {
					if v := g.Villains[id]; v != nil {
						t.Target = id
						return nil
					}
				}
				g.Delete(t.ID)
				return nil
			}
			t.Target = best.ID
			best.Attachments = append(best.Attachments, t.ID)
			return nil
		},
	})

	// 32029 Slammed: stunned; already stunned → 2 damage; the "Boost:
	// reveal this card" rider is not modeled.
	engine.RegisterBehavior("32029", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			if p.Stunned {
				return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 2, Source: t.ID}}
			}
			return []engine.Message{engine.StunEntity{Target: p.ID}}
		},
	})
}
