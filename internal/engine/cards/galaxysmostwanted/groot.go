package galaxysmostwanted

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// registerGrootCards installs Groot's signature deck (16002–16011) plus his
// basic reprints, obligation and nemesis set (16012–16028). Growth counters
// live on the identity (Player.GrowthCounters, max 10); they also feed the
// identity damage prevention registered in galaxysmostwanted.go.
func registerGrootCards() {
	// 16002 Fruition: place 2 growth counters (max 10).
	engine.RegisterBehavior("16002", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			addGrowth(p, 2)
			return nil
		},
	})

	// 16003 "I am Groot": thwart with growth counters.
	engine.RegisterBehavior("16003", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			n := p.GrowthCounters
			return cardutil.ChooseScheme(engine.Tf("c.chooseAScheme", "I am Groot"), func(g *engine.Game, s engine.Entity) int { return n })(g, e)
		},
	})

	// 16004 "I. AM. GROOT!": damage equal to growth counters.
	engine.RegisterBehavior("16004", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			n := p.GrowthCounters
			return cardutil.ChooseEnemy(engine.Tf("c.iAmGrootChooseAnEnemy"),
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return n, nil })(g, e)
		},
	})

	// 16005 Root Stomp: 5 damage; growth counter on the kill.
	engine.RegisterBehavior("16005", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy(engine.Tf("c.rootStompChooseAnEnemy"),
			func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) {
				// Defeat is observable at pick time via remaining HP.
				if entityHP(tgt) <= 5 {
					if p := g.Player(g.ActiveTurn); p != nil && p.HeroCode[:5] == "16001" {
						addGrowth(p, 1)
					}
				}
				return 5, nil
			}),
	})

	// 16006 "We Are Groot": remove up to 4 growth counters → that many
	// friendly characters gain tough.
	engine.RegisterBehavior("16006", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			max := min(4, p.GrowthCounters)
			if max <= 0 {
				return nil
			}
			var chars []engine.EntityID
			var labels []string
			chars = append(chars, p.ID)
			labels = append(labels, p.Name)
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil {
					chars = append(chars, id)
					labels = append(labels, a.EDef().Name)
				}
			}
			var picks []engine.Choice
			for k := 1; k <= min(max, len(chars)); k++ {
				var subs []engine.Choice
				for i, id := range chars {
					subs = append(subs, engine.Choice{Label: engine.S(labels[i]), Kind: engine.ChoiceTarget, SourceID: id}.
						Msgs(engine.ToughEntity{Target: id}))
				}
				picks = append(picks, engine.Choice{
					ID: fmt.Sprintf("k%d", k), Label: engine.Tf("c.removeCounterS", k),
					Kind: engine.ChoiceLabel,
				}.WithThen(engine.AskN(engine.Tf("c.giveToughToCharacterS", k), k, subs...)))
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.weAreGrootRemoveHowManyGrowthCounters"), picks...)}}
		},
	})

	// 16007 Fertile Ground: alter-ego exhaust → +1 growth and draw 1.
	engine.RegisterBehavior("16007", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustFertileGround1GrowthCounterOnGrootDraw1"), Type: engine.AbilityAction,
				Exhaust: true, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					if p := g.Player(e.EOwner()); p != nil {
						addGrowth(p, 1)
					}
					return []engine.Message{engine.DrawCards{Player: e.EOwner(), N: 1}}
				},
			}}
		},
	})

	// 16008 Entangling Vines: interrupt on Groot's basic thwart — 1 growth
	// counter + exhaust → +2 THW (follow-up threat removal).
	engine.RegisterBehavior("16008", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			bt, ok := msg.(engine.BasicThwart)
			if !ok || bt.Player != e.EOwner() {
				return nil
			}
			u := g.Upgrades[e.EID()]
			p := g.Player(e.EOwner())
			if u == nil || u.Exhausted || p == nil || p.GrowthCounters < 1 || !p.IsHero() {
				return nil
			}
			u.Exhausted = true
			p.GrowthCounters--
			g.TLogf("c.entanglingVines2ThwForThisThwart")
			return []engine.Message{engine.ThwartScheme{Scheme: bt.Target, N: 2, Source: e.EID()}}
		},
	})

	// 16009 Lashing Vines: after Groot uses a basic power — 2 growth
	// counters + exhaust → ready Groot.
	engine.RegisterBehavior("16009", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			pid := e.EOwner()
			switch m := msg.(type) {
			case engine.BasicAttack:
				if m.Player != pid {
					return nil
				}
			case engine.BasicThwart:
				if m.Player != pid {
					return nil
				}
			case engine.BasicRecover:
				if m.Player != pid {
					return nil
				}
			default:
				return nil
			}
			u := g.Upgrades[e.EID()]
			p := g.Player(pid)
			if u == nil || u.Exhausted || p == nil || p.GrowthCounters < 2 || !p.IsHero() {
				return nil
			}
			u.Exhausted = true
			p.GrowthCounters -= 2
			g.TLogf("c.lashingVinesGrootReadies")
			return []engine.Message{engine.ReadyEntity{ID: pid}}
		},
	})

	// 16010 Vine Shield: interrupt when Groot defends — 1 growth counter +
	// exhaust → +3 DEF for that attack (BonusDEF is read at defense
	// resolution, so the mutation lands on this very attack).
	engine.RegisterBehavior("16010", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.Defends)
			if !ok || d.Defender != e.EOwner() || d.Undefended {
				return nil
			}
			u := g.Upgrades[e.EID()]
			p := g.Player(e.EOwner())
			if u == nil || u.Exhausted || p == nil || p.GrowthCounters < 1 || !p.IsHero() {
				return nil
			}
			u.Exhausted = true
			p.GrowthCounters--
			p.BonusDEF += 3
			g.TLogf("c.vineShield3DefForThisAttack")
			return nil
		},
	})

	// 16011 Vine Spikes: interrupt on Groot's basic attack — 1 growth
	// counter + exhaust → +2 ATK (follow-up damage).
	engine.RegisterBehavior("16011", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			ba, ok := msg.(engine.BasicAttack)
			if !ok || ba.Player != e.EOwner() {
				return nil
			}
			u := g.Upgrades[e.EID()]
			p := g.Player(e.EOwner())
			if u == nil || u.Exhausted || p == nil || p.GrowthCounters < 1 || !p.IsHero() {
				return nil
			}
			u.Exhausted = true
			p.GrowthCounters--
			g.TLogf("c.vineSpikes2DamageForThisAttack")
			return []engine.Message{engine.DamageEntity{Target: ba.Target, Damage: 2, Source: e.EID()}}
		},
	})

	registerGrootBasics()
	registerGrootObligation()
	registerGrootNemesis()
}

