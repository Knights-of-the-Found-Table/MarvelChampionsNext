// mg_mastermold.go implements the Master Mold scenario set
// (32109–32120).
package mutantgenesis

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerMasterMold()
}

// masterMold returns the Master Mold villain.
func masterMold(g *engine.Game) *engine.Villain {
	for _, v := range g.Villains {
		if v != nil && engine.BaseCodeOf(v.Code) == "32109" {
			return v
		}
	}
	return nil
}

// spawnSentinelFromDeck mills until a Sentinel minion shows up and spawns
// it engaged with pid.
func spawnSentinelFromDeck(g *engine.Game, pid engine.PlayerID, log string) []engine.Message {
	for guards := 0; guards < 40; guards++ {
		if len(g.EncounterDeck) == 0 {
			if len(g.EncounterDiscard) == 0 {
				return nil
			}
			g.EncounterDeck = g.EncounterDiscard
			g.EncounterDiscard = nil
			for i := len(g.EncounterDeck) - 1; i > 0; i-- {
				j := g.Random(i + 1)
				g.EncounterDeck[i], g.EncounterDeck[j] = g.EncounterDeck[j], g.EncounterDeck[i]
			}
		}
		top := g.EncounterDeck[0]
		g.EncounterDeck = g.EncounterDeck[1:]
		def := top.Def()
		if def.Type == "minion" && def.HasTrait("sentinel") {
			hp := 1
			if def.HP != nil {
				hp = *def.HP
			}
			atk, sch := 0, 0
			if def.Attack != nil {
				atk = *def.Attack
			}
			if def.Scheme != nil {
				sch = *def.Scheme
			}
			mn := &engine.Minion{
				ID: g.NextEntityID(engine.KindMinion), Code: def.Code,
				MaxHP: hp, AttackVal: atk, SchemeVal: sch, EngagedWith: pid,
				Tough: def.HasKeyword("Toughness"), Guard: def.HasKeyword("Guard"),
			}
			g.Minions[mn.ID] = mn
			g.Logf("%s: %s engages %s", log, def.Name, g.Player(pid).Name)
			return []engine.Message{engine.MinionEntersPlay{MinionID: mn.ID, Player: pid}}
		}
		g.EncounterDiscard = append(g.EncounterDiscard, top)
	}
	return nil
}

func registerMasterMold() {
	// 32109–32111 Master Mold stages: when he schemes, mill to a Sentinel
	// minion and put it into play engaged (no boost card).
	for _, code := range []string{"32109", "32110", "32111"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				m, ok := msg.(engine.ApplyVillainScheme)
				if !ok || m.VillainID != e.EID() {
					return nil
				}
				return spawnSentinelFromDeck(g, m.Player, "Master Mold's factory")
			},
		})
	}

	// 32112 The Sentinel Factory: When Revealed — each player gets a
	// Sentinel from the deck (the per-minion guard aura is enforced in
	// guardBlocksVillain).
	engine.RegisterBehavior("32112", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, spawnSentinelFromDeck(g, p.ID, "The Sentinel Factory")...)
			}
			return msgs
		},
	})

	// 32113 Master Mold's Agenda: completing it loses (default behavior).
	engine.RegisterBehavior("32113", &engine.Behavior{})

	// 32114 Sentinel Mark VIII: on engage, attach the topmost Sentinel
	// attachment from the discard pile.
	engine.RegisterBehavior("32114", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionEntersPlay)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			for i, c := range g.EncounterDiscard {
				if c.Def().HasTrait("sentinel") && c.Def().Type == "attachment" {
					g.EncounterDiscard = append(g.EncounterDiscard[:i], g.EncounterDiscard[i+1:]...)
					return []engine.Message{engine.RevealEncounterCard{Player: m.Player, Card: c}}
				}
			}
			return nil
		},
	})

	// 32115 Unit Upgrade: attach to a Sentinel minion (+2 HP, retaliate
	// cosmetic).
	engine.RegisterBehavior("32115", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn == nil || !mn.EDef().HasTrait("sentinel") {
					continue
				}
				t.Target = mn.ID
				mn.Attachments = append(mn.Attachments, t.ID)
				mn.MaxHP += 2
				return nil
			}
			g.Delete(t.ID)
			return []engine.Message{engine.RevealNextEncounter{Player: cardutil.FirstPlayerID(g)}}
		},
	})

	// 32116 Stun Beam: attach to a Sentinel minion; after it attacks and
	// damages a character, stun that character (approximation: the
	// defender is stunned on defense).
	engine.RegisterBehavior("32116", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn == nil || !mn.EDef().HasTrait("sentinel") {
					continue
				}
				attached := false
				for _, aid := range mn.Attachments {
					if a := g.Attachments[aid]; a != nil && a.Code == "32116" {
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
			m, ok := msg.(engine.AskAttack)
			t := g.Attachments[e.EID()]
			if !ok || t == nil || m.Enemy != t.Target {
				return nil
			}
			return []engine.Message{engine.StunEntity{Target: m.Player}}
		},
	})

	// 32117 Master Mold's Children: each engaged minion activates against
	// you; with none, Master Mold does.
	engine.RegisterBehavior("32117", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var msgs []engine.Message
			for _, id := range cardutil.SortedIDs(g.Minions) {
				if mn := g.Minions[id]; mn != nil && mn.EngagedWith == p.ID {
					msgs = append(msgs, engine.MinionActivates{MinionID: id, Player: p.ID})
				}
			}
			if len(msgs) == 0 {
				if mm := masterMold(g); mm != nil {
					if p.IsHero() {
						msgs = append(msgs, engine.AskAttack{Enemy: mm.ID, Player: p.ID, Trigger: engine.TriggerVillainAttacksYou})
					} else if g.MainScheme != nil {
						msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: mm.SchemeVal, Source: mm.ID})
					}
				}
			}
			return msgs
		},
	})

	// 32118 Shields Up: tough for each engaged Sentinel; boost — villain
	// tough.
	engine.RegisterBehavior("32118", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var msgs []engine.Message
			for _, id := range cardutil.SortedIDs(g.Minions) {
				if mn := g.Minions[id]; mn != nil && mn.EngagedWith == p.ID && mn.EDef().HasTrait("sentinel") {
					msgs = append(msgs, engine.ToughEntity{Target: id})
				}
			}
			if len(msgs) == 0 {
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			return msgs
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if v := masterMold(g); v != nil {
				return []engine.Message{engine.ToughEntity{Target: v.ID}}
			}
			return nil
		},
	})

	// 32119 Intruder Alert!: When Defeated — the defeater gets a Sentinel.
	engine.RegisterBehavior("32119", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			return spawnSentinelFromDeck(g, cardutil.FirstPlayerID(g), "Intruder Alert!")
		},
	})

	// 32120 Insert Virus Program: Hinder 2/hero; When Defeated — 2 damage
	// to each Sentinel enemy.
	engine.RegisterBehavior("32120", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			var msgs []engine.Message
			if v := masterMold(g); v != nil && v.EDef().HasTrait("sentinel") {
				msgs = append(msgs, engine.DamageEntity{Target: v.ID, Damage: 2, Source: e.EID()})
			}
			for _, id := range cardutil.SortedIDs(g.Minions) {
				if mn := g.Minions[id]; mn != nil && mn.EDef().HasTrait("sentinel") {
					msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 2, Source: e.EID()})
				}
			}
			return msgs
		},
	})
}
