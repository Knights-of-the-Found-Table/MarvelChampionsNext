// complete.go implements the remaining Phoenix pack cards (34007–34035):
// the Psionic upgrade suite, shared X-Men pool cards and the Dark Phoenix
// nemesis set.
package phoenix

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() {
	registerRemainingPhoenix()
}

func registerRemainingPhoenix() {
	// 34007 Telekinetic Shield: attach to your identity; attack damage
	// lands here (identity-only approximation); pops at 5 stored.
	engine.RegisterBehavior("34007", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			u := g.Upgrades[e.EID()]
			p := g.Player(e.EOwner())
			if u == nil || p == nil {
				return nil
			}
			return []engine.Message{engine.AttachUpgrade{ID: u.ID, Target: p.ID}}
		},
		DamagePrevention: func(g *engine.Game, u *engine.Upgrade, p *engine.Player, n int) (int, int) {
			if u.Counters >= 5 {
				return 0, 0
			}
			room := 5 - u.Counters
			pv := n
			if pv > room {
				pv = room
			}
			u.Counters += pv
			g.TLogf("c.telekineticShieldAbsorbsDamage5", pv, u.Counters)
			if u.Counters >= 5 {
				g.Delete(u.ID)
				if p != nil {
					p.Discard = append(p.Discard, engine.Card{ID: g.NextCardID(), Code: u.Code, Owner: p.ID})
				}
			}
			return pv, 0
		},
	})

	// 34008 Mental Paralysis: attach to a non-Elite minion; it cannot
	// activate; discard when the owner flips to alter-ego form.
	engine.RegisterBehavior("34008", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			var choices []engine.Choice
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn == nil || mn.EDef().HasTrait("elite") {
					continue
				}
				choices = append(choices, engine.Choice{
					Label: engine.Tf("c.attachTo2", cardutil.EnemyLabel(mn)), Kind: engine.ChoiceTarget, SourceID: id,
				}.Msgs(engine.AttachUpgrade{ID: u.ID, Target: id}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   e.EOwner(),
				Question: engine.Ask(engine.Tf("c.mentalParalysisAttachToANonEliteMinion"), choices...),
			}}
		},
		MinionActivate: func(g *engine.Game, mn *engine.Minion, p *engine.Player) []engine.Message {
			for _, uid := range g.Upgrades {
				if uid != nil && uid.Code == "34008" && uid.AttachTo == mn.ID {
					g.TLogf("c.isParalyzedAndCannotActivate", mn)
					return nil
				}
			}
			// Fall through to the default activation is not possible from
			// here; treat as canceled.
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.ChangeForm)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil || m.Player != u.Owner || g.Player(m.Player) == nil {
				return nil
			}
			if p := g.Player(m.Player); p != nil && !p.IsHero() {
				g.Delete(u.ID)
				p.Discard = append(p.Discard, engine.Card{ID: g.NextCardID(), Code: u.Code, Owner: p.ID})
				g.TLogf("c.mentalParalysisIsDiscarded")
			}
			return nil
		},
	})

	// 34009 Mind Control: attach to a non-Elite minion and convert it
	// into a controlled ally.
	engine.RegisterBehavior("34009", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn == nil || mn.EDef().HasTrait("elite") {
					continue
				}
				choices = append(choices, engine.Choice{
					Label: engine.Tf("c.control", cardutil.EnemyLabel(mn)), Kind: engine.ChoiceTarget, SourceID: id,
				}.Msgs(engine.ConvertMinionToAlly{MinionID: id, Owner: p.ID, Consequential: 1}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.mindControlTakeControlOfANonEliteMinion"), choices...),
			}}
		},
	})

	// 34010 Telekinetic Attack: 7 damage (+2 with the UNLEASHED trait).
	engine.RegisterBehavior("34010", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			dmg := 7
			if g.EntityHasTrait(p.ID, "unleashed") {
				dmg += 2
			}
			choices := cardutil.EnemyChoices(g, dmg, p.ID, func(target engine.EntityID) []engine.Message {
				return []engine.Message{engine.DamageEntity{Target: target, Damage: dmg, Source: p.ID}}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.telekineticAttack"), choices...),
			}}
		},
	})

	// 34011 Psychic Blast: 4 damage to the villain (+4 to each engaged
	// minion when UNLEASHED).
	engine.RegisterBehavior("34011", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			v := firstVillainEntity(g)
			if p == nil || v == nil {
				return nil
			}
			msgs := []engine.Message{engine.DamageEntity{Target: v.ID, Damage: 4, Source: p.ID}}
			if g.EntityHasTrait(p.ID, "unleashed") {
				for _, id := range cardutil.SortedIDs(g.Minions) {
					if mn := g.Minions[id]; mn != nil && mn.EngagedWith == p.ID {
						msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 4, Source: p.ID})
					}
				}
			}
			return msgs
		},
	})

	// 34012 Telepathic Trickery: remove 4 threat (stun + confuse when
	// UNLEASHED).
	engine.RegisterBehavior("34012", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			base := cardutil.ChooseScheme(engine.Tf("c.telepathicTrickeryChooseAScheme"), func(g *engine.Game, e engine.Entity) int { return 4 })
			msgs := base(g, e)
			if g.EntityHasTrait(p.ID, "unleashed") {
				if v := firstVillainEntity(g); v != nil {
					msgs = append(msgs, engine.StunEntity{Target: v.ID}, engine.ConfuseEntity{Target: v.ID})
				}
			}
			return msgs
		},
	})

	// 34013 Phoenix Firebird: adjust the identity's power counters.
	engine.RegisterBehavior("34013", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player: p.ID,
				Question: engine.Ask(engine.Tf("c.phoenixFirebirdChoose"), []engine.Choice{
					engine.Choice{
						ID: "ready", Label: engine.Tf("c.remove1PowerCounterReadyPhoenix"), Kind: engine.ChoiceLabel,
					}.Msgs(engine.AddEntityCounter{ID: p.ID, N: -1}, engine.ReadyEntity{ID: p.ID}),
					engine.Choice{
						ID: "charge", Label: engine.Tf("c.place2PowerCountersOnPhoenixForce"), Kind: engine.ChoiceLabel,
					}.Msgs(engine.AddEntityCounter{ID: p.ID, N: 2}),
				}...),
			}}
		},
	})

	// 34014 Banshee: after he thwarts, confuse a minion.
	engine.RegisterBehavior("34014", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyThwartWindow)
			if !ok || m.Ally != e.EID() {
				return nil
			}
			for _, id := range cardutil.SortedIDs(g.Minions) {
				if mn := g.Minions[id]; mn != nil {
					return []engine.Message{engine.ConfuseEntity{Target: id}}
				}
			}
			return nil
		},
	})

	// 34015 Marvel Girl: after she attacks a minion, remove threat equal
	// to its printed SCH from the main scheme.
	engine.RegisterBehavior("34015", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyAttackWindow)
			if !ok || m.Ally != e.EID() || g.MainScheme == nil {
				return nil
			}
			if mn := g.Minions[m.Target]; mn != nil {
				return []engine.Message{engine.ThwartScheme{Scheme: g.MainScheme.ID, N: mn.SchemeVal, Source: e.EOwner()}}
			}
			return nil
		},
	})

	// 34016 Mission Training: +1 THW / +2 HP to an X-Men ally.
	engine.RegisterBehavior("34016", &engine.Behavior{
		OnPlay: attachTrainingPHX(func() (int, int, int) { return 1, 0, 2 }),
	})

	// 34017 Psychic Manipulation: the scheme-to-thwart conversion has no
	// interrupt window from hand.
	engine.RegisterBehavior("34017", &engine.Behavior{})

	// 34018 Mutant Peacekeepers: exhaust hero + X-Men allies → remove
	// their total THW (single-scheme approximation).
	engine.RegisterBehavior("34018", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "x-men") && p.IsHero() && !p.Exhausted
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			total := p.ThwartStat(g)
			var msgs []engine.Message
			msgs = append(msgs, engine.ExhaustEntity{ID: p.ID})
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a == nil || a.Exhausted || !a.EDef().HasTrait("x-men") {
					continue
				}
				total += a.ThwartVal + a.BonusTHW + a.PermTHW
				msgs = append(msgs, engine.ExhaustEntity{ID: a.ID})
			}
			if g.MainScheme != nil {
				msgs = append(msgs, engine.ThwartScheme{Scheme: g.MainScheme.ID, N: total, Source: p.ID})
			}
			return msgs
		},
	})

	// 34019 Swift Retribution: the villain schemes; deal it 4 damage.
	engine.RegisterBehavior("34019", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			v := firstVillainEntity(g)
			if p == nil || v == nil {
				return nil
			}
			msgs := []engine.Message{engine.DamageEntity{Target: v.ID, Damage: 4, Source: p.ID}}
			if g.MainScheme != nil {
				msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: v.SchemeVal, Source: v.ID})
			}
			return msgs
		},
	})

	// 34020 Passion for Justice: the rider resolves in handlePlayCard.
	engine.RegisterBehavior("34020", &engine.Behavior{})

	// 34021 Storm: costs 1 less for MUTANT/X-MEN; after she thwarts, move
	// 2 threat to another scheme.
	engine.RegisterBehavior("34021", &engine.Behavior{
		CardCost: func(g *engine.Game, p *engine.Player, def *data.CardDef) int {
			if g.EntityHasTrait(p.ID, "x-men") || g.EntityHasTrait(p.ID, "mutant") {
				return 1
			}
			return 0
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyThwartWindow)
			a := g.Allies[e.EID()]
			if !ok || m.Ally != e.EID() || a == nil {
				return nil
			}
			var choices []engine.Choice
			for _, id := range g.Schemes() {
				if id == m.Scheme {
					continue
				}
				s := g.Entity(id)
				choices = append(choices, engine.Choice{
					Label: engine.S("Move 2 threat to " + s.EDef().Name), Kind: engine.ChoiceTarget, SourceID: id,
				}.Msgs(engine.ThwartScheme{Scheme: m.Scheme, N: 2, Source: a.Owner},
					engine.SchemeThreat{Scheme: id, N: 2, Source: a.Owner}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   a.Owner,
				Question: engine.Ask(engine.Tf("c.stormMove2ThreatToAnotherScheme"), choices...),
			}}
		},
	})

	// 34022 Cerebro: Alter-Ego action — search the top 5 (whole deck with
	// a PSIONIC character) for an X-Men ally.
	engine.RegisterBehavior("34022", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "mutant")
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.cerebroSearchForAnXMenAlly"), Type: engine.AbilityAction,
				AlterEgoOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(g.ActiveTurn)
					if p == nil {
						return nil
					}
					limit := 5
					if g.EntityHasTrait(p.ID, "psionic") {
						limit = len(p.Deck)
					}
					for i := 0; i < limit && i < len(p.Deck); i++ {
						c := p.Deck[i]
						if c.Def().Type == "ally" && c.Def().HasTrait("x-men") {
							p.Deck = append(p.Deck[:i], p.Deck[i+1:]...)
							c.Owner = p.ID
							p.Hand = append(p.Hand, c)
							g.TLogf("c.cerebroFinds", c)
							return nil
						}
					}
					return nil
				},
			}}
		},
	})

	// 34023 Psychic Rapport: alias the Cyclops printing.
	if b := engine.LookupBehavior("33023"); b != nil {
		engine.RegisterBehavior("34023", b)
	}

	// 34024 Down Time: +2 REC while in alter-ego form (applied to both
	// sides; the alter-ego-only restriction is player-enforced).
	engine.RegisterBehavior("34024", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{REC: 2} },
	})

	// 34025–34027 basic resources; 34028 Burning Hunger has no text.
	for _, code := range []string{"34025", "34026", "34027", "34028"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	// 34029 Dark Phoenix: her scheme threat lands on Consume the World;
	// on reveal, search it.
	engine.RegisterBehavior("34029", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionEntersPlay)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
				for _, c := range *zone {
					if c.Code == "34030" {
						zone.Remove(c.ID)
						return []engine.Message{engine.RevealEncounterCard{Player: m.Player, Card: c}}
					}
				}
			}
			return nil
		},
		MinionActivate: func(g *engine.Game, mn *engine.Minion, p *engine.Player) []engine.Message {
			// Schemes against you — threat routes to Consume the World.
			for _, s := range g.SideSchemes {
				if s != nil && s.Code == "34030" {
					return []engine.Message{engine.SchemeThreat{Scheme: s.ID, N: mn.SchemeVal, Source: mn.ID}}
				}
			}
			if !p.IsHero() && g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: mn.SchemeVal, Source: mn.ID}}
			}
			return []engine.Message{engine.AskAttack{Enemy: mn.ID, Player: p.ID}}
		},
	})

	// 34030 Consume the World: 12 threat here loses the game.
	engine.RegisterBehavior("34030", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeThreat)
			s := g.SideSchemes[e.EID()]
			if !ok || s == nil || m.Scheme != s.ID {
				return nil
			}
			if s.Threat+m.N >= 12 {
				return []engine.Message{engine.GameOver{Won: false, Reason: engine.Tf("reason.darkPhoenix")}}
			}
			return nil
		},
	})

	// 34031 Fiery Rage: Peril — no effect.
	engine.RegisterBehavior("34031", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			return nil
		},
	})

	// 34032 Psychic Assault: 3 damage and confuse.
	engine.RegisterBehavior("34032", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "psionic")
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			choices := cardutil.EnemyChoices(g, 3, p.ID, func(target engine.EntityID) []engine.Message {
				return []engine.Message{
					engine.DamageEntity{Target: target, Damage: 3, Source: p.ID},
					engine.ConfuseEntity{Target: target},
				}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.psychicAssault"), choices...),
			}}
		},
	})

	// 34033 Psychic Misdirection: defense event — the damage is dealt to
	// another enemy instead (approximation: prevented, then the attack
	// value hits the attacker).
	engine.RegisterBehavior("34033", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "psionic")
		},
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			atk := g.AttackValueOf(against)
			return engine.Defends{Defender: p.ID, Against: against, PreventAll: true},
				[]engine.Message{engine.DamageEntity{Target: against, Damage: atk, Source: p.ID}}, true
		},
	})

	// 34034 Psychic Kicker: ready an ally; +2 THW / +2 ATK this phase.
	engine.RegisterBehavior("34034", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "psionic")
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil {
					aid := id
					choices = append(choices, engine.Choice{
						Label: engine.S("Ready " + a.EDef().Name), Kind: engine.ChoiceTarget, SourceID: aid, CardCode: a.Code,
					}.Msgs(engine.ReadyEntity{ID: aid}, engine.AllyStatBonus{Ally: aid, THW: 2, ATK: 2}))
				}
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.psychicKickerReadyAnAlly2Thw2Atk"), choices...),
			}}
		},
	})

	// 34035 Soul Sisters: ready your hero and heal 2.
	engine.RegisterBehavior("34035", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			return []engine.Message{
				engine.ReadyEntity{ID: pid},
				engine.HealEntity{Target: pid, N: 2},
			}
		},
	})
}

