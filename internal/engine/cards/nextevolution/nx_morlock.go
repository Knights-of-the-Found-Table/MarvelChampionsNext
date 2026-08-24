package nextevolution

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// tagTeamMill discards the top 7 encounter cards and summons the topmost
// Marauder minion from the discard pile (Tag Team).
func tagTeamMill(g *engine.Game, p *engine.Player) []engine.Message {
	for i := 0; i < 7; i++ {
		c, ok := g.DrawEncounter()
		if !ok {
			break
		}
		g.EncounterDiscard = append(g.EncounterDiscard, c)
	}
	for i := len(g.EncounterDiscard) - 1; i >= 0; i-- {
		c := g.EncounterDiscard[i]
		if c.Def().Type == "minion" && c.Def().HasTrait("Marauder") {
			g.EncounterDiscard = append(g.EncounterDiscard[:i:i], g.EncounterDiscard[i+1:]...)
			g.Logf("Tag Team summons %s", c.Def().Name)
			return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
		}
	}
	return nil
}

// registerMarauders installs the seven Marauders (villain and minion
// printings share base codes) and the Morlock Siege board.
func registerMarauders() {
	// Arclight 40070/40094.
	engine.RegisterBehavior("40070", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AskAttack)
			if !ok {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil {
				return nil
			}
			return marauderChoice(g, e, m, "Confuse a character you control",
				[]engine.Message{engine.ConfuseEntity{Target: p.ID}}, 2)
		},
	})

	// Blockbuster 40071/40095.
	engine.RegisterBehavior("40071", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AskAttack)
			if !ok {
				return nil
			}
			return marauderChoice(g, e, m, "Give Blockbuster a tough status card",
				[]engine.Message{engine.ToughEntity{Target: e.EID()}}, 2)
		},
	})

	// Chimera 40072/40096.
	engine.RegisterBehavior("40072", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AskAttack)
			if !ok {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil {
				return nil
			}
			for _, c := range p.Hand {
				for _, r := range c.Def().Resources {
					if r == "mental" || r == "wild" {
						return marauderChoice(g, e, m,
							fmt.Sprintf("Spend %s (a mental resource)", c.Def().Name),
							[]engine.Message{engine.DiscardCards{Player: p.ID, Cards: engine.CardList{c}}}, 2)
					}
				}
			}
			// Nothing to spend: forced into the +2 ATK.
			return []engine.Message{engine.BoostActivation{Enemy: e.EID(), N: 2}}
		},
	})

	// Greycrow 40073/40097.
	engine.RegisterBehavior("40073", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AskAttack)
			if !ok {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil {
				return nil
			}
			// X = printed cost of the highest-cost card in play under the
			// player's control (hand approximation: highest-cost hand card).
			best := 0
			var bestCard engine.Card
			for _, c := range p.Hand {
				if n := cardutil.Cost(c.Def()); n > best {
					best, bestCard = n, c
				}
			}
			var penalty []engine.Message
			label := "Discard your highest-cost card"
			if best > 0 {
				penalty = []engine.Message{engine.DiscardCards{Player: p.ID, Cards: engine.CardList{bestCard}}}
				label = fmt.Sprintf("Discard %s (highest cost)", bestCard.Def().Name)
			}
			return marauderChoice(g, e, m, label, penalty, best)
		},
	})

	// Harpoon 40074/40098.
	engine.RegisterBehavior("40074", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AskAttack)
			if !ok {
				return nil
			}
			return marauderChoice(g, e, m, "Take 2 indirect damage",
				[]engine.Message{engine.IndirectDamage{Player: m.Player, N: 2}}, 2)
		},
	})

	// Riptide 40075/40099.
	engine.RegisterBehavior("40075", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AskAttack)
			if !ok {
				return nil
			}
			var penalty []engine.Message
			if g.MainScheme != nil {
				penalty = append(penalty, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: e.EID()})
			}
			for _, id := range cardutil.SortedIDs(g.SideSchemes) {
				penalty = append(penalty, engine.ApplySchemeThreat{Scheme: id, N: 1, Source: e.EID()})
			}
			return marauderChoice(g, e, m, "2 threat on the main scheme + 1 on each side scheme",
				penalty, 2)
		},
	})

	// Vertigo 40076/40100.
	engine.RegisterBehavior("40076", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AskAttack)
			if !ok {
				return nil
			}
			return marauderChoice(g, e, m, "Stun a character you control",
				[]engine.Message{engine.StunEntity{Target: m.Player}}, 2)
		},
	})

	// 40077 Knock, Knock: knock counters after villain phase step one.
	engine.RegisterBehavior("40077", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.BeginPhase); !ok || g.MainScheme == nil {
				return nil
			}
			if g.MainScheme.EID() != e.EID() || engine.BaseCodeOf(g.MainScheme.Code) != "40077" {
				return nil
			}
			g.MainScheme.Counters++
			g.Logf("Knock, Knock gains a knock counter (%d)", g.MainScheme.Counters)
			if g.MainScheme.Counters >= 3 && g.MainScheme.Stage == 1 {
				g.Logf("The Marauders break down the door!")
				return []engine.Message{engine.ReplaceMainScheme{Scheme: g.MainScheme.ID}}
			}
			return nil
		},
	})

	// 40078 Mutant Massacre: the stage flip's reveal effects run via the
	// scenario's OnMainSchemeMaxed hook (knock-counter path) — the Morlock
	// setup lives there; the lose conditions are checked below.
	engine.RegisterBehavior("40078", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			// Lose when no Morlock allies remain in play at any phase
			// change while this stage is up.
			if _, ok := msg.(engine.BeginPhase); !ok || g.MainScheme == nil {
				return nil
			}
			if engine.BaseCodeOf(g.MainScheme.Code) != "40078" {
				return nil
			}
			for _, p := range g.Players {
				for _, id := range p.Allies {
					if a := g.Allies[id]; a != nil && a.EDef().HasTrait("Morlock") {
						return nil
					}
				}
			}
			return []engine.Message{engine.GameOver{Won: false, Reason: "No Morlock allies remain (Mutant Massacre)"}}
		},
	})

	// 40079 Morlock ally: the attack redirect lives in the engine's
	// handleAskAttack; card abilities cannot remove them from play.
	engine.RegisterBehavior("40079", &engine.Behavior{})

	// 40080 Hide!: tough a Morlock; boost adds a villain boost card.
	engine.RegisterBehavior("40080", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			for _, o := range g.Players {
				for _, id := range o.Allies {
					if a := g.Allies[id]; a != nil && a.EDef().HasTrait("Morlock") {
						return []engine.Message{engine.ToughEntity{Target: id}}
					}
				}
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			for _, o := range g.Players {
				for _, id := range o.Allies {
					if a := g.Allies[id]; a != nil && a.EDef().HasTrait("Morlock") {
						return []engine.Message{engine.ToughEntity{Target: id}, engine.DealBoost{Enemy: boostEnemy(g)}}
					}
				}
			}
			return nil
		},
	})

	// 40081a Routed: tucks defeated villains; the villain-deck flow lives
	// in the scenario hooks.
	engine.RegisterBehavior("40081", &engine.Behavior{})

	// 40082 Bolstered by Wrath: villain +X ATK (X = villains under Routed).
	// Implemented as an engine check in attackValue; the hero discard
	// action is not modeled.
	engine.RegisterBehavior("40082", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			t.Target = strongestVillain(g)
			return nil
		},
	})

	// 40083 Pushed to the Limit: steady/stalwart gains are not modeled.
	engine.RegisterBehavior("40083", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			t.Target = strongestVillain(g)
			return nil
		},
	})

	// 40084-40087 side schemes: extra threat per villain under Routed.
	for _, code := range []string{"40084", "40085", "40086", "40087"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
				s := g.SideSchemes[e.EID()]
				if s == nil {
					return nil
				}
				n := villainsUnderRouted(g) * len(g.Players)
				if n > 0 {
					s.Threat += n
					g.Logf("%s gains %d extra threat (villains under Routed)", s.EDef().Name, n)
				}
				return nil
			},
		})
	}

	// 40088 Back in Action: villain tough + threat per routed villain.
	engine.RegisterBehavior("40088", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var msgs []engine.Message
			if id := strongestVillain(g); id != "" {
				msgs = append(msgs, engine.ToughEntity{Target: id})
				if g.MainScheme != nil {
					msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: villainsUnderRouted(g), Source: t.ID})
				}
			}
			return msgs
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if p := g.Player(cardutil.FirstPlayerID(g)); p != nil {
				for _, id := range p.Allies {
					if a := g.Allies[id]; a != nil && a.EDef().HasTrait("Morlock") {
						return []engine.Message{engine.DamageEntity{Target: id, Damage: 1, Source: engine.EntityID("")}}
					}
				}
			}
			return nil
		},
	})

	// 40089 Seek the Weak: alter-ego surge / hero attacked.
	engine.RegisterBehavior("40089", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if p.IsHero() {
				if id := strongestVillain(g); id != "" {
					return []engine.Message{engine.AskAttack{Enemy: id, Player: p.ID}}
				}
				return nil
			}
			if c, ok := g.DrawEncounter(); ok {
				return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			// Overkill rider not modeled.
			return nil
		},
	})

	// 40090 Heavy Armament: attach to the highest-ATK enemy; retaliate 2
	// lives in the engine's retaliateOf.
	engine.RegisterBehavior("40090", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			t.Target = highestATKEnemy(g)
			return nil
		},
	})

	// 40091 Titanium Exoskeleton: attach to fewest-HP enemy; damage cap
	// lives in the engine's damage().
	engine.RegisterBehavior("40091", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			t.Target = fewestHPEnemy(g)
			return nil
		},
	})

	// 40092 Inhibitor Collar: text blanking not modeled.
	engine.RegisterBehavior("40092", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if p := g.Player(cardutil.FirstPlayerID(g)); p != nil {
				t.Target = p.ID
			}
			return nil
		},
	})

	// 40093 The Senator's Support: Hinder parsed from text; defeat mills
	// for an attachment.
	engine.RegisterBehavior("40093", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			for i := 0; i < 30; i++ {
				c, ok := g.DrawEncounter()
				if !ok {
					return nil
				}
				if c.Def().Type == "attachment" {
					g.Logf("The Senator's Support reveals %s", c.Def().Name)
					return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			return nil
		},
	})

	// 40101 Mutant Slayers: extra threat per MUTANT-ish character.
	engine.RegisterBehavior("40101", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			s := g.SideSchemes[e.EID()]
			if s == nil {
				return nil
			}
			n := 0
			for _, p := range g.Players {
				for _, id := range p.Allies {
					a := g.Allies[id]
					if a == nil {
						continue
					}
					if a.EDef().HasTrait("Mutant") || a.EDef().HasTrait("X-Factor") ||
						a.EDef().HasTrait("X-Force") || a.EDef().HasTrait("X-Men") {
						n++
					}
				}
			}
			if n > 0 {
				s.Threat += n
				g.Logf("%s gains %d extra threat", s.EDef().Name, n)
			}
			return nil
		},
	})

	// 40102 Bound by Business: mill for a fresh Marauder minion.
	engine.RegisterBehavior("40102", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var discarded engine.CardList
			for i, c := range g.EncounterDeck {
				discarded = append(discarded, c)
				if c.Def().Type == "minion" && c.Def().HasTrait("Marauder") && !titleInPlay(g, c.Def().Name) {
					g.EncounterDeck = g.EncounterDeck[i+1:]
					g.EncounterDiscard = append(g.EncounterDiscard, discarded[:i]...)
					g.Logf("Bound by Business finds %s", c.Def().Name)
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
			}
			// Deck exhausted of candidates: discard everything looked at.
			g.EncounterDeck = nil
			g.EncounterDiscard = append(g.EncounterDiscard, discarded...)
			return nil
		},
	})
}