// addGrowth raises Groot's growth counters with the 10 cap.
func addGrowth(p *engine.Player, n int) {
	if p == nil {
		return
	}
	p.GrowthCounters = min(10, p.GrowthCounters+n)
}

// entityHP reads remaining hit points from enemy entities.
func entityHP(e engine.Entity) int {
	switch t := e.(type) {
	case *engine.Villain:
		return t.HP()
	case *engine.Minion:
		return t.HP()
	}
	return 99
}

// registerGrootBasics installs Groot's aspect reprints (16012–16018).
func registerGrootBasics() {
	// 16012 Starhawk: when damaged exactly to death, return to hand.
	engine.RegisterBehavior("16012", &engine.Behavior{
		AllyDefeatInterrupt: func(g *engine.Game, a *engine.Ally, destroy func()) []engine.Message {
			// The engine routes lethal damage here; exact-equality can't
			// be re-derived, so every defeat offers the save.
			p := g.Player(a.Owner)
			if p == nil {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(
				engine.Tf("c.starhawkReturnHimToYourHandInsteadOfDiscarding"),
				engine.Choice{ID: "hand", Label: engine.Tf("c.returnStarhawkToHand"), Kind: engine.ChoiceLabel}.
					Msgs(engine.ReturnControlled{Player: p.ID, ID: a.ID}),
				engine.Choice{ID: "die", Label: engine.Tf("c.letStarhawkBeDefeated"), Kind: engine.ChoiceLabel}.
					Msgs(engine.AllyDestroyed{AllyID: a.ID}),
			)}}
		},
	})

	// 16013 Desperate Defense: alias She-Hulk's 09015.
	if b := engine.LookupBehavior("09015"); b != nil {
		engine.RegisterBehavior("16013", b)
	}

	// 16014 Fighting Fit: 2 damage, 5 at full health.
	engine.RegisterBehavior("16014", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			n := 2
			if p != nil && p.Damage == 0 {
				n = 5
			}
			return cardutil.ChooseEnemy(engine.Tf("c.fightingFitChooseTheVillain"),
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return n, nil })(g, e)
		},
	})

	// 16015 The Power of Protection: doubled-resource bonus is data-driven
	// (payment validation reads the card name).
	engine.RegisterBehavior("16015", &engine.Behavior{})

	// 16016 Dauntless: retaliate 1 while at full health.
	engine.RegisterBehavior("16016", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			if p.Damage == 0 && p.IsHero() {
				return engine.StatBonus{Retaliate: 1}
			}
			return engine.StatBonus{}
		},
	})

	// 16017 Hard to Ignore: after defending with no damage taken, exhaust
	// → remove 1 threat from the main scheme.
	engine.RegisterBehavior("16017", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowDefended)
			if !ok || w.Defender != e.EOwner() || w.DamageTaken != 0 {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if u == nil || u.Exhausted || g.MainScheme == nil {
				return nil
			}
			u.Exhausted = true
			g.TLogf("c.hardToIgnore1ThreatRemovedFromTheMainScheme")
			return []engine.Message{engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 1, Source: e.EID()}}
		},
	})

	// 16018 Indomitable: alias core 01082.
	if b := engine.LookupBehavior("01082"); b != nil {
		engine.RegisterBehavior("16018", b)
	}

	// 16021–16023 basic resources.
	engine.RegisterBehavior("16021", &engine.Behavior{})
	engine.RegisterBehavior("16022", &engine.Behavior{})
	engine.RegisterBehavior("16023", &engine.Behavior{})
}

