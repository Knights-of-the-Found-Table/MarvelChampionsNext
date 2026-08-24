// mg_brotherhood.go implements The Brotherhood Strikes scenario content:
// the four Brotherhood villains (32121–32124), the mansion locations
// (32125–32137) and the Brotherhood modular set (32073–32079).
package mutantgenesis

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerBrotherhoodVillains()
	registerMansionAttack()
	registerBrotherhoodModular()
}

// brotherhoodVillainByBase finds a villain by base code.
func villainByBase(g *engine.Game, base string) *engine.Villain {
	for _, v := range g.Villains {
		if v != nil && engine.BaseCodeOf(v.Code) == base {
			return v
		}
	}
	return nil
}

// activateOrSearch activates the named villain against the player,
// searching it from the encounter zones if needed (Ground Swell family).
func activateOrSearch(g *engine.Game, base string, pid engine.PlayerID) []engine.Message {
	if v := villainByBase(g, base); v != nil {
		if p := g.Player(pid); p != nil && p.IsHero() {
			return []engine.Message{engine.AskAttack{Enemy: v.ID, Player: pid, Trigger: engine.TriggerVillainAttacksYou}}
		}
		if g.MainScheme != nil {
			return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: v.SchemeVal, Source: v.ID}}
		}
		return nil
	}
	for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
		for _, c := range *zone {
			if engine.BaseCodeOf(c.Code) == base {
				zone.Remove(c.ID)
				return []engine.Message{engine.RevealEncounterCard{Player: pid, Card: c}}
			}
		}
	}
	return nil
}

func registerBrotherhoodVillains() {
	// The four villains are plain stat blocks driven by their modular
	// support cards.
	for _, code := range []string{"32121", "32122", "32123", "32124"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}
}

func registerMansionAttack() {
	// 32125 The Brotherhood Strikes!: When Revealed — each player gets a
	// facedown encounter card, then the scheme advances (victory display
	// and loss handling approximate the location rush).
	engine.RegisterBehavior("32125", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, engine.DealEncounterToPlayer{Player: p.ID})
			}
			return msgs
		},
	})

	// 32126–32129 The Atrium/Cafeteria/Basketball Court/Courtyard: their
	// global buffs are not modeled; completing each advances via the
	// ReplaceMainScheme flow (the 3-loss counter approximated by counting
	// victory-display schemes).
	for _, code := range []string{"32126", "32127", "32128", "32129"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	// 32130 Save the School: after the villain is defeated, reveal the
	// next one; all four defeated = win.
	engine.RegisterBehavior("32130", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.VillainDefeated); !ok {
				return nil
			}
			remaining := 0
			for _, v := range g.Villains {
				if v != nil {
					remaining++
				}
			}
			if remaining == 0 {
				return []engine.Message{engine.GameOver{Won: true, Reason: "The X-Mansion stands — every Brotherhood villain is defeated"}}
			}
			g.Logf("Another Brotherhood villain reveals themselves!")
			return nil
		},
	})

	// 32131 Brotherhood Beatdown: punish per Brotherhood member in play.
	engine.RegisterBehavior("32131", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var msgs []engine.Message
			if villainByBase(g, "32121") != nil { // Avalanche
				msgs = append(msgs, engine.ExhaustEntity{ID: p.ID})
			}
			if villainByBase(g, "32122") != nil { // Blob
				msgs = append(msgs, engine.StunEntity{Target: p.ID})
			}
			if villainByBase(g, "32123") != nil { // Pyro
				msgs = append(msgs, engine.IndirectDamage{Player: p.ID, N: 2})
			}
			if villainByBase(g, "32124") != nil { // Toad
				if len(p.Hand) > 0 {
					i := g.Random(len(p.Hand))
					c := p.Hand[i]
					p.Hand = append(p.Hand[:i], p.Hand[i+1:]...)
					p.Discard = append(p.Discard, c)
				}
			}
			if len(msgs) == 0 {
				// The modular minions stand in for their bosses.
				for _, mn := range g.Minions {
					if mn != nil && mn.EDef().HasTrait("brotherhood") {
						msgs = append(msgs, engine.ExhaustEntity{ID: p.ID})
						break
					}
				}
			}
			return msgs
		},
	})

	// 32132–32135 Ground Swell / Immovable / Pyromaniac / Hopping Mad:
	// activate (or search) the matching villain.
	activationTreachery := func(base string) {
		engine.RegisterBehavior(base, &engine.Behavior{})
	}
	_ = activationTreachery
	engine.RegisterBehavior("32132", treacheryActivate("32121"))
	engine.RegisterBehavior("32133", treacheryActivate("32122"))
	engine.RegisterBehavior("32134", treacheryActivate("32123"))
	engine.RegisterBehavior("32135", treacheryActivate("32124"))

	// 32136 Protect the Students: Hinder 2/hero; When Defeated — search
	// your deck for an ally.
	engine.RegisterBehavior("32136", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			// Approximation: the first player draws 1 (the ally search is
			// the defeater's, who is unknown here).
			return []engine.Message{engine.DrawCards{Player: cardutil.FirstPlayerID(g), N: 1}}
		},
	})

	// 32137 Under Siege: When Revealed — 3 extra threat per Brotherhood
	// character.
	engine.RegisterBehavior("32137", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			n := 0
			for _, mn := range g.Minions {
				if mn != nil && mn.EDef().HasTrait("brotherhood") {
					n++
				}
			}
			if n > 0 {
				return []engine.Message{engine.SchemeThreat{Scheme: e.EID(), N: 3 * n, Source: e.EID()}}
			}
			return nil
		},
	})
}