// attachTrainingPHX attaches a Training upgrade to an X-Men ally.
func attachTrainingPHX(bonus func() (thw, atk, hp int)) func(g *engine.Game, e engine.Entity) []engine.Message {
	return func(g *engine.Game, e engine.Entity) []engine.Message {
		u := g.Upgrades[e.EID()]
		p := g.Player(e.EOwner())
		if u == nil || p == nil {
			return nil
		}
		thw, atk, hp := bonus()
		var choices []engine.Choice
		for _, id := range p.Allies {
			a := g.Allies[id]
			if a == nil || !a.EDef().HasTrait("x-men") {
				continue
			}
			trained := false
			for _, x := range g.Upgrades {
				if x != nil && x.AttachTo == id && x.EDef().HasTrait("training") {
					trained = true
				}
			}
			if trained {
				continue
			}
			choices = append(choices, engine.Choice{
				Label: engine.S("Attach to " + a.EDef().Name), Kind: engine.ChoiceTarget, SourceID: id, CardCode: a.Code,
			}.Msgs(engine.AttachUpgrade{ID: u.ID, Target: id, THW: thw, ATK: atk, MaxHP: hp}))
		}
		if len(choices) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{
			Player:   p.ID,
			Question: engine.Ask(engine.S(u.EDef().Name+" — attach to an X-Men ally"), choices...),
		}}
	}
}

// firstVillainEntity returns the active or first villain entity.
func firstVillainEntity(g *engine.Game) *engine.Villain {
	if id := g.ActiveVillain; id != "" {
		if v := g.Villains[id]; v != nil {
			return v
		}
	}
	for _, v := range g.Villains {
		if v != nil {
			return v
		}
	}
	return nil
}
