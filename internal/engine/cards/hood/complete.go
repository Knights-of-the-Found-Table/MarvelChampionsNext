// Package hood registers "The Hood" scenario: a modular-swarm villain
// built on Foul Play (discard the encounter deck, eat non-Hood cards as
// facedown encounter cards) plus eleven modular sets.
package hood

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// modularSets are the set-aside modular sets Foul Play shuffles in.
var modularSets = []string{
	"beasty_boys", "brothers_grimm", "crossfire_crew", "mister_hyde",
	"ransacked_armory", "sinister_syndicate", "state_of_emergency",
	"streets_of_mayhem", "wrecking_crew_modular",
}

func init() {
	registerScenario()
	registerVillains()
	registerHoodSet()
	registerModulars()
}

// foulPlay discards n encounter cards; non-Hood cards are dealt to the
// player as facedown encounter cards.
func foulPlay(g *engine.Game, pid engine.PlayerID, n int) []engine.Message {
	p := g.Player(pid)
	if p == nil {
		return nil
	}
	for i := 0; i < n; i++ {
		c, ok := g.DrawEncounter()
		if !ok {
			return nil
		}
		if c.Def().CardSet == "the_hood" {
			g.EncounterDiscard = append(g.EncounterDiscard, c)
			g.TLogf("c.foulPlayDiscardsHoodSet", c)
		} else {
			p.EncounterDown = append(p.EncounterDown, c)
			g.TLogf("c.foulPlayDealsToFacedown", c, p.Name)
		}
	}
	return nil
}

// shuffleRandomModular folds one random set-aside modular set into the
// encounter deck.
func shuffleRandomModular(g *engine.Game) {
	// Deterministic pick over the pool of sets not yet folded in.
	avail := []string{}
	for _, s := range modularSets {
		if !g.UsedThisRound["hood-mod-"+s] {
			avail = append(avail, s)
		}
	}
	if len(avail) == 0 {
		return
	}
	pick := avail[g.Random(len(avail))]
	g.UsedThisRound["hood-mod-"+pick] = true
	for _, c := range engine.EncounterSetCards(pick) {
		def := c.Def()
		if def.Category != "encounter" || def.Type == "obligation" {
			continue
		}
		card := engine.Card{ID: g.NextCardID(), Code: def.Code}
		g.EncounterDeck = append(g.EncounterDeck, card)
	}
	g.ShuffleEncounterDeck()
	g.TLogf("c.theModularSetShufflesIntoTheEncounterDeck", pick)
}

func registerScenario() {
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "24004",
		Name:             "The Hood — Making Connections",
		VillainBases:     []string{"24001"},
		MainSchemeStages: []string{"24004b", "24005b", "24006b"},
		ExtraSets:        []string{"standard_ii"},
		Setup: func(g *engine.Game) []engine.Message {
			shuffleRandomModular(g)
			return nil
		},
	})
	// 1A contents marker.
	engine.RegisterBehavior("24004", &engine.Behavior{})
}