// registerGrootObligation installs Wilt (16025).
func registerGrootObligation() {
	engine.RegisterBehavior("16025", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			exhaust := engine.Choice{
				ID: "exhaust", Label: engine.Tf("c.exhaustYourAlterEgoRemoveWiltFromTheGame"), Kind: engine.ChoiceLabel,
			}.Msgs(
				engine.ExhaustEntity{ID: p.ID},
				engine.ObligationResolve{Player: p.ID, Card: card, Remove: true})
			var strip []engine.Choice
			strip = append(strip, exhaust)
			if p.GrowthCounters >= 3 {
				strip = append(strip, engine.Choice{
					ID: "growth", Label: engine.Tf("c.remove3GrowthCountersFromGroot"), Kind: engine.ChoiceLabel,
				}.Msgs(engine.AddProgressCounters{Player: p.ID, N: -3},
					engine.ObligationResolve{Player: p.ID, Card: card}))
			} else {
				strip = append(strip, engine.Choice{
					ID: "surge", Label: engine.Tf("c.noGrowthCountersToRemoveSurge"), Kind: engine.ChoiceLabel, Disabled: true,
				})
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.wiltChooseAnOption"), strip...)}}
		},
	})
}

// registerGrootNemesis installs Blazing Inferno, Furnax and Fan the Flames
// (16026–16028).
func registerGrootNemesis() {
	// 16026 Blazing Inferno: after the villain phase begins, 2 indirect
	// damage to each player.
	engine.RegisterBehavior("16026", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			bp, ok := msg.(engine.BeginPhase)
			if !ok || bp.Phase != engine.PhaseVillain {
				return nil
			}
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, engine.IndirectDamage{Player: p.ID, N: 2})
			}
			return msgs
		},
	})

	// 16027 Furnax: after he activates, 2 indirect damage to each player.
	engine.RegisterBehavior("16027", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			ma, ok := msg.(engine.MinionActivates)
			if !ok || ma.MinionID != e.EID() {
				return nil
			}
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, engine.IndirectDamage{Player: p.ID, N: 2})
			}
			return msgs
		},
	})

	// 16028 Fan the Flames: 2 indirect damage (+1 per inferno/Furnax).
	engine.RegisterBehavior("16028", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			n := 2
			for id := range g.SideSchemes {
				if s := g.SideSchemes[id]; s != nil && s.Code[:5] == "16026" {
					n++
				}
			}
			for _, m := range g.Minions {
				if m != nil && m.Code[:5] == "16027" {
					n++
				}
			}
			g.Delete(t.ID)
			return []engine.Message{engine.IndirectDamage{Player: p.ID, N: n}}
		},
	})
}
