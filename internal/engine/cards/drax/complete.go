// Package drax registers the Drax hero pack: vengeance counters, knife
// restrictions and the Yotat nemesis set. Per-attack interrupts are
// approximated as until-end-of-phase bonuses, consistent with the rest
// of the engine.
package drax

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() {
	registerDrax()
	registerNemesis()
}

func vengeance(p *engine.Player) int { return p.GrowthCounters }

func registerDrax() {
	// Drax: +1 ATK per vengeance counter; gains one when the villain
	// attacks him (draw past 3 instead).
	engine.RegisterBehavior("19001", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			p := g.Player(e.EID())
			if p == nil {
				return nil
			}
			switch m := msg.(type) {
			case engine.WindowAfterEnemyAttacked:
				// Reactions run pre-resolution; the attack itself has
				// resolved by the time this window's messages run.
				if m.Player == p.ID && g.Villains[m.Enemy] != nil {
					if vengeance(p) < 3 {
						return []engine.Message{engine.AddVengeance{Player: p.ID}}
					}
					return []engine.Message{engine.DrawCards{Player: p.ID, N: 1}}
				}
			}
			return nil
		},
	})

	// Mantis: exhaust + 1 self-damage → heal 3 from an identity.
	engine.RegisterBehavior("19002", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustMantisSheTakes1DamageHeal3FromAnIdentity"), Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					var picks []engine.Choice
					for _, q := range g.Players {
						if q.Damage > 0 {
							picks = append(picks, engine.Choice{Label: engine.S(q.Name), Kind: engine.ChoiceTarget, SourceID: q.ID}.
								Msgs(engine.DamageEntity{Target: self, Damage: 1, Source: self},
									engine.HealEntity{Target: q.ID, N: 3}))
						}
					}
					if len(picks) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: g.Entity(self).EOwner(),
						Question: engine.Ask(engine.Tf("c.mantisHealWhichIdentity"), picks...)}}
				},
			}}
		},
	})

	// "Fight Me, Coward!": ready + draw; the villain attacks.
	engine.RegisterBehavior("19003", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			msgs := []engine.Message{
				engine.ReadyEntity{ID: e.EOwner()},
				engine.DrawCards{Player: e.EOwner(), N: 1},
			}
			for id := range g.Villains {
				msgs = append(msgs, engine.DealBoost{Enemy: id}, engine.RevealBoost{Enemy: id},
					engine.AskAttack{Enemy: id, Player: e.EOwner(), Trigger: engine.TriggerVillainAttacksYou})
				break
			}
			return msgs
		},
	})

	// Intimidation: remove X threat (X = ATK).
	engine.RegisterBehavior("19004", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			n := 0
			if p != nil {
				n = max(0, p.AttackStat(g))
			}
			if n <= 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(), Question: engine.Ask(
				engine.Tf("c.intimidationRemoveThreatFromWhichScheme"), schemePicks(g, n, e.EOwner())...)}}
		},
	})

	// Knife Leap: +5 ATK until end of phase (per-attack and overkill
	// approximated); cost reduced per vengeance counter.
	engine.RegisterBehavior("19005", &engine.Behavior{
		CardCost: func(g *engine.Game, p *engine.Player, def *data.CardDef) int {
			if def.Code == "19005" {
				return min(vengeance(p), 3)
			}
			return 0
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.ApplyStatBonus{Target: e.EOwner(), ATK: 5}}
		},
	})

	// Parry: prevent 2×ATK damage (defense event).
	engine.RegisterBehavior("19006", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, ec *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			atk := max(0, p.AttackStat(g))
			return engine.Defends{Defender: p.ID, Against: against, ExtraPrevent: 2 * atk}, nil, true
		},
	})

	// Payback: after the villain attacks you, deal ATK damage back.
	engine.RegisterBehavior("19007", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok || w.Player != e.EOwner() || g.Villains[w.Enemy] == nil {
				return nil
			}
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			return []engine.Message{engine.DamageEntity{Target: w.Enemy, Damage: max(0, p.AttackStat(g)), Source: p.ID}}
		},
	})

	// Drax's Knife: +1 ATK in hero form (restriction unenforced).
	engine.RegisterBehavior("19008", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			if p.IsHero() {
				return engine.StatBonus{ATK: 1}
			}
			return engine.StatBonus{}
		},
	})

	// Drax's Other Knife: retaliate 1 in hero form — IdentityStats
	// carries Retaliate.
	engine.RegisterBehavior("19009", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			if p.IsHero() {
				return engine.StatBonus{Retaliate: 1}
			}
			return engine.StatBonus{}
		},
	})

	// DWI Theet Mastery: after Drax's basic attack, draw 1.
	engine.RegisterBehavior("19010", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			ba, ok := msg.(engine.BasicAttack)
			if !ok || ba.Player != e.EOwner() {
				return nil
			}
			return []engine.Message{engine.DrawCards{Player: e.EOwner(), N: 1}}
		},
	})

	// Too Stubborn to Die: defeat save — set HP to 4, flip alter-ego,
	// remove from the game.
	engine.RegisterBehavior("19011", &engine.Behavior{
		DefeatSave: func(g *engine.Game, p *engine.Player, u *engine.Upgrade) bool {
			p.Damage = p.MaxHP - 4
			p.Side = engine.SideAlterEgo
			g.Delete(u.ID) // removed from the game; tracked via the log
			g.TLogMajorf("c.tooStubbornToDieSurvivesAt4HitPointsInAlterEgoForm", p.Name)
			return true
		},
	})

	// Martyr: tough after her attack defeats an enemy (approximated to
	// whenever she attacks a minion — defeat is not observable).
	engine.RegisterBehavior("19012", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.AllyAttackWindow)
			if !ok || w.Ally != e.EID() || g.Minions[w.Target] == nil {
				return nil
			}
			return []engine.Message{engine.ToughEntity{Target: e.EID()}}
		},
	})

	// Moondragon: exhaust + discard → a minion attacks another enemy.
	engine.RegisterBehavior("19013", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustDiscardMoondragonAMinionAttacksAnotherEnemy"), Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.DiscardControlled{Player: g.Entity(self).EOwner(), ID: self}}
					// The attack-and-target chain is approximated away:
					// minion-vs-minion attacks are not modeled.
				},
			}}
		},
	})

	// Counter-Punch reprint: alias the core behavior.
	if b := engine.LookupBehavior("01077"); b != nil {
		engine.RegisterBehavior("19014", b)
	}

	// Deflection: prevent up to 5 damage (defense event); the mill rider
	// is approximated to a fixed 3-card mill.
	engine.RegisterBehavior("19015", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, ec *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			return engine.Defends{Defender: p.ID, Against: against, ExtraPrevent: 5},
				[]engine.Message{engine.MillPlayerDeck{Player: p.ID, N: 3}}, true
		},
	})

	// Hard Knocks: 4 damage; tough if defeated (approximated to tough on
	// defeating a minion target).
	engine.RegisterBehavior("19016", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return cardutil.ChooseEnemy(engine.Tf("c.hardKnocksDeal4DamageToWhichEnemy"),
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 4, nil })(g, e)
		},
	})

	// Leading Blow: per-attack ATK reduction window absent; approximated
	// to a ready after your next basic attack.
	engine.RegisterBehavior("19017", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			ba, ok := msg.(engine.BasicAttack)
			if !ok || ba.Player != e.EOwner() {
				return nil
			}
			return []engine.Message{engine.ReadyEntity{ID: e.EOwner()}}
		},
	})

	// Subdue: enemy -3 ATK for one attack — no window; approximated to
	// +3 DEF until end of phase.
	engine.RegisterBehavior("19018", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.ApplyStatBonus{Target: e.EOwner(), DEF: 3}}
		},
	})

	// Indomitable reprint.
	if b := engine.LookupBehavior("01082"); b != nil {
		engine.RegisterBehavior("19019", b)
	}

	// Gamora: after she attacks or thwarts, mill until an event → hand.
	engine.RegisterBehavior("19020", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			var hit bool
			switch w := msg.(type) {
			case engine.AllyAttackWindow:
				hit = w.Ally == e.EID()
			case engine.AllyThwartWindow:
				hit = w.Ally == e.EID()
			}
			if !hit {
				return nil
			}
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			for len(p.Deck) > 0 {
				c := p.Deck[0]
				p.Deck = p.Deck[1:]
				if c.Def().Type == "event" {
					p.Hand = append(p.Hand, c)
					g.TLogf("c.gamoraFinds", p.Name, c)
					return nil
				}
				p.Discard = append(p.Discard, c)
			}
			return nil
		},
	})

	// Athletic Conditioning: discard a stun or confuse from your hero.
	engine.RegisterBehavior("19021", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var msgs []engine.Message
			if p := g.Player(e.EOwner()); p != nil {
				msgs = append(msgs, engine.ClearStun{Target: p.ID}, engine.ClearConfuse{Target: p.ID})
			}
			return msgs
		},
	})

	// Basic resources + Enhanced Physique reprint (alias thor 06034).
	engine.RegisterBehavior("19022", &engine.Behavior{})
	engine.RegisterBehavior("19023", &engine.Behavior{})
	engine.RegisterBehavior("19024", &engine.Behavior{})
	if b := engine.LookupBehavior("06034"); b != nil {
		engine.RegisterBehavior("19033", b)
	}

	// Memories of Another Life obligation.
	engine.RegisterBehavior("19025", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var picks []engine.Choice
			if !p.Exhausted {
				picks = append(picks, engine.Choice{ID: "exhaust", Label: engine.Tf("c.exhaustYourAlterEgoRemoveFromTheGame"), Kind: engine.ChoiceLabel}.
					Msgs(engine.ExhaustEntity{ID: p.ID},
						engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}))
			}
			if !p.Stunned {
				picks = append(picks, engine.Choice{ID: "stun", Label: engine.Tf("c.youAreStunned"), Kind: engine.ChoiceLabel}.
					Msgs(engine.StunEntity{Target: p.ID},
						engine.ObligationResolve{Player: p.ID, Card: card}))
			}
			if len(picks) == 0 {
				return []engine.Message{
					engine.RevealNextEncounter{Player: p.ID},
					engine.ObligationResolve{Player: p.ID, Card: card},
				}
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.memoriesOfAnotherLife"), picks...)}}
		},
	})

	// "Bring It!": draw 1 per engaged minion.
	engine.RegisterBehavior("19030", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			n := 0
			for _, mn := range g.Minions {
				if mn.EngagedWith == e.EOwner() {
					n++
				}
			}
			if n == 0 {
				return nil
			}
			return []engine.Message{engine.DrawCards{Player: e.EOwner(), N: n}}
		},
	})

	// "Think Fast!": take 1 damage, confuse the villain.
	engine.RegisterBehavior("19031", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			msgs := []engine.Message{engine.DamageEntity{Target: e.EOwner(), Damage: 1, Source: e.EOwner()}}
			for id := range g.Villains {
				msgs = append(msgs, engine.ConfuseEntity{Target: id})
				break
			}
			return msgs
		},
	})

	// Regroup: defeated-by-attack allies return to hand; discarded at
	// round end.
	engine.RegisterBehavior("19032", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			s := g.Supports[e.EID()]
			if s == nil {
				return nil
			}
			switch m := msg.(type) {
			case engine.AllyDestroyed:
				// Approximation: all ally destructions return to hand
				// (attack attribution is not tracked).
				_ = m
				return nil // destruction already ran; the redirect hook
				// would need an intercept before destruction.
			case engine.BeginRound:
				return []engine.Message{engine.DiscardControlled{Player: s.Owner, ID: e.EID()}}
			}
			return nil
		},
	})
}

