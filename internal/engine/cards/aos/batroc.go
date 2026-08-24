package aos

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() { registerBatroc() }

// alertLevel finds the Alert Level environment (50090a); its Threat-like
// counters ride on Environment.Counters.
func alertLevel(g *engine.Game) *engine.Environment {
	for _, env := range g.Environments {
		if env != nil && data.BaseCode(env.Code) == "50090" {
			return env
		}
	}
	return nil
}

// alertHigh reports whether Alert Level has flipped to its High side
// (tracked via StoredCards marker).
func alertHigh(g *engine.Game) bool {
	env := alertLevel(g)
	return env != nil && len(env.StoredCards) > 0
}

func registerBatroc() {
	// 50086a Batroc: after he attacks, 1 threat on Alert Level; his first
	// defeat resets him to 8 hit points and removes 6 main-scheme threat.
	engine.RegisterBehavior("50086", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok || w.Enemy != e.EID() {
				return nil
			}
			if env := alertLevel(g); env != nil {
				env.Counters++
				g.Logf("Alert Level rises (%d)", env.Counters)
			}
			return nil
		},
		VillainDamageable: func(g *engine.Game, v *engine.Villain, damage int) bool {
			if v.Damage+damage < v.MaxHP {
				return true
			}
			// Heightened Reflexes (50092) prevention: 2 damage per leap
			// counter.
			for _, aid := range v.Attachments {
				if t := g.Attachments[aid]; t != nil && t.Code == "50092" && t.Counters > 0 {
					t.Counters--
					damage -= 2
					g.Logf("Heightened Reflexes prevents 2 damage (%d leap counters left)", t.Counters)
					if v.Damage+damage < v.MaxHP {
						v.Damage += damage
						return false
					}
				}
			}
			if v.Counters == 0 {
				v.Counters = 1 // the once-per-game reset marker
				v.Damage = 0
				g.Logf("Batroc leaps away from defeat and taunts the heroes!")
				if g.MainScheme != nil && g.MainScheme.Threat > 0 {
					g.Push(engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 6, Source: v.ID})
				}
				return false
			}
			return true
		},
	})

	// Main scheme stages.
	engine.RegisterBehavior("50087", &engine.Behavior{})
	engine.RegisterBehavior("50088", &engine.Behavior{})
	// 50089a Extract Captives: entry effect rides on the Alert Level.
	engine.RegisterBehavior("50089", &engine.Behavior{})

	// 50090a Alert Level environment: threat tracked as counters; spend a
	// resource to remove 1; four-per-hero threat flips it High (marker
	// card tucked in StoredCards).
	engine.RegisterBehavior("50090", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			env, ok := e.(*engine.Environment)
			if !ok {
				return nil
			}
			switch msg.(type) {
			case engine.BeginRound, engine.AddEntityCounter:
			default:
				return nil
			}
			if len(env.StoredCards) > 0 || env.Counters < 4*len(g.Players) {
				return nil
			}
			env.Counters = 0
			env.StoredCards = append(env.StoredCards, engine.Card{ID: g.NextCardID(), Code: "50090a"})
			g.LogMajorf("The Alert Level flips to HIGH!")
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			env := g.Environments[e.EID()]
			if env == nil || env.Counters <= 0 || alertHigh(g) {
				return nil
			}
			return []engine.Ability{{
				Label: "Alert Level — spend 1 resource to remove 1 threat", Type: engine.AbilityAction,
				Cost: 1,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					if env := g.Environments[self]; env != nil && env.Counters > 0 {
						env.Counters--
						g.Logf("Alert Level drops to %d", env.Counters)
					}
					return nil
				},
			}}
		},
	})

	// 50091 Rescued Captive: free threat removal, never leaves play.
	engine.RegisterBehavior("50091", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Rescued Captive — remove 1 threat from the main scheme", Type: engine.AbilityAction,
				HeroOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					if g.MainScheme != nil {
						return []engine.Message{engine.ThwartScheme{Scheme: g.MainScheme.ID, N: len(g.Players), Source: self}}
					}
					return nil
				},
			}}
		},
	})

	// 50092 Heightened Reflexes: leap-counter prevention handled inside
	// Batroc's damage hook.
	engine.RegisterBehavior("50092", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			t.Counters = 4
			g.Logf("Heightened Reflexes enters play with 4 leap counters")
			return nil
		},
	})

	// 50093 Embassy Guard: defeat raises the Alert Level.
	engine.RegisterBehavior("50093", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionDefeated)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			if env := alertLevel(g); env != nil {
				env.Counters++
				g.Logf("Embassy Guard's defeat raises the Alert Level (%d)", env.Counters)
			}
			return nil
		},
	})

	// 50094 Embassy Patrol: same defeat trigger.
	engine.RegisterBehavior("50094", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionDefeated)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			if env := alertLevel(g); env != nil {
				env.Counters++
				g.Logf("Embassy Patrol's defeat raises the Alert Level (%d)", env.Counters)
			}
			return nil
		},
	})

	// 50095 Commandeer Security Office: defeat lowers the Alert Level.
	engine.RegisterBehavior("50095", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			if env := alertLevel(g); env != nil && env.Counters > 0 {
				env.Counters -= len(g.Players)
				if env.Counters < 0 {
					env.Counters = 0
				}
				g.Logf("The security office is commandeered — Alert Level drops to %d", env.Counters)
			}
			return nil
		},
	})

	// 50096 Leaping Kick: Batroc schemes (alter-ego) or attacks the
	// healthiest ally (hero; approximated as a direct hit).
	engine.RegisterBehavior("50096", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			for id, v := range g.Villains {
				if data.BaseCode(v.Code) != "50086" {
					continue
				}
				if !p.IsHero() {
					if g.MainScheme != nil {
						return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: v.SchemeVal, Source: id}}
					}
					return nil
				}
				best, bestHP := engine.EntityID(""), -1
				for _, aid := range p.Allies {
					if a := g.Allies[aid]; a != nil && a.HP() > bestHP {
						best, bestHP = aid, a.HP()
					}
				}
				if best != "" {
					return []engine.Message{engine.DamageEntity{Target: best, Damage: v.AttackVal, Source: id}}
				}
				return []engine.Message{engine.AskAttack{Enemy: id, Player: p.ID}}
			}
			return nil
		},
	})

	// 50097 Security Cameras: exhaust your characters or raise the Alert
	// Level (approximated: identity and allies all exhaust or all raise).
	engine.RegisterBehavior("50097", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if !p.IsHero() {
				if env := alertLevel(g); env != nil && env.Counters > 0 {
					env.Counters--
					g.Logf("Security cameras go dark — Alert Level drops")
				}
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			var exhaust, raise []engine.Message
			raiseN := 1
			if alertHigh(g) {
				raiseN = 2
			}
			if env := alertLevel(g); env != nil {
				for i := 0; i < raiseN; i++ {
					raise = append(raise, engine.AddEntityCounter{ID: env.ID, N: 1})
				}
			}
			if !p.Exhausted {
				exhaust = append(exhaust, engine.ExhaustEntity{ID: p.ID})
			}
			for _, aid := range p.Allies {
				if a := g.Allies[aid]; a != nil && !a.Exhausted {
					exhaust = append(exhaust, engine.ExhaustEntity{ID: aid})
				}
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Security Cameras — exhaust your characters or raise the Alert Level?",
				engine.Choice{Label: "Exhaust your characters", Kind: engine.ChoicePass}.Msgs(exhaust...),
				engine.Choice{Label: "Raise the Alert Level", Kind: engine.ChoicePass}.Msgs(raise...),
			)}}
		},
	})

	registerBatrocBrigade()
}