func registerVillains() {
	// The Hood I/II/III: When Revealed (stage advance) folds a random
	// modular set in; Foul Play scales per stage.
	for _, base := range []string{"24001", "24002", "24003"} {
		n := 1
		if base != "24001" {
			n = 2
		}
		engine.RegisterBehavior(base, &engine.Behavior{
			VillainStage: func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
				if base != "24001" {
					shuffleRandomModular(g)
				}
				return nil
			},
			VillainActivate: func(g *engine.Game, v *engine.Villain, p *engine.Player) []engine.Message {
				if p.IsHero() {
					if v.Stunned {
						v.Stunned = false
						g.TLogf("log.stunnedCanceled", v)
						return nil
					}
					g.TLogf("log.attacks", v, p.Name)
					g.Push(engine.DealBoost{Enemy: v.ID})
					g.Push(engine.RevealBoost{Enemy: v.ID})
					g.Push(engine.AskAttack{Enemy: v.ID, Player: p.ID, Trigger: engine.TriggerVillainAttacksYou})
					return nil
				}
				if v.Confused {
					v.Confused = false
					return nil
				}
				g.TLogf("log.schemesAgainst", v, p.Name)
				g.Push(engine.DealBoost{Enemy: v.ID})
				g.Push(engine.RevealBoost{Enemy: v.ID})
				return []engine.Message{engine.ApplyVillainScheme{VillainID: v.ID, Player: p.ID}}
			},
			// Foul Play rides the after-activation window.
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				w, ok := msg.(engine.WindowAfterEnemyAttacked)
				if !ok || w.Enemy != e.EID() {
					return nil
				}
				return foulPlay(g, w.Player, n)
			},
		})
	}

	// Main scheme reveals.
	engine.RegisterBehavior("24005", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			shuffleRandomModular(g)
			return []engine.Message{engine.AddAccelerationToken{Scheme: s.ID}}
		},
	})
	engine.RegisterBehavior("24006", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			shuffleRandomModular(g)
			msgs := []engine.Message{engine.AddAccelerationToken{Scheme: s.ID}}
			// Each player resolves Foul Play (via a carrier message).
			for _, p := range g.Players {
				msgs = append(msgs, engine.HoodFoulPlay{Player: p.ID, N: 2})
			}
			return msgs
		},
	})

	// Foul Play carrier for the scheme stage.
	engine.RegisterBehavior("hood-foulplay", &engine.Behavior{})
}

func registerHoodSet() {
	// Established Dominance: Foul Play after The Hood activates against
	// you; removal via exhaust + 2 threat.
	engine.RegisterBehavior("24007", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, p := range g.Players {
				t.Target = p.ID
				break
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowAfterEnemyAttacked)
			a := g.Attachments[e.EID()]
			if !ok || a == nil || w.Player != a.Target {
				return nil
			}
			for id := range g.Villains {
				if id == w.Enemy {
					return []engine.Message{engine.HoodFoulPlay{Player: a.Target, N: 1}}
				}
			}
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustYourIdentity2ThreatDiscardEstablishedDominance"), Type: engine.AbilityAction,
				AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					a := g.Attachments[self]
					if a == nil {
						return nil
					}
					g.Delete(self)
					g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: a.Code})
					msgs := []engine.Message{engine.ExhaustEntity{ID: a.Target}}
					if g.MainScheme != nil {
						msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: a.Target})
					}
					return msgs
				},
			}}
		},
	})

	// The Hood's Mantle: retaliate 1 (steady not modeled); removal.
	engine.RegisterBehavior("24008", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for id := range g.Villains {
				t.Target = id
				break
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowDefended)
			a := g.Attachments[e.EID()]
			if !ok || a == nil || w.Against != a.Target {
				return nil
			}
			return []engine.Message{engine.DamageEntity{Target: w.Defender, Damage: 1, Source: a.Target}}
		},
		Abilities: iconRemoval("Spend [energy][mental][physical] → discard The Hood's Mantle", "energy:1 mental:1 physical:1", 3),
	})

	// The Hood's Pistol: removal + boost reveal.
	engine.RegisterBehavior("24009", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for id := range g.Villains {
				t.Target = id
				break
			}
			return nil
		},
		Abilities: iconRemoval("Spend [mental][physical] → discard The Hood's Pistol", "mental:1 physical:1", 2),
	})

	// Madame Masque: Foul Play on reveal and on defeat.
	engine.RegisterBehavior("24010", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if mn := g.Minions[e.EID()]; mn != nil && mn.EngagedWith != "" {
				return []engine.Message{engine.HoodFoulPlay{Player: mn.EngagedWith, N: 1}}
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if d, ok := msg.(engine.MinionDefeated); ok && d.MinionID == e.EID() {
				// The defeating player is approximated to the engaged
				// player.
				if mn := g.Minions[e.EID()]; mn != nil && mn.EngagedWith != "" {
					return []engine.Message{engine.HoodFoulPlay{Player: mn.EngagedWith, N: 1}}
				}
			}
			return nil
		},
	})

	// Unbridled Ambition: Hinder 2/hero; Foul Play each villain phase.
	engine.RegisterBehavior("24011", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			msgs := []engine.Message{engine.SchemeThreat{Scheme: e.EID(), N: 2 * len(g.Players), Source: e.EID()}}
			return msgs
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.BeginPhase); !ok || g.SideSchemes[e.EID()] == nil {
				return nil
			}
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, engine.HoodFoulPlay{Player: p.ID, N: 1})
			}
			return msgs
		},
	})

	// Field Recruitment / Upper Hand.
	engine.RegisterBehavior("24012", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			shuffleRandomModular(g)
			return []engine.Message{engine.HoodFoulPlay{Player: p.ID, N: 1}}
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return []engine.Message{engine.HoodFoulPlay{Player: engine.PlayerID(card.Owner), N: 1}}
		},
	})
	engine.RegisterBehavior("24013", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var villain engine.EntityID
			for id := range g.Villains {
				villain = id
			}
			msgs := []engine.Message{}
			if !p.IsHero() {
				msgs = append(msgs, engine.DealBoost{Enemy: villain}, engine.RevealBoost{Enemy: villain},
					engine.ApplyVillainScheme{VillainID: villain, Player: p.ID})
			} else {
				msgs = append(msgs, engine.DealBoost{Enemy: villain}, engine.RevealBoost{Enemy: villain},
					engine.AskAttack{Enemy: villain, Player: p.ID, Trigger: engine.TriggerVillainAttacksYou})
			}
			msgs = append(msgs, engine.HoodFoulPlay{Player: p.ID, N: 1})
			return msgs
		},
	})
}