func registerOnTheRun() {
	// The mutant_slayers minion printings share the Marauder villains'
	// attack choices; alias them to the villain registrations.
	for pair, villain := range map[string]string{
		"40094": "40070", "40095": "40071", "40096": "40072", "40097": "40073",
		"40098": "40074", "40099": "40075", "40100": "40076",
	} {
		code, src := pair, villain
		engine.RegisterBehavior(code, engine.LookupBehavior(src))
	}

	// 40103 Gotta Get Away: each player fetches a Marauder minion on
	// reveal; completed = loss (the default).
	engine.RegisterBehavior("40103", &engine.Behavior{})

	// 40104 Escaping with Hope: villain defeat wins the scenario; the
	// scenario hooks handle it.
	engine.RegisterBehavior("40104", &engine.Behavior{})

	// 40105a Hope's Captor: attack-to-scheme redirect + defeat reset live
	// in the shared Marauder villain hooks (see onTheRunVillainGuards).
	engine.RegisterBehavior("40105", &engine.Behavior{})

	// 40106 Hidden in the Clutter: damage banks on the attachment (engine
	// damage() handles the banking).
	engine.RegisterBehavior("40106", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			t.Target = fewestHPEnemy(g)
			return nil
		},
	})

	// 40107 Favored Weapon: overkill/piercing/ranged riders not modeled.
	engine.RegisterBehavior("40107", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, code := range []string{"40073", "40074"} {
				for _, id := range cardutil.SortedIDs(g.Minions) {
					if mn := g.Minions[id]; mn != nil && engine.BaseCodeOf(mn.Code) == code {
						t.Target = id
						return nil
					}
				}
			}
			// lowest-ATK Marauder fallback
			lowest, lowID := 99, engine.EntityID("")
			for _, id := range cardutil.SortedEnemyIDs(g) {
				enemy := g.Entity(id)
				if !enemy.EDef().HasTrait("Marauder") {
					continue
				}
				if v := g.AttackValueOf(id); v < lowest {
					lowest, lowID = v, id
				}
			}
			t.Target = lowID
			return nil
		},
	})

	// 40108 Bushwhack: defeat summons a Marauder to the defeater.
	engine.RegisterBehavior("40108", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			p := g.Player(cardutil.FirstPlayerID(g))
			if p == nil {
				return nil
			}
			for i, c := range append(engine.CardList{}, g.EncounterDeck...) {
				if c.Def().Type == "minion" && c.Def().HasTrait("Marauder") {
					g.EncounterDeck.Remove(c.ID)
					_ = i
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
			}
			return nil
		},
	})

	// 40109 Pure Force: crisis while Blockbuster lives.
	engine.RegisterBehavior("40109", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			s := g.SideSchemes[e.EID()]
			if s == nil {
				return nil
			}
			if marauderInPlay(g, "40071") {
				s.Crisis = true
				g.Logf("Pure Force gains the crisis icon")
			}
			return nil
		},
	})

	// 40110 Dizzying Deeds: exhaust + rider stuns/damage.
	engine.RegisterBehavior("40110", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			msgs := []engine.Message{engine.ExhaustEntity{ID: p.ID}}
			if marauderInPlay(g, "40070") {
				msgs = append(msgs, engine.StunEntity{Target: p.ID})
			}
			if marauderInPlay(g, "40075") {
				msgs = append(msgs, engine.IndirectDamage{Player: p.ID, N: 3})
			}
			if marauderInPlay(g, "40076") {
				msgs = append(msgs, engine.ConfuseEntity{Target: p.ID})
			}
			return msgs
		},
	})

	// 40111 Tag Team: minions activate, or mill for a Marauder.
	engine.RegisterBehavior("40111", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var activate []engine.Message
			for _, id := range cardutil.SortedIDs(g.Minions) {
				if mn := g.Minions[id]; mn != nil && mn.EngagedWith == p.ID && mn.EDef().HasTrait("Marauder") {
					activate = append(activate, engine.MinionActivates{MinionID: id, Player: p.ID})
				}
			}
			if len(activate) > 0 {
				return []engine.Message{engine.AskQuestion{
					Player: p.ID,
					Question: engine.Ask("Tag Team — choose:",
						engine.Choice{ID: "activate", Label: "Each Marauder engaged with you activates", Kind: engine.ChoiceLabel}.Msgs(activate...),
						engine.Choice{ID: "mill", Label: "Mill 7 for a Marauder minion", Kind: engine.ChoiceLabel}.
							Msgs(tagTeamMill(g, p)...),
					),
				}}
			}
			return tagTeamMill(g, p)
		},
	})
}

