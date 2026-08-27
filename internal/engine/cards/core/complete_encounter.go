package core

// complete_encounter.go implements the remaining Core Set encounter cards
// (minions, side schemes, treacheries, attachments and main-scheme stage
// reveals). Approximations are noted inline.

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func registerRemainingEncounterCards() {
	// ---- Rhino ----

	// Rhino's Armored Suit: absorbed damage is stored on the attachment
	// (up to 5) — wired through the Rhino villain behavior's damage gate.
	engine.RegisterBehavior("01098", &engine.Behavior{})

	// Charge: discard after Rhino attacks. Overkill is not implemented;
	// the discard rider is.
	engine.RegisterBehavior("01099", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok {
				return nil
			}
			a := g.Attachments[e.EID()]
			if a == nil || w.Enemy != a.Target {
				return nil
			}
			g.Delete(a.ID)
			g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: a.Code})
			g.TLogf("c.chargeIsDiscardedAfterTheAttack")
			return nil
		},
	})

	// Enhanced Ivory Horn: Hero Action — spend 3 physical to discard.
	engine.RegisterBehavior("01100", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.spendPhysicalPhysicalPhysicalDiscardEnhancedIvoryHorn"), Type: engine.AbilityAction,
				Cost: 3, CostIcons: "physical:3", HeroOnly: true,
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
		},
	})

	// Keyword-only minions (engine keywords cover them). Weapons Runner's
	// Surge and "Boost: put into play" are both engine-covered too.
	for _, code := range []string{"01101", "01102", "01120", "01121", "01167", "01172", "01184", "01076"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	// Shocker: deals 1 damage to each hero on reveal.
	engine.RegisterBehavior("01103", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: 1, Source: e.EID()})
			}
			return msgs
		},
	})

	// Hydra Soldier: when defeated, deal the engaged player an encounter card.
	engine.RegisterBehavior("01182", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if d, ok := msg.(engine.MinionDefeated); ok && d.MinionID == e.EID() {
				if mn := g.Minions[e.EID()]; mn != nil && mn.EngagedWith != "" {
					return []engine.Message{engine.DealEncounterToPlayer{Player: mn.EngagedWith}}
				}
			}
			return nil
		},
	})

	// ---- Side schemes ----

	// "When Revealed: place an additional 1 [per hero] threat here."
	for _, code := range []string{"01107", "01125", "01126", "01161", "01171", "01176"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
				return []engine.Message{engine.SchemeThreat{Scheme: e.EID(), N: len(g.Players), Source: e.EID()}}
			},
		})
	}

	// Crowd Control: crisis keyword handled at spawn.
	engine.RegisterBehavior("01108", &engine.Behavior{})

	// The "Immortal" Klaw: Klaw gets +10 HP while this scheme is in play.
	engine.RegisterBehavior("01127", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for _, v := range g.Villains {
				v.MaxHP += 10
				g.TLogf("c.gets10HitPointsTheImmortalKlaw", v)
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if d, ok := msg.(engine.SchemeDefeated); ok && d.Scheme == e.EID() {
				for _, v := range g.Villains {
					v.MaxHP -= 10
					if v.Damage > v.MaxHP {
						v.Damage = v.MaxHP
					}
				}
			}
			return nil
		},
	})

	// The Masters of Evil: reveal cards until a Masters of Evil minion
	// enters play engaged with the first player.
	engine.RegisterBehavior("01128", &engine.Behavior{
		OnPlay: discardUntilMinion("Masters of Evil"),
	})

	// Legions of Hydra: fetch Madame Hydra + 2 threat per Hydra enemy.
	engine.RegisterBehavior("01180", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var msgs []engine.Message
			if !minionInPlay(g, "01181") {
				msgs = append(msgs, findAndSpawnMinion(g, "01181")...)
			}
			hydra := 0
			for _, mn := range g.Minions {
				if mn.EDef().HasTrait("hydra") {
					hydra++
				}
			}
			for _, v := range g.Villains {
				if v.EDef().HasTrait("hydra") {
					hydra++
				}
			}
			msgs = append(msgs, engine.SchemeThreat{Scheme: e.EID(), N: 2 * hydra, Source: e.EID()})
			return msgs
		},
	})

	// The Doomsday Chair: fetch M.O.D.O.K.
	engine.RegisterBehavior("01183", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if minionInPlay(g, "01184") {
				return nil
			}
			return findAndSpawnMinion(g, "01184")
		},
	})

	// Madame Hydra: immune while Legions of Hydra is in play; adds threat
	// to it after activating.
	engine.RegisterBehavior("01181", &engine.Behavior{
		MinionDamageable: func(g *engine.Game, m *engine.Minion, n int) bool {
			for _, s := range g.SideSchemes {
				if s.Code == "01180" {
					g.TLogf("c.cannotTakeDamageWhileLegionsOfHydraIsInPlay", m)
					return false
				}
			}
			return true
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			var acted bool
			switch m := msg.(type) {
			case engine.DamageEntity:
				acted = m.Source == e.EID() && m.Target.Is(engine.KindPlayer)
			case engine.SchemeThreat:
				acted = m.Source == e.EID()
			}
			if !acted {
				return nil
			}
			for _, s := range g.SideSchemes {
				if s.Code == "01180" {
					return []engine.Message{engine.SchemeThreat{Scheme: s.ID, N: 2, Source: e.EID()}}
				}
			}
			return nil
		},
	})

	// Highway Robbery: each player tucks a random card here; returned when
	// defeated.
	engine.RegisterBehavior("01166", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			s := g.SideSchemes[e.EID()]
			if s == nil {
				return nil
			}
			for _, p := range g.Players {
				if len(p.Hand) == 0 {
					continue
				}
				i := g.Random(len(p.Hand))
				c := p.Hand[i]
				p.Hand = append(p.Hand[:i], p.Hand[i+1:]...)
				s.StoredCards = append(s.StoredCards, c)
				g.TLogf("c.tucksARandomCardUnderHighwayRobbery", p.Name)
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if d, ok := msg.(engine.SchemeDefeated); ok && d.Scheme == e.EID() {
				s := g.SideSchemes[e.EID()]
				if s == nil {
					return nil
				}
				for _, c := range s.StoredCards {
					if q := g.Player(c.Owner); q != nil {
						q.Hand = append(q.Hand, c)
					}
				}
				s.StoredCards = nil
				g.TLogf("c.highwayRobberyReturnsItsStoredCards")
			}
			return nil
		},
	})

	// Drone Factory / Invasive AI / Ultron's Imperative / Under Attack.
	engine.RegisterBehavior("01148", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, engine.SpawnDrone{Player: p.ID})
			}
			msgs = append(msgs, engine.SchemeThreat{Scheme: e.EID(), N: droneCount(g), Source: e.EID()})
			return msgs
		},
	})
	engine.RegisterBehavior("01149", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, engine.MillPlayerDeck{Player: p.ID, N: 3})
			}
			return msgs
		},
	})
	engine.RegisterBehavior("01150", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			fp := cardutil.FirstPlayerID(g)
			return []engine.Message{engine.SpawnDrone{Player: fp}, engine.SpawnDrone{Player: fp}}
		},
	})
	engine.RegisterBehavior("01151", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				p := p
				msgs = append(msgs, engine.AskQuestion{Player: p.ID, Question: engine.Ask(
					engine.Tf("c.underAttackPlace2ThreatHereOrTake3Damage"),
					engine.Choice{Label: engine.Tf("c.place2ThreatHere"), Kind: engine.ChoiceLabel}.
						Msgs(engine.SchemeThreat{Scheme: e.EID(), N: 2, Source: p.ID}),
					engine.Choice{Label: engine.Tf("c.take3Damage"), Kind: engine.ChoiceLabel}.
						Msgs(engine.DamageEntity{Target: p.ID, Damage: 3, Source: e.EID()}),
				)})
			}
			return msgs
		},
	})

	// ---- Klaw ----

	// Sonic Converter: stun damaged characters; removable for 3 mixed.
	engine.RegisterBehavior("01118", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.DamageEntity)
			a := g.Attachments[e.EID()]
			if !ok || a == nil || d.Source != a.Target || d.Target == a.Target {
				return nil
			}
			switch d.Target.Kind() {
			case engine.KindPlayer, engine.KindAlly:
				return []engine.Message{engine.StunEntity{Target: d.Target}}
			}
			return nil
		},
		Abilities: removalAbility("Spend [energy] [mental] [physical] → discard Sonic Converter", "energy:1 mental:1 physical:1", 3),
	})

	// Solid-Sound Body: Klaw gains retaliate 1 while attached.
	engine.RegisterBehavior("01119", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowDefended)
			a := g.Attachments[e.EID()]
			if !ok || a == nil || w.Against != a.Target {
				return nil
			}
			return []engine.Message{engine.DamageEntity{Target: w.Defender, Damage: 1, Source: a.Target}}
		},
		Abilities: removalAbility("Spend [energy] [mental] [physical] → discard Solid-Sound Body", "energy:1 mental:1 physical:1", 3),
	})

	// Klaw's Vengeance.
	engine.RegisterBehavior("01122", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if p.IsHero() {
				for _, id := range cardutil.SortedEnemyIDs(g) {
					if g.Villains[id] != nil {
						return []engine.Message{
							engine.DealBoost{Enemy: id}, engine.RevealBoost{Enemy: id},
							engine.AskAttack{Enemy: id, Player: p.ID, Trigger: engine.TriggerVillainAttacksYou},
						}
					}
				}
				return nil
			}
			discardRandom(g, p, 1)
			return nil
		},
	})

	// Sonic Boom: pay 3 mixed or exhaust everything you control.
	engine.RegisterBehavior("01123", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			pay := g.CustomPaymentQuestion(p, 3, engine.S("Sonic Boom: pay [energy] [mental] [physical] or exhaust all your characters"),
				map[string]any{"player": p.ID.String(), "abilityIcons": "energy:1 mental:1 physical:1"})
			exhaustAll := func() []engine.Message {
				var msgs []engine.Message
				msgs = append(msgs, engine.ExhaustEntity{ID: p.ID})
				for _, id := range p.Allies {
					msgs = append(msgs, engine.ExhaustEntity{ID: id})
				}
				return msgs
			}
			q := engine.Ask(
				engine.Tf("c.sonicBoomPay3Resources1OfEachTypeOrExhaustYourCharacters"),
				engine.Choice{Label: engine.Tf("c.pay3Resources"), Kind: engine.ChoiceLabel}.WithThen(pay),
				engine.Choice{ID: "exhaust", Label: engine.Tf("c.exhaustEachCharacterYouControl"), Kind: engine.ChoiceLabel}.
					Msgs(exhaustAll()...),
			)
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: q}}
		},
	})

	// Sound Manipulation.
	engine.RegisterBehavior("01124", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			healVillain := func(n int) {
				for _, v := range g.Villains {
					if v.Damage > 0 {
						d := min(n, v.Damage)
						v.Damage -= d
						g.TLogf("log.heals", v, d)
					}
				}
			}
			if p.IsHero() {
				healVillain(2)
				return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 2, Source: t.ID}}
			}
			healed := false
			for _, v := range g.Villains {
				if v.Damage > 0 {
					healed = true
				}
			}
			healVillain(4)
			if !healed {
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			return nil
		},
	})

	// ---- Masters of Evil minions & treachery ----

	// Radioactive Man: discard 1 random after attacking the revealer.
	engine.RegisterBehavior("01129", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.DamageEntity)
			if !ok || d.Source != e.EID() || !d.Target.Is(engine.KindPlayer) {
				return nil
			}
			if p := g.Player(d.Target); p != nil {
				discardRandom(g, p, 1)
			}
			return nil
		},
	})

	// Whirlwind: also attacks each other hero.
	engine.RegisterBehavior("01130", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionActivates)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			var msgs []engine.Message
			for _, p := range g.Players {
				if p.ID != m.Player && !p.KOed {
					msgs = append(msgs, engine.AskAttack{Enemy: e.EID(), Player: p.ID})
				}
			}
			return msgs
		},
	})

	// Tiger Shark: gains tough after attacking (approximation of the
	// after-attack forced response).
	engine.RegisterBehavior("01131", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.DamageEntity)
			if !ok || d.Source != e.EID() {
				return nil
			}
			return []engine.Message{engine.ToughEntity{Target: e.EID()}}
		},
	})

	// Masters of Mayhem: each MoE minion attacks its engaged hero.
	engine.RegisterBehavior("01133", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var msgs []engine.Message
			for _, id := range cardutil.SortedEnemyIDs(g) {
				mn := g.Minions[id]
				if mn == nil || !mn.EDef().HasTrait("masters_of_evil") || mn.EngagedWith == "" {
					continue
				}
				msgs = append(msgs, engine.AskAttack{Enemy: mn.ID, Player: mn.EngagedWith})
			}
			if len(msgs) == 0 {
				msgs = discardUntilMinion("Masters of Evil")(g, t)
			}
			return msgs
		},
	})

	// ---- Ultron ----

	// Main scheme stage reveals: drones for everyone.
	engine.RegisterBehavior("01138", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, engine.SpawnDrone{Player: p.ID})
			}
			return msgs
		},
	})
	engine.RegisterBehavior("01139", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, engine.SpawnDrone{Player: p.ID})
			}
			return msgs
		},
	})

	// Ultron main scheme stage 1B "When Revealed": each player's top deck
	// card enters play facedown as a Drone minion.
	engine.RegisterBehavior("01137b", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, engine.SpawnDrone{Player: p.ID})
			}
			return msgs
		},
	})

	// Ultron Drones environment: drone stats already covered by spawn;
	// defeated drones route to their owner's discard (MinionDefeated).
	engine.RegisterBehavior("01140", &engine.Behavior{})

	// Program Transmitter: +1 threat to each side scheme after Ultron
	// schemes.
	engine.RegisterBehavior("01141", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			sch, ok := msg.(engine.ApplyVillainScheme)
			a := g.Attachments[e.EID()]
			if !ok || a == nil || sch.VillainID != a.Target {
				return nil
			}
			var msgs []engine.Message
			for _, id := range cardutil.SortedIDs(g.SideSchemes) {
				msgs = append(msgs, engine.SchemeThreat{Scheme: id, N: 1, Source: a.Target})
			}
			return msgs
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustYourHeroSpendMentalMentalDiscardProgramTransmitter"),
				Type:  engine.AbilityAction, Cost: 2, CostIcons: "mental:2", HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					a := g.Attachments[self]
					if a == nil {
						return nil
					}
					g.Delete(self)
					g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: a.Code})
					return []engine.Message{engine.ExhaustEntity{ID: actingPlayer(g)}}
				},
			}}
		},
	})

	// Upgraded Drones: attaches to the Ultron Drones environment (bonus
	// handled by the engine's droneBonus).
	engine.RegisterBehavior("01142", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, id := range cardutil.SortedIDs(g.Environments) {
				if env := g.Environments[id]; env != nil {
					t.Target = id
					g.TLogf("c.attachesToEnvironment", t)
					return nil
				}
			}
			// Environment not in play yet: fall back to the villain so the
			// card still has a visible host.
			for id := range g.Villains {
				t.Target = id
				break
			}
			return nil
		},
		Abilities: removalAbility("Spend [energy] [mental] [physical] → discard Upgraded Drones", "energy:1 mental:1 physical:1", 3),
	})

	// Advanced Ultron Drone: spawns a drone on defeat.
	engine.RegisterBehavior("01143", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if d, ok := msg.(engine.MinionDefeated); ok && d.MinionID == e.EID() {
				if mn := g.Minions[e.EID()]; mn != nil && mn.EngagedWith != "" {
					return []engine.Message{engine.SpawnDrone{Player: mn.EngagedWith}}
				}
			}
			return nil
		},
	})

	// Android Efficiency: a drone for each player.
	engine.RegisterBehavior("01144", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var msgs []engine.Message
			for _, q := range g.Players {
				msgs = append(msgs, engine.SpawnDrone{Player: q.ID})
			}
			return msgs
		},
	})

	// Rage of Ultron.
	engine.RegisterBehavior("01145", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if p.IsHero() {
				for _, id := range cardutil.SortedEnemyIDs(g) {
					if g.Villains[id] != nil {
						return []engine.Message{
							engine.DealBoost{Enemy: id}, engine.RevealBoost{Enemy: id},
							engine.AskAttack{Enemy: id, Player: p.ID, Trigger: engine.TriggerVillainAttacksYou},
							engine.MillPlayerDeck{Player: p.ID, N: 2},
						}
					}
				}
				return nil
			}
			for _, v := range g.Villains {
				threat := v.SchemeVal
				return []engine.Message{
					engine.SchemeThreat{Scheme: mainSchemeID(g), N: threat, Source: v.ID},
					engine.MillPlayerDeck{Player: p.ID, N: max(1, threat)},
				}
			}
			return nil
		},
	})

	// Repair Sequence: Ultron heals 2 per engaged drone (surge if none).
	engine.RegisterBehavior("01146", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			n := engagedDrones(g, p.ID)
			healVillains(g, 2*n)
			if n == 0 {
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			return nil
		},
	})

	// Swarm Attack: each of your drones attacks.
	engine.RegisterBehavior("01147", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var msgs []engine.Message
			for _, mn := range g.Minions {
				if mn.IsDrone && mn.EngagedWith == p.ID {
					msgs = append(msgs, engine.AskAttack{Enemy: mn.ID, Player: p.ID})
				}
			}
			if len(msgs) == 0 {
				msgs = append(msgs, engine.SpawnDrone{Player: p.ID})
			}
			return msgs
		},
	})

	// Vibranium Armor: villain gains tough after taking damage.
	engine.RegisterBehavior("01152", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.DamageEntity)
			a := g.Attachments[e.EID()]
			if !ok || a == nil || d.Target != a.Target {
				return nil
			}
			return []engine.Message{engine.ToughEntity{Target: a.Target}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustYourHeroSpendPhysicalPhysicalDiscardVibraniumArmor"),
				Type:  engine.AbilityAction, Cost: 2, CostIcons: "physical:2", HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					discardAttachment(g, self)
					return []engine.Message{engine.ExhaustEntity{ID: actingPlayer(g)}}
				},
			}}
		},
	})

	// Concussion Blasters: villain gains retaliate 1 while attached.
	engine.RegisterBehavior("01153", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowDefended)
			a := g.Attachments[e.EID()]
			if !ok || a == nil || w.Against != a.Target {
				return nil
			}
			return []engine.Message{engine.DamageEntity{Target: w.Defender, Damage: 1, Source: a.Target}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustYourHeroSpendEnergyEnergyDiscardConcussionBlasters"),
				Type:  engine.AbilityAction, Cost: 2, CostIcons: "energy:2", HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					discardAttachment(g, self)
					return []engine.Message{engine.ExhaustEntity{ID: actingPlayer(g)}}
				},
			}}
		},
	})

	// Concussive Blast: 1 damage to each friendly character.
	engine.RegisterBehavior("01154", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var msgs []engine.Message
			msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: 1, Source: t.ID})
			for _, id := range p.Allies {
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: t.ID})
			}
			return msgs
		},
	})

	// ---- Titania (She-Hulk nemesis) ----

	// Titania: THW/ATK scale with remaining HP.
	engine.RegisterBehavior("01162", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil {
				return nil
			}
			switch msg.(type) {
			case engine.DamageEntity, engine.HealEntity:
				if mn.HP() > 0 {
					mn.AttackVal = mn.HP()
					mn.SchemeVal = mn.HP()
				}
			}
			return nil
		},
	})

	// Genetically Enhanced: attach to the highest-HP minion (+3 HP).
	engine.RegisterBehavior("01163", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			var best *engine.Minion
			for _, id := range cardutil.SortedEnemyIDs(g) {
				mn := g.Minions[id]
				if mn == nil {
					continue
				}
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
			best.MaxHP += 3
			best.Attachments = append(best.Attachments, t.ID)
			g.TLogf("c.attachesTo3HitPoints", t, best)
			return nil
		},
	})

	// Titania's Fury: Titania attacks; if absent, fully heal her + surge.
	engine.RegisterBehavior("01164", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			for _, mn := range g.Minions {
				if mn.Code == "01162" {
					return []engine.Message{engine.AskAttack{Enemy: mn.ID, Player: p.ID}}
				}
			}
			// Not in play: heal all damage from her in the encounter
			// discard and surge.
			return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
		},
	})

	// ---- Vulture (Spider-Man nemesis) ----

	// Sweeping Swoop: stun the revealing hero (surge if Vulture in play).
	engine.RegisterBehavior("01168", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			msgs := []engine.Message{engine.StunEntity{Target: p.ID}}
			if !minionInPlay(g, "01167") {
				msgs = append(msgs, engine.RevealNextEncounter{Player: p.ID})
			}
			return msgs
		},
	})

	// The Vulture's Plans: random discard from each player; +1 threat per
	// distinct resource type.
	engine.RegisterBehavior("01169", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			types := map[string]bool{}
			for _, q := range g.Players {
				for i := 0; i < 1; i++ {
					_ = i
				}
				if len(q.Hand) == 0 {
					continue
				}
				i := g.Random(len(q.Hand))
				c := q.Hand[i]
				q.Hand = append(q.Hand[:i], q.Hand[i+1:]...)
				q.Discard = append(q.Discard, c)
				g.TLogf("c.discardsAtRandom", q.Name, c)
				for _, r := range c.Def().Resources {
					types[r] = true
				}
			}
			n := len(types)
			if n > 0 {
				return []engine.Message{engine.SchemeThreat{Scheme: mainSchemeID(g), N: n, Source: t.ID}}
			}
			return nil
		},
	})

	// ---- Yon-Rogg (Captain Marvel nemesis) ----

	// Yon-Rogg: after he activates, +1 threat on The Psyche-Magnitron.
	engine.RegisterBehavior("01177", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			var acted bool
			switch m := msg.(type) {
			case engine.DamageEntity:
				acted = m.Source == e.EID() && m.Target.Is(engine.KindPlayer)
			case engine.SchemeThreat:
				acted = m.Source == e.EID()
			}
			if !acted {
				return nil
			}
			for _, s := range g.SideSchemes {
				if s.Code == "01176" {
					return []engine.Message{engine.SchemeThreat{Scheme: s.ID, N: 1, Source: e.EID()}}
				}
			}
			return nil
		},
	})

	// Kree Manipulator: +1 threat; surge.
	engine.RegisterBehavior("01178", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			return []engine.Message{
				engine.SchemeThreat{Scheme: mainSchemeID(g), N: 1, Source: t.ID},
				engine.RevealNextEncounter{Player: p.ID},
			}
		},
	})

	// Yon-Rogg's Treason: discard each printed [energy] from hand.
	engine.RegisterBehavior("01179", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var dropped engine.CardList
			var kept engine.CardList
			for _, c := range p.Hand {
				energy := false
				for _, r := range c.Def().Resources {
					if r == "energy" {
						energy = true
					}
				}
				if energy {
					dropped = append(dropped, c)
				} else {
					kept = append(kept, c)
				}
			}
			p.Hand = kept
			p.Discard = append(p.Discard, dropped...)
			if len(dropped) == 0 {
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			return nil
		},
	})

	// ---- Whiplash (Iron Man nemesis) ----

	// Electric Whip Attack: 1 damage per upgrade you control or discard one.
	engine.RegisterBehavior("01173", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			n := len(p.Upgrades)
			var picks []engine.Choice
			picks = append(picks, engine.Choice{Label: engine.Tf("c.takeDamage1PerUpgrade", n), Kind: engine.ChoiceLabel}.
				Msgs(engine.DamageEntity{Target: p.ID, Damage: n, Source: t.ID}))
			for _, id := range p.Upgrades {
				picks = append(picks, engine.Choice{Label: engine.Tf("m.discardCard", g.Upgrades[id]), Kind: engine.ChoiceCard, CardCode: g.Upgrades[id].Code}.
					Msgs(engine.DiscardControlled{Player: p.ID, ID: id}))
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.electricWhipAttack"), picks...)}}
		},
	})

	// Electromagnetic Backlash: each player mills 5, 1 damage per energy.
	engine.RegisterBehavior("01174", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var msgs []engine.Message
			for _, q := range g.Players {
				energy := 0
				for i := 0; i < 5 && len(q.Deck) > 0; i++ {
					c := q.Deck[0]
					q.Deck = q.Deck[1:]
					for _, r := range c.Def().Resources {
						if r == "energy" {
							energy++
						}
					}
					q.Discard = append(q.Discard, c)
				}
				if energy > 0 {
					msgs = append(msgs, engine.DamageEntity{Target: q.ID, Damage: energy, Source: t.ID})
				}
			}
			return msgs
		},
	})

	// Heart-Shaped Herb: tough for the villain and engaged minions; surge.
	engine.RegisterBehavior("01158", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			msgs := []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			for _, id := range cardutil.SortedEnemyIDs(g) {
				if g.Villains[id] != nil {
					msgs = append(msgs, engine.ToughEntity{Target: id})
				}
			}
			for _, mn := range g.Minions {
				if mn.EngagedWith == p.ID {
					msgs = append(msgs, engine.ToughEntity{Target: mn.ID})
				}
			}
			return msgs
		},
	})

	// Ritual Combat: X = boost icons on the encounter deck's top + 1.
	engine.RegisterBehavior("01159", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			x := 1
			if len(g.EncounterDeck) > 0 {
				def := g.EncounterDeck[0].Def()
				if def.Boost != nil {
					x = 1 + *def.Boost
				}
				c := g.EncounterDeck[0]
				g.EncounterDeck = g.EncounterDeck[1:]
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(
				engine.Tf("c.ritualCombatXTakeXDamageOrPlaceXThreat", x),
				engine.Choice{Label: engine.Tf("c.takeDamage", x), Kind: engine.ChoiceLabel}.
					Msgs(engine.DamageEntity{Target: p.ID, Damage: x, Source: t.ID}),
				engine.Choice{Label: engine.Tf("c.placeThreat", x), Kind: engine.ChoiceLabel}.
					Msgs(engine.SchemeThreat{Scheme: mainSchemeID(g), N: x, Source: t.ID}),
			)}}
		},
	})

	// Biomechanical Upgrades: the save itself is engine-side (combat.go);
	// here only the attach targeting.
	engine.RegisterBehavior("01185", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			var best *engine.Minion
			for _, id := range cardutil.SortedEnemyIDs(g) {
				mn := g.Minions[id]
				if mn == nil {
					continue
				}
				taken := false
				for _, a := range mn.Attachments {
					if at := g.Attachments[a]; at != nil && at.Code == "01185" {
						taken = true
					}
				}
				if taken {
					continue
				}
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
			best.Attachments = append(best.Attachments, t.ID)
			g.TLogf("log.attachesTo", t, best)
			return nil
		},
	})

	// ---- Expert set ----

	// Exhaustion: exhaust your identity; surge.
	engine.RegisterBehavior("01191", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			return []engine.Message{
				engine.ExhaustEntity{ID: p.ID},
				engine.RevealNextEncounter{Player: p.ID},
			}
		},
	})

	// Masterplan: 4 threat per side scheme (or find one).
	engine.RegisterBehavior("01192", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if len(g.SideSchemes) > 0 {
				var msgs []engine.Message
				for _, id := range cardutil.SortedIDs(g.SideSchemes) {
					msgs = append(msgs, engine.SchemeThreat{Scheme: id, N: 4, Source: t.ID})
				}
				return msgs
			}
			for len(g.EncounterDeck) > 0 {
				c := g.EncounterDeck[0]
				g.EncounterDeck = g.EncounterDeck[1:]
				g.EncounterDiscard = append(g.EncounterDiscard, c)
				if c.Def().Type == "side_scheme" {
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
			}
			return nil
		},
	})

	// Under Fire: reveal the top card; surge.
	engine.RegisterBehavior("01193", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
		},
	})

	// Melter: the engaged player must defend with an ally if able.
	engine.RegisterBehavior("01132", &engine.Behavior{ForceAllyDefense: true})

	// Killmonger: cannot take damage from Black Panther upgrades (the
	// Wakanda Forever! specials stamp the upgrade as the damage source).
	engine.RegisterBehavior("01157", &engine.Behavior{
		MinionDamageableSrc: func(g *engine.Game, m *engine.Minion, n int, src engine.EntityID) bool {
			if u := g.Upgrades[src]; u != nil && u.EDef().HasTrait("black_panther") {
				g.TLogf("c.cannotTakeDamageFromBlackPantherUpgrades", m)
				return false
			}
			return true
		},
	})

	// ---- Main scheme 1A faces (contents/setup already covered by the
	// scenario flow) ----
	for _, code := range []string{"01097", "01116", "01137"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}
}