func registerModulars() {
	// ---- Beasty Boys ----
	// Beast Mode: +1 damage to stunned/confused friendlies (not enforced).
	engine.RegisterBehavior("24014", &engine.Behavior{})
	// Griffin: stun on damaging attacks; shuffle back if a friendly is
	// stunned when he dies.
	engine.RegisterBehavior("24015", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.DamageEntity)
			if !ok || d.Source != e.EID() {
				return nil
			}
			switch d.Target.Kind() {
			case engine.KindPlayer, engine.KindAlly:
				return []engine.Message{engine.StunEntity{Target: d.Target}}
			}
			return nil
		},
	})
	// Mandrill: retaliate = confused count; confuses your characters.
	engine.RegisterBehavior("24016", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(minionEngaged(g, e.EID()))
			if p == nil {
				return nil
			}
			var msgs []engine.Message
			msgs = append(msgs, engine.ConfuseEntity{Target: p.ID})
			for _, id := range p.Allies {
				msgs = append(msgs, engine.ConfuseEntity{Target: id})
			}
			return msgs
		},
	})
	// Double Trouble: stun + confuse your characters.
	engine.RegisterBehavior("24017", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			msgs := []engine.Message{engine.StunEntity{Target: p.ID}, engine.ConfuseEntity{Target: p.ID}}
			if len(p.Allies) > 0 {
				msgs = append(msgs, engine.StunEntity{Target: p.Allies[0]})
			}
			return msgs
		},
	})

	// ---- Brothers Grimm ----
	engine.RegisterBehavior("24018", &engine.Behavior{})
	for _, code := range []string{"24019", "24020", "24021", "24022"} {
		engine.RegisterBehavior(code, mysticAttachment(code))
	}

	// ---- Crossfire's Crew ----
	// Out for Blood: damage the weakest friendly repeatedly.
	engine.RegisterBehavior("24023", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := weakestFriendly(g)
			if p == "" {
				return nil
			}
			return []engine.Message{engine.DamageEntity{Target: p, Damage: 1, Source: e.EID()}}
		},
	})
	// Controller: attacks scaled by target ATK (approximated to +2 flat).
	engine.RegisterBehavior("24024", &engine.Behavior{})
	// Corruptor: exhaust allies; threat per exhausted.
	engine.RegisterBehavior("24025", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(minionEngaged(g, e.EID()))
			if p == nil || g.MainScheme == nil {
				return nil
			}
			n := 0
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && !a.Exhausted {
					a.Exhausted = true
					n++
				}
			}
			if n > 0 {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: n, Source: e.EID()}}
			}
			return nil
		},
	})
	// Crossfire / Mister Fear.
	engine.RegisterBehavior("24026", &engine.Behavior{})
	engine.RegisterBehavior("24027", &engine.Behavior{
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			p := g.Player(engine.PlayerID(card.Owner))
			if p == nil {
				return nil
			}
			for len(p.Deck) > 0 {
				c := p.Deck[0]
				p.Deck = p.Deck[1:]
				p.Discard = append(p.Discard, c)
				if c.Def().Type == "ally" {
					g.TLogf("c.discardsMisterFear", p.Name, c)
					break
				}
			}
			return nil
		},
	})
	// Caught in the Crossfire / Cruel Intentions / Ruination / Seek and
	// Destroy / Slug It Out.
	engine.RegisterBehavior("24028", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			msgs := []engine.Message{}
			for len(g.EncounterDeck) > 0 {
				c, ok := g.DrawEncounter()
				if !ok {
					return msgs
				}
				if c.Def().Type == "minion" && c.Def().HasTrait("crossfire's_crew") {
					return append(msgs, engine.RevealEncounterCard{Player: p.ID, Card: c})
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			return msgs
		},
	})
	engine.RegisterBehavior("24029", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			msgs := []engine.Message{engine.DealEncounterToPlayer{Player: p.ID}}
			for id := range g.Villains {
				msgs = append(msgs, engine.DealBoost{Enemy: id})
				break
			}
			return msgs
		},
	})
	engine.RegisterBehavior("24030", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			msgs := []engine.Message{}
			if g.MainScheme != nil {
				msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: t.ID})
			}
			for len(g.EncounterDeck) > 0 {
				c, ok := g.DrawEncounter()
				if !ok {
					break
				}
				if c.Def().Type == "side_scheme" {
					msgs = append(msgs, engine.RevealEncounterCard{Player: p.ID, Card: c})
					break
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			for _, sid := range g.Schemes() {
				msgs = append(msgs, engine.SchemeThreat{Scheme: sid, N: 2, Source: t.ID})
			}
			return msgs
		},
	})
	engine.RegisterBehavior("24031", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if p.NemesisDeck == nil {
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			var rest engine.CardList
			var minion *engine.Card
			for _, c := range p.NemesisDeck {
				if c.Def().Type == "minion" && minion == nil {
					minion = &c
					continue
				}
				rest = append(rest, c)
			}
			if minion == nil {
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			p.NemesisDeck = nil
			g.EncounterDeck = append(g.EncounterDeck, rest...)
			g.ShuffleEncounterDeck()
			return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: *minion}}
		},
	})
	engine.RegisterBehavior("24032", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			msgs := []engine.Message{engine.ExhaustEntity{ID: p.ID}, engine.DamageEntity{Target: p.ID, Damage: 2, Source: t.ID}}
			for id := range g.Villains {
				msgs = append(msgs, engine.DealBoost{Enemy: id})
				break
			}
			return msgs
		},
	})

	// ---- Mister Hyde ----
	engine.RegisterBehavior("24033", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return fetchAndReveal(g, "24035")
		},
	})
	engine.RegisterBehavior("24034", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for _, mn := range g.Minions {
				if mn.Code[:5] == "24035" {
					mn.AttackVal += 2
					return []engine.Message{engine.AskAttack{Enemy: mn.ID, Player: engine.PlayerID(minionEngaged(g, e.EID()))}}
				}
			}
			return nil
		},
	})
	engine.RegisterBehavior("24035", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var msgs []engine.Message
			msgs = append(msgs, engine.ToughEntity{Target: e.EID()})
			for _, p := range g.Players {
				msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: 1, Source: e.EID()})
				for _, id := range p.Allies {
					msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: e.EID()})
				}
			}
			return msgs
		},
	})
	engine.RegisterBehavior("24036", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			for _, mn := range g.Minions {
				if mn.Code[:5] == "24034" {
					mn.SchemeVal += 3
					msgs := []engine.Message{engine.SchemeThreat{Scheme: mainScheme(g), N: mn.SchemeVal, Source: mn.ID},
						engine.DamageEntity{Target: mn.ID, Damage: 4, Source: t.ID}}
					return msgs
				}
				if mn.Code[:5] == "24035" {
					return []engine.Message{engine.ToughEntity{Target: mn.ID},
						engine.AskAttack{Enemy: mn.ID, Player: p.ID}}
				}
			}
			return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
		},
	})

	// ---- Ransacked Armory ----
	for _, code := range []string{"24037", "24038", "24039", "24040"} {
		engine.RegisterBehavior(code, strongestMinionAttachment(code))
	}
	// Armored Guard reprint (core set).
	engine.RegisterBehavior("24041", &engine.Behavior{})

	// ---- Sinister Syndicate ----
	engine.RegisterBehavior("24042", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for i, c := range g.EncounterDeck {
				if c.Def().Type == "minion" && c.Def().HasTrait("criminal") {
					g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
					return []engine.Message{engine.RevealEncounterCard{Player: g.EOwnerIfPlayer(), Card: c}}
				}
			}
			g.ShuffleEncounterDeck()
			return []engine.Message{engine.RevealNextEncounter{Player: g.EOwnerIfPlayer()}}
		},
	})
	engine.RegisterBehavior("24043", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.DamageEntity)
			if !ok || d.Source != e.EID() || !d.Target.Is(engine.KindPlayer) {
				return nil
			}
			p := g.Player(d.Target)
			if p == nil {
				return nil
			}
			discardLowestCostUpgrade(g, p)
			return nil
		},
	})
	engine.RegisterBehavior("24044", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.DamageEntity)
			if !ok || d.Source != e.EID() || !d.Target.Is(engine.KindPlayer) {
				return nil
			}
			p := g.Player(d.Target)
			if p == nil {
				return nil
			}
			var msgs []engine.Message
			for _, id := range p.Allies {
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: e.EID()})
			}
			return msgs
		},
	})
	engine.RegisterBehavior("24045", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.DamageEntity)
			if !ok || d.Target != e.EID() || !d.Source.Is(engine.KindPlayer) {
				return nil
			}
			return []engine.Message{engine.StunEntity{Target: d.Source}}
		},
	})
	engine.RegisterBehavior("24046", &engine.Behavior{})
	engine.RegisterBehavior("24047", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			aa, ok := msg.(engine.AskAttack)
			mn := g.Minions[e.EID()]
			if !ok || mn == nil || aa.Enemy != e.EID() {
				return nil
			}
			p := g.Player(aa.Player)
			if p == nil || len(p.Hand) == 0 {
				return nil
			}
			i := g.Random(len(p.Hand))
			c := p.Hand[i]
			p.Hand = append(p.Hand[:i], p.Hand[i+1:]...)
			p.Discard = append(p.Discard, c)
			g.TLogf("c.discardsAtRandomWhiteRabbit", p.Name, c)
			return nil
		},
	})
	engine.RegisterBehavior("24048", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var msgs []engine.Message
			fired := false
			for _, mn := range g.Minions {
				if !mn.EDef().HasTrait("criminal") || mn.EngagedWith == "" {
					continue
				}
				fired = true
				if !p.IsHero() {
					if g.MainScheme != nil {
						msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: mn.SchemeVal, Source: mn.ID})
					}
				} else {
					msgs = append(msgs, engine.AskAttack{Enemy: mn.ID, Player: mn.EngagedWith})
				}
			}
			if !fired {
				msgs = append(msgs, engine.RevealNextEncounter{Player: p.ID})
			}
			return msgs
		},
	})

	// ---- Standard II / Expert II ----
	engine.RegisterBehavior("24049", &engine.Behavior{})
	engine.RegisterBehavior("24050", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			for id := range g.Villains {
				v := g.Villains[id]
				threat := v.SchemeVal + 1
				msgs := []engine.Message{engine.SchemeThreat{Scheme: mainScheme(g), N: threat, Source: v.ID}}
				return append(msgs, engine.ClearBoosts{Enemy: v.ID})
			}
			return nil
		},
	})
	engine.RegisterBehavior("24051", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var msgs []engine.Message
			if !p.IsHero() {
				for i := 0; i < 7; i++ {
					c, ok := g.DrawEncounter()
					if !ok {
						break
					}
					if c.Def().Type == "minion" {
						msgs = append(msgs, engine.RevealEncounterCard{Player: p.ID, Card: c})
						break
					}
					g.EncounterDiscard = append(g.EncounterDiscard, c)
				}
				return msgs
			}
			for id := range g.Villains {
				msgs = append(msgs, engine.DealBoost{Enemy: id}, engine.RevealBoost{Enemy: id},
					engine.AskAttack{Enemy: id, Player: p.ID, Trigger: engine.TriggerVillainAttacksYou})
				break
			}
			for _, mn := range g.Minions {
				if mn.EngagedWith == p.ID {
					msgs = append(msgs, engine.AskAttack{Enemy: mn.ID, Player: p.ID})
				}
			}
			msgs = append(msgs, engine.RevealNextEncounter{Player: p.ID})
			return msgs
		},
	})
	engine.RegisterBehavior("24052", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if discardHighestCostPermanent(g, p) {
				return nil
			}
			return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
		},
	})
	engine.RegisterBehavior("24053", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			return []engine.Message{engine.RevealNemesisSet{Player: p.ID}}
		},
	})
	engine.RegisterBehavior("24054", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if !p.IsHero() {
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			for id := range g.Villains {
				return []engine.Message{engine.DealBoost{Enemy: id}, engine.RevealBoost{Enemy: id},
					engine.AskAttack{Enemy: id, Player: p.ID, Trigger: engine.TriggerVillainAttacksYou}}
			}
			return nil
		},
	})

	// ---- State of Emergency ----
	engine.RegisterBehavior("24055", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(g.EOwnerIfPlayer())
			if p == nil || len(p.Hand) == 0 {
				return nil
			}
			best, cost := 0, -1
			for i, c := range p.Hand {
				if v := deref(c.Def().Cost, 0); v > cost {
					best, cost = i, v
				}
			}
			c := p.Hand[best]
			p.Hand = append(p.Hand[:best], p.Hand[best+1:]...)
			p.Discard = append(p.Discard, c)
			return nil
		},
	})
	engine.RegisterBehavior("24056", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(g.EOwnerIfPlayer())
			if p == nil {
				return nil
			}
			return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 3, Source: e.EID()}}
		},
	})
	engine.RegisterBehavior("24057", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(g.EOwnerIfPlayer())
			if p == nil {
				return nil
			}
			discardLowestCostPermanent(g, p)
			return nil
		},
	})
	engine.RegisterBehavior("24058", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(g.EOwnerIfPlayer())
			if p == nil {
				return nil
			}
			for len(g.EncounterDeck) > 0 {
				c, ok := g.DrawEncounter()
				if !ok {
					return nil
				}
				if c.Def().Type == "minion" {
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			return nil
		},
	})
	engine.RegisterBehavior("24059", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var msgs []engine.Message
			for _, s := range g.SideSchemes {
				msgs = append(msgs, engine.ReplaySideSchemeReveal{Scheme: s.ID})
			}
			if len(msgs) == 0 {
				for _, sid := range g.Schemes() {
					msgs = append(msgs, engine.SchemeThreat{Scheme: sid, N: 2, Source: t.ID})
				}
			}
			return msgs
		},
	})

	// ---- Streets of Mayhem (Setting environments) ----
	for _, code := range []string{"24060", "24061", "24062", "24063"} {
		engine.RegisterBehavior(code, settingEnvironment(code))
	}

	// ---- Wrecking Crew modular ----
	engine.RegisterBehavior("24064", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.SchemeThreat{Scheme: e.EID(), N: 2 * len(g.Players), Source: e.EID()}}
		},
	})
	engine.RegisterBehavior("24065", &engine.Behavior{})
	engine.RegisterBehavior("24066", &engine.Behavior{})
	engine.RegisterBehavior("24067", &engine.Behavior{})
	engine.RegisterBehavior("24068", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.DamageEntity)
			mn := g.Minions[e.EID()]
			if !ok || mn == nil || d.Source != e.EID() || !d.Target.Is(engine.KindPlayer) {
				return nil
			}
			p := g.Player(d.Target)
			if p == nil {
				return nil
			}
			var msgs []engine.Message
			msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: 1, Source: e.EID()})
			for _, id := range p.Allies {
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: e.EID()})
			}
			return msgs
		},
	})
	engine.RegisterBehavior("24069", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var msgs []engine.Message
			fired := false
			for _, mn := range g.Minions {
				if mn.EDef().HasTrait("elite") && mn.EngagedWith != "" {
					fired = true
					msgs = append(msgs, engine.AskAttack{Enemy: mn.ID, Player: mn.EngagedWith})
				}
			}
			if !fired {
				msgs = append(msgs, engine.RevealNextEncounter{Player: p.ID})
			}
			return msgs
		},
	})
	engine.RegisterBehavior("24070", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var msgs []engine.Message
			fired := false
			for _, mn := range g.Minions {
				if mn.EDef().HasTrait("brute") && !mn.Tough {
					fired = true
					msgs = append(msgs, engine.ToughEntity{Target: mn.ID})
				}
			}
			if !fired {
				msgs = append(msgs, discardUntilBrute(g, p)...)
			}
			return msgs
		},
	})
}

