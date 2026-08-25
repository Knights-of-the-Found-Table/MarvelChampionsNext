package nextevolution

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// stryfeVillain finds the Stryfe villain.
func stryfeVillain(g *engine.Game) *engine.Villain {
	for _, v := range g.Villains {
		if v != nil && engine.BaseCodeOf(v.Code) == "40163" {
			return v
		}
	}
	return nil
}

// handTypeMax returns the highest most-common-type count across all
// players (Stryfe's scaling approximation).
func handTypeMax(g *engine.Game) int {
	best := 0
	for _, p := range g.Players {
		if n := mostCommonHandType(p); n > best {
			best = n
		}
	}
	return best
}

func registerStryfe() {
	// 40163-40165 Stryfe stages: +X ATK (X = most common hand type).
	engine.RegisterBehavior("40163", &engine.Behavior{
		EnemyStatBonus: func(g *engine.Game, e engine.Entity) (atk, sch int) {
			return handTypeMax(g), 0
		},
		// Stage II reveal: mill for a PSIONIC attachment.
		VillainStage: func(g *engine.Game, v *engine.Villain, nextStage int) []engine.Message {
			if v.Code != "40164" {
				return nil
			}
			for i := 0; i < 30; i++ {
				c, ok := g.DrawEncounter()
				if !ok {
					return nil
				}
				if c.Def().Type == "attachment" && c.Def().HasTrait("Psionic") {
					return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			return nil
		},
		// Stage III: after you attack Stryfe, take X damage.
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.DamageEntity)
			if !ok || m.Target != e.EID() || !m.Source.Is(engine.KindPlayer) {
				return nil
			}
			v := stryfeVillain(g)
			if v == nil || v.Code != "40165" {
				return nil
			}
			if p := g.Player(engine.PlayerID(m.Source)); p != nil {
				x := mostCommonHandType(p)
				if x > 0 {
					g.TLogf("c.stryfeLashesOutTakesDamage", p.Name, x)
					return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: x, Source: e.EID()}}
				}
			}
			return nil
		},
	})

	// Stages II and III share the base registration.
	engine.RegisterBehavior("40164", engine.LookupBehavior("40163"))
	engine.RegisterBehavior("40165", engine.LookupBehavior("40163"))

	// 40166 Uncontrollable Power: each player places threat equal to their
	// most common hand type after villain phase step one (the pre-count
	// discard is approximated away).
	engine.RegisterBehavior("40166", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.BeginPhase)
			if !ok || m.Phase != engine.PhaseVillain {
				return nil
			}
			if g.MainScheme == nil || g.MainScheme.EID() != e.EID() {
				return nil
			}
			var msgs []engine.Message
			for _, p := range g.Players {
				n := mostCommonHandType(p)
				if n > 0 {
					g.TLogf("c.placesThreatUncontrollablePower", p.Name, n)
					msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: n, Source: p.ID})
				}
			}
			return msgs
		},
	})

	// 40167 Left to Your Fate: riders live in the engine (hand size, cost).
	engine.RegisterBehavior("40167", &engine.Behavior{})

	// 40168a Stryfe's Grasp: Hinder parsed from text; the Living Bomb
	// flip is approximated as a log line (the scheme stays in play with
	// its threat).
	engine.RegisterBehavior("40168", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			g.TLogf("c.stryfeSGraspFlipsLivingBombIsRevealed")
			return nil
		},
	})

	// 40169 Mental Transferal: mirror Stryfe's damage onto the attached
	// character once, then discard.
	engine.RegisterBehavior("40169", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			// Hope Summers if Stryfe's Grasp is out, else the revealer.
			for _, p := range g.Players {
				for _, id := range p.Allies {
					if a := g.Allies[id]; a != nil && engine.BaseCodeOf(a.Code) == "40130" && g.SideSchemeInPlay("40168a") {
						t.Target = id
						return nil
					}
				}
			}
			if p := g.Player(cardutil.FirstPlayerID(g)); p != nil {
				t.Target = p.ID
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.DamageEntity)
			if !ok || m.Damage <= 0 {
				return nil
			}
			if v := stryfeVillain(g); v == nil || m.Target != v.ID {
				return nil
			}
			t := g.Attachments[e.EID()]
			if t == nil || t.Target == "" {
				return nil
			}
			g.Delete(t.ID)
			g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: t.Code})
			g.TLogf("c.mentalTransferalMirrorsDamage", m.Damage)
			return []engine.Message{engine.DamageEntity{Target: t.Target, Damage: m.Damage, Source: t.ID}}
		},
	})

	// 40170 Mind Alteration: 1 damage after playing events/upgrades.
	engine.RegisterBehavior("40170", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if p := g.Player(cardutil.FirstPlayerID(g)); p != nil {
				t.Target = p.ID
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			t := g.Attachments[e.EID()]
			if t == nil || t.Target == "" {
				return nil
			}
			switch m := msg.(type) {
			case engine.EventPlayed:
				if engine.PlayerID(m.Player) == engine.PlayerID(t.Target) {
					return []engine.Message{engine.DamageEntity{Target: t.Target, Damage: 1, Source: t.ID}}
				}
			case engine.UpgradeEnterPlay:
				if m.Player == t.Target {
					return []engine.Message{engine.DamageEntity{Target: t.Target, Damage: 1, Source: t.ID}}
				}
			}
			return nil
		},
	})

	// 40171 Mind Trap: characters enter play exhausted (approximation via
	// spawn hooks on the attachment's owner).
	engine.RegisterBehavior("40171", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if p := g.Player(cardutil.FirstPlayerID(g)); p != nil {
				t.Target = p.ID
			}
			return nil
		},
	})

	// 40172 Psionic Amnesia: +2 costs (engine costFor).
	engine.RegisterBehavior("40172", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if p := g.Player(cardutil.FirstPlayerID(g)); p != nil {
				t.Target = p.ID
			}
			return nil
		},
	})

	// 40173 Psychic Inertia: the hero action is approximated away.
	engine.RegisterBehavior("40173", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if p := g.Player(cardutil.FirstPlayerID(g)); p != nil {
				t.Target = p.ID
			}
			return nil
		},
	})

	// 40174 Zero: shuffles back into the encounter deck unless some player
	// holds 3+ cards of one type (approximation: he survives at 1 HP).
	engine.RegisterBehavior("40174", &engine.Behavior{
		MinionDamageable: func(g *engine.Game, m *engine.Minion, damage int) bool {
			if m.HP()-damage > 0 {
				return true
			}
			if handTypeMax(g) >= 3 {
				return true
			}
			g.TLogf("c.zeroShufflesBackIntoTheEncounterDeckInsteadOfBeingDefeated")
			m.Damage = m.MaxHP - 1
			return false
		},
	})

	// 40175 Cerebral Erasure: bounce an upgrade/support on reveal and on
	// defeat (auto-picks the first).
	engine.RegisterBehavior("40175", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(cardutil.FirstPlayerID(g))
			if p == nil {
				return nil
			}
			for _, id := range p.Supports {
				if s := g.Supports[id]; s != nil {
					return bounceToHand(g, p, "support", id)
				}
			}
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil {
					return bounceToHand(g, p, "upgrade", id)
				}
			}
			return nil
		},
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			p := g.Player(cardutil.FirstPlayerID(g))
			if p == nil {
				return nil
			}
			for _, id := range p.Supports {
				if s2 := g.Supports[id]; s2 != nil {
					return bounceToHand(g, p, "support", id)
				}
			}
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil {
					return bounceToHand(g, p, "upgrade", id)
				}
			}
			return nil
		},
	})

	// 40176 Telepathic Camouflage: each player places threat equal to
	// their most common hand type.
	engine.RegisterBehavior("40176", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			s := g.SideSchemes[e.EID()]
			if s == nil {
				return nil
			}
			for _, p := range g.Players {
				if n := mostCommonHandType(p); n > 0 {
					s.Threat += n
				}
			}
			g.TLogf("c.startsWithThreat", s, s.Threat)
			return nil
		},
	})

	// 40177 Psionic Surge: mill the encounter deck X; PSIONIC discards
	// become facedown encounter cards.
	engine.RegisterBehavior("40177", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			x := mostCommonHandType(p)
			for i := 0; i < x; i++ {
				c, ok := g.DrawEncounter()
				if !ok {
					break
				}
				if c.Def().HasTrait("Psionic") {
					p.EncounterDown = append(p.EncounterDown, c)
					g.TLogf("c.psionicSurgeIsDealtToFacedown", c, p.Name)
				} else {
					g.EncounterDiscard = append(g.EncounterDiscard, c)
				}
			}
			return nil
		},
	})

	// 40178 Psychic Override: discard non-matching hand cards, redraw,
	// place threat per matching card (payloads are built per type).
	engine.RegisterBehavior("40178", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var choices []engine.Choice
			for _, typ := range []string{"ally", "event", "support", "upgrade", "resource"} {
				var discarded engine.CardList
				kept := 0
				for _, c := range p.Hand {
					if c.Def().Type == typ {
						kept++
					} else {
						discarded = append(discarded, c)
					}
				}
				msgs := []engine.Message{}
				if len(discarded) > 0 {
					msgs = append(msgs, engine.DiscardCards{Player: p.ID, Cards: discarded})
				}
				if draw := p.HandSize(g) - kept; draw > 0 {
					msgs = append(msgs, engine.DrawCards{Player: p.ID, N: draw})
				}
				if kept > 0 && g.MainScheme != nil {
					msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: kept, Source: t.ID})
				}
				choices = append(choices, engine.Choice{
					ID: "type-" + typ, Label: engine.Tf("c.discardThreat", typ, len(discarded), kept), Kind: engine.ChoiceLabel,
				}.Msgs(msgs...))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.psychicOverrideChooseACardType"), choices...),
			}}
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if p := g.Player(cardutil.FirstPlayerID(g)); p != nil && len(p.Hand) > 0 {
				c := p.Hand[0]
				return []engine.Message{
					engine.DiscardCards{Player: p.ID, Cards: engine.CardList{c}},
					engine.DrawCards{Player: p.ID, N: 1},
				}
			}
			return nil
		},
	})

	// 40179 Telekinetic Wave: bounce + Stryfe activates.
	engine.RegisterBehavior("40179", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var msgs []engine.Message
			for _, id := range p.Supports {
				if s := g.Supports[id]; s != nil {
					msgs = bounceToHand(g, p, "support", id)
					break
				}
			}
			if len(msgs) == 0 {
				for _, id := range p.Upgrades {
					if u := g.Upgrades[id]; u != nil {
						msgs = bounceToHand(g, p, "upgrade", id)
						break
					}
				}
			}
			if v := stryfeVillain(g); v != nil {
				msgs = append(msgs, engine.VillainActivates{VillainID: v.ID, Player: p.ID})
			}
			return msgs
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if p := g.Player(cardutil.FirstPlayerID(g)); p != nil && mostCommonHandType(p) >= 3 && g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 3, Source: engine.EntityID("")}}
			}
			return nil
		},
	})

	registerExtremeMeasures()
	registerMutantInsurrection()
}