// ---- helpers ----

func discardRandom(g *engine.Game, p *engine.Player, n int) {
	for i := 0; i < n && len(p.Hand) > 0; i++ {
		j := g.Random(len(p.Hand))
		c := p.Hand[j]
		p.Hand = append(p.Hand[:j], p.Hand[j+1:]...)
		p.Discard = append(p.Discard, c)
		g.TLogf("c.discardsAtRandom", p.Name, c)
	}
}

// discardUntilMinion discards encounter cards until a minion with the
// trait is found and puts it into play engaged with the first player.
func discardUntilMinion(trait string) func(g *engine.Game, e engine.Entity) []engine.Message {
	return func(g *engine.Game, e engine.Entity) []engine.Message {
		for len(g.EncounterDeck) > 0 {
			c := g.EncounterDeck[0]
			g.EncounterDeck = g.EncounterDeck[1:]
			if c.Def().Type == "minion" && c.Def().HasTrait(trait) {
				return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
			}
			g.EncounterDiscard = append(g.EncounterDiscard, c)
		}
		return nil
	}
}

func findAndSpawnMinion(g *engine.Game, code string) []engine.Message {
	for i, c := range g.EncounterDeck {
		if c.Code == code {
			g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
			return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
		}
	}
	for i, c := range g.EncounterDiscard {
		if c.Code == code {
			g.EncounterDiscard = append(g.EncounterDiscard[:i], g.EncounterDiscard[i+1:]...)
			return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
		}
	}
	return nil
}

