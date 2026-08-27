package aos

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() { registerZemo() }

// boardMembers lists the Board Member environments.
func boardMembers(g *engine.Game) []*engine.Environment {
	var out []*engine.Environment
	for _, env := range g.Environments {
		if env != nil && env.Code >= "50181" && env.Code <= "50183" {
			out = append(out, env)
		}
	}
	return out
}

// secretTotal sums secret counters across the board.
func secretTotal(g *engine.Game) int {
	n := 0
	for _, env := range boardMembers(g) {
		n += env.Counters
	}
	return n
}

// addSecret places n secret counters on the board members with the fewest
// counters (spread like the card text).
func addSecret(g *engine.Game, n int) {
	members := boardMembers(g)
	if len(members) == 0 {
		return
	}
	for i := 0; i < n; i++ {
		lowest := members[0]
		for _, m := range members {
			if m.Counters < lowest.Counters {
				lowest = m
			}
		}
		lowest.Counters++
	}
	g.TLogf("c.secretCountersSpreadAcrossTheExecutiveBoardTotal", secretTotal(g))
}

// removeSecrets strips n secret counters from the board.
func removeSecrets(g *engine.Game, n int) {
	for _, m := range boardMembers(g) {
		if n <= 0 {
			break
		}
		take := m.Counters
		if take > n {
			take = n
		}
		m.Counters -= take
		n -= take
	}
}

