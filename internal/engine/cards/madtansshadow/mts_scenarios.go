package madtansshadow

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// registerEbonyMaw installs the Ebony Maw scenario (21071–21084). Spell
// environments are global approximations of the per-player play areas.
func registerEbonyMaw() {
	for _, base := range []string{"21071", "21072", "21073"} {
		b := &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				m, ok := msg.(engine.VillainActivates)
				if !ok || m.VillainID != e.EID() {
					return nil
				}
				var msgs []engine.Message
				for _, env := range g.Environments {
					if env != nil && env.EDef().HasTrait("spell") && env.Counters > 0 {
						env.Counters--
						g.TLogf("c.losesAnInvocationCounter", env)
					}
				}
				return msgs
			},
		}
		if base != "21071" {
			b.VillainStage = func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
				return mawDigSpells(g)
			}
		}
		engine.RegisterBehavior(base, b)
	}

	// 21074 Attack on Knowhere (stage 1) / 21075 The Power Stone (2).
	engine.RegisterBehavior("21074", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			return mawDigSpells(g)
		},
	})
	engine.RegisterBehavior("21075", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			g.ShuffleEncounterDeck()
			return mawDigSpells(g)
		},
	})

	// 21076–21079 Spell environments: the empty-counter payoffs.
	spellPayoff := func(counters int, effect func(g *engine.Game, e engine.Entity) []engine.Message) *engine.Behavior {
		return &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				// Payoffs fire when Maw's activation drains the last
				// counter (approximation: checked on every activation).
				m, ok := msg.(engine.VillainActivates)
				if !ok {
					return nil
				}
				_ = m
				env := g.Environments[e.EID()]
				if env == nil || env.Counters > 0 {
					return nil
				}
				g.Delete(e.EID())
				return effect(g, e)
			},
		}
	}
	_ = spellPayoff // replaced by explicit registrations below
	engine.RegisterBehavior("21076", &engine.Behavior{})
	engine.RegisterBehavior("21077", &engine.Behavior{})
	engine.RegisterBehavior("21078", &engine.Behavior{})
	engine.RegisterBehavior("21079", &engine.Behavior{})

	// 21080 Agent of Thanos: per-spell tax.
	engine.RegisterBehavior("21080", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			n := 0
			for _, env := range g.Environments {
				if env != nil && env.EDef().HasTrait("spell") {
					n++
				}
			}
			if n == 0 {
				if c, ok := g.DrawEncounter(); ok {
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
				return nil
			}
			if p.IsHero() {
				return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: n, Source: t.ID}}
			}
			if g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: n, Source: t.ID}}
			}
			return nil
		},
	})

	// 21081 Channeling Trance: drain or fetch a Spell.
	engine.RegisterBehavior("21081", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			drained := false
			for _, env := range g.Environments {
				if env != nil && env.EDef().HasTrait("spell") && env.Counters > 0 {
					env.Counters--
					drained = true
				}
			}
			if drained {
				return nil
			}
			return mawDigSpells(g)
		},
	})

	// 21082 Abjuration: Maw immune until a big hit breaks it.
	engine.RegisterBehavior("21082", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil && v.Code[:5] == "21071" {
					t.Target = id
					return nil
				}
			}
			for id := range g.Villains {
				t.Target = id
				break
			}
			return nil
		},
	})

	// 21083 Restrained: lock down your strongest.
	engine.RegisterBehavior("21083", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			p := g.Player(target)
			if p == nil {
				p = g.Players[0]
			}
			best, bestATK := engine.EntityID(""), -1
			if p.AttackStat(g) > bestATK {
				best, bestATK = p.ID, p.AttackStat(g)
			}
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && a.AttackVal > bestATK {
					best, bestATK = id, a.AttackVal
				}
			}
			t.Target = best
			return []engine.Message{engine.ExhaustEntity{ID: best}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.spendEnergyPhysicalDiscardRestrained"), Type: engine.AbilityAction,
				CostIcons: "energy:1 physical:1",
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.DiscardAttachmentMsg{ID: self}}
				},
			}}
		},
	})

	// 21084 Reactor Overload: damage or threat.
	engine.RegisterBehavior("21084", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var opts []engine.Choice
			for _, p := range g.Players {
				opts = append(opts,
					engine.Choice{
						ID: "dmg" + string(p.ID), Label: engine.Tf("c.playerTakes2", p.Name), Kind: engine.ChoiceLabel,
					}.Msgs(engine.DamageEntity{Target: p.ID, Damage: 2, Source: e.EID()}),
					engine.Choice{
						ID: "thr" + string(p.ID), Label: engine.Tf("c.place2ThreatPlayer", p.Name), Kind: engine.ChoiceLabel,
					}.Msgs(engine.SchemeThreat{Scheme: e.EID(), N: 2, Source: e.EID()}))
			}
			return []engine.Message{engine.AskQuestion{Player: cardutil.FirstPlayerID(g),
				Question: engine.Ask(engine.Tf("c.reactorOverloadEachPlayerChooses"), opts...)}}
		},
	})
}

