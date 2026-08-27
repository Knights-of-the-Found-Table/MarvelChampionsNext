// Package angel registers the Angel hero pack: the identity (Angel /
// Archangel form branches approximated to the Angel side), signature
// cards and the Harpoon nemesis set.
package angel

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerAngel()
	registerSignatures()
	registerNemesis()
	registerObligation()
	registerAngelExtras()
}

// registerAngel installs the identity (42001a/b/c). Archangel-form
// branches are approximated to their Angel side.
func registerAngel() {
	engine.RegisterBehavior("42001", &engine.Behavior{
		// Angel of Life — after you play an [[AERIAL]] event, draw 1
		// (limit once per phase; approximated once per round).
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.EventPlayed)
			if !ok || m.Player != e.EID() {
				return nil
			}
			if !m.Card.Def().HasTrait("aerial") {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil {
				return nil
			}
			if g.UsedThisRound["42001-aerial"] {
				return nil
			}
			g.UsedThisRound["42001-aerial"] = true
			g.TLogf("c.angelOfLifeDraw1")
			return []engine.Message{engine.DrawCards{Player: p.ID, N: 1}}
		},
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			return []engine.Ability{{
				// Regrowth — heal 1 (limit once per round).
				Label:        engine.Tf("c.regrowthHeal1Damage"),
				Type:         engine.AbilityAction,
				AlterEgoOnly: true,
				OncePerRound: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.HealEntity{Target: self, N: 1}}
				},
			}}
		},
	})
}

func registerSignatures() {
	// 42002 Psylocke: after she attacks — Angel heals her 1 (Archangel
	// readies; approximation: always heal).
	engine.RegisterBehavior("42002", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.AllyAttackWindow)
			if !ok || w.Ally != e.EID() {
				return nil
			}
			return []engine.Message{engine.HealEntity{Target: e.EID(), N: 1}}
		},
	})

	// 42003 Adaptive Plumage: Angel — remove 3 threat + confuse (the
	// Archangel attack branch is approximated away).
	engine.RegisterBehavior("42003", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			var msgs []engine.Message
			var choices []engine.Choice
			for _, id := range g.Schemes() {
				s := g.Entity(id)
				choices = append(choices, engine.Choice{
					Label: engine.Tf("m.cardName", s), Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: s.ECode(),
				}.Msgs(engine.ThwartScheme{Scheme: id, N: 3, Source: pid}))
			}
			if len(choices) > 0 {
				msgs = append(msgs, engine.AskQuestion{Player: pid, Question: engine.Ask(engine.Tf("c.adaptivePlumageRemove3Threat"), choices...)})
			}
			confuse := cardutil.EnemyChoices(g, 0, pid, func(t engine.EntityID) []engine.Message {
				return []engine.Message{engine.ConfuseEntity{Target: t}}
			})
			if len(confuse) > 0 {
				msgs = append(msgs, engine.AskQuestion{Player: pid, Question: engine.Ask(engine.Tf("c.adaptivePlumageConfuseAnEnemy"), confuse...)})
			}
			return msgs
		},
	})

	// 42004 Aerial Agility: Angel — ignore boost icons for this attack
	// (approximation: played from the defense prompt, cancels boosts).
	engine.RegisterBehavior("42004", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			if !p.IsHero() {
				return engine.Defends{}, nil, false
			}
			var extra []engine.Message
			if v := g.Villains[against]; v != nil && v.BoostCount > 0 {
				g.TLogf("c.aerialAgilityIgnoresBoostIcons", v.BoostCount)
				v.BoostCount = 0
			}
			return engine.Defends{Defender: p.ID, Against: against, Undefended: true}, extra, true
		},
	})

	// 42005 Metamorphosis: change form; then draw 1 (Warren) / remove 2
	// threat (Angel) / deal 3 (Archangel; approximated to the threat
	// branch).
	engine.RegisterBehavior("42005", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			msgs := []engine.Message{engine.ChangeForm{Player: pid}}
			if p.IsHero() {
				// was alter-ego: drew after flipping; now hero: threat
				var choices []engine.Choice
				for _, id := range g.Schemes() {
					s := g.Entity(id)
					choices = append(choices, engine.Choice{
						Label: engine.Tf("m.cardName", s), Kind: engine.ChoiceTarget,
						SourceID: id, CardCode: s.ECode(),
					}.Msgs(engine.ThwartScheme{Scheme: id, N: 2, Source: pid}))
				}
				if len(choices) > 0 {
					msgs = append(msgs, engine.AskQuestion{Player: pid, Question: engine.Ask(engine.Tf("c.metamorphosisRemove2Threat"), choices...)})
				}
			} else {
				msgs = append(msgs, engine.DrawCards{Player: pid, N: 1})
			}
			return msgs
		},
	})

	// 42006 Natural Flight: remove 4 threat.
	engine.RegisterBehavior("42006", &engine.Behavior{
		OnPlay: cardutil.ChooseScheme(engine.Tf("c.chooseAScheme", "Natural Flight"), func(g *engine.Game, e engine.Entity) int { return 4 }),
	})

	// 42007 Razor Dive: deal 6 damage (overkill/piercing not modeled).
	engine.RegisterBehavior("42007", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy(engine.Tf("c.razorDiveDeal6Damage"), func(g *engine.Game, e engine.Entity) (int, []engine.Message) {
			return 6, nil
		}),
	})

	// 42008 Avian Anatomy (resource): after paying for an AERIAL event,
	// return it to hand (approximation: auto-return).
	engine.RegisterBehavior("42008", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.EventPlayed)
			if !ok || m.Player != e.EOwner() {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil || !m.Card.Def().HasTrait("aerial") {
				return nil
			}
			g.TLogf("c.avianAnatomyReturns", m.Card)
			return []engine.Message{engine.ReturnDiscardCard{Player: p.ID, CardID: m.Card.ID}}
		},
	})

	// 42009 Worthington Industries: shuffle an AERIAL card from discard
	// into your deck; alter-ego: draw 1.
	engine.RegisterBehavior("42009", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			hasAerial := false
			for _, c := range p.Discard {
				if c.Def().HasTrait("aerial") {
					hasAerial = true
					break
				}
			}
			if !hasAerial {
				return nil
			}
			return []engine.Ability{{
				Label:   engine.Tf("c.worthingtonIndustriesShuffleAnAerialCardIntoYourDeck"),
				Type:    engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					var choices []engine.Choice
					for _, c := range p.Discard {
						if !c.Def().HasTrait("aerial") {
							continue
						}
						msgs := []engine.Message{engine.ShuffleIntoDeck{Player: p.ID, CardID: c.ID}}
						if !p.IsHero() {
							msgs = append(msgs, engine.DrawCards{Player: p.ID, N: 1})
						}
						choices = append(choices, engine.Choice{
							Label: engine.Tf("c.shuffleInName", c), Kind: engine.ChoiceCard, CardCode: c.Code,
						}.Msgs(msgs...))
					}
					if len(choices) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.worthingtonIndustriesWhichAerialCard"), choices...)}}
				},
			}}
		},
	})

	// 42010 Techno-Organic Wings: Angel — ready your hero (the Archangel
	// discount branch is approximated away).
	engine.RegisterBehavior("42010", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label:    engine.Tf("c.technoOrganicWingsReadyYourHero"),
				Type:     engine.AbilityAction,
				Exhaust:  true,
				HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.ReadyEntity{ID: e.EOwner()}}
				},
			}}
		},
	})

	// 42017 Render Medical Aid lives in extras (player side scheme).

	// 42018 Angel's Aerie: fatigue counters after defending; AE action —
	// heal 1 per counter.
	engine.RegisterBehavior("42018", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowDefended)
			s, ok2 := e.(*engine.Support)
			if !ok || !ok2 || w.Defender != s.Owner {
				return nil
			}
			s.Counters++
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s, ok := e.(*engine.Support)
			if !ok || s.Counters == 0 {
				return nil
			}
			return []engine.Ability{{
				Label:        engine.Tf("c.angelSAerieHeal1PerFatigueCounter"),
				Type:         engine.AbilityAction,
				Exhaust:      true,
				AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					n := s.Counters
					s.Counters = 0
					return []engine.Message{engine.HealEntity{Target: s.Owner, N: n}}
				},
			}}
		},
	})
}

