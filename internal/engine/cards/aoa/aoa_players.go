package aoa

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func registerAoaPlayerCards() {
	// 45011 Cable: after his thwart defeats a side scheme, draw 1.
	engine.RegisterBehavior("45011", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyThwartWindow)
			if !ok || m.Ally != e.EID() {
				return nil
			}
			a := g.Allies[e.EID()]
			if a == nil {
				return nil
			}
			if s := g.SideSchemes[m.Scheme]; s != nil && s.Threat <= a.ThwartVal+a.BonusTHW+a.PermTHW {
				g.Logf("Cable's thwart defeats %s — draw 1", s.EDef().Name)
				return []engine.Message{engine.DrawCards{Player: a.Owner, N: 1}}
			}
			return nil
		},
	})

	// 45012 X-23: after her attack would defeat an enemy, ready her.
	engine.RegisterBehavior("45012", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyAttackWindow)
			if !ok || m.Ally != e.EID() {
				return nil
			}
			enemy := g.Entity(m.Target)
			a := g.Allies[e.EID()]
			if enemy == nil || a == nil {
				return nil
			}
			var hp int
			switch t := enemy.(type) {
			case *engine.Villain:
				hp = t.HP()
			case *engine.Minion:
				hp = t.HP()
			default:
				return nil
			}
			if hp <= a.AttackVal+a.BonusATK+a.PermATK {
				g.Logf("X-23 readies after the kill")
				return []engine.Message{engine.ReadyEntity{ID: a.ID}}
			}
			return nil
		},
	})

	// 45013 Team Training: each ally you control gets +1 hit point.
	engine.RegisterBehavior("45013", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			applyTeamTraining(g, e)
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.AllyEnteredPlay); ok {
				applyTeamTraining(g, e)
			}
			return nil
		},
	})

	// 45014 Advanced Suit: attach to an X-FORCE/X-MEN ally (the healing
	// response is approximated away).
	engine.RegisterBehavior("45014", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			u := g.Upgrades[e.EID()]
			p := g.Player(u.Owner)
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a == nil || !a.EDef().HasTrait("X-Force") && !a.EDef().HasTrait("X-Men") {
					continue
				}
				choices = append(choices, engine.Choice{
					ID: "ally-" + id.String(), Label: a.EDef().Name, Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: a.Code,
				}.Msgs(engine.AttachUpgrade{ID: u.ID, Target: id, MaxHP: 1}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask("Advanced Suit — attach to:", choices...),
			}}
		},
	})

	// 45015 Sidekick: +2 HP on an identity-specific ally; heals 2 after
	// your basic recovery.
	engine.RegisterBehavior("45015", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			u := g.Upgrades[e.EID()]
			p := g.Player(u.Owner)
			if p == nil {
				return nil
			}
			heroSet := ""
			if d, ok := engine.DB.Lookup(p.HeroCode); ok {
				heroSet = d.CardSet
			}
			var choices []engine.Choice
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a == nil || a.EDef().CardSet != heroSet {
					continue
				}
				choices = append(choices, engine.Choice{
					ID: "ally-" + id.String(), Label: a.EDef().Name, Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: a.Code,
				}.Msgs(engine.AttachUpgrade{ID: u.ID, Target: id, MaxHP: 2}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask("Sidekick — attach to:", choices...),
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.BasicRecover)
			if !ok {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if u == nil || u.Owner != m.Player || u.AttachTo == "" {
				return nil
			}
			return []engine.Message{engine.HealEntity{Target: u.AttachTo, N: 2}}
		},
	})

	// 45016 Side-by-Side: ready hero (and sidekick) + heal or buff.
	engine.RegisterBehavior("45016", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var msgs []engine.Message
			msgs = append(msgs, engine.ReadyEntity{ID: p.ID})
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil {
					msgs = append(msgs, engine.ReadyEntity{ID: id})
					break
				}
			}
			return []engine.Message{engine.AskQuestion{
				Player: p.ID,
				Question: engine.Ask("Side-by-Side — choose:",
					engine.Choice{ID: "heal", Label: "Heal 1 damage from both characters", Kind: engine.ChoiceLabel}.
						Msgs(engine.HealEntity{Target: p.ID, N: 1}),
					engine.Choice{ID: "buff", Label: "+1 THW / +1 ATK for both this phase", Kind: engine.ChoiceLabel}.
						Msgs(engine.ApplyStatBonus{Target: p.ID, THW: 1, ATK: 1}),
				),
			}}
		},
	})

	// 45017 Suit Up: fetch an ally plus an attachable upgrade
	// (auto-picked).
	engine.RegisterBehavior("45017", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			got := 0
			for _, c := range append(engine.CardList{}, p.Deck...) {
				if c.Def().Type == "ally" {
					if _, ok := p.Deck.Remove(c.ID); ok {
						p.Hand = append(p.Hand, c)
						got++
					}
					break
				}
			}
			for _, c := range append(engine.CardList{}, p.Deck...) {
				if c.Def().Type == "upgrade" {
					if _, ok := p.Deck.Remove(c.ID); ok {
						p.Hand = append(p.Hand, c)
						got++
					}
					break
				}
			}
			if got > 0 {
				g.Logf("Suit Up pulls %d cards from the deck", got)
				return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
			}
			return nil
		},
	})

	// 45018 Lead from the Front: buff a player's team.
	engine.RegisterBehavior("45018", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, o := range g.Players {
				var msgs []engine.Message
				msgs = append(msgs, engine.ApplyStatBonus{Target: o.ID, THW: 1, ATK: 1})
				for _, id := range o.Allies {
					msgs = append(msgs, engine.AllyStatBonus{Ally: id, THW: 1, ATK: 1})
				}
				choices = append(choices, engine.Choice{
					ID: "p-" + o.ID.String(), Label: o.Name, Kind: engine.ChoiceLabel,
				}.Msgs(msgs...))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask("Lead from the Front — which player's team?", choices...),
			}}
		},
	})

	// 45019 The Power of Leadership / 45047 The Power of Aggression:
	// aspect doubling is data-driven via powerOfBonus.
	engine.RegisterBehavior("45019", &engine.Behavior{})
	engine.RegisterBehavior("45047", &engine.Behavior{})

	// 45020 Legion: after a basic power, mill 1 and apply its icon.
	engine.RegisterBehavior("45020", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			a := g.Allies[e.EID()]
			if a == nil {
				return nil
			}
			switch m := msg.(type) {
			case engine.AllyAttackWindow:
				if m.Ally != e.EID() {
					return nil
				}
			case engine.AllyThwartWindow:
				if m.Ally != e.EID() {
					return nil
				}
			default:
				return nil
			}
			p := g.Player(a.Owner)
			if p == nil || len(p.Deck) == 0 {
				return nil
			}
			c := p.Deck[0]
			msgs := []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: 1}}
			for _, r := range c.Def().Resources {
				switch r {
				case "energy":
					for _, id := range cardutil.SortedEnemyIDs(g) {
						msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 2, Source: p.ID})
						break
					}
				case "mental":
					if ids := g.Schemes(); len(ids) > 0 {
						msgs = append(msgs, engine.ThwartScheme{Scheme: ids[0], N: 2, Source: p.ID})
					}
				case "physical":
					msgs = append(msgs, engine.HealEntity{Target: a.ID, N: 2})
				}
			}
			g.Logf("Legion discards %s", c.Def().Name)
			return msgs
		},
	})

	// 45021 Marrow: X-FORCE/X-MEN requirement; 2 damage on entry.
	engine.RegisterBehavior("45021", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "X-Force") || g.EntityHasTrait(p.ID, "X-Men")
		},
		OnPlay: cardutil.ChooseEnemy("Marrow", func(g *engine.Game, e engine.Entity) (int, []engine.Message) {
			return 2, nil
		}),
	})

	// 45022-45024 Energy/Genius/Strength: deckbuilding limits only.
	for _, code := range []string{"45022", "45023", "45024"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	// 45041 Goldballs: mill up to 3 when attacking → +ATK each.
	engine.RegisterBehavior("45041", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyAttackWindow)
			if !ok || m.Ally != e.EID() {
				return nil
			}
			a := g.Allies[e.EID()]
			p := g.Player(a.Owner)
			if p == nil {
				return nil
			}
			n := 3
			if len(p.Deck) < n {
				n = len(p.Deck)
			}
			a.BonusATK += n
			g.Logf("Goldballs discards %d cards — +%d ATK", n, n)
			return []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: n}}
		},
	})

	// 45042 Tempus: cancel a villain scheme by discarding herself (window
	// in the engine's villain activation).
	engine.RegisterBehavior("45042", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "X-Men")
		},
	})

	// 45043 Blood Rage: draw after a killing basic attack.
	engine.RegisterBehavior("45043", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.BasicAttack)
			if !ok || e.EExhausted() {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if u == nil || u.Owner != m.Player {
				return nil
			}
			enemy := g.Entity(m.Target)
			var hp int
			switch t := enemy.(type) {
			case *engine.Villain:
				hp = t.HP()
			case *engine.Minion:
				hp = t.HP()
			default:
				return nil
			}
			if hp > m.N {
				return nil
			}
			g.Logf("Blood Rage — 1 damage for a card")
			return []engine.Message{
				engine.ExhaustEntity{ID: u.ID},
				engine.DamageEntity{Target: u.Owner, Damage: 1, Source: u.Owner},
				engine.DrawCards{Player: u.Owner, N: 1},
			}
		},
	})

	// 45044 Test the Defense: counters per ATTACK event; 5 pays out.
	engine.RegisterBehavior("45044", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.EventPlayed)
			if !ok || !m.Card.Def().HasTrait("Attack") {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if u == nil || u.Owner != m.Player {
				return nil
			}
			u.Counters++
			g.Logf("Test the Defense gains a test counter (%d/5)", u.Counters)
			if u.Counters >= 5 && len(g.Enemies()) > 0 {
				u.Counters = 0
				return []engine.Message{
					engine.DiscardControlled{Player: u.Owner, ID: u.ID},
					engine.DamageEntity{Target: g.Enemies()[0], Damage: 5, Source: u.Owner},
				}
			}
			return nil
		},
	})

	// 45045 Full-Body Charge: 8 damage (overkill rider approximated).
	engine.RegisterBehavior("45045", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy("Full-Body Charge", func(g *engine.Game, e engine.Entity) (int, []engine.Message) {
			return 8, nil
		}),
	})

	// 45046 Clobber: 3 damage; returns to hand when it's the round's first
	// card (window in the engine's play handler).
	engine.RegisterBehavior("45046", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy("Clobber", func(g *engine.Game, e engine.Entity) (int, []engine.Message) {
			return 3, nil
		}),
	})

	// 45048 Triage: heal 2 from an X-Men character.
	engine.RegisterBehavior("45048", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			if g.EntityHasTrait(p.ID, "X-Men") {
				choices = append(choices, engine.Choice{
					ID: "hero", Label: p.HeroDef().Name, Kind: engine.ChoiceTarget,
				}.Msgs(engine.HealEntity{Target: p.ID, N: 2}))
			}
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a == nil || !a.EDef().HasTrait("X-Men") {
					continue
				}
				choices = append(choices, engine.Choice{
					ID: "ally-" + id.String(), Label: a.EDef().Name, Kind: engine.ChoiceTarget,
				}.Msgs(engine.HealEntity{Target: id, N: 2}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask("Triage — heal 2 damage from:", choices...),
			}}
		},
	})

	// 45049 Stepford Cuckoos: cancel a revealed treachery for a psi
	// counter (window in the engine's treachery window).
	engine.RegisterBehavior("45049", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "X-Men")
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			g.Supports[e.EID()].Counters = 3
			return nil
		},
	})

	// 45050 Bloodgem: wild resource (the 2-damage cost is approximated
	// away — ResourceAbility cannot self-damage the owner).
	engine.RegisterBehavior("45050", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "Mystic")
		},
		Resource: &engine.ResourceAbility{Icon: "wild"},
	})

	// 45051 Basic Spell: three-way toolkit.
	engine.RegisterBehavior("45051", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "Mystic")
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			choices := []engine.Choice{
				engine.Choice{ID: "heal", Label: "Heal 3 damage from an identity", Kind: engine.ChoiceLabel}.
					Msgs(engine.HealEntity{Target: p.ID, N: 3}),
			}
			if len(g.Enemies()) > 0 {
				var dmg []engine.Choice
				for _, id := range cardutil.SortedEnemyIDs(g) {
					enemy := g.Entity(id)
					dmg = append(dmg, engine.Choice{
						Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget,
						SourceID: id, CardCode: enemy.ECode(),
					}.Msgs(engine.DamageEntity{Target: id, Damage: 3, Source: p.ID}))
				}
				choices = append(choices, engine.Choice{ID: "dmg", Label: "Deal 3 damage to an enemy", Kind: engine.ChoiceLabel}.
					Msgs(engine.AskQuestion{Player: p.ID, Question: engine.Ask("Basic Spell — damage:", dmg...)}))
			}
			if len(g.Schemes()) > 0 {
				var thw []engine.Choice
				for _, id := range g.Schemes() {
					s := g.Entity(id)
					thw = append(thw, engine.Choice{
						Label: s.EDef().Name, Kind: engine.ChoiceTarget,
						SourceID: id, CardCode: s.ECode(),
					}.Msgs(engine.ThwartScheme{Scheme: id, N: 3, Source: p.ID}))
				}
				choices = append(choices, engine.Choice{ID: "thw", Label: "Remove 3 threat from a scheme", Kind: engine.ChoiceLabel}.
					Msgs(engine.AskQuestion{Player: p.ID, Question: engine.Ask("Basic Spell — thwart:", thw...)}))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask("Basic Spell — choose:", choices...),
			}}
		},
	})

	// 45052 Spiritual Meditation: draw 2, then discard 1.
	engine.RegisterBehavior("45052", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "Mystic")
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var discard []engine.Choice
			for i, c := range p.Hand {
				card := c
				discard = append(discard, engine.Choice{
					ID: fmt.Sprintf("d-%d", i), Label: card.Def().Name, Kind: engine.ChoiceCard, CardCode: card.Code,
				}.Msgs(engine.DiscardCards{Player: p.ID, Cards: engine.CardList{card}}))
			}
			return []engine.Message{
				engine.DrawCards{Player: p.ID, N: 2},
				engine.AskQuestion{Player: p.ID, Question: engine.Ask("Spiritual Meditation — discard:", discard...)},
			}
		},
	})
}

// applyTeamTraining grants +1 max HP to the owner's allies (once each —
// allies track via PermTHW-style marker? approximated by MaxHP bump).
func applyTeamTraining(g *engine.Game, e engine.Entity) {
	s := g.Supports[e.EID()]
	if s == nil {
		return
	}
	p := g.Player(s.Owner)
	if p == nil {
		return
	}
	for _, id := range p.Allies {
		if a := g.Allies[id]; a != nil {
			a.MaxHP++
		}
	}
}