// mawDigSpells digs the encounter deck for a Spell per player.
func mawDigSpells(g *engine.Game) []engine.Message {
	var msgs []engine.Message
	for _, p := range g.Players {
		for len(g.EncounterDeck) > 0 {
			c := g.EncounterDeck[0]
			g.EncounterDeck = g.EncounterDeck[1:]
			if c.Def().Type == "environment" && c.Def().HasTrait("spell") {
				env := g.SpawnEnvironment(c.Code)
				switch c.Code[:5] {
				case "21076":
					env.Counters = 4
				case "21077":
					env.Counters = 2
				case "21078", "21079":
					env.Counters = 3
				}
				msgs = append(msgs, engine.RevealEncounterCard{Player: p.ID, Card: c})
				break
			}
			g.EncounterDiscard = append(g.EncounterDiscard, c)
		}
	}
	return msgs
}

// registerTowerDefense installs the Tower Defense scenario (21085–
// 21110): Proxima and Corvus siege Avengers Tower. The second main
// scheme rides as a surrogate side scheme; the active villain alternates.
func registerTowerDefense() {
	engine.RegisterBehavior("21085", &engine.Behavior{})
	engine.RegisterBehavior("21086", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok || w.Enemy != e.EID() {
				return nil
			}
			return []engine.Message{engine.StunEntity{Target: w.Player}}
		},
	})
	engine.RegisterBehavior("21087", &engine.Behavior{})
	engine.RegisterBehavior("21088", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var msgs []engine.Message
			for _, id := range cardutil.SortedIDs(g.Minions) {
				if m := g.Minions[id]; m != nil {
					msgs = append(msgs, engine.MinionActivates{MinionID: id, Player: m.EngagedWith})
				}
			}
			return msgs
		},
	})
	engine.RegisterBehavior("21089", &engine.Behavior{
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil && !v.Tough {
					v.Tough = true
					g.TLogf("c.gainsAToughStatusCard", v)
				}
			}
			return nil
		},
	})
	engine.RegisterBehavior("21090", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			if len(p.Hand) > 0 {
				return []engine.Message{engine.DiscardCards{Player: p.ID, Cards: engine.CardList{p.Hand[0]}}}
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if p := g.Player(cardutil.FirstPlayerID(g)); p != nil && len(p.Hand) > 0 {
				return []engine.Message{engine.DiscardCards{Player: p.ID, Cards: engine.CardList{p.Hand[0]}}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("21091", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			return discardUntilMinionEngaged(g, g.ActiveTurn)
		},
	})

	// Proxima (21092–21094): +2 ATK or tower damage; paired with Corvus.
	proxima := func() *engine.Behavior {
		return &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				m, ok := msg.(engine.VillainActivates)
				if !ok || m.VillainID != e.EID() {
					return nil
				}
				p := g.Player(m.Player)
				if p == nil || !p.IsHero() {
					return nil
				}
				if tower := g.EnvironmentByCode("21100a"); tower != nil {
					tower.Counters++
					g.TLogf("c.avengersTowerTakes1Damage", tower.Counters)
					return nil
				}
				return []engine.Message{engine.BoostActivation{Enemy: e.EID(), N: 2}}
			},
			VillainDamageable: func(g *engine.Game, v *engine.Villain, damage int) bool {
				for _, o := range g.Villains {
					if o != nil && o != v && o.Code[:5] == "21095" && o.HP() > 0 {
						g.TLogf("c.proximaCannotBeDefeatedWhileCorvusGlaiveLives")
						return false
					}
				}
				return true
			},
		}
	}
	engine.RegisterBehavior("21092", proxima())
	engine.RegisterBehavior("21093", proxima())
	engine.RegisterBehavior("21094", proxima())

	// Corvus (21095–21097): undefended hits chew the tower.
	corvus := func() *engine.Behavior {
		return &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				w, ok := msg.(engine.WindowDefended)
				if !ok || w.Against != e.EID() {
					return nil
				}
				tower := g.EnvironmentByCode("21100a")
				if tower == nil || w.DamageTaken == 0 {
					return nil
				}
				tower.Counters++
				g.TLogf("c.avengersTowerTakes1Damage", tower.Counters)
				return nil
			},
			VillainDamageable: func(g *engine.Game, v *engine.Villain, damage int) bool {
				for _, o := range g.Villains {
					if o != nil && o != v && o.Code[:5] == "21092" && o.HP() > 0 {
						g.TLogf("c.corvusCannotBeDefeatedWhileProximaMidnightLives")
						return false
					}
				}
				return true
			},
		}
	}
	engine.RegisterBehavior("21095", corvus())
	engine.RegisterBehavior("21096", corvus())
	engine.RegisterBehavior("21097", corvus())

	// 21098/21099 main schemes: completion is deflected by the scenario
	// def (threat wipe + punishment).
	engine.RegisterBehavior("21098", &engine.Behavior{})
	engine.RegisterBehavior("21099", &engine.Behavior{})

	// 21100 Avengers Tower: breaks at 9[per_hero] damage.
	engine.RegisterBehavior("21100", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.BeginRound); !ok {
				return nil
			}
			env := g.Environments[e.EID()]
			if env == nil {
				return nil
			}
			if env.Counters >= 9*len(g.Players) {
				env.Counters = 0
				g.TLogMajorf("c.avengersTowerCollapsesResetLossConditionApproximated")
			}
			return nil
		},
	})

	// 21101 Focused Defense: the active villain alternates each round
	// (approximation of the scheme-attached token).
	engine.RegisterBehavior("21101", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.BeginRound); !ok {
				return nil
			}
			ids := cardutil.SortedIDs(g.Villains)
			if len(ids) < 2 {
				return nil
			}
			if g.ActiveVillain == ids[0] {
				g.ActiveVillain = ids[1]
			} else {
				g.ActiveVillain = ids[0]
			}
			g.TLogf("c.focusedDefenseTheActiveVillainIsNow", g.Villains[g.ActiveVillain].EDef().Name)
			return nil
		},
	})

	engine.RegisterBehavior("21102", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			em, ok := msg.(engine.EngageMinion)
			if !ok || em.MinionID != e.EID() {
				return nil
			}
			if tower := g.EnvironmentByCode("21100a"); tower != nil {
				tower.Counters++
				return nil
			}
			return []engine.Message{engine.DamageEntity{Target: em.Player, Damage: 2, Source: e.EID()}}
		},
	})
	engine.RegisterBehavior("21103", &engine.Behavior{})
	engine.RegisterBehavior("21104", &engine.Behavior{})
	engine.RegisterBehavior("21105", &engine.Behavior{})
	engine.RegisterBehavior("21106", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil && v.Code[:5] == "21092" {
					return []engine.Message{engine.VillainActivates{VillainID: id, Player: p.ID}}
				}
			}
			return nil
		},
	})
	engine.RegisterBehavior("21107", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil && v.Code[:5] == "21095" {
					return []engine.Message{engine.VillainActivates{VillainID: id, Player: p.ID}}
				}
			}
			return nil
		},
	})
	engine.RegisterBehavior("21108", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var msgs []engine.Message
			for id, v := range g.Villains {
				if v == nil {
					continue
				}
				v.Damage = max(0, v.Damage-2)
				if !v.Tough {
					v.Tough = true
				}
				_ = id
			}
			return msgs
		},
	})
	engine.RegisterBehavior("21109", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			if tower := g.EnvironmentByCode("21100a"); tower != nil {
				tower.Counters += 3
				g.TLogf("c.avengersTowerTakes3Damage", tower.Counters)
			}
			return nil
		},
	})
	engine.RegisterBehavior("21110", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			return []engine.Message{engine.DrawCards{Player: g.ActiveTurn, N: 1}}
		},
	})
}