func registerNastyBoys() {
	// 40112 Gorgeous George: exhaust a character when he attacks.
	engine.RegisterBehavior("40112", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AskAttack)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			if p := g.Player(m.Player); p != nil && !p.Exhausted {
				g.Logf("Gorgeous George forces %s to exhaust", p.Name)
				return []engine.Message{engine.ExhaustEntity{ID: p.ID}}
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if p := g.Player(cardutil.FirstPlayerID(g)); p != nil && !p.Exhausted {
				return []engine.Message{engine.ExhaustEntity{ID: p.ID}}
			}
			return nil
		},
	})

	// 40113 Hairbag: surge keyword is data-driven; boost shuffles him back.
	engine.RegisterBehavior("40113", &engine.Behavior{
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			for _, v := range g.Villains {
				for i, c := range v.RevealedBoosts {
					if c.ID == card.ID {
						v.RevealedBoosts = append(v.RevealedBoosts[:i:i], v.RevealedBoosts[i+1:]...)
						g.EncounterDeck = append(g.EncounterDeck, c)
						g.ShuffleEncounterDeck()
						g.Logf("Hairbag shuffles back into the encounter deck")
						return nil
					}
				}
			}
			if _, ok := g.EncounterDiscard.Remove(card.ID); ok {
				g.EncounterDeck = append(g.EncounterDeck, card)
				g.ShuffleEncounterDeck()
				g.Logf("Hairbag shuffles back into the encounter deck")
			}
			return nil
		},
	})

	// 40114 Ramrod: piercing riders not modeled.
	engine.RegisterBehavior("40114", &engine.Behavior{})

	// 40115 Ruckus: stun each character you control.
	engine.RegisterBehavior("40115", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return nil
			}
			p := g.Player(mn.EngagedWith)
			var msgs []engine.Message
			msgs = append(msgs, engine.StunEntity{Target: p.ID})
			for _, id := range p.Allies {
				msgs = append(msgs, engine.StunEntity{Target: id})
			}
			return msgs
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return stunBoost(g, card)
		},
	})

	// 40116 Slab: growth counters boost his attacks.
	engine.RegisterBehavior("40116", &engine.Behavior{
		EnemyStatBonus: func(g *engine.Game, e engine.Entity) (atk, sch int) {
			mn := g.Minions[e.EID()]
			if mn == nil {
				return 0, 0
			}
			return mn.Counters, 0
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AskAttack)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			mn := g.Minions[e.EID()]
			if mn != nil {
				mn.Counters++
				g.Logf("Slab grows (+%d ATK this attack)", mn.Counters)
			}
			return nil
		},
	})

	// 40117 Get Nasty: minions +1 ATK aura (engine attackValue); extra
	// threat per minion in play.
	engine.RegisterBehavior("40117", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			s := g.SideSchemes[e.EID()]
			if s == nil {
				return nil
			}
			n := 0
			for _, mn := range g.Minions {
				if mn == nil {
					continue
				}
				if mn.EDef().HasTrait("Nasty Boy") {
					n += 2
				} else {
					n++
				}
			}
			s.Threat += n
			g.Logf("Get Nasty starts with %d threat (%d minions)", n, len(g.Minions))
			return nil
		},
	})
}