// ---- helpers ----

func mainScheme(g *engine.Game) engine.EntityID {
	if g.MainScheme != nil {
		return g.MainScheme.ID
	}
	return ""
}

func minionEngaged(g *engine.Game, id engine.EntityID) engine.PlayerID {
	if mn := g.Minions[id]; mn != nil {
		return mn.EngagedWith
	}
	return ""
}

func weakestFriendly(g *engine.Game) engine.EntityID {
	var best engine.EntityID
	hp := 1 << 30
	for _, p := range g.Players {
		if p.HP() < hp {
			best, hp = p.ID, p.HP()
		}
		for _, id := range p.Allies {
			if a := g.Allies[id]; a != nil && a.HP() < hp {
				best, hp = a.ID, a.HP()
			}
		}
	}
	return best
}

func fetchAndReveal(g *engine.Game, code string) []engine.Message {
	for i, c := range g.EncounterDeck {
		if c.Code[:5] == code {
			g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
			return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
		}
	}
	for i, c := range g.EncounterDiscard {
		if c.Code[:5] == code {
			g.EncounterDiscard = append(g.EncounterDiscard[:i], g.EncounterDiscard[i+1:]...)
			return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
		}
	}
	return nil
}

func discardUntilBrute(g *engine.Game, p *engine.Player) []engine.Message {
	for len(g.EncounterDeck) > 0 {
		c, ok := g.DrawEncounter()
		if !ok {
			return nil
		}
		if c.Def().Type == "minion" && c.Def().HasTrait("brute") {
			return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
		}
		g.EncounterDiscard = append(g.EncounterDiscard, c)
	}
	return nil
}