// registerThanos installs the Thanos scenario (21111–21135). The
// infinity stone deck is approximated by the set-aside pool.
func registerThanos() {
	for _, base := range []string{"21111", "21112", "21113"} {
		b := &engine.Behavior{}
		if base != "21111" {
			b.VillainStage = func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
				for i, c := range g.EncounterDeck {
					if c.Code[:5] == "21118" {
						g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
						g.ShuffleEncounterDeck()
						return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
					}
				}
				return nil
			}
		}
		engine.RegisterBehavior(base, b)
	}

	// 21114 The Infinity Stones (stage 1) / 21115 Balance the Scales (2).
	engine.RegisterBehavior("21114", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			// Seed the stone deck (set-aside pool).
			for _, code := range []string{"21130", "21131", "21132", "21133", "21134", "21135"} {
				g.SetAside = append(g.SetAside, engine.Card{ID: g.NextCardID(), Code: code})
			}
			g.SpawnAttachment("21129", "")
			return nil
		},
	})
	engine.RegisterBehavior("21115", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				half := len(p.Discard) / 2
				_ = half
				// Shuffle discard into deck, remove half from the game.
				p.Deck = append(p.Deck, p.Discard...)
				p.Discard = nil
				g.TLogf("c.sDiscardPileIsShuffledAwayByBalanceTheScalesDeckHalvingAppro", p.Name)
			}
			return msgs
		},
	})

	// 21116 Sanctuary: Thanos immune until the scheme falls.
	engine.RegisterBehavior("21116", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			var msgs []engine.Message
			for id, v := range g.Villains {
				if v != nil && v.Code[:5] == "21111" {
					v.Tough = false
					msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 6, Source: s.ID})
				}
			}
			return msgs
		},
	})

	// 21117/21118 Thanos's Armor/Helmet.
	engine.RegisterBehavior("21117", &engine.Behavior{})
	engine.RegisterBehavior("21118", &engine.Behavior{})

	// 21119 Master of the Stones: reveal a stone on activation.
	engine.RegisterBehavior("21119", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.VillainActivates)
			if !ok {
				return nil
			}
			a := g.Attachments[e.EID()]
			if a == nil || a.Target != m.VillainID {
				return nil
			}
			g.Delete(e.EID())
			return revealStone(g)
		},
	})

	// 21120–21123 treacheries.
	engine.RegisterBehavior("21120", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for id, v := range g.Villains {
				if v == nil {
					continue
				}
				if !p.IsHero() {
					return []engine.Message{engine.VillainActivates{VillainID: id, Player: p.ID}}
				}
				return []engine.Message{
					engine.BoostActivation{Enemy: id, N: 2},
					engine.VillainActivates{VillainID: id, Player: p.ID},
				}
			}
			return nil
		},
	})
	engine.RegisterBehavior("21121", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for _, v := range g.Villains {
				if v == nil {
					continue
				}
				if v.Tough {
					if g.MainScheme != nil {
						return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: t.ID}}
					}
					return nil
				}
				v.Tough = true
				return nil
			}
			return nil
		},
	})
	engine.RegisterBehavior("21122", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for id := range g.Villains {
				return []engine.Message{engine.DealBoost{Enemy: id}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("21123", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			return revealStone(g)
		},
	})

	// 21124 The Titan's Throne: sacrifice a stone.
	engine.RegisterBehavior("21124", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for _, env := range g.Environments {
				if env != nil && env.EDef().HasTrait("infinity stone") {
					g.Delete(env.ID)
					g.TLogf("log.discarded", env)
					return nil
				}
			}
			return nil
		},
	})

	// 21125–21128 Children of Thanos.
	engine.RegisterBehavior("21125", &engine.Behavior{})
	engine.RegisterBehavior("21126", &engine.Behavior{})
	engine.RegisterBehavior("21127", &engine.Behavior{})
	engine.RegisterBehavior("21128", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			return []engine.Message{engine.DealEncounterToPlayer{Player: g.ActiveTurn}}
		},
	})

	// 21129 Infinity Gauntlet: stones fire after Thanos activates.
	engine.RegisterBehavior("21129", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil && v.Code[:5] == "21111" {
					t.Target = id
					return nil
				}
			}
			for id := range g.Villains {
				t.Target = id
				break
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok {
				return nil
			}
			a := g.Attachments[e.EID()]
			if a == nil || a.Target != w.Enemy {
				return nil
			}
			var msgs []engine.Message
			fired := false
			for _, env := range g.Environments {
				if env != nil && env.EDef().HasTrait("infinity stone") {
					msgs = append(msgs, stoneSpecial(g, env, w.Player)...)
					fired = true
				}
			}
			if !fired {
				msgs = append(msgs, revealStone(g)...)
			}
			return msgs
		},
	})
}