// bounceToHand returns a controlled card to its owner's hand.
func bounceToHand(g *engine.Game, p *engine.Player, kind string, id engine.EntityID) []engine.Message {
	var code string
	switch kind {
	case "support":
		if s := g.Supports[id]; s != nil {
			code = s.Code
		}
	case "upgrade":
		if u := g.Upgrades[id]; u != nil {
			code = u.Code
		}
	}
	if code == "" {
		return nil
	}
	g.Delete(id)
	p.Hand = append(p.Hand, engine.Card{ID: g.NextCardID(), Code: code, Owner: p.ID})
	g.TLogf("c.returnsToSHand", engine.DB.MustLookup(code).Name, p.Name)
	return nil
}

func registerExtremeMeasures() {
	// 40180 Strobe: stun or chip everyone.
	engine.RegisterBehavior("40180", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return nil
			}
			p := g.Player(mn.EngagedWith)
			return []engine.Message{engine.AskQuestion{
				Player: p.ID,
				Question: engine.Ask(engine.Tf("c.strobeChoose"),
					engine.Choice{ID: "stun", Label: engine.Tf("c.stunEachCharacterYouControl"), Kind: engine.ChoiceLabel}.
						Msgs(stunAllOwn(p)...),
					engine.Choice{ID: "dmg", Label: engine.Tf("c.deal1DamageToEachCharacterYouControl"), Kind: engine.ChoiceLabel}.
						Msgs(chipAllOwn(p)...),
				),
			}}
		},
	})

	// 40181 Tempo: after activating, mill twice your hand size (the +1
	// hand size aura lives in the engine).
	engine.RegisterBehavior("40181", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionActivates)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil {
				return nil
			}
			return []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: 2 * len(p.Hand)}}
		},
	})

	// 40182 Thumbelina: -1 damage per hit unless Tiny (engine); bounce the
	// best upgrade on reveal.
	engine.RegisterBehavior("40182", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return nil
			}
			p := g.Player(mn.EngagedWith)
			if p == nil {
				return nil
			}
			best, pick := -1, engine.EntityID("")
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil && cardutil.Cost(u.EDef()) > best {
					best, pick = cardutil.Cost(u.EDef()), id
				}
			}
			if pick != "" {
				return bounceToHand(g, p, "upgrade", pick)
			}
			return nil
		},
	})

	// 40183 Wildside: attack or lose your best support.
	engine.RegisterBehavior("40183", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return nil
			}
			p := g.Player(mn.EngagedWith)
			if p == nil {
				return nil
			}
			atkMsgs := []engine.Message{engine.AskAttack{Enemy: mn.ID, Player: p.ID}}
			var bounce []engine.Message
			best, pick := -1, engine.EntityID("")
			for _, id := range p.Supports {
				if s := g.Supports[id]; s != nil && cardutil.Cost(s.EDef()) > best {
					best, pick = cardutil.Cost(s.EDef()), id
				}
			}
			if pick != "" {
				bounce = bounceToHand(g, p, "support", pick)
			}
			return []engine.Message{engine.AskQuestion{
				Player: p.ID,
				Question: engine.Ask(engine.Tf("c.wildsideChoose"),
					engine.Choice{ID: "attack", Label: engine.Tf("c.wildsideAttacksYou"), Kind: engine.ChoiceLabel}.Msgs(atkMsgs...),
					engine.Choice{ID: "bounce", Label: engine.Tf("c.returnYourHighestCostSupportToHand"), Kind: engine.ChoiceLabel}.Msgs(bounce...),
				),
			}}
		},
	})

	// 40184 Extreme Measures: Hinder 2 (parsed); player cards entering
	// play cost indirect damage.
	engine.RegisterBehavior("40184", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			s := g.SideSchemes[e.EID()]
			if s == nil {
				return nil
			}
			var owner engine.PlayerID
			var cost int
			switch m := msg.(type) {
			case engine.AllyEnteredPlay:
				if a := g.Allies[m.Ally]; a != nil {
					owner, cost = a.Owner, cardutil.Cost(a.EDef())
				}
			case engine.UpgradeEnterPlay:
				owner, cost = m.Player, cardutil.Cost(m.Card.Def())
			}
			if owner == "" || cost <= 0 {
				return nil
			}
			g.TLogf("c.extremeMeasuresTakesIndirectDamage", g.Player(owner).Name, cost)
			return []engine.Message{engine.IndirectDamage{Player: owner, N: cost}}
		},
	})
}