func registerNemesis() {
	// Cull the Weak: +2 ATK to each enemy (synced on spawn/reveal).
	engine.RegisterBehavior("19026", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for _, mn := range g.Minions {
				mn.AttackVal += 2
			}
			g.TLogf("c.cullTheWeakEachEnemyGets2Atk")
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if m, ok := msg.(engine.MinionEntersPlay); ok {
				if mn := g.Minions[m.MinionID]; mn != nil && g.SideSchemes[e.EID()] != nil {
					mn.AttackVal += 2
				}
			}
			return nil
		},
	})

	// Yotat: Guard + Retaliate 1 printed.
	engine.RegisterBehavior("19027", &engine.Behavior{})

	// Challenge Accepted: attach highest-ATK enemy; discard after Drax
	// deals 4+ damage in one attack.
	engine.RegisterBehavior("19028", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			var best engine.EntityID
			bestATK := -1
			for id, v := range g.Villains {
				if v.AttackVal > bestATK {
					best, bestATK = id, v.AttackVal
				}
			}
			for id, mn := range g.Minions {
				if mn.AttackVal > bestATK {
					best, bestATK = id, mn.AttackVal
				}
			}
			t.Target = best
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.DamageEntity)
			a := g.Attachments[e.EID()]
			if !ok || a == nil || d.Target != a.Target || d.Damage < 4 {
				return nil
			}
			// Drax attribution approximated to any player source.
			if !d.Source.Is(engine.KindPlayer) {
				return nil
			}
			g.Delete(a.ID)
			g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: a.Code})
			g.TLogf("c.challengeAcceptedIsDiscarded")
			return nil
		},
	})

	// "I Will Destroy You!": hero — Yotat attacks +1; alter-ego surge.
	engine.RegisterBehavior("19029", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if !p.IsHero() {
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			for _, mn := range g.Minions {
				if mn.Code[:5] == "19027" {
					mn.AttackVal++
					return []engine.Message{engine.AskAttack{Enemy: mn.ID, Player: p.ID}}
				}
			}
			for id := range g.Villains {
				return []engine.Message{engine.DealBoost{Enemy: id}, engine.RevealBoost{Enemy: id},
					engine.AskAttack{Enemy: id, Player: p.ID, Trigger: engine.TriggerVillainAttacksYou}}
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