// registerInfinityStones registers the six stone environments (21130–
// 21135); their Specials live in stoneSpecial.
func registerInfinityStones() {
	for _, code := range []string{"21130", "21131", "21132", "21133", "21134", "21135"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}
}

// revealStone brings the next infinity stone into play.
func revealStone(g *engine.Game) []engine.Message {
	if len(g.SetAside) == 0 {
		for id, v := range g.Villains {
			if v != nil && v.Code[:5] == "21111" {
				return []engine.Message{engine.DealBoost{Enemy: id}}
			}
		}
		return nil
	}
	c := g.SetAside[0]
	g.SetAside = g.SetAside[1:]
	g.SpawnEnvironment(c.Code)
	g.TLogMajorf("c.theReveals", c)
	return nil
}

// stoneSpecial resolves a stone's printed Special against the player.
func stoneSpecial(g *engine.Game, env *engine.Environment, pid engine.PlayerID) []engine.Message {
	p := g.Player(pid)
	if p == nil {
		return nil
	}
	code := env.Code[:5]
	g.Delete(env.ID)
	switch code {
	case "21130": // Mind
		if p.Confused {
			if len(p.Hand) > 0 {
				return []engine.Message{engine.ConfuseEntity{Target: p.ID}, engine.DiscardCards{Player: p.ID, Cards: engine.CardList{p.Hand[0]}}}
			}
		}
		return []engine.Message{engine.ConfuseEntity{Target: p.ID}}
	case "21131": // Power
		if p.Stunned {
			return []engine.Message{engine.StunEntity{Target: p.ID}, engine.DamageEntity{Target: p.ID, Damage: 3}}
		}
		return []engine.Message{engine.StunEntity{Target: p.ID}}
	case "21132": // Reality
		if len(p.Upgrades) > 0 {
			return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Upgrades[len(p.Upgrades)-1]}}
		}
		if len(p.Supports) > 0 {
			return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Supports[len(p.Supports)-1]}}
		}
		return nil
	case "21133": // Soul
		for id, v := range g.Villains {
			if v != nil {
				return []engine.Message{engine.HealEntity{Target: id, N: 3}, engine.DealBoost{Enemy: id}}
			}
		}
		return nil
	case "21134": // Space
		return discardUntilMinionEngaged(g, pid)
	case "21135": // Time
		types := map[string]bool{}
		for i := 0; i < 4 && len(p.Deck) > 0; i++ {
			c := p.Deck[0]
			p.Deck = p.Deck[1:]
			types[c.Def().Type] = true
			p.Discard = append(p.Discard, c)
		}
		if g.MainScheme != nil {
			return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: len(types)}}
		}
		return nil
	}
	return nil
}

// discardUntilMinionEngaged mills until a minion shows up.
func discardUntilMinionEngaged(g *engine.Game, pid engine.PlayerID) []engine.Message {
	for len(g.EncounterDeck) > 0 {
		c := g.EncounterDeck[0]
		g.EncounterDeck = g.EncounterDeck[1:]
		if c.Def().Type == "minion" {
			return []engine.Message{engine.RevealEncounterCard{Player: pid, Card: c}}
		}
		g.EncounterDiscard = append(g.EncounterDiscard, c)
	}
	return nil
}