func registerMutantInsurrection() {
	// 40185 Dragoness: +X SCH/ATK vs the engaged player's energy icons
	// (approximation: max across players).
	engine.RegisterBehavior("40185", &engine.Behavior{
		EnemyStatBonus: func(g *engine.Game, e engine.Entity) (atk, sch int) {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return 0, 0
			}
			n := handIcons(g.Player(mn.EngagedWith), "energy")
			return n, n
		},
	})

	// 40186 Forearm: after attacking, mill physical icons in hand.
	engine.RegisterBehavior("40186", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil {
				return nil
			}
			return []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: handIcons(p, "physical")}}
		},
	})

	// 40187 Reaper: stun/exhaust based on mental icons.
	engine.RegisterBehavior("40187", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AskAttack)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil {
				return nil
			}
			n := handIcons(p, "mental")
			if n >= 4 {
				return []engine.Message{engine.ExhaustEntity{ID: p.ID}}
			}
			if n >= 2 {
				return []engine.Message{engine.StunEntity{Target: p.ID}}
			}
			return nil
		},
	})

	// 40188 Samurai: charge counters; pay damage or discard.
	engine.RegisterBehavior("40188", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			mn := g.Minions[e.EID()]
			p := g.Player(m.Player)
			if mn == nil || p == nil {
				return nil
			}
			mn.Counters++
			n := mn.Counters
			g.TLogf("c.samuraiGainsAChargeCounter", n)
			return []engine.Message{engine.AskQuestion{
				Player: p.ID,
				Question: engine.Ask(engine.Tf("c.samuraiChargeCounters", n),
					engine.Choice{ID: "dmg", Label: engine.Tf("c.takeDamage", n), Kind: engine.ChoiceLabel}.
						Msgs(engine.DamageEntity{Target: p.ID, Damage: n, Source: mn.ID}),
					engine.Choice{ID: "disc", Label: engine.Tf("c.discardACardYouControlWithCost", n), Kind: engine.ChoiceLabel},
				),
			}}
		},
	})

	// 40189 Mutant Insurrection: Assault/toughness riders not modeled;
	// threat scales with MLF characters.
	engine.RegisterBehavior("40189", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			s := g.SideSchemes[e.EID()]
			if s == nil {
				return nil
			}
			n := 0
			for _, mn := range g.Minions {
				if mn != nil && mn.EDef().HasTrait("Mutant Liberation Front") {
					n++
				}
			}
			if v := stryfeVillain(g); v != nil {
				n++
			}
			if n > 0 {
				s.Threat += 2 * n
				g.TLogf("c.gainsExtraThreat", s, 2*n)
				return nil
			}
			// No additional threat: surge.
			if c, ok := g.DrawEncounter(); ok {
				return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
			}
			return nil
		},
	})
}

func stunAllOwn(p *engine.Player) []engine.Message {
	msgs := []engine.Message{engine.StunEntity{Target: p.ID}}
	for _, id := range p.Allies {
		msgs = append(msgs, engine.StunEntity{Target: id})
	}
	return msgs
}

func chipAllOwn(p *engine.Player) []engine.Message {
	msgs := []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 1, Source: engine.EntityID("")}}
	for _, id := range p.Allies {
		msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: engine.EntityID("")})
	}
	return msgs
}