func registerBatrocBrigade() {
	// 50098 Machete: shuffles back into the encounter deck when defeated.
	engine.RegisterBehavior("50098", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionDefeated)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			g.Logf("Machete slips away into the encounter deck")
			return []engine.Message{engine.ShuffleMinionIntoDeck{MinionID: m.MinionID}}
		},
	})

	// 50099 Rapido: 1 damage to each character you control on reveal and
	// on boost.
	engine.RegisterBehavior("50099", &engine.Behavior{
		OnPlay: rapidoSplash,
		Boost:  rapidoSplashBoost,
	})

	// 50100 Zaran: ATK from tucked cards not modeled; entry tuck is
	// approximated by milling the player once.
	engine.RegisterBehavior("50100", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil {
				return nil
			}
			p := g.Player(mn.EngagedWith)
			if p == nil {
				return nil
			}
			if len(p.Deck) > 0 {
				c := p.Deck[0]
				p.Deck = p.Deck[1:]
				p.Discard = append(p.Discard, c)
				mn.AttackVal++
				g.Logf("Zaran arms himself with %s (+1 ATK)", c.Def().Name)
			}
			return nil
		},
	})

	// 50101 Batroc's Brigade: minions gain toughness while it stands.
	engine.RegisterBehavior("50101", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var out []engine.Message
			for _, id := range cardutil.SortedEnemyIDs(g) {
				out = append(out, engine.ToughEntity{Target: id})
			}
			return out
		},
	})

	// 50102 Soldiers of Fortune: pay 3 or a mercenary minion joins.
	engine.RegisterBehavior("50102", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			reveal := engine.RevealNextEncounter{Player: p.ID}
			for i, c := range g.EncounterDeck {
				if c.Def().Type == "minion" && c.Def().HasTrait("mercenary") {
					card := c
					g.EncounterDeck = append(g.EncounterDeck[:i:i], g.EncounterDeck[i+1:]...)
					def := card.Def()
					mn := &engine.Minion{
						ID: g.NextEntityID(engine.KindMinion), Code: card.Code,
						MaxHP:     intValue(def.HP, 1),
						AttackVal: intValue(def.Attack, 0), SchemeVal: intValue(def.Scheme, 0),
						EngagedWith: p.ID,
					}
					g.Minions[mn.ID] = mn
					return []engine.Message{engine.MinionEntersPlay{MinionID: mn.ID, Player: p.ID}}
				}
			}
			return []engine.Message{reveal}
		},
	})
}

func rapidoSplash(g *engine.Game, e engine.Entity) []engine.Message {
	mn := g.Minions[e.EID()]
	if mn == nil {
		return nil
	}
	p := g.Player(mn.EngagedWith)
	if p == nil {
		return nil
	}
	var out []engine.Message
	out = append(out, engine.DamageEntity{Target: p.ID, Damage: 1, Source: mn.ID})
	for _, aid := range p.Allies {
		out = append(out, engine.DamageEntity{Target: aid, Damage: 1, Source: mn.ID})
	}
	return out
}

func rapidoSplashBoost(g *engine.Game, card engine.Card) []engine.Message {
	// Boost rider: 1 damage to each character the revealing player
	// controls (approximated to the first player).
	if len(g.Players) == 0 {
		return nil
	}
	p := g.Players[0]
	var out []engine.Message
	out = append(out, engine.DamageEntity{Target: p.ID, Damage: 1, Source: engine.EntityID("")})
	for _, aid := range p.Allies {
		out = append(out, engine.DamageEntity{Target: aid, Damage: 1, Source: engine.EntityID("")})
	}
	return out
}