func mysticAttachment(code string) *engine.Behavior {
	return &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, mn := range g.Minions {
				if mn.EDef().HasTrait("mystic") {
					t.Target = mn.ID
					return nil
				}
			}
			for id := range g.Villains {
				t.Target = id
				return nil
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			a := g.Attachments[e.EID()]
			if a == nil {
				return nil
			}
			var hit bool
			switch m := msg.(type) {
			case engine.VillainActivates:
				hit = m.VillainID == a.Target
			case engine.MinionActivates:
				hit = m.MinionID == a.Target
			}
			if !hit {
				return nil
			}
			g.Delete(a.ID)
			g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: a.Code})
			var pid engine.PlayerID
			switch m := msg.(type) {
			case engine.VillainActivates:
				pid = m.Player
			case engine.MinionActivates:
				pid = m.Player
			}
			var msgs []engine.Message
			if p := g.Player(pid); p != nil {
				switch code {
				case "24019":
					if len(p.Hand) > 0 {
						i := g.Random(len(p.Hand))
						c := p.Hand[i]
						p.Hand = append(p.Hand[:i], p.Hand[i+1:]...)
						p.Discard = append(p.Discard, c)
					}
					msgs = append(msgs, engine.DealEncounterToPlayer{Player: pid})
				case "24020":
					msgs = append(msgs, engine.DamageEntity{Target: pid, Damage: 3, Source: e.EID()},
						engine.DealEncounterToPlayer{Player: pid})
				case "24021":
					msgs = append(msgs, engine.StunEntity{Target: pid},
						engine.DealEncounterToPlayer{Player: pid})
				case "24022":
					// Discard one controlled permanent + facedown card.
					discardLowestCostPermanent(g, p)
					msgs = append(msgs, engine.DealEncounterToPlayer{Player: pid})
				}
			}
			return msgs
		},
	}
}