func minionInPlay(g *engine.Game, code string) bool {
	for _, mn := range g.Minions {
		if mn.Code == code {
			return true
		}
	}
	return false
}

func mainSchemeID(g *engine.Game) engine.EntityID {
	if g.MainScheme != nil {
		return g.MainScheme.ID
	}
	return ""
}

func droneCount(g *engine.Game) int {
	n := 0
	for _, mn := range g.Minions {
		if mn.IsDrone {
			n++
		}
	}
	return n
}

func engagedDrones(g *engine.Game, pid engine.PlayerID) int {
	n := 0
	for _, mn := range g.Minions {
		if mn.IsDrone && mn.EngagedWith == pid {
			n++
		}
	}
	return n
}

func healVillains(g *engine.Game, n int) {
	for _, v := range g.Villains {
		if v.Damage > 0 {
			d := min(n, v.Damage)
			v.Damage -= d
			g.TLogf("log.heals", v, d)
		}
	}
}

func discardAttachment(g *engine.Game, id engine.EntityID) {
	a := g.Attachments[id]
	if a == nil {
		return
	}
	g.Delete(id)
	g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: a.Code})
	g.TLogf("log.discarded", a)
}

// removalAbility builds a hero-action ability that discards the attachment
// it belongs to after paying the icon cost.
func removalAbility(label, icons string, cost int) func(g *engine.Game, e engine.Entity) []engine.Ability {
	return func(g *engine.Game, e engine.Entity) []engine.Ability {
		return []engine.Ability{{
			Label: engine.S(label), Type: engine.AbilityAction, Cost: cost, CostIcons: icons, HeroOnly: true,
			Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
				discardAttachment(g, self)
				return nil
			},
		}}
	}
}

// actingPlayer returns the player whose menu is being built (ActiveTurn),
// falling back to the first player.
func actingPlayer(g *engine.Game) engine.PlayerID {
	if g.ActiveTurn != "" {
		return g.ActiveTurn
	}
	return cardutil.FirstPlayerID(g)
}