// registerNemesis installs the Harpoon set.
func registerNemesis() {
	// 42025 Harpoon: when revealed — mill until an event; indirect damage
	// equal to its cost (approximation: damage to the identity).
	engine.RegisterBehavior("42025", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn, ok := e.(*engine.Minion)
			if !ok || mn.EngagedWith == "" {
				return nil
			}
			p := g.Player(mn.EngagedWith)
			if p == nil {
				return nil
			}
			n := millUntilEvent(g, p)
			if n > 0 {
				return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: n, Source: e.EID()}}
			}
			return nil
		},
	})

	// 42027 Harpoon's Harpoon: attach to Harpoon (or the villain); after
	// he attacks you, 2 indirect damage (approximation: on attach, +1
	// ATK).
	engine.RegisterBehavior("42027", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for id, mn := range g.Minions {
				if mn.Code == "42025" {
					t.Target = id
					return []engine.Message{engine.BoostEnemyAttack{Enemy: id, N: 1}}
				}
			}
			for id := range g.Villains {
				t.Target = id
				return []engine.Message{engine.BoostEnemyAttack{Enemy: id, N: 1}}
			}
			return nil
		},
	})

	// 42028 Spear Shot: mill until an event; indirect damage = its cost.
	engine.RegisterBehavior("42028", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			n := millUntilEvent(g, p)
			if n > 0 {
				return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: n, Source: t.ID}}
			}
			return nil
		},
	})
}

// millUntilEvent discards from a player's deck until an event appears and
// returns its printed cost.
func millUntilEvent(g *engine.Game, p *engine.Player) int {
	var milled []engine.Card
	n := 0
	for len(p.Deck) > 0 {
		c := p.Deck[0]
		p.Deck = p.Deck[1:]
		if c.Def().Type == "event" {
			n = cardutil.Cost(c.Def())
			milled = append(milled, c)
			break
		}
		milled = append(milled, c)
	}
	if len(milled) > 0 {
		p.Discard = append(p.Discard, milled...)
		g.TLogf("c.milledCardsEventCost", len(milled), n)
	}
	return n
}

// registerObligation installs Apocalyptic Influence.
func registerObligation() {
	// When Revealed: if Archangel (approximated: hero form), place 2
	// threat; otherwise change to hero form. AE action: first player
	// gains an encounter card → discard this.
	engine.RegisterBehavior("42024", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var msgs []engine.Message
			if p.IsHero() {
				if g.MainScheme != nil {
					msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: engine.EntityID("obligation")})
				}
			} else {
				msgs = append(msgs, engine.ChangeForm{Player: p.ID})
			}
			msgs = append(msgs, engine.ObligationResolve{Player: p.ID, Card: card})
			// follow-up: deal an encounter card to the first player
			msgs = append(msgs, engine.DealEncounterToPlayer{Player: cardutil.FirstPlayerID(g)})
			return []engine.Message{engine.AskQuestion{
				Player: p.ID,
				Question: engine.Ask(engine.Tf("c.apocalypticInfluence"),
					engine.Choice{ID: "ok", Label: engine.Tf("c.resolve"), Kind: engine.ChoiceLabel}.Msgs(msgs...),
				),
			}}
		},
	})
}
