// complete.go implements the remaining Gambit pack cards (37011–37035):
// the shared Tactic suite, the Assassins Guild nemesis set and the Exodus
// modular set.
package gambit

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() {
	registerRemainingGambit()
}

func registerRemainingGambit() {
	// 37011 Bishop: charges per attack received; attacks with +2 per
	// charge (max +6).
	engine.RegisterBehavior("37011", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			a := g.Allies[e.EID()]
			if a == nil {
				return nil
			}
			switch m := msg.(type) {
			case engine.WindowAfterEnemyAttacked:
				if m.Player == a.Owner {
					a.Counters++
					g.TLogf("c.bishopAbsorbsAnEnergyCharge", a.Counters)
				}
			case engine.AllyAttackWindow:
				if m.Ally != e.EID() || a.Counters <= 0 {
					return nil
				}
				bonus := 2 * a.Counters
				if bonus > 6 {
					bonus = 6
				}
				a.Counters = 0
				g.TLogf("c.bishopUnleashesAtk", bonus)
				return []engine.Message{engine.DamageEntity{Target: m.Target, Damage: bonus, Source: a.Owner}}
			}
			return nil
		},
	})

	// 37012 Dazzler: on enter — confuse an enemy.
	engine.RegisterBehavior("37012", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			var choices []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				en := g.Entity(id)
				if en != nil {
					choices = append(choices, engine.Choice{
						Label: engine.Tf("c.confuse2", cardutil.EnemyLabel(en)), Kind: engine.ChoiceTarget, SourceID: id,
					}.Msgs(engine.ConfuseEntity{Target: id}))
				}
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.dazzlerConfuseAnEnemy"), choices...),
			}}
		},
	})

	// 37013 Operative Skill: 3 counters; your basic thwarts remove +1.
	engine.RegisterBehavior("37013", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 3}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.BasicThwart)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil || u.Exhausted || u.Counters <= 0 || m.Player != u.Owner {
				return nil
			}
			u.Counters--
			return []engine.Message{
				engine.ExhaustEntity{ID: u.ID},
				engine.ThwartScheme{Scheme: m.Target, N: 1, Source: u.Owner},
			}
		},
	})

	// 37014 Stealth Strike: 4 damage; if it defeats, remove 2 threat.
	engine.RegisterBehavior("37014", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				en := g.Entity(id)
				if en == nil {
					continue
				}
				msgs := []engine.Message{engine.DamageEntity{Target: id, Damage: 4, Source: p.ID}}
				hp := 99
				switch t := en.(type) {
				case *engine.Minion:
					hp = t.HP()
				case *engine.Villain:
					hp = t.HP()
				}
				if hp <= 4 && g.MainScheme != nil {
					msgs = append(msgs, engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 2, Source: p.ID})
				}
				choices = append(choices, engine.Choice{
					Label: cardutil.EnemyLabel(en), Kind: engine.ChoiceTarget, SourceID: id,
				}.Msgs(msgs...))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.stealthStrike"), choices...),
			}}
		},
	})

	// 37015 Breaking and Entering: remove 3 threat (SPY/THIEF only).
	engine.RegisterBehavior("37015", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "spy") || g.EntityHasTrait(p.ID, "thief")
		},
		OnPlay: cardutil.ChooseScheme(engine.Tf("c.chooseAScheme", "Breaking and Entering"), func(g *engine.Game, e engine.Entity) int { return 3 }),
	})

	// 37016 Passion for Justice: rider in handlePlayCard.
	engine.RegisterBehavior("37016", &engine.Behavior{})

	// 37017 Professor X: reprint of the Mutant Genesis implementation
	// pattern (choice + end-of-round discard).
	engine.RegisterBehavior("37017", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, id := range cardutil.SortedIDs(g.Villains) {
				if v := g.Villains[id]; v != nil {
					choices = append(choices, engine.Choice{
						Label: engine.Tf("c.confuseName", v), Kind: engine.ChoiceTarget, SourceID: id,
					}.Msgs(engine.ConfuseEntity{Target: id}))
					break
				}
			}
			for _, id := range cardutil.SortedIDs(g.Minions) {
				if mn := g.Minions[id]; mn != nil {
					choices = append(choices, engine.Choice{
						Label: engine.Tf("c.stun2", cardutil.EnemyLabel(mn)), Kind: engine.ChoiceTarget, SourceID: id,
					}.Msgs(engine.StunEntity{Target: id}))
					break
				}
			}
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && a.EDef().HasTrait("x-men") {
					choices = append(choices, engine.Choice{
						Label: engine.Tf("c.readyName", a), Kind: engine.ChoiceTarget, SourceID: id,
					}.Msgs(engine.ReadyEntity{ID: id}))
					break
				}
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.professorXChooseOne"), choices...),
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.EndRound); !ok {
				return nil
			}
			if a := g.Allies[e.EID()]; a != nil {
				g.Delete(a.ID)
				if p := g.Player(a.Owner); p != nil {
					p.Discard = append(p.Discard, engine.Card{ID: g.NextCardID(), Code: a.Code, Owner: p.ID})
				}
				g.TLogf("c.professorXLeavesPlayAtTheEndOfTheRound")
			}
			return nil
		},
	})

	// 37018 X-Mansion: reprint.
	engine.RegisterBehavior("37018", engine.LookupBehavior("36025"))

	// 37019 Beauty and the Thief: 4 damage and 4 threat removed.
	beautyAndThief := &engine.Behavior{
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
				Question: engine.Ask(engine.Tf("c.beautyAndTheThiefDeal4Damage"), choices...),
			})
		},
	}
	engine.RegisterBehavior("37019", beautyAndThief)

	// 37020 Hit and Run: 2 damage and 2 threat removed.
	engine.RegisterBehavior("37020", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			msgs := []engine.Message{}
			if g.MainScheme != nil {
				msgs = append(msgs, engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 2, Source: p.ID})
			}
			choices := cardutil.EnemyChoices(g, 2, p.ID, func(target engine.EntityID) []engine.Message {
				return []engine.Message{engine.DamageEntity{Target: target, Damage: 2, Source: p.ID}}
			})
			if len(choices) == 0 {
				return msgs
			}
			return append(msgs, engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.hitAndRunDeal2Damage"), choices...),
			})
		},
	})

	// 37021 Mutant Education: shuffle up to 2 signature cards from
	// discard into your deck (+1 draw with X-Mansion).
	engine.RegisterBehavior("37021", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "mutant")
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			set := p.AlterEgoDef().CardSet
			shuffled := 0
			for i := 0; i < len(p.Discard) && shuffled < 2; {
				c := p.Discard[i]
				if c.Def().CardSet == set {
					p.Discard = append(p.Discard[:i], p.Discard[i+1:]...)
					p.Deck = append(p.Deck, c)
					shuffled++
					continue
				}
				i++
			}
			for _, id := range p.Supports {
				if s := g.Supports[id]; s != nil && s.EDef().EName == "X-Mansion" {
					return []engine.Message{engine.DrawCards{Player: p.ID, N: 1}}
				}
			}
			return nil
		},
	})

	// 37022–37024 basic resources.
	for _, code := range []string{"37022", "37023", "37024"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	// 37025 Guild Business: alter-ego exhaust removes it.
	engine.RegisterBehavior("37025", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			choices := []engine.Choice{engine.Choice{
				ID: "keep", Label: engine.Tf("c.keepGuildBusinessInPlay"), Kind: engine.ChoiceLabel,
			}.Msgs(engine.ObligationResolve{Player: p.ID, Card: card})}
			if !p.IsHero() && !p.Exhausted {
				choices = append(choices, engine.Choice{
					ID: "exhaust", Label: engine.Tf("c.exhaustRemyLebeauSpendEnergyRemoveFromTheGame"), Kind: engine.ChoiceLabel,
				}.Msgs(engine.ExhaustEntity{ID: p.ID}, engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.guildBusinessChoose"), choices...),
			}}
		},
	})

	// 37026 Belladonna: after she defeats a character, 2 threat on the
	// main scheme (approximation: any ally defeat triggers it).
	engine.RegisterBehavior("37026", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.AllyDefeated); !ok || g.MainScheme == nil {
				return nil
			}
			if mn := g.Minions[e.EID()]; mn != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: mn.ID}}
			}
			return nil
		},
	})

	// 37027 The Assassins Guild: after an ASSASSIN defeats a character, 2
	// threat here.
	engine.RegisterBehavior("37027", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.AllyDefeated); !ok {
				return nil
			}
			assassin := false
			for _, mn := range g.Minions {
				if mn != nil && mn.EDef().HasTrait("assassin") {
					assassin = true
				}
			}
			if assassin {
				return []engine.Message{engine.SchemeThreat{Scheme: e.EID(), N: 2, Source: e.EID()}}
			}
			return nil
		},
	})

	// 37028 Guild Assassin: after defeating a character, 1 threat on the
	// main scheme.
	engine.RegisterBehavior("37028", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.AllyDefeated); !ok || g.MainScheme == nil {
				return nil
			}
			if mn := g.Minions[e.EID()]; mn != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: mn.ID}}
			}
			return nil
		},
	})

	// 37029 Assassination Attempt: each ASSASSIN attacks you; none →
	// reveal one.
	engine.RegisterBehavior("37029", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var msgs []engine.Message
			for _, id := range cardutil.SortedIDs(g.Minions) {
				if mn := g.Minions[id]; mn != nil && mn.EDef().HasTrait("assassin") {
					msgs = append(msgs, engine.AskAttack{Enemy: id, Player: p.ID})
				}
			}
			if len(msgs) > 0 {
				return msgs
			}
			for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
				for _, c := range *zone {
					if c.Def().Type == "minion" && c.Def().HasTrait("assassin") {
						zone.Remove(c.ID)
						return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
					}
				}
			}
			return nil
		},
	})

	// 37030 War Room: after a minion is defeated (approximation of the
	// ally-attribution), exhaust → remove 1 threat.
	engine.RegisterBehavior("37030", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.MinionDefeated); !ok {
				return nil
			}
			s := g.Supports[e.EID()]
			if s == nil || s.Exhausted || g.MainScheme == nil {
				return nil
			}
			return []engine.Message{
				engine.ExhaustEntity{ID: s.ID},
				engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 1, Source: s.Owner},
			}
		},
	})

	// 37031 X-Men Instruction: shuffle up to 2 X-Men allies back.
	engine.RegisterBehavior("37031", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "mutant")
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			shuffled := 0
			for i := 0; i < len(p.Discard) && shuffled < 2; {
				c := p.Discard[i]
				if c.Def().Type == "ally" && c.Def().HasTrait("x-men") {
					p.Discard = append(p.Discard[:i], p.Discard[i+1:]...)
					p.Deck = append(p.Deck, c)
					shuffled++
					continue
				}
				i++
			}
			for _, id := range p.Supports {
				if s := g.Supports[id]; s != nil && s.EDef().EName == "X-Mansion" {
					return []engine.Message{engine.DrawCards{Player: p.ID, N: 1}}
				}
			}
			return nil
		},
	})

	// 37032 Exodus: reveal — search Psionic Shield.
	engine.RegisterBehavior("37032", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionEntersPlay)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
				for _, c := range *zone {
					if c.Code == "37034" {
						zone.Remove(c.ID)
						return []engine.Message{engine.RevealEncounterCard{Player: m.Player, Card: c}}
					}
				}
			}
			return nil
		},
	})

	// 37033 Herald of Avalon: When Defeated — reveal Exodus.
	engine.RegisterBehavior("37033", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			for _, mn := range g.Minions {
				if mn != nil && mn.Code == "37032" {
					return nil
				}
			}
			for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
				for _, c := range *zone {
					if c.Code == "37032" {
						zone.Remove(c.ID)
						return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
					}
				}
			}
			return nil
		},
	})

	// 37034 Psionic Shield: attach to a minion; a would-be defeat heals
	// it fully instead, then this discards.
	engine.RegisterBehavior("37034", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn == nil {
					continue
				}
				attached := false
				for _, a := range g.Attachments {
					if a != nil && a.Code == "37034" && a.Target == mn.ID {
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
	})

	// 37035 Acolyte Frenzy: engaged Acolytes activate; none → surge.
	engine.RegisterBehavior("37035", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var msgs []engine.Message
			for _, id := range cardutil.SortedIDs(g.Minions) {
				if mn := g.Minions[id]; mn != nil && mn.EngagedWith == p.ID && mn.EDef().HasTrait("acolyte") {
					msgs = append(msgs, engine.MinionActivates{MinionID: id, Player: p.ID})
				}
			}
			if len(msgs) == 0 {
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			return msgs
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			pid := cardutil.FirstPlayerID(g)
			return []engine.Message{engine.StunEntity{Target: pid}, engine.ConfuseEntity{Target: pid}}
		},
	})
}