// treacheryActivate builds a When-Revealed that activates the villain
// with the given base code against the revealer.
func treacheryActivate(base string) *engine.Behavior {
	return &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			return activateOrSearch(g, base, p.ID)
		},
	}
}

func registerBrotherhoodModular() {
	// 32073 Avalanche: after he attacks you, exhaust a character you
	// control (approximation: exhaust the identity).
	engine.RegisterBehavior("32073", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AskAttack)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			return []engine.Message{engine.ExhaustEntity{ID: m.Player}}
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			pid := cardutil.FirstPlayerID(g)
			return []engine.Message{engine.ExhaustEntity{ID: pid}}
		},
	})

	// 32074 Blob: after he attacks and damages, stun that character
	// (approximation: stun the defender).
	engine.RegisterBehavior("32074", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AskAttack)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			return []engine.Message{engine.StunEntity{Target: m.Player}}
		},
	})

	// 32075 Pyro: after he attacks you, discard top 2 of your deck — 1
	// indirect damage per printed resource icon.
	engine.RegisterBehavior("32075", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AskAttack)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil {
				return nil
			}
			n := 0
			for i := 0; i < 2 && len(p.Deck) > 0; i++ {
				top := p.Deck[0]
				p.Deck = p.Deck[1:]
				n += len(top.Def().Resources)
				p.Discard = append(p.Discard, top)
			}
			if n > 0 {
				return []engine.Message{engine.IndirectDamage{Player: p.ID, N: n}}
			}
			return nil
		},
	})

	// 32076 Toad: after he attacks and damages, discard 1 random card.
	engine.RegisterBehavior("32076", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AskAttack)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil || len(p.Hand) == 0 {
				return nil
			}
			i := g.Random(len(p.Hand))
			c := p.Hand[i]
			p.Hand = append(p.Hand[:i], p.Hand[i+1:]...)
			p.Discard = append(p.Discard, c)
			g.Logf("Toad makes %s discard %s", p.Name, c.Def().Name)
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			pid := cardutil.FirstPlayerID(g)
			p := g.Player(pid)
			if p == nil || len(p.Hand) == 0 {
				return nil
			}
			i := g.Random(len(p.Hand))
			c := p.Hand[i]
			p.Hand = append(p.Hand[:i], p.Hand[i+1:]...)
			p.Discard = append(p.Discard, c)
			return nil
		},
	})

	// 32077 Homo Superior: attach to a minion (+5 HP, tough).
	engine.RegisterBehavior("32077", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn == nil {
					continue
				}
				t.Target = mn.ID
				mn.Attachments = append(mn.Attachments, t.ID)
				mn.MaxHP += 5
				return []engine.Message{engine.ToughEntity{Target: mn.ID}}
			}
			g.Delete(t.ID)
			return []engine.Message{engine.RevealNextEncounter{Player: cardutil.FirstPlayerID(g)}}
		},
	})

	// 32078 Mutant Terrorists: search The Brotherhood side scheme and
	// reveal it, else reveal a Brotherhood minion.
	engine.RegisterBehavior("32078", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
				for _, c := range *zone {
					if c.Code == "32079" {
						zone.Remove(c.ID)
						return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
					}
				}
			}
			for guards := 0; guards < 40; guards++ {
				if len(g.EncounterDeck) == 0 {
					return nil
				}
				top := g.EncounterDeck[0]
				g.EncounterDeck = g.EncounterDeck[1:]
				if top.Def().Type == "minion" && top.Def().HasTrait("brotherhood") {
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: top}}
				}
				g.EncounterDiscard = append(g.EncounterDiscard, top)
			}
			return nil
		},
	})

	// 32079 The Brotherhood: Hinder 2/hero; quickstrike aura not modeled.
	engine.RegisterBehavior("32079", &engine.Behavior{})
}
