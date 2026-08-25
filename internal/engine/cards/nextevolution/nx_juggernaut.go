package nextevolution

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// juggernaut finds the Juggernaut villain.
func juggernaut(g *engine.Game) *engine.Villain {
	for _, v := range g.Villains {
		if v != nil && engine.BaseCodeOf(v.Code) == "40118" {
			return v
		}
	}
	return nil
}

// addMomentum places a momentum counter and logs it.
func addMomentum(g *engine.Game, n int) {
	v := juggernaut(g)
	if v == nil {
		return
	}
	v.Counters += n
	g.TLogf("c.juggernautGainsAMomentumCounterTotal", v.Counters)
}

func registerJuggernaut() {
	// 40118-40120 Juggernaut stages: +1 ATK per momentum counter; when
	// revealed +1 momentum + tough (stage III also fetches Head of
	// Steam — see the VillainStage hook).
	engine.RegisterBehavior("40118", &engine.Behavior{
		EnemyStatBonus: func(g *engine.Game, e engine.Entity) (atk, sch int) {
			v, ok := e.(*engine.Villain)
			if !ok {
				return 0, 0
			}
			return v.Counters, 0
		},
		VillainStage: func(g *engine.Game, v *engine.Villain, nextStage int) []engine.Message {
			addMomentum(g, 1)
			v.Tough = true
			if v.Code == "40120" {
				// Search the encounter deck and discard pile for Head of
				// Steam and reveal it.
				for _, c := range append(engine.CardList{}, g.EncounterDeck...) {
					if c.Code == "40123" {
						g.EncounterDeck.Remove(c.ID)
						return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
					}
				}
				for _, c := range append(engine.CardList{}, g.EncounterDiscard...) {
					if c.Code == "40123" {
						g.EncounterDiscard.Remove(c.ID)
						return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
					}
				}
			}
			return nil
		},
	})

	// Stages II and III share the base registration.
	engine.RegisterBehavior("40119", engine.LookupBehavior("40118"))
	engine.RegisterBehavior("40120", engine.LookupBehavior("40118"))

	// 40121 The Unstoppable Juggernaut: instead of completing, the ritual
	// fires — handled by the scenario's OnMainSchemeMaxed.
	engine.RegisterBehavior("40121", &engine.Behavior{})

	// 40122a Juggernaut's Helmet: stalwart/overkill not modeled; the
	// momentum purge action is approximated away.
	engine.RegisterBehavior("40122", &engine.Behavior{})

	// 40123 Head of Steam: attach to Juggernaut + momentum; retaliate X
	// lives in the engine's retaliateOf.
	engine.RegisterBehavior("40123", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if v := juggernaut(g); v != nil {
				t.Target = v.ID
			}
			addMomentum(g, 1)
			return nil
		},
	})

	// 40124 Building Momentum: after defending against Juggernaut, shed a
	// threat.
	engine.RegisterBehavior("40124", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowDefended)
			if !ok {
				return nil
			}
			s := g.SideSchemes[e.EID()]
			if s == nil || s.Threat <= 0 {
				return nil
			}
			if v := juggernaut(g); v == nil || v.ID != w.Against {
				return nil
			}
			s.Threat--
			g.TLogf("c.buildingMomentumLoses1ThreatLeft", s.Threat)
			return nil
		},
	})

	// 40125 Breakthrough: take Juggernaut's ATK or lose your best card.
	engine.RegisterBehavior("40125", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			atk := 0
			if v := juggernaut(g); v != nil {
				atk = g.AttackValueOf(v.ID)
			}
			discardMsgs, discardLabel := highestCostControlledDiscard(g, p)
			return []engine.Message{engine.AskQuestion{
				Player: p.ID,
				Question: engine.Ask(engine.Tf("c.breakthroughJuggernautSAtkIs", atk),
					engine.Choice{ID: "dmg", Label: engine.Tf("c.takeDamage", atk), Kind: engine.ChoiceLabel}.
						Msgs(engine.DamageEntity{Target: p.ID, Damage: atk, Source: t.ID}),
					engine.Choice{ID: "disc", Label: engine.S(discardLabel), Kind: engine.ChoiceLabel}.Msgs(discardMsgs...),
				),
			}}
		},
	})

	// 40126 Flatten: damage or momentum; boost: tough.
	engine.RegisterBehavior("40126", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			atk := 0
			vID := engine.EntityID("")
			if v := juggernaut(g); v != nil {
				atk = g.AttackValueOf(v.ID)
				vID = v.ID
			}
			if vID == "" {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player: p.ID,
				Question: engine.Ask(engine.Tf("c.flattenJuggernautSAtkIs", atk),
					engine.Choice{ID: "dmg", Label: engine.Tf("c.takeDamage", atk), Kind: engine.ChoiceLabel}.
						Msgs(engine.DamageEntity{Target: p.ID, Damage: atk, Source: t.ID}),
					engine.Choice{ID: "mom", Label: engine.Tf("c.giveJuggernaut1MomentumCounter"), Kind: engine.ChoiceLabel}.
						Msgs(engine.AddEntityCounter{ID: vID, N: 1}),
				),
			}}
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if id := strongestVillain(g); id != "" {
				return []engine.Message{engine.ToughEntity{Target: id}}
			}
			return nil
		},
	})

	// 40127 Ground Pound: group indirect = ATK; boost: 1 indirect.
	engine.RegisterBehavior("40127", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			atk := 0
			if v := juggernaut(g); v != nil {
				atk = g.AttackValueOf(v.ID)
			}
			per := atk / len(g.Players)
			if per < 1 {
				per = 1
			}
			var msgs []engine.Message
			for _, o := range g.Players {
				msgs = append(msgs, engine.IndirectDamage{Player: o.ID, N: per})
			}
			return msgs
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if p := g.Player(cardutil.FirstPlayerID(g)); p != nil {
				return []engine.Message{engine.IndirectDamage{Player: p.ID, N: 1}}
			}
			return nil
		},
	})

	// 40128 Trample: alter-ego indirect / hero attacks the weakest ally.
	engine.RegisterBehavior("40128", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if !p.IsHero() {
				return []engine.Message{engine.IndirectDamage{Player: p.ID, N: 2}}
			}
			var weakest engine.EntityID
			best := 1 << 30
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && a.HP() < best {
					best, weakest = a.HP(), id
				}
			}
			v := juggernaut(g)
			if v == nil {
				return nil
			}
			if weakest != "" {
				return []engine.Message{engine.DamageEntity{Target: weakest, Damage: g.AttackValueOf(v.ID), Source: v.ID}}
			}
			return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: g.AttackValueOf(v.ID), Source: v.ID}}
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if p := g.Player(cardutil.FirstPlayerID(g)); p != nil {
				for _, id := range p.Allies {
					return []engine.Message{engine.DamageEntity{Target: id, Damage: 1, Source: engine.EntityID("")}}
				}
			}
			return nil
		},
	})

	// 40129 Cyttorak's Exemplar: flip the helmet + momentum, or threat.
	engine.RegisterBehavior("40129", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			// "If Juggernaut Exposed is in play, flip it" — approximated as
			// removing any Helmet attachment.
			for _, a := range g.Attachments {
				if a != nil && a.Code == "40122a" {
					g.Delete(a.ID)
					g.TLogf("c.juggernautSHelmetFlipsExposed")
					addMomentum(g, 1)
					return nil
				}
			}
			atk := 0
			if v := juggernaut(g); v != nil {
				atk = g.AttackValueOf(v.ID)
			}
			if g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: atk, Source: t.ID}}
			}
			return nil
		},
	})

	// 40130 Hope Summers (setup): leaves play → the players lose. Stats
	// are set by the scenario setups.
	engine.RegisterBehavior("40130", &engine.Behavior{
		AllyDefeatInterrupt: func(g *engine.Game, a *engine.Ally, destroy func()) []engine.Message {
			return []engine.Message{engine.GameOver{Won: false, Reason: engine.Tf("reason.hopeLeftPlay")}}
		},
	})

	// 40131 Captive Hope: Hope cannot ready (engine); exhaust her on
	// reveal.
	engine.RegisterBehavior("40131", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for _, p := range g.Players {
				for _, id := range p.Allies {
					if a := g.Allies[id]; a != nil && engine.BaseCodeOf(a.Code) == "40130" {
						return []engine.Message{engine.ExhaustEntity{ID: id}}
					}
				}
			}
			return nil
		},
	})

	// 40132 Black Tom Cassidy: immune while Creeping Willow lives; fetches
	// one on reveal.
	engine.RegisterBehavior("40132", &engine.Behavior{
		MinionDamageable: func(g *engine.Game, m *engine.Minion, damage int) bool {
			if marauderInPlay(g, "40133") {
				g.TLogf("c.blackTomCassidyCannotTakeDamageWhileCreepingWillowIsInPlay")
				return false
			}
			return true
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return nil
			}
			for _, c := range append(engine.CardList{}, g.EncounterDeck...) {
				if c.Code == "40133" {
					g.EncounterDeck.Remove(c.ID)
					g.ShuffleEncounterDeck()
					return []engine.Message{engine.RevealEncounterCard{Player: mn.EngagedWith, Card: c}}
				}
			}
			for _, c := range append(engine.CardList{}, g.EncounterDiscard...) {
				if c.Code == "40133" {
					g.EncounterDiscard.Remove(c.ID)
					return []engine.Message{engine.RevealEncounterCard{Player: mn.EngagedWith, Card: c}}
				}
			}
			return nil
		},
	})

	// 40133 Creeping Willow: stunned after her damaging attacks.
	engine.RegisterBehavior("40133", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.DamageEntity)
			if !ok || m.Source != e.EID() || m.Damage <= 0 {
				return nil
			}
			g.TLogf("c.creepingWillowSThornsStunHerVictim")
			return []engine.Message{engine.StunEntity{Target: m.Target}}
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return stunBoost(g, card)
		},
	})

	// 40134 Making Green: Hinder 2 parsed from text; Willow surge not
	// modeled.
	engine.RegisterBehavior("40134", &engine.Behavior{})

	// 40135 A Sound Thrashing: each Creeping Willow attacks; otherwise
	// fetch one.
	engine.RegisterBehavior("40135", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var msgs []engine.Message
			attacked := false
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn != nil && mn.Code == "40133" && mn.EngagedWith != "" {
					msgs = append(msgs, engine.AskAttack{Enemy: id, Player: mn.EngagedWith})
					attacked = true
				}
			}
			if !attacked {
				for _, c := range append(engine.CardList{}, g.EncounterDeck...) {
					if c.Code == "40133" {
						g.EncounterDeck.Remove(c.ID)
						return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
					}
				}
			}
			return msgs
		},
	})
}

// highestCostControlledDiscard finds the player's highest-cost upgrade or
// support and builds the discard payload (Breakthrough).
func highestCostControlledDiscard(g *engine.Game, p *engine.Player) ([]engine.Message, string) {
	best := -1
	var label string
	var msgs []engine.Message
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil {
			if c := cardutil.Cost(u.EDef()); c > best {
				best, label = c, "Discard "+u.EDef().Name
				msgs = []engine.Message{engine.DiscardControlled{Player: p.ID, ID: id}}
			}
		}
	}
	for _, id := range p.Supports {
		if s2 := g.Supports[id]; s2 != nil {
			if c := cardutil.Cost(s2.EDef()); c > best {
				best, label = c, "Discard "+s2.EDef().Name
				msgs = []engine.Message{engine.DiscardControlled{Player: p.ID, ID: id}}
			}
		}
	}
	if best < 0 {
		return nil, "Discard nothing (you control no upgrades or supports)"
	}
	return msgs, label
}
