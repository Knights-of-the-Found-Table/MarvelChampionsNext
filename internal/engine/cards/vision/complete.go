// Package vision registers the Vision hero pack: the Dense/Intangible
// mass-form flip economy and the Ultron nemesis set.
package vision

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerVision()
	registerNemesis()
}

// massForm reports the player's current mass form: "dense", "intangible"
// or "". Tracked via ExtraTraits granted by the flip.
func massForm(p *engine.Player) string {
	for _, t := range p.ExtraTraits {
		if t == "dense" || t == "intangible" {
			return t
		}
	}
	return ""
}

// findMassUpgrade finds the Intangible upgrade (its flip toggles the
// form trait).
func findMassUpgrade(g *engine.Game, pid engine.PlayerID) *engine.Upgrade {
	p := g.Player(pid)
	if p == nil {
		return nil
	}
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil && u.Code[:5] == "26002" {
			return u
		}
	}
	return nil
}

// flipMass returns the message flipping the player's mass form.
func flipMass(pid engine.PlayerID) engine.Message {
	return engine.SetMassForm{Player: pid}
}

func registerVision() {
	// Vision identity: Density Manipulation — flip the mass form once
	// per round.
	engine.RegisterBehavior("26001", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			if g.UsedThisRound["vision-flip"] {
				return nil
			}
			if findMassUpgrade(g, e.EID()) == nil {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.densityManipulationFlipYourMassForm"), Type: engine.AbilityAction,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					g.UsedThisRound["vision-flip"] = true
					return []engine.Message{flipMass(self)}
				},
			}}
		},
	})

	// Intangible: the mass-form upgrade itself. Enters play granting
	// "intangible"; the flip message swaps the trait. Damage reduction
	// -2 while intangible.
	engine.RegisterBehavior("26002", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.SetMassForm{Player: e.EOwner(), Form: "intangible"}}
		},
		DamagePrevention: func(g *engine.Game, u *engine.Upgrade, p *engine.Player, n int) (int, int) {
			if massForm(p) == "intangible" {
				return min(2, n), 0
			}
			return 0, 0
		},
	})

	// Vivian: +2 THW intangible / +2 ATK dense (synced on flips).
	engine.RegisterBehavior("26003", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.SetMassForm); !ok {
				return nil
			}
			a := g.Allies[e.EID()]
			owner := g.Player(e.EOwner())
			if a == nil || owner == nil {
				return nil
			}
			switch massForm(owner) {
			case "intangible":
				a.PermTHW, a.PermATK = 2, 0
			case "dense":
				a.PermTHW, a.PermATK = 0, 2
			default:
				a.PermTHW, a.PermATK = 0, 0
			}
			return nil
		},
	})

	// 616 Hickory Branch Lane: Android ally search (deck + discard).
	engine.RegisterBehavior("26004", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.exhaust616HickoryBranchLaneSearchForAnAndroidAlly"), Type: engine.AbilityAction,
				Exhaust: true, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.EOwner())
					if p == nil {
						return nil
					}
					var picks []engine.Choice
					seen := map[string]bool{}
					for _, c := range append(append(engine.CardList{}, p.Deck...), p.Discard...) {
						def := c.Def()
						if def.Type == "ally" && def.HasTrait("android") && !seen[c.Code] {
							seen[c.Code] = true
							inDeck := func() bool { _, ok := p.Deck.Find(c.ID); return ok }()
							if inDeck {
								picks = append(picks, engine.Choice{Label: engine.S(def.Name + " (deck)"), Kind: engine.ChoiceCard, CardCode: def.Code}.
									Msgs(engine.TakeDeckCard{Player: p.ID, CardID: c.ID}, engine.ShufflePlayerDeck{Player: p.ID}))
							} else {
								picks = append(picks, engine.Choice{Label: engine.S(def.Name + " (discard)"), Kind: engine.ChoiceCard, CardCode: def.Code}.
									Msgs(engine.ReturnDiscardCard{Player: p.ID, CardID: c.ID}))
							}
						}
					}
					if len(picks) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID,
						Question: engine.Ask(engine.Tf("c.addWhichAndroidAllyToHand"), picks...)}}
				},
			}}
		},
	})

	// Solar Gem: Aerial + wild resource.
	engine.RegisterBehavior("26005", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.GrantTrait{Target: e.EOwner(), Trait: "aerial"}}
		},
		Resource: &engine.ResourceAbility{Icon: "wild"},
	})

	// Vision's Cape: dense retaliate 1 / intangible stalwart (stalwart
	// not modeled).
	engine.RegisterBehavior("26006", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			if massForm(p) == "dense" {
				return engine.StatBonus{Retaliate: 1}
			}
			return engine.StatBonus{}
		},
	})

	// Density Control: after a mass flip, discard → Vision event to hand.
	engine.RegisterBehavior("26007", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SetMassForm)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil || m.Player != u.Owner {
				return nil
			}
			p := g.Player(u.Owner)
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			seen := map[string]bool{}
			for _, c := range p.Discard {
				def := c.Def()
				if def.Type == "event" && def.Code[:2] == "26" && !seen[c.Code] {
					seen[c.Code] = true
					picks = append(picks, engine.Choice{Label: engine.S(def.Name), Kind: engine.ChoiceCard, CardCode: def.Code}.
						Msgs(engine.DiscardControlled{Player: p.ID, ID: u.ID},
							engine.ReturnDiscardCard{Player: p.ID, CardID: c.ID}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.densityControlReturnWhichVisionEvent"), picks...)}}
		},
	})

	// Solar Beam: dense 7 damage / intangible 5 threat.
	engine.RegisterBehavior("26008", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			switch massForm(p) {
			case "dense":
				return cardutil.ChooseEnemy(engine.Tf("c.solarBeamDenseDeal7DamageToWhichEnemy"),
					func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 7, nil })(g, e)
			case "intangible":
				return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(
					engine.Tf("c.solarBeamIntangibleRemove5ThreatFromWhichScheme"), schemePicks(g, 5, p.ID)...)}}
			}
			return nil
		},
	})

	// Superdense Strike: dense 5 damage.
	engine.RegisterBehavior("26009", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy(engine.Tf("c.superdenseStrikeDeal5DamageToWhichEnemy"),
			func(g *engine.Game, e engine.Entity) (int, []engine.Message) { return 5, nil }),
	})

	// Just Passing Through: intangible 3 threat (crisis ignored — crisis
	// blocks main-scheme thwarts, this event bypasses by direct
	// removal).
	engine.RegisterBehavior("26010", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(), Question: engine.Ask(
				engine.Tf("c.justPassingThroughRemove3ThreatFromWhichScheme"), schemePicks(g, 3, e.EOwner())...)}}
		},
	})

	// Phase Disruption: confuse + attachment removal.
	engine.RegisterBehavior("26011", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var picks []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				enemy := g.Entity(id)
				if enemy == nil {
					continue
				}
				var msgs []engine.Message
				msgs = append(msgs, engine.ConfuseEntity{Target: id})
				for aid, a := range g.Attachments {
					if a.Target == id {
						msgs = append(msgs, engine.DiscardAttachmentMsg{ID: aid})
					}
				}
				picks = append(picks, engine.Choice{Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget, SourceID: id, CardCode: enemy.ECode()}.
					Msgs(msgs...))
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(),
				Question: engine.Ask(engine.Tf("c.phaseDisruptionConfuseWhichEnemy"), picks...)}}
		},
	})

	// Mass Increase: dense full prevention + stun.
	engine.RegisterBehavior("26012", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, ec *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			return engine.Defends{Defender: p.ID, Against: against, PreventAll: true},
				[]engine.Message{engine.StunEntity{Target: against}}, true
		},
	})

	// Jocasta: attach a Defense event facedown (stored; replay approximated
	// to the storage only).
	engine.RegisterBehavior("26013", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			for _, c := range p.Discard {
				def := c.Def()
				if def.Type == "event" && def.HasTrait("defense") {
					picks = append(picks, engine.Choice{Label: engine.S("Attach " + def.Name), Kind: engine.ChoiceCard, CardCode: def.Code}.
						Msgs(engine.SupportStoreCard{ID: e.EID(), Card: c}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.jocastaAttachWhichDefenseEvent"), picks...)}}
		},
	})

	// Protector: mental-paid damage reduction — ally payment windows not
	// modeled; registered with the note.
	engine.RegisterBehavior("26014", &engine.Behavior{})

	// Victor Mancha: -1 damage from each attack.
	engine.RegisterBehavior("26015", &engine.Behavior{})

	// Flow Like Water: after a Defense card, 1 damage to the attacker —
	// EventPlayed reaction.
	engine.RegisterBehavior("26016", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			ep, ok := msg.(engine.EventPlayed)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil || ep.Player != u.Owner {
				return nil
			}
			def, okL := engine.DB.Lookup(ep.Card.Code)
			if !okL || def.Type != "event" || !def.HasTrait("defense") {
				return nil
			}
			var msgs []engine.Message
			for _, id := range cardutil.SortedEnemyIDs(g) {
				enemy := g.Entity(id)
				if enemy != nil {
					msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: u.Owner})
				}
			}
			return msgs
		},
	})

	// Indomitable reprint: alias core 01082.
	if b := engine.LookupBehavior("01082"); b != nil {
		engine.RegisterBehavior("26017", b)
	}

	// Defiance: boost-discard defense — no boost window; approximated to
	// +2 DEF.
	engine.RegisterBehavior("26018", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, ec *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			return engine.Defends{Defender: p.ID, Against: against, DefBonus: 2}, nil, true
		},
	})

	// Side Step reprint: alias quicksilver 14015.
	if b := engine.LookupBehavior("14015"); b != nil {
		engine.RegisterBehavior("26019", b)
	}

	// Get Behind Me! reprint: alias core 01078.
	if b := engine.LookupBehavior("01078"); b != nil {
		engine.RegisterBehavior("26020", b)
	}

	// Preservation: spent-response — plain resource.
	engine.RegisterBehavior("26021", &engine.Behavior{})

	// Machine Man: up to 3 resources → +1/+1 each (per-use window
	// approximated to a flat +1/+1 ability with cost 1).
	engine.RegisterBehavior("26022", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.spend1ResourceMachineMan1Thw1AtkThisPhase"), Type: engine.AbilityAction,
				Cost: 1,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.AllyStatBonus{Ally: self, THW: 1, ATK: 1}}
				},
			}}
		},
	})

	// Avengers Mansion reprint.
	if b := engine.LookupBehavior("01091"); b != nil {
		engine.RegisterBehavior("26023", b)
	}

	// Reboot: ready an Android character + heal 1.
	engine.RegisterBehavior("26024", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			for _, q := range g.Players {
				for _, id := range q.Allies {
					a := g.Allies[id]
					if a != nil && a.EDef().HasTrait("android") {
						picks = append(picks, engine.Choice{Label: engine.S(a.EDef().Name), Kind: engine.ChoiceTarget, SourceID: a.ID, CardCode: a.Code}.
							Msgs(engine.ReadyEntity{ID: a.ID}, engine.HealEntity{Target: a.ID, N: 1}))
					}
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.rebootReadyWhichAndroidCharacter"), picks...)}}
		},
	})

	// Basic resources.
	engine.RegisterBehavior("26025", &engine.Behavior{})
	engine.RegisterBehavior("26026", &engine.Behavior{})
	engine.RegisterBehavior("26027", &engine.Behavior{})

	// Corrupted Programming obligation.
	engine.RegisterBehavior("26028", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			if p.Exhausted {
				return []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
			}
			return []engine.Message{
				engine.ExhaustEntity{ID: p.ID},
				engine.ObligationResolve{Player: p.ID, Card: card, Remove: true},
			}
		},
	})

	// Assault Training: aggression event recursion (alias nebula 22034 shape).
	engine.RegisterBehavior("26033", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 2}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			if s == nil || s.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustCounterShuffleAnAggressionEventFromDiscard"), Type: engine.AbilityAction,
				Exhaust: true, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.EOwner())
					if p == nil {
						return nil
					}
					var picks []engine.Choice
					for _, c := range p.Discard {
						def := c.Def()
						if def.Type == "event" && def.Aspect == "aggression" {
							picks = append(picks, engine.Choice{Label: engine.S(def.Name), Kind: engine.ChoiceCard, CardCode: def.Code}.
								Msgs(engine.ShuffleIntoDeck{Player: p.ID, CardID: c.ID}))
						}
					}
					if len(picks) == 0 {
						return nil
					}
					return append([]engine.Message{engine.AddEntityCounter{ID: self, N: -1}},
						engine.AskQuestion{Player: p.ID, Question: engine.Ask(
							engine.Tf("c.shuffleWhichAggressionEventIntoYourDeck"), picks...)})
				},
			}}
		},
	})

	// Chance Encounter: on scheme defeat, ally from deck/discard.
	engine.RegisterBehavior("26034", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || len(g.SideSchemes) == 0 {
				return nil
			}
			var picks []engine.Choice
			for _, id := range cardutil.SortedIDs(g.SideSchemes) {
				s := g.SideSchemes[id]
				picks = append(picks, engine.Choice{Label: engine.S(s.EDef().Name), Kind: engine.ChoiceTarget, SourceID: id, CardCode: s.Code}.
					Msgs(engine.AttachUpgrade{ID: e.EID(), Target: id}))
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.attachChanceEncounterToWhichSideScheme"), picks...)}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.SchemeDefeated)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil || d.Scheme != u.AttachTo {
				return nil
			}
			p := g.Player(u.Owner)
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			seen := map[string]bool{}
			for _, c := range append(append(engine.CardList{}, p.Deck...), p.Discard...) {
				def := c.Def()
				if def.Type == "ally" && !seen[c.Code] {
					seen[c.Code] = true
					if _, ok := p.Deck.Find(c.ID); ok {
						picks = append(picks, engine.Choice{Label: engine.S(def.Name + " (deck)"), Kind: engine.ChoiceCard, CardCode: def.Code}.
							Msgs(engine.TakeDeckCard{Player: p.ID, CardID: c.ID}, engine.ShufflePlayerDeck{Player: p.ID}))
					} else {
						picks = append(picks, engine.Choice{Label: engine.S(def.Name + " (discard)"), Kind: engine.ChoiceCard, CardCode: def.Code}.
							Msgs(engine.ReturnDiscardCard{Player: p.ID, CardID: c.ID}))
					}
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.chanceEncounterAddWhichAllyToHand"), picks...)}}
		},
	})

	// Joining Forces: group-play an Avenger and a Guardian ally
	// (approximated to the owner playing one of each from hand, free).
	engine.RegisterBehavior("26035", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			seen := map[string]bool{}
			for _, q := range g.Players {
				for _, c := range q.Hand {
					def := c.Def()
					if def.Type == "ally" && (def.HasTrait("avenger") || def.HasTrait("guardian")) && !seen[c.Code] {
						seen[c.Code] = true
						picks = append(picks, engine.Choice{Label: engine.S(def.Name), Kind: engine.ChoiceCard, CardCode: def.Code}.
							Msgs(engine.PlayCard{Player: e.EOwner(), Card: c, Paid: engine.CostPaid{}}))
					}
				}
			}
			if len(picks) == 0 {
				return nil
			}
			q := engine.AskN(engine.Tf("c.joiningForcesPutWhichAlliesIntoPlayFree"), 2, picks...)
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: q}}
		},
	})

	// Meditation: exhaust alter-ego → play a card at -3.
	engine.RegisterBehavior("26036", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || p.IsHero() || p.Exhausted {
				return nil
			}
			var picks []engine.Choice
			for _, c := range p.Hand {
				def := c.Def()
				if def.Cost != nil && *def.Cost > 0 {
					picks = append(picks, engine.Choice{Label: engine.S(def.Name + " (-3)"), Kind: engine.ChoicePlay, CardCode: def.Code}.
						Msgs(engine.ExhaustEntity{ID: p.ID},
							engine.CostDiscountApply{Player: p.ID, Amount: 3}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.meditationDiscountWhichCard"), picks...)}}
		},
	})
}

func registerNemesis() {
	// Ultron (minion): Toughness printed; drone spawn on attack.
	engine.RegisterBehavior("26029", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			aa, ok := msg.(engine.AskAttack)
			mn := g.Minions[e.EID()]
			if !ok || mn == nil || aa.Enemy != e.EID() {
				return nil
			}
			if g.EnvironmentByCode("26031") == nil && g.EnvironmentByCode("26031a") == nil && g.EnvironmentByCode("26031b") == nil {
				return nil
			}
			return []engine.Message{engine.SpawnDrone{Player: aa.Player}}
		},
	})

	// Ultron Unleashed: fetch the Drones environment; everyone spawns a
	// drone.
	engine.RegisterBehavior("26030", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			msgs := []engine.Message{}
			for _, p := range g.Players {
				msgs = append(msgs, engine.SpawnDrone{Player: p.ID})
			}
			return msgs
		},
	})

	// Ultron Drones environment: drone stats covered by spawn; the
	// defeated-card routing is engine-side.
	engine.RegisterBehavior("26031", &engine.Behavior{})

	// Relentless Android: drones or random discard.
	engine.RegisterBehavior("26032", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			drones := g.EnvironmentByCode("26031") != nil
			if drones {
				return []engine.Message{engine.SpawnDrone{Player: p.ID}, engine.SpawnDrone{Player: p.ID}}
			}
			for i := 0; i < 2 && len(p.Hand) > 0; i++ {
				j := g.Random(len(p.Hand))
				c := p.Hand[j]
				p.Hand = append(p.Hand[:j], p.Hand[j+1:]...)
				p.Discard = append(p.Discard, c)
			}
			return nil
		},
	})
}

// ---- helpers ----

func schemePicks(g *engine.Game, n int, pid engine.PlayerID) []engine.Choice {
	return cardutil.SchemeChoices(g, func(s engine.EntityID) []engine.Message {
		return []engine.Message{engine.ThwartScheme{Scheme: s, N: n, Source: pid}}
	})
}