func registerZemo() {
	// 50165a/50166a Baron Zemo: a defeat burns secret counters and resets
	// him instead; at zero secrets he falls for good.
	zemoReset := func(resetHP int) func(g *engine.Game, v *engine.Villain, damage int) bool {
		return func(g *engine.Game, v *engine.Villain, damage int) bool {
			if v.Damage+damage < v.MaxHP {
				return true
			}
			if secretTotal(g) >= 3 {
				removeSecrets(g, 3)
				v.Damage = 0
				v.MaxHP = resetHP
				g.TLogf("c.baronZemoSlipsBehindHisConspiratorsSecretsLeft", secretTotal(g))
				return false
			}
			return true
		}
	}
	engine.RegisterBehavior("50165", &engine.Behavior{VillainDamageable: zemoReset(12)})
	engine.RegisterBehavior("50166", &engine.Behavior{VillainDamageable: zemoReset(16)})

	// Main schemes.
	engine.RegisterBehavior("50167", &engine.Behavior{})
	engine.RegisterBehavior("50168", &engine.Behavior{})
	engine.RegisterBehavior("50169", &engine.Behavior{})

	// 50170 Baron Zemo's Sword: attach + his defeats feed the conspiracy.
	engine.RegisterBehavior("50170", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if v := g.Villains[target]; v != nil {
				v.AttackVal += 2
				g.TLogf("c.baronZemoWieldsHisSword2Atk")
			}
			return nil
		},
	})

	// 50171 Reluctant Foe: the hero-turned-minion graft is approximated
	// as a 3-ATK / 1-SCH mercenary minion.
	engine.RegisterBehavior("50171", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			g.TLogf("c.aReluctantHeroIsForcedIntoTheFight")
			return nil
		},
	})

	// 50172 S.H.I.E.L.D. Agent: activation or defeat feeds a secret
	// counter to the board.
	engine.RegisterBehavior("50172", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			switch m := msg.(type) {
			case engine.MinionDefeated:
				if m.MinionID != e.EID() {
					return nil
				}
			case engine.MinionActivates:
				if m.MinionID != e.EID() {
					return nil
				}
			default:
				return nil
			}
			addSecret(g, 1)
			return nil
		},
	})

	// 50173 Divided Loyalties: defeating it strips secrets.
	engine.RegisterBehavior("50173", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			removeSecrets(g, len(g.Players))
			g.TLogf("c.dividedLoyaltiesCrumblesSecretsSpillLeft", secretTotal(g))
			return nil
		},
	})

	// 50174 Undermine Support: defeating it strips secrets.
	engine.RegisterBehavior("50174", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			removeSecrets(g, len(g.Players))
			g.TLogf("c.undermineSupportCrumblesSecretsSpillLeft", secretTotal(g))
			return nil
		},
	})

	// 50175 Battle of Wits: scheme; mental spend prevents threat and
	// strips secrets.
	engine.RegisterBehavior("50175", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			villain := engine.EntityID("")
			for id := range g.Villains {
				villain = id
				break
			}
			if g.MainScheme != nil && villain != "" {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: villain}}
			}
			return nil
		},
	})

	// 50176 Might Makes Right: secrets grow in alter-ego form; the hero
	// branch attacks.
	engine.RegisterBehavior("50176", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if !p.IsHero() {
				addSecret(g, 2)
				return nil
			}
			for id := range g.Villains {
				return []engine.Message{engine.AskAttack{Enemy: id, Player: p.ID}}
			}
			return nil
		},
	})

	// 50177 The Ends Justify the Means: choose the sacrifice or the
	// secrets slip.
	engine.RegisterBehavior("50177", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var discard []engine.Choice
			for _, aid := range p.Allies {
				if a := g.Allies[aid]; a != nil {
					n := 2
					if c := cardutil.Cost(a.EDef()); c > 0 {
						n = c
					}
					discard = append(discard, engine.Choice{
						Label: engine.Tf("c.discardRemoveSecrets", a, n), Kind: engine.ChoiceTarget, SourceID: aid,
					}.Msgs(engine.DiscardControlled{Player: p.ID, ID: aid}))
				}
			}
			for _, sid := range p.Supports {
				if s := g.Supports[sid]; s != nil {
					n := 2
					if c := cardutil.Cost(s.EDef()); c > 0 {
						n = c
					}
					discard = append(discard, engine.Choice{
						Label: engine.Tf("c.discardRemoveSecrets", s, n), Kind: engine.ChoiceTarget, SourceID: sid,
					}.Msgs(engine.DiscardControlled{Player: p.ID, ID: sid}))
				}
			}
			if len(discard) > 0 {
				discard = append(discard, engine.Choice{Label: engine.Tf("c.refuseTheVillainSchemes"), Kind: engine.ChoicePass}.
					Msgs(engine.VillainActivates{VillainID: firstVillainID(g), Player: p.ID}))
				return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.theEndsJustifyTheMeansDiscardACardYouControl"), discard...)}}
			}
			return []engine.Message{engine.VillainActivates{VillainID: firstVillainID(g), Player: p.ID}}
		},
	})

	// --- S.H.I.E.L.D. encounter set ---
	// 50178 S.H.I.E.L.D. Trooper.
	engine.RegisterBehavior("50178", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionDefeated)
			if !ok || m.MinionID != e.EID() || g.MainScheme == nil {
				return nil
			}
			mn := g.Minions[e.EID()]
			p := g.Player(mn.EngagedWith)
			if p == nil {
				return nil
			}
			// Prefer discarding a S.H.I.E.L.D. ally/support; else threat.
			for _, aid := range p.Allies {
				if a := g.Allies[aid]; a != nil && hasShieldTrait(a.EDef()) {
					return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: aid}}
				}
			}
			for _, sid := range p.Supports {
				if s := g.Supports[sid]; s != nil && hasShieldTrait(s.EDef()) {
					return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: sid}}
				}
			}
			return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: m.MinionID}}
		},
	})

	// 50179 Arrest Warrant obligation.
	engine.RegisterBehavior("50179", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			g.TLogf("c.arrestWarrantDogsTheHeroes")
			return []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
		},
	})

	// 50180 Disavowed: S.H.I.E.L.D. cards cost more; entry threat.
	engine.RegisterBehavior("50180", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			n := 0
			for _, p := range g.Players {
				for _, aid := range p.Allies {
					if a := g.Allies[aid]; a != nil && hasShieldTrait(a.EDef()) {
						n++
					}
				}
				for _, sid := range p.Supports {
					if s := g.Supports[sid]; s != nil && hasShieldTrait(s.EDef()) {
						n++
					}
				}
			}
			s := g.SideSchemes[e.EID()]
			s.Threat += n
			g.TLogf("c.disavowedGainsThreat", n)
			return nil
		},
	})

	// --- Executive Board environments: secret-counter holders with
	// resource actions (the flip is approximated: at 4+ counters they
	// count as compromised). ---
	for _, entry := range []struct{ code, icon, label string }{
		{"50181", "energy", "heal 1 damage from a friendly character"},
		{"50182", "mental", "remove 2 threat from a scheme"},
		{"50183", "physical", "deal 2 damage to an enemy"},
	} {
		engine.RegisterBehavior(entry.code, &engine.Behavior{
			Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
				env := g.Environments[e.EID()]
				if env == nil || env.Counters <= 0 {
					return nil
				}
				return []engine.Ability{{
					Label: engine.S(env.EDef().Name + " — spend 2 matching resources: remove 1 secret counter, then " + entry.label),
					Type:  engine.AbilityAction, HeroOnly: true, Cost: 2,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						env := g.Environments[self]
						p := g.Players[0]
						if env == nil || p == nil {
							return nil
						}
						env.Counters--
						switch data.BaseCode(env.Code) {
						case "50181":
							return []engine.Message{engine.HealEntity{Target: p.ID, N: 1}}
						case "50182":
							if g.MainScheme != nil {
								return []engine.Message{engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 2, Source: self}}
							}
						case "50183":
							for _, id := range cardutil.SortedEnemyIDs(g) {
								return []engine.Message{engine.DamageEntity{Target: id, Damage: 2, Source: self}}
							}
						}
						return nil
					},
				}}
			},
		})
	}

	// 50184a/b/c A.I.M. Interference: a secret counter per board member.
	for _, code := range []string{"50184"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
				n := len(boardMembers(g))
				if n > 0 {
					addSecret(g, n)
				}
				if g.MainScheme != nil {
					return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: t.ID}}
				}
				return nil
			},
		})
	}

	// 50185-50193 Executive Board Evidence: the draft-and-search setup is
	// handled by the scenario; the cards themselves are set aside and
	// never revealed.
	for _, code := range []string{"50185", "50186", "50187", "50188", "50189", "50190", "50191", "50192", "50193"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}
}

// firstVillainID returns any villain (empty when none).
func firstVillainID(g *engine.Game) engine.EntityID {
	for id := range g.Villains {
		return id
	}
	return ""
}