func strongestMinionAttachment(code string) *engine.Behavior {
	return &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			var best *engine.Minion
			for _, mn := range g.Minions {
				if best == nil || mn.MaxHP > best.MaxHP {
					best = mn
				}
			}
			if best == nil {
				g.Delete(t.ID)
				g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: t.Code})
				return []engine.Message{engine.RevealNextEncounter{Player: cardutil.FirstPlayerID(g)}}
			}
			t.Target = best.ID
			switch code {
			case "24038":
				best.MaxHP += 4
			case "24040":
				best.MaxHP += 3
			}
			return nil
		},
	}
}

func settingEnvironment(code string) *engine.Behavior {
	return &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			// Discard other Settings.
			for id, env := range g.Environments {
				if id != e.EID() && env.EDef().HasTrait("setting") {
					g.Delete(id)
					g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: env.Code})
				}
			}
			switch code {
			case "24060":
				for _, p := range g.Players {
					p.BonusATK++
				}
				for _, mn := range g.Minions {
					mn.AttackVal++
				}
			case "24061":
				for _, p := range g.Players {
					p.BonusTHW++
				}
				for _, mn := range g.Minions {
					mn.SchemeVal++
				}
			case "24062":
				for _, p := range g.Players {
					p.BonusATK++ // approximates retaliate for all
				}
			case "24063":
				// steady for everything — not modeled.
			}
			g.TLogf("c.entersPlay", e)
			return nil
		},
	}
}