// marauderInPlay reports whether a Marauder villain or minion with the
// given base code is in play.
func marauderInPlay(g *engine.Game, base string) bool {
	for _, v := range g.Villains {
		if v != nil && engine.BaseCodeOf(v.Code) == base {
			return true
		}
	}
	for _, mn := range g.Minions {
		if mn != nil && engine.BaseCodeOf(mn.Code) == base {
			return true
		}
	}
	return false
}

// titleInPlay reports whether any entity in play shares the card title.
func titleInPlay(g *engine.Game, name string) bool {
	for _, v := range g.Villains {
		if v != nil && v.EDef().Name == name {
			return true
		}
	}
	for _, mn := range g.Minions {
		if mn != nil && mn.EDef().Name == name {
			return true
		}
	}
	return false
}

// strongestVillain returns the first villain (single-active-villain
// scenarios).
func strongestVillain(g *engine.Game) engine.EntityID {
	for id := range g.Villains {
		return id
	}
	return ""
}

func highestATKEnemy(g *engine.Game) engine.EntityID {
	best, bestID := -1, engine.EntityID("")
	for _, id := range cardutil.SortedEnemyIDs(g) {
		if v := g.AttackValueOf(id); v > best {
			best, bestID = v, id
		}
	}
	return bestID
}

func fewestHPEnemy(g *engine.Game) engine.EntityID {
	best, bestID := 1<<30, engine.EntityID("")
	for _, id := range cardutil.SortedEnemyIDs(g) {
		e := g.Entity(id)
		var hp int
		switch t := e.(type) {
		case *engine.Villain:
			hp = t.HP()
		case *engine.Minion:
			hp = t.HP()
		default:
			continue
		}
		if hp < best {
			best, bestID = hp, id
		}
	}
	return bestID
}

