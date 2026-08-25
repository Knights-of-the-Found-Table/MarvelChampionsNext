package x23

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func registerX23Extras() {
	// 43013 Boom Boom: boom counters; detonate for AoE.
	engine.RegisterBehavior("43013", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			a := g.Allies[e.EID()]
			if a == nil {
				return nil
			}
			return []engine.Ability{
				{
					Label: engine.Tf("c.boomBoomArmABoomCounterOrDetonate"), Type: engine.AbilityAction,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						a := g.Allies[self]
						p := g.Player(a.Owner)
						if p == nil {
							return nil
						}
						a.Counters++
						n := a.Counters
						if a.Counters < 2 || len(g.Enemies()) == 0 {
							g.TLogf("c.boomBoomArmsACounter", n)
							return []engine.Message{engine.ExhaustEntity{ID: a.ID}}
						}
						var msgs []engine.Message
						msgs = append(msgs, engine.ExhaustEntity{ID: a.ID}, engine.AllyDestroyed{AllyID: a.ID})
						for _, id := range cardutil.SortedEnemyIDs(g) {
							msgs = append(msgs, engine.DamageEntity{Target: id, Damage: n, Source: p.ID})
						}
						g.TLogf("c.boomBoomDetonatesCounters", n)
						return msgs
					},
				},
			}
		},
	})

	// 43014 Rictor: luck-scaled shockwave.
	engine.RegisterBehavior("43014", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyAttackWindow)
			if !ok || m.Ally != e.EID() {
				return nil
			}
			a := g.Allies[e.EID()]
			p := g.Player(a.Owner)
			if p == nil || len(p.Deck) == 0 {
				return nil
			}
			n := len(p.Deck[0].Def().Resources)
			var msgs []engine.Message
			msgs = append(msgs, engine.MillPlayerDeck{Player: p.ID, N: 1})
			for _, id := range cardutil.SortedIDs(g.Minions) {
				if mn := g.Minions[id]; mn != nil && mn.EngagedWith == p.ID {
					msgs = append(msgs, engine.DamageEntity{Target: id, Damage: n, Source: p.ID})
				}
			}
			return msgs
		},
	})

	// 43015 Shatterstar: +1 ATK vs engaged (approximated flat).
	engine.RegisterBehavior("43015", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyAttackWindow)
			if !ok || m.Ally != e.EID() {
				return nil
			}
			a := g.Allies[e.EID()]
			if a == nil {
				return nil
			}
			a.BonusATK++
			return nil
		},
	})

	// 43016 Critical Hit: stun after attacking.
	engine.RegisterBehavior("43016", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			g.UsedThisRound["43016"] = true
			g.TLogf("c.criticalHitArmedYourNextAttackStuns")
			return nil
		},
	})

	// 43017 Moment of Triumph: heal on overkill (approximated: heal equal
	// to overkill when a basic attack kills; the exact excess tracking is
	// handled at attack time).
	engine.RegisterBehavior("43017", &engine.Behavior{})

	// 43018 Keep Them Busy: defeat removes 5[per_hero] main threat.
	engine.RegisterBehavior("43018", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			if g.MainScheme == nil {
				return nil
			}
			return []engine.Message{engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 5 * len(g.Players), Source: engine.PlayerID("")}}
		},
	})

	// 43019 "Now I'm Mad": low-HP stat swap (IdentityStatsG).
	engine.RegisterBehavior("43019", &engine.Behavior{
		IdentityStatsG: func(g *engine.Game, p *engine.Player, u *engine.Upgrade) engine.StatBonus {
			if p.MaxHP-p.Damage < p.MaxHP/2 {
				return engine.StatBonus{ATK: 1, THW: -1}
			}
			return engine.StatBonus{}
		},
	})

	// 43020 The Direct Approach: attach to a scheme (assault not modeled).
	engine.RegisterBehavior("43020", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			u := g.Upgrades[e.EID()]
			p := g.Player(u.Owner)
			if p == nil || len(g.Schemes()) == 0 {
				return nil
			}
			var choices []engine.Choice
			for _, id := range g.Schemes() {
				s := g.Entity(id)
				choices = append(choices, engine.Choice{
					ID: "sch-" + id.String(), Label: engine.S(s.EDef().Name), Kind: engine.ChoiceTarget,
				}.Msgs(engine.AttachUpgrade{ID: u.ID, Target: id}))
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.theDirectApproachAttachTo"), choices...)}}
		},
	})

	// 43022-43024 Energy/Genius/Strength: deckbuilding.
	for _, code := range []string{"43022", "43023", "43024"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	// 43025 IPAC: facedown encounter → 2 cards.
	engine.RegisterBehavior("43025", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "X-Force")
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.ipacTradeAFacedownEncounterFor2Cards"), Type: engine.AbilityAction,
				HeroOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.Owner)
					if p == nil {
						return nil
					}
					return []engine.Message{
						engine.DealEncounterToPlayer{Player: p.ID},
						engine.DrawCards{Player: p.ID, N: 2},
					}
				},
			}}
		},
	})

	// 43026 X-Bunker: dig for cards.
	engine.RegisterBehavior("43026", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			n := 0
			for _, c := range g.VictoryDisplay {
				if c.Def().Type == "side_scheme" || c.Def().Type == "player_side_scheme" {
					n++
				}
			}
			if n == 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.xBunkerDigCards", n), Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.Owner)
					if p == nil || len(p.Deck) == 0 {
						return nil
					}
					x := len(g.VictoryDisplay)
					if x > len(p.Deck) {
						x = len(p.Deck)
					}
					c := p.Deck[0]
					p.Deck = p.Deck[1:]
					p.Hand = append(p.Hand, c)
					g.TLogf("c.xBunkerRecovers", c)
					return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
				},
			}}
		},
	})

	// 43027 Endurance: +3 HP.
	engine.RegisterBehavior("43027", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			u := g.Upgrades[e.EID()]
			if p := g.Player(u.Owner); p != nil {
				p.MaxHP += 3
				g.TLogf("c.gets3HitPointsEndurance", p.Name)
			}
			return nil
		},
	})

	// 43034-43037 Specializations.
	engine.RegisterBehavior("43034", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{ATK: 1} },
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.BasicAttack); !ok || e.EExhausted() {
				return nil
			}
			u := g.Upgrades[e.EID()]
			return []engine.Message{engine.ExhaustEntity{ID: u.ID}, engine.DrawCards{Player: u.Owner, N: 1}}
		},
	})
	engine.RegisterBehavior("43035", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{DEF: 1} },
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.WindowDefended); !ok || e.EExhausted() {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if u == nil || g.Player(u.Owner) == nil {
				return nil
			}
			return []engine.Message{engine.ExhaustEntity{ID: u.ID}, engine.DrawCards{Player: u.Owner, N: 1}}
		},
	})
	engine.RegisterBehavior("43036", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			u := g.Upgrades[e.EID()]
			if p := g.Player(u.Owner); p != nil {
				p.MaxHP += 4
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowDefended)
			if !ok || e.EExhausted() || w.DamageTaken <= 0 {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if u == nil || w.Defender != u.Owner {
				return nil
			}
			return []engine.Message{engine.ExhaustEntity{ID: u.ID}, engine.DrawCards{Player: u.Owner, N: 1}}
		},
	})
	engine.RegisterBehavior("43037", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{THW: 1} },
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.BasicThwart); !ok || e.EExhausted() {
				return nil
			}
			u := g.Upgrades[e.EID()]
			return []engine.Message{engine.ExhaustEntity{ID: u.ID}, engine.DrawCards{Player: u.Owner, N: 1}}
		},
	})

	// 43021 Specialized Training: defeat deploys a Specialization.
	engine.RegisterBehavior("43021", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			for _, p := range g.Players {
				has := false
				for _, id := range p.Upgrades {
					if u := g.Upgrades[id]; u != nil && u.EDef().HasTrait("Specialization") {
						has = true
					}
				}
				if has {
					continue
				}
				for _, code := range []string{"43034", "43035", "43036", "43037"} {
					if c, zone, ok := firstCard(p, code); ok {
						takeCard(p, c, zone)
						u := &engine.Upgrade{ID: g.NextEntityID(engine.KindUpgrade), Code: c.Code, Owner: p.ID}
						g.Upgrades[u.ID] = u
						p.Upgrades = append(p.Upgrades, u.ID)
						g.TLogf("c.gains", p.Name, c)
						break
					}
				}
			}
			return nil
		},
	})

	// 43038 Predictable Ploy: treachery cancel (TreacheryInterrupt).
	engine.RegisterBehavior("43038", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			for _, c := range g.VictoryDisplay {
				if c.Def().Type == "side_scheme" || c.Def().Type == "player_side_scheme" {
					return true
				}
			}
			return false
		},
		TreacheryInterrupt: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			return []engine.Message{engine.DiscardEncounterCard{Card: card}, engine.RevealNextEncounter{Player: p.ID}}
		},
	})

	// 43039 Rally the Troops: heal allies.
	engine.RegisterBehavior("43039", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				for _, id := range p.Allies {
					msgs = append(msgs, engine.HealEntity{Target: id, N: 2})
				}
			}
			return msgs
		},
	})

	// 43040 Anticipated Attack: tough on defense.
	engine.RegisterBehavior("43040", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			for _, c := range g.VictoryDisplay {
				if c.Def().Type == "side_scheme" || c.Def().Type == "player_side_scheme" {
					return true
				}
			}
			return false
		},
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			d := engine.Defends{Defender: p.ID, Against: against, Undefended: true}
			return d, []engine.Message{engine.ToughEntity{Target: p.ID}}, true
		},
	})
}

func firstCard(p *engine.Player, base string) (engine.Card, string, bool) {
	for _, c := range p.Deck {
		if data.BaseCode(c.Code) == base {
			return c, "deck", true
		}
	}
	for _, c := range p.Discard {
		if data.BaseCode(c.Code) == base {
			return c, "discard", true
		}
	}
	return engine.Card{}, "", false
}

func takeCard(p *engine.Player, c engine.Card, zone string) bool {
	switch zone {
	case "deck":
		_, ok := p.Deck.Remove(c.ID)
		return ok
	case "discard":
		_, ok := p.Discard.Remove(c.ID)
		return ok
	}
	return false
}