func discardLowestCostUpgrade(g *engine.Game, p *engine.Player) {
	var best engine.EntityID
	cost := 1 << 30
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil {
			if c := deref(u.EDef().Cost, 0); c < cost {
				best, cost = id, c
			}
		}
	}
	if best != "" {
		g.Push(engine.DiscardControlled{Player: p.ID, ID: best})
	}
}

func discardLowestCostPermanent(g *engine.Game, p *engine.Player) {
	var best engine.EntityID
	cost := 1 << 30
	for _, id := range p.Supports {
		if s := g.Supports[id]; s != nil {
			if c := deref(s.EDef().Cost, 0); c < cost {
				best, cost = id, c
			}
		}
	}
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil {
			if c := deref(u.EDef().Cost, 0); c < cost {
				best, cost = id, c
			}
		}
	}
	if best != "" {
		g.Push(engine.DiscardControlled{Player: p.ID, ID: best})
	}
}

func discardHighestCostPermanent(g *engine.Game, p *engine.Player) bool {
	var best engine.EntityID
	cost := -1
	for _, id := range p.Supports {
		if s := g.Supports[id]; s != nil {
			if c := deref(s.EDef().Cost, 0); c > cost {
				best, cost = id, c
			}
		}
	}
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil {
			if c := deref(u.EDef().Cost, 0); c > cost {
				best, cost = id, c
			}
		}
	}
	if best != "" {
		g.Push(engine.DiscardControlled{Player: p.ID, ID: best})
		return true
	}
	return false
}

func iconRemoval(label, icons string, cost int) func(g *engine.Game, e engine.Entity) []engine.Ability {
	return func(g *engine.Game, e engine.Entity) []engine.Ability {
		return []engine.Ability{{
			Label: engine.S(label), Type: engine.AbilityAction, Cost: cost, CostIcons: icons, HeroOnly: true,
			Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
				a := g.Attachments[self]
				if a == nil {
					return nil
				}
				g.Delete(self)
				g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: a.Code})
				return nil
			},
		}}
	}
}

func deref(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}
